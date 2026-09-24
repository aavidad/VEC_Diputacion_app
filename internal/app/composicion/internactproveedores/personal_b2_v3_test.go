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
