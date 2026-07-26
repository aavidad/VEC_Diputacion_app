package domain

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"sort"
)

func calcularHuellaSemanticaPropuestaDecisionCobertura(
	canon CanonHuellaSemanticaPropuestaDecisionCobertura,
	p PublicacionPropuestaDecisionCobertura,
) (string, error) {
	material, err := materialCanonicoSemanticoPropuestaDecisionCoberturaV1(
		canon,
		p,
	)
	if err != nil {
		return "", err
	}
	huella := sha256.Sum256(material)
	return hex.EncodeToString(huella[:]), nil
}

func materialCanonicoSemanticoPropuestaDecisionCoberturaV1(
	canon CanonHuellaSemanticaPropuestaDecisionCobertura,
	p PublicacionPropuestaDecisionCobertura,
) ([]byte, error) {
	resultados, err := normalizarResultadosSemanticosPropuesta(p.Resultados)
	if err != nil {
		return nil, ErrDatoInvalido
	}
	evaluaciones, err := normalizarEvaluacionesSemanticasPropuesta(
		p.Evaluaciones,
	)
	if !canon.valido() || err != nil ||
		!referenciaValida(p.OrganizacionRef) ||
		!referenciaValida(p.ExpedienteRef) ||
		p.VersionExpediente == 0 ||
		p.VersionExpediente > maximoEnteroSeguroCatalogoCobertura ||
		!referenciaValida(p.AnalisisRef) ||
		!huellaValida(p.AnalisisHuellaSHA256) ||
		p.Catalogo.Validar() != nil ||
		p.Politica.Validar() != nil ||
		!p.FinalidadClave.Valida() ||
		!referenciaValida(p.FinalidadRef) ||
		!referenciaValida(p.CategoriaRef) ||
		!periodoAnalisisValido(p.Periodo) ||
		!p.Estado.valido() ||
		(p.Estado == PropuestaCoberturaViable) != p.ViaPropuesta.Valida() {
		return nil, ErrDatoInvalido
	}
	var destino bytes.Buffer
	e := escritorCanonCatalogo{destino: &destino}
	e.cadena(canon.Dominio)
	e.entero16(canon.VersionEsquema)
	e.cadena(canon.Algoritmo)
	e.cadena(p.OrganizacionRef)
	e.cadena(p.ExpedienteRef)
	e.entero64(p.VersionExpediente)
	e.cadena(p.AnalisisRef)
	e.cadena(p.AnalisisHuellaSHA256)
	escribirIdentidadCatalogoPropuesta(&e, p.Catalogo)
	e.cadena(p.Politica.Referencia)
	e.entero64(p.Politica.Version)
	e.cadena(p.Politica.HuellaSHA256)
	e.cadena(string(p.FinalidadClave))
	e.cadena(p.FinalidadRef)
	e.cadena(p.CategoriaRef)
	e.instante(p.Periodo.Inicio)
	e.instante(p.Periodo.Fin)
	e.cadena(string(p.Estado))
	e.cadena(string(p.ViaPropuesta))
	escribirResultadosSemanticosPropuesta(&e, resultados)
	escribirEvaluacionesPropuesta(&e, evaluaciones)
	if e.err != nil {
		return nil, ErrDatoInvalido
	}
	return destino.Bytes(), nil
}

type resultadoSemanticoPropuestaCobertura struct {
	Clave      ClaveCatalogo
	Resultados []ResultadoComprobacion
}

