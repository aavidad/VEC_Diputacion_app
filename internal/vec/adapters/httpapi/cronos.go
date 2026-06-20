package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	cronosmodule "vec-diputacion-granada/internal/modules/cronos"
	cronosdomain "vec-diputacion-granada/internal/modules/cronos/domain"
	vecapp "vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/domain"
)

type cronosWorkdayRequest struct {
	EmployeeID        string               `json:"employee_id"`
	Date              string               `json:"date"`
	Age               int                  `json:"age"`
	ProfileID         string               `json:"profile_id"`
	TeleworkMinutes   int                  `json:"telework_minutes"`
	AuthorizedAbsence []cronosAbsenceInput `json:"authorized_absence,omitempty"`
	Punches           []cronosPunchInput   `json:"punches"`
}

type cronosPunchInput struct {
	At      string `json:"at"`
	Kind    string `json:"kind"`
	Channel string `json:"channel"`
	Mode    string `json:"mode"`
}

type cronosAbsenceInput struct {
	Ref     string `json:"ref"`
	Kind    string `json:"kind"`
	Minutes int    `json:"minutes"`
	Paid    bool   `json:"paid"`
}

type cronosLeaveRequestInput struct {
	EmployeeID  string `json:"employee_id"`
	PolicyID    string `json:"policy_id"`
	From        string `json:"from"`
	To          string `json:"to"`
	Amount      int    `json:"amount"`
	Reason      string `json:"reason"`
	DocumentRef string `json:"document_ref"`
}

func (h *Handler) handleCronosTimecards(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	switch r.Method {
	case http.MethodGet:
		if !principal.HasPermission(cronosmodule.PermissionTimeRead) {
			h.writeError(w, http.StatusForbidden, domain.ErrPermissionDenied.Error())
			return
		}
		h.writeJSON(w, http.StatusOK, workspaceCronosData(r.Context(), h.cronos))
	case http.MethodPost:
		if !principal.HasPermission(cronosmodule.PermissionTimeManage) {
			h.writeError(w, http.StatusForbidden, domain.ErrPermissionDenied.Error())
			return
		}
		h.handleCronosTimecardCreate(w, r, principal)
	default:
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *Handler) handleCronosLeaveRequests(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	switch r.Method {
	case http.MethodGet:
		if !principal.HasPermission(cronosmodule.PermissionLeaveRead) {
			h.writeError(w, http.StatusForbidden, domain.ErrPermissionDenied.Error())
			return
		}
		h.writeJSON(w, http.StatusOK, workspaceCronosData(r.Context(), h.cronos))
	case http.MethodPost:
		if !principal.HasPermission(cronosmodule.PermissionLeaveManage) {
			h.writeError(w, http.StatusForbidden, domain.ErrPermissionDenied.Error())
			return
		}
		h.handleCronosLeaveRequestCreate(w, r, principal)
	default:
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *Handler) handleCronosLeaveRequestCreate(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	var input cronosLeaveRequestInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid cronos leave payload")
		return
	}
	request, err := h.leaveRequestFromInput(r, input)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	request, err = h.cronos.RequestLeave(r.Context(), request)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	audit, err := h.service.RecordAudit(r.Context(), vecapp.AuditCommand{
		Principal:  principal,
		Action:     "cronos.permiso.request",
		ModuleID:   cronosmodule.ModuleID,
		SubjectRef: request.ID,
		Result:     "accepted",
		Metadata: map[string]string{
			"receipt_type": "cronos.leave_request",
			"employee_id":  request.EmployeeID,
			"policy_id":    request.PolicyID,
			"source":       "httpapi",
		},
	})
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	policies, err := h.cronos.LeavePolicies(r.Context())
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	leaveRequest := map[string]any{}
	if views := leaveRequestViews([]cronosdomain.LeaveRequest{request}, policies); len(views) > 0 {
		leaveRequest = views[0]
	}
	h.writeJSON(w, http.StatusCreated, map[string]any{
		"leave_request": leaveRequest,
		"receipt":       audit,
	})
}

func (h *Handler) handleCronosTimecardCreate(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	var input cronosWorkdayRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid cronos timecard payload")
		return
	}
	workday, err := h.workdayFromRequest(r, input)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.cronos.RegisterWorkday(r.Context(), workday); err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := cronosdomain.EvaluateWorkday(workday)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	audit, err := h.service.RecordAudit(r.Context(), vecapp.AuditCommand{
		Principal:  principal,
		Action:     "cronos.jornada.register",
		ModuleID:   cronosmodule.ModuleID,
		SubjectRef: workday.EmployeeID + "|" + workday.Date.Format("2006-01-02"),
		Result:     "accepted",
		Metadata: map[string]string{
			"receipt_type": "cronos.timecard",
			"profile_id":   workday.Profile.ID,
			"source":       "httpapi",
		},
	})
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.writeJSON(w, http.StatusCreated, map[string]any{
		"timecard": timecardView(timecardID(result.EmployeeID), timecardTitle(result), result, timecardState(result)),
		"receipt":  audit,
	})
}

