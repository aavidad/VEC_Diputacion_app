package bootstrap

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"vec-diputacion-granada/config"
)

func TestAuditoriaIntentosConfiguracionPrivadaSinValoresSupuestos(t *testing.T) {
	dir := directorioTemporalFueraDeGitPrueba(t)
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{DevelopmentMaterialDir: dir}
	archivo := filepath.Join(dir, "auditoria-intentos.json")
	if _, err := leerConfiguracionAuditoriaIntentosDesarrollo(cfg); err == nil {
		t.Fatal("ausencia abrió auditor nominal")
	}
	valida := `{"esquema":"vec.auditoria.intentos.servidor.v1","dsn_file":"intentos.dsn","proceso":"vec-sintetico","canal":"interna_corporativa","limite_segundos":5}`
	casos := map[string]string{"valida": valida,
		"campo_ajeno":   strings.Replace(valida, `"esquema":`, `"actor_ref":"cliente","esquema":`, 1),
		"duplicado":     strings.Replace(valida, `"proceso":"vec-sintetico"`, `"proceso":"vec-sintetico","proceso":"otro"`, 1),
		"dsn_fuera":     strings.Replace(valida, `"intentos.dsn"`, `"../intentos.dsn"`, 1),
		"proceso_libre": strings.Replace(valida, `"vec-sintetico"`, `"Nombre Persona"`, 1),
		"canal_ajeno":   strings.Replace(valida, `"interna_corporativa"`, `"externa_personal"`, 1),
		"sin_plazo":     strings.Replace(valida, `"limite_segundos":5`, `"limite_segundos":0`, 1)}
	for nombre, b := range casos {
		t.Run(nombre, func(t *testing.T) {
			if err := os.WriteFile(archivo, []byte(b), 0600); err != nil {
				t.Fatal(err)
			}
			c, err := leerConfiguracionAuditoriaIntentosDesarrollo(cfg)
			if (nombre == "valida") != (err == nil) {
				t.Fatalf("configuración %s err=%v", nombre, err)
			}
			if err == nil && (c.Proceso != "vec-sintetico" || c.LimiteSegundos != 5) {
				t.Fatal("proceso o plazo sustituidos")
			}
		})
	}
}

// Los dos canales usan archivos separados y se cotejan de forma exacta.
func TestAuditoriaIntentosCanalesSeparados(t *testing.T) {
	dir := directorioTemporalFueraDeGitPrueba(t)
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{DevelopmentMaterialDir: dir}
	interna := `{"esquema":"vec.auditoria.intentos.servidor.v1","dsn_file":"intentos.dsn","proceso":"vec-sintetico","canal":"interna_corporativa","limite_segundos":5}`
	externa := strings.Replace(interna, `"interna_corporativa"`, `"externa_personal"`, 1)
	if err := os.WriteFile(filepath.Join(dir, "auditoria-intentos.json"), []byte(interna), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := leerConfiguracionAuditoriaIntentosParaCanalDesarrollo(cfg, "auditoria-intentos-externa.json", "externa_personal"); err == nil {
		t.Fatal("la ausencia exterior reutilizó el archivo interno")
	}
	archivo := filepath.Join(dir, "auditoria-intentos-externa.json")
	if err := os.WriteFile(archivo, []byte(interna), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := leerConfiguracionAuditoriaIntentosParaCanalDesarrollo(cfg, "auditoria-intentos-externa.json", "externa_personal"); err == nil {
		t.Fatal("el canal exterior aceptó material corporativo")
	}
	if err := os.WriteFile(archivo, []byte(externa), 0600); err != nil {
		t.Fatal(err)
	}
	c, err := leerConfiguracionAuditoriaIntentosParaCanalDesarrollo(cfg, "auditoria-intentos-externa.json", "externa_personal")
	if err != nil || c.Canal != "externa_personal" {
		t.Fatalf("configuración exterior explícita rechazada: %v", err)
	}
	if _, err := leerConfiguracionAuditoriaIntentosDesarrollo(cfg); err != nil {
		t.Fatal("la configuración exterior sustituyó el archivo interno")
	}
	if err := os.WriteFile(filepath.Join(dir, "auditoria-intentos.json"), []byte(externa), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := leerConfiguracionAuditoriaIntentosDesarrollo(cfg); err == nil {
		t.Fatal("la apertura interna aceptó un canal exterior")
	}
}
