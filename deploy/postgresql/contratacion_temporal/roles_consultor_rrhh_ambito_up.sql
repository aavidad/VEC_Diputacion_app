-- Grupo tecnico independiente del consultor RRHH historico. Ningun LOGIN se
-- crea aqui: Sistemas aprovisiona una identidad nominal con membresia unica.
BEGIN;
SET LOCAL search_path=pg_catalog;
DO $rol$
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
    OR to_regrole('vec_contratacion_temporal_consultor_rrhh') IS NULL
    OR to_regrole('vec_contratacion_temporal_consultor_rrhh_ambito') IS NOT NULL
 THEN RAISE EXCEPTION 'grupo CT ambito: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $rol$;
CREATE ROLE vec_contratacion_temporal_consultor_rrhh_ambito
 NOLOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
DO $connect$
BEGIN
 EXECUTE format('GRANT CONNECT ON DATABASE %I TO vec_contratacion_temporal_consultor_rrhh_ambito',current_database());
END $connect$;
DO $postimagen$
DECLARE r oid:='vec_contratacion_temporal_consultor_rrhh_ambito'::regrole;
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE oid=r AND NOT rolcanlogin
   AND rolinherit AND NOT rolsuper AND NOT rolcreatedb AND NOT rolcreaterole
   AND NOT rolreplication AND NOT rolbypassrls)
   OR EXISTS (SELECT 1 FROM pg_auth_members WHERE member=r OR roleid=r)
   OR has_schema_privilege(r,'vec_contratacion_temporal','USAGE')
   OR has_database_privilege(r,current_database(),'CREATE')
   OR has_database_privilege(r,current_database(),'TEMP')
 THEN RAISE EXCEPTION 'grupo CT ambito: postimagen incompatible' USING ERRCODE='55000'; END IF;
END $postimagen$;
COMMIT;
