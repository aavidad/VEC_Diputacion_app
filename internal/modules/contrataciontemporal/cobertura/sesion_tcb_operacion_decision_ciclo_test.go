package cobertura_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
)

func TestSesionTCBCallbackInfractorNoBloqueaIndefinidamenteNiPublica(
	t *testing.T,
) {
	escenario := nuevoEscenarioConfirmacionOrdenC3(t)
	entrada := make(chan struct{})
	continuar := make(chan struct{})
	porRetornar := make(chan struct{})
	resultadoCallback := make(chan error, 1)
	ejecutor := &ejecutorSesionTCBOperacionDecisionPrueba{
		recibo:                 datosReciboSesionTCBPrueba(escenario.reciboConcedido),
		callbackAsincrono:      true,
		retenerDespuesCallback: true,
		bloquearEnSesion:       "confirmar",
		entradaBloqueo:         entrada,
		continuarBloqueo:       continuar,
		ejecutorPorRetornar:    porRetornar,
		resultadoCallback:      resultadoCallback,
	}
	transaccion, err :=
		cobertura.NuevaTransaccionOperacionDecisionCoberturaTCB(ejecutor)
	if err != nil {
		t.Fatal(err)
	}
	type salida struct {
		resultado cobertura.ResultadoConfirmacionOperacionDecisionCobertura
		err       error
	}
	salidaOperacion := make(chan salida, 1)
	go func() {
		resultado, err := cobertura.ConfirmarOperacionDecisionCobertura(
			context.Background(),
			transaccion,
			escenario.ordenConcedida,
		)
		salidaOperacion <- salida{resultado: resultado, err: err}
	}()

	esperarSenalCicloSesionTCB(t, entrada, "entrada en Confirmar")
	esperarSenalCicloSesionTCB(t, porRetornar, "retorno del ejecutor")
	var salidaFinal salida
	select {
	case salidaFinal = <-salidaOperacion:
	case <-time.After(time.Second):
		t.Fatal("el wrapper quedó bloqueado por el callback infractor")
	}
	if !errors.Is(
		salidaFinal.err,
		cobertura.ErrEjecucionSesionTCBOperacionDecisionCoberturaNoDisponible,
	) {
		t.Fatalf("el callback pendiente no quedó ambiguo: %v", salidaFinal.err)
	}
	if _, err = salidaFinal.resultado.ReciboPara(
		escenario.ordenConcedida,
	); err == nil {
		t.Fatal("el callback pendiente publicó un recibo")
	}
	if err = ejecutor.ejecutarRetenido(); !errors.Is(
		err,
		cobertura.ErrSesionTCBOperacionDecisionCoberturaInvalida,
	) {
		t.Fatalf("el callback retenido sobrevivió al retorno: %v", err)
	}

	close(continuar)
	select {
	case errCallback := <-resultadoCallback:
		if !errors.Is(
			errCallback,
			cobertura.ErrSesionTCBOperacionDecisionCoberturaInvalida,
		) {
			t.Fatalf("el callback tardío conservó autoridad: %v", errCallback)
		}
	case <-time.After(time.Second):
		t.Fatal("el callback de prueba no terminó tras desbloquearlo")
	}
	sesion := ejecutor.ultimaSesion()
	pasosAntes, _, _, _ := sesion.instantanea()
	if err = ejecutor.ejecutarRetenido(); !errors.Is(
		err,
		cobertura.ErrSesionTCBOperacionDecisionCoberturaInvalida,
	) {
		t.Fatalf("el callback retenido volvió a ejecutarse: %v", err)
	}
	pasosDespues, _, _, _ := sesion.instantanea()
	if !reflect.DeepEqual(pasosAntes, pasosDespues) {
		t.Fatalf(
			"el callback escapado produjo pasos: antes=%v después=%v",
			pasosAntes,
			pasosDespues,
		)
	}
}

func esperarSenalCicloSesionTCB(
	t *testing.T,
	senal <-chan struct{},
	nombre string,
) {
	t.Helper()
	select {
	case <-senal:
	case <-time.After(time.Second):
		t.Fatalf("no llegó la señal de %s", nombre)
	}
}
