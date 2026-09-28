\set ON_ERROR_STOP on
-- AD3-95 DOWN solo para un entorno sin historia. No ejecutar sobre bases
-- conservadas ni mientras Bolsa B45 use esta audiencia.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000095',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;

DO $pre$
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario'
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_politica_cese_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.clave_capacidad_version
               WHERE audiencia_consumo='vec_bolsa_llamamientos.politica_cese.consultar.v1')
    OR to_regprocedure('vec_bolsa_llamamientos.consultar_politica_cese_bolsa_v1()') IS NOT NULL
 THEN RAISE EXCEPTION 'AD3-95 DOWN: historia o consumidor presente' USING ERRCODE='55000'; END IF;
END $pre$;

DROP FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_politica_cese_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
REVOKE USAGE ON SCHEMA vec_autorizacion_atestada_v3 FROM vec_bolsa_llamamientos_ejecutor;

DO $audiencia$
DECLARE d text; resto text; a text:='vec_bolsa_llamamientos.politica_cese.consultar.v1';
BEGIN
 SELECT regexp_replace(pg_get_constraintdef(c.oid,true),'\s+',' ','g') INTO STRICT d
 FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
 AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 IF strpos(d,'CHECK (audiencia_consumo = ANY (ARRAY[')<>1 OR right(d,3)<>']))'
    OR length(d)-length(replace(d,quote_literal(a),''))<>length(quote_literal(a))
 THEN RAISE EXCEPTION 'AD3-95 DOWN: audiencia incompatible' USING ERRCODE='55000'; END IF;
 resto:=replace(d,', '||quote_literal(a)||'::text','');
 IF resto=d OR strpos(resto,quote_literal(a))<>0 THEN RAISE EXCEPTION 'AD3-95 DOWN: audiencia no localizada' USING ERRCODE='55000'; END IF;
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||resto;
END $audiencia$;

DO $nucleo$
DECLARE
 f oid:='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 original text; nuevo text; actual text; meta jsonb; deps jsonb; acl aclitem[]; propietario oid; config text[]; definidora boolean;
 marca text:=E'       )\n       OR c ->> ''suite'' <> ''VEC-AD-3-COSE-EDDSA-1''';
 excl text:=E'               p_perfil_mutacion IS DISTINCT FROM ''bolsa_llamamiento''\n';
 excl_nuevo text:=excl||E'               AND p_perfil_mutacion IS DISTINCT FROM ''consulta_politica_cese_bolsa''\n';
 runtime text:=E'(p_perfil_mutacion IS NOT DISTINCT FROM ''bolsa_llamamiento''\n';
 runtime_nuevo text:=runtime||E'               OR p_perfil_mutacion IS NOT DISTINCT FROM ''consulta_politica_cese_bolsa''\n';
 extension text:=$x$           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_politica_cese_bolsa'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.politica_cese.consultar.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'bolsa.politica_cese.consultar'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'politica_cese_bolsa'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM 'politica:bolsa:cese:vigente'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'consulta_politica_cese_rrhh'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["politica_cese"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
$x$;
BEGIN
 SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc',p.proacl,p.proowner,p.proconfig,p.prosecdef
 INTO STRICT original,meta,acl,propietario,config,definidora FROM pg_proc p WHERE p.oid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
 INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 IF propietario<>'vec_autorizacion_atestada_v3_propietario'::regrole OR NOT definidora
    OR config IS DISTINCT FROM ARRAY['search_path=pg_catalog','lock_timeout=2s']
    OR length(original)-length(replace(original,extension,''))<>length(extension)
    OR length(original)-length(replace(original,excl_nuevo,''))<>length(excl_nuevo)
    OR length(original)-length(replace(original,runtime_nuevo,''))<>length(runtime_nuevo)
    OR length(original)-length(replace(original,marca,''))<>length(marca)
 THEN RAISE EXCEPTION 'AD3-95 DOWN: núcleo incompatible' USING ERRCODE='55000'; END IF;
 nuevo:=replace(original,extension,'');
 nuevo:=replace(nuevo,runtime_nuevo,runtime);
 nuevo:=replace(nuevo,excl_nuevo,excl);
 EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo OR strpos(actual,'consulta_politica_cese_bolsa')<>0
    OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc WHERE oid=f) IS DISTINCT FROM meta
    OR (SELECT proacl FROM pg_proc WHERE oid=f) IS DISTINCT FROM acl
    OR (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM propietario
    OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM config
    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS DISTINCT FROM definidora
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb) FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
 THEN RAISE EXCEPTION 'AD3-95 DOWN: núcleo alterado fuera del contrato' USING ERRCODE='55000'; END IF;
END $nucleo$;
COMMIT;
