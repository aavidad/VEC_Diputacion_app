package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
)

const (
	funcionPrepararDecisionCoberturaDurable = "" +
		"vec_contratacion_temporal." +
		"preparar_operacion_decision_cobertura_v1"
	funcionConsultarDecisionCoberturaDurable = "" +
		"vec_contratacion_temporal." +
		"consultar_operacion_decision_cobertura_confirmada_v1"
	maximoIntentosPreparacionDecisionCoberturaDurable = 3
)

var (
	errPersistenciaDecisionCoberturaDurableNoDisponible = errors.New(
		"contratacion temporal: persistencia durable de decision de cobertura no disponible",
	)
	_ cobertura.PreparadorOperacionDecisionCoberturaIdempotente = (*PreparadorOperacionDecisionCoberturaDurablePostgreSQL)(nil)
)

// PreparadorOperacionDecisionCoberturaDurablePostgreSQL implementa reserva,
// reapropiación cercada y replay terminal sobre el primario. Solo transmite
// huellas del token y de la semántica; nunca el token ni la clave funcional.
type PreparadorOperacionDecisionCoberturaDurablePostgreSQL struct {
	pool iniciadorTransacciones
}

func NuevoPreparadorOperacionDecisionCoberturaDurablePostgreSQL(
	pool *pgxpool.Pool,
) (*PreparadorOperacionDecisionCoberturaDurablePostgreSQL, error) {
	return nuevoPreparadorOperacionDecisionCoberturaDurablePostgreSQL(pool)
}

func nuevoPreparadorOperacionDecisionCoberturaDurablePostgreSQL(
	pool iniciadorTransacciones,
) (*PreparadorOperacionDecisionCoberturaDurablePostgreSQL, error) {
	if dependenciaNula(pool) {
		return nil, errPersistenciaDecisionCoberturaDurableNoDisponible
	}
	return &PreparadorOperacionDecisionCoberturaDurablePostgreSQL{
		pool: pool,
	}, nil
}

