-- Proyeccion organizativa opaca, versionada y sin autoridad de lectura.
-- Este documento es autonomo y se ejecuta literalmente en una transaccion.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

DO $precondiciones_minimas$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname = current_user AND rolsuper
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'organizacion corporativa V1 requiere migracion superusuario';
    ELSIF pg_catalog.current_setting('server_version_num')::integer < 180000
       OR pg_catalog.current_setting('server_version_num')::integer >= 190000
       OR pg_catalog.current_setting('server_encoding') <> 'UTF8' THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'organizacion corporativa V1 requiere PostgreSQL 18 y UTF8';
    END IF;
END
$precondiciones_minimas$;

-- Orden global: A compartida y despues B exclusiva. Las relaciones y los
-- catalogos quedan inmoviles antes de acreditar o crear ningun objeto.
SELECT pg_catalog.pg_advisory_xact_lock_shared(
    pg_catalog.hashtextextended(
        'vec_contexto_actor_v1:migracion:acreditacion_uso:v2', 0
    )
);
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contexto_actor_v1:organizacion-corporativa-rrhh:v1', 0
    )
);
LOCK TABLE
    vec_contexto_actor_v1.procedencias,
    vec_contexto_actor_v1.control_generacion_punteros_actuales_v2
IN SHARE MODE;
LOCK TABLE
    pg_catalog.pg_authid, pg_catalog.pg_auth_members,
    pg_catalog.pg_db_role_setting, pg_catalog.pg_class,
    pg_catalog.pg_attribute, pg_catalog.pg_index,
    pg_catalog.pg_namespace, pg_catalog.pg_language,
    pg_catalog.pg_collation, pg_catalog.pg_proc, pg_catalog.pg_type,
    pg_catalog.pg_default_acl, pg_catalog.pg_description,
    pg_catalog.pg_seclabel, pg_catalog.pg_init_privs,
    pg_catalog.pg_depend, pg_catalog.pg_shdepend,
    pg_catalog.pg_constraint, pg_catalog.pg_trigger, pg_catalog.pg_policy,
    pg_catalog.pg_publication, pg_catalog.pg_publication_namespace,
    pg_catalog.pg_publication_rel, pg_catalog.pg_subscription_rel,
    pg_catalog.pg_statistic_ext
IN SHARE MODE;

DO $acreditar_base_y_ausencia$
DECLARE
    propietario constant oid := 'vec_contexto_actor_v1_propietario'::regrole;
    migrador constant oid := 'vec_contexto_actor_v1_migrador'::regrole;
    runtime constant oid := 'vec_contexto_actor_v1_runtime'::regrole;
    esquema constant oid := 'vec_contexto_actor_v1'::regnamespace;
    nombres constant text[] := ARRAY[
        'organizacion_versiones', 'organizacion_actual'
    ];
