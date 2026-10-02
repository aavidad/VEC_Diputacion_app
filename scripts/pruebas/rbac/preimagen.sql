\set ON_ERROR_STOP on
BEGIN ISOLATION LEVEL REPEATABLE READ READ ONLY;
SET LOCAL statement_timeout = '120s';
SET LOCAL lock_timeout = '5s';
SELECT 'server', current_setting('server_version_num');
SELECT 'role', rolname, rolsuper, rolinherit, rolcreaterole, rolcreatedb,
       rolcanlogin, rolreplication, rolbypassrls
FROM pg_roles WHERE rolname !~ '^pg_' ORDER BY rolname;
SELECT 'membership', parent.rolname, member.rolname, m.admin_option,
       m.inherit_option, m.set_option
FROM pg_auth_members m
JOIN pg_roles parent ON parent.oid=m.roleid
JOIN pg_roles member ON member.oid=m.member ORDER BY 2,3;
SELECT 'schema', n.nspname, pg_get_userbyid(n.nspowner), n.nspacl::text
FROM pg_namespace n WHERE n.nspname LIKE 'vec_%' ORDER BY 2;
SELECT 'function', n.nspname, p.proname,
       pg_get_function_identity_arguments(p.oid), pg_get_userbyid(p.proowner),
       p.prosecdef, p.proacl::text,
       encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex')
FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
WHERE n.nspname LIKE 'vec_%' AND p.prokind IN ('f','p') ORDER BY 2,3,4;
SELECT 'table_acl', n.nspname,c.relname,pg_get_userbyid(c.relowner),
       c.relrowsecurity,c.relforcerowsecurity,c.relacl::text
FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace
WHERE n.nspname LIKE 'vec_%' AND c.relkind IN ('r','p') ORDER BY 2,3;
SELECT format(
  'SELECT %L,%L,count(*),encode(sha256(convert_to(coalesce(string_agg(h,%L ORDER BY h),%L),%L)),%L) FROM (SELECT encode(sha256(convert_to(to_jsonb(t)::text,%L)),%L) h FROM %I.%I t) hashed;',
  'data',n.nspname||'.'||c.relname,'','','UTF8','hex','UTF8','hex',n.nspname,c.relname)
FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace
WHERE n.nspname LIKE 'vec_%' AND c.relkind IN ('r','p') ORDER BY n.nspname,c.relname
\gexec
COMMIT;
