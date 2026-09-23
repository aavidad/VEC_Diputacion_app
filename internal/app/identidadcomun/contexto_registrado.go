package identidadcomun

import (
	"bytes"
	"context"
	"errors"

	vec "vec-diputacion-granada/internal/vec/domain"
)

var ErrContextoRegistradoNoDisponible = errors.New("identidad comun: contexto registrado no disponible")

// ResolverEsperadoRegistrado recupera de la autoridad la preimagen operativa
// de una identidad ya fijada al componer. No publica ni crea identidades.
func ResolverEsperadoRegistrado(ctx context.Context, resolutor vec.ResolutorContextoActorRegistradoV2, semilla vec.ResultadoContextoActorRegistradoV2) (vec.ResultadoContextoActorRegistradoV2, error) {
	if ctx == nil || ctx.Err() != nil || resolutor == nil || semilla.Validar() != nil {
		return vec.ResultadoContextoActorRegistradoV2{}, ErrContextoRegistradoNoDisponible
	}
	actor := semilla.Contexto
	registrado, err := resolutor.ResolverContextoActorRegistradoV2(ctx, vec.SolicitudContextoActor{
		Cuenta: vec.CuentaAutenticadaContextoActor{
			CuentaRef: actor.Instantanea.CuentaRef,
			Metodo:    vec.AuthMethodCertificate, Garantia: vec.AuthAssuranceHigh,
		},
		PerfilActivoRef: actor.PerfilActivoRef,
	})
	if err != nil || registrado.Validar() != nil ||
		registrado.AutoridadEfectiva != vec.AutoridadProcedenciaContextoActorMaestraAcreditadaV1 ||
		registrado.Contexto.Principal.AuthMethod != vec.AuthMethodCertificate ||
		registrado.Contexto.Principal.AuthAssurance != vec.AuthAssuranceHigh ||
		registrado.Contexto.Instantanea.CuentaRef != actor.Instantanea.CuentaRef ||
		registrado.Contexto.Instantanea.PersonaRef != actor.Instantanea.PersonaRef ||
		registrado.Contexto.Instantanea.PerfilActivoRef != actor.Instantanea.PerfilActivoRef ||
		registrado.Contexto.Instantanea.VinculoRef != actor.Instantanea.VinculoRef ||
		registrado.Contexto.PersonaRef != actor.PersonaRef ||
		registrado.Contexto.PerfilActivoRef != actor.PerfilActivoRef ||
		registrado.Contexto.Principal.ID != actor.Principal.ID ||
		!registrado.Contexto.Instantanea.VigenteEn(registrado.ResueltoEnAutoritativo) {
		return vec.ResultadoContextoActorRegistradoV2{}, ErrContextoRegistradoNoDisponible
	}
	for _, vinculo := range registrado.Contexto.Instantanea.Vinculos {
		if !vinculo.VigenteEn(registrado.ResueltoEnAutoritativo) {
			return vec.ResultadoContextoActorRegistradoV2{}, ErrContextoRegistradoNoDisponible
		}
	}
	return registrado.Clonar()
}

// MismoContextoEsperadoRegistrado conserva la comparación F1 sin prolongar
// artificialmente la vida del recibo: referencia e instante pueden renovarse.
func MismoContextoEsperadoRegistrado(esperado, actual vec.ResultadoContextoActorRegistradoV2) bool {
	if esperado.Validar() != nil || actual.Validar() != nil ||
		esperado.AutoridadEfectiva != actual.AutoridadEfectiva ||
		!bytes.Equal(esperado.ManifiestoProcedenciaCanonico, actual.ManifiestoProcedenciaCanonico) ||
		esperado.ManifiestoProcedenciaHuellaSHA256 != actual.ManifiestoProcedenciaHuellaSHA256 {
		return false
	}
	actor, err := actual.Contexto.Clonar()
	if err != nil {
		return false
	}
	actor.ResueltoEn = esperado.Contexto.ResueltoEn
	canon, err := actor.RepresentacionCanonicaVinculadaV2()
	return err == nil && bytes.Equal(canon, esperado.RepresentacionCanonica)
}
