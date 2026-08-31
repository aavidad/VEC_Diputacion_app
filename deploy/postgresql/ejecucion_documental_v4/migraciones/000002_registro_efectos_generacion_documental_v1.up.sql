BEGIN;
SET LOCAL ROLE vec_ejecucion_documental_v4_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
-- La reserva consume la decision y conserva la tupla reproducible del puerto.
CREATE TABLE vec_ejecucion_documental_v4.reserva_efecto_generacion_documental_v1 (
    reserva_ref text PRIMARY KEY, decision_ref text NOT NULL UNIQUE,
    efecto_ref text NOT NULL UNIQUE, huella_decision_sha256 text NOT NULL,
    huella_plan_efecto_sha256 text NOT NULL, huella_manifiesto_sha256 text NOT NULL,
    huella_tupla_sha256 text NOT NULL UNIQUE, tupla_canonica jsonb NOT NULL,
    contexto_canonico jsonb NOT NULL, manifiesto_canonico jsonb NOT NULL,
    correlacion_ref text NOT NULL,
    reservada_en timestamptz(6) NOT NULL,
    CHECK (vec_ejecucion_documental_v4.texto_tecnico_valido(reserva_ref, 512)
        AND vec_ejecucion_documental_v4.texto_tecnico_valido(decision_ref, 512)
        AND vec_ejecucion_documental_v4.texto_tecnico_valido(efecto_ref, 512)
        AND vec_ejecucion_documental_v4.texto_tecnico_valido(correlacion_ref, 512)),
    CHECK (vec_ejecucion_documental_v4.huella_sha256_valida(huella_decision_sha256)
        AND vec_ejecucion_documental_v4.huella_sha256_valida(huella_plan_efecto_sha256)
        AND vec_ejecucion_documental_v4.huella_sha256_valida(huella_manifiesto_sha256)
        AND vec_ejecucion_documental_v4.huella_sha256_valida(huella_tupla_sha256)),
    CHECK (jsonb_typeof(tupla_canonica) = 'object'),
    CHECK (jsonb_typeof(contexto_canonico) = 'object'),
    CHECK (jsonb_typeof(manifiesto_canonico) = 'object')
);
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
    resultado_en timestamptz(6),
    registrada_en timestamptz(6) NOT NULL,
    PRIMARY KEY (reserva_ref, paso_ref, estado),
    UNIQUE (reserva_ref, posicion, estado),
    UNIQUE (reserva_ref, referencia_logica, estado),
    UNIQUE (reserva_ref, clave_idempotencia, estado),
    CHECK (posicion BETWEEN 1 AND 256),
    CHECK (vec_ejecucion_documental_v4.texto_tecnico_valido(paso_ref, 256)
        AND vec_ejecucion_documental_v4.texto_tecnico_valido(referencia_logica, 512)
        AND vec_ejecucion_documental_v4.texto_tecnico_valido(clave_idempotencia, 512)
        AND vec_ejecucion_documental_v4.texto_tecnico_valido(formato, 128)
        AND zona = 'admitida'
        AND vec_ejecucion_documental_v4.texto_tecnico_valido(mime, 255)),
    CHECK (tamano > 0
        AND vec_ejecucion_documental_v4.huella_sha256_valida(huella_paso_sha256)
        AND vec_ejecucion_documental_v4.huella_sha256_valida(huella_contenido_sha256)),
    CHECK (estado IN ('reservado', 'confirmado', 'indeterminado')),
    CHECK (
        estado = 'reservado'
        AND objeto_ref IS NULL AND objeto_version IS NULL
        AND conector_id IS NULL AND evidencia_operacion_ref IS NULL
        AND incidente_ref IS NULL AND resultado_en IS NULL
        OR estado = 'confirmado'
        AND vec_ejecucion_documental_v4.texto_tecnico_valido(objeto_ref, 512)
        AND vec_ejecucion_documental_v4.texto_tecnico_valido(objeto_version, 512)
        AND vec_ejecucion_documental_v4.texto_tecnico_valido(conector_id, 128)
        AND vec_ejecucion_documental_v4.texto_tecnico_valido(evidencia_operacion_ref, 512)
        AND incidente_ref IS NULL AND resultado_en IS NOT NULL
        OR estado = 'indeterminado'
        AND objeto_ref IS NULL AND objeto_version IS NULL
        AND conector_id IS NULL AND evidencia_operacion_ref IS NULL
        AND vec_ejecucion_documental_v4.texto_tecnico_valido(incidente_ref, 512)
        AND resultado_en IS NOT NULL
    )
);
CREATE UNIQUE INDEX paso_efecto_generacion_documental_v1_terminal_unico ON vec_ejecucion_documental_v4.paso_efecto_generacion_documental_v1(reserva_ref, paso_ref)
    WHERE estado IN ('confirmado', 'indeterminado');
