\set ON_ERROR_STOP on
-- AD3-81 DOWN: retira solo el USAGE del esquema AD3 al ejecutor de Dietas.
-- No toca decisiones, consumos ni auditoría; la fachada de rutas deja de ser
-- alcanzable (42501) hasta reinstalar.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000081',0));
DO $pre$
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_namespace n, aclexplode(n.nspacl) a
                WHERE n.nspname='vec_autorizacion_atestada_v3'
                  AND a.grantee='vec_dietas_ejecutor'::regrole AND a.privilege_type='USAGE') THEN
  RAISE EXCEPTION 'AD3-81: no instalada' USING ERRCODE='55000';
 END IF;
END $pre$;
REVOKE USAGE ON SCHEMA vec_autorizacion_atestada_v3 FROM vec_dietas_ejecutor;
COMMIT;
