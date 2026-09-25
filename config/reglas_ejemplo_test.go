package config

import (
	"errors"
	"testing"
)

func TestReglasEjemploSeCarganDelEntornoYSeNormalizan(t *testing.T) {
	t.Setenv(EnvBolsaReglasSourcePath, "  data/demo/reglas/bolsa_reglas.ejemplo.demo.json ")
	t.Setenv(EnvCTReglasSourcePath, "data/demo/reglas/ct_reglas.ejemplo.demo.json")
	t.Setenv(EnvBolsaRolesSegregacionSourcePath, " data/demo/reglas/bolsa_roles_segregacion.demo.json")
	t.Setenv(EnvCTPlantillasSourcePath, " plantillas.json ")
	cfg := Load().Normalize()
	if cfg.ReglasEjemplo.BolsaSourcePath != "data/demo/reglas/bolsa_reglas.ejemplo.demo.json" ||
		cfg.ReglasEjemplo.CTSourcePath != "data/demo/reglas/ct_reglas.ejemplo.demo.json" ||
		cfg.ReglasEjemplo.BolsaRolesSegregacionSourcePath != "data/demo/reglas/bolsa_roles_segregacion.demo.json" ||
		cfg.ReglasEjemplo.CTPlantillasSourcePath != "plantillas.json" {
		t.Fatalf("rutas no cargadas: %+v", cfg.ReglasEjemplo)
	}
}

func TestReglasEjemploSoloConDobleLlaveDeDesarrollo(t *testing.T) {
	desarrollo := Config{
		ExecutionProfile: ExecutionProfileDevelopment, AuthMode: AuthModeDevelopment,
		DevelopmentGuard: DevelopmentGuardAcknowledgement,
	}
	if reglas, activas, err := desarrollo.ReglasEjemploDesarrollo(); err != nil || activas || reglas.Configurada() {
		t.Fatalf("sin rutas no se compone nada: %+v %v %v", reglas, activas, err)
	}
	conReglas := desarrollo
	conReglas.ReglasEjemplo = ConfiguracionReglasEjemplo{CTSourcePath: " ct.json "}
	if reglas, activas, err := conReglas.ReglasEjemploDesarrollo(); err != nil || !activas || reglas.CTSourcePath != "ct.json" {
		t.Fatalf("desarrollo con doble llave debe componer: %+v %v %v", reglas, activas, err)
	}
	soloMotivos := desarrollo
	soloMotivos.CTAnalisisMotivosSourcePath = "motivos.json"
	if _, activas, err := soloMotivos.ReglasEjemploDesarrollo(); err != nil || activas {
		t.Fatalf("los motivos no activan el resolutor: %v %v", activas, err)
	}

	fuera := []Config{
		{ReglasEjemplo: ConfiguracionReglasEjemplo{BolsaSourcePath: "bolsa.json"}},
		{ExecutionProfile: ExecutionProfileProduction, ReglasEjemplo: ConfiguracionReglasEjemplo{CTSourcePath: "ct.json"}},
		{ExecutionProfile: ExecutionProfileProduction, CTAnalisisMotivosSourcePath: "motivos.json"},
		{ExecutionProfile: ExecutionProfileProduction, ReglasEjemplo: ConfiguracionReglasEjemplo{BolsaRolesSegregacionSourcePath: "roles.json"}},
		{ExecutionProfile: ExecutionProfileProduction, ReglasEjemplo: ConfiguracionReglasEjemplo{CTPlantillasSourcePath: "plantillas.json"}},
		{ExecutionProfile: ExecutionProfileRRHHPresentation, ReglasEjemplo: ConfiguracionReglasEjemplo{BolsaSourcePath: "bolsa.json"}},
		{ExecutionProfile: ExecutionProfileDevelopment, AuthMode: AuthModeDevelopment,
			ReglasEjemplo: ConfiguracionReglasEjemplo{BolsaSourcePath: "bolsa.json"}},
	}
	for indice, cfg := range fuera {
		if _, activas, err := cfg.ReglasEjemploDesarrollo(); !errors.Is(err, ErrConfiguracionReglasEjemploFueraDesarrollo) || activas {
			t.Errorf("caso %d: debe impedir el arranque: %v %v", indice, activas, err)
		}
	}
}

