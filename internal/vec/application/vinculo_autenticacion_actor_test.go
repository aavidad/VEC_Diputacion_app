package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type revalidadorVinculoAplicacionAdversarial struct {
	resultado    domain.AutenticacionRevalidadaV1
	err          error
	invocaciones int
	despues      func()
}

func (r *revalidadorVinculoAplicacionAdversarial) RevalidarAutenticacionActorV1(
	context.Context,
	domain.SolicitudRevalidacionAutenticacionActorV1,
) (domain.AutenticacionRevalidadaV1, error) {
	r.invocaciones++
	if r.despues != nil {
		r.despues()
	}
	return r.resultado, r.err
}

func TestServicioVinculoAutenticacionActorV1SoloEmiteTrasRevalidar(t *testing.T) {
	ahora := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)
	actor, vinculoEsperado := contextoYVinculoAutenticacionAplicacionPrueba(ahora)
	datosEsperados, _ := vinculoEsperado.Datos()
	revalidador := &revalidadorVinculoAplicacionAdversarial{resultado: datosEsperados.Autenticacion()}
	servicio, err := NuevoServicioVinculoAutenticacionActorV1(
		revalidador, &relojAutorizacionServicioPrueba{ahora: ahora},
	)
	if err != nil {
		t.Fatalf("crear servicio: %v", err)
	}
	vinculo, err := servicio.Crear(context.Background(), domain.SolicitudRevalidacionAutenticacionActorV1{
		AutenticacionRef: datosEsperados.AutenticacionRef, SesionRef: datosEsperados.SesionRef,
	}, actor)
	if err != nil || revalidador.invocaciones != 1 || vinculo.ValidarPara(actor) != nil {
		t.Fatalf("vinculo no emitido: invocaciones=%d error=%v", revalidador.invocaciones, err)
	}
}

func TestServicioVinculoAutenticacionActorV1FallaCerrado(t *testing.T) {
	ahora := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)
	actor, base := contextoYVinculoAutenticacionAplicacionPrueba(ahora)
	datos, _ := base.Datos()
	solicitud := domain.SolicitudRevalidacionAutenticacionActorV1{
		AutenticacionRef: datos.AutenticacionRef, SesionRef: datos.SesionRef,
	}
	falloFuente := errors.New("fuente no disponible")
	casos := []struct {
		nombre      string
		ctx         context.Context
		revalidador *revalidadorVinculoAplicacionAdversarial
		actor       domain.ContextoActor
		solicitud   domain.SolicitudRevalidacionAutenticacionActorV1
	}{
		{"contexto nulo", nil, &revalidadorVinculoAplicacionAdversarial{resultado: datos.Autenticacion()}, actor, solicitud},
		{"contexto cancelado", contextoCanceladoAutorizacionPrueba(), &revalidadorVinculoAplicacionAdversarial{resultado: datos.Autenticacion()}, actor, solicitud},
		{"fuente falla", context.Background(), &revalidadorVinculoAplicacionAdversarial{err: falloFuente}, actor, solicitud},
		{"sesion cruzada", context.Background(), &revalidadorVinculoAplicacionAdversarial{resultado: datos.Autenticacion()}, actor,
			domain.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: datos.AutenticacionRef, SesionRef: "ses_otra234567890abcdefghijkl"}},
		{"actor cero", context.Background(), &revalidadorVinculoAplicacionAdversarial{resultado: datos.Autenticacion()}, domain.ContextoActor{}, solicitud},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			servicio, err := NuevoServicioVinculoAutenticacionActorV1(
				caso.revalidador, &relojAutorizacionServicioPrueba{ahora: ahora},
			)
			if err != nil {
				t.Fatalf("constructor: %v", err)
			}
			vinculo, err := servicio.Crear(caso.ctx, caso.solicitud, caso.actor)
			if err == nil || !errors.Is(err, domain.ErrAutorizacionDenegada) || vinculo.Validar() == nil {
				t.Fatalf("caso aceptado: vinculo=%v err=%v", vinculo, err)
			}
			if (caso.ctx == nil || caso.ctx.Err() != nil || caso.actor.Validar() != nil) && caso.revalidador.invocaciones != 0 {
				t.Fatalf("se consulto fuente con precondicion invalida: %d", caso.revalidador.invocaciones)
			}
		})
	}
}

func TestServicioVinculoAutenticacionActorV1RecompruebaCancelacion(t *testing.T) {
	ahora := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)
	actor, base := contextoYVinculoAutenticacionAplicacionPrueba(ahora)
	datos, _ := base.Datos()
	ctx, cancelar := context.WithCancel(context.Background())
	revalidador := &revalidadorVinculoAplicacionAdversarial{
		resultado: datos.Autenticacion(), despues: cancelar,
	}
	servicio, err := NuevoServicioVinculoAutenticacionActorV1(
		revalidador, &relojAutorizacionServicioPrueba{ahora: ahora},
	)
	if err != nil {
		t.Fatalf("constructor: %v", err)
	}
	vinculo, err := servicio.Crear(ctx, domain.SolicitudRevalidacionAutenticacionActorV1{
		AutenticacionRef: datos.AutenticacionRef, SesionRef: datos.SesionRef,
	}, actor)
	if !errors.Is(err, context.Canceled) || vinculo.Validar() == nil {
		t.Fatalf("cancelacion concurrente no cerro emision: %v", err)
	}
}

func TestNuevoServicioVinculoAutenticacionActorV1RechazaNulosTipados(t *testing.T) {
	var revalidador *revalidadorVinculoAplicacionAdversarial
	var reloj *relojAutorizacionServicioPrueba
	for nombre, caso := range map[string]struct {
		revalidador ports.RevalidadorAutenticacionActorV1
		reloj       ports.Reloj
	}{
		"revalidador": {revalidador: revalidador, reloj: &relojAutorizacionServicioPrueba{ahora: time.Now().UTC()}},
		"reloj":       {revalidador: &revalidadorVinculoAplicacionAdversarial{}, reloj: reloj},
	} {
		t.Run(nombre, func(t *testing.T) {
			servicio, err := NuevoServicioVinculoAutenticacionActorV1(caso.revalidador, caso.reloj)
			if servicio != nil || !errors.Is(err, domain.ErrVinculoAutenticacionActorInvalido) {
				t.Fatalf("dependencia nula aceptada: %v, %v", servicio, err)
			}
		})
	}
}
