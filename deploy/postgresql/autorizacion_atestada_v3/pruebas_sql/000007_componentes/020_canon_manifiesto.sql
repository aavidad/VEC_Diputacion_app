-- Matriz adversarial del canon de manifiesto F0-A2.

CREATE FUNCTION
vec_autorizacion_atestada_v3.acreditar_forma_canon_manifiesto_a2_prueba()
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
    ), candidata AS (
        SELECT p.*
          FROM pg_catalog.pg_proc AS p
          JOIN pg_catalog.pg_namespace AS n ON n.oid = p.pronamespace
          JOIN propietario AS o ON o.oid = p.proowner
         WHERE n.nspname = 'vec_autorizacion_atestada_v3'
           AND p.proname = 'manifiesto_fuente_corporativa_v1_canonico'
           AND pg_catalog.oidvectortypes(p.proargtypes) = 'bytea'
           AND p.prokind = 'f'
           AND pg_catalog.format_type(p.prorettype, NULL) = 'bytea'
           AND p.proargnames = ARRAY['p_manifiesto']::pg_catalog.text[]
           AND p.proallargtypes IS NULL
           AND p.proargmodes IS NULL
           AND p.pronargs = 1
           AND p.pronargdefaults = 0
           AND p.proargdefaults IS NULL
           AND p.provariadic = 0
           AND NOT p.proretset
           AND p.provolatile = 'i'
           AND NOT p.proisstrict
           AND NOT p.prosecdef
           AND NOT p.proleakproof
           AND p.proparallel = 'u'
           AND p.proconfig = ARRAY['search_path=pg_catalog']
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
       AND (SELECT pg_catalog.count(*) FROM candidata) = 1
       AND (
           SELECT pg_catalog.count(*)
             FROM pg_catalog.pg_proc AS p
             JOIN pg_catalog.pg_namespace AS n ON n.oid = p.pronamespace
            WHERE n.nspname = 'vec_autorizacion_atestada_v3'
              AND p.proname =
                  'manifiesto_fuente_corporativa_v1_canonico'
       ) = 1
$funcion$;

REVOKE ALL ON FUNCTION
    vec_autorizacion_atestada_v3.acreditar_forma_canon_manifiesto_a2_prueba()
    FROM PUBLIC;

DO $forma_inicial$
BEGIN
    IF vec_autorizacion_atestada_v3
           .acreditar_forma_canon_manifiesto_a2_prueba() IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = 'XX000',
            MESSAGE = 'A2: forma inicial del canon inválida';
    END IF;
END
$forma_inicial$;

GRANT EXECUTE ON FUNCTION
    vec_autorizacion_atestada_v3
        .manifiesto_fuente_corporativa_v1_canonico(pg_catalog.bytea)
    TO vec_autorizacion_atestada_v3_migrador;

DO $acl_hostil$
BEGIN
    IF vec_autorizacion_atestada_v3
           .acreditar_forma_canon_manifiesto_a2_prueba() IS NOT FALSE THEN
        RAISE EXCEPTION USING ERRCODE = 'XX000',
            MESSAGE = 'A2: ACL adicional no detectada';
    END IF;
END
$acl_hostil$;

REVOKE ALL ON FUNCTION
    vec_autorizacion_atestada_v3
        .manifiesto_fuente_corporativa_v1_canonico(pg_catalog.bytea)
    FROM vec_autorizacion_atestada_v3_migrador;

CREATE FUNCTION
vec_autorizacion_atestada_v3.manifiesto_fuente_corporativa_v1_canonico(
    p_manifiesto pg_catalog.text
)
RETURNS pg_catalog.bytea
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog
AS $sobrecarga$
    SELECT NULL::pg_catalog.bytea
$sobrecarga$;

REVOKE ALL ON FUNCTION
    vec_autorizacion_atestada_v3
        .manifiesto_fuente_corporativa_v1_canonico(pg_catalog.text)
    FROM PUBLIC;

DO $sobrecarga_hostil$
BEGIN
    IF vec_autorizacion_atestada_v3
           .acreditar_forma_canon_manifiesto_a2_prueba() IS NOT FALSE THEN
        RAISE EXCEPTION USING ERRCODE = 'XX000',
            MESSAGE = 'A2: sobrecarga homónima no detectada';
    END IF;
END
$sobrecarga_hostil$;

DROP FUNCTION vec_autorizacion_atestada_v3
    .manifiesto_fuente_corporativa_v1_canonico(pg_catalog.text);

ALTER FUNCTION vec_autorizacion_atestada_v3
    .manifiesto_fuente_corporativa_v1_canonico(pg_catalog.bytea)
    PARALLEL SAFE;

DO $paralelismo_hostil$
BEGIN
    IF vec_autorizacion_atestada_v3
           .acreditar_forma_canon_manifiesto_a2_prueba() IS NOT FALSE THEN
        RAISE EXCEPTION USING ERRCODE = 'XX000',
            MESSAGE = 'A2: paralelismo hostil no detectado';
    END IF;
END
$paralelismo_hostil$;

ALTER FUNCTION vec_autorizacion_atestada_v3
    .manifiesto_fuente_corporativa_v1_canonico(pg_catalog.bytea)
    PARALLEL UNSAFE;

ALTER FUNCTION vec_autorizacion_atestada_v3
    .manifiesto_fuente_corporativa_v1_canonico(pg_catalog.bytea)
    SECURITY DEFINER;

DO $seguridad_hostil$
BEGIN
    IF vec_autorizacion_atestada_v3
           .acreditar_forma_canon_manifiesto_a2_prueba() IS NOT FALSE THEN
        RAISE EXCEPTION USING ERRCODE = 'XX000',
            MESSAGE = 'A2: SECURITY DEFINER hostil no detectado';
    END IF;
END
$seguridad_hostil$;

ALTER FUNCTION vec_autorizacion_atestada_v3
    .manifiesto_fuente_corporativa_v1_canonico(pg_catalog.bytea)
    SECURITY INVOKER;

DO $forma_restaurada$
BEGIN
    IF vec_autorizacion_atestada_v3
           .acreditar_forma_canon_manifiesto_a2_prueba() IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = 'XX000',
            MESSAGE = 'A2: forma exacta no restaurada';
    END IF;
END
$forma_restaurada$;

DROP FUNCTION
    vec_autorizacion_atestada_v3.acreditar_forma_canon_manifiesto_a2_prueba();

-- El vector se construye desde trece valores tipados con el serializador JSON
-- de PostgreSQL. El hash esperado procede del oráculo Go V0 y no se duplica
-- aquí el literal canónico completo.
CREATE FUNCTION
vec_autorizacion_atestada_v3.vector_manifiesto_a2_prueba(
    p_fuente_ref pg_catalog.text DEFAULT 'fuente:sintetica:f0:v1',
    p_fuente_version pg_catalog.numeric DEFAULT 1,
    p_evento_fuente_ref pg_catalog.text DEFAULT 'evento:sintetico:f0:0001',
    p_huella_evento pg_catalog.text DEFAULT pg_catalog.repeat('1', 64),
    p_evento_emitido_en pg_catalog.text DEFAULT
        '2099-01-01T00:00:00.000000Z',
    p_audiencia pg_catalog.text DEFAULT
        'vec_contexto_actor.publicar_organizacion_corporativa_fuente.v1',
    p_accion pg_catalog.text DEFAULT
        'contexto_actor.organizacion_corporativa.publicar',
    p_tipo_efecto pg_catalog.text DEFAULT 'organizacion_corporativa.alta',
    p_operacion_ref pg_catalog.text DEFAULT 'oca_vector_f0_0000000000000001',
    p_efecto_ref pg_catalog.text DEFAULT 'efecto:sintetico:f0:0001',
    p_huella_efecto pg_catalog.text DEFAULT pg_catalog.repeat('4', 64)
)
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
                    'vec.contexto-actor.fuente-corporativa.manifiesto.v1',
                    'version', 1,
                    'fuente_ref', p_fuente_ref,
                    'fuente_version', p_fuente_version,
                    'evento_fuente_ref', p_evento_fuente_ref,
                    'huella_evento_fuente_sha256', p_huella_evento,
                    'evento_fuente_emitido_en', p_evento_emitido_en,
                    'audiencia_consumo', p_audiencia,
                    'accion', p_accion,
                    'tipo_efecto', p_tipo_efecto,
                    'operacion_ref', p_operacion_ref,
                    'efecto_ref', p_efecto_ref,
                    'huella_efecto_sha256', p_huella_efecto
                )::pg_catalog.text,
                '" : ', '":'
            ),
            ', "', ',"'
        ),
        'UTF8'
    )
