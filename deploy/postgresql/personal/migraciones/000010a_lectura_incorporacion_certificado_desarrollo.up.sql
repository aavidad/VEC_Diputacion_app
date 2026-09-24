\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_personal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal.dependencias.alta_ejercicio.v1',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal:migracion:000010a:certificado-desarrollo',0));

-- BASE10 y BASE11 instalan Organización/Personal antes de esta guarda. Su función y
-- tabla tienen propietario/ACL propios; no se reescriben aquí.
DO $base10$
DECLARE t oid:=to_regclass('vec_personal.org_nodo_historia');
  f oid:=to_regprocedure('vec_personal.consultar_organizacion_historica_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
  o oid:='vec_personal_propietario'::regrole;
  e oid:='vec_personal_ejecutor'::regrole;
BEGIN
 IF t IS NULL OR f IS NULL
    OR to_regclass('vec_personal.importacion_organizacion_revision') IS NULL
    OR to_regprocedure('vec_personal.ejecutar_importacion_organizacion_v1(text,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR NOT EXISTS (SELECT 1 FROM pg_class WHERE oid=t AND relkind='r'
       AND relowner=o AND relrowsecurity AND relforcerowsecurity)
    OR (SELECT count(*) FROM pg_policy WHERE polrelid=t)<>1
    OR NOT EXISTS (SELECT 1 FROM pg_policy WHERE polrelid=t
       AND polname='propietario_interno' AND polcmd='*' AND polpermissive
       AND polroles=ARRAY[o] AND pg_get_expr(polqual,polrelid)='true'
       AND pg_get_expr(polwithcheck,polrelid)='true')
    OR EXISTS (SELECT 1 FROM pg_class c,
       LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) a
       WHERE c.oid=t AND a.grantee<>o)
    OR (SELECT count(*) FROM pg_trigger WHERE tgrelid=t AND NOT tgisinternal)<>2
    OR NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgrelid=t
       AND tgname='historia_inmutable' AND tgenabled='O'
       AND tgfoid='vec_personal.rechazar_mutacion_organizacion_v1()'::regprocedure)
    OR NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgrelid=t
       AND tgname='no_truncar' AND tgenabled='O'
       AND tgfoid='vec_personal.rechazar_mutacion_organizacion_v1()'::regprocedure)
    OR NOT EXISTS (SELECT 1 FROM pg_proc WHERE oid=f AND proowner=o
       AND prosecdef AND provolatile='v' AND prorettype='jsonb'::regtype
       AND octet_length(prosrc)=20430
       AND encode(sha256(convert_to(prosrc,'UTF8')),'hex')=
          '22b6ddf4ce19d01ea7440745043d4cd2a72c00e0960842c12b5fc618f11d32b5'
       AND obj_description(oid,'pg_proc')=
          'Consulta B3 con autorización V3 consumida y auditoría; historia estructural sin ocupaciones ni vacantes.')
    OR EXISTS (SELECT 1 FROM pg_proc p,
       LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
       WHERE p.oid=f AND (a.grantor<>o OR a.is_grantable
         OR a.privilege_type<>'EXECUTE' OR a.grantee NOT IN (o,e)))
    OR NOT has_function_privilege(e,f,'EXECUTE') THEN
   RAISE EXCEPTION 'Personal 000010a: BASE10/11 ausente o alterada' USING ERRCODE='55000';
 END IF;
END $base10$;

DO $pre$
DECLARE f oid:=to_regprocedure('vec_personal.acreditar_alta_ejercicio_v2(text,text,text,bigint,text,text,text,text,text,text,bytea,bytea,bytea,bytea,bigint,bigint,bytea,bytea,bytea,bytea)');
  politica oid:=to_regprocedure('vec_identidad_sesiones_v1.admite_politica_certificado_personal_desarrollo_v1(text,text,timestamptz)');
