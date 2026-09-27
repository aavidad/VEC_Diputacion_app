package postgres

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

type filaPoliticaCesePrueba struct {
	estado string
	err    error
}

func (f filaPoliticaCesePrueba) Scan(destinos ...any) error {
	if f.err != nil {
		return f.err
	}
	*destinos[0].(*int64) = 1
	*destinos[1].(*string) = "catalogo:bolsa:cese:ejemplo-sintetico:v1"
	*destinos[2].(*string) = strings.Repeat("a", 64)
	*destinos[3].(*[]byte) = []byte(`{"interinidad":"general","interinidad|acumulacion_tareas":"acumulacion_tareas"}`)
	*destinos[4].(*int32) = 5
	*destinos[5].(*int32) = 9
	*destinos[6].(*string) = "fecha_cese_meses_calendario_ajuste_fin_mes"
	*destinos[7].(*string) = f.estado
	*destinos[8].(*time.Time) = time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	return nil
}

func TestLectorPoliticaCeseCierraSQLYEstadoInesperado(t *testing.T) {
	p, err := escanearPoliticaCese(filaPoliticaCesePrueba{estado: "ejemplo_sintetico"})
	if err != nil || p.Version != 1 || p.MesesAcumulacion != 9 || p.Mapeo["interinidad"] != "general" {
		t.Fatalf("lectura B45: %+v, %v", p, err)
	}
	for _, fila := range []filaPoliticaCesePrueba{{estado: "vigente"}, {err: pgx.ErrNoRows}, {err: errors.New("detalle SQL privado")}} {
		_, err := escanearPoliticaCese(fila)
		if !errors.Is(err, ports.ErrConsultaPoliticaCeseNoDisponible) || strings.Contains(err.Error(), "privado") {
			t.Fatalf("lector no falla cerrado: %v", err)
		}
	}
}
