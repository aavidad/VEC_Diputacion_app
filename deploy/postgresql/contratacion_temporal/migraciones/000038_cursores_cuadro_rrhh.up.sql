-- O4-05/C2-D1: infraestructura cerrada, sin fachada ni secretos en claro.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_contratacion_temporal:o4_05:consultas_rrhh:migraciones', 0
));

SELECT control
  FROM vec_contratacion_temporal.control_migracion_cobertura_o4
 WHERE control
   AND version_esquema = 17
 FOR UPDATE;
SELECT control
  FROM vec_contratacion_temporal.control_migracion_consultas_rrhh
 WHERE control
   AND version_esquema = 1
 FOR UPDATE;

DO $prevalidacion$
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.control_migracion_cobertura_o4
         WHERE control AND version_esquema = 17
    ) OR NOT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal
               .control_migracion_consultas_rrhh
         WHERE control AND version_esquema = 1
    ) OR pg_catalog.to_regclass(
        'vec_contratacion_temporal.registro_acceso_rrhh'
    ) IS NULL OR pg_catalog.to_regclass(
        'vec_contratacion_temporal.publicacion_version_rrhh'
    ) IS NULL OR pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.encuadrar_texto_v1(text)'
    ) IS NULL OR pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.instante_utc_v1(timestamp with time zone)'
    ) IS NULL OR pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.rechazar_mutacion_historia_v1()'
    ) IS NULL OR pg_catalog.to_regprocedure(
        'public.gen_random_bytes(integer)'
    ) IS NULL OR NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc funcion
          JOIN pg_catalog.pg_namespace esquema ON esquema.oid =
              funcion.pronamespace
          JOIN pg_catalog.pg_depend dependencia ON dependencia.objid =
              funcion.oid
          JOIN pg_catalog.pg_extension extension ON extension.oid =
              dependencia.refobjid
         WHERE funcion.oid = pg_catalog.to_regprocedure(
             'public.gen_random_bytes(integer)')
           AND esquema.nspname = 'public'
           AND extension.extname = 'pgcrypto'
           AND extension.extnamespace = esquema.oid
           AND funcion.proowner = extension.extowner
           AND funcion.prosupport = 0
           AND funcion.proconfig IS NULL
           AND pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
               pg_catalog.pg_get_functiondef(funcion.oid), 'UTF8'
           )), 'hex') =
               '3e5d4a298efb95a8c94a2e47a06244bb747c33f2400461b01531b7b12bc010b6'
           AND dependencia.classid = 'pg_catalog.pg_proc'::regclass
           AND dependencia.refclassid = 'pg_catalog.pg_extension'::regclass
           AND dependencia.deptype = 'e'
    ) OR NOT pg_catalog.has_schema_privilege(
        'vec_contratacion_temporal_propietario', 'public', 'USAGE'
    ) OR NOT pg_catalog.has_function_privilege(
        'vec_contratacion_temporal_propietario',
        'public.gen_random_bytes(integer)', 'EXECUTE'
    ) OR pg_catalog.has_schema_privilege(
        'public', 'public', 'USAGE'
    ) OR pg_catalog.has_function_privilege(
        'public', 'public.gen_random_bytes(integer)', 'EXECUTE'
    ) OR pg_catalog.to_regclass(
        'vec_contratacion_temporal.control_cursores_cuadro_rrhh'
    ) IS NOT NULL OR pg_catalog.to_regclass(
        'vec_contratacion_temporal.alcance_acceso_rrhh'
    ) IS NOT NULL OR pg_catalog.to_regclass(
        'vec_contratacion_temporal.familia_cursor_cuadro_rrhh'
    ) IS NOT NULL OR pg_catalog.to_regclass(
        'vec_contratacion_temporal.cursor_cuadro_rrhh'
    ) IS NOT NULL OR pg_catalog.to_regclass(
        'vec_contratacion_temporal.consumo_cursor_cuadro_rrhh'
    ) IS NOT NULL OR pg_catalog.to_regclass(
        'vec_contratacion_temporal.revocacion_familia_cursor_rrhh'
    ) IS NOT NULL OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_constraint
         WHERE conrelid =
               'vec_contratacion_temporal.registro_acceso_rrhh'::regclass
           AND conname IN (
               'registro_acceso_rrhh_cursor_alcance_unico',
               'registro_acceso_rrhh_cursor_identidad_unica',
               'registro_acceso_rrhh_cursor_consumo_unico',
               'registro_acceso_rrhh_cursor_tipo_unico'
           )
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'estado incompatible para cursores del cuadro RRHH';
    END IF;
