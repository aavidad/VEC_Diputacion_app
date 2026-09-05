\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000018',0));
LOCK TABLE vec_autorizacion_atestada_v3.atestacion_decision_v3 IN SHARE MODE;
DO $cambio$
DECLARE
    v_def text; v_acl aclitem[];
    v_anterior text := $operaciones$                   OR c ->> 'operacion' IS NOT DISTINCT FROM 'bolsa.llamamiento.aceptacion_rrhh.registrar'$operaciones$;
    v_nuevo text := $operaciones$                   OR c ->> 'operacion' IS NOT DISTINCT FROM 'bolsa.llamamiento.aceptacion_rrhh.registrar'
                   OR c ->> 'operacion' IS NOT DISTINCT FROM 'bolsa.llamamiento.renuncia_rrhh.registrar'$operaciones$;
BEGIN
    -- Catálogos: no exige USAGE de Bolsa ni lee sus tablas.
    IF EXISTS (SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
        WHERE n.nspname='vec_bolsa_llamamientos' AND p.proname='guardar_integracion_desarrollo_v1'
          AND strpos(pg_get_functiondef(p.oid),'bolsa.llamamiento.renuncia_rrhh.registrar')>0)
       OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.atestacion_decision_v3 a
          WHERE convert_from(a.capacidad_canonica,'UTF8')::jsonb->>'operacion'='bolsa.llamamiento.renuncia_rrhh.registrar') THEN
        RAISE EXCEPTION 'reversión denegada: consumidor o historia de renuncia' USING ERRCODE='55000';
    END IF;
    SELECT pg_get_functiondef(p.oid),p.proacl INTO STRICT v_def,v_acl FROM pg_proc p
     WHERE p.oid='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
       AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef;
    IF length(v_def)-length(replace(v_def,v_nuevo,''))<>length(v_nuevo) THEN
        RAISE EXCEPTION 'núcleo incompatible para retirar renuncia RRHH' USING ERRCODE='55000';
    END IF;
    EXECUTE replace(v_def,v_nuevo,v_anterior);
    IF (SELECT proacl FROM pg_proc WHERE oid='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure) IS DISTINCT FROM v_acl THEN
        RAISE EXCEPTION 'renuncia alteró permisos del núcleo' USING ERRCODE='55000';
    END IF;
END
$cambio$;
COMMIT;
