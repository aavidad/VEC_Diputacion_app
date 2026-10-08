package constitucion

import (
	"context"
	"time"

	importacion "vec-diputacion-granada/internal/modules/bolsa/domain/importacionconvoca"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

// ServicioAutorizado prepara la constitución del lote B1 en memoria. El
// repositorio confirma acta, original cifrado, bolsa, vínculos y consumo V3
// en una sola transacción.
type ServicioAutorizado struct {
	derivador  DerivadorCandidato
	reloj      Reloj
	autorizado ports.RepositorioConstitucionCargaConvoca
}

func NuevoServicioAutorizado(derivador DerivadorCandidato, reloj Reloj, autorizado ports.RepositorioConstitucionCargaConvoca) (*ServicioAutorizado, error) {
	if derivador == nil || reloj == nil || autorizado == nil {
		return nil, ErrDependenciasRequeridas
	}
	return &ServicioAutorizado{derivador: derivador, reloj: reloj, autorizado: autorizado}, nil
}

// Constituir construye la constitución desde el lote y la confirma consumiendo
// el material. El actor es el titular de la decisión.
// El mismo lote que se previsualizó llega al repositorio, sin una lectura de
// staging intermedia ni escrituras previas a la autorización.
func (s *ServicioAutorizado) Constituir(ctx context.Context, lote importacion.LoteValidado, solicitud Solicitud, original ports.OriginalProtegidoCargaConvoca, material puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.ReciboCargaConvoca, error) {
	if ctx == nil || s == nil || s.derivador == nil || s.reloj == nil {
		return ports.ReciboCargaConvoca{}, ErrDependenciasRequeridas
	}
	if solicitud.HuellaFicheroSHA256 == "" || solicitud.CategoriaRef == "" || solicitud.ActorRef == "" || material.ValidarEstructura() != nil ||
		lote.Validar() != nil || lote.Acta.HuellaFicheroSHA256 != solicitud.HuellaFicheroSHA256 || lote.Acta.CategoriaRef != solicitud.CategoriaRef ||
		original.Referencia != lote.Acta.FicheroCustodiadoRef {
		return ports.ReciboCargaConvoca{}, ports.ErrConstitucionBolsaInvalida
	}
	if lote.Acta.BolsaRef != "" && lote.Acta.BolsaRef != bolsaRefDerivadaActa(lote.Acta.ActaRef, lote.Acta.CategoriaRef) {
		return ports.ReciboCargaConvoca{}, ports.ErrConstitucionBolsaInvalida
	}
	ahora := s.reloj().UTC().Truncate(time.Microsecond)
	constitucion, vinculos, pendientes, err := construirConstitucion(lote, solicitud.ActorRef, ahora, s.derivador)
	if err != nil {
		return ports.ReciboCargaConvoca{}, err
	}
	// En B1 la web deja BolsaRef vacía: la constitución deriva una referencia
	// estable del acta. El acta que recibe SQL debe llevar esa misma referencia.
	// Una referencia explícita distinta se rechaza antes de cualquier escritura.
	if lote.Acta.BolsaRef != "" && lote.Acta.BolsaRef != constitucion.Bolsa.BolsaRef {
		return ports.ReciboCargaConvoca{}, ports.ErrConstitucionBolsaInvalida
	}
	lote.Acta.BolsaRef = constitucion.Bolsa.BolsaRef
	if lote.Validar() != nil {
		return ports.ReciboCargaConvoca{}, ports.ErrConstitucionBolsaInvalida
	}
	recibo, err := s.autorizado.ConfirmarCargaConvocaAutorizada(ctx, lote, constitucion, vinculos, original, material)
	if err != nil {
		return ports.ReciboCargaConvoca{}, err
	}
	recibo.PendientesRevision = pendientes
	return recibo, nil
}
