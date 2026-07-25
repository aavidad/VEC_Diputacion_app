-- Lecturas estrechas sobre la proyección 000002. La histórica sirve a la
-- preparación; la actual mantiene una barrera hasta el COMMIT del efecto. Su
-- instalación no convierte por sí sola la proyección en autoridad: hasta que
-- el catálogo maestro externo la alimente sincrónicamente, ambas fallan cerrado.
BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:autorizacion:barrera-motivo-cobertura-v1:000003',
        0
    )
);

DO $prevalidacion$
BEGIN
    IF pg_catalog.to_regclass(
           'vec_autorizacion.motivo_cobertura_v1_checkpoint_origen'
       ) IS NULL
       OR pg_catalog.to_regclass(
           'vec_autorizacion.motivo_cobertura_v1_catalogo_publicado'
       ) IS NULL
       OR pg_catalog.to_regclass(
           'vec_autorizacion.motivo_cobertura_v1_entrada'
       ) IS NULL
       OR pg_catalog.to_regclass(
           'vec_autorizacion.motivo_cobertura_v1_retirada'
       ) IS NULL
       OR NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_roles
            WHERE rolname = 'vec_autorizacion_motivos_evaluador'
              AND NOT rolcanlogin AND NOT rolsuper AND NOT rolbypassrls
       )
       OR NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_roles
            WHERE rolname = 'vec_contratacion_temporal_propietario'
              AND NOT rolcanlogin AND NOT rolsuper AND NOT rolbypassrls
       )
       OR pg_catalog.to_regprocedure(
           'vec_autorizacion.resolver_motivo_cobertura_historico_v1(text,integer,text,text,text,timestamp with time zone)'
       ) IS NOT NULL
       OR pg_catalog.to_regprocedure(
           'vec_autorizacion.resolver_motivo_cobertura_actual_v1(text,integer,text,text,text)'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para instalar la barrera de cobertura';
    END IF;
END
$prevalidacion$;

CREATE FUNCTION vec_autorizacion.resolver_motivo_cobertura_historico_v1(
    p_catalogo_id text,
    p_catalogo_version integer,
    p_catalogo_huella_publicada_sha256 text,
    p_entrada_clave text,
    p_clave_i18n text,
    p_instante timestamptz
)
RETURNS boolean
LANGUAGE plpgsql
STABLE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $funcion$
BEGIN
    IF (p_catalogo_id ~ '^[a-z][a-z0-9._-]{0,127}$') IS NOT TRUE
       OR p_catalogo_version IS NULL OR p_catalogo_version < 1
       OR (
           p_catalogo_huella_publicada_sha256 ~ '^[0-9a-f]{64}$'
       ) IS NOT TRUE
       OR p_catalogo_huella_publicada_sha256 =
          pg_catalog.repeat('0', 64)
       OR (p_entrada_clave ~ '^[a-z][a-z0-9._-]{0,127}$') IS NOT TRUE
       OR (p_clave_i18n ~ '^[a-z][a-z0-9._-]{1,79}$') IS NOT TRUE
       OR p_instante IS NULL
       OR pg_catalog.isfinite(p_instante) IS NOT TRUE
       OR pg_catalog.date_part(
           'year', (p_instante AT TIME ZONE 'UTC')
       ) NOT BETWEEN 1 AND 9999
       OR p_instante > pg_catalog.statement_timestamp() THEN
        RETURN false;
    END IF;
    RETURN EXISTS (
        SELECT 1
          FROM vec_autorizacion.motivo_cobertura_v1_catalogo_publicado c
          JOIN vec_autorizacion.motivo_cobertura_v1_entrada e
            ON e.catalogo_id = c.catalogo_id
           AND e.catalogo_version = c.catalogo_version
         WHERE c.catalogo_id = p_catalogo_id
           AND c.catalogo_version = p_catalogo_version
           AND c.modulo_id = 'contratacion_temporal'
           AND c.catalogo_huella_publicada_sha256 =
               p_catalogo_huella_publicada_sha256
           AND c.publicado_en <= p_instante
           AND e.entrada_clave = p_entrada_clave
           AND e.clave_i18n = p_clave_i18n
           AND e.vigente_desde <= p_instante
           AND (e.vigente_hasta IS NULL OR p_instante < e.vigente_hasta)
           AND NOT EXISTS (
               SELECT 1
                 FROM vec_autorizacion.motivo_cobertura_v1_retirada r
                WHERE r.catalogo_id = c.catalogo_id
                  AND r.catalogo_version = c.catalogo_version
                  AND r.retirado_en <= p_instante
           )
    );
END
$funcion$;

CREATE FUNCTION vec_autorizacion.resolver_motivo_cobertura_actual_v1(
    p_catalogo_id text,
    p_catalogo_version integer,
    p_catalogo_huella_publicada_sha256 text,
    p_entrada_clave text,
    p_clave_i18n text
)
RETURNS boolean
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    v_ahora timestamptz(6);
BEGIN
    IF pg_catalog.current_setting('transaction_isolation') <> 'serializable'
       OR pg_catalog.current_setting('transaction_read_only') <> 'off'
       OR (p_catalogo_id ~ '^[a-z][a-z0-9._-]{0,127}$') IS NOT TRUE
       OR p_catalogo_version IS NULL OR p_catalogo_version < 1
       OR (
           p_catalogo_huella_publicada_sha256 ~ '^[0-9a-f]{64}$'
       ) IS NOT TRUE
       OR p_catalogo_huella_publicada_sha256 =
          pg_catalog.repeat('0', 64)
       OR (p_entrada_clave ~ '^[a-z][a-z0-9._-]{0,127}$') IS NOT TRUE
       OR (p_clave_i18n ~ '^[a-z][a-z0-9._-]{1,79}$') IS NOT TRUE THEN
        RETURN false;
    END IF;
    PERFORM ultima_secuencia
      FROM vec_autorizacion.motivo_cobertura_v1_checkpoint_origen
     WHERE control_id
     FOR SHARE;
    IF NOT FOUND THEN RETURN false; END IF;
    v_ahora := pg_catalog.clock_timestamp();
    RETURN EXISTS (
        SELECT 1
          FROM vec_autorizacion.motivo_cobertura_v1_catalogo_publicado c
          JOIN vec_autorizacion.motivo_cobertura_v1_entrada e
            ON e.catalogo_id = c.catalogo_id
           AND e.catalogo_version = c.catalogo_version
         WHERE c.catalogo_id = p_catalogo_id
           AND c.catalogo_version = p_catalogo_version
           AND c.modulo_id = 'contratacion_temporal'
           AND c.catalogo_huella_publicada_sha256 =
               p_catalogo_huella_publicada_sha256
           AND c.publicado_en <= v_ahora
           AND e.entrada_clave = p_entrada_clave
           AND e.clave_i18n = p_clave_i18n
           AND e.vigente_desde <= v_ahora
           AND (e.vigente_hasta IS NULL OR v_ahora < e.vigente_hasta)
           AND NOT EXISTS (
               SELECT 1
                 FROM vec_autorizacion.motivo_cobertura_v1_retirada r
                WHERE r.catalogo_id = c.catalogo_id
                  AND r.catalogo_version = c.catalogo_version
           )
    );
END
$funcion$;

REVOKE ALL ON FUNCTION
    vec_autorizacion.resolver_motivo_cobertura_historico_v1(
        text, integer, text, text, text, timestamptz
    ),
    vec_autorizacion.resolver_motivo_cobertura_actual_v1(
        text, integer, text, text, text
    )
FROM PUBLIC, vec_autorizacion_motivos_proyector,
     vec_autorizacion_motivos_evaluador,
     vec_contratacion_temporal_propietario;

GRANT USAGE ON SCHEMA vec_autorizacion
    TO vec_autorizacion_motivos_evaluador,
       vec_contratacion_temporal_propietario;
GRANT EXECUTE ON FUNCTION
vec_autorizacion.resolver_motivo_cobertura_historico_v1(
    text, integer, text, text, text, timestamptz
) TO vec_autorizacion_motivos_evaluador;
GRANT EXECUTE ON FUNCTION
vec_autorizacion.resolver_motivo_cobertura_actual_v1(
    text, integer, text, text, text
) TO vec_contratacion_temporal_propietario;

COMMENT ON FUNCTION
vec_autorizacion.resolver_motivo_cobertura_actual_v1(
    text, integer, text, text, text
) IS
    'Barrera privada del COMMIT de cobertura; false incluye proyección externa no inicializada o no sincronizada.';

COMMIT;
