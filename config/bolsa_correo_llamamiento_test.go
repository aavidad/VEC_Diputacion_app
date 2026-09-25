package config

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestCatalogoCorreoLlamamientoBolsaEmbebidoYSustituible(t *testing.T) {
	t.Setenv(EnvCatalogoCorreoLlamamientoBolsa, "")
	embebido, err := CatalogoCorreoLlamamientoBolsa()
	if err != nil || !bytes.Contains(embebido, []byte(`"plantilla_vigente"`)) {
		t.Fatalf("catálogo embebido: %v", err)
	}
	embebido[0] = 'X'
	if otra, _ := CatalogoCorreoLlamamientoBolsa(); otra[0] == 'X' {
		t.Fatal("el catálogo embebido debe entregarse por copia")
	}
	ruta := filepath.Join(t.TempDir(), "catalogo.json")
	if err := os.WriteFile(ruta, []byte(`{"propio":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(EnvCatalogoCorreoLlamamientoBolsa, ruta)
	if datos, err := CatalogoCorreoLlamamientoBolsa(); err != nil || string(datos) != `{"propio":true}` {
		t.Fatalf("sustitución por fichero: %q %v", datos, err)
	}
	t.Setenv(EnvCatalogoCorreoLlamamientoBolsa, filepath.Join(t.TempDir(), "no-existe.json"))
	if _, err := CatalogoCorreoLlamamientoBolsa(); err == nil {
		t.Fatal("una ruta inexistente no debe caer al catálogo embebido")
	}
	if err := os.WriteFile(ruta, make([]byte, maximoCatalogoCorreoLlamamientoBolsa+1), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(EnvCatalogoCorreoLlamamientoBolsa, ruta)
	if _, err := CatalogoCorreoLlamamientoBolsa(); err == nil {
		t.Fatal("un catálogo excesivo debe rechazarse")
	}
}
