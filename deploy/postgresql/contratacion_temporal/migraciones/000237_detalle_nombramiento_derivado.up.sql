\set ON_ERROR_STOP on
-- CT237: misma lectura atestada, con historia propia del hijo e instruccion original separadas.
-- Depende de CT234 y su autorizacion V3; sin ellas no se instala.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000237',0));
DO $pre$
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario'
    OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
    OR to_regclass('vec_contratacion_temporal.nombramiento_derivado_v1') IS NULL
    OR (SELECT count(*) FROM pg_attribute WHERE attrelid='vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1'::regclass AND attnum>0 AND NOT attisdropped)<>16
    OR (SELECT count(*) FROM pg_type t WHERE t.oid='vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1'::regtype
          AND t.typowner='vec_contratacion_temporal_propietario'::regrole
          AND t.typacl::text='{vec_contratacion_temporal_propietario=U/vec_contratacion_temporal_propietario}')<>1
    OR (SELECT count(*) FROM pg_proc p WHERE p.oid='vec_contratacion_temporal.materializar_detalle_rrhh_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,numeric)'::regprocedure
          AND encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')='31b963b0b3b8bdbeb0bcc105e07b8a221e990bc48fa10775dc3a19ad59550725')<>1
    OR (SELECT count(*) FROM pg_proc p WHERE p.oid='vec_contratacion_temporal.canon_contenido_detalle_rrhh_v1(timestamp with time zone,vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1)'::regprocedure
          AND encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')='dd9db72b5bf262bc14508b5cd768b35de9d01540a9fb039c65e3bfb9f506b6ab')<>1
    OR (SELECT count(*) FROM pg_proc p WHERE p.oid='vec_contratacion_temporal.canon_resumen_publicacion_rrhh_v1(vec_contratacion_temporal.resumen_publicacion_rrhh_v1)'::regprocedure
          AND encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')='464e2c9f355f654adbd0c19d2830309c96a9317604dc947bf76f245be5118289')<>1
 THEN RAISE EXCEPTION 'CT237: dependencias o preimagen divergentes' USING ERRCODE='55000'; END IF;
END $pre$;
CREATE TEMP TABLE pg_temp.ct237_funciones_pre AS
 SELECT p.oid,p.proname,p.proowner,p.proacl,p.proconfig,p.provolatile,p.proparallel,p.prosecdef
 FROM pg_proc p WHERE p.oid IN (
 'vec_contratacion_temporal.canon_resumen_publicacion_rrhh_v1(vec_contratacion_temporal.resumen_publicacion_rrhh_v1)'::regprocedure,
 'vec_contratacion_temporal.materializar_detalle_rrhh_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,numeric)'::regprocedure,
 'vec_contratacion_temporal.canon_contenido_detalle_rrhh_v1(timestamp with time zone,vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1)'::regprocedure);
DO $meta$
BEGIN
 IF (SELECT count(*) FROM pg_temp.ct237_funciones_pre)<>3
    OR EXISTS(SELECT 1 FROM pg_temp.ct237_funciones_pre p
      WHERE p.proowner<>'vec_contratacion_temporal_propietario'::regrole
        OR p.proacl::text IS DISTINCT FROM '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario}'
        OR (p.proname LIKE 'canon_%' AND (p.provolatile<>'i' OR p.proparallel<>'s' OR p.prosecdef OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog']))
        OR (p.proname='materializar_detalle_rrhh_v1' AND
           (p.provolatile<>'s' OR p.proparallel<>'u' OR NOT p.prosecdef OR
            p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog','row_security=on','TimeZone=UTC','lock_timeout=1s','statement_timeout=4s','idle_in_transaction_session_timeout=6s'])))
 THEN RAISE EXCEPTION 'CT237: propietario, ACL o configuración divergentes' USING ERRCODE='55000'; END IF;
END $meta$;
ALTER TYPE vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1 ADD ATTRIBUTE instruccion_compartida jsonb;

-- CT237-GENERADO-INICIO: base instalada y parches C4 explicitos.
CREATE OR REPLACE FUNCTION vec_contratacion_temporal.canon_resumen_publicacion_rrhh_v1(p_resumen vec_contratacion_temporal.resumen_publicacion_rrhh_v1)
 RETURNS bytea
 LANGUAGE plpgsql
 IMMUTABLE PARALLEL SAFE STRICT
 SET search_path=pg_catalog,pg_temp
AS $function$
DECLARE
    v_canon bytea := ''::bytea;
    v_campo text;
    v_campos text[];
BEGIN
    IF p_resumen.expediente_ref IS NULL
       OR p_resumen.expediente_ref !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_resumen.organizacion_ref IS NULL
       OR p_resumen.organizacion_ref !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_resumen.numero_visible IS NULL
       OR p_resumen.numero_visible !~
          '^([0-9]{4}/[A-Za-z0-9._-]{1,40}|[0-9]{4}/[A-Za-z0-9._-]{1,40}-P(100|[1-9][0-9]?))$'
       OR p_resumen.version IS NULL
       OR p_resumen.version NOT BETWEEN 1 AND 9007199254740991::numeric
       OR p_resumen.flujo_ref IS NULL
       OR p_resumen.flujo_ref !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_resumen.flujo_version IS NULL
       OR p_resumen.flujo_version
          NOT BETWEEN 1 AND 9007199254740991::numeric
       OR p_resumen.flujo_huella_sha256 IS NULL
       OR p_resumen.flujo_huella_sha256 !~ '^[0-9a-f]{64}$'
       OR p_resumen.flujo_huella_sha256 = pg_catalog.repeat('0', 64)
       OR p_resumen.fase_clave IS NULL
       OR p_resumen.fase_clave !~ '^[a-z][a-z0-9._-]{1,79}$'
       OR p_resumen.estado_clave IS NULL
       OR p_resumen.estado_clave NOT IN (
           'pendiente', 'en_curso', 'espera_externa',
           'completado', 'incidencia', 'cancelado'
       )
       OR p_resumen.centro_ref IS NULL
       OR p_resumen.centro_ref !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_resumen.categoria_ref IS NULL
       OR p_resumen.categoria_ref !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_resumen.modalidad_clave IS NULL
       OR p_resumen.modalidad_clave !~
          '^$|^[a-z][a-z0-9._-]{1,79}$'
       OR p_resumen.unidad_ref IS NULL
       OR p_resumen.unidad_ref !~
          '^$|^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_resumen.creado_en IS NULL
       OR p_resumen.actualizado_en IS NULL
       OR p_resumen.actualizado_en < p_resumen.creado_en THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'resumen canónico RRHH inválido';
    END IF;
    v_campos := ARRAY[
        p_resumen.expediente_ref, p_resumen.organizacion_ref,
        p_resumen.numero_visible, p_resumen.version::text,
        p_resumen.flujo_ref, p_resumen.flujo_version::text,
        p_resumen.flujo_huella_sha256, p_resumen.fase_clave,
        p_resumen.estado_clave, p_resumen.centro_ref,
        p_resumen.categoria_ref, p_resumen.modalidad_clave,
        p_resumen.unidad_ref,
        vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
            p_resumen.creado_en
        ),
        vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
            p_resumen.actualizado_en
        )
    ];
    FOREACH v_campo IN ARRAY v_campos LOOP
        v_canon := v_canon
            || vec_contratacion_temporal.encuadrar_valor_rrhh_v1(
                vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
                    v_campo
                )
            );
    END LOOP;
    RETURN v_canon;
EXCEPTION WHEN OTHERS THEN
    RAISE EXCEPTION USING ERRCODE = '22023',
        MESSAGE = 'resumen canónico RRHH inválido';
END
$function$

;
CREATE OR REPLACE FUNCTION vec_contratacion_temporal.materializar_detalle_rrhh_v1(p_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1, p_consulta vec_contratacion_temporal.consulta_detalle_rrhh_v1, p_corte_global numeric)
 RETURNS vec_contratacion_temporal.materializacion_detalle_rrhh_v1
 LANGUAGE plpgsql
 STABLE SECURITY DEFINER
 SET search_path=pg_catalog,pg_temp
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '1s'
 SET statement_timeout TO '4s'
 SET idle_in_transaction_session_timeout TO '6s'
AS $function$
DECLARE
    v_fila record;
    v_agregado jsonb;
    v_padre jsonb;
    v_instruccion jsonb;
    v_version_instruccion numeric(20,0);
    v_expediente_instruccion_ref text;
    v_derivacion record;
    v_nodo jsonb;
    v_indice integer;
    v_total integer;
    v_resumen
        vec_contratacion_temporal.resumen_publicacion_rrhh_v1;
    v_solicitud
        vec_contratacion_temporal.solicitud_operativa_rrhh_v1;
    v_analisis
        vec_contratacion_temporal.analisis_operativo_rrhh_v1;
    v_cobertura
        vec_contratacion_temporal.cobertura_operativa_rrhh_v1;
    v_asignacion
        vec_contratacion_temporal.asignacion_operativa_rrhh_v1;
    v_fiscalizacion
        vec_contratacion_temporal.fiscalizacion_operativa_rrhh_v1;
    v_subsanacion_nodo jsonb;
    v_subsanaciones integer;
    v_comprobaciones
        vec_contratacion_temporal.comprobacion_operativa_rrhh_v1[];
    v_hitos
        vec_contratacion_temporal.hito_expediente_rrhh_v1[];
    v_detalle
        vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1;
    v_analisis_presente boolean;
    v_cobertura_presente boolean;
    v_asignacion_presente boolean;
    v_fiscalizacion_presente boolean;
    v_subsanacion_presente boolean;
    v_decision_gobernada boolean;
    v_coste_presente boolean;
    v_referencia_analisis numeric(20, 0) := 0;
    v_referencia_cobertura numeric(20, 0) := 0;
    v_referencia_asignacion numeric(20, 0) := 0;
    v_referencia_fiscalizacion numeric(20, 0) := 0;
    v_referencia_subsanacion numeric(20, 0) := 0;
BEGIN
    IF CURRENT_USER <> 'vec_contratacion_temporal_propietario'
       OR p_alcance IS NULL
       OR p_consulta IS NULL
       OR p_corte_global IS NULL
       OR p_corte_global NOT BETWEEN
          1 AND 9007199254740991::numeric
       OR p_corte_global <> pg_catalog.trunc(p_corte_global) THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'detalle RRHH no disponible';
    END IF;

    PERFORM vec_contratacion_temporal.canon_alcance_rrhh_v1(
        p_alcance
    );
    PERFORM vec_contratacion_temporal.canon_consulta_detalle_rrhh_v1(
        p_consulta
    );

    -- MATERIALIZED fija primero la versión actual relativa al corte. El
    -- predicado exterior aplica después el ámbito y el control de versión.
    WITH elegida AS MATERIALIZED (
        SELECT publicacion.*, historia.agregado_json
          FROM vec_contratacion_temporal.publicacion_version_rrhh publicacion
          JOIN vec_contratacion_temporal.expediente_version_integral historia
            ON historia.expediente_ref = publicacion.expediente_ref
           AND historia.version = publicacion.version
           AND historia.flujo_ref = publicacion.flujo_ref
           AND historia.flujo_version = publicacion.flujo_version
           AND historia.flujo_huella_sha256 =
               publicacion.flujo_huella_sha256
           AND historia.fase_clave = publicacion.fase_clave
           AND historia.estado = publicacion.estado_clave
           AND historia.agregado_json_huella_sha256 =
               publicacion.agregado_huella_sha256
           AND historia.registrada_en = publicacion.registrada_en
         WHERE publicacion.expediente_ref = p_consulta.expediente_ref
           AND publicacion.corte_global <= p_corte_global
         ORDER BY publicacion.corte_global DESC
         LIMIT 1
    )
    SELECT elegida.*
      INTO STRICT v_fila
      FROM elegida
     WHERE elegida.organizacion_ref = p_alcance.organizacion_ref
       AND (
           p_consulta.version_observada = 0
           OR p_consulta.version_observada = elegida.version
       )
       AND CASE p_alcance.clase_ambito
           WHEN 'organizacion' THEN
               elegida.organizacion_ref = p_alcance.ambito_ref
           WHEN 'centro' THEN
               elegida.centro_ref = p_alcance.ambito_ref
           WHEN 'unidad_gestion' THEN
               elegida.unidad_ref = p_alcance.ambito_ref
           ELSE false
       END;

    v_agregado := v_fila.agregado_json;
    v_version_instruccion := v_fila.version;
    v_expediente_instruccion_ref := v_fila.expediente_ref;
    IF v_agregado ? 'instruccion_compartida' THEN
        SELECT derivacion.*, alta.derivacion_ref AS alta_derivacion_ref
          INTO STRICT v_derivacion
          FROM vec_contratacion_temporal.nombramiento_derivado_v1 derivacion
          JOIN vec_contratacion_temporal.expediente_alta alta
            ON alta.expediente_ref=derivacion.hijo_ref
           AND alta.derivacion_ref=derivacion.derivacion_ref
         WHERE derivacion.hijo_ref=v_fila.expediente_ref;
        -- CT234 fija el snapshot desde agregado_json y la huella desde JSON Go.
        -- El recibo y la versión v6, no una igualdad textual de fechas, lo acreditan.
        v_padre := v_agregado #> '{instruccion_compartida,expediente_padre}';
        IF pg_catalog.jsonb_typeof(v_padre)<>'object'
           OR pg_catalog.jsonb_typeof(v_padre->'actuaciones')<>'array'
           OR v_agregado #>> '{instruccion_compartida,oferta_ref}' IS DISTINCT FROM v_derivacion.oferta_ref
           OR v_agregado #>> '{instruccion_compartida,numero_plaza}' IS DISTINCT FROM v_derivacion.numero_plaza::text
           OR v_agregado #>> '{instruccion_compartida,recibo_derivacion_ref}' IS DISTINCT FROM v_derivacion.recibo_ref
           OR v_agregado #>> '{instruccion_compartida,huella_padre_sha256}' IS DISTINCT FROM
                pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
                    v_derivacion.material::jsonb->>'padre_canonico','UTF8')),'hex')
           OR v_derivacion.recibo_json#>'{expediente,instruccion_compartida}' IS DISTINCT FROM
                v_agregado->'instruccion_compartida'
           OR NOT EXISTS(SELECT 1 FROM vec_contratacion_temporal.expediente_version_integral inicial
                WHERE inicial.expediente_ref=v_derivacion.hijo_ref AND inicial.version=6
                  AND inicial.agregado_json=v_derivacion.recibo_json->'expediente')
           OR v_padre->>'referencia' IS DISTINCT FROM v_derivacion.padre_ref
           OR v_padre->>'version' IS DISTINCT FROM v_derivacion.version_padre::text
           OR v_agregado #>> '{actuaciones,0,recibo_ref}' IS DISTINCT FROM v_derivacion.recibo_ref
           OR v_agregado #>> '{actuaciones,0,actor_ref}' IS DISTINCT FROM v_derivacion.actor_ref
           OR v_agregado #>> '{actuaciones,0,accion_clave}' IS DISTINCT FROM 'contratacion_temporal.nombramiento.derivar'
           OR v_agregado->'analisis' IS DISTINCT FROM v_padre->'analisis'
           OR v_agregado->'via_cobertura' IS DISTINCT FROM v_padre->'via_cobertura'
           OR v_agregado->'asignacion' IS DISTINCT FROM v_padre->'asignacion'
           OR v_agregado->'informe_juridico' IS DISTINCT FROM v_padre->'informe_juridico'
           OR v_agregado->'fiscalizacion' IS DISTINCT FROM v_padre->'fiscalizacion'
           OR v_agregado->>'numero_visible' IS DISTINCT FROM
                (v_padre->>'numero_visible')||'-P'||v_derivacion.numero_plaza::text
        THEN RAISE EXCEPTION 'detalle RRHH no disponible' USING ERRCODE='42501'; END IF;
        v_version_instruccion := (v_padre->>'version')::numeric(20,0);
        v_expediente_instruccion_ref := v_derivacion.padre_ref;
        IF v_version_instruccion<6 OR
           pg_catalog.jsonb_array_length(v_padre->'actuaciones')<>v_version_instruccion THEN
           RAISE EXCEPTION 'detalle RRHH no disponible' USING ERRCODE='42501'; END IF;
        v_instruccion := pg_catalog.jsonb_build_object(
          'derivacion_ref',v_derivacion.derivacion_ref,
          'padre_ref',v_derivacion.padre_ref,
          'numero_visible_padre',v_padre->>'numero_visible',
          'oferta_ref',v_derivacion.oferta_ref,
          'numero_plaza',v_derivacion.numero_plaza,
          'recibo_derivacion_ref',v_derivacion.recibo_ref,
          'huella_padre_sha256',v_agregado#>>'{instruccion_compartida,huella_padre_sha256}',
          'creado_en_padre',v_padre->>'creado_en',
          'actualizado_en_padre',v_padre->>'actualizado_en',
          'hitos_padre',(
             SELECT pg_catalog.jsonb_agg(pg_catalog.jsonb_build_object(
               'secuencia',nodo->'secuencia','version_expediente',nodo->'version_expediente',
               'accion_clave',nodo->'accion_clave','realizada_en',nodo->'realizada_en',
               'fase_origen',COALESCE(nodo->>'fase_origen',''),
               'fase_destino',nodo->'fase_destino','estado_origen',nodo->'estado_origen',
               'estado_destino',nodo->'estado_destino') ORDER BY ord)
             FROM pg_catalog.jsonb_array_elements(v_padre->'actuaciones')
               WITH ORDINALITY AS original(nodo,ord)));
    END IF;
    IF pg_catalog.octet_length(v_agregado::text) > 262144
       OR NOT vec_contratacion_temporal.json_rrhh_seguro_v1(v_agregado)
       OR pg_catalog.jsonb_typeof(v_agregado -> 'solicitud') <> 'object'
       OR pg_catalog.jsonb_typeof(
           v_agregado #> '{solicitud,periodo}'
       ) <> 'object'
       OR pg_catalog.jsonb_typeof(v_agregado -> 'actuaciones') <> 'array'
       OR v_agregado ->> 'referencia' IS DISTINCT FROM
          v_fila.expediente_ref
       OR v_agregado ->> 'organizacion_ref' IS DISTINCT FROM
          v_fila.organizacion_ref
       OR v_agregado ->> 'numero_visible' IS DISTINCT FROM
          v_fila.numero_visible
       OR v_agregado ->> 'version' IS DISTINCT FROM
          v_fila.version::text
       OR v_agregado #>> '{flujo,definicion_ref}' IS DISTINCT FROM
          v_fila.flujo_ref
       OR v_agregado #>> '{flujo,version}' IS DISTINCT FROM
          v_fila.flujo_version::text
       OR v_agregado #>> '{flujo,huella_sha256}' IS DISTINCT FROM
          v_fila.flujo_huella_sha256
       OR v_agregado ->> 'fase_actual' IS DISTINCT FROM
          v_fila.fase_clave
       OR v_agregado ->> 'estado_actual' IS DISTINCT FROM
          v_fila.estado_clave THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'detalle RRHH no disponible';
    END IF;

    v_resumen := ROW(
        v_fila.expediente_ref,
        v_fila.organizacion_ref,
        vec_contratacion_temporal.numero_visible_vigente_v1(
            v_fila.expediente_ref, v_fila.numero_visible
        ),
        v_fila.version,
        v_fila.flujo_ref,
        v_fila.flujo_version,
        v_fila.flujo_huella_sha256,
        v_fila.fase_clave,
        v_fila.estado_clave,
        v_fila.centro_ref,
        v_fila.categoria_ref,
        COALESCE(v_fila.modalidad_clave, ''),
        COALESCE(v_fila.unidad_ref, ''),
        v_fila.creado_en,
        v_fila.actualizado_en
    );
    v_solicitud := ROW(
        v_agregado #>> '{solicitud,grupo_subgrupo}',
        v_agregado #>> '{solicitud,motivo_clave}',
        (v_agregado #>> '{solicitud,periodo,inicio}')::timestamptz,
        (v_agregado #>> '{solicitud,periodo,fin}')::timestamptz,
        v_agregado #>> '{solicitud,periodo,causa_fin}'
    );

    v_analisis_presente := v_agregado ? 'analisis';
    IF v_analisis_presente THEN
        IF pg_catalog.jsonb_typeof(v_agregado -> 'analisis') <> 'object'
           OR pg_catalog.jsonb_typeof(
               v_agregado #> '{analisis,periodo}'
           ) <> 'object'
           OR pg_catalog.jsonb_typeof(
               v_agregado #> '{analisis,validacion_rc}'
           ) <> 'object'
           OR pg_catalog.jsonb_typeof(
               v_agregado #> '{analisis,actuacion_registro}'
           ) <> 'object' THEN
            RAISE EXCEPTION USING
                ERRCODE = '42501',
                MESSAGE = 'detalle RRHH no disponible';
        END IF;
        -- Sin coste previsto Go tampoco guarda la fuente del coste: sólo
        -- entonces «hay coste» es false. Con fuente y sin coste sigue NULL y
        -- el canon lo rechaza, como antes de CT180.
        v_coste_presente := COALESCE(
            pg_catalog.jsonb_typeof(
                v_agregado #> '{analisis,coste_previsto}'
            ) = 'object',
            CASE WHEN v_agregado #> '{analisis,fuente_coste_ref}' IS NULL
                 THEN false END
        );
        v_referencia_analisis := (
            v_agregado #>>
            '{analisis,actuacion_registro,version_expediente}'
        )::numeric(20, 0);
        v_analisis := ROW(
            v_agregado #>> '{analisis,modalidad_clave}',
            v_agregado #>> '{analisis,categoria_ref}',
            v_agregado #>> '{analisis,causa_clave}',
            (v_agregado #>> '{analisis,periodo,inicio}')::timestamptz,
            (v_agregado #>> '{analisis,periodo,fin}')::timestamptz,
            (v_agregado #>> '{analisis,porcentaje_jornada}')::smallint,
            v_agregado #>> '{analisis,validacion_rc,resultado}',
            v_coste_presente,
            CASE WHEN v_coste_presente THEN
                (v_agregado #>>
                 '{analisis,coste_previsto,centimos}')::bigint
            ELSE 0::bigint END,
            CASE WHEN v_coste_presente THEN
                v_agregado #>> '{analisis,coste_previsto,moneda}'
            ELSE '' END,
            CASE WHEN v_coste_presente THEN
                v_agregado #>> '{analisis,fuente_coste_ref}'
            ELSE '' END,
            COALESCE(v_agregado #>> '{analisis,observaciones}', ''),
            v_agregado #>> '{analisis,periodo,causa_fin}'
        );
    ELSE
        v_analisis := NULL;
    END IF;

    v_cobertura_presente := v_agregado ? 'via_cobertura';
    IF v_cobertura_presente THEN
        IF pg_catalog.jsonb_typeof(
               v_agregado -> 'via_cobertura'
           ) <> 'object' THEN
            RAISE EXCEPTION USING
                ERRCODE = '42501',
                MESSAGE = 'detalle RRHH no disponible';
        END IF;
        v_decision_gobernada :=
            pg_catalog.jsonb_typeof(
                v_agregado #>
                '{via_cobertura,decision_gobernada}'
            ) = 'object';
        v_comprobaciones := ARRAY[]::vec_contratacion_temporal
            .comprobacion_operativa_rrhh_v1[];
        IF v_decision_gobernada THEN
            IF NOT v_analisis_presente THEN
                RAISE EXCEPTION USING
                    ERRCODE = '42501',
                    MESSAGE = 'detalle RRHH no disponible';
            END IF;
            v_referencia_cobertura := (
                v_agregado #>>
                '{via_cobertura,decision_gobernada,actuacion,version_expediente}'
            )::numeric(20, 0);
            SELECT ARRAY(
                SELECT ROW(
                    evidencia.comprobacion_clave,
                    evidencia.comprobacion_resultado
                )::vec_contratacion_temporal.comprobacion_operativa_rrhh_v1
                  FROM vec_contratacion_temporal.consumo_cobertura_lote lote
                  JOIN vec_contratacion_temporal.consumo_cobertura_evidencia evidencia
                    ON evidencia.lote_ref = lote.lote_ref
                 WHERE lote.preparacion_c1_ref = v_agregado #>>
                       '{via_cobertura,decision_gobernada,preparacion_evidencias_ref}'
                   AND lote.expediente_ref = v_expediente_instruccion_ref
                   AND lote.version_expediente = v_referencia_analisis
                   AND evidencia.expediente_ref = lote.expediente_ref
                   AND evidencia.version_expediente = lote.version_expediente
                 ORDER BY evidencia.posicion
            ) INTO v_comprobaciones;
            IF pg_catalog.cardinality(v_comprobaciones) = 0
               OR pg_catalog.cardinality(v_comprobaciones) > 32 THEN
                RAISE EXCEPTION USING
                    ERRCODE = '42501',
                    MESSAGE = 'detalle RRHH no disponible';
            END IF;
        ELSE
            IF NOT v_analisis_presente
               OR pg_catalog.jsonb_typeof(
                   v_agregado #> '{via_cobertura,comprobaciones}'
               ) <> 'array' THEN
                RAISE EXCEPTION USING
                    ERRCODE = '42501',
                    MESSAGE = 'detalle RRHH no disponible';
            END IF;
            v_referencia_cobertura := v_referencia_analisis + 1;
            v_total := pg_catalog.jsonb_array_length(
                v_agregado #> '{via_cobertura,comprobaciones}'
            );
            FOR v_indice IN 0..v_total - 1 LOOP
                v_nodo := (
                    v_agregado #> '{via_cobertura,comprobaciones}'
                ) -> v_indice;
                v_comprobaciones := pg_catalog.array_append(
                    v_comprobaciones,
                    ROW(
                        v_nodo ->> 'clave',
                        v_nodo ->> 'resultado'
                    )::vec_contratacion_temporal
                        .comprobacion_operativa_rrhh_v1
                );
            END LOOP;
        END IF;
        v_cobertura := ROW(
            v_agregado #>> '{via_cobertura,via_clave}',
            v_decision_gobernada,
            CASE WHEN v_decision_gobernada THEN '' ELSE
                v_agregado #>> '{via_cobertura,procedimiento_ref}' END,
            CASE WHEN v_decision_gobernada THEN '' ELSE
                COALESCE(
                    v_agregado #>> '{via_cobertura,bolsa_ref}', ''
                ) END,
            v_comprobaciones
        );
    ELSE
        v_cobertura := NULL;
    END IF;

    v_asignacion_presente := v_agregado ? 'asignacion';
    IF v_asignacion_presente THEN
        IF pg_catalog.jsonb_typeof(v_agregado -> 'asignacion') <> 'object'
           OR pg_catalog.jsonb_typeof(
               v_agregado #> '{asignacion,actuacion_registro}'
           ) <> 'object' THEN
            RAISE EXCEPTION USING
                ERRCODE = '42501',
                MESSAGE = 'detalle RRHH no disponible';
        END IF;
        v_referencia_asignacion := (
            v_agregado #>>
            '{asignacion,actuacion_registro,version_expediente}'
        )::numeric(20, 0);
        v_asignacion := ROW(
            v_agregado #>> '{asignacion,unidad_ref}',
            (v_agregado #>> '{asignacion,asignada_en}')::timestamptz,
            COALESCE(v_agregado #>> '{asignacion,motivo_clave}', '')
        );
    ELSE
        v_asignacion := NULL;
    END IF;

    v_fiscalizacion_presente := v_agregado ? 'fiscalizacion';
    v_subsanacion_presente := false;
    IF v_fiscalizacion_presente THEN
        IF NOT v_asignacion_presente
           OR pg_catalog.jsonb_typeof(v_agregado -> 'fiscalizacion') <> 'object'
           OR pg_catalog.jsonb_typeof(
               v_agregado #> '{fiscalizacion,actuacion_registro}'
           ) <> 'object'
           OR pg_catalog.jsonb_typeof(
               v_agregado #> '{fiscalizacion,resultado}'
           ) <> 'string'
           OR pg_catalog.jsonb_typeof(
               v_agregado #> '{fiscalizacion,fiscalizada_en}'
           ) <> 'string'
           OR pg_catalog.jsonb_typeof(
               v_agregado #> '{fiscalizacion,observaciones}'
           ) NOT IN ('string', 'null') THEN
            RAISE EXCEPTION USING ERRCODE = '42501',
                MESSAGE = 'detalle RRHH no disponible';
        END IF;
        v_referencia_fiscalizacion := (v_agregado #>>
            '{fiscalizacion,actuacion_registro,secuencia}')::numeric(20, 0);
        IF v_referencia_fiscalizacion <= v_referencia_asignacion
           OR v_referencia_fiscalizacion > v_version_instruccion
           OR v_agregado #>> '{fiscalizacion,resultado}' NOT IN (
                'favorable', 'favorable_con_observaciones', 'desfavorable'
           )
           OR COALESCE(v_agregado #>> '{fiscalizacion,observaciones}', '')
              ~ '[\x00-\x1F]'
           OR pg_catalog.char_length(
                COALESCE(v_agregado #>> '{fiscalizacion,observaciones}', '')
           ) > 2000
           OR (v_agregado #>> '{fiscalizacion,resultado}' = 'favorable'
               AND COALESCE(v_agregado #>> '{fiscalizacion,observaciones}', '') <> '')
           OR (v_agregado #>> '{fiscalizacion,resultado}' <> 'favorable'
               AND pg_catalog.char_length(
                   COALESCE(v_agregado #>> '{fiscalizacion,observaciones}', '')
               ) = 0) THEN
            RAISE EXCEPTION USING ERRCODE = '42501',
                MESSAGE = 'detalle RRHH no disponible';
        END IF;
        v_fiscalizacion := ROW(
            v_agregado #>> '{fiscalizacion,resultado}',
            CASE WHEN v_agregado #>> '{fiscalizacion,resultado}' = 'favorable'
                 THEN '' ELSE 'observaciones_fiscalizacion' END,
            COALESCE(v_agregado #>> '{fiscalizacion,observaciones}', ''),
            (v_agregado #>> '{fiscalizacion,fiscalizada_en}')::timestamptz,
            NULL::timestamptz, NULL::text
        );
        IF v_agregado #>> '{fiscalizacion,resultado}' = 'desfavorable' THEN
            IF pg_catalog.jsonb_typeof(
                   v_agregado #> '{fiscalizacion,retorno}'
               ) <> 'object'
               OR pg_catalog.jsonb_typeof(
                   v_agregado #> '{fiscalizacion,retorno,retorno_ref}'
               ) <> 'string' THEN
                RAISE EXCEPTION USING ERRCODE = '42501',
                    MESSAGE = 'detalle RRHH no disponible';
            END IF;
            SELECT pg_catalog.count(*)
              INTO v_subsanaciones
              FROM pg_catalog.jsonb_array_elements(
                   v_agregado -> 'actuaciones'
              ) nodo
             WHERE nodo ->> 'accion_clave' =
                   'contratacion_temporal.subsanacion_reparos.registrar'
               AND nodo ->> 'retorno_ref' =
                   v_agregado #>> '{fiscalizacion,retorno,retorno_ref}';
            IF v_subsanaciones > 1 THEN
                RAISE EXCEPTION USING ERRCODE = '42501',
                    MESSAGE = 'detalle RRHH no disponible';
            END IF;
            IF v_subsanaciones = 1 THEN
                SELECT nodo INTO STRICT v_subsanacion_nodo
                  FROM pg_catalog.jsonb_array_elements(
                       v_agregado -> 'actuaciones'
                  ) nodo
                 WHERE nodo ->> 'accion_clave' =
                       'contratacion_temporal.subsanacion_reparos.registrar'
                   AND nodo ->> 'retorno_ref' =
                       v_agregado #>> '{fiscalizacion,retorno,retorno_ref}';
                IF pg_catalog.jsonb_typeof(
                       v_subsanacion_nodo -> 'realizada_en'
                   ) <> 'string'
                   OR pg_catalog.jsonb_typeof(
                       v_subsanacion_nodo -> 'observaciones'
                   ) <> 'string'
                   OR pg_catalog.char_length(
                       v_subsanacion_nodo ->> 'observaciones'
                   ) NOT BETWEEN 1 AND 2000
                   OR v_subsanacion_nodo ->> 'observaciones' ~ '[\x00-\x1F]' THEN
                    RAISE EXCEPTION USING ERRCODE = '42501',
                        MESSAGE = 'detalle RRHH no disponible';
                END IF;
                v_referencia_subsanacion := (v_subsanacion_nodo ->>
                    'secuencia')::numeric(20, 0);
                IF v_referencia_subsanacion <= v_referencia_fiscalizacion
                   OR v_referencia_subsanacion > v_version_instruccion THEN
                    RAISE EXCEPTION USING ERRCODE = '42501',
                        MESSAGE = 'detalle RRHH no disponible';
                END IF;
                v_subsanacion_presente := true;
                v_fiscalizacion.subsanacion_registrada_en :=
                    (v_subsanacion_nodo ->> 'realizada_en')::timestamptz;
                v_fiscalizacion.subsanacion_texto :=
                    v_subsanacion_nodo ->> 'observaciones';
            END IF;
        END IF;
    ELSE
        v_fiscalizacion := NULL;
    END IF;

    v_hitos := ARRAY[]::vec_contratacion_temporal
        .hito_expediente_rrhh_v1[];
    v_total := pg_catalog.jsonb_array_length(
        v_agregado -> 'actuaciones'
    );
    IF v_total < 1 OR v_total::numeric + (CASE WHEN v_instruccion IS NULL THEN 0 ELSE 5 END) <> v_fila.version THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'detalle RRHH no disponible';
    END IF;
    FOR v_indice IN 0..v_total - 1 LOOP
        v_nodo := (v_agregado -> 'actuaciones') -> v_indice;
        v_hitos := pg_catalog.array_append(
            v_hitos,
            ROW(
                (v_nodo ->> 'secuencia')::numeric(20, 0),
                (v_nodo ->> 'version_expediente')::numeric(20, 0),
                v_nodo ->> 'accion_clave',
                (v_nodo ->> 'realizada_en')::timestamptz,
                COALESCE(v_nodo ->> 'fase_origen', ''),
                v_nodo ->> 'fase_destino',
                v_nodo ->> 'estado_origen',
                v_nodo ->> 'estado_destino'
            )::vec_contratacion_temporal.hito_expediente_rrhh_v1
        );
    END LOOP;

    v_detalle := ROW(
        v_resumen,
        v_solicitud,
        v_analisis_presente,
        v_analisis,
        v_referencia_analisis,
        v_cobertura_presente,
        v_cobertura,
        v_referencia_cobertura,
        v_asignacion_presente,
        v_asignacion,
        v_referencia_asignacion,
        v_hitos,
        v_fiscalizacion_presente,
        v_fiscalizacion,
        v_referencia_fiscalizacion,
        v_referencia_subsanacion,
        v_instruccion
    );

    -- Reutiliza la validación del canon nominal sin volver a consultar ni
    -- conservar el agregado. El instante de la versión basta para esta
    -- validación estructural; el motor canoniza después su instante real.
    PERFORM
        vec_contratacion_temporal.canon_contenido_detalle_rrhh_v1(
            v_resumen.actualizado_en,
            v_detalle
        );
    RETURN ROW(v_detalle)::
        vec_contratacion_temporal.materializacion_detalle_rrhh_v1;
EXCEPTION
    WHEN SQLSTATE '40001' OR SQLSTATE '40P01'
      OR SQLSTATE '55P03' OR SQLSTATE '57014' THEN
        RAISE;
    WHEN OTHERS THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'detalle RRHH no disponible';
END
$function$

;
CREATE OR REPLACE FUNCTION vec_contratacion_temporal.canon_contenido_detalle_rrhh_v1(p_generada_en timestamp with time zone, p_entrada vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1)
 RETURNS bytea
 LANGUAGE plpgsql
 IMMUTABLE PARALLEL SAFE STRICT
 SET search_path=pg_catalog,pg_temp
AS $function$
DECLARE
    v_canon bytea :=
        vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
            (CASE WHEN p_entrada.instruccion_compartida IS NOT NULL THEN 'VEC-CT-CONTENIDO-DETALLE-RRHH-V5'
                  WHEN (p_entrada.solicitud).periodo_causa_fin IS NOT NULL
                       OR (p_entrada.analisis).periodo_causa_fin IS NOT NULL
                THEN 'VEC-CT-CONTENIDO-DETALLE-RRHH-V4'
                ELSE 'VEC-CT-CONTENIDO-DETALLE-RRHH-V3' END)
                || pg_catalog.chr(10)
        );
    v_resumen vec_contratacion_temporal.resumen_publicacion_rrhh_v1;
    v_solicitud vec_contratacion_temporal.solicitud_operativa_rrhh_v1;
    v_analisis vec_contratacion_temporal.analisis_operativo_rrhh_v1;
    v_cobertura vec_contratacion_temporal.cobertura_operativa_rrhh_v1;
    v_asignacion vec_contratacion_temporal.asignacion_operativa_rrhh_v1;
    v_fiscalizacion vec_contratacion_temporal.fiscalizacion_operativa_rrhh_v1;
    v_hito vec_contratacion_temporal.hito_expediente_rrhh_v1;
    v_anterior vec_contratacion_temporal.hito_expediente_rrhh_v1;
    v_comprobacion
        vec_contratacion_temporal.comprobacion_operativa_rrhh_v1;
    v_campos text[];
    v_campo text;
    v_claves_vistas text[] := ARRAY[]::text[];
    v_indice integer;
    v_total_hitos integer;
    v_total_instruccion integer;
    v_version_instruccion numeric(20,0);
    v_hitos_instruccion vec_contratacion_temporal.hito_expediente_rrhh_v1[];
    v_instruccion jsonb := p_entrada.instruccion_compartida;
    v_derivado boolean := p_entrada.instruccion_compartida IS NOT NULL;
    v_total_comprobaciones integer;
    v_mascara smallint := 0;
    v_analisis_nulo boolean;
    v_cobertura_nula boolean;
    v_asignacion_nula boolean;
    v_fiscalizacion_nula boolean;
    v_subsanacion_presente boolean;
    v_utc timestamp;
BEGIN
    IF p_entrada.resumen IS NOT DISTINCT FROM NULL
       OR p_entrada.solicitud IS NOT DISTINCT FROM NULL
       OR p_entrada.analisis_presente IS NULL
       OR p_entrada.referencia_analisis IS NULL
       OR p_entrada.cobertura_presente IS NULL
       OR p_entrada.referencia_cobertura IS NULL
       OR p_entrada.asignacion_presente IS NULL
       OR p_entrada.referencia_asignacion IS NULL
       OR p_entrada.hitos IS NULL
       OR p_entrada.fiscalizacion_presente IS NULL
       OR p_entrada.referencia_fiscalizacion IS NULL
       OR p_entrada.referencia_subsanacion IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'contenido de detalle RRHH inválido';
    END IF;
    v_resumen := p_entrada.resumen;
    v_solicitud := p_entrada.solicitud;
    v_analisis_nulo := p_entrada.analisis IS NOT DISTINCT FROM NULL;
    v_cobertura_nula := p_entrada.cobertura IS NOT DISTINCT FROM NULL;
    v_asignacion_nula := p_entrada.asignacion IS NOT DISTINCT FROM NULL;
    v_fiscalizacion_nula := p_entrada.fiscalizacion IS NOT DISTINCT FROM NULL;

    PERFORM vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
        p_generada_en
    );
    v_canon := v_canon
        || vec_contratacion_temporal.canon_resumen_publicacion_rrhh_v1(
            v_resumen
        );
    IF p_generada_en < v_resumen.actualizado_en
       OR v_solicitud.grupo_subgrupo IS NULL
       OR v_solicitud.grupo_subgrupo !~
          '^[A-Z][A-Z0-9/+.-]{0,19}$'
       OR v_solicitud.motivo_clave IS NULL
       OR v_solicitud.motivo_clave !~ '^[a-z][a-z0-9._-]{1,79}$'
       OR v_solicitud.periodo_inicio IS NULL
       OR (v_solicitud.periodo_fin IS NULL
           AND coalesce(v_solicitud.periodo_causa_fin,'') !~
               '^[a-z][a-z0-9._-]{1,79}$')
       OR (v_solicitud.periodo_fin IS NOT NULL
           AND v_solicitud.periodo_causa_fin IS NOT NULL)
       OR v_solicitud.periodo_fin < v_solicitud.periodo_inicio THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'contenido de detalle RRHH inválido';
    END IF;
    PERFORM vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
        v_solicitud.periodo_inicio
    );
    PERFORM vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
        v_solicitud.periodo_fin
    );
    v_utc := v_solicitud.periodo_inicio AT TIME ZONE 'UTC';
    IF extract(hour FROM v_utc) <> 0
       OR extract(minute FROM v_utc) <> 0
       OR extract(second FROM v_utc) <> 0 THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'contenido de detalle RRHH inválido';
    END IF;
    v_utc := v_solicitud.periodo_fin AT TIME ZONE 'UTC';
    IF extract(hour FROM v_utc) <> 0
       OR extract(minute FROM v_utc) <> 0
       OR extract(second FROM v_utc) <> 0 THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'contenido de detalle RRHH inválido';
    END IF;

    v_total_hitos := pg_catalog.cardinality(p_entrada.hitos);
    IF v_total_hitos < 1
       OR v_total_hitos::numeric + (CASE WHEN v_derivado THEN 5 ELSE 0 END) <> v_resumen.version
       OR pg_catalog.array_ndims(p_entrada.hitos) <> 1
       OR pg_catalog.array_lower(p_entrada.hitos, 1) <> 1 THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'contenido de detalle RRHH inválido';
    END IF;
    IF pg_catalog.array_position(p_entrada.hitos, NULL) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'contenido de detalle RRHH inválido';
    END IF;
    v_hitos_instruccion := p_entrada.hitos;
    v_total_instruccion := v_total_hitos;
    v_version_instruccion := v_resumen.version;
    IF v_derivado THEN
        IF pg_catalog.jsonb_typeof(v_instruccion)<>'object'
           OR NOT (p_entrada.analisis_presente AND p_entrada.cobertura_presente
                   AND p_entrada.asignacion_presente AND p_entrada.fiscalizacion_presente)
           OR (SELECT count(*) FROM pg_catalog.jsonb_object_keys(v_instruccion))<>10
           OR pg_catalog.jsonb_typeof(v_instruccion->'hitos_padre')<>'array'
           OR COALESCE(v_instruccion->>'padre_ref','') !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
           OR COALESCE(v_instruccion->>'derivacion_ref','') !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
           OR COALESCE(v_instruccion->>'recibo_derivacion_ref','') !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
           OR COALESCE(v_instruccion->>'oferta_ref','') !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
           OR COALESCE(v_instruccion->>'huella_padre_sha256','') !~ '^[0-9a-f]{64}$'
           OR v_instruccion->>'huella_padre_sha256'=pg_catalog.repeat('0',64)
           OR COALESCE(v_instruccion->>'numero_plaza','') !~ '^(100|[1-9][0-9]?)$'
           OR v_resumen.numero_visible IS DISTINCT FROM
              (v_instruccion->>'numero_visible_padre')||'-P'||(v_instruccion->>'numero_plaza')
           OR v_instruccion->>'padre_ref' = v_resumen.expediente_ref
           OR v_instruccion->>'creado_en_padre' IS NULL
           OR v_instruccion->>'actualizado_en_padre' IS NULL
           OR (v_instruccion->>'creado_en_padre')::timestamptz >
              (v_instruccion->>'actualizado_en_padre')::timestamptz
           OR (v_instruccion->>'actualizado_en_padre')::timestamptz > v_resumen.creado_en
        THEN RAISE EXCEPTION 'contenido de detalle RRHH inválido' USING ERRCODE='22023'; END IF;
        SELECT pg_catalog.array_agg(ROW(
          (nodo->>'secuencia')::numeric(20,0),
          (nodo->>'version_expediente')::numeric(20,0),
          nodo->>'accion_clave',(nodo->>'realizada_en')::timestamptz,
          COALESCE(nodo->>'fase_origen',''),nodo->>'fase_destino',
          nodo->>'estado_origen',nodo->>'estado_destino')::
          vec_contratacion_temporal.hito_expediente_rrhh_v1 ORDER BY ord)
          INTO v_hitos_instruccion
          FROM pg_catalog.jsonb_array_elements(v_instruccion->'hitos_padre')
            WITH ORDINALITY AS original(nodo,ord);
        v_total_instruccion := pg_catalog.cardinality(v_hitos_instruccion);
        v_version_instruccion := v_total_instruccion;
        IF v_total_instruccion<6 OR v_total_instruccion>1500 OR
           pg_catalog.array_position(v_hitos_instruccion,NULL) IS NOT NULL THEN
           RAISE EXCEPTION 'contenido de detalle RRHH inválido' USING ERRCODE='22023'; END IF;
    END IF;

    IF p_entrada.analisis_presente THEN
        IF v_analisis_nulo
           OR p_entrada.referencia_analisis
              NOT BETWEEN 2 AND v_version_instruccion THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'contenido de detalle RRHH inválido';
        END IF;
        v_mascara := v_mascara | 1;
        v_analisis := p_entrada.analisis;
    ELSIF NOT v_analisis_nulo OR p_entrada.referencia_analisis <> 0
          OR v_resumen.modalidad_clave <> '' THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'contenido de detalle RRHH inválido';
    END IF;
    IF p_entrada.cobertura_presente THEN
        IF v_cobertura_nula OR NOT p_entrada.analisis_presente
           OR p_entrada.referencia_cobertura
              NOT BETWEEN 2 AND v_version_instruccion
           OR p_entrada.referencia_cobertura
              <= p_entrada.referencia_analisis THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'contenido de detalle RRHH inválido';
        END IF;
        v_mascara := v_mascara | 2;
        v_cobertura := p_entrada.cobertura;
    ELSIF NOT v_cobertura_nula OR p_entrada.referencia_cobertura <> 0 THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'contenido de detalle RRHH inválido';
    END IF;
    IF p_entrada.asignacion_presente THEN
        IF v_asignacion_nula OR NOT p_entrada.cobertura_presente
           OR p_entrada.referencia_asignacion
              NOT BETWEEN 2 AND v_version_instruccion
           OR p_entrada.referencia_asignacion
              <= p_entrada.referencia_cobertura THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'contenido de detalle RRHH inválido';
        END IF;
        v_mascara := v_mascara | 4;
        v_asignacion := p_entrada.asignacion;
    ELSIF NOT v_asignacion_nula OR p_entrada.referencia_asignacion <> 0
          OR v_resumen.unidad_ref <> '' THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'contenido de detalle RRHH inválido';
    END IF;
    IF p_entrada.fiscalizacion_presente THEN
        IF v_fiscalizacion_nula OR NOT p_entrada.asignacion_presente
           OR p_entrada.referencia_fiscalizacion
              NOT BETWEEN 1 AND v_version_instruccion
           OR p_entrada.referencia_fiscalizacion
              <= p_entrada.referencia_asignacion
           OR p_entrada.referencia_subsanacion < 0
           OR p_entrada.referencia_subsanacion > v_version_instruccion THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'contenido de detalle RRHH inválido';
        END IF;
        v_mascara := v_mascara | 8;
        v_fiscalizacion := p_entrada.fiscalizacion;
    ELSIF NOT v_fiscalizacion_nula
          OR p_entrada.referencia_fiscalizacion <> 0
          OR p_entrada.referencia_subsanacion <> 0 THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'contenido de detalle RRHH inválido';
    END IF;
    IF v_mascara NOT IN (0, 1, 3, 7, 15) THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'contenido de detalle RRHH inválido';
    END IF;

    IF p_entrada.analisis_presente THEN
        IF v_analisis.modalidad_clave IS NULL
           OR v_analisis.modalidad_clave !~
              '^[a-z][a-z0-9._-]{1,79}$'
           OR v_analisis.modalidad_clave <> v_resumen.modalidad_clave
           OR v_analisis.categoria_ref IS NULL
           OR v_analisis.categoria_ref !~
              '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
           OR v_analisis.categoria_ref <> v_resumen.categoria_ref
           OR v_analisis.causa_clave IS NULL
           OR v_analisis.causa_clave !~ '^[a-z][a-z0-9._-]{1,79}$'
           OR v_analisis.periodo_inicio IS NULL
           OR (v_analisis.periodo_fin IS NULL
               AND coalesce(v_analisis.periodo_causa_fin,'') !~
                   '^[a-z][a-z0-9._-]{1,79}$')
           OR (v_analisis.periodo_fin IS NOT NULL
               AND v_analisis.periodo_causa_fin IS NOT NULL)
           OR v_analisis.periodo_fin < v_analisis.periodo_inicio
           OR v_analisis.porcentaje_jornada IS NULL
           OR v_analisis.porcentaje_jornada NOT BETWEEN 1 AND 10000
           OR v_analisis.resultado_rc IS NULL
           OR v_analisis.resultado_rc NOT IN (
               'validada', 'no_requerida', 'rechazada'
           )
           OR v_analisis.coste_presente IS NULL
           OR v_analisis.coste_centimos IS NULL
           OR v_analisis.coste_moneda IS NULL
           OR v_analisis.fuente_coste_ref IS NULL
           OR v_analisis.observaciones IS NULL
           OR pg_catalog.length(v_analisis.observaciones) > 4000 THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'contenido de detalle RRHH inválido';
        END IF;
        PERFORM vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
            v_analisis.periodo_inicio
        );
        PERFORM vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
            v_analisis.periodo_fin
        );
        v_utc := v_analisis.periodo_inicio AT TIME ZONE 'UTC';
        IF extract(hour FROM v_utc) <> 0
           OR extract(minute FROM v_utc) <> 0
           OR extract(second FROM v_utc) <> 0 THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'contenido de detalle RRHH inválido';
        END IF;
        v_utc := v_analisis.periodo_fin AT TIME ZONE 'UTC';
        IF extract(hour FROM v_utc) <> 0
           OR extract(minute FROM v_utc) <> 0
           OR extract(second FROM v_utc) <> 0 THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'contenido de detalle RRHH inválido';
        END IF;
        IF v_analisis.coste_presente THEN
            IF v_analisis.coste_centimos <= 0
               OR v_analisis.coste_moneda <> 'EUR'
               OR v_analisis.fuente_coste_ref !~
                  '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' THEN
                RAISE EXCEPTION USING ERRCODE = '22023',
                    MESSAGE = 'contenido de detalle RRHH inválido';
            END IF;
        ELSIF v_analisis.coste_centimos <> 0
              OR v_analisis.coste_moneda <> ''
              OR v_analisis.fuente_coste_ref <> '' THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'contenido de detalle RRHH inválido';
        END IF;
    END IF;

    IF p_entrada.cobertura_presente THEN
        IF v_cobertura.via_clave IS NULL
           OR v_cobertura.via_clave !~ '^[a-z][a-z0-9._-]{1,79}$'
           OR v_cobertura.decision_gobernada IS NULL
           OR v_cobertura.procedimiento_ref IS NULL
           OR v_cobertura.bolsa_ref IS NULL
           OR v_cobertura.comprobaciones IS NULL THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'contenido de detalle RRHH inválido';
        END IF;
        v_total_comprobaciones :=
            pg_catalog.cardinality(v_cobertura.comprobaciones);
        IF v_cobertura.decision_gobernada THEN
            IF v_cobertura.procedimiento_ref <> ''
               OR v_cobertura.bolsa_ref <> '' THEN
                RAISE EXCEPTION USING ERRCODE = '22023',
                    MESSAGE = 'contenido de detalle RRHH inválido';
            END IF;
            IF v_total_comprobaciones = 0 THEN
                IF pg_catalog.array_ndims(v_cobertura.comprobaciones)
                   IS NOT NULL THEN
                    RAISE EXCEPTION USING ERRCODE = '22023',
                        MESSAGE = 'contenido de detalle RRHH inválido';
                END IF;
            ELSIF v_total_comprobaciones NOT BETWEEN 1 AND 32
               OR pg_catalog.array_ndims(v_cobertura.comprobaciones) <> 1
               OR pg_catalog.array_lower(v_cobertura.comprobaciones, 1) <> 1 THEN
                RAISE EXCEPTION USING ERRCODE = '22023',
                    MESSAGE = 'contenido de detalle RRHH inválido';
            END IF;
        ELSE
            IF v_cobertura.procedimiento_ref !~
               '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
               OR v_cobertura.bolsa_ref !~
                  '^$|^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
               OR v_total_comprobaciones NOT BETWEEN 1 AND 32
               OR pg_catalog.array_ndims(
                   v_cobertura.comprobaciones
               ) <> 1
               OR pg_catalog.array_lower(
                   v_cobertura.comprobaciones, 1
               ) <> 1 THEN
                RAISE EXCEPTION USING ERRCODE = '22023',
                    MESSAGE = 'contenido de detalle RRHH inválido';
            END IF;
        END IF;
        IF v_total_comprobaciones > 0 THEN
            IF pg_catalog.array_position(v_cobertura.comprobaciones, NULL) IS NOT NULL THEN
                RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'contenido de detalle RRHH inválido';
            END IF;
            FOR v_indice IN 1..v_total_comprobaciones LOOP
                v_comprobacion := v_cobertura.comprobaciones[v_indice];
                IF v_comprobacion.clave IS NULL OR v_comprobacion.clave !~ '^[a-z][a-z0-9._-]{1,79}$'
                   OR v_comprobacion.resultado IS NULL OR v_comprobacion.resultado NOT IN ('afirmativa', 'negativa', 'no_aplica', 'no_consta')
                   OR v_comprobacion.clave = ANY(v_claves_vistas) THEN
                    RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'contenido de detalle RRHH inválido';
                END IF;
                v_claves_vistas := pg_catalog.array_append(v_claves_vistas, v_comprobacion.clave);
            END LOOP;
        END IF;
    END IF;

    IF p_entrada.asignacion_presente THEN
        IF v_asignacion.unidad_ref IS NULL
           OR v_asignacion.unidad_ref !~
              '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
           OR v_asignacion.unidad_ref <> v_resumen.unidad_ref
           OR v_asignacion.asignada_en IS NULL
           OR v_asignacion.asignada_en < (CASE WHEN v_derivado THEN (v_instruccion->>'creado_en_padre')::timestamptz ELSE v_resumen.creado_en END)
           OR v_asignacion.asignada_en > v_resumen.actualizado_en
           OR v_asignacion.motivo_clave IS NULL
           OR v_asignacion.motivo_clave !~
              '^$|^[a-z][a-z0-9._-]{1,79}$' THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'contenido de detalle RRHH inválido';
        END IF;
        PERFORM vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
            v_asignacion.asignada_en
        );
    END IF;

    FOR v_indice IN 1..v_total_instruccion LOOP
        v_hito := v_hitos_instruccion[v_indice];
        IF v_hito.secuencia IS NULL
           OR v_hito.secuencia <> v_indice
           OR v_hito.version_expediente IS NULL
           OR v_hito.version_expediente <> v_indice
           OR v_hito.accion_clave IS NULL
           OR v_hito.accion_clave !~ '^[a-z][a-z0-9._-]{1,79}$'
           OR v_hito.realizada_en IS NULL
           OR v_hito.realizada_en > v_resumen.actualizado_en
           OR v_hito.fase_origen IS NULL
           OR v_hito.fase_origen !~
              '^$|^[a-z][a-z0-9._-]{1,79}$'
           OR v_hito.fase_destino IS NULL
           OR v_hito.fase_destino !~ '^[a-z][a-z0-9._-]{1,79}$'
           OR v_hito.estado_origen IS NULL
           OR v_hito.estado_origen NOT IN (
               'pendiente', 'en_curso', 'espera_externa',
               'completado', 'incidencia', 'cancelado'
           )
           OR v_hito.estado_destino IS NULL
           OR v_hito.estado_destino NOT IN (
               'pendiente', 'en_curso', 'espera_externa',
               'completado', 'incidencia', 'cancelado'
           ) THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'contenido de detalle RRHH inválido';
        END IF;
        PERFORM vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
            v_hito.realizada_en
        );
        IF p_entrada.fiscalizacion_presente
           AND v_indice = p_entrada.referencia_fiscalizacion THEN
            IF v_hito.accion_clave <>
                   'contratacion_temporal.fiscalizacion.registrar'
               OR v_hito.realizada_en <> v_fiscalizacion.registrada_en THEN
                RAISE EXCEPTION USING ERRCODE = '22023',
                    MESSAGE = 'contenido de detalle RRHH inválido';
            END IF;
        END IF;
        IF p_entrada.fiscalizacion_presente
           AND p_entrada.referencia_subsanacion <> 0
           AND v_indice = p_entrada.referencia_subsanacion THEN
            IF v_hito.accion_clave <>
                   'contratacion_temporal.subsanacion_reparos.registrar'
               OR v_hito.realizada_en <>
                  v_fiscalizacion.subsanacion_registrada_en THEN
                RAISE EXCEPTION USING ERRCODE = '22023',
                    MESSAGE = 'contenido de detalle RRHH inválido';
            END IF;
        END IF;
        IF v_indice = 1 THEN
            IF v_hito.fase_origen <> ''
               OR v_hito.estado_origen <> 'pendiente' THEN
                RAISE EXCEPTION USING ERRCODE = '22023',
                    MESSAGE = 'contenido de detalle RRHH inválido';
            END IF;
        ELSIF v_hito.fase_origen <> v_anterior.fase_destino
              OR v_hito.estado_origen <> v_anterior.estado_destino
              OR v_hito.realizada_en < v_anterior.realizada_en THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'contenido de detalle RRHH inválido';
        END IF;
        v_anterior := v_hito;
    END LOOP;
    IF (NOT v_derivado AND (v_anterior.fase_destino <> v_resumen.fase_clave
       OR v_anterior.estado_destino <> v_resumen.estado_clave
       OR v_anterior.realizada_en <> v_resumen.actualizado_en))
       OR (v_derivado AND (v_anterior.fase_destino <> 'fiscalizacion'
       OR v_anterior.estado_destino <> 'en_curso'
       OR v_anterior.realizada_en <>
           (v_instruccion->>'actualizado_en_padre')::timestamptz
       OR v_hitos_instruccion[1].realizada_en <>
           (v_instruccion->>'creado_en_padre')::timestamptz))
       OR (p_entrada.asignacion_presente AND
           (v_hitos_instruccion[p_entrada.referencia_asignacion::integer]).realizada_en
           <> v_asignacion.asignada_en) THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'contenido de detalle RRHH inválido';
    END IF;

    IF v_derivado THEN
        v_anterior := NULL;
        FOR v_indice IN 1..v_total_hitos LOOP
            v_hito := p_entrada.hitos[v_indice];
            IF v_hito.secuencia IS DISTINCT FROM v_indice+5
               OR v_hito.version_expediente IS DISTINCT FROM v_indice+5
               OR v_hito.accion_clave !~ '^[a-z][a-z0-9._-]{1,79}$'
               OR v_hito.realizada_en IS NULL
               OR v_hito.realizada_en > v_resumen.actualizado_en
               OR v_hito.fase_destino !~ '^[a-z][a-z0-9._-]{1,79}$'
               OR v_hito.estado_origen NOT IN ('pendiente','en_curso','espera_externa','completado','incidencia','cancelado')
               OR v_hito.estado_destino NOT IN ('pendiente','en_curso','espera_externa','completado','incidencia','cancelado')
            THEN RAISE EXCEPTION 'contenido de detalle RRHH inválido' USING ERRCODE='22023'; END IF;
            PERFORM vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(v_hito.realizada_en);
            IF v_indice=1 THEN
                IF v_hito.accion_clave<>'contratacion_temporal.nombramiento.derivar'
                   OR v_hito.fase_origen<>'fiscalizacion' OR v_hito.estado_origen<>'en_curso'
                   OR v_hito.fase_destino<>'fiscalizacion' OR v_hito.estado_destino<>'en_curso'
                   OR v_hito.realizada_en<>v_resumen.creado_en
                THEN RAISE EXCEPTION 'contenido de detalle RRHH inválido' USING ERRCODE='22023'; END IF;
            ELSIF v_hito.fase_origen<>v_anterior.fase_destino
                OR v_hito.estado_origen<>v_anterior.estado_destino
                OR v_hito.realizada_en<v_anterior.realizada_en THEN
                RAISE EXCEPTION 'contenido de detalle RRHH inválido' USING ERRCODE='22023';
            END IF;
            v_anterior:=v_hito;
        END LOOP;
        IF v_anterior.fase_destino<>v_resumen.fase_clave
           OR v_anterior.estado_destino<>v_resumen.estado_clave
           OR v_anterior.realizada_en<>v_resumen.actualizado_en THEN
           RAISE EXCEPTION 'contenido de detalle RRHH inválido' USING ERRCODE='22023'; END IF;
    END IF;

    v_campos := ARRAY[
        v_solicitud.grupo_subgrupo,
        v_solicitud.motivo_clave,
        vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
            v_solicitud.periodo_inicio
        ),
        CASE WHEN v_solicitud.periodo_fin IS NOT NULL THEN
            (CASE WHEN v_derivado OR (p_entrada.solicitud).periodo_causa_fin IS NOT NULL
                    OR (p_entrada.analisis).periodo_causa_fin IS NOT NULL
                THEN 'fecha:' ELSE '' END) ||
            vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
                v_solicitud.periodo_fin
            )
        ELSE 'causa:' || v_solicitud.periodo_causa_fin END,
        v_mascara::text
    ]::text[];
    FOREACH v_campo IN ARRAY v_campos LOOP
        v_canon := v_canon
            || vec_contratacion_temporal.encuadrar_valor_rrhh_v1(
                vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
                    v_campo
                )
            );
    END LOOP;

    v_campos := ARRAY[
        CASE WHEN p_entrada.analisis_presente THEN '1' ELSE '0' END,
        p_entrada.referencia_analisis::text
    ]::text[];
    IF p_entrada.analisis_presente THEN
        v_campos := v_campos || ARRAY[
            v_analisis.modalidad_clave, v_analisis.categoria_ref,
            v_analisis.causa_clave,
            vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
                v_analisis.periodo_inicio
            ),
            CASE WHEN v_analisis.periodo_fin IS NOT NULL THEN
                (CASE WHEN v_derivado OR (p_entrada.solicitud).periodo_causa_fin IS NOT NULL
                        OR (p_entrada.analisis).periodo_causa_fin IS NOT NULL
                    THEN 'fecha:' ELSE '' END) ||
                vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
                    v_analisis.periodo_fin
                )
            ELSE 'causa:' || v_analisis.periodo_causa_fin END,
            v_analisis.porcentaje_jornada::text,
            v_analisis.resultado_rc,
            CASE WHEN v_analisis.coste_presente THEN '1' ELSE '0' END
        ]::text[];
        IF v_analisis.coste_presente THEN
            v_campos := v_campos || ARRAY[
                v_analisis.coste_centimos::text,
                v_analisis.coste_moneda
            ]::text[];
        END IF;
        v_campos := pg_catalog.array_append(
            v_campos, v_analisis.fuente_coste_ref
        );
        v_campos := pg_catalog.array_append(
            v_campos, v_analisis.observaciones
        );
    END IF;
    FOREACH v_campo IN ARRAY v_campos LOOP
        v_canon := v_canon
            || vec_contratacion_temporal.encuadrar_valor_rrhh_v1(
                vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
                    v_campo
                )
            );
    END LOOP;

    v_campos := ARRAY[
        CASE WHEN p_entrada.cobertura_presente THEN '1' ELSE '0' END,
        p_entrada.referencia_cobertura::text
    ]::text[];
    IF p_entrada.cobertura_presente THEN
        v_campos := v_campos || ARRAY[
            v_cobertura.via_clave,
            CASE WHEN v_cobertura.decision_gobernada THEN '1' ELSE '0' END,
            v_cobertura.procedimiento_ref, v_cobertura.bolsa_ref,
            v_total_comprobaciones::text
        ]::text[];
        FOR v_indice IN 1..v_total_comprobaciones LOOP
            v_comprobacion := v_cobertura.comprobaciones[v_indice];
            v_campos := v_campos || ARRAY[
                v_comprobacion.clave, v_comprobacion.resultado
            ]::text[];
        END LOOP;
    END IF;
    FOREACH v_campo IN ARRAY v_campos LOOP
        v_canon := v_canon
            || vec_contratacion_temporal.encuadrar_valor_rrhh_v1(
                vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
                    v_campo
                )
            );
    END LOOP;

    v_campos := ARRAY[
        CASE WHEN p_entrada.asignacion_presente THEN '1' ELSE '0' END,
        p_entrada.referencia_asignacion::text
    ]::text[];
    IF p_entrada.asignacion_presente THEN
        v_campos := v_campos || ARRAY[
            v_asignacion.unidad_ref,
            vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
                v_asignacion.asignada_en
            ),
            v_asignacion.motivo_clave
        ]::text[];
    END IF;
    FOREACH v_campo IN ARRAY v_campos LOOP
        v_canon := v_canon
            || vec_contratacion_temporal.encuadrar_valor_rrhh_v1(
                vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
                    v_campo
                )
            );
    END LOOP;

    IF p_entrada.fiscalizacion_presente THEN
        v_campos := ARRAY[
            '1', p_entrada.referencia_fiscalizacion::text,
            p_entrada.referencia_subsanacion::text,
            v_fiscalizacion.resultado_clave,
            CASE WHEN v_fiscalizacion.reparo_clave = '' THEN '0' ELSE '1' END
        ]::text[];
        IF v_fiscalizacion.reparo_clave <> '' THEN
            v_campos := v_campos || ARRAY[
                v_fiscalizacion.reparo_clave, v_fiscalizacion.reparo_texto
            ]::text[];
        END IF;
        v_campos := v_campos || ARRAY[
            vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
                v_fiscalizacion.registrada_en
            ),
            CASE WHEN v_fiscalizacion.subsanacion_registrada_en IS NULL
                 THEN '0' ELSE '1' END
        ]::text[];
        IF v_fiscalizacion.subsanacion_registrada_en IS NOT NULL THEN
            v_campos := v_campos || ARRAY[
                vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
                    v_fiscalizacion.subsanacion_registrada_en
                ), v_fiscalizacion.subsanacion_texto
            ]::text[];
        END IF;
    ELSE
        v_campos := ARRAY['0', '0', '0']::text[];
    END IF;
    FOREACH v_campo IN ARRAY v_campos LOOP
        v_canon := v_canon
            || vec_contratacion_temporal.encuadrar_valor_rrhh_v1(
                vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
                    v_campo
                )
            );
    END LOOP;

    v_campos := ARRAY[v_total_hitos::text]::text[];
    FOREACH v_campo IN ARRAY v_campos LOOP
        v_canon := v_canon
            || vec_contratacion_temporal.encuadrar_valor_rrhh_v1(
                vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
                    v_campo
                )
            );
    END LOOP;

    IF p_entrada.fiscalizacion_presente THEN
        v_subsanacion_presente :=
            v_fiscalizacion.subsanacion_registrada_en IS NOT NULL
            OR v_fiscalizacion.subsanacion_texto IS NOT NULL;
        IF v_fiscalizacion.resultado_clave NOT IN (
               'favorable', 'favorable_con_observaciones', 'desfavorable'
           )
           OR v_fiscalizacion.reparo_clave IS NULL
           OR v_fiscalizacion.reparo_texto IS NULL
           OR v_fiscalizacion.registrada_en IS NULL
           OR (v_fiscalizacion.resultado_clave = 'favorable'
               AND (v_fiscalizacion.reparo_clave <> ''
                    OR v_fiscalizacion.reparo_texto <> ''))
           OR (v_fiscalizacion.resultado_clave <> 'favorable'
               AND (v_fiscalizacion.reparo_clave <> 'observaciones_fiscalizacion'
                    OR pg_catalog.char_length(v_fiscalizacion.reparo_texto)
                       NOT BETWEEN 1 AND 2000))
           OR (v_subsanacion_presente
               AND (v_fiscalizacion.resultado_clave <> 'desfavorable'
                    OR v_fiscalizacion.subsanacion_registrada_en IS NULL
                    OR v_fiscalizacion.subsanacion_texto IS NULL
                    OR pg_catalog.char_length(v_fiscalizacion.subsanacion_texto)
                       NOT BETWEEN 1 AND 2000
                    OR p_entrada.referencia_subsanacion
                       <= p_entrada.referencia_fiscalizacion))
           OR (NOT v_subsanacion_presente
               AND p_entrada.referencia_subsanacion <> 0) THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'contenido de detalle RRHH inválido';
        END IF;
        PERFORM vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
            v_fiscalizacion.registrada_en
        );
        IF v_subsanacion_presente THEN
            PERFORM vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
                v_fiscalizacion.subsanacion_registrada_en
            );
            IF v_fiscalizacion.subsanacion_registrada_en
               <= v_fiscalizacion.registrada_en THEN
                RAISE EXCEPTION USING ERRCODE = '22023',
                    MESSAGE = 'contenido de detalle RRHH inválido';
            END IF;
        END IF;
    END IF;

    FOR v_indice IN 1..v_total_hitos LOOP
        v_hito := p_entrada.hitos[v_indice];
        v_campos := ARRAY[
            v_hito.secuencia::text,
            v_hito.version_expediente::text,
            v_hito.accion_clave,
            vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
                v_hito.realizada_en
            ),
            v_hito.fase_origen, v_hito.fase_destino,
            v_hito.estado_origen, v_hito.estado_destino
        ]::text[];
        FOREACH v_campo IN ARRAY v_campos LOOP
            v_canon := v_canon
                || vec_contratacion_temporal.encuadrar_valor_rrhh_v1(
                    vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
                        v_campo
                    )
                );
        END LOOP;
        IF pg_catalog.octet_length(v_canon) > 262144 THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'contenido de detalle RRHH inválido';
        END IF;
    END LOOP;
    IF v_derivado THEN
        v_campos := ARRAY[
          v_instruccion->>'derivacion_ref',v_instruccion->>'padre_ref',
          v_instruccion->>'numero_visible_padre',v_instruccion->>'oferta_ref',
          v_instruccion->>'numero_plaza',v_instruccion->>'recibo_derivacion_ref',
          v_instruccion->>'huella_padre_sha256',
          vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
            (v_instruccion->>'creado_en_padre')::timestamptz),
          vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
            (v_instruccion->>'actualizado_en_padre')::timestamptz),
          v_total_instruccion::text
        ];
        FOREACH v_campo IN ARRAY v_campos LOOP
          v_canon:=v_canon||vec_contratacion_temporal.encuadrar_valor_rrhh_v1(
            vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(v_campo));
        END LOOP;
        FOR v_indice IN 1..v_total_instruccion LOOP
          v_hito:=v_hitos_instruccion[v_indice];
          v_campos:=ARRAY[v_hito.secuencia::text,v_hito.version_expediente::text,
            v_hito.accion_clave,vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(v_hito.realizada_en),
            v_hito.fase_origen,v_hito.fase_destino,v_hito.estado_origen,v_hito.estado_destino];
          FOREACH v_campo IN ARRAY v_campos LOOP
            v_canon:=v_canon||vec_contratacion_temporal.encuadrar_valor_rrhh_v1(
              vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(v_campo));
          END LOOP;
        END LOOP;
    END IF;
    IF pg_catalog.octet_length(v_canon)>262144 THEN
      RAISE EXCEPTION 'contenido de detalle RRHH inválido' USING ERRCODE='22023'; END IF;
    RETURN v_canon;
