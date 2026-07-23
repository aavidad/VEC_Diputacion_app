package postgresimportacionconvoca

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrRepositorioNoDisponible = errors.New("bolsa importacion Convoca: repositorio PostgreSQL no disponible")
	ErrLoteNoConfiable         = errors.New("bolsa importacion Convoca: lote no confiable")
	ErrIdentidadNoAutorizada   = errors.New("bolsa importacion Convoca: identidad PostgreSQL no autorizada")
)

// errorPostgreSQL no propaga mensajes, nombres de objetos, DSN ni valores
// suministrados por PostgreSQL. Los errores de contexto son la unica causa
// externa conservada.
func errorPostgreSQL(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	var errorPG *pgconn.PgError
	if errors.As(err, &errorPG) {
		switch errorPG.Code {
		case "22000", "22001", "22023", "23502", "23503", "23505", "23514", "P0001":
			return ErrLoteNoConfiable
		case "42501":
			return ErrIdentidadNoAutorizada
		}
	}
	return ErrRepositorioNoDisponible
}
