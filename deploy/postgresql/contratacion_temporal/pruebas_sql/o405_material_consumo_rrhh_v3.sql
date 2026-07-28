\set ON_ERROR_STOP on
SET ROLE vec_contratacion_temporal_propietario;
SET search_path = pg_catalog;

DO $vector_material$
DECLARE
    v_capacidad_texto text :=
        '{"esquema":"vec.autorizacion.capacidad-registro-consumo-atestado.v3"'
        || ',"version":3,"clave_id":"clave-capacidad-vector-pg"'
        || ',"clave_version":7,"revision_gobierno":9'
        || ',"huella_gobierno_sha256":"'
        || pg_catalog.repeat('1', 64) || '"'
        || ',"emisor_id":"broker-vector-pg"'
        || ',"audiencia_consumo":'
        || '"vec_contratacion_temporal.consultar_cuadro_rrhh.v3"'
        || ',"nonce":"' || pg_catalog.repeat('2', 64) || '"'
        || ',"emitida_en":"2026-07-28T08:09:10.123456Z"'
        || ',"expira_en":"2026-07-28T08:09:14.123456Z"'
        || ',"decision_ref":"decision:rrhh:vector-pg"'
        || ',"huella_decision_sha256":"'
        || pg_catalog.repeat('3', 64) || '"'
        || ',"huella_motivo_sha256":"'
        || pg_catalog.repeat('4', 64) || '"'
        || ',"huella_payload_vec_ad_3_sha256":"'
        || pg_catalog.repeat('5', 64) || '"'
        || ',"huella_sobre_cose_sign1_sha256":"'
        || pg_catalog.repeat('6', 64) || '"'
        || ',"huella_prueba_confianza_sha256":"'
        || pg_catalog.repeat('7', 64) || '"'
        || ',"contexto_ref":"contexto:rrhh:vector-pg"'
        || ',"huella_contexto_sha256":"'
        || pg_catalog.repeat('8', 64) || '"'
        || ',"audiencia_despliegue":'
        || '"vec-diputacion/pruebas/ct000042"'
        || ',"operacion":'
        || '"contratacion_temporal.consultar_cuadro_rrhh"'
        || ',"efecto_ref":"consulta:rrhh:vector-pg"'
        || ',"huella_efecto_sha256":"'
        || pg_catalog.repeat('9', 64) || '"'
        || ',"decision_valida_hasta":"2026-07-28T08:10:00Z"'
        || ',"verificada_en":"2026-07-28T08:09:10.123456Z"'
        || ',"revision_confianza":"configuracion-vector-pg"'
        || ',"configuracion_secuencia":4'
        || ',"huella_configuracion_sha256":"'
        || pg_catalog.repeat('a', 64) || '"'
        || ',"configuracion_publicada_en":"2026-07-28T08:00:00Z"'
        || ',"configuracion_expira_en":"2026-07-28T09:00:00Z"'
        || ',"raiz_clave_id":"raiz-vector-pg","raiz_version":3'
        || ',"huella_raiz_spki_sha256":"'
        || pg_catalog.repeat('b', 64) || '"'
        || ',"raiz_valida_desde":"2026-07-28T07:00:00Z"'
        || ',"raiz_valida_hasta":"2026-07-28T09:00:00Z"'
        || ',"suite":"VEC-AD-3-COSE-EDDSA-1"'
        || ',"mac_sha256":"' || pg_catalog.repeat('c', 64) || '"}';
    v_capacidad bytea;
    v_spki bytea := pg_catalog.decode(
        '302a300506032b6570032100'
        || '2152f8d19b791d24453242e15f2eab6c'
        || 'b7cffa7b6a5ed30097960e069881db12',
        'hex'
    );
    v_huella text;
    v_huella_mutada text;
    v_capacidad_mutada bytea;
    v_claves text[] := ARRAY[
        'decision_ref', 'huella_decision_sha256',
        'huella_motivo_sha256', 'contexto_ref',
        'huella_contexto_sha256', 'operacion', 'efecto_ref',
        'huella_efecto_sha256', 'audiencia_consumo'
    ]::text[];
    v_originales text[] := ARRAY[
        'decision:rrhh:vector-pg',
        pg_catalog.repeat('3', 64),
        pg_catalog.repeat('4', 64),
        'contexto:rrhh:vector-pg',
        pg_catalog.repeat('8', 64),
        'contratacion_temporal.consultar_cuadro_rrhh',
        'consulta:rrhh:vector-pg',
        pg_catalog.repeat('9', 64),
        'vec_contratacion_temporal.consultar_cuadro_rrhh.v3'
    ]::text[];
    v_sustituciones text[] := ARRAY[
        'decision:rrhh:vector-pg-mutada',
        pg_catalog.repeat('d', 64),
        pg_catalog.repeat('e', 64),
        'contexto:rrhh:vector-pg-mutado',
        pg_catalog.repeat('f', 64),
        'contratacion_temporal.consultar_detalle_rrhh',
        'consulta:rrhh:vector-pg-mutada',
        pg_catalog.repeat('0', 64),
        'vec_contratacion_temporal.consultar_detalle_rrhh.v3',
        '2026-07-28T08:09:10.123455Z',
        '2026-07-28T08:09:14.123455Z'
    ]::text[];
    -- Vectores independientes congelados por Go en el test homólogo.
    v_esperadas text[] := ARRAY[
        'b65739d07b756dfda24371f87a251ed5c00403f229edd12beafb1e19c7d7c2ba',
        '99043ef80ec527b4656c6ebf17fbe70cad3948aa166486c1604582f2662244c9',
        '430d2521333467e61a5970248388b9ee71442da105dda03e297b83138df41e6a',
        'a5a4554a9e6cf3fb4fa49b16826139c17d00de3b9c8c2245b26042b4fd0ee35d',
        '0833ad4684de627151eaa26efddf552f48c8a4ab56853689f52664efd8f57f41',
        '5137b04fa254136f9fec6c9ee675424dcb48f492e968369f948e1a5d0540da91',
        '8879232c6eefebd7b09d953fbd752c18e8548e274423a9d4c4811473803d7a69',
        '95ae589e23cf92e71e4e4e631bc738d003bcae9f5407299e4d445564812c0cb5',
        '937fdb129acf8f263b4ed89e42d203270d83c597e50b228f4ad5cf432aff4c83',
        'd2913bbec135a2a45eca11f2b4ade042f6913107d912c6c3c2d9239ec9945f0c',
        'bcebf071b0b2eb8ae2a8155e44f512078779485f221b6cf12003cb1dc1365547',
        '31565565070d5a0f9346ee7dd6422f2bc0b58cd044446baaa418e20e0bd4a434',
        '5a309a3b596f43379e6a1240693dfd08808f73e0876ee3563b1bb24bfe77a4f8',
        'feb4468ae7dc96885e420d3796102f95a1362de0cff8add425aaffc0be0284c1',
        '62e8c10da41b6f5c246ddcd02865fb76c23da0fa0f6073c58c1022641845b7e1',
        '9c0ef62650e647a9463760fd20f71a47ca1c9b9a77b66029d4a892a7acc2b120',
        'a31801128d07a61c9da3d8a45476709f617d1b2bf9fc518d54be7b022ff9d74b',
        '04fab017f8b25ccdf5b4e2b6b036fe455deeb32f2a0ec472801a7badc02f5bef',
        'c1e63385149893769df8abf1ec3946e26b6ba6f9922a5a9903e2487c865abe5c',
        '4b9eee752de03f8aa2869b025d81c2551da36c25facedb14457d9ab9aa95bc26',
        '7a72e01afd99529e2ab2c562f11147278828c08ade21b4a01f6c7718232f469a'
    ]::text[];
    v_texto_mutado text;
    v_indice integer;
    v_mutaciones integer := 0;