func normalizarResultadosSemanticosPropuesta(
	origen []ResultadoAgrupadoPropuestaCobertura,
) ([]resultadoSemanticoPropuestaCobertura, error) {
	if len(origen) > maximoComprobacionesCatalogo {
		return nil, ErrDatoInvalido
	}
	salida := make([]resultadoSemanticoPropuestaCobertura, 0, len(origen))
	claves := make(map[ClaveCatalogo]struct{}, len(origen))
	totalEvidencias := 0
	for _, agrupado := range origen {
		if !agrupado.Clave.Valida() || len(agrupado.Evidencias) == 0 ||
			len(agrupado.Evidencias) >
				maximoEvidenciasPorComprobacionCobertura {
			return nil, ErrDatoInvalido
		}
		totalEvidencias += len(agrupado.Evidencias)
		if totalEvidencias > maximoResultadosEntradaPropuestaCobertura {
			return nil, ErrDatoInvalido
		}
		if _, repetida := claves[agrupado.Clave]; repetida {
			return nil, ErrDatoInvalido
		}
		claves[agrupado.Clave] = struct{}{}
		vistos := make(map[ResultadoComprobacion]struct{})
		resultado := resultadoSemanticoPropuestaCobertura{
			Clave: agrupado.Clave,
		}
		for _, evidencia := range agrupado.Evidencias {
			if !evidencia.Resultado.valido() {
				return nil, ErrDatoInvalido
			}
			if _, repetido := vistos[evidencia.Resultado]; repetido {
				continue
			}
			vistos[evidencia.Resultado] = struct{}{}
			resultado.Resultados = append(
				resultado.Resultados,
				evidencia.Resultado,
			)
		}
		sort.Slice(resultado.Resultados, func(i, j int) bool {
			return resultado.Resultados[i] < resultado.Resultados[j]
		})
		salida = append(salida, resultado)
	}
	sort.Slice(salida, func(i, j int) bool {
		return salida[i].Clave < salida[j].Clave
	})
	return salida, nil
}

func escribirResultadosSemanticosPropuesta(
	e *escritorCanonCatalogo,
	resultados []resultadoSemanticoPropuestaCobertura,
) {
	e.entero32(uint32(len(resultados)))
	for _, resultado := range resultados {
		e.cadena(string(resultado.Clave))
		e.entero32(uint32(len(resultado.Resultados)))
		for _, valor := range resultado.Resultados {
			e.cadena(string(valor))
		}
	}
}

func normalizarEvaluacionesSemanticasPropuesta(
	origen []EvaluacionViaPropuestaCobertura,
) ([]EvaluacionViaPropuestaCobertura, error) {
	if len(origen) == 0 || len(origen) > maximoViasCobertura {
		return nil, ErrDatoInvalido
	}
	salida := clonarEvaluacionesViaPropuesta(origen)
	vias := make(map[ClaveCatalogo]struct{}, len(salida))
	prioridades := make(map[uint16]struct{}, len(salida))
	for indice := range salida {
		evaluacion := &salida[indice]
		if !evaluacion.ViaClave.Valida() || evaluacion.Prioridad == 0 ||
			!estadoEvaluacionViaCoberturaValido(evaluacion.Estado) ||
			!normalizarListasEvaluacionSemantica(evaluacion) {
			return nil, ErrDatoInvalido
		}
		if _, repetida := vias[evaluacion.ViaClave]; repetida {
			return nil, ErrDatoInvalido
		}
		if _, repetida := prioridades[evaluacion.Prioridad]; repetida {
			return nil, ErrDatoInvalido
		}
		vias[evaluacion.ViaClave] = struct{}{}
		prioridades[evaluacion.Prioridad] = struct{}{}
	}
	sort.Slice(salida, func(i, j int) bool {
		if salida[i].Prioridad != salida[j].Prioridad {
			return salida[i].Prioridad < salida[j].Prioridad
		}
		return salida[i].ViaClave < salida[j].ViaClave
	})
	return salida, nil
}

func estadoEvaluacionViaCoberturaValido(e EstadoEvaluacionViaCobertura) bool {
	return e == EvaluacionViaCoberturaViable ||
		e == EvaluacionViaCoberturaIncompleta ||
		e == EvaluacionViaCoberturaConflictiva ||
		e == EvaluacionViaCoberturaNoViable
}

func normalizarListasEvaluacionSemantica(
	e *EvaluacionViaPropuestaCobertura,
) bool {
	listas := []*[]ClaveCatalogo{
		&e.ResultadosOmitidos,
		&e.AusenciasBloqueantes,
		&e.AusenciasAdmitidas,
		&e.NoHabilitantes,
		&e.Conflictos,
	}
	vistas := make(map[ClaveCatalogo]struct{})
	total := 0
	for _, lista := range listas {
		if len(*lista) > maximoComprobacionesPorViaCobertura {
			return false
		}
		sort.Slice(*lista, func(i, j int) bool { return (*lista)[i] < (*lista)[j] })
		for _, clave := range *lista {
			if !clave.Valida() {
				return false
			}
			if _, repetida := vistas[clave]; repetida {
				return false
			}
			vistas[clave] = struct{}{}
			total++
		}
	}
	return total <= maximoComprobacionesPorViaCobertura
}
