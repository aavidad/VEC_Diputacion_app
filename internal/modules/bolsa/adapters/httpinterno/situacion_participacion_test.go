package httpinterno

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type preparadorSituacionHTTPPrueba struct {
	solicitud puertosbolsa.SolicitudCambiarSituacionParticipacion
}

func (p preparadorSituacionHTTPPrueba) PrepararSolicitudCambiarSituacion(_ context.Context, e EntradaCambiarSituacionParticipacion) (puertosbolsa.SolicitudCambiarSituacionParticipacion, error) {
	s := p.solicitud
	s.BolsaRef = e.BolsaRef
	s.ParticipacionRef = e.ParticipacionRef
	s.Destino = e.Destino
	s.Motivo = e.Motivo
	s.ClaveIdempotencia = e.ClaveIdempotencia
	s.FechaDisponible = e.FechaDisponible
	return s, nil
}

type operadorSituacionHTTPPrueba struct {
	resultado puertosbolsa.RegistroSituacionParticipacion
	err       error
}

func (o operadorSituacionHTTPPrueba) Cambiar(context.Context, puertosbolsa.SolicitudCambiarSituacionParticipacion) (puertosbolsa.RegistroSituacionParticipacion, error) {
	return o.resultado, o.err
}

func TestHandlerSituacionParticipacionDevuelveReciboYReplay(t *testing.T) {
	ahora := time.Date(2026, 9, 21, 18, 0, 0, 0, time.UTC)
	for _, caso := range []struct {
		reutilizada bool
		estado      int
	}{{false, 201}, {true, 200}} {
		o := operadorSituacionHTTPPrueba{resultado: puertosbolsa.RegistroSituacionParticipacion{Reutilizada: caso.reutilizada, ReciboRef: "recibo:situacion:01", SituacionParticipacion: puertosbolsa.SituacionParticipacion{ParticipacionRef: "participacion:01", Situacion: "no_disponible", Desde: ahora}}}
		h, _ := NuevoHandlerSituacionParticipacion(preparadorSituacionHTTPPrueba{}, o)
		r := httptest.NewRequest(http.MethodPost, RutaBolsasGestion+"/bolsa:01/candidatos/participacion:01/situacion", strings.NewReader(`{"situacion":"no_disponible","motivo":"Pausa comunicada","fecha_disponible":null}`))
		r.Header.Set("Accept", "application/json")
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Idempotency-Key", "b2-cambio-0001")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != caso.estado || !strings.Contains(w.Body.String(), `"recibo_ref":"recibo:situacion:01"`) {
			t.Fatalf("replay=%v status=%d body=%s", caso.reutilizada, w.Code, w.Body.String())
		}
	}
}

func TestHandlerSituacionParticipacionDistingueDenegacionYConflicto(t *testing.T) {
	for _, caso := range []struct {
		err    error
		estado int
		codigo string
	}{{dominiovec.ErrAutorizacionDenegada, http.StatusForbidden, "acceso_denegado"}, {dominiobolsa.ErrCambioSituacionParticipacionInvalido, http.StatusConflict, "cambio_en_conflicto"}} {
		h, _ := NuevoHandlerSituacionParticipacion(preparadorSituacionHTTPPrueba{}, operadorSituacionHTTPPrueba{err: caso.err})
		r := httptest.NewRequest(http.MethodPost, RutaBolsasGestion+"/bolsa:01/candidatos/participacion:01/situacion", strings.NewReader(`{"situacion":"no_disponible","motivo":"Pausa comunicada","fecha_disponible":null}`))
		r.Header.Set("Accept", "application/json")
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Idempotency-Key", "b2-cambio-0001")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != caso.estado || !strings.Contains(w.Body.String(), caso.codigo) {
			t.Fatalf("err=%v estado=%d cuerpo=%s", caso.err, w.Code, w.Body.String())
		}
	}
}
