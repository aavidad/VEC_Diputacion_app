\set ON_ERROR_STOP 1
BEGIN;
SET LOCAL ROLE vec_ejecucion_documental_v4_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

SELECT EXISTS (
    SELECT 1
      FROM vec_ejecucion_documental_v4.orden_generacion_documental AS orden
      JOIN vec_ejecucion_documental_v4.atestacion_pdp AS atestacion
        ON atestacion.decision_ref = orden.decision_ref
       AND atestacion.efecto_ref = orden.efecto_ref
      JOIN vec_ejecucion_documental_v4.consumo_decision_atomico AS consumo
        ON consumo.decision_ref = orden.decision_ref
       AND consumo.efecto_ref = orden.efecto_ref
       AND consumo.orden_ref = orden.orden_ref
      JOIN vec_ejecucion_documental_v4.consumo_capacidad AS capacidad
        ON capacidad.decision_ref = orden.decision_ref
     WHERE orden.estado = 'pendiente_generacion'
) AS autoridad_disponible \gset

DO $estructura$
DECLARE
    tabla text;
    definicion text;
    seguridad_definidor boolean;
    configuracion text[];
    propietario text;
BEGIN
    IF to_regprocedure(
           'vec_ejecucion_documental_v4.registrar_autoridad_objeto_esperado_v1(numeric,bytea)'
       ) IS NULL THEN
        RAISE EXCEPTION 'falta la operacion cerrada de registro';
    END IF;
    SELECT prosrc, prosecdef, proconfig, pg_get_userbyid(proowner)
      INTO STRICT definicion, seguridad_definidor, configuracion, propietario
      FROM pg_catalog.pg_proc
     WHERE oid = to_regprocedure(
         'vec_ejecucion_documental_v4.registrar_autoridad_objeto_esperado_v1(numeric,bytea)'
     );
    IF propietario <> 'vec_ejecucion_documental_v4_propietario'
       OR NOT seguridad_definidor
       OR NOT ('search_path=pg_catalog, pg_temp' = ANY (configuracion))
       OR strpos(definicion, 'texto_original::json') = 0
       OR strpos(definicion, 'json_object_keys(documento_json)') = 0
       OR strpos(definicion,
          'recibo_material_v2_coteja_autoridad_objeto_v1') = 0
       OR strpos(definicion, 'FOR SHARE OF orden, atestacion, consumo, capacidad') = 0
       OR strpos(definicion, 'huella_efecto_sha256') = 0 THEN
        RAISE EXCEPTION 'la funcion no conserva sus fronteras estructurales';
    END IF;
    IF has_function_privilege(
           'public',
           'vec_ejecucion_documental_v4.registrar_autoridad_objeto_esperado_v1(numeric,bytea)',
           'EXECUTE'
       ) OR has_function_privilege(
           'vec_ejecucion_documental_v4_ejecutor_atestado',
           'vec_ejecucion_documental_v4.registrar_autoridad_objeto_esperado_v1(numeric,bytea)',
           'EXECUTE'
       ) OR has_function_privilege(
           'vec_ejecucion_documental_v4_emisor_capacidad',
           'vec_ejecucion_documental_v4.registrar_autoridad_objeto_esperado_v1(numeric,bytea)',
           'EXECUTE'
       ) THEN
        RAISE EXCEPTION 'un runtime puede fabricar la proyeccion SQL';
    END IF;
    FOREACH tabla IN ARRAY ARRAY[
        'registro_autoridad_objeto_esperado_v1',
        'auditoria_autoridad_objeto_v1',
        'outbox_autoridad_objeto_v1'
    ] LOOP
        IF has_table_privilege(
               'vec_ejecucion_documental_v4_ejecutor_atestado',
               'vec_ejecucion_documental_v4.' || tabla,
               'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER'
           ) OR has_table_privilege(
               'vec_ejecucion_documental_v4_emisor_capacidad',
               'vec_ejecucion_documental_v4.' || tabla,
               'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER'
           ) OR NOT EXISTS (
               SELECT 1
                 FROM pg_catalog.pg_class AS clase
                WHERE clase.oid = (
                    'vec_ejecucion_documental_v4.' || tabla
                )::regclass
                  AND clase.relrowsecurity
                  AND clase.relforcerowsecurity
           ) THEN
            RAISE EXCEPTION 'ACL o RLS invalida en %', tabla;
        END IF;
    END LOOP;
    IF EXISTS (
        SELECT 1
          FROM information_schema.columns
         WHERE table_schema = 'vec_ejecucion_documental_v4'
           AND table_name = 'registro_autoridad_objeto_esperado_v1'
           AND column_name IN (
               'atestacion_codigo', 'atestacion_codigo_hex',
               'proyeccion_json', 'sobre_cose_sign1'
           )
    ) THEN
        RAISE EXCEPTION 'se persiste codigo, JSON libre o sobre completo';
    END IF;
