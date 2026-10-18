package httpinterno

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	vec "vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/pruebas"
)

type identidadJustificanteRRHHPrueba struct {
	valor IdentidadJustificanteRenunciaRRHH
	err   error
}

func (p identidadJustificanteRRHHPrueba) PrepararIdentidadJustificanteRenunciaRRHH(*http.Request) (IdentidadJustificanteRenunciaRRHH, error) {
	return p.valor, p.err
}

type consultaJustificanteRRHHPrueba struct {
	llamadas int
	bolsa    string
	parte    string
	bytes    string
	err      error
}

func (p *consultaJustificanteRRHHPrueba) AbrirJustificanteRenunciaRRHH(_ context.Context,
	_ IdentidadJustificanteRenunciaRRHH, bolsa, parte string) (io.ReadCloser, error) {
	p.llamadas++
	p.bolsa, p.parte = bolsa, parte
	if p.err != nil {
		return nil, p.err
	}
	return io.NopCloser(strings.NewReader(p.bytes)), nil
}

func TestJustificanteRenunciaRRHHSoloDescargaTrasIdentidad(t *testing.T) {
	actor, vinculo, err := pruebas.NuevoContextoRegistradoYVinculoV2(
		time.Date(2026, 10, 15, 8, 0, 0, 0, time.UTC), "per_0123456789abcdefghijkl",
		"prf_0123456789abcdefghijkl", vec.AuthMethodCertificate, vec.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	identidad := identidadJustificanteRRHHPrueba{valor: IdentidadJustificanteRenunciaRRHH{
		Actor: actor, Vinculo: vinculo, CorrelacionRef: "correlacion_" + strings.Repeat("a", 32)}}
	consulta := &consultaJustificanteRRHHPrueba{bytes: "%PDF-1.4\ncontenido\n%%EOF"}
	h, err := NuevoHandlerJustificanteRenunciaRRHH(identidad, consulta)
	if err != nil {
		t.Fatal(err)
	}
	ruta := RutaJustificanteRenunciaRRHH + "?bolsa=bolsa_1&participacion=part_1"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, ruta, nil))
	if w.Code != http.StatusOK || w.Header().Get("Content-Type") != "application/pdf" ||
		w.Header().Get("Content-Disposition") != `attachment; filename="justificante.pdf"` ||
		consulta.llamadas != 1 || consulta.bolsa != "bolsa_1" || consulta.parte != "part_1" ||
		w.Body.String() != consulta.bytes {
		t.Fatalf("descarga: estado=%d cabeceras=%v llamadas=%d", w.Code, w.Header(), consulta.llamadas)
	}
	consulta.llamadas = 0
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaJustificanteRenunciaRRHH+"?bolsa=bolsa_1&participacion=part_1&actor=otro", nil))
	if w.Code != http.StatusForbidden || consulta.llamadas != 0 {
		t.Fatalf("selector ajeno llegó a consulta: %d, llamadas=%d", w.Code, consulta.llamadas)
	}
	identidad.err = errors.New("sin sesión")
	h, _ = NuevoHandlerJustificanteRenunciaRRHH(identidad, consulta)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, ruta, nil))
	if w.Code != http.StatusForbidden || consulta.llamadas != 0 {
		t.Fatalf("sin sesión llegó a consulta: %d, llamadas=%d", w.Code, consulta.llamadas)
	}
	for _, rechazo := range []error{docports.ErrAccesoDenegado, docports.ErrNoEncontrado} {
		consulta.err = rechazo
		h, _ = NuevoHandlerJustificanteRenunciaRRHH(identidadJustificanteRRHHPrueba{valor: identidad.valor}, consulta)
		w = httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, ruta, nil))
		if w.Code != http.StatusForbidden || strings.Contains(w.Body.String(), "PDF") {
			t.Fatalf("rechazo documental: estado=%d", w.Code)
		}
	}
	consulta.err = docports.ErrSubidaLocalNoDisponible
	h, _ = NuevoHandlerJustificanteRenunciaRRHH(identidadJustificanteRRHHPrueba{valor: identidad.valor}, consulta)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, ruta, nil))
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("dependencia caída: %d", w.Code)
	}
}
