\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='2s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000202',0));
-- Preimagen literal instalada en el clon nominal postHZ15+CT164. La función
-- de necesidad CT193 conserva la autoridad sobre el catálogo y fechas civiles.
DO $pre$
DECLARE f oid:=to_regprocedure('vec_contratacion_temporal.expediente_analisis_valido_v2(jsonb,boolean)');
 n oid:=to_regprocedure('vec_contratacion_temporal.normalizar_agregado_dominio_analisis_v2(jsonb)');
 p pg_proc%ROWTYPE; pn pg_proc%ROWTYPE; def_h text; src_h text;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario' OR f IS NULL OR n IS NULL
    OR to_regprocedure('vec_contratacion_temporal.necesidad_alta_valida_v3(jsonb)') IS NULL THEN
  RAISE EXCEPTION 'CT202: PARO clave=dependencias actual=ausentes esperado=CT164_CT193' USING ERRCODE='55000';
 END IF;
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=f;
 def_h:=encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex');
 src_h:=encode(sha256(convert_to(p.prosrc,'UTF8')),'hex');
 IF def_h IS DISTINCT FROM '0050fe42012c943196b14bd4d1a5dad9869bb98df4c181029e5bce006c7ca240'
    OR src_h IS DISTINCT FROM 'cff2755af9a52ef2b3b145651264bb5da5cd5013b9497294aa52a0d398edf3f6'
    OR p.proowner IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
    OR p.proacl::text IS DISTINCT FROM '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario}'
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog']
    OR p.provolatile<>'i' OR p.proparallel<>'u' OR p.prosecdef OR p.proisstrict OR p.proretset THEN
  RAISE EXCEPTION 'CT202: PARO clave=preimagen esperado=postHZ15_CT164:0050fe42/cff2755a actual=def:%/src:%',def_h,src_h USING ERRCODE='55000';
 END IF;
 SELECT * INTO STRICT pn FROM pg_proc WHERE oid=n;
 def_h:=encode(sha256(convert_to(pg_get_functiondef(n),'UTF8')),'hex');
 src_h:=encode(sha256(convert_to(pn.prosrc,'UTF8')),'hex');
 IF def_h IS DISTINCT FROM '52227f7afb7cee58ecf248be6596a9d3b842c87a32262573eb1431fa6f051e9e'
    OR src_h IS DISTINCT FROM 'cf2b63ddedbbc303ced6e2f9e2e97ca2f95092aaf3c0bdcf3adb2fd2b9ca744f'
    OR pn.proowner IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
    OR pn.proacl::text IS DISTINCT FROM '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario}'
    OR pn.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog']
    OR pn.provolatile<>'i' OR pn.proparallel<>'u' OR pn.prosecdef OR NOT pn.proisstrict OR pn.proretset THEN
  RAISE EXCEPTION 'CT202: PARO clave=preimagen_normalizador esperado=postHZ15:52227f7a/cf2b63dd actual=def:%/src:%',def_h,src_h USING ERRCODE='55000';
 END IF;
END $pre$;

CREATE OR REPLACE FUNCTION vec_contratacion_temporal.expediente_analisis_valido_v2(e jsonb, p_exige_analisis boolean)
 RETURNS boolean
 LANGUAGE plpgsql
 IMMUTABLE
 SET search_path TO 'pg_catalog'
AS $function$
DECLARE
    s jsonb := e -> 'solicitud';
    rc jsonb := e #> '{solicitud,rc}';
    v_claves text[] := ARRAY[
      'actuaciones', 'actualizado_en', 'creado_en', 'estado_actual',
      'fase_actual', 'flujo', 'numero_visible', 'organizacion_ref',
      'referencia', 'solicitud', 'version'
    ]::text[];
    v_claves_solicitud text[] := ARRAY[
      'categoria_ref', 'centro_ref', 'contacto_ref', 'detalle',
      'documentos_adjuntos', 'grupo_subgrupo', 'motivo_clave',
      'periodo', 'rc'
    ]::text[];
    v_claves_rc text[];
    v_actuacion jsonb;
    v_documento jsonb;
    v_necesidad boolean;
    v_solicitud_civil jsonb;
    v_inicio_text text;
    v_fin_text text;
