package seleccion

import (
	"encoding/json"
	"os"
	"testing"
)

func TestManifiestoSeleccionValidoYTraducido(t *testing.T) {
	m := Manifest()
	if err := m.Validate(); err != nil {
		t.Fatalf("manifiesto de Selección inválido: %v", err)
	}
	if m.ID != ModuleID || m.BasePath != "/modules/seleccion" || len(m.Menu) != 0 {
		t.Fatal("identidad o ruta base del manifiesto inesperadas")
	}
	bruto, err := os.ReadFile("../../../locales/es.json")
	if err != nil {
		t.Fatal(err)
	}
	var catalogo map[string]string
	if err := json.Unmarshal(bruto, &catalogo); err != nil {
		t.Fatal(err)
	}
	claves := []string{m.NameKey, m.DescriptionKey}
	for _, p := range m.Permissions {
		claves = append(claves, p.LabelKey)
	}
	for _, clave := range claves {
		if catalogo[clave] == "" {
			t.Fatalf("falta la traducción %q", clave)
		}
	}
}
