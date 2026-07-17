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
		referenciaNominalReciboConsumoFuenteValida(referencia.Referencia, prefijo)
}

// DecisionRef carece hoy de un tipo nominal VEC y VEC admite referencias
// historicas de formas distintas. Este contrato puede exigir su espacio de
// nombres y una gramatica cerrada, pero no fingir un UUID/entropia que el
// contrato global aun no garantiza. La procedencia se acredita con la huella
// canonica V2 y el consumo durable, no con la sintaxis del identificador.
func decisionRefReciboConsumoFuenteValida(valor string) bool {
	return referenciaNominalReciboConsumoFuenteValida(valor, "decision:")
}

func correlacionReciboConsumoFuenteValida(valor string) bool {
	return dominiovec.ReferenciaCorrelacionAutorizacionV2Valida(valor)
}

func recursoLecturaReciboConsumoFuenteValido(valor string) bool {
	const prefijo = "fuente:"
	return strings.HasPrefix(valor, prefijo) && huellaSHA256Valida(strings.TrimPrefix(valor, prefijo))
}

// ReferenciaReglasFuenteExactaV1Valida y sus variantes aplican en application
// los mismos perfiles nominales del recibo, sin abrir la gramatica a un
// prefijo elegido por el llamador.
func ReferenciaReglasFuenteExactaV1Valida(valor string) bool {
	return referenciaNominalReciboConsumoFuenteValida(valor, "reglas:")
}

func ReferenciaConvocatoriaFuenteExactaV1Valida(valor string) bool {
	return referenciaNominalReciboConsumoFuenteValida(valor, "convocatoria:")
}

func ReferenciaInstantaneaFuenteExactaV1Valida(valor string) bool {
	const prefijo = "iex_"
	return strings.HasPrefix(valor, prefijo) && huellaSHA256Valida(strings.TrimPrefix(valor, prefijo))
}

func referenciaNominalReciboConsumoFuenteValida(valor, prefijo string) bool {
	if len(valor) <= len(prefijo) || len(valor) > 512 || !strings.HasPrefix(valor, prefijo) {
		return false
	}
	sufijo := strings.TrimPrefix(valor, prefijo)
	if strings.Contains(sufijo, "..") || !caracterAlfanumericoASCII(rune(sufijo[0])) ||
		contieneDocumentoIdentidadEnSufijo(sufijo) {
		return false
	}
	for _, caracter := range sufijo {
		if !((caracter >= 'a' && caracter <= 'z') || (caracter >= '0' && caracter <= '9') ||
			caracter == ':' || caracter == '.' || caracter == '-' || caracter == '_') {
			return false
		}
	}
	return true
}

func contieneDocumentoIdentidadEnSufijo(sufijo string) bool {
	tokens := strings.FieldsFunc(sufijo, func(caracter rune) bool {
		return caracter == ':' || caracter == '.' || caracter == '-' || caracter == '_'
	})
	for _, token := range tokens {
		if dniNominalReciboValido(token) || nieNominalReciboValido(token) {
			return true
		}
	}
	return false
}

func dniNominalReciboValido(valor string) bool {
	if len(valor) != 9 {
		return false
	}
	numero := 0
	for indice := 0; indice < 8; indice++ {
		if valor[indice] < '0' || valor[indice] > '9' {
			return false
		}
		numero = numero*10 + int(valor[indice]-'0')
	}
	return letraDocumentoIdentidad(numero) == valor[8]
}

func nieNominalReciboValido(valor string) bool {
	if len(valor) != 9 || (valor[0] != 'x' && valor[0] != 'y' && valor[0] != 'z') {
		return false
	}
	numero := int(valor[0] - 'x')
	for indice := 1; indice < 8; indice++ {
		if valor[indice] < '0' || valor[indice] > '9' {
			return false
		}
		numero = numero*10 + int(valor[indice]-'0')
	}
	return letraDocumentoIdentidad(numero) == valor[8]
}

func letraDocumentoIdentidad(numero int) byte {
	const letras = "trwagmyfpdxbnjzsqvhlcke"
	if numero < 0 {
		return 0
	}
	return letras[numero%len(letras)]
}
