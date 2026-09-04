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
	funcionPrepararInformeJuridico        = "vec_contratacion_temporal.preparar_informe_juridico_v1"
	esquemaPrepararInformeJuridico        = "vec.contratacion-temporal.preparar-informe-juridico.v1"
	maximoIntentosPrepararInformeJuridico = 3
)

var _ ports.PreparadorInformeJuridicoIdempotente = (*PreparadorInformeJuridicoPostgreSQL)(nil)

type PreparadorInformeJuridicoPostgreSQL struct {
	pool      iniciadorTransacciones
	generador ports.GeneradorReferenciasInformeJuridico
}

func NuevoPreparadorInformeJuridicoPostgreSQL(
	pool *pgxpool.Pool,
	generador ports.GeneradorReferenciasInformeJuridico,
) (*PreparadorInformeJuridicoPostgreSQL, error) {
	return nuevoPreparadorInformeJuridicoPostgreSQL(pool, generador)
}

func nuevoPreparadorInformeJuridicoPostgreSQL(
	pool iniciadorTransacciones,
	generador ports.GeneradorReferenciasInformeJuridico,
) (*PreparadorInformeJuridicoPostgreSQL, error) {
	if dependenciaNula(pool) || dependenciaNula(generador) {
		return nil, ports.ErrPersistenciaInformeJuridicoNoDisponible
	}
	return &PreparadorInformeJuridicoPostgreSQL{
		pool: pool, generador: generador,
	}, nil
}

func (p *PreparadorInformeJuridicoPostgreSQL) PrepararInformeJuridico(
	ctx context.Context,
	solicitud ports.SolicitudPrepararInformeJuridico,
) (ports.PreparacionInformeJuridico, error) {
	if ctx == nil || p == nil || dependenciaNula(p.pool) ||
		dependenciaNula(p.generador) || solicitud.Validar() != nil {
		return ports.PreparacionInformeJuridico{},
			ports.ErrPreparacionInformeJuridicoInvalida
	}
	if err := ctx.Err(); err != nil {
		return ports.PreparacionInformeJuridico{}, err
	}
	referencias, err := p.generador.GenerarReferenciasInformeJuridico(ctx)
	if err != nil {
		return ports.PreparacionInformeJuridico{},
			errorDependenciaInformeJuridico(ctx)
	}
	if referencias.Validar() != nil {
		return ports.PreparacionInformeJuridico{},
			ports.ErrPersistenciaInformeJuridicoNoDisponible
	}
	operacion, err := nuevaOperacionPrepararInformeJuridico(
		solicitud,
		referencias,
	)
	if err != nil {
		return ports.PreparacionInformeJuridico{}, err
	}
	contenido, err := json.Marshal(operacion)
	if err != nil {
		return ports.PreparacionInformeJuridico{},
			ports.ErrPersistenciaInformeJuridicoNoDisponible
	}
	defer borrarBytes(contenido)

	for intento := 1; intento <= maximoIntentosPrepararInformeJuridico; intento++ {
		preparacion, causa := p.prepararInformeJuridicoEnTransaccion(
			ctx,
			solicitud,
			operacion,
			contenido,
		)
		if causa == nil {
			return preparacion, nil
		}
		if ctx.Err() != nil {
			return ports.PreparacionInformeJuridico{}, ctx.Err()
		}
		if !errorPostgreSQLReintentable(causa) ||
			intento == maximoIntentosPrepararInformeJuridico {
			return ports.PreparacionInformeJuridico{},
				normalizarErrorPreparacionInformeJuridico(ctx, causa)
		}
	}
	return ports.PreparacionInformeJuridico{},
		ports.ErrPersistenciaInformeJuridicoNoDisponible
}

func (p *PreparadorInformeJuridicoPostgreSQL) prepararInformeJuridicoEnTransaccion(
	ctx context.Context,
	solicitud ports.SolicitudPrepararInformeJuridico,
	operacion operacionPrepararInformeJuridicoV1,
	contenido []byte,
) (ports.PreparacionInformeJuridico, error) {
	tx, err := p.iniciarInformeJuridico(ctx)
	if err != nil {
		return ports.PreparacionInformeJuridico{}, err
	}
	defer revertirTransaccion(tx)
	fila := filaPreparacionInformeJuridico{}
	err = tx.QueryRow(ctx, `
		SELECT resultado, expediente_json, reserva_ref, informe_ref,
		       documento_ref, recibo_ref, auditoria_ref, evento_ref,
		       ambito_hmac, huella_peticion_hmac, organizacion_ref,
		       expediente_ref, version_expediente, actor_ref, perfil_ref,
		       estado, recibo_json::text
		  FROM `+funcionPrepararInformeJuridico+`($1::jsonb)`,
		contenido,
	).Scan(
		&fila.resultado, &fila.expedienteJSON, &fila.reservaRef,
		&fila.informeRef, &fila.documentoRef, &fila.reciboRef,
		&fila.auditoriaRef, &fila.eventoRef, &fila.ambitoHMAC,
		&fila.huellaPeticionHMAC, &fila.organizacionRef,
		&fila.expedienteRef, &fila.versionExpediente, &fila.actorRef,
		&fila.perfilRef, &fila.estado, &fila.reciboJSON,
	)
	if err != nil {
		return ports.PreparacionInformeJuridico{}, err
	}
	if fila.resultado == "idempotencia_reutilizada" {
		return ports.PreparacionInformeJuridico{},
			ports.ErrClaveIdempotenciaUsada
	}
	preparacion, err := fila.restaurar(solicitud, operacion)
	if err != nil {
		return ports.PreparacionInformeJuridico{}, err
	}
	if err := ctx.Err(); err != nil {
		return ports.PreparacionInformeJuridico{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ports.PreparacionInformeJuridico{}, err
	}
	return preparacion, nil
}

type filaPreparacionInformeJuridico struct {
	resultado          string
	expedienteJSON     string
	reservaRef         string
	informeRef         string
	documentoRef       string
	reciboRef          string
	auditoriaRef       string
	eventoRef          string
	ambitoHMAC         string
	huellaPeticionHMAC string
	organizacionRef    string
	expedienteRef      string
	versionExpediente  int64
	actorRef           string
	perfilRef          string
	estado             string
	reciboJSON         pgtype.Text
}

func (p *PreparadorInformeJuridicoPostgreSQL) iniciarInformeJuridico(
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

func normalizarErrorPreparacionInformeJuridico(
	ctx context.Context,
	causa error,
) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(causa, ports.ErrClaveIdempotenciaUsada) ||
		errors.Is(causa, ports.ErrPreparacionInformeJuridicoInvalida) {
		return causa
	}
	return ports.ErrPersistenciaInformeJuridicoNoDisponible
}

func errorDependenciaInformeJuridico(ctx context.Context) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	return ports.ErrPersistenciaInformeJuridicoNoDisponible
}
