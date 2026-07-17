package calculoexperiencia

import (
	"vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
)

type acumuladorReglaPuntuacionV1 struct {
	regla             reglasbaremo.ReglaExperiencia
	divisorUnidades   racionalExacto
	coeficiente       racionalExacto
	unidadesExactas   racionalExacto
	unidadesAportadas racionalExacto
	brutosPeriodos    racionalExacto
	redondeados       racionalExacto
	redondeoImposible bool
}

type reglaCalculadaPuntuacionV1 struct {
	material ResultadoReglaExperienciaV1
	final    racionalExacto
}

func ejecutarPuntuacionV1(
	contexto contextoPuntuacionV1,
	contador *contadorOperaciones,
) (resultadoPuntuacionExperienciaV1, error) {
	cero, err := nuevoRacionalExactoDesdeEntero(contador, 0)
	if err != nil {
		return resultadoPuntuacionExperienciaV1{}, err
	}
	uno, err := nuevoRacionalExactoDesdeEntero(contador, 1)
	if err != nil {
		return resultadoPuntuacionExperienciaV1{}, err
	}
	acumuladores, posiciones, err := prepararAcumuladoresPuntuacionV1(contexto.reglas, contador, cero)
	if err != nil {
		return resultadoPuntuacionExperienciaV1{}, err
	}
	resultado := resultadoPuntuacionExperienciaV1{
		intervalos:   intervalosDesdeContextoPuntuacionV1(contexto),
		aplicaciones: make([]AplicacionCalculadaResultadoExperienciaV1, 0, len(contexto.orden)),
	}
	for _, entrada := range contexto.orden {
		posicion, existe := posiciones[entrada.regla.Clave()]
		if !existe {
			return resultadoPuntuacionExperienciaV1{}, errorContextoPuntuacionV1()
		}
		calculo, err :=
			calcularAplicacionPuntuacionV1(entrada, &acumuladores[posicion], contador, cero, uno)
		if err != nil {
			return resultadoPuntuacionExperienciaV1{}, err
		}
		resultado.aplicaciones = append(resultado.aplicaciones, calculo.material)
		acumulador := &acumuladores[posicion]
		acumulador.unidadesExactas, err = acumulador.unidadesExactas.sumar(calculo.exactas)
		if err != nil {
			return resultadoPuntuacionExperienciaV1{}, err
		}
		acumulador.unidadesAportadas, err = acumulador.unidadesAportadas.sumar(calculo.aportadas)
		if err != nil {
			return resultadoPuntuacionExperienciaV1{}, err
		}
		acumulador.brutosPeriodos, err = acumulador.brutosPeriodos.sumar(calculo.bruto)
		if err != nil {
			return resultadoPuntuacionExperienciaV1{}, err
		}
		if calculo.tieneRedondeado {
			acumulador.redondeados, err = acumulador.redondeados.sumar(calculo.redondeado)
			if err != nil {
				return resultadoPuntuacionExperienciaV1{}, err
			}
		} else if calculo.bloqueo != nil {
			acumulador.redondeoImposible = true
			resultado.bloqueos = append(resultado.bloqueos, *calculo.bloqueo)
		}
	}

	reglasCalculadas := make([]reglaCalculadaPuntuacionV1, 0, len(acumuladores))
	for indice := range acumuladores {
		regla, bloqueo, err := calcularReglaPuntuacionV1(&acumuladores[indice], contador, cero)
		if err != nil {
			return resultadoPuntuacionExperienciaV1{}, err
		}
		if bloqueo != nil {
			resultado.bloqueos = append(resultado.bloqueos, *bloqueo)
			continue
		}
		if !acumuladores[indice].redondeoImposible {
			reglasCalculadas = append(reglasCalculadas, regla)
		}
	}
	if resultado.bloqueada() {
		return resultado, nil
	}
	resultado.reglas = make([]ResultadoReglaExperienciaV1, len(reglasCalculadas))
	for indice := range reglasCalculadas {
		resultado.reglas[indice] = reglasCalculadas[indice].material
	}
	secciones, total, err := calcularSeccionesPuntuacionV1(
		contexto.secciones, reglasCalculadas, contador, cero,
	)
	if err != nil {
		return resultadoPuntuacionExperienciaV1{}, err
	}
	resultado.secciones, resultado.total = secciones, total
	return resultado, nil
}

func prepararAcumuladoresPuntuacionV1(
	reglas []reglasbaremo.ReglaExperiencia,
	contador *contadorOperaciones,
	cero racionalExacto,
) ([]acumuladorReglaPuntuacionV1, map[string]int, error) {
	resultado := make([]acumuladorReglaPuntuacionV1, len(reglas))
	posiciones := make(map[string]int, len(reglas))
	for indice, regla := range reglas {
		divisor, err := nuevoRacionalExactoDesdeRacional(
			contador, regla.UnidadTemporal().UnidadesBasePorUnidad(),
		)
		if err != nil {
			return nil, nil, err
		}
		coeficiente, err := nuevoRacionalExactoDesdeEntero(
			contador, regla.PuntosPorUnidad().Micropuntos(),
		)
		if err != nil {
			return nil, nil, err
		}
		resultado[indice] = acumuladorReglaPuntuacionV1{
			regla: regla, divisorUnidades: divisor, coeficiente: coeficiente,
			unidadesExactas: cero, unidadesAportadas: cero,
			brutosPeriodos: cero, redondeados: cero,
		}
		posiciones[regla.Clave()] = indice
	}
	return resultado, posiciones, nil
}

func nuevoBloqueoRedondeoPuntuacionV1(
	entrada racionalExacto,
	regla reglasbaremo.ReglaExperiencia,
	tramo []reglasbaremo.ReferenciaVersionada,
) (*BloqueoResultadoExperienciaV1, error) {
	valor, err := nuevoExactoResultadoDesdeRacionalV1(entrada)
	if err != nil {
		return nil, err
	}
	return &BloqueoResultadoExperienciaV1{
		codigo: BloqueoResultadoRedondeoNoExacto, tramos: tramo,
		reglas: []string{regla.Clave()}, seccionClave: regla.SeccionClave(),
		valorExacto: valor, tieneValorExacto: true,
	}, nil
}
