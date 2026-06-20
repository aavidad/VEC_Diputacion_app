package application

import "vec-diputacion-granada/internal/candidate/domain"

type ProfessionalPortalView struct {
	Locale         string                       `json:"locale"`
	Candidate      CandidateView                `json:"candidate"`
	Header         ProfessionalPortalHeaderView `json:"header"`
	Modules        []ProfessionalModuleView     `json:"modules"`
	Capabilities   []ProfessionalCapabilityView `json:"capabilities"`
	Summary        []ProfessionalMetricView     `json:"summary"`
	PendingActions []ProfessionalActionView     `json:"pending_actions"`
	Autobaremo     ProfessionalAutobaremoView   `json:"autobaremo"`
	Audit          ProfessionalAuditView        `json:"audit"`
}

type ProfessionalPortalHeaderView struct {
	TitleKey       string `json:"title_key"`
	StateKey       string `json:"state_key"`
	CallID         string `json:"call_id"`
	NextActionKey  string `json:"next_action_key"`
	DeadlineKey    string `json:"deadline_key"`
	LastUpdatedKey string `json:"last_updated_key"`
}

type ProfessionalModuleView struct {
	ID               string `json:"id"`
	Group            string `json:"group"`
	Accent           string `json:"accent"`
	Icon             string `json:"icon"`
	LabelKey         string `json:"label_key"`
	DescriptionKey   string `json:"description_key"`
	StatusKey        string `json:"status_key"`
	PrimaryActionKey string `json:"primary_action_key"`
	EmptyStateKey    string `json:"empty_state_key"`
	Count            int    `json:"count"`
	AlertCount       int    `json:"alert_count"`
}

type ProfessionalCapabilityView struct {
	ID       string `json:"id"`
	Mode     string `json:"mode"`
	Route    string `json:"route,omitempty"`
	Method   string `json:"method,omitempty"`
	LabelKey string `json:"label_key"`
}

type ProfessionalMetricView struct {
	ID       string  `json:"id"`
	LabelKey string  `json:"label_key"`
	Value    float64 `json:"value"`
	UnitKey  string  `json:"unit_key"`
	StateKey string  `json:"state_key"`
}

type ProfessionalActionView struct {
	ID          string `json:"id"`
	LabelKey    string `json:"label_key"`
	Priority    string `json:"priority"`
	ModuleID    string `json:"module_id"`
	DeadlineKey string `json:"deadline_key"`
}

type ProfessionalAutobaremoView struct {
	TotalPoints    float64                         `json:"total_points"`
	RuleSetID      string                          `json:"rule_set_id"`
	RuleSetVersion string                          `json:"rule_set_version"`
	Sections       []ProfessionalBaremoSectionView `json:"sections"`
	Details        []BaremoDetailView              `json:"details"`
	Warnings       []ProfessionalWarningView       `json:"warnings"`
	Explanation    []ProfessionalExplanationView   `json:"explanation"`
}

type ProfessionalBaremoSectionView struct {
	ID        string  `json:"id"`
	LabelKey  string  `json:"label_key"`
	Points    float64 `json:"points"`
	StatusKey string  `json:"status_key"`
}

type ProfessionalWarningView struct {
	Code       string `json:"code"`
	MessageKey string `json:"message_key"`
	ModuleID   string `json:"module_id"`
	Severity   string `json:"severity"`
}

type ProfessionalExplanationView struct {
	ID         string `json:"id"`
	MessageKey string `json:"message_key"`
	ReceiptKey string `json:"receipt_key"`
}

type ProfessionalAuditView struct {
	ReceiptKey     string `json:"receipt_key"`
	TimelineKey    string `json:"timeline_key"`
	LastActorKey   string `json:"last_actor_key"`
	LastChangedKey string `json:"last_changed_key"`
}

