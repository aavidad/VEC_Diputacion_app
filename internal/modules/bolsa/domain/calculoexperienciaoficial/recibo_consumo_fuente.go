package calculoexperienciaoficial

import (
	"bytes"
	"crypto/subtle"
	"encoding/json"
	"time"
)

const (
	EsquemaReciboConsumoAutorizacionFuenteV2 = "vec.bolsa.calculo-experiencia-oficial.recibo-consumo-autorizacion-fuente.v2"
	formatoInstanteReciboConsumoFuenteV2     = "2006-01-02T15:04:05.000000Z"
)

// ReciboConsumoAutorizacionFuenteV2 es la atestacion opaca que devuelve una
// fuente durable tras consumir una decision de lectura. No concede autoridad:
// permite cotejar que la decision, el selector y los artefactos leidos fueron
// exactamente los mismos que intervienen en el calculo.
type ReciboConsumoAutorizacionFuenteV2 struct {
	material materialReciboConsumoAutorizacionFuenteV2
}

type materialReciboConsumoAutorizacionFuenteV2 struct {
	Esquema                     string             `json:"esquema"`
	Consumo                     ReferenciaExactaV1 `json:"consumo"`
	DecisionRef                 string             `json:"decision_ref"`
	EsquemaHuellaDecision       string             `json:"esquema_huella_decision"`
	HuellaDecisionSHA256        string             `json:"huella_decision_sha256"`
	RecursoRef                  string             `json:"recurso_ref"`
	HuellaContextoRecursoSHA256 string             `json:"huella_contexto_recurso_sha256"`
	CorrelacionRef              string             `json:"correlacion_ref"`
	HuellaSelectorSHA256        string             `json:"huella_selector_sha256"`
	HuellaEntradaSHA256         string             `json:"huella_entrada_sha256"`
	FuenteExacta                ReferenciaExactaV1 `json:"fuente_exacta"`
	Verificador                 ReferenciaExactaV1 `json:"verificador"`
	ConsumoPrueba               ReferenciaExactaV1 `json:"consumo_prueba"`
	Auditoria                   ReferenciaExactaV1 `json:"auditoria"`
	PruebaEmitidaEn             string             `json:"prueba_emitida_en"`
	PruebaValidaHasta           string             `json:"prueba_valida_hasta"`
	ConsumidaEn                 string             `json:"consumida_en"`
	ObtenidaEn                  string             `json:"obtenida_en"`
}

// NuevoReciboConsumoAutorizacionFuenteV2 no acepta un DTO libre para evitar
// reconstrucciones parciales en fronteras de transporte.
func NuevoReciboConsumoAutorizacionFuenteV2(
	consumo ReferenciaExactaV1,
	decisionRef, esquemaHuellaDecision, huellaDecisionSHA256 string,
	recursoRef, huellaContextoRecursoSHA256, correlacionRef string,
	huellaSelectorSHA256 string,
	huellaEntradaSHA256 string,
	fuenteExacta ReferenciaExactaV1,
	verificador ReferenciaExactaV1,
	consumoPrueba ReferenciaExactaV1,
	auditoria ReferenciaExactaV1,
	pruebaEmitidaEn, pruebaValidaHasta time.Time,
	consumidaEn, obtenidaEn time.Time,
) (ReciboConsumoAutorizacionFuenteV2, error) {
	// No normalizar silenciosamente zona ni precision: los instantes deben
	// venir ya canonizados desde la frontera durable que acredita el consumo.
	if !instanteReciboConsumoFuenteValido(pruebaEmitidaEn) ||
		!instanteReciboConsumoFuenteValido(pruebaValidaHasta) ||
		!instanteReciboConsumoFuenteValido(consumidaEn) ||
		!instanteReciboConsumoFuenteValido(obtenidaEn) {
		return ReciboConsumoAutorizacionFuenteV2{},
			nuevoError("recibo_consumo_fuente.instantes", CodigoValorNoCanonico)
	}
	recibo := ReciboConsumoAutorizacionFuenteV2{material: materialReciboConsumoAutorizacionFuenteV2{
		Esquema: EsquemaReciboConsumoAutorizacionFuenteV2,
		Consumo: consumo, DecisionRef: decisionRef,
		EsquemaHuellaDecision: esquemaHuellaDecision,
		HuellaDecisionSHA256:  huellaDecisionSHA256,
		RecursoRef:            recursoRef, HuellaContextoRecursoSHA256: huellaContextoRecursoSHA256,
		CorrelacionRef: correlacionRef, HuellaSelectorSHA256: huellaSelectorSHA256,
		HuellaEntradaSHA256: huellaEntradaSHA256,
		FuenteExacta:        fuenteExacta, Verificador: verificador,
		ConsumoPrueba: consumoPrueba, Auditoria: auditoria,
		PruebaEmitidaEn:   pruebaEmitidaEn.Format(formatoInstanteReciboConsumoFuenteV2),
		PruebaValidaHasta: pruebaValidaHasta.Format(formatoInstanteReciboConsumoFuenteV2),
		ConsumidaEn:       consumidaEn.Format(formatoInstanteReciboConsumoFuenteV2),
		ObtenidaEn:        obtenidaEn.Format(formatoInstanteReciboConsumoFuenteV2),
	}}
	if err := recibo.Validar(); err != nil {
		return ReciboConsumoAutorizacionFuenteV2{}, err
	}
	return recibo, nil
}

