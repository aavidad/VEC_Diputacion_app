package ports

import (
	"context"
	"errors"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/domain"
)

func TestCorrelacionPeticionSoloOrigenInterno(t *testing.T) {
	ctx, cancelar := context.WithCancel(context.Background())
	defer cancelar()
	// Un consumidor no puede colocar la clave privada mediante texto libre.
	ctx = context.WithValue(ctx, "correlacion", strings.Repeat("a", 32))
	if _, ok := CorrelacionIncidenciasPeticion(ctx); ok {
		t.Fatal("admitió una clave libre")
	}
	primero, err := ConCorrelacionIncidenciasPeticion(ctx)
	if err != nil {
		t.Fatal(err)
	}
	segundo, err := ConCorrelacionIncidenciasPeticion(ctx)
	if err != nil {
		t.Fatal(err)
	}
	a, ok := CorrelacionIncidenciasPeticion(primero)
	b, otro := CorrelacionIncidenciasPeticion(segundo)
	if !ok || !otro || a == b {
		t.Fatal("correlaciones no independientes")
	}
	derivado, _ := ConMarcaIncidenciasPeticion(primero)
	if c, ok := CorrelacionIncidenciasPeticion(derivado); !ok || c != a {
		t.Fatal("derivación perdió correlación")
	}
	cancelar()
	if !errors.Is(derivado.Err(), context.Canceled) {
		t.Fatal("se perdió cancelación")
	}
	if _, ok := CorrelacionIncidenciasPeticion(nil); ok {
		t.Fatal("nil tiene correlación")
	}
}

func TestFalloCorrelacionNoReutilizaAnterior(t *testing.T) {
	anterior, err := ConCorrelacionIncidenciasPeticion(nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := conCorrelacionIncidenciasPeticion(anterior, strings.NewReader("secreto"))
	if !errors.Is(err, ErrCorrelacionIncidenciasNoDisponible) || strings.Contains(err.Error(), "secreto") {
		t.Fatalf("error no saneado: %v", err)
	}
	if c, ok := CorrelacionIncidenciasPeticion(ctx); ok || c != "" {
		t.Fatal("fallo heredó coincidencia anterior")
	}
}

type emisorContextualPuerto struct {
	emisorContextoPuerto
	contexto context.Context
}

func (e *emisorContextualPuerto) EmitirConContexto(ctx context.Context, _ domain.SolicitudIncidenciaTecnica) {
	e.contexto = ctx
}

func TestEmitirEnPeticionPrefierePuertoContextual(t *testing.T) {
	ctx, _ := ConCorrelacionIncidenciasPeticion(context.Background())
	emisor := &emisorContextualPuerto{}
	EmitirIncidenciaTecnicaEnPeticion(ctx, emisor, domain.SolicitudIncidenciaTecnica{})
	if emisor.contexto != ctx || emisor.recibidas != 0 {
		t.Fatal("no usó contrato contextual")
	}
}