END
$prevalidacion$;

ALTER TABLE vec_contratacion_temporal.registro_acceso_rrhh
    ADD CONSTRAINT registro_acceso_rrhh_cursor_alcance_unico
        UNIQUE (acceso_ref, organizacion_ref, ambito_ref),
    ADD CONSTRAINT registro_acceso_rrhh_cursor_identidad_unica
        UNIQUE (
            acceso_ref, organizacion_ref, ambito_ref, actor_ref,
            perfil_id, perfil_version, sesion_id, sesion_huella_sha256,
            registrada_en
        ),
    ADD CONSTRAINT registro_acceso_rrhh_cursor_consumo_unico
        UNIQUE (
            acceso_ref, decision_ref, decision_huella_sha256,
            consumo_vec_huella_sha256, registrada_en
        ),
    ADD CONSTRAINT registro_acceso_rrhh_cursor_tipo_unico
        UNIQUE (acceso_ref, tipo_consulta);

CREATE TABLE vec_contratacion_temporal.control_cursores_cuadro_rrhh (
    control boolean PRIMARY KEY DEFAULT true CHECK (control),
    version_esquema integer NOT NULL CHECK (version_esquema = 1),
    reloj timestamptz(6) NOT NULL CHECK (
        reloj = pg_catalog.date_trunc('microseconds', reloj)
    )
);

INSERT INTO
vec_contratacion_temporal.control_cursores_cuadro_rrhh (
    control, version_esquema, reloj
) VALUES (
    true, 1,
    pg_catalog.date_trunc('microseconds', pg_catalog.clock_timestamp())
);

CREATE TABLE vec_contratacion_temporal.alcance_acceso_rrhh (
    acceso_ref text PRIMARY KEY,
    tipo_consulta text NOT NULL,
    familia_ref text,
    organizacion_ref text NOT NULL,
    clase_ambito text NOT NULL,
    ambito_ref text NOT NULL,
    actor_ref text NOT NULL,
    perfil_ref text NOT NULL,
    perfil_version numeric(20, 0) NOT NULL,
    sesion_ref text NOT NULL,
    sesion_huella_sha256 text NOT NULL,
    acceso_registrado_en timestamptz(6) NOT NULL,
    prueba_canonica bytea NOT NULL,
    prueba_huella_sha256 text NOT NULL UNIQUE,
    UNIQUE (acceso_ref, familia_ref),
    UNIQUE (
        acceso_ref, familia_ref, organizacion_ref, clase_ambito,
        ambito_ref, actor_ref, perfil_ref, perfil_version, sesion_ref,
        sesion_huella_sha256, acceso_registrado_en
    ),
    UNIQUE (acceso_ref, familia_ref, acceso_registrado_en),
    FOREIGN KEY (acceso_ref, organizacion_ref, ambito_ref)
        REFERENCES vec_contratacion_temporal.registro_acceso_rrhh(
            acceso_ref, organizacion_ref, ambito_ref
        ) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (acceso_ref, tipo_consulta)
        REFERENCES vec_contratacion_temporal.registro_acceso_rrhh(
            acceso_ref, tipo_consulta
        ) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (
        acceso_ref, organizacion_ref, ambito_ref, actor_ref,
        perfil_ref, perfil_version, sesion_ref, sesion_huella_sha256,
        acceso_registrado_en
    ) REFERENCES vec_contratacion_temporal.registro_acceso_rrhh(
        acceso_ref, organizacion_ref, ambito_ref, actor_ref,
        perfil_id, perfil_version, sesion_id, sesion_huella_sha256,
        registrada_en
    ) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CHECK (
        tipo_consulta = 'cuadro'
        AND (
            familia_ref IS NULL
            OR familia_ref ~ '^familia:cursor:rrhh:[0-9a-f]{32}$'
        )
        AND organizacion_ref ~
            '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND clase_ambito IN ('organizacion', 'centro', 'unidad_gestion')
        AND (clase_ambito <> 'organizacion'
             OR ambito_ref = organizacion_ref)
        AND ambito_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND actor_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND perfil_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND sesion_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND perfil_version BETWEEN 1 AND 9007199254740991::numeric
        AND sesion_huella_sha256 ~ '^[0-9a-f]{64}$'
        AND sesion_huella_sha256 <> pg_catalog.repeat('0', 64)
        AND acceso_registrado_en =
            pg_catalog.date_trunc('microseconds', acceso_registrado_en)
    ),
    CHECK (
        prueba_canonica =
            pg_catalog.convert_to(
                'VEC-CT-ALCANCE-ACCESO-RRHH-V1' || pg_catalog.chr(10),
                'UTF8'
            )
            || vec_contratacion_temporal.encuadrar_texto_v1(acceso_ref)
            || vec_contratacion_temporal.encuadrar_texto_v1(tipo_consulta)
            || vec_contratacion_temporal.encuadrar_texto_v1(
                COALESCE(familia_ref, '')
            )
            || vec_contratacion_temporal.encuadrar_texto_v1(organizacion_ref)
            || vec_contratacion_temporal.encuadrar_texto_v1(clase_ambito)
            || vec_contratacion_temporal.encuadrar_texto_v1(ambito_ref)
            || vec_contratacion_temporal.encuadrar_texto_v1(actor_ref)
            || vec_contratacion_temporal.encuadrar_texto_v1(perfil_ref)
            || vec_contratacion_temporal.encuadrar_texto_v1(
                perfil_version::text
            )
            || vec_contratacion_temporal.encuadrar_texto_v1(sesion_ref)
            || vec_contratacion_temporal.encuadrar_texto_v1(
                sesion_huella_sha256
            )
            || vec_contratacion_temporal.encuadrar_texto_v1(
                vec_contratacion_temporal.instante_utc_v1(
                    acceso_registrado_en
                )
            )
        AND pg_catalog.octet_length(prueba_canonica) BETWEEN 256 AND 4096
        AND prueba_huella_sha256 ~ '^[0-9a-f]{64}$'
        AND prueba_huella_sha256 <> pg_catalog.repeat('0', 64)
        AND pg_catalog.encode(
            pg_catalog.sha256(prueba_canonica), 'hex'
        ) = prueba_huella_sha256
    )
);

