BEGIN;
SET LOCAL ROLE vec_identidad_sesiones_v1_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

DO $prevalidacion$
BEGIN
    IF to_regnamespace('vec_identidad_sesiones_v1') IS NOT NULL
       OR to_regclass('vec_autorizacion.sesion_autenticacion_v1') IS NULL
       OR to_regclass('vec_autorizacion.control_sesion_v1') IS NULL
       OR to_regclass('vec_autorizacion.control_sesion_actual_v1') IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para registro de identidad V1';
    END IF;
END
$prevalidacion$;

CREATE SCHEMA vec_identidad_sesiones_v1
    AUTHORIZATION vec_identidad_sesiones_v1_propietario;
REVOKE ALL ON SCHEMA vec_identidad_sesiones_v1 FROM PUBLIC;

ALTER DEFAULT PRIVILEGES FOR ROLE vec_identidad_sesiones_v1_propietario
    IN SCHEMA vec_identidad_sesiones_v1 REVOKE ALL ON TABLES FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_identidad_sesiones_v1_propietario
    REVOKE ALL ON FUNCTIONS FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_identidad_sesiones_v1_propietario
    REVOKE ALL ON TYPES FROM PUBLIC;

CREATE FUNCTION vec_identidad_sesiones_v1.texto_tecnico_valido(
    p_valor text,
    p_maximo integer
)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
    SELECT p_valor IS NOT NULL AND p_maximo > 0
       AND octet_length(p_valor) BETWEEN 1 AND p_maximo
       AND p_valor ~ '^[!-~]+$'
$funcion$;

CREATE FUNCTION vec_identidad_sesiones_v1.referencia_valida(
    p_valor text,
    p_prefijo text
)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
    SELECT p_prefijo ~ '^[a-z]{3}_$'
       AND p_valor ~ ('^' || p_prefijo || '[A-Za-z0-9_-]{22,128}$')
$funcion$;

CREATE FUNCTION vec_identidad_sesiones_v1.coordenadas_hmac_validas(
    p_esquema text,
    p_dominio_ref text,
    p_clave_id text,
    p_clave_version bigint
)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
    SELECT p_esquema = 'vec.identidad.hmac-sha256.v1'
       AND vec_identidad_sesiones_v1.referencia_valida(
           p_dominio_ref, 'idh_'
       ) IS TRUE
       AND vec_identidad_sesiones_v1.texto_tecnico_valido(
           p_clave_id, 128
       ) IS TRUE
       AND p_clave_version BETWEEN 1 AND 9223372036854775807
$funcion$;

CREATE FUNCTION vec_identidad_sesiones_v1.huella_hmac_valida(p_valor bytea)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
    SELECT p_valor IS NOT NULL
       AND octet_length(p_valor) = 32
       AND p_valor <> decode(repeat('00', 32), 'hex')
$funcion$;

CREATE FUNCTION vec_identidad_sesiones_v1.rechazar_mutacion()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, pg_temp
AS $funcion$
BEGIN
    RAISE EXCEPTION USING ERRCODE = '55000',
        MESSAGE = 'historia de identidad inmutable';
END
$funcion$;

CREATE TABLE vec_identidad_sesiones_v1.cuenta (
    cuenta_ref text PRIMARY KEY,
    cuenta_privilegiada boolean NOT NULL,
    cuenta_ordinaria_ref text,
    provisionada_en timestamptz(6) NOT NULL,
    acto_ref text NOT NULL UNIQUE,
    CONSTRAINT cuenta_referencia CHECK (
        vec_identidad_sesiones_v1.referencia_valida(
            cuenta_ref, 'cta_'
        ) IS TRUE
    ),
    CONSTRAINT cuenta_tipo CHECK (
        (cuenta_privilegiada
         AND cuenta_ordinaria_ref IS NOT NULL
         AND cuenta_ordinaria_ref <> cuenta_ref)
        OR
        (NOT cuenta_privilegiada AND cuenta_ordinaria_ref IS NULL)
    ),
    CONSTRAINT cuenta_acto CHECK (
        vec_identidad_sesiones_v1.referencia_valida(
            acto_ref, 'opr_'
        ) IS TRUE
    ),
    FOREIGN KEY (cuenta_ordinaria_ref)
        REFERENCES vec_identidad_sesiones_v1.cuenta(cuenta_ref)
);

