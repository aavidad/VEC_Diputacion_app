package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	bolsamodule "vec-diputacion-granada/internal/modules/bolsa"
	cronosmodule "vec-diputacion-granada/internal/modules/cronos"
	cronosapp "vec-diputacion-granada/internal/modules/cronos/application"
	dietasmodule "vec-diputacion-granada/internal/modules/dietas"
	personalmodule "vec-diputacion-granada/internal/modules/personal"
	personalapp "vec-diputacion-granada/internal/modules/personal/application"
	"vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/domain"
)

type Handler struct {
	service         *application.Service
	cronos          *cronosapp.Service
	personalCatalog *personalapp.CatalogService
}

func NewHandler(service *application.Service) (*Handler, error) {
	if service == nil {
		return nil, errors.New("vec http handler: service required")
	}
	cronos, err := newWorkspaceCronosService()
	if err != nil {
		return nil, err
	}
	personalCatalog, err := newWorkspacePersonalCatalogService()
	if err != nil {
		return nil, err
	}
	return &Handler{service: service, cronos: cronos, personalCatalog: personalCatalog}, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := vecPath(r.URL.Path)
	principal := principalFromRequest(r)
	switch {
	case path == "/":
		if !h.requireMethod(w, r, http.MethodGet) {
			return
		}
		h.writeJSON(w, http.StatusOK, map[string]any{
			"routes": vecRoutes(),
		})
	case path == "/session":
		if !h.requireMethod(w, r, http.MethodGet) {
			return
		}
		h.writeJSON(w, http.StatusOK, map[string]any{"principal": principal})
	case path == "/modules":
		h.handleModules(w, r)
	case path == "/workspace":
		h.handleWorkspace(w, r)
	case path == "/cronos/timecards":
		h.handleCronosTimecards(w, r, principal)
	case path == "/cronos/leave-requests":
		h.handleCronosLeaveRequests(w, r, principal)
	case path == "/personal/rpt/positions":
		h.handlePersonalRPTPositions(w, r, principal)
	case strings.HasPrefix(path, "/personal/rpt/positions/"):
		h.handlePersonalRPTPosition(w, r, principal, path)
	case path == "/personal/rpt/imports":
		h.handlePersonalRPTImports(w, r, principal)
	case path == "/personal/rpt/stats":
		h.handlePersonalRPTStats(w, r, principal)
	case path == "/personal/categories":
		h.handlePersonalCategories(w, r, principal)
	case strings.HasPrefix(path, "/personal/categories/"):
		h.handlePersonalCategory(w, r, principal, path)
	case path == "/personal/catalogs":
		h.handlePersonalCatalogs(w, r, principal)
	case path == "/menu":
		h.handleMenu(w, r, principal)
	case path == "/audit":
		h.handleAudit(w, r)
	case strings.HasPrefix(path, "/modules/") && strings.HasSuffix(path, "/action"):
		h.handleModuleAction(w, r, principal, path)
	default:
		h.writeError(w, http.StatusNotFound, "vec route not found")
	}
}

func (h *Handler) handleModules(w http.ResponseWriter, r *http.Request) {
	if !h.requireMethod(w, r, http.MethodGet) {
		return
	}
	modules, err := h.service.Modules(r.Context())
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"modules": modules})
}

func (h *Handler) handleMenu(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	if !h.requireMethod(w, r, http.MethodGet) {
		return
	}
	menu, err := h.service.BuildMenu(r.Context(), principal)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"menu": menu, "principal": principal})
}

func (h *Handler) handleAudit(w http.ResponseWriter, r *http.Request) {
	if !h.requireMethod(w, r, http.MethodGet) {
		return
	}
	audit, err := h.service.Audit(r.Context(), r.URL.Query().Get("subject_ref"))
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"audit": audit})
}

