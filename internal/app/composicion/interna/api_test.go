package interna

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	httpct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/vec/adapters/httpapi"
	"vec-diputacion-granada/internal/vec/adapters/memory"
	vecapp "vec-diputacion-granada/internal/vec/application"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type autoridadRutasCTInternasPrueba struct{ llamadas int }

func (a *autoridadRutasCTInternasPrueba) AutorizarRutaExacta(_ context.Context, _ string) error {
	a.llamadas++
	return nil
}

type auditoriaFronteraCTInternaPrueba struct{}

func (auditoriaFronteraCTInternaPrueba) RegistrarAuditoriaFronteraRutaExacta(context.Context, vecports.OrdenAuditoriaFronteraRutaExacta) error {
	return nil
}

func TestAPIInternaCTSoloComponeDosLecturas(t *testing.T) {
	store := memory.NewStore()
	servicio, err := vecapp.NewService(store, store, store)
	if err != nil {
		t.Fatal(err)
	}
	autoridad := &autoridadRutasCTInternasPrueba{}
	rutas := []httpapi.RutaExacta{
		{Ruta: httpct.RutaConsultaCuadroRRHH, Manejador: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })},
		{Ruta: httpct.RutaConsultaDetalleRRHH, Manejador: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })},
	}
	api, err := nuevaAPIInternaCT(servicio, rutas, autoridad, auditoriaFronteraCTInternaPrueba{})
	if err != nil {
		t.Fatal(err)
	}
	for _, caso := range []struct {
		metodo, ruta string
		estado       int
	}{
		{http.MethodPost, httpct.RutaConsultaCuadroRRHH, http.StatusNoContent},
		{http.MethodPost, httpct.RutaConsultaDetalleRRHH, http.StatusNoContent},
		{http.MethodGet, httpct.RutaConsultaCuadroRRHH, http.StatusNotFound},
		{http.MethodGet, "/api/vec/cronos/timecards", http.StatusNotFound},
		{http.MethodGet, "/api/vec/personal/categories", http.StatusNotFound},
		{http.MethodGet, "/api/vec/audit", http.StatusNotFound},
	} {
		rec := httptest.NewRecorder()
		api.ServeHTTP(rec, httptest.NewRequest(caso.metodo, caso.ruta, nil))
		if rec.Code != caso.estado {
			t.Fatalf("%s %s = %d", caso.metodo, caso.ruta, rec.Code)
		}
	}
	if autoridad.llamadas != 2 {
		t.Fatalf("autoridad llamada en rutas ajenas: %d", autoridad.llamadas)
	}
	if api, err := nuevaAPIInternaCT(servicio, rutas[:1], autoridad, auditoriaFronteraCTInternaPrueba{}); api != nil || err == nil {
		t.Fatal("se aceptó un montaje incompleto")
	}
}
