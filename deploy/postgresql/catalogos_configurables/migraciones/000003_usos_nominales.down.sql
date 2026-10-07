\set ON_ERROR_STOP on
-- Sólo para clon desechable sin consumidores. No ejecutar en una instalación.
BEGIN;
SET LOCAL ROLE vec_catalogos_configurables_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='2min';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_catalogos_configurables:migracion:000003',0));
LOCK TABLE vec_catalogos_configurables.evidencia_terminal,
    vec_catalogos_configurables.uso IN ACCESS EXCLUSIVE MODE;
DO $pre$
BEGIN
    IF current_user <> 'vec_catalogos_configurables_propietario'
       OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.reservar_uso_categoria_rpt_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
       OR EXISTS (SELECT 1 FROM vec_catalogos_configurables.evidencia_terminal)
       OR EXISTS (SELECT 1 FROM vec_catalogos_configurables.uso) THEN
        RAISE EXCEPTION 'catalogos configurables 000003: hay consumidores' USING ERRCODE='55000';
    END IF;
END $pre$;
DROP FUNCTION vec_catalogos_configurables.obtener_uso_publicacion(text,text,text);
DROP FUNCTION vec_catalogos_configurables.terminar_uso_con_evidencia(text,text,text,text,text,text,text,text,text,text);
DROP TABLE vec_catalogos_configurables.evidencia_terminal;
COMMIT;
