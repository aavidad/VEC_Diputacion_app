package bootstrap

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/config"
)

// raizOperacionGobiernoUsuarios abre una carpeta 0700 propia fuera de Git.
func raizOperacionGobiernoUsuarios(t *testing.T) (*os.Root, string) {
	t.Helper()
	d := filepath.Join(t.TempDir(), "salida")
	if os.Mkdir(d, 0700) != nil {
		t.Fatal("directorio")
	}
	r, err := AbrirRaizPrivadaDenominacionPersona(filepath.Join(d, ArchivoConfiguracionGobiernoUsuarios))
	if err != nil {
		t.Fatal("raíz")
	}
	t.Cleanup(func() { _ = r.Close() })
	return r, d
}

// poolSinConexionGobiernoUsuarios no conecta hasta la primera consulta; las
// guardas probadas deben rechazar antes de llegar a ella.
func poolSinConexionGobiernoUsuarios(t *testing.T) *pgxpool.Pool {
	t.Helper()
	p, err := pgxpool.New(t.Context(), "host=/nonexistent/vec-sin-pg user=nadie dbname=ninguna sslmode=disable")
	if err != nil {
		t.Fatal("pool")
	}
	t.Cleanup(p.Close)
	return p
}

func TestGobiernoUsuariosOperacionValidezFueraDeRango(t *testing.T) {
	t.Setenv(config.EnvExecutionProfile, config.ExecutionProfileProduction)
	t.Setenv(config.EnvAuthMode, config.AuthModeDisabled)
	t.Setenv(config.EnvDevelopmentGuard, "")
	r, _ := raizOperacionGobiernoUsuarios(t)
	pool := poolSinConexionGobiernoUsuarios(t)
	for _, v := range []time.Duration{59 * time.Minute, 24*time.Hour + time.Second, -time.Hour} {
		origen := MaterialOrigenGobiernoUsuariosAdmin{DirectorioMaterial: "/x", RutaConfiguracionHMAC: "/y", ArchivoSemillaRaiz: "/z", ValidezClaves: v}
		if _, err := PrepararGobiernoUsuariosAdmin(t.Context(), pool, origen, r, relojGobiernoUsuariosEnsayo{}); !errors.Is(err, ErrGobiernoUsuariosAdmin) {
			t.Fatal("validez aceptada", v)
		}
	}
	origen := MaterialOrigenGobiernoUsuariosAdmin{ValidezClaves: time.Hour}
	if _, err := PrepararGobiernoUsuariosAdmin(t.Context(), pool, origen, r, nil); !errors.Is(err, ErrGobiernoUsuariosAdmin) {
		t.Fatal("reloj nulo aceptado")
	}
	if _, err := PrepararGobiernoUsuariosAdmin(t.Context(), pool, origen, nil, relojGobiernoUsuariosEnsayo{}); !errors.Is(err, ErrGobiernoUsuariosAdmin) {
		t.Fatal("salida nula aceptada")
	}
}

func TestGobiernoUsuariosOperacionConfianzaCubreClavesYRaiz(t *testing.T) {
	now := time.Date(2026, 10, 15, 14, 30, 0, 0, time.UTC)
	for _, v := range []time.Duration{time.Hour, 24 * time.Hour, 1440 * time.Hour, 2160 * time.Hour} {
		expira, err := expiracionConfianzaGobiernoUsuarios(now, now.Add(2161*time.Hour), v)
		if err != nil || expira.Before(now.Add(v)) || expira.After(now.Add(2161*time.Hour)) {
			t.Fatalf("confianza no cubre clave y raíz: validez=%v expira=%v error=%v", v, expira, err)
		}
	}
	if _, err := expiracionConfianzaGobiernoUsuarios(now, now.Add(2160*time.Hour-time.Microsecond), 2160*time.Hour); !errors.Is(err, ErrGobiernoUsuariosAdmin) {
		t.Fatal("raíz corta aceptada")
	}
}

