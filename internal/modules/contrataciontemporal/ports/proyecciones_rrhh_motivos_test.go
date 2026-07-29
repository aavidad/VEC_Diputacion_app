package ports_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func TestResolutorMotivoConsultaRRHHTieneDosMetodosNominales(t *testing.T) {
	t.Parallel()

	contrato := reflect.TypeOf((*ports.ResolutorMotivoConsultaRRHH)(nil)).Elem()
	metodos := []string{
		"ResolverMotivoCuadroRRHH",
		"ResolverMotivoDetalleRRHH",
	}

	if contrato.NumMethod() != len(metodos) {
		t.Fatalf(
			"métodos del puerto = %d; se esperaban %d",
			contrato.NumMethod(),
			len(metodos),
		)
	}
	for _, nombre := range metodos {
		_, existe := contrato.MethodByName(nombre)
		if !existe {
			t.Fatalf("falta el método nominal %s", nombre)
		}
	}
}

func TestResolutorMotivoConsultaRRHHNoAdmiteSelectoresLibres(t *testing.T) {
	t.Parallel()

	contrato := reflect.TypeOf((*ports.ResolutorMotivoConsultaRRHH)(nil)).Elem()
	tipoContexto := reflect.TypeOf((*context.Context)(nil)).Elem()
	tipoInstante := reflect.TypeOf(time.Time{})
	tipoReferencia := reflect.TypeOf(dominiovec.ReferenciaEntradaCatalogo{})
	tipoError := reflect.TypeOf((*error)(nil)).Elem()

	for indice := 0; indice < contrato.NumMethod(); indice++ {
		metodo := contrato.Method(indice)
		if metodo.Type.NumIn() != 2 ||
			metodo.Type.In(0) != tipoContexto ||
			metodo.Type.In(1) != tipoInstante {
			t.Fatalf("%s admite entradas ajenas a contexto e instante: %v", metodo.Name, metodo.Type)
		}
		if metodo.Type.NumOut() != 2 ||
			metodo.Type.Out(0) != tipoReferencia ||
			metodo.Type.Out(1) != tipoError {
			t.Fatalf("%s expone una salida distinta de referencia y error: %v", metodo.Name, metodo.Type)
		}
	}
}

func TestMotivoConsultaRRHHFallaConReferenciaCeroYErrorOpaco(t *testing.T) {
	t.Parallel()

	if dominiovec.ReferenciaMotivoAutorizacionV2Valida(
		dominiovec.ReferenciaEntradaCatalogo{},
	) {
		t.Fatal("la referencia cero se interpretó como motivo de autorización")
	}
	if ports.ErrMotivoConsultaRRHHNoDisponible == nil {
		t.Fatal("el contrato perdió su único centinela público")
	}
	envuelto := errors.Join(
		ports.ErrMotivoConsultaRRHHNoDisponible,
		context.Canceled,
	)
	if !errors.Is(envuelto, ports.ErrMotivoConsultaRRHHNoDisponible) ||
		!errors.Is(envuelto, context.Canceled) {
		t.Fatal("el centinela no conserva cierre y cancelación mediante errors.Is")
	}
}
