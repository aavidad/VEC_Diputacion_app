\set ON_ERROR_STOP 1
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

SELECT EXISTS (
    SELECT 1
      FROM vec_ejecucion_documental_v4.orden_generacion_documental AS orden
      JOIN vec_ejecucion_documental_v4.atestacion_pdp AS atestacion
        ON atestacion.decision_ref = orden.decision_ref
       AND atestacion.efecto_ref = orden.efecto_ref
      JOIN vec_ejecucion_documental_v4.consumo_decision_atomico AS decision
        ON decision.decision_ref = orden.decision_ref
       AND decision.efecto_ref = orden.efecto_ref
       AND decision.orden_ref = orden.orden_ref
      JOIN vec_ejecucion_documental_v4.consumo_capacidad AS capacidad
        ON capacidad.decision_ref = orden.decision_ref
      JOIN vec_ejecucion_documental_v4.clave_capacidad_actual AS actual
        ON actual.control_id = true AND actual.clave_id = capacidad.clave_id
       AND actual.version = capacidad.version
     WHERE orden.estado = 'pendiente_generacion'
       AND (capacidad.expira_en > clock_timestamp() OR EXISTS (
           SELECT 1
             FROM vec_ejecucion_documental_v4.reserva_efecto_generacion_documental_v1 AS reserva
            WHERE reserva.orden_ref = orden.orden_ref
              AND reserva.decision_ref = orden.decision_ref
              AND reserva.efecto_ref = orden.efecto_ref
       ))
) AS autoridad_disponible \gset

\if :autoridad_disponible
CREATE TEMP TABLE entrada_efectos_documentales_v1 (
    contexto jsonb NOT NULL, contexto_fabricado jsonb NOT NULL,
    contexto_expirado jsonb NOT NULL, contexto_consumido jsonb NOT NULL,
    manifiesto jsonb NOT NULL
) ON COMMIT DROP;

CREATE FUNCTION pg_temp.huella_plan_efecto_v1(p_contexto jsonb, p_pasos jsonb)
RETURNS text LANGUAGE plpgsql IMMUTABLE SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE canonico bytea := ''::bytea; valor text; paso jsonb;
BEGIN
    FOREACH valor IN ARRAY ARRAY[
        p_contexto ->> 'esquema', p_contexto ->> 'autorizacion_ref',
        p_contexto ->> 'huella_decision_sha256', p_contexto ->> 'accion_negocio',
        p_contexto ->> 'recurso_ref', p_contexto ->> 'huella_recurso_sha256',
        p_contexto ->> 'finalidad', p_contexto ->> 'correlacion_ref',
        p_contexto ->> 'operacion_ref', p_contexto ->> 'carga_ref',
        p_contexto ->> 'clasificacion', p_contexto ->> 'sujeto_seudonimo_hmac',
        p_contexto ->> 'huella_solicitud_hmac', p_contexto ->> 'efecto_ref', '', '',
        p_contexto ->> 'huella_manifiesto_sha256'
    ] LOOP
        canonico := canonico ||
            vec_ejecucion_documental_v4.encuadrar_capacidad(valor);
    END LOOP;
    FOR paso IN SELECT valor FROM jsonb_array_elements(p_pasos) AS valor LOOP
        canonico := canonico ||
            vec_ejecucion_documental_v4.encuadrar_capacidad(paso ->> 'paso_ref') ||
            vec_ejecucion_documental_v4.encuadrar_capacidad('escribir') ||
            vec_ejecucion_documental_v4.encuadrar_capacidad(
                paso ->> 'huella_paso_sha256');
    END LOOP;
    RETURN encode(sha256(canonico), 'hex');
END
$funcion$;

DO $fixture$
DECLARE
    autoridad record; aplicacion jsonb; pasos jsonb := '[]'::jsonb;
    paso jsonb; manifiesto jsonb; contexto jsonb; fabricado jsonb;
    expirado jsonb; consumido jsonb; canonico bytea; huella_paso text;
    huella_manifiesto text; indice integer;
    huella_plantilla text := repeat('a', 64);
    huella_solicitud text := 'hmac-sha256:solicitud:' || repeat('d', 64);
    sujeto text := 'hmac-sha256:sujeto:' || repeat('e', 64);
