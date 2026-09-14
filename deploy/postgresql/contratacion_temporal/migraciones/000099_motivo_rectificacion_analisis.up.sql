\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000099',0));

-- La política transporta la clave catalogada. La actuación conserva la
-- observación nominal completa exigida por la transición v1 y el dominio.
-- No cambia el catálogo, los permisos, la reserva ni el historial existente.
DO $parche$
DECLARE
    v_def text;
    v_cuerpo text;
    v_antes text := $antes$o #>> '{politica,motivo_rectificacion_clave}' =
              o #>> '{actuacion,observaciones}'$antes$;
    v_despues text := $despues$('contratacion_temporal.analisis.rectificacion.' ||
               (o #>> '{politica,motivo_rectificacion_clave}')) =
              o #>> '{actuacion,observaciones}'$despues$;
BEGIN
    SELECT p.prosrc, pg_get_functiondef(p.oid) INTO STRICT v_cuerpo, v_def
    FROM pg_proc p
    WHERE p.oid=to_regprocedure('vec_contratacion_temporal.transicion_confirmacion_analisis_valida_v2(jsonb,jsonb)')
      AND p.proowner='vec_contratacion_temporal_propietario'::regrole
      AND p.provolatile='i' AND p.proisstrict;
    IF encode(sha256(convert_to(v_cuerpo,'UTF8')),'hex') <>
       'bd1f11cb8e6389e9f31e8cfcdc6addb8550fd4af16cd60cbe517ca192b1b44fb'
       OR position(v_antes IN v_def)=0 THEN
        RAISE EXCEPTION 'transición de análisis previa incompatible' USING ERRCODE='55000';
    END IF;
    EXECUTE replace(v_def,v_antes,v_despues);
END
$parche$;
-- La frontera interna elimina el motivo tras sellarlo. Al reconstruir la
-- política para verificar la huella, recupera la clave, no la observación.
DO $parche_huella$
DECLARE
    v_def text;
    v_cuerpo text;
    v_antes text := $antes$ELSE o #>> '{actuacion,observaciones}'$antes$;
    v_despues text := $despues$ELSE substring(o #>> '{actuacion,observaciones}'
                FROM '^contratacion_temporal[.]analisis[.]rectificacion[.](.+)$')$despues$;
BEGIN
    SELECT p.prosrc, pg_get_functiondef(p.oid) INTO STRICT v_cuerpo, v_def
    FROM pg_proc p
    WHERE p.oid=to_regprocedure('vec_contratacion_temporal.huella_contexto_recurso_analisis_v1(jsonb)')
      AND p.proowner='vec_contratacion_temporal_propietario'::regrole
      AND p.provolatile='i' AND p.proisstrict;
    IF encode(sha256(convert_to(v_cuerpo,'UTF8')),'hex') <>
       '1e2c6b55b3134b54867a62364a83fa38151a4c04d494b392e0b2d46083fd4113'
       OR position(v_antes IN v_def)=0 THEN
        RAISE EXCEPTION 'reconstrucción de motivo de análisis previa incompatible' USING ERRCODE='55000';
    END IF;
    EXECUTE replace(v_def,v_antes,v_despues);
END
$parche_huella$;
COMMIT;
