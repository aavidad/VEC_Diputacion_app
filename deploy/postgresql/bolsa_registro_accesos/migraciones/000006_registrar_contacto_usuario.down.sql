\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_accesos_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contacto_usuario_v1:dependencias:v1',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_registro_accesos:migracion:000006',0));
LOCK TABLE vec_bolsa_registro_accesos.registro_acceso IN ACCESS EXCLUSIVE MODE;
DO $historia$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_proc WHERE pronamespace=to_regnamespace('vec_contacto_usuario_v1')) THEN
        RAISE EXCEPTION 'contacto: retirar fachadas del almacén antes de DOWN' USING ERRCODE='55000';
    END IF;
    IF EXISTS (SELECT 1 FROM vec_bolsa_registro_accesos.registro_acceso
        WHERE action IN ('vec.contacto_usuario.alta','vec.contacto_usuario.actualizar','vec.contacto_usuario.consultar')) THEN
        RAISE EXCEPTION 'T13/6: historia de contacto conservada; no admite DOWN' USING ERRCODE='55000';
    END IF;
END $historia$;
DROP FUNCTION vec_bolsa_registro_accesos.registrar_contacto_usuario_v1(bytea,bytea,bytea,bytea,bytea,text,text,text) RESTRICT;
-- USAGE se conserva: puede ser dependencia de otra fachada nominal posterior.
COMMIT;