func NewProfessionalPortalView(expediente ExpedienteView) ProfessionalPortalView {
	modules := defaultProfessionalModules(expediente)
	return ProfessionalPortalView{
		Locale:    "es-ES",
		Candidate: expediente.Candidate,
		Header: ProfessionalPortalHeaderView{
			TitleKey:       "ui.portal.professional.title",
			StateKey:       "ui.portal.professional.state.draft_or_active",
			CallID:         expediente.Candidate.CallID,
			NextActionKey:  "ui.portal.professional.action.review_pending",
			DeadlineKey:    "ui.portal.professional.deadline.active_call",
			LastUpdatedKey: "ui.portal.professional.last_updated.audit",
		},
		Modules:        modules,
		Capabilities:   professionalCapabilities(),
		Summary:        professionalSummary(expediente),
		PendingActions: professionalActions(expediente),
		Autobaremo:     professionalAutobaremo(expediente.Baremo),
		Audit: ProfessionalAuditView{
			ReceiptKey:     "ui.portal.audit.receipt.pending",
			TimelineKey:    "ui.portal.audit.timeline.application",
			LastActorKey:   "ui.portal.audit.actor.candidate",
			LastChangedKey: "ui.portal.audit.changed.current_session",
		},
	}
}

func professionalCapabilities() []ProfessionalCapabilityView {
	return []ProfessionalCapabilityView{
		{ID: "candidates", Mode: "real", Method: "POST", Route: "/api/candidates", LabelKey: "ui.portal.solicitudAction"},
		{ID: "documents", Mode: "real", Method: "POST", Route: "/api/candidates/{id}/documents", LabelKey: "ui.portal.documentAction"},
		{ID: "claims", Mode: "real", Method: "POST", Route: "/api/candidates/{id}/claims", LabelKey: "ui.portal.claimAction"},
		{ID: "notifications", Mode: "real", Method: "POST", Route: "/api/candidates/{id}/notifications", LabelKey: "ui.portal.notificationAction"},
		{ID: "notification_send", Mode: "real", Method: "POST", Route: "/api/notifications/{id}/send", LabelKey: "ui.portal.notificationSendAction"},
		{ID: "notification_read", Mode: "real", Method: "POST", Route: "/api/notifications/{id}/read", LabelKey: "ui.portal.notificationReadAction"},
		{ID: "audit", Mode: "real", Method: "GET", Route: "/api/candidates/{id}/audit", LabelKey: "ui.portal.auditAction"},
		{ID: "audit_scope", Mode: "real", Method: "GET", Route: "/api/audit?candidate_id={id}", LabelKey: "ui.portal.auditAction"},
		{ID: "demo_listing", Mode: "demo", Method: "POST", Route: "/api/demo", LabelKey: "ui.portal.convocatoriaAction"},
	}
}

func defaultProfessionalModules(expediente ExpedienteView) []ProfessionalModuleView {
	meritCount := len(expediente.Merits)
	detailCount := len(expediente.Baremo.Details)
	return []ProfessionalModuleView{
		professionalModule("dashboard", "procedure", "indigo", "layout-dashboard", 4, 1),
		professionalModule("convocatorias", "procedure", "indigo", "calendar-days", 1, 0),
		professionalModule("expediente", "identity", "blue", "folder-open", 1, 0),
		professionalModule("meritos", "merit", "teal", "graduation-cap", meritCount, missingAlert(meritCount)),
		professionalModule("documentos", "evidence", "cyan", "file-check", meritCount, 0),
		professionalModule("autobaremo", "score", "violet", "calculator", detailCount, missingAlert(detailCount)),
		professionalModule("alegaciones", "claims", "orange", "message-square-warning", 0, 0),
		professionalModule("notificaciones", "communications", "amber", "mail-check", 1, 1),
		professionalModule("mensajes", "communications", "amber", "inbox", 1, 1),
		professionalModule("noticias", "communications", "slate", "newspaper", 1, 0),
		professionalModule("perfil", "identity", "blue", "user-round-cog", 1, 0),
		professionalModule("ayuda", "support", "green", "circle-help", 0, 0),
	}
}

func professionalModule(id, group, accent, icon string, count, alerts int) ProfessionalModuleView {
	return ProfessionalModuleView{
		ID: id, Group: group, Accent: accent, Icon: icon,
		LabelKey:         "ui.portal.module." + id + ".label",
		DescriptionKey:   "ui.portal.module." + id + ".description",
		StatusKey:        "ui.portal.module." + id + ".status",
		PrimaryActionKey: "ui.portal.module." + id + ".action",
		EmptyStateKey:    "ui.portal.module." + id + ".empty",
		Count:            count,
		AlertCount:       alerts,
	}
}

