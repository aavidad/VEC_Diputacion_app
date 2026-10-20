package bootstrap

import (
	"context"

	docpg "vec-diputacion-granada/internal/vec/documentos/adapters/postgres"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// Documentos conserva su registrador nominal si el despachador de rutas
// exactas rechaza una capacidad después de la frontera propia.
type registradorFronterasConDocumentos struct {
	delegado   vecports.RegistradorAuditoriaFronteraRutaExacta
	documentos *autoridadDocumentosDesarrollo
}

func (r registradorFronterasConDocumentos) RegistrarAuditoriaFronteraRutaExacta(
	ctx context.Context, orden vecports.OrdenAuditoriaFronteraRutaExacta,
) error {
	if !esRutaDocumentos(orden.Ruta) {
		if r.delegado == nil {
			return ErrComposicionDocumentosNoDisponible
		}
		return r.delegado.RegistrarAuditoriaFronteraRutaExacta(ctx, orden)
	}
	if r.documentos == nil || r.documentos.registrador == nil || !r.documentos.publicadas[orden.Ruta] {
		return ErrComposicionDocumentosNoDisponible
	}
	motivo := ""
	switch orden.Motivo {
	case vecports.MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida:
		motivo = docpg.MotivoFronteraAutenticacion
	case vecports.MotivoAuditoriaFronteraRutaExactaAccesoDenegado:
		motivo = docpg.MotivoFronteraDenegado
	case vecports.MotivoAuditoriaFronteraRutaExactaServicioNoDisponible:
		motivo = docpg.MotivoFronteraDependencia
	default:
		return ErrComposicionDocumentosNoDisponible
	}
	return r.documentos.registrador.RegistrarDenegacion(ctx, docpg.OrdenDenegacionFrontera{
		CorrelacionRef: orden.CorrelacionRef, Motivo: motivo, Ruta: orden.Ruta,
		Metodo: "otro", ActorRef: orden.ActorRef,
	})
}
