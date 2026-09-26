\set ON_ERROR_STOP on
-- CT124: incorporación acreditada (dudas 11 y 12 de RRHH, respuestas de
-- ejemplo): «acredita la incorporación la toma de posesión o el contrato
-- firmado y la confirma el centro; a GINPIX va la ficha actual, confirmada
-- con su número de alta; el cierre exige el cese registrado y GINPIX
-- confirmado».
--  * Confirmación de GINPIX (RRHH): número de alta y fecha de la ficha de la
--    incorporación acreditada más reciente. Añade versión y actuación del
--    expediente, consume AD3-88 (audiencia propia) y publica
--    `ct.ginpix-confirmada.v1`, todo en una transacción. Una por
--    incorporación.
--  * Cierre (CT115): si la regla exige `ginpix_confirmado`, el número y la
--    fecha del cierre deben ser exactamente los de esa confirmación; si no
--    la hay, el cierre se rechaza (`ginpix_no_confirmado`). CT115 no se
--    edita: en sus dos funciones de cierre se inserta la comprobación justo
--    después del rechazo por falta de cese, conservando el resto del cuerpo,
--    el propietario, la configuración y la ACL.
--  * Confirmación de la incorporación por el centro: el centro de la
--    petición ratificada y entregada a RRHH (CT67/CT68) confirma, con sus
--    perfiles nominales, la fecha y el documento que la acredita (tipo que
--    fija el catálogo para la modalidad del expediente; referencia y huella,
--    nunca el contenido). No cambia la versión del expediente: añade su
--    historia, consume AD3-88 y publica `ct.incorporacion-confirmada-centro.v1`.
--  * No incorporación (RRHH, duda 12): con la resolución y la segunda persona
--    que pida el catálogo, el expediente nombrado sin incorporación vuelve a
--    la fiscalización en curso, publica `ct.no-incorporacion.v1` (que Bolsa
--    000042 recibe para aplicar la baja) y deja la intención de siguiente
--    candidato, que la continuación de CT119 admite desde la aceptación.
-- Requiere CT115, CT68, CT119/CT121 y AD3-88. Historia de solo adición.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000124',0));

