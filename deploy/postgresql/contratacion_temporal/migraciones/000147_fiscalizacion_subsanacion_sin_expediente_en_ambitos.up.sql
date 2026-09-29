\set ON_ERROR_STOP on
-- CT-000147: los permisos de fiscalización (Intervención) y de subsanación de
-- reparos (RRHH) ya no llevan el expediente en sus ámbitos.
--
-- Los perfiles fijos cubren la organización y las fases y estados previos que
-- fija el catálogo, no un expediente concreto. El expediente sigue ligado a la
-- decisión por su referencia de recurso (recurso_ref = expediente_ref de la
-- reserva) y por la reserva misma. Esta migración solo quita «expediente_ref»
-- de los ámbitos del contexto canónico que recomponen las cuatro funciones de
-- confirmación (fiscalización inicial, tras subsanación y tras modificación, y
-- subsanación de reparos), para que coincida con la huella de la aplicación.
--
-- No cambia tablas, filas, permisos ni historial. Una operación preparada
-- antes con la huella anterior no se confirma: su repetición responde sin
-- efectos.
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
    v_antes text := $m$'","expediente_ref":"' || r.expediente_ref ||
        '","fase_previa":"' ||$m$;
    v_despues text := $m$'","fase_previa":"' ||$m$;
BEGIN
    SELECT p.prosrc, pg_catalog.pg_get_functiondef(p.oid)
      INTO STRICT v_cuerpo, v_def
      FROM pg_catalog.pg_proc p
     WHERE p.oid = pg_catalog.to_regprocedure(
               'vec_contratacion_temporal.confirmar_fiscalizacion_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')
       AND p.proowner = 'vec_contratacion_temporal_propietario'::regrole;
    IF pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(v_cuerpo, 'UTF8')), 'hex') <>
       'bb21608248394f30ef120618c26c5a753d93d0ba12fd98f69a6953ea9154aecf'
       OR pg_catalog.strpos(v_def, v_antes) = 0
       OR pg_catalog.strpos(pg_catalog.substr(v_def, pg_catalog.strpos(v_def, v_antes) + 1), v_antes) <> 0 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'fiscalizacion: función previa incompatible con CT-000147';
    END IF;
    EXECUTE pg_catalog.replace(v_def, v_antes, v_despues);
END
$parche_fiscalizacion$;

DO $parche_refiscalizacion$
DECLARE
    v_def text;
    v_cuerpo text;
    v_antes text := $m$'","expediente_ref":"' || r.expediente_ref ||
        '","fase_previa":"' ||$m$;
    v_despues text := $m$'","fase_previa":"' ||$m$;
BEGIN
    SELECT p.prosrc, pg_catalog.pg_get_functiondef(p.oid)
      INTO STRICT v_cuerpo, v_def
      FROM pg_catalog.pg_proc p
     WHERE p.oid = pg_catalog.to_regprocedure(
               'vec_contratacion_temporal.confirmar_fiscalizacion_tras_subsanacion_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')
       AND p.proowner = 'vec_contratacion_temporal_propietario'::regrole;
    IF pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(v_cuerpo, 'UTF8')), 'hex') <>
       '031188998be8d21cfe8112591ef8aa1af4cb180a14f855ff7dd010176de9ec97'
       OR pg_catalog.strpos(v_def, v_antes) = 0
       OR pg_catalog.strpos(pg_catalog.substr(v_def, pg_catalog.strpos(v_def, v_antes) + 1), v_antes) <> 0 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'refiscalizacion: función previa incompatible con CT-000147';
    END IF;
    EXECUTE pg_catalog.replace(v_def, v_antes, v_despues);
END
$parche_refiscalizacion$;

DO $parche_modificacion$
DECLARE
    v_def text;
    v_cuerpo text;
    v_antes text := $m$'","expediente_ref":"' || r.expediente_ref ||
        '","fase_previa":"' ||$m$;
    v_despues text := $m$'","fase_previa":"' ||$m$;
BEGIN
    SELECT p.prosrc, pg_catalog.pg_get_functiondef(p.oid)
      INTO STRICT v_cuerpo, v_def
      FROM pg_catalog.pg_proc p
     WHERE p.oid = pg_catalog.to_regprocedure(
               'vec_contratacion_temporal.confirmar_fiscalizacion_tras_modificacion_ct120(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')
       AND p.proowner = 'vec_contratacion_temporal_propietario'::regrole;
    IF pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(v_cuerpo, 'UTF8')), 'hex') <>
       '5e80b2268356fd6bdc636cc108044d51b4baffb4b81ab789f7a7c9db3fc48d71'
       OR pg_catalog.strpos(v_def, v_antes) = 0
       OR pg_catalog.strpos(pg_catalog.substr(v_def, pg_catalog.strpos(v_def, v_antes) + 1), v_antes) <> 0 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'modificacion: función previa incompatible con CT-000147';
    END IF;
    EXECUTE pg_catalog.replace(v_def, v_antes, v_despues);
END
$parche_modificacion$;

DO $parche_subsanacion$
DECLARE
    v_def text;
    v_cuerpo text;
    v_antes text := $m$"expediente_ref":"'||(m->>'expediente_ref')||
        '","fase_previa"$m$;
    v_despues text := $m$"fase_previa"$m$;
BEGIN
    SELECT p.prosrc, pg_catalog.pg_get_functiondef(p.oid)
      INTO STRICT v_cuerpo, v_def
      FROM pg_catalog.pg_proc p
     WHERE p.oid = pg_catalog.to_regprocedure(
               'vec_contratacion_temporal.confirmar_subsanacion_reparos_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')
       AND p.proowner = 'vec_contratacion_temporal_propietario'::regrole;
    IF pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(v_cuerpo, 'UTF8')), 'hex') <>
       'b3763b7f953e1ca2762a0ac278028cea09f29fe847fa7bc497c17723da3529ca'
       OR pg_catalog.strpos(v_def, v_antes) = 0
       OR pg_catalog.strpos(pg_catalog.substr(v_def, pg_catalog.strpos(v_def, v_antes) + 1), v_antes) <> 0 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'subsanacion: función previa incompatible con CT-000147';
    END IF;
    EXECUTE pg_catalog.replace(v_def, v_antes, v_despues);
END
$parche_subsanacion$;

COMMIT;
