package calculoexperiencia

import (
	"vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
	"vec-diputacion-granada/internal/shared/baremacion"
)

func reconstruirIntervaloResultadoV1(
	m materialIntervaloAplicacionV1,
) (IntervaloAplicacionResultadoExperienciaV1, error) {
	tramo, err := reconstruirReferenciaResultadoV1(m.Tramo)
	if err != nil {
		return IntervaloAplicacionResultadoExperienciaV1{}, err
	}
	periodo, err := reconstruirPeriodo(m.Periodo)
	if err != nil {
		return IntervaloAplicacionResultadoExperienciaV1{}, err
	}
	resultado := IntervaloAplicacionResultadoExperienciaV1{
		tramo: tramo, reglaClave: m.Regla, periodo: periodo, extremo: m.Extremo, razon: m.Razon,
	}
	if m.Efectivo != nil {
		intervalo, err := baremacion.NuevoIntervaloCivil(
			m.Efectivo.Desde,
			m.Efectivo.HastaExclusivo,
		)
		if err != nil {
			return IntervaloAplicacionResultadoExperienciaV1{},
				nuevoError("resultado.intervalo.efectivo", CodigoValorNoCanonico)
		}
		resultado.efectivo = intervalo
		resultado.tieneEfectivo = true
		resultado.dias = m.Efectivo.Dias
	}
	if err := validarIntervaloResultadoV1(resultado); err != nil {
		return IntervaloAplicacionResultadoExperienciaV1{}, err
	}
	return resultado, nil
}

func reconstruirAplicacionResultadoV1(
	m materialAplicacionCalculadaV1,
) (AplicacionCalculadaResultadoExperienciaV1, error) {
	tramo, err := reconstruirReferenciaResultadoV1(m.Tramo)
	if err != nil {
		return AplicacionCalculadaResultadoExperienciaV1{}, err
	}
	factor, err := nuevoExactoResultadoV1(m.Jornada.Factor)
	if err != nil {
		return AplicacionCalculadaResultadoExperienciaV1{}, err
	}
	exactas, err := nuevoExactoResultadoV1(m.Unidades.Exactas)
	if err != nil {
		return AplicacionCalculadaResultadoExperienciaV1{}, err
	}
	aportadas, err := nuevoExactoResultadoV1(m.Unidades.Aportadas)
	if err != nil {
		return AplicacionCalculadaResultadoExperienciaV1{}, err
	}
	resto, err := nuevoExactoResultadoV1(m.Unidades.Resto)
	if err != nil {
		return AplicacionCalculadaResultadoExperienciaV1{}, err
	}
	bruto, err := nuevoExactoResultadoV1(m.Puntuacion.Bruto)
	if err != nil {
		return AplicacionCalculadaResultadoExperienciaV1{}, err
	}
	resultado := AplicacionCalculadaResultadoExperienciaV1{
		tramo:      tramo,
		reglaClave: m.Regla,
		jornada: JornadaResultadoExperienciaV1{
			origen: m.Jornada.Origen, modo: m.Jornada.Modo, factor: factor,
			atestacionPresente: m.Jornada.AtestacionPresente,
			atestacionUsada:    m.Jornada.AtestacionUsada, razon: m.Jornada.Razon,
		},
		unidades: UnidadesAplicacionResultadoExperienciaV1{
			exactas: exactas, aportadas: aportadas, resto: resto, frontera: m.Unidades.Frontera,
		},
		puntuacion: PuntuacionPeriodoResultadoExperienciaV1{bruto: bruto},
	}
	if m.Puntuacion.Redondeado != nil {
		redondeado, err := nuevoExactoResultadoV1(*m.Puntuacion.Redondeado)
		if err != nil {
			return AplicacionCalculadaResultadoExperienciaV1{}, err
		}
		resultado.puntuacion.redondeado = redondeado
		resultado.puntuacion.tieneRedondeado = true
	}
	if err := validarAplicacionCalculadaResultadoV1(resultado); err != nil {
		return AplicacionCalculadaResultadoExperienciaV1{}, err
	}
	return resultado, nil
}

func reconstruirTopeResultadoV1(m materialTopeV1) (TopeResultadoExperienciaV1, error) {
	antes, err := nuevoExactoResultadoV1(m.Antes)
	if err != nil {
		return TopeResultadoExperienciaV1{}, err
	}
	despues, err := nuevoExactoResultadoV1(m.Despues)
	if err != nil {
		return TopeResultadoExperienciaV1{}, err
	}
	resultado := TopeResultadoExperienciaV1{
		antes: antes, despues: despues, aplicado: m.Aplicado,
	}
	if m.Limite != nil {
		limite, err := nuevoExactoResultadoV1(*m.Limite)
		if err != nil {
			return TopeResultadoExperienciaV1{}, err
		}
		resultado.limitado = true
		resultado.limite = limite
	}
	if err := validarTopeResultadoV1(resultado); err != nil {
		return TopeResultadoExperienciaV1{}, err
	}
	return resultado, nil
}

