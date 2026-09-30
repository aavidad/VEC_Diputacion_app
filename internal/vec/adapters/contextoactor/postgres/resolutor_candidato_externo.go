package postgres

import (
	"context"
	"crypto/rand"
	"errors"
	"io"

	"github.com/jackc/pgx/v5/pgxpool"

	postgresqlcomun "vec-diputacion-granada/internal/shared/postgresql"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const (
	consultaResolverContextoCandidatoExternoV1 = `
		SELECT operacion_ref, registro_contexto_ref,
		       representacion_canonica, huella_sha256,
		       manifiesto_procedencia_canonico,
		       manifiesto_procedencia_huella_sha256,
		       autoridad_efectiva, resuelto_en
		  FROM vec_contexto_actor_v1.resolver_contexto_candidato_externo_v1(
		       $1, $2, $3, $4, $5)`
	consultaReconciliarContextoCandidatoExternoV1 = `
		SELECT operacion_ref, registro_contexto_ref,
		       representacion_canonica, huella_sha256,
		       manifiesto_procedencia_canonico,
		       manifiesto_procedencia_huella_sha256,
		       autoridad_efectiva, resuelto_en
		  FROM vec_contexto_actor_v1.reconciliar_contexto_candidato_externo_v1(
		       $1, $2, $3, $4, $5)`
)

// ResolutorRegistroContextoActorExternoPostgreSQLV1 usa exclusivamente las
// funciones SQL del candidato externo. Su LOGIN se acredita al construirlo.
type ResolutorRegistroContextoActorExternoPostgreSQLV1 struct {
	base     *ResolutorRegistroContextoActorPostgreSQLV2
	usuarios bool
}

func NuevoResolutorRegistroContextoActorExternoPostgreSQLV1(
	ctx context.Context,
	pool *pgxpool.Pool,
) (*ResolutorRegistroContextoActorExternoPostgreSQLV1, error) {
	if ctx == nil || pool == nil {
		return nil, ports.ErrResolutorRegistroContextoActorNoDisponible
	}
	var identidad string
	var acreditada bool
	if err := pool.QueryRow(ctx, `
		SELECT identidad_login, acreditada
		  FROM vec_contexto_actor_v1.acreditar_runtime_candidato_externo_v1()`,
	).Scan(&identidad, &acreditada); err != nil || !acreditada || identidad == "" {
		return nil, errorResolutorContextoActorPostgreSQL(ctx)
	}
	return nuevoResolutorRegistroContextoActorExternoPostgreSQLV1(pool, rand.Reader)
}

func nuevoResolutorRegistroContextoActorExternoPostgreSQLV1(
	pool iniciadorContextoActorPostgreSQL,
	aleatorio io.Reader,
) (*ResolutorRegistroContextoActorExternoPostgreSQLV1, error) {
	base, err := nuevoResolutorRegistroContextoActorPostgreSQLV2(pool, aleatorio)
	if err != nil {
		return nil, err
	}
	return &ResolutorRegistroContextoActorExternoPostgreSQLV1{base: base}, nil
}

func (r *ResolutorRegistroContextoActorExternoPostgreSQLV1) ResolverYRegistrarContextoActorV2(
	ctx context.Context,
	solicitud ports.SolicitudResolucionRegistroContextoActorV2,
) (ports.ConfirmacionRegistroContextoActorV2, error) {
	if ctx == nil || r == nil || r.base == nil ||
		valorNuloContextoActorPostgreSQL(r.base.pool) ||
		valorNuloContextoActorPostgreSQL(r.base.aleatorio) ||
		solicitud.Validar() != nil || !solicitud.Proyecciones.Vacio() ||
		solicitud.Contexto.Cuenta.Metodo != domain.AuthMethodCertificate ||
		solicitud.Contexto.Cuenta.Garantia != domain.AuthAssuranceHigh {
		return ports.ConfirmacionRegistroContextoActorV2{},
			ports.ErrResolutorRegistroContextoActorNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return ports.ConfirmacionRegistroContextoActorV2{}, err
	}
	reciboRef, err := nuevaReferenciaContextoActorV2(
		ctx, r.base.aleatorio, "rca_", ports.ErrResolutorRegistroContextoActorNoDisponible,
	)
	if err != nil {
		return ports.ConfirmacionRegistroContextoActorV2{}, errorResolutorContextoActorPostgreSQL(ctx)
	}
	argumentos := []any{
		solicitud.OperacionRef, reciboRef, solicitud.Contexto.Cuenta.CuentaRef,
		solicitud.Contexto.PerfilActivoRef, solicitud.SolicitadoEn,
	}
	resolver, reconciliar := consultaResolverContextoCandidatoExternoV1, consultaReconciliarContextoCandidatoExternoV1
	if r.usuarios {
		resolver, reconciliar = consultaResolverContextoUsuariosExternoV1, consultaReconciliarContextoUsuariosExternoV1
	}
	confirmar := func(respuesta respuestaContextoActorPostgreSQL) (ports.ConfirmacionRegistroContextoActorV2, error) {
		if r.usuarios {
			return confirmarRespuestaUsuariosExterno(solicitud, respuesta)
		}
		return confirmarRespuestaCandidatoExterno(solicitud, respuesta)
	}
	// Los abortos serializables siguen la política común. Cada intento abre
	// otra transacción y conserva la operación y sus argumentos; un COMMIT
	// incierto exige antes reconciliar y confirmar ausencia.
	repetidoTrasIncierto := false
	for intento := 1; intento <= postgresqlcomun.IntentosMaximosCarreraSerializable; intento++ {
		if intento > 1 && !postgresqlcomun.EsperarReintentoCarreraSerializable(ctx, intento-1) {
			return ports.ConfirmacionRegistroContextoActorV2{}, errorResolutorContextoActorPostgreSQL(ctx)
		}
		respuesta, estado, denegacion := r.base.ejecutarConClasificador(ctx, resolver, argumentos, postgresqlcomun.EsCarreraSerializable)
		if estado == estadoContextoActorConfirmado {
			return confirmar(respuesta)
		}
		if estado == estadoContextoActorDenegado {
			return ports.ConfirmacionRegistroContextoActorV2{}, errors.Join(
				ports.ErrResolutorRegistroContextoActorNoDisponible, denegacion,
			)
		}
		if estado == estadoContextoActorReintentable {
			continue
		}
		if estado != estadoContextoActorCommitIncierto {
			return ports.ConfirmacionRegistroContextoActorV2{}, errorResolutorContextoActorPostgreSQL(ctx)
		}
		reconciliada, estadoReconciliacion := r.base.reconciliar(ctx, reconciliar, argumentos)
		switch estadoReconciliacion {
		case estadoContextoActorConfirmado:
			if !respuestasContextoActorIguales(respuesta, reconciliada) {
				return ports.ConfirmacionRegistroContextoActorV2{}, ports.ErrResolutorRegistroContextoActorNoDisponible
			}
			return confirmar(reconciliada)
		case estadoContextoActorAusente:
			// Solo la ausencia confirmada permite el único reintento histórico
			// tras un COMMIT incierto, conservando la misma operación y recibo.
			if repetidoTrasIncierto {
				return ports.ConfirmacionRegistroContextoActorV2{}, errorResolutorContextoActorPostgreSQL(ctx)
			}
			repetidoTrasIncierto = true
			continue
		default:
			return ports.ConfirmacionRegistroContextoActorV2{}, errorResolutorContextoActorPostgreSQL(ctx)
		}
	}
	return ports.ConfirmacionRegistroContextoActorV2{}, ports.ErrResolutorRegistroContextoActorNoDisponible
}

func confirmarRespuestaCandidatoExterno(
	solicitud ports.SolicitudResolucionRegistroContextoActorV2,
	respuesta respuestaContextoActorPostgreSQL,
) (ports.ConfirmacionRegistroContextoActorV2, error) {
	confirmacion, err := confirmarRespuestaContextoActor(solicitud, respuesta)
	if err != nil || len(confirmacion.Contexto.Instantanea.Vinculos) != 1 ||
		confirmacion.Contexto.Instantanea.Vinculos[0].Tipo != domain.TipoReferenciaContextoActorCandidato {
		return ports.ConfirmacionRegistroContextoActorV2{}, ports.ErrResolutorRegistroContextoActorNoDisponible
	}
	return confirmacion, nil
}

var _ ports.ResolutorRegistroContextoActorV2 = (*ResolutorRegistroContextoActorExternoPostgreSQLV1)(nil)
