package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	vecapp "vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/ports"
)

// RegistroOperacionContactoPostgreSQL sólo confirma un contacto si la
// intención preparada y su compromiso HMAC siguen vigentes. Contacto3 enlaza
// operación, versión, consumo V3, recibo T13 y outbox en una transacción.
type RegistroOperacionContactoPostgreSQL struct{ pool iniciadorTransacciones }

func NuevoRegistroOperacionContactoPostgreSQL(pool *pgxpool.Pool) (*RegistroOperacionContactoPostgreSQL, error) {
	return nuevoRegistroOperacionContactoPostgreSQL(pool)
}

func nuevoRegistroOperacionContactoPostgreSQL(pool iniciadorTransacciones) (*RegistroOperacionContactoPostgreSQL, error) {
	if valorNuloPostgreSQL(pool) {
		return nil, vecapp.ErrContactoUsuarioNoDisponible
	}
	return &RegistroOperacionContactoPostgreSQL{pool: pool}, nil
}

func (r *RegistroOperacionContactoPostgreSQL) GuardarContactoUsuario(ctx context.Context, orden ports.OrdenRegistroContactoUsuario) (ports.ReciboContactoUsuario, error) {
	if r == nil {
		return ports.ReciboContactoUsuario{}, vecapp.ErrContactoUsuarioNoDisponible
	}
	return guardarContactoPostgreSQL(ctx, r.pool, orden, true)
}

var _ ports.RegistroContactoUsuario = (*RegistroOperacionContactoPostgreSQL)(nil)
