package application

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

var ErrServiceDependencyRequired = errors.New("cronos service dependency required")

type Service struct {
	store ports.Store
}

type Snapshot struct {
	Date          time.Time
	Profiles      []domain.ScheduleProfile
	Workdays      []domain.Workday
	Results       []domain.DayResult
	LeavePolicies []domain.LeavePolicy
	LeaveBalances []domain.LeaveBalance
	LeaveRequests []domain.LeaveRequest
}

func NewService(store ports.Store) (*Service, error) {
	if store == nil {
		return nil, ErrServiceDependencyRequired
	}
	return &Service{store: store}, nil
}

func (s *Service) SaveProfile(ctx context.Context, profile domain.ScheduleProfile) error {
	if err := profile.Validate(); err != nil {
		return err
	}
	return s.store.SaveProfile(ctx, profile)
}

func (s *Service) RegisterWorkday(ctx context.Context, workday domain.Workday) error {
	if _, err := domain.EvaluateWorkday(workday); err != nil {
		return err
	}
	return s.store.SaveWorkday(ctx, workday)
}

func (s *Service) SaveLeavePolicy(ctx context.Context, policy domain.LeavePolicy) error {
	if err := policy.Validate(); err != nil {
		return err
	}
	return s.store.SaveLeavePolicy(ctx, policy)
}

func (s *Service) SaveLeaveBalance(ctx context.Context, balance domain.LeaveBalance) error {
	if err := balance.Validate(); err != nil {
		return err
	}
	return s.store.SaveLeaveBalance(ctx, balance)
}

func (s *Service) RequestLeave(ctx context.Context, request domain.LeaveRequest) (domain.LeaveRequest, error) {
	if request.EmployeeID == "" || request.EmployeeID != strings.TrimSpace(request.EmployeeID) ||
		request.PolicyID == "" || request.PolicyID != strings.TrimSpace(request.PolicyID) {
		return domain.LeaveRequest{}, domain.ErrLeaveRequestInvalid
	}
	if request.CreatedAt.IsZero() {
		request.CreatedAt = time.Now().UTC()
	}
	if strings.TrimSpace(request.ID) == "" {
		request.ID = fmt.Sprintf("CRONOS-ABS-%d-%s-%s-%d", request.From.Year(), request.EmployeeID, request.PolicyID, request.CreatedAt.UnixNano())
	}
	if request.State == "" {
		request.State = domain.LeaveStateReview
	}
	policy, err := s.leavePolicy(ctx, request.PolicyID)
	if err != nil {
		return domain.LeaveRequest{}, err
	}
	balance, err := s.leaveBalance(ctx, request.EmployeeID, request.From.Year(), request.PolicyID)
	if err != nil {
		return domain.LeaveRequest{}, err
	}
	if err := request.Validate(policy, balance); err != nil {
		return domain.LeaveRequest{}, err
	}
	if err := s.store.SaveLeaveRequest(ctx, request); err != nil {
		return domain.LeaveRequest{}, err
	}
	return request, nil
}

func (s *Service) Profiles(ctx context.Context) ([]domain.ScheduleProfile, error) {
	profiles, err := s.store.ListProfiles(ctx)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(profiles, func(i, j int) bool { return profiles[i].ID < profiles[j].ID })
	return profiles, nil
}

func (s *Service) Snapshot(ctx context.Context, date time.Time, employeeIDs []string) (Snapshot, error) {
	empleados, err := validateEmployeeScope(employeeIDs)
	if date.IsZero() || err != nil {
		return Snapshot{}, domain.ErrWorkdayInvalid
	}
	profiles, err := s.Profiles(ctx)
	if err != nil {
		return Snapshot{}, err
	}
	workdays, err := s.store.ListWorkdays(ctx, empleados, date)
	if err != nil {
		return Snapshot{}, err
	}
	policies, err := s.store.ListLeavePolicies(ctx)
	if err != nil {
		return Snapshot{}, err
	}
	identificadores := make(map[string]struct{}, len(workdays))
	for _, workday := range workdays {
		if workday.EmployeeID == "" || workday.EmployeeID != strings.TrimSpace(workday.EmployeeID) {
			return Snapshot{}, domain.ErrWorkdayInvalid
		}
		identificadores[workday.EmployeeID] = struct{}{}
	}
	empleadosConJornada := make([]string, 0, len(identificadores))
	for employeeID := range identificadores {
		empleadosConJornada = append(empleadosConJornada, employeeID)
	}
	sort.Strings(empleadosConJornada)
	balances := make([]domain.LeaveBalance, 0)
	requests := make([]domain.LeaveRequest, 0)
	for _, employeeID := range empleadosConJornada {
		balancesEmpleado, err := s.store.ListLeaveBalances(ctx, employeeID, normalizeDate(date).Year())
		if err != nil {
			return Snapshot{}, err
		}
		balances = append(balances, balancesEmpleado...)
		requestsEmpleado, err := s.store.ListLeaveRequests(ctx, employeeID, normalizeDate(date).Year())
		if err != nil {
			return Snapshot{}, err
		}
		requests = append(requests, requestsEmpleado...)
	}
	results := make([]domain.DayResult, 0, len(workdays))
	for _, workday := range workdays {
		result, err := domain.EvaluateWorkday(workday)
		if err != nil {
			return Snapshot{}, err
		}
		results = append(results, result)
	}
	sort.SliceStable(workdays, func(i, j int) bool {
		if workdays[i].Date.Equal(workdays[j].Date) {
			return workdays[i].EmployeeID < workdays[j].EmployeeID
		}
		return workdays[i].Date.Before(workdays[j].Date)
	})
	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Date.Equal(results[j].Date) {
			return results[i].EmployeeID < results[j].EmployeeID
		}
		return results[i].Date.Before(results[j].Date)
	})
	return Snapshot{
		Date:          normalizeDate(date),
		Profiles:      profiles,
		Workdays:      workdays,
		Results:       results,
		LeavePolicies: policies,
		LeaveBalances: balances,
		LeaveRequests: requests,
	}, nil
}

