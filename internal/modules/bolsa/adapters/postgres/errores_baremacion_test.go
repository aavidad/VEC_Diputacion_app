package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

func TestErrorPostgreSQLClasificaIntegridadConcurrenciaYCancelacion(t *testing.T) {
	t.Parallel()
	casos := []struct {
		nombre   string
		codigo   string
		esperado error
	}{
		{"restriccion de integridad", "23514", puertosbolsa.ErrEvidenciaBaremacionNoConfiable},
		{"dato invalido", "22023", puertosbolsa.ErrEvidenciaBaremacionNoConfiable},
		{"estado interno divergente", "55000", puertosbolsa.ErrEvidenciaBaremacionNoConfiable},
		{"serializacion", "40001", puertosbolsa.ErrCambioBaremacionEnCurso},
		{"interbloqueo", "40P01", puertosbolsa.ErrCambioBaremacionEnCurso},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			err := errorPostgreSQL(context.Background(), &pgconn.PgError{Code: caso.codigo})
			if !errors.Is(err, caso.esperado) {
				t.Fatalf("codigo %s clasificado como %v", caso.codigo, err)
			}
		})
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if err := errorPostgreSQL(ctx, &pgconn.PgError{Code: "23514"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("la cancelacion no prevalece: %v", err)
	}
}
