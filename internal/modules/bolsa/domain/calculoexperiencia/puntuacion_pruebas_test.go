package calculoexperiencia

import (
	"testing"

	"vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
	"vec-diputacion-granada/internal/shared/baremacion"
)

type escenarioPuntuacionPrueba struct {
	plan       PlanExperiencia
	entrada    EntradaExperiencia
	seleccion  seleccionExperiencia
	temporales resultadoAplicacionesTemporales
}

func escenarioPuntuacionPruebaUnaRegla(
	t *testing.T,
	configuracion configuracionCompilacionPrueba,
	tramos ...TramoExperiencia,
) escenarioPuntuacionPrueba {
	t.Helper()
	plan := debePuntuacion(Compilar(conjuntoCompilacionPrueba(t, configuracion)))
	entrada := debePuntuacion(NuevaEntradaExperiencia(
		referenciaTokenSeleccionPrueba(t, prefijoInstantaneaEntrada, "puntuacion", 1), tramos,
	))
	seleccion := debePuntuacion(seleccionarAplicaciones(plan, entrada))
	temporales := debePuntuacion(resolverAplicacionesTemporales(plan, entrada, seleccion))
	return escenarioPuntuacionPrueba{plan, entrada, seleccion, temporales}
}

func tramoPuntuacionPrueba(
	t *testing.T,
	plan PlanExperiencia,
	etiqueta string,
	desde baremacion.FechaCivil,
	hasta baremacion.FechaCivil,
	jornada baremacion.FraccionJornada,
	atestado bool,
) TramoExperiencia {
	t.Helper()
	regla := plan.Reglas()[0]
	criterio := regla.Criterios()[0]
	atributo := debePuntuacion(NuevoAtributoCatalogado(
		criterio.Clave(), criterio.Catalogo(), criterio.Valores()[0],
	))
	atestacion := SinComputoIntegroAtestado()
	if atestado {
		atestacion = debePuntuacion(NuevoComputoIntegroAtestado(
			referenciaTokenSeleccionPrueba(t, prefijoAtestacionEntrada, "atestacion-"+etiqueta, 1),
		))
	}
	return debePuntuacion(NuevoTramoExperiencia(
		referenciaTokenSeleccionPrueba(t, prefijoTramoEntrada, "puntuacion-"+etiqueta, 1),
		tokenSeleccionPrueba(prefijoServicioEntrada, "puntuacion-"+etiqueta),
		debePuntuacion(NuevoPeriodoServicioCerrado(desde, hasta)), jornada, atestacion,
		[]AtributoCatalogado{atributo},
	))
}

func construirEscenarioPuntuacionPrueba(
	t *testing.T,
	configuracion configuracionCompilacionPrueba,
	periodos [][2]baremacion.FechaCivil,
	jornada baremacion.FraccionJornada,
	atestado bool,
) escenarioPuntuacionPrueba {
	t.Helper()
	plan := debePuntuacion(Compilar(conjuntoCompilacionPrueba(t, configuracion)))
	tramos := make([]TramoExperiencia, len(periodos))
	for indice, periodo := range periodos {
		tramos[indice] = tramoPuntuacionPrueba(
			t, plan, string(rune('a'+indice)), periodo[0], periodo[1], jornada, atestado,
		)
	}
	entrada := debePuntuacion(NuevaEntradaExperiencia(
		referenciaTokenSeleccionPrueba(t, prefijoInstantaneaEntrada, "puntuacion", 1), tramos,
	))
	seleccion := debePuntuacion(seleccionarAplicaciones(plan, entrada))
	temporales := debePuntuacion(resolverAplicacionesTemporales(plan, entrada, seleccion))
	return escenarioPuntuacionPrueba{plan, entrada, seleccion, temporales}
}

func ejecutarEscenarioPuntuacionPrueba(
	t *testing.T,
	escenario escenarioPuntuacionPrueba,
) (resultadoPuntuacionExperienciaV1, ResultadoExperienciaV1) {
	t.Helper()
	calculado := debePuntuacion(puntuarExperienciaV1(
		escenario.plan, escenario.entrada, escenario.seleccion, escenario.temporales,
	))
	registrador := debePuntuacion(nuevoRegistradorResultadoExperienciaV1(
		escenario.plan, escenario.entrada, escenario.seleccion,
	))
	if err := registrarPuntuacionExperienciaV1(registrador, calculado); err != nil {
		t.Fatal(err)
	}
	var resultado ResultadoExperienciaV1
	var err error
	if calculado.bloqueada() {
		resultado, err = registrador.sellarBloqueado(FaseResultadoPuntuacion)
	} else {
		resultado, err = registrador.sellarCompletado(calculado.total)
	}
	if err != nil {
		t.Fatal(err)
	}
	return calculado, resultado
}

func configuracionPuntuacionPrueba(t *testing.T) configuracionCompilacionPrueba {
	t.Helper()
	configuracion := configuracionBaseCompilacion(t)
	configuracion.unidad = debePuntuacion(reglasbaremo.NuevaPoliticaUnidadTemporal(
		reglasbaremo.UnidadTemporalDia, reglasbaremo.UnidadTemporalDia,
		debePuntuacion(baremacion.NuevoRacional(1, 1)), reglasbaremo.ExtremoFinalInclusivo,
	))
	configuracion.jornada = debePuntuacion(reglasbaremo.NuevaPoliticaJornada(
		reglasbaremo.JornadaProporcional,
	))
	configuracion.restos = debePuntuacion(reglasbaremo.NuevaPoliticaRestos(
		reglasbaremo.RestosConservarExactos,
	))
	configuracion.redondeo = debePuntuacion(reglasbaremo.NuevaPoliticaRedondeo(
		reglasbaremo.RedondearPorRegla, baremacion.RedondeoExacto,
	))
	configuracion.puntosPorUnidad = debePuntuacion(baremacion.PuntosDesdeMicropuntos(2))
	configuracion.maximoUnidades = reglasbaremo.SinLimiteUnidades()
	configuracion.maximoPuntos = reglasbaremo.SinLimitePuntos()
	configuracion.maximoSeccion = debePuntuacion(baremacion.PuntosDesdeMicropuntos(1_000_000_000))
	return configuracion
}

func fechasPuntuacionPrueba(
	t *testing.T,
	desde [3]int,
	hasta [3]int,
) [2]baremacion.FechaCivil {
	t.Helper()
	return [2]baremacion.FechaCivil{
		debePuntuacion(baremacion.NuevaFechaCivil(desde[0], desde[1], desde[2])),
		debePuntuacion(baremacion.NuevaFechaCivil(hasta[0], hasta[1], hasta[2])),
	}
}

func debePuntuacion[T interface{}](valor T, err error) T {
	if err != nil {
		panic(err)
	}
	return valor
}
