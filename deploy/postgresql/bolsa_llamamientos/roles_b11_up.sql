-- Rol técnico nominal para la consulta exterior B11. Las identidades LOGIN se
-- administran fuera de Git; este fichero no crea cuentas ni concesiones a ellas.
\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_bolsa_llamamientos:roles:b11:up', 0)
);

DO $prevalidacion$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname = current_user AND rolsuper
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'bootstrap B11 requiere superusuario';
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname = 'vec_bolsa_llamamientos_propietario'
           AND NOT rolcanlogin AND NOT rolsuper AND NOT rolcreatedb
           AND NOT rolcreaterole AND NOT rolinherit AND NOT rolreplication
           AND NOT rolbypassrls
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'falta propietario de Bolsa para B11';
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname = 'vec_autorizacion_atestada_v3_propietario'
           AND NOT rolcanlogin AND NOT rolsuper AND NOT rolcreatedb
           AND NOT rolcreaterole AND NOT rolinherit AND NOT rolreplication
           AND NOT rolbypassrls
    )
       OR NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_namespace n
            WHERE n.nspname = 'vec_autorizacion_atestada_v3'
              AND n.nspowner =
                  'vec_autorizacion_atestada_v3_propietario'::regrole
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'falta propietario o esquema VEC-AD-3 para B11';
    END IF;
    -- No se atribuye una concesión anterior desconocida a B11.
    IF pg_catalog.has_schema_privilege(
           'vec_bolsa_llamamientos_propietario',
           'vec_autorizacion_atestada_v3', 'USAGE'
       )
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_namespace n
             CROSS JOIN LATERAL pg_catalog.aclexplode(
                 coalesce(n.nspacl, pg_catalog.acldefault('n', n.nspowner))
             ) a
            WHERE n.nspname = 'vec_autorizacion_atestada_v3'
              AND a.grantee = 'vec_bolsa_llamamientos_propietario'::regrole
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'ACL VEC-AD-3 previa no atribuible a B11';
    END IF;
    IF EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname = 'vec_bolsa_llamamientos_consultor_participaciones_propias'
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'rol exterior B11 ya existe';
    END IF;
END
$prevalidacion$;

CREATE ROLE vec_bolsa_llamamientos_consultor_participaciones_propias
    NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;

GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3
    TO vec_bolsa_llamamientos_propietario;

DO $postvalidacion$
BEGIN
    IF NOT pg_catalog.has_schema_privilege(
           'vec_bolsa_llamamientos_propietario',
           'vec_autorizacion_atestada_v3', 'USAGE'
       )
       OR pg_catalog.has_schema_privilege(
           'vec_bolsa_llamamientos_propietario',
           'vec_autorizacion_atestada_v3', 'CREATE'
       )
       OR (SELECT count(*)
             FROM pg_catalog.pg_namespace n
             CROSS JOIN LATERAL pg_catalog.aclexplode(
                 coalesce(n.nspacl, pg_catalog.acldefault('n', n.nspowner))
             ) a
            WHERE n.nspname = 'vec_autorizacion_atestada_v3'
              AND a.grantee = 'vec_bolsa_llamamientos_propietario'::regrole
              AND (a.privilege_type <> 'USAGE' OR a.is_grantable)) <> 0
       OR (SELECT count(*)
             FROM pg_catalog.pg_namespace n
             CROSS JOIN LATERAL pg_catalog.aclexplode(
                 coalesce(n.nspacl, pg_catalog.acldefault('n', n.nspowner))
             ) a
            WHERE n.nspname = 'vec_autorizacion_atestada_v3'
              AND a.grantee = 'vec_bolsa_llamamientos_propietario'::regrole
              AND a.privilege_type = 'USAGE' AND NOT a.is_grantable) <> 1
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_auth_members m
            WHERE m.member = 'vec_bolsa_llamamientos_propietario'::regrole
              AND m.roleid = 'vec_autorizacion_atestada_v3_propietario'::regrole
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'ACL B11 VEC-AD-3 no es mínima';
    END IF;
END
$postvalidacion$;
COMMIT;
