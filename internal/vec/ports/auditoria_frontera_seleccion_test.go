package ports_test

import (
	"testing"

	"vec-diputacion-granada/internal/vec/ports"
)

func TestPuertoAuditoriaFronteraSeleccion(t *testing.T) {
	orden := ports.OrdenAuditoriaFronteraRutaExacta{
		CorrelacionRef: "corr_16816816816816816816816816816816",
		Motivo:         ports.MotivoAuditoriaFronteraRutaExactaAccesoDenegado,
		Superficie:     ports.SuperficieAuditoriaFronteraRutaExactaSeleccionPreparacionBases,
		Ruta:           "/api/vec/seleccion/preparacion-bases/guardar",
	}
	if err := orden.Validar(); err != nil {
		t.Fatal(err)
	}
}