CREATE TABLE vec_contratacion_temporal.familia_cursor_cuadro_rrhh (
    familia_ref text PRIMARY KEY,
    organizacion_ref text NOT NULL,
    clase_ambito text NOT NULL,
    ambito_ref text NOT NULL,
    actor_ref text NOT NULL,
    perfil_ref text NOT NULL,
    perfil_version numeric(20, 0) NOT NULL,
    sesion_ref text NOT NULL,
    sesion_huella_sha256 text NOT NULL,
    dominio_filtros text NOT NULL,
    filtros_huella_sha256 text NOT NULL,
    limite smallint NOT NULL,
    corte_global numeric(20, 0) NOT NULL,
    creada_en timestamptz(6) NOT NULL,
    valida_hasta timestamptz(6) NOT NULL,
    acceso_origen_ref text NOT NULL UNIQUE,
    prueba_canonica bytea NOT NULL,
    prueba_huella_sha256 text NOT NULL UNIQUE,
    UNIQUE (
        familia_ref, organizacion_ref, clase_ambito, ambito_ref,
        actor_ref, perfil_ref, perfil_version, sesion_ref,
        sesion_huella_sha256
    ),
    UNIQUE (familia_ref, creada_en),
    UNIQUE (familia_ref, creada_en, valida_hasta),
    UNIQUE (familia_ref, creada_en, valida_hasta, acceso_origen_ref),
    FOREIGN KEY (
        acceso_origen_ref, familia_ref, organizacion_ref, clase_ambito,
        ambito_ref, actor_ref, perfil_ref, perfil_version, sesion_ref,
        sesion_huella_sha256, creada_en
    )
        REFERENCES vec_contratacion_temporal.alcance_acceso_rrhh(
            acceso_ref, familia_ref, organizacion_ref, clase_ambito,
            ambito_ref, actor_ref, perfil_ref, perfil_version, sesion_ref,
            sesion_huella_sha256, acceso_registrado_en
        ) ON UPDATE RESTRICT ON DELETE RESTRICT
          DEFERRABLE INITIALLY DEFERRED,
    FOREIGN KEY (corte_global)
        REFERENCES vec_contratacion_temporal.publicacion_version_rrhh(
            corte_global
        ) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CHECK (familia_ref ~ '^familia:cursor:rrhh:[0-9a-f]{32}$'),
    CHECK (
        organizacion_ref ~
            '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND ambito_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND actor_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND perfil_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND sesion_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND clase_ambito IN ('organizacion', 'centro', 'unidad_gestion')
    ),
    CHECK (
        perfil_version BETWEEN 1 AND 9007199254740991::numeric
        AND limite BETWEEN 1 AND 100
        AND corte_global BETWEEN 1 AND 9007199254740991::numeric
        AND dominio_filtros =
            'vec.contratacion_temporal.filtros_rrhh.cuadro.v1'
    ),
    CHECK (
        sesion_huella_sha256 ~ '^[0-9a-f]{64}$'
        AND filtros_huella_sha256 ~ '^[0-9a-f]{64}$'
        AND prueba_huella_sha256 ~ '^[0-9a-f]{64}$'
        AND sesion_huella_sha256 <> pg_catalog.repeat('0', 64)
        AND filtros_huella_sha256 <> pg_catalog.repeat('0', 64)
        AND prueba_huella_sha256 <> pg_catalog.repeat('0', 64)
    ),
    CHECK (
        creada_en = pg_catalog.date_trunc('microseconds', creada_en)
        AND valida_hasta =
            pg_catalog.date_trunc('microseconds', valida_hasta)
        AND valida_hasta > creada_en
        AND valida_hasta <= creada_en + interval '5 minutes'
    ),
    CHECK (
        prueba_canonica =
            pg_catalog.convert_to(
                'VEC-CT-FAMILIA-CURSOR-CUADRO-RRHH-V1'
                    || pg_catalog.chr(10), 'UTF8'
            )
            || vec_contratacion_temporal.encuadrar_texto_v1(familia_ref)
            || vec_contratacion_temporal.encuadrar_texto_v1(organizacion_ref)
            || vec_contratacion_temporal.encuadrar_texto_v1(clase_ambito)
            || vec_contratacion_temporal.encuadrar_texto_v1(ambito_ref)
            || vec_contratacion_temporal.encuadrar_texto_v1(actor_ref)
            || vec_contratacion_temporal.encuadrar_texto_v1(perfil_ref)
            || vec_contratacion_temporal.encuadrar_texto_v1(
                perfil_version::text
            )
            || vec_contratacion_temporal.encuadrar_texto_v1(sesion_ref)
            || vec_contratacion_temporal.encuadrar_texto_v1(
                sesion_huella_sha256
            )
            || vec_contratacion_temporal.encuadrar_texto_v1(dominio_filtros)
            || vec_contratacion_temporal.encuadrar_texto_v1(
                filtros_huella_sha256
            )
            || vec_contratacion_temporal.encuadrar_texto_v1(limite::text)
            || vec_contratacion_temporal.encuadrar_texto_v1(
                corte_global::text
            )
            || vec_contratacion_temporal.encuadrar_texto_v1(
                vec_contratacion_temporal.instante_utc_v1(creada_en)
            )
            || vec_contratacion_temporal.encuadrar_texto_v1(
                vec_contratacion_temporal.instante_utc_v1(valida_hasta)
            )
            || vec_contratacion_temporal.encuadrar_texto_v1(
                acceso_origen_ref
            )
        AND pg_catalog.octet_length(prueba_canonica) BETWEEN 384 AND 8192
        AND pg_catalog.encode(
            pg_catalog.sha256(prueba_canonica), 'hex'
        ) = prueba_huella_sha256
    )
);

