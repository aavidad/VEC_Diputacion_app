\set ON_ERROR_STOP on
-- CT-000146: los permisos de asignación e informe jurídico ya no llevan el
-- expediente en sus ámbitos.
--
-- Los perfiles fijos de RRHH para la asignación a unidad y el informe
-- jurídico cubren la organización y las fases y estados previos que fija el
-- catálogo, no un expediente concreto. El expediente sigue ligado a la
-- decisión por su referencia de recurso (recurso_ref = expediente_ref de la
-- reserva) y por la reserva misma. La asignación no recompone la huella en
-- PostgreSQL; el informe sí, en sus dos funciones de confirmación (inicial y
-- tras subsanación): esta migración solo quita «expediente_ref» de los ámbitos
-- de ese JSON canónico, para que coincida con la huella de la aplicación.
--
-- No cambia tablas, filas, permisos ni historial. Un informe preparado antes
-- con la huella anterior no se confirma: su repetición responde sin efectos.
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
    v_antes text := $m$'","expediente_ref":"' || r.expediente_ref ||
        '","fase_previa":"' ||$m$;
    v_despues text := $m$'","fase_previa":"' ||$m$;
BEGIN
    SELECT p.prosrc, pg_catalog.pg_get_functiondef(p.oid)
      INTO STRICT v_cuerpo, v_def
      FROM pg_catalog.pg_proc p
     WHERE p.oid = pg_catalog.to_regprocedure(
               'vec_contratacion_temporal.confirmar_informe_juridico_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')
       AND p.proowner = 'vec_contratacion_temporal_propietario'::regrole;
    IF pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(v_cuerpo, 'UTF8')), 'hex') <>
       'bde7c18fd552f86124c9b4dc2dd7fb7864a3a6339cf545c876cb0498cf929d1e'
       OR pg_catalog.strpos(v_def, v_antes) = 0
       OR pg_catalog.strpos(pg_catalog.substr(v_def, pg_catalog.strpos(v_def, v_antes) + 1), v_antes) <> 0 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'informe jurídico (inicial) previo incompatible con CT-000146';
    END IF;
    EXECUTE pg_catalog.replace(v_def, v_antes, v_despues);
END
$parche_inicial$;

DO $parche_subsanacion$
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
               'vec_contratacion_temporal.confirmar_informe_juridico_tras_subsanacion_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')
       AND p.proowner = 'vec_contratacion_temporal_propietario'::regrole;
    IF pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(v_cuerpo, 'UTF8')), 'hex') <>
       'b1a469e8c58508eb5a7fefe0c5df58dd27a09bb1910cba044dc17df2ad641fde'
       OR pg_catalog.strpos(v_def, v_antes) = 0
       OR pg_catalog.strpos(pg_catalog.substr(v_def, pg_catalog.strpos(v_def, v_antes) + 1), v_antes) <> 0 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'informe jurídico (subsanacion) previo incompatible con CT-000146';
    END IF;
    EXECUTE pg_catalog.replace(v_def, v_antes, v_despues);
END
$parche_subsanacion$;

COMMIT;
