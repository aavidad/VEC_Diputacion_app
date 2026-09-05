\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000014', 0));
LOCK TABLE vec_autorizacion_atestada_v3.atestacion_decision_v3 IN SHARE MODE;
DO $retirar$
DECLARE
    v_def text; v_acl aclitem[];
    v_extension text := $perfil$           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'respuesta_recibida_rrhh'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM
                   'vec_contratacion_temporal.confirmar_alta_atestada.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM
                   'contratacion_temporal.llamamiento.respuesta.registrar'
               AND d ->> 'accion' IS NOT DISTINCT FROM
                   'contratacion_temporal.llamamiento.respuesta.registrar'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM
                   'respuesta_recibida_llamamiento_contratacion_temporal'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM
                   'gestionar_contratacion_temporal'
           )
$perfil$;
BEGIN
    IF to_regprocedure('vec_contratacion_temporal.registrar_respuesta_recibida_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
       OR to_regclass('vec_contratacion_temporal.respuesta_recibida_rrhh') IS NOT NULL
       OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.atestacion_decision_v3 a
           WHERE convert_from(a.capacidad_canonica,'UTF8')::jsonb->>'operacion' =
             'contratacion_temporal.llamamiento.respuesta.registrar') THEN
        RAISE EXCEPTION 'reversión denegada: consumidor o historia de respuesta recibida' USING ERRCODE = '55000';
    END IF;
    SELECT pg_get_functiondef(p.oid),p.proacl INTO STRICT v_def,v_acl
      FROM pg_proc p
     WHERE p.oid = 'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
       AND p.proowner = 'vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef;
    IF length(v_def)-length(replace(v_def,v_extension,'')) <> length(v_extension) THEN
        RAISE EXCEPTION 'núcleo incompatible para retirar respuesta recibida' USING ERRCODE = '55000';
    END IF;
    EXECUTE replace(v_def,v_extension,'');
    IF (SELECT proacl FROM pg_proc WHERE oid = 'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure) IS DISTINCT FROM v_acl THEN
        RAISE EXCEPTION 'reversión alteró permisos del núcleo' USING ERRCODE = '55000';
    END IF;
END
$retirar$;

-- Solo se alcanza tras la guarda anterior de ausencia de consumidor/historia,
-- dentro de la misma transacción y bajo su bloqueo de atestaciones.
DO $fechas_reversa$
DECLARE
    v_def text; v_acl aclitem[];
    v_fecha_textual text := $textual$d ->> 'valida_hasta' <> c ->> 'decision_valida_hasta'$textual$;
    v_fecha_instante text := $instante$(d ->> 'valida_hasta')::timestamptz <> (c ->> 'decision_valida_hasta')::timestamptz$instante$;
BEGIN
    SELECT pg_get_functiondef(p.oid),p.proacl INTO STRICT v_def,v_acl
      FROM pg_proc p
     WHERE p.oid = 'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
       AND p.proowner = 'vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef;
    IF length(v_def)-length(replace(v_def,v_fecha_instante,'')) <> length(v_fecha_instante)
       OR strpos(v_def,v_fecha_textual) <> 0 THEN
        RAISE EXCEPTION 'núcleo incompatible para revertir ligadura temporal VEC-AD-3' USING ERRCODE = '55000';
    END IF;
    EXECUTE replace(v_def,v_fecha_instante,v_fecha_textual);
    IF (SELECT proacl FROM pg_proc WHERE oid = 'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure) IS DISTINCT FROM v_acl THEN
        RAISE EXCEPTION 'reversión temporal alteró permisos del núcleo' USING ERRCODE = '55000';
    END IF;
END
$fechas_reversa$;
DROP FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_respuesta_recibida_rrhh_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
COMMIT;