func (h *Handler) leaveRequestFromInput(r *http.Request, input cronosLeaveRequestInput) (cronosdomain.LeaveRequest, error) {
	from, err := parseCronosDateTime(input.From)
	if err != nil {
		return cronosdomain.LeaveRequest{}, err
	}
	to, err := parseCronosDateTime(input.To)
	if err != nil {
		return cronosdomain.LeaveRequest{}, err
	}
	policy, err := h.leavePolicy(r, input.PolicyID)
	if err != nil {
		return cronosdomain.LeaveRequest{}, err
	}
	amount := input.Amount
	if amount <= 0 && policy.Unit == cronosdomain.LeaveUnitDay {
		amount = int(to.Sub(from).Hours()/24) + 1
	}
	return cronosdomain.LeaveRequest{
		EmployeeID:  strings.TrimSpace(input.EmployeeID),
		PolicyID:    strings.TrimSpace(input.PolicyID),
		From:        from,
		To:          to,
		Amount:      amount,
		Unit:        policy.Unit,
		State:       cronosdomain.LeaveStateReview,
		Reason:      strings.TrimSpace(input.Reason),
		DocumentRef: strings.TrimSpace(input.DocumentRef),
	}, nil
}

func (h *Handler) leavePolicy(r *http.Request, policyID string) (cronosdomain.LeavePolicy, error) {
	policies, err := h.cronos.LeavePolicies(r.Context())
	if err != nil {
		return cronosdomain.LeavePolicy{}, err
	}
	policyID = strings.TrimSpace(policyID)
	for _, policy := range policies {
		if policy.ID == policyID {
			return policy, nil
		}
	}
	return cronosdomain.LeavePolicy{}, cronosdomain.ErrLeavePolicyInvalid
}

func (h *Handler) workdayFromRequest(r *http.Request, input cronosWorkdayRequest) (cronosdomain.Workday, error) {
	date, err := parseCronosDate(input.Date)
	if err != nil {
		return cronosdomain.Workday{}, err
	}
	profile, err := h.cronosProfile(r, input.ProfileID)
	if err != nil {
		return cronosdomain.Workday{}, err
	}
	punches := make([]cronosdomain.Punch, 0, len(input.Punches))
	for _, item := range input.Punches {
		at, err := parseCronosTime(date, item.At)
		if err != nil {
			return cronosdomain.Workday{}, err
		}
		mode := cronosdomain.WorkMode(strings.TrimSpace(item.Mode))
		if mode == "" {
			mode = cronosdomain.WorkModeOnSite
		}
		punches = append(punches, cronosdomain.Punch{
			At:      at,
			Kind:    cronosdomain.PunchKind(strings.TrimSpace(item.Kind)),
			Channel: strings.TrimSpace(item.Channel),
			Mode:    mode,
		})
	}
	absences := make([]cronosdomain.AuthorizedAbsence, 0, len(input.AuthorizedAbsence))
	for _, item := range input.AuthorizedAbsence {
		absences = append(absences, cronosdomain.AuthorizedAbsence{
			Ref:      strings.TrimSpace(item.Ref),
			Kind:     strings.TrimSpace(item.Kind),
			Duration: time.Duration(item.Minutes) * time.Minute,
			Paid:     item.Paid,
		})
	}
	return cronosdomain.Workday{
		EmployeeID:       strings.TrimSpace(input.EmployeeID),
		Date:             date,
		Age:              input.Age,
		Profile:          profile,
		Punches:          punches,
		TeleworkDuration: time.Duration(input.TeleworkMinutes) * time.Minute,
		Absences:         absences,
	}, nil
}

func (h *Handler) cronosProfile(r *http.Request, profileID string) (cronosdomain.ScheduleProfile, error) {
	profiles, err := h.cronos.Profiles(r.Context())
	if err != nil {
		return cronosdomain.ScheduleProfile{}, err
	}
	profileID = strings.TrimSpace(profileID)
	if profileID == "" {
		profileID = "H-FLEX-ADM"
	}
	for _, profile := range profiles {
		if profile.ID == profileID {
			return profile, nil
		}
	}
	return cronosdomain.ScheduleProfile{}, cronosdomain.ErrScheduleProfileInvalid
}

func parseCronosDate(value string) (time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return workspaceCronosDate, nil
	}
	return time.Parse("2006-01-02", strings.TrimSpace(value))
}

func parseCronosDateTime(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if strings.Contains(value, "T") {
		return time.Parse(time.RFC3339, value)
	}
	return time.Parse("2006-01-02", value)
}

func parseCronosTime(day time.Time, value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if strings.Contains(value, "T") {
		return time.Parse(time.RFC3339, value)
	}
	clock, err := time.Parse("15:04", value)
	if err != nil {
		return time.Time{}, err
	}
	return time.Date(day.Year(), day.Month(), day.Day(), clock.Hour(), clock.Minute(), 0, 0, day.Location()), nil
}
