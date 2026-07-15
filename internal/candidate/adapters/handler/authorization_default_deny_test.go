package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/candidate/ports"
)

func TestHTTPHandlerDoesNotChooseAuthorityFromAmbiguousRoles(t *testing.T) {
	principals := []ports.AuthPrincipal{
		{Subject: "cand-1", Role: ports.AuthRoleCiudadano, Roles: []ports.AuthRole{ports.AuthRoleCiudadano, ports.AuthRoleValidatorL1}, Mechanism: ports.AuthMechanismClave},
		{Subject: "cand-1", Role: ports.AuthRoleCiudadano, Roles: []ports.AuthRole{ports.AuthRoleCiudadano, ports.AuthRole("rol_desconocido")}, Mechanism: ports.AuthMechanismClave},
		{Subject: "cand-1", Role: ports.AuthRole("rol_desconocido"), Roles: []ports.AuthRole{ports.AuthRoleCiudadano}, Mechanism: ports.AuthMechanismClave},
	}
	for _, principal := range principals {
		service := &recordingService{}
		handler := mustTestHandler(t, service, &recordingAuthenticator{principal: principal})
		response := performJSON(t, handler, http.MethodPost, "/api/candidates",
			`{"id":"cand-1","dni":"12345678A","nombre":"Ana","email":"ana@example.test","call_id":"call-1"}`)
		assertStatus(t, response, http.StatusUnauthorized)
		if service.createCalls != 0 {
			t.Fatalf("un conjunto ambiguo de roles alcanzo el caso de uso: %d", service.createCalls)
		}
	}
}

func TestHTTPHandlerRejectsNonCanonicalOrAmbiguousCredentialsBeforeAuthentication(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*http.Request)
	}{
		{name: "missing subject", mutate: func(request *http.Request) { request.Header.Del("X-Auth-Subject") }},
		{name: "non canonical subject", mutate: func(request *http.Request) { request.Header.Set("X-Auth-Subject", " cand-1") }},
		{name: "two distinct subjects", mutate: func(request *http.Request) { request.Header.Add("X-Auth-Subject", "cand-2") }},
		{name: "exact subject repetition", mutate: func(request *http.Request) { request.Header.Add("X-Auth-Subject", "cand-1") }},
		{name: "missing mechanism", mutate: func(request *http.Request) { request.Header.Del("X-Auth-Mechanism") }},
		{name: "non canonical mechanism", mutate: func(request *http.Request) { request.Header.Set("X-Auth-Mechanism", "clave ") }},
		{name: "two distinct mechanisms", mutate: func(request *http.Request) { request.Header.Add("X-Auth-Mechanism", string(ports.AuthMechanismDNIe)) }},
		{name: "exact mechanism repetition", mutate: func(request *http.Request) { request.Header.Add("X-Auth-Mechanism", string(ports.AuthMechanismClave)) }},
		{name: "missing token", mutate: func(request *http.Request) { request.Header.Del("Authorization") }},
		{name: "non canonical bearer token", mutate: func(request *http.Request) { request.Header.Set("Authorization", "Bearer  citizen-token") }},
		{name: "wrong bearer scheme", mutate: func(request *http.Request) { request.Header.Set("Authorization", "Basic citizen-token") }},
		{name: "two bearer values", mutate: func(request *http.Request) { request.Header.Add("Authorization", "Bearer citizen-token") }},
		{name: "two token representations", mutate: func(request *http.Request) { request.Header.Set("X-Auth-Token", "citizen-token") }},
		{name: "combined direct tokens", mutate: func(request *http.Request) {
			request.Header.Del("Authorization")
			request.Header.Set("X-Auth-Token", "token-1,token-2")
		}},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			service := &recordingService{}
			authenticator := &recordingAuthenticator{principal: ports.AuthPrincipal{
				Subject: "cand-1", Role: ports.AuthRoleCandidate, Mechanism: ports.AuthMechanismClave,
			}}
			handler := mustTestHandler(t, service, authenticator)
			request := httptest.NewRequest(http.MethodPost, "/candidates", strings.NewReader(
				`{"id":"cand-1","dni":"12345678A","nombre":"Ana","email":"ana@example.test","call_id":"call-1"}`,
			))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("X-Auth-Mechanism", string(ports.AuthMechanismClave))
			request.Header.Set("X-Auth-Subject", "cand-1")
			request.Header.Set("Authorization", "Bearer citizen-token")
			tt.mutate(request)

			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
			}
			if authenticator.calls != 0 || service.createCalls != 0 {
				t.Fatalf("invalid credentials reached authority: auth=%d usecase=%d", authenticator.calls, service.createCalls)
			}
		})
	}
}

func TestHTTPHandlerRejectsMalformedPrincipalReturnedByAuthenticator(t *testing.T) {
	principals := []ports.AuthPrincipal{
		{Subject: "cand-1", Mechanism: ports.AuthMechanismClave},
		{Subject: "cand-1", Role: ports.AuthRoleCandidate, Roles: []ports.AuthRole{"unknown"}, Mechanism: ports.AuthMechanismClave},
		{Subject: "cand-1", Role: ports.AuthRoleCandidate, Roles: []ports.AuthRole{ports.AuthRoleValidatorL1}, Mechanism: ports.AuthMechanismClave},
		{Subject: "cand-1", Role: ports.AuthRoleCandidate},
		{Subject: "cand-1", Role: ports.AuthRoleCandidate, Mechanism: ports.AuthMechanismClave, Method: ports.AuthMechanismDNIe},
		{Subject: "cand-1", Role: ports.AuthRoleCandidate, Mechanism: ports.AuthMechanismClave, Method: "password"},
		{Subject: " cand-1", Role: ports.AuthRoleCandidate, Mechanism: ports.AuthMechanismClave},
	}
	for _, principal := range principals {
		service := &recordingService{}
		handler := mustTestHandler(t, service, &recordingAuthenticator{principal: principal})
		response := performJSON(t, handler, http.MethodPost, "/candidates",
			`{"id":"cand-1","dni":"12345678A","nombre":"Ana","email":"ana@example.test","call_id":"call-1"}`)
		assertStatus(t, response, http.StatusUnauthorized)
		if service.createCalls != 0 {
			t.Fatalf("malformed principal reached use case: %+v", principal)
		}
	}
}

