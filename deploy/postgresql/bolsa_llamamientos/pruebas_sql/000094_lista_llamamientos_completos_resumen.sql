\set ON_ERROR_STOP on
-- Clon PG18 con B85/B86/B94; solo lectura, sin modificar historia.
BEGIN ISOLATION LEVEL REPEATABLE READ READ ONLY;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $acl$
DECLARE f oid:=pg_catalog.to_regprocedure('vec_bolsa_llamamientos.leer_llamamientos_completos_resumen_v1()');
BEGIN
 IF f IS NULL OR NOT pg_catalog.has_function_privilege('vec_bolsa_llamamientos_ejecutor',f,'EXECUTE')
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
 THEN RAISE EXCEPTION 'B94 prueba: ACL/metadata incompatibles' USING ERRCODE='42501'; END IF;
END $acl$;
ROLLBACK;

SET SESSION AUTHORIZATION vec_bolsa_llamamientos_desarrollo;
BEGIN ISOLATION LEVEL REPEATABLE READ READ ONLY;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $paridad$
DECLARE divergencias bigint; duplicados bigint;
BEGIN
 SELECT pg_catalog.count(*) INTO divergencias FROM (
   SELECT coalesce(c.bolsa_ref,l.bolsa_ref) bolsa_ref,c.llamamientos_en_curso,l.total
   FROM vec_bolsa_llamamientos.leer_llamamientos_en_curso_bolsas_v1() c
   FULL JOIN (
     SELECT r.bolsa_ref,pg_catalog.count(*) total
     FROM vec_bolsa_llamamientos.leer_llamamientos_completos_resumen_v1() r
     GROUP BY r.bolsa_ref
   ) l ON l.bolsa_ref=c.bolsa_ref
 ) x WHERE x.llamamientos_en_curso IS NULL
   OR coalesce(x.total,0)<>x.llamamientos_en_curso;
 SELECT pg_catalog.count(*) INTO duplicados FROM (
   SELECT l.llamamiento_ref FROM vec_bolsa_llamamientos.leer_llamamientos_completos_resumen_v1() l
   GROUP BY l.llamamiento_ref HAVING pg_catalog.count(*)<>1
 ) x;
 IF divergencias<>0 OR duplicados<>0 THEN
   RAISE EXCEPTION 'B94 prueba: lista y recuento divergentes %, %',divergencias,duplicados USING ERRCODE='55000';
 END IF;
END $paridad$;
DO $denegacion$
BEGIN
 BEGIN
   PERFORM 1 FROM vec_bolsa_llamamientos.completitud_correo_llamamiento LIMIT 1;
   RAISE EXCEPTION 'B94 prueba: tabla de completitud accesible' USING ERRCODE='55000';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 BEGIN
   PERFORM 1 FROM vec_bolsa_llamamientos.llamamiento_emitido LIMIT 1;
   RAISE EXCEPTION 'B94 prueba: tabla de emisiones accesible' USING ERRCODE='55000';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $denegacion$;
ROLLBACK;
RESET SESSION AUTHORIZATION;

SET SESSION AUTHORIZATION vec_bolsa_llamamientos_propietario;
BEGIN READ ONLY;
DO $denegacion_propietario$
BEGIN
 BEGIN
   PERFORM 1 FROM vec_bolsa_llamamientos.leer_llamamientos_completos_resumen_v1();
   RAISE EXCEPTION 'B94 prueba: llamada directa del propietario aceptada' USING ERRCODE='55000';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $denegacion_propietario$;
ROLLBACK;
RESET SESSION AUTHORIZATION;
