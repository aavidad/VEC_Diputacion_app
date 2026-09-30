\set ON_ERROR_STOP on
-- CT153: el perfil fijo de reincorporación se concede por organización.
-- El expediente permanece como recurso exacto de la decisión, la lectura y
-- todas las guardas de versión, relación, cese y documento. No se reescribe
-- material, HMAC, historia ni el camino de repetición de CT130/CT134.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SET LOCAL client_min_messages=warning;
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000153',0));

DO $pre$
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario'
    OR NOT pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
    OR to_regprocedure('vec_contratacion_temporal.perfil_reincorporacion_ct153_v1()') IS NOT NULL
    OR to_regprocedure('vec_contratacion_temporal.confirmar_reincorporacion_titular_acreditada_v1(jsonb,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_contratacion_temporal.huella_contexto_go_ct115(jsonb,jsonb)') IS NULL THEN
  RAISE EXCEPTION 'CT153: preimagen o rol incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

DO $confirmacion$
DECLARE
 f regprocedure:='vec_contratacion_temporal.confirmar_reincorporacion_titular_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 anterior text; nuevo text; actual text; meta jsonb; deps jsonb; config text[];
 marca_ambitos text:=$m$v_ambitos:=jsonb_build_object('organizacion_ref',m->>'organizacion_ref','expediente_ref',m->>'expediente_ref',
  'fase_previa','nombramiento','estado_previo','en_curso');$m$;
 cambio_ambitos text:=$m$v_ambitos:=jsonb_build_object('organizacion_ref',m->>'organizacion_ref');$m$;
 marca_atributos text:=$m$v_atributos:=jsonb_build_object('version_expediente',v_version::text,'relacion_ref',v_origen.relacion_ref,$m$;
 cambio_atributos text:=$m$v_atributos:=jsonb_build_object('fase_previa','nombramiento','estado_previo','en_curso',
  'version_expediente',v_version::text,'relacion_ref',v_origen.relacion_ref,$m$;
BEGIN
 SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc',p.proconfig,
  (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
   FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f)
 INTO STRICT anterior,meta,config,deps FROM pg_proc p WHERE p.oid=f;
 IF (SELECT encode(sha256(convert_to(prosrc,'UTF8')),'hex') FROM pg_proc WHERE oid=f)
      IS DISTINCT FROM '128229aaf356667755b7a07acedb7dcba555233d2572d53cea5c2d1c1bc6a990'
    OR (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM current_user::regrole
    OR (SELECT proacl FROM pg_proc WHERE oid=f) IS DISTINCT FROM
       ARRAY['vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario']::aclitem[]
    OR (SELECT prorettype FROM pg_proc WHERE oid=f) IS DISTINCT FROM 'jsonb'::regtype
    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS NOT TRUE
    OR (SELECT provolatile FROM pg_proc WHERE oid=f) IS DISTINCT FROM 'v'
    OR (SELECT proparallel FROM pg_proc WHERE oid=f) IS DISTINCT FROM 'u'
    OR NOT ('search_path=pg_catalog'=ANY(config))
    OR NOT ('row_security=on'=ANY(config))
    OR NOT ('lock_timeout=2s'=ANY(config))
    OR lower(array_to_string(config,',')) NOT LIKE '%timezone=utc%'
    OR has_function_privilege('public',f,'EXECUTE')
    OR has_function_privilege('vec_contratacion_temporal_ejecutor',f,'EXECUTE')
    OR length(anterior)-length(replace(anterior,marca_ambitos,''))<>length(marca_ambitos)
    OR length(anterior)-length(replace(anterior,marca_atributos,''))<>length(marca_atributos) THEN
  RAISE EXCEPTION 'CT153: confirmación CT130 incompatible' USING ERRCODE='55000'; END IF;
 nuevo:=replace(replace(anterior,marca_ambitos,cambio_ambitos),marca_atributos,cambio_atributos);
 EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo
    OR replace(replace(actual,cambio_ambitos,marca_ambitos),cambio_atributos,marca_atributos) IS DISTINCT FROM anterior
    OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
        FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps THEN
  RAISE EXCEPTION 'CT153: confirmación alterada fuera del contrato' USING ERRCODE='55000'; END IF;
END $confirmacion$;

DO $lectura$
DECLARE
 f regprocedure:='vec_contratacion_temporal.leer_antecedente_reincorporacion_titular_atestada_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 anterior text; nuevo text; actual text; meta jsonb; deps jsonb; config text[];
 marca text:=$m$v_ambitos:=jsonb_build_object('organizacion_ref',p_material->>'organizacion_ref','expediente_ref',p_material->>'expediente_ref');$m$;
 cambio text:=$m$v_ambitos:=jsonb_build_object('organizacion_ref',p_material->>'organizacion_ref');$m$;
BEGIN
 SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc',p.proconfig,
  (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
   FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f)
 INTO STRICT anterior,meta,config,deps FROM pg_proc p WHERE p.oid=f;
 IF (SELECT encode(sha256(convert_to(prosrc,'UTF8')),'hex') FROM pg_proc WHERE oid=f)
      IS DISTINCT FROM '604f0d2ee4a88b3b61cdbc8f2f4f9c506d195ba181dbdc228f7d563a380f0d3b'
    OR (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM current_user::regrole
    OR (SELECT proacl FROM pg_proc WHERE oid=f) IS DISTINCT FROM ARRAY[
       'vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario',
       'vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario']::aclitem[]
    OR (SELECT prorettype FROM pg_proc WHERE oid=f) IS DISTINCT FROM 'jsonb'::regtype
    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS NOT TRUE
    OR (SELECT provolatile FROM pg_proc WHERE oid=f) IS DISTINCT FROM 'v'
    OR (SELECT proparallel FROM pg_proc WHERE oid=f) IS DISTINCT FROM 'u'
    OR NOT ('search_path=pg_catalog'=ANY(config))
    OR NOT ('row_security=on'=ANY(config))
    OR NOT ('lock_timeout=2s'=ANY(config))
    OR lower(array_to_string(config,',')) NOT LIKE '%timezone=utc%'
    OR has_function_privilege('public',f,'EXECUTE')
    OR NOT has_function_privilege('vec_contratacion_temporal_ejecutor',f,'EXECUTE')
    OR length(anterior)-length(replace(anterior,marca,''))<>length(marca) THEN
  RAISE EXCEPTION 'CT153: lectura CT134 incompatible' USING ERRCODE='55000'; END IF;
 nuevo:=replace(anterior,marca,cambio);
 EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo OR replace(actual,cambio,marca) IS DISTINCT FROM anterior
    OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
        FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps THEN
  RAISE EXCEPTION 'CT153: lectura alterada fuera del contrato' USING ERRCODE='55000'; END IF;
END $lectura$;

CREATE FUNCTION vec_contratacion_temporal.perfil_reincorporacion_ct153_v1()
RETURNS text LANGUAGE sql IMMUTABLE SECURITY INVOKER PARALLEL SAFE
SET search_path=pg_catalog AS $f$ SELECT 'organizacion_ref'::text $f$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.perfil_reincorporacion_ct153_v1() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.perfil_reincorporacion_ct153_v1()
 TO vec_contratacion_temporal_ejecutor;
COMMIT;
