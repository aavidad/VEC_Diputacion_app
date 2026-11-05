package ports

import (
	"context"
	"errors"
	"testing"
)

func TestResolucionDocumentoMismosDatosAntesYDespuesDelActo(t *testing.T) {
	s := solicitudResolucionPrueba()
	ctx, err := ConResolucionDocumentoRRHH(context.Background(), s)
	if err != nil {
		t.Fatal(err)
	}
	antes, err := ResolucionParaDocumentoRRHH(ctx, s.ExpedienteRef, func(context.Context, string) (*ResultadoResolucionFormalizacion, error) {
		t.Fatal("el renderizado previo no relee un acto todavía ausente")
		return nil, nil
	})
	if err != nil || antes == nil || antes.Solicitud != s {
		t.Fatalf("solicitud perdida: %v", err)
	}
	despues, err := ResolucionParaDocumentoRRHH(context.Background(), s.ExpedienteRef, func(context.Context, string) (*ResultadoResolucionFormalizacion, error) {
		r := reciboResolucionPrueba(s)
		return &r, nil
	})
	if err != nil || despues == nil || despues.Solicitud != antes.Solicitud {
		t.Fatalf("recuperación distinta: %v", err)
	}
	if _, err := ResolucionParaDocumentoRRHH(ctx, "expediente:otro", nil); !errors.Is(err, ErrResultadoResolucionFormalizacionNoConfiable) {
		t.Fatal("cruce de expediente aceptado", err)
	}
	s.NumeroResolucion = ""
	if _, err := ConResolucionDocumentoRRHH(context.Background(), s); err == nil {
		t.Fatal("solicitud inválida admitida")
	}
}

func TestResolucionDocumentoAusenciaCancelacionYFalloNoSeInventan(t *testing.T) {
	if dato, err := ResolucionParaDocumentoRRHH(context.Background(), "expediente:prueba", nil); dato != nil || err != nil {
		t.Fatal("se inventó resolución", err)
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err := ResolucionParaDocumentoRRHH(ctx, "expediente:prueba", nil); !errors.Is(err, context.Canceled) {
		t.Fatal("cancelación perdida", err)
	}
	if _, err := ResolucionParaDocumentoRRHH(context.Background(), "expediente:prueba", func(context.Context, string) (*ResultadoResolucionFormalizacion, error) {
		return nil, ErrAutorizacionDenegada
	}); !errors.Is(err, ErrAutorizacionDenegada) {
		t.Fatal("fallo lector ocultado", err)
	}
}
