package application

import (
	"context"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// ProponerVersionarRolBolsaB103 prepara las cuatro concesiones RRHH desde el catálogo central.
// Las asignaciones solicitadas son preimágenes CAS sin autoridad; el puerto
// nominal coteja cada documento real dentro de su transacción.
func (s *ServicioAdministracionPerfiles) ProponerVersionarRolBolsaB103(ctx context.Context,
	solicitud domain.SolicitudPropuestaVersionarRolBolsaB103) (ports.ResultadoPropuestaVersionarRolBolsaB103, error) {
	var vacio ports.ResultadoPropuestaVersionarRolBolsaB103
	if s == nil || s.catalogo == nil || s.actos == nil || s.reloj == nil {
		return vacio, ports.ConClaseVersionBolsa("servicio_no_configurado", ErrAdministracionPerfilesNoConfigurada)
	}
	if ctx == nil || solicitud.Validar() != nil {
		return vacio, ports.ConClaseVersionBolsa("solicitud_invalida", domain.ErrVersionarRolBolsaB103Invalido)
	}
	if solicitud.Evidencia.ValidarEn(solicitud.Actor, s.reloj.Ahora()) != nil {
		return vacio, ports.ConClaseVersionBolsa("evidencia_sesion", domain.ErrVersionarRolBolsaB103Invalido)
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	autoridad, ok := s.actos.(ports.AutoridadVersionarRolBolsaB103)
	if !ok {
		return vacio, ports.ConClaseVersionBolsa("autoridad_sin_puerto", ports.ErrAutoridadAdministracionPerfilesNoDisponible)
	}
	if err := s.validarAdministrador(ctx, solicitud.InstantaneaAutorizacion); err != nil {
		return vacio, err
	}
	catalogo, err := autoridad.ResolverCatalogoVersionarRolBolsaB103(ctx, solicitud)
	if err != nil {
		return vacio, err
	}
	plan, err := domain.PrepararPlanVersionarRolBolsaB103(catalogo, solicitud.Intencion, s.reloj.Ahora())
	if err != nil {
		return vacio, ports.ConClaseVersionBolsa("plan_servicio", err)
	}
	orden := domain.OrdenPropuestaVersionarRolBolsaB103{Solicitud: solicitud,
		Material: domain.MaterialPropuestaVersionarRolBolsaB103{
			OperacionRef: solicitud.OperacionRef, ProponentePersonaRef: solicitud.Actor.PersonaRef,
			PerfilActivoRef:     solicitud.Actor.PerfilActivoRef,
			AsignacionPerfilRef: solicitud.InstantaneaAutorizacion.AsignacionPerfil.Referencia(), Plan: plan}}
	if orden.Validar() != nil {
		return vacio, ports.ConClaseVersionBolsa("orden_invalida", domain.ErrVersionarRolBolsaB103Invalido)
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	paraPuerto := orden
	paraPuerto.Material = orden.Material.Copia()
	resultado, err := autoridad.ProponerVersionarRolBolsaB103(ctx, paraPuerto)
	if err != nil {
		return vacio, err
	}
	if resultado.ValidarPara(orden, s.reloj.Ahora()) != nil {
		return vacio, ports.ConClaseVersionBolsa("resultado_invalido", domain.ErrVersionarRolBolsaB103Invalido)
	}
	resultado.Propuesta.Material = resultado.Propuesta.Material.Copia()
	return resultado, nil
}

// CerrarVersionarRolBolsaB103PorReferencia no completa el plan desde el cliente.
// La autoridad recupera el material inmutable bajo la decisión V3 y devuelve
// el recibo del único acto que publica rol y avanza las asignaciones elegidas.
func (s *ServicioAdministracionPerfiles) CerrarVersionarRolBolsaB103PorReferencia(ctx context.Context,
	solicitud domain.SolicitudCierreVersionarRolBolsaB103) (domain.CierreVersionarRolBolsaB103, error) {
	var vacio domain.CierreVersionarRolBolsaB103
	if s == nil || s.catalogo == nil || s.actos == nil || s.reloj == nil {
		return vacio, ErrAdministracionPerfilesNoConfigurada
	}
	if ctx == nil || solicitud.Validar() != nil ||
		solicitud.Evidencia.ValidarEn(solicitud.Aprobador, s.reloj.Ahora()) != nil {
		return vacio, domain.ErrVersionarRolBolsaB103Invalido
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	autoridad, ok := s.actos.(ports.AutoridadVersionarRolBolsaB103)
	if !ok {
		return vacio, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	if err := s.validarAdministrador(ctx, solicitud.InstantaneaAutorizacion); err != nil {
		return vacio, err
	}
	cierre, err := autoridad.CerrarVersionarRolBolsaB103(ctx, solicitud)
	if err != nil {
		return vacio, err
	}
	if cierre.ValidarPara(solicitud) != nil || cierre.ConfirmadoEn.After(s.reloj.Ahora()) {
		return vacio, domain.ErrVersionarRolBolsaB103Invalido
	}
	return cierre.Copia(), nil
}
