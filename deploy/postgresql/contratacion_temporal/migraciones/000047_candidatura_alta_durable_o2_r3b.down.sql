\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_contratacion_temporal:000047:candidatura-alta:o2-r3b', 0
));
DO $prevalidacion$
DECLARE
    v_hay_historia boolean;
BEGIN
    IF pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.confirmar_alta_atestada_v2(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea)'
       ) IS NULL
       OR pg_catalog.to_regclass(
           'vec_contratacion_temporal.candidatura_alta_tecnica'
       ) IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'candidatura de alta no instalada';
    END IF;
    SELECT EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.candidatura_alta_tecnica c
         WHERE c.origen <> 'backfill'
            OR NOT EXISTS (
                SELECT 1
                  FROM vec_contratacion_temporal.identidad_reserva_alta i
                 WHERE i.ambito_hmac = c.ambito_raiz_hmac
                   AND i.reserva_ref = c.reserva_ref
                   AND i.expediente_ref = c.expediente_ref
                   AND i.numero_visible = c.numero_visible
                   AND i.recibo_ref = c.recibo_ref
                   AND i.huella_peticion_hmac = c.huella_raiz_hmac
                   AND i.organizacion_ref = c.organizacion_ref
                   AND i.actor_ref = c.actor_ref
                   AND i.perfil_ref = c.perfil_ref
                   AND i.creada_en = c.instante_efecto
            )
    ) OR EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.candidatura_alta_alias a
         WHERE NOT EXISTS (
             SELECT 1
               FROM vec_contratacion_temporal.alias_ambito_alta aa
               JOIN vec_contratacion_temporal.alias_huella_alta ah
                 ON ah.ambito_raiz_hmac = aa.ambito_raiz_hmac
                AND ah.generacion = aa.generacion
              WHERE aa.alias_hmac = a.ambito_hmac
                AND ah.alias_hmac = a.huella_hmac
                AND aa.ambito_raiz_hmac = a.ambito_raiz_hmac
                AND aa.generacion = a.generacion
         )
    ) INTO v_hay_historia;
    IF v_hay_historia AND pg_catalog.current_setting(
           'vec.confirmar_destruccion_contratacion_temporal', true
       ) IS DISTINCT FROM
       'DESTRUIR_HISTORIA_CONTRATACION_TEMPORAL_IRREVERSIBLE' THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'retirada de candidatura bloqueada por historia';
    END IF;
END
$prevalidacion$;
REVOKE ALL ON FUNCTION
    vec_contratacion_temporal.confirmar_alta_atestada_v2(
        bytea, bytea, bytea, bytea, numeric, numeric,
        bytea, bytea, bytea, bytea, bytea, bytea
    ),
    vec_contratacion_temporal.resolver_candidatura_alta_tecnica_v1(
        text[], text[], text, text, text, text, text, text, text, timestamptz
    ) FROM PUBLIC, vec_contratacion_temporal_ejecutor;
DROP FUNCTION vec_contratacion_temporal.confirmar_alta_atestada_v2(
    bytea, bytea, bytea, bytea, numeric, numeric,
    bytea, bytea, bytea, bytea, bytea, bytea
);
DROP FUNCTION vec_contratacion_temporal.resolver_candidatura_alta_tecnica_v1(
    text[], text[], text, text, text, text, text, text, text, timestamptz
);
DROP TABLE vec_contratacion_temporal.candidatura_alta_alias;
DROP TABLE vec_contratacion_temporal.candidatura_alta_tecnica;
GRANT EXECUTE ON FUNCTION
    vec_contratacion_temporal.confirmar_alta_atestada_v1(
        bytea, bytea, bytea, bytea, numeric, numeric,
        bytea, bytea, bytea, bytea, bytea, bytea
    ) TO vec_contratacion_temporal_ejecutor;
COMMIT;
