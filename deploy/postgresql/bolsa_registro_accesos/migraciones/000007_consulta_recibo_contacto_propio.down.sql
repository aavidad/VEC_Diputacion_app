\set ON_ERROR_STOP on
-- Fuente candidata: consulta del recibo histórico propio; no acredita replay de Guardar.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contacto_usuario_v1:dependencias:v1',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_registro_accesos:migracion:000007',0));
LOCK TABLE vec_bolsa_registro_accesos.registro_acceso IN ACCESS EXCLUSIVE MODE;
DO $historia$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
       OR to_regprocedure('vec_contacto_usuario_v1.consultar_recibo_contacto_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea)') IS NOT NULL
       OR to_regprocedure('vec_contacto_usuario_v1.consultar_version_contacto_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea)') IS NOT NULL
       OR EXISTS (SELECT 1 FROM vec_bolsa_registro_accesos.registro_acceso
            WHERE action='vec.contacto_usuario.consultar' AND purpose='gestion_contacto_propio'
                AND metadata ? 'recibo_encontrado')
       OR EXISTS (SELECT 1 FROM vec_bolsa_registro_accesos.registro_acceso
            WHERE action IN ('vec.contacto_usuario.version_propia','vec.contacto_usuario.version_para_llamamiento')
                AND metadata ? 'version_encontrada') THEN
        RAISE EXCEPTION 'T13/7: dependencias o historia conservada; no admite DOWN' USING ERRCODE='55000';
    END IF;
END $historia$;
SET LOCAL ROLE vec_bolsa_accesos_propietario;
DROP FUNCTION vec_bolsa_registro_accesos.registrar_consulta_version_contacto_v1(bytea,bytea,bytea,bytea,bytea,text,text,text,boolean,numeric) RESTRICT;
DROP FUNCTION vec_bolsa_registro_accesos.registrar_consulta_recibo_contacto_v1(bytea,bytea,bytea,bytea,bytea,text,text,text,bytea) RESTRICT;
COMMIT;