BEGIN
    SELECT orden.*, atestacion.aplicacion_registro
      INTO STRICT autoridad
      FROM vec_ejecucion_documental_v4.orden_generacion_documental AS orden
      JOIN vec_ejecucion_documental_v4.atestacion_pdp AS atestacion
        ON atestacion.decision_ref = orden.decision_ref
       AND atestacion.efecto_ref = orden.efecto_ref
      JOIN vec_ejecucion_documental_v4.consumo_decision_atomico AS decision
        ON decision.decision_ref = orden.decision_ref
       AND decision.efecto_ref = orden.efecto_ref
       AND decision.orden_ref = orden.orden_ref
      JOIN vec_ejecucion_documental_v4.consumo_capacidad AS capacidad
        ON capacidad.decision_ref = orden.decision_ref
      JOIN vec_ejecucion_documental_v4.clave_capacidad_actual AS actual
        ON actual.control_id = true AND actual.clave_id = capacidad.clave_id
       AND actual.version = capacidad.version
     WHERE orden.estado = 'pendiente_generacion'
       AND (capacidad.expira_en > clock_timestamp() OR EXISTS (
           SELECT 1
             FROM vec_ejecucion_documental_v4.reserva_efecto_generacion_documental_v1 AS reserva
            WHERE reserva.orden_ref = orden.orden_ref
              AND reserva.decision_ref = orden.decision_ref
              AND reserva.efecto_ref = orden.efecto_ref
       ))
     ORDER BY orden.registrada_en DESC LIMIT 1;
    aplicacion := autoridad.aplicacion_registro;
    FOR indice IN 1..2 LOOP
        paso := jsonb_build_object(
            'referencia_logica', CASE indice WHEN 1 THEN 'documento:prueba:docx'
                ELSE 'documento:prueba:pdf' END,
            'clave_idempotencia', CASE indice WHEN 1 THEN 'contenido:prueba:docx'
                ELSE 'contenido:prueba:pdf' END,
            'formato', CASE indice WHEN 1 THEN 'docx' ELSE 'pdf' END,
            'zona', 'admitida',
            'mime', CASE indice WHEN 1 THEN
                'application/vnd.openxmlformats-officedocument.wordprocessingml.document'
                ELSE 'application/pdf' END,
            'tamano', CASE indice WHEN 1 THEN 2048 ELSE 1024 END,
            'huella_sha256', repeat(indice::text, 64));
        canonico := vec_ejecucion_documental_v4.encuadrar_capacidad(
            'vec.documentos.manifiesto-generacion.paso.v1') ||
            vec_ejecucion_documental_v4.encuadrar_capacidad(huella_plantilla) ||
            vec_ejecucion_documental_v4.encuadrar_capacidad('vec.documentos.generar') ||
            vec_ejecucion_documental_v4.encuadrar_capacidad(paso ->> 'referencia_logica') ||
            vec_ejecucion_documental_v4.encuadrar_capacidad(paso ->> 'clave_idempotencia') ||
            vec_ejecucion_documental_v4.encuadrar_capacidad(paso ->> 'formato') ||
            vec_ejecucion_documental_v4.encuadrar_capacidad(paso ->> 'zona') ||
            vec_ejecucion_documental_v4.encuadrar_capacidad(paso ->> 'mime') ||
            vec_ejecucion_documental_v4.encuadrar_capacidad(
                (paso ->> 'tamano')::bigint::text) ||
            vec_ejecucion_documental_v4.encuadrar_capacidad(paso ->> 'huella_sha256');
        huella_paso := encode(sha256(canonico), 'hex');
        pasos := pasos || jsonb_build_array(paso || jsonb_build_object(
            'paso_ref', 'generar_documento_' || huella_paso,
            'huella_paso_sha256', huella_paso));
    END LOOP;
    canonico := vec_ejecucion_documental_v4.encuadrar_capacidad(
        'vec.documentos.manifiesto-generacion.v1') ||
        vec_ejecucion_documental_v4.encuadrar_capacidad('plantilla:prueba') ||
        vec_ejecucion_documental_v4.encuadrar_capacidad('1') ||
        vec_ejecucion_documental_v4.encuadrar_capacidad(aplicacion ->> 'modulo_id') ||
        vec_ejecucion_documental_v4.encuadrar_capacidad('informe_prueba') ||
        vec_ejecucion_documental_v4.encuadrar_capacidad(huella_plantilla) ||
        vec_ejecucion_documental_v4.encuadrar_capacidad('vec.documentos.generar') ||
        vec_ejecucion_documental_v4.encuadrar_capacidad('2');
    FOR paso IN SELECT valor FROM jsonb_array_elements(pasos) AS valor LOOP
        canonico := canonico ||
            vec_ejecucion_documental_v4.encuadrar_capacidad(paso ->> 'paso_ref') ||
            vec_ejecucion_documental_v4.encuadrar_capacidad(paso ->> 'huella_paso_sha256') ||
            vec_ejecucion_documental_v4.encuadrar_capacidad(paso ->> 'referencia_logica') ||
            vec_ejecucion_documental_v4.encuadrar_capacidad(paso ->> 'clave_idempotencia') ||
            vec_ejecucion_documental_v4.encuadrar_capacidad(paso ->> 'formato') ||
            vec_ejecucion_documental_v4.encuadrar_capacidad(paso ->> 'zona') ||
            vec_ejecucion_documental_v4.encuadrar_capacidad(paso ->> 'mime') ||
            vec_ejecucion_documental_v4.encuadrar_capacidad(
                (paso ->> 'tamano')::bigint::text) ||
            vec_ejecucion_documental_v4.encuadrar_capacidad(paso ->> 'huella_sha256');
    END LOOP;
    huella_manifiesto := encode(sha256(canonico), 'hex');
    manifiesto := jsonb_build_object(
        'esquema', 'vec.documentos.manifiesto-generacion.v1',
        'plantilla_id', 'plantilla:prueba', 'plantilla_version', 1,
        'modulo_id', aplicacion ->> 'modulo_id',
        'tipo_documental', 'informe_prueba',
        'huella_plantilla_sha256', huella_plantilla,
        'permiso_generar', 'vec.documentos.generar',
        'huella_manifiesto_sha256', huella_manifiesto, 'pasos', pasos);
    contexto := jsonb_build_object(
        'esquema', 'vec.almacen.contexto-operacion.v1',
        'operacion_ref', 'operacion:documental:prueba',
        'correlacion_ref', aplicacion ->> 'correlacion_ref',
        'autorizacion_ref', autoridad.decision_ref,
        'finalidad', aplicacion ->> 'finalidad', 'clasificacion', 'restringida',
        'accion_negocio', 'vec.documentos.generar', 'accion_tecnica', 'escribir',
        'carga_ref', 'carga:documental:prueba', 'sujeto_seudonimo_hmac', sujeto,
        'recurso_ref', aplicacion ->> 'recurso_ref',
        'modulo_id', aplicacion ->> 'modulo_id',
        'tipo_recurso', aplicacion ->> 'tipo_recurso',
        'huella_recurso_sha256', aplicacion ->> 'huella_recurso_sha256',
        'huella_solicitud_hmac', huella_solicitud,
        'efecto_ref', autoridad.efecto_ref,
        'huella_plan_efecto_sha256', repeat('0', 64),
        'huella_manifiesto_sha256', huella_manifiesto,
        'huella_paso_sha256', pasos -> 0 ->> 'huella_paso_sha256',
        'paso_ref', pasos -> 0 ->> 'paso_ref',
        'objeto_vinculado_ref', '', 'objeto_vinculado_version', '',
        'huella_decision_sha256', autoridad.huella_decision_sha256,
        'verificada_en', aplicacion ->> 'verificada_en',
        'valida_hasta', aplicacion ->> 'valida_hasta');
    contexto := jsonb_set(contexto, '{huella_plan_efecto_sha256}',
        to_jsonb(pg_temp.huella_plan_efecto_v1(contexto, pasos)));
    fabricado := contexto || jsonb_build_object(
        'autorizacion_ref', 'decision:documental:fabricada',
        'huella_decision_sha256', repeat('f', 64));
    fabricado := jsonb_set(fabricado, '{huella_plan_efecto_sha256}',
        to_jsonb(pg_temp.huella_plan_efecto_v1(fabricado, pasos)));
    expirado := jsonb_set(contexto, '{valida_hasta}', to_jsonb(to_char(
        (clock_timestamp() - interval '1 second') AT TIME ZONE 'UTC',
        'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')));
    consumido := jsonb_set(contexto, '{efecto_ref}',
        '"efecto:generacion-documental:consumido-ajeno"');
    consumido := jsonb_set(consumido, '{huella_plan_efecto_sha256}',
        to_jsonb(pg_temp.huella_plan_efecto_v1(consumido, pasos)));
    INSERT INTO entrada_efectos_documentales_v1 VALUES
        (contexto, fabricado, expirado, consumido, manifiesto);
