package ports

import (
	"context"
	"errors"

	"vec-diputacion-granada/internal/vec/domain"
)

var ErrSolicitudAccesoContactoUsuarioInvalida = errors.New("vec: solicitud de acceso a contacto de usuario invalida")

// SolicitudAccesoContactoUsuario liga la apertura efímera a un actor VEC ya
// resuelto. No contiene dirección ni un indicador booleano de autorización.
type SolicitudAccesoContactoUsuario struct {
	SujetoRef     string
	ContextoActor domain.ContextoActor
	FinalidadRef  string
}

func (s SolicitudAccesoContactoUsuario) Validar() error {
	if s.ContextoActor.Validar() != nil || !domain.ReferenciaSujetoContactoUsuarioValida(s.SujetoRef) || s.FinalidadRef == "" {
		return ErrSolicitudAccesoContactoUsuarioInvalida
	}
	return nil
}

// RegistroContactoUsuario aplica la versión esperada en la persistencia.
// Alta y actualización reciben el valor privado, que debe haber sido creado
// antes en la frontera de alta; no aceptan correo como identificador.
type RegistroContactoUsuario interface {
	RegistrarContactoUsuario(context.Context, domain.ContextoActor, domain.ContactoUsuario) error
	ActualizarContactoUsuario(context.Context, domain.ContextoActor, domain.ContactoUsuario, uint64) (domain.ContactoUsuario, error)
}

// ResolutorContactoUsuarioAutorizado es la única lectura prevista para datos
// personales: el adaptador implementador comprueba su autoridad y entrega el
// contacto sólo en el callback, sin DTO con dirección.
type ResolutorContactoUsuarioAutorizado interface {
	ConContactoUsuario(context.Context, SolicitudAccesoContactoUsuario, func(domain.ContactoUsuario) error) error
}
