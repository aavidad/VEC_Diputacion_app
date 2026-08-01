-- Matriz adversarial de los validadores privados F0-A1.

CREATE FUNCTION
vec_autorizacion_atestada_v3.acreditar_forma_validadores_a1_prueba()
RETURNS pg_catalog.bool
LANGUAGE sql
VOLATILE
SET search_path = pg_catalog
AS $funcion$
    WITH esperadas(
        nombre, argumentos, nombres_argumentos, retorno, lenguaje, estricta
    ) AS (
        VALUES
        ('octetos_en_intervalo_validos',
         'bytea, integer, integer',
         ARRAY['p_valor', 'p_minimo', 'p_maximo']::pg_catalog.text[],
         'boolean', 'sql', false),
        ('texto_utf8_exacto_en_intervalo_valido',
         'bytea, integer, integer',
         ARRAY['p_valor', 'p_minimo', 'p_maximo']::pg_catalog.text[],
         'boolean', 'plpgsql', false),
        ('manifiesto_fuente_bytes_validos', 'bytea', ARRAY['p_valor'],
         'boolean', 'sql', false),
        ('capacidad_fuente_bytes_validos', 'bytea', ARRAY['p_valor'],
         'boolean', 'sql', false),
        ('sobre_cose_sign1_fuente_bytes_validos', 'bytea', ARRAY['p_valor'],
         'boolean', 'sql', false),
        ('evidencia_verificacion_fuente_bytes_validos', 'bytea',
         ARRAY['p_valor'], 'boolean', 'sql', false),
        ('raiz_publica_spki_fuente_bytes_validos', 'bytea', ARRAY['p_valor'],
         'boolean', 'sql', false),
        ('referencia_opaca_fuente_corporativa_valida', 'text',
         ARRAY['p_valor'], 'boolean', 'sql', false),
        ('operacion_ref_fuente_corporativa_valida', 'text', ARRAY['p_valor'],
         'boolean', 'sql', false),
        ('entero_json_seguro_fuente_valido', 'json', ARRAY['p_valor'],
         'boolean', 'plpgsql', false),
        ('instante_fuente_finito_valido', 'timestamp with time zone',
         ARRAY['p_valor'], 'boolean', 'sql', false),
        ('representacion_instante_utc_fuente', 'timestamp with time zone',
         ARRAY['p_valor'], 'text', 'sql', true),
        ('instante_utc_fuente_texto_valido', 'text', ARRAY['p_valor'],
         'boolean', 'plpgsql', false)
    ), propietario AS (
        SELECT r.oid
          FROM pg_catalog.pg_roles AS r
         WHERE r.rolname = 'vec_autorizacion_atestada_v3_propietario'
           AND NOT r.rolcanlogin
           AND NOT r.rolsuper
           AND NOT r.rolcreatedb
           AND NOT r.rolcreaterole
           AND NOT r.rolreplication
           AND NOT r.rolbypassrls
    ), forma_exacta AS (
        SELECT p.oid
          FROM esperadas AS e
          JOIN pg_catalog.pg_namespace AS n
            ON n.nspname = 'vec_autorizacion_atestada_v3'
          JOIN pg_catalog.pg_proc AS p
            ON p.pronamespace = n.oid
           AND p.proname = e.nombre
           AND pg_catalog.oidvectortypes(p.proargtypes) = e.argumentos
          JOIN pg_catalog.pg_language AS l ON l.oid = p.prolang
          JOIN propietario AS o ON o.oid = p.proowner
         WHERE p.prokind = 'f'
           AND pg_catalog.format_type(p.prorettype, NULL) = e.retorno
           AND p.proargnames = e.nombres_argumentos
           AND p.proallargtypes IS NULL
           AND p.proargmodes IS NULL
           AND p.pronargs = pg_catalog.cardinality(e.nombres_argumentos)
           AND p.pronargdefaults = 0
           AND p.proargdefaults IS NULL
           AND p.provariadic = 0
           AND NOT p.proretset
           AND NOT p.proleakproof
           AND p.proparallel = 'u'
           AND p.provolatile = 'i'
           AND NOT p.prosecdef
           AND p.proconfig = ARRAY['search_path=pg_catalog']
           AND p.proisstrict = e.estricta
           AND l.lanname = e.lenguaje
           AND (
               SELECT pg_catalog.count(*)
                 FROM pg_catalog.aclexplode(
                     COALESCE(
                         p.proacl, pg_catalog.acldefault('f', p.proowner)
                     )
                 ) AS a
           ) = 1
           AND EXISTS (
               SELECT 1
                 FROM pg_catalog.aclexplode(
                     COALESCE(
                         p.proacl, pg_catalog.acldefault('f', p.proowner)
                     )
                 ) AS a
                WHERE a.grantor = o.oid
                  AND a.grantee = o.oid
                  AND a.privilege_type = 'EXECUTE'
                  AND NOT a.is_grantable
           )
    )
    SELECT (SELECT pg_catalog.count(*) FROM esperadas) = 13
       AND (SELECT pg_catalog.count(*) FROM propietario) = 1
       AND (SELECT pg_catalog.count(*) FROM forma_exacta) = 13
       AND (
           SELECT pg_catalog.count(*)
             FROM pg_catalog.pg_proc AS p
             JOIN pg_catalog.pg_namespace AS n ON n.oid = p.pronamespace
            WHERE n.nspname = 'vec_autorizacion_atestada_v3'
              AND p.proname IN (SELECT e.nombre FROM esperadas AS e)
       ) = 13
