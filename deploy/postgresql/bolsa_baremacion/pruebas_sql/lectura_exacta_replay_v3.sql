-- Prueba la primera lectura de una version concreta y su repeticion exacta.
-- La misma autorizacion solo puede dejar un uso durable y ambas respuestas
-- deben ser identicas, incluido el archivo probatorio reconstruido.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

\ir fixture_decision_v3.sql

DO $lectura_exacta_replay_v3$
DECLARE
    ahora timestamptz(6) := clock_timestamp();
    referencia_decision text := 'decision:lectura-exacta:v3:001';
    referencia_decision_ausente text :=
        'decision:lectura-exacta-ausente:v3:001';
    recurso bytea := convert_to(
        '{"ambitos":{"sujeto_ref":"sujeto:bolsa:001"},"atributos":{}}',
        'UTF8'
    );
    operacion jsonb := jsonb_build_object(
        'esquema', 'vec.bolsa.baremacion.lectura-version-postgresql.v3',
        'baremacion_merito_ref', 'baremacion:001',
        'numero_version', '2',
        'huella_efecto_sha256', encode(sha256(convert_to(
            'efecto:lectura-exacta:v3:001', 'UTF8'
        )), 'hex')
    );
    prueba jsonb;
    decision_canonica bytea;
    prueba_ausente jsonb;
    decision_canonica_ausente bytea;
    primera record;
    repetida record;
    ausente record;
BEGIN
    PERFORM pg_temp.crear_decision_bolsa_prueba(
        referencia_decision, 'bolsa.baremacion.version.consultar',
        'baremacion', 'baremacion:001', '["baremacion"]'::jsonb, ahora
    );
    SELECT decision.prueba, decision.decision_canonica
      INTO STRICT prueba, decision_canonica
      FROM pg_temp.decision_bolsa_prueba AS decision
     WHERE decision.decision_ref = referencia_decision;

    SELECT * INTO STRICT primera
      FROM vec_bolsa_baremacion.obtener_version_con_archivo_probatorio_v3(
          operacion, prueba, decision_canonica, recurso
      );
    IF primera.resultado IS DISTINCT FROM 'obtenida'
       OR primera.numero_version IS DISTINCT FROM '2'
       OR primera.archivo_probatorio_documento IS NULL
       OR primera.archivo_probatorio_documento ->> 'numero_version' <> '2'
       OR (SELECT count(*)
             FROM vec_bolsa_baremacion.uso_decision AS uso
            WHERE uso.decision_ref = referencia_decision)
          <> 1 THEN
        RAISE EXCEPTION 'primera lectura exacta V3 divergente: %',
            primera.resultado;
    END IF;

    SELECT * INTO STRICT repetida
      FROM vec_bolsa_baremacion.obtener_version_con_archivo_probatorio_v3(
          operacion, prueba, decision_canonica, recurso
      );
    IF repetida.resultado IS DISTINCT FROM primera.resultado
       OR repetida.numero_version IS DISTINCT FROM primera.numero_version
       OR repetida.huella_estado_sha256 IS DISTINCT FROM
          primera.huella_estado_sha256
       OR repetida.agregado_canonico IS DISTINCT FROM
          primera.agregado_canonico
       OR repetida.confirmada_en IS DISTINCT FROM primera.confirmada_en
       OR repetida.auditoria_ref IS DISTINCT FROM primera.auditoria_ref
       OR repetida.archivo_probatorio_documento IS DISTINCT FROM
          primera.archivo_probatorio_documento
       OR (SELECT count(*)
             FROM vec_bolsa_baremacion.uso_decision AS uso
            WHERE uso.decision_ref = referencia_decision)
          <> 1 THEN
        RAISE EXCEPTION 'replay exacto de lectura V3 no fue idempotente';
    END IF;

    PERFORM pg_temp.crear_decision_bolsa_prueba(
        referencia_decision_ausente,
        'bolsa.baremacion.version.consultar', 'baremacion',
        'baremacion:001', '["baremacion"]'::jsonb, clock_timestamp()
    );
    SELECT decision.prueba, decision.decision_canonica
      INTO STRICT prueba_ausente, decision_canonica_ausente
      FROM pg_temp.decision_bolsa_prueba AS decision
     WHERE decision.decision_ref = referencia_decision_ausente;
    SELECT * INTO STRICT ausente
      FROM vec_bolsa_baremacion.obtener_version_con_archivo_probatorio_v3(
          operacion || '{"numero_version":"999"}'::jsonb,
          prueba_ausente, decision_canonica_ausente, recurso
      );
    IF ausente.resultado IS DISTINCT FROM 'no_encontrada'
       OR ausente.numero_version IS DISTINCT FROM ''
       OR ausente.huella_estado_sha256 IS DISTINCT FROM ''
       OR ausente.agregado_canonico IS NOT NULL
       OR ausente.confirmada_en IS NOT NULL
       OR ausente.auditoria_ref IS DISTINCT FROM ''
       OR ausente.archivo_probatorio_documento IS NOT NULL
       OR EXISTS (
           SELECT 1
             FROM vec_bolsa_baremacion.uso_decision AS uso
            WHERE uso.decision_ref = referencia_decision_ausente
       ) THEN
        RAISE EXCEPTION 'lectura exacta ausente V3 no fallo limpiamente: %',
            ausente.resultado;
    END IF;
END
$lectura_exacta_replay_v3$;

COMMIT;
