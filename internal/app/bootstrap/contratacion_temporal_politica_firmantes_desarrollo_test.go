package bootstrap

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	ctapplication "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	ctdomain "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

// La política de firmantes sale del mismo circuito vigente que fija el paso y
// sólo para su versión y huella exactas. Cualquier otra respuesta deniega.
func TestPoliticaFirmantesDesdeCircuitoVigente(t *testing.T) {
	bruto, err := os.ReadFile(rutaCircuitoFirmaRRHHV2Prueba)
	if err != nil {
		t.Fatal(err)
	}
	const no = `"misma_persona_en_dos_pasos": "false"`
	if !strings.Contains(string(bruto), no) {
		t.Fatal("el catálogo RRHH v2 ha cambiado")
	}
	permite := filepath.Join(t.TempDir(), "circuito_permite.json")
	if err := os.WriteFile(permite, []byte(strings.ReplaceAll(string(bruto), no, `"misma_persona_en_dos_pasos": "true"`)), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, caso := range []struct {
		ruta    string
		permite bool
	}{{rutaCircuitoFirmaCTEjemploPrueba, false}, {rutaCircuitoFirmaRRHHV2Prueba, false}, {permite, true}} {
		t.Run(filepath.Base(caso.ruta), func(t *testing.T) {
			fuente := fuenteCircuitoFirmaReglasDesarrollo{resolutor: resolverCircuitoFirmaR5Prueba(t, caso.ruta)}
			c, err := fuente.CircuitoFirma(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			p, err := fuente.PoliticaMismaPersonaEnPasos(t.Context(), c.CatalogoRef, c.HuellaCatalogo)
			if err != nil || p.Permite != caso.permite || p.CatalogoRef != c.CatalogoRef || p.CatalogoHuella != c.HuellaCatalogo {
				t.Fatalf("política distinta del circuito: %+v, %v", p, err)
			}
			permitido, err := ctapplication.ResolverPoliticaMismaPersonaEnPasos(t.Context(), fuente, c.CatalogoRef, c.HuellaCatalogo)
			if err != nil || permitido != caso.permite {
				t.Fatalf("la aplicación no resuelve la misma política: %v, %v", permitido, err)
			}
			otraVersion := c.CatalogoRef + "0"
			for nombre, par := range map[string][2]string{
				"otra huella":  {c.CatalogoRef, otraHuellaValida(c.HuellaCatalogo)},
				"otra versión": {otraVersion, c.HuellaCatalogo},
			} {
				if _, err := fuente.PoliticaMismaPersonaEnPasos(t.Context(), par[0], par[1]); !errors.Is(err, ctdomain.ErrCircuitoFirmaIncoherente) {
					t.Fatalf("%s aceptada: %v", nombre, err)
				}
			}
		})
	}
	sin := fuenteCircuitoFirmaReglasDesarrollo{}
	if _, err := sin.PoliticaMismaPersonaEnPasos(t.Context(), "vec.contratacion_temporal.circuito_firma:1", strings.Repeat("a", 64)); !errors.Is(err, ctapplication.ErrCircuitoFirmaNoDisponible) {
		t.Fatalf("sin circuito no debe haber política: %v", err)
	}
	cancelado, cancelar := context.WithCancel(t.Context())
	cancelar()
	con := fuenteCircuitoFirmaReglasDesarrollo{resolutor: resolverCircuitoFirmaR5Prueba(t, rutaCircuitoFirmaRRHHV2Prueba)}
	if _, err := con.PoliticaMismaPersonaEnPasos(cancelado, "vec.contratacion_temporal.circuito_firma:1", strings.Repeat("a", 64)); err == nil {
		t.Fatal("contexto cancelado aceptado")
	}
}

// otraHuellaValida devuelve una huella SHA-256 bien formada y distinta.
func otraHuellaValida(h string) string {
	if h == strings.Repeat("a", 64) {
		return strings.Repeat("b", 64)
	}
	return strings.Repeat("a", 64)
}
