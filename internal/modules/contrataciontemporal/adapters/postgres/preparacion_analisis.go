package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	funcionPrepararAnalisis  = "vec_contratacion_temporal.preparar_operacion_analisis_v1"
	funcionConsultarAnalisis = "vec_contratacion_temporal.consultar_operacion_analisis_v1"
	esquemaPrepararAnalisis  = "vec.contratacion-temporal.preparar-operacion-analisis.v1"
	maximoIntentosAnalisis   = 3
)

var _ ports.PreparadorOperacionAnalisisIdempotente = (*PreparadorOperacionAnalisisPostgreSQL)(nil)

type PreparadorOperacionAnalisisPostgreSQL struct {
	pool iniciadorTransacciones
}

func NuevoPreparadorOperacionAnalisisPostgreSQL(
	pool *pgxpool.Pool,
) (*PreparadorOperacionAnalisisPostgreSQL, error) {
	return nuevoPreparadorOperacionAnalisisPostgreSQL(pool)
}

func nuevoPreparadorOperacionAnalisisPostgreSQL(
	pool iniciadorTransacciones,
) (*PreparadorOperacionAnalisisPostgreSQL, error) {
	if dependenciaNula(pool) {
		return nil, ports.ErrPersistenciaOperacionAnalisisNoDisponible
	}
	return &PreparadorOperacionAnalisisPostgreSQL{pool: pool}, nil
}

func (p *PreparadorOperacionAnalisisPostgreSQL) PrepararOperacionAnalisis(
	ctx context.Context,
	solicitud ports.SolicitudPrepararOperacionAnalisis,
) (ports.PreparacionOperacionAnalisis, error) {
	if ctx == nil || p == nil || dependenciaNula(p.pool) ||
		solicitud.Validar() != nil {
		return ports.PreparacionOperacionAnalisis{},
			ports.ErrPreparacionOperacionAnalisisInvalida
	}
	operacion, err := nuevaOperacionPrepararAnalisis(solicitud)
	if err != nil {
		return ports.PreparacionOperacionAnalisis{}, err
	}
	contenido, err := json.Marshal(operacion)
	if err != nil {
		return ports.PreparacionOperacionAnalisis{},
			ports.ErrPersistenciaOperacionAnalisisNoDisponible
	}
	defer borrarBytes(contenido)
	for intento := 1; intento <= maximoIntentosAnalisis; intento++ {
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
			return ports.PreparacionOperacionAnalisis{}, ctx.Err()
		}
		if !errorPostgreSQLReintentable(err) ||
			intento == maximoIntentosAnalisis {
			return ports.PreparacionOperacionAnalisis{},
				normalizarErrorPreparacionAnalisis(ctx, err)
		}
	}
	return ports.PreparacionOperacionAnalisis{},
		ports.ErrPersistenciaOperacionAnalisisNoDisponible
}

func (p *PreparadorOperacionAnalisisPostgreSQL) prepararEnTransaccion(
	ctx context.Context,
	solicitud ports.SolicitudPrepararOperacionAnalisis,
	operacion operacionPrepararAnalisisV1,
	contenido []byte,
) (ports.PreparacionOperacionAnalisis, error) {
	tx, err := p.iniciar(ctx, pgx.ReadWrite)
	if err != nil {
		return ports.PreparacionOperacionAnalisis{}, err
	}
	defer revertirTransaccion(tx)
	fila := filaPreparacionAnalisis{}
	err = tx.QueryRow(ctx, `
		SELECT resultado, expediente_json, recibo_json, reserva_ref,
		       recibo_ref, operacion, organizacion_ref, expediente_ref,
		       version_expediente, actor_ref, perfil_ref, artefacto_ref,
		       artefacto_huella_sha256, ambito_hmac,
		       huella_semantica_hmac, estado
		  FROM `+funcionPrepararAnalisis+`($1::jsonb)`,
		contenido,
	).Scan(
		&fila.resultado, &fila.expedienteJSON, &fila.reciboJSON,
		&fila.reservaRef, &fila.reciboRef, &fila.operacion,
		&fila.organizacionRef, &fila.expedienteRef,
		&fila.versionExpediente, &fila.actorRef, &fila.perfilRef,
		&fila.artefactoRef, &fila.artefactoHuellaSHA256,
		&fila.ambitoHMAC, &fila.huellaSemanticaHMAC, &fila.estado,
	)
	if err != nil {
		return ports.PreparacionOperacionAnalisis{}, err
	}
	if fila.resultado == "idempotencia_reutilizada" {
		return ports.PreparacionOperacionAnalisis{},
			ports.ErrClaveIdempotenciaOperacionAnalisisUsada
	}
	preparacion, err := fila.restaurar(solicitud, operacion)
	if err != nil {
		return ports.PreparacionOperacionAnalisis{}, err
	}
	if fila.resultado == "reservada" &&
		!paresOperacionAnalisisCoinciden(
			operacion,
			fila.ambitoHMAC,
			fila.huellaSemanticaHMAC,
		) {
		return ports.PreparacionOperacionAnalisis{},
			ports.ErrPersistenciaOperacionAnalisisNoDisponible
	}
	if err := tx.Commit(ctx); err != nil {
		return ports.PreparacionOperacionAnalisis{}, err
	}
	return preparacion, nil
}

