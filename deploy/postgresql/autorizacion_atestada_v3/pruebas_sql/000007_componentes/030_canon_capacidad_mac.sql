-- Matriz adversarial del canon, preimagen y MAC de capacidad F0-A3.
CREATE FUNCTION
vec_autorizacion_atestada_v3.acreditar_forma_canon_capacidad_a3_prueba()
RETURNS pg_catalog.bool
LANGUAGE sql
VOLATILE
SET search_path = pg_catalog
AS $funcion$
    WITH propietario AS (
        SELECT r.oid
          FROM pg_catalog.pg_roles AS r
         WHERE r.rolname = 'vec_autorizacion_atestada_v3_propietario'
           AND NOT r.rolcanlogin
           AND NOT r.rolsuper
           AND NOT r.rolcreatedb
           AND NOT r.rolcreaterole
           AND NOT r.rolreplication
           AND NOT r.rolbypassrls
    ), esperada(nombre, argumentos, retorno, nombres_argumentos, huella) AS (
        VALUES
        ('capacidad_fuente_corporativa_v1_canonica', 'bytea', 'bytea',
         ARRAY['p_capacidad']::pg_catalog.text[],
         '18ddbe29f482044926c30b1b1e3fa3c3941cb56e8812934a0ac68146941ca406'),
        ('preimagen_mac_fuente_corporativa_v1', 'bytea', 'bytea',
         ARRAY['p_capacidad']::pg_catalog.text[],
         'dbe0617b54871024b910391fd380c904e4be57b22974786359a6a07222d479a7'),
        ('mac_capacidad_fuente_corporativa_v1_valido', 'bytea, bytea',
         'boolean', ARRAY['p_capacidad', 'p_secreto_hmac']::pg_catalog.text[],
         'bc29b4f84e495a076b83cc8626a8de5063bbcff0ec763ebaf47a4776590af58b')
    ), candidata AS (
        SELECT p.oid, p.proname
          FROM pg_catalog.pg_proc AS p
          JOIN pg_catalog.pg_namespace AS n ON n.oid = p.pronamespace
          JOIN propietario AS o ON o.oid = p.proowner
          JOIN esperada AS e
            ON e.nombre = p.proname
           AND e.argumentos = pg_catalog.oidvectortypes(p.proargtypes)
           AND e.retorno = pg_catalog.format_type(p.prorettype, NULL)
           AND e.nombres_argumentos = p.proargnames
         WHERE n.nspname = 'vec_autorizacion_atestada_v3'
           AND p.prokind = 'f'
           AND p.proallargtypes IS NULL
           AND p.proargmodes IS NULL
           AND p.pronargs = pg_catalog.cardinality(e.nombres_argumentos)
           AND p.pronargdefaults = 0
           AND p.proargdefaults IS NULL
           AND p.provariadic = 0
           AND NOT p.proretset
           AND p.provolatile = 'i'
           AND NOT p.proisstrict
           AND NOT p.prosecdef
           AND NOT p.proleakproof
           AND p.proparallel = 'u'
           AND p.procost = 100
           AND p.prorows = 0
           AND p.prosupport = 0
           AND p.protrftypes IS NULL
           AND p.probin IS NULL
           AND p.prosqlbody IS NULL
           AND p.proconfig = ARRAY['search_path=pg_catalog']
           AND pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
                   p.prosrc, 'UTF8')), 'hex') = e.huella
           AND p.prolang = (
               SELECT l.oid FROM pg_catalog.pg_language AS l
                WHERE l.lanname = 'plpgsql'
           )
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
    SELECT (SELECT pg_catalog.count(*) FROM propietario) = 1
       AND (SELECT pg_catalog.count(*) FROM candidata) = 3
       AND NOT EXISTS (
           SELECT 1 FROM esperada AS e
            WHERE NOT EXISTS (
                SELECT 1 FROM candidata AS c WHERE c.proname = e.nombre
            )
       )
       AND (
           SELECT pg_catalog.count(*)
             FROM pg_catalog.pg_proc AS p
             JOIN pg_catalog.pg_namespace AS n ON n.oid = p.pronamespace
            WHERE n.nspname = 'vec_autorizacion_atestada_v3'
              AND p.proname IN (SELECT e.nombre FROM esperada AS e)
       ) = 3
       AND EXISTS (
           SELECT 1
             FROM pg_catalog.pg_proc AS d
             JOIN pg_catalog.pg_namespace AS n ON n.oid = d.pronamespace
             JOIN propietario AS o ON o.oid = d.proowner
            WHERE d.oid = 'vec_autorizacion_atestada_v3.bytea_igual_constante(bytea,bytea)'::pg_catalog.regprocedure
              AND n.nspname = 'vec_autorizacion_atestada_v3'
              AND NOT pg_catalog.has_function_privilege(
                  'vec_autorizacion_atestada_v3_migrador', d.oid, 'EXECUTE')
              AND NOT EXISTS (
                  SELECT 1 FROM pg_catalog.aclexplode(COALESCE(
                      d.proacl, pg_catalog.acldefault('f', d.proowner))) AS a
                   WHERE a.grantee <> o.oid OR a.privilege_type <> 'EXECUTE'
                      OR a.is_grantable)
       )
