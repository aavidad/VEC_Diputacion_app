-- O4-04E/3: canon SQL V1 de la decisión de cobertura gobernada.
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
 WHERE control AND version_esquema = 5
 FOR UPDATE;

DO $prevalidacion$
BEGIN
    IF NOT EXISTS (
           SELECT 1
             FROM vec_contratacion_temporal.control_migracion_cobertura_o4
            WHERE control AND version_esquema = 5
       )
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.o404e_material_propuesta_cobertura_v1(jsonb)'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.o404e_material_decision_cobertura_v1(jsonb)'
       ) IS NOT NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.o404e_ligar_outbox_terminal_v1()'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'estado incompatible para canon de decisión O4-04E';
    END IF;
END
$prevalidacion$;

CREATE FUNCTION
vec_contratacion_temporal.o404e_ligar_outbox_terminal_v1()
RETURNS trigger
LANGUAGE plpgsql
VOLATILE
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_payload jsonb;
    v_rama text;
BEGIN
    IF NEW.tipo_evento IN (
        'contratacion_temporal.cobertura_aplicada',
        'contratacion_temporal.cobertura_denegada_vec'
    ) THEN
        v_payload := pg_catalog.convert_from(
            NEW.payload_canonico, 'UTF8'
        )::jsonb;
        v_rama := CASE NEW.tipo_evento
            WHEN 'contratacion_temporal.cobertura_aplicada' THEN 'concedida'
            ELSE 'denegada'
        END;
        IF pg_catalog.jsonb_typeof(v_payload) <> 'object'
           OR v_payload ->> 'esquema' <>
              'vec.contratacion-temporal.evento-decision-cobertura.o4-04e.v1'
           OR v_payload ->> 'rama' <> v_rama
           OR v_payload ->> 'recibo_ref' IS NULL
           OR v_payload ->> 'auditoria_ref' IS NULL
           OR v_payload ->> 'decision_vec_ref' IS NULL THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'payload outbox O4-04E inválido';
        END IF;
        NEW.o404e_recibo_ref := v_payload ->> 'recibo_ref';
        NEW.o404e_auditoria_ref := v_payload ->> 'auditoria_ref';
        NEW.o404e_decision_vec_ref := v_payload ->> 'decision_vec_ref';
        NEW.o404e_rama := v_rama;
    ELSE
        NEW.o404e_recibo_ref := NULL;
        NEW.o404e_auditoria_ref := NULL;
        NEW.o404e_decision_vec_ref := NULL;
        NEW.o404e_rama := NULL;
    END IF;
    RETURN NEW;
EXCEPTION WHEN data_exception THEN
    RAISE EXCEPTION USING ERRCODE = '22023',
        MESSAGE = 'payload outbox O4-04E inválido';
END
$funcion$;

REVOKE ALL ON FUNCTION
    vec_contratacion_temporal.o404e_ligar_outbox_terminal_v1()
FROM PUBLIC, vec_contratacion_temporal_ejecutor,
     vec_contratacion_temporal_migrador,
     vec_contratacion_temporal_gobernador;

CREATE FUNCTION
vec_contratacion_temporal.o404e_material_decision_cobertura_v1(
    p_decision jsonb
)
RETURNS bytea
LANGUAGE plpgsql
IMMUTABLE
STRICT
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_material bytea := ''::bytea;
    v_claves text[];
    v_texto text;
    v_origen numeric;
    v_resultante numeric;
    v_decidida_en timestamptz;
    v_realizada_en timestamptz;
    v_motivo_valido boolean;
