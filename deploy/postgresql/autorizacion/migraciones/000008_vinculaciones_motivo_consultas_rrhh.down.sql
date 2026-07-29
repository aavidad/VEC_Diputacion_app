-- Retirada estructural segura. Nunca borra una publicacion ni un checkpoint
-- avanzado; la evidencia exige un procedimiento de conservacion separado.
BEGIN;

SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_autorizacion:migracion:vinculaciones-motivo-rrhh:000008', 0
    )
);

SET LOCAL ROLE vec_autorizacion_propietario;

LOCK TABLE
    vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1,
    vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1
    IN ACCESS EXCLUSIVE MODE;

DO $conservacion$
BEGIN
    IF EXISTS (
        SELECT 1
          FROM vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1
    )
       OR (SELECT pg_catalog.count(*)
             FROM vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1
            WHERE clase_consulta IN ('cuadro', 'detalle')
              AND ultima_publicacion_version = 0
              AND ultima_publicacion_ref IS NULL
              AND ultima_publicacion_huella_sha256 IS NULL) <> 2
       OR (SELECT pg_catalog.count(*)
             FROM vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1)
          <> 2 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = '000008 no se revierte: existe evidencia de vinculaciones RRHH';
    END IF;
END
$conservacion$;

DROP TABLE
    vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1;
DROP TABLE vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1;
DROP FUNCTION
    vec_autorizacion.validar_avance_vinculacion_motivo_rrhh_v1();
DROP FUNCTION
    vec_autorizacion.bloquear_mutacion_vinculacion_motivo_rrhh_v1();
ALTER TABLE vec_autorizacion.motivo_v2_catalogo_publicado
    DROP CONSTRAINT motivo_v2_catalogo_referencia_completa_unica;

COMMIT;
