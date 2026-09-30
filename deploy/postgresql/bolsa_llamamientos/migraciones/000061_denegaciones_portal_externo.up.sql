\set ON_ERROR_STOP on
-- Bolsa 000061. El LOGIN nominal se provisiona fuera de Git con una sola
-- membresía heredada en el grupo nuevo, sin SET ni ADMIN y DSN propio.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:migracion:000061',0));

DO $pre$ BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=session_user AND rolsuper)
    OR pg_catalog.to_regrole('vec_bolsa_llamamientos_registrador_portal_externo') IS NOT NULL
    OR pg_catalog.to_regrole('vec_bolsa_llamamientos_portal_externo') IS NULL
    OR pg_catalog.to_regrole('vec_bolsa_llamamientos_propietario') IS NULL
    OR pg_catalog.to_regclass('vec_bolsa_llamamientos.denegacion_frontera_portal_externo') IS NOT NULL
    OR pg_catalog.to_regprocedure('vec_bolsa_llamamientos.registrar_denegacion_portal_externo_v1(text,text,text,text,text)') IS NOT NULL
    OR pg_catalog.to_regprocedure('vec_bolsa_llamamientos.constitucion_rechazar_mutacion()') IS NULL
 THEN RAISE EXCEPTION 'Bolsa 000061: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE ROLE vec_bolsa_llamamientos_registrador_portal_externo NOLOGIN NOSUPERUSER
 NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
DO $conexion$ BEGIN
 EXECUTE pg_catalog.format('GRANT CONNECT ON DATABASE %I TO vec_bolsa_llamamientos_registrador_portal_externo',pg_catalog.current_database());
END $conexion$;

SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
CREATE TABLE vec_bolsa_llamamientos.denegacion_frontera_portal_externo (
 evento_id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 correlacion_ref text NOT NULL CHECK(correlacion_ref='corr_no_disponible' OR correlacion_ref ~ '^corr_[0-9a-f]{32}$'),
 motivo text NOT NULL CHECK(motivo IN ('autenticacion_requerida','acceso_denegado')),
 superficie text NOT NULL CHECK(superficie='api.bolsa.candidato.ruta_exacta'),
 ruta text NOT NULL CHECK(ruta IN (
  '/api/vec/bolsa/mi-bolsa',
  '/api/vec/bolsa/mi-bolsa/historial',
  '/api/vec/bolsa/mi-bolsa/solicitudes',
  '/api/vec/bolsa/mi-bolsa/respuestas',
  '/api/vec/bolsa/mi-bolsa/disposiciones',
  '/api/vec/bolsa/mi-bolsa/contacto')),
 actor_ref text CHECK(actor_ref IS NULL OR actor_ref ~ '^per_[A-Za-z0-9_-]{22,128}$'),
 registrada_en timestamptz(6) NOT NULL DEFAULT pg_catalog.clock_timestamp(),
 CHECK((motivo='autenticacion_requerida' AND actor_ref IS NULL)
    OR (motivo='acceso_denegado' AND actor_ref IS NOT NULL))
);
-- Una correlación real representa una petición. Su repetición devuelve 23505
-- y no añade otro evento; corr_no_disponible no identifica una petición.
CREATE UNIQUE INDEX denegacion_portal_externo_correlacion_unica
 ON vec_bolsa_llamamientos.denegacion_frontera_portal_externo(correlacion_ref)
 WHERE correlacion_ref<>'corr_no_disponible';
CREATE TRIGGER denegacion_portal_externo_inmutable BEFORE UPDATE OR DELETE
 ON vec_bolsa_llamamientos.denegacion_frontera_portal_externo FOR EACH ROW
 EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();
CREATE TRIGGER denegacion_portal_externo_no_truncar BEFORE TRUNCATE
 ON vec_bolsa_llamamientos.denegacion_frontera_portal_externo FOR EACH STATEMENT
 EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();
ALTER TABLE vec_bolsa_llamamientos.denegacion_frontera_portal_externo ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.denegacion_frontera_portal_externo FORCE ROW LEVEL SECURITY;
CREATE POLICY denegacion_portal_externo_insertar ON vec_bolsa_llamamientos.denegacion_frontera_portal_externo
 FOR INSERT TO vec_bolsa_llamamientos_propietario WITH CHECK (
   pg_catalog.pg_has_role(session_user,'vec_bolsa_llamamientos_registrador_portal_externo','MEMBER')
   AND superficie='api.bolsa.candidato.ruta_exacta'
   AND ((motivo='autenticacion_requerida' AND actor_ref IS NULL)
     OR (motivo='acceso_denegado' AND actor_ref IS NOT NULL)));
