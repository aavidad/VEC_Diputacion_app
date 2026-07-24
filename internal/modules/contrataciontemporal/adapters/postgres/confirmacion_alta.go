package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	funcionConfirmarAltaAtestadaV1 = "vec_contratacion_temporal.confirmar_alta_atestada_v1"
	maximoIntentosConfirmacionAlta = 3
	plazoReconciliacionAlta        = 5 * time.Second
)

type estadoIntentoConfirmacion uint8

const (
	intentoDeterminado estadoIntentoConfirmacion = iota
	intentoReintentable
	intentoIndeterminado
)

type TransaccionAltasPostgreSQL struct {
	pool     iniciadorTransacciones
	material ports.ProveedorMaterialConfirmacionAlta
}

func NuevoTransaccionAltasPostgreSQL(
	pool *pgxpool.Pool,
	material ports.ProveedorMaterialConfirmacionAlta,
) (*TransaccionAltasPostgreSQL, error) {
	return nuevaTransaccionAltasPostgreSQL(pool, material)
}

func nuevaTransaccionAltasPostgreSQL(
	pool iniciadorTransacciones,
	material ports.ProveedorMaterialConfirmacionAlta,
) (*TransaccionAltasPostgreSQL, error) {
	if dependenciaNula(pool) || dependenciaNula(material) {
		return nil, ports.ErrPersistenciaNoDisponible
	}
	return &TransaccionAltasPostgreSQL{pool: pool, material: material}, nil
}

func (t *TransaccionAltasPostgreSQL) ConfirmarAlta(
	ctx context.Context,
	orden ports.OrdenConfirmarAlta,
) (ports.ReciboAlta, error) {
	if ctx == nil || t == nil || dependenciaNula(t.pool) ||
		dependenciaNula(t.material) {
		return ports.ReciboAlta{}, ports.ErrOrdenAltaInvalida
	}
	if err := ctx.Err(); err != nil {
		return ports.ReciboAlta{}, err
	}
	evidencia, err := orden.Datos()
	if err != nil {
		return ports.ReciboAlta{}, ports.ErrOrdenAltaInvalida
	}
	material, err := t.material.ObtenerMaterialConfirmacionAlta(ctx, orden)
	if err != nil {
		return ports.ReciboAlta{}, normalizarErrorMaterialConfirmacion(ctx, err)
	}
	if err := ctx.Err(); err != nil {
		return ports.ReciboAlta{}, err
	}
	parametros, err := construirParametrosConfirmacionAlta(orden, material)
	if err != nil {
		return ports.ReciboAlta{}, err
	}
	defer parametros.borrar()
	return t.confirmarParametros(ctx, evidencia.Expediente, parametros)
}

func (t *TransaccionAltasPostgreSQL) confirmarParametros(
	ctx context.Context,
	expediente domain.Expediente,
	parametros parametrosConfirmacionAlta,
) (ports.ReciboAlta, error) {
	if ctx == nil || expediente.Validar() != nil {
		return ports.ReciboAlta{}, ports.ErrOrdenAltaInvalida
	}
	for intento := 1; intento <= maximoIntentosConfirmacionAlta; intento++ {
		recibo, estado, errIntento := t.ejecutarIntento(
			ctx, expediente, parametros,
		)
		switch {
		case errIntento == nil:
			return recibo, nil
		case estado == intentoIndeterminado:
			return t.reconciliar(ctx, expediente, parametros, recibo)
		case estado == intentoReintentable &&
			intento < maximoIntentosConfirmacionAlta:
			if err := ctx.Err(); err != nil {
				return ports.ReciboAlta{}, err
			}
			continue
		default:
			return ports.ReciboAlta{},
				normalizarErrorConfirmacion(ctx, errIntento)
		}
	}
	return ports.ReciboAlta{}, ports.ErrPersistenciaNoDisponible
}

