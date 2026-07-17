package calculoexperiencia

import "vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"

func seleccionesPuntuacionIguales(izquierda, derecha seleccionExperiencia) bool {
	if izquierda.evaluaciones != derecha.evaluaciones ||
		len(izquierda.aplicaciones) != len(derecha.aplicaciones) ||
		len(izquierda.descartes) != len(derecha.descartes) ||
		len(izquierda.noCoincidencias) != len(derecha.noCoincidencias) ||
		len(izquierda.bloqueos) != len(derecha.bloqueos) {
		return false
	}
	for indice, uno := range izquierda.aplicaciones {
		otro := derecha.aplicaciones[indice]
		if !referenciasPlanIguales(uno.tramo, otro.tramo) ||
			uno.reglaClave != otro.reglaClave || uno.grupoClave != otro.grupoClave ||
			uno.seccionClave != otro.seccionClave || uno.prioridad != otro.prioridad ||
			uno.razon != otro.razon {
			return false
		}
	}
	for indice, uno := range izquierda.descartes {
		otro := derecha.descartes[indice]
		if !referenciasPlanIguales(uno.tramo, otro.tramo) ||
			uno.reglaClave != otro.reglaClave || uno.grupoClave != otro.grupoClave ||
			uno.reglaSeleccionada != otro.reglaSeleccionada || uno.razon != otro.razon {
			return false
		}
	}
	for indice, uno := range izquierda.noCoincidencias {
		otro := derecha.noCoincidencias[indice]
		if !referenciasPlanIguales(uno.tramo, otro.tramo) || uno.razon != otro.razon {
			return false
		}
	}
	for indice, uno := range izquierda.bloqueos {
		otro := derecha.bloqueos[indice]
		if uno.codigo != otro.codigo || !referenciasPlanIguales(uno.tramo, otro.tramo) ||
			uno.claveGobernada != otro.claveGobernada ||
			!cadenasPuntuacionIguales(uno.reglas, otro.reglas) {
			return false
		}
	}
	return true
}

func tramosPuntuacionIguales(izquierda, derecha TramoExperiencia) bool {
	if !referenciasPlanIguales(izquierda.referencia, derecha.referencia) ||
		izquierda.servicioRef != derecha.servicioRef ||
		!periodosPuntuacionIguales(izquierda.periodo, derecha.periodo) ||
		izquierda.jornada.Numerador() != derecha.jornada.Numerador() ||
		izquierda.jornada.Denominador() != derecha.jornada.Denominador() ||
		izquierda.atestacion.modo != derecha.atestacion.modo ||
		!referenciasPlanIguales(izquierda.atestacion.referencia, derecha.atestacion.referencia) ||
		len(izquierda.atributos) != len(derecha.atributos) {
		return false
	}
	for indice := range izquierda.atributos {
		uno, otro := izquierda.atributos[indice], derecha.atributos[indice]
		if uno.clave != otro.clave || uno.valor != otro.valor ||
			!referenciasPlanIguales(uno.catalogo, otro.catalogo) {
			return false
		}
	}
	return true
}

func periodosPuntuacionIguales(izquierda, derecha PeriodoServicio) bool {
	return izquierda.modo == derecha.modo && izquierda.desde == derecha.desde &&
		izquierda.finInformado == derecha.finInformado
}

