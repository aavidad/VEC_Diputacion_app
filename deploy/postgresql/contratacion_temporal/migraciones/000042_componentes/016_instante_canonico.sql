-- CT-000042: representación UTC independiente de TimeZone/DateStyle/lc_time.
CREATE FUNCTION
vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
    p_instante timestamptz
)
RETURNS text
LANGUAGE plpgsql
IMMUTABLE
STRICT
PARALLEL SAFE
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_utc timestamp := p_instante AT TIME ZONE 'UTC';
BEGIN
    IF NOT pg_catalog.isfinite(p_instante)
       OR extract(year FROM v_utc) NOT BETWEEN 1 AND 9999 THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'instante canónico RRHH inválido';
    END IF;
    RETURN pg_catalog.lpad(
        extract(year FROM v_utc)::integer::text, 4, '0'
    ) || '-' || pg_catalog.lpad(
        extract(month FROM v_utc)::integer::text, 2, '0'
    ) || '-' || pg_catalog.lpad(
        extract(day FROM v_utc)::integer::text, 2, '0'
    ) || 'T' || pg_catalog.lpad(
        extract(hour FROM v_utc)::integer::text, 2, '0'
    ) || ':' || pg_catalog.lpad(
        extract(minute FROM v_utc)::integer::text, 2, '0'
    ) || ':' || pg_catalog.lpad(
        pg_catalog.floor(extract(second FROM v_utc))::integer::text,
        2, '0'
    ) || '.' || pg_catalog.lpad(
        pg_catalog.mod(
            extract(microseconds FROM v_utc)::bigint, 1000000
        )::text,
        6, '0'
    ) || 'Z';
END
$funcion$;