DO $prevalidacion$
DECLARE v_origen text;
BEGIN
    IF current_user<>'vec_contratacion_temporal_propietario' THEN
        RAISE EXCEPTION 'CT124: rol de migración incompatible' USING ERRCODE='55000';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_bolsa_llamamientos_propietario' AND NOT rolcanlogin) THEN
        RAISE EXCEPTION 'CT124: falta el propietario de Bolsa (destinatario de la comprobación de origen)' USING ERRCODE='55000';
    END IF;
    IF to_regclass('vec_contratacion_temporal.confirmacion_ginpix_v1') IS NOT NULL
       OR to_regclass('vec_contratacion_temporal.incorporacion_centro_v1') IS NOT NULL
       OR to_regclass('vec_contratacion_temporal.no_incorporacion_v1') IS NOT NULL THEN
        RAISE EXCEPTION 'CT124 ya instalada: no se reaplica' USING ERRCODE='55000';
    END IF;
    IF to_regclass('vec_contratacion_temporal.cese_nombramiento_v1') IS NULL
       OR to_regclass('vec_contratacion_temporal.cierre_expediente_v1') IS NULL
       OR to_regprocedure('vec_contratacion_temporal.validar_confirmacion_ct115(jsonb,text,text,text,text,text,bytea,bytea,numeric,numeric)') IS NULL
       OR to_regprocedure('vec_contratacion_temporal.validar_preparacion_ct115(jsonb,text,text,text)') IS NULL
       OR to_regprocedure('vec_contratacion_temporal.incorporacion_expediente_ct115(text,text)') IS NULL
       OR to_regclass('vec_contratacion_temporal.entrega_peticion_centro_confirmacion') IS NULL
       OR to_regclass('vec_contratacion_temporal.peticion_centro_revision') IS NULL
       OR to_regprocedure('vec_contratacion_temporal.continuar_llamamiento_rrhh_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
       OR strpos(pg_get_functiondef('vec_contratacion_temporal.leer_expediente_aviso_confirmado_v1(text,text,text)'::regprocedure),
                 $c$IN ('renuncia','expiracion_gobernada')$c$)=0 THEN
        RAISE EXCEPTION 'CT124: dependencias incompatibles (CT115, CT68, CT119 y CT121 requeridas)' USING ERRCODE='55000';
    END IF;
    IF to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_confirmacion_ginpix_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
       OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_incorporacion_centro_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
       OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_no_incorporacion_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
       OR NOT has_function_privilege(current_user,'vec_autorizacion_atestada_v3.registrar_y_consumir_no_incorporacion_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
       OR NOT has_function_privilege(current_user,'vec_autorizacion_atestada_v3.registrar_y_consumir_confirmacion_ginpix_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
       OR NOT has_function_privilege(current_user,'vec_autorizacion_atestada_v3.registrar_y_consumir_incorporacion_centro_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE') THEN
        RAISE EXCEPTION 'CT124: AD3-88 requerida' USING ERRCODE='55000';
    END IF;
    SELECT pg_get_constraintdef(oid) INTO STRICT v_origen FROM pg_constraint
     WHERE conrelid='vec_contratacion_temporal.expediente_version_integral'::regclass
       AND conname='expediente_version_integral_origen_version_check' AND contype='c' AND convalidated;
    IF strpos(v_origen,'''cierre_expediente_ct115''::text')=0 OR right(v_origen,4)<>'])))'
       OR strpos(v_origen,'confirmacion_ginpix_ct124')<>0 OR strpos(v_origen,'no_incorporacion_ct124')<>0 THEN
        RAISE EXCEPTION 'CT124: preimagen de origen de versión incompatible' USING ERRCODE='55000';
    END IF;
END
$prevalidacion$;

DO $origen$
DECLARE v_origen text;
BEGIN
    SELECT pg_get_constraintdef(oid) INTO STRICT v_origen FROM pg_constraint
     WHERE conrelid='vec_contratacion_temporal.expediente_version_integral'::regclass
       AND conname='expediente_version_integral_origen_version_check';
    ALTER TABLE vec_contratacion_temporal.expediente_version_integral
        DROP CONSTRAINT expediente_version_integral_origen_version_check;
    EXECUTE 'ALTER TABLE vec_contratacion_temporal.expediente_version_integral ADD CONSTRAINT expediente_version_integral_origen_version_check '
        ||left(v_origen,length(v_origen)-4)||', ''confirmacion_ginpix_ct124''::text, ''no_incorporacion_ct124''::text])))';
END
$origen$;

-- ============================================================ GINPIX
CREATE TABLE vec_contratacion_temporal.confirmacion_ginpix_v1 (
    ambito_hmac text PRIMARY KEY CHECK (ambito_hmac ~ '^hmac-sha256:vec[.]contratacion-temporal[.]confirmacion-ginpix[.]ambito/v[1-9][0-9]{0,8}:[0-9a-f]{64}$'),
    huella_peticion_hmac text NOT NULL CHECK (huella_peticion_hmac ~ '^hmac-sha256:vec[.]contratacion-temporal[.]confirmacion-ginpix[.]peticion/v[1-9][0-9]{0,8}:[0-9a-f]{64}$'),
    organizacion_ref text NOT NULL,
    expediente_ref text NOT NULL,
    version_esperada numeric(20,0) NOT NULL CHECK (version_esperada BETWEEN 1 AND 9007199254740990),
    actor_ref text NOT NULL,
    perfil_ref text NOT NULL,
    ginpix_numero text NOT NULL CHECK (ginpix_numero ~ '^[A-Za-z0-9][A-Za-z0-9._/-]{0,63}$'),
    ginpix_confirmada_en date NOT NULL CHECK (isfinite(ginpix_confirmada_en)),
    observaciones text NOT NULL CHECK (vec_contratacion_temporal.texto_valido_ct115(observaciones,2000,true)),
    incorporacion_ref text NOT NULL UNIQUE REFERENCES vec_contratacion_temporal.incorporacion_registro_v2(recibo_ref),
    inicio_incorporacion date NOT NULL CHECK (ginpix_confirmada_en>=inicio_incorporacion),
    estado text NOT NULL CHECK (estado='confirmada'),
    reserva_ref text NOT NULL UNIQUE,
    recibo_ref text NOT NULL UNIQUE,
    evento_ref text NOT NULL UNIQUE,
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
    registrada_en timestamptz(6) NOT NULL CHECK (isfinite(registrada_en)),
    confirmada_en timestamptz(6) NOT NULL CHECK (confirmada_en>=registrada_en),
    UNIQUE (expediente_ref,version_esperada),
    FOREIGN KEY (expediente_ref,version_esperada) REFERENCES vec_contratacion_temporal.expediente_version_integral
);
ALTER TABLE vec_contratacion_temporal.confirmacion_ginpix_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contratacion_temporal.confirmacion_ginpix_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY lectura_operacion_ct124 ON vec_contratacion_temporal.confirmacion_ginpix_v1 FOR SELECT TO vec_contratacion_temporal_propietario USING (
    organizacion_ref=current_setting('vec.ct115.organizacion_ref',true)
    AND expediente_ref=current_setting('vec.ct115.expediente_ref',true)
    AND pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
    AND NOT pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
    AND NOT pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER'));
CREATE POLICY escritura_operacion_ct124 ON vec_contratacion_temporal.confirmacion_ginpix_v1 FOR INSERT TO vec_contratacion_temporal_propietario WITH CHECK (
    organizacion_ref=current_setting('vec.ct115.organizacion_ref',true)
    AND expediente_ref=current_setting('vec.ct115.expediente_ref',true)
    AND actor_ref=current_setting('vec.ct115.actor_ref',true)
    AND perfil_ref=current_setting('vec.ct115.perfil_ref',true)
    AND ambito_hmac=current_setting('vec.ct115.ambito_hmac',true)
    AND pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
    AND NOT pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
    AND NOT pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER'));
CREATE TRIGGER confirmacion_ginpix_v1_inmutable BEFORE UPDATE OR DELETE ON vec_contratacion_temporal.confirmacion_ginpix_v1
    FOR EACH ROW EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();

-- Material de la confirmación: forma exacta y tipos. Devuelve la fecha.
CREATE FUNCTION vec_contratacion_temporal.validar_material_ginpix_ct124(m jsonb)
RETURNS date LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog AS $$
DECLARE k text; f date;
BEGIN
    IF m IS NULL OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(m,ARRAY['organizacion_ref','expediente_ref','version_esperada',
        'actor_ref','perfil_ref','ginpix_numero','ginpix_confirmada_en','observaciones']) IS NOT TRUE
       OR EXISTS (SELECT 1 FROM jsonb_each(m) c WHERE jsonb_typeof(c.value) IS DISTINCT FROM CASE WHEN c.key='version_esperada' THEN 'number' ELSE 'string' END)
       OR m->>'version_esperada' !~ '^[1-9][0-9]{0,15}$' OR (m->>'version_esperada')::numeric>9007199254740990
       OR m->>'ginpix_numero' !~ '^[A-Za-z0-9][A-Za-z0-9._/-]{0,63}$'
       OR m->>'ginpix_confirmada_en' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$'
       OR NOT vec_contratacion_temporal.texto_valido_ct115(m->>'observaciones',2000,true) THEN
        RAISE EXCEPTION 'CT124: material de GINPIX inválido' USING ERRCODE='22023';
    END IF;
    FOREACH k IN ARRAY ARRAY['organizacion_ref','expediente_ref','actor_ref','perfil_ref'] LOOP
        IF NOT vec_contratacion_temporal.referencia_valida_ct115(m->>k) THEN
            RAISE EXCEPTION 'CT124: referencia de GINPIX inválida' USING ERRCODE='22023';
        END IF;
    END LOOP;
    f:=(m->>'ginpix_confirmada_en')::date;
    IF to_char(f,'YYYY-MM-DD')<>m->>'ginpix_confirmada_en' THEN
        RAISE EXCEPTION 'CT124: fecha de GINPIX inválida' USING ERRCODE='22023';
    END IF;
    RETURN f;
EXCEPTION WHEN invalid_datetime_format OR datetime_field_overflow THEN
    RAISE EXCEPTION 'CT124: fecha de GINPIX inválida' USING ERRCODE='22023';
END
$$;

CREATE FUNCTION vec_contratacion_temporal.resultado_ginpix_ct124(r vec_contratacion_temporal.confirmacion_ginpix_v1)
RETURNS jsonb LANGUAGE sql STABLE SET search_path=pg_catalog AS $$
    SELECT jsonb_build_object('esquema','vec.contratacion-temporal.resultado-confirmacion-ginpix.v1','resultado','confirmada',
        'material',jsonb_build_object('organizacion_ref',r.organizacion_ref,'expediente_ref',r.expediente_ref,
            'version_esperada',r.version_esperada,'actor_ref',r.actor_ref,'perfil_ref',r.perfil_ref,'ginpix_numero',r.ginpix_numero,
            'ginpix_confirmada_en',to_char(r.ginpix_confirmada_en,'YYYY-MM-DD'),'observaciones',r.observaciones),
        'expediente',r.expediente_siguiente_json,
        'referencias',jsonb_build_object('reserva_ref',r.reserva_ref,'recibo_ref',r.recibo_ref,'evento_ref',r.evento_ref),
        'incorporacion',jsonb_build_object('recibo_ref',r.incorporacion_ref,'inicio',to_char(r.inicio_incorporacion,'YYYY-MM-DD')),
        'ambito_idempotencia_hmac',r.ambito_hmac,'huella_peticion_hmac',r.huella_peticion_hmac,'recibo',r.recibo_json)
$$;

-- Confirmación de GINPIX de la incorporación acreditada más reciente. Solo se
-- consulta con las marcas de la operación en curso (RLS).
CREATE FUNCTION vec_contratacion_temporal.ginpix_confirmado_ct124(p_organizacion text, p_expediente text)
RETURNS TABLE(ginpix_numero text, ginpix_confirmada_en date, recibo_ref text, registrada_en timestamptz)
LANGUAGE sql STABLE SET search_path=pg_catalog AS $$
    SELECT g.ginpix_numero, g.ginpix_confirmada_en, g.recibo_ref, g.registrada_en
      FROM vec_contratacion_temporal.confirmacion_ginpix_v1 g
     WHERE g.organizacion_ref=p_organizacion AND g.expediente_ref=p_expediente
       AND g.incorporacion_ref=(SELECT i.recibo_ref FROM vec_contratacion_temporal.incorporacion_expediente_ct115(p_organizacion,p_expediente) i)
$$;

CREATE FUNCTION vec_contratacion_temporal.preparar_confirmacion_ginpix_v1(p_operacion jsonb)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security='on' SET timezone='UTC' SET lock_timeout='2s'
AS $funcion$
DECLARE
    m jsonb:=p_operacion->'material'; v_pares jsonb; v_par jsonb; v_fecha date;
    r vec_contratacion_temporal.confirmacion_ginpix_v1%ROWTYPE; v_actual record; v_inc record;
    e text:='vec.contratacion-temporal.resultado-confirmacion-ginpix.v1';
BEGIN
    PERFORM vec_contratacion_temporal.exigir_sesion_ct115(true);
    v_pares:=vec_contratacion_temporal.validar_preparacion_ct115(p_operacion,'vec.contratacion-temporal.preparar-confirmacion-ginpix.v1','confirmar_ginpix','confirmacion-ginpix');
    v_fecha:=vec_contratacion_temporal.validar_material_ginpix_ct124(m);
    PERFORM set_config('vec.ct115.organizacion_ref',m->>'organizacion_ref',true);
    PERFORM set_config('vec.ct115.expediente_ref',m->>'expediente_ref',true);
    FOR v_par IN SELECT value FROM jsonb_array_elements(v_pares) LOOP
        SELECT * INTO r FROM vec_contratacion_temporal.confirmacion_ginpix_v1 WHERE ambito_hmac=v_par->>'ambito_hmac';
        IF FOUND THEN
            IF r.huella_peticion_hmac IS DISTINCT FROM v_par->>'huella_peticion_hmac' OR r.organizacion_ref<>m->>'organizacion_ref'
               OR r.expediente_ref<>m->>'expediente_ref' OR r.version_esperada<>(m->>'version_esperada')::numeric
               OR r.actor_ref<>m->>'actor_ref' OR r.perfil_ref<>m->>'perfil_ref' OR r.ginpix_numero<>m->>'ginpix_numero'
               OR r.ginpix_confirmada_en<>v_fecha OR r.observaciones<>m->>'observaciones' THEN
                RETURN jsonb_build_object('esquema',e,'resultado','idempotencia_reutilizada');
            END IF;
            RETURN vec_contratacion_temporal.resultado_ginpix_ct124(r);
        END IF;
    END LOOP;
    IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.ginpix_confirmado_ct124(m->>'organizacion_ref',m->>'expediente_ref')) THEN
        RETURN jsonb_build_object('esquema',e,'resultado','ginpix_existente');
    END IF;
    SELECT a.version, v.agregado_json INTO v_actual
      FROM vec_contratacion_temporal.expediente_integral_actual a
      JOIN vec_contratacion_temporal.expediente_version_integral v USING (expediente_ref,version)
     WHERE a.expediente_ref=m->>'expediente_ref';
    IF NOT FOUND OR v_actual.version<>(m->>'version_esperada')::numeric
       OR v_actual.agregado_json->>'organizacion_ref' IS DISTINCT FROM m->>'organizacion_ref'
       OR v_actual.agregado_json->>'fase_actual' IS DISTINCT FROM 'nombramiento'
       OR v_actual.agregado_json->>'estado_actual' IS DISTINCT FROM 'en_curso'
       OR NOT vec_contratacion_temporal.referencia_valida_ct115(v_actual.agregado_json#>>'{asignacion,unidad_ref}') THEN
        RETURN jsonb_build_object('esquema',e,'resultado','version_en_conflicto');
    END IF;
    SELECT * INTO v_inc FROM vec_contratacion_temporal.incorporacion_expediente_ct115(m->>'organizacion_ref',m->>'expediente_ref');
    IF NOT FOUND OR v_inc.inicio IS NULL THEN
        RETURN jsonb_build_object('esquema',e,'resultado','sin_incorporacion');
    END IF;
    IF v_fecha<v_inc.inicio THEN
        RETURN jsonb_build_object('esquema',e,'resultado','fecha_anterior_incorporacion');
    END IF;
    RETURN jsonb_build_object('esquema',e,'resultado','preparada','material',m,'expediente',v_actual.agregado_json,
        'referencias',p_operacion->'referencias_candidatas',
        'incorporacion',jsonb_build_object('recibo_ref',v_inc.recibo_ref,'inicio',to_char(v_inc.inicio,'YYYY-MM-DD')),
        'ambito_idempotencia_hmac',p_operacion#>>'{sellos_hmac,activo,ambito_hmac}',
        'huella_peticion_hmac',p_operacion#>>'{sellos_hmac,activo,huella_peticion_hmac}');
EXCEPTION WHEN invalid_text_representation OR datetime_field_overflow OR numeric_value_out_of_range OR character_not_in_repertoire THEN
    RAISE EXCEPTION 'CT124: entrada de GINPIX inválida' USING ERRCODE='22023';
END
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.confirmar_confirmacion_ginpix_v1(
    p_operacion jsonb,
    p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,
    p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security='on' SET timezone='UTC' SET lock_timeout='2s'
AS $funcion$
DECLARE
    m jsonb:=p_operacion->'material'; refs jsonb:=p_operacion->'referencias'; pol jsonb:=p_operacion->'politica'; a jsonb:=p_operacion->'autorizacion';
    e text:='vec.contratacion-temporal.resultado-confirmacion-ginpix.v1';
    r vec_contratacion_temporal.confirmacion_ginpix_v1%ROWTYPE;
    v_fecha date; v_version numeric; v_instante timestamptz; v_ahora timestamptz(6); v_actual record; v_inc record;
    v_decision jsonb; v_consumo record; v_actuacion jsonb; v_siguiente jsonb; v_ambitos jsonb; v_atributos jsonb; v_contexto text;
    v_secuencia_actuacion numeric; v_agregado_huella text; v_prueba bytea; v_payload bytea; v_anterior text; v_secuencia numeric;
    v_recibo jsonb; v_restriccion text; v_tabla text; v_esquema text;
BEGIN
    PERFORM vec_contratacion_temporal.exigir_sesion_ct115(false);
    PERFORM vec_contratacion_temporal.validar_confirmacion_ct115(p_operacion,'vec.contratacion-temporal.confirmar-confirmacion-ginpix.v1','confirmar_ginpix',
        'confirmacion-ginpix','contratacion_temporal.ginpix.confirmar','confirmar_ginpix_contratacion_temporal',p_decision,p_motivo,p_persona_version,p_perfil_version);
    v_fecha:=vec_contratacion_temporal.validar_material_ginpix_ct124(m);
    IF p_capacidad IS NULL OR p_contexto IS NULL OR p_payload IS NULL OR p_sobre IS NULL OR p_evidencia IS NULL OR p_raiz IS NULL THEN
        RAISE EXCEPTION 'CT124: autorización incompleta' USING ERRCODE='42501';
    END IF;
    v_version:=(m->>'version_esperada')::numeric;
    v_instante:=(p_operacion->>'instante_efecto')::timestamptz;
    v_decision:=convert_from(p_decision,'UTF8')::jsonb;
    PERFORM set_config('vec.ct115.organizacion_ref',m->>'organizacion_ref',true);
    PERFORM set_config('vec.ct115.expediente_ref',m->>'expediente_ref',true);
    PERFORM set_config('vec.ct115.actor_ref',m->>'actor_ref',true);
    PERFORM set_config('vec.ct115.perfil_ref',m->>'perfil_ref',true);
    PERFORM set_config('vec.ct115.ambito_hmac',p_operacion->>'ambito_idempotencia_hmac',true);
    SELECT * INTO r FROM vec_contratacion_temporal.confirmacion_ginpix_v1 WHERE ambito_hmac=p_operacion->>'ambito_idempotencia_hmac';
    IF FOUND THEN
        IF r.huella_peticion_hmac IS DISTINCT FROM p_operacion->>'huella_peticion_hmac' OR r.version_esperada<>v_version
           OR r.ginpix_numero<>m->>'ginpix_numero' OR r.ginpix_confirmada_en<>v_fecha OR r.observaciones<>m->>'observaciones' THEN
            RETURN jsonb_build_object('esquema',e,'resultado','idempotencia_reutilizada');
        END IF;
        IF r.decision_ref IS DISTINCT FROM a->>'decision_ref' THEN
            RAISE EXCEPTION 'CT124: evidencia de repetición divergente' USING ERRCODE='42501';
        END IF;
        RETURN jsonb_build_object('esquema',e,'resultado','confirmada','recibo',r.recibo_json);
    END IF;
    SELECT v.* INTO v_actual FROM vec_contratacion_temporal.expediente_integral_actual ac
      JOIN vec_contratacion_temporal.expediente_version_integral v USING (expediente_ref,version)
     WHERE ac.expediente_ref=m->>'expediente_ref' FOR UPDATE OF ac,v;
    IF NOT FOUND OR v_actual.version<>v_version
       OR v_actual.agregado_json IS DISTINCT FROM p_operacion->'expediente_anterior'
       OR v_actual.agregado_json->>'organizacion_ref' IS DISTINCT FROM m->>'organizacion_ref'
       OR v_actual.agregado_json->>'fase_actual' IS DISTINCT FROM 'nombramiento'
       OR v_actual.agregado_json->>'estado_actual' IS DISTINCT FROM 'en_curso'
       OR jsonb_typeof(v_actual.agregado_json->'actuaciones') IS DISTINCT FROM 'array'
       OR NOT vec_contratacion_temporal.referencia_valida_ct115(v_actual.agregado_json#>>'{asignacion,unidad_ref}') THEN
        RETURN jsonb_build_object('esquema',e,'resultado','version_en_conflicto');
    END IF;
    SELECT * INTO v_inc FROM vec_contratacion_temporal.incorporacion_expediente_ct115(m->>'organizacion_ref',m->>'expediente_ref');
    IF NOT FOUND OR v_inc.inicio IS NULL THEN
        RETURN jsonb_build_object('esquema',e,'resultado','sin_incorporacion');
    END IF;
    IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.confirmacion_ginpix_v1 WHERE incorporacion_ref=v_inc.recibo_ref) THEN
        RETURN jsonb_build_object('esquema',e,'resultado','ginpix_existente');
    END IF;
    IF v_fecha<v_inc.inicio THEN
        RETURN jsonb_build_object('esquema',e,'resultado','fecha_anterior_incorporacion');
    END IF;
    v_secuencia_actuacion:=jsonb_array_length(v_actual.agregado_json->'actuaciones')+1;
    v_actuacion:=jsonb_build_object('secuencia',v_secuencia_actuacion,'version_expediente',v_version+1,
        'accion_clave','contratacion_temporal.ginpix.confirmar','actor_ref',m->>'actor_ref',
        'unidad_ref',v_actual.agregado_json#>>'{asignacion,unidad_ref}','recibo_ref',refs->>'recibo_ref',
        'realizada_en',p_operacion->'instante_efecto','fase_origen','nombramiento','fase_destino','nombramiento',
        'estado_origen','en_curso','estado_destino','en_curso','documentos_ref',jsonb_build_array('ginpix:'||(m->>'ginpix_numero')));
    IF m->>'observaciones'<>'' THEN
        v_actuacion:=v_actuacion||jsonb_build_object('observaciones',m->>'observaciones');
    END IF;
    v_siguiente:=v_actual.agregado_json||jsonb_build_object('version',v_version+1,'actualizado_en',p_operacion->'instante_efecto',
        'actuaciones',(v_actual.agregado_json->'actuaciones')||jsonb_build_array(v_actuacion));
    IF v_instante<(v_actual.agregado_json->>'actualizado_en')::timestamptz
       OR v_actuacion IS DISTINCT FROM p_operacion->'actuacion'
       OR v_siguiente IS DISTINCT FROM p_operacion->'expediente_siguiente' THEN
        RAISE EXCEPTION 'CT124: proyección de GINPIX divergente' USING ERRCODE='22023';
    END IF;
    v_ambitos:=jsonb_build_object('organizacion_ref',m->>'organizacion_ref','expediente_ref',m->>'expediente_ref',
        'fase_previa','nombramiento','estado_previo','en_curso');
    v_atributos:=jsonb_build_object('version_expediente',v_version::text,'ginpix_numero',m->>'ginpix_numero',
        'ginpix_confirmada_en',m->>'ginpix_confirmada_en',
        'observaciones_huella_sha256',encode(sha256(convert_to(m->>'observaciones','UTF8')),'hex'),
        'incorporacion_ref',v_inc.recibo_ref,'politica_ref',pol->>'definicion_ref','politica_version',pol->>'definicion_version',
        'politica_huella_sha256',pol->>'definicion_huella_sha256','ambito_idempotencia_hmac',p_operacion->>'ambito_idempotencia_hmac',
        'huella_peticion_hmac',p_operacion->>'huella_peticion_hmac');
    IF p_operacion#>'{contexto,ambitos}' IS DISTINCT FROM v_ambitos OR p_operacion#>'{contexto,atributos}' IS DISTINCT FROM v_atributos THEN
        RAISE EXCEPTION 'CT124: contexto de GINPIX divergente' USING ERRCODE='42501';
    END IF;
    v_contexto:=vec_contratacion_temporal.huella_contexto_go_ct115(v_ambitos,v_atributos);
    IF a->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_contexto
       OR v_decision->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_contexto
       OR v_decision->>'principal_id' IS DISTINCT FROM m->>'actor_ref'
       OR v_decision->>'perfil_activo_ref' IS DISTINCT FROM m->>'perfil_ref'
       OR v_decision->>'recurso_ref' IS DISTINCT FROM m->>'expediente_ref'
       OR v_decision->>'accion' IS DISTINCT FROM 'contratacion_temporal.ginpix.confirmar'
       OR v_decision->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
       OR v_decision->>'tipo_recurso' IS DISTINCT FROM 'confirmacion_ginpix_contratacion_temporal'
       OR v_decision->>'finalidad' IS DISTINCT FROM 'confirmar_ginpix_contratacion_temporal'
       OR v_decision->>'decision_ref' IS DISTINCT FROM a->>'decision_ref' THEN
        RAISE EXCEPTION 'CT124: contexto autorizado de GINPIX divergente' USING ERRCODE='42501';
    END IF;
    v_ahora:=date_trunc('microseconds',clock_timestamp());
    IF (pol->>'evaluada_en')::timestamptz>v_instante OR v_ahora<v_instante OR v_ahora>=(pol->>'valida_hasta')::timestamptz THEN
        RAISE EXCEPTION 'CT124: vigencia de GINPIX agotada' USING ERRCODE='42501';
    END IF;
    SELECT * INTO STRICT v_consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_confirmacion_ginpix_ct_v3_atestada(
        p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
    IF v_consumo.decision_ref IS DISTINCT FROM a->>'decision_ref' OR v_consumo.efecto_ref IS DISTINCT FROM m->>'expediente_ref'
       OR v_consumo.huella_efecto_sha256 IS DISTINCT FROM v_contexto OR coalesce(v_consumo.auditoria_ref,'') !~ '^aud_v3_[0-9a-f]{32}$'
       OR v_consumo.consumo_nuevo IS NOT TRUE THEN
        RAISE EXCEPTION 'CT124: consumo de GINPIX divergente' USING ERRCODE='42501';
    END IF;
    v_ahora:=date_trunc('microseconds',clock_timestamp());
    IF v_ahora<v_instante OR v_ahora>=(pol->>'valida_hasta')::timestamptz THEN
        RAISE EXCEPTION 'CT124: vigencia final de GINPIX agotada' USING ERRCODE='42501';
    END IF;
    v_agregado_huella:=encode(sha256(convert_to(v_siguiente::text,'UTF8')),'hex');
    v_prueba:=convert_to('VEC-CT-EXPEDIENTE-GINPIX-CT124'||chr(10)||(m->>'expediente_ref')||chr(10)||(v_version+1)::text||chr(10)
        ||v_agregado_huella||chr(10)||(refs->>'reserva_ref')||chr(10)||(refs->>'recibo_ref')||chr(10)||v_consumo.decision_ref||chr(10)||v_ahora::text,'UTF8');
    INSERT INTO vec_contratacion_temporal.expediente_version_integral(
        expediente_ref,version,agregado_json,agregado_json_huella_sha256,prueba_canonica,prueba_huella_sha256,
        flujo_ref,flujo_version,flujo_huella_sha256,fase_clave,estado,origen_version,operacion_ref,registrada_en)
    VALUES(m->>'expediente_ref',v_version+1,v_siguiente,v_agregado_huella,v_prueba,encode(sha256(v_prueba),'hex'),
        v_actual.flujo_ref,v_actual.flujo_version,v_actual.flujo_huella_sha256,'nombramiento','en_curso',
        'confirmacion_ginpix_ct124',refs->>'reserva_ref',v_ahora);
    UPDATE vec_contratacion_temporal.expediente_integral_actual SET version=v_version+1,actualizada_en=v_ahora,operacion_ref=refs->>'reserva_ref'
     WHERE expediente_ref=m->>'expediente_ref' AND version=v_version;
    IF NOT FOUND THEN RAISE EXCEPTION 'CT124: CAS final de GINPIX perdido' USING ERRCODE='40001'; END IF;
    v_prueba:=convert_to('VEC-CT-ACTUACION-GINPIX-CT124'||chr(10)||encode(sha256(convert_to(v_actuacion::text,'UTF8')),'hex')||chr(10)
        ||(refs->>'recibo_ref')||chr(10)||v_ahora::text,'UTF8');
    INSERT INTO vec_contratacion_temporal.actuacion_expediente_integral(
        expediente_ref,secuencia,version_expediente,operacion_ref,recibo_ref,actuacion_json,
        actuacion_json_huella_sha256,prueba_canonica,prueba_huella_sha256,registrada_en)
    VALUES(m->>'expediente_ref',v_secuencia_actuacion,v_version+1,refs->>'reserva_ref',refs->>'recibo_ref',v_actuacion,
        encode(sha256(convert_to(v_actuacion::text,'UTF8')),'hex'),v_prueba,encode(sha256(v_prueba),'hex'),v_ahora);
    SELECT secuencia_outbox,cabeza_outbox_sha256 INTO STRICT v_secuencia,v_anterior
      FROM vec_contratacion_temporal.control_cadenas_expediente_integral WHERE control_id FOR UPDATE;
    IF v_secuencia>=9007199254740991 THEN RAISE EXCEPTION 'CT124: límite de outbox alcanzado' USING ERRCODE='22003'; END IF;
    v_secuencia:=v_secuencia+1;
    -- Evento `ct.ginpix-confirmada.v1`: solo referencias opacas, número y fecha.
    v_payload:=convert_to(jsonb_build_object('esquema','vec.contratacion-temporal.ginpix-confirmada.v1','organizacion_ref',m->>'organizacion_ref',
        'expediente_ref',m->>'expediente_ref','version_resultante',v_version+1,'incorporacion_ref',v_inc.recibo_ref,
        'ginpix_numero',m->>'ginpix_numero','ginpix_confirmada_en',m->>'ginpix_confirmada_en',
        'recibo_ref',refs->>'recibo_ref','registrada_en',p_operacion->'instante_efecto')::text,'UTF8');
    INSERT INTO vec_contratacion_temporal.outbox_expediente_integral(
        evento_ref,secuencia,operacion_ref,expediente_ref,version_expediente,tipo_evento,payload_canonico,
        payload_huella_sha256,anterior_sha256,huella_sha256,registrada_en)
    VALUES(refs->>'evento_ref',v_secuencia,refs->>'reserva_ref',m->>'expediente_ref',v_version+1,'ct.ginpix-confirmada.v1',v_payload,
        encode(sha256(v_payload),'hex'),v_anterior,encode(sha256(v_anterior::bytea||v_payload),'hex'),v_ahora);
    UPDATE vec_contratacion_temporal.control_cadenas_expediente_integral
       SET secuencia_outbox=v_secuencia,cabeza_outbox_sha256=encode(sha256(v_anterior::bytea||v_payload),'hex'),actualizada_en=v_ahora
     WHERE control_id;
    v_recibo:=jsonb_build_object('operacion','confirmar_ginpix','organizacion_ref',m->>'organizacion_ref','expediente_ref',m->>'expediente_ref',
        'version_anterior',v_version,'version_resultante',v_version+1,'fase_resultante','nombramiento','estado_resultante','en_curso',
        'ginpix_numero',m->>'ginpix_numero','recibo_ref',refs->>'recibo_ref',
        'auditoria_ref',v_consumo.auditoria_ref,'evento_ref',refs->>'evento_ref','actor_ref',m->>'actor_ref',
        'registrada_en',p_operacion->'instante_efecto');
    INSERT INTO vec_contratacion_temporal.confirmacion_ginpix_v1(
        ambito_hmac,huella_peticion_hmac,organizacion_ref,expediente_ref,version_esperada,actor_ref,perfil_ref,ginpix_numero,ginpix_confirmada_en,
        observaciones,incorporacion_ref,inicio_incorporacion,estado,reserva_ref,recibo_ref,evento_ref,expediente_anterior_json,expediente_siguiente_json,
        recibo_json,decision_ref,decision_huella_sha256,consumo_huella_sha256,auditoria_ref,politica_ref,politica_version,politica_huella_sha256,
        registrada_en,confirmada_en)
    VALUES(p_operacion->>'ambito_idempotencia_hmac',p_operacion->>'huella_peticion_hmac',m->>'organizacion_ref',m->>'expediente_ref',v_version,
        m->>'actor_ref',m->>'perfil_ref',m->>'ginpix_numero',v_fecha,m->>'observaciones',v_inc.recibo_ref,v_inc.inicio,'confirmada',
        refs->>'reserva_ref',refs->>'recibo_ref',refs->>'evento_ref',v_actual.agregado_json,v_siguiente,v_recibo,v_consumo.decision_ref,
        a->>'decision_huella_sha256',v_consumo.consumo_huella_sha256,v_consumo.auditoria_ref,pol->>'definicion_ref',
        (pol->>'definicion_version')::numeric,pol->>'definicion_huella_sha256',v_instante,v_ahora);
    RETURN jsonb_build_object('esquema',e,'resultado','confirmada','recibo',v_recibo);
EXCEPTION
WHEN unique_violation THEN
    GET STACKED DIAGNOSTICS v_restriccion=CONSTRAINT_NAME,v_tabla=TABLE_NAME,v_esquema=SCHEMA_NAME;
    IF v_esquema='vec_contratacion_temporal' AND v_tabla='confirmacion_ginpix_v1' THEN
        RETURN jsonb_build_object('esquema',e,'resultado',CASE WHEN v_restriccion='confirmacion_ginpix_v1_pkey' THEN 'idempotencia_reutilizada' ELSE 'ginpix_existente' END);
    END IF;
    RAISE;
WHEN invalid_text_representation OR datetime_field_overflow OR numeric_value_out_of_range OR character_not_in_repertoire THEN
    RAISE EXCEPTION 'CT124: entrada de GINPIX inválida' USING ERRCODE='22023';
END
$funcion$;

-- ============================================================ CIERRE (CT115)
-- En las dos funciones del cierre, justo tras el rechazo por falta de cese:
-- con la condición `ginpix_confirmado`, el número y la fecha deben ser los de
-- la confirmación de GINPIX de la incorporación vigente.
DO $cierre$
DECLARE
    v_funcion text; v_ancla text; v_antes record; v_despues record; v_definicion text; v_insercion text;
    v_comprobacion text:=$c$
    IF m->'condiciones' @> '["ginpix_confirmado"]'::jsonb AND NOT EXISTS (
        SELECT 1 FROM vec_contratacion_temporal.ginpix_confirmado_ct124(m->>'organizacion_ref',m->>'expediente_ref') g
         WHERE g.ginpix_numero=m->>'ginpix_numero' AND to_char(g.ginpix_confirmada_en,'YYYY-MM-DD')=m->>'ginpix_confirmada_en') THEN
        -- CT124: GINPIX de la confirmación registrada.
        RETURN jsonb_build_object('esquema',e,'resultado','ginpix_no_confirmado');
    END IF;$c$;
BEGIN
    FOR v_funcion, v_ancla IN SELECT * FROM (VALUES
        ('preparar_cierre_expediente_v1(jsonb)',
         E'    IF v_cese.recibo_ref IS NULL THEN\n        RETURN jsonb_build_object(''esquema'',e,''resultado'',''sin_cese'');\n    END IF;'),
        ('confirmar_cierre_expediente_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
         E'    IF NOT FOUND THEN\n        RETURN jsonb_build_object(''esquema'',e,''resultado'',''sin_cese'');\n    END IF;')
    ) AS f(firma,ancla) LOOP
        SELECT p.oid,pg_get_functiondef(p.oid) AS definicion,p.proacl AS acl,p.proowner AS propietario,
               p.proconfig AS configuracion,p.prosecdef AS definidor
          INTO v_antes FROM pg_proc p
         WHERE p.oid=to_regprocedure('vec_contratacion_temporal.'||v_funcion)
           AND p.proowner='vec_contratacion_temporal_propietario'::regrole AND p.prosecdef;
        IF NOT FOUND THEN
            RAISE EXCEPTION 'CT124: función ausente: %',v_funcion USING ERRCODE='55000';
        END IF;
        v_definicion:=v_antes.definicion;
        IF strpos(v_definicion,'ginpix_confirmado_ct124')<>0 THEN
            RAISE EXCEPTION 'CT124 ya instalada: no se reaplica' USING ERRCODE='55000';
        END IF;
        IF length(v_definicion)-length(replace(v_definicion,v_ancla,''))<>length(v_ancla) THEN
            RAISE EXCEPTION 'CT124: preimagen de cierre incompatible: %',v_funcion USING ERRCODE='55000';
        END IF;
        v_insercion:=v_ancla||v_comprobacion;
        v_definicion:=replace(v_definicion,v_ancla,v_insercion);
        EXECUTE v_definicion;
        SELECT pg_get_functiondef(p.oid) AS definicion,p.proacl AS acl,p.proowner AS propietario,
               p.proconfig AS configuracion,p.prosecdef AS definidor
          INTO STRICT v_despues FROM pg_proc p WHERE p.oid=v_antes.oid;
        IF v_despues.definicion IS DISTINCT FROM v_definicion
           OR replace(v_despues.definicion,v_insercion,v_ancla) IS DISTINCT FROM v_antes.definicion
           OR v_despues.acl IS DISTINCT FROM v_antes.acl
           OR v_despues.propietario IS DISTINCT FROM v_antes.propietario
           OR v_despues.configuracion IS DISTINCT FROM v_antes.configuracion
           OR v_despues.definidor IS NOT TRUE THEN
            RAISE EXCEPTION 'CT124: definición o permisos alterados: %',v_funcion USING ERRCODE='55000';
        END IF;
    END LOOP;
END
$cierre$;

-- ============================================================ CENTRO
-- Una confirmación por expediente. La modalidad es la del expediente; el
-- tipo de documento lo fija la regla del catálogo (referencia y huella).
CREATE TABLE vec_contratacion_temporal.incorporacion_centro_v1 (
    confirmacion_ref text PRIMARY KEY CHECK (confirmacion_ref ~ '^confirmacion:incorporacion-centro:[0-9a-f-]{36}$'),
    clave_idempotencia uuid NOT NULL UNIQUE,
    material text NOT NULL CHECK (octet_length(material) BETWEEN 1 AND 65536),
    material_sha256 text NOT NULL CHECK (material_sha256=encode(sha256(convert_to(material,'UTF8')),'hex')),
    organizacion_ref text NOT NULL CHECK (organizacion_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
    peticion_ref text NOT NULL REFERENCES vec_contratacion_temporal.entrega_peticion_centro_confirmacion(peticion_ref),
    expediente_ref text NOT NULL UNIQUE,
    version_expediente numeric(20,0) NOT NULL CHECK (version_expediente BETWEEN 1 AND 9007199254740991),
    actor_ref text NOT NULL, perfil_ref text NOT NULL, centro_ref text NOT NULL, puesto_ref text NOT NULL,
    modalidad_clave text NOT NULL CHECK (modalidad_clave ~ '^[a-z][a-z0-9_.-]{1,79}$'),
    fecha_incorporacion date NOT NULL CHECK (isfinite(fecha_incorporacion)),
    documento_tipo text NOT NULL CHECK (documento_tipo ~ '^[a-z][a-z0-9_.-]{1,79}$'),
    documento_ref text NOT NULL CHECK (documento_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
    documento_sha256 text NOT NULL CHECK (documento_sha256 ~ '^[0-9a-f]{64}$' AND documento_sha256<>repeat('0',64)),
    regla_ref text NOT NULL CHECK (regla_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,254}$'),
    regla_huella_sha256 text NOT NULL CHECK (regla_huella_sha256 ~ '^[0-9a-f]{64}$'),
    recibo_ref text NOT NULL UNIQUE,
    recibo_json jsonb NOT NULL CHECK (jsonb_typeof(recibo_json)='object'),
    evento_ref text NOT NULL UNIQUE,
    auditoria_ref text NOT NULL UNIQUE, decision_ref text NOT NULL UNIQUE,
    consumo_huella_sha256 text NOT NULL UNIQUE,
    registrada_en timestamptz(6) NOT NULL CHECK (registrada_en=date_trunc('microseconds',registrada_en)),
    FOREIGN KEY (expediente_ref,version_expediente) REFERENCES vec_contratacion_temporal.expediente_version_integral
);
-- Una fila por consulta autorizada del centro; no conserva datos del resultado.
CREATE TABLE vec_contratacion_temporal.incorporacion_centro_acceso_v1 (
    acceso_ref text PRIMARY KEY,
    recurso_ref text NOT NULL,
    actor_ref text NOT NULL, perfil_ref text NOT NULL, centro_ref text NOT NULL, puesto_ref text NOT NULL,
    auditoria_ref text NOT NULL UNIQUE, decision_ref text NOT NULL UNIQUE,
    consumo_huella_sha256 text NOT NULL UNIQUE,
    consultada_en timestamptz(6) NOT NULL
);
DO $seguridad_centro$
DECLARE v_tabla text;
BEGIN
    FOREACH v_tabla IN ARRAY ARRAY['incorporacion_centro_v1','incorporacion_centro_acceso_v1'] LOOP
        EXECUTE format('ALTER TABLE vec_contratacion_temporal.%I ENABLE ROW LEVEL SECURITY',v_tabla);
        EXECUTE format('ALTER TABLE vec_contratacion_temporal.%I FORCE ROW LEVEL SECURITY',v_tabla);
        EXECUTE format($p$CREATE POLICY propietario_ct124 ON vec_contratacion_temporal.%I TO vec_contratacion_temporal_propietario
            USING (pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
                   AND NOT pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER'))
            WITH CHECK (pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
                   AND NOT pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER'))$p$,v_tabla);
        EXECUTE format('CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_contratacion_temporal.%I FOR EACH ROW EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1()',v_tabla);
    END LOOP;
END
$seguridad_centro$;

-- Actor del centro: las cuatro referencias opacas, sin otras claves.
CREATE FUNCTION vec_contratacion_temporal.actor_centro_valido_ct124(a jsonb)
RETURNS boolean LANGUAGE sql IMMUTABLE SET search_path=pg_catalog AS $$
    SELECT jsonb_typeof(a)='object' AND (SELECT count(*) FROM jsonb_object_keys(a))=4
       AND (a-ARRAY['actor_ref','perfil_ref','centro_ref','puesto_ref'])='{}'::jsonb
       AND NOT EXISTS (SELECT 1 FROM jsonb_each(a) x WHERE jsonb_typeof(x.value) IS DISTINCT FROM 'string'
                        OR (x.value #>> '{}') !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$')
$$;

-- Peticiones ratificadas del centro del actor (como solicitante o ratificador)
-- y entregadas a RRHH, con su expediente vigente.
CREATE FUNCTION vec_contratacion_temporal.expedientes_centro_ct124(a jsonb, p_organizacion text)
RETURNS TABLE(peticion_ref text, peticion jsonb, expediente_ref text, numero_visible text, version numeric, agregado jsonb)
LANGUAGE sql STABLE SET search_path=pg_catalog AS $$
    SELECT u.peticion_ref, u.peticion, c.expediente_ref, c.numero_visible, ac.version, v.agregado_json
      FROM (SELECT DISTINCT ON (r.peticion_ref) r.peticion_ref, r.peticion, r.estado, r.centro_ref
              FROM vec_contratacion_temporal.peticion_centro_revision r
             WHERE r.centro_ref=a->>'centro_ref'
             ORDER BY r.peticion_ref, r.version DESC) u
      JOIN vec_contratacion_temporal.entrega_peticion_centro_confirmacion c ON c.peticion_ref=u.peticion_ref
      JOIN vec_contratacion_temporal.expediente_integral_actual ac ON ac.expediente_ref=c.expediente_ref
      JOIN vec_contratacion_temporal.expediente_version_integral v ON v.expediente_ref=ac.expediente_ref AND v.version=ac.version
     WHERE u.estado='ratificada'
       AND (a=u.peticion->'configuracion'->'solicitante' OR a=u.peticion->'configuracion'->'ratificador')
       AND v.agregado_json->>'organizacion_ref'=p_organizacion
$$;

CREATE FUNCTION vec_contratacion_temporal.consumir_incorporacion_centro_ct124(
    p_accion text, p_recurso text, a jsonb, p_organizacion text, p_material text,
    p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea,
    OUT decision_ref text, OUT auditoria_ref text, OUT consumo_huella_sha256 text)
LANGUAGE plpgsql VOLATILE SET search_path=pg_catalog AS $$
DECLARE d jsonb; v_contexto text; v_consumo record;
BEGIN
    v_contexto:=vec_contratacion_temporal.huella_contexto_go_ct115(
        jsonb_build_object('centro_ref',a->>'centro_ref','organizacion_ref',p_organizacion),
        jsonb_build_object('material_sha256',encode(sha256(convert_to(p_material,'UTF8')),'hex')));
    IF p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 1 AND 524288 THEN
        RAISE EXCEPTION 'CT124: decisión del centro inválida' USING ERRCODE='42501';
    END IF;
    BEGIN d:=convert_from(p_decision,'UTF8')::jsonb;
    EXCEPTION WHEN data_exception THEN RAISE EXCEPTION 'CT124: decisión del centro inválida' USING ERRCODE='42501'; END;
    IF d->>'accion' IS DISTINCT FROM p_accion OR d->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
       OR d->>'tipo_recurso' IS DISTINCT FROM 'incorporacion_centro_contratacion_temporal'
       OR d->>'finalidad' IS DISTINCT FROM 'gestionar_peticion_centro' OR d->>'recurso_ref' IS DISTINCT FROM p_recurso
       OR d->>'principal_id' IS DISTINCT FROM a->>'actor_ref' OR d->>'perfil_activo_ref' IS DISTINCT FROM a->>'perfil_ref'
       OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_contexto THEN
        RAISE EXCEPTION 'CT124: autorización del centro divergente' USING ERRCODE='42501';
    END IF;
    SELECT * INTO STRICT v_consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_incorporacion_centro_v3_atestada(
        p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
    IF v_consumo.consumo_nuevo IS NOT TRUE OR v_consumo.efecto_ref IS DISTINCT FROM p_recurso
       OR v_consumo.huella_efecto_sha256 IS DISTINCT FROM v_contexto
       OR coalesce(v_consumo.auditoria_ref,'') !~ '^aud_v3_[0-9a-f]{32}$' THEN
        RAISE EXCEPTION 'CT124: consumo del centro divergente' USING ERRCODE='42501';
    END IF;
    decision_ref:=v_consumo.decision_ref; auditoria_ref:=v_consumo.auditoria_ref; consumo_huella_sha256:=v_consumo.consumo_huella_sha256;
END
$$;

-- Bandeja del centro: expedientes de sus peticiones y su confirmación.
CREATE FUNCTION vec_contratacion_temporal.consultar_incorporaciones_centro_v1(
    p_consulta text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,
    p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s'
AS $funcion$
DECLARE q jsonb; a jsonb; v_org text; v_recurso text; v_consumo record; v_resultado jsonb;
BEGIN
    IF current_user<>'vec_contratacion_temporal_propietario' OR session_user=current_user
       OR NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
       OR pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER') THEN
        RAISE EXCEPTION 'CT124: sesión no autorizada' USING ERRCODE='42501';
    END IF;
    IF p_consulta IS NULL OR octet_length(p_consulta) NOT BETWEEN 1 AND 4096 THEN
        RAISE EXCEPTION 'CT124: consulta del centro inválida' USING ERRCODE='22023';
    END IF;
    BEGIN q:=p_consulta::jsonb;
    EXCEPTION WHEN data_exception THEN RAISE EXCEPTION 'CT124: consulta del centro inválida' USING ERRCODE='22023'; END;
    a:=q->'actor'; v_org:=q->>'organizacion_ref';
    IF jsonb_typeof(q) IS DISTINCT FROM 'object' OR (SELECT count(*) FROM jsonb_object_keys(q))<>3
       OR (q-ARRAY['modo','organizacion_ref','actor'])<>'{}'::jsonb OR q->>'modo' IS DISTINCT FROM 'bandeja'
       OR NOT vec_contratacion_temporal.actor_centro_valido_ct124(a) OR NOT vec_contratacion_temporal.referencia_valida_ct115(v_org) THEN
        RAISE EXCEPTION 'CT124: consulta del centro inválida' USING ERRCODE='22023';
    END IF;
    v_recurso:='incorporaciones:centro:'||(a->>'centro_ref');
    SELECT * INTO STRICT v_consumo FROM vec_contratacion_temporal.consumir_incorporacion_centro_ct124('contratacion_temporal.incorporacion.consultar_centro',
        v_recurso,a,v_org,p_consulta,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
    SELECT coalesce(jsonb_agg(x.fila ORDER BY x.orden DESC, x.expediente_ref),'[]'::jsonb) INTO v_resultado FROM (
        SELECT e.expediente_ref, (e.agregado->>'actualizado_en') AS orden, jsonb_build_object(
            'peticion_ref',e.peticion_ref,'expediente_ref',e.expediente_ref,'numero_visible',e.numero_visible,
            'version',e.version,'fase',e.agregado->>'fase_actual','estado',e.agregado->>'estado_actual',
            'modalidad_clave',e.agregado#>>'{analisis,modalidad_clave}',
            'categoria_ref',e.peticion#>>'{solicitud,categoria_ref}','periodo',e.peticion#>'{solicitud,periodo}',
            'confirmacion',(SELECT jsonb_build_object('fecha_incorporacion',to_char(i.fecha_incorporacion,'YYYY-MM-DD'),
                    'documento_tipo',i.documento_tipo,'documento_ref',i.documento_ref,'recibo_ref',i.recibo_ref,'registrada_en',i.registrada_en)
                FROM vec_contratacion_temporal.incorporacion_centro_v1 i WHERE i.expediente_ref=e.expediente_ref)) AS fila
          FROM vec_contratacion_temporal.expedientes_centro_ct124(a,v_org) e
         ORDER BY e.agregado->>'actualizado_en' DESC, e.expediente_ref LIMIT 50) x;
    INSERT INTO vec_contratacion_temporal.incorporacion_centro_acceso_v1(acceso_ref,recurso_ref,actor_ref,perfil_ref,centro_ref,puesto_ref,
        auditoria_ref,decision_ref,consumo_huella_sha256,consultada_en)
    VALUES('acceso:incorporacion-centro:'||gen_random_uuid()::text,v_recurso,a->>'actor_ref',a->>'perfil_ref',a->>'centro_ref',a->>'puesto_ref',
        v_consumo.auditoria_ref,v_consumo.decision_ref,v_consumo.consumo_huella_sha256,date_trunc('microseconds',clock_timestamp()));
    RETURN jsonb_build_object('esquema','vec.contratacion-temporal.incorporaciones-centro.v1','expedientes',v_resultado);
END
$funcion$;

-- Confirmación de la incorporación por el centro. Errores: 22023 material,
-- 42501 autorización, P0681 clave reutilizada con otro contenido, P0682 el
-- expediente no admite la confirmación (fase, petición ajena o ya confirmada).
CREATE FUNCTION vec_contratacion_temporal.confirmar_incorporacion_centro_v1(
    p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,
    p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s'
AS $funcion$
DECLARE
    m jsonb; a jsonb; doc jsonb; regla jsonb; v_clave uuid; v_fecha date; v_org text; v_consumo record;
    v_previa vec_contratacion_temporal.incorporacion_centro_v1%ROWTYPE; v_exp record; v_ahora timestamptz(6);
    v_ref text; v_recibo jsonb; v_payload bytea; v_anterior text; v_secuencia numeric; v_evento text;
BEGIN
    IF current_user<>'vec_contratacion_temporal_propietario' OR session_user=current_user
       OR NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
       OR pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
       OR current_setting('transaction_read_only')<>'off' THEN
        RAISE EXCEPTION 'CT124: sesión no autorizada' USING ERRCODE='42501';
    END IF;
    IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 1 AND 65536 THEN
        RAISE EXCEPTION 'CT124: material del centro inválido' USING ERRCODE='22023';
    END IF;
    BEGIN
        m:=p_material::jsonb; a:=m->'actor'; doc:=m->'documento'; regla:=m->'regla'; v_org:=m->>'organizacion_ref';
        IF jsonb_typeof(m) IS DISTINCT FROM 'object' OR (SELECT count(*) FROM jsonb_object_keys(m))<>9
           OR (m-ARRAY['operacion','clave_idempotencia','organizacion_ref','actor','peticion_ref','expediente_ref','fecha_incorporacion','documento','regla'])<>'{}'::jsonb
           OR (SELECT count(*) FROM json_each(p_material::json))<>9
           OR m->>'operacion' IS DISTINCT FROM 'confirmar'
           OR jsonb_typeof(m->'clave_idempotencia') IS DISTINCT FROM 'string'
           OR m->>'clave_idempotencia' !~ '^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
           OR NOT vec_contratacion_temporal.actor_centro_valido_ct124(a)
           OR NOT vec_contratacion_temporal.referencia_valida_ct115(v_org)
           OR NOT vec_contratacion_temporal.referencia_valida_ct115(m->>'peticion_ref')
           OR NOT vec_contratacion_temporal.referencia_valida_ct115(m->>'expediente_ref')
           OR jsonb_typeof(m->'fecha_incorporacion') IS DISTINCT FROM 'string'
           OR m->>'fecha_incorporacion' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$'
           OR jsonb_typeof(doc) IS DISTINCT FROM 'object' OR (SELECT count(*) FROM jsonb_object_keys(doc))<>3
           OR (doc-ARRAY['tipo','referencia','sha256'])<>'{}'::jsonb
           OR doc->>'tipo' !~ '^[a-z][a-z0-9_.-]{1,79}$'
           OR NOT vec_contratacion_temporal.referencia_valida_ct115(doc->>'referencia')
           OR doc->>'sha256' !~ '^[0-9a-f]{64}$' OR doc->>'sha256'=repeat('0',64)
           OR jsonb_typeof(regla) IS DISTINCT FROM 'object' OR (SELECT count(*) FROM jsonb_object_keys(regla))<>3
           OR (regla-ARRAY['referencia','huella_sha256','modalidad_clave'])<>'{}'::jsonb
           OR regla->>'referencia' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,254}$'
           OR regla->>'huella_sha256' !~ '^[0-9a-f]{64}$'
           OR regla->>'modalidad_clave' !~ '^[a-z][a-z0-9_.-]{1,79}$' THEN
            RAISE EXCEPTION 'CT124: material del centro inválido' USING ERRCODE='22023';
        END IF;
        v_clave:=(m->>'clave_idempotencia')::uuid;
        v_fecha:=(m->>'fecha_incorporacion')::date;
        IF to_char(v_fecha,'YYYY-MM-DD')<>m->>'fecha_incorporacion' THEN RAISE data_exception; END IF;
    EXCEPTION WHEN data_exception THEN
        RAISE EXCEPTION 'CT124: material del centro inválido' USING ERRCODE='22023';
    END;
    SELECT * INTO STRICT v_consumo FROM vec_contratacion_temporal.consumir_incorporacion_centro_ct124('contratacion_temporal.incorporacion.confirmar_centro',
        m->>'expediente_ref',a,v_org,p_material,
        p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
    PERFORM pg_advisory_xact_lock(hashtextextended('vec:incorporacion-centro:clave:'||(m->>'clave_idempotencia'),0));
    PERFORM pg_advisory_xact_lock(hashtextextended('vec:incorporacion-centro:expediente:'||(m->>'expediente_ref'),0));
    SELECT * INTO v_previa FROM vec_contratacion_temporal.incorporacion_centro_v1 WHERE clave_idempotencia=v_clave;
    IF FOUND THEN
        IF v_previa.actor_ref IS DISTINCT FROM a->>'actor_ref' OR v_previa.perfil_ref IS DISTINCT FROM a->>'perfil_ref'
           OR v_previa.centro_ref IS DISTINCT FROM a->>'centro_ref' OR v_previa.puesto_ref IS DISTINCT FROM a->>'puesto_ref' THEN
            RAISE EXCEPTION 'CT124: confirmación ajena' USING ERRCODE='42501';
        ELSIF v_previa.material IS DISTINCT FROM p_material THEN
            RAISE EXCEPTION 'CT124: clave usada con otro contenido' USING ERRCODE='P0681';
        END IF;
        RETURN v_previa.recibo_json||jsonb_build_object('estado_local','replay_confirmado');
    END IF;
    SELECT * INTO v_exp FROM vec_contratacion_temporal.expedientes_centro_ct124(a,v_org) e WHERE e.expediente_ref=m->>'expediente_ref';
    IF NOT FOUND OR v_exp.peticion_ref IS DISTINCT FROM m->>'peticion_ref'
       OR v_exp.agregado->>'fase_actual' IS DISTINCT FROM 'nombramiento'
       OR v_exp.agregado->>'estado_actual' IS DISTINCT FROM 'en_curso'
       OR v_exp.agregado#>>'{analisis,modalidad_clave}' IS DISTINCT FROM regla->>'modalidad_clave'
       OR EXISTS (SELECT 1 FROM vec_contratacion_temporal.incorporacion_centro_v1 WHERE expediente_ref=m->>'expediente_ref') THEN
        RAISE EXCEPTION 'CT124: el expediente no admite la confirmación del centro' USING ERRCODE='P0682';
    END IF;
    v_ahora:=date_trunc('microseconds',clock_timestamp());
    IF v_fecha>(v_ahora AT TIME ZONE 'Europe/Madrid')::date THEN
        RAISE EXCEPTION 'CT124: la incorporación no puede ser futura' USING ERRCODE='22023';
    END IF;
    v_ref:='confirmacion:incorporacion-centro:'||gen_random_uuid()::text;
    v_evento:='evento:incorporacion-centro:'||gen_random_uuid()::text;
    v_recibo:=jsonb_build_object('recibo_ref','recibo:incorporacion-centro:'||gen_random_uuid()::text,'confirmacion_ref',v_ref,
        'peticion_ref',m->>'peticion_ref','expediente_ref',m->>'expediente_ref','numero_visible',v_exp.numero_visible,
        'fecha_incorporacion',m->>'fecha_incorporacion','documento_tipo',doc->>'tipo','actor_ref',a->>'actor_ref',
        'registrado_en',to_char(v_ahora,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'estado_local','registrado');
    SELECT secuencia_outbox,cabeza_outbox_sha256 INTO STRICT v_secuencia,v_anterior
      FROM vec_contratacion_temporal.control_cadenas_expediente_integral WHERE control_id FOR UPDATE;
    IF v_secuencia>=9007199254740991 THEN RAISE EXCEPTION 'CT124: límite de outbox alcanzado' USING ERRCODE='22003'; END IF;
    v_secuencia:=v_secuencia+1;
    -- Evento `ct.incorporacion-confirmada-centro.v1`: referencias, fecha y
    -- tipo de documento; ni el contenido ni la persona.
    v_payload:=convert_to(jsonb_build_object('esquema','vec.contratacion-temporal.incorporacion-confirmada-centro.v1',
        'organizacion_ref',v_org,'expediente_ref',m->>'expediente_ref','version_expediente',v_exp.version,
        'peticion_ref',m->>'peticion_ref','fecha_incorporacion',m->>'fecha_incorporacion','documento_tipo',doc->>'tipo',
        'documento_sha256',doc->>'sha256','recibo_ref',v_recibo->>'recibo_ref','registrada_en',v_recibo->>'registrado_en')::text,'UTF8');
    INSERT INTO vec_contratacion_temporal.outbox_expediente_integral(
        evento_ref,secuencia,operacion_ref,expediente_ref,version_expediente,tipo_evento,payload_canonico,
        payload_huella_sha256,anterior_sha256,huella_sha256,registrada_en)
    VALUES(v_evento,v_secuencia,v_ref,m->>'expediente_ref',v_exp.version,'ct.incorporacion-confirmada-centro.v1',v_payload,
        encode(sha256(v_payload),'hex'),v_anterior,encode(sha256(v_anterior::bytea||v_payload),'hex'),v_ahora);
    UPDATE vec_contratacion_temporal.control_cadenas_expediente_integral
       SET secuencia_outbox=v_secuencia,cabeza_outbox_sha256=encode(sha256(v_anterior::bytea||v_payload),'hex'),actualizada_en=v_ahora
     WHERE control_id;
    INSERT INTO vec_contratacion_temporal.incorporacion_centro_v1(confirmacion_ref,clave_idempotencia,material,material_sha256,organizacion_ref,
        peticion_ref,expediente_ref,version_expediente,actor_ref,perfil_ref,centro_ref,puesto_ref,modalidad_clave,fecha_incorporacion,
        documento_tipo,documento_ref,documento_sha256,regla_ref,regla_huella_sha256,recibo_ref,recibo_json,evento_ref,
        auditoria_ref,decision_ref,consumo_huella_sha256,registrada_en)
    VALUES(v_ref,v_clave,p_material,encode(sha256(convert_to(p_material,'UTF8')),'hex'),v_org,m->>'peticion_ref',m->>'expediente_ref',
        v_exp.version,a->>'actor_ref',a->>'perfil_ref',a->>'centro_ref',a->>'puesto_ref',regla->>'modalidad_clave',v_fecha,
        doc->>'tipo',doc->>'referencia',doc->>'sha256',regla->>'referencia',regla->>'huella_sha256',v_recibo->>'recibo_ref',v_recibo,v_evento,
        v_consumo.auditoria_ref,v_consumo.decision_ref,v_consumo.consumo_huella_sha256,v_ahora);
    RETURN v_recibo;
END
$funcion$;

-- ============================================================ NO INCORPORACIÓN
-- Duda 12 de RRHH (respuesta de ejemplo): «no incorporación = baja y
-- siguiente». RRHH registra, con la resolución (referencia y huella) y la
-- segunda persona que exija el catálogo (c22), que la persona aceptada no se
-- incorpora. En una transacción: versión y actuación del expediente, que
-- vuelve a la fiscalización en curso (la necesidad sigue fiscalizada y
-- pendiente de cubrir), consumo de AD3-88, evento `ct.no-incorporacion.v1` en
-- el outbox y la intención de siguiente candidato (mismo esquema que CT59),
-- que continúa el llamamiento como tras una renuncia (CT119, ampliada abajo).
-- La baja la aplica Bolsa al recibir el evento (Bolsa 000042).
CREATE TABLE vec_contratacion_temporal.no_incorporacion_v1 (
    ambito_hmac text PRIMARY KEY CHECK (ambito_hmac ~ '^hmac-sha256:vec[.]contratacion-temporal[.]no-incorporacion[.]ambito/v[1-9][0-9]{0,8}:[0-9a-f]{64}$'),
    huella_peticion_hmac text NOT NULL CHECK (huella_peticion_hmac ~ '^hmac-sha256:vec[.]contratacion-temporal[.]no-incorporacion[.]peticion/v[1-9][0-9]{0,8}:[0-9a-f]{64}$'),
    organizacion_ref text NOT NULL,
    expediente_ref text NOT NULL,
    version_esperada numeric(20,0) NOT NULL CHECK (version_esperada BETWEEN 1 AND 9007199254740990),
    actor_ref text NOT NULL,
    perfil_ref text NOT NULL,
    motivo_clave text NOT NULL CHECK (motivo_clave ~ '^[a-z][a-z0-9_]{1,63}$'),
    consecuencia_clave text NOT NULL CHECK (consecuencia_clave ~ '^[a-z0-9][a-z0-9._-]{0,127}$'),
    resolucion_ref text NOT NULL CHECK (vec_contratacion_temporal.referencia_valida_ct115(resolucion_ref)),
    resolucion_sha256 text NOT NULL CHECK (resolucion_sha256 ~ '^[0-9a-f]{64}$' AND resolucion_sha256<>repeat('0',64)),
    resuelta_por text NOT NULL CHECK (vec_contratacion_temporal.referencia_valida_ct115(resuelta_por)),
    segunda_persona boolean NOT NULL,
    fecha_notificacion date NOT NULL CHECK (isfinite(fecha_notificacion)),
    observaciones text NOT NULL CHECK (vec_contratacion_temporal.texto_valido_ct115(observaciones,2000,true)),
    aceptacion_resolucion_ref text NOT NULL UNIQUE REFERENCES vec_contratacion_temporal.resolucion_manual_respuesta_rrhh(resolucion_ref),
    propuesta_ref text NOT NULL UNIQUE REFERENCES vec_contratacion_temporal.propuesta_formalizacion(propuesta_ref),
    llamamiento_ref text NOT NULL CHECK (vec_contratacion_temporal.referencia_valida_ct115(llamamiento_ref)),
    intencion_ref text NOT NULL UNIQUE CHECK (vec_contratacion_temporal.referencia_valida_ct115(intencion_ref)),
    comando_siguiente_ref text NOT NULL UNIQUE CHECK (vec_contratacion_temporal.referencia_valida_ct115(comando_siguiente_ref)),
    comando_siguiente_json jsonb NOT NULL CHECK (
        vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(comando_siguiente_json,ARRAY['esquema','comando_ref','intencion_ref',
            'organizacion_ref','expediente_ref','llamamiento_ref','justificante_ref','seleccion_clave']) IS TRUE
        AND comando_siguiente_json->>'esquema'='vec.contratacion-temporal.siguiente-candidato.intencion.v1'
        AND comando_siguiente_json->>'comando_ref'=comando_siguiente_ref
        AND comando_siguiente_json->>'intencion_ref'=intencion_ref
        AND comando_siguiente_json->>'organizacion_ref'=organizacion_ref
        AND comando_siguiente_json->>'expediente_ref'=expediente_ref
        AND comando_siguiente_json->>'llamamiento_ref'=llamamiento_ref),
    estado text NOT NULL CHECK (estado='registrada'),
    reserva_ref text NOT NULL UNIQUE,
    recibo_ref text NOT NULL UNIQUE,
    evento_ref text NOT NULL UNIQUE,
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
    registrada_en timestamptz(6) NOT NULL CHECK (isfinite(registrada_en)),
    confirmada_en timestamptz(6) NOT NULL CHECK (confirmada_en>=registrada_en),
    -- Posición de publicación a Bolsa (como CT113): la transacción que la
    -- escribió; la lectura solo publica transacciones ya terminadas.
    transaccion_publicacion xid8 NOT NULL DEFAULT pg_current_xact_id(),
    CHECK (NOT segunda_persona OR resuelta_por<>actor_ref),
    UNIQUE (expediente_ref,version_esperada),
    FOREIGN KEY (expediente_ref,version_esperada) REFERENCES vec_contratacion_temporal.expediente_version_integral
);
CREATE INDEX no_incorporacion_v1_publicacion ON vec_contratacion_temporal.no_incorporacion_v1 (transaccion_publicacion, evento_ref);
ALTER TABLE vec_contratacion_temporal.no_incorporacion_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contratacion_temporal.no_incorporacion_v1 FORCE ROW LEVEL SECURITY;
-- Lectura: solo desde una sesión del ejecutor (fachadas definidoras de CT: la
-- propia operación, la continuación de CT119, la publicación y la consulta).
CREATE POLICY lectura_no_incorporacion_ct124 ON vec_contratacion_temporal.no_incorporacion_v1 FOR SELECT TO vec_contratacion_temporal_propietario USING (
    pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
    AND NOT pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
    AND NOT pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER'));
-- Comprobación de origen de Bolsa: solo la fila nombrada, y solo dentro de
-- no_incorporacion_publicada_bolsa_v1 (que fija y vacía la opción).
CREATE POLICY existencia_no_incorporacion_ct124 ON vec_contratacion_temporal.no_incorporacion_v1 FOR SELECT TO vec_contratacion_temporal_propietario USING (
    evento_ref=current_setting('vec.ct124.existencia_origen_ref',true));
CREATE POLICY escritura_no_incorporacion_ct124 ON vec_contratacion_temporal.no_incorporacion_v1 FOR INSERT TO vec_contratacion_temporal_propietario WITH CHECK (
    organizacion_ref=current_setting('vec.ct115.organizacion_ref',true)
    AND expediente_ref=current_setting('vec.ct115.expediente_ref',true)
    AND actor_ref=current_setting('vec.ct115.actor_ref',true)
    AND perfil_ref=current_setting('vec.ct115.perfil_ref',true)
    AND ambito_hmac=current_setting('vec.ct115.ambito_hmac',true)
    AND pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
    AND NOT pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
    AND NOT pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER'));
CREATE TRIGGER no_incorporacion_v1_inmutable BEFORE UPDATE OR DELETE ON vec_contratacion_temporal.no_incorporacion_v1
    FOR EACH ROW EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();

-- Material de la no incorporación: forma exacta y tipos. Devuelve la fecha
-- de notificación de la resolución.
CREATE FUNCTION vec_contratacion_temporal.validar_material_no_incorporacion_ct124(m jsonb)
RETURNS date LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog AS $$
DECLARE k text; f date;
BEGIN
    IF m IS NULL OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(m,ARRAY['organizacion_ref','expediente_ref','version_esperada',
        'actor_ref','perfil_ref','motivo_clave','consecuencia_clave','resolucion_ref','resolucion_sha256','resuelta_por','segunda_persona',
        'fecha_notificacion','observaciones']) IS NOT TRUE
       OR EXISTS (SELECT 1 FROM jsonb_each(m) c WHERE jsonb_typeof(c.value) IS DISTINCT FROM
            CASE c.key WHEN 'version_esperada' THEN 'number' WHEN 'segunda_persona' THEN 'boolean' ELSE 'string' END)
       OR m->>'version_esperada' !~ '^[1-9][0-9]{0,15}$' OR (m->>'version_esperada')::numeric>9007199254740990
       OR m->>'motivo_clave' !~ '^[a-z][a-z0-9_]{1,63}$'
       OR m->>'consecuencia_clave' !~ '^[a-z0-9][a-z0-9._-]{0,127}$'
       OR m->>'resolucion_sha256' !~ '^[0-9a-f]{64}$' OR m->>'resolucion_sha256'=repeat('0',64)
       OR m->>'fecha_notificacion' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$'
       OR NOT vec_contratacion_temporal.texto_valido_ct115(m->>'observaciones',2000,true)
       OR ((m->'segunda_persona')::boolean AND m->>'resuelta_por'=m->>'actor_ref') THEN
        RAISE EXCEPTION 'CT124: material de no incorporación inválido' USING ERRCODE='22023';
    END IF;
    FOREACH k IN ARRAY ARRAY['organizacion_ref','expediente_ref','actor_ref','perfil_ref','resolucion_ref','resuelta_por'] LOOP
        IF NOT vec_contratacion_temporal.referencia_valida_ct115(m->>k) THEN
            RAISE EXCEPTION 'CT124: referencia de no incorporación inválida' USING ERRCODE='22023';
        END IF;
    END LOOP;
    f:=(m->>'fecha_notificacion')::date;
    IF to_char(f,'YYYY-MM-DD')<>m->>'fecha_notificacion' THEN
        RAISE EXCEPTION 'CT124: fecha de no incorporación inválida' USING ERRCODE='22023';
    END IF;
    RETURN f;
EXCEPTION WHEN invalid_datetime_format OR datetime_field_overflow THEN
    RAISE EXCEPTION 'CT124: fecha de no incorporación inválida' USING ERRCODE='22023';
END
$$;

-- Aceptación vigente del expediente: la de la propuesta de nombramiento más
-- reciente, confirmada por RRHH y todavía sin continuación.
CREATE FUNCTION vec_contratacion_temporal.aceptacion_vigente_ct124(p_organizacion text, p_expediente text)
RETURNS TABLE(propuesta_ref text, llamamiento_ref text, resolucion_ref text, justificante_ref text, seleccion_clave text, resuelta_en timestamptz)
LANGUAGE sql STABLE SET search_path=pg_catalog AS $$
    SELECT pf.propuesta_ref, pf.llamamiento_ref, r.resolucion_ref, r.justificante_ref, r.seleccion_clave::text, r.resuelta_en
      FROM vec_contratacion_temporal.propuesta_formalizacion pf
      JOIN vec_contratacion_temporal.resolucion_manual_respuesta_rrhh r ON r.resolucion_ref=pf.resolucion_ref
     WHERE pf.organizacion_ref=p_organizacion AND pf.expediente_ref=p_expediente
       AND r.organizacion_ref=p_organizacion AND r.expediente_ref=p_expediente AND r.llamamiento_ref=pf.llamamiento_ref
       AND r.estado='confirmado' AND r.solicitud_json->>'Respuesta'='aceptacion'
       AND r.justificante_ref IS NOT NULL AND r.continuacion_clave IS NULL
     ORDER BY pf.confirmada_en DESC, pf.propuesta_ref COLLATE "C" DESC LIMIT 1
$$;

-- Intención de siguiente candidato derivada del ámbito de idempotencia: la
-- misma petición produce siempre las mismas referencias.
CREATE FUNCTION vec_contratacion_temporal.intencion_no_incorporacion_ct124(p_ambito text, p_organizacion text, p_expediente text,
    p_llamamiento text, p_justificante text, p_seleccion text)
RETURNS jsonb LANGUAGE sql IMMUTABLE SET search_path=pg_catalog AS $$
    SELECT jsonb_build_object('esquema','vec.contratacion-temporal.siguiente-candidato.intencion.v1',
        'comando_ref','comando:ct124:'||encode(sha256(convert_to('comando'||chr(31)||p_ambito,'UTF8')),'hex'),
        'intencion_ref','intencion:ct124:'||encode(sha256(convert_to('intencion'||chr(31)||p_ambito,'UTF8')),'hex'),
        'organizacion_ref',p_organizacion,'expediente_ref',p_expediente,'llamamiento_ref',p_llamamiento,
        'justificante_ref',p_justificante,'seleccion_clave',p_seleccion)
$$;

CREATE FUNCTION vec_contratacion_temporal.resultado_no_incorporacion_ct124(r vec_contratacion_temporal.no_incorporacion_v1)
RETURNS jsonb LANGUAGE sql STABLE SET search_path=pg_catalog AS $$
    SELECT jsonb_build_object('esquema','vec.contratacion-temporal.resultado-no-incorporacion.v1','resultado','confirmada',
        'material',jsonb_build_object('organizacion_ref',r.organizacion_ref,'expediente_ref',r.expediente_ref,
            'version_esperada',r.version_esperada,'actor_ref',r.actor_ref,'perfil_ref',r.perfil_ref,'motivo_clave',r.motivo_clave,
            'consecuencia_clave',r.consecuencia_clave,'resolucion_ref',r.resolucion_ref,'resolucion_sha256',r.resolucion_sha256,
            'resuelta_por',r.resuelta_por,'segunda_persona',r.segunda_persona,
            'fecha_notificacion',to_char(r.fecha_notificacion,'YYYY-MM-DD'),'observaciones',r.observaciones),
        'expediente',r.expediente_siguiente_json,
        'referencias',jsonb_build_object('reserva_ref',r.reserva_ref,'recibo_ref',r.recibo_ref,'evento_ref',r.evento_ref),
        'aceptacion',jsonb_build_object('resolucion_ref',r.aceptacion_resolucion_ref),
        'ambito_idempotencia_hmac',r.ambito_hmac,'huella_peticion_hmac',r.huella_peticion_hmac,'recibo',r.recibo_json)
$$;

CREATE FUNCTION vec_contratacion_temporal.preparar_no_incorporacion_v1(p_operacion jsonb)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security='on' SET timezone='UTC' SET lock_timeout='2s'
AS $funcion$
DECLARE
    m jsonb:=p_operacion->'material'; v_pares jsonb; v_par jsonb; v_fecha date;
    r vec_contratacion_temporal.no_incorporacion_v1%ROWTYPE; v_actual record; v_acept record;
    e text:='vec.contratacion-temporal.resultado-no-incorporacion.v1';
BEGIN
    PERFORM vec_contratacion_temporal.exigir_sesion_ct115(true);
    v_pares:=vec_contratacion_temporal.validar_preparacion_ct115(p_operacion,'vec.contratacion-temporal.preparar-no-incorporacion.v1','registrar_no_incorporacion','no-incorporacion');
    v_fecha:=vec_contratacion_temporal.validar_material_no_incorporacion_ct124(m);
    PERFORM set_config('vec.ct115.organizacion_ref',m->>'organizacion_ref',true);
    PERFORM set_config('vec.ct115.expediente_ref',m->>'expediente_ref',true);
    FOR v_par IN SELECT value FROM jsonb_array_elements(v_pares) LOOP
        SELECT * INTO r FROM vec_contratacion_temporal.no_incorporacion_v1 WHERE ambito_hmac=v_par->>'ambito_hmac';
        IF FOUND THEN
            IF r.huella_peticion_hmac IS DISTINCT FROM v_par->>'huella_peticion_hmac' OR r.organizacion_ref<>m->>'organizacion_ref'
               OR r.expediente_ref<>m->>'expediente_ref' OR r.version_esperada<>(m->>'version_esperada')::numeric
               OR r.actor_ref<>m->>'actor_ref' OR r.perfil_ref<>m->>'perfil_ref' OR r.motivo_clave<>m->>'motivo_clave'
               OR r.consecuencia_clave<>m->>'consecuencia_clave' OR r.resolucion_ref<>m->>'resolucion_ref'
               OR r.resolucion_sha256<>m->>'resolucion_sha256' OR r.resuelta_por<>m->>'resuelta_por'
               OR r.segunda_persona<>(m->'segunda_persona')::boolean OR r.fecha_notificacion<>v_fecha OR r.observaciones<>m->>'observaciones' THEN
                RETURN jsonb_build_object('esquema',e,'resultado','idempotencia_reutilizada');
            END IF;
            RETURN vec_contratacion_temporal.resultado_no_incorporacion_ct124(r);
        END IF;
    END LOOP;
    IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.no_incorporacion_v1 n
                WHERE n.organizacion_ref=m->>'organizacion_ref' AND n.expediente_ref=m->>'expediente_ref'
                  AND n.propuesta_ref=(SELECT a.propuesta_ref FROM vec_contratacion_temporal.propuesta_formalizacion a
                                        WHERE a.organizacion_ref=m->>'organizacion_ref' AND a.expediente_ref=m->>'expediente_ref')) THEN
        RETURN jsonb_build_object('esquema',e,'resultado','no_incorporacion_existente');
    END IF;
    SELECT a.version, v.agregado_json INTO v_actual
      FROM vec_contratacion_temporal.expediente_integral_actual a
      JOIN vec_contratacion_temporal.expediente_version_integral v USING (expediente_ref,version)
     WHERE a.expediente_ref=m->>'expediente_ref';
    IF NOT FOUND OR v_actual.version<>(m->>'version_esperada')::numeric
       OR v_actual.agregado_json->>'organizacion_ref' IS DISTINCT FROM m->>'organizacion_ref'
       OR v_actual.agregado_json->>'fase_actual' IS DISTINCT FROM 'nombramiento'
       OR v_actual.agregado_json->>'estado_actual' IS DISTINCT FROM 'en_curso'
       OR NOT vec_contratacion_temporal.referencia_valida_ct115(v_actual.agregado_json#>>'{asignacion,unidad_ref}') THEN
        RETURN jsonb_build_object('esquema',e,'resultado','version_en_conflicto');
    END IF;
    IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.incorporacion_expediente_ct115(m->>'organizacion_ref',m->>'expediente_ref')) THEN
        RETURN jsonb_build_object('esquema',e,'resultado','incorporacion_existente');
    END IF;
    SELECT * INTO v_acept FROM vec_contratacion_temporal.aceptacion_vigente_ct124(m->>'organizacion_ref',m->>'expediente_ref');
    IF NOT FOUND THEN
        RETURN jsonb_build_object('esquema',e,'resultado','sin_aceptacion');
    END IF;
    RETURN jsonb_build_object('esquema',e,'resultado','preparada','material',m,'expediente',v_actual.agregado_json,
        'referencias',p_operacion->'referencias_candidatas',
        'aceptacion',jsonb_build_object('resolucion_ref',v_acept.resolucion_ref),
        'ambito_idempotencia_hmac',p_operacion#>>'{sellos_hmac,activo,ambito_hmac}',
        'huella_peticion_hmac',p_operacion#>>'{sellos_hmac,activo,huella_peticion_hmac}');
EXCEPTION WHEN invalid_text_representation OR datetime_field_overflow OR numeric_value_out_of_range OR character_not_in_repertoire THEN
    RAISE EXCEPTION 'CT124: entrada de no incorporación inválida' USING ERRCODE='22023';
END
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.confirmar_no_incorporacion_v1(
    p_operacion jsonb,
    p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,
    p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security='on' SET timezone='UTC' SET lock_timeout='2s'
AS $funcion$
DECLARE
    m jsonb:=p_operacion->'material'; refs jsonb:=p_operacion->'referencias'; pol jsonb:=p_operacion->'politica'; a jsonb:=p_operacion->'autorizacion';
    e text:='vec.contratacion-temporal.resultado-no-incorporacion.v1';
    r vec_contratacion_temporal.no_incorporacion_v1%ROWTYPE;
    v_fecha date; v_version numeric; v_instante timestamptz; v_ahora timestamptz(6); v_actual record; v_acept record;
    v_decision jsonb; v_consumo record; v_actuacion jsonb; v_siguiente jsonb; v_ambitos jsonb; v_atributos jsonb; v_contexto text;
    v_secuencia_actuacion numeric; v_agregado_huella text; v_prueba bytea; v_payload bytea; v_anterior text; v_secuencia numeric;
    v_recibo jsonb; v_comando jsonb; v_restriccion text; v_tabla text; v_esquema text;
BEGIN
    PERFORM vec_contratacion_temporal.exigir_sesion_ct115(false);
    PERFORM vec_contratacion_temporal.validar_confirmacion_ct115(p_operacion,'vec.contratacion-temporal.confirmar-no-incorporacion.v1','registrar_no_incorporacion',
        'no-incorporacion','contratacion_temporal.incorporacion.no_incorporacion','registrar_no_incorporacion_contratacion_temporal',p_decision,p_motivo,p_persona_version,p_perfil_version);
    v_fecha:=vec_contratacion_temporal.validar_material_no_incorporacion_ct124(m);
    IF p_capacidad IS NULL OR p_contexto IS NULL OR p_payload IS NULL OR p_sobre IS NULL OR p_evidencia IS NULL OR p_raiz IS NULL THEN
        RAISE EXCEPTION 'CT124: autorización incompleta' USING ERRCODE='42501';
    END IF;
    v_version:=(m->>'version_esperada')::numeric;
    v_instante:=(p_operacion->>'instante_efecto')::timestamptz;
    v_decision:=convert_from(p_decision,'UTF8')::jsonb;
    PERFORM set_config('vec.ct115.organizacion_ref',m->>'organizacion_ref',true);
    PERFORM set_config('vec.ct115.expediente_ref',m->>'expediente_ref',true);
    PERFORM set_config('vec.ct115.actor_ref',m->>'actor_ref',true);
    PERFORM set_config('vec.ct115.perfil_ref',m->>'perfil_ref',true);
    PERFORM set_config('vec.ct115.ambito_hmac',p_operacion->>'ambito_idempotencia_hmac',true);
    SELECT * INTO r FROM vec_contratacion_temporal.no_incorporacion_v1 WHERE ambito_hmac=p_operacion->>'ambito_idempotencia_hmac';
    IF FOUND THEN
        IF r.huella_peticion_hmac IS DISTINCT FROM p_operacion->>'huella_peticion_hmac' OR r.version_esperada<>v_version
           OR r.motivo_clave<>m->>'motivo_clave' OR r.consecuencia_clave<>m->>'consecuencia_clave' OR r.resolucion_ref<>m->>'resolucion_ref'
           OR r.resolucion_sha256<>m->>'resolucion_sha256' OR r.resuelta_por<>m->>'resuelta_por' OR r.segunda_persona<>(m->'segunda_persona')::boolean
           OR r.fecha_notificacion<>v_fecha OR r.observaciones<>m->>'observaciones' THEN
            RETURN jsonb_build_object('esquema',e,'resultado','idempotencia_reutilizada');
        END IF;
        IF r.decision_ref IS DISTINCT FROM a->>'decision_ref' THEN
            RAISE EXCEPTION 'CT124: evidencia de repetición divergente' USING ERRCODE='42501';
        END IF;
        RETURN jsonb_build_object('esquema',e,'resultado','confirmada','recibo',r.recibo_json);
    END IF;
    SELECT v.* INTO v_actual FROM vec_contratacion_temporal.expediente_integral_actual ac
      JOIN vec_contratacion_temporal.expediente_version_integral v USING (expediente_ref,version)
     WHERE ac.expediente_ref=m->>'expediente_ref' FOR UPDATE OF ac,v;
    IF NOT FOUND OR v_actual.version<>v_version
       OR v_actual.agregado_json IS DISTINCT FROM p_operacion->'expediente_anterior'
       OR v_actual.agregado_json->>'organizacion_ref' IS DISTINCT FROM m->>'organizacion_ref'
       OR v_actual.agregado_json->>'fase_actual' IS DISTINCT FROM 'nombramiento'
       OR v_actual.agregado_json->>'estado_actual' IS DISTINCT FROM 'en_curso'
       OR jsonb_typeof(v_actual.agregado_json->'actuaciones') IS DISTINCT FROM 'array'
       OR NOT vec_contratacion_temporal.referencia_valida_ct115(v_actual.agregado_json#>>'{asignacion,unidad_ref}') THEN
        RETURN jsonb_build_object('esquema',e,'resultado','version_en_conflicto');
    END IF;
    IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.incorporacion_expediente_ct115(m->>'organizacion_ref',m->>'expediente_ref')) THEN
        RETURN jsonb_build_object('esquema',e,'resultado','incorporacion_existente');
    END IF;
    SELECT * INTO v_acept FROM vec_contratacion_temporal.aceptacion_vigente_ct124(m->>'organizacion_ref',m->>'expediente_ref');
    IF NOT FOUND THEN
        RETURN jsonb_build_object('esquema',e,'resultado','sin_aceptacion');
    END IF;
    -- Bloquea la resolución de aceptación: su continuación la ocupa CT119.
    PERFORM 1 FROM vec_contratacion_temporal.resolucion_manual_respuesta_rrhh WHERE resolucion_ref=v_acept.resolucion_ref FOR UPDATE;
    IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.no_incorporacion_v1 WHERE aceptacion_resolucion_ref=v_acept.resolucion_ref) THEN
        RETURN jsonb_build_object('esquema',e,'resultado','no_incorporacion_existente');
    END IF;
    IF v_fecha>(v_instante AT TIME ZONE 'Europe/Madrid')::date OR v_instante<v_acept.resuelta_en THEN
        RETURN jsonb_build_object('esquema',e,'resultado','fecha_no_admitida');
    END IF;
    v_secuencia_actuacion:=jsonb_array_length(v_actual.agregado_json->'actuaciones')+1;
    v_actuacion:=jsonb_build_object('secuencia',v_secuencia_actuacion,'version_expediente',v_version+1,
        'accion_clave','contratacion_temporal.incorporacion.no_incorporacion','actor_ref',m->>'actor_ref',
        'unidad_ref',v_actual.agregado_json#>>'{asignacion,unidad_ref}','recibo_ref',refs->>'recibo_ref',
        'realizada_en',p_operacion->'instante_efecto','fase_origen','nombramiento','fase_destino','fiscalizacion',
        'estado_origen','en_curso','estado_destino','en_curso','documentos_ref',jsonb_build_array(m->>'resolucion_ref'));
    IF m->>'observaciones'<>'' THEN
        v_actuacion:=v_actuacion||jsonb_build_object('observaciones',m->>'observaciones');
    END IF;
    v_siguiente:=v_actual.agregado_json||jsonb_build_object('version',v_version+1,'actualizado_en',p_operacion->'instante_efecto',
        'fase_actual','fiscalizacion','actuaciones',(v_actual.agregado_json->'actuaciones')||jsonb_build_array(v_actuacion));
    IF v_instante<(v_actual.agregado_json->>'actualizado_en')::timestamptz
       OR v_actuacion IS DISTINCT FROM p_operacion->'actuacion'
       OR v_siguiente IS DISTINCT FROM p_operacion->'expediente_siguiente' THEN
        RAISE EXCEPTION 'CT124: proyección de no incorporación divergente' USING ERRCODE='22023';
    END IF;
    v_ambitos:=jsonb_build_object('organizacion_ref',m->>'organizacion_ref','expediente_ref',m->>'expediente_ref',
        'fase_previa','nombramiento','estado_previo','en_curso');
    v_atributos:=jsonb_build_object('version_expediente',v_version::text,'motivo_clave',m->>'motivo_clave',
        'consecuencia_clave',m->>'consecuencia_clave','resolucion_ref',m->>'resolucion_ref','resolucion_sha256',m->>'resolucion_sha256',
        'resuelta_por',m->>'resuelta_por','segunda_persona',CASE WHEN (m->'segunda_persona')::boolean THEN 'si' ELSE 'no' END,
        'fecha_notificacion',m->>'fecha_notificacion',
        'observaciones_huella_sha256',encode(sha256(convert_to(m->>'observaciones','UTF8')),'hex'),
        'aceptacion_ref',v_acept.resolucion_ref,'politica_ref',pol->>'definicion_ref','politica_version',pol->>'definicion_version',
        'politica_huella_sha256',pol->>'definicion_huella_sha256','ambito_idempotencia_hmac',p_operacion->>'ambito_idempotencia_hmac',
        'huella_peticion_hmac',p_operacion->>'huella_peticion_hmac');
    IF p_operacion#>'{contexto,ambitos}' IS DISTINCT FROM v_ambitos OR p_operacion#>'{contexto,atributos}' IS DISTINCT FROM v_atributos THEN
        RAISE EXCEPTION 'CT124: contexto de no incorporación divergente' USING ERRCODE='42501';
    END IF;
    v_contexto:=vec_contratacion_temporal.huella_contexto_go_ct115(v_ambitos,v_atributos);
    IF a->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_contexto
       OR v_decision->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_contexto
       OR v_decision->>'principal_id' IS DISTINCT FROM m->>'actor_ref'
       OR v_decision->>'perfil_activo_ref' IS DISTINCT FROM m->>'perfil_ref'
       OR v_decision->>'recurso_ref' IS DISTINCT FROM m->>'expediente_ref'
       OR v_decision->>'accion' IS DISTINCT FROM 'contratacion_temporal.incorporacion.no_incorporacion'
       OR v_decision->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
       OR v_decision->>'tipo_recurso' IS DISTINCT FROM 'no_incorporacion_contratacion_temporal'
       OR v_decision->>'finalidad' IS DISTINCT FROM 'registrar_no_incorporacion_contratacion_temporal'
       OR v_decision->>'decision_ref' IS DISTINCT FROM a->>'decision_ref' THEN
        RAISE EXCEPTION 'CT124: contexto autorizado de no incorporación divergente' USING ERRCODE='42501';
    END IF;
    v_ahora:=date_trunc('microseconds',clock_timestamp());
    IF (pol->>'evaluada_en')::timestamptz>v_instante OR v_ahora<v_instante OR v_ahora>=(pol->>'valida_hasta')::timestamptz THEN
        RAISE EXCEPTION 'CT124: vigencia de no incorporación agotada' USING ERRCODE='42501';
    END IF;
    SELECT * INTO STRICT v_consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_no_incorporacion_ct_v3_atestada(
        p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
    IF v_consumo.decision_ref IS DISTINCT FROM a->>'decision_ref' OR v_consumo.efecto_ref IS DISTINCT FROM m->>'expediente_ref'
       OR v_consumo.huella_efecto_sha256 IS DISTINCT FROM v_contexto OR coalesce(v_consumo.auditoria_ref,'') !~ '^aud_v3_[0-9a-f]{32}$'
       OR v_consumo.consumo_nuevo IS NOT TRUE THEN
        RAISE EXCEPTION 'CT124: consumo de no incorporación divergente' USING ERRCODE='42501';
    END IF;
    v_ahora:=date_trunc('microseconds',clock_timestamp());
    IF v_ahora<v_instante OR v_ahora>=(pol->>'valida_hasta')::timestamptz THEN
        RAISE EXCEPTION 'CT124: vigencia final de no incorporación agotada' USING ERRCODE='42501';
    END IF;
    v_comando:=vec_contratacion_temporal.intencion_no_incorporacion_ct124(p_operacion->>'ambito_idempotencia_hmac',m->>'organizacion_ref',
        m->>'expediente_ref',v_acept.llamamiento_ref,v_acept.justificante_ref,v_acept.seleccion_clave);
    v_agregado_huella:=encode(sha256(convert_to(v_siguiente::text,'UTF8')),'hex');
    v_prueba:=convert_to('VEC-CT-EXPEDIENTE-NO-INCORPORACION-CT124'||chr(10)||(m->>'expediente_ref')||chr(10)||(v_version+1)::text||chr(10)
        ||v_agregado_huella||chr(10)||(refs->>'reserva_ref')||chr(10)||(refs->>'recibo_ref')||chr(10)||v_consumo.decision_ref||chr(10)||v_ahora::text,'UTF8');
    INSERT INTO vec_contratacion_temporal.expediente_version_integral(
        expediente_ref,version,agregado_json,agregado_json_huella_sha256,prueba_canonica,prueba_huella_sha256,
        flujo_ref,flujo_version,flujo_huella_sha256,fase_clave,estado,origen_version,operacion_ref,registrada_en)
    VALUES(m->>'expediente_ref',v_version+1,v_siguiente,v_agregado_huella,v_prueba,encode(sha256(v_prueba),'hex'),
        v_actual.flujo_ref,v_actual.flujo_version,v_actual.flujo_huella_sha256,'fiscalizacion','en_curso',
        'no_incorporacion_ct124',refs->>'reserva_ref',v_ahora);
    UPDATE vec_contratacion_temporal.expediente_integral_actual SET version=v_version+1,actualizada_en=v_ahora,operacion_ref=refs->>'reserva_ref'
     WHERE expediente_ref=m->>'expediente_ref' AND version=v_version;
    IF NOT FOUND THEN RAISE EXCEPTION 'CT124: CAS final de no incorporación perdido' USING ERRCODE='40001'; END IF;
    v_prueba:=convert_to('VEC-CT-ACTUACION-NO-INCORPORACION-CT124'||chr(10)||encode(sha256(convert_to(v_actuacion::text,'UTF8')),'hex')||chr(10)
        ||(refs->>'recibo_ref')||chr(10)||v_ahora::text,'UTF8');
    INSERT INTO vec_contratacion_temporal.actuacion_expediente_integral(
        expediente_ref,secuencia,version_expediente,operacion_ref,recibo_ref,actuacion_json,
        actuacion_json_huella_sha256,prueba_canonica,prueba_huella_sha256,registrada_en)
    VALUES(m->>'expediente_ref',v_secuencia_actuacion,v_version+1,refs->>'reserva_ref',refs->>'recibo_ref',v_actuacion,
        encode(sha256(convert_to(v_actuacion::text,'UTF8')),'hex'),v_prueba,encode(sha256(v_prueba),'hex'),v_ahora);
    SELECT secuencia_outbox,cabeza_outbox_sha256 INTO STRICT v_secuencia,v_anterior
      FROM vec_contratacion_temporal.control_cadenas_expediente_integral WHERE control_id FOR UPDATE;
    IF v_secuencia>=9007199254740991 THEN RAISE EXCEPTION 'CT124: límite de outbox alcanzado' USING ERRCODE='22003'; END IF;
    v_secuencia:=v_secuencia+1;
    -- Evento `ct.no-incorporacion.v1`: referencias opacas, claves del
    -- catálogo, la resolución (referencia y huella) y quién la resolvió.
    v_payload:=convert_to(jsonb_build_object('esquema','vec.contratacion-temporal.no-incorporacion.v1','organizacion_ref',m->>'organizacion_ref',
        'expediente_ref',m->>'expediente_ref','version_resultante',v_version+1,'llamamiento_ref',v_acept.llamamiento_ref,
        'motivo_clave',m->>'motivo_clave','consecuencia_clave',m->>'consecuencia_clave','resolucion_ref',m->>'resolucion_ref',
        'resolucion_sha256',m->>'resolucion_sha256','resuelta_por',m->>'resuelta_por','fecha_notificacion',m->>'fecha_notificacion',
        'intencion_ref',v_comando->>'intencion_ref','recibo_ref',refs->>'recibo_ref','registrada_en',p_operacion->'instante_efecto')::text,'UTF8');
    INSERT INTO vec_contratacion_temporal.outbox_expediente_integral(
        evento_ref,secuencia,operacion_ref,expediente_ref,version_expediente,tipo_evento,payload_canonico,
        payload_huella_sha256,anterior_sha256,huella_sha256,registrada_en)
    VALUES(refs->>'evento_ref',v_secuencia,refs->>'reserva_ref',m->>'expediente_ref',v_version+1,'ct.no-incorporacion.v1',v_payload,
        encode(sha256(v_payload),'hex'),v_anterior,encode(sha256(v_anterior::bytea||v_payload),'hex'),v_ahora);
    UPDATE vec_contratacion_temporal.control_cadenas_expediente_integral
       SET secuencia_outbox=v_secuencia,cabeza_outbox_sha256=encode(sha256(v_anterior::bytea||v_payload),'hex'),actualizada_en=v_ahora
     WHERE control_id;
    v_recibo:=jsonb_build_object('operacion','registrar_no_incorporacion','organizacion_ref',m->>'organizacion_ref','expediente_ref',m->>'expediente_ref',
        'version_anterior',v_version,'version_resultante',v_version+1,'fase_resultante','fiscalizacion','estado_resultante','en_curso',
        'causa_clave',m->>'motivo_clave','recibo_ref',refs->>'recibo_ref',
        'auditoria_ref',v_consumo.auditoria_ref,'evento_ref',refs->>'evento_ref','actor_ref',m->>'actor_ref',
        'registrada_en',p_operacion->'instante_efecto');
    INSERT INTO vec_contratacion_temporal.no_incorporacion_v1(
        ambito_hmac,huella_peticion_hmac,organizacion_ref,expediente_ref,version_esperada,actor_ref,perfil_ref,motivo_clave,consecuencia_clave,
        resolucion_ref,resolucion_sha256,resuelta_por,segunda_persona,fecha_notificacion,observaciones,aceptacion_resolucion_ref,propuesta_ref,
        llamamiento_ref,intencion_ref,comando_siguiente_ref,comando_siguiente_json,estado,reserva_ref,recibo_ref,evento_ref,
        expediente_anterior_json,expediente_siguiente_json,recibo_json,decision_ref,decision_huella_sha256,consumo_huella_sha256,auditoria_ref,
        politica_ref,politica_version,politica_huella_sha256,registrada_en,confirmada_en)
    VALUES(p_operacion->>'ambito_idempotencia_hmac',p_operacion->>'huella_peticion_hmac',m->>'organizacion_ref',m->>'expediente_ref',v_version,
        m->>'actor_ref',m->>'perfil_ref',m->>'motivo_clave',m->>'consecuencia_clave',m->>'resolucion_ref',m->>'resolucion_sha256',
        m->>'resuelta_por',(m->'segunda_persona')::boolean,v_fecha,m->>'observaciones',v_acept.resolucion_ref,v_acept.propuesta_ref,
        v_acept.llamamiento_ref,v_comando->>'intencion_ref',v_comando->>'comando_ref',v_comando,'registrada',
        refs->>'reserva_ref',refs->>'recibo_ref',refs->>'evento_ref',v_actual.agregado_json,v_siguiente,v_recibo,v_consumo.decision_ref,
        a->>'decision_huella_sha256',v_consumo.consumo_huella_sha256,v_consumo.auditoria_ref,pol->>'definicion_ref',
        (pol->>'definicion_version')::numeric,pol->>'definicion_huella_sha256',v_instante,v_ahora);
    RETURN jsonb_build_object('esquema',e,'resultado','confirmada','recibo',v_recibo);
EXCEPTION
WHEN unique_violation THEN
    GET STACKED DIAGNOSTICS v_restriccion=CONSTRAINT_NAME,v_tabla=TABLE_NAME,v_esquema=SCHEMA_NAME;
    IF v_esquema='vec_contratacion_temporal' AND v_tabla='no_incorporacion_v1' THEN
        RETURN jsonb_build_object('esquema',e,'resultado',CASE WHEN v_restriccion='no_incorporacion_v1_pkey' THEN 'idempotencia_reutilizada' ELSE 'no_incorporacion_existente' END);
    END IF;
    RAISE;
WHEN invalid_text_representation OR datetime_field_overflow OR numeric_value_out_of_range OR character_not_in_repertoire THEN
    RAISE EXCEPTION 'CT124: entrada de no incorporación inválida' USING ERRCODE='22023';
END
$funcion$;

-- Cuerpo del evento que se publica a Bolsa (bandeja de Bolsa 000042): el
-- mismo para la lectura del relevo y para la comprobación de origen. Lleva
-- referencias opacas, las claves del catálogo y la resolución por referencia
-- y huella; nunca datos de la persona.
CREATE FUNCTION vec_contratacion_temporal.evento_no_incorporacion_bolsa_ct124(n vec_contratacion_temporal.no_incorporacion_v1)
RETURNS jsonb LANGUAGE sql STABLE SET search_path=pg_catalog SET timezone='UTC' AS $$
    SELECT jsonb_build_object('esquema','vec.contratacion-temporal.no-incorporacion-bolsa.v1','tipo','no_incorporacion',
        'evento_ref','evento:ct:no-incorporacion-bolsa:'||encode(sha256(convert_to('no_incorporacion'||chr(31)||n.evento_ref,'UTF8')),'hex'),
        'origen_ref',n.evento_ref,'organizacion_ref',n.organizacion_ref,'expediente_ref',n.expediente_ref,
        'llamamiento_ref',n.llamamiento_ref,'motivo_clave',n.motivo_clave,'consecuencia_clave',n.consecuencia_clave,
        'resolucion_ref',n.resolucion_ref,'resolucion_sha256',n.resolucion_sha256,'resuelta_por',n.resuelta_por,
        'actor_ref',n.actor_ref,'fecha_notificacion',to_char(n.fecha_notificacion,'YYYY-MM-DD'),
        'ocurrido_en',to_char(n.registrada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))
$$;

-- Publicación a Bolsa, como CT113: paginada por (posición, evento) y solo de
-- transacciones ya terminadas.
CREATE FUNCTION vec_contratacion_temporal.leer_no_incorporaciones_bolsa_v1(p_desde_posicion bigint, p_desde_ref text, p_limite integer)
RETURNS TABLE(evento_ref text, evento jsonb, huella_sha256 text, origen_ref text, origen_posicion bigint, origen_creada_en timestamptz)
LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path=pg_catalog SET row_security='on' SET timezone='UTC' AS $f$
BEGIN
    IF current_user<>'vec_contratacion_temporal_propietario' OR session_user=current_user
       OR NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
       OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
       OR pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER') THEN
        RAISE EXCEPTION 'CT124: lectura no autorizada' USING ERRCODE='42501';
    END IF;
    IF p_limite IS NULL OR p_limite NOT BETWEEN 1 AND 100 OR (p_desde_posicion IS NULL)<>(p_desde_ref IS NULL)
       OR p_desde_posicion<0 OR octet_length(p_desde_ref)>512 THEN
        RAISE EXCEPTION 'CT124: lectura de no incorporaciones inválida' USING ERRCODE='22023';
    END IF;
    RETURN QUERY
    WITH base AS (
        SELECT n.evento_ref AS origen, n.confirmada_en, n.transaccion_publicacion::text::bigint AS posicion,
               vec_contratacion_temporal.evento_no_incorporacion_bolsa_ct124(n) AS cuerpo
          FROM vec_contratacion_temporal.no_incorporacion_v1 n
          JOIN vec_contratacion_temporal.outbox_expediente_integral o
            ON o.evento_ref=n.evento_ref AND o.expediente_ref=n.expediente_ref AND o.tipo_evento='ct.no-incorporacion.v1'
         WHERE (n.transaccion_publicacion<pg_snapshot_xmin(pg_current_snapshot())
                OR n.transaccion_publicacion=pg_current_xact_id_if_assigned())
           AND (p_desde_posicion IS NULL OR (n.transaccion_publicacion::text::bigint,n.evento_ref)>(p_desde_posicion,p_desde_ref))
         ORDER BY 3, n.evento_ref
         LIMIT p_limite)
    SELECT b.cuerpo->>'evento_ref', b.cuerpo, encode(sha256(convert_to(b.cuerpo::text,'UTF8')),'hex'),
           b.origen, b.posicion, b.confirmada_en
      FROM base b ORDER BY b.posicion, b.origen;
END
$f$;

-- Comprobación de origen para Bolsa 000042: dice solo si CT publicó ese
-- evento exacto (referencia de origen, huella del cuerpo y posición de
-- publicación). No devuelve datos. La invoca la bandeja de Bolsa (su
-- propietario), que así no depende de la palabra del relevo. La fila se lee
-- con una política propia que solo deja ver la no incorporación nombrada
-- mientras dura esta función (la opción se vacía antes de salir; si falla,
-- la transacción se deshace con ella).
CREATE FUNCTION vec_contratacion_temporal.no_incorporacion_publicada_bolsa_v1(p_origen_ref text, p_huella_sha256 text, p_posicion bigint)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security='on' SET timezone='UTC'
AS $f$
DECLARE v_cuerpo jsonb; v_posicion bigint;
BEGIN
    IF current_user<>'vec_contratacion_temporal_propietario' OR p_origen_ref IS NULL OR p_huella_sha256 IS NULL OR p_posicion IS NULL
       OR octet_length(p_origen_ref) NOT BETWEEN 1 AND 512 OR p_huella_sha256 !~ '^[0-9a-f]{64}$' THEN
        RETURN false;
    END IF;
    PERFORM set_config('vec.ct124.existencia_origen_ref',p_origen_ref,true);
    SELECT vec_contratacion_temporal.evento_no_incorporacion_bolsa_ct124(n), n.transaccion_publicacion::text::bigint
      INTO v_cuerpo, v_posicion
      FROM vec_contratacion_temporal.no_incorporacion_v1 n
      JOIN vec_contratacion_temporal.outbox_expediente_integral o
        ON o.evento_ref=n.evento_ref AND o.expediente_ref=n.expediente_ref AND o.tipo_evento='ct.no-incorporacion.v1'
     WHERE n.evento_ref=p_origen_ref;
    PERFORM set_config('vec.ct124.existencia_origen_ref','',true);
    RETURN v_cuerpo IS NOT NULL AND v_posicion=p_posicion
       AND encode(sha256(convert_to(v_cuerpo::text,'UTF8')),'hex')=p_huella_sha256;
END
$f$;

-- ============================================================ CONTINUACIÓN (CT119)
-- La continuación de CT119 admite, además de la renuncia y la expiración,
-- la aceptación seguida de una no incorporación registrada: el antecedente
-- es la resolución de aceptación y la intención de siguiente candidato la de
-- la no incorporación. CT119, CT121 y las funciones del sucesor no se editan:
-- en cada función viva se sustituye un único fragmento exacto, conservando
-- propietario, configuración y ACL. La propuesta del sucesor tras una no
-- incorporación no se amplía: el expediente ya tiene su propuesta de
-- nombramiento (única por expediente).
CREATE FUNCTION vec_contratacion_temporal.antecedente_no_incorporacion_ct124(p_resolucion text, p_intencion text)
RETURNS jsonb LANGUAGE sql STABLE SET search_path=pg_catalog AS $$
    SELECT jsonb_build_object('ComandoSiguienteRef',n.comando_siguiente_ref,'ComandoSiguiente',n.comando_siguiente_json,
        'NoIncorporacion',jsonb_build_object('ReciboRef',n.recibo_ref,'IntencionRef',n.intencion_ref,
            'ComandoRef',n.comando_siguiente_ref,'VersionResultante',n.version_esperada+1,
            'RegistradaEn',to_char(n.registrada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')))
      FROM vec_contratacion_temporal.no_incorporacion_v1 n
     WHERE n.aceptacion_resolucion_ref=p_resolucion AND n.intencion_ref=p_intencion
$$;

DO $continuacion$
DECLARE
    v_antes record; v_despues record; v_definicion text; v_funcion text; v_anterior text; v_nuevo text; i integer;
    v_firmas text[]; v_viejos text[]; v_nuevos text[];
BEGIN
    -- 1. CT119: antecedente y consulta.
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
    -- 2. Circuito del sucesor (CT62-CT64 y la lectura del aviso, ya
    -- ampliados por CT121): la aceptación seguida de no incorporación.
    FOR v_funcion,v_anterior IN SELECT * FROM (VALUES
        ('registrar_comunicacion_llamamiento_local_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','r'),
        ('registrar_respuesta_recibida_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','r'),
        ('consultar_justificante_respuesta_recibida_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','continuacion'),
        ('registrar_resolucion_manual_respuesta_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','continuacion'),
        ('leer_expediente_aviso_confirmado_v1(text,text,text)','r')) AS x(f,alias) LOOP
        v_firmas:=v_firmas||v_funcion;
        v_viejos:=v_viejos||(v_anterior||$a$.solicitud_json->>'Respuesta' IN ('renuncia','expiracion_gobernada')$a$);
        v_nuevos:=v_nuevos||(v_anterior||$a$.solicitud_json->>'Respuesta' IN ('renuncia','expiracion_gobernada','aceptacion')$a$);
    END LOOP;
    FOR i IN 1..array_length(v_firmas,1) LOOP
        SELECT p.oid,pg_get_functiondef(p.oid) AS definicion,p.proacl AS acl,p.proowner AS propietario,
               p.proconfig AS configuracion,p.prosecdef AS definidor
          INTO v_antes FROM pg_proc p
         WHERE p.oid=to_regprocedure('vec_contratacion_temporal.'||v_firmas[i])
           AND p.proowner='vec_contratacion_temporal_propietario'::regrole AND p.prosecdef;
        IF NOT FOUND THEN
            RAISE EXCEPTION 'CT124: función ausente: %',v_firmas[i] USING ERRCODE='55000';
        END IF;
        v_definicion:=v_antes.definicion;
        IF length(v_definicion)-length(replace(v_definicion,v_viejos[i],''))<>length(v_viejos[i])
           OR strpos(v_definicion,'no_incorporacion')<>0 AND i>2 THEN
            RAISE EXCEPTION 'CT124: preimagen incompatible: %',v_firmas[i] USING ERRCODE='55000';
        END IF;
        v_definicion:=replace(v_definicion,v_viejos[i],v_nuevos[i]);
        EXECUTE v_definicion;
        SELECT pg_get_functiondef(p.oid) AS definicion,p.proacl AS acl,p.proowner AS propietario,
               p.proconfig AS configuracion,p.prosecdef AS definidor
          INTO STRICT v_despues FROM pg_proc p WHERE p.oid=v_antes.oid;
        IF v_despues.definicion IS DISTINCT FROM v_definicion
           OR v_despues.acl IS DISTINCT FROM v_antes.acl
           OR v_despues.propietario IS DISTINCT FROM v_antes.propietario
           OR v_despues.configuracion IS DISTINCT FROM v_antes.configuracion
           OR v_despues.definidor IS NOT TRUE THEN
            RAISE EXCEPTION 'CT124: definición o permisos alterados: %',v_firmas[i] USING ERRCODE='55000';
        END IF;
    END LOOP;
END
$continuacion$;

-- La confirmación de la continuación ocupa las columnas de continuación de la
-- resolución de aceptación: la restricción admite la aceptación sin comando
-- propio (la intención es la de su no incorporación, que CT119 comprueba).
DO $restriccion$
DECLARE v_def text; v_nueva text;
    v_resp text:=$r$((solicitud_json ->> 'Respuesta'::text) = ANY (ARRAY['renuncia'::text, 'expiracion_gobernada'::text])) AND ((octet_length(continuacion_material)$r$;
    v_resp_n text:=$r$((solicitud_json ->> 'Respuesta'::text) = ANY (ARRAY['renuncia'::text, 'expiracion_gobernada'::text, 'aceptacion'::text])) AND ((octet_length(continuacion_material)$r$;
    v_int text:=$i$(((continuacion_recibo -> 'Solicitud'::text) ->> 'IntencionRef'::text) = (comando_siguiente_json ->> 'intencion_ref'::text))$i$;
    v_int_n text:=$i$((((continuacion_recibo -> 'Solicitud'::text) ->> 'IntencionRef'::text) = (comando_siguiente_json ->> 'intencion_ref'::text)) OR (((solicitud_json ->> 'Respuesta'::text) = 'aceptacion'::text) AND (comando_siguiente_json IS NULL)))$i$;
BEGIN
    SELECT pg_get_constraintdef(oid) INTO STRICT v_def FROM pg_constraint
     WHERE conrelid='vec_contratacion_temporal.resolucion_manual_respuesta_rrhh'::regclass
       AND conname='continuacion_confirmacion_completa' AND contype='c' AND convalidated;
    IF length(v_def)-length(replace(v_def,v_resp,''))<>length(v_resp)
       OR length(v_def)-length(replace(v_def,v_int,''))<>length(v_int) THEN
        RAISE EXCEPTION 'CT124: preimagen de la continuación incompatible' USING ERRCODE='55000';
    END IF;
    v_nueva:=replace(replace(v_def,v_resp,v_resp_n),v_int,v_int_n);
    ALTER TABLE vec_contratacion_temporal.resolucion_manual_respuesta_rrhh DROP CONSTRAINT continuacion_confirmacion_completa;
    EXECUTE 'ALTER TABLE vec_contratacion_temporal.resolucion_manual_respuesta_rrhh ADD CONSTRAINT continuacion_confirmacion_completa '||v_nueva;
END
$restriccion$;

-- ============================================================ CONSULTA RRHH
-- Confirmación de GINPIX, confirmación del centro y no incorporación para el
-- detalle. Como la
-- consulta de CT115, solo tras acreditar la lectura V3 del mismo detalle; sin
-- texto libre ni actor.
CREATE FUNCTION vec_contratacion_temporal.consultar_incorporacion_acreditada_v1(p_organizacion text, p_expediente text)
RETURNS jsonb LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path=pg_catalog SET row_security='on' SET timezone='UTC'
AS $funcion$
DECLARE g record; c record; n record;
BEGIN
    IF current_user<>'vec_contratacion_temporal_propietario' OR session_user=current_user
       OR NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
       OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
       OR pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER') THEN
        RAISE EXCEPTION 'CT124: consulta no autorizada' USING ERRCODE='42501';
    END IF;
    IF NOT vec_contratacion_temporal.referencia_valida_ct115(p_organizacion) OR NOT vec_contratacion_temporal.referencia_valida_ct115(p_expediente) THEN
        RAISE EXCEPTION 'CT124: consulta inválida' USING ERRCODE='22023';
    END IF;
    IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.expediente_integral_actual a
                 JOIN vec_contratacion_temporal.expediente_version_integral v
                   ON v.expediente_ref=a.expediente_ref AND v.version=a.version
                WHERE a.expediente_ref=p_expediente
                  AND v.agregado_json->>'organizacion_ref' IS DISTINCT FROM p_organizacion) THEN
        RAISE EXCEPTION 'CT124: consulta no autorizada' USING ERRCODE='42501';
    END IF;
    PERFORM set_config('vec.ct115.organizacion_ref',p_organizacion,true);
    PERFORM set_config('vec.ct115.expediente_ref',p_expediente,true);
    SELECT * INTO g FROM vec_contratacion_temporal.ginpix_confirmado_ct124(p_organizacion,p_expediente);
    SELECT * INTO c FROM vec_contratacion_temporal.incorporacion_centro_v1 WHERE organizacion_ref=p_organizacion AND expediente_ref=p_expediente;
    SELECT * INTO n FROM vec_contratacion_temporal.no_incorporacion_v1 WHERE organizacion_ref=p_organizacion AND expediente_ref=p_expediente
     ORDER BY registrada_en DESC, recibo_ref COLLATE "C" DESC LIMIT 1;
    RETURN jsonb_build_object('esquema','vec.contratacion-temporal.incorporacion-acreditada.v1','expediente_ref',p_expediente,
        'ginpix',CASE WHEN g.recibo_ref IS NULL THEN NULL ELSE jsonb_build_object('ginpix_numero',g.ginpix_numero,
            'ginpix_confirmada_en',to_char(g.ginpix_confirmada_en,'YYYY-MM-DD'),
            'recibo',jsonb_build_object('recibo_ref',g.recibo_ref,'registrada_en',to_char(g.registrada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))) END,
        'centro',CASE WHEN c.recibo_ref IS NULL THEN NULL ELSE jsonb_build_object('fecha_incorporacion',to_char(c.fecha_incorporacion,'YYYY-MM-DD'),
            'documento_tipo',c.documento_tipo,'documento_ref',c.documento_ref,'documento_sha256',c.documento_sha256,
            'recibo',jsonb_build_object('recibo_ref',c.recibo_ref,'registrada_en',to_char(c.registrada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))) END,
        'no_incorporacion',CASE WHEN n.recibo_ref IS NULL THEN NULL ELSE jsonb_build_object('motivo_clave',n.motivo_clave,
            'consecuencia_clave',n.consecuencia_clave,'resolucion_ref',n.resolucion_ref,'resolucion_sha256',n.resolucion_sha256,
            'resuelta_por',n.resuelta_por,'fecha_notificacion',to_char(n.fecha_notificacion,'YYYY-MM-DD'),
            'recibo',jsonb_build_object('recibo_ref',n.recibo_ref,'registrada_en',to_char(n.registrada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))) END);
END
$funcion$;

-- ACL: solo el ejecutor de CT invoca las fachadas; tablas y auxiliares
-- quedan cerrados, también frente a privilegios por defecto.
DO $acl$
DECLARE v record; f regprocedure; t text; destinatario text;
    tablas text[]:=ARRAY['confirmacion_ginpix_v1','incorporacion_centro_v1','incorporacion_centro_acceso_v1','no_incorporacion_v1'];
    fachadas regprocedure[]:=ARRAY[
      'vec_contratacion_temporal.preparar_confirmacion_ginpix_v1(jsonb)'::regprocedure,
      'vec_contratacion_temporal.confirmar_confirmacion_ginpix_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
      'vec_contratacion_temporal.consultar_incorporaciones_centro_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
      'vec_contratacion_temporal.confirmar_incorporacion_centro_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
      'vec_contratacion_temporal.consultar_incorporacion_acreditada_v1(text,text)'::regprocedure,
      'vec_contratacion_temporal.preparar_no_incorporacion_v1(jsonb)'::regprocedure,
      'vec_contratacion_temporal.confirmar_no_incorporacion_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
      'vec_contratacion_temporal.leer_no_incorporaciones_bolsa_v1(bigint,text,integer)'::regprocedure];
    auxiliares regprocedure[]:=ARRAY[
      'vec_contratacion_temporal.validar_material_ginpix_ct124(jsonb)'::regprocedure,
      'vec_contratacion_temporal.resultado_ginpix_ct124(vec_contratacion_temporal.confirmacion_ginpix_v1)'::regprocedure,
      'vec_contratacion_temporal.ginpix_confirmado_ct124(text,text)'::regprocedure,
      'vec_contratacion_temporal.actor_centro_valido_ct124(jsonb)'::regprocedure,
      'vec_contratacion_temporal.expedientes_centro_ct124(jsonb,text)'::regprocedure,
      'vec_contratacion_temporal.consumir_incorporacion_centro_ct124(text,text,jsonb,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
      'vec_contratacion_temporal.validar_material_no_incorporacion_ct124(jsonb)'::regprocedure,
      'vec_contratacion_temporal.aceptacion_vigente_ct124(text,text)'::regprocedure,
      'vec_contratacion_temporal.intencion_no_incorporacion_ct124(text,text,text,text,text,text)'::regprocedure,
      'vec_contratacion_temporal.resultado_no_incorporacion_ct124(vec_contratacion_temporal.no_incorporacion_v1)'::regprocedure,
      'vec_contratacion_temporal.antecedente_no_incorporacion_ct124(text,text)'::regprocedure,
      'vec_contratacion_temporal.evento_no_incorporacion_bolsa_ct124(vec_contratacion_temporal.no_incorporacion_v1)'::regprocedure,
      'vec_contratacion_temporal.no_incorporacion_publicada_bolsa_v1(text,text,bigint)'::regprocedure];
BEGIN
    FOREACH t IN ARRAY tablas LOOP
        FOR v IN SELECT DISTINCT x.grantee FROM pg_class c CROSS JOIN LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) x
                  WHERE c.oid=('vec_contratacion_temporal.'||t)::regclass AND x.grantee<>c.relowner LOOP
            destinatario:=CASE WHEN v.grantee=0 THEN 'PUBLIC' ELSE quote_ident(pg_get_userbyid(v.grantee)) END;
            EXECUTE format('REVOKE ALL ON TABLE vec_contratacion_temporal.%I FROM %s',t,destinatario);
        END LOOP;
        EXECUTE format('REVOKE ALL ON TABLE vec_contratacion_temporal.%I FROM PUBLIC',t);
    END LOOP;
    FOREACH f IN ARRAY fachadas||auxiliares LOOP
        FOR v IN SELECT DISTINCT x.grantee FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x
                  WHERE p.oid=f AND x.grantee<>p.proowner LOOP
            destinatario:=CASE WHEN v.grantee=0 THEN 'PUBLIC' ELSE quote_ident(pg_get_userbyid(v.grantee)) END;
            EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %s',f::text,destinatario);
        END LOOP;
    END LOOP;
    FOREACH f IN ARRAY fachadas LOOP
        EXECUTE format('GRANT EXECUTE ON FUNCTION %s TO vec_contratacion_temporal_ejecutor',f::text);
    END LOOP;
    IF EXISTS (SELECT 1 FROM pg_class c CROSS JOIN LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) x
                WHERE c.oid IN (SELECT ('vec_contratacion_temporal.'||u)::regclass FROM unnest(tablas) u)
                  AND (c.relowner<>'vec_contratacion_temporal_propietario'::regrole OR x.grantee<>c.relowner))
       OR EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x
                   WHERE p.oid=ANY(fachadas||auxiliares)
                     AND (p.proowner<>'vec_contratacion_temporal_propietario'::regrole
                          OR (x.grantee<>p.proowner AND NOT (p.oid=ANY(fachadas) AND x.grantee='vec_contratacion_temporal_ejecutor'::regrole
                              AND x.privilege_type='EXECUTE' AND NOT x.is_grantable))))
       OR EXISTS (SELECT 1 FROM unnest(tablas) u
                   WHERE has_table_privilege('vec_contratacion_temporal_ejecutor','vec_contratacion_temporal.'||u,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER'))
       OR EXISTS (SELECT 1 FROM unnest(auxiliares) x WHERE has_function_privilege('vec_contratacion_temporal_ejecutor',x,'EXECUTE'))
       OR EXISTS (SELECT 1 FROM unnest(fachadas) x WHERE NOT has_function_privilege('vec_contratacion_temporal_ejecutor',x,'EXECUTE'))
       OR EXISTS (SELECT 1 FROM unnest(fachadas) x WHERE (SELECT NOT prosecdef FROM pg_proc WHERE oid=x)) THEN
        RAISE EXCEPTION 'CT124: ACL efectiva incompatible' USING ERRCODE='42501';
    END IF;
END
$acl$;
-- Comprobación de origen de Bolsa 000042: el propietario de Bolsa (sus
-- funciones definidoras) solo puede invocar esta función del esquema de CT.
GRANT USAGE ON SCHEMA vec_contratacion_temporal TO vec_bolsa_llamamientos_propietario;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.no_incorporacion_publicada_bolsa_v1(text,text,bigint) TO vec_bolsa_llamamientos_propietario;
DO $acl_bolsa$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
                WHERE n.nspname='vec_contratacion_temporal' AND has_function_privilege('vec_bolsa_llamamientos_propietario',p.oid,'EXECUTE')
                  AND p.oid<>'vec_contratacion_temporal.no_incorporacion_publicada_bolsa_v1(text,text,bigint)'::regprocedure)
       OR EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace
                   WHERE n.nspname='vec_contratacion_temporal'
                     AND has_table_privilege('vec_bolsa_llamamientos_propietario',c.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER')) THEN
        RAISE EXCEPTION 'CT124: el propietario de Bolsa alcanza más que la comprobación de origen' USING ERRCODE='42501';
    END IF;
END
$acl_bolsa$;
COMMENT ON TABLE vec_contratacion_temporal.confirmacion_ginpix_v1 IS
    'CT124: confirmación de GINPIX (número de alta y fecha) de la ficha de la incorporación acreditada; el cierre toma de aquí su número.';
COMMENT ON TABLE vec_contratacion_temporal.incorporacion_centro_v1 IS
    'CT124: confirmación de la incorporación por el centro, con el documento acreditativo que fija el catálogo (referencia y huella).';
COMMENT ON TABLE vec_contratacion_temporal.no_incorporacion_v1 IS
    'CT124: no incorporación registrada por RRHH (resolución por referencia y huella); vuelve a fiscalización, publica la baja a Bolsa y la intención de siguiente candidato.';
COMMIT;