$funcion$;

REVOKE ALL ON FUNCTION
    vec_autorizacion_atestada_v3.acreditar_forma_validadores_a1_prueba()
    FROM PUBLIC;

DO $catalogo_inicial$
BEGIN
    IF vec_autorizacion_atestada_v3
           .acreditar_forma_validadores_a1_prueba() IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = 'XX000',
            MESSAGE = 'A1: forma inicial de validadores inválida';
    END IF;
END
$catalogo_inicial$;

GRANT EXECUTE ON FUNCTION
    vec_autorizacion_atestada_v3.octetos_en_intervalo_validos(
        pg_catalog.bytea, pg_catalog.int4, pg_catalog.int4
    ) TO vec_autorizacion_atestada_v3_migrador;

DO $acl_hostil$
BEGIN
    IF vec_autorizacion_atestada_v3
           .acreditar_forma_validadores_a1_prueba() IS NOT FALSE THEN
        RAISE EXCEPTION USING ERRCODE = 'XX000',
            MESSAGE = 'A1: ACL adicional no detectada';
    END IF;
END
$acl_hostil$;

REVOKE ALL ON FUNCTION
    vec_autorizacion_atestada_v3.octetos_en_intervalo_validos(
        pg_catalog.bytea, pg_catalog.int4, pg_catalog.int4
    ) FROM vec_autorizacion_atestada_v3_migrador;

DO $acl_restaurada$
BEGIN
    IF vec_autorizacion_atestada_v3
           .acreditar_forma_validadores_a1_prueba() IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = 'XX000',
            MESSAGE = 'A1: ACL exacta no restaurada';
    END IF;
END
$acl_restaurada$;

CREATE FUNCTION
vec_autorizacion_atestada_v3.octetos_en_intervalo_validos(
    p_valor pg_catalog.text
)
RETURNS pg_catalog.bool
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog
AS $sobrecarga$
    SELECT false
$sobrecarga$;

REVOKE ALL ON FUNCTION
    vec_autorizacion_atestada_v3.octetos_en_intervalo_validos(
        pg_catalog.text
    ) FROM PUBLIC;

DO $sobrecarga_hostil$
BEGIN
    IF vec_autorizacion_atestada_v3
           .acreditar_forma_validadores_a1_prueba() IS NOT FALSE THEN
        RAISE EXCEPTION USING ERRCODE = 'XX000',
            MESSAGE = 'A1: sobrecarga homónima no detectada';
    END IF;
END
$sobrecarga_hostil$;

DROP FUNCTION vec_autorizacion_atestada_v3.octetos_en_intervalo_validos(
    pg_catalog.text
);

