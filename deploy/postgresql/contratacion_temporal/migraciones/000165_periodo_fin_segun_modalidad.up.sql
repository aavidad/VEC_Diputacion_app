\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_contratacion_temporal:000165:periodo_fin', 0));

-- La modalidad y su política pertenecen al catálogo versionado de aplicación.
-- Aquí se comprueba sólo la forma de ambos periodos, sin inferir una fecha.
CREATE FUNCTION vec_contratacion_temporal.politica_fin_estructural_v1(p jsonb)
RETURNS boolean LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog
AS $funcion$
DECLARE
    v_modo text;
    v_claves integer;
BEGIN
 IF pg_catalog.jsonb_typeof(p) IS DISTINCT FROM 'object' THEN RETURN false; END IF;
 v_claves := (SELECT count(*) FROM pg_catalog.jsonb_object_keys(p));
 v_modo := p->>'fecha_fin';
 IF (p-ARRAY['regla_ref','catalogo_version','catalogo_huella_sha256',
              'fecha_fin','causa_fin'])<>'{}'::jsonb
    OR NOT (p ?& ARRAY['regla_ref','catalogo_version',
                       'catalogo_huella_sha256','fecha_fin'])
    OR v_claves NOT IN (4,5)
    OR pg_catalog.jsonb_typeof(p->'regla_ref') IS DISTINCT FROM 'string'
    OR p->>'regla_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
    OR pg_catalog.jsonb_typeof(p->'catalogo_version') IS DISTINCT FROM 'number'
    OR (p->>'catalogo_version') !~ '^[1-9][0-9]{0,15}$'
    OR pg_catalog.jsonb_typeof(p->'catalogo_huella_sha256') IS DISTINCT FROM 'string'
    OR p->>'catalogo_huella_sha256' !~ '^[0-9a-f]{64}$'
    OR pg_catalog.jsonb_typeof(p->'fecha_fin') IS DISTINCT FROM 'string'
    OR v_modo NOT IN ('obligatoria','opcional','no_aplica') THEN
    RETURN false;
 END IF;
 IF v_modo='obligatoria' THEN RETURN v_claves=4; END IF;
 RETURN v_claves=5
    AND pg_catalog.jsonb_typeof(p->'causa_fin')='string'
    AND p->>'causa_fin' ~ '^[a-z][a-z0-9._-]{1,79}$';
END
$funcion$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.politica_fin_estructural_v1(jsonb) FROM PUBLIC;

