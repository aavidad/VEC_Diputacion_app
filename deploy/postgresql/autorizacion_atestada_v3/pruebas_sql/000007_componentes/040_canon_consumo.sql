-- Matriz adversarial del canon y la huella de consumo F0-A4.

CREATE FUNCTION
vec_autorizacion_atestada_v3.acreditar_forma_canon_consumo_a4_prueba()
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
           AND p.proname =
               'canon_y_huella_consumo_fuente_corporativa_v1'
           AND pg_catalog.oidvectortypes(p.proargtypes) =
               'bytea, timestamp with time zone'
           AND p.prokind = 'f'
           AND pg_catalog.format_type(p.prorettype, NULL) = 'record'
           AND p.proargnames = ARRAY[
               'p_capacidad_canonica', 'p_consumida_en',
               'consumo_canonico', 'consumo_huella_sha256'
           ]::pg_catalog.text[]
           AND p.proallargtypes = ARRAY[
               'bytea'::pg_catalog.regtype::pg_catalog.oid,
               'timestamptz'::pg_catalog.regtype::pg_catalog.oid,
               'bytea'::pg_catalog.regtype::pg_catalog.oid,
               'text'::pg_catalog.regtype::pg_catalog.oid
           ]
           AND p.proargmodes = ARRAY['i', 'i', 't', 't']::"char"[]
           AND p.pronargs = 2
           AND p.pronargdefaults = 0
           AND p.proargdefaults IS NULL
           AND p.provariadic = 0
           AND p.proretset
           AND p.provolatile = 'i'
           AND NOT p.proisstrict
           AND NOT p.prosecdef
           AND NOT p.proleakproof
           AND p.proparallel = 'u'
           AND p.procost = 100
           AND p.prorows = 1000
           AND p.prosupport = 0
           AND p.protrftypes IS NULL
           AND p.probin IS NULL
           AND p.prosqlbody IS NULL
           AND p.proconfig = ARRAY['search_path=pg_catalog']
           AND pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
                   p.prosrc, 'UTF8')), 'hex') =
               'acd9a07ebdb7e0f9b7cbd4c9f67e93d009afcf1e81aebb2b014e57bbcfc2ad6e'
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
                  'canon_y_huella_consumo_fuente_corporativa_v1'
       ) = 1
$funcion$;

REVOKE ALL ON FUNCTION
    vec_autorizacion_atestada_v3.acreditar_forma_canon_consumo_a4_prueba()
    FROM PUBLIC;

DO $forma_inicial$
BEGIN
    IF vec_autorizacion_atestada_v3
           .acreditar_forma_canon_consumo_a4_prueba() IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = 'XX000',
            MESSAGE = 'A4: forma inicial inválida';
    END IF;
END
$forma_inicial$;

GRANT EXECUTE ON FUNCTION
    vec_autorizacion_atestada_v3
        .canon_y_huella_consumo_fuente_corporativa_v1(
            pg_catalog.bytea, pg_catalog.timestamptz
        ) TO vec_autorizacion_atestada_v3_migrador;

DO $acl_hostil$
BEGIN
    IF vec_autorizacion_atestada_v3
           .acreditar_forma_canon_consumo_a4_prueba() IS NOT FALSE THEN
        RAISE EXCEPTION USING ERRCODE = 'XX000',
            MESSAGE = 'A4: ACL adicional no detectada';
    END IF;
END
$acl_hostil$;

REVOKE ALL ON FUNCTION
    vec_autorizacion_atestada_v3
        .canon_y_huella_consumo_fuente_corporativa_v1(
            pg_catalog.bytea, pg_catalog.timestamptz
        ) FROM vec_autorizacion_atestada_v3_migrador;