func validateEmployeeScope(employeeIDs []string) ([]string, error) {
	if len(employeeIDs) == 0 || len(employeeIDs) > 1024 {
		return nil, domain.ErrWorkdayInvalid
	}
	seen := make(map[string]struct{}, len(employeeIDs))
	result := make([]string, 0, len(employeeIDs))
	for _, employeeID := range employeeIDs {
		if employeeID == "" || employeeID != strings.TrimSpace(employeeID) || strings.ContainsRune(employeeID, '*') {
			return nil, domain.ErrWorkdayInvalid
		}
		if _, duplicate := seen[employeeID]; duplicate {
			return nil, domain.ErrWorkdayInvalid
		}
		seen[employeeID] = struct{}{}
		result = append(result, employeeID)
	}
	sort.Strings(result)
	return result, nil
}

func (s *Service) LeavePolicies(ctx context.Context) ([]domain.LeavePolicy, error) {
	policies, err := s.store.ListLeavePolicies(ctx)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(policies, func(i, j int) bool { return policies[i].ID < policies[j].ID })
	return policies, nil
}

func (s *Service) LeaveBalances(ctx context.Context, employeeID string, year int) ([]domain.LeaveBalance, error) {
	if employeeID == "" || employeeID != strings.TrimSpace(employeeID) || year <= 0 {
		return nil, domain.ErrLeaveBalanceInvalid
	}
	return s.store.ListLeaveBalances(ctx, employeeID, year)
}

func (s *Service) LeaveRequests(ctx context.Context, employeeID string, year int) ([]domain.LeaveRequest, error) {
	if employeeID == "" || employeeID != strings.TrimSpace(employeeID) || year <= 0 {
		return nil, domain.ErrLeaveRequestInvalid
	}
	return s.store.ListLeaveRequests(ctx, employeeID, year)
}

func (s *Service) leavePolicy(ctx context.Context, policyID string) (domain.LeavePolicy, error) {
	if policyID == "" || policyID != strings.TrimSpace(policyID) {
		return domain.LeavePolicy{}, domain.ErrLeavePolicyInvalid
	}
	policies, err := s.store.ListLeavePolicies(ctx)
	if err != nil {
		return domain.LeavePolicy{}, err
	}
	for _, policy := range policies {
		if policy.ID == policyID {
			return policy, nil
		}
	}
	return domain.LeavePolicy{}, domain.ErrLeavePolicyInvalid
}

func (s *Service) leaveBalance(ctx context.Context, employeeID string, year int, policyID string) (domain.LeaveBalance, error) {
	if employeeID == "" || employeeID != strings.TrimSpace(employeeID) || policyID == "" ||
		policyID != strings.TrimSpace(policyID) || year <= 0 {
		return domain.LeaveBalance{}, domain.ErrLeaveBalanceInvalid
	}
	balances, err := s.store.ListLeaveBalances(ctx, employeeID, year)
	if err != nil {
		return domain.LeaveBalance{}, err
	}
	for _, balance := range balances {
		if balance.PolicyID == policyID {
			return balance, nil
		}
	}
	return domain.LeaveBalance{}, domain.ErrLeaveBalanceInvalid
}

func normalizeDate(value time.Time) time.Time {
	if value.IsZero() {
		return value
	}
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, value.Location())
}