END
$estructura$;

\if :autoridad_disponible
CREATE TEMP TABLE entrada_autoridad_objeto_v1 (
    proyeccion bytea NOT NULL
) ON COMMIT DROP;

CREATE FUNCTION pg_temp.tlv_v2(p_etiqueta integer, p_valor bytea)
RETURNS bytea
LANGUAGE sql
IMMUTABLE
STRICT
SET search_path = pg_catalog, pg_temp
AS $funcion$
    SELECT int2send(p_etiqueta::smallint)
        || int8send(octet_length(p_valor)::bigint) || p_valor
$funcion$;

DO $fixture$
DECLARE
    autoridad record;
    recibo bytea := ''::bytea;
    documento jsonb;
    recibo_ref text;
    conector text := 'conector_material_prueba_v2';
    objeto text;
    objeto_version text := 'version_material_1';
    codigo bytea := decode(repeat('ab', 32), 'hex');
    huella_a text := repeat('a', 64);
    huella_b text := repeat('b', 64);
    huella_c text := repeat('c', 64);
    huella_d text := repeat('d', 64);
    huella_e text := repeat('e', 64);
    huella_f text := repeat('f', 64);
    huella_1 text := repeat('1', 64);
    huella_2 text := repeat('2', 64);
BEGIN
    SELECT orden.orden_ref, orden.efecto_ref,
           atestacion.aplicacion_registro
      INTO STRICT autoridad
      FROM vec_ejecucion_documental_v4.orden_generacion_documental AS orden
      JOIN vec_ejecucion_documental_v4.atestacion_pdp AS atestacion
        ON atestacion.decision_ref = orden.decision_ref
       AND atestacion.efecto_ref = orden.efecto_ref
      JOIN vec_ejecucion_documental_v4.consumo_decision_atomico AS consumo
        ON consumo.decision_ref = orden.decision_ref
       AND consumo.efecto_ref = orden.efecto_ref
       AND consumo.orden_ref = orden.orden_ref
      JOIN vec_ejecucion_documental_v4.consumo_capacidad AS capacidad
        ON capacidad.decision_ref = orden.decision_ref
     WHERE orden.estado = 'pendiente_generacion'
     ORDER BY orden.registrada_en DESC
     LIMIT 1;
    recibo_ref := 'recibo_material_' || substr(
        encode(sha256(convert_to(autoridad.efecto_ref, 'UTF8')), 'hex'), 1, 40
    );
    objeto := 'objeto_material_' || substr(
        encode(sha256(convert_to(recibo_ref, 'UTF8')), 'hex'), 1, 40
    );
    recibo := recibo
        || pg_temp.tlv_v2(0, convert_to(
            'vec.almacen.recibo-escritura-material.v2', 'UTF8'))
        || pg_temp.tlv_v2(1, decode('0002', 'hex'))
        || pg_temp.tlv_v2(2, convert_to(recibo_ref, 'UTF8'))
        || pg_temp.tlv_v2(3, convert_to(conector, 'UTF8'))
        || pg_temp.tlv_v2(4, convert_to('perfil_material_prueba_v2', 'UTF8'))
        || pg_temp.tlv_v2(5, int4send(1))
        || pg_temp.tlv_v2(6, decode(huella_a, 'hex'))
        || pg_temp.tlv_v2(7, convert_to(
            autoridad.aplicacion_registro ->> 'modulo_id', 'UTF8'))
        || pg_temp.tlv_v2(8, convert_to('vec.documentos.generar', 'UTF8'))
        || pg_temp.tlv_v2(9, convert_to('escribir', 'UTF8'))
        || pg_temp.tlv_v2(10, convert_to(
            autoridad.aplicacion_registro ->> 'recurso_ref', 'UTF8'))
        || pg_temp.tlv_v2(11, convert_to('operacion_material_prueba', 'UTF8'))
        || pg_temp.tlv_v2(12, convert_to('carga_material_prueba', 'UTF8'))
        || pg_temp.tlv_v2(13, convert_to(autoridad.efecto_ref, 'UTF8'))
        || pg_temp.tlv_v2(14, convert_to('plan_material_prueba_v2', 'UTF8'))
        || pg_temp.tlv_v2(15, int4send(1))
        || pg_temp.tlv_v2(16, decode(huella_b, 'hex'))
        || pg_temp.tlv_v2(17, decode(huella_c, 'hex'))
        || pg_temp.tlv_v2(18, convert_to('restringida', 'UTF8'))
        || pg_temp.tlv_v2(19, convert_to(objeto, 'UTF8'))
        || pg_temp.tlv_v2(20, convert_to(objeto_version, 'UTF8'))
        || pg_temp.tlv_v2(21, convert_to('admitida', 'UTF8'))
        || pg_temp.tlv_v2(22, convert_to('application/pdf', 'UTF8'))
        || pg_temp.tlv_v2(23, int8send(1024))
        || pg_temp.tlv_v2(24, decode(huella_d, 'hex'))
        || pg_temp.tlv_v2(25, convert_to('evidencia_material_prueba', 'UTF8'))
        || pg_temp.tlv_v2(26, int8send(1788163200123456))
        || pg_temp.tlv_v2(27, decode('00', 'hex'))
        || pg_temp.tlv_v2(29, convert_to('no_inmovilizado', 'UTF8'))
        || pg_temp.tlv_v2(30, convert_to('activo', 'UTF8'));
    documento := jsonb_build_object(
        'esquema', 'vec.documentos.autoridad-objeto-esperado.v1',
        'version', 1,
        'recibo_material_ref', recibo_ref,
        'huella_recibo_material_sha256', encode(sha256(recibo), 'hex'),
        'huella_declaracion_v4_sha256', huella_e,
        'objeto_ref', objeto,
        'objeto_version', objeto_version,
        'conector_id', conector,
        'efecto_ref', autoridad.efecto_ref,
        'huella_plan_efecto_sha256', huella_f,
        'huella_manifiesto_sha256', huella_1,
        'paso_ref', '01_custodiar_documento_firmado',
        'huella_paso_sha256', huella_2,
        'recibo_material_canonico_hex', encode(recibo, 'hex'),
        'atestacion_algoritmo', 'hmac-sha-256',
        'atestacion_clave_ref', 'clave_material_prueba_v2',
        'atestacion_clave_version', 1,
        'atestacion_dominio', 'recibo-escritura-objeto-material-v2',
        'atestacion_codigo_hex', encode(codigo, 'hex')
    );
    IF huella_e = encode(sha256(convert_to(documento::text, 'UTF8')), 'hex') THEN
        RAISE EXCEPTION 'el fixture reutilizo una huella de dominio';
    END IF;
    INSERT INTO entrada_autoridad_objeto_v1(proyeccion)
    VALUES (convert_to(documento::text, 'UTF8'));
