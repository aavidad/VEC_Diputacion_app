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

// Huella SHA-256 de pg_get_constraintdef de la restricción tal como la deja
// AD300. Se compara entera: buscar solo el literal del motivo aceptaría una
// definición distinta que lo nombre (por ejemplo, con «<>»).
const huellaRestriccionAuditoriaModalidades = "9364e6e24509ac95340610b822310a82d1d556f5ad39f372907bc269b6de260e"

const firmaGuardarPreferenciasModalidades = "vec_bolsa_llamamientos.guardar_preferencias_modalidades_v1(text,text[],jsonb,bigint,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)"

// Las modalidades solo se montan si B138 (escritura) y AD300 (motivo de
// auditoría propio de la proyección de cinco campos) están instaladas. Sin
// AD300 la fachada de auditoría rechazaría cada lectura; se comprueba aquí
// para que el servidor no ofrezca una ruta que siempre fallaría.
const consultaSQLPreferenciasModalidades = `SELECT EXISTS(SELECT 1 FROM pg_catalog.pg_proc p
 WHERE p.oid=pg_catalog.to_regprocedure($1) AND p.prokind='f'
 AND session_user=current_user
 AND pg_catalog.has_function_privilege(session_user,p.oid,'EXECUTE')
 AND NOT EXISTS(SELECT 1 FROM pg_catalog.aclexplode(
  coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
  WHERE a.grantee=0 AND a.privilege_type='EXECUTE'))
 AND EXISTS(SELECT 1 FROM pg_catalog.pg_constraint c
  JOIN pg_catalog.pg_class t ON t.oid=c.conrelid
  JOIN pg_catalog.pg_namespace n ON n.oid=t.relnamespace
  WHERE n.nspname='vec_autorizacion_atestada_v3' AND t.relname='auditoria_consumo_v3'
   AND c.conname='auditoria_lectura_preferencias_bolsas_formato_v2'
   AND c.contype='c' AND c.convalidated
   AND pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
    pg_catalog.pg_get_constraintdef(c.oid,false),'UTF8')),'hex')=$2)`

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

// comprobarSQLPreferenciasModalidades exige la escritura B138 ejecutable por
// el LOGIN y sin PUBLIC, y la restricción de auditoría ampliada por AD300.
func comprobarSQLPreferenciasModalidades(ctx context.Context, pool *pgxpool.Pool) error {
	if ctx == nil || pool == nil {
		return errMiBolsaNoDisponible
	}
	var valida bool
	if err := pool.QueryRow(ctx, consultaSQLPreferenciasModalidades,
		firmaGuardarPreferenciasModalidades, huellaRestriccionAuditoriaModalidades).Scan(&valida); err != nil {
		return err
	}
	if !valida {
		return errMiBolsaNoDisponible
	}
	return nil
}
