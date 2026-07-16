-- Pruebas adversarias y de cardinalidad maxima del archivo probatorio V3.
-- Todo el escenario se revierte: sustituye temporalmente el manifiesto del
-- fixture confirmado para ejercitar los triggers con 4096+4096 hijos reales.
\set VERBOSITY verbose

BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL statement_timeout = '15s';

DO $vectores_cerrados_v3$
DECLARE
    manifiesto jsonb;
BEGIN
    SELECT almacenado.manifiesto
      INTO STRICT manifiesto
      FROM vec_bolsa_baremacion.manifiesto_probatorio_v3 AS almacenado
     WHERE almacenado.baremacion_merito_ref = 'baremacion:001'
       AND almacenado.numero_version = 2;

    IF vec_bolsa_baremacion.referencia_manifiesto_v3_valida(
           'referencia:ASCII_!~', 512
       ) IS NOT TRUE
       OR vec_bolsa_baremacion.referencia_manifiesto_v3_valida(
           E'referencia\tcontrol', 512
       ) IS NOT FALSE
       OR vec_bolsa_baremacion.referencia_manifiesto_v3_valida(
           convert_from(decode('7f', 'hex'), 'UTF8'), 512
       ) IS NOT FALSE
       OR vec_bolsa_baremacion.referencia_manifiesto_v3_valida(
           'referencia:á', 512
       ) IS NOT FALSE
       OR vec_bolsa_baremacion.referencia_manifiesto_v3_valida(
           'referencia:*', 512
       ) IS NOT FALSE
       OR vec_bolsa_baremacion.sello_hmac_manifiesto_v3_valido(
           'hmac-sha256:' || repeat('a', 128) || ':' || repeat('0', 64)
       ) IS NOT TRUE
       OR vec_bolsa_baremacion.sello_hmac_manifiesto_v3_valido(
           'hmac-sha256:' || repeat('a', 129) || ':' || repeat('0', 64)
       ) IS NOT FALSE
       OR vec_bolsa_baremacion.instante_canonico_manifiesto_v3(
           '0001-01-01T00:00:00.000000001Z'
       ) IS NOT TRUE
       OR vec_bolsa_baremacion.instante_canonico_manifiesto_v3(
           '9999-12-31T23:59:59.999999999Z'
       ) IS NOT TRUE
       OR vec_bolsa_baremacion.instante_canonico_manifiesto_v3(
           '0001-01-01T00:00:00Z'
       ) IS NOT FALSE
       OR vec_bolsa_baremacion.instante_canonico_manifiesto_v3(
           '0000-01-01T00:00:00Z'
       ) IS NOT FALSE
       OR vec_bolsa_baremacion.instante_canonico_manifiesto_v3(
           '10000-01-01T00:00:00Z'
       ) IS NOT FALSE
       OR vec_bolsa_baremacion.instante_canonico_manifiesto_v3(
           '2026-07-16T24:00:00Z'
       ) IS NOT FALSE
       OR vec_bolsa_baremacion.instante_canonico_manifiesto_v3(
           '2026-07-16T09:00:60Z'
       ) IS NOT FALSE
       OR vec_bolsa_baremacion.instante_canonico_manifiesto_v3(
           '2026-07-16T09:00:00.123400Z'
       ) IS NOT FALSE
       OR vec_bolsa_baremacion.contenido_manifiesto_probatorio_v3(
           jsonb_set(manifiesto, '{version_esquema}', '"3"'::jsonb)
       ) IS NOT NULL
       OR vec_bolsa_baremacion.contenido_manifiesto_probatorio_v3(
           jsonb_set(manifiesto, '{referencia}', '3'::jsonb)
       ) IS NOT NULL
       OR vec_bolsa_baremacion.contenido_manifiesto_probatorio_v3(
           jsonb_set(manifiesto, '{referencia}', 'null'::jsonb)
       ) IS NOT NULL
       OR vec_bolsa_baremacion.contenido_manifiesto_probatorio_v3(
           jsonb_set(
               manifiesto, '{huella_version_base_sha256}', '3'::jsonb
           )
       ) IS NOT NULL
       OR vec_bolsa_baremacion.contenido_manifiesto_probatorio_v3(
           jsonb_set(
               manifiesto, '{huella_version_base_sha256}', 'null'::jsonb
           )
       ) IS NOT NULL
       OR vec_bolsa_baremacion.contenido_manifiesto_probatorio_v3(
           jsonb_set(
               manifiesto, '{huella_manifiesto_sha256}', '3'::jsonb
           )
       ) IS NOT NULL
       OR vec_bolsa_baremacion.contenido_manifiesto_probatorio_v3(
           jsonb_set(
               manifiesto, '{sello_manifiesto_hmac_sha256}', '3'::jsonb
           )
       ) IS NOT NULL
       OR vec_bolsa_baremacion.contenido_manifiesto_probatorio_v3(
           manifiesto || '{"campo_desconocido":true}'::jsonb
       ) IS NOT NULL
       OR vec_bolsa_baremacion.contenido_manifiesto_probatorio_v3(
           jsonb_set(
               manifiesto, '{autorizaciones,0,secuencia}', '"1"'::jsonb
           )
       ) IS NOT NULL
       OR vec_bolsa_baremacion.contenido_manifiesto_probatorio_v3(
           jsonb_set(
               manifiesto, '{autorizaciones,0,secuencia}', 'null'::jsonb
           )
       ) IS NOT NULL
       OR vec_bolsa_baremacion.contenido_manifiesto_probatorio_v3(
           jsonb_set(
               manifiesto, '{autorizaciones,0}',
               (manifiesto #> '{autorizaciones,0}') ||
                   '{"campo_desconocido":true}'::jsonb
           )
       ) IS NOT NULL
       OR vec_bolsa_baremacion.contenido_manifiesto_probatorio_v3(
           jsonb_set(
               manifiesto, '{autorizaciones,0,accion}', '3'::jsonb
           )
       ) IS NOT NULL
       OR vec_bolsa_baremacion.contenido_manifiesto_probatorio_v3(
           jsonb_set(
               manifiesto, '{autorizaciones,0,recurso_ref}', 'null'::jsonb
           )
       ) IS NOT NULL
       OR vec_bolsa_baremacion.contenido_manifiesto_probatorio_v3(
           jsonb_set(
               manifiesto, '{evidencias,0,secuencia}', '"1"'::jsonb
           )
       ) IS NOT NULL
       OR vec_bolsa_baremacion.contenido_manifiesto_probatorio_v3(
           jsonb_set(
               manifiesto, '{evidencias,0,secuencia}', 'null'::jsonb
           )
       ) IS NOT NULL
       OR vec_bolsa_baremacion.contenido_manifiesto_probatorio_v3(
           jsonb_set(
               manifiesto, '{evidencias,0}',
               (manifiesto #> '{evidencias,0}') ||
                   '{"campo_desconocido":true}'::jsonb
           )
       ) IS NOT NULL
       OR vec_bolsa_baremacion.contenido_manifiesto_probatorio_v3(
           jsonb_set(
               manifiesto, '{evidencias,0,tipo}', '3'::jsonb
           )
       ) IS NOT NULL
       OR vec_bolsa_baremacion.contenido_manifiesto_probatorio_v3(
           jsonb_set(
               manifiesto, '{evidencias,0,referencia}', 'null'::jsonb
           )
       ) IS NOT NULL
       OR vec_bolsa_baremacion.contenido_manifiesto_probatorio_v3(
           jsonb_set(
               manifiesto, '{evidencias,0,huella_evidencia_sha256}',
               '3'::jsonb
           )
       ) IS NOT NULL THEN
        RAISE EXCEPTION 'perfil cerrado V3 divergente de Go';
    END IF;
END
$vectores_cerrados_v3$;

CREATE TEMP TABLE pg_temp.manifiesto_original_v3 ON COMMIT DROP AS
SELECT *
  FROM vec_bolsa_baremacion.manifiesto_probatorio_v3
 WHERE baremacion_merito_ref = 'baremacion:001'
   AND numero_version = 2;

CREATE TEMP TABLE pg_temp.cronometro_cardinalidad_v3 ON COMMIT DROP AS
SELECT clock_timestamp() AS inicio;

DO $preparar_cardinalidad_maxima_v3$
DECLARE
    original vec_bolsa_baremacion.manifiesto_probatorio_v3%ROWTYPE;
    autorizaciones jsonb;
    evidencias jsonb;
    manifiesto jsonb;
    contenido bytea;
    material bytea;
    representacion bytea;
    preimagen bytea;
    huella text;
BEGIN
    SELECT * INTO STRICT original FROM pg_temp.manifiesto_original_v3;
    SELECT jsonb_agg(jsonb_build_object(
               'secuencia', numero,
               'accion', 'bolsa.baremacion.decision.confirmar',
               'clase_recurso', 'baremacion',
               'recurso_ref', 'recurso:escalabilidad:' || numero,
               'autorizacion_ref', 'autorizacion:escalabilidad:' || numero
           ) ORDER BY numero)
      INTO autorizaciones
      FROM generate_series(1, 4096) AS serie(numero);
    SELECT jsonb_agg(jsonb_build_object(
               'secuencia', numero,
               'tipo', 'documento_merito',
               'referencia', 'evidencia:escalabilidad:' || numero,
               'huella_evidencia_sha256', encode(sha256(convert_to(
                   'evidencia:escalabilidad:' || numero, 'UTF8'
               )), 'hex')
           ) ORDER BY numero)
      INTO evidencias
      FROM generate_series(1, 4096) AS serie(numero);
    manifiesto := original.manifiesto || jsonb_build_object(
        'referencia', 'manifiesto:escalabilidad:v3',
        'decision_ref', 'decision:escalabilidad:v3',
        'creado_en', '2026-07-16T09:00:00.000000001Z',
        'autorizaciones', autorizaciones,
        'evidencias', evidencias,
        'huella_manifiesto_sha256', repeat('0', 64),
        'sello_manifiesto_hmac_sha256',
            'hmac-sha256:manifiesto_v3:' || repeat('a', 64)
    );
    contenido :=
        vec_bolsa_baremacion.contenido_manifiesto_probatorio_v3(manifiesto);
    IF contenido IS NULL THEN
        RAISE EXCEPTION 'no se construyo contenido 4096+4096';
    END IF;
    huella := encode(sha256(contenido), 'hex');
    manifiesto := jsonb_set(
        manifiesto, '{huella_manifiesto_sha256}', to_jsonb(huella), false
    );
    IF vec_bolsa_baremacion.contenido_manifiesto_probatorio_v3(manifiesto)
       IS DISTINCT FROM contenido THEN
        RAISE EXCEPTION 'la huella altero el contenido canonico';
    END IF;
    material := contenido ||
        vec_bolsa_baremacion.parte_canonica_manifiesto_v3(huella);
    representacion :=
        vec_bolsa_baremacion.parte_canonica_manifiesto_v3(
            'manifiesto_probatorio_baremacion_v3'
        ) || int8send(octet_length(material)::bigint) || material;
    preimagen := convert_to(
        'manifiesto_probatorio_baremacion_v3', 'UTF8'
    ) || decode('00', 'hex') || representacion;

    EXECUTE 'ALTER TABLE vec_bolsa_baremacion.manifiesto_autorizacion_v3 DISABLE TRIGGER manifiesto_autorizacion_v3_inmutable';
    EXECUTE 'ALTER TABLE vec_bolsa_baremacion.manifiesto_evidencia_v3 DISABLE TRIGGER manifiesto_evidencia_v3_inmutable';
    EXECUTE 'ALTER TABLE vec_bolsa_baremacion.manifiesto_probatorio_v3 DISABLE TRIGGER manifiesto_probatorio_v3_inmutable';
    DELETE FROM vec_bolsa_baremacion.manifiesto_autorizacion_v3
     WHERE manifiesto_ref = original.referencia;
    DELETE FROM vec_bolsa_baremacion.manifiesto_evidencia_v3
     WHERE manifiesto_ref = original.referencia;
    DELETE FROM vec_bolsa_baremacion.manifiesto_probatorio_v3
     WHERE referencia = original.referencia;
    EXECUTE 'ALTER TABLE vec_bolsa_baremacion.manifiesto_autorizacion_v3 ENABLE TRIGGER manifiesto_autorizacion_v3_inmutable';
    EXECUTE 'ALTER TABLE vec_bolsa_baremacion.manifiesto_evidencia_v3 ENABLE TRIGGER manifiesto_evidencia_v3_inmutable';
    EXECUTE 'ALTER TABLE vec_bolsa_baremacion.manifiesto_probatorio_v3 ENABLE TRIGGER manifiesto_probatorio_v3_inmutable';

    INSERT INTO vec_bolsa_baremacion.manifiesto_probatorio_v3 (
        referencia, baremacion_merito_ref, numero_version, decision_ref,
        version_base, huella_version_base_sha256, auditoria_ref,
        evento_outbox_ref, reserva_ref, total_autorizaciones,
        total_evidencias, manifiesto, contenido_manifiesto_canonico,
        representacion_manifiesto_canonica, preimagen_hmac_manifiesto,
        huella_manifiesto_sha256, sello_manifiesto_hmac_sha256,
        registrado_en
    ) VALUES (
        manifiesto ->> 'referencia', original.baremacion_merito_ref,
        original.numero_version, manifiesto ->> 'decision_ref',
        original.version_base, original.huella_version_base_sha256,
        original.auditoria_ref, original.evento_outbox_ref,
        original.reserva_ref, 4096, 4096, manifiesto, contenido,
        representacion, preimagen, huella,
        manifiesto ->> 'sello_manifiesto_hmac_sha256', clock_timestamp()
    );
END
$preparar_cardinalidad_maxima_v3$;

DO $adversarios_hijos_v3$
DECLARE
    campo text;
BEGIN
    BEGIN
        INSERT INTO vec_bolsa_baremacion.manifiesto_autorizacion_v3
            (manifiesto_ref, secuencia, accion, clase_recurso,
             recurso_ref, autorizacion_ref)
        VALUES (
            'manifiesto:inexistente:v3', 1,
            'bolsa.baremacion.decision.confirmar', 'baremacion',
            'recurso:inexistente:v3', 'autorizacion:inexistente:v3'
        );
        RAISE EXCEPTION 'se acepto autorizacion huerfana';
    EXCEPTION WHEN foreign_key_violation THEN
        NULL;
    END;
    FOREACH campo IN ARRAY ARRAY[
        'secuencia', 'accion', 'clase_recurso',
        'recurso_ref', 'autorizacion_ref'
    ] LOOP
        BEGIN
            INSERT INTO vec_bolsa_baremacion.manifiesto_autorizacion_v3
                (manifiesto_ref, secuencia, accion, clase_recurso,
                 recurso_ref, autorizacion_ref)
            VALUES (
                'manifiesto:escalabilidad:v3',
                CASE WHEN campo = 'secuencia' THEN 2 ELSE 1 END,
                CASE WHEN campo = 'accion'
                    THEN 'bolsa.baremacion.decision.reservar'
                    ELSE 'bolsa.baremacion.decision.confirmar' END,
                CASE WHEN campo = 'clase_recurso'
                    THEN 'proceso' ELSE 'baremacion' END,
                CASE WHEN campo = 'recurso_ref'
                    THEN 'recurso:divergente:v3'
                    ELSE 'recurso:escalabilidad:1' END,
                CASE WHEN campo = 'autorizacion_ref'
                    THEN 'autorizacion:divergente:v3'
                    ELSE 'autorizacion:escalabilidad:1' END
            );
            RAISE EXCEPTION
                'se acepto autorizacion divergente en %', campo;
        EXCEPTION WHEN check_violation THEN
            NULL;
        END;
    END LOOP;
    BEGIN
        INSERT INTO vec_bolsa_baremacion.manifiesto_autorizacion_v3
            (manifiesto_ref, secuencia, accion, clase_recurso,
             recurso_ref, autorizacion_ref)
        VALUES (
            'manifiesto:escalabilidad:v3', 0,
            'bolsa.baremacion.decision.confirmar', 'baremacion',
            'recurso:cero:v3', 'autorizacion:cero:v3'
        );
        RAISE EXCEPTION 'se acepto secuencia cero';
    EXCEPTION WHEN check_violation THEN
        NULL;
    END;
    FOREACH campo IN ARRAY ARRAY[
        'secuencia', 'tipo', 'referencia', 'huella_evidencia_sha256'
    ] LOOP
        BEGIN
            INSERT INTO vec_bolsa_baremacion.manifiesto_evidencia_v3
                (manifiesto_ref, secuencia, tipo, referencia,
                 huella_evidencia_sha256)
            VALUES (
                'manifiesto:escalabilidad:v3',
                CASE WHEN campo = 'secuencia' THEN 2 ELSE 1 END,
                CASE WHEN campo = 'tipo'
                    THEN 'calculo_oficial' ELSE 'documento_merito' END,
                CASE WHEN campo = 'referencia'
                    THEN 'evidencia:divergente:v3'
                    ELSE 'evidencia:escalabilidad:1' END,
                CASE WHEN campo = 'huella_evidencia_sha256'
                    THEN repeat('0', 64)
                    ELSE encode(sha256(convert_to(
                        'evidencia:escalabilidad:1', 'UTF8'
                    )), 'hex') END
            );
            RAISE EXCEPTION 'se acepto evidencia divergente en %', campo;
        EXCEPTION WHEN check_violation THEN
            NULL;
        END;
    END LOOP;
END
$adversarios_hijos_v3$;

INSERT INTO vec_bolsa_baremacion.manifiesto_autorizacion_v3 (
    manifiesto_ref, secuencia, accion, clase_recurso,
    recurso_ref, autorizacion_ref
)
SELECT almacenado.referencia, (elemento ->> 'secuencia')::integer,
       elemento ->> 'accion', elemento ->> 'clase_recurso',
       elemento ->> 'recurso_ref', elemento ->> 'autorizacion_ref'
  FROM vec_bolsa_baremacion.manifiesto_probatorio_v3 AS almacenado
  CROSS JOIN LATERAL jsonb_array_elements(
      almacenado.manifiesto -> 'autorizaciones'
  ) AS lista(elemento)
 WHERE almacenado.referencia = 'manifiesto:escalabilidad:v3';

INSERT INTO vec_bolsa_baremacion.manifiesto_evidencia_v3 (
    manifiesto_ref, secuencia, tipo, referencia, huella_evidencia_sha256
)
SELECT almacenado.referencia, (elemento ->> 'secuencia')::integer,
       elemento ->> 'tipo', elemento ->> 'referencia',
       elemento ->> 'huella_evidencia_sha256'
  FROM vec_bolsa_baremacion.manifiesto_probatorio_v3 AS almacenado
  CROSS JOIN LATERAL jsonb_array_elements(
      almacenado.manifiesto -> 'evidencias'
  ) AS lista(elemento)
 WHERE almacenado.referencia = 'manifiesto:escalabilidad:v3';

SET CONSTRAINTS
    vec_bolsa_baremacion.manifiesto_probatorio_v3_completitud IMMEDIATE;

DO $posteriores_y_archivo_v3$
DECLARE
    archivo jsonb;
    inicio timestamptz;
BEGIN
    BEGIN
        INSERT INTO vec_bolsa_baremacion.manifiesto_autorizacion_v3
            (manifiesto_ref, secuencia, accion, clase_recurso,
             recurso_ref, autorizacion_ref)
        VALUES (
            'manifiesto:escalabilidad:v3', 1,
            'bolsa.baremacion.decision.confirmar', 'baremacion',
            'recurso:escalabilidad:1', 'autorizacion:escalabilidad:1'
        );
        RAISE EXCEPTION 'se acepto duplicado posterior';
    EXCEPTION WHEN unique_violation THEN
        NULL;
    END;
    BEGIN
        INSERT INTO vec_bolsa_baremacion.manifiesto_evidencia_v3
            (manifiesto_ref, secuencia, tipo, referencia,
             huella_evidencia_sha256)
        VALUES (
            'manifiesto:escalabilidad:v3', 4097, 'documento_merito',
            'evidencia:posterior:v3', repeat('0', 64)
        );
        RAISE EXCEPTION 'se acepto append N+1 posterior';
    EXCEPTION WHEN check_violation THEN
        NULL;
    END;
    SELECT vec_bolsa_baremacion.construir_archivo_probatorio_v3(
               'baremacion:001', 2
           )
      INTO archivo;
    IF archivo IS NULL
       OR jsonb_array_length(archivo -> 'manifiestos') <> 1
       OR archivo ->> 'huella_archivo_sha256' !~ '^[0-9a-f]{64}$' THEN
        RAISE EXCEPTION 'archivo de cardinalidad maxima invalido';
    END IF;
    SELECT cronometro.inicio
      INTO STRICT inicio
      FROM pg_temp.cronometro_cardinalidad_v3 AS cronometro;
    IF clock_timestamp() - inicio >= interval '15 seconds' THEN
        RAISE EXCEPTION
            'flujo 4096+4096 y archivo excedio 15 segundos: %',
            clock_timestamp() - inicio;
    END IF;
    RAISE NOTICE 'flujo 4096+4096 y archivo completado en %',
        clock_timestamp() - inicio;
END
$posteriores_y_archivo_v3$;

-- Mide el mismo primitivo lineal usado por construir_archivo_probatorio_v3
-- en el limite material de 64 MiB, con una huella calculada externamente.
DO $agregacion_archivo_64m_v3$
DECLARE
    material bytea;
BEGIN
    SELECT string_agg(
               convert_to(repeat('a', 16384), 'UTF8'),
               ''::bytea ORDER BY numero
           )
      INTO material
      FROM generate_series(1, 4096) AS serie(numero);
    IF octet_length(material) <> 67108864
       OR encode(sha256(material), 'hex') <>
          'fae972222d455a2eaee1661ad9625502ec3bfc5ec38b87a6eec5afd5107331b5' THEN
        RAISE EXCEPTION 'agregacion lineal de 64 MiB divergente';
    END IF;
END
$agregacion_archivo_64m_v3$;

ROLLBACK;