CREATE FUNCTION
vec_autorizacion_atestada_v3.canon_y_huella_consumo_fuente_corporativa_v1(
    p_capacidad_canonica pg_catalog.text,
    p_consumida_en pg_catalog.timestamptz
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
        .canon_y_huella_consumo_fuente_corporativa_v1(
            pg_catalog.text, pg_catalog.timestamptz
        ) FROM PUBLIC;

DO $sobrecarga_hostil$
BEGIN
    IF vec_autorizacion_atestada_v3
           .acreditar_forma_canon_consumo_a4_prueba() IS NOT FALSE THEN
        RAISE EXCEPTION USING ERRCODE = 'XX000',
            MESSAGE = 'A4: sobrecarga homónima no detectada';
    END IF;
END
$sobrecarga_hostil$;

DROP FUNCTION vec_autorizacion_atestada_v3
    .canon_y_huella_consumo_fuente_corporativa_v1(
        pg_catalog.text, pg_catalog.timestamptz
    );

ALTER FUNCTION vec_autorizacion_atestada_v3
    .canon_y_huella_consumo_fuente_corporativa_v1(
        pg_catalog.bytea, pg_catalog.timestamptz) COST 777;
DO $coste_hostil$
BEGIN
    IF vec_autorizacion_atestada_v3
           .acreditar_forma_canon_consumo_a4_prueba() IS NOT FALSE THEN
        RAISE EXCEPTION USING ERRCODE = 'XX000',
            MESSAGE = 'A4: coste hostil no detectado';
    END IF;
END
$coste_hostil$;
ALTER FUNCTION vec_autorizacion_atestada_v3
    .canon_y_huella_consumo_fuente_corporativa_v1(
        pg_catalog.bytea, pg_catalog.timestamptz) COST 100;

DO $cuerpo_hostil$
DECLARE
    v_original pg_catalog.text := pg_catalog.pg_get_functiondef(
        'vec_autorizacion_atestada_v3.canon_y_huella_consumo_fuente_corporativa_v1(bytea,timestamptz)'::pg_catalog.regprocedure);
    v_mutante pg_catalog.text;
BEGIN
    v_mutante := pg_catalog.replace(v_original,
        'vec.contexto-actor.fuente-corporativa.consumo.v1',
        'vec.contexto-actor.fuente-corporativa.consumo.v2');
    IF v_mutante = v_original THEN
        RAISE EXCEPTION USING ERRCODE = 'XX000',
            MESSAGE = 'A4: mutante de cuerpo no construido';
    END IF;
    EXECUTE v_mutante;
    IF vec_autorizacion_atestada_v3
           .acreditar_forma_canon_consumo_a4_prueba() IS NOT FALSE THEN
        RAISE EXCEPTION USING ERRCODE = 'XX000',
            MESSAGE = 'A4: cuerpo hostil no detectado';
    END IF;
    EXECUTE v_original;
END
$cuerpo_hostil$;

DO $forma_restaurada$
BEGIN
    IF vec_autorizacion_atestada_v3
           .acreditar_forma_canon_consumo_a4_prueba() IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = 'XX000',
            MESSAGE = 'A4: forma exacta no restaurada';
    END IF;
END
$forma_restaurada$;

DROP FUNCTION
    vec_autorizacion_atestada_v3.acreditar_forma_canon_consumo_a4_prueba();

-- Vector independiente V0: los campos tipados siguen el mismo orden de la
-- estructura Go. Las huellas V0 impiden que constructor y producto se
-- aprueben comparándose únicamente entre sí.
CREATE FUNCTION vec_autorizacion_atestada_v3.vector_capacidad_a4_prueba()
RETURNS pg_catalog.bytea
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog
AS $vector$
    WITH campos AS (
        SELECT
            'vec.contexto-actor.fuente-corporativa.capacidad.v1'
                ::pg_catalog.text AS esquema,
            1::pg_catalog.numeric AS version,
            'fuente:sintetica:f0:v1'::pg_catalog.text AS fuente_ref,
            1::pg_catalog.numeric AS fuente_version,
            'evento:sintetico:f0:0001'::pg_catalog.text AS evento_fuente_ref,
            pg_catalog.repeat('1', 64)::pg_catalog.text
                AS huella_evento_fuente,
            '2099-01-01T00:00:00.000000Z'::pg_catalog.text
                AS evento_fuente_emitido_en,
            'f16cab3533e7a5b4126ae1bddf8afbc989ce564330cf9703d1429ceb21678325'
                ::pg_catalog.text AS huella_manifiesto,
            pg_catalog.repeat('2', 64)::pg_catalog.text AS huella_sobre,
            pg_catalog.repeat('3', 64)::pg_catalog.text AS huella_prueba,
            'vec_contexto_actor.publicar_organizacion_corporativa_fuente.v1'
                ::pg_catalog.text AS audiencia,
            'contexto_actor.organizacion_corporativa.publicar'
                ::pg_catalog.text AS accion,
            'organizacion_corporativa.alta'::pg_catalog.text AS tipo_efecto,
            'oca_vector_f0_0000000000000001'::pg_catalog.text
                AS operacion_ref,
            'efecto:sintetico:f0:0001'::pg_catalog.text AS efecto_ref,
            pg_catalog.repeat('4', 64)::pg_catalog.text AS huella_efecto,
            'clave:sintetica:f0:0001'::pg_catalog.text AS clave_id,
            2::pg_catalog.numeric AS clave_version,
            7::pg_catalog.numeric AS revision_gobierno,
            pg_catalog.repeat('5', 64)::pg_catalog.text AS huella_gobierno,
            'emisor:sintetico:f0:0001'::pg_catalog.text AS emisor_id,
            'configuracion:sintetica:f0:0001'::pg_catalog.text
                AS configuracion_revision,
            11::pg_catalog.numeric AS configuracion_secuencia,
            pg_catalog.repeat('6', 64)::pg_catalog.text
                AS huella_configuracion,
            'raiz:sintetica:f0:0001'::pg_catalog.text AS raiz_clave_id,
            3::pg_catalog.numeric AS raiz_version,
            pg_catalog.repeat('7', 64)::pg_catalog.text AS huella_raiz,
            'vec-diputacion/pruebas/f0/fuente-corporativa'
                ::pg_catalog.text AS audiencia_despliegue,
            'VEC-AD-3-COSE-EDDSA-1'::pg_catalog.text AS suite,
            pg_catalog.repeat('8', 64)::pg_catalog.text AS nonce,
            '2099-01-01T00:00:00.500000Z'::pg_catalog.text AS emitida_en,
            '2099-01-01T00:00:04.500000Z'::pg_catalog.text AS expira_en,
            '16279292d2792d56230b30787745231a11a3fad585a9e476d65dc5d3723325b0'
                ::pg_catalog.text AS mac
    )
    SELECT pg_catalog.convert_to(
        pg_catalog.replace(
            pg_catalog.replace(
                pg_catalog.json_build_object(
                    'esquema', c.esquema,
                    'version', c.version,
                    'fuente_ref', c.fuente_ref,
                    'fuente_version', c.fuente_version,
                    'evento_fuente_ref', c.evento_fuente_ref,
                    'huella_evento_fuente_sha256', c.huella_evento_fuente,
                    'evento_fuente_emitido_en', c.evento_fuente_emitido_en,
                    'huella_manifiesto_fuente_sha256', c.huella_manifiesto,
                    'huella_sobre_cose_sign1_sha256', c.huella_sobre,
                    'huella_prueba_confianza_sha256', c.huella_prueba,
                    'audiencia_consumo', c.audiencia,
                    'accion', c.accion,
                    'tipo_efecto', c.tipo_efecto,
                    'operacion_ref', c.operacion_ref,
                    'efecto_ref', c.efecto_ref,
                    'huella_efecto_sha256', c.huella_efecto,
                    'clave_id', c.clave_id,
                    'clave_version', c.clave_version,
                    'revision_gobierno', c.revision_gobierno,
                    'huella_gobierno_sha256', c.huella_gobierno,
                    'emisor_id', c.emisor_id,
                    'configuracion_revision', c.configuracion_revision,
                    'configuracion_secuencia', c.configuracion_secuencia,
                    'huella_configuracion_sha256', c.huella_configuracion,
                    'raiz_clave_id', c.raiz_clave_id,
                    'raiz_version', c.raiz_version,
                    'huella_raiz_spki_sha256', c.huella_raiz,
                    'audiencia_despliegue', c.audiencia_despliegue,
                    'suite', c.suite,
                    'nonce', c.nonce,
                    'emitida_en', c.emitida_en,
                    'expira_en', c.expira_en,
                    'mac_sha256', c.mac
                )::pg_catalog.text,
                '" : ', '":'
            ),
            ', "', ',"'
        ),
        'UTF8'
    )
      FROM campos AS c
$vector$;

REVOKE ALL ON FUNCTION
    vec_autorizacion_atestada_v3.vector_capacidad_a4_prueba() FROM PUBLIC;

DO $vector_dorado$
DECLARE
    v_capacidad pg_catalog.bytea := vec_autorizacion_atestada_v3
        .vector_capacidad_a4_prueba();
    v_resultado record;
BEGIN
    SELECT * INTO STRICT v_resultado
      FROM vec_autorizacion_atestada_v3
           .canon_y_huella_consumo_fuente_corporativa_v1(
               v_capacidad, '2099-01-01T00:00:01Z'::pg_catalog.timestamptz
           );
    IF pg_catalog.octet_length(v_capacidad) <> 1891
       OR pg_catalog.encode(pg_catalog.sha256(v_capacidad), 'hex') <>
          'd3baaa6bf9e8e757d659f42233186a799e3c0b6e9a8e5eab1b5930ca0e7f7e54'
       OR pg_catalog.octet_length(v_resultado.consumo_canonico) <> 2021
       OR pg_catalog.encode(
              pg_catalog.sha256(v_resultado.consumo_canonico), 'hex') <>
          '0755995c42bdbdf7de83d6066c3b17c3f95534bc17de35a5d91f43560b3f1e85'
       OR v_resultado.consumo_huella_sha256 <>
          '0755995c42bdbdf7de83d6066c3b17c3f95534bc17de35a5d91f43560b3f1e85'
       OR pg_catalog.convert_from(v_resultado.consumo_canonico, 'UTF8')
          NOT LIKE '%"capacidad_ref":"cfc_d3baaa6bf9e8e757d659f42233186a799e3c0b6e9a8e5eab1b5930ca0e7f7e54"%'
       OR pg_catalog.convert_from(v_resultado.consumo_canonico, 'UTF8')
          NOT LIKE '%"consumida_en":"2099-01-01T00:00:01.000000Z"}' THEN
        RAISE EXCEPTION USING ERRCODE = 'XX000',
            MESSAGE = 'A4: canon o huella no coincide byte a byte con V0';
    END IF;
END
$vector_dorado$;

DO $forma_tipos_y_adulteracion$
DECLARE
    v_base pg_catalog.text := pg_catalog.convert_from(
        vec_autorizacion_atestada_v3.vector_capacidad_a4_prueba(), 'UTF8'
    );
    v_caso pg_catalog.text;
    v_cantidad pg_catalog.int4;
BEGIN
    FOREACH v_caso IN ARRAY ARRAY[
        pg_catalog.replace(v_base, '{"esquema":', '{ "esquema":'),
        pg_catalog.replace(v_base,
            '"version":1,"fuente_ref":', '"fuente_ref":'),
        pg_catalog.replace(v_base,
            '"version":1,"fuente_ref":"fuente:sintetica:f0:v1"',
            '"fuente_ref":"fuente:sintetica:f0:v1","version":1'),
        pg_catalog.replace(v_base, '{"esquema":',
            '{"esquema":"duplicado","esquema_descartado":'),
        pg_catalog.left(v_base, pg_catalog.length(v_base) - 1) ||
            ',"consumo_canonico":"aportado"}',
        pg_catalog.left(v_base, pg_catalog.length(v_base) - 1) ||
            ',"consumo_huella_sha256":"' || pg_catalog.repeat('a', 64) || '"}',
        pg_catalog.left(v_base, pg_catalog.length(v_base) - 1) ||
            ',"capacidad_ref":"cfc_aportada"}',
        pg_catalog.replace(v_base, '"version":1', '"version":"1"'),
        pg_catalog.replace(v_base, '"fuente_version":1',
            '"fuente_version":1.0'),
        pg_catalog.replace(v_base, '"clave_version":2',
            '"clave_version":2e0'),
        pg_catalog.replace(v_base, '"revision_gobierno":7',
            '"revision_gobierno":null'),
        pg_catalog.replace(v_base, '"configuracion_secuencia":11',
            '"configuracion_secuencia":[]'),
        pg_catalog.replace(v_base, '"raiz_version":3',
            '"raiz_version":{}'),
        pg_catalog.replace(v_base,
            '"fuente_ref":"fuente:sintetica:f0:v1"',
            '"fuente_ref":"\u0066uente:sintetica:f0:v1"'),
        pg_catalog.replace(v_base,
            '"audiencia_despliegue":"vec-diputacion/pruebas/f0/fuente-corporativa"',
            '"audiencia_despliegue":"vec-diputacion\/pruebas\/f0\/fuente-corporativa"'),
        pg_catalog.replace(v_base,
            '"evento_fuente_ref":"evento:sintetico:f0:0001"',
            '"evento_fuente_ref":["evento:sintetico:f0:0001"]'),
        pg_catalog.replace(v_base,
            '"huella_evento_fuente_sha256":"' || pg_catalog.repeat('1', 64) || '"',
            '"huella_evento_fuente_sha256":1'),
        pg_catalog.replace(v_base,
            '"emitida_en":"2099-01-01T00:00:00.500000Z"',
            '"emitida_en":true'),
        '[' || v_base || ']',
        v_base || 'x'
    ] LOOP
        SELECT pg_catalog.count(*) INTO v_cantidad
          FROM vec_autorizacion_atestada_v3
               .canon_y_huella_consumo_fuente_corporativa_v1(
                   pg_catalog.convert_to(v_caso, 'UTF8'),
                   '2099-01-01T00:00:01Z'::pg_catalog.timestamptz
               );
        IF v_cantidad <> 0 THEN
            RAISE EXCEPTION USING ERRCODE = 'XX000',
                MESSAGE = 'A4: forma, tipo o dato derivado adulterado aceptado';
        END IF;
    END LOOP;
END
$forma_tipos_y_adulteracion$;

DO $semantica_limites_e_instantes$
DECLARE
    v_base pg_catalog.text := pg_catalog.convert_from(
        vec_autorizacion_atestada_v3.vector_capacidad_a4_prueba(), 'UTF8'
    );
    v_caso pg_catalog.text;
    v_cantidad pg_catalog.int4;
BEGIN
    FOREACH v_caso IN ARRAY ARRAY[
        pg_catalog.replace(v_base,
            'vec.contexto-actor.fuente-corporativa.capacidad.v1',
            'vec.contexto-actor.fuente-corporativa.capacidad.v2'),
        pg_catalog.replace(v_base, '"version":1', '"version":2'),
        pg_catalog.replace(v_base,
            '"fuente_ref":"fuente:sintetica:f0:v1"', '"fuente_ref":"f0"'),
        pg_catalog.replace(v_base, '"fuente_version":1',
            '"fuente_version":9007199254740992'),
        pg_catalog.replace(v_base,
            '"evento_fuente_ref":"evento:sintetico:f0:0001"',
            '"evento_fuente_ref":"evento con espacio"'),
        pg_catalog.replace(v_base,
            '"operacion_ref":"oca_vector_f0_0000000000000001"',
            '"operacion_ref":"oca_corta"'),
        pg_catalog.replace(v_base,
            '"efecto_ref":"efecto:sintetico:f0:0001"',
            '"efecto_ref":"efecto con espacio"'),
        pg_catalog.replace(v_base, pg_catalog.repeat('1', 64),
            pg_catalog.repeat('0', 64)),
        pg_catalog.replace(v_base, pg_catalog.repeat('8', 64),
            'A' || pg_catalog.repeat('8', 63)),
        pg_catalog.replace(v_base,
            '"clave_id":"clave:sintetica:f0:0001"',
            '"clave_id":"' || pg_catalog.repeat('k', 513) || '"'),
        pg_catalog.replace(v_base,
            '"suite":"VEC-AD-3-COSE-EDDSA-1"',
            '"suite":"otra"'),
        pg_catalog.replace(v_base,
            '"evento_fuente_emitido_en":"2099-01-01T00:00:00.000000Z"',
            '"evento_fuente_emitido_en":"2099-01-01T00:00:01.000000Z"'),
        pg_catalog.replace(v_base,
            '"emitida_en":"2099-01-01T00:00:00.500000Z"',
            '"emitida_en":"2099-01-01T00:00:00Z"'),
        pg_catalog.replace(v_base,
            '"expira_en":"2099-01-01T00:00:04.500000Z"',
            '"expira_en":"2099-01-01T00:00:05.500001Z"'),
        pg_catalog.replace(v_base,
            '"expira_en":"2099-01-01T00:00:04.500000Z"',
            '"expira_en":"2099-01-01T00:00:00.500000Z"'),
        pg_catalog.replace(v_base,
            '"accion":"contexto_actor.organizacion_corporativa.publicar"',
            '"accion":"contexto_actor.vinculo_corporativo.publicar"'),
        pg_catalog.replace(v_base,
            '"tipo_efecto":"organizacion_corporativa.alta"',
            '"tipo_efecto":"vinculo_corporativo.alta"')
    ] LOOP
        SELECT pg_catalog.count(*) INTO v_cantidad
          FROM vec_autorizacion_atestada_v3
               .canon_y_huella_consumo_fuente_corporativa_v1(
                   pg_catalog.convert_to(v_caso, 'UTF8'),
                   '2099-01-01T00:00:01Z'::pg_catalog.timestamptz
               );
        IF v_cantidad <> 0 THEN
            RAISE EXCEPTION USING ERRCODE = 'XX000',
                MESSAGE = 'A4: semántica, límite o cruce inválido aceptado';
        END IF;
    END LOOP;

    FOREACH v_caso IN ARRAY ARRAY[
        '2099-01-01T00:00:00.499999Z',
        '2099-01-01T00:00:04.500000Z',
        'infinity'
    ] LOOP
        SELECT pg_catalog.count(*) INTO v_cantidad
          FROM vec_autorizacion_atestada_v3
               .canon_y_huella_consumo_fuente_corporativa_v1(
                   vec_autorizacion_atestada_v3.vector_capacidad_a4_prueba(),
                   v_caso::pg_catalog.timestamptz
               );
        IF v_cantidad <> 0 THEN
            RAISE EXCEPTION USING ERRCODE = 'XX000',
                MESSAGE = 'A4: instante de consumo fuera de vigencia aceptado';
        END IF;
    END LOOP;
END
$semantica_limites_e_instantes$;

-- Cada fila altera un solo campo y queda ligada al validador que M040 debe
-- aplicarle. Así, omitir cualquiera de las entradas de ambos conjuntos deja
-- de ser una mutación equivalente y hace fallar esta etapa.
DO $mapeo_exhaustivo_validadores$
DECLARE
    v_base pg_catalog.text := pg_catalog.convert_from(
        vec_autorizacion_atestada_v3.vector_capacidad_a4_prueba(), 'UTF8'
    );
    v_caso pg_catalog.text;
    v_original pg_catalog.text;
    v_fila record;
    v_cantidad pg_catalog.int4;
BEGIN
    FOR v_fila IN
        SELECT * FROM (VALUES
            ('huella_evento_fuente_sha256', 'huella_sha256_valida', 'x'),
            ('huella_manifiesto_fuente_sha256', 'huella_sha256_valida', 'x'),
            ('huella_sobre_cose_sign1_sha256', 'huella_sha256_valida', 'x'),
            ('huella_prueba_confianza_sha256', 'huella_sha256_valida', 'x'),
            ('huella_efecto_sha256', 'huella_sha256_valida', 'x'),
            ('huella_gobierno_sha256', 'huella_sha256_valida', 'x'),
            ('huella_configuracion_sha256', 'huella_sha256_valida', 'x'),
            ('huella_raiz_spki_sha256', 'huella_sha256_valida', 'x'),
            ('nonce', 'huella_sha256_valida', 'x'),
            ('mac_sha256', 'huella_sha256_valida', 'x'),
            ('clave_id', 'texto_tecnico_valido', '*'),
            ('emisor_id', 'texto_tecnico_valido', '*'),
            ('configuracion_revision', 'texto_tecnico_valido', '*'),
            ('raiz_clave_id', 'texto_tecnico_valido', '*'),
            ('audiencia_despliegue', 'texto_tecnico_valido', '*')
        ) AS m(campo, validador, invalido)
    LOOP
        v_original := (v_base::pg_catalog.json) ->> v_fila.campo;
        v_caso := pg_catalog.replace(
            v_base,
            '"' || v_fila.campo || '":' ||
                vec_autorizacion_atestada_v3.texto_json_go(v_original),
            '"' || v_fila.campo || '":' ||
                vec_autorizacion_atestada_v3.texto_json_go(v_fila.invalido)
        );
        IF v_caso = v_base THEN
            RAISE EXCEPTION USING ERRCODE = 'XX000',
                MESSAGE = 'A4: campo sin mutar en matriz: ' || v_fila.campo;
        END IF;
        SELECT pg_catalog.count(*) INTO v_cantidad
          FROM vec_autorizacion_atestada_v3
               .canon_y_huella_consumo_fuente_corporativa_v1(
                   pg_catalog.convert_to(v_caso, 'UTF8'),
                   '2099-01-01T00:00:01Z'::pg_catalog.timestamptz
               );
        IF v_cantidad <> 0 THEN
            RAISE EXCEPTION USING ERRCODE = 'XX000',
                MESSAGE = 'A4: mapping omitible ' || v_fila.campo ||
                          ' -> ' || v_fila.validador;
        END IF;
    END LOOP;
END
$mapeo_exhaustivo_validadores$;

DO $seis_enteros_seguros$
DECLARE
    v_base pg_catalog.text := pg_catalog.convert_from(
        vec_autorizacion_atestada_v3.vector_capacidad_a4_prueba(), 'UTF8');
    v_campo pg_catalog.text;
    v_original pg_catalog.text;
    v_invalido pg_catalog.text;
    v_caso pg_catalog.text;
    v_cantidad pg_catalog.int4;
BEGIN
    FOREACH v_campo IN ARRAY ARRAY[
        'version', 'fuente_version', 'clave_version',
        'revision_gobierno', 'configuracion_secuencia', 'raiz_version'
    ] LOOP
        v_original := (v_base::pg_catalog.json -> v_campo)::pg_catalog.text;
        FOREACH v_invalido IN ARRAY ARRAY[
            '1.0', '0', '9007199254740992'
        ] LOOP
            v_caso := pg_catalog.replace(
                v_base, '"' || v_campo || '":' || v_original,
                '"' || v_campo || '":' || v_invalido);
            IF v_caso = v_base THEN
                RAISE EXCEPTION USING ERRCODE = 'XX000',
                    MESSAGE = 'A4: entero sin mutar: ' || v_campo;
            END IF;
            SELECT pg_catalog.count(*) INTO v_cantidad
              FROM vec_autorizacion_atestada_v3
                   .canon_y_huella_consumo_fuente_corporativa_v1(
                       pg_catalog.convert_to(v_caso, 'UTF8'),
                       '2099-01-01T00:00:01Z'::pg_catalog.timestamptz);
            IF v_cantidad <> 0 THEN
                RAISE EXCEPTION USING ERRCODE = 'XX000', MESSAGE =
                    'A4: entero sin validador seguro: ' || v_campo;
            END IF;
        END LOOP;
    END LOOP;
END
$seis_enteros_seguros$;

DO $cuatro_cruces_y_bordes$
DECLARE
    v_base pg_catalog.text := pg_catalog.convert_from(
        vec_autorizacion_atestada_v3.vector_capacidad_a4_prueba(), 'UTF8'
    );
    v_caso pg_catalog.text;
    v_instante pg_catalog.text;
    v_fila record;
    v_cantidad pg_catalog.int4;
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
        v_caso := pg_catalog.replace(
            pg_catalog.replace(
                pg_catalog.replace(v_base,
                    'vec_contexto_actor.publicar_organizacion_corporativa_fuente.v1',
                    v_fila.audiencia),
                'contexto_actor.organizacion_corporativa.publicar',
                v_fila.accion),
            'organizacion_corporativa.alta', v_fila.tipo);
        SELECT pg_catalog.count(*) INTO v_cantidad
          FROM vec_autorizacion_atestada_v3
               .canon_y_huella_consumo_fuente_corporativa_v1(
                   pg_catalog.convert_to(v_caso, 'UTF8'),
                   '2099-01-01T00:00:01Z'::pg_catalog.timestamptz
               );
        IF v_cantidad <> 1 THEN
            RAISE EXCEPTION USING ERRCODE = 'XX000',
                MESSAGE = 'A4: cruce nominal válido rechazado';
        END IF;
    END LOOP;

    -- Los bordes de consumo son [emitida_en,expira_en).
    FOREACH v_instante IN ARRAY ARRAY[
        '2099-01-01T00:00:00.500000Z',
        '2099-01-01T00:00:04.499999Z'
    ] LOOP
        SELECT pg_catalog.count(*) INTO v_cantidad
          FROM vec_autorizacion_atestada_v3
               .canon_y_huella_consumo_fuente_corporativa_v1(
                   vec_autorizacion_atestada_v3.vector_capacidad_a4_prueba(),
                   v_instante::pg_catalog.timestamptz);
        IF v_cantidad <> 1 THEN
            RAISE EXCEPTION USING ERRCODE = 'XX000',
                MESSAGE = 'A4: borde inclusivo de consumo rechazado';
        END IF;
    END LOOP;

    -- El evento puede coincidir exactamente con la emisión de la capacidad.
    v_caso := pg_catalog.replace(v_base,
        '"evento_fuente_emitido_en":"2099-01-01T00:00:00.000000Z"',
        '"evento_fuente_emitido_en":"2099-01-01T00:00:00.500000Z"');
    SELECT pg_catalog.count(*) INTO v_cantidad
      FROM vec_autorizacion_atestada_v3
           .canon_y_huella_consumo_fuente_corporativa_v1(
               pg_catalog.convert_to(v_caso, 'UTF8'),
               '2099-01-01T00:00:01Z'::pg_catalog.timestamptz);
    IF v_cantidad <> 1 THEN
        RAISE EXCEPTION USING ERRCODE = 'XX000',
            MESSAGE = 'A4: evento simultáneo a emisión rechazado';
    END IF;

    -- La vigencia positiva máxima de cinco segundos es inclusiva. Este caso
    -- mata una mutación de la guarda productiva de > a >=.
    v_caso := pg_catalog.replace(
        v_base,
        '"expira_en":"2099-01-01T00:00:04.500000Z"',
        '"expira_en":"2099-01-01T00:00:05.500000Z"'
    );
    SELECT pg_catalog.count(*) INTO v_cantidad
      FROM vec_autorizacion_atestada_v3
           .canon_y_huella_consumo_fuente_corporativa_v1(
               pg_catalog.convert_to(v_caso, 'UTF8'),
               '2099-01-01T00:00:01Z'::pg_catalog.timestamptz
           );
    IF v_cantidad <> 1 THEN
        RAISE EXCEPTION USING ERRCODE = 'XX000',
            MESSAGE = 'A4: vigencia máxima exacta de cinco segundos rechazada';
    END IF;
END
$cuatro_cruces_y_bordes$;

DO $escapes_validos_go$
DECLARE
    v_base pg_catalog.text := pg_catalog.convert_from(
        vec_autorizacion_atestada_v3.vector_capacidad_a4_prueba(), 'UTF8'
    );
    v_caso pg_catalog.bytea;
    v_resultado record;
BEGIN
    v_caso := pg_catalog.convert_to(pg_catalog.replace(
        v_base,
        '"emisor_id":"emisor:sintetico:f0:0001"',
        '"emisor_id":"emisor:\"x\\y\u0026\u003c\u003e"'
    ), 'UTF8');
    SELECT * INTO STRICT v_resultado
      FROM vec_autorizacion_atestada_v3
           .canon_y_huella_consumo_fuente_corporativa_v1(
               v_caso, '2099-01-01T00:00:01Z'::pg_catalog.timestamptz
           );
    IF pg_catalog.convert_from(v_resultado.consumo_canonico, 'UTF8')
       NOT LIKE '%"emisor_id":"emisor:\"x\\y\u0026\u003c\u003e"%' ESCAPE ''
       OR v_resultado.consumo_huella_sha256 <>
          pg_catalog.encode(
              pg_catalog.sha256(v_resultado.consumo_canonico), 'hex') THEN
        RAISE EXCEPTION USING ERRCODE = 'XX000',
            MESSAGE = 'A4: escapes canónicos de Go no conservados';
    END IF;
END
$escapes_validos_go$;

DO $entrada_binaria_y_nulos$
DECLARE
    v_base pg_catalog.bytea := vec_autorizacion_atestada_v3
        .vector_capacidad_a4_prueba();
    v_caso pg_catalog.bytea;
    v_cantidad pg_catalog.int4;
BEGIN
    FOREACH v_caso IN ARRAY ARRAY[
        NULL::pg_catalog.bytea,
        pg_catalog.convert_to(pg_catalog.repeat('x', 511), 'UTF8'),
        pg_catalog.convert_to(pg_catalog.repeat('x', 32769), 'UTF8'),
        pg_catalog.decode('efbbbf', 'hex') || v_base,
        pg_catalog.decode('c328', 'hex'),
        pg_catalog.substr(v_base, 1, 900) || pg_catalog.decode('00', 'hex') ||
            pg_catalog.substr(v_base, 901),
        pg_catalog.convert_to('{"esquema":', 'UTF8')
    ] LOOP
        SELECT pg_catalog.count(*) INTO v_cantidad
          FROM vec_autorizacion_atestada_v3
               .canon_y_huella_consumo_fuente_corporativa_v1(
                   v_caso,
                   '2099-01-01T00:00:01Z'::pg_catalog.timestamptz
               );
        IF v_cantidad <> 0 THEN
            RAISE EXCEPTION USING ERRCODE = 'XX000',
                MESSAGE = 'A4: tamaño, UTF-8, NUL o JSON inválido aceptado';
        END IF;
    END LOOP;

    SELECT pg_catalog.count(*) INTO v_cantidad
      FROM vec_autorizacion_atestada_v3
           .canon_y_huella_consumo_fuente_corporativa_v1(v_base, NULL);
    IF v_cantidad <> 0 THEN
        RAISE EXCEPTION USING ERRCODE = 'XX000',
            MESSAGE = 'A4: instante nulo aceptado';
    END IF;
END
$entrada_binaria_y_nulos$;

DROP FUNCTION vec_autorizacion_atestada_v3.vector_capacidad_a4_prueba();
