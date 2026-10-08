\set ON_ERROR_STOP on
-- CT178: el ejecutor CT comprueba, antes del PDP, que el plan nominal de firma
-- que usa el descriptor (id, versión y SHA del documento) es la publicación
-- vigente de Catálogos (CC8). Sólo devuelve la fecha y la revisión de esa
-- publicación. No sustituye el pin ni la revalidación de CT176 en la
-- transacción de la firma. Requiere CC7 y CC8. Una sola vez; sin DOWN.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL TimeZone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000178',0));
DO $pre$
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario'
  OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
  OR to_regprocedure('vec_contratacion_temporal.comprobar_plan_firma_publicado_v1(text,bigint,text)') IS NOT NULL
  OR to_regprocedure('vec_catalogos_configurables.comprobar_publicacion_plan_nominal_firma_v1(text,bigint,text)') IS NULL
  OR NOT has_function_privilege(current_user,'vec_catalogos_configurables.comprobar_publicacion_plan_nominal_firma_v1(text,bigint,text)','EXECUTE')
  OR to_regrole('vec_contratacion_temporal_ejecutor') IS NULL THEN
  RAISE EXCEPTION 'CT178: PARO clave=preimagen actual=incompatible esperado=CC8_sin_CT178' USING ERRCODE='55000'; END IF;
END $pre$;
CREATE FUNCTION vec_contratacion_temporal.comprobar_plan_firma_publicado_v1(
 p_catalogo_id text,p_version bigint,p_publicacion_sha256 text)
RETURNS jsonb LANGUAGE plpgsql STABLE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s' SET statement_timeout='10s' AS $f$
DECLARE r record;
BEGIN
 SELECT * INTO STRICT r FROM vec_catalogos_configurables.comprobar_publicacion_plan_nominal_firma_v1(
  p_catalogo_id,p_version,p_publicacion_sha256);
 RETURN jsonb_build_object('publicada_en',to_char(r.publicada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  'revision',r.revision);
END $f$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.comprobar_plan_firma_publicado_v1(text,bigint,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.comprobar_plan_firma_publicado_v1(text,bigint,text) TO vec_contratacion_temporal_ejecutor;
DO $acl$
DECLARE f regprocedure:='vec_contratacion_temporal.comprobar_plan_firma_publicado_v1(text,bigint,text)'::regprocedure; x record;
BEGIN
 FOR x IN SELECT DISTINCT a.grantee FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
  WHERE p.oid=f AND a.grantee NOT IN(p.proowner,'vec_contratacion_temporal_ejecutor'::regrole) LOOP
  EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %s',f::text,CASE WHEN x.grantee=0 THEN 'PUBLIC' ELSE quote_ident(pg_get_userbyid(x.grantee)) END);
 END LOOP;
 IF (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f)<>2
  OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f
   AND (a.grantee NOT IN(p.proowner,'vec_contratacion_temporal_ejecutor'::regrole) OR a.privilege_type<>'EXECUTE' OR a.is_grantable OR a.grantor<>p.proowner)) THEN
  RAISE EXCEPTION 'CT178: PARO clave=ACL actual=incompatible esperado=propietario_ejecutor_EXECUTE' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
