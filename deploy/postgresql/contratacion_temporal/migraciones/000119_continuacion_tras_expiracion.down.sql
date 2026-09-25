\set ON_ERROR_STOP on
-- Reversión de CT119. Se deniega si alguna continuación cuelga de una
-- expiración: la restricción de CT60 no la admitiría y la historia es de solo
-- adición. Las continuaciones de renuncia confirmadas con la versión 2 se
-- conservan: la restricción original de CT60 las sigue describiendo.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000119',0));
LOCK TABLE vec_contratacion_temporal.resolucion_manual_respuesta_rrhh IN ACCESS EXCLUSIVE MODE;
DO $preservar$
BEGIN
    IF to_regprocedure('vec_contratacion_temporal.continuar_llamamiento_rrhh_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL THEN
        RAISE EXCEPTION 'CT119 no instalada' USING ERRCODE='55000';
    END IF;
    IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.resolucion_manual_respuesta_rrhh
        WHERE continuacion_clave IS NOT NULL AND solicitud_json->>'Respuesta'='expiracion_gobernada') THEN
        RAISE EXCEPTION 'reversión denegada: continuación tras expiración existente' USING ERRCODE='55000';
    END IF;
END
$preservar$;
DROP FUNCTION vec_contratacion_temporal.continuar_llamamiento_rrhh_v2(
    text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DO $restriccion$
DECLARE v_def text;
    v_anterior text := $a$((solicitud_json ->> 'Respuesta'::text) = ANY (ARRAY['renuncia'::text, 'expiracion_gobernada'::text]))$a$;
    v_nuevo text := $n$((solicitud_json ->> 'Respuesta'::text) = 'renuncia'::text)$n$;
BEGIN
    SELECT pg_get_constraintdef(oid) INTO STRICT v_def FROM pg_constraint
     WHERE conrelid='vec_contratacion_temporal.resolucion_manual_respuesta_rrhh'::regclass
       AND conname='continuacion_confirmacion_completa' AND contype='c' AND convalidated;
    IF length(v_def)-length(replace(v_def,v_anterior,''))<>length(v_anterior) THEN
        RAISE EXCEPTION 'restricción de continuación incompatible' USING ERRCODE='55000';
    END IF;
    EXECUTE 'ALTER TABLE vec_contratacion_temporal.resolucion_manual_respuesta_rrhh '
        ||'DROP CONSTRAINT continuacion_confirmacion_completa, '
        ||'ADD CONSTRAINT continuacion_confirmacion_completa '||replace(v_def,v_anterior,v_nuevo);
END
$restriccion$;
COMMIT;
