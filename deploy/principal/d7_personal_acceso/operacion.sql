\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('d7:personal:acceso:20260924', 0));
LOCK TABLE vec_autorizacion.asignacion_perfil_actual IN SHARE ROW EXCLUSIVE MODE;
SET LOCAL vec.d7.modo = :'d7_modo';

-- Única ACL heredable por los LOGIN D7. En preparar, CONNECT del ejecutor
-- puede faltar: esta operación lo concede después del cotejo de preimagen.
CREATE TEMP TABLE d7_acl_permitida
 (grupo text, clase oid, objeto oid, columna integer, privilegio text, grantable boolean)
 ON COMMIT DROP;
INSERT INTO d7_acl_permitida VALUES
 ('vec_personal_d7_ejecutor','pg_database'::regclass,(SELECT oid FROM pg_database WHERE datname=current_database()),0,'CONNECT',false),
 ('vec_personal_registrador_frontera','pg_database'::regclass,(SELECT oid FROM pg_database WHERE datname=current_database()),0,'CONNECT',false),
 ('vec_personal_d7_ejecutor','pg_namespace'::regclass,to_regnamespace('vec_personal'),0,'USAGE',false),
 ('vec_personal_registrador_frontera','pg_namespace'::regclass,to_regnamespace('vec_personal'),0,'USAGE',false),
 ('vec_personal_d7_ejecutor','pg_proc'::regclass,to_regprocedure('vec_personal.registrar_asignacion_dietas_inicial_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'),0,'EXECUTE',false),
 ('vec_personal_d7_ejecutor','pg_proc'::regclass,to_regprocedure('vec_personal.consultar_asignacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'),0,'EXECUTE',false),
 ('vec_personal_d7_ejecutor','pg_proc'::regclass,to_regprocedure('vec_personal.corregir_asignacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'),0,'EXECUTE',false),
 ('vec_personal_d7_ejecutor','pg_proc'::regclass,to_regprocedure('vec_personal.corregir_grupo_dieta_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'),0,'EXECUTE',false),
 ('vec_personal_registrador_frontera','pg_proc'::regclass,to_regprocedure('vec_personal.registrar_auditoria_frontera_asignacion_dietas_v1(text,text,text,text,text,text,text,smallint)'),0,'EXECUTE',false);

CREATE TEMP TABLE d7_acl_observada ON COMMIT DROP AS
WITH grupos AS (SELECT oid,rolname FROM pg_roles WHERE rolname IN
 ('vec_personal_d7_ejecutor','vec_personal_registrador_frontera'))
SELECT g.rolname::text grupo,'pg_database'::regclass::oid clase,d.oid objeto,
       0::integer columna,a.privilege_type::text privilegio,a.is_grantable grantable
  FROM pg_database d,LATERAL aclexplode(coalesce(d.datacl,acldefault('d',d.datdba))) a
  JOIN grupos g ON g.oid=a.grantee
UNION ALL
SELECT g.rolname,'pg_namespace'::regclass::oid,n.oid,0,a.privilege_type::text,a.is_grantable
  FROM pg_namespace n,LATERAL aclexplode(coalesce(n.nspacl,acldefault('n',n.nspowner))) a
  JOIN grupos g ON g.oid=a.grantee
UNION ALL
SELECT g.rolname,'pg_class'::regclass::oid,c.oid,0,a.privilege_type::text,a.is_grantable
  FROM pg_class c,LATERAL aclexplode(coalesce(c.relacl,acldefault((CASE WHEN c.relkind='S' THEN 'S' ELSE 'r' END)::"char",c.relowner))) a
  JOIN grupos g ON g.oid=a.grantee
UNION ALL
SELECT g.rolname,'pg_class'::regclass::oid,c.oid,x.attnum,a.privilege_type::text,a.is_grantable
  FROM pg_attribute x JOIN pg_class c ON c.oid=x.attrelid,
       LATERAL aclexplode(x.attacl) a JOIN grupos g ON g.oid=a.grantee WHERE x.attnum>0
