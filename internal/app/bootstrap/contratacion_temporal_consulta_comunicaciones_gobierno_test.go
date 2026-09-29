package bootstrap

import (
	"slices"
	"testing"

	postgrescontratacion "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
)

// Sin esta audiencia en el gobierno publicado, el arranque rechaza la consulta
// de comunicaciones del expediente (CT140) y el servidor no arranca.
func TestGobiernoCTIncluyeConsultaComunicacionesExpediente(t *testing.T) {
	if !slices.Contains(audienciasConsumoGobiernoCTDesarrollo(), postgrescontratacion.AudienciaConsultaComunicacionesExpediente) {
		t.Fatal("falta la audiencia de consulta de comunicaciones del expediente en el gobierno CT")
	}
}
