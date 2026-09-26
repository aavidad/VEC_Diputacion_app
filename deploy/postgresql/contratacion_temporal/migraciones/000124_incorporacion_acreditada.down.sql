\set ON_ERROR_STOP on
-- CT124 DOWN: solo sin historia. Si existe alguna confirmación de GINPIX o
-- del centro, la reversión se niega: la historia es de solo adición.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000124',0));
LOCK TABLE vec_contratacion_temporal.expediente_version_integral IN SHARE ROW EXCLUSIVE MODE;

DO $pre$
DECLARE v_origen text;
BEGIN
    IF current_user<>'vec_contratacion_temporal_propietario'
       OR to_regclass('vec_contratacion_temporal.confirmacion_ginpix_v1') IS NULL
       OR to_regclass('vec_contratacion_temporal.incorporacion_centro_v1') IS NULL THEN
        RAISE EXCEPTION 'CT124 DOWN: CT124 no está instalada' USING ERRCODE='55000';
    END IF;
    IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.expediente_version_integral WHERE origen_version='confirmacion_ginpix_ct124')
       OR EXISTS (SELECT 1 FROM vec_contratacion_temporal.outbox_expediente_integral
                  WHERE tipo_evento IN ('ct.ginpix-confirmada.v1','ct.incorporacion-confirmada-centro.v1')) THEN
        RAISE EXCEPTION 'CT124 DOWN: no admitido con historia de confirmaciones' USING ERRCODE='55000';
    END IF;
    SELECT pg_get_constraintdef(oid) INTO STRICT v_origen FROM pg_constraint
     WHERE conrelid='vec_contratacion_temporal.expediente_version_integral'::regclass
       AND conname='expediente_version_integral_origen_version_check';
    IF strpos(v_origen,', ''confirmacion_ginpix_ct124''::text')=0 THEN
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
        ||replace(v_origen,', ''confirmacion_ginpix_ct124''::text','');
END
$origen$;
COMMIT;
