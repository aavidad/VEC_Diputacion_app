\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000060',0));
LOCK TABLE vec_contratacion_temporal.resolucion_manual_respuesta_rrhh IN ACCESS EXCLUSIVE MODE;
DO $preservar$
BEGIN
    IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.resolucion_manual_respuesta_rrhh
        WHERE num_nonnulls(continuacion_clave,continuacion_material,continuacion_material_sha256,
            continuacion_recibo,continuacion_actor_ref,continuacion_perfil_ref,continuacion_consumo)>0) THEN
        RAISE EXCEPTION 'reversión denegada: confirmación de continuación existente' USING ERRCODE='55000';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_trigger
        WHERE tgrelid='vec_contratacion_temporal.resolucion_manual_respuesta_rrhh'::regclass
          AND tgname='historia_inmutable' AND NOT tgisinternal AND tgenabled='O' AND tgtype=27
          AND tgfoid='vec_contratacion_temporal.proteger_historia_continuacion_ct_v1()'::regprocedure
          AND tgnargs=0 AND tgqual IS NULL) THEN
        RAISE EXCEPTION 'trigger de continuación incompatible' USING ERRCODE='55000';
    END IF;
END
$preservar$;
DROP FUNCTION vec_contratacion_temporal.continuar_llamamiento_rrhh_v1(
    text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP TRIGGER historia_inmutable ON vec_contratacion_temporal.resolucion_manual_respuesta_rrhh;
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE
    ON vec_contratacion_temporal.resolucion_manual_respuesta_rrhh FOR EACH ROW
    EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();
DROP FUNCTION vec_contratacion_temporal.proteger_historia_continuacion_ct_v1();
DROP INDEX vec_contratacion_temporal.continuacion_recibo_unico;
ALTER TABLE vec_contratacion_temporal.resolucion_manual_respuesta_rrhh
    DROP CONSTRAINT continuacion_confirmacion_completa,
    DROP CONSTRAINT continuacion_clave_unica,
    DROP COLUMN continuacion_consumo,
    DROP COLUMN continuacion_perfil_ref,
    DROP COLUMN continuacion_actor_ref,
    DROP COLUMN continuacion_recibo,
    DROP COLUMN continuacion_material_sha256,
    DROP COLUMN continuacion_material,
    DROP COLUMN continuacion_clave;
-- No borra historia CT ni autorizaciones V3: retirar el consumidor 19 es otra
-- operación protegida que rechaza cualquier historia de acceso o confirmación.
COMMIT;
