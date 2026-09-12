\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal.dependencias.incorporacion_ejercicio.v2',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:000070:canon:v2',0));
SET LOCAL ROLE vec_contratacion_temporal_propietario;
-- RESTRICT protege referencias catalogadas. PL/pgSQL por nombre puede no crear
-- pg_depend: también inventariamos cuerpos, todas las sobrecargas de negocio y
-- relaciones de incorporación/seguimiento. Nunca se borra historia ni se vacía
-- para permitir retirada. No se resuelven regprocedure en esquemas ajenos.
DO $guardas$
DECLARE codec oid[]; n int; nombres text[]:=ARRAY['texto_json_incorporacion_go_v2','fecha_civil_incorporacion_go_v2','instante_incorporacion_go_v2','clave_instante_incorporacion_v2','bytes_incorporacion_v2','nodo_incorporacion_canonico_v2','campo_solicitud_incorporacion_v2','solicitud_personal_incorporacion_canonica_v2','material_incorporacion_ejercicio_canonico_v2','material_incorporacion_ejercicio_sha256_v2','contexto_incorporacion_ejercicio_canonico_v2','contexto_incorporacion_ejercicio_sha256_v2','recurso_incorporacion_ejercicio_v2'];
BEGIN
 SELECT array_agg(p.oid),count(*) INTO codec,n
 FROM pg_proc p JOIN pg_namespace ns ON ns.oid=p.pronamespace
 WHERE ns.nspname='vec_contratacion_temporal' AND p.proname=ANY(nombres)
 AND p.proowner='vec_contratacion_temporal_propietario'::regrole;
 IF n<>13 OR (SELECT count(*) FROM pg_proc p JOIN pg_namespace ns ON ns.oid=p.pronamespace
   WHERE ns.nspname='vec_contratacion_temporal' AND p.proname=ANY(nombres))<>13 THEN
  RAISE EXCEPTION USING ERRCODE='55000', MESSAGE='retirada codec CT: inventario/overloads incompatible';
 END IF;
 IF EXISTS (
  SELECT 1 FROM pg_proc p JOIN pg_namespace ns ON ns.oid=p.pronamespace
  WHERE p.oid<>ALL(codec) AND (
   (ns.nspname='vec_contratacion_temporal' AND p.proname ~ '(confirmar_incorporacion|registrar_incorporacion|incorporacion_ejercicio)')
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
  RAISE EXCEPTION USING ERRCODE='55000', MESSAGE='retirada codec CT: dependencia o historia conservada';
 END IF;
END $guardas$;
DROP FUNCTION vec_contratacion_temporal.recurso_incorporacion_ejercicio_v2(jsonb) RESTRICT;
DROP FUNCTION vec_contratacion_temporal.contexto_incorporacion_ejercicio_sha256_v2(jsonb) RESTRICT;
DROP FUNCTION vec_contratacion_temporal.contexto_incorporacion_ejercicio_canonico_v2(jsonb) RESTRICT;
DROP FUNCTION vec_contratacion_temporal.material_incorporacion_ejercicio_sha256_v2(jsonb) RESTRICT;
DROP FUNCTION vec_contratacion_temporal.material_incorporacion_ejercicio_canonico_v2(jsonb) RESTRICT;
DROP FUNCTION vec_contratacion_temporal.solicitud_personal_incorporacion_canonica_v2(jsonb) RESTRICT;
DROP FUNCTION vec_contratacion_temporal.campo_solicitud_incorporacion_v2(text,text) RESTRICT;
DROP FUNCTION vec_contratacion_temporal.nodo_incorporacion_canonico_v2(jsonb,text) RESTRICT;
DROP FUNCTION vec_contratacion_temporal.bytes_incorporacion_v2(jsonb) RESTRICT;
DROP FUNCTION vec_contratacion_temporal.clave_instante_incorporacion_v2(text) RESTRICT;
DROP FUNCTION vec_contratacion_temporal.instante_incorporacion_go_v2(text,boolean) RESTRICT;
DROP FUNCTION vec_contratacion_temporal.fecha_civil_incorporacion_go_v2(text) RESTRICT;
DROP FUNCTION vec_contratacion_temporal.texto_json_incorporacion_go_v2(text) RESTRICT;
COMMIT;

