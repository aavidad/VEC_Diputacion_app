\set ON_ERROR_STOP on
-- CT-000144: el permiso del análisis ya no lleva el expediente en sus ámbitos.
--
-- El perfil fijo de RRHH para el análisis cubre la organización y la fase y
-- el estado previos que fija el catálogo, no un expediente concreto. El
-- expediente sigue ligado a la decisión por su referencia de recurso, que
-- confirmar_operacion_analisis_v1 compara con la reserva (recurso_ref =
-- expediente_ref). Esta migración solo quita la clave «expediente_ref» de los
-- ámbitos con que PostgreSQL recompone la huella de contexto de la decisión,
-- para que coincida con la que calcula la aplicación.
--
-- No cambia tablas, filas, permisos ni historial. Una operación preparada
-- antes con la huella anterior no se confirma: su repetición con la misma
-- clave responde «clave reutilizada», sin efectos.
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
    v_antes text := $antes$) || ',"expediente_ref":' ||
        vec_contratacion_temporal.texto_json_go_v1(
            o ->> 'expediente_ref'
        ) || ',"fase_previa":' ||$antes$;
    v_despues text := $despues$) || ',"fase_previa":' ||$despues$;
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
       '55b2cdc253d75aa446c730afcfd52eb36c60826c4909707c0f882c24ed167743'
       OR pg_catalog.strpos(v_def, v_antes) = 0
       OR pg_catalog.strpos(pg_catalog.substr(v_def, pg_catalog.strpos(v_def, v_antes) + 1), v_antes) <> 0 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'huella de contexto del análisis previa incompatible';
    END IF;
    EXECUTE pg_catalog.replace(v_def, v_antes, v_despues);
END
$parche$;
COMMIT;
