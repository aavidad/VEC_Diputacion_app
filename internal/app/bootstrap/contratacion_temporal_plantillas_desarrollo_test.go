package bootstrap

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/config"
)

func configPlantillasDesarrolloPrueba(ruta string) config.Config {
	return config.Config{
		ExecutionProfile: config.ExecutionProfileDevelopment, AuthMode: config.AuthModeDevelopment,
		DevelopmentGuard: config.DevelopmentGuardAcknowledgement,
		ReglasEjemplo:    config.ConfiguracionReglasEjemplo{CTPlantillasSourcePath: ruta},
	}
}

func TestPlantillasCTSeCarganDelRepositorioOdeLaRutaDeclarada(t *testing.T) {
	instante := time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
	porDefecto, err := cargarPlantillasBorradorCTDesarrollo(config.Config{}, instante)
	if err != nil || len(porDefecto.Tipos()) != 10 {
		t.Fatalf("catálogo del repositorio: %v", err)
	}
	declarado, err := cargarPlantillasBorradorCTDesarrollo(configPlantillasDesarrolloPrueba("../../../"+rutaPlantillasCTEjemplo), instante)
	if err != nil || declarado.Huella() != porDefecto.Huella() {
		t.Fatalf("ruta declarada: %v", err)
	}
	if _, err := cargarPlantillasBorradorCTDesarrollo(configPlantillasDesarrolloPrueba(filepath.Join(t.TempDir(), "no-existe.json")), instante); !errors.Is(err, errPlantillasCTNoDisponibles) {
		t.Fatalf("ruta declarada inexistente: %v", err)
	}
	original, err := os.ReadFile("../../../" + rutaPlantillasCTEjemplo)
	if err != nil {
		t.Fatal(err)
	}
	roto := filepath.Join(t.TempDir(), "plantillas.json")
	if err := os.WriteFile(roto, []byte(strings.Replace(string(original), "{{numero_expediente}}", "{{nombre_persona}}", 1)), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := cargarPlantillasBorradorCTDesarrollo(configPlantillasDesarrolloPrueba(roto), instante); !errors.Is(err, errPlantillasCTNoDisponibles) {
		t.Fatalf("campo desconocido debe impedir arrancar: %v", err)
	}
	fuera := config.Config{ExecutionProfile: config.ExecutionProfileProduction,
		ReglasEjemplo: config.ConfiguracionReglasEjemplo{CTPlantillasSourcePath: roto}}
	if _, err := cargarPlantillasBorradorCTDesarrollo(fuera, instante); !errors.Is(err, config.ErrConfiguracionReglasEjemploFueraDesarrollo) {
		t.Fatalf("fuera de desarrollo: %v", err)
	}
}
