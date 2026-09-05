BEGIN;
SET LOCAL search_path = pg_catalog;
DO $abrir_creacion_base$
BEGIN
    IF NOT EXISTS (
      SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper
    ) THEN
        RAISE EXCEPTION USING ERRCODE='42501',
          MESSAGE='migracion de contexto actor requiere superusuario';
    END IF;
    EXECUTE format(
      'GRANT CREATE ON DATABASE %I TO vec_contexto_actor_v1_propietario',
      current_database()
    );
END
$abrir_creacion_base$;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

CREATE SCHEMA vec_contexto_actor_v1 AUTHORIZATION vec_contexto_actor_v1_propietario;
REVOKE ALL ON SCHEMA vec_contexto_actor_v1 FROM PUBLIC;

ALTER DEFAULT PRIVILEGES FOR ROLE vec_contexto_actor_v1_propietario
    REVOKE ALL ON TABLES FROM PUBLIC, vec_contexto_actor_v1_runtime;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_contexto_actor_v1_propietario
    REVOKE ALL ON SEQUENCES FROM PUBLIC, vec_contexto_actor_v1_runtime;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_contexto_actor_v1_propietario
    REVOKE ALL ON FUNCTIONS FROM PUBLIC, vec_contexto_actor_v1_runtime;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_contexto_actor_v1_propietario
    REVOKE ALL ON TYPES FROM PUBLIC, vec_contexto_actor_v1_runtime;

CREATE FUNCTION vec_contexto_actor_v1.referencia_valida(p_valor text, p_prefijo text)
RETURNS boolean LANGUAGE sql IMMUTABLE SET search_path = pg_catalog AS $f$
    SELECT p_valor IS NOT NULL AND p_prefijo IS NOT NULL
       AND octet_length(p_valor) BETWEEN octet_length(p_prefijo) + 22 AND octet_length(p_prefijo) + 128
       AND left(p_valor, octet_length(p_prefijo)) = p_prefijo
       AND substring(p_valor FROM octet_length(p_prefijo) + 1) ~ '^[A-Za-z0-9_-]+$'
$f$;

CREATE FUNCTION vec_contexto_actor_v1.instante_valido(p_instante timestamptz)
RETURNS boolean LANGUAGE sql IMMUTABLE SET search_path = pg_catalog AS $f$
    SELECT p_instante IS NOT NULL AND isfinite(p_instante)
       AND extract(year FROM p_instante AT TIME ZONE 'UTC') BETWEEN 1 AND 9999
$f$;

CREATE FUNCTION vec_contexto_actor_v1.referencia_operacion_valida(
    p_valor text, p_prefijo text
) RETURNS boolean LANGUAGE sql IMMUTABLE SET search_path = pg_catalog AS $f$
    SELECT p_valor IS NOT NULL AND p_prefijo IN ('oca_','rca_')
       AND octet_length(p_valor) BETWEEN octet_length(p_prefijo)+24 AND octet_length(p_prefijo)+128
       AND left(p_valor,octet_length(p_prefijo))=p_prefijo
       AND substring(p_valor FROM octet_length(p_prefijo)+1) ~ '^[A-Za-z0-9_-]+$'
$f$;

CREATE FUNCTION vec_contexto_actor_v1.procedencia_valida(
    p_ref text, p_version numeric, p_huella text, p_autoridad text
) RETURNS boolean LANGUAGE sql IMMUTABLE SET search_path = pg_catalog AS $f$
    SELECT vec_contexto_actor_v1.referencia_valida(p_ref,'prc_')
       AND p_version BETWEEN 1 AND 18446744073709551615::numeric
       AND p_huella ~ '^[0-9a-f]{64}$'
       AND p_autoridad IN ('autoridad_maestra_acreditada','no_autoritativa')
$f$;

CREATE FUNCTION vec_contexto_actor_v1.rechazar_mutacion_historia()
RETURNS trigger LANGUAGE plpgsql SET search_path = pg_catalog AS $f$
BEGIN
    IF TG_OP <> 'INSERT' THEN
        RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'historia de contexto actor V1 inmutable';
    END IF;
    RETURN NEW;
END
$f$;

CREATE FUNCTION vec_contexto_actor_v1.rechazar_truncado()
RETURNS trigger LANGUAGE plpgsql SET search_path = pg_catalog AS $f$
BEGIN
    RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'TRUNCATE de contexto actor V1 rechazado';
END
$f$;

CREATE TABLE vec_contexto_actor_v1.procedencias (
    procedencia_ref text NOT NULL,
    procedencia_version numeric(20,0) NOT NULL,
    procedencia_huella_sha256 text NOT NULL,
    procedencia_autoridad text NOT NULL,
    PRIMARY KEY (procedencia_ref,procedencia_version),
    UNIQUE (procedencia_ref,procedencia_version,procedencia_huella_sha256,procedencia_autoridad),
    CHECK (vec_contexto_actor_v1.procedencia_valida(
      procedencia_ref,procedencia_version,procedencia_huella_sha256,procedencia_autoridad))
);

CREATE FUNCTION vec_contexto_actor_v1.validar_procedencia_monotona()
RETURNS trigger LANGUAGE plpgsql SET search_path = pg_catalog AS $f$
BEGIN
    PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
      'vec_contexto_actor_v1:procedencia:v1:' || NEW.procedencia_ref,0));
    IF EXISTS (
      SELECT 1 FROM vec_contexto_actor_v1.procedencias p
       WHERE p.procedencia_ref=NEW.procedencia_ref
         AND p.procedencia_version>=NEW.procedencia_version
    ) THEN
      RAISE EXCEPTION USING ERRCODE='23505',
        MESSAGE='revision de procedencia no monotona';
    END IF;
    RETURN NEW;
END
$f$;
CREATE TRIGGER procedencia_monotona BEFORE INSERT ON vec_contexto_actor_v1.procedencias
FOR EACH ROW EXECUTE FUNCTION vec_contexto_actor_v1.validar_procedencia_monotona();

