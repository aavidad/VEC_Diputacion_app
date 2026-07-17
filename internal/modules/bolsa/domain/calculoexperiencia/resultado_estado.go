package calculoexperiencia

import (
	"vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
	"vec-diputacion-granada/internal/shared/baremacion"
)

func validarBloqueoResultadoV1(b BloqueoResultadoExperienciaV1) error {
	if len(b.tramos) > maximoReferenciasBloqueoV1 || len(b.reglas) > maximoReferenciasBloqueoV1 {
		return nuevoError("resultado.bloqueo", CodigoFueraDeLimites)
	}
	for indice, tramo := range b.tramos {
		if validarReferenciaVersionada(tramo, "resultado.bloqueo.tramo") != nil ||
			(indice > 0 && compararReferenciasSeleccion(b.tramos[indice-1], tramo) >= 0) {
			return nuevoError("resultado.bloqueo.tramos", CodigoValorNoCanonico)
		}
	}
	for indice, regla := range b.reglas {
		if !claveGobernadaValida(regla) ||
			(indice > 0 && b.reglas[indice-1] >= regla) {
			return nuevoError("resultado.bloqueo.reglas", CodigoValorNoCanonico)
		}
	}
	if b.grupoClave != "" && !claveGobernadaValida(b.grupoClave) {
		return nuevoError("resultado.bloqueo.grupo", CodigoValorNoCanonico)
	}
	if b.seccionClave != "" && !claveGobernadaValida(b.seccionClave) {
		return nuevoError("resultado.bloqueo.seccion", CodigoValorNoCanonico)
	}
	if b.claveGobernada != "" && !claveGobernadaValida(b.claveGobernada) {
		return nuevoError("resultado.bloqueo.clave", CodigoValorNoCanonico)
	}
	switch b.codigo {
	case BloqueoResultadoCatalogoIncompatible:
		if len(b.tramos) != 1 || len(b.reglas) != 0 || b.claveGobernada == "" ||
			b.grupoClave != "" || b.seccionClave != "" || b.tieneValorExacto {
			return nuevoError("resultado.bloqueo.catalogo", CodigoValorNoCanonico)
		}
	case BloqueoResultadoGruposDistintos, BloqueoResultadoCoincidenciaRechazada:
		if len(b.tramos) != 1 || len(b.reglas) < 2 || b.claveGobernada != "" ||
			b.grupoClave != "" || b.seccionClave != "" || b.tieneValorExacto {
			return nuevoError("resultado.bloqueo.coincidencia", CodigoValorNoCanonico)
		}
	case BloqueoResultadoSolape:
		if len(b.tramos) != 2 || len(b.reglas) != 0 || b.grupoClave == "" ||
			b.seccionClave != "" || b.claveGobernada != "" || b.tieneValorExacto {
			return nuevoError("resultado.bloqueo.solape", CodigoValorNoCanonico)
		}
	case BloqueoResultadoRedondeoNoExacto:
		if len(b.tramos) > 1 || len(b.reglas) != 1 || b.seccionClave == "" ||
			b.grupoClave != "" || b.claveGobernada != "" || !b.tieneValorExacto ||
			b.valorExacto.validar() != nil || b.valorExacto.esEntero() {
			return nuevoError("resultado.bloqueo.redondeo", CodigoValorNoCanonico)
		}
	default:
		return nuevoError("resultado.bloqueo.codigo", CodigoValorNoCanonico)
	}
	if !b.tieneValorExacto && b.valorExacto.canonico != "" {
		return nuevoError("resultado.bloqueo.valor_exacto", CodigoValorNoCanonico)
	}
	return nil
}

