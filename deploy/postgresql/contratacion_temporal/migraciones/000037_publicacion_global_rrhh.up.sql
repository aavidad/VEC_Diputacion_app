-- O4-05/C2-C: ordinal global de versiones RRHH alineado con COMMIT.
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
   AND version_esquema = 16
 FOR UPDATE;

-- SHARE impide INSERT concurrente en la historia hasta que el trigger y el
-- backfill confirmen juntos. Los escritores que esperan se publican después.
LOCK TABLE vec_contratacion_temporal.expediente_version_integral
    IN SHARE MODE;

DO $prevalidacion$
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.control_migracion_cobertura_o4
         WHERE control
           AND version_esquema = 16
    ) OR NOT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal
               .control_migracion_consultas_rrhh
         WHERE control
           AND version_esquema = 1
    ) OR pg_catalog.to_regclass(
        'vec_contratacion_temporal.expediente_version_integral'
    ) IS NULL OR pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.rechazar_mutacion_historia_v1()'
    ) IS NULL OR pg_catalog.to_regclass(
        'vec_contratacion_temporal.control_publicacion_rrhh'
    ) IS NOT NULL OR pg_catalog.to_regclass(
        'vec_contratacion_temporal.publicacion_version_rrhh'
    ) IS NOT NULL OR pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.extraer_publicacion_rrhh_v1(text,numeric,jsonb,text,text,numeric,text,text,text,timestamp with time zone,text,text)'
    ) IS NOT NULL OR pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.publicar_version_rrhh_v1()'
    ) IS NOT NULL OR (
        SELECT pg_catalog.count(*)
          FROM vec_contratacion_temporal.expediente_version_integral
    ) > 9007199254740991::numeric THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'estado incompatible para publicación global RRHH';
    END IF;
END
$prevalidacion$;

CREATE TABLE vec_contratacion_temporal.control_publicacion_rrhh (
    control boolean PRIMARY KEY DEFAULT true CHECK (control),
    corte_base numeric(20, 0) NOT NULL CHECK (
        corte_base BETWEEN 0 AND 9007199254740991::numeric
    ),
    ultimo_corte numeric(20, 0) NOT NULL CHECK (
        ultimo_corte BETWEEN corte_base
            AND 9007199254740991::numeric
    ),
    creada_en timestamptz(6) NOT NULL,
    actualizada_en timestamptz(6) NOT NULL,
    CHECK (creada_en = pg_catalog.date_trunc('microseconds', creada_en)),
    CHECK (
        actualizada_en =
            pg_catalog.date_trunc('microseconds', actualizada_en)
        AND actualizada_en >= creada_en
    )
);

CREATE TABLE vec_contratacion_temporal.publicacion_version_rrhh (
    expediente_ref text NOT NULL,
    version numeric(20, 0) NOT NULL,
    corte_global numeric(20, 0) NOT NULL UNIQUE,
    organizacion_ref text NOT NULL,
    numero_visible text NOT NULL,
    flujo_ref text NOT NULL,
    flujo_version numeric(20, 0) NOT NULL,
    flujo_huella_sha256 text NOT NULL,
    fase_clave text NOT NULL,
    estado_clave text NOT NULL,
    centro_ref text NOT NULL,
    categoria_ref text NOT NULL,
    modalidad_clave text,
    unidad_ref text,
    creado_en timestamptz(6) NOT NULL,
    actualizado_en timestamptz(6) NOT NULL,
    agregado_huella_sha256 text NOT NULL,
    registrada_en timestamptz(6) NOT NULL,
    PRIMARY KEY (expediente_ref, version),
    FOREIGN KEY (expediente_ref, version)
        REFERENCES
            vec_contratacion_temporal.expediente_version_integral(
                expediente_ref,
                version
            ),
    CHECK (version BETWEEN 1 AND 9007199254740991::numeric),
    CHECK (corte_global BETWEEN 1 AND 9007199254740991::numeric),
    CHECK (
        expediente_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND organizacion_ref ~
            '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND numero_visible ~ '^[0-9]{4}/[A-Za-z0-9._-]{1,40}$'
        AND flujo_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND centro_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND categoria_ref ~
            '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND (
            unidad_ref IS NULL
            OR unidad_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        )
    ),
    CHECK (flujo_version BETWEEN 1 AND 9007199254740991::numeric),
    CHECK (
        flujo_huella_sha256 ~ '^[0-9a-f]{64}$'
        AND flujo_huella_sha256 <> pg_catalog.repeat('0', 64)
        AND agregado_huella_sha256 ~ '^[0-9a-f]{64}$'
        AND agregado_huella_sha256 <> pg_catalog.repeat('0', 64)
    ),
    CHECK (fase_clave ~ '^[a-z][a-z0-9._-]{1,79}$'),
    CHECK (estado_clave IN ('en_curso', 'completado', 'cancelado')),
    CHECK (
        modalidad_clave IS NULL
        OR modalidad_clave ~ '^[a-z][a-z0-9._-]{1,79}$'
    ),
    CHECK (
        creado_en = pg_catalog.date_trunc('microseconds', creado_en)
        AND actualizado_en =
            pg_catalog.date_trunc('microseconds', actualizado_en)
        AND registrada_en =
            pg_catalog.date_trunc('microseconds', registrada_en)
        AND creado_en <= actualizado_en
        AND actualizado_en <= registrada_en
    )
);

