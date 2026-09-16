\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000103',0));

CREATE TABLE vec_contratacion_temporal.numeracion_expedientes (
 anio smallint PRIMARY KEY CHECK (anio BETWEEN 1 AND 9999),
 ultimo integer NOT NULL CHECK (ultimo BETWEEN 0 AND 999999)
);
CREATE FUNCTION vec_contratacion_temporal.siguiente_numero_visible_v1(p_anio integer)
RETURNS text LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path = pg_catalog AS $f$
DECLARE v_ultimo integer;
BEGIN
 IF p_anio NOT BETWEEN 1 AND 9999 THEN RAISE EXCEPTION USING ERRCODE='22023'; END IF;
 INSERT INTO vec_contratacion_temporal.numeracion_expedientes(anio,ultimo) VALUES(p_anio,0)
 ON CONFLICT (anio) DO NOTHING;
 UPDATE vec_contratacion_temporal.numeracion_expedientes SET ultimo=ultimo+1
 WHERE anio=p_anio RETURNING ultimo INTO v_ultimo;
 IF v_ultimo IS NULL OR v_ultimo > 999999 THEN RAISE EXCEPTION USING ERRCODE='22003'; END IF;
 RETURN format('%s/CT-%s', lpad(p_anio::text,4,'0'), lpad(v_ultimo::text,6,'0'));
END $f$;
REVOKE ALL ON TABLE vec_contratacion_temporal.numeracion_expedientes FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.siguiente_numero_visible_v1(integer) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.siguiente_numero_visible_v1(integer) TO vec_contratacion_temporal_ejecutor;
COMMIT;
