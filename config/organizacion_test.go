package config

import "testing"

func TestConfiguracionOrganizacionEsExplicitaYVersionada(t *testing.T) {
	if (Config{}).Normalize().PersonalOrganizacionSourcePath != "" {
		t.Fatal("no se debe cargar una organización implícita")
	}
	t.Setenv(EnvPersonalOrganizacionSourcePath, "data/catalogos/estructura-organizativa/v2.json")
	t.Setenv(EnvPersonalOrganizacionVersion, "2")
	cfg := Load().Normalize()
	if cfg.PersonalOrganizacionSourcePath != "data/catalogos/estructura-organizativa/v2.json" || cfg.PersonalOrganizacionVersion != 2 {
		t.Fatal("la configuración debe seleccionar la versión sin recompilar")
	}
}
