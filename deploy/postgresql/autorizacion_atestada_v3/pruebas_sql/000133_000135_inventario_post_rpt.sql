\set ON_ERROR_STOP on
-- Inventario de solo lectura. Guardar la salida fuera de Git.
-- El canon completo se coteja por linaje; los OID sólo se cotejan localmente.
-- Ejecutar tras la cadena causal admitida y repetir tras AD133/AD135.
BEGIN TRANSACTION ISOLATION LEVEL REPEATABLE READ READ ONLY;
SET LOCAL search_path = pg_catalog, pg_temp;
WITH funciones AS (
 SELECT pg_catalog.jsonb_build_object(
  'firma',p.oid::pg_catalog.regprocedure::pg_catalog.text,
  'sha_prosrc',pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p.prosrc,'UTF8')),'hex'),
  'sha_definicion',pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.pg_get_functiondef(p.oid),'UTF8')),'hex'),
  'owner',pg_catalog.pg_get_userbyid(p.proowner),'language',l.lanname,
  'argtypes',(SELECT pg_catalog.jsonb_agg(pg_catalog.format_type(x,NULL) ORDER BY i)
       FROM pg_catalog.unnest(p.proargtypes::pg_catalog.oid[]) WITH ORDINALITY AS t(x,i)),
  'allargtypes',(SELECT pg_catalog.jsonb_agg(pg_catalog.format_type(x,NULL) ORDER BY i)
       FROM pg_catalog.unnest(p.proallargtypes) WITH ORDINALITY AS t(x,i)),
  'return',pg_catalog.format_type(p.prorettype,NULL),
  'variadic',CASE WHEN p.provariadic=0 THEN NULL ELSE pg_catalog.format_type(p.provariadic,NULL) END,
  'support',CASE WHEN p.prosupport=0 THEN NULL ELSE p.prosupport::pg_catalog.regproc::pg_catalog.text END,
  'pg_proc',pg_catalog.to_jsonb(p)-ARRAY['oid','pronamespace','proowner','prolang','proargtypes','proallargtypes','prorettype','provariadic','prosupport','prosrc','proacl'],
  'acl_effective',coalesce((SELECT pg_catalog.jsonb_agg(pg_catalog.jsonb_build_object(
       'grantee',coalesce(gr.rolname,'PUBLIC'),'grantor',go.rolname,
       'privilege',a.privilege_type,'grantable',a.is_grantable)
       ORDER BY coalesce(gr.rolname,'PUBLIC'),go.rolname,a.privilege_type,a.is_grantable)
       FROM pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) AS a
       LEFT JOIN pg_catalog.pg_roles AS gr ON gr.oid=a.grantee
       LEFT JOIN pg_catalog.pg_roles AS go ON go.oid=a.grantor),'[]'::pg_catalog.jsonb),
  'pg_depend',coalesce((SELECT pg_catalog.jsonb_agg(pg_catalog.jsonb_build_object(
       'class',d.classid::pg_catalog.regclass::pg_catalog.text,'objsubid',d.objsubid,
       'refclass',d.refclassid::pg_catalog.regclass::pg_catalog.text,'refobjsubid',d.refobjsubid,
       'deptype',d.deptype,'target',pg_catalog.to_jsonb(i))
       ORDER BY d.classid::pg_catalog.regclass::pg_catalog.text,d.objsubid,
          d.refclassid::pg_catalog.regclass::pg_catalog.text,d.refobjsubid,d.deptype,i.identity)
       FROM pg_catalog.pg_depend AS d CROSS JOIN LATERAL
          pg_catalog.pg_identify_object(d.refclassid,d.refobjid,d.refobjsubid) AS i
       WHERE d.classid='pg_catalog.pg_proc'::pg_catalog.regclass AND d.objid=p.oid),'[]'::pg_catalog.jsonb),
  'pg_shdepend',coalesce((SELECT pg_catalog.jsonb_agg(pg_catalog.jsonb_build_object(
       'class',d.classid::pg_catalog.regclass::pg_catalog.text,'objsubid',d.objsubid,
       'refclass',d.refclassid::pg_catalog.regclass::pg_catalog.text,'refobjsubid',0,
       'deptype',d.deptype,'target',pg_catalog.to_jsonb(i))
       ORDER BY d.classid::pg_catalog.regclass::pg_catalog.text,d.objsubid,
          d.refclassid::pg_catalog.regclass::pg_catalog.text,d.deptype,i.identity)
       FROM pg_catalog.pg_shdepend AS d CROSS JOIN LATERAL
          pg_catalog.pg_identify_object(d.refclassid,d.refobjid,0) AS i
       WHERE d.classid='pg_catalog.pg_proc'::pg_catalog.regclass AND d.objid=p.oid
          AND d.dbid=(SELECT oid FROM pg_catalog.pg_database WHERE datname=pg_catalog.current_database())),
       '[]'::pg_catalog.jsonb)
 ) AS ficha FROM pg_catalog.pg_proc AS p
 JOIN pg_catalog.pg_language AS l ON l.oid=p.prolang
 WHERE p.pronamespace=pg_catalog.to_regnamespace('vec_autorizacion_atestada_v3')
), frontera AS (
 SELECT pg_catalog.jsonb_build_object(
  'audiencias',(SELECT pg_catalog.jsonb_build_object(
    'definicion',pg_catalog.pg_get_constraintdef(c.oid,true),
    'metadatos',pg_catalog.to_jsonb(c)-ARRAY['oid','connamespace','conrelid','conbin'])
    FROM pg_catalog.pg_constraint AS c
    WHERE c.conrelid=pg_catalog.to_regclass('vec_autorizacion_atestada_v3.clave_capacidad_version')
       AND c.conname='clave_capacidad_version_audiencia_consumo_check'),
  'tipos',(SELECT pg_catalog.jsonb_agg(pg_catalog.jsonb_build_object(
    'nombre',t.typname,'propietario',pg_catalog.pg_get_userbyid(t.typowner),
    'metadatos',pg_catalog.to_jsonb(t)-ARRAY['oid','typnamespace','typowner','typrelid','typarray','typacl'],
    'atributos',coalesce((SELECT pg_catalog.jsonb_agg(pg_catalog.jsonb_build_object(
       'metadatos',pg_catalog.to_jsonb(a)-ARRAY['attrelid','atttypid','attcollation'],
       'tipo',pg_catalog.format_type(a.atttypid,a.atttypmod),
       'collation',CASE WHEN a.attcollation=0 THEN NULL ELSE a.attcollation::pg_catalog.regcollation::pg_catalog.text END)
       ORDER BY a.attnum) FROM pg_catalog.pg_attribute AS a
       WHERE a.attrelid=t.typrelid AND a.attnum>0),'[]'::pg_catalog.jsonb),
    'acl',coalesce((SELECT pg_catalog.jsonb_agg(pg_catalog.jsonb_build_object(
       'grantee',coalesce(gr.rolname,'PUBLIC'),'grantor',go.rolname,
       'privilege',a.privilege_type,'grantable',a.is_grantable)
       ORDER BY coalesce(gr.rolname,'PUBLIC'),go.rolname,a.privilege_type,a.is_grantable)
       FROM pg_catalog.aclexplode(coalesce(t.typacl,pg_catalog.acldefault('T',t.typowner))) AS a
       LEFT JOIN pg_catalog.pg_roles AS gr ON gr.oid=a.grantee
       LEFT JOIN pg_catalog.pg_roles AS go ON go.oid=a.grantor),'[]'::pg_catalog.jsonb)
    ) ORDER BY t.typname) FROM pg_catalog.pg_type AS t
    WHERE t.typnamespace=pg_catalog.to_regnamespace('vec_autorizacion_atestada_v3') AND t.typtype='c'),
  'membresias',coalesce((SELECT pg_catalog.jsonb_agg(pg_catalog.jsonb_build_object(
    'rol',g.rolname,'miembro',m.rolname,'otorgante',r.rolname,
    'admin',a.admin_option,'inherit',a.inherit_option,'set',a.set_option)
    ORDER BY g.rolname,m.rolname,r.rolname)
    FROM pg_catalog.pg_auth_members AS a
    JOIN pg_catalog.pg_roles AS g ON g.oid=a.roleid
    JOIN pg_catalog.pg_roles AS m ON m.oid=a.member
    JOIN pg_catalog.pg_roles AS r ON r.oid=a.grantor
    WHERE pg_catalog.left(g.rolname,4)='vec_' OR pg_catalog.left(m.rolname,4)='vec_'),
    '[]'::pg_catalog.jsonb),
  'roles',(SELECT pg_catalog.jsonb_agg(pg_catalog.to_jsonb(r)-ARRAY['oid','rolpassword'] ORDER BY r.rolname)
    FROM pg_catalog.pg_roles AS r WHERE pg_catalog.left(r.rolname,4)='vec_'),
  'public_temp',(SELECT coalesce(pg_catalog.bool_or(a.grantee=0 AND a.privilege_type='TEMPORARY'),false)
    FROM pg_catalog.pg_database AS d LEFT JOIN LATERAL
       pg_catalog.aclexplode(coalesce(d.datacl,pg_catalog.acldefault('d',d.datdba))) AS a ON true
    WHERE d.datname=pg_catalog.current_database())
 ) AS ficha
)
SELECT pg_catalog.jsonb_build_object(
 'matriz',(SELECT pg_catalog.jsonb_agg(ficha ORDER BY ficha->>'firma') FROM funciones),
 'frontera',(SELECT ficha FROM frontera),
 'catalogo_local',(SELECT pg_catalog.jsonb_agg(pg_catalog.jsonb_build_object(
     'pg_proc',pg_catalog.to_jsonb(p),'definicion',pg_catalog.pg_get_functiondef(p.oid)) ORDER BY p.oid)
    FROM pg_catalog.pg_proc AS p
    WHERE p.pronamespace=pg_catalog.to_regnamespace('vec_autorizacion_atestada_v3'))
) AS inventario_completo;
ROLLBACK;
