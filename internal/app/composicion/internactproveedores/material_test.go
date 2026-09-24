package internactproveedores

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCargarMaterialFallaCerradoAnteArchivoAusenteYPermisos(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := CargarMaterial(dir); !errors.Is(err, ErrMaterialCTNoDisponible) {
		t.Fatalf("archivo ausente: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ct_v3.json"), []byte(`{"version":1}`), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := CargarMaterial(dir); !errors.Is(err, ErrMaterialCTNoDisponible) {
		t.Fatalf("archivo legible por terceros: %v", err)
	}
	if err := os.Chmod(filepath.Join(dir, "ct_v3.json"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := CargarMaterial(dir); !errors.Is(err, ErrMaterialCTNoDisponible) {
		t.Fatalf("inventario incompleto: %v", err)
	}
}

func TestCargarMaterialRechazaClaveDuplicadaAntesDeInterpretarla(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	b := []byte(`{"version":1,"version":2}`)
	if err := os.WriteFile(filepath.Join(dir, "ct_v3.json"), b, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := CargarMaterial(dir); !errors.Is(err, ErrMaterialCTNoDisponible) {
		t.Fatalf("clave duplicada: %v", err)
	}
}

func TestMaterialNoSerializaDSN(t *testing.T) {
	m := Material{Pools: map[string]PoolMaterial{"privado": {DSN: "DSN_PRIVADO_PRUEBA", Login: "cuenta_privada"}}}
	texto := m.String() + m.GoString()
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	texto += string(b)
	if strings.Contains(texto, "DSN_PRIVADO_PRUEBA") || strings.Contains(texto, "cuenta_privada") {
		t.Fatal("material privado expuesto")
	}
}
