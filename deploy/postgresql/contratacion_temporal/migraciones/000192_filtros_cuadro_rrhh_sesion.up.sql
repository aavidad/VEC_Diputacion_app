\set ON_ERROR_STOP on
-- CT192: predicado privado de filtros RRHH para el lector de sesión W.
-- No concede lectura al runtime ni crea una decisión V3 por consulta.
BEGIN;
SET LOCAL search_path = pg_catalog, pg_temp;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '60s';
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_contratacion_temporal:migracion:000192', 0)
);

DO $pre$
BEGIN
    IF pg_catalog.current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
       OR pg_catalog.to_regtype('vec_contratacion_temporal.consulta_cuadro_rrhh_v2') IS NOT NULL
       OR pg_catalog.to_regclass('vec_contratacion_temporal.publicacion_version_rrhh') IS NULL
       OR pg_catalog.to_regclass('vec_contratacion_temporal.numeracion_anual_asignada') IS NULL
       OR pg_catalog.to_regtype('vec_contratacion_temporal.resumen_publicacion_rrhh_v1') IS NULL
       OR pg_catalog.to_regprocedure('vec_contratacion_temporal.canon_alcance_rrhh_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1)') IS NULL
       OR pg_catalog.to_regprocedure('vec_contratacion_temporal.texto_json_go_v1(text)') IS NULL THEN
        RAISE EXCEPTION 'CT192: base incompatible o migración ya presente' USING ERRCODE = '55000';
    END IF;
END $pre$;

SET LOCAL ROLE vec_contratacion_temporal_propietario;

CREATE TYPE vec_contratacion_temporal.consulta_cuadro_rrhh_v2 AS (
    texto text,
    centro_ref text,
    categoria_ref text,
    estados_clave text[],
    fases_clave text[],
    limite smallint,
    cursor text
);
REVOKE ALL ON TYPE vec_contratacion_temporal.consulta_cuadro_rrhh_v2 FROM PUBLIC;

-- Devuelve los campos posteriores al dominio. Validar aquí una sola vez
-- mantiene idénticas las dos familias de canon sin aceptar arrays ambiguos.
CREATE FUNCTION vec_contratacion_temporal.campos_consulta_cuadro_rrhh_v2(
    p_consulta vec_contratacion_temporal.consulta_cuadro_rrhh_v2
) RETURNS text LANGUAGE plpgsql IMMUTABLE STRICT PARALLEL SAFE
SET search_path = pg_catalog, pg_temp AS $funcion$
DECLARE
    v_estados text := '';
    v_fases text := '';
    v_anterior text;
    v_clave text;
