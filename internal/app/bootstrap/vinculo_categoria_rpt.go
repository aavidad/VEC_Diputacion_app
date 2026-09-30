package bootstrap

import (
	"github.com/jackc/pgx/v5/pgxpool"

	ctpostgres "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	ctapp "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// ComponerVinculoCategoriaRPT conecta los dos consumidores nominales al mismo
// principal. La autoridad se inyecta desde la composicion de identidad V3
// con sus perfiles fijos; esta funcion no publica concesiones ni crea sesiones.
func ComponerVinculoCategoriaRPT(
	pool *pgxpool.Pool,
	autoridad ctports.AutoridadVinculoCategoriaRPT,
	reloj ctports.Reloj,
) (*ctapp.ServicioVinculoCategoriaRPT, error) {
	if pool == nil || autoridad == nil || reloj == nil {
		return nil, ctports.ErrVinculoCategoriaRPTNoDisponible
	}
	ct, e := ctpostgres.NuevaFuenteVinculoCategoriaRPTPostgreSQL(pool)
	if e != nil {
		return nil, e
	}
	rpt, e := ctpostgres.NuevaFuentePublicacionCategoriaRPTPostgreSQL(pool)
	if e != nil {
		return nil, e
	}
	return ctapp.NuevoServicioVinculoCategoriaRPT(autoridad, ct, rpt, reloj)
}