DO $sobrecarga_retirada$
BEGIN
    IF vec_autorizacion_atestada_v3
           .acreditar_forma_validadores_a1_prueba() IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = 'XX000',
            MESSAGE = 'A1: inventario nominal no restaurado';
    END IF;
END
$sobrecarga_retirada$;

ALTER FUNCTION vec_autorizacion_atestada_v3.octetos_en_intervalo_validos(
    pg_catalog.bytea, pg_catalog.int4, pg_catalog.int4
) PARALLEL SAFE;

DO $paralelismo_hostil$
BEGIN
    IF vec_autorizacion_atestada_v3
           .acreditar_forma_validadores_a1_prueba() IS NOT FALSE THEN
        RAISE EXCEPTION USING ERRCODE = 'XX000',
            MESSAGE = 'A1: PARALLEL SAFE no detectado';
    END IF;
END
$paralelismo_hostil$;

ALTER FUNCTION vec_autorizacion_atestada_v3.octetos_en_intervalo_validos(
    pg_catalog.bytea, pg_catalog.int4, pg_catalog.int4
) PARALLEL UNSAFE;

DO $paralelismo_restaurado$
BEGIN
    IF vec_autorizacion_atestada_v3
           .acreditar_forma_validadores_a1_prueba() IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = 'XX000',
            MESSAGE = 'A1: paralelismo inseguro no restaurado';
    END IF;
END
$paralelismo_restaurado$;

DROP FUNCTION
    vec_autorizacion_atestada_v3.acreditar_forma_validadores_a1_prueba();

DO $utf8_y_limites$
DECLARE
    v_ascii pg_catalog.bytea;
    v_invalido pg_catalog.bytea := pg_catalog.decode('c328', 'hex');
