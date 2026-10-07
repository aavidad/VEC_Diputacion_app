package domain

import "strings"

// ModuloReferenciaExpedienteV1 resuelve únicamente las referencias tipadas
// publicadas para expedientes. La versión 1 del catálogo contiene ct, con un
// identificador hexadecimal minúsculo de 64 caracteres. No concede acceso.
func ModuloReferenciaExpedienteV1(referencia string) string {
	const prefijo = "expediente:ct:"
	if len(referencia) != len(prefijo)+64 || !strings.HasPrefix(referencia, prefijo) {
		return ""
	}
	for i := len(prefijo); i < len(referencia); i++ {
		c := referencia[i]
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return ""
		}
	}
	return "contratacion_temporal"
}
