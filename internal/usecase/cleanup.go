package usecase

import (
	"context"
	"fmt"
	"time"

	"cleanup/internal/domain"
)

// Store is owned by the use case; filesystem details stay in the adapter.
type Store interface {
	Walk(context.Context, func(domain.File) error) error
	Delete(domain.File) (bool, error)
}

type Result struct {
	Matched int
	Deleted int
	Changed int
	Bytes   int64
}

type Event struct {
	File   domain.File
	Action string
}

type Cleanup struct {
	Store Store
}

// Run is a one-shot batch. On error it returns the completed portion of the batch.
func (c Cleanup) Run(ctx context.Context, cutoff time.Time, deleteFiles bool, report func(Event) error) (Result, error) {
	var result Result
	err := c.Store.Walk(ctx, func(file domain.File) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !file.OlderThan(cutoff) {
			return nil
		}
		result.Matched++
		result.Bytes += file.Size
		action := "would-delete"
		if deleteFiles {
			deleted, err := c.Store.Delete(file)
			if err != nil {
				return fmt.Errorf("delete %q: %w", file.Path, err)
			}
			if deleted {
				result.Deleted++
				action = "deleted"
			} else {
				result.Changed++
				action = "skipped-changed"
			}
		}
		if report != nil {
			return report(Event{File: file, Action: action})
		}
		return nil
	})
	return result, err
}
