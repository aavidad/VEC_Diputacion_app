package capturacopias

import (
	"context"
	"errors"
	"testing"
)

func TestVentanaRetenidaHastaCerrarYNoReutilizable(t *testing.T) {
	p, s, solicitud := ejemplo(t)
	w, err := s.AbrirVentana(context.Background(), solicitud)
	if err != nil {
		t.Fatal(err)
	}
	defer s.CerrarVentana(w)
	r, err := s.CapturarEnVentana(context.Background(), w)
	if err != nil {
		t.Fatal(err)
	}
	for _, paso := range p.pasos {
		if paso == "reabrir" || paso == "liberar" {
			t.Fatal("libera antes sustitución")
		}
	}
	if err = s.ComprobarVentana(context.Background(), w, solicitud); err != nil {
		t.Fatal(err)
	}
	otra := solicitud
	otra.OperacionRef = "operacion:distinta"
	if !errors.Is(s.ComprobarVentana(context.Background(), w, otra), ErrPrecondicion) {
		t.Fatal("operación distinta admitida")
	}
	if _, err = s.CapturarEnVentana(context.Background(), w); !errors.Is(err, ErrPrecondicion) {
		t.Fatal("repite captura")
	}
	if s.CerrarVentana(w) != nil || s.CerrarVentana(w) != nil {
		t.Fatal("cierre idempotente")
	}
	if r.Completa || r.Valida || r.Publicable {
		t.Fatal(r)
	}
	if !errors.Is(s.ComprobarVentana(context.Background(), w, solicitud), ErrPrecondicion) {
		t.Fatal("ventana cerrada usable")
	}
	if _, err = s.CapturarEnVentana(context.Background(), &Ventana{}); !errors.Is(err, ErrPrecondicion) {
		t.Fatal("capability fabricada")
	}
}

func TestCancelacionVentanaRetenidaNoReabreHastaOrdenPropietario(t *testing.T) {
	p, s, solicitud := ejemplo(t)
	ctx, cancel := context.WithCancel(context.Background())
	w, err := s.AbrirVentana(ctx, solicitud)
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	if !errors.Is(s.ComprobarVentana(context.Background(), w, solicitud), ErrPrecondicion) {
		t.Fatal("contexto cancelado usable")
	}
	for _, paso := range p.pasos {
		if paso == "reabrir" {
			t.Fatal("reabre automáticamente")
		}
	}
	if s.CerrarVentana(w) != nil {
		t.Fatal("no limpia tras cancelación")
	}
}

func TestPreimagenSeSellaSinCompartirSlicesDelSolicitante(t *testing.T) {
	_, s, solicitud := ejemplo(t)
	w, err := s.AbrirVentana(context.Background(), solicitud)
	if err != nil {
		t.Fatal(err)
	}
	defer s.CerrarVentana(w)
	solicitud.Esperado.PostgreSQL.Bases[0] = "base:otra"
	if !errors.Is(s.ComprobarVentana(context.Background(), w, solicitud), ErrPrecondicion) {
		t.Fatal("acepta preimagen alterada")
	}
}