func validarEstadoResultadoV1(r ResultadoExperienciaV1) error {
	switch r.estado {
	case ResultadoExperienciaCompletado:
		if r.fase != FaseResultadoCompletado || len(r.bloqueos) != 0 || !r.tieneTotal ||
			!r.total.EsValido() || len(r.reglas) == 0 || len(r.secciones) == 0 ||
			len(r.intervalos) != len(r.seleccion.aplicaciones) ||
			len(r.aplicaciones) != len(r.seleccion.aplicaciones) {
			return nuevoError("resultado.completado", CodigoValorNoCanonico)
		}
		if !aplicacionesResultadoLigadasV1(r) || !totalResultadoV1Coincide(r) {
			return nuevoError("resultado.completado", CodigoContextoIncompatible)
		}
		if !puntuacionResultadoV1Coherente(r) {
			return nuevoError("resultado.completado.puntuacion", CodigoContextoIncompatible)
		}
	case ResultadoExperienciaBloqueado:
		if len(r.bloqueos) == 0 || r.tieneTotal || r.total.Micropuntos() != 0 ||
			len(r.reglas) != 0 || len(r.secciones) != 0 {
			return nuevoError("resultado.bloqueado", CodigoValorNoCanonico)
		}
		switch r.fase {
		case FaseResultadoSeleccion:
			if len(r.intervalos) != 0 || len(r.aplicaciones) != 0 ||
				!soloBloqueosSeleccionResultadoV1(r.bloqueos) {
				return nuevoError("resultado.bloqueado.seleccion", CodigoValorNoCanonico)
			}
		case FaseResultadoIntervalos:
			if len(r.seleccion.aplicaciones) == 0 ||
				len(r.intervalos) != len(r.seleccion.aplicaciones) ||
				len(r.aplicaciones) != 0 || !soloBloqueosSolapeResultadoV1(r.bloqueos) ||
				!intervalosResultadoLigadosV1(r) {
				return nuevoError("resultado.bloqueado.intervalos", CodigoValorNoCanonico)
			}
		case FaseResultadoPuntuacion:
			if len(r.intervalos) != len(r.seleccion.aplicaciones) ||
				len(r.aplicaciones) != len(r.seleccion.aplicaciones) ||
				!soloBloqueosRedondeoResultadoV1(r.bloqueos) ||
				!aplicacionesResultadoLigadasV1(r) {
				return nuevoError("resultado.bloqueado.puntuacion", CodigoValorNoCanonico)
			}
		default:
			return nuevoError("resultado.bloqueado.fase", CodigoValorNoCanonico)
		}
	default:
		return nuevoError("resultado.estado", CodigoValorNoCanonico)
	}
	return nil
}

func aplicacionesResultadoLigadasV1(r ResultadoExperienciaV1) bool {
	if !intervalosResultadoLigadosV1(r) || len(r.aplicaciones) != len(r.seleccion.aplicaciones) {
		return false
	}
	for indice, seleccion := range r.seleccion.aplicaciones {
		aplicacion := r.aplicaciones[indice]
		if compararReferenciasSeleccion(seleccion.tramo, aplicacion.tramo) != 0 ||
			seleccion.reglaClave != aplicacion.reglaClave {
			return false
		}
		if !r.intervalos[indice].tieneEfectivo && !aplicacionResultadoEsCeroV1(aplicacion) {
			return false
		}
	}
	return true
}

func intervalosResultadoLigadosV1(r ResultadoExperienciaV1) bool {
	if len(r.intervalos) != len(r.seleccion.aplicaciones) {
		return false
	}
	for indice, seleccion := range r.seleccion.aplicaciones {
		intervalo := r.intervalos[indice]
		if compararReferenciasSeleccion(seleccion.tramo, intervalo.tramo) != 0 ||
			seleccion.reglaClave != intervalo.reglaClave ||
			!intervaloResultadoNormalizadoV1(intervalo, r.vinculos.fechaCorte) {
			return false
		}
	}
	return true
}

