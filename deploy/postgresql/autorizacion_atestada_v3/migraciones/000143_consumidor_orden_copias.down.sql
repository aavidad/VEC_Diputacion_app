\set ON_ERROR_STOP on
-- AD3-143 DOWN documental. La retirada de la capacidad se hace desactivando
-- consumidores y volviendo a un artefacto compatible; se conserva la historia.
-- No retirar el núcleo ni sus roles después de consumir órdenes/aprobaciones.
-- Para un ensayo nuevo, restaurar el snapshot privado en otro clon efímero.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL lock_timeout='5s'; SET LOCAL statement_timeout='30s';
DO $retirada$
BEGIN
 RAISE EXCEPTION 'AD3-143: DOWN no ejecutable; conservar historia y restaurar el clon para un ensayo nuevo'
 USING ERRCODE='55000';
END $retirada$;
ROLLBACK;