REVOKE ALL ON TABLE vec_bolsa_llamamientos.denegacion_frontera_portal_externo
 FROM PUBLIC,vec_bolsa_llamamientos_ejecutor,vec_bolsa_llamamientos_portal_externo,
 vec_bolsa_llamamientos_registrador_portal_externo;
REVOKE ALL ON SEQUENCE vec_bolsa_llamamientos.denegacion_frontera_portal_externo_evento_id_seq
 FROM PUBLIC,vec_bolsa_llamamientos_ejecutor,vec_bolsa_llamamientos_portal_externo,
 vec_bolsa_llamamientos_registrador_portal_externo;
REVOKE ALL ON TYPE vec_bolsa_llamamientos.denegacion_frontera_portal_externo
 FROM PUBLIC,vec_bolsa_llamamientos_ejecutor,vec_bolsa_llamamientos_portal_externo,
 vec_bolsa_llamamientos_registrador_portal_externo;

CREATE FUNCTION vec_bolsa_llamamientos.registrar_denegacion_portal_externo_v1(
 p_correlacion text,p_motivo text,p_superficie text,p_ruta text,p_actor text)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
 SET lock_timeout='1s' SET statement_timeout='2s' AS $funcion$
DECLARE login pg_catalog.pg_roles%ROWTYPE; grupo pg_catalog.pg_roles%ROWTYPE;
        fachada oid := 'vec_bolsa_llamamientos.registrar_denegacion_portal_externo_v1(text,text,text,text,text)'::pg_catalog.regprocedure;