func TestGobiernoUsuariosOperacionAplicarValidaMaterialPreparado(t *testing.T) {
	now := time.Date(2026, 10, 15, 14, 30, 0, 0, time.UTC)
	dev := config.Config{ExecutionProfile: config.ExecutionProfileDevelopment, AuthMode: config.AuthModeDevelopment, DevelopmentGuard: config.DevelopmentGuardAcknowledgement}
	prod := config.Config{ExecutionProfile: config.ExecutionProfileProduction}
	for _, tc := range []struct {
		nombre   string
		cfg      config.Config
		clave    time.Duration
		confia   time.Duration
		invalida bool
	}{
		{"produccion_24", prod, 24 * time.Hour, 24 * time.Hour, false},
		{"produccion_rechaza_60_dias", prod, 1440 * time.Hour, 1440 * time.Hour, true},
		{"desarrollo_90_dias", dev, 2160 * time.Hour, 2160 * time.Hour, false},
		{"desarrollo_rechaza_91_dias", dev, 2184 * time.Hour, 2184 * time.Hour, true},
		{"replay_diario_24_horas", prod, 24 * time.Hour, 24 * time.Hour, false},
	} {
		t.Run(tc.nombre, func(t *testing.T) {
			publicada := now
			if tc.nombre == "replay_diario_24_horas" {
				publicada = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
			}
			material := ConfiguracionMaterialUsuariosAdmin{}
			material.Gobierno.PublicadaEn, material.Gobierno.ExpiraEn = publicada, publicada.Add(tc.confia)
			material.Entradas = []DescriptorClaveUsuariosAdmin{{ValidaHasta: now.Add(tc.clave)}}
			plan := planGobiernoUsuariosAdmin{PreparadoEn: now}
			plan.Configuracion.PublicadaEn, plan.Configuracion.ExpiraEn = material.Gobierno.PublicadaEn, material.Gobierno.ExpiraEn
			err := validarVigenciaPreparadaGobiernoUsuarios(material, plan, tc.cfg)
			if (err != nil) != tc.invalida {
				t.Fatalf("validación de material: %v", err)
			}
		})
	}
	// Un cambio de perfil entre preparar y aplicar se detecta con los bytes
	// preparados antes de reservar un acuse o conectar al operador.
	t.Setenv(config.EnvExecutionProfile, config.ExecutionProfileProduction)
	t.Setenv(config.EnvAuthMode, config.AuthModeDisabled)
	t.Setenv(config.EnvDevelopmentGuard, "")
	r, d := raizOperacionGobiernoUsuarios(t)
	material := ConfiguracionMaterialUsuariosAdmin{}
	material.Gobierno.PublicadaEn, material.Gobierno.ExpiraEn = now, now.Add(1440*time.Hour)
	material.Entradas = []DescriptorClaveUsuariosAdmin{{ValidaHasta: now.Add(1440 * time.Hour)}}
	plan := planGobiernoUsuariosAdmin{PreparadoEn: now}
	plan.Configuracion.PublicadaEn, plan.Configuracion.ExpiraEn = material.Gobierno.PublicadaEn, material.Gobierno.ExpiraEn
	for _, entrada := range []struct {
		nombre string
		valor  any
	}{{ArchivoConfiguracionGobiernoUsuarios, material}, {ArchivoPlanGobiernoUsuarios, plan}} {
		b, err := json.Marshal(entrada.valor)
		if err != nil || escribirPrivadoGobiernoUsuarios(r, entrada.nombre, b) != nil {
			t.Fatal("preparación sintética")
		}
	}
	if _, err := AplicarGobiernoUsuariosAdminPreparado(t.Context(), poolSinConexionGobiernoUsuarios(t), r, "acuse-guardia.json", relojGobiernoUsuariosEnsayo{}); !errors.Is(err, ErrGobiernoUsuariosAdmin) {
		t.Fatal("material de 60 días aceptado en producción")
	}
	if _, err := os.Lstat(filepath.Join(d, "acuse-guardia.json")); !os.IsNotExist(err) {
		t.Fatal("se reservó acuse antes de rechazar vigencia")
	}
}

func TestGobiernoUsuariosOperacionNombreAcuse(t *testing.T) {
	validos := []string{"acuse-aplicar.json", "acuse-replay-2.json"}
	invalidos := []string{"", ".", "..", "../acuse.json", "sub/acuse.json", "/tmp/acuse.json", ArchivoMaterialGobiernoUsuarios, ArchivoPlanGobiernoUsuarios, ArchivoConfiguracionGobiernoUsuarios, ArchivoAprobacionGobiernoUsuarios, strings.Repeat("a", 129)}
	for _, n := range validos {
		if !nombreAcuseGobiernoUsuariosValido(n) {
			t.Fatal("nombre válido rechazado", n)
		}
	}
	for _, n := range invalidos {
		if nombreAcuseGobiernoUsuariosValido(n) {
			t.Fatal("nombre inválido aceptado", n)
		}
	}
}

