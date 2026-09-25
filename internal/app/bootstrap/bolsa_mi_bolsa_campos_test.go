package bootstrap

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

func TestCamposPortalMiBolsaSaleDelCatalogoDeEjemplo(t *testing.T) {
	sinCatalogo, err := camposPortalMiBolsaDesarrollo(configuracionDesarrolloReglasEjemplo("", ""), relojPresentacionReglasEjemplo)
	if err != nil || sinCatalogo != nil {
		t.Fatalf("sin catálogo debe quedar la conducta actual: %v %v", sinCatalogo, err)
	}
	campos, err := camposPortalMiBolsaDesarrollo(configuracionDesarrolloReglasEjemplo(rutaReglasBolsaEjemploPrueba, ""), relojPresentacionReglasEjemplo)
	if err != nil || campos == nil {
		t.Fatalf("catálogo de ejemplo no compuesto: %v", err)
	}
	lista, err := campos.CamposVisiblesMiBolsa(t.Context())
	if err != nil || !slices.Equal(lista, puertosbolsa.CamposPortalMiBolsaTodos()) {
		t.Fatalf("campos del ejemplo: %v %v", lista, err)
	}
}

func TestCamposPortalMiBolsaSigueAlCatalogoYRechazaListasRotas(t *testing.T) {
	original, err := os.ReadFile(rutaReglasBolsaEjemploPrueba)
	if err != nil {
		t.Fatal(err)
	}
	const valor = `"valor": "bolsa,posicion,estado,ultimo_llamamiento,contratos,fecha_disponible"`
	if strings.Count(string(original), valor) != 1 {
		t.Fatal("la regla de campos del portal cambió de forma")
	}
	escribir := func(nuevo string) string {
		ruta := filepath.Join(t.TempDir(), "bolsa.demo.json")
		if err := os.WriteFile(ruta, []byte(strings.Replace(string(original), valor, nuevo, 1)), 0o600); err != nil {
			t.Fatal(err)
		}
		return ruta
	}
	reducida, err := camposPortalMiBolsaDesarrollo(configuracionDesarrolloReglasEjemplo(escribir(`"valor": "bolsa,estado"`), ""), relojPresentacionReglasEjemplo)
	if err != nil {
		t.Fatal(err)
	}
	if lista, err := reducida.CamposVisiblesMiBolsa(t.Context()); err != nil || !slices.Equal(lista, []string{"bolsa", "estado"}) {
		t.Fatalf("no sigue al catálogo: %v %v", lista, err)
	}
	for _, roto := range []string{`"valor": "posicion,estado"`, `"valor": "bolsa,puntuacion"`, `"valor": "bolsa,bolsa"`} {
		if _, err := camposPortalMiBolsaDesarrollo(configuracionDesarrolloReglasEjemplo(escribir(roto), ""), relojPresentacionReglasEjemplo); !errors.Is(err, errReglasEjemploNoValidas) {
			t.Fatalf("%s admitido: %v", roto, err)
		}
	}
}
