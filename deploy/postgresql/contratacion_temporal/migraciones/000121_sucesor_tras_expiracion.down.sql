\set ON_ERROR_STOP on
-- CT121 DOWN: solo sin historia. Si algún aviso del sucesor ya cuelga de la
-- continuación de una expiración, la reversión dejaría su circuito sin
-- antecedente admitido y se rechaza.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000121',0));

DO $sucesor_expiracion$
DECLARE
    v_funcion text; v_alias text; v_antes record; v_despues record; v_definicion text; v_anterior text; v_nuevo text;
BEGIN
    IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.comunicacion_llamamiento_local c
                 JOIN vec_contratacion_temporal.resolucion_manual_respuesta_rrhh r
                   ON r.continuacion_recibo->>'ReciboRef'=c.material_json->'solicitud'->>'PruebaEntregaRef'
                WHERE c.material_json->'solicitud'->>'TipoAntecedente'='continuacion_confirmada'
                  AND r.solicitud_json->>'Respuesta'='expiracion_gobernada') THEN
        RAISE EXCEPTION 'reversión denegada: hay avisos de sucesor tras una expiración' USING ERRCODE='55000';
    END IF;
    FOR v_funcion, v_alias IN SELECT * FROM (VALUES
        ('registrar_comunicacion_llamamiento_local_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','r'),
        ('registrar_respuesta_recibida_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','r'),
        ('consultar_justificante_respuesta_recibida_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','continuacion'),
        ('registrar_resolucion_manual_respuesta_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','continuacion'),
        ('registrar_propuesta_formalizacion_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','continuacion'),
        ('registrar_propuesta_formalizacion_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','continuacion'),
        ('leer_expediente_aviso_confirmado_v1(text,text,text)','r')
    ) AS f(firma,alias) LOOP
        v_anterior:=v_alias||$a$.solicitud_json->>'Respuesta'='renuncia'$a$;
        v_nuevo:=v_alias||$n$.solicitud_json->>'Respuesta' IN ('renuncia','expiracion_gobernada')$n$;
        SELECT p.oid,pg_get_functiondef(p.oid) AS definicion,p.proacl AS acl,p.proowner AS propietario,
               p.proconfig AS configuracion,p.prosecdef AS definidor
          INTO v_antes FROM pg_proc p WHERE p.oid=to_regprocedure('vec_contratacion_temporal.'||v_funcion);
        IF NOT FOUND OR length(v_antes.definicion)-length(replace(v_antes.definicion,v_nuevo,''))<>length(v_nuevo) THEN
            RAISE EXCEPTION 'CT121 no instalada: %',v_funcion USING ERRCODE='55000';
        END IF;
        v_definicion:=replace(v_antes.definicion,v_nuevo,v_anterior);
        EXECUTE v_definicion;
        SELECT pg_get_functiondef(p.oid) AS definicion,p.proacl AS acl,p.proowner AS propietario,
               p.proconfig AS configuracion,p.prosecdef AS definidor
          INTO STRICT v_despues FROM pg_proc p WHERE p.oid=v_antes.oid;
        IF v_despues.definicion IS DISTINCT FROM v_definicion
           OR v_despues.acl IS DISTINCT FROM v_antes.acl
           OR v_despues.propietario IS DISTINCT FROM v_antes.propietario
           OR v_despues.configuracion IS DISTINCT FROM v_antes.configuracion
           OR v_despues.definidor IS NOT TRUE THEN
            RAISE EXCEPTION 'CT121 DOWN: definición o permisos alterados: %',v_funcion USING ERRCODE='55000';
        END IF;
    END LOOP;
END
$sucesor_expiracion$;
COMMIT;
