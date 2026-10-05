\set ON_ERROR_STOP on
-- AUT52: acredita que la asignación actual de Aplicación (rol
-- administracion_perfiles con metadatos de perfil fijo, como AUT48) está activa
-- y vigente, es de la persona indicada y tiene la organización y la unidad del
-- recurso de gobierno del plan nominal de firma. La usa AD201 dentro de la
-- transacción del consumo. Sólo la ejecuta el propietario AD. No escribe nada.
-- Una sola vez; sin DOWN.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion:migracion:000052',0));
DO $pre$
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 THEN RAISE EXCEPTION 'AUT52: PARO clave=operador actual=incompatible esperado=superusuario_PG18' USING ERRCODE='55000';END IF;
 IF to_regprocedure('vec_autorizacion.acreditar_ambito_certificado_nominal_vigente_aut48(text,text,text)') IS NULL
 OR to_regclass('vec_autorizacion.perfil_fijo_categoria_nominal_v1') IS NULL
 OR to_regclass('vec_autorizacion.asignacion_perfil_actual') IS NULL
 OR to_regprocedure('vec_autorizacion.acreditar_ambitos_gobierno_plan_firma_v1(text,text,text,text,text)') IS NOT NULL
 OR to_regrole('vec_autorizacion_atestada_v3_propietario') IS NULL
 THEN RAISE EXCEPTION 'AUT52: PARO clave=preimagen actual=incompatible esperado=AUT48_sin_AUT52' USING ERRCODE='55000';END IF;
END $pre$;
SET LOCAL ROLE vec_autorizacion_propietario;
CREATE FUNCTION vec_autorizacion.acreditar_ambitos_gobierno_plan_firma_v1(
 version_ref text,p_asignacion_ref text,principal_ref text,org text,unidad text)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE a record; ahora timestamptz;
BEGIN
 IF pg_catalog.current_setting('transaction_isolation')<>'serializable'
  OR pg_catalog.current_setting('transaction_read_only')<>'off'
  OR (version_ref ~ '^rol:administracion_perfiles:v[1-9][0-9]{0,8}$') IS NOT TRUE
  OR principal_ref IS NULL
  OR (org ~ '^org_[a-z0-9]{16,80}$') IS NOT TRUE
  OR (unidad ~ '^[A-Za-z0-9][A-Za-z0-9:._-]{2,159}$') IS NOT TRUE THEN RETURN false; END IF;
 -- Mismo criterio de versión que AUT48: perfil fijo de Aplicación ligado a ella.
 IF NOT EXISTS(SELECT 1 FROM vec_autorizacion.version_rol r JOIN vec_autorizacion.perfil_fijo_categoria_nominal_v1 meta
  ON meta.version_rol_ref=r.version_rol_ref AND meta.fuente_ref=r.version_rol_ref AND meta.fuente_version=r.version AND meta.version_rol_huella_sha256=r.huella_sha256
  WHERE r.version_rol_ref=version_ref AND r.rol_id='administracion_perfiles' AND r.version_rol_ref='rol:administracion_perfiles:v'||r.version::text
  AND meta.categoria_administrativa='aplicacion' AND meta.tipo_perfil='fijo_sistema') THEN RETURN false; END IF;
 SELECT x.* INTO a FROM vec_autorizacion.asignacion_perfil_actual p
 JOIN vec_autorizacion.asignacion_perfil x USING(perfil_activo_ref,asignacion_ref)
 WHERE p.asignacion_ref=p_asignacion_ref FOR SHARE OF p,x;
 ahora:=pg_catalog.clock_timestamp();
 IF NOT FOUND OR a.version_rol_ref<>version_ref OR a.principal_id IS DISTINCT FROM principal_ref
  OR a.documento->>'estado'<>'activa'
  OR (ahora>=(a.documento->>'vigente_desde')::timestamptz AND ahora<(a.documento->>'vigente_hasta')::timestamptz) IS NOT TRUE THEN RETURN false; END IF;
 RETURN a.documento->'ambitos' @> pg_catalog.jsonb_build_array(
   pg_catalog.jsonb_build_object('clave','organizacion_ref','valores',pg_catalog.jsonb_build_array(org)))
  AND a.documento->'ambitos' @> pg_catalog.jsonb_build_array(
   pg_catalog.jsonb_build_object('clave','unidad_ref','valores',pg_catalog.jsonb_build_array(unidad)));
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.acreditar_ambitos_gobierno_plan_firma_v1(text,text,text,text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion.acreditar_ambitos_gobierno_plan_firma_v1(text,text,text,text,text) TO vec_autorizacion_atestada_v3_propietario;
RESET ROLE;
DO $post$
DECLARE f regprocedure:='vec_autorizacion.acreditar_ambitos_gobierno_plan_firma_v1(text,text,text,text,text)'::regprocedure;
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f AND p.proowner='vec_autorizacion_propietario'::regrole AND p.prosecdef)
 OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f)<>2
 OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f
   AND (a.grantee NOT IN(p.proowner,'vec_autorizacion_atestada_v3_propietario'::regrole) OR a.privilege_type<>'EXECUTE' OR a.is_grantable OR a.grantor<>p.proowner))
 THEN RAISE EXCEPTION 'AUT52: PARO clave=acl_post esperado=propietario_y_AD actual=divergente' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
