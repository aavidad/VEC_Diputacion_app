package postgres

import (
	"context"
	"crypto/rand"
	"errors"
	"io"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const consultaResolverContextoActorPerfilPersonalB11V1 = `
	SELECT operacion_ref, registro_contexto_ref,
	       representacion_canonica, huella_sha256,
	       manifiesto_procedencia_canonico,
	       manifiesto_procedencia_huella_sha256,
	       autoridad_efectiva, resuelto_en
	  FROM vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_perfil_personal_b11_v1(
       $1, $2, $3, $4, $5, $6)`

const consultaReconciliarContextoActorPerfilPersonalB11V1 = `
	SELECT operacion_ref, registro_contexto_ref,
	       representacion_canonica, huella_sha256,
	       manifiesto_procedencia_canonico,
	       manifiesto_procedencia_huella_sha256,
	       autoridad_efectiva, resuelto_en
	  FROM vec_contexto_actor_v1.reconciliar_contexto_actor_perfil_personal_b11_v1(
	       $1, $2, $3, $4, $5, $6)`

// ResolutorContextoActorPerfilPersonalB11PostgreSQLV1 es la frontera de Mi
// bolsa. La cuenta, metodo y garantia proceden exclusivamente de la
// autenticacion revalidada; el perfil nunca es una entrada del navegador.
type ResolutorContextoActorPerfilPersonalB11PostgreSQLV1 struct {
	pool      iniciadorContextoActorPostgreSQL
	aleatorio io.Reader
	reloj     func() time.Time
}

func NuevoResolutorContextoActorPerfilPersonalB11PostgreSQLV1(
	ctx context.Context,
	pool *pgxpool.Pool,
) (*ResolutorContextoActorPerfilPersonalB11PostgreSQLV1, error) {
	if ctx == nil || pool == nil {
		return nil, domain.ErrVinculoAutenticacionActorV2Invalido
	}
	// La acreditacion del LOGIN debe corresponder a la fachada B11. Esta
	// llamada no acepta GUC ni atributos declarados por el cliente.
	var identidad string
	var acreditada bool
	if err := pool.QueryRow(ctx, `
		SELECT identidad_login, acreditada
		  FROM vec_contexto_actor_v1.acreditar_runtime_contexto_actor_perfil_personal_b11_v1()`).
		Scan(&identidad, &acreditada); err != nil || !acreditada || identidad == "" {
		return nil, domain.ErrVinculoAutenticacionActorV2Invalido
	}
	return nuevoResolutorContextoActorPerfilPersonalB11PostgreSQLV1(pool, rand.Reader, time.Now)
}

func nuevoResolutorContextoActorPerfilPersonalB11PostgreSQLV1(
	pool iniciadorContextoActorPostgreSQL,
	aleatorio io.Reader,
	reloj func() time.Time,
) (*ResolutorContextoActorPerfilPersonalB11PostgreSQLV1, error) {
	if valorNuloContextoActorPostgreSQL(pool) || valorNuloContextoActorPostgreSQL(aleatorio) || reloj == nil {
		return nil, domain.ErrVinculoAutenticacionActorV2Invalido
	}
	return &ResolutorContextoActorPerfilPersonalB11PostgreSQLV1{pool: pool, aleatorio: aleatorio, reloj: reloj}, nil
}

// ResolverContextoActorGobernadoV1 satisface la futura autoridad de dominio.
// Conserva operacion y recibo durante los reintentos internos, pero no expone
// esas referencias como una capacidad del transporte.
func (r *ResolutorContextoActorPerfilPersonalB11PostgreSQLV1) ResolverContextoActorGobernadoV1(
	ctx context.Context,
	autenticacion domain.AutenticacionRevalidadaV1,
) (domain.ResultadoContextoActorRegistradoV2, error) {
	if ctx == nil || r == nil || valorNuloContextoActorPostgreSQL(r.pool) ||
		valorNuloContextoActorPostgreSQL(r.aleatorio) || r.reloj == nil ||
		autenticacion.Validar() != nil || autenticacion.CuentaPrivilegiada ||
		autenticacion.Superficie != domain.SuperficieAutenticacionExternaPersonalV1 {
		return domain.ResultadoContextoActorRegistradoV2{}, domain.ErrVinculoAutenticacionActorV2Invalido
	}
	if err := ctx.Err(); err != nil {
		return domain.ResultadoContextoActorRegistradoV2{}, err
	}
	operacionRef, err := nuevaReferenciaContextoActorV2(ctx, r.aleatorio, "oca_", domain.ErrVinculoAutenticacionActorV2Invalido)
	if err != nil {
		return domain.ResultadoContextoActorRegistradoV2{}, err
	}
	reciboRef, err := nuevaReferenciaContextoActorV2(ctx, r.aleatorio, "rca_", domain.ErrVinculoAutenticacionActorV2Invalido)
	if err != nil {
		return domain.ResultadoContextoActorRegistradoV2{}, err
	}
	solicitadoEn := r.reloj().UTC().Truncate(time.Microsecond)
	if solicitadoEn.IsZero() {
		return domain.ResultadoContextoActorRegistradoV2{}, domain.ErrVinculoAutenticacionActorV2Invalido
	}
	argumentos := []any{operacionRef, reciboRef, autenticacion.CuentaRef,
		string(autenticacion.MetodoObservado), string(autenticacion.GarantiaObservada), solicitadoEn}

	for intento := 0; intento < 2; intento++ {
		respuesta, estado := r.ejecutarB11(ctx, argumentos)
		if estado == estadoContextoActorReintentable {
			continue
		}
		if estado == estadoContextoActorRechazado {
			return domain.ResultadoContextoActorRegistradoV2{}, domain.ErrVinculoAutenticacionActorV2Invalido
		}
		if estado == estadoContextoActorCommitIncierto {
			reconciliada, estadoReconciliacion := r.reconciliarB11(ctx, argumentos)
			if estadoReconciliacion == estadoContextoActorConfirmado {
				return resultadoPerfilPersonalB11(autenticacion, reconciliada)
			}
			if estadoReconciliacion == estadoContextoActorAusente {
				continue
			}
			if estadoReconciliacion == estadoContextoActorRechazado {
				return domain.ResultadoContextoActorRegistradoV2{}, domain.ErrVinculoAutenticacionActorV2Invalido
			}
		}
		if estado != estadoContextoActorConfirmado {
			return domain.ResultadoContextoActorRegistradoV2{}, errors.Join(
				domain.ErrVinculoAutenticacionActorV2Invalido,
				ports.ErrFuenteContextoActorNoDisponible,
			)
		}
		return resultadoPerfilPersonalB11(autenticacion, respuesta)
	}
	return domain.ResultadoContextoActorRegistradoV2{}, errors.Join(
		domain.ErrVinculoAutenticacionActorV2Invalido,
		ports.ErrFuenteContextoActorNoDisponible,
	)
}

func (r *ResolutorContextoActorPerfilPersonalB11PostgreSQLV1) ejecutarB11(
	ctx context.Context, argumentos []any,
) (respuestaContextoActorPostgreSQL, estadoEjecucionContextoActor) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return respuestaContextoActorPostgreSQL{}, estadoContextoActorFallido
	}
	defer revertirContextoActorPostgreSQL(tx)
	if prepararTransaccionContextoActorPostgreSQL(ctx, tx) != nil {
		return respuestaContextoActorPostgreSQL{}, estadoContextoActorFallido
	}
	respuesta, err := consultarRespuestaContextoActor(ctx, tx, consultaResolverContextoActorPerfilPersonalB11V1, argumentos)
	if err != nil {
		if errorContextoActorPostgreSQLReintentable(err) {
			return respuestaContextoActorPostgreSQL{}, estadoContextoActorReintentable
		}
		if errorContextoActorPostgreSQLRechazado(err) {
			return respuestaContextoActorPostgreSQL{}, estadoContextoActorRechazado
		}
		return respuestaContextoActorPostgreSQL{}, estadoContextoActorFallido
	}
	if err := tx.Commit(ctx); err != nil {
		return respuesta, estadoContextoActorCommitIncierto
	}
	return respuesta, estadoContextoActorConfirmado
}

