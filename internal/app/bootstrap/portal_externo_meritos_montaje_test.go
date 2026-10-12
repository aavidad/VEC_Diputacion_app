package bootstrap

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	vec "vec-diputacion-granada/internal/vec/domain"
)

func TestMotivoCarteraExternaExigeReferenciaPublicadaEnMaterialPrivado(t *testing.T) {
	raiz := t.TempDir()
	ruta := filepath.Join(raiz, rutaMotivoCarteraExterna)
	if err := os.MkdirAll(filepath.Dir(ruta), 0o700); err != nil {
		t.Fatal(err)
	}
	referencia := vec.ReferenciaEntradaCatalogo{
		CatalogoID: "motivos_autorizacion_meritos_cartera", CatalogoVersion: 2,
		CatalogoHuellaSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		EntradaClave:         "motivo_00000000000000000000000000000001",
	}
	b, err := json.Marshal(configuracionMotivoCarteraExterna{Version: 1, MotivoRetirada: referencia})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ruta, b, 0o600); err != nil {
		t.Fatal(err)
	}
	obtenida, err := leerMotivoCarteraExterna(raiz)
	if err != nil || obtenida != referencia {
		t.Fatalf("motivo privado valido: %v %#v", err, obtenida)
	}
	if err := os.Chmod(ruta, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := leerMotivoCarteraExterna(raiz); err == nil {
		t.Fatal("aceptó material legible por terceros")
	}
	if err := os.Chmod(ruta, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ruta, []byte(`{"version":1,"version":1}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := leerMotivoCarteraExterna(raiz); err == nil {
		t.Fatal("aceptó claves duplicadas")
	}
}
