-- O4-04E/5: canones cruzados y validadores privados de la función exterior.
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
 WHERE control AND version_esquema = 7
 FOR UPDATE;

DO $prevalidacion$
BEGIN
    IF NOT EXISTS (
           SELECT 1
             FROM vec_contratacion_temporal.control_migracion_cobertura_o4
            WHERE control AND version_esquema = 7
       )
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.o404e_material_prueba_denegacion_v1(jsonb)'
       ) IS NOT NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.o404e_construir_lote_c1_v1(jsonb,text)'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para validadores O4-04E';
    END IF;
END
$prevalidacion$;

CREATE FUNCTION
vec_contratacion_temporal.o404e_claves_exactas_v1(
  p_valor jsonb,p_claves text[]
)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
STRICT
SET search_path=pg_catalog
AS $funcion$
  SELECT pg_catalog.jsonb_typeof(p_valor)='object'
    AND (SELECT pg_catalog.array_agg(k ORDER BY k COLLATE "C")
           FROM pg_catalog.jsonb_object_keys(p_valor) k)=p_claves
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.o404e_texto_v1(
    p_material bytea,
    p_texto text
)
RETURNS bytea
LANGUAGE sql
IMMUTABLE
STRICT
SET search_path = pg_catalog
AS $funcion$
    SELECT p_material || pg_catalog.int4send(
               pg_catalog.octet_length(p_texto)
           ) || pg_catalog.convert_to(p_texto, 'UTF8')
     WHERE pg_catalog.octet_length(p_texto) <= 65536
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.o404e_mapa_v1(
    p_material bytea,
    p_mapa jsonb
)
RETURNS bytea
LANGUAGE plpgsql
IMMUTABLE
STRICT
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_material bytea := p_material;
    v_par record;
    v_total bigint;
BEGIN
    IF pg_catalog.jsonb_typeof(p_mapa) <> 'object' THEN
        RETURN NULL;
    END IF;
    SELECT pg_catalog.count(*) INTO v_total
      FROM pg_catalog.jsonb_object_keys(p_mapa);
    IF v_total > 512 THEN
        RETURN NULL;
    END IF;
    v_material := v_material ||
        pg_catalog.int8send(v_total);
    FOR v_par IN
        SELECT clave, valor
          FROM pg_catalog.jsonb_each_text(p_mapa) e(clave, valor)
         ORDER BY clave COLLATE "C"
    LOOP
        IF v_par.clave !~ '^[a-z][a-z0-9._-]{0,127}$'
           OR pg_catalog.octet_length(v_par.valor) > 4096 THEN
            RETURN NULL;
        END IF;
        v_material := vec_contratacion_temporal.o404e_texto_v1(
            v_material, v_par.clave
        );
        v_material := vec_contratacion_temporal.o404e_texto_v1(
            v_material, v_par.valor
        );
    END LOOP;
    RETURN v_material;
END
$funcion$;

-- Reproduce calcularHuellaPruebaDenegacionOperacionDecisionCobertura.
CREATE FUNCTION
vec_contratacion_temporal.o404e_material_prueba_denegacion_v1(
    p_carga jsonb
)
RETURNS bytea
LANGUAGE plpgsql
IMMUTABLE
STRICT
SET search_path = pg_catalog
AS $funcion$
DECLARE
    c jsonb := p_carga -> 'cabecera';
    d jsonb := p_carga -> 'denegacion';
    v_material bytea := ''::bytea;
    v_texto text;
