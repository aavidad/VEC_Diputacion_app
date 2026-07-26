package postgres

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

type vectorExpedienteValidoO404E struct {
	Caso     string            `json:"caso"`
	Agregado domain.Expediente `json:"agregado"`
}

func TestVectoresExpedienteO404ESonDominioCanonico(t *testing.T) {
	ruta := filepath.Join(
		"..", "..", "..", "..", "..", "deploy", "postgresql",
		"contratacion_temporal", "pruebas_sql",
		"o404e_expedientes_validos.jsonl",
	)
	fichero, err := os.Open(ruta)
	if err != nil {
		t.Fatal(err)
	}
	defer fichero.Close()

	esperados := map[string]bool{"uno": false, "maximo": false}
	scanner := bufio.NewScanner(fichero)
	scanner.Buffer(make([]byte, 4096), 1<<20)
	for scanner.Scan() {
		linea := append([]byte(nil), scanner.Bytes()...)
		var vector vectorExpedienteValidoO404E
		decoder := json.NewDecoder(bytes.NewReader(linea))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&vector); err != nil {
			t.Fatalf("vector O4-04E no decodificable: %v", err)
		}
		if err := exigirFinJSONO404E(decoder); err != nil {
			t.Fatalf("vector O4-04E con contenido residual: %v", err)
		}
		if _, existe := esperados[vector.Caso]; !existe || esperados[vector.Caso] {
			t.Fatalf("caso O4-04E inesperado o duplicado: %q", vector.Caso)
		}
		esperados[vector.Caso] = true
		e := vector.Agregado
		if err := e.Validar(); err != nil {
			t.Fatalf("%s: expediente de dominio inválido: %v", vector.Caso, err)
		}
		if e.Referencia != "expediente:o404e:"+vector.Caso ||
			e.NumeroVisible != "2026/"+vector.Caso ||
			e.Version != 2 || len(e.Actuaciones) != 2 ||
			e.Analisis == nil || !e.Analisis.HabilitaAvance() ||
			e.ViaCobertura != nil || len(e.DecisionesCobertura) != 0 ||
			e.Asignacion != nil {
			t.Fatalf("%s: precondición inicial O4-04E divergente", vector.Caso)
		}
		remarshal, err := json.Marshal(vector)
		if err != nil || !bytes.Equal(remarshal, linea) {
			t.Fatalf("%s: JSONL no es el marshal canónico Go: %v", vector.Caso, err)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	for caso, visto := range esperados {
		if !visto {
			t.Fatalf("falta vector O4-04E %q", caso)
		}
	}
}

func exigirFinJSONO404E(decoder *json.Decoder) error {
	var residual any
	err := decoder.Decode(&residual)
	if err == io.EOF {
		return nil
	}
	if err == nil {
		return errors.New("segundo valor JSON")
	}
	return err
}
