\set ON_ERROR_STOP on
-- Solo para una instalación sin historia; no ejecutar en bases conservadas.
BEGIN;
SET LOCAL ROLE vec_usuarios_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_usuarios:migracion:000003',0));
LOCK TABLE vec_usuarios.denegacion_frontera_preferencias IN ACCESS EXCLUSIVE MODE;
DO $guard$ BEGIN
 IF NOT pg_catalog.pg_has_role(session_user,'vec_usuarios_migrador','MEMBER')
    OR EXISTS (SELECT 1 FROM vec_usuarios.denegacion_frontera_preferencias)
 THEN RAISE EXCEPTION 'Usuarios: historia impide retirar frontera' USING ERRCODE='55000'; END IF;
END $guard$;
REVOKE EXECUTE ON FUNCTION vec_usuarios.registrar_denegacion_preferencias_v1(text,text,text,text,text)
 FROM vec_usuarios_registrador_frontera_interno,vec_usuarios_registrador_frontera_externo;
REVOKE USAGE ON SCHEMA vec_usuarios FROM vec_usuarios_registrador_frontera_interno,
 vec_usuarios_registrador_frontera_externo;
DROP FUNCTION vec_usuarios.registrar_denegacion_preferencias_v1(text,text,text,text,text);
DROP TABLE vec_usuarios.denegacion_frontera_preferencias;
COMMIT;
