-- Borrador CT000092: reserva confirmada por integrador; NO aplicado.
-- Dependencias: CT86 (CHECK exacto), CT52, consumidor AD3 nominal de subsanación.
-- Incluye preparación read-only, helper de recuperación, confirmación y ACL conjunta.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal.dependencias.subsanacion_reparos.v1',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000092',0));

DO $prevalidacion$
DECLARE v_origen text;
BEGIN
    IF pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_subsanacion_reparos_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
       OR NOT pg_catalog.has_function_privilege(current_user,
           'vec_autorizacion_atestada_v3.registrar_y_consumir_subsanacion_reparos_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)', 'EXECUTE')
       OR pg_catalog.to_regclass('vec_contratacion_temporal.reserva_subsanacion_reparos') IS NOT NULL THEN
        RAISE EXCEPTION 'dependencias de subsanación incompatibles' USING ERRCODE='55000';
    END IF;
    SELECT pg_catalog.pg_get_constraintdef(oid) INTO STRICT v_origen
      FROM pg_catalog.pg_constraint
     WHERE conrelid='vec_contratacion_temporal.expediente_version_integral'::regclass
       AND conname='expediente_version_integral_origen_version_check'
       AND contype='c' AND convalidated;
    IF v_origen IS DISTINCT FROM $preimagen$CHECK ((origen_version = ANY (ARRAY['alta_o2'::text, 'analisis_o3'::text, 'cobertura_o4'::text, 'asignacion_o5'::text, 'informe_juridico_o5'::text, 'fiscalizacion_o5'::text, 'propuesta_formalizacion_o6'::text, 'resolucion_formalizacion_o6'::text, 'anotacion_administrativa_ct86'::text])))$preimagen$ THEN
        RAISE EXCEPTION 'preimagen de origen subsanación incompatible' USING ERRCODE='55000';
    END IF;
END
$prevalidacion$;

ALTER TABLE vec_contratacion_temporal.expediente_version_integral
    DROP CONSTRAINT expediente_version_integral_origen_version_check;
ALTER TABLE vec_contratacion_temporal.expediente_version_integral
    ADD CONSTRAINT expediente_version_integral_origen_version_check CHECK (origen_version IN (
        'alta_o2','analisis_o3','cobertura_o4','asignacion_o5','informe_juridico_o5',
        'fiscalizacion_o5','propuesta_formalizacion_o6','resolucion_formalizacion_o6',
        'anotacion_administrativa_ct86','subsanacion_reparos_v1'));

