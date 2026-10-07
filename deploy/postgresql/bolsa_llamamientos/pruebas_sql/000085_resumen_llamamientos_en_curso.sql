\set ON_ERROR_STOP on
-- Ensayo de solo lectura en clon sintético PG18 tras B85.
BEGIN READ ONLY;
SET LOCAL search_path=pg_catalog;
DO $acl$
DECLARE f oid:=pg_catalog.to_regprocedure('vec_bolsa_llamamientos.leer_llamamientos_en_curso_bolsas_v1()');
BEGIN
 IF f IS NULL
    OR NOT pg_catalog.has_function_privilege('vec_bolsa_llamamientos_ejecutor',f,'EXECUTE')
    OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_proc p
       CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,
         pg_catalog.acldefault('f',p.proowner))) a WHERE p.oid=f)<>2
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_proc p
       CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,
         pg_catalog.acldefault('f',p.proowner))) a WHERE p.oid=f
       AND (a.grantee NOT IN (p.proowner,'vec_bolsa_llamamientos_ejecutor'::pg_catalog.regrole)
         OR a.grantor<>p.proowner OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
    OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_proc p WHERE p.oid=f
       AND p.proowner='vec_bolsa_llamamientos_propietario'::pg_catalog.regrole
       AND p.prosecdef AND p.provolatile='s'
       AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s'])
 THEN RAISE EXCEPTION 'B85 prueba: ACL incompatible' USING ERRCODE='42501'; END IF;
END $acl$;
ROLLBACK;

SET SESSION AUTHORIZATION vec_bolsa_llamamientos_desarrollo;
BEGIN ISOLATION LEVEL REPEATABLE READ READ ONLY;
SET LOCAL search_path=pg_catalog;
DO $paridad$
DECLARE bolsas integer; distintas integer; incorrectas integer;
BEGIN
 SELECT pg_catalog.count(*),pg_catalog.count(DISTINCT r.bolsa_ref) INTO bolsas,distintas
 FROM vec_bolsa_llamamientos.leer_llamamientos_en_curso_bolsas_v1() r;
 SELECT pg_catalog.count(*) INTO incorrectas FROM (
   SELECT k.bolsa_ref,
     vec_bolsa_llamamientos.contar_llamamientos_en_curso_v1(k.bolsa_ref) anterior,
     r.llamamientos_en_curso actual
   FROM (SELECT DISTINCT c.bolsa_ref FROM vec_bolsa_llamamientos.listar_constituciones_v1() c) k
   FULL JOIN vec_bolsa_llamamientos.leer_llamamientos_en_curso_bolsas_v1() r
     ON r.bolsa_ref=k.bolsa_ref
 ) x WHERE x.bolsa_ref IS NULL OR x.anterior IS DISTINCT FROM x.actual OR x.actual<0;
 IF bolsas=0 OR bolsas<>distintas OR incorrectas<>0 THEN
   RAISE EXCEPTION 'B85 prueba: recuento agrupado divergente' USING ERRCODE='55000'; END IF;
END $paridad$;
ROLLBACK;
RESET SESSION AUTHORIZATION;
