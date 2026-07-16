package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	adminmodule "vec-diputacion-granada/internal/modules/administracion"
	bolsamodule "vec-diputacion-granada/internal/modules/bolsa"
	cronosmodule "vec-diputacion-granada/internal/modules/cronos"
	dietasmodule "vec-diputacion-granada/internal/modules/dietas"
	personalmodule "vec-diputacion-granada/internal/modules/personal"
	"vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/domain"
)

type Handler struct {
	service                 *application.Service
	internal                *application.InternalOperations
	personalCatalog         CatalogoPersonal
	categoriasProfesionales ConsultaCategoriasProfesionales
	roadRoute               *roadRouteConnector
	identityPolicy          identityPolicy
}

type HandlerOptions struct {
	InternalOperations      *application.InternalOperations
	PersonalCatalog         CatalogoPersonal
	CategoriasProfesionales ConsultaCategoriasProfesionales
	OSRMBaseURL             string
	OSRMScopeName           string
	OSRMScopeBounds         string
	OSRMAllowedCIDRs        []string
	AllowDemoIdentity       bool
	DemoIdentityResolver    DemoIdentityResolver
	TrustIdentityHeaders    bool
	TrustedProxyCIDRs       []string
	IdentitySubjectHeader   string
	IdentityRolesHeader     string
	IdentityMechanismHeader string
}

// DemoIdentityResolver es el unico origen admitido para el modo fake. La
// implementacion productiva de composicion resuelve un Bearer opaco contra un
// fichero local y no consume identidad, roles ni garantia desde cabeceras.
type DemoIdentityResolver interface {
	ResolveDemoIdentity(context.Context, *http.Request) (domain.Principal, error)
}

func NewHandler(service *application.Service) (*Handler, error) {
	return NewHandlerWithOptions(service, HandlerOptions{})
}

func NewHandlerWithOptions(service *application.Service, options HandlerOptions) (*Handler, error) {
	if service == nil {
		return nil, errors.New("vec http handler: service required")
	}
	if options.InternalOperations != nil && !options.InternalOperations.Matches(service) {
		return nil, application.ErrInternalOperationsMismatch
	}
	identityPolicy, err := newIdentityPolicy(options)
	if err != nil {
		return nil, err
	}
	roadRoute, err := newRoadRouteConnector(
		options.OSRMBaseURL,
		options.OSRMScopeName,
		options.OSRMScopeBounds,
		options.OSRMAllowedCIDRs,
	)
	if err != nil {
		return nil, err
	}
	return &Handler{
		service:                 service,
		internal:                options.InternalOperations,
		personalCatalog:         options.PersonalCatalog,
		categoriasProfesionales: options.CategoriasProfesionales,
		roadRoute:               roadRoute,
		identityPolicy:          identityPolicy,
	}, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := vecPath(r.URL.Path)
	principal, err := principalFromRequest(r, h.identityPolicy)
	if err != nil || principal.Validate() != nil {
		h.writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	switch {
	case path == "/":
		if !h.requireMethod(w, r, http.MethodGet) {
			return
		}
		if !principal.HasPermission("vec.modules.read") {
			h.writeError(w, http.StatusForbidden, domain.ErrPermissionDenied.Error())
			return
		}
		h.writeJSON(w, http.StatusOK, map[string]any{
			"routes": vecRoutes(),
		})
	case path == "/session":
		if !h.requireMethod(w, r, http.MethodGet) {
			return
		}
		if !principal.HasPermission("vec.session.read") {
			h.writeError(w, http.StatusForbidden, domain.ErrPermissionDenied.Error())
			return
		}
		h.writeJSON(w, http.StatusOK, map[string]any{"principal": principal})
	case path == "/modules":
		h.handleModules(w, r, principal)
	case path == "/workspace":
		h.handleWorkspace(w, r, principal)
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
	case path == "/dietas/road-route":
		h.handleDietasRoadRoute(w, r, principal)
	case path == "/menu":
		h.handleMenu(w, r, principal)
	case path == "/audit":
		h.handleAudit(w, r, principal)
	case strings.HasPrefix(path, "/modules/") && strings.HasSuffix(path, "/action"):
		h.handleModuleAction(w, r, principal, path)
	default:
		h.writeError(w, http.StatusNotFound, "vec route not found")
	}
}

func (h *Handler) handleModules(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	if !h.requireMethod(w, r, http.MethodGet) {
		return
	}
	if !principal.HasPermission("vec.modules.read") {
		h.writeError(w, http.StatusForbidden, domain.ErrPermissionDenied.Error())
		return
	}
	modules, err := h.service.Modules(r.Context(), principal)
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
	if !principal.HasPermission("vec.menu.read") {
		h.writeError(w, http.StatusForbidden, domain.ErrPermissionDenied.Error())
		return
	}
	menu, err := h.service.BuildMenu(r.Context(), principal)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"menu": menu, "principal": principal})
}