func TestHTTPHandlerAcceptsExactDuplicatePrincipalRepresentationWithoutExtraAuthority(t *testing.T) {
	service := &recordingService{createView: CandidateView{ID: "cand-1", CallID: "call-1"}}
	handler := mustTestHandler(t, service, &recordingAuthenticator{principal: ports.AuthPrincipal{
		Subject:   "cand-1",
		Role:      ports.AuthRoleCandidate,
		Roles:     []ports.AuthRole{ports.AuthRoleCandidate, ports.AuthRoleCandidate},
		Mechanism: ports.AuthMechanismClave,
		Method:    ports.AuthMechanismClave,
	}})
	response := performJSON(t, handler, http.MethodPost, "/candidates",
		`{"id":"cand-1","dni":"12345678A","nombre":"Ana","email":"ana@example.test","call_id":"call-1"}`)
	assertStatus(t, response, http.StatusCreated)
	if service.createCalls != 1 {
		t.Fatalf("exact duplicate representation calls = %d, want 1", service.createCalls)
	}
}

func TestAPIRootDoesNotAdvertiseCrossProfileRoutes(t *testing.T) {
	for _, route := range apiRootRoutes(ports.AuthRoleSystemAdmin) {
		if strings.Contains(route, "/candidates") || route == "/api/demo" || route == "/api/portal" ||
			strings.Contains(route, "/notifications") || strings.Contains(route, "/audit") {
			t.Fatalf("administracion tecnica anuncio una ruta funcional: %s", route)
		}
	}
	for _, route := range apiRootRoutes(ports.AuthRoleCiudadano) {
		if strings.Contains(route, "/admin/") || strings.Contains(route, "/modules/") || route == "/api/portal" {
			t.Fatalf("el perfil ciudadano anuncio una ruta interna: %s", route)
		}
	}
	if routes := apiRootRoutes(ports.AuthRole("desconocido")); len(routes) != 0 {
		t.Fatalf("un perfil desconocido anuncio rutas: %v", routes)
	}
}

func TestHTTPHandlerTechnicalAdminDoesNotInheritFunctionalActions(t *testing.T) {
	handler := mustTestHandlerWithDemo(t, &recordingService{}, &recordingDemoRunner{}, &recordingAuthenticator{
		principal: ports.AuthPrincipal{Subject: "technical-admin", Role: ports.AuthRoleSystemAdmin, Mechanism: ports.AuthMechanismKerberosAD},
	})

	assertStatus(t, performStaffJSON(t, handler, http.MethodPost, "/api/demo", ""), http.StatusForbidden)
	assertStatus(t, performStaffJSON(t, handler, http.MethodGet, "/api/portal", ""), http.StatusForbidden)
	assertStatus(t, performStaffJSON(t, handler, http.MethodGet, "/api/admin/status", ""), http.StatusOK)
}

func TestHTTPHandlerRequiresOneExactAdministrativeScope(t *testing.T) {
	handler := mustTestHandlerWithAdministrativeFlow(t, ports.AuthPrincipal{
		Subject: "staff", Role: ports.AuthRolePersonalInterno, Mechanism: ports.AuthMechanismKerberosAD,
	})
	for _, path := range []string{
		"/api/audit",
		"/api/audit?candidate_id=",
		"/api/audit?candidate_id=*",
		"/api/audit?candidate_id=cand%201",
		"/api/audit?candidate_id=cand-1&scope=candidate:cand-1",
		"/api/audit?candidate_id=cand-1&candidate_id=cand-2",
		"/api/audit?scope=procedure:all",
		"/api/notifications",
		"/api/notifications?candidate_id=*",
		"/api/notifications?candidate_id=cand-1&candidate_id=cand-2",
	} {
		response := performStaffJSON(t, handler, http.MethodGet, path, "")
		assertStatus(t, response, http.StatusBadRequest)
	}

	assertStatus(t, performStaffJSON(t, handler, http.MethodGet, "/api/audit?candidate_id=cand-1", ""), http.StatusOK)
	assertStatus(t, performStaffJSON(t, handler, http.MethodGet, "/api/notifications?candidate_id=cand-1", ""), http.StatusOK)
}

func TestHTTPHandlerRejectsTrailingJSONWithoutCallingUseCase(t *testing.T) {
	service := &recordingService{}
	handler := mustTestHandler(t, service, &recordingAuthenticator{principal: ports.AuthPrincipal{
		Subject: "cand-1", Role: ports.AuthRoleCiudadano, Mechanism: ports.AuthMechanismClave,
	}})

	response := performJSON(t, handler, http.MethodPost, "/api/candidates",
		`{"id":"cand-1","dni":"12345678A","nombre":"Ana","email":"ana@example.test","call_id":"call-1"} {}`)
	assertStatus(t, response, http.StatusBadRequest)
	if service.createCalls != 0 {
		t.Fatalf("JSON adicional alcanzo el caso de uso: %d", service.createCalls)
	}
}
