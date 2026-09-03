package application

import (
	"context"
	"errors"
	"testing"
)

func TestRegistroSolicitudRespetaCancelacionPrevia(t *testing.T) {
	escenario := nuevoEscenarioRegistro(t)
	servicio, d := construirServicioRegistro(t, escenario)
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()

	_, err := servicio.Registrar(ctx, escenario.solicitud)
	if !errors.Is(err, context.Canceled) || d.contextos.llamadas != 0 ||
		d.autorizador.llamadas != 0 || d.candidaturas.llamadas != 0 ||
		d.transaccion.llamadas != 0 {
		t.Fatalf("cancelación previa produjo trabajo: %v", err)
	}
}

func TestRegistroSolicitudCancelaTrasResolverMotivo(t *testing.T) {
	escenario := nuevoEscenarioRegistro(t)
	servicio, d := construirServicioRegistro(t, escenario)
	ctx, cancelar := context.WithCancel(context.Background())
	d.motivos.despues = cancelar

	_, err := servicio.Registrar(ctx, escenario.solicitud)
	if !errors.Is(err, context.Canceled) || d.correlaciones.llamadas != 0 ||
		d.autorizador.llamadas != 0 || d.candidaturas.llamadas != 1 ||
		d.transaccion.llamadas != 0 {
		t.Fatalf("cancelación tras motivo cruzó la frontera: %v", err)
	}
}

func TestRegistroSolicitudCancelaTrasGenerarCorrelacion(t *testing.T) {
	escenario := nuevoEscenarioRegistro(t)
	servicio, d := construirServicioRegistro(t, escenario)
	ctx, cancelar := context.WithCancel(context.Background())
	d.correlaciones.despues = cancelar

	_, err := servicio.Registrar(ctx, escenario.solicitud)
	if !errors.Is(err, context.Canceled) || d.correlaciones.llamadas != 1 ||
		d.autorizador.llamadas != 0 || d.candidaturas.llamadas != 1 ||
		d.transaccion.llamadas != 0 {
		t.Fatalf("cancelación tras correlación alcanzó PDP o efecto: %v", err)
	}
}

func TestRegistroSolicitudCancelaJustoAntesDeConfirmarAlta(t *testing.T) {
	escenario := nuevoEscenarioRegistro(t)
	servicio, d := construirServicioRegistro(t, escenario)
	ctx, cancelar := context.WithCancel(context.Background())
	d.reloj.despues = func(llamada int) {
		if llamada == 5 {
			cancelar()
		}
	}

	_, err := servicio.Registrar(ctx, escenario.solicitud)
	if !errors.Is(err, context.Canceled) || d.candidaturas.llamadas != 1 ||
		d.transaccion.llamadas != 0 {
		t.Fatalf("cancelación previa a ConfirmarAlta produjo efecto: %v", err)
	}
}

func TestRegistroSolicitudNoConvierteCommitConfirmadoEnCancelacionTardia(t *testing.T) {
	escenario := nuevoEscenarioRegistro(t)
	servicio, d := construirServicioRegistro(t, escenario)
	ctx, cancelar := context.WithCancel(context.Background())
	d.transaccion.despues = cancelar

	recibo, err := servicio.Registrar(ctx, escenario.solicitud)
	if err != nil || recibo != escenario.recibo ||
		!errors.Is(ctx.Err(), context.Canceled) {
		t.Fatalf("éxito durable convertido en fallo: recibo=%#v err=%v", recibo, err)
	}
}
