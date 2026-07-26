-- O4-05/C2-B: persistencia minimizada y encadenada de accesos RRHH.
-- No crea lectores ni cursor. La historia de expedientes no dispone todavía
-- de un corte global monotónico alineado con COMMIT; usar el reloj sería
-- insuficiente para una paginación histórica estable.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:o4_05:consultas_rrhh:migraciones',
        0
    )
);

SELECT control
  FROM vec_contratacion_temporal.control_migracion_cobertura_o4
 WHERE control
   AND version_esquema = 15
 FOR UPDATE;

DO $prevalidacion$
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.control_migracion_cobertura_o4
         WHERE control
           AND version_esquema = 15
    ) OR pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.recuperar_resultado_propio_decision_cobertura_o405_v1(jsonb)'
    ) IS NULL OR pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.o404e_claves_exactas_v1(jsonb,text[])'
    ) IS NULL OR pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.encuadrar_texto_v1(text)'
    ) IS NULL OR pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.instante_utc_v1(timestamp with time zone)'
    ) IS NULL OR NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_roles
         WHERE rolname = 'vec_contratacion_temporal_consultor_rrhh'
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
          JOIN pg_catalog.pg_roles miembro
            ON miembro.oid = membresia.member
         WHERE miembro.rolname =
               'vec_contratacion_temporal_consultor_rrhh'
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_auth_members membresia
          JOIN pg_catalog.pg_roles grupo
            ON grupo.oid = membresia.roleid
          JOIN pg_catalog.pg_roles miembro
            ON miembro.oid = membresia.member
         WHERE grupo.rolname =
               'vec_contratacion_temporal_consultor_rrhh'
           AND (
               membresia.admin_option
               OR NOT membresia.inherit_option
               OR membresia.set_option
               OR NOT miembro.rolcanlogin
               OR miembro.rolsuper
               OR miembro.rolcreatedb
               OR miembro.rolcreaterole
               OR miembro.rolreplication
               OR miembro.rolbypassrls
               OR (
                   SELECT pg_catalog.count(*)
                     FROM pg_catalog.pg_auth_members directas
                    WHERE directas.member = miembro.oid
               ) <> 1
           )
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_auth_members directa
          JOIN pg_catalog.pg_roles grupo
            ON grupo.oid = directa.roleid
          JOIN pg_catalog.pg_auth_members puente
            ON puente.roleid = directa.member
         WHERE grupo.rolname =
               'vec_contratacion_temporal_consultor_rrhh'
    ) OR pg_catalog.to_regclass(
        'vec_contratacion_temporal.control_migracion_consultas_rrhh'
    ) IS NOT NULL OR pg_catalog.to_regclass(
        'vec_contratacion_temporal.control_cadena_accesos_rrhh'
    ) IS NOT NULL OR pg_catalog.to_regclass(
        'vec_contratacion_temporal.registro_acceso_rrhh'
    ) IS NOT NULL OR pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.registrar_acceso_rrhh_interno_v1(jsonb)'
    ) IS NOT NULL THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'estado incompatible para registro de accesos RRHH';
    END IF;
END
$prevalidacion$;

CREATE TABLE
vec_contratacion_temporal.control_migracion_consultas_rrhh (
    control boolean PRIMARY KEY DEFAULT true CHECK (control),
    version_esquema integer NOT NULL CHECK (version_esquema BETWEEN 1 AND 99),
    actualizada_en timestamptz(6) NOT NULL CHECK (
        actualizada_en = pg_catalog.date_trunc(
            'microseconds',
            actualizada_en
        )
    )
);

INSERT INTO
vec_contratacion_temporal.control_migracion_consultas_rrhh (
    control,
    version_esquema,
    actualizada_en
) VALUES (
    true,
    1,
    pg_catalog.date_trunc(
        'microseconds',
        pg_catalog.clock_timestamp()
    )
);

