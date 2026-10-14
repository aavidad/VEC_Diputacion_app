package httpinterno

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/cronos/ports"
)

type resolverInformeC12Prueba struct{ llamadas *int }

func (r resolverInformeC12Prueba) ResolverInformeC12(*http.Request, string) (ports.OrdenInformeC12, error) {
	*r.llamadas++
	return ports.OrdenInformeC12{}, ports.ErrDependenciaNoDisponible
}

type casoInformeC12Prueba struct{}

func (casoInformeC12Prueba) ExportarInforme(context.Context, ports.OrdenInformeC12, ports.PeticionInformeC12) (ports.ResultadoInformeC12, error) {
	return ports.ResultadoInformeC12{}, ports.ErrDependenciaNoDisponible
}

type resolverInformeIdiomaPrueba struct{}

func (resolverInformeIdiomaPrueba) ResolverInformeC12(*http.Request, string) (ports.OrdenInformeC12, error) {
	return ports.OrdenInformeC12{}, nil
}

type casoInformeIdiomaPrueba struct{ idioma *string }

func (c casoInformeIdiomaPrueba) ExportarInforme(_ context.Context, _ ports.OrdenInformeC12, p ports.PeticionInformeC12) (ports.ResultadoInformeC12, error) {
	*c.idioma = p.Idioma
	return ports.ResultadoInformeC12{}, ports.ErrDependenciaNoDisponible
}

func TestInformeC12TransportaIdiomaActivoSinCuerpoNiURL(t *testing.T) {
	var idioma string
	h, err := NuevoManejadorInformeC12(casoInformeIdiomaPrueba{&idioma}, resolverInformeIdiomaPrueba{})
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodPost, "/api/interna/cronos/informes/saldo.pdf",
		strings.NewReader(`{"desde":"2026-10-01","hasta":"2026-10-02","clave_operacion":"prueba123"}`))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept-Language", "en")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if idioma != "en" || w.Code != http.StatusServiceUnavailable {
		t.Fatalf("idioma=%q estado=%d", idioma, w.Code)
	}
}

func TestInformeC12NoExponeClaveEnURLNiAceptaGET(t *testing.T) {
	llamadas := 0
	h, err := NuevoManejadorInformeC12(casoInformeC12Prueba{}, resolverInformeC12Prueba{&llamadas})
	if err != nil {
		t.Fatal(err)
	}
	url := "/api/interna/cronos/informes/saldo.pdf?desde=2026-10-01&hasta=2026-10-02&clave_operacion=prueba123"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, url, nil))
	if w.Code != http.StatusMethodNotAllowed || llamadas != 0 {
		t.Fatalf("GET: %d, resoluciones: %d", w.Code, llamadas)
	}
	r := httptest.NewRequest(http.MethodPost, url, strings.NewReader(`{"desde":"2026-10-01","hasta":"2026-10-02","clave_operacion":"prueba123"}`))
	r.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest || llamadas != 0 {
		t.Fatalf("query: %d, resoluciones: %d", w.Code, llamadas)
	}
}
