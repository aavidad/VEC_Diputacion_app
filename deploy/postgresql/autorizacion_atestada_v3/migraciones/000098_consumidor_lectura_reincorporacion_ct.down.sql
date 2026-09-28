\set ON_ERROR_STOP on
-- AD3-98 DOWN: solo sin CT 000134 ni historia de esta audiencia. Con
-- decisiones o consumos se conservan núcleo, audiencia y auditoría.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000098',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;

DO $pre$
BEGIN
 IF to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_lectura_reincorporacion_titular_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 THEN RAISE EXCEPTION 'AD3-98 DOWN: AD3-98 no instalada' USING ERRCODE='55000'; END IF;
 -- El propietario AD3 no tiene USAGE en el esquema CT: inspeccionar el
 -- catálogo evita que to_regprocedure falle antes de la guarda nominal.
 IF EXISTS (SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
            WHERE n.nspname='vec_contratacion_temporal'
              AND p.proname='leer_antecedente_reincorporacion_titular_atestada_v1')
    OR EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace
               WHERE n.nspname='vec_contratacion_temporal' AND c.relname='lectura_reincorporacion_titular_v1')
 THEN RAISE EXCEPTION 'AD3-98 DOWN: CT 000134 sigue instalada' USING ERRCODE='55000'; END IF;
 IF EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.clave_capacidad_version
            WHERE audiencia_consumo='vec_contratacion_temporal.lectura_reincorporacion_titular.v1')
    OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.atestacion_decision_v3 a
               WHERE convert_from(a.capacidad_canonica,'UTF8')::jsonb->>'audiencia_consumo'='vec_contratacion_temporal.lectura_reincorporacion_titular.v1')
 THEN RAISE EXCEPTION 'AD3-98 DOWN: no admitido con historia de lectura de reincorporación' USING ERRCODE='55000'; END IF;
END $pre$;

DROP FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_lectura_reincorporacion_titular_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);

DO $audiencias$
DECLARE d text; resto text; audiencia text:='vec_contratacion_temporal.lectura_reincorporacion_titular.v1';
BEGIN
 SELECT regexp_replace(pg_get_constraintdef(c.oid,true),'\s+',' ','g') INTO STRICT d
 FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
 AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 -- Solo se retira esta audiencia; las añadidas después se conservan.
 IF strpos(d,'CHECK (audiencia_consumo = ANY (ARRAY[')<>1 OR right(d,3)<>']))'
    OR length(d)-length(replace(d,quote_literal(audiencia),''))<>length(quote_literal(audiencia))
 THEN RAISE EXCEPTION 'AD3-98 DOWN: audiencias incompatibles' USING ERRCODE='55000'; END IF;
 resto:=replace(d,', '||quote_literal(audiencia)||'::text','');
 IF resto=d OR strpos(resto,quote_literal(audiencia))<>0 THEN
  RAISE EXCEPTION 'AD3-98 DOWN: audiencia de lectura no localizada' USING ERRCODE='55000';
 END IF;
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version
  DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||resto;
END $audiencias$;

DO $nucleo$
DECLARE
 f oid:='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 original text; nuevo text; actual text; meta jsonb; deps jsonb; acl aclitem[];
 propietario oid; config text[]; definidora boolean;
 marca text:=E'       )\n       OR c ->> ''suite'' <> ''VEC-AD-3-COSE-EDDSA-1''';
 extension text:=$x$           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'lectura_reincorporacion_titular_ct'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contratacion_temporal.lectura_reincorporacion_titular.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'contratacion_temporal.seguimiento.consultar_reincorporacion_titular'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'lectura_reincorporacion_titular_contratacion_temporal'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'verificar_antecedente_reincorporacion_titular'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["cese_evento_ref","cese_recibo_ref","documento_ref","documento_sha256","existe_cese","fecha_efectiva","relacion_ref"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
$x$;
BEGIN
 SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc',p.proacl,p.proowner,p.proconfig,p.prosecdef
 INTO STRICT original,meta,acl,propietario,config,definidora FROM pg_proc p WHERE p.oid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
 INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 -- La extensión se localiza sola, exactamente una vez: otra extensión
 -- instalada después pudo insertarse entre ella y la marca.
 IF length(original)-length(replace(original,extension,''))<>length(extension)
    OR length(original)-length(replace(original,marca,''))<>length(marca)
    OR length(original)-length(replace(original,'lectura_reincorporacion_titular_ct',''))<>length('lectura_reincorporacion_titular_ct')
 THEN RAISE EXCEPTION 'AD3-98 DOWN: extensión no localizada en el núcleo' USING ERRCODE='55000'; END IF;
 nuevo:=replace(original,extension,'');
 EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo OR strpos(actual,'lectura_reincorporacion_titular_ct')<>0
    OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
    OR (SELECT proacl FROM pg_proc WHERE oid=f) IS DISTINCT FROM acl
    OR (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM propietario
    OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM config
    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS DISTINCT FROM definidora
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb) FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
 THEN RAISE EXCEPTION 'AD3-98 DOWN: núcleo alterado fuera del contrato' USING ERRCODE='55000'; END IF;
END $nucleo$;
COMMIT;
