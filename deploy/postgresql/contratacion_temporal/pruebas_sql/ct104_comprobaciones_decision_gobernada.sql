\set ON_ERROR_STOP on
BEGIN;

DO $prueba$
DECLARE
    v_lote text;
    v_ordenes integer[];
    v_indice integer;
BEGIN
    SELECT lote.lote_ref,
           pg_catalog.array_agg(evidencia.posicion ORDER BY evidencia.posicion)
      INTO v_lote, v_ordenes
      FROM vec_contratacion_temporal.consumo_cobertura_lote lote
      JOIN vec_contratacion_temporal.consumo_cobertura_evidencia evidencia
        ON evidencia.lote_ref = lote.lote_ref
     GROUP BY lote.lote_ref
    HAVING pg_catalog.count(*) >= 2
     ORDER BY lote.lote_ref
     LIMIT 1;

    IF v_lote IS NULL OR pg_catalog.cardinality(v_ordenes) < 2 THEN
        RAISE EXCEPTION 'CT104 no encontró dos evidencias ordenadas del mismo lote';
    END IF;
    FOR v_indice IN 2..pg_catalog.cardinality(v_ordenes) LOOP
        IF v_ordenes[v_indice] <= v_ordenes[v_indice - 1] THEN
            RAISE EXCEPTION 'CT104 alteró el orden de las evidencias';
        END IF;
    END LOOP;
END
$prueba$;

ROLLBACK;
