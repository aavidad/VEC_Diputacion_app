\set ON_ERROR_STOP on
\pset tuples_only on
\pset format unaligned
SET timezone='UTC';
SET search_path=pg_catalog,pg_temp;
BEGIN ISOLATION LEVEL REPEATABLE READ READ ONLY;
SELECT 'roles|'||encode(sha256(convert_to(coalesce(jsonb_agg(to_jsonb(r) ORDER BY rolname)::text,'[]'),'UTF8')),'hex')
FROM pg_roles r;
SELECT 'membresias|'||encode(sha256(convert_to(coalesce(jsonb_agg(to_jsonb(m) ORDER BY roleid,member,grantor)::text,'[]'),'UTF8')),'hex')
FROM pg_auth_members m;
SELECT 'base|'||encode(sha256(convert_to(jsonb_build_object('owner',datdba,'acl',datacl,'encoding',encoding,'collate',datcollate,'ctype',datctype)::text,'UTF8')),'hex')
FROM pg_database WHERE datname=current_database();
SELECT 'schema|'||encode(sha256(convert_to(coalesce(jsonb_agg(to_jsonb(n) ORDER BY nspname)::text,'[]'),'UTF8')),'hex')
FROM pg_namespace n WHERE nspname LIKE 'vec_%';
SELECT 'funciones|'||encode(sha256(convert_to(coalesce(jsonb_agg(jsonb_build_object('nombre',n.nspname||'.'||p.proname,'args',pg_get_function_identity_arguments(p.oid),'def',pg_get_functiondef(p.oid),'acl',p.proacl,'owner',p.proowner) ORDER BY n.nspname,p.proname,pg_get_function_identity_arguments(p.oid))::text,'[]'),'UTF8')),'hex')
FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname LIKE 'vec_%' AND p.prokind IN ('f','p');
SELECT 'relaciones|'||encode(sha256(convert_to(coalesce(jsonb_agg(jsonb_build_object('schema',n.nspname,'nombre',c.relname,'owner',c.relowner,'kind',c.relkind,'acl',c.relacl,'rls',c.relrowsecurity,'force',c.relforcerowsecurity,'opts',c.reloptions) ORDER BY n.nspname,c.relname)::text,'[]'),'UTF8')),'hex')
FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname LIKE 'vec_%';
SELECT 'columnas|'||encode(sha256(convert_to(coalesce(jsonb_agg(jsonb_build_object('schema',n.nspname,'tabla',c.relname,'nombre',a.attname,'numero',a.attnum,'tipo',format_type(a.atttypid,a.atttypmod),'notnull',a.attnotnull,'default',pg_get_expr(d.adbin,d.adrelid),'acl',a.attacl) ORDER BY n.nspname,c.relname,a.attnum)::text,'[]'),'UTF8')),'hex')
FROM pg_attribute a JOIN pg_class c ON c.oid=a.attrelid JOIN pg_namespace n ON n.oid=c.relnamespace LEFT JOIN pg_attrdef d ON d.adrelid=c.oid AND d.adnum=a.attnum
WHERE n.nspname LIKE 'vec_%' AND a.attnum>0 AND NOT a.attisdropped;
SELECT 'restricciones|'||encode(sha256(convert_to(coalesce(jsonb_agg(jsonb_build_object('schema',n.nspname,'catalogo',to_jsonb(c),'def',pg_get_constraintdef(c.oid)) ORDER BY n.nspname,c.conname,c.conrelid)::text,'[]'),'UTF8')),'hex')
FROM pg_constraint c JOIN pg_namespace n ON n.oid=c.connamespace WHERE n.nspname LIKE 'vec_%';
SELECT 'politicas|'||encode(sha256(convert_to(coalesce(jsonb_agg(to_jsonb(p) ORDER BY schemaname,tablename,policyname)::text,'[]'),'UTF8')),'hex')
FROM pg_policies p WHERE schemaname LIKE 'vec_%';
SELECT 'triggers|'||encode(sha256(convert_to(coalesce(jsonb_agg(jsonb_build_object('tabla',c.oid::regclass::text,'nombre',t.tgname,'enabled',t.tgenabled,'interno',t.tgisinternal,'def',pg_get_triggerdef(t.oid)) ORDER BY c.oid::regclass::text,t.tgname)::text,'[]'),'UTF8')),'hex')
FROM pg_trigger t JOIN pg_class c ON c.oid=t.tgrelid JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname LIKE 'vec_%';
SELECT 'reglas|'||encode(sha256(convert_to(coalesce(jsonb_agg(jsonb_build_object('tabla',c.oid::regclass::text,'nombre',r.rulename,'enabled',r.ev_enabled,'def',pg_get_ruledef(r.oid)) ORDER BY c.oid::regclass::text,r.rulename)::text,'[]'),'UTF8')),'hex')
FROM pg_rewrite r JOIN pg_class c ON c.oid=r.ev_class JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname LIKE 'vec_%';
SELECT 'tipos|'||encode(sha256(convert_to(coalesce(jsonb_agg(jsonb_build_object('schema',n.nspname,'tipo',t.typname,'owner',t.typowner,'acl',t.typacl,'kind',t.typtype,'categoria',t.typcategory,'base',t.typbasetype,'notnull',t.typnotnull,'default',t.typdefault,'relacion',t.typrelid,'elemento',t.typelem) ORDER BY n.nspname,t.typname)::text,'[]'),'UTF8')),'hex')
FROM pg_type t JOIN pg_namespace n ON n.oid=t.typnamespace WHERE n.nspname LIKE 'vec_%';
SELECT 'enums|'||encode(sha256(convert_to(coalesce(jsonb_agg(jsonb_build_object('tipo',e.enumtypid::regtype::text,'orden',e.enumsortorder,'valor',e.enumlabel) ORDER BY e.enumtypid::regtype::text,e.enumsortorder)::text,'[]'),'UTF8')),'hex')
FROM pg_enum e JOIN pg_type t ON t.oid=e.enumtypid JOIN pg_namespace n ON n.oid=t.typnamespace WHERE n.nspname LIKE 'vec_%';
SELECT 'acl_predeterminadas|'||encode(sha256(convert_to(coalesce(jsonb_agg(to_jsonb(d) ORDER BY defaclrole,defaclnamespace,defaclobjtype)::text,'[]'),'UTF8')),'hex')
FROM pg_default_acl d;
SELECT 'vistas|'||encode(sha256(convert_to(coalesce(jsonb_agg(jsonb_build_object('schema',n.nspname,'nombre',c.relname,'def',pg_get_viewdef(c.oid,false)) ORDER BY n.nspname,c.relname)::text,'[]'),'UTF8')),'hex')
FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname LIKE 'vec_%' AND c.relkind IN ('v','m');
SELECT 'indices|'||encode(sha256(convert_to(coalesce(jsonb_agg(jsonb_build_object('schema',n.nspname,'nombre',c.relname,'def',pg_get_indexdef(c.oid),'valido',i.indisvalid,'ready',i.indisready) ORDER BY n.nspname,c.relname)::text,'[]'),'UTF8')),'hex')
FROM pg_index i JOIN pg_class c ON c.oid=i.indexrelid JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname LIKE 'vec_%';
SELECT 'secuencias|'||encode(sha256(convert_to(coalesce(jsonb_agg(to_jsonb(s) ORDER BY schemaname,sequencename)::text,'[]'),'UTF8')),'hex')
FROM pg_sequences s WHERE schemaname LIKE 'vec_%';
SELECT format('SELECT %L||last_value||''|''||is_called FROM %I.%I;', 'secuencia_estado|'||n.nspname||'.'||c.relname||'|',n.nspname,c.relname)
FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname LIKE 'vec_%' AND c.relkind='S' ORDER BY n.nspname,c.relname
\gexec
SELECT format('SELECT %L||count(*)||''|''||encode(sha256(convert_to(coalesce(string_agg(j,E''\n'' ORDER BY j),''''),''UTF8'')),''hex'') FROM (SELECT to_jsonb(t)::text j FROM %I.%I t) s;', 'datos|'||n.nspname||'.'||c.relname||'|',n.nspname,c.relname)
FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname LIKE 'vec_%' AND c.relkind IN ('r','p') ORDER BY n.nspname,c.relname
\gexec
COMMIT;
