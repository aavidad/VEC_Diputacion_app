\set ON_ERROR_STOP on
-- CT166: C1 conserva el periodo y la política de fin del análisis original.
-- Las filas y los cánones anteriores no se reinterpretan ni se reescriben.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_contratacion_temporal:o4_04:migraciones', 0));
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_contratacion_temporal:000166:periodo_fin_c1', 0));

DO $precondicion$
BEGIN
 IF pg_catalog.to_regprocedure('vec_contratacion_temporal.periodo_previsto_analisis_valido_v1(jsonb)') IS NULL
    OR pg_catalog.to_regprocedure('vec_contratacion_temporal.o404d_material_evidencia_v1(jsonb)') IS NULL
    OR pg_catalog.to_regprocedure('vec_contratacion_temporal.o404e_material_propuesta_cobertura_v1(jsonb)') IS NULL
    OR pg_catalog.to_regprocedure('vec_contratacion_temporal.o404e_huella_ordenes_c1_v1(jsonb)') IS NULL
    OR pg_catalog.to_regprocedure('vec_contratacion_temporal.o404e_semantica_propuesta_v1(jsonb)') IS NULL
    OR pg_catalog.to_regclass('vec_contratacion_temporal.consumo_cobertura_evidencia') IS NULL
    OR pg_catalog.to_regprocedure('vec_contratacion_temporal.ct166_periodo_material_v1(jsonb)') IS NOT NULL THEN
    RAISE EXCEPTION 'CT166 preimagen incompatible' USING ERRCODE='55000';
 END IF;
END
$precondicion$;

-- La función CT165 comprueba las claves, la política y el modo de fin. Esta
-- función añade los límites temporales propios del intercambio C1.
CREATE FUNCTION vec_contratacion_temporal.ct166_periodo_valido_v1(p jsonb)
RETURNS boolean LANGUAGE plpgsql IMMUTABLE STRICT SET search_path=pg_catalog
AS $funcion$
DECLARE v_inicio timestamptz; v_fin timestamptz;
BEGIN
 IF NOT vec_contratacion_temporal.gobi_o404b_instante_texto_valido(p->>'inicio',false)
    OR (p ? 'fin' AND NOT vec_contratacion_temporal.gobi_o404b_instante_texto_valido(p->>'fin',false)) THEN
    RETURN false;
 END IF;
 v_inicio := (p->>'inicio')::timestamptz;
 -- El intercambio anterior aceptaba horas distintas de medianoche.
 IF NOT p ? 'politica_fin' THEN
    RETURN p ? 'fin' AND pg_catalog.jsonb_typeof(p->'fin')='string'
       AND (p-ARRAY['inicio','fin'])='{}'::jsonb
       AND v_inicio<=(p->>'fin')::timestamptz
       AND (p->>'fin')::timestamptz<=v_inicio+interval '100 years';
 END IF;
 IF vec_contratacion_temporal.periodo_previsto_analisis_valido_v1(p) IS NOT TRUE
    OR v_inicio <> pg_catalog.date_trunc('day',v_inicio) THEN RETURN false; END IF;
 IF p ? 'fin' THEN
    v_fin := (p->>'fin')::timestamptz;
    RETURN v_fin=pg_catalog.date_trunc('day',v_fin)
       AND v_fin>=v_inicio AND v_fin<=v_inicio+interval '100 years';
 END IF;
 RETURN p ? 'politica_fin';
EXCEPTION WHEN data_exception OR datetime_field_overflow
  OR invalid_text_representation THEN RETURN false;
END
$funcion$;

