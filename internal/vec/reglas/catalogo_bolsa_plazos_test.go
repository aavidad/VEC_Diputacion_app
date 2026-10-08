package reglas

import (
	"slices"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/adapters/fichero"
)

const rutaReglasBolsaV6Prueba = "../../../data/demo/reglas/bolsa_reglas.rrhh-20261008.v6.json"

func TestCatalogoBolsaV6DeclaraEdicionSoloDelPlazoDirecto(t *testing.T) {
	consulta, err := fichero.NuevaConsultaCatalogos(rutaReglasBolsaV6Prueba)
	if err != nil {
		t.Fatal(err)
	}
	r, err := NuevoResolutor(Configuracion{
		Consulta: consulta, Metadatos: consulta, CatalogoID: CatalogoBolsa, ModuloID: ModuloBolsa,
		Reloj: relojFijo(time.Date(2026, 10, 9, 8, 0, 0, 0, time.UTC)), MunicipioSede: MunicipioSedeDiputacion,
	})
	if err != nil {
		t.Fatal(err)
	}
	reglas, err := r.Reglas(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	var plazo *Regla
	for i := range reglas {
		if reglas[i].Edicion != nil && reglas[i].Clave != BolsaPlazoRespuesta {
			t.Fatalf("regla editable inesperada: %s", reglas[i].Clave)
		}
		if reglas[i].Clave == BolsaPlazoRespuesta {
			plazo = &reglas[i]
		}
	}
	if plazo == nil || plazo.Edicion == nil {
		t.Fatal("b05 no se puede ajustar")
	}
	if plazo.Referencia != "vec.bolsa.reglas:6:b05.plazo_respuesta" ||
		!slices.Equal(plazo.Edicion.Campos, []string{CampoCantidad, CampoUnidad, CampoComputo}) ||
		!slices.Equal(plazo.Edicion.OpcionesUnidad, []Unidad{UnidadDiasHabiles, UnidadDiasNaturales}) ||
		!slices.Equal(plazo.Edicion.OpcionesComputo, []Computo{ComputoAdministrativo, ComputoCivil}) ||
		plazo.Edicion.CantidadMinima != 1 || plazo.Edicion.CantidadMaxima != 30 {
		t.Fatalf("edición de b05 inesperada: %+v", plazo.Edicion)
	}
}
