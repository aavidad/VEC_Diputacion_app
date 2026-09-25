\set ON_ERROR_STOP on
-- Reversión de CT111 solo sin historia propia: ningún evento de plazo ni
-- resolución que dependa de él. Nunca borra filas ni usa CASCADE.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000111',0));
LOCK TABLE vec_contratacion_temporal.resolucion_manual_respuesta_rrhh IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_contratacion_temporal.evento_plazo_llamamiento_rrhh IN ACCESS EXCLUSIVE MODE;
DO $conservar$
BEGIN
    -- CT119 continúa tras la expiración que añade CT111: se retira antes.
    IF to_regprocedure('vec_contratacion_temporal.continuar_llamamiento_rrhh_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL THEN
        RAISE EXCEPTION 'reversión denegada: CT119 instalada; retirar CT119 antes' USING ERRCODE='55000';
    END IF;
    IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.evento_plazo_llamamiento_rrhh)
       OR EXISTS (SELECT 1 FROM vec_contratacion_temporal.resolucion_manual_respuesta_rrhh
           WHERE solicitud_json->>'Respuesta'='expiracion_gobernada' OR justificante_ref IS NULL
              OR contacto_ref IS NOT NULL OR causa_justificada_ref IS NOT NULL OR respuesta_fuera_de_plazo
              OR estado_plazo<>'vigente'
              OR politica_ref<>'politica:ct:revision-manual-sintetica:20260906' OR politica_version<>1
              OR politica_sha256<>'ea41d65808044fa75b597855e81a469ed274403a521890bafa07c33ae89ec2e3') THEN
        RAISE EXCEPTION 'reversión denegada: se conserva la historia de plazo CT111' USING ERRCODE='55000';
    END IF;