func reconstruirReglaResultadoV1(
	m materialReglaResultadoV1,
) (ResultadoReglaExperienciaV1, error) {
	unidadesAgregadas, err := nuevoExactoResultadoV1(m.UnidadesAgregadas)
	if err != nil {
		return ResultadoReglaExperienciaV1{}, err
	}
	unidadesTrasRestos, err := nuevoExactoResultadoV1(m.UnidadesTrasRestos)
	if err != nil {
		return ResultadoReglaExperienciaV1{}, err
	}
	restoRegla, err := nuevoExactoResultadoV1(m.RestoRegla)
	if err != nil {
		return ResultadoReglaExperienciaV1{}, err
	}
	topeUnidades, err := reconstruirTopeResultadoV1(m.TopeUnidades)
	if err != nil {
		return ResultadoReglaExperienciaV1{}, err
	}
	bruto, err := nuevoExactoResultadoV1(m.Bruto)
	if err != nil {
		return ResultadoReglaExperienciaV1{}, err
	}
	entradaRedondeo, err := nuevoExactoResultadoV1(m.Redondeo.Entrada)
	if err != nil {
		return ResultadoReglaExperienciaV1{}, err
	}
	salidaRedondeo, err := nuevoExactoResultadoV1(m.Redondeo.Salida)
	if err != nil {
		return ResultadoReglaExperienciaV1{}, err
	}
	topePuntos, err := reconstruirTopeResultadoV1(m.TopePuntos)
	if err != nil {
		return ResultadoReglaExperienciaV1{}, err
	}
	puntosFinales, err := nuevoExactoResultadoV1(m.PuntosFinales)
	if err != nil {
		return ResultadoReglaExperienciaV1{}, err
	}
	resultado := ResultadoReglaExperienciaV1{
		seccionClave: m.Seccion, reglaClave: m.Regla,
		unidadesAgregadas: unidadesAgregadas, unidadesTrasRestos: unidadesTrasRestos,
		restoRegla: restoRegla, topeUnidades: topeUnidades, coeficiente: m.Coeficiente,
		bruto: bruto,
		redondeo: RedondeoResultadoExperienciaV1{
			momento: m.Redondeo.Momento, modo: m.Redondeo.Modo,
			entrada: entradaRedondeo, salida: salidaRedondeo,
		},
		topePuntos: topePuntos, puntosFinales: puntosFinales,
	}
	if err := validarReglaResultadoV1(resultado); err != nil {
		return ResultadoReglaExperienciaV1{}, err
	}
	return resultado, nil
}

func reconstruirSeccionResultadoV1(
	m materialSeccionResultadoV1,
) (SubtotalSeccionResultadoExperienciaV1, error) {
	antes, err := nuevoExactoResultadoV1(m.AntesTope)
	if err != nil {
		return SubtotalSeccionResultadoExperienciaV1{}, err
	}
	tope, err := reconstruirTopeResultadoV1(m.Tope)
	if err != nil {
		return SubtotalSeccionResultadoExperienciaV1{}, err
	}
	resultado := SubtotalSeccionResultadoExperienciaV1{
		seccionClave: m.Seccion, antesTope: antes, tope: tope, puntosFinales: m.PuntosFinales,
	}
	if err := validarSeccionResultadoV1(resultado); err != nil {
		return SubtotalSeccionResultadoExperienciaV1{}, err
	}
	return resultado, nil
}

func reconstruirBloqueoResultadoV1(
	m materialBloqueoResultadoV1,
) (BloqueoResultadoExperienciaV1, error) {
	resultado := BloqueoResultadoExperienciaV1{
		codigo: m.Codigo, tramos: make([]reglasbaremo.ReferenciaVersionada, len(m.Tramos)),
		reglas: append([]string(nil), m.Reglas...), grupoClave: m.Grupo,
		seccionClave: m.Seccion, claveGobernada: m.ClaveGobernada,
	}
	for indice, origen := range m.Tramos {
		tramo, err := reconstruirReferenciaResultadoV1(origen)
		if err != nil {
			return BloqueoResultadoExperienciaV1{}, err
		}
		resultado.tramos[indice] = tramo
	}
	if m.ValorExacto != nil {
		valor, err := nuevoExactoResultadoV1(*m.ValorExacto)
		if err != nil {
			return BloqueoResultadoExperienciaV1{}, err
		}
		resultado.valorExacto = valor
		resultado.tieneValorExacto = true
	}
	if err := validarBloqueoResultadoV1(resultado); err != nil {
		return BloqueoResultadoExperienciaV1{}, err
	}
	return resultado, nil
}
