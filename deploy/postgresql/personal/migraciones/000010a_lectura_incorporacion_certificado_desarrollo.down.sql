\set ON_ERROR_STOP on
-- Solo ensayo en PostgreSQL desechable sin lecturas históricas. Nunca sobre la
-- base conservada de desarrollo o presentación.
BEGIN;
SET LOCAL ROLE vec_personal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal.dependencias.alta_ejercicio.v1',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal:migracion:000010a:certificado-desarrollo',0));

DO $retirada$
DECLARE f oid:=to_regprocedure('vec_personal.acreditar_alta_ejercicio_v2(text,text,text,bigint,text,text,text,text,text,text,bytea,bytea,bytea,bytea,bigint,bigint,bytea,bytea,bytea,bytea)');
  helper oid:=to_regprocedure('vec_personal.admite_lectura_incorporacion_certificado_desarrollo_v1(jsonb,timestamptz)');
  actual text; previo text;
  viejo text:=$x$d#>>'{vinculo_autenticacion_actor,garantia_observada}' IS DISTINCT FROM 'alto'$x$;
  nuevo text:=$x$(d#>>'{vinculo_autenticacion_actor,garantia_observada}' IS DISTINCT FROM 'alto'
           AND vec_personal.admite_lectura_incorporacion_certificado_desarrollo_v1(
               d->'vinculo_autenticacion_actor',inicio) IS DISTINCT FROM true)$x$;
  retorno text:=$x$    RETURN respuesta;$x$;
  retorno_nuevo text:=$x$    IF d#>>'{vinculo_autenticacion_actor,garantia_observada}' IS NOT DISTINCT FROM 'sustancial'
       AND vec_personal.admite_lectura_incorporacion_certificado_desarrollo_v1(
           d->'vinculo_autenticacion_actor',clock_timestamp()) IS DISTINCT FROM true THEN
        RAISE EXCEPTION 'política de lectura Personal retirada' USING ERRCODE='42501';
    END IF;
    RETURN respuesta;$x$;
BEGIN
 IF current_user<>'vec_personal_propietario' OR f IS NULL OR helper IS NULL
    OR EXISTS (SELECT 1 FROM vec_personal.auditoria_lectura_incorporacion)
    OR obj_description(f,'pg_proc') IS DISTINCT FROM
      'Personal000010a:lector_incorporacion:v2:certificado_desarrollo_gobernado'
    OR EXISTS (SELECT 1 FROM pg_depend WHERE refclassid='pg_proc'::regclass AND refobjid=helper AND objid<>f) THEN
   RAISE EXCEPTION 'Personal 000010a DOWN protege historia o dependencias' USING ERRCODE='55000';
 END IF;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF length(actual)-length(replace(actual,nuevo,''))<>length(nuevo)
    OR length(actual)-length(replace(actual,retorno_nuevo,''))<>length(retorno_nuevo) THEN
   RAISE EXCEPTION 'Personal 000010a DOWN: definición incompatible' USING ERRCODE='55000';
 END IF;
 previo:=replace(replace(actual,nuevo,viejo),retorno_nuevo,retorno);
 EXECUTE previo;
 IF NOT EXISTS (SELECT 1 FROM pg_proc WHERE oid=f AND octet_length(prosrc)=18515
   AND encode(sha256(convert_to(prosrc,'UTF8')),'hex')=
       '38e12724f1addf6dbec363394f941641369bfb4dd8906eeb3f469eb243266073') THEN
   RAISE EXCEPTION 'Personal 000010a DOWN: función previa no restaurada' USING ERRCODE='55000';
 END IF;
END $retirada$;
COMMENT ON FUNCTION vec_personal.acreditar_alta_ejercicio_v2(text,text,text,bigint,text,text,text,text,text,text,bytea,bytea,bytea,bytea,bigint,bigint,bytea,bytea,bytea,bytea)
 IS 'Personal000005:lector_incorporacion:v2:ambitos_org_unidad';
DROP FUNCTION vec_personal.admite_lectura_incorporacion_certificado_desarrollo_v1(jsonb,timestamptz) RESTRICT;
COMMIT;
