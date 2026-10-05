package administracion

import (
	"errors"
	"testing"
)

func escenarioConfianzaUsuariosPrueba(t *testing.T) (ConfiguracionConfianzaUsuariosV3, DependenciasConfianzaUsuariosV3) {
	t.Helper()
	cfg, deps := escenarioConfianzaPerfilesPrueba(t)
	cfg.EntradasCapacidad = cfg.EntradasCapacidad[:2]
	cfg.EntradasCapacidad[0].Audiencia = AudienciaUsuariosListarV3
	cfg.EntradasCapacidad[1].Audiencia = AudienciaUsuariosConsultarV3
	return cfg, deps
}

func TestConfianzaUsuariosConstruyeCadenaRealSeparadaDeLegacy(t *testing.T) {
	cfg, deps := escenarioConfianzaUsuariosPrueba(t)
	confianza, err := NuevaConfianzaUsuariosV3(cfg, deps)
	if err != nil || confianza.Fuente == nil || len(confianza.Emisores) != 2 || confianza.Emisores[AudienciaUsuariosListarV3] == nil || confianza.Emisores[AudienciaUsuariosConsultarV3] == nil {
		t.Fatal("confianza_usuarios_no_cerrada")
	}
	if legacy, err := NuevaConfianzaPerfilesV3(cfg, deps); !errors.Is(err, ErrConfiguracion) || legacy.Fuente != nil || len(legacy.Emisores) != 0 {
		t.Fatal("usuarios_abren_factory_legacy")
	}
	cfg, deps = escenarioConfianzaPerfilesPrueba(t)
	if usuario, err := NuevaConfianzaUsuariosV3(cfg, deps); !errors.Is(err, ErrConfiguracion) || usuario.Fuente != nil || len(usuario.Emisores) != 0 {
		t.Fatal("legacy_abre_factory_usuarios")
	}
}

func TestConfianzaUsuariosRechazaAudienciasClavesGobiernoYPools(t *testing.T) {
	casos := map[string]func(*ConfiguracionConfianzaUsuariosV3, *DependenciasConfianzaUsuariosV3){
		"tercera_audiencia": func(c *ConfiguracionConfianzaUsuariosV3, _ *DependenciasConfianzaUsuariosV3) {
			c.EntradasCapacidad = append(c.EntradasCapacidad, c.EntradasCapacidad[0])
		},
		"legacy": func(c *ConfiguracionConfianzaUsuariosV3, _ *DependenciasConfianzaUsuariosV3) {
			c.EntradasCapacidad[0].Audiencia = AudienciaPerfilesCapacidadesV3
		},
		"duplicada": func(c *ConfiguracionConfianzaUsuariosV3, _ *DependenciasConfianzaUsuariosV3) {
			c.EntradasCapacidad[1].Audiencia = c.EntradasCapacidad[0].Audiencia
		},
		"material_repetido": func(c *ConfiguracionConfianzaUsuariosV3, _ *DependenciasConfianzaUsuariosV3) {
			c.EntradasCapacidad[1].Material = c.EntradasCapacidad[0].Material
		},
		"referencia_raiz": func(c *ConfiguracionConfianzaUsuariosV3, _ *DependenciasConfianzaUsuariosV3) {
			c.EntradasCapacidad[1].ClaveID = c.Raiz.ClaveID
		},
		"pool_reutilizado": func(_ *ConfiguracionConfianzaUsuariosV3, d *DependenciasConfianzaUsuariosV3) {
			d.PoolFuente = d.PoolRegistro
		},
		"gobierno_ajeno": func(c *ConfiguracionConfianzaUsuariosV3, _ *DependenciasConfianzaUsuariosV3) {
			c.Gobierno.HuellaSHA256 = c.EntradasCapacidad[0].HuellaGobierno
		},
	}
	for nombre, cambiar := range casos {
		t.Run(nombre, func(t *testing.T) {
			c, d := escenarioConfianzaUsuariosPrueba(t)
			cambiar(&c, &d)
			f, err := NuevaConfianzaUsuariosV3(c, d)
			if !errors.Is(err, ErrConfiguracion) || f.Fuente != nil || len(f.Emisores) != 0 {
				t.Fatal("configuracion_usuarios_no_cerrada")
			}
		})
	}
}
