package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrLeavePolicyInvalid   = errors.New("cronos leave policy invalid")
	ErrLeaveBalanceInvalid  = errors.New("cronos leave balance invalid")
	ErrLeaveRequestInvalid  = errors.New("cronos leave request invalid")
	ErrLeaveBalanceExceeded = errors.New("cronos leave balance exceeded")
)

type LeaveUnit string

const (
	LeaveUnitDay  LeaveUnit = "dia"
	LeaveUnitHour LeaveUnit = "hora"
)

type LeaveState string

const (
	LeaveStateDraft     LeaveState = "borrador"
	LeaveStateRequested LeaveState = "solicitado"
	LeaveStateReview    LeaveState = "pendiente_responsable"
	LeaveStateApproved  LeaveState = "aprobado"
	LeaveStateDenied    LeaveState = "denegado"
	LeaveStateCancelled LeaveState = "cancelado"
	LeaveStateConsumed  LeaveState = "disfrutado"
)

type LeavePolicy struct {
	ID               string
	Name             string
	Unit             LeaveUnit
	AnnualAllowance  int
	Requestable      bool
	RequiresDocument bool
	RequiresApproval bool
	MinRequest       int
	MaxRequest       int
	PayrollImpact    bool
}

func (p LeavePolicy) Validate() error {
	if strings.TrimSpace(p.ID) == "" || strings.TrimSpace(p.Name) == "" {
		return ErrLeavePolicyInvalid
	}
	if p.Unit != LeaveUnitDay && p.Unit != LeaveUnitHour {
		return ErrLeavePolicyInvalid
	}
	if p.AnnualAllowance < 0 || p.MinRequest < 0 || p.MaxRequest < 0 {
		return ErrLeavePolicyInvalid
	}
	if p.MaxRequest > 0 && p.MinRequest > p.MaxRequest {
		return ErrLeavePolicyInvalid
	}
	return nil
}

type LeaveBalance struct {
	EmployeeID string
	Year       int
	PolicyID   string
	Granted    int
	Requested  int
	Approved   int
	Consumed   int
}

func (b LeaveBalance) Validate() error {
	if strings.TrimSpace(b.EmployeeID) == "" || strings.TrimSpace(b.PolicyID) == "" || b.Year <= 0 {
		return ErrLeaveBalanceInvalid
	}
	if b.Granted < 0 || b.Requested < 0 || b.Approved < 0 || b.Consumed < 0 {
		return ErrLeaveBalanceInvalid
	}
	return nil
}

func (b LeaveBalance) Remaining() int {
	remaining := b.Granted - b.Requested - b.Approved - b.Consumed
	if remaining < 0 {
		return 0
	}
	return remaining
}

type LeaveRequest struct {
	ID          string
	EmployeeID  string
	PolicyID    string
	From        time.Time
	To          time.Time
	Amount      int
	Unit        LeaveUnit
	State       LeaveState
	Reason      string
	DocumentRef string
	CreatedAt   time.Time
}

func (r LeaveRequest) Validate(policy LeavePolicy, balance LeaveBalance) error {
	if strings.TrimSpace(r.EmployeeID) == "" || strings.TrimSpace(r.PolicyID) == "" {
		return ErrLeaveRequestInvalid
	}
	if r.PolicyID != policy.ID || r.PolicyID != balance.PolicyID || r.EmployeeID != balance.EmployeeID {
		return ErrLeaveRequestInvalid
	}
	if !policy.Requestable {
		return ErrLeaveRequestInvalid
	}
	if r.From.IsZero() || r.To.IsZero() || r.To.Before(r.From) || r.Amount <= 0 {
		return ErrLeaveRequestInvalid
	}
	if r.Unit != policy.Unit {
		return ErrLeaveRequestInvalid
	}
	if policy.MinRequest > 0 && r.Amount < policy.MinRequest {
		return ErrLeaveRequestInvalid
	}
	if policy.MaxRequest > 0 && r.Amount > policy.MaxRequest {
		return ErrLeaveRequestInvalid
	}
	if policy.RequiresDocument && strings.TrimSpace(r.DocumentRef) == "" {
		return ErrLeaveRequestInvalid
	}
	if r.Amount > balance.Remaining() {
		return ErrLeaveBalanceExceeded
	}
	return nil
}

func DefaultLeavePolicies() []LeavePolicy {
	return []LeavePolicy{
		{ID: "asuntos_propios", Name: "ASUNTOS PROPIOS", Unit: LeaveUnitDay, AnnualAllowance: 6, Requestable: true, RequiresApproval: true, MinRequest: 1, MaxRequest: 6},
		{ID: "vacaciones", Name: "VACACIONES", Unit: LeaveUnitDay, AnnualAllowance: 22, Requestable: true, RequiresApproval: true, MinRequest: 1, MaxRequest: 22},
		{ID: "bolsa_conciliacion", Name: "BOLSA HORARIA POR CONCILIACION", Unit: LeaveUnitHour, AnnualAllowance: 30 * 60, Requestable: true, RequiresApproval: true, MinRequest: 15, MaxRequest: 8 * 60},
		{ID: "compensacion_horaria", Name: "COMPENSACION HORARIA - PERMISO", Unit: LeaveUnitHour, AnnualAllowance: 19, Requestable: true, RequiresApproval: true, MinRequest: 1, MaxRequest: 19},
		{ID: "horas_sindicales", Name: "HORAS SINDICALES", Unit: LeaveUnitHour, AnnualAllowance: 60 * 60, Requestable: true, RequiresDocument: true, RequiresApproval: true, MinRequest: 30, MaxRequest: 8 * 60},
		{ID: "medico", Name: "HORAS DE MEDICO", Unit: LeaveUnitHour, AnnualAllowance: 3 * 60, Requestable: true, RequiresDocument: true, RequiresApproval: true, MinRequest: 15, MaxRequest: 3 * 60},
		{ID: "enfermedad_sin_baja", Name: "ENFERMEDAD SIN BAJA (PERMISO)", Unit: LeaveUnitDay, AnnualAllowance: 4, Requestable: true, RequiresDocument: true, RequiresApproval: true, MinRequest: 1, MaxRequest: 4, PayrollImpact: true},
		{ID: "gestion_servicio", Name: "GESTION DE SERVICIO", Unit: LeaveUnitHour, AnnualAllowance: 999 * 60, Requestable: true, RequiresApproval: true, MinRequest: 15, MaxRequest: 12 * 60},
	}
}

func FormatLeaveAmount(amount int, unit LeaveUnit) string {
	switch unit {
	case LeaveUnitHour:
		return fmt.Sprintf("%02d:%02d", amount/60, amount%60)
	default:
		return fmt.Sprintf("%d", amount)
	}
}
