\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000018',0));
-- Una acción en el perfil Bolsa vigente. No nuevo consumidor, tabla ni crypto.
-- Se exige 17/16 y se conservan literalmente sus perfiles y todas las ACL.
DO $cambio$
DECLARE
    v_def text; v_acl aclitem[];
    v_anterior text := $operaciones$                   OR c ->> 'operacion' IS NOT DISTINCT FROM 'bolsa.llamamiento.aceptacion_rrhh.registrar'$operaciones$;
    v_nuevo text := $operaciones$                   OR c ->> 'operacion' IS NOT DISTINCT FROM 'bolsa.llamamiento.aceptacion_rrhh.registrar'
                   OR c ->> 'operacion' IS NOT DISTINCT FROM 'bolsa.llamamiento.renuncia_rrhh.registrar'$operaciones$;
BEGIN
    IF to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_resolucion_manual_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL THEN
        RAISE EXCEPTION 'falta consumidor previo de revisión manual' USING ERRCODE='55000';
    END IF;
    SELECT pg_get_functiondef(p.oid),p.proacl INTO STRICT v_def,v_acl FROM pg_proc p
     WHERE p.oid='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
       AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef;
    IF length(v_def)-length(replace(v_def,v_anterior,''))<>length(v_anterior)
       OR strpos(v_def,'bolsa.llamamiento.renuncia_rrhh.registrar')<>0
       OR strpos(v_def,'p_perfil_mutacion IS NOT DISTINCT FROM ''resolucion_manual_respuesta_ct''')=0
       OR strpos(v_def,'p_perfil_mutacion IS NOT DISTINCT FROM ''consulta_justificante_respuesta_ct''')=0 THEN
        RAISE EXCEPTION 'núcleo incompatible para renuncia RRHH' USING ERRCODE='55000';
    END IF;
    EXECUTE replace(v_def,v_anterior,v_nuevo);
    IF (SELECT proacl FROM pg_proc WHERE oid='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure) IS DISTINCT FROM v_acl THEN
        RAISE EXCEPTION 'renuncia alteró permisos del núcleo' USING ERRCODE='55000';
    END IF;
END
$cambio$;
COMMIT;