$funcion$;
REVOKE ALL ON FUNCTION
    vec_autorizacion_atestada_v3.acreditar_forma_canon_capacidad_a3_prueba()
    FROM PUBLIC;
CREATE FUNCTION vec_autorizacion_atestada_v3.exigir_forma_a3_prueba(
    p_esperada pg_catalog.bool)
RETURNS pg_catalog.void LANGUAGE plpgsql VOLATILE
SET search_path = pg_catalog AS $exigir$
BEGIN
    IF vec_autorizacion_atestada_v3
           .acreditar_forma_canon_capacidad_a3_prueba()
       IS DISTINCT FROM p_esperada THEN
        RAISE EXCEPTION USING ERRCODE = 'XX000',
            MESSAGE = 'A3: mutación catalogal no detectada o no restaurada';
    END IF;
END
$exigir$;
REVOKE ALL ON FUNCTION
    vec_autorizacion_atestada_v3.exigir_forma_a3_prueba(pg_catalog.bool)
    FROM PUBLIC;
SELECT vec_autorizacion_atestada_v3.exigir_forma_a3_prueba(true);
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3
    .bytea_igual_constante(pg_catalog.bytea, pg_catalog.bytea) TO PUBLIC;
SELECT vec_autorizacion_atestada_v3.exigir_forma_a3_prueba(false);
REVOKE EXECUTE ON FUNCTION vec_autorizacion_atestada_v3
    .bytea_igual_constante(pg_catalog.bytea, pg_catalog.bytea) FROM PUBLIC;
SELECT vec_autorizacion_atestada_v3.exigir_forma_a3_prueba(true);
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3
    .preimagen_mac_fuente_corporativa_v1(pg_catalog.bytea)
    TO vec_autorizacion_atestada_v3_migrador;
SELECT vec_autorizacion_atestada_v3.exigir_forma_a3_prueba(false);
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3
    .preimagen_mac_fuente_corporativa_v1(pg_catalog.bytea)
    FROM vec_autorizacion_atestada_v3_migrador;
SELECT vec_autorizacion_atestada_v3.exigir_forma_a3_prueba(true);
CREATE FUNCTION
vec_autorizacion_atestada_v3.capacidad_fuente_corporativa_v1_canonica(
    p_capacidad pg_catalog.text)
RETURNS pg_catalog.bytea LANGUAGE sql IMMUTABLE
SET search_path = pg_catalog AS $sobrecarga$
    SELECT NULL::pg_catalog.bytea
$sobrecarga$;
REVOKE ALL ON FUNCTION
    vec_autorizacion_atestada_v3
        .capacidad_fuente_corporativa_v1_canonica(pg_catalog.text)
    FROM PUBLIC;
SELECT vec_autorizacion_atestada_v3.exigir_forma_a3_prueba(false);
DROP FUNCTION vec_autorizacion_atestada_v3
    .capacidad_fuente_corporativa_v1_canonica(pg_catalog.text);
SELECT vec_autorizacion_atestada_v3.exigir_forma_a3_prueba(true);
ALTER FUNCTION vec_autorizacion_atestada_v3
    .preimagen_mac_fuente_corporativa_v1(pg_catalog.bytea) PARALLEL SAFE;
SELECT vec_autorizacion_atestada_v3.exigir_forma_a3_prueba(false);
ALTER FUNCTION vec_autorizacion_atestada_v3
    .preimagen_mac_fuente_corporativa_v1(pg_catalog.bytea) PARALLEL UNSAFE;
SELECT vec_autorizacion_atestada_v3.exigir_forma_a3_prueba(true);
ALTER FUNCTION vec_autorizacion_atestada_v3
    .mac_capacidad_fuente_corporativa_v1_valido(
        pg_catalog.bytea, pg_catalog.bytea) SECURITY DEFINER;
SELECT vec_autorizacion_atestada_v3.exigir_forma_a3_prueba(false);
ALTER FUNCTION vec_autorizacion_atestada_v3
    .mac_capacidad_fuente_corporativa_v1_valido(
        pg_catalog.bytea, pg_catalog.bytea) SECURITY INVOKER;
SELECT vec_autorizacion_atestada_v3.exigir_forma_a3_prueba(true);
ALTER FUNCTION vec_autorizacion_atestada_v3
    .capacidad_fuente_corporativa_v1_canonica(pg_catalog.bytea) COST 777;
SELECT vec_autorizacion_atestada_v3.exigir_forma_a3_prueba(false);
ALTER FUNCTION vec_autorizacion_atestada_v3
    .capacidad_fuente_corporativa_v1_canonica(pg_catalog.bytea) COST 100;
SELECT vec_autorizacion_atestada_v3.exigir_forma_a3_prueba(true);

DO $cuerpos_hostiles$
DECLARE
    v_mac_original pg_catalog.text := pg_catalog.pg_get_functiondef(
        'vec_autorizacion_atestada_v3.mac_capacidad_fuente_corporativa_v1_valido(bytea,bytea)'::pg_catalog.regprocedure);
    v_auxiliar_original pg_catalog.text := pg_catalog.pg_get_functiondef(
        'vec_autorizacion_atestada_v3.preimagen_mac_fuente_corporativa_v1(bytea)'::pg_catalog.regprocedure);
    v_mutante pg_catalog.text;
