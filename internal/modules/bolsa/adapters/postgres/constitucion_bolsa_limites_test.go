package postgres

import (
	"context"
	"errors"
	"math"
	"testing"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

func TestVersionConstitucionRespetaBigintPostgreSQL(t *testing.T) {
	ultimo, err := versionBigintConstitucion(math.MaxInt64)
	if err != nil || ultimo != math.MaxInt64 {
		t.Fatalf("máximo bigint rechazado: %d %v", ultimo, err)
	}
	if _, err := versionBigintConstitucion(uint64(math.MaxInt64) + 1); !errors.Is(err, ports.ErrConstitucionBolsaInvalida) {
		t.Fatalf("versión fuera de bigint admitida: %v", err)
	}
}

func TestBolsaConstituidaVigenteFallaCerradaSinDependencias(t *testing.T) {
	var r *RepositorioConstitucionPostgreSQL
	if vigente, err := r.BolsaConstituidaVigente(context.Background(), "bolsa:x"); vigente || !errors.Is(err, ports.ErrConstitucionBolsaNoDisponible) {
		t.Fatalf("sin pool: %v %v", vigente, err)
	}
	if vigente, err := (&RepositorioConstitucionPostgreSQL{}).BolsaConstituidaVigente(context.Background(), ""); vigente || !errors.Is(err, ports.ErrConstitucionBolsaNoDisponible) {
		t.Fatalf("referencia vacía: %v %v", vigente, err)
	}
}