ALTER TABLE vec_contratacion_temporal.alcance_acceso_rrhh
    ADD CONSTRAINT alcance_acceso_rrhh_familia_fk
    FOREIGN KEY (
        familia_ref, organizacion_ref, clase_ambito, ambito_ref,
        actor_ref, perfil_ref, perfil_version, sesion_ref,
        sesion_huella_sha256
    ) REFERENCES vec_contratacion_temporal.familia_cursor_cuadro_rrhh(
        familia_ref, organizacion_ref, clase_ambito, ambito_ref,
        actor_ref, perfil_ref, perfil_version, sesion_ref,
        sesion_huella_sha256
    ) ON UPDATE RESTRICT ON DELETE RESTRICT
      DEFERRABLE INITIALLY DEFERRED;

CREATE INDEX familia_cursor_rrhh_caducidad_idx
    ON vec_contratacion_temporal.familia_cursor_cuadro_rrhh(
        valida_hasta, familia_ref
    );

CREATE TABLE vec_contratacion_temporal.cursor_cuadro_rrhh (
    token_huella_sha256 text PRIMARY KEY,
    familia_ref text NOT NULL,
    padre_token_huella_sha256 text UNIQUE,
    pagina numeric(20, 0) NOT NULL,
    pagina_padre numeric(20, 0)
        GENERATED ALWAYS AS (pagina - 1) STORED,
    padre_emitida_en timestamptz(6),
    ultimo_actualizado_en timestamptz(6) NOT NULL,
    ultimo_expediente_ref text NOT NULL,
    familia_creada_en timestamptz(6) NOT NULL,
    familia_valida_hasta timestamptz(6) NOT NULL,
    emitida_en timestamptz(6) NOT NULL,
    acceso_emision_ref text NOT NULL UNIQUE,
    acceso_origen_p2 text GENERATED ALWAYS AS
        (CASE WHEN pagina = 2 THEN acceso_emision_ref END) STORED,
    prueba_canonica bytea NOT NULL,
    prueba_huella_sha256 text NOT NULL UNIQUE,
    UNIQUE (token_huella_sha256, familia_ref),
    UNIQUE (familia_ref, pagina),
    UNIQUE (
        token_huella_sha256, familia_ref, pagina, emitida_en,
        familia_valida_hasta
    ),
    UNIQUE (
        token_huella_sha256, familia_ref, emitida_en,
        familia_valida_hasta, acceso_emision_ref
    ),
    FOREIGN KEY (familia_ref, familia_creada_en, familia_valida_hasta)
        REFERENCES vec_contratacion_temporal.familia_cursor_cuadro_rrhh(
            familia_ref, creada_en, valida_hasta
        ) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (
        familia_ref, familia_creada_en, familia_valida_hasta,
        acceso_origen_p2
    ) REFERENCES vec_contratacion_temporal.familia_cursor_cuadro_rrhh(
        familia_ref, creada_en, valida_hasta, acceso_origen_ref
    ) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (
        padre_token_huella_sha256, familia_ref, pagina_padre,
        padre_emitida_en, familia_valida_hasta
    )
        REFERENCES vec_contratacion_temporal.cursor_cuadro_rrhh(
            token_huella_sha256, familia_ref, pagina, emitida_en,
            familia_valida_hasta
        ) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (acceso_emision_ref, familia_ref, emitida_en)
        REFERENCES vec_contratacion_temporal.alcance_acceso_rrhh(
            acceso_ref, familia_ref, acceso_registrado_en
        )
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CHECK (
        token_huella_sha256 ~ '^[0-9a-f]{64}$'
        AND token_huella_sha256 <> pg_catalog.repeat('0', 64)
        AND (
            padre_token_huella_sha256 IS NULL
            OR (
                padre_token_huella_sha256 ~ '^[0-9a-f]{64}$'
                AND padre_token_huella_sha256 <>
                    pg_catalog.repeat('0', 64)
                AND padre_token_huella_sha256 <> token_huella_sha256
            )
        )
        AND prueba_huella_sha256 ~ '^[0-9a-f]{64}$'
        AND prueba_huella_sha256 <> pg_catalog.repeat('0', 64)
    ),
    CHECK (
        pagina BETWEEN 2 AND 9007199254740991::numeric
        AND (
            (
                pagina = 2
                AND padre_token_huella_sha256 IS NULL
                AND padre_emitida_en IS NULL
            )
            OR (
                pagina > 2
                AND padre_token_huella_sha256 IS NOT NULL
                AND padre_emitida_en IS NOT NULL
                AND padre_emitida_en <= emitida_en
            )
        )
    ),
    CHECK (
        ultimo_expediente_ref ~
            '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND ultimo_actualizado_en =
            pg_catalog.date_trunc('microseconds', ultimo_actualizado_en)
        AND familia_creada_en =
            pg_catalog.date_trunc('microseconds', familia_creada_en)
        AND familia_valida_hasta =
            pg_catalog.date_trunc('microseconds', familia_valida_hasta)
        AND emitida_en = pg_catalog.date_trunc('microseconds', emitida_en)
        AND ultimo_actualizado_en <= emitida_en
        AND emitida_en >= familia_creada_en
        AND emitida_en < familia_valida_hasta
        AND (pagina <> 2 OR emitida_en = familia_creada_en)
        AND (
            padre_emitida_en IS NULL
            OR padre_emitida_en =
                pg_catalog.date_trunc('microseconds', padre_emitida_en)
        )
    ),
    CHECK (
        prueba_canonica =
            pg_catalog.convert_to(
                'VEC-CT-CURSOR-CUADRO-RRHH-V1' || pg_catalog.chr(10),
                'UTF8'
            )
            || vec_contratacion_temporal.encuadrar_texto_v1(
                token_huella_sha256
            )
            || vec_contratacion_temporal.encuadrar_texto_v1(familia_ref)
            || vec_contratacion_temporal.encuadrar_texto_v1(
                COALESCE(padre_token_huella_sha256, '')
            )
            || vec_contratacion_temporal.encuadrar_texto_v1(pagina::text)
            || vec_contratacion_temporal.encuadrar_texto_v1(
                CASE
                    WHEN padre_emitida_en IS NULL THEN ''
                    ELSE vec_contratacion_temporal.instante_utc_v1(
                        padre_emitida_en
                    )
                END
            )
            || vec_contratacion_temporal.encuadrar_texto_v1(
                vec_contratacion_temporal.instante_utc_v1(
                    ultimo_actualizado_en
                )
            )
            || vec_contratacion_temporal.encuadrar_texto_v1(
                ultimo_expediente_ref
            )
            || vec_contratacion_temporal.encuadrar_texto_v1(
                vec_contratacion_temporal.instante_utc_v1(
                    familia_creada_en
                )
            )
            || vec_contratacion_temporal.encuadrar_texto_v1(
                vec_contratacion_temporal.instante_utc_v1(
                    familia_valida_hasta
                )
            )
            || vec_contratacion_temporal.encuadrar_texto_v1(
                vec_contratacion_temporal.instante_utc_v1(emitida_en)
            )
            || vec_contratacion_temporal.encuadrar_texto_v1(
                acceso_emision_ref
            )
        AND pg_catalog.octet_length(prueba_canonica) BETWEEN 320 AND 4096
        AND pg_catalog.encode(
            pg_catalog.sha256(prueba_canonica), 'hex'
        ) = prueba_huella_sha256
    )
);

