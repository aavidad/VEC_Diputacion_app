-- C2-D2-C: registro v2 ligado a identidad viva y acceso as-of eficiente.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog; SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s'; SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_contratacion_temporal:o4_05:consultas_rrhh:migraciones', 0
));
SELECT control
  FROM vec_contratacion_temporal.control_migracion_cobertura_o4
 WHERE control AND version_esquema = 18
 FOR UPDATE;
SELECT control
  FROM vec_contratacion_temporal.control_migracion_consultas_rrhh
 WHERE control AND version_esquema = 2
 FOR UPDATE;
SELECT control
  FROM vec_contratacion_temporal.control_cadena_accesos_rrhh
 WHERE control
 FOR UPDATE;

DO $prevalidacion$
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.control_migracion_cobertura_o4
         WHERE control AND version_esquema = 18
    ) OR NOT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.control_migracion_consultas_rrhh
         WHERE control AND version_esquema = 2
    ) OR (
        SELECT pg_catalog.count(*)
          FROM vec_contratacion_temporal.control_cadena_accesos_rrhh
         WHERE control
           AND ultima_secuencia BETWEEN
               0 AND 9007199254740991::numeric
           AND cabeza_sha256 ~ '^[0-9a-f]{64}$'
    ) <> 1
    OR pg_catalog.to_regclass(
        'vec_contratacion_temporal.registro_acceso_rrhh'
    ) IS NULL
    OR pg_catalog.to_regclass(
        'vec_contratacion_temporal.alcance_acceso_rrhh'
    ) IS NULL
    OR pg_catalog.to_regclass(
        'vec_contratacion_temporal.publicacion_version_rrhh'
    ) IS NULL
    OR pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.registrar_acceso_rrhh_interno_v1(jsonb)'
    ) IS NULL
    OR pg_catalog.to_regprocedure(
        'vec_identidad_sesiones_v1.'
        || 'revalidar_consulta_rrhh_v1(text,text)'
    ) IS NULL
    OR NOT pg_catalog.has_function_privilege(
        'vec_contratacion_temporal_propietario',
        'vec_identidad_sesiones_v1.'
        || 'revalidar_consulta_rrhh_v1(text,text)',
        'EXECUTE'
    )
    OR NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_constraint
         WHERE conrelid =
               'vec_contratacion_temporal.registro_acceso_rrhh'::regclass
           AND conname =
               'registro_acceso_rrhh_cursor_identidad_unica'
           AND contype = 'u'
    )
    OR pg_catalog.to_regclass(
        'vec_contratacion_temporal.control_registrador_acceso_rrhh_v2'
    ) IS NOT NULL
    OR pg_catalog.to_regclass(
        'vec_contratacion_temporal.vinculo_identidad_acceso_rrhh_v2'
    ) IS NOT NULL
    OR pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.registrar_acceso_rrhh_interno_v2(jsonb)'
    ) IS NOT NULL
    OR pg_catalog.to_regclass(
        'vec_contratacion_temporal.'
        || 'publicacion_rrhh_organizacion_expediente_corte_desc_idx'
    ) IS NOT NULL THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'estado incompatible para registrador RRHH v2';
    END IF;
END
$prevalidacion$;

CREATE TABLE
vec_contratacion_temporal.control_registrador_acceso_rrhh_v2 (
    control boolean PRIMARY KEY DEFAULT true CHECK (control),
    version_esquema integer NOT NULL CHECK (version_esquema = 1),
    secuencia_base numeric(20, 0) NOT NULL CHECK (
        secuencia_base BETWEEN 0 AND 9007199254740991::numeric
    ),
    cabeza_base_sha256 text NOT NULL CHECK (
        cabeza_base_sha256 ~ '^[0-9a-f]{64}$'
    ),
    creada_en timestamptz(6) NOT NULL CHECK (
        creada_en = pg_catalog.date_trunc('microseconds', creada_en)
    ),
    prueba_canonica bytea NOT NULL,
    prueba_huella_sha256 text NOT NULL UNIQUE,
    CHECK (
        prueba_canonica =
            pg_catalog.convert_to(
                'VEC-CT-CONTROL-REGISTRADOR-ACCESO-RRHH-V2'
                || pg_catalog.chr(10), 'UTF8'
            )
            || vec_contratacion_temporal.encuadrar_texto_v1(
                version_esquema::text
            )
            || vec_contratacion_temporal.encuadrar_texto_v1(
                secuencia_base::text
            )
            || vec_contratacion_temporal.encuadrar_texto_v1(
                cabeza_base_sha256
            )
            || vec_contratacion_temporal.encuadrar_texto_v1(
                vec_contratacion_temporal.instante_utc_v1(creada_en)
            )
        AND prueba_huella_sha256 ~ '^[0-9a-f]{64}$'
        AND prueba_huella_sha256 <> pg_catalog.repeat('0', 64)
        AND pg_catalog.encode(
            pg_catalog.sha256(prueba_canonica), 'hex'
        ) = prueba_huella_sha256
    )
);

