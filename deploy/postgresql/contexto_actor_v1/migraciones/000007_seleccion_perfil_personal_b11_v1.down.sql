-- Retirada exclusivamente sin historia B11; no elimina contexto historico.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contexto_actor_v1:perfil-personal-b11:v1',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contexto_actor_v1:mutacion_punteros_actuales:v2',0));
LOCK TABLE vec_contexto_actor_v1.seleccion_perfil_personal_b11_actual,
 vec_contexto_actor_v1.seleccion_perfil_personal_b11_versiones,
 vec_contexto_actor_v1.seleccion_perfil_personal_b11_usos IN ACCESS EXCLUSIVE MODE;
DO $guard$
BEGIN
 IF current_setting('vec.confirmar_retirada_seleccion_perfil_personal_b11_v1',true) IS DISTINCT FROM 'RETIRAR_SELECCION_PERFIL_PERSONAL_B11_V1'
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 OR EXISTS(SELECT 1 FROM vec_contexto_actor_v1.seleccion_perfil_personal_b11_versiones)
 OR EXISTS(SELECT 1 FROM vec_contexto_actor_v1.seleccion_perfil_personal_b11_actual)
 OR EXISTS(SELECT 1 FROM vec_contexto_actor_v1.seleccion_perfil_personal_b11_usos)
 THEN RAISE EXCEPTION USING ERRCODE='55000',MESSAGE='retirada B11 rechazada por historia o confirmacion'; END IF;
END $guard$;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
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
      pg_catalog.to_regprocedure('vec_contexto_actor_v1.reconciliar_contexto_actor_v2(text,text,text,text,text,text,timestamptz)')
    ];
    SELECT count(*) INTO membresias FROM pg_catalog.pg_auth_members
     WHERE member = login_oid;
    IF login_oid IS NULL OR runtime_oid IS NULL OR esquema_oid IS NULL OR base_oid IS NULL
       OR cardinality(funciones) <> 3 OR array_position(funciones,NULL) IS NOT NULL
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
           SELECT count(*)=5 AND bool_and(
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
           SELECT count(*)=3 AND count(DISTINCT p.oid)=3
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

DROP FUNCTION vec_contexto_actor_v1.acreditar_runtime_contexto_actor_perfil_personal_b11_v1() RESTRICT;
DROP FUNCTION vec_contexto_actor_v1.reconciliar_contexto_actor_perfil_personal_b11_v1(text,text,text,text,text,timestamptz) RESTRICT;
DROP FUNCTION vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_perfil_personal_b11_v1(text,text,text,text,text,timestamptz) RESTRICT;
DROP FUNCTION vec_contexto_actor_v1.revalidar_registro_b11(text) RESTRICT;
DROP FUNCTION vec_contexto_actor_v1.validar_entrada_b11(text,text,text,text,text,timestamptz) RESTRICT;
DROP FUNCTION vec_contexto_actor_v1.publicar_seleccion_perfil_personal_b11_v1(text,numeric,text,text,text,numeric,text,text,timestamptz,timestamptz) RESTRICT;
DROP FUNCTION vec_contexto_actor_v1.exigir_runtime_exacto_b11(boolean) RESTRICT;
DROP TABLE vec_contexto_actor_v1.seleccion_perfil_personal_b11_usos RESTRICT;
DROP TABLE vec_contexto_actor_v1.seleccion_perfil_personal_b11_actual RESTRICT;
DROP TABLE vec_contexto_actor_v1.seleccion_perfil_personal_b11_versiones RESTRICT;
DROP FUNCTION vec_contexto_actor_v1.validar_enlace_seleccion_b11() RESTRICT;
REVOKE USAGE ON SCHEMA vec_contexto_actor_v1 FROM vec_contexto_actor_perfil_personal_b11_runtime;
COMMIT;
