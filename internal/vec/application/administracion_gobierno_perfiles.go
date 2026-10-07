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
	r, err := s.proponerGobiernoPerfilConEstado(ctx, solicitud)
	if err != nil {
		return domain.PropuestaGobiernoPerfil{}, err
	}
	return r.Propuesta, nil
}

// proponerGobiernoPerfilConEstado conserva la preparación y el doble control
// comunes. Sólo el adaptador nominal de creación puede declarar un replay
// histórico con auditoría actual; todos los puertos anteriores siguen
// exigiendo propuesta con caducidad futura.
func (s *ServicioAdministracionPerfiles) proponerGobiernoPerfilConEstado(ctx context.Context, solicitud domain.SolicitudPropuestaGobiernoPerfil) (ports.ResultadoPropuestaGobiernoRolNuevo, error) {
	var vacio ports.ResultadoPropuestaGobiernoRolNuevo
	if s == nil || s.catalogo == nil || s.actos == nil || s.reloj == nil {
		return vacio, ErrAdministracionPerfilesNoConfigurada
	}
	if ctx == nil || solicitud.Validar() != nil || solicitud.Evidencia.ValidarEn(solicitud.Actor, s.reloj.Ahora()) != nil {
		return vacio, domain.ErrPlanGobiernoPerfilInvalido
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	if err := s.validarAdministrador(ctx, solicitud.InstantaneaAutorizacion); err != nil {
		return vacio, err
	}
	autoridad, ok := s.actos.(ports.AutoridadGobiernoPerfiles)
	if !ok {
		return vacio, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	catalogo, err := autoridad.ResolverCatalogoGobiernoPerfil(ctx, solicitud)
	if err != nil {
		return vacio, err
	}
	var plan domain.PlanGobiernoPerfil
	if _, recuperable := autoridad.(ports.AutoridadPropuestaGobiernoRolNuevoRecuperable); recuperable &&
		solicitud.Intencion.Operacion == domain.OperacionCrearPerfilGobernado {
		plan, _, err = domain.PrepararPlanGobiernoRolNuevoDesdeCatalogo(catalogo, solicitud.Intencion,
			s.reloj.Ahora(), solicitud.Actor.PersonaRef)
	} else {
		plan, err = domain.PrepararPlanGobiernoPerfil(catalogo, solicitud.Intencion, s.reloj.Ahora())
	}
	if err != nil {
		return vacio, err
	}
	orden := domain.OrdenPropuestaGobiernoPerfil{Solicitud: solicitud,
		Material: domain.MaterialPropuestaGobiernoPerfil{OperacionRef: solicitud.OperacionRef,
			ProponentePersonaRef: solicitud.Actor.PersonaRef, PerfilActivoRef: solicitud.Actor.PerfilActivoRef,
			AsignacionPerfilRef: solicitud.InstantaneaAutorizacion.AsignacionPerfil.Referencia(), Plan: plan}}
	if orden.Validar() != nil {
		return vacio, domain.ErrPlanGobiernoPerfilInvalido
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	paraPuerto := orden
	paraPuerto.Material.Plan = orden.Material.Plan.Copia()
	var resultado ports.ResultadoPropuestaGobiernoRolNuevo
	if especial, ok := autoridad.(ports.AutoridadPropuestaGobiernoRolNuevoRecuperable); ok &&
		plan.Operacion == domain.OperacionCrearPerfilGobernado {
		resultado, err = especial.ProponerGobiernoRolNuevoRecuperable(ctx, paraPuerto)
		if err == nil {
			err = resultado.ValidarPara(orden, s.reloj.Ahora())
		}
	} else {
		resultado.Propuesta, err = autoridad.ProponerGobiernoPerfil(ctx, paraPuerto)
		if err == nil && (resultado.Propuesta.ValidarPara(orden) != nil ||
			!resultado.Propuesta.CaducaEn.After(s.reloj.Ahora())) {
			err = domain.ErrPlanGobiernoPerfilInvalido
		}
	}
	if err != nil {
		return vacio, err
	}
	resultado.Propuesta.Material.Plan = resultado.Propuesta.Material.Plan.Copia()
	return resultado, nil
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
