package ports

import (
	"context"
	"time"
	"vec-diputacion-granada/internal/modules/provision/domain"
	b "vec-diputacion-granada/internal/shared/baremacion"
)

// Estos contratos son dependencias futuras. No se montan en el CLI de ensayo.
// Las referencias no acreditan una concesión, una fuente ni una firma.
type ReferenciaFuenteCiclo struct {
	Autoridad    string
	Referencia   string
	Version      string
	HuellaSHA256 string
}

type ConsultaInstantaneaCiclo struct {
	PersonaRef          string
	EmpleadoRef         string
	PuestoRef           string
	ProcesoRef          string
	ProcesoVersion      string
	FechaCorte          b.FechaCivil
	ConcesionCentralRef string
	CorrelacionRef      string
}

type InstantaneaFuentesCiclo struct {
	Entrada  domain.Entrada
	Personal ReferenciaFuenteCiclo
	RUM      ReferenciaFuenteCiclo
	RPT      ReferenciaFuenteCiclo
}

// FuenteInstantaneasCiclo consulta por puertos de Personal/RUM/RPT y fecha
// exacta. Una dependencia ausente falla; no devuelve un mérito vacío como cero.
type FuenteInstantaneasCiclo interface {
	ObtenerAutorizada(context.Context, ConsultaInstantaneaCiclo) (InstantaneaFuentesCiclo, error)
}

type EscrituraCicloProvision struct {
	ProcesoRef            string
	ProcesoVersion        string
	SolicitudRef          string
	SolicitudVersion      string
	ValoracionRef         string
	VersionEsperada       uint32
	IdempotenciaRef       string
	HuellaSemanticaSHA256 string
	ConcesionCentralRef   string
	CorrelacionRef        string
	NuevaValoracion       domain.ValoracionCiclo
	Reclamacion           domain.Reclamacion
	Decision              domain.DecisionRevision
	Fuentes               InstantaneaFuentesCiclo
}

type ReciboCicloProvision struct {
	Referencia     string
	ProcesoRef     string
	SolicitudRef   string
	ValoracionRef  string
	Version        uint32
	HuellaRevision string
	Fecha          time.Time
	HistoriaRef    string
	AuditoriaRef   string
	OutboxRef      string
}

type ConsultaReciboCiclo struct {
	ValoracionRef       string
	IdempotenciaRef     string
	ConcesionCentralRef string
	CorrelacionRef      string
}

// RepositorioCicloProvision debe resolver la concesión central y revalidarla
// antes del efecto. Consume la autorización en la transacción que comprueba
// version_esperada e idempotencia y añade valoración, reclamación/decisión,
// recibo, historia, auditoría y outbox. Nunca modifica una versión publicada.
// Mismo material/clave recupera el recibo original; otra huella o versión
// obsoleta produce conflicto sin efecto. Recuperar vuelve a autorizar lectura.
// Ninguna implementación ni garantía PostgreSQL se acredita en este corte.
type RepositorioCicloProvision interface {
	AnadirRevisionAutorizada(context.Context, EscrituraCicloProvision) (ReciboCicloProvision, error)
	RecuperarAutorizado(context.Context, ConsultaReciboCiclo) (ReciboCicloProvision, error)
}

type PeticionDocumentoCiclo struct {
	ProcesoRef            string
	ProcesoVersion        string
	DocumentoRef          string
	DocumentoVersion      string
	HuellaDocumentoSHA256 string
	VersionValoracion     uint32
	HuellaValoracion      string
	ConcesionCentralRef   string
	IdempotenciaRef       string
	CorrelacionRef        string
}

// Firma y publicación son autoridades documentales separadas. No se inferirá
// ninguna de autenticación, huellas locales o un borrador descargado.
type FirmaResolucionProvision interface {
	FirmarAutorizada(context.Context, PeticionDocumentoCiclo) (ReferenciaFuenteCiclo, error)
}

type PublicacionResolucionProvision interface {
	PublicarFirmadaAutorizada(context.Context, PeticionDocumentoCiclo, ReferenciaFuenteCiclo) (ReferenciaFuenteCiclo, error)
}