END
$fixture$;

\if :{?solo_registro}
DO $registro_concurrente$
DECLARE
    respuesta record;
BEGIN
    SELECT registro.* INTO STRICT respuesta
      FROM entrada_autoridad_objeto_v1 AS entrada
      CROSS JOIN LATERAL
        vec_ejecucion_documental_v4.registrar_autoridad_objeto_esperado_v1(
            0, entrada.proyeccion
        ) AS registro;
    IF respuesta.resultado NOT IN ('registrada', 'repetida')
       OR respuesta.estado <> 'autoridad_objeto_registrada'
       OR respuesta.version_estado <> 1 THEN
        RAISE EXCEPTION 'el registro concurrente no convergio';
    END IF;
END
$registro_concurrente$;
\if :{?retener_registro}
SELECT pg_sleep(3);
\endif
\else
DO $matriz$
DECLARE
    entrada bytea;
    texto text;
    documento jsonb;
    primera record;
    repetida record;
    candidata bytea;
    rechazadas integer := 0;
    antes record;
    despues record;
BEGIN
    SELECT proyeccion INTO STRICT entrada FROM entrada_autoridad_objeto_v1;
    texto := convert_from(entrada, 'UTF8');
    documento := texto::jsonb;
    SELECT * INTO STRICT primera
      FROM vec_ejecucion_documental_v4.registrar_autoridad_objeto_esperado_v1(
          0, entrada
      );
    SELECT * INTO STRICT repetida
      FROM vec_ejecucion_documental_v4.registrar_autoridad_objeto_esperado_v1(
          0, entrada
      );
    IF repetida.resultado <> 'repetida'
       OR repetida.registro_ref <> primera.registro_ref
       OR repetida.estado <> primera.estado
       OR repetida.version_estado <> primera.version_estado
       OR repetida.auditoria_ref <> primera.auditoria_ref
       OR repetida.evento_outbox_ref <> primera.evento_outbox_ref
       OR repetida.huella_proyeccion_sha256 <>
          primera.huella_proyeccion_sha256
       OR repetida.registrada_en <> primera.registrada_en THEN
        RAISE EXCEPTION 'el replay exacto no devolvio el mismo recibo';
    END IF;
    SELECT count(*) AS registros,
           (SELECT count(*) FROM
               vec_ejecucion_documental_v4.auditoria_autoridad_objeto_v1
           ) AS auditorias,
           (SELECT count(*) FROM
               vec_ejecucion_documental_v4.outbox_autoridad_objeto_v1
           ) AS eventos
      INTO antes
      FROM vec_ejecucion_documental_v4.registro_autoridad_objeto_esperado_v1;

    FOREACH candidata IN ARRAY ARRAY[
        convert_to((jsonb_set(documento, '{objeto_ref}',
            '"objeto_material_ajeno"'))::text, 'UTF8'),
        convert_to((jsonb_set(documento, '{huella_paso_sha256}',
            to_jsonb(repeat('0', 64))))::text, 'UTF8'),
        convert_to((documento || jsonb_build_object('extra', true))::text,
            'UTF8'),
        convert_to((documento - 'conector_id')::text, 'UTF8'),
        convert_to((jsonb_set(documento, '{version}', '"1"'))::text,
            'UTF8'),
        convert_to((jsonb_set(documento, '{atestacion_codigo_hex}',
            '"ab"'))::text, 'UTF8'),
        convert_to((jsonb_set(documento, '{atestacion_codigo_hex}',
            to_jsonb(repeat('cd', 32))))::text, 'UTF8'),
        convert_to(' ' || texto, 'UTF8'),
        convert_to(overlay(texto placing '"esquema":"duplicado",'
            from 2 for 0), 'UTF8'),
        convert_to((jsonb_set(documento, '{efecto_ref}',
            '"efecto_material_ajeno"'))::text, 'UTF8')
    ] LOOP
        BEGIN
            PERFORM * FROM
                vec_ejecucion_documental_v4.registrar_autoridad_objeto_esperado_v1(
                    0, candidata
                );
        EXCEPTION
            WHEN invalid_parameter_value OR check_violation
                OR serialization_failure THEN
                rechazadas := rechazadas + 1;
        END;
    END LOOP;
    BEGIN
        PERFORM * FROM
            vec_ejecucion_documental_v4.registrar_autoridad_objeto_esperado_v1(
                1, entrada
            );
    EXCEPTION WHEN invalid_parameter_value THEN
        rechazadas := rechazadas + 1;
    END;
    BEGIN
        PERFORM * FROM
            vec_ejecucion_documental_v4.registrar_autoridad_objeto_esperado_v1(
                0, convert_to(repeat('x', 196609), 'UTF8')
            );
    EXCEPTION WHEN invalid_parameter_value THEN
        rechazadas := rechazadas + 1;
    END;
    IF rechazadas <> 12 THEN
        RAISE EXCEPTION 'la matriz adversarial acepto % mutaciones',
            12 - rechazadas;
    END IF;
    SELECT count(*) AS registros,
           (SELECT count(*) FROM
               vec_ejecucion_documental_v4.auditoria_autoridad_objeto_v1
           ) AS auditorias,
           (SELECT count(*) FROM
               vec_ejecucion_documental_v4.outbox_autoridad_objeto_v1
           ) AS eventos
      INTO despues
      FROM vec_ejecucion_documental_v4.registro_autoridad_objeto_esperado_v1;
    IF antes IS DISTINCT FROM despues
       OR despues.registros <> 1 OR despues.auditorias <> 1
       OR despues.eventos <> 1 THEN
        RAISE EXCEPTION 'un rechazo dejo estado parcial';
    END IF;
