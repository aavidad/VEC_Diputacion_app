\set ON_ERROR_STOP on
-- CT232 / F1-65A. Retira la entrada de compatibilidad CT226 sin consumidores.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000232',0));
DO $pre$
DECLARE antigua record; vigente record;
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
    OR current_user<>'vec_contratacion_temporal_propietario'
 THEN RAISE EXCEPTION 'CT232: entorno incompatible' USING ERRCODE='55000'; END IF;
 SELECT p.* INTO antigua FROM pg_proc p
  WHERE p.oid=to_regprocedure('vec_contratacion_temporal.comprobar_catalogo_plantillas_base_v1(text,bigint,text)');
 SELECT p.* INTO vigente FROM pg_proc p
  WHERE p.oid=to_regprocedure('vec_contratacion_temporal.comprobar_catalogo_plantillas_base_v2(text,text,bigint,text)');
 IF antigua.oid IS NULL OR vigente.oid IS NULL
 THEN RAISE EXCEPTION 'CT232: CT226 incompleta o ya retirada' USING ERRCODE='55000'; END IF;
 IF encode(sha256(convert_to(pg_get_functiondef(antigua.oid),'UTF8')),'hex')
      IS DISTINCT FROM '021713a64d1c10265e402e211cd885d5d81f55dc5a21bc96d0475792e2e056bc'
    OR encode(sha256(convert_to(pg_get_functiondef(vigente.oid),'UTF8')),'hex')
      IS DISTINCT FROM '0ffe98d71d237a5bd056e6a7e59c69ac16adb9db80144fad1ec566ffc9b0dc82'
    OR antigua.proowner IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
    OR vigente.proowner IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
    OR antigua.prokind IS DISTINCT FROM 'f' OR vigente.prokind IS DISTINCT FROM 'f'
    OR antigua.prosecdef IS DISTINCT FROM true OR vigente.prosecdef IS DISTINCT FROM true
    OR antigua.provolatile IS DISTINCT FROM 's' OR vigente.provolatile IS DISTINCT FROM 's'
    OR antigua.proacl IS DISTINCT FROM ARRAY[
      'vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario',
      'vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario']::aclitem[]
    OR vigente.proacl IS DISTINCT FROM antigua.proacl
    OR antigua.proconfig IS DISTINCT FROM ARRAY[
      'search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s']::text[]
    OR vigente.proconfig IS DISTINCT FROM antigua.proconfig
 THEN RAISE EXCEPTION 'CT232: definición o permisos divergentes' USING ERRCODE='55000'; END IF;
 IF EXISTS (SELECT 1 FROM pg_depend d
            WHERE d.refclassid='pg_proc'::regclass AND d.refobjid=antigua.oid)
 THEN RAISE EXCEPTION 'CT232: V1 tiene dependencias registradas' USING ERRCODE='55000'; END IF;
END $pre$;
DROP FUNCTION vec_contratacion_temporal.comprobar_catalogo_plantillas_base_v1(text,bigint,text) RESTRICT;
DO $post$
DECLARE p record;
BEGIN
 IF to_regprocedure('vec_contratacion_temporal.comprobar_catalogo_plantillas_base_v1(text,bigint,text)') IS NOT NULL
 THEN RAISE EXCEPTION 'CT232: V1 persiste' USING ERRCODE='55000'; END IF;
 SELECT x.* INTO p FROM pg_proc x
  WHERE x.oid=to_regprocedure('vec_contratacion_temporal.comprobar_catalogo_plantillas_base_v2(text,text,bigint,text)');
 IF p.oid IS NULL OR encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex')
       IS DISTINCT FROM '0ffe98d71d237a5bd056e6a7e59c69ac16adb9db80144fad1ec566ffc9b0dc82'
    OR p.proowner IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
    OR p.proacl IS DISTINCT FROM ARRAY[
      'vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario',
      'vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario']::aclitem[]
    OR p.prosecdef IS DISTINCT FROM true OR p.provolatile IS DISTINCT FROM 's'
    OR p.proconfig IS DISTINCT FROM ARRAY[
      'search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s']::text[]
 THEN RAISE EXCEPTION 'CT232: V2 divergente tras retirar V1' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
