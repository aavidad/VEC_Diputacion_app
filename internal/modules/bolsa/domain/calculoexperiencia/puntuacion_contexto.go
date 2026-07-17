package calculoexperiencia

import "vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"

type claveAplicacionPuntuacionV1 struct {
	referencia string
	version    uint64
	huella     string
	regla      string
}

type entradaAplicacionPuntuacionV1 struct {
	seleccion aplicacionSeleccion
	tramo     TramoExperiencia
	regla     reglasbaremo.ReglaExperiencia
	temporal  aplicacionTemporal
	exclusion exclusionAplicacionTemporal
	efectiva  bool
}

type contextoPuntuacionV1 struct {
	orden     []entradaAplicacionPuntuacionV1
	reglas    []reglasbaremo.ReglaExperiencia
	secciones []reglasbaremo.SeccionBaremo
}

func prepararContextoPuntuacionV1(
	plan PlanExperiencia,
	entrada EntradaExperiencia,
	seleccion seleccionExperiencia,
	temporales resultadoAplicacionesTemporales,
) (contextoPuntuacionV1, error) {
	contexto, err := prepararContextoComunPuntuacionV1(plan, entrada, seleccion, temporales)
	if err != nil {
		return contextoPuntuacionV1{}, err
	}
	if temporales.bloqueada() {
		return contextoPuntuacionV1{}, nuevoError(
			"puntuacion.fase_previa", CodigoContextoIncompatible,
		)
	}
	return contexto, nil
}

func prepararContextoComunPuntuacionV1(
	plan PlanExperiencia,
	entrada EntradaExperiencia,
	seleccion seleccionExperiencia,
	temporales resultadoAplicacionesTemporales,
) (contextoPuntuacionV1, error) {
	if err := plan.Validar(); err != nil {
		return contextoPuntuacionV1{}, err
	}
	if err := entrada.Validar(); err != nil {
		return contextoPuntuacionV1{}, err
	}
	seleccionEsperada, err := seleccionarAplicaciones(plan, entrada)
	if err != nil {
		return contextoPuntuacionV1{}, err
	}
	if !seleccionesPuntuacionIguales(seleccionEsperada, seleccion) {
		return contextoPuntuacionV1{}, nuevoError(
			"puntuacion.seleccion", CodigoContextoIncompatible,
		)
	}
	if seleccion.bloqueada() {
		return contextoPuntuacionV1{}, nuevoError(
			"puntuacion.fase_previa", CodigoContextoIncompatible,
		)
	}
	if _, bloqueos, err := materializarSeleccionResultadoV1(seleccion); err != nil || len(bloqueos) != 0 {
		return contextoPuntuacionV1{}, nuevoError(
			"puntuacion.seleccion", CodigoContextoIncompatible,
		)
	}
	seleccionadas := seleccion.aplicacionesCopia()
	if temporales.aplicacionesProcesadas != numeroAplicacionesTemporales(len(seleccionadas)) ||
		temporales.eventosProcesados > maximoEventosTemporales ||
		len(temporales.aplicaciones)+len(temporales.exclusiones) != len(seleccionadas) {
		return contextoPuntuacionV1{}, errorContextoPuntuacionV1()
	}

	tramos := make(map[string]TramoExperiencia, len(entrada.tramos))
	for _, tramo := range entrada.tramos {
		tramos[tramo.referencia.Referencia()] = tramo
	}
	reglas := make(map[string]reglasbaremo.ReglaExperiencia, len(plan.reglas))
	for _, regla := range plan.reglas {
		reglas[regla.Clave()] = regla
	}
	efectivas := make(map[claveAplicacionPuntuacionV1]aplicacionTemporal, len(temporales.aplicaciones))
	for _, temporal := range temporales.aplicaciones {
		clave := nuevaClaveAplicacionPuntuacionV1(temporal.tramo.referencia, temporal.regla.Clave())
		if _, existe := efectivas[clave]; existe {
			return contextoPuntuacionV1{}, errorContextoPuntuacionV1()
		}
		efectivas[clave] = temporal
	}
	exclusiones := make(map[claveAplicacionPuntuacionV1]exclusionAplicacionTemporal, len(temporales.exclusiones))
	for _, exclusion := range temporales.exclusiones {
		clave := nuevaClaveAplicacionPuntuacionV1(exclusion.tramo, exclusion.reglaClave)
		if _, existe := exclusiones[clave]; existe {
			return contextoPuntuacionV1{}, errorContextoPuntuacionV1()
		}
		if _, existe := efectivas[clave]; existe {
			return contextoPuntuacionV1{}, errorContextoPuntuacionV1()
		}
		exclusiones[clave] = exclusion
	}

	contexto := contextoPuntuacionV1{
		orden:     make([]entradaAplicacionPuntuacionV1, 0, len(seleccionadas)),
		reglas:    append([]reglasbaremo.ReglaExperiencia(nil), plan.reglas...),
		secciones: append([]reglasbaremo.SeccionBaremo(nil), plan.secciones...),
	}
	for _, elegida := range seleccionadas {
		tramo, existe := tramos[elegida.tramo.Referencia()]
		if !existe || !referenciasPlanIguales(tramo.referencia, elegida.tramo) {
			return contextoPuntuacionV1{}, errorContextoPuntuacionV1()
		}
		regla, existe := reglas[elegida.reglaClave]
		if !existe || regla.GrupoConcurrenciaClave() != elegida.grupoClave ||
			regla.SeccionClave() != elegida.seccionClave ||
			regla.PrioridadConcurrencia() != elegida.prioridad ||
			!razonAplicacionTemporalValida(elegida.razon) {
			return contextoPuntuacionV1{}, errorContextoPuntuacionV1()
		}
		clave := nuevaClaveAplicacionPuntuacionV1(elegida.tramo, elegida.reglaClave)
		temporal, esEfectiva := efectivas[clave]
		exclusion, estaExcluida := exclusiones[clave]
		if esEfectiva == estaExcluida {
			return contextoPuntuacionV1{}, errorContextoPuntuacionV1()
		}
		if esEfectiva {
			if !aplicacionTemporalPuntuacionValida(plan, elegida, tramo, regla, temporal) {
				return contextoPuntuacionV1{}, errorContextoPuntuacionV1()
			}
			delete(efectivas, clave)
		} else {
			if !exclusionTemporalPuntuacionValida(plan, elegida, tramo, regla, exclusion) {
				return contextoPuntuacionV1{}, errorContextoPuntuacionV1()
			}
			delete(exclusiones, clave)
		}
		contexto.orden = append(contexto.orden, entradaAplicacionPuntuacionV1{
			seleccion: elegida, tramo: tramo, regla: regla,
			temporal: temporal, exclusion: exclusion, efectiva: esEfectiva,
		})
	}
	if len(efectivas) != 0 || len(exclusiones) != 0 {
		return contextoPuntuacionV1{}, errorContextoPuntuacionV1()
	}
	if !bloqueosTemporalesPuntuacionValidos(plan, temporales) {
		return contextoPuntuacionV1{}, errorContextoPuntuacionV1()
	}
	return contexto, nil
}