func (h *Handler) handleModuleAction(w http.ResponseWriter, r *http.Request, principal domain.Principal, path string) {
	if !h.requireMethod(w, r, http.MethodPost) {
		return
	}
	action, ok := actionForPath(path)
	if !ok {
		h.writeError(w, http.StatusNotFound, "vec module action not found")
		return
	}
	if !principal.HasPermission(action.permission) {
		h.writeError(w, http.StatusForbidden, domain.ErrPermissionDenied.Error())
		return
	}
	audit, err := h.service.RecordAudit(r.Context(), application.AuditCommand{
		Principal:  principal,
		Action:     action.action,
		ModuleID:   action.moduleID,
		SubjectRef: action.subjectRef,
		Result:     "accepted",
		Metadata: map[string]string{
			"receipt_type": "vec.module.action",
			"source":       "httpapi",
			"module_key":   action.key,
		},
	})
	if err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	_ = h.service.PublishEvent(r.Context(), domain.Event{
		Type:       action.eventType,
		ModuleID:   action.moduleID,
		SubjectRef: audit.SubjectRef,
		ActorID:    principal.ID,
		OccurredAt: time.Now().UTC(),
		Payload:    map[string]string{"audit_id": audit.ID},
	})
	h.writeJSON(w, http.StatusAccepted, map[string]any{"receipt": audit})
}

type moduleAction struct {
	key        string
	moduleID   string
	permission string
	action     string
	subjectRef string
	eventType  string
}

func actionForPath(path string) (moduleAction, bool) {
	key := strings.TrimSuffix(strings.TrimPrefix(path, "/modules/"), "/action")
	actions := map[string]moduleAction{
		"cronos": {
			key:        "cronos",
			moduleID:   cronosmodule.ModuleID,
			permission: cronosmodule.PermissionApprovalManage,
			action:     cronosmodule.ActionReviewJustification,
			subjectRef: "cronos-incidencia-demo",
			eventType:  "vec.module.cronos.action.executed",
		},
		"horarios": {
			key:        "horarios",
			moduleID:   cronosmodule.ModuleID,
			permission: cronosmodule.PermissionScheduleManage,
			action:     cronosmodule.ActionReviewJustification,
			subjectRef: "cronos-horario-demo",
			eventType:  "vec.module.cronos.schedule.executed",
		},
		"permisos": {
			key:        "permisos",
			moduleID:   cronosmodule.ModuleID,
			permission: cronosmodule.PermissionLeaveManage,
			action:     cronosmodule.ActionReviewLeaveAndHoliday,
			subjectRef: "cronos-permiso-demo",
			eventType:  "vec.module.cronos.leave.executed",
		},
		"dietas": {
			key:        "dietas",
			moduleID:   dietasmodule.ModuleID,
			permission: dietasmodule.PermissionApprovalManage,
			action:     dietasmodule.ActionReviewTravelExpense,
			subjectRef: "dietas-comision-demo",
			eventType:  "vec.module.dietas.action.executed",
		},
		"rutas": {
			key:        "rutas",
			moduleID:   dietasmodule.ModuleID,
			permission: dietasmodule.PermissionRouteManage,
			action:     dietasmodule.ActionReviewRouteKM,
			subjectRef: "dietas-ruta-demo",
			eventType:  "vec.module.dietas.route.executed",
		},
		"bolsa": {
			key:        "bolsa",
			moduleID:   bolsamodule.ModuleID,
			permission: bolsamodule.PermissionDemoAction,
			action:     bolsamodule.ActionDemoIntegration,
			subjectRef: "bolsa-demo-action",
			eventType:  "vec.module.bolsa.action.executed",
		},
		"personal": {
			key:        "personal",
			moduleID:   personalmodule.ModuleID,
			permission: personalmodule.PermissionCertificateManage,
			action:     personalmodule.ActionIssueServiceCertificate,
			subjectRef: "personal-certificado-servicios-demo",
			eventType:  "vec.module.personal.action.executed",
		},
		"nominas": {
			key:        "nominas",
			moduleID:   personalmodule.ModuleID,
			permission: personalmodule.PermissionPayrollManage,
			action:     personalmodule.ActionReviewPayrollIncident,
			subjectRef: "personal-nomina-demo",
			eventType:  "vec.module.personal.payroll.executed",
		},
	}
	action, ok := actions[key]
	return action, ok
}

