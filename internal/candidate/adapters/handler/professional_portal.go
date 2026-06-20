package handler

import (
	"context"
	"net/http"
	"strings"
	"time"

	"vec-diputacion-granada/internal/candidate/domain"
	"vec-diputacion-granada/internal/candidate/ports"
	"vec-diputacion-granada/internal/candidate/usecases"
)

type professionalPortalView struct {
	Principal professionalPortalPrincipalView `json:"principal"`
	Sections  []professionalPortalSectionView `json:"sections"`
	Routes    []string                        `json:"routes"`
}

type professionalPortalPrincipalView struct {
	Subject string         `json:"subject"`
	Role    ports.AuthRole `json:"role"`
}

type professionalPortalSectionView struct {
	Key     string   `json:"key"`
	Routes  []string `json:"routes"`
	Actions []string `json:"actions"`
}

type administrativeNotificationReceiptRequest struct {
	NotificationID string    `json:"notification_id"`
	CSV            string    `json:"csv"`
	RecipientID    string    `json:"recipient_id"`
	Channel        string    `json:"channel"`
	IssuedAt       time.Time `json:"issued_at"`
}

type administrativeGlobalNotificationRequest struct {
	CandidateID string    `json:"candidate_id"`
	ID          string    `json:"id"`
	SolicitudID string    `json:"solicitud_id"`
	Type        string    `json:"type"`
	Subject     string    `json:"subject"`
	Body        string    `json:"body"`
	CreatedAt   time.Time `json:"created_at"`
}

func (r administrativeGlobalNotificationRequest) candidateRequest() administrativeNotificationRequest {
	return administrativeNotificationRequest{
		ID: r.ID, SolicitudID: r.SolicitudID, Type: r.Type,
		Subject: r.Subject, Body: r.Body, CreatedAt: r.CreatedAt,
	}
}

func (h *Handler) handleProfessionalPortalRoute(
	w http.ResponseWriter,
	r *http.Request,
	principal ports.AuthPrincipal,
) {
	if !h.requireStaff(w, principal) || !h.requireMethod(w, r, http.MethodGet) {
		return
	}
	h.writeJSON(w, http.StatusOK, responseEnvelope{
		Data: professionalPortalView{
			Principal: professionalPortalPrincipalView{
				Subject: principal.Subject,
				Role:    principal.Role,
			},
			Sections: professionalPortalSections(),
			Routes:   apiRoutes(),
		},
	})
}

func professionalPortalSections() []professionalPortalSectionView {
	return []professionalPortalSectionView{
		{
			Key:     "procedure_dashboard",
			Routes:  []string{"/api/demo"},
			Actions: []string{"review_procedure", "publish_listing"},
		},
		{
			Key:     "candidate_management",
			Routes:  []string{"/api/candidates", "/api/candidates/{id}/claims", "/api/candidates/{id}/notifications", "/api/candidates/{id}/audit", "/api/notifications", "/api/notifications?candidate_id={id}", "/api/audit?candidate_id={id}"},
			Actions: []string{"review_candidate", "export_expediente", "present_claim", "create_notification", "send_notification", "mark_notification_read", "review_audit"},
		},
		{
			Key:     "operations_status",
			Routes:  []string{"/api/admin/status", "/api/admin/capabilities"},
			Actions: []string{"check_operational_status", "review_capabilities"},
		},
	}
}

func apiRoutes() []string {
	return []string{
		"/api/demo",
		"/api/portal",
		"/api/admin/status",
		"/api/admin/capabilities",
		"/api/candidates",
		"/api/candidates/{id}/merits",
		"/api/candidates/{id}/baremo",
		"/api/candidates/{id}/expediente",
		"/api/candidates/{id}/documents",
		"/api/candidates/{id}/claims",
		"/api/candidates/{id}/notifications",
		"/api/candidates/{id}/audit",
		"/api/notifications",
		"/api/notifications?candidate_id={id}",
		"/api/notifications/{id}/send",
		"/api/notifications/{id}/read",
		"/api/audit?candidate_id={id}",
	}
}

func (s administrativeFlowService) SendNotification(
	ctx context.Context,
	principal ports.AuthPrincipal,
	request administrativeNotificationReceiptRequest,
) (administrativeNotificationView, error) {
	return s.applyNotificationReceipt(ctx, principal, request, (*usecases.AdministrativeFlowUseCase).SendNotification)
}

func (s administrativeFlowService) MarkNotificationRead(
	ctx context.Context,
	principal ports.AuthPrincipal,
	request administrativeNotificationReceiptRequest,
) (administrativeNotificationView, error) {
	return s.applyNotificationReceipt(ctx, principal, request, (*usecases.AdministrativeFlowUseCase).MarkNotificationRead)
}

func (s administrativeFlowService) applyNotificationReceipt(
	ctx context.Context,
	principal ports.AuthPrincipal,
	request administrativeNotificationReceiptRequest,
	apply func(*usecases.AdministrativeFlowUseCase, context.Context, usecases.ReceiptCommand) (domain.Notification, domain.AuditEntry, error),
) (administrativeNotificationView, error) {
	at := request.IssuedAt
	if at.IsZero() {
		at = s.now().UTC()
	}
	notification, audit, err := apply(s.usecase, ctx, usecases.ReceiptCommand{
		NotificationID: strings.TrimSpace(request.NotificationID),
		CSV:            strings.TrimSpace(request.CSV),
		RecipientID:    strings.TrimSpace(defaultString(request.RecipientID, principal.Subject)),
		Channel:        strings.TrimSpace(defaultString(request.Channel, "vec")),
		IssuedAt:       at,
	})
	if err != nil {
		return administrativeNotificationView{}, err
	}
	view := notificationView(notification)
	view.AuditSequence = audit.Sequence
	return view, nil
}

func (s administrativeFlowService) ListAuditByScope(ctx context.Context, scope string) ([]administrativeAuditView, error) {
	entries, err := s.usecase.ListAuditByScope(ctx, scope)
	if err != nil {
		return nil, err
	}
	views := make([]administrativeAuditView, 0, len(entries))
	for _, entry := range entries {
		views = append(views, auditView(entry))
	}
	return views, nil
}
