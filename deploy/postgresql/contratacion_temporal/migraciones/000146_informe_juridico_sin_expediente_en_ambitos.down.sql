\set ON_ERROR_STOP on
-- CT-000146 (reversión): el contexto canónico del informe jurídico vuelve a
-- incluir «expediente_ref» en los ámbitos. Solo es coherente con un binario
-- anterior a los perfiles fijos de asignación e informe.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_contratacion_temporal:migracion:000146', 0)
);

DO $rol$
BEGIN
    IF current_user <> 'vec_contratacion_temporal_propietario' THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'CT-000146 exige el rol propietario';
    END IF;
END
$rol$;

DO $parche_inicial$
DECLARE
    v_def text;
    v_cuerpo text;
    v_antes text := $m$'","fase_previa":"' ||$m$;
    v_despues text := $m$'","expediente_ref":"' || r.expediente_ref ||
        '","fase_previa":"' ||$m$;
BEGIN
    SELECT p.prosrc, pg_catalog.pg_get_functiondef(p.oid)
      INTO STRICT v_cuerpo, v_def
      FROM pg_catalog.pg_proc p
     WHERE p.oid = pg_catalog.to_regprocedure(
               'vec_contratacion_temporal.confirmar_informe_juridico_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')
       AND p.proowner = 'vec_contratacion_temporal_propietario'::regrole;
    IF pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(v_cuerpo, 'UTF8')), 'hex') <>
       '28bbfc9c2850b776eb8796049ecb09cd90236a48c49b5f33ca302f655bd11286'
       OR pg_catalog.strpos(v_def, v_antes) = 0
       OR pg_catalog.strpos(pg_catalog.substr(v_def, pg_catalog.strpos(v_def, v_antes) + 1), v_antes) <> 0 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'informe jurídico (inicial) sin CT-000146 o alterado';
    END IF;
    EXECUTE pg_catalog.replace(v_def, v_antes, v_despues);
END
$parche_inicial$;

DO $parche_subsanacion$
DECLARE
    v_def text;
    v_cuerpo text;
    v_antes text := $m$'","fase_previa":"' ||$m$;
    v_despues text := $m$'","expediente_ref":"' || r.expediente_ref ||
        '","fase_previa":"' ||$m$;
BEGIN
    SELECT p.prosrc, pg_catalog.pg_get_functiondef(p.oid)
      INTO STRICT v_cuerpo, v_def
      FROM pg_catalog.pg_proc p
     WHERE p.oid = pg_catalog.to_regprocedure(
               'vec_contratacion_temporal.confirmar_informe_juridico_tras_subsanacion_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')
       AND p.proowner = 'vec_contratacion_temporal_propietario'::regrole;
    IF pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(v_cuerpo, 'UTF8')), 'hex') <>
       '5d55e5d59c074f84e1f6b2d391185293dec201ae07b386a964150bad2faea4f1'
       OR pg_catalog.strpos(v_def, v_antes) = 0
       OR pg_catalog.strpos(pg_catalog.substr(v_def, pg_catalog.strpos(v_def, v_antes) + 1), v_antes) <> 0 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'informe jurídico (subsanacion) sin CT-000146 o alterado';
    END IF;
    EXECUTE pg_catalog.replace(v_def, v_antes, v_despues);
END
$parche_subsanacion$;

COMMIT;
