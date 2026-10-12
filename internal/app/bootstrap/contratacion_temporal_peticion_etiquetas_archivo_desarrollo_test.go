package bootstrap

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestEtiquetasHistoricasCentroSonOpcionalesYNoContienenAutoridad(t *testing.T) {
	root := t.TempDir()
	if etiquetas, err := cargarEtiquetasHistoricasPeticionCentro(root); err != nil || etiquetas != nil {
		t.Fatal("sin catálogo se conserva la fuente actual")
	}
	if err := os.Mkdir(filepath.Join(root, "identidad"), 0700); err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(root, "identidad", "centros-etiquetas.json")
	valido := `{"version":1,"fuente_ref":"fuente:ct:archivo-centros","entradas":[{"actor_ref":"per_antiguo","nombre":"Manuel García Robles"}]}`
	if err := os.WriteFile(ruta, []byte(valido), 0600); err != nil {
		t.Fatal(err)
	}
	r, err := cargarEtiquetasHistoricasPeticionCentro(root)
	if err != nil || len(r) != 1 || r["per_antiguo"] != "Manuel García Robles" {
		t.Fatalf("etiquetas: %v %v", r, err)
	}
	// Las peticiones conservadas también usan el espacio de referencias del
	// resolvedor original: no exigir el prefijo de la provisión más reciente.
	conservado := `{"version":1,"fuente_ref":"fuente:ct:archivo-centros","entradas":[{"actor_ref":"desarrollo:centro:solicitante","nombre":"Lucía Fernández Castillo"}]}`
	if err := os.WriteFile(ruta, []byte(conservado), 0600); err != nil {
		t.Fatal(err)
	}
	if r, err := cargarEtiquetasHistoricasPeticionCentro(root); err != nil || r["desarrollo:centro:solicitante"] != "Lucía Fernández Castillo" {
		t.Fatal("debe admitir las referencias válidas conservadas")
	}
	for _, contenido := range []string{
		`{"version":1,"fuente_ref":"fuente:ct:archivo-centros","entradas":[{"actor_ref":"per_antiguo","nombre":"Manuel García Robles","role":"ratificador_centro"}]}`,
		`{"version":1,"fuente_ref":"fuente:ct:archivo-centros","entradas":[{"actor_ref":"per_antiguo","nombre":"Manuel García Robles"},{"actor_ref":"per_antiguo","nombre":"Otra persona"}]}`,
		`{"version":1,"version":1,"fuente_ref":"fuente:ct:archivo-centros","entradas":[]}`,
		`{"version":1,"fuente_ref":"","entradas":[]}`,
		`{"version":1,"fuente_ref":"fuente:ct:archivo-centros","entradas":[{"actor_ref":"per_antiguo","nombre":"Manuel\nGarcía"}]}`,
	} {
		if err := os.WriteFile(ruta, []byte(contenido), 0600); err != nil {
			t.Fatal(err)
		}
		if r, err := cargarEtiquetasHistoricasPeticionCentro(root); !errors.Is(err, errEtiquetasHistoricasPeticionCentro) || r != nil {
			t.Fatal("catálogo inválido no debe aportar etiquetas")
		}
	}
	if err := os.WriteFile(ruta, []byte(valido), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(ruta, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := cargarEtiquetasHistoricasPeticionCentro(root); !errors.Is(err, errEtiquetasHistoricasPeticionCentro) {
		t.Fatal("no admite etiquetas con permisos públicos")
	}
}
