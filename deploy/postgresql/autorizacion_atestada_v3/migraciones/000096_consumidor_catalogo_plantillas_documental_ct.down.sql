\set ON_ERROR_STOP on
-- AD3-96 DOWN solo sin CT-133 ni decisiones/capacidades de esta audiencia.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000096',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $pre$
BEGIN
 IF EXISTS (SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
            WHERE n.nspname='vec_contratacion_temporal'
              AND p.proname='obtener_catalogo_plantillas_publicado_documental_v1')
    OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.clave_capacidad_version
               WHERE audiencia_consumo='vec_contratacion_temporal.catalogo_plantillas_documental.v1')
    OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.atestacion_decision_v3 a
               WHERE convert_from(a.capacidad_canonica,'UTF8')::jsonb->>'audiencia_consumo'='vec_contratacion_temporal.catalogo_plantillas_documental.v1')
 THEN RAISE EXCEPTION 'AD3-96: DOWN denegado con dependencias o historia' USING ERRCODE='55000'; END IF;
END $pre$;
DROP FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_plantillas_doc_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DO $nucleo$
DECLARE f oid:='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 original text; nuevo text; actual text;
 extension text:=$x$           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'catalogo_plantillas_documental_ct'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contratacion_temporal.catalogo_plantillas_documental.v1'
 AND c->>'operacion' = ANY (ARRAY['contratacion_temporal.plantillas_documentos.documental_listar','contratacion_temporal.plantillas_documentos.documental_descargar'])
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'catalogo_plantillas_documental_ct'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_borradores_expediente'
 AND d->>'recurso_ref' IS NOT NULL AND d->>'recurso_ref' LIKE 'expediente:%'
 AND c->>'efecto_ref' IS NOT DISTINCT FROM d->>'recurso_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["catalogo","catalogo_huella_sha256","contenido_json_sha256","procedencia_ref","revision","version"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
$x$;
BEGIN
 SELECT pg_get_functiondef(f) INTO STRICT original FROM pg_proc WHERE oid=f;
 IF length(original)-length(replace(original,extension,''))<>length(extension)
 THEN RAISE EXCEPTION 'AD3-96: extensión incompatible' USING ERRCODE='55000'; END IF;
 nuevo:=replace(original,extension,''); EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo OR strpos(actual,'catalogo_plantillas_documental_ct')<>0
 THEN RAISE EXCEPTION 'AD3-96: reversión del núcleo incompatible' USING ERRCODE='55000'; END IF;
END $nucleo$;
DO $audiencia$
DECLARE d text; nueva text;
BEGIN
 SELECT regexp_replace(pg_get_constraintdef(c.oid,true),'\s+',' ','g') INTO STRICT d
 FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
 AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 nueva:=replace(d,', ''vec_contratacion_temporal.catalogo_plantillas_documental.v1''::text','');
 IF length(d)-length(nueva)<>length(', ''vec_contratacion_temporal.catalogo_plantillas_documental.v1''::text')
 THEN RAISE EXCEPTION 'AD3-96: audiencia incompatible' USING ERRCODE='55000'; END IF;
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||nueva;
END $audiencia$;
COMMIT;