func (r ReciboConsumoAutorizacionFuenteV2) Validar() error {
	m := r.material
	pruebaEmitidaEn, errEmitida := time.Parse(
		formatoInstanteReciboConsumoFuenteV2, m.PruebaEmitidaEn,
	)
	pruebaValidaHasta, errValidez := time.Parse(
		formatoInstanteReciboConsumoFuenteV2, m.PruebaValidaHasta,
	)
	consumidaEn, errConsumo := time.Parse(formatoInstanteReciboConsumoFuenteV2, m.ConsumidaEn)
	obtenidaEn, errObtencion := time.Parse(formatoInstanteReciboConsumoFuenteV2, m.ObtenidaEn)
	if m.Esquema != EsquemaReciboConsumoAutorizacionFuenteV2 ||
		!referenciaExactaNominalReciboValida(m.Consumo, "consumo:autorizacion:") ||
		!decisionRefReciboConsumoFuenteValida(m.DecisionRef) ||
		!tokenTecnicoValido(m.EsquemaHuellaDecision, 160) ||
		!huellaSHA256Valida(m.HuellaDecisionSHA256) ||
		!recursoLecturaReciboConsumoFuenteValido(m.RecursoRef) ||
		!huellaSHA256Valida(m.HuellaContextoRecursoSHA256) ||
		!correlacionReciboConsumoFuenteValida(m.CorrelacionRef) ||
		!huellaSHA256Valida(m.HuellaSelectorSHA256) ||
		!huellaSHA256Valida(m.HuellaEntradaSHA256) ||
		!referenciaExactaNominalReciboValida(m.FuenteExacta, "evidencia:fuente:") ||
		!referenciaExactaNominalReciboValida(m.Verificador, "verificador:fuente:") ||
		!referenciaExactaNominalReciboValida(m.ConsumoPrueba, "consumo:prueba:") ||
		!referenciaExactaNominalReciboValida(m.Auditoria, "auditoria:fuente:") ||
		errEmitida != nil || errValidez != nil || errConsumo != nil || errObtencion != nil ||
		!instanteReciboConsumoFuenteValido(pruebaEmitidaEn) ||
		!instanteReciboConsumoFuenteValido(pruebaValidaHasta) ||
		!instanteReciboConsumoFuenteValido(consumidaEn) ||
		!instanteReciboConsumoFuenteValido(obtenidaEn) ||
		!pruebaValidaHasta.After(pruebaEmitidaEn) || consumidaEn.Before(pruebaEmitidaEn) ||
		obtenidaEn.Before(consumidaEn) || !obtenidaEn.Before(pruebaValidaHasta) ||
		pruebaEmitidaEn.Format(formatoInstanteReciboConsumoFuenteV2) != m.PruebaEmitidaEn ||
		pruebaValidaHasta.Format(formatoInstanteReciboConsumoFuenteV2) != m.PruebaValidaHasta ||
		consumidaEn.Format(formatoInstanteReciboConsumoFuenteV2) != m.ConsumidaEn ||
		obtenidaEn.Format(formatoInstanteReciboConsumoFuenteV2) != m.ObtenidaEn ||
		!rolesReciboConsumoFuenteDistintos(m) {
		return nuevoError("recibo_consumo_fuente", CodigoValorNoCanonico)
	}
	return nil
}

