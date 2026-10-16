package application

import (
	"context"
	"errors"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestFalloSeleccionConservaDiagnosticoOfrecimiento(t *testing.T) {
	causa := &ports.ErrorCriteriosOfrecimientoPendientesRRHH{Comprobacion: ports.ComprobacionOfrecimientoPendiente{
		BolsaRef: "bolsa:prueba", OrdenActa: 2, VersionPoliticaOfertas: 3,
		MotivoClave: "pendiente_comprobacion_rrhh",
	}}
	resultado := normalizarFalloSeleccionLlamamiento(context.Background(), causa)
	var recuperada *ports.ErrorCriteriosOfrecimientoPendientesRRHH
	if !errors.Is(resultado, ports.ErrCriteriosOfrecimientoPendientesRRHH) ||
		!errors.As(resultado, &recuperada) || recuperada.Comprobacion != causa.Comprobacion {
		t.Fatalf("normalización perdió causa auditada: %v", resultado)
	}
}
