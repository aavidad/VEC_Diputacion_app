package calculoexperiencia

import (
	"vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
	"vec-diputacion-granada/internal/shared/baremacion"
)

type claveUnionTemporal struct {
	grupoClave      string
	tramoReferencia string
}

type unionTemporalTramo struct {
	grupoClave string
	tramo      reglasbaremo.ReferenciaVersionada
	intervalo  baremacion.IntervaloCivil
}

func detectarSolapesTemporales(
	aplicaciones []aplicacionTemporal,
	grupos map[string]reglasbaremo.GrupoConcurrenciaExperiencia,
	presupuesto *presupuestoAplicacionesTemporales,
	resultado *resultadoAplicacionesTemporales,
) error {
	porTramo := make(map[claveUnionTemporal]unionTemporalTramo)
	for _, aplicacion := range aplicaciones {
		grupoClave := aplicacion.regla.GrupoConcurrenciaClave()
		tramo := aplicacion.tramo.Referencia()
		clave := claveUnionTemporal{
			grupoClave:      grupoClave,
			tramoReferencia: tramo.Referencia(),
		}
		acumulada, existe := porTramo[clave]
		if !existe {
			porTramo[clave] = unionTemporalTramo{
				grupoClave: grupoClave,
				tramo:      tramo,
				intervalo:  aplicacion.intervalo,
			}
			continue
		}
		intervalo, err := unirIntervalosTemporales(acumulada.intervalo, aplicacion.intervalo)
		if err != nil {
			return err
		}
		acumulada.intervalo = intervalo
		porTramo[clave] = acumulada
	}

	uniones := make([]unionTemporalTramo, 0, len(porTramo))
	for _, union := range porTramo {
		uniones = append(uniones, union)
	}
	ordenarUnionesTemporales(uniones)

	grupoActual := ""
	var maximoFin baremacion.FechaCivil
	var tramoMaximoFin reglasbaremo.ReferenciaVersionada
	bloqueoEmitido := false
	for _, union := range uniones {
		// Cada union aporta exactamente los eventos de apertura y cierre. El
		// presupuesto se consume incluso cuando el grupo ya tiene un bloqueo,
		// para que el coste no dependa de encontrar pronto un conflicto.
		if err := presupuesto.consumirEvento(); err != nil {
			return err
		}
		if err := presupuesto.consumirEvento(); err != nil {
			return err
		}
		grupo, existe := grupos[union.grupoClave]
		if !existe || grupo.Solape().Modo() != reglasbaremo.SolapeRechazar {
			return errorSeleccionTemporalInvalida()
		}

		if union.grupoClave != grupoActual {
			grupoActual = union.grupoClave
			maximoFin = union.intervalo.Hasta()
			tramoMaximoFin = union.tramo
			bloqueoEmitido = false
			continue
		}

		inicioAntesDeMaximo, err := union.intervalo.Desde().Comparar(maximoFin)
		if err != nil {
			return errorSeleccionTemporalInvalida()
		}
		if inicioAntesDeMaximo < 0 && !bloqueoEmitido {
			primero, segundo := ordenarParTramosTemporal(tramoMaximoFin, union.tramo)
			resultado.bloqueos = append(resultado.bloqueos, bloqueoAplicacionesTemporales{
				codigo:       bloqueoTemporalSolapeRechazado,
				grupoClave:   union.grupoClave,
				tramoPrimero: primero,
				tramoSegundo: segundo,
			})
			bloqueoEmitido = true
		}

		comparacionFin, err := maximoFin.Comparar(union.intervalo.Hasta())
		if err != nil {
			return errorSeleccionTemporalInvalida()
		}
		if comparacionFin < 0 ||
			(comparacionFin == 0 && compararReferenciasTemporales(union.tramo, tramoMaximoFin) < 0) {
			maximoFin = union.intervalo.Hasta()
			tramoMaximoFin = union.tramo
		}
	}
	return nil
}

