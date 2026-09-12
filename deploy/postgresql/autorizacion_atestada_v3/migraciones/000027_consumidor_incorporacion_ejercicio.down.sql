\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
-- Mismo identificador primero que UP, CT000070 y futuro negocio CT.
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal.dependencias.incorporacion_ejercicio.v2',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000027',0));
LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_autorizacion_atestada_v3.atestacion_decision_v3 IN SHARE MODE;

DO $proteger$
BEGIN
 IF to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_incorporacion_ejercicio_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR EXISTS (SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname='vec_autorizacion_atestada_v3' AND p.proname='registrar_y_consumir_incorporacion_ejercicio_ct_v3_atestada' AND p.oid<>'vec_autorizacion_atestada_v3.registrar_y_consumir_incorporacion_ejercicio_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure)
    OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.clave_capacidad_version WHERE audiencia_consumo='vec_contratacion_temporal.incorporacion_ejercicio.v2')
    OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.atestacion_decision_v3 WHERE convert_from(capacidad_canonica,'UTF8')::jsonb->>'audiencia_consumo'='vec_contratacion_temporal.incorporacion_ejercicio.v2')
    OR EXISTS (SELECT 1 FROM pg_depend d WHERE d.refclassid='pg_proc'::regclass AND d.refobjid='vec_autorizacion_atestada_v3.registrar_y_consumir_incorporacion_ejercicio_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure AND d.deptype<>'i')
    OR EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid<>'vec_autorizacion_atestada_v3.registrar_y_consumir_incorporacion_ejercicio_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure AND p.prosrc LIKE '%registrar_y_consumir_incorporacion_ejercicio_ct_v3_atestada%') THEN
   RAISE EXCEPTION 'retirada AD3-27 denegada: función, sobrecarga, clave, historia o dependencia' USING ERRCODE='55000';
 END IF;
END $proteger$;

DO $retirar$
DECLARE v_def text; v_acl aclitem[]; v_owner oid; v_config text[];
 v_extension text := $perfil$           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'incorporacion_ejercicio_ct'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contratacion_temporal.incorporacion_ejercicio.v2'
               AND c ->> 'operacion' IS NOT DISTINCT FROM 'contratacion_temporal.incorporacion.confirmar'
               AND c ->> 'operacion' IS NOT DISTINCT FROM d ->> 'accion'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'confirmacion_incorporacion_ejercicio_v2'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'registrar_incorporacion_confirmada_por_personal'
           )
$perfil$;
BEGIN
 SELECT pg_get_functiondef(p.oid),p.proacl,p.proowner,p.proconfig INTO STRICT v_def,v_acl,v_owner,v_config FROM pg_proc p WHERE p.oid='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef;
 IF length(v_def)-length(replace(v_def,v_extension,''))<>length(v_extension) THEN RAISE EXCEPTION 'núcleo incompatible; no retirar perfiles ajenos' USING ERRCODE='55000'; END IF;
 EXECUTE replace(v_def,v_extension,'');
 IF EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure AND (p.proacl IS DISTINCT FROM v_acl OR p.proowner IS DISTINCT FROM v_owner OR p.proconfig IS DISTINCT FROM v_config OR NOT p.prosecdef)) THEN RAISE EXCEPTION 'retirada AD3-27 alteró núcleo o autoridad ajena' USING ERRCODE='55000'; END IF;
END $retirar$;

DO $audiencia$
DECLARE v_def text; v_a6 text[] := ARRAY['vec_contratacion_temporal.confirmar_alta_atestada.v1','vec_bolsa_llamamientos.confirmar_integracion_desarrollo.v1','vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1','vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1','vec_personal.alta_ejercicio.v1','vec_contratacion_temporal.incorporacion_ejercicio.v2']; v_a10 text[] := ARRAY['vec_contratacion_temporal.confirmar_alta_atestada.v1','vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1','vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1','vec_contexto_actor.publicar_organizacion_corporativa_fuente.v1','vec_contexto_actor.revocar_organizacion_corporativa_fuente.v1','vec_contexto_actor.publicar_vinculo_corporativo_fuente.v1','vec_contexto_actor.revocar_vinculo_corporativo_fuente.v1','vec_bolsa_llamamientos.confirmar_integracion_desarrollo.v1','vec_personal.alta_ejercicio.v1','vec_contratacion_temporal.incorporacion_ejercicio.v2']; v_a text[];
BEGIN
 SELECT regexp_replace(pg_get_constraintdef(c.oid,true),'\s+',' ','g') INTO STRICT v_def FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated AND c.conkey=ARRAY[8]::smallint[];
 IF v_def='CHECK (audiencia_consumo = ANY (ARRAY['||array_to_string(ARRAY(SELECT quote_literal(a)||'::text' FROM unnest(v_a6) WITH ORDINALITY u(a,n) ORDER BY n),', ')||']))' THEN v_a:=v_a6[1:5]; ELSIF v_def='CHECK (audiencia_consumo = ANY (ARRAY['||array_to_string(ARRAY(SELECT quote_literal(a)||'::text' FROM unnest(v_a10) WITH ORDINALITY u(a,n) ORDER BY n),', ')||']))' THEN v_a:=v_a10[1:9]; ELSE RAISE EXCEPTION 'audiencias incompatibles; no retirar AD3-27' USING ERRCODE='55000'; END IF;
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check CHECK (audiencia_consumo IN ('||array_to_string(ARRAY(SELECT quote_literal(a) FROM unnest(v_a) WITH ORDINALITY u(a,n) ORDER BY n),', ')||'))';
END $audiencia$;
DROP FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_incorporacion_ejercicio_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
COMMIT;