INSERT INTO
vec_contratacion_temporal.control_registrador_acceso_rrhh_v2 (
    control, version_esquema, secuencia_base, cabeza_base_sha256,
    creada_en, prueba_canonica, prueba_huella_sha256
)
SELECT true, 1, base.ultima_secuencia, base.cabeza_sha256,
       reloj.instante, prueba.canon,
       pg_catalog.encode(pg_catalog.sha256(prueba.canon), 'hex')
  FROM vec_contratacion_temporal.control_cadena_accesos_rrhh base
 CROSS JOIN LATERAL (
     SELECT pg_catalog.date_trunc(
         'microseconds', pg_catalog.clock_timestamp()
     ) AS instante
 ) reloj
 CROSS JOIN LATERAL (
     SELECT pg_catalog.convert_to(
                'VEC-CT-CONTROL-REGISTRADOR-ACCESO-RRHH-V2'
                || pg_catalog.chr(10), 'UTF8'
            )
            || vec_contratacion_temporal.encuadrar_texto_v1('1')
            || vec_contratacion_temporal.encuadrar_texto_v1(
                base.ultima_secuencia::text
            )
            || vec_contratacion_temporal.encuadrar_texto_v1(
                base.cabeza_sha256
            )
            || vec_contratacion_temporal.encuadrar_texto_v1(
                vec_contratacion_temporal.instante_utc_v1(reloj.instante)
            ) AS canon
 ) prueba
 WHERE base.control;

CREATE TABLE
vec_contratacion_temporal.vinculo_identidad_acceso_rrhh_v2 (
    acceso_ref text PRIMARY KEY,
    login_tecnico text NOT NULL,
    autenticacion_ref text NOT NULL,
    autenticacion_huella_sha256 text NOT NULL,
    sesion_ref text NOT NULL,
    control_sesion_ref text NOT NULL,
    control_sesion_revision numeric(20, 0) NOT NULL,
    control_sesion_huella_sha256 text NOT NULL,
    actor_ref text NOT NULL,
    perfil_ref text NOT NULL,
    perfil_version numeric(20, 0) NOT NULL,
    organizacion_ref text NOT NULL,
    clase_ambito text NOT NULL,
    ambito_ref text NOT NULL,
    sesion_huella_sha256 text NOT NULL,
    acceso_registrado_en timestamptz(6) NOT NULL,
    prueba_canonica bytea NOT NULL,
    prueba_huella_sha256 text NOT NULL UNIQUE,
    FOREIGN KEY (
        acceso_ref, organizacion_ref, ambito_ref, actor_ref,
        perfil_ref, perfil_version, sesion_ref,
        sesion_huella_sha256, acceso_registrado_en
    ) REFERENCES vec_contratacion_temporal.registro_acceso_rrhh(
        acceso_ref, organizacion_ref, ambito_ref, actor_ref,
        perfil_id, perfil_version, sesion_id,
        sesion_huella_sha256, registrada_en
    ) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CHECK (
        login_tecnico ~ '^[A-Za-z_][A-Za-z0-9_-]{0,62}$'
        AND autenticacion_ref ~
            '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND sesion_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND control_sesion_ref ~
            '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND actor_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND perfil_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND organizacion_ref ~
            '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND ambito_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND clase_ambito IN ('organizacion', 'centro', 'unidad_gestion')
        AND (
            clase_ambito <> 'organizacion'
            OR ambito_ref = organizacion_ref
        )
    ),
    CHECK (
        control_sesion_revision BETWEEN
            1 AND 18446744073709551615::numeric
        AND perfil_version BETWEEN
            1 AND 9007199254740991::numeric
    ),
    CHECK (
        autenticacion_huella_sha256 ~ '^[0-9a-f]{64}$'
        AND control_sesion_huella_sha256 ~ '^[0-9a-f]{64}$'
        AND sesion_huella_sha256 ~ '^[0-9a-f]{64}$'
        AND autenticacion_huella_sha256 <>
            pg_catalog.repeat('0', 64)
        AND control_sesion_huella_sha256 <>
            pg_catalog.repeat('0', 64)
        AND sesion_huella_sha256 <> pg_catalog.repeat('0', 64)
    ),
    CHECK (
        acceso_registrado_en =
            pg_catalog.date_trunc('microseconds', acceso_registrado_en)
    ),
    CHECK (
        prueba_canonica =
            pg_catalog.convert_to(
                'VEC-CT-VINCULO-IDENTIDAD-ACCESO-RRHH-V2'
                || pg_catalog.chr(10), 'UTF8'
            )
            || vec_contratacion_temporal.encuadrar_texto_v1(acceso_ref)
            || vec_contratacion_temporal.encuadrar_texto_v1(login_tecnico)
            || vec_contratacion_temporal.encuadrar_texto_v1(
                autenticacion_ref
            )
            || vec_contratacion_temporal.encuadrar_texto_v1(
                autenticacion_huella_sha256
            )
            || vec_contratacion_temporal.encuadrar_texto_v1(sesion_ref)
            || vec_contratacion_temporal.encuadrar_texto_v1(
                control_sesion_ref
            )
            || vec_contratacion_temporal.encuadrar_texto_v1(
                control_sesion_revision::text
            )
            || vec_contratacion_temporal.encuadrar_texto_v1(
                control_sesion_huella_sha256
            )
            || vec_contratacion_temporal.encuadrar_texto_v1(actor_ref)
            || vec_contratacion_temporal.encuadrar_texto_v1(perfil_ref)
            || vec_contratacion_temporal.encuadrar_texto_v1(
                perfil_version::text
            )
            || vec_contratacion_temporal.encuadrar_texto_v1(
                organizacion_ref
            )
            || vec_contratacion_temporal.encuadrar_texto_v1(clase_ambito)
            || vec_contratacion_temporal.encuadrar_texto_v1(ambito_ref)
            || vec_contratacion_temporal.encuadrar_texto_v1(
                sesion_huella_sha256
            )
            || vec_contratacion_temporal.encuadrar_texto_v1(
                vec_contratacion_temporal.instante_utc_v1(
                    acceso_registrado_en
                )
            )
        AND prueba_huella_sha256 ~ '^[0-9a-f]{64}$'
        AND prueba_huella_sha256 <> pg_catalog.repeat('0', 64)
        AND pg_catalog.encode(
            pg_catalog.sha256(prueba_canonica), 'hex'
        ) = prueba_huella_sha256
    )
);

