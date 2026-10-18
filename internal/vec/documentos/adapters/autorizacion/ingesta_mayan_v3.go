package autorizacion

import (
	"context"

	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// EmisorOperacionIngestaMayan usa la identidad y el perfil del contexto
// autenticado de la peticion. Ningun campo del plan concede una capacidad.
type EmisorOperacionIngestaMayan interface {
	AutorizarOperacionIngestaMayan(context.Context, string, []byte, string, string) (docports.AutorizacionV3, error)
}

type AutoridadIngestaMayanV3 struct {
	emisor EmisorOperacionIngestaMayan
	reloj  vecports.Reloj
}

func NuevaAutoridadIngestaMayanV3(emisor EmisorOperacionIngestaMayan, reloj vecports.Reloj) (*AutoridadIngestaMayanV3, error) {
	if nulo(emisor) || nulo(reloj) {
		return nil, vecports.ErrAutorizacionAlmacenInvalida
	}
	return &AutoridadIngestaMayanV3{emisor: emisor, reloj: reloj}, nil
}

func (a *AutoridadIngestaMayanV3) AutorizarIngestaMayan(ctx context.Context, accion string, preimagen []byte, planRef, expedienteRef string) (docports.AutorizacionV3, error) {
	if a == nil || nulo(a.emisor) || nulo(a.reloj) || ctx == nil || ctx.Err() != nil ||
		len(preimagen) == 0 || len(preimagen) > 16384 || planRef == "" || expedienteRef == "" ||
		(accion != docports.AccionReservarRecursoMayan && accion != docports.AccionConsultarReservaMayan &&
			accion != docports.AccionAbandonarRecursoMayan &&
			accion != docports.AccionPrepararIngestaMayan && accion != docports.AccionDespacharIngestaMayan &&
			accion != docports.AccionResolverIngestaMayan && accion != docports.AccionConciliarIngestaMayan &&
			accion != docports.AccionConsultarIngestaMayan) {
		return docports.AutorizacionV3{}, denegadoPor(contextoCancelado(ctx))
	}
	concesion, err := a.emisor.AutorizarOperacionIngestaMayan(ctx, accion, preimagen, planRef, expedienteRef)
	if err != nil {
		return docports.AutorizacionV3{}, denegadoPor(err)
	}
	if concesion.ValidarParaIngestaMayan(accion, planRef, expedienteRef, a.reloj.Ahora().UTC()) != nil ||
		concesion.Material.ResumenCapacidad().EfectoHuellaSHA256() != docports.HuellaEfectoV3(preimagen) {
		return docports.AutorizacionV3{}, denegadoPor(nil)
	}
	return concesion, nil
}

var _ docports.AutoridadIngestaMayan = (*AutoridadIngestaMayanV3)(nil)
