package calculoexperiencia

import (
	"vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
)

func validarResultadoExperienciaV1(r ResultadoExperienciaV1) error {
	if err := validarVinculosResultadoV1(r.vinculos); err != nil {
		return err
	}
	if err := validarSeleccionResultadoV1(r.seleccion); err != nil {
		return err
	}
	if len(r.intervalos) > maximoAplicacionesResultadoV1 ||
		len(r.aplicaciones) > maximoAplicacionesResultadoV1 ||
		len(r.reglas) > maximoReglasResultadoV1 ||
		len(r.secciones) > maximoSeccionesResultadoV1 ||
		len(r.bloqueos) > maximoBloqueosResultadoV1 {
		return nuevoError("resultado.colecciones", CodigoFueraDeLimites)
	}
	for _, intervalo := range r.intervalos {
		if err := validarIntervaloResultadoV1(intervalo); err != nil {
			return err
		}
	}
	for _, aplicacion := range r.aplicaciones {
		if err := validarAplicacionCalculadaResultadoV1(aplicacion); err != nil {
			return err
		}
	}
	for _, regla := range r.reglas {
		if err := validarReglaResultadoV1(regla); err != nil {
			return err
		}
	}
	for _, seccion := range r.secciones {
		if err := validarSeccionResultadoV1(seccion); err != nil {
			return err
		}
	}
	for _, bloqueo := range r.bloqueos {
		if err := validarBloqueoResultadoV1(bloqueo); err != nil {
			return err
		}
	}
	if !ordenResultadoV1Valido(r) {
		return nuevoError("resultado.orden", CodigoValorNoCanonico)
	}
	return validarEstadoResultadoV1(r)
}

func validarVinculosResultadoV1(v VinculosResultadoExperienciaV1) error {
	if v.motor.contrato != contratoMotorResultadoV1 ||
		v.motor.version != versionMotorResultadoV1 ||
		!huellaSHA256Canonica(v.motor.huellaContratoSHA256) ||
		v.plan.esquema != esquemaPlanResultadoV1 ||
		!huellaSHA256Canonica(v.plan.huellaSHA256) ||
		validarReferenciaVersionada(v.conjunto, "resultado.conjunto") != nil ||
		validarReferenciaVersionada(v.entrada.instantanea, "resultado.entrada") != nil ||
		!huellaSHA256Canonica(v.entrada.huellaContenidoSHA256) || !v.fechaCorte.EsValida() {
		return nuevoError("resultado.vinculos", CodigoValorNoCanonico)
	}
	if _, err := v.fechaCorte.Siguiente(); err != nil {
		return nuevoError("resultado.vinculos.fecha_corte", CodigoFueraDeLimites)
	}
	motorEsperado, err := vinculoMotorResultadoExperienciaV1()
	if err != nil || motorEsperado.huellaContratoSHA256 != v.motor.huellaContratoSHA256 {
		return nuevoError("resultado.vinculos.motor", CodigoValorNoCanonico)
	}
	planEsperado, err := huellaPlanResultadoExperienciaV1(v.motor, v.conjunto)
	if err != nil || planEsperado != v.plan.huellaSHA256 {
		return nuevoError("resultado.vinculos.plan", CodigoValorNoCanonico)
	}
	return nil
}