-- El sufijo nuevo se usa sólo con un snapshot de política. En ausencia de él,
-- conserva exactamente los dos int8 de la fecha de la variante histórica.
CREATE FUNCTION vec_contratacion_temporal.ct166_periodo_material_v1(p jsonb)
RETURNS bytea LANGUAGE plpgsql IMMUTABLE STRICT SET search_path=pg_catalog
AS $funcion$
DECLARE v bytea; q jsonb;
BEGIN
 IF vec_contratacion_temporal.ct166_periodo_valido_v1(p) IS NOT TRUE THEN RETURN NULL; END IF;
 v := pg_catalog.int8send(vec_contratacion_temporal.gobi_o404b_microsegundos((p->>'inicio')::timestamptz));
 IF NOT p ? 'politica_fin' THEN
    RETURN v || pg_catalog.int8send(vec_contratacion_temporal.gobi_o404b_microsegundos((p->>'fin')::timestamptz));
 END IF;
 q := p->'politica_fin';
 IF p ? 'fin' THEN
    v := vec_contratacion_temporal.gobi_o404b_texto_canon(v,'fecha') ||
       pg_catalog.int8send(vec_contratacion_temporal.gobi_o404b_microsegundos((p->>'fin')::timestamptz));
 ELSE
    v := vec_contratacion_temporal.gobi_o404b_texto_canon(v,'causa');
    v := vec_contratacion_temporal.gobi_o404b_texto_canon(v,p->>'causa_fin');
 END IF;
 v := vec_contratacion_temporal.gobi_o404b_texto_canon(v,q->>'regla_ref') ||
      pg_catalog.int8send((q->>'catalogo_version')::bigint);
 v := vec_contratacion_temporal.gobi_o404b_texto_canon(v,q->>'catalogo_huella_sha256');
 v := vec_contratacion_temporal.gobi_o404b_texto_canon(v,q->>'fecha_fin');
 RETURN vec_contratacion_temporal.gobi_o404b_texto_canon(v,coalesce(q->>'causa_fin',''));
EXCEPTION WHEN data_exception OR datetime_field_overflow
  OR invalid_text_representation OR numeric_value_out_of_range THEN RETURN NULL;
END
$funcion$;

-- La identidad de las órdenes C1 codifica instantes como texto, igual que Go.
CREATE FUNCTION vec_contratacion_temporal.ct166_periodo_orden_v1(p jsonb)
RETURNS bytea LANGUAGE plpgsql IMMUTABLE STRICT SET search_path=pg_catalog
AS $funcion$
DECLARE v bytea:=''::bytea; q jsonb;
BEGIN
 IF vec_contratacion_temporal.ct166_periodo_valido_v1(p) IS NOT TRUE THEN RETURN NULL; END IF;
 v := vec_contratacion_temporal.o404e_texto_v1(v,p->>'inicio');
 IF NOT p ? 'politica_fin' THEN
    RETURN vec_contratacion_temporal.o404e_texto_v1(v,p->>'fin');
 END IF;
 q := p->'politica_fin';
 IF p ? 'fin' THEN
    v := vec_contratacion_temporal.o404e_texto_v1(v,'fecha');
    v := vec_contratacion_temporal.o404e_texto_v1(v,p->>'fin');
 ELSE
    v := vec_contratacion_temporal.o404e_texto_v1(v,'causa');
    v := vec_contratacion_temporal.o404e_texto_v1(v,p->>'causa_fin');
 END IF;
 v := vec_contratacion_temporal.o404e_texto_v1(v,q->>'regla_ref') ||
      pg_catalog.int8send((q->>'catalogo_version')::bigint);
 v := vec_contratacion_temporal.o404e_texto_v1(v,q->>'catalogo_huella_sha256');
 v := vec_contratacion_temporal.o404e_texto_v1(v,q->>'fecha_fin');
 RETURN vec_contratacion_temporal.o404e_texto_v1(v,coalesce(q->>'causa_fin',''));
EXCEPTION WHEN data_exception OR datetime_field_overflow
  OR invalid_text_representation OR numeric_value_out_of_range THEN RETURN NULL;
END
$funcion$;

REVOKE ALL ON FUNCTION
 vec_contratacion_temporal.ct166_periodo_valido_v1(jsonb),
 vec_contratacion_temporal.ct166_periodo_material_v1(jsonb),
 vec_contratacion_temporal.ct166_periodo_orden_v1(jsonb)
FROM PUBLIC, vec_contratacion_temporal_ejecutor,
 vec_contratacion_temporal_migrador, vec_contratacion_temporal_gobernador;

-- La columna de fin pasa a ser opcional sólo para nuevas filas con causa y
-- snapshot. Ninguna fila previa cambia; las restricciones siguen completas.
ALTER TABLE vec_contratacion_temporal.consumo_cobertura_evidencia
    ALTER COLUMN periodo_fin DROP NOT NULL,
    ADD COLUMN periodo_causa_fin text,
    ADD COLUMN politica_fin jsonb;
