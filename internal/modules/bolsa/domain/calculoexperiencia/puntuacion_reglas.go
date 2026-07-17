package calculoexperiencia

import (
	"vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
)

func calcularReglaPuntuacionV1(
	acumulador *acumuladorReglaPuntuacionV1,
	contador *contadorOperaciones,
	cero racionalExacto,
) (reglaCalculadaPuntuacionV1, *BloqueoResultadoExperienciaV1, error) {
	if acumulador == nil {
		return reglaCalculadaPuntuacionV1{}, nil, errorContextoPuntuacionV1()
	}
	trasRestos, resto, err := aplicarRestosReglaPuntuacionV1(
		acumulador.unidadesExactas, acumulador.unidadesAportadas,
		acumulador.regla.Restos().Modo(), cero,
	)
	if err != nil {
		return reglaCalculadaPuntuacionV1{}, nil, err
	}
	topeUnidades, unidadesLimitadas, err := aplicarTopeUnidadesPuntuacionV1(
		trasRestos, acumulador.regla.MaximoUnidades(), contador,
	)
	if err != nil {
		return reglaCalculadaPuntuacionV1{}, nil, err
	}
	bruto := acumulador.brutosPeriodos
	salidaRedondeo := acumulador.redondeados
	if acumulador.regla.Redondeo().Momento() == reglasbaremo.RedondearPorRegla {
		bruto, err = unidadesLimitadas.multiplicar(acumulador.coeficiente)
		if err != nil {
			return reglaCalculadaPuntuacionV1{}, nil, err
		}
		salidaRedondeo, err = bruto.redondearAEnteroExacto(acumulador.regla.Redondeo().Modo())
		if err != nil && codigoError(err) == CodigoResultadoNoExacto {
			bloqueo, bloqueoErr := nuevoBloqueoRedondeoPuntuacionV1(
				bruto, acumulador.regla, nil,
			)
			return reglaCalculadaPuntuacionV1{}, bloqueo, bloqueoErr
		}
		if err != nil {
			return reglaCalculadaPuntuacionV1{}, nil, err
		}
	} else if acumulador.regla.Redondeo().Momento() != reglasbaremo.RedondearPorPeriodo {
		return reglaCalculadaPuntuacionV1{}, nil, errorContextoPuntuacionV1()
	}
	topePuntos, final, err := aplicarTopePuntosReglaPuntuacionV1(
		salidaRedondeo, acumulador.regla.MaximoPuntos(), contador,
	)
	if err != nil {
		return reglaCalculadaPuntuacionV1{}, nil, err
	}
	material, err := materializarReglaPuntuacionV1(
		acumulador, trasRestos, resto, topeUnidades, bruto,
		salidaRedondeo, topePuntos, final,
	)
	if err != nil {
		return reglaCalculadaPuntuacionV1{}, nil, err
	}
	return reglaCalculadaPuntuacionV1{material: material, final: final}, nil, nil
}

func aplicarRestosReglaPuntuacionV1(
	exactas racionalExacto,
	aportadas racionalExacto,
	modo reglasbaremo.ModoRestos,
	cero racionalExacto,
) (racionalExacto, racionalExacto, error) {
	switch modo {
	case reglasbaremo.RestosConservarExactos, reglasbaremo.RestosAcumularPorRegla:
		return exactas, cero, nil
	case reglasbaremo.RestosDescartarPorPeriodo:
		resto, err := exactas.restar(aportadas)
		if err != nil {
			return racionalExacto{}, racionalExacto{}, err
		}
		return aportadas, resto, nil
	case reglasbaremo.RestosDescartarPorRegla:
		enteras, err := exactas.truncar()
		if err != nil {
			return racionalExacto{}, racionalExacto{}, err
		}
		resto, err := exactas.resto()
		if err != nil {
			return racionalExacto{}, racionalExacto{}, err
		}
		return enteras, resto, nil
	default:
		return racionalExacto{}, racionalExacto{}, errorContextoPuntuacionV1()
	}
}

func aplicarTopeUnidadesPuntuacionV1(
	valor racionalExacto,
	limite reglasbaremo.LimiteUnidades,
	contador *contadorOperaciones,
) (TopeResultadoExperienciaV1, racionalExacto, error) {
	publicado, limitado := limite.Valor()
	if !limitado {
		return materializarTopePuntuacionV1(valor, racionalExacto{}, false, valor, false)
	}
	maximo, err := nuevoRacionalExactoDesdeRacional(contador, publicado)
	if err != nil {
		return TopeResultadoExperienciaV1{}, racionalExacto{}, err
	}
	return aplicarTopeRacionalPuntuacionV1(valor, maximo)
}

