package numeracion

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCargarPoliticaMOADSinSustituirFallos(t *testing.T) {
	p, err := Cargar("")
	if err != nil || p.ValidarNumero("2026/5487") != nil {
		t.Fatalf("catálogo distribuido: %v", err)
	}
	ruta := filepath.Join(t.TempDir(), "politica.json")
	contenido := `{"referencia":"catalogo:ct:moad","version":2,"patron":"^[0-9]{4}/EXP-[1-9][0-9]{0,9}$","ejemplo":"2026/EXP-5487"}`
	if err := os.WriteFile(ruta, []byte(contenido), 0600); err != nil {
		t.Fatal(err)
	}
	p, err = Cargar(ruta)
	if err != nil || p.Version != 2 || p.ValidarNumero("2026/EXP-5487") != nil {
		t.Fatalf("catálogo configurable: %v", err)
	}
	if err := os.WriteFile(ruta, []byte(`{"version":2}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = Cargar(ruta); err == nil {
		t.Fatal("catálogo declarado inválido reemplazado silenciosamente")
	}
	if _, err = Cargar(ruta + "-ausente"); err == nil {
		t.Fatal("catálogo declarado ausente reemplazado silenciosamente")
	}
}
