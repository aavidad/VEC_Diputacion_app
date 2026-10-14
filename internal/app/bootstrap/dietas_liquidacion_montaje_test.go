package bootstrap

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vec-diputacion-granada/config"
	dietashttp "vec-diputacion-granada/internal/modules/dietas/adapters/httpinterno"
)

func TestLiquidacionEconomicaOpcionalYConfiguracionPrivada(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	identidad := filepath.Join(dir, "identidad")
	if err := os.Mkdir(identidad, 0700); err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{DevelopmentMaterialDir: dir}
	if c, err := leerConfiguracionLiquidacionDietas(cfg); c != nil || err != nil {
		t.Fatalf("ausente: %v %v", c, err)
	}
	ruta := filepath.Join(identidad, "dietas-liquidacion.json")
	valido := `{"version":1,"habilitada":true}`
	if err := os.WriteFile(ruta, []byte(valido), 0600); err != nil {
		t.Fatal(err)
	}
	c, err := leerConfiguracionLiquidacionDietas(cfg)
	if err != nil || c == nil || !c.Habilitada {
		t.Fatalf("válido: %v %v", c, err)
	}
	if err := os.Chmod(ruta, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := leerConfiguracionLiquidacionDietas(cfg); err == nil {
		t.Fatal("permisos amplios")
	}
	if err := os.Chmod(ruta, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ruta, []byte(strings.Replace(valido, `"habilitada":true`, `"habilitada":true,"habilitada":true`, 1)), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := leerConfiguracionLiquidacionDietas(cfg); err == nil {
		t.Fatal("clave duplicada")
	}
	if err := os.Remove(ruta); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(identidad, "inexistente"), ruta); err != nil {
		t.Fatal(err)
	}
	if _, err := leerConfiguracionLiquidacionDietas(cfg); err == nil {
		t.Fatal("enlace")
	}
}

func TestLiquidacionEconomicaRutasSoloFormasExactas(t *testing.T) {
	ref := "dco_" + strings.Repeat("a", 22)
	casos := []struct {
		ruta, metodo string
		acepta       bool
	}{
		{dietashttp.RutaCircuito + "/" + ref + "/liquidaciones/propuesta", "GET", true},
		{dietashttp.RutaCircuito + "/" + ref + "/liquidaciones", "POST", true},
		{dietashttp.RutaCircuito + "/" + ref + "/liquidaciones/2", "GET", true},
		{dietashttp.RutaBorradores + "/" + ref + "/liquidaciones/actual", "GET", true},
		{dietashttp.RutaBorradores + "/" + ref + "/liquidaciones", "POST", false},
		{dietashttp.RutaCircuito + "/" + ref + "/liquidaciones/01", "GET", false},
		{dietashttp.RutaCircuito + "/" + ref + "/liquidaciones/propuesta/extra", "GET", false},
	}
	for _, caso := range casos {
		if got := metodoComisionesDietasValido(caso.ruta, caso.metodo); got != caso.acepta {
			t.Fatalf("%s %s: %v", caso.metodo, caso.ruta, got)
		}
	}
}
