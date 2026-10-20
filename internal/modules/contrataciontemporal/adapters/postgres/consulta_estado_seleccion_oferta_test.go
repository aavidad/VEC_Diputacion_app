package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestConsultaEstadoSeleccionOfertaConflictoSQLNoPareceAusencia(t *testing.T) {
	for _, codigo := range []string{"23505", "55000", "40001", "40P01"} {
		err := errorRegistroSeleccionOferta(context.Background(), &pgconn.PgError{Code: codigo})
		if !errors.Is(err, ports.ErrVinculoOfertaBolsaCTInvalido) {
			t.Fatalf("SQLSTATE %s no conserva conflicto: %v", codigo, err)
		}
	}
}