CREATE TABLE vec_ejecucion_documental_v4.auditoria_efecto_generacion_documental_v1 (
    auditoria_ref text PRIMARY KEY, secuencia numeric(20, 0) NOT NULL UNIQUE,
    reserva_ref text NOT NULL REFERENCES
        vec_ejecucion_documental_v4.reserva_efecto_generacion_documental_v1(reserva_ref),
    paso_ref text, accion text NOT NULL, resultado text NOT NULL,
    correlacion_ref text NOT NULL, ocurrida_en timestamptz(6) NOT NULL,
    huella_anterior_sha256 text NOT NULL,
    huella_registro_sha256 text NOT NULL UNIQUE,
    CHECK (vec_ejecucion_documental_v4.texto_tecnico_valido(auditoria_ref, 512)),
    CHECK (paso_ref IS NULL OR
        vec_ejecucion_documental_v4.texto_tecnico_valido(paso_ref, 256)),
    CHECK ((accion = 'reservar_efecto_generacion_documental' AND
            resultado = 'reservado' AND paso_ref IS NULL)
        OR (accion = 'confirmar_paso_generacion_documental' AND
            resultado = 'confirmado' AND paso_ref IS NOT NULL)
        OR (accion = 'marcar_paso_generacion_documental_indeterminado' AND
            resultado = 'indeterminado' AND paso_ref IS NOT NULL)),
    CHECK (vec_ejecucion_documental_v4.huella_sha256_valida(huella_anterior_sha256)),
    CHECK (vec_ejecucion_documental_v4.huella_sha256_valida(huella_registro_sha256))
);
CREATE TABLE vec_ejecucion_documental_v4.evento_outbox_efecto_generacion_documental_v1 (
    evento_ref text PRIMARY KEY, secuencia numeric(20, 0) NOT NULL UNIQUE,
    tipo text NOT NULL, estado text NOT NULL,
    reserva_ref text NOT NULL REFERENCES
        vec_ejecucion_documental_v4.reserva_efecto_generacion_documental_v1(reserva_ref),
    paso_ref text,
    auditoria_ref text NOT NULL UNIQUE REFERENCES
        vec_ejecucion_documental_v4.auditoria_efecto_generacion_documental_v1(auditoria_ref),
    huella_auditoria_sha256 text NOT NULL, correlacion_ref text NOT NULL,
    registrada_en timestamptz(6) NOT NULL,
    huella_registro_sha256 text NOT NULL UNIQUE,
    CHECK (vec_ejecucion_documental_v4.texto_tecnico_valido(evento_ref, 512)),
    CHECK (estado = 'pendiente'),
    CHECK ((tipo = 'efecto_generacion_documental_reservado' AND paso_ref IS NULL)
        OR (tipo IN ('paso_generacion_documental_confirmado',
            'paso_generacion_documental_indeterminado') AND paso_ref IS NOT NULL)),
    CHECK (paso_ref IS NULL OR
        vec_ejecucion_documental_v4.texto_tecnico_valido(paso_ref, 256)),
    CHECK (vec_ejecucion_documental_v4.huella_sha256_valida(huella_auditoria_sha256)),
    CHECK (vec_ejecucion_documental_v4.huella_sha256_valida(huella_registro_sha256))
);
-- Auxiliar privado: enlaza el cambio a 000001; sin cargas libres ni concesion.
CREATE FUNCTION vec_ejecucion_documental_v4.registrar_cambio_efecto_generacion_documental_v1(
    p_reserva_ref text, p_paso_ref text, p_accion text,
    p_resultado text, p_instante timestamptz
)
RETURNS void LANGUAGE plpgsql VOLATILE
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    reserva record;
    v_secuencia numeric(20, 0); v_huella_anterior text;
    v_auditoria_ref text; v_evento_ref text; v_tipo_evento text;
    v_huella_auditoria text; v_huella_evento text;
