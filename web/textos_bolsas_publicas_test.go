package web

import (
	"encoding/json"
	"io/fs"
	"path"
	"testing"
)

func TestCatalogosBolsasPublicasMantienenClavesYTraducciones(t *testing.T) {
	archivos, err := fs.Glob(textosBolsasPublicas, "static/textos/*/bolsas-publicas.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(archivos) != 2 {
		t.Fatalf("faltan catalogos ES/EN: %v", archivos)
	}
	catalogo, err := CatalogoBolsasPublicas()
	if err != nil {
		t.Fatal(err)
	}
	var claves map[string]string
	for _, archivo := range archivos {
		contenido, err := textosBolsasPublicas.ReadFile(archivo)
		if err != nil {
			t.Fatal(err)
		}
		var mensajes map[string]string
		if err := json.Unmarshal(contenido, &mensajes); err != nil {
			t.Fatal(err)
		}
		if claves == nil {
			claves = mensajes
		}
		if len(claves) != len(mensajes) {
			t.Fatalf("claves distintas: %s", archivo)
		}
		for clave := range claves {
			if mensajes[clave] == "" || catalogo.T(path.Base(path.Dir(archivo)), clave) != mensajes[clave] {
				t.Fatalf("traduccion ausente: %s %s", archivo, clave)
			}
		}
	}
}
