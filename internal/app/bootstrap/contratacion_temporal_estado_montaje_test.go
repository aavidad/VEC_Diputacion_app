package bootstrap

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
)

type consultorSeleccionAlternativoPrueba struct {
	httpinterno.EjecutorSeleccionLlamamiento
	httpinterno.ConsultorEstadoSeleccionOferta
}

func TestConsultaEstadoSeleccionOfertaSoloSeMontaConServicio(t *testing.T) {
	if consultorEstadoSeleccionOfertaMontadoCT(nil) != nil ||
		consultorEstadoSeleccionOfertaMontadoCT(&ejecutorSeleccionLlamamientoDesarrollo{}) != nil {
		t.Fatal("la ruta de estado se publicó sin servicio")
	}
	real := &ejecutorSeleccionLlamamientoDesarrollo{
		estadoOfertas: &application.ServicioConsultaEstadoSeleccionOfertaCT{},
	}
	if consultorEstadoSeleccionOfertaMontadoCT(real) != real {
		t.Fatal("el servicio compuesto no se publicó")
	}
	alternativo := &consultorSeleccionAlternativoPrueba{}
	if consultorEstadoSeleccionOfertaMontadoCT(alternativo) != alternativo {
		t.Fatal("un consultor alternativo válido dejó de montarse")
	}
	h, err := httpinterno.NuevoManejadorConsultaEstadoSeleccionOferta(
		consultorEstadoSeleccionOfertaMontadoCT(real))
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet,
		httpinterno.RutaConsultaEstadoSeleccionOferta+"?expediente_ref="+
			url.QueryEscape("expediente:ct:prueba"), nil))
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("servicio montado fallido debe conservar 503: %d", w.Code)
	}
}
