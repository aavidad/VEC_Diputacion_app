\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_contratacion_temporal:000047:candidatura-alta:o2-r3b', 0
));
DO $prevalidacion$
BEGIN
    IF pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.confirmar_alta_atestada_v1(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea)'
       ) IS NULL
       OR pg_catalog.to_regclass(
           'vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6'
       ) IS NULL
       OR pg_catalog.to_regclass(
           'vec_contratacion_temporal.candidatura_alta_tecnica'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para candidatura de alta';
    END IF;
END
$prevalidacion$;
\ir 000047_componentes/010_candidatura_y_aliases.sql
\ir 000047_componentes/020_resolucion_candidatura.sql
\ir 000047_componentes/030_confirmacion_ligada.sql
\ir 000047_componentes/090_acl_y_barrera.sql
COMMIT;