CREATE INDEX publicacion_rrhh_organizacion_orden_idx
    ON vec_contratacion_temporal.publicacion_version_rrhh (
        (organizacion_ref COLLATE "C"),
        actualizado_en DESC,
        (expediente_ref COLLATE "C") DESC,
        corte_global
    );
CREATE INDEX publicacion_rrhh_centro_orden_idx
    ON vec_contratacion_temporal.publicacion_version_rrhh (
        (organizacion_ref COLLATE "C"),
        (centro_ref COLLATE "C"),
        actualizado_en DESC,
        (expediente_ref COLLATE "C") DESC
    );
CREATE INDEX publicacion_rrhh_unidad_orden_idx
    ON vec_contratacion_temporal.publicacion_version_rrhh (
        (organizacion_ref COLLATE "C"),
        (unidad_ref COLLATE "C"),
        actualizado_en DESC,
        (expediente_ref COLLATE "C") DESC
    ) WHERE unidad_ref IS NOT NULL;
CREATE INDEX publicacion_rrhh_filtros_idx
    ON vec_contratacion_temporal.publicacion_version_rrhh (
        (organizacion_ref COLLATE "C"),
        (fase_clave COLLATE "C"),
        (estado_clave COLLATE "C"),
        (categoria_ref COLLATE "C"),
        (modalidad_clave COLLATE "C")
    );

CREATE FUNCTION vec_contratacion_temporal.extraer_publicacion_rrhh_v1(
    p_expediente_ref text,
    p_version numeric,
    p_agregado jsonb,
    p_agregado_huella text,
    p_flujo_ref text,
    p_flujo_version numeric,
    p_flujo_huella text,
    p_fase_clave text,
    p_estado text,
    p_registrada_en timestamptz,
    p_organizacion_esperada text,
    p_numero_esperado text
)
RETURNS TABLE (
    organizacion_ref text,
    numero_visible text,
    centro_ref text,
    categoria_ref text,
    modalidad_clave text,
    unidad_ref text,
    creado_en timestamptz,
    actualizado_en timestamptz
)
LANGUAGE plpgsql
STABLE
SECURITY DEFINER
SET search_path = pg_catalog
SET row_security = 'on'
SET timezone = 'UTC'
AS $funcion$
DECLARE
    v_version_agregado numeric(20, 0);
    v_version_flujo numeric(20, 0);
