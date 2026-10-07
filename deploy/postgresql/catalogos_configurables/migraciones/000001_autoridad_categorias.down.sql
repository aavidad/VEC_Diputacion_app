-- Reversión exclusiva de instalación vacía. Nunca sobre historia real.
BEGIN;
SET LOCAL ROLE vec_catalogos_configurables_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_catalogos_configurables:migracion:000001', 0));
LOCK TABLE vec_catalogos_configurables.publicacion,
    vec_catalogos_configurables.entrada_publicada,
    vec_catalogos_configurables.categoria_control,
    vec_catalogos_configurables.uso,
    vec_catalogos_configurables.historia IN ACCESS EXCLUSIVE MODE;
DO $pre$
BEGIN
    IF current_user <> 'vec_catalogos_configurables_propietario'
       OR pg_catalog.to_regnamespace('vec_catalogos_configurables') IS NULL
       OR EXISTS (SELECT 1 FROM vec_catalogos_configurables.publicacion)
       OR EXISTS (SELECT 1 FROM vec_catalogos_configurables.entrada_publicada)
       OR EXISTS (SELECT 1 FROM vec_catalogos_configurables.categoria_control)
       OR EXISTS (SELECT 1 FROM vec_catalogos_configurables.uso)
       OR EXISTS (SELECT 1 FROM vec_catalogos_configurables.historia) THEN
        RAISE EXCEPTION 'DOWN de catalogos rechazado: hay historia o preimagen incompatible' USING ERRCODE = '55000';
    END IF;
END $pre$;
REVOKE EXECUTE ON FUNCTION vec_catalogos_configurables.publicar(text,integer,text,text,jsonb,text,text,text,text,text,text,text),
    vec_catalogos_configurables.reservar(text,text,text,text,integer,text,text,text,text,text),
    vec_catalogos_configurables.terminar_uso(text,text,text,text,text,text,text,text),
    vec_catalogos_configurables.cambiar_proyeccion(text,bigint,text,text,bigint,text,text,text,text)
    FROM vec_autorizacion_atestada_v3_propietario;
REVOKE USAGE ON SCHEMA vec_catalogos_configurables FROM vec_autorizacion_atestada_v3_propietario;
DROP FUNCTION vec_catalogos_configurables.cambiar_proyeccion(text,bigint,text,text,bigint,text,text,text,text);
DROP FUNCTION vec_catalogos_configurables.terminar_uso(text,text,text,text,text,text,text,text);
DROP FUNCTION vec_catalogos_configurables.reservar(text,text,text,text,integer,text,text,text,text,text);
DROP FUNCTION vec_catalogos_configurables.publicar(text,integer,text,text,jsonb,text,text,text,text,text,text,text);
DROP TABLE vec_catalogos_configurables.historia;
DROP TABLE vec_catalogos_configurables.uso;
DROP TABLE vec_catalogos_configurables.categoria_control;
DROP TABLE vec_catalogos_configurables.entrada_publicada;
DROP TABLE vec_catalogos_configurables.publicacion;
DROP FUNCTION vec_catalogos_configurables.rechazar_cambio_inmutable();
ALTER DEFAULT PRIVILEGES FOR ROLE vec_catalogos_configurables_propietario GRANT EXECUTE ON FUNCTIONS TO PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_catalogos_configurables_propietario GRANT USAGE ON TYPES TO PUBLIC;
COMMIT;
