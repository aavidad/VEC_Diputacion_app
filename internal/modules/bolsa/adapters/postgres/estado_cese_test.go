package postgres

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

type filaEstadoCesePrueba struct {
	efecto, disponible, pendienteDesde sql.NullTime
	restringida, cesado, pendiente     bool
	err                                error
}

func (f filaEstadoCesePrueba) Scan(destinos ...any) error {
	if f.err != nil {
		return f.err
	}
	*destinos[0].(*sql.NullTime) = f.efecto
	*destinos[1].(*sql.NullTime) = f.disponible
	*destinos[2].(*bool) = f.restringida
	*destinos[3].(*bool) = f.cesado
	*destinos[4].(*bool) = f.pendiente
	*destinos[5].(*sql.NullTime) = f.pendienteDesde
	return nil
}

func TestEstadoCeseLeeFechaRecienteYMaximoSinExponerCandidato(t *testing.T) {
	corte := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)
	fila := filaEstadoCesePrueba{efecto: sql.NullTime{Time: time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC), Valid: true},
		disponible: sql.NullTime{Time: time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC), Valid: true}, restringida: true, cesado: true}
	estado, presente, err := escanearEstadoCese(fila, corte)
	if err != nil || !presente || estado.FechaEfecto.In(time.UTC).Day() != 29 || estado.DisponibleDesde.In(time.UTC).Day() != 30 {
		t.Fatalf("fechas locales B45 alteradas: %+v %v %v", estado, presente, err)
	}
	if _, presente, err := escanearEstadoCese(filaEstadoCesePrueba{err: pgx.ErrNoRows}, corte); presente || err != nil {
		t.Fatalf("ausencia de cese: %v %v", presente, err)
	}
	pendiente := filaEstadoCesePrueba{pendiente: true,
		pendienteDesde: sql.NullTime{Time: corte.Add(-time.Hour), Valid: true}}
	if estado, presente, err := escanearEstadoCese(pendiente, corte); err != nil || !presente || !estado.CesePendiente ||
		!estado.FechaEfecto.IsZero() || !estado.DisponibleDesde.IsZero() || !estado.PendienteDesde.Equal(corte.Add(-time.Hour)) {
		t.Fatalf("pendiente sin fecha inventada: %+v %v %v", estado, presente, err)
	}
	for _, invalida := range []sql.NullTime{{}, {Time: corte.Add(time.Hour), Valid: true}} {
		pendiente.pendienteDesde = invalida
		if _, presente, err := escanearEstadoCese(pendiente, corte); presente || !errors.Is(err, ports.ErrConsultaEstadoCeseNoDisponible) {
			t.Fatalf("instante pendiente ausente o futuro aceptado: %+v %v", invalida, err)
		}
	}
	pendiente.pendienteDesde = sql.NullTime{Time: corte.Add(-time.Hour), Valid: true}
	fila.pendiente = true
	fila.pendienteDesde = pendiente.pendienteDesde
	if estado, presente, err := escanearEstadoCese(fila, corte); err != nil || !presente || !estado.CesePendiente || estado.FechaEfecto.IsZero() {
		t.Fatalf("pendiente con cese previo acreditado: %+v %v %v", estado, presente, err)
	}
	fila.pendiente = false
	fila.pendienteDesde = sql.NullTime{}
	fila.disponible = fila.efecto
	if _, presente, err := escanearEstadoCese(fila, corte); presente || !errors.Is(err, ports.ErrConsultaEstadoCeseNoDisponible) {
		t.Fatalf("restricción vencida aceptada: %v %v", presente, err)
	}
}
