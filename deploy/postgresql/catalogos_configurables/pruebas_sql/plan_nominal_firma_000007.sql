\set ON_ERROR_STOP on
-- Prueba estructural CC7. Se ejecuta después de L, AD177 y CC7 sobre clon.
-- No inserta decisiones, consumos ni concesiones de prueba.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL statement_timeout='15s';
DO $prueba$
DECLARE t text; f regprocedure; p record; permiso record;
 propietario oid:='vec_catalogos_configurables_propietario'::regrole;
 ad oid:='vec_autorizacion_atestada_v3_propietario'::regrole;
 ct oid:='vec_contratacion_temporal_propietario'::regrole;
BEGIN
 FOREACH t IN ARRAY ARRAY['plan_firma_control','plan_firma_historia','plan_firma_publicacion',
  'plan_firma_efecto','plan_firma_outbox'] LOOP
  IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_class c
    WHERE c.oid=pg_catalog.to_regclass('vec_catalogos_configurables.'||t)
      AND c.relowner=propietario AND c.relrowsecurity AND c.relforcerowsecurity) THEN
   RAISE EXCEPTION 'CC7: tabla sin propietario o RLS: %',t USING ERRCODE='55000'; END IF;
  IF EXISTS(SELECT 1 FROM pg_catalog.pg_class c
    CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(c.relacl,
      pg_catalog.acldefault('r',c.relowner))) x
    WHERE c.oid=pg_catalog.to_regclass('vec_catalogos_configurables.'||t)
      AND x.grantee<>propietario) THEN
   RAISE EXCEPTION 'CC7: tabla concedida fuera de propietario: %',t USING ERRCODE='55000'; END IF;
  IF EXISTS(SELECT 1 FROM pg_catalog.pg_type y
    CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(y.typacl,
      pg_catalog.acldefault('T',y.typowner))) x
    WHERE y.typrelid=pg_catalog.to_regclass('vec_catalogos_configurables.'||t)
      AND x.grantee<>propietario) THEN
   RAISE EXCEPTION 'CC7: tipo de fila concedido fuera de propietario: %',t USING ERRCODE='55000'; END IF;
 END LOOP;
 FOREACH f IN ARRAY ARRAY[
  'vec_catalogos_configurables.validar_plan_nominal_firma_v1(bytea,text)'::regprocedure,
  'vec_catalogos_configurables.leer_plan_nominal_firma_v1(text,bigint,text,text,jsonb)'::regprocedure,
  'vec_catalogos_configurables.confirmar_gobierno_plan_nominal_firma_v1(bytea,jsonb)'::regprocedure] LOOP
  SELECT x.proowner,x.prosecdef,x.provolatile,x.proconfig INTO STRICT p
   FROM pg_catalog.pg_proc x WHERE x.oid=f;
  IF p.proowner<>propietario OR NOT p.prosecdef OR
     (f='vec_catalogos_configurables.validar_plan_nominal_firma_v1(bytea,text)'::regprocedure AND p.provolatile<>'i') OR
     (f<>'vec_catalogos_configurables.validar_plan_nominal_firma_v1(bytea,text)'::regprocedure AND p.provolatile<>'v')
     OR NOT (p.proconfig @> ARRAY['search_path=pg_catalog','row_security=on']) THEN
   RAISE EXCEPTION 'CC7: función sin frontera fija: %',f USING ERRCODE='55000'; END IF;
  FOR permiso IN SELECT a.grantee FROM pg_catalog.pg_proc pp
   CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(pp.proacl,
     pg_catalog.acldefault('f',pp.proowner))) a
   WHERE pp.oid=f LOOP
   IF permiso.grantee=0 OR
      (f='vec_catalogos_configurables.validar_plan_nominal_firma_v1(bytea,text)'::regprocedure AND permiso.grantee<>propietario) OR
      (f='vec_catalogos_configurables.leer_plan_nominal_firma_v1(text,bigint,text,text,jsonb)'::regprocedure AND permiso.grantee NOT IN (propietario,ct)) OR
      (f='vec_catalogos_configurables.confirmar_gobierno_plan_nominal_firma_v1(bytea,jsonb)'::regprocedure AND permiso.grantee NOT IN (propietario,ad)) THEN
    RAISE EXCEPTION 'CC7: ACL de función demasiado amplia: %',f USING ERRCODE='55000'; END IF;
  END LOOP;
 END LOOP;
 IF NOT pg_catalog.has_function_privilege(ct,
   'vec_catalogos_configurables.leer_plan_nominal_firma_v1(text,bigint,text,text,jsonb)','EXECUTE')
    OR pg_catalog.has_function_privilege(ct,
   'vec_catalogos_configurables.confirmar_gobierno_plan_nominal_firma_v1(bytea,jsonb)','EXECUTE')
    OR NOT pg_catalog.has_function_privilege(ad,
   'vec_catalogos_configurables.confirmar_gobierno_plan_nominal_firma_v1(bytea,jsonb)','EXECUTE') THEN
  RAISE EXCEPTION 'CC7: ejecutores técnicos incompatibles' USING ERRCODE='55000'; END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_trigger g
    WHERE g.tgrelid='vec_catalogos_configurables.plan_firma_publicacion'::regclass
      AND g.tgname='plan_firma_publicacion_inmutable' AND g.tgenabled IN ('O','A')) THEN
  RAISE EXCEPTION 'CC7: publicación mutable' USING ERRCODE='55000'; END IF;
END $prueba$;
ROLLBACK;