$vector$;

REVOKE ALL ON FUNCTION
    vec_autorizacion_atestada_v3.vector_manifiesto_a2_prueba(
        pg_catalog.text, pg_catalog.numeric, pg_catalog.text,
        pg_catalog.text, pg_catalog.text, pg_catalog.text,
        pg_catalog.text, pg_catalog.text, pg_catalog.text,
        pg_catalog.text, pg_catalog.text
    ) FROM PUBLIC;

DO $vector_dorado$
DECLARE
    v_vector pg_catalog.bytea := vec_autorizacion_atestada_v3
        .vector_manifiesto_a2_prueba();
BEGIN
    IF pg_catalog.octet_length(v_vector) <> 705
       OR pg_catalog.encode(pg_catalog.sha256(v_vector), 'hex') <>
          'f16cab3533e7a5b4126ae1bddf8afbc989ce564330cf9703d1429ceb21678325'
       OR vec_autorizacion_atestada_v3
              .manifiesto_fuente_corporativa_v1_canonico(v_vector)
          IS DISTINCT FROM v_vector THEN
        RAISE EXCEPTION USING ERRCODE = 'XX000',
            MESSAGE = 'A2: el vector SQL no coincide byte a byte con V0';
    END IF;
END
$vector_dorado$;

DO $cuatro_cruces$
DECLARE
    v_vector pg_catalog.bytea;
    v_fila record;
