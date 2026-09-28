\set ON_ERROR_STOP on
-- Usuarios 000003: depende de roles_000003_up y de 000001/000002. Solo la
-- ruta exacta de preferencias; no consulta ni amplía CT108/CT136.
BEGIN;
SET LOCAL ROLE vec_usuarios_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_usuarios:migracion:000003',0));
DO $pre$ BEGIN
 IF current_user<>'vec_usuarios_propietario'
    OR to_regclass('vec_usuarios.preferencias_recibo') IS NULL
    OR to_regprocedure('vec_usuarios.guardar_preferencias_propias_v1(text,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_usuarios.rechazar_cambio_inmutable()') IS NULL
    OR to_regclass('vec_usuarios.denegacion_frontera_preferencias') IS NOT NULL
    OR to_regprocedure('vec_usuarios.registrar_denegacion_preferencias_v1(text,text,text,text,text)') IS NOT NULL
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_usuarios_registrador_frontera'
      AND NOT rolcanlogin AND NOT rolsuper AND NOT rolcreatedb AND NOT rolcreaterole
      AND rolinherit AND NOT rolreplication AND NOT rolbypassrls)
    OR EXISTS (SELECT 1 FROM pg_auth_members m JOIN pg_roles r ON r.oid=m.member
      WHERE r.rolname='vec_usuarios_registrador_frontera')
    OR has_schema_privilege('vec_usuarios_registrador_frontera','vec_usuarios','CREATE')
    OR has_schema_privilege('vec_usuarios_registrador_frontera','vec_usuarios','USAGE')
    OR EXISTS (SELECT 1 FROM pg_proc p WHERE p.pronamespace='vec_usuarios'::regnamespace
      AND has_function_privilege('vec_usuarios_registrador_frontera',p.oid,'EXECUTE'))
    OR EXISTS (SELECT 1 FROM pg_class c WHERE c.relnamespace='vec_usuarios'::regnamespace
      AND c.relkind IN ('r','p','v','m') AND (has_table_privilege('vec_usuarios_registrador_frontera',c.oid,'SELECT')
       OR has_table_privilege('vec_usuarios_registrador_frontera',c.oid,'INSERT')
       OR has_table_privilege('vec_usuarios_registrador_frontera',c.oid,'UPDATE')
       OR has_table_privilege('vec_usuarios_registrador_frontera',c.oid,'DELETE')
       OR has_table_privilege('vec_usuarios_registrador_frontera',c.oid,'TRUNCATE')
       OR has_table_privilege('vec_usuarios_registrador_frontera',c.oid,'REFERENCES')
       OR has_table_privilege('vec_usuarios_registrador_frontera',c.oid,'TRIGGER')
       OR has_table_privilege('vec_usuarios_registrador_frontera',c.oid,'MAINTAIN')
       OR has_any_column_privilege('vec_usuarios_registrador_frontera',c.oid,'SELECT')
       OR has_any_column_privilege('vec_usuarios_registrador_frontera',c.oid,'INSERT')
       OR has_any_column_privilege('vec_usuarios_registrador_frontera',c.oid,'UPDATE')
       OR has_any_column_privilege('vec_usuarios_registrador_frontera',c.oid,'REFERENCES')))
    OR EXISTS (SELECT 1 FROM pg_class c WHERE c.relnamespace='vec_usuarios'::regnamespace
      AND c.relkind='S' AND (has_sequence_privilege('vec_usuarios_registrador_frontera',c.oid,'USAGE')
       OR has_sequence_privilege('vec_usuarios_registrador_frontera',c.oid,'SELECT')
       OR has_sequence_privilege('vec_usuarios_registrador_frontera',c.oid,'UPDATE')))
 THEN RAISE EXCEPTION 'Usuarios 000003: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE TABLE vec_usuarios.denegacion_frontera_preferencias (
 evento_id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 correlacion_ref text NOT NULL CHECK(correlacion_ref='corr_no_disponible' OR correlacion_ref ~ '^corr_[0-9a-f]{32}$'),
 motivo text NOT NULL CHECK(motivo IN ('autenticacion_requerida','acceso_denegado')),
 superficie text NOT NULL CHECK(superficie='api.usuarios.preferencias.ruta_exacta'),
 ruta text NOT NULL CHECK(ruta='/api/vec/usuarios/mis-preferencias'),
 actor_ref text CHECK(actor_ref IS NULL OR actor_ref ~ '^per_[A-Za-z0-9_-]{22,128}$'),
 registrada_en timestamptz(6) NOT NULL,
 CHECK(motivo<>'autenticacion_requerida' OR actor_ref IS NULL)
);
CREATE INDEX denegacion_frontera_preferencias_correlacion_idx
 ON vec_usuarios.denegacion_frontera_preferencias(correlacion_ref,evento_id);
CREATE TRIGGER denegacion_inmutable BEFORE UPDATE OR DELETE
 ON vec_usuarios.denegacion_frontera_preferencias FOR EACH ROW
 EXECUTE FUNCTION vec_usuarios.rechazar_cambio_inmutable();
