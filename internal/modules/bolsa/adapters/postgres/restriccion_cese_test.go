package postgres

import (
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

type filaRestriccionPrueba struct {
	disponible time.Time
	efecto     time.Time
	recibo     string
	version    int64
	err        error
}

func (f filaRestriccionPrueba) Scan(destinos ...any) error {
	if f.err != nil {
		return f.err
	}
	*destinos[0].(*time.Time) = f.disponible
	*destinos[1].(*time.Time) = f.efecto
	*destinos[2].(*string) = f.recibo
	*destinos[3].(*int64) = f.version
	return nil
}

func TestLecturaRestriccionCeseRespetaFechaMadridYFallaCerrada(t *testing.T) {
	corte := time.Date(2026, 10, 24, 23, 30, 0, 0, time.UTC) // 25 de octubre en Granada.
	fila := filaRestriccionPrueba{
		disponible: time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC),
		efecto:     time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC),
		recibo:     "recibo:cese:1", version: 3,
	}
	r, activa, err := escanearRestriccionCese(fila, corte)
	if err != nil || !activa || r.PoliticaVersion != 3 || r.DisponibleDesde.UTC() != (time.Date(2026, 10, 25, 23, 0, 0, 0, time.UTC)) || r.FechaEfecto.UTC() != (time.Date(2026, 10, 24, 22, 0, 0, 0, time.UTC)) {
		t.Fatalf("fecha local B45: %+v, activa=%v, err=%v", r, activa, err)
	}
	if _, activa, err := escanearRestriccionCese(filaRestriccionPrueba{err: pgx.ErrNoRows}, corte); activa || err != nil {
		t.Fatalf("ausencia de restricción: activa=%v err=%v", activa, err)
	}
	fila.disponible = fila.efecto
	if _, activa, err := escanearRestriccionCese(fila, corte); activa || !errors.Is(err, ports.ErrConsultaRestriccionCeseNoDisponible) {
		t.Fatalf("fecha no activa debe fallar cerrada: activa=%v err=%v", activa, err)
	}
}