END
$conservar$;
DO $resolucion$
DECLARE v_antes record; v_despues record; v_fragmento record; v_definicion text;
BEGIN
    SELECT p.oid,pg_get_functiondef(p.oid) AS definicion,p.proacl AS acl,
           p.proowner AS propietario,p.proconfig AS configuracion
      INTO STRICT v_antes FROM pg_proc p
     WHERE p.oid='vec_contratacion_temporal.registrar_resolucion_manual_respuesta_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
       AND p.proowner='vec_contratacion_temporal_propietario'::regrole AND p.prosecdef;
    v_definicion:=v_antes.definicion;
    FOR v_fragmento IN SELECT anterior,nuevo FROM (VALUES
    (1,$a1$    v_comando_ref text; v_comando_json jsonb; v_intencion_ref text; v_intencion_siguiente jsonb;$a1$,
       $n1$    v_comando_ref text; v_comando_json jsonb; v_intencion_ref text; v_intencion_siguiente jsonb;
    -- CT111: plazo evaluado con hechos durables, nunca con el material.
    v_tipo_politica text; v_contacto vec_contratacion_temporal.evento_plazo_llamamiento_rrhh%ROWTYPE;
    v_con_contacto boolean; v_seleccion_expiracion uuid; v_estado_plazo text;
    v_fuera_plazo boolean; v_causa_ref text;$n1$),
    (2,$a2$       OR (s->'Respuesta') NOT IN ('"aceptacion"'::jsonb,'"renuncia"'::jsonb)$a2$,
       $n2$       OR (s->'Respuesta') NOT IN ('"aceptacion"'::jsonb,'"renuncia"'::jsonb,'"expiracion_gobernada"'::jsonb)
       OR (s->>'Respuesta'='expiracion_gobernada' AND s->'PruebaRespuestaRef' IS DISTINCT FROM '""'::jsonb)$n2$),
    (3,$a3$        'OrganizacionRef','ExpedienteRef','LlamamientoRef','ComunicacionRef','PruebaRespuestaRef'
    ] LOOP$a3$,
       $n3$        'OrganizacionRef','ExpedienteRef','LlamamientoRef','ComunicacionRef'
    ] || CASE WHEN s->>'Respuesta'='expiracion_gobernada' THEN ARRAY[]::text[]
              ELSE ARRAY['PruebaRespuestaRef'] END LOOP$n3$),
    (4,$a4$    IF s->'CriterioValidacionRef' IS DISTINCT FROM '"politica:ct:revision-manual-sintetica:20260906"'::jsonb
       OR p->'Referencia' IS DISTINCT FROM s->'CriterioValidacionRef'
       OR jsonb_typeof(p->'Version') IS DISTINCT FROM 'number'
       OR p->>'Version' IS DISTINCT FROM '1'
       OR p->'HuellaSHA256' IS DISTINCT FROM '"ea41d65808044fa75b597855e81a469ed274403a521890bafa07c33ae89ec2e3"'::jsonb THEN
        RAISE EXCEPTION 'política de ejercicio sintético incompatible' USING ERRCODE='P0580';
    END IF;$a4$,
       $n4$    -- CT111: política admitida por la lista versionada, no por un literal.
    IF jsonb_typeof(s->'CriterioValidacionRef') IS DISTINCT FROM 'string'
       OR p->'Referencia' IS DISTINCT FROM s->'CriterioValidacionRef'
       OR jsonb_typeof(p->'Version') IS DISTINCT FROM 'number'
       OR (p->>'Version')!~'^[1-9][0-9]{0,15}$'
       OR jsonb_typeof(p->'HuellaSHA256') IS DISTINCT FROM 'string' THEN
        RAISE EXCEPTION 'política de resolución inválida' USING ERRCODE='P0580';
    END IF;
    v_tipo_politica:=vec_contratacion_temporal.politica_llamamiento_admitida_v1(
        p->>'Referencia',(p->>'Version')::numeric,p->>'HuellaSHA256',s->>'Respuesta');
    IF v_tipo_politica IS NULL THEN
        RAISE EXCEPTION 'política de resolución no admitida' USING ERRCODE='P0580';
    END IF;$n4$),
    (5,$a5$    -- Solo tablas propias CT. La declaración, el aviso local y la selección
    -- original deben conservar todas sus coordenadas y la propuesta confirmada.
    SELECT r.* INTO v_declaracion$a5$,
       $n5$    -- CT111: contacto efectivo de esta comunicación, si RRHH lo registró.
    SELECT * INTO v_contacto FROM vec_contratacion_temporal.evento_plazo_llamamiento_rrhh
     WHERE organizacion_ref=s->>'OrganizacionRef' AND comunicacion_ref=s->>'ComunicacionRef'
       AND expediente_ref=s->>'ExpedienteRef' AND llamamiento_ref=s->>'LlamamientoRef'
       AND tipo='contacto_efectivo'
     FOR SHARE;
    v_con_contacto:=FOUND;
    IF s->>'Respuesta'='expiracion_gobernada' THEN
        -- Sin respuesta personal: el antecedente es el aviso local con plazo abierto.
        IF NOT v_con_contacto THEN
            RAISE EXCEPTION 'expiración sin contacto efectivo' USING ERRCODE='P0582';
        END IF;
        SELECT c.seleccion_clave INTO v_seleccion_expiracion
          FROM vec_contratacion_temporal.comunicacion_llamamiento_local c
         WHERE c.comunicacion_ref=s->>'ComunicacionRef'
           AND c.organizacion_ref=s->>'OrganizacionRef'
           AND c.expediente_ref=s->>'ExpedienteRef'
           AND c.llamamiento_ref=s->>'LlamamientoRef'
           AND c.version_resultante=2 AND c.estado='registrada_localmente'
         FOR SHARE;
        IF NOT FOUND THEN
            RAISE EXCEPTION 'aviso incompatible con la expiración' USING ERRCODE='P0582';
        END IF;
        v_declaracion.justificante_ref:=NULL;
        v_declaracion.seleccion_clave:=v_seleccion_expiracion;
        v_declaracion.registrada_en:=v_contacto.registrado_en;
    ELSE
    -- Solo tablas propias CT. La declaración, el aviso local y la selección
    -- original deben conservar todas sus coordenadas y la propuesta confirmada.
    SELECT r.* INTO v_declaracion$n5$),
    (6,$a6$        RAISE EXCEPTION 'justificante o antecedentes incompatibles' USING ERRCODE='P0582';
    END IF;$a6$,
       $n6$        RAISE EXCEPTION 'justificante o antecedentes incompatibles' USING ERRCODE='P0582';
    END IF;
    END IF;$n6$),
    (7,$a7$        RAISE EXCEPTION 'reloj anterior al justificante registrado' USING ERRCODE='P0584';
    END IF;$a7$,
       $n7$        RAISE EXCEPTION 'reloj anterior al justificante registrado' USING ERRCODE='P0584';
    END IF;
    -- CT111: la regla aplicable es la capturada al abrir el plazo.
    v_estado_plazo:='vigente'; v_fuera_plazo:=false; v_causa_ref:=NULL;
    IF v_tipo_politica='entrada_catalogo' AND NOT v_con_contacto THEN
        RAISE EXCEPTION 'política de catálogo sin contacto efectivo' USING ERRCODE='P0582';
    END IF;
    IF v_con_contacto AND s->>'Respuesta'='expiracion_gobernada' THEN
        IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.respuesta_recibida_rrhh
            WHERE organizacion_ref=s->>'OrganizacionRef' AND comunicacion_ref=s->>'ComunicacionRef') THEN
            RAISE EXCEPTION 'respuesta registrada: no procede expiración' USING ERRCODE='P0582';
        END IF;
        IF v_ahora<v_contacto.respuesta_hasta THEN
            RAISE EXCEPTION 'plazo de respuesta no vencido' USING ERRCODE='P0586';
        END IF;
        IF v_contacto.confirmacion_expiracion IS DISTINCT FROM 'rrhh' THEN
            RAISE EXCEPTION 'confirmación de expiración no admitida' USING ERRCODE='P0580';
        END IF;
        v_estado_plazo:='expirado';
    ELSIF v_con_contacto AND v_declaracion.recibida_en>=v_contacto.respuesta_hasta THEN
        v_fuera_plazo:=true;
        IF v_contacto.tratamiento_fuera_plazo='exige_causa_justificada' THEN
            SELECT evento_ref INTO v_causa_ref FROM vec_contratacion_temporal.evento_plazo_llamamiento_rrhh
             WHERE organizacion_ref=s->>'OrganizacionRef' AND comunicacion_ref=s->>'ComunicacionRef'
               AND tipo='causa_justificada'
             FOR SHARE;
            IF NOT FOUND THEN
                RAISE EXCEPTION 'respuesta fuera de plazo sin causa justificada acreditada' USING ERRCODE='P0585';
            END IF;
        ELSIF v_contacto.tratamiento_fuera_plazo IS DISTINCT FROM 'admitir' THEN
            RAISE EXCEPTION 'respuesta fuera de plazo no admitida' USING ERRCODE='P0585';
        END IF;
    END IF;$n7$),
    (8,$a8$    IF s->>'Respuesta'='renuncia' THEN
        v_comando_ref:='comando:'||gen_random_uuid()::text;$a8$,
       $n8$    IF s->>'Respuesta' IN ('renuncia','expiracion_gobernada') THEN
        v_comando_ref:='comando:'||gen_random_uuid()::text;$n8$),
    (9,$a9$        'Solicitud',s,'Politica',p,'EvaluacionPlazoRef',v_evaluacion,'EstadoPlazo','vigente',$a9$,
       $n9$        'Solicitud',s,'Politica',p,'EvaluacionPlazoRef',v_evaluacion,'EstadoPlazo',v_estado_plazo,$n9$),
    (10,$a10$        'ResueltaEn',to_char(v_ahora,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'Estado','confirmado');
    INSERT INTO$a10$,
       $n10$        'ResueltaEn',to_char(v_ahora,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'Estado','confirmado');
    IF v_con_contacto THEN
        v_resultado:=v_resultado||jsonb_build_object(
            'RespuestaHasta',to_char(v_contacto.respuesta_hasta,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
            'RespuestaFueraDePlazo',v_fuera_plazo)
            ||CASE WHEN v_causa_ref IS NULL THEN '{}'::jsonb
                   ELSE jsonb_build_object('CausaJustificadaRef',v_causa_ref) END;
    END IF;
    INSERT INTO$n10$),
    (11,$a11$        comando_siguiente_ref,comando_siguiente_json
    ) VALUES ($a11$,
       $n11$        comando_siguiente_ref,comando_siguiente_json,
        contacto_ref,causa_justificada_ref,respuesta_fuera_de_plazo
    ) VALUES ($n11$),
    (12,$a12$        d->>'principal_id',d->>'perfil_activo_ref',true,true,p->>'Referencia',1,p->>'HuellaSHA256',
        v_evaluacion,'vigente',3,p_material,m,v_material_huella,s,v_consumo.auditoria_ref,$a12$,
       $n12$        d->>'principal_id',d->>'perfil_activo_ref',true,true,p->>'Referencia',(p->>'Version')::numeric,p->>'HuellaSHA256',
        v_evaluacion,v_estado_plazo,3,p_material,m,v_material_huella,s,v_consumo.auditoria_ref,$n12$),
    (13,$a13$        v_recibo,v_resultado,'confirmado',v_ahora,v_comando_ref,v_comando_json);$a13$,
       $n13$        v_recibo,v_resultado,'confirmado',v_ahora,v_comando_ref,v_comando_json,
        CASE WHEN v_con_contacto THEN v_contacto.evento_ref END,v_causa_ref,v_fuera_plazo);$n13$)
    ) AS fragmentos(orden,anterior,nuevo) ORDER BY orden DESC
    LOOP
        IF length(v_definicion)-length(replace(v_definicion,v_fragmento.nuevo,''))<>length(v_fragmento.nuevo) THEN
            RAISE EXCEPTION 'fragmento CT111 incompatible en reversión' USING ERRCODE='55000';
        END IF;
        v_definicion:=replace(v_definicion,v_fragmento.nuevo,v_fragmento.anterior);
    END LOOP;
    EXECUTE v_definicion;
    SELECT p.prosrc AS cuerpo,p.proacl AS acl,p.proowner AS propietario,
           p.proconfig AS configuracion,p.prosecdef AS definidor
      INTO STRICT v_despues FROM pg_proc p WHERE p.oid=v_antes.oid;
    IF encode(sha256(convert_to(v_despues.cuerpo,'UTF8')),'hex')
         IS DISTINCT FROM '3e27f8a9a1dfcec701db5721e7cc679955bbd6bdd921e3cc402e2110ce750274'
       OR v_despues.acl IS DISTINCT FROM v_antes.acl
       OR v_despues.propietario IS DISTINCT FROM v_antes.propietario
       OR v_despues.configuracion IS DISTINCT FROM v_antes.configuracion
       OR v_despues.definidor IS NOT TRUE THEN
        RAISE EXCEPTION 'reversión CT111 no recupera exactamente CT64' USING ERRCODE='55000';
    END IF;
