\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SELECT pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:migracion:000021', 0));

DO $precondiciones$
BEGIN
  IF to_regclass('vec_bolsa_llamamientos.constitucion') IS NULL
     OR to_regclass('vec_bolsa_llamamientos.constitucion_entrada') IS NULL
     OR to_regclass('vec_bolsa_llamamientos.vinculo_candidato') IS NULL THEN
    RAISE EXCEPTION 'dependencias Bolsa 000007/000008 ausentes' USING ERRCODE='55000';
  END IF;
  IF to_regprocedure('vec_bolsa_llamamientos.rellenar_vinculos_candidato_v1(text,jsonb,timestamptz)') IS NOT NULL THEN
    RAISE EXCEPTION 'migracion Bolsa 000021 ya aplicada' USING ERRCODE='55000';
  END IF;
END $precondiciones$;

-- La clave HMAC y el staging descifrado nunca entran en PostgreSQL. El
-- recuperador de importación acredita cada fila y deriva `can_*` con el mismo
-- derivador que la constitución desde 000008. Esta función solo enlaza esas
-- referencias con las filas numeradas de un acta ya constituida.
CREATE FUNCTION vec_bolsa_llamamientos.rellenar_vinculos_candidato_v1(
  p_acta_ref text, p_filas jsonb, p_registrada_en timestamptz
)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog
SET lock_timeout='2s'
SET statement_timeout='30s'
AS $funcion$
DECLARE
  v_constitucion vec_bolsa_llamamientos.constitucion%ROWTYPE;
  v_fila jsonb;
  v_numero integer;
  v_candidato text;
  v_participacion text;
  v_actual text;
  v_nuevos integer := 0;
  v_existentes integer := 0;
BEGIN
  IF current_user <> 'vec_bolsa_llamamientos_propietario' THEN
    RAISE EXCEPTION 'relleno de candidato no disponible' USING ERRCODE='42501';
  END IF;
  IF p_acta_ref IS NULL OR p_filas IS NULL OR jsonb_typeof(p_filas) <> 'array'
     OR jsonb_array_length(p_filas) NOT BETWEEN 1 AND 100000
     OR p_registrada_en IS NULL THEN
    RAISE EXCEPTION 'relleno de candidato invalido' USING ERRCODE='22023';
  END IF;
  PERFORM pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:constitucion:' || p_acta_ref, 0));
  SELECT * INTO v_constitucion FROM vec_bolsa_llamamientos.constitucion
    WHERE acta_ref=p_acta_ref FOR SHARE;
  IF NOT FOUND THEN
    RAISE EXCEPTION 'acta no constituida' USING ERRCODE='23503';
  END IF;
  IF jsonb_array_length(p_filas) <> (
    SELECT count(*) FROM vec_bolsa_llamamientos.constitucion_entrada e
    WHERE e.instantanea_ref=v_constitucion.instantanea_ref
      AND e.version_instantanea=v_constitucion.version_instantanea
  ) THEN
    RAISE EXCEPTION 'filas de acta incompletas' USING ERRCODE='22023';
  END IF;
  IF EXISTS (
    SELECT 1 FROM jsonb_array_elements(p_filas) f
    GROUP BY f->>'fila_numero' HAVING count(*)>1
  ) THEN
    RAISE EXCEPTION 'fila de candidato duplicada' USING ERRCODE='22023';
  END IF;
  FOR v_fila IN SELECT value FROM jsonb_array_elements(p_filas) LOOP
    IF jsonb_typeof(v_fila) <> 'object'
       OR (SELECT count(*) FROM jsonb_object_keys(v_fila)) <> 2
       OR NOT (v_fila ? 'fila_numero' AND v_fila ? 'candidato_ref')
       OR jsonb_typeof(v_fila->'fila_numero') <> 'number'
       OR (v_fila->>'fila_numero') !~ '^[1-9][0-9]{0,8}$'
       OR jsonb_typeof(v_fila->'candidato_ref') <> 'string'
       OR (v_fila->>'candidato_ref') !~ '^can_[A-Za-z0-9_-]{43}$' THEN
      RAISE EXCEPTION 'fila de candidato invalida' USING ERRCODE='22023';
    END IF;
    v_numero := (v_fila->>'fila_numero')::integer;
    v_candidato := v_fila->>'candidato_ref';
    SELECT e.participacion_ref INTO v_participacion
      FROM vec_bolsa_llamamientos.constitucion_entrada e
     WHERE e.instantanea_ref=v_constitucion.instantanea_ref
       AND e.version_instantanea=v_constitucion.version_instantanea
       AND e.fila_numero=v_numero;
    IF NOT FOUND THEN
      RAISE EXCEPTION 'fila ajena a acta' USING ERRCODE='23503';
    END IF;
    SELECT vc.candidato_ref INTO v_actual
      FROM vec_bolsa_llamamientos.vinculo_candidato vc
     WHERE vc.participacion_ref=v_participacion;
    IF FOUND THEN
      IF v_actual <> v_candidato THEN
        RAISE EXCEPTION 'participacion vinculada a otro candidato' USING ERRCODE='23505';
      END IF;
      v_existentes := v_existentes+1;
    ELSE
      INSERT INTO vec_bolsa_llamamientos.vinculo_candidato
        (participacion_ref,candidato_ref,acta_ref,instantanea_ref,version_instantanea,registrada_en)
      VALUES (v_participacion,v_candidato,p_acta_ref,v_constitucion.instantanea_ref,
              v_constitucion.version_instantanea,p_registrada_en);
      v_nuevos := v_nuevos+1;
    END IF;
  END LOOP;
  RETURN jsonb_build_object('nuevos',v_nuevos,'existentes',v_existentes);
END $funcion$;

REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.rellenar_vinculos_candidato_v1(text,jsonb,timestamptz) FROM PUBLIC;
-- No se concede al ejecutor de aplicación: esta operación de mantenimiento
-- solo se invoca tras SET ROLE propietario desde la conexión administrativa.
COMMIT;