END
$fixture$;

\if :{?solo_reserva}
DO $reserva_concurrente$
DECLARE respuesta record;
BEGIN
    SELECT r.* INTO STRICT respuesta
      FROM entrada_efectos_documentales_v1 AS e
      CROSS JOIN LATERAL
        vec_ejecucion_documental_v4.reservar_efecto_generacion_documental_v1(
            e.contexto, e.manifiesto) AS r;
    IF respuesta.resultado NOT IN ('reservada', 'repetida')
       OR jsonb_array_length(respuesta.pasos) <> 2 THEN
        RAISE EXCEPTION 'la reserva concurrente no produjo replay exacto';
    END IF;
END
$reserva_concurrente$;
\if :{?retener_reserva}
SELECT pg_sleep(3);
\endif
\else
DO $contrato$
DECLARE
    entrada record; reserva record; paso_uno jsonb; paso_dos jsonb;
    evidencia jsonb; contenido jsonb; confirmacion jsonb; marca jsonb;
    respuesta record; candidata jsonb; rechazadas integer := 0; ahora text;
BEGIN
    SELECT * INTO STRICT entrada FROM entrada_efectos_documentales_v1;
    SELECT r.* INTO STRICT reserva
      FROM vec_ejecucion_documental_v4.reservar_efecto_generacion_documental_v1(
          entrada.contexto, entrada.manifiesto) AS r;
    IF reserva.resultado <> 'repetida' OR NOT reserva.repetida
       OR jsonb_array_length(reserva.pasos) <> 2 THEN
        RAISE EXCEPTION 'el replay de reserva no fue exacto';
    END IF;
    FOREACH candidata IN ARRAY ARRAY[entrada.contexto_fabricado,
        entrada.contexto_expirado, entrada.contexto_consumido] LOOP
        BEGIN
            PERFORM * FROM
                vec_ejecucion_documental_v4.reservar_efecto_generacion_documental_v1(
                    candidata, entrada.manifiesto);
        EXCEPTION WHEN check_violation OR invalid_parameter_value THEN
            rechazadas := rechazadas + 1;
        END;
    END LOOP;
    IF rechazadas <> 3 THEN
        RAISE EXCEPTION 'se acepto decision fabricada, expirada o ya consumida';
    END IF;

    paso_uno := entrada.manifiesto -> 'pasos' -> 0;
    paso_dos := entrada.manifiesto -> 'pasos' -> 1;
    ahora := to_char(clock_timestamp() AT TIME ZONE 'UTC',
        'YYYY-MM-DD"T"HH24:MI:SS.US"Z"');
    evidencia := jsonb_build_object(
        'referencia', 'evidencia:almacen:prueba', 'conector_id', 'almacen:prueba',
        'esquema_contexto', entrada.contexto ->> 'esquema',
        'accion_negocio', entrada.contexto ->> 'accion_negocio',
        'accion', entrada.contexto ->> 'accion_tecnica',
        'efecto_ref', entrada.contexto ->> 'efecto_ref',
        'huella_plan_efecto_sha256', entrada.contexto ->> 'huella_plan_efecto_sha256',
        'huella_manifiesto_sha256', entrada.contexto ->> 'huella_manifiesto_sha256',
        'huella_paso_sha256', paso_uno ->> 'huella_paso_sha256',
        'paso_ref', paso_uno ->> 'paso_ref',
        'huella_decision_sha256', entrada.contexto ->> 'huella_decision_sha256',
        'objeto', jsonb_build_object('referencia', 'objeto:documental:prueba',
            'version', 'version:1'),
        'operacion_ref', entrada.contexto ->> 'operacion_ref',
        'correlacion_ref', entrada.contexto ->> 'correlacion_ref',
        'autorizacion_ref', entrada.contexto ->> 'autorizacion_ref',
        'finalidad', entrada.contexto ->> 'finalidad',
        'clasificacion', entrada.contexto ->> 'clasificacion',
        'realizada_en', ahora, 'carga_ref', entrada.contexto ->> 'carga_ref',
        'sujeto_seudonimo_hmac', entrada.contexto ->> 'sujeto_seudonimo_hmac',
        'recurso_ref', entrada.contexto ->> 'recurso_ref',
        'modulo_id', entrada.contexto ->> 'modulo_id',
        'huella_solicitud_hmac', entrada.contexto ->> 'huella_solicitud_hmac',
        'fundamento_ref', '', 'reintento_idempotente', false);
    contenido := jsonb_build_object(
        'referencia_logica', paso_uno ->> 'referencia_logica',
        'referencia', evidencia -> 'objeto' ->> 'referencia',
        'version', evidencia -> 'objeto' ->> 'version',
        'conector_id', evidencia ->> 'conector_id', 'zona', paso_uno ->> 'zona',
        'mime', paso_uno ->> 'mime', 'tamano', paso_uno -> 'tamano',
        'huella_sha256', paso_uno ->> 'huella_sha256',
        'evidencia_operacion', evidencia);
    confirmacion := jsonb_build_object(
        'esquema', 'vec.documentos.registro-efectos-generacion.confirmacion.v1',
        'reserva_ref', reserva.reserva_ref,
        'decision_ref', entrada.contexto ->> 'autorizacion_ref',
        'efecto_ref', entrada.contexto ->> 'efecto_ref',
        'huella_decision_sha256', entrada.contexto ->> 'huella_decision_sha256',
        'huella_plan_efecto_sha256', entrada.contexto ->> 'huella_plan_efecto_sha256',
        'huella_manifiesto_sha256', entrada.contexto ->> 'huella_manifiesto_sha256',
        'paso_ref', paso_uno ->> 'paso_ref',
        'huella_paso_sha256', paso_uno ->> 'huella_paso_sha256',
        'contenido_guardado', contenido);
    SELECT * INTO STRICT respuesta FROM
        vec_ejecucion_documental_v4.confirmar_paso_generacion_documental_v1(
            confirmacion);
    IF respuesta.resultado <> 'confirmada' THEN
        RAISE EXCEPTION 'no se confirmo el contrato canonico completo';
    END IF;
    SELECT * INTO STRICT respuesta FROM
        vec_ejecucion_documental_v4.confirmar_paso_generacion_documental_v1(
            confirmacion);
    IF respuesta.resultado <> 'repetida' THEN
        RAISE EXCEPTION 'la confirmacion exacta no fue idempotente';
    END IF;
    rechazadas := 0;
    FOR candidata IN SELECT valor FROM jsonb_array_elements(jsonb_build_array(
        jsonb_set(confirmacion, '{contenido_guardado,referencia_logica}', '"otra"'),
        jsonb_set(confirmacion, '{contenido_guardado,zona}', '"cuarentena"'),
        jsonb_set(confirmacion, '{contenido_guardado,mime}', '"text/plain"'),
        jsonb_set(confirmacion, '{contenido_guardado,tamano}', '1'),
        jsonb_set(confirmacion, '{contenido_guardado,huella_sha256}',
            to_jsonb(repeat('9', 64))),
        jsonb_set(confirmacion,
            '{contenido_guardado,evidencia_operacion,objeto,referencia}', '"objeto:ajeno"'),
        jsonb_set(confirmacion,
            '{contenido_guardado,evidencia_operacion,huella_decision_sha256}',
            to_jsonb(repeat('8', 64))),
        jsonb_set(confirmacion,
            '{contenido_guardado,evidencia_operacion,operacion_ref}', '"operacion:ajena"'),
        jsonb_set(confirmacion, '{contenido_guardado,version}',
            to_jsonb(repeat('v', 257))),
        jsonb_set(jsonb_set(confirmacion, '{contenido_guardado,referencia}',
            '"objeto:documental:sustituto"'),
            '{contenido_guardado,evidencia_operacion,objeto,referencia}',
            '"objeto:documental:sustituto"')
    )) AS valor LOOP
        BEGIN
            PERFORM * FROM
                vec_ejecucion_documental_v4.confirmar_paso_generacion_documental_v1(
                    candidata);
        EXCEPTION WHEN check_violation OR invalid_parameter_value THEN
            rechazadas := rechazadas + 1;
        END;
    END LOOP;
    IF rechazadas <> 10 THEN
        RAISE EXCEPTION 'se aceptaron terminales o ligaduras mutadas';
    END IF;

    marca := jsonb_build_object(
        'esquema', 'vec.documentos.registro-efectos-generacion.indeterminado.v1',
        'reserva_ref', reserva.reserva_ref,
        'decision_ref', entrada.contexto ->> 'autorizacion_ref',
        'efecto_ref', entrada.contexto ->> 'efecto_ref',
        'huella_decision_sha256', entrada.contexto ->> 'huella_decision_sha256',
        'huella_plan_efecto_sha256', entrada.contexto ->> 'huella_plan_efecto_sha256',
        'huella_manifiesto_sha256', entrada.contexto ->> 'huella_manifiesto_sha256',
        'paso_ref', paso_dos ->> 'paso_ref',
        'huella_paso_sha256', paso_dos ->> 'huella_paso_sha256',
        'incidente_ref', 'incidente:generacion-documental:prueba');
    SELECT * INTO STRICT respuesta FROM
        vec_ejecucion_documental_v4.marcar_paso_generacion_documental_indeterminado_v1(
            marca);
    IF respuesta.resultado <> 'marcada' THEN
        RAISE EXCEPTION 'no se marco el paso ambiguo';
    END IF;
    SELECT * INTO STRICT respuesta FROM
        vec_ejecucion_documental_v4.marcar_paso_generacion_documental_indeterminado_v1(
            marca);
    IF respuesta.resultado <> 'repetida' THEN
        RAISE EXCEPTION 'la marca indeterminada exacta no fue idempotente';
    END IF;
    candidata := confirmacion || jsonb_build_object(
        'paso_ref', paso_dos ->> 'paso_ref',
        'huella_paso_sha256', paso_dos ->> 'huella_paso_sha256');
    candidata := jsonb_set(candidata, '{contenido_guardado,referencia_logica}',
        to_jsonb(paso_dos ->> 'referencia_logica'));
    candidata := jsonb_set(candidata, '{contenido_guardado,mime}',
        to_jsonb(paso_dos ->> 'mime'));
    candidata := jsonb_set(candidata, '{contenido_guardado,tamano}', paso_dos -> 'tamano');
    candidata := jsonb_set(candidata, '{contenido_guardado,huella_sha256}',
        to_jsonb(paso_dos ->> 'huella_sha256'));
    candidata := jsonb_set(candidata,
        '{contenido_guardado,evidencia_operacion,paso_ref}',
        to_jsonb(paso_dos ->> 'paso_ref'));
    candidata := jsonb_set(candidata,
        '{contenido_guardado,evidencia_operacion,huella_paso_sha256}',
        to_jsonb(paso_dos ->> 'huella_paso_sha256'));
    rechazadas := 0;
    BEGIN
        PERFORM * FROM
            vec_ejecucion_documental_v4.confirmar_paso_generacion_documental_v1(
                candidata);
    EXCEPTION WHEN check_violation THEN rechazadas := rechazadas + 1;
    END;
    BEGIN
        PERFORM * FROM
            vec_ejecucion_documental_v4.marcar_paso_generacion_documental_indeterminado_v1(
                marca || jsonb_build_object(
                    'paso_ref', paso_uno ->> 'paso_ref',
                    'huella_paso_sha256', paso_uno ->> 'huella_paso_sha256'));
    EXCEPTION WHEN check_violation THEN rechazadas := rechazadas + 1;
    END;
    IF rechazadas <> 2 THEN
        RAISE EXCEPTION 'un terminal confirmado o indeterminado fue sustituido';
    END IF;
