\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000063',0));

-- Misma declaración CT56 de diez campos, sin resolver la respuesta ni leer
-- Bolsa. El tipo procede del aviso CT62 persistido, nunca del solicitante.
DO $respuesta_sucesor$
DECLARE
    v_antes record; v_despues record; v_fragmento record; v_definicion text;
BEGIN
    SELECT p.oid,p.prosrc AS cuerpo,pg_get_functiondef(p.oid) AS definicion,
           p.proacl AS acl,p.proowner AS propietario,p.proconfig AS configuracion
      INTO v_antes FROM pg_proc p
     WHERE p.oid=to_regprocedure('vec_contratacion_temporal.registrar_respuesta_recibida_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')
       AND p.proowner='vec_contratacion_temporal_propietario'::regrole AND p.prosecdef;
    IF NOT FOUND OR encode(sha256(convert_to(v_antes.cuerpo,'UTF8')),'hex')
       IS DISTINCT FROM '3fbbc01c66c85959cd922738a2fb72580c872221cd5387706ceea25d73da442d'
       OR NOT EXISTS (SELECT 1 FROM pg_proc p
          WHERE p.oid=to_regprocedure('vec_contratacion_temporal.registrar_comunicacion_llamamiento_local_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')
            AND p.proowner='vec_contratacion_temporal_propietario'::regrole AND p.prosecdef
            AND encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')='fce60866f45b43caa8e51c8d6ae4064663577933cf1986268702f9354708ba7d') THEN
        RAISE EXCEPTION 'se requieren CT56 exacta y aviso CT62' USING ERRCODE='55000';
    END IF;
    v_definicion:=v_antes.definicion;
    FOR v_fragmento IN SELECT * FROM (VALUES
        ($antes1$    IF v_seleccion.situacion IS DISTINCT FROM 'confirmada'$antes1$,
         $despues1$    -- CT63 antecedente: consumo fresco y comunicación ya comprobados.
    IF (v_comunicacion.material_json->'solicitud') ? 'TipoAntecedente' THEN
        IF v_comunicacion.material_json->'solicitud'->'TipoAntecedente'
           IS DISTINCT FROM '"continuacion_confirmada"'::jsonb
           OR NOT EXISTS (
            SELECT 1 FROM vec_contratacion_temporal.resolucion_manual_respuesta_rrhh r
             WHERE r.seleccion_clave=v_seleccion.clave_idempotencia
               AND r.organizacion_ref=s->>'OrganizacionRef' AND r.expediente_ref=s->>'ExpedienteRef'
               AND r.estado='confirmado' AND r.solicitud_json->>'Respuesta'='renuncia'
               AND r.continuacion_clave IS NOT NULL
               AND r.continuacion_recibo->>'Estado'='confirmado'
               AND r.continuacion_recibo->>'ReciboRef'=
                   v_comunicacion.material_json->'solicitud'->>'PruebaEntregaRef'
               AND r.continuacion_recibo->'Solicitud'->>'OrganizacionRef'=r.organizacion_ref
               AND r.continuacion_recibo->'Solicitud'->>'ExpedienteRef'=r.expediente_ref
               AND r.continuacion_recibo->'Solicitud'->>'ResolucionRef'=r.resolucion_ref
               AND r.continuacion_recibo->'Solicitud'->>'ClaveIdempotencia'=r.continuacion_clave::text
               AND r.continuacion_recibo->>'LlamamientoAnteriorRef'=r.llamamiento_ref
               AND r.continuacion_recibo->'ReciboBolsa'->>'LlamamientoRef'=s->>'LlamamientoRef'
               AND r.llamamiento_ref=v_seleccion.recibo_json->>'llamamiento_ref'
               AND r.llamamiento_ref<>s->>'LlamamientoRef'
           ) THEN
            RAISE EXCEPTION 'antecedente de respuesta del sucesor incompatible' USING ERRCODE='P0562';
        END IF;
    END IF;
    -- CT63 fin antecedente.
    IF v_seleccion.situacion IS DISTINCT FROM 'confirmada'$despues1$),
        ($antes2$       OR v_seleccion.recibo_json ->> 'llamamiento_ref' IS DISTINCT FROM s ->> 'LlamamientoRef'$antes2$,
         $despues2$       OR (v_comunicacion.material_json->'solicitud'->>'TipoAntecedente' IS DISTINCT FROM 'continuacion_confirmada'
           AND v_seleccion.recibo_json ->> 'llamamiento_ref' IS DISTINCT FROM s ->> 'LlamamientoRef')$despues2$),
        ($antes3$       OR v_seleccion.recibo_json ->> 'recibo_ref' IS DISTINCT FROM
           v_comunicacion.material_json -> 'solicitud' ->> 'PruebaEntregaRef'$antes3$,
         $despues3$       OR (v_comunicacion.material_json->'solicitud'->>'TipoAntecedente' IS DISTINCT FROM 'continuacion_confirmada'
           AND v_seleccion.recibo_json ->> 'recibo_ref' IS DISTINCT FROM
           v_comunicacion.material_json -> 'solicitud' ->> 'PruebaEntregaRef')$despues3$)
    ) AS fragmentos(anterior,nuevo)
    LOOP
        IF length(v_definicion)-length(replace(v_definicion,v_fragmento.anterior,''))<>length(v_fragmento.anterior)
           OR strpos(v_definicion,v_fragmento.nuevo)<>0 THEN
            RAISE EXCEPTION 'fragmento CT56 incompatible' USING ERRCODE='55000';
        END IF;
        v_definicion:=replace(v_definicion,v_fragmento.anterior,v_fragmento.nuevo);
    END LOOP;
    EXECUTE v_definicion;
    SELECT p.prosrc AS cuerpo,pg_get_functiondef(p.oid) AS definicion,p.proacl AS acl,
           p.proowner AS propietario,p.proconfig AS configuracion,p.prosecdef AS definidor
      INTO STRICT v_despues FROM pg_proc p WHERE p.oid=v_antes.oid;
    IF encode(sha256(convert_to(v_despues.cuerpo,'UTF8')),'hex')
       IS DISTINCT FROM 'ca76c1e1dc2a3746afd39e978e49d7a4f1c3787c59f6507f2c666b28e0023d79'
       OR v_despues.definicion IS DISTINCT FROM v_definicion
       OR v_despues.acl IS DISTINCT FROM v_antes.acl
       OR v_despues.propietario IS DISTINCT FROM v_antes.propietario
       OR v_despues.configuracion IS DISTINCT FROM v_antes.configuracion
       OR v_despues.definidor IS NOT TRUE THEN
        RAISE EXCEPTION 'respuesta del sucesor alteró definición o permisos' USING ERRCODE='55000';
    END IF;
END
$respuesta_sucesor$;
COMMIT;
