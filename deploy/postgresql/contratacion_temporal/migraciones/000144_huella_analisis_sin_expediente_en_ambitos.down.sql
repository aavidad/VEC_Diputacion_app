\set ON_ERROR_STOP on
-- CT-000144 (reversión): la huella de contexto del análisis vuelve a incluir
-- «expediente_ref» en los ámbitos. Solo es coherente con un binario anterior
-- al perfil fijo del análisis; con el actual, el análisis se denegaría.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_contratacion_temporal:migracion:000144', 0)
);

DO $parche$
DECLARE
    v_def text;
    v_cuerpo text;
    v_antes text := $antes$) || ',"fase_previa":' ||$antes$;
    v_despues text := $despues$) || ',"expediente_ref":' ||
        vec_contratacion_temporal.texto_json_go_v1(
            o ->> 'expediente_ref'
        ) || ',"fase_previa":' ||$despues$;
BEGIN
    IF current_user <> 'vec_contratacion_temporal_propietario' THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'CT-000144 exige el rol propietario';
    END IF;
    SELECT p.prosrc, pg_catalog.pg_get_functiondef(p.oid)
      INTO STRICT v_cuerpo, v_def
      FROM pg_catalog.pg_proc p
     WHERE p.oid = pg_catalog.to_regprocedure(
               'vec_contratacion_temporal.huella_contexto_recurso_analisis_v2(jsonb)')
       AND p.proowner = 'vec_contratacion_temporal_propietario'::regrole
       AND p.provolatile = 'i' AND p.proisstrict;
    IF pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(v_cuerpo, 'UTF8')), 'hex') <>
       'eccdebedd287e37410e5238136344b156c80098aac0b5acef3e3de1cfa5b8fa8'
       OR pg_catalog.strpos(v_def, v_antes) = 0
       OR pg_catalog.strpos(pg_catalog.substr(v_def, pg_catalog.strpos(v_def, v_antes) + 1), v_antes) <> 0 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'huella de contexto del análisis sin CT-000144 o alterada';
    END IF;
    EXECUTE pg_catalog.replace(v_def, v_antes, v_despues);
END
$parche$;
COMMIT;
