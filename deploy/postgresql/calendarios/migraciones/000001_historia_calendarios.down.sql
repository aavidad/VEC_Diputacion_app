\set ON_ERROR_STOP on
-- Retirada no destructiva: solo procede sobre una historia vacía. Con
-- cualquier versión registrada falla con 55000 y no toca nada.
BEGIN;
SET LOCAL ROLE vec_calendarios_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_calendarios:migracion:000001:historia:v1',0));
LOCK TABLE vec_calendarios.version_calendario, vec_calendarios.dia_calendario IN ACCESS EXCLUSIVE MODE;
DO $post$ BEGIN
 IF EXISTS (SELECT 1 FROM vec_calendarios.version_calendario) OR EXISTS (SELECT 1 FROM vec_calendarios.dia_calendario) THEN
   RAISE EXCEPTION 'Calendarios 000001: la historia no se retira' USING ERRCODE='55000';
 END IF;
END $post$;
DROP FUNCTION vec_calendarios.centros_con_calendario_v1(integer,timestamptz);
DROP FUNCTION vec_calendarios.versiones_vigentes_v1(integer,text[],text[],timestamptz);
DROP TABLE vec_calendarios.dia_calendario;
DROP TABLE vec_calendarios.version_calendario;
DROP FUNCTION vec_calendarios.historia_inmutable();
DROP FUNCTION vec_calendarios.antes_de_insertar_dia();
DROP FUNCTION vec_calendarios.antes_de_insertar_version();
COMMIT;
