package administracion

import (
	"context"
	api "vec-diputacion-granada/internal/vec/adapters/httpapi/administracionperfiles"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// Ausencia de fuente: no devuelve datos, perfiles, capacidades ni recibos.
// Permite conservar la frontera y mostrar el fallo HTTP nominal del servicio.
type lecturasNoDisponibles struct{}

func (lecturasNoDisponibles) Capacidades(context.Context, domain.ContextoActor, domain.EvidenciaSesionAdministracionPerfiles) (api.Capacidades, error) {
	return api.Capacidades{}, ports.ErrAutoridadAdministracionPerfilesNoDisponible
}
func (lecturasNoDisponibles) BuscarPersonas(context.Context, domain.ContextoActor, domain.EvidenciaSesionAdministracionPerfiles, api.ConsultaPersonas) (api.PaginaPersonas, error) {
	return api.PaginaPersonas{}, ports.ErrAutoridadAdministracionPerfilesNoDisponible
}
func (lecturasNoDisponibles) ConsultarPersona(context.Context, domain.ContextoActor, domain.EvidenciaSesionAdministracionPerfiles, string) (api.FichaPersona, error) {
	return api.FichaPersona{}, ports.ErrAutoridadAdministracionPerfilesNoDisponible
}
func (lecturasNoDisponibles) ListarRoles(context.Context, domain.ContextoActor, domain.EvidenciaSesionAdministracionPerfiles) (api.Roles, error) {
	return api.Roles{}, ports.ErrAutoridadAdministracionPerfilesNoDisponible
}
func (lecturasNoDisponibles) ListarPropuestas(context.Context, domain.ContextoActor, domain.EvidenciaSesionAdministracionPerfiles) (api.PaginaPropuestas, error) {
	return api.PaginaPropuestas{}, ports.ErrAutoridadAdministracionPerfilesNoDisponible
}
func (lecturasNoDisponibles) ConsultarPropuesta(context.Context, domain.ContextoActor, domain.EvidenciaSesionAdministracionPerfiles, string) (api.Propuesta, error) {
	return api.Propuesta{}, ports.ErrAutoridadAdministracionPerfilesNoDisponible
}
func (lecturasNoDisponibles) ConsultarRecibo(context.Context, domain.ContextoActor, domain.EvidenciaSesionAdministracionPerfiles, string) (domain.ReciboAdministracionPerfiles, error) {
	return domain.ReciboAdministracionPerfiles{}, ports.ErrAutoridadAdministracionPerfilesNoDisponible
}

var _ api.FuenteLecturas = lecturasNoDisponibles{}
