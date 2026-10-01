package filesystem_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"cleanup/internal/adapter/filesystem"
	"cleanup/internal/domain"
	"cleanup/internal/usecase"
)

func TestDeleteSkipsChangedFilesAndRejectsEscapes(t *testing.T) {
	directory := t.TempDir()
	filename := filepath.Join(directory, "old.txt")
	if err := os.WriteFile(filename, []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	store, err := filesystem.Open(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	var found domain.File
	if err := store.Walk(context.Background(), func(file domain.File) error { found = file; return nil }); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filename, []byte("updated payload"), 0600); err != nil {
		t.Fatal(err)
	}
	if deleted, err := store.Delete(found); deleted || err != nil {
		t.Fatalf("changed file: deleted=%t err=%v", deleted, err)
	}
	if deleted, err := store.Delete(domain.File{Path: "missing.txt"}); deleted || err != nil {
		t.Fatalf("missing file: deleted=%t err=%v", deleted, err)
	}
	outside := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(outside, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Dir(outside), filepath.Join(directory, "escape")); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"../outside.txt", "escape/outside.txt"} {
		if deleted, err := store.Delete(domain.File{Path: name}); deleted || err == nil {
			t.Fatalf("escape %q: deleted=%t err=%v", name, deleted, err)
		}
	}
	if _, err := os.Stat(outside); err != nil {
		t.Fatalf("outside file was affected: %v", err)
	}
}

func TestCancellationPreservesFiles(t *testing.T) {
	directory := t.TempDir()
	filename := filepath.Join(directory, "old.txt")
	if err := os.WriteFile(filename, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	store, err := filesystem.Open(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result, err := (usecase.Cleanup{Store: store}).Run(ctx, time.Now().Add(time.Hour), true, nil)
	if !errors.Is(err, context.Canceled) || result.Deleted != 0 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if _, err := os.Stat(filename); err != nil {
		t.Fatal(err)
	}
}

func TestOpenRejectsFilesystemRoot(t *testing.T) {
	if store, err := filesystem.Open(string(filepath.Separator)); err == nil {
		store.Close()
		t.Fatal("accepted filesystem root")
	}
}
