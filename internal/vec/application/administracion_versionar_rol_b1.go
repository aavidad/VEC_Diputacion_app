package application

import (
	"context"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// ProponerVersionarRolBolsa prepara la concesión B1 desde el catálogo central.
// Las asignaciones solicitadas son preimágenes CAS sin autoridad; el puerto
// nominal coteja cada documento real dentro de su transacción.
func (s *ServicioAdministracionPerfiles) ProponerVersionarRolBolsa(ctx context.Context,
	solicitud domain.SolicitudPropuestaVersionarRolBolsa) (ports.ResultadoPropuestaVersionarRolBolsa, error) {
	var vacio ports.ResultadoPropuestaVersionarRolBolsa
	if s == nil || s.catalogo == nil || s.actos == nil || s.reloj == nil {
		return vacio, ErrAdministracionPerfilesNoConfigurada
	}
	if ctx == nil || solicitud.Validar() != nil ||
		solicitud.Evidencia.ValidarEn(solicitud.Actor, s.reloj.Ahora()) != nil {
		return vacio, domain.ErrVersionarRolBolsaInvalido
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	autoridad, ok := s.actos.(ports.AutoridadVersionarRolBolsa)
	if !ok {
		return vacio, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	if err := s.validarAdministrador(ctx, solicitud.InstantaneaAutorizacion); err != nil {
		return vacio, err
	}
	catalogo, err := autoridad.ResolverCatalogoVersionarRolBolsa(ctx, solicitud)
	if err != nil {
		return vacio, err
	}
	plan, err := domain.PrepararPlanVersionarRolBolsa(catalogo, solicitud.Intencion, s.reloj.Ahora())
	if err != nil {
		return vacio, err
	}
	orden := domain.OrdenPropuestaVersionarRolBolsa{Solicitud: solicitud,
		Material: domain.MaterialPropuestaVersionarRolBolsa{
			OperacionRef: solicitud.OperacionRef, ProponentePersonaRef: solicitud.Actor.PersonaRef,
			PerfilActivoRef:     solicitud.Actor.PerfilActivoRef,
			AsignacionPerfilRef: solicitud.InstantaneaAutorizacion.AsignacionPerfil.Referencia(), Plan: plan}}
	if orden.Validar() != nil {
		return vacio, domain.ErrVersionarRolBolsaInvalido
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	paraPuerto := orden
	paraPuerto.Material = orden.Material.Copia()
	resultado, err := autoridad.ProponerVersionarRolBolsa(ctx, paraPuerto)
	if err != nil {
		return vacio, err
	}
	if resultado.ValidarPara(orden, s.reloj.Ahora()) != nil {
		return vacio, domain.ErrVersionarRolBolsaInvalido
	}
	resultado.Propuesta.Material = resultado.Propuesta.Material.Copia()
	return resultado, nil
}

// CerrarVersionarRolBolsaPorReferencia no completa el plan desde el cliente.
// La autoridad recupera el material inmutable bajo la decisión V3 y devuelve
// el recibo del único acto que publica rol y avanza las asignaciones elegidas.
func (s *ServicioAdministracionPerfiles) CerrarVersionarRolBolsaPorReferencia(ctx context.Context,
	solicitud domain.SolicitudCierreVersionarRolBolsa) (domain.CierreVersionarRolBolsa, error) {
	var vacio domain.CierreVersionarRolBolsa
	if s == nil || s.catalogo == nil || s.actos == nil || s.reloj == nil {
		return vacio, ErrAdministracionPerfilesNoConfigurada
	}
	if ctx == nil || solicitud.Validar() != nil ||
		solicitud.Evidencia.ValidarEn(solicitud.Aprobador, s.reloj.Ahora()) != nil {
		return vacio, domain.ErrVersionarRolBolsaInvalido
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	autoridad, ok := s.actos.(ports.AutoridadVersionarRolBolsa)
	if !ok {
		return vacio, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	if err := s.validarAdministrador(ctx, solicitud.InstantaneaAutorizacion); err != nil {
		return vacio, err
	}
	cierre, err := autoridad.CerrarVersionarRolBolsa(ctx, solicitud)
	if err != nil {
		return vacio, err
	}
	if cierre.ValidarPara(solicitud) != nil || cierre.ConfirmadoEn.After(s.reloj.Ahora()) {
		return vacio, domain.ErrVersionarRolBolsaInvalido
	}
	return cierre.Copia(), nil
}
