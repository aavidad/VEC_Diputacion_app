\set ON_ERROR_STOP on
-- AD3-86 DOWN: solo sin historia. Se rechaza si Bolsa 000040 sigue instalada
-- o si ya existe alguna clave de capacidad de la audiencia de confirmación;
-- sin historia deja el núcleo, las audiencias y las ACL como antes de AD3-86.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000086',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;

DO $pre$
BEGIN
 IF to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_contacto_propio_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 THEN RAISE EXCEPTION 'AD3-86 DOWN: AD3-86 no instalada' USING ERRCODE='55000'; END IF;
 IF EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace
            WHERE n.nspname='vec_bolsa_llamamientos' AND c.relname='confirmacion_contacto_participacion')
 THEN RAISE EXCEPTION 'AD3-86 DOWN: Bolsa 000040 sigue instalada' USING ERRCODE='55000'; END IF;
 IF EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.clave_capacidad_version
            WHERE audiencia_consumo='vec_bolsa_llamamientos.participaciones_propias.confirmar_contacto.v1')
 THEN RAISE EXCEPTION 'AD3-86 DOWN: no admitido con historia de la audiencia de confirmación' USING ERRCODE='55000'; END IF;
END $pre$;

DROP FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_contacto_propio_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);

DO $audiencias$
DECLARE d text; resto text; audiencia text:='vec_bolsa_llamamientos.participaciones_propias.confirmar_contacto.v1';
BEGIN
 SELECT regexp_replace(pg_get_constraintdef(c.oid,true),'\s+',' ','g') INTO STRICT d
 FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
 AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 -- Solo se retira la audiencia de confirmación; las añadidas después se conservan.
 IF strpos(d,'CHECK (audiencia_consumo = ANY (ARRAY[')<>1 OR right(d,3)<>']))'
    OR length(d)-length(replace(d,quote_literal(audiencia),''))<>length(quote_literal(audiencia))
 THEN RAISE EXCEPTION 'AD3-86 DOWN: audiencias incompatibles' USING ERRCODE='55000'; END IF;
 resto:=replace(d,', '||quote_literal(audiencia)||'::text','');
 IF resto=d OR strpos(resto,'confirmar_contacto.v1')<>0 THEN
  RAISE EXCEPTION 'AD3-86 DOWN: audiencia de confirmación no localizada' USING ERRCODE='55000';
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
 p_perfil_mutacion IS NOT DISTINCT FROM 'portal_candidato_bolsa'
 AND c->>'operacion' IS NOT DISTINCT FROM 'bolsa.participaciones_propias.confirmar_contacto'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.participaciones_propias.confirmar_contacto.v1'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'participaciones_candidato'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestion_participaciones_propias'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
$x$;
BEGIN
 SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc',p.proacl,p.proowner,p.proconfig,p.prosecdef
 INTO STRICT original,meta,acl,propietario,config,definidora FROM pg_proc p WHERE p.oid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
 INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 IF length(original)-length(replace(original,extension||marca,''))<>length(extension||marca)
    OR length(original)-length(replace(original,'participaciones_propias.confirmar_contacto.v1',''))<>length('participaciones_propias.confirmar_contacto.v1')
 THEN RAISE EXCEPTION 'AD3-86 DOWN: extensión no localizada en el núcleo' USING ERRCODE='55000'; END IF;
 nuevo:=replace(original,extension||marca,marca);
 EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo OR strpos(actual,'participaciones_propias.confirmar_contacto')<>0
    OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
    OR (SELECT proacl FROM pg_proc WHERE oid=f) IS DISTINCT FROM acl
    OR (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM propietario
    OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM config
    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS DISTINCT FROM definidora
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb) FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
 THEN RAISE EXCEPTION 'AD3-86 DOWN: núcleo alterado fuera del contrato' USING ERRCODE='55000'; END IF;
END $nucleo$;
COMMIT;
