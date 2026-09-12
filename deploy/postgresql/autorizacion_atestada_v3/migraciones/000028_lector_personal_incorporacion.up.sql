\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
-- Orden compartido con Personal-000003 y CT-000070: Personal, CT, local.
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal.dependencias.alta_ejercicio.v1',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal.dependencias.incorporacion_ejercicio.v2',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000028',0));

DO $precondicion$
BEGIN
 IF EXISTS (SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname='vec_autorizacion_atestada_v3' AND p.proname='registrar_y_consumir_lectura_personal_incorporacion_v3_atestada')
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_incorporacion_ejercicio_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_personal_propietario' AND NOT rolcanlogin AND NOT rolsuper AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication AND NOT rolbypassrls)
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_contratacion_temporal_propietario' AND NOT rolcanlogin AND NOT rolsuper AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication AND NOT rolbypassrls) THEN
  RAISE EXCEPTION 'estado incompatible para AD3-28' USING ERRCODE='55000';
 END IF;
END $precondicion$;

DO $nucleo$
DECLARE v_def text; v_acl aclitem[]; v_owner oid; v_config text[];
 v_marca text := E'       )\n       OR c ->> ''suite'' <> ''VEC-AD-3-COSE-EDDSA-1''';
 v_extension text := $perfil$           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'lectura_registro_personal_incorporacion'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.lectura_incorporacion.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM 'personal.alta_ejercicio.registro.consultar_incorporacion'
               AND c ->> 'operacion' IS NOT DISTINCT FROM d ->> 'accion'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'personal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'registro_alta_ejercicio_incorporacion'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'preparar_confirmacion_incorporacion_ct'
           )
$perfil$;
 v_runtime_anterior text := $anterior$       OR NOT (
           (
               p_perfil_mutacion IS DISTINCT FROM 'bolsa_llamamiento'
               AND p_perfil_mutacion IS DISTINCT FROM 'alta_personal_ejercicio'
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_personal_ejecutor', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_personal_propietario', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_personal_migrador', 'MEMBER')
               AND pg_catalog.pg_has_role(
                   session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_contratacion_temporal_migrador', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_contratacion_temporal_propietario', 'MEMBER')
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'bolsa_llamamiento'
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_personal_ejecutor', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_personal_propietario', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_personal_migrador', 'MEMBER')
               AND pg_catalog.pg_has_role(
                   session_user, 'vec_bolsa_llamamientos_ejecutor', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_bolsa_llamamientos_propietario', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_bolsa_llamamientos_migrador', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_contratacion_temporal_propietario', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_contratacion_temporal_migrador', 'MEMBER')
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'alta_personal_ejercicio'
               AND pg_catalog.pg_has_role(
                   session_user, 'vec_personal_ejecutor', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_personal_propietario', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_personal_migrador', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_contratacion_temporal_propietario', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_contratacion_temporal_migrador', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_bolsa_llamamientos_ejecutor', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_bolsa_llamamientos_propietario', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_bolsa_llamamientos_migrador', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_autorizacion_atestada_v3_propietario', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_autorizacion_atestada_v3_migrador', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_autorizacion_atestada_v3_emisor', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_autorizacion_atestada_v3_consumidor', 'MEMBER')
           )
       ) THEN$anterior$;
 v_runtime_nuevo text := replace(v_runtime_anterior,
   'p_perfil_mutacion IS DISTINCT FROM ''alta_personal_ejercicio''',
   'p_perfil_mutacion IS DISTINCT FROM ''alta_personal_ejercicio'' AND p_perfil_mutacion IS DISTINCT FROM ''lectura_registro_personal_incorporacion''');
 v_lector text := $lector$           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'lectura_registro_personal_incorporacion'
               AND (
                 (
                   pg_catalog.pg_has_role(session_user,'vec_personal_ejecutor','MEMBER')
                   AND NOT pg_catalog.pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
                   AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole AND m.roleid='vec_personal_ejecutor'::regrole AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option)
                   AND NOT EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member='vec_personal_ejecutor'::regrole)
                 )
                 OR (
                   pg_catalog.pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
                   AND NOT pg_catalog.pg_has_role(session_user,'vec_personal_ejecutor','MEMBER')
                   AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole AND m.roleid='vec_contratacion_temporal_ejecutor'::regrole AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option)
                   AND NOT EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member='vec_contratacion_temporal_ejecutor'::regrole)
                 )
               )
               AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=1
               AND NOT pg_catalog.pg_has_role(session_user,'vec_personal_propietario','MEMBER')
               AND NOT pg_catalog.pg_has_role(session_user,'vec_personal_migrador','MEMBER')
               AND NOT pg_catalog.pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
               AND NOT pg_catalog.pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
               AND NOT pg_catalog.pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER')
               AND NOT pg_catalog.pg_has_role(session_user,'vec_bolsa_llamamientos_propietario','MEMBER')
               AND NOT pg_catalog.pg_has_role(session_user,'vec_bolsa_llamamientos_migrador','MEMBER')
               AND NOT pg_catalog.pg_has_role(session_user,'vec_autorizacion_atestada_v3_propietario','MEMBER')
               AND NOT pg_catalog.pg_has_role(session_user,'vec_autorizacion_atestada_v3_migrador','MEMBER')
               AND NOT pg_catalog.pg_has_role(session_user,'vec_autorizacion_atestada_v3_emisor','MEMBER')
               AND NOT pg_catalog.pg_has_role(session_user,'vec_autorizacion_atestada_v3_consumidor','MEMBER')
               AND NOT EXISTS (
                 SELECT 1 FROM pg_roles r
                 WHERE left(r.rolname,4)='vec_' AND r.rolname<>session_user
                   AND r.rolname NOT IN ('vec_personal_ejecutor','vec_contratacion_temporal_ejecutor')
                   AND pg_catalog.pg_has_role(session_user,r.oid,'MEMBER')
               )
           )
