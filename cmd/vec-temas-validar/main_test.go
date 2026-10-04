package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLI(t *testing.T) {
	path := filepath.Join("..", "..", "data", "temas", "politica-v1.json")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(b)
	args := []string{"-politica", path, "-politica-sha256", hex.EncodeToString(sum[:])}
	packageBytes, err := os.ReadFile(filepath.Join("..", "..", "data", "temas", "institucional-v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	var out, diagnostico bytes.Buffer
	if ejecutar(args, bytes.NewReader(packageBytes), &out, &diagnostico) != 0 || diagnostico.Len() != 0 || !json.Valid(out.Bytes()) {
		t.Fatal("success", diagnostico.String())
	}
	for _, flags := range [][]string{nil, {"-h"}, {"-politica", path}, {"-politica", path, "-politica-sha256", strings.Repeat("0", 64)}} {
		out.Reset()
		diagnostico.Reset()
		if ejecutar(flags, bytes.NewReader(packageBytes), &out, &diagnostico) == 0 || out.Len() != 0 || !json.Valid(diagnostico.Bytes()) || strings.Contains(diagnostico.String(), path) {
			t.Fatal("closed error")
		}
	}
	out.Reset()
	diagnostico.Reset()
	if ejecutar(args, strings.NewReader(`{"css":"secret-content"}`), &out, &diagnostico) == 0 || out.Len() != 0 || strings.Contains(diagnostico.String(), "secret-content") {
		t.Fatal("input disclosure")
	}
	if ejecutar(args, bytes.NewReader(packageBytes), escritorCorto{}, io.Discard) == 0 {
		t.Fatal("short writer")
	}
	if ejecutar(nil, bytes.NewReader(packageBytes), io.Discard, escritorCorto{}) != MotivoFalloDiagnostico {
		t.Fatal("diagnostic failure")
	}
}

type escritorCorto struct{}

func (escritorCorto) Write([]byte) (int, error) { return 0, nil }

func TestSalidaCSSConservaValidacion(t *testing.T) {
	path := filepath.Join("..", "..", "data", "temas", "politica-v1.json")
	politica, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(politica)
	args := []string{"-politica", path, "-politica-sha256", hex.EncodeToString(sum[:]), "-css"}
	paquete, err := os.ReadFile(filepath.Join("..", "..", "data", "temas", "institucional-v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	var salida, diagnostico bytes.Buffer
	if ejecutar(args, bytes.NewReader(paquete), &salida, &diagnostico) != 0 || diagnostico.Len() != 0 {
		t.Fatal("no se pudo preparar la hoja", diagnostico.String())
	}
	if !strings.HasPrefix(salida.String(), "@media (forced-colors: none)") || strings.Contains(salida.String(), "\"material\"") {
		t.Fatal("la salida CSS mezcla el informe con la hoja")
	}
	salida.Reset()
	diagnostico.Reset()
	if ejecutar(args, strings.NewReader(`{"esquema":1,"css":"input-no-admitido"}`), &salida, &diagnostico) == 0 || salida.Len() != 0 || !json.Valid(diagnostico.Bytes()) || strings.Contains(diagnostico.String(), "input-no-admitido") {
		t.Fatal("el modo CSS eludió la validación")
	}
	if ejecutar(args, bytes.NewReader(paquete), escritorCorto{}, io.Discard) == 0 {
		t.Fatal("se confirmó una escritura CSS incompleta")
	}
}
