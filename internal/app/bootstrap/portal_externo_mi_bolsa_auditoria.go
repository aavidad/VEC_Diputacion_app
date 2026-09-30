package bootstrap

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	bolsapg "vec-diputacion-granada/internal/modules/bolsa/adapters/postgres"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// abrirAuditoriaMiBolsaPortalExterno exige un DSN exclusivo del registrador.
// Su LOGIN solo hereda vec_bolsa_llamamientos_registrador_portal_externo;
// el constructor comprueba la ACL efectiva antes de montar las rutas.
func abrirAuditoriaMiBolsaPortalExterno(ctx context.Context, dsn string) (*pgxpool.Pool, vecports.RegistradorAuditoriaFronteraRutaExacta, error) {
	pool, _, err := abrirPoolMiBolsaPortalExterno(ctx, dsn, "vec_bolsa_llamamientos_registrador_portal_externo")
	if err != nil {
		return nil, nil, errMiBolsaNoDisponible
	}
	registrador, err := bolsapg.NuevoRegistradorFronteraBolsaExternaPostgreSQL(ctx, pool)
	if err != nil {
		pool.Close()
		return nil, nil, errMiBolsaNoDisponible
	}
	return pool, registrador, nil
}
