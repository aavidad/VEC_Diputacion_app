package domain

import (
	"context"
	"errors"
	"time"
)

// ResolutorContextoActorGobernadoV1 resuelve el contexto exclusivamente desde
// la autenticacion ya revalidada. No recibe perfil, persona ni candidatura
// declarados por el cliente: la autoridad gobernada los obtiene de sus fuentes
// autoritativas y devuelve el recibo registrado.
type ResolutorContextoActorGobernadoV1 interface {
	ResolverContextoActorGobernadoV1(
		context.Context,
		AutenticacionRevalidadaV1,
	) (ResultadoContextoActorRegistradoV2, error)
}

// CrearVinculoAutenticacionActorGobernadoV1 produce el vinculo V2 y el
// resultado registrado que lo sustenta. Revalida y resuelve una sola vez; el
// resolutor recibe la superficie, cuenta, metodo y garantia revalidados, nunca
// atributos de identidad suministrados por el navegador.
func CrearVinculoAutenticacionActorGobernadoV1(
	ctx context.Context,
	revalidador RevalidadorAutenticacionActorV1,
	solicitud SolicitudRevalidacionAutenticacionActorV1,
	resolutor ResolutorContextoActorGobernadoV1,
	reloj RelojVinculoAutenticacionActorV2,
) (VinculoAutenticacionActorV2, ResultadoContextoActorRegistradoV2, error) {
	if ctx == nil || ctx.Err() != nil ||
		dependenciaVinculoAutenticacionActorV2Nula(revalidador) ||
		dependenciaVinculoAutenticacionActorV2Nula(resolutor) ||
		dependenciaVinculoAutenticacionActorV2Nula(reloj) ||
		solicitud.Validar() != nil {
		return VinculoAutenticacionActorV2{}, ResultadoContextoActorRegistradoV2{},
			ErrVinculoAutenticacionActorV2Invalido
	}

	autenticacion, err := revalidador.RevalidarAutenticacionActorV1(ctx, solicitud)
	if err != nil || ctx.Err() != nil || autenticacion.Validar() != nil ||
		autenticacion.AutenticacionRef != solicitud.AutenticacionRef ||
		autenticacion.SesionRef != solicitud.SesionRef {
		return VinculoAutenticacionActorV2{}, ResultadoContextoActorRegistradoV2{},
			errors.Join(ErrVinculoAutenticacionActorV2Invalido, err, ctx.Err())
	}

	resultado, err := resolutor.ResolverContextoActorGobernadoV1(ctx, autenticacion)
	if err != nil || ctx.Err() != nil || resultado.Validar() != nil {
		return VinculoAutenticacionActorV2{}, ResultadoContextoActorRegistradoV2{},
			errors.Join(ErrVinculoAutenticacionActorV2Invalido, err, ctx.Err())
	}
	resultado, err = resultado.Clonar()
	if err != nil {
		return VinculoAutenticacionActorV2{}, ResultadoContextoActorRegistradoV2{},
			ErrVinculoAutenticacionActorV2Invalido
	}

	ahora := reloj.Ahora().UTC().Truncate(time.Microsecond)
	if errContexto := ctx.Err(); errContexto != nil {
		return VinculoAutenticacionActorV2{}, ResultadoContextoActorRegistradoV2{},
			errors.Join(ErrVinculoAutenticacionActorV2Invalido, errContexto)
	}
	actor := resultado.Contexto
	if !instanteAutorizacionCanonico(ahora) ||
		autenticacion.CuentaRef != actor.Instantanea.CuentaRef ||
		autenticacion.MetodoObservado != actor.Principal.AuthMethod ||
		autenticacion.GarantiaObservada != actor.Principal.AuthAssurance ||
		!autenticacion.Superficie.Valida() ||
		ahora.Before(autenticacion.SesionRevalidadaEn) ||
		!ahora.Before(autenticacion.SesionValidaHasta) ||
		ahora.Before(resultado.ResueltoEnAutoritativo) ||
		!contextoActorV2VigenteEn(actor, ahora) {
		return VinculoAutenticacionActorV2{}, ResultadoContextoActorRegistradoV2{},
			ErrVinculoAutenticacionActorV2Invalido
	}

	vinculo, err := nuevoVinculoAutenticacionActorV2(autenticacion, resultado)
	if err != nil || ctx.Err() != nil || vinculo.ValidarPara(resultado) != nil {
		return VinculoAutenticacionActorV2{}, ResultadoContextoActorRegistradoV2{},
			errors.Join(ErrVinculoAutenticacionActorV2Invalido, err, ctx.Err())
	}
	return vinculo, resultado, nil
}