END
$contrato$;

DO $persistencia_y_acl$
DECLARE tabla text; rechazada boolean; definicion text;
BEGIN
    IF (SELECT count(*) FROM
        vec_ejecucion_documental_v4.reserva_efecto_generacion_documental_v1) <> 1
       OR (SELECT count(*) FROM
        vec_ejecucion_documental_v4.paso_efecto_generacion_documental_v1) <> 4
       OR (SELECT count(*) FROM
        vec_ejecucion_documental_v4.auditoria_efecto_generacion_documental_v1) <> 3
       OR (SELECT count(*) FROM
        vec_ejecucion_documental_v4.evento_outbox_efecto_generacion_documental_v1) <> 3
       OR NOT EXISTS (
           SELECT 1 FROM vec_ejecucion_documental_v4.reserva_efecto_generacion_documental_v1 AS r
           JOIN vec_ejecucion_documental_v4.consumo_decision_atomico AS d
             ON d.decision_ref = r.decision_ref AND d.efecto_ref = r.efecto_ref
            AND d.orden_ref = r.orden_ref
           JOIN vec_ejecucion_documental_v4.consumo_capacidad AS c
             ON c.clave_id = r.capacidad_clave_id AND c.version = r.capacidad_version
            AND c.nonce = r.capacidad_nonce
          WHERE r.tupla_canonica = jsonb_build_object(
              'esquema', 'vec.documentos.registro-efectos-generacion.tupla.v1',
              'orden_ref', r.orden_ref, 'decision_ref', r.decision_ref,
              'efecto_ref', r.efecto_ref,
              'huella_decision_sha256', r.huella_decision_sha256,
              'huella_plan_autorizacion_sha256', r.huella_plan_autorizacion_sha256,
              'huella_orden_sha256', r.huella_orden_sha256,
              'huella_capacidad_sha256', r.huella_capacidad_sha256,
              'huella_plan_efecto_sha256', r.huella_plan_efecto_sha256,
              'huella_manifiesto_sha256', r.huella_manifiesto_sha256)
       ) OR NOT EXISTS (
           SELECT 1
             FROM vec_ejecucion_documental_v4.paso_efecto_generacion_documental_v1 AS p
            WHERE p.estado = 'confirmado'
              AND p.contenido_guardado_canonico -> 'evidencia_operacion' =
                  p.evidencia_operacion_canonica
              AND p.objeto_ref = p.evidencia_operacion_canonica -> 'objeto' ->> 'referencia'
              AND p.objeto_version = p.evidencia_operacion_canonica -> 'objeto' ->> 'version'
              AND octet_length(p.objeto_version) <= 256
       ) OR NOT EXISTS (
           SELECT 1
             FROM vec_ejecucion_documental_v4.control_cadena_auditoria AS control
             JOIN LATERAL (
                 SELECT secuencia, huella_registro_sha256
                   FROM (
                       SELECT secuencia, huella_registro_sha256
                         FROM vec_ejecucion_documental_v4.auditoria
                       UNION ALL
                       SELECT secuencia, huella_registro_sha256
                         FROM vec_ejecucion_documental_v4.auditoria_efecto_generacion_documental_v1
                   ) AS cadena ORDER BY secuencia DESC LIMIT 1
             ) AS ultimo ON ultimo.secuencia = control.ultima_secuencia
                        AND ultimo.huella_registro_sha256 = control.ultima_huella_sha256
            WHERE control.control_id = true
       ) THEN
        RAISE EXCEPTION 'persistencia, autoridad o cadena global incompleta';
    END IF;
    FOREACH tabla IN ARRAY ARRAY['reserva_efecto_generacion_documental_v1',
        'paso_efecto_generacion_documental_v1',
        'auditoria_efecto_generacion_documental_v1',
        'evento_outbox_efecto_generacion_documental_v1'] LOOP
        IF has_table_privilege('vec_ejecucion_documental_v4_ejecutor_atestado',
               'vec_ejecucion_documental_v4.' || tabla,
               'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER')
           OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_class AS c
                WHERE c.oid = ('vec_ejecucion_documental_v4.' || tabla)::regclass
                  AND c.relrowsecurity AND c.relforcerowsecurity) THEN
            RAISE EXCEPTION 'ACL/RLS invalida en %', tabla;
        END IF;
        IF has_table_privilege('vec_ejecucion_documental_v4_emisor_capacidad',
               'vec_ejecucion_documental_v4.' || tabla,
               'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER') THEN
            RAISE EXCEPTION 'el emisor accede a la tabla %', tabla;
        END IF;
    END LOOP;
    IF NOT has_function_privilege('vec_ejecucion_documental_v4_ejecutor_atestado',
           'vec_ejecucion_documental_v4.reservar_efecto_generacion_documental_v1(jsonb,jsonb)',
           'EXECUTE')
       OR NOT has_function_privilege('vec_ejecucion_documental_v4_ejecutor_atestado',
           'vec_ejecucion_documental_v4.confirmar_paso_generacion_documental_v1(jsonb)',
           'EXECUTE')
       OR has_function_privilege('vec_ejecucion_documental_v4_ejecutor_atestado',
           'vec_ejecucion_documental_v4.contenido_documento_guardado_valido_v1(jsonb,jsonb,jsonb,timestamptz,timestamptz)',
           'EXECUTE') THEN
        RAISE EXCEPTION 'superficie funcional del registro de efectos invalida';
    END IF;
    definicion := pg_get_functiondef(to_regprocedure(
        'vec_ejecucion_documental_v4.reservar_efecto_generacion_documental_v1(jsonb,jsonb)'));
    IF strpos(definicion, 'revalidar_decision_ejecucion_documental_v4') = 0
       OR strpos(definicion, 'consumo_decision_atomico') = 0
       OR strpos(definicion, 'consumo_capacidad') = 0
       OR strpos(definicion, 'FOR UPDATE OF orden, atestacion, consumo, capacidad') = 0 THEN
        RAISE EXCEPTION 'la reserva no bloquea y revalida la autoridad exacta';
    END IF;
    rechazada := false;
    BEGIN
        UPDATE vec_ejecucion_documental_v4.paso_efecto_generacion_documental_v1
           SET estado = 'confirmado' WHERE estado = 'reservado';
    EXCEPTION WHEN SQLSTATE '55000' THEN rechazada := true;
    END;
    IF NOT rechazada THEN RAISE EXCEPTION 'los pasos no son append-only'; END IF;
    rechazada := false;
    BEGIN
        DELETE FROM vec_ejecucion_documental_v4.reserva_efecto_generacion_documental_v1;
    EXCEPTION WHEN SQLSTATE '55000' THEN rechazada := true;
    END;
    IF NOT rechazada THEN RAISE EXCEPTION 'la reserva se pudo liberar'; END IF;