-- Proyeccion tecnica de una cuenta ya acreditada por la frontera de identidad.
-- No es una fuente maestra ni admite identificadores civiles. procedencia_ref
-- permite ligar cada version a la carga gobernada que la materializo.
CREATE TABLE vec_contexto_actor_v1.proyeccion_cuenta_versiones (
    cuenta_ref text NOT NULL,
    version numeric(20,0) NOT NULL CHECK (version BETWEEN 1 AND 18446744073709551615::numeric),
    procedencia_ref text NOT NULL,
    procedencia_version numeric(20,0) NOT NULL,
    procedencia_huella_sha256 text NOT NULL,
    procedencia_autoridad text NOT NULL,
    estado text NOT NULL CHECK (estado IN ('activo', 'revocado')),
    vigente_desde timestamptz NOT NULL,
    vigente_hasta timestamptz NOT NULL,
    PRIMARY KEY (cuenta_ref, version),
    CHECK (vec_contexto_actor_v1.referencia_valida(cuenta_ref, 'cta_')),
    CHECK (vec_contexto_actor_v1.procedencia_valida(
      procedencia_ref,procedencia_version,procedencia_huella_sha256,procedencia_autoridad)),
    CHECK (vec_contexto_actor_v1.instante_valido(vigente_desde)),
    CHECK (vec_contexto_actor_v1.instante_valido(vigente_hasta)),
    CHECK (vigente_hasta > vigente_desde)
    ,FOREIGN KEY (procedencia_ref,procedencia_version,procedencia_huella_sha256,procedencia_autoridad)
      REFERENCES vec_contexto_actor_v1.procedencias(procedencia_ref,procedencia_version,procedencia_huella_sha256,procedencia_autoridad)
);
CREATE TABLE vec_contexto_actor_v1.proyeccion_cuenta_actual (
    cuenta_ref text PRIMARY KEY,
    version numeric(20,0) NOT NULL,
    FOREIGN KEY (cuenta_ref, version) REFERENCES vec_contexto_actor_v1.proyeccion_cuenta_versiones(cuenta_ref, version)
);

CREATE TABLE vec_contexto_actor_v1.persona_versiones (
    persona_ref text NOT NULL,
    version numeric(20,0) NOT NULL CHECK (version BETWEEN 1 AND 18446744073709551615::numeric),
    procedencia_ref text NOT NULL,
    procedencia_version numeric(20,0) NOT NULL,
    procedencia_huella_sha256 text NOT NULL,
    procedencia_autoridad text NOT NULL,
    estado text NOT NULL CHECK (estado IN ('activo', 'revocado')),
    vigente_desde timestamptz NOT NULL,
    vigente_hasta timestamptz NOT NULL,
    PRIMARY KEY (persona_ref, version),
    CHECK (vec_contexto_actor_v1.referencia_valida(persona_ref, 'per_')),
    CHECK (vec_contexto_actor_v1.procedencia_valida(
      procedencia_ref,procedencia_version,procedencia_huella_sha256,procedencia_autoridad)),
    CHECK (vec_contexto_actor_v1.instante_valido(vigente_desde)),
    CHECK (vec_contexto_actor_v1.instante_valido(vigente_hasta)),
    CHECK (vigente_hasta > vigente_desde)
    ,FOREIGN KEY (procedencia_ref,procedencia_version,procedencia_huella_sha256,procedencia_autoridad)
      REFERENCES vec_contexto_actor_v1.procedencias(procedencia_ref,procedencia_version,procedencia_huella_sha256,procedencia_autoridad)
);
CREATE TABLE vec_contexto_actor_v1.persona_actual (
    persona_ref text PRIMARY KEY,
    version numeric(20,0) NOT NULL,
    FOREIGN KEY (persona_ref, version) REFERENCES vec_contexto_actor_v1.persona_versiones(persona_ref, version)
);

CREATE TABLE vec_contexto_actor_v1.perfil_versiones (
    perfil_ref text NOT NULL,
    version numeric(20,0) NOT NULL CHECK (version BETWEEN 1 AND 18446744073709551615::numeric),
    persona_ref text NOT NULL,
    procedencia_ref text NOT NULL,
    procedencia_version numeric(20,0) NOT NULL,
    procedencia_huella_sha256 text NOT NULL,
    procedencia_autoridad text NOT NULL,
    estado text NOT NULL CHECK (estado IN ('activo', 'revocado')),
    vigente_desde timestamptz NOT NULL,
    vigente_hasta timestamptz NOT NULL,
    PRIMARY KEY (perfil_ref, version),
    CHECK (vec_contexto_actor_v1.referencia_valida(perfil_ref, 'prf_')),
    CHECK (vec_contexto_actor_v1.referencia_valida(persona_ref, 'per_')),
    CHECK (vec_contexto_actor_v1.procedencia_valida(
      procedencia_ref,procedencia_version,procedencia_huella_sha256,procedencia_autoridad)),
    CHECK (vec_contexto_actor_v1.instante_valido(vigente_desde)),
    CHECK (vec_contexto_actor_v1.instante_valido(vigente_hasta)),
    CHECK (vigente_hasta > vigente_desde)
    ,FOREIGN KEY (procedencia_ref,procedencia_version,procedencia_huella_sha256,procedencia_autoridad)
      REFERENCES vec_contexto_actor_v1.procedencias(procedencia_ref,procedencia_version,procedencia_huella_sha256,procedencia_autoridad)
);
CREATE TABLE vec_contexto_actor_v1.perfil_actual (
    perfil_ref text PRIMARY KEY,
    version numeric(20,0) NOT NULL,
    FOREIGN KEY (perfil_ref, version) REFERENCES vec_contexto_actor_v1.perfil_versiones(perfil_ref, version)
);

