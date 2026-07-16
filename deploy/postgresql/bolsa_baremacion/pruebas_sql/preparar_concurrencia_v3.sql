-- Prepara en una unica transaccion las dos autorizaciones de la carrera.
-- La carrera posterior no ejecuta DDL y mide solo serializacion funcional.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

\ir fixture_decision_v3.sql

CREATE TABLE vec_prueba_bolsa_baremacion_v3.entrada_concurrencia (
    sufijo text PRIMARY KEY,
    operacion jsonb NOT NULL,
    prueba jsonb NOT NULL,
    decision_canonica bytea NOT NULL,
    recurso_canonico bytea NOT NULL
);
CREATE TABLE vec_prueba_bolsa_baremacion_v3.resultado_concurrencia (
    sufijo text PRIMARY KEY,
    resultado text NOT NULL CHECK (resultado IN ('reservada', 'en_curso')),
    reserva_ref text NOT NULL
);
REVOKE ALL ON
    vec_prueba_bolsa_baremacion_v3.entrada_concurrencia,
    vec_prueba_bolsa_baremacion_v3.resultado_concurrencia
    FROM PUBLIC;

DO $preparar_carrera_v3$
DECLARE
    sufijo text;
    ahora timestamptz(6);
    referencia_decision text;
    prueba jsonb;
    canonica bytea;
    recurso bytea := convert_to(
        '{"ambitos":{"sujeto_ref":"sujeto:bolsa:001"},"atributos":{}}',
        'UTF8'
    );
    base record;
    operacion jsonb;
BEGIN
    SELECT actual.numero, actual.huella_estado_sha256
      INTO STRICT base
      FROM vec_bolsa_baremacion.baremacion_actual AS actual
     WHERE actual.baremacion_merito_ref = 'baremacion:001';
    IF base.numero <> 2 THEN
        RAISE EXCEPTION 'la carrera exige fixture confirmado V2';
    END IF;
    FOREACH sufijo IN ARRAY ARRAY['a', 'b'] LOOP
        ahora := clock_timestamp();
        referencia_decision := 'decision:concurrencia:v3:' || sufijo;
        PERFORM pg_temp.crear_decision_bolsa_prueba(
            referencia_decision, 'bolsa.baremacion.decision.reservar',
            'baremacion', 'baremacion:001',
            '["reserva.decision"]'::jsonb, ahora
        );
        SELECT decision.prueba, decision.decision_canonica
          INTO STRICT prueba, canonica
          FROM pg_temp.decision_bolsa_prueba AS decision
         WHERE decision.decision_ref = referencia_decision;
        operacion := jsonb_build_object(
            'esquema', 'vec.bolsa.baremacion.reserva-postgresql.v3',
            'reserva_ref', 'reserva:concurrencia:v3:' || sufijo,
            'huella_token_sha256', encode(sha256(convert_to(
                'token:concurrencia:v3:' || sufijo, 'UTF8'
            )), 'hex'),
            'ambito_idempotencia_sha256', encode(sha256(convert_to(
                'ambito:concurrencia:v3:' || sufijo, 'UTF8'
            )), 'hex'),
            'clase', 'incorporar_decision',
            'baremacion_merito_ref', 'baremacion:001',
            'version_esperada', '2',
            'huella_version_esperada_sha256', base.huella_estado_sha256,
            'huella_solicitud_hmac', 'hmac-sha256:reserva_v1:' ||
                encode(sha256(convert_to(
                    'hmac:concurrencia:v3:' || sufijo, 'UTF8'
                )), 'hex'),
            'huella_efecto_sha256', encode(sha256(convert_to(
                'efecto:concurrencia:v3:' || sufijo, 'UTF8'
            )), 'hex'),
            'solicitada_en', to_char(
                ahora - interval '1 second',
                'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
            ),
            'expira_en', to_char(
                ahora + interval '5 minutes',
                'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
            )
        );
        INSERT INTO
            vec_prueba_bolsa_baremacion_v3.entrada_concurrencia (
                sufijo, operacion, prueba, decision_canonica, recurso_canonico
            )
        VALUES (sufijo, operacion, prueba, canonica, recurso);
    END LOOP;
END
$preparar_carrera_v3$;

COMMIT;
