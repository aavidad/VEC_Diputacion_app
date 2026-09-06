\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000065',0));
-- Mismo registro CT61, permiso y transacción; selección raíz siempre íntegra.
-- Solo la confirmación CT60 permite acreditar la apertura del primer sucesor.
DO $propuesta_sucesor$
DECLARE
    v_antes record; v_despues record; v_fragmento record; v_definicion text;
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_proc p
         WHERE p.oid=to_regprocedure('vec_contratacion_temporal.registrar_resolucion_manual_respuesta_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')
           AND p.proowner='vec_contratacion_temporal_propietario'::regrole AND p.prosecdef
           AND encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')='3e27f8a9a1dfcec701db5721e7cc679955bbd6bdd921e3cc402e2110ce750274'
    ) OR NOT EXISTS (
        SELECT 1 FROM pg_index
         WHERE indexrelid=to_regclass('vec_contratacion_temporal.continuacion_recibo_unico')
           AND indisunique AND indisvalid
    ) THEN
        RAISE EXCEPTION 'se requiere resolución CT64 y continuación CT60 única' USING ERRCODE='55000';
    END IF;
    SELECT p.oid,p.prosrc AS cuerpo,pg_get_functiondef(p.oid) AS definicion,
           p.proacl AS acl,p.proowner AS propietario,p.proconfig AS configuracion
      INTO v_antes FROM pg_proc p
     WHERE p.oid=to_regprocedure('vec_contratacion_temporal.registrar_propuesta_formalizacion_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')
       AND p.proowner='vec_contratacion_temporal_propietario'::regrole AND p.prosecdef;
    IF NOT FOUND OR encode(sha256(convert_to(v_antes.cuerpo,'UTF8')),'hex')
       IS DISTINCT FROM 'db71857357fccb4e1be745b4fe7f7fb03b325837dc54b48741cc475c2ac143cb' THEN
        RAISE EXCEPTION 'función de propuesta incompatible' USING ERRCODE='55000';
    END IF;
    v_definicion:=v_antes.definicion;
    FOR v_fragmento IN SELECT anterior,nuevo FROM (VALUES
        (1,$anterior1$    v_respuesta jsonb; v_seleccion jsonb; v_resultado jsonb;$anterior1$,$nuevo1$    v_respuesta jsonb; v_seleccion jsonb; v_resultado jsonb; v_continuacion jsonb;$nuevo1$),
        (2,$anterior2$    SELECT j.recibo_json,e.recibo_json INTO v_respuesta,v_seleccion$anterior2$,$nuevo2$    SELECT j.recibo_json,e.recibo_json,continuacion.continuacion_recibo
      INTO v_respuesta,v_seleccion,v_continuacion$nuevo2$),
        (3,$anterior3$        ON e.clave_idempotencia=j.seleccion_clave AND e.clave_idempotencia=c.seleccion_clave$anterior3$,$nuevo3$        ON e.clave_idempotencia=j.seleccion_clave AND e.clave_idempotencia=c.seleccion_clave
      -- CT65: confirmación original del primer sucesor, ligada al aviso exacto.
      LEFT JOIN vec_contratacion_temporal.resolucion_manual_respuesta_rrhh continuacion
        ON c.material_json->'solicitud'->>'TipoAntecedente'='continuacion_confirmada'
       AND continuacion.seleccion_clave=e.clave_idempotencia
       AND continuacion.organizacion_ref=j.organizacion_ref
       AND continuacion.expediente_ref=j.expediente_ref
       AND continuacion.llamamiento_ref=e.recibo_json->>'llamamiento_ref'
       AND continuacion.llamamiento_ref<>j.llamamiento_ref
       AND continuacion.estado='confirmado'
       AND continuacion.solicitud_json->>'Respuesta'='renuncia'
       AND continuacion.continuacion_clave IS NOT NULL
       AND continuacion.continuacion_recibo->>'Estado'='confirmado'
       AND continuacion.continuacion_recibo->>'ReciboRef'=c.material_json->'solicitud'->>'PruebaEntregaRef'
       AND continuacion.continuacion_recibo->'Solicitud'->>'OrganizacionRef'=j.organizacion_ref
       AND continuacion.continuacion_recibo->'Solicitud'->>'ExpedienteRef'=j.expediente_ref
       AND continuacion.continuacion_recibo->'Solicitud'->>'ResolucionRef'=continuacion.resolucion_ref
       AND continuacion.continuacion_recibo->'Solicitud'->>'ClaveIdempotencia'=continuacion.continuacion_clave::text
       AND continuacion.continuacion_recibo->>'LlamamientoAnteriorRef'=continuacion.llamamiento_ref
       AND continuacion.continuacion_recibo->'ReciboBolsa'->>'LlamamientoRef'=j.llamamiento_ref$nuevo3$),
        (4,$anterior4$       AND e.recibo_json->>'llamamiento_ref'=j.llamamiento_ref AND e.recibo_json->'propuesta_generada'='true'::jsonb
       AND e.recibo_json->>'recibo_ref'=c.material_json->'solicitud'->>'PruebaEntregaRef'$anterior4$,$nuevo4$       AND (
           (NOT ((c.material_json->'solicitud') ? 'TipoAntecedente')
            AND e.recibo_json->>'llamamiento_ref'=j.llamamiento_ref
            AND e.recibo_json->>'recibo_ref'=c.material_json->'solicitud'->>'PruebaEntregaRef')
           OR (c.material_json->'solicitud'->>'TipoAntecedente'='continuacion_confirmada'
               AND continuacion.resolucion_ref IS NOT NULL)
       )
       AND e.recibo_json->'propuesta_generada'='true'::jsonb$nuevo4$),
        (5,$anterior5$            'Justificante',jsonb_build_object('Respuesta',v_respuesta,'Seleccion',v_seleccion),$anterior5$,$nuevo5$            'Justificante',jsonb_build_object('Respuesta',v_respuesta,'Seleccion',v_seleccion)||
                CASE WHEN v_continuacion IS NULL THEN '{}'::jsonb
                     ELSE jsonb_build_object('Continuacion',v_continuacion) END,$nuevo5$),
        (6,$anterior6$    IF b->>'AperturaOperacionRef' IS DISTINCT FROM v_seleccion->>'operacion_ref'$anterior6$,$nuevo6$    IF b->>'AperturaOperacionRef' IS DISTINCT FROM
           (CASE WHEN v_continuacion IS NULL THEN v_seleccion->>'operacion_ref'
                ELSE v_continuacion->'ReciboBolsa'->>'OperacionRef' END)$nuevo6$)
    ) AS fragmentos(orden,anterior,nuevo) ORDER BY orden
    LOOP
        IF length(v_definicion)-length(replace(v_definicion,v_fragmento.anterior,''))<>length(v_fragmento.anterior) THEN
            RAISE EXCEPTION 'fragmento de propuesta incompatible' USING ERRCODE='55000';
        END IF;
        v_definicion:=replace(v_definicion,v_fragmento.anterior,v_fragmento.nuevo);
    END LOOP;
    EXECUTE v_definicion;
    SELECT p.prosrc AS cuerpo,pg_get_functiondef(p.oid) AS definicion,p.proacl AS acl,
           p.proowner AS propietario,p.proconfig AS configuracion,p.prosecdef AS definidor
      INTO STRICT v_despues FROM pg_proc p WHERE p.oid=v_antes.oid;
    IF encode(sha256(convert_to(v_despues.cuerpo,'UTF8')),'hex')
       IS DISTINCT FROM 'f3e681486de901b464497af49737e72487329f5014322857c8845f5f782135bf'
       OR v_despues.definicion IS DISTINCT FROM v_definicion
       OR v_despues.acl IS DISTINCT FROM v_antes.acl
       OR v_despues.propietario IS DISTINCT FROM v_antes.propietario
       OR v_despues.configuracion IS DISTINCT FROM v_antes.configuracion
       OR v_despues.definidor IS NOT TRUE THEN
        RAISE EXCEPTION 'definición o permisos de propuesta alterados' USING ERRCODE='55000';
    END IF;
END
$propuesta_sucesor$;
COMMIT;
