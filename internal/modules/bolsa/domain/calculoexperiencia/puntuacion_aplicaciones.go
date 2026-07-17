package calculoexperiencia

import (
	"vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
)

type calculoAplicacionPuntuacionV1 struct {
	material        AplicacionCalculadaResultadoExperienciaV1
	exactas         racionalExacto
	aportadas       racionalExacto
	bruto           racionalExacto
	redondeado      racionalExacto
	tieneRedondeado bool
	bloqueo         *BloqueoResultadoExperienciaV1
}

func calcularAplicacionPuntuacionV1(
	entrada entradaAplicacionPuntuacionV1,
	acumulador *acumuladorReglaPuntuacionV1,
	contador *contadorOperaciones,
	cero racionalExacto,
	uno racionalExacto,
) (calculoAplicacionPuntuacionV1, error) {
	jornada, factor, err := calcularJornadaPuntuacionV1(
		entrada.tramo, entrada.regla.Jornada(), contador, uno,
	)
	if err != nil {
		return calculoAplicacionPuntuacionV1{}, err
	}
	unidadesExactas := cero
	if entrada.efectiva {
		dias, err := nuevoRacionalExactoDesdeEntero(contador, entrada.temporal.dias)
		if err != nil {
			return calculoAplicacionPuntuacionV1{}, err
		}
		unidadesExactas, err = dias.multiplicar(factor)
		if err == nil {
			unidadesExactas, err = unidadesExactas.dividirPositivo(acumulador.divisorUnidades)
		}
		if err != nil {
			return calculoAplicacionPuntuacionV1{}, err
		}
	}
	aportadas, resto, frontera, err := aplicarRestosAplicacionPuntuacionV1(
		unidadesExactas, entrada.regla.Restos().Modo(), cero,
	)
	if err != nil {
		return calculoAplicacionPuntuacionV1{}, err
	}
	bruto, err := aportadas.multiplicar(acumulador.coeficiente)
	if err != nil {
		return calculoAplicacionPuntuacionV1{}, err
	}
	redondeado, tieneRedondeado := cero, false
	var bloqueo *BloqueoResultadoExperienciaV1
	if entrada.regla.Redondeo().Momento() == reglasbaremo.RedondearPorPeriodo {
		redondeado, err = bruto.redondearAEnteroExacto(entrada.regla.Redondeo().Modo())
		if err != nil && codigoError(err) == CodigoResultadoNoExacto {
			bloqueo, err = nuevoBloqueoRedondeoPuntuacionV1(
				bruto, entrada.regla,
				[]reglasbaremo.ReferenciaVersionada{entrada.tramo.referencia},
			)
		} else if err == nil {
			tieneRedondeado = true
		}
		if err != nil {
			return calculoAplicacionPuntuacionV1{}, err
		}
	}
	resultado, err := materializarAplicacionPuntuacionV1(
		entrada, jornada, unidadesExactas, aportadas, resto, frontera,
		bruto, redondeado, tieneRedondeado,
	)
	if err != nil {
		return calculoAplicacionPuntuacionV1{}, err
	}
	return calculoAplicacionPuntuacionV1{
		material: resultado, exactas: unidadesExactas, aportadas: aportadas,
		bruto: bruto, redondeado: redondeado,
		tieneRedondeado: tieneRedondeado, bloqueo: bloqueo,
	}, nil
}