func vecRoutes() []string {
	return []string{
		"/api/vec/session",
		"/api/vec/modules",
		"/api/vec/workspace",
		"/api/vec/menu",
		"/api/vec/audit",
		"/api/vec/cronos/timecards",
		"/api/vec/cronos/leave-requests",
		"/api/vec/personal/rpt/positions",
		"/api/vec/personal/rpt/positions/{code}",
		"/api/vec/personal/rpt/imports",
		"/api/vec/personal/rpt/stats",
		"/api/vec/personal/categories",
		"/api/vec/personal/categories/{slug}",
		"/api/vec/personal/catalogs",
		"/api/vec/modules/cronos/action",
		"/api/vec/modules/horarios/action",
		"/api/vec/modules/permisos/action",
		"/api/vec/modules/dietas/action",
		"/api/vec/modules/rutas/action",
		"/api/vec/modules/bolsa/action",
		"/api/vec/modules/personal/action",
		"/api/vec/modules/nominas/action",
	}
}

func (h *Handler) requireMethod(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method == method {
		return true
	}
	w.Header().Set("Allow", method)
	h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	return false
}

func (h *Handler) writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
}

func (h *Handler) writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": message})
}

func vecPath(path string) string {
	if path == "/api/vec" {
		return "/"
	}
	return strings.TrimPrefix(path, "/api/vec")
}

func principalFromRequest(r *http.Request) domain.Principal {
	identity := identityFromRequest(r)
	principal := domain.Principal{
		ID:            identity.subject,
		DisplayName:   identity.displayName,
		Email:         identity.email,
		Roles:         identity.roles,
		AuthMethod:    identity.method,
		AuthAssurance: identity.assurance,
		Permissions:   permissionsForRoles(identity.roles),
		Attributes:    identity.attributes,
	}
	return principal
}

func authMethod(value string) domain.AuthMethod {
	switch value {
	case string(domain.AuthMethodCertificate):
		return domain.AuthMethodCertificate
	case string(domain.AuthMethodDNIe):
		return domain.AuthMethodDNIe
	case string(domain.AuthMethodClave):
		return domain.AuthMethodClave
	case string(domain.AuthMethodKerberos):
		return domain.AuthMethodKerberos
	case string(domain.AuthMethodSSO):
		return domain.AuthMethodSSO
	default:
		return domain.AuthMethodDemo
	}
}

func permissionsForRoles(roles []string) []string {
	if hasRole(roles, "ciudadano") {
		return []string{
			bolsamodule.PermissionRead,
			bolsamodule.PermissionDocument,
			bolsamodule.PermissionClaim,
			bolsamodule.PermissionNotification,
		}
	}
	permissions := baseVecPermissions()
	if hasAnyRole(roles, "administrador", "jefatura_rrhh") {
		return dedupePermissions(append(permissions, privilegedStaffPermissions()...))
	}
	if hasRole(roles, "tecnico_rrhh") {
		return dedupePermissions(append(permissions,
			personalmodule.PermissionEmployeeRead,
			personalmodule.PermissionEmployeeManage,
			personalmodule.PermissionPositionRead,
			personalmodule.PermissionPositionManage,
			personalmodule.PermissionPayrollRead,
			personalmodule.PermissionPayrollManage,
			personalmodule.PermissionSeniorityRead,
			personalmodule.PermissionCertificateManage,
			personalmodule.PermissionAdministrativeManage,
			personalmodule.PermissionAudit,
			cronosmodule.PermissionTimeRead,
			cronosmodule.PermissionTimeManage,
			cronosmodule.PermissionScheduleRead,
			cronosmodule.PermissionScheduleManage,
			cronosmodule.PermissionLeaveRead,
			cronosmodule.PermissionLeaveManage,
			cronosmodule.PermissionApprovalManage,
			cronosmodule.PermissionAudit,
			dietasmodule.PermissionExpenseRead,
			dietasmodule.PermissionRouteRead,
			bolsamodule.PermissionRead,
			bolsamodule.PermissionManage,
			bolsamodule.PermissionDocument,
			bolsamodule.PermissionClaim,
			bolsamodule.PermissionNotification,
			bolsamodule.PermissionDemoAction,
			bolsamodule.PermissionAudit,
		))
	}
	if hasAnyRole(roles, "jefe_servicio", "jefe_seccion") {
		return dedupePermissions(append(permissions,
			personalmodule.PermissionEmployeeRead,
			personalmodule.PermissionPositionRead,
			cronosmodule.PermissionTimeRead,
			cronosmodule.PermissionTimeManage,
			cronosmodule.PermissionScheduleRead,
			cronosmodule.PermissionLeaveRead,
			cronosmodule.PermissionLeaveManage,
			cronosmodule.PermissionApprovalManage,
			dietasmodule.PermissionExpenseRead,
			dietasmodule.PermissionExpenseManage,
			dietasmodule.PermissionRouteRead,
			dietasmodule.PermissionApprovalManage,
			bolsamodule.PermissionRead,
			bolsamodule.PermissionDocument,
			bolsamodule.PermissionNotification,
		))
	}
	if hasRole(roles, "administrativo") {
		return dedupePermissions(append(permissions,
			personalmodule.PermissionEmployeeRead,
			personalmodule.PermissionEmployeeManage,
			personalmodule.PermissionPositionRead,
			cronosmodule.PermissionTimeRead,
			cronosmodule.PermissionLeaveRead,
			cronosmodule.PermissionLeaveManage,
			dietasmodule.PermissionExpenseRead,
			dietasmodule.PermissionRouteRead,
			bolsamodule.PermissionRead,
			bolsamodule.PermissionManage,
			bolsamodule.PermissionDocument,
			bolsamodule.PermissionClaim,
			bolsamodule.PermissionNotification,
		))
	}
	return dedupePermissions(append(permissions,
		personalmodule.PermissionEmployeeRead,
		personalmodule.PermissionPositionRead,
		cronosmodule.PermissionTimeRead,
		cronosmodule.PermissionLeaveRead,
		dietasmodule.PermissionExpenseRead,
		dietasmodule.PermissionRouteRead,
		bolsamodule.PermissionRead,
		bolsamodule.PermissionDocument,
	))
}