DO $restriccion$
DECLARE v_nombre text; v_total integer;
BEGIN
 SELECT pg_catalog.count(*), pg_catalog.min(c.conname)
   INTO v_total,v_nombre
   FROM pg_catalog.pg_constraint c
  WHERE c.conrelid='vec_contratacion_temporal.consumo_cobertura_evidencia'::pg_catalog.regclass
    AND c.contype='c'
    AND pg_catalog.pg_get_constraintdef(c.oid) LIKE '%periodo_inicio < periodo_fin%';
 IF v_total<>1 THEN RAISE EXCEPTION 'CT166 restricción temporal incompatible' USING ERRCODE='55000'; END IF;
 EXECUTE pg_catalog.format('ALTER TABLE vec_contratacion_temporal.consumo_cobertura_evidencia DROP CONSTRAINT %I',v_nombre);
END
$restriccion$;
ALTER TABLE vec_contratacion_temporal.consumo_cobertura_evidencia
 ADD CONSTRAINT consumo_cobertura_evidencia_periodo_ct166 CHECK (
    (((periodo_fin IS NOT NULL AND periodo_causa_fin IS NULL
       AND (politica_fin IS NULL OR
          (vec_contratacion_temporal.politica_fin_estructural_v1(politica_fin) IS TRUE
           AND politica_fin->>'fecha_fin' IN ('obligatoria','opcional')))
       AND ((politica_fin IS NULL AND periodo_inicio<periodo_fin)
         OR (politica_fin IS NOT NULL AND periodo_inicio<=periodo_fin)))
     OR (periodo_fin IS NULL AND periodo_causa_fin IS NOT NULL
       AND periodo_causa_fin ~ '^[a-z][a-z0-9._-]{1,79}$'
       AND vec_contratacion_temporal.politica_fin_estructural_v1(politica_fin) IS TRUE
       AND politica_fin->>'causa_fin'=periodo_causa_fin
       AND politica_fin->>'fecha_fin' IN ('opcional','no_aplica')))) IS TRUE
    AND solicitada_en<=emitida_en AND emitida_en<valida_hasta
 );

-- La fila plana del lote se reconstruye con las mismas claves que CT165.
-- periodo_fin:null es explícito en la rama abierta; ausente no equivale a NULL.
CREATE FUNCTION vec_contratacion_temporal.ct166_periodo_evidencia_v1(e jsonb)
RETURNS jsonb LANGUAGE plpgsql IMMUTABLE STRICT SET search_path=pg_catalog
AS $funcion$
DECLARE p jsonb;
BEGIN
 IF NOT (e ? 'periodo_inicio') OR NOT (e ? 'periodo_fin') THEN RETURN NULL; END IF;
 p := pg_catalog.jsonb_build_object('inicio',e->'periodo_inicio');
 IF pg_catalog.jsonb_typeof(e->'periodo_fin')='string' THEN
    IF e ? 'causa_fin' THEN RETURN NULL; END IF;
    p := p || pg_catalog.jsonb_build_object('fin',e->'periodo_fin');
 ELSIF pg_catalog.jsonb_typeof(e->'periodo_fin')='null' THEN
    IF NOT (e ? 'causa_fin') THEN RETURN NULL; END IF;
    p := p || pg_catalog.jsonb_build_object('causa_fin',e->'causa_fin');
 ELSE RETURN NULL;
 END IF;
 IF e ? 'politica_fin' THEN
    p := p || pg_catalog.jsonb_build_object('politica_fin',e->'politica_fin');
 END IF;
 IF vec_contratacion_temporal.ct166_periodo_valido_v1(p) IS NOT TRUE THEN RETURN NULL; END IF;
 IF NOT p ? 'politica_fin' AND NOT p ? 'fin' THEN RETURN NULL; END IF;
 RETURN p;
END
$funcion$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.ct166_periodo_evidencia_v1(jsonb)
FROM PUBLIC, vec_contratacion_temporal_ejecutor,
 vec_contratacion_temporal_migrador, vec_contratacion_temporal_gobernador;

