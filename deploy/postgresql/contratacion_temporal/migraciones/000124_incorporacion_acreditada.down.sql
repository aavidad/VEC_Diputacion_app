\set ON_ERROR_STOP on
-- CT124 DOWN: solo sin historia. Si existe alguna confirmación de GINPIX o
-- del centro, o alguna no incorporación, la reversión se niega: la historia
-- es de solo adición.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000124',0));
LOCK TABLE vec_contratacion_temporal.expediente_version_integral IN SHARE ROW EXCLUSIVE MODE;
LOCK TABLE vec_contratacion_temporal.resolucion_manual_respuesta_rrhh IN SHARE ROW EXCLUSIVE MODE;

DO $pre$
DECLARE v_origen text;
BEGIN
    IF current_user<>'vec_contratacion_temporal_propietario'
       OR to_regclass('vec_contratacion_temporal.confirmacion_ginpix_v1') IS NULL
       OR to_regclass('vec_contratacion_temporal.incorporacion_centro_v1') IS NULL
       OR to_regclass('vec_contratacion_temporal.no_incorporacion_v1') IS NULL THEN
        RAISE EXCEPTION 'CT124 DOWN: CT124 no está instalada' USING ERRCODE='55000';
    END IF;
    IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.expediente_version_integral
                WHERE origen_version IN ('confirmacion_ginpix_ct124','no_incorporacion_ct124'))
       OR EXISTS (SELECT 1 FROM vec_contratacion_temporal.outbox_expediente_integral
                  WHERE tipo_evento IN ('ct.ginpix-confirmada.v1','ct.incorporacion-confirmada-centro.v1','ct.no-incorporacion.v1'))
       OR EXISTS (SELECT 1 FROM vec_contratacion_temporal.resolucion_manual_respuesta_rrhh
                  WHERE solicitud_json->>'Respuesta'='aceptacion' AND continuacion_clave IS NOT NULL) THEN
        RAISE EXCEPTION 'CT124 DOWN: no admitido con historia de confirmaciones' USING ERRCODE='55000';
    END IF;
    IF EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace
                WHERE n.nspname='vec_bolsa_llamamientos' AND c.relname='no_incorporacion_bolsa') THEN
        RAISE EXCEPTION 'CT124 DOWN: Bolsa 000042 depende de su comprobación de origen; retírela antes' USING ERRCODE='55000';
    END IF;
    SELECT pg_get_constraintdef(oid) INTO STRICT v_origen FROM pg_constraint
     WHERE conrelid='vec_contratacion_temporal.expediente_version_integral'::regclass
       AND conname='expediente_version_integral_origen_version_check';
    IF strpos(v_origen,', ''confirmacion_ginpix_ct124''::text, ''no_incorporacion_ct124''::text')=0 THEN
        RAISE EXCEPTION 'CT124 DOWN: origen de versión no localizado' USING ERRCODE='55000';
    END IF;
END
$pre$;

-- Retira la comprobación insertada en las dos funciones del cierre.
DO $cierre$
DECLARE
    v_funcion text; v_antes record; v_definicion text; v_inicio integer; v_fin integer;
    v_marca_inicio text:=E'\n    IF m->''condiciones'' @> ''["ginpix_confirmado"]''::jsonb AND NOT EXISTS (';
    v_marca_fin text:=E'RETURN jsonb_build_object(''esquema'',e,''resultado'',''ginpix_no_confirmado'');\n    END IF;';
