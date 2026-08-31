BEGIN; SET LOCAL ROLE vec_ejecucion_documental_v4_propietario; SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
-- La reserva ancla el consumo previo y conserva la tupla reproducible del puerto.
CREATE TABLE vec_ejecucion_documental_v4.reserva_efecto_generacion_documental_v1 (
    reserva_ref text PRIMARY KEY, orden_ref text NOT NULL UNIQUE,
    decision_ref text NOT NULL UNIQUE, efecto_ref text NOT NULL UNIQUE,
    huella_decision_sha256 text NOT NULL, huella_plan_autorizacion_sha256 text NOT NULL,
    huella_orden_sha256 text NOT NULL, huella_capacidad_sha256 text NOT NULL,
    capacidad_clave_id text NOT NULL, capacidad_version numeric(20, 0) NOT NULL,
    capacidad_nonce text NOT NULL,
    huella_plan_efecto_sha256 text NOT NULL, huella_manifiesto_sha256 text NOT NULL,
    huella_tupla_sha256 text NOT NULL UNIQUE, tupla_canonica jsonb NOT NULL,
    contexto_canonico jsonb NOT NULL, manifiesto_canonico jsonb NOT NULL,
    correlacion_ref text NOT NULL, reservada_en timestamptz(6) NOT NULL,
    FOREIGN KEY (decision_ref) REFERENCES
        vec_ejecucion_documental_v4.consumo_decision_atomico(decision_ref),
    FOREIGN KEY (efecto_ref) REFERENCES
        vec_ejecucion_documental_v4.consumo_decision_atomico(efecto_ref),
    FOREIGN KEY (orden_ref) REFERENCES
        vec_ejecucion_documental_v4.consumo_decision_atomico(orden_ref),
    FOREIGN KEY (capacidad_clave_id, capacidad_version, capacidad_nonce)
        REFERENCES vec_ejecucion_documental_v4.consumo_capacidad(clave_id, version, nonce),
    CHECK (vec_ejecucion_documental_v4.texto_tecnico_valido(reserva_ref, 512)
        AND vec_ejecucion_documental_v4.texto_tecnico_valido(orden_ref, 512)
        AND vec_ejecucion_documental_v4.texto_tecnico_valido(decision_ref, 512)
        AND vec_ejecucion_documental_v4.texto_tecnico_valido(efecto_ref, 512)
        AND vec_ejecucion_documental_v4.texto_tecnico_valido(capacidad_clave_id, 512)
        AND capacidad_nonce ~ '^[0-9a-f]{64}$'
        AND vec_ejecucion_documental_v4.texto_tecnico_valido(correlacion_ref, 512)),
    CHECK (vec_ejecucion_documental_v4.huella_sha256_valida(huella_decision_sha256)
        AND vec_ejecucion_documental_v4.huella_sha256_valida(huella_plan_autorizacion_sha256)
        AND vec_ejecucion_documental_v4.huella_sha256_valida(huella_orden_sha256)
        AND vec_ejecucion_documental_v4.huella_sha256_valida(huella_capacidad_sha256)
        AND vec_ejecucion_documental_v4.huella_sha256_valida(huella_plan_efecto_sha256)
        AND vec_ejecucion_documental_v4.huella_sha256_valida(huella_manifiesto_sha256)
        AND vec_ejecucion_documental_v4.huella_sha256_valida(huella_tupla_sha256)),
    CHECK (jsonb_typeof(tupla_canonica) = 'object'),
    CHECK (jsonb_typeof(contexto_canonico) = 'object'),
    CHECK (jsonb_typeof(manifiesto_canonico) = 'object') );
-- Cada paso tiene una reserva y, como maximo, un terminal; todo es adicion.
CREATE TABLE vec_ejecucion_documental_v4.paso_efecto_generacion_documental_v1 (
    reserva_ref text NOT NULL REFERENCES
        vec_ejecucion_documental_v4.reserva_efecto_generacion_documental_v1(reserva_ref),
    posicion integer NOT NULL, paso_ref text NOT NULL,
    huella_paso_sha256 text NOT NULL, referencia_logica text NOT NULL,
    clave_idempotencia text NOT NULL, formato text NOT NULL,
    zona text NOT NULL, mime text NOT NULL, tamano bigint NOT NULL,
    huella_contenido_sha256 text NOT NULL, estado text NOT NULL,
    objeto_ref text, objeto_version text, conector_id text,
    evidencia_operacion_ref text, incidente_ref text,
    contenido_guardado_canonico jsonb, evidencia_operacion_canonica jsonb,
    resultado_en timestamptz(6), registrada_en timestamptz(6) NOT NULL,
    PRIMARY KEY (reserva_ref, paso_ref, estado), UNIQUE (reserva_ref, posicion, estado),
    UNIQUE (reserva_ref, referencia_logica, estado),
    UNIQUE (reserva_ref, clave_idempotencia, estado), CHECK (posicion BETWEEN 1 AND 256),
    CHECK (vec_ejecucion_documental_v4.texto_tecnico_valido(paso_ref, 256)
        AND vec_ejecucion_documental_v4.texto_tecnico_valido(referencia_logica, 512)
        AND vec_ejecucion_documental_v4.texto_tecnico_valido(clave_idempotencia, 512)
        AND vec_ejecucion_documental_v4.texto_tecnico_valido(formato, 128) AND zona = 'admitida'
        AND vec_ejecucion_documental_v4.texto_tecnico_valido(mime, 255)), CHECK (tamano > 0
        AND vec_ejecucion_documental_v4.huella_sha256_valida(huella_paso_sha256)
        AND vec_ejecucion_documental_v4.huella_sha256_valida(huella_contenido_sha256)),
    CHECK (estado IN ('reservado', 'confirmado', 'indeterminado')), CHECK ( estado = 'reservado'
        AND objeto_ref IS NULL AND objeto_version IS NULL
        AND conector_id IS NULL AND evidencia_operacion_ref IS NULL
        AND incidente_ref IS NULL AND resultado_en IS NULL OR estado = 'confirmado'
        AND vec_ejecucion_documental_v4.texto_tecnico_valido(objeto_ref, 512)
        AND vec_ejecucion_documental_v4.texto_tecnico_valido(objeto_version, 256)
        AND vec_ejecucion_documental_v4.texto_tecnico_valido(conector_id, 128)
        AND vec_ejecucion_documental_v4.texto_tecnico_valido(evidencia_operacion_ref, 512)
        AND incidente_ref IS NULL AND resultado_en IS NOT NULL
        AND jsonb_typeof(contenido_guardado_canonico) = 'object'
        AND jsonb_typeof(evidencia_operacion_canonica) = 'object' OR estado = 'indeterminado'
        AND objeto_ref IS NULL AND objeto_version IS NULL
        AND conector_id IS NULL AND evidencia_operacion_ref IS NULL
        AND vec_ejecucion_documental_v4.texto_tecnico_valido(incidente_ref, 512)
        AND resultado_en IS NOT NULL ), CHECK (estado = 'confirmado' OR
        contenido_guardado_canonico IS NULL AND evidencia_operacion_canonica IS NULL) );
CREATE UNIQUE INDEX paso_efecto_generacion_documental_v1_terminal_unico ON vec_ejecucion_documental_v4.paso_efecto_generacion_documental_v1(reserva_ref, paso_ref)
    WHERE estado IN ('confirmado', 'indeterminado');
CREATE TABLE vec_ejecucion_documental_v4.auditoria_efecto_generacion_documental_v1 (
    auditoria_ref text PRIMARY KEY, secuencia numeric(20, 0) NOT NULL UNIQUE,
    reserva_ref text NOT NULL REFERENCES
        vec_ejecucion_documental_v4.reserva_efecto_generacion_documental_v1(reserva_ref),
    paso_ref text, accion text NOT NULL, resultado text NOT NULL,
    correlacion_ref text NOT NULL, ocurrida_en timestamptz(6) NOT NULL,
    huella_anterior_sha256 text NOT NULL, huella_registro_sha256 text NOT NULL UNIQUE,
    CHECK (vec_ejecucion_documental_v4.texto_tecnico_valido(auditoria_ref, 512)),
    CHECK (paso_ref IS NULL OR vec_ejecucion_documental_v4.texto_tecnico_valido(paso_ref, 256)),
    CHECK ((accion = 'reservar_efecto_generacion_documental' AND
            resultado = 'reservado' AND paso_ref IS NULL)
        OR (accion = 'confirmar_paso_generacion_documental' AND
            resultado = 'confirmado' AND paso_ref IS NOT NULL)
        OR (accion = 'marcar_paso_generacion_documental_indeterminado' AND
            resultado = 'indeterminado' AND paso_ref IS NOT NULL)),
    CHECK (vec_ejecucion_documental_v4.huella_sha256_valida(huella_anterior_sha256)),
    CHECK (vec_ejecucion_documental_v4.huella_sha256_valida(huella_registro_sha256)) );
