BEGIN;
SET LOCAL search_path = pg_catalog;

SELECT pg_advisory_xact_lock(hashtextextended('vec_personal.dependencias.alta_ejercicio.v1', 0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal:roles_up:v1', 0));

DO $prevalidacion$
DECLARE propietario oid; migrador oid; ejecutor oid; acl_incompatible boolean;
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = current_user AND rolsuper) THEN
    RAISE EXCEPTION USING ERRCODE = '42501', MESSAGE = 'retirada Personal rechazada: requiere superusuario';
  END IF;
  SELECT oid INTO propietario FROM pg_roles WHERE rolname='vec_personal_propietario';
  SELECT oid INTO migrador FROM pg_roles WHERE rolname='vec_personal_migrador';
  SELECT oid INTO ejecutor FROM pg_roles WHERE rolname='vec_personal_ejecutor';
  IF propietario IS NULL OR migrador IS NULL OR ejecutor IS NULL OR to_regnamespace('vec_personal') IS NULL THEN
    RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'retirada Personal rechazada: instalación incompleta';
  END IF;
  IF (SELECT nspowner FROM pg_namespace WHERE oid='vec_personal'::regnamespace) <> propietario THEN
    RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'retirada Personal rechazada: propietario de esquema alterado';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_roles WHERE oid=propietario AND (rolsuper OR rolcreatedb OR rolcreaterole OR rolcanlogin OR rolinherit OR rolreplication OR rolbypassrls))
     OR EXISTS (SELECT 1 FROM pg_roles WHERE oid=migrador AND (rolsuper OR rolcreatedb OR rolcreaterole OR rolcanlogin OR rolinherit OR rolreplication OR rolbypassrls))
     OR EXISTS (SELECT 1 FROM pg_roles WHERE oid=ejecutor AND (rolsuper OR rolcreatedb OR rolcreaterole OR rolcanlogin OR NOT rolinherit OR rolreplication OR rolbypassrls)) THEN
    RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'retirada Personal rechazada: atributos de rol alterados';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_class WHERE relnamespace='vec_personal'::regnamespace)
     OR EXISTS (SELECT 1 FROM pg_proc WHERE pronamespace='vec_personal'::regnamespace)
     OR EXISTS (SELECT 1 FROM pg_type WHERE typnamespace='vec_personal'::regnamespace AND typtype <> 'p') THEN
    RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'retirada Personal rechazada: el esquema contiene objetos';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_auth_members WHERE (roleid IN (propietario,migrador,ejecutor) OR member IN (propietario,migrador,ejecutor)) AND NOT (roleid=propietario AND member=migrador AND NOT admin_option AND NOT inherit_option AND set_option)) THEN
    RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'retirada Personal rechazada: membresías externas';
  END IF;
  IF (SELECT count(*) FROM pg_auth_members WHERE roleid=propietario AND member=migrador AND NOT admin_option AND NOT inherit_option AND set_option) <> 1 THEN
    RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'retirada Personal rechazada: membresía gestionada ausente';
  END IF;
  IF (SELECT count(*) FROM pg_default_acl WHERE defaclrole=propietario) <> 2
     OR EXISTS (SELECT 1 FROM pg_default_acl WHERE defaclrole=propietario AND (defaclnamespace <> 0 OR defaclobjtype NOT IN ('f','T')))
     OR (SELECT count(*) FROM pg_default_acl d CROSS JOIN LATERAL aclexplode(d.defaclacl) a
        WHERE d.defaclrole=propietario AND ((d.defaclobjtype='f' AND a.grantee=propietario AND a.privilege_type='EXECUTE') OR (d.defaclobjtype='T' AND a.grantee=propietario AND a.privilege_type='USAGE'))) <> 2
     OR EXISTS (SELECT 1 FROM pg_default_acl d CROSS JOIN LATERAL aclexplode(d.defaclacl) a
        WHERE d.defaclrole=propietario AND NOT (
          (d.defaclobjtype='f' AND a.grantee=propietario AND a.privilege_type='EXECUTE') OR
          (d.defaclobjtype='T' AND a.grantee=propietario AND a.privilege_type='USAGE')
        )) THEN
    RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'retirada Personal rechazada: ACL por defecto alterada';
  END IF;
  SELECT EXISTS (SELECT 1 FROM pg_namespace n CROSS JOIN LATERAL aclexplode(n.nspacl) a
    WHERE n.oid='vec_personal'::regnamespace AND NOT (
      (a.grantee=propietario AND a.privilege_type IN ('USAGE','CREATE')) OR
      (a.grantee IN (migrador,ejecutor) AND a.privilege_type='USAGE')
    )) INTO acl_incompatible;
  IF acl_incompatible THEN
    RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'retirada Personal rechazada: ACL ajena';
  END IF;
END $prevalidacion$;

REVOKE USAGE ON SCHEMA vec_personal FROM vec_personal_migrador, vec_personal_ejecutor;
DROP SCHEMA vec_personal;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_personal_propietario GRANT EXECUTE ON FUNCTIONS TO PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_personal_propietario GRANT USAGE ON TYPES TO PUBLIC;
REVOKE vec_personal_propietario FROM vec_personal_migrador;
DROP ROLE vec_personal_ejecutor;
DROP ROLE vec_personal_migrador;
DROP ROLE vec_personal_propietario;
COMMIT;
