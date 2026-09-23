package postgres

import (
	"context"
	"errors"
	"testing"

	"vec-diputacion-granada/internal/modules/cronos/ports"
)

func TestRegistroResultadoEjecucionFallaCerradoSinPoolNiDatos(t *testing.T) {
	if _, err := NuevoRegistroResultadoEjecucionMarcajePostgreSQL(nil); !errors.Is(err, ports.ErrAuditoriaMarcajeNoDisponible) {
		t.Fatalf("sin pool: %v", err)
	}
	var r *RegistroResultadoEjecucionMarcajePostgreSQL
	if err := r.RegistrarResultadoEjecucionMarcaje(context.Background(), ports.ResultadoEjecucionMarcaje{}); !errors.Is(err, ports.ErrAuditoriaMarcajeNoDisponible) {
		t.Fatalf("registro nulo: %v", err)
	}
}