BEGIN
    IF p_instante IS NULL OR NOT (
        p_accion = 'reservar_efecto_generacion_documental'
        AND p_resultado = 'reservado' AND p_paso_ref IS NULL
        OR p_accion = 'confirmar_paso_generacion_documental'
        AND p_resultado = 'confirmado' AND p_paso_ref IS NOT NULL
        OR p_accion = 'marcar_paso_generacion_documental_indeterminado'
        AND p_resultado = 'indeterminado' AND p_paso_ref IS NOT NULL
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'cambio documental no valido';
    END IF;
    SELECT r.* INTO STRICT reserva
      FROM vec_ejecucion_documental_v4.reserva_efecto_generacion_documental_v1 AS r
     WHERE r.reserva_ref = p_reserva_ref;
    SELECT c.ultima_secuencia + 1, c.ultima_huella_sha256
      INTO v_secuencia, v_huella_anterior
      FROM vec_ejecucion_documental_v4.control_cadena_auditoria AS c
     WHERE c.control_id = true FOR UPDATE;
    IF NOT FOUND OR v_secuencia > 18446744073709551615 THEN
        RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'cadena de auditoria no disponible';
    END IF;
    v_auditoria_ref := 'auditoria:efecto-generacion-documental:v1:' ||
        encode(sha256(convert_to(concat_ws(E'\n', p_reserva_ref,
            COALESCE(p_paso_ref, ''), p_accion, v_secuencia::text), 'UTF8')), 'hex');
    v_evento_ref := 'evento:efecto-generacion-documental:v1:' ||
        encode(sha256(convert_to(concat_ws(E'\n', p_reserva_ref,
            COALESCE(p_paso_ref, ''), p_resultado, v_secuencia::text), 'UTF8')), 'hex');
    v_tipo_evento := CASE p_resultado
        WHEN 'reservado' THEN 'efecto_generacion_documental_reservado'
        WHEN 'confirmado' THEN 'paso_generacion_documental_confirmado'
        ELSE 'paso_generacion_documental_indeterminado'
    END;
    v_huella_auditoria := encode(sha256(convert_to(concat_ws(E'\n',
        v_auditoria_ref, v_secuencia::text, reserva.decision_ref,
        reserva.efecto_ref, reserva.huella_plan_efecto_sha256,
        reserva.huella_manifiesto_sha256, COALESCE(p_paso_ref, ''), p_accion,
        p_resultado, reserva.correlacion_ref,
        to_char(p_instante AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
        v_huella_anterior
    ), 'UTF8')), 'hex');
    v_huella_evento := encode(sha256(convert_to(concat_ws(E'\n',
        v_evento_ref, v_secuencia::text, v_tipo_evento, 'pendiente',
        p_reserva_ref, COALESCE(p_paso_ref, ''), v_auditoria_ref,
        v_huella_auditoria, reserva.correlacion_ref,
        to_char(p_instante AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
    ), 'UTF8')), 'hex');
    INSERT INTO vec_ejecucion_documental_v4.auditoria_efecto_generacion_documental_v1 (
        auditoria_ref, secuencia, reserva_ref, paso_ref, accion, resultado,
        correlacion_ref, ocurrida_en, huella_anterior_sha256,
        huella_registro_sha256
    ) VALUES (
        v_auditoria_ref, v_secuencia, p_reserva_ref, p_paso_ref, p_accion,
        p_resultado, reserva.correlacion_ref, p_instante, v_huella_anterior,
        v_huella_auditoria
    );
    INSERT INTO vec_ejecucion_documental_v4.evento_outbox_efecto_generacion_documental_v1 (
        evento_ref, secuencia, tipo, estado, reserva_ref, paso_ref,
        auditoria_ref, huella_auditoria_sha256, correlacion_ref,
        registrada_en, huella_registro_sha256
    ) VALUES (
        v_evento_ref, v_secuencia, v_tipo_evento, 'pendiente', p_reserva_ref,
        p_paso_ref, v_auditoria_ref, v_huella_auditoria,
        reserva.correlacion_ref, p_instante, v_huella_evento
    );
    UPDATE vec_ejecucion_documental_v4.control_cadena_auditoria
       SET ultima_secuencia = v_secuencia,
           ultima_huella_sha256 = v_huella_auditoria
     WHERE control_id = true AND ultima_secuencia = v_secuencia - 1
       AND ultima_huella_sha256 = v_huella_anterior;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING ERRCODE = '40001', MESSAGE = 'CAS de auditoria perdido';
    END IF;
END
$funcion$;
CREATE FUNCTION vec_ejecucion_documental_v4.reservar_efecto_generacion_documental_v1(
    p_contexto jsonb, p_manifiesto jsonb
)
RETURNS TABLE (
    resultado text, reserva_ref text, efecto_ref text,
    huella_decision_sha256 text, huella_plan_efecto_sha256 text,
    huella_manifiesto_sha256 text, repetida boolean, pasos jsonb
)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    paso record; existente record;
    v_anterior text := '';
    v_canonico bytea := ''::bytea;
    v_pasos jsonb := '[]'::jsonb;
    v_manifiesto jsonb; v_contexto jsonb; v_tupla jsonb;
    v_huella_paso text; v_huella_manifiesto text; v_huella_plan text;
    v_huella_tupla text; v_reserva_ref text;
    v_creada boolean := false;
    v_cantidad integer;
    v_ahora timestamptz(6) := clock_timestamp();
BEGIN
    IF p_contexto IS NULL OR jsonb_typeof(p_contexto) <> 'object'
       OR pg_column_size(p_contexto) > 262144
       OR (SELECT count(*) FROM jsonb_object_keys(p_contexto)) <> 25
       OR NOT (p_contexto ?& ARRAY[
           'esquema', 'operacion_ref', 'correlacion_ref', 'autorizacion_ref',
           'finalidad', 'clasificacion', 'accion_negocio', 'accion_tecnica',
           'carga_ref', 'sujeto_seudonimo_hmac', 'recurso_ref', 'modulo_id',
           'tipo_recurso', 'huella_recurso_sha256', 'huella_solicitud_hmac',
           'efecto_ref', 'huella_plan_efecto_sha256',
           'huella_manifiesto_sha256', 'huella_paso_sha256', 'paso_ref',
           'objeto_vinculado_ref', 'objeto_vinculado_version',
           'huella_decision_sha256', 'verificada_en', 'valida_hasta'
       ]) OR EXISTS (
           SELECT 1 FROM jsonb_object_keys(p_contexto) AS clave
            WHERE jsonb_typeof(p_contexto -> clave) <> 'string'
       ) OR p_manifiesto IS NULL OR jsonb_typeof(p_manifiesto) <> 'object'
       OR pg_column_size(p_manifiesto) > 2097152
       OR (SELECT count(*) FROM jsonb_object_keys(p_manifiesto)) <> 9
       OR NOT (p_manifiesto ?& ARRAY[
           'esquema', 'plantilla_id', 'plantilla_version', 'modulo_id',
           'tipo_documental', 'huella_plantilla_sha256', 'permiso_generar',
           'huella_manifiesto_sha256', 'pasos'
       ]) OR EXISTS (SELECT 1 FROM jsonb_object_keys(p_manifiesto) AS clave WHERE clave NOT IN ('plantilla_version', 'pasos') AND jsonb_typeof(p_manifiesto -> clave) <> 'string') OR jsonb_typeof(p_manifiesto -> 'plantilla_version') <> 'number'
       OR jsonb_typeof(p_manifiesto -> 'pasos') <> 'array'
       OR jsonb_array_length(p_manifiesto -> 'pasos') NOT BETWEEN 1 AND 256 THEN
        RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'reserva documental invalida';
    END IF;
    IF p_contexto ->> 'esquema' <> 'vec.almacen.contexto-operacion.v1'
       OR p_contexto ->> 'accion_tecnica' <> 'escribir'
       OR p_contexto ->> 'objeto_vinculado_ref' <> ''
       OR p_contexto ->> 'objeto_vinculado_version' <> ''
       OR p_manifiesto ->> 'esquema' <> 'vec.documentos.manifiesto-generacion.v1'
       OR p_contexto ->> 'modulo_id' <> p_manifiesto ->> 'modulo_id'
       OR p_contexto ->> 'accion_negocio' <> p_manifiesto ->> 'permiso_generar'
       OR p_contexto ->> 'huella_manifiesto_sha256' <>
          p_manifiesto ->> 'huella_manifiesto_sha256'
       OR (p_manifiesto ->> 'plantilla_version')::numeric <> trunc(
          (p_manifiesto ->> 'plantilla_version')::numeric)
       OR (p_manifiesto ->> 'plantilla_version')::numeric NOT BETWEEN
          1 AND 9223372036854775807
       OR EXISTS (
           SELECT 1 FROM unnest(ARRAY[
               p_contexto ->> 'operacion_ref', p_contexto ->> 'correlacion_ref',
               p_contexto ->> 'autorizacion_ref', p_contexto ->> 'carga_ref',
               p_contexto ->> 'recurso_ref', p_contexto ->> 'efecto_ref'
           ]) AS valor WHERE
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
          '^hmac-sha256:[^:[:space:]*]{1,64}:[0-9a-f]{64}$'
       OR EXISTS (
           SELECT 1 FROM unnest(ARRAY[
               'huella_recurso_sha256', 'huella_plan_efecto_sha256',
               'huella_manifiesto_sha256', 'huella_paso_sha256',
               'huella_decision_sha256'
           ]) AS clave WHERE vec_ejecucion_documental_v4.huella_sha256_valida(
               p_contexto ->> clave
           ) IS NOT TRUE
       ) OR vec_ejecucion_documental_v4.instante_valido(
           p_contexto ->> 'verificada_en') IS NOT TRUE
       OR vec_ejecucion_documental_v4.instante_valido(
           p_contexto ->> 'valida_hasta') IS NOT TRUE
       OR (p_contexto ->> 'verificada_en')::timestamptz > v_ahora
       OR v_ahora >= (p_contexto ->> 'valida_hasta')::timestamptz THEN
        RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'contexto documental no valido';
    END IF;
    IF EXISTS (
        SELECT 1 FROM unnest(ARRAY[
            p_manifiesto ->> 'plantilla_id', p_manifiesto ->> 'modulo_id',
            p_manifiesto ->> 'tipo_documental'
        ]) AS valor WHERE
            vec_ejecucion_documental_v4.texto_tecnico_valido(valor, 128) IS NOT TRUE
    ) OR vec_ejecucion_documental_v4.texto_tecnico_valido(
        p_manifiesto ->> 'permiso_generar', 256) IS NOT TRUE
       OR vec_ejecucion_documental_v4.huella_sha256_valida(
        p_manifiesto ->> 'huella_plantilla_sha256') IS NOT TRUE
       OR vec_ejecucion_documental_v4.huella_sha256_valida(
        p_manifiesto ->> 'huella_manifiesto_sha256') IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'manifiesto documental no valido';
    END IF;
    FOR paso IN
        SELECT valor, ordinalidad::integer AS posicion
          FROM jsonb_array_elements(p_manifiesto -> 'pasos')
               WITH ORDINALITY AS elemento(valor, ordinalidad)
         ORDER BY ordinalidad
    LOOP
        IF jsonb_typeof(paso.valor) <> 'object'
           OR (SELECT count(*) FROM jsonb_object_keys(paso.valor)) <> 9
           OR NOT (paso.valor ?& ARRAY[
               'paso_ref', 'referencia_logica', 'clave_idempotencia',
               'formato', 'zona', 'mime', 'tamano', 'huella_sha256',
               'huella_paso_sha256'
           ]) OR jsonb_typeof(paso.valor -> 'tamano') <> 'number'
           OR EXISTS (
               SELECT 1 FROM jsonb_object_keys(paso.valor) AS clave
                WHERE clave <> 'tamano'
                  AND jsonb_typeof(paso.valor -> clave) <> 'string'
           ) OR (paso.valor ->> 'tamano')::numeric <> trunc(
               (paso.valor ->> 'tamano')::numeric)
           OR (paso.valor ->> 'tamano')::numeric NOT BETWEEN 1 AND 9223372036854775807
           OR paso.valor ->> 'zona' <> 'admitida'
           OR vec_ejecucion_documental_v4.texto_tecnico_valido(
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
               paso.valor ->> 'huella_sha256') IS NOT TRUE
           OR (v_anterior <> '' AND
               paso.valor ->> 'referencia_logica' <= v_anterior) THEN
            RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'paso documental no valido';
        END IF;
        v_canonico := vec_ejecucion_documental_v4.encuadrar_capacidad(
            'vec.documentos.manifiesto-generacion.paso.v1') ||
            vec_ejecucion_documental_v4.encuadrar_capacidad(
                p_manifiesto ->> 'huella_plantilla_sha256') ||
            vec_ejecucion_documental_v4.encuadrar_capacidad(
                p_manifiesto ->> 'permiso_generar') ||
            vec_ejecucion_documental_v4.encuadrar_capacidad(
                paso.valor ->> 'referencia_logica') ||
            vec_ejecucion_documental_v4.encuadrar_capacidad(
                paso.valor ->> 'clave_idempotencia') ||
            vec_ejecucion_documental_v4.encuadrar_capacidad(
                paso.valor ->> 'formato') ||
            vec_ejecucion_documental_v4.encuadrar_capacidad(
                paso.valor ->> 'zona') ||
            vec_ejecucion_documental_v4.encuadrar_capacidad(
                paso.valor ->> 'mime') ||
            vec_ejecucion_documental_v4.encuadrar_capacidad(
                (paso.valor ->> 'tamano')::bigint::text) ||
            vec_ejecucion_documental_v4.encuadrar_capacidad(
                paso.valor ->> 'huella_sha256');
        v_huella_paso := encode(sha256(v_canonico), 'hex');
        IF paso.valor ->> 'huella_paso_sha256' <> v_huella_paso
           OR paso.valor ->> 'paso_ref' <> 'generar_documento_' || v_huella_paso THEN
            RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'huella de paso documental no valida';
        END IF;
        v_pasos := v_pasos || jsonb_build_array(jsonb_build_object(
            'paso_ref', paso.valor ->> 'paso_ref',
            'referencia_logica', paso.valor ->> 'referencia_logica',
            'clave_idempotencia', paso.valor ->> 'clave_idempotencia',
            'formato', paso.valor ->> 'formato', 'zona', paso.valor ->> 'zona',
            'mime', paso.valor ->> 'mime',
            'tamano', (paso.valor ->> 'tamano')::bigint,
            'huella_sha256', paso.valor ->> 'huella_sha256',
            'huella_paso_sha256', v_huella_paso
        ));
        v_anterior := paso.valor ->> 'referencia_logica';
    END LOOP;
    IF (SELECT count(DISTINCT valor ->> 'clave_idempotencia')
          FROM jsonb_array_elements(v_pasos) AS valor) <> jsonb_array_length(v_pasos)
       OR (SELECT count(DISTINCT valor ->> 'paso_ref')
          FROM jsonb_array_elements(v_pasos) AS valor) <> jsonb_array_length(v_pasos) THEN
        RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'manifiesto documental duplicado';
    END IF;
    v_canonico := vec_ejecucion_documental_v4.encuadrar_capacidad(
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
    END LOOP;
    v_huella_manifiesto := encode(sha256(v_canonico), 'hex');
    IF v_huella_manifiesto <> p_manifiesto ->> 'huella_manifiesto_sha256'
       OR p_contexto ->> 'paso_ref' <> v_pasos -> 0 ->> 'paso_ref'
       OR p_contexto ->> 'huella_paso_sha256' <>
          v_pasos -> 0 ->> 'huella_paso_sha256' THEN
        RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'manifiesto documental no coincide con el contexto';
    END IF;
    v_manifiesto := jsonb_build_object(
        'esquema', p_manifiesto ->> 'esquema',
        'plantilla_id', p_manifiesto ->> 'plantilla_id',
        'plantilla_version', (p_manifiesto ->> 'plantilla_version')::bigint,
        'modulo_id', p_manifiesto ->> 'modulo_id',
        'tipo_documental', p_manifiesto ->> 'tipo_documental',
        'huella_plantilla_sha256', p_manifiesto ->> 'huella_plantilla_sha256',
        'permiso_generar', p_manifiesto ->> 'permiso_generar',
        'huella_manifiesto_sha256', v_huella_manifiesto, 'pasos', v_pasos
    );
    v_contexto := p_contexto;
    v_canonico := ''::bytea;
    FOREACH v_anterior IN ARRAY ARRAY[
        p_contexto ->> 'esquema', p_contexto ->> 'autorizacion_ref',
        p_contexto ->> 'huella_decision_sha256', p_contexto ->> 'accion_negocio',
        p_contexto ->> 'recurso_ref', p_contexto ->> 'huella_recurso_sha256',
        p_contexto ->> 'finalidad', p_contexto ->> 'correlacion_ref',
        p_contexto ->> 'operacion_ref', p_contexto ->> 'carga_ref',
        p_contexto ->> 'clasificacion', p_contexto ->> 'sujeto_seudonimo_hmac',
        p_contexto ->> 'huella_solicitud_hmac', p_contexto ->> 'efecto_ref', '', '',
        v_huella_manifiesto
    ] LOOP
        v_canonico := v_canonico ||
            vec_ejecucion_documental_v4.encuadrar_capacidad(v_anterior);
    END LOOP;
    FOR paso IN SELECT e.valor FROM jsonb_array_elements(v_pasos) AS e(valor) LOOP
        v_canonico := v_canonico ||
            vec_ejecucion_documental_v4.encuadrar_capacidad(paso.valor ->> 'paso_ref') ||
            vec_ejecucion_documental_v4.encuadrar_capacidad('escribir') ||
            vec_ejecucion_documental_v4.encuadrar_capacidad(paso.valor ->> 'huella_paso_sha256');
    END LOOP;
    v_huella_plan := encode(sha256(v_canonico), 'hex');
    IF v_huella_plan <> p_contexto ->> 'huella_plan_efecto_sha256' THEN
        RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'plan documental no canonico';
    END IF;
    v_tupla := jsonb_build_object(
        'esquema', 'vec.documentos.registro-efectos-generacion.tupla.v1',
        'decision_ref', p_contexto ->> 'autorizacion_ref',
        'efecto_ref', p_contexto ->> 'efecto_ref',
        'huella_decision_sha256', p_contexto ->> 'huella_decision_sha256',
        'huella_plan_efecto_sha256', v_huella_plan,
        'huella_manifiesto_sha256', v_huella_manifiesto
    );
    v_canonico := ''::bytea;
    FOREACH v_anterior IN ARRAY ARRAY[
        v_tupla ->> 'esquema', v_tupla ->> 'decision_ref',
        v_tupla ->> 'efecto_ref', v_tupla ->> 'huella_decision_sha256',
        v_tupla ->> 'huella_plan_efecto_sha256',
        v_tupla ->> 'huella_manifiesto_sha256'
    ] LOOP
        v_canonico := v_canonico ||
            vec_ejecucion_documental_v4.encuadrar_capacidad(v_anterior);
    END LOOP;
    v_huella_tupla := encode(sha256(v_canonico), 'hex');
    v_reserva_ref := 'reserva:efecto-generacion-documental:v1:' || encode(sha256(
        vec_ejecucion_documental_v4.encuadrar_capacidad(
            'vec.documentos.registro-efectos-generacion.reserva.v1') ||
        vec_ejecucion_documental_v4.encuadrar_capacidad(v_huella_tupla) ||
        vec_ejecucion_documental_v4.encuadrar_capacidad(v_huella_manifiesto)
    ), 'hex');
    INSERT INTO vec_ejecucion_documental_v4.reserva_efecto_generacion_documental_v1 (
        reserva_ref, decision_ref, efecto_ref, huella_decision_sha256,
        huella_plan_efecto_sha256, huella_manifiesto_sha256,
        huella_tupla_sha256, tupla_canonica, contexto_canonico,
        manifiesto_canonico, correlacion_ref, reservada_en
    ) VALUES (
        v_reserva_ref, p_contexto ->> 'autorizacion_ref',
        p_contexto ->> 'efecto_ref', p_contexto ->> 'huella_decision_sha256',
        v_huella_plan, v_huella_manifiesto, v_huella_tupla, v_tupla,
        v_contexto, v_manifiesto, p_contexto ->> 'correlacion_ref', v_ahora
    ) ON CONFLICT DO NOTHING RETURNING true INTO v_creada;
    PERFORM 1
      FROM vec_ejecucion_documental_v4.reserva_efecto_generacion_documental_v1 AS r
     WHERE r.decision_ref = p_contexto ->> 'autorizacion_ref'
        OR r.efecto_ref = p_contexto ->> 'efecto_ref'
     ORDER BY r.reserva_ref FOR UPDATE;
    SELECT count(*), min(r.reserva_ref) INTO v_cantidad, v_reserva_ref
      FROM vec_ejecucion_documental_v4.reserva_efecto_generacion_documental_v1 AS r
     WHERE r.decision_ref = p_contexto ->> 'autorizacion_ref'
        OR r.efecto_ref = p_contexto ->> 'efecto_ref';
    IF v_cantidad <> 1 THEN
        RAISE EXCEPTION USING ERRCODE = '23514', MESSAGE = 'conflicto de decision o efecto documental';
    END IF;
    SELECT r.* INTO STRICT existente
      FROM vec_ejecucion_documental_v4.reserva_efecto_generacion_documental_v1 AS r
     WHERE r.reserva_ref = v_reserva_ref;
    IF existente.tupla_canonica IS DISTINCT FROM v_tupla
       OR existente.contexto_canonico - 'verificada_en' IS DISTINCT FROM
          v_contexto - 'verificada_en'
       OR existente.manifiesto_canonico IS DISTINCT FROM v_manifiesto THEN
        RAISE EXCEPTION USING ERRCODE = '23514', MESSAGE = 'replay documental incompatible';
    END IF;
    IF COALESCE(v_creada, false) THEN
        FOR paso IN
            SELECT valor, ordinalidad::integer AS posicion
              FROM jsonb_array_elements(v_pasos)
                   WITH ORDINALITY AS elemento(valor, ordinalidad)
             ORDER BY ordinalidad
        LOOP
            INSERT INTO vec_ejecucion_documental_v4.paso_efecto_generacion_documental_v1 (
                reserva_ref, posicion, paso_ref, huella_paso_sha256,
                referencia_logica, clave_idempotencia, formato, zona, mime,
                tamano, huella_contenido_sha256, estado, registrada_en
            ) VALUES (
                v_reserva_ref, paso.posicion, paso.valor ->> 'paso_ref',
                paso.valor ->> 'huella_paso_sha256',
                paso.valor ->> 'referencia_logica',
                paso.valor ->> 'clave_idempotencia', paso.valor ->> 'formato',
                paso.valor ->> 'zona', paso.valor ->> 'mime',
                (paso.valor ->> 'tamano')::bigint,
                paso.valor ->> 'huella_sha256', 'reservado', v_ahora
            );
        END LOOP;
        PERFORM vec_ejecucion_documental_v4.registrar_cambio_efecto_generacion_documental_v1(
            v_reserva_ref, NULL, 'reservar_efecto_generacion_documental',
            'reservado', v_ahora
        );
    END IF;
    resultado := CASE WHEN COALESCE(v_creada, false) THEN 'reservada' ELSE 'repetida' END;
    reserva_ref := v_reserva_ref;
    efecto_ref := existente.efecto_ref;
    huella_decision_sha256 := existente.huella_decision_sha256;
    huella_plan_efecto_sha256 := existente.huella_plan_efecto_sha256;
    huella_manifiesto_sha256 := existente.huella_manifiesto_sha256;
    repetida := NOT COALESCE(v_creada, false);
    SELECT jsonb_agg(jsonb_build_object(
        'paso_ref', base.paso_ref, 'huella_paso_sha256', base.huella_paso_sha256,
        'estado', COALESCE(terminal.estado, base.estado),
        'objeto_ref', COALESCE(terminal.objeto_ref, ''),
        'objeto_version', COALESCE(terminal.objeto_version, ''),
        'conector_id', COALESCE(terminal.conector_id, ''),
        'evidencia_operacion_ref', COALESCE(terminal.evidencia_operacion_ref, ''),
        'incidente_ref', COALESCE(terminal.incidente_ref, '')
    ) ORDER BY base.posicion) INTO pasos
      FROM vec_ejecucion_documental_v4.paso_efecto_generacion_documental_v1 AS base
      LEFT JOIN vec_ejecucion_documental_v4.paso_efecto_generacion_documental_v1 AS terminal
        ON terminal.reserva_ref = base.reserva_ref
       AND terminal.paso_ref = base.paso_ref
       AND terminal.estado IN ('confirmado', 'indeterminado')
     WHERE base.reserva_ref = v_reserva_ref AND base.estado = 'reservado';
    RETURN NEXT;
END
$funcion$;
CREATE FUNCTION vec_ejecucion_documental_v4.confirmar_paso_generacion_documental_v1(
    p_confirmacion jsonb)
RETURNS TABLE (resultado text, estado text) LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    reserva record; base record; terminal record; v_terminal boolean;
    v_ahora timestamptz(6) := clock_timestamp();
    v_realizada_en timestamptz(6);
BEGIN
    IF p_confirmacion IS NULL OR jsonb_typeof(p_confirmacion) <> 'object'
       OR pg_column_size(p_confirmacion) > 65536
       OR (SELECT count(*) FROM jsonb_object_keys(p_confirmacion)) <> 14
       OR NOT (p_confirmacion ?& ARRAY[
           'esquema', 'reserva_ref', 'decision_ref', 'efecto_ref',
           'huella_decision_sha256', 'huella_plan_efecto_sha256',
           'huella_manifiesto_sha256', 'paso_ref', 'huella_paso_sha256',
           'objeto_ref', 'objeto_version', 'conector_id',
           'evidencia_operacion_ref', 'realizada_en'
       ]) OR EXISTS (
           SELECT 1 FROM jsonb_object_keys(p_confirmacion) AS clave
            WHERE jsonb_typeof(p_confirmacion -> clave) <> 'string'
       ) OR p_confirmacion ->> 'esquema' <>
          'vec.documentos.registro-efectos-generacion.confirmacion.v1'
       OR EXISTS (
           SELECT 1 FROM unnest(ARRAY[
               p_confirmacion ->> 'reserva_ref', p_confirmacion ->> 'decision_ref',
               p_confirmacion ->> 'efecto_ref', p_confirmacion ->> 'paso_ref',
               p_confirmacion ->> 'objeto_ref', p_confirmacion ->> 'objeto_version',
               p_confirmacion ->> 'evidencia_operacion_ref'
           ]) AS valor WHERE
               vec_ejecucion_documental_v4.texto_tecnico_valido(valor, 512) IS NOT TRUE
       ) OR vec_ejecucion_documental_v4.texto_tecnico_valido(
           p_confirmacion ->> 'conector_id', 128) IS NOT TRUE
       OR EXISTS (
           SELECT 1 FROM unnest(ARRAY[
               'huella_decision_sha256', 'huella_plan_efecto_sha256',
               'huella_manifiesto_sha256', 'huella_paso_sha256'
           ]) AS clave WHERE vec_ejecucion_documental_v4.huella_sha256_valida(
               p_confirmacion ->> clave) IS NOT TRUE
       ) OR vec_ejecucion_documental_v4.instante_valido(
           p_confirmacion ->> 'realizada_en') IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'confirmacion documental invalida';
    END IF;
    v_realizada_en := (p_confirmacion ->> 'realizada_en')::timestamptz;
    SELECT r.* INTO STRICT reserva
      FROM vec_ejecucion_documental_v4.reserva_efecto_generacion_documental_v1 AS r
     WHERE r.reserva_ref = p_confirmacion ->> 'reserva_ref' FOR UPDATE;
    IF reserva.decision_ref <> p_confirmacion ->> 'decision_ref'
       OR reserva.efecto_ref <> p_confirmacion ->> 'efecto_ref'
       OR reserva.huella_decision_sha256 <> p_confirmacion ->> 'huella_decision_sha256'
       OR reserva.huella_plan_efecto_sha256 <> p_confirmacion ->> 'huella_plan_efecto_sha256'
       OR reserva.huella_manifiesto_sha256 <> p_confirmacion ->> 'huella_manifiesto_sha256'
       OR v_realizada_en < reserva.reservada_en OR v_realizada_en > v_ahora THEN
        RAISE EXCEPTION USING ERRCODE = '23514', MESSAGE = 'confirmacion documental no coincide con la reserva';
    END IF;
    SELECT p.* INTO STRICT base
      FROM vec_ejecucion_documental_v4.paso_efecto_generacion_documental_v1 AS p
     WHERE p.reserva_ref = reserva.reserva_ref
       AND p.paso_ref = p_confirmacion ->> 'paso_ref'
       AND p.estado = 'reservado' FOR UPDATE;
    IF base.huella_paso_sha256 <> p_confirmacion ->> 'huella_paso_sha256' THEN
        RAISE EXCEPTION USING ERRCODE = '23514', MESSAGE = 'confirmacion documental no coincide con el paso';
    END IF;
    SELECT p.* INTO terminal
      FROM vec_ejecucion_documental_v4.paso_efecto_generacion_documental_v1 AS p
     WHERE p.reserva_ref = reserva.reserva_ref AND p.paso_ref = base.paso_ref
       AND p.estado IN ('confirmado', 'indeterminado') FOR UPDATE;
    v_terminal := FOUND;
    IF v_terminal THEN
        IF terminal.estado = 'confirmado'
           AND terminal.objeto_ref = p_confirmacion ->> 'objeto_ref'
           AND terminal.objeto_version = p_confirmacion ->> 'objeto_version'
           AND terminal.conector_id = p_confirmacion ->> 'conector_id'
           AND terminal.evidencia_operacion_ref =
               p_confirmacion ->> 'evidencia_operacion_ref'
           AND terminal.resultado_en = v_realizada_en THEN
            resultado := 'repetida'; estado := 'confirmado'; RETURN NEXT; RETURN;
        END IF;
        RAISE EXCEPTION USING ERRCODE = '23514', MESSAGE = 'paso documental terminal incompatible';
    END IF;
    INSERT INTO vec_ejecucion_documental_v4.paso_efecto_generacion_documental_v1 (
        reserva_ref, posicion, paso_ref, huella_paso_sha256,
        referencia_logica, clave_idempotencia, formato, zona, mime, tamano,
        huella_contenido_sha256, estado, objeto_ref, objeto_version,
        conector_id, evidencia_operacion_ref, resultado_en, registrada_en
    ) VALUES (
        base.reserva_ref, base.posicion, base.paso_ref, base.huella_paso_sha256,
        base.referencia_logica, base.clave_idempotencia, base.formato,
        base.zona, base.mime, base.tamano, base.huella_contenido_sha256,
        'confirmado', p_confirmacion ->> 'objeto_ref',
        p_confirmacion ->> 'objeto_version', p_confirmacion ->> 'conector_id',
        p_confirmacion ->> 'evidencia_operacion_ref', v_realizada_en, v_ahora
    );
    PERFORM vec_ejecucion_documental_v4.registrar_cambio_efecto_generacion_documental_v1(
        reserva.reserva_ref, base.paso_ref, 'confirmar_paso_generacion_documental',
        'confirmado', v_ahora
    );
    resultado := 'confirmada'; estado := 'confirmado'; RETURN NEXT;
EXCEPTION WHEN no_data_found OR too_many_rows THEN
    RAISE EXCEPTION USING ERRCODE = '23514', MESSAGE = 'reserva o paso documental ausente o ambiguo';
END
$funcion$;
CREATE FUNCTION vec_ejecucion_documental_v4.marcar_paso_generacion_documental_indeterminado_v1(
    p_marca jsonb)
RETURNS TABLE (resultado text, estado text) LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    reserva record; base record; terminal record; v_terminal boolean;
    v_ahora timestamptz(6) := clock_timestamp();
BEGIN
    IF p_marca IS NULL OR jsonb_typeof(p_marca) <> 'object'
       OR pg_column_size(p_marca) > 32768
       OR (SELECT count(*) FROM jsonb_object_keys(p_marca)) <> 10
       OR NOT (p_marca ?& ARRAY[
           'esquema', 'reserva_ref', 'decision_ref', 'efecto_ref',
           'huella_decision_sha256', 'huella_plan_efecto_sha256',
           'huella_manifiesto_sha256', 'paso_ref', 'huella_paso_sha256',
           'incidente_ref'
       ]) OR EXISTS (
           SELECT 1 FROM jsonb_object_keys(p_marca) AS clave
            WHERE jsonb_typeof(p_marca -> clave) <> 'string'
       ) OR p_marca ->> 'esquema' <>
          'vec.documentos.registro-efectos-generacion.indeterminado.v1'
       OR EXISTS (
           SELECT 1 FROM unnest(ARRAY[
               p_marca ->> 'reserva_ref', p_marca ->> 'decision_ref',
               p_marca ->> 'efecto_ref', p_marca ->> 'paso_ref',
               p_marca ->> 'incidente_ref'
           ]) AS valor WHERE
               vec_ejecucion_documental_v4.texto_tecnico_valido(valor, 512) IS NOT TRUE
       ) OR EXISTS (
           SELECT 1 FROM unnest(ARRAY[
               'huella_decision_sha256', 'huella_plan_efecto_sha256',
               'huella_manifiesto_sha256', 'huella_paso_sha256'
           ]) AS clave WHERE vec_ejecucion_documental_v4.huella_sha256_valida(
               p_marca ->> clave) IS NOT TRUE
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'marca documental indeterminada invalida';
    END IF;
    SELECT r.* INTO STRICT reserva
      FROM vec_ejecucion_documental_v4.reserva_efecto_generacion_documental_v1 AS r
     WHERE r.reserva_ref = p_marca ->> 'reserva_ref' FOR UPDATE;
    IF reserva.decision_ref <> p_marca ->> 'decision_ref'
       OR reserva.efecto_ref <> p_marca ->> 'efecto_ref'
       OR reserva.huella_decision_sha256 <> p_marca ->> 'huella_decision_sha256'
       OR reserva.huella_plan_efecto_sha256 <> p_marca ->> 'huella_plan_efecto_sha256'
       OR reserva.huella_manifiesto_sha256 <> p_marca ->> 'huella_manifiesto_sha256' THEN
        RAISE EXCEPTION USING ERRCODE = '23514', MESSAGE = 'marca indeterminada no coincide con la reserva';
    END IF;
    SELECT p.* INTO STRICT base
      FROM vec_ejecucion_documental_v4.paso_efecto_generacion_documental_v1 AS p
     WHERE p.reserva_ref = reserva.reserva_ref
       AND p.paso_ref = p_marca ->> 'paso_ref'
       AND p.estado = 'reservado' FOR UPDATE;
    IF base.huella_paso_sha256 <> p_marca ->> 'huella_paso_sha256' THEN
        RAISE EXCEPTION USING ERRCODE = '23514', MESSAGE = 'marca indeterminada no coincide con el paso';
    END IF;
    SELECT p.* INTO terminal
      FROM vec_ejecucion_documental_v4.paso_efecto_generacion_documental_v1 AS p
     WHERE p.reserva_ref = reserva.reserva_ref AND p.paso_ref = base.paso_ref
       AND p.estado IN ('confirmado', 'indeterminado') FOR UPDATE;
    v_terminal := FOUND;
    IF v_terminal THEN
        IF terminal.estado = 'indeterminado'
           AND terminal.incidente_ref = p_marca ->> 'incidente_ref' THEN
            resultado := 'repetida'; estado := 'indeterminado'; RETURN NEXT; RETURN;
        END IF;
        RAISE EXCEPTION USING ERRCODE = '23514', MESSAGE = 'paso documental terminal incompatible';
    END IF;
    INSERT INTO vec_ejecucion_documental_v4.paso_efecto_generacion_documental_v1 (
        reserva_ref, posicion, paso_ref, huella_paso_sha256,
        referencia_logica, clave_idempotencia, formato, zona, mime, tamano,
        huella_contenido_sha256, estado, incidente_ref, resultado_en, registrada_en
    ) VALUES (
        base.reserva_ref, base.posicion, base.paso_ref, base.huella_paso_sha256,
        base.referencia_logica, base.clave_idempotencia, base.formato,
        base.zona, base.mime, base.tamano, base.huella_contenido_sha256,
        'indeterminado', p_marca ->> 'incidente_ref', v_ahora, v_ahora
    );
    PERFORM vec_ejecucion_documental_v4.registrar_cambio_efecto_generacion_documental_v1(
        reserva.reserva_ref, base.paso_ref,
        'marcar_paso_generacion_documental_indeterminado', 'indeterminado', v_ahora
    );
    resultado := 'marcada'; estado := 'indeterminado'; RETURN NEXT;
EXCEPTION WHEN no_data_found OR too_many_rows THEN
    RAISE EXCEPTION USING ERRCODE = '23514', MESSAGE = 'reserva o paso documental ausente o ambiguo';
END
$funcion$;
DO $protecciones$
DECLARE tabla text;
BEGIN
    FOREACH tabla IN ARRAY ARRAY[
        'reserva_efecto_generacion_documental_v1',
        'paso_efecto_generacion_documental_v1',
        'auditoria_efecto_generacion_documental_v1',
        'evento_outbox_efecto_generacion_documental_v1'
    ] LOOP
        EXECUTE format('CREATE TRIGGER %I_inmutable BEFORE UPDATE OR DELETE ON vec_ejecucion_documental_v4.%I FOR EACH ROW EXECUTE FUNCTION vec_ejecucion_documental_v4.rechazar_mutacion_inmutable()', tabla, tabla);
        EXECUTE format('CREATE TRIGGER %I_no_truncar BEFORE TRUNCATE ON vec_ejecucion_documental_v4.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_ejecucion_documental_v4.rechazar_mutacion_inmutable()', tabla, tabla);
        EXECUTE format('ALTER TABLE vec_ejecucion_documental_v4.%I ENABLE ROW LEVEL SECURITY', tabla);
        EXECUTE format('ALTER TABLE vec_ejecucion_documental_v4.%I FORCE ROW LEVEL SECURITY', tabla);
        EXECUTE format(
            'CREATE POLICY acceso_propietario_exacto ON vec_ejecucion_documental_v4.%I FOR ALL TO vec_ejecucion_documental_v4_propietario USING (current_user = %L) WITH CHECK (current_user = %L)',
            tabla, 'vec_ejecucion_documental_v4_propietario',
            'vec_ejecucion_documental_v4_propietario'
        );
    END LOOP;
END
$protecciones$;
REVOKE ALL ON ALL TABLES IN SCHEMA vec_ejecucion_documental_v4 FROM PUBLIC;
REVOKE ALL ON ALL FUNCTIONS IN SCHEMA vec_ejecucion_documental_v4 FROM PUBLIC;
DO $cerrar_tipos$
DECLARE tipo record;
BEGIN
    FOR tipo IN
        SELECT n.nspname, t.typname
          FROM pg_catalog.pg_type AS t
          JOIN pg_catalog.pg_namespace AS n ON n.oid = t.typnamespace
         WHERE n.nspname = 'vec_ejecucion_documental_v4'
           AND t.typelem = 0 AND t.typisdefined
    LOOP
        EXECUTE format('REVOKE ALL PRIVILEGES ON TYPE %I.%I FROM PUBLIC',
            tipo.nspname, tipo.typname);
    END LOOP;
END
$cerrar_tipos$;
GRANT EXECUTE ON FUNCTION vec_ejecucion_documental_v4.reservar_efecto_generacion_documental_v1(jsonb, jsonb)
    TO vec_ejecucion_documental_v4_ejecutor_atestado;
GRANT EXECUTE ON FUNCTION vec_ejecucion_documental_v4.confirmar_paso_generacion_documental_v1(jsonb)
    TO vec_ejecucion_documental_v4_ejecutor_atestado;
GRANT EXECUTE ON FUNCTION vec_ejecucion_documental_v4.marcar_paso_generacion_documental_indeterminado_v1(jsonb)
    TO vec_ejecucion_documental_v4_ejecutor_atestado;
COMMIT;
