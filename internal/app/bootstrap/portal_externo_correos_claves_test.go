package bootstrap

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/app/separacionportales"
	usuariosseguridad "vec-diputacion-granada/internal/modules/usuarios/adapters/seguridad"
)

type filaPoblacionCorreosPrueba struct {
	vacia bool
	err   error
}

func (f filaPoblacionCorreosPrueba) Scan(destino ...any) error {
	if f.err != nil {
		return f.err
	}
	*(destino[0].(*bool)) = f.vacia
	return nil
}

type consultaPoblacionCorreosPrueba struct {
	fila filaPoblacionCorreosPrueba
	sql  string
}

func (c *consultaPoblacionCorreosPrueba) QueryRow(_ context.Context, sql string, _ ...any) pgx.Row {
	c.sql = sql
	return c.fila
}

func materialCorreosExternoPrueba(t *testing.T, semilla byte) config.Config {
	t.Helper()
	raiz := t.TempDir()
	if err := os.MkdirAll(filepath.Dir(filepath.Join(raiz, filepath.FromSlash(config.DevelopmentExternalMailSeedRelativePath))), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(raiz, separacionportales.FicheroMarcaPortal), []byte(`{"version":1,"portal":"externo"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	clave := make([]byte, 32)
	for i := range clave {
		clave[i] = semilla
	}
	if err := os.WriteFile(filepath.Join(raiz, filepath.FromSlash(config.DevelopmentExternalMailSeedRelativePath)), clave, 0o600); err != nil {
		t.Fatal(err)
	}
	return config.Config{ExecutionProfile: config.ExecutionProfileDevelopment, AuthMode: config.AuthModeDevelopment,
		DevelopmentGuard: config.DevelopmentGuardAcknowledgement, PortalProceso: "externo", DevelopmentMaterialDir: raiz}
}

func TestClavesCorreosExternosSoloConPoblacionSinClavesAjenas(t *testing.T) {
	cfg := materialCorreosExternoPrueba(t, 0x51)
	consulta := &consultaPoblacionCorreosPrueba{fila: filaPoblacionCorreosPrueba{vacia: true}}
	fuente, err := nuevaFuenteClavesCorreosPortalExternoConConsulta(t.Context(), cfg, consulta)
	if err != nil || fuente == nil || consulta.sql != sondaCorreosPortalExternoSinClavesAjenasSQL {
		t.Fatalf("población no acreditada: %v", err)
	}
	claves, err := fuente.CargarClavesCorreos(t.Context())
	if err != nil || claves.CifradoActivo.Material == claves.Igualdad.Material ||
		claves.Igualdad.Material == claves.SemanticaActiva.Material ||
		claves.SemanticaActiva.Material == claves.CodigoActivo.Material ||
		claves.CifradoActivo.Ref == "clave:kms:desarrollo:usuarios-correos-cifrado:v1" {
		t.Fatal("ámbitos o referencias de correos externos no separados")
	}
	otra := materialCorreosExternoPrueba(t, 0x52)
	segunda, err := nuevaFuenteClavesCorreosPortalExternoConConsulta(t.Context(), otra, consulta)
	if err != nil {
		t.Fatal(err)
	}
	clavesOtras, _ := segunda.CargarClavesCorreos(t.Context())
	if claves.CifradoActivo.Material == clavesOtras.CifradoActivo.Material {
		t.Fatal("materiales distintos derivaron la misma clave")
	}
	fuente.borrar()
	if _, err := fuente.CargarClavesCorreos(t.Context()); err == nil {
		t.Fatal("fuente cerrada entregó claves")
	}
}

func TestClavesCorreosExternosDeniegaPoblacionAnteriorOConsultaFallida(t *testing.T) {
	cfg := materialCorreosExternoPrueba(t, 0x53)
	for _, fila := range []filaPoblacionCorreosPrueba{{vacia: false}, {err: errors.New("consulta indisponible")}} {
		consulta := &consultaPoblacionCorreosPrueba{fila: fila}
		if fuente, err := nuevaFuenteClavesCorreosPortalExternoConConsulta(t.Context(), cfg, consulta); fuente != nil || err == nil {
			t.Fatal("clave habilitada sin preflight positivo")
		}
	}
	cfg.PortalProceso = "interno"
	if fuente, err := nuevaFuenteClavesCorreosPortalExternoConConsulta(t.Context(), cfg, &consultaPoblacionCorreosPrueba{fila: filaPoblacionCorreosPrueba{vacia: true}}); fuente != nil || err == nil {
		t.Fatal("material externo usado desde el portal interno")
	}
}

func TestClavesCorreosExternosCierreConcurrenteNoEntregaClaveParcial(t *testing.T) {
	cfg := materialCorreosExternoPrueba(t, 0x54)
	fuente, err := nuevaFuenteClavesCorreosPortalExternoConConsulta(t.Context(), cfg,
		&consultaPoblacionCorreosPrueba{fila: filaPoblacionCorreosPrueba{vacia: true}})
	if err != nil {
		t.Fatal(err)
	}
	var grupo sync.WaitGroup
	errores := make(chan error, 64)
	for i := 0; i < cap(errores); i++ {
		grupo.Add(1)
		go func() {
			defer grupo.Done()
			claves, err := fuente.CargarClavesCorreos(context.Background())
			if err == nil && claves.CifradoActivo.Material == ([32]byte{}) {
				errores <- errors.New("clave parcialmente borrada")
			} else if err != nil && !errors.Is(err, ErrClavesCorreosPortalExternoNoDisponibles) {
				errores <- err
			}
		}()
	}
	fuente.borrar()
	grupo.Wait()
	close(errores)
	for err := range errores {
		t.Fatal(err)
	}
}

// Derivación histórica: la ruta del fichero no entra en los cuatro ámbitos.
func fuenteHistoricaCorreosExternoPrueba(valor byte) *fuenteClavesCorreosPortalExterno {
	var semilla [32]byte
	for i := range semilla {
		semilla[i] = valor
	}
	defer clear(semilla[:])
	clave := func(ambito string) usuariosseguridad.ClaveCorreo {
		return usuariosseguridad.ClaveCorreo{
			Ref:      "clave:kms:desarrollo:usuarios-correos-externo-" + ambito + ":v1",
			Material: derivarClaveDesarrollo(semilla, "vec.kms.desarrollo.usuarios-correos.externo."+ambito+".v1"),
		}
	}
	return &fuenteClavesCorreosPortalExterno{claves: usuariosseguridad.ClavesCorreos{
		CifradoActivo: clave("cifrado"), Igualdad: clave("igualdad"),
		SemanticaActiva: clave("semantica"), CodigoActivo: clave("codigo"),
	}}
}

func TestClavesCorreosExternosRutaPropiaConservaClavesYSobreAnterior(t *testing.T) {
	cfg := materialCorreosExternoPrueba(t, 0x55)
	historica := fuenteHistoricaCorreosExternoPrueba(0x55)
	defer historica.borrar()
	antes, err := historica.CargarClavesCorreos(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	adaptadorAnterior, err := usuariosseguridad.NuevoAdaptadorCorreos(historica, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	persona, correo := "persona:00000000000000000000000000000007", "correo:00000000000000000000000000000017"
	direccion := []byte("ines.moreno@example.invalid")
	sobre, err := adaptadorAnterior.CifrarDireccionCorreo(t.Context(), persona, correo, 1, direccion)
	if err != nil {
		t.Fatal(err)
	}
	fuente, err := nuevaFuenteClavesCorreosPortalExternoConConsulta(t.Context(), cfg, &consultaPoblacionCorreosPrueba{fila: filaPoblacionCorreosPrueba{vacia: true}})
	if err != nil {
		t.Fatal(err)
	}
	defer fuente.borrar()
	despues, err := fuente.CargarClavesCorreos(t.Context())
	if err != nil || antes.CifradoActivo != despues.CifradoActivo || antes.Igualdad != despues.Igualdad || antes.SemanticaActiva != despues.SemanticaActiva || antes.CodigoActivo != despues.CodigoActivo {
		t.Fatal("el cambio de ruta alteró claves o referencias")
	}
	adaptadorNuevo, err := usuariosseguridad.NuevoAdaptadorCorreos(fuente, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	leida := false
	err = adaptadorNuevo.ConDireccionCorreoDescifrada(t.Context(), persona, sobre, func(claro []byte) error {
		leida = bytes.Equal(claro, direccion)
		return nil
	})
	if err != nil || !leida {
		t.Fatal("el sobre anterior no se recupera con la ruta propia")
	}
}

func TestClavesCorreosExternosNoAceptaResiduoKMSNiSemillaAusente(t *testing.T) {
	t.Run("semilla ausente", func(t *testing.T) {
		cfg := materialCorreosExternoPrueba(t, 0x56)
		if err := os.Remove(cfg.DevelopmentPaths().SemillaCorreosExterna); err != nil {
			t.Fatal(err)
		}
		if f, err := nuevaFuenteClavesCorreosPortalExternoConConsulta(t.Context(), cfg, &consultaPoblacionCorreosPrueba{fila: filaPoblacionCorreosPrueba{vacia: true}}); err == nil || f != nil {
			t.Fatal("se admitió la ausencia de semilla propia")
		}
	})
	t.Run("residuo del KMS", func(t *testing.T) {
		cfg := materialCorreosExternoPrueba(t, 0x57)
		ruta := cfg.DevelopmentPaths().KMSSecret
		if err := os.MkdirAll(filepath.Dir(ruta), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(ruta, bytes.Repeat([]byte{0x57}, 32), 0o600); err != nil {
			t.Fatal(err)
		}
		if f, err := nuevaFuenteClavesCorreosPortalExternoConConsulta(t.Context(), cfg, &consultaPoblacionCorreosPrueba{fila: filaPoblacionCorreosPrueba{vacia: true}}); err == nil || f != nil {
			t.Fatal("se admitió material interno junto a la semilla propia")
		}
	})
}
