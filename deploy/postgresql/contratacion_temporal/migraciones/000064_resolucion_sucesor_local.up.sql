\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000064',0));
-- Misma consulta y resolución CT57/58: sin otro permiso, tabla ni efecto Bolsa.
-- La selección raíz no se transforma en el recibo de apertura de su sucesor.
DO $resolucion_sucesor$
DECLARE
    v_objeto record; v_antes record; v_despues record; v_fragmento record; v_definicion text;
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_proc p
         WHERE p.oid=to_regprocedure('vec_contratacion_temporal.registrar_respuesta_recibida_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')
           AND p.proowner='vec_contratacion_temporal_propietario'::regrole AND p.prosecdef
           AND encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')='ca76c1e1dc2a3746afd39e978e49d7a4f1c3787c59f6507f2c666b28e0023d79'
    ) OR NOT EXISTS (
        SELECT 1 FROM pg_index
         WHERE indexrelid=to_regclass('vec_contratacion_temporal.continuacion_recibo_unico')
           AND indisunique AND indisvalid
    ) THEN
        RAISE EXCEPTION 'se requieren CT63 y confirmación CT60 única' USING ERRCODE='55000';
    END IF;
    FOR v_objeto IN SELECT * FROM (VALUES
        ('consultar_justificante_respuesta_recibida_rrhh_v1',true,'637ade035c0d181b38681a26c51a8b49269e0fdbb35c083e9f8b47b17dc80b0b','64ab2899de68d5eefff5fc24dc71a81b0a60d61ccd16290aa3aa1f0baff438bd'),
        ('registrar_resolucion_manual_respuesta_rrhh_v1',false,'786607bb25d814189290295b704c682e085b67f93bd0af78e0a9dfc8144efd13','3e27f8a9a1dfcec701db5721e7cc679955bbd6bdd921e3cc402e2110ce750274')
    ) AS objetos(nombre,es_consulta,huella_antes,huella_despues)
    LOOP
        SELECT p.oid,p.prosrc AS cuerpo,pg_get_functiondef(p.oid) AS definicion,
               p.proacl AS acl,p.proowner AS propietario,p.proconfig AS configuracion
          INTO v_antes FROM pg_proc p
         WHERE p.oid=to_regprocedure('vec_contratacion_temporal.'||v_objeto.nombre||'(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')
           AND p.proowner='vec_contratacion_temporal_propietario'::regrole AND p.prosecdef;
        IF NOT FOUND OR encode(sha256(convert_to(v_antes.cuerpo,'UTF8')),'hex')
           IS DISTINCT FROM v_objeto.huella_antes THEN
            RAISE EXCEPTION 'función incompatible: %',v_objeto.nombre USING ERRCODE='55000';
        END IF;
        v_definicion:=v_antes.definicion;
        FOR v_fragmento IN SELECT anterior,nuevo FROM (VALUES
        (1,false,$anterior1$        ON e.clave_idempotencia=r.seleccion_clave AND e.clave_idempotencia=c.seleccion_clave$anterior1$,$nuevo1$        ON e.clave_idempotencia=r.seleccion_clave AND e.clave_idempotencia=c.seleccion_clave
      -- CT64: confirmación original del primer sucesor, ligada al aviso exacto.
      LEFT JOIN vec_contratacion_temporal.resolucion_manual_respuesta_rrhh continuacion
        ON c.material_json->'solicitud'->>'TipoAntecedente'='continuacion_confirmada'
       AND continuacion.seleccion_clave=e.clave_idempotencia
       AND continuacion.organizacion_ref=r.organizacion_ref
       AND continuacion.expediente_ref=r.expediente_ref
       AND continuacion.llamamiento_ref=e.recibo_json->>'llamamiento_ref'
       AND continuacion.llamamiento_ref<>r.llamamiento_ref
       AND continuacion.estado='confirmado'
       AND continuacion.solicitud_json->>'Respuesta'='renuncia'
       AND continuacion.continuacion_clave IS NOT NULL
       AND continuacion.continuacion_recibo->>'Estado'='confirmado'
       AND continuacion.continuacion_recibo->>'ReciboRef'=c.material_json->'solicitud'->>'PruebaEntregaRef'
       AND continuacion.continuacion_recibo->'Solicitud'->>'OrganizacionRef'=r.organizacion_ref
       AND continuacion.continuacion_recibo->'Solicitud'->>'ExpedienteRef'=r.expediente_ref
       AND continuacion.continuacion_recibo->'Solicitud'->>'ResolucionRef'=continuacion.resolucion_ref
       AND continuacion.continuacion_recibo->'Solicitud'->>'ClaveIdempotencia'=continuacion.continuacion_clave::text
       AND continuacion.continuacion_recibo->>'LlamamientoAnteriorRef'=continuacion.llamamiento_ref
       AND continuacion.continuacion_recibo->'ReciboBolsa'->>'LlamamientoRef'=r.llamamiento_ref$nuevo1$),
        (2,false,$anterior2$       AND e.recibo_json->>'llamamiento_ref'=r.llamamiento_ref$anterior2$,$nuevo2$       AND (
           (NOT ((c.material_json->'solicitud') ? 'TipoAntecedente')
            AND e.recibo_json->>'llamamiento_ref'=r.llamamiento_ref
            AND e.recibo_json->>'recibo_ref'=c.material_json->'solicitud'->>'PruebaEntregaRef')
           OR (c.material_json->'solicitud'->>'TipoAntecedente'='continuacion_confirmada'
               AND continuacion.resolucion_ref IS NOT NULL)
       )$nuevo2$),
        (3,false,$anterior3$       AND e.recibo_json->>'recibo_ref'=c.material_json->'solicitud'->>'PruebaEntregaRef'
$anterior3$,$nuevo3$       -- CT64: prueba del aviso cotejada en la alternativa anterior.
$nuevo3$),
        (4,true,$anterior4$    v_respuesta jsonb; v_seleccion jsonb;$anterior4$,$nuevo4$    v_respuesta jsonb; v_seleccion jsonb; v_continuacion jsonb;$nuevo4$),
        (5,true,$anterior5$    SELECT r.recibo_json,e.recibo_json INTO v_respuesta,v_seleccion$anterior5$,$nuevo5$    SELECT r.recibo_json,e.recibo_json,continuacion.continuacion_recibo
      INTO v_respuesta,v_seleccion,v_continuacion$nuevo5$),
        (6,true,$anterior6$    RETURN jsonb_build_object('Respuesta',v_respuesta,'Seleccion',v_seleccion);$anterior6$,$nuevo6$    -- Seleccion permanece íntegra. La confirmación CT60 no se renombra como raíz.
    IF v_continuacion IS NOT NULL THEN
        RETURN jsonb_build_object('Respuesta',v_respuesta,'Seleccion',v_seleccion,'Continuacion',v_continuacion);
    END IF;
    RETURN jsonb_build_object('Respuesta',v_respuesta,'Seleccion',v_seleccion);$nuevo6$)
        ) AS fragmentos(orden,solo_consulta,anterior,nuevo)
          WHERE NOT solo_consulta OR v_objeto.es_consulta
          ORDER BY orden
        LOOP
            IF length(v_definicion)-length(replace(v_definicion,v_fragmento.anterior,''))<>length(v_fragmento.anterior) THEN
                RAISE EXCEPTION 'fragmento incompatible: %',v_objeto.nombre USING ERRCODE='55000';
            END IF;
            v_definicion:=replace(v_definicion,v_fragmento.anterior,v_fragmento.nuevo);
        END LOOP;
        EXECUTE v_definicion;
        SELECT p.prosrc AS cuerpo,pg_get_functiondef(p.oid) AS definicion,p.proacl AS acl,
               p.proowner AS propietario,p.proconfig AS configuracion,p.prosecdef AS definidor
          INTO STRICT v_despues FROM pg_proc p WHERE p.oid=v_antes.oid;
        IF encode(sha256(convert_to(v_despues.cuerpo,'UTF8')),'hex') IS DISTINCT FROM v_objeto.huella_despues
           OR v_despues.definicion IS DISTINCT FROM v_definicion
           OR v_despues.acl IS DISTINCT FROM v_antes.acl
           OR v_despues.propietario IS DISTINCT FROM v_antes.propietario
           OR v_despues.configuracion IS DISTINCT FROM v_antes.configuracion
           OR v_despues.definidor IS NOT TRUE THEN
            RAISE EXCEPTION 'definición o permisos alterados: %',v_objeto.nombre USING ERRCODE='55000';
        END IF;
    END LOOP;
END
$resolucion_sucesor$;
COMMIT;
