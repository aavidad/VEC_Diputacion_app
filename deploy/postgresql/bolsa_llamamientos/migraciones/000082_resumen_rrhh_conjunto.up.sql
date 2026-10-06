\set ON_ERROR_STOP on
-- Bolsa 000082: lecturas de conjunto para el cuadro y las estadísticas de RRHH.
-- El cuadro pedía, por cada participación, leer_situacion_participacion_v1 y
-- consultar_estado_cese_bolsa_v1, y por cada bolsa leer_orden_vigente_bolsa_v1
-- solo para conocer su política. Con miles de participaciones eran miles de
-- idas y vueltas. Estas dos funciones devuelven lo mismo en una consulta cada
-- una. Solo lectura: no se modifica ninguna función, tabla ni ACL instalada.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:migracion:000082',0));

DO $precondicion$
DECLARE actual jsonb; esperado jsonb:=pg_catalog.jsonb_build_object(
 'rol',true,'constituciones',true,'entradas',true,'situaciones',true,'cese',true,'politicas',true,
 'resumen_libre',true,'politicas_libre',true);
BEGIN
 actual:=pg_catalog.jsonb_build_object(
  'rol',current_user='vec_bolsa_llamamientos_propietario',
  'constituciones',pg_catalog.to_regprocedure('vec_bolsa_llamamientos.listar_constituciones_v1()') IS NOT NULL,
  'entradas',pg_catalog.to_regclass('vec_bolsa_llamamientos.constitucion_entrada') IS NOT NULL,
  'situaciones',pg_catalog.to_regclass('vec_bolsa_llamamientos.situacion_participacion') IS NOT NULL,
  'cese',pg_catalog.to_regprocedure('vec_bolsa_llamamientos.estado_cese_bolsa_v1(text,timestamp with time zone)') IS NOT NULL
    AND pg_catalog.to_regclass('vec_bolsa_llamamientos.restriccion_cese_bolsa') IS NOT NULL
    AND pg_catalog.to_regclass('vec_bolsa_llamamientos.vinculo_candidato') IS NOT NULL,
  'politicas',pg_catalog.to_regclass('vec_bolsa_llamamientos.politica_orden_bolsa') IS NOT NULL,
  'resumen_libre',pg_catalog.to_regprocedure('vec_bolsa_llamamientos.leer_resumen_situaciones_bolsas_v1(timestamp with time zone)') IS NULL,
  'politicas_libre',pg_catalog.to_regprocedure('vec_bolsa_llamamientos.leer_politicas_orden_vigentes_v1(timestamp with time zone)') IS NULL);
 IF actual IS DISTINCT FROM esperado THEN
  RAISE EXCEPTION 'PARO clave=B82.preimagen, actual=%, esperado=%',actual,esperado USING ERRCODE='55000';
 END IF;
END $precondicion$;

-- Situación vigente y estado de cese de todas las participaciones de las
-- constituciones vigentes (las de listar_constituciones_v1, en su mismo orden
-- por categoría), con la bolsa, la categoría y la fecha de constitución, para
-- no tener que descargar la instantánea canónica completa de cada bolsa.
-- - situación: misma selección que leer_situacion_participacion_v1 (la de
--   «desde» más reciente, sin corte); NULL si no hay ninguna.
-- - cese: exactamente estado_cese_bolsa_v1(participación, corte). Esa función
--   solo devuelve fila si el candidato vinculado tiene alguna restricción de
--   cese con fecha de efecto hasta el día del corte en Madrid; «con_restriccion»
--   es esa misma condición necesaria y evita llamarla para quien no tiene
--   ninguna (la inmensa mayoría). estado_cese_bolsa_v1 devuelve como mucho una
--   fila, así que el LEFT JOIN no duplica participaciones.
-- Misma guarda de rol que consultar_estado_cese_bolsa_v1 (Bolsa 000050).
CREATE FUNCTION vec_bolsa_llamamientos.leer_resumen_situaciones_bolsas_v1(p_corte timestamptz)
RETURNS TABLE(bolsa_ref text,categoria_ref text,confirmada_en timestamptz,
 instantanea_ref text,version_instantanea bigint,orden bigint,participacion_ref text,
 situacion text,desde timestamptz,fecha_disponible timestamptz,
 cese_fecha_efecto date,cese_disponible_desde date,cese_en_restriccion boolean,cese_trabajo_cesado boolean)
LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path=pg_catalog AS $f$
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_migrador','MEMBER')
    OR p_corte IS NULL OR NOT isfinite(p_corte) THEN
  RAISE EXCEPTION 'lectura del resumen de bolsas denegada' USING ERRCODE='42501';
 END IF;
 RETURN QUERY
 WITH con_restriccion AS MATERIALIZED (
   SELECT DISTINCT vc.participacion_ref
     FROM vec_bolsa_llamamientos.vinculo_candidato vc
     JOIN vec_bolsa_llamamientos.restriccion_cese_bolsa r ON r.candidato_ref=vc.candidato_ref
    WHERE r.fecha_efecto<=(p_corte AT TIME ZONE 'Europe/Madrid')::date
 ), ceses AS MATERIALIZED (
   SELECT cr.participacion_ref,ec.fecha_efecto,ec.disponible_desde,ec.en_restriccion,ec.trabajo_cesado
     FROM con_restriccion cr
    CROSS JOIN LATERAL vec_bolsa_llamamientos.estado_cese_bolsa_v1(cr.participacion_ref,p_corte) ec
 )
 SELECT k.bolsa_ref,k.categoria_ref,k.confirmada_en,
        e.instantanea_ref,e.version_instantanea,e.orden,e.participacion_ref,
        s.situacion,s.desde,s.fecha_disponible,
        x.fecha_efecto,x.disponible_desde,x.en_restriccion,x.trabajo_cesado
   FROM (SELECT DISTINCT c.bolsa_ref,c.categoria_ref,c.confirmada_en,c.instantanea_ref,c.version_instantanea
           FROM vec_bolsa_llamamientos.listar_constituciones_v1() c) k
   JOIN vec_bolsa_llamamientos.constitucion_entrada e
     ON e.instantanea_ref=k.instantanea_ref AND e.version_instantanea=k.version_instantanea
   LEFT JOIN LATERAL (
     SELECT sp.situacion,sp.desde,sp.fecha_disponible
       FROM vec_bolsa_llamamientos.situacion_participacion sp
      WHERE sp.participacion_ref=e.participacion_ref
      ORDER BY sp.desde DESC LIMIT 1) s ON true
   LEFT JOIN ceses x ON x.participacion_ref=e.participacion_ref
  ORDER BY k.categoria_ref,e.orden;
END $f$;

-- Política de orden vigente de cada bolsa en el instante dado: misma selección
-- que el CTE «politica» de leer_orden_vigente_bolsa_v1.
CREATE FUNCTION vec_bolsa_llamamientos.leer_politicas_orden_vigentes_v1(p_en timestamptz)
RETURNS TABLE(bolsa_ref text,politica_ref text,version_politica bigint,criterio text,tipo_lista text,
 reposicion text,provisional boolean,rotulo text,actor text,vigente_desde timestamptz)
LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path=pg_catalog AS $f$
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_migrador','MEMBER')
    OR p_en IS NULL OR NOT isfinite(p_en) THEN
  RAISE EXCEPTION 'lectura de políticas de orden denegada' USING ERRCODE='42501';
 END IF;
 RETURN QUERY
 SELECT DISTINCT ON (p.bolsa_ref) p.bolsa_ref,p.politica_ref,p.version,p.criterio,p.tipo_lista,
        p.reposicion,p.provisional,p.rotulo,p.actor,p.vigente_desde
   FROM vec_bolsa_llamamientos.politica_orden_bolsa p
  WHERE p.vigente_desde<=p_en AND (p.vigente_hasta IS NULL OR p.vigente_hasta>p_en)
  ORDER BY p.bolsa_ref,p.version DESC;
