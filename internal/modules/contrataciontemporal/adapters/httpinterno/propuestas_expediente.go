package httpinterno

import (
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// propuestasExpedienteJSON publica en el estado del seguimiento las
// propuestas de nombramiento del expediente (CT128): la vigente y las
// sustituidas por una no incorporación, en orden. Siempre una lista.
func propuestasExpedienteJSON(p []ports.EstadoPropuestaExpediente) []map[string]any {
	salida := make([]map[string]any, 0, len(p))
	for _, e := range p {
		m := map[string]any{"orden": e.Orden, "version_resultante": e.VersionResultante,
			"confirmada_en": e.ConfirmadaEn.UTC().Format(time.RFC3339Nano), "recibo_ref": e.ReciboRef,
			"vigente": e.Vigente, "sustitucion": nil}
		if s := e.Sustitucion; s != nil {
			m["sustitucion"] = map[string]string{"no_incorporacion_recibo_ref": s.NoIncorporacionReciboRef,
				"motivo_clave": s.MotivoClave, "registrada_en": s.RegistradaEn.UTC().Format(time.RFC3339Nano)}
		}
		salida = append(salida, m)
	}
	return salida
}