BEGIN
    IF p_consulta.texto IS NULL
       OR pg_catalog.octet_length(p_consulta.texto) > 160
       OR p_consulta.texto <> pg_catalog.btrim(p_consulta.texto)
       OR p_consulta.texto !~ '^[0-9A-Za-zÁÉÍÓÚÜÑáéíóúüñ/._ -]{0,80}$'
       OR p_consulta.centro_ref IS NULL
       OR p_consulta.categoria_ref IS NULL
       OR (p_consulta.centro_ref <> '' AND (
           pg_catalog.octet_length(p_consulta.centro_ref) > 160
           OR p_consulta.centro_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'))
       OR (p_consulta.categoria_ref <> '' AND (
           pg_catalog.octet_length(p_consulta.categoria_ref) > 160
           OR p_consulta.categoria_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'))
       OR p_consulta.estados_clave IS NULL
       OR p_consulta.fases_clave IS NULL
       OR pg_catalog.cardinality(p_consulta.estados_clave) > 6
       OR pg_catalog.cardinality(p_consulta.fases_clave) > 32
       OR pg_catalog.array_ndims(p_consulta.estados_clave) > 1
       OR pg_catalog.array_ndims(p_consulta.fases_clave) > 1
       OR p_consulta.limite IS NULL
       OR p_consulta.limite NOT BETWEEN 1 AND 100
       OR p_consulta.cursor IS NULL
       OR pg_catalog.octet_length(p_consulta.cursor) > 43
       OR (p_consulta.cursor <> '' AND (
           p_consulta.cursor !~ '^[A-Za-z0-9_-]{43}$'
           OR pg_catalog.rtrim(pg_catalog.translate(pg_catalog.encode(
               pg_catalog.decode(pg_catalog.translate(p_consulta.cursor, '-_', '+/') || '=', 'base64'),
               'base64'), '+/', '-_'), E'=\n') <> p_consulta.cursor)) THEN
        RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'consulta RRHH inválida';
    END IF;
    v_anterior := NULL;
    FOREACH v_clave IN ARRAY p_consulta.estados_clave LOOP
        IF v_clave IS NULL OR pg_catalog.octet_length(v_clave) > 16
           OR v_clave NOT IN ('pendiente', 'en_curso', 'espera_externa', 'completado', 'incidencia', 'cancelado')
           OR (v_anterior IS NOT NULL AND v_anterior COLLATE "C" >= v_clave COLLATE "C") THEN
            RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'consulta RRHH inválida';
        END IF;
        v_estados := v_estados || CASE WHEN v_anterior IS NULL THEN '' ELSE ',' END
            || vec_contratacion_temporal.texto_json_go_v1(v_clave);
        v_anterior := v_clave;
    END LOOP;
    v_anterior := NULL;
    FOREACH v_clave IN ARRAY p_consulta.fases_clave LOOP
        IF v_clave IS NULL OR pg_catalog.octet_length(v_clave) > 80
           OR v_clave !~ '^[a-z][a-z0-9._-]{1,79}$'
           OR (v_anterior IS NOT NULL AND v_anterior COLLATE "C" >= v_clave COLLATE "C") THEN
            RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'consulta RRHH inválida';
        END IF;
        v_fases := v_fases || CASE WHEN v_anterior IS NULL THEN '' ELSE ',' END
            || vec_contratacion_temporal.texto_json_go_v1(v_clave);
        v_anterior := v_clave;
    END LOOP;
    RETURN ',"texto":' || vec_contratacion_temporal.texto_json_go_v1(p_consulta.texto)
        || ',"centro_ref":' || vec_contratacion_temporal.texto_json_go_v1(p_consulta.centro_ref)
        || ',"categoria_ref":' || vec_contratacion_temporal.texto_json_go_v1(p_consulta.categoria_ref)
        || ',"estados_clave":[' || v_estados || ']'
        || ',"fases_clave":[' || v_fases || ']'
        || ',"limite":' || p_consulta.limite::text;
EXCEPTION WHEN OTHERS THEN
    RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'consulta RRHH inválida';
END $funcion$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.campos_consulta_cuadro_rrhh_v2(
    vec_contratacion_temporal.consulta_cuadro_rrhh_v2) FROM PUBLIC;

CREATE FUNCTION vec_contratacion_temporal.canon_consulta_cuadro_rrhh_v2(
    p_consulta vec_contratacion_temporal.consulta_cuadro_rrhh_v2
) RETURNS bytea LANGUAGE sql IMMUTABLE STRICT PARALLEL SAFE
SET search_path = pg_catalog, pg_temp AS $funcion$
    SELECT pg_catalog.convert_to(
        '{"dominio":"vec.contratacion_temporal.consulta_rrhh.cuadro.v2","version":2'
        || vec_contratacion_temporal.campos_consulta_cuadro_rrhh_v2(p_consulta)
        || ',"cursor":' || vec_contratacion_temporal.texto_json_go_v1(p_consulta.cursor) || '}', 'UTF8')
$funcion$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.canon_consulta_cuadro_rrhh_v2(
    vec_contratacion_temporal.consulta_cuadro_rrhh_v2) FROM PUBLIC;

CREATE FUNCTION vec_contratacion_temporal.canon_familia_cuadro_rrhh_v2(
    p_consulta vec_contratacion_temporal.consulta_cuadro_rrhh_v2
) RETURNS bytea LANGUAGE sql IMMUTABLE STRICT PARALLEL SAFE
SET search_path = pg_catalog, pg_temp AS $funcion$
    SELECT pg_catalog.convert_to(
        '{"dominio":"vec.contratacion_temporal.filtros_rrhh.cuadro.v2","version":2'
        || vec_contratacion_temporal.campos_consulta_cuadro_rrhh_v2(p_consulta) || '}', 'UTF8')
$funcion$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.canon_familia_cuadro_rrhh_v2(
    vec_contratacion_temporal.consulta_cuadro_rrhh_v2) FROM PUBLIC;

-- Solo el propietario interno puede usar este conjunto. El futuro lector W
-- acreditará sesión y registrará el acceso nominal en su misma transacción.
-- La última versión se selecciona antes del ámbito y de cualquier filtro.
CREATE FUNCTION vec_contratacion_temporal.filtrar_publicaciones_cuadro_rrhh_v2(
    p_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    p_consulta vec_contratacion_temporal.consulta_cuadro_rrhh_v2,
    p_corte_global numeric
) RETURNS SETOF vec_contratacion_temporal.resumen_publicacion_rrhh_v1
LANGUAGE plpgsql STABLE SECURITY DEFINER PARALLEL UNSAFE
SET search_path = pg_catalog, pg_temp
SET row_security = 'on'
SET timezone = 'UTC'
SET lock_timeout = '1s'
SET statement_timeout = '4s'
SET idle_in_transaction_session_timeout = '6s'
AS $funcion$
BEGIN
    IF CURRENT_USER <> 'vec_contratacion_temporal_propietario'
       OR p_alcance IS NULL OR p_consulta IS NULL
       OR p_corte_global IS NULL
       OR p_corte_global NOT BETWEEN 0 AND 9007199254740991::numeric
       OR p_corte_global <> pg_catalog.trunc(p_corte_global) THEN
        RAISE EXCEPTION USING ERRCODE = '42501', MESSAGE = 'cuadro RRHH no disponible';
    END IF;
    PERFORM vec_contratacion_temporal.canon_alcance_rrhh_v1(p_alcance);
    PERFORM vec_contratacion_temporal.canon_consulta_cuadro_rrhh_v2(p_consulta);
    RETURN QUERY
    WITH ultimas AS MATERIALIZED (
        SELECT DISTINCT ON (publicada.expediente_ref COLLATE "C")
               publicada.expediente_ref, publicada.organizacion_ref,
               publicada.numero_visible, publicada.version,
               publicada.flujo_ref, publicada.flujo_version,
               publicada.flujo_huella_sha256, publicada.fase_clave,
               publicada.estado_clave, publicada.centro_ref,
               publicada.categoria_ref, publicada.modalidad_clave,
               publicada.unidad_ref, publicada.creado_en, publicada.actualizado_en
          FROM vec_contratacion_temporal.publicacion_version_rrhh publicada
         WHERE publicada.corte_global <= p_corte_global
         ORDER BY publicada.expediente_ref COLLATE "C", publicada.corte_global DESC
    )
    SELECT (ROW(
               ultima.expediente_ref, ultima.organizacion_ref,
               COALESCE(numeracion.numero_visible, ultima.numero_visible),
               ultima.version, ultima.flujo_ref, ultima.flujo_version,
               ultima.flujo_huella_sha256, ultima.fase_clave,
               ultima.estado_clave, ultima.centro_ref,
               ultima.categoria_ref, COALESCE(ultima.modalidad_clave, ''),
               COALESCE(ultima.unidad_ref, ''), ultima.creado_en,
               ultima.actualizado_en
           )::vec_contratacion_temporal.resumen_publicacion_rrhh_v1).*
      FROM ultimas ultima
      LEFT JOIN vec_contratacion_temporal.numeracion_anual_asignada numeracion
        ON numeracion.expediente_ref = ultima.expediente_ref
     WHERE ultima.organizacion_ref COLLATE "C" = p_alcance.organizacion_ref COLLATE "C"
       AND (p_alcance.clase_ambito = 'organizacion'
            OR p_alcance.clase_ambito = 'centro'
               AND ultima.centro_ref COLLATE "C" = p_alcance.ambito_ref COLLATE "C"
            OR p_alcance.clase_ambito = 'unidad_gestion'
               AND ultima.unidad_ref IS NOT NULL
               AND ultima.unidad_ref COLLATE "C" = p_alcance.ambito_ref COLLATE "C")
       AND (p_consulta.texto = '' OR pg_catalog.left(
           COALESCE(numeracion.numero_visible, ultima.numero_visible),
           pg_catalog.char_length(p_consulta.texto)) COLLATE "C" = p_consulta.texto COLLATE "C")
       AND (p_consulta.centro_ref = '' OR ultima.centro_ref COLLATE "C" = p_consulta.centro_ref COLLATE "C")
       AND (p_consulta.categoria_ref = '' OR ultima.categoria_ref COLLATE "C" = p_consulta.categoria_ref COLLATE "C")
       AND (pg_catalog.cardinality(p_consulta.estados_clave) = 0
            OR ultima.estado_clave COLLATE "C" = ANY(p_consulta.estados_clave))
       AND (pg_catalog.cardinality(p_consulta.fases_clave) = 0
            OR ultima.fase_clave COLLATE "C" = ANY(p_consulta.fases_clave));
END $funcion$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.filtrar_publicaciones_cuadro_rrhh_v2(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_cuadro_rrhh_v2, numeric) FROM PUBLIC;

-- El consumidor de sesión proporcionará alcance/corte/ancla tras validarlos.
-- Página y recuentos leen una sola materialización filtrada en el mismo corte.
-- Este helper no concede acceso, no interpreta el token ni crea auditoría.
CREATE FUNCTION vec_contratacion_temporal.paginar_y_contar_cuadro_rrhh_v2(
    p_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    p_consulta vec_contratacion_temporal.consulta_cuadro_rrhh_v2,
    p_corte_global numeric,
    p_ultimo_actualizado_en timestamptz,
    p_ultimo_expediente_ref text
) RETURNS TABLE (
    resumenes vec_contratacion_temporal.resumen_publicacion_rrhh_v1[],
    hay_mas boolean,
    total_filtrado bigint,
    en_tramite bigint,
    terminados bigint,
    recuento_estados text[],
    recuento_fases text[],
    recuento_numeros bigint[],
    ultimo_actualizado_en timestamptz,
    ultimo_expediente_ref text
)
LANGUAGE plpgsql STABLE SECURITY DEFINER PARALLEL UNSAFE
SET search_path = pg_catalog, pg_temp
SET row_security = 'on'
SET timezone = 'UTC'
SET lock_timeout = '1s'
SET statement_timeout = '4s'
SET idle_in_transaction_session_timeout = '6s'
AS $funcion$
BEGIN
    IF CURRENT_USER <> 'vec_contratacion_temporal_propietario'
       OR p_alcance IS NULL OR p_consulta IS NULL
       OR p_corte_global IS NULL
       OR p_corte_global NOT BETWEEN 0 AND 9007199254740991::numeric
       OR p_corte_global <> pg_catalog.trunc(p_corte_global)
       OR (p_consulta.cursor = '' AND (
           p_ultimo_actualizado_en IS NOT NULL OR p_ultimo_expediente_ref IS NOT NULL))
       OR (p_consulta.cursor <> '' AND (
           p_ultimo_actualizado_en IS NULL OR p_ultimo_expediente_ref IS NULL))
       OR (p_ultimo_actualizado_en IS NOT NULL AND (
           p_ultimo_actualizado_en <> pg_catalog.date_trunc('microseconds', p_ultimo_actualizado_en)
           OR p_ultimo_expediente_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$')) THEN
        RAISE EXCEPTION USING ERRCODE = '42501', MESSAGE = 'cuadro RRHH no disponible';
    END IF;
    PERFORM vec_contratacion_temporal.canon_alcance_rrhh_v1(p_alcance);
    PERFORM vec_contratacion_temporal.canon_consulta_cuadro_rrhh_v2(p_consulta);
    RETURN QUERY
    WITH conjunto AS MATERIALIZED (
        SELECT f.* FROM vec_contratacion_temporal.filtrar_publicaciones_cuadro_rrhh_v2(
            p_alcance, p_consulta, p_corte_global) f
    ), grupos AS MATERIALIZED (
        SELECT f.estado_clave, f.fase_clave, pg_catalog.count(*)::bigint numero
          FROM conjunto f
         GROUP BY f.estado_clave, f.fase_clave
    ), pagina AS MATERIALIZED (
        SELECT f.*, pg_catalog.row_number() OVER (
                   ORDER BY f.actualizado_en DESC, f.expediente_ref COLLATE "C" DESC
               ) AS posicion
          FROM conjunto f
         WHERE p_ultimo_actualizado_en IS NULL
            OR f.actualizado_en < p_ultimo_actualizado_en
            OR (f.actualizado_en = p_ultimo_actualizado_en
                AND f.expediente_ref COLLATE "C" < p_ultimo_expediente_ref COLLATE "C")
         ORDER BY f.actualizado_en DESC, f.expediente_ref COLLATE "C" DESC
         LIMIT p_consulta.limite::integer + 1
    )
    SELECT COALESCE((
               SELECT pg_catalog.array_agg(
                   ROW(p.expediente_ref, p.organizacion_ref, p.numero_visible,
                       p.version, p.flujo_ref, p.flujo_version,
                       p.flujo_huella_sha256, p.fase_clave, p.estado_clave,
                       p.centro_ref, p.categoria_ref, p.modalidad_clave,
                       p.unidad_ref, p.creado_en, p.actualizado_en
                   )::vec_contratacion_temporal.resumen_publicacion_rrhh_v1
                   ORDER BY p.posicion
               ) FROM pagina p WHERE p.posicion <= p_consulta.limite
           ), ARRAY[]::vec_contratacion_temporal.resumen_publicacion_rrhh_v1[]),
           (SELECT pg_catalog.count(*) > p_consulta.limite FROM pagina),
           (SELECT pg_catalog.count(*)::bigint FROM conjunto),
           (SELECT pg_catalog.count(*)::bigint FROM conjunto f
             WHERE f.estado_clave NOT IN ('completado', 'cancelado')),
           (SELECT pg_catalog.count(*)::bigint FROM conjunto f
             WHERE f.estado_clave IN ('completado', 'cancelado')),
           COALESCE((SELECT pg_catalog.array_agg(g.estado_clave
               ORDER BY g.estado_clave COLLATE "C", g.fase_clave COLLATE "C")
               FROM grupos g), ARRAY[]::text[]),
           COALESCE((SELECT pg_catalog.array_agg(g.fase_clave
               ORDER BY g.estado_clave COLLATE "C", g.fase_clave COLLATE "C")
               FROM grupos g), ARRAY[]::text[]),
           COALESCE((SELECT pg_catalog.array_agg(g.numero
               ORDER BY g.estado_clave COLLATE "C", g.fase_clave COLLATE "C")
               FROM grupos g), ARRAY[]::bigint[]),
           (SELECT p.actualizado_en FROM pagina p
             WHERE p.posicion <= p_consulta.limite
             ORDER BY p.posicion DESC LIMIT 1),
           (SELECT p.expediente_ref FROM pagina p
             WHERE p.posicion <= p_consulta.limite
             ORDER BY p.posicion DESC LIMIT 1);
END $funcion$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.paginar_y_contar_cuadro_rrhh_v2(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_cuadro_rrhh_v2,
    numeric, timestamptz, text) FROM PUBLIC;

COMMENT ON FUNCTION vec_contratacion_temporal.filtrar_publicaciones_cuadro_rrhh_v2(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_cuadro_rrhh_v2, numeric) IS
'Conjunto privado de publicaciones al corte para el lector RRHH de sesión; no concede acceso ni registra auditoría.';
COMMENT ON FUNCTION vec_contratacion_temporal.paginar_y_contar_cuadro_rrhh_v2(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_cuadro_rrhh_v2,
    numeric, timestamptz, text) IS
'Página y recuentos RRHH de un mismo conjunto filtrado; el consumidor acredita sesión, cursor y auditoría en su transacción.';
COMMIT;
