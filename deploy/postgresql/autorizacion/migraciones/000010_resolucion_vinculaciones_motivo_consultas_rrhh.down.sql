-- Safe-down M1.3: retira solo las dos resoluciones nominales.
BEGIN;

SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock_shared(pg_catalog.hashtextextended(
    'vec_autorizacion:rol-motivos-rrhh-resolutor:v1', 0));
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_autorizacion:migracion:vinculaciones-motivo-rrhh:000008', 0));
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_autorizacion:migracion:vinculaciones-motivo-rrhh:000009', 0));
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_autorizacion:migracion:vinculaciones-motivo-rrhh:000010', 0));

SET LOCAL ROLE vec_autorizacion_propietario;

LOCK TABLE
    vec_autorizacion.motivo_v2_evento_origen,
    vec_autorizacion.motivo_v2_catalogo_publicado,
    vec_autorizacion.motivo_v2_entrada,
    vec_autorizacion.motivo_v2_retirada,
    vec_autorizacion.motivo_v2_checkpoint_origen,
    vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1,
    vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1,
    vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1
    IN ACCESS SHARE MODE;

DO $prevalidacion$
DECLARE
  rol oid := 'vec_autorizacion_motivos_rrhh_resolutor'::regrole;
  propietario oid := 'vec_autorizacion_propietario'::regrole;
  tablas regclass[] := ARRAY[
    'vec_autorizacion.motivo_v2_evento_origen'::regclass,
    'vec_autorizacion.motivo_v2_catalogo_publicado'::regclass,
    'vec_autorizacion.motivo_v2_entrada'::regclass,
    'vec_autorizacion.motivo_v2_retirada'::regclass,
    'vec_autorizacion.motivo_v2_checkpoint_origen'::regclass,
    'vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1'::regclass,
    'vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1'::regclass,
    'vec_autorizacion.vinculacion_motivo_consulta_rrhh_evento_v1'::regclass];
  nombres_funciones name[] := ARRAY[
    'bloquear_mutacion_vinculacion_motivo_rrhh_v1',
    'validar_avance_vinculacion_motivo_rrhh_v1',
    'bloquear_mutacion_vinculacion_motivo_rrhh_evento_v1',
    'validar_insercion_vinculacion_motivo_rrhh_evento_v1',
    'registrar_publicacion_vinculacion_motivo_consulta_rrhh_v1',
    'registrar_retirada_vinculacion_motivo_consulta_rrhh_v1',
    'publicar_vinculacion_motivo_cuadro_rrhh_v1',
    'publicar_vinculacion_motivo_detalle_rrhh_v1',
    'retirar_vinculacion_motivo_cuadro_rrhh_v1',
    'retirar_vinculacion_motivo_detalle_rrhh_v1'];
  funciones oid[] := ARRAY[
    'vec_autorizacion.resolver_motivo_cuadro_rrhh_v1(timestamptz)'::regprocedure,
    'vec_autorizacion.resolver_motivo_detalle_rrhh_v1(timestamptz)'::regprocedure];