END
$matriz$;

DO $persistencia$
DECLARE
    registro record;
    control record;
    rechazada boolean;
BEGIN
    SELECT r.* INTO STRICT registro
      FROM vec_ejecucion_documental_v4.registro_autoridad_objeto_esperado_v1 AS r;
    IF registro.esquema <> 'vec.documentos.autoridad-objeto-esperado.v1'
       OR registro.version_contrato <> 1
       OR registro.huella_declaracion_v4_sha256 <> repeat('e', 64)
       OR registro.huella_plan_efecto_sha256 <> repeat('f', 64)
       OR registro.huella_manifiesto_sha256 <> repeat('1', 64)
       OR registro.huella_paso_sha256 <> repeat('2', 64)
       OR registro.paso_ref <> '01_custodiar_documento_firmado'
       OR registro.conector_id <> 'conector_material_prueba_v2'
       OR encode(sha256(registro.recibo_material_canonico), 'hex') <>
           registro.huella_recibo_material_sha256
       OR registro.huella_proyeccion_sha256 <> (
           SELECT encode(sha256(proyeccion), 'hex')
             FROM entrada_autoridad_objeto_v1
       )
       OR registro.atestacion_algoritmo <> 'hmac-sha-256'
       OR registro.atestacion_clave_ref <> 'clave_material_prueba_v2'
       OR registro.atestacion_clave_version <> 1
       OR registro.atestacion_dominio <>
          'recibo-escritura-objeto-material-v2'
       OR registro.atestacion_codigo_longitud <> 32
       OR registro.huella_atestacion_codigo_sha256 <>
          encode(sha256(decode(repeat('ab', 32), 'hex')), 'hex')
       OR registro.huella_sello_atestacion_sha256 <> encode(sha256(
           vec_ejecucion_documental_v4.encuadrar_autoridad_objeto_v1(
               convert_to(
                   'vec.documentos.sello-atestacion-autoridad-objeto-esperado.v1',
                   'UTF8'
               )
           ) || vec_ejecucion_documental_v4.encuadrar_autoridad_objeto_v1(
               registro.recibo_material_canonico
           ) || vec_ejecucion_documental_v4.encuadrar_autoridad_objeto_v1(
               convert_to(registro.atestacion_algoritmo, 'UTF8')
           ) || vec_ejecucion_documental_v4.encuadrar_autoridad_objeto_v1(
               convert_to(registro.atestacion_clave_ref, 'UTF8')
           ) || vec_ejecucion_documental_v4.encuadrar_autoridad_objeto_v1(
               decode(lpad(to_hex(
                   registro.atestacion_clave_version::bigint
               ), 8, '0'), 'hex')
           ) || vec_ejecucion_documental_v4.encuadrar_autoridad_objeto_v1(
               convert_to(registro.atestacion_dominio, 'UTF8')
           ) || vec_ejecucion_documental_v4.encuadrar_autoridad_objeto_v1(
               decode(repeat('ab', 32), 'hex')
           )
       ), 'hex')
       OR registro.huella_efecto_v4_sha256 = repeat('0', 64) THEN
        RAISE EXCEPTION 'faltan compromisos o minimizacion durable';
    END IF;
    IF NOT EXISTS (
        SELECT 1
          FROM vec_ejecucion_documental_v4.orden_generacion_documental AS o
          JOIN vec_ejecucion_documental_v4.atestacion_pdp AS a
            ON a.decision_ref = o.decision_ref
           AND a.efecto_ref = o.efecto_ref
          JOIN vec_ejecucion_documental_v4.consumo_capacidad AS c
            ON c.decision_ref = o.decision_ref
         WHERE o.orden_ref = registro.orden_ref
           AND o.huella_plan_sha256 =
               registro.huella_plan_autorizacion_sha256
           AND a.huella_plan_sha256 =
               registro.huella_plan_autorizacion_sha256
           AND c.capacidad ->> 'huella_efecto_sha256' =
               registro.huella_efecto_v4_sha256
    ) THEN
        RAISE EXCEPTION 'los compromisos P0-1 no proceden de sus autoridades';
    END IF;
    SELECT c.* INTO STRICT control
      FROM vec_ejecucion_documental_v4.control_cadena_auditoria AS c
      JOIN vec_ejecucion_documental_v4.auditoria_autoridad_objeto_v1 AS a
        ON a.secuencia = c.ultima_secuencia
       AND a.huella_registro_sha256 = c.ultima_huella_sha256
     WHERE c.control_id = true;
    rechazada := false;
    BEGIN
        UPDATE vec_ejecucion_documental_v4.registro_autoridad_objeto_esperado_v1
           SET version_estado = 2;
    EXCEPTION WHEN SQLSTATE '55000' THEN
        rechazada := true;
    END;
    IF NOT rechazada THEN
        RAISE EXCEPTION 'el estado terminal pudo mutarse';
    END IF;
    rechazada := false;
    BEGIN
        DELETE FROM vec_ejecucion_documental_v4.auditoria_autoridad_objeto_v1;
    EXCEPTION WHEN SQLSTATE '55000' THEN
        rechazada := true;
    END;
    IF NOT rechazada THEN
        RAISE EXCEPTION 'la auditoria pudo borrarse';
    END IF;
    rechazada := false;
    BEGIN
        TRUNCATE vec_ejecucion_documental_v4.outbox_autoridad_objeto_v1;
    EXCEPTION WHEN SQLSTATE '55000' THEN
        rechazada := true;
    END;
    IF NOT rechazada THEN
        RAISE EXCEPTION 'el outbox pudo truncarse';
    END IF;
END
$persistencia$;
\endif
\else
\if :{?exigir_autoridad}
DO $autoridad_obligatoria$
BEGIN
    RAISE EXCEPTION 'falta una orden V4 para la matriz de autoridad de objeto';
END
$autoridad_obligatoria$;
\endif
\endif
COMMIT;
