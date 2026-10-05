\set ON_ERROR_STOP on
-- CC8: comprobador de sólo lectura de la publicación vigente del plan nominal
-- de firma CT. Lo usa CT178 antes del PDP para que el descriptor del plan no
-- avance con un fichero que no sea la publicación vigente. Repite las
-- comprobaciones de publicación de leer_plan_nominal_firma_v1 (CC7) sin exigir
-- un consumo de firma: no devuelve el documento ni ninguna entrada, sólo la
-- fecha y la revisión de la publicación. La comprobación definitiva sigue en
-- CT176 con el pin y el consumo. Sin acceso LOGIN: sólo el propietario CT.
-- Una sola vez; sin DOWN.
BEGIN;
SET LOCAL ROLE vec_catalogos_configurables_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_catalogos_configurables:migracion:000008',0));
DO $pre$
BEGIN
 IF current_user<>'vec_catalogos_configurables_propietario'
    OR pg_catalog.current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
    OR pg_catalog.to_regprocedure('vec_catalogos_configurables.leer_plan_nominal_firma_v1(text,bigint,text,text,jsonb)') IS NULL
    OR pg_catalog.to_regprocedure('vec_catalogos_configurables.validar_plan_nominal_firma_v1(bytea,text)') IS NULL
    OR pg_catalog.to_regprocedure('vec_catalogos_configurables.comprobar_publicacion_plan_nominal_firma_v1(text,bigint,text)') IS NOT NULL
    OR pg_catalog.to_regrole('vec_contratacion_temporal_propietario') IS NULL
    OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_class c WHERE c.oid=pg_catalog.to_regclass('vec_catalogos_configurables.plan_firma_control')
      AND c.relowner=current_user::pg_catalog.regrole AND c.relrowsecurity AND c.relforcerowsecurity)
    OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_class c WHERE c.oid=pg_catalog.to_regclass('vec_catalogos_configurables.plan_firma_publicacion')
      AND c.relowner=current_user::pg_catalog.regrole AND c.relrowsecurity AND c.relforcerowsecurity) THEN
  RAISE EXCEPTION 'CC8: PARO clave=preimagen actual=incompatible esperado=CC7_sin_CC8' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE FUNCTION vec_catalogos_configurables.comprobar_publicacion_plan_nominal_firma_v1(
 p_catalogo_id text,p_version bigint,p_publicacion_sha256 text)
RETURNS TABLE(publicada_en timestamptz,revision bigint)
LANGUAGE plpgsql STABLE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog SET row_security='on' SET lock_timeout='2s' SET statement_timeout='10s' SET TimeZone='UTC' AS $f$
DECLARE control vec_catalogos_configurables.plan_firma_control%ROWTYPE;
 publicacion vec_catalogos_configurables.plan_firma_publicacion%ROWTYPE; documento jsonb;
BEGIN
 IF (p_catalogo_id ~ '^[a-z][a-z0-9._-]{2,127}$') IS NOT TRUE
    OR p_version IS NULL OR p_version NOT BETWEEN 1 AND 2147483647
    OR (p_publicacion_sha256 ~ '^[0-9a-f]{64}$') IS NOT TRUE THEN
  RAISE EXCEPTION 'CC8: pin de plan inválido' USING ERRCODE='22023'; END IF;
 SELECT * INTO control FROM vec_catalogos_configurables.plan_firma_control c
  WHERE c.catalogo_id=p_catalogo_id AND c.version=p_version;
 IF NOT FOUND OR control.estado IS DISTINCT FROM 'publicado'
    OR control.publicacion_sha256 IS DISTINCT FROM p_publicacion_sha256
    OR control.modulo_id IS DISTINCT FROM 'contratacion_temporal' THEN
  RAISE EXCEPTION 'CC8: plan retirado o no publicado' USING ERRCODE='42501'; END IF;
 -- Mismo criterio que CC7: sólo vale la publicación vigente del módulo.
 IF EXISTS(SELECT 1 FROM vec_catalogos_configurables.plan_firma_control x
    WHERE x.modulo_id='contratacion_temporal'
      AND ((x.catalogo_id<>p_catalogo_id AND x.estado='publicado')
        OR (x.catalogo_id=p_catalogo_id AND x.version>p_version AND x.estado IN ('publicado','retirado')))) THEN
  RAISE EXCEPTION 'CC8: plan sustituido por otra publicación' USING ERRCODE='42501'; END IF;
 SELECT * INTO publicacion FROM vec_catalogos_configurables.plan_firma_publicacion p
  WHERE p.catalogo_id=p_catalogo_id AND p.version=p_version;
 IF NOT FOUND OR publicacion.publicacion_sha256 IS DISTINCT FROM control.publicacion_sha256
    OR publicacion.revision IS DISTINCT FROM control.publicacion_revision
    OR pg_catalog.encode(pg_catalog.sha256(publicacion.canonico_exacto),'hex') IS DISTINCT FROM p_publicacion_sha256
    OR publicacion.publicada_en>pg_catalog.clock_timestamp() THEN
  RAISE EXCEPTION 'CC8: publicación original incoherente' USING ERRCODE='55000'; END IF;
 documento:=vec_catalogos_configurables.validar_plan_nominal_firma_v1(publicacion.canonico_exacto,p_publicacion_sha256);
 IF documento->>'id' IS DISTINCT FROM p_catalogo_id
    OR documento->>'version' IS DISTINCT FROM p_version::text
    OR documento->>'estado' IS DISTINCT FROM 'publicado'
    OR (documento->>'publicado_en')::timestamptz IS DISTINCT FROM publicacion.publicada_en
    OR documento->>'publicado_por' IS DISTINCT FROM publicacion.publicado_por
    OR documento->>'aprobacion_ref' IS DISTINCT FROM publicacion.aprobacion_ref THEN
  RAISE EXCEPTION 'CC8: publicación ajena' USING ERRCODE='55000'; END IF;
 RETURN QUERY SELECT publicacion.publicada_en,publicacion.revision;
END $f$;
REVOKE ALL ON FUNCTION vec_catalogos_configurables.comprobar_publicacion_plan_nominal_firma_v1(text,bigint,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_catalogos_configurables.comprobar_publicacion_plan_nominal_firma_v1(text,bigint,text)
 TO vec_contratacion_temporal_propietario;
DO $acl$
DECLARE f regprocedure:='vec_catalogos_configurables.comprobar_publicacion_plan_nominal_firma_v1(text,bigint,text)'::regprocedure; x record;
BEGIN
 FOR x IN SELECT DISTINCT a.grantee FROM pg_catalog.pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
  WHERE p.oid=f AND a.grantee NOT IN(p.proowner,'vec_contratacion_temporal_propietario'::pg_catalog.regrole) LOOP
  EXECUTE pg_catalog.format('REVOKE ALL ON FUNCTION %s FROM %s',f::text,
   CASE WHEN x.grantee=0 THEN 'PUBLIC' ELSE pg_catalog.quote_ident(pg_catalog.pg_get_userbyid(x.grantee)) END);
 END LOOP;
 IF (SELECT count(*) FROM pg_catalog.pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) a WHERE p.oid=f)<>2
  OR EXISTS(SELECT 1 FROM pg_catalog.pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) a WHERE p.oid=f
   AND (a.grantee NOT IN(p.proowner,'vec_contratacion_temporal_propietario'::pg_catalog.regrole) OR a.privilege_type<>'EXECUTE' OR a.is_grantable OR a.grantor<>p.proowner)) THEN
  RAISE EXCEPTION 'CC8: PARO clave=ACL actual=incompatible esperado=propietario_CT_EXECUTE' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
