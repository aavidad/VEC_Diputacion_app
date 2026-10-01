package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestCLISoloEscenarioSinteticoEIdiomaReal(t *testing.T) {
	args := []string{"../../web/static/textos/es/cronos-informe-saldo.json", "../../internal/modules/cronos/adapters/informesaldo/escenario_sintetico.json"}
	var salida bytes.Buffer
	if err := ejecutar(context.Background(), args, &salida); err != nil || !bytes.HasPrefix(salida.Bytes(), []byte("%PDF-")) {
		t.Fatal(err)
	}
	args[0] = "../../web/static/textos/en/cronos-informe-saldo.json"
	salida.Reset()
	if err := ejecutar(context.Background(), args, &salida); err == nil || salida.Len() != 0 {
		t.Fatal("idioma discordante publico bytes")
	}
	fixture := filepath.Join(t.TempDir(), "sin-demo.json")
	if err := os.WriteFile(fixture, []byte(`{"demo":false}`), 0600); err != nil {
		t.Fatal(err)
	}
	args[0] = "../../web/static/textos/es/cronos-informe-saldo.json"
	args[1] = fixture
	salida.Reset()
	if err := ejecutar(context.Background(), args, &salida); err == nil || salida.Len() != 0 {
		t.Fatal("datos sin marca sintetica aceptados")
	}
}
