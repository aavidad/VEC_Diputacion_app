\set ON_ERROR_STOP on
-- Fuente preparada sobre AD3-133/135/136. DOWN cerrado por defecto.
-- AD3-136 tampoco publica una cadena inversa medida. Retirar el perfil RPT
-- requiere clon desechable sin historia, preimagen global post-AD3-134,
-- huellas PG18 de la cadena inversa y dos revisiones del orden completo.
-- Nunca ejecutar DOWN sobre Cat1/2/3, AD117/126 o gobierno con historia.
BEGIN;
SET LOCAL search_path=pg_catalog, pg_temp;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
DO $denegar$
BEGIN
    RAISE EXCEPTION 'AD3-134 DOWN: falta cadena inversa PG18 medida post-AD3-136'
        USING ERRCODE='55000';
END $denegar$;
COMMIT;
