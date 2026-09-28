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

type ConsultaRestriccionCesePostgreSQL struct{ pool *pgxpool.Pool }

var _ ports.ConsultaRestriccionCese = (*ConsultaRestriccionCesePostgreSQL)(nil)

func NuevaConsultaRestriccionCesePostgreSQL(pool *pgxpool.Pool) (*ConsultaRestriccionCesePostgreSQL, error) {
	if pool == nil {
		return nil, ports.ErrConsultaRestriccionCeseNoDisponible
	}
	return &ConsultaRestriccionCesePostgreSQL{pool: pool}, nil
}

// ConsultarRestriccionCese usa exclusivamente la función B45 de Bolsa. La
// ausencia de fila es un resultado normal; errores y datos incoherentes cierran
// la lectura sin exponer SQL, candidato_ref ni información de otra bolsa.
func (c *ConsultaRestriccionCesePostgreSQL) ConsultarRestriccionCese(ctx context.Context, participacionRef string, corte time.Time) (ports.RestriccionCese, bool, error) {
	if c == nil || c.pool == nil || ctx == nil || participacionRef == "" || strings.TrimSpace(participacionRef) != participacionRef || corte.IsZero() {
		return ports.RestriccionCese{}, false, ports.ErrConsultaRestriccionCeseNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return ports.RestriccionCese{}, false, ports.ErrConsultaRestriccionCeseNoDisponible
	}
	return escanearRestriccionCese(c.pool.QueryRow(ctx,
		`SELECT disponible_desde,fecha_efecto,recibo_ref,politica_version FROM vec_bolsa_llamamientos.consultar_restriccion_cese_bolsa_v1($1,$2)`,
		participacionRef, corte.UTC()), corte)
}

func escanearRestriccionCese(fila pgx.Row, corte time.Time) (ports.RestriccionCese, bool, error) {
	var disponible, efecto time.Time
	var recibo string
	var version int64
	err := fila.Scan(&disponible, &efecto, &recibo, &version)
	if errors.Is(err, pgx.ErrNoRows) {
		return ports.RestriccionCese{}, false, nil
	}
	if err != nil {
		return ports.RestriccionCese{}, false, ports.ErrConsultaRestriccionCeseNoDisponible
	}
	madrid, err := time.LoadLocation("Europe/Madrid")
	if err != nil || disponible.IsZero() || efecto.IsZero() || recibo == "" || strings.TrimSpace(recibo) != recibo || version <= 0 {
		return ports.RestriccionCese{}, false, ports.ErrConsultaRestriccionCeseNoDisponible
	}
	anyoD, mesD, diaD := disponible.Date()
	anyoE, mesE, diaE := efecto.Date()
	disponible = time.Date(anyoD, mesD, diaD, 0, 0, 0, 0, madrid)
	efecto = time.Date(anyoE, mesE, diaE, 0, 0, 0, 0, madrid)
	anyoC, mesC, diaC := corte.In(madrid).Date()
	fechaCorte := time.Date(anyoC, mesC, diaC, 0, 0, 0, 0, madrid)
	if !disponible.After(fechaCorte) || efecto.After(fechaCorte) {
		return ports.RestriccionCese{}, false, ports.ErrConsultaRestriccionCeseNoDisponible
	}
	return ports.RestriccionCese{DisponibleDesde: disponible, FechaEfecto: efecto, ReciboRef: recibo, PoliticaVersion: uint64(version)}, true, nil
}