COMMENT ON COLUMN
    vec_contratacion_temporal.vinculo_identidad_acceso_rrhh_v2.login_tecnico
IS 'LOGIN técnico crudo: no identifica a la persona; se conserva para atribución forense exacta. Una huella sería enumerable contra pg_roles y perdería operatividad. La tabla usa RLS forzada y no se publica al runtime.';

CREATE INDEX
    publicacion_rrhh_organizacion_expediente_corte_desc_idx
ON vec_contratacion_temporal.publicacion_version_rrhh (
    (organizacion_ref COLLATE "C"),
    (expediente_ref COLLATE "C"),
    corte_global DESC
);

CREATE FUNCTION
vec_contratacion_temporal.registrar_acceso_rrhh_interno_v2(
    p_peticion jsonb
)
RETURNS jsonb
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
PARALLEL UNSAFE
SET search_path = pg_catalog
SET row_security = 'on'
SET timezone = 'UTC'
SET lock_timeout = '1s'
SET statement_timeout = '4s'
AS $funcion$
DECLARE
    r jsonb;
    a jsonb;
    i jsonb;
    identidad record;
    clave text;
    huella text;
    anterior text;
    acceso_ref text;
    prueba bytea;
    prueba_alcance bytea;
    prueba_vinculo bytea;
    huella_alcance text;
    huella_vinculo text;
    secuencia numeric(20, 0);
    registrada_en timestamptz(6);
    total integer;
    perfil_version numeric(20, 0);
    version_expediente numeric(20, 0);
    control_revision numeric(20, 0);
    claves_registro text[] := ARRAY[
        'accion', 'actor_ref', 'ambito_ref', 'audiencia',
        'auditoria_vec_huella_sha256', 'auditoria_vec_ref',
        'capacidad_huella_sha256', 'consulta_huella_sha256',
        'consumo_vec_huella_sha256', 'correlacion_ref',
        'decision_huella_sha256', 'decision_ref',
        'dominio_huella_consulta', 'expediente_ref', 'finalidad',
        'modulo_id', 'organizacion_ref', 'perfil_id',
        'perfil_version', 'recurso_ref', 'recurso_tipo',
        'resultado_generico', 'resultado_huella_sha256',
        'sesion_huella_sha256', 'sesion_id', 'tipo_consulta',
        'total', 'version_expediente'
    ]::text[];
