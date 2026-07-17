package calculoexperienciaoficial

import (
	"crypto/subtle"
	"time"
)

// ValidarPara coteja el recibo con la evidencia V2 y los artefactos exactos
// esperados por el caso de uso. Los limites de solicitud y comprobacion son
// inclusivos; la caducidad de la prueba es exclusiva.
func (r ReciboConsumoAutorizacionFuenteV2) ValidarPara(
	decisionRef, esquemaHuellaDecision, huellaDecisionSHA256 string,
	recursoRef, huellaContextoRecursoSHA256, correlacionRef string,
	huellaSelectorSHA256 string,
	huellaEntradaSHA256 string,
	fuenteExacta ReferenciaExactaV1,
	verificador ReferenciaExactaV1,
	consumoPrueba ReferenciaExactaV1,
	auditoria ReferenciaExactaV1,
	pruebaEmitidaEn, pruebaValidaHasta time.Time,
	obtenidaEn time.Time,
	noAntesDe, noDespuesDe time.Time,
) error {
	if r.Validar() != nil || !instanteReciboConsumoFuenteValido(noAntesDe) ||
		!instanteReciboConsumoFuenteValido(noDespuesDe) ||
		!instanteReciboConsumoFuenteValido(pruebaEmitidaEn) ||
		!instanteReciboConsumoFuenteValido(pruebaValidaHasta) ||
		!instanteReciboConsumoFuenteValido(obtenidaEn) ||
		noDespuesDe.Before(noAntesDe) || !pruebaValidaHasta.After(pruebaEmitidaEn) ||
		!noDespuesDe.Before(pruebaValidaHasta) {
		return nuevoError("recibo_consumo_fuente.cotejo", CodigoValorInvalido)
	}
	m := r.material
	consumidaEn, errConsumo := time.Parse(formatoInstanteReciboConsumoFuenteV2, m.ConsumidaEn)
	obtenidaEnRecibo, errObtencion := time.Parse(formatoInstanteReciboConsumoFuenteV2, m.ObtenidaEn)
	if errConsumo != nil || errObtencion != nil || consumidaEn.Before(noAntesDe) ||
		obtenidaEnRecibo.Before(consumidaEn) || obtenidaEnRecibo.After(noDespuesDe) ||
		m.DecisionRef != decisionRef || m.EsquemaHuellaDecision != esquemaHuellaDecision ||
		!cotejoHuellaReciboConsumo(m.HuellaDecisionSHA256, huellaDecisionSHA256) ||
		m.RecursoRef != recursoRef ||
		!cotejoHuellaReciboConsumo(m.HuellaContextoRecursoSHA256, huellaContextoRecursoSHA256) ||
		m.CorrelacionRef != correlacionRef ||
		!cotejoHuellaReciboConsumo(m.HuellaSelectorSHA256, huellaSelectorSHA256) ||
		!cotejoHuellaReciboConsumo(m.HuellaEntradaSHA256, huellaEntradaSHA256) ||
		!referenciasExactasIguales(m.FuenteExacta, fuenteExacta) ||
		!referenciasExactasIguales(m.Verificador, verificador) ||
		!referenciasExactasIguales(m.ConsumoPrueba, consumoPrueba) ||
		!referenciasExactasIguales(m.Auditoria, auditoria) ||
		m.PruebaEmitidaEn != pruebaEmitidaEn.Format(formatoInstanteReciboConsumoFuenteV2) ||
		m.PruebaValidaHasta != pruebaValidaHasta.Format(formatoInstanteReciboConsumoFuenteV2) ||
		m.ObtenidaEn != obtenidaEn.Format(formatoInstanteReciboConsumoFuenteV2) {
		return nuevoError("recibo_consumo_fuente.cotejo", CodigoHuellaNoCoincide)
	}
	return nil
}

func (r ReciboConsumoAutorizacionFuenteV2) Consumo() (ReferenciaExactaV1, error) {
	if err := r.Validar(); err != nil {
		return ReferenciaExactaV1{}, err
	}
	return r.material.Consumo, nil
}

func (r ReciboConsumoAutorizacionFuenteV2) ConsumidaEn() (time.Time, error) {
	if err := r.Validar(); err != nil {
		return time.Time{}, err
	}
	instante, err := time.Parse(formatoInstanteReciboConsumoFuenteV2, r.material.ConsumidaEn)
	if err != nil {
		return time.Time{}, nuevoError("recibo_consumo_fuente.consumida_en", CodigoValorNoCanonico)
	}
	return instante, nil
}

func (r ReciboConsumoAutorizacionFuenteV2) ObtenidaEn() (time.Time, error) {
	if err := r.Validar(); err != nil {
		return time.Time{}, err
	}
	instante, err := time.Parse(formatoInstanteReciboConsumoFuenteV2, r.material.ObtenidaEn)
	if err != nil {
		return time.Time{}, nuevoError("recibo_consumo_fuente.obtenida_en", CodigoValorNoCanonico)
	}
	return instante, nil
}

func referenciasExactasIguales(a, b ReferenciaExactaV1) bool {
	return a.Referencia == b.Referencia && a.Version == b.Version &&
		cotejoHuellaReciboConsumo(a.HuellaSHA256, b.HuellaSHA256)
}

func cotejoHuellaReciboConsumo(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
