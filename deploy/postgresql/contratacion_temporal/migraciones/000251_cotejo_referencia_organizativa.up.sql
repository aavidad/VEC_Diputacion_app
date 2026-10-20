\set ON_ERROR_STOP on
-- F1-128: cotejo técnico de configuración; no concede acceso a expedientes.
-- Depende de CA45 y Personal54. Consumidor: montaje CT previo a su gobierno.
BEGIN;
SET LOCAL search_path = pg_catalog, pg_temp;
SET LOCAL lock_timeout = '2s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000251',0));
DO $pre$
BEGIN
 IF to_regprocedure('vec_contratacion_temporal.cotejar_referencia_organizativa_v1(text,jsonb)') IS NOT NULL
 OR to_regprocedure('vec_contexto_actor_v1.resolver_correspondencia_organizacion_v1(text)') IS NULL
 OR to_regprocedure('vec_personal.cotejar_unidad_bootstrap_admin_v1(text,jsonb,timestamptz)') IS NULL
 OR NOT has_function_privilege('vec_contratacion_temporal_propietario',
   'vec_personal.cotejar_unidad_bootstrap_admin_v1(text,jsonb,timestamptz)','EXECUTE')
 OR NOT has_function_privilege('vec_contratacion_temporal_propietario',
   'vec_contexto_actor_v1.resolver_correspondencia_organizacion_v1(text)','EXECUTE')
 THEN RAISE EXCEPTION 'CT251: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
CREATE FUNCTION vec_contratacion_temporal.cotejar_referencia_organizativa_v1(p_organizacion text,p_unidad jsonb)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path = pg_catalog, pg_temp SET row_security = on SET timezone = 'UTC'
SET lock_timeout = '2s' SET statement_timeout = '5s' AS $f$
DECLARE organizacion jsonb; unidad jsonb;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
 THEN RAISE EXCEPTION 'CT251: requiere SERIALIZABLE READ WRITE' USING ERRCODE='25000'; END IF;
 -- La rama canónica de CA45 verifica estado, vigencia y procedencia. Una
 -- correspondencia operativa, aunque esté publicada, no es esta configuración.
 organizacion:=vec_contexto_actor_v1.resolver_correspondencia_organizacion_v1(p_organizacion);
 IF organizacion->>'organizacion_canonica_ref' IS DISTINCT FROM p_organizacion
 OR organizacion->'prueba' IS DISTINCT FROM 'null'::jsonb
 THEN RAISE EXCEPTION 'CT251: organización maestra requerida' USING ERRCODE='42501'; END IF;
 -- Personal31 exige una sola unidad, su última revisión y fuente exacta;
 -- mantiene la barrera frente a una retirada concurrente hasta COMMIT.
 unidad:=vec_personal.cotejar_unidad_bootstrap_admin_v1(p_organizacion,p_unidad,clock_timestamp()+interval '5 seconds');
 IF unidad->>'organizacion_ref' IS DISTINCT FROM p_organizacion
 OR unidad->>'unidad_ref' IS DISTINCT FROM p_unidad#>>'{valores,0}'
 THEN RAISE EXCEPTION 'CT251: unidad maestra divergente' USING ERRCODE='42501'; END IF;
 RETURN true;
END $f$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.cotejar_referencia_organizativa_v1(text,jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.cotejar_referencia_organizativa_v1(text,jsonb)
 TO vec_contratacion_temporal_ejecutor;
DO $acl$
DECLARE f oid := 'vec_contratacion_temporal.cotejar_referencia_organizativa_v1(text,jsonb)'::regprocedure;
BEGIN
 IF EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
 WHERE p.oid=f AND (a.grantee NOT IN ('vec_contratacion_temporal_propietario'::regrole,'vec_contratacion_temporal_ejecutor'::regrole)
 OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
 THEN RAISE EXCEPTION 'CT251: ACL inesperada' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