BEGIN
    IF current_user <> 'vec_contratacion_temporal_propietario'
       OR pg_catalog.pg_is_in_recovery()
       OR p_peticion IS NULL
       OR pg_catalog.jsonb_typeof(p_peticion) <> 'object'
       OR pg_catalog.pg_column_size(p_peticion) > 49152
       OR NOT vec_contratacion_temporal.o404e_claves_exactas_v1(
           p_peticion, ARRAY['alcance', 'identidad', 'registro']::text[]
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'registro de acceso RRHH v2 no autorizado';
    END IF;
    r := p_peticion -> 'registro';
    a := p_peticion -> 'alcance';
    i := p_peticion -> 'identidad';
    IF pg_catalog.jsonb_typeof(r) <> 'object'
       OR pg_catalog.jsonb_typeof(a) <> 'object'
       OR pg_catalog.jsonb_typeof(i) <> 'object'
       OR NOT vec_contratacion_temporal.o404e_claves_exactas_v1(
           r, claves_registro
       )
       OR NOT vec_contratacion_temporal.o404e_claves_exactas_v1(
           a, ARRAY['clase_ambito', 'familia_ref']::text[]
       )
       OR NOT vec_contratacion_temporal.o404e_claves_exactas_v1(
           i, ARRAY[
               'actor_ref', 'autenticacion_huella_sha256',
               'autenticacion_ref', 'control_sesion_huella_sha256',
               'control_sesion_ref', 'control_sesion_revision',
               'organizacion_ref', 'perfil_ref', 'perfil_version',
               'sesion_ref'
           ]::text[]
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'registro de acceso RRHH v2 inválido';
    END IF;

    FOREACH clave IN ARRAY ARRAY[
        'decision_ref', 'sesion_id', 'actor_ref', 'perfil_id',
        'organizacion_ref', 'ambito_ref', 'recurso_ref',
        'correlacion_ref', 'auditoria_vec_ref'
    ]::text[] LOOP
        IF pg_catalog.jsonb_typeof(r -> clave) <> 'string'
           OR r ->> clave !~
              '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'registro de acceso RRHH v2 inválido';
        END IF;
    END LOOP;
    FOREACH clave IN ARRAY ARRAY[
        'decision_huella_sha256', 'sesion_huella_sha256',
        'consulta_huella_sha256', 'capacidad_huella_sha256',
        'consumo_vec_huella_sha256', 'auditoria_vec_huella_sha256',
        'resultado_huella_sha256'
    ]::text[] LOOP
        huella := r ->> clave;
        IF pg_catalog.jsonb_typeof(r -> clave) <> 'string'
           OR huella !~ '^[0-9a-f]{64}$'
           OR huella = pg_catalog.repeat('0', 64) THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'registro de acceso RRHH v2 inválido';
        END IF;
    END LOOP;
    IF r ->> 'perfil_version' !~ '^[1-9][0-9]{0,15}$'
       OR r ->> 'total' !~ '^(0|[1-9][0-9]{0,2})$'
       OR COALESCE(
           pg_catalog.jsonb_typeof(r -> 'expediente_ref'), ''
       ) NOT IN ('string', 'null')
       OR COALESCE(
           pg_catalog.jsonb_typeof(r -> 'version_expediente'), ''
       ) NOT IN ('number', 'null')
       OR (
           pg_catalog.jsonb_typeof(r -> 'version_expediente') = 'number'
           AND r ->> 'version_expediente' !~ '^[1-9][0-9]{0,15}$'
       )
       OR i ->> 'control_sesion_revision' !~ '^[1-9][0-9]{0,19}$'
       OR i ->> 'perfil_version' !~ '^[1-9][0-9]{0,15}$' THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'registro de acceso RRHH v2 inválido';
    END IF;
    BEGIN
        total := (r ->> 'total')::integer;
        perfil_version := (r ->> 'perfil_version')::numeric(20, 0);
        control_revision :=
            (i ->> 'control_sesion_revision')::numeric(20, 0);
        IF pg_catalog.jsonb_typeof(
            r -> 'version_expediente'
        ) = 'number' THEN
            version_expediente :=
                (r ->> 'version_expediente')::numeric(20, 0);
        END IF;
    EXCEPTION WHEN OTHERS THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'registro de acceso RRHH v2 inválido';
    END;
    IF perfil_version NOT BETWEEN 1 AND 9007199254740991::numeric
       OR total NOT BETWEEN 0 AND 100
       OR control_revision NOT BETWEEN
           1 AND 18446744073709551615::numeric
       OR (version_expediente IS NOT NULL AND version_expediente
           NOT BETWEEN 1 AND 9007199254740991::numeric)
       OR a ->> 'clase_ambito'
          NOT IN ('organizacion', 'centro', 'unidad_gestion')
       OR COALESCE(
           pg_catalog.jsonb_typeof(a -> 'familia_ref'), ''
       ) NOT IN ('string', 'null')
       OR (
           pg_catalog.jsonb_typeof(a -> 'familia_ref') = 'string'
           AND a ->> 'familia_ref' !~
               '^familia:cursor:rrhh:[0-9a-f]{32}$'
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'registro de acceso RRHH v2 inválido';
    END IF;

    IF r ->> 'modulo_id' <> 'contratacion_temporal'
       OR (
           r ->> 'tipo_consulta' = 'cuadro'
           AND (
               r ->> 'accion' <>
                   'contratacion_temporal.cuadro.consultar'
               OR r ->> 'finalidad' <>
                   'gestion_operativa_contratacion_temporal'
               OR r ->> 'audiencia' <>
                   'vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1'
               OR r ->> 'recurso_tipo' <>
                   'cuadro_rrhh_contratacion_temporal'
               OR r ->> 'recurso_ref' <> r ->> 'ambito_ref'
               OR r ->> 'dominio_huella_consulta' <>
                   'vec.contratacion_temporal.consulta_rrhh.cuadro.v1'
               OR pg_catalog.jsonb_typeof(r -> 'expediente_ref') <> 'null'
               OR pg_catalog.jsonb_typeof(
                   r -> 'version_expediente'
               ) <> 'null'
               OR r ->> 'resultado_generico' <> 'entregado'
           )
       ) OR (
           r ->> 'tipo_consulta' = 'detalle'
           AND (
               r ->> 'accion' <>
                   'contratacion_temporal.expediente.consultar'
               OR r ->> 'finalidad' <>
                   'tramitacion_expediente_contratacion_temporal'
               OR r ->> 'audiencia' <>
                   'vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1'
               OR r ->> 'recurso_tipo' <>
                   'expediente_contratacion_temporal'
               OR r ->> 'dominio_huella_consulta' <>
                   'vec.contratacion_temporal.consulta_rrhh.detalle.v1'
               OR r ->> 'recurso_ref' IS DISTINCT FROM
                  r ->> 'expediente_ref'
               OR (
                   total = 1 AND (
                       r ->> 'resultado_generico' <> 'entregado'
                       OR version_expediente IS NULL
                   )
               )
               OR (
                   total = 0 AND (
                       r ->> 'resultado_generico' <> 'sin_resultado'
                       OR version_expediente IS NOT NULL
                   )
               )
               OR total NOT IN (0, 1)
               OR pg_catalog.jsonb_typeof(a -> 'familia_ref') <> 'null'
           )
       ) OR r ->> 'tipo_consulta' NOT IN ('cuadro', 'detalle') THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'registro de acceso RRHH v2 inválido';
    END IF;

    IF i ->> 'sesion_ref' IS DISTINCT FROM r ->> 'sesion_id'
       OR i ->> 'actor_ref' IS DISTINCT FROM r ->> 'actor_ref'
       OR i ->> 'perfil_ref' IS DISTINCT FROM r ->> 'perfil_id'
       OR i ->> 'perfil_version' IS DISTINCT FROM
          r ->> 'perfil_version'
       OR i ->> 'organizacion_ref' IS DISTINCT FROM
          r ->> 'organizacion_ref'
       OR (
           a ->> 'clase_ambito' = 'organizacion'
           AND r ->> 'ambito_ref' <> r ->> 'organizacion_ref'
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'identidad de acceso RRHH v2 no coincide';
    END IF;
    BEGIN
        SELECT *
          INTO STRICT identidad
          FROM vec_identidad_sesiones_v1.revalidar_consulta_rrhh_v1(
              i ->> 'autenticacion_ref', i ->> 'sesion_ref'
          );
    EXCEPTION WHEN no_data_found OR too_many_rows THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'identidad de acceso RRHH v2 no disponible';
    END;
    IF identidad.autenticacion_ref IS DISTINCT FROM
           i ->> 'autenticacion_ref'
       OR identidad.autenticacion_huella_sha256 IS DISTINCT FROM
          i ->> 'autenticacion_huella_sha256'
       OR identidad.sesion_ref IS DISTINCT FROM i ->> 'sesion_ref'
       OR identidad.control_sesion_ref IS DISTINCT FROM
          i ->> 'control_sesion_ref'
       OR identidad.control_sesion_revision IS DISTINCT FROM
          i ->> 'control_sesion_revision'
       OR identidad.control_sesion_huella_sha256 IS DISTINCT FROM
          i ->> 'control_sesion_huella_sha256' THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'identidad de acceso RRHH v2 no coincide';
    END IF;
    PERFORM 1
      FROM vec_contratacion_temporal.control_migracion_consultas_rrhh
     WHERE control AND version_esquema = 3
     FOR SHARE;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'barrera de consultas RRHH v2 no disponible';
    END IF;
    SELECT ultima_secuencia + 1, cabeza_sha256
      INTO STRICT secuencia, anterior
      FROM vec_contratacion_temporal.control_cadena_accesos_rrhh
     WHERE control
     FOR UPDATE;
    IF secuencia > 9007199254740991::numeric THEN
        RAISE EXCEPTION USING ERRCODE = '54000',
            MESSAGE = 'capacidad del registro de accesos RRHH agotada';
    END IF;
    registrada_en := pg_catalog.date_trunc(
        'microseconds', pg_catalog.clock_timestamp()
    );
    acceso_ref := 'acceso:rrhh:' || pg_catalog.substr(
        pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
            'acceso:rrhh:' || (r ->> 'consumo_vec_huella_sha256'),
            'UTF8'
        )), 'hex'), 1, 32
    );
    prueba := pg_catalog.convert_to(
        'VEC-CT-ACCESO-RRHH-O4-05-V2' || pg_catalog.chr(10), 'UTF8'
    )
        || vec_contratacion_temporal.encuadrar_texto_v1(acceso_ref)
        || vec_contratacion_temporal.encuadrar_texto_v1(secuencia::text);
    FOREACH clave IN ARRAY claves_registro LOOP
        prueba := prueba || vec_contratacion_temporal.encuadrar_texto_v1(
            CASE WHEN pg_catalog.jsonb_typeof(r -> clave) = 'null'
                 THEN '' ELSE r ->> clave END
        );
    END LOOP;
    prueba := prueba || vec_contratacion_temporal.encuadrar_texto_v1(
        vec_contratacion_temporal.instante_utc_v1(registrada_en)
    );
    huella := pg_catalog.encode(pg_catalog.sha256(
        pg_catalog.decode(anterior, 'hex') || prueba
    ), 'hex');

    INSERT INTO vec_contratacion_temporal.registro_acceso_rrhh (
        acceso_ref, secuencia, tipo_consulta, decision_ref,
        decision_huella_sha256, sesion_id, sesion_huella_sha256,
        actor_ref, perfil_id, perfil_version, organizacion_ref,
        ambito_ref, modulo_id, accion, finalidad, audiencia,
        recurso_tipo, recurso_ref, dominio_huella_consulta,
        consulta_huella_sha256, correlacion_ref, expediente_ref,
        version_expediente, capacidad_huella_sha256,
        consumo_vec_huella_sha256, auditoria_vec_ref,
        auditoria_vec_huella_sha256, resultado_huella_sha256,
        total, resultado_generico, prueba_canonica,
        anterior_sha256, huella_sha256, registrada_en
    ) VALUES (
        acceso_ref, secuencia, r ->> 'tipo_consulta',
        r ->> 'decision_ref', r ->> 'decision_huella_sha256',
        r ->> 'sesion_id', r ->> 'sesion_huella_sha256',
        r ->> 'actor_ref', r ->> 'perfil_id', perfil_version,
        r ->> 'organizacion_ref', r ->> 'ambito_ref',
        r ->> 'modulo_id', r ->> 'accion', r ->> 'finalidad',
        r ->> 'audiencia', r ->> 'recurso_tipo',
        r ->> 'recurso_ref', r ->> 'dominio_huella_consulta',
        r ->> 'consulta_huella_sha256', r ->> 'correlacion_ref',
        CASE WHEN pg_catalog.jsonb_typeof(r -> 'expediente_ref') = 'null'
             THEN NULL ELSE r ->> 'expediente_ref' END,
        version_expediente, r ->> 'capacidad_huella_sha256',
        r ->> 'consumo_vec_huella_sha256',
        r ->> 'auditoria_vec_ref',
        r ->> 'auditoria_vec_huella_sha256',
        r ->> 'resultado_huella_sha256', total,
        r ->> 'resultado_generico', prueba, anterior, huella,
        registrada_en
    );

    prueba_vinculo := pg_catalog.convert_to(
        'VEC-CT-VINCULO-IDENTIDAD-ACCESO-RRHH-V2'
        || pg_catalog.chr(10), 'UTF8'
    );
    FOREACH clave IN ARRAY ARRAY[
        acceso_ref, identidad.login_tecnico,
        identidad.autenticacion_ref,
        identidad.autenticacion_huella_sha256,
        identidad.sesion_ref, identidad.control_sesion_ref,
        identidad.control_sesion_revision,
        identidad.control_sesion_huella_sha256,
        r ->> 'actor_ref', r ->> 'perfil_id',
        perfil_version::text, r ->> 'organizacion_ref',
        a ->> 'clase_ambito', r ->> 'ambito_ref',
        r ->> 'sesion_huella_sha256',
        vec_contratacion_temporal.instante_utc_v1(registrada_en)
    ]::text[] LOOP
        prueba_vinculo := prueba_vinculo
            || vec_contratacion_temporal.encuadrar_texto_v1(clave);
    END LOOP;
    huella_vinculo := pg_catalog.encode(
        pg_catalog.sha256(prueba_vinculo), 'hex'
    );
    INSERT INTO
    vec_contratacion_temporal.vinculo_identidad_acceso_rrhh_v2 (
        acceso_ref, login_tecnico, autenticacion_ref,
        autenticacion_huella_sha256, sesion_ref, control_sesion_ref,
        control_sesion_revision, control_sesion_huella_sha256,
        actor_ref, perfil_ref, perfil_version, organizacion_ref,
        clase_ambito, ambito_ref, sesion_huella_sha256,
        acceso_registrado_en, prueba_canonica, prueba_huella_sha256
    ) VALUES (
        acceso_ref, identidad.login_tecnico,
        identidad.autenticacion_ref,
        identidad.autenticacion_huella_sha256,
        identidad.sesion_ref, identidad.control_sesion_ref,
        control_revision, identidad.control_sesion_huella_sha256,
        r ->> 'actor_ref', r ->> 'perfil_id', perfil_version,
        r ->> 'organizacion_ref', a ->> 'clase_ambito',
        r ->> 'ambito_ref', r ->> 'sesion_huella_sha256',
        registrada_en, prueba_vinculo, huella_vinculo
    );

    IF r ->> 'tipo_consulta' = 'cuadro' THEN
        prueba_alcance := pg_catalog.convert_to(
            'VEC-CT-ALCANCE-ACCESO-RRHH-V1'
            || pg_catalog.chr(10), 'UTF8'
        );
        FOREACH clave IN ARRAY ARRAY[
            acceso_ref, 'cuadro', COALESCE(a ->> 'familia_ref', ''),
            r ->> 'organizacion_ref', a ->> 'clase_ambito',
            r ->> 'ambito_ref', r ->> 'actor_ref',
            r ->> 'perfil_id', perfil_version::text,
            r ->> 'sesion_id', r ->> 'sesion_huella_sha256',
            vec_contratacion_temporal.instante_utc_v1(registrada_en)
        ]::text[] LOOP
            prueba_alcance := prueba_alcance
                || vec_contratacion_temporal.encuadrar_texto_v1(clave);
        END LOOP;
        huella_alcance := pg_catalog.encode(
            pg_catalog.sha256(prueba_alcance), 'hex'
        );
        INSERT INTO vec_contratacion_temporal.alcance_acceso_rrhh (
            acceso_ref, tipo_consulta, familia_ref, organizacion_ref,
            clase_ambito, ambito_ref, actor_ref, perfil_ref,
            perfil_version, sesion_ref, sesion_huella_sha256,
            acceso_registrado_en, prueba_canonica,
            prueba_huella_sha256
        ) VALUES (
            acceso_ref, 'cuadro',
            CASE WHEN pg_catalog.jsonb_typeof(a -> 'familia_ref') = 'null'
                 THEN NULL ELSE a ->> 'familia_ref' END,
            r ->> 'organizacion_ref', a ->> 'clase_ambito',
            r ->> 'ambito_ref', r ->> 'actor_ref',
            r ->> 'perfil_id', perfil_version, r ->> 'sesion_id',
            r ->> 'sesion_huella_sha256', registrada_en,
            prueba_alcance, huella_alcance
        );
    END IF;

    UPDATE vec_contratacion_temporal.control_cadena_accesos_rrhh
       SET ultima_secuencia = secuencia,
           cabeza_sha256 = huella,
           actualizada_en = registrada_en
     WHERE control;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'control del registro de accesos RRHH ausente';
    END IF;
    RETURN pg_catalog.jsonb_build_object(
        'esquema',
        'vec.contratacion-temporal.recibo-acceso-rrhh.o4-05.v2',
        'acceso_ref', acceso_ref, 'secuencia', secuencia,
        'anterior_sha256', anterior, 'huella_sha256', huella,
        'vinculo_identidad_huella_sha256', huella_vinculo,
        'alcance_huella_sha256', huella_alcance,
        'registrada_en',
        vec_contratacion_temporal.instante_utc_v1(registrada_en)
    );
END
$funcion$;

CREATE TRIGGER control_registrador_acceso_rrhh_v2_inmutable BEFORE UPDATE OR DELETE ON
vec_contratacion_temporal.control_registrador_acceso_rrhh_v2
FOR EACH ROW EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();
CREATE TRIGGER control_registrador_acceso_rrhh_v2_no_truncar BEFORE TRUNCATE ON
vec_contratacion_temporal.control_registrador_acceso_rrhh_v2
FOR EACH STATEMENT EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();
CREATE TRIGGER vinculo_identidad_acceso_rrhh_v2_inmutable BEFORE UPDATE OR DELETE ON
vec_contratacion_temporal.vinculo_identidad_acceso_rrhh_v2
FOR EACH ROW EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();
CREATE TRIGGER vinculo_identidad_acceso_rrhh_v2_no_truncar BEFORE TRUNCATE ON
vec_contratacion_temporal.vinculo_identidad_acceso_rrhh_v2
FOR EACH STATEMENT EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();

DO $rls$
DECLARE
    tabla text;
BEGIN
    FOREACH tabla IN ARRAY ARRAY[
        'control_registrador_acceso_rrhh_v2',
        'vinculo_identidad_acceso_rrhh_v2'
    ]::text[] LOOP
        EXECUTE pg_catalog.format(
            'ALTER TABLE vec_contratacion_temporal.%I '
            || 'ENABLE ROW LEVEL SECURITY', tabla
        );
        EXECUTE pg_catalog.format(
            'ALTER TABLE vec_contratacion_temporal.%I '
            || 'FORCE ROW LEVEL SECURITY', tabla
        );
        EXECUTE pg_catalog.format(
            'CREATE POLICY propietario_total '
            || 'ON vec_contratacion_temporal.%I '
            || 'TO vec_contratacion_temporal_propietario '
            || 'USING (true) WITH CHECK (true)', tabla
        );
    END LOOP;
END
$rls$;

REVOKE ALL ON TABLE
    vec_contratacion_temporal.control_registrador_acceso_rrhh_v2,
    vec_contratacion_temporal.vinculo_identidad_acceso_rrhh_v2
FROM PUBLIC,
    vec_contratacion_temporal_migrador,
    vec_contratacion_temporal_ejecutor,
    vec_contratacion_temporal_gobernador,
    vec_contratacion_temporal_confirmador_cobertura,
    vec_contratacion_temporal_lector_resultado_cobertura,
    vec_contratacion_temporal_consultor_rrhh;
REVOKE ALL ON FUNCTION
    vec_contratacion_temporal.registrar_acceso_rrhh_interno_v2(jsonb)
FROM PUBLIC,
    vec_contratacion_temporal_migrador,
    vec_contratacion_temporal_ejecutor,
    vec_contratacion_temporal_gobernador,
    vec_contratacion_temporal_confirmador_cobertura,
    vec_contratacion_temporal_lector_resultado_cobertura,
    vec_contratacion_temporal_consultor_rrhh;

UPDATE vec_contratacion_temporal.control_migracion_consultas_rrhh
   SET version_esquema = 3,
       actualizada_en =
           pg_catalog.date_trunc('microseconds', pg_catalog.clock_timestamp())
 WHERE control AND version_esquema = 2;
UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4
   SET version_esquema = 19,
       actualizada_en =
           pg_catalog.date_trunc('microseconds', pg_catalog.clock_timestamp())
 WHERE control AND version_esquema = 18;

COMMIT;
