package ports

import (
	"context"
	"errors"
	"time"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const (
	EsquemaMiBolsaV1       = "vec.bolsa.mi-bolsa.v1"
	AccionConsultarMiBolsa = "bolsa.participaciones_propias.consultar"
	ModuloMiBolsa          = "bolsa"
	TipoRecursoMiBolsa     = "participaciones_candidato"
	FinalidadMiBolsa       = "consulta_participaciones_propias"
	AudienciaMiBolsa       = "vec.bolsa.mi-bolsa.v1"
	CampoMiBolsa           = "participaciones_candidato_minimizadas"
)

var (
	ErrMaterialMiBolsaNoDisponible = errors.New("bolsa: material mi bolsa no disponible")
	ErrConsultaMiBolsaInvalida     = errors.New("bolsa: consulta de mi bolsa invalida")
	ErrResultadoMiBolsaInvalido    = errors.New("bolsa: resultado de mi bolsa invalido")
)

// ParticipacionMiBolsa es la proyeccion cerrada que el SQL debe devolver. No
// contiene la referencia de participacion, candidato, persona ni baremo.
type ParticipacionMiBolsa struct {
	Bolsa             string
	Categoria         string
	Version           uint64
	OrdenInicial      uint64
	TotalInstantanea  uint64
	EstadoBolsa       string
	VigenteDesde      time.Time
	VigenteHasta      *time.Time
	SituacionActual   *SituacionActualMiBolsa
	UltimoLlamamiento *UltimoLlamamientoMiBolsa
}

// UltimoLlamamientoMiBolsa solo expone el resultado de correo propio B7.
// No acredita recepción ni respuesta de la persona.
type UltimoLlamamientoMiBolsa struct {
	EmitidoEn time.Time
	Canal     string
	Resultado string
}

// SituacionActualMiBolsa contiene solo el ultimo hecho B2 autorizado.
// El motivo libre, actor, recibo y referencia interna no salen de Bolsa.
type SituacionActualMiBolsa struct {
	Estado          string
	Desde           time.Time
	Hasta           *time.Time
	FechaDisponible *time.Time
}

type InstantaneaMiBolsa struct {
	ConsultadaEn    time.Time
	Participaciones []ParticipacionMiBolsa
	// Portal solo existe si la consulta pidió el estado del portal propio.
	Portal []EstadoPortalCandidato
	// ReglasPortal resume lo que la persona necesita para actuar: causas de
	// renuncia justificada, fin máximo de una pausa pedida hoy y modo.
	ReglasPortal *ReglasPortalVisibles
	// Ofertas solo existe si la consulta pidió las ofertas de sus bolsas.
	Ofertas []OfertaPortalCandidato
	// Contactos solo existe si la consulta pidió el estado de su contacto.
	Contactos []ContactoPortalCandidato
}

type ReglasPortalVisibles struct {
	CausasRenuncia []string
	PausaMaxima    time.Time
	ModoRespuesta  string
}

// SolicitudConsultaMiBolsa transporta el selector y material nominal emitido
// tras autorización. No concede autoridad: SQL debe cotejar y consumir todo
// el material contra candidato, acción, recurso, audiencia y campos exactos.
type SolicitudConsultaMiBolsa struct {
	CandidatoRef string
	Material     puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3
	ConsultadaEn time.Time
	// ResultadosEfectivos, si no está vacío, pide en la misma transacción el
	// estado del portal propio (llamamiento abierto y solicitudes pendientes).
	ResultadosEfectivos []string
	// LeerOfertas pide en la misma transacción las ofertas abiertas de sus
	// bolsas y aquellas en que ya manifestó disposición (Bolsa 000029).
	LeerOfertas bool
	// LeerContacto pide el estado del contacto de cada bolsa (Bolsa 000040).
	LeerContacto bool
}

// ConsultaMiBolsa es el contrato exacto para PostgreSQL. Debe revalidar y
// consumir la evidencia con la auditoría de lectura en la misma transacción.
type ConsultaMiBolsa interface {
	ConsultarMiBolsa(context.Context, SolicitudConsultaMiBolsa) (InstantaneaMiBolsa, error)
}

// ProveedorMaterialMiBolsa recibe solo una decisión V3 ya exigida y cotejada.
// La composición debe usar la cadena nominal de atestación y confianza central,
// sin repetir el PDP ni reconstruir materiales a partir de bytes del cliente.
type ProveedorMaterialMiBolsa interface {
	EmitirMaterialMiBolsa(context.Context, dominiovec.SolicitudAutorizacionLigadaV3, dominiovec.ResultadoContextoActorRegistradoV2, dominiovec.DecisionAutorizacionLigadaV3, puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3) (puertosvec.ExportadorMaterialConsumoAutorizacionAtestadaV3, error)
}