func TestRechazarReglasEjemploSinComposicion(t *testing.T) {
	desarrollo := Config{
		ExecutionProfile: ExecutionProfileDevelopment, AuthMode: AuthModeDevelopment,
		DevelopmentGuard: DevelopmentGuardAcknowledgement,
	}
	if err := desarrollo.RechazarReglasEjemploSinComposicion(); err != nil {
		t.Fatalf("sin catálogos no se rechaza: %v", err)
	}
	if err := (Config{}).RechazarReglasEjemploSinComposicion(); err != nil {
		t.Fatalf("sin catálogos ni perfil no se rechaza: %v", err)
	}
	conDobleLlave := []Config{desarrollo, desarrollo, desarrollo}
	conDobleLlave[0].ReglasEjemplo.BolsaSourcePath = "bolsa.json"
	conDobleLlave[1].ReglasEjemplo.CTSourcePath = " ct.json "
	conDobleLlave[2].CTAnalisisMotivosSourcePath = "motivos.json"
	for indice, cfg := range conDobleLlave {
		if err := cfg.RechazarReglasEjemploSinComposicion(); !errors.Is(err, ErrConfiguracionReglasEjemploSinComposicion) {
			t.Errorf("caso %d con doble llave: %v", indice, err)
		}
	}
	fuera := Config{ExecutionProfile: ExecutionProfileProduction,
		ReglasEjemplo: ConfiguracionReglasEjemplo{BolsaSourcePath: "bolsa.json"}}
	if err := fuera.RechazarReglasEjemploSinComposicion(); !errors.Is(err, ErrConfiguracionReglasEjemploFueraDesarrollo) {
		t.Fatalf("fuera de desarrollo: %v", err)
	}
	blancos := desarrollo
	blancos.ReglasEjemplo.BolsaSourcePath = "   "
	if err := blancos.RechazarReglasEjemploSinComposicion(); err != nil {
		t.Fatalf("una ruta en blanco no declara catálogo: %v", err)
	}
}

func TestRetribucionesCTSiguenLaDobleLlaveDeLasReglasEjemplo(t *testing.T) {
	t.Setenv(EnvCTRetribucionesSourcePath, " data/demo/reglas/ct_retribuciones.demo.json ")
	cargada := Load().Normalize()
	if cargada.ReglasEjemplo.CTRetribucionesSourcePath != "data/demo/reglas/ct_retribuciones.demo.json" {
		t.Fatalf("ruta de retribuciones no cargada: %+v", cargada.ReglasEjemplo)
	}
	desarrollo := Config{
		ExecutionProfile: ExecutionProfileDevelopment, AuthMode: AuthModeDevelopment,
		DevelopmentGuard: DevelopmentGuardAcknowledgement,
		ReglasEjemplo:    ConfiguracionReglasEjemplo{CTRetribucionesSourcePath: "retribuciones.json"},
	}
	if reglas, activas, err := desarrollo.ReglasEjemploDesarrollo(); err != nil || !activas || reglas.CTRetribucionesSourcePath != "retribuciones.json" {
		t.Fatalf("desarrollo con doble llave debe componer las retribuciones: %+v %v %v", reglas, activas, err)
	}
	produccion := desarrollo
	produccion.ExecutionProfile = ExecutionProfileProduction
	if _, activas, err := produccion.ReglasEjemploDesarrollo(); !errors.Is(err, ErrConfiguracionReglasEjemploFueraDesarrollo) || activas {
		t.Fatalf("las retribuciones de ejemplo no se admiten fuera de desarrollo: %v %v", activas, err)
	}
}
