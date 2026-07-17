package calculoexperiencia

import (
	"testing"

	"vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
	"vec-diputacion-granada/internal/shared/baremacion"
)

func TestPuntuacionDistingueRedondeoPorPeriodoYPorRegla(t *testing.T) {
	periodos := [][2]baremacion.FechaCivil{
		fechasPuntuacionPrueba(t, [3]int{2026, 1, 1}, [3]int{2026, 1, 1}),
		fechasPuntuacionPrueba(t, [3]int{2026, 1, 2}, [3]int{2026, 1, 2}),
	}
	casos := []struct {
		momento reglasbaremo.MomentoRedondeo
		final   string
		salida  string
	}{
		{reglasbaremo.RedondearPorPeriodo, "2/1", "2/1"},
		{reglasbaremo.RedondearPorRegla, "1/1", "1/1"},
	}
	for _, caso := range casos {
		t.Run(string(caso.momento), func(t *testing.T) {
			configuracion := configuracionPuntuacionPrueba(t)
			configuracion.unidad = debePuntuacion(reglasbaremo.NuevaPoliticaUnidadTemporal(
				reglasbaremo.UnidadTemporalDia, reglasbaremo.UnidadTemporalMes,
				debePuntuacion(baremacion.NuevoRacional(2, 1)), reglasbaremo.ExtremoFinalInclusivo,
			))
			configuracion.puntosPorUnidad = debePuntuacion(baremacion.PuntosDesdeMicropuntos(1))
			configuracion.redondeo = debePuntuacion(reglasbaremo.NuevaPoliticaRedondeo(
				caso.momento, baremacion.RedondeoHaciaArriba,
			))
			escenario := construirEscenarioPuntuacionPrueba(
				t, configuracion, periodos, baremacion.JornadaCompleta(), false,
			)
			_, resultado := ejecutarEscenarioPuntuacionPrueba(t, escenario)
			regla := resultado.Reglas()[0]
			if regla.BrutoExacto() != "1/1" || regla.Redondeo().SalidaExacta() != caso.salida ||
				regla.PuntosFinalesExactos() != caso.final {
				t.Fatalf("redondeo %s incoherente: %#v", caso.momento, regla)
			}
		})
	}
}

func TestPuntuacionRedondeoExactoFraccionalBloqueaSinParcial(t *testing.T) {
	periodo := fechasPuntuacionPrueba(t, [3]int{2026, 1, 1}, [3]int{2026, 1, 1})
	for _, momento := range []reglasbaremo.MomentoRedondeo{
		reglasbaremo.RedondearPorPeriodo,
		reglasbaremo.RedondearPorRegla,
	} {
		t.Run(string(momento), func(t *testing.T) {
			configuracion := configuracionPuntuacionPrueba(t)
			configuracion.unidad = debePuntuacion(reglasbaremo.NuevaPoliticaUnidadTemporal(
				reglasbaremo.UnidadTemporalDia, reglasbaremo.UnidadTemporalMes,
				debePuntuacion(baremacion.NuevoRacional(2, 1)), reglasbaremo.ExtremoFinalInclusivo,
			))
			configuracion.puntosPorUnidad = debePuntuacion(baremacion.PuntosDesdeMicropuntos(1))
			configuracion.redondeo = debePuntuacion(reglasbaremo.NuevaPoliticaRedondeo(
				momento, baremacion.RedondeoExacto,
			))
			escenario := construirEscenarioPuntuacionPrueba(
				t, configuracion, [][2]baremacion.FechaCivil{periodo},
				baremacion.JornadaCompleta(), false,
			)
			calculado, resultado := ejecutarEscenarioPuntuacionPrueba(t, escenario)
			if !calculado.bloqueada() || resultado.Estado() != ResultadoExperienciaBloqueado ||
				resultado.Fase() != FaseResultadoPuntuacion || len(resultado.Bloqueos()) != 1 ||
				len(resultado.Aplicaciones()) != 1 || len(resultado.Reglas()) != 0 ||
				len(resultado.Secciones()) != 0 {
				t.Fatalf("el exacto fraccional produjo parcial: %#v", resultado)
			}
			valor, presente := resultado.Bloqueos()[0].ValorExacto()
			if !presente || valor != "1/2" {
				t.Fatalf("valor exacto de bloqueo = %q, %v", valor, presente)
			}
		})
	}
}

func TestPuntuacionConectaTodosLosModosEnAmbosMomentos(t *testing.T) {
	periodo := fechasPuntuacionPrueba(t, [3]int{2026, 1, 1}, [3]int{2026, 1, 5})
	modos := []struct {
		modo     baremacion.ModoRedondeo
		esperado string
	}{
		{baremacion.RedondeoTruncar, "2/1"},
		{baremacion.RedondeoHaciaArriba, "3/1"},
		{baremacion.RedondeoMitadArriba, "3/1"},
		{baremacion.RedondeoMitadAlPar, "2/1"},
	}
	for _, momento := range []reglasbaremo.MomentoRedondeo{
		reglasbaremo.RedondearPorPeriodo,
		reglasbaremo.RedondearPorRegla,
	} {
		for _, caso := range modos {
			t.Run(string(momento)+"_"+string(caso.modo), func(t *testing.T) {
				configuracion := configuracionPuntuacionPrueba(t)
				configuracion.unidad = debePuntuacion(reglasbaremo.NuevaPoliticaUnidadTemporal(
					reglasbaremo.UnidadTemporalDia, reglasbaremo.UnidadTemporalMes,
					debePuntuacion(baremacion.NuevoRacional(2, 1)),
					reglasbaremo.ExtremoFinalInclusivo,
				))
				configuracion.puntosPorUnidad = debePuntuacion(baremacion.PuntosDesdeMicropuntos(1))
				configuracion.redondeo = debePuntuacion(reglasbaremo.NuevaPoliticaRedondeo(
					momento, caso.modo,
				))
				escenario := construirEscenarioPuntuacionPrueba(
					t, configuracion, [][2]baremacion.FechaCivil{periodo},
					baremacion.JornadaCompleta(), false,
				)
				_, resultado := ejecutarEscenarioPuntuacionPrueba(t, escenario)
				regla := resultado.Reglas()[0]
				if regla.BrutoExacto() != "5/2" ||
					regla.Redondeo().SalidaExacta() != caso.esperado ||
					regla.PuntosFinalesExactos() != caso.esperado {
					t.Fatalf("modo no conectado: %#v", regla.Redondeo())
				}
			})
		}
	}
}
