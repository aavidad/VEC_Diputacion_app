\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_dietas_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
LOCK TABLE vec_dietas.borrador_comision, vec_dietas.calculo_comision IN ACCESS EXCLUSIVE MODE;
DO $pre$
BEGIN
 IF current_user<>'vec_dietas_propietario' OR to_regclass('vec_dietas.calculo_comision') IS NULL
    OR to_regprocedure('vec_dietas.crear_o_recuperar_comision_calculada_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_dietas.recuperar_comision_por_clave_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_dietas.consultar_comisiones_calculadas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR EXISTS(SELECT 1 FROM vec_dietas.calculo_comision)
    OR EXISTS(SELECT 1 FROM vec_dietas.borrador_comision)
 THEN RAISE EXCEPTION 'Dietas 000004: DOWN con historia o estructura incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
DROP FUNCTION vec_dietas.consultar_comisiones_calculadas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_dietas.recuperar_comision_por_clave_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_dietas.crear_o_recuperar_comision_calculada_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP TABLE vec_dietas.calculo_comision;
GRANT EXECUTE ON FUNCTION vec_dietas.crear_o_recuperar_borrador_propio_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea), vec_dietas.consultar_borradores_propios_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_dietas_ejecutor;
COMMIT;