BEGIN
    FOR v_fila IN
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
        v_vector := vec_autorizacion_atestada_v3.vector_manifiesto_a2_prueba(
            p_audiencia => v_fila.audiencia,
            p_accion => v_fila.accion,
            p_tipo_efecto => v_fila.tipo
        );
        IF vec_autorizacion_atestada_v3
               .manifiesto_fuente_corporativa_v1_canonico(v_vector)
           IS DISTINCT FROM v_vector THEN
            RAISE EXCEPTION USING ERRCODE = 'XX000',
                MESSAGE = 'A2: cruce nominal válido rechazado';
        END IF;
    END LOOP;

    v_vector := vec_autorizacion_atestada_v3.vector_manifiesto_a2_prueba(
        p_audiencia =>
            'vec_contexto_actor.publicar_vinculo_corporativo_fuente.v1',
        p_accion => 'contexto_actor.organizacion_corporativa.publicar',
        p_tipo_efecto => 'vinculo_corporativo.alta'
    );
    IF vec_autorizacion_atestada_v3
           .manifiesto_fuente_corporativa_v1_canonico(v_vector) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = 'XX000',
            MESSAGE = 'A2: cruce nominal hostil aceptado';
    END IF;
END
$cuatro_cruces$;

DO $forma_y_tipos$
DECLARE
    v_base pg_catalog.text := pg_catalog.convert_from(
        vec_autorizacion_atestada_v3.vector_manifiesto_a2_prueba(), 'UTF8'
    );
    v_caso pg_catalog.text;
