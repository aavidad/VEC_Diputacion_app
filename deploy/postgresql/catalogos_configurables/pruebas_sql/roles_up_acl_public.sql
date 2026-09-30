\set ON_ERROR_STOP on
-- Solo psql con superusuario en base desechable, antes de instalar roles_up.
-- Conserva los permisos PUBLIC ya existentes, incluso si son amplios.
CREATE TEMP TABLE vec_rpt_acl_public_pre AS
SELECT 'database'::text AS ambito, a.privilege_type, a.is_grantable
  FROM pg_catalog.pg_database AS d,
       LATERAL pg_catalog.aclexplode(COALESCE(d.datacl,
           pg_catalog.acldefault('d', d.datdba))) AS a
 WHERE d.datname = pg_catalog.current_database() AND a.grantee = 0
UNION ALL
SELECT 'public'::text, a.privilege_type, a.is_grantable
  FROM pg_catalog.pg_namespace AS n,
       LATERAL pg_catalog.aclexplode(COALESCE(n.nspacl,
           pg_catalog.acldefault('n', n.nspowner))) AS a
 WHERE n.nspname = 'public' AND a.grantee = 0;

\ir ../roles_up.sql

DO $verificar$
DECLARE distinta boolean;
BEGIN
    WITH actual AS (
        SELECT 'database'::text AS ambito, a.privilege_type, a.is_grantable
          FROM pg_catalog.pg_database AS d,
               LATERAL pg_catalog.aclexplode(COALESCE(d.datacl,
                   pg_catalog.acldefault('d', d.datdba))) AS a
         WHERE d.datname = pg_catalog.current_database() AND a.grantee = 0
        UNION ALL
        SELECT 'public'::text, a.privilege_type, a.is_grantable
          FROM pg_catalog.pg_namespace AS n,
               LATERAL pg_catalog.aclexplode(COALESCE(n.nspacl,
                   pg_catalog.acldefault('n', n.nspowner))) AS a
         WHERE n.nspname = 'public' AND a.grantee = 0
    ), diferencia AS (
        (SELECT ambito, privilege_type, is_grantable FROM pg_temp.vec_rpt_acl_public_pre
         EXCEPT ALL
         SELECT ambito, privilege_type, is_grantable FROM actual)
        UNION ALL
        (SELECT ambito, privilege_type, is_grantable FROM actual
         EXCEPT ALL
         SELECT ambito, privilege_type, is_grantable FROM pg_temp.vec_rpt_acl_public_pre)
    )
    SELECT EXISTS (SELECT 1 FROM diferencia) INTO distinta;
    IF pg_catalog.to_regnamespace('vec_catalogos_configurables') IS NULL
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_namespace AS n,
                   LATERAL pg_catalog.aclexplode(COALESCE(n.nspacl,
                       pg_catalog.acldefault('n', n.nspowner))) AS a
                  WHERE n.nspname = 'vec_catalogos_configurables' AND a.grantee = 0)
       OR distinta THEN
        RAISE EXCEPTION 'roles_up altero ACL PUBLIC o dejo abierto su esquema';
    END IF;
END $verificar$;

\ir ../roles_down.sql
DROP TABLE pg_temp.vec_rpt_acl_public_pre;
