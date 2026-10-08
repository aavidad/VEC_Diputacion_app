package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

type consultaCeseLotePrueba struct {
	filas *filasCeseLotePrueba
	err   error
	n     int
	args  []any
	sql   string
}

func (c *consultaCeseLotePrueba) Query(_ context.Context, sqlTexto string, args ...any) (pgx.Rows, error) {
	c.n++
	c.sql, c.args = sqlTexto, args
	if c.err != nil {
		return nil, c.err
	}
	return c.filas, nil
}

type filaCeseLotePrueba struct {
	ref                                string
	efecto, disponible, pendienteDesde sql.NullTime
	restringida, cesado, pendiente     bool
}

type filasCeseLotePrueba struct {
	pgx.Rows
	filas  []filaCeseLotePrueba
	indice int
	err    error
	closed bool
}

func (f *filasCeseLotePrueba) Close()     { f.closed = true }
func (f *filasCeseLotePrueba) Err() error { return f.err }
func (f *filasCeseLotePrueba) Next() bool {
	if f.indice >= len(f.filas) {
		return false
	}
	f.indice++
	return true
}
func (f *filasCeseLotePrueba) Scan(destinos ...any) error {
	fila := f.filas[f.indice-1]
	*destinos[0].(*string) = fila.ref
	*destinos[1].(*sql.NullTime) = fila.efecto
	*destinos[2].(*sql.NullTime) = fila.disponible
	*destinos[3].(*bool) = fila.restringida
	*destinos[4].(*bool) = fila.cesado
	*destinos[5].(*bool) = fila.pendiente
	*destinos[6].(*sql.NullTime) = fila.pendienteDesde
	return nil
}

func TestEstadoCeseLoteUnaSentenciaYPendienteSinFecha(t *testing.T) {
	corte := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	refs := []string{"participacion:1", "participacion:2", "participacion:3"}
	query := &consultaCeseLotePrueba{filas: &filasCeseLotePrueba{filas: []filaCeseLotePrueba{
		{ref: refs[0], pendiente: true, pendienteDesde: sql.NullTime{Time: corte.Add(-time.Hour), Valid: true}},
		{ref: refs[1], efecto: sql.NullTime{Time: time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC), Valid: true},
			disponible:  sql.NullTime{Time: time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC), Valid: true},
			restringida: true, cesado: true, pendiente: true,
			pendienteDesde: sql.NullTime{Time: corte.Add(-30 * time.Minute), Valid: true}},
	}}}
	esperadas := map[string]struct{}{refs[0]: {}, refs[1]: {}, refs[2]: {}}
	salida, err := consultarEstadosCeseLote(context.Background(), query, refs, esperadas, corte, map[string]ports.EstadoCese{})
	if err != nil || query.n != 1 || !strings.Contains(query.sql, "consultar_estado_cese_bolsa_lote_v2") || len(query.args) != 2 || len(salida) != 2 || !query.filas.closed {
		t.Fatalf("lote no ejecutado en una consulta: n=%d salida=%+v err=%v", query.n, salida, err)
	}
	if !salida[refs[0]].CesePendiente || !salida[refs[0]].FechaEfecto.IsZero() ||
		!salida[refs[0]].PendienteDesde.Equal(corte.Add(-time.Hour)) ||
		!salida[refs[1]].CesePendiente || salida[refs[1]].FechaEfecto.IsZero() ||
		!salida[refs[1]].PendienteDesde.Equal(corte.Add(-30*time.Minute)) {
		t.Fatalf("estado pendiente o B45 previo alterado: %+v", salida)
	}
	if _, hay := salida[refs[2]]; hay {
		t.Fatal("la ausencia de cese debe conservar la situación legada")
	}
}

func TestEstadoCeseLotePropagaErrorYRechazaFilaAjena(t *testing.T) {
	corte := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	ref := "participacion:1"
	esperadas := map[string]struct{}{ref: {}}
	fallo := errors.New("lectura interrumpida")
	consulta := &consultaCeseLotePrueba{err: fallo}
	if _, err := consultarEstadosCeseLote(context.Background(), consulta, []string{ref}, esperadas, corte, map[string]ports.EstadoCese{}); !errors.Is(err, fallo) || !errors.Is(err, ports.ErrConsultaEstadoCeseNoDisponible) {
		t.Fatalf("error SQL silenciado: %v", err)
	}
	consulta = &consultaCeseLotePrueba{filas: &filasCeseLotePrueba{filas: []filaCeseLotePrueba{{ref: "participacion:ajena", pendiente: true,
		pendienteDesde: sql.NullTime{Time: corte.Add(-time.Hour), Valid: true}}}}}
	if _, err := consultarEstadosCeseLote(context.Background(), consulta, []string{ref}, esperadas, corte, map[string]ports.EstadoCese{}); !errors.Is(err, ports.ErrConsultaEstadoCeseNoDisponible) {
		t.Fatalf("fila ajena aceptada: %v", err)
	}
	consulta = &consultaCeseLotePrueba{filas: &filasCeseLotePrueba{err: fallo}}
	if _, err := consultarEstadosCeseLote(context.Background(), consulta, []string{ref}, esperadas, corte, map[string]ports.EstadoCese{}); !errors.Is(err, fallo) {
		t.Fatalf("error de filas silenciado: %v", err)
	}
}
