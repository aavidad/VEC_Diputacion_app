-- Primitivas privadas del perfil de fuente corporativa de ContextoActor V1.
-- Este componente no construye canones ni publica una API ejecutable.

CREATE FUNCTION vec_autorizacion_atestada_v3.octetos_en_intervalo_validos(
    p_valor pg_catalog.bytea,
    p_minimo pg_catalog.int4,
    p_maximo pg_catalog.int4
)
RETURNS pg_catalog.bool
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog
AS $funcion$
    SELECT p_valor IS NOT NULL
       AND p_minimo IS NOT NULL
       AND p_maximo IS NOT NULL
       AND p_minimo BETWEEN 1 AND 262144
       AND p_maximo BETWEEN p_minimo AND 262144
       AND pg_catalog.octet_length(p_valor) BETWEEN p_minimo AND p_maximo
$funcion$;

-- El limite se comprueba antes de convertir. Solo se rechaza el BOM inicial;
-- U+FEFF dentro de una cadena JSON sigue siendo contenido, no una marca.
CREATE FUNCTION
vec_autorizacion_atestada_v3.texto_utf8_exacto_en_intervalo_valido(
    p_valor pg_catalog.bytea,
    p_minimo pg_catalog.int4,
    p_maximo pg_catalog.int4
)
RETURNS pg_catalog.bool
LANGUAGE plpgsql
IMMUTABLE
SET search_path = pg_catalog
AS $funcion$
BEGIN
    IF vec_autorizacion_atestada_v3.octetos_en_intervalo_validos(
           p_valor, p_minimo, p_maximo
       ) IS NOT TRUE THEN
        RETURN false;
    END IF;
    IF pg_catalog.octet_length(p_valor) >= 3
       AND pg_catalog.substr(p_valor, 1, 3) =
           pg_catalog.decode('efbbbf', 'hex') THEN
        RETURN false;
    END IF;
    PERFORM pg_catalog.convert_from(p_valor, 'UTF8');
    RETURN true;
EXCEPTION
    WHEN character_not_in_repertoire OR untranslatable_character THEN
        RETURN false;
END
$funcion$;

CREATE FUNCTION
vec_autorizacion_atestada_v3.manifiesto_fuente_bytes_validos(
    p_valor pg_catalog.bytea
)
RETURNS pg_catalog.bool
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog
AS $funcion$
    SELECT vec_autorizacion_atestada_v3
        .texto_utf8_exacto_en_intervalo_valido(p_valor, 128, 16384)
$funcion$;

CREATE FUNCTION
vec_autorizacion_atestada_v3.capacidad_fuente_bytes_validos(
    p_valor pg_catalog.bytea
)
RETURNS pg_catalog.bool
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog
AS $funcion$
    SELECT vec_autorizacion_atestada_v3
        .texto_utf8_exacto_en_intervalo_valido(p_valor, 512, 32768)
$funcion$;

CREATE FUNCTION
vec_autorizacion_atestada_v3.sobre_cose_sign1_fuente_bytes_validos(
    p_valor pg_catalog.bytea
)
RETURNS pg_catalog.bool
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog
AS $funcion$
    SELECT vec_autorizacion_atestada_v3
        .octetos_en_intervalo_validos(p_valor, 128, 65536)
$funcion$;

CREATE FUNCTION
vec_autorizacion_atestada_v3.evidencia_verificacion_fuente_bytes_validos(
    p_valor pg_catalog.bytea
)
RETURNS pg_catalog.bool
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog
AS $funcion$
    SELECT vec_autorizacion_atestada_v3
        .octetos_en_intervalo_validos(p_valor, 32, 262144)
$funcion$;

CREATE FUNCTION
vec_autorizacion_atestada_v3.raiz_publica_spki_fuente_bytes_validos(
    p_valor pg_catalog.bytea
)
RETURNS pg_catalog.bool
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog
AS $funcion$
    SELECT vec_autorizacion_atestada_v3
        .octetos_en_intervalo_validos(p_valor, 44, 44)
$funcion$;

-- Perfil unico de fuente_ref, evento_fuente_ref y efecto_ref.
CREATE FUNCTION
vec_autorizacion_atestada_v3.referencia_opaca_fuente_corporativa_valida(
    p_valor pg_catalog.text
)
RETURNS pg_catalog.bool
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog
AS $funcion$
    SELECT p_valor IS NOT NULL
       AND pg_catalog.octet_length(p_valor) BETWEEN 3 AND 160
       AND (p_valor COLLATE pg_catalog."C") ~
           '^[A-Za-z0-9][A-Za-z0-9_.:/_-]{2,159}$'
$funcion$;

CREATE FUNCTION
vec_autorizacion_atestada_v3.operacion_ref_fuente_corporativa_valida(
    p_valor pg_catalog.text
)
RETURNS pg_catalog.bool
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog
AS $funcion$
    SELECT p_valor IS NOT NULL
       AND pg_catalog.octet_length(p_valor) BETWEEN 28 AND 132
       AND (p_valor COLLATE pg_catalog."C") ~
           '^oca_[A-Za-z0-9_-]{24,128}$'
