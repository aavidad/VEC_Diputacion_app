package calculoexperienciaoficial

import (
	"context"
	"errors"
	"strings"
	"testing"

	oficial "vec-diputacion-granada/internal/modules/bolsa/domain/calculoexperienciaoficial"
)

func TestCommitAmbiguoSeReconciliaSinPerderElRecibo(t *testing.T) {
	escenario := nuevoEscenarioServicioPrueba(t, perfilInternoAlto, false)
	escenario.confirmador.error = errors.New("driver: socket y detalle privado")
	escenario.confirmador.cometerYFallar = true
	salida, err := escenario.servicio.Ejecutar(context.Background(), escenario.orden)
	if err != nil {
		t.Fatal(err)
	}
	recibo, err := salida.Recibo()
	if err != nil || recibo.Validar() != nil || escenario.confirmador.llamadas != 1 ||
		escenario.confirmador.reconciliaciones != 1 {
		t.Fatal("el commit ambiguo no recupero su recibo durable")
	}
}

func TestFalloAmbiguoDevuelveCapacidadYPermiteReconciliarMismoIntento(t *testing.T) {
	escenario := nuevoEscenarioServicioPrueba(t, perfilInternoAlto, false)
	escenario.confirmador.error = errors.New("driver: secreto que no debe cruzar")
	escenario.confirmador.errorReconciliacion = errors.New("aun no visible")
	_, err := escenario.servicio.Ejecutar(context.Background(), escenario.orden)
	if !errors.Is(err, ErrResultadoConfirmacionIndeterminado) ||
		!errors.Is(err, ErrReconciliacionRequerida) ||
		strings.Contains(err.Error(), "driver") || strings.Contains(err.Error(), "secreto") {
		t.Fatalf("el fallo ambiguo no se normalizo: %v", err)
	}
	var indeterminado *ErrorConfirmacionIndeterminada
	if !errors.As(err, &indeterminado) {
		t.Fatal("no se devolvio la capacidad tipada de reconciliacion")
	}
	intento, err := indeterminado.Intento()
	if err != nil {
		t.Fatal(err)
	}
	referencia, err := intento.ReferenciaOpaca()
	esperada, _ := escenario.datosOrden.CorrelacionEscritura.ValorCanonico()
	if err != nil || referencia != esperada {
		t.Fatal("la referencia no fue emitida antes de confirmar")
	}
	escenario.confirmador.ultimoResultado = resultadoReconciliablePrueba(
		t, escenario.confirmador.ultimaSolicitud,
	)
	escenario.confirmador.errorReconciliacion = nil
	salida, err := escenario.servicio.Reconciliar(context.Background(), intento)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := salida.Recibo(); err != nil || escenario.confirmador.llamadas != 1 ||
		escenario.confirmador.reconciliaciones != 2 {
		t.Fatal("reconciliar repitio Confirmar o perdio el recibo")
	}
}

func TestReconciliacionRechazaReciboAjenoYNoFiltraNoEncontrado(t *testing.T) {
	propio := nuevoEscenarioServicioPrueba(t, perfilInternoAlto, false)
	propio.confirmador.error = errors.New("fallo ambiguo")
	propio.confirmador.errorReconciliacion = errors.New("almacen: fila privada no encontrada")
	_, err := propio.servicio.Ejecutar(context.Background(), propio.orden)
	var indeterminado *ErrorConfirmacionIndeterminada
	if !errors.As(err, &indeterminado) {
		t.Fatal(err)
	}
	intento, _ := indeterminado.Intento()

	_, err = propio.servicio.Reconciliar(context.Background(), intento)
	if !errors.Is(err, ErrResultadoConfirmacionIndeterminado) ||
		strings.Contains(err.Error(), "fila privada") {
		t.Fatalf("no encontrado no se expurgo: %v", err)
	}

	ajeno := nuevoEscenarioServicioPrueba(t, perfilInternoAlto, true)
	if _, err := ajeno.servicio.Ejecutar(context.Background(), ajeno.orden); err != nil {
		t.Fatal(err)
	}
	propio.confirmador.errorReconciliacion = nil
	propio.confirmador.ultimoResultado = ajeno.confirmador.ultimoResultado
	_, err = propio.servicio.Reconciliar(context.Background(), intento)
	if !errors.Is(err, ErrResultadoConfirmacionIndeterminado) {
		t.Fatalf("se acepto un recibo de otra intencion: %v", err)
	}
	if propio.confirmador.llamadas != 1 {
		t.Fatal("el rechazo IDOR repitio la confirmacion")
	}
}

func TestReconciliacionRechazaCapacidadCeroODeOtroPerfil(t *testing.T) {
	interno := nuevoEscenarioServicioPrueba(t, perfilInternoAlto, false)
	if _, err := interno.servicio.Reconciliar(
		context.Background(), IntentoReconciliacionCalculoOficial{},
	); !errors.Is(err, ErrServicioInvalido) {
		t.Fatalf("capacidad cero aceptada: %v", err)
	}
	interno.confirmador.error = errors.New("ambiguo")
	_, err := interno.servicio.Ejecutar(context.Background(), interno.orden)
	var indeterminado *ErrorConfirmacionIndeterminada
	if !errors.As(err, &indeterminado) {
		t.Fatal(err)
	}
	intento, _ := indeterminado.Intento()
	externo := nuevoEscenarioServicioPrueba(t, perfilExternoOrdinario, false)
	if _, err := externo.servicio.Reconciliar(context.Background(), intento); !errors.Is(
		err, ErrServicioInvalido,
	) || externo.confirmador.reconciliaciones != 0 {
		t.Fatalf("otro perfil acepto la capacidad: %v", err)
	}
}

func resultadoReconciliablePrueba(
	t *testing.T,
	solicitud SolicitudConfirmacionDuradera,
) ResultadoConfirmacionDuradera {
	t.Helper()
	datos, err := solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	indice := strings.Repeat("8", 64)
	recibo, err := oficial.NuevoReciboV1(
		"recibo:calculo:reconciliado:1", 1, indice, datos.Intencion,
	)
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := NuevoResultadoConfirmacionDuradera(
		datos.ReferenciaIntento, recibo, indice, datos.HuellaResultadoSHA256,
		ConfirmacionCreada,
	)
	if err != nil {
		t.Fatal(err)
	}
	return resultado
}
