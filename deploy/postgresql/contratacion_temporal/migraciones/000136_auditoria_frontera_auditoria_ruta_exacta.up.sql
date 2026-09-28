\set ON_ERROR_STOP on
-- CT136: registro minimizado y de solo adición de denegaciones tempranas de Auditoría.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:migracion:000136', 0
    )
);

DO $precondicion$
BEGIN
    IF pg_catalog.to_regclass(
           'vec_contratacion_temporal.auditoria_frontera_auditoria_ruta_exacta'
       ) IS NOT NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.registrar_auditoria_frontera_auditoria_v1(text,text,text,text,text)'
       ) IS NOT NULL
       OR NOT EXISTS (
            SELECT 1 FROM pg_catalog.pg_roles
             WHERE rolname = 'vec_contratacion_temporal_registrador_auditoria'
               AND NOT rolcanlogin AND NOT rolsuper AND NOT rolcreatedb
               AND NOT rolcreaterole AND rolinherit AND NOT rolreplication
               AND NOT rolbypassrls
       ) OR EXISTS (
            SELECT 1 FROM pg_catalog.pg_auth_members membresia
              JOIN pg_catalog.pg_roles miembro ON miembro.oid = membresia.member
             WHERE miembro.rolname = 'vec_contratacion_temporal_registrador_auditoria'
       ) OR pg_catalog.has_schema_privilege(
            'vec_contratacion_temporal_registrador_auditoria',
            'vec_contratacion_temporal', 'USAGE'
       ) OR pg_catalog.has_schema_privilege(
            'vec_contratacion_temporal_registrador_auditoria',
            'vec_contratacion_temporal', 'CREATE'
       ) OR EXISTS (
            SELECT 1 FROM pg_catalog.pg_proc funcion
             WHERE funcion.pronamespace = 'vec_contratacion_temporal'::pg_catalog.regnamespace
               AND pg_catalog.has_function_privilege(
                    'vec_contratacion_temporal_registrador_auditoria', funcion.oid, 'EXECUTE'
               )
       ) OR EXISTS (
            SELECT 1 FROM pg_catalog.pg_class objeto
             WHERE objeto.relnamespace = 'vec_contratacion_temporal'::pg_catalog.regnamespace
               AND objeto.relkind IN ('r', 'p', 'v', 'm')
               AND (
                    pg_catalog.has_table_privilege(
                        'vec_contratacion_temporal_registrador_auditoria', objeto.oid, 'SELECT'
                    ) OR pg_catalog.has_table_privilege(
                        'vec_contratacion_temporal_registrador_auditoria', objeto.oid, 'INSERT'
                    ) OR pg_catalog.has_table_privilege(
                        'vec_contratacion_temporal_registrador_auditoria', objeto.oid, 'UPDATE'
                    ) OR pg_catalog.has_table_privilege(
                        'vec_contratacion_temporal_registrador_auditoria', objeto.oid, 'DELETE'
                    ) OR pg_catalog.has_table_privilege(
                        'vec_contratacion_temporal_registrador_auditoria', objeto.oid, 'TRUNCATE'
                    ) OR pg_catalog.has_table_privilege(
                        'vec_contratacion_temporal_registrador_auditoria', objeto.oid, 'REFERENCES'
                    ) OR pg_catalog.has_table_privilege(
                        'vec_contratacion_temporal_registrador_auditoria', objeto.oid, 'TRIGGER'
                    ) OR pg_catalog.has_table_privilege(
                        'vec_contratacion_temporal_registrador_auditoria', objeto.oid, 'MAINTAIN'
                    ) OR pg_catalog.has_any_column_privilege(
                        'vec_contratacion_temporal_registrador_auditoria', objeto.oid, 'SELECT'
                    ) OR pg_catalog.has_any_column_privilege(
                        'vec_contratacion_temporal_registrador_auditoria', objeto.oid, 'INSERT'
                    ) OR pg_catalog.has_any_column_privilege(
                        'vec_contratacion_temporal_registrador_auditoria', objeto.oid, 'UPDATE'
                    ) OR pg_catalog.has_any_column_privilege(
                        'vec_contratacion_temporal_registrador_auditoria', objeto.oid, 'REFERENCES'
                    )
               )
       ) OR EXISTS (
            SELECT 1 FROM pg_catalog.pg_class secuencia
             WHERE secuencia.relnamespace = 'vec_contratacion_temporal'::pg_catalog.regnamespace
               AND secuencia.relkind = 'S'
               AND (
                    pg_catalog.has_sequence_privilege(
                        'vec_contratacion_temporal_registrador_auditoria', secuencia.oid, 'USAGE'
                    ) OR pg_catalog.has_sequence_privilege(
                        'vec_contratacion_temporal_registrador_auditoria', secuencia.oid, 'SELECT'
                    ) OR pg_catalog.has_sequence_privilege(
                        'vec_contratacion_temporal_registrador_auditoria', secuencia.oid, 'UPDATE'
                    )
               )
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'CT136: preimagen del registrador de auditoria incompatible';
    END IF;
END
$precondicion$;

CREATE TABLE vec_contratacion_temporal.auditoria_frontera_auditoria_ruta_exacta (
    evento_id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    correlacion_ref text NOT NULL,
    motivo text NOT NULL,
    superficie text NOT NULL,
    ruta text NOT NULL,
    actor_ref text,
    registrada_en timestamptz(6) NOT NULL,
    CHECK (correlacion_ref = 'corr_no_disponible' OR correlacion_ref ~ '^corr_[0-9a-f]{32}$'),
    CHECK (motivo IN ('autenticacion_requerida', 'acceso_denegado')),
    CHECK (superficie = 'api.auditoria.ruta_exacta'),
    CHECK (ruta IN ('/api/vec/auditoria/opciones', '/api/vec/auditoria/consultas')),
    CHECK (actor_ref IS NULL OR (
        pg_catalog.length(actor_ref) BETWEEN 1 AND 512
        AND actor_ref ~ '^[A-Za-z0-9:_-]+$'
    )),
    CHECK (registrada_en = pg_catalog.date_trunc('microseconds', registrada_en))
);

CREATE INDEX auditoria_frontera_auditoria_ruta_exacta_correlacion_idx
    ON vec_contratacion_temporal.auditoria_frontera_auditoria_ruta_exacta (correlacion_ref, evento_id);

CREATE FUNCTION vec_contratacion_temporal.rechazar_mutacion_auditoria_frontera_auditoria_v1()
RETURNS trigger
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
BEGIN
    RAISE EXCEPTION USING ERRCODE = '42501',
        MESSAGE = 'auditoria de frontera inmutable';
END
$funcion$;

CREATE TRIGGER auditoria_frontera_auditoria_ruta_exacta_inmutable
BEFORE UPDATE OR DELETE ON vec_contratacion_temporal.auditoria_frontera_auditoria_ruta_exacta
FOR EACH ROW EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_auditoria_frontera_auditoria_v1();

CREATE TRIGGER auditoria_frontera_auditoria_ruta_exacta_no_truncar
BEFORE TRUNCATE ON vec_contratacion_temporal.auditoria_frontera_auditoria_ruta_exacta
FOR EACH STATEMENT EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_auditoria_frontera_auditoria_v1();

CREATE FUNCTION vec_contratacion_temporal.registrar_auditoria_frontera_auditoria_v1(
    p_correlacion_ref text,
    p_motivo text,
    p_superficie text,
    p_ruta text,
    p_actor_ref text
)
RETURNS boolean
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
SET row_security = 'on'
SET timezone = 'UTC'
SET lock_timeout = '1s'
SET statement_timeout = '2s'
AS $funcion$
DECLARE
    v_login pg_catalog.pg_roles%ROWTYPE;
    v_registrador pg_catalog.pg_roles%ROWTYPE;
    v_funcion_oid oid := 'vec_contratacion_temporal.registrar_auditoria_frontera_auditoria_v1(text,text,text,text,text)'::pg_catalog.regprocedure;
BEGIN
    SELECT * INTO v_login FROM pg_catalog.pg_roles WHERE rolname = SESSION_USER;
    SELECT * INTO v_registrador FROM pg_catalog.pg_roles
     WHERE rolname = 'vec_contratacion_temporal_registrador_auditoria';
    IF CURRENT_USER <> 'vec_contratacion_temporal_propietario'
       OR SESSION_USER = CURRENT_USER
       OR v_login.oid IS NULL OR NOT v_login.rolcanlogin OR NOT v_login.rolinherit
       OR v_login.rolsuper OR v_login.rolcreatedb OR v_login.rolcreaterole
       OR v_login.rolreplication OR v_login.rolbypassrls
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_auth_members m
            WHERE m.member = v_login.oid) <> 1
       OR NOT EXISTS (
            SELECT 1 FROM pg_catalog.pg_auth_members m
              JOIN pg_catalog.pg_roles r ON r.oid = m.roleid
             WHERE m.member = v_login.oid
               AND r.rolname = 'vec_contratacion_temporal_registrador_auditoria'
               AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option
       )
       OR EXISTS (
            SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.roleid = v_login.oid
       )
       OR v_registrador.oid IS NULL OR v_registrador.rolcanlogin
       OR v_registrador.rolsuper OR v_registrador.rolcreatedb
       OR v_registrador.rolcreaterole OR NOT v_registrador.rolinherit
       OR v_registrador.rolreplication OR v_registrador.rolbypassrls
       OR EXISTS (
            SELECT 1 FROM pg_catalog.pg_auth_members m
             WHERE m.member = v_registrador.oid
       )
       OR NOT pg_catalog.has_schema_privilege(
            v_registrador.oid, 'vec_contratacion_temporal', 'USAGE'
       )
       OR pg_catalog.has_schema_privilege(
            v_registrador.oid, 'vec_contratacion_temporal', 'CREATE'
       )
       OR EXISTS (
            SELECT 1 FROM pg_catalog.pg_proc funcion
             WHERE funcion.pronamespace = 'vec_contratacion_temporal'::pg_catalog.regnamespace
               AND funcion.oid <> v_funcion_oid
               AND pg_catalog.has_function_privilege(v_registrador.oid, funcion.oid, 'EXECUTE')
       )
       OR EXISTS (
            SELECT 1 FROM pg_catalog.pg_class objeto
             WHERE objeto.relnamespace = 'vec_contratacion_temporal'::pg_catalog.regnamespace
               AND objeto.relkind IN ('r', 'p', 'v', 'm')
               AND (
                    pg_catalog.has_table_privilege(v_registrador.oid, objeto.oid, 'SELECT')
                    OR pg_catalog.has_table_privilege(v_registrador.oid, objeto.oid, 'INSERT')
                    OR pg_catalog.has_table_privilege(v_registrador.oid, objeto.oid, 'UPDATE')
                    OR pg_catalog.has_table_privilege(v_registrador.oid, objeto.oid, 'DELETE')
                    OR pg_catalog.has_table_privilege(v_registrador.oid, objeto.oid, 'TRUNCATE')
                    OR pg_catalog.has_table_privilege(v_registrador.oid, objeto.oid, 'REFERENCES')
                    OR pg_catalog.has_table_privilege(v_registrador.oid, objeto.oid, 'TRIGGER')
                    OR pg_catalog.has_table_privilege(v_registrador.oid, objeto.oid, 'MAINTAIN')
                    OR pg_catalog.has_any_column_privilege(v_registrador.oid, objeto.oid, 'SELECT')
                    OR pg_catalog.has_any_column_privilege(v_registrador.oid, objeto.oid, 'INSERT')
                    OR pg_catalog.has_any_column_privilege(v_registrador.oid, objeto.oid, 'UPDATE')
                    OR pg_catalog.has_any_column_privilege(v_registrador.oid, objeto.oid, 'REFERENCES')
                )
       )
       OR EXISTS (
            SELECT 1 FROM pg_catalog.pg_class secuencia
             WHERE secuencia.relnamespace = 'vec_contratacion_temporal'::pg_catalog.regnamespace
               AND secuencia.relkind = 'S'
               AND (
                    pg_catalog.has_sequence_privilege(v_registrador.oid, secuencia.oid, 'USAGE')
                    OR pg_catalog.has_sequence_privilege(v_registrador.oid, secuencia.oid, 'SELECT')
                    OR pg_catalog.has_sequence_privilege(v_registrador.oid, secuencia.oid, 'UPDATE')
               )
       )
       OR pg_catalog.pg_is_in_recovery()
       OR pg_catalog.current_setting('transaction_read_only') <> 'off'
       OR pg_catalog.current_setting('TimeZone') <> 'UTC'
       OR p_correlacion_ref IS NULL
       OR (p_correlacion_ref <> 'corr_no_disponible'
           AND p_correlacion_ref !~ '^corr_[0-9a-f]{32}$')
       OR p_motivo NOT IN ('autenticacion_requerida', 'acceso_denegado')
       OR p_superficie <> 'api.auditoria.ruta_exacta'
       OR p_ruta NOT IN ('/api/vec/auditoria/opciones', '/api/vec/auditoria/consultas')
       OR p_actor_ref IS NOT NULL AND (
            pg_catalog.length(p_actor_ref) > 512
            OR p_actor_ref <> pg_catalog.btrim(p_actor_ref)
            OR p_actor_ref !~ '^[A-Za-z0-9:_-]+$'
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'auditoria de frontera invalida';
    END IF;

    INSERT INTO vec_contratacion_temporal.auditoria_frontera_auditoria_ruta_exacta (
        correlacion_ref, motivo, superficie, ruta, actor_ref, registrada_en
    ) VALUES (
        p_correlacion_ref, p_motivo, p_superficie, p_ruta, p_actor_ref,
        pg_catalog.date_trunc('microseconds', pg_catalog.clock_timestamp())
    );
    RETURN true;
END
$funcion$;

ALTER TABLE vec_contratacion_temporal.auditoria_frontera_auditoria_ruta_exacta ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contratacion_temporal.auditoria_frontera_auditoria_ruta_exacta FORCE ROW LEVEL SECURITY;
CREATE POLICY auditoria_frontera_auditoria_ruta_exacta_propietario_lectura
    ON vec_contratacion_temporal.auditoria_frontera_auditoria_ruta_exacta
    FOR SELECT
    TO vec_contratacion_temporal_propietario
    USING (
        superficie = 'api.auditoria.ruta_exacta'
        AND ruta IN ('/api/vec/auditoria/opciones', '/api/vec/auditoria/consultas')
    );
CREATE POLICY auditoria_frontera_auditoria_ruta_exacta_propietario_insercion
    ON vec_contratacion_temporal.auditoria_frontera_auditoria_ruta_exacta
    FOR INSERT
    TO vec_contratacion_temporal_propietario
    WITH CHECK (
        superficie = 'api.auditoria.ruta_exacta'
        AND ruta IN ('/api/vec/auditoria/opciones', '/api/vec/auditoria/consultas')
    );

REVOKE ALL ON TABLE vec_contratacion_temporal.auditoria_frontera_auditoria_ruta_exacta FROM PUBLIC;
REVOKE ALL ON SEQUENCE vec_contratacion_temporal.auditoria_frontera_auditoria_ruta_exacta_evento_id_seq FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.rechazar_mutacion_auditoria_frontera_auditoria_v1() FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.registrar_auditoria_frontera_auditoria_v1(text,text,text,text,text) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_contratacion_temporal TO vec_contratacion_temporal_registrador_auditoria;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.registrar_auditoria_frontera_auditoria_v1(text,text,text,text,text)
    TO vec_contratacion_temporal_registrador_auditoria;

COMMIT;
