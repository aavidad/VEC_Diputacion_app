-- La fachada no conserva datos propios; los vínculos y constituciones previos siguen intactos.
\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_bolsa_llamamientos:migracion:000009', 0)
);
REVOKE USAGE ON SCHEMA vec_bolsa_llamamientos
    FROM vec_bolsa_llamamientos_consultor_participaciones_propias;
DROP FUNCTION vec_bolsa_llamamientos.consultar_participaciones_propias_v1(
    bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea
) RESTRICT;
COMMIT;