CREATE TABLE vec_contratacion_temporal.consumo_cursor_cuadro_rrhh (
    token_huella_sha256 text PRIMARY KEY,
    familia_ref text NOT NULL,
    decision_ref text NOT NULL UNIQUE,
    decision_huella_sha256 text NOT NULL UNIQUE,
    consumo_vec_huella_sha256 text NOT NULL UNIQUE,
    acceso_emision_ref text NOT NULL,
    acceso_consumo_ref text NOT NULL UNIQUE,
    cursor_emitida_en timestamptz(6) NOT NULL,
    familia_valida_hasta timestamptz(6) NOT NULL,
    consumido_en timestamptz(6) NOT NULL,
    prueba_canonica bytea NOT NULL,
    prueba_huella_sha256 text NOT NULL UNIQUE,
    UNIQUE (token_huella_sha256, familia_ref,
            acceso_consumo_ref, consumido_en),
    FOREIGN KEY (
        token_huella_sha256, familia_ref, cursor_emitida_en,
        familia_valida_hasta, acceso_emision_ref
    )
        REFERENCES vec_contratacion_temporal.cursor_cuadro_rrhh(
            token_huella_sha256, familia_ref, emitida_en,
            familia_valida_hasta, acceso_emision_ref
        ) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (acceso_consumo_ref, familia_ref, consumido_en)
        REFERENCES vec_contratacion_temporal.alcance_acceso_rrhh(
            acceso_ref, familia_ref, acceso_registrado_en
        ) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (
        acceso_consumo_ref, decision_ref, decision_huella_sha256,
        consumo_vec_huella_sha256, consumido_en
    ) REFERENCES vec_contratacion_temporal.registro_acceso_rrhh(
        acceso_ref, decision_ref, decision_huella_sha256,
        consumo_vec_huella_sha256, registrada_en
    )
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CHECK (
        decision_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND decision_huella_sha256 ~ '^[0-9a-f]{64}$'
        AND consumo_vec_huella_sha256 ~ '^[0-9a-f]{64}$'
        AND prueba_huella_sha256 ~ '^[0-9a-f]{64}$'
        AND decision_huella_sha256 <> pg_catalog.repeat('0', 64)
        AND consumo_vec_huella_sha256 <> pg_catalog.repeat('0', 64)
        AND prueba_huella_sha256 <> pg_catalog.repeat('0', 64)
        AND acceso_consumo_ref <> acceso_emision_ref
        AND cursor_emitida_en =
            pg_catalog.date_trunc('microseconds', cursor_emitida_en)
        AND familia_valida_hasta =
            pg_catalog.date_trunc('microseconds', familia_valida_hasta)
        AND consumido_en =
            pg_catalog.date_trunc('microseconds', consumido_en)
        AND consumido_en >= cursor_emitida_en
        AND consumido_en < familia_valida_hasta
    ),
    CHECK (
        prueba_canonica =
            pg_catalog.convert_to(
                'VEC-CT-CONSUMO-CURSOR-CUADRO-RRHH-V1'
                    || pg_catalog.chr(10), 'UTF8'
            )
            || vec_contratacion_temporal.encuadrar_texto_v1(
                token_huella_sha256
            )
            || vec_contratacion_temporal.encuadrar_texto_v1(familia_ref)
            || vec_contratacion_temporal.encuadrar_texto_v1(decision_ref)
            || vec_contratacion_temporal.encuadrar_texto_v1(
                decision_huella_sha256
            )
            || vec_contratacion_temporal.encuadrar_texto_v1(
                consumo_vec_huella_sha256
            )
            || vec_contratacion_temporal.encuadrar_texto_v1(
                acceso_emision_ref
            )
            || vec_contratacion_temporal.encuadrar_texto_v1(
                acceso_consumo_ref
            )
            || vec_contratacion_temporal.encuadrar_texto_v1(
                vec_contratacion_temporal.instante_utc_v1(
                    cursor_emitida_en
                )
            )
            || vec_contratacion_temporal.encuadrar_texto_v1(
                vec_contratacion_temporal.instante_utc_v1(
                    familia_valida_hasta
                )
            )
            || vec_contratacion_temporal.encuadrar_texto_v1(
                vec_contratacion_temporal.instante_utc_v1(consumido_en)
            )
        AND pg_catalog.octet_length(prueba_canonica) BETWEEN 320 AND 4096
        AND pg_catalog.encode(
            pg_catalog.sha256(prueba_canonica), 'hex'
        ) = prueba_huella_sha256
    )
);

