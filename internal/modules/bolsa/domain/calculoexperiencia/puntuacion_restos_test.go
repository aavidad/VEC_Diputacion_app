package calculoexperiencia

import (
	"testing"

	"vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
	"vec-diputacion-granada/internal/shared/baremacion"
)

func TestPuntuacionAplicaLasCuatroPoliticasDeRestos(t *testing.T) {
	periodo := fechasPuntuacionPrueba(t, [3]int{2026, 1, 1}, [3]int{2026, 2, 14})
	casos := []struct {
		modo            reglasbaremo.ModoRestos
		frontera        FronteraRestosResultadoExperienciaV1
		aportadas       string
		restoAplicacion string
		agregadas       string
		trasRestos      string
		restoRegla      string
		puntos          string
	}{
		{
			modo: reglasbaremo.RestosConservarExactos, frontera: FronteraRestosResultadoExacta,
			aportadas: "3/2", restoAplicacion: "0/1", agregadas: "3/2",
			trasRestos: "3/2", restoRegla: "0/1", puntos: "3/1",
		},
		{
			modo: reglasbaremo.RestosAcumularPorRegla, frontera: FronteraRestosResultadoRegla,
			aportadas: "3/2", restoAplicacion: "0/1", agregadas: "3/2",
			trasRestos: "3/2", restoRegla: "0/1", puntos: "3/1",
		},
		{
			modo: reglasbaremo.RestosDescartarPorPeriodo, frontera: FronteraRestosResultadoPeriodo,
			aportadas: "1/1", restoAplicacion: "1/2", agregadas: "3/2",
			trasRestos: "1/1", restoRegla: "1/2", puntos: "2/1",
		},
		{
			modo: reglasbaremo.RestosDescartarPorRegla, frontera: FronteraRestosResultadoRegla,
			aportadas: "3/2", restoAplicacion: "0/1", agregadas: "3/2",
			trasRestos: "1/1", restoRegla: "1/2", puntos: "2/1",
		},
	}
	for _, caso := range casos {
		t.Run(string(caso.modo), func(t *testing.T) {
			configuracion := configuracionPuntuacionPrueba(t)
			configuracion.unidad = debePuntuacion(reglasbaremo.NuevaPoliticaUnidadTemporal(
				reglasbaremo.UnidadTemporalDia, reglasbaremo.UnidadTemporalMes,
				debePuntuacion(baremacion.NuevoRacional(30, 1)), reglasbaremo.ExtremoFinalInclusivo,
			))
			configuracion.restos = debePuntuacion(reglasbaremo.NuevaPoliticaRestos(caso.modo))
			escenario := construirEscenarioPuntuacionPrueba(
				t, configuracion, [][2]baremacion.FechaCivil{periodo},
				baremacion.JornadaCompleta(), false,
			)
			_, resultado := ejecutarEscenarioPuntuacionPrueba(t, escenario)
			aplicacion := resultado.Aplicaciones()[0]
			regla := resultado.Reglas()[0]
			if aplicacion.Unidades().Exactas() != "3/2" ||
				aplicacion.Unidades().Aportadas() != caso.aportadas ||
				aplicacion.Unidades().Resto() != caso.restoAplicacion ||
				aplicacion.Unidades().Frontera() != caso.frontera ||
				regla.UnidadesAgregadas() != caso.agregadas ||
				regla.UnidadesTrasRestos() != caso.trasRestos ||
				regla.RestoRegla() != caso.restoRegla ||
				regla.PuntosFinalesExactos() != caso.puntos {
				t.Fatalf("restos distintos: aplicacion=%#v regla=%#v", aplicacion, regla)
			}
		})
	}
}

func TestPuntuacionDescartarPorPeriodoConservaRestoAgregadoExplicito(t *testing.T) {
	configuracion := configuracionPuntuacionPrueba(t)
	configuracion.unidad = debePuntuacion(reglasbaremo.NuevaPoliticaUnidadTemporal(
		reglasbaremo.UnidadTemporalDia, reglasbaremo.UnidadTemporalMes,
		debePuntuacion(baremacion.NuevoRacional(30, 1)), reglasbaremo.ExtremoFinalInclusivo,
	))
	configuracion.restos = debePuntuacion(reglasbaremo.NuevaPoliticaRestos(
		reglasbaremo.RestosDescartarPorPeriodo,
	))
	periodos := [][2]baremacion.FechaCivil{
		fechasPuntuacionPrueba(t, [3]int{2026, 1, 1}, [3]int{2026, 1, 20}),
		fechasPuntuacionPrueba(t, [3]int{2026, 2, 1}, [3]int{2026, 2, 20}),
	}
	escenario := construirEscenarioPuntuacionPrueba(
		t, configuracion, periodos, baremacion.JornadaCompleta(), false,
	)
	_, resultado := ejecutarEscenarioPuntuacionPrueba(t, escenario)
	regla := resultado.Reglas()[0]
	if regla.UnidadesAgregadas() != "4/3" || regla.UnidadesTrasRestos() != "0/1" ||
		regla.RestoRegla() != "4/3" || regla.PuntosFinalesExactos() != "0/1" {
		t.Fatalf("el descarte oculto el resto agregado: %#v", regla)
	}
}
