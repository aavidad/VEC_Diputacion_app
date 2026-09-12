package ports

import (
	"context"

	"vec-diputacion-granada/internal/vec/domain"
)

// SolicitudRegistroContactoUsuario sólo sirve mientras se prepara el sobre.
// El contacto no puede cruzar la frontera durable posterior.
type SolicitudRegistroContactoUsuario struct {
	ContextoActor     domain.ContextoActor
	Contacto          domain.ContactoUsuario
	VersionEsperada   uint64
	FinalidadRef      string
	Recurso           domain.RecursoAutorizable
	Audiencia         string
	SolicitudBase     domain.DatosSolicitudAutorizacionLigadaV3
	ResultadoContexto domain.ResultadoContextoActorRegistradoV2
}

// PreparacionRegistroContactoUsuario fija los bytes exactos que autoriza V3.
// No contiene ni referencia al valor claro ni al agregado ContactoUsuario.
type PreparacionRegistroContactoUsuario struct {
	SujetoRef       string
	VersionEsperada uint64
	VersionNueva    uint64
	Sobre           SobreContactoUsuario
	Auditoria       domain.AuditEntry
	FinalidadRef    string
	Audiencia       string
	Recurso         domain.RecursoAutorizable
	PayloadNegocio  []byte
}

// OrdenRegistroContactoUsuario es el único transporte durable. El registro
// debe reconstruir y comparar PayloadNegocio y la huella/selector del Recurso;
// consumir V3 con firmas, gobierno y revocación frescos en la MISMA transacción
// que sobre, versión, auditoría y outbox. El cuerpo no es PayloadVECAD3: este
// último es la atestación nativa que compromete la decisión sobre el recurso.
type OrdenRegistroContactoUsuario struct {
	Preparacion PreparacionRegistroContactoUsuario
	Material    ExportacionMaterialConsumoAutorizacionAtestadaV3
}

type ReciboContactoUsuario struct {
	SujetoRef string
	Version   uint64
	Auditoria domain.AuditEntry
}

type SobreContactoUsuario struct {
	Version        uint64
	ClaveRef       string
	Nonce, Cifrado []byte
}

type ProtectorContactoUsuario interface {
	CifrarContactoUsuario(context.Context, string, uint64, []byte) (SobreContactoUsuario, error)
	ConContactoUsuarioDescifrado(context.Context, string, SobreContactoUsuario, func([]byte) error) error
}

// PreparadorAuditoriaContactoUsuario procede de la autoridad central: debe
// seudonimizar el actor con HMAC vigente y no aceptar un ActorID declarado.
// AuthorizationRef permanece vacío: la decisión todavía no existe. El
// registro enlaza la auditoría final con la DecisionRef que verifica/consume
// dentro de su transacción, sin modificar la preimagen preparada.
type PreparadorAuditoriaContactoUsuario interface {
	PrepararAuditoriaContactoUsuario(context.Context, domain.ContextoActor, string, string, string, uint64) (domain.AuditEntry, error)
}

// AutorizadorContactoUsuario conserva la firma nativa del emisor V3 existente.
// La composición debe inyectar esa autoridad; este puerto no registra módulos,
// concede permisos ni acepta exportaciones estructurales como autorización.
type AutorizadorContactoUsuario interface {
	EmitirMaterialAutorizacionAtestadaV3(context.Context, domain.SolicitudAutorizacionLigadaV3, domain.ResultadoContextoActorRegistradoV2) (domain.DecisionAutorizacionLigadaV3, ConfirmacionRegistroConcesionAutorizacionLigadaV3, ExportadorMaterialConsumoAutorizacionAtestadaV3, error)
}

type RegistroContactoUsuario interface {
	GuardarContactoUsuario(context.Context, OrdenRegistroContactoUsuario) (ReciboContactoUsuario, error)
}

// SolicitudAccesoContactoUsuario sigue siendo la frontera de lectura. El
// adaptador verifica firmas/gobierno, consumo y auditoría de acceso, revalida
// revocación y vigencia tras descifrar y antes del único callback activo.
// Ningún campo proviene directamente de una petición HTTP sin autoridad.
type SolicitudAccesoContactoUsuario struct {
	Auditoria         domain.AuditEntry
	SujetoRef         string
	ContextoActor     domain.ContextoActor
	FinalidadRef      string
	Recurso           domain.RecursoAutorizable
	Audiencia         string
	PayloadNegocio    []byte
	Material          ExportacionMaterialConsumoAutorizacionAtestadaV3
	Version           uint64
	Solicitud         domain.SolicitudAutorizacionLigadaV3
	Decision          domain.DecisionAutorizacionLigadaV3
	Confirmacion      ConfirmacionRegistroConcesionAutorizacionLigadaV3
	ResultadoContexto domain.ResultadoContextoActorRegistradoV2
}

type ResolutorContactoUsuarioAutorizado interface {
	ConContactoUsuario(context.Context, SolicitudAccesoContactoUsuario, func(domain.ContactoUsuario) error) error
}
