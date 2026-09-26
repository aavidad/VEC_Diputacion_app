\set ON_ERROR_STOP on
-- Retira CT-000126 solo sin historia: si ya se publicó algún formato propio,
-- retirarla borraría ese hecho y volvería al formato de siempre sin rastro, y
-- se rechaza. Devuelve siguiente_numero_visible_v1 a su cuerpo de CT-000103
-- (mismo contador; los números asignados no cambian). La aplicación que
-- publica los parámetros debe retirarse antes.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '60s';
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_contratacion_temporal:migracion:000126', 0)
);

DO $prevalidacion$
BEGIN
    IF pg_catalog.to_regclass(
        'vec_contratacion_temporal.numeracion_parametros'
    ) IS NULL
    OR pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.publicar_numeracion_parametros_v1(text,integer,text)'
    ) IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'CT-000126 no instalada';
    END IF;
END
$prevalidacion$;

LOCK TABLE vec_contratacion_temporal.numeracion_parametros
    IN ACCESS EXCLUSIVE MODE;

DO $historia$
BEGIN
    IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.numeracion_parametros) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'CT-000126: reversión denegada, hay parámetros de numeración publicados';
    END IF;
END
$historia$;

DROP FUNCTION vec_contratacion_temporal.publicar_numeracion_parametros_v1(text, integer, text);

CREATE OR REPLACE FUNCTION vec_contratacion_temporal.siguiente_numero_visible_v1(p_anio integer)
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

DROP TABLE vec_contratacion_temporal.numeracion_parametros;
COMMIT;
