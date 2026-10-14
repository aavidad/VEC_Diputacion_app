package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
)

func TestLiquidacionEconomicaErroresSQLNoRepitenEfectosAjenos(t *testing.T) {
	casos := []struct {
		codigo  string
		repetir bool
		causa   error
	}{
		{"40001", true, nil},
		{"40P01", true, nil},
		{"PD002", false, dietasports.ErrConflictoIdempotencia},
		{"PD003", false, dietasports.ErrAccesoCircuitoDenegado},
		{"PD004", false, dietasports.ErrComisionNoEncontrada},
		{"PD005", false, dietasports.ErrEstadoCircuitoConflicto},
		{"P7201", false, dietasports.ErrRelacionNoDisponible},
		{"XX000", false, dietasports.ErrResultadoCircuitoIncierto},
	}
	for _, caso := range casos {
		t.Run(caso.codigo, func(t *testing.T) {
			repetir, err := errorReintentoLiquidacion(context.Background(), &pgconn.PgError{Code: caso.codigo})
			if repetir != caso.repetir || !errors.Is(err, caso.causa) {
				t.Fatalf("repetir=%v, err=%v", repetir, err)
			}
		})
	}
}

func TestLecturaLiquidacionNoDistingueAusenciaDeDenegacion(t *testing.T) {
	for _, codigo := range []string{"PD004", "42501"} {
		repetir, err := errorReintentoLecturaLiquidacion(context.Background(), &pgconn.PgError{Code: codigo})
		if repetir || !errors.Is(err, dietasports.ErrAccesoCircuitoDenegado) {
			t.Fatalf("codigo=%s, repetir=%v, err=%v", codigo, repetir, err)
		}
	}
}