BEGIN
    IF pg_catalog.pg_column_size(p_decision) > 262144
       OR pg_catalog.jsonb_typeof(p_decision) <> 'object'
       OR pg_catalog.jsonb_typeof(p_decision -> 'canon') <> 'object'
       OR pg_catalog.jsonb_typeof(p_decision -> 'catalogo') <> 'object'
       OR pg_catalog.jsonb_typeof(p_decision -> 'politica') <> 'object'
       OR pg_catalog.jsonb_typeof(p_decision -> 'motivo') <> 'object'
       OR pg_catalog.jsonb_typeof(p_decision -> 'actuacion') <> 'object'
       OR NOT vec_contratacion_temporal.gobi_o404b_claves_exactas(
           p_decision -> 'canon',
           ARRAY['algoritmo', 'dominio', 'version_esquema']::text[]
       )
       OR NOT vec_contratacion_temporal.gobi_o404b_claves_exactas(
           p_decision -> 'catalogo',
           ARRAY['huella_sha256', 'referencia', 'version']::text[]
       )
       OR NOT vec_contratacion_temporal.gobi_o404b_claves_exactas(
           p_decision -> 'politica',
           ARRAY['huella_sha256', 'referencia', 'version']::text[]
       )
       OR NOT vec_contratacion_temporal.gobi_o404b_claves_exactas(
           p_decision -> 'motivo',
           ARRAY['clave_i18n', 'referencia_catalogo']::text[]
       )
       OR NOT vec_contratacion_temporal.gobi_o404b_claves_exactas(
           p_decision #> '{motivo,referencia_catalogo}',
           ARRAY[
               'catalogo_huella_sha256', 'catalogo_id',
               'catalogo_version', 'entrada_clave'
           ]::text[]
       )
       OR NOT vec_contratacion_temporal.gobi_o404b_claves_exactas(
           p_decision -> 'actuacion',
           ARRAY[
               'accion_clave', 'actor_ref', 'estado_destino',
               'estado_origen', 'fase_destino', 'fase_origen',
               'realizada_en', 'recibo_ref', 'secuencia',
               'unidad_ref', 'version_expediente'
           ]::text[]
       )
       OR p_decision #>> '{canon,dominio}' <>
          'vec.dipgra.contratacion-temporal.decision-cobertura-gobernada'
       OR NOT vec_contratacion_temporal.numero_entero_json_canonico_v2(
           p_decision #> '{canon,version_esquema}', 1, 1
       )
       OR p_decision #>> '{canon,algoritmo}' <> 'sha-256'
       OR (p_decision ->> 'tipo') NOT IN ('inicial', 'rectificacion') THEN
        RETURN NULL;
    END IF;

    SELECT pg_catalog.array_agg(clave ORDER BY clave COLLATE "C")
      INTO v_claves
      FROM pg_catalog.jsonb_object_keys(p_decision) AS k(clave);
    IF (
           p_decision ->> 'tipo' = 'inicial'
           AND v_claves IS DISTINCT FROM ARRAY[
               'actor_ref', 'actuacion', 'analisis_huella_sha256',
               'analisis_ref', 'canon', 'catalogo', 'decidida_en',
               'expediente_ref', 'huella_sha256', 'motivo',
               'organizacion_ref', 'perfil_ref', 'politica',
               'preparacion_evidencias_huella_sha256',
               'preparacion_evidencias_ref', 'propuesta_huella_sha256',
               'propuesta_ref', 'referencia', 'tipo',
               'version_expediente', 'version_expediente_origen',
               'via_elegida', 'via_recomendada'
           ]::text[]
       )
       OR (
           p_decision ->> 'tipo' = 'rectificacion'
           AND v_claves IS DISTINCT FROM ARRAY[
               'actor_ref', 'actuacion', 'analisis_huella_sha256',
               'analisis_ref', 'canon', 'catalogo', 'decidida_en',
               'expediente_ref', 'huella_sha256', 'motivo',
               'organizacion_ref', 'perfil_ref', 'politica',
               'predecesora_huella_sha256', 'predecesora_ref',
               'preparacion_evidencias_huella_sha256',
               'preparacion_evidencias_ref', 'propuesta_huella_sha256',
               'propuesta_ref', 'referencia', 'tipo',
               'version_expediente', 'version_expediente_origen',
               'via_elegida', 'via_recomendada'
           ]::text[]
       ) THEN
        RETURN NULL;
    END IF;

    IF (p_decision ->> 'referencia') !~
          '^decision-cobertura:sha256:[a-f0-9]{64}$'
       OR (p_decision ->> 'huella_sha256') !~ '^[a-f0-9]{64}$'
       OR p_decision ->> 'referencia' <>
          'decision-cobertura:sha256:' ||
          (p_decision ->> 'huella_sha256')
       OR (p_decision ->> 'organizacion_ref') !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR (p_decision ->> 'expediente_ref') !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR (p_decision ->> 'actor_ref') !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR (p_decision ->> 'perfil_ref') !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR (p_decision ->> 'propuesta_ref') !~
          '^propuesta-cobertura:sha256:[a-f0-9]{64}$'
       OR p_decision ->> 'propuesta_ref' <>
          'propuesta-cobertura:sha256:' ||
          (p_decision ->> 'propuesta_huella_sha256')
       OR (p_decision ->> 'preparacion_evidencias_ref') !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR (p_decision ->> 'analisis_ref') !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR (p_decision #>> '{catalogo,referencia}') !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR (p_decision #>> '{politica,referencia}') !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR (p_decision ->> 'via_elegida') !~
          '^[a-z][a-z0-9._-]{1,79}$'
       OR (p_decision ->> 'via_recomendada') !~
          '^[a-z][a-z0-9._-]{1,79}$' THEN
        RETURN NULL;
    END IF;

    FOREACH v_texto IN ARRAY ARRAY[
        p_decision ->> 'huella_sha256',
        p_decision ->> 'propuesta_huella_sha256',
        p_decision ->> 'preparacion_evidencias_huella_sha256',
        p_decision ->> 'analisis_huella_sha256',
        p_decision #>> '{catalogo,huella_sha256}',
        p_decision #>> '{politica,huella_sha256}'
    ]::text[]
    LOOP
        IF v_texto !~ '^[a-f0-9]{64}$'
           OR v_texto = pg_catalog.repeat('0', 64) THEN
            RETURN NULL;
        END IF;
    END LOOP;

    IF NOT vec_contratacion_temporal.numero_entero_json_canonico_v2(
           p_decision -> 'version_expediente_origen',
           1, 9007199254740990::numeric
       )
       OR NOT vec_contratacion_temporal.numero_entero_json_canonico_v2(
           p_decision -> 'version_expediente',
           2, 9007199254740991::numeric
       )
       OR NOT vec_contratacion_temporal.numero_entero_json_canonico_v2(
           p_decision #> '{catalogo,version}',
           1, 9007199254740991::numeric
       )
       OR NOT vec_contratacion_temporal.numero_entero_json_canonico_v2(
           p_decision #> '{politica,version}',
           1, 9007199254740991::numeric
       ) THEN
        RETURN NULL;
    END IF;
    v_origen := (p_decision ->> 'version_expediente_origen')::numeric;
    v_resultante := (p_decision ->> 'version_expediente')::numeric;
    IF v_resultante <> v_origen + 1
       OR NOT vec_contratacion_temporal.numero_entero_json_canonico_v2(
           p_decision #> '{actuacion,secuencia}',
           2, 9007199254740991::numeric
       )
       OR NOT vec_contratacion_temporal.numero_entero_json_canonico_v2(
           p_decision #> '{actuacion,version_expediente}',
           2, 9007199254740991::numeric
       )
       OR (p_decision #>> '{actuacion,secuencia}')::numeric <> v_resultante
       OR (p_decision #>> '{actuacion,version_expediente}')::numeric <>
          v_resultante THEN
        RETURN NULL;
    END IF;

    IF (p_decision #>> '{actuacion,accion_clave}') <>
          (CASE p_decision ->> 'tipo'
              WHEN 'inicial'
                  THEN 'contratacion_temporal.cobertura.decidir'
              ELSE 'contratacion_temporal.cobertura.rectificar'
          END)
       OR p_decision #>> '{actuacion,actor_ref}' <>
          p_decision ->> 'actor_ref'
       OR (p_decision #>> '{actuacion,unidad_ref}') !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR (p_decision #>> '{actuacion,fase_origen}') !~
          '^[a-z][a-z0-9._-]{1,79}$'
       OR (p_decision #>> '{actuacion,fase_destino}') !~
          '^[a-z][a-z0-9._-]{1,79}$'
       OR (p_decision #>> '{actuacion,estado_origen}') NOT IN (
           'pendiente', 'en_curso', 'espera_externa',
           'completado', 'incidencia', 'cancelado'
       )
       OR (p_decision #>> '{actuacion,estado_destino}') NOT IN (
           'pendiente', 'en_curso', 'espera_externa',
           'completado', 'incidencia', 'cancelado'
       )
       OR (p_decision #>> '{actuacion,recibo_ref}') !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR NOT vec_contratacion_temporal.gobi_o404b_instante_texto_valido(
           p_decision ->> 'decidida_en', false
       )
       OR NOT vec_contratacion_temporal.gobi_o404b_instante_texto_valido(
           p_decision #>> '{actuacion,realizada_en}', false
       ) THEN
        RETURN NULL;
    END IF;
    v_decidida_en := (p_decision ->> 'decidida_en')::timestamptz;
    v_realizada_en :=
        (p_decision #>> '{actuacion,realizada_en}')::timestamptz;
    IF v_decidida_en <> v_realizada_en THEN
        RETURN NULL;
    END IF;

    v_motivo_valido :=
        (p_decision #>> '{motivo,referencia_catalogo,catalogo_id}')
            ~ '^[a-z][a-z0-9._-]{0,127}$'
        AND vec_contratacion_temporal.numero_entero_json_canonico_v2(
            p_decision #> '{motivo,referencia_catalogo,catalogo_version}',
            1, 2147483647
        )
        AND (
            p_decision
                #>> '{motivo,referencia_catalogo,catalogo_huella_sha256}'
        ) ~ '^[a-f0-9]{64}$'
        AND (
            p_decision
                #>> '{motivo,referencia_catalogo,catalogo_huella_sha256}'
        ) <> pg_catalog.repeat('0', 64)
        AND (
            p_decision #>> '{motivo,referencia_catalogo,entrada_clave}'
        ) ~ '^[a-z][a-z0-9._-]{0,127}$'
        AND (p_decision #>> '{motivo,clave_i18n}')
            ~ '^[a-z][a-z0-9._-]{1,79}$';
    IF p_decision ->> 'tipo' = 'inicial' THEN
        IF p_decision ? 'predecesora_ref'
           OR p_decision ? 'predecesora_huella_sha256'
           OR (
               p_decision ->> 'via_elegida' =
                   p_decision ->> 'via_recomendada'
               AND (
                   p_decision #>> '{motivo,referencia_catalogo,catalogo_id}'
                       <> ''
                   OR (
                       p_decision
                           #>> '{motivo,referencia_catalogo,catalogo_version}'
                   ) <> '0'
                   OR (
                       p_decision
                           #>> '{motivo,referencia_catalogo,catalogo_huella_sha256}'
                   ) <> ''
                   OR (
                       p_decision
                           #>> '{motivo,referencia_catalogo,entrada_clave}'
                   ) <> ''
                   OR p_decision #>> '{motivo,clave_i18n}' <> ''
               )
           )
           OR (
               p_decision ->> 'via_elegida' <>
                   p_decision ->> 'via_recomendada'
               AND NOT v_motivo_valido
           ) THEN
            RETURN NULL;
        END IF;
    ELSIF NOT v_motivo_valido
       OR (p_decision ->> 'predecesora_ref') !~
          '^decision-cobertura:sha256:[a-f0-9]{64}$'
       OR (p_decision ->> 'predecesora_huella_sha256') !~
          '^[a-f0-9]{64}$'
       OR (p_decision ->> 'predecesora_huella_sha256') =
          pg_catalog.repeat('0', 64)
       OR p_decision ->> 'predecesora_ref' <>
          'decision-cobertura:sha256:' ||
          (p_decision ->> 'predecesora_huella_sha256') THEN
        RETURN NULL;
    END IF;

    v_material := vec_contratacion_temporal.gobi_o404b_texto_canon(
        v_material, p_decision #>> '{canon,dominio}'
    ) || pg_catalog.decode('0001', 'hex');
    v_material := vec_contratacion_temporal.gobi_o404b_texto_canon(
        v_material, 'sha-256'
    );
    FOREACH v_texto IN ARRAY ARRAY[
        p_decision ->> 'tipo',
        p_decision ->> 'organizacion_ref',
        p_decision ->> 'expediente_ref'
    ]::text[]
    LOOP
        v_material := vec_contratacion_temporal.gobi_o404b_texto_canon(
            v_material, v_texto
        );
    END LOOP;
    v_material := v_material
        || pg_catalog.int8send(v_origen::bigint)
        || pg_catalog.int8send(v_resultante::bigint);
    FOREACH v_texto IN ARRAY ARRAY[
        p_decision ->> 'actor_ref',
        p_decision ->> 'perfil_ref',
        p_decision ->> 'propuesta_ref',
        p_decision ->> 'propuesta_huella_sha256',
        p_decision ->> 'preparacion_evidencias_ref',
        p_decision ->> 'preparacion_evidencias_huella_sha256',
        p_decision ->> 'analisis_ref',
        p_decision ->> 'analisis_huella_sha256',
        p_decision #>> '{catalogo,referencia}'
    ]::text[]
    LOOP
        v_material := vec_contratacion_temporal.gobi_o404b_texto_canon(
            v_material, v_texto
        );
    END LOOP;
    v_material := v_material || pg_catalog.int8send(
        (p_decision #>> '{catalogo,version}')::bigint
    );
    FOREACH v_texto IN ARRAY ARRAY[
        p_decision #>> '{catalogo,huella_sha256}',
        p_decision #>> '{politica,referencia}'
    ]::text[]
    LOOP
        v_material := vec_contratacion_temporal.gobi_o404b_texto_canon(
            v_material, v_texto
        );
    END LOOP;
    v_material := v_material || pg_catalog.int8send(
        (p_decision #>> '{politica,version}')::bigint
    );
    FOREACH v_texto IN ARRAY ARRAY[
        p_decision #>> '{politica,huella_sha256}',
        p_decision ->> 'via_elegida',
        p_decision ->> 'via_recomendada',
        p_decision #>> '{motivo,referencia_catalogo,catalogo_id}'
    ]::text[]
    LOOP
        v_material := vec_contratacion_temporal.gobi_o404b_texto_canon(
            v_material, v_texto
        );
    END LOOP;
    v_material := v_material || pg_catalog.int8send(
        (p_decision
            #>> '{motivo,referencia_catalogo,catalogo_version}')::bigint
    );
    FOREACH v_texto IN ARRAY ARRAY[
        p_decision #>> '{motivo,referencia_catalogo,catalogo_huella_sha256}',
        p_decision #>> '{motivo,referencia_catalogo,entrada_clave}',
        p_decision #>> '{motivo,clave_i18n}',
        coalesce(p_decision ->> 'predecesora_ref', ''),
        coalesce(p_decision ->> 'predecesora_huella_sha256', '')
    ]::text[]
    LOOP
        v_material := vec_contratacion_temporal.gobi_o404b_texto_canon(
            v_material, v_texto
        );
    END LOOP;
    v_material := v_material || pg_catalog.int8send(
        vec_contratacion_temporal.gobi_o404b_microsegundos(v_decidida_en)
    ) || pg_catalog.int8send(
        (p_decision #>> '{actuacion,secuencia}')::bigint
    ) || pg_catalog.int8send(
        (p_decision #>> '{actuacion,version_expediente}')::bigint
    );
    FOREACH v_texto IN ARRAY ARRAY[
        p_decision #>> '{actuacion,accion_clave}',
        p_decision #>> '{actuacion,actor_ref}',
        p_decision #>> '{actuacion,unidad_ref}'
    ]::text[]
    LOOP
        v_material := vec_contratacion_temporal.gobi_o404b_texto_canon(
            v_material, v_texto
        );
    END LOOP;
    v_material := v_material || pg_catalog.int8send(
        vec_contratacion_temporal.gobi_o404b_microsegundos(v_realizada_en)
    );
    FOREACH v_texto IN ARRAY ARRAY[
        p_decision #>> '{actuacion,fase_origen}',
        p_decision #>> '{actuacion,fase_destino}',
        p_decision #>> '{actuacion,estado_origen}',
        p_decision #>> '{actuacion,estado_destino}',
        p_decision #>> '{actuacion,recibo_ref}'
    ]::text[]
    LOOP
        v_material := vec_contratacion_temporal.gobi_o404b_texto_canon(
            v_material, v_texto
        );
    END LOOP;
    IF pg_catalog.octet_length(v_material) > 262144 THEN
        RETURN NULL;
    END IF;
    RETURN v_material;
EXCEPTION
    WHEN data_exception OR invalid_text_representation
      OR datetime_field_overflow OR numeric_value_out_of_range THEN
        RETURN NULL;
END
$funcion$;

CREATE FUNCTION
vec_contratacion_temporal.o404e_decision_cobertura_exacta_v1(
    p_decision jsonb
)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
STRICT
SET search_path = pg_catalog
BEGIN ATOMIC
    SELECT vec_contratacion_temporal
               .o404e_material_decision_cobertura_v1(p_decision)
               IS NOT NULL
       AND pg_catalog.encode(
               pg_catalog.sha256(
                   vec_contratacion_temporal
                       .o404e_material_decision_cobertura_v1(p_decision)
               ), 'hex'
           ) = p_decision ->> 'huella_sha256';
END;

REVOKE ALL ON FUNCTION
    vec_contratacion_temporal.o404e_material_decision_cobertura_v1(jsonb),
    vec_contratacion_temporal.o404e_decision_cobertura_exacta_v1(jsonb)
FROM PUBLIC, vec_contratacion_temporal_ejecutor,
     vec_contratacion_temporal_migrador,
     vec_contratacion_temporal_gobernador;

UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4
   SET version_esquema = 6,
       actualizada_en = pg_catalog.date_trunc(
           'microseconds', pg_catalog.clock_timestamp()
       )
 WHERE control AND version_esquema = 5;

COMMIT;
