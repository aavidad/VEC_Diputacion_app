\set ON_ERROR_STOP on
-- Ejecutar tras B86 en un clon PostgreSQL 18, con o sin bolsas.
BEGIN ISOLATION LEVEL REPEATABLE READ READ ONLY;
SET LOCAL search_path=pg_catalog;
DO $integridad$
DECLARE divergencias bigint; duplicados bigint;
BEGIN
 SELECT pg_catalog.count(*) INTO divergencias
 FROM vec_bolsa_llamamientos.llamamiento_emitido l
 LEFT JOIN vec_bolsa_llamamientos.completitud_correo_llamamiento m
   ON m.llamamiento_ref=l.llamamiento_ref
 WHERE ((SELECT pg_catalog.count(*)
   FROM pg_catalog.jsonb_array_elements_text(l.participaciones) WITH ORDINALITY x(ref,ordinality)
   JOIN vec_bolsa_llamamientos.contacto_participacion c
     ON c.participacion_ref=x.ref AND c.llamamiento_ref=l.llamamiento_ref
    AND c.canal='correo'
    AND c.clave_idempotencia=l.clave_idempotencia||':correo:'||x.ordinality
    AND c.recibo_ref='recibo:contacto:'||pg_catalog.encode(pg_catalog.sha256(
      pg_catalog.convert_to(l.bolsa_ref||pg_catalog.chr(31)||l.clave_idempotencia||
        pg_catalog.chr(31)||x.ref,'UTF8')),'hex'))
    =pg_catalog.jsonb_array_length(l.participaciones))
     IS DISTINCT FROM (m.llamamiento_ref IS NOT NULL)
    OR (m.llamamiento_ref IS NOT NULL AND m.bolsa_ref IS DISTINCT FROM l.bolsa_ref);
 SELECT pg_catalog.count(*) INTO duplicados
 FROM vec_bolsa_llamamientos.completitud_correo_llamamiento m
 LEFT JOIN vec_bolsa_llamamientos.llamamiento_emitido l ON l.llamamiento_ref=m.llamamiento_ref
 WHERE l.llamamiento_ref IS NULL;
 IF divergencias<>0 OR duplicados<>0 THEN
   RAISE EXCEPTION 'B86: proyeccion divergente: %, %',divergencias,duplicados USING ERRCODE='55000';
 END IF;
END $integridad$;

DO $acl$
DECLARE f oid:='vec_bolsa_llamamientos.leer_llamamientos_en_curso_bolsas_v1()'::pg_catalog.regprocedure;
BEGIN
 IF NOT pg_catalog.has_function_privilege('vec_bolsa_llamamientos_ejecutor',f,'EXECUTE')
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_proc p
       CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,
         pg_catalog.acldefault('f',p.proowner))) a
       WHERE p.oid='vec_bolsa_llamamientos.proyectar_completitud_correo_insert_v1()'::pg_catalog.regprocedure
         AND a.grantee=0)
    OR pg_catalog.has_table_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos.completitud_correo_llamamiento','SELECT')
    OR pg_catalog.has_table_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos.coordinacion_completitud_correo','SELECT')
    OR pg_catalog.has_type_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos.completitud_correo_llamamiento','USAGE')
    OR pg_catalog.has_type_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos.coordinacion_completitud_correo','USAGE')
    OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_class WHERE oid='vec_bolsa_llamamientos.completitud_correo_llamamiento'::pg_catalog.regclass
      AND relrowsecurity AND relforcerowsecurity)
    OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_class WHERE oid='vec_bolsa_llamamientos.coordinacion_completitud_correo'::pg_catalog.regclass
      AND relrowsecurity AND relforcerowsecurity)
 THEN RAISE EXCEPTION 'B86: ACL/RLS incompatible' USING ERRCODE='42501'; END IF;
END $acl$;
ROLLBACK;

SET SESSION AUTHORIZATION vec_bolsa_llamamientos_desarrollo;
BEGIN ISOLATION LEVEL REPEATABLE READ READ ONLY;
SET LOCAL search_path=pg_catalog;
DO $paridad$
DECLARE divergencias bigint; repetidas bigint;
BEGIN
 SELECT pg_catalog.count(*) INTO divergencias FROM (
   SELECT k.bolsa_ref,
     vec_bolsa_llamamientos.contar_llamamientos_en_curso_v1(k.bolsa_ref) anterior,
     r.llamamientos_en_curso actual
   FROM (SELECT DISTINCT c.bolsa_ref FROM vec_bolsa_llamamientos.listar_constituciones_v1() c) k
   FULL JOIN vec_bolsa_llamamientos.leer_llamamientos_en_curso_bolsas_v1() r
     ON r.bolsa_ref=k.bolsa_ref
 ) x WHERE x.bolsa_ref IS NULL OR x.anterior IS DISTINCT FROM x.actual OR x.actual<0;
 SELECT pg_catalog.count(*) INTO repetidas FROM (
   SELECT r.bolsa_ref FROM vec_bolsa_llamamientos.leer_llamamientos_en_curso_bolsas_v1() r
   GROUP BY r.bolsa_ref HAVING pg_catalog.count(*)<>1
 ) x;
 -- Cero bolsas tambien es un resultado valido: ninguna de estas comparaciones
 -- impone que exista una constitucion en el clon.
 IF divergencias<>0 OR repetidas<>0 THEN
   RAISE EXCEPTION 'B86: recuento agrupado divergente: %, %',divergencias,repetidas USING ERRCODE='55000';
 END IF;
END $paridad$;
DO $denegacion$
BEGIN
 BEGIN
   PERFORM 1 FROM vec_bolsa_llamamientos.completitud_correo_llamamiento LIMIT 1;
   RAISE EXCEPTION 'B86: lectura directa inesperada' USING ERRCODE='55000';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 BEGIN
   PERFORM 1 FROM vec_bolsa_llamamientos.coordinacion_completitud_correo LIMIT 1;
   RAISE EXCEPTION 'B86: lectura directa inesperada' USING ERRCODE='55000';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $denegacion$;
ROLLBACK;
RESET SESSION AUTHORIZATION;
