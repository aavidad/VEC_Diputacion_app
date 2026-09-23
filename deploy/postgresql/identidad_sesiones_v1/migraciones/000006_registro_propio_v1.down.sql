\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_identidad_sesiones_v1_propietario;
SET LOCAL search_path=pg_catalog;
DO $guardia$ BEGIN
 IF EXISTS(SELECT 1 FROM vec_identidad_sesiones_v1.registro_propio_v1)
    OR EXISTS(SELECT 1 FROM vec_identidad_sesiones_v1.sujeto_persona_registro_propio_v1)
 THEN RAISE EXCEPTION 'registro propio con historia: DOWN prohibido' USING ERRCODE='55000'; END IF;
END $guardia$;
REVOKE EXECUTE ON FUNCTION vec_identidad_sesiones_v1.registrar_propio_v1(bytea,bytea,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM vec_identidad_sesiones_v1_provisionador;
DROP FUNCTION vec_identidad_sesiones_v1.registrar_propio_v1(bytea,bytea,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_identidad_sesiones_v1.validar_cuenta_registro_propio_v1(text);
DROP TABLE vec_identidad_sesiones_v1.registro_propio_outbox_v1;
DROP TABLE vec_identidad_sesiones_v1.registro_propio_v1;
DROP TABLE vec_identidad_sesiones_v1.sujeto_persona_registro_propio_v1;
COMMIT;
