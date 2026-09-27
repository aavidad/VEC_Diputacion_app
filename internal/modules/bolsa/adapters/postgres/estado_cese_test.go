package postgres

import (
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

type filaEstadoCesePrueba struct {
	efecto, disponible  time.Time
	restringida, cesado bool
	err                 error
}

func (f filaEstadoCesePrueba) Scan(destinos ...any) error {
	if f.err != nil {
		return f.err
	}
	*destinos[0].(*time.Time) = f.efecto
	*destinos[1].(*time.Time) = f.disponible
	*destinos[2].(*bool) = f.restringida
	*destinos[3].(*bool) = f.cesado
	return nil
}

func TestEstadoCeseLeeFechaRecienteYMaximoSinExponerCandidato(t *testing.T) {
	corte := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)
	fila := filaEstadoCesePrueba{efecto: time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC),
		disponible: time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC), restringida: true, cesado: true}
	estado, presente, err := escanearEstadoCese(fila, corte)
	if err != nil || !presente || estado.FechaEfecto.In(time.UTC).Day() != 29 || estado.DisponibleDesde.In(time.UTC).Day() != 30 {
		t.Fatalf("fechas locales B45 alteradas: %+v %v %v", estado, presente, err)
	}
	if _, presente, err := escanearEstadoCese(filaEstadoCesePrueba{err: pgx.ErrNoRows}, corte); presente || err != nil {
		t.Fatalf("ausencia de cese: %v %v", presente, err)
	}
	fila.disponible = fila.efecto
	if _, presente, err := escanearEstadoCese(fila, corte); presente || !errors.Is(err, ports.ErrConsultaEstadoCeseNoDisponible) {
		t.Fatalf("restricción vencida aceptada: %v %v", presente, err)
	}
}
