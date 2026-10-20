\set ON_ERROR_STOP on
-- F1-128: CT coteja la unidad maestra antes de montar sus capacidades.
-- Reutiliza Personal31; no publica unidades ni asigna permisos funcionales.
BEGIN;
SET LOCAL search_path = pg_catalog, pg_temp;
SET LOCAL lock_timeout = '2s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal:migracion:000054',0));
DO $pre$
DECLARE f oid := to_regprocedure('vec_personal.cotejar_unidad_bootstrap_admin_v1(text,jsonb,timestamptz)');
BEGIN
 IF f IS NULL OR to_regrole('vec_contratacion_temporal_propietario') IS NULL
 OR encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex')
    IS DISTINCT FROM '6facbfac01afaa5b4f70dc2b9ab92063d3a697d08d6d650c269ee67e4e89efc0'
 OR NOT EXISTS(SELECT 1 FROM pg_proc WHERE oid=f AND proowner='vec_personal_propietario'::regrole AND prosecdef)
 OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
    WHERE p.oid=f AND (a.grantee NOT IN ('vec_personal_propietario'::regrole,'vec_autorizacion_propietario'::regrole)
      OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
 THEN RAISE EXCEPTION 'Personal54: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_personal_propietario;
GRANT USAGE ON SCHEMA vec_personal TO vec_contratacion_temporal_propietario;
GRANT EXECUTE ON FUNCTION vec_personal.cotejar_unidad_bootstrap_admin_v1(text,jsonb,timestamptz)
 TO vec_contratacion_temporal_propietario;
COMMIT;
