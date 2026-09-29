package bootstrap

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/app/separacionportales"
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
	if err := os.MkdirAll(filepath.Join(raiz, "kms"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(raiz, separacionportales.FicheroMarcaPortal), []byte(`{"version":1,"portal":"externo"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	clave := make([]byte, 32)
	for i := range clave {
		clave[i] = semilla
	}
	if err := os.WriteFile(filepath.Join(raiz, filepath.FromSlash(config.DevelopmentKMSSecretRelativePath)), clave, 0o600); err != nil {
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