CREATE TABLE vec_identidad_sesiones_v1.alias_hmac_cuenta (
    alias_ref text PRIMARY KEY,
    cuenta_ref text NOT NULL
        REFERENCES vec_identidad_sesiones_v1.cuenta(cuenta_ref),
    esquema_hmac text NOT NULL,
    dominio_hmac_ref text NOT NULL,
    clave_hmac_id text NOT NULL,
    clave_hmac_version bigint NOT NULL,
    cuenta_id_hmac bytea NOT NULL,
    sujeto_id_hmac bytea NOT NULL,
    registrado_en timestamptz(6) NOT NULL,
    acto_ref text NOT NULL UNIQUE,
    CONSTRAINT alias_referencia CHECK (
        vec_identidad_sesiones_v1.referencia_valida(
            alias_ref, 'ali_'
        ) IS TRUE
    ),
    CONSTRAINT alias_coordenadas CHECK (
        vec_identidad_sesiones_v1.coordenadas_hmac_validas(
            esquema_hmac, dominio_hmac_ref,
            clave_hmac_id, clave_hmac_version
        ) IS TRUE
    ),
    CONSTRAINT alias_huellas CHECK (
        vec_identidad_sesiones_v1.huella_hmac_valida(
            cuenta_id_hmac
        ) IS TRUE
        AND vec_identidad_sesiones_v1.huella_hmac_valida(
            sujeto_id_hmac
        ) IS TRUE
        AND cuenta_id_hmac <> sujeto_id_hmac
    ),
    CONSTRAINT alias_acto CHECK (
        vec_identidad_sesiones_v1.referencia_valida(
            acto_ref, 'opr_'
        ) IS TRUE
    ),
    UNIQUE (
        esquema_hmac, dominio_hmac_ref, clave_hmac_id,
        clave_hmac_version, cuenta_id_hmac
    )
);

CREATE TABLE vec_identidad_sesiones_v1.estado_cuenta (
    cuenta_ref text NOT NULL
        REFERENCES vec_identidad_sesiones_v1.cuenta(cuenta_ref),
    revision numeric(20, 0) NOT NULL,
    estado text NOT NULL,
    registrada_en timestamptz(6) NOT NULL,
    acto_ref text NOT NULL UNIQUE,
    PRIMARY KEY (cuenta_ref, revision),
    CONSTRAINT estado_cuenta_revision CHECK (
        revision BETWEEN 1 AND 18446744073709551615
    ),
    CONSTRAINT estado_cuenta_cerrado CHECK (
        estado IN ('activa', 'inactiva')
    ),
    CONSTRAINT estado_cuenta_acto CHECK (
        vec_identidad_sesiones_v1.referencia_valida(
            acto_ref, 'opr_'
        ) IS TRUE
    )
);

CREATE TABLE vec_identidad_sesiones_v1.estado_cuenta_actual (
    cuenta_ref text PRIMARY KEY,
    revision numeric(20, 0) NOT NULL,
    actualizada_en timestamptz(6) NOT NULL,
    acto_ref text NOT NULL,
    FOREIGN KEY (cuenta_ref, revision)
        REFERENCES vec_identidad_sesiones_v1.estado_cuenta(
            cuenta_ref, revision
        ),
    CONSTRAINT estado_cuenta_actual_acto CHECK (
        vec_identidad_sesiones_v1.referencia_valida(
            acto_ref, 'opr_'
        ) IS TRUE
    )
);

