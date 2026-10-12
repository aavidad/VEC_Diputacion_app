package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCatalogoExacto1065(t *testing.T) {
	ruta := "../../../data/catalogos/contratacion-temporal/plantillas_documentos.v3.json"
	c, h, _, err := cargar(context.Background(), ruta)
	if err != nil || h != huellaV3 || len(c.Entradas) != 11 {
		t.Fatalf("catálogo original: %s %v", h, err)
	}
	b, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatal(err)
	}
	cambiado := filepath.Join(t.TempDir(), "catalogo.json")
	b = []byte(strings.Replace(string(b), "Resolución de nombramiento", "Resolución distinta", 1))
	if err = os.WriteFile(cambiado, b, 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err = cargar(context.Background(), cambiado); err == nil || !strings.Contains(err.Error(), "catálogo.huella") {
		t.Fatalf("debe rechazar otro contenido válido antes de conectar: %v", err)
	}
}

func TestComprobarNoPuedeConvertirseEnAplicar(t *testing.T) {
	for _, args := range [][]string{
		{"--catalogo", "v3.json", "--comprobar", "--comprobar=false", "--aplicar", "PUBLICA-PLANTILLAS-V3"},
		{"--catalogo", "v3.json", "--comprobar=false"},
		{"--catalogo", "v3.json", "--aplicar", "OTRA-ORDEN"},
	} {
		if _, _, err := argumentos(args); err == nil {
			t.Errorf("aceptó %v", args)
		}
	}
	if _, solo, err := argumentos([]string{"--catalogo", "v3.json", "--comprobar"}); err != nil || !solo {
		t.Fatalf("comprobar: %v", err)
	}
}