END $f$;

ALTER FUNCTION vec_bolsa_llamamientos.leer_resumen_situaciones_bolsas_v1(timestamptz) ROWS 10000;
ALTER FUNCTION vec_bolsa_llamamientos.leer_politicas_orden_vigentes_v1(timestamptz) ROWS 50;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.leer_resumen_situaciones_bolsas_v1(timestamptz) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.leer_politicas_orden_vigentes_v1(timestamptz) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.leer_resumen_situaciones_bolsas_v1(timestamptz) TO vec_bolsa_llamamientos_ejecutor;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.leer_politicas_orden_vigentes_v1(timestamptz) TO vec_bolsa_llamamientos_ejecutor;

DO $acl$
DECLARE actual jsonb; esperado jsonb:=pg_catalog.jsonb_build_object(
 'resumen_ejecutor',true,'politicas_ejecutor',true,'resumen_public',false,'politicas_public',false,
 'resumen_relevo',false,'definidoras',true,'cese_interno_cerrado',true);
BEGIN
 actual:=pg_catalog.jsonb_build_object(
  'resumen_ejecutor',pg_catalog.has_function_privilege('vec_bolsa_llamamientos_ejecutor',
     'vec_bolsa_llamamientos.leer_resumen_situaciones_bolsas_v1(timestamptz)','EXECUTE'),
  'politicas_ejecutor',pg_catalog.has_function_privilege('vec_bolsa_llamamientos_ejecutor',
     'vec_bolsa_llamamientos.leer_politicas_orden_vigentes_v1(timestamptz)','EXECUTE'),
  'resumen_public',EXISTS (SELECT 1 FROM pg_catalog.pg_proc p, pg_catalog.aclexplode(p.proacl) a
     WHERE p.oid='vec_bolsa_llamamientos.leer_resumen_situaciones_bolsas_v1(timestamptz)'::pg_catalog.regprocedure AND a.grantee=0),
  'politicas_public',EXISTS (SELECT 1 FROM pg_catalog.pg_proc p, pg_catalog.aclexplode(p.proacl) a
     WHERE p.oid='vec_bolsa_llamamientos.leer_politicas_orden_vigentes_v1(timestamptz)'::pg_catalog.regprocedure AND a.grantee=0),
  'resumen_relevo',pg_catalog.to_regrole('vec_bolsa_llamamientos_relevo_cese') IS NOT NULL
     AND pg_catalog.has_function_privilege('vec_bolsa_llamamientos_relevo_cese',
     'vec_bolsa_llamamientos.leer_resumen_situaciones_bolsas_v1(timestamptz)','EXECUTE'),
  'definidoras',(SELECT pg_catalog.bool_and(p.prosecdef AND p.proowner='vec_bolsa_llamamientos_propietario'::pg_catalog.regrole
       AND p.proconfig=ARRAY['search_path=pg_catalog'])
     FROM pg_catalog.pg_proc p WHERE p.oid IN (
       'vec_bolsa_llamamientos.leer_resumen_situaciones_bolsas_v1(timestamptz)'::pg_catalog.regprocedure,
       'vec_bolsa_llamamientos.leer_politicas_orden_vigentes_v1(timestamptz)'::pg_catalog.regprocedure)),
  'cese_interno_cerrado',NOT pg_catalog.has_function_privilege('vec_bolsa_llamamientos_ejecutor',
     'vec_bolsa_llamamientos.estado_cese_bolsa_v1(text,timestamptz)','EXECUTE'));
 IF actual IS DISTINCT FROM esperado THEN
  RAISE EXCEPTION 'PARO clave=B82.acl, actual=%, esperado=%',actual,esperado USING ERRCODE='42501';
 END IF;
END $acl$;
COMMIT;