func (r ReciboConsumoAutorizacionFuenteV2) RepresentacionCanonicaV2() ([]byte, error) {
	if err := r.Validar(); err != nil {
		return nil, err
	}
	contenido, err := json.Marshal(r.material)
	if err != nil || len(contenido) == 0 || len(contenido) > maximoBytesRepresentacionV1 {
		return nil, nuevoError("recibo_consumo_fuente.representacion_canonica", CodigoFueraDeLimites)
	}
	return contenido, nil
}

func (r ReciboConsumoAutorizacionFuenteV2) HuellaSHA256V2() (string, error) {
	contenido, err := r.RepresentacionCanonicaV2()
	if err != nil {
		return "", err
	}
	return sha256Hex(contenido), nil
}

func restaurarReciboConsumoAutorizacionFuenteV2(
	contenido []byte,
) (ReciboConsumoAutorizacionFuenteV2, error) {
	var material materialReciboConsumoAutorizacionFuenteV2
	if err := decodificarJSONEstricto(contenido, &material); err != nil {
		return ReciboConsumoAutorizacionFuenteV2{}, err
	}
	if material.Esquema != EsquemaReciboConsumoAutorizacionFuenteV2 {
		return ReciboConsumoAutorizacionFuenteV2{},
			nuevoError("recibo_consumo_fuente.esquema", CodigoEsquemaIncompatible)
	}
	recibo := ReciboConsumoAutorizacionFuenteV2{material: material}
	canonico, err := recibo.RepresentacionCanonicaV2()
	if err != nil || !bytes.Equal(canonico, contenido) {
		return ReciboConsumoAutorizacionFuenteV2{},
			nuevoError("recibo_consumo_fuente.representacion_canonica", CodigoValorNoCanonico)
	}
	return recibo, nil
}

func RestaurarReciboConsumoAutorizacionFuenteV2ConHuellaSHA256(
	contenido []byte,
	huellaEsperada string,
) (ReciboConsumoAutorizacionFuenteV2, error) {
	if !huellaSHA256Valida(huellaEsperada) {
		return ReciboConsumoAutorizacionFuenteV2{},
			nuevoError("recibo_consumo_fuente.huella_esperada_sha256", CodigoValorNoCanonico)
	}
	recibo, err := restaurarReciboConsumoAutorizacionFuenteV2(contenido)
	if err != nil {
		return ReciboConsumoAutorizacionFuenteV2{}, err
	}
	huella, err := recibo.HuellaSHA256V2()
	if err != nil || subtle.ConstantTimeCompare([]byte(huella), []byte(huellaEsperada)) != 1 {
		return ReciboConsumoAutorizacionFuenteV2{},
			nuevoError("recibo_consumo_fuente.huella_esperada_sha256", CodigoHuellaNoCoincide)
	}
	return recibo, nil
}

func instanteReciboConsumoFuenteValido(instante time.Time) bool {
	return !instante.IsZero() && instante.Location() == time.UTC &&
		instante.Equal(instante.Truncate(time.Microsecond))
}

func rolesReciboConsumoFuenteDistintos(m materialReciboConsumoAutorizacionFuenteV2) bool {
	referencias := []string{m.Consumo.Referencia, m.DecisionRef, m.RecursoRef, m.CorrelacionRef,
		m.FuenteExacta.Referencia, m.Verificador.Referencia, m.ConsumoPrueba.Referencia,
		m.Auditoria.Referencia}
	vistas := make(map[string]struct{}, len(referencias))
	for _, referencia := range referencias {
		if _, existe := vistas[referencia]; existe {
			return false
		}
		vistas[referencia] = struct{}{}
	}
	return true
}
