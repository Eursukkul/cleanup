package domain

import (
	"fmt"
	"time"
)

type File struct {
	Path       string
	Size       int64
	ModifiedAt time.Time
}

// Cutoff subtracts calendar months, clamping to the last day of the target month.
func Cutoff(now time.Time, months int) (time.Time, error) {
	if months < 1 || months > 1200 {
		return time.Time{}, fmt.Errorf("months must be between 1 and 1200")
	}
	target := time.Date(now.Year(), now.Month(), 1, now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), now.Location()).AddDate(0, -months, 0)
	lastDay := time.Date(target.Year(), target.Month()+1, 0, 0, 0, 0, 0, now.Location()).Day()
	return time.Date(target.Year(), target.Month(), min(now.Day(), lastDay), now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), now.Location()), nil
}

func (f File) OlderThan(cutoff time.Time) bool {
	return f.ModifiedAt.Before(cutoff)
}
