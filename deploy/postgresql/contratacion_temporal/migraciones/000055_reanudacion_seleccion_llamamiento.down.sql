\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
LOCK TABLE vec_contratacion_temporal.historia_reanudacion_seleccion_llamamiento,
    vec_contratacion_temporal.outbox_reanudacion_seleccion_llamamiento IN ACCESS EXCLUSIVE MODE;
DO $proteger$
BEGIN
    IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.historia_reanudacion_seleccion_llamamiento)
       OR EXISTS (SELECT 1 FROM vec_contratacion_temporal.outbox_reanudacion_seleccion_llamamiento) THEN
        RAISE EXCEPTION 'reversión denegada: se conserva historia de reanudación' USING ERRCODE = '55000';
    END IF;
END
$proteger$;

-- Restaura exactamente los dos fragmentos de 000046.
DO $terminal$
DECLARE
    v_def text; v_acl aclitem[];
    v_anterior text := $anterior$    ELSIF v_ejecucion.situacion <> 'confirmada' THEN
        RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'terminal O6 no confirmado';
    ELSIF v_consulta->>'organizacion_ref' IS DISTINCT FROM v_ejecucion.solicitud_json->>'organizacion_ref'$anterior$;
    v_autoridad text := $autoridad$    ELSIF v_consulta->>'organizacion_ref' IS DISTINCT FROM v_ejecucion.solicitud_json->>'organizacion_ref'$autoridad$;
    v_marca text := $marca$        RAISE EXCEPTION USING ERRCODE = '42501', MESSAGE = 'replay O6 denegado';
    ELSE
$marca$;
    v_extension text := $extension$        RAISE EXCEPTION USING ERRCODE = '42501', MESSAGE = 'replay O6 denegado';
    ELSIF v_ejecucion.situacion = 'indeterminada'
       AND v_ejecucion.efecto = 'preparar_orden'
       AND v_ejecucion.ventana_orden_abierta IS TRUE
       AND v_ejecucion.ventana_llamamiento_abierta IS FALSE
       AND v_ejecucion.recibo_json IS NULL AND v_ejecucion.artefacto_canonico IS NULL THEN
        RETURN QUERY SELECT '', '', '', '', '', '';
    ELSIF v_ejecucion.situacion <> 'confirmada' THEN
        RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'terminal O6 no confirmado';
    ELSE
$extension$;
BEGIN
    SELECT pg_get_functiondef(p.oid),p.proacl INTO STRICT v_def,v_acl
      FROM pg_proc p WHERE p.oid = 'vec_contratacion_temporal.resolver_terminal_autorizado_seleccion_llamamiento_o6_v2(uuid,text)'::regprocedure
       AND p.proowner = 'vec_contratacion_temporal_propietario'::regrole AND p.prosecdef;
    IF length(v_def)-length(replace(v_def,v_autoridad,'')) <> length(v_autoridad)
       OR length(v_def)-length(replace(v_def,v_extension,'')) <> length(v_extension) THEN
        RAISE EXCEPTION 'terminal O6 incompatible con reversión' USING ERRCODE = '55000';
    END IF;
    v_def := replace(v_def,v_extension,v_marca);
    v_def := replace(v_def,v_autoridad,v_anterior);
    EXECUTE v_def;
    IF (SELECT proacl FROM pg_proc WHERE oid = 'vec_contratacion_temporal.resolver_terminal_autorizado_seleccion_llamamiento_o6_v2(uuid,text)'::regprocedure) IS DISTINCT FROM v_acl THEN
        RAISE EXCEPTION 'cambio de terminal alteró permisos' USING ERRCODE = '55000';
    END IF;
END
$terminal$;
DROP FUNCTION vec_contratacion_temporal.reanudar_preparacion_orden_seleccion_v1(
    text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP TABLE vec_contratacion_temporal.outbox_reanudacion_seleccion_llamamiento;
DROP TABLE vec_contratacion_temporal.historia_reanudacion_seleccion_llamamiento;
-- La ejecución y todas sus ventanas permanecen intactas.
COMMIT;