CREATE TRIGGER denegacion_no_truncar BEFORE TRUNCATE
 ON vec_usuarios.denegacion_frontera_preferencias FOR EACH STATEMENT
 EXECUTE FUNCTION vec_usuarios.rechazar_cambio_inmutable();
ALTER TABLE vec_usuarios.denegacion_frontera_preferencias ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_usuarios.denegacion_frontera_preferencias FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_lectura ON vec_usuarios.denegacion_frontera_preferencias
 FOR SELECT TO vec_usuarios_propietario USING (true);
CREATE POLICY propietario_insercion ON vec_usuarios.denegacion_frontera_preferencias
 FOR INSERT TO vec_usuarios_propietario WITH CHECK (true);
REVOKE ALL ON TABLE vec_usuarios.denegacion_frontera_preferencias FROM PUBLIC,vec_usuarios_ejecutor;
REVOKE ALL ON SEQUENCE vec_usuarios.denegacion_frontera_preferencias_evento_id_seq FROM PUBLIC,vec_usuarios_ejecutor;

CREATE FUNCTION vec_usuarios.registrar_denegacion_preferencias_v1(
 p_correlacion_ref text,p_motivo text,p_superficie text,p_ruta text,p_actor_ref text)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
 SET lock_timeout='1s' SET statement_timeout='2s' AS $f$
