package internagobierno

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	postgrescontexto "vec-diputacion-granada/internal/vec/adapters/contextoactor/postgres"
	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	postgresidentidad "vec-diputacion-granada/internal/vec/adapters/httpseguridad/postgres"
	postgresvec "vec-diputacion-granada/internal/vec/adapters/postgres"
	appvec "vec-diputacion-granada/internal/vec/application"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// ContextoPostgreSQL conserva sólo adaptadores de lectura/registro de F1.
// Los pools pertenecen al llamador, que los cierra con el resto del runtime.
type ContextoPostgreSQL struct {
	Revalidador        vecports.RevalidadorAutenticacionActorV1
	Resolutor          core.ResolutorContextoActorRegistradoV2
	VinculoCorporativo vecports.RevalidadorVinculoCorporativoRRHHV1
}

type IdentidadPostgreSQL struct {
	Registro    httpseguridad.RegistroSesiones
	Revalidador vecports.RevalidadorAutenticacionActorV1
}

// NuevoIdentidadPostgreSQL acredita dos LOGIN distintos y su capacidad
// nominal. Seudonimizador procede del módulo criptográfico externo gobernado;
// este constructor no genera ni almacena claves.
func NuevoIdentidadPostgreSQL(ctx context.Context, registro, revalidacion *pgxpool.Pool,
	seudonimizador postgresidentidad.SeudonimizadorAlta, espacioIdentidad, dominioHMACRef string,
) (IdentidadPostgreSQL, error) {
	if ctx == nil || ctx.Err() != nil || registro == nil || revalidacion == nil || registro == revalidacion || seudonimizador == nil {
		return IdentidadPostgreSQL{}, ErrGobiernoInternoNoDisponible
	}
	sesiones, err := postgresidentidad.NuevoRegistroSesionesPostgreSQL(ctx, registro, revalidacion, seudonimizador, espacioIdentidad, dominioHMACRef)
	if err != nil {
		return IdentidadPostgreSQL{}, ErrGobiernoInternoNoDisponible
	}
	revalidador, err := postgresidentidad.NuevoRevalidadorAutenticacionActorPostgreSQL(ctx, revalidacion)
	if err != nil {
		return IdentidadPostgreSQL{}, ErrGobiernoInternoNoDisponible
	}
	return IdentidadPostgreSQL{Registro: sesiones, Revalidador: revalidador}, nil
}

// NuevoContextoPostgreSQL acredita los roles nominales con los propios
// constructores de Identidad y F1. No escribe personas ni sesiones al arrancar.
func NuevoContextoPostgreSQL(ctx context.Context, revalidacion, contexto *pgxpool.Pool, reloj vecports.Reloj) (ContextoPostgreSQL, error) {
	if ctx == nil || ctx.Err() != nil || revalidacion == nil || contexto == nil || revalidacion == contexto || reloj == nil {
		return ContextoPostgreSQL{}, ErrGobiernoInternoNoDisponible
	}
	r, err := postgresidentidad.NuevoRevalidadorAutenticacionActorPostgreSQL(ctx, revalidacion)
	if err != nil {
		return ContextoPostgreSQL{}, ErrGobiernoInternoNoDisponible
	}
	f, err := postgrescontexto.NuevoResolutorRegistroContextoActorPostgreSQLV2(ctx, contexto)
	if err != nil {
		return ContextoPostgreSQL{}, ErrGobiernoInternoNoDisponible
	}
	servicio, err := appvec.NuevoServicioContextoActorProductivoV2(f, postgrescontexto.NuevoGeneradorOperacionContextoActorV2Criptografico(), reloj)
	if err != nil {
		return ContextoPostgreSQL{}, ErrGobiernoInternoNoDisponible
	}
	autoridad, err := appvec.NuevaAutoridadContextoActorRegistradoV2(servicio)
	if err != nil {
		return ContextoPostgreSQL{}, ErrGobiernoInternoNoDisponible
	}
	// Mismo LOGIN runtime de ContextoActor; sin 000008 no arranca.
	corporativo, err := postgrescontexto.NuevoRevalidadorVinculoCorporativoRRHHPostgreSQLV1(ctx, contexto)
	if err != nil {
		return ContextoPostgreSQL{}, ErrGobiernoInternoNoDisponible
	}
	return ContextoPostgreSQL{Revalidador: r, Resolutor: autoridad, VinculoCorporativo: corporativo}, nil
}

// NuevoRegistradorAuditoriaPostgreSQL exige EXECUTE sobre la función exacta
// antes de que la aplicación empiece a atender peticiones.
func NuevoRegistradorAuditoriaPostgreSQL(ctx context.Context, pool *pgxpool.Pool) (vecports.RegistradorAuditoriaFronteraRutaExacta, error) {
	if ctx == nil || ctx.Err() != nil || pool == nil {
		return nil, ErrGobiernoInternoNoDisponible
	}
	r, err := postgresvec.NuevoRegistradorAuditoriaFronteraRutaExactaPostgreSQL(pool)
	if err != nil || r.PreflightAuditoriaFronteraRutaExacta(ctx) != nil {
		return nil, ErrGobiernoInternoNoDisponible
	}
	return r, nil
}
