BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:000001_preparacion_altas',
        0
    )
);

REVOKE ALL ON SCHEMA vec_contratacion_temporal FROM PUBLIC;

DO $prevalidacion$
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_namespace
         WHERE nspname = 'vec_contratacion_temporal'
           AND nspowner = current_user::regrole
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'esquema de contratación temporal ausente o ajeno';
    END IF;
    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_class AS objeto
          JOIN pg_catalog.pg_namespace AS espacio
            ON espacio.oid = objeto.relnamespace
         WHERE espacio.nspname = 'vec_contratacion_temporal'
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc AS procedimiento
          JOIN pg_catalog.pg_namespace AS espacio
            ON espacio.oid = procedimiento.pronamespace
         WHERE espacio.nspname = 'vec_contratacion_temporal'
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'migración inicial rechazada: el esquema no está vacío';
    END IF;
END
$prevalidacion$;

CREATE TABLE vec_contratacion_temporal.identidad_reserva_alta (
    ambito_hmac text PRIMARY KEY,
    reserva_ref text NOT NULL UNIQUE,
    expediente_ref text NOT NULL UNIQUE,
    numero_visible text NOT NULL UNIQUE,
    recibo_ref text NOT NULL UNIQUE,
    huella_peticion_hmac text NOT NULL,
    organizacion_ref text NOT NULL,
    actor_ref text NOT NULL,
    perfil_ref text NOT NULL,
    creada_en timestamptz NOT NULL,
    CONSTRAINT identidad_reserva_ambito_hmac_valido CHECK (
        ambito_hmac ~ (
            '^hmac-sha256:vec[.]contratacion-temporal[.]'
            || 'ambito-idempotencia/v1:[a-f0-9]{64}$'
        )
        AND ambito_hmac <>
            'hmac-sha256:vec.contratacion-temporal.ambito-idempotencia/v1:'
            || repeat('0', 64)
    ),
    CONSTRAINT identidad_reserva_huella_peticion_valida CHECK (
        huella_peticion_hmac ~ (
            '^hmac-sha256:vec[.]contratacion-temporal[.]'
            || 'huella-peticion/v1:[a-f0-9]{64}$'
        )
        AND huella_peticion_hmac <>
            'hmac-sha256:vec.contratacion-temporal.huella-peticion/v1:'
            || repeat('0', 64)
    ),
    CONSTRAINT identidad_reserva_referencias_validas CHECK (
        reserva_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND expediente_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND recibo_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND numero_visible ~ '^[0-9]{4}/[A-Za-z0-9._-]{1,40}$'
        AND organizacion_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND actor_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND perfil_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
    ),
    CONSTRAINT identidad_reserva_creada_canonica CHECK (
        creada_en = date_trunc('microseconds', creada_en)
    )
);

CREATE TABLE vec_contratacion_temporal.reserva_alta_version (
    ambito_hmac text NOT NULL,
    revision bigint NOT NULL,
    estado text NOT NULL,
    version_expediente bigint,
    auditoria_ref text,
    evento_ref text,
    confirmada_en timestamptz,
    registrada_en timestamptz NOT NULL,
    PRIMARY KEY (ambito_hmac, revision),
    CONSTRAINT reserva_version_identidad_fk
        FOREIGN KEY (ambito_hmac)
        REFERENCES vec_contratacion_temporal.identidad_reserva_alta(ambito_hmac)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT reserva_version_revision_positiva CHECK (revision > 0),
    CONSTRAINT reserva_version_estado_valido CHECK (
        estado IN ('reservada', 'confirmada')
    ),
    CONSTRAINT reserva_version_confirmacion_coherente CHECK (
        (
            estado = 'reservada'
            AND version_expediente IS NULL
            AND auditoria_ref IS NULL
            AND evento_ref IS NULL
            AND confirmada_en IS NULL
        )
        OR
        (
            estado = 'confirmada'
            AND version_expediente > 0
            AND auditoria_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
            AND evento_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
            AND confirmada_en IS NOT NULL
            AND confirmada_en = date_trunc('microseconds', confirmada_en)
        )
    ),
    CONSTRAINT reserva_version_registrada_canonica CHECK (
        registrada_en = date_trunc('microseconds', registrada_en)
    )
);

