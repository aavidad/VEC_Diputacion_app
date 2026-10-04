\set ON_ERROR_STOP on
-- CA36: contexto ADMIN con runtime segregado. Borrador cerrado hasta IS16/AD192.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec:admin:continuidad:v1',0));
DO $pre$
DECLARE r record;
BEGIN
 IF true THEN RAISE EXCEPTION 'CA36: borrador dependiente de IS16/AD192 y dos revisiones' USING ERRCODE='55000'; END IF;
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 OR to_regrole('vec_contexto_actor_v1_admin_contexto') IS NOT NULL
 OR to_regprocedure('vec_contexto_actor_v1.resolver_contexto_ca36_interno_v1(text,text,text,text,text,text,timestamptz,text[])') IS NOT NULL
 OR to_regprocedure('vec_contexto_actor_v1.reconciliar_contexto_ca36_interno_v1(text,text,text,text,text,text,timestamptz,text[])') IS NOT NULL
 THEN RAISE EXCEPTION 'CA36: preimagen incompatible' USING ERRCODE='55000'; END IF;
 FOR r IN SELECT p.oid,p.proowner,p.prosecdef,p.proconfig,
    ARRAY(SELECT a.acl::text FROM unnest(p.proacl) AS a(acl) ORDER BY a.acl::text) AS acl_texto,
    encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') AS fuente_sha,
    encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex') AS definicion_sha
   FROM pg_proc p WHERE p.oid IN(
    to_regprocedure('vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2(text,text,text,text,text,text,timestamptz,text[])'),
    to_regprocedure('vec_contexto_actor_v1.reconciliar_contexto_actor_v2(text,text,text,text,text,text,timestamptz,text[])')) LOOP
  IF r.proowner IS DISTINCT FROM to_regrole('vec_contexto_actor_v1_propietario')
   OR r.prosecdef IS NOT TRUE OR r.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog']::text[]
   OR r.acl_texto IS DISTINCT FROM ARRAY[
     'vec_contexto_actor_v1_propietario=X/vec_contexto_actor_v1_propietario',
     'vec_contexto_actor_v1_runtime=X/vec_contexto_actor_v1_propietario']::text[]
   OR (r.oid=to_regprocedure('vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2(text,text,text,text,text,text,timestamptz,text[])')
      AND (r.fuente_sha IS DISTINCT FROM '903fa70a63e86ccb3cf71090b87e948288e8dde1958652ebf032f7050634496c'
       OR r.definicion_sha IS DISTINCT FROM '7ef8569f502fbd5b8e5562c83b3720f7385e972fa9178b84483759c807d8f88c'))
   OR (r.oid=to_regprocedure('vec_contexto_actor_v1.reconciliar_contexto_actor_v2(text,text,text,text,text,text,timestamptz,text[])')
      AND (r.fuente_sha IS DISTINCT FROM 'cff0c40236bb2ba32dac5cbf422e41a0b35e5184526652adf76ee86b5e136fb1'
       OR r.definicion_sha IS DISTINCT FROM 'c0f78c8896672dd167cbe509d80bbe3c38941f38a4b0d958b90c201f059db14f'))
  THEN RAISE EXCEPTION 'CA36: núcleo V2 divergente' USING ERRCODE='55000'; END IF;
 END LOOP;
 IF NOT FOUND THEN RAISE EXCEPTION 'CA36: núcleo V2 ausente' USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
-- Extrae exactamente el cuerpo medido; las entradas generales retienen el
-- chequeo de su runtime. Los helpers nuevos sólo pertenecen al owner CA.
DO $extraer$
DECLARE original oid;definicion text;cabecera text;nombre text;marca text;
BEGIN
 FOREACH original IN ARRAY ARRAY[
  to_regprocedure('vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2(text,text,text,text,text,text,timestamptz,text[])'),
  to_regprocedure('vec_contexto_actor_v1.reconciliar_contexto_actor_v2(text,text,text,text,text,text,timestamptz,text[])')
 ] LOOP
  definicion:=pg_get_functiondef(original);
  IF original=to_regprocedure('vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2(text,text,text,text,text,text,timestamptz,text[])') THEN
   cabecera:='CREATE OR REPLACE FUNCTION vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2(';
   nombre:='CREATE FUNCTION vec_contexto_actor_v1.resolver_contexto_ca36_interno_v1(';
  ELSE
   cabecera:='CREATE OR REPLACE FUNCTION vec_contexto_actor_v1.reconciliar_contexto_actor_v2(';
   nombre:='CREATE FUNCTION vec_contexto_actor_v1.reconciliar_contexto_ca36_interno_v1(';
  END IF;
  marca:='    PERFORM vec_contexto_actor_v1.exigir_runtime_contexto_actor_v1();';
  IF left(definicion,length(cabecera)) IS DISTINCT FROM cabecera
   OR (length(definicion)-length(replace(definicion,marca,''))) IS DISTINCT FROM length(marca)
  THEN RAISE EXCEPTION 'CA36: extracción privada no exacta' USING ERRCODE='55000'; END IF;
  EXECUTE replace(replace(definicion,cabecera,nombre),marca,'');
 END LOOP;
END $extraer$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.resolver_contexto_ca36_interno_v1(text,text,text,text,text,text,timestamptz,text[]),
 vec_contexto_actor_v1.reconciliar_contexto_ca36_interno_v1(text,text,text,text,text,text,timestamptz,text[]) FROM PUBLIC;
CREATE OR REPLACE FUNCTION vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2(
 p_operacion_ref text,p_registro_contexto_ref text,p_cuenta_ref text,p_perfil_ref text,
 p_metodo text,p_garantia text,p_solicitado_en timestamptz,p_proyecciones text[])
RETURNS TABLE(operacion_ref text,registro_contexto_ref text,representacion_canonica bytea,
 huella_sha256 text,manifiesto_procedencia_canonico bytea,
 manifiesto_procedencia_huella_sha256 text,autoridad_efectiva text,resuelto_en timestamptz)
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
BEGIN
 PERFORM vec_contexto_actor_v1.exigir_runtime_contexto_actor_v1();
 RETURN QUERY SELECT * FROM vec_contexto_actor_v1.resolver_contexto_ca36_interno_v1(
  p_operacion_ref,p_registro_contexto_ref,p_cuenta_ref,p_perfil_ref,p_metodo,p_garantia,p_solicitado_en,p_proyecciones);
END $f$;
CREATE OR REPLACE FUNCTION vec_contexto_actor_v1.reconciliar_contexto_actor_v2(
 p_operacion_ref text,p_registro_contexto_ref text,p_cuenta_ref text,p_perfil_ref text,
 p_metodo text,p_garantia text,p_solicitado_en timestamptz,p_proyecciones text[])
RETURNS TABLE(operacion_ref text,registro_contexto_ref text,representacion_canonica bytea,
 huella_sha256 text,manifiesto_procedencia_canonico bytea,
 manifiesto_procedencia_huella_sha256 text,autoridad_efectiva text,resuelto_en timestamptz)
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
BEGIN
 PERFORM vec_contexto_actor_v1.exigir_runtime_contexto_actor_v1();
 RETURN QUERY SELECT * FROM vec_contexto_actor_v1.reconciliar_contexto_ca36_interno_v1(
  p_operacion_ref,p_registro_contexto_ref,p_cuenta_ref,p_perfil_ref,p_metodo,p_garantia,p_solicitado_en,p_proyecciones);
END $f$;
-- Las fachadas ADMIN, enlace durable y ACL exclusivas se añaden tras fijar
-- IS16 y la consulta AD192; el bloqueo $pre$ impide instalar este borrador.
COMMIT;
