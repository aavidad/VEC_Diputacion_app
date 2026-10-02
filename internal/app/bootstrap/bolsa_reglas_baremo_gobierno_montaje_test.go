package bootstrap

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"vec-diputacion-granada/config"
)

func configGobiernoBaremoHTTPPrueba(t *testing.T) (config.Config, configuracionGobiernoReglasBaremoHTTPV3) {
	t.Helper()
	raiz := t.TempDir()
	if err := os.Chmod(raiz, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(raiz, "bolsa"), 0700); err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{ExecutionProfile: config.ExecutionProfileDevelopment, AuthMode: config.AuthModeDevelopment,
		DevelopmentGuard: config.DevelopmentGuardAcknowledgement, DevelopmentMaterialDir: raiz}
	c := configuracionGobiernoReglasBaremoHTTPV3{Esquema: "vec.bolsa.gobierno-reglas-baremo.configuracion.v3",
		ConvocatoriaRef: "convocatoria:prueba", ExpedienteRef: "expediente:prueba", CatalogoMotivosID: "motivos_prueba",
		DSNFiles: map[string]string{"runtime": "runtime.conf", "fuente_autorizacion": "fuente.conf", "motivos_autorizacion": "motivos.conf"}}
	return cfg, c
}

func escribirConfigGobiernoBaremoHTTPPrueba(t *testing.T, cfg config.Config, c any) string {
	t.Helper()
	b, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(cfg.DevelopmentMaterialDir, archivoGobiernoReglasBaremoHTTPV3)
	if err := os.WriteFile(ruta, b, 0600); err != nil {
		t.Fatal(err)
	}
	return ruta
}

func TestGobiernoBaremoHTTPConfiguracionPrivadaOpcionalSinProvisionImplicita(t *testing.T) {
	cfg, c := configGobiernoBaremoHTTPPrueba(t)
	if obtenida, err := leerConfiguracionGobiernoReglasBaremoHTTPV3(cfg); err != nil || obtenida != nil {
		t.Fatal("ausencia no conservó familia apagada")
	}
	escribirConfigGobiernoBaremoHTTPPrueba(t, cfg, c)
	obtenida, err := leerConfiguracionGobiernoReglasBaremoHTTPV3(cfg)
	if err != nil || obtenida == nil || obtenida.ProvisionarPerfil || obtenida.AprobacionRef != "" || obtenida.PreimagenPerfilSHA256 != "" {
		t.Fatalf("configuración no cargada: err=%v valor=%#v dobleLlave=%v arbol=%v", err, obtenida, cfg.DevelopmentEnabledByDoubleKey(), validarArbolMaterialDesarrollo(cfg.DevelopmentMaterialDir))
	}
	base, _, _ := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	m, err := prepararMontajeGobiernoReglasBaremoHTTPV3(config.Config{}, base, relojContratacionTemporalDesarrollo{})
	if err != nil || m.perfil != nil || m.configuracion != nil || m.perfilRef == "" {
		t.Fatal("declaración sin configuración inventó permiso")
	}
	rutas, cerrar, err := m.rutas(context.Background(), config.Config{}, nil, nil, catalogoFronterasComunDesarrollo{}, nil, relojContratacionTemporalDesarrollo{})
	if err != nil || len(rutas) != 3 || cerrar == nil {
		t.Fatal("familia ausente intentó construir dependencias")
	}
	cerrar()
}

func TestGobiernoBaremoHTTPConfiguracionRechazaFronterasPrivadasInvalidas(t *testing.T) {
	for _, caso := range []string{"sin_doble_llave", "modo_publico", "enlace", "en_repositorio", "pool_extra", "dsn_escape", "dsn_alias", "aprobacion_sin_provision", "huella_sin_aprobacion", "campo_actor"} {
		t.Run(caso, func(t *testing.T) {
			cfg, c := configGobiernoBaremoHTTPPrueba(t)
			if caso == "pool_extra" {
				c.DSNFiles["gobierno"] = "gobierno.conf"
			}
			if caso == "dsn_escape" {
				c.DSNFiles["runtime"] = "../runtime.conf"
			}
			if caso == "dsn_alias" {
				c.DSNFiles["runtime"] = c.DSNFiles["fuente_autorizacion"]
			}
			if caso == "aprobacion_sin_provision" {
				c.AprobacionRef = "aprobacion:prueba"
			}
			if caso == "huella_sin_aprobacion" {
				c.ProvisionarPerfil = true
				c.PreimagenPerfilSHA256 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
			}
			ruta := escribirConfigGobiernoBaremoHTTPPrueba(t, cfg, c)
			switch caso {
			case "sin_doble_llave":
				cfg.DevelopmentGuard = ""
			case "modo_publico":
				if err := os.Chmod(ruta, 0644); err != nil {
					t.Fatal(err)
				}
			case "en_repositorio":
				if err := os.Mkdir(filepath.Join(cfg.DevelopmentMaterialDir, ".git"), 0700); err != nil {
					t.Fatal(err)
				}
			case "enlace":
				target := filepath.Join(cfg.DevelopmentMaterialDir, "original.json")
				if err := os.Rename(ruta, target); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, ruta); err != nil {
					t.Fatal(err)
				}
			case "campo_actor":
				b, err := json.Marshal(c)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(ruta, append([]byte(`{"actor_ref":"actor:cliente",`), b[1:]...), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := leerConfiguracionGobiernoReglasBaremoHTTPV3(cfg); err == nil {
				t.Fatal("aceptó configuración privada incompatible")
			}
		})
	}
}
