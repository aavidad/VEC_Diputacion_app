\set ON_ERROR_STOP on
-- B94: detalle mínimo de los mismos llamamientos completos del recuento B85/B86.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='2s';
SET LOCAL statement_timeout='30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:migracion:000094',0));

DO $pre$
DECLARE f oid:=pg_catalog.to_regprocedure('vec_bolsa_llamamientos.leer_llamamientos_en_curso_bolsas_v1()');
BEGIN
 IF pg_catalog.current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
    OR current_user<>'vec_bolsa_llamamientos_propietario'
    OR f IS NULL
    OR pg_catalog.to_regprocedure('vec_bolsa_llamamientos.listar_constituciones_v1()') IS NULL
    OR pg_catalog.to_regclass('vec_bolsa_llamamientos.completitud_correo_llamamiento') IS NULL
    OR pg_catalog.to_regclass('vec_bolsa_llamamientos.llamamiento_emitido') IS NULL
    OR pg_catalog.to_regprocedure('vec_bolsa_llamamientos.leer_llamamientos_completos_resumen_v1()') IS NOT NULL
 THEN RAISE EXCEPTION 'B94: preimagen incompatible' USING ERRCODE='55000'; END IF;
 IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_proc p WHERE p.oid=f
      AND p.proowner='vec_bolsa_llamamientos_propietario'::pg_catalog.regrole
      AND p.prosecdef AND p.provolatile='s'
      AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s'])
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_proc p
      CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
      WHERE p.oid=f AND (a.grantee NOT IN (p.proowner,'vec_bolsa_llamamientos_ejecutor'::pg_catalog.regrole)
        OR a.grantor<>p.proowner OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
    OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_proc p
      CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
      WHERE p.oid=f)<>2
 THEN RAISE EXCEPTION 'B94: B86 ACL o metadata incompatible' USING ERRCODE='42501'; END IF;
END $pre$;

CREATE FUNCTION vec_bolsa_llamamientos.leer_llamamientos_completos_resumen_v1()
RETURNS TABLE(llamamiento_ref text,bolsa_ref text,referencia text,emitido_en timestamptz,participaciones integer)
LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET lock_timeout='2s' AS $f$
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR session_user=current_user
    OR NOT pg_catalog.pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER')
    OR pg_catalog.pg_has_role(session_user,'vec_bolsa_llamamientos_propietario','MEMBER')
    OR pg_catalog.pg_has_role(session_user,'vec_bolsa_llamamientos_migrador','MEMBER')
 THEN RAISE EXCEPTION 'B94: lectura de llamamientos denegada' USING ERRCODE='42501'; END IF;
 RETURN QUERY
 WITH bolsas AS MATERIALIZED (
   SELECT DISTINCT k.bolsa_ref FROM vec_bolsa_llamamientos.listar_constituciones_v1() k
 )
 SELECT l.llamamiento_ref,l.bolsa_ref,l.configuracion->>'referencia',l.emitido_en,
        pg_catalog.jsonb_array_length(l.participaciones)
 FROM bolsas b
 JOIN vec_bolsa_llamamientos.completitud_correo_llamamiento m ON m.bolsa_ref=b.bolsa_ref
 JOIN vec_bolsa_llamamientos.llamamiento_emitido l
   ON l.llamamiento_ref=m.llamamiento_ref AND l.bolsa_ref=m.bolsa_ref
 ORDER BY l.emitido_en DESC,l.llamamiento_ref;
END $f$;

REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.leer_llamamientos_completos_resumen_v1() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.leer_llamamientos_completos_resumen_v1() TO vec_bolsa_llamamientos_ejecutor;
DO $acl$
DECLARE f oid:='vec_bolsa_llamamientos.leer_llamamientos_completos_resumen_v1()'::pg_catalog.regprocedure;
BEGIN
 IF NOT pg_catalog.has_function_privilege('vec_bolsa_llamamientos_ejecutor',f,'EXECUTE')
    OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_proc p
      CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
      WHERE p.oid=f)<>2
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_proc p
      CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
      WHERE p.oid=f AND (a.grantee NOT IN (p.proowner,'vec_bolsa_llamamientos_ejecutor'::pg_catalog.regrole)
        OR a.grantor<>p.proowner OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
    OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_proc p WHERE p.oid=f
      AND p.proowner='vec_bolsa_llamamientos_propietario'::pg_catalog.regrole
      AND p.prosecdef AND p.provolatile='s'
      AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s'])
 THEN RAISE EXCEPTION 'B94: ACL o metadata incompatible' USING ERRCODE='42501'; END IF;
END $acl$;
COMMIT;
