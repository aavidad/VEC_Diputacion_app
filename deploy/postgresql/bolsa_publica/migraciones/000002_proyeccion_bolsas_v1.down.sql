-- Reversion solo para una instalacion nueva controlada. No usar sobre historia
-- conservada: retira B10 e invalida el ancla conjunta V3 antes de restaurar V2.
\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path = pg_catalog;
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_bolsa_publica:migracion:000002', 0)
);
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_bolsa_publica:publicacion:v2', 0)
);
DO $prevalidacion$
BEGIN
    IF current_user <> 'vec_bolsa_publica_migrador'
       OR NOT pg_catalog.pg_has_role(current_user, 'vec_bolsa_publica_propietario', 'SET')
       OR NOT pg_catalog.pg_has_role(current_user, 'vec_bolsa_publica_publicacion_propietario', 'SET') THEN
        RAISE EXCEPTION USING ERRCODE = '42501', MESSAGE = 'reversion B10 rechazada: identidad incorrecta';
    END IF;
END
$prevalidacion$;
SET LOCAL ROLE vec_bolsa_publica_propietario;
DELETE FROM vec_bolsa_publica_datos.bolsa_publica;
UPDATE vec_bolsa_publica_datos.fuente
   SET manifiesto_sha256 = pg_catalog.repeat('0', 64)
 WHERE manifiesto_sha256 <> pg_catalog.repeat('0', 64);
DROP VIEW vec_bolsa_publica_lectura.posiciones_bolsa_v1;
DROP VIEW vec_bolsa_publica_lectura.bolsas_v1;
DROP TABLE vec_bolsa_publica_datos.posicion_bolsa_publica;
DROP TABLE vec_bolsa_publica_datos.bolsa_publica;
DROP FUNCTION vec_bolsa_publica_datos.grupos_b10_validos(text[]);
SET LOCAL ROLE vec_bolsa_publica_publicacion_propietario;
DROP FUNCTION vec_bolsa_publica_publicacion.publicar_proyeccion_v3(jsonb, jsonb, text);
GRANT EXECUTE ON FUNCTION vec_bolsa_publica_publicacion.publicar_proyeccion_v2(jsonb, text)
    TO vec_bolsa_publica_publicador_login;
COMMIT;
