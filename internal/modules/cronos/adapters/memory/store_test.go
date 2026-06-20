package memory

import (
	"context"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/domain"
)

func TestStoreCopiesWorkdaysOnSaveAndList(t *testing.T) {
	ctx := context.Background()
	store := NewStore()
	day := time.Date(2026, time.June, 19, 0, 0, 0, 0, time.UTC)
	profile := domain.ScheduleProfile{
		ID:           "H-FLEX-ADM",
		Name:         "Flexible administrativo",
		DailyTarget:  domain.Minutes(7, 30),
		WeeklyTarget: domain.Minutes(37, 30),
	}
	workday := domain.Workday{
		EmployeeID: "EMP-0042",
		Date:       day,
		Age:        39,
		Profile:    profile,
		Punches: []domain.Punch{
			{At: at(day, 8, 0), Kind: domain.PunchEntry},
			{At: at(day, 15, 30), Kind: domain.PunchExit},
		},
	}
	if err := store.SaveWorkday(ctx, workday); err != nil {
		t.Fatalf("SaveWorkday() error = %v", err)
	}
	workday.Punches[0].Kind = domain.PunchExit
	listed, err := store.ListWorkdays(ctx, day)
	if err != nil {
		t.Fatalf("ListWorkdays() error = %v", err)
	}
	if got := listed[0].Punches[0].Kind; got != domain.PunchEntry {
		t.Fatalf("stored punch kind = %s, want %s", got, domain.PunchEntry)
	}
	listed[0].Punches[0].Kind = domain.PunchExit
	again, err := store.ListWorkdays(ctx, day)
	if err != nil {
		t.Fatalf("ListWorkdays() second error = %v", err)
	}
	if got := again[0].Punches[0].Kind; got != domain.PunchEntry {
		t.Fatalf("stored punch kind after external mutation = %s, want %s", got, domain.PunchEntry)
	}
}

func at(day time.Time, hour, minute int) time.Time {
	return time.Date(day.Year(), day.Month(), day.Day(), hour, minute, 0, 0, day.Location())
}
