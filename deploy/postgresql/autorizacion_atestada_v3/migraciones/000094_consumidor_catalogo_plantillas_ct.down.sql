\set ON_ERROR_STOP on
-- AD3-94 DOWN solo sin CT-131 ni decisiones/capacidades de esta audiencia.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000094',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $pre$
BEGIN
 IF EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace
            WHERE n.nspname='vec_contratacion_temporal' AND c.relname='catalogo_plantillas_historia_v1')
    OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.clave_capacidad_version
               WHERE audiencia_consumo='vec_contratacion_temporal.catalogo_plantillas.v1')
    OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.atestacion_decision_v3 a
               WHERE convert_from(a.capacidad_canonica,'UTF8')::jsonb->>'audiencia_consumo'='vec_contratacion_temporal.catalogo_plantillas.v1')
 THEN RAISE EXCEPTION 'AD3-94: DOWN denegado con dependencias o historia' USING ERRCODE='55000'; END IF;
END $pre$;
DROP FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_catalogo_plantillas_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DO $nucleo$
DECLARE f oid:='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 original text; nuevo text; actual text;
 extension text:=$x$           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'catalogo_plantillas_ct'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contratacion_temporal.catalogo_plantillas.v1'
 AND c->>'operacion' = ANY (ARRAY['contratacion_temporal.plantillas_documentos.consultar','contratacion_temporal.plantillas_documentos.editar','contratacion_temporal.plantillas_documentos.publicar'])
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'catalogo_plantillas_contratacion_temporal'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestionar_catalogo_plantillas_contratacion_temporal'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM 'vec.contratacion_temporal.plantillas_documentos'
 AND c->>'efecto_ref' IS NOT DISTINCT FROM d->>'recurso_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM CASE WHEN c->>'operacion'='contratacion_temporal.plantillas_documentos.consultar'
      THEN '["borrador","editor_de_esta_version","publicado"]'::jsonb ELSE '["catalogo","recibo"]'::jsonb END
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
$x$;
BEGIN
 SELECT pg_get_functiondef(f) INTO STRICT original FROM pg_proc WHERE oid=f;
 IF length(original)-length(replace(original,extension,''))<>length(extension)
 THEN RAISE EXCEPTION 'AD3-94: extensión incompatible' USING ERRCODE='55000'; END IF;
 nuevo:=replace(original,extension,''); EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo OR strpos(actual,'catalogo_plantillas_ct')<>0
 THEN RAISE EXCEPTION 'AD3-94: reversión del núcleo incompatible' USING ERRCODE='55000'; END IF;
END $nucleo$;
DO $audiencia$
DECLARE d text; nueva text;
BEGIN
 SELECT regexp_replace(pg_get_constraintdef(c.oid,true),'\s+',' ','g') INTO STRICT d
 FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
 AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 nueva:=replace(d,', ''vec_contratacion_temporal.catalogo_plantillas.v1''::text','');
 IF length(d)-length(nueva)<>length(', ''vec_contratacion_temporal.catalogo_plantillas.v1''::text')
 THEN RAISE EXCEPTION 'AD3-94: audiencia incompatible' USING ERRCODE='55000'; END IF;
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||nueva;
END $audiencia$;
COMMIT;
