-- CT-000042: códec UTF-8 privado y realmente inmutable.
--
-- PostgreSQL 18 marca convert_to/convert_from/textsend como STABLE. Estos
-- auxiliares no los usan: recorren puntos de código y octetos con primitivas
-- IMMUTABLE. La migración acredita antes que server_encoding es UTF8.
CREATE FUNCTION vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
    p_texto text
)
RETURNS bytea
LANGUAGE plpgsql
IMMUTABLE
STRICT
PARALLEL SAFE
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_resultado bytea := ''::bytea;
    v_indice integer;
    v_codigo integer;
    v_octetos bytea;
BEGIN
    FOR v_indice IN 1..pg_catalog.char_length(p_texto) LOOP
        v_codigo := pg_catalog.ascii(
            substring(p_texto FROM v_indice FOR 1)
        );
        IF v_codigo <= 127 THEN
            v_octetos := '\x00'::bytea;
            v_octetos := pg_catalog.set_byte(v_octetos, 0, v_codigo);
        ELSIF v_codigo <= 2047 THEN
            v_octetos := '\x0000'::bytea;
            v_octetos := pg_catalog.set_byte(
                v_octetos, 0, 192 | (v_codigo >> 6)
            );
            v_octetos := pg_catalog.set_byte(
                v_octetos, 1, 128 | (v_codigo & 63)
            );
        ELSIF v_codigo <= 65535 THEN
            IF v_codigo BETWEEN 55296 AND 57343 THEN
                RAISE EXCEPTION USING ERRCODE = '22023',
                    MESSAGE = 'texto UTF-8 RRHH inválido';
            END IF;
            v_octetos := '\x000000'::bytea;
            v_octetos := pg_catalog.set_byte(
                v_octetos, 0, 224 | (v_codigo >> 12)
            );
            v_octetos := pg_catalog.set_byte(
                v_octetos, 1, 128 | ((v_codigo >> 6) & 63)
            );
            v_octetos := pg_catalog.set_byte(
                v_octetos, 2, 128 | (v_codigo & 63)
            );
        ELSIF v_codigo <= 1114111 THEN
            v_octetos := '\x00000000'::bytea;
            v_octetos := pg_catalog.set_byte(
                v_octetos, 0, 240 | (v_codigo >> 18)
            );
            v_octetos := pg_catalog.set_byte(
                v_octetos, 1, 128 | ((v_codigo >> 12) & 63)
            );
            v_octetos := pg_catalog.set_byte(
                v_octetos, 2, 128 | ((v_codigo >> 6) & 63)
            );
            v_octetos := pg_catalog.set_byte(
                v_octetos, 3, 128 | (v_codigo & 63)
            );
        ELSE
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'texto UTF-8 RRHH inválido';
        END IF;
        v_resultado := v_resultado || v_octetos;
    END LOOP;
    RETURN v_resultado;
EXCEPTION WHEN OTHERS THEN
    RAISE EXCEPTION USING ERRCODE = '22023',
        MESSAGE = 'texto UTF-8 RRHH inválido';
END
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.decodificar_texto_utf8_rrhh_v1(
    p_octetos bytea
)
RETURNS text
LANGUAGE plpgsql
IMMUTABLE
STRICT
PARALLEL SAFE
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_resultado text := '';
    v_indice integer := 0;
    v_longitud integer := pg_catalog.octet_length(p_octetos);
    v_primero integer;
    v_segundo integer;
    v_tercero integer;
    v_cuarto integer;
    v_codigo integer;
BEGIN
    WHILE v_indice < v_longitud LOOP
        v_primero := pg_catalog.get_byte(p_octetos, v_indice);
        IF v_primero <= 127 THEN
            v_codigo := v_primero;
            v_indice := v_indice + 1;
        ELSIF v_primero BETWEEN 194 AND 223
              AND v_indice + 1 < v_longitud THEN
            v_segundo := pg_catalog.get_byte(p_octetos, v_indice + 1);
            IF v_segundo NOT BETWEEN 128 AND 191 THEN
                RAISE EXCEPTION USING ERRCODE = '22023',
                    MESSAGE = 'octetos UTF-8 RRHH inválidos';
            END IF;
            v_codigo := ((v_primero & 31) << 6) | (v_segundo & 63);
            v_indice := v_indice + 2;
        ELSIF v_primero BETWEEN 224 AND 239
              AND v_indice + 2 < v_longitud THEN
            v_segundo := pg_catalog.get_byte(p_octetos, v_indice + 1);
            v_tercero := pg_catalog.get_byte(p_octetos, v_indice + 2);
            IF v_segundo NOT BETWEEN 128 AND 191
               OR v_tercero NOT BETWEEN 128 AND 191
               OR v_primero = 224 AND v_segundo < 160
               OR v_primero = 237 AND v_segundo > 159 THEN
                RAISE EXCEPTION USING ERRCODE = '22023',
                    MESSAGE = 'octetos UTF-8 RRHH inválidos';
            END IF;
            v_codigo := ((v_primero & 15) << 12)
                | ((v_segundo & 63) << 6) | (v_tercero & 63);
            v_indice := v_indice + 3;
        ELSIF v_primero BETWEEN 240 AND 244
              AND v_indice + 3 < v_longitud THEN
            v_segundo := pg_catalog.get_byte(p_octetos, v_indice + 1);
            v_tercero := pg_catalog.get_byte(p_octetos, v_indice + 2);
            v_cuarto := pg_catalog.get_byte(p_octetos, v_indice + 3);
            IF v_segundo NOT BETWEEN 128 AND 191
               OR v_tercero NOT BETWEEN 128 AND 191
               OR v_cuarto NOT BETWEEN 128 AND 191
               OR v_primero = 240 AND v_segundo < 144
               OR v_primero = 244 AND v_segundo > 143 THEN
                RAISE EXCEPTION USING ERRCODE = '22023',
                    MESSAGE = 'octetos UTF-8 RRHH inválidos';
            END IF;
            v_codigo := ((v_primero & 7) << 18)
                | ((v_segundo & 63) << 12)
                | ((v_tercero & 63) << 6) | (v_cuarto & 63);
            v_indice := v_indice + 4;
        ELSE
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'octetos UTF-8 RRHH inválidos';
        END IF;
        IF v_codigo = 0 THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'octetos UTF-8 RRHH inválidos';
        END IF;
        v_resultado := v_resultado || pg_catalog.chr(v_codigo);
    END LOOP;
    RETURN v_resultado;
EXCEPTION WHEN OTHERS THEN
    RAISE EXCEPTION USING ERRCODE = '22023',
        MESSAGE = 'octetos UTF-8 RRHH inválidos';
END
$funcion$;
