\set ON_ERROR_STOP on
-- AUT47. CONNECT sobre la base actual al grupo NOLOGIN vec_admin_usuarios_lector.
-- AUT43 creó el grupo sin CONNECT; sin PUBLIC CONNECT el LOGIN lector de
-- cmd/vec-admin no podía entrar. Sólo concede CONNECT (sin GRANT OPTION); nada
-- más. Si ya lo tiene no hace nada. Sin DOWN: retirar CONNECT rompería el lector.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion:migracion:000047',0));
DO $pre$
DECLARE g oid:=to_regrole('vec_admin_usuarios_lector');
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper) THEN RAISE EXCEPTION 'AUT47: PARO clave=operador actual=incompatible esperado=superusuario_PG18' USING ERRCODE='55000';END IF;
 IF g IS NULL OR EXISTS(SELECT 1 FROM pg_roles WHERE oid=g AND(rolcanlogin OR rolinherit OR rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls OR rolconfig IS NOT NULL))
 OR EXISTS(SELECT 1 FROM pg_auth_members WHERE member=g)
 OR EXISTS(SELECT 1 FROM pg_db_role_setting WHERE setrole=g) THEN RAISE EXCEPTION 'AUT47: PARO clave=grupo_lector actual=incompatible esperado=grupo_NOLOGIN_aislado_AUT43' USING ERRCODE='55000';END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=to_regprocedure('vec_autorizacion.listar_usuarios_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') AND p.proowner='vec_autorizacion_propietario'::regrole AND p.prosecdef)
 OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=to_regprocedure('vec_autorizacion.consultar_usuario_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') AND p.proowner='vec_autorizacion_propietario'::regrole AND p.prosecdef)
 OR to_regprocedure('vec_autorizacion.validar_administrador_usuarios_v1(jsonb,jsonb)') IS NULL THEN RAISE EXCEPTION 'AUT47: PARO clave=preimagen actual=sin_AUT43 esperado=AUT43_instalada' USING ERRCODE='55000';END IF;
 -- El lector de Go sólo admite CONNECT sin GRANT OPTION sobre la base: cualquier
 -- otro privilegio directo del grupo es un estado ajeno que no se corrige aquí.
 IF EXISTS(SELECT 1 FROM pg_database d CROSS JOIN LATERAL aclexplode(coalesce(d.datacl,acldefault('d',d.datdba))) a
  WHERE d.datname=current_database() AND a.grantee=g AND(a.privilege_type<>'CONNECT' OR a.is_grantable)) THEN RAISE EXCEPTION 'AUT47: PARO clave=acl_base_grupo actual=privilegio_ajeno esperado=sin_privilegios_o_solo_CONNECT' USING ERRCODE='55000';END IF;
END $pre$;
DO $connect$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_database d CROSS JOIN LATERAL aclexplode(coalesce(d.datacl,acldefault('d',d.datdba))) a
  WHERE d.datname=current_database() AND a.grantee=to_regrole('vec_admin_usuarios_lector') AND a.privilege_type='CONNECT') THEN
  EXECUTE format('GRANT CONNECT ON DATABASE %I TO vec_admin_usuarios_lector',current_database());
 END IF;
END $connect$;
DO $post$
BEGIN
 IF (SELECT count(*) FROM pg_database d CROSS JOIN LATERAL aclexplode(coalesce(d.datacl,acldefault('d',d.datdba))) a
  WHERE d.datname=current_database() AND a.grantee=to_regrole('vec_admin_usuarios_lector'))<>1
 OR NOT EXISTS(SELECT 1 FROM pg_database d CROSS JOIN LATERAL aclexplode(coalesce(d.datacl,acldefault('d',d.datdba))) a
  WHERE d.datname=current_database() AND a.grantee=to_regrole('vec_admin_usuarios_lector') AND a.privilege_type='CONNECT' AND NOT a.is_grantable)
 THEN RAISE EXCEPTION 'AUT47: PARO clave=postcondicion actual=incompatible esperado=solo_CONNECT_sin_grant_option' USING ERRCODE='55000';END IF;
END $post$;
COMMIT;
