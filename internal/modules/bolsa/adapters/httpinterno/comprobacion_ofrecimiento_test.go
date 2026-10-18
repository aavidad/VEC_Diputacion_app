package httpinterno

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
)

type preparadorComprobacionPrueba struct{ err error }

func (p preparadorComprobacionPrueba) PrepararComprobacionOfrecimiento(_ context.Context, e EntradaComprobacionOfrecimiento) (ports.SolicitudComprobacion, error) {
	if p.err != nil {
		return ports.SolicitudComprobacion{}, p.err
	}
	return ports.SolicitudComprobacion{Contexto: e.Contexto, Hechos: e.Hechos, VersionEsperada: e.VersionEsperada, ClaveIdempotencia: e.ClaveIdempotencia}, nil
}

func TestComprobacionHTTPResultadoNoConfirmado(t *testing.T) {
	w := httptest.NewRecorder()
	responderErrorComprobacion(w, errors.Join(ports.ErrComprobacionResultadoNoConfirmado, vd.ErrAutorizacionDenegada))
	if w.Code != 503 || !strings.Contains(w.Body.String(), `"resultado_no_confirmado"`) {
		t.Fatalf("estado ambiguo: %d %s", w.Code, w.Body.String())
	}
}

type servicioComprobacionPrueba struct {
	recibo   ports.ReciboComprobacion
	err      error
	llamadas *int
}

func (s servicioComprobacionPrueba) Registrar(_ context.Context, _ ports.SolicitudComprobacion) (ports.ReciboComprobacion, error) {
	if s.llamadas != nil {
		*s.llamadas++
	}
	return s.recibo, s.err
}
func peticionComprobacionPrueba(cuerpo string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/api/vec/bolsa/bolsas/bolsa:uno/candidatos/participacion:uno/comprobaciones-ofrecimiento", strings.NewReader(cuerpo))
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Idempotency-Key", "clave:12345678")
	return r
}
func TestComprobacionHTTPFronteraYReplay(t *testing.T) {
	cuerpo := `{"contexto":{"bolsa_ref":"bolsa:uno","participacion_ref":"participacion:uno"},"hechos":{},"version_esperada":0}`
	llamadas := 0
	s := servicioComprobacionPrueba{recibo: ports.ReciboComprobacion{ReciboRef: "recibo:uno"}, llamadas: &llamadas}
	for _, caso := range []struct {
		nombre string
		h      http.Handler
		r      *http.Request
		status int
		codigo string
	}{
		{"dependencia", NuevoHandlerComprobacionOfrecimiento(nil, nil), peticionComprobacionPrueba(cuerpo), 503, "comprobacion_ofrecimiento_no_habilitada"},
		{"camino", NuevoHandlerComprobacionOfrecimiento(preparadorComprobacionPrueba{}, s), peticionComprobacionPrueba(cuerpo), 201, ""},
		{"contexto ajeno", NuevoHandlerComprobacionOfrecimiento(preparadorComprobacionPrueba{}, s), peticionComprobacionPrueba(strings.Replace(cuerpo, "participacion:uno", "participacion:otra", 1)), 400, "solicitud_invalida"},
		{"campo extraño", NuevoHandlerComprobacionOfrecimiento(preparadorComprobacionPrueba{}, s), peticionComprobacionPrueba(strings.Replace(cuerpo, `"hechos":{}`, `"nombre":"X","hechos":{}`, 1)), 400, "solicitud_invalida"},
		{"denegacion", NuevoHandlerComprobacionOfrecimiento(preparadorComprobacionPrueba{}, servicioComprobacionPrueba{err: vd.ErrAutorizacionDenegada}), peticionComprobacionPrueba(cuerpo), 403, "acceso_denegado"},
		{"conflicto", NuevoHandlerComprobacionOfrecimiento(preparadorComprobacionPrueba{}, servicioComprobacionPrueba{err: ports.ErrComprobacionConflicto}), peticionComprobacionPrueba(cuerpo), 409, "comprobacion_en_conflicto"},
		{"malformada", NuevoHandlerComprobacionOfrecimiento(preparadorComprobacionPrueba{}, servicioComprobacionPrueba{err: ports.ErrComprobacionInvalida}), peticionComprobacionPrueba(cuerpo), 400, "solicitud_invalida"},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			w := httptest.NewRecorder()
			caso.h.ServeHTTP(w, caso.r)
			if w.Code != caso.status {
				t.Fatalf("estado %d: %s", w.Code, w.Body.String())
			}
			if caso.codigo != "" && !strings.Contains(w.Body.String(), caso.codigo) {
				t.Fatalf("codigo: %s", w.Body.String())
			}
		})
	}
	if llamadas != 1 {
		t.Fatalf("servicio llamado %d veces", llamadas)
	}
	s.recibo.Reutilizada = true
	w := httptest.NewRecorder()
	NuevoHandlerComprobacionOfrecimiento(preparadorComprobacionPrueba{}, s).ServeHTTP(w, peticionComprobacionPrueba(cuerpo))
	if w.Code != 200 {
		t.Fatalf("replay %d", w.Code)
	}
	var respuesta map[string]map[string]any
	if json.Unmarshal(w.Body.Bytes(), &respuesta) != nil || respuesta["data"]["recibo_ref"] != "recibo:uno" {
		t.Fatal("recibo original perdido")
	}
}
