package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/config"
)

type comprobadorDisponibilidadPrueba struct {
	err    error
	llamas int
}

func (c *comprobadorDisponibilidadPrueba) ComprobarDisponibilidad(context.Context) error {
	c.llamas++
	return c.err
}

func TestRutasDisponibilidadPublicasSonGenericasYSoloLectura(t *testing.T) {
	comprobador := &comprobadorDisponibilidadPrueba{err: errors.New("dsn=secreto")}
	handler := NewHandlerPublicoWithConfigConComprobadorDisponibilidad(config.Config{}, http.NotFoundHandler(), comprobador)
	for _, caso := range []struct {
		ruta   string
		metodo string
		estado int
	}{
		{"/livez", http.MethodGet, http.StatusOK},
		{"/readyz", http.MethodGet, http.StatusServiceUnavailable},
		{"/healthz", http.MethodHead, http.StatusServiceUnavailable},
		{"/readyz", http.MethodPost, http.StatusMethodNotAllowed},
	} {
		req := httptest.NewRequest(caso.metodo, caso.ruta, nil)
		req.RemoteAddr = "127.0.0.1:1234"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != caso.estado || rr.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("%s %s = %d, cache=%q", caso.metodo, caso.ruta, rr.Code, rr.Header().Get("Cache-Control"))
		}
		if rr.Body.String() != "" && (strings.Contains(rr.Body.String(), "secreto") || strings.Contains(rr.Body.String(), "dsn")) {
			t.Fatalf("respuesta filtro detalle: %q", rr.Body.String())
		}
	}
}

func TestDisponibilidadNoInvocaComprobadorConCredencialesProhibidas(t *testing.T) {
	for _, cabecera := range []string{"Cookie", "Authorization", "Proxy-Authorization"} {
		t.Run(cabecera, func(t *testing.T) {
			comprobador := &comprobadorDisponibilidadPrueba{}
			handler := NewHandlerPublicoWithConfigConComprobadorDisponibilidad(config.Config{}, http.NotFoundHandler(), comprobador)
			req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
			req.RemoteAddr = "127.0.0.1:1234"
			req.Header.Set(cabecera, "prohibida")
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			if rr.Code != http.StatusBadRequest || comprobador.llamas != 0 {
				t.Fatalf("respuesta=%d llamadas=%d", rr.Code, comprobador.llamas)
			}
		})
	}
}
