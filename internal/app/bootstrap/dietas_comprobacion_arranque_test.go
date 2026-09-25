package bootstrap

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vec-diputacion-granada/config"
)

func configuracionComprobacionDietasPrueba(t *testing.T, dir string, osrm bool) config.Config {
	t.Helper()
	dsn := func(u string) string {
		return "postgres://" + u + ":secreto-no-visible@127.0.0.1:1/postgres?sslmode=verify-full&sslrootcert=/no/existe/ca.crt"
	}
	c, err := config.NuevaConfiguracionDietasBorradores(dsn("dietas"), dsn("personal"), dsn("asignacion"), dsn("auditoria"))
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{
		ExecutionProfile:           config.ExecutionProfileDevelopment,
		AuthMode:                   config.AuthModeDevelopment,
		DevelopmentGuard:           config.DevelopmentGuardAcknowledgement,
		DevelopmentMaterialDir:     dir,
		DietasBorradoresEnabled:    "true",
		DietasBorradoresPostgreSQL: c,
	}
	if osrm {
		cfg.OSRMBaseURL = "http://127.0.0.1:5000"
		cfg.OSRMScopeName = "Granada provincia + 15 km"
		cfg.OSRMScopeBounds = "36.45,-4.6,38.25,-2.15"
		cfg.OSRMAllowedCIDRs = []string{"127.0.0.1/32"}
		cfg.OSRMGraphVersion = versionGrafoCartografiaPrueba
	}
	return cfg
}

func comprobarRechazoDietas(t *testing.T, err error, etapa string) {
	t.Helper()
	if !errors.Is(err, ErrComprobacionArranqueDietas) || !strings.Contains(err.Error(), etapa) {
		t.Fatalf("se esperaba rechazo en %q: %v", etapa, err)
	}
	if strings.Contains(err.Error(), "secreto-no-visible") || strings.Contains(err.Error(), "postgres://") {
		t.Fatalf("el error filtra conexión: %v", err)
	}
}

func TestComprobacionArranqueDietasFallaCerradoSinFiltrarSecretos(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	var nulo context.Context
	if err := ComprobarArranqueDietasSoloLectura(nulo, config.Config{}); !errors.Is(err, ErrComprobacionArranqueDietas) {
		t.Fatalf("ctx nil: %v", err)
	}
	comprobarRechazoDietas(t, ComprobarArranqueDietasSoloLectura(ctx, config.Config{}), "no es true")
	comprobarRechazoDietas(t, ComprobarArranqueDietasSoloLectura(ctx, config.Config{DietasBorradoresEnabled: "si"}), "selector")
	sinLlave := configuracionComprobacionDietasPrueba(t, dir, true)
	sinLlave.DevelopmentGuard = ""
	comprobarRechazoDietas(t, ComprobarArranqueDietasSoloLectura(ctx, sinLlave), "selector")
	comprobarRechazoDietas(t, ComprobarArranqueDietasSoloLectura(ctx, configuracionComprobacionDietasPrueba(t, dir, false)), "cartografia")

	cfg := configuracionComprobacionDietasPrueba(t, dir, true)
	comprobarRechazoDietas(t, ComprobarArranqueDietasSoloLectura(ctx, cfg), "dietas-comisiones.json")

	identidad := filepath.Join(dir, "identidad")
	if err := os.Mkdir(identidad, 0o700); err != nil {
		t.Fatal(err)
	}
	comisiones := filepath.Join(identidad, "dietas-comisiones.json")
	if err := os.WriteFile(comisiones, []byte(`{"dsn_contexto":"a","dsn_contexto":"b"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	comprobarRechazoDietas(t, ComprobarArranqueDietasSoloLectura(ctx, cfg), "claves repetidas")
	if err := os.WriteFile(comisiones, []byte(`{"version":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(comisiones, 0o644); err != nil {
		t.Fatal(err)
	}
	comprobarRechazoDietas(t, ComprobarArranqueDietasSoloLectura(ctx, cfg), "0600")
	if err := os.Chmod(comisiones, 0o600); err != nil {
		t.Fatal(err)
	}
	rutas := filepath.Join(identidad, "dietas-rutas.json")
	if err := os.WriteFile(rutas, []byte(`[`), 0o600); err != nil {
		t.Fatal(err)
	}
	comprobarRechazoDietas(t, ComprobarArranqueDietasSoloLectura(ctx, cfg), "dietas-rutas.json")
	if err := os.Remove(rutas); err != nil {
		t.Fatal(err)
	}
	// Con los manifiestos legibles, la primera conexión (TLS sin CA legible)
	// se rechaza antes de abrir socket y sin revelar la URL.
	comprobarRechazoDietas(t, ComprobarArranqueDietasSoloLectura(ctx, cfg), "pools VEC_DIETAS_BORRADORES/PERSONAL_RELACIONES")
}

func TestLeerDSNsMaterialDietasComprobacionOpcional(t *testing.T) {
	m, err := leerDSNsMaterialDietasComprobacion(filepath.Join(t.TempDir(), "no-existe.json"), 1024, true)
	if m != nil || err != nil {
		t.Fatalf("opcional ausente: %v %v", m, err)
	}
	if _, err := leerDSNsMaterialDietasComprobacion(filepath.Join(t.TempDir(), "no-existe.json"), 1024, false); err == nil {
		t.Fatal("obligatorio ausente aceptado")
	}
	ruta := filepath.Join(t.TempDir(), "m.json")
	if err := os.WriteFile(ruta, []byte(`{"dsn_consumo":"x","cuentas":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	m, err = leerDSNsMaterialDietasComprobacion(ruta, 1024, false)
	if err != nil || m == nil || m.DSNConsumo != "x" {
		t.Fatalf("lectura: %v %+v", err, m)
	}
}
