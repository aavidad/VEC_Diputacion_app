BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:000006_candidatura_tecnica_o2_06',
        0
    )
);

DO $proteccion$
BEGIN
    IF EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.candidatura_alta_tecnica c
          LEFT JOIN vec_contratacion_temporal.identidad_reserva_alta i
            ON i.ambito_hmac = c.ambito_raiz_hmac
         WHERE i.ambito_hmac IS NULL
    )
       AND pg_catalog.current_setting(
           'vec.confirmar_destruccion_candidaturas_tecnicas',
           true
       ) IS DISTINCT FROM
       'DESTRUIR_CANDIDATURAS_TECNICAS_IRREVERSIBLE' THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'retirada de candidaturas técnicas protegida';
    END IF;
END
$proteccion$;

REVOKE EXECUTE ON FUNCTION
    vec_contratacion_temporal.resolver_candidatura_alta_tecnica_v1(
        text[], text[], text, text, text, text, text, text, text, text
    ),
    vec_contratacion_temporal.confirmar_alta_atestada_v2(
        bytea, bytea, bytea, bytea, numeric, numeric,
        bytea, bytea, bytea, bytea, bytea, bytea
    )
    FROM vec_contratacion_temporal_ejecutor;
DROP FUNCTION vec_contratacion_temporal.confirmar_alta_atestada_v2(
    bytea, bytea, bytea, bytea, numeric, numeric,
    bytea, bytea, bytea, bytea, bytea, bytea
);
DROP FUNCTION
    vec_contratacion_temporal.resolver_candidatura_alta_tecnica_v1(
        text[], text[], text, text, text, text, text, text, text, text
    );

DROP TRIGGER alias_huella_candidatura_alta_inmutable
    ON vec_contratacion_temporal.alias_huella_candidatura_alta;
DROP TRIGGER alias_ambito_candidatura_alta_inmutable
    ON vec_contratacion_temporal.alias_ambito_candidatura_alta;
DROP TRIGGER candidatura_alta_tecnica_inmutable
    ON vec_contratacion_temporal.candidatura_alta_tecnica;
DROP TABLE vec_contratacion_temporal.alias_huella_candidatura_alta;
DROP TABLE vec_contratacion_temporal.alias_ambito_candidatura_alta;
DROP TABLE vec_contratacion_temporal.candidatura_alta_tecnica;

GRANT EXECUTE ON FUNCTION
    vec_contratacion_temporal.confirmar_alta_atestada_v1(
        bytea, bytea, bytea, bytea, numeric, numeric,
        bytea, bytea, bytea, bytea, bytea, bytea
    )
    TO vec_contratacion_temporal_ejecutor;

COMMIT;
