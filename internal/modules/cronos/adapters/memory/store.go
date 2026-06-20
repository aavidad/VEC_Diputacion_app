package memory

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/domain"
)

type Store struct {
	mu            sync.RWMutex
	profiles      map[string]domain.ScheduleProfile
	workdays      map[string]domain.Workday
	leavePolicies map[string]domain.LeavePolicy
	leaveBalances map[string]domain.LeaveBalance
	leaveRequests map[string]domain.LeaveRequest
}

func NewStore() *Store {
	return &Store{
		profiles:      map[string]domain.ScheduleProfile{},
		workdays:      map[string]domain.Workday{},
		leavePolicies: map[string]domain.LeavePolicy{},
		leaveBalances: map[string]domain.LeaveBalance{},
		leaveRequests: map[string]domain.LeaveRequest{},
	}
}

func (s *Store) SaveProfile(_ context.Context, profile domain.ScheduleProfile) error {
	if err := profile.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.profiles[profile.ID] = profile
	return nil
}

func (s *Store) ListProfiles(_ context.Context) ([]domain.ScheduleProfile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	profiles := make([]domain.ScheduleProfile, 0, len(s.profiles))
	for _, profile := range s.profiles {
		profiles = append(profiles, profile)
	}
	sort.SliceStable(profiles, func(i, j int) bool { return profiles[i].ID < profiles[j].ID })
	return profiles, nil
}

func (s *Store) SaveWorkday(_ context.Context, workday domain.Workday) error {
	if _, err := domain.EvaluateWorkday(workday); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.workdays[workdayKey(workday.EmployeeID, workday.Date)] = cloneWorkday(workday)
	return nil
}

func (s *Store) ListWorkdays(_ context.Context, date time.Time) ([]domain.Workday, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	filter := dayKey(date)
	workdays := make([]domain.Workday, 0, len(s.workdays))
	for _, workday := range s.workdays {
		if filter == "" || dayKey(workday.Date) == filter {
			workdays = append(workdays, cloneWorkday(workday))
		}
	}
	sort.SliceStable(workdays, func(i, j int) bool {
		if workdays[i].Date.Equal(workdays[j].Date) {
			return workdays[i].EmployeeID < workdays[j].EmployeeID
		}
		return workdays[i].Date.Before(workdays[j].Date)
	})
	return workdays, nil
}

func workdayKey(employeeID string, date time.Time) string {
	return strings.TrimSpace(employeeID) + "|" + dayKey(date)
}

func dayKey(date time.Time) string {
	if date.IsZero() {
		return ""
	}
	return date.Format("2006-01-02")
}

func cloneWorkday(workday domain.Workday) domain.Workday {
	workday.Punches = append([]domain.Punch(nil), workday.Punches...)
	workday.Absences = append([]domain.AuthorizedAbsence(nil), workday.Absences...)
	return workday
}

func (s *Store) SaveLeavePolicy(_ context.Context, policy domain.LeavePolicy) error {
	if err := policy.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.leavePolicies[policy.ID] = policy
	return nil
}

func (s *Store) ListLeavePolicies(_ context.Context) ([]domain.LeavePolicy, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	policies := make([]domain.LeavePolicy, 0, len(s.leavePolicies))
	for _, policy := range s.leavePolicies {
		policies = append(policies, policy)
	}
	sort.SliceStable(policies, func(i, j int) bool { return policies[i].ID < policies[j].ID })
	return policies, nil
}

func (s *Store) SaveLeaveBalance(_ context.Context, balance domain.LeaveBalance) error {
	if err := balance.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.leaveBalances[leaveBalanceKey(balance.EmployeeID, balance.Year, balance.PolicyID)] = balance
	return nil
}

func (s *Store) ListLeaveBalances(_ context.Context, employeeID string, year int) ([]domain.LeaveBalance, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	balances := make([]domain.LeaveBalance, 0, len(s.leaveBalances))
	for _, balance := range s.leaveBalances {
		if strings.TrimSpace(employeeID) != "" && balance.EmployeeID != strings.TrimSpace(employeeID) {
			continue
		}
		if year > 0 && balance.Year != year {
			continue
		}
		balances = append(balances, balance)
	}
	sort.SliceStable(balances, func(i, j int) bool {
		if balances[i].EmployeeID != balances[j].EmployeeID {
			return balances[i].EmployeeID < balances[j].EmployeeID
		}
		return balances[i].PolicyID < balances[j].PolicyID
	})
	return balances, nil
}

func (s *Store) SaveLeaveRequest(_ context.Context, request domain.LeaveRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.leaveRequests[request.ID] = request
	if balance, ok := s.leaveBalances[leaveBalanceKey(request.EmployeeID, request.From.Year(), request.PolicyID)]; ok {
		balance.Requested += request.Amount
		s.leaveBalances[leaveBalanceKey(request.EmployeeID, request.From.Year(), request.PolicyID)] = balance
	}
	return nil
}

func (s *Store) ListLeaveRequests(_ context.Context, employeeID string, year int) ([]domain.LeaveRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	requests := make([]domain.LeaveRequest, 0, len(s.leaveRequests))
	for _, request := range s.leaveRequests {
		if strings.TrimSpace(employeeID) != "" && request.EmployeeID != strings.TrimSpace(employeeID) {
			continue
		}
		if year > 0 && request.From.Year() != year {
			continue
		}
		requests = append(requests, request)
	}
	sort.SliceStable(requests, func(i, j int) bool {
		if requests[i].From.Equal(requests[j].From) {
			return requests[i].ID < requests[j].ID
		}
		return requests[i].From.Before(requests[j].From)
	})
	return requests, nil
}

func leaveBalanceKey(employeeID string, year int, policyID string) string {
	return strings.TrimSpace(employeeID) + "|" + dayYear(year) + "|" + strings.TrimSpace(policyID)
}

func dayYear(year int) string {
	if year <= 0 {
		return ""
	}
	return time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC).Format("2006")
}
