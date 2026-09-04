-- Lectura mínima para que el caso de uso resuelva una entrada funcional sin
-- reconstruir el catálogo maestro ni obtener acceso directo a la proyección.
BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:autorizacion:resolver-motivo-decision:000007',
        0
    )
);

DO $prevalidacion$
BEGIN
    IF pg_catalog.to_regclass(
           'vec_autorizacion.motivo_cobertura_v1_catalogo_publicado'
       ) IS NULL
       OR pg_catalog.to_regclass(
           'vec_autorizacion.motivo_cobertura_v1_entrada'
       ) IS NULL
       OR pg_catalog.to_regclass(
           'vec_autorizacion.motivo_cobertura_v1_retirada'
       ) IS NULL
       OR pg_catalog.to_regrole(
           'vec_contratacion_temporal_ejecutor'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_autorizacion.resolver_motivo_decision_cobertura_v1(text,text,timestamp with time zone)'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para resolver el motivo de decisión';
    END IF;
END
$prevalidacion$;

CREATE FUNCTION vec_autorizacion.resolver_motivo_decision_cobertura_v1(
    p_catalogo_id text,
    p_entrada_clave text,
    p_instante timestamptz
)
RETURNS TABLE (
    catalogo_version integer,
    catalogo_huella_sha256 text,
    modulo_id text,
    entrada_clave text,
    clave_i18n text
)
LANGUAGE plpgsql
STABLE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $funcion$
BEGIN
    IF pg_catalog.pg_has_role(
           session_user,
           'vec_contratacion_temporal_ejecutor',
           'MEMBER'
       ) IS NOT TRUE
       OR pg_catalog.current_setting('transaction_isolation') <>
          'serializable'
       OR (p_catalogo_id ~ '^[a-z][a-z0-9._-]{0,127}$') IS NOT TRUE
       OR (p_entrada_clave ~ '^[a-z][a-z0-9._-]{0,127}$') IS NOT TRUE
       OR p_instante IS NULL
       OR pg_catalog.isfinite(p_instante) IS NOT TRUE
       OR pg_catalog.date_part(
           'year', (p_instante AT TIME ZONE 'UTC')
       ) NOT BETWEEN 1 AND 9999
       OR p_instante > pg_catalog.statement_timestamp() THEN
        RETURN;
    END IF;

    RETURN QUERY
    SELECT c.catalogo_version,
           c.catalogo_huella_publicada_sha256,
           c.modulo_id,
           e.entrada_clave,
           e.clave_i18n
      FROM vec_autorizacion.motivo_cobertura_v1_catalogo_publicado AS c
      JOIN vec_autorizacion.motivo_cobertura_v1_entrada AS e
        ON e.catalogo_id = c.catalogo_id
       AND e.catalogo_version = c.catalogo_version
     WHERE c.catalogo_id = p_catalogo_id
       AND c.modulo_id = 'contratacion_temporal'
       AND c.publicado_en <= p_instante
       AND e.entrada_clave = p_entrada_clave
       AND e.vigente_desde <= p_instante
       AND (e.vigente_hasta IS NULL OR p_instante < e.vigente_hasta)
       AND NOT EXISTS (
           SELECT 1
             FROM vec_autorizacion.motivo_cobertura_v1_retirada AS r
            WHERE r.catalogo_id = c.catalogo_id
              AND r.catalogo_version = c.catalogo_version
              AND r.retirado_en <= p_instante
       )
     ORDER BY c.catalogo_version DESC
     LIMIT 1;
END
$funcion$;

REVOKE ALL ON FUNCTION
vec_autorizacion.resolver_motivo_decision_cobertura_v1(
    text, text, timestamptz
) FROM PUBLIC, vec_contratacion_temporal_ejecutor;

GRANT USAGE ON SCHEMA vec_autorizacion
    TO vec_contratacion_temporal_ejecutor;
GRANT EXECUTE ON FUNCTION
vec_autorizacion.resolver_motivo_decision_cobertura_v1(
    text, text, timestamptz
) TO vec_contratacion_temporal_ejecutor;

COMMENT ON FUNCTION
vec_autorizacion.resolver_motivo_decision_cobertura_v1(
    text, text, timestamptz
) IS
    'Devuelve como máximo la entrada de motivo publicada más reciente y vigente para una decisión de Contratación Temporal.';

COMMIT;
