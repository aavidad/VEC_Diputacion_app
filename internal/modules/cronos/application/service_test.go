package application

import (
	"context"
	"errors"
	"testing"
	"time"

	cronosmemory "vec-diputacion-granada/internal/modules/cronos/adapters/memory"
	"vec-diputacion-granada/internal/modules/cronos/domain"
)

func TestServicePersistsAndEvaluatesWorkdays(t *testing.T) {
	ctx := context.Background()
	service, err := NewService(cronosmemory.NewStore())
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	profile := domain.ScheduleProfile{
		ID:               "H-FLEX-ADM",
		Name:             "Flexible administrativo",
		Unit:             "Administracion General",
		Flexible:         true,
		AllowsTelework:   true,
		DailyTarget:      domain.Minutes(7, 30),
		WeeklyTarget:     domain.Minutes(37, 30),
		EntryWindowStart: domain.Minutes(7, 30),
		EntryWindowEnd:   domain.Minutes(9, 30),
		CoreStart:        domain.Minutes(9, 30),
		CoreEnd:          domain.Minutes(14, 0),
	}
	if err := service.SaveProfile(ctx, profile); err != nil {
		t.Fatalf("SaveProfile() error = %v", err)
	}
	day := time.Date(2026, time.June, 19, 0, 0, 0, 0, time.UTC)
	if err := service.RegisterWorkday(ctx, domain.Workday{
		EmployeeID: "EMP-0064",
		Date:       day,
		Age:        64,
		Profile:    profile,
		Punches: []domain.Punch{
			{At: at(day, 8, 0), Kind: domain.PunchEntry},
			{At: at(day, 13, 30), Kind: domain.PunchExit},
		},
	}); err != nil {
		t.Fatalf("RegisterWorkday() error = %v", err)
	}
	snapshot, err := service.Snapshot(ctx, day, []string{"EMP-0064"})
	if err != nil {
		t.Fatalf("Snapshot() error = %v", err)
	}
	if len(snapshot.Profiles) != 1 || len(snapshot.Workdays) != 1 || len(snapshot.Results) != 1 {
		t.Fatalf("snapshot sizes = profiles %d workdays %d results %d, want 1/1/1", len(snapshot.Profiles), len(snapshot.Workdays), len(snapshot.Results))
	}
	result := snapshot.Results[0]
	if result.Reduction != 2*time.Hour || result.Theoretical != domain.Minutes(5, 30) || result.Balance != 0 {
		t.Fatalf("result = reduction %s theoretical %s balance %s", result.Reduction, result.Theoretical, result.Balance)
	}
}

func TestServiceCronosExigeAlcancesConcretos(t *testing.T) {
	service, err := NewService(cronosmemory.NewStore())
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	ctx := context.Background()
	if _, err := service.Snapshot(ctx, time.Time{}, []string{"EMP-0031"}); !errors.Is(err, domain.ErrWorkdayInvalid) {
		t.Fatalf("Snapshot(fecha cero) error = %v", err)
	}
	for _, employeeIDs := range [][]string{nil, {}, {"*"}, {"EMP-0031", "EMP-0031"}} {
		if _, err := service.Snapshot(ctx, time.Now().UTC(), employeeIDs); !errors.Is(err, domain.ErrWorkdayInvalid) {
			t.Fatalf("Snapshot(%#v) error = %v", employeeIDs, err)
		}
	}
	if _, err := service.LeaveBalances(ctx, "", 2026); !errors.Is(err, domain.ErrLeaveBalanceInvalid) {
		t.Fatalf("LeaveBalances(empleado vacio) error = %v", err)
	}
	if _, err := service.LeaveRequests(ctx, "EMP-0031", 0); !errors.Is(err, domain.ErrLeaveRequestInvalid) {
		t.Fatalf("LeaveRequests(anio cero) error = %v", err)
	}
}

func TestServiceRequestsLeaveAndUpdatesBalance(t *testing.T) {
	ctx := context.Background()
	service, err := NewService(cronosmemory.NewStore())
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	policy := domain.LeavePolicy{ID: "asuntos_propios", Name: "ASUNTOS PROPIOS", Unit: domain.LeaveUnitDay, AnnualAllowance: 6, Requestable: true, RequiresApproval: true, MinRequest: 1, MaxRequest: 6}
	if err := service.SaveLeavePolicy(ctx, policy); err != nil {
		t.Fatalf("SaveLeavePolicy() error = %v", err)
	}
	if err := service.SaveLeaveBalance(ctx, domain.LeaveBalance{EmployeeID: "EMP-0031", Year: 2026, PolicyID: policy.ID, Granted: 6, Consumed: 2}); err != nil {
		t.Fatalf("SaveLeaveBalance() error = %v", err)
	}
	request, err := service.RequestLeave(ctx, domain.LeaveRequest{
		EmployeeID: "EMP-0031",
		PolicyID:   policy.ID,
		From:       time.Date(2026, time.June, 20, 0, 0, 0, 0, time.UTC),
		To:         time.Date(2026, time.June, 20, 0, 0, 0, 0, time.UTC),
		Amount:     1,
		Unit:       domain.LeaveUnitDay,
		Reason:     "Asunto propio",
	})
	if err != nil {
		t.Fatalf("RequestLeave() error = %v", err)
	}
	if request.ID == "" || request.State != domain.LeaveStateReview {
		t.Fatalf("request = %#v, want generated id and review state", request)
	}
	balances, err := service.LeaveBalances(ctx, "EMP-0031", 2026)
	if err != nil {
		t.Fatalf("LeaveBalances() error = %v", err)
	}
	if len(balances) != 1 || balances[0].Requested != 1 || balances[0].Remaining() != 3 {
		t.Fatalf("balances = %#v, want requested=1 remaining=3", balances)
	}
	requests, err := service.LeaveRequests(ctx, "EMP-0031", 2026)
	if err != nil {
		t.Fatalf("LeaveRequests() error = %v", err)
	}
	if len(requests) != 1 || requests[0].PolicyID != policy.ID {
		t.Fatalf("requests = %#v, want one asuntos_propios request", requests)
	}
}

func at(day time.Time, hour, minute int) time.Time {
	return time.Date(day.Year(), day.Month(), day.Day(), hour, minute, 0, 0, day.Location())
}
