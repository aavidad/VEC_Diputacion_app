package ports

import (
	"context"
	"vec-diputacion-granada/internal/vec/domain"
)

// SolicitudConsultaReciboContactoUsuario selecciona una versión histórica
// exacta del sujeto del contexto registrado. No transporta correo ni sobre.
type SolicitudConsultaReciboContactoUsuario struct {
	ContextoActor     domain.ContextoActor
	Version           uint64
	Recurso           domain.RecursoAutorizable
	SolicitudBase     domain.DatosSolicitudAutorizacionLigadaV3
	ResultadoContexto domain.ResultadoContextoActorRegistradoV2
}

// OrdenConsultaReciboContactoUsuario reutiliza el transporte nominal de
// autorización. Su esquema/audiencia de recibo se validan por separado de la
// consulta del correo para envío; esta orden nunca permite descifrarlo.
type OrdenConsultaReciboContactoUsuario struct {
	Acceso SolicitudAccesoContactoUsuario
}

// ResultadoConsultaReciboContactoUsuario separa el registro original del
// acceso actual. Encontrado=false también conserva su auditoría de lectura;
// no demuestra que una escritura concurrente o posterior no pueda confirmar.
type ResultadoConsultaReciboContactoUsuario struct {
	Encontrado                  bool
	SujetoRef                   string
	Version                     uint64
	ReciboOriginal              ReciboContactoUsuario
	AuditoriaConsulta           EvidenciaAuditoriaCentralContactoUsuario
	ConsumoConsultaRef          string
	ConsumoConsultaHuellaSHA256 string
}

type ConsultorReciboContactoUsuario interface {
	ConsultarReciboContactoUsuario(context.Context, OrdenConsultaReciboContactoUsuario) (ResultadoConsultaReciboContactoUsuario, error)
}