CREATE TABLE vec_identidad_sesiones_v1.consumo_asercion (
    operacion_ref text PRIMARY KEY,
    esquema_hmac text NOT NULL,
    dominio_hmac_ref text NOT NULL,
    clave_hmac_id text NOT NULL,
    clave_hmac_version bigint NOT NULL,
    asercion_id_hmac bytea NOT NULL,
    sesion_id_hmac bytea NOT NULL,
    sujeto_id_hmac bytea NOT NULL,
    cuenta_id_hmac bytea NOT NULL,
    cuenta_ordinaria_id_hmac bytea,
    autenticacion_ref text NOT NULL UNIQUE,
    autenticacion_huella_sha256 text NOT NULL,
    asercion_ref text NOT NULL UNIQUE,
    sesion_ref text NOT NULL UNIQUE
        REFERENCES vec_autorizacion.sesion_autenticacion_v1(sesion_ref),
    control_sesion_ref text NOT NULL UNIQUE,
    control_sesion_revision numeric(20, 0) NOT NULL,
    cuenta_ref text NOT NULL
        REFERENCES vec_identidad_sesiones_v1.cuenta(cuenta_ref),
    cuenta_revision numeric(20, 0) NOT NULL,
    cuenta_ordinaria_ref text NOT NULL
        REFERENCES vec_identidad_sesiones_v1.cuenta(cuenta_ref),
    cuenta_ordinaria_revision numeric(20, 0) NOT NULL,
    consumida_en timestamptz(6) NOT NULL,
    CONSTRAINT consumo_operacion CHECK (
        vec_identidad_sesiones_v1.referencia_valida(
            operacion_ref, 'opr_'
        ) IS TRUE
    ),
    CONSTRAINT consumo_coordenadas CHECK (
        vec_identidad_sesiones_v1.coordenadas_hmac_validas(
            esquema_hmac, dominio_hmac_ref,
            clave_hmac_id, clave_hmac_version
        ) IS TRUE
    ),
    CONSTRAINT consumo_huellas CHECK (
        vec_identidad_sesiones_v1.huella_hmac_valida(
            asercion_id_hmac
        ) IS TRUE
        AND vec_identidad_sesiones_v1.huella_hmac_valida(
            sesion_id_hmac
        ) IS TRUE
        AND vec_identidad_sesiones_v1.huella_hmac_valida(
            sujeto_id_hmac
        ) IS TRUE
        AND vec_identidad_sesiones_v1.huella_hmac_valida(
            cuenta_id_hmac
        ) IS TRUE
        AND asercion_id_hmac <> sesion_id_hmac
        AND asercion_id_hmac <> sujeto_id_hmac
        AND asercion_id_hmac <> cuenta_id_hmac
        AND sesion_id_hmac <> sujeto_id_hmac
        AND sesion_id_hmac <> cuenta_id_hmac
        AND sujeto_id_hmac <> cuenta_id_hmac
        AND (
            cuenta_ordinaria_id_hmac IS NULL
            OR (
                vec_identidad_sesiones_v1.huella_hmac_valida(
                    cuenta_ordinaria_id_hmac
                ) IS TRUE
                AND cuenta_ordinaria_id_hmac <> asercion_id_hmac
                AND cuenta_ordinaria_id_hmac <> sesion_id_hmac
                AND cuenta_ordinaria_id_hmac <> sujeto_id_hmac
                AND cuenta_ordinaria_id_hmac <> cuenta_id_hmac
            )
        )
        AND autenticacion_huella_sha256 ~ '^[0-9a-f]{64}$'
        AND autenticacion_huella_sha256 <> repeat('0', 64)
    ),
    CONSTRAINT consumo_referencias CHECK (
        vec_identidad_sesiones_v1.referencia_valida(
            autenticacion_ref, 'aut_'
        ) IS TRUE
        AND vec_identidad_sesiones_v1.referencia_valida(
            asercion_ref, 'ase_'
        ) IS TRUE
        AND vec_identidad_sesiones_v1.referencia_valida(
            sesion_ref, 'ses_'
        ) IS TRUE
        AND vec_identidad_sesiones_v1.referencia_valida(
            control_sesion_ref, 'cse_'
        ) IS TRUE
    ),
    CONSTRAINT consumo_control_revision CHECK (
        control_sesion_revision BETWEEN 1 AND 18446744073709551615
    ),
    CONSTRAINT consumo_revisiones_cuenta CHECK (
        cuenta_revision BETWEEN 1 AND 18446744073709551615
        AND cuenta_ordinaria_revision BETWEEN 1 AND 18446744073709551615
    ),
    FOREIGN KEY (
        sesion_ref, autenticacion_ref, autenticacion_huella_sha256,
        cuenta_ref, cuenta_ordinaria_ref
    ) REFERENCES vec_autorizacion.sesion_autenticacion_v1(
        sesion_ref, autenticacion_ref, autenticacion_huella_sha256,
        cuenta_ref, cuenta_ordinaria_ref
    ),
    FOREIGN KEY (
        sesion_ref, control_sesion_ref, control_sesion_revision
    ) REFERENCES vec_autorizacion.control_sesion_v1(
        sesion_ref, control_sesion_ref, revision
    ),
    FOREIGN KEY (cuenta_ref, cuenta_revision)
        REFERENCES vec_identidad_sesiones_v1.estado_cuenta(
            cuenta_ref, revision
        ),
    FOREIGN KEY (cuenta_ordinaria_ref, cuenta_ordinaria_revision)
        REFERENCES vec_identidad_sesiones_v1.estado_cuenta(
            cuenta_ref, revision
        ),
    UNIQUE (
        esquema_hmac, dominio_hmac_ref, clave_hmac_id,
        clave_hmac_version, asercion_id_hmac
    ),
    UNIQUE (
        dominio_hmac_ref, autenticacion_huella_sha256
    )
);

