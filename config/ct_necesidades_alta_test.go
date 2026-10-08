package config

import "testing"

func TestFuenteNecesidadesAltaSeDeclaraPorConfiguracionCanonica(t *testing.T) {
	if (Config{}).Normalize().CTNecesidadesAltaSourcePath != "" {
		t.Fatal("sin ruta declarada debe regir el paquete distribuido")
	}
	t.Setenv(EnvCTNecesidadesAltaSourcePath, "  /fuentes/ct-necesidades-v2.json  ")
	if got := Load().CTNecesidadesAltaSourcePath; got != "/fuentes/ct-necesidades-v2.json" {
		t.Fatalf("ruta no normalizada: %q", got)
	}
}