CREATE TABLE vec_contexto_actor_v1.vinculo_contexto_versiones (
    vinculo_ref text NOT NULL,
    version numeric(20,0) NOT NULL CHECK (version BETWEEN 1 AND 18446744073709551615::numeric),
    cuenta_ref text NOT NULL,
    perfil_ref text NOT NULL,
    persona_ref text NOT NULL,
    procedencia_ref text NOT NULL,
    procedencia_version numeric(20,0) NOT NULL,
    procedencia_huella_sha256 text NOT NULL,
    procedencia_autoridad text NOT NULL,
    estado text NOT NULL CHECK (estado IN ('activo', 'revocado')),
    vigente_desde timestamptz NOT NULL,
    vigente_hasta timestamptz NOT NULL,
    PRIMARY KEY (vinculo_ref, version),
    CHECK (vec_contexto_actor_v1.referencia_valida(vinculo_ref, 'vca_')),
    CHECK (vec_contexto_actor_v1.referencia_valida(cuenta_ref, 'cta_')),
    CHECK (vec_contexto_actor_v1.referencia_valida(perfil_ref, 'prf_')),
    CHECK (vec_contexto_actor_v1.referencia_valida(persona_ref, 'per_')),
    CHECK (vec_contexto_actor_v1.procedencia_valida(
      procedencia_ref,procedencia_version,procedencia_huella_sha256,procedencia_autoridad)),
    CHECK (vec_contexto_actor_v1.instante_valido(vigente_desde)),
    CHECK (vec_contexto_actor_v1.instante_valido(vigente_hasta)),
    CHECK (vigente_hasta > vigente_desde)
    ,FOREIGN KEY (procedencia_ref,procedencia_version,procedencia_huella_sha256,procedencia_autoridad)
      REFERENCES vec_contexto_actor_v1.procedencias(procedencia_ref,procedencia_version,procedencia_huella_sha256,procedencia_autoridad)
);
CREATE TABLE vec_contexto_actor_v1.vinculo_contexto_actual (
    vinculo_ref text PRIMARY KEY,
    version numeric(20,0) NOT NULL,
    FOREIGN KEY (vinculo_ref, version) REFERENCES vec_contexto_actor_v1.vinculo_contexto_versiones(vinculo_ref, version)
);
CREATE INDEX vinculo_contexto_resolucion_idx
    ON vec_contexto_actor_v1.vinculo_contexto_versiones(cuenta_ref, perfil_ref, vinculo_ref, version);

CREATE TABLE vec_contexto_actor_v1.vinculo_referencia_versiones (
    vinculo_ref text NOT NULL,
    version numeric(20,0) NOT NULL CHECK (version BETWEEN 1 AND 18446744073709551615::numeric),
    persona_ref text NOT NULL,
    tipo text NOT NULL CHECK (tipo IN ('candidato', 'empleado')),
    referencia text NOT NULL,
    procedencia_ref text NOT NULL,
    procedencia_version numeric(20,0) NOT NULL,
    procedencia_huella_sha256 text NOT NULL,
    procedencia_autoridad text NOT NULL,
    estado text NOT NULL CHECK (estado IN ('activo', 'revocado')),
    vigente_desde timestamptz NOT NULL,
    vigente_hasta timestamptz NOT NULL,
    PRIMARY KEY (vinculo_ref, version),
    CHECK (vec_contexto_actor_v1.referencia_valida(vinculo_ref, 'vin_')),
    CHECK (vec_contexto_actor_v1.referencia_valida(persona_ref, 'per_')),
    CHECK ((tipo = 'candidato' AND vec_contexto_actor_v1.referencia_valida(referencia, 'can_')) OR
           (tipo = 'empleado' AND vec_contexto_actor_v1.referencia_valida(referencia, 'emp_'))),
    CHECK (vec_contexto_actor_v1.procedencia_valida(
      procedencia_ref,procedencia_version,procedencia_huella_sha256,procedencia_autoridad)),
    CHECK (vec_contexto_actor_v1.instante_valido(vigente_desde)),
    CHECK (vec_contexto_actor_v1.instante_valido(vigente_hasta)),
    CHECK (vigente_hasta > vigente_desde)
    ,FOREIGN KEY (procedencia_ref,procedencia_version,procedencia_huella_sha256,procedencia_autoridad)
      REFERENCES vec_contexto_actor_v1.procedencias(procedencia_ref,procedencia_version,procedencia_huella_sha256,procedencia_autoridad)
);
CREATE TABLE vec_contexto_actor_v1.vinculo_referencia_actual (
    vinculo_ref text PRIMARY KEY,
    version numeric(20,0) NOT NULL,
    FOREIGN KEY (vinculo_ref, version) REFERENCES vec_contexto_actor_v1.vinculo_referencia_versiones(vinculo_ref, version)
);
CREATE INDEX vinculo_referencia_persona_idx
    ON vec_contexto_actor_v1.vinculo_referencia_versiones(persona_ref, vinculo_ref, version);

CREATE TABLE vec_contexto_actor_v1.registros_contexto (
    operacion_ref text PRIMARY KEY,
    registro_contexto_ref text NOT NULL UNIQUE,
    cuenta_ref text NOT NULL,
    perfil_ref text NOT NULL,
    metodo text NOT NULL,
    garantia text NOT NULL,
    solicitado_en timestamptz NOT NULL,
    resuelto_en timestamptz NOT NULL,
    representacion_canonica bytea NOT NULL,
    huella_sha256 text NOT NULL,
    manifiesto_procedencia_canonico bytea NOT NULL,
    manifiesto_procedencia_huella_sha256 text NOT NULL,
    autoridad_efectiva text NOT NULL CHECK (autoridad_efectiva='autoridad_maestra_acreditada'),
    CHECK (vec_contexto_actor_v1.referencia_operacion_valida(operacion_ref, 'oca_')),
    CHECK (vec_contexto_actor_v1.referencia_operacion_valida(registro_contexto_ref, 'rca_')),
    CHECK (metodo IN ('certificado','dnie','sso','clave','kerberos_ad','demo')),
    CHECK (garantia IN ('bajo','sustancial','alto')),
    CHECK (octet_length(representacion_canonica) BETWEEN 1 AND 65536),
    CHECK (huella_sha256 ~ '^[0-9a-f]{64}$'),
    CHECK (encode(pg_catalog.sha256(representacion_canonica), 'hex') = huella_sha256),
    CHECK (octet_length(manifiesto_procedencia_canonico) BETWEEN 1 AND 65536),
    CHECK (manifiesto_procedencia_huella_sha256 ~ '^[0-9a-f]{64}$'),
    CHECK (encode(pg_catalog.sha256(manifiesto_procedencia_canonico), 'hex') = manifiesto_procedencia_huella_sha256)
);

