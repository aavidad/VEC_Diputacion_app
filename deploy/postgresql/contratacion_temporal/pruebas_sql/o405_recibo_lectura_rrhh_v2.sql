\set ON_ERROR_STOP on
SET ROLE vec_contratacion_temporal_propietario;
SET search_path = pg_catalog;

DO $recibos$
DECLARE
    v_detalle vec_contratacion_temporal.evidencia_recibo_lectura_rrhh_v2;
    v_cuadro vec_contratacion_temporal.evidencia_recibo_lectura_rrhh_v2;
    v_canon bytea;
    v_esperado bytea;
    v_clave text;
    v_mutada vec_contratacion_temporal.evidencia_recibo_lectura_rrhh_v2;
    v_base jsonb;
    v_valor_sustituto jsonb;
    v_mutaciones integer := 0;
BEGIN
    v_detalle := ROW(
        'vec.contratacion-temporal.recibo-acceso-rrhh.o4-05.v2',
        'acceso:rrhh:f996e98f214fcc5c252569b61e95bb8c',
        1, pg_catalog.repeat('0', 64), pg_catalog.repeat('2', 64),
        pg_catalog.repeat('3', 64), '',
        '2026-07-26T08:00:02Z',
        'auditoria:vec:detalle:vector-pg', pg_catalog.repeat('a', 64),
        pg_catalog.repeat('d', 64),
        'dec_0123456789abcdef0123456789abcdef',
        'a805880b9bf33fbd9747fbd68e0784f7566ab06af8a04ae97bcd9408c69de8f3',
        '2ea16988ca9a3b973ff11693e6de4bd078775655cd6715c5a06a120f71b3e827',
        '000960b66c33a1b60730a47fc053e69bd76d381036406464916ccf226baa8a43',
        '4bdb64a7842bf5f7a8052e63fae9290530ada44b1c8bdbcaa71deeb329517e6c',
        'correlacion_11111111111111111111111111111111',
        'aut_0123456789abcdefghijkl', pg_catalog.repeat('1', 64),
        'ses_0123456789abcdefghijkl',
        'cse_0123456789abcdefghijkl', 2, pg_catalog.repeat('2', 64),
        'per_0123456789abcdefghijkla',
        'prf_0123456789abcdefghijkla', 5,
        'organizacion:diputacion-granada', 'organizacion',
        'organizacion:diputacion-granada',
        'contratacion_temporal.expediente.consultar',
        'tramitacion_expediente_contratacion_temporal',
        'expediente:rrhh:minimizado', 4, 1,
        '97b2d440c764090e452e51fb3623900a2cac78d97f337a42437b599ec6335e9b',
        '1c21303d3904a4e7de6f6b5aac2ff0c3086bafe68dee9037b95d74515f3bcf26',
        '', '2026-07-26T08:00:01Z'
    );
    v_canon :=
        vec_contratacion_temporal.canon_recibo_lectura_rrhh_v2(v_detalle);
    v_esperado := pg_catalog.convert_to(
        'VEC-CT-RECIBO-LECTURA-RRHH-V2' || pg_catalog.chr(10)
        || '53:vec.contratacion-temporal.recibo-acceso-rrhh.o4-05.v2'
        || pg_catalog.chr(10)
        || '44:acceso:rrhh:f996e98f214fcc5c252569b61e95bb8c'
        || pg_catalog.chr(10) || '1:1' || pg_catalog.chr(10)
        || '64:' || pg_catalog.repeat('0', 64) || pg_catalog.chr(10)
        || '64:' || pg_catalog.repeat('2', 64) || pg_catalog.chr(10)
        || '64:' || pg_catalog.repeat('3', 64) || pg_catalog.chr(10)
        || '0:' || pg_catalog.chr(10)
        || '27:2026-07-26T08:00:02.000000Z' || pg_catalog.chr(10)
        || '31:auditoria:vec:detalle:vector-pg' || pg_catalog.chr(10)
        || '64:' || pg_catalog.repeat('a', 64) || pg_catalog.chr(10)
        || '64:' || pg_catalog.repeat('d', 64) || pg_catalog.chr(10)
        || '36:dec_0123456789abcdef0123456789abcdef' || pg_catalog.chr(10)
        || '64:a805880b9bf33fbd9747fbd68e0784f7566ab06af8a04ae97bcd9408c69de8f3'
        || pg_catalog.chr(10)
        || '64:2ea16988ca9a3b973ff11693e6de4bd078775655cd6715c5a06a120f71b3e827'
        || pg_catalog.chr(10)
        || '64:000960b66c33a1b60730a47fc053e69bd76d381036406464916ccf226baa8a43'
        || pg_catalog.chr(10)
        || '64:4bdb64a7842bf5f7a8052e63fae9290530ada44b1c8bdbcaa71deeb329517e6c'
        || pg_catalog.chr(10)
        || '44:correlacion_11111111111111111111111111111111'
        || pg_catalog.chr(10)
        || '26:aut_0123456789abcdefghijkl' || pg_catalog.chr(10)
        || '64:' || pg_catalog.repeat('1', 64) || pg_catalog.chr(10)
        || '26:ses_0123456789abcdefghijkl' || pg_catalog.chr(10)
        || '26:cse_0123456789abcdefghijkl' || pg_catalog.chr(10)
        || '1:2' || pg_catalog.chr(10)
        || '64:' || pg_catalog.repeat('2', 64) || pg_catalog.chr(10)
        || '27:per_0123456789abcdefghijkla' || pg_catalog.chr(10)
        || '27:prf_0123456789abcdefghijkla' || pg_catalog.chr(10)
        || '1:5' || pg_catalog.chr(10)
        || '31:organizacion:diputacion-granada' || pg_catalog.chr(10)
        || '12:organizacion' || pg_catalog.chr(10)
        || '31:organizacion:diputacion-granada' || pg_catalog.chr(10)
        || '42:contratacion_temporal.expediente.consultar'
        || pg_catalog.chr(10)
        || '44:tramitacion_expediente_contratacion_temporal'
        || pg_catalog.chr(10)
        || '26:expediente:rrhh:minimizado' || pg_catalog.chr(10)
        || '1:4' || pg_catalog.chr(10) || '1:1' || pg_catalog.chr(10)
        || '64:97b2d440c764090e452e51fb3623900a2cac78d97f337a42437b599ec6335e9b'
        || pg_catalog.chr(10)
        || '64:1c21303d3904a4e7de6f6b5aac2ff0c3086bafe68dee9037b95d74515f3bcf26'
        || pg_catalog.chr(10) || '0:' || pg_catalog.chr(10)
        || '27:2026-07-26T08:00:01.000000Z' || pg_catalog.chr(10),
        'UTF8'
    );
    IF v_canon IS DISTINCT FROM v_esperado
       OR pg_catalog.octet_length(v_canon) <> 1592
       OR pg_catalog.encode(pg_catalog.sha256(v_canon), 'hex') <>
          '6eb2257f9b385b6e49df904a38b714f6f68b457b2c8b76efe4ad8097bc3b8b9f'
    THEN
        RAISE EXCEPTION 'vector de recibo detalle divergente';
    END IF;

    v_cuadro := ROW(
        'vec.contratacion-temporal.recibo-acceso-rrhh.o4-05.v2',
        'acceso:rrhh:f996e98f214fcc5c252569b61e95bb8c',
        2, pg_catalog.repeat('1', 64), pg_catalog.repeat('2', 64),
        pg_catalog.repeat('3', 64), pg_catalog.repeat('4', 64),
        '2026-07-26T08:00:01Z',
        'auditoria:vec:0000000000000001', pg_catalog.repeat('a', 64),
        pg_catalog.repeat('d', 64),
        'dec_0123456789abcdef0123456789abcdef',
        '9a4c8622daa96db4b087153836318b44da6a0f977d8a0752fb54da073a9c02a6',
        '2ea16988ca9a3b973ff11693e6de4bd078775655cd6715c5a06a120f71b3e827',
        '01d0fe86954a5f649088a2b03f1a0f7ba7cb2aa87b0016a6348ac204bf9e2dfc',
        '9d4a4f77529936b1b786ce0716ac113e3afb96800c1a6a96ef39999c931f41f5',
        'correlacion_11111111111111111111111111111111',
        'aut_0123456789abcdefghijkl', pg_catalog.repeat('1', 64),
        'ses_0123456789abcdefghijkl',
        'cse_0123456789abcdefghijkl', 2, pg_catalog.repeat('2', 64),
        'per_0123456789abcdefghijkla',
        'prf_0123456789abcdefghijkla', 5,
        'organizacion:diputacion-granada', 'organizacion',
        'organizacion:diputacion-granada',
        'contratacion_temporal.cuadro.consultar',
        'gestion_operativa_contratacion_temporal',
        '', 0, 1,
        'acf18cd8e268f93f451f6ef6566e617c9128ba84c068b5b3124132f5f1ef5f07',
        'e77c8c791e996bd33783716a65cf6ad36868ea03f916c368263ea1031d2a50be',
        'af9613760f72635fbdb44a5a0a63c39f12af30f950a6ee5c971be188e89c4051',
        '2026-07-26T08:00:00Z'
    );
    v_canon :=
        vec_contratacion_temporal.canon_recibo_lectura_rrhh_v2(v_cuadro);
    IF pg_catalog.octet_length(v_canon) <> 1685
       OR pg_catalog.encode(pg_catalog.sha256(v_canon), 'hex') <>
          '19793c6b478daa8649bf0ca4020cd3ce035a986a4ad204d037f3890a30773f63'
    THEN
        RAISE EXCEPTION 'vector de recibo cuadro divergente';
    END IF;

    v_base := pg_catalog.to_jsonb(v_cuadro);
    FOR v_clave IN
        SELECT clave
          FROM pg_catalog.jsonb_object_keys(v_base) AS claves(clave)
         ORDER BY clave COLLATE "C"
    LOOP
        v_mutada := pg_catalog.jsonb_populate_record(
            NULL::vec_contratacion_temporal
                .evidencia_recibo_lectura_rrhh_v2,
            v_base - v_clave
        );
        BEGIN
            PERFORM
                vec_contratacion_temporal.canon_recibo_lectura_rrhh_v2(
                    v_mutada
                );
            RAISE EXCEPTION 'atributo de recibo omitido aceptado: %',
                v_clave;
        EXCEPTION WHEN SQLSTATE '22023' THEN
            IF SQLERRM <> 'recibo de lectura RRHH inválido' THEN
                RAISE;
            END IF;
        END;
    END LOOP;

    -- Sustituye individualmente cada atributo por un valor no nulo inválido.
    -- Así se acredita que los 38 atributos se validan y no solo su presencia.
    FOR v_clave IN
        SELECT clave
          FROM pg_catalog.jsonb_object_keys(v_base) AS claves(clave)
         ORDER BY clave COLLATE "C"
    LOOP
        v_valor_sustituto := CASE
            WHEN v_clave = ANY(ARRAY[
                'secuencia', 'control_sesion_revision', 'perfil_version',
                'version_expediente', 'total'
            ]::text[]) THEN '-1'::jsonb
            WHEN v_clave = ANY(ARRAY[
                'registrada_en', 'generada_en'
            ]::text[]) THEN '"infinity"'::jsonb
            ELSE '"!"'::jsonb
        END;
        BEGIN
            v_mutada := pg_catalog.jsonb_populate_record(
                NULL::vec_contratacion_temporal
                    .evidencia_recibo_lectura_rrhh_v2,
                pg_catalog.jsonb_set(
                    v_base, ARRAY[v_clave], v_valor_sustituto, false
                )
            );
            PERFORM
                vec_contratacion_temporal.canon_recibo_lectura_rrhh_v2(
                    v_mutada
                );
            RAISE EXCEPTION 'sustitución de recibo aceptada: %',
                v_clave;
        EXCEPTION WHEN SQLSTATE '22023' THEN
            IF SQLERRM <> 'recibo de lectura RRHH inválido' THEN
                RAISE;
            END IF;
        END;
        v_mutaciones := v_mutaciones + 1;
    END LOOP;
    IF (
        SELECT pg_catalog.count(*)
          FROM pg_catalog.jsonb_object_keys(v_base)
    ) <> 38 THEN
        RAISE EXCEPTION 'el vector de recibo no contiene 38 atributos';
    END IF;
    IF v_mutaciones <> 38 THEN
        RAISE EXCEPTION 'matriz de sustituciones incompleta: %',
            v_mutaciones;
    END IF;
END
$recibos$;

RESET ROLE;
