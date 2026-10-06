package postgres

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

type ConsultaEstadoCesePostgreSQL struct{ pool *pgxpool.Pool }

var _ ports.ConsultaEstadoCese = (*ConsultaEstadoCesePostgreSQL)(nil)

func NuevaConsultaEstadoCesePostgreSQL(pool *pgxpool.Pool) (*ConsultaEstadoCesePostgreSQL, error) {
	if pool == nil {
		return nil, ports.ErrConsultaEstadoCeseNoDisponible
	}
	return &ConsultaEstadoCesePostgreSQL{pool: pool}, nil
}

// ConsultarEstadoCese lee la fachada propia Bolsa 000050, nunca tablas
// Personal/CT ni una referencia de candidato. La función B45 interna permanece
// privada; la fachada solo concede EXECUTE al ejecutor nominal.
func (c *ConsultaEstadoCesePostgreSQL) ConsultarEstadoCese(ctx context.Context, participacionRef string, corte time.Time) (ports.EstadoCese, bool, error) {
	if c == nil || c.pool == nil || ctx == nil || ctx.Err() != nil || participacionRef == "" ||
		participacionRef != strings.TrimSpace(participacionRef) || corte.IsZero() {
		return ports.EstadoCese{}, false, ports.ErrConsultaEstadoCeseNoDisponible
	}
	return escanearEstadoCese(c.pool.QueryRow(ctx,
		`SELECT fecha_efecto,disponible_desde,en_restriccion,trabajo_cesado FROM vec_bolsa_llamamientos.consultar_estado_cese_bolsa_v1($1,$2)`,
		participacionRef, corte.UTC()), corte)
}

var _ ports.ConsultaEstadosCese = (*ConsultaEstadoCesePostgreSQL)(nil)

const maximoLoteEstadosCese = 20000

// ConsultarEstadosCese envía en un único lote la misma consulta a la fachada
// Bolsa 000050 que ConsultarEstadoCese, una por participación, y valida cada
// fila igual. La función conserva su guarda de rol en cada llamada.
func (c *ConsultaEstadoCesePostgreSQL) ConsultarEstadosCese(ctx context.Context, refs []string, corte time.Time) (map[string]ports.EstadoCese, error) {
	if c == nil || c.pool == nil || ctx == nil || ctx.Err() != nil || corte.IsZero() || len(refs) > maximoLoteEstadosCese {
		return nil, ports.ErrConsultaEstadoCeseNoDisponible
	}
	for _, ref := range refs {
		if ref == "" || ref != strings.TrimSpace(ref) {
			return nil, ports.ErrConsultaEstadoCeseNoDisponible
		}
	}
	salida := make(map[string]ports.EstadoCese)
	if len(refs) == 0 {
		return salida, nil
	}
	lote := &pgx.Batch{}
	for _, ref := range refs {
		lote.Queue(`SELECT fecha_efecto,disponible_desde,en_restriccion,trabajo_cesado FROM vec_bolsa_llamamientos.consultar_estado_cese_bolsa_v1($1,$2)`, ref, corte.UTC())
	}
	resultados := c.pool.SendBatch(ctx, lote)
	defer resultados.Close()
	for _, ref := range refs {
		estado, presente, err := escanearEstadoCese(resultados.QueryRow(), corte)
		if err != nil {
			return nil, err
		}
		if presente {
			salida[ref] = estado
		}
	}
	if err := resultados.Close(); err != nil {
		return nil, ports.ErrConsultaEstadoCeseNoDisponible
	}
	return salida, nil
}

func escanearEstadoCese(fila pgx.Row, corte time.Time) (ports.EstadoCese, bool, error) {
	var efecto, disponible time.Time
	var restringida, cesado bool
	err := fila.Scan(&efecto, &disponible, &restringida, &cesado)
	if errors.Is(err, pgx.ErrNoRows) {
		return ports.EstadoCese{}, false, nil
	}
	if err != nil {
		return ports.EstadoCese{}, false, ports.ErrConsultaEstadoCeseNoDisponible
	}
	madrid, err := time.LoadLocation("Europe/Madrid")
	if err != nil || efecto.IsZero() || disponible.IsZero() {
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
