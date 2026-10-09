package ports

import (
	"context"

	"vec-diputacion-granada/internal/vec/domain"
)

// AutoridadVersionInscripcion pertenece al gobierno central de autorización.
// Resolver consulta la fuente publicada completa; proponer y cerrar usan dos
// ADMIN nominales distintos y consumen V3, CAS, auditoría y recibo en SQL.
// Una instalación de migración no equivale a publicar descriptores o roles.
type AutoridadVersionInscripcion interface {
	ResolverCatalogoVersionInscripcion(context.Context, domain.PlanVersionInscripcion) (domain.CatalogoAccionesAdministracionV1, error)
	ProponerVersionInscripcion(context.Context, domain.OrdenPropuestaVersionInscripcion) (domain.PropuestaVersionInscripcion, bool, error)
	CerrarVersionInscripcion(context.Context, domain.SolicitudCierreVersionInscripcion) (domain.CierreVersionInscripcion, bool, error)
}