func (r *ResolutorContextoActorPerfilPersonalB11PostgreSQLV1) reconciliarB11(
	ctx context.Context, argumentos []any,
) (respuestaContextoActorPostgreSQL, estadoEjecucionContextoActor) {
	ctxReconciliacion, cancelar := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancelar()
	tx, err := r.pool.BeginTx(ctxReconciliacion, pgx.TxOptions{IsoLevel: pgx.ReadCommitted, AccessMode: pgx.ReadWrite})
	if err != nil {
		return respuestaContextoActorPostgreSQL{}, estadoContextoActorFallido
	}
	defer revertirContextoActorPostgreSQL(tx)
	if prepararTransaccionContextoActorPostgreSQL(ctxReconciliacion, tx) != nil {
		return respuestaContextoActorPostgreSQL{}, estadoContextoActorFallido
	}
	respuesta, err := consultarRespuestaContextoActor(ctxReconciliacion, tx, consultaReconciliarContextoActorPerfilPersonalB11V1, argumentos)
	if errors.Is(err, pgx.ErrNoRows) {
		if tx.Commit(ctxReconciliacion) != nil {
			return respuestaContextoActorPostgreSQL{}, estadoContextoActorFallido
		}
		return respuestaContextoActorPostgreSQL{}, estadoContextoActorAusente
	}
	if errorContextoActorPostgreSQLRechazado(err) {
		return respuestaContextoActorPostgreSQL{}, estadoContextoActorRechazado
	}
	if err != nil || tx.Commit(ctxReconciliacion) != nil {
		return respuestaContextoActorPostgreSQL{}, estadoContextoActorFallido
	}
	return respuesta, estadoContextoActorConfirmado
}

