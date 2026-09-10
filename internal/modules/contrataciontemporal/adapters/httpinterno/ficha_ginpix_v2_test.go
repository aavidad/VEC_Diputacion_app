package httpinterno

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"vec-diputacion-granada/internal/app/incorporacionejercicio"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/ginpixfichero"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

type autoridadFichaPrueba struct{ err error }

func (x autoridadFichaPrueba) ResolverContextoFichaGINPIXV2(context.Context) error { return x.err }

type preparadorFichaPrueba struct {
	p   ginpixfichero.PreparacionExportacion
	err error
}

func (x preparadorFichaPrueba) Preparar(context.Context, string) (ginpixfichero.PreparacionExportacion, error) {
	return x.p, x.err
}
func TestManejadorFichaGINPIXV2RechazaMetodoYDenegacion(t *testing.T) {
	for _, tc := range []struct {
		metodo string
		err    error
		want   int
	}{{http.MethodPost, incorporacionejercicio.ErrFichaGINPIXV2Denegada, 405}, {http.MethodGet, incorporacionejercicio.ErrFichaGINPIXV2Denegada, 403}, {http.MethodGet, incorporacionejercicio.ErrFichaGINPIXV2Conflicto, 409}, {http.MethodGet, incorporacionejercicio.ErrFichaGINPIXV2NoDisponible, 503}} {
		h, e := NuevoManejadorFichaGINPIXV2(autoridadFichaPrueba{}, preparadorFichaPrueba{err: tc.err})
		if e != nil {
			t.Fatal(e)
		}
		r := httptest.NewRequest(tc.metodo, RutaFichaGINPIXV2+"?expediente_ref=expediente:uno", nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != tc.want {
			t.Fatalf("%s: got %d", tc.metodo, w.Code)
		}
	}
}

func TestManejadorFichaGINPIXV2EntregaJSONConUTF8(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, RutaFichaGINPIXV2+"?expediente_ref=expediente:uno", nil)
	w := httptest.NewRecorder()
	h, e := NuevoManejadorFichaGINPIXV2(autoridadFichaPrueba{}, preparadorFichaPrueba{p: preparacionFichaGINPIXV2Prueba(t)})
	if e != nil {
		t.Fatal(e)
	}
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("estado: got %d", w.Code)
	}
	if got := w.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("content-type: got %q", got)
	}
}

func preparacionFichaGINPIXV2Prueba(t *testing.T) ginpixfichero.PreparacionExportacion {
	t.Helper()
	campo, err := domain.CampoValorGINPIX("expediente:uno")
	if err != nil {
		t.Fatal(err)
	}
	modelo, err := domain.NuevoModeloCanonicoGINPIX(domain.BorradorModeloCanonicoGINPIX{
		Esquema: domain.EsquemaModeloCanonicoGINPIXV1, VersionExpediente: 1,
		ExpedienteRef: "expediente:uno", IncorporacionRef: "actuacion:uno",
		ProcedenciaRef: "recibo:uno", CorrelacionRef: "correlacion:uno", IdempotenciaRef: "idempotencia:uno",
		Datos: []domain.DatoCanonicoGINPIX{{Clave: "expediente_ref", Campo: campo}},
	})
	if err != nil {
		t.Fatal(err)
	}
	mapeo, err := domain.PublicarMapeoVersionadoGINPIX(domain.BorradorMapeoVersionadoGINPIX{
		Esquema: domain.EsquemaMapeoGINPIXV1, Referencia: "mapeo:uno", Version: 1, ProcedenciaRef: "configuracion:uno",
		Reglas: []domain.ReglaMapeoGINPIX{{CampoCanonico: "expediente_ref", CampoDestino: "ginpix_expediente", Obligatorio: true}},
	})
	if err != nil {
		t.Fatal(err)
	}
	carga, err := domain.AplicarMapeoGINPIX(modelo, mapeo)
	if err != nil {
		t.Fatal(err)
	}
	p, err := ginpixfichero.PrepararExportacion(carga)
	if err != nil {
		t.Fatal(err)
	}
	return p
}