func calcularJornadaPuntuacionV1(
	tramo TramoExperiencia,
	politica reglasbaremo.PoliticaJornada,
	contador *contadorOperaciones,
	uno racionalExacto,
) (JornadaResultadoExperienciaV1, racionalExacto, error) {
	origen, err := nuevoRacionalExactoDesdeRacional(contador, tramo.jornada.Racional())
	if err != nil {
		return JornadaResultadoExperienciaV1{}, racionalExacto{}, err
	}
	factor := origen
	razon := RazonJornadaProporcional
	usada := false
	switch politica.Modo() {
	case reglasbaremo.JornadaProporcional:
	case reglasbaremo.JornadaIntegra:
		factor, razon = uno, RazonJornadaIntegra
	case reglasbaremo.JornadaIntegraDesdeUmbral:
		umbral, presente := politica.Umbral()
		if !presente {
			return JornadaResultadoExperienciaV1{}, racionalExacto{}, errorContextoPuntuacionV1()
		}
		limite, err := nuevoRacionalExactoDesdeRacional(contador, umbral.Racional())
		if err != nil {
			return JornadaResultadoExperienciaV1{}, racionalExacto{}, err
		}
		comparacion, err := origen.comparar(limite)
		if err != nil {
			return JornadaResultadoExperienciaV1{}, racionalExacto{}, err
		}
		razon = RazonUmbralNoAlcanzado
		if comparacion >= 0 {
			factor, razon = uno, RazonUmbralAlcanzado
		}
	case reglasbaremo.JornadaProtegidaIntegra:
		razon = RazonProteccionNoAtestada
		if tramo.atestacion.EstaAtestado() {
			factor, razon, usada = uno, RazonProteccionAtestada, true
		}
	default:
		return JornadaResultadoExperienciaV1{}, racionalExacto{}, errorContextoPuntuacionV1()
	}
	material, err := nuevoExactoResultadoDesdeRacionalV1(factor)
	if err != nil {
		return JornadaResultadoExperienciaV1{}, racionalExacto{}, err
	}
	return JornadaResultadoExperienciaV1{
		origen: tramo.jornada, modo: politica.Modo(), factor: material,
		atestacionPresente: tramo.atestacion.EstaAtestado(),
		atestacionUsada:    usada, razon: razon,
	}, factor, nil
}

func aplicarRestosAplicacionPuntuacionV1(
	exactas racionalExacto,
	modo reglasbaremo.ModoRestos,
	cero racionalExacto,
) (racionalExacto, racionalExacto, FronteraRestosResultadoExperienciaV1, error) {
	switch modo {
	case reglasbaremo.RestosConservarExactos:
		return exactas, cero, FronteraRestosResultadoExacta, nil
	case reglasbaremo.RestosAcumularPorRegla, reglasbaremo.RestosDescartarPorRegla:
		return exactas, cero, FronteraRestosResultadoRegla, nil
	case reglasbaremo.RestosDescartarPorPeriodo:
		aportadas, err := exactas.truncar()
		if err != nil {
			return racionalExacto{}, racionalExacto{}, "", err
		}
		resto, err := exactas.resto()
		if err != nil {
			return racionalExacto{}, racionalExacto{}, "", err
		}
		return aportadas, resto, FronteraRestosResultadoPeriodo, nil
	default:
		return racionalExacto{}, racionalExacto{}, "", errorContextoPuntuacionV1()
	}
}

func materializarAplicacionPuntuacionV1(
	entrada entradaAplicacionPuntuacionV1,
	jornada JornadaResultadoExperienciaV1,
	exactas racionalExacto,
	aportadas racionalExacto,
	resto racionalExacto,
	frontera FronteraRestosResultadoExperienciaV1,
	bruto racionalExacto,
	redondeado racionalExacto,
	tieneRedondeado bool,
) (AplicacionCalculadaResultadoExperienciaV1, error) {
	exactasResultado, err := nuevoExactoResultadoDesdeRacionalV1(exactas)
	if err != nil {
		return AplicacionCalculadaResultadoExperienciaV1{}, err
	}
	aportadasResultado, err := nuevoExactoResultadoDesdeRacionalV1(aportadas)
	if err != nil {
		return AplicacionCalculadaResultadoExperienciaV1{}, err
	}
	restoResultado, err := nuevoExactoResultadoDesdeRacionalV1(resto)
	if err != nil {
		return AplicacionCalculadaResultadoExperienciaV1{}, err
	}
	brutoResultado, err := nuevoExactoResultadoDesdeRacionalV1(bruto)
	if err != nil {
		return AplicacionCalculadaResultadoExperienciaV1{}, err
	}
	puntuacion := PuntuacionPeriodoResultadoExperienciaV1{bruto: brutoResultado}
	if tieneRedondeado {
		puntuacion.redondeado, err = nuevoExactoResultadoDesdeRacionalV1(redondeado)
		if err != nil {
			return AplicacionCalculadaResultadoExperienciaV1{}, err
		}
		puntuacion.tieneRedondeado = true
	}
	return AplicacionCalculadaResultadoExperienciaV1{
		tramo: entrada.tramo.referencia, reglaClave: entrada.regla.Clave(), jornada: jornada,
		unidades: UnidadesAplicacionResultadoExperienciaV1{
			exactas: exactasResultado, aportadas: aportadasResultado,
			resto: restoResultado, frontera: frontera,
		},
		puntuacion: puntuacion,
	}, nil
}