func intervaloResultadoNormalizadoV1(
	registrado IntervaloAplicacionResultadoExperienciaV1,
	corte baremacion.FechaCivil,
) bool {
	esperado, presente, err := normalizarPeriodoEfectivo(
		registrado.periodo,
		corte,
		registrado.extremo,
	)
	if err != nil || presente != registrado.tieneEfectivo {
		return false
	}
	if presente {
		dias, err := esperado.NumeroDias()
		return err == nil && dias > 0 && uint64(dias) == registrado.dias &&
			registrado.razon == "" &&
			esperado.Desde() == registrado.efectivo.Desde() &&
			esperado.Hasta() == registrado.efectivo.Hasta()
	}
	corteExclusivo, err := corte.Siguiente()
	if err != nil {
		return false
	}
	comparacion, err := registrado.periodo.Desde().Comparar(corteExclusivo)
	if err != nil {
		return false
	}
	razonEsperada := RazonIntervaloVacio
	if comparacion >= 0 {
		razonEsperada = RazonPosteriorCorte
	}
	return registrado.razon == razonEsperada && registrado.dias == 0
}

func totalResultadoV1Coincide(r ResultadoExperienciaV1) bool {
	total := int64(0)
	for _, seccion := range r.secciones {
		puntos := seccion.puntosFinales.Micropuntos()
		if puntos < 0 || puntos > baremacion.MaximoMicropuntos-total {
			return false
		}
		total += puntos
	}
	return total == r.total.Micropuntos()
}

func puntuacionResultadoV1Coherente(r ResultadoExperienciaV1) bool {
	reglas := make(map[string]ResultadoReglaExperienciaV1, len(r.reglas))
	for _, regla := range r.reglas {
		reglas[regla.reglaClave] = regla
	}
	for _, seleccion := range r.seleccion.aplicaciones {
		regla, existe := reglas[seleccion.reglaClave]
		if !existe || regla.seccionClave != seleccion.seccionClave {
			return false
		}
	}
	brutosPorRegla := make(map[string][]exactoResultadoV1, len(r.reglas))
	redondeadosPorRegla := make(map[string][]exactoResultadoV1, len(r.reglas))
	unidadesExactasPorRegla := make(map[string][]exactoResultadoV1, len(r.reglas))
	unidadesAportadasPorRegla := make(map[string][]exactoResultadoV1, len(r.reglas))
	restosPorRegla := make(map[string][]exactoResultadoV1, len(r.reglas))
	fronteraPorRegla := make(map[string]FronteraRestosResultadoExperienciaV1, len(r.reglas))
	for _, aplicacion := range r.aplicaciones {
		regla, existe := reglas[aplicacion.reglaClave]
		if !existe || !productoExactoPorMicropuntosResultadoV1(
			aplicacion.unidades.aportadas,
			regla.coeficiente.Micropuntos(),
			aplicacion.puntuacion.bruto,
		) {
			return false
		}
		frontera, existe := fronteraPorRegla[regla.reglaClave]
		if existe && frontera != aplicacion.unidades.frontera {
			return false
		}
		fronteraPorRegla[regla.reglaClave] = aplicacion.unidades.frontera
		unidadesExactasPorRegla[regla.reglaClave] = append(
			unidadesExactasPorRegla[regla.reglaClave], aplicacion.unidades.exactas,
		)
		unidadesAportadasPorRegla[regla.reglaClave] = append(
			unidadesAportadasPorRegla[regla.reglaClave], aplicacion.unidades.aportadas,
		)
		restosPorRegla[regla.reglaClave] = append(
			restosPorRegla[regla.reglaClave], aplicacion.unidades.resto,
		)
		switch regla.redondeo.momento {
		case reglasbaremo.RedondearPorPeriodo:
			if !aplicacion.puntuacion.tieneRedondeado {
				return false
			}
			esperada, err := redondearExactoResultadoV1(
				aplicacion.puntuacion.bruto,
				regla.redondeo.modo,
			)
			if err != nil || esperada.canonico != aplicacion.puntuacion.redondeado.canonico {
				return false
			}
			brutosPorRegla[regla.reglaClave] = append(
				brutosPorRegla[regla.reglaClave], aplicacion.puntuacion.bruto,
			)
			redondeadosPorRegla[regla.reglaClave] = append(
				redondeadosPorRegla[regla.reglaClave], aplicacion.puntuacion.redondeado,
			)
		case reglasbaremo.RedondearPorRegla:
			if aplicacion.puntuacion.tieneRedondeado {
				return false
			}
		default:
			return false
		}
	}
	for _, regla := range r.reglas {
		exactas, err := sumarExactosResultadoV1(
			unidadesExactasPorRegla[regla.reglaClave],
		)
		if err != nil || exactas.canonico != regla.unidadesAgregadas.canonico {
			return false
		}
		if frontera, existe := fronteraPorRegla[regla.reglaClave]; existe {
			aportadas, err := sumarExactosResultadoV1(
				unidadesAportadasPorRegla[regla.reglaClave],
			)
			if err != nil {
				return false
			}
			switch frontera {
			case FronteraRestosResultadoExacta:
				if aportadas.canonico != exactas.canonico ||
					regla.unidadesTrasRestos.canonico != exactas.canonico ||
					regla.restoRegla.canonico != "0/1" {
					return false
				}
			case FronteraRestosResultadoPeriodo:
				restos, err := sumarExactosResultadoV1(
					restosPorRegla[regla.reglaClave],
				)
				if err != nil || aportadas.canonico != regla.unidadesTrasRestos.canonico ||
					restos.canonico != regla.restoRegla.canonico {
					return false
				}
			case FronteraRestosResultadoRegla:
				if aportadas.canonico != exactas.canonico {
					return false
				}
			default:
				return false
			}
		}
		if regla.redondeo.momento == reglasbaremo.RedondearPorPeriodo {
			bruto, err := sumarExactosResultadoV1(brutosPorRegla[regla.reglaClave])
			if err != nil || bruto.canonico != regla.bruto.canonico ||
				regla.redondeo.entrada.canonico != bruto.canonico {
				return false
			}
			redondeado, err := sumarExactosResultadoV1(redondeadosPorRegla[regla.reglaClave])
			if err != nil || redondeado.canonico != regla.redondeo.salida.canonico {
				return false
			}
		} else if !productoExactoPorMicropuntosResultadoV1(
			regla.topeUnidades.despues,
			regla.coeficiente.Micropuntos(),
			regla.bruto,
		) {
			return false
		}
	}
	return subtotalesResultadoV1Coherentes(r)
}