BEGIN
    v_mutante := pg_catalog.replace(v_mac_original,
        'RETURN vec_autorizacion_atestada_v3.bytea_igual_constante(',
        'RETURN (');
    v_mutante := pg_catalog.replace(v_mutante,
        E'public.hmac(v_preimagen, p_secreto_hmac, ''sha256''),\n        pg_catalog.decode',
        E'public.hmac(v_preimagen, p_secreto_hmac, ''sha256'') =\n        pg_catalog.decode');
    IF v_mutante = v_mac_original THEN
        RAISE EXCEPTION USING ERRCODE = 'XX000',
            MESSAGE = 'A3: el mutante de comparación no se construyó';
    END IF;
    EXECUTE v_mutante;
    PERFORM vec_autorizacion_atestada_v3.exigir_forma_a3_prueba(false);
    EXECUTE v_mac_original;
    PERFORM vec_autorizacion_atestada_v3.exigir_forma_a3_prueba(true);
    v_mutante := pg_catalog.replace(v_auxiliar_original,
        'VEC-CONTEXTO-ACTOR-FUENTE-CORPORATIVA-V1',
        'VEC-CONTEXTO-ACTOR-FUENTE-CORPORATIVA-V2');
    IF v_mutante = v_auxiliar_original THEN
        RAISE EXCEPTION USING ERRCODE = 'XX000',
            MESSAGE = 'A3: el mutante auxiliar no se construyó';
    END IF;
    EXECUTE v_mutante;
    PERFORM vec_autorizacion_atestada_v3.exigir_forma_a3_prueba(false);
    EXECUTE v_auxiliar_original;
    PERFORM vec_autorizacion_atestada_v3.exigir_forma_a3_prueba(true);
END
$cuerpos_hostiles$;
DROP FUNCTION vec_autorizacion_atestada_v3.exigir_forma_a3_prueba(
    pg_catalog.bool);
DROP FUNCTION
    vec_autorizacion_atestada_v3.acreditar_forma_canon_capacidad_a3_prueba();
-- El vector nace de 33 valores tipados. Longitud, huellas y MAC proceden de
-- V0; no se duplica aquí el JSON canónico completo.
CREATE FUNCTION vec_autorizacion_atestada_v3.vector_capacidad_a3_prueba()
RETURNS pg_catalog.bytea
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog
AS $vector$
    SELECT pg_catalog.convert_to(
        pg_catalog.replace(
            pg_catalog.replace(
                pg_catalog.json_build_object(
                    'esquema',
                    'vec.contexto-actor.fuente-corporativa.capacidad.v1',
                    'version', 1,
                    'fuente_ref', 'fuente:sintetica:f0:v1',
                    'fuente_version', 1,
                    'evento_fuente_ref', 'evento:sintetico:f0:0001',
                    'huella_evento_fuente_sha256', pg_catalog.repeat('1', 64),
                    'evento_fuente_emitido_en',
                    '2099-01-01T00:00:00.000000Z',
                    'huella_manifiesto_fuente_sha256',
                    'f16cab3533e7a5b4126ae1bddf8afbc989ce564330cf9703d1429ceb21678325',
                    'huella_sobre_cose_sign1_sha256', pg_catalog.repeat('2', 64),
                    'huella_prueba_confianza_sha256', pg_catalog.repeat('3', 64),
                    'audiencia_consumo',
                    'vec_contexto_actor.publicar_organizacion_corporativa_fuente.v1',
                    'accion', 'contexto_actor.organizacion_corporativa.publicar',
                    'tipo_efecto', 'organizacion_corporativa.alta',
                    'operacion_ref', 'oca_vector_f0_0000000000000001',
                    'efecto_ref', 'efecto:sintetico:f0:0001',
                    'huella_efecto_sha256', pg_catalog.repeat('4', 64),
                    'clave_id', 'clave:sintetica:f0:0001',
                    'clave_version', 2,
                    'revision_gobierno', 7,
                    'huella_gobierno_sha256', pg_catalog.repeat('5', 64),
                    'emisor_id', 'emisor:sintetico:f0:0001',
                    'configuracion_revision',
                    'configuracion:sintetica:f0:0001',
                    'configuracion_secuencia', 11,
                    'huella_configuracion_sha256', pg_catalog.repeat('6', 64),
                    'raiz_clave_id', 'raiz:sintetica:f0:0001',
                    'raiz_version', 3,
                    'huella_raiz_spki_sha256', pg_catalog.repeat('7', 64),
                    'audiencia_despliegue',
                    'vec-diputacion/pruebas/f0/fuente-corporativa',
                    'suite', 'VEC-AD-3-COSE-EDDSA-1',
                    'nonce', pg_catalog.repeat('8', 64),
                    'emitida_en', '2099-01-01T00:00:00.500000Z',
                    'expira_en', '2099-01-01T00:00:04.500000Z',
                    'mac_sha256',
                    '16279292d2792d56230b30787745231a11a3fad585a9e476d65dc5d3723325b0'
                )::pg_catalog.text,
                '" : ', '":'
            ),
            ', "', ',"'
        ),
        'UTF8'
    )