CREATE TABLE vec_contratacion_temporal.control_cadena_accesos_rrhh (
    control boolean PRIMARY KEY DEFAULT true CHECK (control),
    ultima_secuencia numeric(20, 0) NOT NULL CHECK (
        ultima_secuencia BETWEEN 0 AND 9007199254740991::numeric
    ),
    cabeza_sha256 text NOT NULL CHECK (
        cabeza_sha256 ~ '^[0-9a-f]{64}$'
    ),
    actualizada_en timestamptz(6) NOT NULL CHECK (
        actualizada_en = pg_catalog.date_trunc(
            'microseconds',
            actualizada_en
        )
    )
);

INSERT INTO vec_contratacion_temporal.control_cadena_accesos_rrhh (
    control,
    ultima_secuencia,
    cabeza_sha256,
    actualizada_en
) VALUES (
    true,
    0,
    pg_catalog.repeat('0', 64),
    pg_catalog.date_trunc(
        'microseconds',
        pg_catalog.clock_timestamp()
    )
);

CREATE TABLE vec_contratacion_temporal.registro_acceso_rrhh (
    acceso_ref text PRIMARY KEY,
    secuencia numeric(20, 0) NOT NULL UNIQUE,
    tipo_consulta text NOT NULL,
    decision_ref text NOT NULL UNIQUE,
    decision_huella_sha256 text NOT NULL UNIQUE,
    sesion_id text NOT NULL,
    sesion_huella_sha256 text NOT NULL,
    actor_ref text NOT NULL,
    perfil_id text NOT NULL,
    perfil_version numeric(20, 0) NOT NULL,
    organizacion_ref text NOT NULL,
    ambito_ref text NOT NULL,
    modulo_id text NOT NULL,
    accion text NOT NULL,
    finalidad text NOT NULL,
    audiencia text NOT NULL,
    recurso_tipo text NOT NULL,
    recurso_ref text NOT NULL,
    dominio_huella_consulta text NOT NULL,
    consulta_huella_sha256 text NOT NULL,
    correlacion_ref text NOT NULL UNIQUE,
    expediente_ref text,
    version_expediente numeric(20, 0),
    capacidad_huella_sha256 text NOT NULL UNIQUE,
    consumo_vec_huella_sha256 text NOT NULL UNIQUE,
    auditoria_vec_ref text NOT NULL UNIQUE,
    auditoria_vec_huella_sha256 text NOT NULL UNIQUE,
    resultado_huella_sha256 text NOT NULL,
    total integer NOT NULL,
    resultado_generico text NOT NULL,
    prueba_canonica bytea NOT NULL,
    anterior_sha256 text NOT NULL,
    huella_sha256 text NOT NULL UNIQUE,
    registrada_en timestamptz(6) NOT NULL,
    CHECK (acceso_ref ~ '^acceso:rrhh:[0-9a-f]{32}$'),
    CHECK (
        acceso_ref = 'acceso:rrhh:' || pg_catalog.substr(
            pg_catalog.encode(
                pg_catalog.sha256(
                    pg_catalog.convert_to(
                        'acceso:rrhh:' || consumo_vec_huella_sha256,
                        'UTF8'
                    )
                ),
                'hex'
            ),
            1,
            32
        )
    ),
    CHECK (secuencia BETWEEN 1 AND 9007199254740991::numeric),
    CHECK (tipo_consulta IN ('cuadro', 'detalle')),
    CHECK (
        decision_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND sesion_id ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND actor_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND perfil_id ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND organizacion_ref ~
            '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND ambito_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND recurso_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND correlacion_ref ~
            '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND auditoria_vec_ref ~
            '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND (
            expediente_ref IS NULL
            OR expediente_ref ~
                '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        )
    ),
    CHECK (
        perfil_version BETWEEN 1 AND 9007199254740991::numeric
        AND (
            version_expediente IS NULL
            OR version_expediente BETWEEN
                1 AND 9007199254740991::numeric
        )
    ),
    CHECK (
        decision_huella_sha256 ~ '^[0-9a-f]{64}$'
        AND sesion_huella_sha256 ~ '^[0-9a-f]{64}$'
        AND consulta_huella_sha256 ~ '^[0-9a-f]{64}$'
        AND capacidad_huella_sha256 ~ '^[0-9a-f]{64}$'
        AND consumo_vec_huella_sha256 ~ '^[0-9a-f]{64}$'
        AND auditoria_vec_huella_sha256 ~ '^[0-9a-f]{64}$'
        AND resultado_huella_sha256 ~ '^[0-9a-f]{64}$'
        AND anterior_sha256 ~ '^[0-9a-f]{64}$'
        AND huella_sha256 ~ '^[0-9a-f]{64}$'
        AND decision_huella_sha256 <> pg_catalog.repeat('0', 64)
        AND sesion_huella_sha256 <> pg_catalog.repeat('0', 64)
        AND consulta_huella_sha256 <> pg_catalog.repeat('0', 64)
        AND capacidad_huella_sha256 <> pg_catalog.repeat('0', 64)
        AND consumo_vec_huella_sha256 <> pg_catalog.repeat('0', 64)
        AND auditoria_vec_huella_sha256 <> pg_catalog.repeat('0', 64)
        AND resultado_huella_sha256 <> pg_catalog.repeat('0', 64)
        AND huella_sha256 <> pg_catalog.repeat('0', 64)
    ),
    CHECK (
        modulo_id = 'contratacion_temporal'
        AND (
            (
                tipo_consulta = 'cuadro'
                AND accion =
                    'contratacion_temporal.cuadro.consultar'
                AND finalidad =
                    'gestion_operativa_contratacion_temporal'
                AND audiencia =
                    'vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1'
                AND recurso_tipo =
                    'cuadro_rrhh_contratacion_temporal'
                AND dominio_huella_consulta =
                    'vec.contratacion_temporal.consulta_rrhh.cuadro.v1'
                AND recurso_ref = ambito_ref
                AND expediente_ref IS NULL
                AND version_expediente IS NULL
                AND total BETWEEN 0 AND 100
                AND resultado_generico = 'entregado'
            )
            OR (
                tipo_consulta = 'detalle'
                AND accion =
                    'contratacion_temporal.expediente.consultar'
                AND finalidad =
                    'tramitacion_expediente_contratacion_temporal'
                AND audiencia =
                    'vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1'
                AND recurso_tipo =
                    'expediente_contratacion_temporal'
                AND dominio_huella_consulta =
                    'vec.contratacion_temporal.consulta_rrhh.detalle.v1'
                AND recurso_ref = expediente_ref
                AND expediente_ref IS NOT NULL
                AND (
                    (
                        total = 1
                        AND resultado_generico = 'entregado'
                        AND version_expediente IS NOT NULL
                    )
                    OR (
                        total = 0
                        AND resultado_generico = 'sin_resultado'
                        AND version_expediente IS NULL
                    )
                )
            )
        )
    ),
    CHECK (pg_catalog.octet_length(prueba_canonica) BETWEEN 256 AND 32768),
    CHECK (
        pg_catalog.encode(
            pg_catalog.sha256(
                pg_catalog.decode(anterior_sha256, 'hex')
                || prueba_canonica
            ),
            'hex'
        ) = huella_sha256
    ),
    CHECK (
        registrada_en = pg_catalog.date_trunc(
            'microseconds',
            registrada_en
        )
    )
);

