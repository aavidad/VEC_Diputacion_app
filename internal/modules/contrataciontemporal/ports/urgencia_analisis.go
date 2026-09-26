package ports

import (
	"unicode/utf8"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

// MaximoCaracteresMotivoUrgenciaAnalisis acota el motivo de la urgencia que
// RRHH declara al analizar (CT-000125 exige el mismo límite).
const MaximoCaracteresMotivoUrgenciaAnalisis = 1000

// marcaUrgenciaIdempotencia separa la urgencia de las observaciones en la
// identidad semántica de la operación.
const marcaUrgenciaIdempotencia = "vec.contratacion-temporal.analisis.urgencia.v1"

// MotivoUrgenciaAnalisisValido admite el vacío (sin urgencia) o un texto
// limpio de hasta mil caracteres con las mismas reglas que las observaciones.
func MotivoUrgenciaAnalisisValido(motivo string) bool {
	return motivo == "" || (domain.ObservacionesAnalisisValidas(motivo) &&
		utf8.RuneCountInString(motivo) <= MaximoCaracteresMotivoUrgenciaAnalisis)
}