DO $triggers$
DECLARE tabla text;
BEGIN
    FOREACH tabla IN ARRAY ARRAY[
        'procedencias','proyeccion_cuenta_versiones','persona_versiones','perfil_versiones',
        'vinculo_contexto_versiones','vinculo_referencia_versiones','registros_contexto'
    ] LOOP
        EXECUTE format('CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_contexto_actor_v1.%I FOR EACH ROW EXECUTE FUNCTION vec_contexto_actor_v1.rechazar_mutacion_historia()', tabla);
        EXECUTE format('CREATE TRIGGER historia_no_truncable BEFORE TRUNCATE ON vec_contexto_actor_v1.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_contexto_actor_v1.rechazar_truncado()', tabla);
    END LOOP;
END
$triggers$;

-- Comprueba privilegios efectivos, incluidos los heredados de PUBLIC o de
-- membresias. Solo excluye catalogos/esquemas internos de PostgreSQL; cualquier
-- superficie definida por usuarios fuera del manifiesto cerrado deniega el
-- LOGIN aunque no exista una concesion directa visible en pg_shdepend.
CREATE FUNCTION vec_contexto_actor_v1.privilegios_efectivos_runtime_minimos(
    p_login_oid oid, p_base_oid oid, p_esquema_oid oid, p_funciones oid[]
) RETURNS boolean LANGUAGE sql STABLE SET search_path = pg_catalog AS $f$
    SELECT pg_catalog.has_database_privilege(p_login_oid,p_base_oid,'CONNECT')
       AND NOT pg_catalog.has_database_privilege(p_login_oid,p_base_oid,'CREATE')
       AND NOT pg_catalog.has_database_privilege(p_login_oid,p_base_oid,'TEMPORARY')
       AND NOT EXISTS (
         SELECT 1 FROM pg_catalog.pg_namespace n
          WHERE n.nspname <> 'information_schema' AND n.nspname !~ '^pg_'
            AND ((n.oid <> p_esquema_oid
                  AND pg_catalog.has_schema_privilege(p_login_oid,n.oid,'USAGE'))
                 OR pg_catalog.has_schema_privilege(p_login_oid,n.oid,'CREATE'))
       )
       AND NOT EXISTS (
         SELECT 1 FROM pg_catalog.pg_class c
         JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
          WHERE n.nspname <> 'information_schema' AND n.nspname !~ '^pg_'
            AND c.relkind IN ('r','p','v','m','f') AND (
              pg_catalog.has_table_privilege(p_login_oid,c.oid,'SELECT') OR
              pg_catalog.has_table_privilege(p_login_oid,c.oid,'INSERT') OR
              pg_catalog.has_table_privilege(p_login_oid,c.oid,'UPDATE') OR
              pg_catalog.has_table_privilege(p_login_oid,c.oid,'DELETE') OR
              pg_catalog.has_table_privilege(p_login_oid,c.oid,'TRUNCATE') OR
              pg_catalog.has_table_privilege(p_login_oid,c.oid,'REFERENCES') OR
              pg_catalog.has_table_privilege(p_login_oid,c.oid,'TRIGGER') OR
              pg_catalog.has_table_privilege(p_login_oid,c.oid,'MAINTAIN') OR
              pg_catalog.has_any_column_privilege(p_login_oid,c.oid,'SELECT') OR
              pg_catalog.has_any_column_privilege(p_login_oid,c.oid,'INSERT') OR
              pg_catalog.has_any_column_privilege(p_login_oid,c.oid,'UPDATE') OR
              pg_catalog.has_any_column_privilege(p_login_oid,c.oid,'REFERENCES'))
       )
       AND NOT EXISTS (
         SELECT 1 FROM pg_catalog.pg_class c
         JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
          WHERE n.nspname <> 'information_schema' AND n.nspname !~ '^pg_'
            AND c.relkind='S' AND (
              pg_catalog.has_sequence_privilege(p_login_oid,c.oid,'USAGE') OR
              pg_catalog.has_sequence_privilege(p_login_oid,c.oid,'SELECT') OR
              pg_catalog.has_sequence_privilege(p_login_oid,c.oid,'UPDATE'))
       )
       AND NOT EXISTS (
         SELECT 1 FROM pg_catalog.pg_proc p
         JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
          WHERE n.nspname <> 'information_schema' AND n.nspname !~ '^pg_'
            AND p.oid <> ALL(p_funciones)
            AND pg_catalog.has_function_privilege(p_login_oid,p.oid,'EXECUTE')
       )
       AND NOT EXISTS (
         SELECT 1 FROM pg_catalog.pg_type t
         JOIN pg_catalog.pg_namespace n ON n.oid=t.typnamespace
          WHERE n.nspname <> 'information_schema' AND n.nspname !~ '^pg_'
            AND t.typtype IN ('c','d','e','m','r')
            AND pg_catalog.has_type_privilege(p_login_oid,t.oid,'USAGE')
            -- PostgreSQL atribuye USAGE implícito al tipo fila de una tabla.
            -- Sin ACL propia ni acceso a su esquema, no concede acceso a datos.
            -- Los tipos independientes y cualquier concesión explícita siguen
            -- rechazados, también si pertenecen a otro esquema cerrado.
            AND NOT (
              t.typtype='c' AND t.typacl IS NULL
              AND NOT pg_catalog.has_schema_privilege(p_login_oid,n.oid,'USAGE')
              AND EXISTS (
                SELECT 1 FROM pg_catalog.pg_class relacion
                 WHERE relacion.oid=t.typrelid
                   AND relacion.relkind IN ('r','p','v','m','f')
              )
            )
       )
       AND NOT EXISTS (
         SELECT 1 FROM pg_catalog.pg_largeobject_metadata l
          WHERE pg_catalog.has_largeobject_privilege(p_login_oid,l.oid,'SELECT')
             OR pg_catalog.has_largeobject_privilege(p_login_oid,l.oid,'UPDATE')
       )
       AND NOT EXISTS (
         SELECT 1 FROM pg_catalog.pg_foreign_data_wrapper f
          WHERE pg_catalog.has_foreign_data_wrapper_privilege(p_login_oid,f.oid,'USAGE')
       )
       AND NOT EXISTS (
         SELECT 1 FROM pg_catalog.pg_foreign_server s
          WHERE pg_catalog.has_server_privilege(p_login_oid,s.oid,'USAGE')
       )
       AND NOT EXISTS (
         SELECT 1 FROM pg_catalog.pg_language l
          WHERE l.oid >= 16384
            AND pg_catalog.has_language_privilege(p_login_oid,l.oid,'USAGE')
       )
       AND NOT EXISTS (
         SELECT 1 FROM pg_catalog.pg_tablespace t
          WHERE t.oid >= 16384
            AND pg_catalog.has_tablespace_privilege(p_login_oid,t.oid,'CREATE')
       )
       AND NOT EXISTS (
         SELECT 1 FROM pg_catalog.pg_parameter_acl a
          WHERE pg_catalog.has_parameter_privilege(p_login_oid,a.parname,'SET')
             OR pg_catalog.has_parameter_privilege(p_login_oid,a.parname,'ALTER SYSTEM')
       )
