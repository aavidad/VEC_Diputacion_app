-- Prueba integral y adversaria. Por defecto revierte todo; el arnes puede
-- confirmar el fixture para probar reinicio y recuperacion de respuesta.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

CREATE SCHEMA vec_prueba_bolsa_baremacion_v3 AUTHORIZATION postgres;
REVOKE ALL ON SCHEMA vec_prueba_bolsa_baremacion_v3 FROM PUBLIC;
CREATE TABLE vec_prueba_bolsa_baremacion_v3.recuperacion (
    id boolean PRIMARY KEY DEFAULT true CHECK (id),
    operacion_prevalidacion jsonb NOT NULL,
    prueba_prevalidacion jsonb NOT NULL,
    canonica_prevalidacion bytea NOT NULL,
    recurso_canonico bytea NOT NULL,
    operacion_confirmacion jsonb NOT NULL,
    prueba_confirmacion jsonb NOT NULL,
    canonica_confirmacion bytea NOT NULL,
    agregado_canonico bytea NOT NULL,
    manifiesto jsonb NOT NULL,
    contenido_manifiesto bytea NOT NULL,
    representacion_manifiesto bytea NOT NULL,
    preimagen_manifiesto bytea NOT NULL,
    huella_prevalidacion_entrada text NOT NULL,
    archivo_esperado jsonb NOT NULL,
    huella_prevalidacion_final text NOT NULL
);
REVOKE ALL ON vec_prueba_bolsa_baremacion_v3.recuperacion FROM PUBLIC;

\ir fixture_decision_v3.sql

\ir fixture_rbac_v3.sql

\if :{?RUTA_MANIFIESTO_DORADO_V3}
\else
    \set RUTA_MANIFIESTO_DORADO_V3 '/tmp/pruebas_sql_v3/manifiesto_probatorio_v3_dorado.json'
\endif
CREATE TEMP TABLE pg_temp.fixture_manifiesto_dorado_v3 (
    documento jsonb NOT NULL
) ON COMMIT DROP;
INSERT INTO pg_temp.fixture_manifiesto_dorado_v3 (documento)
SELECT pg_catalog.pg_read_file(:'RUTA_MANIFIESTO_DORADO_V3')::jsonb;

DO $vectores_dorados_v3$
DECLARE
    fixture jsonb;
    manifiesto jsonb;
    contenido bytea;
    material bytea;
    representacion bytea;
    preimagen bytea;
BEGIN
    SELECT documento INTO STRICT fixture
      FROM pg_temp.fixture_manifiesto_dorado_v3;
    manifiesto := fixture -> 'manifiesto';
    contenido :=
        vec_bolsa_baremacion.contenido_manifiesto_probatorio_v3(manifiesto);
    material := contenido ||
        vec_bolsa_baremacion.parte_canonica_manifiesto_v3(
            manifiesto ->> 'huella_manifiesto_sha256'
        );
    representacion :=
        vec_bolsa_baremacion.parte_canonica_manifiesto_v3(
            'manifiesto_probatorio_baremacion_v3'
        ) || int8send(octet_length(material)::bigint) || material;
    preimagen := convert_to(
        'manifiesto_probatorio_baremacion_v3', 'UTF8'
    ) || decode('00', 'hex') || representacion;

    IF fixture ->> 'esquema_fixture' <>
          'vec.pruebas.bolsa.manifiesto-probatorio-v3-dorado.v1'
       OR (SELECT count(*) FROM jsonb_object_keys(fixture)) <> 5
       OR jsonb_array_length(manifiesto -> 'autorizaciones') <> 18
       OR jsonb_array_length(manifiesto -> 'evidencias') <> 16
       OR fixture -> 'contenido_canonico' ->> 'tamano' <> '5248'
       OR fixture -> 'contenido_canonico' ->> 'sha256' <>
          '427bb5da122a4185bc572ab144bcf143480be38178b91c630276567c8f2122cd'
       OR fixture -> 'representacion_canonica' ->> 'tamano' <> '5371'
       OR fixture -> 'representacion_canonica' ->> 'sha256' <>
          'f26c109a5e0ca5ff4ba7d42f7594ae6f71566576e9d657d7ddd544876a90ec78'
       OR fixture -> 'preimagen_hmac' ->> 'tamano' <> '5407'
       OR fixture -> 'preimagen_hmac' ->> 'sha256' <>
          '23bc76aee2da8ceb0ec2d19591c39b3ee7ba8a714f72995fd47baa37d2eccbe1'
       OR octet_length(contenido) <> 5248
       OR encode(sha256(contenido), 'hex') <>
          '427bb5da122a4185bc572ab144bcf143480be38178b91c630276567c8f2122cd'
       OR octet_length(representacion) <> 5371
       OR encode(sha256(representacion), 'hex') <>
          'f26c109a5e0ca5ff4ba7d42f7594ae6f71566576e9d657d7ddd544876a90ec78'
       OR octet_length(preimagen) <> 5407
       OR encode(sha256(preimagen), 'hex') <>
          '23bc76aee2da8ceb0ec2d19591c39b3ee7ba8a714f72995fd47baa37d2eccbe1'
       OR vec_bolsa_baremacion.archivo_unitario_manifiesto_v3_valido(
           manifiesto, contenido, representacion, preimagen
       ) IS NOT TRUE
       OR vec_bolsa_baremacion.archivo_unitario_manifiesto_v3_valido(
           manifiesto, contenido || decode('00', 'hex'),
           representacion, preimagen
       ) IS NOT FALSE
       OR vec_bolsa_baremacion.contenido_manifiesto_probatorio_v3(
           manifiesto || '{"extra":true}'::jsonb
       ) IS NOT NULL THEN
        RAISE EXCEPTION 'fixture dorado Go/PostgreSQL V3 divergente';
    END IF;
