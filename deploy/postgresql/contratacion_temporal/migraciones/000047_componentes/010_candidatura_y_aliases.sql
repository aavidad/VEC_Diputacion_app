CREATE TABLE vec_contratacion_temporal.candidatura_alta_tecnica (
    ambito_raiz_hmac text PRIMARY KEY,
    huella_raiz_hmac text NOT NULL UNIQUE,
    reserva_ref text NOT NULL UNIQUE,
    expediente_ref text NOT NULL UNIQUE,
    numero_visible text NOT NULL UNIQUE,
    recibo_ref text NOT NULL UNIQUE,
    organizacion_ref text NOT NULL,
    actor_ref text NOT NULL,
    perfil_ref text NOT NULL,
    instante_efecto timestamptz(6) NOT NULL,
    origen text NOT NULL,
    creada_en timestamptz(6) NOT NULL,
    CHECK (ambito_raiz_hmac ~ ('^hmac-sha256:vec[.]contratacion-temporal[.]' ||
        'ambito-idempotencia/v[1-9][0-9]{0,8}:[a-f0-9]{64}$')),
    CHECK (huella_raiz_hmac ~ ('^hmac-sha256:vec[.]contratacion-temporal[.]' ||
        'huella-peticion/v[1-9][0-9]{0,8}:[a-f0-9]{64}$')),
    CHECK (pg_catalog.right(ambito_raiz_hmac, 64) <> pg_catalog.repeat('0', 64)),
    CHECK (pg_catalog.right(huella_raiz_hmac, 64) <> pg_catalog.repeat('0', 64)),
    CHECK (substring(ambito_raiz_hmac FROM '/v([1-9][0-9]{0,8}):') =
           substring(huella_raiz_hmac FROM '/v([1-9][0-9]{0,8}):')),
    CHECK (reserva_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
    CHECK (expediente_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
    CHECK (numero_visible ~ '^[0-9]{4}/[A-Za-z0-9._-]{1,40}$'),
    CHECK (recibo_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
    CHECK (organizacion_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
    CHECK (actor_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
    CHECK (perfil_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
    CHECK (origen IN ('backfill', 'resolucion')),
    CHECK (instante_efecto = pg_catalog.date_trunc('microseconds', instante_efecto)),
    CHECK (creada_en = pg_catalog.date_trunc('microseconds', creada_en))
);

CREATE TABLE vec_contratacion_temporal.candidatura_alta_alias (
    ambito_hmac text PRIMARY KEY,
    huella_hmac text NOT NULL UNIQUE,
    ambito_raiz_hmac text NOT NULL,
    generacion integer NOT NULL,
    registrada_en timestamptz(6) NOT NULL,
    UNIQUE (ambito_raiz_hmac, generacion),
    FOREIGN KEY (ambito_raiz_hmac) REFERENCES
        vec_contratacion_temporal.candidatura_alta_tecnica(ambito_raiz_hmac),
    CHECK (generacion BETWEEN 1 AND 999999999),
    CHECK (ambito_hmac ~ ('^hmac-sha256:vec[.]contratacion-temporal[.]' ||
        'ambito-idempotencia/v[1-9][0-9]{0,8}:[a-f0-9]{64}$')),
    CHECK (huella_hmac ~ ('^hmac-sha256:vec[.]contratacion-temporal[.]' ||
        'huella-peticion/v[1-9][0-9]{0,8}:[a-f0-9]{64}$')),
    CHECK (pg_catalog.right(ambito_hmac, 64) <> pg_catalog.repeat('0', 64)),
    CHECK (pg_catalog.right(huella_hmac, 64) <> pg_catalog.repeat('0', 64)),
    CHECK (substring(ambito_hmac FROM '/v([1-9][0-9]{0,8}):')::integer = generacion),
    CHECK (substring(huella_hmac FROM '/v([1-9][0-9]{0,8}):')::integer = generacion),
    CHECK (registrada_en = pg_catalog.date_trunc('microseconds', registrada_en))
);

INSERT INTO vec_contratacion_temporal.candidatura_alta_tecnica (
    ambito_raiz_hmac, huella_raiz_hmac, reserva_ref, expediente_ref,
    numero_visible, recibo_ref, organizacion_ref, actor_ref, perfil_ref,
    instante_efecto, origen, creada_en
)
SELECT ambito_hmac, huella_peticion_hmac, reserva_ref, expediente_ref,
       numero_visible, recibo_ref, organizacion_ref, actor_ref, perfil_ref,
       creada_en, 'backfill', creada_en
  FROM vec_contratacion_temporal.identidad_reserva_alta;

INSERT INTO vec_contratacion_temporal.candidatura_alta_alias (
    ambito_hmac, huella_hmac, ambito_raiz_hmac, generacion, registrada_en
)
SELECT a.alias_hmac, h.alias_hmac, a.ambito_raiz_hmac, a.generacion,
       pg_catalog.greatest(a.registrada_en, h.registrada_en)
  FROM vec_contratacion_temporal.alias_ambito_alta a
  JOIN vec_contratacion_temporal.alias_huella_alta h
    ON h.ambito_raiz_hmac = a.ambito_raiz_hmac
   AND h.generacion = a.generacion;

DO $validar_backfill$
BEGIN
    IF EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.alias_ambito_alta a
          LEFT JOIN vec_contratacion_temporal.alias_huella_alta h
            ON h.ambito_raiz_hmac = a.ambito_raiz_hmac
           AND h.generacion = a.generacion
         WHERE h.ambito_raiz_hmac IS NULL
    ) OR EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.alias_huella_alta h
          LEFT JOIN vec_contratacion_temporal.alias_ambito_alta a
            ON a.ambito_raiz_hmac = h.ambito_raiz_hmac
           AND a.generacion = h.generacion
         WHERE a.ambito_raiz_hmac IS NULL
    ) OR EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.candidatura_alta_tecnica c
         WHERE c.origen = 'backfill'
           AND NOT EXISTS (
               SELECT 1
                 FROM vec_contratacion_temporal.candidatura_alta_alias a
                WHERE a.ambito_raiz_hmac = c.ambito_raiz_hmac
                  AND a.ambito_hmac = c.ambito_raiz_hmac
                  AND a.huella_hmac = c.huella_raiz_hmac
           )
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'historia HMAC incompatible con el backfill';
    END IF;
END
$validar_backfill$;

CREATE TRIGGER candidatura_alta_tecnica_inmutable
BEFORE UPDATE OR DELETE ON vec_contratacion_temporal.candidatura_alta_tecnica
FOR EACH ROW EXECUTE FUNCTION
    vec_contratacion_temporal.rechazar_mutacion_historia_v1();
CREATE TRIGGER candidatura_alta_alias_inmutable
BEFORE UPDATE OR DELETE ON vec_contratacion_temporal.candidatura_alta_alias
FOR EACH ROW EXECUTE FUNCTION
    vec_contratacion_temporal.rechazar_mutacion_historia_v1();
