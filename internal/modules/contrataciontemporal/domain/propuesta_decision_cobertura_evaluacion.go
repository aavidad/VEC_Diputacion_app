package domain

import (
	"sort"
	"time"
)

func agruparResultadosPropuestaCobertura(
	resultados []ComprobacionCobertura,
	catalogo CatalogoViasCobertura,
	generadaEn time.Time,
) ([]ResultadoAgrupadoPropuestaCobertura, error) {
	clavesPublicadas := make(map[ClaveCatalogo]struct{})
	for _, via := range catalogo.Vias() {
		for _, comprobacion := range via.Comprobaciones {
			clavesPublicadas[comprobacion.Clave] = struct{}{}
		}
	}
	agrupados := make(map[ClaveCatalogo][]EvidenciaComprobacionPropuestaCobertura)
	for _, resultado := range resultados {
		if resultado.Validar() != nil || resultado.Detalle != "" ||
			resultado.EvaluadaEn.After(generadaEn) {
			return nil, ErrDatoInvalido
		}
		if _, publicada := clavesPublicadas[resultado.Clave]; !publicada {
			return nil, ErrDatoInvalido
		}
		evidencia := EvidenciaComprobacionPropuestaCobertura{
			Resultado:  resultado.Resultado,
			FuenteRef:  resultado.FuenteRef,
			ReciboRef:  resultado.ReciboRef,
			EvaluadaEn: resultado.EvaluadaEn,
		}
		if evidencia.validar() != nil {
			return nil, ErrDatoInvalido
		}
		existentes := agrupados[resultado.Clave]
		repetida := false
		for _, existente := range existentes {
			if existente == evidencia {
				repetida = true
				break
			}
		}
		if !repetida {
			if len(existentes) >=
				maximoEvidenciasPorComprobacionCobertura {
				return nil, ErrDatoInvalido
			}
			agrupados[resultado.Clave] = append(existentes, evidencia)
		}
	}
	claves := make([]ClaveCatalogo, 0, len(agrupados))
	for clave := range agrupados {
		claves = append(claves, clave)
	}
	ordenarClavesCatalogo(claves)
	salida := make([]ResultadoAgrupadoPropuestaCobertura, 0, len(claves))
	for _, clave := range claves {
		evidencias := agrupados[clave]
		sort.Slice(evidencias, func(i, j int) bool {
			return evidenciaPropuestaCoberturaMenor(
				evidencias[i],
				evidencias[j],
			)
		})
		salida = append(salida, ResultadoAgrupadoPropuestaCobertura{
			Clave:      clave,
			Evidencias: evidencias,
		})
	}
	return salida, nil
}

func evidenciaPropuestaCoberturaMenor(
	izquierda EvidenciaComprobacionPropuestaCobertura,
	derecha EvidenciaComprobacionPropuestaCobertura,
) bool {
	if izquierda.Resultado != derecha.Resultado {
		return izquierda.Resultado < derecha.Resultado
	}
	if izquierda.FuenteRef != derecha.FuenteRef {
		return izquierda.FuenteRef < derecha.FuenteRef
	}
	if izquierda.ReciboRef != derecha.ReciboRef {
		return izquierda.ReciboRef < derecha.ReciboRef
	}
	return izquierda.EvaluadaEn.Before(derecha.EvaluadaEn)
}

func evaluarPropuestaCobertura(
	vias []ReglaViaDecisionCobertura,
	resultados []ResultadoAgrupadoPropuestaCobertura,
) (
	[]EvaluacionViaPropuestaCobertura,
	EstadoPropuestaDecisionCobertura,
	ClaveCatalogo,
) {
	porClave := make(map[ClaveCatalogo]ResultadoAgrupadoPropuestaCobertura,
		len(resultados))
	hayConflicto := false
	for _, resultado := range resultados {
		porClave[resultado.Clave] = resultado
		hayConflicto = hayConflicto || resultado.conflictivo()
	}
	evaluaciones := make([]EvaluacionViaPropuestaCobertura, 0, len(vias))
	hayAusenciaBloqueante := false
	var primeraViable ClaveCatalogo
	for _, via := range vias {
		evaluacion := EvaluacionViaPropuestaCobertura{
			ViaClave:  via.ViaClave,
			Prioridad: via.Prioridad,
			Estado:    EvaluacionViaCoberturaViable,
		}
		for _, regla := range via.Comprobaciones {
			resultado, consta := porClave[regla.Clave]
			if !consta {
				evaluacion.ResultadosOmitidos = append(
					evaluacion.ResultadosOmitidos,
					regla.Clave,
				)
				hayAusenciaBloqueante = true
				continue
			}
			if resultado.conflictivo() {
				evaluacion.Conflictos = append(
					evaluacion.Conflictos,
					regla.Clave,
				)
				continue
			}
			resultadoFuncional := resultado.Evidencias[0].Resultado
			if resultadoFuncional == ComprobacionNoConsta {
				if regla.TratamientoAusencia == AusenciaCoberturaBloquea {
					evaluacion.AusenciasBloqueantes = append(
						evaluacion.AusenciasBloqueantes,
						regla.Clave,
					)
					hayAusenciaBloqueante = true
				} else {
					evaluacion.AusenciasAdmitidas = append(
						evaluacion.AusenciasAdmitidas,
						regla.Clave,
					)
				}
				continue
			}
			if !regla.habilita(resultadoFuncional) {
				evaluacion.NoHabilitantes = append(
					evaluacion.NoHabilitantes,
					regla.Clave,
				)
			}
		}
		switch {
		case len(evaluacion.Conflictos) > 0:
			evaluacion.Estado = EvaluacionViaCoberturaConflictiva
		case len(evaluacion.ResultadosOmitidos) > 0 ||
			len(evaluacion.AusenciasBloqueantes) > 0:
			evaluacion.Estado = EvaluacionViaCoberturaIncompleta
		case len(evaluacion.NoHabilitantes) > 0:
			evaluacion.Estado = EvaluacionViaCoberturaNoViable
		default:
			if primeraViable == "" {
				primeraViable = via.ViaClave
			}
		}
		evaluaciones = append(evaluaciones, evaluacion)
	}
	switch {
	case hayConflicto:
		return evaluaciones, PropuestaCoberturaConflictiva, ""
	case hayAusenciaBloqueante:
		return evaluaciones, PropuestaCoberturaIncompleta, ""
	case primeraViable != "":
		return evaluaciones, PropuestaCoberturaViable, primeraViable
	default:
		return evaluaciones, PropuestaCoberturaSinVia, ""
	}
}

func desagruparResultadosPropuestaCobertura(
	resultados []ResultadoAgrupadoPropuestaCobertura,
) []ComprobacionCobertura {
	cantidad := 0
	for _, resultado := range resultados {
		cantidad += len(resultado.Evidencias)
	}
	salida := make([]ComprobacionCobertura, 0, cantidad)
	for _, resultado := range resultados {
		for _, evidencia := range resultado.Evidencias {
			salida = append(salida, ComprobacionCobertura{
				Clave:      resultado.Clave,
				Resultado:  evidencia.Resultado,
				FuenteRef:  evidencia.FuenteRef,
				ReciboRef:  evidencia.ReciboRef,
				EvaluadaEn: evidencia.EvaluadaEn,
			})
		}
	}
	return salida
}