END
$resolucion$;
ALTER TABLE vec_contratacion_temporal.resolucion_manual_respuesta_rrhh
    DROP CONSTRAINT resolucion_manual_comando_siguiente_check,
    DROP CONSTRAINT resolucion_manual_plazo_coherente_check,
    DROP CONSTRAINT resolucion_manual_estado_plazo_check,
    DROP CONSTRAINT resolucion_manual_politica_forma_check,
    DROP COLUMN respuesta_fuera_de_plazo,
    DROP COLUMN causa_justificada_ref,
    DROP COLUMN contacto_ref,
    ALTER COLUMN justificante_ref SET NOT NULL,
    ADD CONSTRAINT resolucion_manual_respuesta_rrhh_politica_ref_check
        CHECK (politica_ref='politica:ct:revision-manual-sintetica:20260906'),
    ADD CONSTRAINT resolucion_manual_respuesta_rrhh_politica_version_check
        CHECK (politica_version=1),
    ADD CONSTRAINT resolucion_manual_respuesta_rrhh_politica_sha256_check
        CHECK (politica_sha256='ea41d65808044fa75b597855e81a469ed274403a521890bafa07c33ae89ec2e3'),
    ADD CONSTRAINT resolucion_manual_respuesta_rrhh_estado_plazo_check
        CHECK (estado_plazo='vigente'),
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
DROP FUNCTION vec_contratacion_temporal.registrar_evento_plazo_llamamiento_v1(
    text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP TABLE vec_contratacion_temporal.evento_plazo_llamamiento_rrhh;
DROP FUNCTION vec_contratacion_temporal.politica_llamamiento_admitida_v1(text,numeric,text,text);
DROP TABLE vec_contratacion_temporal.politica_llamamiento_admitida;
COMMIT;
