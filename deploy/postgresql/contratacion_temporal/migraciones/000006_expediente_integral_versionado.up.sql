BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:000006_expediente_integral_versionado', 0
    )
);

DO $prevalidacion$
BEGIN
    IF pg_catalog.to_regclass(
           'vec_contratacion_temporal.expediente_alta_version') IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.reconstruir_efecto_alta_v2(jsonb)'
       ) IS NULL
       OR pg_catalog.to_regclass(
           'vec_contratacion_temporal.expediente_version_integral') IS NOT NULL
       OR pg_catalog.to_regclass(
           'vec_contratacion_temporal.expediente_integral_actual') IS NOT NULL THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'estado incompatible para expediente integral';
    END IF;
END
$prevalidacion$;

-- Única historia de instantáneas para alta, análisis, cobertura y asignación.
-- El JSONB es la representación de lectura; prueba_canonica conserva los bytes
-- probatorios construidos al insertar y no depende de una recodificación futura.
CREATE TABLE vec_contratacion_temporal.expediente_version_integral (
    expediente_ref text NOT NULL,
    version numeric(20, 0) NOT NULL,
    agregado_json jsonb NOT NULL,
    agregado_json_huella_sha256 text NOT NULL,
    prueba_canonica bytea NOT NULL,
    prueba_huella_sha256 text NOT NULL UNIQUE,
    flujo_ref text NOT NULL,
    flujo_version numeric(20, 0) NOT NULL,
    flujo_huella_sha256 text NOT NULL,
    fase_clave text NOT NULL,
    estado text NOT NULL,
    origen_version text NOT NULL,
    operacion_ref text NOT NULL,
    registrada_en timestamptz(6) NOT NULL,
    PRIMARY KEY (expediente_ref, version),
    FOREIGN KEY (expediente_ref)
        REFERENCES vec_contratacion_temporal.expediente_alta,
    CHECK (version BETWEEN 1 AND 9007199254740991::numeric),
    CHECK (pg_catalog.jsonb_typeof(agregado_json) = 'object'),
    CHECK (
        pg_catalog.encode(
            pg_catalog.sha256(
                pg_catalog.convert_to(agregado_json::text, 'UTF8')
            ),
            'hex'
        ) = agregado_json_huella_sha256
    ),
    CHECK (agregado_json_huella_sha256 ~ '^[0-9a-f]{64}$'),
    CHECK (
        pg_catalog.encode(
            pg_catalog.sha256(prueba_canonica), 'hex'
        ) = prueba_huella_sha256
    ),
    CHECK (pg_catalog.octet_length(prueba_canonica) BETWEEN 128 AND 4096),
    CHECK (prueba_huella_sha256 ~ '^[0-9a-f]{64}$'),
    CHECK (flujo_version BETWEEN 1 AND 9007199254740991::numeric),
    CHECK (flujo_huella_sha256 ~ '^[0-9a-f]{64}$'),
    CHECK (estado IN ('en_curso', 'completado', 'cancelado')),
    CHECK (origen_version IN (
        'alta_o2', 'analisis_o3', 'cobertura_o4', 'asignacion_o5'
    )),
    CHECK (operacion_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
    CHECK (registrada_en = pg_catalog.date_trunc(
        'microseconds', registrada_en
    ))
);

CREATE TABLE vec_contratacion_temporal.expediente_integral_actual (
    expediente_ref text PRIMARY KEY,
    version numeric(20, 0) NOT NULL,
    actualizada_en timestamptz(6) NOT NULL,
    operacion_ref text NOT NULL,
    FOREIGN KEY (expediente_ref, version)
        REFERENCES vec_contratacion_temporal.expediente_version_integral,
    CHECK (version BETWEEN 1 AND 9007199254740991::numeric),
    CHECK (actualizada_en = pg_catalog.date_trunc(
        'microseconds', actualizada_en
    )),
    CHECK (operacion_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$')
);

ALTER TABLE vec_contratacion_temporal.expediente_version_integral
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contratacion_temporal.expediente_version_integral
    FORCE ROW LEVEL SECURITY;
CREATE POLICY expediente_version_integral_propietario
    ON vec_contratacion_temporal.expediente_version_integral
    TO vec_contratacion_temporal_propietario
    USING (true) WITH CHECK (true);

ALTER TABLE vec_contratacion_temporal.expediente_integral_actual
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contratacion_temporal.expediente_integral_actual
    FORCE ROW LEVEL SECURITY;
CREATE POLICY expediente_integral_actual_propietario
    ON vec_contratacion_temporal.expediente_integral_actual
    TO vec_contratacion_temporal_propietario
    USING (true) WITH CHECK (true);

CREATE FUNCTION vec_contratacion_temporal.materializar_version_inicial_v1(
    p_expediente_ref text,
    p_version numeric,
    p_alta_canonica bytea,
    p_flujo_ref text,
    p_flujo_version numeric,
    p_flujo_huella_sha256 text,
    p_fase_clave text,
    p_estado text,
    p_registrada_en timestamptz
)
RETURNS void
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_efecto jsonb;
    v_agregado jsonb;
    v_agregado_huella text;
    v_prueba bytea;
    v_operacion_ref text;
    v_organizacion_ref text;
    v_numero_visible text;
BEGIN
    IF p_version <> 1 THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'versión inicial incompatible';
    END IF;
    BEGIN
        v_efecto := pg_catalog.convert_from(p_alta_canonica, 'UTF8')::jsonb;
    EXCEPTION
        WHEN OTHERS THEN
            RAISE EXCEPTION USING
                ERRCODE = '22023',
                MESSAGE = 'alta canónica no interpretable';
    END;
    IF vec_contratacion_temporal.reconstruir_efecto_alta_v2(v_efecto)
           IS DISTINCT FROM p_alta_canonica
       OR v_efecto ->> 'expediente_ref' IS DISTINCT FROM p_expediente_ref
       OR (v_efecto ->> 'version')::numeric IS DISTINCT FROM p_version
       OR v_efecto #>> '{flujo,definicion_ref}' IS DISTINCT FROM p_flujo_ref
       OR (v_efecto #>> '{flujo,version}')::numeric
            IS DISTINCT FROM p_flujo_version
       OR v_efecto #>> '{flujo,huella_sha256}'
            IS DISTINCT FROM p_flujo_huella_sha256
       OR v_efecto ->> 'fase_actual' IS DISTINCT FROM p_fase_clave
       OR v_efecto ->> 'estado_actual' IS DISTINCT FROM p_estado
       OR (v_efecto #>> '{actuacion,secuencia}')::numeric <> 1
       OR (v_efecto #>> '{actuacion,version_expediente}')::numeric <> 1 THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'alta no ligada al expediente integral';
    END IF;

    SELECT e.organizacion_ref, e.numero_visible
      INTO STRICT v_organizacion_ref, v_numero_visible
      FROM vec_contratacion_temporal.expediente_alta e
     WHERE e.expediente_ref = p_expediente_ref;
    IF v_efecto ->> 'organizacion_ref'
           IS DISTINCT FROM v_organizacion_ref
       OR v_efecto ->> 'numero_visible'
           IS DISTINCT FROM v_numero_visible THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'identidad de alta incompatible';
    END IF;

    v_agregado := pg_catalog.jsonb_build_object(
        'referencia', p_expediente_ref,
        'organizacion_ref', v_organizacion_ref,
        'numero_visible', v_numero_visible,
        'version', p_version,
        'flujo', v_efecto -> 'flujo',
        'fase_actual', p_fase_clave,
        'estado_actual', p_estado,
        'solicitud', v_efecto -> 'solicitud',
        'creado_en', v_efecto -> 'creado_en',
        'actualizado_en', v_efecto -> 'actualizado_en',
        'actuaciones', pg_catalog.jsonb_build_array(
            v_efecto -> 'actuacion'
        )
    );
    v_agregado_huella := pg_catalog.encode(
        pg_catalog.sha256(
            pg_catalog.convert_to(v_agregado::text, 'UTF8')
        ),
        'hex'
    );
    v_operacion_ref := 'alta:' || (v_efecto ->> 'recibo_ref');
    v_prueba :=
        pg_catalog.convert_to(
            'VEC-CT-EXPEDIENTE-INTEGRAL-V1' || chr(10), 'UTF8'
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(p_expediente_ref)
        || vec_contratacion_temporal.encuadrar_texto_v1(p_version::text)
        || vec_contratacion_temporal.encuadrar_texto_v1(v_agregado_huella)
        || vec_contratacion_temporal.encuadrar_texto_v1(p_flujo_ref)
        || vec_contratacion_temporal.encuadrar_texto_v1(
            p_flujo_version::text
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            p_flujo_huella_sha256
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(p_fase_clave)
        || vec_contratacion_temporal.encuadrar_texto_v1(p_estado)
        || vec_contratacion_temporal.encuadrar_texto_v1('alta_o2')
        || vec_contratacion_temporal.encuadrar_texto_v1(v_operacion_ref)
        || vec_contratacion_temporal.encuadrar_texto_v1(
            vec_contratacion_temporal.instante_utc_v1(p_registrada_en)
        );

    INSERT INTO vec_contratacion_temporal.expediente_version_integral (
        expediente_ref, version, agregado_json,
        agregado_json_huella_sha256, prueba_canonica,
        prueba_huella_sha256, flujo_ref, flujo_version,
        flujo_huella_sha256, fase_clave, estado, origen_version,
        operacion_ref, registrada_en
    ) VALUES (
        p_expediente_ref, p_version, v_agregado,
        v_agregado_huella, v_prueba,
        pg_catalog.encode(pg_catalog.sha256(v_prueba), 'hex'),
        p_flujo_ref, p_flujo_version, p_flujo_huella_sha256,
        p_fase_clave, p_estado, 'alta_o2', v_operacion_ref,
        p_registrada_en
    );
    INSERT INTO vec_contratacion_temporal.expediente_integral_actual (
        expediente_ref, version, actualizada_en, operacion_ref
    ) VALUES (
        p_expediente_ref, p_version, p_registrada_en, v_operacion_ref
    );
END
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.materializar_alta_integral_v1()
RETURNS trigger
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
BEGIN
    PERFORM vec_contratacion_temporal.materializar_version_inicial_v1(
        NEW.expediente_ref, NEW.version, NEW.alta_canonica,
        NEW.flujo_ref, NEW.flujo_version, NEW.flujo_huella_sha256,
        NEW.fase_clave, NEW.estado, NEW.registrada_en
    );
    RETURN NEW;
END
$funcion$;

CREATE TRIGGER expediente_alta_version_materializar_integral
AFTER INSERT ON vec_contratacion_temporal.expediente_alta_version
FOR EACH ROW
EXECUTE FUNCTION
    vec_contratacion_temporal.materializar_alta_integral_v1();

SELECT vec_contratacion_temporal.materializar_version_inicial_v1(
    v.expediente_ref, v.version, v.alta_canonica,
    v.flujo_ref, v.flujo_version, v.flujo_huella_sha256,
    v.fase_clave, v.estado, v.registrada_en
)
FROM vec_contratacion_temporal.expediente_alta_version v
ORDER BY v.expediente_ref, v.version;

CREATE TRIGGER expediente_version_integral_inmutable
BEFORE UPDATE OR DELETE
ON vec_contratacion_temporal.expediente_version_integral
FOR EACH ROW
EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();

REVOKE ALL ON TABLE
    vec_contratacion_temporal.expediente_version_integral,
    vec_contratacion_temporal.expediente_integral_actual
    FROM PUBLIC, vec_contratacion_temporal_ejecutor;
REVOKE ALL ON FUNCTION
    vec_contratacion_temporal.materializar_version_inicial_v1(
        text, numeric, bytea, text, numeric, text, text, text, timestamptz
    ),
    vec_contratacion_temporal.materializar_alta_integral_v1()
    FROM PUBLIC, vec_contratacion_temporal_ejecutor;

COMMIT;