func professionalSummary(expediente ExpedienteView) []ProfessionalMetricView {
	return []ProfessionalMetricView{
		{ID: "total_points", LabelKey: "ui.portal.metric.total_points", Value: expediente.Baremo.TotalPoints, UnitKey: "ui.portal.unit.points", StateKey: "ui.portal.metric.state.provisional"},
		{ID: "merits", LabelKey: "ui.portal.metric.merits", Value: float64(len(expediente.Merits)), UnitKey: "ui.portal.unit.items", StateKey: "ui.portal.metric.state.reviewable"},
		{ID: "baremo_details", LabelKey: "ui.portal.metric.baremo_details", Value: float64(len(expediente.Baremo.Details)), UnitKey: "ui.portal.unit.items", StateKey: "ui.portal.metric.state.explained"},
	}
}

func professionalActions(expediente ExpedienteView) []ProfessionalActionView {
	actions := []ProfessionalActionView{
		{ID: "review_autobaremo", LabelKey: "ui.portal.action.review_autobaremo", Priority: "high", ModuleID: "autobaremo", DeadlineKey: "ui.portal.deadline.before_submit"},
		{ID: "check_notifications", LabelKey: "ui.portal.action.check_notifications", Priority: "medium", ModuleID: "notificaciones", DeadlineKey: "ui.portal.deadline.active"},
	}
	if len(expediente.Merits) == 0 {
		actions = append(actions, ProfessionalActionView{ID: "add_merit", LabelKey: "ui.portal.action.add_merit", Priority: "high", ModuleID: "meritos", DeadlineKey: "ui.portal.deadline.before_submit"})
	}
	return actions
}

func professionalAutobaremo(baremo BaremoView) ProfessionalAutobaremoView {
	view := ProfessionalAutobaremoView{
		TotalPoints: baremo.TotalPoints, RuleSetID: baremo.RuleSetID, RuleSetVersion: baremo.RuleSetVersion,
		Sections:    baremoSections(baremo),
		Details:     append([]BaremoDetailView(nil), baremo.Details...),
		Warnings:    baremoWarnings(baremo),
		Explanation: baremoExplanation(),
	}
	return view
}

func baremoSections(baremo BaremoView) []ProfessionalBaremoSectionView {
	sections := []domain.BaremoSection{
		domain.BaremoSectionExperiencia,
		domain.BaremoSectionFormacion,
		domain.BaremoSectionOtros,
	}
	result := make([]ProfessionalBaremoSectionView, 0, len(sections))
	for _, section := range sections {
		id := string(section)
		result = append(result, ProfessionalBaremoSectionView{
			ID: id, LabelKey: "ui.portal.autobaremo.section." + id,
			Points: baremo.SectionPoints[id], StatusKey: "ui.portal.autobaremo.section." + id + ".status",
		})
	}
	return result
}

func baremoWarnings(baremo BaremoView) []ProfessionalWarningView {
	warnings := []ProfessionalWarningView{}
	if len(baremo.Details) == 0 {
		warnings = append(warnings, ProfessionalWarningView{Code: "baremo_without_details", MessageKey: "ui.portal.autobaremo.warning.no_details", ModuleID: "autobaremo", Severity: "warning"})
	}
	for _, detail := range baremo.Details {
		if detail.Capped {
			warnings = append(warnings, ProfessionalWarningView{Code: "section_cap_applied", MessageKey: "ui.portal.autobaremo.warning.section_cap", ModuleID: "autobaremo", Severity: "info"})
			break
		}
	}
	return warnings
}

func baremoExplanation() []ProfessionalExplanationView {
	return []ProfessionalExplanationView{
		{ID: "ruleset", MessageKey: "ui.portal.autobaremo.explain.ruleset", ReceiptKey: "ui.portal.autobaremo.receipt.ruleset"},
		{ID: "sections", MessageKey: "ui.portal.autobaremo.explain.sections", ReceiptKey: "ui.portal.autobaremo.receipt.sections"},
		{ID: "caps", MessageKey: "ui.portal.autobaremo.explain.caps", ReceiptKey: "ui.portal.autobaremo.receipt.caps"},
	}
}

func missingAlert(count int) int {
	if count == 0 {
		return 1
	}
	return 0
}
