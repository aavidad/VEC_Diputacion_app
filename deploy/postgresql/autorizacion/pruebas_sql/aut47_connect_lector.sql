\set ON_ERROR_STOP on
-- AUT47: el grupo lector tiene CONNECT sin GRANT OPTION y ningún otro privilegio
-- de base; un LOGIN sintético miembro del grupo hereda CONNECT. Todo en ROLLBACK.
BEGIN;
SET LOCAL search_path=pg_catalog;
DO $prueba$
DECLARE g oid:=to_regrole('vec_admin_usuarios_lector');
BEGIN
 IF g IS NULL THEN RAISE EXCEPTION 'AUT47: grupo_ausente';END IF;
 IF (SELECT count(*) FROM pg_database d CROSS JOIN LATERAL aclexplode(coalesce(d.datacl,acldefault('d',d.datdba))) a
  WHERE d.datname=current_database() AND a.grantee=g)<>1
 OR NOT EXISTS(SELECT 1 FROM pg_database d CROSS JOIN LATERAL aclexplode(coalesce(d.datacl,acldefault('d',d.datdba))) a
  WHERE d.datname=current_database() AND a.grantee=g AND a.privilege_type='CONNECT' AND NOT a.is_grantable)
 THEN RAISE EXCEPTION 'AUT47: acl_grupo_inesperada';END IF;
 IF NOT has_database_privilege(g,current_database(),'CONNECT') THEN RAISE EXCEPTION 'AUT47: grupo_sin_connect';END IF;
 IF has_database_privilege(g,current_database(),'CREATE') THEN RAISE EXCEPTION 'AUT47: grupo_con_create';END IF;
 IF EXISTS(SELECT 1 FROM pg_roles WHERE oid=g AND(rolcanlogin OR rolinherit)) THEN RAISE EXCEPTION 'AUT47: grupo_no_aislado';END IF;
END $prueba$;
CREATE ROLE aut47_lector_prueba LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
GRANT vec_admin_usuarios_lector TO aut47_lector_prueba WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
DO $miembro$
BEGIN
 IF NOT has_database_privilege('aut47_lector_prueba',current_database(),'CONNECT') THEN RAISE EXCEPTION 'AUT47: miembro_sin_connect';END IF;
 IF has_database_privilege('aut47_lector_prueba',current_database(),'CREATE') THEN RAISE EXCEPTION 'AUT47: miembro_con_create';END IF;
END $miembro$;
ROLLBACK;
