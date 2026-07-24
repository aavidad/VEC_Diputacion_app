package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	funcionPrepararAsignacion        = "vec_contratacion_temporal.preparar_asignacion_v1"
	esquemaPrepararAsignacion        = "vec.contratacion-temporal.preparar-asignacion.v1"
	maximoIntentosPrepararAsignacion = 3
)

var _ ports.PreparadorAsignacionIdempotente = (*PreparadorAsignacionPostgreSQL)(nil)

type PreparadorAsignacionPostgreSQL struct {
	pool      iniciadorTransacciones
	generador ports.GeneradorReferenciasAsignacion
}

func NuevoPreparadorAsignacionPostgreSQL(
	pool *pgxpool.Pool,
	generador ports.GeneradorReferenciasAsignacion,
) (*PreparadorAsignacionPostgreSQL, error) {
	return nuevoPreparadorAsignacionPostgreSQL(pool, generador)
}

func nuevoPreparadorAsignacionPostgreSQL(
	pool iniciadorTransacciones,
	generador ports.GeneradorReferenciasAsignacion,
) (*PreparadorAsignacionPostgreSQL, error) {
	if dependenciaNula(pool) || dependenciaNula(generador) {
		return nil, ports.ErrPersistenciaAsignacionNoDisponible
	}
	return &PreparadorAsignacionPostgreSQL{
		pool: pool, generador: generador,
	}, nil
}

func (p *PreparadorAsignacionPostgreSQL) PrepararAsignacion(
	ctx context.Context,
	solicitud ports.SolicitudPrepararAsignacion,
) (ports.PreparacionAsignacion, error) {
	if ctx == nil || p == nil || dependenciaNula(p.pool) ||
		dependenciaNula(p.generador) || solicitud.Validar() != nil {
		return ports.PreparacionAsignacion{},
			ports.ErrPreparacionAsignacionInvalida
	}
	if err := ctx.Err(); err != nil {
		return ports.PreparacionAsignacion{}, err
	}
	referencias, err := p.generador.GenerarReferenciasAsignacion(ctx)
	if err != nil {
		return ports.PreparacionAsignacion{},
			errorDependenciaAsignacion(ctx)
	}
	if referencias.Validar() != nil {
		return ports.PreparacionAsignacion{},
			ports.ErrPersistenciaAsignacionNoDisponible
	}
	operacion, err := nuevaOperacionPrepararAsignacion(
		solicitud,
		referencias,
	)
	if err != nil {
		return ports.PreparacionAsignacion{}, err
	}
	contenido, err := json.Marshal(operacion)
	if err != nil {
		return ports.PreparacionAsignacion{},
			ports.ErrPersistenciaAsignacionNoDisponible
	}
	defer borrarBytes(contenido)

	for intento := 1; intento <= maximoIntentosPrepararAsignacion; intento++ {
		preparacion, err := p.prepararEnTransaccion(
			ctx,
			solicitud,
			operacion,
			contenido,
		)
		if err == nil {
			return preparacion, nil
		}
		if ctx.Err() != nil {
			return ports.PreparacionAsignacion{}, ctx.Err()
		}
		if !errorPostgreSQLReintentable(err) ||
			intento == maximoIntentosPrepararAsignacion {
			return ports.PreparacionAsignacion{},
				normalizarErrorPreparacionAsignacion(ctx, err)
		}
	}
	return ports.PreparacionAsignacion{},
		ports.ErrPersistenciaAsignacionNoDisponible
}

func (p *PreparadorAsignacionPostgreSQL) prepararEnTransaccion(
	ctx context.Context,
	solicitud ports.SolicitudPrepararAsignacion,
	operacion operacionPrepararAsignacionV1,
	contenido []byte,
) (ports.PreparacionAsignacion, error) {
	tx, err := p.iniciarAsignacion(ctx)
	if err != nil {
		return ports.PreparacionAsignacion{}, err
	}
	defer revertirTransaccion(tx)
	fila := filaPreparacionAsignacion{}
	err = tx.QueryRow(ctx, `
		SELECT resultado, expediente_json, reserva_ref, recibo_ref,
		       notificacion_ref, bandeja_ref, auditoria_ref, evento_ref,
		       ambito_hmac, huella_peticion_hmac, operacion,
		       organizacion_ref, actor_ref, perfil_ref, unidad_ref,
		       responsable_ref, estado, version_resultante,
		       concesion_v3_decision_ref, confirmada_en
		  FROM `+funcionPrepararAsignacion+`($1::jsonb)`,
		contenido,
	).Scan(
		&fila.resultado, &fila.expedienteJSON, &fila.reservaRef,
		&fila.reciboRef, &fila.notificacionRef, &fila.bandejaRef,
		&fila.auditoriaRef, &fila.eventoRef, &fila.ambitoHMAC,
		&fila.huellaPeticionHMAC, &fila.operacion,
		&fila.organizacionRef, &fila.actorRef, &fila.perfilRef,
		&fila.unidadRef, &fila.responsableRef, &fila.estado,
		&fila.versionResultante, &fila.concesionV3DecisionRef,
		&fila.confirmadaEn,
	)
	if err != nil {
		return ports.PreparacionAsignacion{}, err
	}
	if fila.resultado == "idempotencia_reutilizada" {
		return ports.PreparacionAsignacion{},
			ports.ErrClaveIdempotenciaUsada
	}
	preparacion, err := fila.restaurar(solicitud, operacion)
	if err != nil {
		return ports.PreparacionAsignacion{}, err
	}
	if err := ctx.Err(); err != nil {
		return ports.PreparacionAsignacion{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ports.PreparacionAsignacion{}, err
	}
	return preparacion, nil
}

type filaPreparacionAsignacion struct {
	resultado              string
	expedienteJSON         string
	reservaRef             string
	reciboRef              string
	notificacionRef        string
	bandejaRef             string
	auditoriaRef           string
	eventoRef              string
	ambitoHMAC             string
	huellaPeticionHMAC     string
	operacion              string
	organizacionRef        string
	actorRef               string
	perfilRef              string
	unidadRef              string
	responsableRef         string
	estado                 string
	versionResultante      pgtype.Int8
	concesionV3DecisionRef pgtype.Text
	confirmadaEn           pgtype.Timestamptz
}

func (p *PreparadorAsignacionPostgreSQL) iniciarAsignacion(
	ctx context.Context,
) (pgx.Tx, error) {
	tx, err := p.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite,
	})
	if err != nil {
		return nil, err
	}
	_, err = tx.Exec(ctx, `
		SELECT set_config('search_path', 'pg_catalog', true),
		       set_config('row_security', 'on', true),
		       set_config('timezone', 'UTC', true),
		       set_config('lock_timeout', '2s', true),
		       set_config('statement_timeout', '15s', true),
		       set_config('idle_in_transaction_session_timeout', '20s', true)`)
	if err != nil {
		revertirTransaccion(tx)
		return nil, err
	}
	return tx, nil
}

func normalizarErrorPreparacionAsignacion(
	ctx context.Context,
	causa error,
) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(causa, ports.ErrClaveIdempotenciaUsada) ||
		errors.Is(causa, ports.ErrPreparacionAsignacionInvalida) {
		return causa
	}
	return ports.ErrPersistenciaAsignacionNoDisponible
}

func errorDependenciaAsignacion(ctx context.Context) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	return ports.ErrPersistenciaAsignacionNoDisponible
}
