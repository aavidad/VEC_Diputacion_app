-- O4-04D/2: operaciones internas, cierre de historia y barrera pública v4.
-- No publica la transacción exterior de O4-04E ni capacidades runtime.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:o4_04:migraciones', 0
    )
);

SELECT control
  FROM vec_contratacion_temporal.control_migracion_cobertura_o4
 WHERE control AND version_esquema = 3
 FOR UPDATE;

DO $prevalidacion$
BEGIN
    IF NOT EXISTS (
           SELECT 1
             FROM vec_contratacion_temporal.control_migracion_cobertura_o4
            WHERE control AND version_esquema = 3
       )
       OR pg_catalog.to_regclass(
           'vec_contratacion_temporal.consumo_cobertura_lote'
       ) IS NULL
       OR pg_catalog.to_regclass(
           'vec_contratacion_temporal.consumo_cobertura_evidencia'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.o404d_material_lote_v1(jsonb)'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.prevalidar_bloquear_lote_consumo_c1_cobertura_o404d_v1(jsonb)'
       ) IS NOT NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.persistir_lote_consumo_c1_cobertura_o404d_v1(jsonb,jsonb)'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'estado incompatible para operaciones C1 O4-04D';
    END IF;
END
$prevalidacion$;

CREATE FUNCTION
vec_contratacion_temporal.prevalidar_bloquear_lote_consumo_c1_cobertura_o404d_v1(
    p_lote jsonb
)
RETURNS TABLE (
    estado text,
    lote_ref text,
    lote_huella_sha256 text,
    numero_evidencias integer
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
SET row_security = 'on'
SET timezone = 'UTC'
SET lock_timeout = '2s'
AS $funcion$
DECLARE
    v_material bytea;
    v_huella text;
    v_total integer;
    v_bloqueo record;
    v_evidencia record;
    v_coincidencias integer;
    v_exactas boolean;
    v_repetida boolean := false;
    v_statement_ms numeric;
    v_idle_ms numeric;
BEGIN
    SELECT CASE
               WHEN unit = 'ms' AND setting ~ '^[0-9]{1,18}$'
               THEN setting::numeric
           END
      INTO v_statement_ms
      FROM pg_catalog.pg_settings
     WHERE name = 'statement_timeout';

    SELECT CASE
               WHEN unit = 'ms' AND setting ~ '^[0-9]{1,18}$'
               THEN setting::numeric
           END
      INTO v_idle_ms
      FROM pg_catalog.pg_settings
     WHERE name = 'idle_in_transaction_session_timeout';

    IF pg_catalog.current_setting('transaction_isolation') <>
           'serializable'
       OR pg_catalog.current_setting('transaction_read_only') <> 'off'
       OR v_statement_ms IS NULL
       OR v_statement_ms NOT BETWEEN 1 AND 15000
       OR v_idle_ms IS NULL
       OR v_idle_ms NOT BETWEEN 1 AND 20000 THEN
        RAISE EXCEPTION USING
            ERRCODE = '25000',
            MESSAGE = 'prevalidación C1 O4-04D requiere transacción acotada';
    END IF;

    v_material :=
        vec_contratacion_temporal.o404d_material_lote_v1(p_lote);
    IF v_material IS NULL THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'lote C1 O4-04D inválido';
    END IF;
    v_huella := pg_catalog.encode(pg_catalog.sha256(v_material), 'hex');
    v_total := pg_catalog.jsonb_array_length(p_lote -> 'evidencias');
    IF v_huella <> p_lote ->> 'lote_huella_sha256' THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'canon del lote C1 O4-04D divergente';
    END IF;

    PERFORM 1
      FROM vec_contratacion_temporal.control_migracion_cobertura_o4
     WHERE control AND version_esquema = 4
     FOR SHARE;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'barrera O4-04D no disponible';
    END IF;
    IF EXISTS (
           SELECT 1
             FROM pg_catalog.jsonb_array_elements(
                      p_lote -> 'evidencias'
                  ) AS e(valor)
            WHERE pg_catalog.clock_timestamp() >=
                  (e.valor ->> 'valida_hasta')::timestamptz
       ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'lote C1 O4-04D caducado';
    END IF;

    FOR v_bloqueo IN
        SELECT clave
          FROM (
              SELECT 'P:' || (e.valor ->> 'organizacion_ref') || ':'
                     || (e.valor ->> 'peticion_ref') AS clave
                FROM pg_catalog.jsonb_array_elements(
                         p_lote -> 'evidencias'
                     ) AS e(valor)
              UNION
              SELECT 'R:' || (e.valor ->> 'autoridad_ref') || ':'
                     || (e.valor ->> 'generacion') || ':'
                     || (e.valor ->> 'recibo_respuesta_ref')
                FROM pg_catalog.jsonb_array_elements(
                         p_lote -> 'evidencias'
                     ) AS e(valor)
          ) AS claves
         ORDER BY clave COLLATE "C"
    LOOP
        PERFORM pg_catalog.pg_advisory_xact_lock(
            pg_catalog.hashtextextended(
                'vec_contratacion_temporal:o404d:c1:' ||
                    v_bloqueo.clave,
                0
            )
        );
    END LOOP;

    SELECT pg_catalog.count(*),
           pg_catalog.bool_and(
               l.lote_ref = p_lote ->> 'lote_ref'
               AND l.lote_huella_sha256 = v_huella
               AND l.numero_evidencias = v_total
           )
      INTO v_coincidencias, v_exactas
      FROM vec_contratacion_temporal.consumo_cobertura_lote AS l
     WHERE l.lote_ref = p_lote ->> 'lote_ref'
        OR l.lote_huella_sha256 = v_huella;
    IF v_coincidencias > 0 THEN
        IF v_coincidencias <> 1 OR v_exactas IS NOT TRUE THEN
            RAISE EXCEPTION USING
                ERRCODE = '23505',
                MESSAGE = 'identidad de lote C1 O4-04D en colisión';
        END IF;
        v_repetida := true;
    END IF;

    FOR v_evidencia IN
        SELECT valor, ordinalidad
          FROM pg_catalog.jsonb_array_elements(
                   p_lote -> 'evidencias'
               ) WITH ORDINALITY AS e(valor, ordinalidad)
    LOOP
        SELECT pg_catalog.count(*),
               pg_catalog.bool_and(
                   c.lote_ref = p_lote ->> 'lote_ref'
                   AND c.posicion = v_evidencia.ordinalidad
                   AND c.evidencia_huella_sha256 =
                       v_evidencia.valor ->> 'evidencia_huella_sha256'
               )
          INTO v_coincidencias, v_exactas
          FROM vec_contratacion_temporal.consumo_cobertura_evidencia AS c
         WHERE (
                   c.organizacion_ref =
                       v_evidencia.valor ->> 'organizacion_ref'
                   AND c.peticion_ref =
                       v_evidencia.valor ->> 'peticion_ref'
               )
            OR (
                   c.autoridad_ref =
                       v_evidencia.valor ->> 'autoridad_ref'
                   AND c.generacion =
                       (v_evidencia.valor ->> 'generacion')::numeric
                   AND c.recibo_respuesta_ref =
                       v_evidencia.valor ->> 'recibo_respuesta_ref'
               );
        IF v_coincidencias > 0
           AND (v_coincidencias <> 1 OR v_exactas IS NOT TRUE) THEN
            RAISE EXCEPTION USING
                ERRCODE = '23505',
                MESSAGE = 'evidencia C1 O4-04D ya consumida';
        END IF;
        IF v_repetida AND v_coincidencias <> 1 THEN
            RAISE EXCEPTION USING
                ERRCODE = '55000',
                MESSAGE = 'replay C1 O4-04D incompleto';
        END IF;
    END LOOP;

    RETURN QUERY
    SELECT CASE WHEN v_repetida THEN 'repetida' ELSE 'nueva' END,
           p_lote ->> 'lote_ref',
           v_huella,
           v_total;
END
$funcion$;

CREATE FUNCTION
vec_contratacion_temporal.persistir_lote_consumo_c1_cobertura_o404d_v1(
    p_lote jsonb,
    p_resultado_vec jsonb
)
RETURNS TABLE (
    estado text,
    lote_ref text,
    lote_huella_sha256 text,
    numero_evidencias integer
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
SET row_security = 'on'
SET timezone = 'UTC'
SET lock_timeout = '2s'
AS $funcion$
DECLARE
    v_claves text[];
    v_material bytea;
    v_huella text;
    v_total integer;
    v_existente record;
    v_evidencia record;
    v_persistido timestamptz(6);
    v_registrada timestamptz;
    v_revalidada timestamptz;
    v_prueba_vinculo text;
BEGIN
    PERFORM *
      FROM vec_contratacion_temporal
           .prevalidar_bloquear_lote_consumo_c1_cobertura_o404d_v1(
               p_lote
           );
    IF p_resultado_vec IS NULL
       OR pg_catalog.jsonb_typeof(p_resultado_vec) <> 'object'
       OR pg_catalog.pg_column_size(p_resultado_vec) > 16384 THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'resultado VEC O4-04D inválido';
    END IF;

    SELECT pg_catalog.array_agg(clave ORDER BY clave COLLATE "C")
      INTO v_claves
      FROM pg_catalog.jsonb_object_keys(p_resultado_vec) AS c(clave);
    IF v_claves IS DISTINCT FROM ARRAY[
           'codigo', 'concedida',
           'contexto_recurso_huella_sha256', 'correlacion_ref',
           'decision_huella_sha256', 'decision_ref',
           'expediente_ref', 'huella_orden_sha256',
           'lote_huella_sha256', 'organizacion_ref',
           'prueba_vinculo_sha256', 'rama', 'registrada_en',
           'reserva_ref', 'revalidada_en', 'version_expediente'
       ]::text[]
       OR p_resultado_vec ->> 'rama' IS DISTINCT FROM 'concedida'
       OR pg_catalog.jsonb_typeof(
              p_resultado_vec -> 'concedida'
          ) <> 'boolean'
       OR (p_resultado_vec ->> 'concedida')::boolean IS NOT TRUE
       OR p_resultado_vec ->> 'codigo' IS DISTINCT FROM 'concedida'
       OR p_resultado_vec ->> 'decision_ref' IS DISTINCT FROM
          p_lote ->> 'decision_vec_ref'
       OR p_resultado_vec ->> 'correlacion_ref' IS DISTINCT FROM
          p_lote ->> 'correlacion_vec_ref'
       OR p_resultado_vec ->> 'organizacion_ref' IS DISTINCT FROM
          p_lote ->> 'organizacion_ref'
       OR p_resultado_vec ->> 'expediente_ref' IS DISTINCT FROM
          p_lote ->> 'expediente_ref'
       OR pg_catalog.jsonb_typeof(
              p_resultado_vec -> 'version_expediente'
          ) IS DISTINCT FROM 'number'
       OR (p_resultado_vec ->> 'version_expediente') !~
          '^([2-9]|[1-9][0-9]+)$'
       OR (p_resultado_vec ->> 'version_expediente')::numeric
          IS DISTINCT FROM
          (p_lote ->> 'version_expediente')::numeric
       OR p_resultado_vec ->> 'reserva_ref' IS DISTINCT FROM
          p_lote ->> 'reserva_ref'
       OR p_resultado_vec ->> 'lote_huella_sha256' IS DISTINCT FROM
          p_lote ->> 'lote_huella_sha256'
       OR p_resultado_vec ->> 'huella_orden_sha256' IS DISTINCT FROM
          p_lote ->> 'huella_orden_sha256'
       OR (p_resultado_vec ->> 'contexto_recurso_huella_sha256') !~
          '^[a-f0-9]{64}$'
       OR (p_resultado_vec ->> 'decision_huella_sha256') !~
          '^[a-f0-9]{64}$'
       OR (p_resultado_vec ->> 'prueba_vinculo_sha256') !~
          '^[a-f0-9]{64}$'
       OR NOT vec_contratacion_temporal.gobi_o404b_instante_texto_valido(
           p_resultado_vec ->> 'registrada_en', false
       )
       OR NOT vec_contratacion_temporal.gobi_o404b_instante_texto_valido(
           p_resultado_vec ->> 'revalidada_en', false
       ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'resultado VEC O4-04D divergente';
    END IF;
    v_registrada := (p_resultado_vec ->> 'registrada_en')::timestamptz;
    v_revalidada := (p_resultado_vec ->> 'revalidada_en')::timestamptz;
    v_prueba_vinculo := pg_catalog.encode(
        pg_catalog.sha256(
            pg_catalog.decode(
                p_resultado_vec ->> 'decision_huella_sha256', 'hex'
            )
            || pg_catalog.decode(
                p_resultado_vec
                    ->> 'contexto_recurso_huella_sha256',
                'hex'
            )
            || pg_catalog.decode(
                p_lote ->> 'huella_orden_sha256', 'hex'
            )
            || pg_catalog.decode(
                p_lote ->> 'lote_huella_sha256', 'hex'
            )
            || pg_catalog.int4send(
                pg_catalog.octet_length(
                    p_resultado_vec ->> 'decision_ref'
                )
            )
            || pg_catalog.convert_to(
                p_resultado_vec ->> 'decision_ref', 'UTF8'
            )
            || pg_catalog.int4send(
                pg_catalog.octet_length(
                    p_resultado_vec ->> 'correlacion_ref'
                )
            )
            || pg_catalog.convert_to(
                p_resultado_vec ->> 'correlacion_ref', 'UTF8'
            )
            || pg_catalog.int8send(
                vec_contratacion_temporal.gobi_o404b_microsegundos(
                    v_registrada
                )
            )
            || pg_catalog.int8send(
                vec_contratacion_temporal.gobi_o404b_microsegundos(
                    v_revalidada
                )
            )
        ),
        'hex'
    );
    IF v_revalidada < v_registrada
       OR v_revalidada < (p_lote ->> 'efecto_en')::timestamptz
       OR v_prueba_vinculo IS DISTINCT FROM
          p_resultado_vec ->> 'prueba_vinculo_sha256' THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'instantes VEC O4-04D divergentes';
    END IF;

    v_material := vec_contratacion_temporal.o404d_material_lote_v1(p_lote);
    v_huella := pg_catalog.encode(pg_catalog.sha256(v_material), 'hex');
    v_total := pg_catalog.jsonb_array_length(p_lote -> 'evidencias');
    SELECT l.*
      INTO v_existente
      FROM vec_contratacion_temporal.consumo_cobertura_lote AS l
     WHERE l.lote_ref = p_lote ->> 'lote_ref';
    IF FOUND THEN
        IF v_existente.lote_huella_sha256 IS DISTINCT FROM v_huella
           OR v_existente.huella_orden_sha256 IS DISTINCT FROM
              p_lote ->> 'huella_orden_sha256'
           OR v_existente.decision_vec_ref IS DISTINCT FROM
              p_resultado_vec ->> 'decision_ref'
           OR v_existente.correlacion_vec_ref IS DISTINCT FROM
              p_resultado_vec ->> 'correlacion_ref'
           OR v_existente.reserva_ref IS DISTINCT FROM
              p_resultado_vec ->> 'reserva_ref'
           OR v_existente.contexto_recurso_huella_sha256
              IS DISTINCT FROM
              p_resultado_vec
                  ->> 'contexto_recurso_huella_sha256'
           OR v_existente.decision_vec_huella_sha256 IS DISTINCT FROM
              p_resultado_vec ->> 'decision_huella_sha256'
           OR v_existente.prueba_vinculo_vec_sha256 IS DISTINCT FROM
              v_prueba_vinculo
           OR v_existente.codigo_probatorio_vec IS DISTINCT FROM
              p_resultado_vec ->> 'codigo'
           OR v_existente.registrada_vec_en IS DISTINCT FROM v_registrada
           OR v_existente.revalidada_vec_en IS DISTINCT FROM v_revalidada THEN
            RAISE EXCEPTION USING
                ERRCODE = '23505',
                MESSAGE = 'replay VEC O4-04D divergente';
        END IF;
        RETURN QUERY
        SELECT 'repetida'::text,
               p_lote ->> 'lote_ref',
               v_existente.lote_huella_sha256,
               v_total;
        RETURN;
    END IF;

    v_persistido := pg_catalog.date_trunc(
        'microseconds', pg_catalog.clock_timestamp()
    );
    INSERT INTO vec_contratacion_temporal.consumo_cobertura_lote (
        lote_ref,
        lote_huella_sha256,
        organizacion_ref,
        expediente_ref,
        version_expediente,
        reserva_ref,
        preparacion_c1_ref,
        preparacion_c1_huella_sha256,
        huella_orden_sha256,
        huella_ordenes_c1_sha256,
        catalogo_ref,
        catalogo_version,
        catalogo_huella_sha256,
        decision_vec_ref,
        correlacion_vec_ref,
        contexto_recurso_huella_sha256,
        decision_vec_huella_sha256,
        prueba_vinculo_vec_sha256,
        codigo_probatorio_vec,
        registrada_vec_en,
        revalidada_vec_en,
        numero_evidencias,
        efecto_en,
        persistido_en,
        lote_canon
    ) VALUES (
        p_lote ->> 'lote_ref',
        v_huella,
        p_lote ->> 'organizacion_ref', p_lote ->> 'expediente_ref',
        (p_lote ->> 'version_expediente')::numeric,
        p_lote ->> 'reserva_ref',
        p_lote ->> 'preparacion_c1_ref',
        p_lote ->> 'preparacion_c1_huella_sha256',
        p_lote ->> 'huella_orden_sha256',
        p_lote ->> 'huella_ordenes_c1_sha256',
        p_lote ->> 'catalogo_ref',
        (p_lote ->> 'catalogo_version')::numeric,
        p_lote ->> 'catalogo_huella_sha256',
        p_lote ->> 'decision_vec_ref',
        p_lote ->> 'correlacion_vec_ref',
        p_resultado_vec ->> 'contexto_recurso_huella_sha256',
        p_resultado_vec ->> 'decision_huella_sha256',
        p_resultado_vec ->> 'prueba_vinculo_sha256',
        'concedida',
        v_registrada,
        v_revalidada,
        v_total,
        (p_lote ->> 'efecto_en')::timestamptz,
        v_persistido,
        v_material
    );

    FOR v_evidencia IN
        SELECT valor, ordinalidad
          FROM pg_catalog.jsonb_array_elements(
                   p_lote -> 'evidencias'
               ) WITH ORDINALITY AS e(valor, ordinalidad)
    LOOP
        INSERT INTO
        vec_contratacion_temporal.consumo_cobertura_evidencia (
            lote_ref,
            posicion,
            total,
            evidencia_huella_sha256,
            organizacion_ref,
            expediente_ref,
            version_expediente,
            peticion_ref,
            huella_peticion_sha256,
            huella_resultado_sha256,
            autoridad_ref,
            generacion,
            recibo_respuesta_ref,
            huella_respuesta_sha256,
            catalogo_ref,
            catalogo_version,
            catalogo_huella_sha256,
            via_clave,
            comprobacion_clave,
            comprobacion_resultado,
            orden_comprobacion,
            comprobacion_obligatoria,
            procedencia_clave,
            definicion_fuente_ref,
            categoria_ref,
            periodo_inicio,
            periodo_fin,
            solicitada_en,
            emitida_en,
            valida_hasta,
            verificador_ref,
            publicador_catalogo_ref,
            peticion_canon,
            resultado_canon,
            atestacion_canon,
            confirmacion_tcb_canon,
            catalogo_canon,
            verificador_canon,
            resumen_canon,
            evidencia_canon
        ) VALUES (
            p_lote ->> 'lote_ref',
            v_evidencia.ordinalidad,
            v_total,
            v_evidencia.valor ->> 'evidencia_huella_sha256',
            v_evidencia.valor ->> 'organizacion_ref',
            v_evidencia.valor ->> 'expediente_ref',
            (v_evidencia.valor ->> 'version_expediente')::numeric,
            v_evidencia.valor ->> 'peticion_ref',
            v_evidencia.valor ->> 'huella_peticion_sha256',
            v_evidencia.valor ->> 'huella_resultado_sha256',
            v_evidencia.valor ->> 'autoridad_ref',
            (v_evidencia.valor ->> 'generacion')::numeric,
            v_evidencia.valor ->> 'recibo_respuesta_ref',
            v_evidencia.valor ->> 'huella_respuesta_sha256',
            v_evidencia.valor ->> 'catalogo_ref',
            (v_evidencia.valor ->> 'catalogo_version')::numeric,
            v_evidencia.valor ->> 'catalogo_huella_sha256',
            v_evidencia.valor ->> 'via_clave',
            v_evidencia.valor ->> 'comprobacion_clave',
            v_evidencia.valor ->> 'comprobacion_resultado',
            (v_evidencia.valor ->> 'orden_comprobacion')::integer,
            (v_evidencia.valor ->> 'comprobacion_obligatoria')::boolean,
            v_evidencia.valor ->> 'procedencia_clave',
            v_evidencia.valor ->> 'definicion_fuente_ref',
            v_evidencia.valor ->> 'categoria_ref',
            (v_evidencia.valor ->> 'periodo_inicio')::timestamptz,
            (v_evidencia.valor ->> 'periodo_fin')::timestamptz,
            (v_evidencia.valor ->> 'solicitada_en')::timestamptz,
            (v_evidencia.valor ->> 'emitida_en')::timestamptz,
            (v_evidencia.valor ->> 'valida_hasta')::timestamptz,
            v_evidencia.valor ->> 'verificador_ref',
            v_evidencia.valor ->> 'publicador_catalogo_ref',
            vec_contratacion_temporal.o404d_prueba_canon_v1(
                v_evidencia.valor, 'peticion_canon_hex'
            ),
            vec_contratacion_temporal.o404d_prueba_canon_v1(
                v_evidencia.valor, 'resultado_canon_hex'
            ),
            vec_contratacion_temporal.o404d_prueba_canon_v1(
                v_evidencia.valor, 'atestacion_canon_hex'
            ),
            vec_contratacion_temporal.o404d_prueba_canon_v1(
                v_evidencia.valor, 'confirmacion_tcb_canon_hex'
            ),
            vec_contratacion_temporal.o404d_prueba_canon_v1(
                v_evidencia.valor, 'catalogo_canon_hex'
            ),
            vec_contratacion_temporal.o404d_prueba_canon_v1(
                v_evidencia.valor, 'verificador_canon_hex'
            ),
            vec_contratacion_temporal.o404d_prueba_canon_v1(
                v_evidencia.valor, 'resumen_canon_hex'
            ),
            vec_contratacion_temporal.o404d_material_evidencia_v1(
                v_evidencia.valor
            )
        );
    END LOOP;

    RETURN QUERY
    SELECT 'persistida'::text,
           p_lote ->> 'lote_ref',
           v_huella,
           v_total;
END
$funcion$;

COMMENT ON FUNCTION
vec_contratacion_temporal.prevalidar_bloquear_lote_consumo_c1_cobertura_o404d_v1(
    jsonb
) IS
    'Primitiva D privada, sin EXECUTE runtime; O4-04E deberá invocarla dentro de su única transacción exterior.';

COMMENT ON FUNCTION
vec_contratacion_temporal.persistir_lote_consumo_c1_cobertura_o404d_v1(
    jsonb, jsonb
) IS
    'Primitiva D privada, sin EXECUTE runtime: p_resultado_vec es solo la proyección interna del wrapper VEC; O4-04E no aceptará resultados VEC externos y persistirá en la misma transacción.';

DO $inmutabilidad$
DECLARE
    v_tabla text;
BEGIN
    FOREACH v_tabla IN ARRAY ARRAY[
        'consumo_cobertura_lote',
        'consumo_cobertura_evidencia'
    ]::text[]
    LOOP
        EXECUTE pg_catalog.format(
            'CREATE TRIGGER bloquear_mutacion ' ||
            'BEFORE UPDATE OR DELETE ON vec_contratacion_temporal.%I ' ||
            'FOR EACH ROW EXECUTE FUNCTION ' ||
            'vec_contratacion_temporal.rechazar_mutacion_historia_v1()',
            v_tabla
        );
        EXECUTE pg_catalog.format(
            'CREATE TRIGGER bloquear_truncado ' ||
            'BEFORE TRUNCATE ON vec_contratacion_temporal.%I ' ||
            'FOR EACH STATEMENT EXECUTE FUNCTION ' ||
            'vec_contratacion_temporal.rechazar_mutacion_historia_v1()',
            v_tabla
        );
        EXECUTE pg_catalog.format(
            'ALTER TABLE vec_contratacion_temporal.%I ' ||
            'ENABLE ROW LEVEL SECURITY',
            v_tabla
        );
        EXECUTE pg_catalog.format(
            'ALTER TABLE vec_contratacion_temporal.%I ' ||
            'FORCE ROW LEVEL SECURITY',
            v_tabla
        );
        EXECUTE pg_catalog.format(
            'CREATE POLICY propietario_total ON ' ||
            'vec_contratacion_temporal.%I ' ||
            'TO vec_contratacion_temporal_propietario ' ||
            'USING (true) WITH CHECK (true)',
            v_tabla
        );
    END LOOP;
END
$inmutabilidad$;

UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4
   SET version_esquema = 4,
       actualizada_en = pg_catalog.date_trunc(
           'microseconds', pg_catalog.clock_timestamp()
       )
 WHERE control AND version_esquema = 3;

REVOKE ALL ON FUNCTION
    vec_contratacion_temporal.prevalidar_bloquear_lote_consumo_c1_cobertura_o404d_v1(
        jsonb
    ),
    vec_contratacion_temporal.persistir_lote_consumo_c1_cobertura_o404d_v1(
        jsonb, jsonb
    )
FROM PUBLIC, vec_contratacion_temporal_ejecutor,
     vec_contratacion_temporal_migrador,
     vec_contratacion_temporal_gobernador;

COMMIT;
