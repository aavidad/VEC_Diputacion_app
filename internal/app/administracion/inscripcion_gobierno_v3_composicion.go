package administracion

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	postgres "vec-diputacion-granada/internal/vec/adapters/postgres"
	"vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// Cada capacidad usa su LOGIN propio: el pool de gobierno sólo tiene EXECUTE
// en las dos fachadas AUT68, y el catálogo se lee por el pool AUT58 existente.
type DependenciasGobiernoInscripcionV3 struct {
	PoolGobierno    *pgxpool.Pool
	FuenteCatalogo  ports.FuenteCatalogoAccionesAdministracionV1
	Confianza       ConfiguracionConfianzaPerfilesV3
	DependenciasPDP DependenciasConfianzaPerfilesV3
	Motivos         map[string]domain.ReferenciaEntradaCatalogo
	Reloj           ports.Reloj
}

func NuevoServicioGobiernoInscripcionV3(ctx context.Context,
	d DependenciasGobiernoInscripcionV3) (*application.ServicioGobiernoInscripcion, error) {
	if ctx == nil || d.PoolGobierno == nil || dependenciaConfianzaPerfilesNula(d.FuenteCatalogo) ||
		dependenciaConfianzaPerfilesNula(d.Reloj) ||
		d.PoolGobierno == d.DependenciasPDP.PoolFuente ||
		d.PoolGobierno == d.DependenciasPDP.PoolRegistro ||
		d.PoolGobierno == d.DependenciasPDP.PoolMotivos {
		return nil, ErrConfiguracion
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	confianza, err := NuevaConfianzaGobiernoInscripcionV3(d.Confianza, d.DependenciasPDP)
	if err != nil {
		return nil, ErrConfiguracion
	}
	emisor, err := NuevoEmisorGobiernoInscripcionV3(confianza.Emisores, d.Motivos, d.Reloj)
	if err != nil {
		return nil, ErrConfiguracion
	}
	autoridad, err := postgres.NuevaAutoridadGobiernoInscripcionPostgreSQL(ctx, d.PoolGobierno, d.FuenteCatalogo, emisor, d.Reloj)
	if err != nil {
		return nil, err
	}
	return application.NuevoServicioGobiernoInscripcion(autoridad, d.Reloj)
}
