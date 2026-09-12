\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_accesos_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended(
    'vec_administracion.dependencias.configuracion_correo.v1', 0));
SELECT pg_advisory_xact_lock(hashtextextended(
    'vec_bolsa_registro_accesos:migracion:000003', 0));
-- Los append no toman los advisory de migración. Este bloqueo espera las
-- escrituras previas y excluye nuevos INSERT hasta terminar la comprobación
-- y el DROP, conservando intacta toda la historia.
LOCK TABLE vec_bolsa_registro_accesos.registro_acceso IN SHARE ROW EXCLUSIVE MODE;
DO $historia$
BEGIN
    IF EXISTS (SELECT 1 FROM vec_bolsa_registro_accesos.registro_acceso
                WHERE module_id='vec.module.administracion'
                  AND action='administracion.configuracion_correo.consultar') THEN
        RAISE EXCEPTION 'T13/3 DOWN rechazado: existe historia administrativa'
            USING ERRCODE='55000';
    END IF;
    -- PL/pgSQL no declara dependencias de llamadas en prosrc. Una integración
    -- ADMIN que siga nombrando la fachada también impide retirarla.
    IF EXISTS (SELECT 1 FROM pg_proc
                WHERE pronamespace=to_regnamespace('vec_administracion')
                  AND strpos(prosrc,'registrar_consulta_configuracion_correo_admin_v1')>0) THEN
        RAISE EXCEPTION 'T13/3 DOWN rechazado: ADMIN conserva la integracion'
            USING ERRCODE='55000';
    END IF;
END $historia$;
DROP FUNCTION vec_bolsa_registro_accesos.registrar_consulta_configuracion_correo_admin_v1(jsonb,text,text,bigint) RESTRICT;
-- Se conserva USAGE: no se retiran privilegios de otros wrappers posteriores.
COMMIT;
