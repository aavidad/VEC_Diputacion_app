\set ON_ERROR_STOP on
-- Retira CT-000125 solo sin historia: si algún análisis ya declaró la
-- urgencia de un expediente, retirarla borraría ese hecho y cambiaría sus
-- plazos, y se rechaza. La aplicación que consulta la fachada v4 y registra
-- urgencias debe retirarse antes. CT-000110 (fachada v3) no se toca; su DOWN
-- exige retirar antes esta migración.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '60s';
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_contratacion_temporal:migracion:000125', 0)
);

DO $prevalidacion$
BEGIN
    IF pg_catalog.to_regclass(
        'vec_contratacion_temporal.urgencia_expediente_analisis'
    ) IS NULL
    OR pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.registrar_urgencia_analisis_v1(text,text)'
    ) IS NULL
    OR pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v4(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
    ) IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'CT-000125 no instalada';
    END IF;
END
$prevalidacion$;

LOCK TABLE vec_contratacion_temporal.urgencia_expediente_analisis
    IN ACCESS EXCLUSIVE MODE;

DO $historia$
BEGIN
    IF EXISTS (
        SELECT 1 FROM vec_contratacion_temporal.urgencia_expediente_analisis
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'reversión denegada: hay urgencias declaradas';
    END IF;
END
$historia$;

DROP FUNCTION vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v4(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
    bytea, bytea, bytea, bytea, numeric, numeric,
    bytea, bytea, bytea, bytea
) RESTRICT;
DROP FUNCTION vec_contratacion_temporal.registrar_urgencia_analisis_v1(text, text)
RESTRICT;
DROP TABLE vec_contratacion_temporal.urgencia_expediente_analisis RESTRICT;
COMMIT;