func reglasPuntuacionIguales(
	izquierda reglasbaremo.ReglaExperiencia,
	derecha reglasbaremo.ReglaExperiencia,
) bool {
	if izquierda.Clave() != derecha.Clave() ||
		!referenciasPlanIguales(izquierda.Definicion(), derecha.Definicion()) ||
		izquierda.SeccionClave() != derecha.SeccionClave() ||
		izquierda.Orden() != derecha.Orden() ||
		izquierda.GrupoConcurrenciaClave() != derecha.GrupoConcurrenciaClave() ||
		izquierda.PrioridadConcurrencia() != derecha.PrioridadConcurrencia() ||
		!unidadesTemporalesPuntuacionIguales(izquierda, derecha) ||
		!jornadasPuntuacionIguales(izquierda, derecha) ||
		izquierda.Restos().Modo() != derecha.Restos().Modo() ||
		izquierda.Redondeo().Momento() != derecha.Redondeo().Momento() ||
		izquierda.Redondeo().Modo() != derecha.Redondeo().Modo() ||
		izquierda.PuntosPorUnidad().Micropuntos() != derecha.PuntosPorUnidad().Micropuntos() ||
		!limitesPuntuacionIguales(izquierda, derecha) {
		return false
	}
	criteriosIzquierda, criteriosDerecha := izquierda.Criterios(), derecha.Criterios()
	if len(criteriosIzquierda) != len(criteriosDerecha) {
		return false
	}
	for indice := range criteriosIzquierda {
		uno, otro := criteriosIzquierda[indice], criteriosDerecha[indice]
		if uno.Clave() != otro.Clave() ||
			!referenciasPlanIguales(uno.Catalogo(), otro.Catalogo()) ||
			!cadenasPuntuacionIguales(uno.Valores(), otro.Valores()) {
			return false
		}
	}
	return true
}

func unidadesTemporalesPuntuacionIguales(
	izquierda reglasbaremo.ReglaExperiencia,
	derecha reglasbaremo.ReglaExperiencia,
) bool {
	uno, otro := izquierda.UnidadTemporal(), derecha.UnidadTemporal()
	conversionUno, conversionOtro := uno.UnidadesBasePorUnidad(), otro.UnidadesBasePorUnidad()
	return uno.UnidadBase() == otro.UnidadBase() &&
		uno.UnidadPuntuable() == otro.UnidadPuntuable() &&
		uno.ExtremoFinal() == otro.ExtremoFinal() &&
		conversionUno.Numerador() == conversionOtro.Numerador() &&
		conversionUno.Denominador() == conversionOtro.Denominador()
}

func jornadasPuntuacionIguales(
	izquierda reglasbaremo.ReglaExperiencia,
	derecha reglasbaremo.ReglaExperiencia,
) bool {
	uno, otro := izquierda.Jornada(), derecha.Jornada()
	if uno.Modo() != otro.Modo() {
		return false
	}
	umbralUno, existeUno := uno.Umbral()
	umbralOtro, existeOtro := otro.Umbral()
	return existeUno == existeOtro && (!existeUno ||
		(umbralUno.Numerador() == umbralOtro.Numerador() &&
			umbralUno.Denominador() == umbralOtro.Denominador()))
}

func limitesPuntuacionIguales(
	izquierda reglasbaremo.ReglaExperiencia,
	derecha reglasbaremo.ReglaExperiencia,
) bool {
	unidadesUno, tieneUnidadesUno := izquierda.MaximoUnidades().Valor()
	unidadesOtro, tieneUnidadesOtro := derecha.MaximoUnidades().Valor()
	if tieneUnidadesUno != tieneUnidadesOtro || (tieneUnidadesUno &&
		(unidadesUno.Numerador() != unidadesOtro.Numerador() ||
			unidadesUno.Denominador() != unidadesOtro.Denominador())) {
		return false
	}
	puntosUno, tienePuntosUno := izquierda.MaximoPuntos().Valor()
	puntosOtro, tienePuntosOtro := derecha.MaximoPuntos().Valor()
	return tienePuntosUno == tienePuntosOtro &&
		(!tienePuntosUno || puntosUno.Micropuntos() == puntosOtro.Micropuntos())
}

func cadenasPuntuacionIguales(izquierda, derecha []string) bool {
	if len(izquierda) != len(derecha) {
		return false
	}
	for indice := range izquierda {
		if izquierda[indice] != derecha[indice] {
			return false
		}
	}
	return true
}
