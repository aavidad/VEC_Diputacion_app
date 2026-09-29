package domain

import (
	"errors"

	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

var (
	ErrIdentidadInvalida   = errors.New("aspirantes: identidad acreditada invalida")
	ErrSesionInvalida      = errors.New("aspirantes: sesion invalida")
	ErrSuperficieProhibida = errors.New("aspirantes: superficie prohibida")
)

// IdentidadAcreditada es lo que acredita la identificación electrónica de la
// sesión: nombre, apellidos y documento. La persona no la escribe; la toma la
// frontera del portal externo del mismo certificado verificado de la sesión.
type IdentidadAcreditada struct {
	nombre, apellidos string
	documento         DocumentoIdentidad
}

// NuevaIdentidadAcreditada normaliza y valida. Los apellidos van juntos, tal
// como los acredita el certificado.
func NuevaIdentidadAcreditada(nombre, apellidos string, documento DocumentoIdentidad) (IdentidadAcreditada, error) {
	n, err := NormalizarValor(CampoNombre, nombre)
	if err != nil {
		return IdentidadAcreditada{}, ErrIdentidadInvalida
	}
	a, err := NormalizarValor(CampoApellidos, apellidos)
	if err != nil || documento.Validar() != nil {
		return IdentidadAcreditada{}, ErrIdentidadInvalida
	}
	return IdentidadAcreditada{nombre: n, apellidos: a, documento: documento}, nil
}

func (i IdentidadAcreditada) Validar() error {
	if _, err := NuevaIdentidadAcreditada(i.nombre, i.apellidos, i.documento); err != nil {
		return ErrIdentidadInvalida
	}
	return nil
}

func (i IdentidadAcreditada) Documento() DocumentoIdentidad { return i.documento }

// Valores devuelve los campos de identidad, listos para cifrar.
func (i IdentidadAcreditada) Valores() map[CampoFicha]string {
	return map[CampoFicha]string{CampoNombre: i.nombre, CampoApellidos: i.apellidos}
}

// String nunca muestra datos personales en registros ni errores.
func (IdentidadAcreditada) String() string   { return "aspirantes.IdentidadAcreditada{redactada}" }
func (IdentidadAcreditada) GoString() string { return "aspirantes.IdentidadAcreditada{redactada}" }

// SesionAspirante liga la persona canónica y el vínculo V2 certificado a la
// superficie del portal externo. Aspirantes no admite la superficie interna:
// sus datos son de la población externa.
type SesionAspirante struct {
	actor   vecdomain.ContextoActor
	vinculo vecdomain.VinculoAutenticacionActorV2
}

func cotejarSesion(actor vecdomain.ContextoActor, vinculo vecdomain.VinculoAutenticacionActorV2) error {
	if actor.Validar() != nil || (actor.Principal.AuthMethod != vecdomain.AuthMethodCertificate && actor.Principal.AuthMethod != vecdomain.AuthMethodDNIe) || actor.Instantanea.CuentaVersion == 0 {
		return ErrSesionInvalida
	}
	datos, err := vinculo.Datos()
	if err != nil {
		return ErrSesionInvalida
	}
	huella, err := actor.HuellaSHA256VinculadaV2()
	if err != nil {
		return ErrSesionInvalida
	}
	if datos.CuentaPrivilegiada || datos.PrincipalID != actor.PersonaRef || datos.PerfilActivoRef != actor.PerfilActivoRef ||
		datos.CuentaRef != actor.Instantanea.CuentaRef || datos.CuentaOrdinariaRef != actor.Instantanea.CuentaRef ||
		datos.MetodoObservado != actor.Principal.AuthMethod || datos.GarantiaObservada != actor.Principal.AuthAssurance ||
		datos.ContextoActorRef != actor.Instantanea.VinculoRef || datos.ContextoActorVersion != actor.Instantanea.VinculoVersion ||
		datos.ContextoActorCuentaVersion != actor.Instantanea.CuentaVersion || datos.ContextoActorHuellaSHA256 != huella {
		return ErrSesionInvalida
	}
	if datos.Superficie != vecdomain.SuperficieAutenticacionExternaPersonalV1 {
		return ErrSuperficieProhibida
	}
	return nil
}

// NuevaSesionAspirante exige que la ruta y el vínculo sean del portal externo.
func NuevaSesionAspirante(actor vecdomain.ContextoActor, vinculo vecdomain.VinculoAutenticacionActorV2, superficieRuta vecdomain.SuperficieAutenticacionActorV1) (SesionAspirante, error) {
	if superficieRuta != vecdomain.SuperficieAutenticacionExternaPersonalV1 {
		return SesionAspirante{}, ErrSuperficieProhibida
	}
	if err := cotejarSesion(actor, vinculo); err != nil {
		return SesionAspirante{}, err
	}
	copia, err := actor.Clonar()
	if err != nil {
		return SesionAspirante{}, ErrSesionInvalida
	}
	return SesionAspirante{actor: copia, vinculo: vinculo}, nil
}

// Datos vuelve a cotejar en cada uso y entrega copias.
func (s SesionAspirante) Datos() (vecdomain.ContextoActor, vecdomain.VinculoAutenticacionActorV2, error) {
	if err := cotejarSesion(s.actor, s.vinculo); err != nil {
		return vecdomain.ContextoActor{}, vecdomain.VinculoAutenticacionActorV2{}, err
	}
	copia, err := s.actor.Clonar()
	if err != nil {
		return vecdomain.ContextoActor{}, vecdomain.VinculoAutenticacionActorV2{}, ErrSesionInvalida
	}
	return copia, s.vinculo, nil
}