BEGIN
    IF vec_autorizacion_atestada_v3
           .texto_utf8_exacto_en_intervalo_valido(NULL, 1, 1) IS NOT FALSE
       OR vec_autorizacion_atestada_v3
           .texto_utf8_exacto_en_intervalo_valido(''::pg_catalog.bytea, 1, 1)
          IS NOT FALSE
       OR vec_autorizacion_atestada_v3
           .texto_utf8_exacto_en_intervalo_valido(
               pg_catalog.convert_to('á', 'UTF8'), 2, 2
           ) IS NOT TRUE
       OR vec_autorizacion_atestada_v3
           .texto_utf8_exacto_en_intervalo_valido(v_invalido, 1, 8)
          IS NOT FALSE
       OR vec_autorizacion_atestada_v3
           .texto_utf8_exacto_en_intervalo_valido(
               pg_catalog.decode('c0af', 'hex'), 1, 8
           ) IS NOT FALSE
       OR vec_autorizacion_atestada_v3
           .texto_utf8_exacto_en_intervalo_valido(
               pg_catalog.decode('eda080', 'hex'), 1, 8
           ) IS NOT FALSE
       OR vec_autorizacion_atestada_v3
           .texto_utf8_exacto_en_intervalo_valido(
               pg_catalog.decode('00', 'hex'), 1, 8
           ) IS NOT FALSE
       OR vec_autorizacion_atestada_v3
           .texto_utf8_exacto_en_intervalo_valido(
               pg_catalog.decode('efbbbf7b7d', 'hex'), 1, 8
           ) IS NOT FALSE THEN
        RAISE EXCEPTION USING ERRCODE = 'XX000',
            MESSAGE = 'A1: UTF-8 estricto incompleto';
    END IF;

    -- Un exceso con bytes inválidos debe cortarse antes de convertirlos.
    IF vec_autorizacion_atestada_v3
           .texto_utf8_exacto_en_intervalo_valido(
               pg_catalog.convert_to(pg_catalog.repeat('a', 9), 'UTF8')
               || v_invalido, 1, 8
           ) IS NOT FALSE
       OR vec_autorizacion_atestada_v3.octetos_en_intervalo_validos(
              pg_catalog.decode('00', 'hex'), 0, 1
          ) IS NOT FALSE
       OR vec_autorizacion_atestada_v3.octetos_en_intervalo_validos(
              pg_catalog.decode('00', 'hex'), NULL, 1
          ) IS NOT FALSE
       OR vec_autorizacion_atestada_v3.octetos_en_intervalo_validos(
              pg_catalog.decode('00', 'hex'), 1, NULL
          ) IS NOT FALSE
       OR vec_autorizacion_atestada_v3.octetos_en_intervalo_validos(
              pg_catalog.decode('00', 'hex'), 2, 1
          ) IS NOT FALSE
       OR vec_autorizacion_atestada_v3.octetos_en_intervalo_validos(
              pg_catalog.decode('00', 'hex'), 1, 262145
          ) IS NOT FALSE THEN
        RAISE EXCEPTION USING ERRCODE = 'XX000',
            MESSAGE = 'A1: intervalo binario inseguro';
    END IF;

    v_ascii := pg_catalog.convert_to(pg_catalog.repeat('m', 128), 'UTF8');
    IF vec_autorizacion_atestada_v3.manifiesto_fuente_bytes_validos(v_ascii)
          IS NOT TRUE
       OR vec_autorizacion_atestada_v3.manifiesto_fuente_bytes_validos(
              pg_catalog.convert_to(pg_catalog.repeat('m', 127), 'UTF8')
          ) IS NOT FALSE
       OR vec_autorizacion_atestada_v3.manifiesto_fuente_bytes_validos(
              pg_catalog.convert_to(pg_catalog.repeat('m', 16384), 'UTF8')
          ) IS NOT TRUE
       OR vec_autorizacion_atestada_v3.manifiesto_fuente_bytes_validos(
              pg_catalog.convert_to(pg_catalog.repeat('m', 16385), 'UTF8')
          ) IS NOT FALSE
       OR vec_autorizacion_atestada_v3.capacidad_fuente_bytes_validos(
              pg_catalog.convert_to(pg_catalog.repeat('c', 512), 'UTF8')
          ) IS NOT TRUE
       OR vec_autorizacion_atestada_v3.capacidad_fuente_bytes_validos(
              pg_catalog.convert_to(pg_catalog.repeat('c', 511), 'UTF8')
          ) IS NOT FALSE
       OR vec_autorizacion_atestada_v3.capacidad_fuente_bytes_validos(
              pg_catalog.convert_to(pg_catalog.repeat('c', 32768), 'UTF8')
          ) IS NOT TRUE
       OR vec_autorizacion_atestada_v3.capacidad_fuente_bytes_validos(
              pg_catalog.convert_to(pg_catalog.repeat('c', 32769), 'UTF8')
          ) IS NOT FALSE THEN
        RAISE EXCEPTION USING ERRCODE = 'XX000',
            MESSAGE = 'A1: límites de manifiesto o capacidad inválidos';
    END IF;

    IF vec_autorizacion_atestada_v3.sobre_cose_sign1_fuente_bytes_validos(
           pg_catalog.convert_to(pg_catalog.repeat('s', 128), 'UTF8')
       ) IS NOT TRUE
       OR vec_autorizacion_atestada_v3.sobre_cose_sign1_fuente_bytes_validos(
              pg_catalog.convert_to(pg_catalog.repeat('s', 127), 'UTF8')
          ) IS NOT FALSE
       OR vec_autorizacion_atestada_v3.sobre_cose_sign1_fuente_bytes_validos(
              pg_catalog.convert_to(pg_catalog.repeat('s', 65536), 'UTF8')
          ) IS NOT TRUE
       OR vec_autorizacion_atestada_v3.sobre_cose_sign1_fuente_bytes_validos(
              pg_catalog.convert_to(pg_catalog.repeat('s', 65537), 'UTF8')
          ) IS NOT FALSE
       OR vec_autorizacion_atestada_v3
           .evidencia_verificacion_fuente_bytes_validos(
               pg_catalog.convert_to(pg_catalog.repeat('e', 32), 'UTF8')
           ) IS NOT TRUE
       OR vec_autorizacion_atestada_v3
           .evidencia_verificacion_fuente_bytes_validos(
               pg_catalog.convert_to(pg_catalog.repeat('e', 31), 'UTF8')
           ) IS NOT FALSE
       OR vec_autorizacion_atestada_v3
           .evidencia_verificacion_fuente_bytes_validos(
               pg_catalog.convert_to(pg_catalog.repeat('e', 262144), 'UTF8')
           ) IS NOT TRUE
       OR vec_autorizacion_atestada_v3
           .evidencia_verificacion_fuente_bytes_validos(
               pg_catalog.convert_to(pg_catalog.repeat('e', 262145), 'UTF8')
           ) IS NOT FALSE
       OR vec_autorizacion_atestada_v3.raiz_publica_spki_fuente_bytes_validos(
              pg_catalog.convert_to(pg_catalog.repeat('r', 44), 'UTF8')
          ) IS NOT TRUE
       OR vec_autorizacion_atestada_v3.raiz_publica_spki_fuente_bytes_validos(
              pg_catalog.convert_to(pg_catalog.repeat('r', 43), 'UTF8')
          ) IS NOT FALSE
       OR vec_autorizacion_atestada_v3.raiz_publica_spki_fuente_bytes_validos(
              pg_catalog.convert_to(pg_catalog.repeat('r', 45), 'UTF8')
          ) IS NOT FALSE THEN
        RAISE EXCEPTION USING ERRCODE = 'XX000',
            MESSAGE = 'A1: límites binarios nominales inválidos';
    END IF;