END
$vectores_dorados_v3$;

DO $flujo_v3$
DECLARE
    ahora timestamptz(6) := clock_timestamp();
    recurso_bytes bytea := convert_to(
        '{"ambitos":{"sujeto_ref":"sujeto:bolsa:001"},"atributos":{}}',
        'UTF8'
    );
    baremacion_ref text := 'baremacion:001';
    reserva_ref text := 'reserva:bolsa:v3:001';
    token_texto text := 'token-reserva-bolsa-v3-prueba-001';
    huella_token text := encode(sha256(convert_to(token_texto, 'UTF8')), 'hex');
    ambito text := repeat('1', 64);
    huella_reserva_hmac text :=
        'hmac-sha256:reserva_v1:' || repeat('2', 64);
    huella_confirmacion_hmac text :=
        'hmac-sha256:confirmacion_v2:' || repeat('3', 64);
    huella_confirmacion text := repeat('4', 64);
    huella_efecto_prevalidacion text;
    decision_reserva_ref text := 'decision:reserva:v3:001';
    decision_prevalidacion_ref text := 'decision:prevalidacion:v3:001';
    decision_prevalidacion_vinculo_ref text :=
        'decision:prevalidacion:vinculo-distinto:v3:001';
    decision_confirmacion_ref text := 'decision:confirmacion:v3:001';
    prueba_reserva jsonb;
    canonica_reserva bytea;
    prueba_prevalidacion jsonb;
    canonica_prevalidacion bytea;
    prueba_prevalidacion_vinculo jsonb;
    canonica_prevalidacion_vinculo bytea;
    prueba_confirmacion jsonb;
    canonica_confirmacion bytea;
    base record;
    operacion jsonb;
    operacion_prevalidacion jsonb;
    resultado text;
    reserva_devuelta text;
    numero text;
    huella text;
    agregado_base jsonb;
    agregado_nuevo jsonb;
    agregado_bytes bytea;
    decision_tecnica jsonb;
    manifiesto jsonb;
    contenido bytea;
    material bytea;
    representacion bytea;
    preimagen bytea;
    archivo jsonb;
    archivo_repetido jsonb;
    huella_prevalidacion text;
    huella_prevalidacion_repetida text;
    huella_prevalidacion_final text;
    resultado_vinculo text;
    creada_texto text := '2026-07-16T10:20:30.123456Z';
    solicitada_texto text;
    expira_texto text;
    confirmada_texto text;
