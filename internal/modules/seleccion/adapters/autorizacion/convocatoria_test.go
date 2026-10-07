package autorizacion

import (
	"context"
	"errors"
	"testing"

	"vec-diputacion-granada/internal/modules/seleccion/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

func TestSoloDenegacionCentralRegistradaEsDenegada(t *testing.T) {
	denegada := vecports.ErrDenegacionExplicitaAutorizacionLigadaV3
	if clasificarError(context.Background(), denegada) != ports.ErrConsultaConvocatoriaDenegada {
		t.Fatal("denegación nominal no conservada")
	}
	for _, tecnica := range []error{vecports.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible, vecports.ErrFuenteAutorizacionNoDisponible, vecports.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible} {
		if clasificarError(context.Background(), errors.Join(denegada, tecnica)) != ports.ErrConvocatoriaNoDisponible {
			t.Fatal("fallo técnico tratado como denegación durable")
		}
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if clasificarError(ctx, denegada) != context.Canceled {
		t.Fatal("cancelación oculta")
	}
}