func (p *PreparadorOperacionDecisionCoberturaDurablePostgreSQL) ConsultarOperacionDecisionCoberturaConfirmada(
	ctx context.Context,
	solicitud cobertura.SolicitudConsultarOperacionDecisionCoberturaConfirmada,
) (cobertura.PreparacionOperacionDecisionCobertura, bool, error) {
	if ctx == nil || p == nil || dependenciaNula(p.pool) ||
		solicitud.Validar() != nil {
		return cobertura.PreparacionOperacionDecisionCobertura{}, false,
			cobertura.ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	ambitos, err := codificarAmbitosConsultaDecisionCoberturaDurable(
		solicitud,
	)
	if err != nil {
		return cobertura.PreparacionOperacionDecisionCobertura{}, false, err
	}
	defer borrarBytes(ambitos)
	for intento := 1; intento <=
		maximoIntentosPreparacionDecisionCoberturaDurable; intento++ {
		preparacion, existe, err := p.consultarEnTransaccion(
			ctx,
			solicitud,
			ambitos,
		)
		if err == nil {
			return preparacion, existe, nil
		}
		if ctx.Err() != nil {
			return cobertura.PreparacionOperacionDecisionCobertura{}, false,
				ctx.Err()
		}
		if !errorPostgreSQLReintentable(err) ||
			intento == maximoIntentosPreparacionDecisionCoberturaDurable {
			return cobertura.PreparacionOperacionDecisionCobertura{}, false,
				normalizarErrorDecisionCoberturaDurable(ctx, err)
		}
	}
	return cobertura.PreparacionOperacionDecisionCobertura{}, false,
		errPersistenciaDecisionCoberturaDurableNoDisponible
}

func (p *PreparadorOperacionDecisionCoberturaDurablePostgreSQL) consultarEnTransaccion(
	ctx context.Context,
	solicitud cobertura.SolicitudConsultarOperacionDecisionCoberturaConfirmada,
	ambitos []byte,
) (cobertura.PreparacionOperacionDecisionCobertura, bool, error) {
	tx, err := p.iniciar(ctx, pgx.ReadOnly)
	if err != nil {
		return cobertura.PreparacionOperacionDecisionCobertura{}, false, err
	}
	defer revertirTransaccion(tx)
	var ambitoPersistido string
	var semanticaPersistida string
	var contenido string
	err = tx.QueryRow(ctx, `
		SELECT ambito_persistido_hmac,
		       huella_semantica_persistida_hmac,
		       carga_json
		  FROM `+funcionConsultarDecisionCoberturaDurable+`($1::jsonb)`,
		ambitos,
	).Scan(&ambitoPersistido, &semanticaPersistida, &contenido)
	if errors.Is(err, pgx.ErrNoRows) {
		if err := tx.Commit(ctx); err != nil {
			return cobertura.PreparacionOperacionDecisionCobertura{}, false,
				err
		}
		return cobertura.PreparacionOperacionDecisionCobertura{}, false, nil
	}
	if err != nil {
		return cobertura.PreparacionOperacionDecisionCobertura{}, false, err
	}
	preparacion, err :=
		restaurarPreparacionTerminalDecisionCoberturaDurable(
			solicitud,
			contenido,
		)
	if err != nil {
		return cobertura.PreparacionOperacionDecisionCobertura{}, false, err
	}
	recibo, err := preparacion.ReciboConfirmadoPara(solicitud)
	if err != nil ||
		recibo.AmbitoIdempotenciaHMAC != ambitoPersistido ||
		recibo.HuellaSemanticaHMAC != semanticaPersistida {
		return cobertura.PreparacionOperacionDecisionCobertura{}, false,
			cobertura.ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	if err := tx.Commit(ctx); err != nil {
		return cobertura.PreparacionOperacionDecisionCobertura{}, false, err
	}
	return preparacion, true, nil
}

func (p *PreparadorOperacionDecisionCoberturaDurablePostgreSQL) ReservarOReapropiarOperacionDecisionCobertura(
	ctx context.Context,
	solicitud cobertura.SolicitudReservarOperacionDecisionCobertura,
) (cobertura.PreparacionOperacionDecisionCobertura, error) {
	if ctx == nil || p == nil || dependenciaNula(p.pool) ||
		solicitud.Validar() != nil {
		return cobertura.PreparacionOperacionDecisionCobertura{},
			cobertura.ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	operacion, err :=
		nuevaOperacionPrepararDecisionCoberturaDurableV1(solicitud)
	if err != nil {
		return cobertura.PreparacionOperacionDecisionCobertura{}, err
	}
	contenido, err := json.Marshal(operacion)
	if err != nil {
		return cobertura.PreparacionOperacionDecisionCobertura{},
			errPersistenciaDecisionCoberturaDurableNoDisponible
	}
	defer borrarBytes(contenido)
	for intento := 1; intento <=
		maximoIntentosPreparacionDecisionCoberturaDurable; intento++ {
		preparacion, err := p.reservarEnTransaccion(
			ctx,
			solicitud,
			operacion,
			contenido,
		)
		if err == nil {
			return preparacion, nil
		}
		if ctx.Err() != nil {
			return cobertura.PreparacionOperacionDecisionCobertura{},
				ctx.Err()
		}
		if !errorPostgreSQLReintentable(err) ||
			intento == maximoIntentosPreparacionDecisionCoberturaDurable {
			return cobertura.PreparacionOperacionDecisionCobertura{},
				normalizarErrorDecisionCoberturaDurable(ctx, err)
		}
	}
	return cobertura.PreparacionOperacionDecisionCobertura{},
		errPersistenciaDecisionCoberturaDurableNoDisponible
}

func (p *PreparadorOperacionDecisionCoberturaDurablePostgreSQL) reservarEnTransaccion(
	ctx context.Context,
	solicitud cobertura.SolicitudReservarOperacionDecisionCobertura,
	operacion operacionPrepararDecisionCoberturaDurableV1,
	contenido []byte,
) (cobertura.PreparacionOperacionDecisionCobertura, error) {
	tx, err := p.iniciar(ctx, pgx.ReadWrite)
	if err != nil {
		return cobertura.PreparacionOperacionDecisionCobertura{}, err
	}
	defer revertirTransaccion(tx)

	fila, err := consultarPreparacionDecisionCoberturaDurable(
		ctx,
		tx,
		contenido,
		nil,
	)
	if err != nil {
		return cobertura.PreparacionOperacionDecisionCobertura{}, err
	}
	if fila.resultado == "requiere_validacion" {
		if !solicitud.CoincideParPersistido(
			fila.ambitoPersistido,
			fila.semanticaPersistida,
		) {
			return cobertura.PreparacionOperacionDecisionCobertura{},
				errors.Join(
					cobertura.ErrClaveOperacionDecisionCoberturaUsada,
					cobertura.ErrOperacionDecisionCoberturaIdempotenteInvalida,
				)
		}
		validacion, err := json.Marshal(
			parPersistidoDecisionCoberturaDurableV1{
				AmbitoHMAC:          fila.ambitoPersistido,
				HuellaSemanticaHMAC: fila.semanticaPersistida,
			},
		)
		if err != nil {
			return cobertura.PreparacionOperacionDecisionCobertura{},
				errPersistenciaDecisionCoberturaDurableNoDisponible
		}
		defer borrarBytes(validacion)
		fila, err = consultarPreparacionDecisionCoberturaDurable(
			ctx,
			tx,
			contenido,
			validacion,
		)
		if err != nil {
			return cobertura.PreparacionOperacionDecisionCobertura{}, err
		}
	}
	var preparacion cobertura.PreparacionOperacionDecisionCobertura
	switch fila.resultado {
	case "propietaria":
		preparacion, err =
			restaurarPreparacionPropietariaDecisionCoberturaDurable(
				solicitud,
				operacion,
				fila.contenido,
			)
	case "ocupada":
		preparacion, err =
			cobertura.NuevaPreparacionOperacionDecisionCoberturaOcupada(
				solicitud,
				fila.ambitoPersistido,
				fila.semanticaPersistida,
			)
	case "confirmada":
		var consulta cobertura.SolicitudConsultarOperacionDecisionCoberturaConfirmada
		consulta, err = solicitud.ConsultaConfirmada()
		if err == nil {
			preparacion, err =
				restaurarPreparacionTerminalDecisionCoberturaDurable(
					consulta,
					fila.contenido,
				)
		}
	case "colision":
		err = errors.Join(
			cobertura.ErrClaveOperacionDecisionCoberturaUsada,
			cobertura.ErrOperacionDecisionCoberturaIdempotenteInvalida,
		)
	default:
		err = errPersistenciaDecisionCoberturaDurableNoDisponible
	}
	if err != nil {
		return cobertura.PreparacionOperacionDecisionCobertura{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return cobertura.PreparacionOperacionDecisionCobertura{}, err
	}
	return preparacion, nil
}

type filaPreparacionDecisionCoberturaDurable struct {
	resultado           string
	ambitoPersistido    string
	semanticaPersistida string
	contenido           string
}

func consultarPreparacionDecisionCoberturaDurable(
	ctx context.Context,
	tx pgx.Tx,
	operacion []byte,
	validacion []byte,
) (filaPreparacionDecisionCoberturaDurable, error) {
	var fila filaPreparacionDecisionCoberturaDurable
	var argumentoValidacion any
	if len(validacion) > 0 {
		argumentoValidacion = validacion
	}
	err := tx.QueryRow(ctx, `
		SELECT resultado, ambito_persistido_hmac,
		       huella_semantica_persistida_hmac, carga_json
		  FROM `+funcionPrepararDecisionCoberturaDurable+`(
		       $1::jsonb, $2::jsonb
		  )`,
		operacion,
		argumentoValidacion,
	).Scan(
		&fila.resultado,
		&fila.ambitoPersistido,
		&fila.semanticaPersistida,
		&fila.contenido,
	)
	return fila, err
}

func (p *PreparadorOperacionDecisionCoberturaDurablePostgreSQL) iniciar(
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
		       set_config(
		           'idle_in_transaction_session_timeout',
		           '20s',
		           true
		       )`)
	if err != nil {
		revertirTransaccion(tx)
		return nil, err
	}
	return tx, nil
}

func normalizarErrorDecisionCoberturaDurable(
	ctx context.Context,
	causa error,
) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(
		causa,
		cobertura.ErrOperacionDecisionCoberturaIdempotenteInvalida,
	) || errors.Is(
		causa,
		cobertura.ErrClaveOperacionDecisionCoberturaUsada,
	) {
		return causa
	}
	return errPersistenciaDecisionCoberturaDurableNoDisponible
}
