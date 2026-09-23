\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
SET LOCAL search_path=pg_catalog;
DO $guardia$ BEGIN
 IF to_regclass('vec_identidad_sesiones_v1.registro_propio_v1') IS NOT NULL
 THEN RAISE EXCEPTION 'retirar primero el registro de Identidad sin historia' USING ERRCODE='55000'; END IF;
END $guardia$;
REVOKE EXECUTE ON FUNCTION vec_contexto_actor_v1.registrar_persona_registro_propio_v1(text,text,text,text,boolean,text,numeric,text,text,timestamptz,timestamptz) FROM vec_identidad_sesiones_v1_propietario;
DROP FUNCTION vec_contexto_actor_v1.registrar_persona_registro_propio_v1(text,text,text,text,boolean,text,numeric,text,text,timestamptz,timestamptz);
COMMIT;