CREATE FUNCTION
vec_contratacion_temporal.registrar_acceso_rrhh_interno_v1(
    p_registro jsonb
)
RETURNS jsonb
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
SET row_security = 'on'
SET timezone = 'UTC'
SET lock_timeout = '1s'
SET statement_timeout = '4s'
AS $funcion$
DECLARE
    v_clave text;
    v_huella text;
    v_anterior text;
    v_acceso_ref text;
    v_prueba bytea;
    v_secuencia numeric(20, 0);
    v_registrada_en timestamptz(6);
    v_total integer;
    v_perfil_version numeric(20, 0);
    v_version_expediente numeric(20, 0);
    v_requeridas_constantes text[] := ARRAY[
        'accion', 'actor_ref', 'ambito_ref', 'audiencia',
        'auditoria_vec_huella_sha256', 'auditoria_vec_ref',
        'capacidad_huella_sha256', 'consumo_vec_huella_sha256',
        'consulta_huella_sha256', 'correlacion_ref',
        'decision_huella_sha256', 'decision_ref',
        'dominio_huella_consulta', 'finalidad', 'modulo_id',
        'organizacion_ref',
        'perfil_id', 'recurso_ref', 'recurso_tipo',
        'resultado_generico', 'resultado_huella_sha256',
        'sesion_huella_sha256', 'sesion_id', 'tipo_consulta'
    ]::text[];
    v_claves text[] := ARRAY[
        'accion', 'actor_ref', 'ambito_ref', 'audiencia',
        'auditoria_vec_huella_sha256', 'auditoria_vec_ref',
        'capacidad_huella_sha256', 'consulta_huella_sha256',
        'consumo_vec_huella_sha256',
        'correlacion_ref',
        'decision_huella_sha256', 'decision_ref',
        'dominio_huella_consulta', 'expediente_ref', 'finalidad',
        'modulo_id', 'organizacion_ref',
        'perfil_id', 'perfil_version', 'recurso_ref', 'recurso_tipo',
        'resultado_generico', 'resultado_huella_sha256',
        'sesion_huella_sha256', 'sesion_id', 'tipo_consulta',
        'total', 'version_expediente'
    ]::text[];
