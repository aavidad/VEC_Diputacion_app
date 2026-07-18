package bolsa

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestEstadoOperativoNormalizaModosYNoDeclaraPreparacionLegal(t *testing.T) {
	t.Parallel()
	estado := OperationalStatusForModes(true, "  custom-auth  ", "\tmemory-db \n")
	if estado.ModuleRef != ModuleID || estado.RuntimeMode != "local_productizable" || estado.Status != "operational" {
		t.Fatalf("estado operativo inesperado: %+v", estado)
	}
	if estado.AuthMode != "custom-auth" || estado.PersistenceMode != "memory-db" || !estado.DemoEnabled {
		t.Fatalf("modos no normalizados: %+v", estado)
	}
	if estado.LegalProductionReady {
		t.Fatal("el prototipo no puede declararse listo juridicamente")
	}
	if !reflect.DeepEqual(estado.AdminRoutes, AdminRoutes()) || !reflect.DeepEqual(estado.LegalIntegrations, LegalIntegrations()) {
		t.Fatalf("contratos del estado divergentes: %+v", estado)
	}
}

func TestEstadoOperativoAplicaValoresPredeterminados(t *testing.T) {
	t.Parallel()
	estado := OperationalStatusForModes(false, "   ", "\n\t")
	if estado.AuthMode != "disabled" || estado.PersistenceMode != "memory" || estado.DemoEnabled || estado.LegalProductionReady {
		t.Fatalf("valores predeterminados inesperados: %+v", estado)
	}
}

func TestManifiestoBolsaConservaRutasEIntegracionesLegales(t *testing.T) {
	t.Parallel()
	manifiesto := ModuleManifestForCandidatePortal()
	if manifiesto.ModuleRef != ModuleID || manifiesto.BaseRoute != "/modules/bolsa" ||
		manifiesto.APIPrefix != "/api/modules/bolsa" || manifiesto.PrototypeAPI != "/api" ||
		manifiesto.AuthorizationPolicySource != "rbac_abac_published" {
		t.Fatalf("manifiesto inesperado: %+v", manifiesto)
	}
	rutasEsperadas := []ModuleHTTPRoute{
		{Method: "GET", Route: "/api/admin/status", Mode: "real"},
		{Method: "GET", Route: "/api/admin/capabilities", Mode: "real"},
	}
	if !reflect.DeepEqual(AdminRoutes(), rutasEsperadas) {
		t.Fatalf("AdminRoutes() = %#v; esperado %#v", AdminRoutes(), rutasEsperadas)
	}
	integracionesEsperadas := []IntegrationStatus{
		{IntegrationRef: "registro_electronico", Status: "not_configured", Mode: "external_legal"},
		{IntegrationRef: "firma_electronica", Status: "not_configured", Mode: "external_legal"},
		{IntegrationRef: "notificacion_feaciente", Status: "not_configured", Mode: "external_legal"},
		{IntegrationRef: "auditoria_probatoria_externa", Status: "not_configured", Mode: "external_legal"},
	}
	if !reflect.DeepEqual(LegalIntegrations(), integracionesEsperadas) {
		t.Fatalf("LegalIntegrations() = %#v; esperado %#v", LegalIntegrations(), integracionesEsperadas)
	}
	if len(manifiesto.HTTPRoutes) == 0 || manifiesto.EventsPublished == nil {
		t.Fatalf("manifiesto incompleto: %+v", manifiesto)
	}
	for _, evento := range manifiesto.EventsPublished {
		if strings.Contains(evento, "autobaremo") {
			t.Fatalf("la autobaremacion aparcada no puede anunciar eventos activos: %q", evento)
		}
	}
}

func TestManifiestoBolsaNoIncrustaRolesComoAutoridad(t *testing.T) {
	t.Parallel()
	datos, err := json.Marshal(ModuleManifestForCandidatePortal())
	if err != nil {
		t.Fatalf("serializar manifiesto: %v", err)
	}
	texto := string(datos)
	if strings.Contains(texto, "required_roles") || strings.Contains(texto, "system_admin") ||
		strings.Contains(texto, "validator_l1") || strings.Contains(texto, "validator_l2") {
		t.Fatalf("el manifiesto incrusta roles como autoridad: %s", texto)
	}
}

func TestContratosBolsaNoCompartenColeccionesMutables(t *testing.T) {
	t.Parallel()
	primero := OperationalStatusDefault(true)
	segundo := OperationalStatusDefault(false)
	if len(primero.AdminRoutes) == 0 || len(primero.LegalIntegrations) == 0 ||
		len(segundo.AdminRoutes) == 0 || len(segundo.LegalIntegrations) == 0 {
		t.Fatal("colecciones operativas vacias")
	}
	primero.AdminRoutes[0].Route = "/alterada"
	primero.LegalIntegrations[0].Status = "alterada"
	tercero := OperationalStatusDefault(true)
	if tercero.AdminRoutes[0].Route == "/alterada" || tercero.LegalIntegrations[0].Status == "alterada" {
		t.Fatal("el estado operativo comparte memoria mutable entre llamadas")
	}

	adminUno := AdminCapabilitiesContract()
	adminDos := AdminCapabilitiesContract()
	adminUno.Capabilities[0].LabelI18nKey = "alterada"
	adminUno.HTTPRoutes[0].Route = "/alterada"
	adminUno.LegalIntegrations[0].Status = "alterada"
	if adminDos.Capabilities[0].LabelI18nKey == "alterada" || adminDos.HTTPRoutes[0].Route == "/alterada" ||
		adminDos.LegalIntegrations[0].Status == "alterada" {
		t.Fatal("el contrato administrativo comparte memoria mutable entre llamadas")
	}
}
