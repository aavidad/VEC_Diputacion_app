package config

import "testing"

func TestCronosEquipoInformesAisladoDelArranque(t *testing.T) {
	for _, selector := range []string{"", "false"} {
		activo, err := (Config{CronosEquipoInformesEnabled: selector}).CronosEquipoInformesDesarrolloActivo()
		if err != nil || activo {
			t.Fatalf("selector apagado %q: activo=%v error=%v", selector, activo, err)
		}
	}
	for _, c := range []Config{
		{CronosEquipoInformesEnabled: "true"},
		{CronosEquipoInformesEnabled: "TRUE"},
		{CronosEquipoInformesEnabled: "true", CronosEquipoInformesConfigFile: "../identidad/cronos-equipo-informes.json"},
		{CronosEquipoInformesEnabled: "true", CronosEquipoInformesConfigFile: "/etc/vec/cronos-equipo-informes.json"},
	} {
		activo, err := c.CronosEquipoInformesDesarrolloActivo()
		if err == nil || activo {
			t.Fatalf("configuración inválida activada: %+v", c)
		}
	}
	c := Config{
		ExecutionProfile: ExecutionProfileDevelopment, AuthMode: AuthModeDevelopment,
		DevelopmentGuard: DevelopmentGuardAcknowledgement,
		DevelopmentMaterialDir: "/material-de-ensayo",
		CronosEmpleadoEnabled: "true", CronosEquipoInformesEnabled: "true",
		CronosEquipoInformesConfigFile: RutaPrivadaCronosEquipoInformes,
	}
	if activo, err := c.CronosEquipoInformesDesarrolloActivo(); err != nil || !activo {
		t.Fatalf("configuración completa cerrada: activo=%v error=%v", activo, err)
	}
}
