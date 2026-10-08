\set ON_ERROR_STOP on
-- Ensayo focal de CT192 en un clon desechable con publicaciones sintéticas.
-- Los helpers son internos: esta prueba usa el propietario solo dentro de ROLLBACK.
BEGIN;
CREATE ROLE vec_ct192_foco NOLOGIN;
GRANT USAGE ON SCHEMA vec_contratacion_temporal TO vec_ct192_foco;
GRANT USAGE ON TYPE vec_contratacion_temporal.consulta_cuadro_rrhh_v2
    TO vec_ct192_foco;
GRANT USAGE ON TYPE vec_contratacion_temporal.alcance_consulta_rrhh_v1
    TO vec_ct192_foco;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.filtrar_publicaciones_cuadro_rrhh_v2(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_cuadro_rrhh_v2, numeric) TO vec_ct192_foco;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.contextos_plazo_cuadro_rrhh_v2(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_cuadro_rrhh_v2, numeric) TO vec_ct192_foco;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.paginar_y_contar_cuadro_rrhh_v2(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_cuadro_rrhh_v2,
    numeric, timestamptz, text, text[]) TO vec_ct192_foco;
SET LOCAL ROLE vec_ct192_foco;

DO $prueba_denegacion$
DECLARE
    v_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1 :=
        ROW('organizacion:prueba', 'organizacion', 'organizacion:prueba');
    v_consulta vec_contratacion_temporal.consulta_cuadro_rrhh_v2 :=
        ROW('', '', '', ARRAY[]::text[], ARRAY[]::text[], '', 1::smallint, '');
BEGIN
    BEGIN
        PERFORM vec_contratacion_temporal.filtrar_publicaciones_cuadro_rrhh_v2(
            v_alcance, v_consulta, 1);
        RAISE EXCEPTION 'CT192: filtro interno aceptó rol sin autoridad';
    EXCEPTION WHEN SQLSTATE '42501' THEN NULL;
    END;
    BEGIN
        PERFORM vec_contratacion_temporal.contextos_plazo_cuadro_rrhh_v2(
            v_alcance, v_consulta, 1);
        RAISE EXCEPTION 'CT192: contextos internos aceptaron rol sin autoridad';
    EXCEPTION WHEN SQLSTATE '42501' THEN NULL;
    END;
    BEGIN
        PERFORM vec_contratacion_temporal.paginar_y_contar_cuadro_rrhh_v2(
            v_alcance, v_consulta, 1, NULL, NULL, NULL);
        RAISE EXCEPTION 'CT192: página interna aceptó rol sin autoridad';
    EXCEPTION WHEN SQLSTATE '42501' THEN NULL;
    END;
END $prueba_denegacion$;

RESET ROLE;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog, pg_temp;

DO $prueba$
DECLARE
    v_organizacion text;
    v_corte numeric;
    v_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1;
    v_consulta vec_contratacion_temporal.consulta_cuadro_rrhh_v2;
    v_referencias text[];
    v_publicadas text[];
    v_resultado record;
BEGIN
    SELECT p.organizacion_ref, pg_catalog.max(p.corte_global)
      INTO v_organizacion, v_corte
      FROM vec_contratacion_temporal.publicacion_version_rrhh p
     GROUP BY p.organizacion_ref
     ORDER BY pg_catalog.count(*) DESC
     LIMIT 1;
    IF v_organizacion IS NULL OR v_corte IS NULL THEN
        RAISE EXCEPTION 'CT192: faltan publicaciones sintéticas para el ensayo';
    END IF;
    v_alcance := ROW(v_organizacion, 'organizacion', v_organizacion)::
        vec_contratacion_temporal.alcance_consulta_rrhh_v1;
    v_consulta := ROW('', '', '', ARRAY[]::text[], ARRAY[]::text[],
        'vencido', 100::smallint, '')::vec_contratacion_temporal.consulta_cuadro_rrhh_v2;

    SELECT pg_catalog.array_agg(c.expediente_ref ORDER BY c.expediente_ref COLLATE "C")
      INTO v_referencias
      FROM (
        SELECT contexto.expediente_ref
          FROM vec_contratacion_temporal.contextos_plazo_cuadro_rrhh_v2(
               v_alcance, v_consulta, v_corte) contexto
         ORDER BY contexto.expediente_ref COLLATE "C"
         LIMIT 3
      ) c;
    IF pg_catalog.cardinality(v_referencias) < 1 THEN
        RAISE EXCEPTION 'CT192: faltan expedientes activos sintéticos para el ensayo';
    END IF;

    SELECT * INTO STRICT v_resultado
      FROM vec_contratacion_temporal.paginar_y_contar_cuadro_rrhh_v2(
           v_alcance, v_consulta, v_corte, NULL, NULL, v_referencias);
    SELECT pg_catalog.array_agg(p.expediente_ref
               ORDER BY p.expediente_ref COLLATE "C")
      INTO v_publicadas
      FROM pg_catalog.unnest(v_resultado.resumenes) p;
    IF v_resultado.total_filtrado <> pg_catalog.cardinality(v_referencias)
       OR v_resultado.en_tramite <> v_resultado.total_filtrado
       OR v_resultado.hay_mas IS DISTINCT FROM false
       OR v_publicadas IS DISTINCT FROM v_referencias THEN
        RAISE EXCEPTION 'CT192: página/total/contextos distintos: refs=%, página=%, total=%',
            v_referencias, v_publicadas, v_resultado.total_filtrado;
    END IF;

    SELECT * INTO STRICT v_resultado
      FROM vec_contratacion_temporal.paginar_y_contar_cuadro_rrhh_v2(
           v_alcance, v_consulta, v_corte, NULL, NULL, ARRAY[]::text[]);
    IF v_resultado.total_filtrado <> 0
       OR pg_catalog.cardinality(v_resultado.resumenes) <> 0
       OR v_resultado.hay_mas IS DISTINCT FROM false THEN
        RAISE EXCEPTION 'CT192: conjunto vacío no produce página vacía';
    END IF;

    BEGIN
        PERFORM vec_contratacion_temporal.paginar_y_contar_cuadro_rrhh_v2(
            v_alcance, v_consulta, v_corte, NULL, NULL,
            ARRAY[v_referencias[1], v_referencias[1]]);
        RAISE EXCEPTION 'CT192: referencias duplicadas aceptadas';
    EXCEPTION WHEN SQLSTATE '42501' THEN
        NULL;
    END;
END $prueba$;

ROLLBACK;
