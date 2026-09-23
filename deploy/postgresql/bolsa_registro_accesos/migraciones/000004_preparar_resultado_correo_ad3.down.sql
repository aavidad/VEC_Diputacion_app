\set ON_ERROR_STOP on
-- DOWN solo en ensayo vacío antes de T13/5; jamás con historial conservado.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_registro_accesos:migracion:000004',0));
DO $pre$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
       OR to_regprocedure('vec_bolsa_registro_accesos.registrar_resultado_correo_llamamiento_ct_v1(bytea,text,bytea,bytea,text,text,text,boolean,text)') IS NOT NULL
       OR to_regprocedure('vec_bolsa_registro_accesos.registrar_contacto_usuario_v1(bytea,bytea,bytea,bytea,bytea,text,text,text)') IS NOT NULL
       OR EXISTS (SELECT 1 FROM vec_bolsa_registro_accesos.registro_acceso)
       OR NOT has_schema_privilege('vec_bolsa_accesos_propietario','vec_autorizacion_atestada_v3','USAGE') THEN
        RAISE EXCEPTION 'T13/4: dependencias o historia conservada' USING ERRCODE='55000';
    END IF;
END $pre$;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
REVOKE USAGE ON SCHEMA vec_autorizacion_atestada_v3 FROM vec_bolsa_accesos_propietario;
COMMIT;
