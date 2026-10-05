package adminperfiles

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	api "vec-diputacion-granada/internal/vec/adapters/httpapi/administracionperfiles"
	"vec-diputacion-granada/internal/vec/ports"
)

func TestIS16SoloEnvelopeConfirmadoDevuelveDenegacionFuncional(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	o := ObservacionADMIN{Entorno: "desarrollo", Host: "admin.example.invalid", Audiencia: "vec.admin.perfiles.v1", CertificadoSHA256: strings.Repeat("a", 64), CASHA256: strings.Repeat("b", 64), AutenticacionVerificadaEn: ahora, RevocacionVerificadaEn: ahora, CRLVigenteHasta: ahora.Add(time.Minute), CertificadoVigenteHasta: ahora.Add(time.Minute)}
	fallo := &pgconn.PgError{Code: "42501", Message: "detalle técnico sintético que no sale"}
	for _, caso := range []struct {
		nombre     string
		cambiar    func(*poolIS16Prueba)
		denegacion bool
	}{
		{"envelope y commit confirmados", func(*poolIS16Prueba) {}, true},
		{"BEGIN técnico", func(p *poolIS16Prueba) { p.falloBegin = fallo }, false},
		{"SET técnico", func(p *poolIS16Prueba) { p.falloSet = fallo }, false},
		{"QUERY sin envelope", func(p *poolIS16Prueba) { p.falloQuery = fallo }, false},
		{"SAVEPOINT técnico", func(p *poolIS16Prueba) { p.falloSubBegin = fallo }, false},
		{"RELEASE técnico", func(p *poolIS16Prueba) { p.falloSubCommit = fallo }, false},
		{"COMMIT incierto", func(p *poolIS16Prueba) { p.falloCommit = fallo }, false},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			pool := &poolIS16Prueba{ahora: ahora, cuenta: []byte(`{"estado":"denegado","datos":null}`)}
			caso.cambiar(pool)
			p := &PostgreSQL{pool: pool, reloj: relojPrueba{ahora}, identificadores: fuenteIDsPruebaADMIN{}, seudonimizador: &seudIDsPruebaADMIN{}}
			ctx, err := ports.ConCorrelacionIncidenciasPeticion(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			c, err := p.ResolverCuentaADMIN(ctx, o)
			esperado := api.ErrConfiguracionIncompleta
			if caso.denegacion {
				esperado = api.ErrAccesoDenegado
			}
			if !errors.Is(err, esperado) || c != (CuentaADMIN{}) || pool.inicios != 1 || (!caso.denegacion && errors.Is(err, api.ErrAccesoDenegado)) || strings.Contains(err.Error(), "detalle técnico") {
				t.Fatal("clasificación técnica incorrecta, dato parcial o reintento de transacción")
			}
		})
	}
}

func TestFuenteIdentificadoresADMINExigeRaizPrivadaSinGitNiEnlaces(t *testing.T) {
	e := entradaIdentificadoresADMIN{Persona: "per_" + strings.Repeat("a", 32), Cuenta: "cta_" + strings.Repeat("b", 32), Ordinaria: "cta_" + strings.Repeat("c", 32), Certificado: strings.Repeat("d", 64), CA: strings.Repeat("e", 64),
		Espacio: "https://sintetico.example.invalid", Dominio: "idh_" + strings.Repeat("f", 32), Clave: "clave-sintetica", Version: 1, Fuente: "fuente:ids:sintetica", FuenteSHA: strings.Repeat("1", 64),
		SujetoID: "sujeto-sintetico-original", CuentaID: "admin-sintetica-original", OrdinariaID: "ordinaria-sintetica-original"}
	b, err := json.Marshal(struct {
		Version  uint64                        `json:"version"`
		Entradas []entradaIdentificadoresADMIN `json:"entradas"`
	}{1, []entradaIdentificadoresADMIN{e}})
	if err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256(b)
	sha := hex.EncodeToString(h[:])
	for _, caso := range []struct {
		nombre    string
		cambiar   func(*testing.T, string) string
		permitida bool
	}{
		{"ruta privada propia", func(_ *testing.T, ruta string) string { return ruta }, true},
		{"ruta relativa", func(t *testing.T, ruta string) string {
			cwd, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			r, err := filepath.Rel(cwd, ruta)
			if err != nil {
				t.Fatal(err)
			}
			return r
		}, false},
		{"directorio no privado", func(t *testing.T, ruta string) string {
			if os.Chmod(filepath.Dir(ruta), 0755) != nil {
				t.Fatal("chmod")
			}
			return ruta
		}, false},
		{"archivo no privado", func(t *testing.T, ruta string) string {
			if os.Chmod(ruta, 0640) != nil {
				t.Fatal("chmod")
			}
			return ruta
		}, false},
		{"worktree Git", func(t *testing.T, ruta string) string {
			if os.WriteFile(filepath.Join(filepath.Dir(ruta), ".git"), []byte("gitdir: sintetico"), 0600) != nil {
				t.Fatal("git")
			}
			return ruta
		}, false},
		{"repositorio Git", func(t *testing.T, ruta string) string {
			git := filepath.Join(filepath.Dir(ruta), ".git")
			if os.Mkdir(git, 0700) != nil || os.WriteFile(filepath.Join(git, "HEAD"), []byte("ref: refs/heads/sintetica"), 0600) != nil {
				t.Fatal("git")
			}
			return ruta
		}, false},
		{"enlace final", func(t *testing.T, ruta string) string {
			r := filepath.Join(filepath.Dir(ruta), "enlace.json")
			if os.Symlink(ruta, r) != nil {
				t.Fatal("symlink")
			}
			return r
		}, false},
		{"enlace ancestro", func(t *testing.T, ruta string) string {
			alias := filepath.Join(t.TempDir(), "alias")
			if os.Symlink(filepath.Dir(ruta), alias) != nil {
				t.Fatal("symlink")
			}
			return filepath.Join(alias, filepath.Base(ruta))
		}, false},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			root := t.TempDir()
			if os.Chmod(root, 0700) != nil {
				t.Fatal("chmod")
			}
			ruta := filepath.Join(root, "identificadores.json")
			if os.WriteFile(ruta, b, 0600) != nil {
				t.Fatal("escribir fixture")
			}
			f, err := NuevaFuenteIdentificadoresADMINDesdeArchivo(caso.cambiar(t, ruta), sha)
			if (err == nil) != caso.permitida || (!caso.permitida && f != nil) {
				t.Fatal("fuente aceptada fuera de su raíz privada")
			}
		})
	}
}