BEGIN
  IF pg_catalog.current_setting('server_version_num')::integer / 10000 <> 18
     OR pg_catalog.getdatabaseencoding() IS DISTINCT FROM 'UTF8'
     OR NOT EXISTS (
       SELECT 1 FROM pg_catalog.pg_namespace
        WHERE nspname='vec_autorizacion' AND nspowner=propietario)
     OR NOT EXISTS (
       SELECT 1 FROM pg_catalog.pg_roles r WHERE r.oid=rol
         AND NOT r.rolcanlogin AND NOT r.rolsuper AND NOT r.rolcreatedb
         AND NOT r.rolcreaterole AND NOT r.rolinherit AND NOT r.rolreplication
         AND NOT r.rolbypassrls AND r.rolconnlimit=-1
         AND (r.rolpassword IS NULL OR r.rolpassword='********')
         AND r.rolvaliduntil IS NULL AND r.rolconfig IS NULL
         AND pg_catalog.shobj_description(r.oid,'pg_authid')=
             'vec_autorizacion:rol-motivos-rrhh-resolutor:v1')
     OR EXISTS (SELECT 1 FROM pg_catalog.pg_db_role_setting WHERE setrole=rol)
     OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE oid=10 AND rolsuper)
     OR EXISTS (
       SELECT 1 FROM pg_catalog.pg_auth_members m
       LEFT JOIN pg_catalog.pg_roles a ON a.oid=m.member
        WHERE m.member=rol OR m.grantor=rol OR (m.roleid=rol AND
          (m.admin_option OR NOT m.inherit_option OR m.set_option OR m.grantor<>10
           OR NOT a.rolcanlogin OR NOT a.rolinherit OR a.rolsuper OR a.rolcreatedb
           OR a.rolcreaterole OR a.rolreplication OR a.rolbypassrls))) THEN
    RAISE EXCEPTION USING ERRCODE='55000',
      MESSAGE='retirada de resolución RRHH rechazada';
  END IF;

  IF (SELECT pg_catalog.count(*) FROM pg_catalog.pg_class c
       WHERE c.oid=ANY(tablas) AND c.relkind='r' AND c.relpersistence='p'
         AND c.relam=(SELECT oid FROM pg_catalog.pg_am
                       WHERE amname='heap' AND amtype='t')
         AND NOT c.relispartition AND c.relowner=propietario
         AND c.relrowsecurity AND c.relforcerowsecurity)<>8
     OR EXISTS (SELECT 1 FROM pg_catalog.pg_inherits
                 WHERE inhrelid=ANY(tablas) OR inhparent=ANY(tablas))
     OR EXISTS (SELECT 1 FROM pg_catalog.pg_rewrite WHERE ev_class=ANY(tablas))
     OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_policy
          WHERE polrelid=ANY(tablas) AND polname='acceso_propietario_exacto'
            AND polpermissive AND polcmd='*'
            AND polroles=ARRAY[propietario]
            AND pg_catalog.pg_get_expr(polqual,polrelid)=
                '(CURRENT_USER = ''vec_autorizacion_propietario''::name)'
            AND pg_catalog.pg_get_expr(polwithcheck,polrelid)=
                '(CURRENT_USER = ''vec_autorizacion_propietario''::name)')<>8
     OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_policy
          WHERE polrelid=ANY(tablas))<>8
     OR EXISTS (
       SELECT 1 FROM pg_catalog.pg_class c
       CROSS JOIN LATERAL pg_catalog.aclexplode(
         COALESCE(c.relacl,pg_catalog.acldefault('r',c.relowner))) a
        WHERE c.oid=ANY(tablas) AND a.grantee<>c.relowner)
     OR EXISTS (
       SELECT 1 FROM pg_catalog.pg_attribute c
       CROSS JOIN LATERAL pg_catalog.aclexplode(c.attacl) a
        WHERE c.attrelid=ANY(tablas) AND c.attnum>0 AND a.grantee<>propietario)
     OR EXISTS (
       SELECT 1 FROM pg_catalog.pg_type c
       CROSS JOIN LATERAL pg_catalog.aclexplode(c.typacl) a
        WHERE c.typrelid=ANY(tablas) AND a.grantee<>c.typowner)
     OR (SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
          pg_catalog.jsonb_agg(pg_catalog.jsonb_build_array(
            c.oid::regclass::text,a.attnum,a.attname,a.atttypid::regtype::text,
            a.atttypmod,a.attnotnull,a.attidentity,a.attgenerated,
            a.attcollation::regcollation::text,a.attstorage,a.attcompression,
            a.attisdropped,pg_catalog.pg_get_expr(d.adbin,d.adrelid,true))
            ORDER BY c.oid::regclass::text,a.attnum)::text,'UTF8')),'hex')
          FROM pg_catalog.pg_class c
          JOIN pg_catalog.pg_attribute a ON a.attrelid=c.oid AND a.attnum>0
          LEFT JOIN pg_catalog.pg_attrdef d
            ON (d.adrelid,d.adnum)=(a.attrelid,a.attnum)
         WHERE c.oid=ANY(tablas))<>
        '89c15ed702c1897c9e99b4fc5e2f030182cb1324e1a7a65f91d820102a5b83e8'
  THEN
    RAISE EXCEPTION USING ERRCODE='55000',
      MESSAGE='retirada de resolución RRHH rechazada';
  END IF;

  IF (WITH locales AS (SELECT tablas[6:8] AS lista),
      v2(tabla,nombre) AS (VALUES
        (tablas[1],'motivo_v2_evento_coordenadas_unicas'),
        (tablas[2],'motivo_v2_catalogo_evento_fk'),
        (tablas[3],'motivo_v2_entrada_catalogo_fk'),
        (tablas[4],'motivo_v2_retirada_catalogo_fk'),
        (tablas[4],'motivo_v2_retirada_evento_fk'),
        (tablas[2],'motivo_v2_catalogo_referencia_completa_unica')),
      restricciones AS (
        SELECT c.* FROM pg_catalog.pg_constraint c,locales l
         WHERE c.contype<>'n' AND (c.conrelid=ANY(l.lista) OR EXISTS(
           SELECT 1 FROM v2 WHERE (tabla,nombre)=(c.conrelid,c.conname)))),
      claves AS (SELECT oid FROM restricciones WHERE contype='f'),
      disparadores AS (
        SELECT t.*,c.conname AS restriccion_nombre
          FROM pg_catalog.pg_trigger t
          LEFT JOIN pg_catalog.pg_constraint c ON c.oid=t.tgconstraint,locales l
         WHERE t.tgconstraint=ANY(ARRAY(SELECT oid FROM claves))
            OR (t.tgisinternal AND
                (t.tgrelid=ANY(l.lista) OR t.tgconstrrelid=ANY(l.lista)))
            OR (NOT t.tgisinternal AND t.tgrelid=ANY(l.lista))),
      manifiesto AS (
        SELECT pg_catalog.jsonb_build_object(
          'restricciones',(SELECT pg_catalog.jsonb_agg(
            pg_catalog.jsonb_build_array(conrelid::regclass::text,conname,contype,
              pg_catalog.pg_get_constraintdef(oid,true),
              pg_catalog.obj_description(oid,'pg_constraint'),convalidated,
              condeferrable,condeferred,connoinherit)
            ORDER BY conrelid::regclass::text,conname) FROM restricciones),
          'disparadores',(SELECT pg_catalog.jsonb_agg(
            pg_catalog.jsonb_build_array(tgrelid::regclass::text,
              CASE WHEN tgisinternal THEN NULL ELSE tgname END,tgtype,tgenabled,
              tgisinternal,tgfoid::regprocedure::text,restriccion_nombre,
              tgconstrrelid::regclass::text,tgconstrindid::regclass::text,
              tgparentid=0,tgdeferrable,tginitdeferred,tgnargs,tgattr::text,
              pg_catalog.encode(tgargs,'hex'),pg_catalog.pg_get_expr(tgqual,tgrelid),
              tgoldtable,tgnewtable)
            ORDER BY tgrelid::regclass::text,tgisinternal,
              COALESCE(restriccion_nombre,''),tgtype,
              tgfoid::regprocedure::text,COALESCE(tgname,''))
            FROM disparadores)) valor)
      SELECT pg_catalog.encode(pg_catalog.sha256(
        pg_catalog.convert_to(valor::text,'UTF8')),'hex') FROM manifiesto)
      <>'d4a4f60c63d99cd7e3e1164ae542011b086318654cb2c7cfd3f3b47aec5570f4'
  THEN
    RAISE EXCEPTION USING ERRCODE='55000',
      MESSAGE='retirada de resolución RRHH rechazada';
  END IF;

  IF EXISTS (
    WITH esperado(objeto,identidad,expuesta) AS (VALUES
      ('vec_autorizacion.bloquear_mutacion_vinculacion_motivo_rrhh_evento_v1()','7d2c390e2c17c3d4eaca7da98d8c767f40707d81203fef9897be745b929143a7',false),
      ('vec_autorizacion.bloquear_mutacion_vinculacion_motivo_rrhh_v1()','73e9bbb667fa19234325b8b90acd6a8759b4cde136cae94e8b87dd3182317182',false),
      ('vec_autorizacion.publicar_vinculacion_motivo_cuadro_rrhh_v1(text,text,bigint,text,text,text,integer,text,text,timestamp with time zone)','822fd8c60a85e305cd9f87e9d2fd366db6261b0f1344498e93c9346bbfc1181c',true),
      ('vec_autorizacion.publicar_vinculacion_motivo_detalle_rrhh_v1(text,text,bigint,text,text,text,integer,text,text,timestamp with time zone)','025ce8b53f6c2bd9f68f410921b98fb0edbaab9ba2af66426ad543c2abdc4de5',true),
      ('vec_autorizacion.registrar_publicacion_vinculacion_motivo_consulta_rrhh_v1(text,text,text,bigint,text,text,text,integer,text,text,timestamp with time zone)','eebea97900a4cecba6900f9908499b58c047ea6888478f0573c745964c08fa83',false),
      ('vec_autorizacion.registrar_retirada_vinculacion_motivo_consulta_rrhh_v1(text,text,text,bigint,text,text,timestamp with time zone)','7bbb751a6963dd4e503d7607e73d3a5a61be3583f1b36f695c78f610b2fa1c18',false),
      ('vec_autorizacion.retirar_vinculacion_motivo_cuadro_rrhh_v1(text,text,bigint,text,text,timestamp with time zone)','3a3c5460c4eeebf945613990a2589f26085f6c535968a14eebc00ed08373c778',true),
      ('vec_autorizacion.retirar_vinculacion_motivo_detalle_rrhh_v1(text,text,bigint,text,text,timestamp with time zone)','32cc411780f0044ce8262d4367a390cef63c075a6e438dd4700e917606177626',true),
      ('vec_autorizacion.validar_avance_vinculacion_motivo_rrhh_v1()','3dffb1171f1d2e766ec927abeef5e8dc0de6801256845cc2c2fd930597c597a5',false),
      ('vec_autorizacion.validar_insercion_vinculacion_motivo_rrhh_evento_v1()','b1fb4729e54a7d27c584055376db7d9ff738769eaaac68295aa50299d96f71f8',false)
    ), actual AS (
      SELECT p.oid::regprocedure::text,pg_catalog.encode(pg_catalog.sha256(
        pg_catalog.convert_to(pg_catalog.jsonb_build_array(l.lanname,
          p.proowner::regrole::text,p.prokind::text,p.provolatile::text,
          p.proparallel::text,p.prosecdef,p.proleakproof,p.proisstrict,p.proretset,
          p.pronargs,p.pronargdefaults,p.prorettype::regtype::text,
          p.proargtypes::text,p.proallargtypes,p.proargmodes,p.proargnames,
          p.protrftypes,p.provariadic::regtype::text,p.proconfig,p.prosrc,p.probin,
          p.prosqlbody::text,p.procost,p.prorows,p.prosupport::regprocedure::text,
          pg_catalog.obj_description(p.oid,'pg_proc'))::text,'UTF8')),'hex'),
        p.prosecdef
        FROM pg_catalog.pg_proc p
        JOIN pg_catalog.pg_language l ON l.oid=p.prolang
       WHERE p.pronamespace='vec_autorizacion'::regnamespace
         AND p.proname=ANY(nombres_funciones)
    ), acl_esperada AS (
      SELECT e.objeto,propietario,propietario,'EXECUTE'::text,false FROM esperado e
      UNION ALL
      SELECT e.objeto,propietario,
             'vec_autorizacion_motivos_proyector'::regrole::oid,
             'EXECUTE'::text,false FROM esperado e WHERE e.expuesta
    ), acl_actual AS (
      SELECT p.oid::regprocedure::text,a.grantor,a.grantee,
             a.privilege_type,a.is_grantable
        FROM pg_catalog.pg_proc p
        CROSS JOIN LATERAL pg_catalog.aclexplode(p.proacl) a
       WHERE p.pronamespace='vec_autorizacion'::regnamespace
         AND p.proname=ANY(nombres_funciones)
    ), diferencia AS (
      SELECT 1 FROM (SELECT * FROM esperado EXCEPT ALL SELECT * FROM actual) d
      UNION ALL SELECT 1 FROM (SELECT * FROM actual EXCEPT ALL SELECT * FROM esperado) d
      UNION ALL SELECT 1 FROM (SELECT * FROM acl_esperada EXCEPT ALL SELECT * FROM acl_actual) d
      UNION ALL SELECT 1 FROM (SELECT * FROM acl_actual EXCEPT ALL SELECT * FROM acl_esperada) d
    ) SELECT 1 FROM diferencia
  ) THEN
    RAISE EXCEPTION USING ERRCODE='55000',
      MESSAGE='retirada de resolución RRHH rechazada';
  END IF;

  IF EXISTS (
    WITH esperado(objeto,identidad,cuerpo,longitud) AS (VALUES
      ('vec_autorizacion.resolver_motivo_cuadro_rrhh_v1(timestamp with time zone)','d92704658d0af8acea83cd765e02976561c787a95906b9a10ee8a43ac0be16ef','a3d784e0a266885ca98b01e355a36cfd3acb0be1c3da6eea53a09376bc680264',6699),
      ('vec_autorizacion.resolver_motivo_detalle_rrhh_v1(timestamp with time zone)','ec662cc7118eb25eb2ebe79107c1ad1f16e5f5197fab8ff5e2051b3ddbc9fc7a','a6c3617ccb33da4bf69e2deff447bb1f93966472409fca518723919e2ce80d60',6702)
    ), actual AS (
      SELECT p.oid::regprocedure::text,pg_catalog.encode(pg_catalog.sha256(
        pg_catalog.convert_to(pg_catalog.jsonb_build_array(l.lanname,
          p.proowner::regrole::text,p.prokind::text,p.provolatile::text,
          p.proparallel::text,p.prosecdef,p.proleakproof,p.proisstrict,p.proretset,
          p.pronargs,p.pronargdefaults,p.prorettype::regtype::text,
          p.proargtypes::text,p.proallargtypes,p.proargmodes,p.proargnames,
          p.protrftypes,p.provariadic::regtype::text,p.proconfig,p.prosrc,p.probin,
          p.prosqlbody::text,p.procost,p.prorows,p.prosupport::regprocedure::text,
          pg_catalog.obj_description(p.oid,'pg_proc'))::text,'UTF8')),'hex'),
        pg_catalog.encode(pg_catalog.sha256(
          pg_catalog.convert_to(p.prosrc,'UTF8')),'hex'),
        pg_catalog.octet_length(p.prosrc)
        FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_language l ON l.oid=p.prolang
       WHERE p.pronamespace='vec_autorizacion'::regnamespace AND
             p.proname=ANY(ARRAY['resolver_motivo_cuadro_rrhh_v1',
                                 'resolver_motivo_detalle_rrhh_v1']::name[])
    ), acl_esperada(clase,objeto,grantor,grantee,privilegio,delegable) AS (
      SELECT 'base',d.datname,d.datdba,rol,'CONNECT',false
        FROM pg_catalog.pg_database d WHERE d.datname=pg_catalog.current_database()
      UNION ALL SELECT 'esquema','vec_autorizacion',propietario,rol,'USAGE',false
      UNION ALL SELECT 'funcion',e.objeto,propietario,propietario,'EXECUTE',false FROM esperado e
      UNION ALL SELECT 'funcion',e.objeto,propietario,rol,'EXECUTE',false FROM esperado e
    ), acl_actual AS (
      SELECT 'base',d.datname,a.grantor,a.grantee,a.privilege_type,a.is_grantable
        FROM pg_catalog.pg_database d
        CROSS JOIN LATERAL pg_catalog.aclexplode(d.datacl) a
       WHERE a.grantee=rol OR a.grantor=rol
      UNION ALL
      SELECT 'esquema',n.nspname,a.grantor,a.grantee,a.privilege_type,a.is_grantable
        FROM pg_catalog.pg_namespace n
        CROSS JOIN LATERAL pg_catalog.aclexplode(n.nspacl) a
       WHERE a.grantee=rol OR a.grantor=rol
      UNION ALL
      SELECT 'funcion',p.oid::regprocedure::text,a.grantor,a.grantee,
             a.privilege_type,a.is_grantable
        FROM pg_catalog.pg_proc p
        CROSS JOIN LATERAL pg_catalog.aclexplode(p.proacl) a
       WHERE p.oid=ANY(funciones)
    ), diferencia AS (
      SELECT 1 FROM (SELECT * FROM esperado EXCEPT ALL SELECT * FROM actual) d
      UNION ALL SELECT 1 FROM (SELECT * FROM actual EXCEPT ALL SELECT * FROM esperado) d
      UNION ALL SELECT 1 FROM (SELECT * FROM acl_esperada EXCEPT ALL SELECT * FROM acl_actual) d
      UNION ALL SELECT 1 FROM (SELECT * FROM acl_actual EXCEPT ALL SELECT * FROM acl_esperada) d
    ) SELECT 1 FROM diferencia
  ) OR EXISTS (
    SELECT 1 FROM pg_catalog.pg_class c
    CROSS JOIN LATERAL pg_catalog.aclexplode(c.relacl) a
     WHERE a.grantee=rol OR a.grantor=rol
    UNION ALL SELECT 1 FROM pg_catalog.pg_attribute c
    CROSS JOIN LATERAL pg_catalog.aclexplode(c.attacl) a
     WHERE a.grantee=rol OR a.grantor=rol
    UNION ALL SELECT 1 FROM pg_catalog.pg_type c
    CROSS JOIN LATERAL pg_catalog.aclexplode(c.typacl) a
     WHERE a.grantee=rol OR a.grantor=rol
    UNION ALL SELECT 1 FROM pg_catalog.pg_default_acl c
    CROSS JOIN LATERAL pg_catalog.aclexplode(c.defaclacl) a
     WHERE c.defaclrole=rol OR a.grantee=rol OR a.grantor=rol
    UNION ALL SELECT 1 FROM pg_catalog.pg_proc p
    CROSS JOIN LATERAL pg_catalog.aclexplode(p.proacl) a
     WHERE p.oid<>ALL(funciones) AND (a.grantee=rol OR a.grantor=rol)
  ) OR EXISTS (
    SELECT 1 FROM pg_catalog.pg_database WHERE datdba=rol
    UNION ALL SELECT 1 FROM pg_catalog.pg_namespace WHERE nspowner=rol
    UNION ALL SELECT 1 FROM pg_catalog.pg_class WHERE relowner=rol
    UNION ALL SELECT 1 FROM pg_catalog.pg_proc WHERE proowner=rol
    UNION ALL SELECT 1 FROM pg_catalog.pg_type WHERE typowner=rol
  ) THEN
    RAISE EXCEPTION USING ERRCODE='55000',
      MESSAGE='retirada de resolución RRHH rechazada';
  END IF;
