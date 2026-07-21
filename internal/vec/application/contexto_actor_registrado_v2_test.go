package application

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

func TestAutoridadContextoActorRegistradoV2ProyectaReciboMinimoExacto(t *testing.T) {
	solicitadoEn := instanteServicioContextoActorPrueba()
	resueltoEnDB := solicitadoEn.Add(2 * time.Millisecond)
	comprobadoEn := resueltoEnDB.Add(time.Millisecond)
	solicitud := solicitudServicioContextoActorPrueba()
	generador := nuevoGeneradorOperacionContextoActorV2Prueba()
	confirmacion := confirmacionRegistroContextoActorV2Prueba(
		t, contextoActorServicioPrueba(t, resueltoEnDB, solicitud), generador.ref,
	)
	resolutor := &resolutorRegistroContextoActorV2Prueba{resultado: confirmacion}
	servicio, err := NuevoServicioContextoActorProductivoV2(
		resolutor, generador,
		&relojSecuenciaContextoActorPrueba{instantes: []time.Time{solicitadoEn, comprobadoEn}},
	)
	if err != nil {
		t.Fatalf("crear servicio productivo: %v", err)
	}
	autoridad, err := NuevaAutoridadContextoActorRegistradoV2(servicio)
	if err != nil {
		t.Fatalf("crear autoridad: %v", err)
	}

	resultado, err := autoridad.ResolverContextoActorRegistradoV2(context.Background(), solicitud)
	if err != nil || resultado.Validar() != nil {
		t.Fatalf("resolver contexto registrado: resultado=%#v error=%v", resultado, err)
	}
	llamadas, recibida := resolutor.observacion()
	if llamadas != 1 || recibida.OperacionRef != generador.ref || recibida.Contexto != solicitud ||
		resultado.RegistroContextoRef != confirmacion.RegistroContextoRef ||
		!reflect.DeepEqual(resultado.Contexto, confirmacion.Contexto) ||
		!bytes.Equal(resultado.RepresentacionCanonica, confirmacion.RepresentacionCanonica) ||
		resultado.HuellaSHA256 != confirmacion.HuellaSHA256 ||
		!bytes.Equal(resultado.ManifiestoProcedenciaCanonico, confirmacion.ManifiestoProcedenciaCanonico) ||
		resultado.ManifiestoProcedenciaHuellaSHA256 != confirmacion.ManifiestoProcedenciaHuellaSHA256 ||
		resultado.AutoridadEfectiva != confirmacion.AutoridadEfectiva ||
		!resultado.ResueltoEnAutoritativo.Equal(confirmacion.ResueltoEnAutoritativo) {
		t.Fatalf("proyeccion no exacta: solicitud=%#v resultado=%#v", recibida, resultado)
	}
	if _, existe := reflect.TypeOf(resultado).FieldByName("OperacionRef"); existe {
		t.Fatal("la proyeccion expuso OperacionRef")
	}

	canonOriginal := append([]byte(nil), resultado.RepresentacionCanonica...)
	manifiestoOriginal := append([]byte(nil), resultado.ManifiestoProcedenciaCanonico...)
	confirmacion.RepresentacionCanonica[0] ^= 0xff
	confirmacion.ManifiestoProcedenciaCanonico[0] ^= 0xff
	resolutor.resultado.Contexto.Instantanea.Vinculos[0].Referencia =
		referenciaServicioContextoActorPrueba("can_", "x")
	if !bytes.Equal(resultado.RepresentacionCanonica, canonOriginal) ||
		!bytes.Equal(resultado.ManifiestoProcedenciaCanonico, manifiestoOriginal) ||
		resultado.Contexto.Instantanea.Vinculos[0].Referencia ==
			resolutor.resultado.Contexto.Instantanea.Vinculos[0].Referencia {
		t.Fatal("la proyeccion compartio memoria con el recibo durable")
	}
}

func TestNuevaAutoridadContextoActorRegistradoV2ExigeServicioProductivo(t *testing.T) {
	instante := instanteServicioContextoActorPrueba()
	legacy, err := NuevoServicioContextoActor(
		&fuenteContextoActorPrueba{}, &relojContextoActorPrueba{ahora: instante},
	)
	if err != nil {
		t.Fatalf("crear servicio heredado: %v", err)
	}
	for nombre, servicio := range map[string]*ServicioContextoActor{
		"nil": nil, "heredado": legacy,
	} {
		t.Run(nombre, func(t *testing.T) {
			autoridad, err := NuevaAutoridadContextoActorRegistradoV2(servicio)
			if autoridad != nil || !errors.Is(err, domain.ErrVinculoAutenticacionActorV2Invalido) {
				t.Fatalf("servicio no productivo aceptado: autoridad=%#v error=%v", autoridad, err)
			}
		})
	}
}

func TestAutoridadContextoActorRegistradoV2FallaCerradoAntesDelServicio(t *testing.T) {
	instante := instanteServicioContextoActorPrueba()
	resolutor := &resolutorRegistroContextoActorV2Prueba{}
	generador := nuevoGeneradorOperacionContextoActorV2Prueba()
	servicio, err := NuevoServicioContextoActorProductivoV2(
		resolutor, generador, &relojContextoActorPrueba{ahora: instante},
	)
	if err != nil {
		t.Fatalf("crear servicio productivo: %v", err)
	}
	autoridad, err := NuevaAutoridadContextoActorRegistradoV2(servicio)
	if err != nil {
		t.Fatalf("crear autoridad: %v", err)
	}
	cancelado, cancelar := context.WithCancel(context.Background())
	cancelar()
	casos := []struct {
		nombre    string
		ctx       context.Context
		solicitud domain.SolicitudContextoActor
	}{
		{"contexto nil", nil, solicitudServicioContextoActorPrueba()},
		{"contexto cancelado", cancelado, solicitudServicioContextoActorPrueba()},
		{"solicitud invalida", context.Background(), domain.SolicitudContextoActor{}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			resultado, err := autoridad.ResolverContextoActorRegistradoV2(caso.ctx, caso.solicitud)
			if resultado.Validar() == nil ||
				!errors.Is(err, domain.ErrVinculoAutenticacionActorV2Invalido) {
				t.Fatalf("precondicion invalida aceptada: resultado=%#v error=%v", resultado, err)
			}
		})
	}
	llamadas, _ := resolutor.observacion()
	if llamadas != 0 || generador.llamadas != 0 {
		t.Fatalf("servicio invocado con precondicion invalida: resolutor=%d generador=%d",
			llamadas, generador.llamadas)
	}

	var nula *AutoridadContextoActorRegistradoV2
	resultado, err := nula.ResolverContextoActorRegistradoV2(
		context.Background(), solicitudServicioContextoActorPrueba(),
	)
	if resultado.Validar() == nil || !errors.Is(err, domain.ErrVinculoAutenticacionActorV2Invalido) {
		t.Fatalf("autoridad nula tipada aceptada: resultado=%#v error=%v", resultado, err)
	}
}
