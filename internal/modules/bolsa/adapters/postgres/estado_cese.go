package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

type ConsultaEstadoCesePostgreSQL struct{ pool *pgxpool.Pool }

var _ ports.ConsultaEstadoCese = (*ConsultaEstadoCesePostgreSQL)(nil)
var _ ports.ConsultaEstadosCese = (*ConsultaEstadoCesePostgreSQL)(nil)

func NuevaConsultaEstadoCesePostgreSQL(pool *pgxpool.Pool) (*ConsultaEstadoCesePostgreSQL, error) {
	if pool == nil {
		return nil, ports.ErrConsultaEstadoCeseNoDisponible
	}
	return &ConsultaEstadoCesePostgreSQL{pool: pool}, nil
}

// ConsultarEstadoCese lee la política y el pendiente propios de Bolsa.
// La función no expone una referencia de candidato ni datos de Personal/CT.
func (c *ConsultaEstadoCesePostgreSQL) ConsultarEstadoCese(ctx context.Context, participacionRef string, corte time.Time) (ports.EstadoCese, bool, error) {
	if c == nil || c.pool == nil || ctx == nil || ctx.Err() != nil || participacionRef == "" ||
		participacionRef != strings.TrimSpace(participacionRef) || corte.IsZero() {
		return ports.EstadoCese{}, false, ports.ErrConsultaEstadoCeseNoDisponible
	}
	return escanearEstadoCese(c.pool.QueryRow(ctx,
		`SELECT fecha_efecto,disponible_desde,en_restriccion,trabajo_cesado,cese_pendiente
		FROM vec_bolsa_llamamientos.consultar_estado_cese_bolsa_v2($1,$2)`,
		participacionRef, corte.UTC()), corte)
}

const maximoLoteEstadosCese = 20000

// ConsultarEstadosCese emite una sola sentencia SQL para toda la página.
// Una ausencia en el resultado conserva la situación anterior de la participación.
func (c *ConsultaEstadoCesePostgreSQL) ConsultarEstadosCese(ctx context.Context, refs []string, corte time.Time) (map[string]ports.EstadoCese, error) {
	if c == nil || c.pool == nil || ctx == nil || ctx.Err() != nil || corte.IsZero() || len(refs) > maximoLoteEstadosCese {
		return nil, ports.ErrConsultaEstadoCeseNoDisponible
	}
	esperadas := make(map[string]struct{}, len(refs))
	for _, ref := range refs {
		if ref == "" || ref != strings.TrimSpace(ref) {
			return nil, ports.ErrConsultaEstadoCeseNoDisponible
		}
		esperadas[ref] = struct{}{}
	}
	salida := make(map[string]ports.EstadoCese, len(refs))
	if len(refs) == 0 {
		return salida, nil
	}
	return consultarEstadosCeseLote(ctx, c.pool, refs, esperadas, corte, salida)
}

type consultaEstadosCeseQueryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func consultarEstadosCeseLote(ctx context.Context, consulta consultaEstadosCeseQueryer, refs []string, esperadas map[string]struct{}, corte time.Time, salida map[string]ports.EstadoCese) (map[string]ports.EstadoCese, error) {
	filas, err := consulta.Query(ctx,
		`SELECT participacion_ref,fecha_efecto,disponible_desde,en_restriccion,trabajo_cesado,cese_pendiente
		FROM vec_bolsa_llamamientos.consultar_estado_cese_bolsa_lote_v2($1::text[],$2)`, refs, corte.UTC())
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ports.ErrConsultaEstadoCeseNoDisponible, err)
	}
	defer filas.Close()
	for filas.Next() {
		var ref string
		var efecto, disponible sql.NullTime
		var restringida, cesado, pendiente bool
		if err := filas.Scan(&ref, &efecto, &disponible, &restringida, &cesado, &pendiente); err != nil {
			return nil, fmt.Errorf("%w: %w", ports.ErrConsultaEstadoCeseNoDisponible, err)
		}
		if _, solicitada := esperadas[ref]; !solicitada || len(salida) >= len(esperadas) {
			return nil, ports.ErrConsultaEstadoCeseNoDisponible
		}
		if _, repetida := salida[ref]; repetida {
			return nil, ports.ErrConsultaEstadoCeseNoDisponible
		}
		estado, presente, err := validarEstadoCeseNullable(efecto, disponible, restringida, cesado, pendiente, corte)
		if err != nil || !presente {
			return nil, ports.ErrConsultaEstadoCeseNoDisponible
		}
		salida[ref] = estado
	}
	if err := filas.Err(); err != nil {
		return nil, fmt.Errorf("%w: %w", ports.ErrConsultaEstadoCeseNoDisponible, err)
	}
	return salida, nil
}

func escanearEstadoCese(fila pgx.Row, corte time.Time) (ports.EstadoCese, bool, error) {
	var efecto, disponible sql.NullTime
	var restringida, cesado, pendiente bool
	err := fila.Scan(&efecto, &disponible, &restringida, &cesado, &pendiente)
	if errors.Is(err, pgx.ErrNoRows) {
		return ports.EstadoCese{}, false, nil
	}
	if err != nil {
		return ports.EstadoCese{}, false, fmt.Errorf("%w: %w", ports.ErrConsultaEstadoCeseNoDisponible, err)
	}
	return validarEstadoCeseNullable(efecto, disponible, restringida, cesado, pendiente, corte)
}

func validarEstadoCeseNullable(efecto, disponible sql.NullTime, restringida, cesado, pendiente bool, corte time.Time) (ports.EstadoCese, bool, error) {
	if !efecto.Valid && !disponible.Valid && pendiente && !restringida && !cesado && !corte.IsZero() {
		return ports.EstadoCese{CesePendiente: true}, true, nil
	}
	if !efecto.Valid || !disponible.Valid {
		return ports.EstadoCese{}, false, ports.ErrConsultaEstadoCeseNoDisponible
	}
	estado, presente, err := validarEstadoCese(efecto.Time, disponible.Time, restringida, cesado, corte)
	if err != nil {
		return ports.EstadoCese{}, false, err
	}
	estado.CesePendiente = pendiente
	return estado, presente, nil
}

// validarEstadoCese convierte las fechas del día de Madrid y rechaza estados
// incoherentes con el corte; lo comparten el resumen legado y las lecturas v2.
func validarEstadoCese(efecto, disponible time.Time, restringida, cesado bool, corte time.Time) (ports.EstadoCese, bool, error) {
	madrid, err := time.LoadLocation("Europe/Madrid")
	if err != nil || efecto.IsZero() || disponible.IsZero() || corte.IsZero() {
		return ports.EstadoCese{}, false, ports.ErrConsultaEstadoCeseNoDisponible
	}
	yE, mE, dE := efecto.Date()
	yD, mD, dD := disponible.Date()
	efecto = time.Date(yE, mE, dE, 0, 0, 0, 0, madrid)
	disponible = time.Date(yD, mD, dD, 0, 0, 0, 0, madrid)
	yC, mC, dC := corte.In(madrid).Date()
	diaCorte := time.Date(yC, mC, dC, 0, 0, 0, 0, madrid)
	if efecto.After(diaCorte) || disponible.Before(efecto) || (restringida && !disponible.After(diaCorte)) {
		return ports.EstadoCese{}, false, ports.ErrConsultaEstadoCeseNoDisponible
	}
	return ports.EstadoCese{FechaEfecto: efecto, DisponibleDesde: disponible,
		EnRestriccion: restringida, TrabajoCesado: cesado}, true, nil
}
