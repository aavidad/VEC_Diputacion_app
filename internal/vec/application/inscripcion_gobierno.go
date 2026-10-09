package application

import (
	"context"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// ProponerVersionInscripcion prepara una única versión desde el catálogo
// central y entrega su material al puerto durable. AUT68 reconstruye fuente,
// actor y preimagen dentro de la transacción antes de guardar la propuesta.
func (s *ServicioAdministracionPerfiles) ProponerVersionInscripcion(
	ctx context.Context, solicitud domain.SolicitudPropuestaVersionInscripcion,
) (domain.PropuestaVersionInscripcion, bool, error) {
	var vacia domain.PropuestaVersionInscripcion
	if s == nil || s.catalogo == nil || s.actos == nil || s.reloj == nil {
		return vacia, false, ErrAdministracionPerfilesNoConfigurada
	}
	if ctx == nil || solicitud.Validar() != nil ||
		solicitud.Evidencia.ValidarEn(solicitud.Actor, s.reloj.Ahora()) != nil {
		return vacia, false, domain.ErrPlanVersionInscripcionInvalido
	}
	if err := ctx.Err(); err != nil {
		return vacia, false, err
	}
	if err := s.validarAdministrador(ctx, solicitud.InstantaneaAutorizacion); err != nil {
		return vacia, false, err
	}
	autoridad, ok := s.actos.(ports.AutoridadVersionInscripcion)
	if !ok {
		return vacia, false, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	catalogo, err := autoridad.ResolverCatalogoVersionInscripcion(ctx, solicitud.Plan)
	if err != nil || solicitud.Plan.ValidarFuenteHistorica(catalogo, s.reloj.Ahora()) != nil {
		return vacia, false, domain.ErrPlanVersionInscripcionInvalido
	}
	material := domain.MaterialPropuestaVersionInscripcion{
		OperacionRef: solicitud.OperacionRef, ProponentePersonaRef: solicitud.Actor.PersonaRef,
		PerfilActivoRef:     solicitud.Actor.PerfilActivoRef,
		AsignacionPerfilRef: solicitud.InstantaneaAutorizacion.AsignacionPerfil.Referencia(),
		Plan:                solicitud.Plan,
	}
	orden := domain.OrdenPropuestaVersionInscripcion{Solicitud: solicitud, Material: material}
	if orden.Validar() != nil {
		return vacia, false, domain.ErrPlanVersionInscripcionInvalido
	}
	if err := ctx.Err(); err != nil {
		return vacia, false, err
	}
	propuesta, replay, err := autoridad.ProponerVersionInscripcion(ctx, orden)
	if err != nil {
		return vacia, false, err
	}
	if propuesta.ValidarPara(orden) != nil || (!replay && !propuesta.CaducaEn.After(s.reloj.Ahora())) {
		return vacia, false, domain.ErrPlanVersionInscripcionInvalido
	}
	return propuesta, replay, nil
}

func (s *ServicioAdministracionPerfiles) CerrarVersionInscripcion(
	ctx context.Context, solicitud domain.SolicitudCierreVersionInscripcion,
) (domain.CierreVersionInscripcion, bool, error) {
	var vacio domain.CierreVersionInscripcion
	if s == nil || s.catalogo == nil || s.actos == nil || s.reloj == nil {
		return vacio, false, ErrAdministracionPerfilesNoConfigurada
	}
	if ctx == nil || solicitud.Validar() != nil ||
		solicitud.Evidencia.ValidarEn(solicitud.Aprobador, s.reloj.Ahora()) != nil {
		return vacio, false, domain.ErrPlanVersionInscripcionInvalido
	}
	if err := ctx.Err(); err != nil {
		return vacio, false, err
	}
	if err := s.validarAdministrador(ctx, solicitud.InstantaneaAutorizacion); err != nil {
		return vacio, false, err
	}
	autoridad, ok := s.actos.(ports.AutoridadVersionInscripcion)
	if !ok {
		return vacio, false, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vacio, false, err
	}
	cierre, replay, err := autoridad.CerrarVersionInscripcion(ctx, solicitud)
	if err != nil {
		return vacio, false, err
	}
	if cierre.ValidarPara(solicitud) != nil || cierre.ConfirmadoEn.After(s.reloj.Ahora()) {
		return vacio, false, domain.ErrPlanVersionInscripcionInvalido
	}
	return cierre, replay, nil
}
