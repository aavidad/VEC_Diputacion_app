\set ON_ERROR_STOP on
SET search_path = pg_catalog;
WITH RECURSIVE grupos(nombre) AS (VALUES
 ('vec_identidad_sesiones_v1_registrador'),
 ('vec_identidad_sesiones_v1_revalidador'),
 ('vec_contexto_actor_v1_runtime'),
 ('vec_autorizacion_fuente'),
 ('vec_autorizacion_registro'),
 ('vec_autorizacion_motivos_evaluador'),
 ('vec_dietas_ejecutor'),
 ('vec_dietas_registrador_frontera')
), rutas AS (
 SELECT g.rolname AS grupo, g.oid AS grupo_oid, m.member AS miembro_oid,
        ARRAY[g.oid,m.member] AS oids,
        ARRAY[g.rolname::text,r.rolname::text] AS camino,
        m.inherit_option AS hereda, m.set_option AS puede_set
 FROM grupos x JOIN pg_roles g ON g.rolname=x.nombre
 JOIN pg_auth_members m ON m.roleid=g.oid
 JOIN pg_roles r ON r.oid=m.member
 UNION ALL
 SELECT r.grupo,r.grupo_oid,m.member,r.oids||m.member,
        r.camino||miembro.rolname::text,
        r.hereda AND m.inherit_option,r.puede_set AND m.set_option
 FROM rutas r JOIN pg_auth_members m ON m.roleid=r.miembro_oid
 JOIN pg_roles miembro ON miembro.oid=m.member
 WHERE NOT m.member=ANY(r.oids)
)
SELECT jsonb_pretty(jsonb_build_object(
 'capturado_en', clock_timestamp(),
 'roles', (SELECT coalesce(jsonb_agg(jsonb_build_object(
     'nombre',rolname,'login',rolcanlogin,'super',rolsuper,
     'createdb',rolcreatedb,'createrole',rolcreaterole,
     'inherit',rolinherit,'replication',rolreplication,'bypassrls',rolbypassrls
   ) ORDER BY rolname),'[]'::jsonb)
   FROM pg_roles WHERE rolname ~ '^vec_dietas_r1d_.*_desarrollo$'),
 'membresias', (SELECT coalesce(jsonb_agg(jsonb_build_object(
     'miembro',m.rolname,'rol',r.rolname,'admin',a.admin_option,
     'inherit',a.inherit_option,'set',a.set_option
   ) ORDER BY m.rolname,r.rolname),'[]'::jsonb)
   FROM pg_auth_members a JOIN pg_roles m ON m.oid=a.member
   JOIN pg_roles r ON r.oid=a.roleid
   WHERE m.rolname ~ '^vec_dietas_r1d_.*_desarrollo$'
      OR r.rolname ~ '^vec_dietas_r1d_.*_desarrollo$'),
 'sesiones', (SELECT coalesce(jsonb_agg(jsonb_build_object(
     'usuario',usename,'pid',pid,'inicio',backend_start,'estado',state
   ) ORDER BY usename,pid),'[]'::jsonb)
   FROM pg_stat_activity WHERE usename ~ '^vec_dietas_r1d_.*_desarrollo$'),
 'grupos_tecnicos', (SELECT coalesce(jsonb_agg(jsonb_build_object(
     'nombre',x.nombre,'existe',g.oid IS NOT NULL
   ) ORDER BY x.nombre),'[]'::jsonb)
   FROM grupos x LEFT JOIN pg_roles g ON g.rolname=x.nombre),
 'rutas_grupos', (SELECT coalesce(jsonb_agg(jsonb_build_object(
     'grupo',r.grupo,'camino',r.camino,'miembro',miembro.rolname,
     'login',miembro.rolcanlogin,'hereda_aristas',r.hereda,
     'set_aristas',r.puede_set,
     'uso_efectivo',pg_has_role(miembro.oid,r.grupo_oid,'USAGE'),
     'set_efectivo',pg_has_role(miembro.oid,r.grupo_oid,'SET')
   ) ORDER BY r.grupo,r.camino),'[]'::jsonb)
   FROM rutas r JOIN pg_roles miembro ON miembro.oid=r.miembro_oid),
 'login_con_grupo', (SELECT coalesce(jsonb_agg(jsonb_build_object(
     'grupo',g.rolname,'login',l.rolname,
     'miembro',pg_has_role(l.oid,g.oid,'MEMBER'),
     'uso',pg_has_role(l.oid,g.oid,'USAGE'),
     'set',pg_has_role(l.oid,g.oid,'SET')
   ) ORDER BY g.rolname,l.rolname),'[]'::jsonb)
   FROM grupos x JOIN pg_roles g ON g.rolname=x.nombre
   JOIN pg_roles l ON l.rolcanlogin AND NOT l.rolsuper AND l.oid<>g.oid
   WHERE pg_has_role(l.oid,g.oid,'MEMBER')),
 'asignaciones', (SELECT coalesce(jsonb_agg(jsonb_build_object(
     'ref',a.asignacion_ref,'version',a.version,'estado',a.documento->>'estado',
     'huella',a.huella_sha256,'perfil',a.perfil_activo_ref,
     'puntero',p.asignacion_ref IS NOT NULL,'actualizada_por',p.actualizada_por,
     'acto_ref',p.acto_ref
   ) ORDER BY a.version),'[]'::jsonb)
   FROM vec_autorizacion.asignacion_perfil a
   JOIN vec_autorizacion.version_rol r ON r.version_rol_ref=a.version_rol_ref
   LEFT JOIN vec_autorizacion.asignacion_perfil_actual p ON p.asignacion_ref=a.asignacion_ref
   WHERE r.rol_id='dietas_r1d_provisional')
));
