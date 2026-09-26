\set ON_ERROR_STOP on
-- CT-000126: parámetros gobernados de la numeración visible de los
-- expedientes (duda 15). El prefijo y los dígitos del número dejan de estar
-- fijos en CT-000103: la aplicación los publica al arrancar desde el catálogo
-- de reglas (regla c16.numeracion) en una tabla de solo adición, y la función
-- que reserva cada número lee la versión vigente. Sin ninguna versión rige el
-- formato de siempre, «AAAA/CT-NNNNNN». El año delante y el contador anual no
-- cambian (las tablas de expedientes exigen «AAAA/...»); los números ya
-- asignados tampoco. La publicación solo añade una versión si los parámetros
-- difieren de los vigentes: arrancar dos veces no añade historia.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '60s';
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_contratacion_temporal:migracion:000126', 0)
);

DO $prevalidacion$
BEGIN
    IF pg_catalog.to_regclass(
        'vec_contratacion_temporal.numeracion_parametros'
    ) IS NOT NULL
    OR pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.publicar_numeracion_parametros_v1(text,integer,text)'
    ) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'CT-000126 ya instalada';
    END IF;
    IF pg_catalog.to_regclass(
        'vec_contratacion_temporal.numeracion_expedientes'
    ) IS NULL
    OR pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.siguiente_numero_visible_v1(integer)'
    ) IS NULL
    OR pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.rechazar_mutacion_historia_v1()'
    ) IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para CT-000126: falta CT-000103';
    END IF;
END
$prevalidacion$;

CREATE TABLE vec_contratacion_temporal.numeracion_parametros (
    version bigint PRIMARY KEY,
    prefijo text NOT NULL,
    digitos smallint NOT NULL,
    huella_sha256 text NOT NULL,
    fuente_ref text NOT NULL,
    publicado_por text NOT NULL,
    publicado_en timestamptz(6) NOT NULL,
    CHECK (version BETWEEN 1 AND 9007199254740991),
    -- El prefijo no termina en cifra: si terminara, «CT-1» + «5» y «CT-» +
    -- «15» darían el mismo número visible.
    CHECK (prefijo ~ '^([A-Za-z0-9._-]{0,19}[A-Za-z._-])?$'),
    CHECK (digitos BETWEEN 1 AND 9),
    CHECK (huella_sha256 = pg_catalog.encode(pg_catalog.sha256(
        pg_catalog.convert_to(prefijo || '|' || digitos::text, 'UTF8')
    ), 'hex')),
    CHECK (fuente_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
    CHECK (pg_catalog.length(publicado_por) BETWEEN 1 AND 63),
    CHECK (publicado_en = pg_catalog.date_trunc('microseconds', publicado_en))
);

CREATE TRIGGER numeracion_parametros_inmutable
BEFORE UPDATE OR DELETE
ON vec_contratacion_temporal.numeracion_parametros
FOR EACH ROW
EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();

CREATE TRIGGER numeracion_parametros_no_truncar
BEFORE TRUNCATE
ON vec_contratacion_temporal.numeracion_parametros
FOR EACH STATEMENT
EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();

ALTER TABLE vec_contratacion_temporal.numeracion_parametros
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contratacion_temporal.numeracion_parametros
    FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_total
    ON vec_contratacion_temporal.numeracion_parametros
    TO vec_contratacion_temporal_propietario
    USING (true) WITH CHECK (true);
REVOKE ALL ON TABLE vec_contratacion_temporal.numeracion_parametros
FROM PUBLIC,
    vec_contratacion_temporal_migrador,
    vec_contratacion_temporal_ejecutor,
    vec_contratacion_temporal_gobernador,
    vec_contratacion_temporal_confirmador_cobertura,
    vec_contratacion_temporal_lector_resultado_cobertura,
    vec_contratacion_temporal_consultor_rrhh;

-- publicar_numeracion_parametros_v1 la llama el gobernador al arrancar. Con
-- los mismos parámetros que los vigentes (o, sin ninguna versión, que los de
-- siempre) devuelve «vigente» sin escribir; si difieren, añade la versión
-- siguiente, contigua, y devuelve «publicada».
CREATE FUNCTION vec_contratacion_temporal.publicar_numeracion_parametros_v1(
    p_prefijo text,
    p_digitos integer,
    p_fuente_ref text
)
RETURNS TABLE (resultado text, version bigint)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
SET row_security = 'on'
SET timezone = 'UTC'
SET lock_timeout = '2s'
AS $funcion$
DECLARE
    v_actual record;
    v_hay boolean;
BEGIN
    IF session_user = current_user
       OR NOT pg_catalog.pg_has_role(
           session_user, 'vec_contratacion_temporal_gobernador', 'MEMBER'
       )
       OR pg_catalog.pg_has_role(
           session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER'
       )
       OR pg_catalog.pg_has_role(
           session_user, 'vec_contratacion_temporal_migrador', 'MEMBER'
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'publicación de numeración no autorizada';
    END IF;
    IF p_prefijo IS NULL OR p_prefijo !~ '^([A-Za-z0-9._-]{0,19}[A-Za-z._-])?$'
       OR p_digitos IS NULL OR p_digitos NOT BETWEEN 1 AND 9
       OR p_fuente_ref IS NULL
       OR p_fuente_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'parámetros de numeración no válidos';
    END IF;
    PERFORM pg_catalog.pg_advisory_xact_lock(
        pg_catalog.hashtextextended(
            'vec_contratacion_temporal:numeracion_parametros', 0
        )
    );
    SELECT n.version, n.prefijo, n.digitos
      INTO v_actual
      FROM vec_contratacion_temporal.numeracion_parametros n
     ORDER BY n.version DESC
     LIMIT 1;
    v_hay := FOUND;
    IF (v_hay AND v_actual.prefijo = p_prefijo AND v_actual.digitos = p_digitos)
       OR (NOT v_hay AND p_prefijo = 'CT-' AND p_digitos = 6) THEN
        RETURN QUERY SELECT 'vigente'::text,
            CASE WHEN v_hay THEN v_actual.version ELSE 0::bigint END;
        RETURN;
    END IF;
    RETURN QUERY
    INSERT INTO vec_contratacion_temporal.numeracion_parametros AS n (
        version, prefijo, digitos, huella_sha256, fuente_ref,
        publicado_por, publicado_en
    ) VALUES (
        CASE WHEN v_hay THEN v_actual.version + 1 ELSE 1 END,
        p_prefijo, p_digitos::smallint,
        pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
            p_prefijo || '|' || p_digitos::text, 'UTF8'
        )), 'hex'),
        p_fuente_ref, session_user::text,
        pg_catalog.date_trunc('microseconds', pg_catalog.clock_timestamp())
    )
    RETURNING 'publicada'::text, n.version;
