package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

var (
	ErrFuentePanelInternoPostgreSQLNoDisponible = errors.New(
		"bolsa: fuente PostgreSQL del panel interno no disponible",
	)
	ErrConsultaPanelInternoPostgreSQLEnCurso = errors.New(
		"bolsa: consulta PostgreSQL del panel interno en curso",
	)
)

func errorPostgreSQLPanelInterno(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	var errorPostgreSQL *pgconn.PgError
	if errors.As(err, &errorPostgreSQL) {
		switch errorPostgreSQL.Code {
		case "40001", "40P01", "55P03", "57014":
			return ErrConsultaPanelInternoPostgreSQLEnCurso
		case "22000", "22023", "23503", "23514", "55000":
			return puertosbolsa.ErrResultadoPanelInternoInvalido
		}
	}
	return ErrFuentePanelInternoPostgreSQLNoDisponible
}
