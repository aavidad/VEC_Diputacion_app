\set ON_ERROR_STOP on
-- Retirada de CT123. Se deniega si algún informe nuevo tras subsanar consta
-- ya reservado o emitido, o si se publicó alguna política de informe nuevo:
-- su historia depende de estas funciones y tablas. La confirmación de CT93
-- recupera exactamente su cuerpo anterior.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_contratacion_temporal:migracion:000123', 0));
DO $pre$
BEGIN
    IF pg_catalog.to_regprocedure('vec_contratacion_temporal.preparar_informe_juridico_tras_subsanacion_v1(jsonb)') IS NULL THEN
        RAISE EXCEPTION 'CT123 no instalada' USING ERRCODE = '55000';
    END IF;
    IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.politica_informe_tras_subsanacion) THEN
        RAISE EXCEPTION 'CT123: reversión denegada, hay política de informe nuevo publicada' USING ERRCODE = '55000';
    END IF;
    IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.reserva_informe_juridico
                WHERE version_expediente <> 4) THEN
        RAISE EXCEPTION 'CT123: reversión denegada, hay informes nuevos tras subsanar' USING ERRCODE = '55000';
    END IF;
END
$pre$;
-- Devuelve a la confirmación de CT93 su cuerpo exacto previo a CT123.
DO $refiscalizacion$
DECLARE
    v_ancla text := E'    v_antecedente := vec_contratacion_temporal.antecedente_refiscalizacion_v1(v_actual.agregado_json);\n';
    v_insercion text := E'    -- CT123: con la política publicada que exige informe nuevo tras\n    -- subsanar, no se fiscaliza de nuevo hasta emitirlo para este retorno.\n    IF vec_contratacion_temporal.informe_nuevo_exigido_ct123()\n       AND vec_contratacion_temporal.informe_nuevo_admisible_ct123(v_actual.agregado_json) THEN\n        RAISE EXCEPTION USING ERRCODE = ''55000'',\n            MESSAGE = ''informe jurídico nuevo tras subsanación pendiente'';\n    END IF;\n';
    v_antes record; v_despues record; v_definicion text;
BEGIN
    SELECT p.oid, pg_catalog.pg_get_functiondef(p.oid) AS definicion, p.proacl AS acl,
           p.proowner AS propietario, p.proconfig AS configuracion
      INTO v_antes FROM pg_catalog.pg_proc p
     WHERE p.oid = pg_catalog.to_regprocedure('vec_contratacion_temporal.confirmar_fiscalizacion_tras_subsanacion_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')
       AND p.proowner = 'vec_contratacion_temporal_propietario'::regrole AND p.prosecdef;
    IF NOT FOUND
       OR pg_catalog.length(v_antes.definicion)
          - pg_catalog.length(pg_catalog.replace(v_antes.definicion, v_ancla || v_insercion, ''))
          <> pg_catalog.length(v_ancla || v_insercion) THEN
        RAISE EXCEPTION 'CT123: confirmación de CT93 con cuerpo inesperado' USING ERRCODE = '55000';
    END IF;
    v_definicion := pg_catalog.replace(v_antes.definicion, v_ancla || v_insercion, v_ancla);
    IF pg_catalog.strpos(v_definicion, 'ct123') <> 0 THEN
        RAISE EXCEPTION 'CT123: confirmación de CT93 con cuerpo inesperado' USING ERRCODE = '55000';
    END IF;
    EXECUTE v_definicion;
    SELECT pg_catalog.pg_get_functiondef(p.oid) AS definicion, p.proacl AS acl,
           p.proowner AS propietario, p.proconfig AS configuracion
      INTO STRICT v_despues FROM pg_catalog.pg_proc p WHERE p.oid = v_antes.oid;
    IF v_despues.definicion IS DISTINCT FROM v_definicion
       OR v_despues.acl IS DISTINCT FROM v_antes.acl
       OR v_despues.propietario IS DISTINCT FROM v_antes.propietario
       OR v_despues.configuracion IS DISTINCT FROM v_antes.configuracion THEN
        RAISE EXCEPTION 'CT123: confirmación de CT93 alterada al revertir' USING ERRCODE = '55000';
    END IF;
END
$refiscalizacion$;
DROP FUNCTION vec_contratacion_temporal.informe_nuevo_exigido_ct123();
DROP FUNCTION vec_contratacion_temporal.publicar_politica_informe_tras_subsanacion_v1(boolean,text);
DROP TABLE vec_contratacion_temporal.politica_informe_tras_subsanacion;
DROP FUNCTION vec_contratacion_temporal.inicio_ronda_informe_nuevo_v1(text,text);
DROP FUNCTION vec_contratacion_temporal.confirmar_informe_juridico_tras_subsanacion_v1(
    jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_contratacion_temporal.preparar_informe_juridico_tras_subsanacion_v1(jsonb);
DROP FUNCTION vec_contratacion_temporal.informe_nuevo_admisible_ct123(jsonb);
ALTER TABLE vec_contratacion_temporal.reserva_informe_juridico
    DROP CONSTRAINT reserva_informe_juridico_version_expediente_check,
    ADD CONSTRAINT reserva_informe_juridico_version_expediente_check
        CHECK (version_expediente = 4);
COMMIT;