$vector$;

CREATE FUNCTION vec_autorizacion_atestada_v3.mutar_capacidad_a3_prueba(
    p_base pg_catalog.bytea,
    p_clave pg_catalog.text,
    p_valor_json pg_catalog.text
)
RETURNS pg_catalog.bytea
LANGUAGE plpgsql
IMMUTABLE
SET search_path = pg_catalog
AS $mutacion$
DECLARE
    v_documento pg_catalog.json;
    v_anterior pg_catalog.text;
    v_texto pg_catalog.text;
BEGIN
    v_texto := pg_catalog.convert_from(p_base, 'UTF8');
    v_documento := v_texto::pg_catalog.json;
    SELECT e.valor::pg_catalog.text INTO STRICT v_anterior
      FROM pg_catalog.json_each(v_documento) AS e(clave, valor)
     WHERE e.clave = p_clave;
    RETURN pg_catalog.convert_to(
        pg_catalog.replace(
            v_texto,
            '"' || p_clave || '":' || v_anterior,
            '"' || p_clave || '":' || p_valor_json
        ),
        'UTF8'
    );
END
$mutacion$;

REVOKE ALL ON FUNCTION
    vec_autorizacion_atestada_v3.vector_capacidad_a3_prueba(),
    vec_autorizacion_atestada_v3.mutar_capacidad_a3_prueba(
        pg_catalog.bytea, pg_catalog.text, pg_catalog.text
    ) FROM PUBLIC;

DO $vector_dorado$
DECLARE
    v_vector pg_catalog.bytea := vec_autorizacion_atestada_v3
        .vector_capacidad_a3_prueba();
    v_preimagen pg_catalog.bytea;
    v_clave pg_catalog.bytea := pg_catalog.sha256(
        pg_catalog.convert_to('material-hmac-sintetico-f0-v1', 'UTF8')
    );
BEGIN
    v_preimagen := vec_autorizacion_atestada_v3
        .preimagen_mac_fuente_corporativa_v1(v_vector);
    IF pg_catalog.octet_length(v_vector) <> 1891
       OR pg_catalog.encode(pg_catalog.sha256(v_vector), 'hex') <>
          'd3baaa6bf9e8e757d659f42233186a799e3c0b6e9a8e5eab1b5930ca0e7f7e54'
       OR vec_autorizacion_atestada_v3
              .capacidad_fuente_corporativa_v1_canonica(v_vector)
          IS DISTINCT FROM v_vector
       OR pg_catalog.octet_length(v_preimagen) <> 1280
       OR pg_catalog.encode(pg_catalog.sha256(v_preimagen), 'hex') <>
          '334ec3d3b1f648cee1a9a9a387d704ed448772f0e03bb4d97c08b933518be3d5'
       OR vec_autorizacion_atestada_v3
              .mac_capacidad_fuente_corporativa_v1_valido(
                  v_vector, v_clave
              ) IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = 'XX000',
            MESSAGE = 'A3: vector, preimagen o MAC no coincide con V0';
    END IF;
END
$vector_dorado$;

DO $orden_claves_y_tipos$
DECLARE
    v_base pg_catalog.bytea := vec_autorizacion_atestada_v3
        .vector_capacidad_a3_prueba();
    v_texto pg_catalog.text := pg_catalog.convert_from(v_base, 'UTF8');
    v_caso pg_catalog.bytea;
    v_clave pg_catalog.text;
