\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_contratacion_temporal:migracion:000113', 0)
);

-- Petición RRHH p. 2: histórico de contratos de cada candidato en Bolsa.
-- CT no crea otra tabla ni otro outbox: publica, en solo lectura, el outbox
-- que CT75 ya escribe en la misma transacción que la incorporación. Cada
-- fila se proyecta a un evento de integración con referencias opacas; Bolsa
-- lo consume por su inbox idempotente. Solo se publica la incorporación de
-- expedientes cubiertos por un llamamiento de Bolsa (propuesta CT61/CT65).
-- No existe hoy un hecho de cese en CT (CT87 cierra sin cese), así que no se
-- inventa un evento de fin: fin_previsto es el periodo confirmado a Personal.
DO $prevalidacion$
BEGIN
    IF current_user <> 'vec_contratacion_temporal_propietario'
       OR pg_catalog.to_regclass('vec_contratacion_temporal.incorporacion_outbox_v2') IS NULL
       OR pg_catalog.to_regclass('vec_contratacion_temporal.incorporacion_registro_v2') IS NULL
       OR pg_catalog.to_regclass('vec_contratacion_temporal.propuesta_formalizacion') IS NULL
       OR pg_catalog.to_regclass('vec_contratacion_temporal.expediente_version_integral') IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.leer_contratos_bolsa_v1(timestamptz,text,integer)'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para publicar contratos a Bolsa';
    END IF;
END
$prevalidacion$;

-- Instante canónico común a CT y Bolsa: UTC con microsegundos fijos.
CREATE FUNCTION vec_contratacion_temporal.instante_contrato_bolsa_v1(p timestamptz)
RETURNS text LANGUAGE sql IMMUTABLE PARALLEL SAFE SET search_path = pg_catalog
BEGIN ATOMIC
    SELECT pg_catalog.to_char(p AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"');
END;

-- Lectura paginada por (creada_en, outbox_ref). El cursor lo guarda Bolsa en
-- su propio inbox; la reentrega de un mismo evento produce el mismo contenido
-- y la misma huella, de modo que el consumidor puede releer sin duplicar.
CREATE FUNCTION vec_contratacion_temporal.leer_contratos_bolsa_v1(
    p_desde_en timestamptz,
    p_desde_ref text,
    p_limite integer
) RETURNS TABLE(
    evento_ref text,
    evento jsonb,
    huella_sha256 text,
    origen_ref text,
    origen_creada_en timestamptz
)
LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path = pg_catalog SET timezone = 'UTC' AS $f$
BEGIN
    IF p_limite IS NULL OR p_limite NOT BETWEEN 1 AND 100
       OR (p_desde_en IS NULL) <> (p_desde_ref IS NULL)
       OR (p_desde_en IS NOT NULL AND NOT pg_catalog.isfinite(p_desde_en))
       OR pg_catalog.octet_length(p_desde_ref) > 512 THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'lectura de contratos para Bolsa inválida';
    END IF;
    RETURN QUERY
    WITH base AS (
        SELECT o.outbox_ref, o.creada_en,
               'evento:ct:contrato-bolsa:' || pg_catalog.encode(pg_catalog.sha256(
                   pg_catalog.convert_to('incorporacion' || pg_catalog.chr(31) || o.outbox_ref, 'UTF8')
               ), 'hex') AS ref,
               pg_catalog.jsonb_build_object(
                   'esquema', 'vec.contratacion-temporal.contrato-bolsa.v1',
                   'tipo', 'incorporacion',
                   'origen_ref', o.outbox_ref,
                   'organizacion_ref', r.organizacion_ref,
                   'expediente_ref', r.expediente_ref,
                   'llamamiento_ref', p.llamamiento_ref,
                   'inicio', vec_contratacion_temporal.instante_contrato_bolsa_v1(
                       (r.material_json #>> '{Confirmacion,PeriodoIncorporacion,desde}')::timestamptz),
                   'fin_previsto', vec_contratacion_temporal.instante_contrato_bolsa_v1(
                       (r.material_json #>> '{Confirmacion,PeriodoIncorporacion,hasta}')::timestamptz),
                   'modalidad_clave', e.agregado_json #>> '{analisis,modalidad_clave}',
                   'categoria_ref', e.agregado_json #>> '{analisis,categoria_ref}',
                   'causa_clave', e.agregado_json #>> '{analisis,causa_clave}',
                   'ocurrido_en', vec_contratacion_temporal.instante_contrato_bolsa_v1(r.registrada_en)
               ) AS cuerpo
          FROM vec_contratacion_temporal.incorporacion_outbox_v2 o
          JOIN vec_contratacion_temporal.incorporacion_registro_v2 r
            ON r.recibo_ref = o.recibo_ref AND r.outbox_ref = o.outbox_ref
          JOIN vec_contratacion_temporal.propuesta_formalizacion p
            ON p.organizacion_ref = r.organizacion_ref AND p.expediente_ref = r.expediente_ref
          JOIN vec_contratacion_temporal.expediente_version_integral e
            ON e.expediente_ref = r.expediente_ref AND e.version = r.version_expediente
         WHERE p_desde_en IS NULL OR (o.creada_en, o.outbox_ref) > (p_desde_en, p_desde_ref)
         ORDER BY o.creada_en, o.outbox_ref
         LIMIT p_limite
    )
    SELECT b.ref, b.cuerpo || pg_catalog.jsonb_build_object('evento_ref', b.ref),
           pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
               (b.cuerpo || pg_catalog.jsonb_build_object('evento_ref', b.ref))::text, 'UTF8')), 'hex'),
           b.outbox_ref, b.creada_en
      FROM base b
     ORDER BY b.creada_en, b.outbox_ref;
END
$f$;

REVOKE ALL ON FUNCTION vec_contratacion_temporal.instante_contrato_bolsa_v1(timestamptz) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.leer_contratos_bolsa_v1(timestamptz, text, integer) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.leer_contratos_bolsa_v1(timestamptz, text, integer)
    TO vec_contratacion_temporal_ejecutor;
COMMENT ON FUNCTION vec_contratacion_temporal.leer_contratos_bolsa_v1(timestamptz, text, integer) IS
    'CT113: publica a Bolsa, desde el outbox CT75, las incorporaciones de expedientes cubiertos por llamamiento; solo referencias opacas, fechas y claves.';
COMMIT;
