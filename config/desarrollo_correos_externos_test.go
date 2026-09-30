package config

import (
	"path/filepath"
	"testing"
)

func TestSemillaCorreosExternaTieneRutaNominalPropia(t *testing.T) {
	raiz := t.TempDir()
	rutas := (Config{DevelopmentMaterialDir: raiz}).DevelopmentPaths()
	if obtenida, esperada := rutas.SemillaCorreosExterna, filepath.Join(raiz, "usuarios", "correos-externos-semilla.bin"); obtenida != esperada {
		t.Fatalf("semilla de correo externo fuera de su ubicación nominal: %q", obtenida)
	}
	if rutas.SemillaCorreosExterna == rutas.KMSSecret {
		t.Fatal("la semilla externa no puede usar el KMS interno")
	}
	if ruta := (Config{}).DevelopmentPaths().SemillaCorreosExterna; ruta != "" {
		t.Fatal("la semilla externa no debe inventar un directorio material")
	}
}
