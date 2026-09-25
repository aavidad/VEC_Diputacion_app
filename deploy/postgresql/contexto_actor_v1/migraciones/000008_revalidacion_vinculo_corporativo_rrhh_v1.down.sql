-- Reversión de ContextoActor 000008: retira la revalidación corporativa y
-- restaura exactamente la comprobación de runtime de 000007 (cinco funciones).
-- La función retirada no firma ni registra nada; no hay historia que perder.
BEGIN;
SET LOCAL search_path = pg_catalog;
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
  'vec_contexto_actor_v1:migracion:revalidacion_vinculo_corporativo:v1',0));
DO $preimagen$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
     OR pg_catalog.to_regprocedure('vec_contexto_actor_v1.revalidar_vinculo_corporativo_rrhh_v1(text,text,text,text,numeric)') IS NULL
     OR (SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(prosrc,'UTF8')),'hex')
           FROM pg_catalog.pg_proc
          WHERE oid = pg_catalog.to_regprocedure('vec_contexto_actor_v1.exigir_runtime_contexto_actor_v1()'))
        IS DISTINCT FROM '005fff9328377a75bcde2a23e997959f2d1603f658ae39a2e6f2eb8d8986d3c8' THEN
    RAISE EXCEPTION 'ContextoActor 000008 no instalada o divergente' USING ERRCODE='55000';
  END IF;
END
$preimagen$;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
SET LOCAL search_path = pg_catalog;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.revalidar_vinculo_corporativo_rrhh_v1(text,text,text,text,numeric)
    FROM vec_contexto_actor_v1_runtime;
DROP FUNCTION vec_contexto_actor_v1.revalidar_vinculo_corporativo_rrhh_v1(text,text,text,text,numeric);

CREATE OR REPLACE FUNCTION vec_contexto_actor_v1.exigir_runtime_contexto_actor_v1()
RETURNS text LANGUAGE plpgsql SECURITY DEFINER SET search_path = pg_catalog AS $f$
DECLARE
    login_oid oid; runtime_oid oid; esquema_oid oid; base_oid oid;
    membresias integer; funciones oid[]; login record; grupo record;
