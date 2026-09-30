\set ON_ERROR_STOP on
-- AD3-112 DOWN: retira las dos funciones de lectura del portal externo y su
-- rol de preflight. No guardan estado ni historia. Se niega si algún LOGIN
-- sigue siendo miembro del rol (un proceso externo desplegado dejaría de
-- arrancar) o si algún objeto depende de las funciones.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_autorizacion_atestada_v3:migracion:000112', 0));

DO $preimagen$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles
                   WHERE rolname = current_user AND rolsuper)
       OR pg_catalog.to_regprocedure(
           'vec_autorizacion_atestada_v3.comprobar_material_emision_externa_v1(text,jsonb)') IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_autorizacion_atestada_v3.leer_configuracion_externa_v1(text,jsonb)') IS NULL
       OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles
                       WHERE rolname = 'vec_autorizacion_atestada_v3_preflight_externo')
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_proc AS p
             JOIN pg_catalog.pg_namespace AS n ON n.oid = p.pronamespace
             JOIN pg_catalog.pg_roles AS r ON r.oid = p.proowner
            WHERE n.nspname = 'vec_autorizacion_atestada_v3'
              AND p.proname IN ('comprobar_material_emision_externa_v1',
                                'leer_configuracion_externa_v1')
              AND r.rolname <> 'vec_autorizacion_atestada_v3_propietario')
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_auth_members AS m
            WHERE m.roleid = (SELECT r.oid FROM pg_catalog.pg_roles AS r
                               WHERE r.rolname = 'vec_autorizacion_atestada_v3_preflight_externo'))
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_depend AS d
            WHERE d.refclassid = 'pg_catalog.pg_proc'::regclass
              AND d.refobjid IN (
                  pg_catalog.to_regprocedure(
                      'vec_autorizacion_atestada_v3.comprobar_material_emision_externa_v1(text,jsonb)'),
                  pg_catalog.to_regprocedure(
                      'vec_autorizacion_atestada_v3.leer_configuracion_externa_v1(text,jsonb)'))
              AND d.deptype <> 'i')
    THEN
        RAISE EXCEPTION 'AD3-112: preimagen de retirada incompatible' USING ERRCODE = '55000';
    END IF;
END $preimagen$;

SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
DROP FUNCTION vec_autorizacion_atestada_v3.comprobar_material_emision_externa_v1(text,jsonb) RESTRICT;
DROP FUNCTION vec_autorizacion_atestada_v3.leer_configuracion_externa_v1(text,jsonb) RESTRICT;
REVOKE USAGE ON SCHEMA vec_autorizacion_atestada_v3
    FROM vec_autorizacion_atestada_v3_preflight_externo;
RESET ROLE;
DO $base$
BEGIN
    EXECUTE pg_catalog.format(
        'REVOKE CONNECT ON DATABASE %I FROM vec_autorizacion_atestada_v3_preflight_externo',
        pg_catalog.current_database());
END $base$;
DROP ROLE vec_autorizacion_atestada_v3_preflight_externo;
COMMIT;
