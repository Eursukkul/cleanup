package domain

import (
	"testing"
	"time"
)

func TestCutoffCalendarMonths(t *testing.T) {
	for _, test := range []struct {
		name string
		now  string
		want string
	}{
		{"ordinary", "2026-10-01T12:34:56+07:00", "2026-07-01T12:34:56+07:00"},
		{"month end", "2026-05-31T12:34:56+07:00", "2026-02-28T12:34:56+07:00"},
		{"leap year", "2024-05-31T12:34:56+07:00", "2024-02-29T12:34:56+07:00"},
		{"year boundary", "2026-01-31T12:34:56+07:00", "2025-10-31T12:34:56+07:00"},
	} {
		t.Run(test.name, func(t *testing.T) {
			now, err := time.Parse(time.RFC3339, test.now)
			if err != nil {
				t.Fatal(err)
			}
			got, err := Cutoff(now, 3)
			if err != nil || got.Format(time.RFC3339) != test.want {
				t.Fatalf("got %v, %v; want %s", got, err, test.want)
			}
		})
	}
	for _, months := range []int{-1, 0, 1201} {
		if _, err := Cutoff(time.Now(), months); err == nil {
			t.Fatalf("accepted invalid retention %d", months)
		}
	}
}