type filaPreparacionAnalisis struct {
	resultado             string
	expedienteJSON        string
	reciboJSON            string
	reservaRef            string
	reciboRef             string
	operacion             string
	organizacionRef       string
	expedienteRef         string
	versionExpediente     int64
	actorRef              string
	perfilRef             string
	artefactoRef          string
	artefactoHuellaSHA256 string
	ambitoHMAC            string
	huellaSemanticaHMAC   string
	estado                string
}

func (p *PreparadorOperacionAnalisisPostgreSQL) ConsultarOperacionAnalisisConfirmada(
	ctx context.Context,
	solicitud ports.SolicitudConsultarOperacionAnalisisConfirmada,
) (ports.ReciboOperacionAnalisis, bool, error) {
	if ctx == nil || p == nil || dependenciaNula(p.pool) ||
		solicitud.Validar() != nil {
		return ports.ReciboOperacionAnalisis{}, false,
			ports.ErrPreparacionOperacionAnalisisInvalida
	}
	ambitos, err := codificarAmbitosConsultaAnalisis(solicitud)
	if err != nil {
		return ports.ReciboOperacionAnalisis{}, false, err
	}
	defer borrarBytes(ambitos)
	for intento := 1; intento <= maximoIntentosAnalisis; intento++ {
		recibo, existe, err := p.consultarEnTransaccion(
			ctx,
			solicitud,
			ambitos,
		)
		if err == nil {
			return recibo, existe, nil
		}
		if ctx.Err() != nil {
			return ports.ReciboOperacionAnalisis{}, false, ctx.Err()
		}
		if !errorPostgreSQLReintentable(err) ||
			intento == maximoIntentosAnalisis {
			return ports.ReciboOperacionAnalisis{}, false,
				normalizarErrorPreparacionAnalisis(ctx, err)
		}
	}
	return ports.ReciboOperacionAnalisis{}, false,
		ports.ErrPersistenciaOperacionAnalisisNoDisponible
}

func (p *PreparadorOperacionAnalisisPostgreSQL) consultarEnTransaccion(
	ctx context.Context,
	solicitud ports.SolicitudConsultarOperacionAnalisisConfirmada,
	ambitos []byte,
) (ports.ReciboOperacionAnalisis, bool, error) {
	tx, err := p.iniciar(ctx, pgx.ReadOnly)
	if err != nil {
		return ports.ReciboOperacionAnalisis{}, false, err
	}
	defer revertirTransaccion(tx)
	var contenido string
	err = tx.QueryRow(ctx, `
		SELECT recibo_json
		  FROM `+funcionConsultarAnalisis+`($1::jsonb)`,
		ambitos,
	).Scan(&contenido)
	if errors.Is(err, pgx.ErrNoRows) {
		if err := tx.Commit(ctx); err != nil {
			return ports.ReciboOperacionAnalisis{}, false, err
		}
		return ports.ReciboOperacionAnalisis{}, false, nil
	}
	if err != nil {
		return ports.ReciboOperacionAnalisis{}, false, err
	}
	recibo, err := reciboConsultaAnalisisSeguro(solicitud, contenido)
	if err != nil {
		return ports.ReciboOperacionAnalisis{}, false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ports.ReciboOperacionAnalisis{}, false, err
	}
	return recibo, true, nil
}

func (p *PreparadorOperacionAnalisisPostgreSQL) iniciar(
	ctx context.Context,
	modo pgx.TxAccessMode,
) (pgx.Tx, error) {
	tx, err := p.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.Serializable, AccessMode: modo,
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

func normalizarErrorPreparacionAnalisis(
	ctx context.Context,
	causa error,
) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(
		causa,
		ports.ErrClaveIdempotenciaOperacionAnalisisUsada,
	) || errors.Is(
		causa,
		ports.ErrPreparacionOperacionAnalisisInvalida,
	) {
		return causa
	}
	return ports.ErrPersistenciaOperacionAnalisisNoDisponible
}
