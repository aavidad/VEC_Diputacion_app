package postgres

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestConsultaReciboRespuestaNoTraduceConflictosComoDenegacion(t *testing.T) {
	for _, codigo := range []string{"P1394", "40001", "40P01", "55P03", "57014"} {
		got := normalizarErrorConsultaReciboRespuesta(context.Background(), &pgconn.PgError{Code: codigo})
		if got != ports.ErrConsultaReciboRespuestaFallo {
			t.Fatalf("SQLSTATE %s: %v", codigo, got)
		}
	}
	for _, codigo := range []string{"P1393", "42501"} {
		got := normalizarErrorConsultaReciboRespuesta(context.Background(), &pgconn.PgError{Code: codigo})
		if got != ports.ErrConsultaReciboRespuestaDenegada {
			t.Fatalf("SQLSTATE %s: %v", codigo, got)
		}
	}
}