func (t *TransaccionAltasPostgreSQL) ejecutarIntento(
	ctx context.Context,
	expediente domain.Expediente,
	p parametrosConfirmacionAlta,
) (ports.ReciboAlta, estadoIntentoConfirmacion, error) {
	tx, err := t.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel:   pgx.Serializable,
		AccessMode: pgx.ReadWrite,
	})
	if err != nil {
		return ports.ReciboAlta{}, clasificarErrorDeterminado(err), err
	}
	cerrada := false
	defer func() {
		if !cerrada {
			revertirTransaccion(tx)
		}
	}()
	if err := configurarTransaccionConfirmacion(ctx, tx); err != nil {
		return ports.ReciboAlta{}, clasificarErrorDeterminado(err), err
	}
	recibo, err := consultarReciboConfirmacion(ctx, tx, p)
	if err != nil {
		return ports.ReciboAlta{}, clasificarErrorDeterminado(err), err
	}
	if recibo.ValidarPara(expediente) != nil {
		return ports.ReciboAlta{}, intentoDeterminado,
			ports.ErrResultadoAltaNoConfiable
	}
	err = tx.Commit(ctx)
	cerrada = true
	if err == nil {
		if recibo.ValidarPara(expediente) != nil {
			return ports.ReciboAlta{}, intentoDeterminado,
				ports.ErrResultadoAltaNoConfiable
		}
		return recibo, intentoDeterminado, nil
	}
	if errors.Is(err, pgx.ErrTxCommitRollback) {
		return ports.ReciboAlta{}, intentoDeterminado,
			ports.ErrPersistenciaNoDisponible
	}
	// pgconn solo marca SafeToRetry cuando garantiza que no llegó a enviar
	// ningún dato al servidor. No existe entonces un COMMIT que reconciliar:
	// repetir con un contexto nuevo podría crear un efecto después de que el
	// solicitante hubiera cancelado.
	if pgconn.SafeToRetry(err) {
		return ports.ReciboAlta{}, intentoDeterminado, err
	}
	if errorPostgreSQLReintentable(err) {
		return ports.ReciboAlta{}, intentoReintentable, err
	}
	var postgres *pgconn.PgError
	if errors.As(err, &postgres) {
		if postgres.Code == "08007" {
			return recibo, intentoIndeterminado, err
		}
		return ports.ReciboAlta{}, intentoDeterminado, err
	}
	return recibo, intentoIndeterminado, err
}

func (t *TransaccionAltasPostgreSQL) reconciliar(
	ctx context.Context,
	expediente domain.Expediente,
	p parametrosConfirmacionAlta,
	reciboPrevio ports.ReciboAlta,
) (ports.ReciboAlta, error) {
	base := context.Background()
	if ctx != nil {
		base = context.WithoutCancel(ctx)
	}
	ctxReconciliacion, cancelar := context.WithTimeout(
		base, plazoReconciliacionAlta,
	)
	defer cancelar()
	recibo, estado, err := t.ejecutarIntento(
		ctxReconciliacion, expediente, p,
	)
	if err != nil || estado != intentoDeterminado ||
		recibo != reciboPrevio {
		return ports.ReciboAlta{}, ports.ErrResultadoAltaIndeterminado
	}
	return recibo, nil
}

func configurarTransaccionConfirmacion(
	ctx context.Context,
	tx pgx.Tx,
) error {
	_, err := tx.Exec(ctx, `
		SELECT set_config('search_path', 'pg_catalog', true),
		       set_config('row_security', 'on', true),
		       set_config('timezone', 'UTC', true),
		       set_config('lock_timeout', '2s', true),
		       set_config('statement_timeout', '15s', true),
		       set_config(
		           'idle_in_transaction_session_timeout', '20s', true
		       )`)
	return err
}

