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

func TestMaterialRPTPublicaV3ExigePerfilYCapacidadPropios(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(dir, "rpt_publica_v3.json")
	motivo := map[string]any{"catalogo_id": "motivos.rpt", "catalogo_version": 1, "catalogo_huella_sha256": strings.Repeat("a", 64), "entrada_clave": "motivo_0123456789abcdef0123456789abcdef"}
	doc := map[string]any{"version": 1, "catalogo_motivos": "motivos.rpt", "motivo_consulta": motivo,
		"capacidad": map[string]any{"archivo": "rpt-publica.key"},
		"perfiles":  map[string]any{"cta_0123456789abcdefghijkl": map[string]any{"perfil_activo_ref": "prf_0123456789abcdefghijkl", "perfil_version": 5}}}
	escribir := func() {
		t.Helper()
		b, err := json.Marshal(doc)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(ruta, b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	escribir()
	m, err := CargarMaterialRPTPublicaV3(dir)
	if err != nil || len(m.Perfiles) != 1 {
		t.Fatalf("material válido: %v", err)
	}
	_ = m.Cerrar()
	doc["perfiles"].(map[string]any)["cta_0123456789abcdefghijkl"].(map[string]any)["perfil_version"] = 0
	escribir()
	if _, err := CargarMaterialRPTPublicaV3(dir); !errors.Is(err, ErrRPTPublicaV3NoDisponible) {
		t.Fatalf("perfil sin versión: %v", err)
	}
	doc["perfiles"].(map[string]any)["cta_0123456789abcdefghijkl"].(map[string]any)["perfil_version"] = 5
	doc["capacidad"].(map[string]any)["archivo"] = "../capacidad.key"
	escribir()
	if _, err := CargarMaterialRPTPublicaV3(dir); !errors.Is(err, ErrRPTPublicaV3NoDisponible) {
		t.Fatalf("escape de raíz privada: %v", err)
	}
}

func TestRPTPublicaV3SinAutoridadesFallaCerrada(t *testing.T) {
	if _, err := ConstruirRPTPublicaV3(context.Background(), MaterialRPTPublicaV3{}, nil, nil, nil); !errors.Is(err, ErrRPTPublicaV3NoDisponible) {
		t.Fatalf("constructor: %v", err)
	}
	var p *ProveedorAutorizacionRPTPublicaV3
	if _, err := p.AutorizarConsultaRPTPublicaV2(context.Background(), domain.MaterialConsultaRPTPublicaV2{}); !errors.Is(err, ErrRPTPublicaV3NoDisponible) {
		t.Fatalf("consulta: %v", err)
	}
}
