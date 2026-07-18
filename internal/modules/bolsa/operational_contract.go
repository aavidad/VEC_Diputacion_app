package bolsa

import "strings"

type ModuleManifestContract struct {
	ModuleRef      string `json:"module_ref"`
	Version        string `json:"version"`
	TitleI18nKey   string `json:"title_i18n_key"`
	DescriptionKey string `json:"description_i18n_key"`
	CategoryRef    string `json:"category_ref"`
	BaseRoute      string `json:"base_route"`
	APIPrefix      string `json:"api_prefix"`
	PrototypeAPI   string `json:"prototype_api_prefix"`
	// AuthorizationPolicySource declara de donde debe obtenerse la decision,
	// pero nunca concede acceso. Los roles y concesiones son datos publicados y
	// versionados; no se incrustan en el manifiesto del modulo.
	AuthorizationPolicySource string            `json:"authorization_policy_source"`
	MenuEntries               []MenuEntry       `json:"menu_entries"`
	Capabilities              []Capability      `json:"capabilities"`
	EventsPublished           []string          `json:"events_published"`
	HealthRoute               string            `json:"health_route"`
	HTTPRoutes                []ModuleHTTPRoute `json:"http_routes"`
}

type MenuEntry struct {
	EntryRef            string   `json:"entry_ref"`
	LabelI18nKey        string   `json:"label_i18n_key"`
	Route               string   `json:"route"`
	RequiredPermissions []string `json:"required_permissions"`
}

type Capability struct {
	CapabilityRef string `json:"capability_ref"`
	Mode          string `json:"mode"`
	Method        string `json:"method,omitempty"`
	Route         string `json:"route,omitempty"`
	LabelI18nKey  string `json:"label_i18n_key"`
}

type ModuleHTTPRoute struct {
	Method string `json:"method"`
	Route  string `json:"route"`
	Mode   string `json:"mode"`
}

type OperationalStatus struct {
	ModuleRef            string              `json:"module_ref"`
	RuntimeMode          string              `json:"runtime_mode"`
	Status               string              `json:"status"`
	AuthMode             string              `json:"auth_mode"`
	PersistenceMode      string              `json:"persistence_mode"`
	DemoEnabled          bool                `json:"demo_enabled"`
	LegalProductionReady bool                `json:"legal_production_ready"`
	AdminRoutes          []ModuleHTTPRoute   `json:"admin_routes"`
	LegalIntegrations    []IntegrationStatus `json:"legal_integrations"`
}

type IntegrationStatus struct {
	IntegrationRef string `json:"integration_ref"`
	Status         string `json:"status"`
	Mode           string `json:"mode"`
}

type AdminCapabilities struct {
	ModuleRef         string              `json:"module_ref"`
	Capabilities      []Capability        `json:"capabilities"`
	HTTPRoutes        []ModuleHTTPRoute   `json:"http_routes"`
	LegalIntegrations []IntegrationStatus `json:"legal_integrations"`
}

func OperationalStatusForModes(demoEnabled bool, authMode, persistenceMode string) OperationalStatus {
	return OperationalStatus{
		ModuleRef:            ModuleID,
		RuntimeMode:          "local_productizable",
		Status:               "operational",
		AuthMode:             defaultMode(authMode, "disabled"),
		PersistenceMode:      defaultMode(persistenceMode, "memory"),
		DemoEnabled:          demoEnabled,
		LegalProductionReady: false,
		AdminRoutes:          AdminRoutes(),
		LegalIntegrations:    LegalIntegrations(),
	}
}

func OperationalStatusDefault(demoEnabled bool) OperationalStatus {
	return OperationalStatusForModes(demoEnabled, "disabled", "memory")
}

func defaultMode(value, fallback string) string {
	if trimmed := strings.TrimSpace(value); trimmed != "" {
		return trimmed
	}
	return fallback
}

func AdminCapabilitiesContract() AdminCapabilities {
	return AdminCapabilities{
		ModuleRef:         ModuleID,
		Capabilities:      AdminCapabilityList(),
		HTTPRoutes:        AdminRoutes(),
		LegalIntegrations: LegalIntegrations(),
	}
}

func AdminRoutes() []ModuleHTTPRoute {
	return []ModuleHTTPRoute{
		{Method: "GET", Route: "/api/admin/status", Mode: "real"},
		{Method: "GET", Route: "/api/admin/capabilities", Mode: "real"},
	}
}

func AdminCapabilityList() []Capability {
	return []Capability{
		{CapabilityRef: "bolsa.admin_status", Mode: "real", Method: "GET", Route: "/api/admin/status", LabelI18nKey: "ui.portal.adminStatusAction"},
		{CapabilityRef: "bolsa.admin_capabilities", Mode: "real", Method: "GET", Route: "/api/admin/capabilities", LabelI18nKey: "ui.portal.adminCapabilitiesAction"},
	}
}

func LegalIntegrations() []IntegrationStatus {
	return []IntegrationStatus{
		{IntegrationRef: "registro_electronico", Status: "not_configured", Mode: "external_legal"},
		{IntegrationRef: "firma_electronica", Status: "not_configured", Mode: "external_legal"},
		{IntegrationRef: "notificacion_feaciente", Status: "not_configured", Mode: "external_legal"},
		{IntegrationRef: "auditoria_probatoria_externa", Status: "not_configured", Mode: "external_legal"},
	}
}

