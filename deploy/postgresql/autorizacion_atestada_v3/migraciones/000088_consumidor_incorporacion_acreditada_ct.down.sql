\set ON_ERROR_STOP on
-- AD3-88 DOWN: solo sin historia. Exige que CT 000124 esté retirada y que no
-- exista ninguna clave de capacidad publicada para su audiencia propia: sin
-- clave no pudo haber consumo. Con historia, la reversión es un procedimiento
-- supervisado que conserva decisiones, consumos y auditoría.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000088',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;

DO $nucleo$
DECLARE f oid:='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 original text; nuevo text; actual text; meta jsonb; deps jsonb; acl aclitem[]; propietario oid; config text[]; definidora boolean;
 extension text:=$x$           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'confirmacion_ginpix_ct'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contratacion_temporal.confirmacion_ginpix.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'contratacion_temporal.ginpix.confirmar'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'confirmacion_ginpix_contratacion_temporal'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'confirmar_ginpix_contratacion_temporal'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'incorporacion_centro_ct'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contratacion_temporal.confirmar_alta_atestada.v1'
 AND c->>'operacion' = ANY (ARRAY['contratacion_temporal.incorporacion.confirmar_centro','contratacion_temporal.incorporacion.consultar_centro'])
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'incorporacion_centro_contratacion_temporal'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestionar_peticion_centro'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
$x$;
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario'
    OR EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace
               WHERE n.nspname='vec_contratacion_temporal' AND c.relname IN ('confirmacion_ginpix_v1','incorporacion_centro_v1'))
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_confirmacion_ginpix_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_incorporacion_centro_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.clave_capacidad_version
               WHERE audiencia_consumo='vec_contratacion_temporal.confirmacion_ginpix.v1')
 THEN RAISE EXCEPTION 'AD3-88: DOWN no admitido (CT 000124 instalada o historia de claves)' USING ERRCODE='55000'; END IF;
 SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc',p.proacl,p.proowner,p.proconfig,p.prosecdef
 INTO STRICT original,meta,acl,propietario,config,definidora FROM pg_proc p WHERE p.oid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
 INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 IF length(original)-length(replace(original,extension,''))<>length(extension)
 THEN RAISE EXCEPTION 'AD3-88: extensión del núcleo no localizada exactamente una vez' USING ERRCODE='55000'; END IF;
 nuevo:=replace(original,extension,'');
 EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo OR strpos(actual,'confirmacion_ginpix_ct')<>0 OR strpos(actual,'incorporacion_centro_ct')<>0
    OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
    OR (SELECT proacl FROM pg_proc WHERE oid=f) IS DISTINCT FROM acl
    OR (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM propietario
    OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM config
    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS DISTINCT FROM definidora
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
        FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
 THEN RAISE EXCEPTION 'AD3-88: núcleo alterado fuera de contrato al revertir' USING ERRCODE='55000'; END IF;
END $nucleo$;

DROP FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_confirmacion_ginpix_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_incorporacion_centro_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);

DO $audiencias$
DECLARE d text; nueva text;
BEGIN
 SELECT regexp_replace(pg_get_constraintdef(c.oid,true),'\s+',' ','g') INTO STRICT d
 FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
 AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 nueva:=replace(d,', ''vec_contratacion_temporal.confirmacion_ginpix.v1''::text','');
 IF length(d)-length(nueva)<>length(', ''vec_contratacion_temporal.confirmacion_ginpix.v1''::text')
 THEN RAISE EXCEPTION 'AD3-88: audiencias no localizadas al revertir' USING ERRCODE='55000'; END IF;
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version
   DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||nueva;
END $audiencias$;
COMMIT;