END
$persistencia_y_acl$;
\endif
\else
\if :{?exigir_autoridad}
DO $autoridad_obligatoria$
BEGIN
    RAISE EXCEPTION 'no existe orden atestada con capacidad vigente para probar efectos';
END
$autoridad_obligatoria$;
\else
DO $contrato_estatico_sql_only$
DECLARE reserva text; confirmacion text; control record; ultimo record;
BEGIN
    reserva := pg_get_functiondef(to_regprocedure(
        'vec_ejecucion_documental_v4.reservar_efecto_generacion_documental_v1(jsonb,jsonb)'));
    confirmacion := pg_get_functiondef(to_regprocedure(
        'vec_ejecucion_documental_v4.confirmar_paso_generacion_documental_v1(jsonb)'));
    IF strpos(reserva, 'revalidar_decision_ejecucion_documental_v4') = 0
       OR strpos(reserva, 'consumo_decision_atomico') = 0
       OR strpos(reserva, 'consumo_capacidad') = 0
       OR strpos(reserva, 'FOR UPDATE OF orden, atestacion, consumo, capacidad') = 0
       OR strpos(confirmacion, 'contenido_documento_guardado_valido_v1') = 0
       OR strpos(confirmacion, 'contenido_guardado_canonico') = 0 THEN
        RAISE EXCEPTION 'contrato estatico de autoridad o terminal incompleto';
    END IF;
    SELECT ultima_secuencia, ultima_huella_sha256 INTO STRICT control
      FROM vec_ejecucion_documental_v4.control_cadena_auditoria
     WHERE control_id = true;
    SELECT secuencia, huella_registro_sha256 INTO ultimo
      FROM (
          SELECT secuencia, huella_registro_sha256
            FROM vec_ejecucion_documental_v4.auditoria
          UNION ALL
          SELECT secuencia, huella_registro_sha256
            FROM vec_ejecucion_documental_v4.auditoria_efecto_generacion_documental_v1
      ) AS cadena ORDER BY secuencia DESC LIMIT 1;
    IF FOUND AND (control.ultima_secuencia <> ultimo.secuencia
          OR control.ultima_huella_sha256 <> ultimo.huella_registro_sha256)
       OR NOT FOUND AND (control.ultima_secuencia <> 0
          OR control.ultima_huella_sha256 <> repeat('0', 64)) THEN
        RAISE EXCEPTION 'el control no conserva el ultimo eslabon real';
    END IF;
END
$contrato_estatico_sql_only$;
\endif
\endif
COMMIT;
