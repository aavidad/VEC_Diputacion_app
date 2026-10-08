package reglas

import (
	"slices"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/adapters/fichero"
	vecreglas "vec-diputacion-granada/internal/vec/reglas"
)

type relojIntentosV5 struct{}

func (relojIntentosV5) Ahora() time.Time { return time.Date(2026, 10, 9, 8, 0, 0, 0, time.UTC) }

// La versión 5 solo añade «comunica» a los resultados sin contacto (duda 151).
func TestCatalogoV5CuentaComunicaSinContacto(t *testing.T) {
	for _, caso := range []struct {
		fichero  string
		comunica bool
	}{{"bolsa_reglas.rrhh-20261002.v4.json", false}, {"bolsa_reglas.rrhh-20261008.v5.json", true}} {
		consulta, err := fichero.NuevaConsultaCatalogos("../../../../../data/demo/reglas/" + caso.fichero)
		if err != nil {
			t.Fatalf("%s: %v", caso.fichero, err)
		}
		resolutor, err := vecreglas.NuevoResolutor(vecreglas.Configuracion{
			Consulta: consulta, Metadatos: consulta, CatalogoID: vecreglas.CatalogoBolsa, ModuloID: vecreglas.ModuloBolsa,
			Reloj: relojIntentosV5{}, MunicipioSede: vecreglas.MunicipioSedeDiputacion,
		})
		if err != nil {
			t.Fatal(err)
		}
		politica, _, hay, err := NuevosIntentosContacto(resolutor).PoliticaIntentosTelefonicos(t.Context())
		if err != nil || !hay {
			t.Fatalf("%s: hay=%v err=%v", caso.fichero, hay, err)
		}
		if got := slices.Contains(politica.ResultadosSinContacto, "comunica"); got != caso.comunica || !politica.SinContacto("no_contesta") {
			t.Fatalf("%s: %v", caso.fichero, politica.ResultadosSinContacto)
		}
	}
}
