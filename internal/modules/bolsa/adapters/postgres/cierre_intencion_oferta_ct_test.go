package postgres

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

func TestCierreIntencionOfertaCTConflictoFisicoNoEsAusencia(t *testing.T) {
	t.Parallel()
	for _, codigo := range []string{"23505", "40001", "40P01", "55P03"} {
		t.Run(codigo, func(t *testing.T) {
			err := errorResolucionIntencionOfertaCT(t.Context(), &pgconn.PgError{Code: codigo})
			if !errors.Is(err, ports.ErrCierreIntencionOfertaCTIncierta) {
				t.Fatalf("carrera %s interpretada como resultado cerrado: %v", codigo, err)
			}
		})
	}
}

func TestCierreIntencionOfertaCTRechazaDosResultados(t *testing.T) {
	t.Parallel()
	_, err := decodificarResolucionIntencionOfertaCT("cerrada",
		[]byte(`{"oferta_ref":"oferta:falsa"}`),
		[]byte(`{"esquema":"vec.bolsa.oferta-ct.cierre.v1","estado":"cerrada_sin_publicar"}`))
	if !errors.Is(err, ports.ErrCierreIntencionOfertaCTNoDisponible) {
		t.Fatalf("oferta y cierre simultáneos aceptados: %v", err)
	}
}
