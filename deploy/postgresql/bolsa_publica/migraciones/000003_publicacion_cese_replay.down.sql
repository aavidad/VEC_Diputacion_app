-- Solo para una instalacion nueva sin publicaciones posteriores a 000003.
-- Nunca revertir sobre historia conservada.
\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path = pg_catalog;
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_publica:migracion:000003', 0));
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_publica:publicacion:v2', 0));
DO $prevalidacion$
BEGIN
    IF current_user <> 'vec_bolsa_publica_migrador'
       OR NOT pg_catalog.pg_has_role(current_user, 'vec_bolsa_publica_publicacion_propietario', 'SET')
       OR NOT pg_catalog.pg_has_role(current_user, 'vec_bolsa_publica_propietario', 'SET') THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'reversion de replay publico rechazada: identidad incorrecta';
    END IF;
END
$prevalidacion$;
SET LOCAL ROLE vec_bolsa_publica_publicacion_propietario;
DO $historia$
BEGIN
    IF EXISTS (SELECT 1 FROM vec_bolsa_publica_publicacion.testigo_replay_v3) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'reversion de replay publico rechazada: historia conservada';
    END IF;
END
$historia$;
DROP FUNCTION vec_bolsa_publica_publicacion.confirmar_replay_proyeccion_v3(jsonb,jsonb,text);
DROP FUNCTION vec_bolsa_publica_publicacion.publicar_proyeccion_v3(jsonb,jsonb,text);
DROP TABLE vec_bolsa_publica_publicacion.testigo_replay_v3;
ALTER FUNCTION vec_bolsa_publica_publicacion.publicar_proyeccion_v3_original(jsonb,jsonb,text)
    RENAME TO publicar_proyeccion_v3;
GRANT EXECUTE ON FUNCTION vec_bolsa_publica_publicacion.publicar_proyeccion_v3(jsonb,jsonb,text)
    TO vec_bolsa_publica_publicador_login;
SET LOCAL ROLE vec_bolsa_publica_propietario;
REVOKE SELECT (control_id, manifiesto_sha256, actualizada_en)
    ON vec_bolsa_publica_datos.fuente
    FROM vec_bolsa_publica_publicacion_propietario;
COMMIT;
