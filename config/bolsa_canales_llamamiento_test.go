package config

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestCatalogoCanalesLlamamientoBolsaEmbebidoYSustituible(t *testing.T) {
	t.Setenv(EnvCatalogoCanalesLlamamientoBolsa, "")
	embebido, err := CatalogoCanalesLlamamientoBolsa()
	if err != nil || !bytes.Contains(embebido, []byte(`"bolsa-canales-llamamiento-v1"`)) {
		t.Fatalf("catálogo embebido: %v", err)
	}
	embebido[0] = 'X'
	if otra, _ := CatalogoCanalesLlamamientoBolsa(); otra[0] == 'X' {
		t.Fatal("el catálogo embebido debe entregarse por copia")
	}
	ruta := filepath.Join(t.TempDir(), "catalogo.json")
	if err := os.WriteFile(ruta, []byte(`{"propio":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(EnvCatalogoCanalesLlamamientoBolsa, ruta)
	if datos, err := CatalogoCanalesLlamamientoBolsa(); err != nil || string(datos) != `{"propio":true}` {
		t.Fatalf("sustitución por fichero: %q %v", datos, err)
	}
	t.Setenv(EnvCatalogoCanalesLlamamientoBolsa, filepath.Join(t.TempDir(), "no-existe.json"))
	if _, err := CatalogoCanalesLlamamientoBolsa(); err == nil {
		t.Fatal("una ruta inexistente no debe caer al catálogo embebido")
	}
	if err := os.WriteFile(ruta, make([]byte, maximoCatalogoCanalesLlamamientoBolsa+1), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(EnvCatalogoCanalesLlamamientoBolsa, ruta)
	if _, err := CatalogoCanalesLlamamientoBolsa(); err == nil {
		t.Fatal("un catálogo excesivo debe rechazarse")
	}
}
