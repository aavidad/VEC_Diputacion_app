package application

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestRectificarCoberturaComponeOrdenSegregadaConPredecesoraExacta(
	t *testing.T,
) {
	escenario := nuevoEscenarioRectificacionConfirmacionCobertura(t, true)
	recibo, err := escenario.servicio.Rectificar(
		context.Background(),
		escenario.solicitudRectificar,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, denegada := recibo.ResultadoDenegadoVEC(); !denegada ||
		escenario.vec.total() != 1 ||
		escenario.transaccion.total() != 1 {
		t.Fatalf("rectificación válida no alcanzó la Tx nominal: %+v", recibo)
	}
}

func TestRectificarCoberturaRechazaMismoActorAntesDeVEC(
	t *testing.T,
) {
	escenario := nuevoEscenarioRectificacionConfirmacionCobertura(t, false)
	_, err := escenario.servicio.Rectificar(
		context.Background(),
		escenario.solicitudRectificar,
	)
	if !errors.Is(err, ErrConfirmacionDecisionCoberturaEnConflicto) ||
		escenario.vec.total() != 0 ||
		escenario.transaccion.total() != 0 {
		t.Fatalf("el mismo actor rectificó su decisión: %v", err)
	}
}

func TestRectificarCoberturaRechazaPredecesoraObsoletaAntesDeVEC(
	t *testing.T,
) {
	escenario := nuevoEscenarioRectificacionConfirmacionCobertura(t, true)
	huellaAjena := strings.Repeat("e", 64)
	escenario.solicitudRectificar.PredecesoraHuella = huellaAjena
	escenario.solicitudRectificar.PredecesoraRef =
		"decision-cobertura:sha256:" + huellaAjena
	_, err := escenario.servicio.Rectificar(
		context.Background(),
		escenario.solicitudRectificar,
	)
	if !errors.Is(err, ErrConfirmacionDecisionCoberturaEnConflicto) ||
		escenario.vec.total() != 0 ||
		escenario.transaccion.total() != 0 {
		t.Fatalf("la predecesora obsoleta alcanzó VEC/Tx: %v", err)
	}
}
