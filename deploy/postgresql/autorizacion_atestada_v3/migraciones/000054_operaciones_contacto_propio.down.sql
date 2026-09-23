\set ON_ERROR_STOP on
-- AD3-54 se revierte exclusivamente en base de ensayo sin operaciones,
-- concesiones, consumos ni auditoría. Requiere postimagen exacta y orden
-- Contacto3 DOWN → T13/8 DOWN → AD3-54 DOWN.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contacto_usuario_v1:dependencias:v1',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000054',0));
DO $incompleta$ BEGIN RAISE EXCEPTION 'AD3-54 DOWN WIP: falta inversión y guarda de historia' USING ERRCODE='55000'; END $incompleta$;
COMMIT;
