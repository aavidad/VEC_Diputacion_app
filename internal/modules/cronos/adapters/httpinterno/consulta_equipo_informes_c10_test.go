package httpinterno

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

type resolverEquipoC10Prueba struct{ llamadas *int }

func (r resolverEquipoC10Prueba) ResolverConsultaEquipo(*http.Request) (ports.OrdenConsultaEquipo, error) {
	*r.llamadas++
	return ports.OrdenConsultaEquipo{}, ports.ErrDependenciaNoDisponible
}

type casoEquipoC10Prueba struct{}

func (casoEquipoC10Prueba) ConsultarEquipo(_ context.Context, _ ports.OrdenConsultaEquipo, _ domain.TipoConsultaEquipo, _, _, _, _ string) (ports.ResultadoConsultaEquipo, error) {
	return ports.ResultadoConsultaEquipo{}, nil
}

func TestConsultaEquipoC10RechazaAlcanceDelClienteAntesDeResolver(t *testing.T) {
	llamadas := 0
	h, err := NuevoManejadorEquipoC10(casoEquipoC10Prueba{}, resolverEquipoC10Prueba{&llamadas})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/interna/cronos/equipos/eq_1/presencia?desde=2026-10-01&hasta=2026-10-02&alcance=rrhh", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest || llamadas != 0 {
		t.Fatalf("codigo=%d resoluciones=%d", w.Code, llamadas)
	}
}
