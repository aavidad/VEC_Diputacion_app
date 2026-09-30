-- Solo para una instalacion desechable sin AD3-117 ni consumidores.
-- No altera publicaciones, controles, usos ni historia.
BEGIN;
SET LOCAL ROLE vec_catalogos_configurables_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_catalogos_configurables:migracion:000002', 0));
DO $pre$
BEGIN
    IF current_user <> 'vec_catalogos_configurables_propietario'
       OR pg_catalog.to_regprocedure('vec_catalogos_configurables.listar_habilitadas(text,text,integer)') IS NULL
       OR pg_catalog.to_regprocedure('vec_catalogos_configurables.leer_publicacion_categoria(text,integer,text,text)') IS NULL
       OR pg_catalog.to_regprocedure('vec_catalogos_configurables.consultar_uso(text,text,text)') IS NULL THEN
        RAISE EXCEPTION 'catalogos 000002 DOWN: preimagen incompatible' USING ERRCODE = '55000';
    END IF;
END $pre$;
REVOKE EXECUTE ON FUNCTION vec_catalogos_configurables.listar_habilitadas(text,text,integer),
    vec_catalogos_configurables.leer_publicacion_categoria(text,integer,text,text),
    vec_catalogos_configurables.consultar_uso(text,text,text) FROM vec_autorizacion_atestada_v3_propietario;
DROP FUNCTION vec_catalogos_configurables.consultar_uso(text,text,text);
DROP FUNCTION vec_catalogos_configurables.leer_publicacion_categoria(text,integer,text,text);
DROP FUNCTION vec_catalogos_configurables.listar_habilitadas(text,text,integer);
COMMIT;
