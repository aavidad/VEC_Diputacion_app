\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000015',0));
-- Una acción nueva en el perfil Bolsa existente, no otro consumidor ni
-- autoridad de CT. El proveedor debe acreditar justificante y evaluación
-- de plazo antes de emitir material para esta acción; aquí no se inventan.
DO $ampliar$
DECLARE
    v_def text; v_acl aclitem[];
    v_anterior text := $operaciones$               AND (
                   c ->> 'operacion' IS NOT DISTINCT FROM 'bolsa.orden.preparar'
                   OR c ->> 'operacion' IS NOT DISTINCT FROM 'bolsa.llamamiento.abrir'
               )$operaciones$;
    v_nuevo text := $operaciones$               AND (
                   c ->> 'operacion' IS NOT DISTINCT FROM 'bolsa.orden.preparar'
                   OR c ->> 'operacion' IS NOT DISTINCT FROM 'bolsa.llamamiento.abrir'
                   OR c ->> 'operacion' IS NOT DISTINCT FROM 'bolsa.llamamiento.aceptacion_rrhh.registrar'
               )$operaciones$;
BEGIN
    IF to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_bolsa_llamamiento_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
       OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_respuesta_recibida_rrhh_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL THEN
        RAISE EXCEPTION 'faltan consumidores previos de Bolsa/respuesta' USING ERRCODE='55000';
    END IF;
    SELECT pg_get_functiondef(p.oid),p.proacl INTO STRICT v_def,v_acl
      FROM pg_proc p
     WHERE p.oid='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
       AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef;
    IF length(v_def)-length(replace(v_def,v_anterior,''))<>length(v_anterior)
       OR strpos(v_def,'bolsa.llamamiento.aceptacion_rrhh.registrar')<>0 THEN
        RAISE EXCEPTION 'núcleo incompatible para aceptación RRHH de Bolsa' USING ERRCODE='55000';
    END IF;
    EXECUTE replace(v_def,v_anterior,v_nuevo);
    IF (SELECT proacl FROM pg_proc WHERE oid='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure) IS DISTINCT FROM v_acl THEN
        RAISE EXCEPTION 'aceptación alteró permisos del núcleo' USING ERRCODE='55000';
    END IF;
END
$ampliar$;
COMMIT;
