\set ON_ERROR_STOP on
-- CT134 DOWN solo es válido antes de producir historia de lecturas.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000134',0));
LOCK TABLE vec_contratacion_temporal.lectura_reincorporacion_titular_v1 IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_contratacion_temporal.uso_lectura_reincorporacion_titular_v1 IN ACCESS EXCLUSIVE MODE;
-- La política CT130 ordinaria oculta filas al migrador; habilitar solo esta
-- inspección dentro del DOWN y retirarla antes del COMMIT.
CREATE POLICY ct134_down_inspeccion ON vec_contratacion_temporal.reincorporacion_titular_v1
 FOR SELECT TO vec_contratacion_temporal_propietario USING (
 pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
 AND NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER'));
DO $pre$
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario'
    OR NOT pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
    OR pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
    OR to_regprocedure('vec_contratacion_temporal.leer_antecedente_reincorporacion_titular_atestada_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_contratacion_temporal.exigir_lectura_reincorporacion_ct134(jsonb,text,text,text)') IS NULL
    OR to_regprocedure('vec_contratacion_temporal.preparar_reincorporacion_titular_acreditada_v1(jsonb,text,text)') IS NULL
    OR to_regprocedure('vec_contratacion_temporal.confirmar_reincorporacion_titular_acreditada_v1(jsonb,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR has_function_privilege('vec_contratacion_temporal_ejecutor',
      'vec_contratacion_temporal.preparar_reincorporacion_titular_v1(jsonb)','EXECUTE')
    OR has_function_privilege('vec_contratacion_temporal_ejecutor',
      'vec_contratacion_temporal.confirmar_reincorporacion_titular_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR EXISTS (SELECT 1 FROM vec_contratacion_temporal.lectura_reincorporacion_titular_v1)
    OR EXISTS (SELECT 1 FROM vec_contratacion_temporal.uso_lectura_reincorporacion_titular_v1)
    OR EXISTS (SELECT 1 FROM vec_contratacion_temporal.reincorporacion_titular_v1)
    OR EXISTS (SELECT 1 FROM vec_contratacion_temporal.expediente_version_integral
       WHERE origen_version='reincorporacion_titular_ct130')
    OR EXISTS (SELECT 1 FROM vec_contratacion_temporal.outbox_expediente_integral
       WHERE tipo_evento='ct.reincorporacion_titular.v1') THEN
  RAISE EXCEPTION 'CT134 DOWN: historia o estado incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

DO $preparacion_ro$
DECLARE f regprocedure:='vec_contratacion_temporal.preparar_reincorporacion_titular_v1(jsonb)'::regprocedure;
 original text; nuevo text; actual text; marca text:='PERFORM vec_contratacion_temporal.exigir_sesion_ct115(false);';
 reemplazo text:='PERFORM vec_contratacion_temporal.exigir_sesion_ct115(true);';
 meta jsonb; acl aclitem[]; deps jsonb; config text[];
BEGIN
 SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc',p.proacl,p.proconfig,
  (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
   FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f)
 INTO STRICT original,meta,acl,config,deps FROM pg_proc p WHERE p.oid=f;
 IF (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS NOT TRUE
    OR (SELECT provolatile FROM pg_proc WHERE oid=f) IS DISTINCT FROM 'v'
    OR config IS NULL OR NOT ('search_path=pg_catalog'=ANY(config))
    OR NOT ('row_security=on'=ANY(config))
    OR NOT ('lock_timeout=2s'=ANY(config))
    OR lower(array_to_string(config,',')) NOT LIKE '%timezone=utc%'
    OR length(original)-length(replace(original,marca,''))<>length(marca)
    OR strpos(original,reemplazo)<>0 THEN
  RAISE EXCEPTION 'CT134 DOWN: preparación CT130 incompatible' USING ERRCODE='55000'; END IF;
 nuevo:=replace(original,marca,reemplazo);
 EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo OR replace(actual,reemplazo,marca) IS DISTINCT FROM original
    OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
    OR (SELECT proacl FROM pg_proc WHERE oid=f) IS DISTINCT FROM acl
    OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM config
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
        FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps THEN
  RAISE EXCEPTION 'CT134 DOWN: preparación CT130 alterada fuera del contrato' USING ERRCODE='55000'; END IF;
END $preparacion_ro$;

DROP FUNCTION vec_contratacion_temporal.confirmar_reincorporacion_titular_acreditada_v1(
 jsonb,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_contratacion_temporal.preparar_reincorporacion_titular_acreditada_v1(jsonb,text,text);
DROP FUNCTION vec_contratacion_temporal.exigir_lectura_reincorporacion_ct134(jsonb,text,text,text);
DROP FUNCTION vec_contratacion_temporal.leer_antecedente_reincorporacion_titular_atestada_v1(
 jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP TABLE vec_contratacion_temporal.uso_lectura_reincorporacion_titular_v1;
DROP TABLE vec_contratacion_temporal.lectura_reincorporacion_titular_v1;
DROP POLICY ct134_down_inspeccion ON vec_contratacion_temporal.reincorporacion_titular_v1;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.preparar_reincorporacion_titular_v1(jsonb)
 TO vec_contratacion_temporal_ejecutor;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.confirmar_reincorporacion_titular_v1(
 jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_contratacion_temporal_ejecutor;
DO $acl_restaurada$
BEGIN
 IF NOT has_function_privilege('vec_contratacion_temporal_ejecutor',
  'vec_contratacion_temporal.preparar_reincorporacion_titular_v1(jsonb)','EXECUTE')
   OR NOT has_function_privilege('vec_contratacion_temporal_ejecutor',
  'vec_contratacion_temporal.confirmar_reincorporacion_titular_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
   OR has_function_privilege('public',
  'vec_contratacion_temporal.preparar_reincorporacion_titular_v1(jsonb)','EXECUTE')
   OR has_function_privilege('public',
  'vec_contratacion_temporal.confirmar_reincorporacion_titular_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE') THEN
  RAISE EXCEPTION 'CT134 DOWN: ACL CT130 no restaurada' USING ERRCODE='55000'; END IF;
END $acl_restaurada$;
COMMIT;