CREATE TABLE vec_ejecucion_documental_v4.evento_outbox_efecto_generacion_documental_v1 (
    evento_ref text PRIMARY KEY, secuencia numeric(20, 0) NOT NULL UNIQUE,
    tipo text NOT NULL, estado text NOT NULL, reserva_ref text NOT NULL REFERENCES
        vec_ejecucion_documental_v4.reserva_efecto_generacion_documental_v1(reserva_ref),
    paso_ref text, auditoria_ref text NOT NULL UNIQUE REFERENCES
        vec_ejecucion_documental_v4.auditoria_efecto_generacion_documental_v1(auditoria_ref),
    huella_auditoria_sha256 text NOT NULL, correlacion_ref text NOT NULL,
    registrada_en timestamptz(6) NOT NULL, huella_registro_sha256 text NOT NULL UNIQUE,
    CHECK (vec_ejecucion_documental_v4.texto_tecnico_valido(evento_ref, 512)),
    CHECK (estado = 'pendiente'),
    CHECK ((tipo = 'efecto_generacion_documental_reservado' AND paso_ref IS NULL)
        OR (tipo IN ('paso_generacion_documental_confirmado',
            'paso_generacion_documental_indeterminado') AND paso_ref IS NOT NULL)),
    CHECK (paso_ref IS NULL OR vec_ejecucion_documental_v4.texto_tecnico_valido(paso_ref, 256)),
    CHECK (vec_ejecucion_documental_v4.huella_sha256_valida(huella_auditoria_sha256)),
    CHECK (vec_ejecucion_documental_v4.huella_sha256_valida(huella_registro_sha256)) );
-- Auxiliar privado: enlaza el cambio a 000001; sin cargas libres ni concesion.
CREATE FUNCTION vec_ejecucion_documental_v4.registrar_cambio_efecto_generacion_documental_v1(
    p_reserva_ref text, p_paso_ref text, p_accion text, p_resultado text, p_instante timestamptz )
