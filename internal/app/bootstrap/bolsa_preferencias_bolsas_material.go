package bootstrap

import (
	"context"

	bolsapuertos "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

// El emisor de la escritura reutiliza el material gobernado del portal
// exterior. La lectura propia no emite ni consume material V3.
type proveedorPreferenciasBolsasPortalExterno struct {
	delegado *proveedorMaterialAltaContratacionTemporalDesarrollo
}

var _ bolsapuertos.ProveedorMaterialPreferenciasBolsas = proveedorPreferenciasBolsasPortalExterno{}

type proveedorPreferenciasBolsasInterna struct {
	delegado *proveedorMaterialAltaContratacionTemporalDesarrollo
}

var _ bolsapuertos.ProveedorMaterialPreferenciasBolsas = proveedorPreferenciasBolsasInterna{}

func (p proveedorPreferenciasBolsasInterna) EmitirMaterialPreferenciasBolsas(ctx context.Context,
	solicitud dominiovec.SolicitudAutorizacionLigadaV3, resultado dominiovec.ResultadoContextoActorRegistradoV2,
	decision dominiovec.DecisionAutorizacionLigadaV3,
	confirmacion puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3,
) (puertosvec.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	if p.delegado == nil {
		return nil, errMiBolsaNoDisponible
	}
	return emitirMaterialNominalMiBolsaDesarrollo(ctx, p.delegado, motivoPortalMiBolsaDesarrollo(),
		solicitud, resultado, decision, confirmacion)
}

func (p proveedorPreferenciasBolsasPortalExterno) EmitirMaterialPreferenciasBolsas(ctx context.Context,
	solicitud dominiovec.SolicitudAutorizacionLigadaV3, resultado dominiovec.ResultadoContextoActorRegistradoV2,
	decision dominiovec.DecisionAutorizacionLigadaV3,
	confirmacion puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3,
) (puertosvec.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	if p.delegado == nil {
		return nil, errMiBolsaNoDisponible
	}
	return emitirMaterialNominalMiBolsaDesarrollo(ctx, p.delegado, motivoPortalMiBolsaDesarrollo(),
		solicitud, resultado, decision, confirmacion)
}
