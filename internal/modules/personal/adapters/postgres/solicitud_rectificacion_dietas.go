package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
)

const columnasRectificacionDietas = `solicitud_ref,recibo_ref,estado,registrada_en,asignacion_ref,version_origen,decision_ref,efecto_ref,consumo_huella_sha256,auditoria_ad3_ref`
const solicitarRectificacionDietasSQL = `SELECT ` + columnasRectificacionDietas + ` FROM vec_personal.solicitar_rectificacion_dietas_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`
const consultarRectificacionDietasSQL = `SELECT ` + columnasRectificacionDietas + ` FROM vec_personal.consultar_rectificacion_dietas_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`
const resolverRectificacionDietasSQL = `SELECT ` + columnasRectificacionDietas + ` FROM vec_personal.resolver_rectificacion_dietas_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`
const confirmarRectificacionDietasSQL = `SELECT ` + columnasRectificacionDietas + ` FROM vec_personal.confirmar_rectificacion_dietas_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17::numeric,$18::numeric,$19,$20,$21,$22)`

type RepositorioRectificacionDietasPostgreSQL struct{ pool iniciadorConsultaRelacion }

var _ personalports.RepositorioRectificacionDietas = (*RepositorioRectificacionDietasPostgreSQL)(nil)

func NuevoRepositorioRectificacionDietasPostgreSQL(pool *pgxpool.Pool) (*RepositorioRectificacionDietasPostgreSQL, error) {
	return nuevoRepositorioRectificacionDietasPostgreSQL(pool)
}

func nuevoRepositorioRectificacionDietasPostgreSQL(pool iniciadorConsultaRelacion) (*RepositorioRectificacionDietasPostgreSQL, error) {
	if nuloRelacion(pool) {
		return nil, personalports.ErrRectificacionDietasNoDisponible
	}
	return &RepositorioRectificacionDietasPostgreSQL{pool: pool}, nil
}

func (r *RepositorioRectificacionDietasPostgreSQL) EjecutarRectificacionDietas(ctx context.Context, orden personalports.OrdenRectificacionDietas) (personalports.ResultadoRectificacionDietas, error) {
	var vacio personalports.ResultadoRectificacionDietas
	if r == nil || ctx == nil || nuloRelacion(r.pool) {
		return vacio, personalports.ErrRectificacionDietasNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	material := orden.Material.Canonico()
	s := orden.Material.Solicitud()
	if len(material) == 0 || orden.Autorizacion.ValidarEstructura() != nil {
		return vacio, personalports.ErrRectificacionDietasInvalida
	}
	consulta := ""
	switch s.Operacion {
	case personaldomain.SolicitarRectificacionDietas:
		consulta = solicitarRectificacionDietasSQL
	case personaldomain.ConsultarRectificacionDietas:
		consulta = consultarRectificacionDietasSQL
	case personaldomain.RechazarRectificacionDietas:
		consulta = resolverRectificacionDietasSQL
	case personaldomain.ConfirmarRectificacionDietas:
		consulta = confirmarRectificacionDietasSQL
	default:
		return vacio, personalports.ErrRectificacionDietasInvalida
	}
	args, secretos, err := parametrosConsultaRelacionPropia(material, orden.Autorizacion)
	if err != nil {
		return vacio, personalports.ErrRectificacionDietasInvalida
	}
	defer borrarRelacion(secretos[:])
	if s.Operacion == personaldomain.ConfirmarRectificacionDietas {
		correccion := orden.Correccion.Canonico()
		if len(correccion) == 0 || orden.AutorizacionCorreccion.ValidarEstructura() != nil {
			return vacio, personalports.ErrRectificacionDietasDenegada
		}
		argsCorreccion, secretosCorreccion, err := parametrosConsultaRelacionPropia(correccion, orden.AutorizacionCorreccion)
		if err != nil {
			return vacio, personalports.ErrRectificacionDietasDenegada
		}
		defer borrarRelacion(secretosCorreccion[:])
		args = append(args, argsCorreccion...)
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil || tx == nil {
		return vacio, normalizarErrorRectificacionDietas(ctx, err)
	}
	confirmada := false
	defer func() {
		if !confirmada {
			_ = tx.Rollback(context.Background())
		}
	}()
	if _, err = tx.Exec(ctx, ajustesAsignacionDietas); err != nil {
		return vacio, normalizarErrorRectificacionDietas(ctx, err)
	}
	var resultado personalports.ResultadoRectificacionDietas
	err = tx.QueryRow(ctx, consulta, args...).Scan(
		&resultado.SolicitudRef, &resultado.ReciboRef, &resultado.Estado, &resultado.RegistradaEn,
		&resultado.AsignacionRef, &resultado.VersionOrigen, &resultado.DecisionRef, &resultado.EfectoRef,
		&resultado.ConsumoHuellaSHA256, &resultado.AuditoriaAD3Ref,
	)
	if errors.Is(err, pgx.ErrNoRows) && s.Operacion == personaldomain.ConsultarRectificacionDietas {
		if err = tx.Commit(ctx); err != nil {
			return vacio, normalizarErrorRectificacionDietas(ctx, err)
		}
		confirmada = true
		return vacio, personalports.ErrRectificacionDietasNoEncontrada
	}
	if err != nil {
		return vacio, normalizarErrorRectificacionDietas(ctx, err)
	}
	resultado.RegistradaEn = resultado.RegistradaEn.UTC()
	if err = ctx.Err(); err != nil {
		return vacio, err
	}
	if err = tx.Commit(ctx); err != nil {
		return vacio, normalizarErrorRectificacionDietas(ctx, err)
	}
	confirmada = true
	return resultado, nil
}

func normalizarErrorRectificacionDietas(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "42501":
			return personalports.ErrRectificacionDietasDenegada
		case "P7203", "P7204", "23505":
			return personalports.ErrRectificacionDietasConflicto
		case "22023":
			return personalports.ErrRectificacionDietasInvalida
		}
	}
	return personalports.ErrRectificacionDietasNoDisponible
}