BEGIN
    IF p_agregado IS NULL
       OR pg_catalog.jsonb_typeof(p_agregado) <> 'object'
       OR pg_catalog.encode(
           pg_catalog.sha256(pg_catalog.convert_to(p_agregado::text, 'UTF8')),
           'hex'
       ) IS DISTINCT FROM p_agregado_huella
       OR p_agregado_huella = pg_catalog.repeat('0', 64)
       OR pg_catalog.jsonb_typeof(p_agregado -> 'version') <> 'number'
       OR p_agregado ->> 'version' !~ '^[1-9][0-9]{0,15}$'
       OR pg_catalog.jsonb_typeof(p_agregado -> 'flujo') <> 'object'
       OR pg_catalog.jsonb_typeof(p_agregado -> 'solicitud') <> 'object' THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'agregado no proyectable para RRHH';
    END IF;

    BEGIN
        v_version_agregado :=
            (p_agregado ->> 'version')::numeric(20, 0);
        v_version_flujo :=
            (p_agregado #>> '{flujo,version}')::numeric(20, 0);
        creado_en := (p_agregado ->> 'creado_en')::timestamptz;
        actualizado_en :=
            (p_agregado ->> 'actualizado_en')::timestamptz;
    EXCEPTION
        WHEN OTHERS THEN
            RAISE EXCEPTION USING
                ERRCODE = '22023',
                MESSAGE = 'agregado no proyectable para RRHH';
    END;

    organizacion_ref := p_agregado ->> 'organizacion_ref';
    numero_visible := p_agregado ->> 'numero_visible';
    centro_ref := p_agregado #>> '{solicitud,centro_ref}';
    categoria_ref := COALESCE(
        p_agregado #>> '{analisis,categoria_ref}',
        p_agregado #>> '{solicitud,categoria_ref}'
    );
    modalidad_clave := p_agregado #>> '{analisis,modalidad_clave}';
    IF p_agregado ? 'analisis' THEN
        IF pg_catalog.jsonb_typeof(p_agregado -> 'analisis') <> 'object'
           OR COALESCE(
               p_agregado #>> '{analisis,categoria_ref}',
               ''
           ) = ''
           OR COALESCE(
               p_agregado #>> '{analisis,modalidad_clave}',
               ''
           ) = '' THEN
            RAISE EXCEPTION USING
                ERRCODE = '22023',
                MESSAGE = 'agregado no proyectable para RRHH';
        END IF;
    END IF;
    IF p_agregado ? 'asignacion'
       AND (
           pg_catalog.jsonb_typeof(p_agregado -> 'asignacion') <> 'object'
           OR COALESCE(
               p_agregado #>> '{asignacion,unidad_ref}',
               ''
           ) = ''
       ) THEN
            RAISE EXCEPTION USING
                ERRCODE = '22023',
                MESSAGE = 'agregado no proyectable para RRHH';
    END IF;
    unidad_ref := p_agregado #>> '{asignacion,unidad_ref}';

    IF v_version_agregado IS DISTINCT FROM p_version
       OR p_agregado ->> 'referencia' IS DISTINCT FROM p_expediente_ref
       OR organizacion_ref IS DISTINCT FROM p_organizacion_esperada
       OR numero_visible IS DISTINCT FROM p_numero_esperado
       OR p_agregado #>> '{flujo,definicion_ref}'
            IS DISTINCT FROM p_flujo_ref
       OR v_version_flujo IS DISTINCT FROM p_flujo_version
       OR p_agregado #>> '{flujo,huella_sha256}'
            IS DISTINCT FROM p_flujo_huella
       OR p_flujo_huella = pg_catalog.repeat('0', 64)
       OR p_agregado ->> 'fase_actual' IS DISTINCT FROM p_fase_clave
       OR p_agregado ->> 'estado_actual' IS DISTINCT FROM p_estado
       OR COALESCE(organizacion_ref, '') !~
            '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR COALESCE(numero_visible, '') !~
            '^[0-9]{4}/[A-Za-z0-9._-]{1,40}$'
       OR COALESCE(centro_ref, '') !~
            '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR COALESCE(categoria_ref, '') !~
            '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR (
            modalidad_clave IS NOT NULL
            AND modalidad_clave !~ '^[a-z][a-z0-9._-]{1,79}$'
       ) OR (
            unidad_ref IS NOT NULL
            AND unidad_ref !~
                '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       ) OR COALESCE(p_agregado ->> 'creado_en', '') !~
            '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]{1,6})?Z$'
       OR COALESCE(p_agregado ->> 'actualizado_en', '') !~
            '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]{1,6})?Z$'
       OR creado_en <> pg_catalog.date_trunc('microseconds', creado_en)
       OR actualizado_en <>
            pg_catalog.date_trunc('microseconds', actualizado_en)
       OR creado_en > actualizado_en
       OR actualizado_en > p_registrada_en THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'agregado no proyectable para RRHH';
    END IF;
    RETURN NEXT;
END
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.publicar_version_rrhh_v1()
RETURNS trigger
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
SET row_security = 'on'
SET timezone = 'UTC'
AS $funcion$
DECLARE
    v_corte numeric(20, 0);
    v_extraida record;
BEGIN
    IF TG_OP <> 'INSERT'
       OR TG_TABLE_SCHEMA <> 'vec_contratacion_temporal'
       OR TG_TABLE_NAME <> 'expediente_version_integral'
       OR current_user <> 'vec_contratacion_temporal_propietario' THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'publicación de versión RRHH no autorizada';
    END IF;

    SELECT extraida.*
      INTO STRICT v_extraida
      FROM vec_contratacion_temporal.expediente_alta alta
      CROSS JOIN LATERAL
           vec_contratacion_temporal.extraer_publicacion_rrhh_v1(
               NEW.expediente_ref,
               NEW.version,
               NEW.agregado_json,
               NEW.agregado_json_huella_sha256,
               NEW.flujo_ref,
               NEW.flujo_version,
               NEW.flujo_huella_sha256,
               NEW.fase_clave,
               NEW.estado,
               NEW.registrada_en,
               alta.organizacion_ref,
               alta.numero_visible
           ) extraida
     WHERE alta.expediente_ref = NEW.expediente_ref;

    SELECT ultimo_corte + 1
      INTO STRICT v_corte
      FROM vec_contratacion_temporal.control_publicacion_rrhh
     WHERE control
     FOR UPDATE;
    IF v_corte > 9007199254740991::numeric THEN
        RAISE EXCEPTION USING
            ERRCODE = '54000',
            MESSAGE = 'capacidad de publicación RRHH agotada';
    END IF;

    INSERT INTO vec_contratacion_temporal.publicacion_version_rrhh (
        expediente_ref, version, corte_global, organizacion_ref,
        numero_visible, flujo_ref, flujo_version, flujo_huella_sha256,
        fase_clave, estado_clave, centro_ref, categoria_ref,
        modalidad_clave, unidad_ref, creado_en, actualizado_en,
        agregado_huella_sha256, registrada_en
    ) VALUES (
        NEW.expediente_ref, NEW.version, v_corte,
        v_extraida.organizacion_ref, v_extraida.numero_visible,
        NEW.flujo_ref, NEW.flujo_version, NEW.flujo_huella_sha256,
        NEW.fase_clave, NEW.estado, v_extraida.centro_ref,
        v_extraida.categoria_ref, v_extraida.modalidad_clave,
        v_extraida.unidad_ref, v_extraida.creado_en,
        v_extraida.actualizado_en, NEW.agregado_json_huella_sha256,
        NEW.registrada_en
    );
    UPDATE vec_contratacion_temporal.control_publicacion_rrhh
       SET ultimo_corte = v_corte,
           actualizada_en = pg_catalog.date_trunc(
               'microseconds',
               pg_catalog.clock_timestamp()
           )
     WHERE control;
    RETURN NEW;
END
$funcion$;

-- El orden C del backfill hace reproducible el corte base, pero no afirma que
-- esas filas históricas confirmasen en ese orden. Solo los cortes posteriores
-- al corte_base quedan alineados con el orden de COMMIT.
WITH fuente AS (
    SELECT historia.*,
           alta.organizacion_ref AS organizacion_esperada,
           alta.numero_visible AS numero_esperado,
           pg_catalog.row_number() OVER (
               ORDER BY historia.expediente_ref COLLATE "C", historia.version
           )::numeric(20, 0) AS corte_global
      FROM vec_contratacion_temporal.expediente_version_integral historia
      JOIN vec_contratacion_temporal.expediente_alta alta
        ON alta.expediente_ref = historia.expediente_ref
)
INSERT INTO vec_contratacion_temporal.publicacion_version_rrhh (
    expediente_ref, version, corte_global, organizacion_ref,
    numero_visible, flujo_ref, flujo_version, flujo_huella_sha256,
    fase_clave, estado_clave, centro_ref, categoria_ref,
    modalidad_clave, unidad_ref, creado_en, actualizado_en,
    agregado_huella_sha256, registrada_en
)
SELECT fuente.expediente_ref, fuente.version, fuente.corte_global,
       extraida.organizacion_ref, extraida.numero_visible,
       fuente.flujo_ref, fuente.flujo_version, fuente.flujo_huella_sha256,
       fuente.fase_clave, fuente.estado, extraida.centro_ref,
       extraida.categoria_ref, extraida.modalidad_clave,
       extraida.unidad_ref, extraida.creado_en, extraida.actualizado_en,
       fuente.agregado_json_huella_sha256, fuente.registrada_en
  FROM fuente
 CROSS JOIN LATERAL
      vec_contratacion_temporal.extraer_publicacion_rrhh_v1(
          fuente.expediente_ref, fuente.version, fuente.agregado_json,
          fuente.agregado_json_huella_sha256, fuente.flujo_ref,
          fuente.flujo_version, fuente.flujo_huella_sha256,
          fuente.fase_clave, fuente.estado, fuente.registrada_en,
          fuente.organizacion_esperada, fuente.numero_esperado
      ) extraida
 ORDER BY fuente.corte_global;

INSERT INTO vec_contratacion_temporal.control_publicacion_rrhh (
    control, corte_base, ultimo_corte, creada_en, actualizada_en
)
SELECT true, cantidad, cantidad, instante, instante
  FROM (
      SELECT pg_catalog.count(*)::numeric(20, 0) AS cantidad
        FROM vec_contratacion_temporal.publicacion_version_rrhh
  ) total
 CROSS JOIN (
      SELECT pg_catalog.date_trunc(
          'microseconds',
          pg_catalog.clock_timestamp()
      ) AS instante
  ) reloj;

CREATE TRIGGER expediente_version_integral_publicar_rrhh
AFTER INSERT
ON vec_contratacion_temporal.expediente_version_integral
FOR EACH ROW
EXECUTE FUNCTION
    vec_contratacion_temporal.publicar_version_rrhh_v1();

CREATE TRIGGER publicacion_version_rrhh_inmutable
BEFORE UPDATE OR DELETE
ON vec_contratacion_temporal.publicacion_version_rrhh
FOR EACH ROW
EXECUTE FUNCTION
    vec_contratacion_temporal.rechazar_mutacion_historia_v1();

CREATE TRIGGER publicacion_version_rrhh_no_truncar
BEFORE TRUNCATE
ON vec_contratacion_temporal.publicacion_version_rrhh
FOR EACH STATEMENT
EXECUTE FUNCTION
    vec_contratacion_temporal.rechazar_mutacion_historia_v1();

DO $rls$
DECLARE
    v_tabla text;
BEGIN
    FOREACH v_tabla IN ARRAY ARRAY[
        'control_publicacion_rrhh',
        'publicacion_version_rrhh'
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
    vec_contratacion_temporal.control_publicacion_rrhh,
    vec_contratacion_temporal.publicacion_version_rrhh
FROM PUBLIC,
    vec_contratacion_temporal_migrador,
    vec_contratacion_temporal_ejecutor,
    vec_contratacion_temporal_gobernador,
    vec_contratacion_temporal_confirmador_cobertura,
    vec_contratacion_temporal_lector_resultado_cobertura,
    vec_contratacion_temporal_consultor_rrhh;

REVOKE ALL ON FUNCTION
    vec_contratacion_temporal.extraer_publicacion_rrhh_v1(
        text, numeric, jsonb, text, text, numeric, text, text, text,
        timestamptz, text, text
    ),
    vec_contratacion_temporal.publicar_version_rrhh_v1()
FROM PUBLIC,
    vec_contratacion_temporal_migrador,
    vec_contratacion_temporal_ejecutor,
    vec_contratacion_temporal_gobernador,
    vec_contratacion_temporal_confirmador_cobertura,
    vec_contratacion_temporal_lector_resultado_cobertura,
    vec_contratacion_temporal_consultor_rrhh;

UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4
   SET version_esquema = 17,
       actualizada_en = pg_catalog.date_trunc(
           'microseconds',
           pg_catalog.clock_timestamp()
       )
 WHERE control
   AND version_esquema = 16;

COMMIT;
