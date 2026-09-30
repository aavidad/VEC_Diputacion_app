\set ON_ERROR_STOP on
-- CT153 DOWN solo admite instalación aún sin lecturas ni reincorporaciones.
-- Un contexto emitido con el perfil fijo exige conservar la definición UP.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SET LOCAL client_min_messages=warning;
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000153',0));
LOCK TABLE vec_contratacion_temporal.lectura_reincorporacion_titular_v1 IN SHARE ROW EXCLUSIVE MODE;
LOCK TABLE vec_contratacion_temporal.reincorporacion_titular_v1 IN SHARE ROW EXCLUSIVE MODE;

DO $pre$
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario'
    OR NOT pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
    OR to_regprocedure('vec_contratacion_temporal.perfil_reincorporacion_ct153_v1()') IS NULL
    OR (SELECT encode(sha256(convert_to(prosrc,'UTF8')),'hex') FROM pg_proc
        WHERE oid=to_regprocedure('vec_contratacion_temporal.perfil_reincorporacion_ct153_v1()'))
       IS DISTINCT FROM 'a91b48e7845a5a87bd903b50c6131579b0fa6ec5ff0fe121f29e2f871a894cf1'
    OR (SELECT proowner FROM pg_proc
        WHERE oid=to_regprocedure('vec_contratacion_temporal.perfil_reincorporacion_ct153_v1()'))
       IS DISTINCT FROM current_user::regrole
    OR (SELECT proacl FROM pg_proc
        WHERE oid=to_regprocedure('vec_contratacion_temporal.perfil_reincorporacion_ct153_v1()'))
       IS DISTINCT FROM ARRAY[
         'vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario',
         'vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario']::aclitem[]
    OR EXISTS (SELECT 1 FROM vec_contratacion_temporal.lectura_reincorporacion_titular_v1)
    OR EXISTS (SELECT 1 FROM vec_contratacion_temporal.reincorporacion_titular_v1) THEN
  RAISE EXCEPTION 'CT153 DOWN: historia o estado incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

DO $confirmacion$
DECLARE
 f regprocedure:='vec_contratacion_temporal.confirmar_reincorporacion_titular_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 anterior text; nuevo text; actual text; meta jsonb; deps jsonb; config text[];
 marca_ambitos text:=$m$v_ambitos:=jsonb_build_object('organizacion_ref',m->>'organizacion_ref');$m$;
 cambio_ambitos text:=$m$v_ambitos:=jsonb_build_object('organizacion_ref',m->>'organizacion_ref','expediente_ref',m->>'expediente_ref',
  'fase_previa','nombramiento','estado_previo','en_curso');$m$;
 marca_atributos text:=$m$v_atributos:=jsonb_build_object('fase_previa','nombramiento','estado_previo','en_curso',
  'version_expediente',v_version::text,'relacion_ref',v_origen.relacion_ref,$m$;
 cambio_atributos text:=$m$v_atributos:=jsonb_build_object('version_expediente',v_version::text,'relacion_ref',v_origen.relacion_ref,$m$;
BEGIN
 SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc',p.proconfig,
  (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
   FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f)
 INTO STRICT anterior,meta,config,deps FROM pg_proc p WHERE p.oid=f;
 IF (SELECT encode(sha256(convert_to(prosrc,'UTF8')),'hex') FROM pg_proc WHERE oid=f)
      IS DISTINCT FROM '0e35099bbb4eba74671609d9553948bef0812e9d4d5eaf1b46be550aec3268c6'
    OR (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM current_user::regrole
    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS NOT TRUE
    OR NOT ('search_path=pg_catalog'=ANY(config))
    OR NOT ('row_security=on'=ANY(config))
    OR NOT ('lock_timeout=2s'=ANY(config))
    OR has_function_privilege('public',f,'EXECUTE')
    OR has_function_privilege('vec_contratacion_temporal_ejecutor',f,'EXECUTE')
    OR length(anterior)-length(replace(anterior,marca_ambitos,''))<>length(marca_ambitos)
    OR length(anterior)-length(replace(anterior,marca_atributos,''))<>length(marca_atributos) THEN
  RAISE EXCEPTION 'CT153 DOWN: confirmación incompatible' USING ERRCODE='55000'; END IF;
 nuevo:=replace(replace(anterior,marca_ambitos,cambio_ambitos),marca_atributos,cambio_atributos);
 EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo
    OR replace(replace(actual,cambio_ambitos,marca_ambitos),cambio_atributos,marca_atributos) IS DISTINCT FROM anterior
    OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
        FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps THEN
  RAISE EXCEPTION 'CT153 DOWN: confirmación alterada fuera del contrato' USING ERRCODE='55000'; END IF;
END $confirmacion$;

DO $lectura$
DECLARE
 f regprocedure:='vec_contratacion_temporal.leer_antecedente_reincorporacion_titular_atestada_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 anterior text; nuevo text; actual text; meta jsonb; deps jsonb; config text[];
 marca text:=$m$v_ambitos:=jsonb_build_object('organizacion_ref',p_material->>'organizacion_ref');$m$;
 cambio text:=$m$v_ambitos:=jsonb_build_object('organizacion_ref',p_material->>'organizacion_ref','expediente_ref',p_material->>'expediente_ref');$m$;
BEGIN
 SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc',p.proconfig,
  (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
   FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f)
 INTO STRICT anterior,meta,config,deps FROM pg_proc p WHERE p.oid=f;
 IF (SELECT encode(sha256(convert_to(prosrc,'UTF8')),'hex') FROM pg_proc WHERE oid=f)
      IS DISTINCT FROM '570e9652dc6ff181230b08a708f7535053ccd59316e447bc8610b3d71b0aef88'
    OR (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM current_user::regrole
    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS NOT TRUE
    OR NOT ('search_path=pg_catalog'=ANY(config))
    OR NOT ('row_security=on'=ANY(config))
    OR NOT ('lock_timeout=2s'=ANY(config))
    OR has_function_privilege('public',f,'EXECUTE')
    OR NOT has_function_privilege('vec_contratacion_temporal_ejecutor',f,'EXECUTE')
    OR length(anterior)-length(replace(anterior,marca,''))<>length(marca) THEN
  RAISE EXCEPTION 'CT153 DOWN: lectura incompatible' USING ERRCODE='55000'; END IF;
 nuevo:=replace(anterior,marca,cambio);
 EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo OR replace(actual,cambio,marca) IS DISTINCT FROM anterior
    OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
        FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps THEN
  RAISE EXCEPTION 'CT153 DOWN: lectura alterada fuera del contrato' USING ERRCODE='55000'; END IF;
END $lectura$;

DROP FUNCTION vec_contratacion_temporal.perfil_reincorporacion_ct153_v1();
COMMIT;
