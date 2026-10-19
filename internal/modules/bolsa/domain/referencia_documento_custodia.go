package domain

import "strings"

// ReferenciaDocumentoCustodiaValida es el espejo en Go de
// vec_bolsa_llamamientos.b77_documento_ref_opaco_v1: la referencia legible
// que RRHH da a un documento que sigue en su custodia (por ejemplo, un número
// de registro). Admite entre 3 y 256 caracteres sin espacios y rechaza
// etiquetas o números de documento de identidad; las referencias propias del
// sistema (espacio de nombres y SHA-256) quedan fuera del filtro numérico.
func ReferenciaDocumentoCustodiaValida(v string) bool {
	return len(v) >= 3 && len(v) <= 256 && referenciaLlamamientoOpacaValida(v)
}

// ReferenciaEventoContratoParticipacion indica si v tiene la forma exacta del
// evento de contrato que Bolsa recibe de Contratación temporal. Su huella
// hexadecimal puede casar por azar con el patrón de DNI y su espacio de
// nombres lleva guion, así que no pasa por ReferenciaPropiaSistema.
func ReferenciaEventoContratoParticipacion(v string) bool {
	huella, ok := strings.CutPrefix(v, prefijoEventoContratoParticipacion)
	return ok && patronHuellaSHA256.MatchString(huella)
}
