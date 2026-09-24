\set ON_ERROR_STOP on
SET search_path=pg_catalog;
WITH grupos AS (
 SELECT oid,rolname,rolcanlogin,rolinherit,rolsuper,rolcreatedb,rolcreaterole,
        rolreplication,rolbypassrls
 FROM pg_roles WHERE rolname IN ('vec_personal_d7_ejecutor','vec_personal_registrador_frontera')
), acl AS (
 SELECT g.rolname,'database:'||d.datname::text objeto,a.privilege_type::text privilegio,a.is_grantable grantable
 FROM pg_database d,LATERAL aclexplode(coalesce(d.datacl,acldefault('d',d.datdba))) a JOIN grupos g ON g.oid=a.grantee
 UNION ALL
 SELECT g.rolname,'schema:'||n.nspname,a.privilege_type::text,a.is_grantable
 FROM pg_namespace n,LATERAL aclexplode(coalesce(n.nspacl,acldefault('n',n.nspowner))) a JOIN grupos g ON g.oid=a.grantee
 UNION ALL
 SELECT g.rolname,'relation:'||c.oid::regclass::text,a.privilege_type::text,a.is_grantable
 FROM pg_class c,LATERAL aclexplode(coalesce(c.relacl,acldefault((CASE WHEN c.relkind='S' THEN 'S' ELSE 'r' END)::"char",c.relowner))) a JOIN grupos g ON g.oid=a.grantee
 UNION ALL
 SELECT g.rolname,'column:'||c.oid::regclass::text||'.'||x.attname,a.privilege_type::text,a.is_grantable
 FROM pg_attribute x JOIN pg_class c ON c.oid=x.attrelid,LATERAL aclexplode(x.attacl) a JOIN grupos g ON g.oid=a.grantee WHERE x.attnum>0
 UNION ALL
 SELECT g.rolname,'function:'||p.oid::regprocedure::text,a.privilege_type::text,a.is_grantable
 FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a JOIN grupos g ON g.oid=a.grantee
 UNION ALL
 SELECT g.rolname,'type:'||t.oid::regtype::text,a.privilege_type::text,a.is_grantable
 FROM pg_type t,LATERAL aclexplode(coalesce(t.typacl,acldefault('T',t.typowner))) a JOIN grupos g ON g.oid=a.grantee
)
SELECT jsonb_build_object(
 'f4', (SELECT jsonb_build_object(
    'cuentas', (SELECT count(*) FROM pg_roles WHERE rolname ~ '^vec_dietas_r1d_.*_desarrollo$'),
    'login', (SELECT count(*) FROM pg_roles WHERE rolname ~ '^vec_dietas_r1d_.*_desarrollo$' AND rolcanlogin),
    'puntero', (SELECT coalesce(jsonb_agg(jsonb_build_object('ref',p.asignacion_ref,'version',a.version,'estado',a.documento->>'estado')), '[]'::jsonb)
      FROM vec_autorizacion.asignacion_perfil_actual p JOIN vec_autorizacion.asignacion_perfil a USING (asignacion_ref)
      WHERE a.version_rol_ref='rol:dietas_r1d_provisional:v1'))),
 'grupos', (SELECT coalesce(jsonb_object_agg(g.rolname,jsonb_build_object(
    'login',g.rolcanlogin,'inherit',g.rolinherit,'super',g.rolsuper,
    'createdb',g.rolcreatedb,'createrole',g.rolcreaterole,
    'replication',g.rolreplication,'bypassrls',g.rolbypassrls,
    'membresias_salida',(SELECT count(*) FROM pg_auth_members m WHERE m.member=g.oid),
    'miembros_entrantes',(SELECT coalesce(jsonb_agg(r.rolname ORDER BY r.rolname),'[]'::jsonb)
      FROM pg_auth_members m JOIN pg_roles r ON r.oid=m.member WHERE m.roleid=g.oid),
    'propiedades',(SELECT count(*) FROM pg_shdepend d WHERE d.refobjid=g.oid AND d.deptype='o'),
    'acl',(SELECT coalesce(jsonb_agg(jsonb_build_object('objeto',a.objeto,'privilegio',a.privilegio,'grantable',a.grantable)
      ORDER BY a.objeto,a.privilegio), '[]'::jsonb) FROM acl a WHERE a.rolname=g.rolname)
  )), '{}'::jsonb) FROM grupos g),
 'cuentas', jsonb_build_object(
    'vec_personal_d7_asignacion', (SELECT jsonb_build_object('login',r.rolcanlogin,
      'miembros',(SELECT count(*) FROM pg_auth_members m WHERE m.member=r.oid))
      FROM pg_roles r WHERE r.rolname='vec_personal_d7_asignacion'),
    'vec_personal_d7_auditoria_frontera', (SELECT jsonb_build_object('login',r.rolcanlogin,
      'miembros',(SELECT count(*) FROM pg_auth_members m WHERE m.member=r.oid))
      FROM pg_roles r WHERE r.rolname='vec_personal_d7_auditoria_frontera'))
)::text;