BEGIN
    IF pg_catalog.jsonb_typeof(c) <> 'object'
       OR pg_catalog.jsonb_typeof(d) <> 'object'
       OR NOT vec_contratacion_temporal.o404e_claves_exactas_v1(
           d, ARRAY[
             'accion_vec','actor_ref','ambitos','atributos','auditoria_ref',
             'correlacion_vec_ref','decision_vec_ref','expediente_ref',
             'finalidad_vec','limite_preparacion','motivo_vec',
             'organizacion_ref','perfil_ref','prueba_huella_sha256',
             'recibo_ref','recurso_huella_sha256','recurso_modulo',
             'recurso_ref','recurso_tipo','reserva_ref','revision_cercado',
             'valida_hasta','version_expediente'
           ]::text[]
       )
       OR NOT vec_contratacion_temporal.o404e_claves_exactas_v1(
           d -> 'motivo_vec',
           ARRAY[
             'catalogo_huella_sha256','catalogo_id',
             'catalogo_version','entrada_clave'
           ]::text[]
       ) THEN
        RETURN NULL;
    END IF;
    v_material := vec_contratacion_temporal.o404e_texto_v1(
        v_material,
        'VEC-CT-PRUEBA-DENEGACION-DECISION-COBERTURA-C3-V1'
    );
    FOREACH v_texto IN ARRAY ARRAY[
        c ->> 'organizacion_ref', c ->> 'expediente_ref'
    ]::text[] LOOP
        v_material := vec_contratacion_temporal.o404e_texto_v1(
            v_material, v_texto
        );
    END LOOP;
    v_material := v_material ||
        pg_catalog.int8send((c ->> 'version_expediente')::bigint);
    FOREACH v_texto IN ARRAY ARRAY[
        c ->> 'ambito_idempotencia_hmac',
        c ->> 'huella_semantica_hmac',
        c ->> 'token_propietario_sha256',
        c ->> 'reserva_ref', c ->> 'recibo_ref', c ->> 'actuacion_ref',
        c ->> 'auditoria_ref', c ->> 'evento_ref',
        c ->> 'correlacion_vec_ref', c ->> 'decision_vec_ref'
    ]::text[] LOOP
        v_material := vec_contratacion_temporal.o404e_texto_v1(
            v_material, v_texto
        );
    END LOOP;
    v_material := v_material ||
        pg_catalog.int8send((c ->> 'revision_cercado_anterior')::bigint) ||
        pg_catalog.int8send((c ->> 'revision_cercado')::bigint);
    FOREACH v_texto IN ARRAY ARRAY[
        c ->> 'observada_en_db', c ->> 'propiedad_hasta',
        c ->> 'huella_orden_sha256',
        d ->> 'recurso_ref', d ->> 'recurso_modulo', d ->> 'recurso_tipo'
    ]::text[] LOOP
        v_material := vec_contratacion_temporal.o404e_texto_v1(
            v_material, v_texto
        );
    END LOOP;
    v_material := vec_contratacion_temporal.o404e_mapa_v1(
        v_material, d -> 'ambitos'
    );
    v_material := vec_contratacion_temporal.o404e_mapa_v1(
        v_material, d -> 'atributos'
    );
    FOREACH v_texto IN ARRAY ARRAY[
        d ->> 'actor_ref', d ->> 'perfil_ref',
        d ->> 'accion_vec', d ->> 'finalidad_vec',
        d #>> '{motivo_vec,catalogo_id}'
    ]::text[] LOOP
        v_material := vec_contratacion_temporal.o404e_texto_v1(
            v_material, v_texto
        );
    END LOOP;
    v_material := v_material || pg_catalog.int8send(
        (d #>> '{motivo_vec,catalogo_version}')::bigint
    );
    FOREACH v_texto IN ARRAY ARRAY[
        d #>> '{motivo_vec,catalogo_huella_sha256}',
        d #>> '{motivo_vec,entrada_clave}',
        d ->> 'limite_preparacion'
    ]::text[] LOOP
        v_material := vec_contratacion_temporal.o404e_texto_v1(
            v_material, v_texto
        );
    END LOOP;
    RETURN v_material;
EXCEPTION
    WHEN data_exception OR invalid_text_representation
      OR numeric_value_out_of_range THEN
        RETURN NULL;
END
$funcion$;

-- Reproduce identidadOrdenesC1ConfirmacionOperacionDecisionCobertura.
-- Las tres coordenadas comprobacion_* son autoridad nominal necesaria.
CREATE FUNCTION
vec_contratacion_temporal.o404e_huella_ordenes_c1_v1(
    p_consumos jsonb
)
RETURNS text
LANGUAGE plpgsql
IMMUTABLE
STRICT
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_acumulada bytea;
    v_material bytea;
    v_consumo record;
    v_texto text;
    v_total integer;
BEGIN
    IF pg_catalog.jsonb_typeof(p_consumos) <> 'array'
       OR pg_catalog.jsonb_array_length(p_consumos) NOT BETWEEN 1 AND 512 THEN
        RETURN NULL;
    END IF;
    v_total := pg_catalog.jsonb_array_length(p_consumos);
    v_acumulada := pg_catalog.int8send(v_total);
    FOR v_consumo IN
        SELECT valor, ordinalidad
          FROM pg_catalog.jsonb_array_elements(p_consumos)
               WITH ORDINALITY e(valor, ordinalidad)
         ORDER BY ordinalidad
    LOOP
        IF NOT vec_contratacion_temporal.o404e_claves_exactas_v1(
               v_consumo.valor,
               ARRAY[
                 'autoridad_ref','catalogo_huella_sha256','catalogo_ref',
                 'catalogo_version','categoria_ref','comprobacion_clave',
                 'comprobacion_evaluada_en','comprobacion_fuente_ref',
                 'comprobacion_recibo_ref','comprobacion_resultado',
                 'definicion_fuente_ref','emitida_en','expediente_ref',
                 'generacion','huella_peticion_sha256',
                 'huella_respuesta_sha256','huella_resultado_sha256',
                 'obligatoria','orden_comprobacion','organizacion_ref',
                 'periodo','peticion_ref','posicion','procedencia_clave',
                 'pruebas_canonicas','publicador_catalogo_ref',
                 'recibo_respuesta_ref','solicitada_en','total',
                 'valida_hasta','verificador_ref','version_expediente',
                 'via_clave'
               ]::text[]
           )
           OR NOT vec_contratacion_temporal.o404e_claves_exactas_v1(
               v_consumo.valor -> 'periodo',
               ARRAY['fin','inicio']::text[]
           )
           OR NOT vec_contratacion_temporal.o404e_claves_exactas_v1(
               v_consumo.valor -> 'pruebas_canonicas',
               ARRAY[
                 'atestacion_hex','catalogo_hex',
                 'confirmacion_tcb_hex','peticion_hex','resultado_hex',
                 'resumen_hex','verificador_hex'
               ]::text[]
           )
           OR (v_consumo.valor ->> 'posicion')::integer <>
               v_consumo.ordinalidad
           OR (v_consumo.valor ->> 'total')::integer <> v_total
           OR NOT vec_contratacion_temporal.gobi_o404b_instante_texto_valido(
               v_consumo.valor ->> 'comprobacion_evaluada_en', false
           ) THEN
            RETURN NULL;
        END IF;
        v_material := vec_contratacion_temporal.o404e_texto_v1(
            ''::bytea,
            'VEC-CT-IDENTIDAD-ORDEN-C1-CONFIRMACION-C3-V1'
        );
        FOREACH v_texto IN ARRAY ARRAY[
            v_consumo.valor ->> 'peticion_ref',
            v_consumo.valor ->> 'organizacion_ref',
            v_consumo.valor ->> 'expediente_ref'
        ]::text[] LOOP
            v_material := vec_contratacion_temporal.o404e_texto_v1(
                v_material, v_texto
            );
        END LOOP;
        v_material := v_material || pg_catalog.int8send(
            (v_consumo.valor ->> 'version_expediente')::bigint
        );
        v_material := vec_contratacion_temporal.o404e_texto_v1(
            v_material, v_consumo.valor ->> 'catalogo_ref'
        ) || pg_catalog.int8send(
            (v_consumo.valor ->> 'catalogo_version')::bigint
        );
        FOREACH v_texto IN ARRAY ARRAY[
            v_consumo.valor ->> 'catalogo_huella_sha256',
            v_consumo.valor ->> 'via_clave'
        ]::text[] LOOP
            v_material := vec_contratacion_temporal.o404e_texto_v1(
                v_material, v_texto
            );
        END LOOP;
        v_material := v_material || pg_catalog.int8send(
            (v_consumo.valor ->> 'orden_comprobacion')::bigint
        );
        v_material := vec_contratacion_temporal.o404e_texto_v1(
            v_material,
            CASE WHEN (v_consumo.valor ->> 'obligatoria')::boolean
                 THEN 'obligatoria' ELSE 'opcional' END
        );
        FOREACH v_texto IN ARRAY ARRAY[
            v_consumo.valor ->> 'comprobacion_clave',
            v_consumo.valor ->> 'comprobacion_resultado',
            v_consumo.valor ->> 'comprobacion_fuente_ref',
            v_consumo.valor ->> 'comprobacion_recibo_ref',
            v_consumo.valor ->> 'comprobacion_evaluada_en',
            v_consumo.valor ->> 'procedencia_clave',
            v_consumo.valor ->> 'definicion_fuente_ref',
            v_consumo.valor ->> 'categoria_ref',
            v_consumo.valor #>> '{periodo,inicio}',
            v_consumo.valor #>> '{periodo,fin}',
            v_consumo.valor ->> 'solicitada_en',
            v_consumo.valor ->> 'emitida_en',
            v_consumo.valor ->> 'valida_hasta',
            v_consumo.valor ->> 'huella_peticion_sha256',
            v_consumo.valor ->> 'huella_resultado_sha256',
            v_consumo.valor ->> 'huella_respuesta_sha256',
            v_consumo.valor ->> 'autoridad_ref'
        ]::text[] LOOP
            v_material := vec_contratacion_temporal.o404e_texto_v1(
                v_material, v_texto
            );
        END LOOP;
        v_material := v_material || pg_catalog.int8send(
            (v_consumo.valor ->> 'generacion')::bigint
        );
        FOREACH v_texto IN ARRAY ARRAY[
            v_consumo.valor ->> 'recibo_respuesta_ref',
            v_consumo.valor ->> 'verificador_ref',
            v_consumo.valor ->> 'publicador_catalogo_ref'
        ]::text[] LOOP
            v_material := vec_contratacion_temporal.o404e_texto_v1(
                v_material, v_texto
            );
        END LOOP;
        v_acumulada := v_acumulada || pg_catalog.sha256(v_material);
    END LOOP;
    RETURN pg_catalog.encode(pg_catalog.sha256(v_acumulada), 'hex');
EXCEPTION
    WHEN data_exception OR invalid_text_representation
      OR datetime_field_overflow OR numeric_value_out_of_range THEN
        RETURN NULL;
END
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.o404e_construir_lote_c1_v1(
    p_carga jsonb,
    p_raiz text
)
RETURNS jsonb
LANGUAGE plpgsql
IMMUTABLE
STRICT
SET search_path = pg_catalog
AS $funcion$
DECLARE
    c jsonb := p_carga -> 'cabecera';
    g jsonb := p_carga -> 'gobierno';
    x jsonb;
    e jsonb;
    v_evidencias jsonb := '[]'::jsonb;
    v_lote jsonb;
    v_huella text;
BEGIN
    IF vec_contratacion_temporal.o404e_huella_ordenes_c1_v1(
           p_carga -> 'consumos_c1'
       ) IS DISTINCT FROM c ->> 'huella_ordenes_consumo_c1_sha256' THEN
        RETURN NULL;
    END IF;
    FOR x IN SELECT value
               FROM pg_catalog.jsonb_array_elements(
                   p_carga -> 'consumos_c1'
               )
    LOOP
        e := pg_catalog.jsonb_build_object(
          'posicion',x->'posicion','total',x->'total',
          'peticion_ref',x->'peticion_ref',
          'organizacion_ref',x->'organizacion_ref',
          'expediente_ref',x->'expediente_ref',
          'version_expediente',x->'version_expediente',
          'catalogo_ref',x->'catalogo_ref',
          'catalogo_version',x->'catalogo_version',
          'catalogo_huella_sha256',x->'catalogo_huella_sha256',
          'via_clave',x->'via_clave',
          'comprobacion_clave',x->'comprobacion_clave',
          'comprobacion_resultado',x->'comprobacion_resultado',
          'orden_comprobacion',x->'orden_comprobacion',
          'comprobacion_obligatoria',x->'obligatoria',
          'procedencia_clave',x->'procedencia_clave',
          'definicion_fuente_ref',x->'definicion_fuente_ref',
          'categoria_ref',x->'categoria_ref',
          'periodo_inicio',x#>'{periodo,inicio}',
          'periodo_fin',x#>'{periodo,fin}',
          'solicitada_en',x->'solicitada_en',
          'emitida_en',x->'emitida_en','valida_hasta',x->'valida_hasta',
          'huella_peticion_sha256',x->'huella_peticion_sha256',
          'huella_resultado_sha256',x->'huella_resultado_sha256',
          'huella_respuesta_sha256',x->'huella_respuesta_sha256',
          'autoridad_ref',x->'autoridad_ref','generacion',x->'generacion',
          'recibo_respuesta_ref',x->'recibo_respuesta_ref',
          'verificador_ref',x->'verificador_ref',
          'publicador_catalogo_ref',x->'publicador_catalogo_ref',
          'peticion_canon_hex',x#>'{pruebas_canonicas,peticion_hex}',
          'resultado_canon_hex',x#>'{pruebas_canonicas,resultado_hex}',
          'atestacion_canon_hex',x#>'{pruebas_canonicas,atestacion_hex}',
          'confirmacion_tcb_canon_hex',
              x#>'{pruebas_canonicas,confirmacion_tcb_hex}',
          'catalogo_canon_hex',x#>'{pruebas_canonicas,catalogo_hex}',
          'verificador_canon_hex',x#>'{pruebas_canonicas,verificador_hex}',
          'resumen_canon_hex',x#>'{pruebas_canonicas,resumen_hex}',
          'evidencia_huella_sha256',pg_catalog.repeat('f',64)
        );
        v_huella := pg_catalog.encode(pg_catalog.sha256(
            vec_contratacion_temporal.o404d_material_evidencia_v1(e)
        ), 'hex');
        e := pg_catalog.jsonb_set(
            e, '{evidencia_huella_sha256}', pg_catalog.to_jsonb(v_huella)
        );
        IF vec_contratacion_temporal.o404d_material_evidencia_v1(e)
               IS NULL THEN
            RETURN NULL;
        END IF;
        v_evidencias := v_evidencias || pg_catalog.jsonb_build_array(e);
    END LOOP;
    v_lote := pg_catalog.jsonb_build_object(
      'esquema','vec.contratacion-temporal.consumo-c1.o4-04d.v1',
      'lote_ref',vec_contratacion_temporal.o404c_referencia_derivada_v1(
          'lote-c1-o404e:',p_raiz
      ),
      'lote_huella_sha256',pg_catalog.repeat('f',64),
      'organizacion_ref',c->'organizacion_ref',
      'expediente_ref',c->'expediente_ref',
      'version_expediente',c->'version_expediente',
      'reserva_ref',c->'reserva_ref',
      'preparacion_c1_ref',c->'preparacion_c1_ref',
      'preparacion_c1_huella_sha256',c->'preparacion_c1_huella_sha256',
      'huella_orden_sha256',c->'huella_orden_sha256',
      'huella_ordenes_c1_sha256',c->'huella_ordenes_consumo_c1_sha256',
      'decision_vec_ref',c->'decision_vec_ref',
      'correlacion_vec_ref',c->'correlacion_vec_ref',
      'catalogo_ref',g#>'{catalogo,referencia}',
      'catalogo_version',g#>'{catalogo,version}',
      'catalogo_huella_sha256',g#>'{catalogo,huella_sha256}',
      'efecto_en',p_carga#>'{concesion,efecto_en}',
      'evidencias',v_evidencias
    );
    v_huella := pg_catalog.encode(pg_catalog.sha256(
        vec_contratacion_temporal.o404d_material_lote_v1(v_lote)
    ), 'hex');
    v_lote := pg_catalog.jsonb_set(
        v_lote, '{lote_huella_sha256}', pg_catalog.to_jsonb(v_huella)
    );
    IF vec_contratacion_temporal.o404d_material_lote_v1(v_lote) IS NULL THEN
        RETURN NULL;
    END IF;
    RETURN v_lote;
EXCEPTION
    WHEN data_exception OR invalid_text_representation
      OR datetime_field_overflow OR numeric_value_out_of_range THEN
        RETURN NULL;
END
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.o404e_transicion_exacta_v1(
    p_anterior jsonb,
    p_siguiente jsonb,
    p_propuesta jsonb,
    p_motivo jsonb,
    p_efecto timestamptz
)
RETURNS boolean
LANGUAGE plpgsql
IMMUTABLE
STRICT
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_decision jsonb;
    v_actuacion jsonb;
    v_num_decisiones integer;
    v_num_actuaciones integer;
BEGIN
    IF pg_catalog.jsonb_typeof(p_anterior) <> 'object'
       OR pg_catalog.jsonb_typeof(p_siguiente) <> 'object'
       OR vec_contratacion_temporal.o404e_propuesta_cobertura_exacta_v1(
              p_propuesta
          ) IS NOT TRUE
       OR pg_catalog.jsonb_typeof(
              p_siguiente -> 'decisiones_cobertura'
          ) <> 'array'
       OR pg_catalog.jsonb_typeof(p_siguiente -> 'actuaciones') <> 'array'
       OR pg_catalog.jsonb_typeof(p_siguiente -> 'via_cobertura') <> 'object'
       OR (p_siguiente ->> 'version')::numeric <>
          (p_anterior ->> 'version')::numeric + 1
       OR p_siguiente ->> 'referencia' <>
          p_anterior ->> 'referencia'
       OR p_siguiente ->> 'organizacion_ref' <>
          p_anterior ->> 'organizacion_ref'
       OR (p_siguiente - ARRAY[
             'version','actualizado_en','fase_actual','estado_actual',
             'via_cobertura','decisiones_cobertura','actuaciones'
           ]::text[]) <>
          (p_anterior - ARRAY[
             'version','actualizado_en','fase_actual','estado_actual',
             'via_cobertura','decisiones_cobertura','actuaciones'
           ]::text[]) THEN
        RETURN false;
    END IF;
    v_num_decisiones :=
        pg_catalog.jsonb_array_length(p_siguiente -> 'decisiones_cobertura');
    v_num_actuaciones :=
        pg_catalog.jsonb_array_length(p_siguiente -> 'actuaciones');
    IF v_num_decisiones NOT BETWEEN 1 AND 64
       OR v_num_decisiones <>
          coalesce(pg_catalog.jsonb_array_length(
              p_anterior -> 'decisiones_cobertura'
          ), 0) + 1
       OR v_num_actuaciones <>
          pg_catalog.jsonb_array_length(p_anterior -> 'actuaciones') + 1
       OR (p_siguiente -> 'decisiones_cobertura') - (v_num_decisiones - 1) <>
          coalesce(p_anterior -> 'decisiones_cobertura', '[]'::jsonb)
       OR (p_siguiente -> 'actuaciones') - (v_num_actuaciones - 1) <>
          p_anterior -> 'actuaciones' THEN
        RETURN false;
    END IF;
    v_decision :=
        p_siguiente #> ARRAY['decisiones_cobertura',(v_num_decisiones-1)::text];
    v_actuacion :=
        p_siguiente #> ARRAY['actuaciones',(v_num_actuaciones-1)::text];
    RETURN
        vec_contratacion_temporal.o404e_decision_cobertura_exacta_v1(
            v_decision
        )
        AND vec_contratacion_temporal.o404e_claves_exactas_v1(
            v_actuacion, ARRAY[
              'accion_clave','actor_ref','estado_destino','estado_origen',
              'fase_destino','fase_origen','realizada_en','recibo_ref',
              'secuencia','unidad_ref','version_expediente'
            ]::text[]
        )
        AND v_decision ->> 'propuesta_ref' = p_propuesta ->> 'referencia'
        AND v_decision ->> 'propuesta_huella_sha256' =
            p_propuesta ->> 'huella_sha256'
        AND v_decision -> 'motivo' = p_motivo
        AND v_decision ->> 'decidida_en' =
            vec_contratacion_temporal.texto_instante_utc_go_v2(
                p_efecto::text
            )
        AND p_siguiente ->> 'actualizado_en' =
            vec_contratacion_temporal.texto_instante_utc_go_v2(
                p_efecto::text
            )
        AND p_siguiente -> 'via_cobertura' =
            pg_catalog.jsonb_build_object(
                'via_clave',v_decision->'via_elegida',
                'decision_gobernada',v_decision
            )
        AND v_actuacion ->> 'secuencia' =
            p_siguiente ->> 'version'
        AND v_actuacion ->> 'version_expediente' =
            p_siguiente ->> 'version'
        AND v_actuacion ->> 'recibo_ref' =
            v_decision #>> '{actuacion,recibo_ref}'
        AND v_actuacion ->> 'realizada_en' =
            v_decision #>> '{actuacion,realizada_en}'
        AND v_actuacion ->> 'accion_clave' =
            v_decision #>> '{actuacion,accion_clave}'
        AND v_actuacion ->> 'actor_ref' =
            v_decision #>> '{actuacion,actor_ref}'
        AND v_actuacion ->> 'unidad_ref' =
            v_decision #>> '{actuacion,unidad_ref}'
        AND v_actuacion ->> 'fase_origen' =
            v_decision #>> '{actuacion,fase_origen}'
        AND v_actuacion ->> 'fase_destino' =
            v_decision #>> '{actuacion,fase_destino}'
        AND v_actuacion ->> 'estado_origen' =
            v_decision #>> '{actuacion,estado_origen}'
        AND v_actuacion ->> 'estado_destino' =
            v_decision #>> '{actuacion,estado_destino}'
        AND p_siguiente ->> 'fase_actual' =
            v_actuacion ->> 'fase_destino'
        AND p_siguiente ->> 'estado_actual' =
            v_actuacion ->> 'estado_destino'
        AND v_actuacion ->> 'fase_origen' =
            p_anterior ->> 'fase_actual'
        AND v_actuacion ->> 'estado_origen' =
            p_anterior ->> 'estado_actual';
EXCEPTION
    WHEN data_exception OR invalid_text_representation
      OR datetime_field_overflow OR numeric_value_out_of_range THEN
        RETURN false;
END
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.o404e_material_recibo_v1(
    p_recibo jsonb
)
RETURNS bytea
LANGUAGE plpgsql
IMMUTABLE
STRICT
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_material bytea := ''::bytea;
    v_clave text;
BEGIN
    IF NOT vec_contratacion_temporal.o404e_claves_exactas_v1(
        p_recibo, ARRAY[
          'actuacion_ref','ambito_idempotencia_hmac','aplicada',
          'auditoria_ref','codigo_probatorio_vec','concedida_vec',
          'confirmada_en','correlacion_vec_ref',
          'decision_cobertura_huella_sha256','decision_cobertura_ref',
          'decision_vec_huella_sha256','decision_vec_ref','denegada_vec',
          'esquema','evento_ref','huella_semantica_hmac','recibo_ref',
          'reserva_ref','revision_cercado','version_resultante'
        ]::text[]
    ) THEN
        RETURN NULL;
    END IF;
    v_material := vec_contratacion_temporal.o404e_texto_v1(
        v_material, 'VEC-CT-RECIBO-DECISION-COBERTURA-O4-04E-V1'
    );
    FOREACH v_clave IN ARRAY ARRAY[
      'esquema','recibo_ref','reserva_ref','auditoria_ref',
      'correlacion_vec_ref','decision_vec_ref',
      'decision_vec_huella_sha256','codigo_probatorio_vec',
      'ambito_idempotencia_hmac','huella_semantica_hmac',
      'confirmada_en','decision_cobertura_ref',
      'decision_cobertura_huella_sha256','evento_ref','actuacion_ref'
    ]::text[] LOOP
        v_material := vec_contratacion_temporal.o404e_texto_v1(
            v_material, p_recibo ->> v_clave
        );
    END LOOP;
    RETURN v_material
        || pg_catalog.int8send((p_recibo ->> 'revision_cercado')::bigint)
        || CASE WHEN (p_recibo ->> 'concedida_vec')::boolean
                THEN E'\\x01'::bytea ELSE E'\\x00'::bytea END
        || CASE WHEN (p_recibo ->> 'aplicada')::boolean
                THEN E'\\x01'::bytea ELSE E'\\x00'::bytea END
        || CASE WHEN (p_recibo ->> 'denegada_vec')::boolean
                THEN E'\\x01'::bytea ELSE E'\\x00'::bytea END
        || pg_catalog.int8send(
            (p_recibo ->> 'version_resultante')::bigint
        );
EXCEPTION
    WHEN data_exception OR invalid_text_representation
      OR numeric_value_out_of_range THEN
        RETURN NULL;
END
$funcion$;

UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4
   SET version_esquema = 8,
       actualizada_en = pg_catalog.date_trunc(
           'microseconds', pg_catalog.clock_timestamp()
       )
 WHERE control AND version_esquema = 7;

REVOKE ALL ON FUNCTION
    vec_contratacion_temporal.o404e_claves_exactas_v1(jsonb,text[]),
    vec_contratacion_temporal.o404e_texto_v1(bytea,text),
    vec_contratacion_temporal.o404e_mapa_v1(bytea,jsonb),
    vec_contratacion_temporal.o404e_material_prueba_denegacion_v1(jsonb),
    vec_contratacion_temporal.o404e_huella_ordenes_c1_v1(jsonb),
    vec_contratacion_temporal.o404e_construir_lote_c1_v1(jsonb,text),
    vec_contratacion_temporal.o404e_transicion_exacta_v1(
        jsonb,jsonb,jsonb,jsonb,timestamptz
    ),
    vec_contratacion_temporal.o404e_material_recibo_v1(jsonb)
FROM PUBLIC, vec_contratacion_temporal_ejecutor,
     vec_contratacion_temporal_confirmador_cobertura,
     vec_contratacion_temporal_migrador,
     vec_contratacion_temporal_gobernador;

COMMIT;