RETURNS void LANGUAGE plpgsql VOLATILE SET search_path = pg_catalog, pg_temp AS $funcion$ DECLARE
    reserva record; v_secuencia numeric(20, 0); v_huella_anterior text;
    v_auditoria_ref text; v_evento_ref text; v_tipo_evento text;
    v_huella_auditoria text; v_huella_evento text; BEGIN IF p_instante IS NULL OR NOT (
        p_accion = 'reservar_efecto_generacion_documental'
        AND p_resultado = 'reservado' AND p_paso_ref IS NULL
        OR p_accion = 'confirmar_paso_generacion_documental'
        AND p_resultado = 'confirmado' AND p_paso_ref IS NOT NULL
        OR p_accion = 'marcar_paso_generacion_documental_indeterminado'
        AND p_resultado = 'indeterminado' AND p_paso_ref IS NOT NULL ) THEN
        RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'cambio documental no valido'; END IF;
    SELECT r.* INTO STRICT reserva
      FROM vec_ejecucion_documental_v4.reserva_efecto_generacion_documental_v1 AS r
     WHERE r.reserva_ref = p_reserva_ref; SELECT c.ultima_secuencia + 1, c.ultima_huella_sha256
      INTO v_secuencia, v_huella_anterior
      FROM vec_ejecucion_documental_v4.control_cadena_auditoria AS c
     WHERE c.control_id = true FOR UPDATE; IF NOT FOUND OR v_secuencia > 18446744073709551615 THEN
        RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'cadena de auditoria no disponible';
    END IF; v_auditoria_ref := 'auditoria:efecto-generacion-documental:v1:' ||
        encode(sha256(convert_to(concat_ws(E'\n', p_reserva_ref,
            COALESCE(p_paso_ref, ''), p_accion, v_secuencia::text), 'UTF8')), 'hex');
    v_evento_ref := 'evento:efecto-generacion-documental:v1:' ||
        encode(sha256(convert_to(concat_ws(E'\n', p_reserva_ref,
            COALESCE(p_paso_ref, ''), p_resultado, v_secuencia::text), 'UTF8')), 'hex');
    v_tipo_evento := CASE p_resultado WHEN 'reservado' THEN 'efecto_generacion_documental_reservado'
        WHEN 'confirmado' THEN 'paso_generacion_documental_confirmado'
        ELSE 'paso_generacion_documental_indeterminado' END;
    v_huella_auditoria := encode(sha256(convert_to(concat_ws(E'\n',
        v_auditoria_ref, v_secuencia::text, reserva.decision_ref,
        reserva.efecto_ref, reserva.huella_plan_efecto_sha256,
        reserva.huella_manifiesto_sha256, COALESCE(p_paso_ref, ''), p_accion,
        p_resultado, reserva.correlacion_ref,
        to_char(p_instante AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'), v_huella_anterior
    ), 'UTF8')), 'hex'); v_huella_evento := encode(sha256(convert_to(concat_ws(E'\n',
        v_evento_ref, v_secuencia::text, v_tipo_evento, 'pendiente',
        p_reserva_ref, COALESCE(p_paso_ref, ''), v_auditoria_ref,
        v_huella_auditoria, reserva.correlacion_ref,
        to_char(p_instante AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"') ), 'UTF8')), 'hex');
    INSERT INTO vec_ejecucion_documental_v4.auditoria_efecto_generacion_documental_v1 (
        auditoria_ref, secuencia, reserva_ref, paso_ref, accion, resultado,
        correlacion_ref, ocurrida_en, huella_anterior_sha256, huella_registro_sha256 ) VALUES (
        v_auditoria_ref, v_secuencia, p_reserva_ref, p_paso_ref, p_accion,
        p_resultado, reserva.correlacion_ref, p_instante, v_huella_anterior, v_huella_auditoria );
    INSERT INTO vec_ejecucion_documental_v4.evento_outbox_efecto_generacion_documental_v1 (
        evento_ref, secuencia, tipo, estado, reserva_ref, paso_ref,
        auditoria_ref, huella_auditoria_sha256, correlacion_ref,
        registrada_en, huella_registro_sha256 ) VALUES (
        v_evento_ref, v_secuencia, v_tipo_evento, 'pendiente', p_reserva_ref,
        p_paso_ref, v_auditoria_ref, v_huella_auditoria,
        reserva.correlacion_ref, p_instante, v_huella_evento );
    UPDATE vec_ejecucion_documental_v4.control_cadena_auditoria SET ultima_secuencia = v_secuencia,
           ultima_huella_sha256 = v_huella_auditoria
     WHERE control_id = true AND ultima_secuencia = v_secuencia - 1
       AND ultima_huella_sha256 = v_huella_anterior; IF NOT FOUND THEN
        RAISE EXCEPTION USING ERRCODE = '40001', MESSAGE = 'CAS de auditoria perdido'; END IF; END
$funcion$;
CREATE FUNCTION vec_ejecucion_documental_v4.reservar_efecto_generacion_documental_v1(
    p_contexto jsonb, p_manifiesto jsonb ) RETURNS TABLE (
    resultado text, reserva_ref text, efecto_ref text,
    huella_decision_sha256 text, huella_plan_efecto_sha256 text,
    huella_manifiesto_sha256 text, repetida boolean, pasos jsonb )
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path = pg_catalog, pg_temp AS $funcion$
DECLARE paso record; existente record; autoridad record; capacidad_actual record;
    v_anterior text := ''; v_canonico bytea := ''::bytea; v_pasos jsonb := '[]'::jsonb;
    v_manifiesto jsonb; v_contexto jsonb; v_tupla jsonb;
    v_huella_paso text; v_huella_manifiesto text; v_huella_plan text;
    v_huella_tupla text; v_reserva_ref text; v_reserva_calculada text;
    v_creada boolean := false; v_cantidad integer; v_ahora timestamptz(6) := clock_timestamp(); BEGIN
    IF p_contexto IS NULL OR jsonb_typeof(p_contexto) <> 'object'
       OR pg_column_size(p_contexto) > 262144
       OR (SELECT count(*) FROM jsonb_object_keys(p_contexto)) <> 25 OR NOT (p_contexto ?& ARRAY[
           'esquema', 'operacion_ref', 'correlacion_ref', 'autorizacion_ref',
           'finalidad', 'clasificacion', 'accion_negocio', 'accion_tecnica',
           'carga_ref', 'sujeto_seudonimo_hmac', 'recurso_ref', 'modulo_id',
           'tipo_recurso', 'huella_recurso_sha256', 'huella_solicitud_hmac',
           'efecto_ref', 'huella_plan_efecto_sha256',
           'huella_manifiesto_sha256', 'huella_paso_sha256', 'paso_ref',
           'objeto_vinculado_ref', 'objeto_vinculado_version',
           'huella_decision_sha256', 'verificada_en', 'valida_hasta' ]) OR EXISTS (
           SELECT 1 FROM jsonb_object_keys(p_contexto) AS clave
            WHERE jsonb_typeof(p_contexto -> clave) <> 'string'
       ) OR p_manifiesto IS NULL OR jsonb_typeof(p_manifiesto) <> 'object'
       OR pg_column_size(p_manifiesto) > 2097152
       OR (SELECT count(*) FROM jsonb_object_keys(p_manifiesto)) <> 9 OR NOT (p_manifiesto ?& ARRAY[
           'esquema', 'plantilla_id', 'plantilla_version', 'modulo_id',
           'tipo_documental', 'huella_plantilla_sha256', 'permiso_generar',
           'huella_manifiesto_sha256', 'pasos'
       ]) OR EXISTS (SELECT 1 FROM jsonb_object_keys(p_manifiesto) AS clave WHERE clave NOT IN ('plantilla_version', 'pasos') AND jsonb_typeof(p_manifiesto -> clave) <> 'string') OR jsonb_typeof(p_manifiesto -> 'plantilla_version') <> 'number'
       OR jsonb_typeof(p_manifiesto -> 'pasos') <> 'array'
       OR jsonb_array_length(p_manifiesto -> 'pasos') NOT BETWEEN 1 AND 256 THEN
        RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'reserva documental invalida'; END IF;
    IF p_contexto ->> 'esquema' <> 'vec.almacen.contexto-operacion.v1'
       OR p_contexto ->> 'accion_tecnica' <> 'escribir' OR p_contexto ->> 'objeto_vinculado_ref' <> ''
       OR p_contexto ->> 'objeto_vinculado_version' <> ''
       OR p_manifiesto ->> 'esquema' <> 'vec.documentos.manifiesto-generacion.v1'
       OR p_contexto ->> 'modulo_id' <> p_manifiesto ->> 'modulo_id'
       OR p_contexto ->> 'accion_negocio' <> p_manifiesto ->> 'permiso_generar'
       OR p_contexto ->> 'huella_manifiesto_sha256' <> p_manifiesto ->> 'huella_manifiesto_sha256'
       OR (p_manifiesto ->> 'plantilla_version')::numeric <> trunc(
          (p_manifiesto ->> 'plantilla_version')::numeric)
       OR (p_manifiesto ->> 'plantilla_version')::numeric NOT BETWEEN 1 AND 9223372036854775807
       OR EXISTS ( SELECT 1 FROM unnest(ARRAY[
               p_contexto ->> 'operacion_ref', p_contexto ->> 'correlacion_ref',
               p_contexto ->> 'autorizacion_ref', p_contexto ->> 'carga_ref',
               p_contexto ->> 'recurso_ref', p_contexto ->> 'efecto_ref' ]) AS valor WHERE
               vec_ejecucion_documental_v4.texto_tecnico_valido(valor, 512) IS NOT TRUE
       ) OR vec_ejecucion_documental_v4.texto_tecnico_valido(p_contexto ->> 'finalidad', 1024) IS NOT TRUE
       OR vec_ejecucion_documental_v4.texto_tecnico_valido(p_contexto ->> 'clasificacion', 256) IS NOT TRUE
       OR vec_ejecucion_documental_v4.texto_tecnico_valido(p_contexto ->> 'accion_negocio', 256) IS NOT TRUE
       OR vec_ejecucion_documental_v4.texto_tecnico_valido(
           p_contexto ->> 'modulo_id', 128) IS NOT TRUE
       OR vec_ejecucion_documental_v4.texto_tecnico_valido(
           p_contexto ->> 'tipo_recurso', 128) IS NOT TRUE
       OR (p_contexto ->> 'sujeto_seudonimo_hmac') !~
          '^hmac-sha256:[^:[:space:]*]{1,64}:[0-9a-f]{64}$'
       OR (p_contexto ->> 'huella_solicitud_hmac') !~
          '^hmac-sha256:[^:[:space:]*]{1,64}:[0-9a-f]{64}$' OR EXISTS ( SELECT 1 FROM unnest(ARRAY[
               'huella_recurso_sha256', 'huella_plan_efecto_sha256',
               'huella_manifiesto_sha256', 'huella_paso_sha256', 'huella_decision_sha256'
           ]) AS clave WHERE vec_ejecucion_documental_v4.huella_sha256_valida( p_contexto ->> clave
           ) IS NOT TRUE ) OR vec_ejecucion_documental_v4.instante_valido(
           p_contexto ->> 'verificada_en') IS NOT TRUE
       OR vec_ejecucion_documental_v4.instante_valido( p_contexto ->> 'valida_hasta') IS NOT TRUE
       OR (p_contexto ->> 'verificada_en')::timestamptz > v_ahora
       OR v_ahora >= (p_contexto ->> 'valida_hasta')::timestamptz THEN
        RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'contexto documental no valido'; END IF;
    IF EXISTS ( SELECT 1 FROM unnest(ARRAY[
            p_manifiesto ->> 'plantilla_id', p_manifiesto ->> 'modulo_id',
            p_manifiesto ->> 'tipo_documental' ]) AS valor WHERE
            vec_ejecucion_documental_v4.texto_tecnico_valido(valor, 128) IS NOT TRUE
    ) OR vec_ejecucion_documental_v4.texto_tecnico_valido(
        p_manifiesto ->> 'permiso_generar', 256) IS NOT TRUE
       OR vec_ejecucion_documental_v4.huella_sha256_valida(
        p_manifiesto ->> 'huella_plantilla_sha256') IS NOT TRUE
       OR vec_ejecucion_documental_v4.huella_sha256_valida(
        p_manifiesto ->> 'huella_manifiesto_sha256') IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'manifiesto documental no valido';
    END IF; FOR paso IN SELECT valor, ordinalidad::integer AS posicion
          FROM jsonb_array_elements(p_manifiesto -> 'pasos')
               WITH ORDINALITY AS elemento(valor, ordinalidad) ORDER BY ordinalidad LOOP
        IF jsonb_typeof(paso.valor) <> 'object'
           OR (SELECT count(*) FROM jsonb_object_keys(paso.valor)) <> 9 OR NOT (paso.valor ?& ARRAY[
               'paso_ref', 'referencia_logica', 'clave_idempotencia',
               'formato', 'zona', 'mime', 'tamano', 'huella_sha256', 'huella_paso_sha256'
           ]) OR jsonb_typeof(paso.valor -> 'tamano') <> 'number' OR EXISTS (
               SELECT 1 FROM jsonb_object_keys(paso.valor) AS clave WHERE clave <> 'tamano'
                  AND jsonb_typeof(paso.valor -> clave) <> 'string'
           ) OR (paso.valor ->> 'tamano')::numeric <> trunc( (paso.valor ->> 'tamano')::numeric)
           OR (paso.valor ->> 'tamano')::numeric NOT BETWEEN 1 AND 9223372036854775807
           OR paso.valor ->> 'zona' <> 'admitida' OR vec_ejecucion_documental_v4.texto_tecnico_valido(
               paso.valor ->> 'referencia_logica', 512) IS NOT TRUE
           OR vec_ejecucion_documental_v4.texto_tecnico_valido(
               paso.valor ->> 'clave_idempotencia', 512) IS NOT TRUE
           OR vec_ejecucion_documental_v4.texto_tecnico_valido(
               paso.valor ->> 'formato', 128) IS NOT TRUE
           OR vec_ejecucion_documental_v4.texto_tecnico_valido(
               paso.valor ->> 'mime', 255) IS NOT TRUE
           OR NOT ((paso.valor ->> 'formato' = 'pdf' AND paso.valor ->> 'mime' = 'application/pdf')
               OR (paso.valor ->> 'formato' = 'docx' AND paso.valor ->> 'mime' = 'application/vnd.openxmlformats-officedocument.wordprocessingml.document'))
           OR vec_ejecucion_documental_v4.huella_sha256_valida(
               paso.valor ->> 'huella_sha256') IS NOT TRUE OR (v_anterior <> '' AND
               paso.valor ->> 'referencia_logica' <= v_anterior) THEN
            RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'paso documental no valido'; END IF;
        v_canonico := vec_ejecucion_documental_v4.encuadrar_capacidad(
            'vec.documentos.manifiesto-generacion.paso.v1') ||
            vec_ejecucion_documental_v4.encuadrar_capacidad(
                p_manifiesto ->> 'huella_plantilla_sha256') ||
            vec_ejecucion_documental_v4.encuadrar_capacidad( p_manifiesto ->> 'permiso_generar') ||
            vec_ejecucion_documental_v4.encuadrar_capacidad( paso.valor ->> 'referencia_logica') ||
            vec_ejecucion_documental_v4.encuadrar_capacidad( paso.valor ->> 'clave_idempotencia') ||
            vec_ejecucion_documental_v4.encuadrar_capacidad( paso.valor ->> 'formato') ||
            vec_ejecucion_documental_v4.encuadrar_capacidad( paso.valor ->> 'zona') ||
            vec_ejecucion_documental_v4.encuadrar_capacidad( paso.valor ->> 'mime') ||
            vec_ejecucion_documental_v4.encuadrar_capacidad(
                (paso.valor ->> 'tamano')::bigint::text) ||
            vec_ejecucion_documental_v4.encuadrar_capacidad( paso.valor ->> 'huella_sha256');
        v_huella_paso := encode(sha256(v_canonico), 'hex');
        IF paso.valor ->> 'huella_paso_sha256' <> v_huella_paso
           OR paso.valor ->> 'paso_ref' <> 'generar_documento_' || v_huella_paso THEN
            RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'huella de paso documental no valida';
        END IF; v_pasos := v_pasos || jsonb_build_array(jsonb_build_object(
            'paso_ref', paso.valor ->> 'paso_ref',
            'referencia_logica', paso.valor ->> 'referencia_logica',
            'clave_idempotencia', paso.valor ->> 'clave_idempotencia',
            'formato', paso.valor ->> 'formato', 'zona', paso.valor ->> 'zona',
            'mime', paso.valor ->> 'mime', 'tamano', (paso.valor ->> 'tamano')::bigint,
            'huella_sha256', paso.valor ->> 'huella_sha256', 'huella_paso_sha256', v_huella_paso ));
        v_anterior := paso.valor ->> 'referencia_logica'; END LOOP;
    IF (SELECT count(DISTINCT valor ->> 'clave_idempotencia')
          FROM jsonb_array_elements(v_pasos) AS valor) <> jsonb_array_length(v_pasos)
       OR (SELECT count(DISTINCT valor ->> 'paso_ref')
          FROM jsonb_array_elements(v_pasos) AS valor) <> jsonb_array_length(v_pasos) THEN
        RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'manifiesto documental duplicado';
    END IF; v_canonico := vec_ejecucion_documental_v4.encuadrar_capacidad(
        p_manifiesto ->> 'esquema') ||
        vec_ejecucion_documental_v4.encuadrar_capacidad(p_manifiesto ->> 'plantilla_id') ||
        vec_ejecucion_documental_v4.encuadrar_capacidad(
            (p_manifiesto ->> 'plantilla_version')::bigint::text) ||
        vec_ejecucion_documental_v4.encuadrar_capacidad(p_manifiesto ->> 'modulo_id') ||
        vec_ejecucion_documental_v4.encuadrar_capacidad(p_manifiesto ->> 'tipo_documental') ||
        vec_ejecucion_documental_v4.encuadrar_capacidad(
            p_manifiesto ->> 'huella_plantilla_sha256') ||
        vec_ejecucion_documental_v4.encuadrar_capacidad(p_manifiesto ->> 'permiso_generar') ||
        vec_ejecucion_documental_v4.encuadrar_capacidad(jsonb_array_length(v_pasos)::text);
    FOR paso IN SELECT e.valor FROM jsonb_array_elements(v_pasos) AS e(valor) LOOP
        v_canonico := v_canonico ||
            vec_ejecucion_documental_v4.encuadrar_capacidad(paso.valor ->> 'paso_ref') ||
            vec_ejecucion_documental_v4.encuadrar_capacidad(paso.valor ->> 'huella_paso_sha256') ||
            vec_ejecucion_documental_v4.encuadrar_capacidad(paso.valor ->> 'referencia_logica') ||
            vec_ejecucion_documental_v4.encuadrar_capacidad(paso.valor ->> 'clave_idempotencia') ||
            vec_ejecucion_documental_v4.encuadrar_capacidad(paso.valor ->> 'formato') ||
            vec_ejecucion_documental_v4.encuadrar_capacidad(paso.valor ->> 'zona') ||
            vec_ejecucion_documental_v4.encuadrar_capacidad(paso.valor ->> 'mime') ||
            vec_ejecucion_documental_v4.encuadrar_capacidad((paso.valor ->> 'tamano')::bigint::text) ||
            vec_ejecucion_documental_v4.encuadrar_capacidad(paso.valor ->> 'huella_sha256');
    END LOOP; v_huella_manifiesto := encode(sha256(v_canonico), 'hex');
    IF v_huella_manifiesto <> p_manifiesto ->> 'huella_manifiesto_sha256'
       OR p_contexto ->> 'paso_ref' <> v_pasos -> 0 ->> 'paso_ref'
       OR p_contexto ->> 'huella_paso_sha256' <> v_pasos -> 0 ->> 'huella_paso_sha256' THEN
        RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'manifiesto documental no coincide con el contexto';
    END IF; v_manifiesto := jsonb_build_object( 'esquema', p_manifiesto ->> 'esquema',
        'plantilla_id', p_manifiesto ->> 'plantilla_id',
        'plantilla_version', (p_manifiesto ->> 'plantilla_version')::bigint,
        'modulo_id', p_manifiesto ->> 'modulo_id',
        'tipo_documental', p_manifiesto ->> 'tipo_documental',
        'huella_plantilla_sha256', p_manifiesto ->> 'huella_plantilla_sha256',
        'permiso_generar', p_manifiesto ->> 'permiso_generar',
        'huella_manifiesto_sha256', v_huella_manifiesto, 'pasos', v_pasos );
    v_contexto := p_contexto; v_canonico := ''::bytea; FOREACH v_anterior IN ARRAY ARRAY[
        p_contexto ->> 'esquema', p_contexto ->> 'autorizacion_ref',
        p_contexto ->> 'huella_decision_sha256', p_contexto ->> 'accion_negocio',
        p_contexto ->> 'recurso_ref', p_contexto ->> 'huella_recurso_sha256',
        p_contexto ->> 'finalidad', p_contexto ->> 'correlacion_ref',
        p_contexto ->> 'operacion_ref', p_contexto ->> 'carga_ref',
        p_contexto ->> 'clasificacion', p_contexto ->> 'sujeto_seudonimo_hmac',
        p_contexto ->> 'huella_solicitud_hmac', p_contexto ->> 'efecto_ref', '', '',
        v_huella_manifiesto ] LOOP v_canonico := v_canonico ||
            vec_ejecucion_documental_v4.encuadrar_capacidad(v_anterior); END LOOP;
    FOR paso IN SELECT e.valor FROM jsonb_array_elements(v_pasos) AS e(valor) LOOP
        v_canonico := v_canonico ||
            vec_ejecucion_documental_v4.encuadrar_capacidad(paso.valor ->> 'paso_ref') ||
            vec_ejecucion_documental_v4.encuadrar_capacidad('escribir') ||
            vec_ejecucion_documental_v4.encuadrar_capacidad(paso.valor ->> 'huella_paso_sha256');
    END LOOP; v_huella_plan := encode(sha256(v_canonico), 'hex');
    IF v_huella_plan <> p_contexto ->> 'huella_plan_efecto_sha256' THEN
        RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'plan documental no canonico'; END IF;
    SELECT orden.*, atestacion.aplicacion_registro, atestacion.decision_canonica,
           atestacion.huella_plan_sha256 AS huella_plan_autorizacion,
           consumo.huella_aplicacion_sha256 AS huella_aplicacion_consumida,
           atestacion.registrada_en AS atestacion_registrada_en,
           consumo.consumida_en AS decision_consumida_en,
           capacidad.clave_id AS capacidad_clave_id, capacidad.version AS capacidad_version,
           capacidad.nonce AS capacidad_nonce, capacidad.huella_capacidad_sha256, capacidad.capacidad,
           capacidad.emitida_en AS capacidad_emitida_en, capacidad.expira_en AS capacidad_expira_en,
           capacidad.consumida_en AS capacidad_consumida_en INTO STRICT autoridad
      FROM vec_ejecucion_documental_v4.orden_generacion_documental AS orden
      JOIN vec_ejecucion_documental_v4.atestacion_pdp AS atestacion
        ON atestacion.decision_ref = orden.decision_ref AND atestacion.efecto_ref = orden.efecto_ref
       AND atestacion.huella_plan_sha256 = orden.huella_plan_sha256
       AND atestacion.huella_decision_sha256 = orden.huella_decision_sha256
      JOIN vec_ejecucion_documental_v4.consumo_decision_atomico AS consumo
        ON consumo.decision_ref = orden.decision_ref
       AND consumo.efecto_ref = orden.efecto_ref AND consumo.orden_ref = orden.orden_ref
       AND consumo.huella_decision_sha256 = orden.huella_decision_sha256
       AND consumo.huella_aplicacion_sha256 = orden.huella_aplicacion_sha256
      JOIN vec_ejecucion_documental_v4.consumo_capacidad AS capacidad
        ON capacidad.decision_ref = orden.decision_ref
     WHERE orden.decision_ref = p_contexto ->> 'autorizacion_ref'
        OR orden.efecto_ref = p_contexto ->> 'efecto_ref' ORDER BY orden.orden_ref
     FOR UPDATE OF orden, atestacion, consumo, capacidad;
    IF autoridad.estado <> 'pendiente_generacion'
       OR autoridad.orden_ref <> p_contexto ->> 'efecto_ref'
       OR autoridad.decision_ref <> p_contexto ->> 'autorizacion_ref'
       OR autoridad.efecto_ref <> p_contexto ->> 'efecto_ref'
       OR autoridad.huella_decision_sha256 <> p_contexto ->> 'huella_decision_sha256'
       OR autoridad.huella_plan_sha256 <> autoridad.huella_plan_autorizacion
       OR autoridad.huella_aplicacion_sha256 <> autoridad.huella_aplicacion_consumida
       OR autoridad.aplicacion_registro ->> 'decision_ref' <> autoridad.decision_ref
       OR autoridad.aplicacion_registro ->> 'efecto_ref' <> autoridad.efecto_ref
       OR autoridad.aplicacion_registro ->> 'huella_decision_sha256' <>
          autoridad.huella_decision_sha256
       OR autoridad.aplicacion_registro ->> 'huella_plan_sha256' <> autoridad.huella_plan_sha256
       OR autoridad.aplicacion_registro ->> 'huella_solicitud_aplicacion_sha256' <>
          autoridad.huella_aplicacion_sha256
       OR autoridad.aplicacion_registro ->> 'correlacion_ref' <> p_contexto ->> 'correlacion_ref'
       OR autoridad.correlacion_ref <> p_contexto ->> 'correlacion_ref'
       OR autoridad.aplicacion_registro ->> 'solicitada_en' <>
          to_char(autoridad.solicitada_en AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
       OR autoridad.aplicacion_registro ->> 'recurso_ref' <> p_contexto ->> 'recurso_ref'
       OR autoridad.aplicacion_registro ->> 'modulo_id' <> p_contexto ->> 'modulo_id'
       OR autoridad.aplicacion_registro ->> 'tipo_recurso' <> p_contexto ->> 'tipo_recurso'
       OR autoridad.aplicacion_registro ->> 'huella_recurso_sha256' <>
          p_contexto ->> 'huella_recurso_sha256'
       OR autoridad.aplicacion_registro ->> 'finalidad' <> p_contexto ->> 'finalidad'
       OR autoridad.aplicacion_registro ->> 'verificada_en' <> p_contexto ->> 'verificada_en'
       OR autoridad.aplicacion_registro ->> 'valida_hasta' <> p_contexto ->> 'valida_hasta'
       OR encode(sha256(convert_to(autoridad.capacidad::text, 'UTF8')), 'hex') <>
          autoridad.huella_capacidad_sha256
       OR autoridad.capacidad ->> 'clave_id' <> autoridad.capacidad_clave_id
       OR (autoridad.capacidad ->> 'clave_version')::numeric <> autoridad.capacidad_version
       OR autoridad.capacidad ->> 'nonce' <> autoridad.capacidad_nonce
       OR autoridad.capacidad ->> 'huella_decision_sha256' <> autoridad.huella_decision_sha256
       OR autoridad.capacidad ->> 'audiencia' <> 'vec_ejecucion_documental_v4.ejecutar_plan_atestado'
       OR (autoridad.capacidad ->> 'emitida_en')::timestamptz <> autoridad.capacidad_emitida_en
       OR (autoridad.capacidad ->> 'expira_en')::timestamptz <> autoridad.capacidad_expira_en
       OR autoridad.capacidad_consumida_en < autoridad.capacidad_emitida_en
       OR autoridad.capacidad_consumida_en >= autoridad.capacidad_expira_en
       OR autoridad.registrada_en <> autoridad.atestacion_registrada_en
       OR autoridad.registrada_en <> autoridad.decision_consumida_en
       OR autoridad.registrada_en <> autoridad.capacidad_consumida_en THEN
        RAISE EXCEPTION USING ERRCODE = '23514',
            MESSAGE = 'autoridad documental no coincide con la orden atestada'; END IF;
    v_tupla := jsonb_build_object( 'esquema', 'vec.documentos.registro-efectos-generacion.tupla.v1',
        'orden_ref', autoridad.orden_ref, 'decision_ref', p_contexto ->> 'autorizacion_ref',
        'efecto_ref', p_contexto ->> 'efecto_ref',
        'huella_decision_sha256', p_contexto ->> 'huella_decision_sha256',
        'huella_plan_autorizacion_sha256', autoridad.huella_plan_sha256,
        'huella_orden_sha256', autoridad.huella_orden_sha256,
        'huella_capacidad_sha256', autoridad.huella_capacidad_sha256,
        'huella_plan_efecto_sha256', v_huella_plan, 'huella_manifiesto_sha256', v_huella_manifiesto
    ); v_canonico := ''::bytea; FOREACH v_anterior IN ARRAY ARRAY[
        v_tupla ->> 'esquema', v_tupla ->> 'orden_ref', v_tupla ->> 'decision_ref',
        v_tupla ->> 'efecto_ref', v_tupla ->> 'huella_decision_sha256',
        v_tupla ->> 'huella_plan_autorizacion_sha256',
        v_tupla ->> 'huella_orden_sha256', v_tupla ->> 'huella_capacidad_sha256',
        v_tupla ->> 'huella_plan_efecto_sha256', v_tupla ->> 'huella_manifiesto_sha256' ] LOOP
        v_canonico := v_canonico || vec_ejecucion_documental_v4.encuadrar_capacidad(v_anterior);
    END LOOP; v_huella_tupla := encode(sha256(v_canonico), 'hex');
    v_reserva_calculada := 'reserva:efecto-generacion-documental:v1:' || encode(sha256(
        vec_ejecucion_documental_v4.encuadrar_capacidad(
            'vec.documentos.registro-efectos-generacion.reserva.v1') ||
        vec_ejecucion_documental_v4.encuadrar_capacidad(v_huella_tupla) ||
        vec_ejecucion_documental_v4.encuadrar_capacidad(v_huella_manifiesto) ), 'hex'); PERFORM 1
      FROM vec_ejecucion_documental_v4.reserva_efecto_generacion_documental_v1 AS r
     WHERE r.decision_ref = p_contexto ->> 'autorizacion_ref'
        OR r.efecto_ref = p_contexto ->> 'efecto_ref' ORDER BY r.reserva_ref FOR UPDATE;
    SELECT count(*), min(r.reserva_ref) INTO v_cantidad, v_reserva_ref
      FROM vec_ejecucion_documental_v4.reserva_efecto_generacion_documental_v1 AS r
     WHERE r.decision_ref = p_contexto ->> 'autorizacion_ref'
        OR r.efecto_ref = p_contexto ->> 'efecto_ref'; IF v_cantidad > 1 THEN
        RAISE EXCEPTION USING ERRCODE = '23514', MESSAGE = 'conflicto de decision o efecto documental';
    END IF; IF v_cantidad = 0 THEN SELECT version.* INTO STRICT capacidad_actual
          FROM vec_ejecucion_documental_v4.clave_capacidad_actual AS actual
          JOIN vec_ejecucion_documental_v4.clave_capacidad_version AS version
            ON version.clave_id = actual.clave_id AND version.version = actual.version
         WHERE actual.control_id = true FOR UPDATE OF actual, version;
        IF vec_autorizacion.revalidar_decision_ejecucion_documental_v4(
               autoridad.aplicacion_registro, autoridad.decision_canonica ) IS NOT TRUE THEN
            RAISE EXCEPTION USING ERRCODE = '23514', MESSAGE = 'decision documental no vigente';
        END IF; v_ahora := clock_timestamp();
        IF capacidad_actual.clave_id <> autoridad.capacidad_clave_id
           OR capacidad_actual.version <> autoridad.capacidad_version
           OR capacidad_actual.estado <> 'activa' OR capacidad_actual.revocada_en IS NOT NULL
           OR v_ahora < capacidad_actual.valida_desde OR v_ahora >= capacidad_actual.valida_hasta
           OR v_ahora >= autoridad.capacidad_expira_en
           OR vec_ejecucion_documental_v4.bytea_igual_constante( public.hmac(
                      vec_ejecucion_documental_v4.preimagen_capacidad( autoridad.capacidad
                      ), capacidad_actual.secreto_hmac, 'sha256'
                  ), decode(autoridad.capacidad ->> 'mac_sha256', 'hex') ) IS NOT TRUE THEN
            RAISE EXCEPTION USING ERRCODE = '23514', MESSAGE = 'capacidad documental no vigente';
        END IF; INSERT INTO vec_ejecucion_documental_v4.reserva_efecto_generacion_documental_v1 (
            reserva_ref, orden_ref, decision_ref, efecto_ref,
            huella_decision_sha256, huella_plan_autorizacion_sha256,
            huella_orden_sha256, huella_capacidad_sha256,
            capacidad_clave_id, capacidad_version, capacidad_nonce,
            huella_plan_efecto_sha256, huella_manifiesto_sha256,
            huella_tupla_sha256, tupla_canonica, contexto_canonico,
            manifiesto_canonico, correlacion_ref, reservada_en ) VALUES (
            v_reserva_calculada, autoridad.orden_ref, autoridad.decision_ref,
            autoridad.efecto_ref, autoridad.huella_decision_sha256,
            autoridad.huella_plan_sha256, autoridad.huella_orden_sha256,
            autoridad.huella_capacidad_sha256, autoridad.capacidad_clave_id,
            autoridad.capacidad_version, autoridad.capacidad_nonce,
            v_huella_plan, v_huella_manifiesto, v_huella_tupla, v_tupla,
            v_contexto, v_manifiesto, p_contexto ->> 'correlacion_ref', v_ahora ); v_creada := true;
        v_reserva_ref := v_reserva_calculada; v_cantidad := 1; END IF;
    SELECT r.* INTO STRICT existente
      FROM vec_ejecucion_documental_v4.reserva_efecto_generacion_documental_v1 AS r
     WHERE r.reserva_ref = v_reserva_ref; IF existente.tupla_canonica IS DISTINCT FROM v_tupla
       OR existente.contexto_canonico IS DISTINCT FROM v_contexto
       OR existente.manifiesto_canonico IS DISTINCT FROM v_manifiesto THEN
        RAISE EXCEPTION USING ERRCODE = '23514', MESSAGE = 'replay documental incompatible'; END IF;
    IF COALESCE(v_creada, false) THEN FOR paso IN SELECT valor, ordinalidad::integer AS posicion
              FROM jsonb_array_elements(v_pasos) WITH ORDINALITY AS elemento(valor, ordinalidad)
             ORDER BY ordinalidad LOOP
            INSERT INTO vec_ejecucion_documental_v4.paso_efecto_generacion_documental_v1 (
                reserva_ref, posicion, paso_ref, huella_paso_sha256,
                referencia_logica, clave_idempotencia, formato, zona, mime,
                tamano, huella_contenido_sha256, estado, registrada_en ) VALUES (
                v_reserva_ref, paso.posicion, paso.valor ->> 'paso_ref',
                paso.valor ->> 'huella_paso_sha256', paso.valor ->> 'referencia_logica',
                paso.valor ->> 'clave_idempotencia', paso.valor ->> 'formato',
                paso.valor ->> 'zona', paso.valor ->> 'mime', (paso.valor ->> 'tamano')::bigint,
                paso.valor ->> 'huella_sha256', 'reservado', v_ahora ); END LOOP;
        PERFORM vec_ejecucion_documental_v4.registrar_cambio_efecto_generacion_documental_v1(
            v_reserva_ref, NULL, 'reservar_efecto_generacion_documental', 'reservado', v_ahora );
    END IF; resultado := CASE WHEN COALESCE(v_creada, false) THEN 'reservada' ELSE 'repetida' END;
    reserva_ref := v_reserva_ref; efecto_ref := existente.efecto_ref;
    huella_decision_sha256 := existente.huella_decision_sha256;
    huella_plan_efecto_sha256 := existente.huella_plan_efecto_sha256;
    huella_manifiesto_sha256 := existente.huella_manifiesto_sha256;
    repetida := NOT COALESCE(v_creada, false); SELECT jsonb_agg(jsonb_build_object(
        'paso_ref', base.paso_ref, 'huella_paso_sha256', base.huella_paso_sha256,
        'estado', COALESCE(terminal.estado, base.estado),
        'objeto_ref', COALESCE(terminal.objeto_ref, ''),
        'objeto_version', COALESCE(terminal.objeto_version, ''),
        'conector_id', COALESCE(terminal.conector_id, ''),
        'contenido_guardado', COALESCE(terminal.contenido_guardado_canonico, '{}'::jsonb),
        'incidente_ref', COALESCE(terminal.incidente_ref, '') ) ORDER BY base.posicion) INTO pasos
      FROM vec_ejecucion_documental_v4.paso_efecto_generacion_documental_v1 AS base
      LEFT JOIN vec_ejecucion_documental_v4.paso_efecto_generacion_documental_v1 AS terminal
        ON terminal.reserva_ref = base.reserva_ref AND terminal.paso_ref = base.paso_ref
       AND terminal.estado IN ('confirmado', 'indeterminado')
     WHERE base.reserva_ref = v_reserva_ref AND base.estado = 'reservado'; RETURN NEXT;
EXCEPTION WHEN no_data_found OR too_many_rows THEN RAISE EXCEPTION USING ERRCODE = '23514',
        MESSAGE = 'orden, decision o capacidad documental ausente o ambigua'; END $funcion$;
-- Valida el resultado completo del puerto contra el paso y su contexto fijados.
CREATE FUNCTION vec_ejecucion_documental_v4.contenido_documento_guardado_valido_v1(
    p_contenido jsonb, p_contexto jsonb, p_paso jsonb,
    p_reservada_en timestamptz, p_comprobada_en timestamptz )
RETURNS boolean LANGUAGE plpgsql IMMUTABLE SET search_path = pg_catalog, pg_temp AS $funcion$
DECLARE evidencia jsonb := p_contenido -> 'evidencia_operacion';
    objeto jsonb := p_contenido -> 'evidencia_operacion' -> 'objeto'; realizada_en timestamptz;
BEGIN IF p_contenido IS NULL OR jsonb_typeof(p_contenido) <> 'object'
       OR pg_column_size(p_contenido) > 65536
       OR (SELECT count(*) FROM jsonb_object_keys(p_contenido)) <> 9
       OR NOT (p_contenido ?& ARRAY['referencia_logica', 'referencia',
           'version', 'conector_id', 'zona', 'mime', 'huella_sha256',
           'tamano', 'evidencia_operacion']) OR jsonb_typeof(p_contenido -> 'tamano') <> 'number'
       OR jsonb_typeof(evidencia) <> 'object'
       OR EXISTS (SELECT 1 FROM jsonb_object_keys(p_contenido) AS clave
                   WHERE clave NOT IN ('tamano', 'evidencia_operacion')
                     AND jsonb_typeof(p_contenido -> clave) <> 'string')
       OR (SELECT count(*) FROM jsonb_object_keys(evidencia)) <> 25
       OR NOT (evidencia ?& ARRAY['referencia', 'conector_id',
           'esquema_contexto', 'accion_negocio', 'accion', 'efecto_ref',
           'huella_plan_efecto_sha256', 'huella_manifiesto_sha256',
           'huella_paso_sha256', 'paso_ref', 'huella_decision_sha256',
           'objeto', 'operacion_ref', 'correlacion_ref', 'autorizacion_ref',
           'finalidad', 'clasificacion', 'realizada_en', 'carga_ref',
           'sujeto_seudonimo_hmac', 'recurso_ref', 'modulo_id',
           'huella_solicitud_hmac', 'fundamento_ref', 'reintento_idempotente'])
       OR jsonb_typeof(evidencia -> 'objeto') <> 'object'
       OR jsonb_typeof(evidencia -> 'reintento_idempotente') <> 'boolean'
       OR EXISTS (SELECT 1 FROM jsonb_object_keys(evidencia) AS clave
                   WHERE clave NOT IN ('objeto', 'reintento_idempotente')
                     AND jsonb_typeof(evidencia -> clave) <> 'string')
       OR (SELECT count(*) FROM jsonb_object_keys(objeto)) <> 2
       OR NOT (objeto ?& ARRAY['referencia', 'version'])
       OR EXISTS (SELECT 1 FROM jsonb_object_keys(objeto) AS clave
                   WHERE jsonb_typeof(objeto -> clave) <> 'string') THEN RETURN false; END IF;
    realizada_en := (evidencia ->> 'realizada_en')::timestamptz;
    RETURN (p_contenido ->> 'tamano')::numeric = trunc( (p_contenido ->> 'tamano')::numeric)
       AND (p_contenido ->> 'tamano')::numeric BETWEEN 1 AND 9223372036854775807
       AND vec_ejecucion_documental_v4.texto_tecnico_valido( p_contenido ->> 'referencia_logica', 512)
       AND vec_ejecucion_documental_v4.texto_tecnico_valido( p_contenido ->> 'referencia', 512)
       AND vec_ejecucion_documental_v4.texto_tecnico_valido( p_contenido ->> 'version', 256)
       AND vec_ejecucion_documental_v4.texto_tecnico_valido( p_contenido ->> 'conector_id', 128)
       AND p_contenido ->> 'zona' = 'admitida' AND vec_ejecucion_documental_v4.texto_tecnico_valido(
               p_contenido ->> 'mime', 255) AND vec_ejecucion_documental_v4.huella_sha256_valida(
               p_contenido ->> 'huella_sha256')
       AND p_contenido ->> 'referencia_logica' = p_paso ->> 'referencia_logica'
       AND p_contenido ->> 'zona' = p_paso ->> 'zona' AND p_contenido ->> 'mime' = p_paso ->> 'mime'
       AND (p_contenido ->> 'tamano')::bigint = (p_paso ->> 'tamano')::bigint
       AND p_contenido ->> 'huella_sha256' = p_paso ->> 'huella_sha256'
       AND p_contenido ->> 'referencia' = objeto ->> 'referencia'
       AND p_contenido ->> 'version' = objeto ->> 'version'
       AND p_contenido ->> 'conector_id' = evidencia ->> 'conector_id'
       AND vec_ejecucion_documental_v4.texto_tecnico_valido( evidencia ->> 'referencia', 512)
       AND vec_ejecucion_documental_v4.texto_tecnico_valido( evidencia ->> 'conector_id', 128)
       AND evidencia ->> 'esquema_contexto' = 'vec.almacen.contexto-operacion.v1'
       AND vec_ejecucion_documental_v4.texto_tecnico_valido( evidencia ->> 'accion_negocio', 256)
       AND evidencia ->> 'accion' = 'escribir' AND vec_ejecucion_documental_v4.texto_tecnico_valido(
               evidencia ->> 'efecto_ref', 512)
       AND vec_ejecucion_documental_v4.texto_tecnico_valido( evidencia ->> 'paso_ref', 256)
       AND vec_ejecucion_documental_v4.texto_tecnico_valido( objeto ->> 'referencia', 512)
       AND vec_ejecucion_documental_v4.texto_tecnico_valido( objeto ->> 'version', 256)
       AND vec_ejecucion_documental_v4.texto_tecnico_valido( evidencia ->> 'operacion_ref', 512)
       AND vec_ejecucion_documental_v4.texto_tecnico_valido( evidencia ->> 'correlacion_ref', 512)
       AND vec_ejecucion_documental_v4.texto_tecnico_valido( evidencia ->> 'autorizacion_ref', 512)
       AND vec_ejecucion_documental_v4.texto_tecnico_valido( evidencia ->> 'finalidad', 1024)
       AND vec_ejecucion_documental_v4.texto_tecnico_valido( evidencia ->> 'clasificacion', 256)
       AND vec_ejecucion_documental_v4.instante_valido( evidencia ->> 'realizada_en')
       AND vec_ejecucion_documental_v4.texto_tecnico_valido( evidencia ->> 'carga_ref', 512)
       AND (evidencia ->> 'sujeto_seudonimo_hmac') ~ '^hmac-sha256:[^:[:space:]*]{1,64}:[0-9a-f]{64}$'
       AND vec_ejecucion_documental_v4.texto_tecnico_valido( evidencia ->> 'recurso_ref', 512)
       AND vec_ejecucion_documental_v4.texto_tecnico_valido( evidencia ->> 'modulo_id', 128)
       AND (evidencia ->> 'huella_solicitud_hmac') ~ '^hmac-sha256:[^:[:space:]*]{1,64}:[0-9a-f]{64}$'
       AND evidencia ->> 'fundamento_ref' = '' AND NOT (evidencia ->> 'accion_negocio' LIKE '%*%'
                OR evidencia ->> 'accion' LIKE '%*%' OR evidencia ->> 'efecto_ref' LIKE '%*%'
                OR evidencia ->> 'paso_ref' LIKE '%*%') AND NOT EXISTS (SELECT 1 FROM unnest(ARRAY[
               'huella_plan_efecto_sha256', 'huella_manifiesto_sha256',
               'huella_paso_sha256', 'huella_decision_sha256']) AS clave
           WHERE vec_ejecucion_documental_v4.huella_sha256_valida( evidencia ->> clave) IS NOT TRUE)
       AND evidencia ->> 'esquema_contexto' = p_contexto ->> 'esquema'
       AND evidencia ->> 'operacion_ref' = p_contexto ->> 'operacion_ref'
       AND evidencia ->> 'correlacion_ref' = p_contexto ->> 'correlacion_ref'
       AND evidencia ->> 'autorizacion_ref' = p_contexto ->> 'autorizacion_ref'
       AND evidencia ->> 'finalidad' = p_contexto ->> 'finalidad'
       AND evidencia ->> 'clasificacion' = p_contexto ->> 'clasificacion'
       AND evidencia ->> 'accion_negocio' = p_contexto ->> 'accion_negocio'
       AND evidencia ->> 'accion' = p_contexto ->> 'accion_tecnica'
       AND evidencia ->> 'efecto_ref' = p_contexto ->> 'efecto_ref'
       AND evidencia ->> 'huella_plan_efecto_sha256' = p_contexto ->> 'huella_plan_efecto_sha256'
       AND evidencia ->> 'huella_manifiesto_sha256' = p_contexto ->> 'huella_manifiesto_sha256'
       AND evidencia ->> 'huella_paso_sha256' = p_paso ->> 'huella_paso_sha256'
       AND evidencia ->> 'paso_ref' = p_paso ->> 'paso_ref'
       AND evidencia ->> 'huella_decision_sha256' = p_contexto ->> 'huella_decision_sha256'
       AND evidencia ->> 'carga_ref' = p_contexto ->> 'carga_ref'
       AND evidencia ->> 'sujeto_seudonimo_hmac' = p_contexto ->> 'sujeto_seudonimo_hmac'
       AND evidencia ->> 'recurso_ref' = p_contexto ->> 'recurso_ref'
       AND evidencia ->> 'modulo_id' = p_contexto ->> 'modulo_id'
       AND evidencia ->> 'huella_solicitud_hmac' = p_contexto ->> 'huella_solicitud_hmac'
       AND realizada_en >= p_reservada_en AND realizada_en <= p_comprobada_en;
EXCEPTION WHEN OTHERS THEN RETURN false; END $funcion$;
CREATE FUNCTION vec_ejecucion_documental_v4.confirmar_paso_generacion_documental_v1(
    p_confirmacion jsonb)
RETURNS TABLE (resultado text, estado text) LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path = pg_catalog, pg_temp AS $funcion$ DECLARE
    reserva record; base record; terminal record; v_terminal boolean;
    v_ahora timestamptz(6) := clock_timestamp(); v_realizada_en timestamptz(6);
    contenido jsonb; evidencia jsonb; paso_canonico jsonb; BEGIN
    IF p_confirmacion IS NULL OR jsonb_typeof(p_confirmacion) <> 'object'
       OR pg_column_size(p_confirmacion) > 131072
       OR (SELECT count(*) FROM jsonb_object_keys(p_confirmacion)) <> 10
       OR NOT (p_confirmacion ?& ARRAY[ 'esquema', 'reserva_ref', 'decision_ref', 'efecto_ref',
           'huella_decision_sha256', 'huella_plan_efecto_sha256',
           'huella_manifiesto_sha256', 'paso_ref', 'huella_paso_sha256', 'contenido_guardado'
       ]) OR EXISTS ( SELECT 1 FROM jsonb_object_keys(p_confirmacion) AS clave
            WHERE clave <> 'contenido_guardado' AND jsonb_typeof(p_confirmacion -> clave) <> 'string'
       ) OR p_confirmacion ->> 'esquema' <>
          'vec.documentos.registro-efectos-generacion.confirmacion.v1' OR EXISTS (
           SELECT 1 FROM unnest(ARRAY[
               p_confirmacion ->> 'reserva_ref', p_confirmacion ->> 'decision_ref',
               p_confirmacion ->> 'efecto_ref', p_confirmacion ->> 'paso_ref' ]) AS valor WHERE
               vec_ejecucion_documental_v4.texto_tecnico_valido(valor, 512) IS NOT TRUE )
       OR EXISTS ( SELECT 1 FROM unnest(ARRAY[ 'huella_decision_sha256', 'huella_plan_efecto_sha256',
               'huella_manifiesto_sha256', 'huella_paso_sha256'
           ]) AS clave WHERE vec_ejecucion_documental_v4.huella_sha256_valida(
               p_confirmacion ->> clave) IS NOT TRUE
       ) OR jsonb_typeof(p_confirmacion -> 'contenido_guardado') <> 'object' THEN
        RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'confirmacion documental invalida';
    END IF; contenido := p_confirmacion -> 'contenido_guardado';
    evidencia := contenido -> 'evidencia_operacion'; SELECT r.* INTO STRICT reserva
      FROM vec_ejecucion_documental_v4.reserva_efecto_generacion_documental_v1 AS r
     WHERE r.reserva_ref = p_confirmacion ->> 'reserva_ref' FOR UPDATE;
    IF reserva.decision_ref <> p_confirmacion ->> 'decision_ref'
       OR reserva.efecto_ref <> p_confirmacion ->> 'efecto_ref'
       OR reserva.huella_decision_sha256 <> p_confirmacion ->> 'huella_decision_sha256'
       OR reserva.huella_plan_efecto_sha256 <> p_confirmacion ->> 'huella_plan_efecto_sha256'
       OR reserva.huella_manifiesto_sha256 <> p_confirmacion ->> 'huella_manifiesto_sha256' THEN
        RAISE EXCEPTION USING ERRCODE = '23514', MESSAGE = 'confirmacion documental no coincide con la reserva';
    END IF; SELECT p.* INTO STRICT base
      FROM vec_ejecucion_documental_v4.paso_efecto_generacion_documental_v1 AS p
     WHERE p.reserva_ref = reserva.reserva_ref AND p.paso_ref = p_confirmacion ->> 'paso_ref'
       AND p.estado = 'reservado' FOR UPDATE;
    IF base.huella_paso_sha256 <> p_confirmacion ->> 'huella_paso_sha256' THEN
        RAISE EXCEPTION USING ERRCODE = '23514', MESSAGE = 'confirmacion documental no coincide con el paso';
    END IF; paso_canonico := jsonb_build_object(
        'paso_ref', base.paso_ref, 'huella_paso_sha256', base.huella_paso_sha256,
        'referencia_logica', base.referencia_logica, 'zona', base.zona,
        'mime', base.mime, 'tamano', base.tamano, 'huella_sha256', base.huella_contenido_sha256 );
    IF vec_ejecucion_documental_v4.contenido_documento_guardado_valido_v1(
           contenido, reserva.contexto_canonico, paso_canonico,
           reserva.reservada_en, v_ahora) IS NOT TRUE THEN RAISE EXCEPTION USING ERRCODE = '23514',
            MESSAGE = 'contenido o evidencia documental no coincide con el paso'; END IF;
    v_realizada_en := (evidencia ->> 'realizada_en')::timestamptz; SELECT p.* INTO terminal
      FROM vec_ejecucion_documental_v4.paso_efecto_generacion_documental_v1 AS p
     WHERE p.reserva_ref = reserva.reserva_ref AND p.paso_ref = base.paso_ref
       AND p.estado IN ('confirmado', 'indeterminado') FOR UPDATE; v_terminal := FOUND;
    IF v_terminal THEN IF terminal.estado = 'confirmado'
           AND terminal.contenido_guardado_canonico = contenido
           AND terminal.evidencia_operacion_canonica = evidencia
           AND terminal.resultado_en = v_realizada_en THEN
            resultado := 'repetida'; estado := 'confirmado'; RETURN NEXT; RETURN; END IF;
        RAISE EXCEPTION USING ERRCODE = '23514', MESSAGE = 'paso documental terminal incompatible';
    END IF; INSERT INTO vec_ejecucion_documental_v4.paso_efecto_generacion_documental_v1 (
        reserva_ref, posicion, paso_ref, huella_paso_sha256,
        referencia_logica, clave_idempotencia, formato, zona, mime, tamano,
        huella_contenido_sha256, estado, objeto_ref, objeto_version,
        conector_id, evidencia_operacion_ref, contenido_guardado_canonico,
        evidencia_operacion_canonica, resultado_en, registrada_en ) VALUES (
        base.reserva_ref, base.posicion, base.paso_ref, base.huella_paso_sha256,
        base.referencia_logica, base.clave_idempotencia, base.formato,
        base.zona, base.mime, base.tamano, base.huella_contenido_sha256,
        'confirmado', contenido ->> 'referencia', contenido ->> 'version',
        contenido ->> 'conector_id', evidencia ->> 'referencia',
        contenido, evidencia, v_realizada_en, v_ahora );
    PERFORM vec_ejecucion_documental_v4.registrar_cambio_efecto_generacion_documental_v1(
        reserva.reserva_ref, base.paso_ref, 'confirmar_paso_generacion_documental',
        'confirmado', v_ahora ); resultado := 'confirmada'; estado := 'confirmado'; RETURN NEXT;
EXCEPTION WHEN no_data_found OR too_many_rows THEN
    RAISE EXCEPTION USING ERRCODE = '23514', MESSAGE = 'reserva o paso documental ausente o ambiguo';
END $funcion$;
CREATE FUNCTION vec_ejecucion_documental_v4.marcar_paso_generacion_documental_indeterminado_v1(
    p_marca jsonb)
RETURNS TABLE (resultado text, estado text) LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path = pg_catalog, pg_temp AS $funcion$ DECLARE
    reserva record; base record; terminal record; v_terminal boolean;
    v_ahora timestamptz(6) := clock_timestamp(); BEGIN
    IF p_marca IS NULL OR jsonb_typeof(p_marca) <> 'object' OR pg_column_size(p_marca) > 32768
       OR (SELECT count(*) FROM jsonb_object_keys(p_marca)) <> 10 OR NOT (p_marca ?& ARRAY[
           'esquema', 'reserva_ref', 'decision_ref', 'efecto_ref',
           'huella_decision_sha256', 'huella_plan_efecto_sha256',
           'huella_manifiesto_sha256', 'paso_ref', 'huella_paso_sha256', 'incidente_ref'
       ]) OR EXISTS ( SELECT 1 FROM jsonb_object_keys(p_marca) AS clave
            WHERE jsonb_typeof(p_marca -> clave) <> 'string' ) OR p_marca ->> 'esquema' <>
          'vec.documentos.registro-efectos-generacion.indeterminado.v1' OR EXISTS (
           SELECT 1 FROM unnest(ARRAY[ p_marca ->> 'reserva_ref', p_marca ->> 'decision_ref',
               p_marca ->> 'efecto_ref', p_marca ->> 'paso_ref', p_marca ->> 'incidente_ref'
           ]) AS valor WHERE vec_ejecucion_documental_v4.texto_tecnico_valido(valor, 512) IS NOT TRUE
       ) OR EXISTS ( SELECT 1 FROM unnest(ARRAY[
               'huella_decision_sha256', 'huella_plan_efecto_sha256',
               'huella_manifiesto_sha256', 'huella_paso_sha256'
           ]) AS clave WHERE vec_ejecucion_documental_v4.huella_sha256_valida(
               p_marca ->> clave) IS NOT TRUE ) THEN
        RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'marca documental indeterminada invalida';
    END IF; SELECT r.* INTO STRICT reserva
      FROM vec_ejecucion_documental_v4.reserva_efecto_generacion_documental_v1 AS r
     WHERE r.reserva_ref = p_marca ->> 'reserva_ref' FOR UPDATE;
    IF reserva.decision_ref <> p_marca ->> 'decision_ref'
       OR reserva.efecto_ref <> p_marca ->> 'efecto_ref'
       OR reserva.huella_decision_sha256 <> p_marca ->> 'huella_decision_sha256'
       OR reserva.huella_plan_efecto_sha256 <> p_marca ->> 'huella_plan_efecto_sha256'
       OR reserva.huella_manifiesto_sha256 <> p_marca ->> 'huella_manifiesto_sha256' THEN
        RAISE EXCEPTION USING ERRCODE = '23514', MESSAGE = 'marca indeterminada no coincide con la reserva';
    END IF; SELECT p.* INTO STRICT base
      FROM vec_ejecucion_documental_v4.paso_efecto_generacion_documental_v1 AS p
     WHERE p.reserva_ref = reserva.reserva_ref AND p.paso_ref = p_marca ->> 'paso_ref'
       AND p.estado = 'reservado' FOR UPDATE;
    IF base.huella_paso_sha256 <> p_marca ->> 'huella_paso_sha256' THEN
        RAISE EXCEPTION USING ERRCODE = '23514', MESSAGE = 'marca indeterminada no coincide con el paso';
    END IF; SELECT p.* INTO terminal
      FROM vec_ejecucion_documental_v4.paso_efecto_generacion_documental_v1 AS p
     WHERE p.reserva_ref = reserva.reserva_ref AND p.paso_ref = base.paso_ref
       AND p.estado IN ('confirmado', 'indeterminado') FOR UPDATE; v_terminal := FOUND;
    IF v_terminal THEN IF terminal.estado = 'indeterminado'
           AND terminal.incidente_ref = p_marca ->> 'incidente_ref' THEN
            resultado := 'repetida'; estado := 'indeterminado'; RETURN NEXT; RETURN; END IF;
        RAISE EXCEPTION USING ERRCODE = '23514', MESSAGE = 'paso documental terminal incompatible';
    END IF; INSERT INTO vec_ejecucion_documental_v4.paso_efecto_generacion_documental_v1 (
        reserva_ref, posicion, paso_ref, huella_paso_sha256,
        referencia_logica, clave_idempotencia, formato, zona, mime, tamano,
        huella_contenido_sha256, estado, incidente_ref, resultado_en, registrada_en ) VALUES (
        base.reserva_ref, base.posicion, base.paso_ref, base.huella_paso_sha256,
        base.referencia_logica, base.clave_idempotencia, base.formato,
        base.zona, base.mime, base.tamano, base.huella_contenido_sha256,
        'indeterminado', p_marca ->> 'incidente_ref', v_ahora, v_ahora );
    PERFORM vec_ejecucion_documental_v4.registrar_cambio_efecto_generacion_documental_v1(
        reserva.reserva_ref, base.paso_ref,
        'marcar_paso_generacion_documental_indeterminado', 'indeterminado', v_ahora );
    resultado := 'marcada'; estado := 'indeterminado'; RETURN NEXT;
EXCEPTION WHEN no_data_found OR too_many_rows THEN
    RAISE EXCEPTION USING ERRCODE = '23514', MESSAGE = 'reserva o paso documental ausente o ambiguo';
END $funcion$;
DO $protecciones$ DECLARE tabla text; BEGIN FOREACH tabla IN ARRAY ARRAY[
        'reserva_efecto_generacion_documental_v1', 'paso_efecto_generacion_documental_v1',
        'auditoria_efecto_generacion_documental_v1', 'evento_outbox_efecto_generacion_documental_v1'
    ] LOOP
        EXECUTE format('CREATE TRIGGER %I_inmutable BEFORE UPDATE OR DELETE ON vec_ejecucion_documental_v4.%I FOR EACH ROW EXECUTE FUNCTION vec_ejecucion_documental_v4.rechazar_mutacion_inmutable()', tabla, tabla);
        EXECUTE format('CREATE TRIGGER %I_no_truncar BEFORE TRUNCATE ON vec_ejecucion_documental_v4.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_ejecucion_documental_v4.rechazar_mutacion_inmutable()', tabla, tabla);
        EXECUTE format('ALTER TABLE vec_ejecucion_documental_v4.%I ENABLE ROW LEVEL SECURITY', tabla);
        EXECUTE format('ALTER TABLE vec_ejecucion_documental_v4.%I FORCE ROW LEVEL SECURITY', tabla);
        EXECUTE format(
            'CREATE POLICY acceso_propietario_exacto ON vec_ejecucion_documental_v4.%I FOR ALL TO vec_ejecucion_documental_v4_propietario USING (current_user = %L) WITH CHECK (current_user = %L)',
            tabla, 'vec_ejecucion_documental_v4_propietario',
            'vec_ejecucion_documental_v4_propietario' ); END LOOP; END $protecciones$;
REVOKE ALL ON ALL TABLES IN SCHEMA vec_ejecucion_documental_v4 FROM PUBLIC;
REVOKE ALL ON ALL FUNCTIONS IN SCHEMA vec_ejecucion_documental_v4 FROM PUBLIC;
DO $cerrar_tipos$
DECLARE tipo record; BEGIN FOR tipo IN SELECT n.nspname, t.typname FROM pg_catalog.pg_type AS t
          JOIN pg_catalog.pg_namespace AS n ON n.oid = t.typnamespace
         WHERE n.nspname = 'vec_ejecucion_documental_v4' AND t.typelem = 0 AND t.typisdefined LOOP
        EXECUTE format('REVOKE ALL PRIVILEGES ON TYPE %I.%I FROM PUBLIC',
            tipo.nspname, tipo.typname); END LOOP; END $cerrar_tipos$;
GRANT EXECUTE ON FUNCTION vec_ejecucion_documental_v4.reservar_efecto_generacion_documental_v1(jsonb, jsonb)
    TO vec_ejecucion_documental_v4_ejecutor_atestado;
GRANT EXECUTE ON FUNCTION vec_ejecucion_documental_v4.confirmar_paso_generacion_documental_v1(jsonb)
    TO vec_ejecucion_documental_v4_ejecutor_atestado;
GRANT EXECUTE ON FUNCTION vec_ejecucion_documental_v4.marcar_paso_generacion_documental_indeterminado_v1(jsonb)
    TO vec_ejecucion_documental_v4_ejecutor_atestado;
COMMIT;
