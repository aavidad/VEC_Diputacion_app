\set ON_ERROR_STOP on
-- Sólo para una instalación sin consumidores ni historia. No ejecutar en la
-- principal conservada ni tras una incorporación que haya usado esta fachada.
BEGIN;
SET LOCAL search_path = pg_catalog, pg_temp;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_contexto_actor_v1:migracion:persona_candidato_incorporacion:000018', 0));
-- B67 UP toma este mismo cerrojo antes de comprobar CTX18. Impide que el
-- consumidor se instale entre la comprobación de catálogo y el DROP.
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_bolsa_llamamientos:migracion:000067', 0));
DO $preimagen$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles
                    WHERE rolname = current_user AND rolsuper)
       OR pg_catalog.to_regprocedure(
           'vec_contexto_actor_v1.acreditar_persona_candidato_incorporacion_v1(text)') IS NULL THEN
        RAISE EXCEPTION 'ContextoActor 000018 DOWN: preimagen incompatible'
            USING ERRCODE = '55000';
    END IF;
    IF pg_catalog.to_regprocedure(
           'vec_bolsa_llamamientos.consultar_persona_aceptacion_ct_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')
       IS NOT NULL THEN
        RAISE EXCEPTION 'ContextoActor 000018 DOWN: consumidor Bolsa 000067 instalado'
            USING ERRCODE = '55000';
    END IF;
END
$preimagen$;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
DROP FUNCTION vec_contexto_actor_v1.acreditar_persona_candidato_incorporacion_v1(text);
DO $uso$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_proc p
                    WHERE p.pronamespace = 'vec_contexto_actor_v1'::regnamespace
                      AND pg_catalog.has_function_privilege(
                          'vec_bolsa_llamamientos_propietario', p.oid, 'EXECUTE')) THEN
        REVOKE USAGE ON SCHEMA vec_contexto_actor_v1
            FROM vec_bolsa_llamamientos_propietario;
    END IF;
END
$uso$;
COMMIT;
