package postgres

import (
	"context"

	"vec-diputacion-granada/internal/vec/ports"
)

var _ ports.CatalogoRolesAdministrables = (*AutoridadVersionarRolBolsaExterna)(nil)

// ResolverRolAdministrable usa el mismo catálogo y las mismas comprobaciones
// nominales de B1. La base comparte el pool Externa y no ejecuta el preflight B1.
func (a *AutoridadVersionarRolBolsaExterna) ResolverRolAdministrable(ctx context.Context,
	ref string) (ports.RolAdministrable, error) {
	if err := a.disponible(ctx); err != nil {
		return ports.RolAdministrable{}, err
	}
	return a.base.ResolverRolAdministrable(ctx, ref)
}
