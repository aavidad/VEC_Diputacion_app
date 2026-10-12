package postgres

import (
	"context"

	"vec-diputacion-granada/internal/vec/ports"
)

var _ ports.CatalogoRolesAdministrables = (*AutoridadVersionarRolBolsaB103)(nil)

// ResolverRolAdministrable usa el mismo catálogo y las mismas comprobaciones
// nominales de B1. La base comparte el pool B103 y no ejecuta el preflight B1.
func (a *AutoridadVersionarRolBolsaB103) ResolverRolAdministrable(ctx context.Context,
	ref string) (ports.RolAdministrable, error) {
	if err := a.disponible(ctx); err != nil {
		return ports.RolAdministrable{}, err
	}
	return a.base.ResolverRolAdministrable(ctx, ref)
}
