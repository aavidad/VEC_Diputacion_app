package handler

import (
	"net/http"
	"testing"

	"vec-diputacion-granada/internal/candidate/ports"
)

func TestHTTPHandlerReturnsProfessionalPortalForStaff(t *testing.T) {
	handler := mustTestHandler(t, &recordingService{}, &recordingAuthenticator{
		principal: ports.AuthPrincipal{
			Subject:   "staff",
			Role:      ports.AuthRolePersonalInterno,
			Mechanism: ports.AuthMechanismKerberosAD,
		},
	})

	response := performStaffJSON(t, handler, http.MethodGet, "/api/portal", "")
	assertStatus(t, response, http.StatusOK)

	var portal professionalPortalView
	decodeData(t, response, &portal)
	if portal.Principal.Subject != "staff" || portal.Principal.Role != ports.AuthRolePersonalInterno {
		t.Fatalf("principal = %+v", portal.Principal)
	}
	if len(portal.Sections) != 2 || !hasRoute(portal.Routes, "/api/portal") ||
		!hasRoute(portal.Routes, "/api/candidates/{id}/notifications") ||
		!hasRoute(portal.Routes, "/api/candidates/{id}/audit") ||
		!hasRoute(portal.Routes, "/api/audit?candidate_id={id}") {
		t.Fatalf("portal data = %+v", portal)
	}
	for _, forbidden := range []string{"/api/admin/status", "/api/admin/capabilities", "/api/candidates/{id}/claims"} {
		if hasRoute(portal.Routes, forbidden) {
			t.Fatalf("el portal funcional anuncio una ruta no concedida: %s", forbidden)
		}
	}
}

func TestHTTPHandlerListsProfessionalPortalAtAPIRoot(t *testing.T) {
	handler := mustTestHandler(t, &recordingService{}, &recordingAuthenticator{
		principal: ports.AuthPrincipal{
			Subject:   "staff",
			Role:      ports.AuthRolePersonalInterno,
			Mechanism: ports.AuthMechanismKerberosAD,
		},
	})

	response := performStaffJSON(t, handler, http.MethodGet, "/api", "")
	assertStatus(t, response, http.StatusOK)

	var root struct {
		Routes []string `json:"routes"`
	}
	decodeData(t, response, &root)
	if !hasRoute(root.Routes, "/api/portal") ||
		!hasRoute(root.Routes, "/api/candidates/{id}/notifications") ||
		!hasRoute(root.Routes, "/api/candidates/{id}/audit") ||
		!hasRoute(root.Routes, "/api/audit?candidate_id={id}") {
		t.Fatalf("routes = %+v", root.Routes)
	}
	if hasRoute(root.Routes, "/api/admin/status") || hasRoute(root.Routes, "/api/admin/capabilities") {
		t.Fatalf("la raiz funcional anuncio rutas tecnicas: %+v", root.Routes)
	}
}

func TestHTTPHandlerReturnsAdminStatusAndCapabilitiesForTechnicalAdmin(t *testing.T) {
	handler := mustTestHandlerWithDemo(t, &recordingService{}, nil, &recordingAuthenticator{
		principal: ports.AuthPrincipal{Subject: "technical-admin", Role: ports.AuthRoleSystemAdmin, Mechanism: ports.AuthMechanismKerberosAD},
	})

	statusResponse := performStaffJSON(t, handler, http.MethodGet, "/api/admin/status", "")
	assertStatus(t, statusResponse, http.StatusOK)
	var status struct {
		RuntimeMode          string `json:"runtime_mode"`
		DemoEnabled          bool   `json:"demo_enabled"`
		LegalProductionReady bool   `json:"legal_production_ready"`
		LegalIntegrations    []struct {
			Status string `json:"status"`
		} `json:"legal_integrations"`
	}
	decodeData(t, statusResponse, &status)
	if status.RuntimeMode != "local_productizable" || status.DemoEnabled || status.LegalProductionReady {
		t.Fatalf("status = %+v", status)
	}
	if len(status.LegalIntegrations) == 0 || status.LegalIntegrations[0].Status != "not_configured" {
		t.Fatalf("legal integrations = %+v", status.LegalIntegrations)
	}

	capabilitiesResponse := performStaffJSON(t, handler, http.MethodGet, "/api/admin/capabilities", "")
	assertStatus(t, capabilitiesResponse, http.StatusOK)
	var capabilities struct {
		HTTPRoutes []struct {
			Route string `json:"route"`
		} `json:"http_routes"`
	}
	decodeData(t, capabilitiesResponse, &capabilities)
	if !hasAdminRoute(capabilities.HTTPRoutes, "/api/admin/status") ||
		!hasAdminRoute(capabilities.HTTPRoutes, "/api/admin/capabilities") {
		t.Fatalf("capabilities = %+v", capabilities)
	}
}

func TestHTTPHandlerProtectsAdminEndpoints(t *testing.T) {
	candidate := mustTestHandler(t, &recordingService{}, &recordingAuthenticator{
		principal: ports.AuthPrincipal{Subject: "candidate", Role: ports.AuthRoleCiudadano, Mechanism: ports.AuthMechanismClave},
	})
	assertStatus(t, performJSON(t, candidate, http.MethodGet, "/api/admin/status", ""), http.StatusForbidden)

	staff := mustTestHandler(t, &recordingService{}, &recordingAuthenticator{
		principal: ports.AuthPrincipal{Subject: "staff", Role: ports.AuthRolePersonalInterno, Mechanism: ports.AuthMechanismKerberosAD},
	})
	assertStatus(t, performStaffJSON(t, staff, http.MethodGet, "/api/admin/status", ""), http.StatusForbidden)
	assertStatus(t, performStaffJSON(t, staff, http.MethodPost, "/api/admin/status", ""), http.StatusForbidden)
}

func TestHTTPHandlerRejectsCandidateOnProfessionalPortal(t *testing.T) {
	handler := mustTestHandler(t, &recordingService{}, &recordingAuthenticator{
		principal: ports.AuthPrincipal{
			Subject:   "candidate",
			Role:      ports.AuthRoleCiudadano,
			Mechanism: ports.AuthMechanismClave,
		},
	})

	response := performJSON(t, handler, http.MethodGet, "/api/portal", "")
	assertStatus(t, response, http.StatusForbidden)
}

func TestHTTPHandlerRejectsProfessionalPortalMethodMismatch(t *testing.T) {
	handler := mustTestHandler(t, &recordingService{}, &recordingAuthenticator{
		principal: ports.AuthPrincipal{
			Subject:   "staff",
			Role:      ports.AuthRolePersonalInterno,
			Mechanism: ports.AuthMechanismKerberosAD,
		},
	})

	response := performStaffJSON(t, handler, http.MethodPost, "/api/portal", "")
	assertStatus(t, response, http.StatusMethodNotAllowed)
}

func hasRoute(routes []string, want string) bool {
	for _, route := range routes {
		if route == want {
			return true
		}
	}
	return false
}

func hasAdminRoute(routes []struct {
	Route string `json:"route"`
}, want string) bool {
	for _, route := range routes {
		if route.Route == want {
			return true
		}
	}
	return false
}
