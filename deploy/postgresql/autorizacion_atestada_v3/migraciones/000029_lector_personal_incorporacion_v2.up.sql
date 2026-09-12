\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal.dependencias.alta_ejercicio.v1',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal.dependencias.incorporacion_ejercicio.v2',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000029',0));
DO $precondicion$
BEGIN
 IF EXISTS (SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
     WHERE n.nspname='vec_autorizacion_atestada_v3' AND p.proname='registrar_y_consumir_lectura_personal_incorporacion_v2_atestada')
 OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_lectura_personal_incorporacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_personal_propietario' AND NOT rolcanlogin AND NOT rolsuper AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication AND NOT rolbypassrls) THEN
   RAISE EXCEPTION 'estado incompatible para AD3-29' USING ERRCODE='55000';
 END IF;
END $precondicion$;
-- Cambio reversible de tres fragmentos; no se reescriben ramas V1 ni validaciones comunes.
DO $nucleo$
DECLARE
 v_oid oid := 'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 v_def text; v_nueva text; v_meta jsonb; v_v1 jsonb; v_par text;
 v_perfil1 text := $perfil1$           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'lectura_registro_personal_incorporacion'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.lectura_incorporacion.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM 'personal.alta_ejercicio.registro.consultar_incorporacion'
               AND c ->> 'operacion' IS NOT DISTINCT FROM d ->> 'accion'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'personal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'registro_alta_ejercicio_incorporacion'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'preparar_confirmacion_incorporacion_ct'
           )
$perfil1$;
 v_perfil2 text := $perfil2$           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'lectura_registro_personal_incorporacion_v2'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.lectura_incorporacion.v2'
               AND c ->> 'operacion' IS NOT DISTINCT FROM 'personal.alta_ejercicio.registro.consultar_incorporacion'
               AND c ->> 'operacion' IS NOT DISTINCT FROM d ->> 'accion'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'personal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'registro_alta_ejercicio_incorporacion_v2'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'preparar_confirmacion_incorporacion_ct'
           )