func validarSeleccionResultadoV1(s SeleccionResultadoExperienciaV1) error {
	if len(s.aplicaciones) > maximoAplicacionesResultadoV1 ||
		len(s.descartes) > maximoDescartesResultadoV1 ||
		len(s.sinCoincidencia) > maximoSinCoincidenciaResultadoV1 ||
		s.evaluaciones > uint64(maximoEvaluacionesSeleccion) {
		return nuevoError("resultado.seleccion", CodigoFueraDeLimites)
	}
	for indice, aplicacion := range s.aplicaciones {
		if validarReferenciaVersionada(aplicacion.tramo, "resultado.tramo") != nil ||
			!claveGobernadaValida(aplicacion.reglaClave) ||
			!claveGobernadaValida(aplicacion.grupoClave) ||
			!claveGobernadaValida(aplicacion.seccionClave) ||
			aplicacion.prioridad == 0 || !razonAplicacionValida(aplicacion.razon) {
			return nuevoError("resultado.seleccion.aplicacion", CodigoValorNoCanonico)
		}
		if indice > 0 && compararAplicacionesSeleccionResultadoV1(
			s.aplicaciones[indice-1], aplicacion,
		) >= 0 {
			return nuevoError("resultado.seleccion.aplicaciones", CodigoValorNoCanonico)
		}
	}
	for indice, descarte := range s.descartes {
		if validarReferenciaVersionada(descarte.tramo, "resultado.tramo") != nil ||
			!claveGobernadaValida(descarte.reglaClave) ||
			!claveGobernadaValida(descarte.grupoClave) ||
			!claveGobernadaValida(descarte.reglaSeleccionada) ||
			descarte.razon != RazonPrioridadInferior {
			return nuevoError("resultado.seleccion.descarte", CodigoValorNoCanonico)
		}
		if indice > 0 && compararDescartesSeleccionResultadoV1(
			s.descartes[indice-1], descarte,
		) >= 0 {
			return nuevoError("resultado.seleccion.descartes", CodigoValorNoCanonico)
		}
	}
	for indice, ausencia := range s.sinCoincidencia {
		if validarReferenciaVersionada(ausencia.tramo, "resultado.tramo") != nil ||
			ausencia.razon != RazonNingunaCoincidencia {
			return nuevoError("resultado.seleccion.sin_coincidencia", CodigoValorNoCanonico)
		}
		if indice > 0 && compararReferenciasSeleccion(
			s.sinCoincidencia[indice-1].tramo, ausencia.tramo,
		) >= 0 {
			return nuevoError("resultado.seleccion.sin_coincidencia", CodigoValorNoCanonico)
		}
	}
	return nil
}

func validarIntervaloResultadoV1(i IntervaloAplicacionResultadoExperienciaV1) error {
	if validarReferenciaVersionada(i.tramo, "resultado.tramo") != nil ||
		!claveGobernadaValida(i.reglaClave) || i.periodo.validar() != nil ||
		(i.extremo != reglasbaremo.ExtremoFinalInclusivo &&
			i.extremo != reglasbaremo.ExtremoFinalExclusivo) {
		return nuevoError("resultado.intervalo", CodigoValorNoCanonico)
	}
	if !i.tieneEfectivo {
		if i.efectivo.EsValido() || i.dias != 0 ||
			(i.razon != RazonPosteriorCorte && i.razon != RazonIntervaloVacio) {
			return nuevoError("resultado.intervalo", CodigoValorNoCanonico)
		}
		return nil
	}
	dias, err := i.efectivo.NumeroDias()
	if err != nil || dias <= 0 || uint64(dias) != i.dias || i.razon != "" {
		return nuevoError("resultado.intervalo", CodigoValorNoCanonico)
	}
	return nil
}

func validarAplicacionCalculadaResultadoV1(a AplicacionCalculadaResultadoExperienciaV1) error {
	if validarReferenciaVersionada(a.tramo, "resultado.tramo") != nil ||
		!claveGobernadaValida(a.reglaClave) || validarJornadaResultadoV1(a.jornada) != nil ||
		validarUnidadesResultadoV1(a.unidades) != nil || a.puntuacion.bruto.validar() != nil {
		return nuevoError("resultado.aplicacion", CodigoValorNoCanonico)
	}
	if a.puntuacion.tieneRedondeado {
		if a.puntuacion.redondeado.validar() != nil || !a.puntuacion.redondeado.esEntero() {
			return nuevoError("resultado.aplicacion.redondeado", CodigoValorNoCanonico)
		}
	} else if a.puntuacion.redondeado.canonico != "" {
		return nuevoError("resultado.aplicacion.redondeado", CodigoValorNoCanonico)
	}
	return nil
}