BEGIN
    FOREACH v_caso IN ARRAY ARRAY[
        pg_catalog.convert_to(' ' || v_texto, 'UTF8'),
        pg_catalog.convert_to(
            pg_catalog.replace(
                v_texto,
                '{"esquema":' || vec_autorizacion_atestada_v3.texto_json_go(
                    'vec.contexto-actor.fuente-corporativa.capacidad.v1'
                ) || ',"version":1',
                '{"version":1,"esquema":' ||
                vec_autorizacion_atestada_v3.texto_json_go(
                    'vec.contexto-actor.fuente-corporativa.capacidad.v1'
                )
            ),
            'UTF8'
        ),
        pg_catalog.convert_to(
            pg_catalog.replace(
                v_texto, '{"esquema":',
                '{"esquema":"duplicado","esquema":'
            ),
            'UTF8'
        ),
        pg_catalog.convert_to(
            pg_catalog.left(v_texto, pg_catalog.length(v_texto) - 1) ||
            ',"sobrante":"x"}', 'UTF8'
        ),
        pg_catalog.convert_to(
            pg_catalog.replace(
                v_texto,
                ',"mac_sha256":' || vec_autorizacion_atestada_v3.texto_json_go(
                    '16279292d2792d56230b30787745231a11a3fad585a9e476d65dc5d3723325b0'
                ),
                ''
            ),
            'UTF8'
        ),
        pg_catalog.convert_to('[' || v_texto || ']', 'UTF8'),
        pg_catalog.convert_to(pg_catalog.left(v_texto, 300), 'UTF8')
    ] LOOP
        IF vec_autorizacion_atestada_v3
               .capacidad_fuente_corporativa_v1_canonica(v_caso)
           IS NOT NULL THEN
            RAISE EXCEPTION USING ERRCODE = 'XX000',
                MESSAGE = 'A3: orden, repetición o forma abierta aceptada';
        END IF;
    END LOOP;

    FOREACH v_clave IN ARRAY ARRAY[
        'version', 'fuente_version', 'clave_version',
        'revision_gobierno', 'configuracion_secuencia', 'raiz_version'
    ] LOOP
        FOREACH v_caso IN ARRAY ARRAY[
            vec_autorizacion_atestada_v3.mutar_capacidad_a3_prueba(
                v_base, v_clave, '"1"'),
            vec_autorizacion_atestada_v3.mutar_capacidad_a3_prueba(
                v_base, v_clave, '1.0'),
            vec_autorizacion_atestada_v3.mutar_capacidad_a3_prueba(
                v_base, v_clave, '1e0'),
            vec_autorizacion_atestada_v3.mutar_capacidad_a3_prueba(
                v_base, v_clave, 'null')
        ] LOOP
            IF vec_autorizacion_atestada_v3
                   .capacidad_fuente_corporativa_v1_canonica(v_caso)
               IS NOT NULL THEN
                RAISE EXCEPTION USING ERRCODE = 'XX000',
                    MESSAGE = 'A3: tipo o léxico numérico inválido aceptado';
            END IF;
        END LOOP;
    END LOOP;

    FOREACH v_clave IN ARRAY ARRAY[
        'esquema', 'fuente_ref', 'evento_fuente_ref',
        'huella_evento_fuente_sha256', 'evento_fuente_emitido_en',
        'huella_manifiesto_fuente_sha256',
        'huella_sobre_cose_sign1_sha256',
        'huella_prueba_confianza_sha256', 'audiencia_consumo', 'accion',
        'tipo_efecto', 'operacion_ref', 'efecto_ref',
        'huella_efecto_sha256', 'clave_id', 'huella_gobierno_sha256',
        'emisor_id', 'configuracion_revision',
        'huella_configuracion_sha256', 'raiz_clave_id',
        'huella_raiz_spki_sha256', 'audiencia_despliegue', 'suite',
        'nonce', 'emitida_en', 'expira_en', 'mac_sha256'
    ] LOOP
        FOREACH v_caso IN ARRAY ARRAY[
            vec_autorizacion_atestada_v3.mutar_capacidad_a3_prueba(
                v_base, v_clave, 'null'
            ),
            vec_autorizacion_atestada_v3.mutar_capacidad_a3_prueba(
                v_base, v_clave, '1'
            )
        ] LOOP
            IF vec_autorizacion_atestada_v3
                   .capacidad_fuente_corporativa_v1_canonica(v_caso)
               IS NOT NULL THEN
                RAISE EXCEPTION USING ERRCODE = 'XX000',
                    MESSAGE = 'A3: cadena ausente o de tipo erróneo aceptada';
            END IF;
        END LOOP;
    END LOOP;
END
$orden_claves_y_tipos$;

DO $escapes_utf8_y_limites$
DECLARE
    v_base pg_catalog.bytea := vec_autorizacion_atestada_v3
        .vector_capacidad_a3_prueba();
    v_escape pg_catalog.bytea;
BEGIN
    v_escape := vec_autorizacion_atestada_v3.mutar_capacidad_a3_prueba(
        v_base, 'emisor_id',
        vec_autorizacion_atestada_v3.texto_json_go('emisor<&>')
    );
    IF vec_autorizacion_atestada_v3
           .capacidad_fuente_corporativa_v1_canonica(v_escape)
       IS DISTINCT FROM v_escape
       OR vec_autorizacion_atestada_v3
           .capacidad_fuente_corporativa_v1_canonica(
               vec_autorizacion_atestada_v3.mutar_capacidad_a3_prueba(
                   v_base, 'emisor_id', '"emisor<&>"'
               )
           ) IS NOT NULL
       OR vec_autorizacion_atestada_v3
           .capacidad_fuente_corporativa_v1_canonica(
               vec_autorizacion_atestada_v3.mutar_capacidad_a3_prueba(
                   v_base, 'emisor_id', '"\u0065misor:sintetico:f0:0001"'
               )
           ) IS NOT NULL
       OR vec_autorizacion_atestada_v3
           .capacidad_fuente_corporativa_v1_canonica(
               pg_catalog.decode('efbbbf', 'hex') || v_base
           ) IS NOT NULL
       OR vec_autorizacion_atestada_v3
           .capacidad_fuente_corporativa_v1_canonica(
               pg_catalog.decode('c328', 'hex')
           ) IS NOT NULL
       OR vec_autorizacion_atestada_v3
           .capacidad_fuente_corporativa_v1_canonica(
               pg_catalog.substr(v_base, 1, 220) ||
               pg_catalog.decode('00', 'hex') ||
               pg_catalog.substr(v_base, 221)
           ) IS NOT NULL
       OR vec_autorizacion_atestada_v3
           .capacidad_fuente_corporativa_v1_canonica(NULL) IS NOT NULL
       OR vec_autorizacion_atestada_v3
           .capacidad_fuente_corporativa_v1_canonica(
               pg_catalog.convert_to(pg_catalog.repeat('x', 511), 'UTF8')
           ) IS NOT NULL
       OR vec_autorizacion_atestada_v3
           .capacidad_fuente_corporativa_v1_canonica(
               pg_catalog.convert_to(pg_catalog.repeat('x', 32769), 'UTF8')
           ) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = 'XX000',
            MESSAGE = 'A3: escape, UTF-8 o límite no cerrado';
    END IF;
