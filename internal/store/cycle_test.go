package store

import (
	"testing"
	"time"
)

func TestComputeCycleStart(t *testing.T) {
	tests := []struct {
		name      string
		now       time.Time
		day       int
		wantYear  int
		wantMonth time.Month
		wantDay   int
	}{
		{
			name:      "day 1, today 2026-05-12 -> 2026-05-01",
			now:       time.Date(2026, time.May, 12, 10, 0, 0, 0, time.UTC),
			day:       1,
			wantYear:  2026,
			wantMonth: time.May,
			wantDay:   1,
		},
		{
			name:      "day 15, today 2026-05-12 -> previous month 2026-04-15",
			now:       time.Date(2026, time.May, 12, 10, 0, 0, 0, time.UTC),
			day:       15,
			wantYear:  2026,
			wantMonth: time.April,
			wantDay:   15,
		},
		{
			name:      "day 15, today 2026-05-20 -> 2026-05-15",
			now:       time.Date(2026, time.May, 20, 10, 0, 0, 0, time.UTC),
			day:       15,
			wantYear:  2026,
			wantMonth: time.May,
			wantDay:   15,
		},
		{
			// March 1: day 31 hasn't happened in March yet so we step back to
			// February; February (2026, non-leap) has 28 days so the renewal
			// day clamps to 28.
			name:      "day 31 stepping back into February non-leap clamps to Feb 28 (2026)",
			now:       time.Date(2026, time.March, 1, 10, 0, 0, 0, time.UTC),
			day:       31,
			wantYear:  2026,
			wantMonth: time.February,
			wantDay:   28,
		},
		{
			// May 1: day 31 hasn't happened in May yet so we step back to
			// April; April has 30 days so the renewal day clamps to 30.
			name:      "day 31 stepping back into April clamps to April 30",
			now:       time.Date(2026, time.May, 1, 10, 0, 0, 0, time.UTC),
			day:       31,
			wantYear:  2026,
			wantMonth: time.April,
			wantDay:   30,
		},
		{
			name:      "day 0 clamps up to 1",
			now:       time.Date(2026, time.May, 12, 10, 0, 0, 0, time.UTC),
			day:       0,
			wantYear:  2026,
			wantMonth: time.May,
			wantDay:   1,
		},
		{
			name:      "day 99 clamps down to 31 (May has 31 days)",
			now:       time.Date(2026, time.May, 31, 10, 0, 0, 0, time.UTC),
			day:       99,
			wantYear:  2026,
			wantMonth: time.May,
			wantDay:   31,
		},
		{
			name:      "month wrap: today 2026-01-05, day 15 -> 2025-12-15",
			now:       time.Date(2026, time.January, 5, 10, 0, 0, 0, time.UTC),
			day:       15,
			wantYear:  2025,
			wantMonth: time.December,
			wantDay:   15,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeCycleStart(tt.now, tt.day)
			if got.Year() != tt.wantYear || got.Month() != tt.wantMonth || got.Day() != tt.wantDay {
				t.Errorf("computeCycleStart(%s, %d) = %d-%02d-%02d, want %d-%02d-%02d",
					tt.now.Format("2006-01-02"), tt.day,
					got.Year(), got.Month(), got.Day(),
					tt.wantYear, tt.wantMonth, tt.wantDay)
			}
		})
	}
}