BEGIN
 IF current_user<>'vec_personal_propietario' OR f IS NULL
    OR to_regprocedure('vec_personal.admite_lectura_incorporacion_certificado_desarrollo_v1(jsonb,timestamptz)') IS NOT NULL
    OR politica IS NULL
    OR NOT has_schema_privilege(current_user,'vec_identidad_sesiones_v1','USAGE')
    OR NOT has_function_privilege(current_user,politica,'EXECUTE')
    OR NOT EXISTS (SELECT 1 FROM pg_proc WHERE oid=politica
      AND proowner='vec_identidad_sesiones_v1_propietario'::regrole AND prosecdef
      AND prorettype='boolean'::regtype AND provolatile='v')
    OR NOT EXISTS (SELECT 1 FROM pg_proc WHERE oid=f AND proowner=current_user::regrole
      AND prosecdef AND provolatile='v' AND prorettype='jsonb'::regtype
      AND octet_length(prosrc)=18515
      AND encode(sha256(convert_to(prosrc,'UTF8')),'hex')='38e12724f1addf6dbec363394f941641369bfb4dd8906eeb3f469eb243266073'
      AND obj_description(oid,'pg_proc')='Personal000005:lector_incorporacion:v2:ambitos_org_unidad') THEN
   RAISE EXCEPTION 'Personal 000010a: lector previo incompatible' USING ERRCODE='55000';
 END IF;
END $pre$;

-- Identidad común conserva la única fila de política y su revocación. Personal
-- consulta exclusivamente la fachada nominal de esa autoridad.
CREATE FUNCTION vec_personal.admite_lectura_incorporacion_certificado_desarrollo_v1(vinculo jsonb, instante timestamptz)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY INVOKER SET search_path=pg_catalog
AS $f$
BEGIN
 IF current_user<>'vec_personal_propietario' OR vinculo IS NULL OR instante IS NULL OR NOT isfinite(instante)
    OR vinculo->>'garantia_observada' IS DISTINCT FROM 'sustancial'
    OR vinculo->>'metodo_observado' IS DISTINCT FROM 'certificado'
    OR vinculo->>'superficie' IS DISTINCT FROM 'interna_corporativa'
    OR vinculo->>'cuenta_privilegiada' IS DISTINCT FROM 'false' THEN
   RETURN false;
 END IF;
 RETURN vec_identidad_sesiones_v1.admite_politica_certificado_personal_desarrollo_v1(
    vinculo->>'politica_garantia_ref',vinculo->>'politica_garantia_huella_sha256',instante);
END $f$;
REVOKE ALL ON FUNCTION vec_personal.admite_lectura_incorporacion_certificado_desarrollo_v1(jsonb,timestamptz) FROM PUBLIC;
COMMENT ON FUNCTION vec_personal.admite_lectura_incorporacion_certificado_desarrollo_v1(jsonb,timestamptz)
 IS 'Personal000010a: excepción temporal de consulta V2; no autoriza alta, efectos ni lectura V1';

DO $parche$
DECLARE f oid:='vec_personal.acreditar_alta_ejercicio_v2(text,text,text,bigint,text,text,text,text,text,text,bytea,bytea,bytea,bytea,bigint,bigint,bytea,bytea,bytea,bytea)'::regprocedure;
  original text; nuevo text; meta jsonb;
  antes text:=$x$d#>>'{vinculo_autenticacion_actor,garantia_observada}' IS DISTINCT FROM 'alto'$x$;
  despues text:=$x$(d#>>'{vinculo_autenticacion_actor,garantia_observada}' IS DISTINCT FROM 'alto'
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
 SELECT pg_get_functiondef(oid),to_jsonb(p)-'prosrc' INTO STRICT original,meta FROM pg_proc p WHERE oid=f;
 IF length(original)-length(replace(original,antes,''))<>length(antes)
    OR length(original)-length(replace(original,retorno,''))<>length(retorno) THEN
   RAISE EXCEPTION 'Personal 000010a: guarda previa incompatible' USING ERRCODE='55000';
 END IF;
 nuevo:=replace(replace(original,antes,despues),retorno,retorno_nuevo);
 EXECUTE nuevo;
 IF (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE oid=f) IS DISTINCT FROM meta
    OR pg_get_functiondef(f) IS DISTINCT FROM nuevo THEN
   RAISE EXCEPTION 'Personal 000010a: definición o ACL alterada' USING ERRCODE='55000';
 END IF;
END $parche$;
COMMENT ON FUNCTION vec_personal.acreditar_alta_ejercicio_v2(text,text,text,bigint,text,text,text,text,text,text,bytea,bytea,bytea,bytea,bigint,bigint,bytea,bytea,bytea,bytea)
 IS 'Personal000010a:lector_incorporacion:v2:certificado_desarrollo_gobernado';
COMMIT;
