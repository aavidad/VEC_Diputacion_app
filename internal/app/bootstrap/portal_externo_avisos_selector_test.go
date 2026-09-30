package bootstrap

import (
	"errors"
	"testing"

	"vec-diputacion-granada/config"
)

func TestPortalExternoAvisosExigePreferenciasYCorreosAntesDelRetornoTemprano(t *testing.T) {
	for _, caso := range []struct {
		nombre, preferencias, correos, avisos string
		miBolsa                               bool
		esperado                              error
	}{
		{"todo_apagado", "false", "false", "false", false, nil},
		{"aviso_sin_usuarios", "false", "false", "true", false, errAvisosExternos},
		{"aviso_con_preferencias_sin_correos", "true", "false", "true", false, errAvisosExternos},
		{"mi_bolsa_no_suple_usuarios", "false", "false", "true", true, errAvisosExternos},
		{"aviso_con_usuarios_completo", "true", "true", "true", false, ErrEmisorIncidenciasRequerido},
		{"aviso_apagado_con_preferencias", "true", "false", "false", false, ErrEmisorIncidenciasRequerido},
		{"selector_aviso_invalido", "false", "false", "TRUE", false, ErrActivacionDesarrolloInvalida},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			t.Setenv(envUsuariosPreferenciasDesarrollo, caso.preferencias)
			t.Setenv(envUsuariosCorreosDesarrollo, caso.correos)
			t.Setenv(envUsuariosImagenDesarrollo, "false")
			t.Setenv(envAvisosExternos, caso.avisos)
			if caso.miBolsa {
				t.Setenv(config.EnvExternoBolsaDatabaseURL, "postgresql://vec_externo_bolsa_sintetico@127.0.0.1:1/vec")
			} else {
				t.Setenv(config.EnvExternoBolsaDatabaseURL, "")
			}
			cfg := config.Load()
			cfg.ExecutionProfile = config.ExecutionProfileDevelopment
			cfg.AuthMode = config.AuthModeDevelopment
			cfg.DevelopmentGuard = config.DevelopmentGuardAcknowledgement
			manejador, cerrar, err := nuevasCapacidadesPersonalesPortalExterno(cfg, nil, nil)
			if !errors.Is(err, caso.esperado) || manejador != nil || cerrar == nil {
				t.Fatalf("flags no preservan frontera de avisos: manejador=%v cierre=%v error=%v", manejador, cerrar != nil, err)
			}
			cerrar()
		})
	}
}