-- La preparación no escribe. Esta fila nace terminal dentro de la transacción
-- del efecto y conserva la respuesta para recuperación, incluso tras reinicio.
CREATE TABLE vec_contratacion_temporal.reserva_subsanacion_reparos (
    ambito_hmac text PRIMARY KEY,
    huella_peticion_hmac text NOT NULL,
    organizacion_ref text NOT NULL,
    expediente_ref text NOT NULL,
    version_esperada numeric(20,0) NOT NULL CHECK (version_esperada BETWEEN 1 AND 9007199254740990),
    actor_ref text NOT NULL,
    perfil_ref text NOT NULL,
    observaciones text NOT NULL CHECK (char_length(observaciones) BETWEEN 1 AND 2000 AND octet_length(observaciones)<=8192),
    estado text NOT NULL CHECK (estado='confirmada'),
    reserva_ref text NOT NULL UNIQUE,
    recibo_ref text NOT NULL UNIQUE,
    evento_ref text NOT NULL UNIQUE,
    retorno_ref text NOT NULL UNIQUE REFERENCES vec_contratacion_temporal.retorno_fiscalizacion_unidad(retorno_ref),
    expediente_anterior_json jsonb NOT NULL CHECK (jsonb_typeof(expediente_anterior_json)='object'),
    expediente_siguiente_json jsonb NOT NULL CHECK (jsonb_typeof(expediente_siguiente_json)='object'),
    recibo_json jsonb NOT NULL CHECK (jsonb_typeof(recibo_json)='object'),
    decision_ref text NOT NULL UNIQUE,
    decision_huella_sha256 text NOT NULL CHECK (decision_huella_sha256 ~ '^[0-9a-f]{64}$'),
    consumo_huella_sha256 text NOT NULL UNIQUE CHECK (consumo_huella_sha256 ~ '^[0-9a-f]{64}$'),
    auditoria_ref text NOT NULL UNIQUE CHECK (auditoria_ref ~ '^aud_v3_[0-9a-f]{32}$'),
    politica_ref text NOT NULL,
    politica_version numeric(20,0) NOT NULL CHECK (politica_version BETWEEN 1 AND 9007199254740991),
    politica_huella_sha256 text NOT NULL CHECK (politica_huella_sha256 ~ '^[0-9a-f]{64}$'),
    registrada_en timestamptz(6) NOT NULL,
    confirmada_en timestamptz(6) NOT NULL CHECK (confirmada_en>=registrada_en),
    FOREIGN KEY (expediente_ref,version_esperada) REFERENCES vec_contratacion_temporal.expediente_version_integral,
    CHECK (ambito_hmac ~ '^hmac-sha256:vec[.]contratacion-temporal[.]subsanacion-reparos[.]ambito/v[1-9][0-9]{0,8}:[0-9a-f]{64}$'),
    CHECK (huella_peticion_hmac ~ '^hmac-sha256:vec[.]contratacion-temporal[.]subsanacion-reparos[.]peticion/v[1-9][0-9]{0,8}:[0-9a-f]{64}$')
);
ALTER TABLE vec_contratacion_temporal.reserva_subsanacion_reparos ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contratacion_temporal.reserva_subsanacion_reparos FORCE ROW LEVEL SECURITY;
CREATE POLICY lectura_subsanacion ON vec_contratacion_temporal.reserva_subsanacion_reparos
    FOR SELECT TO vec_contratacion_temporal_propietario USING (
        organizacion_ref=current_setting('vec.ct_subsanacion.organizacion_ref',true)
        AND actor_ref=current_setting('vec.ct_subsanacion.actor_ref',true)
        AND perfil_ref=current_setting('vec.ct_subsanacion.perfil_ref',true)
        AND ambito_hmac=current_setting('vec.ct_subsanacion.ambito_hmac',true)
        AND pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
        AND NOT pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
        AND NOT pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER'));
CREATE POLICY escritura_subsanacion ON vec_contratacion_temporal.reserva_subsanacion_reparos
    FOR INSERT TO vec_contratacion_temporal_propietario WITH CHECK (
        organizacion_ref=current_setting('vec.ct_subsanacion.organizacion_ref',true)
        AND expediente_ref=current_setting('vec.ct_subsanacion.expediente_ref',true)
        AND actor_ref=current_setting('vec.ct_subsanacion.actor_ref',true)
        AND perfil_ref=current_setting('vec.ct_subsanacion.perfil_ref',true)
        AND ambito_hmac=current_setting('vec.ct_subsanacion.ambito_hmac',true)
        AND pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
        AND NOT pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
        AND NOT pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER'));
CREATE TRIGGER subsanacion_reparos_inmutable BEFORE UPDATE OR DELETE
    ON vec_contratacion_temporal.reserva_subsanacion_reparos FOR EACH ROW
    EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();

CREATE FUNCTION vec_contratacion_temporal.resultado_preparar_subsanacion_reparos_v1(p_ambito text)
RETURNS jsonb LANGUAGE sql STABLE SECURITY DEFINER
SET search_path=pg_catalog SET row_security='on' SET timezone='UTC'
AS $funcion$
    SELECT jsonb_build_object(
        'esquema','vec.contratacion-temporal.resultado-subsanacion-reparos.v1',
        'resultado','confirmada',
        'material',jsonb_build_object('organizacion_ref',r.organizacion_ref,
            'expediente_ref',r.expediente_ref,'version_esperada',r.version_esperada,
            'actor_ref',r.actor_ref,'perfil_ref',r.perfil_ref,'observaciones',r.observaciones),
        'expediente',r.expediente_siguiente_json,
        'referencias',jsonb_build_object('reserva_ref',r.reserva_ref,'recibo_ref',r.recibo_ref,'evento_ref',r.evento_ref),
        'retorno_ref',r.retorno_ref,'ambito_idempotencia_hmac',r.ambito_hmac,
        'huella_peticion_hmac',r.huella_peticion_hmac,'recibo',r.recibo_json)
      FROM vec_contratacion_temporal.reserva_subsanacion_reparos r WHERE r.ambito_hmac=p_ambito;
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.confirmar_subsanacion_reparos_v1(
    p_operacion jsonb,
    p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,
    p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security='on' SET timezone='UTC' SET lock_timeout='2s'
AS $funcion$
DECLARE
    m jsonb;
    a jsonb;
    pol jsonb;
    refs jsonb;
    r vec_contratacion_temporal.reserva_subsanacion_reparos%ROWTYPE;
    v_actual record;
    v_retorno record;
    v_consumo record;
    v_decision jsonb;
    v_version numeric;
    v_secuencia_actuacion numeric;
    v_instante timestamptz;
    v_ahora timestamptz(6);
    v_actuacion jsonb;
    v_siguiente jsonb;
    v_contexto_canonico bytea;
    v_contexto_huella text;
    v_observaciones_huella text;
    v_agregado_huella text;
    v_prueba bytea;
    v_payload_evento bytea;
    v_anterior text;
    v_secuencia numeric;
    v_recibo jsonb;
    v_ref text;
    v_restriccion text;
    v_tabla_error text;
    v_esquema_error text;
BEGIN
    IF current_user<>'vec_contratacion_temporal_propietario'
       OR session_user=current_user
       OR NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
       OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
       OR pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
       OR current_setting('transaction_isolation')<>'serializable'
       OR current_setting('transaction_read_only')<>'off' THEN
        RAISE EXCEPTION 'confirmación de subsanación denegada' USING ERRCODE='42501';
    END IF;
    IF p_operacion IS NULL OR pg_column_size(p_operacion)>3145728
       OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(p_operacion,
          ARRAY['esquema','operacion','material','referencias','ambito_idempotencia_hmac',
                'huella_peticion_hmac','retorno_ref','expediente_anterior','expediente_siguiente',
                'actuacion','politica','autorizacion','instante_efecto']) IS NOT TRUE THEN
        RAISE EXCEPTION 'entrada de subsanación inválida' USING ERRCODE='22023';
    END IF;
    m:=p_operacion->'material'; a:=p_operacion->'autorizacion';
    pol:=p_operacion->'politica'; refs:=p_operacion->'referencias';
    IF vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(m,
           ARRAY['organizacion_ref','expediente_ref','version_esperada','actor_ref','perfil_ref','observaciones']) IS NOT TRUE
       OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(refs,
           ARRAY['reserva_ref','recibo_ref','evento_ref']) IS NOT TRUE
       OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(pol,
           ARRAY['definicion_ref','definicion_version','definicion_huella_sha256','accion','finalidad','evaluada_en','valida_hasta']) IS NOT TRUE
       OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(a,
           ARRAY['accion','contexto_recurso_huella_sha256','decision_canonica_hex',
                 'decision_huella_sha256','decision_ref','finalidad','motivo_canonico_hex',
                 'perfil_activo_ref','perfil_version','persona_version','principal_id','recurso_ref']) IS NOT TRUE THEN
        RAISE EXCEPTION 'estructura de subsanación inválida' USING ERRCODE='22023';
    END IF;
    IF EXISTS (SELECT 1 FROM jsonb_each(p_operacion) c WHERE c.value='null'::jsonb)
       OR EXISTS (SELECT 1 FROM jsonb_each(m) c WHERE jsonb_typeof(c.value) IS DISTINCT FROM CASE WHEN c.key='version_esperada' THEN 'number' ELSE 'string' END)
       OR EXISTS (SELECT 1 FROM jsonb_each(refs) c WHERE jsonb_typeof(c.value) IS DISTINCT FROM 'string')
       OR EXISTS (SELECT 1 FROM jsonb_each(pol) c WHERE jsonb_typeof(c.value) IS DISTINCT FROM CASE WHEN c.key='definicion_version' THEN 'number' ELSE 'string' END)
       OR EXISTS (SELECT 1 FROM jsonb_each(a) c WHERE jsonb_typeof(c.value) IS DISTINCT FROM CASE WHEN c.key IN ('persona_version','perfil_version') THEN 'number' ELSE 'string' END)
       OR EXISTS (SELECT 1 FROM jsonb_each(p_operacion) c WHERE jsonb_typeof(c.value) IS DISTINCT FROM CASE WHEN c.key IN ('material','referencias','expediente_anterior','expediente_siguiente','actuacion','politica','autorizacion') THEN 'object' ELSE 'string' END)
       OR p_operacion->>'esquema'<>'vec.contratacion-temporal.confirmar-subsanacion-reparos.v1'
       OR p_operacion->>'operacion'<>'registrar_subsanacion'
       OR m->>'version_esperada' !~ '^[1-9][0-9]{0,15}$'
       OR (m->>'version_esperada')::numeric>9007199254740990
       OR pol->>'definicion_version' !~ '^[1-9][0-9]{0,15}$'
       OR (pol->>'definicion_version')::numeric>9007199254740991
       OR char_length(m->>'observaciones') NOT BETWEEN 1 AND 2000
       OR octet_length(m->>'observaciones')>8192
       OR m->>'observaciones'<>btrim(m->>'observaciones',E' \t\n\r\f'||chr(11)||chr(133)||chr(160)||chr(5760)||chr(8192)||chr(8193)||chr(8194)||chr(8195)||chr(8196)||chr(8197)||chr(8198)||chr(8199)||chr(8200)||chr(8201)||chr(8202)||chr(8232)||chr(8233)||chr(8239)||chr(8287)||chr(12288))
       OR m->>'observaciones'<>normalize(m->>'observaciones',NFC)
       OR translate(m->>'observaciones',E'\t\n','') ~ '[[:cntrl:]]'
       OR p_operacion->>'ambito_idempotencia_hmac' !~ '^hmac-sha256:vec[.]contratacion-temporal[.]subsanacion-reparos[.]ambito/v[1-9][0-9]{0,8}:[0-9a-f]{64}$'
       OR p_operacion->>'huella_peticion_hmac' !~ '^hmac-sha256:vec[.]contratacion-temporal[.]subsanacion-reparos[.]peticion/v[1-9][0-9]{0,8}:[0-9a-f]{64}$'
       OR pol->>'definicion_huella_sha256' !~ '^[0-9a-f]{64}$'
       OR pol->>'accion'<>'contratacion_temporal.subsanacion_reparos.registrar'
       OR pol->>'finalidad'<>'gestionar_contratacion_temporal' THEN
        RAISE EXCEPTION 'material de subsanación inválido' USING ERRCODE='22023';
    END IF;
    FOREACH v_ref IN ARRAY ARRAY[m->>'organizacion_ref',m->>'expediente_ref',m->>'actor_ref',m->>'perfil_ref',
        refs->>'reserva_ref',refs->>'recibo_ref',refs->>'evento_ref',p_operacion->>'retorno_ref',pol->>'definicion_ref'] LOOP
        IF v_ref IS NULL OR v_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' THEN
            RAISE EXCEPTION 'referencia de subsanación inválida' USING ERRCODE='22023';
        END IF;
    END LOOP;
    v_version:=(m->>'version_esperada')::numeric;
    IF p_operacion->>'instante_efecto' !~ '^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}([.][0-9]{1,6})?Z$'
       OR pol->>'evaluada_en' !~ '^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}([.][0-9]{1,6})?Z$'
       OR pol->>'valida_hasta' !~ '^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}([.][0-9]{1,6})?Z$' THEN
        RAISE EXCEPTION 'instante de subsanación inválido' USING ERRCODE='22023';
    END IF;
    v_instante:=(p_operacion->>'instante_efecto')::timestamptz;
    IF p_capacidad IS NULL OR p_decision IS NULL OR p_motivo IS NULL OR p_contexto IS NULL
       OR p_persona_version IS NULL OR p_perfil_version IS NULL OR p_payload IS NULL
       OR p_sobre IS NULL OR p_evidencia IS NULL OR p_raiz IS NULL
       OR octet_length(p_decision) NOT BETWEEN 1 AND 524288
       OR octet_length(p_motivo) NOT BETWEEN 1 AND 65536
       OR a->>'accion' IS DISTINCT FROM pol->>'accion'
       OR a->>'finalidad' IS DISTINCT FROM pol->>'finalidad'
       OR a->>'recurso_ref' IS DISTINCT FROM m->>'expediente_ref'
       OR a->>'principal_id' IS DISTINCT FROM m->>'actor_ref'
       OR a->>'perfil_activo_ref' IS DISTINCT FROM m->>'perfil_ref'
       OR a->>'decision_canonica_hex' IS DISTINCT FROM encode(p_decision,'hex')
       OR a->>'motivo_canonico_hex' IS DISTINCT FROM encode(p_motivo,'hex')
       OR (a->>'persona_version')::numeric IS DISTINCT FROM p_persona_version
       OR (a->>'perfil_version')::numeric IS DISTINCT FROM p_perfil_version
       OR a->>'decision_huella_sha256' IS DISTINCT FROM encode(sha256(p_decision),'hex') THEN
        RAISE EXCEPTION 'autorización de subsanación divergente' USING ERRCODE='42501';
    END IF;
    v_decision:=convert_from(p_decision,'UTF8')::jsonb;
    PERFORM set_config('vec.ct_subsanacion.organizacion_ref',m->>'organizacion_ref',true);
    PERFORM set_config('vec.ct_subsanacion.expediente_ref',m->>'expediente_ref',true);
    PERFORM set_config('vec.ct_subsanacion.actor_ref',m->>'actor_ref',true);
    PERFORM set_config('vec.ct_subsanacion.perfil_ref',m->>'perfil_ref',true);
    PERFORM set_config('vec.ct_subsanacion.ambito_hmac',p_operacion->>'ambito_idempotencia_hmac',true);
    SELECT * INTO r FROM vec_contratacion_temporal.reserva_subsanacion_reparos
     WHERE ambito_hmac=p_operacion->>'ambito_idempotencia_hmac';
    IF FOUND THEN
        IF r.huella_peticion_hmac IS DISTINCT FROM p_operacion->>'huella_peticion_hmac'
           OR r.organizacion_ref IS DISTINCT FROM m->>'organizacion_ref'
           OR r.expediente_ref IS DISTINCT FROM m->>'expediente_ref'
           OR r.version_esperada IS DISTINCT FROM v_version
           OR r.actor_ref IS DISTINCT FROM m->>'actor_ref'
           OR r.perfil_ref IS DISTINCT FROM m->>'perfil_ref'
           OR r.observaciones IS DISTINCT FROM m->>'observaciones'
           OR r.retorno_ref IS DISTINCT FROM p_operacion->>'retorno_ref' THEN
            RETURN jsonb_build_object('esquema','vec.contratacion-temporal.resultado-subsanacion-reparos.v1','resultado','idempotencia_reutilizada');
        END IF;
        -- Dos preparaciones pueden proponer referencias distintas: recuperar por
        -- PREPARAR la ganadora, nunca devolver a la otra un recibo que no valida.
        IF r.reserva_ref IS DISTINCT FROM refs->>'reserva_ref'
           OR r.recibo_ref IS DISTINCT FROM refs->>'recibo_ref'
           OR r.evento_ref IS DISTINCT FROM refs->>'evento_ref'
           OR r.registrada_en IS DISTINCT FROM v_instante
           OR r.expediente_anterior_json IS DISTINCT FROM p_operacion->'expediente_anterior'
           OR r.expediente_siguiente_json IS DISTINCT FROM p_operacion->'expediente_siguiente'
           OR (r.expediente_siguiente_json->'actuaciones')->-1 IS DISTINCT FROM p_operacion->'actuacion' THEN
            RAISE EXCEPTION 'recuperar preparación de subsanación confirmada' USING ERRCODE='40001';
        END IF;
        -- La repetición directa usa la misma evidencia ya consumida. Una
        -- petición HTTP nueva recupera mediante preparación y permiso fresco.
        IF r.decision_ref IS DISTINCT FROM a->>'decision_ref'
           OR r.decision_huella_sha256 IS DISTINCT FROM a->>'decision_huella_sha256' THEN
            RAISE EXCEPTION 'evidencia de replay de subsanación divergente' USING ERRCODE='42501';
        END IF;
        RETURN jsonb_build_object('esquema','vec.contratacion-temporal.resultado-subsanacion-reparos.v1','resultado','confirmada','recibo',r.recibo_json);
    END IF;
    SELECT v.* INTO v_actual FROM vec_contratacion_temporal.expediente_integral_actual ac
      JOIN vec_contratacion_temporal.expediente_version_integral v USING(expediente_ref,version)
     WHERE ac.expediente_ref=m->>'expediente_ref' FOR UPDATE OF ac,v;
    IF NOT FOUND THEN
        RETURN jsonb_build_object('esquema','vec.contratacion-temporal.resultado-subsanacion-reparos.v1','resultado','version_en_conflicto');
    END IF;
    IF v_actual.version IS DISTINCT FROM v_version
       OR v_actual.agregado_json IS DISTINCT FROM p_operacion->'expediente_anterior'
       OR v_actual.agregado_json->>'organizacion_ref' IS DISTINCT FROM m->>'organizacion_ref'
       OR v_actual.agregado_json->>'fase_actual' IS DISTINCT FROM 'subsanacion_unidad'
       OR v_actual.agregado_json->>'estado_actual' IS DISTINCT FROM 'incidencia'
       OR v_actual.agregado_json#>>'{fiscalizacion,resultado}' IS DISTINCT FROM 'desfavorable'
       OR v_actual.agregado_json#>>'{fiscalizacion,retorno,retorno_ref}' IS DISTINCT FROM p_operacion->>'retorno_ref'
       OR v_actual.agregado_json#>>'{fiscalizacion,retorno,estado}' IS DISTINCT FROM 'pendiente'
       OR jsonb_typeof(v_actual.agregado_json->'actuaciones') IS DISTINCT FROM 'array' THEN
        RETURN jsonb_build_object('esquema','vec.contratacion-temporal.resultado-subsanacion-reparos.v1','resultado','version_en_conflicto');
    END IF;
    SELECT u.* INTO v_retorno FROM vec_contratacion_temporal.retorno_fiscalizacion_unidad u
      JOIN vec_contratacion_temporal.reserva_fiscalizacion f USING(ambito_hmac)
     WHERE u.retorno_ref=p_operacion->>'retorno_ref' AND u.expediente_ref=m->>'expediente_ref'
       AND f.estado='confirmada' AND f.organizacion_ref=m->>'organizacion_ref'
       AND f.expediente_ref=u.expediente_ref AND f.retorno_ref=u.retorno_ref
       AND f.resultado='desfavorable'
       AND f.fiscalizacion_ref=v_actual.agregado_json#>>'{fiscalizacion,fiscalizacion_ref}'
     FOR SHARE OF u,f;
    IF NOT FOUND OR v_retorno.estado IS DISTINCT FROM 'pendiente'
       OR v_retorno.version_expediente>v_version
       OR v_retorno.unidad_ref IS DISTINCT FROM v_actual.agregado_json#>>'{asignacion,unidad_ref}'
       OR v_retorno.responsable_ref IS DISTINCT FROM v_actual.agregado_json#>>'{asignacion,responsable_ref}'
       OR v_retorno.unidad_ref IS DISTINCT FROM v_actual.agregado_json#>>'{fiscalizacion,retorno,unidad_ref}'
       OR v_retorno.responsable_ref IS DISTINCT FROM v_actual.agregado_json#>>'{fiscalizacion,retorno,responsable_ref}'
       OR v_retorno.creada_en IS DISTINCT FROM (v_actual.agregado_json#>>'{fiscalizacion,retorno,creado_en}')::timestamptz
       OR EXISTS (SELECT 1 FROM jsonb_array_elements(v_actual.agregado_json->'actuaciones') x
                   WHERE x->>'accion_clave'='contratacion_temporal.subsanacion_reparos.registrar'
                     AND x->>'retorno_ref'=p_operacion->>'retorno_ref') THEN
        RAISE EXCEPTION 'retorno de subsanación no disponible' USING ERRCODE='42501';
    END IF;
    v_secuencia_actuacion:=jsonb_array_length(v_actual.agregado_json->'actuaciones')+1;
    v_actuacion:=jsonb_build_object('secuencia',v_secuencia_actuacion,'version_expediente',v_version+1,
        'accion_clave','contratacion_temporal.subsanacion_reparos.registrar',
        'actor_ref',m->>'actor_ref','unidad_ref',v_retorno.unidad_ref,'recibo_ref',refs->>'recibo_ref',
        'realizada_en',p_operacion->'instante_efecto','fase_origen','subsanacion_unidad',
        'fase_destino','subsanacion_unidad','estado_origen','incidencia','estado_destino','incidencia',
        'observaciones',m->>'observaciones','retorno_ref',v_retorno.retorno_ref);
    v_siguiente:=v_actual.agregado_json||jsonb_build_object('version',v_version+1,
        'actualizado_en',p_operacion->'instante_efecto',
        'actuaciones',(v_actual.agregado_json->'actuaciones')||jsonb_build_array(v_actuacion));
    IF v_instante<(v_actual.agregado_json->>'actualizado_en')::timestamptz
       OR v_actuacion IS DISTINCT FROM p_operacion->'actuacion'
       OR v_siguiente IS DISTINCT FROM p_operacion->'expediente_siguiente' THEN
        RAISE EXCEPTION 'proyección de subsanación divergente' USING ERRCODE='22023';
    END IF;
    v_observaciones_huella:=encode(sha256(convert_to(m->>'observaciones','UTF8')),'hex');
    -- Mismo JSON canónico de mapas de Go; todas las referencias admitidas son ASCII.
    v_contexto_canonico:=convert_to('{"ambitos":{"estado_previo":"incidencia","expediente_ref":"'||(m->>'expediente_ref')||
        '","fase_previa":"subsanacion_unidad","organizacion_ref":"'||(m->>'organizacion_ref')||
        '"},"atributos":{"ambito_idempotencia_hmac":"'||(p_operacion->>'ambito_idempotencia_hmac')||
        '","huella_peticion_hmac":"'||(p_operacion->>'huella_peticion_hmac')||
        '","observaciones_huella_sha256":"'||v_observaciones_huella||
        '","politica_huella_sha256":"'||(pol->>'definicion_huella_sha256')||
        '","politica_ref":"'||(pol->>'definicion_ref')||'","politica_version":"'||(pol->>'definicion_version')||
        '","responsable_asignado_ref":"'||v_retorno.responsable_ref||'","retorno_ref":"'||v_retorno.retorno_ref||
        '","unidad_asignada_ref":"'||v_retorno.unidad_ref||'","version_expediente":"'||v_version::text||'"}}','UTF8');
    v_contexto_huella:=encode(sha256(v_contexto_canonico),'hex');
    IF a->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_contexto_huella
       OR v_decision->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_contexto_huella
       OR v_decision->>'principal_id' IS DISTINCT FROM m->>'actor_ref'
       OR v_decision->>'perfil_activo_ref' IS DISTINCT FROM m->>'perfil_ref'
       OR v_decision->>'recurso_ref' IS DISTINCT FROM m->>'expediente_ref'
       OR v_decision->>'accion' IS DISTINCT FROM 'contratacion_temporal.subsanacion_reparos.registrar'
       OR v_decision->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
       OR v_decision->>'tipo_recurso' IS DISTINCT FROM 'subsanacion_reparo_contratacion_temporal'
       OR v_decision->>'finalidad' IS DISTINCT FROM 'gestionar_contratacion_temporal'
       OR v_decision->>'decision_ref' IS DISTINCT FROM a->>'decision_ref' THEN
        RAISE EXCEPTION 'contexto autorizado de subsanación divergente' USING ERRCODE='42501';
    END IF;
    v_ahora:=date_trunc('microseconds',clock_timestamp());
    IF (pol->>'evaluada_en')::timestamptz>v_instante OR v_ahora<v_instante
       OR v_ahora>=(pol->>'valida_hasta')::timestamptz THEN
        RAISE EXCEPTION 'vigencia de subsanación agotada' USING ERRCODE='42501';
    END IF;
    SELECT * INTO STRICT v_consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_subsanacion_reparos_v3_atestada(
        p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
    IF v_consumo.decision_ref IS DISTINCT FROM a->>'decision_ref'
       OR v_consumo.efecto_ref IS DISTINCT FROM m->>'expediente_ref'
       OR v_consumo.huella_efecto_sha256 IS DISTINCT FROM v_contexto_huella
       OR coalesce(v_consumo.auditoria_ref,'') !~ '^aud_v3_[0-9a-f]{32}$'
       OR v_consumo.consumo_nuevo IS NOT TRUE THEN
        RAISE EXCEPTION 'consumo de subsanación divergente' USING ERRCODE='42501';
    END IF;
    v_ahora:=date_trunc('microseconds',clock_timestamp());
    IF v_ahora<v_instante OR v_ahora>=(pol->>'valida_hasta')::timestamptz THEN
        RAISE EXCEPTION 'vigencia final de subsanación agotada' USING ERRCODE='42501';
    END IF;
    v_agregado_huella:=encode(sha256(convert_to(v_siguiente::text,'UTF8')),'hex');
    v_prueba:=convert_to('VEC-CT-EXPEDIENTE-SUBSANACION-REPAROS-V1'||chr(10)||
        (m->>'expediente_ref')||chr(10)||(v_version+1)::text||chr(10)||v_agregado_huella||chr(10)||
        (refs->>'reserva_ref')||chr(10)||(refs->>'recibo_ref')||chr(10)||v_consumo.decision_ref||chr(10)||v_ahora::text,'UTF8');
    INSERT INTO vec_contratacion_temporal.expediente_version_integral(
        expediente_ref,version,agregado_json,agregado_json_huella_sha256,prueba_canonica,prueba_huella_sha256,
        flujo_ref,flujo_version,flujo_huella_sha256,fase_clave,estado,origen_version,operacion_ref,registrada_en)
    VALUES(m->>'expediente_ref',v_version+1,v_siguiente,v_agregado_huella,v_prueba,encode(sha256(v_prueba),'hex'),
        v_actual.flujo_ref,v_actual.flujo_version,v_actual.flujo_huella_sha256,'subsanacion_unidad','incidencia',
        'subsanacion_reparos_v1',refs->>'reserva_ref',v_ahora);
    UPDATE vec_contratacion_temporal.expediente_integral_actual
       SET version=v_version+1,actualizada_en=v_ahora,operacion_ref=refs->>'reserva_ref'
     WHERE expediente_ref=m->>'expediente_ref' AND version=v_version;
    IF NOT FOUND THEN RAISE EXCEPTION 'CAS final de subsanación perdido' USING ERRCODE='40001'; END IF;
    v_prueba:=convert_to('VEC-CT-ACTUACION-SUBSANACION-REPAROS-V1'||chr(10)||
        encode(sha256(convert_to(v_actuacion::text,'UTF8')),'hex')||chr(10)||
        (refs->>'recibo_ref')||chr(10)||v_ahora::text,'UTF8');
    INSERT INTO vec_contratacion_temporal.actuacion_expediente_integral(
        expediente_ref,secuencia,version_expediente,operacion_ref,recibo_ref,actuacion_json,
        actuacion_json_huella_sha256,prueba_canonica,prueba_huella_sha256,registrada_en)
    VALUES(m->>'expediente_ref',v_secuencia_actuacion,v_version+1,refs->>'reserva_ref',refs->>'recibo_ref',v_actuacion,
        encode(sha256(convert_to(v_actuacion::text,'UTF8')),'hex'),v_prueba,encode(sha256(v_prueba),'hex'),v_ahora);
    SELECT secuencia_outbox,cabeza_outbox_sha256 INTO STRICT v_secuencia,v_anterior
      FROM vec_contratacion_temporal.control_cadenas_expediente_integral WHERE control_id FOR UPDATE;
    IF v_secuencia>=9007199254740991 THEN RAISE EXCEPTION 'límite de outbox alcanzado' USING ERRCODE='22003'; END IF;
    v_secuencia:=v_secuencia+1;
    v_payload_evento:=convert_to(jsonb_build_object('esquema','vec.contratacion-temporal.subsanacion-reparos-registrada.v1',
        'expediente_ref',m->>'expediente_ref','version_resultante',v_version+1,'retorno_ref',v_retorno.retorno_ref,
        'recibo_ref',refs->>'recibo_ref','fase_resultante','subsanacion_unidad','estado_resultante','incidencia')::text,'UTF8');
    INSERT INTO vec_contratacion_temporal.outbox_expediente_integral(
        evento_ref,secuencia,operacion_ref,expediente_ref,version_expediente,tipo_evento,payload_canonico,
        payload_huella_sha256,anterior_sha256,huella_sha256,registrada_en)
    VALUES(refs->>'evento_ref',v_secuencia,refs->>'reserva_ref',m->>'expediente_ref',v_version+1,
        'contratacion_temporal.subsanacion_reparos_registrada',v_payload_evento,encode(sha256(v_payload_evento),'hex'),
        v_anterior,encode(sha256(v_anterior::bytea||v_payload_evento),'hex'),v_ahora);
    UPDATE vec_contratacion_temporal.control_cadenas_expediente_integral
       SET secuencia_outbox=v_secuencia,cabeza_outbox_sha256=encode(sha256(v_anterior::bytea||v_payload_evento),'hex'),actualizada_en=v_ahora
     WHERE control_id;
    v_recibo:=jsonb_build_object('operacion','registrar_subsanacion','organizacion_ref',m->>'organizacion_ref',
        'expediente_ref',m->>'expediente_ref','version_anterior',v_version,'version_resultante',v_version+1,
        'fase_resultante','subsanacion_unidad','estado_resultante','incidencia','recibo_ref',refs->>'recibo_ref',
        'auditoria_ref',v_consumo.auditoria_ref,'evento_ref',refs->>'evento_ref','actor_ref',m->>'actor_ref',
        'registrada_en',p_operacion->'instante_efecto');
    INSERT INTO vec_contratacion_temporal.reserva_subsanacion_reparos(
        ambito_hmac,huella_peticion_hmac,organizacion_ref,expediente_ref,version_esperada,actor_ref,perfil_ref,observaciones,
        estado,reserva_ref,recibo_ref,evento_ref,retorno_ref,expediente_anterior_json,expediente_siguiente_json,recibo_json,
        decision_ref,decision_huella_sha256,consumo_huella_sha256,auditoria_ref,politica_ref,politica_version,politica_huella_sha256,
        registrada_en,confirmada_en)
    VALUES(p_operacion->>'ambito_idempotencia_hmac',p_operacion->>'huella_peticion_hmac',m->>'organizacion_ref',
        m->>'expediente_ref',v_version,m->>'actor_ref',m->>'perfil_ref',m->>'observaciones','confirmada',
        refs->>'reserva_ref',refs->>'recibo_ref',refs->>'evento_ref',v_retorno.retorno_ref,v_actual.agregado_json,v_siguiente,v_recibo,
        v_consumo.decision_ref,a->>'decision_huella_sha256',v_consumo.consumo_huella_sha256,v_consumo.auditoria_ref,
        pol->>'definicion_ref',(pol->>'definicion_version')::numeric,pol->>'definicion_huella_sha256',v_instante,v_ahora);
    RETURN jsonb_build_object('esquema','vec.contratacion-temporal.resultado-subsanacion-reparos.v1','resultado','confirmada','recibo',v_recibo);
EXCEPTION
WHEN unique_violation THEN
    GET STACKED DIAGNOSTICS v_restriccion=CONSTRAINT_NAME,
        v_tabla_error=TABLE_NAME,v_esquema_error=SCHEMA_NAME;
    -- La misma reserva pudo confirmarse tras tomar la instantánea: clasificar
    -- sólo esa colisión después de revertir todo el efecto en este bloque.
    IF v_esquema_error='vec_contratacion_temporal'
       AND v_tabla_error='reserva_subsanacion_reparos'
       AND v_restriccion='reserva_subsanacion_reparos_pkey' THEN
        RETURN jsonb_build_object('esquema','vec.contratacion-temporal.resultado-subsanacion-reparos.v1','resultado','idempotencia_reutilizada');
    END IF;
    RAISE;
WHEN invalid_text_representation OR datetime_field_overflow OR numeric_value_out_of_range OR character_not_in_repertoire THEN
    RAISE EXCEPTION 'entrada de subsanación inválida' USING ERRCODE='22023';
END
$funcion$;

-- Preparación compañera incorporada y corregida en esta copia; original preservado.
CREATE FUNCTION vec_contratacion_temporal.preparar_subsanacion_reparos_v1(
    p_operacion jsonb
) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path = pg_catalog
SET row_security = 'on'
SET timezone = 'UTC'
SET lock_timeout = '2s'
AS $funcion$
DECLARE
    v_activo jsonb := p_operacion #> '{sellos_hmac,activo}';
    v_retenido jsonb;
    v_huella_buscada text;
    v_encontrada boolean := false;
    v_par jsonb;
    v_generaciones text[] := ARRAY[]::text[];
    v_actual record;
    v_reserva vec_contratacion_temporal.reserva_subsanacion_reparos%ROWTYPE;
BEGIN
    IF current_user<>'vec_contratacion_temporal_propietario' OR session_user=current_user
       OR NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
       OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
       OR pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
       OR current_setting('transaction_isolation')<>'serializable'
       OR current_setting('transaction_read_only')<>'on' THEN
        RAISE EXCEPTION 'preparación de subsanación no autorizada' USING ERRCODE='42501';
    END IF;
    IF p_operacion IS NULL OR pg_column_size(p_operacion)>65536
       OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(p_operacion,
           ARRAY['esquema','operacion','material','sellos_hmac','referencias_candidatas']) IS NOT TRUE THEN
        RAISE EXCEPTION 'entrada de preparación inválida' USING ERRCODE='22023';
    END IF;
    IF vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(p_operacion->'material',
           ARRAY['organizacion_ref','expediente_ref','version_esperada','actor_ref','perfil_ref','observaciones']) IS NOT TRUE
       OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(p_operacion->'referencias_candidatas',
           ARRAY['reserva_ref','recibo_ref','evento_ref']) IS NOT TRUE
       OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(p_operacion->'sellos_hmac',ARRAY['activo','retenidos']) IS NOT TRUE
       OR jsonb_typeof(p_operacion#>'{sellos_hmac,retenidos}') IS DISTINCT FROM 'array'
       OR EXISTS (SELECT 1 FROM jsonb_each(p_operacion) c WHERE jsonb_typeof(c.value) IS DISTINCT FROM
                    CASE WHEN c.key IN ('esquema','operacion') THEN 'string' ELSE 'object' END)
       OR EXISTS (SELECT 1 FROM jsonb_each(p_operacion->'material') c WHERE jsonb_typeof(c.value) IS DISTINCT FROM
                    CASE WHEN c.key='version_esperada' THEN 'number' ELSE 'string' END)
       OR EXISTS (SELECT 1 FROM jsonb_each(p_operacion->'referencias_candidatas') c WHERE jsonb_typeof(c.value) IS DISTINCT FROM 'string') THEN
        RAISE EXCEPTION 'estructura de preparación inválida' USING ERRCODE='22023';
    END IF;
    IF p_operacion->>'esquema'<>'vec.contratacion-temporal.preparar-subsanacion-reparos.v1'
       OR p_operacion->>'operacion'<>'registrar_subsanacion'
       OR jsonb_array_length(p_operacion#>'{sellos_hmac,retenidos}')>16
       OR p_operacion#>>'{material,version_esperada}' !~ '^[1-9][0-9]{0,15}$'
       OR (p_operacion#>>'{material,version_esperada}')::numeric>9007199254740990
       OR EXISTS (SELECT 1 FROM jsonb_each_text(p_operacion->'material') c
                   WHERE c.key IN ('organizacion_ref','expediente_ref','actor_ref','perfil_ref')
                     AND c.value !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$')
       OR EXISTS (SELECT 1 FROM jsonb_each_text(p_operacion->'referencias_candidatas') c
                   WHERE c.value !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$')
       OR char_length(p_operacion#>>'{material,observaciones}') NOT BETWEEN 1 AND 2000
       OR octet_length(p_operacion#>>'{material,observaciones}')>8192
       OR p_operacion#>>'{material,observaciones}'<>btrim(p_operacion#>>'{material,observaciones}',E' \t\n\r\f'||chr(11)||chr(133)||chr(160)||chr(5760)||chr(8192)||chr(8193)||chr(8194)||chr(8195)||chr(8196)||chr(8197)||chr(8198)||chr(8199)||chr(8200)||chr(8201)||chr(8202)||chr(8232)||chr(8233)||chr(8239)||chr(8287)||chr(12288))
       OR p_operacion#>>'{material,observaciones}'<>normalize(p_operacion#>>'{material,observaciones}',NFC)
       OR translate(p_operacion#>>'{material,observaciones}',E'\t\n','') ~ '[[:cntrl:]]' THEN
        RAISE EXCEPTION 'material de preparación inválido' USING ERRCODE='22023';
    END IF;
    FOR v_par IN SELECT value FROM jsonb_array_elements(
        jsonb_build_array(v_activo)||(p_operacion#>'{sellos_hmac,retenidos}')) LOOP
        IF vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(v_par,
               ARRAY['ambito_hmac','generacion','huella_peticion_hmac']) IS NOT TRUE THEN
            RAISE EXCEPTION 'par HMAC de preparación inválido' USING ERRCODE='22023';
        END IF;
        IF EXISTS (SELECT 1 FROM jsonb_each(v_par) c WHERE jsonb_typeof(c.value) IS DISTINCT FROM
                    CASE WHEN c.key='generacion' THEN 'number' ELSE 'string' END)
           OR v_par->>'generacion' !~ '^[1-9][0-9]{0,8}$'
           OR v_par->>'generacion'=ANY(v_generaciones)
           OR v_par->>'ambito_hmac' !~ ('^hmac-sha256:vec[.]contratacion-temporal[.]subsanacion-reparos[.]ambito/v'||(v_par->>'generacion')||':[0-9a-f]{64}$')
           OR v_par->>'huella_peticion_hmac' !~ ('^hmac-sha256:vec[.]contratacion-temporal[.]subsanacion-reparos[.]peticion/v'||(v_par->>'generacion')||':[0-9a-f]{64}$') THEN
            RAISE EXCEPTION 'dominio o generación HMAC inválidos' USING ERRCODE='22023';
        END IF;
        v_generaciones:=array_append(v_generaciones,v_par->>'generacion');
    END LOOP;
    PERFORM set_config('vec.ct_subsanacion.organizacion_ref',p_operacion#>>'{material,organizacion_ref}',true);
    PERFORM set_config('vec.ct_subsanacion.expediente_ref',p_operacion#>>'{material,expediente_ref}',true);
    PERFORM set_config('vec.ct_subsanacion.actor_ref',p_operacion#>>'{material,actor_ref}',true);
    PERFORM set_config('vec.ct_subsanacion.perfil_ref',p_operacion#>>'{material,perfil_ref}',true);
    PERFORM set_config('vec.ct_subsanacion.ambito_hmac',v_activo->>'ambito_hmac',true);

    -- La reserva sólo se lee: preparación no reserva, no consume autorización,
    -- no escribe historia ni outbox. Una confirmación previa se recupera con
    -- el mismo recibo y sus referencias originales.
    v_huella_buscada := v_activo->>'huella_peticion_hmac';
    SELECT r.* INTO v_reserva
      FROM vec_contratacion_temporal.reserva_subsanacion_reparos r
     WHERE r.ambito_hmac = v_activo->>'ambito_hmac';
    v_encontrada := FOUND;
    IF NOT v_encontrada THEN
      FOR v_retenido IN SELECT valor FROM pg_catalog.jsonb_array_elements(p_operacion #> '{sellos_hmac,retenidos}') AS e(valor) LOOP
        v_huella_buscada := v_retenido->>'huella_peticion_hmac';
        PERFORM set_config('vec.ct_subsanacion.ambito_hmac',v_retenido->>'ambito_hmac',true);
        SELECT r.* INTO v_reserva FROM vec_contratacion_temporal.reserva_subsanacion_reparos r WHERE r.ambito_hmac = v_retenido->>'ambito_hmac';
        v_encontrada := FOUND;
        EXIT WHEN v_encontrada;
      END LOOP;
    END IF;
    IF v_encontrada THEN
        IF v_reserva.huella_peticion_hmac IS DISTINCT FROM v_huella_buscada
           OR v_reserva.organizacion_ref IS DISTINCT FROM p_operacion #>> '{material,organizacion_ref}'
           OR v_reserva.expediente_ref IS DISTINCT FROM p_operacion #>> '{material,expediente_ref}'
           OR v_reserva.version_esperada IS DISTINCT FROM (p_operacion #>> '{material,version_esperada}')::numeric
           OR v_reserva.actor_ref IS DISTINCT FROM p_operacion #>> '{material,actor_ref}'
           OR v_reserva.perfil_ref IS DISTINCT FROM p_operacion #>> '{material,perfil_ref}'
           OR v_reserva.observaciones IS DISTINCT FROM p_operacion #>> '{material,observaciones}' THEN
            RETURN jsonb_build_object('esquema','vec.contratacion-temporal.resultado-subsanacion-reparos.v1','resultado','idempotencia_reutilizada');
        END IF;
        RETURN vec_contratacion_temporal.resultado_preparar_subsanacion_reparos_v1(v_reserva.ambito_hmac);
    END IF;

    SELECT a.version, v.agregado_json INTO v_actual
      FROM vec_contratacion_temporal.expediente_integral_actual a
      JOIN vec_contratacion_temporal.expediente_version_integral v USING (expediente_ref, version)
     WHERE a.expediente_ref = p_operacion #>> '{material,expediente_ref}';
    IF NOT FOUND OR v_actual.version IS DISTINCT FROM (p_operacion #>> '{material,version_esperada}')::numeric
       OR v_actual.agregado_json->>'organizacion_ref' IS DISTINCT FROM p_operacion #>> '{material,organizacion_ref}'
       OR v_actual.agregado_json->>'fase_actual' IS DISTINCT FROM 'subsanacion_unidad'
       OR v_actual.agregado_json->>'estado_actual' IS DISTINCT FROM 'incidencia'
       OR NOT (v_actual.agregado_json ? 'asignacion')
       OR NOT (v_actual.agregado_json ? 'fiscalizacion')
       OR v_actual.agregado_json #>> '{fiscalizacion,resultado}' IS DISTINCT FROM 'desfavorable'
       OR coalesce(v_actual.agregado_json#>>'{fiscalizacion,retorno,retorno_ref}','') !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR v_actual.agregado_json#>>'{fiscalizacion,retorno,estado}' IS DISTINCT FROM 'pendiente' THEN
        RETURN jsonb_build_object('esquema','vec.contratacion-temporal.resultado-subsanacion-reparos.v1','resultado','version_en_conflicto');
    END IF;
    PERFORM set_config('vec.ct_subsanacion.ambito_hmac',v_activo->>'ambito_hmac',true);
    RETURN jsonb_build_object(
      'esquema','vec.contratacion-temporal.resultado-subsanacion-reparos.v1',
      'resultado','preparada','material',p_operacion->'material',
      'expediente',v_actual.agregado_json,
      'referencias',p_operacion->'referencias_candidatas',
      'retorno_ref',v_actual.agregado_json #>> '{fiscalizacion,retorno,retorno_ref}',
      'ambito_idempotencia_hmac',v_activo->>'ambito_hmac',
      'huella_peticion_hmac',v_activo->>'huella_peticion_hmac');
END
$funcion$;

-- Cerrar también ACL introducidas por ALTER DEFAULT PRIVILEGES: ningún rol
-- ajeno conserva acceso por haber sido destinatario predeterminado del owner.
DO $cerrar_acl$
DECLARE v record; v_destinatario text;
BEGIN
    FOR v IN
        SELECT DISTINCT x.grantee
          FROM pg_class c CROSS JOIN LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) x
         WHERE c.oid='vec_contratacion_temporal.reserva_subsanacion_reparos'::regclass
           AND x.grantee<>c.relowner
    LOOP
        v_destinatario:=CASE WHEN v.grantee=0 THEN 'PUBLIC' ELSE quote_ident(pg_get_userbyid(v.grantee)) END;
        EXECUTE 'REVOKE ALL ON TABLE vec_contratacion_temporal.reserva_subsanacion_reparos FROM '||v_destinatario;
    END LOOP;
    FOR v IN
        SELECT DISTINCT p.oid::regprocedure AS firma,x.grantee
          FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x
         WHERE p.oid IN (
            'vec_contratacion_temporal.resultado_preparar_subsanacion_reparos_v1(text)'::regprocedure,
            'vec_contratacion_temporal.preparar_subsanacion_reparos_v1(jsonb)'::regprocedure,
            'vec_contratacion_temporal.confirmar_subsanacion_reparos_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure)
           AND x.grantee<>p.proowner
    LOOP
        v_destinatario:=CASE WHEN v.grantee=0 THEN 'PUBLIC' ELSE quote_ident(pg_get_userbyid(v.grantee)) END;
        EXECUTE 'REVOKE ALL ON FUNCTION '||v.firma::text||' FROM '||v_destinatario;
    END LOOP;
END
$cerrar_acl$;
REVOKE ALL ON TABLE vec_contratacion_temporal.reserva_subsanacion_reparos FROM PUBLIC,vec_contratacion_temporal_ejecutor;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.resultado_preparar_subsanacion_reparos_v1(text),
    vec_contratacion_temporal.preparar_subsanacion_reparos_v1(jsonb),
    vec_contratacion_temporal.confirmar_subsanacion_reparos_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
    FROM PUBLIC,vec_contratacion_temporal_ejecutor;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.preparar_subsanacion_reparos_v1(jsonb),
    vec_contratacion_temporal.confirmar_subsanacion_reparos_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
    TO vec_contratacion_temporal_ejecutor;
DO $comprobar_acl$
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_class c CROSS JOIN LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) x
         WHERE c.oid='vec_contratacion_temporal.reserva_subsanacion_reparos'::regclass
           AND (c.relowner<>'vec_contratacion_temporal_propietario'::regrole OR x.grantee<>c.relowner))
       OR EXISTS (
        SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x
         WHERE p.oid IN ('vec_contratacion_temporal.resultado_preparar_subsanacion_reparos_v1(text)'::regprocedure,
            'vec_contratacion_temporal.preparar_subsanacion_reparos_v1(jsonb)'::regprocedure,
            'vec_contratacion_temporal.confirmar_subsanacion_reparos_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure)
           AND (p.proowner<>'vec_contratacion_temporal_propietario'::regrole OR (x.grantee<>p.proowner
             AND NOT (p.proname IN ('confirmar_subsanacion_reparos_v1','preparar_subsanacion_reparos_v1') AND x.grantee='vec_contratacion_temporal_ejecutor'::regrole
                      AND x.privilege_type='EXECUTE' AND NOT x.is_grantable))))
       OR has_table_privilege('vec_contratacion_temporal_ejecutor','vec_contratacion_temporal.reserva_subsanacion_reparos','SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER')
       OR has_function_privilege('vec_contratacion_temporal_ejecutor','vec_contratacion_temporal.resultado_preparar_subsanacion_reparos_v1(text)','EXECUTE')
       OR NOT has_function_privilege('vec_contratacion_temporal_ejecutor','vec_contratacion_temporal.preparar_subsanacion_reparos_v1(jsonb)','EXECUTE')
       OR NOT has_function_privilege('vec_contratacion_temporal_ejecutor','vec_contratacion_temporal.confirmar_subsanacion_reparos_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE') THEN
        RAISE EXCEPTION 'ACL efectiva de subsanación incompatible' USING ERRCODE='42501';
    END IF;
END
$comprobar_acl$;
COMMIT;
