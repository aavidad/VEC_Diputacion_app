package postgres

import (
	"context"
	"crypto/rand"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/vec/ports"
)

// Las funciones y la familia se fijan al construir el adaptador. Ninguna
// petición puede elegir el resolutor general ni la población candidata.
var (
	consultaResolverContextoUsuariosExternoV1 = strings.Replace(consultaResolverContextoCandidatoExternoV1,
		"resolver_contexto_candidato_externo_v1", "resolver_contexto_usuarios_externo_v1", 1)
	consultaReconciliarContextoUsuariosExternoV1 = strings.Replace(consultaReconciliarContextoCandidatoExternoV1,
		"reconciliar_contexto_candidato_externo_v1", "reconciliar_contexto_usuarios_externo_v1", 1)
)

// NuevoResolutorRegistroContextoUsuariosExternoPostgreSQLV1 usa la población
// externa propia de Usuarios y exige el grupo nominal de CTX15.
func NuevoResolutorRegistroContextoUsuariosExternoPostgreSQLV1(ctx context.Context,
	pool *pgxpool.Pool,
) (*ResolutorRegistroContextoActorExternoPostgreSQLV1, error) {
	if ctx == nil || pool == nil || ctx.Err() != nil {
		return nil, ports.ErrResolutorRegistroContextoActorNoDisponible
	}
	var login string
	var acreditada bool
	if err := pool.QueryRow(ctx, `SELECT identidad_login, acreditada
	 FROM vec_contexto_actor_v1.acreditar_runtime_usuarios_externo_v1()`).Scan(&login, &acreditada); err != nil || !acreditada || login == "" {
		return nil, errorResolutorContextoActorPostgreSQL(ctx)
	}
	r, err := nuevoResolutorRegistroContextoActorExternoPostgreSQLV1(pool, rand.Reader)
	if err != nil {
		return nil, err
	}
	r.usuarios = true
	return r, nil
}

func confirmarRespuestaUsuariosExterno(solicitud ports.SolicitudResolucionRegistroContextoActorV2,
	respuesta respuestaContextoActorPostgreSQL,
) (ports.ConfirmacionRegistroContextoActorV2, error) {
	confirmacion, err := confirmarRespuestaContextoActor(solicitud, respuesta)
	if err != nil || len(confirmacion.Contexto.Instantanea.Vinculos) != 0 {
		return ports.ConfirmacionRegistroContextoActorV2{}, ports.ErrResolutorRegistroContextoActorNoDisponible
	}
	return confirmacion, nil
}
