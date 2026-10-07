package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/seleccion/domain"
)

func TestCLICotejaSalidaExactaDelTribunal(t *testing.T) {
	material, err := os.ReadFile(filepath.Join("..", "vec-selectivos-preparar-tribunal", "testdata", "material.json"))
	if err != nil {
		t.Fatal(err)
	}
	var propuesto domain.MaterialTribunalPropuesto
	if err := json.Unmarshal(material, &propuesto); err != nil {
		t.Fatal(err)
	}
	preparacion, err := domain.PrepararTribunal(propuesto)
	if err != nil {
		t.Fatal(err)
	}
	var salidaTribunal bytes.Buffer
	enc := json.NewEncoder(&salidaTribunal)
	enc.SetIndent("", "  ")
	if err := enc.Encode(struct {
		Preparacion domain.PreparacionTribunal `json:"preparacion"`
		Limite      string                     `json:"limite"`
		Mensajes    []pendienteVisible         `json:"mensajes"`
	}{preparacion, "pendiente", []pendienteVisible{}}); err != nil {
		t.Fatal(err)
	}
	archivo := filepath.Join(t.TempDir(), "tribunal.json")
	if err := os.WriteFile(archivo, salidaTribunal.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	acta, err := os.ReadFile(filepath.Join("testdata", "material-cotejo.json"))
	if err != nil {
		t.Fatal(err)
	}
	var salida, errores bytes.Buffer
	args := []string{"-catalogos-dir", "../../web/static/textos", "-tribunal-salida", archivo}
	if codigo := ejecutar(context.Background(), args, bytes.NewReader(acta), &salida, &errores); codigo != 0 {
		t.Fatalf("código %d, diagnóstico %s", codigo, errores.String())
	}
	huella := sha256.Sum256(salidaTribunal.Bytes())
	huellaHex := hex.EncodeToString(huella[:])
	if !bytes.Contains(salida.Bytes(), []byte(huellaHex)) || !bytes.Contains(salida.Bytes(), []byte(`"cotejo_local": "salida_tribunal_sha256"`)) {
		t.Fatalf("falta huella de la salida exacta: %s", salida.String())
	}
	acta = bytes.Replace(acta, []byte("fase:ejemplo-sintetico"), []byte("fase:ajena"), 1)
	salida.Reset()
	errores.Reset()
	if codigo := ejecutar(context.Background(), args, bytes.NewReader(acta), &salida, &errores); codigo != 1 || salida.Len() != 0 || !strings.Contains(errores.String(), "cotejo_invalido") {
		t.Fatalf("fase ajena: código %d, salida %s, error %s", codigo, salida.String(), errores.String())
	}
}