BEGIN
    IF vec_contratacion_temporal.circuito_agregado_valido_ct164(e) IS NOT TRUE THEN RETURN false; END IF;
    IF e ? 'circuito' THEN
        SELECT array_agg(k ORDER BY k) INTO v_claves FROM unnest(array_append(v_claves,'circuito')) k;
    END IF;
    IF p_exige_analisis IS NULL
       OR pg_catalog.jsonb_typeof(e) <> 'object'
       OR pg_catalog.jsonb_exists(e, 'via_cobertura')
       OR pg_catalog.jsonb_exists(e, 'asignacion') THEN
        RETURN false;
    END IF;
    IF p_exige_analisis THEN
        v_claves := pg_catalog.array_append(v_claves, 'analisis');
        SELECT pg_catalog.array_agg(x ORDER BY x)
          INTO v_claves
          FROM pg_catalog.unnest(v_claves) AS c(x);
    ELSIF pg_catalog.jsonb_exists(e, 'analisis') THEN
        RETURN false;
    END IF;
    v_necesidad := s ? 'necesidad';
    IF v_necesidad THEN
        v_claves_solicitud := pg_catalog.array_append(v_claves_solicitud,'necesidad');
        -- O3 conserva la proyección UTC del dominio. CT193 gobierna la fecha
        -- civil: se retroconvierte una copia estricta, nunca el agregado/HMAC.
        v_inicio_text := s #>> '{periodo,inicio}';
        IF v_inicio_text !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T00:00:00Z$'
           OR v_inicio_text IS DISTINCT FROM s #>> '{necesidad,periodo,inicio}' THEN
            RETURN false;
        END IF;
        v_solicitud_civil := pg_catalog.jsonb_set(s,'{periodo,inicio}',
            pg_catalog.to_jsonb(pg_catalog.left(v_inicio_text,10)),false);
        v_solicitud_civil := pg_catalog.jsonb_set(v_solicitud_civil,
            '{necesidad,periodo,inicio}',
            pg_catalog.to_jsonb(pg_catalog.left(v_inicio_text,10)),false);
        v_fin_text := s #>> '{periodo,fin}';
        IF v_fin_text IS NOT NULL THEN
            IF v_fin_text !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T00:00:00Z$'
               OR v_fin_text IS DISTINCT FROM s #>> '{necesidad,periodo,fin}' THEN
                RETURN false;
            END IF;
            v_solicitud_civil := pg_catalog.jsonb_set(v_solicitud_civil,
                '{periodo,fin}',pg_catalog.to_jsonb(pg_catalog.left(v_fin_text,10)),false);
            v_solicitud_civil := pg_catalog.jsonb_set(v_solicitud_civil,
                '{necesidad,periodo,fin}',
                pg_catalog.to_jsonb(pg_catalog.left(v_fin_text,10)),false);
        ELSIF s #> '{periodo,fin}' IS DISTINCT FROM
              s #> '{necesidad,periodo,fin}' THEN
            RETURN false;
        END IF;
        IF vec_contratacion_temporal.necesidad_alta_valida_v3(v_solicitud_civil) IS NOT TRUE THEN
            RETURN false;
        END IF;
    END IF;
    -- CT200 puede aún no estar instalado. Se conserva su contrato de tres
    -- campos como dato opcional sin hacer depender CT202 de esa migración.
    IF s ?| ARRAY['jornada_minutos','numero_personas','puesto_solicitado'] THEN
        IF NOT (s ?& ARRAY['jornada_minutos','numero_personas','puesto_solicitado'])
           OR NOT (CASE WHEN pg_catalog.jsonb_typeof(s->'jornada_minutos')='number'
                         AND s->>'jornada_minutos' ~ '^[0-9]+$'
                    THEN (s->>'jornada_minutos')::numeric BETWEEN 1 AND 10080
                    ELSE false END)
           OR NOT (CASE WHEN pg_catalog.jsonb_typeof(s->'numero_personas')='number'
                         AND s->>'numero_personas' ~ '^[0-9]+$'
                    THEN (s->>'numero_personas')::numeric BETWEEN 1 AND 4294967295
                    ELSE false END)
           OR pg_catalog.jsonb_typeof(s->'puesto_solicitado')<>'string'
           OR pg_catalog.length(s->>'puesto_solicitado') NOT BETWEEN 1 AND 160
           OR pg_catalog.btrim(s->>'puesto_solicitado') IS DISTINCT FROM s->>'puesto_solicitado'
           OR (s->>'puesto_solicitado') ~ '[[:cntrl:]]' THEN
            RETURN false;
        END IF;
        v_claves_solicitud := v_claves_solicitud ||
            ARRAY['jornada_minutos','numero_personas','puesto_solicitado'];
    END IF;
    IF pg_catalog.jsonb_exists(s, 'observaciones') THEN
        v_claves_solicitud :=
            pg_catalog.array_append(v_claves_solicitud, 'observaciones');
        SELECT pg_catalog.array_agg(x ORDER BY x)
          INTO v_claves_solicitud
          FROM pg_catalog.unnest(v_claves_solicitud) AS c(x);
    END IF;
    SELECT pg_catalog.array_agg(x ORDER BY x) INTO v_claves_solicitud
    FROM pg_catalog.unnest(v_claves_solicitud) AS c(x);
    v_claves_rc := CASE WHEN rc ->> 'existe' = 'true' THEN ARRAY[
      'documento_ref', 'existe', 'fecha', 'importe', 'numero'
    ]::text[] ELSE ARRAY['existe', 'fecha', 'importe']::text[] END;
    IF NOT vec_contratacion_temporal.claves_json_exactas_v1(e, v_claves)
       OR NOT vec_contratacion_temporal.claves_json_exactas_v1(
           e -> 'flujo',
           ARRAY['definicion_ref', 'huella_sha256', 'version']::text[]
       )
       OR NOT vec_contratacion_temporal.claves_json_exactas_v1(
           s, v_claves_solicitud
       )
       OR (NOT v_necesidad AND NOT vec_contratacion_temporal.periodo_previsto_analisis_valido_v1(s -> 'periodo'))
       OR NOT vec_contratacion_temporal.claves_json_exactas_v1(
           rc, v_claves_rc
       )
       OR NOT vec_contratacion_temporal.numero_entero_json_canonico_v2(
           e -> 'version', 1, 9007199254740991::numeric
       )
       OR NOT vec_contratacion_temporal.numero_entero_json_canonico_v2(
           e #> '{flujo,version}', 1, 9007199254740991::numeric
       )
       OR NOT vec_contratacion_temporal.instante_utc_json_canonico_v2(
           e -> 'creado_en', false
       )
       OR NOT vec_contratacion_temporal.instante_utc_json_canonico_v2(
           e -> 'actualizado_en', false
       )
       OR NOT vec_contratacion_temporal.instante_utc_json_canonico_v2(
           s #> '{periodo,inicio}', true
       )
       OR (s #> '{periodo,fin}' IS NOT NULL
           AND NOT vec_contratacion_temporal.instante_utc_json_canonico_v2(
             s #> '{periodo,fin}', true
           ))
       OR pg_catalog.jsonb_typeof(e -> 'referencia') <> 'string'
       OR pg_catalog.jsonb_typeof(e -> 'organizacion_ref') <> 'string'
       OR pg_catalog.jsonb_typeof(e -> 'numero_visible') <> 'string'
       OR pg_catalog.jsonb_typeof(e -> 'fase_actual') <> 'string'
       OR pg_catalog.jsonb_typeof(e -> 'estado_actual') <> 'string'
       OR pg_catalog.jsonb_typeof(e #> '{flujo,definicion_ref}') <> 'string'
       OR pg_catalog.jsonb_typeof(e #> '{flujo,huella_sha256}') <> 'string'
       OR pg_catalog.jsonb_typeof(s -> 'centro_ref') <> 'string'
       OR pg_catalog.jsonb_typeof(s -> 'contacto_ref') <> 'string'
       OR pg_catalog.jsonb_typeof(s -> 'categoria_ref') <> 'string'
       OR pg_catalog.jsonb_typeof(s -> 'grupo_subgrupo') <> 'string'
       OR pg_catalog.jsonb_typeof(s -> 'motivo_clave') <> 'string'
       OR pg_catalog.jsonb_typeof(s -> 'detalle') <> 'string'
       OR pg_catalog.jsonb_typeof(rc -> 'existe') <> 'boolean'
       OR pg_catalog.jsonb_typeof(e -> 'actuaciones') <> 'array'
       OR pg_catalog.jsonb_typeof(s -> 'documentos_adjuntos')
            NOT IN ('array', 'null')
       OR (e ->> 'actualizado_en')::timestamptz <
          (e ->> 'creado_en')::timestamptz
       OR (s #>> '{periodo,fin}')::timestamptz <
          (s #>> '{periodo,inicio}')::timestamptz THEN
        RETURN false;
    END IF;
    IF p_exige_analisis
       AND vec_contratacion_temporal.huella_analisis_derivado_v2(
               e -> 'analisis'
           ) IS NULL THEN
        RETURN false;
    END IF;
    IF rc ->> 'existe' = 'true' THEN
        IF NOT vec_contratacion_temporal.instante_utc_json_canonico_v2(
               rc -> 'fecha', true
           )
           OR NOT vec_contratacion_temporal.claves_json_exactas_v1(
               rc -> 'importe', ARRAY['centimos', 'moneda']::text[]
           )
           OR NOT vec_contratacion_temporal.numero_entero_json_canonico_v2(
               rc #> '{importe,centimos}', 1,
               9223372036854775807::numeric
           )
           OR pg_catalog.jsonb_typeof(rc -> 'numero') <> 'string'
           OR pg_catalog.jsonb_typeof(rc -> 'documento_ref') <> 'string'
           OR pg_catalog.jsonb_typeof(rc #> '{importe,moneda}') <> 'string' THEN
            RETURN false;
        END IF;
    ELSE
        IF rc ->> 'fecha' <> '0001-01-01T00:00:00Z'
           OR NOT vec_contratacion_temporal.claves_json_exactas_v1(
               rc -> 'importe', ARRAY['centimos', 'moneda']::text[]
           )
           OR NOT vec_contratacion_temporal.numero_entero_json_canonico_v2(
               rc #> '{importe,centimos}', 0, 0
           )
           OR rc #>> '{importe,moneda}' <> '' THEN
            RETURN false;
        END IF;
    END IF;
    FOR v_documento IN
        SELECT d.v
          FROM pg_catalog.jsonb_array_elements(
              CASE
                WHEN s -> 'documentos_adjuntos' = 'null'::jsonb
                THEN '[]'::jsonb
                ELSE s -> 'documentos_adjuntos'
              END
          ) AS d(v)
    LOOP
        IF pg_catalog.jsonb_typeof(v_documento) <> 'string' THEN
            RETURN false;
        END IF;
    END LOOP;
    FOR v_actuacion IN
        SELECT a.v
          FROM pg_catalog.jsonb_array_elements(e -> 'actuaciones') AS a(v)
    LOOP
        IF vec_contratacion_temporal.actuacion_analisis_valida_v2(
               v_actuacion
           ) IS NOT TRUE THEN
            RETURN false;
        END IF;
    END LOOP;
    RETURN true;
EXCEPTION
    WHEN data_exception OR datetime_field_overflow
      OR invalid_text_representation OR numeric_value_out_of_range THEN
        RETURN false;
END
$function$;

CREATE OR REPLACE FUNCTION vec_contratacion_temporal.normalizar_agregado_dominio_analisis_v2(p_agregado jsonb)
 RETURNS jsonb
 LANGUAGE plpgsql
 IMMUTABLE STRICT
 SET search_path TO 'pg_catalog'
AS $function$
DECLARE
    resultado jsonb := p_agregado;
    actuacion jsonb;
    analisis jsonb;
    indice integer;
    cantidad integer;
BEGIN
    resultado := pg_catalog.jsonb_set(
        resultado, '{creado_en}',
        pg_catalog.to_jsonb(
            vec_contratacion_temporal.texto_instante_utc_go_v2(
                resultado ->> 'creado_en'
            )
        ), false
    );
    resultado := pg_catalog.jsonb_set(
        resultado, '{actualizado_en}',
        pg_catalog.to_jsonb(
            vec_contratacion_temporal.texto_instante_utc_go_v2(
                resultado ->> 'actualizado_en'
            )
        ), false
    );
    -- O2-07 usa una forma canónica de transporte para RC ausente.
    -- Solo su valor neutro exacto se convierte a la forma del dominio Go.
    IF resultado #>> '{solicitud,rc,existe}' = 'false'
       AND resultado #>> '{solicitud,rc,fecha}' = ''
       AND resultado #>> '{solicitud,rc,importe,centimos}' = '0'
       AND resultado #>> '{solicitud,rc,importe,moneda}' = 'EUR'
       AND coalesce(resultado #>> '{solicitud,rc,numero}', '') = ''
       AND coalesce(
           resultado #>> '{solicitud,rc,documento_ref}', ''
       ) = '' THEN
        resultado := pg_catalog.jsonb_set(
            resultado, '{solicitud,rc,fecha}',
            pg_catalog.to_jsonb('0001-01-01T00:00:00Z'::text), false
        );
        resultado := pg_catalog.jsonb_set(
            resultado, '{solicitud,rc,importe,moneda}',
            pg_catalog.to_jsonb(''::text), false
        );
    END IF;
    IF resultado #>> '{solicitud,periodo,inicio}' ~
           '^[0-9]{4}-[0-9]{2}-[0-9]{2}$' THEN
        resultado := pg_catalog.jsonb_set(
            resultado, '{solicitud,periodo,inicio}',
            pg_catalog.to_jsonb(
                (resultado #>> '{solicitud,periodo,inicio}') ||
                'T00:00:00Z'
            ), false
        );
    END IF;
    IF resultado #>> '{solicitud,periodo,fin}' ~
           '^[0-9]{4}-[0-9]{2}-[0-9]{2}$' THEN
        resultado := pg_catalog.jsonb_set(
            resultado, '{solicitud,periodo,fin}',
            pg_catalog.to_jsonb(
                (resultado #>> '{solicitud,periodo,fin}') ||
                'T00:00:00Z'
            ), false
        );
    END IF;
    -- CT193 guarda la misma fecha civil también en la necesidad sellada.
    -- La comparación con la proyección Go usa UTC a medianoche en ambos
    -- lugares; el agregado persistido conserva sus bytes originales.
    IF resultado #>> '{solicitud,necesidad,periodo,inicio}' ~
           '^[0-9]{4}-[0-9]{2}-[0-9]{2}$' THEN
        resultado := pg_catalog.jsonb_set(
            resultado, '{solicitud,necesidad,periodo,inicio}',
            pg_catalog.to_jsonb(
                (resultado #>> '{solicitud,necesidad,periodo,inicio}') ||
                'T00:00:00Z'
            ), false
        );
    END IF;
    IF resultado #>> '{solicitud,necesidad,periodo,fin}' ~
           '^[0-9]{4}-[0-9]{2}-[0-9]{2}$' THEN
        resultado := pg_catalog.jsonb_set(
            resultado, '{solicitud,necesidad,periodo,fin}',
            pg_catalog.to_jsonb(
                (resultado #>> '{solicitud,necesidad,periodo,fin}') ||
                'T00:00:00Z'
            ), false
        );
    END IF;
    IF resultado #>> '{solicitud,observaciones}' = '' THEN
        resultado := resultado #- '{solicitud,observaciones}';
    END IF;
    IF resultado #>> '{solicitud,rc,numero}' = '' THEN
        resultado := resultado #- '{solicitud,rc,numero}';
    END IF;
    IF resultado #>> '{solicitud,rc,documento_ref}' = '' THEN
        resultado := resultado #- '{solicitud,rc,documento_ref}';
    END IF;
    IF resultado #> '{solicitud,documentos_adjuntos}' = '[]'::jsonb THEN
        resultado := pg_catalog.jsonb_set(
            resultado, '{solicitud,documentos_adjuntos}',
            'null'::jsonb, false
        );
    END IF;
    IF pg_catalog.jsonb_typeof(resultado -> 'actuaciones') = 'array' THEN
        cantidad := pg_catalog.jsonb_array_length(
            resultado -> 'actuaciones'
        );
        IF cantidad > 0 THEN
            FOR indice IN 0..cantidad - 1 LOOP
                actuacion := resultado #> ARRAY[
                    'actuaciones', indice::text
                ];
                IF actuacion ->> 'observaciones' = '' THEN
                    actuacion := actuacion - 'observaciones';
                END IF;
                IF actuacion -> 'documentos_ref' = '[]'::jsonb THEN
                    actuacion := actuacion - 'documentos_ref';
                END IF;
                actuacion := pg_catalog.jsonb_set(
                    actuacion, '{realizada_en}',
                    pg_catalog.to_jsonb(
                        vec_contratacion_temporal.texto_instante_utc_go_v2(
                            actuacion ->> 'realizada_en'
                        )
                    ), false
                );
                resultado := pg_catalog.jsonb_set(
                    resultado,
                    ARRAY['actuaciones', indice::text],
                    actuacion,
                    false
                );
            END LOOP;
        END IF;
    END IF;
    IF pg_catalog.jsonb_typeof(resultado -> 'analisis') = 'object' THEN
        analisis := resultado -> 'analisis';
        analisis := pg_catalog.jsonb_set(
            analisis, '{validacion_rc,validada_en}',
            pg_catalog.to_jsonb(
                vec_contratacion_temporal.texto_instante_utc_go_v2(
                    analisis #>> '{validacion_rc,validada_en}'
                )
            ), false
        );
        resultado := pg_catalog.jsonb_set(
            resultado, '{analisis}', analisis, false
        );
    END IF;
    RETURN resultado;
END
$function$;

DO $post$
DECLARE f oid:=to_regprocedure('vec_contratacion_temporal.expediente_analisis_valido_v2(jsonb,boolean)');
 n oid:=to_regprocedure('vec_contratacion_temporal.normalizar_agregado_dominio_analisis_v2(jsonb)');
 p pg_proc%ROWTYPE; pn pg_proc%ROWTYPE; def_h text; src_h text;
BEGIN
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=f;
 def_h:=encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex');
 src_h:=encode(sha256(convert_to(p.prosrc,'UTF8')),'hex');
 IF p.proowner IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
    OR def_h IS DISTINCT FROM '9375556f82f1ed8dc61249ada052e2f7ea77744bd62b994684743b0eae3a7fc1'
    OR src_h IS DISTINCT FROM '451559f07243678531858c2968e13b191e739870902f678c8f80eaa92560407c'
    OR p.proacl::text IS DISTINCT FROM '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario}'
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog']
    OR p.provolatile<>'i' OR p.proparallel<>'u' OR p.prosecdef OR p.proisstrict OR p.proretset
    OR strpos(p.prosrc,'v_solicitud_civil')=0 THEN
  RAISE EXCEPTION 'CT202: PARO clave=postimagen esperado=validacion_native_sin_acl_nueva actual=def:%/src:%',def_h,src_h USING ERRCODE='55000';
 END IF;
 SELECT * INTO STRICT pn FROM pg_proc WHERE oid=n;
 def_h:=encode(sha256(convert_to(pg_get_functiondef(n),'UTF8')),'hex');
 src_h:=encode(sha256(convert_to(pn.prosrc,'UTF8')),'hex');
 IF def_h IS DISTINCT FROM 'ac6c9964bee22ed4cc04d7ddd852c8372a75c9b5bf970f77f7534de053031d86'
    OR src_h IS DISTINCT FROM '5c20e4afdcfb846cad161c80c2b4a588fb29262ea971457eab5829e3f7e43178'
    OR pn.proowner IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
    OR pn.proacl::text IS DISTINCT FROM '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario}'
    OR pn.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog']
    OR pn.provolatile<>'i' OR pn.proparallel<>'u' OR pn.prosecdef OR NOT pn.proisstrict OR pn.proretset
    OR strpos(pn.prosrc,'solicitud,necesidad,periodo,inicio')=0 THEN
  RAISE EXCEPTION 'CT202: PARO clave=postimagen_normalizador esperado=fechas_nativas_sin_acl_nueva actual=divergente' USING ERRCODE='55000';
 END IF;
END $post$;
COMMIT;