BEGIN
    FOREACH v_caso IN ARRAY ARRAY[
        pg_catalog.replace(v_base,
            '{"esquema":', '{ "esquema":'),
        pg_catalog.replace(v_base,
            '"version":1,"fuente_ref":"fuente:sintetica:f0:v1"',
            '"fuente_ref":"fuente:sintetica:f0:v1","version":1'),
        pg_catalog.replace(v_base,
            '{"esquema":',
            '{"esquema":"vec.contexto-actor.fuente-corporativa.manifiesto.v1",' ||
            '"esquema":'),
        pg_catalog.replace(v_base,
            'vec.contexto-actor.fuente-corporativa.manifiesto.v1',
            'vec.contexto-actor.fuente-corporativa.manifiesto.v2'),
        pg_catalog.replace(v_base,
            '"version":1', '"version":2'),
        pg_catalog.replace(v_base,
            '"version":1', '"version":"1"'),
        pg_catalog.replace(v_base,
            '"fuente_version":1', '"fuente_version":1.0'),
        pg_catalog.replace(v_base,
            '"fuente_version":1', '"fuente_version":1e0'),
        pg_catalog.replace(v_base,
            '"fuente_version":1', '"fuente_version":null'),
        pg_catalog.replace(v_base,
            '"fuente_ref":"fuente:sintetica:f0:v1"',
            '"fuente_ref":{"valor":"fuente:sintetica:f0:v1"}'),
        pg_catalog.replace(v_base,
            '"evento_fuente_ref":"evento:sintetico:f0:0001"',
            '"evento_fuente_ref":["evento:sintetico:f0:0001"]'),
        pg_catalog.replace(v_base,
            '"huella_evento_fuente_sha256":"' || pg_catalog.repeat('1', 64) || '"',
            '"huella_evento_fuente_sha256":1'),
        pg_catalog.replace(v_base,
            '"evento_fuente_emitido_en":"2099-01-01T00:00:00.000000Z"',
            '"evento_fuente_emitido_en":true'),
        pg_catalog.replace(v_base,
            '"audiencia_consumo":' ||
            '"vec_contexto_actor.publicar_organizacion_corporativa_fuente.v1"',
            '"audiencia_consumo":{"valor":1}'),
        pg_catalog.replace(v_base,
            '"accion":"contexto_actor.organizacion_corporativa.publicar"',
            '"accion":[]'),
        pg_catalog.replace(v_base,
            '"tipo_efecto":"organizacion_corporativa.alta"',
            '"tipo_efecto":null'),
        pg_catalog.replace(v_base,
            '"operacion_ref":"oca_vector_f0_0000000000000001"',
            '"operacion_ref":1'),
        pg_catalog.replace(v_base,
            '"efecto_ref":"efecto:sintetico:f0:0001"',
            '"efecto_ref":false'),
        pg_catalog.replace(v_base,
            '"huella_efecto_sha256":"' || pg_catalog.repeat('4', 64) || '"',
            '"huella_efecto_sha256":{}'),
        pg_catalog.left(v_base, pg_catalog.length(v_base) - 1) ||
            ',"sobrante":"x"}',
        pg_catalog.replace(v_base,
            ',"efecto_ref":"efecto:sintetico:f0:0001"', ''),
        pg_catalog.replace(v_base,
            '{"esquema":',
            '{"esquema":"vec.contexto-actor.fuente-corporativa.manifiesto.v1",' ||
            '"esquema_descartado":'),
        pg_catalog.replace(v_base,
            '"fuente_ref":"fuente:sintetica:f0:v1"',
            '"fuente_ref":"\u0066uente:sintetica:f0:v1"'),
        pg_catalog.replace(v_base,
            '"fuente_ref":"fuente:sintetica:f0:v1"',
            '"fuente_ref":"fuente\u003asintetica:f0:v1"'),
        pg_catalog.replace(v_base,
            '"fuente_ref":"fuente:sintetica:f0:v1"',
            '"fuente_ref":"\u0000fuente:sintetica:f0:v1"'),
        pg_catalog.replace(v_base,
            '"fuente_ref":"fuente:sintetica:f0:v1"',
            '"fuente_ref":"\ud800fuente:sintetica:f0:v1"'),
        v_base || 'x',
        '[' || v_base || ']'
    ] LOOP
        IF vec_autorizacion_atestada_v3
               .manifiesto_fuente_corporativa_v1_canonico(
                   pg_catalog.convert_to(v_caso, 'UTF8')
               ) IS NOT NULL THEN
            RAISE EXCEPTION USING ERRCODE = 'XX000',
                MESSAGE = 'A2: forma, tipo o escape no canónico aceptado';
        END IF;
    END LOOP;

    IF vec_autorizacion_atestada_v3
           .manifiesto_fuente_corporativa_v1_canonico(NULL) IS NOT NULL
       OR vec_autorizacion_atestada_v3
           .manifiesto_fuente_corporativa_v1_canonico(
               pg_catalog.convert_to(pg_catalog.repeat('x', 127), 'UTF8')
           ) IS NOT NULL
       OR vec_autorizacion_atestada_v3
           .manifiesto_fuente_corporativa_v1_canonico(
               pg_catalog.convert_to(pg_catalog.repeat('x', 16385), 'UTF8')
           ) IS NOT NULL
       OR vec_autorizacion_atestada_v3
           .manifiesto_fuente_corporativa_v1_canonico(
               pg_catalog.decode('efbbbf', 'hex') ||
               pg_catalog.convert_to(v_base, 'UTF8')
           ) IS NOT NULL
       OR vec_autorizacion_atestada_v3
           .manifiesto_fuente_corporativa_v1_canonico(
               pg_catalog.decode('c328', 'hex')
           ) IS NOT NULL
       OR vec_autorizacion_atestada_v3
           .manifiesto_fuente_corporativa_v1_canonico(
               pg_catalog.convert_to(
                   pg_catalog.left(v_base, 180), 'UTF8'
               ) || pg_catalog.decode('00', 'hex') ||
               pg_catalog.convert_to(
                   pg_catalog.substr(v_base, 181), 'UTF8'
               )
           ) IS NOT NULL
       OR vec_autorizacion_atestada_v3
           .manifiesto_fuente_corporativa_v1_canonico(
               pg_catalog.convert_to('{"esquema":', 'UTF8')
           ) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = 'XX000',
            MESSAGE = 'A2: límite, UTF-8 o JSON malformado aceptado';
    END IF;
END
$forma_y_tipos$;

DO $semantica$
DECLARE
    v_caso pg_catalog.bytea;
