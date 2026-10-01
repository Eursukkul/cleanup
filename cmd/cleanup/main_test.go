package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestBinaryLoadsConfigBesideExecutableFromAnotherDirectory(t *testing.T) {
	base := t.TempDir()
	binary := filepath.Join(base, "cleanup")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	build := exec.Command("go", "build", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, output)
	}
	directory := filepath.Join(base, "data")
	if err := os.Mkdir(directory, 0755); err != nil {
		t.Fatal(err)
	}
	oldFile := filepath.Join(directory, "old.txt")
	if err := os.WriteFile(oldFile, []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	old := time.Now().AddDate(-1, 0, 0)
	if err := os.Chtimes(oldFile, old, old); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "config.yaml"), []byte("directories:\n  - data\ndry_run: true\n"), 0600); err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	if err := os.Mkdir(filepath.Join(workingDirectory, "override-data"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workingDirectory, "custom.yaml"), []byte("directories:\n  - override-data\ndry_run: true\n"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name    string
		args    []string
		summary string
	}{
		{name: "default", summary: "matched=1 deleted=0"},
		{name: "explicit relative config", args: []string{"--config", "custom.yaml"}, summary: "matched=0 deleted=0"},
	} {
		t.Run(test.name, func(t *testing.T) {
			command := exec.Command(binary, test.args...)
			command.Dir = workingDirectory
			output, err := command.CombinedOutput()
			if err != nil {
				t.Fatalf("run from another directory: %v\n%s", err, output)
			}
			if !strings.Contains(string(output), test.summary) {
				t.Fatalf("unexpected output: %s", output)
			}
		})
	}
	if _, err := os.Stat(oldFile); err != nil {
		t.Fatalf("dry-run removed file: %v", err)
	}
}

func TestRunReloadsYAMLAndCleansOnlyOldRegularFiles(t *testing.T) {
	now := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	cutoff := now.AddDate(0, -3, 0)
	base := t.TempDir()
	directory := filepath.Join(base, "data", "nested")
	if err := os.MkdirAll(directory, 0755); err != nil {
		t.Fatal(err)
	}
	files := map[string]time.Time{
		"old.txt":      cutoff.Add(-time.Second),
		"boundary.txt": cutoff,
		"recent.txt":   now,
	}
	for name, modified := range files {
		filename := filepath.Join(directory, name)
		if err := os.WriteFile(filename, []byte("payload"), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(filename, modified, modified); err != nil {
			t.Fatal(err)
		}
	}
	outside := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(outside, []byte("outside"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(outside, cutoff.Add(-time.Hour), cutoff.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(directory, "link")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Dir(outside), filepath.Join(directory, "linked-directory")); err != nil {
		t.Fatal(err)
	}
	documents := filepath.Join(base, "documents")
	if err := os.Mkdir(documents, 0755); err != nil {
		t.Fatal(err)
	}
	document := filepath.Join(documents, "old.txt")
	if err := os.WriteFile(document, []byte("document"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(document, cutoff.Add(-time.Second), cutoff.Add(-time.Second)); err != nil {
		t.Fatal(err)
	}
	configFile := filepath.Join(base, "settings.yaml")
	for _, dryRun := range []bool{true, false} {
		content := fmt.Sprintf("directories:\n  - data\n  - documents\nretention_months: 3\ndry_run: %t\n", dryRun)
		if err := os.WriteFile(configFile, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
		var output, stderr bytes.Buffer
		if err := run(context.Background(), []string{"--config", configFile}, &output, &stderr, now); err != nil {
			t.Fatal(err)
		}
		deleted := 1
		if dryRun {
			deleted = 0
		}
		if strings.Count(output.String(), fmt.Sprintf("matched=1 deleted=%d changed=0", deleted)) != 2 {
			t.Fatalf("unexpected summary: %s", &output)
		}
		for _, oldFile := range []string{filepath.Join(directory, "old.txt"), document} {
			_, err := os.Stat(oldFile)
			if dryRun && err != nil {
				t.Fatalf("dry-run removed %s: %v", oldFile, err)
			}
			if !dryRun && !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("old file %s still exists: %v", oldFile, err)
			}
		}
		for _, kept := range []string{filepath.Join(directory, "boundary.txt"), filepath.Join(directory, "recent.txt"), outside, link, directory} {
			if _, err := os.Lstat(kept); err != nil {
				t.Fatalf("expected %s preserved: %v", kept, err)
			}
		}
	}
	var output bytes.Buffer
	if err := run(context.Background(), []string{"--config", configFile}, &output, &output, now); err != nil {
		t.Fatal(err)
	}
	if strings.Count(output.String(), "matched=0 deleted=0") != 2 {
		t.Fatalf("second deletion should be idempotent: %s", &output)
	}
}