func validarJornadaResultadoV1(j JornadaResultadoExperienciaV1) error {
	if !j.origen.EsValida() || j.factor.validar() != nil ||
		(j.atestacionUsada && !j.atestacionPresente) {
		return nuevoError("resultado.jornada", CodigoValorNoCanonico)
	}
	uno, _ := nuevoExactoResultadoV1("1/1")
	origen, err := exactoResultadoDesdeJornadaV1(j.origen)
	if err != nil {
		return nuevoError("resultado.jornada.origen", CodigoValorNoCanonico)
	}
	comparacion, err := compararExactosResultadoV1(j.factor, uno)
	if err != nil || comparacion > 0 {
		return nuevoError("resultado.jornada.factor", CodigoValorNoCanonico)
	}
	switch j.modo {
	case reglasbaremo.JornadaProporcional:
		if err := exigirRazonJornadaV1(j, RazonJornadaProporcional, false); err != nil ||
			j.factor.canonico != origen.canonico {
			return nuevoError("resultado.jornada.proporcional", CodigoValorNoCanonico)
		}
	case reglasbaremo.JornadaIntegra:
		if err := exigirRazonJornadaV1(j, RazonJornadaIntegra, false); err != nil ||
			j.factor.canonico != uno.canonico {
			return nuevoError("resultado.jornada.integra", CodigoValorNoCanonico)
		}
	case reglasbaremo.JornadaIntegraDesdeUmbral:
		if j.razon != RazonUmbralAlcanzado && j.razon != RazonUmbralNoAlcanzado {
			return nuevoError("resultado.jornada.razon", CodigoValorNoCanonico)
		}
		esperado := origen.canonico
		if j.razon == RazonUmbralAlcanzado {
			esperado = uno.canonico
		}
		if j.factor.canonico != esperado || j.atestacionUsada {
			return nuevoError("resultado.jornada.umbral", CodigoValorNoCanonico)
		}
	case reglasbaremo.JornadaProtegidaIntegra:
		if j.razon != RazonProteccionAtestada && j.razon != RazonProteccionNoAtestada {
			return nuevoError("resultado.jornada.razon", CodigoValorNoCanonico)
		}
		if j.atestacionUsada != (j.razon == RazonProteccionAtestada) {
			return nuevoError("resultado.jornada.atestacion", CodigoValorNoCanonico)
		}
		esperado := origen.canonico
		if j.razon == RazonProteccionAtestada {
			esperado = uno.canonico
		}
		if j.factor.canonico != esperado {
			return nuevoError("resultado.jornada.proteccion", CodigoValorNoCanonico)
		}
	default:
		return nuevoError("resultado.jornada.modo", CodigoValorNoCanonico)
	}
	return nil
}

func exigirRazonJornadaV1(
	j JornadaResultadoExperienciaV1,
	esperada CodigoRazonResultadoExperienciaV1,
	permitirUso bool,
) error {
	if j.razon != esperada || (!permitirUso && j.atestacionUsada) {
		return nuevoError("resultado.jornada.razon", CodigoValorNoCanonico)
	}
	return nil
}

func validarUnidadesResultadoV1(u UnidadesAplicacionResultadoExperienciaV1) error {
	if u.exactas.validar() != nil || u.aportadas.validar() != nil || u.resto.validar() != nil ||
		(u.frontera != FronteraRestosResultadoExacta &&
			u.frontera != FronteraRestosResultadoPeriodo &&
			u.frontera != FronteraRestosResultadoRegla) ||
		!sumaExactosResultadoV1Coincide(u.aportadas, u.resto, u.exactas) {
		return nuevoError("resultado.unidades", CodigoValorNoCanonico)
	}
	if (u.frontera == FronteraRestosResultadoExacta || u.frontera == FronteraRestosResultadoRegla) &&
		(u.aportadas.canonico != u.exactas.canonico || u.resto.canonico != "0/1") {
		return nuevoError("resultado.unidades.frontera", CodigoValorNoCanonico)
	}
	return nil
}