END
$funcion$;

-- La reserva de números conserva su firma, su contador anual y su límite; el
-- prefijo y el relleno salen de la versión vigente de los parámetros. Si el
-- número tiene más cifras que los dígitos pedidos, se escribe entero: nunca se
-- recorta.
CREATE OR REPLACE FUNCTION vec_contratacion_temporal.siguiente_numero_visible_v1(p_anio integer)
RETURNS text LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path = pg_catalog AS $f$
DECLARE
 v_ultimo integer;
 v_prefijo text := 'CT-';
 v_digitos integer := 6;
BEGIN
 IF p_anio NOT BETWEEN 1 AND 9999 THEN RAISE EXCEPTION USING ERRCODE='22023'; END IF;
 SELECT n.prefijo, n.digitos INTO v_prefijo, v_digitos
   FROM vec_contratacion_temporal.numeracion_parametros n
  ORDER BY n.version DESC LIMIT 1;
 IF NOT FOUND THEN v_prefijo := 'CT-'; v_digitos := 6; END IF;
 INSERT INTO vec_contratacion_temporal.numeracion_expedientes(anio,ultimo) VALUES(p_anio,0)
 ON CONFLICT (anio) DO NOTHING;
 UPDATE vec_contratacion_temporal.numeracion_expedientes SET ultimo=ultimo+1
 WHERE anio=p_anio RETURNING ultimo INTO v_ultimo;
 IF v_ultimo IS NULL OR v_ultimo > 999999 THEN RAISE EXCEPTION USING ERRCODE='22003'; END IF;
 RETURN format('%s/%s%s', lpad(p_anio::text,4,'0'), v_prefijo,
   CASE WHEN length(v_ultimo::text) >= v_digitos THEN v_ultimo::text
        ELSE lpad(v_ultimo::text, v_digitos, '0') END);
END $f$;

ALTER FUNCTION vec_contratacion_temporal.publicar_numeracion_parametros_v1(text, integer, text)
    OWNER TO vec_contratacion_temporal_propietario;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.publicar_numeracion_parametros_v1(text, integer, text)
FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.publicar_numeracion_parametros_v1(text, integer, text)
TO vec_contratacion_temporal_gobernador;
COMMIT;
