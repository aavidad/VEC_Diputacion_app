package bootstrap

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
	contextopg "vec-diputacion-granada/internal/vec/adapters/contextoactor/postgres"
	vecpg "vec-diputacion-granada/internal/vec/adapters/postgres"
	"vec-diputacion-granada/internal/vec/adapters/seguridad"
	vecapp "vec-diputacion-granada/internal/vec/application"
)

// fronteraPreferenciasUsuarios mantiene nominales las autoridades de cada proceso.
// La ruta y sus casos de uso se reutilizan; los pools y adaptadores no se comparten.
type fronteraPreferenciasUsuarios struct {
	roles       [6]string
	abrirPool   func(context.Context, string, string) (*pgxpool.Pool, string, error)
	contextos   func(context.Context, *pgxpool.Pool) (*vecapp.AutoridadContextoActorRegistradoV2, error)
	autorizador func(context.Context, *pgxpool.Pool, *pgxpool.Pool, *pgxpool.Pool, string) (*vecapp.ServicioAutorizacionSolicitudLigadaV3, error)
}

func fronteraPreferenciasUsuariosCombinada() fronteraPreferenciasUsuarios {
	return fronteraPreferenciasUsuarios{
		roles:     [6]string{"vec_identidad_sesiones_v1_registrador", "vec_identidad_sesiones_v1_revalidador", "vec_contexto_actor_v1_runtime", "vec_autorizacion_fuente", "vec_autorizacion_registro", "vec_autorizacion_motivos_evaluador"},
		abrirPool: abrirPoolRutasDietas,
		contextos: func(ctx context.Context, pool *pgxpool.Pool) (*vecapp.AutoridadContextoActorRegistradoV2, error) {
			resolutor, err := contextopg.NuevoResolutorRegistroContextoActorPostgreSQLV2(ctx, pool)
			if err != nil {
				return nil, err
			}
			servicio, err := vecapp.NuevoServicioContextoActorProductivoV2(resolutor, contextopg.NuevoGeneradorOperacionContextoActorV2Criptografico(), relojRutasDietas{})
			if err != nil {
				return nil, err
			}
			return vecapp.NuevaAutoridadContextoActorRegistradoV2(servicio)
		},
		autorizador: func(ctx context.Context, fuentePool, registroPool, motivosPool *pgxpool.Pool, catalogoID string) (*vecapp.ServicioAutorizacionSolicitudLigadaV3, error) {
			fuente, err := vecpg.NuevoAlmacenAutorizacion(fuentePool)
			if err != nil {
				return nil, err
			}
			registro, err := vecpg.NuevoAlmacenAutorizacion(registroPool)
			if err != nil {
				return nil, err
			}
			motivos, err := vecpg.NuevoValidadorReferenciaMotivoPostgreSQLV2(motivosPool, catalogoID)
			if err != nil {
				return nil, err
			}
			return vecapp.NuevoServicioAutorizacionSolicitudLigadaV3(fuente, registro, registro, motivos, relojRutasDietas{}, seguridad.GeneradorReferenciasCriptograficas{}, vecapp.ConfiguracionServicioAutorizacion{VigenciaDecision: 30 * time.Second})
		},
	}
}
