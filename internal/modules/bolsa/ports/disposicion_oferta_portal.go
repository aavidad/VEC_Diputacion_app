package ports

import (
	"context"
	"errors"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

// Disposición a ofertas publicadas desde «Mi bolsa» (Bolsa 000029). La
// persona actúa sobre la oferta (TipoRecursoOfertaBolsa) con la acción propia
// AccionManifestarDisposicionPropia; PostgreSQL coteja al candidato con el
// vínculo del contexto atestado y resuelve su participación.

// Estados de una oferta tal como la ve la persona. No revelan a quién se
// adjudicó ni cuántas personas se ofrecieron.
const (
	EstadoOfertaPortalAbierta             = "abierta"
	EstadoOfertaPortalPendienteResolucion = "pendiente_resolucion"
	EstadoOfertaPortalResuelta            = "resuelta"
	EstadoOfertaPortalAdjudicadaPropia    = "adjudicada_propia"
)

var (
	// ErrPortalOfertaNoAbierta: la oferta venció o ya está resuelta.
	ErrPortalOfertaNoAbierta = errors.New("bolsa: la oferta no está abierta")
	// ErrPortalDisposicionYaManifestada: la persona ya se ofreció con otra clave.
	ErrPortalDisposicionYaManifestada = errors.New("bolsa: disposición ya manifestada")
)

// OfertaPortalCandidato es una oferta de una bolsa de la persona: abierta o
// en la que ya manifestó disposición.
type OfertaPortalCandidato struct {
	OfertaRef    string
	Bolsa        string
	Datos        dominiobolsa.DatosOferta
	PublicadaEn  time.Time
	VenceAntesDe time.Time
	Estado       string
	Disposicion  *DisposicionPropiaPortal
}

type DisposicionPropiaPortal struct {
	Recibo        string
	ManifestadaEn time.Time
}

// DisposicionPortalCandidato llega a PostgreSQL con el material ya emitido.
type DisposicionPortalCandidato struct {
	OfertaRef, ReciboRef, CandidatoRef, Clave string
	ManifestadaEn                             time.Time
	Material                                  puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

type ReciboDisposicionPortal struct {
	Reutilizada          bool
	OfertaRef, ReciboRef string
	ManifestadaEn        time.Time
}

// RegistroDisposicionOferta consume la decisión y registra la disposición en
// una sola transacción.
type RegistroDisposicionOferta interface {
	ManifestarDisposicion(context.Context, DisposicionPortalCandidato) (ReciboDisposicionPortal, error)
}
