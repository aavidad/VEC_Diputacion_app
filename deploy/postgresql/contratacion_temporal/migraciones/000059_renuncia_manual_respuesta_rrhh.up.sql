\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:000059_renuncia_manual_respuesta_rrhh',0));
LOCK TABLE vec_contratacion_temporal.resolucion_manual_respuesta_rrhh IN ACCESS EXCLUSIVE MODE;
-- Extensión de CT58; se reutiliza el consumidor 17 y su material completo.
-- No reescribe filas aceptadas, cambia declaración CT56 ni invoca tablas Bolsa.
-- La intención pendiente pertenece al mismo commit, asiento y recibo locales.
ALTER TABLE vec_contratacion_temporal.resolucion_manual_respuesta_rrhh
    ADD COLUMN comando_siguiente_ref text,
    ADD COLUMN comando_siguiente_json jsonb,
    ADD CONSTRAINT resolucion_manual_comando_siguiente_unico UNIQUE (comando_siguiente_ref),
    ADD CONSTRAINT resolucion_manual_comando_siguiente_check CHECK ((
        (solicitud_json->>'Respuesta'='aceptacion'
            AND comando_siguiente_ref IS NULL AND comando_siguiente_json IS NULL)
        OR (solicitud_json->>'Respuesta'='renuncia'
            AND comando_siguiente_ref IS NOT NULL AND comando_siguiente_json IS NOT NULL
            AND comando_siguiente_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
            AND vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(comando_siguiente_json,ARRAY[
                'esquema','comando_ref','intencion_ref','organizacion_ref','expediente_ref',
                'llamamiento_ref','justificante_ref','seleccion_clave']) IS TRUE
            AND comando_siguiente_json->>'esquema'='vec.contratacion-temporal.siguiente-candidato.intencion.v1'
            AND comando_siguiente_json->>'comando_ref'=comando_siguiente_ref
            AND comando_siguiente_json->>'intencion_ref' ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
            AND comando_siguiente_json->>'organizacion_ref'=organizacion_ref
            AND comando_siguiente_json->>'expediente_ref'=expediente_ref
            AND comando_siguiente_json->>'llamamiento_ref'=llamamiento_ref
            AND comando_siguiente_json->>'justificante_ref'=justificante_ref
            AND comando_siguiente_json->>'seleccion_clave'=seleccion_clave::text
            AND recibo_json->'IntencionSiguiente'=jsonb_build_object(
                'Solicitud',solicitud_json,'ResolucionRef',resolucion_ref,'LlamamientoRef',llamamiento_ref,
                'ClaveIdempotencia',clave_idempotencia::text,'VersionEsperada',2,'VersionResultante',3,
                'IntencionRef',comando_siguiente_json->>'intencion_ref','ComandoOpacoRef',comando_siguiente_ref,
                'Estado','pendiente','ActualizadaEn',recibo_json->>'ResueltaEn'))
    ) IS TRUE);
COMMENT ON COLUMN vec_contratacion_temporal.resolucion_manual_respuesta_rrhh.comando_siguiente_json IS
    'Carga de intención CT pendiente y durable; sin candidato, orden, baremo, despacho ni efecto externo. Misma fila y recibo de renuncia. NULL para aceptaciones previas y nuevas.';
DO $cambio$
DECLARE v_def text; v_acl aclitem[]; v_cambio record;
BEGIN
    SELECT pg_get_functiondef(p.oid),p.proacl INTO STRICT v_def,v_acl FROM pg_proc p
     WHERE p.oid='vec_contratacion_temporal.registrar_resolucion_manual_respuesta_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
       AND p.proowner='vec_contratacion_temporal_propietario'::regrole AND p.prosecdef;
    FOR v_cambio IN SELECT * FROM (VALUES
      ('variables',$antes$    v_resultado jsonb;$antes$,$despues$    v_resultado jsonb;
    v_comando_ref text; v_comando_json jsonb; v_intencion_ref text; v_intencion_siguiente jsonb;$despues$),
      ('respuesta',$antes$       OR s->'Respuesta' IS DISTINCT FROM '"aceptacion"'::jsonb$antes$,$despues$       OR (s->'Respuesta') NOT IN ('"aceptacion"'::jsonb,'"renuncia"'::jsonb)$despues$),
      ('original',$antes$       AND r.respuesta='aceptacion' AND r.version_comunicacion=2$antes$,$despues$       AND r.respuesta=s->>'Respuesta' AND r.version_comunicacion=2$despues$),
      ('intencion',$antes$    v_resultado:=jsonb_build_object($antes$,$despues$    -- La intención CT queda registrada, no ejecutada ni enviada a Bolsa.
    -- Sus referencias resuelven a la carga de esta misma fila inmutable.
    v_intencion_siguiente:='{}'::jsonb;
    IF s->>'Respuesta'='renuncia' THEN
        v_comando_ref:='comando:'||gen_random_uuid()::text;
        v_intencion_ref:='intencion:'||gen_random_uuid()::text;
        v_comando_json:=jsonb_build_object(
            'esquema','vec.contratacion-temporal.siguiente-candidato.intencion.v1',
            'comando_ref',v_comando_ref,'intencion_ref',v_intencion_ref,
            'organizacion_ref',s->>'OrganizacionRef','expediente_ref',s->>'ExpedienteRef',
            'llamamiento_ref',s->>'LlamamientoRef','justificante_ref',v_declaracion.justificante_ref,
            'seleccion_clave',v_declaracion.seleccion_clave::text);
        v_intencion_siguiente:=jsonb_build_object(
            'Solicitud',s,'ResolucionRef',v_resolucion,'LlamamientoRef',s->>'LlamamientoRef',
            'ClaveIdempotencia',s->>'ClaveIdempotencia','VersionEsperada',2,'VersionResultante',3,
            'IntencionRef',v_intencion_ref,'ComandoOpacoRef',v_comando_ref,'Estado','pendiente',
            'ActualizadaEn',to_char(v_ahora,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'));
    END IF;
    v_resultado:=jsonb_build_object($despues$),
      ('recibo',$antes$        'IntencionSiguiente','{}'::jsonb,'VersionResultante',3,$antes$,$despues$        'IntencionSiguiente',v_intencion_siguiente,'VersionResultante',3,$despues$),
      ('columnas',$antes$        consumo_huella_sha256,evidencia_huella_sha256,recibo_ref,recibo_json,estado,resuelta_en$antes$,$despues$        consumo_huella_sha256,evidencia_huella_sha256,recibo_ref,recibo_json,estado,resuelta_en,
        comando_siguiente_ref,comando_siguiente_json$despues$),
      ('valores',$antes$        v_recibo,v_resultado,'confirmado',v_ahora);$antes$,$despues$        v_recibo,v_resultado,'confirmado',v_ahora,v_comando_ref,v_comando_json);$despues$)
    ) AS cambios(nombre,anterior,nuevo) LOOP
        IF length(v_def)-length(replace(v_def,v_cambio.anterior,''))<>length(v_cambio.anterior) THEN
            RAISE EXCEPTION 'resolución CT incompatible: %',v_cambio.nombre USING ERRCODE='55000';
        END IF;
        v_def:=replace(v_def,v_cambio.anterior,v_cambio.nuevo);
    END LOOP;
    EXECUTE v_def;
    IF (SELECT proacl FROM pg_proc WHERE oid='vec_contratacion_temporal.registrar_resolucion_manual_respuesta_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure) IS DISTINCT FROM v_acl THEN
        RAISE EXCEPTION 'renuncia alteró permisos CT' USING ERRCODE='55000';
    END IF;
END
$cambio$;
COMMIT;