BEGIN
    IF current_user <> 'vec_contratacion_temporal_propietario'
       OR pg_catalog.pg_is_in_recovery()
       OR p_registro IS NULL
       OR pg_catalog.jsonb_typeof(p_registro) <> 'object'
       OR pg_catalog.pg_column_size(p_registro) > 32768
       OR NOT vec_contratacion_temporal.o404e_claves_exactas_v1(
           p_registro,
           v_claves
       ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'registro de acceso RRHH no autorizado';
    END IF;

    FOREACH v_clave IN ARRAY v_requeridas_constantes LOOP
        IF pg_catalog.jsonb_typeof(p_registro -> v_clave) <> 'string' THEN
            RAISE EXCEPTION USING
                ERRCODE = '22023',
                MESSAGE = 'registro de acceso RRHH inválido';
        END IF;
    END LOOP;
    IF pg_catalog.jsonb_typeof(p_registro -> 'perfil_version') <> 'number'
       OR pg_catalog.jsonb_typeof(p_registro -> 'total') <> 'number'
       OR p_registro ->> 'perfil_version' !~ '^[1-9][0-9]{0,15}$'
       OR p_registro ->> 'total' !~ '^(0|[1-9][0-9]{0,2})$'
       OR COALESCE(
           pg_catalog.jsonb_typeof(p_registro -> 'expediente_ref'),
           ''
       ) NOT IN ('string', 'null')
       OR COALESCE(
           pg_catalog.jsonb_typeof(p_registro -> 'version_expediente'),
           ''
       ) NOT IN ('number', 'null')
       OR (
           pg_catalog.jsonb_typeof(
               p_registro -> 'version_expediente'
           ) = 'number'
           AND p_registro ->> 'version_expediente' !~
               '^[1-9][0-9]{0,15}$'
       ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'registro de acceso RRHH inválido';
    END IF;

    BEGIN
        v_total := (p_registro ->> 'total')::integer;
        v_perfil_version :=
            (p_registro ->> 'perfil_version')::numeric(20, 0);
        IF pg_catalog.jsonb_typeof(
            p_registro -> 'version_expediente'
        ) = 'number' THEN
            v_version_expediente :=
                (p_registro ->> 'version_expediente')::numeric(20, 0);
        END IF;
    EXCEPTION
        WHEN OTHERS THEN
            RAISE EXCEPTION USING
                ERRCODE = '22023',
                MESSAGE = 'registro de acceso RRHH inválido';
    END;

    FOREACH v_clave IN ARRAY ARRAY[
        'decision_ref', 'sesion_id', 'actor_ref', 'perfil_id',
        'organizacion_ref', 'ambito_ref', 'recurso_ref',
        'correlacion_ref', 'auditoria_vec_ref'
    ]::text[] LOOP
        IF COALESCE(p_registro ->> v_clave, '') !~
           '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' THEN
            RAISE EXCEPTION USING
                ERRCODE = '22023',
                MESSAGE = 'registro de acceso RRHH inválido';
        END IF;
    END LOOP;
    FOREACH v_clave IN ARRAY ARRAY[
        'decision_huella_sha256', 'sesion_huella_sha256',
        'consulta_huella_sha256', 'capacidad_huella_sha256',
        'consumo_vec_huella_sha256',
        'auditoria_vec_huella_sha256', 'resultado_huella_sha256'
    ]::text[] LOOP
        v_huella := p_registro ->> v_clave;
        IF COALESCE(v_huella, '') !~ '^[0-9a-f]{64}$'
           OR v_huella = pg_catalog.repeat('0', 64) THEN
            RAISE EXCEPTION USING
                ERRCODE = '22023',
                MESSAGE = 'registro de acceso RRHH inválido';
        END IF;
    END LOOP;
    IF v_perfil_version NOT BETWEEN
           1 AND 9007199254740991::numeric
       OR v_total NOT BETWEEN 0 AND 100
       OR (
           pg_catalog.jsonb_typeof(
               p_registro -> 'expediente_ref'
           ) = 'string'
           AND COALESCE(
               p_registro ->> 'expediente_ref',
               ''
           ) !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       ) OR (
           v_version_expediente IS NOT NULL
           AND v_version_expediente NOT BETWEEN
               1 AND 9007199254740991::numeric
       ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'registro de acceso RRHH inválido';
    END IF;

    IF p_registro ->> 'modulo_id' <> 'contratacion_temporal'
       OR (
           p_registro ->> 'tipo_consulta' = 'cuadro'
           AND (
               p_registro ->> 'accion' <>
                   'contratacion_temporal.cuadro.consultar'
               OR p_registro ->> 'finalidad' <>
                   'gestion_operativa_contratacion_temporal'
               OR p_registro ->> 'audiencia' <>
                   'vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1'
               OR p_registro ->> 'recurso_tipo' <>
                   'cuadro_rrhh_contratacion_temporal'
               OR p_registro ->> 'recurso_ref' <>
                   p_registro ->> 'ambito_ref'
               OR p_registro ->> 'dominio_huella_consulta' <>
                   'vec.contratacion_temporal.consulta_rrhh.cuadro.v1'
               OR pg_catalog.jsonb_typeof(
                   p_registro -> 'expediente_ref'
               ) <> 'null'
               OR pg_catalog.jsonb_typeof(
                   p_registro -> 'version_expediente'
               ) <> 'null'
               OR p_registro ->> 'resultado_generico' <> 'entregado'
           )
       ) OR (
           p_registro ->> 'tipo_consulta' = 'detalle'
           AND (
               p_registro ->> 'accion' <>
                   'contratacion_temporal.expediente.consultar'
               OR p_registro ->> 'finalidad' <>
                   'tramitacion_expediente_contratacion_temporal'
               OR p_registro ->> 'audiencia' <>
                   'vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1'
               OR p_registro ->> 'recurso_tipo' <>
                   'expediente_contratacion_temporal'
               OR p_registro ->> 'dominio_huella_consulta' <>
                   'vec.contratacion_temporal.consulta_rrhh.detalle.v1'
               OR pg_catalog.jsonb_typeof(
                   p_registro -> 'expediente_ref'
               ) <> 'string'
               OR p_registro ->> 'recurso_ref' IS DISTINCT FROM
                   p_registro ->> 'expediente_ref'
               OR (
                   v_total = 1
                   AND (
                       p_registro ->> 'resultado_generico' <> 'entregado'
                       OR v_version_expediente IS NULL
                   )
               )
               OR (
                   v_total = 0
                   AND (
                       p_registro ->> 'resultado_generico' <>
                           'sin_resultado'
                       OR v_version_expediente IS NOT NULL
                   )
               )
               OR v_total NOT IN (0, 1)
           )
       ) OR p_registro ->> 'tipo_consulta' NOT IN ('cuadro', 'detalle') THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'registro de acceso RRHH inválido';
    END IF;

    PERFORM 1
      FROM vec_contratacion_temporal.control_migracion_consultas_rrhh
     WHERE control
       AND version_esquema = 1
     FOR SHARE;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'barrera de consultas RRHH no disponible';
    END IF;
    SELECT ultima_secuencia + 1, cabeza_sha256
      INTO STRICT v_secuencia, v_anterior
      FROM vec_contratacion_temporal.control_cadena_accesos_rrhh
     WHERE control
     FOR UPDATE;
    IF v_secuencia > 9007199254740991::numeric THEN
        RAISE EXCEPTION USING
            ERRCODE = '54000',
            MESSAGE = 'capacidad del registro de accesos RRHH agotada';
    END IF;

    v_registrada_en := pg_catalog.date_trunc(
        'microseconds',
        pg_catalog.clock_timestamp()
    );
    v_acceso_ref := 'acceso:rrhh:' || pg_catalog.substr(
        pg_catalog.encode(
            pg_catalog.sha256(
                pg_catalog.convert_to(
                    'acceso:rrhh:'
                    || (p_registro ->> 'consumo_vec_huella_sha256'),
                    'UTF8'
                )
            ),
            'hex'
        ),
        1,
        32
    );
    v_prueba :=
        pg_catalog.convert_to(
            'VEC-CT-ACCESO-RRHH-O4-05-V1' || chr(10),
            'UTF8'
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(v_acceso_ref)
        || vec_contratacion_temporal.encuadrar_texto_v1(v_secuencia::text);
    FOREACH v_clave IN ARRAY v_claves LOOP
        v_prueba := v_prueba
            || vec_contratacion_temporal.encuadrar_texto_v1(
                CASE
                    WHEN pg_catalog.jsonb_typeof(
                        p_registro -> v_clave
                    ) = 'null' THEN ''
                    ELSE p_registro ->> v_clave
                END
            );
    END LOOP;
    v_prueba := v_prueba
        || vec_contratacion_temporal.encuadrar_texto_v1(
            vec_contratacion_temporal.instante_utc_v1(v_registrada_en)
        );
    v_huella := pg_catalog.encode(
        pg_catalog.sha256(
            pg_catalog.decode(v_anterior, 'hex') || v_prueba
        ),
        'hex'
    );

    INSERT INTO vec_contratacion_temporal.registro_acceso_rrhh (
        acceso_ref, secuencia, tipo_consulta, decision_ref,
        decision_huella_sha256, sesion_id, sesion_huella_sha256,
        actor_ref, perfil_id, perfil_version, organizacion_ref,
        ambito_ref, modulo_id, accion, finalidad, audiencia,
        recurso_tipo, recurso_ref, dominio_huella_consulta,
        consulta_huella_sha256, correlacion_ref, expediente_ref,
        version_expediente, capacidad_huella_sha256,
        consumo_vec_huella_sha256,
        auditoria_vec_ref,
        auditoria_vec_huella_sha256, resultado_huella_sha256,
        total, resultado_generico, prueba_canonica,
        anterior_sha256, huella_sha256, registrada_en
    ) VALUES (
        v_acceso_ref, v_secuencia, p_registro ->> 'tipo_consulta',
        p_registro ->> 'decision_ref',
        p_registro ->> 'decision_huella_sha256',
        p_registro ->> 'sesion_id',
        p_registro ->> 'sesion_huella_sha256',
        p_registro ->> 'actor_ref', p_registro ->> 'perfil_id',
        v_perfil_version, p_registro ->> 'organizacion_ref',
        p_registro ->> 'ambito_ref', p_registro ->> 'modulo_id',
        p_registro ->> 'accion', p_registro ->> 'finalidad',
        p_registro ->> 'audiencia', p_registro ->> 'recurso_tipo',
        p_registro ->> 'recurso_ref',
        p_registro ->> 'dominio_huella_consulta',
        p_registro ->> 'consulta_huella_sha256',
        p_registro ->> 'correlacion_ref',
        CASE
            WHEN pg_catalog.jsonb_typeof(
                p_registro -> 'expediente_ref'
            ) = 'null' THEN NULL
            ELSE p_registro ->> 'expediente_ref'
        END,
        v_version_expediente,
        p_registro ->> 'capacidad_huella_sha256',
        p_registro ->> 'consumo_vec_huella_sha256',
        p_registro ->> 'auditoria_vec_ref',
        p_registro ->> 'auditoria_vec_huella_sha256',
        p_registro ->> 'resultado_huella_sha256',
        v_total, p_registro ->> 'resultado_generico',
        v_prueba, v_anterior, v_huella, v_registrada_en
    );
    UPDATE vec_contratacion_temporal.control_cadena_accesos_rrhh
       SET ultima_secuencia = v_secuencia,
           cabeza_sha256 = v_huella,
           actualizada_en = v_registrada_en
     WHERE control;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'control del registro de accesos RRHH ausente';
    END IF;

    RETURN pg_catalog.jsonb_build_object(
        'esquema',
        'vec.contratacion-temporal.recibo-acceso-rrhh.o4-05.v1',
        'acceso_ref',
        v_acceso_ref,
        'secuencia',
        v_secuencia,
        'anterior_sha256',
        v_anterior,
        'huella_sha256',
        v_huella,
        'registrada_en',
        vec_contratacion_temporal.instante_utc_v1(v_registrada_en)
    );
END
$funcion$;

CREATE TRIGGER registro_acceso_rrhh_inmutable
BEFORE UPDATE OR DELETE
ON vec_contratacion_temporal.registro_acceso_rrhh
FOR EACH ROW
EXECUTE FUNCTION
    vec_contratacion_temporal.rechazar_mutacion_historia_v1();

CREATE TRIGGER registro_acceso_rrhh_no_truncar
BEFORE TRUNCATE
ON vec_contratacion_temporal.registro_acceso_rrhh
FOR EACH STATEMENT
EXECUTE FUNCTION
    vec_contratacion_temporal.rechazar_mutacion_historia_v1();

DO $rls$
DECLARE
    v_tabla text;
BEGIN
    FOREACH v_tabla IN ARRAY ARRAY[
        'control_migracion_consultas_rrhh',
        'control_cadena_accesos_rrhh',
        'registro_acceso_rrhh'
    ]::text[] LOOP
        EXECUTE pg_catalog.format(
            'ALTER TABLE vec_contratacion_temporal.%I '
            || 'ENABLE ROW LEVEL SECURITY',
            v_tabla
        );
        EXECUTE pg_catalog.format(
            'ALTER TABLE vec_contratacion_temporal.%I '
            || 'FORCE ROW LEVEL SECURITY',
            v_tabla
        );
        EXECUTE pg_catalog.format(
            'CREATE POLICY propietario_total '
            || 'ON vec_contratacion_temporal.%I '
            || 'TO vec_contratacion_temporal_propietario '
            || 'USING (true) WITH CHECK (true)',
            v_tabla
        );
    END LOOP;
END
$rls$;

REVOKE ALL ON TABLE
    vec_contratacion_temporal.control_migracion_consultas_rrhh,
    vec_contratacion_temporal.control_cadena_accesos_rrhh,
    vec_contratacion_temporal.registro_acceso_rrhh
FROM PUBLIC,
    vec_contratacion_temporal_migrador,
    vec_contratacion_temporal_ejecutor,
    vec_contratacion_temporal_gobernador,
    vec_contratacion_temporal_confirmador_cobertura,
    vec_contratacion_temporal_lector_resultado_cobertura,
    vec_contratacion_temporal_consultor_rrhh;

REVOKE ALL ON FUNCTION
    vec_contratacion_temporal.registrar_acceso_rrhh_interno_v1(jsonb)
FROM PUBLIC,
    vec_contratacion_temporal_migrador,
    vec_contratacion_temporal_ejecutor,
    vec_contratacion_temporal_gobernador,
    vec_contratacion_temporal_confirmador_cobertura,
    vec_contratacion_temporal_lector_resultado_cobertura,
    vec_contratacion_temporal_consultor_rrhh;

UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4
   SET version_esquema = 16,
       actualizada_en = pg_catalog.date_trunc(
           'microseconds',
           pg_catalog.clock_timestamp()
       )
 WHERE control
   AND version_esquema = 15;

COMMIT;