UNION ALL
SELECT g.rolname,'pg_proc'::regclass::oid,p.oid,0,a.privilege_type::text,a.is_grantable
  FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
  JOIN grupos g ON g.oid=a.grantee
UNION ALL
SELECT g.rolname,'pg_type'::regclass::oid,t.oid,0,a.privilege_type::text,a.is_grantable
  FROM pg_type t,LATERAL aclexplode(coalesce(t.typacl,acldefault('T',t.typowner))) a
  JOIN grupos g ON g.oid=a.grantee;

DO $d7$
DECLARE
  modo text := current_setting('vec.d7.modo');
  f text;
  grupo text;
  cuenta_esperada record;
  n integer;
BEGIN
  IF modo NOT IN ('preparar', 'activar')
     OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
     OR session_user <> current_user
     OR NOT (SELECT rolsuper FROM pg_roles WHERE rolname = session_user)
     OR current_database() <> 'postgres'
     OR inet_server_addr() IS NOT NULL THEN
    RAISE EXCEPTION 'D7 exige PostgreSQL 18, DBA, socket local y base postgres' USING ERRCODE='42501';
  END IF;

  -- F4 queda cerrado con su puntero v3 y exactamente sus ocho LOGIN. D7 no
  -- acepta una novena cuenta bajo ese prefijo ni modifica ese conjunto.
  IF to_regclass('vec_autorizacion.asignacion_perfil') IS NULL
     OR to_regclass('vec_autorizacion.version_rol') IS NULL THEN
    RAISE EXCEPTION 'preimagen F4 ausente' USING ERRCODE='55000';
  END IF;
  SELECT count(*) INTO n
    FROM vec_autorizacion.asignacion_perfil_actual p
    JOIN vec_autorizacion.asignacion_perfil a ON a.asignacion_ref=p.asignacion_ref
    JOIN vec_autorizacion.version_rol v ON v.version_rol_ref=a.version_rol_ref
   WHERE v.rol_id='dietas_r1d_provisional';
  IF n<>1 OR NOT EXISTS (
    SELECT 1 FROM vec_autorizacion.asignacion_perfil_actual p
    JOIN vec_autorizacion.asignacion_perfil a ON a.asignacion_ref=p.asignacion_ref
    WHERE a.version_rol_ref='rol:dietas_r1d_provisional:v1'
      AND a.version=3 AND a.documento->>'version'='3'
      AND a.asignacion_ref='asignacion:'||(a.documento->>'asignacion_id')||':v3'
      AND a.huella_sha256 ~ '^[0-9a-f]{64}$'
      AND a.documento->>'version_rol_ref'=a.version_rol_ref
      AND a.documento->>'estado'='activa'
      AND a.documento->>'emitida_por'='administracion:f4:reactivacion-dietas-r1d'
      AND (a.documento->>'vigente_desde')::timestamptz <= clock_timestamp()
      AND (a.documento->>'vigente_hasta')::timestamptz > clock_timestamp()
      AND p.actualizada_por='administracion:f4:reactivacion-dietas-r1d'
      AND p.acto_ref='acto:f4:reactivacion-dietas-r1d:20260924'
  ) THEN RAISE EXCEPTION 'puntero F4 v3 no acreditado' USING ERRCODE='55000'; END IF;
  WITH esperados(nombre) AS (VALUES
    ('vec_dietas_r1d_registro_identidad_desarrollo'),
    ('vec_dietas_r1d_revalidacion_identidad_desarrollo'),
    ('vec_dietas_r1d_contexto_desarrollo'),
    ('vec_dietas_r1d_fuente_autorizacion_desarrollo'),
    ('vec_dietas_r1d_registro_autorizacion_desarrollo'),
    ('vec_dietas_r1d_motivos_desarrollo'),
    ('vec_dietas_r1d_dietas_desarrollo'),
    ('vec_dietas_r1d_personal_desarrollo')
  ) SELECT count(*) INTO n FROM esperados e JOIN pg_roles f4_rol ON f4_rol.rolname=e.nombre AND f4_rol.rolcanlogin;
  IF n<>8 OR (SELECT count(*) FROM pg_roles WHERE rolname ~ '^vec_dietas_r1d_.*_desarrollo$')<>8
  THEN RAISE EXCEPTION 'ocho LOGIN F4 no acreditados' USING ERRCODE='55000'; END IF;

  IF to_regnamespace('vec_personal') IS NULL
     OR EXISTS (SELECT 1 FROM pg_roles WHERE rolname IN
       ('vec_personal_d7_ejecutor','vec_personal_registrador_frontera')
       AND (rolcanlogin OR rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls OR NOT rolinherit))
     OR (SELECT count(*) FROM pg_roles WHERE rolname IN
       ('vec_personal_d7_ejecutor','vec_personal_registrador_frontera'))<>2
     OR EXISTS (SELECT 1 FROM pg_auth_members m JOIN pg_roles r ON r.oid=m.member
       WHERE r.rolname IN ('vec_personal_d7_ejecutor','vec_personal_registrador_frontera'))
     OR EXISTS (SELECT 1 FROM pg_authid r WHERE r.rolname IN
       ('vec_personal_d7_ejecutor','vec_personal_registrador_frontera')
       AND (r.rolpassword IS NOT NULL OR r.rolconnlimit<>-1 OR r.rolvaliduntil IS NOT NULL))
     OR EXISTS (SELECT 1 FROM pg_db_role_setting s JOIN pg_roles r ON r.oid=s.setrole
       WHERE r.rolname IN ('vec_personal_d7_ejecutor','vec_personal_registrador_frontera'))
     OR EXISTS (
       SELECT 1 FROM pg_auth_members m JOIN pg_roles g ON g.oid=m.roleid
       WHERE g.rolname IN ('vec_personal_d7_ejecutor','vec_personal_registrador_frontera')
         AND (modo='preparar' OR NOT EXISTS (
           SELECT 1 FROM (VALUES
             ('vec_personal_d7_ejecutor','vec_personal_d7_asignacion'),
             ('vec_personal_registrador_frontera','vec_personal_d7_auditoria_frontera')
           ) x(grupo,cuenta) JOIN pg_roles c ON c.rolname=x.cuenta
           WHERE x.grupo=g.rolname AND c.oid=m.member)))
  THEN RAISE EXCEPTION 'grupos Personal incompatibles' USING ERRCODE='55000'; END IF;

  IF EXISTS (SELECT 1 FROM d7_acl_permitida WHERE objeto IS NULL)
     OR EXISTS (SELECT * FROM d7_acl_observada EXCEPT SELECT * FROM d7_acl_permitida)
     OR EXISTS (
       SELECT * FROM d7_acl_permitida e
       WHERE NOT (modo='preparar' AND e.grupo='vec_personal_d7_ejecutor'
         AND e.clase='pg_database'::regclass AND e.privilegio='CONNECT')
       EXCEPT SELECT * FROM d7_acl_observada)
     OR EXISTS (SELECT 1 FROM pg_shdepend d JOIN pg_roles r ON r.oid=d.refobjid
       WHERE r.rolname IN ('vec_personal_d7_ejecutor','vec_personal_registrador_frontera')
         AND d.refclassid='pg_authid'::regclass AND d.deptype='o')
     OR EXISTS (SELECT 1 FROM pg_shdepend d JOIN pg_roles r ON r.oid=d.refobjid
       WHERE r.rolname IN ('vec_personal_d7_ejecutor','vec_personal_registrador_frontera')
         AND d.refclassid='pg_authid'::regclass AND d.deptype='a'
         AND (d.dbid NOT IN (0,(SELECT oid FROM pg_database WHERE datname=current_database()))
           OR d.classid NOT IN ('pg_database'::regclass,'pg_namespace'::regclass,
             'pg_class'::regclass,'pg_proc'::regclass,'pg_type'::regclass)))
  THEN RAISE EXCEPTION 'ACL o propiedad de grupos D7 fuera de preimagen permitida' USING ERRCODE='55000'; END IF;

  FOREACH f IN ARRAY ARRAY[
   'vec_personal.registrar_asignacion_dietas_inicial_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
   'vec_personal.consultar_asignacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
   'vec_personal.corregir_asignacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
   'vec_personal.corregir_grupo_dieta_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
   'vec_personal.registrar_auditoria_frontera_asignacion_dietas_v1(text,text,text,text,text,text,text,smallint)'
  ] LOOP
    grupo := CASE WHEN f LIKE '%auditoria_frontera%' THEN
      'vec_personal_registrador_frontera' ELSE 'vec_personal_d7_ejecutor' END;
    IF to_regprocedure(f) IS NULL
       OR NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=to_regprocedure(f)
           AND p.proowner='vec_personal_propietario'::regrole AND p.prosecdef)
       OR NOT has_function_privilege(grupo, f, 'EXECUTE')
       OR NOT EXISTS (SELECT 1 FROM pg_proc p, aclexplode(p.proacl) a
          WHERE p.oid=to_regprocedure(f) AND a.grantee=grupo::regrole
            AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)
       OR EXISTS (SELECT 1 FROM pg_proc p, aclexplode(p.proacl) a
          WHERE p.oid=to_regprocedure(f) AND (a.grantee=0 OR
            a.grantee=CASE WHEN grupo='vec_personal_d7_ejecutor' THEN
              'vec_personal_registrador_frontera'::regrole ELSE 'vec_personal_d7_ejecutor'::regrole END))
    THEN RAISE EXCEPTION 'funcion o ACL Personal requerida ausente/cruzada: %',f USING ERRCODE='55000'; END IF;
  END LOOP;
  IF NOT has_database_privilege('vec_personal_registrador_frontera',current_database(),'CONNECT')
     OR NOT has_schema_privilege('vec_personal_d7_ejecutor','vec_personal','USAGE')
     OR NOT has_schema_privilege('vec_personal_registrador_frontera','vec_personal','USAGE')
  THEN RAISE EXCEPTION 'CONNECT/USAGE Personal no exacto' USING ERRCODE='55000'; END IF;

  IF EXISTS (SELECT 1 FROM pg_database d, aclexplode(coalesce(d.datacl,acldefault('d',d.datdba))) a
       WHERE d.datname=current_database() AND a.grantee=0)
     OR EXISTS (SELECT 1 FROM pg_namespace ns, aclexplode(coalesce(ns.nspacl,acldefault('n',ns.nspowner))) a
       WHERE ns.nspname LIKE 'vec\_%' ESCAPE '\' AND a.grantee=0)
     OR EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace ns ON ns.oid=c.relnamespace,
       aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) a
       WHERE ns.nspname LIKE 'vec\_%' ESCAPE '\' AND a.grantee=0)
     OR EXISTS (SELECT 1 FROM pg_proc p JOIN pg_namespace ns ON ns.oid=p.pronamespace,
       aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
       WHERE ns.nspname LIKE 'vec\_%' ESCAPE '\' AND a.grantee=0)
  THEN RAISE EXCEPTION 'ACL PUBLIC VEC/base fuera de preimagen' USING ERRCODE='55000'; END IF;

  FOR cuenta_esperada IN SELECT * FROM (VALUES
      ('vec_personal_d7_asignacion','vec_personal_d7_ejecutor'),
      ('vec_personal_d7_auditoria_frontera','vec_personal_registrador_frontera')
    ) AS x(nombre,grupo) LOOP
    IF modo='preparar' THEN
      IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname=cuenta_esperada.nombre) THEN
        RAISE EXCEPTION 'cuenta D7 preexistente; preparar no repetible' USING ERRCODE='55000';
      END IF;
    ELSE
      IF NOT EXISTS (SELECT 1 FROM pg_authid a WHERE a.rolname=cuenta_esperada.nombre
         AND NOT a.rolcanlogin AND NOT a.rolsuper AND NOT a.rolcreatedb
         AND NOT a.rolcreaterole AND a.rolinherit AND NOT a.rolreplication
         AND NOT a.rolbypassrls AND a.rolconnlimit=-1 AND a.rolvaliduntil IS NULL
         AND a.rolpassword LIKE 'SCRAM-SHA-256$%')
         OR EXISTS (SELECT 1 FROM pg_db_role_setting s
               WHERE s.setrole=cuenta_esperada.nombre::regrole)
         OR (SELECT count(*) FROM pg_auth_members m
               WHERE m.member=cuenta_esperada.nombre::regrole)<>1
         OR NOT EXISTS (SELECT 1 FROM pg_auth_members m
               WHERE m.member=cuenta_esperada.nombre::regrole AND m.roleid=cuenta_esperada.grupo::regrole
                 AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
         OR EXISTS (SELECT 1 FROM pg_auth_members m
               WHERE m.roleid=cuenta_esperada.nombre::regrole)
         OR EXISTS (SELECT 1 FROM pg_shdepend d
               WHERE d.refobjid=cuenta_esperada.nombre::regrole AND d.deptype='o')
         OR EXISTS (SELECT 1 FROM pg_database d, aclexplode(d.datacl) a
               WHERE a.grantee=cuenta_esperada.nombre::regrole)
         OR EXISTS (SELECT 1 FROM pg_namespace ns, aclexplode(ns.nspacl) a
               WHERE a.grantee=cuenta_esperada.nombre::regrole)
         OR EXISTS (SELECT 1 FROM pg_class c, aclexplode(c.relacl) a
               WHERE a.grantee=cuenta_esperada.nombre::regrole)
         OR EXISTS (SELECT 1 FROM pg_proc p, aclexplode(p.proacl) a
               WHERE a.grantee=cuenta_esperada.nombre::regrole)
         OR EXISTS (SELECT 1 FROM pg_type t, aclexplode(t.typacl) a
               WHERE a.grantee=cuenta_esperada.nombre::regrole)
         OR EXISTS (SELECT 1 FROM pg_stat_activity WHERE usename=cuenta_esperada.nombre)
      THEN RAISE EXCEPTION 'preimagen LOGIN D7 incompatible: %',cuenta_esperada.nombre USING ERRCODE='55000'; END IF;
    END IF;
  END LOOP;

  IF modo='activar' AND
     (SELECT a.rolpassword FROM pg_authid a WHERE a.rolname='vec_personal_d7_asignacion')
     IS NOT DISTINCT FROM
     (SELECT a.rolpassword FROM pg_authid a WHERE a.rolname='vec_personal_d7_auditoria_frontera')
  THEN RAISE EXCEPTION 'credenciales D7 no segregadas' USING ERRCODE='55000'; END IF;

  IF modo='preparar' THEN
    GRANT CONNECT ON DATABASE postgres TO vec_personal_d7_ejecutor;
    CREATE ROLE vec_personal_d7_asignacion NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
    CREATE ROLE vec_personal_d7_auditoria_frontera NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
    GRANT vec_personal_d7_ejecutor TO vec_personal_d7_asignacion WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
    GRANT vec_personal_registrador_frontera TO vec_personal_d7_auditoria_frontera WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
  ELSE
    ALTER ROLE vec_personal_d7_asignacion LOGIN;
    ALTER ROLE vec_personal_d7_auditoria_frontera LOGIN;
  END IF;
  IF (SELECT count(*) FROM pg_roles WHERE rolname IN
      ('vec_personal_d7_asignacion','vec_personal_d7_auditoria_frontera')
      AND rolcanlogin=(modo='activar'))<>2 THEN
    RAISE EXCEPTION 'postcondicion LOGIN D7 fallida' USING ERRCODE='55000';
  END IF;
  IF modo='preparar' AND
     (SELECT count(*) FROM pg_auth_members m JOIN pg_roles g ON g.oid=m.roleid
       WHERE g.rolname IN ('vec_personal_d7_ejecutor','vec_personal_registrador_frontera'))<>2
  THEN RAISE EXCEPTION 'postcondicion membresias D7 fallida' USING ERRCODE='55000'; END IF;
END $d7$;

:d7_finalizar
