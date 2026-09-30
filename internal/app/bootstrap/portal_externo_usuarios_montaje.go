package bootstrap

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	contextopg "vec-diputacion-granada/internal/vec/adapters/contextoactor/postgres"
	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	identidadpg "vec-diputacion-granada/internal/vec/adapters/httpseguridad/postgres"
	vecapp "vec-diputacion-granada/internal/vec/application"
	core "vec-diputacion-granada/internal/vec/domain"
)

// fronteraPreferenciasUsuariosPortalExterno nunca construye adaptadores de
// autorización ni ContextoActor del proceso interno. Las mismas rutas Usuarios
// consumen exclusivamente sus fachadas nominales y datos propios del externo.
func fronteraPreferenciasUsuariosPortalExterno(topologia topologiaPostgreSQLPreferenciasUsuarios) fronteraPreferenciasUsuarios {
	return fronteraPreferenciasUsuarios{
		roles: [6]string{"vec_identidad_externa_v1_registrador", "vec_identidad_externa_v1_revalidador", "vec_contexto_actor_v1_usuarios_externo", rolFuenteUsuariosExterno, rolRegistroUsuariosExterno, rolMotivosUsuariosExterno},
		abrirPool: func(ctx context.Context, dsn, rol string) (*pgxpool.Pool, string, error) {
			logins := map[string]string{
				rolFuenteUsuariosExterno:   loginFuenteUsuariosExterno,
				rolRegistroUsuariosExterno: loginRegistroUsuariosExterno,
				rolMotivosUsuariosExterno:  loginMotivosUsuariosExterno,
			}
			if login, nominal := logins[rol]; nominal {
				pool, err := abrirPoolAutorizacionUsuariosExterno(ctx, dsn, rol, login, topologia)
				return pool, login, err
			}
			if rol != "vec_identidad_externa_v1_registrador" && rol != "vec_identidad_externa_v1_revalidador" && rol != "vec_contexto_actor_v1_usuarios_externo" {
				return nil, "", ErrUsuariosPortalExternoNoDisponible
			}
			pool, login, err := abrirPoolMiBolsaPortalExterno(ctx, dsn, rol)
			if err != nil {
				return nil, "", ErrUsuariosPortalExternoNoDisponible
			}
			if err := cotejarTopologiaPostgreSQLPreferenciasUsuarios(ctx, pool, topologia); err != nil {
				pool.Close()
				return nil, "", ErrUsuariosPortalExternoNoDisponible
			}
			return pool, login, nil
		},
		identidad: func(ctx context.Context, registroPool, revalidacionPool *pgxpool.Pool, derivador *derivadorIdentidadOperacionDesarrollo) (httpseguridad.RegistroSesiones, core.RevalidadorAutenticacionActorV1, error) {
			registro, err := identidadpg.NuevoRegistroSesionesExternoPostgreSQL(ctx, registroPool, revalidacionPool, &seudonimizadorSesionDesarrollo{derivador: derivador}, espacioIdentidadSesionDesarrollo, dominioIdentidadSesionDesarrollo)
			if err != nil {
				return nil, nil, ErrUsuariosPortalExternoNoDisponible
			}
			revalidador, err := identidadpg.NuevoRevalidadorAutenticacionActorExternoPostgreSQL(ctx, revalidacionPool)
			if err != nil {
				return nil, nil, ErrUsuariosPortalExternoNoDisponible
			}
			return registro, revalidador, nil
		},
		contextos: func(ctx context.Context, pool *pgxpool.Pool) (*vecapp.AutoridadContextoActorRegistradoV2, error) {
			resolutor, err := contextopg.NuevoResolutorRegistroContextoUsuariosExternoPostgreSQLV1(ctx, pool)
			if err != nil {
				return nil, ErrUsuariosPortalExternoNoDisponible
			}
			servicio, err := vecapp.NuevoServicioContextoActorProductivoV2(resolutor, contextopg.NuevoGeneradorOperacionContextoActorV2Criptografico(), relojRutasDietas{})
			if err != nil {
				return nil, ErrUsuariosPortalExternoNoDisponible
			}
			return vecapp.NuevaAutoridadContextoActorRegistradoV2(servicio)
		},
		autorizador: nuevoServicioAutorizacionUsuariosExterno,
	}
}
