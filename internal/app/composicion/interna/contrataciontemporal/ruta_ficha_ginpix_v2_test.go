package contrataciontemporal

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	inc "vec-diputacion-granada/internal/app/incorporacionejercicio"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	dom "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type fuenteFichaRutaPrueba struct{ lecturas *int }

func (f fuenteFichaRutaPrueba) RecuperarFichaGINPIXV2(context.Context, string) (inc.RecuperacionFichaGINPIXV2, error) {
	*f.lecturas++
	return inc.RecuperacionFichaGINPIXV2{}, ct.ErrConflictoIncorporacionAplicacion
}

type mapeoFichaRutaPrueba struct{ t *testing.T }

func (m mapeoFichaRutaPrueba) ResolverMapeoFichaGINPIXV2(context.Context, ct.ReciboIncorporacionAplicacionV2) (dom.MapeoVersionadoGINPIX, error) {
	m.t.Fatal("se pidió mapeo sin recibo")
	return dom.MapeoVersionadoGINPIX{}, nil
}
func TestFichaGINPIXV2RutaPeticionNuevaSinEfectos(t *testing.T) {
	nuevas, lecturas := 0, 0
	h := manejadorFichaGINPIXV2{mapeos: mapeoFichaRutaPrueba{t}, nueva: func(context.Context) (inc.FuenteFichaGINPIXV2, error) {
		nuevas++
		return fuenteFichaRutaPrueba{&lecturas}, nil
	}}
	for _, metodo := range []string{http.MethodGet, http.MethodGet, http.MethodPost} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(metodo, httpinterno.RutaFichaGINPIXV2+"?expediente_ref=expediente:real:1", nil)
		h.ServeHTTP(w, r)
		esperado := 409
		if metodo == http.MethodPost {
			esperado = 405
		}
		if w.Code != esperado {
			t.Fatalf("%s: %d", metodo, w.Code)
		}
	}
	if nuevas != 2 || lecturas != 2 {
		t.Fatalf("peticiones compartidas o POST ejecutado: %d/%d", nuevas, lecturas)
	}
	h.nueva = func(context.Context) (inc.FuenteFichaGINPIXV2, error) {
		return nil, ct.ErrDenegadaIncorporacionAplicacion
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, httpinterno.RutaFichaGINPIXV2+"?expediente_ref=expediente:real:1", nil))
	if w.Code != 403 || lecturas != 2 {
		t.Fatalf("denegacion leyó datos: %d/%d", w.Code, lecturas)
	}
}
