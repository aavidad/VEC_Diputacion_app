package calculoexperiencia

import "vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"

func ordenarSeleccion(resultado *seleccionExperiencia) {
	if resultado == nil {
		return
	}
	ordenarValoresSeleccion(resultado.aplicaciones, func(izquierda, derecha aplicacionSeleccion) bool {
		if comparacion := compararReferenciasSeleccion(izquierda.tramo, derecha.tramo); comparacion != 0 {
			return comparacion < 0
		}
		if izquierda.grupoClave != derecha.grupoClave {
			return izquierda.grupoClave < derecha.grupoClave
		}
		return izquierda.reglaClave < derecha.reglaClave
	})
	ordenarValoresSeleccion(resultado.descartes, func(izquierda, derecha descarteSeleccion) bool {
		if comparacion := compararReferenciasSeleccion(izquierda.tramo, derecha.tramo); comparacion != 0 {
			return comparacion < 0
		}
		if izquierda.grupoClave != derecha.grupoClave {
			return izquierda.grupoClave < derecha.grupoClave
		}
		return izquierda.reglaClave < derecha.reglaClave
	})
	ordenarValoresSeleccion(resultado.noCoincidencias, func(izquierda, derecha noCoincidenciaSeleccion) bool {
		return compararReferenciasSeleccion(izquierda.tramo, derecha.tramo) < 0
	})
	for indice := range resultado.bloqueos {
		ordenarCadenasSeleccion(resultado.bloqueos[indice].reglas)
	}
	ordenarBloqueosSeleccion(resultado.bloqueos)
}

type elementoOrdenSeleccion interface {
	aplicacionSeleccion | descarteSeleccion | noCoincidenciaSeleccion |
		bloqueoSeleccion | claveGobernadaSeleccion | anclaSeleccion |
		grupoSeleccionIndexado | reglasbaremo.CriterioExperiencia | string | int
}

// ordenarValoresSeleccion implementa una mezcla estable y acotada. Evita que
// esta fase dependa del orden de mapas o entradas sin ampliar las importaciones
// permitidas por la barrera arquitectonica del dominio.
func ordenarValoresSeleccion[T elementoOrdenSeleccion](valores []T, menor func(T, T) bool) {
	if len(valores) < 2 {
		return
	}
	auxiliar := make([]T, len(valores))
	ordenarRangoSeleccion(valores, auxiliar, 0, len(valores), menor)
}

func ordenarRangoSeleccion[T elementoOrdenSeleccion](
	valores []T,
	auxiliar []T,
	inicio int,
	fin int,
	menor func(T, T) bool,
) {
	if fin-inicio < 2 {
		return
	}
	medio := inicio + (fin-inicio)/2
	ordenarRangoSeleccion(valores, auxiliar, inicio, medio, menor)
	ordenarRangoSeleccion(valores, auxiliar, medio, fin, menor)

	izquierda, derecha, destino := inicio, medio, inicio
	for izquierda < medio && derecha < fin {
		// Elegir izquierda en igualdad mantiene la estabilidad.
		if !menor(valores[derecha], valores[izquierda]) {
			auxiliar[destino] = valores[izquierda]
			izquierda++
		} else {
			auxiliar[destino] = valores[derecha]
			derecha++
		}
		destino++
	}
	for izquierda < medio {
		auxiliar[destino] = valores[izquierda]
		izquierda++
		destino++
	}
	for derecha < fin {
		auxiliar[destino] = valores[derecha]
		derecha++
		destino++
	}
	copy(valores[inicio:fin], auxiliar[inicio:fin])
}

func ordenarCriteriosSeleccion(criterios []reglasbaremo.CriterioExperiencia) {
	ordenarValoresSeleccion(criterios, func(izquierda, derecha reglasbaremo.CriterioExperiencia) bool {
		return izquierda.Clave() < derecha.Clave()
	})
}

func ordenarCadenasSeleccion(valores []string) {
	ordenarValoresSeleccion(valores, func(izquierda, derecha string) bool {
		return izquierda < derecha
	})
}

func ordenarEnterosSeleccion(valores []int) {
	ordenarValoresSeleccion(valores, func(izquierda, derecha int) bool {
		return izquierda < derecha
	})
}

func ordenarClavesGobernadasSeleccion(valores []claveGobernadaSeleccion) {
	ordenarValoresSeleccion(valores, func(izquierda, derecha claveGobernadaSeleccion) bool {
		return izquierda.clave < derecha.clave
	})
}

func ordenarAnclasSeleccion(valores []anclaSeleccion) {
	ordenarValoresSeleccion(valores, func(izquierda, derecha anclaSeleccion) bool {
		if izquierda.clave != derecha.clave {
			return izquierda.clave < derecha.clave
		}
		return izquierda.valor < derecha.valor
	})
}

func ordenarGruposSeleccion(valores []grupoSeleccionIndexado) {
	ordenarValoresSeleccion(valores, func(izquierda, derecha grupoSeleccionIndexado) bool {
		return izquierda.clave < derecha.clave
	})
}

func ordenarReglasPorPrioridadSeleccion(
	posiciones []int,
	reglas []reglaSeleccionIndexada,
) {
	ordenarValoresSeleccion(posiciones, func(izquierda, derecha int) bool {
		reglaIzquierda := reglas[izquierda]
		reglaDerecha := reglas[derecha]
		if reglaIzquierda.prioridad != reglaDerecha.prioridad {
			return reglaIzquierda.prioridad < reglaDerecha.prioridad
		}
		return reglaIzquierda.clave < reglaDerecha.clave
	})
}

func ordenarBloqueosSeleccion(valores []bloqueoSeleccion) {
	ordenarValoresSeleccion(valores, func(izquierda, derecha bloqueoSeleccion) bool {
		if comparacion := compararReferenciasSeleccion(izquierda.tramo, derecha.tramo); comparacion != 0 {
			return comparacion < 0
		}
		if izquierda.codigo != derecha.codigo {
			return izquierda.codigo < derecha.codigo
		}
		return izquierda.claveGobernada < derecha.claveGobernada
	})
}

func compararReferenciasSeleccion(
	izquierda reglasbaremo.ReferenciaVersionada,
	derecha reglasbaremo.ReferenciaVersionada,
) int {
	if izquierda.Referencia() < derecha.Referencia() {
		return -1
	}
	if izquierda.Referencia() > derecha.Referencia() {
		return 1
	}
	if izquierda.Version() < derecha.Version() {
		return -1
	}
	if izquierda.Version() > derecha.Version() {
		return 1
	}
	if izquierda.HuellaSHA256() < derecha.HuellaSHA256() {
		return -1
	}
	if izquierda.HuellaSHA256() > derecha.HuellaSHA256() {
		return 1
	}
	return 0
}