END
$utf8_y_limites$;

DO $textos_y_referencias$
BEGIN
    -- La primitiva V3 existente conserva la autoridad del texto técnico.
    IF vec_autorizacion_atestada_v3.texto_tecnico_valido(NULL, 160)
          IS NOT FALSE
       OR vec_autorizacion_atestada_v3.texto_tecnico_valido('', 160)
          IS NOT FALSE
       OR vec_autorizacion_atestada_v3.texto_tecnico_valido('Exacto.v1', 10)
          IS NOT TRUE
       OR vec_autorizacion_atestada_v3.texto_tecnico_valido(' Exacto', 160)
          IS NOT FALSE
       OR vec_autorizacion_atestada_v3.texto_tecnico_valido(E'Exacto\n', 160)
          IS NOT FALSE
       OR vec_autorizacion_atestada_v3.texto_tecnico_valido('Exacto*', 160)
          IS NOT FALSE
       OR vec_autorizacion_atestada_v3.texto_tecnico_valido('Ｅxacto', 160)
          IS NOT FALSE THEN
        RAISE EXCEPTION USING ERRCODE = 'XX000',
            MESSAGE = 'A1: reutilización de texto técnico V3 inválida';
    END IF;

    IF vec_autorizacion_atestada_v3
           .referencia_opaca_fuente_corporativa_valida('a._') IS NOT TRUE
       OR vec_autorizacion_atestada_v3
           .referencia_opaca_fuente_corporativa_valida(
               'f' || pg_catalog.repeat('-', 159)
           ) IS NOT TRUE
       OR vec_autorizacion_atestada_v3
           .referencia_opaca_fuente_corporativa_valida(NULL) IS NOT FALSE
       OR vec_autorizacion_atestada_v3
           .referencia_opaca_fuente_corporativa_valida('') IS NOT FALSE
       OR vec_autorizacion_atestada_v3
           .referencia_opaca_fuente_corporativa_valida('ab') IS NOT FALSE
       OR vec_autorizacion_atestada_v3
           .referencia_opaca_fuente_corporativa_valida(
               'f' || pg_catalog.repeat('-', 160)
           ) IS NOT FALSE
       OR vec_autorizacion_atestada_v3
           .referencia_opaca_fuente_corporativa_valida('_ab') IS NOT FALSE
       OR vec_autorizacion_atestada_v3
           .referencia_opaca_fuente_corporativa_valida('a+b') IS NOT FALSE
       OR vec_autorizacion_atestada_v3
           .referencia_opaca_fuente_corporativa_valida('a b') IS NOT FALSE
       OR vec_autorizacion_atestada_v3
           .referencia_opaca_fuente_corporativa_valida('ａbc') IS NOT FALSE THEN
        RAISE EXCEPTION USING ERRCODE = 'XX000',
            MESSAGE = 'A1: regex de fuente_ref inválida';
    END IF;

    -- El mismo perfil único se aplica a evento_fuente_ref y efecto_ref.
    IF vec_autorizacion_atestada_v3
           .referencia_opaca_fuente_corporativa_valida('evt:fuente/001')
          IS NOT TRUE
       OR vec_autorizacion_atestada_v3
           .referencia_opaca_fuente_corporativa_valida('efecto_recurso-001')
          IS NOT TRUE
       OR vec_autorizacion_atestada_v3
           .referencia_opaca_fuente_corporativa_valida('événement')
          IS NOT FALSE THEN
        RAISE EXCEPTION USING ERRCODE = 'XX000',
            MESSAGE = 'A1: perfil de evento o efecto inválido';
    END IF;

    IF vec_autorizacion_atestada_v3.operacion_ref_fuente_corporativa_valida(
           'oca_' || pg_catalog.repeat('a', 24)
       ) IS NOT TRUE
       OR vec_autorizacion_atestada_v3.operacion_ref_fuente_corporativa_valida(
              'oca_' || pg_catalog.repeat('Z', 128)
          ) IS NOT TRUE
       OR vec_autorizacion_atestada_v3.operacion_ref_fuente_corporativa_valida(
              NULL
          ) IS NOT FALSE
       OR vec_autorizacion_atestada_v3.operacion_ref_fuente_corporativa_valida(
              ''
          ) IS NOT FALSE
       OR vec_autorizacion_atestada_v3.operacion_ref_fuente_corporativa_valida(
              'oca_' || pg_catalog.repeat('a', 23)
          ) IS NOT FALSE
       OR vec_autorizacion_atestada_v3.operacion_ref_fuente_corporativa_valida(
              'oca_' || pg_catalog.repeat('a', 129)
          ) IS NOT FALSE
       OR vec_autorizacion_atestada_v3.operacion_ref_fuente_corporativa_valida(
              'oca_' || pg_catalog.repeat('a', 23) || '/'
          ) IS NOT FALSE
       OR vec_autorizacion_atestada_v3.operacion_ref_fuente_corporativa_valida(
              'ocａ_' || pg_catalog.repeat('a', 24)
          ) IS NOT FALSE THEN
        RAISE EXCEPTION USING ERRCODE = 'XX000',
            MESSAGE = 'A1: contrato de operacion_ref inválido';
    END IF;
