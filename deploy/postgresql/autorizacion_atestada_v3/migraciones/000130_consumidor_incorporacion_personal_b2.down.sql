\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog, pg_temp;
SET LOCAL lock_timeout='5s';SET LOCAL statement_timeout='2min';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000130',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
DO $pre$
BEGIN IF to_regclass('vec_contratacion_temporal.plan_incorporacion_personal_b2_v1') IS NOT NULL THEN RAISE EXCEPTION 'AD3-130: retirar CT155 antes de DOWN' USING ERRCODE='55000';END IF;END $pre$;
DROP FUNCTION vec_autorizacion_atestada_v3.consumir_incorporacion_personal_ct_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DO $nucleo$
DECLARE f oid:='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;old text;neu text;meta jsonb;
 extension text:=$x$           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'incorporacion_personal_ct'
 AND c->>'operacion' = ANY (ARRAY['contratacion_temporal.incorporacion_personal.plan.consultar','contratacion_temporal.incorporacion_personal.plan.registrar','contratacion_temporal.incorporacion_personal.origen.confirmar'])
 AND ((c->>'operacion'='contratacion_temporal.incorporacion_personal.plan.consultar'
       AND c->>'audiencia_consumo'='vec_contratacion_temporal.incorporacion_personal.plan.consultar.v1'
       AND d->'campos_permitidos'='["plan"]'::jsonb)
   OR (c->>'operacion'='contratacion_temporal.incorporacion_personal.plan.registrar'
       AND c->>'audiencia_consumo'='vec_contratacion_temporal.incorporacion_personal.plan.registrar.v1'
       AND d->'campos_permitidos'='["recibo"]'::jsonb)
   OR (c->>'operacion'='contratacion_temporal.incorporacion_personal.origen.confirmar' AND c->>'audiencia_consumo'='vec_contratacion_temporal.incorporacion_personal.origen.confirmar.v1' AND d->'campos_permitidos'='["recibo"]'::jsonb))
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'incorporacion_personal_ct'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'incorporar_personal_desde_ct'
 AND d->>'recurso_ref' IS NOT NULL AND d->>'recurso_ref' LIKE 'expediente:%'
 AND c->>'efecto_ref' IS NOT DISTINCT FROM d->>'recurso_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
$x$;
BEGIN
 SELECT pg_get_functiondef(oid),to_jsonb(p)-'prosrc' INTO STRICT old,meta FROM pg_proc p WHERE oid=f;
 IF length(old)-length(replace(old,extension,''))<>length(extension) THEN RAISE EXCEPTION 'AD3-130: preimagen incompatible' USING ERRCODE='55000';END IF;
 neu:=replace(old,extension,'');EXECUTE neu;
 IF (SELECT pg_get_functiondef(oid) FROM pg_proc WHERE oid=f) IS DISTINCT FROM neu OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE oid=f) IS DISTINCT FROM meta THEN RAISE EXCEPTION 'AD3-130: metadata alterada' USING ERRCODE='55000';END IF;
END $nucleo$;
LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $audiencias$
DECLARE d text;sufijo text:=', ''vec_contratacion_temporal.incorporacion_personal.plan.consultar.v1''::text, ''vec_contratacion_temporal.incorporacion_personal.plan.registrar.v1''::text, ''vec_contratacion_temporal.incorporacion_personal.origen.confirmar.v1''::text]))';
BEGIN
 SELECT regexp_replace(pg_get_constraintdef(c.oid,true),'\s+',' ','g') INTO STRICT d FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass AND c.conname='clave_capacidad_version_audiencia_consumo_check';
 IF right(d,length(sufijo))<>sufijo THEN RAISE EXCEPTION 'AD3-130: audiencia posterior' USING ERRCODE='55000';END IF;
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||left(d,length(d)-length(sufijo))||']))';
END $audiencias$;
COMMIT;
