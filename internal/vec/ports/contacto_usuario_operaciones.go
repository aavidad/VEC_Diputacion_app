package ports

import (
	"context"

	"vec-diputacion-granada/internal/vec/domain"
)

// EstadoOperacionContactoUsuario describe una intención de contacto propia.
// La fila confirmada y su recibo se escriben en la misma transacción que el
// contacto; cancelada conserva historia y permite preparar otra intención.
type EstadoOperacionContactoUsuario string

const (
	OperacionContactoPreparada  EstadoOperacionContactoUsuario = "preparada"
	OperacionContactoConfirmada EstadoOperacionContactoUsuario = "confirmada"
	OperacionContactoCancelada  EstadoOperacionContactoUsuario = "cancelada"
)

// OperacionContactoUsuario omite correo, HMAC y sobre incluso en consultas
// internas. OperacionRef sólo selecciona; nunca acredita titularidad ni permiso.
type OperacionContactoUsuario struct {
	OperacionRef     string
	Estado           EstadoOperacionContactoUsuario
	VersionEsperada  uint64
	Version          uint64
	ReciboRef        string
	ReplayConfirmado bool // Sólo para que HTTP distinga 201 de 200.
}

// Las órdenes durables transportan la autorización nominal V3 ya emitida. La
// persistencia revalida y consume dentro de su transacción SERIALIZABLE.
type OrdenPrepararOperacionContacto struct {
	OperacionRef    string
	SujetoRef       string
	VersionEsperada uint64
	HuellasReplay   []HuellaSolicitudContactoUsuario
	Acceso          SolicitudAccesoContactoUsuario
}

type OrdenCancelarOperacionContacto struct {
	OperacionRef string
	SujetoRef    string
	Acceso       SolicitudAccesoContactoUsuario
}

type OrdenListarOperacionesContacto struct {
	SujetoRef string
	Limite    uint32
	DespuesDe string
	Acceso    SolicitudAccesoContactoUsuario
}

type ResultadoListaOperacionesContacto struct {
	Operaciones    []OperacionContactoUsuario
	SiguienteDesde string
	Auditoria      EvidenciaAuditoriaCentralContactoUsuario
}

type OrdenDetalleOperacionContacto struct {
	OperacionRef string
	SujetoRef    string
	Acceso       SolicitudAccesoContactoUsuario
}

type ResultadoDetalleOperacionContacto struct {
	Encontrada bool
	Operacion  OperacionContactoUsuario
	Auditoria  EvidenciaAuditoriaCentralContactoUsuario
}

type RepositorioOperacionesContactoUsuario interface {
	PrepararOperacionContacto(context.Context, OrdenPrepararOperacionContacto) (OperacionContactoUsuario, error)
	CancelarOperacionContacto(context.Context, OrdenCancelarOperacionContacto) (OperacionContactoUsuario, error)
	ListarOperacionesContacto(context.Context, OrdenListarOperacionesContacto) (ResultadoListaOperacionesContacto, error)
	DetalleOperacionContacto(context.Context, OrdenDetalleOperacionContacto) (ResultadoDetalleOperacionContacto, error)
}

// GeneradorOperacionContactoUsuario produce una referencia aleatoria; el
// cliente no la elige. No acepta identificadores derivados de correo/persona.
type GeneradorOperacionContactoUsuario interface {
	NuevaOperacionContactoUsuario(context.Context) (string, error)
}

// La preparación de aplicación parte siempre de ContextoActor registrado.
// Correo sólo vive el tiempo necesario para derivar HMAC semántico; ningún
// campo claro se inserta en OrdenPrepararOperacionContacto.
type SolicitudPrepararOperacionContacto struct {
	ContextoActor     domain.ContextoActor
	Correo            string
	VersionEsperada   uint64
	Recurso           domain.RecursoAutorizable
	SolicitudBase     domain.DatosSolicitudAutorizacionLigadaV3
	ResultadoContexto domain.ResultadoContextoActorRegistradoV2
}