// Los periodos seleccionados que quedan vacios tras corte/extremo se conservan
// para explicar el cero. Su aplicacion asociada solo puede contener ceros.
func aplicacionResultadoEsCeroV1(a AplicacionCalculadaResultadoExperienciaV1) bool {
	const cero = "0/1"
	if a.unidades.exactas.canonico != cero || a.unidades.aportadas.canonico != cero ||
		a.unidades.resto.canonico != cero || a.puntuacion.bruto.canonico != cero {
		return false
	}
	return !a.puntuacion.tieneRedondeado || a.puntuacion.redondeado.canonico == cero
}

func subtotalesResultadoV1Coherentes(r ResultadoExperienciaV1) bool {
	porSeccion := make(map[string][]exactoResultadoV1, len(r.secciones))
	for _, regla := range r.reglas {
		porSeccion[regla.seccionClave] = append(
			porSeccion[regla.seccionClave], regla.puntosFinales,
		)
	}
	for _, seccion := range r.secciones {
		valores, existe := porSeccion[seccion.seccionClave]
		if !existe {
			return false
		}
		suma, err := sumarExactosResultadoV1(valores)
		if err != nil || suma.canonico != seccion.antesTope.canonico {
			return false
		}
		delete(porSeccion, seccion.seccionClave)
	}
	return len(porSeccion) == 0
}

func soloBloqueosSeleccionResultadoV1(bloqueos []BloqueoResultadoExperienciaV1) bool {
	for _, bloqueo := range bloqueos {
		switch bloqueo.codigo {
		case BloqueoResultadoCatalogoIncompatible,
			BloqueoResultadoGruposDistintos,
			BloqueoResultadoCoincidenciaRechazada:
		default:
			return false
		}
	}
	return true
}

func soloBloqueosSolapeResultadoV1(bloqueos []BloqueoResultadoExperienciaV1) bool {
	for _, bloqueo := range bloqueos {
		if bloqueo.codigo != BloqueoResultadoSolape {
			return false
		}
	}
	return true
}