$funcion$;

-- Se recibe json, no jsonb, para conservar y validar la forma lexica. Asi un
-- exponente o una fraccion no se normalizan antes de aplicar el contrato.
CREATE FUNCTION
vec_autorizacion_atestada_v3.entero_json_seguro_fuente_valido(
    p_valor pg_catalog.json
)
RETURNS pg_catalog.bool
LANGUAGE plpgsql
IMMUTABLE
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_texto pg_catalog.text;
    v_numero pg_catalog.numeric;
BEGIN
    IF pg_catalog.json_typeof(p_valor) IS DISTINCT FROM 'number' THEN
        RETURN false;
    END IF;
    v_texto := p_valor::pg_catalog.text;
    IF (v_texto COLLATE pg_catalog."C") !~ '^[1-9][0-9]{0,15}$' THEN
        RETURN false;
    END IF;
    v_numero := v_texto::pg_catalog.numeric;
    RETURN v_numero BETWEEN 1 AND 9007199254740991::pg_catalog.numeric;
EXCEPTION
    WHEN invalid_text_representation OR numeric_value_out_of_range THEN
        RETURN false;
END
$funcion$;

CREATE FUNCTION
vec_autorizacion_atestada_v3.instante_fuente_finito_valido(
    p_valor pg_catalog.timestamptz
)
RETURNS pg_catalog.bool
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog
AS $funcion$
    SELECT p_valor IS NOT NULL
       AND pg_catalog.isfinite(p_valor)
       AND pg_catalog.date_part(
               'year', pg_catalog.timezone('UTC', p_valor)
           ) BETWEEN 1 AND 9999
$funcion$;

CREATE FUNCTION
vec_autorizacion_atestada_v3.representacion_instante_utc_fuente(
    p_valor pg_catalog.timestamptz
)
RETURNS pg_catalog.text
LANGUAGE sql
IMMUTABLE
STRICT
SET search_path = pg_catalog
AS $funcion$
    SELECT CASE
        WHEN vec_autorizacion_atestada_v3
                 .instante_fuente_finito_valido(p_valor)
        THEN pg_catalog.to_char(
            pg_catalog.timezone('UTC', p_valor),
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        )
        ELSE NULL
    END
$funcion$;

CREATE FUNCTION
vec_autorizacion_atestada_v3.instante_utc_fuente_texto_valido(
    p_valor pg_catalog.text
)
RETURNS pg_catalog.bool
LANGUAGE plpgsql
IMMUTABLE
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_instante pg_catalog.timestamptz;
BEGIN
    IF p_valor IS NULL
       OR (p_valor COLLATE pg_catalog."C") !~
          '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}[.][0-9]{6}Z$' THEN
        RETURN false;
    END IF;
    v_instante := p_valor::pg_catalog.timestamptz;
    RETURN vec_autorizacion_atestada_v3
               .representacion_instante_utc_fuente(v_instante) = p_valor;
EXCEPTION
    WHEN datetime_field_overflow OR invalid_datetime_format THEN
        RETURN false;
END
$funcion$;

REVOKE ALL ON FUNCTION
    vec_autorizacion_atestada_v3.octetos_en_intervalo_validos(
        pg_catalog.bytea, pg_catalog.int4, pg_catalog.int4
    ),
    vec_autorizacion_atestada_v3.texto_utf8_exacto_en_intervalo_valido(
        pg_catalog.bytea, pg_catalog.int4, pg_catalog.int4
    ),
    vec_autorizacion_atestada_v3.manifiesto_fuente_bytes_validos(
        pg_catalog.bytea
    ),
    vec_autorizacion_atestada_v3.capacidad_fuente_bytes_validos(
        pg_catalog.bytea
    ),
    vec_autorizacion_atestada_v3.sobre_cose_sign1_fuente_bytes_validos(
        pg_catalog.bytea
    ),
    vec_autorizacion_atestada_v3.evidencia_verificacion_fuente_bytes_validos(
        pg_catalog.bytea
    ),
    vec_autorizacion_atestada_v3.raiz_publica_spki_fuente_bytes_validos(
        pg_catalog.bytea
    ),
    vec_autorizacion_atestada_v3.referencia_opaca_fuente_corporativa_valida(
        pg_catalog.text
    ),
    vec_autorizacion_atestada_v3.operacion_ref_fuente_corporativa_valida(
        pg_catalog.text
    ),
    vec_autorizacion_atestada_v3.entero_json_seguro_fuente_valido(
        pg_catalog.json
    ),
    vec_autorizacion_atestada_v3.instante_fuente_finito_valido(
        pg_catalog.timestamptz
    ),
    vec_autorizacion_atestada_v3.representacion_instante_utc_fuente(
        pg_catalog.timestamptz
    ),
    vec_autorizacion_atestada_v3.instante_utc_fuente_texto_valido(
        pg_catalog.text
    ) FROM PUBLIC;
