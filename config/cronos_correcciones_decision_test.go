package config

import "testing"

func TestCronosCorreccionesDecisionAisladaDelSelectorBase(t *testing.T) {
	base := Config{CronosEmpleadoEnabled: "true", ExecutionProfile: ExecutionProfileDevelopment,
		AuthMode: AuthModeDevelopment, DevelopmentGuard: DevelopmentGuardAcknowledgement,
		DevelopmentMaterialDir: t.TempDir()}
	for _, valor := range []string{"", "false", "si", "TRUE"} {
		base.CronosCorreccionesDecisionEnabled = valor
		if base.CronosCorreccionesDecisionDesarrolloActiva() {
			t.Fatalf("selector %q activó decisiones", valor)
		}
		if activo, err := base.CronosEmpleadoDesarrolloActivo(); err != nil || !activo {
			t.Fatalf("selector C5b afectó a Cronos base: %v", err)
		}
	}
	base.CronosCorreccionesDecisionEnabled = " true "
	if !base.CronosCorreccionesDecisionDesarrolloActiva() {
		t.Fatal("selector normalizado no activó C5b")
	}
	base.CronosEmpleadoEnabled = "false"
	if base.CronosCorreccionesDecisionDesarrolloActiva() {
		t.Fatal("C5b activa con Cronos apagado")
	}
}
