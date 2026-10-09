package application

import (
	"context"
	"reflect"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// ServicioGobiernoInscripcion coordina el canal ADMIN de esta capacidad.
// El rol/permiso de ADMIN es una restricción local; la autoridad central SQL
// comprueba categoría, concesión y dos personas vigentes bajo bloqueo.
type ServicioGobiernoInscripcion struct {
	autoridad ports.AutoridadVersionInscripcion
	reloj     ports.Reloj
}

func NuevoServicioGobiernoInscripcion(a ports.AutoridadVersionInscripcion, reloj ports.Reloj) (*ServicioGobiernoInscripcion, error) {
	if interfazGobiernoInscripcionNula(a) || interfazGobiernoInscripcionNula(reloj) {
		return nil, ErrAdministracionPerfilesNoConfigurada
	}
	return &ServicioGobiernoInscripcion{autoridad: a, reloj: reloj}, nil
}

func interfazGobiernoInscripcionNula(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return r.IsNil()
	default:
		return false
	}
}

func (s *ServicioGobiernoInscripcion) comprobarAdmin(i domain.InstantaneaAutorizacion) bool {
	if s == nil || s.reloj == nil || i.Validar() != nil {
		return false
	}
	return i.VersionRol.RolID == "administracion_perfiles" &&
		i.VersionRol.Estado == domain.EstadoVersionRolPublicada &&
		i.ControlVigenciaVersionRol.Estado == domain.EstadoControlVigenciaVersionRolHabilitada &&
		i.AsignacionPerfil.VigenteEn(s.reloj.Ahora())
}

func (s *ServicioGobiernoInscripcion) ProponerVersionInscripcion(
	ctx context.Context, solicitud domain.SolicitudPropuestaVersionInscripcion,
) (domain.PropuestaVersionInscripcion, bool, error) {
	var vacia domain.PropuestaVersionInscripcion
	if s == nil || interfazGobiernoInscripcionNula(s.autoridad) || interfazGobiernoInscripcionNula(s.reloj) {
		return vacia, false, ErrAdministracionPerfilesNoConfigurada
	}
	if ctx == nil || solicitud.Validar() != nil || !s.comprobarAdmin(solicitud.InstantaneaAutorizacion) ||
		solicitud.Evidencia.ValidarEn(solicitud.Actor, s.reloj.Ahora()) != nil {
		return vacia, false, domain.ErrPlanVersionInscripcionInvalido
	}
	if err := ctx.Err(); err != nil {
		return vacia, false, err
	}
	catalogo, err := s.autoridad.ResolverCatalogoVersionInscripcion(ctx, solicitud.Plan)
	if err != nil || solicitud.Plan.ValidarFuenteHistorica(catalogo, s.reloj.Ahora()) != nil {
		return vacia, false, domain.ErrPlanVersionInscripcionInvalido
	}
	orden := domain.OrdenPropuestaVersionInscripcion{Solicitud: solicitud,
		Material: domain.MaterialPropuestaVersionInscripcion{OperacionRef: solicitud.OperacionRef,
			ProponentePersonaRef: solicitud.Actor.PersonaRef, PerfilActivoRef: solicitud.Actor.PerfilActivoRef,
			AsignacionPerfilRef: solicitud.InstantaneaAutorizacion.AsignacionPerfil.Referencia(), Plan: solicitud.Plan}}
	if orden.Validar() != nil {
		return vacia, false, domain.ErrPlanVersionInscripcionInvalido
	}
	propuesta, replay, err := s.autoridad.ProponerVersionInscripcion(ctx, orden)
	if err != nil {
		return vacia, false, err
	}
	if propuesta.ValidarPara(orden) != nil || (!replay && !propuesta.CaducaEn.After(s.reloj.Ahora())) {
		return vacia, false, domain.ErrPlanVersionInscripcionInvalido
	}
	return propuesta, replay, nil
}

func (s *ServicioGobiernoInscripcion) CerrarVersionInscripcion(
	ctx context.Context, solicitud domain.SolicitudCierreVersionInscripcion,
) (domain.CierreVersionInscripcion, bool, error) {
	var vacio domain.CierreVersionInscripcion
	if s == nil || interfazGobiernoInscripcionNula(s.autoridad) || interfazGobiernoInscripcionNula(s.reloj) {
		return vacio, false, ErrAdministracionPerfilesNoConfigurada
	}
	if ctx == nil || solicitud.Validar() != nil || !s.comprobarAdmin(solicitud.InstantaneaAutorizacion) ||
		solicitud.Evidencia.ValidarEn(solicitud.Aprobador, s.reloj.Ahora()) != nil {
		return vacio, false, domain.ErrPlanVersionInscripcionInvalido
	}
	if err := ctx.Err(); err != nil {
		return vacio, false, err
	}
	cierre, replay, err := s.autoridad.CerrarVersionInscripcion(ctx, solicitud)
	if err != nil {
		return vacio, false, err
	}
	if cierre.ValidarPara(solicitud) != nil || cierre.ConfirmadoEn.After(s.reloj.Ahora()) {
		return vacio, false, domain.ErrPlanVersionInscripcionInvalido
	}
	return cierre, replay, nil
}
