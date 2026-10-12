package postgres

import (
	"bufio"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/documentos/ports"
)

// VEC_D16_SQL_SALIDAS apunta a la salida del ensayo SQL sintético sobre un
// clon PG18. Son respuestas de la función SQL real; sin clon el test se omite.
func TestDecodificarPreparacionYReplaySQLRealD16(t *testing.T) {
	ruta := os.Getenv("VEC_D16_SQL_SALIDAS")
	if ruta == "" {
		t.Skip("requiere salida del clon PostgreSQL 18")
	}
	f, err := os.Open(ruta)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var primera, replay ports.PlanIngestaMayan
	s := bufio.NewScanner(f)
	s.Buffer(make([]byte, 4096), 1<<20)
	for s.Scan() {
		linea := s.Text()
		for _, marca := range []struct {
			prefijo string
			destino *ports.PlanIngestaMayan
		}{
			{"D16_PLAN_JSON=", &primera}, {"D16_REPLAY_JSON=", &replay},
		} {
			pos := strings.Index(linea, marca.prefijo)
			if pos < 0 {
				continue
			}
			var salida struct {
				Plan ports.PlanIngestaMayan `json:"plan"`
			}
			if err := json.Unmarshal([]byte(linea[pos+len(marca.prefijo):]), &salida); err != nil {
				t.Fatalf("%s: %v", marca.prefijo, err)
			}
			*marca.destino = salida.Plan
		}
	}
	if err := s.Err(); err != nil {
		t.Fatal(err)
	}
	if primera.ValidarPersistido() != nil || replay.ValidarPersistido() != nil ||
		primera.PlanRef != replay.PlanRef || primera.CargaID != replay.CargaID ||
		primera.CreadoEn.IsZero() || primera.ActualizadoEn.IsZero() ||
		!primera.CreadoEn.Equal(replay.CreadoEn) || !primera.ActualizadoEn.Equal(replay.ActualizadoEn) {
		t.Fatalf("respuesta SQL no recuperable: plan=%+v replay=%+v", primera, replay)
	}
}