BEGIN
    FOREACH v_caso IN ARRAY ARRAY[
        vec_autorizacion_atestada_v3.vector_manifiesto_a2_prueba(
            p_fuente_ref => 'f0'),
        vec_autorizacion_atestada_v3.vector_manifiesto_a2_prueba(
            p_fuente_ref => pg_catalog.repeat('f', 161)),
        vec_autorizacion_atestada_v3.vector_manifiesto_a2_prueba(
            p_fuente_version => 0),
        vec_autorizacion_atestada_v3.vector_manifiesto_a2_prueba(
            p_fuente_version => 9007199254740992),
        vec_autorizacion_atestada_v3.vector_manifiesto_a2_prueba(
            p_evento_fuente_ref => 'evento con espacios'),
        vec_autorizacion_atestada_v3.vector_manifiesto_a2_prueba(
            p_huella_evento => pg_catalog.repeat('0', 64)),
        vec_autorizacion_atestada_v3.vector_manifiesto_a2_prueba(
            p_huella_evento => 'A' || pg_catalog.repeat('1', 63)),
        vec_autorizacion_atestada_v3.vector_manifiesto_a2_prueba(
            p_evento_emitido_en => '2099-01-01T00:00:00Z'),
        vec_autorizacion_atestada_v3.vector_manifiesto_a2_prueba(
            p_audiencia => 'audiencia.no.catalogada'),
        vec_autorizacion_atestada_v3.vector_manifiesto_a2_prueba(
            p_operacion_ref => 'oca_corta'),
        vec_autorizacion_atestada_v3.vector_manifiesto_a2_prueba(
            p_efecto_ref => 'efecto no opaco'),
        vec_autorizacion_atestada_v3.vector_manifiesto_a2_prueba(
            p_huella_efecto => pg_catalog.repeat('4', 63))
    ] LOOP
        IF vec_autorizacion_atestada_v3
               .manifiesto_fuente_corporativa_v1_canonico(v_caso)
           IS NOT NULL THEN
            RAISE EXCEPTION USING ERRCODE = 'XX000',
                MESSAGE = 'A2: semántica inválida aceptada';
        END IF;
    END LOOP;

    v_caso := vec_autorizacion_atestada_v3.vector_manifiesto_a2_prueba(
        p_fuente_ref => 'f01',
        p_fuente_version => 9007199254740991,
        p_evento_fuente_ref => pg_catalog.repeat('e', 160),
        p_operacion_ref => 'oca_' || pg_catalog.repeat('o', 128),
        p_efecto_ref => pg_catalog.repeat('x', 160)
    );
    IF vec_autorizacion_atestada_v3
           .manifiesto_fuente_corporativa_v1_canonico(v_caso)
       IS DISTINCT FROM v_caso THEN
        RAISE EXCEPTION USING ERRCODE = 'XX000',
            MESSAGE = 'A2: límites semánticos inclusivos rechazados';
    END IF;

    v_caso := vec_autorizacion_atestada_v3.vector_manifiesto_a2_prueba(
        p_fuente_ref => 'f01',
        p_evento_fuente_ref => 'e01',
        p_operacion_ref => 'oca_' || pg_catalog.repeat('o', 24),
        p_efecto_ref => 'x01'
    );
    IF vec_autorizacion_atestada_v3
           .manifiesto_fuente_corporativa_v1_canonico(v_caso)
       IS DISTINCT FROM v_caso THEN
        RAISE EXCEPTION USING ERRCODE = 'XX000',
            MESSAGE = 'A2: mínimos semánticos inclusivos rechazados';
    END IF;

    v_caso := vec_autorizacion_atestada_v3.vector_manifiesto_a2_prueba(
        p_fuente_ref => 'fuente/sintetica/f0/v1'
    );
    IF vec_autorizacion_atestada_v3
           .manifiesto_fuente_corporativa_v1_canonico(v_caso)
       IS DISTINCT FROM v_caso
       OR vec_autorizacion_atestada_v3
           .manifiesto_fuente_corporativa_v1_canonico(
               pg_catalog.convert_to(
                   pg_catalog.replace(
                       pg_catalog.convert_from(v_caso, 'UTF8'),
                       'fuente/sintetica/f0/v1',
                       'fuente\/sintetica\/f0\/v1'
                   ),
                   'UTF8'
               )
           ) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = 'XX000',
            MESSAGE = 'A2: escape opcional de barra no cerrado';
    END IF;
END
$semantica$;

DROP FUNCTION vec_autorizacion_atestada_v3.vector_manifiesto_a2_prueba(
    pg_catalog.text, pg_catalog.numeric, pg_catalog.text,
    pg_catalog.text, pg_catalog.text, pg_catalog.text,
    pg_catalog.text, pg_catalog.text, pg_catalog.text,
    pg_catalog.text, pg_catalog.text
);