BEGIN
    SELECT version.numero, version.huella_estado_sha256,
           version.agregado, version.agregado_canonico
      INTO base
      FROM vec_bolsa_baremacion.version_baremacion AS version
     WHERE version.baremacion_merito_ref = baremacion_ref
       AND version.numero = 1;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'falta fixture V1 base';
    END IF;
    agregado_base := base.agregado;
    solicitada_texto := to_char(
        ahora - interval '1 second', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
    );
    expira_texto := to_char(
        ahora + interval '5 minutes', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
    );
    -- Equivalente textual de time.RFC3339Nano: elimina exclusivamente los
    -- ceros finales de la fraccion y despues el punto si queda vacio.
    confirmada_texto := rtrim(rtrim(to_char(
        ahora, 'YYYY-MM-DD"T"HH24:MI:SS.US'
    ), '0'), '.') || 'Z';

    PERFORM pg_temp.crear_decision_bolsa_prueba(
        decision_reserva_ref, 'bolsa.baremacion.decision.reservar',
        'baremacion', baremacion_ref, '["reserva.decision"]'::jsonb, ahora
    );
    SELECT prueba, decision_canonica
      INTO prueba_reserva, canonica_reserva
      FROM pg_temp.decision_bolsa_prueba
     WHERE decision_ref = decision_reserva_ref;
    operacion := jsonb_build_object(
        'esquema', 'vec.bolsa.baremacion.reserva-postgresql.v3',
        'reserva_ref', reserva_ref,
        'huella_token_sha256', huella_token,
        'ambito_idempotencia_sha256', ambito,
        'clase', 'incorporar_decision',
        'baremacion_merito_ref', baremacion_ref,
        'version_esperada', '1',
        'huella_version_esperada_sha256', base.huella_estado_sha256,
        'huella_solicitud_hmac', huella_reserva_hmac,
        'huella_efecto_sha256', repeat('5', 64),
        'solicitada_en', solicitada_texto,
        'expira_en', expira_texto
    );
    SELECT r.resultado, r.reserva_ref, r.archivo_probatorio_documento
      INTO resultado, reserva_devuelta, archivo
      FROM vec_bolsa_baremacion.reservar_cambio_con_archivo_probatorio_v3(
          operacion, prueba_reserva, canonica_reserva, recurso_bytes
      ) AS r;
    IF resultado <> 'reservada' OR reserva_devuelta <> reserva_ref
       OR archivo IS NOT NULL THEN
        RAISE EXCEPTION 'reserva V3 inesperada: %, %, %',
            resultado, reserva_devuelta, archivo;
    END IF;

    huella_efecto_prevalidacion :=
        vec_bolsa_baremacion.huella_canonica(ARRAY[
            'efecto-prevalidacion-archivo-probatorio-baremacion-v3',
            huella_confirmacion
        ]);
    operacion_prevalidacion := jsonb_build_object(
        'esquema',
            'vec.bolsa.baremacion.prevalidacion-archivo-postgresql.v3',
        'clase', 'incorporar_decision',
        'baremacion_merito_ref', baremacion_ref,
        'version_esperada', '1',
        'huella_version_esperada_sha256', base.huella_estado_sha256,
        'huella_token_sha256', huella_token,
        'huella_confirmacion_sha256', huella_confirmacion,
        'huella_efecto_prevalidacion_sha256',
            huella_efecto_prevalidacion
    );

    -- Una autorizacion valida del mismo principal, pero emitida tras cambiar
    -- el ContextoActor, no puede consumir la prevalidacion de una reserva
    -- ligada al contexto anterior. La subtransaccion revierte el fixture
    -- adversario para continuar el flujo positivo con el vinculo original.
    BEGIN
        INSERT INTO vec_autorizacion.contexto_actor_v1 (
            contexto_actor_ref, version, cuenta_ref, principal_id,
            perfil_activo_ref, estado, huella_sha256,
            vigente_desde, vigente_hasta
        ) VALUES (
            'vca_bolsa_postgresql_prueba_000001', 2,
            'cta_bolsa_postgresql_prueba_000001',
            'per_bolsa_postgresql_prueba_000001',
            'prf_bolsa_postgresql_prueba_000001', 'activo', repeat('9', 64),
            ahora - interval '1 minute', ahora + interval '1 hour'
        );
        UPDATE vec_autorizacion.contexto_actor_actual_v1
           SET version = 2, actualizada_en = clock_timestamp(),
               acto_ref = 'acto:contexto:bolsa:prueba:v3:v2'
         WHERE cuenta_ref = 'cta_bolsa_postgresql_prueba_000001'
           AND perfil_activo_ref = 'prf_bolsa_postgresql_prueba_000001';
        IF NOT FOUND THEN
            RAISE EXCEPTION 'falta ContextoActor actual para prueba adversaria';
        END IF;
        PERFORM pg_temp.crear_decision_bolsa_prueba(
            decision_prevalidacion_vinculo_ref,
            'bolsa.baremacion.archivo.prevalidar',
            'baremacion', baremacion_ref,
            '["archivo_probatorio"]'::jsonb, ahora
        );
        SELECT prueba, decision_canonica
          INTO prueba_prevalidacion_vinculo,
               canonica_prevalidacion_vinculo
          FROM pg_temp.decision_bolsa_prueba
         WHERE decision_ref = decision_prevalidacion_vinculo_ref;
        SELECT p.resultado INTO resultado_vinculo
          FROM vec_bolsa_baremacion.obtener_archivo_probatorio_previo_cambio_v3(
              operacion_prevalidacion, prueba_prevalidacion_vinculo,
              canonica_prevalidacion_vinculo, recurso_bytes
          ) AS p;
        IF resultado_vinculo <> 'reserva_invalida'
           OR EXISTS (
               SELECT 1 FROM vec_bolsa_baremacion.uso_decision
                WHERE decision_ref = decision_prevalidacion_vinculo_ref
           )
           OR EXISTS (
               SELECT 1
                 FROM vec_bolsa_baremacion.prevalidacion_archivo_probatorio_v3
                WHERE autorizacion_prevalidacion_ref =
                      decision_prevalidacion_vinculo_ref
           ) THEN
            RAISE EXCEPTION
                'prevalidacion consumio autorizacion con vinculo distinto: %',
                resultado_vinculo;
        END IF;
        RAISE EXCEPTION USING ERRCODE = 'ZV001',
            MESSAGE = 'revertir fixture adversario de ContextoActor';
    EXCEPTION
        WHEN SQLSTATE 'ZV001' THEN NULL;
    END;

    PERFORM pg_temp.crear_decision_bolsa_prueba(
        decision_prevalidacion_ref,
        'bolsa.baremacion.archivo.prevalidar',
        'baremacion', baremacion_ref, '["archivo_probatorio"]'::jsonb, ahora
    );
    SELECT prueba, decision_canonica
      INTO prueba_prevalidacion, canonica_prevalidacion
      FROM pg_temp.decision_bolsa_prueba
     WHERE decision_ref = decision_prevalidacion_ref;
    operacion := operacion_prevalidacion;
    SELECT p.resultado, p.numero_version,
           p.archivo_probatorio_documento,
           p.huella_prevalidacion_sha256
      INTO resultado, numero, archivo, huella_prevalidacion
      FROM vec_bolsa_baremacion.obtener_archivo_probatorio_previo_cambio_v3(
          operacion, prueba_prevalidacion, canonica_prevalidacion,
          recurso_bytes
      ) AS p;
    IF resultado <> 'activa' OR numero <> '1'
       OR archivo ->> 'numero_version' <> '1'
       OR jsonb_array_length(archivo -> 'manifiestos') <> 0
       OR vec_bolsa_baremacion.huella_sha256_valida(
           huella_prevalidacion
       ) IS NOT TRUE
       OR (SELECT huella_efecto_sha256
             FROM vec_bolsa_baremacion.uso_decision
            WHERE decision_ref = decision_prevalidacion_ref)
          <> huella_efecto_prevalidacion THEN
        RAISE EXCEPTION 'prevalidacion V3 activa divergente';
    END IF;

    SELECT p.resultado, p.archivo_probatorio_documento,
           p.huella_prevalidacion_sha256
      INTO resultado, archivo_repetido, huella_prevalidacion_repetida
      FROM vec_bolsa_baremacion.obtener_archivo_probatorio_previo_cambio_v3(
          operacion, prueba_prevalidacion, canonica_prevalidacion,
          recurso_bytes
      ) AS p;
    IF resultado <> 'activa'
       OR archivo_repetido IS DISTINCT FROM archivo
       OR huella_prevalidacion_repetida <> huella_prevalidacion THEN
        RAISE EXCEPTION 'replay exacto de prevalidacion no fue idempotente';
    END IF;

    PERFORM pg_temp.crear_decision_bolsa_prueba(
        decision_confirmacion_ref,
        'bolsa.baremacion.decision.confirmar',
        'baremacion', baremacion_ref,
        '["baremacion","decision","evidencia_transaccion"]'::jsonb,
        ahora
    );
    SELECT prueba, decision_canonica
      INTO prueba_confirmacion, canonica_confirmacion
      FROM pg_temp.decision_bolsa_prueba
     WHERE decision_ref = decision_confirmacion_ref;

    manifiesto := jsonb_build_object(
        'esquema', 'vec.bolsa.manifiesto_probatorio',
        'finalidad', 'decision_tecnica_baremacion',
        'version_esquema', 3,
        'referencia', 'manifiesto:bolsa:v3:001',
        'proceso_ref', agregado_base ->> 'proceso_ref',
        'solicitud_ref', agregado_base ->> 'solicitud_ref',
        'sujeto_ref', agregado_base ->> 'sujeto_ref',
        'baremacion_merito_ref', baremacion_ref,
        'decision_ref', 'decision-tecnica:bolsa:v3:001',
        'version_base', 1,
        'huella_version_base_sha256', base.huella_estado_sha256,
        'autorizaciones', jsonb_build_array(jsonb_build_object(
            'secuencia', 1,
            'accion', 'bolsa.baremacion.archivo.prevalidar',
            'clase_recurso', 'baremacion',
            'recurso_ref', baremacion_ref,
            'autorizacion_ref', decision_prevalidacion_ref
        )),
        'evidencias', jsonb_build_array(jsonb_build_object(
            'secuencia', 1, 'tipo', 'estado_base',
            'referencia', 'estado-base:bolsa:v3:001',
            'huella_evidencia_sha256', repeat('6', 64)
        )),
        'creado_en', creada_texto,
        'huella_manifiesto_sha256', repeat('0', 64),
        'sello_manifiesto_hmac_sha256',
            'hmac-sha256:manifiesto_v3:' || repeat('7', 64)
    );
    contenido :=
        vec_bolsa_baremacion.contenido_manifiesto_probatorio_v3(manifiesto);
    manifiesto := jsonb_set(
        manifiesto, '{huella_manifiesto_sha256}',
        to_jsonb(encode(sha256(contenido), 'hex')), false
    );
    contenido :=
        vec_bolsa_baremacion.contenido_manifiesto_probatorio_v3(manifiesto);
    material := contenido ||
        vec_bolsa_baremacion.parte_canonica_manifiesto_v3(
            manifiesto ->> 'huella_manifiesto_sha256'
        );
    representacion :=
        vec_bolsa_baremacion.parte_canonica_manifiesto_v3(
            'manifiesto_probatorio_baremacion_v3'
        ) || int8send(octet_length(material)::bigint) || material;
    preimagen := convert_to(
        'manifiesto_probatorio_baremacion_v3', 'UTF8'
    ) || decode('00', 'hex') || representacion;
    IF vec_bolsa_baremacion.archivo_unitario_manifiesto_v3_valido(
        manifiesto, contenido, representacion, preimagen
    ) IS NOT TRUE THEN
        RAISE EXCEPTION 'manifiesto funcional V3 no reconstruible';
    END IF;

    decision_tecnica := jsonb_build_object(
        'contenido', jsonb_build_object(
            'id', manifiesto ->> 'decision_ref',
            'baremacion_merito_ref', baremacion_ref,
            'proceso_ref', agregado_base ->> 'proceso_ref',
            'solicitud_ref', agregado_base ->> 'solicitud_ref',
            'sujeto_ref', agregado_base ->> 'sujeto_ref',
            'version_anterior_baremacion', 1,
            'version_baremacion', 2,
            'decisor_ref', 'per_bolsa_postgresql_prueba_000001',
            'perfil_decisor_clave', 'prf_bolsa_postgresql_prueba_000001',
            'autorizacion_ref', 'autorizacion:adopcion:v3:001',
            'finalidad_clave', 'gestion_bolsa',
            'correlacion_ref', 'correlacion:bolsa:001'
        ),
        'huella_sha256', repeat('8', 64),
        'firma', jsonb_build_object(
            'manifiesto_probatorio_ref', manifiesto ->> 'referencia',
            'huella_manifiesto_probatorio_sha256',
                manifiesto ->> 'huella_manifiesto_sha256',
            'sello_manifiesto_probatorio_hmac_sha256',
                manifiesto ->> 'sello_manifiesto_hmac_sha256',
            'documento_firmado_custodiado_ref',
                'documento-firmado:bolsa:v3:001',
            'evidencia_custodia_documento_firmado_ref',
                'evidencia-custodia:bolsa:v3:001',
            'evidencia_retencion_documento_firmado_ref',
                'evidencia-retencion:bolsa:v3:001'
        )
    );
    agregado_nuevo := jsonb_set(
        agregado_base, '{decisiones}',
        (agregado_base -> 'decisiones') ||
            jsonb_build_array(decision_tecnica), false
    );
    agregado_bytes := convert_to(agregado_nuevo::text, 'UTF8');
    operacion := jsonb_build_object(
        'esquema', 'vec.bolsa.baremacion.confirmacion-postgresql.v3',
        'huella_token_sha256', huella_token,
        'clase', 'incorporar_decision',
        'version_esperada', '1',
        'huella_version_esperada_sha256', base.huella_estado_sha256,
        'huella_solicitud_hmac', huella_confirmacion_hmac,
        'huella_efecto_sha256', huella_confirmacion,
        'huella_confirmacion_sha256', huella_confirmacion,
        'huella_efecto_prevalidacion_sha256',
            huella_efecto_prevalidacion,
        'huella_agregado_sha256',
            encode(sha256(agregado_bytes), 'hex'),
        'motivo_clave', 'revision_tecnica',
        'motivo', 'Revision tecnica de integracion V3',
        'confirmada_en', confirmada_texto,
        'auditoria_ref', 'auditoria:bolsa:v3:002',
        'evento_outbox_ref', 'evento:bolsa:v3:002',
        'autorizacion_prevalidacion_ref', decision_prevalidacion_ref
    );
    SELECT c.resultado, c.numero_version,
           c.huella_estado_sha256,
           c.archivo_probatorio_documento,
           c.huella_prevalidacion_sha256
      INTO resultado, numero, huella, archivo,
           huella_prevalidacion_final
      FROM vec_bolsa_baremacion.confirmar_cambio_con_archivo_probatorio_v3(
          operacion, prueba_confirmacion, canonica_confirmacion,
          recurso_bytes, agregado_bytes, manifiesto, contenido,
          representacion, preimagen, huella_prevalidacion
      ) AS c;
    IF resultado <> 'confirmada' OR numero <> '2'
       OR huella <> encode(sha256(agregado_bytes), 'hex')
       OR archivo ->> 'numero_version' <> '2'
       OR jsonb_array_length(archivo -> 'manifiestos') <> 1
       OR archivo -> 'manifiestos' -> 0 -> 'manifiesto'
          IS DISTINCT FROM manifiesto
       OR huella_prevalidacion_final = huella_prevalidacion
       OR vec_bolsa_baremacion.huella_sha256_valida(
           huella_prevalidacion_final
       ) IS NOT TRUE THEN
        RAISE EXCEPTION 'confirmacion V3 o archivo N-1 divergente: %', resultado;
    END IF;
    IF huella_reserva_hmac = huella_confirmacion_hmac
       OR (SELECT huella_solicitud_hmac
             FROM vec_bolsa_baremacion.auditoria
            WHERE referencia = 'auditoria:bolsa:v3:002')
          <> huella_confirmacion_hmac
       OR (SELECT huella_solicitud_hmac
             FROM vec_bolsa_baremacion.reserva_version AS version_reserva
            WHERE version_reserva.reserva_ref = 'reserva:bolsa:v3:001'
              AND version_reserva.version = 1)
          <> huella_reserva_hmac THEN
        RAISE EXCEPTION 'HMAC reserva/confirmacion no permanecen separados';
    END IF;

    SELECT p.resultado, p.archivo_probatorio_documento,
           p.huella_prevalidacion_sha256
      INTO resultado, archivo_repetido, huella_prevalidacion_repetida
      FROM vec_bolsa_baremacion.obtener_archivo_probatorio_previo_cambio_v3(
          operacion_prevalidacion,
          prueba_prevalidacion, canonica_prevalidacion, recurso_bytes
      ) AS p;
    IF resultado <> 'confirmada'
       OR archivo_repetido IS DISTINCT FROM archivo
       OR huella_prevalidacion_repetida <> huella_prevalidacion_final THEN
        RAISE EXCEPTION 'recuperacion tras respuesta perdida divergente: %',
            resultado;
    END IF;

    INSERT INTO vec_prueba_bolsa_baremacion_v3.recuperacion (
        operacion_prevalidacion, prueba_prevalidacion,
        canonica_prevalidacion, recurso_canonico,
        operacion_confirmacion, prueba_confirmacion,
        canonica_confirmacion, agregado_canonico, manifiesto,
        contenido_manifiesto, representacion_manifiesto,
        preimagen_manifiesto, huella_prevalidacion_entrada,
        archivo_esperado, huella_prevalidacion_final
    ) VALUES (
        operacion_prevalidacion, prueba_prevalidacion,
        canonica_prevalidacion, recurso_bytes,
        operacion, prueba_confirmacion, canonica_confirmacion,
        agregado_bytes, manifiesto, contenido, representacion, preimagen,
        huella_prevalidacion, archivo, huella_prevalidacion_final
    );

    BEGIN
        UPDATE vec_bolsa_baremacion.manifiesto_probatorio_v3 AS almacenado
           SET registrado_en = clock_timestamp()
         WHERE almacenado.referencia = 'manifiesto:bolsa:v3:001';
        RAISE EXCEPTION 'se permitio mutar manifiesto append-only';
    EXCEPTION WHEN sqlstate '55000' THEN
        NULL;
    END;
    IF vec_bolsa_baremacion.archivo_unitario_manifiesto_v3_valido(
        manifiesto, contenido, set_byte(representacion, 0, 1), preimagen
    ) IS NOT FALSE THEN
        RAISE EXCEPTION 'corrupcion bytea no fue detectada';
    END IF;