-- Parches guardados por marcas únicas del cuerpo real. Nunca se modifica el
-- contenido de CT23/24 ya instalado ni sus comprobaciones de autorización.
DO $parches_c1$
DECLARE item record; v_def text;
BEGIN
 FOR item IN SELECT * FROM (VALUES
  ('vec_contratacion_temporal.o404d_material_evidencia_v1(jsonb)',
   $old$IF v_claves IS DISTINCT FROM ARRAY[$old$,
   $new$IF pg_catalog.array_remove(pg_catalog.array_remove(v_claves,'causa_fin'),'politica_fin') IS DISTINCT FROM ARRAY[$new$),
  ('vec_contratacion_temporal.o404d_material_evidencia_v1(jsonb)',
   $old$           'version_expediente', 'via_clave'
       ]::text[] THEN$old$,
   $new$           'version_expediente', 'via_clave'
       ]::text[]
       OR (p_evidencia ? 'causa_fin' AND NOT p_evidencia ? 'politica_fin')
       OR vec_contratacion_temporal.ct166_periodo_evidencia_v1(p_evidencia) IS NULL THEN$new$),
  ('vec_contratacion_temporal.o404d_material_evidencia_v1(jsonb)',
   $old$        'periodo_inicio', 'periodo_fin', 'solicitada_en', 'emitida_en',$old$,
   $new$        'periodo_inicio', 'solicitada_en', 'emitida_en',$new$),
  ('vec_contratacion_temporal.o404d_material_evidencia_v1(jsonb)',
   $old$    v_fin := (p_evidencia ->> 'periodo_fin')::timestamptz;$old$,
   $new$    v_fin := CASE WHEN pg_catalog.jsonb_typeof(p_evidencia->'periodo_fin')='string'
                  THEN (p_evidencia->>'periodo_fin')::timestamptz END;$new$),
  ('vec_contratacion_temporal.o404d_material_evidencia_v1(jsonb)',
   $old$    IF v_inicio >= v_fin
       OR v_fin > v_inicio + interval '100 years'$old$,
   $new$    IF (v_fin IS NOT NULL AND
        ((NOT p_evidencia ? 'politica_fin' AND v_inicio >= v_fin)
         OR (p_evidencia ? 'politica_fin' AND v_inicio > v_fin)
         OR v_fin > v_inicio + interval '100 years'))$new$),
  ('vec_contratacion_temporal.o404d_material_evidencia_v1(jsonb)',
   $old$        v_material, 'VEC-CT-CONSUMO-C1-EVIDENCIA-O4-04D-V1'$old$,
   $new$        v_material, CASE WHEN p_evidencia ? 'politica_fin'
            THEN 'VEC-CT-CONSUMO-C1-EVIDENCIA-O4-04D-V2'
            ELSE 'VEC-CT-CONSUMO-C1-EVIDENCIA-O4-04D-V1' END$new$),
  ('vec_contratacion_temporal.o404d_material_evidencia_v1(jsonb)',
   $old$    v_material := v_material
        || pg_catalog.int8send(
            vec_contratacion_temporal.gobi_o404b_microsegundos(v_inicio)
        )
        || pg_catalog.int8send(
            vec_contratacion_temporal.gobi_o404b_microsegundos(v_fin)
        )
        || pg_catalog.int8send(
            vec_contratacion_temporal.gobi_o404b_microsegundos(
                v_solicitada
            )
        )$old$,
   $new$    v_material := v_material
        || vec_contratacion_temporal.ct166_periodo_material_v1(
            vec_contratacion_temporal.ct166_periodo_evidencia_v1(p_evidencia)
        )
        || pg_catalog.int8send(
            vec_contratacion_temporal.gobi_o404b_microsegundos(
                v_solicitada
            )
        )$new$),
  ('vec_contratacion_temporal.o404d_material_lote_v1(jsonb)',
   $old$    v_total := pg_catalog.jsonb_array_length(p_lote -> 'evidencias');$old$,
   $new$    v_total := pg_catalog.jsonb_array_length(p_lote -> 'evidencias');
    IF EXISTS (SELECT 1 FROM pg_catalog.jsonb_array_elements(p_lote->'evidencias') e
                 WHERE e ? 'politica_fin')
       AND EXISTS (SELECT 1 FROM pg_catalog.jsonb_array_elements(p_lote->'evidencias') e
                 WHERE NOT e ? 'politica_fin') THEN
        RETURN NULL;
    END IF;$new$),
  ('vec_contratacion_temporal.o404d_material_lote_v1(jsonb)',
   $old$        v_material, 'VEC-CT-CONSUMO-C1-LOTE-O4-04D-V1'$old$,
   $new$        v_material, CASE WHEN (p_lote->'evidencias'->0) ? 'politica_fin'
            THEN 'VEC-CT-CONSUMO-C1-LOTE-O4-04D-V2'
            ELSE 'VEC-CT-CONSUMO-C1-LOTE-O4-04D-V1' END$new$),
  ('vec_contratacion_temporal.persistir_lote_consumo_c1_cobertura_o404d_v1(jsonb,jsonb)',
   $old$            periodo_fin,
            solicitada_en,$old$,
   $new$            periodo_fin,
            periodo_causa_fin,
            politica_fin,
            solicitada_en,$new$),
  ('vec_contratacion_temporal.persistir_lote_consumo_c1_cobertura_o404d_v1(jsonb,jsonb)',
   $old$            (v_evidencia.valor ->> 'periodo_fin')::timestamptz,
            (v_evidencia.valor ->> 'solicitada_en')::timestamptz,$old$,
   $new$            (v_evidencia.valor ->> 'periodo_fin')::timestamptz,
            v_evidencia.valor ->> 'causa_fin',
            v_evidencia.valor -> 'politica_fin',
            (v_evidencia.valor ->> 'solicitada_en')::timestamptz,$new$)
 ) AS cambios(firma,anterior,nuevo) LOOP
   IF pg_catalog.to_regprocedure(item.firma) IS NULL THEN
      RAISE EXCEPTION 'CT166 función ausente: %',item.firma USING ERRCODE='55000';
   END IF;
   SELECT pg_catalog.pg_get_functiondef(item.firma::pg_catalog.regprocedure) INTO v_def;
   IF pg_catalog.length(v_def)-pg_catalog.length(pg_catalog.replace(v_def,item.anterior,''))
      <>pg_catalog.length(item.anterior) THEN
      RAISE EXCEPTION 'CT166 marca incompatible: %',item.firma USING ERRCODE='55000';
   END IF;
   EXECUTE pg_catalog.replace(v_def,item.anterior,item.nuevo);
 END LOOP;
