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

// Emisión V3 de la resolución de permisos y de los avisos (AD3-57): una
// decisión nueva por acción, sólo para la persona, el perfil y el empleado
// propio de la petición en curso. Quien resuelve se autoriza sobre su propia
// persona y el paso; nunca sobre la persona de la solicitud.

func (p proveedorPeticion) resolucionDisponible() bool {
	return p.autorizador != nil && p.autorizador.ResolucionConfigurada()
}

func (p proveedorPeticion) ProveerMaterialBandejaPermisos(ctx context.Context, m domain.MaterialBandejaPermisos) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	if !p.resolucionDisponible() || !mismaPersona(p.identidad, m.ActorRef, m.PerfilRef, m.EmpleadoRef) {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrDependenciaNoDisponible
	}
	recurso, err := application.RecursoBandejaPermisos(m)
	if err != nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrDependenciaNoDisponible
	}
	return p.autorizador.emitir(ctx, p.identidad, contratoV3{application.AccionConsultarBandeja, application.FinalidadConsultarBandeja, application.AudienciaBandejaPermisos, p.autorizador.motivos.Bandeja}, recurso)
}

func (p proveedorPeticion) ProveerMaterialResolucionPermiso(ctx context.Context, m domain.MaterialResolucionPermiso) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	if !p.resolucionDisponible() || !mismaPersona(p.identidad, m.ActorRef, m.PerfilRef, m.EmpleadoRef) {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrDependenciaNoDisponible
	}
	recurso, err := application.RecursoResolucionPermiso(m)
	if err != nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrDependenciaNoDisponible
	}
	return p.autorizador.emitir(ctx, p.identidad, contratoV3{application.AccionResolverPermiso, application.FinalidadResolverPermiso, application.AudienciaResolucionPermiso, p.autorizador.motivos.Resolucion}, recurso)
}

func (p proveedorPeticion) ProveerMaterialConsultaAvisosPropios(ctx context.Context, m domain.MaterialConsultaAvisosPropios) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	if !p.resolucionDisponible() || !mismaPersona(p.identidad, m.ActorRef, m.PerfilRef, m.EmpleadoRef) {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrDependenciaNoDisponible
	}
	recurso, err := application.RecursoConsultaAvisosPropios(m)
	if err != nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrDependenciaNoDisponible
	}
	return p.autorizador.emitir(ctx, p.identidad, contratoV3{application.AccionConsultarAvisosPropios, application.FinalidadConsultarAvisosPropios, application.AudienciaConsultaAvisosPropios, p.autorizador.motivos.Avisos}, recurso)
}

func (p proveedorPeticion) ProveerMaterialArchivoAvisoPropio(ctx context.Context, m domain.MaterialArchivoAvisoPropio) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	if !p.resolucionDisponible() || !mismaPersona(p.identidad, m.ActorRef, m.PerfilRef, m.EmpleadoRef) {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrDependenciaNoDisponible
	}
	recurso, err := application.RecursoArchivoAvisoPropio(m)
	if err != nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrDependenciaNoDisponible
	}
	return p.autorizador.emitir(ctx, p.identidad, contratoV3{application.AccionArchivarAvisoPropio, application.FinalidadArchivarAvisoPropio, application.AudienciaArchivoAvisoPropio, p.autorizador.motivos.ArchivoAviso}, recurso)
}

func (r *ResolutorPeticionCronos) ResolverResolucionPermisos(req *http.Request) (ports.OrdenResolucionPermisos, error) {
	p, err := r.peticion(req)
	if err != nil {
		return ports.OrdenResolucionPermisos{}, err
	}
	if !p.resolucionDisponible() {
		return ports.OrdenResolucionPermisos{}, ports.ErrDependenciaNoDisponible
	}
	return ports.NuevaOrdenResolucionPermisos(p.identidad.Contexto.Contexto, p)
}

func (r *ResolutorPeticionCronos) ResolverAvisosPropios(req *http.Request) (ports.OrdenAvisosPropios, error) {
	p, err := r.peticion(req)
	if err != nil {
		return ports.OrdenAvisosPropios{}, err
	}
	if !p.resolucionDisponible() {
		return ports.OrdenAvisosPropios{}, ports.ErrDependenciaNoDisponible
	}
	return ports.NuevaOrdenAvisosPropios(p.identidad.Contexto.Contexto, p)
}

var (
	_ ports.ProveedorMaterialResolucionPermisos = proveedorPeticion{}
	_ ports.ProveedorMaterialAvisosPropios      = proveedorPeticion{}
	_ httpinterno.ResolverResolucionPermisos    = (*ResolutorPeticionCronos)(nil)
	_ httpinterno.ResolverAvisosPropios         = (*ResolutorPeticionCronos)(nil)
)
