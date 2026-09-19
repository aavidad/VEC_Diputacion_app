\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:migracion:000108', 0
    )
);

DO $precondicion$
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_roles
         WHERE rolname = 'vec_contratacion_temporal_registrador_frontera'
           AND NOT rolcanlogin
           AND NOT rolsuper
           AND NOT rolcreatedb
           AND NOT rolcreaterole
           AND rolinherit
           AND NOT rolreplication
           AND NOT rolbypassrls
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_auth_members membresia
          JOIN pg_catalog.pg_roles miembro ON miembro.oid = membresia.member
         WHERE miembro.rolname = 'vec_contratacion_temporal_registrador_frontera'
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'rol registrador de frontera no preparado';
    END IF;
END
$precondicion$;

CREATE TABLE vec_contratacion_temporal.auditoria_frontera_ruta_exacta (
    evento_id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    correlacion_ref text NOT NULL,
    motivo text NOT NULL,
    superficie text NOT NULL,
    ruta text NOT NULL,
    actor_ref text,
    registrada_en timestamptz(6) NOT NULL,
    CHECK (correlacion_ref = 'corr_no_disponible' OR correlacion_ref ~ '^corr_[0-9a-f]{32}$'),
    CHECK (motivo IN ('autenticacion_requerida', 'acceso_denegado')),
    CHECK (superficie = 'api.contratacion_temporal.ruta_exacta'),
    CHECK (ruta ~ '^/api/vec/contratacion-temporal/[a-z0-9_-]+(/([a-z0-9_-]+))*$'),
    CHECK (actor_ref IS NULL OR actor_ref ~ '^[A-Za-z0-9:_-]{1,512}$'),
    CHECK (registrada_en = pg_catalog.date_trunc('microseconds', registrada_en))
);

CREATE INDEX auditoria_frontera_ruta_exacta_correlacion_idx
    ON vec_contratacion_temporal.auditoria_frontera_ruta_exacta (correlacion_ref, evento_id);

CREATE FUNCTION vec_contratacion_temporal.rechazar_mutacion_auditoria_frontera_ruta_exacta_v1()
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

CREATE TRIGGER auditoria_frontera_ruta_exacta_inmutable
BEFORE UPDATE OR DELETE ON vec_contratacion_temporal.auditoria_frontera_ruta_exacta
FOR EACH ROW EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_auditoria_frontera_ruta_exacta_v1();

CREATE TRIGGER auditoria_frontera_ruta_exacta_no_truncar
BEFORE TRUNCATE ON vec_contratacion_temporal.auditoria_frontera_ruta_exacta
FOR EACH STATEMENT EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_auditoria_frontera_ruta_exacta_v1();

CREATE FUNCTION vec_contratacion_temporal.registrar_auditoria_frontera_ruta_exacta_v1(
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
    v_segmentos text[];
BEGIN
    IF p_correlacion_ref IS NULL
       OR (p_correlacion_ref <> 'corr_no_disponible' AND p_correlacion_ref !~ '^corr_[0-9a-f]{32}$')
       OR p_motivo NOT IN ('autenticacion_requerida', 'acceso_denegado')
       OR p_superficie <> 'api.contratacion_temporal.ruta_exacta'
       OR p_ruta IS NULL
       OR pg_catalog.length(p_ruta) <= pg_catalog.length('/api/vec/contratacion-temporal/')
       OR pg_catalog.length(p_ruta) > 512
       OR p_ruta !~ '^/api/vec/contratacion-temporal/[a-z0-9_-]+(/([a-z0-9_-]+))*$'
       OR p_actor_ref IS NOT NULL AND (
            pg_catalog.length(p_actor_ref) > 512
            OR p_actor_ref <> pg_catalog.btrim(p_actor_ref)
            OR p_actor_ref !~ '^[A-Za-z0-9:_-]{1,512}$'
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'auditoria de frontera invalida';
    END IF;

    v_segmentos := pg_catalog.string_to_array(
        pg_catalog.substr(p_ruta, pg_catalog.length('/api/vec/') + 1), '/'
    );
    IF v_segmentos IS NULL OR v_segmentos[1] <> 'contratacion-temporal'
       OR EXISTS (
           SELECT 1 FROM pg_catalog.unnest(v_segmentos) AS segmento
            WHERE pg_catalog.length(segmento) NOT BETWEEN 1 AND 64
              OR segmento !~ '^[a-z0-9_-]+$'
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'auditoria de frontera invalida';
    END IF;

    INSERT INTO vec_contratacion_temporal.auditoria_frontera_ruta_exacta (
        correlacion_ref, motivo, superficie, ruta, actor_ref, registrada_en
    ) VALUES (
        p_correlacion_ref, p_motivo, p_superficie, p_ruta, p_actor_ref,
        pg_catalog.date_trunc('microseconds', pg_catalog.clock_timestamp())
    );
    RETURN true;
END
$funcion$;

ALTER TABLE vec_contratacion_temporal.auditoria_frontera_ruta_exacta ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contratacion_temporal.auditoria_frontera_ruta_exacta FORCE ROW LEVEL SECURITY;
CREATE POLICY auditoria_frontera_ruta_exacta_propietario_total
    ON vec_contratacion_temporal.auditoria_frontera_ruta_exacta
    TO vec_contratacion_temporal_propietario
    USING (true) WITH CHECK (true);

REVOKE ALL ON TABLE vec_contratacion_temporal.auditoria_frontera_ruta_exacta FROM PUBLIC;
REVOKE ALL ON SEQUENCE vec_contratacion_temporal.auditoria_frontera_ruta_exacta_evento_id_seq FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.rechazar_mutacion_auditoria_frontera_ruta_exacta_v1() FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.registrar_auditoria_frontera_ruta_exacta_v1(text,text,text,text,text) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_contratacion_temporal TO vec_contratacion_temporal_registrador_frontera;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.registrar_auditoria_frontera_ruta_exacta_v1(text,text,text,text,text)
    TO vec_contratacion_temporal_registrador_frontera;

COMMIT;