END $prevalidacion$;

REVOKE EXECUTE ON FUNCTION
    vec_autorizacion.resolver_motivo_cuadro_rrhh_v1(timestamptz),
    vec_autorizacion.resolver_motivo_detalle_rrhh_v1(timestamptz)
    FROM vec_autorizacion_motivos_rrhh_resolutor;
REVOKE USAGE ON SCHEMA vec_autorizacion
    FROM vec_autorizacion_motivos_rrhh_resolutor;

DO $postrevocacion$
DECLARE
  rol oid := 'vec_autorizacion_motivos_rrhh_resolutor'::regrole;
  propietario oid := 'vec_autorizacion_propietario'::regrole;
  funciones oid[] := ARRAY[
    'vec_autorizacion.resolver_motivo_cuadro_rrhh_v1(timestamptz)'::regprocedure,
    'vec_autorizacion.resolver_motivo_detalle_rrhh_v1(timestamptz)'::regprocedure];
BEGIN
  IF EXISTS (
    WITH esperado(objeto,identidad,cuerpo,longitud) AS (VALUES
      ('vec_autorizacion.resolver_motivo_cuadro_rrhh_v1(timestamp with time zone)','d92704658d0af8acea83cd765e02976561c787a95906b9a10ee8a43ac0be16ef','a3d784e0a266885ca98b01e355a36cfd3acb0be1c3da6eea53a09376bc680264',6699),
      ('vec_autorizacion.resolver_motivo_detalle_rrhh_v1(timestamp with time zone)','ec662cc7118eb25eb2ebe79107c1ad1f16e5f5197fab8ff5e2051b3ddbc9fc7a','a6c3617ccb33da4bf69e2deff447bb1f93966472409fca518723919e2ce80d60',6702)
    ), actual AS (
      SELECT p.oid::regprocedure::text,pg_catalog.encode(pg_catalog.sha256(
        pg_catalog.convert_to(pg_catalog.jsonb_build_array(l.lanname,
          p.proowner::regrole::text,p.prokind::text,p.provolatile::text,
          p.proparallel::text,p.prosecdef,p.proleakproof,p.proisstrict,p.proretset,
          p.pronargs,p.pronargdefaults,p.prorettype::regtype::text,
          p.proargtypes::text,p.proallargtypes,p.proargmodes,p.proargnames,
          p.protrftypes,p.provariadic::regtype::text,p.proconfig,p.prosrc,p.probin,
          p.prosqlbody::text,p.procost,p.prorows,p.prosupport::regprocedure::text,
          pg_catalog.obj_description(p.oid,'pg_proc'))::text,'UTF8')),'hex'),
        pg_catalog.encode(pg_catalog.sha256(
          pg_catalog.convert_to(p.prosrc,'UTF8')),'hex'),
        pg_catalog.octet_length(p.prosrc)
        FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_language l ON l.oid=p.prolang
       WHERE p.pronamespace='vec_autorizacion'::regnamespace AND
             p.proname=ANY(ARRAY['resolver_motivo_cuadro_rrhh_v1',
                                 'resolver_motivo_detalle_rrhh_v1']::name[])
    ), acl_esperada(objeto,grantor,grantee,privilegio,delegable) AS (
      SELECT e.objeto,propietario,propietario,'EXECUTE'::text,false FROM esperado e
    ), acl_actual AS (
      SELECT p.oid::regprocedure::text,a.grantor,a.grantee,
             a.privilege_type,a.is_grantable
        FROM pg_catalog.pg_proc p
        CROSS JOIN LATERAL pg_catalog.aclexplode(p.proacl) a
       WHERE p.oid=ANY(funciones)
    ), diferencia AS (
      SELECT 1 FROM (SELECT * FROM esperado EXCEPT ALL SELECT * FROM actual) d
      UNION ALL SELECT 1 FROM (SELECT * FROM actual EXCEPT ALL SELECT * FROM esperado) d
      UNION ALL SELECT 1 FROM (SELECT * FROM acl_esperada EXCEPT ALL SELECT * FROM acl_actual) d
      UNION ALL SELECT 1 FROM (SELECT * FROM acl_actual EXCEPT ALL SELECT * FROM acl_esperada) d
    ) SELECT 1 FROM diferencia
  ) OR EXISTS (
    WITH esperado(objeto,grantor,grantee,privilegio,delegable) AS (
      SELECT d.datname,d.datdba,rol,'CONNECT'::text,false
        FROM pg_catalog.pg_database d WHERE d.datname=pg_catalog.current_database()
    ), actual AS (
      SELECT d.datname,a.grantor,a.grantee,a.privilege_type,a.is_grantable
        FROM pg_catalog.pg_database d
        CROSS JOIN LATERAL pg_catalog.aclexplode(d.datacl) a
       WHERE a.grantee=rol OR a.grantor=rol
    ) (SELECT * FROM esperado EXCEPT ALL SELECT * FROM actual)
      UNION ALL (SELECT * FROM actual EXCEPT ALL SELECT * FROM esperado)
  ) OR EXISTS (
    SELECT 1 FROM pg_catalog.pg_namespace n
    CROSS JOIN LATERAL pg_catalog.aclexplode(n.nspacl) a
     WHERE a.grantee=rol OR a.grantor=rol
    UNION ALL SELECT 1 FROM pg_catalog.pg_class c
    CROSS JOIN LATERAL pg_catalog.aclexplode(c.relacl) a
     WHERE a.grantee=rol OR a.grantor=rol
    UNION ALL SELECT 1 FROM pg_catalog.pg_attribute c
    CROSS JOIN LATERAL pg_catalog.aclexplode(c.attacl) a
     WHERE a.grantee=rol OR a.grantor=rol
    UNION ALL SELECT 1 FROM pg_catalog.pg_type c
    CROSS JOIN LATERAL pg_catalog.aclexplode(c.typacl) a
     WHERE a.grantee=rol OR a.grantor=rol
    UNION ALL SELECT 1 FROM pg_catalog.pg_default_acl c
    CROSS JOIN LATERAL pg_catalog.aclexplode(c.defaclacl) a
     WHERE c.defaclrole=rol OR a.grantee=rol OR a.grantor=rol
    UNION ALL SELECT 1 FROM pg_catalog.pg_proc p
    CROSS JOIN LATERAL pg_catalog.aclexplode(p.proacl) a
     WHERE p.oid<>ALL(funciones) AND (a.grantee=rol OR a.grantor=rol)
  ) THEN
    RAISE EXCEPTION USING ERRCODE='55000',
      MESSAGE='retirada de resolución RRHH rechazada';
  END IF;