$f$;

CREATE FUNCTION vec_contexto_actor_v1.exigir_runtime_contexto_actor_v1()
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

CREATE FUNCTION vec_contexto_actor_v1.acreditar_runtime_contexto_actor_v1(
    OUT identidad_login text, OUT acreditada boolean
) RETURNS record LANGUAGE plpgsql SECURITY DEFINER SET search_path = pg_catalog AS $f$
BEGIN
    identidad_login := vec_contexto_actor_v1.exigir_runtime_contexto_actor_v1();
    acreditada := true;
END
$f$;

CREATE FUNCTION vec_contexto_actor_v1.reconciliar_contexto_actor_v2(
    p_operacion_ref text, p_registro_contexto_ref text, p_cuenta_ref text,
    p_perfil_ref text, p_metodo text, p_garantia text, p_solicitado_en timestamptz
) RETURNS TABLE (
    operacion_ref text, registro_contexto_ref text, representacion_canonica bytea,
    huella_sha256 text, manifiesto_procedencia_canonico bytea,
    manifiesto_procedencia_huella_sha256 text, autoridad_efectiva text,
    resuelto_en timestamptz
) LANGUAGE plpgsql SECURITY DEFINER SET search_path = pg_catalog AS $f$
BEGIN
    PERFORM vec_contexto_actor_v1.exigir_runtime_contexto_actor_v1();
    IF current_setting('transaction_isolation') <> 'read committed'
       OR current_setting('transaction_read_only') <> 'off' THEN
        RAISE EXCEPTION USING ERRCODE = '25000',
            MESSAGE = 'reconciliacion de contexto actor V2 requiere READ COMMITTED de escritura';
    END IF;
    IF vec_contexto_actor_v1.referencia_operacion_valida(p_operacion_ref, 'oca_') IS NOT TRUE
       OR vec_contexto_actor_v1.referencia_operacion_valida(p_registro_contexto_ref, 'rca_') IS NOT TRUE
       OR vec_contexto_actor_v1.referencia_valida(p_cuenta_ref, 'cta_') IS NOT TRUE
       OR vec_contexto_actor_v1.referencia_valida(p_perfil_ref, 'prf_') IS NOT TRUE
       OR p_metodo NOT IN ('certificado','dnie','sso','clave','kerberos_ad','demo')
       OR p_garantia NOT IN ('bajo','sustancial','alto')
       OR vec_contexto_actor_v1.instante_valido(p_solicitado_en) IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'reconciliacion de contexto actor V2 invalida';
    END IF;
    -- READ COMMITTED es deliberado: tras esperar al mismo lock que la
    -- escritura, la consulta siguiente adquiere un snapshot nuevo y puede ver
    -- su COMMIT. SERIALIZABLE podria conservar un snapshot anterior al lock y
    -- concluir ausencia mientras la finalizacion concurrente ya fue durable.
    PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
      'vec_contexto_actor_v1:operacion:v2:' || p_operacion_ref,0));
    RETURN QUERY
    SELECT r.operacion_ref, r.registro_contexto_ref, r.representacion_canonica,
           r.huella_sha256,r.manifiesto_procedencia_canonico,
           r.manifiesto_procedencia_huella_sha256,r.autoridad_efectiva,r.resuelto_en
      FROM vec_contexto_actor_v1.registros_contexto AS r
     WHERE r.operacion_ref = p_operacion_ref
       AND r.registro_contexto_ref = p_registro_contexto_ref
       AND r.cuenta_ref = p_cuenta_ref AND r.perfil_ref = p_perfil_ref
       AND r.metodo = p_metodo AND r.garantia = p_garantia
       AND r.solicitado_en = p_solicitado_en;
END
$f$;

CREATE FUNCTION vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2(
    p_operacion_ref text, p_registro_contexto_ref text, p_cuenta_ref text,
    p_perfil_ref text, p_metodo text, p_garantia text, p_solicitado_en timestamptz
) RETURNS TABLE (
    operacion_ref text, registro_contexto_ref text, representacion_canonica bytea,
    huella_sha256 text, manifiesto_procedencia_canonico bytea,
    manifiesto_procedencia_huella_sha256 text, autoridad_efectiva text,
    resuelto_en timestamptz
) LANGUAGE plpgsql SECURITY DEFINER SET search_path = pg_catalog AS $f$
DECLARE
    ahora timestamptz; coincidencias integer; cuenta record; perfil record;
    persona record; enlace record; enlaces_texto text; documento text;
    enlaces_procedencia_texto text; manifiesto_procedencia text;
    manifiesto_procedencia_bytes bytea; manifiesto_procedencia_huella text;
    canonica bytea; huella text; numero_enlaces integer; tipos integer; referencias integer;
