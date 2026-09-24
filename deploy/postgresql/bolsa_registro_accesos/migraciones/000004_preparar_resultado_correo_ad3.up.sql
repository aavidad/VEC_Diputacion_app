\set ON_ERROR_STOP on
-- Dependencia nominal de T13/5: resolver la función AD3 existente sin
-- conceder EXECUTE, CREATE ni acceso a tablas al propietario T13.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_registro_accesos:migracion:000004',0));
DO $pre$
BEGIN
    IF current_user<>'vec_autorizacion_atestada_v3_propietario'
       OR NOT EXISTS (SELECT 1 FROM pg_namespace WHERE nspname='vec_autorizacion_atestada_v3'
            AND nspowner=current_user::regrole)
       OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_bolsa_accesos_propietario'
            AND NOT rolcanlogin AND NOT rolinherit AND NOT rolsuper AND NOT rolbypassrls)
       OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_resultado_correo_llamamiento_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
       OR has_schema_privilege('vec_bolsa_accesos_propietario','vec_autorizacion_atestada_v3','USAGE')
       OR has_schema_privilege('vec_bolsa_accesos_propietario','vec_autorizacion_atestada_v3','CREATE') THEN
        RAISE EXCEPTION 'T13/4: preimagen AD3 o ACL incompatible' USING ERRCODE='55000';
    END IF;
END $pre$;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_bolsa_accesos_propietario;
DO $post$
BEGIN
    IF NOT has_schema_privilege('vec_bolsa_accesos_propietario','vec_autorizacion_atestada_v3','USAGE')
       OR has_schema_privilege('vec_bolsa_accesos_propietario','vec_autorizacion_atestada_v3','CREATE')
       OR has_function_privilege('vec_bolsa_accesos_propietario',
            'vec_autorizacion_atestada_v3.registrar_y_consumir_resultado_correo_llamamiento_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE') THEN
        RAISE EXCEPTION 'T13/4: concesión excedida' USING ERRCODE='55000';
    END IF;
END $post$;
COMMIT;
