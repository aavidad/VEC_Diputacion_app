package calculoexperiencia

import (
	"testing"

	"vec-diputacion-granada/internal/shared/baremacion"
)

func TestPuntuacionEmiteReglasYSeccionesACeroSinAplicaciones(t *testing.T) {
	plan := debePuntuacion(Compilar(conjuntoOrdenadoCompilacion(t)))
	entrada := debePuntuacion(NuevaEntradaExperiencia(
		referenciaTokenSeleccionPrueba(t, prefijoInstantaneaEntrada, "ceros", 1), nil,
	))
	seleccion := debePuntuacion(seleccionarAplicaciones(plan, entrada))
	temporales := debePuntuacion(resolverAplicacionesTemporales(plan, entrada, seleccion))
	escenario := escenarioPuntuacionPrueba{plan, entrada, seleccion, temporales}
	_, resultado := ejecutarEscenarioPuntuacionPrueba(t, escenario)
	total, presente := resultado.Total()
	if len(resultado.Intervalos()) != 0 || len(resultado.Aplicaciones()) != 0 ||
		len(resultado.Reglas()) != 2 || len(resultado.Secciones()) != 2 ||
		resultado.Reglas()[0].UnidadesAgregadas() != "0/1" ||
		resultado.Reglas()[1].PuntosFinalesExactos() != "0/1" ||
		resultado.Reglas()[0].PuntosFinalesExactos() != "0/1" ||
		resultado.Secciones()[0].AntesTopeExacto() != "0/1" ||
		resultado.Secciones()[1].AntesTopeExacto() != "0/1" ||
		!presente || total.Micropuntos() != 0 {
		t.Fatalf("el cero no quedo explicitado: %#v", resultado)
	}
}

func TestPuntuacionConservaExclusionUnoAUnoConCalculoCero(t *testing.T) {
	configuracion := configuracionPuntuacionPrueba(t)
	plan := debePuntuacion(Compilar(conjuntoCompilacionPrueba(t, configuracion)))
	periodo := fechasPuntuacionPrueba(t, [3]int{2026, 8, 1}, [3]int{2026, 8, 2})
	tramo := tramoPuntuacionPrueba(
		t, plan, "excluido", periodo[0], periodo[1], baremacion.JornadaCompleta(), false,
	)
	entrada := debePuntuacion(NuevaEntradaExperiencia(
		referenciaTokenSeleccionPrueba(t, prefijoInstantaneaEntrada, "exclusion", 1),
		[]TramoExperiencia{tramo},
	))
	seleccion := debePuntuacion(seleccionarAplicaciones(plan, entrada))
	temporales := debePuntuacion(resolverAplicacionesTemporales(plan, entrada, seleccion))
	escenario := escenarioPuntuacionPrueba{plan, entrada, seleccion, temporales}
	_, resultado := ejecutarEscenarioPuntuacionPrueba(t, escenario)
	intervalo, aplicacion := resultado.Intervalos()[0], resultado.Aplicaciones()[0]
	if _, efectivo := intervalo.Efectivo(); efectivo || intervalo.Razon() != RazonPosteriorCorte ||
		aplicacion.Unidades().Exactas() != "0/1" ||
		aplicacion.Unidades().Aportadas() != "0/1" ||
		aplicacion.Puntuacion().BrutoExacto() != "0/1" ||
		resultado.Reglas()[0].PuntosFinalesExactos() != "0/1" {
		t.Fatalf("la exclusion no se conservo a cero: intervalo=%#v aplicacion=%#v",
			intervalo, aplicacion)
	}
}
