\set ON_ERROR_STOP on
-- Preimagen: AD277 de O, SHA-256 del fichero
-- e930828a6a193dfcea2a058d65997326282385498fd810fb20cb0a502535cb5d.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000278',0));
DO $pre$
DECLARE f oid;h text;p pg_proc%ROWTYPE;
BEGIN
 f:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF f IS NOT NULL THEN
  SELECT * INTO p FROM pg_proc WHERE oid=f;
  h:=encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex');
 END IF;
 IF current_user<>'vec_autorizacion_atestada_v3_propietario'
    OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
    OR f IS NULL
    OR h IS DISTINCT FROM '7d13731f452b0f8d7020ea0ab602c465897617fce61a84a7887491309e1bbb48'
    OR p.proowner IS DISTINCT FROM 'vec_autorizacion_atestada_v3_propietario'::regrole
    OR p.proacl::text IS DISTINCT FROM '{vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario}'
    OR NOT p.prosecdef OR p.provolatile<>'v'
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_vinculo_oferta_bolsa_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR to_regclass('vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1') IS NULL
 THEN RAISE EXCEPTION 'AD278: preimagen AD277 incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

-- La acción de oferta tiene su propia audiencia. Mantiene intacto el perfil
-- singular CT201 y requiere una capacidad gobernada para la acción nueva.
DO $patch$
DECLARE f oid:='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 definicion text;marca text;nueva text;
BEGIN
 definicion:=pg_get_functiondef(f);
 marca:=$m$           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'vinculo_emision_bolsa_ct'$m$;
 nueva:=$m$           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'vinculo_oferta_bolsa_ct'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM
                   'vec_contratacion_temporal.vinculo_oferta_bolsa.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM
                   'contratacion_temporal.bolsa.oferta.vincular'
               AND d ->> 'accion' IS NOT DISTINCT FROM c ->> 'operacion'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'vinculo_oferta_bolsa_expediente'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'tramitacion_expediente_contratacion_temporal'
               AND d ->> 'recurso_ref' IS NOT DISTINCT FROM c ->> 'efecto_ref'
               AND d ->> 'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c ->> 'huella_efecto_sha256'
               AND d -> 'campos_permitidos' IS NOT DISTINCT FROM '[]'::jsonb
               AND d -> 'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb
           )
$m$||marca;
 IF length(definicion)-length(replace(definicion,marca,''))<>length(marca)
 THEN RAISE EXCEPTION 'AD278: marca de CT201 no singular' USING ERRCODE='55000'; END IF;
 EXECUTE replace(definicion,marca,nueva);
END $patch$;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_vinculo_oferta_bolsa_ct_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,
 consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET lock_timeout='2s' AS $f$
DECLARE v record;
BEGIN
 SELECT * INTO STRICT v FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
   'vinculo_oferta_bolsa_ct',p_capacidad,p_decision,p_motivo,p_contexto,
   p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF v.consumo_nuevo IS NOT TRUE THEN
  RAISE EXCEPTION 'AD278: consumo previo rechazado' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT v.decision_ref,v.efecto_ref,v.huella_efecto_sha256,
  v.consumo_huella_sha256,v.auditoria_ref,v.consumida_en,true;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_vinculo_oferta_bolsa_ct_v3_atestada(
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_vinculo_oferta_bolsa_ct_v3_atestada(
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_contratacion_temporal_propietario;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_contratacion_temporal_propietario;

-- Misma cuenta nominal CT que CT201, operación y audiencia distintas. El
-- registro de origen no concede el nuevo permiso ni crea una capacidad.
DO $origen$
DECLARE anterior vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1%ROWTYPE;
BEGIN
 SELECT * INTO STRICT anterior FROM vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1
 WHERE audiencia_consumo='vec_contratacion_temporal.vinculo_emision_bolsa.v1'
   AND operacion='contratacion_temporal.bolsa.vincular'
   AND canal_permitido='interna_corporativa';
 IF EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1
   WHERE audiencia_consumo='vec_contratacion_temporal.vinculo_oferta_bolsa.v1'
     OR operacion='contratacion_temporal.bolsa.oferta.vincular')
    OR NOT EXISTS(SELECT 1 FROM pg_roles r WHERE r.rolname=anterior.login_nombre
      AND r.rolcanlogin AND r.rolinherit AND NOT(r.rolsuper OR r.rolcreaterole OR r.rolcreatedb OR r.rolreplication OR r.rolbypassrls)
      AND pg_has_role(r.oid,'vec_contratacion_temporal_ejecutor'::regrole,'MEMBER'))
 THEN RAISE EXCEPTION 'AD278: origen CT nominal incompatible' USING ERRCODE='55000'; END IF;
 INSERT INTO vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1(
  login_nombre,audiencia_consumo,operacion,proceso,canal_permitido,configurada_en)
 VALUES(anterior.login_nombre,'vec_contratacion_temporal.vinculo_oferta_bolsa.v1',
  'contratacion_temporal.bolsa.oferta.vincular',anterior.proceso,anterior.canal_permitido,clock_timestamp());
END $origen$;

DO $audiencia$
DECLARE anterior text;ampliada text;interior text;
BEGIN
 SELECT pg_get_constraintdef(c.oid,false) INTO STRICT anterior FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
   AND c.conname='clave_capacidad_version_audiencia_consumo_check'
   AND c.contype='c' AND c.convalidated;
 IF left(anterior,7)<>'CHECK (' OR right(anterior,1)<>')'
    OR position('vec_contratacion_temporal.vinculo_emision_bolsa.v1' in anterior)=0
    OR position('vec_documentos.bolsa.justificante_renuncia.subida.v1' in anterior)=0
    OR position('vec_contratacion_temporal.vinculo_oferta_bolsa.v1' in anterior)>0
 THEN RAISE EXCEPTION 'AD278: audiencia AD277 incompatible' USING ERRCODE='55000'; END IF;
 interior:=substr(anterior,8,length(anterior)-8);
 ampliada:='CHECK (('||interior||') OR audiencia_consumo = ''vec_contratacion_temporal.vinculo_oferta_bolsa.v1'')';
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version
  DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||ampliada;
END $audiencia$;

DO $post$
DECLARE f oid; definicion text; p pg_proc%ROWTYPE;
BEGIN
 f:='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=f;
 definicion:=pg_get_functiondef(f);
 IF p.proowner IS DISTINCT FROM 'vec_autorizacion_atestada_v3_propietario'::regrole
    OR p.proacl::text IS DISTINCT FROM '{vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario}'
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
    OR NOT p.prosecdef OR p.provolatile<>'v'
    OR (length(definicion)-length(replace(definicion,
      'p_perfil_mutacion IS NOT DISTINCT FROM ''vinculo_oferta_bolsa_ct''','')))<>
      length('p_perfil_mutacion IS NOT DISTINCT FROM ''vinculo_oferta_bolsa_ct''')
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_vinculo_emision_bolsa_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_vinculo_oferta_bolsa_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 THEN RAISE EXCEPTION 'AD278: postimagen central incompatible' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
