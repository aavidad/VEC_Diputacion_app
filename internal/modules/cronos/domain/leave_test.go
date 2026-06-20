package domain

import (
	"errors"
	"testing"
	"time"
)

func TestLeaveRequestChecksBalance(t *testing.T) {
	policy := LeavePolicy{ID: "asuntos_propios", Name: "ASUNTOS PROPIOS", Unit: LeaveUnitDay, AnnualAllowance: 6, Requestable: true, MinRequest: 1, MaxRequest: 6}
	balance := LeaveBalance{EmployeeID: "EMP-0031", Year: 2026, PolicyID: "asuntos_propios", Granted: 6, Requested: 1, Approved: 2, Consumed: 1}
	request := LeaveRequest{
		EmployeeID: "EMP-0031",
		PolicyID:   "asuntos_propios",
		From:       time.Date(2026, time.June, 20, 0, 0, 0, 0, time.UTC),
		To:         time.Date(2026, time.June, 22, 0, 0, 0, 0, time.UTC),
		Amount:     3,
		Unit:       LeaveUnitDay,
	}
	if err := request.Validate(policy, balance); !errors.Is(err, ErrLeaveBalanceExceeded) {
		t.Fatalf("Validate() error = %v, want ErrLeaveBalanceExceeded", err)
	}
}

func TestLeaveRequestRequiresDocumentWhenPolicyRequiresIt(t *testing.T) {
	policy := LeavePolicy{ID: "medico", Name: "HORAS DE MEDICO", Unit: LeaveUnitHour, AnnualAllowance: 180, Requestable: true, RequiresDocument: true, MinRequest: 15, MaxRequest: 180}
	balance := LeaveBalance{EmployeeID: "EMP-0031", Year: 2026, PolicyID: "medico", Granted: 180}
	request := LeaveRequest{
		EmployeeID: "EMP-0031",
		PolicyID:   "medico",
		From:       time.Date(2026, time.June, 20, 10, 0, 0, 0, time.UTC),
		To:         time.Date(2026, time.June, 20, 11, 0, 0, 0, time.UTC),
		Amount:     60,
		Unit:       LeaveUnitHour,
	}
	if err := request.Validate(policy, balance); !errors.Is(err, ErrLeaveRequestInvalid) {
		t.Fatalf("Validate() error = %v, want ErrLeaveRequestInvalid", err)
	}
	request.DocumentRef = "DOC-MED-1"
	if err := request.Validate(policy, balance); err != nil {
		t.Fatalf("Validate() with document error = %v", err)
	}
}