CREATE FUNCTION vec_contratacion_temporal.periodo_previsto_estructural_v1(p jsonb)
RETURNS boolean LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog
AS $funcion$
DECLARE v_claves integer;
BEGIN
 IF pg_catalog.jsonb_typeof(p) IS DISTINCT FROM 'object' THEN RETURN false; END IF;
 v_claves := (SELECT count(*) FROM pg_catalog.jsonb_object_keys(p));
 IF v_claves NOT IN (2,3)
    OR (p-ARRAY['inicio','fin','causa_fin','politica_fin'])<>'{}'::jsonb
    OR pg_catalog.jsonb_typeof(p->'inicio') IS DISTINCT FROM 'string'
    OR (p ? 'politica_fin' AND
        NOT vec_contratacion_temporal.politica_fin_estructural_v1(p->'politica_fin')) THEN
    RETURN false;
 END IF;
 IF p ? 'fin' AND NOT (p ? 'causa_fin') THEN
    RETURN pg_catalog.jsonb_typeof(p->'fin')='string'
       AND ((v_claves=2 AND NOT (p ? 'politica_fin'))
         OR (v_claves=3 AND p ? 'politica_fin'
             AND p #>> '{politica_fin,fecha_fin}' IN ('obligatoria','opcional')));
 END IF;
 IF p ? 'causa_fin' AND NOT (p ? 'fin') THEN
    RETURN v_claves=3 AND p ? 'politica_fin'
       AND pg_catalog.jsonb_typeof(p->'causa_fin')='string'
       AND p->>'causa_fin' ~ '^[a-z][a-z0-9._-]{1,79}$'
       AND p #>> '{politica_fin,fecha_fin}' IN ('opcional','no_aplica')
       AND p->>'causa_fin'=p #>> '{politica_fin,causa_fin}';
 END IF;
 RETURN false;
END
$funcion$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.periodo_previsto_estructural_v1(jsonb) FROM PUBLIC;

CREATE FUNCTION vec_contratacion_temporal.periodo_previsto_analisis_valido_v1(p jsonb)
RETURNS boolean LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog
AS $funcion$
BEGIN
 IF NOT vec_contratacion_temporal.periodo_previsto_estructural_v1(p)
    OR NOT vec_contratacion_temporal.instante_utc_json_canonico_v2(p->'inicio',true) THEN
    RETURN false;
 END IF;
 IF p ? 'fin' THEN
    RETURN vec_contratacion_temporal.instante_utc_json_canonico_v2(p->'fin',true)
       AND (p->>'fin')::timestamptz >= (p->>'inicio')::timestamptz;
 END IF;
 RETURN true;
EXCEPTION WHEN data_exception OR datetime_field_overflow
  OR invalid_text_representation THEN
 RETURN false;
END
$funcion$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.periodo_previsto_analisis_valido_v1(jsonb) FROM PUBLIC;

-- Cada sustitución exige una sola marca exacta de la preimagen instalada. Se
-- conserva toda la definición existente: permisos, CAS, consumo V3 y recibos.
DO $parches$
DECLARE
    item record;
    v_def text;
BEGIN
    FOR item IN
      SELECT * FROM (VALUES
        (
          'vec_contratacion_temporal.registrar_peticion_centro_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
          $antiguo$OR (SELECT count(*) FROM jsonb_object_keys(sol->'periodo'))<>2 OR ((sol->'periodo')-ARRAY['inicio','fin'])<>'{}'::jsonb$antiguo$,
          $nuevo$OR NOT vec_contratacion_temporal.periodo_previsto_estructural_v1(sol->'periodo')$nuevo$
        ),
        (
          'vec_contratacion_temporal.confirmar_alta_atestada_v1(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea)',
          $antiguo$OR (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_object_keys(
               a #> '{solicitud,periodo}')) <> 2
       OR NOT ((a #> '{solicitud,periodo}') ?& ARRAY['inicio', 'fin'])
       OR pg_catalog.jsonb_typeof(a #> '{solicitud,periodo,inicio}')
          <> 'string'
       OR pg_catalog.jsonb_typeof(a #> '{solicitud,periodo,fin}')
          <> 'string'$antiguo$,
          $nuevo$OR NOT vec_contratacion_temporal.periodo_previsto_estructural_v1(a #> '{solicitud,periodo}')$nuevo$
        ),
        (
          'vec_contratacion_temporal.reconstruir_solicitud_efecto_v2(jsonb)',
          $antiguo$',"fin":' || vec_contratacion_temporal.texto_json_go_v1(s #>> '{periodo,fin}') || '}'$antiguo$,
          $nuevo$(CASE WHEN s #> '{periodo,fin}' IS NOT NULL THEN
          ',"fin":' || vec_contratacion_temporal.texto_json_go_v1(s #>> '{periodo,fin}')
        ELSE
          ',"causa_fin":' || vec_contratacion_temporal.texto_json_go_v1(s #>> '{periodo,causa_fin}')
        END) ||
        (CASE WHEN s #> '{periodo,politica_fin}' IS NOT NULL THEN
          ',"politica_fin":{"regla_ref":' ||
          vec_contratacion_temporal.texto_json_go_v1(s #>> '{periodo,politica_fin,regla_ref}') ||
          ',"catalogo_version":' || (s #> '{periodo,politica_fin,catalogo_version}')::text ||
          ',"catalogo_huella_sha256":' ||
          vec_contratacion_temporal.texto_json_go_v1(s #>> '{periodo,politica_fin,catalogo_huella_sha256}') ||
          ',"fecha_fin":' ||
          vec_contratacion_temporal.texto_json_go_v1(s #>> '{periodo,politica_fin,fecha_fin}') ||
          (CASE WHEN s #> '{periodo,politica_fin,causa_fin}' IS NOT NULL THEN
            ',"causa_fin":' ||
            vec_contratacion_temporal.texto_json_go_v1(s #>> '{periodo,politica_fin,causa_fin}')
           ELSE '' END) || '}'
         ELSE '' END) || '}'$nuevo$
        )
        ,
        (
          'vec_contratacion_temporal.huella_analisis_derivado_v2(jsonb)',
          $antiguo$OR NOT vec_contratacion_temporal.claves_json_exactas_v1(
           a -> 'periodo', ARRAY['fin', 'inicio']::text[]
       )$antiguo$,
          $nuevo$OR NOT vec_contratacion_temporal.periodo_previsto_analisis_valido_v1(a -> 'periodo')$nuevo$
        ),
        (
          'vec_contratacion_temporal.huella_analisis_derivado_v2(jsonb)',
          $antiguo$OR NOT vec_contratacion_temporal.instante_utc_json_canonico_v2(
           a #> '{periodo,fin}', true
       )$antiguo$,
          $nuevo$OR (a #> '{periodo,fin}' IS NOT NULL
           AND NOT vec_contratacion_temporal.instante_utc_json_canonico_v2(
             a #> '{periodo,fin}', true
           ))$nuevo$
        ),
        (
          'vec_contratacion_temporal.huella_analisis_derivado_v2(jsonb)',
          $antiguo$|| vec_contratacion_temporal.microsegundos_unix_analisis_v1(
            a #>> '{periodo,fin}'
        )$antiguo$,
          $nuevo$|| CASE WHEN a #> '{periodo,fin}' IS NOT NULL THEN
            vec_contratacion_temporal.microsegundos_unix_analisis_v1(
              a #>> '{periodo,fin}'
            )
          ELSE vec_contratacion_temporal.encuadrar_binario_analisis_v1(
              a #>> '{periodo,causa_fin}'
            )
          END$nuevo$
        ),
        (
          'vec_contratacion_temporal.huella_analisis_derivado_v2(jsonb)',
          $antiguo$'VEC-CT-ANALISIS-DERIVADO-O3-V1'$antiguo$,
          $nuevo$CASE WHEN a #> '{periodo,politica_fin}' IS NOT NULL
              THEN 'VEC-CT-ANALISIS-DERIVADO-O3-V2'
              ELSE 'VEC-CT-ANALISIS-DERIVADO-O3-V1' END$nuevo$
        ),
        (
          'vec_contratacion_temporal.huella_analisis_derivado_v2(jsonb)',
          $antiguo$|| pg_catalog.int8send((a ->> 'porcentaje_jornada')::bigint)$antiguo$,
          $nuevo$|| CASE WHEN a #> '{periodo,politica_fin}' IS NOT NULL THEN
              vec_contratacion_temporal.encuadrar_binario_analisis_v1(
                a #>> '{periodo,politica_fin,regla_ref}'
              )
              || pg_catalog.int8send((a #>> '{periodo,politica_fin,catalogo_version}')::bigint)
              || vec_contratacion_temporal.encuadrar_binario_analisis_v1(
                a #>> '{periodo,politica_fin,catalogo_huella_sha256}'
              )
              || vec_contratacion_temporal.encuadrar_binario_analisis_v1(
                a #>> '{periodo,politica_fin,fecha_fin}'
              )
              || vec_contratacion_temporal.encuadrar_binario_analisis_v1(
                coalesce(a #>> '{periodo,politica_fin,causa_fin}','')
              )
            ELSE ''::bytea END
          || pg_catalog.int8send((a ->> 'porcentaje_jornada')::bigint)$nuevo$
        ),
        (
          'vec_contratacion_temporal.expediente_analisis_valido_v2(jsonb,boolean)',
          $antiguo$OR NOT vec_contratacion_temporal.claves_json_exactas_v1(
           s -> 'periodo', ARRAY['fin', 'inicio']::text[]
       )$antiguo$,
          $nuevo$OR NOT vec_contratacion_temporal.periodo_previsto_analisis_valido_v1(s -> 'periodo')$nuevo$
        ),
        (
          'vec_contratacion_temporal.expediente_analisis_valido_v2(jsonb,boolean)',
          $antiguo$OR NOT vec_contratacion_temporal.instante_utc_json_canonico_v2(
           s #> '{periodo,fin}', true
       )$antiguo$,
          $nuevo$OR (s #> '{periodo,fin}' IS NOT NULL
           AND NOT vec_contratacion_temporal.instante_utc_json_canonico_v2(
             s #> '{periodo,fin}', true
           ))$nuevo$
        )
      ) AS cambios(firma, anterior, nuevo)
    LOOP
      IF pg_catalog.to_regprocedure(item.firma) IS NULL THEN
        RAISE EXCEPTION 'CT165 preimagen ausente: %', item.firma USING ERRCODE='55000';
      END IF;
      SELECT pg_catalog.pg_get_functiondef(item.firma::pg_catalog.regprocedure) INTO v_def;
      IF pg_catalog.length(v_def)-pg_catalog.length(pg_catalog.replace(v_def,item.anterior,''))
           <> pg_catalog.length(item.anterior) THEN
        RAISE EXCEPTION 'CT165 marca incompatible: %', item.firma USING ERRCODE='55000';
      END IF;
      EXECUTE pg_catalog.replace(v_def,item.anterior,item.nuevo);
    END LOOP;
END
$parches$;

-- CT159 conserva los recibos históricos y coteja el canon sólo al insertar
-- nuevos detalles. Los atributos se añaden al final para mantener la posición
-- de todos los campos existentes.
ALTER TYPE vec_contratacion_temporal.solicitud_operativa_rrhh_v1
    ADD ATTRIBUTE periodo_causa_fin text;
ALTER TYPE vec_contratacion_temporal.analisis_operativo_rrhh_v1
    ADD ATTRIBUTE periodo_causa_fin text;

DO $detalle$
DECLARE
    item record;
    v_def text;
BEGIN
    FOR item IN SELECT * FROM (VALUES
      (
        'vec_contratacion_temporal.materializar_detalle_rrhh_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,numeric)',
        $antiguo$(v_agregado #>> '{solicitud,periodo,fin}')::timestamptz
    );$antiguo$,
        $nuevo$(v_agregado #>> '{solicitud,periodo,fin}')::timestamptz,
        v_agregado #>> '{solicitud,periodo,causa_fin}'
    );$nuevo$
      ),
      (
        'vec_contratacion_temporal.materializar_detalle_rrhh_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,numeric)',
        $antiguo$COALESCE(v_agregado #>> '{analisis,observaciones}', '')
        );$antiguo$,
        $nuevo$COALESCE(v_agregado #>> '{analisis,observaciones}', ''),
            v_agregado #>> '{analisis,periodo,causa_fin}'
        );$nuevo$
      ),
      (
        'vec_contratacion_temporal.canon_contenido_detalle_rrhh_v1(timestamp with time zone,vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1)',
        $antiguo$'VEC-CT-CONTENIDO-DETALLE-RRHH-V3' || pg_catalog.chr(10)$antiguo$,
        $nuevo$(CASE WHEN (p_entrada.solicitud).periodo_causa_fin IS NOT NULL
                       OR (p_entrada.analisis).periodo_causa_fin IS NOT NULL
                THEN 'VEC-CT-CONTENIDO-DETALLE-RRHH-V4'
                ELSE 'VEC-CT-CONTENIDO-DETALLE-RRHH-V3' END)
                || pg_catalog.chr(10)$nuevo$
      ),
      (
        'vec_contratacion_temporal.canon_contenido_detalle_rrhh_v1(timestamp with time zone,vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1)',
        $antiguo$OR v_solicitud.periodo_fin IS NULL
       OR v_solicitud.periodo_fin < v_solicitud.periodo_inicio$antiguo$,
        $nuevo$OR (v_solicitud.periodo_fin IS NULL
           AND coalesce(v_solicitud.periodo_causa_fin,'') !~
               '^[a-z][a-z0-9._-]{1,79}$')
       OR (v_solicitud.periodo_fin IS NOT NULL
           AND v_solicitud.periodo_causa_fin IS NOT NULL)
       OR v_solicitud.periodo_fin < v_solicitud.periodo_inicio$nuevo$
      ),
      (
        'vec_contratacion_temporal.canon_contenido_detalle_rrhh_v1(timestamp with time zone,vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1)',
        $antiguo$OR v_analisis.periodo_fin IS NULL
           OR v_analisis.periodo_fin < v_analisis.periodo_inicio$antiguo$,
        $nuevo$OR (v_analisis.periodo_fin IS NULL
               AND coalesce(v_analisis.periodo_causa_fin,'') !~
                   '^[a-z][a-z0-9._-]{1,79}$')
           OR (v_analisis.periodo_fin IS NOT NULL
               AND v_analisis.periodo_causa_fin IS NOT NULL)
           OR v_analisis.periodo_fin < v_analisis.periodo_inicio$nuevo$
      ),
      (
        'vec_contratacion_temporal.canon_contenido_detalle_rrhh_v1(timestamp with time zone,vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1)',
        $antiguo$vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
            v_solicitud.periodo_fin
        )$antiguo$,
        $nuevo$CASE WHEN v_solicitud.periodo_fin IS NOT NULL THEN
            (CASE WHEN (p_entrada.solicitud).periodo_causa_fin IS NOT NULL
                    OR (p_entrada.analisis).periodo_causa_fin IS NOT NULL
                THEN 'fecha:' ELSE '' END) ||
            vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
                v_solicitud.periodo_fin
            )
        ELSE 'causa:' || v_solicitud.periodo_causa_fin END$nuevo$
      ),
      (
        'vec_contratacion_temporal.canon_contenido_detalle_rrhh_v1(timestamp with time zone,vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1)',
        $antiguo$vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
                v_analisis.periodo_fin
            )$antiguo$,
        $nuevo$CASE WHEN v_analisis.periodo_fin IS NOT NULL THEN
                (CASE WHEN (p_entrada.solicitud).periodo_causa_fin IS NOT NULL
                        OR (p_entrada.analisis).periodo_causa_fin IS NOT NULL
                    THEN 'fecha:' ELSE '' END) ||
                vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
                    v_analisis.periodo_fin
                )
            ELSE 'causa:' || v_analisis.periodo_causa_fin END$nuevo$
      )
    ) AS cambios(firma, anterior, nuevo) LOOP
      SELECT pg_catalog.pg_get_functiondef(item.firma::pg_catalog.regprocedure) INTO STRICT v_def;
      IF pg_catalog.length(v_def)-pg_catalog.length(pg_catalog.replace(v_def,item.anterior,''))
         <> pg_catalog.length(item.anterior) THEN
        RAISE EXCEPTION 'CT165 detalle incompatible: %', item.firma USING ERRCODE='55000';
      END IF;
      EXECUTE pg_catalog.replace(v_def,item.anterior,item.nuevo);
    END LOOP;
END
$detalle$;

COMMIT;
