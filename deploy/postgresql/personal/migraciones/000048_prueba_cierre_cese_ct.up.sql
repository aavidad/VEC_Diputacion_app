\set ON_ERROR_STOP on
-- Personal48: prueba propietaria de los dos cierres P46 antes del efecto Bolsa.
-- No concede acceso a tablas ni capacidad de lectura a un LOGIN.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal:migracion:000048',0));
DO $pre$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR to_regclass('vec_personal.registro_empleado_b2_recibo') IS NULL
 OR to_regclass('vec_personal.relacion_servicio_historia') IS NULL
 OR to_regclass('vec_personal.ocupacion_empleado_historia') IS NULL
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='vec_bolsa_llamamientos_propietario' AND NOT rolcanlogin)
 THEN RAISE EXCEPTION 'Personal48: dependencias incompatibles' USING ERRCODE='55000'; END IF;
 IF to_regprocedure('vec_personal.probar_cierre_cese_ct_v1(bytea)') IS NOT NULL
 THEN RAISE EXCEPTION 'Personal48: ya instalada' USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_personal_propietario;
CREATE FUNCTION vec_personal.probar_cierre_cese_ct_v1(p_material bytea)
RETURNS boolean LANGUAGE plpgsql SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on
SET lock_timeout='2s' SET statement_timeout='15s' AS $f$
DECLARE m jsonb; hasta date; version_ct bigint; version_rel bigint; version_ocu bigint;
BEGIN
 IF current_user<>'vec_personal_propietario' OR session_user=current_user
 OR NOT pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER')
 OR pg_has_role(session_user,'vec_personal_propietario','MEMBER')
 OR current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off'
 OR current_setting('TimeZone')<>'UTC'
 OR p_material IS NULL OR octet_length(p_material) NOT BETWEEN 1 AND 8192
 THEN RAISE EXCEPTION 'Personal48: prueba de cierre denegada' USING ERRCODE='42501'; END IF;
 m:=convert_from(p_material,'UTF8')::jsonb;
 IF jsonb_typeof(m) IS DISTINCT FROM 'object'
 OR ARRAY(SELECT jsonb_object_keys(m) ORDER BY 1) IS DISTINCT FROM ARRAY[
   'actor_ref','empleado_ref','esquema','fuente_huella_sha256','ocupacion_recibo_ref',
   'ocupacion_ref','ocupacion_version','organismo_ref','recibo_ct_ref',
   'relacion_recibo_ref','relacion_ref','relacion_version','unidad_ref','version_ct','vigente_hasta']
 OR m->>'esquema' IS DISTINCT FROM 'vec.personal.cierre-cese-ct.bolsa.v1'
 OR EXISTS(SELECT 1 FROM jsonb_each(m) x WHERE x.key NOT IN
    ('version_ct','relacion_version','ocupacion_version') AND jsonb_typeof(x.value) IS DISTINCT FROM 'string')
 OR EXISTS(SELECT 1 FROM jsonb_each(m) x WHERE x.key IN
    ('version_ct','relacion_version','ocupacion_version') AND
    (jsonb_typeof(x.value) IS DISTINCT FROM 'number' OR x.value::text !~ '^[1-9][0-9]{0,18}$'))
 OR m->>'fuente_huella_sha256' !~ '^[0-9a-f]{64}$'
 OR m->>'relacion_recibo_ref' IS NOT DISTINCT FROM m->>'ocupacion_recibo_ref'
 THEN RETURN false; END IF;
 hasta:=(m->>'vigente_hasta')::date;
 version_ct:=(m->>'version_ct')::bigint;
 version_rel:=(m->>'relacion_version')::bigint;
 version_ocu:=(m->>'ocupacion_version')::bigint;
 IF NOT isfinite(hasta) OR hasta::text IS DISTINCT FROM m->>'vigente_hasta'
 OR version_rel NOT BETWEEN 2 AND 2147483647 OR version_ocu NOT BETWEEN 2 AND 2147483647
 THEN RETURN false; END IF;
 -- Los recibos identifican revisiones inmutables; el cotejo no depende de
 -- la última relación que la persona pudiera iniciar después de este cese.
 RETURN EXISTS(
  SELECT 1 FROM vec_personal.registro_empleado_b2_recibo rr
  JOIN vec_personal.registro_empleado_b2_recibo ro ON ro.recibo_ref=m->>'ocupacion_recibo_ref'
  JOIN vec_personal.relacion_servicio_historia r ON r.relacion_ref=rr.relacion_ref AND r.revision=rr.version
  JOIN vec_personal.ocupacion_empleado_historia o ON o.ocupacion_ref=ro.hecho_ref AND o.revision=ro.version
  WHERE rr.recibo_ref=m->>'relacion_recibo_ref'
   AND rr.operacion='hecho' AND ro.operacion='hecho' AND rr.tipo='relacion' AND ro.tipo='ocupacion'
   AND rr.actor_ref=m->>'actor_ref' AND ro.actor_ref=rr.actor_ref
   AND rr.empleado_ref=m->>'empleado_ref' AND ro.empleado_ref=rr.empleado_ref
   AND rr.relacion_ref=m->>'relacion_ref' AND ro.relacion_ref=rr.relacion_ref
   AND ro.hecho_ref=m->>'ocupacion_ref' AND rr.version=version_rel AND ro.version=version_ocu
   AND r.empleado_ref=rr.empleado_ref AND o.empleado_ref=rr.empleado_ref
   AND o.relacion_ref=r.relacion_ref AND o.relacion_revision=r.revision
   AND r.organismo_ref=m->>'organismo_ref' AND o.organismo_ref=r.organismo_ref
   AND r.unidad_ref=m->>'unidad_ref' AND o.unidad_ref=r.unidad_ref
   AND r.estado='finalizada' AND o.estado='finalizada' AND r.vigente_hasta=hasta AND o.vigente_hasta=hasta
   AND r.acto_ref=m->>'recibo_ct_ref' AND r.fuente_ref=r.acto_ref
   AND o.acto_ref=r.acto_ref AND o.fuente_ref=r.fuente_ref
   AND r.fuente_version=version_ct AND o.fuente_version=r.fuente_version
   AND r.fuente_huella_sha256=m->>'fuente_huella_sha256' AND o.fuente_huella_sha256=r.fuente_huella_sha256
   AND r.decision_ref=rr.decision_ref AND o.decision_ref=ro.decision_ref
   AND r.auditoria_ref=rr.auditoria_ref AND o.auditoria_ref=ro.auditoria_ref
 );
EXCEPTION WHEN data_exception THEN RETURN false;
END $f$;
REVOKE ALL ON FUNCTION vec_personal.probar_cierre_cese_ct_v1(bytea) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_personal TO vec_bolsa_llamamientos_propietario;
GRANT EXECUTE ON FUNCTION vec_personal.probar_cierre_cese_ct_v1(bytea) TO vec_bolsa_llamamientos_propietario;
COMMIT;