func ModuleManifestForCandidatePortal() ModuleManifestContract {
	return ModuleManifestContract{
		ModuleRef:                 ModuleID,
		Version:                   "v1",
		TitleI18nKey:              "module.bolsa.title",
		DescriptionKey:            "module.bolsa.description",
		CategoryRef:               "seleccion_y_bolsas",
		BaseRoute:                 "/modules/bolsa",
		APIPrefix:                 "/api/modules/bolsa",
		PrototypeAPI:              "/api",
		AuthorizationPolicySource: "rbac_abac_published",
		MenuEntries: []MenuEntry{
			{EntryRef: "bolsa.dashboard", LabelI18nKey: "module.bolsa.menu.dashboard", Route: "/modules/bolsa", RequiredPermissions: []string{"bolsa.dashboard.read"}},
			{EntryRef: "bolsa.solicitudes", LabelI18nKey: "module.bolsa.menu.solicitudes", Route: "/modules/bolsa/solicitudes", RequiredPermissions: []string{PermissionRead}},
			{EntryRef: "bolsa.documentos", LabelI18nKey: "module.bolsa.menu.documentos", Route: "/modules/bolsa/documentos", RequiredPermissions: []string{PermissionDocument}},
			{EntryRef: "bolsa.alegaciones", LabelI18nKey: "module.bolsa.menu.alegaciones", Route: "/modules/bolsa/alegaciones", RequiredPermissions: []string{PermissionClaim}},
			{EntryRef: "bolsa.notificaciones", LabelI18nKey: "module.bolsa.menu.notificaciones", Route: "/modules/bolsa/notificaciones", RequiredPermissions: []string{PermissionNotification}},
			{EntryRef: "bolsa.auditoria", LabelI18nKey: "module.bolsa.menu.auditoria", Route: "/modules/bolsa/auditoria", RequiredPermissions: []string{PermissionAudit}},
		},
		Capabilities: []Capability{
			{CapabilityRef: "bolsa.candidates", Mode: "real", Method: "POST", Route: "/api/candidates", LabelI18nKey: "ui.portal.solicitudAction"},
			{CapabilityRef: "bolsa.merits", Mode: "real", Method: "POST", Route: "/api/candidates/{id}/merits", LabelI18nKey: "ui.portal.meritAction"},
			{CapabilityRef: "bolsa.documents", Mode: "real", Method: "POST", Route: "/api/candidates/{id}/documents", LabelI18nKey: "ui.portal.documentAction"},
			{CapabilityRef: "bolsa.claims", Mode: "real", Method: "POST", Route: "/api/candidates/{id}/claims", LabelI18nKey: "ui.portal.claimAction"},
			{CapabilityRef: "bolsa.notifications", Mode: "real", Method: "POST", Route: "/api/candidates/{id}/notifications", LabelI18nKey: "ui.portal.notificationAction"},
			{CapabilityRef: "bolsa.notification_send", Mode: "real", Method: "POST", Route: "/api/notifications/{id}/send", LabelI18nKey: "ui.portal.notificationSendAction"},
			{CapabilityRef: "bolsa.notification_read", Mode: "real", Method: "POST", Route: "/api/notifications/{id}/read", LabelI18nKey: "ui.portal.notificationReadAction"},
			{CapabilityRef: "bolsa.audit", Mode: "real", Method: "GET", Route: "/api/candidates/{id}/audit", LabelI18nKey: "ui.portal.auditAction"},
			{CapabilityRef: "bolsa.notification_list", Mode: "real", Method: "GET", Route: "/api/notifications?candidate_id={id}", LabelI18nKey: "ui.portal.notificationSyncAction"},
			{CapabilityRef: "bolsa.audit_scope", Mode: "real", Method: "GET", Route: "/api/audit?candidate_id={id}", LabelI18nKey: "ui.portal.auditAction"},
			{CapabilityRef: "bolsa.demo_listing", Mode: "demo", Method: "POST", Route: "/api/demo", LabelI18nKey: "ui.portal.convocatoriaAction"},
			AdminCapabilityList()[0],
			AdminCapabilityList()[1],
		},
		EventsPublished: []string{"bolsa.documento_registrado", "bolsa.alegacion_presentada", "bolsa.notificacion_creada"},
		HealthRoute:     "/api/modules/bolsa/healthz",
		HTTPRoutes: []ModuleHTTPRoute{
			{Method: "GET", Route: "/api/modules/bolsa", Mode: "real"},
			{Method: "GET", Route: "/api/modules/bolsa/manifest", Mode: "real"},
			{Method: "GET", Route: "/api/modules/bolsa/healthz", Mode: "real"},
			{Method: "GET", Route: "/api/candidates/{id}/documents", Mode: "real"},
			{Method: "POST", Route: "/api/candidates/{id}/documents", Mode: "real"},
			{Method: "GET", Route: "/api/candidates/{id}/claims", Mode: "real"},
			{Method: "POST", Route: "/api/candidates/{id}/claims", Mode: "real"},
			{Method: "GET", Route: "/api/candidates/{id}/notifications", Mode: "real"},
			{Method: "POST", Route: "/api/candidates/{id}/notifications", Mode: "real"},
			{Method: "GET", Route: "/api/notifications?candidate_id={id}", Mode: "real"},
			{Method: "POST", Route: "/api/notifications", Mode: "real"},
			{Method: "POST", Route: "/api/notifications/{id}/send", Mode: "real"},
			{Method: "POST", Route: "/api/notifications/{id}/read", Mode: "real"},
			{Method: "GET", Route: "/api/candidates/{id}/audit", Mode: "real"},
			{Method: "GET", Route: "/api/audit?candidate_id={id}", Mode: "real"},
			{Method: "GET", Route: "/api/admin/status", Mode: "real"},
			{Method: "GET", Route: "/api/admin/capabilities", Mode: "real"},
		},
	}
}
