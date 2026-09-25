package composicion

import (
	"context"
	"net/http"

	"vec-diputacion-granada/internal/modules/cronos/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/cronos/application"
	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// Emisión V3 de las notificaciones a RRHH (AD3-58): una decisión nueva por
// acción, sólo para la persona, el perfil y el empleado propio de la petición
// en curso. RRHH se autoriza sobre su propia persona y el paso de
// administración; nunca sobre la persona que notificó.

func (p proveedorPeticion) notificacionesDisponibles() bool {
	return p.autorizador != nil && p.autorizador.NotificacionesConfiguradas()
}

func (p proveedorPeticion) ProveerMaterialRegistroNotificacion(ctx context.Context, m domain.MaterialRegistroNotificacion) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	if !p.notificacionesDisponibles() || !mismaPersona(p.identidad, m.ActorRef, m.PerfilRef, m.EmpleadoRef) {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrDependenciaNoDisponible
	}
	recurso, err := application.RecursoRegistroNotificacion(m)
	if err != nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrDependenciaNoDisponible
	}
	return p.autorizador.emitir(ctx, p.identidad, contratoV3{application.AccionRegistrarNotificacion, application.FinalidadRegistrarNotificacion, application.AudienciaRegistroNotificacion, p.autorizador.motivos.Notificacion}, recurso)
}

func (p proveedorPeticion) ProveerMaterialConsultaNotificacionesPropias(ctx context.Context, m domain.MaterialConsultaNotificaciones) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	if !p.notificacionesDisponibles() || !mismaPersona(p.identidad, m.ActorRef, m.PerfilRef, m.EmpleadoRef) {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrDependenciaNoDisponible
	}
	recurso, err := application.RecursoConsultaNotificacionesPropias(m)
	if err != nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrDependenciaNoDisponible
	}
	return p.autorizador.emitir(ctx, p.identidad, contratoV3{application.AccionConsultarNotificacionesPropias, application.FinalidadConsultarNotificacionesPropias, application.AudienciaConsultaNotificacionesPropias, p.autorizador.motivos.Notificaciones}, recurso)
}

func (p proveedorPeticion) ProveerMaterialBandejaNotificaciones(ctx context.Context, m domain.MaterialConsultaNotificaciones) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	if !p.notificacionesDisponibles() || !mismaPersona(p.identidad, m.ActorRef, m.PerfilRef, m.EmpleadoRef) {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrDependenciaNoDisponible
	}
	recurso, err := application.RecursoBandejaNotificaciones(m)
	if err != nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrDependenciaNoDisponible
	}
	return p.autorizador.emitir(ctx, p.identidad, contratoV3{application.AccionConsultarBandejaNotif, application.FinalidadConsultarBandejaNotif, application.AudienciaBandejaNotificaciones, p.autorizador.motivos.BandejaNotificaciones}, recurso)
}

func (p proveedorPeticion) ProveerMaterialAtencionNotificacion(ctx context.Context, m domain.MaterialAtencionNotificacion) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	if !p.notificacionesDisponibles() || !mismaPersona(p.identidad, m.ActorRef, m.PerfilRef, m.EmpleadoRef) {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrDependenciaNoDisponible
	}
	recurso, err := application.RecursoAtencionNotificacion(m)
	if err != nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrDependenciaNoDisponible
	}
	return p.autorizador.emitir(ctx, p.identidad, contratoV3{application.AccionAtenderNotificacion, application.FinalidadAtenderNotificacion, application.AudienciaAtencionNotificacion, p.autorizador.motivos.AtencionNotificacion}, recurso)
}

func (r *ResolutorPeticionCronos) ResolverNotificacionesPropias(req *http.Request) (ports.OrdenNotificacionesPropias, error) {
	p, err := r.peticion(req)
	if err != nil {
		return ports.OrdenNotificacionesPropias{}, err
	}
	if !p.notificacionesDisponibles() {
		return ports.OrdenNotificacionesPropias{}, ports.ErrDependenciaNoDisponible
	}
	return ports.NuevaOrdenNotificacionesPropias(p.identidad.Contexto.Contexto, p)
}

func (r *ResolutorPeticionCronos) ResolverBandejaNotificaciones(req *http.Request) (ports.OrdenBandejaNotificaciones, error) {
	p, err := r.peticion(req)
	if err != nil {
		return ports.OrdenBandejaNotificaciones{}, err
	}
	if !p.notificacionesDisponibles() {
		return ports.OrdenBandejaNotificaciones{}, ports.ErrDependenciaNoDisponible
	}
	return ports.NuevaOrdenBandejaNotificaciones(p.identidad.Contexto.Contexto, p)
}

var (
	_ ports.ProveedorMaterialNotificacionesPropias = proveedorPeticion{}
	_ ports.ProveedorMaterialBandejaNotificaciones = proveedorPeticion{}
	_ httpinterno.ResolverNotificacionesPropias    = (*ResolutorPeticionCronos)(nil)
	_ httpinterno.ResolverBandejaNotificaciones    = (*ResolutorPeticionCronos)(nil)
)