func (h *Handler) handleAudit(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	w.Header().Set("Cache-Control", "no-store")
	if !h.requireMethod(w, r, http.MethodGet) {
		return
	}
	if !principal.HasPermission(adminmodule.PermissionAuditRead) {
		h.writeError(w, http.StatusForbidden, domain.ErrPermissionDenied.Error())
		return
	}
	if h.internal == nil {
		h.writeError(w, http.StatusForbidden, domain.ErrPermissionDenied.Error())
		return
	}
	subjectRef, err := referenciaAuditoriaDesdeConsulta(r.URL.RawQuery)
	if err != nil {
		h.writeError(w, http.StatusForbidden, domain.ErrPermissionDenied.Error())
		return
	}
	query, err := application.NewAuditQuery(principal, subjectRef)
	if err != nil {
		h.writeError(w, http.StatusForbidden, domain.ErrPermissionDenied.Error())
		return
	}
	audit, err := h.internal.Audit(r.Context(), query)
	if err != nil {
		if errors.Is(err, domain.ErrPermissionDenied) {
			h.writeError(w, http.StatusForbidden, domain.ErrPermissionDenied.Error())
			return
		}
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
	if h.internal == nil {
		h.writeError(w, http.StatusForbidden, domain.ErrPermissionDenied.Error())
		return
	}
	authorized, err := application.NewAuthorizedAuditCommand(application.AuditCommand{
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
	}, action.permission, action.eventType)
	if err != nil {
		h.writeError(w, http.StatusForbidden, domain.ErrPermissionDenied.Error())
		return
	}
	receipt, err := h.internal.RecordAudit(r.Context(), authorized)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	audit := receipt.Entry()
	if err := h.internal.PublishEvent(r.Context(), receipt, domain.Event{
		Type:       action.eventType,
		ModuleID:   action.moduleID,
		SubjectRef: audit.SubjectRef,
		ActorID:    principal.ID,
		OccurredAt: time.Now().UTC(),
		Payload:    map[string]string{"audit_id": audit.ID},
	}); err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}
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
		"administracion": {
			key:        "administracion",
			moduleID:   adminmodule.ModuleID,
			permission: adminmodule.PermissionCatalogsManage,
			action:     adminmodule.ActionPublishCatalog,
			subjectRef: "admin-catalogos-demo",
			eventType:  "vec.module.administracion.catalog.executed",
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
		"/api/vec/dietas/road-route",
		"/api/vec/modules/cronos/action",
		"/api/vec/modules/horarios/action",
		"/api/vec/modules/permisos/action",
		"/api/vec/modules/dietas/action",
		"/api/vec/modules/rutas/action",
		"/api/vec/modules/bolsa/action",
		"/api/vec/modules/administracion/action",
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

func principalFromRequest(r *http.Request, policy identityPolicy) (domain.Principal, error) {
	if policy.allowDemo {
		if policy.demoResolver == nil || r == nil {
			return domain.Principal{}, domain.ErrPrincipalInvalid
		}
		principal, err := policy.demoResolver.ResolveDemoIdentity(r.Context(), r)
		if err != nil {
			return domain.Principal{}, err
		}
		// El fichero declara identidad y roles. Los permisos son una lista
		// positiva propia de esta carcasa y nunca se aceptan del resolvedor.
		principal.Permissions = permissionsForRoles(principal.Roles)
		if err := principal.Validate(); err != nil {
			return domain.Principal{}, err
		}
		return principal, nil
	}
	identity := identityFromRequest(r, policy)
	// En una superficie real los roles de la asercion son informativos: el
	// autorizador RBAC+ABAC debe resolver cada caso de uso y esta frontera no
	// concede permisos por su cuenta.
	principal := domain.Principal{
		ID:            identity.subject,
		DisplayName:   identity.displayName,
		Email:         identity.email,
		Roles:         identity.roles,
		AuthMethod:    identity.method,
		AuthAssurance: identity.assurance,
		Permissions:   nil,
		Attributes:    identity.attributes,
	}
	return principal, nil
}

func authMethod(value string) domain.AuthMethod {
	// El metodo participa en la garantia de autenticacion. Solo la
	// representacion canonica exacta puede llegar al dominio; una variante no se
	// transforma en una afirmacion mas fuerte.
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
		return ""
	}
}

func permissionsForRoles(roles []string) []string {
	// El modo fake solo admite un perfil canonico por sesion. Una mezcla, un
	// alias adicional o cualquier valor no exacto es ambiguedad y, por tanto,
	// no concede capacidades.
	if len(roles) != 1 {
		return nil
	}

	switch roles[0] {
	case "ciudadano", "candidate":
		return permisosCiudadanoDemo()
	case "administrador", "system_admin":
		return permisosAdministradorTecnicoDemo()
	case "jefatura_rrhh":
		return permisosJefaturaRRHHDemo()
	case "tecnico_rrhh", "validator_l2":
		return permisosTecnicoRRHHDemo()
	case "administrativo", "validator_l1":
		return permisosAdministrativoRRHHDemo()
	case "personal_interno":
		return permisosPersonalInternoDemo()
	case "jefe_servicio", "jefe_seccion":
		return permisosJefaturaOperativaDemo()
	default:
		return nil
	}
}

func permisosCiudadanoDemo() []string {
	return []string{
		"vec.session.read",
		"vec.menu.read",
		bolsamodule.PermissionRead,
		bolsamodule.PermissionDocument,
		bolsamodule.PermissionClaim,
		bolsamodule.PermissionNotification,
	}
}

// permisosAdministradorTecnicoDemo no incluye datos ni acciones funcionales.
// La auditoria generica tampoco se concede: hasta separar sus campos tecnicos
// puede revelar referencias de expedientes o actuaciones de otros modulos.
func permisosAdministradorTecnicoDemo() []string {
	return append(permisosCarcasaInternaDemo(),
		adminmodule.PermissionRolesManage,
		adminmodule.PermissionCatalogsManage,
		adminmodule.PermissionIntegrationsManage,
		adminmodule.PermissionMonitoringRead,
	)
}

func permisosJefaturaRRHHDemo() []string {
	return permisosCarcasaInternaDemo()
}

func permisosTecnicoRRHHDemo() []string {
	return permisosCarcasaInternaDemo()
}

func permisosAdministrativoRRHHDemo() []string {
	return permisosCarcasaInternaDemo()
}

func permisosPersonalInternoDemo() []string {
	return permisosCarcasaInternaDemo()
}

func permisosJefaturaOperativaDemo() []string {
	return permisosCarcasaInternaDemo()
}

func permisosCarcasaInternaDemo() []string {
	return []string{
		"vec.session.read",
		"vec.modules.read",
		"vec.menu.read",
	}
}
