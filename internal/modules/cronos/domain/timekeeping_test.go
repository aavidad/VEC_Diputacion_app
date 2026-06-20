package domain

import (
	"testing"
	"time"
)

func TestEvaluateWorkdayAppliesPreRetirementReductions(t *testing.T) {
	profile := flexibleAdminProfile()
	day := workDate()

	tests := []struct {
		name      string
		age       int
		exitHour  int
		exitMin   int
		wantGoal  time.Duration
		wantDelta time.Duration
	}{
		{name: "age 63 receives one hour", age: 63, exitHour: 14, exitMin: 30, wantGoal: Minutes(6, 30), wantDelta: 0},
		{name: "age 64 receives two hours", age: 64, exitHour: 13, exitMin: 30, wantGoal: Minutes(5, 30), wantDelta: 0},
		{name: "age under 63 has full target", age: 62, exitHour: 15, exitMin: 30, wantGoal: Minutes(7, 30), wantDelta: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := EvaluateWorkday(Workday{
				EmployeeID: "EMP-0063",
				Date:       day,
				Age:        tt.age,
				Profile:    profile,
				Punches: []Punch{
					{At: at(day, 8, 0), Kind: PunchEntry, Channel: "web"},
					{At: at(day, tt.exitHour, tt.exitMin), Kind: PunchExit, Channel: "web"},
				},
			})
			if err != nil {
				t.Fatalf("EvaluateWorkday() error = %v", err)
			}
			if result.Theoretical != tt.wantGoal {
				t.Fatalf("theoretical = %s, want %s", result.Theoretical, tt.wantGoal)
			}
			if result.Balance != tt.wantDelta {
				t.Fatalf("balance = %s, want %s", result.Balance, tt.wantDelta)
			}
			if len(result.Incidents) != 0 {
				t.Fatalf("incidents = %#v, want none", result.Incidents)
			}
		})
	}
}

func TestEvaluateWorkdayDetectsFixedCoverageBreach(t *testing.T) {
	day := workDate()
	result, err := EvaluateWorkday(Workday{
		EmployeeID: "EMP-0100",
		Date:       day,
		Age:        45,
		Profile: ScheduleProfile{
			ID:               "H-FIJO-MAYORES",
			Name:             "Atencion personas mayores",
			Unit:             "Centro servicios sociales",
			Flexible:         false,
			AllowsTelework:   false,
			RequiresCoverage: true,
			DailyTarget:      Minutes(7, 30),
			WeeklyTarget:     Minutes(37, 30),
			EntryWindowStart: Minutes(8, 0),
			EntryWindowEnd:   Minutes(15, 30),
		},
		Punches: []Punch{
			{At: at(day, 9, 0), Kind: PunchEntry, Channel: "terminal"},
			{At: at(day, 15, 30), Kind: PunchExit, Channel: "terminal"},
		},
	})
	if err != nil {
		t.Fatalf("EvaluateWorkday() error = %v", err)
	}
	if !result.FixedCoverageBreach {
		t.Fatalf("FixedCoverageBreach = false, want true")
	}
	if !hasIncident(result, "cobertura_obligatoria_incumplida") {
		t.Fatalf("incidents = %#v, want coverage breach", result.Incidents)
	}
}

func TestEvaluateWorkdayKeepsOpenEntryAsBlockingIncident(t *testing.T) {
	day := workDate()
	result, err := EvaluateWorkday(Workday{
		EmployeeID: "EMP-0042",
		Date:       day,
		Age:        39,
		Profile:    flexibleAdminProfile(),
		Punches: []Punch{
			{At: at(day, 7, 55), Kind: PunchEntry, Channel: "web"},
		},
	})
	if err != nil {
		t.Fatalf("EvaluateWorkday() error = %v", err)
	}
	if !result.OpenEntry {
		t.Fatalf("OpenEntry = false, want true")
	}
	if !hasIncident(result, "fichaje_sin_salida") {
		t.Fatalf("incidents = %#v, want fichaje_sin_salida", result.Incidents)
	}
	if got := FormatHHMM(result.Balance); got != "-07:30" {
		t.Fatalf("balance = %s, want -07:30", got)
	}
}

func flexibleAdminProfile() ScheduleProfile {
	return ScheduleProfile{
		ID:               "H-FLEX-ADM",
		Name:             "Flexible administrativo",
		Unit:             "Administracion General",
		Flexible:         true,
		AllowsTelework:   true,
		RequiresCoverage: false,
		DailyTarget:      Minutes(7, 30),
		WeeklyTarget:     Minutes(37, 30),
		EntryWindowStart: Minutes(7, 30),
		EntryWindowEnd:   Minutes(9, 30),
		CoreStart:        Minutes(9, 30),
		CoreEnd:          Minutes(14, 0),
	}
}

func workDate() time.Time {
	return time.Date(2026, time.June, 19, 0, 0, 0, 0, time.UTC)
}

func at(day time.Time, hour, minute int) time.Time {
	return time.Date(day.Year(), day.Month(), day.Day(), hour, minute, 0, 0, day.Location())
}

func hasIncident(result DayResult, code string) bool {
	for _, incident := range result.Incidents {
		if incident.Code == code {
			return true
		}
	}
	return false
}