func consultarReciboConfirmacion(
	ctx context.Context,
	tx pgx.Tx,
	p parametrosConfirmacionAlta,
) (ports.ReciboAlta, error) {
	filas, err := tx.Query(ctx, `
		SELECT expediente_ref, numero_visible, version::text,
		       recibo_ref, auditoria_ref, evento_ref, confirmada_en,
		       recibo_huella_sha256
		  FROM vec_contratacion_temporal.confirmar_alta_atestada_v1(
		       $1, $2, $3, $4, $5::numeric, $6::numeric,
		       $7, $8, $9, $10, $11, $12
		  )`,
		p.capacidad, p.decision, p.motivo, p.contexto,
		p.personaVersion, p.perfilVersion, p.payload, p.cose,
		p.evidencia, p.spki, p.alta, p.sellos,
	)
	if err != nil {
		return ports.ReciboAlta{}, err
	}
	defer filas.Close()
	if !filas.Next() {
		if err := filas.Err(); err != nil {
			return ports.ReciboAlta{}, err
		}
		return ports.ReciboAlta{}, ports.ErrResultadoAltaNoConfiable
	}
	var recibo ports.ReciboAlta
	var version string
	if err := filas.Scan(
		&recibo.ExpedienteRef,
		&recibo.NumeroVisible,
		&version,
		&recibo.ReciboRef,
		&recibo.AuditoriaRef,
		&recibo.EventoRef,
		&recibo.ConfirmadaEn,
		&recibo.ReciboHuellaSHA256,
	); err != nil {
		return ports.ReciboAlta{}, ports.ErrResultadoAltaNoConfiable
	}
	// timestamptz no conserva zona. pgx puede materializar el mismo instante
	// con time.Local aunque la sesión esté fijada a UTC; el contrato nominal
	// exige la localización canónica time.UTC.
	recibo.ConfirmadaEn = recibo.ConfirmadaEn.UTC()
	versionNumero, err := parsearVersionReciboAlta(version)
	if err != nil {
		return ports.ReciboAlta{}, ports.ErrResultadoAltaNoConfiable
	}
	recibo.Version = versionNumero
	if filas.Next() {
		return ports.ReciboAlta{}, ports.ErrResultadoAltaNoConfiable
	}
	if err := filas.Err(); err != nil {
		return ports.ReciboAlta{}, err
	}
	if recibo.ValidarEstructura() != nil {
		return ports.ReciboAlta{}, ports.ErrResultadoAltaNoConfiable
	}
	return recibo, nil
}

func parsearVersionReciboAlta(valor string) (uint64, error) {
	if valor == "" || (len(valor) > 1 && valor[0] == '0') {
		return 0, ports.ErrResultadoAltaNoConfiable
	}
	var version uint64
	for _, digito := range []byte(valor) {
		if digito < '0' || digito > '9' ||
			version > (^uint64(0)-uint64(digito-'0'))/10 {
			return 0, ports.ErrResultadoAltaNoConfiable
		}
		version = version*10 + uint64(digito-'0')
	}
	if version == 0 {
		return 0, ports.ErrResultadoAltaNoConfiable
	}
	return version, nil
}

func clasificarErrorDeterminado(err error) estadoIntentoConfirmacion {
	if errorPostgreSQLReintentable(err) {
		return intentoReintentable
	}
	return intentoDeterminado
}

func normalizarErrorMaterialConfirmacion(
	ctx context.Context,
	err error,
) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, ports.ErrAutorizacionDenegada) {
		return ports.ErrAutorizacionDenegada
	}
	return ports.ErrPersistenciaNoDisponible
}

func normalizarErrorConfirmacion(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, ports.ErrResultadoAltaNoConfiable) {
		return ports.ErrResultadoAltaNoConfiable
	}
	var postgres *pgconn.PgError
	if errors.As(err, &postgres) && postgres.Code == "42501" {
		return ports.ErrAutorizacionDenegada
	}
	return ports.ErrPersistenciaNoDisponible
}

func (p *parametrosConfirmacionAlta) borrar() {
	if p == nil {
		return
	}
	for _, contenido := range [][]byte{
		p.capacidad, p.decision, p.motivo, p.contexto, p.payload,
		p.cose, p.evidencia, p.spki, p.alta, p.sellos,
	} {
		borrarBytes(contenido)
	}
	p.personaVersion = ""
	p.perfilVersion = ""
}

var _ ports.TransaccionAltas = (*TransaccionAltasPostgreSQL)(nil)
