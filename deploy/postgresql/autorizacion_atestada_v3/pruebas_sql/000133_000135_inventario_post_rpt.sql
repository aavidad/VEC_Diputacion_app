\set ON_ERROR_STOP on
-- Inventario de solo lectura para dos clones PG18 independientes.
-- Ejecutar tras H6/AD132 y el prefijo exacto de #222; repetir tras AD133 nueva.
-- Guardar la salida fuera de Git. Una huella aislada no sustituye la comparación
-- del JSON entero de cada linaje ni el cotejo local de OID antes/después.
BEGIN TRANSACTION ISOLATION LEVEL REPEATABLE READ READ ONLY;
SET LOCAL search_path = pg_catalog, pg_temp;
WITH funciones AS (
 SELECT pg_catalog.jsonb_build_object(
  'firma',p.oid::pg_catalog.regprocedure::pg_catalog.text,
  'oid_local',p.oid,
  'sha_definicion',pg_catalog.encode(pg_catalog.sha256(
      pg_catalog.convert_to(pg_catalog.pg_get_functiondef(p.oid),'UTF8')),'hex'),
  'sha_cuerpo',pg_catalog.encode(pg_catalog.sha256(
      pg_catalog.convert_to(p.prosrc,'UTF8')),'hex'),
  'propietario',pg_catalog.pg_get_userbyid(p.proowner),
  'lenguaje',l.lanname,
  'configuracion',p.proconfig,
  'metadatos',pg_catalog.to_jsonb(p) - ARRAY[
      'oid','pronamespace','proowner','prolang','prosrc','proacl'],
  'acl',pg_catalog.coalesce((
      SELECT pg_catalog.jsonb_agg(pg_catalog.jsonb_build_object(
        'grantee',pg_catalog.coalesce(gr.rolname,'PUBLIC'),
        'grantor',go.rolname,'privilege',a.privilege_type,
        'grantable',a.is_grantable)
        ORDER BY pg_catalog.coalesce(gr.rolname,'PUBLIC'),
                 go.rolname,a.privilege_type,a.is_grantable)
      FROM pg_catalog.aclexplode(pg_catalog.coalesce(
           p.proacl,pg_catalog.acldefault('f',p.proowner))) AS a
      LEFT JOIN pg_catalog.pg_roles AS gr ON gr.oid=a.grantee
      LEFT JOIN pg_catalog.pg_roles AS go ON go.oid=a.grantor),'[]'::pg_catalog.jsonb),
  'dependencias_locales',pg_catalog.coalesce((
      SELECT pg_catalog.jsonb_agg(pg_catalog.jsonb_build_object(
        'class',d.classid::pg_catalog.regclass::pg_catalog.text,
        'refclass',d.refclassid::pg_catalog.regclass::pg_catalog.text,
        'deptype',d.deptype,'target',pg_catalog.to_jsonb(i))
        ORDER BY d.classid,d.refclassid,d.deptype,i.identity)
      FROM pg_catalog.pg_depend AS d
      CROSS JOIN LATERAL pg_catalog.pg_identify_object(
          d.refclassid,d.refobjid,d.refobjsubid) AS i
      WHERE d.classid='pg_catalog.pg_proc'::pg_catalog.regclass
        AND d.objid=p.oid),'[]'::pg_catalog.jsonb),
  'dependencias_compartidas',pg_catalog.coalesce((
      SELECT pg_catalog.jsonb_agg(pg_catalog.jsonb_build_object(
        'refclass',d.refclassid::pg_catalog.regclass::pg_catalog.text,
        'deptype',d.deptype,'target',pg_catalog.to_jsonb(i))
        ORDER BY d.refclassid,d.deptype,i.identity)
      FROM pg_catalog.pg_shdepend AS d
      CROSS JOIN LATERAL pg_catalog.pg_identify_object(
          d.refclassid,d.refobjid,0) AS i
      WHERE d.classid='pg_catalog.pg_proc'::pg_catalog.regclass
        AND d.dbid=(SELECT oid FROM pg_catalog.pg_database
                    WHERE datname=pg_catalog.current_database())
        AND d.objid=p.oid),'[]'::pg_catalog.jsonb)
 ) AS ficha
 FROM pg_catalog.pg_proc AS p
 JOIN pg_catalog.pg_language AS l ON l.oid=p.prolang
 WHERE p.pronamespace=pg_catalog.to_regnamespace(
       'vec_autorizacion_atestada_v3')
), audiencia AS (
 SELECT pg_catalog.jsonb_build_object(
   'definicion',pg_catalog.pg_get_constraintdef(c.oid,true),
   'metadatos',pg_catalog.to_jsonb(c) - ARRAY[
       'oid','connamespace','conrelid','conbin'],
   'tabla',c.conrelid::pg_catalog.regclass::pg_catalog.text,
   'dependencias',pg_catalog.coalesce((
     SELECT pg_catalog.jsonb_agg(pg_catalog.jsonb_build_object(
       'refclass',d.refclassid::pg_catalog.regclass::pg_catalog.text,
       'deptype',d.deptype,'target',pg_catalog.to_jsonb(i))
       ORDER BY d.refclassid,d.deptype,i.identity)
     FROM pg_catalog.pg_depend AS d
     CROSS JOIN LATERAL pg_catalog.pg_identify_object(
       d.refclassid,d.refobjid,d.refobjsubid) AS i
     WHERE d.classid='pg_catalog.pg_constraint'::pg_catalog.regclass
       AND d.objid=c.oid),'[]'::pg_catalog.jsonb)) AS ficha
 FROM pg_catalog.pg_constraint AS c
 WHERE c.conrelid=pg_catalog.to_regclass(
       'vec_autorizacion_atestada_v3.clave_capacidad_version')
   AND c.conname='clave_capacidad_version_audiencia_consumo_check'
), tipos AS (
 SELECT pg_catalog.jsonb_build_object(
   'nombre',t.oid::pg_catalog.regtype::pg_catalog.text,
   'propietario',pg_catalog.pg_get_userbyid(t.typowner),
   'metadatos',pg_catalog.to_jsonb(t) - ARRAY[
       'oid','typnamespace','typowner','typrelid','typarray','typacl'],
   'acl',pg_catalog.coalesce((
     SELECT pg_catalog.jsonb_agg(pg_catalog.jsonb_build_object(
       'grantee',pg_catalog.coalesce(gr.rolname,'PUBLIC'),
       'grantor',go.rolname,'privilege',a.privilege_type,
       'grantable',a.is_grantable)
       ORDER BY pg_catalog.coalesce(gr.rolname,'PUBLIC'),
                go.rolname,a.privilege_type,a.is_grantable)
     FROM pg_catalog.aclexplode(pg_catalog.coalesce(
          t.typacl,pg_catalog.acldefault('T',t.typowner))) AS a
     LEFT JOIN pg_catalog.pg_roles AS gr ON gr.oid=a.grantee
     LEFT JOIN pg_catalog.pg_roles AS go ON go.oid=a.grantor),'[]'::pg_catalog.jsonb)
 ) AS ficha
 FROM pg_catalog.pg_type AS t
 WHERE t.typnamespace=pg_catalog.to_regnamespace(
       'vec_autorizacion_atestada_v3') AND t.typtype='c'
), membresias AS (
 SELECT pg_catalog.jsonb_build_object(
   'rol',g.rolname,'miembro',m.rolname,
   'admin',a.admin_option,'inherit',a.inherit_option,'set',a.set_option)
   AS ficha
 FROM pg_catalog.pg_auth_members AS a
 JOIN pg_catalog.pg_roles AS g ON g.oid=a.roleid
 JOIN pg_catalog.pg_roles AS m ON m.oid=a.member
 WHERE pg_catalog.left(g.rolname,4)='vec_'
    OR pg_catalog.left(m.rolname,4)='vec_'
), temporal AS (
 SELECT pg_catalog.coalesce(pg_catalog.bool_or(
   a.grantee=0 AND a.privilege_type='TEMPORARY'),false) AS public_temp
 FROM pg_catalog.pg_database AS d
 LEFT JOIN LATERAL pg_catalog.aclexplode(pg_catalog.coalesce(
   d.datacl,pg_catalog.acldefault('d',d.datdba))) AS a ON true
 WHERE d.datname=pg_catalog.current_database()
), inventario AS (
 SELECT pg_catalog.jsonb_build_object(
   'funciones',(SELECT pg_catalog.jsonb_agg(ficha ORDER BY ficha->>'firma')
                FROM funciones),
   'check_audiencia',(SELECT ficha FROM audiencia),
   'tipos_compuestos',(SELECT pg_catalog.jsonb_agg(ficha ORDER BY ficha->>'nombre')
                       FROM tipos),
   'membresias',(SELECT pg_catalog.jsonb_agg(ficha ORDER BY
                  ficha->>'rol',ficha->>'miembro') FROM membresias),
   'public_temp',(SELECT public_temp FROM temporal)) AS datos
)
SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
         datos::pg_catalog.text,'UTF8')),'hex') AS sha_inventario_local,
       datos AS inventario_completo
FROM inventario;
ROLLBACK;
