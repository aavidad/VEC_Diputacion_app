package internactproveedores

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/personal/domain"
)

func TestMaterialPersonalB2RechazaInventarioInseguro(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := CargarMaterialPersonalB2(dir); !errors.Is(err, ErrPersonalB2V3NoDisponible) {
		t.Fatalf("inventario ausente: %v", err)
	}
	ruta := filepath.Join(dir, "personal_b2_v3.json")
	if err := os.WriteFile(ruta, []byte(`{"version":1,"version":2}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := CargarMaterialPersonalB2(dir); !errors.Is(err, ErrPersonalB2V3NoDisponible) {
		t.Fatalf("claves duplicadas: %v", err)
	}
	if err := os.WriteFile(ruta, []byte(`{"version":1}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(ruta, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := CargarMaterialPersonalB2(dir); !errors.Is(err, ErrPersonalB2V3NoDisponible) {
		t.Fatalf("permisos abiertos: %v", err)
	}
	if err := os.Chmod(ruta, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := CargarMaterialPersonalB2(dir); !errors.Is(err, ErrPersonalB2V3NoDisponible) {
		t.Fatalf("inventario incompleto: %v", err)
	}
}

func TestProveedorPersonalB2FallaCerradoSinAutoridades(t *testing.T) {
	if _, err := ConstruirPersonalB2(context.Background(), ConfiguracionPersonalB2{}); !errors.Is(err, ErrPersonalB2V3NoDisponible) {
		t.Fatalf("constructor: %v", err)
	}
	var p *ProveedorAutorizacionPersonalB2
	if _, err := p.AutorizarConsultaRegistroEmpleadoB2(context.Background(), domain.MaterialConsultaRegistroEmpleadoB2{}); !errors.Is(err, ErrPersonalB2V3NoDisponible) {
		t.Fatalf("consulta: %v", err)
	}
	if _, err := p.AutorizarActoRegistroEmpleadoB2(context.Background(), domain.MaterialActoRegistroEmpleadoB2{}); !errors.Is(err, ErrPersonalB2V3NoDisponible) {
		t.Fatalf("acto: %v", err)
	}
	if _, err := p.AutorizarCatalogoRegistroEmpleadoB2(context.Background(), domain.MaterialCatalogoEmpleadoB2{}); !errors.Is(err, ErrPersonalB2V3NoDisponible) {
		t.Fatalf("catálogo: %v", err)
	}
}

func TestMaterialPersonalB2VersionCuatroExigeCatalogoYListaDeEmpleados(t *testing.T) {
	// El inventario real exige estar fuera de cualquier repositorio Git.
	dir, err := os.MkdirTemp("/var/tmp", "personal-b2-v3-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(dir, "personal_b2_v3.json")
	referencia := map[string]any{"catalogo_id": "motivos.b2", "catalogo_version": 1, "catalogo_huella_sha256": strings.Repeat("a", 64), "entrada_clave": "motivo_0123456789abcdef0123456789abcdef"}
	motivos := map[string]any{"ficha": referencia, "vacantes": referencia, "alta": referencia, "hecho": referencia}
	capacidades := map[string]any{"ficha": map[string]any{"archivo": "ficha.key"}, "vacantes": map[string]any{"archivo": "vacantes.key"}, "alta": map[string]any{"archivo": "alta.key"}, "hecho": map[string]any{"archivo": "hecho.key"}}
	inventario := map[string]any{"version": 4, "catalogo_motivos": "motivos.b2", "motivos": motivos, "v3": map[string]any{"capacidades": capacidades}}
	escribir := func() {
		t.Helper()
		b, err := json.Marshal(inventario)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(ruta, b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	escribir()
	if _, err := CargarMaterialPersonalB2(dir); !errors.Is(err, ErrPersonalB2V3NoDisponible) {
		t.Fatalf("motivos de catálogo ausentes: %v", err)
	}
	for _, clave := range []string{"catalogo_consultar", "catalogo_publicar", "catalogo_retirar"} {
		motivos[clave] = referencia
	}
	escribir()
	if _, err := CargarMaterialPersonalB2(dir); !errors.Is(err, ErrPersonalB2V3NoDisponible) {
		t.Fatalf("claves de catálogo ausentes: %v", err)
	}
	for _, clave := range []string{"catalogo_consultar", "catalogo_publicar", "catalogo_retirar"} {
		capacidades[clave] = map[string]any{"archivo": clave + ".key"}
	}
	escribir()
	if _, err := CargarMaterialPersonalB2(dir); !errors.Is(err, ErrPersonalB2V3NoDisponible) {
		t.Fatalf("lista de empleados ausente: %v", err)
	}
	motivos["empleados"] = referencia
	capacidades["empleados"] = map[string]any{"archivo": "empleados.key"}
	escribir()
	m, err := CargarMaterialPersonalB2(dir)
	if err != nil {
		t.Fatalf("inventario completo: %v", err)
	}
	if err := m.Cerrar(); err != nil {
		t.Fatal(err)
	}
	// El formato 3 (raíz y configuración B2 propias) ya no se admite.
	for _, version := range []int{2, 3} {
		inventario["version"] = version
		escribir()
		if _, err := CargarMaterialPersonalB2(dir); !errors.Is(err, ErrPersonalB2V3NoDisponible) {
			t.Fatalf("formato anterior %d: %v", version, err)
		}
	}
}

func TestMaterialPersonalB2NoExponeMaterialEnSerializacion(t *testing.T) {
	m := MaterialPersonalB2{CatalogoMotivos: "catalogo_privado_prueba"}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(m.String()+m.GoString()+string(b), "catalogo_privado_prueba") {
		t.Fatal("material expuesto")
	}
}