END $postrevocacion$;

DROP FUNCTION
    vec_autorizacion.resolver_motivo_cuadro_rrhh_v1(timestamptz),
    vec_autorizacion.resolver_motivo_detalle_rrhh_v1(timestamptz)
    RESTRICT;

DO $postretirada$
DECLARE
  rol oid := 'vec_autorizacion_motivos_rrhh_resolutor'::regrole;
BEGIN
  IF EXISTS (
    SELECT 1 FROM pg_catalog.pg_proc
     WHERE pronamespace='vec_autorizacion'::regnamespace
       AND proname=ANY(ARRAY['resolver_motivo_cuadro_rrhh_v1',
                             'resolver_motivo_detalle_rrhh_v1']::name[])
  ) OR EXISTS (
    WITH esperado(objeto,grantor,grantee,privilegio,delegable) AS (
      SELECT d.datname,d.datdba,rol,'CONNECT'::text,false
        FROM pg_catalog.pg_database d WHERE d.datname=pg_catalog.current_database()
    ), actual AS (
      SELECT d.datname,a.grantor,a.grantee,a.privilege_type,a.is_grantable
        FROM pg_catalog.pg_database d
        CROSS JOIN LATERAL pg_catalog.aclexplode(d.datacl) a
       WHERE a.grantee=rol OR a.grantor=rol
    ) (SELECT * FROM esperado EXCEPT ALL SELECT * FROM actual)
      UNION ALL (SELECT * FROM actual EXCEPT ALL SELECT * FROM esperado)
  ) OR EXISTS (
    SELECT 1 FROM pg_catalog.pg_namespace n
    CROSS JOIN LATERAL pg_catalog.aclexplode(n.nspacl) a
     WHERE a.grantee=rol OR a.grantor=rol
    UNION ALL SELECT 1 FROM pg_catalog.pg_class c
    CROSS JOIN LATERAL pg_catalog.aclexplode(c.relacl) a
     WHERE a.grantee=rol OR a.grantor=rol
    UNION ALL SELECT 1 FROM pg_catalog.pg_attribute c
    CROSS JOIN LATERAL pg_catalog.aclexplode(c.attacl) a
     WHERE a.grantee=rol OR a.grantor=rol
    UNION ALL SELECT 1 FROM pg_catalog.pg_type c
    CROSS JOIN LATERAL pg_catalog.aclexplode(c.typacl) a
     WHERE a.grantee=rol OR a.grantor=rol
    UNION ALL SELECT 1 FROM pg_catalog.pg_default_acl c
    CROSS JOIN LATERAL pg_catalog.aclexplode(c.defaclacl) a
     WHERE c.defaclrole=rol OR a.grantee=rol OR a.grantor=rol
    UNION ALL SELECT 1 FROM pg_catalog.pg_proc p
    CROSS JOIN LATERAL pg_catalog.aclexplode(p.proacl) a
     WHERE a.grantee=rol OR a.grantor=rol
  ) OR EXISTS (
    SELECT 1 FROM pg_catalog.pg_database WHERE datdba=rol
    UNION ALL SELECT 1 FROM pg_catalog.pg_namespace WHERE nspowner=rol
    UNION ALL SELECT 1 FROM pg_catalog.pg_class WHERE relowner=rol
    UNION ALL SELECT 1 FROM pg_catalog.pg_proc WHERE proowner=rol
    UNION ALL SELECT 1 FROM pg_catalog.pg_type WHERE typowner=rol
  ) OR NOT EXISTS (
    SELECT 1 FROM pg_catalog.pg_roles r WHERE r.oid=rol
      AND NOT r.rolcanlogin AND NOT r.rolsuper AND NOT r.rolcreatedb
      AND NOT r.rolcreaterole AND NOT r.rolinherit AND NOT r.rolreplication
      AND NOT r.rolbypassrls AND r.rolconnlimit=-1
      AND (r.rolpassword IS NULL OR r.rolpassword='********')
      AND r.rolvaliduntil IS NULL AND r.rolconfig IS NULL
      AND pg_catalog.shobj_description(r.oid,'pg_authid')=
          'vec_autorizacion:rol-motivos-rrhh-resolutor:v1'
  ) OR EXISTS (SELECT 1 FROM pg_catalog.pg_db_role_setting WHERE setrole=rol)
    OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE oid=10 AND rolsuper)
    OR EXISTS (
      SELECT 1 FROM pg_catalog.pg_auth_members m
      LEFT JOIN pg_catalog.pg_roles a ON a.oid=m.member
       WHERE m.member=rol OR m.grantor=rol OR (m.roleid=rol AND
         (m.admin_option OR NOT m.inherit_option OR m.set_option OR m.grantor<>10
          OR NOT a.rolcanlogin OR NOT a.rolinherit OR a.rolsuper OR a.rolcreatedb
          OR a.rolcreaterole OR a.rolreplication OR a.rolbypassrls))) THEN
    RAISE EXCEPTION USING ERRCODE='55000',
      MESSAGE='retirada de resolución RRHH incompleta';
  END IF;
END $postretirada$;

COMMIT;
