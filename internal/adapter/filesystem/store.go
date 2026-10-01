package filesystem

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"cleanup/internal/domain"
)

type Store struct {
	root *os.Root
}

func Open(directory string) (*Store, error) {
	if directory == "" {
		return nil, errors.New("directory is required")
	}
	absolute, err := filepath.Abs(directory)
	if err != nil {
		return nil, err
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return nil, fmt.Errorf("resolve directory: %w", err)
	}
	if filepath.Dir(resolved) == resolved {
		return nil, errors.New("filesystem root cannot be cleaned")
	}
	root, err := os.OpenRoot(resolved)
	if err != nil {
		return nil, fmt.Errorf("open directory: %w", err)
	}
	return &Store{root: root}, nil
}

func (s *Store) Close() error {
	return s.root.Close()
}

func (s *Store) Walk(ctx context.Context, visit func(domain.File) error) error {
	return fs.WalkDir(s.root.FS(), ".", func(name string, entry fs.DirEntry, walkErr error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if walkErr != nil {
			return fmt.Errorf("walk %q: %w", name, walkErr)
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("stat %q: %w", name, err)
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		return visit(domain.File{Path: name, Size: info.Size(), ModifiedAt: info.ModTime()})
	})
}

// Delete rechecks the file to skip common changes between discovery and deletion.
func (s *Store) Delete(file domain.File) (bool, error) {
	info, err := s.root.Lstat(file.Path)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !info.Mode().IsRegular() || info.Size() != file.Size || !info.ModTime().Equal(file.ModifiedAt) {
		return false, nil
	}
	if err := s.root.Remove(file.Path); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
