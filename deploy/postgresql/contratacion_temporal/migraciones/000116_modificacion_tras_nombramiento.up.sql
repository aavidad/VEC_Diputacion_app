\set ON_ERROR_STOP on
-- CT116: modificación de fechas o de jornada de un expediente ya nombrado
-- (petición RRHH p. 1, punto 3; duda 12). Crea una versión nueva del análisis
-- con el periodo y la jornada nuevos y el coste recalculado por la fuente de
-- coste, y devuelve el expediente a la fase que fija la regla c09 del
-- catálogo (fiscalización o informe jurídico). Retira de la proyección
-- vigente la fiscalización (y, si vuelve a informe, también el informe) que
-- correspondían al análisis anterior; su historia se conserva en las
-- actuaciones y en sus propias tablas. Consume AD3-83 y publica
-- `ct.modificacion.v1` en el outbox del expediente en la misma transacción.
-- Requiere CT115 (utilidades comunes) y AD3-83.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000116',0));

DO $prevalidacion$
DECLARE v_origen text;
BEGIN
    IF current_user<>'vec_contratacion_temporal_propietario'
       OR to_regclass('vec_contratacion_temporal.cese_nombramiento_v1') IS NULL
       OR to_regprocedure('vec_contratacion_temporal.validar_confirmacion_ct115(jsonb,text,text,text,text,text,bytea,bytea,numeric,numeric)') IS NULL
       OR to_regprocedure('vec_contratacion_temporal.validar_preparacion_ct115(jsonb,text,text,text)') IS NULL
       OR to_regclass('vec_contratacion_temporal.modificacion_nombramiento_v1') IS NOT NULL
       OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_modificacion_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
       OR NOT has_function_privilege(current_user,'vec_autorizacion_atestada_v3.registrar_y_consumir_modificacion_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE') THEN
        RAISE EXCEPTION 'CT116: dependencias incompatibles (CT115 y AD3-83 requeridas)' USING ERRCODE='55000';
    END IF;
    SELECT pg_get_constraintdef(oid) INTO STRICT v_origen FROM pg_constraint
     WHERE conrelid='vec_contratacion_temporal.expediente_version_integral'::regclass
       AND conname='expediente_version_integral_origen_version_check' AND contype='c' AND convalidated;
    IF strpos(v_origen,'''cierre_expediente_ct115''::text')=0 OR right(v_origen,4)<>'])))'
       OR strpos(v_origen,'modificacion_nombramiento_ct116')<>0 THEN
        RAISE EXCEPTION 'CT116: preimagen de origen de versión incompatible' USING ERRCODE='55000';
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
        ||left(v_origen,length(v_origen)-4)||', ''modificacion_nombramiento_ct116''::text])))';
END
$origen$;

CREATE TABLE vec_contratacion_temporal.modificacion_nombramiento_v1 (
    ambito_hmac text PRIMARY KEY CHECK (ambito_hmac ~ '^hmac-sha256:vec[.]contratacion-temporal[.]modificacion-nombramiento[.]ambito/v[1-9][0-9]{0,8}:[0-9a-f]{64}$'),
    huella_peticion_hmac text NOT NULL CHECK (huella_peticion_hmac ~ '^hmac-sha256:vec[.]contratacion-temporal[.]modificacion-nombramiento[.]peticion/v[1-9][0-9]{0,8}:[0-9a-f]{64}$'),
    organizacion_ref text NOT NULL,
    expediente_ref text NOT NULL,
    version_esperada numeric(20,0) NOT NULL CHECK (version_esperada BETWEEN 1 AND 9007199254740990),
    actor_ref text NOT NULL,
    perfil_ref text NOT NULL,
    motivo_clave text NOT NULL CHECK (motivo_clave ~ '^[a-z][a-z0-9_.-]{1,79}$'),
    periodo_anterior jsonb NOT NULL CHECK (jsonb_typeof(periodo_anterior)='object'),
    jornada_anterior numeric(5,0) NOT NULL CHECK (jornada_anterior BETWEEN 1 AND 10000),
    coste_anterior_centimos numeric(20,0) CHECK (coste_anterior_centimos>=0),
    periodo_inicio date NOT NULL CHECK (isfinite(periodo_inicio)),
    periodo_fin date NOT NULL CHECK (isfinite(periodo_fin) AND periodo_fin>=periodo_inicio),
    porcentaje_jornada numeric(5,0) NOT NULL CHECK (porcentaje_jornada BETWEEN 1 AND 10000),
    coste_centimos numeric(20,0) NOT NULL CHECK (coste_centimos BETWEEN 1 AND 922337203685477),
    fuente_coste_ref text NOT NULL CHECK (fuente_coste_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
    fase_retorno text NOT NULL CHECK (fase_retorno IN ('fiscalizacion','informe_juridico')),
    estado_retorno text NOT NULL CHECK (estado_retorno='en_curso'),
    observaciones text NOT NULL CHECK (vec_contratacion_temporal.texto_valido_ct115(observaciones,2000,false)),
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
ALTER TABLE vec_contratacion_temporal.modificacion_nombramiento_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contratacion_temporal.modificacion_nombramiento_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY lectura_operacion_ct116 ON vec_contratacion_temporal.modificacion_nombramiento_v1 FOR SELECT TO vec_contratacion_temporal_propietario USING (
    organizacion_ref=current_setting('vec.ct115.organizacion_ref',true)
    AND expediente_ref=current_setting('vec.ct115.expediente_ref',true)
    AND pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
    AND NOT pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
    AND NOT pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER'));
CREATE POLICY escritura_operacion_ct116 ON vec_contratacion_temporal.modificacion_nombramiento_v1 FOR INSERT TO vec_contratacion_temporal_propietario WITH CHECK (
    organizacion_ref=current_setting('vec.ct115.organizacion_ref',true)
    AND expediente_ref=current_setting('vec.ct115.expediente_ref',true)
    AND actor_ref=current_setting('vec.ct115.actor_ref',true)
    AND perfil_ref=current_setting('vec.ct115.perfil_ref',true)
    AND ambito_hmac=current_setting('vec.ct115.ambito_hmac',true)
    AND pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
    AND NOT pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
    AND NOT pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER'));
CREATE TRIGGER modificacion_nombramiento_v1_inmutable BEFORE UPDATE OR DELETE ON vec_contratacion_temporal.modificacion_nombramiento_v1
    FOR EACH ROW EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();

CREATE FUNCTION vec_contratacion_temporal.validar_material_modificacion_ct116(m jsonb)
RETURNS void LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog AS $$
DECLARE k text; f date;
BEGIN
    IF m IS NULL OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(m,ARRAY['organizacion_ref','expediente_ref','version_esperada',
        'actor_ref','perfil_ref','motivo_clave','periodo_inicio','periodo_fin','porcentaje_jornada','coste_centimos','fuente_coste_ref',
        'fase_retorno','estado_retorno','observaciones']) IS NOT TRUE
       OR EXISTS (SELECT 1 FROM jsonb_each(m) c WHERE jsonb_typeof(c.value) IS DISTINCT FROM
            CASE WHEN c.key IN ('version_esperada','porcentaje_jornada','coste_centimos') THEN 'number' ELSE 'string' END)
       OR m->>'version_esperada' !~ '^[1-9][0-9]{0,15}$' OR (m->>'version_esperada')::numeric>9007199254740990
       OR m->>'porcentaje_jornada' !~ '^[1-9][0-9]{0,4}$' OR (m->>'porcentaje_jornada')::numeric>10000
       OR m->>'coste_centimos' !~ '^[1-9][0-9]{0,14}$' OR (m->>'coste_centimos')::numeric>922337203685477
       OR m->>'motivo_clave' !~ '^[a-z][a-z0-9_.-]{1,79}$'
       OR m->>'fase_retorno' NOT IN ('fiscalizacion','informe_juridico') OR m->>'estado_retorno'<>'en_curso'
       OR m->>'periodo_inicio' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$' OR m->>'periodo_fin' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$'
       OR NOT vec_contratacion_temporal.texto_valido_ct115(m->>'observaciones',2000,false) THEN
        RAISE EXCEPTION 'CT116: material de modificación inválido' USING ERRCODE='22023';
    END IF;
    FOREACH k IN ARRAY ARRAY['organizacion_ref','expediente_ref','actor_ref','perfil_ref','fuente_coste_ref'] LOOP
        IF NOT vec_contratacion_temporal.referencia_valida_ct115(m->>k) THEN
            RAISE EXCEPTION 'CT116: referencia de modificación inválida' USING ERRCODE='22023';
        END IF;
    END LOOP;
    FOREACH k IN ARRAY ARRAY['periodo_inicio','periodo_fin'] LOOP
        f:=(m->>k)::date;
        IF to_char(f,'YYYY-MM-DD')<>m->>k THEN RAISE EXCEPTION 'CT116: fecha inválida' USING ERRCODE='22023'; END IF;
    END LOOP;
    IF (m->>'periodo_fin')::date<(m->>'periodo_inicio')::date THEN
        RAISE EXCEPTION 'CT116: periodo invertido' USING ERRCODE='22023';
    END IF;
EXCEPTION WHEN invalid_datetime_format OR datetime_field_overflow THEN
    RAISE EXCEPTION 'CT116: fecha inválida' USING ERRCODE='22023';
END
$$;

-- Estado admisible del expediente y proyección siguiente de la modificación.
-- Devuelve NULL si el estado no admite la modificación (conflicto) y un
-- objeto con `resultado` si la regla de negocio la rechaza.
CREATE FUNCTION vec_contratacion_temporal.proyeccion_modificacion_ct116(ag jsonb, m jsonb, p_recibo text, p_instante jsonb)
RETURNS jsonb LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog AS $$
DECLARE a jsonb:=ag->'analisis'; v numeric:=(ag->>'version')::numeric; act jsonb; nuevo jsonb; sig jsonb; periodo jsonb;
BEGIN
    IF ag->>'organizacion_ref' IS DISTINCT FROM m->>'organizacion_ref' OR ag->>'fase_actual' IS DISTINCT FROM 'nombramiento'
       OR ag->>'estado_actual' IS DISTINCT FROM 'en_curso' OR jsonb_typeof(a) IS DISTINCT FROM 'object'
       OR jsonb_typeof(ag->'actuaciones') IS DISTINCT FROM 'array'
       OR NOT vec_contratacion_temporal.referencia_valida_ct115(ag#>>'{asignacion,unidad_ref}')
       OR (m->>'fase_retorno'='informe_juridico' AND NOT ag ? 'informe_juridico')
       OR (m->>'fase_retorno'='fiscalizacion' AND NOT ag ? 'informe_juridico') THEN
        RETURN NULL;
    END IF;
    periodo:=jsonb_build_object('inicio',(m->>'periodo_inicio')||'T00:00:00Z','fin',(m->>'periodo_fin')||'T00:00:00Z');
    IF a->'periodo'=periodo AND (a->>'porcentaje_jornada')::numeric=(m->>'porcentaje_jornada')::numeric THEN
        RETURN jsonb_build_object('resultado','sin_cambios');
    END IF;
    IF a#>>'{validacion_rc,resultado}'='validada' AND (m->>'coste_centimos')::numeric>(a#>>'{validacion_rc,importe,centimos}')::numeric THEN
        RETURN jsonb_build_object('resultado','credito_insuficiente');
    END IF;
    act:=jsonb_build_object('secuencia',jsonb_array_length(ag->'actuaciones')+1,'version_expediente',v+1,
        'accion_clave','contratacion_temporal.expediente.modificar_tras_nombramiento','actor_ref',m->>'actor_ref',
        'unidad_ref',ag#>>'{asignacion,unidad_ref}','recibo_ref',p_recibo,'realizada_en',p_instante,
        'fase_origen','nombramiento','fase_destino',m->>'fase_retorno','estado_origen','en_curso','estado_destino',m->>'estado_retorno',
        'observaciones',m->>'observaciones');
    nuevo:=a||jsonb_build_object('periodo',periodo,'porcentaje_jornada',(m->>'porcentaje_jornada')::numeric,
        'coste_previsto',jsonb_build_object('moneda','EUR','centimos',(m->>'coste_centimos')::numeric),
        'fuente_coste_ref',m->>'fuente_coste_ref',
        'actuacion_registro',jsonb_build_object('secuencia',jsonb_array_length(ag->'actuaciones')+1,'version_expediente',v+1,
            'accion_clave','contratacion_temporal.expediente.modificar_tras_nombramiento','fase_destino',m->>'fase_retorno','recibo_ref',p_recibo));
    sig:=(ag-'fiscalizacion')||jsonb_build_object('version',v+1,'fase_actual',m->>'fase_retorno','estado_actual',m->>'estado_retorno',
        'actualizado_en',p_instante,'analisis',nuevo,'actuaciones',(ag->'actuaciones')||jsonb_build_array(act));
    IF m->>'fase_retorno'='informe_juridico' THEN sig:=sig-'informe_juridico'; END IF;
    RETURN jsonb_build_object('resultado','preparada','actuacion',act,'expediente',sig);
END
$$;

CREATE FUNCTION vec_contratacion_temporal.resultado_modificacion_ct116(r vec_contratacion_temporal.modificacion_nombramiento_v1)
RETURNS jsonb LANGUAGE sql STABLE SET search_path=pg_catalog AS $$
    SELECT jsonb_build_object('esquema','vec.contratacion-temporal.resultado-modificacion-nombramiento.v1','resultado','confirmada',
        'material',jsonb_build_object('organizacion_ref',r.organizacion_ref,'expediente_ref',r.expediente_ref,'version_esperada',r.version_esperada,
            'actor_ref',r.actor_ref,'perfil_ref',r.perfil_ref,'motivo_clave',r.motivo_clave,'periodo_inicio',to_char(r.periodo_inicio,'YYYY-MM-DD'),
            'periodo_fin',to_char(r.periodo_fin,'YYYY-MM-DD'),'porcentaje_jornada',r.porcentaje_jornada,'coste_centimos',r.coste_centimos,
            'fuente_coste_ref',r.fuente_coste_ref,'fase_retorno',r.fase_retorno,'estado_retorno',r.estado_retorno,'observaciones',r.observaciones),
        'expediente',r.expediente_siguiente_json,
        'referencias',jsonb_build_object('reserva_ref',r.reserva_ref,'recibo_ref',r.recibo_ref,'evento_ref',r.evento_ref),
        'ambito_idempotencia_hmac',r.ambito_hmac,'huella_peticion_hmac',r.huella_peticion_hmac,'recibo',r.recibo_json)
$$;

CREATE FUNCTION vec_contratacion_temporal.preparar_modificacion_nombramiento_v1(p_operacion jsonb)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security='on' SET timezone='UTC' SET lock_timeout='2s'
AS $funcion$
DECLARE
    m jsonb:=p_operacion->'material'; v_pares jsonb; v_par jsonb; v_actual record; v_proy jsonb;
    r vec_contratacion_temporal.modificacion_nombramiento_v1%ROWTYPE;
    e text:='vec.contratacion-temporal.resultado-modificacion-nombramiento.v1';
BEGIN
    PERFORM vec_contratacion_temporal.exigir_sesion_ct115(true);
    v_pares:=vec_contratacion_temporal.validar_preparacion_ct115(p_operacion,'vec.contratacion-temporal.preparar-modificacion-nombramiento.v1',
        'modificar_tras_nombramiento','modificacion-nombramiento');
    PERFORM vec_contratacion_temporal.validar_material_modificacion_ct116(m);
    PERFORM set_config('vec.ct115.organizacion_ref',m->>'organizacion_ref',true);
    PERFORM set_config('vec.ct115.expediente_ref',m->>'expediente_ref',true);
    FOR v_par IN SELECT value FROM jsonb_array_elements(v_pares) LOOP
        SELECT * INTO r FROM vec_contratacion_temporal.modificacion_nombramiento_v1 WHERE ambito_hmac=v_par->>'ambito_hmac';
        IF FOUND THEN
            IF r.huella_peticion_hmac IS DISTINCT FROM v_par->>'huella_peticion_hmac' OR r.organizacion_ref<>m->>'organizacion_ref'
               OR r.expediente_ref<>m->>'expediente_ref' OR r.version_esperada<>(m->>'version_esperada')::numeric
               OR r.actor_ref<>m->>'actor_ref' OR r.perfil_ref<>m->>'perfil_ref' OR r.motivo_clave<>m->>'motivo_clave'
               OR to_char(r.periodo_inicio,'YYYY-MM-DD')<>m->>'periodo_inicio' OR to_char(r.periodo_fin,'YYYY-MM-DD')<>m->>'periodo_fin'
               OR r.porcentaje_jornada<>(m->>'porcentaje_jornada')::numeric OR r.coste_centimos<>(m->>'coste_centimos')::numeric
               OR r.fuente_coste_ref<>m->>'fuente_coste_ref' OR r.fase_retorno<>m->>'fase_retorno' OR r.observaciones<>m->>'observaciones' THEN
                RETURN jsonb_build_object('esquema',e,'resultado','idempotencia_reutilizada');
            END IF;
            RETURN vec_contratacion_temporal.resultado_modificacion_ct116(r);
        END IF;
    END LOOP;
    IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.cese_nombramiento_v1 c WHERE c.expediente_ref=m->>'expediente_ref') THEN
        RETURN jsonb_build_object('esquema',e,'resultado','cese_existente');
    END IF;
    SELECT a.version, v.agregado_json INTO v_actual
      FROM vec_contratacion_temporal.expediente_integral_actual a
      JOIN vec_contratacion_temporal.expediente_version_integral v USING (expediente_ref,version)
     WHERE a.expediente_ref=m->>'expediente_ref';
    IF NOT FOUND OR v_actual.version<>(m->>'version_esperada')::numeric THEN
        RETURN jsonb_build_object('esquema',e,'resultado','version_en_conflicto');
    END IF;
    v_proy:=vec_contratacion_temporal.proyeccion_modificacion_ct116(v_actual.agregado_json,m,p_operacion#>>'{referencias_candidatas,recibo_ref}','"1970-01-01T00:00:00Z"');
    IF v_proy IS NULL THEN
        RETURN jsonb_build_object('esquema',e,'resultado','version_en_conflicto');
    END IF;
    IF v_proy->>'resultado'<>'preparada' THEN
        RETURN jsonb_build_object('esquema',e,'resultado',v_proy->>'resultado');
    END IF;
    RETURN jsonb_build_object('esquema',e,'resultado','preparada','material',m,'expediente',v_actual.agregado_json,
        'referencias',p_operacion->'referencias_candidatas',
        'ambito_idempotencia_hmac',p_operacion#>>'{sellos_hmac,activo,ambito_hmac}',
        'huella_peticion_hmac',p_operacion#>>'{sellos_hmac,activo,huella_peticion_hmac}');
EXCEPTION WHEN invalid_text_representation OR datetime_field_overflow OR numeric_value_out_of_range OR character_not_in_repertoire THEN
    RAISE EXCEPTION 'CT116: entrada de modificación inválida' USING ERRCODE='22023';
END
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.confirmar_modificacion_nombramiento_v1(
    p_operacion jsonb,
    p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,
    p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security='on' SET timezone='UTC' SET lock_timeout='2s'
AS $funcion$
DECLARE
    m jsonb:=p_operacion->'material'; refs jsonb:=p_operacion->'referencias'; pol jsonb:=p_operacion->'politica'; a jsonb:=p_operacion->'autorizacion';
    e text:='vec.contratacion-temporal.resultado-modificacion-nombramiento.v1';
    r vec_contratacion_temporal.modificacion_nombramiento_v1%ROWTYPE;
    v_version numeric; v_instante timestamptz; v_ahora timestamptz(6); v_actual record; v_proy jsonb; v_decision jsonb; v_consumo record;
    v_actuacion jsonb; v_siguiente jsonb; v_ambitos jsonb; v_atributos jsonb; v_contexto text;
    v_agregado_huella text; v_prueba bytea; v_payload bytea; v_anterior text; v_secuencia numeric; v_recibo jsonb;
    v_restriccion text; v_tabla text; v_esquema text;
BEGIN
    PERFORM vec_contratacion_temporal.exigir_sesion_ct115(false);
    PERFORM vec_contratacion_temporal.validar_confirmacion_ct115(p_operacion,'vec.contratacion-temporal.confirmar-modificacion-nombramiento.v1',
        'modificar_tras_nombramiento','modificacion-nombramiento','contratacion_temporal.expediente.modificar_tras_nombramiento',
        'modificar_expediente_tras_nombramiento',p_decision,p_motivo,p_persona_version,p_perfil_version);
    PERFORM vec_contratacion_temporal.validar_material_modificacion_ct116(m);
    IF p_capacidad IS NULL OR p_contexto IS NULL OR p_payload IS NULL OR p_sobre IS NULL OR p_evidencia IS NULL OR p_raiz IS NULL THEN
        RAISE EXCEPTION 'CT116: autorización incompleta' USING ERRCODE='42501';
    END IF;
    v_version:=(m->>'version_esperada')::numeric;
    v_instante:=(p_operacion->>'instante_efecto')::timestamptz;
    v_decision:=convert_from(p_decision,'UTF8')::jsonb;
    PERFORM set_config('vec.ct115.organizacion_ref',m->>'organizacion_ref',true);
    PERFORM set_config('vec.ct115.expediente_ref',m->>'expediente_ref',true);
    PERFORM set_config('vec.ct115.actor_ref',m->>'actor_ref',true);
    PERFORM set_config('vec.ct115.perfil_ref',m->>'perfil_ref',true);
    PERFORM set_config('vec.ct115.ambito_hmac',p_operacion->>'ambito_idempotencia_hmac',true);
    SELECT * INTO r FROM vec_contratacion_temporal.modificacion_nombramiento_v1 WHERE ambito_hmac=p_operacion->>'ambito_idempotencia_hmac';
    IF FOUND THEN
        IF r.huella_peticion_hmac IS DISTINCT FROM p_operacion->>'huella_peticion_hmac' OR r.version_esperada<>v_version
           OR r.motivo_clave<>m->>'motivo_clave' OR to_char(r.periodo_inicio,'YYYY-MM-DD')<>m->>'periodo_inicio'
           OR to_char(r.periodo_fin,'YYYY-MM-DD')<>m->>'periodo_fin' OR r.porcentaje_jornada<>(m->>'porcentaje_jornada')::numeric
           OR r.coste_centimos<>(m->>'coste_centimos')::numeric OR r.observaciones<>m->>'observaciones' THEN
            RETURN jsonb_build_object('esquema',e,'resultado','idempotencia_reutilizada');
        END IF;
        IF r.decision_ref IS DISTINCT FROM a->>'decision_ref' THEN
            RAISE EXCEPTION 'CT116: evidencia de repetición divergente' USING ERRCODE='42501';
        END IF;
        RETURN jsonb_build_object('esquema',e,'resultado','confirmada','recibo',r.recibo_json);
    END IF;
    IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.cese_nombramiento_v1 c WHERE c.expediente_ref=m->>'expediente_ref') THEN
        RETURN jsonb_build_object('esquema',e,'resultado','cese_existente');
    END IF;
    SELECT v.* INTO v_actual FROM vec_contratacion_temporal.expediente_integral_actual ac
      JOIN vec_contratacion_temporal.expediente_version_integral v USING (expediente_ref,version)
     WHERE ac.expediente_ref=m->>'expediente_ref' FOR UPDATE OF ac,v;
    IF NOT FOUND OR v_actual.version<>v_version OR v_actual.agregado_json IS DISTINCT FROM p_operacion->'expediente_anterior' THEN
        RETURN jsonb_build_object('esquema',e,'resultado','version_en_conflicto');
    END IF;
    v_proy:=vec_contratacion_temporal.proyeccion_modificacion_ct116(v_actual.agregado_json,m,refs->>'recibo_ref',p_operacion->'instante_efecto');
    IF v_proy IS NULL THEN
        RETURN jsonb_build_object('esquema',e,'resultado','version_en_conflicto');
    END IF;
    IF v_proy->>'resultado'<>'preparada' THEN
        RETURN jsonb_build_object('esquema',e,'resultado',v_proy->>'resultado');
    END IF;
    v_actuacion:=v_proy->'actuacion'; v_siguiente:=v_proy->'expediente';
    IF v_instante<(v_actual.agregado_json->>'actualizado_en')::timestamptz
       OR v_actuacion IS DISTINCT FROM p_operacion->'actuacion'
       OR v_siguiente IS DISTINCT FROM p_operacion->'expediente_siguiente' THEN
        RAISE EXCEPTION 'CT116: proyección de modificación divergente' USING ERRCODE='22023';
    END IF;
    v_ambitos:=jsonb_build_object('organizacion_ref',m->>'organizacion_ref','expediente_ref',m->>'expediente_ref',
        'fase_previa','nombramiento','estado_previo','en_curso');
    v_atributos:=jsonb_build_object('version_expediente',v_version::text,'motivo_clave',m->>'motivo_clave',
        'periodo_inicio',m->>'periodo_inicio','periodo_fin',m->>'periodo_fin','porcentaje_jornada',m->>'porcentaje_jornada',
        'coste_centimos',m->>'coste_centimos','fuente_coste_ref',m->>'fuente_coste_ref','fase_retorno',m->>'fase_retorno',
        'estado_retorno',m->>'estado_retorno','observaciones_huella_sha256',encode(sha256(convert_to(m->>'observaciones','UTF8')),'hex'),
        'politica_ref',pol->>'definicion_ref','politica_version',pol->>'definicion_version','politica_huella_sha256',pol->>'definicion_huella_sha256',
        'ambito_idempotencia_hmac',p_operacion->>'ambito_idempotencia_hmac','huella_peticion_hmac',p_operacion->>'huella_peticion_hmac');
    IF p_operacion#>'{contexto,ambitos}' IS DISTINCT FROM v_ambitos OR p_operacion#>'{contexto,atributos}' IS DISTINCT FROM v_atributos THEN
        RAISE EXCEPTION 'CT116: contexto de modificación divergente' USING ERRCODE='42501';
    END IF;
    v_contexto:=vec_contratacion_temporal.huella_contexto_go_ct115(v_ambitos,v_atributos);
    IF a->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_contexto
       OR v_decision->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_contexto
       OR v_decision->>'principal_id' IS DISTINCT FROM m->>'actor_ref'
       OR v_decision->>'perfil_activo_ref' IS DISTINCT FROM m->>'perfil_ref'
       OR v_decision->>'recurso_ref' IS DISTINCT FROM m->>'expediente_ref'
       OR v_decision->>'accion' IS DISTINCT FROM 'contratacion_temporal.expediente.modificar_tras_nombramiento'
       OR v_decision->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
       OR v_decision->>'tipo_recurso' IS DISTINCT FROM 'modificacion_contratacion_temporal'
       OR v_decision->>'finalidad' IS DISTINCT FROM 'modificar_expediente_tras_nombramiento'
       OR v_decision->>'decision_ref' IS DISTINCT FROM a->>'decision_ref' THEN
        RAISE EXCEPTION 'CT116: contexto autorizado de modificación divergente' USING ERRCODE='42501';
    END IF;
    v_ahora:=date_trunc('microseconds',clock_timestamp());
    IF (pol->>'evaluada_en')::timestamptz>v_instante OR v_ahora<v_instante OR v_ahora>=(pol->>'valida_hasta')::timestamptz THEN
        RAISE EXCEPTION 'CT116: vigencia de modificación agotada' USING ERRCODE='42501';
    END IF;
    SELECT * INTO STRICT v_consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_modificacion_ct_v3_atestada(
        p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
    IF v_consumo.decision_ref IS DISTINCT FROM a->>'decision_ref' OR v_consumo.efecto_ref IS DISTINCT FROM m->>'expediente_ref'
       OR v_consumo.huella_efecto_sha256 IS DISTINCT FROM v_contexto OR coalesce(v_consumo.auditoria_ref,'') !~ '^aud_v3_[0-9a-f]{32}$'
       OR v_consumo.consumo_nuevo IS NOT TRUE THEN
        RAISE EXCEPTION 'CT116: consumo de modificación divergente' USING ERRCODE='42501';
    END IF;
    v_ahora:=date_trunc('microseconds',clock_timestamp());
    IF v_ahora<v_instante OR v_ahora>=(pol->>'valida_hasta')::timestamptz THEN
        RAISE EXCEPTION 'CT116: vigencia final de modificación agotada' USING ERRCODE='42501';
    END IF;
    v_agregado_huella:=encode(sha256(convert_to(v_siguiente::text,'UTF8')),'hex');
    v_prueba:=convert_to('VEC-CT-EXPEDIENTE-MODIFICACION-CT116'||chr(10)||(m->>'expediente_ref')||chr(10)||(v_version+1)::text||chr(10)
        ||v_agregado_huella||chr(10)||(refs->>'reserva_ref')||chr(10)||(refs->>'recibo_ref')||chr(10)||v_consumo.decision_ref||chr(10)||v_ahora::text,'UTF8');
    INSERT INTO vec_contratacion_temporal.expediente_version_integral(
        expediente_ref,version,agregado_json,agregado_json_huella_sha256,prueba_canonica,prueba_huella_sha256,
        flujo_ref,flujo_version,flujo_huella_sha256,fase_clave,estado,origen_version,operacion_ref,registrada_en)
    VALUES(m->>'expediente_ref',v_version+1,v_siguiente,v_agregado_huella,v_prueba,encode(sha256(v_prueba),'hex'),
        v_actual.flujo_ref,v_actual.flujo_version,v_actual.flujo_huella_sha256,m->>'fase_retorno',m->>'estado_retorno',
        'modificacion_nombramiento_ct116',refs->>'reserva_ref',v_ahora);
    UPDATE vec_contratacion_temporal.expediente_integral_actual SET version=v_version+1,actualizada_en=v_ahora,operacion_ref=refs->>'reserva_ref'
     WHERE expediente_ref=m->>'expediente_ref' AND version=v_version;
    IF NOT FOUND THEN RAISE EXCEPTION 'CT116: CAS final de modificación perdido' USING ERRCODE='40001'; END IF;
    v_prueba:=convert_to('VEC-CT-ACTUACION-MODIFICACION-CT116'||chr(10)||encode(sha256(convert_to(v_actuacion::text,'UTF8')),'hex')||chr(10)
        ||(refs->>'recibo_ref')||chr(10)||v_ahora::text,'UTF8');
    INSERT INTO vec_contratacion_temporal.actuacion_expediente_integral(
        expediente_ref,secuencia,version_expediente,operacion_ref,recibo_ref,actuacion_json,
        actuacion_json_huella_sha256,prueba_canonica,prueba_huella_sha256,registrada_en)
    VALUES(m->>'expediente_ref',(v_actuacion->>'secuencia')::numeric,v_version+1,refs->>'reserva_ref',refs->>'recibo_ref',v_actuacion,
        encode(sha256(convert_to(v_actuacion::text,'UTF8')),'hex'),v_prueba,encode(sha256(v_prueba),'hex'),v_ahora);
    SELECT secuencia_outbox,cabeza_outbox_sha256 INTO STRICT v_secuencia,v_anterior
      FROM vec_contratacion_temporal.control_cadenas_expediente_integral WHERE control_id FOR UPDATE;
    IF v_secuencia>=9007199254740991 THEN RAISE EXCEPTION 'CT116: límite de outbox alcanzado' USING ERRCODE='22003'; END IF;
    v_secuencia:=v_secuencia+1;
    v_payload:=convert_to(jsonb_build_object('esquema','vec.contratacion-temporal.modificacion-nombramiento.v1','organizacion_ref',m->>'organizacion_ref',
        'expediente_ref',m->>'expediente_ref','version_resultante',v_version+1,'motivo_clave',m->>'motivo_clave',
        'periodo_inicio',m->>'periodo_inicio','periodo_fin',m->>'periodo_fin','porcentaje_jornada',(m->>'porcentaje_jornada')::numeric,
        'coste_centimos',(m->>'coste_centimos')::numeric,'fase_resultante',m->>'fase_retorno','recibo_ref',refs->>'recibo_ref',
        'registrada_en',p_operacion->'instante_efecto')::text,'UTF8');
    INSERT INTO vec_contratacion_temporal.outbox_expediente_integral(
        evento_ref,secuencia,operacion_ref,expediente_ref,version_expediente,tipo_evento,payload_canonico,
        payload_huella_sha256,anterior_sha256,huella_sha256,registrada_en)
    VALUES(refs->>'evento_ref',v_secuencia,refs->>'reserva_ref',m->>'expediente_ref',v_version+1,'ct.modificacion.v1',v_payload,
        encode(sha256(v_payload),'hex'),v_anterior,encode(sha256(v_anterior::bytea||v_payload),'hex'),v_ahora);
    UPDATE vec_contratacion_temporal.control_cadenas_expediente_integral
       SET secuencia_outbox=v_secuencia,cabeza_outbox_sha256=encode(sha256(v_anterior::bytea||v_payload),'hex'),actualizada_en=v_ahora
     WHERE control_id;
    v_recibo:=jsonb_build_object('operacion','modificar_tras_nombramiento','organizacion_ref',m->>'organizacion_ref','expediente_ref',m->>'expediente_ref',
        'version_anterior',v_version,'version_resultante',v_version+1,'fase_resultante',m->>'fase_retorno','estado_resultante',m->>'estado_retorno',
        'coste_centimos',(m->>'coste_centimos')::numeric,'recibo_ref',refs->>'recibo_ref','auditoria_ref',v_consumo.auditoria_ref,
        'evento_ref',refs->>'evento_ref','actor_ref',m->>'actor_ref','registrada_en',p_operacion->'instante_efecto');
    INSERT INTO vec_contratacion_temporal.modificacion_nombramiento_v1(
        ambito_hmac,huella_peticion_hmac,organizacion_ref,expediente_ref,version_esperada,actor_ref,perfil_ref,motivo_clave,periodo_anterior,
        jornada_anterior,coste_anterior_centimos,periodo_inicio,periodo_fin,porcentaje_jornada,coste_centimos,fuente_coste_ref,fase_retorno,
        estado_retorno,observaciones,estado,reserva_ref,recibo_ref,evento_ref,expediente_anterior_json,expediente_siguiente_json,recibo_json,
        decision_ref,decision_huella_sha256,consumo_huella_sha256,auditoria_ref,politica_ref,politica_version,politica_huella_sha256,registrada_en,confirmada_en)
    VALUES(p_operacion->>'ambito_idempotencia_hmac',p_operacion->>'huella_peticion_hmac',m->>'organizacion_ref',m->>'expediente_ref',v_version,
        m->>'actor_ref',m->>'perfil_ref',m->>'motivo_clave',v_actual.agregado_json#>'{analisis,periodo}',
        (v_actual.agregado_json#>>'{analisis,porcentaje_jornada}')::numeric,(v_actual.agregado_json#>>'{analisis,coste_previsto,centimos}')::numeric,
        (m->>'periodo_inicio')::date,(m->>'periodo_fin')::date,(m->>'porcentaje_jornada')::numeric,(m->>'coste_centimos')::numeric,
        m->>'fuente_coste_ref',m->>'fase_retorno',m->>'estado_retorno',m->>'observaciones','confirmada',refs->>'reserva_ref',refs->>'recibo_ref',
        refs->>'evento_ref',v_actual.agregado_json,v_siguiente,v_recibo,v_consumo.decision_ref,a->>'decision_huella_sha256',
        v_consumo.consumo_huella_sha256,v_consumo.auditoria_ref,pol->>'definicion_ref',(pol->>'definicion_version')::numeric,
        pol->>'definicion_huella_sha256',v_instante,v_ahora);
    RETURN jsonb_build_object('esquema',e,'resultado','confirmada','recibo',v_recibo);
EXCEPTION
WHEN unique_violation THEN
    GET STACKED DIAGNOSTICS v_restriccion=CONSTRAINT_NAME,v_tabla=TABLE_NAME,v_esquema=SCHEMA_NAME;
    IF v_esquema='vec_contratacion_temporal' AND v_tabla='modificacion_nombramiento_v1' THEN
        RETURN jsonb_build_object('esquema',e,'resultado',CASE WHEN v_restriccion='modificacion_nombramiento_v1_pkey' THEN 'idempotencia_reutilizada' ELSE 'version_en_conflicto' END);
    END IF;
    RAISE;
WHEN invalid_text_representation OR datetime_field_overflow OR numeric_value_out_of_range OR character_not_in_repertoire THEN
    RAISE EXCEPTION 'CT116: entrada de modificación inválida' USING ERRCODE='22023';
END
$funcion$;

DO $acl$
DECLARE v record; f regprocedure; destinatario text;
    fachadas regprocedure[]:=ARRAY[
      'vec_contratacion_temporal.preparar_modificacion_nombramiento_v1(jsonb)'::regprocedure,
      'vec_contratacion_temporal.confirmar_modificacion_nombramiento_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure];
    auxiliares regprocedure[]:=ARRAY[
      'vec_contratacion_temporal.validar_material_modificacion_ct116(jsonb)'::regprocedure,
      'vec_contratacion_temporal.proyeccion_modificacion_ct116(jsonb,jsonb,text,jsonb)'::regprocedure,
      'vec_contratacion_temporal.resultado_modificacion_ct116(vec_contratacion_temporal.modificacion_nombramiento_v1)'::regprocedure];
BEGIN
    FOR v IN SELECT DISTINCT x.grantee FROM pg_class c CROSS JOIN LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) x
              WHERE c.oid='vec_contratacion_temporal.modificacion_nombramiento_v1'::regclass AND x.grantee<>c.relowner LOOP
        destinatario:=CASE WHEN v.grantee=0 THEN 'PUBLIC' ELSE quote_ident(pg_get_userbyid(v.grantee)) END;
        EXECUTE 'REVOKE ALL ON TABLE vec_contratacion_temporal.modificacion_nombramiento_v1 FROM '||destinatario;
    END LOOP;
    REVOKE ALL ON TABLE vec_contratacion_temporal.modificacion_nombramiento_v1 FROM PUBLIC;
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
                WHERE c.oid='vec_contratacion_temporal.modificacion_nombramiento_v1'::regclass
                  AND (c.relowner<>'vec_contratacion_temporal_propietario'::regrole OR x.grantee<>c.relowner))
       OR EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x
                   WHERE p.oid=ANY(fachadas||auxiliares)
                     AND (p.proowner<>'vec_contratacion_temporal_propietario'::regrole
                          OR (x.grantee<>p.proowner AND NOT (p.oid=ANY(fachadas) AND x.grantee='vec_contratacion_temporal_ejecutor'::regrole
                              AND x.privilege_type='EXECUTE' AND NOT x.is_grantable))))
       OR has_table_privilege('vec_contratacion_temporal_ejecutor','vec_contratacion_temporal.modificacion_nombramiento_v1','SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER')
       OR EXISTS (SELECT 1 FROM unnest(auxiliares) x WHERE has_function_privilege('vec_contratacion_temporal_ejecutor',x,'EXECUTE'))
       OR EXISTS (SELECT 1 FROM unnest(fachadas) x WHERE NOT has_function_privilege('vec_contratacion_temporal_ejecutor',x,'EXECUTE')) THEN
        RAISE EXCEPTION 'CT116: ACL efectiva incompatible' USING ERRCODE='42501';
    END IF;
END
$acl$;
COMMENT ON TABLE vec_contratacion_temporal.modificacion_nombramiento_v1 IS
    'CT116: modificación de fechas o jornada tras el nombramiento: análisis y coste nuevos y vuelta a la fase de la regla c09.';
COMMIT;