ALTER TABLE vec_contratacion_temporal.cursor_cuadro_rrhh
    ADD CONSTRAINT cursor_cuadro_rrhh_consumo_padre_fk
    FOREIGN KEY (
        padre_token_huella_sha256, familia_ref,
        acceso_emision_ref, emitida_en
    ) REFERENCES vec_contratacion_temporal.consumo_cursor_cuadro_rrhh(
        token_huella_sha256, familia_ref, acceso_consumo_ref, consumido_en
    ) ON UPDATE RESTRICT ON DELETE RESTRICT;

CREATE TABLE vec_contratacion_temporal.revocacion_familia_cursor_rrhh (
    familia_ref text PRIMARY KEY,
    familia_creada_en timestamptz(6) NOT NULL,
    decision_ref text NOT NULL UNIQUE,
    decision_huella_sha256 text NOT NULL UNIQUE,
    auditoria_vec_ref text NOT NULL UNIQUE,
    auditoria_vec_huella_sha256 text NOT NULL UNIQUE,
    motivo_ref text NOT NULL,
    motivo_version numeric(20, 0) NOT NULL,
    motivo_huella_sha256 text NOT NULL,
    revocada_en timestamptz(6) NOT NULL,
    prueba_canonica bytea NOT NULL,
    prueba_huella_sha256 text NOT NULL UNIQUE,
    FOREIGN KEY (familia_ref, familia_creada_en)
        REFERENCES vec_contratacion_temporal.familia_cursor_cuadro_rrhh(
            familia_ref, creada_en
        ) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CHECK (
        decision_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND auditoria_vec_ref ~
            '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND motivo_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND motivo_version BETWEEN 1 AND 9007199254740991::numeric
    ),
    CHECK (
        decision_huella_sha256 ~ '^[0-9a-f]{64}$'
        AND auditoria_vec_huella_sha256 ~ '^[0-9a-f]{64}$'
        AND motivo_huella_sha256 ~ '^[0-9a-f]{64}$'
        AND prueba_huella_sha256 ~ '^[0-9a-f]{64}$'
        AND decision_huella_sha256 <> pg_catalog.repeat('0', 64)
        AND auditoria_vec_huella_sha256 <> pg_catalog.repeat('0', 64)
        AND motivo_huella_sha256 <> pg_catalog.repeat('0', 64)
        AND prueba_huella_sha256 <> pg_catalog.repeat('0', 64)
        AND familia_creada_en =
            pg_catalog.date_trunc('microseconds', familia_creada_en)
        AND revocada_en =
            pg_catalog.date_trunc('microseconds', revocada_en)
        AND revocada_en >= familia_creada_en
    ),
    CHECK (
        prueba_canonica =
            pg_catalog.convert_to(
                'VEC-CT-REVOCACION-FAMILIA-CURSOR-RRHH-V1'
                    || pg_catalog.chr(10), 'UTF8'
            )
            || vec_contratacion_temporal.encuadrar_texto_v1(familia_ref)
            || vec_contratacion_temporal.encuadrar_texto_v1(
                vec_contratacion_temporal.instante_utc_v1(
                    familia_creada_en
                )
            )
            || vec_contratacion_temporal.encuadrar_texto_v1(decision_ref)
            || vec_contratacion_temporal.encuadrar_texto_v1(
                decision_huella_sha256
            )
            || vec_contratacion_temporal.encuadrar_texto_v1(
                auditoria_vec_ref
            )
            || vec_contratacion_temporal.encuadrar_texto_v1(
                auditoria_vec_huella_sha256
            )
            || vec_contratacion_temporal.encuadrar_texto_v1(motivo_ref)
            || vec_contratacion_temporal.encuadrar_texto_v1(
                motivo_version::text
            )
            || vec_contratacion_temporal.encuadrar_texto_v1(
                motivo_huella_sha256
            )
            || vec_contratacion_temporal.encuadrar_texto_v1(
                vec_contratacion_temporal.instante_utc_v1(revocada_en)
            )
        AND pg_catalog.octet_length(prueba_canonica) BETWEEN 384 AND 4096
        AND pg_catalog.encode(
            pg_catalog.sha256(prueba_canonica), 'hex'
        ) = prueba_huella_sha256
    )
);

