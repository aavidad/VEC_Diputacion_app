package reglas

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/vec/adapters/fichero"
	vecreglas "vec-diputacion-granada/internal/vec/reglas"
)

const rutaCatalogoIntentosPrueba = "../../../../../data/demo/reglas/bolsa_reglas.ejemplo.demo.json"

type relojIntentosPrueba struct{}

func (relojIntentosPrueba) Ahora() time.Time { return time.Date(2026, 9, 28, 8, 0, 0, 0, time.UTC) }

func resolutorIntentosPrueba(t *testing.T, ruta string) *vecreglas.Resolutor {
	t.Helper()
	consulta, err := fichero.NuevaConsultaCatalogos(ruta)
	if err != nil {
		t.Fatal(err)
	}
	resolutor, err := vecreglas.NuevoResolutor(vecreglas.Configuracion{
		Consulta: consulta, Metadatos: consulta, CatalogoID: vecreglas.CatalogoBolsa, ModuloID: vecreglas.ModuloBolsa, Reloj: relojIntentosPrueba{},
	})
	if err != nil {
		t.Fatal(err)
	}
	return resolutor
}

func TestIntentosContactoLeeLasReglasDelCatalogoDeEjemplo(t *testing.T) {
	politica, reglas, configurada, err := NuevosIntentosContacto(resolutorIntentosPrueba(t, rutaCatalogoIntentosPrueba)).PoliticaIntentosTelefonicos(t.Context())
	if err != nil || !configurada {
		t.Fatalf("configurada=%v err=%v", configurada, err)
	}
	if politica.IntentosPorProceso != 2 || politica.Procesos != 2 || politica.SeparacionMinima != 2*time.Hour ||
		politica.ControlSeparacion != dominiobolsa.ControlReglaImpedir ||
		!slices.Equal(politica.ResultadosSinContacto, []string{"no_contesta", "buzon", "numero_erroneo"}) {
		t.Fatalf("política: %+v", politica)
	}
	f := politica.Franja
	if f.Zona == nil || f.DesdeMinuto != 540 || f.HastaMinuto != 840 || !f.SoloDiasHabiles || f.Control != dominiobolsa.ControlReglaAdvertir {
		t.Fatalf("franja: %+v", f)
	}
	var claves []string
	for _, r := range reglas {
		if !strings.HasPrefix(r.Referencia, vecreglas.CatalogoBolsa+":") || r.Etiqueta == "" {
			t.Fatalf("regla sin referencia o etiqueta: %+v", r)
		}
		claves = append(claves, r.Clave)
	}
	if !slices.Equal(claves, []string{vecreglas.BolsaIntentosContacto, vecreglas.BolsaSeparacionIntentos, vecreglas.BolsaProcesosSinContacto, vecreglas.BolsaFranjaLlamadas, vecreglas.BolsaCorreoNoAbrePlazo}) {
		t.Fatalf("reglas mostradas: %v", claves)
	}
}

func TestIntentosContactoSinCatalogoNoControla(t *testing.T) {
	_, _, configurada, err := NuevosIntentosContacto(nil).PoliticaIntentosTelefonicos(t.Context())
	if configurada || err != nil {
		t.Fatalf("sin catálogo: configurada=%v err=%v", configurada, err)
	}
}

func TestIntentosContactoRechazaCatalogoIncompleto(t *testing.T) {
	contenido, err := os.ReadFile(rutaCatalogoIntentosPrueba)
	if err != nil {
		t.Fatal(err)
	}
	for nombre, cambio := range map[string][2]string{
		"sin control de separación": {`,
          "control": "impedir"`, ``},
		"sin resultados sin contacto": {`"resultados_sin_contacto": "no_contesta,buzon,numero_erroneo",`, ``},
		"días desconocidos":           {`"dias": "habiles"`, `"dias": "laborables"`},
	} {
		if !strings.Contains(string(contenido), cambio[0]) {
			t.Fatalf("%s: el catálogo de ejemplo ya no tiene %q", nombre, cambio[0])
		}
		ruta := filepath.Join(t.TempDir(), "reglas.demo.json")
		if err := os.WriteFile(ruta, []byte(strings.Replace(string(contenido), cambio[0], cambio[1], 1)), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, _, _, err := NuevosIntentosContacto(resolutorIntentosPrueba(t, ruta)).PoliticaIntentosTelefonicos(t.Context()); err == nil {
			t.Errorf("%s: catálogo aceptado", nombre)
		}
	}
}
