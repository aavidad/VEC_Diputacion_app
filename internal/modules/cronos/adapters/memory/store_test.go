package memory

import (
	"context"
	"errors"
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
	otra := workday
	otra.EmployeeID = "EMP-0099"
	if err := store.SaveWorkday(ctx, otra); err != nil {
		t.Fatalf("SaveWorkday(otra persona) error = %v", err)
	}
	workday.Punches[0].Kind = domain.PunchExit
	listed, err := store.ListWorkdays(ctx, []string{"EMP-0042"}, day)
	if err != nil {
		t.Fatalf("ListWorkdays() error = %v", err)
	}
	if len(listed) != 1 || listed[0].EmployeeID != "EMP-0042" {
		t.Fatalf("el alcance positivo devolvio otras personas: %#v", listed)
	}
	if got := listed[0].Punches[0].Kind; got != domain.PunchEntry {
		t.Fatalf("stored punch kind = %s, want %s", got, domain.PunchEntry)
	}
	listed[0].Punches[0].Kind = domain.PunchExit
	again, err := store.ListWorkdays(ctx, []string{"EMP-0042"}, day)
	if err != nil {
		t.Fatalf("ListWorkdays() second error = %v", err)
	}
	if got := again[0].Punches[0].Kind; got != domain.PunchEntry {
		t.Fatalf("stored punch kind after external mutation = %s, want %s", got, domain.PunchEntry)
	}
}

func TestStoreNoInterpretaFiltrosCronosVaciosComoTodos(t *testing.T) {
	store := NewStore()
	ctx := context.Background()
	if resultado, err := store.ListWorkdays(ctx, []string{"EMP-0042"}, time.Time{}); !errors.Is(err, domain.ErrWorkdayInvalid) || resultado != nil {
		t.Fatalf("ListWorkdays(fecha cero) = (%#v, %v)", resultado, err)
	}
	for _, employeeIDs := range [][]string{nil, {}, {""}, {"*"}, {"EMP-0042", "EMP-0042"}, {" EMP-0042"}} {
		if resultado, err := store.ListWorkdays(ctx, employeeIDs, time.Now().UTC()); !errors.Is(err, domain.ErrWorkdayInvalid) || resultado != nil {
			t.Fatalf("ListWorkdays(%#v) = (%#v, %v)", employeeIDs, resultado, err)
		}
	}
	for _, employeeID := range []string{"", " ", " EMP-0042"} {
		if resultado, err := store.ListLeaveBalances(ctx, employeeID, 2026); !errors.Is(err, domain.ErrLeaveBalanceInvalid) || resultado != nil {
			t.Fatalf("ListLeaveBalances(%q) = (%#v, %v)", employeeID, resultado, err)
		}
		if resultado, err := store.ListLeaveRequests(ctx, employeeID, 2026); !errors.Is(err, domain.ErrLeaveRequestInvalid) || resultado != nil {
			t.Fatalf("ListLeaveRequests(%q) = (%#v, %v)", employeeID, resultado, err)
		}
	}
	if resultado, err := store.ListLeaveBalances(ctx, "EMP-0042", 0); !errors.Is(err, domain.ErrLeaveBalanceInvalid) || resultado != nil {
		t.Fatalf("ListLeaveBalances(anio cero) = (%#v, %v)", resultado, err)
	}
	if resultado, err := store.ListLeaveRequests(ctx, "EMP-0042", 0); !errors.Is(err, domain.ErrLeaveRequestInvalid) || resultado != nil {
		t.Fatalf("ListLeaveRequests(anio cero) = (%#v, %v)", resultado, err)
	}
}

func at(day time.Time, hour, minute int) time.Time {
	return time.Date(day.Year(), day.Month(), day.Day(), hour, minute, 0, 0, day.Location())
}
