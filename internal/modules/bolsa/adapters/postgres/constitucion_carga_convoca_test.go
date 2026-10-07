package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

func TestConstituirCargaConvocaAutorizadaFallaCerradaSinDependencias(t *testing.T) {
	var r *RepositorioConstitucionPostgreSQL
	if _, err := r.ConstituirCargaConvocaAutorizada(context.Background(), ports.Constitucion{}, puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{}); !errors.Is(err, ports.ErrConstitucionBolsaNoDisponible) {
		t.Fatalf("repositorio nulo: %v", err)
	}
	r = &RepositorioConstitucionPostgreSQL{}
	if _, err := r.ConstituirCargaConvocaAutorizada(context.Background(), ports.Constitucion{}, puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{}); !errors.Is(err, ports.ErrConstitucionBolsaNoDisponible) {
		t.Fatalf("sin pool ni material: %v", err)
	}
}

func TestErrorConstitucionCargaConvocaSeparaDenegacion(t *testing.T) {
	ctx := context.Background()
	if err := errorConstitucionCargaConvoca(ctx, &pgconn.PgError{Code: "42501"}); !errors.Is(err, dominiovec.ErrAutorizacionDenegada) {
		t.Fatalf("42501: %v", err)
	}
	if err := errorConstitucionCargaConvoca(ctx, &pgconn.PgError{Code: "VA172"}); !errors.Is(err, ports.ErrConstitucionBolsaNoDisponible) {
		t.Fatalf("VA172 debe ser fallo técnico, no denegación: %v", err)
	}
	if err := errorConstitucionCargaConvoca(ctx, &pgconn.PgError{Code: "23505"}); !errors.Is(err, ports.ErrConstitucionBolsaEnConflicto) {
		t.Fatalf("23505: %v", err)
	}
	if err := errorConstitucionCargaConvoca(ctx, errors.New("red")); !errors.Is(err, ports.ErrConstitucionBolsaNoDisponible) {
		t.Fatalf("otro: %v", err)
	}
	cancelado, cancelar := context.WithCancel(ctx)
	cancelar()
	if err := errorConstitucionCargaConvoca(cancelado, &pgconn.PgError{Code: "42501"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelado: %v", err)
	}
}
