package bootstrap

import (
	"context"
	"errors"

	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

// Conserva el único PDP común. La captura procede de la misma evaluación que
// registró la decisión, no de una segunda lectura de asignación reconstruida.
func (a *autorizadorAnalisisContratacionTemporalDesarrollo) ExigirSolicitudLigadaV3ConCaptura(ctx context.Context,
	s core.SolicitudAutorizacionLigadaV3, r core.ResultadoContextoActorRegistradoV2) (core.DecisionAutorizacionLigadaV3,
	vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3, vp.CapturaEvaluacionSolicitudLigadaV3, error) {
	var d core.DecisionAutorizacionLigadaV3
	var c vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3
	if a == nil {
		return d, c, nil, errAutorizacionComunDesarrolloNoDisponible
	}
	comun, ok := a.delegado.(*autorizadorComunDesarrollo)
	if !ok || comun == nil || comun.servicio == nil {
		return d, c, nil, errAutorizacionComunDesarrolloNoDisponible
	}
	ligado, err := comun.contextoSolicitud(ctx, s)
	if err != nil {
		return d, c, nil, err
	}
	frontera, ok := fronteraSeguridadComunDesdeContexto(ligado)
	if !ok || r.Validar() != nil || !frontera.descriptor.admitePerfil(r.Contexto.PerfilActivoRef) {
		return d, c, nil, errors.Join(core.ErrAutorizacionDenegada, errAutorizacionComunDesarrolloNoDisponible)
	}
	return comun.servicio.ExigirSolicitudLigadaV3ConCaptura(ligado, s, r)
}

func (e *emisorMaterialRenovableCTDesarrollo) EmitirMaterialAutorizacionAtestadaV3ConCaptura(ctx context.Context,
	s core.SolicitudAutorizacionLigadaV3, r core.ResultadoContextoActorRegistradoV2) (core.DecisionAutorizacionLigadaV3,
	vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3, vp.ExportadorMaterialConsumoAutorizacionAtestadaV3, vp.CapturaEvaluacionSolicitudLigadaV3, error) {
	var d core.DecisionAutorizacionLigadaV3
	var c vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3
	if e == nil || e.proveedor == nil || ctx == nil || ctx.Err() != nil {
		return d, c, nil, nil, errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	snapshot, err := e.proveedor.instantaneaConfianza(ctx)
	if err != nil {
		return d, c, nil, nil, err
	}
	emisor, err := confianza.NuevoEmisorMaterialAutorizacionAtestadaV3(e.autoridad, e.proveedor.atestador, snapshot, e.proveedor.emisor)
	if err != nil {
		return d, c, nil, nil, err
	}
	return emisor.EmitirMaterialAutorizacionAtestadaV3ConCaptura(ctx, s, r)
}
