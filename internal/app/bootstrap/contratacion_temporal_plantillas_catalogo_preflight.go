package bootstrap

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	plantillasapp "vec-diputacion-granada/internal/modules/contrataciontemporal/application/plantillascatalogo"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

// funcionesV3PermitidasLoginCT es la única lista de fachadas de
// vec_autorizacion_atestada_v3 que el LOGIN CT puede ejecutar, directa o
// heredadamente, sin que fallen los preflights de plantillas (documental y
// gobierno). Son las lecturas y usos de categorías RPT que AD3-117 y AD3-126
// conceden a vec_contratacion_temporal_ejecutor; AD3-117 y AD3-177 le dan por
// eso USAGE del esquema. Cada entrada es la firma regprocedure sin esquema
// (nombre y tipos de entrada), cotejada en pg_proc sin resolver nombres, para
// que funcione igual con USAGE o sin él. Cualquier otra función V3
// ejecutable por el LOGIN, incluidos los consumidores de plantillas, hace
// fallar el arranque: añadir una exige cambiar esta lista y revisarlo.
var funcionesV3PermitidasLoginCT = []string{
	"listar_categorias_habilitadas_rpt_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)",
	"leer_publicacion_categoria_rpt_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)",
	"consultar_uso_categoria_rpt_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)",
	"reservar_uso_categoria_rpt_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)",
	"confirmar_uso_categoria_rpt_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)",
	"cancelar_uso_categoria_rpt_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)",
}

