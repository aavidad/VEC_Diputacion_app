package application

import (
	"errors"
	"regexp"

	"vec-diputacion-granada/internal/vec/ports"
)

var (
	ErrOperacionContactoNoEncontrada   = errors.New("vec: operacion de contacto no encontrada")
	ErrOperacionContactoPreparada      = errors.New("vec: otra operacion de contacto preparada")
	ErrOperacionContactoAccesoDenegado = errors.New("vec: acceso a operacion de contacto denegado")
)

var referenciaOperacionContacto = regexp.MustCompile(`^opr_[A-Za-z0-9_-]{22,128}$`)

func ReferenciaOperacionContactoValida(ref string) bool {
	return referenciaOperacionContacto.MatchString(ref)
}

func ValidarOperacionContacto(op ports.OperacionContactoUsuario) error {
	if !ReferenciaOperacionContactoValida(op.OperacionRef) || op.VersionEsperada >= 1<<53-1 {
		return ErrContactoUsuarioNoDisponible
	}
	switch op.Estado {
	case ports.OperacionContactoPreparada, ports.OperacionContactoCancelada:
		if op.Version != 0 || op.ReciboRef != "" {
			return ErrContactoUsuarioNoDisponible
		}
	case ports.OperacionContactoConfirmada:
		if op.Version != op.VersionEsperada+1 || op.ReciboRef == "" {
			return ErrContactoUsuarioNoDisponible
		}
	default:
		return ErrContactoUsuarioNoDisponible
	}
	return nil
}