END
$escapes_utf8_y_limites$;

DO $numeros_semantica_y_cruces$
DECLARE
    v_base pg_catalog.bytea := vec_autorizacion_atestada_v3
        .vector_capacidad_a3_prueba();
    v_caso pg_catalog.bytea;
    v_clave pg_catalog.text;
    v_cruce record;
BEGIN
    FOREACH v_clave IN ARRAY ARRAY[
        'version', 'fuente_version', 'clave_version',
        'revision_gobierno', 'configuracion_secuencia', 'raiz_version'
    ] LOOP
        FOREACH v_caso IN ARRAY ARRAY[
            vec_autorizacion_atestada_v3.mutar_capacidad_a3_prueba(
                v_base, v_clave, '0'),
            vec_autorizacion_atestada_v3.mutar_capacidad_a3_prueba(
                v_base, v_clave, '9007199254740992')
        ] LOOP
            IF vec_autorizacion_atestada_v3
                   .capacidad_fuente_corporativa_v1_canonica(v_caso)
               IS NOT NULL THEN
                RAISE EXCEPTION USING ERRCODE = 'XX000',
                    MESSAGE = 'A3: entero fuera de rango aceptado';
            END IF;
        END LOOP;
        IF v_clave <> 'version' THEN
            v_caso := vec_autorizacion_atestada_v3.mutar_capacidad_a3_prueba(
                v_base, v_clave, '9007199254740991'
            );
            IF vec_autorizacion_atestada_v3
                   .capacidad_fuente_corporativa_v1_canonica(v_caso)
               IS DISTINCT FROM v_caso THEN
                RAISE EXCEPTION USING ERRCODE = 'XX000',
                    MESSAGE = 'A3: máximo entero seguro rechazado';
            END IF;
        END IF;
    END LOOP;

    FOR v_cruce IN
        SELECT * FROM (VALUES
            ('vec_contexto_actor.publicar_organizacion_corporativa_fuente.v1',
             'contexto_actor.organizacion_corporativa.publicar',
             'organizacion_corporativa.alta'),
            ('vec_contexto_actor.revocar_organizacion_corporativa_fuente.v1',
             'contexto_actor.organizacion_corporativa.revocar',
             'organizacion_corporativa.revocacion'),
            ('vec_contexto_actor.publicar_vinculo_corporativo_fuente.v1',
             'contexto_actor.vinculo_corporativo.publicar',
             'vinculo_corporativo.alta'),
            ('vec_contexto_actor.revocar_vinculo_corporativo_fuente.v1',
             'contexto_actor.vinculo_corporativo.revocar',
             'vinculo_corporativo.revocacion')
        ) AS c(audiencia, accion, tipo)
    LOOP
        v_caso := vec_autorizacion_atestada_v3.mutar_capacidad_a3_prueba(
            v_base, 'audiencia_consumo',
            vec_autorizacion_atestada_v3.texto_json_go(v_cruce.audiencia)
        );
        v_caso := vec_autorizacion_atestada_v3.mutar_capacidad_a3_prueba(
            v_caso, 'accion',
            vec_autorizacion_atestada_v3.texto_json_go(v_cruce.accion)
        );
        v_caso := vec_autorizacion_atestada_v3.mutar_capacidad_a3_prueba(
            v_caso, 'tipo_efecto',
            vec_autorizacion_atestada_v3.texto_json_go(v_cruce.tipo)
        );
        IF vec_autorizacion_atestada_v3
               .capacidad_fuente_corporativa_v1_canonica(v_caso)
           IS DISTINCT FROM v_caso THEN
            RAISE EXCEPTION USING ERRCODE = 'XX000',
                MESSAGE = 'A3: cruce nominal válido rechazado';
        END IF;
    END LOOP;

    v_caso := vec_autorizacion_atestada_v3.mutar_capacidad_a3_prueba(
        v_base, 'accion',
        vec_autorizacion_atestada_v3.texto_json_go(
            'contexto_actor.vinculo_corporativo.publicar'
        )
    );
    IF vec_autorizacion_atestada_v3
           .capacidad_fuente_corporativa_v1_canonica(v_caso) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = 'XX000',
            MESSAGE = 'A3: cruce nominal hostil aceptado';
    END IF;

    FOREACH v_caso IN ARRAY ARRAY[
        vec_autorizacion_atestada_v3.mutar_capacidad_a3_prueba(
            v_base, 'audiencia_consumo', '"audiencia.no.catalogada"'),
        vec_autorizacion_atestada_v3.mutar_capacidad_a3_prueba(
            v_base, 'tipo_efecto', '"vinculo_corporativo.alta"')
    ] LOOP
        IF vec_autorizacion_atestada_v3
               .capacidad_fuente_corporativa_v1_canonica(v_caso)
           IS NOT NULL THEN
            RAISE EXCEPTION USING ERRCODE = 'XX000',
                MESSAGE = 'A3: audiencia o tipo hostil aceptado';
        END IF;
    END LOOP;
