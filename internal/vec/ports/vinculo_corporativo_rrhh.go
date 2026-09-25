package ports

import (
	"context"
	"errors"
	"strings"
)

var (
	// ErrVinculoCorporativoRRHHNoVigente cubre por igual ausencia, revocación,
	// caducidad o versión distinta de cualquier eslabón: no distingue motivos
	// para no revelar el estado corporativo de la persona.
	ErrVinculoCorporativoRRHHNoVigente = errors.New("vec: vinculo corporativo RRHH no vigente")
	// ErrVinculoCorporativoRRHHNoDisponible indica que la autoridad no pudo
	// responder. Nunca equivale a una respuesta positiva.
	ErrVinculoCorporativoRRHHNoDisponible      = errors.New("vec: revalidacion de vinculo corporativo RRHH no disponible")
	ErrSolicitudVinculoCorporativoRRHHInvalida = errors.New("vec: solicitud de revalidacion de vinculo corporativo RRHH invalida")
)

// SolicitudRevalidacionVinculoCorporativoRRHHV1 fija los datos exactos que la
// resolución F1 de esta petición acaba de acreditar. No lleva organización ni
// instante: la organización la fija el propio vínculo y el instante lo pone la
// autoridad.
type SolicitudRevalidacionVinculoCorporativoRRHHV1 struct {
	CuentaRef              string
	PerfilRef              string
	PersonaRef             string
	VinculoContextoRef     string
	VinculoContextoVersion uint64
}

func (s SolicitudRevalidacionVinculoCorporativoRRHHV1) Validar() error {
	if !referenciaContextoActorValida(s.CuentaRef, "cta_") || !referenciaContextoActorValida(s.PerfilRef, "prf_") ||
		!referenciaContextoActorValida(s.PersonaRef, "per_") ||
		!referenciaContextoActorValida(s.VinculoContextoRef, "vca_") ||
		s.VinculoContextoVersion == 0 {
		return ErrSolicitudVinculoCorporativoRRHHInvalida
	}
	return nil
}

// RevalidadorVinculoCorporativoRRHHV1 consulta, en cada llamada y sin caché,
// si la persona conserva el vínculo corporativo interna_corporativa/
// consulta_rrhh actual, activo y vigente ligado al vínculo de contexto dado.
// Devuelve nil solo con respuesta positiva de la autoridad; cualquier otra
// situación es un error.
type RevalidadorVinculoCorporativoRRHHV1 interface {
	RevalidarVinculoCorporativoRRHHV1(context.Context, SolicitudRevalidacionVinculoCorporativoRRHHV1) error
}

// referenciaContextoActorValida replica referencia_valida de ContextoActor
// 000001: prefijo y sufijo de 22 a 128 octetos [A-Za-z0-9_-].
func referenciaContextoActorValida(valor, prefijo string) bool {
	if !strings.HasPrefix(valor, prefijo) {
		return false
	}
	sufijo := valor[len(prefijo):]
	if len(sufijo) < 22 || len(sufijo) > 128 {
		return false
	}
	for i := 0; i < len(sufijo); i++ {
		c := sufijo[i]
		if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (c < '0' || c > '9') && c != '_' && c != '-' {
			return false
		}
	}
	return true
}
