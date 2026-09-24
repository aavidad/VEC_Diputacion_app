package ports

import (
	"context"
	"vec-diputacion-granada/internal/vec/domain"
)

// ClaseVersionContactoUsuario separa la lectura del titular de la reserva
// administrativa para llamamiento. El cliente HTTP nunca elige la clase RRHH.
type ClaseVersionContactoUsuario string

const (
	VersionContactoPropia      ClaseVersionContactoUsuario = "propia"
	VersionContactoLlamamiento ClaseVersionContactoUsuario = "llamamiento"
)

type SolicitudVersionContactoUsuario struct {
	Clase             ClaseVersionContactoUsuario
	SujetoRef         string
	ContextoActor     domain.ContextoActor
	Recurso           domain.RecursoAutorizable
	SolicitudBase     domain.DatosSolicitudAutorizacionLigadaV3
	ResultadoContexto domain.ResultadoContextoActorRegistradoV2
}

type OrdenVersionContactoUsuario struct {
	Clase  ClaseVersionContactoUsuario
	Acceso SolicitudAccesoContactoUsuario
}

type ResultadoVersionContactoUsuario struct {
	Encontrado                  bool
	SujetoRef                   string
	Version                     uint64
	AuditoriaConsulta           EvidenciaAuditoriaCentralContactoUsuario
	ConsumoConsultaRef          string
	ConsumoConsultaHuellaSHA256 string
}

type ConsultorVersionContactoUsuario interface {
	ConsultarVersionContactoUsuario(context.Context, OrdenVersionContactoUsuario) (ResultadoVersionContactoUsuario, error)
}