BEGIN
    FOREACH v_funcion IN ARRAY ARRAY['preparar_cierre_expediente_v1(jsonb)',
        'confirmar_cierre_expediente_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'] LOOP
        SELECT p.oid,pg_get_functiondef(p.oid) AS definicion,p.proacl AS acl,p.proowner AS propietario,p.proconfig AS configuracion
          INTO STRICT v_antes FROM pg_proc p WHERE p.oid=to_regprocedure('vec_contratacion_temporal.'||v_funcion);
        v_definicion:=v_antes.definicion;
        v_inicio:=strpos(v_definicion,v_marca_inicio);
        v_fin:=strpos(v_definicion,v_marca_fin);
        IF v_inicio=0 OR v_fin<v_inicio
           OR length(v_definicion)-length(replace(v_definicion,v_marca_fin,''))<>length(v_marca_fin) THEN
            RAISE EXCEPTION 'CT124 DOWN: comprobación de cierre no localizada: %',v_funcion USING ERRCODE='55000';
        END IF;
        v_definicion:=left(v_definicion,v_inicio-1)||substr(v_definicion,v_fin+length(v_marca_fin));
        EXECUTE v_definicion;
        IF (SELECT proacl FROM pg_proc WHERE oid=v_antes.oid) IS DISTINCT FROM v_antes.acl
           OR (SELECT proowner FROM pg_proc WHERE oid=v_antes.oid) IS DISTINCT FROM v_antes.propietario
           OR (SELECT proconfig FROM pg_proc WHERE oid=v_antes.oid) IS DISTINCT FROM v_antes.configuracion
           OR strpos(pg_get_functiondef(v_antes.oid),'ginpix_confirmado_ct124')<>0 THEN
            RAISE EXCEPTION 'CT124 DOWN: cierre alterado: %',v_funcion USING ERRCODE='55000';
        END IF;
    END LOOP;
END
$cierre$;

-- Retira la ampliación de la continuación (CT119/CT121) y de su restricción.
DO $continuacion$
DECLARE
    v_antes record; v_definicion text; v_funcion text; v_alias text; i integer;
    v_firmas text[]; v_viejos text[]; v_nuevos text[];
BEGIN
    v_firmas:=ARRAY['continuar_llamamiento_rrhh_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
                    'continuar_llamamiento_rrhh_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'];
    v_viejos:=ARRAY[
$v1$       AND ((solicitud_json->>'Respuesta'='renuncia' AND estado_plazo='vigente' AND justificante_ref IS NOT NULL)
         OR (solicitud_json->>'Respuesta'='expiracion_gobernada' AND estado_plazo='expirado'
             AND justificante_ref IS NULL AND contacto_ref IS NOT NULL))
       AND revision_respuesta_rrhh AND revision_plazo_rrhh AND version_resultante=3
       AND comando_siguiente_ref IS NOT NULL
       AND comando_siguiente_json->>'intencion_ref'=s->>'IntencionRef'
       AND recibo_json->>'Estado'='confirmado'
       AND recibo_json->'IntencionSiguiente'->>'Estado'='pendiente'
       AND recibo_json->'Solicitud'=solicitud_json$v1$,
$v2$        v_resultado:=jsonb_build_object('Resolucion',v_fila.recibo_json,
            'ComandoSiguienteRef',v_fila.comando_siguiente_ref,'ComandoSiguiente',v_fila.comando_siguiente_json);$v2$];
    v_nuevos:=ARRAY[
$n1$       AND (((solicitud_json->>'Respuesta'='renuncia' AND estado_plazo='vigente' AND justificante_ref IS NOT NULL)
         OR (solicitud_json->>'Respuesta'='expiracion_gobernada' AND estado_plazo='expirado'
             AND justificante_ref IS NULL AND contacto_ref IS NOT NULL))
       AND revision_respuesta_rrhh AND revision_plazo_rrhh AND version_resultante=3
       AND comando_siguiente_ref IS NOT NULL
       AND comando_siguiente_json->>'intencion_ref'=s->>'IntencionRef'
       AND recibo_json->>'Estado'='confirmado'
       AND recibo_json->'IntencionSiguiente'->>'Estado'='pendiente'
       -- CT124: aceptación seguida de una no incorporación registrada.
       OR (solicitud_json->>'Respuesta'='aceptacion' AND estado_plazo='vigente' AND justificante_ref IS NOT NULL
           AND revision_respuesta_rrhh AND revision_plazo_rrhh AND version_resultante=3
           AND comando_siguiente_ref IS NULL AND recibo_json->>'Estado'='confirmado'
           AND vec_contratacion_temporal.antecedente_no_incorporacion_ct124(resolucion_ref,s->>'IntencionRef') IS NOT NULL))
       AND recibo_json->'Solicitud'=solicitud_json$n1$,
$n2$        v_resultado:=jsonb_build_object('Resolucion',v_fila.recibo_json,
            'ComandoSiguienteRef',v_fila.comando_siguiente_ref,'ComandoSiguiente',v_fila.comando_siguiente_json);
        IF v_fila.solicitud_json->>'Respuesta'='aceptacion' THEN
            v_resultado:=jsonb_build_object('Resolucion',v_fila.recibo_json)
                ||vec_contratacion_temporal.antecedente_no_incorporacion_ct124(v_fila.resolucion_ref,s->>'IntencionRef');
        END IF;$n2$];
    FOR v_funcion,v_alias IN SELECT * FROM (VALUES
        ('registrar_comunicacion_llamamiento_local_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','r'),
        ('registrar_respuesta_recibida_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','r'),
        ('consultar_justificante_respuesta_recibida_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','continuacion'),
        ('registrar_resolucion_manual_respuesta_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','continuacion'),
        ('leer_expediente_aviso_confirmado_v1(text,text,text)','r')) AS x(f,alias) LOOP
        v_firmas:=v_firmas||v_funcion;
        v_viejos:=v_viejos||(v_alias||$a$.solicitud_json->>'Respuesta' IN ('renuncia','expiracion_gobernada')$a$);
        v_nuevos:=v_nuevos||(v_alias||$a$.solicitud_json->>'Respuesta' IN ('renuncia','expiracion_gobernada','aceptacion')$a$);
    END LOOP;
    FOR i IN 1..array_length(v_firmas,1) LOOP
        SELECT p.oid,pg_get_functiondef(p.oid) AS definicion,p.proacl AS acl,p.proowner AS propietario,p.proconfig AS configuracion
          INTO v_antes FROM pg_proc p WHERE p.oid=to_regprocedure('vec_contratacion_temporal.'||v_firmas[i]);
        IF NOT FOUND OR length(v_antes.definicion)-length(replace(v_antes.definicion,v_nuevos[i],''))<>length(v_nuevos[i]) THEN
            RAISE EXCEPTION 'CT124 DOWN: continuación no localizada: %',v_firmas[i] USING ERRCODE='55000';
        END IF;
        v_definicion:=replace(v_antes.definicion,v_nuevos[i],v_viejos[i]);
        EXECUTE v_definicion;
        IF pg_get_functiondef(v_antes.oid) IS DISTINCT FROM v_definicion
           OR (SELECT proacl FROM pg_proc WHERE oid=v_antes.oid) IS DISTINCT FROM v_antes.acl
           OR (SELECT proowner FROM pg_proc WHERE oid=v_antes.oid) IS DISTINCT FROM v_antes.propietario
           OR (SELECT proconfig FROM pg_proc WHERE oid=v_antes.oid) IS DISTINCT FROM v_antes.configuracion THEN
            RAISE EXCEPTION 'CT124 DOWN: continuación alterada: %',v_firmas[i] USING ERRCODE='55000';
        END IF;
    END LOOP;
