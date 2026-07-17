package calculoexperienciaoficial

import (
	"bytes"
	"crypto/subtle"
	"encoding/json"
	"time"
)

const (
	EsquemaReciboConsumoAutorizacionFuenteV1 = "vec.bolsa.calculo-experiencia-oficial.recibo-consumo-autorizacion-fuente.v1"
	formatoInstanteReciboConsumoFuenteV1     = "2006-01-02T15:04:05.000000Z"
)

// ReciboConsumoAutorizacionFuenteV1 es la atestacion opaca que devuelve una
// fuente durable tras consumir una decision de lectura. No concede autoridad:
// permite cotejar que la decision, el selector y los artefactos leidos fueron
// exactamente los mismos que intervienen en el calculo.
type ReciboConsumoAutorizacionFuenteV1 struct {
	material materialReciboConsumoAutorizacionFuenteV1
}

type materialReciboConsumoAutorizacionFuenteV1 struct {
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
}

// NuevoReciboConsumoAutorizacionFuenteV1 no acepta un DTO libre para evitar
// reconstrucciones parciales en fronteras de transporte.
func NuevoReciboConsumoAutorizacionFuenteV1(
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
	consumidaEn time.Time,
) (ReciboConsumoAutorizacionFuenteV1, error) {
	// No normalizar silenciosamente zona ni precision: los instantes deben
	// venir ya canonizados desde la frontera durable que acredita el consumo.
	if !instanteReciboConsumoFuenteValido(pruebaEmitidaEn) ||
		!instanteReciboConsumoFuenteValido(pruebaValidaHasta) ||
		!instanteReciboConsumoFuenteValido(consumidaEn) {
		return ReciboConsumoAutorizacionFuenteV1{},
			nuevoError("recibo_consumo_fuente.instantes", CodigoValorNoCanonico)
	}
	recibo := ReciboConsumoAutorizacionFuenteV1{material: materialReciboConsumoAutorizacionFuenteV1{
		Esquema: EsquemaReciboConsumoAutorizacionFuenteV1,
		Consumo: consumo, DecisionRef: decisionRef,
		EsquemaHuellaDecision: esquemaHuellaDecision,
		HuellaDecisionSHA256:  huellaDecisionSHA256,
		RecursoRef:            recursoRef, HuellaContextoRecursoSHA256: huellaContextoRecursoSHA256,
		CorrelacionRef: correlacionRef, HuellaSelectorSHA256: huellaSelectorSHA256,
		HuellaEntradaSHA256: huellaEntradaSHA256,
		FuenteExacta:        fuenteExacta, Verificador: verificador,
		ConsumoPrueba: consumoPrueba, Auditoria: auditoria,
		PruebaEmitidaEn:   pruebaEmitidaEn.Format(formatoInstanteReciboConsumoFuenteV1),
		PruebaValidaHasta: pruebaValidaHasta.Format(formatoInstanteReciboConsumoFuenteV1),
		ConsumidaEn:       consumidaEn.Format(formatoInstanteReciboConsumoFuenteV1),
	}}
	if err := recibo.Validar(); err != nil {
		return ReciboConsumoAutorizacionFuenteV1{}, err
	}
	return recibo, nil
}

