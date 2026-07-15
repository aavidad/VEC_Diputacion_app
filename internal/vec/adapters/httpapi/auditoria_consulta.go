package httpapi

import (
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"

	"vec-diputacion-granada/internal/vec/domain"
)

const maximoBytesReferenciaAuditoria = 512

// referenciaAuditoriaDesdeConsulta aplica una lista positiva al contrato HTTP:
// existe un unico parametro subject_ref con un unico valor exacto. El vacio,
// los parametros extra y los comodines nunca se reinterpretan como "todo".
func referenciaAuditoriaDesdeConsulta(consultaCruda string) (string, error) {
	valores, err := url.ParseQuery(consultaCruda)
	if err != nil || len(valores) != 1 {
		return "", domain.ErrPermissionDenied
	}
	referencias, existe := valores["subject_ref"]
	if !existe || len(referencias) != 1 || !referenciaAuditoriaCanonica(referencias[0]) {
		return "", domain.ErrPermissionDenied
	}
	return referencias[0], nil
}

func referenciaAuditoriaCanonica(referencia string) bool {
	if referencia == "" || referencia != strings.TrimSpace(referencia) ||
		len(referencia) > maximoBytesReferenciaAuditoria || !utf8.ValidString(referencia) ||
		strings.ContainsRune(referencia, '*') {
		return false
	}
	for _, caracter := range referencia {
		if unicode.IsControl(caracter) {
			return false
		}
	}
	return true
}
