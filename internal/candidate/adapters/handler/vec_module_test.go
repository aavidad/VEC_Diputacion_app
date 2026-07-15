package handler

import (
	"net/http"
	"testing"

	"vec-diputacion-granada/internal/candidate/ports"
	bolsamodule "vec-diputacion-granada/internal/modules/bolsa"
)

func TestHTTPHandlerVECManifestMethodAuthHappyPathAndErrors(t *testing.T) {
	handler := mustTestHandlerWithDemo(t, &recordingService{}, nil, &recordingAuthenticator{
		principal: ports.AuthPrincipal{Subject: "staff", Role: ports.AuthRolePersonalInterno, Mechanism: ports.AuthMechanismKerberosAD},
	})

	response := performStaffJSON(t, handler, http.MethodGet, "/api/modules/bolsa", "")
	assertStatus(t, response, http.StatusOK)
	var manifest bolsamodule.ModuleManifestContract
	decodeData(t, response, &manifest)
	if manifest.ModuleRef != bolsamodule.ModuleID || manifest.BaseRoute != "/modules/bolsa" {
		t.Fatalf("manifest = %+v", manifest)
	}

	method := performStaffJSON(t, handler, http.MethodPost, "/api/modules/bolsa", "")
	assertStatus(t, method, http.StatusMethodNotAllowed)
	candidate := mustTestHandler(t, &recordingService{}, &recordingAuthenticator{
		principal: ports.AuthPrincipal{Subject: "candidate", Role: ports.AuthRoleCiudadano, Mechanism: ports.AuthMechanismClave},
	})
	assertStatus(t, performJSON(t, candidate, http.MethodGet, "/api/modules/bolsa", ""), http.StatusForbidden)
}

func TestHTTPHandlerAdminStatusCapabilitiesMethodAuthHappyPathAndErrors(t *testing.T) {
	handler := mustTestHandlerWithDemo(t, &recordingService{}, &recordingDemoRunner{}, &recordingAuthenticator{
		principal: ports.AuthPrincipal{Subject: "technical-admin", Role: ports.AuthRoleSystemAdmin, Mechanism: ports.AuthMechanismKerberosAD},
	})

	statusResponse := performStaffJSON(t, handler, http.MethodGet, "/api/admin/status", "")
	assertStatus(t, statusResponse, http.StatusOK)
	var status bolsamodule.OperationalStatus
	decodeData(t, statusResponse, &status)
	if !status.DemoEnabled || status.LegalProductionReady {
		t.Fatalf("status = %+v", status)
	}

	capabilitiesResponse := performStaffJSON(t, handler, http.MethodGet, "/api/admin/capabilities", "")
	assertStatus(t, capabilitiesResponse, http.StatusOK)
	var capabilities bolsamodule.AdminCapabilities
	decodeData(t, capabilitiesResponse, &capabilities)
	if capabilities.ModuleRef != "vec.module.bolsa" || len(capabilities.HTTPRoutes) != 2 {
		t.Fatalf("capabilities = %+v", capabilities)
	}

	assertStatus(t, performStaffJSON(t, handler, http.MethodPost, "/api/admin/status", ""), http.StatusMethodNotAllowed)
	candidate := mustTestHandler(t, &recordingService{}, &recordingAuthenticator{
		principal: ports.AuthPrincipal{Subject: "candidate", Role: ports.AuthRoleCiudadano, Mechanism: ports.AuthMechanismClave},
	})
	assertStatus(t, performJSON(t, candidate, http.MethodGet, "/api/admin/status", ""), http.StatusForbidden)
	staff := mustTestHandler(t, &recordingService{}, &recordingAuthenticator{
		principal: ports.AuthPrincipal{Subject: "staff", Role: ports.AuthRolePersonalInterno, Mechanism: ports.AuthMechanismKerberosAD},
	})
	assertStatus(t, performStaffJSON(t, staff, http.MethodGet, "/api/admin/status", ""), http.StatusForbidden)
}