END
$parches_c1$;

-- La propuesta V2 y la identidad semántica usan el mismo periodo original.
DO $parches_propuesta$
DECLARE item record; v_def text;
BEGIN
 FOR item IN SELECT * FROM (VALUES
  ('vec_contratacion_temporal.o404e_material_propuesta_cobertura_v1(jsonb)',
   $old$           p_publicacion -> 'periodo', ARRAY['fin', 'inicio']::text[]
       )$old$,
   $new$           p_publicacion -> 'periodo', ARRAY['fin', 'inicio']::text[]
       ) AND NOT (p_publicacion #> '{periodo,politica_fin}' IS NOT NULL)
       OR vec_contratacion_temporal.ct166_periodo_valido_v1(p_publicacion->'periodo') IS NOT TRUE$new$),
  ('vec_contratacion_temporal.o404e_material_propuesta_cobertura_v1(jsonb)',
   $old$           p_publicacion #> '{canon,version_esquema}', 1, 1
       )$old$,
   $new$           p_publicacion #> '{canon,version_esquema}', 1, 2
       )
       OR (p_publicacion #>> '{canon,version_esquema}')::integer<>
          (CASE WHEN p_publicacion #> '{periodo,politica_fin}' IS NOT NULL THEN 2 ELSE 1 END)$new$),
  ('vec_contratacion_temporal.o404e_material_propuesta_cobertura_v1(jsonb)',
   $old$       OR NOT vec_contratacion_temporal.gobi_o404b_instante_texto_valido(
           p_publicacion #>> '{periodo,fin}', false
       )$old$,
   $new$       OR (p_publicacion #> '{periodo,fin}' IS NOT NULL
          AND NOT vec_contratacion_temporal.gobi_o404b_instante_texto_valido(
              p_publicacion #>> '{periodo,fin}', false
          ))$new$),
  ('vec_contratacion_temporal.o404e_material_propuesta_cobertura_v1(jsonb)',
   $old$    v_fin := (p_publicacion #>> '{periodo,fin}')::timestamptz;$old$,
   $new$    v_fin := CASE WHEN p_publicacion #> '{periodo,fin}' IS NOT NULL
                  THEN (p_publicacion #>> '{periodo,fin}')::timestamptz END;$new$),
  ('vec_contratacion_temporal.o404e_material_propuesta_cobertura_v1(jsonb)',
   $old$       OR v_fin <> pg_catalog.date_trunc('day', v_fin)
       OR v_fin < v_inicio
       OR v_fin > v_inicio + interval '100 years'$old$,
   $new$       OR (v_fin IS NOT NULL AND
           (v_fin <> pg_catalog.date_trunc('day',v_fin)
            OR v_fin < v_inicio
            OR v_fin > v_inicio + interval '100 years'))$new$),
  ('vec_contratacion_temporal.o404e_material_propuesta_cobertura_v1(jsonb)',
   $old$    ) || pg_catalog.decode('0001', 'hex');$old$,
   $new$    ) || pg_catalog.int2send((p_publicacion #>> '{canon,version_esquema}')::smallint);$new$),
  ('vec_contratacion_temporal.o404e_material_propuesta_cobertura_v1(jsonb)',
   $old$    v_material := v_material
        || pg_catalog.int8send(
            vec_contratacion_temporal.gobi_o404b_microsegundos(v_inicio)
        )
        || pg_catalog.int8send(
            vec_contratacion_temporal.gobi_o404b_microsegundos(v_fin)
        )
        || pg_catalog.int8send(
            vec_contratacion_temporal.gobi_o404b_microsegundos(v_generada)
        )$old$,
   $new$    v_material := v_material
        || vec_contratacion_temporal.ct166_periodo_material_v1(p_publicacion->'periodo')
        || pg_catalog.int8send(
            vec_contratacion_temporal.gobi_o404b_microsegundos(v_generada)
        )$new$),
  ('vec_contratacion_temporal.o404e_semantica_propuesta_v1(jsonb)',
   $old$      'propuesta-decision-cobertura-semantica')||
    pg_catalog.int2send(1::smallint);$old$,
   $new$      'propuesta-decision-cobertura-semantica')||
    pg_catalog.int2send(CASE WHEN p#>'{periodo,politica_fin}' IS NOT NULL
      THEN 2::smallint ELSE 1::smallint END);$new$),
  ('vec_contratacion_temporal.o404e_semantica_propuesta_v1(jsonb)',
   $old$  v:=v||pg_catalog.int8send(
    vec_contratacion_temporal.gobi_o404b_microsegundos(
      (p#>>'{periodo,inicio}')::timestamptz))||
    pg_catalog.int8send(
    vec_contratacion_temporal.gobi_o404b_microsegundos(
      (p#>>'{periodo,fin}')::timestamptz));$old$,
   $new$  v:=v||vec_contratacion_temporal.ct166_periodo_material_v1(p->'periodo');$new$),
  ('vec_contratacion_temporal.o404e_contexto_recurso_concesion_v1(jsonb)',
   $old$        'propuesta-cobertura-semantica:sha256:'||v_semantica$old$,
   $new$        (CASE WHEN p#>'{periodo,politica_fin}' IS NOT NULL
          THEN 'propuesta-cobertura-semantica:v2:sha256:'
          ELSE 'propuesta-cobertura-semantica:sha256:' END) || v_semantica$new$)
 ) AS cambios(firma,anterior,nuevo) LOOP
   IF pg_catalog.to_regprocedure(item.firma) IS NULL THEN
      RAISE EXCEPTION 'CT166 función ausente: %',item.firma USING ERRCODE='55000';
   END IF;
   SELECT pg_catalog.pg_get_functiondef(item.firma::pg_catalog.regprocedure) INTO v_def;
   IF pg_catalog.length(v_def)-pg_catalog.length(pg_catalog.replace(v_def,item.anterior,''))
      <>pg_catalog.length(item.anterior) THEN
      RAISE EXCEPTION 'CT166 marca incompatible: %',item.firma USING ERRCODE='55000';
   END IF;
   EXECUTE pg_catalog.replace(v_def,item.anterior,item.nuevo);
 END LOOP;
END
$parches_propuesta$;

-- La confirmación calcula la huella de las órdenes antes de crear el lote.
DO $parches_confirmacion$
DECLARE item record; v_def text;
BEGIN
 FOR item IN SELECT * FROM (VALUES
  ('vec_contratacion_temporal.o404e_huella_ordenes_c1_v1(jsonb)',
   $old$           OR NOT vec_contratacion_temporal.o404e_claves_exactas_v1(
               v_consumo.valor -> 'periodo',
               ARRAY['fin','inicio']::text[]
           )$old$,
   $new$           OR vec_contratacion_temporal.ct166_periodo_valido_v1(
               v_consumo.valor -> 'periodo') IS NOT TRUE$new$),
  ('vec_contratacion_temporal.o404e_huella_ordenes_c1_v1(jsonb)',
   $old$            'VEC-CT-IDENTIDAD-ORDEN-C1-CONFIRMACION-C3-V1'$old$,
   $new$            CASE WHEN v_consumo.valor#>'{periodo,politica_fin}' IS NOT NULL
              THEN 'VEC-CT-IDENTIDAD-ORDEN-C1-CONFIRMACION-C3-V2'
              ELSE 'VEC-CT-IDENTIDAD-ORDEN-C1-CONFIRMACION-C3-V1' END$new$),
  ('vec_contratacion_temporal.o404e_huella_ordenes_c1_v1(jsonb)',
   $old$            v_consumo.valor ->> 'categoria_ref',
            v_consumo.valor #>> '{periodo,inicio}',
            v_consumo.valor #>> '{periodo,fin}',
            v_consumo.valor ->> 'solicitada_en',$old$,
   $new$            v_consumo.valor ->> 'categoria_ref'
        ]::text[] LOOP
            v_material := vec_contratacion_temporal.o404e_texto_v1(
                v_material, v_texto
            );
        END LOOP;
        v_material := v_material ||
            vec_contratacion_temporal.ct166_periodo_orden_v1(v_consumo.valor->'periodo');
        FOREACH v_texto IN ARRAY ARRAY[
            v_consumo.valor ->> 'solicitada_en',$new$),
  ('vec_contratacion_temporal.o404e_construir_lote_c1_v1(jsonb,text)',
   $old$    IF vec_contratacion_temporal.o404e_huella_ordenes_c1_v1(
           p_carga -> 'consumos_c1'
       ) IS DISTINCT FROM c ->> 'huella_ordenes_consumo_c1_sha256' THEN$old$,
   $new$    IF EXISTS (
         SELECT 1 FROM pg_catalog.jsonb_array_elements(p_carga->'consumos_c1') x
          WHERE (x#>'{periodo,politica_fin}' IS NOT NULL
              OR p_carga#>'{concesion,propuesta,periodo,politica_fin}' IS NOT NULL)
            AND x->'periodo' IS DISTINCT FROM
                p_carga#>'{concesion,propuesta,periodo}'
       ) THEN
        RETURN NULL;
    END IF;
    IF vec_contratacion_temporal.o404e_huella_ordenes_c1_v1(
           p_carga -> 'consumos_c1'
       ) IS DISTINCT FROM c ->> 'huella_ordenes_consumo_c1_sha256' THEN$new$),
  ('vec_contratacion_temporal.o404e_construir_lote_c1_v1(jsonb,text)',
   $old$        v_huella := pg_catalog.encode(pg_catalog.sha256(
            vec_contratacion_temporal.o404d_material_evidencia_v1(e)$old$,
   $new$        IF x#>'{periodo,politica_fin}' IS NOT NULL THEN
            e := e || pg_catalog.jsonb_build_object('politica_fin',x#>'{periodo,politica_fin}');
        END IF;
        IF x#>'{periodo,causa_fin}' IS NOT NULL THEN
            e := e || pg_catalog.jsonb_build_object('causa_fin',x#>'{periodo,causa_fin}');
        END IF;
        v_huella := pg_catalog.encode(pg_catalog.sha256(
            vec_contratacion_temporal.o404d_material_evidencia_v1(e)$new$)
 ) AS cambios(firma,anterior,nuevo) LOOP
   IF pg_catalog.to_regprocedure(item.firma) IS NULL THEN
      RAISE EXCEPTION 'CT166 función ausente: %',item.firma USING ERRCODE='55000';
   END IF;
   SELECT pg_catalog.pg_get_functiondef(item.firma::pg_catalog.regprocedure) INTO v_def;
   IF pg_catalog.length(v_def)-pg_catalog.length(pg_catalog.replace(v_def,item.anterior,''))
      <>pg_catalog.length(item.anterior) THEN
      RAISE EXCEPTION 'CT166 marca incompatible: %',item.firma USING ERRCODE='55000';
   END IF;
   EXECUTE pg_catalog.replace(v_def,item.anterior,item.nuevo);
 END LOOP;
END
$parches_confirmacion$;

COMMIT;
