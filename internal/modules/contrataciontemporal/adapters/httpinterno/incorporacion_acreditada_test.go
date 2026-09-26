package httpinterno

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type ejecutorGINPIXPrueba struct {
	ejecutorSeguimientoPrueba
	ginpix application.SolicitudConfirmarGINPIX
	estado ports.EstadoSeguimientoExpediente
	conGIN bool
}

func (e *ejecutorGINPIXPrueba) ConfirmarGINPIX(_ context.Context, s application.SolicitudConfirmarGINPIX) (ports.ReciboOperacionSeguimiento, error) {
	e.ginpix = s
	r, err := e.recibo(ports.OperacionConfirmarGINPIX)
	r.CausaClave, r.FechaEfecto, r.GINPIXNumero = "", "", s.GINPIXNumero
	return r, err
}
func (e *ejecutorGINPIXPrueba) Opciones(ctx context.Context) (ports.OpcionesSeguimiento, error) {
	o, err := e.ejecutorSeguimientoPrueba.Opciones(ctx)
	o.ConfirmacionGINPIX = e.conGIN
	return o, err
}
func (e *ejecutorGINPIXPrueba) Estado(context.Context, string, string) (ports.EstadoSeguimientoExpediente, error) {
	return e.estado, nil
}

func TestConfirmacionGINPIXHTTPSoloConEjecutorYConSusConflictos(t *testing.T) {
	sin, _ := NuevosManejadoresSeguimiento(autoridadSeguimientoPrueba{}, autoridadSeguimientoPrueba{}, &ejecutorSeguimientoPrueba{})
	if _, ok := sin[RutaConfirmacionesGINPIX]; ok {
		t.Fatal("sin incorporación acreditada no hay ruta de GINPIX")
	}
	e := &ejecutorGINPIXPrueba{conGIN: true, estado: ports.EstadoSeguimientoExpediente{ExpedienteRef: "expediente:prueba",
		Acreditada: &ports.EstadoIncorporacionAcreditada{GINPIX: &ports.EstadoGINPIXConfirmado{Numero: "GX-1", ConfirmadaEn: "2027-02-10",
			ReciboRef: "recibo:ginpix:1", RegistradaEn: time.Date(2027, 2, 10, 9, 0, 0, 0, time.UTC)}}}}
	m, err := NuevosManejadoresSeguimiento(autoridadSeguimientoPrueba{}, autoridadSeguimientoPrueba{}, e)
	if err != nil {
		t.Fatal(err)
	}
	cuerpo := `{"expediente_ref":"expediente:prueba","version_esperada":7,"clave_idempotencia":"44444444-4444-4444-8444-444444444444","ginpix_numero":"GX-1","ginpix_confirmada_en":"2027-02-10","observaciones":""}`
	w := peticionSeguimiento(t, m, RutaConfirmacionesGINPIX, cuerpo)
	if w.Code != http.StatusCreated || e.ginpix.GINPIXNumero != "GX-1" || !e.ginpix.GINPIXConfirmada.Equal(time.Date(2027, 2, 10, 0, 0, 0, 0, time.UTC)) ||
		!strings.Contains(w.Body.String(), `"ginpix_numero":"GX-1"`) || !strings.Contains(w.Body.String(), `"operacion":"confirmar_ginpix"`) {
		t.Fatalf("confirmación: %d %s", w.Code, w.Body.String())
	}
	if w := peticionSeguimiento(t, m, RutaConfirmacionesGINPIX, strings.Replace(cuerpo, `"2027-02-10"`, `""`, 1)); w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("sin fecha: %d", w.Code)
	}
	for err, codigo := range map[error]string{ports.ErrGINPIXYaConfirmado: "ginpix_existente", ports.ErrGINPIXNoConfirmado: "ginpix_no_confirmado",
		ports.ErrGINPIXDistinto: "ginpix_distinto"} {
		e.error = err
		w := peticionSeguimiento(t, m, RutaConfirmacionesGINPIX, cuerpo)
		if w.Code != http.StatusConflict || !strings.Contains(w.Body.String(), `"codigo":"`+codigo+`"`) {
			t.Fatalf("%v: %d %s", err, w.Code, w.Body.String())
		}
	}
	e.error = nil
	w = peticionSeguimiento(t, m, RutaSeguimientoCese, `{"expediente_ref":"expediente:prueba"}`)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"confirmacion_ginpix":true`) || !strings.Contains(w.Body.String(), `"ginpix_numero":"GX-1"`) ||
		!strings.Contains(w.Body.String(), `"confirmacion_centro":null`) {
		t.Fatalf("consulta con incorporación acreditada: %d %s", w.Code, w.Body.String())
	}
}

func TestAdmisionCierreSinCeseLaDecideLaRegla(t *testing.T) {
	llamado := false
	base := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { llamado = true; w.WriteHeader(http.StatusOK) })
	if ExigirAdmisionCierreSinCese(base, nil) == nil {
		t.Fatal("sin decisión se conserva el manejador")
	}
	casos := []struct {
		admitido bool
		err      error
		estado   int
		llamado  bool
	}{{true, nil, http.StatusOK, true}, {false, nil, http.StatusConflict, false}, {false, errors.New("regla ilegible"), http.StatusServiceUnavailable, false}}
	for _, c := range casos {
		llamado = false
		h := ExigirAdmisionCierreSinCese(base, func(context.Context) (bool, error) { return c.admitido, c.err })
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaPreparacionCierreSinCese, nil))
		if w.Code != c.estado || llamado != c.llamado {
			t.Fatalf("%+v: %d %v %s", c, w.Code, llamado, w.Body.String())
		}
		if !c.admitido && c.err == nil && !strings.Contains(w.Body.String(), `"codigo":"`+CodigoCierreSinCeseNoContemplado+`"`) {
			t.Fatalf("código del rechazo: %s", w.Body.String())
		}
	}
}
