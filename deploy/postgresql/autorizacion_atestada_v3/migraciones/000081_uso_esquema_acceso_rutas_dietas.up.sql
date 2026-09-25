\set ON_ERROR_STOP on
-- AD3-81: USAGE del esquema AD3 para el ejecutor de Dietas.
-- AD3-50 concede a vec_dietas_ejecutor EXECUTE sobre la fachada
-- registrar_y_consumir_acceso_rutas_dietas_v3_atestada, cuya guarda exige que
-- la sesión sea miembro exacto de ese grupo, pero no le concede USAGE del
-- esquema: la llamada cualificada desde el adaptador Go de rutas
-- (internal/modules/dietas/adapters/postgres/acceso_rutas.go, pool
-- `dsn_consumo`) falla con 42501 «permission denied for schema». Solo se
-- añade USAGE: ninguna otra función, tabla ni tipo del esquema queda al
-- alcance del ejecutor, porque todas revocan PUBLIC y no le conceden nada.
-- No reaplicar: se detiene si el USAGE directo ya existe.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000081',0));

DO $pre$
DECLARE f regprocedure:=to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_acceso_rutas_dietas_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
BEGIN
 IF to_regrole('vec_dietas_ejecutor') IS NULL OR f IS NULL
    OR NOT EXISTS (SELECT 1 FROM pg_proc p, aclexplode(p.proacl) a
                   WHERE p.oid=f AND a.grantee='vec_dietas_ejecutor'::regrole
                     AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)
    OR EXISTS (SELECT 1 FROM pg_proc p, aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
               WHERE p.pronamespace='vec_autorizacion_atestada_v3'::regnamespace AND a.grantee=0)
    OR EXISTS (SELECT 1 FROM pg_proc p, aclexplode(p.proacl) a
               WHERE p.pronamespace='vec_autorizacion_atestada_v3'::regnamespace
                 AND a.grantee='vec_dietas_ejecutor'::regrole AND p.oid<>f)
    OR EXISTS (SELECT 1 FROM pg_class c, aclexplode(c.relacl) a
               WHERE c.relnamespace='vec_autorizacion_atestada_v3'::regnamespace
                 AND a.grantee IN (0,'vec_dietas_ejecutor'::regrole))
    OR EXISTS (SELECT 1 FROM pg_namespace n, aclexplode(n.nspacl) a
               WHERE n.nspname='vec_autorizacion_atestada_v3'
                 AND a.grantee IN (0,'vec_dietas_ejecutor'::regrole)) THEN
  RAISE EXCEPTION 'AD3-81: preimagen AD3-50 incompatible o ya instalada' USING ERRCODE='55000';
 END IF;
END $pre$;

GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_dietas_ejecutor;
COMMIT;
