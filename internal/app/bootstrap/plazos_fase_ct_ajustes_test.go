package bootstrap

import (
	"context"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/adapters/fichero"
	"vec-diputacion-granada/internal/vec/reglas"
)

// ajustesPlazoFasePrueba devuelve la única versión de ajustes si ya estaba
// vigente en el instante pedido.
type ajustesPlazoFasePrueba struct{ version reglas.VersionAjustes }

func (a ajustesPlazoFasePrueba) AjustesVigentesEn(_ context.Context, id string, instante time.Time) (reglas.VersionAjustes, bool, error) {
	if id != a.version.CatalogoID || a.version.VigenteDesde.After(instante) {
		return reglas.VersionAjustes{}, false, nil
	}
	return a.version, true, nil
}

// La lista de expedientes calcula el plazo de cada uno con el valor vigente
// cuando entró en la fase: un cambio de RRHH no mueve los plazos que ya corren.
func TestPlazoFaseCTEnCursoConservaElValorAnteriorAlAjuste(t *testing.T) {
	cambio := time.Date(2026, 9, 20, 9, 0, 0, 0, time.UTC)
	ajustes := map[string]map[string]string{reglas.CTPlazoFiscalizacion: {reglas.CampoCantidad: "7"}}
	huella, err := reglas.HuellaAjustes(ajustes)
	if err != nil {
		t.Fatal(err)
	}
	consulta, err := fichero.NuevaConsultaCatalogos(rutaReglasCTEjemploPrueba)
	if err != nil {
		t.Fatal(err)
	}
	calendarios := calendariosPlazoFasePrueba(t)
	resolutor, err := reglas.NuevoResolutor(reglas.Configuracion{
		Consulta: consulta, Metadatos: consulta, CatalogoID: reglas.CatalogoContratacionTemporal,
		ModuloID: reglas.ModuloContratacionTemporal, Reloj: relojPresentacionReglasEjemplo,
		Calculadora: calculadoraPlazosCalendarios{consulta: calendarios}, MunicipioSede: reglas.MunicipioSedeDiputacion,
		Ajustes: ajustesPlazoFasePrueba{reglas.VersionAjustes{
			CatalogoID: reglas.CatalogoAjustesDe(reglas.CatalogoContratacionTemporal), Version: 1, HuellaSHA256: huella,
			VigenteDesde: cambio, Ajustes: ajustes,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	calculadora := nuevaCalculadoraPlazoFaseCT(resolutor)
	ahora := time.Date(2026, 9, 28, 8, 0, 0, 0, time.UTC)
	casos := []struct {
		desde    time.Time
		cantidad int
		ref      string
	}{
		{cambio.Add(-24 * time.Hour), 10, "vec.contratacion_temporal.reglas:1:c03.plazo_fiscalizacion"},
		{cambio.Add(24 * time.Hour), 7, "vec.contratacion_temporal.reglas.ajustes:1:c03.plazo_fiscalizacion"},
	}
	for _, caso := range casos {
		plazo, aplicable, err := calculadora.CalcularPlazoFase(t.Context(), ports.SolicitudPlazoFaseRRHH{
			Fase: "fiscalizacion", Desde: caso.desde, Ahora: ahora,
		})
		if err != nil || !aplicable || !plazo.Valido() || plazo.ReglaRef != caso.ref ||
			calendarios.recibida.Cantidad != caso.cantidad || !plazo.ReglaEjemplo {
			t.Fatalf("desde %v: %+v cantidad %d %v", caso.desde, plazo, calendarios.recibida.Cantidad, err)
		}
		if !strings.HasSuffix(plazo.ReglaRef, reglas.CTPlazoFiscalizacion) {
			t.Fatalf("referencia %s", plazo.ReglaRef)
		}
	}
}