END
$continuacion$;

DO $restriccion$
DECLARE v_def text; v_nueva text;
    v_resp text:=$r$((solicitud_json ->> 'Respuesta'::text) = ANY (ARRAY['renuncia'::text, 'expiracion_gobernada'::text])) AND ((octet_length(continuacion_material)$r$;
    v_resp_n text:=$r$((solicitud_json ->> 'Respuesta'::text) = ANY (ARRAY['renuncia'::text, 'expiracion_gobernada'::text, 'aceptacion'::text])) AND ((octet_length(continuacion_material)$r$;
    v_int text:=$i$(((continuacion_recibo -> 'Solicitud'::text) ->> 'IntencionRef'::text) = (comando_siguiente_json ->> 'intencion_ref'::text))$i$;
    v_int_n text:=$i$((((continuacion_recibo -> 'Solicitud'::text) ->> 'IntencionRef'::text) = (comando_siguiente_json ->> 'intencion_ref'::text)) OR (((solicitud_json ->> 'Respuesta'::text) = 'aceptacion'::text) AND (comando_siguiente_json IS NULL)))$i$;
BEGIN
    SELECT pg_get_constraintdef(oid) INTO STRICT v_def FROM pg_constraint
     WHERE conrelid='vec_contratacion_temporal.resolucion_manual_respuesta_rrhh'::regclass AND conname='continuacion_confirmacion_completa';
    IF length(v_def)-length(replace(v_def,v_resp_n,''))<>length(v_resp_n)
       OR length(v_def)-length(replace(v_def,v_int_n,''))<>length(v_int_n) THEN
        RAISE EXCEPTION 'CT124 DOWN: restricción de continuación no localizada' USING ERRCODE='55000';
    END IF;
    v_nueva:=replace(replace(v_def,v_resp_n,v_resp),v_int_n,v_int);
    ALTER TABLE vec_contratacion_temporal.resolucion_manual_respuesta_rrhh DROP CONSTRAINT continuacion_confirmacion_completa;
    EXECUTE 'ALTER TABLE vec_contratacion_temporal.resolucion_manual_respuesta_rrhh ADD CONSTRAINT continuacion_confirmacion_completa '||v_nueva;
END
$restriccion$;

DROP FUNCTION vec_contratacion_temporal.antecedente_no_incorporacion_ct124(text,text);
DROP FUNCTION vec_contratacion_temporal.no_incorporacion_publicada_bolsa_v1(text,text,bigint);
REVOKE USAGE ON SCHEMA vec_contratacion_temporal FROM vec_bolsa_llamamientos_propietario;
DROP FUNCTION vec_contratacion_temporal.leer_no_incorporaciones_bolsa_v1(bigint,text,integer);
DROP FUNCTION vec_contratacion_temporal.evento_no_incorporacion_bolsa_ct124(vec_contratacion_temporal.no_incorporacion_v1);
DROP FUNCTION vec_contratacion_temporal.confirmar_no_incorporacion_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_contratacion_temporal.preparar_no_incorporacion_v1(jsonb);
DROP FUNCTION vec_contratacion_temporal.resultado_no_incorporacion_ct124(vec_contratacion_temporal.no_incorporacion_v1);
DROP FUNCTION vec_contratacion_temporal.intencion_no_incorporacion_ct124(text,text,text,text,text,text);
DROP FUNCTION vec_contratacion_temporal.aceptacion_vigente_ct124(text,text);
DROP FUNCTION vec_contratacion_temporal.validar_material_no_incorporacion_ct124(jsonb);
DROP TABLE vec_contratacion_temporal.no_incorporacion_v1;
DROP FUNCTION vec_contratacion_temporal.consultar_incorporacion_acreditada_v1(text,text);
DROP FUNCTION vec_contratacion_temporal.confirmar_incorporacion_centro_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_contratacion_temporal.consultar_incorporaciones_centro_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_contratacion_temporal.consumir_incorporacion_centro_ct124(text,text,jsonb,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_contratacion_temporal.expedientes_centro_ct124(jsonb,text);
DROP FUNCTION vec_contratacion_temporal.actor_centro_valido_ct124(jsonb);
DROP TABLE vec_contratacion_temporal.incorporacion_centro_acceso_v1;
DROP TABLE vec_contratacion_temporal.incorporacion_centro_v1;
DROP FUNCTION vec_contratacion_temporal.confirmar_confirmacion_ginpix_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_contratacion_temporal.preparar_confirmacion_ginpix_v1(jsonb);
DROP FUNCTION vec_contratacion_temporal.ginpix_confirmado_ct124(text,text);
DROP FUNCTION vec_contratacion_temporal.resultado_ginpix_ct124(vec_contratacion_temporal.confirmacion_ginpix_v1);
DROP FUNCTION vec_contratacion_temporal.validar_material_ginpix_ct124(jsonb);
DROP TABLE vec_contratacion_temporal.confirmacion_ginpix_v1;

DO $origen$
DECLARE v_origen text;
BEGIN
    SELECT pg_get_constraintdef(oid) INTO STRICT v_origen FROM pg_constraint
     WHERE conrelid='vec_contratacion_temporal.expediente_version_integral'::regclass
       AND conname='expediente_version_integral_origen_version_check';
    ALTER TABLE vec_contratacion_temporal.expediente_version_integral
        DROP CONSTRAINT expediente_version_integral_origen_version_check;
    EXECUTE 'ALTER TABLE vec_contratacion_temporal.expediente_version_integral ADD CONSTRAINT expediente_version_integral_origen_version_check '
        ||replace(v_origen,', ''confirmacion_ginpix_ct124''::text, ''no_incorporacion_ct124''::text','');
END
$origen$;
COMMIT;
