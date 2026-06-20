package ports

import (
	"context"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/domain"
)

type Store interface {
	SaveProfile(context.Context, domain.ScheduleProfile) error
	ListProfiles(context.Context) ([]domain.ScheduleProfile, error)
	SaveWorkday(context.Context, domain.Workday) error
	ListWorkdays(context.Context, time.Time) ([]domain.Workday, error)
	SaveLeavePolicy(context.Context, domain.LeavePolicy) error
	ListLeavePolicies(context.Context) ([]domain.LeavePolicy, error)
	SaveLeaveBalance(context.Context, domain.LeaveBalance) error
	ListLeaveBalances(context.Context, string, int) ([]domain.LeaveBalance, error)
	SaveLeaveRequest(context.Context, domain.LeaveRequest) error
	ListLeaveRequests(context.Context, string, int) ([]domain.LeaveRequest, error)
}