func (r ReciboConsumoAutorizacionFuenteV1) Validar() error {
	m := r.material
	pruebaEmitidaEn, errEmitida := time.Parse(
		formatoInstanteReciboConsumoFuenteV1, m.PruebaEmitidaEn,
	)
	pruebaValidaHasta, errValidez := time.Parse(
		formatoInstanteReciboConsumoFuenteV1, m.PruebaValidaHasta,
	)
	consumidaEn, err := time.Parse(formatoInstanteReciboConsumoFuenteV1, m.ConsumidaEn)
	if m.Esquema != EsquemaReciboConsumoAutorizacionFuenteV1 ||
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
		errEmitida != nil || errValidez != nil || err != nil ||
		!instanteReciboConsumoFuenteValido(pruebaEmitidaEn) ||
		!instanteReciboConsumoFuenteValido(pruebaValidaHasta) ||
		!instanteReciboConsumoFuenteValido(consumidaEn) ||
		!pruebaValidaHasta.After(pruebaEmitidaEn) || consumidaEn.Before(pruebaEmitidaEn) ||
		!consumidaEn.Before(pruebaValidaHasta) ||
		pruebaEmitidaEn.Format(formatoInstanteReciboConsumoFuenteV1) != m.PruebaEmitidaEn ||
		pruebaValidaHasta.Format(formatoInstanteReciboConsumoFuenteV1) != m.PruebaValidaHasta ||
		consumidaEn.Format(formatoInstanteReciboConsumoFuenteV1) != m.ConsumidaEn ||
		!rolesReciboConsumoFuenteDistintos(m) {
		return nuevoError("recibo_consumo_fuente", CodigoValorNoCanonico)
	}
	return nil
}

func (r ReciboConsumoAutorizacionFuenteV1) RepresentacionCanonicaV1() ([]byte, error) {
	if err := r.Validar(); err != nil {
		return nil, err
	}
	contenido, err := json.Marshal(r.material)
	if err != nil || len(contenido) == 0 || len(contenido) > maximoBytesRepresentacionV1 {
		return nil, nuevoError("recibo_consumo_fuente.representacion_canonica", CodigoFueraDeLimites)
	}
	return contenido, nil
}

func (r ReciboConsumoAutorizacionFuenteV1) HuellaSHA256V1() (string, error) {
	contenido, err := r.RepresentacionCanonicaV1()
	if err != nil {
		return "", err
	}
	return sha256Hex(contenido), nil
}

func restaurarReciboConsumoAutorizacionFuenteV1(
	contenido []byte,
) (ReciboConsumoAutorizacionFuenteV1, error) {
	var material materialReciboConsumoAutorizacionFuenteV1
	if err := decodificarJSONEstricto(contenido, &material); err != nil {
		return ReciboConsumoAutorizacionFuenteV1{}, err
	}
	if material.Esquema != EsquemaReciboConsumoAutorizacionFuenteV1 {
		return ReciboConsumoAutorizacionFuenteV1{},
			nuevoError("recibo_consumo_fuente.esquema", CodigoEsquemaIncompatible)
	}
	recibo := ReciboConsumoAutorizacionFuenteV1{material: material}
	canonico, err := recibo.RepresentacionCanonicaV1()
	if err != nil || !bytes.Equal(canonico, contenido) {
		return ReciboConsumoAutorizacionFuenteV1{},
			nuevoError("recibo_consumo_fuente.representacion_canonica", CodigoValorNoCanonico)
	}
	return recibo, nil
}

func RestaurarReciboConsumoAutorizacionFuenteV1ConHuellaSHA256(
	contenido []byte,
	huellaEsperada string,
) (ReciboConsumoAutorizacionFuenteV1, error) {
	if !huellaSHA256Valida(huellaEsperada) {
		return ReciboConsumoAutorizacionFuenteV1{},
			nuevoError("recibo_consumo_fuente.huella_esperada_sha256", CodigoValorNoCanonico)
	}
	recibo, err := restaurarReciboConsumoAutorizacionFuenteV1(contenido)
	if err != nil {
		return ReciboConsumoAutorizacionFuenteV1{}, err
	}
	huella, err := recibo.HuellaSHA256V1()
	if err != nil || subtle.ConstantTimeCompare([]byte(huella), []byte(huellaEsperada)) != 1 {
		return ReciboConsumoAutorizacionFuenteV1{},
			nuevoError("recibo_consumo_fuente.huella_esperada_sha256", CodigoHuellaNoCoincide)
	}
	return recibo, nil
}

func instanteReciboConsumoFuenteValido(instante time.Time) bool {
	return !instante.IsZero() && instante.Location() == time.UTC &&
		instante.Equal(instante.Truncate(time.Microsecond))
}

func rolesReciboConsumoFuenteDistintos(m materialReciboConsumoAutorizacionFuenteV1) bool {
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