func validarTopeResultadoV1(t TopeResultadoExperienciaV1) error {
	if t.antes.validar() != nil || t.despues.validar() != nil {
		return nuevoError("resultado.tope", CodigoValorNoCanonico)
	}
	if !t.limitado {
		if t.limite.canonico != "" || t.aplicado || t.antes.canonico != t.despues.canonico {
			return nuevoError("resultado.tope", CodigoValorNoCanonico)
		}
		return nil
	}
	if t.limite.validar() != nil {
		return nuevoError("resultado.tope.limite", CodigoValorNoCanonico)
	}
	comparacion, err := compararExactosResultadoV1(t.antes, t.limite)
	if err != nil || t.aplicado != (comparacion > 0) {
		return nuevoError("resultado.tope", CodigoValorNoCanonico)
	}
	esperado := t.antes.canonico
	if comparacion > 0 {
		esperado = t.limite.canonico
	}
	if t.despues.canonico != esperado {
		return nuevoError("resultado.tope.despues", CodigoValorNoCanonico)
	}
	return nil
}

func validarReglaResultadoV1(r ResultadoReglaExperienciaV1) error {
	if !claveGobernadaValida(r.seccionClave) || !claveGobernadaValida(r.reglaClave) ||
		r.unidadesAgregadas.validar() != nil || r.unidadesTrasRestos.validar() != nil ||
		r.restoRegla.validar() != nil ||
		!sumaExactosResultadoV1Coincide(r.unidadesTrasRestos, r.restoRegla, r.unidadesAgregadas) ||
		validarTopeResultadoV1(r.topeUnidades) != nil ||
		r.topeUnidades.antes.canonico != r.unidadesTrasRestos.canonico ||
		!r.coeficiente.EsValido() || r.bruto.validar() != nil ||
		validarRedondeoResultadoV1(r.redondeo) != nil ||
		r.redondeo.entrada.canonico != r.bruto.canonico ||
		validarTopeResultadoV1(r.topePuntos) != nil ||
		r.topePuntos.antes.canonico != r.redondeo.salida.canonico ||
		r.puntosFinales.validar() != nil || !r.puntosFinales.esEntero() ||
		r.puntosFinales.canonico != r.topePuntos.despues.canonico {
		return nuevoError("resultado.regla", CodigoValorNoCanonico)
	}
	return nil
}

func validarRedondeoResultadoV1(r RedondeoResultadoExperienciaV1) error {
	if (r.momento != reglasbaremo.RedondearPorPeriodo &&
		r.momento != reglasbaremo.RedondearPorRegla) || !r.modo.EsValido() ||
		r.entrada.validar() != nil || r.salida.validar() != nil || !r.salida.esEntero() {
		return nuevoError("resultado.redondeo", CodigoValorNoCanonico)
	}
	if r.momento == reglasbaremo.RedondearPorPeriodo {
		// La operacion se prueba en cada aplicacion. Aqui salida es la suma de
		// enteros ya redondeados, no un segundo redondeo de la suma bruta.
		return nil
	}
	esperada, err := redondearExactoResultadoV1(r.entrada, r.modo)
	if err != nil || esperada.canonico != r.salida.canonico {
		return nuevoError("resultado.redondeo.salida", CodigoValorNoCanonico)
	}
	return nil
}

func validarSeccionResultadoV1(s SubtotalSeccionResultadoExperienciaV1) error {
	if !claveGobernadaValida(s.seccionClave) || s.antesTope.validar() != nil ||
		validarTopeResultadoV1(s.tope) != nil || s.tope.antes.canonico != s.antesTope.canonico ||
		!s.puntosFinales.EsValido() {
		return nuevoError("resultado.seccion", CodigoValorNoCanonico)
	}
	final, err := exactoResultadoDesdeMicropuntosV1(s.puntosFinales.Micropuntos())
	if err != nil || final.canonico != s.tope.despues.canonico {
		return nuevoError("resultado.seccion.puntos_finales", CodigoValorNoCanonico)
	}
	return nil
}