func baseVecPermissions() []string {
	return []string{"vec.modules.read", "vec.menu.read"}
}

func privilegedStaffPermissions() []string {
	return []string{
		"vec.audit.read",
		"vec.roles.manage",
		"vec.catalogs.manage",
		personalmodule.PermissionEmployeeRead,
		personalmodule.PermissionEmployeeManage,
		personalmodule.PermissionPositionRead,
		personalmodule.PermissionPositionManage,
		personalmodule.PermissionPayrollRead,
		personalmodule.PermissionPayrollManage,
		personalmodule.PermissionSeniorityRead,
		personalmodule.PermissionCertificateManage,
		personalmodule.PermissionAdministrativeManage,
		personalmodule.PermissionAudit,
		cronosmodule.PermissionTimeRead,
		cronosmodule.PermissionTimeManage,
		cronosmodule.PermissionScheduleRead,
		cronosmodule.PermissionScheduleManage,
		cronosmodule.PermissionLeaveRead,
		cronosmodule.PermissionLeaveManage,
		cronosmodule.PermissionApprovalManage,
		cronosmodule.PermissionAudit,
		dietasmodule.PermissionExpenseRead,
		dietasmodule.PermissionExpenseManage,
		dietasmodule.PermissionRouteRead,
		dietasmodule.PermissionRouteManage,
		dietasmodule.PermissionApprovalManage,
		dietasmodule.PermissionAudit,
		bolsamodule.PermissionRead,
		bolsamodule.PermissionManage,
		bolsamodule.PermissionDocument,
		bolsamodule.PermissionClaim,
		bolsamodule.PermissionNotification,
		bolsamodule.PermissionDemoAction,
		bolsamodule.PermissionAudit,
	}
}

func hasRole(roles []string, role string) bool {
	for _, candidate := range roles {
		if strings.TrimSpace(candidate) == role {
			return true
		}
	}
	return false
}

func hasAnyRole(roles []string, candidates ...string) bool {
	for _, candidate := range candidates {
		if hasRole(roles, candidate) {
			return true
		}
	}
	return false
}

func dedupePermissions(values []string) []string {
	seen := map[string]struct{}{}
	permissions := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		permissions = append(permissions, value)
	}
	return permissions
}
