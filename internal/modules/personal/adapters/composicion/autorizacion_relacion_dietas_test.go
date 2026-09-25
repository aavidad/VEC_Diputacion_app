package composicion

import (
	"context"
	"errors"
	"testing"

	personalports "vec-diputacion-granada/internal/modules/personal/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

func TestClasificarErrorAutorizacionRelacionDietasExigeDenegacionRegistrada(t *testing.T) {
	for _, caso := range []struct {
		nombre   string
		ctx      context.Context
		err      error
		esperado error
	}{
		{"denegacion registrada", context.Background(), errors.Join(errors.New("emision no disponible"), vecports.ErrDenegacionExplicitaAutorizacionLigadaV3), personalports.ErrRelacionEmpleadoDenegada},
		{"emisor caido", context.Background(), errors.New("emisor caido"), personalports.ErrRelacionEmpleadoNoDisponible},
		{"registro de denegacion caido", context.Background(), errors.Join(vecports.ErrDenegacionExplicitaAutorizacionLigadaV3, vecports.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible), personalports.ErrRelacionEmpleadoNoDisponible},
		{"timeout mezclado", context.Background(), errors.Join(vecports.ErrDenegacionExplicitaAutorizacionLigadaV3, context.DeadlineExceeded), personalports.ErrRelacionEmpleadoNoDisponible},
		{"sin contexto", nil, vecports.ErrDenegacionExplicitaAutorizacionLigadaV3, personalports.ErrRelacionEmpleadoNoDisponible},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			got := clasificarErrorAutorizacionRelacionDietas(caso.ctx, caso.err)
			if !errors.Is(got, caso.esperado) {
				t.Fatalf("clasificación: %v", got)
			}
		})
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if got := clasificarErrorAutorizacionRelacionDietas(ctx, vecports.ErrDenegacionExplicitaAutorizacionLigadaV3); !errors.Is(got, personalports.ErrRelacionEmpleadoNoDisponible) {
		t.Fatalf("contexto cancelado: %v", got)
	}
}
