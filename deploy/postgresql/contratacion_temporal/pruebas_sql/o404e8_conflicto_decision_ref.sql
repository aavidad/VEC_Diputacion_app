\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path = pg_catalog, pg_temp;
CREATE TEMP TABLE prueba_o404e8 (decision_ref text PRIMARY KEY);
CREATE FUNCTION pg_temp.probar_conflicto_ambiguo()
RETURNS TABLE(decision_ref text)
LANGUAGE plpgsql AS $f$
BEGIN
  INSERT INTO pg_temp.prueba_o404e8(decision_ref) VALUES ('decision:abc')
  ON CONFLICT (decision_ref) DO NOTHING;
  RETURN QUERY SELECT 'decision:abc'::text;
END $f$;
CREATE FUNCTION pg_temp.probar_conflicto_explicito()
RETURNS TABLE(decision_ref text)
LANGUAGE plpgsql AS $f$
BEGIN
  INSERT INTO pg_temp.prueba_o404e8(decision_ref) VALUES ('decision:abc')
  ON CONFLICT ON CONSTRAINT prueba_o404e8_pkey DO NOTHING;
  RETURN QUERY SELECT 'decision:abc'::text;
END $f$;
DO $prueba$
DECLARE codigo text;
BEGIN
  BEGIN
    PERFORM * FROM pg_temp.probar_conflicto_ambiguo();
    RAISE EXCEPTION 'O4-04E8: faltó error de columna ambigua';
  EXCEPTION WHEN ambiguous_column THEN
    GET STACKED DIAGNOSTICS codigo = RETURNED_SQLSTATE;
    IF codigo <> '42702' THEN RAISE EXCEPTION 'O4-04E8: SQLSTATE %',codigo; END IF;
  END;
  PERFORM * FROM pg_temp.probar_conflicto_explicito();
  PERFORM * FROM pg_temp.probar_conflicto_explicito();
  IF (SELECT count(*) FROM pg_temp.prueba_o404e8) <> 1 THEN
    RAISE EXCEPTION 'O4-04E8: replay duplicó fila';
  END IF;
  IF pg_catalog.strpos(pg_catalog.pg_get_functiondef('vec_autorizacion.registrar_decision_cobertura_contratacion_temporal_v1(bytea,bytea,numeric,numeric,jsonb)'::pg_catalog.regprocedure),'ON CONFLICT ON CONSTRAINT enlace_decision_cobertura_ct_o404e_pkey DO NOTHING')=0 THEN
    RAISE EXCEPTION 'O4-04E8: función instalada sin árbitro explícito';
  END IF;
END $prueba$;
ROLLBACK;