BEGIN
    v_capacidad :=
        vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
            v_capacidad_texto
        );
    IF pg_catalog.octet_length(v_capacidad) <> 2123 THEN
        RAISE EXCEPTION 'capacidad del vector material divergente: %',
            pg_catalog.octet_length(v_capacidad);
    END IF;
    v_huella := vec_contratacion_temporal.huella_material_consumo_rrhh_v3(
        v_capacidad,
        vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
            'decision-canonica-vector-pg'
        ),
        vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
            'motivo-canonico-vector-pg'
        ),
        vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
            'contexto-actor-canonico-vector-pg'
        ),
        7, 11,
        vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
            'payload-vec-ad-3-vector-pg'
        ),
        vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
            'sobre-cose-sign1-vector-pg'
        ),
        vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
            'evidencia-verificacion-vector-pg'
        ),
        v_spki
    );
    IF v_huella <>
       '7195a7271ec37370688df27f8e3dfddf4ea91864babf4906910b8faf81f6a0a3'
    THEN
        RAISE EXCEPTION 'vector material divergente: %', v_huella;
    END IF;

    -- Bloque 1: cambia solo los bytes completos de capacidad; sus once
    -- resúmenes conservan valor. Bloques 2..12: la API los deriva del mismo
    -- canon, por lo que cada caso se compara con su golden Go exacto.
    v_texto_mutado := pg_catalog.left(
        v_capacidad_texto,
        pg_catalog.char_length(v_capacidad_texto) - 1
    ) || ',"marca_prueba":"mutada"}';
    v_capacidad_mutada :=
        vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
            v_texto_mutado
        );
    v_huella_mutada :=
        vec_contratacion_temporal.huella_material_consumo_rrhh_v3(
            v_capacidad_mutada,
            vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
                'decision-canonica-vector-pg'
            ),
            vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
                'motivo-canonico-vector-pg'
            ),
            vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
                'contexto-actor-canonico-vector-pg'
            ),
            7, 11,
            vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
                'payload-vec-ad-3-vector-pg'
            ),
            vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
                'sobre-cose-sign1-vector-pg'
            ),
            vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
                'evidencia-verificacion-vector-pg'
            ),
            v_spki
        );
    IF v_huella_mutada <> v_esperadas[1] THEN
        RAISE EXCEPTION 'golden Go del bloque material 1 divergente: %',
            v_huella_mutada;
    END IF;
    v_mutaciones := v_mutaciones + 1;

    FOR v_indice IN 1..11 LOOP
        IF v_indice <= 9 THEN
            v_texto_mutado := pg_catalog.replace(
                v_capacidad_texto,
                pg_catalog.format(
                    '"%s":"%s"', v_claves[v_indice],
                    v_originales[v_indice]
                ),
                pg_catalog.format(
                    '"%s":"%s"', v_claves[v_indice],
                    v_sustituciones[v_indice]
                )
            );
        ELSIF v_indice = 10 THEN
            v_texto_mutado := pg_catalog.replace(
                v_capacidad_texto,
                '"emitida_en":"2026-07-28T08:09:10.123456Z"',
                '"emitida_en":"2026-07-28T08:09:10.123455Z"'
            );
        ELSE
            v_texto_mutado := pg_catalog.replace(
                v_capacidad_texto,
                '"expira_en":"2026-07-28T08:09:14.123456Z"',
                '"expira_en":"2026-07-28T08:09:14.123455Z"'
            );
        END IF;
        v_capacidad_mutada :=
            vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
                v_texto_mutado
            );
        v_huella_mutada :=
            vec_contratacion_temporal.huella_material_consumo_rrhh_v3(
                v_capacidad_mutada,
                vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
                    'decision-canonica-vector-pg'
                ),
                vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
                    'motivo-canonico-vector-pg'
                ),
                vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
                    'contexto-actor-canonico-vector-pg'
                ),
                7, 11,
                vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
                    'payload-vec-ad-3-vector-pg'
                ),
                vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
                    'sobre-cose-sign1-vector-pg'
                ),
                vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
                    'evidencia-verificacion-vector-pg'
                ),
                v_spki
            );
        IF v_huella_mutada <> v_esperadas[v_indice + 1] THEN
            RAISE EXCEPTION 'golden Go del bloque material % divergente: %',
                v_indice + 1, v_huella_mutada;
        END IF;
        v_mutaciones := v_mutaciones + 1;
    END LOOP;

    FOR v_indice IN 1..9 LOOP
        v_huella_mutada :=
            vec_contratacion_temporal.huella_material_consumo_rrhh_v3(
                v_capacidad,
                CASE WHEN v_indice = 1 THEN '\x02'::bytea
                     ELSE vec_contratacion_temporal
                         .codificar_texto_utf8_rrhh_v1(
                             'decision-canonica-vector-pg'
                         ) END,
                CASE WHEN v_indice = 2 THEN '\x02'::bytea
                     ELSE vec_contratacion_temporal
                         .codificar_texto_utf8_rrhh_v1(
                             'motivo-canonico-vector-pg'
                         ) END,
                CASE WHEN v_indice = 3 THEN '\x02'::bytea
                     ELSE vec_contratacion_temporal
                         .codificar_texto_utf8_rrhh_v1(
                             'contexto-actor-canonico-vector-pg'
                         ) END,
                CASE WHEN v_indice = 4 THEN 8 ELSE 7 END,
                CASE WHEN v_indice = 5 THEN 12 ELSE 11 END,
                CASE WHEN v_indice = 6 THEN '\x02'::bytea
                     ELSE vec_contratacion_temporal
                         .codificar_texto_utf8_rrhh_v1(
                             'payload-vec-ad-3-vector-pg'
                         ) END,
                CASE WHEN v_indice = 7 THEN '\x02'::bytea
                     ELSE vec_contratacion_temporal
                         .codificar_texto_utf8_rrhh_v1(
                             'sobre-cose-sign1-vector-pg'
                         ) END,
                CASE WHEN v_indice = 8 THEN '\x02'::bytea
                     ELSE vec_contratacion_temporal
                         .codificar_texto_utf8_rrhh_v1(
                             'evidencia-verificacion-vector-pg'
                         ) END,
                CASE WHEN v_indice = 9
                     THEN pg_catalog.set_byte(
                         v_spki, 43,
                         (pg_catalog.get_byte(v_spki, 43) + 1) % 256
                     )
                     ELSE v_spki END
            );
        IF v_huella_mutada <> v_esperadas[v_indice + 12] THEN
            RAISE EXCEPTION 'golden Go del bloque material % divergente: %',
                v_indice + 12, v_huella_mutada;
        END IF;
        v_mutaciones := v_mutaciones + 1;
    END LOOP;
    IF v_mutaciones <> 21
       OR pg_catalog.cardinality(v_esperadas) <> 21 THEN
        RAISE EXCEPTION 'matriz material incompleta: %', v_mutaciones;
    END IF;
