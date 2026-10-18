package bootstrap

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/config"
	reglasbolsa "vec-diputacion-granada/internal/modules/bolsa/adapters/reglasvec"
	bolsapuertos "vec-diputacion-granada/internal/modules/bolsa/ports"
	"vec-diputacion-granada/internal/vec/reglas"
)

const firmaGuardarPreferenciasModalidades = "vec_bolsa_llamamientos.guardar_preferencias_modalidades_v1(text,text[],jsonb,bigint,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)"

// Usa la fuente CT ya declarada para el catálogo común. Su ausencia solo
// desactiva esta proyección opcional de preferencias.
func catalogoPreferenciasModalidadesDesarrollo(ctx context.Context, cfg config.Config) (bolsapuertos.CatalogoPreferenciasModalidades, error) {
	rutas, activas, err := cfg.ReglasEjemploDesarrollo()
	if err != nil || !activas || rutas.CTSourcePath == "" {
		return nil, errMiBolsaNoDisponible
	}
	resolutor, err := nuevoResolutorReglasEjemplo(rutas.CTSourcePath,
		reglas.CatalogoContratacionTemporal, reglas.ModuloContratacionTemporal, nil, relojCalendariosDesarrollo{})
	if err != nil || resolutor == nil {
		return nil, errMiBolsaNoDisponible
	}
	catalogo := reglasbolsa.NuevoCatalogoPreferenciasModalidades(resolutor)
	if _, err := catalogo.ConsultarModalidadesDisponibles(ctx); err != nil {
		return nil, errors.Join(errMiBolsaNoDisponible, err)
	}
	return catalogo, nil
}

func comprobarFuncionPreferenciasModalidades(ctx context.Context, pool *pgxpool.Pool) bool {
	if ctx == nil || pool == nil {
		return false
	}
	var valida bool
	err := pool.QueryRow(ctx, `WITH funcion AS (SELECT pg_catalog.to_regprocedure($1) AS oid)
SELECT oid IS NOT NULL AND pg_catalog.has_function_privilege(session_user,oid,'EXECUTE')
 AND NOT pg_catalog.has_function_privilege('public',oid,'EXECUTE') FROM funcion`,
		firmaGuardarPreferenciasModalidades).Scan(&valida)
	return err == nil && valida
}