BEGIN
    PERFORM vec_contexto_actor_v1.exigir_runtime_contexto_actor_v1();
    IF current_setting('transaction_isolation') <> 'serializable'
       OR current_setting('transaction_read_only') <> 'off' THEN
        RAISE EXCEPTION USING ERRCODE = '25000',
            MESSAGE = 'registro de contexto actor V2 requiere SERIALIZABLE de escritura';
    END IF;
    IF vec_contexto_actor_v1.referencia_operacion_valida(p_operacion_ref, 'oca_') IS NOT TRUE
       OR vec_contexto_actor_v1.referencia_operacion_valida(p_registro_contexto_ref, 'rca_') IS NOT TRUE
       OR vec_contexto_actor_v1.referencia_valida(p_cuenta_ref, 'cta_') IS NOT TRUE
       OR vec_contexto_actor_v1.referencia_valida(p_perfil_ref, 'prf_') IS NOT TRUE
       OR p_metodo NOT IN ('certificado','dnie','sso','clave','kerberos_ad','demo')
       OR p_garantia NOT IN ('bajo','sustancial','alto')
       OR vec_contexto_actor_v1.instante_valido(p_solicitado_en) IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'solicitud de contexto actor V2 invalida';
    END IF;

    -- El advisory estable serializa la identidad de operacion sin bloquear
    -- actores independientes. El adaptador repite SERIALIZABLE con el mismo
    -- oca_/rca_ ante un snapshot concurrente que termine en 23505/40001.
    PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
      'vec_contexto_actor_v1:operacion:v2:' || p_operacion_ref,0));

    -- La entrada normal es idempotente por operacion y solicitud. El rca_ ya
    -- persistido se conserva; p_registro_contexto_ref solo se usa al crear.
    -- La funcion separada de reconciliacion si exige ese rca_ exacto porque
    -- coteja la respuesta observada antes de un COMMIT ambiguo.
    IF EXISTS (SELECT 1 FROM vec_contexto_actor_v1.registros_contexto r WHERE r.operacion_ref = p_operacion_ref) THEN
        RETURN QUERY
        SELECT r.operacion_ref,r.registro_contexto_ref,r.representacion_canonica,
               r.huella_sha256,r.manifiesto_procedencia_canonico,
               r.manifiesto_procedencia_huella_sha256,r.autoridad_efectiva,r.resuelto_en
          FROM vec_contexto_actor_v1.registros_contexto r
         WHERE r.operacion_ref=p_operacion_ref AND r.cuenta_ref=p_cuenta_ref
           AND r.perfil_ref=p_perfil_ref AND r.metodo=p_metodo
           AND r.garantia=p_garantia AND r.solicitado_en=p_solicitado_en;
        IF NOT FOUND THEN
            RAISE EXCEPTION USING ERRCODE = '23505', MESSAGE = 'colision de operacion de contexto actor V2';
        END IF;
        RETURN;
    END IF;

    -- Locks relevantes y deterministas: cuenta exacta, perfil exacto,
    -- candidatos vca_, personas derivadas y referencias de esas personas.
    -- Ninguna referencia se deriva de DNI, certificado, rol o claim libre.
    PERFORM 1 FROM vec_contexto_actor_v1.proyeccion_cuenta_actual ca
     WHERE ca.cuenta_ref=p_cuenta_ref ORDER BY ca.cuenta_ref FOR UPDATE OF ca;
    PERFORM 1 FROM vec_contexto_actor_v1.perfil_actual pa
     WHERE pa.perfil_ref=p_perfil_ref ORDER BY pa.perfil_ref FOR UPDATE OF pa;
    PERFORM 1
      FROM vec_contexto_actor_v1.vinculo_contexto_actual va
      JOIN vec_contexto_actor_v1.vinculo_contexto_versiones vv USING (vinculo_ref,version)
     WHERE vv.cuenta_ref=p_cuenta_ref AND vv.perfil_ref=p_perfil_ref
     ORDER BY va.vinculo_ref FOR UPDATE OF va;
    PERFORM 1 FROM vec_contexto_actor_v1.persona_actual pe
     WHERE pe.persona_ref IN (
       SELECT pv.persona_ref FROM vec_contexto_actor_v1.perfil_actual pa
       JOIN vec_contexto_actor_v1.perfil_versiones pv USING (perfil_ref,version)
       WHERE pa.perfil_ref=p_perfil_ref
       UNION
       SELECT vv.persona_ref FROM vec_contexto_actor_v1.vinculo_contexto_actual va
       JOIN vec_contexto_actor_v1.vinculo_contexto_versiones vv USING (vinculo_ref,version)
       WHERE vv.cuenta_ref=p_cuenta_ref AND vv.perfil_ref=p_perfil_ref
     ) ORDER BY pe.persona_ref FOR UPDATE OF pe;
    PERFORM 1
      FROM vec_contexto_actor_v1.vinculo_referencia_actual ra
      JOIN vec_contexto_actor_v1.vinculo_referencia_versiones rv USING (vinculo_ref,version)
     WHERE rv.persona_ref IN (
       SELECT pv.persona_ref FROM vec_contexto_actor_v1.perfil_actual pa
       JOIN vec_contexto_actor_v1.perfil_versiones pv USING (perfil_ref,version)
       WHERE pa.perfil_ref=p_perfil_ref
       UNION
       SELECT vv.persona_ref FROM vec_contexto_actor_v1.vinculo_contexto_actual va
       JOIN vec_contexto_actor_v1.vinculo_contexto_versiones vv USING (vinculo_ref,version)
       WHERE vv.cuenta_ref=p_cuenta_ref AND vv.perfil_ref=p_perfil_ref
     ) ORDER BY ra.vinculo_ref FOR UPDATE OF ra;

    -- Este es el primer y unico reloj autoritativo. Toda lectura de negocio que
    -- sigue relee los punteros ya bloqueados y usa ventanas [desde,hasta).
    ahora := pg_catalog.clock_timestamp();
    IF ahora < p_solicitado_en OR ahora > p_solicitado_en + interval '5 seconds' THEN
        RAISE EXCEPTION USING ERRCODE = '57014', MESSAGE = 'ventana fresca de contexto actor V2 agotada';
    END IF;

    SELECT count(*) INTO coincidencias
      FROM vec_contexto_actor_v1.vinculo_contexto_actual a
      JOIN vec_contexto_actor_v1.vinculo_contexto_versiones v
        ON v.vinculo_ref=a.vinculo_ref AND v.version=a.version
     WHERE v.cuenta_ref=p_cuenta_ref AND v.perfil_ref=p_perfil_ref;
    IF coincidencias <> 1 THEN
        RAISE EXCEPTION USING ERRCODE = 'P0002', MESSAGE = 'contexto actor V2 no resuelto';
    END IF;

    SELECT cv.version, cv.procedencia_ref,cv.procedencia_version,
           cv.procedencia_huella_sha256,cv.procedencia_autoridad,
           cv.estado, cv.vigente_desde, cv.vigente_hasta
      INTO STRICT cuenta
      FROM vec_contexto_actor_v1.proyeccion_cuenta_actual ca
      JOIN vec_contexto_actor_v1.proyeccion_cuenta_versiones cv USING (cuenta_ref,version)
     WHERE ca.cuenta_ref=p_cuenta_ref;
    SELECT pv.version, pv.persona_ref,pv.procedencia_ref,pv.procedencia_version,
           pv.procedencia_huella_sha256,pv.procedencia_autoridad,
           pv.estado, pv.vigente_desde, pv.vigente_hasta
      INTO STRICT perfil
      FROM vec_contexto_actor_v1.perfil_actual pa
      JOIN vec_contexto_actor_v1.perfil_versiones pv USING (perfil_ref,version)
     WHERE pa.perfil_ref=p_perfil_ref;
    SELECT vv.vinculo_ref, vv.version, vv.persona_ref,
           vv.procedencia_ref,vv.procedencia_version,vv.procedencia_huella_sha256,
           vv.procedencia_autoridad,vv.estado, vv.vigente_desde, vv.vigente_hasta
      INTO STRICT enlace
      FROM vec_contexto_actor_v1.vinculo_contexto_actual va
      JOIN vec_contexto_actor_v1.vinculo_contexto_versiones vv USING (vinculo_ref,version)
     WHERE vv.cuenta_ref=p_cuenta_ref AND vv.perfil_ref=p_perfil_ref;
    SELECT pv.version,pv.procedencia_ref,pv.procedencia_version,
           pv.procedencia_huella_sha256,pv.procedencia_autoridad,
           pv.estado, pv.vigente_desde, pv.vigente_hasta
      INTO STRICT persona
      FROM vec_contexto_actor_v1.persona_actual pa
      JOIN vec_contexto_actor_v1.persona_versiones pv USING (persona_ref,version)
     WHERE pa.persona_ref=perfil.persona_ref;

    IF perfil.persona_ref <> enlace.persona_ref
       OR cuenta.estado <> 'activo' OR perfil.estado <> 'activo'
       OR persona.estado <> 'activo' OR enlace.estado <> 'activo'
       OR cuenta.procedencia_autoridad <> 'autoridad_maestra_acreditada'
       OR perfil.procedencia_autoridad <> 'autoridad_maestra_acreditada'
       OR persona.procedencia_autoridad <> 'autoridad_maestra_acreditada'
       OR enlace.procedencia_autoridad <> 'autoridad_maestra_acreditada'
       OR ahora < cuenta.vigente_desde OR ahora >= cuenta.vigente_hasta
       OR ahora < perfil.vigente_desde OR ahora >= perfil.vigente_hasta
       OR ahora < persona.vigente_desde OR ahora >= persona.vigente_hasta
       OR ahora < enlace.vigente_desde OR ahora >= enlace.vigente_hasta THEN
        RAISE EXCEPTION USING ERRCODE = 'P0002', MESSAGE = 'contexto actor V2 no vigente';
    END IF;

    SELECT count(*), count(DISTINCT vr.tipo), count(DISTINCT (vr.tipo,vr.referencia)),
           string_agg(format(
             '{"vinculo_ref":%s,"version":%s,"tipo":%s,"referencia":%s,"estado":%s,"vigente_desde":%s,"vigente_hasta":%s}',
             to_json(vr.vinculo_ref)::text, vr.version::text, to_json(vr.tipo)::text,
             to_json(vr.referencia)::text, to_json(vr.estado)::text,
             to_json(to_char(vr.vigente_desde AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))::text,
             to_json(to_char(vr.vigente_hasta AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))::text
           ), ',' ORDER BY vr.tipo,vr.referencia,vr.version,vr.vinculo_ref),
           string_agg(format(
             '{"vinculo_ref":%s,"version":%s,"tipo":%s,"referencia":%s,"procedencia_ref":%s,"procedencia_version":%s,"procedencia_huella_sha256":%s,"procedencia_autoridad":%s}',
             to_json(vr.vinculo_ref)::text,vr.version::text,to_json(vr.tipo)::text,
             to_json(vr.referencia)::text,to_json(vr.procedencia_ref)::text,
             vr.procedencia_version::text,to_json(vr.procedencia_huella_sha256)::text,
             to_json(vr.procedencia_autoridad)::text
           ), ',' ORDER BY vr.tipo,vr.referencia,vr.version,vr.vinculo_ref)
      INTO numero_enlaces, tipos, referencias, enlaces_texto,enlaces_procedencia_texto
      FROM vec_contexto_actor_v1.vinculo_referencia_actual ra
      JOIN vec_contexto_actor_v1.vinculo_referencia_versiones vr USING (vinculo_ref,version)
     WHERE vr.persona_ref=perfil.persona_ref;
    IF numero_enlaces > 128 OR tipos <> numero_enlaces OR referencias <> numero_enlaces
       OR EXISTS (
           SELECT 1 FROM vec_contexto_actor_v1.vinculo_referencia_actual ra
           JOIN vec_contexto_actor_v1.vinculo_referencia_versiones vr USING (vinculo_ref,version)
           WHERE vr.persona_ref=perfil.persona_ref AND (
             vr.estado <> 'activo' OR vr.procedencia_autoridad <> 'autoridad_maestra_acreditada'
             OR ahora < vr.vigente_desde OR ahora >= vr.vigente_hasta)
       ) THEN
        RAISE EXCEPTION USING ERRCODE = 'P0002', MESSAGE = 'referencias de contexto actor V2 no vigentes';
    END IF;

    manifiesto_procedencia := format(
      '{"esquema":"vec.contexto-actor.procedencia-manifiesto.v1","autoridad_efectiva":"autoridad_maestra_acreditada","cuenta":{"cuenta_ref":%s,"version":%s,"procedencia_ref":%s,"procedencia_version":%s,"procedencia_huella_sha256":%s,"procedencia_autoridad":%s},"persona":{"persona_ref":%s,"version":%s,"procedencia_ref":%s,"procedencia_version":%s,"procedencia_huella_sha256":%s,"procedencia_autoridad":%s},"perfil":{"perfil_ref":%s,"version":%s,"procedencia_ref":%s,"procedencia_version":%s,"procedencia_huella_sha256":%s,"procedencia_autoridad":%s},"contexto":{"vinculo_ref":%s,"version":%s,"procedencia_ref":%s,"procedencia_version":%s,"procedencia_huella_sha256":%s,"procedencia_autoridad":%s},"vinculos":[%s]}',
      to_json(p_cuenta_ref)::text,cuenta.version::text,to_json(cuenta.procedencia_ref)::text,
      cuenta.procedencia_version::text,to_json(cuenta.procedencia_huella_sha256)::text,to_json(cuenta.procedencia_autoridad)::text,
      to_json(perfil.persona_ref)::text,persona.version::text,to_json(persona.procedencia_ref)::text,
      persona.procedencia_version::text,to_json(persona.procedencia_huella_sha256)::text,to_json(persona.procedencia_autoridad)::text,
      to_json(p_perfil_ref)::text,perfil.version::text,to_json(perfil.procedencia_ref)::text,
      perfil.procedencia_version::text,to_json(perfil.procedencia_huella_sha256)::text,to_json(perfil.procedencia_autoridad)::text,
      to_json(enlace.vinculo_ref)::text,enlace.version::text,to_json(enlace.procedencia_ref)::text,
      enlace.procedencia_version::text,to_json(enlace.procedencia_huella_sha256)::text,to_json(enlace.procedencia_autoridad)::text,
      coalesce(enlaces_procedencia_texto,''));
    manifiesto_procedencia_bytes := convert_to(manifiesto_procedencia,'UTF8');
    manifiesto_procedencia_huella := encode(pg_catalog.sha256(manifiesto_procedencia_bytes),'hex');

    documento := format(
      '{"esquema":"vec.contexto-actor.vinculado.v2","principal_ref":%s,"metodo":%s,"garantia":%s,"perfil_activo_ref":%s,"persona_ref":%s,"contexto_actor_ref":%s,"contexto_version":%s,"cuenta_ref":%s,"cuenta_version":%s,"persona_version":%s,"perfil_version":%s,"estado":%s,"vigente_desde":%s,"vigente_hasta":%s,"resuelto_en":%s,"vinculos":[%s]}',
      to_json(perfil.persona_ref)::text, to_json(p_metodo)::text, to_json(p_garantia)::text,
      to_json(p_perfil_ref)::text, to_json(perfil.persona_ref)::text,
      to_json(enlace.vinculo_ref)::text, enlace.version::text, to_json(p_cuenta_ref)::text, cuenta.version::text,
      persona.version::text, perfil.version::text, to_json(enlace.estado)::text,
      to_json(to_char(enlace.vigente_desde AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))::text,
      to_json(to_char(enlace.vigente_hasta AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))::text,
      to_json(to_char(ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))::text,
      coalesce(enlaces_texto,''));
    canonica := convert_to(documento,'UTF8');
    IF octet_length(canonica) > 65536 THEN
        RAISE EXCEPTION USING ERRCODE = '22001', MESSAGE = 'snapshot de contexto actor V2 excede cota';
    END IF;
    huella := encode(pg_catalog.sha256(canonica),'hex');
    INSERT INTO vec_contexto_actor_v1.registros_contexto(
        operacion_ref,registro_contexto_ref,cuenta_ref,perfil_ref,metodo,garantia,
        solicitado_en,resuelto_en,representacion_canonica,huella_sha256,
        manifiesto_procedencia_canonico,manifiesto_procedencia_huella_sha256,autoridad_efectiva
    ) VALUES (
        p_operacion_ref,p_registro_contexto_ref,p_cuenta_ref,p_perfil_ref,p_metodo,p_garantia,
        p_solicitado_en,ahora,canonica,huella,
        manifiesto_procedencia_bytes,manifiesto_procedencia_huella,'autoridad_maestra_acreditada'
    );
    RETURN QUERY SELECT p_operacion_ref,p_registro_contexto_ref,canonica,huella,
      manifiesto_procedencia_bytes,manifiesto_procedencia_huella,
      'autoridad_maestra_acreditada'::text,ahora;
