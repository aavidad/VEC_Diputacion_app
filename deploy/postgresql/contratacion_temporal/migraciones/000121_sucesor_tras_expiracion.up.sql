\set ON_ERROR_STOP on
-- CT121: el circuito del sucesor también tras una expiración confirmada.
-- CT119 abre el siguiente llamamiento tras una expiración confirmada por RRHH
-- (CT111), pero el aviso, la respuesta, la consulta del justificante, la
-- resolución y la propuesta del sucesor (CT62-CT65) y sus variantes de
-- versión fiscalizada (CT95, CT96) solo reconocen como antecedente la
-- continuación de una renuncia. CT121 no edita esas migraciones: en cada una
-- de las siete funciones vivas sustituye la única condición
--   <alias>.solicitud_json->>'Respuesta'='renuncia'
-- del antecedente de continuación por
--   <alias>.solicitud_json->>'Respuesta' IN ('renuncia','expiracion_gobernada')
-- y conserva el resto del cuerpo, el propietario, la configuración y la ACL.
-- La continuación sigue exigiendo el recibo confirmado de CT60/CT119 ligado a
-- la misma resolución, el mismo llamamiento anterior y la apertura Bolsa.
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
    IF current_user<>'vec_contratacion_temporal_propietario'
       OR to_regprocedure('vec_contratacion_temporal.continuar_llamamiento_rrhh_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL THEN
        RAISE EXCEPTION 'CT121 requiere CT119' USING ERRCODE='55000';
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
          INTO v_antes FROM pg_proc p
         WHERE p.oid=to_regprocedure('vec_contratacion_temporal.'||v_funcion)
           AND p.proowner='vec_contratacion_temporal_propietario'::regrole AND p.prosecdef;
        IF NOT FOUND THEN
            RAISE EXCEPTION 'CT121: función ausente: %',v_funcion USING ERRCODE='55000';
        END IF;
        v_definicion:=v_antes.definicion;
        IF strpos(v_definicion,v_nuevo)<>0 THEN
            RAISE EXCEPTION 'CT121 ya instalada: no se reaplica' USING ERRCODE='55000';
        END IF;
        -- Exactamente una condición de renuncia en toda la función, y es la del
        -- antecedente de continuación.
        IF length(v_definicion)-length(replace(v_definicion,$c$solicitud_json->>'Respuesta'='renuncia'$c$,''))
              <>length($c$solicitud_json->>'Respuesta'='renuncia'$c$)
           OR length(v_definicion)-length(replace(v_definicion,v_anterior,''))<>length(v_anterior) THEN
            RAISE EXCEPTION 'CT121: preimagen incompatible: %',v_funcion USING ERRCODE='55000';
        END IF;
        v_definicion:=replace(v_definicion,v_anterior,v_nuevo);
        EXECUTE v_definicion;
        SELECT pg_get_functiondef(p.oid) AS definicion,p.proacl AS acl,p.proowner AS propietario,
               p.proconfig AS configuracion,p.prosecdef AS definidor
          INTO STRICT v_despues FROM pg_proc p WHERE p.oid=v_antes.oid;
        IF v_despues.definicion IS DISTINCT FROM v_definicion
           OR replace(v_despues.definicion,v_nuevo,v_anterior) IS DISTINCT FROM v_antes.definicion
           OR v_despues.acl IS DISTINCT FROM v_antes.acl
           OR v_despues.propietario IS DISTINCT FROM v_antes.propietario
           OR v_despues.configuracion IS DISTINCT FROM v_antes.configuracion
           OR v_despues.definidor IS NOT TRUE THEN
            RAISE EXCEPTION 'CT121: definición o permisos alterados: %',v_funcion USING ERRCODE='55000';
        END IF;
    END LOOP;
END
$sucesor_expiracion$;
COMMIT;
