package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// ConsultorEstadisticasRRHHPostgreSQL lee las estadísticas por periodo con la
// función `consultar_estadisticas_rrhh_v1` (migración 000107) sobre el pool
// acreditado de consultas RRHH: mismo login nominal y mismo rol consultor que
// el cuadro. Lectura sin efectos: transacción de solo lectura.
type ConsultorEstadisticasRRHHPostgreSQL struct {
	pool iniciadorTransacciones
}

var _ ports.ConsultorEstadisticasRRHH = (*ConsultorEstadisticasRRHHPostgreSQL)(nil)

func NuevoConsultorEstadisticasRRHHPostgreSQL(pool *PoolConsultasRRHHPostgreSQL) (*ConsultorEstadisticasRRHHPostgreSQL, error) {
	if pool == nil || pool.iniciador == nil {
		return nil, ports.ErrEstadisticasRRHHNoDisponible
	}
	return &ConsultorEstadisticasRRHHPostgreSQL{pool: pool.iniciador}, nil
}

const consultaSQLEstadisticasRRHH = `SELECT corte_global::bigint, inicio, altas::bigint, llamamientos::bigint,
       formalizaciones::bigint, cierres::bigint, incidencias::bigint
  FROM vec_contratacion_temporal.consultar_estadisticas_rrhh_v1(
       ROW($1::text, $2::text, $3::text)::vec_contratacion_temporal.alcance_consulta_rrhh_v1,
       $4::text, $5::date, $6::date)`

func (c *ConsultorEstadisticasRRHHPostgreSQL) ConsultarEstadisticasRRHH(
	ctx context.Context, alcance ports.AlcanceEstadisticasRRHH, consulta ports.ConsultaEstadisticasRRHH,
) (ports.EstadisticasRRHH, error) {
	if ctx == nil || c == nil || c.pool == nil {
		return ports.EstadisticasRRHH{}, ports.ErrEstadisticasRRHHNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return ports.EstadisticasRRHH{}, err
	}
	if err := alcance.Validar(); err != nil {
		return ports.EstadisticasRRHH{}, err
	}
	if err := consulta.Validar(); err != nil {
		return ports.EstadisticasRRHH{}, err
	}
	tx, err := c.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return ports.EstadisticasRRHH{}, errorEstadisticasRRHH(ctx, err)
	}
	defer revertirTransaccion(tx)
	filas, err := tx.Query(ctx, consultaSQLEstadisticasRRHH,
		alcance.OrganizacionRef, string(alcance.ClaseAmbito), alcance.AmbitoRef,
		consulta.Periodo, consulta.Desde, consulta.Hasta)
	if err != nil {
		return ports.EstadisticasRRHH{}, errorEstadisticasRRHH(ctx, err)
	}
	defer filas.Close()
	resultado := ports.EstadisticasRRHH{Series: []ports.SerieEstadisticasRRHH{}}
	var corte int64
	for filas.Next() {
		var serie ports.SerieEstadisticasRRHH
		var inicio time.Time
		var altas, llamamientos, formalizaciones, cierres, incidencias int64
		if err := filas.Scan(&corte, &inicio, &altas, &llamamientos, &formalizaciones, &cierres, &incidencias); err != nil {
			return ports.EstadisticasRRHH{}, errorEstadisticasRRHH(ctx, err)
		}
		if corte < 0 || altas < 0 || llamamientos < 0 || formalizaciones < 0 || cierres < 0 || incidencias < 0 ||
			len(resultado.Series) > 0 && !inicio.After(resultado.Series[len(resultado.Series)-1].Inicio) {
			return ports.EstadisticasRRHH{}, ports.ErrEstadisticasRRHHNoDisponible
		}
		serie.Inicio = inicio.UTC()
		serie.Altas, serie.Llamamientos, serie.Formalizaciones = uint64(altas), uint64(llamamientos), uint64(formalizaciones)
		serie.Cierres, serie.Incidencias = uint64(cierres), uint64(incidencias)
		resultado.Series = append(resultado.Series, serie)
	}
	if err := filas.Err(); err != nil {
		return ports.EstadisticasRRHH{}, errorEstadisticasRRHH(ctx, err)
	}
	if len(resultado.Series) == 0 {
		return ports.EstadisticasRRHH{}, ports.ErrEstadisticasRRHHNoDisponible
	}
	resultado.CorteGlobal = uint64(corte)
	return resultado, nil
}

func errorEstadisticasRRHH(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	var errorPG *pgconn.PgError
	if errors.As(err, &errorPG) && errorPG.Code == "22023" {
		return ports.ErrEstadisticasRRHHInvalida
	}
	return ports.ErrEstadisticasRRHHNoDisponible
}