CREATE INDEX consumo_asercion_sesion_hmac_v1_idx
    ON vec_identidad_sesiones_v1.consumo_asercion (
        esquema_hmac, dominio_hmac_ref, clave_hmac_id,
        clave_hmac_version, sesion_id_hmac
    );

CREATE FUNCTION vec_identidad_sesiones_v1.validar_avance_estado_cuenta()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, pg_temp
AS $funcion$
BEGIN
    IF NEW.cuenta_ref IS DISTINCT FROM OLD.cuenta_ref
       OR NEW.revision IS DISTINCT FROM OLD.revision + 1
       OR NEW.actualizada_en <= OLD.actualizada_en THEN
        RAISE EXCEPTION USING ERRCODE = '23514',
            MESSAGE = 'avance de estado de cuenta invalido';
    END IF;
    RETURN NEW;
END
$funcion$;

CREATE TRIGGER estado_cuenta_actual_avance
    BEFORE UPDATE ON vec_identidad_sesiones_v1.estado_cuenta_actual
    FOR EACH ROW EXECUTE FUNCTION
        vec_identidad_sesiones_v1.validar_avance_estado_cuenta();

DO $protecciones$
DECLARE
    tabla text;
BEGIN
    FOREACH tabla IN ARRAY ARRAY[
        'cuenta', 'alias_hmac_cuenta', 'estado_cuenta', 'consumo_asercion'
    ] LOOP
        EXECUTE format(
            'CREATE TRIGGER %I_inmutable BEFORE UPDATE OR DELETE ON vec_identidad_sesiones_v1.%I FOR EACH ROW EXECUTE FUNCTION vec_identidad_sesiones_v1.rechazar_mutacion()',
            tabla, tabla
        );
        EXECUTE format(
            'CREATE TRIGGER %I_no_truncar BEFORE TRUNCATE ON vec_identidad_sesiones_v1.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_identidad_sesiones_v1.rechazar_mutacion()',
            tabla, tabla
        );
    END LOOP;
    EXECUTE 'CREATE TRIGGER estado_cuenta_actual_no_eliminar BEFORE DELETE OR TRUNCATE ON vec_identidad_sesiones_v1.estado_cuenta_actual FOR EACH STATEMENT EXECUTE FUNCTION vec_identidad_sesiones_v1.rechazar_mutacion()';
END
$protecciones$;

DO $rls$
DECLARE
    tabla text;
BEGIN
    FOREACH tabla IN ARRAY ARRAY[
        'cuenta', 'alias_hmac_cuenta', 'estado_cuenta',
        'estado_cuenta_actual', 'consumo_asercion'
    ] LOOP
        EXECUTE format(
            'ALTER TABLE vec_identidad_sesiones_v1.%I ENABLE ROW LEVEL SECURITY',
            tabla
        );
        EXECUTE format(
            'ALTER TABLE vec_identidad_sesiones_v1.%I FORCE ROW LEVEL SECURITY',
            tabla
        );
        EXECUTE format(
            'CREATE POLICY acceso_propietario_exacto ON vec_identidad_sesiones_v1.%I FOR ALL TO vec_identidad_sesiones_v1_propietario USING (current_user = %L) WITH CHECK (current_user = %L)',
            tabla,
            'vec_identidad_sesiones_v1_propietario',
            'vec_identidad_sesiones_v1_propietario'
        );
    END LOOP;
END
$rls$;

REVOKE ALL ON ALL TABLES IN SCHEMA vec_identidad_sesiones_v1 FROM PUBLIC;
REVOKE ALL ON ALL FUNCTIONS IN SCHEMA vec_identidad_sesiones_v1 FROM PUBLIC;
-- Los tipos compuestos implícitos de las tablas no heredan el cierre de
-- ALTER DEFAULT PRIVILEGES ON TYPES en PostgreSQL 18.
REVOKE ALL ON TYPE vec_identidad_sesiones_v1.cuenta FROM PUBLIC;
REVOKE ALL ON TYPE vec_identidad_sesiones_v1.alias_hmac_cuenta FROM PUBLIC;
REVOKE ALL ON TYPE vec_identidad_sesiones_v1.estado_cuenta FROM PUBLIC;
REVOKE ALL ON TYPE vec_identidad_sesiones_v1.estado_cuenta_actual FROM PUBLIC;
REVOKE ALL ON TYPE vec_identidad_sesiones_v1.consumo_asercion FROM PUBLIC;
COMMIT;
