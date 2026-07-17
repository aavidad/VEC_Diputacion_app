package application

import "vec-diputacion-granada/internal/vec/domain"

// proyectarSolicitudAutorizacionLigadaV2 adapta la capacidad nominal al motor
// RBAC/ABAC compartido como detalle privado. La proyeccion deriva identidad,
// perfil, metodo y garantia del vinculo autoritativo; nunca acepta Principal ni
// Motivo declarados por el llamador.
func proyectarSolicitudAutorizacionLigadaV2(
	solicitud domain.SolicitudAutorizacionLigadaV2,
) (domain.SolicitudAutorizacion, error) {
	datos, err := solicitud.Datos()
	if err != nil {
		return domain.SolicitudAutorizacion{}, domain.ErrSolicitudAutorizacionInvalida
	}
	vinculo, err := datos.VinculoAutenticacionActor.Datos()
	if err != nil {
		return domain.SolicitudAutorizacion{}, domain.ErrSolicitudAutorizacionInvalida
	}
	correlacionRef, err := datos.Correlacion.ValorCanonico()
	if err != nil {
		return domain.SolicitudAutorizacion{}, domain.ErrSolicitudAutorizacionInvalida
	}
	proyeccion := domain.SolicitudAutorizacion{
		Principal: domain.Principal{
			ID: vinculo.PrincipalID, AuthMethod: vinculo.MetodoObservado,
			AuthAssurance: vinculo.GarantiaObservada,
		},
		PerfilActivoRef: vinculo.PerfilActivoRef,
		ContextoActor:   datos.ContextoActor, VinculoAutenticacionActor: datos.VinculoAutenticacionActor,
		ReferenciaMotivo: datos.ReferenciaMotivo,
		Accion:           datos.Accion, Recurso: datos.Recurso, Finalidad: datos.Finalidad,
		CorrelacionRef: correlacionRef, Motivo: datos.ReferenciaMotivo.EntradaClave,
	}
	if proyeccion.ValidarVinculoAutenticacionActor() != nil ||
		!domain.ReferenciaMotivoAutorizacionV2Valida(proyeccion.ReferenciaMotivo) ||
		!domain.ReferenciaCorrelacionAutorizacionV2Valida(proyeccion.CorrelacionRef) {
		return domain.SolicitudAutorizacion{}, domain.ErrSolicitudAutorizacionInvalida
	}
	return proyeccion, nil
}
