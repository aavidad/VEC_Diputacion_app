package domain

import (
	"errors"

	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

var ErrIdentidadCorreosInvalida = errors.New("usuarios correos: identidad invalida")
var ErrSuperficieCorreosProhibida = errors.New("usuarios correos: superficie prohibida")

// IdentidadCorreos vincula la persona canónica y el certificado V2 a la
// superficie que fija la ruta. Sus campos son opacos para los adaptadores.
type IdentidadCorreos struct {
	actor      vecdomain.ContextoActor
	vinculo    vecdomain.VinculoAutenticacionActorV2
	superficie vecdomain.SuperficieAutenticacionActorV1
}

func cotejarIdentidadCorreos(actor vecdomain.ContextoActor, vinculo vecdomain.VinculoAutenticacionActorV2) bool {
	if actor.Validar() != nil || (actor.Principal.AuthMethod != vecdomain.AuthMethodCertificate && actor.Principal.AuthMethod != vecdomain.AuthMethodDNIe) || actor.Instantanea.CuentaVersion == 0 {
		return false
	}
	datos, err := vinculo.Datos()
	if err != nil || datos.CuentaPrivilegiada {
		return false
	}
	huella, err := actor.HuellaSHA256VinculadaV2()
	return err == nil && datos.PrincipalID == actor.PersonaRef && datos.PerfilActivoRef == actor.PerfilActivoRef &&
		datos.CuentaRef == actor.Instantanea.CuentaRef && datos.CuentaOrdinariaRef == actor.Instantanea.CuentaRef &&
		datos.MetodoObservado == actor.Principal.AuthMethod && datos.GarantiaObservada == actor.Principal.AuthAssurance &&
		datos.ContextoActorRef == actor.Instantanea.VinculoRef && datos.ContextoActorVersion == actor.Instantanea.VinculoVersion &&
		datos.ContextoActorCuentaVersion == actor.Instantanea.CuentaVersion && datos.ContextoActorHuellaSHA256 == huella
}

func NuevaIdentidadCorreos(actor vecdomain.ContextoActor, vinculo vecdomain.VinculoAutenticacionActorV2, superficieRuta vecdomain.SuperficieAutenticacionActorV1) (IdentidadCorreos, error) {
	if !cotejarIdentidadCorreos(actor, vinculo) {
		return IdentidadCorreos{}, ErrIdentidadCorreosInvalida
	}
	datos, _ := vinculo.Datos()
	if (superficieRuta != vecdomain.SuperficieAutenticacionInternaCorporativaV1 && superficieRuta != vecdomain.SuperficieAutenticacionExternaPersonalV1) || datos.Superficie != superficieRuta {
		return IdentidadCorreos{}, ErrSuperficieCorreosProhibida
	}
	copia, err := actor.Clonar()
	if err != nil {
		return IdentidadCorreos{}, ErrIdentidadCorreosInvalida
	}
	return IdentidadCorreos{actor: copia, vinculo: vinculo, superficie: superficieRuta}, nil
}

func (i IdentidadCorreos) Datos() (vecdomain.ContextoActor, vecdomain.VinculoAutenticacionActorV2, vecdomain.SuperficieAutenticacionActorV1, error) {
	if !cotejarIdentidadCorreos(i.actor, i.vinculo) {
		return vecdomain.ContextoActor{}, vecdomain.VinculoAutenticacionActorV2{}, "", ErrIdentidadCorreosInvalida
	}
	datos, _ := i.vinculo.Datos()
	if (i.superficie != vecdomain.SuperficieAutenticacionInternaCorporativaV1 && i.superficie != vecdomain.SuperficieAutenticacionExternaPersonalV1) || datos.Superficie != i.superficie {
		return vecdomain.ContextoActor{}, vecdomain.VinculoAutenticacionActorV2{}, "", ErrSuperficieCorreosProhibida
	}
	copia, err := i.actor.Clonar()
	if err != nil {
		return vecdomain.ContextoActor{}, vecdomain.VinculoAutenticacionActorV2{}, "", ErrIdentidadCorreosInvalida
	}
	return copia, i.vinculo, i.superficie, nil
}
