BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:000052_fiscalizacion_v5_v6', 0
    )
);

DO $prevalidacion$
BEGIN
    IF pg_catalog.to_regclass(
           'vec_contratacion_temporal.expediente_version_integral') IS NULL
       OR pg_catalog.to_regclass(
           'vec_contratacion_temporal.expediente_integral_actual') IS NULL
       OR pg_catalog.to_regclass(
           'vec_contratacion_temporal.actuacion_expediente_integral') IS NULL
       OR pg_catalog.to_regclass(
           'vec_contratacion_temporal.outbox_expediente_integral') IS NULL
       OR pg_catalog.to_regclass(
           'vec_contratacion_temporal.control_cadenas_expediente_integral') IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_autorizacion_atestada_v3.registrar_y_consumir_fiscalizacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
       ) IS NULL
       OR NOT pg_catalog.has_function_privilege(
           'vec_contratacion_temporal_propietario',
           'vec_autorizacion_atestada_v3.registrar_y_consumir_fiscalizacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
           'EXECUTE'
       )
       OR pg_catalog.to_regclass(
           'vec_contratacion_temporal.reserva_fiscalizacion') IS NOT NULL
       OR NOT EXISTS (
           SELECT 1
             FROM pg_catalog.pg_constraint
            WHERE conrelid =
                  'vec_contratacion_temporal.expediente_version_integral'::regclass
              AND conname =
                  'expediente_version_integral_origen_version_check'
       )
       OR NOT EXISTS (
           SELECT 1
             FROM pg_catalog.pg_constraint
            WHERE conrelid =
                  'vec_contratacion_temporal.expediente_version_integral'::regclass
              AND conname =
                  'expediente_version_integral_estado_check'
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para fiscalización durable v5-v6';
    END IF;
END
$prevalidacion$;

ALTER TABLE vec_contratacion_temporal.expediente_version_integral
    DROP CONSTRAINT expediente_version_integral_origen_version_check;
ALTER TABLE vec_contratacion_temporal.expediente_version_integral
    ADD CONSTRAINT expediente_version_integral_origen_version_check
    CHECK (origen_version IN (
        'alta_o2', 'analisis_o3', 'cobertura_o4', 'asignacion_o5',
        'informe_juridico_o5', 'fiscalizacion_o5'
    ));

ALTER TABLE vec_contratacion_temporal.expediente_version_integral
    DROP CONSTRAINT expediente_version_integral_estado_check;
ALTER TABLE vec_contratacion_temporal.expediente_version_integral
    ADD CONSTRAINT expediente_version_integral_estado_check
    CHECK (estado IN ('en_curso', 'completado', 'incidencia', 'cancelado'));

CREATE TABLE vec_contratacion_temporal.reserva_fiscalizacion (
    ambito_hmac text PRIMARY KEY,
    huella_peticion_hmac text NOT NULL,
    operacion text NOT NULL,
    organizacion_ref text NOT NULL,
    expediente_ref text NOT NULL,
    version_expediente numeric(20, 0) NOT NULL,
    actor_ref text NOT NULL,
    perfil_ref text NOT NULL,
    resultado text NOT NULL,
    observaciones text NOT NULL,
    unidad_fiscalizadora_ref text NOT NULL,
    reserva_ref text NOT NULL UNIQUE,
    fiscalizacion_ref text NOT NULL UNIQUE,
    recibo_ref text NOT NULL UNIQUE,
    evento_ref text NOT NULL UNIQUE,
    retorno_ref text UNIQUE,
    expediente_anterior_json jsonb NOT NULL,
    estado text NOT NULL DEFAULT 'reservada',
    reservada_en timestamptz(6) NOT NULL,
    confirmada_en timestamptz(6),
    FOREIGN KEY (expediente_ref, version_expediente)
        REFERENCES vec_contratacion_temporal.expediente_version_integral,
    CHECK (operacion = 'registrar_resultado'),
    CHECK (version_expediente = 5),
    CHECK (resultado IN (
        'favorable', 'favorable_con_observaciones', 'desfavorable'
    )),
    CHECK (observaciones = pg_catalog.btrim(observaciones)),
	CHECK (pg_catalog.char_length(observaciones) <= 2000),
    CHECK (pg_catalog.octet_length(observaciones) <= 8192),
    CHECK (
        (resultado = 'favorable' AND observaciones = '' AND retorno_ref IS NULL)
        OR
        (resultado = 'favorable_con_observaciones' AND
         observaciones <> '' AND retorno_ref IS NULL)
        OR
        (resultado = 'desfavorable' AND
         observaciones <> '' AND retorno_ref IS NOT NULL)
    ),
    CHECK (estado IN ('reservada', 'confirmada')),
    CHECK (pg_catalog.jsonb_typeof(expediente_anterior_json) = 'object'),
    CHECK (reservada_en = pg_catalog.date_trunc('microseconds', reservada_en)),
    CHECK (confirmada_en IS NULL OR
           confirmada_en = pg_catalog.date_trunc('microseconds', confirmada_en)),
    CHECK ((estado = 'reservada' AND confirmada_en IS NULL) OR
           (estado = 'confirmada' AND confirmada_en IS NOT NULL))
);

CREATE TABLE vec_contratacion_temporal.terminal_fiscalizacion (
    ambito_hmac text PRIMARY KEY,
    huella_peticion_hmac text NOT NULL,
    decision_ref text NOT NULL UNIQUE,
    decision_huella_sha256 text NOT NULL,
    consumo_huella_sha256 text NOT NULL UNIQUE,
    auditoria_ref text NOT NULL UNIQUE,
    politica_ref text NOT NULL,
    politica_version numeric(20, 0) NOT NULL,
    politica_huella_sha256 text NOT NULL,
    carga_huella_sha256 text NOT NULL,
    confirmada_en timestamptz(6) NOT NULL,
    FOREIGN KEY (ambito_hmac)
        REFERENCES vec_contratacion_temporal.reserva_fiscalizacion,
    CHECK (decision_huella_sha256 ~ '^[0-9a-f]{64}$'),
    CHECK (consumo_huella_sha256 ~ '^[0-9a-f]{64}$'),
    CHECK (auditoria_ref ~ '^aud_v3_[0-9a-f]{32}$'),
    CHECK (politica_version BETWEEN 1 AND 9007199254740991::numeric),
    CHECK (politica_huella_sha256 ~ '^[0-9a-f]{64}$'),
    CHECK (carga_huella_sha256 ~ '^[0-9a-f]{64}$'),
    CHECK (confirmada_en = pg_catalog.date_trunc('microseconds', confirmada_en))
);

CREATE TABLE vec_contratacion_temporal.retorno_fiscalizacion_unidad (
    retorno_ref text PRIMARY KEY,
    ambito_hmac text NOT NULL UNIQUE,
    expediente_ref text NOT NULL,
    version_expediente numeric(20, 0) NOT NULL,
    unidad_ref text NOT NULL,
    responsable_ref text NOT NULL,
    estado text NOT NULL,
    creada_en timestamptz(6) NOT NULL,
    FOREIGN KEY (ambito_hmac)
        REFERENCES vec_contratacion_temporal.reserva_fiscalizacion,
    FOREIGN KEY (expediente_ref, version_expediente)
        REFERENCES vec_contratacion_temporal.expediente_version_integral,
    CHECK (version_expediente = 6),
    CHECK (estado = 'pendiente'),
    CHECK (creada_en = pg_catalog.date_trunc('microseconds', creada_en))
);

DO $seguridad$
DECLARE
    v_tabla text;
BEGIN
    FOREACH v_tabla IN ARRAY ARRAY[
        'reserva_fiscalizacion',
        'terminal_fiscalizacion',
        'retorno_fiscalizacion_unidad'
    ] LOOP
        EXECUTE pg_catalog.format(
            'ALTER TABLE vec_contratacion_temporal.%I ENABLE ROW LEVEL SECURITY',
            v_tabla
        );
        EXECUTE pg_catalog.format(
            'ALTER TABLE vec_contratacion_temporal.%I FORCE ROW LEVEL SECURITY',
            v_tabla
        );
        EXECUTE pg_catalog.format(
            'CREATE POLICY %I ON vec_contratacion_temporal.%I TO vec_contratacion_temporal_propietario USING (true) WITH CHECK (true)',
            v_tabla || '_propietario', v_tabla
        );
        IF v_tabla <> 'reserva_fiscalizacion' THEN
            EXECUTE pg_catalog.format(
                'CREATE TRIGGER %I BEFORE UPDATE OR DELETE ON vec_contratacion_temporal.%I FOR EACH ROW EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1()',
                v_tabla || '_inmutable', v_tabla
            );
        END IF;
    END LOOP;
END
$seguridad$;

CREATE FUNCTION vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(
    p_documento jsonb,
    p_claves text[]
)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
STRICT
SET search_path = pg_catalog
AS $funcion$
    SELECT pg_catalog.jsonb_typeof(p_documento) = 'object'
       AND ARRAY(
           SELECT clave
             FROM pg_catalog.jsonb_object_keys(p_documento) AS k(clave)
            ORDER BY clave
       ) = ARRAY(
           SELECT clave
             FROM pg_catalog.unnest(p_claves) clave
            ORDER BY clave
       )
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.recibo_fiscalizacion_v1(
    p_ambito_hmac text
)
RETURNS jsonb
LANGUAGE sql
STABLE
SECURITY DEFINER
SET search_path = pg_catalog
SET row_security = 'on'
SET timezone = 'UTC'
AS $funcion$
    SELECT pg_catalog.jsonb_strip_nulls(pg_catalog.jsonb_build_object(
        'operacion', r.operacion,
        'organizacion_ref', r.organizacion_ref,
        'expediente_ref', r.expediente_ref,
        'version_anterior', r.version_expediente,
        'version_resultante', r.version_expediente + 1,
        'resultado', r.resultado,
        'fase_resultante', CASE
            WHEN r.resultado = 'desfavorable' THEN 'subsanacion_unidad'
            ELSE 'fiscalizacion'
        END,
        'estado_resultante', CASE
            WHEN r.resultado = 'desfavorable' THEN 'incidencia'
            ELSE 'en_curso'
        END,
        'recibo_ref', r.recibo_ref,
        'auditoria_ref', t.auditoria_ref,
        'evento_ref', r.evento_ref,
        'actor_ref', r.actor_ref,
        'unidad_retorno_ref', u.unidad_ref,
        'responsable_retorno_ref', u.responsable_ref,
        'registrada_en', t.confirmada_en
    ))
      FROM vec_contratacion_temporal.reserva_fiscalizacion r
      JOIN vec_contratacion_temporal.terminal_fiscalizacion t
        USING (ambito_hmac)
      LEFT JOIN vec_contratacion_temporal.retorno_fiscalizacion_unidad u
        USING (ambito_hmac)
     WHERE r.ambito_hmac = p_ambito_hmac
       AND r.estado = 'confirmada'
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.preparar_fiscalizacion_v1(
    p_operacion jsonb
)
RETURNS TABLE (
    resultado text, expediente_json text, reserva_ref text,
    fiscalizacion_ref text, recibo_ref text, evento_ref text,
    retorno_ref text, ambito_hmac text, huella_peticion_hmac text,
    organizacion_ref text, expediente_ref text, version_expediente bigint,
    actor_ref text, perfil_ref text, resultado_fiscalizacion text,
    observaciones text, estado text, recibo_json jsonb
)
LANGUAGE plpgsql
STABLE
SECURITY DEFINER
SET search_path = pg_catalog
SET row_security = 'on'
SET timezone = 'UTC'
SET lock_timeout = '2s'
AS $funcion$
DECLARE
    v_activo jsonb := p_operacion #> '{sellos_hmac,activo}';
    v_retenido jsonb;
    v_huella_buscada text;
    v_reserva vec_contratacion_temporal.reserva_fiscalizacion%ROWTYPE;
    v_actual record;
    v_encontrada boolean := false;
BEGIN
    IF session_user = current_user
       OR NOT pg_catalog.pg_has_role(
           session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER')
       OR pg_catalog.pg_has_role(
           session_user, 'vec_contratacion_temporal_propietario', 'MEMBER')
       OR pg_catalog.current_setting('transaction_isolation') <> 'serializable'
       OR pg_catalog.current_setting('transaction_read_only') <> 'on'
       OR pg_catalog.pg_column_size(p_operacion) > 65536
       OR NOT vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(
           p_operacion, ARRAY[
               'actor_ref','esquema','expediente_ref','observaciones',
               'operacion','organizacion_ref','perfil_ref',
               'referencias_candidatas','resultado','sellos_hmac',
               'version_expediente'
           ])
	   OR EXISTS (
	       SELECT 1
	         FROM pg_catalog.jsonb_each(p_operacion) AS campo(clave, valor)
	        WHERE valor = 'null'::jsonb
	   )
       OR p_operacion ->> 'esquema' <>
          'vec.contratacion-temporal.preparar-fiscalizacion.v1'
       OR p_operacion ->> 'operacion' <> 'registrar_resultado'
       OR (p_operacion ->> 'version_expediente')::numeric <> 5
       OR p_operacion ->> 'resultado' NOT IN (
           'favorable', 'favorable_con_observaciones', 'desfavorable'
       )
       OR p_operacion ->> 'observaciones' <>
          pg_catalog.btrim(p_operacion ->> 'observaciones')
	   OR pg_catalog.char_length(p_operacion ->> 'observaciones') > 2000
       OR pg_catalog.octet_length(p_operacion ->> 'observaciones') > 8192
       OR (
           (p_operacion ->> 'resultado' = 'favorable' AND
            p_operacion ->> 'observaciones' <> '')
           OR
           (p_operacion ->> 'resultado' IN (
                'favorable_con_observaciones', 'desfavorable'
            ) AND p_operacion ->> 'observaciones' = '')
       )
       OR NOT vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(
           p_operacion -> 'referencias_candidatas',
           ARRAY['evento_ref','fiscalizacion_ref','recibo_ref',
                 'reserva_ref','retorno_ref'])
       OR (
           (p_operacion ->> 'resultado' = 'desfavorable' AND
            coalesce(
                p_operacion #>> '{referencias_candidatas,retorno_ref}', ''
            ) !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$')
           OR
           (p_operacion ->> 'resultado' <> 'desfavorable' AND
            p_operacion #>> '{referencias_candidatas,retorno_ref}' <> '')
       )
       OR NOT vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(
           p_operacion -> 'sellos_hmac', ARRAY['activo','retenidos'])
       OR NOT vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(
           v_activo, ARRAY['ambito_hmac','generacion','huella_peticion_hmac'])
       OR pg_catalog.jsonb_typeof(p_operacion #> '{sellos_hmac,retenidos}')
          <> 'array'
       OR pg_catalog.jsonb_array_length(
           p_operacion #> '{sellos_hmac,retenidos}') > 16
       OR coalesce(v_activo ->> 'ambito_hmac', '') !~
          '^hmac-sha256:vec[.]contratacion-temporal[.]fiscalizacion[.]ambito/v[1-9][0-9]{0,8}:[0-9a-f]{64}$'
       OR coalesce(v_activo ->> 'huella_peticion_hmac', '') !~
          '^hmac-sha256:vec[.]contratacion-temporal[.]fiscalizacion[.]peticion/v[1-9][0-9]{0,8}:[0-9a-f]{64}$'
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.jsonb_array_elements(
                 p_operacion #> '{sellos_hmac,retenidos}') AS e(valor)
            WHERE NOT vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(
                      e.valor,
                      ARRAY['ambito_hmac','generacion','huella_peticion_hmac'])
               OR coalesce(e.valor ->> 'ambito_hmac', '') !~
                  '^hmac-sha256:vec[.]contratacion-temporal[.]fiscalizacion[.]ambito/v[1-9][0-9]{0,8}:[0-9a-f]{64}$'
               OR coalesce(e.valor ->> 'huella_peticion_hmac', '') !~
                  '^hmac-sha256:vec[.]contratacion-temporal[.]fiscalizacion[.]peticion/v[1-9][0-9]{0,8}:[0-9a-f]{64}$'
       )
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.jsonb_each_text(
                 p_operacion -> 'referencias_candidatas') c
            WHERE c.key <> 'retorno_ref'
	      AND coalesce(c.value, '') !~
	          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       )
       OR p_operacion ->> 'organizacion_ref'
          !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_operacion ->> 'expediente_ref'
          !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_operacion ->> 'actor_ref'
          !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_operacion ->> 'perfil_ref'
          !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'preparación de fiscalización no autorizada';
    END IF;

    v_huella_buscada := v_activo ->> 'huella_peticion_hmac';
    SELECT r.*
      INTO v_reserva
      FROM vec_contratacion_temporal.reserva_fiscalizacion r
     WHERE r.ambito_hmac = v_activo ->> 'ambito_hmac';
    v_encontrada := FOUND;
    IF NOT v_encontrada THEN
        FOR v_retenido IN
            SELECT valor
              FROM pg_catalog.jsonb_array_elements(
                  p_operacion #> '{sellos_hmac,retenidos}'
              ) AS e(valor)
        LOOP
            v_huella_buscada := v_retenido ->> 'huella_peticion_hmac';
            SELECT r.*
              INTO v_reserva
              FROM vec_contratacion_temporal.reserva_fiscalizacion r
             WHERE r.ambito_hmac = v_retenido ->> 'ambito_hmac';
            IF FOUND THEN
                v_encontrada := true;
                EXIT;
            END IF;
        END LOOP;
    END IF;

    IF v_encontrada THEN
        IF v_reserva.huella_peticion_hmac IS DISTINCT FROM v_huella_buscada
           OR v_reserva.operacion IS DISTINCT FROM p_operacion ->> 'operacion'
           OR v_reserva.organizacion_ref IS DISTINCT FROM
              p_operacion ->> 'organizacion_ref'
           OR v_reserva.expediente_ref IS DISTINCT FROM
              p_operacion ->> 'expediente_ref'
           OR v_reserva.version_expediente IS DISTINCT FROM
              (p_operacion ->> 'version_expediente')::numeric
           OR v_reserva.actor_ref IS DISTINCT FROM p_operacion ->> 'actor_ref'
           OR v_reserva.perfil_ref IS DISTINCT FROM p_operacion ->> 'perfil_ref'
           OR v_reserva.resultado IS DISTINCT FROM p_operacion ->> 'resultado'
           OR v_reserva.observaciones IS DISTINCT FROM
              p_operacion ->> 'observaciones' THEN
            resultado := 'idempotencia_reutilizada';
        ELSIF v_reserva.estado = 'confirmada' THEN
            resultado := 'confirmada';
        ELSE
            resultado := 'reutilizada';
        END IF;
    ELSE
        SELECT a.version, v.agregado_json
          INTO STRICT v_actual
          FROM vec_contratacion_temporal.expediente_integral_actual a
          JOIN vec_contratacion_temporal.expediente_version_integral v
            USING (expediente_ref, version)
         WHERE a.expediente_ref = p_operacion ->> 'expediente_ref';
        IF v_actual.version <> 5
           OR v_actual.agregado_json ->> 'organizacion_ref' <>
              p_operacion ->> 'organizacion_ref'
           OR v_actual.agregado_json ->> 'fase_actual' <> 'informe_juridico'
           OR v_actual.agregado_json ->> 'estado_actual' <> 'en_curso'
           OR NOT (v_actual.agregado_json ? 'asignacion')
           OR NOT (v_actual.agregado_json ? 'informe_juridico')
           OR v_actual.agregado_json ? 'fiscalizacion' THEN
            RAISE EXCEPTION USING ERRCODE = '40001',
                MESSAGE = 'expediente no disponible para fiscalización';
        END IF;
        v_reserva.ambito_hmac := v_activo ->> 'ambito_hmac';
        v_reserva.huella_peticion_hmac :=
            v_activo ->> 'huella_peticion_hmac';
        v_reserva.operacion := 'registrar_resultado';
        v_reserva.organizacion_ref := p_operacion ->> 'organizacion_ref';
        v_reserva.expediente_ref := p_operacion ->> 'expediente_ref';
        v_reserva.version_expediente := 5;
        v_reserva.actor_ref := p_operacion ->> 'actor_ref';
        v_reserva.perfil_ref := p_operacion ->> 'perfil_ref';
        v_reserva.resultado := p_operacion ->> 'resultado';
        v_reserva.observaciones := p_operacion ->> 'observaciones';
        v_reserva.reserva_ref :=
            p_operacion #>> '{referencias_candidatas,reserva_ref}';
        v_reserva.fiscalizacion_ref :=
            p_operacion #>> '{referencias_candidatas,fiscalizacion_ref}';
        v_reserva.recibo_ref :=
            p_operacion #>> '{referencias_candidatas,recibo_ref}';
        v_reserva.evento_ref :=
            p_operacion #>> '{referencias_candidatas,evento_ref}';
        v_reserva.retorno_ref := nullif(
            p_operacion #>> '{referencias_candidatas,retorno_ref}', '');
        v_reserva.expediente_anterior_json := v_actual.agregado_json;
        v_reserva.estado := 'preparada';
        resultado := 'preparada';
    END IF;

    RETURN QUERY SELECT resultado, v_reserva.expediente_anterior_json::text,
        v_reserva.reserva_ref, v_reserva.fiscalizacion_ref,
        v_reserva.recibo_ref, v_reserva.evento_ref,
        coalesce(v_reserva.retorno_ref, ''),
        v_reserva.ambito_hmac, v_reserva.huella_peticion_hmac,
        v_reserva.organizacion_ref, v_reserva.expediente_ref,
        v_reserva.version_expediente::bigint, v_reserva.actor_ref,
        v_reserva.perfil_ref, v_reserva.resultado,
        v_reserva.observaciones, v_reserva.estado,
        CASE WHEN v_reserva.estado = 'confirmada'
             THEN vec_contratacion_temporal.recibo_fiscalizacion_v1(
                      v_reserva.ambito_hmac)
        END;
EXCEPTION
    WHEN invalid_text_representation OR numeric_value_out_of_range THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'preparación de fiscalización inválida';
END
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.confirmar_fiscalizacion_v1(
    p_operacion jsonb,
    p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea,
    p_persona_version numeric, p_perfil_version numeric,
    p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea
)
RETURNS TABLE (recibo_json jsonb)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
SET row_security = 'on'
SET timezone = 'UTC'
SET lock_timeout = '2s'
AS $funcion$
DECLARE
    r vec_contratacion_temporal.reserva_fiscalizacion%ROWTYPE;
    v_actual record;
    v_consumo record;
    v_decision jsonb;
    v_ahora timestamptz(6);
    v_carga_huella text;
    v_agregado_huella text;
    v_prueba bytea;
    v_payload_evento bytea;
    v_anterior text;
    v_secuencia numeric;
    v_actuacion jsonb;
    v_vinculo jsonb;
    v_fiscalizacion jsonb;
    v_retorno jsonb;
    v_expediente_esperado jsonb;
    v_fase_destino text;
    v_estado_destino text;
    v_unidad_retorno text;
    v_responsable_retorno text;
    v_informe_ref text;
    v_documento_ref text;
    v_contexto_canonico bytea;
    v_contexto_huella text;
    v_observaciones_huella text;
    v_reserva_persistida boolean := false;
BEGIN
    IF session_user = current_user
       OR NOT pg_catalog.pg_has_role(
           session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER')
       OR pg_catalog.pg_has_role(
           session_user, 'vec_contratacion_temporal_propietario', 'MEMBER')
       OR pg_catalog.current_setting('transaction_isolation') <> 'serializable'
       OR pg_catalog.current_setting('transaction_read_only') <> 'off'
       OR pg_catalog.pg_column_size(p_operacion) > 3145728
       OR NOT vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(
           p_operacion, ARRAY[
               'actor_ref','actuacion','ambito_idempotencia_hmac',
               'autorizacion','esquema','expediente_anterior',
               'expediente_ref','expediente_siguiente',
               'huella_peticion_hmac','instante_efecto','observaciones',
               'operacion','organizacion_ref','perfil_ref','politica',
               'referencias','reserva_ref','resultado','version_anterior'
           ])
	   OR EXISTS (
	       SELECT 1
	         FROM pg_catalog.jsonb_each(p_operacion) AS campo(clave, valor)
	        WHERE valor = 'null'::jsonb
	   )
       OR p_operacion ->> 'esquema' <>
          'vec.contratacion-temporal.confirmar-fiscalizacion.v1'
       OR p_operacion ->> 'operacion' <> 'registrar_resultado'
       OR (p_operacion ->> 'version_anterior')::numeric <> 5
       OR p_operacion ->> 'resultado' NOT IN (
           'favorable', 'favorable_con_observaciones', 'desfavorable'
       )
       OR p_operacion ->> 'observaciones' <>
          pg_catalog.btrim(p_operacion ->> 'observaciones')
       OR pg_catalog.char_length(p_operacion ->> 'observaciones') > 2000
       OR pg_catalog.octet_length(p_operacion ->> 'observaciones') > 8192
       OR (
           (p_operacion ->> 'resultado' = 'favorable' AND
            p_operacion ->> 'observaciones' <> '')
           OR
           (p_operacion ->> 'resultado' IN (
                'favorable_con_observaciones', 'desfavorable'
            ) AND p_operacion ->> 'observaciones' = '')
       )
       OR p_operacion ->> 'ambito_idempotencia_hmac' !~
          '^hmac-sha256:vec[.]contratacion-temporal[.]fiscalizacion[.]ambito/v[1-9][0-9]{0,8}:[0-9a-f]{64}$'
       OR p_operacion ->> 'huella_peticion_hmac' !~
          '^hmac-sha256:vec[.]contratacion-temporal[.]fiscalizacion[.]peticion/v[1-9][0-9]{0,8}:[0-9a-f]{64}$'
       OR NOT vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(
           p_operacion -> 'referencias',
           ARRAY['evento_ref','fiscalizacion_ref','recibo_ref',
                 'reserva_ref','retorno_ref'])
       OR NOT vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(
           p_operacion -> 'politica',
           ARRAY['accion','definicion_huella_sha256','definicion_ref',
                 'definicion_version','evaluada_en','finalidad',
                 'unidad_fiscalizadora_ref','valida_hasta'])
       OR NOT vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(
           p_operacion -> 'autorizacion',
           ARRAY['accion','contexto_recurso_huella_sha256',
                 'decision_canonica_hex','decision_huella_sha256',
                 'decision_ref','finalidad','motivo_canonico_hex',
                 'perfil_activo_ref','perfil_version','persona_version',
                 'principal_id','recurso_ref'])
	   OR EXISTS (
	       SELECT 1
	         FROM pg_catalog.jsonb_each(p_operacion -> 'referencias')
	              AS campo(clave, valor)
	        WHERE valor = 'null'::jsonb
	   )
	   OR EXISTS (
	       SELECT 1
	         FROM pg_catalog.jsonb_each(p_operacion -> 'politica')
	              AS campo(clave, valor)
	        WHERE valor = 'null'::jsonb
	   )
	   OR EXISTS (
	       SELECT 1
	         FROM pg_catalog.jsonb_each(p_operacion -> 'autorizacion')
	              AS campo(clave, valor)
	        WHERE valor = 'null'::jsonb
	   )
       OR p_operacion ->> 'reserva_ref' <>
          p_operacion #>> '{referencias,reserva_ref}'
       OR (
           (p_operacion ->> 'resultado' = 'desfavorable' AND
            coalesce(
                p_operacion #>> '{referencias,retorno_ref}', ''
            ) !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$')
           OR
           (p_operacion ->> 'resultado' <> 'desfavorable' AND
            p_operacion #>> '{referencias,retorno_ref}' <> '')
       )
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.jsonb_each_text(
                 p_operacion -> 'referencias') c
            WHERE c.key <> 'retorno_ref'
	      AND coalesce(c.value, '') !~
	          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       )
       OR p_operacion ->> 'organizacion_ref'
          !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_operacion ->> 'expediente_ref'
          !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_operacion ->> 'actor_ref'
          !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_operacion ->> 'perfil_ref'
          !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_operacion #>> '{politica,unidad_fiscalizadora_ref}'
          !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_operacion #>> '{politica,definicion_ref}'
          !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR (p_operacion #>> '{politica,definicion_version}')::numeric
          NOT BETWEEN 1 AND 9007199254740991::numeric
       OR p_operacion #>> '{politica,definicion_huella_sha256}'
          !~ '^[0-9a-f]{64}$'
       OR p_operacion #>> '{politica,accion}' <>
          'contratacion_temporal.fiscalizacion.registrar'
       OR p_operacion #>> '{politica,finalidad}' <>
          'gestionar_contratacion_temporal'
       OR p_operacion #>> '{autorizacion,accion}' <>
          'contratacion_temporal.fiscalizacion.registrar'
       OR p_operacion #>> '{autorizacion,finalidad}' <>
          'gestionar_contratacion_temporal'
       OR p_operacion #>> '{autorizacion,recurso_ref}' <>
          p_operacion ->> 'expediente_ref'
       OR p_operacion #>> '{autorizacion,principal_id}' <>
          p_operacion ->> 'actor_ref'
       OR p_operacion #>> '{autorizacion,perfil_activo_ref}' <>
          p_operacion ->> 'perfil_ref'
       OR p_operacion #>> '{autorizacion,decision_canonica_hex}' <>
          pg_catalog.encode(p_decision, 'hex')
       OR p_operacion #>> '{autorizacion,motivo_canonico_hex}' <>
          pg_catalog.encode(p_motivo, 'hex')
       OR (p_operacion #>> '{autorizacion,persona_version}')::numeric <>
          p_persona_version
       OR (p_operacion #>> '{autorizacion,perfil_version}')::numeric <>
          p_perfil_version
       OR p_operacion #>> '{autorizacion,decision_huella_sha256}' <>
          pg_catalog.encode(pg_catalog.sha256(p_decision), 'hex') THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'confirmación de fiscalización no autorizada';
    END IF;

    v_decision := pg_catalog.convert_from(p_decision, 'UTF8')::jsonb;
    v_carga_huella := pg_catalog.encode(pg_catalog.sha256(
        pg_catalog.convert_to(p_operacion::text, 'UTF8')), 'hex');

    SELECT reserva.* INTO r
      FROM vec_contratacion_temporal.reserva_fiscalizacion reserva
     WHERE reserva.ambito_hmac =
           p_operacion ->> 'ambito_idempotencia_hmac'
     FOR UPDATE;
    v_reserva_persistida := FOUND;

    IF v_reserva_persistida THEN
        IF r.huella_peticion_hmac <>
              p_operacion ->> 'huella_peticion_hmac'
           OR r.operacion <> p_operacion ->> 'operacion'
           OR r.organizacion_ref <> p_operacion ->> 'organizacion_ref'
           OR r.expediente_ref <> p_operacion ->> 'expediente_ref'
           OR r.version_expediente <>
              (p_operacion ->> 'version_anterior')::numeric
           OR r.actor_ref <> p_operacion ->> 'actor_ref'
           OR r.perfil_ref <> p_operacion ->> 'perfil_ref'
           OR r.resultado <> p_operacion ->> 'resultado'
           OR r.observaciones <> p_operacion ->> 'observaciones'
           OR r.reserva_ref <> p_operacion ->> 'reserva_ref'
           OR r.fiscalizacion_ref <>
              p_operacion #>> '{referencias,fiscalizacion_ref}'
           OR r.recibo_ref <> p_operacion #>> '{referencias,recibo_ref}'
           OR r.evento_ref <> p_operacion #>> '{referencias,evento_ref}'
           OR coalesce(r.retorno_ref, '') <>
              p_operacion #>> '{referencias,retorno_ref}'
           OR r.expediente_anterior_json <>
              p_operacion -> 'expediente_anterior' THEN
            RAISE EXCEPTION USING ERRCODE = '23505',
                MESSAGE = 'clave de fiscalización reutilizada';
        END IF;
        IF r.estado = 'confirmada' THEN
            RETURN QUERY SELECT
                vec_contratacion_temporal.recibo_fiscalizacion_v1(
                    r.ambito_hmac
                );
            RETURN;
        END IF;
        RAISE EXCEPTION USING ERRCODE = '40001',
            MESSAGE = 'reserva de fiscalización no terminal';
    END IF;

    r.ambito_hmac := p_operacion ->> 'ambito_idempotencia_hmac';
    r.huella_peticion_hmac := p_operacion ->> 'huella_peticion_hmac';
    r.operacion := 'registrar_resultado';
    r.organizacion_ref := p_operacion ->> 'organizacion_ref';
    r.expediente_ref := p_operacion ->> 'expediente_ref';
    r.version_expediente := 5;
    r.actor_ref := p_operacion ->> 'actor_ref';
    r.perfil_ref := p_operacion ->> 'perfil_ref';
    r.resultado := p_operacion ->> 'resultado';
    r.observaciones := p_operacion ->> 'observaciones';
    r.unidad_fiscalizadora_ref :=
        p_operacion #>> '{politica,unidad_fiscalizadora_ref}';
    r.reserva_ref := p_operacion #>> '{referencias,reserva_ref}';
    r.fiscalizacion_ref :=
        p_operacion #>> '{referencias,fiscalizacion_ref}';
    r.recibo_ref := p_operacion #>> '{referencias,recibo_ref}';
    r.evento_ref := p_operacion #>> '{referencias,evento_ref}';
    r.retorno_ref := nullif(
        p_operacion #>> '{referencias,retorno_ref}', '');
    r.expediente_anterior_json := p_operacion -> 'expediente_anterior';

    SELECT v.* INTO STRICT v_actual
      FROM vec_contratacion_temporal.expediente_integral_actual a
      JOIN vec_contratacion_temporal.expediente_version_integral v
        USING (expediente_ref, version)
     WHERE a.expediente_ref = r.expediente_ref
     FOR UPDATE OF a, v;

    IF v_actual.version <> 5
       OR v_actual.agregado_json <> r.expediente_anterior_json
       OR v_actual.agregado_json ->> 'organizacion_ref' <>
          r.organizacion_ref
       OR v_actual.agregado_json ->> 'fase_actual' <> 'informe_juridico'
       OR v_actual.agregado_json ->> 'estado_actual' <> 'en_curso'
       OR NOT (v_actual.agregado_json ? 'asignacion')
       OR NOT (v_actual.agregado_json ? 'informe_juridico')
       OR v_actual.agregado_json ? 'fiscalizacion' THEN
        RAISE EXCEPTION USING ERRCODE = '40001',
            MESSAGE = 'CAS de fiscalización perdido';
    END IF;

    v_unidad_retorno :=
        v_actual.agregado_json #>> '{asignacion,unidad_ref}';
    v_responsable_retorno :=
        v_actual.agregado_json #>> '{asignacion,responsable_ref}';
    v_informe_ref :=
        v_actual.agregado_json #>> '{informe_juridico,informe_ref}';
    v_documento_ref :=
        v_actual.agregado_json #>> '{informe_juridico,documento_ref}';
    v_ahora := pg_catalog.date_trunc(
        'microseconds', pg_catalog.clock_timestamp());

    IF v_unidad_retorno
          !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR v_responsable_retorno
          !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR v_informe_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR v_documento_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR (p_operacion #>> '{politica,evaluada_en}')::timestamptz >
          (p_operacion ->> 'instante_efecto')::timestamptz
       OR v_ahora < (p_operacion ->> 'instante_efecto')::timestamptz
       OR v_ahora >=
          (p_operacion #>> '{politica,valida_hasta}')::timestamptz THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'coordenadas o vigencia de fiscalización inválidas';
    END IF;

    IF r.resultado = 'desfavorable' THEN
        v_fase_destino := 'subsanacion_unidad';
        v_estado_destino := 'incidencia';
        v_retorno := pg_catalog.jsonb_build_object(
            'retorno_ref', r.retorno_ref,
            'unidad_ref', v_unidad_retorno,
            'responsable_ref', v_responsable_retorno,
            'estado', 'pendiente',
            'creado_en', p_operacion -> 'instante_efecto'
        );
    ELSE
        v_fase_destino := 'fiscalizacion';
        v_estado_destino := 'en_curso';
        v_retorno := NULL;
    END IF;

    v_actuacion := pg_catalog.jsonb_build_object(
        'secuencia', 6,
        'version_expediente', 6,
        'accion_clave', 'contratacion_temporal.fiscalizacion.registrar',
        'actor_ref', r.actor_ref,
        'unidad_ref', r.unidad_fiscalizadora_ref,
        'recibo_ref', r.recibo_ref,
        'realizada_en', p_operacion -> 'instante_efecto',
        'fase_origen', v_actual.agregado_json -> 'fase_actual',
        'fase_destino', v_fase_destino,
        'estado_origen', v_actual.agregado_json -> 'estado_actual',
        'estado_destino', v_estado_destino,
        'observaciones', r.observaciones,
        'documentos_ref', pg_catalog.jsonb_build_array(v_documento_ref)
    );
    IF r.observaciones = '' THEN
        v_actuacion := v_actuacion - 'observaciones';
    END IF;

    v_vinculo := pg_catalog.jsonb_build_object(
        'secuencia', 6,
        'version_expediente', 6,
        'accion_clave', 'contratacion_temporal.fiscalizacion.registrar',
        'fase_destino', v_fase_destino,
        'estado_destino', v_estado_destino,
        'recibo_ref', r.recibo_ref,
        'fiscalizacion_ref', r.fiscalizacion_ref,
        'resultado', r.resultado,
        'unidad_fiscalizadora_ref', r.unidad_fiscalizadora_ref,
        'informe_juridico_ref', v_informe_ref,
        'documento_informe_ref', v_documento_ref
    );
    IF r.resultado = 'desfavorable' THEN
        v_vinculo := v_vinculo || pg_catalog.jsonb_build_object(
            'retorno_ref', r.retorno_ref,
            'unidad_retorno_ref', v_unidad_retorno,
            'responsable_retorno_ref', v_responsable_retorno
        );
    END IF;

    v_fiscalizacion := pg_catalog.jsonb_build_object(
        'fiscalizacion_ref', r.fiscalizacion_ref,
        'resultado', r.resultado,
        'unidad_fiscalizadora_ref', r.unidad_fiscalizadora_ref,
        'informe_juridico_ref', v_informe_ref,
        'documento_informe_ref', v_documento_ref,
        'observaciones', r.observaciones,
        'fiscalizada_en', p_operacion -> 'instante_efecto',
        'actuacion_registro', v_vinculo
    );
    IF r.observaciones = '' THEN
        v_fiscalizacion := v_fiscalizacion - 'observaciones';
    END IF;
    IF v_retorno IS NOT NULL THEN
        v_fiscalizacion := v_fiscalizacion ||
            pg_catalog.jsonb_build_object('retorno', v_retorno);
    END IF;

    v_expediente_esperado := v_actual.agregado_json ||
        pg_catalog.jsonb_build_object(
            'version', 6,
            'fase_actual', v_fase_destino,
            'estado_actual', v_estado_destino,
            'actualizado_en', p_operacion -> 'instante_efecto',
            'actuaciones', (v_actual.agregado_json -> 'actuaciones') ||
                pg_catalog.jsonb_build_array(v_actuacion),
            'fiscalizacion', v_fiscalizacion
        );
    IF pg_catalog.jsonb_array_length(
           v_actual.agregado_json -> 'actuaciones') <> 5
       OR p_operacion -> 'actuacion' IS DISTINCT FROM v_actuacion
       OR p_operacion -> 'expediente_siguiente' IS DISTINCT FROM
          v_expediente_esperado THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'proyección de fiscalización divergente';
    END IF;
    v_observaciones_huella := pg_catalog.encode(pg_catalog.sha256(
        pg_catalog.convert_to(r.observaciones, 'UTF8')), 'hex');
    v_contexto_canonico := pg_catalog.convert_to(
        '{"ambitos":{"estado_previo":"' ||
        (v_actual.agregado_json ->> 'estado_actual') ||
        '","expediente_ref":"' || r.expediente_ref ||
        '","fase_previa":"' ||
        (v_actual.agregado_json ->> 'fase_actual') ||
        '","organizacion_ref":"' || r.organizacion_ref ||
        '"},"atributos":{"ambito_idempotencia_hmac":"' || r.ambito_hmac ||
        '","documento_informe_ref":"' || v_documento_ref ||
        '","huella_peticion_hmac":"' || r.huella_peticion_hmac ||
        '","informe_juridico_ref":"' || v_informe_ref ||
        '","observaciones_huella_sha256":"' || v_observaciones_huella ||
        '","politica_huella_sha256":"' ||
        (p_operacion #>> '{politica,definicion_huella_sha256}') ||
        '","politica_ref":"' ||
        (p_operacion #>> '{politica,definicion_ref}') ||
        '","politica_version":"' ||
        (p_operacion #>> '{politica,definicion_version}') ||
        '","responsable_asignado_ref":"' || v_responsable_retorno ||
        '","resultado":"' || r.resultado ||
        '","unidad_asignada_ref":"' || v_unidad_retorno ||
        '","unidad_fiscalizadora_ref":"' || r.unidad_fiscalizadora_ref ||
        '","version_expediente":"5"}}',
        'UTF8'
    );
    v_contexto_huella := pg_catalog.encode(
        pg_catalog.sha256(v_contexto_canonico), 'hex');

    IF p_operacion #>> '{autorizacion,contexto_recurso_huella_sha256}' <>
          v_contexto_huella
       OR v_decision ->> 'contexto_recurso_huella_sha256' <>
          v_contexto_huella
       OR v_decision ->> 'principal_id' <> r.actor_ref
       OR v_decision ->> 'perfil_activo_ref' <> r.perfil_ref
       OR v_decision ->> 'recurso_ref' <> r.expediente_ref
       OR v_decision ->> 'accion' <>
          'contratacion_temporal.fiscalizacion.registrar'
       OR v_decision ->> 'modulo_id' <> 'contratacion_temporal'
       OR v_decision ->> 'tipo_recurso' <>
          'fiscalizacion_contratacion_temporal'
       OR v_decision ->> 'finalidad' <> 'gestionar_contratacion_temporal'
       OR v_decision ->> 'decision_ref' <>
          p_operacion #>> '{autorizacion,decision_ref}'
       OR p_operacion #>> '{politica,unidad_fiscalizadora_ref}' <>
          r.unidad_fiscalizadora_ref THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'contexto autorizado de fiscalización divergente';
    END IF;

    SELECT * INTO STRICT v_consumo
      FROM vec_autorizacion_atestada_v3
           .registrar_y_consumir_fiscalizacion_v3_atestada(
          p_capacidad, p_decision, p_motivo, p_contexto,
          p_persona_version, p_perfil_version, p_payload, p_sobre,
          p_evidencia, p_raiz
      );
    IF v_consumo.decision_ref <>
          p_operacion #>> '{autorizacion,decision_ref}'
       OR v_consumo.efecto_ref <> r.expediente_ref
       OR v_consumo.huella_efecto_sha256 <> v_contexto_huella
       OR coalesce(v_consumo.auditoria_ref, '') !~
          '^aud_v3_[0-9a-f]{32}$'
       OR v_consumo.consumo_nuevo IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consumo de autorización de fiscalización divergente';
    END IF;

    v_ahora := pg_catalog.date_trunc(
        'microseconds', pg_catalog.clock_timestamp());
    IF v_ahora < (p_operacion ->> 'instante_efecto')::timestamptz
       OR v_ahora >=
          (p_operacion #>> '{politica,valida_hasta}')::timestamptz THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'vigencia de fiscalización agotada';
    END IF;

    INSERT INTO vec_contratacion_temporal.reserva_fiscalizacion (
        ambito_hmac, huella_peticion_hmac, operacion, organizacion_ref,
        expediente_ref, version_expediente, actor_ref, perfil_ref,
        resultado, observaciones, unidad_fiscalizadora_ref,
        reserva_ref, fiscalizacion_ref, recibo_ref, evento_ref, retorno_ref,
        expediente_anterior_json, reservada_en
    ) VALUES (
        r.ambito_hmac, r.huella_peticion_hmac, r.operacion,
        r.organizacion_ref, r.expediente_ref, r.version_expediente,
        r.actor_ref, r.perfil_ref, r.resultado, r.observaciones,
        r.unidad_fiscalizadora_ref, r.reserva_ref, r.fiscalizacion_ref,
        r.recibo_ref, r.evento_ref, r.retorno_ref,
        r.expediente_anterior_json, v_ahora
    );

    v_agregado_huella := pg_catalog.encode(pg_catalog.sha256(
        pg_catalog.convert_to(
            (p_operacion -> 'expediente_siguiente')::text, 'UTF8')), 'hex');
    v_prueba := pg_catalog.convert_to(
        'VEC-CT-EXPEDIENTE-FISCALIZACION-V1' || chr(10) ||
        r.expediente_ref || chr(10) || '6' || chr(10) ||
        v_agregado_huella || chr(10) || r.reserva_ref || chr(10) ||
        r.recibo_ref || chr(10) || v_consumo.decision_ref || chr(10) ||
        v_ahora::text,
        'UTF8'
    );
    INSERT INTO vec_contratacion_temporal.expediente_version_integral (
        expediente_ref, version, agregado_json, agregado_json_huella_sha256,
        prueba_canonica, prueba_huella_sha256, flujo_ref, flujo_version,
        flujo_huella_sha256, fase_clave, estado, origen_version,
        operacion_ref, registrada_en
    ) VALUES (
        r.expediente_ref, 6, p_operacion -> 'expediente_siguiente',
        v_agregado_huella, v_prueba,
        pg_catalog.encode(pg_catalog.sha256(v_prueba), 'hex'),
        v_actual.flujo_ref, v_actual.flujo_version,
        v_actual.flujo_huella_sha256, v_fase_destino, v_estado_destino,
        'fiscalizacion_o5', r.reserva_ref, v_ahora
    );

    UPDATE vec_contratacion_temporal.expediente_integral_actual
       SET version = 6,
           actualizada_en = v_ahora,
           operacion_ref = r.reserva_ref
     WHERE expediente_ref = r.expediente_ref
       AND version = 5;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING ERRCODE = '40001',
            MESSAGE = 'CAS final de fiscalización perdido';
    END IF;

    v_prueba := pg_catalog.convert_to(
        'VEC-CT-ACTUACION-FISCALIZACION-V1' || chr(10) ||
        (p_operacion -> 'actuacion')::text || chr(10) || r.recibo_ref ||
        chr(10) || v_ahora::text,
        'UTF8'
    );
    INSERT INTO vec_contratacion_temporal.actuacion_expediente_integral (
        expediente_ref, secuencia, version_expediente, operacion_ref,
        recibo_ref, actuacion_json, actuacion_json_huella_sha256,
        prueba_canonica, prueba_huella_sha256, registrada_en
    ) VALUES (
        r.expediente_ref, 6, 6, r.reserva_ref, r.recibo_ref,
        p_operacion -> 'actuacion',
        pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
            (p_operacion -> 'actuacion')::text, 'UTF8')), 'hex'),
        v_prueba, pg_catalog.encode(pg_catalog.sha256(v_prueba), 'hex'),
        v_ahora
    );

    IF r.resultado = 'desfavorable' THEN
        INSERT INTO vec_contratacion_temporal.retorno_fiscalizacion_unidad (
            retorno_ref, ambito_hmac, expediente_ref, version_expediente,
            unidad_ref, responsable_ref, estado, creada_en
        ) VALUES (
            r.retorno_ref, r.ambito_hmac, r.expediente_ref, 6,
            v_unidad_retorno, v_responsable_retorno, 'pendiente',
            (p_operacion ->> 'instante_efecto')::timestamptz
        );
    END IF;

    SELECT secuencia_outbox, cabeza_outbox_sha256
      INTO STRICT v_secuencia, v_anterior
      FROM vec_contratacion_temporal.control_cadenas_expediente_integral
     WHERE control_id
     FOR UPDATE;
    IF v_secuencia >= 9007199254740991::numeric THEN
        RAISE EXCEPTION USING ERRCODE = '22003',
            MESSAGE = 'límite de outbox alcanzado';
    END IF;
    v_secuencia := v_secuencia + 1;
    v_payload_evento := pg_catalog.convert_to(
        (
            pg_catalog.jsonb_build_object(
                'esquema',
                    'vec.contratacion-temporal.fiscalizacion-registrada.v1',
                'expediente_ref', r.expediente_ref,
                'version_resultante', 6,
                'fiscalizacion_ref', r.fiscalizacion_ref,
                'resultado', r.resultado,
                'unidad_fiscalizadora_ref', r.unidad_fiscalizadora_ref,
                'recibo_ref', r.recibo_ref
            ) ||
            CASE WHEN r.resultado = 'desfavorable'
                 THEN pg_catalog.jsonb_build_object(
                     'retorno_ref', r.retorno_ref,
                     'unidad_retorno_ref', v_unidad_retorno,
                     'responsable_retorno_ref', v_responsable_retorno,
                     'estado_retorno', 'pendiente'
                 )
                 ELSE '{}'::jsonb
            END
        )::text,
        'UTF8'
    );
    INSERT INTO vec_contratacion_temporal.outbox_expediente_integral (
        evento_ref, secuencia, operacion_ref, expediente_ref,
        version_expediente, tipo_evento, payload_canonico,
        payload_huella_sha256, anterior_sha256, huella_sha256, registrada_en
    ) VALUES (
        r.evento_ref, v_secuencia, r.reserva_ref, r.expediente_ref, 6,
        'contratacion_temporal.fiscalizacion_registrada', v_payload_evento,
        pg_catalog.encode(pg_catalog.sha256(v_payload_evento), 'hex'),
        v_anterior,
        pg_catalog.encode(pg_catalog.sha256(
            v_anterior::bytea || v_payload_evento), 'hex'),
        v_ahora
    );
    UPDATE vec_contratacion_temporal.control_cadenas_expediente_integral
       SET secuencia_outbox = v_secuencia,
           cabeza_outbox_sha256 = pg_catalog.encode(pg_catalog.sha256(
               v_anterior::bytea || v_payload_evento), 'hex'),
           actualizada_en = v_ahora
     WHERE control_id;

    INSERT INTO vec_contratacion_temporal.terminal_fiscalizacion (
        ambito_hmac, huella_peticion_hmac, decision_ref,
        decision_huella_sha256, consumo_huella_sha256, auditoria_ref,
        politica_ref, politica_version, politica_huella_sha256,
        carga_huella_sha256, confirmada_en
    ) VALUES (
        r.ambito_hmac, r.huella_peticion_hmac, v_consumo.decision_ref,
        p_operacion #>> '{autorizacion,decision_huella_sha256}',
        v_consumo.consumo_huella_sha256, v_consumo.auditoria_ref,
        p_operacion #>> '{politica,definicion_ref}',
        (p_operacion #>> '{politica,definicion_version}')::numeric,
        p_operacion #>> '{politica,definicion_huella_sha256}',
        v_carga_huella, v_ahora
    );
    UPDATE vec_contratacion_temporal.reserva_fiscalizacion
       SET estado = 'confirmada',
           confirmada_en = v_ahora
     WHERE ambito_hmac = r.ambito_hmac
       AND estado = 'reservada';
    IF NOT FOUND THEN
        RAISE EXCEPTION USING ERRCODE = '40001',
            MESSAGE = 'reserva de fiscalización perdida';
    END IF;

    RETURN QUERY SELECT
        vec_contratacion_temporal.recibo_fiscalizacion_v1(r.ambito_hmac);
EXCEPTION
    WHEN invalid_text_representation OR datetime_field_overflow OR
         numeric_value_out_of_range OR character_not_in_repertoire THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'entrada de fiscalización inválida';
END
$funcion$;

REVOKE ALL ON TABLE
    vec_contratacion_temporal.reserva_fiscalizacion,
    vec_contratacion_temporal.terminal_fiscalizacion,
    vec_contratacion_temporal.retorno_fiscalizacion_unidad
FROM PUBLIC, vec_contratacion_temporal_ejecutor;
REVOKE ALL ON FUNCTION
    vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(jsonb,text[]),
    vec_contratacion_temporal.recibo_fiscalizacion_v1(text),
    vec_contratacion_temporal.preparar_fiscalizacion_v1(jsonb),
    vec_contratacion_temporal.confirmar_fiscalizacion_v1(
        jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
FROM PUBLIC, vec_contratacion_temporal_ejecutor;
GRANT EXECUTE ON FUNCTION
    vec_contratacion_temporal.preparar_fiscalizacion_v1(jsonb),
    vec_contratacion_temporal.confirmar_fiscalizacion_v1(
        jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
TO vec_contratacion_temporal_ejecutor;

COMMIT;
