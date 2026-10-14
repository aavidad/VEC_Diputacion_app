package httpinterno

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

type recuperacionCorreccionPrueba struct {
	clave    ports.ClaveRecuperacionCorreccion
	err      error
	llamadas int
}

func (c *recuperacionCorreccionPrueba) RecuperarRecibo(_ context.Context, _ ports.OrdenConsumoCorreccion, clave ports.ClaveRecuperacionCorreccion) (ports.ReciboCorreccion, error) {
	c.llamadas++
	c.clave = clave
	return ports.ReciboCorreccion{
		SolicitudRef: clave.SolicitudRef, ActuacionRef: "correccion:actuacion:original",
		ReciboRef: "recibo:cronos:original", Estado: domain.CorreccionPendienteResponsable,
		Version: 1, InstanteUTC: time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC), Replay: true,
	}, c.err
}

func TestRecuperacionCorreccionSoloAceptaClaveYDerivaSolicitud(t *testing.T) {
	caso, resolver := &recuperacionCorreccionPrueba{}, &resolverSolicitudesPrueba{}
	m, err := NuevoManejadorRecuperacionCorreccionPropia(caso, resolver)
	if err != nil {
		t.Fatal(err)
	}
	clave := "01234567-89ab-4cde-8fab-0123456789ab"
	ruta := RutaRecuperarReciboCorreccionPropia + "?clave_operacion=" + clave
	w := httptest.NewRecorder()
	m.ServeHTTP(w, httptest.NewRequest(http.MethodGet, ruta, nil))
	if w.Code != http.StatusOK || resolver.llamadas != 1 || caso.llamadas != 1 ||
		caso.clave != (ports.ClaveRecuperacionCorreccion{SolicitudRef: "correccion:cronos:" + clave, ClaveOperacion: clave, Paso: domain.PasoSolicitudCorreccion}) ||
		!strings.Contains(w.Body.String(), `"replay":true`) || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("recuperación propia: %d %s", w.Code, w.Body.String())
	}
	for _, mala := range []string{
		RutaRecuperarReciboCorreccionPropia,
		ruta + "&empleado_ref=emp_ajeno", ruta + "&actor_ref=per_ajena",
		ruta + "&clave_operacion=" + clave,
		RutaRecuperarReciboCorreccionPropia + "?clave_operacion=olvido_0001",
	} {
		w = httptest.NewRecorder()
		m.ServeHTTP(w, httptest.NewRequest(http.MethodGet, mala, nil))
		if w.Code != http.StatusBadRequest || caso.llamadas != 1 || resolver.llamadas != 1 {
			t.Fatalf("entrada ajena llegó a autoridad: %d %s", w.Code, mala)
		}
	}
}

func TestRecuperacionCorreccionDenegadaYDependenciaNoExponenRecibo(t *testing.T) {
	caso, resolver := &recuperacionCorreccionPrueba{}, &resolverSolicitudesPrueba{}
	m, _ := NuevoManejadorRecuperacionCorreccionPropia(caso, resolver)
	ruta := RutaRecuperarReciboCorreccionPropia + "?clave_operacion=01234567-89ab-4cde-8fab-0123456789ab"
	for _, prueba := range []struct {
		err    error
		estado int
	}{
		{ports.ErrCorreccionNoAutorizada, http.StatusForbidden},
		{ports.ErrDependenciaNoDisponible, http.StatusServiceUnavailable},
		{errors.New("fallo interno"), http.StatusServiceUnavailable},
	} {
		caso.err = prueba.err
		w := httptest.NewRecorder()
		m.ServeHTTP(w, httptest.NewRequest(http.MethodGet, ruta, nil))
		if w.Code != prueba.estado || strings.Contains(w.Body.String(), "recibo:cronos:") || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("filtración de recibo: %d %s", w.Code, w.Body.String())
		}
	}
}
