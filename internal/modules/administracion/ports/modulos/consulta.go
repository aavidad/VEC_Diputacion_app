package modulos

import (
	"context"
	domain "vec-diputacion-granada/internal/modules/administracion/domain/modulos"
	vec "vec-diputacion-granada/internal/vec/domain"
)

// RegistroCompuesto lo provee el registro existente de VEC, no el cliente ADMIN.
type RegistroCompuesto interface {
	ListModules(context.Context) ([]vec.ModuleManifest, error)
}

// ControlOperativo debe consultar la publicación vigente en cada operación.
// No sustituye al PDP ni autoriza el acceso a los datos del módulo.
type ControlOperativo interface {
	ExigirHabilitado(context.Context, string) error
}

type Consulta interface {
	Consultar(context.Context) (domain.Estado, error)
	PrepararCambio(context.Context, domain.Cambio) (domain.Preparacion, error)
}