BEGIN
    SELECT oid, rolsuper, rolinherit, rolcreaterole, rolcreatedb, rolcanlogin,
           rolreplication, rolbypassrls, rolconfig
      INTO login FROM pg_catalog.pg_roles WHERE rolname = session_user;
    SELECT oid, rolsuper, rolinherit, rolcreaterole, rolcreatedb, rolcanlogin,
           rolreplication, rolbypassrls, rolconfig
      INTO grupo FROM pg_catalog.pg_roles WHERE rolname = 'vec_contexto_actor_v1_runtime';
    runtime_oid := grupo.oid;
    login_oid := login.oid;
    SELECT oid INTO esquema_oid FROM pg_catalog.pg_namespace WHERE nspname='vec_contexto_actor_v1';
    SELECT oid INTO base_oid FROM pg_catalog.pg_database WHERE datname=current_database();
    funciones := ARRAY[
      pg_catalog.to_regprocedure('vec_contexto_actor_v1.acreditar_runtime_contexto_actor_v1()'),
      pg_catalog.to_regprocedure('vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2(text,text,text,text,text,text,timestamptz)'),
      pg_catalog.to_regprocedure('vec_contexto_actor_v1.reconciliar_contexto_actor_v2(text,text,text,text,text,text,timestamptz)'),
      pg_catalog.to_regprocedure('vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2(text,text,text,text,text,text,timestamptz,text[])'),
      pg_catalog.to_regprocedure('vec_contexto_actor_v1.reconciliar_contexto_actor_v2(text,text,text,text,text,text,timestamptz,text[])')
    ];
    SELECT count(*) INTO membresias FROM pg_catalog.pg_auth_members
     WHERE member = login_oid;
    IF login_oid IS NULL OR runtime_oid IS NULL OR esquema_oid IS NULL OR base_oid IS NULL
       OR cardinality(funciones) <> 5 OR array_position(funciones,NULL) IS NOT NULL
       OR login.rolcanlogin IS NOT TRUE
       OR login.rolsuper OR NOT login.rolinherit OR login.rolcreaterole OR login.rolcreatedb
       OR login.rolreplication OR login.rolbypassrls OR login.rolconfig IS NOT NULL
       OR grupo.rolcanlogin OR grupo.rolsuper OR grupo.rolinherit
       OR grupo.rolcreaterole OR grupo.rolcreatedb OR grupo.rolreplication
       OR grupo.rolbypassrls OR grupo.rolconfig IS NOT NULL
       OR current_setting('role') <> 'none' OR membresias <> 1
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_roles r
            WHERE r.oid<>login_oid
              AND pg_catalog.pg_has_role(login_oid,r.oid,'MEMBER')
              AND r.oid<>runtime_oid
       )
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_roles r
            WHERE r.oid<>runtime_oid
              AND pg_catalog.pg_has_role(runtime_oid,r.oid,'MEMBER')
       )
       OR NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_auth_members
            WHERE member = login_oid AND roleid = runtime_oid
              AND admin_option IS FALSE AND inherit_option IS TRUE AND set_option IS FALSE
       )
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_db_role_setting s
            WHERE s.setrole IN (login_oid,runtime_oid)
       )
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_default_acl d
           LEFT JOIN LATERAL pg_catalog.aclexplode(
             coalesce(d.defaclacl,'{}'::aclitem[])
           ) a ON true
            WHERE d.defaclrole IN (login_oid,runtime_oid)
               OR a.grantee IN (login_oid,runtime_oid)
               OR a.grantor IN (login_oid,runtime_oid)
       )
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_policy p
            WHERE login_oid=ANY(p.polroles) OR runtime_oid=ANY(p.polroles)
       )
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_shdepend d
            WHERE d.refclassid='pg_catalog.pg_authid'::regclass
              AND d.refobjid=login_oid
       )
       OR NOT COALESCE((
           SELECT count(*)=7 AND bool_and(
             d.deptype='a' AND d.objsubid=0 AND (
               (d.classid='pg_catalog.pg_database'::regclass AND d.objid=base_oid) OR
               (d.classid='pg_catalog.pg_namespace'::regclass AND d.objid=esquema_oid) OR
               (d.classid='pg_catalog.pg_proc'::regclass AND d.objid=ANY(funciones))
             ))
             FROM pg_catalog.pg_shdepend d
            WHERE d.refclassid='pg_catalog.pg_authid'::regclass
              AND d.refobjid=runtime_oid
       ),false)
       OR NOT COALESCE((
           SELECT count(*)=1 AND bool_and(a.privilege_type='CONNECT' AND NOT a.is_grantable)
             FROM pg_catalog.pg_database b
             CROSS JOIN LATERAL pg_catalog.aclexplode(
               coalesce(b.datacl,pg_catalog.acldefault('d',b.datdba))
             ) a
            WHERE b.oid=base_oid AND a.grantee=runtime_oid
       ),false)
       OR NOT COALESCE((
           SELECT count(*)=1 AND bool_and(a.privilege_type='USAGE' AND NOT a.is_grantable)
             FROM pg_catalog.pg_namespace n
             CROSS JOIN LATERAL pg_catalog.aclexplode(
               coalesce(n.nspacl,pg_catalog.acldefault('n',n.nspowner))
             ) a
            WHERE n.oid=esquema_oid AND a.grantee=runtime_oid
       ),false)
       OR NOT COALESCE((
           SELECT count(*)=5 AND count(DISTINCT p.oid)=5
                  AND bool_and(a.privilege_type='EXECUTE' AND NOT a.is_grantable)
             FROM pg_catalog.pg_proc p
             CROSS JOIN LATERAL pg_catalog.aclexplode(
               coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))
            ) a
            WHERE p.oid=ANY(funciones) AND a.grantee=runtime_oid
       ),false)
       OR vec_contexto_actor_v1.privilegios_efectivos_runtime_minimos(
            login_oid,base_oid,esquema_oid,funciones) IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = '42501', MESSAGE = 'LOGIN runtime de contexto actor V1 no acreditado';
    END IF;
    RETURN session_user;
END
$f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.exigir_runtime_contexto_actor_v1() FROM PUBLIC;

DO $postimagen$
BEGIN
    IF (SELECT count(*) FROM pg_catalog.pg_proc p
          WHERE p.pronamespace = 'vec_contexto_actor_v1'::regnamespace
            AND pg_catalog.has_function_privilege('vec_contexto_actor_v1_runtime', p.oid, 'EXECUTE')) <> 5
       OR (SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(prosrc,'UTF8')),'hex')
             FROM pg_catalog.pg_proc
            WHERE oid = 'vec_contexto_actor_v1.exigir_runtime_contexto_actor_v1()'::regprocedure)
          <> 'f29b3374ea974e13e49454456e7fe80d6e1e7030cab0f566eb6c41eea0f3ab88' THEN
        RAISE EXCEPTION 'reversion ContextoActor 000008 divergente' USING ERRCODE='55000';
    END IF;
END
$postimagen$;
COMMIT;