func bloqueosTemporalesPuntuacionValidos(
	plan PlanExperiencia,
	temporales resultadoAplicacionesTemporales,
) bool {
	grupos := make(map[string]reglasbaremo.GrupoConcurrenciaExperiencia, len(plan.gruposConcurrencia))
	for _, grupo := range plan.gruposConcurrencia {
		grupos[grupo.Clave()] = grupo
	}
	presupuesto := presupuestoAplicacionesTemporales{limites: limitesAplicacionesTemporales{
		aplicaciones: maximoAplicacionesTemporales,
		eventos:      maximoEventosTemporales,
	}}
	esperado := resultadoAplicacionesTemporales{}
	if err := detectarSolapesTemporales(
		temporales.aplicacionesCopia(), grupos, &presupuesto, &esperado,
	); err != nil || presupuesto.eventosConsumidos != temporales.eventosProcesados ||
		len(esperado.bloqueos) != len(temporales.bloqueos) {
		return false
	}
	ordenarResultadoAplicacionesTemporales(&esperado)
	for indice, bloqueo := range esperado.bloqueos {
		recibido := temporales.bloqueos[indice]
		if bloqueo.codigo != recibido.codigo || bloqueo.grupoClave != recibido.grupoClave ||
			!referenciasPlanIguales(bloqueo.tramoPrimero, recibido.tramoPrimero) ||
			!referenciasPlanIguales(bloqueo.tramoSegundo, recibido.tramoSegundo) {
			return false
		}
	}
	return true
}

func nuevaClaveAplicacionPuntuacionV1(
	tramo reglasbaremo.ReferenciaVersionada,
	regla string,
) claveAplicacionPuntuacionV1 {
	return claveAplicacionPuntuacionV1{
		referencia: tramo.Referencia(), version: tramo.Version(),
		huella: tramo.HuellaSHA256(), regla: regla,
	}
}

func aplicacionTemporalPuntuacionValida(
	plan PlanExperiencia,
	seleccion aplicacionSeleccion,
	tramo TramoExperiencia,
	regla reglasbaremo.ReglaExperiencia,
	temporal aplicacionTemporal,
) bool {
	if !tramosPuntuacionIguales(tramo, temporal.tramo) ||
		!reglasPuntuacionIguales(regla, temporal.regla) || temporal.razon != seleccion.razon {
		return false
	}
	esperado, efectivo, err := normalizarPeriodoEfectivo(
		tramo.periodo, plan.fechaCorteInclusiva, regla.UnidadTemporal().ExtremoFinal(),
	)
	if err != nil || !efectivo || esperado.Desde() != temporal.intervalo.Desde() ||
		esperado.Hasta() != temporal.intervalo.Hasta() {
		return false
	}
	dias, err := esperado.NumeroDias()
	return err == nil && dias > 0 && dias == temporal.dias
}

func exclusionTemporalPuntuacionValida(
	plan PlanExperiencia,
	seleccion aplicacionSeleccion,
	tramo TramoExperiencia,
	regla reglasbaremo.ReglaExperiencia,
	exclusion exclusionAplicacionTemporal,
) bool {
	if !referenciasPlanIguales(exclusion.tramo, seleccion.tramo) ||
		exclusion.reglaClave != regla.Clave() ||
		exclusion.grupoClave != regla.GrupoConcurrenciaClave() {
		return false
	}
	_, efectivo, err := normalizarPeriodoEfectivo(
		tramo.periodo, plan.fechaCorteInclusiva, regla.UnidadTemporal().ExtremoFinal(),
	)
	if err != nil || efectivo {
		return false
	}
	esperada := nuevaExclusionAplicacionTemporal(plan.fechaCorteInclusiva, tramo, regla)
	return exclusion.razon == esperada.razon
}

func errorContextoPuntuacionV1() error {
	return nuevoError("puntuacion.contexto", CodigoContextoIncompatible)
}
