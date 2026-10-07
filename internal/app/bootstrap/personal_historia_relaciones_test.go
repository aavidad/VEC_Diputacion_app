package bootstrap

import (
	"os"
	"path/filepath"
	"testing"

	"vec-diputacion-granada/config"
	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
)

func TestHistoriaRelacionesSoloSeSeleccionaConConfiguracionPropia(t *testing.T) {
	for _, caso := range []struct {
		nombre, contenido string
		seleccion, falla  bool
	}{
		{"ausente", `{"version":2}`, false, false},
		{"servicios_no_presta", `{"version":2,"historia_servicios":{}}`, false, false},
		{"relaciones", `{"version":2,"historia_relaciones":{}}`, true, false},
		{"duplicada", `{"version":2,"historia_relaciones":{},"historia_relaciones":{}}`, false, true},
		{"ajena", `{"version":2,"historia_relaciones":{"empleado_ref":"emp_ajeno"}}`, false, true},
		{"version_no_admitida", `{"version":1,"historia_relaciones":{}}`, false, true},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			raiz := t.TempDir()
			if err := os.Mkdir(filepath.Join(raiz, "identidad"), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(raiz, "identidad", "personal-empleado.json"), []byte(caso.contenido), 0600); err != nil {
				t.Fatal(err)
			}
			cfg := config.Config{PersonalEmpleadoEnabled: "true", DevelopmentMaterialDir: raiz}
			seleccion, err := historiaRelacionesPersonalSolicitada(cfg)
			if seleccion != caso.seleccion || (err != nil) != caso.falla {
				t.Fatalf("seleccion=%t falla=%t", seleccion, err != nil)
			}
			cfg.PersonalEmpleadoEnabled = "false"
			if seleccion, err := historiaRelacionesPersonalSolicitada(cfg); seleccion || err != nil {
				t.Fatal("selector apagado consultó configuración")
			}
		})
	}
}

func TestHistoriaRelacionesNoHeredaMaterialServicios(t *testing.T) {
	descriptor := descriptorMaterialHistoriaRelacionesPersonal()
	if descriptor.Audiencia != personaldomain.AudienciaHistoriaRelacionesPropia || descriptor == descriptorMaterialHistoriaServiciosPersonal() {
		t.Fatal("material de otra capacidad")
	}
	seleccion := seleccionMaterialCTDesarrollo{historiaRelacionesPersonal: true}
	propios := 0
	for _, d := range descriptoresMaterialSeleccionadosCTDesarrollo(seleccion) {
		if d.Audiencia == personaldomain.AudienciaHistoriaRelacionesPropia {
			propios++
		}
		if d.Audiencia == personaldomain.AudienciaHistoriaServiciosPropia {
			t.Fatal("se seleccionó historia de servicios")
		}
	}
	if propios != 1 {
		t.Fatal("audiencia propia ausente o duplicada")
	}
}
