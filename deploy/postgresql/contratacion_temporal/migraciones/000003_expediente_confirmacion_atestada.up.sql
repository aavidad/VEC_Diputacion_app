BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:000003_confirmacion_atestada', 0
    )
);

DO $prevalidacion$
BEGIN
    IF pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.preparar_alta_v2(jsonb)'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_autorizacion_atestada_v3.registrar_y_consumir_decision_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
       ) IS NULL
       OR pg_catalog.to_regclass(
           'vec_contratacion_temporal.expediente_alta'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'dependencias de confirmación atestada ausentes';
    END IF;
END
$prevalidacion$;

CREATE FUNCTION vec_contratacion_temporal.texto_json_go_v1(
    p_valor text
)
RETURNS text
LANGUAGE sql
IMMUTABLE
STRICT
SET search_path = pg_catalog
AS $funcion$
    SELECT pg_catalog.replace(
             pg_catalog.replace(
               pg_catalog.replace(pg_catalog.to_json(p_valor)::text,
                                  '&', '\u0026'),
               '<', '\u003c'),
             '>', '\u003e')
$funcion$;

-- Contrato byte-a-byte de la proyección mínima del alta. No es json.Marshal
-- del agregado: es un esquema explícito y versionado compartido con el futuro
-- adaptador O2-06.
CREATE FUNCTION vec_contratacion_temporal.reconstruir_alta_v1(
    a jsonb
)
RETURNS bytea
LANGUAGE sql
IMMUTABLE
STRICT
SET search_path = pg_catalog
AS $funcion$
    SELECT pg_catalog.convert_to(
      '{"esquema":' || vec_contratacion_temporal.texto_json_go_v1(a ->> 'esquema') ||
      ',"reserva_ref":' || vec_contratacion_temporal.texto_json_go_v1(a ->> 'reserva_ref') ||
      ',"expediente_ref":' || vec_contratacion_temporal.texto_json_go_v1(a ->> 'expediente_ref') ||
      ',"numero_visible":' || vec_contratacion_temporal.texto_json_go_v1(a ->> 'numero_visible') ||
      ',"recibo_ref":' || vec_contratacion_temporal.texto_json_go_v1(a ->> 'recibo_ref') ||
      ',"organizacion_ref":' || vec_contratacion_temporal.texto_json_go_v1(a ->> 'organizacion_ref') ||
      ',"centro_ref":' || vec_contratacion_temporal.texto_json_go_v1(a ->> 'centro_ref') ||
      ',"categoria_ref":' || vec_contratacion_temporal.texto_json_go_v1(a ->> 'categoria_ref') ||
      ',"actor_ref":' || vec_contratacion_temporal.texto_json_go_v1(a ->> 'actor_ref') ||
      ',"perfil_ref":' || vec_contratacion_temporal.texto_json_go_v1(a ->> 'perfil_ref') ||
      ',"version":' || (a -> 'version')::text ||
      ',"flujo_ref":' || vec_contratacion_temporal.texto_json_go_v1(a ->> 'flujo_ref') ||
      ',"flujo_version":' || (a -> 'flujo_version')::text ||
      ',"flujo_huella_sha256":' || vec_contratacion_temporal.texto_json_go_v1(a ->> 'flujo_huella_sha256') ||
      ',"fase_clave":' || vec_contratacion_temporal.texto_json_go_v1(a ->> 'fase_clave') ||
      ',"estado":' || vec_contratacion_temporal.texto_json_go_v1(a ->> 'estado') ||
      ',"solicitud_huella_sha256":' || vec_contratacion_temporal.texto_json_go_v1(a ->> 'solicitud_huella_sha256') ||
      ',"accion_clave":' || vec_contratacion_temporal.texto_json_go_v1(a ->> 'accion_clave') ||
      ',"unidad_ref":' || vec_contratacion_temporal.texto_json_go_v1(a ->> 'unidad_ref') ||
      ',"realizada_en":' || vec_contratacion_temporal.texto_json_go_v1(a ->> 'realizada_en') || '}',
      'UTF8'
    )
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.reconstruir_sellos_hmac_v1(
    s jsonb
)
RETURNS bytea
LANGUAGE sql
IMMUTABLE
STRICT
SET search_path = pg_catalog
AS $funcion$
    SELECT pg_catalog.convert_to(
      '{"esquema":' || vec_contratacion_temporal.texto_json_go_v1(s ->> 'esquema') ||
      ',"activo":{"generacion":' ||
        (s #> '{activo,generacion}')::text ||
      ',"ambito_hmac":' ||
        vec_contratacion_temporal.texto_json_go_v1(
            s #>> '{activo,ambito_hmac}'
        ) ||
      ',"huella_hmac":' ||
        vec_contratacion_temporal.texto_json_go_v1(
            s #>> '{activo,huella_hmac}'
        ) || '}' ||
      ',"retenidos":[' || coalesce((
          SELECT pg_catalog.string_agg(
              '{"generacion":' || (e.valor -> 'generacion')::text ||
              ',"ambito_hmac":' ||
                vec_contratacion_temporal.texto_json_go_v1(
                    e.valor ->> 'ambito_hmac'
                ) ||
              ',"huella_hmac":' ||
                vec_contratacion_temporal.texto_json_go_v1(
                    e.valor ->> 'huella_hmac'
                ) || '}',
              ',' ORDER BY e.orden
          )
            FROM pg_catalog.jsonb_array_elements(s -> 'retenidos')
                 WITH ORDINALITY AS e(valor, orden)
      ), '') || ']}',
      'UTF8'
    )
$funcion$;

CREATE TABLE vec_contratacion_temporal.expediente_alta (
    expediente_ref text PRIMARY KEY,
    reserva_ref text NOT NULL UNIQUE,
    numero_visible text NOT NULL UNIQUE,
    organizacion_ref text NOT NULL,
    actor_ref text NOT NULL,
    perfil_ref text NOT NULL,
    decision_ref text NOT NULL UNIQUE,
    efecto_ref text NOT NULL UNIQUE,
    huella_efecto_sha256 text NOT NULL,
    creada_en timestamptz(6) NOT NULL,
    FOREIGN KEY (reserva_ref)
        REFERENCES vec_contratacion_temporal.identidad_reserva_alta(
            reserva_ref
        ),
    FOREIGN KEY (decision_ref, efecto_ref, huella_efecto_sha256)
        REFERENCES vec_autorizacion_atestada_v3.consumo_decision_v3(
            decision_ref, efecto_ref, huella_efecto_sha256
        ),
    CHECK (expediente_ref ~
        '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
    CHECK (numero_visible ~ '^[0-9]{4}/[A-Za-z0-9._-]{1,40}$')
);

CREATE TABLE vec_contratacion_temporal.expediente_alta_version (
    expediente_ref text NOT NULL,
    version numeric(20, 0) NOT NULL,
    alta_canonica bytea NOT NULL,
    huella_alta_sha256 text NOT NULL UNIQUE,
    flujo_ref text NOT NULL,
    flujo_version numeric(20, 0) NOT NULL,
    flujo_huella_sha256 text NOT NULL,
    fase_clave text NOT NULL,
    estado text NOT NULL,
    solicitud_huella_sha256 text NOT NULL,
    registrada_en timestamptz(6) NOT NULL,
    PRIMARY KEY (expediente_ref, version),
    FOREIGN KEY (expediente_ref)
        REFERENCES vec_contratacion_temporal.expediente_alta,
    CHECK (version BETWEEN 1 AND 9007199254740991::numeric),
    CHECK (pg_catalog.encode(
        pg_catalog.sha256(alta_canonica), 'hex'
    ) = huella_alta_sha256),
    CHECK (flujo_version BETWEEN 1 AND 9007199254740991::numeric),
    CHECK (flujo_huella_sha256 ~ '^[0-9a-f]{64}$'),
    CHECK (solicitud_huella_sha256 ~ '^[0-9a-f]{64}$'),
    CHECK (estado IN ('en_curso', 'completado', 'cancelado'))
);

CREATE TABLE vec_contratacion_temporal.actuacion_alta (
    expediente_ref text NOT NULL,
    secuencia numeric(20, 0) NOT NULL,
    version_expediente numeric(20, 0) NOT NULL,
    accion_clave text NOT NULL,
    actor_ref text NOT NULL,
    unidad_ref text NOT NULL,
    recibo_ref text NOT NULL UNIQUE,
    fase_destino text NOT NULL,
    estado_destino text NOT NULL,
    realizada_en timestamptz(6) NOT NULL,
    huella_sha256 text NOT NULL UNIQUE,
    PRIMARY KEY (expediente_ref, secuencia),
    FOREIGN KEY (expediente_ref, version_expediente)
        REFERENCES vec_contratacion_temporal.expediente_alta_version,
    CHECK (secuencia = 1 AND version_expediente = 1)
);

CREATE TABLE vec_contratacion_temporal.control_cadenas_alta (
    control_id boolean PRIMARY KEY DEFAULT true CHECK (control_id),
    secuencia_auditoria numeric(20, 0) NOT NULL,
    cabeza_auditoria_sha256 text NOT NULL,
    secuencia_outbox numeric(20, 0) NOT NULL,
    cabeza_outbox_sha256 text NOT NULL,
    actualizada_en timestamptz(6) NOT NULL
);
INSERT INTO vec_contratacion_temporal.control_cadenas_alta VALUES (
    true, 0, pg_catalog.repeat('0', 64),
    0, pg_catalog.repeat('0', 64), clock_timestamp()
);

CREATE TABLE vec_contratacion_temporal.auditoria_alta (
    auditoria_ref text PRIMARY KEY,
    secuencia numeric(20, 0) NOT NULL UNIQUE,
    expediente_ref text NOT NULL UNIQUE,
    decision_ref text NOT NULL UNIQUE,
    anterior_sha256 text NOT NULL,
    huella_sha256 text NOT NULL UNIQUE,
    registrada_en timestamptz(6) NOT NULL,
    FOREIGN KEY (expediente_ref)
        REFERENCES vec_contratacion_temporal.expediente_alta
);

CREATE TABLE vec_contratacion_temporal.outbox_alta (
    evento_ref text PRIMARY KEY,
    secuencia numeric(20, 0) NOT NULL UNIQUE,
    expediente_ref text NOT NULL UNIQUE,
    tipo_evento text NOT NULL,
    payload_canonico bytea NOT NULL,
    payload_huella_sha256 text NOT NULL UNIQUE,
    anterior_sha256 text NOT NULL,
    huella_sha256 text NOT NULL UNIQUE,
    registrada_en timestamptz(6) NOT NULL,
    publicada_en timestamptz(6),
    FOREIGN KEY (expediente_ref)
        REFERENCES vec_contratacion_temporal.expediente_alta,
    CHECK (tipo_evento =
        'contratacion_temporal.expediente.registrado.v1'),
    CHECK (pg_catalog.encode(
        pg_catalog.sha256(payload_canonico), 'hex'
    ) = payload_huella_sha256)
);

CREATE TRIGGER expediente_alta_inmutable
BEFORE UPDATE OR DELETE ON vec_contratacion_temporal.expediente_alta
FOR EACH ROW
EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();
CREATE TRIGGER expediente_alta_version_inmutable
BEFORE UPDATE OR DELETE ON vec_contratacion_temporal.expediente_alta_version
FOR EACH ROW
EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();
CREATE TRIGGER actuacion_alta_inmutable
BEFORE UPDATE OR DELETE ON vec_contratacion_temporal.actuacion_alta
FOR EACH ROW
EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();
CREATE TRIGGER auditoria_alta_inmutable
BEFORE UPDATE OR DELETE ON vec_contratacion_temporal.auditoria_alta
FOR EACH ROW
EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();

REVOKE ALL ON TABLE
    vec_contratacion_temporal.expediente_alta,
    vec_contratacion_temporal.expediente_alta_version,
    vec_contratacion_temporal.actuacion_alta,
    vec_contratacion_temporal.control_cadenas_alta,
    vec_contratacion_temporal.auditoria_alta,
    vec_contratacion_temporal.outbox_alta
    FROM PUBLIC, vec_contratacion_temporal_ejecutor;
REVOKE ALL ON FUNCTION
    vec_contratacion_temporal.texto_json_go_v1(text),
    vec_contratacion_temporal.reconstruir_alta_v1(jsonb),
    vec_contratacion_temporal.reconstruir_sellos_hmac_v1(jsonb)
    FROM PUBLIC, vec_contratacion_temporal_ejecutor;

COMMIT;