END
$textos_y_referencias$;

DO $numeros_y_huellas$
BEGIN
    IF vec_autorizacion_atestada_v3.entero_json_seguro_fuente_valido(
           '1'::pg_catalog.json
       ) IS NOT TRUE
       OR vec_autorizacion_atestada_v3.entero_json_seguro_fuente_valido(
              '1234567890'::pg_catalog.json
          ) IS NOT TRUE
       OR vec_autorizacion_atestada_v3.entero_json_seguro_fuente_valido(
              '9007199254740991'::pg_catalog.json
          ) IS NOT TRUE
       OR vec_autorizacion_atestada_v3.entero_json_seguro_fuente_valido(NULL)
          IS NOT FALSE
       OR vec_autorizacion_atestada_v3.entero_json_seguro_fuente_valido(
              'null'::pg_catalog.json
          ) IS NOT FALSE
       OR vec_autorizacion_atestada_v3.entero_json_seguro_fuente_valido(
              '"1"'::pg_catalog.json
          ) IS NOT FALSE
       OR vec_autorizacion_atestada_v3.entero_json_seguro_fuente_valido(
              '0'::pg_catalog.json
          ) IS NOT FALSE
       OR vec_autorizacion_atestada_v3.entero_json_seguro_fuente_valido(
              '-1'::pg_catalog.json
          ) IS NOT FALSE
       OR vec_autorizacion_atestada_v3.entero_json_seguro_fuente_valido(
              '1.0'::pg_catalog.json
          ) IS NOT FALSE
       OR vec_autorizacion_atestada_v3.entero_json_seguro_fuente_valido(
              '1e0'::pg_catalog.json
          ) IS NOT FALSE
       OR vec_autorizacion_atestada_v3.entero_json_seguro_fuente_valido(
              '1E+3'::pg_catalog.json
          ) IS NOT FALSE
       OR vec_autorizacion_atestada_v3.entero_json_seguro_fuente_valido(
              '9007199254740992'::pg_catalog.json
          ) IS NOT FALSE
       OR vec_autorizacion_atestada_v3.entero_json_seguro_fuente_valido(
              ' 1 '::pg_catalog.json
          ) IS NOT FALSE THEN
        RAISE EXCEPTION USING ERRCODE = 'XX000',
            MESSAGE = 'A1: entero JSON seguro inválido';
    END IF;

    -- La huella común permanece en la primitiva V3: minúscula y no nula.
    IF vec_autorizacion_atestada_v3.huella_sha256_valida(
           '1' || pg_catalog.repeat('0', 63)
       ) IS NOT TRUE
       OR vec_autorizacion_atestada_v3.huella_sha256_valida(NULL) IS TRUE
       OR vec_autorizacion_atestada_v3.huella_sha256_valida('') IS TRUE
       OR vec_autorizacion_atestada_v3.huella_sha256_valida(
              pg_catalog.repeat('0', 64)
          ) IS TRUE
       OR vec_autorizacion_atestada_v3.huella_sha256_valida(
              'A' || pg_catalog.repeat('0', 63)
          ) IS TRUE
       OR vec_autorizacion_atestada_v3.huella_sha256_valida(
              pg_catalog.repeat('a', 63)
          ) IS TRUE
       OR vec_autorizacion_atestada_v3.huella_sha256_valida(
              pg_catalog.repeat('a', 65)
          ) IS TRUE
       OR vec_autorizacion_atestada_v3.huella_sha256_valida(
              'ａ' || pg_catalog.repeat('0', 63)
          ) IS TRUE THEN
        RAISE EXCEPTION USING ERRCODE = 'XX000',
            MESSAGE = 'A1: primitiva SHA-256 V3 inválida';
    END IF;
