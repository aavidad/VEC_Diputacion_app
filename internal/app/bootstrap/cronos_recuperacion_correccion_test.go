package bootstrap

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	cronoshttp "vec-diputacion-granada/internal/modules/cronos/adapters/httpinterno"
	cronosapp "vec-diputacion-granada/internal/modules/cronos/application"
	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
)

func TestRecuperacionCorreccionCronosAisladaDeSeleccionBase(t *testing.T) {
	base, _ := generarMaterialDesarrolloPrueba(t)
	base.CronosEmpleadoEnabled = "true"
	// El valor no canónico y la ausencia de fichero privado no desactivan
	// la selección del material ya validado de Cronos.
	base.CronosRecuperacionCorreccionEnabled = "si"
	seleccion, err := seleccionMaterialCTDesarrolloDesdeConfig(base)
	if err != nil || !seleccion.cronos || seleccion.cronosRecuperacionCorreccion {
		t.Fatalf("selector C5a inválido afectó a Cronos base: %+v, %v", seleccion, err)
	}
	base.CronosRecuperacionCorreccionEnabled = "true"
	seleccion, err = seleccionMaterialCTDesarrolloDesdeConfig(base)
	if err != nil || !seleccion.cronos || !seleccion.cronosRecuperacionCorreccion {
		t.Fatalf("selector C5a no seleccionó las dos audiencias: %+v, %v", seleccion, err)
	}
	descriptores := descriptoresMaterialRecuperacionCorreccionCronosDesarrollo()
	if descriptores[0].Audiencia != cronosapp.AudienciaRecuperarReciboCorreccionPropia ||
		descriptores[1].Audiencia != personaldomain.AudienciaVinculoPropioCRN11 {
		t.Fatal("lectura de recibo y vínculo no usan audiencias nominales separadas")
	}
}

func TestRecuperacionCorreccionCronosSinDependenciaCierraSoloGET(t *testing.T) {
	if err := preflightRecuperacionCorreccionCronos(context.Background(), nil, nil); err == nil {
		t.Fatal("preflight permitió fuentes ausentes")
	}
	r := httptest.NewRequest(http.MethodGet, cronoshttp.RutaRecuperarReciboCorreccionPropia, nil)
	w := httptest.NewRecorder()
	reciboCorreccionNoDisponible().ServeHTTP(w, r)
	if w.Code != http.StatusServiceUnavailable || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("GET opcional no cerró con 503: %d", w.Code)
	}
	contrato, ok := contratoDenegacionVinculoCronos(cronoshttp.RutaRecuperarReciboCorreccionPropia)
	if !ok || contrato.accion != cronosapp.AccionRecuperarReciboCorreccion ||
		contrato.finalidad != cronosapp.FinalidadRecuperarReciboCorreccionPropia {
		t.Fatal("denegación del GET sin contrato de auditoría nominal")
	}
}