$lector$;
BEGIN
 SELECT pg_get_functiondef(p.oid),p.proacl,p.proowner,p.proconfig INTO STRICT v_def,v_acl,v_owner,v_config FROM pg_proc p WHERE p.oid='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef;
 IF length(v_def)-length(replace(v_def,v_runtime_anterior,''))<>length(v_runtime_anterior) OR length(v_def)-length(replace(v_def,v_marca,''))<>length(v_marca) OR strpos(v_def,'lectura_registro_personal_incorporacion')<>0 OR strpos(v_def,'incorporacion_ejercicio_ct')=0 OR strpos(v_def,'alta_personal_ejercicio')=0 THEN RAISE EXCEPTION 'núcleo incompatible para AD3-28' USING ERRCODE='55000'; END IF;
 v_runtime_nuevo := replace(v_runtime_nuevo,'       ) THEN','           '||v_lector||'       ) THEN');
 EXECUTE replace(replace(v_def,v_runtime_anterior,v_runtime_nuevo),v_marca,v_extension||v_marca);
 IF EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure AND (p.proacl IS DISTINCT FROM v_acl OR p.proowner IS DISTINCT FROM v_owner OR p.proconfig IS DISTINCT FROM v_config OR NOT p.prosecdef)) THEN RAISE EXCEPTION 'AD3-28 alteró núcleo o autoridad ajena' USING ERRCODE='55000'; END IF;
END $nucleo$;

LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $audiencia$
DECLARE v_def text; v_a6 text[]:=ARRAY['vec_contratacion_temporal.confirmar_alta_atestada.v1','vec_bolsa_llamamientos.confirmar_integracion_desarrollo.v1','vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1','vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1','vec_personal.alta_ejercicio.v1','vec_contratacion_temporal.incorporacion_ejercicio.v2']; v_a10 text[]:=ARRAY['vec_contratacion_temporal.confirmar_alta_atestada.v1','vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1','vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1','vec_contexto_actor.publicar_organizacion_corporativa_fuente.v1','vec_contexto_actor.revocar_organizacion_corporativa_fuente.v1','vec_contexto_actor.publicar_vinculo_corporativo_fuente.v1','vec_contexto_actor.revocar_vinculo_corporativo_fuente.v1','vec_bolsa_llamamientos.confirmar_integracion_desarrollo.v1','vec_personal.alta_ejercicio.v1','vec_contratacion_temporal.incorporacion_ejercicio.v2']; v_a text[];
BEGIN
 SELECT regexp_replace(pg_get_constraintdef(c.oid,true),'\s+',' ','g') INTO STRICT v_def FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated AND c.conkey=ARRAY[8]::smallint[];
 IF v_def='CHECK (audiencia_consumo = ANY (ARRAY['||array_to_string(ARRAY(SELECT quote_literal(a)||'::text' FROM unnest(v_a6) WITH ORDINALITY u(a,n) ORDER BY n),', ')||']))' THEN v_a:=v_a6; ELSIF v_def='CHECK (audiencia_consumo = ANY (ARRAY['||array_to_string(ARRAY(SELECT quote_literal(a)||'::text' FROM unnest(v_a10) WITH ORDINALITY u(a,n) ORDER BY n),', ')||']))' THEN v_a:=v_a10; ELSE RAISE EXCEPTION 'audiencias incompatibles con AD3-28' USING ERRCODE='55000'; END IF;
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check CHECK (audiencia_consumo IN ('||array_to_string(ARRAY(SELECT quote_literal(a) FROM unnest(array_append(v_a,'vec_personal.lectura_incorporacion.v1')) WITH ORDINALITY u(a,n) ORDER BY n),', ')||'))';
END $audiencia$;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_lectura_personal_incorporacion_v3_atestada(p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE v record;
BEGIN
 SELECT * INTO STRICT v FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna('lectura_registro_personal_incorporacion',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF v.consumo_nuevo IS NOT TRUE THEN RAISE EXCEPTION 'lectura Personal requiere consumo nuevo' USING ERRCODE='P1102'; END IF;
 RETURN QUERY SELECT v.decision_ref,v.efecto_ref,v.huella_efecto_sha256,v.consumo_huella_sha256,v.auditoria_ref,v.consumida_en,true;
END $f$;
ALTER FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_lectura_personal_incorporacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) OWNER TO vec_autorizacion_atestada_v3_propietario;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_lectura_personal_incorporacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC,vec_autorizacion_atestada_v3_consumidor,vec_autorizacion_atestada_v3_emisor,vec_autorizacion_atestada_v3_migrador,vec_personal_ejecutor,vec_personal_migrador,vec_contratacion_temporal_ejecutor,vec_contratacion_temporal_migrador,vec_contratacion_temporal_propietario;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_lectura_personal_incorporacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_personal_propietario;
COMMIT;