func resultadoPerfilPersonalB11(
	autenticacion domain.AutenticacionRevalidadaV1,
	respuesta respuestaContextoActorPostgreSQL,
) (domain.ResultadoContextoActorRegistradoV2, error) {
	contexto, err := domain.RehidratarContextoActorVinculadoV2(respuesta.representacion)
	if err != nil || contexto.Principal.AuthMethod != autenticacion.MetodoObservado ||
		contexto.Principal.AuthAssurance != autenticacion.GarantiaObservada ||
		contexto.Instantanea.CuentaRef != autenticacion.CuentaRef {
		return domain.ResultadoContextoActorRegistradoV2{}, domain.ErrVinculoAutenticacionActorV2Invalido
	}
	resultado := domain.ResultadoContextoActorRegistradoV2{
		RegistroContextoRef: respuesta.reciboRef, Contexto: contexto,
		RepresentacionCanonica:            append([]byte(nil), respuesta.representacion...),
		HuellaSHA256:                      respuesta.huella,
		ManifiestoProcedenciaCanonico:     append([]byte(nil), respuesta.manifiesto...),
		ManifiestoProcedenciaHuellaSHA256: respuesta.huellaManifiesto,
		AutoridadEfectiva:                 domain.AutoridadProcedenciaContextoActorV1(respuesta.autoridadEfectiva),
		ResueltoEnAutoritativo:            respuesta.resueltoEn,
	}
	clon, err := resultado.Clonar()
	if err != nil {
		return domain.ResultadoContextoActorRegistradoV2{}, errors.Join(domain.ErrVinculoAutenticacionActorV2Invalido, err)
	}
	return clon, nil
}

var _ domain.ResolutorContextoActorGobernadoV1 = (*ResolutorContextoActorPerfilPersonalB11PostgreSQLV1)(nil)