EXCEPTION WHEN OTHERS THEN
    RAISE EXCEPTION USING ERRCODE = '22023',
        MESSAGE = 'contenido de detalle RRHH inválido';
END
$function$

;
-- CT237-GENERADO-FIN
-- Misma OID, dueño, ACL, configuración y cuerpo esperado tras CREATE OR REPLACE.
DO $post$
DECLARE fila record; actual pg_proc%ROWTYPE;
BEGIN
 FOR fila IN SELECT * FROM (VALUES
  ('canon_resumen_publicacion_rrhh_v1','9b5ea485a08a5a40160b1db0248c1ee33699ed2a550e2c0f0472972f0059db02'),
  ('materializar_detalle_rrhh_v1','66a80e90c6371e2a5c9a4a9c80368921072382f939f6fef73a4606d0d6d28bbc'),
  ('canon_contenido_detalle_rrhh_v1','f0685b3d58f9d448c57a806ae4079b47ed2f74703177f08680088ce9ced27b66')
 ) AS esperado(nombre,huella) LOOP
   SELECT p.* INTO STRICT actual FROM pg_proc p WHERE p.proname=fila.nombre
     AND p.oid=(SELECT oid FROM pg_temp.ct237_funciones_pre WHERE proname=fila.nombre);
   IF encode(sha256(convert_to(actual.prosrc,'UTF8')),'hex')<>fila.huella
      OR actual.proowner IS DISTINCT FROM (SELECT proowner FROM pg_temp.ct237_funciones_pre WHERE proname=fila.nombre)
      OR actual.proacl IS DISTINCT FROM (SELECT proacl FROM pg_temp.ct237_funciones_pre WHERE proname=fila.nombre)
      OR actual.proconfig IS DISTINCT FROM (SELECT array_replace(proconfig,'search_path=pg_catalog','search_path=pg_catalog, pg_temp') FROM pg_temp.ct237_funciones_pre WHERE proname=fila.nombre)
      OR actual.provolatile IS DISTINCT FROM (SELECT provolatile FROM pg_temp.ct237_funciones_pre WHERE proname=fila.nombre)
      OR actual.proparallel IS DISTINCT FROM (SELECT proparallel FROM pg_temp.ct237_funciones_pre WHERE proname=fila.nombre)
      OR actual.prosecdef IS DISTINCT FROM (SELECT prosecdef FROM pg_temp.ct237_funciones_pre WHERE proname=fila.nombre)
   THEN RAISE EXCEPTION 'CT237: postimagen divergente: %',fila.nombre USING ERRCODE='55000'; END IF;
 END LOOP;
END $post$;
COMMIT;
