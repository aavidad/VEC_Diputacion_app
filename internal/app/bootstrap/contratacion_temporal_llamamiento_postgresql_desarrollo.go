package bootstrap

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/config"
)

const rolEjecucionBolsaLlamamientosDesarrollo = "vec_bolsa_llamamientos_ejecutor"

func abrirBolsaLlamamientosPostgreSQLDesarrollo(ctx context.Context, configuracion config.ConfiguracionPostgreSQLContratacionTemporal) (*pgxpool.Pool, error) {
	dsn, err := configuracion.DSNBolsaLlamamientosSeparado()
	if err != nil {
		return nil, err
	}
	pool, _, err := abrirPoolPostgreSQLContratacionTemporalDesarrollo(ctx, dsn,
		"vec-bolsa-llamamientos-desarrollo", rolEjecucionBolsaLlamamientosDesarrollo)
	if err != nil {
		return nil, err
	}
	var separada bool
	err = pool.QueryRow(ctx, `SELECT NOT EXISTS (
	 SELECT 1 FROM pg_catalog.pg_roles r
	 WHERE (r.rolname LIKE 'vec_contratacion_temporal_%'
	   OR r.rolname IN ('vec_bolsa_llamamientos_propietario', 'vec_bolsa_llamamientos_migrador',
	    'vec_autorizacion_atestada_v3_propietario', 'vec_autorizacion_atestada_v3_migrador', 'vec_autorizacion_registro'))
	 AND pg_catalog.pg_has_role(session_user,r.oid,'MEMBER'))`).Scan(&separada)
	if err != nil || !separada {
		pool.Close()
		return nil, errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	return pool, nil
}