$perfil2$;
 v_lector1 text := $lector1$           OR (
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
$lector1$;
 v_lector2 text := $lector2$           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'lectura_registro_personal_incorporacion_v2'
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
$lector2$;
 v_exclusion1 text := $exclusion1$p_perfil_mutacion IS DISTINCT FROM 'alta_personal_ejercicio' AND p_perfil_mutacion IS DISTINCT FROM 'lectura_registro_personal_incorporacion'$exclusion1$;
 v_exclusion2 text := $exclusion2$p_perfil_mutacion IS DISTINCT FROM 'alta_personal_ejercicio' AND p_perfil_mutacion IS DISTINCT FROM 'lectura_registro_personal_incorporacion' AND p_perfil_mutacion IS DISTINCT FROM 'lectura_registro_personal_incorporacion_v2'$exclusion2$;
BEGIN
 SELECT pg_get_functiondef(p.oid),to_jsonb(p)-'prosrc' INTO STRICT v_def,v_meta
 FROM pg_proc p WHERE p.oid=v_oid AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef;
 SELECT to_jsonb(p) INTO STRICT v_v1 FROM pg_proc p
 WHERE p.oid='vec_autorizacion_atestada_v3.registrar_y_consumir_lectura_personal_incorporacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
   AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef;
 FOREACH v_par IN ARRAY ARRAY[v_perfil1,v_lector1,v_exclusion1] LOOP
   IF length(v_def)-length(replace(v_def,v_par,''))<>length(v_par) THEN
     RAISE EXCEPTION 'núcleo incompatible para AD3-29' USING ERRCODE='55000';
   END IF;
 END LOOP;
 IF strpos(v_def,'lectura_registro_personal_incorporacion_v2')<>0 THEN RAISE EXCEPTION 'AD3-29 ya presente o parcial' USING ERRCODE='55000'; END IF;
 v_nueva := replace(replace(replace(v_def,v_perfil1,v_perfil1||v_perfil2),v_lector1,v_lector1||v_lector2),v_exclusion1,v_exclusion2);
 EXECUTE v_nueva;
 IF (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=v_oid) IS DISTINCT FROM v_meta
    OR (SELECT pg_get_functiondef(v_oid)) IS DISTINCT FROM v_nueva
    OR (SELECT to_jsonb(p) FROM pg_proc p WHERE p.oid='vec_autorizacion_atestada_v3.registrar_y_consumir_lectura_personal_incorporacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure) IS DISTINCT FROM v_v1 THEN
   RAISE EXCEPTION 'AD3-29 alteró metadatos o lector V1' USING ERRCODE='55000';
 END IF;
END $nucleo$;
LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $audiencia$
DECLARE v_def text; v_a7 text[]:=ARRAY['vec_contratacion_temporal.confirmar_alta_atestada.v1','vec_bolsa_llamamientos.confirmar_integracion_desarrollo.v1','vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1','vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1','vec_personal.alta_ejercicio.v1','vec_contratacion_temporal.incorporacion_ejercicio.v2','vec_personal.lectura_incorporacion.v1']; v_a11 text[]:=ARRAY['vec_contratacion_temporal.confirmar_alta_atestada.v1','vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1','vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1','vec_contexto_actor.publicar_organizacion_corporativa_fuente.v1','vec_contexto_actor.revocar_organizacion_corporativa_fuente.v1','vec_contexto_actor.publicar_vinculo_corporativo_fuente.v1','vec_contexto_actor.revocar_vinculo_corporativo_fuente.v1','vec_bolsa_llamamientos.confirmar_integracion_desarrollo.v1','vec_personal.alta_ejercicio.v1','vec_contratacion_temporal.incorporacion_ejercicio.v2','vec_personal.lectura_incorporacion.v1']; v_a text[];
BEGIN
 SELECT regexp_replace(pg_get_constraintdef(c.oid,true),'\s+',' ','g') INTO STRICT v_def FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated AND c.conkey=ARRAY[8]::smallint[];
 IF v_def='CHECK (audiencia_consumo = ANY (ARRAY['||array_to_string(ARRAY(SELECT quote_literal(a)||'::text' FROM unnest(v_a7) WITH ORDINALITY u(a,n) ORDER BY n),', ')||']))' THEN v_a:=v_a7; ELSIF v_def='CHECK (audiencia_consumo = ANY (ARRAY['||array_to_string(ARRAY(SELECT quote_literal(a)||'::text' FROM unnest(v_a11) WITH ORDINALITY u(a,n) ORDER BY n),', ')||']))' THEN v_a:=v_a11; ELSE RAISE EXCEPTION 'audiencias incompatibles; instalar AD3-29' USING ERRCODE='55000'; END IF;
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check CHECK (audiencia_consumo IN ('||array_to_string(ARRAY(SELECT quote_literal(a) FROM unnest(array_append(v_a,'vec_personal.lectura_incorporacion.v2')) WITH ORDINALITY u(a,n) ORDER BY n),', ')||'))';
END $audiencia$;
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_lectura_personal_incorporacion_v2_atestada(p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE v record;
BEGIN
 SELECT * INTO STRICT v FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna('lectura_registro_personal_incorporacion_v2',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF v.consumo_nuevo IS NOT TRUE THEN RAISE EXCEPTION 'lectura Personal requiere consumo nuevo' USING ERRCODE='P1102'; END IF;
 RETURN QUERY SELECT v.decision_ref,v.efecto_ref,v.huella_efecto_sha256,v.consumo_huella_sha256,v.auditoria_ref,v.consumida_en,true;
END $f$;
DO $acl$
DECLARE v_oid oid := 'vec_autorizacion_atestada_v3.registrar_y_consumir_lectura_personal_incorporacion_v2_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure; v_grantee oid;
BEGIN
 FOR v_grantee IN SELECT DISTINCT a.grantee FROM pg_proc p
   CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
   WHERE p.oid=v_oid AND a.grantee<>p.proowner LOOP
   EXECUTE 'REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_lectura_personal_incorporacion_v2_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM '||
     CASE WHEN v_grantee=0 THEN 'PUBLIC' ELSE quote_ident(pg_get_userbyid(v_grantee)) END;
 END LOOP;
 GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_lectura_personal_incorporacion_v2_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_personal_propietario;
 IF EXISTS (
   SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
   WHERE p.oid=v_oid AND (a.grantee NOT IN (p.proowner,'vec_personal_propietario'::regrole)
      OR a.grantor<>p.proowner OR a.privilege_type<>'EXECUTE' OR a.is_grantable)
 ) OR NOT EXISTS (
   SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(p.proacl) a
   WHERE p.oid=v_oid AND a.grantee='vec_personal_propietario'::regrole AND a.privilege_type='EXECUTE' AND NOT a.is_grantable
 ) THEN RAISE EXCEPTION 'ACL incompatible para lector V2' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