BEGIN
    IF pg_catalog.current_setting('server_version_num')::integer < 180000
       OR pg_catalog.current_setting('server_version_num')::integer >= 190000
       OR pg_catalog.current_setting('server_encoding') <> 'UTF8'
       OR pg_catalog.current_setting('timezone') <> 'UTC'
       OR NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_authid
            WHERE rolname = current_user AND rolsuper
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'entorno de organizacion corporativa V1 no reacreditado';
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_authid
         WHERE oid = propietario AND NOT rolcanlogin AND NOT rolsuper
           AND NOT rolcreaterole AND NOT rolcreatedb AND NOT rolreplication
           AND NOT rolbypassrls
    ) OR NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_namespace
         WHERE oid = esquema AND nspowner = propietario
    ) OR EXISTS (
        SELECT 1 FROM pg_catalog.pg_class
         WHERE relnamespace = esquema AND relname = ANY(nombres)
    ) OR EXISTS (
        SELECT 1 FROM pg_catalog.pg_type
         WHERE typnamespace = esquema
           AND typname = ANY(nombres || ARRAY[
               '_organizacion_versiones', '_organizacion_actual'
           ])
    ) OR EXISTS (
        SELECT 1 FROM pg_catalog.pg_proc
         WHERE pronamespace = esquema AND proname = 'organizacion_ref_valida'
    ) OR EXISTS (
        SELECT 1 FROM pg_catalog.pg_publication WHERE puballtables
    ) OR EXISTS (
        SELECT 1 FROM pg_catalog.pg_publication_namespace
         WHERE pnnspid = esquema
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'base o ausencia nominal de organizacion corporativa V1 no acreditada';
    END IF;

    IF (
        SELECT pg_catalog.count(*)
          FROM pg_catalog.pg_authid
         WHERE rolname = ANY(ARRAY[
             'vec_contexto_actor_v1_propietario',
             'vec_contexto_actor_v1_migrador',
             'vec_contexto_actor_v1_runtime'
         ]) AND NOT rolsuper AND NOT rolcanlogin
           AND NOT rolinherit
           AND NOT rolcreaterole AND NOT rolcreatedb
           AND NOT rolreplication AND NOT rolbypassrls
           AND rolconnlimit = -1 AND rolpassword IS NULL
           AND rolvaliduntil IS NULL
    ) <> 3 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'roles base de organizacion corporativa V1 no acreditados';
    END IF;

    IF (
        SELECT pg_catalog.count(*)
          FROM pg_catalog.pg_auth_members AS m
         WHERE m.roleid = propietario OR m.member = ANY(ARRAY[
             propietario, migrador, runtime
         ]::oid[]) OR m.roleid = migrador
    ) <> 1 OR NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_auth_members AS m
         WHERE m.roleid = propietario AND m.member = migrador
           AND NOT m.admin_option AND NOT m.inherit_option AND m.set_option
    ) OR EXISTS (
        SELECT 1 FROM pg_catalog.pg_auth_members AS m
        JOIN pg_catalog.pg_authid AS login ON login.oid = m.member
         WHERE m.roleid = runtime
           AND (m.admin_option OR NOT m.inherit_option OR m.set_option
                OR NOT login.rolcanlogin OR login.rolsuper
                OR NOT login.rolinherit OR login.rolcreaterole
                OR login.rolcreatedb OR login.rolreplication
                OR login.rolbypassrls
                OR (SELECT pg_catalog.count(*)
                      FROM pg_catalog.pg_auth_members AS otra
                     WHERE otra.member = m.member) <> 1
                OR EXISTS (
                    SELECT 1 FROM pg_catalog.pg_db_role_setting AS ajuste
                     WHERE ajuste.setrole = m.member
                ))
    ) OR EXISTS (
        SELECT 1 FROM pg_catalog.pg_auth_members AS m
         WHERE m.roleid = (
                   SELECT oid FROM pg_catalog.pg_authid
                    WHERE rolname =
                      'vec_contexto_actor_corporativo_rrhh_selector'
               ) OR m.member = (
                   SELECT oid FROM pg_catalog.pg_authid
                    WHERE rolname =
                      'vec_contexto_actor_corporativo_rrhh_selector'
               )
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'topologia de roles de organizacion corporativa V1 no acreditada';
    END IF;

    IF (
        SELECT pg_catalog.count(*)
          FROM pg_catalog.pg_class
         WHERE oid IN (
             'vec_contexto_actor_v1.procedencias'::regclass,
             'vec_contexto_actor_v1.control_generacion_punteros_actuales_v2'::regclass
         ) AND relkind = 'r' AND relpersistence = 'p'
           AND relowner = propietario
    ) <> 2 OR (
        SELECT pg_catalog.count(*)
          FROM pg_catalog.pg_proc
         WHERE oid IN (
             'vec_contexto_actor_v1.procedencia_valida(text,numeric,text,text)'::regprocedure,
             'vec_contexto_actor_v1.instante_valido(timestamptz)'::regprocedure,
             'vec_contexto_actor_v1.rechazar_mutacion_historia()'::regprocedure,
             'vec_contexto_actor_v1.rechazar_truncado()'::regprocedure,
             'vec_contexto_actor_v1.serializar_mutacion_punteros_actuales_v2()'::regprocedure,
             'vec_contexto_actor_v1.avanzar_generacion_punteros_actuales_v2()'::regprocedure
         ) AND proowner = propietario
    ) <> 6 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'dependencias de organizacion corporativa V1 no acreditadas';
    END IF;
END
$acreditar_base_y_ausencia$;

SET LOCAL ROLE vec_contexto_actor_v1_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

CREATE FUNCTION vec_contexto_actor_v1.organizacion_ref_valida(p_valor text)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog
AS $funcion$
    SELECT p_valor IS NOT NULL
       AND pg_catalog.octet_length(p_valor) BETWEEN 20 AND 84
       AND pg_catalog.left(p_valor, 4) = 'org_'
       AND pg_catalog.translate(
               pg_catalog.substr(p_valor, 5),
               'abcdefghijklmnopqrstuvwxyz0123456789', ''
           ) = ''
$funcion$;

CREATE TABLE vec_contexto_actor_v1.organizacion_versiones (
    organizacion_ref text NOT NULL,
    version numeric(20,0) NOT NULL,
    procedencia_ref text NOT NULL,
    procedencia_version numeric(20,0) NOT NULL,
    procedencia_huella_sha256 text NOT NULL,
    procedencia_autoridad text NOT NULL,
    estado text NOT NULL,
    vigente_desde timestamptz(6) NOT NULL,
    vigente_hasta timestamptz(6) NOT NULL,
    CONSTRAINT organizacion_versiones_pk
        PRIMARY KEY (organizacion_ref, version),
    CONSTRAINT organizacion_versiones_procedencia_uq UNIQUE (
        organizacion_ref, version, procedencia_ref, procedencia_version,
        procedencia_huella_sha256, procedencia_autoridad
    ),
    CONSTRAINT organizacion_versiones_procedencia_fk FOREIGN KEY (
        procedencia_ref, procedencia_version,
        procedencia_huella_sha256, procedencia_autoridad
    ) REFERENCES vec_contexto_actor_v1.procedencias (
        procedencia_ref, procedencia_version,
        procedencia_huella_sha256, procedencia_autoridad
    ) MATCH FULL,
    CONSTRAINT organizacion_versiones_ref_ck CHECK (
        vec_contexto_actor_v1.organizacion_ref_valida(organizacion_ref)
    ),
    CONSTRAINT organizacion_versiones_version_ck CHECK (
        version BETWEEN 1 AND 18446744073709551615::numeric
    ),
    CONSTRAINT organizacion_versiones_procedencia_ck CHECK (
        vec_contexto_actor_v1.procedencia_valida(
            procedencia_ref, procedencia_version,
            procedencia_huella_sha256, procedencia_autoridad
        )
    ),
    CONSTRAINT organizacion_versiones_autoridad_ck CHECK (
        procedencia_autoridad = 'autoridad_maestra_acreditada'
    ),
    CONSTRAINT organizacion_versiones_estado_ck CHECK (
        estado IN ('activo', 'revocado')
    ),
    CONSTRAINT organizacion_versiones_vigente_desde_ck CHECK (
        vec_contexto_actor_v1.instante_valido(vigente_desde)
    ),
    CONSTRAINT organizacion_versiones_vigente_hasta_ck CHECK (
        vec_contexto_actor_v1.instante_valido(vigente_hasta)
    ),
    CONSTRAINT organizacion_versiones_ventana_ck CHECK (
        vigente_hasta > vigente_desde
    )
);

CREATE TABLE vec_contexto_actor_v1.organizacion_actual (
    organizacion_ref text NOT NULL,
    version numeric(20,0) NOT NULL,
    CONSTRAINT organizacion_actual_pk PRIMARY KEY (organizacion_ref),
    CONSTRAINT organizacion_actual_version_ck CHECK (
        version BETWEEN 1 AND 18446744073709551615::numeric
    ),
    CONSTRAINT organizacion_actual_version_fk FOREIGN KEY (
        organizacion_ref, version
    ) REFERENCES vec_contexto_actor_v1.organizacion_versiones (
        organizacion_ref, version
    ) MATCH FULL
);

ALTER TABLE vec_contexto_actor_v1.organizacion_versiones
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contexto_actor_v1.organizacion_versiones
    FORCE ROW LEVEL SECURITY;
CREATE POLICY acceso_propietario_exacto
    ON vec_contexto_actor_v1.organizacion_versiones
    AS PERMISSIVE FOR ALL
    TO vec_contexto_actor_v1_propietario
    USING (CURRENT_USER = 'vec_contexto_actor_v1_propietario')
    WITH CHECK (CURRENT_USER = 'vec_contexto_actor_v1_propietario');

ALTER TABLE vec_contexto_actor_v1.organizacion_actual
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contexto_actor_v1.organizacion_actual
    FORCE ROW LEVEL SECURITY;
CREATE POLICY acceso_propietario_exacto
    ON vec_contexto_actor_v1.organizacion_actual
    AS PERMISSIVE FOR ALL
    TO vec_contexto_actor_v1_propietario
    USING (CURRENT_USER = 'vec_contexto_actor_v1_propietario')
    WITH CHECK (CURRENT_USER = 'vec_contexto_actor_v1_propietario');

CREATE TRIGGER historia_inmutable
BEFORE UPDATE OR DELETE ON vec_contexto_actor_v1.organizacion_versiones
FOR EACH ROW EXECUTE FUNCTION
    vec_contexto_actor_v1.rechazar_mutacion_historia();
CREATE TRIGGER historia_no_truncable
BEFORE TRUNCATE ON vec_contexto_actor_v1.organizacion_versiones
FOR EACH STATEMENT EXECUTE FUNCTION vec_contexto_actor_v1.rechazar_truncado();

CREATE TRIGGER puntero_actual_no_truncable_v2
BEFORE TRUNCATE ON vec_contexto_actor_v1.organizacion_actual
FOR EACH STATEMENT EXECUTE FUNCTION vec_contexto_actor_v1.rechazar_truncado();
CREATE TRIGGER serializar_mutacion_punteros_actuales_v2
BEFORE INSERT OR UPDATE OR DELETE ON vec_contexto_actor_v1.organizacion_actual
FOR EACH STATEMENT EXECUTE FUNCTION
    vec_contexto_actor_v1.serializar_mutacion_punteros_actuales_v2();
CREATE TRIGGER avanzar_generacion_punteros_actuales_v2
AFTER INSERT OR UPDATE OR DELETE ON vec_contexto_actor_v1.organizacion_actual
FOR EACH STATEMENT EXECUTE FUNCTION
    vec_contexto_actor_v1.avanzar_generacion_punteros_actuales_v2();

REVOKE ALL ON FUNCTION
    vec_contexto_actor_v1.organizacion_ref_valida(text)
    FROM PUBLIC, vec_contexto_actor_v1_runtime;
REVOKE ALL ON TABLE
    vec_contexto_actor_v1.organizacion_versiones,
    vec_contexto_actor_v1.organizacion_actual
    FROM PUBLIC, vec_contexto_actor_v1_runtime;
REVOKE ALL ON TYPE
    vec_contexto_actor_v1.organizacion_versiones,
    vec_contexto_actor_v1.organizacion_actual
    FROM PUBLIC, vec_contexto_actor_v1_runtime;

DO $postcondiciones$
DECLARE
    propietario constant oid := 'vec_contexto_actor_v1_propietario'::regrole;
    esquema constant oid := 'vec_contexto_actor_v1'::regnamespace;
    versiones constant oid := 'vec_contexto_actor_v1.organizacion_versiones'::regclass;
    actual constant oid := 'vec_contexto_actor_v1.organizacion_actual'::regclass;
    validador constant oid := 'vec_contexto_actor_v1.organizacion_ref_valida(text)'::regprocedure;
BEGIN
    IF (
        SELECT pg_catalog.count(*)
          FROM pg_catalog.pg_class
         WHERE oid IN (versiones, actual) AND relkind = 'r'
           AND relpersistence = 'p' AND relowner = propietario
           AND relrowsecurity AND relforcerowsecurity
           AND reloptions IS NULL
    ) <> 2 OR (
        SELECT pg_catalog.count(*)
          FROM pg_catalog.pg_policy
         WHERE polrelid IN (versiones, actual)
           AND polname = 'acceso_propietario_exacto'
           AND polpermissive AND polcmd = '*'
           AND polroles = ARRAY[propietario]::oid[]
           AND polqual IS NOT NULL AND polwithcheck IS NOT NULL
    ) <> 2 OR (
        SELECT pg_catalog.count(*)
          FROM pg_catalog.pg_trigger
         WHERE tgrelid IN (versiones, actual) AND NOT tgisinternal
    ) <> 5 OR (
        SELECT pg_catalog.count(*)
          FROM pg_catalog.pg_constraint
         WHERE connamespace = esquema
           AND conname = ANY(ARRAY[
               'organizacion_versiones_pk',
               'organizacion_versiones_procedencia_uq',
               'organizacion_versiones_procedencia_fk',
               'organizacion_versiones_ref_ck',
               'organizacion_versiones_version_ck',
               'organizacion_versiones_procedencia_ck',
               'organizacion_versiones_autoridad_ck',
               'organizacion_versiones_estado_ck',
               'organizacion_versiones_vigente_desde_ck',
               'organizacion_versiones_vigente_hasta_ck',
               'organizacion_versiones_ventana_ck',
               'organizacion_actual_pk',
               'organizacion_actual_version_ck',
               'organizacion_actual_version_fk'
           ])
    ) <> 14 OR NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_proc
         WHERE oid = validador AND proowner = propietario
           AND provolatile = 'i' AND NOT prosecdef AND NOT proretset
           AND proconfig = ARRAY['search_path=pg_catalog']
           AND pg_catalog.pg_get_function_result(oid) = 'boolean'
    ) OR EXISTS (
        SELECT 1 FROM pg_catalog.pg_attribute
         WHERE attrelid IN (versiones, actual) AND attnum > 0
           AND NOT attisdropped AND attacl IS NOT NULL
    ) OR EXISTS (
        SELECT 1
          FROM (
              SELECT c.oid, pg_catalog.count(a.*) AS numero,
                     pg_catalog.bool_and(
                         a.grantor = propietario AND a.grantee = propietario
                         AND NOT a.is_grantable
                         AND a.privilege_type = ANY(ARRAY[
                             'INSERT', 'SELECT', 'UPDATE', 'DELETE',
                             'TRUNCATE', 'REFERENCES', 'TRIGGER', 'MAINTAIN'
                         ])
                     ) AS propios
                FROM pg_catalog.pg_class AS c
                LEFT JOIN LATERAL pg_catalog.aclexplode(c.relacl) AS a ON true
               WHERE c.oid IN (versiones, actual)
               GROUP BY c.oid
          ) AS acl WHERE numero <> 8 OR propios IS NOT TRUE
    ) OR (
        SELECT pg_catalog.count(*) <> 1 OR NOT pg_catalog.bool_and(
                   a.grantor = propietario AND a.grantee = propietario
                   AND a.privilege_type = 'EXECUTE' AND NOT a.is_grantable
               )
          FROM pg_catalog.pg_proc AS p
          LEFT JOIN LATERAL pg_catalog.aclexplode(p.proacl) AS a ON true
         WHERE p.oid = validador
    ) OR EXISTS (
        SELECT 1
          FROM (
              SELECT t.oid, pg_catalog.count(a.*) AS numero,
                     pg_catalog.bool_and(
                         a.grantor = propietario AND a.grantee = propietario
                         AND a.privilege_type = 'USAGE'
                         AND NOT a.is_grantable
                     ) AS propios
                FROM pg_catalog.pg_type AS t
                LEFT JOIN LATERAL pg_catalog.aclexplode(t.typacl) AS a ON true
               WHERE t.typrelid IN (versiones, actual)
               GROUP BY t.oid
          ) AS acl WHERE numero <> 1 OR propios IS NOT TRUE
    ) OR EXISTS (
        WITH roles_objetivo AS (
            SELECT oid FROM pg_catalog.pg_roles
             WHERE rolname IN (
                 'vec_contexto_actor_v1_runtime',
                 'vec_contexto_actor_corporativo_rrhh_selector'
             )
            UNION
            SELECT m.member FROM pg_catalog.pg_auth_members AS m
             WHERE m.roleid = 'vec_contexto_actor_v1_runtime'::regrole
        ), tablas AS (
            SELECT versiones AS oid UNION ALL SELECT actual
        ), tipos AS (
            SELECT t.oid FROM pg_catalog.pg_type AS t
             WHERE t.typrelid IN (versiones, actual)
                OR t.typelem IN (
                    SELECT fila.oid FROM pg_catalog.pg_type AS fila
                     WHERE fila.typrelid IN (versiones, actual)
                )
        )
        SELECT 1 FROM roles_objetivo AS rol
         WHERE pg_catalog.has_function_privilege(
                   rol.oid, validador, 'EXECUTE'
               ) OR EXISTS (
                   SELECT 1 FROM tablas AS tabla
                    WHERE pg_catalog.has_table_privilege(
                              rol.oid, tabla.oid, 'SELECT'
                          ) OR pg_catalog.has_table_privilege(
                              rol.oid, tabla.oid, 'INSERT'
                          ) OR pg_catalog.has_table_privilege(
                              rol.oid, tabla.oid, 'UPDATE'
                          ) OR pg_catalog.has_table_privilege(
                              rol.oid, tabla.oid, 'DELETE'
                          ) OR pg_catalog.has_table_privilege(
                              rol.oid, tabla.oid, 'TRUNCATE'
                          ) OR pg_catalog.has_table_privilege(
                              rol.oid, tabla.oid, 'REFERENCES'
                          ) OR pg_catalog.has_table_privilege(
                              rol.oid, tabla.oid, 'TRIGGER'
                          ) OR pg_catalog.has_table_privilege(
                              rol.oid, tabla.oid, 'MAINTAIN'
                          ) OR pg_catalog.has_any_column_privilege(
                              rol.oid, tabla.oid, 'SELECT'
                          ) OR pg_catalog.has_any_column_privilege(
                              rol.oid, tabla.oid, 'INSERT'
                          ) OR pg_catalog.has_any_column_privilege(
                              rol.oid, tabla.oid, 'UPDATE'
                          ) OR pg_catalog.has_any_column_privilege(
                              rol.oid, tabla.oid, 'REFERENCES'
                          )
               ) OR EXISTS (
                   SELECT 1 FROM tipos AS tipo
                    WHERE pg_catalog.has_type_privilege(
                              rol.oid, tipo.oid, 'USAGE'
                          )
               )
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'postcondiciones de organizacion corporativa V1 no acreditadas';
    END IF;
END
$postcondiciones$;

COMMIT;
