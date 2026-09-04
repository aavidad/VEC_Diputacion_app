\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_contratacion_temporal:000048:replay-alta:o2-07', 0
));
DO $prevalidacion$
BEGIN
    IF pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.confirmar_alta_atestada_v3(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea)'
       ) IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'replay de alta no instalado';
    END IF;
END
$prevalidacion$;
REVOKE ALL ON FUNCTION
    vec_contratacion_temporal.confirmar_alta_atestada_v3(
        bytea, bytea, bytea, bytea, numeric, numeric,
        bytea, bytea, bytea, bytea, bytea, bytea
    ) FROM PUBLIC, vec_contratacion_temporal_ejecutor;
DROP FUNCTION vec_contratacion_temporal.confirmar_alta_atestada_v3(
    bytea, bytea, bytea, bytea, numeric, numeric,
    bytea, bytea, bytea, bytea, bytea, bytea
);
DO $destipar_estado_previo$
DECLARE
    v_definicion text;
    v_sin_tipo text := E'ERRCODE = ''55000'',\n            MESSAGE = ''estado previo de alta incoherente'';';
    v_tipado text := E'ERRCODE = ''V2070'',\n            MESSAGE = ''estado previo de alta incoherente'';';
BEGIN
    SELECT pg_catalog.pg_get_functiondef(
        'vec_contratacion_temporal.confirmar_alta_atestada_v1(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea)'::regprocedure
    ) INTO STRICT v_definicion;
    IF pg_catalog.length(v_definicion) - pg_catalog.length(
           pg_catalog.replace(v_definicion, v_tipado, '')
       ) <> pg_catalog.length(v_tipado)
       OR pg_catalog.strpos(v_definicion, v_sin_tipo) <> 0 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'contrato tipado de estado previo incompatible';
    END IF;
    EXECUTE pg_catalog.replace(v_definicion, v_tipado, v_sin_tipo);
END
$destipar_estado_previo$;
GRANT EXECUTE ON FUNCTION
    vec_contratacion_temporal.confirmar_alta_atestada_v2(
        bytea, bytea, bytea, bytea, numeric, numeric,
        bytea, bytea, bytea, bytea, bytea, bytea
    ) TO vec_contratacion_temporal_ejecutor;
COMMIT;
