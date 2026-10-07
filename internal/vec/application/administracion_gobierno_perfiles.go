package application

import (
	"context"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// PrepararPlanGobiernoPerfilOffline es el consumidor común de preparación.
// Admite archivos sin tratarlos como fuente confiable. No construye contexto,
// autorización, proveedor ni un rol publicado: sólo devuelve material verificable.
func PrepararPlanGobiernoPerfilOffline(c domain.CatalogoAccionesAdministracionV1, s domain.SolicitudPlanGobiernoPerfil, instante time.Time) (domain.PlanGobiernoPerfil, error) {
	return domain.PrepararPlanGobiernoPerfil(c, s, instante)
}

func (s *ServicioAdministracionPerfiles) ProponerGobiernoPerfil(ctx context.Context, solicitud domain.SolicitudPropuestaGobiernoPerfil) (domain.PropuestaGobiernoPerfil, error) {
	if s == nil || s.catalogo == nil || s.actos == nil || s.reloj == nil {
		return domain.PropuestaGobiernoPerfil{}, ErrAdministracionPerfilesNoConfigurada
	}
	if ctx == nil || solicitud.Validar() != nil || solicitud.Evidencia.ValidarEn(solicitud.Actor, s.reloj.Ahora()) != nil {
		return domain.PropuestaGobiernoPerfil{}, domain.ErrPlanGobiernoPerfilInvalido
	}
	if err := ctx.Err(); err != nil {
		return domain.PropuestaGobiernoPerfil{}, err
	}
	if err := s.validarAdministrador(ctx, solicitud.InstantaneaAutorizacion); err != nil {
		return domain.PropuestaGobiernoPerfil{}, err
	}
	autoridad, ok := s.actos.(ports.AutoridadGobiernoPerfiles)
	if !ok {
		return domain.PropuestaGobiernoPerfil{}, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	catalogo, err := autoridad.ResolverCatalogoGobiernoPerfil(ctx, solicitud)
	if err != nil {
		return domain.PropuestaGobiernoPerfil{}, err
	}
	plan, err := domain.PrepararPlanGobiernoPerfil(catalogo, solicitud.Intencion, s.reloj.Ahora())
	if err != nil {
		return domain.PropuestaGobiernoPerfil{}, err
	}
	orden := domain.OrdenPropuestaGobiernoPerfil{Solicitud: solicitud,
		Material: domain.MaterialPropuestaGobiernoPerfil{OperacionRef: solicitud.OperacionRef,
			ProponentePersonaRef: solicitud.Actor.PersonaRef, PerfilActivoRef: solicitud.Actor.PerfilActivoRef,
			AsignacionPerfilRef: solicitud.InstantaneaAutorizacion.AsignacionPerfil.Referencia(), Plan: plan}}
	if orden.Validar() != nil {
		return domain.PropuestaGobiernoPerfil{}, domain.ErrPlanGobiernoPerfilInvalido
	}
	if err := ctx.Err(); err != nil {
		return domain.PropuestaGobiernoPerfil{}, err
	}
	paraPuerto := orden
	paraPuerto.Material.Plan = orden.Material.Plan.Copia()
	propuesta, err := autoridad.ProponerGobiernoPerfil(ctx, paraPuerto)
	if err != nil {
		return domain.PropuestaGobiernoPerfil{}, err
	}
	if propuesta.ValidarPara(orden) != nil || !propuesta.CaducaEn.After(s.reloj.Ahora()) {
		return domain.PropuestaGobiernoPerfil{}, domain.ErrPlanGobiernoPerfilInvalido
	}
	propuesta.Material.Plan = propuesta.Material.Plan.Copia()
	return propuesta, nil
}

func (s *ServicioAdministracionPerfiles) CerrarGobiernoPerfil(ctx context.Context, solicitud domain.SolicitudCierreGobiernoPerfil) (domain.CierreGobiernoPerfil, error) {
	if s == nil || s.catalogo == nil || s.actos == nil || s.reloj == nil {
		return domain.CierreGobiernoPerfil{}, ErrAdministracionPerfilesNoConfigurada
	}
	if ctx == nil || solicitud.Validar() != nil || solicitud.Evidencia.ValidarEn(solicitud.Aprobador, s.reloj.Ahora()) != nil {
		return domain.CierreGobiernoPerfil{}, domain.ErrPlanGobiernoPerfilInvalido
	}
	if err := ctx.Err(); err != nil {
		return domain.CierreGobiernoPerfil{}, err
	}
	if err := s.validarAdministrador(ctx, solicitud.InstantaneaAutorizacion); err != nil {
		return domain.CierreGobiernoPerfil{}, err
	}
	autoridad, ok := s.actos.(ports.AutoridadGobiernoPerfiles)
	if !ok {
		return domain.CierreGobiernoPerfil{}, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return domain.CierreGobiernoPerfil{}, err
	}
	cierre, err := autoridad.CerrarGobiernoPerfil(ctx, solicitud)
	if err != nil {
		return domain.CierreGobiernoPerfil{}, err
	}
	if cierre.ValidarPara(solicitud) != nil || cierre.ConfirmadoEn.After(s.reloj.Ahora()) {
		return domain.CierreGobiernoPerfil{}, domain.ErrPlanGobiernoPerfilInvalido
	}
	cierre.Material.Plan = cierre.Material.Plan.Copia()
	if cierre.Recibo != nil {
		r := cierre.Recibo.Copia()
		cierre.Recibo = &r
	}
	return cierre, nil
}