DO $protecciones$
DECLARE
    v_tabla text;
BEGIN
    FOREACH v_tabla IN ARRAY ARRAY[
        'alcance_acceso_rrhh',
        'familia_cursor_cuadro_rrhh',
        'cursor_cuadro_rrhh',
        'consumo_cursor_cuadro_rrhh',
        'revocacion_familia_cursor_rrhh'
    ]::text[] LOOP
        EXECUTE pg_catalog.format(
            'CREATE TRIGGER %I_inmutable BEFORE UPDATE OR DELETE '
            || 'ON vec_contratacion_temporal.%I FOR EACH ROW '
            || 'EXECUTE FUNCTION '
            || 'vec_contratacion_temporal.rechazar_mutacion_historia_v1()',
            v_tabla, v_tabla
        );
        EXECUTE pg_catalog.format(
            'CREATE TRIGGER %I_no_truncar BEFORE TRUNCATE '
            || 'ON vec_contratacion_temporal.%I FOR EACH STATEMENT '
            || 'EXECUTE FUNCTION '
            || 'vec_contratacion_temporal.rechazar_mutacion_historia_v1()',
            v_tabla, v_tabla
        );
    END LOOP;
END
$protecciones$;

DO $rls$
DECLARE
    v_tabla text;
BEGIN
    FOREACH v_tabla IN ARRAY ARRAY[
        'control_cursores_cuadro_rrhh',
        'alcance_acceso_rrhh',
        'familia_cursor_cuadro_rrhh',
        'cursor_cuadro_rrhh',
        'consumo_cursor_cuadro_rrhh',
        'revocacion_familia_cursor_rrhh'
    ]::text[] LOOP
        EXECUTE pg_catalog.format(
            'ALTER TABLE vec_contratacion_temporal.%I '
            || 'ENABLE ROW LEVEL SECURITY', v_tabla
        );
        EXECUTE pg_catalog.format(
            'ALTER TABLE vec_contratacion_temporal.%I '
            || 'FORCE ROW LEVEL SECURITY', v_tabla
        );
        EXECUTE pg_catalog.format(
            'CREATE POLICY propietario_total '
            || 'ON vec_contratacion_temporal.%I '
            || 'TO vec_contratacion_temporal_propietario '
            || 'USING (true) WITH CHECK (true)', v_tabla
        );
    END LOOP;
