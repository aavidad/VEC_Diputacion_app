package application

import (
	"context"
	"reflect"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

var _ ports.ServicioVersionarRolBolsaExterna = (*ServicioAdministracionPerfiles)(nil)

// NuevoServicioVersionarRolBolsaExterna monta la autoridad nominal sobre el
// servicio ADMIN existente y falla cerrado ante dependencias ausentes.
func NuevoServicioVersionarRolBolsaExterna(a ports.AutoridadCompletaVersionarRolBolsaExterna,
	reloj ports.Reloj) (*ServicioAdministracionPerfiles, error) {
	if a == nil || reloj == nil {
		return nil, ErrAdministracionPerfilesNoConfigurada
	}
	v := reflect.ValueOf(a)
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Func, reflect.Map, reflect.Slice, reflect.Chan:
		if v.IsNil() {
			return nil, ErrAdministracionPerfilesNoConfigurada
		}
	}
	return NuevoServicioAdministracionPerfiles(a, a, reloj)
}

// ProponerVersionarRolBolsaExterna prepara las dos concesiones de preferencias entre bolsas desde el catálogo central.
// Las asignaciones solicitadas son preimágenes CAS sin autoridad; el puerto
// nominal coteja cada documento real dentro de su transacción.
func (s *ServicioAdministracionPerfiles) ProponerVersionarRolBolsaExterna(ctx context.Context,
	solicitud domain.SolicitudPropuestaVersionarRolBolsaExterna) (ports.ResultadoPropuestaVersionarRolBolsaExterna, error) {
	var vacio ports.ResultadoPropuestaVersionarRolBolsaExterna
	if s == nil || s.catalogo == nil || s.actos == nil || s.reloj == nil {
		return vacio, ports.ConClaseVersionBolsa("servicio_no_configurado", ErrAdministracionPerfilesNoConfigurada)
	}
	if ctx == nil || solicitud.Validar() != nil {
		return vacio, ports.ConClaseVersionBolsa("solicitud_invalida", domain.ErrVersionarRolBolsaExternaInvalido)
	}
	if solicitud.Evidencia.ValidarEn(solicitud.Actor, s.reloj.Ahora()) != nil {
		return vacio, ports.ConClaseVersionBolsa("evidencia_sesion", domain.ErrVersionarRolBolsaExternaInvalido)
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	autoridad, ok := s.actos.(ports.AutoridadVersionarRolBolsaExterna)
	if !ok {
		return vacio, ports.ConClaseVersionBolsa("autoridad_sin_puerto", ports.ErrAutoridadAdministracionPerfilesNoDisponible)
	}
	if err := s.validarAdministrador(ctx, solicitud.InstantaneaAutorizacion); err != nil {
		return vacio, err
	}
	fuentes, err := autoridad.ResolverFuentesVersionarRolBolsaExterna(ctx, solicitud)
	if err != nil {
		return vacio, err
	}
	plan, err := domain.ReconstruirPlanVersionarRolBolsaExterna(fuentes.Catalogo, fuentes.Base, solicitud.Intencion, s.reloj.Ahora())
	if err != nil {
		return vacio, ports.ConClaseVersionBolsa("plan_servicio", err)
	}
	orden := domain.OrdenPropuestaVersionarRolBolsaExterna{Solicitud: solicitud,
		Material: domain.MaterialPropuestaVersionarRolBolsaExterna{
			OperacionRef: solicitud.OperacionRef, ProponentePersonaRef: solicitud.Actor.PersonaRef,
			PerfilActivoRef:     solicitud.Actor.PerfilActivoRef,
			AsignacionPerfilRef: solicitud.InstantaneaAutorizacion.AsignacionPerfil.Referencia(), Plan: plan}}
	if orden.Validar() != nil {
		return vacio, ports.ConClaseVersionBolsa("orden_invalida", domain.ErrVersionarRolBolsaExternaInvalido)
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	paraPuerto := orden
	paraPuerto.Material = orden.Material.Copia()
	resultado, err := autoridad.ProponerVersionarRolBolsaExterna(ctx, paraPuerto)
	if err != nil {
		return vacio, err
	}
	if resultado.ValidarPara(orden, s.reloj.Ahora()) != nil {
		return vacio, ports.ConClaseVersionBolsa("resultado_invalido", domain.ErrVersionarRolBolsaExternaInvalido)
	}
	resultado.Propuesta.Material = resultado.Propuesta.Material.Copia()
	return resultado, nil
}

// CerrarVersionarRolBolsaExternaPorReferencia no completa el plan desde el cliente.
// La autoridad recupera el material inmutable bajo la decisión V3 y devuelve
// el recibo del único acto que publica rol y avanza las asignaciones elegidas.
func (s *ServicioAdministracionPerfiles) CerrarVersionarRolBolsaExternaPorReferencia(ctx context.Context,
	solicitud domain.SolicitudCierreVersionarRolBolsaExterna) (domain.CierreVersionarRolBolsaExterna, error) {
	var vacio domain.CierreVersionarRolBolsaExterna
	if s == nil || s.catalogo == nil || s.actos == nil || s.reloj == nil {
		return vacio, ErrAdministracionPerfilesNoConfigurada
	}
	if ctx == nil || solicitud.Validar() != nil ||
		solicitud.Evidencia.ValidarEn(solicitud.Aprobador, s.reloj.Ahora()) != nil {
		return vacio, domain.ErrVersionarRolBolsaExternaInvalido
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	autoridad, ok := s.actos.(ports.AutoridadVersionarRolBolsaExterna)
	if !ok {
		return vacio, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	if err := s.validarAdministrador(ctx, solicitud.InstantaneaAutorizacion); err != nil {
		return vacio, err
	}
	cierre, err := autoridad.CerrarVersionarRolBolsaExterna(ctx, solicitud)
	if err != nil {
		return vacio, err
	}
	if cierre.ValidarPara(solicitud) != nil || cierre.ConfirmadoEn.After(s.reloj.Ahora()) {
		return vacio, domain.ErrVersionarRolBolsaExternaInvalido
	}
	return cierre.Copia(), nil
}