END
$vector_material$;

DO $omisiones_y_tiempos$
DECLARE
    v_base jsonb := (
        '{"decision_ref":"decision:rrhh:vector-pg"'
        || ',"huella_decision_sha256":"' || pg_catalog.repeat('3', 64) || '"'
        || ',"huella_motivo_sha256":"' || pg_catalog.repeat('4', 64) || '"'
        || ',"contexto_ref":"contexto:rrhh:vector-pg"'
        || ',"huella_contexto_sha256":"' || pg_catalog.repeat('8', 64) || '"'
        || ',"operacion":"contratacion_temporal.consultar_cuadro_rrhh"'
        || ',"efecto_ref":"consulta:rrhh:vector-pg"'
        || ',"huella_efecto_sha256":"' || pg_catalog.repeat('9', 64) || '"'
        || ',"audiencia_consumo":'
        || '"vec_contratacion_temporal.consultar_cuadro_rrhh.v3"'
        || ',"emitida_en":"2026-07-28T08:09:10.123456Z"'
        || ',"expira_en":"2026-07-28T08:09:14.123456Z"'
        || ',"relleno":"' || pg_catalog.repeat('r', 600) || '"}'
    )::jsonb;
    v_clave text;
    v_mutilada bytea;
    v_version numeric;
    v_spki bytea := pg_catalog.decode(
        '302a300506032b6570032100'
        || pg_catalog.repeat('42', 32),
        'hex'
    );
