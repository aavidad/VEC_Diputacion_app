\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000062',0));

-- Extiende la única operación CT54. El recibo CT60 acredita la continuación,
-- no entrega, plazo ni otra selección. No se consulta ni modifica Bolsa.
-- Los cinco fragmentos conservan el cuerpo original, ACL y política v1.
DO $comunicacion_sucesor$
DECLARE
    v_antes record; v_despues record; v_fragmento record;
    v_definicion text;
BEGIN
    SELECT p.oid,p.prosrc AS cuerpo,pg_get_functiondef(p.oid) AS definicion,
           p.proacl AS acl,p.proowner AS propietario,p.proconfig AS configuracion
      INTO v_antes FROM pg_proc p
     WHERE p.oid=to_regprocedure('vec_contratacion_temporal.registrar_comunicacion_llamamiento_local_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')
       AND p.proowner='vec_contratacion_temporal_propietario'::regrole AND p.prosecdef;
    IF NOT FOUND OR encode(sha256(convert_to(v_antes.cuerpo,'UTF8')),'hex')
       IS DISTINCT FROM 'aeebedee4d9bbb252a475fb7498ad4370d64c591939c3fb133f6e55fca262cde'
       OR NOT EXISTS (SELECT 1 FROM pg_attribute
          WHERE attrelid='vec_contratacion_temporal.resolucion_manual_respuesta_rrhh'::regclass
            AND attname='continuacion_recibo' AND atttypid='jsonb'::regtype AND NOT attisdropped) THEN
        RAISE EXCEPTION 'se requieren CT54 exacta y confirmación CT60' USING ERRCODE='55000';
    END IF;
    v_definicion:=v_antes.definicion;
    FOR v_fragmento IN SELECT * FROM (VALUES
        ($antes1$    s := m -> 'solicitud';$antes1$,
         $despues1$    s := m -> 'solicitud';
    -- CT62 tipo: intención no confiable, nunca autoridad del solicitante.
    IF s ? 'TipoAntecedente' THEN
        IF s->'TipoAntecedente' IS DISTINCT FROM '"continuacion_confirmada"'::jsonb
           OR m->'politica' IS DISTINCT FROM '{"Referencia":"politica:ct:desarrollo:registro-local:v2","Version":2,"HuellaSHA256":"24d08a85ca5e62b8b3832df49d208ccea098f2b202909639d900b9d42da0bacb"}'::jsonb THEN
            RAISE EXCEPTION 'tipo o política de continuación inválidos' USING ERRCODE='22023';
        END IF;
    ELSIF m->'politica'->>'Referencia'='politica:ct:desarrollo:registro-local:v2' THEN
        RAISE EXCEPTION 'política de continuación sin antecedente' USING ERRCODE='22023';
    END IF;
    -- CT62 fin tipo.$despues1$),
        ($antes2$ARRAY['ClaveIdempotencia','OrganizacionRef','ExpedienteRef','LlamamientoRef','VersionEsperada','PruebaEntregaRef'])$antes2$,
         $despues2$ARRAY['ClaveIdempotencia','OrganizacionRef','ExpedienteRef','LlamamientoRef','VersionEsperada','PruebaEntregaRef'] || CASE WHEN s ? 'TipoAntecedente' THEN ARRAY['TipoAntecedente'] ELSE ARRAY[]::text[] END)$despues2$),
        ($antes3$    -- Propiedad CT: se usa exclusivamente su recibo de selección confirmado.$antes3$,
         $despues3$    -- CT62 sucesor: solo después del consumo fresco y del replay exacto.
    IF s ? 'TipoAntecedente' THEN
        BEGIN
            SELECT e.* INTO STRICT v_seleccion
              FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 e
              JOIN vec_contratacion_temporal.resolucion_manual_respuesta_rrhh r
                ON r.seleccion_clave=e.clave_idempotencia
              JOIN vec_contratacion_temporal.comunicacion_llamamiento_local c
                ON c.comunicacion_ref=r.comunicacion_ref AND c.seleccion_clave=e.clave_idempotencia
               AND c.organizacion_ref=r.organizacion_ref AND c.expediente_ref=r.expediente_ref
               AND c.llamamiento_ref=r.llamamiento_ref
             WHERE r.organizacion_ref=s->>'OrganizacionRef' AND r.expediente_ref=s->>'ExpedienteRef'
               AND r.estado='confirmado' AND r.solicitud_json->>'Respuesta'='renuncia'
               AND r.continuacion_clave IS NOT NULL
               AND r.continuacion_recibo->>'Estado'='confirmado'
               AND r.continuacion_recibo->>'ReciboRef'=s->>'PruebaEntregaRef'
               AND r.continuacion_recibo->'Solicitud'->>'OrganizacionRef'=r.organizacion_ref
               AND r.continuacion_recibo->'Solicitud'->>'ExpedienteRef'=r.expediente_ref
               AND r.continuacion_recibo->'Solicitud'->>'ResolucionRef'=r.resolucion_ref
               AND r.continuacion_recibo->'Solicitud'->>'ClaveIdempotencia'=r.continuacion_clave::text
               AND r.continuacion_recibo->>'LlamamientoAnteriorRef'=r.llamamiento_ref
               AND r.continuacion_recibo->'ReciboBolsa'->>'LlamamientoRef'=s->>'LlamamientoRef'
               AND r.llamamiento_ref<>s->>'LlamamientoRef'
               AND e.situacion='confirmada' AND e.recibo_json->'propuesta_generada'='true'::jsonb
               AND e.solicitud_json->>'organizacion_ref'=r.organizacion_ref
               AND e.solicitud_json->>'expediente_ref'=r.expediente_ref
               AND e.recibo_json->>'llamamiento_ref'=r.llamamiento_ref
             FOR UPDATE OF e,r;
        EXCEPTION WHEN no_data_found OR too_many_rows THEN
            RAISE EXCEPTION 'continuación confirmada no disponible' USING ERRCODE='42501';
        END;
    ELSE
    -- CT62 fin prefijo sucesor.
    -- Propiedad CT: se usa exclusivamente su recibo de selección confirmado.$despues3$),
        ($antes4$    IF v_seleccion.recibo_json ->> 'organizacion_ref' IS DISTINCT FROM s ->> 'OrganizacionRef'$antes4$,
         $despues4$    END IF; -- CT62 fin rama antecedente.
    IF v_seleccion.recibo_json ->> 'organizacion_ref' IS DISTINCT FROM s ->> 'OrganizacionRef'$despues4$),
        ($antes5$'recibo_seleccion_ref', s ->> 'PruebaEntregaRef'$antes5$,
         $despues5$CASE WHEN s ? 'TipoAntecedente' THEN 'recibo_continuacion_ref' ELSE 'recibo_seleccion_ref' END, s ->> 'PruebaEntregaRef'$despues5$)
    ) AS fragmentos(anterior,nuevo)
    LOOP
        IF length(v_definicion)-length(replace(v_definicion,v_fragmento.anterior,''))<>length(v_fragmento.anterior)
           OR strpos(v_definicion,v_fragmento.nuevo)<>0 THEN
            RAISE EXCEPTION 'fragmento CT54 incompatible' USING ERRCODE='55000';
        END IF;
        v_definicion:=replace(v_definicion,v_fragmento.anterior,v_fragmento.nuevo);
    END LOOP;
    EXECUTE v_definicion;
    SELECT p.prosrc AS cuerpo,pg_get_functiondef(p.oid) AS definicion,p.proacl AS acl,
           p.proowner AS propietario,p.proconfig AS configuracion,p.prosecdef AS definidor
      INTO STRICT v_despues FROM pg_proc p WHERE p.oid=v_antes.oid;
    IF encode(sha256(convert_to(v_despues.cuerpo,'UTF8')),'hex')
       IS DISTINCT FROM 'fce60866f45b43caa8e51c8d6ae4064663577933cf1986268702f9354708ba7d'
       OR v_despues.definicion IS DISTINCT FROM v_definicion
       OR v_despues.acl IS DISTINCT FROM v_antes.acl
       OR v_despues.propietario IS DISTINCT FROM v_antes.propietario
       OR v_despues.configuracion IS DISTINCT FROM v_antes.configuracion
       OR v_despues.definidor IS NOT TRUE THEN
        RAISE EXCEPTION 'comunicación del sucesor alteró definición o permisos' USING ERRCODE='55000';
    END IF;
END
$comunicacion_sucesor$;
COMMIT;