// preflightCatalogoPlantillasCT comprueba CT131, CT135, CT137 y AD3-100
// antes de publicar las rutas documentales al LOGIN CT nominal. No
// provisiona la preimagen ni toca datos: eso pertenece al migrador separado.
// La existencia, firma y ACL se cotejan en pg_catalog; el consumo atestado y
// los tres ámbitos se revalidan al ejecutar las fachadas SQL.
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
			"vec_contratacion_temporal.obtener_catalogo_plantillas_publicado_documental_ambitos_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)")
	}
	exigirDocumental := len(documental) == 0 || documental[0]
	const sql = `SELECT EXISTS (
  SELECT 1 FROM pg_catalog.pg_roles l
  JOIN pg_catalog.pg_roles g ON g.rolname='vec_contratacion_temporal_ejecutor'
  JOIN pg_catalog.pg_auth_members m ON m.member=l.oid AND m.roleid=g.oid
  WHERE l.rolname=session_user AND session_user=current_user
    AND l.rolcanlogin AND l.rolinherit AND NOT l.rolsuper
    AND NOT l.rolcreatedb AND NOT l.rolcreaterole AND NOT l.rolreplication AND NOT l.rolbypassrls
    AND NOT g.rolcanlogin AND NOT g.rolsuper AND NOT g.rolcreatedb
    AND NOT g.rolcreaterole AND NOT g.rolreplication AND NOT g.rolbypassrls
    AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option
    AND (SELECT count(*) FROM pg_catalog.pg_auth_members x WHERE x.member=l.oid)=1
    AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members x WHERE x.roleid=l.oid)
    AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members x WHERE x.member=g.oid)
    AND pg_catalog.pg_has_role(session_user,g.oid,'MEMBER')
    AND pg_catalog.pg_has_role(session_user,g.oid,'USAGE')
    AND pg_catalog.has_schema_privilege(session_user,'vec_contratacion_temporal','USAGE')
    AND pg_catalog.has_schema_privilege(g.oid,'vec_contratacion_temporal','USAGE')
    AND NOT pg_catalog.has_schema_privilege(session_user,'vec_contratacion_temporal','CREATE')
    AND NOT pg_catalog.has_schema_privilege(g.oid,'vec_contratacion_temporal','CREATE')
    AND NOT pg_catalog.pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
    AND NOT pg_catalog.pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
    AND NOT pg_catalog.pg_has_role(session_user,'vec_contratacion_temporal_gobernador','MEMBER')
    AND (SELECT coalesce(bool_and(EXISTS (
           SELECT 1 FROM pg_catalog.pg_proc p
           WHERE p.oid=pg_catalog.to_regprocedure(f.nombre)
             AND p.proowner='vec_contratacion_temporal_propietario'::pg_catalog.regrole
             AND p.prokind='f' AND p.prosecdef
             AND pg_catalog.has_function_privilege(session_user,p.oid,'EXECUTE')
             AND pg_catalog.has_function_privilege(g.oid,p.oid,'EXECUTE')
             AND NOT EXISTS (
               SELECT 1 FROM pg_catalog.aclexplode(coalesce(p.proacl,
                 pg_catalog.acldefault('f',p.proowner))) a
               WHERE a.grantee NOT IN (p.proowner,g.oid) OR a.privilege_type<>'EXECUTE'
             ))),false)
         FROM pg_catalog.unnest($1::text[]) AS f(nombre))
    AND NOT pg_catalog.has_function_privilege(session_user,
        'vec_contratacion_temporal.provisionar_catalogo_plantillas_base_v1(jsonb,text,text,text)','EXECUTE')
    AND NOT EXISTS (
      SELECT 1 FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
      WHERE n.nspname='vec_contratacion_temporal' AND c.relkind IN ('r','p','v','m')
        AND (pg_catalog.has_table_privilege(session_user,c.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER,MAINTAIN')
          OR pg_catalog.has_any_column_privilege(session_user,c.oid,'SELECT,INSERT,UPDATE,REFERENCES')))
    AND NOT coalesce(pg_catalog.has_schema_privilege(session_user,pg_catalog.to_regnamespace('vec_bolsa_llamamientos'),'USAGE'),false)
    AND NOT coalesce(pg_catalog.has_schema_privilege(session_user,pg_catalog.to_regnamespace('vec_autorizacion_atestada_v3'),'CREATE'),false)
    AND NOT EXISTS (
      SELECT 1 FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
      WHERE n.nspname='vec_autorizacion_atestada_v3'
        AND pg_catalog.has_function_privilege(session_user,p.oid,'EXECUTE')
        AND NOT coalesce(p.proname||'('||pg_catalog.replace(pg_catalog.oidvectortypes(p.proargtypes),', ',',')||')'
          = ANY($3::text[]),false))
    AND NOT EXISTS (
      SELECT 1 FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
      WHERE n.nspname='vec_autorizacion_atestada_v3'
        AND CASE WHEN c.relkind='S' THEN pg_catalog.has_sequence_privilege(session_user,c.oid,'USAGE,SELECT,UPDATE')
          WHEN c.relkind IN ('r','p','v','m','f') THEN
            pg_catalog.has_table_privilege(session_user,c.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER,MAINTAIN')
            OR pg_catalog.has_any_column_privilege(session_user,c.oid,'SELECT,INSERT,UPDATE,REFERENCES')
          ELSE false END)
    AND NOT coalesce(pg_catalog.has_function_privilege(session_user,
      pg_catalog.to_regprocedure('vec_contratacion_temporal.consultar_auditoria_ct_atestada_v1(text,text,text,timestamptz,timestamptz,integer,timestamptz,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'),'EXECUTE'),false)
    AND NOT coalesce(pg_catalog.has_function_privilege(session_user,
      pg_catalog.to_regprocedure('vec_contratacion_temporal.registrar_auditoria_frontera_ruta_exacta_v1(text,text,text,text,text)'),'EXECUTE'),false)
    AND NOT coalesce(pg_catalog.has_function_privilege(session_user,
      pg_catalog.to_regprocedure('vec_contratacion_temporal.registrar_auditoria_frontera_auditoria_v1(text,text,text,text,text)'),'EXECUTE'),false)
    AND pg_catalog.has_schema_privilege('vec_contratacion_temporal_propietario','vec_autorizacion_atestada_v3','USAGE')
    AND EXISTS (
      SELECT 1 FROM pg_catalog.pg_proc p
      JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
      WHERE n.nspname='vec_autorizacion_atestada_v3'
        AND p.proname='registrar_y_consumir_catalogo_plantillas_ct_org_v3_atestada'
        AND p.pronargs=11
        AND p.prokind='f' AND p.prosecdef
        AND p.proowner='vec_autorizacion_atestada_v3_propietario'::pg_catalog.regrole
        AND p.proargtypes[0]='pg_catalog.jsonb'::pg_catalog.regtype
        AND p.proargtypes[1]='pg_catalog.bytea'::pg_catalog.regtype
        AND p.proargtypes[2]='pg_catalog.bytea'::pg_catalog.regtype
        AND p.proargtypes[3]='pg_catalog.bytea'::pg_catalog.regtype
        AND p.proargtypes[4]='pg_catalog.bytea'::pg_catalog.regtype
        AND p.proargtypes[5]='pg_catalog.numeric'::pg_catalog.regtype
        AND p.proargtypes[6]='pg_catalog.numeric'::pg_catalog.regtype
        AND p.proargtypes[7]='pg_catalog.bytea'::pg_catalog.regtype
        AND p.proargtypes[8]='pg_catalog.bytea'::pg_catalog.regtype
        AND p.proargtypes[9]='pg_catalog.bytea'::pg_catalog.regtype
        AND p.proargtypes[10]='pg_catalog.bytea'::pg_catalog.regtype
        AND pg_catalog.has_function_privilege('vec_contratacion_temporal_propietario',p.oid,'EXECUTE')
        AND NOT pg_catalog.has_function_privilege(session_user,p.oid,'EXECUTE')
        AND NOT pg_catalog.has_function_privilege(g.oid,p.oid,'EXECUTE')
        AND NOT EXISTS (SELECT 1 FROM pg_catalog.aclexplode(coalesce(p.proacl,
            pg_catalog.acldefault('f',p.proowner))) a
          WHERE a.grantee NOT IN (p.proowner,'vec_contratacion_temporal_propietario'::pg_catalog.regrole)
            OR a.privilege_type<>'EXECUTE'))
    AND EXISTS (SELECT 1 FROM pg_catalog.pg_attribute a
      WHERE a.attrelid=pg_catalog.to_regclass('vec_contratacion_temporal.catalogo_plantillas_historia_v1')
        AND a.attname='organizacion_ref' AND NOT a.attisdropped)
    AND ($2::boolean IS NOT TRUE OR (
      EXISTS (SELECT 1 FROM pg_catalog.pg_proc p
        JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
        WHERE n.nspname='vec_autorizacion_atestada_v3'
          AND p.proname='registrar_y_consumir_plantillas_doc_ct_ambitos_v3_atestada'
          AND p.pronargs=11 AND p.prokind='f' AND p.prosecdef
          AND p.proowner='vec_autorizacion_atestada_v3_propietario'::pg_catalog.regrole
          AND p.proargtypes[0]='pg_catalog.jsonb'::pg_catalog.regtype
          AND p.proargtypes[1]='pg_catalog.bytea'::pg_catalog.regtype
          AND p.proargtypes[2]='pg_catalog.bytea'::pg_catalog.regtype
          AND p.proargtypes[3]='pg_catalog.bytea'::pg_catalog.regtype
          AND p.proargtypes[4]='pg_catalog.bytea'::pg_catalog.regtype
          AND p.proargtypes[5]='pg_catalog.numeric'::pg_catalog.regtype
          AND p.proargtypes[6]='pg_catalog.numeric'::pg_catalog.regtype
          AND p.proargtypes[7]='pg_catalog.bytea'::pg_catalog.regtype
          AND p.proargtypes[8]='pg_catalog.bytea'::pg_catalog.regtype
          AND p.proargtypes[9]='pg_catalog.bytea'::pg_catalog.regtype
          AND p.proargtypes[10]='pg_catalog.bytea'::pg_catalog.regtype
          AND pg_catalog.has_function_privilege('vec_contratacion_temporal_propietario',p.oid,'EXECUTE')
          AND NOT pg_catalog.has_function_privilege(session_user,p.oid,'EXECUTE')
          AND NOT pg_catalog.has_function_privilege(g.oid,p.oid,'EXECUTE')
          AND NOT EXISTS (SELECT 1 FROM pg_catalog.aclexplode(coalesce(p.proacl,
              pg_catalog.acldefault('f',p.proowner))) a
            WHERE a.grantee NOT IN (p.proowner,'vec_contratacion_temporal_propietario'::pg_catalog.regrole)
              OR a.privilege_type<>'EXECUTE'))
      AND EXISTS (SELECT 1 FROM pg_catalog.pg_proc p
        JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
        WHERE n.nspname='vec_autorizacion_atestada_v3'
          AND p.proname='registrar_y_consumir_plantillas_doc_ct_org_v3_atestada'
          AND p.pronargs=11 AND p.prokind='f'
          AND p.proargtypes[0]='pg_catalog.jsonb'::pg_catalog.regtype
          AND p.proargtypes[1]='pg_catalog.bytea'::pg_catalog.regtype
          AND p.proargtypes[2]='pg_catalog.bytea'::pg_catalog.regtype
          AND p.proargtypes[3]='pg_catalog.bytea'::pg_catalog.regtype
          AND p.proargtypes[4]='pg_catalog.bytea'::pg_catalog.regtype
          AND p.proargtypes[5]='pg_catalog.numeric'::pg_catalog.regtype
          AND p.proargtypes[6]='pg_catalog.numeric'::pg_catalog.regtype
          AND p.proargtypes[7]='pg_catalog.bytea'::pg_catalog.regtype
          AND p.proargtypes[8]='pg_catalog.bytea'::pg_catalog.regtype
          AND p.proargtypes[9]='pg_catalog.bytea'::pg_catalog.regtype
          AND p.proargtypes[10]='pg_catalog.bytea'::pg_catalog.regtype
          AND NOT pg_catalog.has_function_privilege('vec_contratacion_temporal_propietario',p.oid,'EXECUTE')
          AND NOT pg_catalog.has_function_privilege(session_user,p.oid,'EXECUTE')
          AND NOT pg_catalog.has_function_privilege(g.oid,p.oid,'EXECUTE'))
      AND EXISTS (SELECT 1 FROM pg_catalog.pg_proc p
        JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
        WHERE n.nspname='vec_contratacion_temporal'
          AND p.proname='obtener_catalogo_plantillas_publicado_documental_v1'
          AND p.pronargs=11 AND p.prokind='f'
          AND p.proargtypes[0]='pg_catalog.jsonb'::pg_catalog.regtype
          AND p.proargtypes[1]='pg_catalog.bytea'::pg_catalog.regtype
          AND p.proargtypes[2]='pg_catalog.bytea'::pg_catalog.regtype
          AND p.proargtypes[3]='pg_catalog.bytea'::pg_catalog.regtype
          AND p.proargtypes[4]='pg_catalog.bytea'::pg_catalog.regtype
          AND p.proargtypes[5]='pg_catalog.numeric'::pg_catalog.regtype
          AND p.proargtypes[6]='pg_catalog.numeric'::pg_catalog.regtype
          AND p.proargtypes[7]='pg_catalog.bytea'::pg_catalog.regtype
          AND p.proargtypes[8]='pg_catalog.bytea'::pg_catalog.regtype
          AND p.proargtypes[9]='pg_catalog.bytea'::pg_catalog.regtype
          AND p.proargtypes[10]='pg_catalog.bytea'::pg_catalog.regtype
          AND NOT pg_catalog.has_function_privilege(session_user,p.oid,'EXECUTE')
          AND NOT pg_catalog.has_function_privilege(g.oid,p.oid,'EXECUTE'))
    ))
 )`
	var valido bool
	permitidasV3 := append([]string(nil), funcionesV3PermitidasLoginCT...)
	if err := consulta.QueryRow(ctx, sql, funciones, exigirDocumental, permitidasV3).Scan(&valido); err != nil || !valido {
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
