\set ON_ERROR_STOP on
-- Retira CT-000110. La aplicación que consulta la fachada v3 debe retirarse
-- antes; v1/v2 y la publicación CT37 no se tocan.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '60s';
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_contratacion_temporal:migracion:000110', 0)
);
LOCK TABLE vec_contratacion_temporal.publicacion_version_rrhh IN SHARE MODE;

DO $prevalidacion$
BEGIN
    IF pg_catalog.to_regclass(
        'vec_contratacion_temporal.fase_entrada_publicacion_rrhh'
    ) IS NULL
    OR pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v3(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
    ) IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para retirar CT-000110';
    END IF;
END
$prevalidacion$;

DROP FUNCTION vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v3(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
    bytea, bytea, bytea, bytea, numeric, numeric,
    bytea, bytea, bytea, bytea
) RESTRICT;
DROP FUNCTION vec_contratacion_temporal.expedientes_contenido_cuadro_rrhh_v1(bytea)
RESTRICT;
DROP TRIGGER publicacion_version_rrhh_fase_entrada
    ON vec_contratacion_temporal.publicacion_version_rrhh;
DROP TABLE vec_contratacion_temporal.fase_entrada_publicacion_rrhh RESTRICT;
DROP FUNCTION vec_contratacion_temporal.registrar_fase_entrada_publicacion_rrhh_v1()
RESTRICT;
COMMIT;
