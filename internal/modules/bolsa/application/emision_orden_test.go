package application

import (
	"context"
	"errors"
	"testing"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

type motivosSaltoPrueba struct {
	motivos []puertosbolsa.MotivoSaltoOrden
	err     error
}

func (m motivosSaltoPrueba) MotivosSaltoOrden(context.Context) ([]puertosbolsa.MotivoSaltoOrden, error) {
	return m.motivos, m.err
}
func (m motivosSaltoPrueba) ResolverMotivoSaltoOrden(_ context.Context, clave string) (puertosbolsa.SaltoOrdenLlamamiento, error) {
	for _, motivo := range m.motivos {
		if motivo.Clave == clave {
			return puertosbolsa.SaltoOrdenLlamamiento{Clave: clave, ReglaRef: "vec.bolsa.reglas:8:" + clave, HuellaCatalogo: "h"}, nil
		}
	}
	if m.err != nil {
		return puertosbolsa.SaltoOrdenLlamamiento{}, m.err
	}
	return puertosbolsa.SaltoOrdenLlamamiento{}, puertosbolsa.ErrMotivoSaltoNoAdmitido
}

func TestMotivoSaltoSoloDelCatalogo(t *testing.T) {
	ctx := t.Context()
	sin := &ServicioEmisionLlamamiento{}
	if salto, err := sin.resolverMotivoSalto(ctx, ""); salto != nil || err != nil {
		t.Fatal("sin motivo no hay salto")
	}
	if _, err := sin.resolverMotivoSalto(ctx, "b01.salto.x"); !errors.Is(err, puertosbolsa.ErrMotivoSaltoNoAdmitido) || !errors.Is(err, puertosbolsa.ErrEmisionLlamamientoInvalida) {
		t.Fatalf("sin catálogo ningún motivo vale: %v", err)
	}
	con := &ServicioEmisionLlamamiento{motivosSalto: motivosSaltoPrueba{motivos: []puertosbolsa.MotivoSaltoOrden{{Clave: "b01.salto.x", Etiqueta: "X"}}}}
	if salto, err := con.resolverMotivoSalto(ctx, "b01.salto.x"); err != nil || salto == nil || salto.ReglaRef != "vec.bolsa.reglas:8:b01.salto.x" {
		t.Fatalf("motivo del catálogo: %+v %v", salto, err)
	}
	if _, err := con.resolverMotivoSalto(ctx, "b01.salto.y"); !errors.Is(err, puertosbolsa.ErrMotivoSaltoNoAdmitido) {
		t.Fatalf("motivo ajeno: %v", err)
	}
	caido := &ServicioEmisionLlamamiento{motivosSalto: motivosSaltoPrueba{err: errors.New("catálogo caído")}}
	if _, err := caido.resolverMotivoSalto(ctx, "b01.salto.x"); !errors.Is(err, puertosbolsa.ErrEmisionLlamamientoNoDisponible) {
		t.Fatalf("un catálogo caído no admite el salto: %v", err)
	}
}

func TestRechazoPorOrdenLlevaLosMotivosVigentes(t *testing.T) {
	ctx := t.Context()
	fuera := &puertosbolsa.LlamamientoFueraDeOrden{ParticipacionRef: "participacion:2", OrdenVigente: 2}
	s := &ServicioEmisionLlamamiento{motivosSalto: motivosSaltoPrueba{motivos: []puertosbolsa.MotivoSaltoOrden{{Clave: "b01.salto.x", Etiqueta: "X"}}}}
	var salida *puertosbolsa.LlamamientoFueraDeOrden
	if !errors.As(s.conMotivosSalto(ctx, fuera), &salida) || len(salida.Motivos) != 1 || salida.ParticipacionRef != "participacion:2" || len(fuera.Motivos) != 0 {
		t.Fatalf("motivos no añadidos o original modificado: %+v", salida)
	}
	caido := &ServicioEmisionLlamamiento{motivosSalto: motivosSaltoPrueba{err: errors.New("caído")}}
	if !errors.As(caido.conMotivosSalto(ctx, fuera), &salida) || len(salida.Motivos) != 0 {
		t.Fatalf("sin catálogo legible el rechazo sigue sin motivos: %+v", salida)
	}
	otro := errors.New("otro")
	if s.conMotivosSalto(ctx, otro) != otro {
		t.Fatal("otros errores no cambian")
	}
}
