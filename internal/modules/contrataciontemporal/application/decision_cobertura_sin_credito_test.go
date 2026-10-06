package application

import (
	"context"
	"errors"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

// Sin crédito no se ofrece: tras autorizar la lectura, la propuesta de
// cobertura no se construye y el motivo llega a la frontera sin gastar la
// preparación ni consultar el gobierno de la operación.
func TestProponerCoberturaSinCreditoExplicaElMotivoSinEfectos(t *testing.T) {
	escenario := nuevoEscenarioPresentacionCobertura(t, viasPresentacionCoberturaPrueba(2))
	expediente := escenario.analisis.expediente.Clonar()
	expediente.Analisis.ValidacionRC.Resultado = domain.RCRechazada
	if expediente.Validar() != nil {
		t.Fatal("el expediente de prueba con la retención rechazada debe ser íntegro")
	}
	escenario.analisis.expediente = expediente
	_, err := escenario.servicio.Proponer(context.Background(), escenario.solicitud)
	motivo, ok := domain.MotivoSinCreditoDe(err)
	if !errors.Is(err, ErrPresentacionPropuestaCoberturaEstadoNoAdmite) ||
		!ok || motivo != domain.SinCreditoRetencionRechazada {
		t.Fatalf("sin crédito debía explicar el motivo: %v", err)
	}
	if escenario.accesos.total() == 0 || escenario.gobierno.total() != 0 ||
		escenario.global.generador.llamadas() != 0 {
		t.Fatal("sin crédito no debe consultar el gobierno ni preparar la propuesta")
	}
	exigirCeroConsumoPreparacionGlobal(t, escenario.global)
}

// La decisión aún no ha autorizado el expediente cuando lee la instantánea:
// no revela el motivo de crédito y responde como antes, sin disponibilidad.
func TestConfirmacionSinCreditoNoRevelaElMotivoAntesDeAutorizar(t *testing.T) {
	servicio := &ServicioConfirmacionDecisionCobertura{}
	err := servicio.errorDependencia(context.Background(),
		errors.Join(errors.New("instantánea"), domain.NuevoErrorSinCredito(domain.SinCreditoRetencionRechazada)))
	if _, ok := domain.MotivoSinCreditoDe(err); ok || !errors.Is(err, ErrConfirmacionDecisionCoberturaNoDisponible) {
		t.Fatalf("la decisión reveló el motivo de crédito: %v", err)
	}
	// Una causa ajena no se convierte en motivo de crédito.
	if _, ok := errorSinCreditoCobertura(ErrConfirmacionDecisionCoberturaEnConflicto, errors.New("otra")); ok {
		t.Fatal("error ajeno tratado como falta de crédito")
	}
}
