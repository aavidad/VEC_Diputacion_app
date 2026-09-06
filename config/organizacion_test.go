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
	if cfg.PersonalOrganizacionPostgreSQL {
		t.Fatal("la escritura no debe activarse implícitamente")
	}
	t.Setenv(EnvPersonalOrganizacionPostgreSQL, "1")
	if !Load().Normalize().PersonalOrganizacionPostgreSQL {
		t.Fatal("activación explícita no leída")
	}
}
