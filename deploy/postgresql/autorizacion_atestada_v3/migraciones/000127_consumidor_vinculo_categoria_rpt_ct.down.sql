\set ON_ERROR_STOP on
-- Inversión posible solo antes de CT-154 y de cualquier acto confirmado.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog, pg_temp;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000127',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
DO $pre$
BEGIN
 IF EXISTS(SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace
           WHERE n.nspname='vec_contratacion_temporal' AND c.relname='vinculo_categoria_rpt_ct_v1')
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_vinculo_categoria_rpt_ct_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 THEN RAISE EXCEPTION 'AD3-127: DOWN requiere retirar CT-154 sin historia' USING ERRCODE='55000'; END IF;
END $pre$;
REVOKE EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.leer_publicacion_categoria_rpt_v3_atestada(
 jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 FROM vec_contratacion_temporal_propietario;
DROP FUNCTION vec_autorizacion_atestada_v3.consumir_vinculo_categoria_rpt_ct_v3_atestada(
 text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DO $nucleo$
DECLARE
 f oid:='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 original text; nuevo text;
 extension text:=$x$           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'vinculo_categoria_rpt_ct'
 AND c->>'operacion' = ANY (ARRAY['contratacion_temporal.categoria_rpt.vinculo.consultar','contratacion_temporal.categoria_rpt.vinculo.registrar'])
 AND ((c->>'operacion'='contratacion_temporal.categoria_rpt.vinculo.consultar'
       AND c->>'audiencia_consumo'='vec_contratacion_temporal.categoria_rpt.vinculo.consultar.v1'
       AND d->'campos_permitidos'='["analisis","vinculo"]'::jsonb)
   OR (c->>'operacion'='contratacion_temporal.categoria_rpt.vinculo.registrar'
       AND c->>'audiencia_consumo'='vec_contratacion_temporal.categoria_rpt.vinculo.registrar.v1'
       AND d->'campos_permitidos'='["recibo"]'::jsonb))
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'vinculo_categoria_rpt_ct'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestionar_vinculo_categoria_rpt_ct'
 AND d->>'recurso_ref' IS NOT NULL AND d->>'recurso_ref' LIKE 'expediente:%'
 AND c->>'efecto_ref' IS NOT DISTINCT FROM d->>'recurso_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
$x$;
BEGIN
 SELECT pg_get_functiondef(f) INTO STRICT original FROM pg_proc WHERE oid=f;
 IF length(original)-length(replace(original,extension,''))<>length(extension)
 THEN RAISE EXCEPTION 'AD3-127: núcleo no reversible' USING ERRCODE='55000'; END IF;
 nuevo:=replace(original,extension,'');
 EXECUTE nuevo;
 IF (SELECT pg_get_functiondef(f) FROM pg_proc WHERE oid=f) IS DISTINCT FROM nuevo
 THEN RAISE EXCEPTION 'AD3-127: restauración divergente' USING ERRCODE='55000'; END IF;
END $nucleo$;
LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $audiencias$
DECLARE d text; sufijo text:=', ''vec_contratacion_temporal.categoria_rpt.vinculo.consultar.v1''::text, ''vec_contratacion_temporal.categoria_rpt.vinculo.registrar.v1''::text]))';
BEGIN
 SELECT regexp_replace(pg_get_constraintdef(c.oid,true),'\s+',' ','g') INTO STRICT d
 FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
 AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 IF right(d,length(sufijo))<>sufijo
 THEN RAISE EXCEPTION 'AD3-127: audiencias posteriores o divergentes' USING ERRCODE='55000'; END IF;
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version
  DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '
  ||left(d,length(d)-length(sufijo))||']))';
END $audiencias$;
COMMIT;
