package bootstrap

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
)

func TestPreparacionVigenteCoberturaExigeCanalYContextoV3Propios(t *testing.T) {
	soporte, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	ruta := httpinterno.RutaPreparacionCoberturaVigente
	if !rutaCoberturaContratacionTemporalDesarrollo(ruta) ||
		!rutaContextoAutorizacionContratacionTemporalDesarrollo(ruta) ||
		!esRutaContratacionTemporalDesarrollo(httptest.NewRequest(http.MethodGet, ruta, nil)) {
		t.Fatal("el GET no pasa por la misma frontera de canal, contexto y mTLS")
	}
	ctx := contextoRutaCoberturaDesarrolloPrueba(soporte, principal, ruta)
	canal, err := soporte.ResolverContextoCanalCobertura(ctx)
	if err != nil || canal.OrganizacionRef != organizacionAltaContratacionTemporalDesarrollo {
		t.Fatalf("el canal GET no resolvió contexto confiable: %v", err)
	}
	solicitud := solicitudContextoPreparacionVigentePrueba(t, soporte)
	if _, err := soporte.ResolverContextoAutorizacionAltaV3(ctx, solicitud); err != nil {
		t.Fatalf("el contexto V3 GET no se resolvió: %v", err)
	}
	if _, err := soporte.ResolverContextoCanalCobertura(
		contextoRutaCoberturaDesarrolloPrueba(soporte, principal, httpinterno.RutaResultadoCobertura),
	); err == nil {
		t.Fatal("la ruta de resultado recibió autoridad de canal GET")
	}
}
