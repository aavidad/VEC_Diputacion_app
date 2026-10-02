package application

import (
	"context"
	"errors"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestRegistroMOADExigeNumeroAntesDeReservar(t *testing.T) {
	e := nuevoEscenarioRegistro(t)
	s, d := construirServicioRegistro(t, e)
	e.solicitud.NumeroExpedienteMOAD = ""
	if _, err := s.Registrar(context.Background(), e.solicitud); !errors.Is(err, ErrSolicitudRegistroInvalida) {
		t.Fatalf("alta sin MOAD: %v", err)
	}
	if d.referencias.llamadasReferencias != 0 {
		t.Fatal("se generaron referencias para un número ausente")
	}
	p := domain.PoliticaNumeroExpediente{Referencia: "catalogo:ct:moad", Version: 2, Patron: `^[0-9]{4}/[1-9][0-9]{0,9}$`, Ejemplo: "2026/5487"}
	s.politicaNumero = &p
	e.solicitud.NumeroExpedienteMOAD = "2026/CT-0001"
	if _, err := s.Registrar(context.Background(), e.solicitud); !errors.Is(err, ErrSolicitudRegistroInvalida) {
		t.Fatalf("se ignoró formato configurado: %v", err)
	}
	if d.referencias.llamadasReferencias != 0 {
		t.Fatal("se reservó un número rechazado por el catálogo")
	}
}

func TestRegistroMOADNoAdmiteSustituirNumeroEnCandidatura(t *testing.T) {
	e := nuevoEscenarioRegistro(t)
	s, d := construirServicioRegistro(t, e)
	e.solicitud.NumeroExpedienteMOAD = "2026/5487"
	d.referencias.referencias.NumeroVisible = e.solicitud.NumeroExpedienteMOAD
	if _, err := s.Registrar(context.Background(), e.solicitud); !errors.Is(err, ports.ErrClaveIdempotenciaUsada) {
		t.Fatalf("otra numeración reservada: %v", err)
	}
}
