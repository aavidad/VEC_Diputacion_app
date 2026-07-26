package postgresimportacionconvoca

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"

	aplicacion "vec-diputacion-granada/internal/modules/bolsa/application/importacionconvoca"
)

var (
	ErrRepositorioNoDisponible = errors.New("bolsa importacion Convoca: repositorio PostgreSQL no disponible")
	ErrLoteNoConfiable         = errors.New("bolsa importacion Convoca: lote no confiable")
	ErrIdentidadNoAutorizada   = errors.New("bolsa importacion Convoca: identidad PostgreSQL no autorizada")
	ErrResultadoNoConfiable    = errors.New("bolsa importacion Convoca: resultado PostgreSQL no confiable")
)

func errorPostgreSQL(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	var errorPG *pgconn.PgError
	if errors.As(err, &errorPG) {
		switch errorPG.Code {
		case "B1701", "22000", "22001", "22003", "22023", "23502", "23503", "23505", "23514":
			return ErrLoteNoConfiable
		case "B1702":
			return aplicacion.ErrConciliacionEnConflicto
		case "B1703":
			return aplicacion.ErrRetencionEnConflicto
		case "B1704":
			return aplicacion.ErrImportacionNoEncontrada
		case "B1705":
			return aplicacion.ErrStagingExpurgado
		case "B1706":
			return aplicacion.ErrImportacionEnConflicto
		case "42501":
			return ErrIdentidadNoAutorizada
		}
	}
	return ErrRepositorioNoDisponible
}

func esReintentable(err error) bool {
	var errorPG *pgconn.PgError
	return errors.As(err, &errorPG) &&
		(errorPG.Code == "40001" || errorPG.Code == "40P01")
}
