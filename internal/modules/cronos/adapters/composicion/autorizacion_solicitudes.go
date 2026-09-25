package composicion

import (
	"context"

	"vec-diputacion-granada/internal/modules/cronos/application"
	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// Emisión V3 del segundo corte: una decisión nueva por acción, sólo para la
// persona, el perfil y el empleado de la petición en curso.

func (p proveedorPeticion) ProveerMaterialConsultaMovimientosPropios(ctx context.Context, m domain.MaterialConsultaMovimientosPropios) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	if !mismaPersona(p.identidad, m.ActorRef, m.PerfilRef, m.EmpleadoRef) {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrDependenciaNoDisponible
	}
	recurso, err := application.RecursoConsultaMovimientosPropios(m)
	if err != nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrDependenciaNoDisponible
	}
	return p.autorizador.emitir(ctx, p.identidad, contratoV3{application.AccionConsultarMovimientosPropios, application.FinalidadConsultarMovimientosPropios, application.AudienciaConsultaMovimientosPropios, p.autorizador.motivos.Movimientos}, recurso)
}

// ProveerMaterialCorreccion sólo emite para la solicitud de la propia
// persona; las decisiones de jefatura y RRHH son otro corte.
func (p proveedorPeticion) ProveerMaterialCorreccion(ctx context.Context, m domain.MaterialAutorizacionCorreccion) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	if m.Paso != domain.PasoSolicitudCorreccion || !mismaPersona(p.identidad, m.ActorRef, m.PerfilRef, m.EmpleadoRef) {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrDependenciaNoDisponible
	}
	recurso, err := application.RecursoSolicitudCorreccionPropia(m)
	if err != nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrDependenciaNoDisponible
	}
	return p.autorizador.emitir(ctx, p.identidad, contratoV3{application.AccionSolicitarCorreccion, application.FinalidadSolicitarCorreccion, application.AudienciaSolicitudCorreccionPropia, p.autorizador.motivos.Correccion}, recurso)
}

func (p proveedorPeticion) ProveerMaterialConsultaPermisosPropios(ctx context.Context, m domain.MaterialConsultaPermisosPropios) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	if !mismaPersona(p.identidad, m.ActorRef, m.PerfilRef, m.EmpleadoRef) {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrDependenciaNoDisponible
	}
	recurso, err := application.RecursoConsultaPermisosPropios(m)
	if err != nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrDependenciaNoDisponible
	}
	return p.autorizador.emitir(ctx, p.identidad, contratoV3{application.AccionConsultarPermisosPropios, application.FinalidadConsultarPermisosPropios, application.AudienciaConsultaPermisosPropios, p.autorizador.motivos.Permisos}, recurso)
}

func (p proveedorPeticion) ProveerMaterialSolicitudPermisoPropio(ctx context.Context, m domain.MaterialSolicitudPermisoPropio) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	if !mismaPersona(p.identidad, m.ActorRef, m.PerfilRef, m.EmpleadoRef) {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrDependenciaNoDisponible
	}
	recurso, err := application.RecursoSolicitudPermisoPropio(m)
	if err != nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrDependenciaNoDisponible
	}
	return p.autorizador.emitir(ctx, p.identidad, contratoV3{application.AccionSolicitarPermisoPropio, application.FinalidadSolicitarPermisoPropio, application.AudienciaSolicitudPermisoPropio, p.autorizador.motivos.Permiso}, recurso)
}

var (
	_ ports.ProveedorMaterialConsultaMovimientosPropios = proveedorPeticion{}
	_ ports.ProveedorMaterialCorreccion                 = proveedorPeticion{}
	_ ports.ProveedorMaterialPermisosPropios            = proveedorPeticion{}
)
