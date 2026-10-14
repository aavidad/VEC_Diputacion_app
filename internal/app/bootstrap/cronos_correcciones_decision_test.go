package bootstrap

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	cronoshttp "vec-diputacion-granada/internal/modules/cronos/adapters/httpinterno"
	cronosapp "vec-diputacion-granada/internal/modules/cronos/application"
)

func TestCronosCorreccionesDecisionPublicaSeisAudienciasNominales(t *testing.T) {
	d := descriptoresMaterialDecisionCorreccionesCronos()
	esperadas := [6]string{
		cronosapp.AudienciaBandejaCorrecciones,
		cronosapp.AudienciaDecisionResponsableCorreccion,
		cronosapp.AudienciaResolucionRRHHCorreccion,
		cronosapp.AudienciaAplicacionCorreccion,
		cronosapp.AudienciaHistoriaCorreccionPropia,
		cronosapp.AudienciaHistoriaCorreccionEquipo,
	}
	vistas := map[string]bool{}
	for i, descriptor := range d {
		if descriptor.Audiencia != esperadas[i] || descriptor.ProveedorNominal == "" || vistas[descriptor.Audiencia] {
			t.Fatalf("descriptor nominal C5b %d incorrecto", i)
		}
		vistas[descriptor.Audiencia] = true
	}
	if _, err := publicarMaterialDecisionCorreccionesCronosEnLote(context.Background(), nil,
		materialAtestacionContratacionTemporalDesarrollo{}, relojContratacionTemporalDesarrollo{}); err == nil {
		t.Fatal("se publicaron audiencias sin gobierno V3")
	}
}

func TestCronosCorreccionesDecisionCierraSusTresRutas(t *testing.T) {
	rutas := map[string]http.Handler{}
	rutasDecisionCorreccionesNoDisponibles(rutas)
	for _, ruta := range []string{cronoshttp.RutaBandejaCorrecciones, cronoshttp.RutaActuacionCorrecciones, cronoshttp.RutaHistoriaCorrecciones} {
		h := rutas[ruta]
		if h == nil {
			t.Fatalf("ruta %s ausente", ruta)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, ruta, nil))
		if w.Code != http.StatusServiceUnavailable || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("ruta %s no cerró con 503", ruta)
		}
	}
	if preflightDecisionCorreccionesCronos(context.Background(), nil) == nil {
		t.Fatal("preflight aceptó ejecutor ausente")
	}
}