BEGIN
    FOREACH v_clave IN ARRAY ARRAY[
        'decision_ref', 'huella_decision_sha256',
        'huella_motivo_sha256', 'contexto_ref',
        'huella_contexto_sha256', 'operacion', 'efecto_ref',
        'huella_efecto_sha256', 'audiencia_consumo',
        'emitida_en', 'expira_en'
    ]::text[] LOOP
        v_mutilada := pg_catalog.convert_to(
            (v_base - v_clave)::text, 'UTF8'
        );
        BEGIN
            PERFORM
                vec_contratacion_temporal.huella_material_consumo_rrhh_v3(
                    v_mutilada, '\x01', '\x01', '\x01', 1, 1,
                    '\x01', '\x01', '\x01', v_spki
                );
            RAISE EXCEPTION 'omisión aceptada: %', v_clave;
        EXCEPTION WHEN SQLSTATE '22023' THEN
            NULL;
        END;
    END LOOP;
    FOREACH v_mutilada IN ARRAY ARRAY[
        pg_catalog.convert_to(pg_catalog.replace(
            v_base::text,
            '2026-07-28T08:09:10.123456Z',
            '2026-07-28T24:09:10.123456Z'
        ), 'UTF8'),
        pg_catalog.convert_to(pg_catalog.replace(
            v_base::text,
            '2026-07-28T08:09:10.123456Z',
            '2026-07-28T08:09:60.123456Z'
        ), 'UTF8'),
        pg_catalog.convert_to(pg_catalog.replace(
            v_base::text,
            '2026-07-28T08:09:10.123456Z',
            '2026-07-28T08:09:10.1234560Z'
        ), 'UTF8'),
        pg_catalog.convert_to(pg_catalog.replace(
            v_base::text,
            '2026-07-28T08:09:10.123456Z',
            '2026-07-28T08:09:10.123450Z'
        ), 'UTF8'),
        pg_catalog.convert_to(pg_catalog.replace(
            v_base::text,
            '2026-07-28T08:09:10.123456Z',
            '2026-07-28T08:09:10.000000Z'
        ), 'UTF8'),
        pg_catalog.convert_to(pg_catalog.replace(
            v_base::text,
            '2026-07-28T08:09:10.123456Z',
            '2026-07-28T08:09:10.123456+00:00'
        ), 'UTF8')
    ]::bytea[] LOOP
        BEGIN
            PERFORM
                vec_contratacion_temporal.huella_material_consumo_rrhh_v3(
                    v_mutilada, '\x01', '\x01', '\x01', 1, 1,
                    '\x01', '\x01', '\x01', v_spki
                );
            RAISE EXCEPTION 'instante imposible aceptado';
        EXCEPTION WHEN SQLSTATE '22023' THEN
            NULL;
        END;
    END LOOP;
    FOREACH v_version IN ARRAY ARRAY[
        'NaN'::numeric, 'Infinity'::numeric, '-Infinity'::numeric,
        0::numeric, 1.5::numeric, 9007199254740992::numeric
    ] LOOP
        BEGIN
            PERFORM
                vec_contratacion_temporal.huella_material_consumo_rrhh_v3(
                    pg_catalog.convert_to(v_base::text, 'UTF8'),
                    '\x01', '\x01', '\x01', v_version, 1,
                    '\x01', '\x01', '\x01', v_spki
                );
            RAISE EXCEPTION 'versión numérica imposible aceptada';
        EXCEPTION WHEN SQLSTATE '22023' THEN
            NULL;
        END;
    END LOOP;
    BEGIN
        PERFORM
            vec_contratacion_temporal.huella_material_consumo_rrhh_v3(
                pg_catalog.convert_to(v_base::text, 'UTF8'),
                '\x01', '\x01', '\x01', 1, 1,
                '\x01', '\x01', '\x01',
                pg_catalog.set_byte(v_spki, 0, 0)
            );
        RAISE EXCEPTION 'SPKI con prefijo alterado aceptada';
    EXCEPTION WHEN SQLSTATE '22023' THEN
        NULL;
    END;
END
$omisiones_y_tiempos$;

RESET ROLE;