DECLARE login pg_catalog.pg_roles%ROWTYPE; grupo pg_catalog.pg_roles%ROWTYPE;
BEGIN
 SELECT * INTO login FROM pg_catalog.pg_roles WHERE rolname=session_user;
 SELECT * INTO grupo FROM pg_catalog.pg_roles WHERE rolname='vec_usuarios_registrador_frontera';
 IF current_user<>'vec_usuarios_propietario' OR session_user=current_user
    OR login.oid IS NULL OR NOT login.rolcanlogin OR NOT login.rolinherit
    OR login.rolsuper OR login.rolcreatedb OR login.rolcreaterole
    OR login.rolreplication OR login.rolbypassrls
    OR grupo.oid IS NULL OR grupo.rolcanlogin OR grupo.rolsuper
    OR grupo.rolcreatedb OR grupo.rolcreaterole OR NOT grupo.rolinherit
    OR grupo.rolreplication OR grupo.rolbypassrls
    OR (SELECT count(*) FROM pg_catalog.pg_auth_members m WHERE m.member=login.oid)<>1
    OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.member=login.oid
      AND m.roleid=grupo.oid AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.roleid=login.oid OR m.member=grupo.oid)
    -- La cuenta privada no recibe ACL directas ni propiedad en ningún módulo
    -- vec_*. La función nominal le llega únicamente por el grupo exclusivo.
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_namespace n
      WHERE n.nspname ~ '^vec_' AND (n.nspowner=login.oid
        OR EXISTS (SELECT 1 FROM pg_catalog.aclexplode(n.nspacl) a WHERE a.grantee=login.oid)))
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_class c
      JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
      WHERE n.nspname ~ '^vec_' AND (c.relowner=login.oid
        OR EXISTS (SELECT 1 FROM pg_catalog.aclexplode(c.relacl) a WHERE a.grantee=login.oid)))
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_attribute a
      JOIN pg_catalog.pg_class c ON c.oid=a.attrelid
      JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
      WHERE n.nspname ~ '^vec_' AND EXISTS
        (SELECT 1 FROM pg_catalog.aclexplode(a.attacl) x WHERE x.grantee=login.oid))
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_proc p
      JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
      WHERE n.nspname ~ '^vec_' AND (p.proowner=login.oid
        OR EXISTS (SELECT 1 FROM pg_catalog.aclexplode(p.proacl) a WHERE a.grantee=login.oid)))
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_type t
      JOIN pg_catalog.pg_namespace n ON n.oid=t.typnamespace
      WHERE n.nspname ~ '^vec_' AND (t.typowner=login.oid
        OR EXISTS (SELECT 1 FROM pg_catalog.aclexplode(t.typacl) a WHERE a.grantee=login.oid)))
    -- El grupo tampoco puede adquirir otra capacidad directa en VEC.
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_namespace n WHERE n.nspname ~ '^vec_'
      AND (n.nspowner=grupo.oid OR EXISTS
        (SELECT 1 FROM pg_catalog.aclexplode(n.nspacl) a WHERE a.grantee=grupo.oid
         AND (n.nspname<>'vec_usuarios' OR a.privilege_type<>'USAGE'))))
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_class c
      JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
      WHERE n.nspname ~ '^vec_' AND (c.relowner=grupo.oid
        OR EXISTS (SELECT 1 FROM pg_catalog.aclexplode(c.relacl) a WHERE a.grantee=grupo.oid)))
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_attribute a
      JOIN pg_catalog.pg_class c ON c.oid=a.attrelid
      JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
      WHERE n.nspname ~ '^vec_' AND EXISTS
        (SELECT 1 FROM pg_catalog.aclexplode(a.attacl) x WHERE x.grantee=grupo.oid))
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_proc p
      JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
      WHERE n.nspname ~ '^vec_' AND (p.proowner=grupo.oid OR EXISTS
        (SELECT 1 FROM pg_catalog.aclexplode(p.proacl) a WHERE a.grantee=grupo.oid
         AND (p.oid<>'vec_usuarios.registrar_denegacion_preferencias_v1(text,text,text,text,text)'::pg_catalog.regprocedure
           OR a.privilege_type<>'EXECUTE'))))
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_type t
      JOIN pg_catalog.pg_namespace n ON n.oid=t.typnamespace
      WHERE n.nspname ~ '^vec_' AND (t.typowner=grupo.oid
        OR EXISTS (SELECT 1 FROM pg_catalog.aclexplode(t.typacl) a WHERE a.grantee=grupo.oid)))
    OR NOT has_schema_privilege(grupo.oid,'vec_usuarios','USAGE')
    OR has_schema_privilege(grupo.oid,'vec_usuarios','CREATE')
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_proc p WHERE p.pronamespace='vec_usuarios'::pg_catalog.regnamespace
      AND p.oid<>'vec_usuarios.registrar_denegacion_preferencias_v1(text,text,text,text,text)'::regprocedure
      AND has_function_privilege(grupo.oid,p.oid,'EXECUTE'))
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_class c WHERE c.relnamespace='vec_usuarios'::pg_catalog.regnamespace
      AND c.relkind IN ('r','p','v','m') AND (has_table_privilege(grupo.oid,c.oid,'SELECT')
       OR has_table_privilege(grupo.oid,c.oid,'INSERT') OR has_table_privilege(grupo.oid,c.oid,'UPDATE')
       OR has_table_privilege(grupo.oid,c.oid,'DELETE') OR has_table_privilege(grupo.oid,c.oid,'TRUNCATE')
       OR has_table_privilege(grupo.oid,c.oid,'REFERENCES') OR has_table_privilege(grupo.oid,c.oid,'TRIGGER')
       OR has_table_privilege(grupo.oid,c.oid,'MAINTAIN')
       OR has_any_column_privilege(grupo.oid,c.oid,'SELECT')
       OR has_any_column_privilege(grupo.oid,c.oid,'INSERT')
       OR has_any_column_privilege(grupo.oid,c.oid,'UPDATE')
       OR has_any_column_privilege(grupo.oid,c.oid,'REFERENCES')))
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_class c WHERE c.relnamespace='vec_usuarios'::pg_catalog.regnamespace
      AND c.relkind='S' AND (has_sequence_privilege(grupo.oid,c.oid,'USAGE')
       OR has_sequence_privilege(grupo.oid,c.oid,'SELECT')
       OR has_sequence_privilege(grupo.oid,c.oid,'UPDATE')))
    OR pg_is_in_recovery() OR current_setting('transaction_read_only')<>'off'
 THEN RAISE EXCEPTION 'Usuarios: registrador de frontera invalido' USING ERRCODE='42501'; END IF;
 IF p_correlacion_ref IS NULL OR (p_correlacion_ref<>'corr_no_disponible'
       AND p_correlacion_ref !~ '^corr_[0-9a-f]{32}$')
    OR p_motivo IS NULL OR p_motivo NOT IN ('autenticacion_requerida','acceso_denegado')
    OR p_superficie IS DISTINCT FROM 'api.usuarios.preferencias.ruta_exacta'
    OR p_ruta IS DISTINCT FROM '/api/vec/usuarios/mis-preferencias'
    OR (p_motivo='autenticacion_requerida' AND p_actor_ref IS NOT NULL)
    OR (p_actor_ref IS NOT NULL AND p_actor_ref !~ '^per_[A-Za-z0-9_-]{22,128}$')
 THEN RAISE EXCEPTION 'Usuarios: denegacion de frontera invalida' USING ERRCODE='22023'; END IF;
 INSERT INTO vec_usuarios.denegacion_frontera_preferencias
  (correlacion_ref,motivo,superficie,ruta,actor_ref,registrada_en)
 VALUES (p_correlacion_ref,p_motivo,p_superficie,p_ruta,p_actor_ref,
  date_trunc('microseconds',clock_timestamp()));
 RETURN true;
END $f$;
REVOKE ALL ON FUNCTION vec_usuarios.registrar_denegacion_preferencias_v1(text,text,text,text,text)
 FROM PUBLIC,vec_usuarios_ejecutor,vec_usuarios_migrador;
GRANT USAGE ON SCHEMA vec_usuarios TO vec_usuarios_registrador_frontera;
GRANT EXECUTE ON FUNCTION vec_usuarios.registrar_denegacion_preferencias_v1(text,text,text,text,text)
 TO vec_usuarios_registrador_frontera;
COMMIT;