END
$f$;

REVOKE ALL ON ALL TABLES IN SCHEMA vec_contexto_actor_v1 FROM PUBLIC, vec_contexto_actor_v1_runtime;
REVOKE ALL ON ALL SEQUENCES IN SCHEMA vec_contexto_actor_v1 FROM PUBLIC, vec_contexto_actor_v1_runtime;
REVOKE ALL ON ALL FUNCTIONS IN SCHEMA vec_contexto_actor_v1 FROM PUBLIC, vec_contexto_actor_v1_runtime;
GRANT USAGE ON SCHEMA vec_contexto_actor_v1 TO vec_contexto_actor_v1_runtime;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.acreditar_runtime_contexto_actor_v1()
    TO vec_contexto_actor_v1_runtime;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2(text,text,text,text,text,text,timestamptz)
    TO vec_contexto_actor_v1_runtime;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.reconciliar_contexto_actor_v2(text,text,text,text,text,text,timestamptz)
    TO vec_contexto_actor_v1_runtime;

DO $cerrar_tipos$
DECLARE t record;
BEGIN
    FOR t IN SELECT typname FROM pg_catalog.pg_type ty JOIN pg_catalog.pg_namespace ns ON ns.oid=ty.typnamespace
              WHERE ns.nspname='vec_contexto_actor_v1' AND ty.typtype IN ('c','d','e')
    LOOP
        EXECUTE format('REVOKE ALL ON TYPE vec_contexto_actor_v1.%I FROM PUBLIC, vec_contexto_actor_v1_runtime',t.typname);
    END LOOP;
END
$cerrar_tipos$;
RESET ROLE;
DO $cerrar_creacion_base$
BEGIN
    EXECUTE format(
      'REVOKE CREATE ON DATABASE %I FROM vec_contexto_actor_v1_propietario',
      current_database()
    );
    IF pg_catalog.has_database_privilege(
         'vec_contexto_actor_v1_propietario',current_database(),'CREATE') THEN
        RAISE EXCEPTION 'el propietario conserva CREATE sobre la base dedicada';
    END IF;
END
$cerrar_creacion_base$;
COMMIT;