END
$flujo_v3$;

DO $acl_v3$
BEGIN
    IF has_function_privilege(
           'vec_bolsa_baremacion_ejecutor',
           'vec_bolsa_baremacion.confirmar_cambio(jsonb,jsonb,bytea,bytea,bytea)',
           'EXECUTE'
       ) OR NOT has_function_privilege(
           'vec_bolsa_baremacion_ejecutor',
           'vec_bolsa_baremacion.confirmar_cambio_con_archivo_probatorio_v3(jsonb,jsonb,bytea,bytea,bytea,jsonb,bytea,bytea,bytea,text)',
           'EXECUTE'
       ) OR has_function_privilege(
           'vec_bolsa_baremacion_ejecutor',
           'vec_bolsa_baremacion.construir_archivo_probatorio_v3(text,numeric)',
           'EXECUTE'
       ) OR has_table_privilege(
           'vec_bolsa_baremacion_ejecutor',
           'vec_bolsa_baremacion.manifiesto_probatorio_v3',
           'SELECT'
       ) THEN
        RAISE EXCEPTION 'ACL V3 no es minima';
    END IF;
END
$acl_v3$;

\if :{?CONFIRMAR_FIXTURE_V3}
    \if :CONFIRMAR_FIXTURE_V3
        COMMIT;
    \else
        ROLLBACK;
    \endif
\else
    ROLLBACK;
\endif
