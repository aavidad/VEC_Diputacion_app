package admincanon_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"testing"

	"vec-diputacion-granada/internal/vec/domain"
)

// The same test consumes fixed E vectors and the output of the real SQL
// serializer. Domain validation and hashes use the production types.
func TestCanonAdminConTiposCentrales(t *testing.T) {
	ruta := os.Getenv("VEC_ADMIN_CANON_VECTORES")
	if ruta == "" {
		ruta = "vectores.jsonl"
	}
	f, err := os.Open(ruta)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 4096), 1<<20)
	n := 0
	for scanner.Scan() {
		n++
		var v struct {
			Tipo   string          `json:"tipo"`
			Input  json.RawMessage `json:"input"`
			Canon  string          `json:"canon"`
			SHA256 string          `json:"sha256"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &v); err != nil {
			t.Fatal(err)
		}
		var destino interface{ HuellaSHA256() (string, error) }
		switch v.Tipo {
		case "asignacion":
			destino = &domain.AsignacionPerfil{}
		case "rol":
			destino = &domain.VersionRol{}
		case "control":
			destino = &domain.ControlVigenciaVersionRol{}
		default:
			t.Fatalf("tipo desconocido: %s", v.Tipo)
		}
		d := json.NewDecoder(bytes.NewReader(v.Input))
		d.DisallowUnknownFields()
		if err := d.Decode(destino); err != nil {
			t.Fatalf("vector %d: %v", n, err)
		}
		if err := d.Decode(new(any)); err != io.EOF {
			t.Fatalf("vector %d: JSON adicional", n)
		}
		canon, err := json.Marshal(destino)
		if err != nil {
			t.Fatal(err)
		}
		if string(canon) != v.Canon {
			t.Fatalf("vector %d: canon SQL/Go divergente", n)
		}
		huella, err := destino.HuellaSHA256()
		if err != nil {
			t.Fatalf("vector %d: dominio rechaza documento: %v", n, err)
		}
		if huella != v.SHA256 {
			t.Fatalf("vector %d: SHA256 SQL/Go divergente", n)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	if n < 6 {
		t.Fatalf("solo %d vectores", n)
	}
}