func aplicarTopePuntosReglaPuntuacionV1(
	valor racionalExacto,
	limite reglasbaremo.LimitePuntos,
	contador *contadorOperaciones,
) (TopeResultadoExperienciaV1, racionalExacto, error) {
	publicado, limitado := limite.Valor()
	if !limitado {
		return materializarTopePuntuacionV1(valor, racionalExacto{}, false, valor, false)
	}
	maximo, err := nuevoRacionalExactoDesdeEntero(contador, publicado.Micropuntos())
	if err != nil {
		return TopeResultadoExperienciaV1{}, racionalExacto{}, err
	}
	return aplicarTopeRacionalPuntuacionV1(valor, maximo)
}

func aplicarTopeRacionalPuntuacionV1(
	valor racionalExacto,
	limite racionalExacto,
) (TopeResultadoExperienciaV1, racionalExacto, error) {
	comparacion, err := valor.comparar(limite)
	if err != nil {
		return TopeResultadoExperienciaV1{}, racionalExacto{}, err
	}
	despues, aplicado := valor, false
	if comparacion > 0 {
		despues, aplicado = limite, true
	}
	return materializarTopePuntuacionV1(valor, limite, true, despues, aplicado)
}

func materializarTopePuntuacionV1(
	antes racionalExacto,
	limite racionalExacto,
	limitado bool,
	despues racionalExacto,
	aplicado bool,
) (TopeResultadoExperienciaV1, racionalExacto, error) {
	antesResultado, err := nuevoExactoResultadoDesdeRacionalV1(antes)
	if err != nil {
		return TopeResultadoExperienciaV1{}, racionalExacto{}, err
	}
	despuesResultado, err := nuevoExactoResultadoDesdeRacionalV1(despues)
	if err != nil {
		return TopeResultadoExperienciaV1{}, racionalExacto{}, err
	}
	resultado := TopeResultadoExperienciaV1{
		antes: antesResultado, limitado: limitado,
		despues: despuesResultado, aplicado: aplicado,
	}
	if limitado {
		resultado.limite, err = nuevoExactoResultadoDesdeRacionalV1(limite)
		if err != nil {
			return TopeResultadoExperienciaV1{}, racionalExacto{}, err
		}
	}
	return resultado, despues, nil
}

func materializarReglaPuntuacionV1(
	acumulador *acumuladorReglaPuntuacionV1,
	trasRestos racionalExacto,
	resto racionalExacto,
	topeUnidades TopeResultadoExperienciaV1,
	bruto racionalExacto,
	salidaRedondeo racionalExacto,
	topePuntos TopeResultadoExperienciaV1,
	final racionalExacto,
) (ResultadoReglaExperienciaV1, error) {
	agregadasResultado, err := nuevoExactoResultadoDesdeRacionalV1(acumulador.unidadesExactas)
	if err != nil {
		return ResultadoReglaExperienciaV1{}, err
	}
	trasRestosResultado, err := nuevoExactoResultadoDesdeRacionalV1(trasRestos)
	if err != nil {
		return ResultadoReglaExperienciaV1{}, err
	}
	restoResultado, err := nuevoExactoResultadoDesdeRacionalV1(resto)
	if err != nil {
		return ResultadoReglaExperienciaV1{}, err
	}
	brutoResultado, err := nuevoExactoResultadoDesdeRacionalV1(bruto)
	if err != nil {
		return ResultadoReglaExperienciaV1{}, err
	}
	salidaResultado, err := nuevoExactoResultadoDesdeRacionalV1(salidaRedondeo)
	if err != nil {
		return ResultadoReglaExperienciaV1{}, err
	}
	finalResultado, err := nuevoExactoResultadoDesdeRacionalV1(final)
	if err != nil {
		return ResultadoReglaExperienciaV1{}, err
	}
	return ResultadoReglaExperienciaV1{
		seccionClave: acumulador.regla.SeccionClave(), reglaClave: acumulador.regla.Clave(),
		unidadesAgregadas: agregadasResultado, unidadesTrasRestos: trasRestosResultado,
		restoRegla: restoResultado, topeUnidades: topeUnidades,
		coeficiente: acumulador.regla.PuntosPorUnidad(), bruto: brutoResultado,
		redondeo: RedondeoResultadoExperienciaV1{
			momento: acumulador.regla.Redondeo().Momento(),
			modo:    acumulador.regla.Redondeo().Modo(), entrada: brutoResultado, salida: salidaResultado,
		},
		topePuntos: topePuntos, puntosFinales: finalResultado,
	}, nil
}
