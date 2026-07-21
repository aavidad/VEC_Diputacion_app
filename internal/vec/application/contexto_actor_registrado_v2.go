package application

import (
	"context"
	"errors"

	"vec-diputacion-granada/internal/vec/domain"
)

// AutoridadContextoActorRegistradoV2 adapta la salida durable del caso de uso
// al contrato que puede invocar la fabrica del vinculo autenticacion-actor.
// No conserva ni publica OperacionRef.
type AutoridadContextoActorRegistradoV2 struct {
	servicio *ServicioContextoActor
}

func NuevaAutoridadContextoActorRegistradoV2(
	servicio *ServicioContextoActor,
) (*AutoridadContextoActorRegistradoV2, error) {
	if servicio == nil || servicio.modo != modoServicioContextoActorProductivoV2 {
		return nil, domain.ErrVinculoAutenticacionActorV2Invalido
	}
	return &AutoridadContextoActorRegistradoV2{servicio: servicio}, nil
}

func (a *AutoridadContextoActorRegistradoV2) ResolverContextoActorRegistradoV2(
	ctx context.Context,
	solicitud domain.SolicitudContextoActor,
) (domain.ResultadoContextoActorRegistradoV2, error) {
	if ctx == nil || ctx.Err() != nil || a == nil || a.servicio == nil ||
		a.servicio.modo != modoServicioContextoActorProductivoV2 || solicitud.Validar() != nil {
		return domain.ResultadoContextoActorRegistradoV2{},
			domain.ErrVinculoAutenticacionActorV2Invalido
	}
	confirmacion, err := a.servicio.ResolverRegistrado(ctx, solicitud)
	if err != nil || ctx.Err() != nil {
		return domain.ResultadoContextoActorRegistradoV2{}, errors.Join(
			domain.ErrVinculoAutenticacionActorV2Invalido, ctx.Err(), err,
		)
	}
	resultado := domain.ResultadoContextoActorRegistradoV2{
		RegistroContextoRef: confirmacion.RegistroContextoRef,
		Contexto:            confirmacion.Contexto,
		RepresentacionCanonica: append(
			[]byte(nil), confirmacion.RepresentacionCanonica...,
		),
		HuellaSHA256: confirmacion.HuellaSHA256,
		ManifiestoProcedenciaCanonico: append(
			[]byte(nil), confirmacion.ManifiestoProcedenciaCanonico...,
		),
		ManifiestoProcedenciaHuellaSHA256: confirmacion.ManifiestoProcedenciaHuellaSHA256,
		AutoridadEfectiva:                 confirmacion.AutoridadEfectiva,
		ResueltoEnAutoritativo:            confirmacion.ResueltoEnAutoritativo,
	}
	clon, err := resultado.Clonar()
	if err != nil {
		return domain.ResultadoContextoActorRegistradoV2{},
			domain.ErrVinculoAutenticacionActorV2Invalido
	}
	return clon, nil
}

var _ domain.ResolutorContextoActorRegistradoV2 = (*AutoridadContextoActorRegistradoV2)(nil)