BEGIN
 SELECT * INTO login FROM pg_catalog.pg_roles WHERE rolname=session_user;
 SELECT * INTO grupo FROM pg_catalog.pg_roles WHERE rolname='vec_bolsa_llamamientos_registrador_portal_externo';
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR session_user=current_user
    OR login.oid IS NULL OR NOT login.rolcanlogin OR NOT login.rolinherit
    OR login.rolsuper OR login.rolcreatedb OR login.rolcreaterole OR login.rolreplication OR login.rolbypassrls
    OR grupo.oid IS NULL OR grupo.rolcanlogin OR NOT grupo.rolinherit
    OR grupo.rolsuper OR grupo.rolcreatedb OR grupo.rolcreaterole OR grupo.rolreplication OR grupo.rolbypassrls
    OR (SELECT count(*) FROM pg_catalog.pg_auth_members m WHERE m.member=login.oid)<>1
    OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.member=login.oid
      AND m.roleid=grupo.oid AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.roleid=login.oid OR m.member=grupo.oid)
    OR NOT pg_catalog.has_schema_privilege(login.oid,'vec_bolsa_llamamientos','USAGE')
    OR pg_catalog.has_schema_privilege(login.oid,'vec_bolsa_llamamientos','CREATE')
    OR NOT pg_catalog.has_function_privilege(login.oid,fachada,'EXECUTE')
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_proc p,
       LATERAL pg_catalog.aclexplode(p.proacl) a WHERE p.oid=fachada
       AND a.privilege_type='EXECUTE' AND
         (a.grantee=0 OR (a.grantee=grupo.oid AND a.is_grantable)))
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_namespace n WHERE n.nspname LIKE 'vec_%'
      AND n.nspname<>'vec_bolsa_llamamientos'
      AND pg_catalog.has_schema_privilege(login.oid,n.oid,'USAGE,CREATE'))
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_namespace n WHERE n.nspname LIKE 'vec_%'
      AND (n.nspowner IN (login.oid,grupo.oid) OR EXISTS
        (SELECT 1 FROM pg_catalog.aclexplode(n.nspacl) a WHERE a.grantee=login.oid
          OR (a.grantee=grupo.oid AND (n.nspname<>'vec_bolsa_llamamientos' OR a.privilege_type<>'USAGE')))))
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
      WHERE n.nspname LIKE 'vec_%' AND (c.relowner IN (login.oid,grupo.oid)
        OR EXISTS (SELECT 1 FROM pg_catalog.aclexplode(c.relacl) a WHERE a.grantee IN (login.oid,grupo.oid))))
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_attribute a JOIN pg_catalog.pg_class c ON c.oid=a.attrelid
      JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname LIKE 'vec_%'
      AND EXISTS (SELECT 1 FROM pg_catalog.aclexplode(a.attacl) x WHERE x.grantee IN (login.oid,grupo.oid)))
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
      WHERE n.nspname LIKE 'vec_%' AND (p.proowner IN (login.oid,grupo.oid)
       OR EXISTS (SELECT 1 FROM pg_catalog.aclexplode(p.proacl) a WHERE a.grantee=login.oid
         OR (a.grantee=grupo.oid AND (p.oid<>fachada OR a.privilege_type<>'EXECUTE')))))
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_type t JOIN pg_catalog.pg_namespace n ON n.oid=t.typnamespace
      WHERE n.nspname LIKE 'vec_%' AND (t.typowner IN (login.oid,grupo.oid)
        OR EXISTS (SELECT 1 FROM pg_catalog.aclexplode(t.typacl) a WHERE a.grantee IN (login.oid,grupo.oid))))
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
      WHERE n.nspname LIKE 'vec_%' AND p.oid<>fachada AND pg_catalog.has_function_privilege(login.oid,p.oid,'EXECUTE'))
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
      WHERE n.nspname LIKE 'vec_%' AND CASE WHEN c.relkind IN ('r','p','v','m','f') THEN
        pg_catalog.has_table_privilege(login.oid,c.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER,MAINTAIN')
        OR pg_catalog.has_any_column_privilege(login.oid,c.oid,'SELECT,INSERT,UPDATE,REFERENCES') ELSE false END)
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
      WHERE n.nspname LIKE 'vec_%' AND CASE WHEN c.relkind='S' THEN
       pg_catalog.has_sequence_privilege(login.oid,c.oid,'USAGE,SELECT,UPDATE') ELSE false END)
    OR pg_catalog.pg_is_in_recovery() OR pg_catalog.current_setting('transaction_read_only')<>'off'
 THEN RAISE EXCEPTION 'Bolsa: registrador exterior invalido' USING ERRCODE='42501'; END IF;
 IF p_correlacion IS NULL OR (p_correlacion<>'corr_no_disponible' AND p_correlacion !~ '^corr_[0-9a-f]{32}$')
    OR p_motivo IS NULL OR p_motivo NOT IN ('autenticacion_requerida','acceso_denegado')
    OR p_superficie IS DISTINCT FROM 'api.bolsa.candidato.ruta_exacta'
    OR p_ruta IS NULL OR p_ruta NOT IN (
      '/api/vec/bolsa/mi-bolsa','/api/vec/bolsa/mi-bolsa/historial',
      '/api/vec/bolsa/mi-bolsa/solicitudes','/api/vec/bolsa/mi-bolsa/respuestas',
      '/api/vec/bolsa/mi-bolsa/disposiciones','/api/vec/bolsa/mi-bolsa/contacto')
    OR (p_motivo='autenticacion_requerida' AND p_actor IS NOT NULL)
    OR (p_motivo='acceso_denegado' AND (p_actor IS NULL OR p_actor !~ '^per_[A-Za-z0-9_-]{22,128}$'))
 THEN RAISE EXCEPTION 'Bolsa: denegacion exterior invalida' USING ERRCODE='22023'; END IF;
 INSERT INTO vec_bolsa_llamamientos.denegacion_frontera_portal_externo
  (correlacion_ref,motivo,superficie,ruta,actor_ref,registrada_en)
 VALUES (p_correlacion,p_motivo,p_superficie,p_ruta,p_actor,
  pg_catalog.date_trunc('microseconds',pg_catalog.clock_timestamp()));
 RETURN true;
END $funcion$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.registrar_denegacion_portal_externo_v1(text,text,text,text,text)
 FROM PUBLIC,vec_bolsa_llamamientos_ejecutor,vec_bolsa_llamamientos_portal_externo,
 vec_bolsa_llamamientos_registrador_frontera,vec_bolsa_llamamientos_migrador;
GRANT USAGE ON SCHEMA vec_bolsa_llamamientos TO vec_bolsa_llamamientos_registrador_portal_externo;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.registrar_denegacion_portal_externo_v1(text,text,text,text,text)
 TO vec_bolsa_llamamientos_registrador_portal_externo;
COMMIT;