END
$numeros_semantica_y_cruces$;

DO $campos_y_tiempo$
DECLARE
    v_base pg_catalog.bytea := vec_autorizacion_atestada_v3
        .vector_capacidad_a3_prueba();
    v_caso pg_catalog.bytea;
    v_clave pg_catalog.text;
BEGIN
    FOREACH v_clave IN ARRAY ARRAY[
        'huella_evento_fuente_sha256',
        'huella_manifiesto_fuente_sha256',
        'huella_sobre_cose_sign1_sha256',
        'huella_prueba_confianza_sha256', 'huella_efecto_sha256',
        'huella_gobierno_sha256', 'huella_configuracion_sha256',
        'huella_raiz_spki_sha256', 'nonce', 'mac_sha256'
    ] LOOP
        v_caso := vec_autorizacion_atestada_v3.mutar_capacidad_a3_prueba(
            v_base, v_clave,
            vec_autorizacion_atestada_v3.texto_json_go(
                pg_catalog.repeat('0', 64)
            )
        );
        IF vec_autorizacion_atestada_v3
               .capacidad_fuente_corporativa_v1_canonica(v_caso)
           IS NOT NULL THEN
            RAISE EXCEPTION USING ERRCODE = 'XX000',
                MESSAGE = 'A3: huella cero aceptada';
        END IF;
        v_caso := vec_autorizacion_atestada_v3.mutar_capacidad_a3_prueba(
            v_base, v_clave,
            vec_autorizacion_atestada_v3.texto_json_go(
                'A' || pg_catalog.repeat('1', 63)
            )
        );
        IF vec_autorizacion_atestada_v3
               .capacidad_fuente_corporativa_v1_canonica(v_caso)
           IS NOT NULL THEN
            RAISE EXCEPTION USING ERRCODE = 'XX000',
                MESSAGE = 'A3: huella con mayúscula aceptada';
        END IF;
    END LOOP;

    FOREACH v_caso IN ARRAY ARRAY[
        vec_autorizacion_atestada_v3.mutar_capacidad_a3_prueba(
            v_base, 'esquema',
            '"vec.contexto-actor.fuente-corporativa.capacidad.v2"'),
        vec_autorizacion_atestada_v3.mutar_capacidad_a3_prueba(
            v_base, 'version', '2'),
        vec_autorizacion_atestada_v3.mutar_capacidad_a3_prueba(
            v_base, 'fuente_ref', '"f0"'),
        vec_autorizacion_atestada_v3.mutar_capacidad_a3_prueba(
            v_base, 'evento_fuente_ref', '"evento con espacios"'),
        vec_autorizacion_atestada_v3.mutar_capacidad_a3_prueba(
            v_base, 'operacion_ref', '"oca_corta"'),
        vec_autorizacion_atestada_v3.mutar_capacidad_a3_prueba(
            v_base, 'efecto_ref', '"efecto no opaco"'),
        vec_autorizacion_atestada_v3.mutar_capacidad_a3_prueba(
            v_base, 'suite', '"VEC-AD-3-COSE-EDDSA-2"'),
        vec_autorizacion_atestada_v3.mutar_capacidad_a3_prueba(
            v_base, 'evento_fuente_emitido_en',
            '"2099-01-01T00:00:01.000000Z"'),
        vec_autorizacion_atestada_v3.mutar_capacidad_a3_prueba(
            v_base, 'emitida_en', '"2099-01-01T01:00:00.500000+01:00"'),
        vec_autorizacion_atestada_v3.mutar_capacidad_a3_prueba(
            v_base, 'expira_en', '"2099-01-01T00:00:00.500000Z"'),
        vec_autorizacion_atestada_v3.mutar_capacidad_a3_prueba(
            v_base, 'expira_en', '"2099-01-01T00:00:05.500001Z"')
    ] LOOP
        IF vec_autorizacion_atestada_v3
               .capacidad_fuente_corporativa_v1_canonica(v_caso)
           IS NOT NULL THEN
            RAISE EXCEPTION USING ERRCODE = 'XX000',
                MESSAGE = 'A3: semántica o cronología inválida aceptada';
        END IF;
    END LOOP;

    v_caso := vec_autorizacion_atestada_v3.mutar_capacidad_a3_prueba(
        v_base, 'expira_en', '"2099-01-01T00:00:05.500000Z"'
    );
    IF vec_autorizacion_atestada_v3
           .capacidad_fuente_corporativa_v1_canonica(v_caso)
       IS DISTINCT FROM v_caso THEN
        RAISE EXCEPTION USING ERRCODE = 'XX000',
            MESSAGE = 'A3: vigencia inclusiva de cinco segundos rechazada';
    END IF;
