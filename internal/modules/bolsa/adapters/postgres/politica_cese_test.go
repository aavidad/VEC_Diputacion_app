package postgres

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

type filaPoliticaCesePrueba struct {
	estado string
	err    error
	mapeo  []byte
}

func (f filaPoliticaCesePrueba) Scan(destinos ...any) error {
	if f.err != nil {
		return f.err
	}
	*destinos[0].(*int64) = 1
	*destinos[1].(*string) = "catalogo:bolsa:cese:ejemplo-sintetico:v1"
	*destinos[2].(*string) = strings.Repeat("a", 64)
	mapeo := f.mapeo
	if mapeo == nil {
		mapeo = []byte(`{"interinidad":"general","interinidad|acumulacion_tareas":"acumulacion_tareas"}`)
	}
	*destinos[3].(*[]byte) = mapeo
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
	mapaGrande := make(map[string]string, 100)
	for i := 0; i < 100; i++ {
		mapaGrande[fmt.Sprintf("modalidad_%02d_%s", i, strings.Repeat("x", 60))] = "general"
	}
	bytes, err := json.Marshal(mapaGrande)
	if err != nil || len(bytes) <= 8192 || len(bytes) > 16384 {
		t.Fatalf("precondición 8-16 KiB: %d bytes, %v", len(bytes), err)
	}
	p, err = escanearPoliticaCese(filaPoliticaCesePrueba{estado: "ejemplo_sintetico", mapeo: bytes})
	if err != nil || len(p.Mapeo) != 100 {
		t.Fatalf("política válida de 100 entradas rechazada: %v", err)
	}
	for _, fila := range []filaPoliticaCesePrueba{{estado: "vigente"}, {err: pgx.ErrNoRows}, {err: errors.New("detalle SQL privado")}} {
		_, err := escanearPoliticaCese(fila)
		if !errors.Is(err, ports.ErrConsultaPoliticaCeseNoDisponible) || strings.Contains(err.Error(), "privado") {
			t.Fatalf("lector no falla cerrado: %v", err)
		}
	}
}
