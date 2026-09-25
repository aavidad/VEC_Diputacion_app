\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_contratacion_temporal:migracion:000117', 0)
);

-- Solo retira una lectura: no hay historia propia que conservar. La historia
-- de versiones del expediente queda intacta.
DO $prevalidacion$
BEGIN
    IF pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.consultar_cambios_expediente_rrhh_atestado_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
    ) IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'no está instalada la consulta de cambios del expediente';
    END IF;
END
$prevalidacion$;

DROP FUNCTION vec_contratacion_temporal.consultar_cambios_expediente_rrhh_atestado_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_detalle_rrhh_v1,
    bytea, bytea, bytea, bytea, numeric, numeric,
    bytea, bytea, bytea, bytea
) RESTRICT;
DROP FUNCTION vec_contratacion_temporal.hojas_instantanea_expediente_v1(jsonb) RESTRICT;
DROP FUNCTION vec_contratacion_temporal.valor_traza_cambio_v1(jsonb) RESTRICT;
COMMIT;
