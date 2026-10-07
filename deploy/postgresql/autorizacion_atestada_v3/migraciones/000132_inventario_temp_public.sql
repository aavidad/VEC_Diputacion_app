\set ON_ERROR_STOP on
SET search_path = pg_catalog;
SET statement_timeout = '30s';
WITH db AS (
  SELECT oid, datname, datdba, datallowconn, datacl,
         pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
           pg_catalog.jsonb_build_object('oid', oid, 'name', datname,
             'owner', datdba, 'allowconn', datallowconn,
             'acl', datacl::text)::text, 'UTF8')), 'hex') AS acl_sha256
    FROM pg_catalog.pg_database WHERE datname = current_database()
), role_graph AS (
  SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
    pg_catalog.jsonb_build_object(
      'roles', (SELECT coalesce(pg_catalog.jsonb_agg(
          pg_catalog.jsonb_build_array(oid, rolname, rolsuper, rolinherit,
            rolcreaterole, rolcreatedb, rolcanlogin, rolreplication,
            rolbypassrls, rolconnlimit, rolvaliduntil, rolconfig)
          ORDER BY oid), '[]'::jsonb) FROM pg_catalog.pg_roles),
      'memberships', (SELECT coalesce(pg_catalog.jsonb_agg(
          pg_catalog.jsonb_build_array(roleid, member, grantor, admin_option,
            inherit_option, set_option) ORDER BY roleid, member, grantor),
          '[]'::jsonb) FROM pg_catalog.pg_auth_members),
      'role_settings', (SELECT coalesce(pg_catalog.jsonb_agg(
          pg_catalog.jsonb_build_array(setdatabase, setrole, setconfig)
          ORDER BY setdatabase, setrole), '[]'::jsonb)
          FROM pg_catalog.pg_db_role_setting)
    )::text, 'UTF8')), 'hex') AS sha256
), logins AS (
  SELECT r.rolname, r.oid, r.rolsuper, r.rolcanlogin,
         pg_catalog.has_database_privilege(r.oid, db.oid, 'CONNECT') AS can_connect,
         pg_catalog.has_database_privilege(r.oid, db.oid, 'TEMPORARY') AS can_temp,
         (SELECT count(*) FROM pg_catalog.pg_stat_activity a
           WHERE a.usesysid = r.oid AND a.pid <> pg_catalog.pg_backend_pid()) AS active_sessions,
         (SELECT coalesce(pg_catalog.jsonb_agg(g.rolname ORDER BY g.rolname), '[]'::jsonb)
            FROM pg_catalog.pg_roles g
           WHERE g.oid <> r.oid AND pg_catalog.pg_has_role(r.oid, g.oid, 'MEMBER')) AS memberships,
         (SELECT coalesce(pg_catalog.jsonb_agg(n.nspname ORDER BY n.nspname), '[]'::jsonb)
            FROM pg_catalog.pg_namespace n
           WHERE n.nspname LIKE 'vec\_%' ESCAPE '\'
             AND pg_catalog.has_schema_privilege(r.oid, n.oid, 'USAGE')) AS vec_schema_usage
    FROM pg_catalog.pg_roles r CROSS JOIN db WHERE r.rolcanlogin
)
SELECT pg_catalog.jsonb_build_object(
   'database_name', db.datname, 'database_oid', db.oid::text,
   'owner_name', owner.rolname, 'owner_oid', db.datdba::text,
   'allowconn', db.datallowconn, 'acl_sha256', db.acl_sha256,
   'role_graph_sha256', role_graph.sha256,
   'public_connect', EXISTS (
      SELECT 1 FROM pg_catalog.aclexplode(coalesce(db.datacl,
        pg_catalog.acldefault('d', db.datdba))) a
       WHERE a.grantee=0 AND a.privilege_type='CONNECT'),
   'public_temp', EXISTS (
      SELECT 1 FROM pg_catalog.aclexplode(coalesce(db.datacl,
        pg_catalog.acldefault('d', db.datdba))) a
       WHERE a.grantee=0 AND a.privilege_type='TEMPORARY'),
   'connect_roles', (SELECT coalesce(pg_catalog.jsonb_agg(rolname ORDER BY rolname), '[]'::jsonb)
       FROM logins WHERE can_connect),
   'login_roles', (SELECT coalesce(pg_catalog.jsonb_agg(rolname ORDER BY rolname), '[]'::jsonb)
       FROM logins),
   'logins', (SELECT coalesce(pg_catalog.jsonb_agg(
      pg_catalog.jsonb_build_object('role', rolname, 'oid', oid::text,
        'superuser', rolsuper, 'connect', can_connect, 'temp', can_temp,
        'active_sessions', active_sessions, 'memberships', memberships,
        'vec_schema_usage', vec_schema_usage) ORDER BY rolname), '[]'::jsonb)
       FROM logins),
   'vec_anchors', (SELECT coalesce(pg_catalog.jsonb_agg(
      pg_catalog.jsonb_build_array(n.nspname, r.rolname) ORDER BY n.nspname), '[]'::jsonb)
       FROM pg_catalog.pg_namespace n JOIN pg_catalog.pg_roles r ON r.oid=n.nspowner
       WHERE n.nspname IN ('vec_autorizacion_atestada_v3',
         'vec_contratacion_temporal', 'vec_identidad_sesiones_v1')),
   'non_system_schemas', (SELECT coalesce(pg_catalog.jsonb_agg(
       n.nspname || '|' || r.rolname ORDER BY n.nspname), '[]'::jsonb)
       FROM pg_catalog.pg_namespace n JOIN pg_catalog.pg_roles r ON r.oid=n.nspowner
       WHERE n.nspname <> 'public' AND n.nspname <> 'information_schema'
         AND n.nspname !~ '^pg_'),
   'public_relations', (SELECT coalesce(pg_catalog.jsonb_agg(
       c.relkind::text || ':' || c.relname || ':' || r.rolname
       ORDER BY c.relkind::text, c.relname), '[]'::jsonb)
       FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
       JOIN pg_catalog.pg_roles r ON r.oid=c.relowner WHERE n.nspname='public'),
   'public_nonextension_functions', (SELECT coalesce(pg_catalog.jsonb_agg(
       p.proname ORDER BY p.proname), '[]'::jsonb)
       FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
       WHERE n.nspname='public' AND NOT EXISTS (
         SELECT 1 FROM pg_catalog.pg_depend d JOIN pg_catalog.pg_extension e ON e.oid=d.refobjid
          WHERE d.classid='pg_catalog.pg_proc'::regclass AND d.objid=p.oid
            AND d.refclassid='pg_catalog.pg_extension'::regclass
            AND d.deptype='e' AND e.extname='pgcrypto')),
   'extensions', (SELECT coalesce(pg_catalog.jsonb_agg(
       pg_catalog.jsonb_build_array(e.extname, n.nspname) ORDER BY e.extname), '[]'::jsonb)
       FROM pg_catalog.pg_extension e JOIN pg_catalog.pg_namespace n ON n.oid=e.extnamespace),
   'functions', pg_catalog.jsonb_build_object(
       'ad3_core', pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL,
       'identity_read', pg_catalog.to_regprocedure('vec_identidad_sesiones_v1.leer_autenticacion_original_v1(text,text,text)') IS NOT NULL)
 ) FROM db JOIN role_graph ON true JOIN pg_catalog.pg_roles owner ON owner.oid=db.datdba;
