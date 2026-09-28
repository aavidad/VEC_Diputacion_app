package bootstrap

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	plantillasapp "vec-diputacion-granada/internal/modules/contrataciontemporal/application/plantillascatalogo"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

// preflightCatalogoPlantillasCT comprueba que CT131 y CT133 estén instaladas
// y ejecutables por el LOGIN CT nominal antes de publicar las rutas. No
// provisiona la preimagen ni toca datos: eso pertenece al migrador separado.
func preflightCatalogoPlantillasCT(ctx context.Context, pool *pgxpool.Pool) error {
	if pool == nil {
		return plantillasapp.ErrNoDisponible
	}
	return comprobarPreflightCatalogoPlantillasCT(ctx, pool)
}

// CT131 puede montarse con CT133 pendiente. Ambos comparten el mismo LOGIN,
// pero solo se exige la fachada de gobierno que este corte consume.
func preflightCatalogoPlantillasGobiernoCT(ctx context.Context, pool *pgxpool.Pool) error {
	if pool == nil {
		return plantillasapp.ErrNoDisponible
	}
	return comprobarPreflightCatalogoPlantillasCT(ctx, pool, false)
}

func comprobarPreflightCatalogoPlantillasCT(ctx context.Context, consulta interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, documental ...bool) error {
	if ctx == nil || ctx.Err() != nil || consulta == nil {
		return plantillasapp.ErrNoDisponible
	}
	funciones := []string{
		"vec_contratacion_temporal.operar_catalogo_plantillas_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)",
		"vec_contratacion_temporal.comprobar_catalogo_plantillas_base_v1(text,bigint,text)",
	}
	if len(documental) == 0 || documental[0] {
		funciones = append(funciones,
			"vec_contratacion_temporal.obtener_catalogo_plantillas_publicado_documental_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)")
	}
	const sql = `SELECT EXISTS (
  SELECT 1 FROM pg_catalog.pg_roles l
  JOIN pg_catalog.pg_roles g ON g.rolname='vec_contratacion_temporal_ejecutor'
  JOIN pg_catalog.pg_auth_members m ON m.member=l.oid AND m.roleid=g.oid
  WHERE l.rolname=session_user AND session_user=current_user
    AND l.rolcanlogin AND l.rolinherit AND NOT l.rolsuper
    AND NOT l.rolcreatedb AND NOT l.rolcreaterole AND NOT l.rolreplication AND NOT l.rolbypassrls
    AND NOT g.rolcanlogin AND NOT g.rolbypassrls
    AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option
    AND (SELECT count(*) FROM pg_catalog.pg_auth_members x WHERE x.member=l.oid)=1
    AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members x WHERE x.roleid=l.oid)
    AND pg_catalog.pg_has_role(session_user,g.oid,'MEMBER')
    AND pg_catalog.pg_has_role(session_user,g.oid,'USAGE')
    AND pg_catalog.has_schema_privilege(session_user,'vec_contratacion_temporal','USAGE')
    AND pg_catalog.has_schema_privilege(g.oid,'vec_contratacion_temporal','USAGE')
    AND NOT pg_catalog.has_schema_privilege(session_user,'vec_contratacion_temporal','CREATE')
    AND NOT pg_catalog.has_schema_privilege(g.oid,'vec_contratacion_temporal','CREATE')
    AND NOT pg_catalog.pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
    AND NOT pg_catalog.pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
    AND NOT pg_catalog.pg_has_role(session_user,'vec_contratacion_temporal_gobernador','MEMBER')
    AND (SELECT coalesce(bool_and(pg_catalog.to_regprocedure(f.nombre) IS NOT NULL
           AND pg_catalog.has_function_privilege(session_user,pg_catalog.to_regprocedure(f.nombre),'EXECUTE')),false)
         FROM pg_catalog.unnest($1::text[]) AS f(nombre))
    AND NOT pg_catalog.has_function_privilege(session_user,
        'vec_contratacion_temporal.provisionar_catalogo_plantillas_base_v1(jsonb,text,text,text)','EXECUTE')
    AND NOT EXISTS (
      SELECT 1 FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
      WHERE n.nspname='vec_contratacion_temporal' AND c.relkind IN ('r','p','v','m')
        AND (pg_catalog.has_table_privilege(session_user,c.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER,MAINTAIN')
          OR pg_catalog.has_any_column_privilege(session_user,c.oid,'SELECT,INSERT,UPDATE,REFERENCES')))
    AND NOT coalesce(pg_catalog.has_schema_privilege(session_user,pg_catalog.to_regnamespace('vec_bolsa_llamamientos'),'USAGE'),false)
    AND NOT coalesce(pg_catalog.has_schema_privilege(session_user,pg_catalog.to_regnamespace('vec_autorizacion_atestada_v3'),'USAGE'),false)
    AND NOT coalesce(pg_catalog.has_function_privilege(session_user,
      pg_catalog.to_regprocedure('vec_contratacion_temporal.consultar_auditoria_ct_atestada_v1(text,text,text,timestamptz,timestamptz,integer,timestamptz,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'),'EXECUTE'),false)
    AND NOT coalesce(pg_catalog.has_function_privilege(session_user,
      pg_catalog.to_regprocedure('vec_contratacion_temporal.registrar_auditoria_frontera_auditoria_v1(text,text,text,text,text)'),'EXECUTE'),false)
    AND pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_catalogo_plantillas_ct_org_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    AND EXISTS (SELECT 1 FROM pg_catalog.pg_attribute a
      WHERE a.attrelid=pg_catalog.to_regclass('vec_contratacion_temporal.catalogo_plantillas_historia_v1')
        AND a.attname='organizacion_ref' AND NOT a.attisdropped)
    AND coalesce(pg_catalog.strpos(pg_catalog.pg_get_functiondef(
      pg_catalog.to_regprocedure('vec_contratacion_temporal.operar_catalogo_plantillas_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')),
      'registrar_y_consumir_catalogo_plantillas_ct_org_v3_atestada')>0,false)
 )`
	var valido bool
	if err := consulta.QueryRow(ctx, sql, funciones).Scan(&valido); err != nil || !valido {
		return plantillasapp.ErrNoDisponible
	}
	return nil
}

// La presencia de la función no demuestra que la preimagen gobernada esté
// provisionada. Se coteja su primera publicación y auditoría con el LOGIN CT
// antes de exponer la ruta; no se instala ni corrige desde el servidor.
func comprobarPreimagenCatalogoPlantillasCT(ctx context.Context, consulta interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, catalogo vecdomain.CatalogoConfigurable) error {
	if ctx == nil || ctx.Err() != nil || consulta == nil || catalogo.Validar() != nil ||
		catalogo.ID != plantillasapp.CatalogoID || catalogo.ModuloID != plantillasapp.ModuloID ||
		catalogo.Estado != vecdomain.EstadoCatalogoPublicado || catalogo.Version < 1 || catalogo.FuenteRef == "" {
		return plantillasapp.ErrNoDisponible
	}
	huella, err := catalogo.HuellaSHA256()
	if err != nil {
		return plantillasapp.ErrNoDisponible
	}
	var valida bool
	if err := consulta.QueryRow(ctx,
		`SELECT vec_contratacion_temporal.comprobar_catalogo_plantillas_base_v1($1,$2,$3)`,
		huella, int64(catalogo.Version), catalogo.FuenteRef).Scan(&valida); err != nil || !valida {
		return plantillasapp.ErrNoDisponible
	}
	return nil
}