END
$campos_y_tiempo$;

DO $preimagen_y_hmac_hostiles$
DECLARE
    v_base pg_catalog.bytea := vec_autorizacion_atestada_v3
        .vector_capacidad_a3_prueba();
    v_clave pg_catalog.bytea := pg_catalog.sha256(
        pg_catalog.convert_to('material-hmac-sintetico-f0-v1', 'UTF8')
    );
    v_mac_alterado pg_catalog.bytea;
    v_nonce_alterado pg_catalog.bytea;
    v_clave_maxima pg_catalog.bytea := pg_catalog.decode(
        pg_catalog.repeat('ab', 4096), 'hex');
    v_clave_excesiva pg_catalog.bytea := pg_catalog.decode(
        pg_catalog.repeat('cd', 4097), 'hex');
    v_capacidad_maxima pg_catalog.bytea;
    v_capacidad_excesiva pg_catalog.bytea;
BEGIN
    v_mac_alterado := vec_autorizacion_atestada_v3.mutar_capacidad_a3_prueba(
        v_base, 'mac_sha256',
        vec_autorizacion_atestada_v3.texto_json_go(pg_catalog.repeat('9', 64))
    );
    v_nonce_alterado := vec_autorizacion_atestada_v3.mutar_capacidad_a3_prueba(
        v_base, 'nonce',
        vec_autorizacion_atestada_v3.texto_json_go(pg_catalog.repeat('9', 64))
    );
    v_capacidad_maxima := vec_autorizacion_atestada_v3
        .mutar_capacidad_a3_prueba(
            v_base, 'mac_sha256', vec_autorizacion_atestada_v3.texto_json_go(
                pg_catalog.encode(public.hmac(
                    vec_autorizacion_atestada_v3
                        .preimagen_mac_fuente_corporativa_v1(v_base),
                    v_clave_maxima, 'sha256'), 'hex')));
    v_capacidad_excesiva := vec_autorizacion_atestada_v3
        .mutar_capacidad_a3_prueba(
            v_base, 'mac_sha256', vec_autorizacion_atestada_v3.texto_json_go(
                pg_catalog.encode(public.hmac(
                    vec_autorizacion_atestada_v3
                        .preimagen_mac_fuente_corporativa_v1(v_base),
                    v_clave_excesiva, 'sha256'), 'hex')));
    IF vec_autorizacion_atestada_v3
           .capacidad_fuente_corporativa_v1_canonica(v_mac_alterado)
       IS DISTINCT FROM v_mac_alterado
       OR vec_autorizacion_atestada_v3
           .preimagen_mac_fuente_corporativa_v1(v_mac_alterado)
          IS DISTINCT FROM vec_autorizacion_atestada_v3
           .preimagen_mac_fuente_corporativa_v1(v_base)
       OR vec_autorizacion_atestada_v3
           .mac_capacidad_fuente_corporativa_v1_valido(
               v_mac_alterado, v_clave
           ) IS NOT FALSE
       OR vec_autorizacion_atestada_v3
           .mac_capacidad_fuente_corporativa_v1_valido(
               v_nonce_alterado, v_clave
           ) IS NOT FALSE
       OR vec_autorizacion_atestada_v3
           .mac_capacidad_fuente_corporativa_v1_valido(
               v_base, pg_catalog.sha256(
                   pg_catalog.convert_to('otra-clave-sintetica', 'UTF8')
               )
           ) IS NOT FALSE
       OR vec_autorizacion_atestada_v3
           .mac_capacidad_fuente_corporativa_v1_valido(v_base, NULL)
          IS NOT FALSE
       OR vec_autorizacion_atestada_v3
           .mac_capacidad_fuente_corporativa_v1_valido(
               v_base, pg_catalog.decode(pg_catalog.repeat('00', 32), 'hex')
           ) IS NOT FALSE
       OR vec_autorizacion_atestada_v3
           .mac_capacidad_fuente_corporativa_v1_valido(
               v_base, pg_catalog.decode(pg_catalog.repeat('01', 31), 'hex')
           ) IS NOT FALSE
       OR vec_autorizacion_atestada_v3
           .mac_capacidad_fuente_corporativa_v1_valido(
               v_capacidad_maxima, v_clave_maxima) IS NOT TRUE
       OR vec_autorizacion_atestada_v3
           .mac_capacidad_fuente_corporativa_v1_valido(
               v_capacidad_excesiva, v_clave_excesiva) IS NOT FALSE
       OR vec_autorizacion_atestada_v3
           .preimagen_mac_fuente_corporativa_v1(
               pg_catalog.convert_to('{}', 'UTF8')
           ) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = 'XX000',
            MESSAGE = 'A3: preimagen o verificación HMAC abierta';
    END IF;
END
$preimagen_y_hmac_hostiles$;

DROP FUNCTION vec_autorizacion_atestada_v3.mutar_capacidad_a3_prueba(
    pg_catalog.bytea, pg_catalog.text, pg_catalog.text
);
DROP FUNCTION vec_autorizacion_atestada_v3.vector_capacidad_a3_prueba();
