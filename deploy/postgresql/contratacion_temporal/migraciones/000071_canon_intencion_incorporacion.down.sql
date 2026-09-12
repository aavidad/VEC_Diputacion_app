\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal.dependencias.incorporacion_ejercicio.v2',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:000071:canon:intencion:v2',0));
SET LOCAL ROLE vec_contratacion_temporal_propietario;

-- No retirar si hay consumidor (incluido PL/pgSQL por nombre), sobrecargas,
-- dependencias catalogadas o relaciones de historia/seguimiento. No consulta
-- tablas ni resuelve firmas en esquemas ajenos; no borra datos para habilitar DOWN.
DO $guardas$
DECLARE codec oid[]; n int; nombres text[]:=ARRAY[
 'intencion_registro_incorporacion_canonica_v2','intencion_registro_incorporacion_sha256_v2'];
BEGIN
 SELECT array_agg(p.oid),count(*) INTO codec,n
 FROM pg_proc p JOIN pg_namespace ns ON ns.oid=p.pronamespace
 WHERE ns.nspname='vec_contratacion_temporal' AND p.proname=ANY(nombres)
  AND p.proowner='vec_contratacion_temporal_propietario'::regrole
  AND p.prokind='f' AND p.pronargs=1 AND p.proargtypes[0]='jsonb'::regtype
  AND p.prorettype=CASE WHEN p.proname='intencion_registro_incorporacion_canonica_v2'
   THEN 'bytea'::regtype ELSE 'text'::regtype END;
 IF n<>2 OR (SELECT count(*) FROM pg_proc p JOIN pg_namespace ns ON ns.oid=p.pronamespace
  WHERE ns.nspname='vec_contratacion_temporal' AND p.proname=ANY(nombres))<>2 THEN
  RAISE EXCEPTION USING ERRCODE='55000', MESSAGE='retirada intencion CT: inventario/overloads incompatible';
 END IF;
 IF EXISTS (
  SELECT 1 FROM pg_proc p JOIN pg_namespace ns ON ns.oid=p.pronamespace
  WHERE p.oid<>ALL(codec) AND (
   (ns.nspname='vec_contratacion_temporal' AND p.proname ~ '^(registrar|confirmar|recuperar).*incorporacion')
   OR EXISTS (SELECT 1 FROM unnest(nombres) k WHERE strpos(p.prosrc,k)>0)
  )
 ) OR EXISTS (
  SELECT 1 FROM pg_class c JOIN pg_namespace ns ON ns.oid=c.relnamespace
  WHERE ns.nspname='vec_contratacion_temporal' AND c.relkind IN ('r','p','v','m','f','S')
   AND c.relname ~ '(incorporacion|seguimiento)'
 ) OR EXISTS (
  SELECT 1 FROM pg_depend d WHERE d.refclassid='pg_proc'::regclass AND d.refobjid=ANY(codec)
   AND NOT (d.classid='pg_proc'::regclass AND d.objid=ANY(codec))
 ) THEN
  RAISE EXCEPTION USING ERRCODE='55000', MESSAGE='retirada intencion CT: dependencia o historia conservada';
 END IF;
END $guardas$;
DROP FUNCTION vec_contratacion_temporal.intencion_registro_incorporacion_sha256_v2(jsonb) RESTRICT;
DROP FUNCTION vec_contratacion_temporal.intencion_registro_incorporacion_canonica_v2(jsonb) RESTRICT;
COMMIT;