func TestGobiernoUsuariosOperacionAplicarRetiraReservaSinCommit(t *testing.T) {
	r, d := raizOperacionGobiernoUsuarios(t)
	pool := poolSinConexionGobiernoUsuarios(t)
	// Sin preparación previa la operación falla antes de la base y no deja acuse.
	_, err := AplicarGobiernoUsuariosAdminPreparado(t.Context(), pool, r, "acuse.json", relojGobiernoUsuariosEnsayo{})
	if !errors.Is(err, ErrGobiernoUsuariosAdmin) || errors.Is(err, ErrCommitGobiernoUsuariosIndeterminado) || errors.Is(err, ErrAcuseGobiernoUsuariosNoGuardado) {
		t.Fatal("error no limpio")
	}
	if _, err := os.Lstat(filepath.Join(d, "acuse.json")); !os.IsNotExist(err) {
		t.Fatal("reserva conservada sin COMMIT")
	}
	// Un acuse existente se rechaza y no se trunca.
	if os.WriteFile(filepath.Join(d, "previo.json"), []byte("original"), 0600) != nil {
		t.Fatal("previo")
	}
	if _, err := AplicarGobiernoUsuariosAdminPreparado(t.Context(), pool, r, "previo.json", relojGobiernoUsuariosEnsayo{}); err == nil {
		t.Fatal("acuse existente aceptado")
	}
	if b, _ := os.ReadFile(filepath.Join(d, "previo.json")); string(b) != "original" {
		t.Fatal("acuse previo alterado o retirado")
	}
	for _, n := range []string{"../fuera.json", ArchivoPlanGobiernoUsuarios} {
		if _, err := AplicarGobiernoUsuariosAdminPreparado(t.Context(), pool, r, n, relojGobiernoUsuariosEnsayo{}); err == nil {
			t.Fatal("nombre aceptado", n)
		}
	}
	if _, err := VerificarCadenaGobiernoUsuariosAdmin(t.Context(), pool, r, ArchivoMaterialGobiernoUsuarios); err == nil {
		t.Fatal("informe sobre material aceptado")
	}
}

func TestGobiernoUsuariosOperacionLecturaPrivada(t *testing.T) {
	r, d := raizOperacionGobiernoUsuarios(t)
	if escribirPrivadoGobiernoUsuarios(r, "plan.json", []byte("{}")) != nil {
		t.Fatal("escritura")
	}
	if b, err := leerPrivadoGobiernoUsuarios(r, "plan.json"); err != nil || string(b) != "{}" {
		t.Fatal("lectura")
	}
	if i, err := os.Lstat(filepath.Join(d, "plan.json")); err != nil || i.Mode().Perm() != 0600 {
		t.Fatal("permisos")
	}
	if os.WriteFile(filepath.Join(d, "abierto.json"), []byte("{}"), 0644) != nil || os.Chmod(filepath.Join(d, "abierto.json"), 0644) != nil {
		t.Fatal("abierto")
	}
	if _, err := leerPrivadoGobiernoUsuarios(r, "abierto.json"); err == nil {
		t.Fatal("fichero legible por otros aceptado")
	}
	if os.Symlink(filepath.Join(d, "plan.json"), filepath.Join(d, "alias.json")) != nil {
		t.Fatal("enlace")
	}
	if _, err := leerPrivadoGobiernoUsuarios(r, "alias.json"); err == nil {
		t.Fatal("enlace seguido")
	}
	if os.WriteFile(filepath.Join(d, "grande.json"), make([]byte, limiteArchivoGobiernoUsuarios+1), 0600) != nil {
		t.Fatal("grande")
	}
	if _, err := leerPrivadoGobiernoUsuarios(r, "grande.json"); err == nil {
		t.Fatal("fichero sin límite aceptado")
	}
	if _, err := leerPrivadoGobiernoUsuarios(r, "../fuera.json"); err == nil {
		t.Fatal("escape aceptado")
	}
}
