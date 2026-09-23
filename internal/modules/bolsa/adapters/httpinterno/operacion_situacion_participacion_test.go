package httpinterno

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type operadorOperacionesHTTPPrueba struct {
	resultado ports.RegistroSituacionParticipacion
	items     []ports.RegistroOperacionSituacion
	err       error
}

func (o operadorOperacionesHTTPPrueba) Operar(context.Context, ports.SolicitudOperacionSituacion) (ports.RegistroSituacionParticipacion, error) {
	return o.resultado, o.err
}
func (o operadorOperacionesHTTPPrueba) ListarOperaciones(context.Context, ports.SolicitudCambiarSituacionParticipacion) ([]ports.RegistroOperacionSituacion, error) {
	return o.items, o.err
}

func TestOperacionesSituacionContratoHTTP(t *testing.T) {
	ruta := RutaBolsasGestion + "/bolsa:01/candidatos/participacion:01/operaciones"
	ahora := time.Date(2026, 9, 23, 9, 0, 0, 0, time.UTC)
	cuerpo := `{"operacion":"pausar","motivo":"Solicitud registrada","validador":"per_validadora","justificante":{"tipo":"solicitud_candidato","referencia":"justificante:01","sha256":"` + strings.Repeat("a", 64) + `"}}`
	for _, caso := range []struct {
		reutilizada bool
		estado      int
	}{{false, 201}, {true, 200}} {
		o := operadorOperacionesHTTPPrueba{resultado: ports.RegistroSituacionParticipacion{Reutilizada: caso.reutilizada, ReciboRef: "recibo:01", SituacionParticipacion: ports.SituacionParticipacion{Situacion: "no_disponible", Desde: ahora}}}
		h, _ := NuevoHandlerOperacionesSituacion(preparadorSituacionHTTPPrueba{}, o)
		r := httptest.NewRequest(http.MethodPost, ruta, strings.NewReader(cuerpo))
		r.Header.Set("Accept", "application/json")
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Idempotency-Key", "b8-01")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != caso.estado || !strings.Contains(w.Body.String(), `"recibo_ref":"recibo:01"`) {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
	}
	h, _ := NuevoHandlerOperacionesSituacion(preparadorSituacionHTTPPrueba{}, operadorOperacionesHTTPPrueba{items: []ports.RegistroOperacionSituacion{}})
	r := httptest.NewRequest(http.MethodGet, ruta, nil)
	r.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"items":[]`) {
		t.Fatalf("GET status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestOperacionesSituacionDenegacionSinDatos(t *testing.T) {
	ruta := RutaBolsasGestion + "/bolsa:01/candidatos/participacion:01/operaciones"
	h, _ := NuevoHandlerOperacionesSituacion(preparadorSituacionHTTPPrueba{}, operadorOperacionesHTTPPrueba{err: dominiovec.ErrAutorizacionDenegada})
	r := httptest.NewRequest(http.MethodGet, ruta, nil)
	r.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 403 || strings.Contains(w.Body.String(), "participacion:01") {
		t.Fatalf("GET status=%d body=%s", w.Code, w.Body.String())
	}
}