CREATE TABLE vec_contratacion_temporal.reserva_alta_actual (
    ambito_hmac text PRIMARY KEY,
    revision bigint NOT NULL,
    CONSTRAINT reserva_actual_version_fk
        FOREIGN KEY (ambito_hmac, revision)
        REFERENCES vec_contratacion_temporal.reserva_alta_version(
            ambito_hmac,
            revision
        )
        ON UPDATE RESTRICT ON DELETE RESTRICT
        DEFERRABLE INITIALLY IMMEDIATE
);

CREATE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog
AS $funcion$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = 'la historia de contratación temporal es inmutable';
END
$funcion$;

CREATE TRIGGER identidad_reserva_alta_inmutable
BEFORE UPDATE OR DELETE
ON vec_contratacion_temporal.identidad_reserva_alta
FOR EACH ROW
EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();

CREATE TRIGGER reserva_alta_version_inmutable
BEFORE UPDATE OR DELETE
ON vec_contratacion_temporal.reserva_alta_version
FOR EACH ROW
EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();

CREATE FUNCTION vec_contratacion_temporal.preparar_alta_v1(
    p_operacion jsonb
)
RETURNS TABLE (
    resultado text,
    ambito_hmac text,
    reserva_ref text,
    expediente_ref text,
    numero_visible text,
    recibo_ref text,
    huella_peticion_hmac text,
    organizacion_ref text,
    actor_ref text,
    perfil_ref text,
    estado text,
    version_expediente bigint,
    auditoria_ref text,
    evento_ref text,
    confirmada_en timestamptz
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_insertada boolean;
    v_filas bigint;
    v_identidad vec_contratacion_temporal.identidad_reserva_alta%ROWTYPE;
    v_version vec_contratacion_temporal.reserva_alta_version%ROWTYPE;
    v_ahora timestamptz := date_trunc('microseconds', clock_timestamp());
    v_claves text[];
    v_claves_referencias text[];
BEGIN
    IF session_user = current_user
       OR NOT pg_catalog.pg_has_role(
           session_user,
           'vec_contratacion_temporal_ejecutor',
           'MEMBER'
       )
       OR pg_catalog.pg_has_role(
           session_user,
           'vec_contratacion_temporal_propietario',
           'MEMBER'
       )
       OR pg_catalog.pg_has_role(
           session_user,
           'vec_contratacion_temporal_migrador',
           'MEMBER'
       ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'identidad de ejecución no autorizada';
    END IF;

    IF p_operacion IS NULL OR jsonb_typeof(p_operacion) <> 'object' THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'operación de preparación inválida';
    END IF;

    SELECT array_agg(clave ORDER BY clave)
      INTO v_claves
      FROM jsonb_object_keys(p_operacion) AS claves(clave);
    IF v_claves IS DISTINCT FROM ARRAY[
        'actor_ref',
        'ambito_hmac',
        'esquema',
        'huella_peticion_hmac',
        'organizacion_ref',
        'perfil_ref',
        'referencias_candidatas',
        'reserva_ref_candidata'
    ]::text[] THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'campos de preparación inválidos';
    END IF;

    IF jsonb_typeof(p_operacion -> 'referencias_candidatas') <> 'object' THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'referencias candidatas inválidas';
    END IF;
    SELECT array_agg(clave ORDER BY clave)
      INTO v_claves_referencias
      FROM jsonb_object_keys(
          p_operacion -> 'referencias_candidatas'
      ) AS claves(clave);
    IF v_claves_referencias IS DISTINCT FROM ARRAY[
        'expediente_ref',
        'numero_visible',
        'recibo_ref'
    ]::text[] THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'campos de referencias inválidos';
    END IF;

    IF p_operacion ->> 'esquema'
           <> 'vec.contratacion-temporal.preparar-alta.v1'
       OR coalesce(p_operacion ->> 'ambito_hmac', '')
           !~ (
               '^hmac-sha256:vec[.]contratacion-temporal[.]'
               || 'ambito-idempotencia/v1:[a-f0-9]{64}$'
           )
       OR p_operacion ->> 'ambito_hmac' =
           'hmac-sha256:vec.contratacion-temporal.ambito-idempotencia/v1:'
           || repeat('0', 64)
       OR coalesce(p_operacion ->> 'huella_peticion_hmac', '')
           !~ (
               '^hmac-sha256:vec[.]contratacion-temporal[.]'
               || 'huella-peticion/v1:[a-f0-9]{64}$'
           )
       OR p_operacion ->> 'huella_peticion_hmac' =
           'hmac-sha256:vec.contratacion-temporal.huella-peticion/v1:'
           || repeat('0', 64)
       OR coalesce(p_operacion ->> 'reserva_ref_candidata', '')
           !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR coalesce(
           p_operacion #>> '{referencias_candidatas,expediente_ref}',
           ''
       ) !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR coalesce(
           p_operacion #>> '{referencias_candidatas,numero_visible}',
           ''
       ) !~ '^[0-9]{4}/[A-Za-z0-9._-]{1,40}$'
       OR coalesce(
           p_operacion #>> '{referencias_candidatas,recibo_ref}',
           ''
       ) !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR coalesce(p_operacion ->> 'organizacion_ref', '')
           !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR coalesce(p_operacion ->> 'actor_ref', '')
           !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR coalesce(p_operacion ->> 'perfil_ref', '')
           !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'contenido de preparación inválido';
    END IF;

    INSERT INTO vec_contratacion_temporal.identidad_reserva_alta (
        ambito_hmac,
        reserva_ref,
        expediente_ref,
        numero_visible,
        recibo_ref,
        huella_peticion_hmac,
        organizacion_ref,
        actor_ref,
        perfil_ref,
        creada_en
    )
    VALUES (
        p_operacion ->> 'ambito_hmac',
        p_operacion ->> 'reserva_ref_candidata',
        p_operacion #>> '{referencias_candidatas,expediente_ref}',
        p_operacion #>> '{referencias_candidatas,numero_visible}',
        p_operacion #>> '{referencias_candidatas,recibo_ref}',
        p_operacion ->> 'huella_peticion_hmac',
        p_operacion ->> 'organizacion_ref',
        p_operacion ->> 'actor_ref',
        p_operacion ->> 'perfil_ref',
        v_ahora
    )
    ON CONFLICT ON CONSTRAINT identidad_reserva_alta_pkey DO NOTHING;
    GET DIAGNOSTICS v_filas = ROW_COUNT;
    v_insertada := v_filas = 1;

    SELECT *
      INTO STRICT v_identidad
      FROM vec_contratacion_temporal.identidad_reserva_alta AS identidad
     WHERE identidad.ambito_hmac = p_operacion ->> 'ambito_hmac'
     FOR UPDATE;

    IF v_identidad.huella_peticion_hmac
           <> p_operacion ->> 'huella_peticion_hmac'
       OR v_identidad.organizacion_ref
           <> p_operacion ->> 'organizacion_ref'
       OR v_identidad.actor_ref <> p_operacion ->> 'actor_ref'
       OR v_identidad.perfil_ref <> p_operacion ->> 'perfil_ref' THEN
        -- No devolvemos la identidad ya persistida ante un conflicto. Así se
        -- conserva el contrato tabular (sin NULL incompatibles con el
        -- adaptador) sin revelar referencias ni huellas de otro intento.
        RETURN QUERY
        SELECT
            'idempotencia_reutilizada'::text,
            p_operacion ->> 'ambito_hmac',
            p_operacion ->> 'reserva_ref_candidata',
            p_operacion #>> '{referencias_candidatas,expediente_ref}',
            p_operacion #>> '{referencias_candidatas,numero_visible}',
            p_operacion #>> '{referencias_candidatas,recibo_ref}',
            p_operacion ->> 'huella_peticion_hmac',
            p_operacion ->> 'organizacion_ref',
            p_operacion ->> 'actor_ref',
            p_operacion ->> 'perfil_ref',
            'reservada'::text,
            NULL::bigint,
            NULL::text,
            NULL::text,
            NULL::timestamptz;
        RETURN;
    END IF;

    IF v_insertada THEN
        INSERT INTO vec_contratacion_temporal.reserva_alta_version (
            ambito_hmac,
            revision,
            estado,
            registrada_en
        )
        VALUES (
            v_identidad.ambito_hmac,
            1,
            'reservada',
            v_ahora
        );
        INSERT INTO vec_contratacion_temporal.reserva_alta_actual (
            ambito_hmac,
            revision
        )
        VALUES (
            v_identidad.ambito_hmac,
            1
        );
    END IF;

    SELECT version.*
      INTO STRICT v_version
      FROM vec_contratacion_temporal.reserva_alta_actual AS actual
      JOIN vec_contratacion_temporal.reserva_alta_version AS version
        ON version.ambito_hmac = actual.ambito_hmac
       AND version.revision = actual.revision
     WHERE actual.ambito_hmac = v_identidad.ambito_hmac;

    RETURN QUERY
    SELECT
        CASE
            WHEN v_version.estado = 'confirmada' THEN 'confirmada'
            WHEN v_insertada THEN 'reservada'
            ELSE 'reutilizada'
        END,
        v_identidad.ambito_hmac,
        v_identidad.reserva_ref,
        v_identidad.expediente_ref,
        v_identidad.numero_visible,
        v_identidad.recibo_ref,
        v_identidad.huella_peticion_hmac,
        v_identidad.organizacion_ref,
        v_identidad.actor_ref,
        v_identidad.perfil_ref,
        v_version.estado,
        v_version.version_expediente,
        v_version.auditoria_ref,
        v_version.evento_ref,
        v_version.confirmada_en;
END
$funcion$;

ALTER TABLE vec_contratacion_temporal.identidad_reserva_alta
    OWNER TO vec_contratacion_temporal_propietario;
ALTER TABLE vec_contratacion_temporal.reserva_alta_version
    OWNER TO vec_contratacion_temporal_propietario;
ALTER TABLE vec_contratacion_temporal.reserva_alta_actual
    OWNER TO vec_contratacion_temporal_propietario;
ALTER FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1()
    OWNER TO vec_contratacion_temporal_propietario;
ALTER FUNCTION vec_contratacion_temporal.preparar_alta_v1(jsonb)
    OWNER TO vec_contratacion_temporal_propietario;

REVOKE ALL ON ALL TABLES IN SCHEMA vec_contratacion_temporal FROM PUBLIC;
REVOKE ALL ON ALL FUNCTIONS IN SCHEMA vec_contratacion_temporal FROM PUBLIC;
REVOKE ALL ON ALL SEQUENCES IN SCHEMA vec_contratacion_temporal FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_contratacion_temporal
    TO vec_contratacion_temporal_ejecutor;
GRANT EXECUTE ON FUNCTION
    vec_contratacion_temporal.preparar_alta_v1(jsonb)
    TO vec_contratacion_temporal_ejecutor;

ALTER DEFAULT PRIVILEGES
    FOR ROLE vec_contratacion_temporal_propietario
    IN SCHEMA vec_contratacion_temporal
    REVOKE ALL ON TABLES FROM PUBLIC;
ALTER DEFAULT PRIVILEGES
    FOR ROLE vec_contratacion_temporal_propietario
    IN SCHEMA vec_contratacion_temporal
    REVOKE ALL ON FUNCTIONS FROM PUBLIC;
ALTER DEFAULT PRIVILEGES
    FOR ROLE vec_contratacion_temporal_propietario
    IN SCHEMA vec_contratacion_temporal
    REVOKE ALL ON SEQUENCES FROM PUBLIC;

COMMIT;