func unirIntervalosTemporales(
	izquierdo baremacion.IntervaloCivil,
	derecho baremacion.IntervaloCivil,
) (baremacion.IntervaloCivil, error) {
	if !izquierdo.EsValido() || !derecho.EsValido() {
		return baremacion.IntervaloCivil{}, errorSeleccionTemporalInvalida()
	}
	desde := izquierdo.Desde()
	if comparacion, _ := derecho.Desde().Comparar(desde); comparacion < 0 {
		desde = derecho.Desde()
	}
	hasta := izquierdo.Hasta()
	if comparacion, _ := derecho.Hasta().Comparar(hasta); comparacion > 0 {
		hasta = derecho.Hasta()
	}
	intervalo, err := baremacion.NuevoIntervaloCivil(desde, hasta)
	if err != nil {
		return baremacion.IntervaloCivil{}, errorSeleccionTemporalInvalida()
	}
	return intervalo, nil
}

func ordenarParTramosTemporal(
	izquierdo reglasbaremo.ReferenciaVersionada,
	derecho reglasbaremo.ReferenciaVersionada,
) (reglasbaremo.ReferenciaVersionada, reglasbaremo.ReferenciaVersionada) {
	if compararReferenciasTemporales(izquierdo, derecho) <= 0 {
		return izquierdo, derecho
	}
	return derecho, izquierdo
}

func compararReferenciasTemporales(
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

func ordenarResultadoAplicacionesTemporales(resultado *resultadoAplicacionesTemporales) {
	if resultado == nil {
		return
	}
	ordenarValoresTemporales(resultado.aplicaciones, func(a, b aplicacionTemporal) bool {
		grupoA, grupoB := a.regla.GrupoConcurrenciaClave(), b.regla.GrupoConcurrenciaClave()
		if grupoA != grupoB {
			return grupoA < grupoB
		}
		if comparacion := compararReferenciasTemporales(
			a.tramo.Referencia(), b.tramo.Referencia(),
		); comparacion != 0 {
			return comparacion < 0
		}
		return a.regla.Clave() < b.regla.Clave()
	})
	ordenarValoresTemporales(resultado.exclusiones, func(a, b exclusionAplicacionTemporal) bool {
		if a.grupoClave != b.grupoClave {
			return a.grupoClave < b.grupoClave
		}
		if comparacion := compararReferenciasTemporales(a.tramo, b.tramo); comparacion != 0 {
			return comparacion < 0
		}
		return a.reglaClave < b.reglaClave
	})
	ordenarValoresTemporales(resultado.bloqueos, func(a, b bloqueoAplicacionesTemporales) bool {
		if a.grupoClave != b.grupoClave {
			return a.grupoClave < b.grupoClave
		}
		if comparacion := compararReferenciasTemporales(
			a.tramoPrimero, b.tramoPrimero,
		); comparacion != 0 {
			return comparacion < 0
		}
		return compararReferenciasTemporales(a.tramoSegundo, b.tramoSegundo) < 0
	})
}

func ordenarUnionesTemporales(uniones []unionTemporalTramo) {
	ordenarValoresTemporales(uniones, func(a, b unionTemporalTramo) bool {
		if a.grupoClave != b.grupoClave {
			return a.grupoClave < b.grupoClave
		}
		comparacionInicio, _ := a.intervalo.Desde().Comparar(b.intervalo.Desde())
		if comparacionInicio != 0 {
			return comparacionInicio < 0
		}
		comparacionFin, _ := a.intervalo.Hasta().Comparar(b.intervalo.Hasta())
		if comparacionFin != 0 {
			return comparacionFin < 0
		}
		return compararReferenciasTemporales(a.tramo, b.tramo) < 0
	})
}

type elementoOrdenTemporal interface {
	aplicacionTemporal | exclusionAplicacionTemporal |
		bloqueoAplicacionesTemporales | unionTemporalTramo
}

func ordenarValoresTemporales[T elementoOrdenTemporal](valores []T, menor func(T, T) bool) {
	if len(valores) < 2 {
		return
	}
	auxiliar := make([]T, len(valores))
	ordenarRangoTemporal(valores, auxiliar, 0, len(valores), menor)
}

func ordenarRangoTemporal[T elementoOrdenTemporal](
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
	ordenarRangoTemporal(valores, auxiliar, inicio, medio, menor)
	ordenarRangoTemporal(valores, auxiliar, medio, fin, menor)

	izquierda, derecha, destino := inicio, medio, inicio
	for izquierda < medio && derecha < fin {
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
