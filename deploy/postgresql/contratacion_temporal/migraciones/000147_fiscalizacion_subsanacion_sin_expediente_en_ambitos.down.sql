\set ON_ERROR_STOP on
-- CT-000147 (reversión): el contexto canónico de la fiscalización y de la
-- subsanación vuelve a incluir «expediente_ref» en los ámbitos. Solo es
-- coherente con un binario anterior a los perfiles fijos de esas operaciones.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_contratacion_temporal:migracion:000147', 0)
);

DO $rol$
BEGIN
    IF current_user <> 'vec_contratacion_temporal_propietario' THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'CT-000147 exige el rol propietario';
    END IF;
END
$rol$;

DO $parche_fiscalizacion$
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
               'vec_contratacion_temporal.confirmar_fiscalizacion_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')
       AND p.proowner = 'vec_contratacion_temporal_propietario'::regrole;
    IF pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(v_cuerpo, 'UTF8')), 'hex') <>
       '6a30b018c4ee3e1fa6f55084e2ebfc6d661ea40ad899a94a7481ec65413fc250'
       OR pg_catalog.strpos(v_def, v_antes) = 0
       OR pg_catalog.strpos(pg_catalog.substr(v_def, pg_catalog.strpos(v_def, v_antes) + 1), v_antes) <> 0 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'fiscalizacion: función sin CT-000147 o alterada';
    END IF;
    EXECUTE pg_catalog.replace(v_def, v_antes, v_despues);
END
$parche_fiscalizacion$;

DO $parche_refiscalizacion$
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
               'vec_contratacion_temporal.confirmar_fiscalizacion_tras_subsanacion_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')
       AND p.proowner = 'vec_contratacion_temporal_propietario'::regrole;
    IF pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(v_cuerpo, 'UTF8')), 'hex') <>
       '7220cbe98bc19fe044e194affade7c8ca61f12dd1d2c8ba1d72cc79d27956af0'
       OR pg_catalog.strpos(v_def, v_antes) = 0
       OR pg_catalog.strpos(pg_catalog.substr(v_def, pg_catalog.strpos(v_def, v_antes) + 1), v_antes) <> 0 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'refiscalizacion: función sin CT-000147 o alterada';
    END IF;
    EXECUTE pg_catalog.replace(v_def, v_antes, v_despues);
END
$parche_refiscalizacion$;

DO $parche_modificacion$
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
               'vec_contratacion_temporal.confirmar_fiscalizacion_tras_modificacion_ct120(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')
       AND p.proowner = 'vec_contratacion_temporal_propietario'::regrole;
    IF pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(v_cuerpo, 'UTF8')), 'hex') <>
       '405d043d7e442f705fdac5dcc4c16498b6b551b3dbdfda29bd9fede5470f0d8d'
       OR pg_catalog.strpos(v_def, v_antes) = 0
       OR pg_catalog.strpos(pg_catalog.substr(v_def, pg_catalog.strpos(v_def, v_antes) + 1), v_antes) <> 0 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'modificacion: función sin CT-000147 o alterada';
    END IF;
    EXECUTE pg_catalog.replace(v_def, v_antes, v_despues);
END
$parche_modificacion$;

DO $parche_subsanacion$
DECLARE
    v_def text;
    v_cuerpo text;
    v_antes text := $m$"fase_previa"$m$;
    v_despues text := $m$"expediente_ref":"'||(m->>'expediente_ref')||
        '","fase_previa"$m$;
BEGIN
    SELECT p.prosrc, pg_catalog.pg_get_functiondef(p.oid)
      INTO STRICT v_cuerpo, v_def
      FROM pg_catalog.pg_proc p
     WHERE p.oid = pg_catalog.to_regprocedure(
               'vec_contratacion_temporal.confirmar_subsanacion_reparos_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')
       AND p.proowner = 'vec_contratacion_temporal_propietario'::regrole;
    IF pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(v_cuerpo, 'UTF8')), 'hex') <>
       '42d976e96151e330559068705f0a0211d80e365b56c36f8cae8408a67930dc36'
       OR pg_catalog.strpos(v_def, v_antes) = 0
       OR pg_catalog.strpos(pg_catalog.substr(v_def, pg_catalog.strpos(v_def, v_antes) + 1), v_antes) <> 0 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'subsanacion: función sin CT-000147 o alterada';
    END IF;
    EXECUTE pg_catalog.replace(v_def, v_antes, v_despues);
END
$parche_subsanacion$;

COMMIT;
