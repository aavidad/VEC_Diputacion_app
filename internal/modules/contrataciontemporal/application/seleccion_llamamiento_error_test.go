package application

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
)

type causaPrivadaSeleccionPrueba struct{}

func (*causaPrivadaSeleccionPrueba) Error() string { return "contenido reservado de la dependencia" }

func TestSeleccionIndeterminadaConservaCausaSinPublicarla(t *testing.T) {
	causa := &causaPrivadaSeleccionPrueba{}
	err := errorSeleccionIndeterminada(context.Background(), causa)
	var recuperada *causaPrivadaSeleccionPrueba
	if !errors.Is(err, ErrEjecucionSeleccionLlamamientoIndeterminada) ||
		!errors.As(err, &recuperada) || recuperada != causa {
		t.Fatal("la operación perdió su estado o su causa de diagnóstico")
	}
	var registro bytes.Buffer
	slog.New(slog.NewJSONHandler(&registro, nil)).Error("operación indeterminada", "error", err)
	if err.Error() != ErrEjecucionSeleccionLlamamientoIndeterminada.Error() ||
		strings.Contains(registro.String(), causa.Error()) {
		t.Fatal("la causa privada apareció en el texto o en el registro")
	}
}

func TestSeleccionIndeterminadaConservaCausaAlCancelar(t *testing.T) {
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	causa := errors.New("dependencia fallida antes de cancelar")
	err := errorSeleccionIndeterminada(ctx, causa)
	if !errors.Is(err, ErrEjecucionSeleccionLlamamientoIndeterminada) ||
		!errors.Is(err, context.Canceled) || !errors.Is(err, causa) {
		t.Fatal("la cancelación ocultó la causa anterior de la operación")
	}
}