func soloBloqueosRedondeoResultadoV1(bloqueos []BloqueoResultadoExperienciaV1) bool {
	for _, bloqueo := range bloqueos {
		if bloqueo.codigo != BloqueoResultadoRedondeoNoExacto {
			return false
		}
	}
	return true
}

func ordenResultadoV1Valido(r ResultadoExperienciaV1) bool {
	for indice := 1; indice < len(r.reglas); indice++ {
		anterior, actual := r.reglas[indice-1], r.reglas[indice]
		if anterior.seccionClave > actual.seccionClave ||
			(anterior.seccionClave == actual.seccionClave && anterior.reglaClave >= actual.reglaClave) {
			return false
		}
	}
	for indice := 1; indice < len(r.secciones); indice++ {
		if r.secciones[indice-1].seccionClave >= r.secciones[indice].seccionClave {
			return false
		}
	}
	for indice := 1; indice < len(r.bloqueos); indice++ {
		if compararBloqueosResultadoV1(r.bloqueos[indice-1], r.bloqueos[indice]) >= 0 {
			return false
		}
	}
	return true
}

func compararBloqueosResultadoV1(
	izquierda BloqueoResultadoExperienciaV1,
	derecha BloqueoResultadoExperienciaV1,
) int {
	if izquierda.codigo < derecha.codigo {
		return -1
	}
	if izquierda.codigo > derecha.codigo {
		return 1
	}
	if len(izquierda.tramos) > 0 && len(derecha.tramos) > 0 {
		if comparacion := compararReferenciasSeleccion(izquierda.tramos[0], derecha.tramos[0]); comparacion != 0 {
			return comparacion
		}
	} else if len(izquierda.tramos) != len(derecha.tramos) {
		if len(izquierda.tramos) < len(derecha.tramos) {
			return -1
		}
		return 1
	}
	if izquierda.grupoClave < derecha.grupoClave {
		return -1
	}
	if izquierda.grupoClave > derecha.grupoClave {
		return 1
	}
	if izquierda.claveGobernada < derecha.claveGobernada {
		return -1
	}
	if izquierda.claveGobernada > derecha.claveGobernada {
		return 1
	}
	if len(izquierda.reglas) > 0 && len(derecha.reglas) > 0 {
		if izquierda.reglas[0] < derecha.reglas[0] {
			return -1
		}
		if izquierda.reglas[0] > derecha.reglas[0] {
			return 1
		}
	}
	return 0
}

func compararAplicacionesSeleccionResultadoV1(
	izquierda AplicacionSeleccionResultadoExperienciaV1,
	derecha AplicacionSeleccionResultadoExperienciaV1,
) int {
	if comparacion := compararReferenciasSeleccion(izquierda.tramo, derecha.tramo); comparacion != 0 {
		return comparacion
	}
	if izquierda.grupoClave < derecha.grupoClave {
		return -1
	}
	if izquierda.grupoClave > derecha.grupoClave {
		return 1
	}
	if izquierda.reglaClave < derecha.reglaClave {
		return -1
	}
	if izquierda.reglaClave > derecha.reglaClave {
		return 1
	}
	return 0
}

func compararDescartesSeleccionResultadoV1(
	izquierda DescarteSeleccionResultadoExperienciaV1,
	derecha DescarteSeleccionResultadoExperienciaV1,
) int {
	if comparacion := compararReferenciasSeleccion(izquierda.tramo, derecha.tramo); comparacion != 0 {
		return comparacion
	}
	if izquierda.grupoClave < derecha.grupoClave {
		return -1
	}
	if izquierda.grupoClave > derecha.grupoClave {
		return 1
	}
	if izquierda.reglaClave < derecha.reglaClave {
		return -1
	}
	if izquierda.reglaClave > derecha.reglaClave {
		return 1
	}
	return 0
}

func razonAplicacionValida(razon CodigoRazonResultadoExperienciaV1) bool {
	return razon == RazonCoincidenciaUnica || razon == RazonPrioridad || razon == RazonAcumulacion
}
