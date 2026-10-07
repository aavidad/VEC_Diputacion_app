package bootstrap

import (
	"context"
	"errors"
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

// La consulta retrospectiva de ajustes no acredita qué versión vio la
// transacción que inició el plazo. El cuadro falla cerrado hasta que CT
// conserve una instantánea durable al abrir cada tramo.
func TestPlazoFaseCTConAjustesNoRecalculaSinInstantanea(t *testing.T) {
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
	for _, desde := range []time.Time{cambio.Add(-24 * time.Hour), cambio.Add(24 * time.Hour)} {
		plazo, aplicable, err := calculadora.CalcularPlazoFase(t.Context(), ports.SolicitudPlazoFaseRRHH{
			Fase: "fiscalizacion", Desde: desde, Ahora: ahora,
		})
		if !errors.Is(err, reglas.ErrAjustesNoDisponibles) || aplicable || plazo.Valido() ||
			plazo.ReglaRef != "" || calendarios.recibida.Cantidad != 0 {
			t.Fatalf("desde %v se recalculó sin instantánea: %+v cantidad %d %v",
				desde, plazo, calendarios.recibida.Cantidad, err)
		}
	}
}

// Con ajustes compuestos la calculadora preparada también falla cerrado.
func TestPlazoFaseCTPreparadaConAjustesNoRecalcula(t *testing.T) {
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
			VigenteDesde: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), Ajustes: ajustes,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	preparada, err := nuevaCalculadoraPlazoFaseCT(resolutor).(ports.PreparadorPlazosFaseRRHH).PrepararPlazosFase(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	plazo, aplicable, err := preparada.CalcularPlazoFase(t.Context(), ports.SolicitudPlazoFaseRRHH{
		Fase: "fiscalizacion", Desde: time.Date(2026, 9, 15, 9, 0, 0, 0, time.UTC), Ahora: time.Date(2026, 9, 28, 8, 0, 0, 0, time.UTC),
	})
	if !errors.Is(err, reglas.ErrAjustesNoDisponibles) || aplicable || plazo.Valido() || calendarios.recibida.Cantidad != 0 {
		t.Fatalf("se recalculó con ajustes: %+v %v %v", plazo, aplicable, err)
	}
}
