package calculoexperienciaoficial

import (
	"strings"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func referenciaExactaNominalReciboValida(
	referencia ReferenciaExactaV1,
	prefijo string,
) bool {
	return validarReferencia(referencia, "recibo_consumo_fuente.referencia") == nil &&
		referenciaOpacaAcuñadaReciboValida(referencia.Referencia, prefijo)
}

func decisionRefReciboConsumoFuenteValida(valor string) bool {
	return referenciaOpacaAcuñadaReciboValida(valor, "decision:")
}

func correlacionReciboConsumoFuenteValida(valor string) bool {
	return dominiovec.ReferenciaCorrelacionAutorizacionV2Valida(valor)
}

func recursoLecturaReciboConsumoFuenteValido(valor string) bool {
	return referenciaOpacaAcuñadaReciboValida(valor, "fuente:")
}

// ReferenciaReglasFuenteExactaV2Valida y sus variantes aplican en application
// los mismos perfiles nominales del recibo, sin abrir la gramatica a un
// prefijo elegido por el llamador.
func ReferenciaReglasFuenteExactaV2Valida(valor string) bool {
	return referenciaOpacaAcuñadaReciboValida(valor, "reglas:")
}

func ReferenciaConvocatoriaFuenteExactaV2Valida(valor string) bool {
	return referenciaOpacaAcuñadaReciboValida(valor, "convocatoria:")
}

func ReferenciaInstantaneaFuenteExactaV2Valida(valor string) bool {
	return referenciaOpacaAcuñadaReciboValida(valor, "iex_")
}

func ReferenciaEvidenciaFuenteExactaV2Valida(valor string) bool {
	return referenciaOpacaAcuñadaReciboValida(valor, "evidencia:fuente:")
}

func ReferenciaVerificadorFuenteExactaV2Valida(valor string) bool {
	return referenciaOpacaAcuñadaReciboValida(valor, "verificador:fuente:")
}

func ReferenciaConsumoPruebaFuenteExactaV2Valida(valor string) bool {
	return referenciaOpacaAcuñadaReciboValida(valor, "consumo:prueba:")
}

func ReferenciaAuditoriaFuenteExactaV2Valida(valor string) bool {
	return referenciaOpacaAcuñadaReciboValida(valor, "auditoria:fuente:")
}

func referenciaOpacaAcuñadaReciboValida(valor, prefijo string) bool {
	return strings.HasPrefix(valor, prefijo) && len(valor) == len(prefijo)+64 &&
		huellaSHA256Valida(strings.TrimPrefix(valor, prefijo))
}