END
$numeros_y_huellas$;

SET LOCAL timezone = 'Europe/Madrid';

DO $instantes$
DECLARE
    v_canon pg_catalog.text := '2026-08-01T12:34:56.123456Z';
    v_instante pg_catalog.timestamptz := v_canon::pg_catalog.timestamptz;
BEGIN
    IF vec_autorizacion_atestada_v3.instante_fuente_finito_valido(v_instante)
          IS NOT TRUE
       OR vec_autorizacion_atestada_v3.instante_fuente_finito_valido(NULL)
          IS NOT FALSE
       OR vec_autorizacion_atestada_v3.instante_fuente_finito_valido(
              'infinity'::pg_catalog.timestamptz
          ) IS NOT FALSE
       OR vec_autorizacion_atestada_v3.instante_fuente_finito_valido(
              '-infinity'::pg_catalog.timestamptz
          ) IS NOT FALSE
       OR vec_autorizacion_atestada_v3.instante_fuente_finito_valido(
              '0001-01-01 00:00:00+00'::pg_catalog.timestamptz
          ) IS NOT TRUE
       OR vec_autorizacion_atestada_v3.instante_fuente_finito_valido(
              '9999-12-31 23:59:59.999999+00'::pg_catalog.timestamptz
          ) IS NOT TRUE
       OR vec_autorizacion_atestada_v3.instante_fuente_finito_valido(
              '10000-01-01 00:00:00+00'::pg_catalog.timestamptz
          ) IS NOT FALSE
       OR vec_autorizacion_atestada_v3.instante_fuente_finito_valido(
              '0001-01-01 00:00:00 BC'::pg_catalog.timestamptz
          ) IS NOT FALSE THEN
        RAISE EXCEPTION USING ERRCODE = 'XX000',
            MESSAGE = 'A1: instante timestamptz finito inválido';
    END IF;

    IF vec_autorizacion_atestada_v3.representacion_instante_utc_fuente(
           v_instante
       ) IS DISTINCT FROM v_canon
       OR vec_autorizacion_atestada_v3.representacion_instante_utc_fuente(NULL)
          IS NOT NULL
       OR vec_autorizacion_atestada_v3.representacion_instante_utc_fuente(
              'infinity'::pg_catalog.timestamptz
          ) IS NOT NULL
       OR vec_autorizacion_atestada_v3.representacion_instante_utc_fuente(
              '2026-08-01 14:34:56.123456+02'::pg_catalog.timestamptz
          ) IS DISTINCT FROM v_canon THEN
        RAISE EXCEPTION USING ERRCODE = 'XX000',
            MESSAGE = 'A1: representación UTC dependiente de zona';
    END IF;

    IF vec_autorizacion_atestada_v3.instante_utc_fuente_texto_valido(v_canon)
          IS NOT TRUE
       OR vec_autorizacion_atestada_v3.instante_utc_fuente_texto_valido(
              '0001-01-01T00:00:00.000000Z'
          ) IS NOT TRUE
       OR vec_autorizacion_atestada_v3.instante_utc_fuente_texto_valido(
              '9999-12-31T23:59:59.999999Z'
          ) IS NOT TRUE
       OR vec_autorizacion_atestada_v3.instante_utc_fuente_texto_valido(NULL)
          IS NOT FALSE
       OR vec_autorizacion_atestada_v3.instante_utc_fuente_texto_valido('')
          IS NOT FALSE
       OR vec_autorizacion_atestada_v3.instante_utc_fuente_texto_valido(
              'infinity'
          ) IS NOT FALSE
       OR vec_autorizacion_atestada_v3.instante_utc_fuente_texto_valido(
              '0000-01-01T00:00:00.000000Z'
          ) IS NOT FALSE
       OR vec_autorizacion_atestada_v3.instante_utc_fuente_texto_valido(
              '10000-01-01T00:00:00.000000Z'
          ) IS NOT FALSE
       OR vec_autorizacion_atestada_v3.instante_utc_fuente_texto_valido(
              '2026-08-01T12:34:56.123456+00:00'
          ) IS NOT FALSE
       OR vec_autorizacion_atestada_v3.instante_utc_fuente_texto_valido(
              '2026-08-01T14:34:56.123456+02:00'
          ) IS NOT FALSE
       OR vec_autorizacion_atestada_v3.instante_utc_fuente_texto_valido(
              '2026-08-01T12:34:56.12345Z'
          ) IS NOT FALSE
       OR vec_autorizacion_atestada_v3.instante_utc_fuente_texto_valido(
              '2026-08-01T12:34:56.1234567Z'
          ) IS NOT FALSE
       OR vec_autorizacion_atestada_v3.instante_utc_fuente_texto_valido(
              '2026-08-01T12:34:56Z'
          ) IS NOT FALSE
       OR vec_autorizacion_atestada_v3.instante_utc_fuente_texto_valido(
              '2026-08-01T12:34:56.123456z'
          ) IS NOT FALSE
       OR vec_autorizacion_atestada_v3.instante_utc_fuente_texto_valido(
              '2026-02-29T12:34:56.123456Z'
          ) IS NOT FALSE
       OR vec_autorizacion_atestada_v3.instante_utc_fuente_texto_valido(
              '2026-08-01T24:00:00.000000Z'
          ) IS NOT FALSE
       OR vec_autorizacion_atestada_v3.instante_utc_fuente_texto_valido(
              '2026-08-01T12:34:60.000000Z'
          ) IS NOT FALSE THEN
        RAISE EXCEPTION USING ERRCODE = 'XX000',
            MESSAGE = 'A1: texto temporal canónico inválido';
    END IF;

    IF vec_autorizacion_atestada_v3.representacion_instante_utc_fuente(
           v_canon::pg_catalog.timestamptz
       ) IS DISTINCT FROM v_canon THEN
        RAISE EXCEPTION USING ERRCODE = 'XX000',
            MESSAGE = 'A1: roundtrip temporal no exacto';
    END IF;
END
$instantes$;
