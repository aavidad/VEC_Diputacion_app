package registropropio

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type preparadorPrueba struct{ llamadas int }

func (p *preparadorPrueba) PrepararSolicitudRegistroPropio(context.Context) (ports.SolicitudRegistroPropioV1, error) {
	p.llamadas++
	return ports.SolicitudRegistroPropioV1{}, nil
}

type registroPrueba struct {
	llamadas int
	recibo   domain.ReciboRegistroPropioV1
}

func (r *registroPrueba) Registrar(context.Context, ports.SolicitudRegistroPropioV1) (domain.ReciboRegistroPropioV1, error) {
	r.llamadas++
	return r.recibo, nil
}

type contactoPrueba struct {
	persona  string
	llamadas int
}

func (c *contactoPrueba) CompletarContactoPropio(_ context.Context, _ domain.ReciboRegistroPropioV1, _ string) (ResultadoContactoAlta, error) {
	c.llamadas++
	return ResultadoContactoAlta{PersonaRef: c.persona, Version: 1, ReciboRef: "rco_abcdefghijklmnopqrstuv", EvidenciaRef: "evi_abcdefghijklmnopqrstuv"}, nil
}

func reciboPendientePrueba() domain.ReciboRegistroPropioV1 {
	return domain.ReciboRegistroPropioV1{OperacionRef: "opr_abcdefghijklmnopqrstuv", ReciboRef: "rpr_abcdefghijklmnopqrstuv", Estado: domain.EstadoRegistroPropioPendienteContacto,
		CuentaRef: "cta_abcdefghijklmnopqrstuv", CuentaVersion: 1, PersonaRef: "per_abcdefghijklmnopqrstuv", PersonaVersion: 1,
		PerfilRef: "prf_abcdefghijklmnopqrstuv", PerfilVersion: 1, VinculoRef: "vca_abcdefghijklmnopqrstuv", VinculoVersion: 1,
		ProcedenciaRef: "prc_abcdefghijklmnopqrstuv", ProcedenciaVersion: 1, ProcedenciaSHA256: strings.Repeat("a", 64)}
}

func TestRegistroPropioHTTPNoCierraAltaConContactoAjenoNiExponeCorreo(t *testing.T) {
	p := &preparadorPrueba{}
	r := &registroPrueba{recibo: reciboPendientePrueba()}
	c := &contactoPrueba{persona: "per_bbbbbbbbbbbbbbbbbbbbbb"}
	h, err := NuevoHandler(p, r, c)
	if err != nil {
		t.Fatal(err)
	}
	respuesta := httptest.NewRecorder()
	h.ServeHTTP(respuesta, httptest.NewRequest(http.MethodPost, "/api/vec/usuarios/registro-propio", strings.NewReader(`{"correo":"prueba@example.invalid","confirmacion":"prueba@example.invalid"}`)))
	if respuesta.Code != http.StatusAccepted || !strings.Contains(respuesta.Body.String(), "pendiente_contacto") || strings.Contains(respuesta.Body.String(), "prueba@example.invalid") || p.llamadas != 1 || r.llamadas != 1 || c.llamadas != 1 {
		t.Fatalf("alta cerrada o correo expuesto: estado=%d cuerpo=%s", respuesta.Code, respuesta.Body.String())
	}
	respuesta = httptest.NewRecorder()
	h.ServeHTTP(respuesta, httptest.NewRequest(http.MethodPost, "/api/vec/usuarios/registro-propio", strings.NewReader(`{"correo":"prueba@example.invalid","confirmacion":"otra@example.invalid"}`)))
	if respuesta.Code != http.StatusBadRequest || r.llamadas != 1 {
		t.Fatal("confirmacion distinta produjo efecto")
	}
}

func TestRegistroPropioSinAutoridadInstitucionalDeniega(t *testing.T) {
	f := FuenteInstitucionalNoDisponible{}
	if _, err := f.AcreditarRegistroPropio(context.Background(), "cre_abcdefghijklmnopqrstuv"); !errors.Is(err, ports.ErrRegistroPropioNoDisponible) {
		t.Fatal("fuente ausente aceptada")
	}
	if _, err := f.ResolverEquivalenciaPersona(context.Background(), "opr_abcdefghijklmnopqrstuv", domain.AcreditacionInstitucionalRegistroPropioV1{}); !errors.Is(err, ports.ErrRegistroPropioNoDisponible) {
		t.Fatal("equivalencia ausente aceptada")
	}
	if _, err := NuevoHandler(nil, &registroPrueba{}, &contactoPrueba{}); !errors.Is(err, ports.ErrRegistroPropioNoDisponible) {
		t.Fatal("handler sin frontera montado")
	}
}