END
$rls$;

REVOKE ALL ON TABLE
    vec_contratacion_temporal.control_cursores_cuadro_rrhh,
    vec_contratacion_temporal.alcance_acceso_rrhh,
    vec_contratacion_temporal.familia_cursor_cuadro_rrhh,
    vec_contratacion_temporal.cursor_cuadro_rrhh,
    vec_contratacion_temporal.consumo_cursor_cuadro_rrhh,
    vec_contratacion_temporal.revocacion_familia_cursor_rrhh
FROM PUBLIC,
    vec_contratacion_temporal_migrador,
    vec_contratacion_temporal_ejecutor,
    vec_contratacion_temporal_gobernador,
    vec_contratacion_temporal_confirmador_cobertura,
    vec_contratacion_temporal_lector_resultado_cobertura,
    vec_contratacion_temporal_consultor_rrhh;

UPDATE vec_contratacion_temporal.control_migracion_consultas_rrhh
   SET version_esquema = 2,
       actualizada_en =
           pg_catalog.date_trunc('microseconds', pg_catalog.clock_timestamp())
 WHERE control AND version_esquema = 1;
UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4
   SET version_esquema = 18,
       actualizada_en =
           pg_catalog.date_trunc('microseconds', pg_catalog.clock_timestamp())
 WHERE control AND version_esquema = 17;

COMMIT;
