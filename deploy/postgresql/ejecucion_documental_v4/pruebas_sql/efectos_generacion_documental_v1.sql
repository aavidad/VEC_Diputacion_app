\set ON_ERROR_STOP 1
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

CREATE TEMP TABLE entrada_efectos_documentales_v1 (
    contexto jsonb NOT NULL,
    contexto_conflictivo jsonb NOT NULL,
    manifiesto jsonb NOT NULL
) ON COMMIT DROP;

DO $fixture$
DECLARE
    huella_plantilla text := repeat('a', 64);
    huella_decision text := repeat('b', 64);
    huella_recurso text := repeat('c', 64);
    huella_solicitud text := 'hmac-sha256:solicitud:' || repeat('d', 64);
    sujeto text := 'hmac-sha256:sujeto:' || repeat('e', 64);
    pasos jsonb := '[]'::jsonb;
    paso jsonb;
    manifiesto jsonb;
    contexto jsonb;
    conflicto jsonb;
    canonico bytea;
    huella_paso text;
    huella_manifiesto text;
    huella_plan text;
    indice integer;
    efecto text;
BEGIN
    FOR indice IN 1..2 LOOP
        paso := jsonb_build_object(
            'referencia_logica', CASE indice WHEN 1 THEN 'documento:prueba:docx'
                ELSE 'documento:prueba:pdf' END,
            'clave_idempotencia', CASE indice WHEN 1 THEN 'contenido:prueba:docx'
                ELSE 'contenido:prueba:pdf' END,
            'formato', CASE indice WHEN 1 THEN 'docx' ELSE 'pdf' END,
            'zona', 'admitida',
            'mime', CASE indice WHEN 1
                THEN 'application/vnd.openxmlformats-officedocument.wordprocessingml.document'
                ELSE 'application/pdf' END,
            'tamano', CASE indice WHEN 1 THEN 2048 ELSE 1024 END,
            'huella_sha256', CASE indice WHEN 1 THEN repeat('1', 64)
                ELSE repeat('2', 64) END
        );
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
        paso := paso || jsonb_build_object(
            'paso_ref', 'generar_documento_' || huella_paso,
            'huella_paso_sha256', huella_paso
        );
        pasos := pasos || jsonb_build_array(paso);
    END LOOP;
    canonico := vec_ejecucion_documental_v4.encuadrar_capacidad(
        'vec.documentos.manifiesto-generacion.v1') ||
        vec_ejecucion_documental_v4.encuadrar_capacidad('plantilla:prueba') ||
        vec_ejecucion_documental_v4.encuadrar_capacidad('1') ||
        vec_ejecucion_documental_v4.encuadrar_capacidad('vec') ||
        vec_ejecucion_documental_v4.encuadrar_capacidad('informe_prueba') ||
        vec_ejecucion_documental_v4.encuadrar_capacidad(huella_plantilla) ||
        vec_ejecucion_documental_v4.encuadrar_capacidad('vec.documentos.generar') ||
        vec_ejecucion_documental_v4.encuadrar_capacidad('2');
    FOR paso IN SELECT e.valor FROM jsonb_array_elements(pasos) AS e(valor) LOOP
        canonico := canonico ||
            vec_ejecucion_documental_v4.encuadrar_capacidad(paso ->> 'paso_ref') ||
            vec_ejecucion_documental_v4.encuadrar_capacidad(paso ->> 'huella_paso_sha256') ||
            vec_ejecucion_documental_v4.encuadrar_capacidad(paso ->> 'referencia_logica') ||
            vec_ejecucion_documental_v4.encuadrar_capacidad(paso ->> 'clave_idempotencia') ||
            vec_ejecucion_documental_v4.encuadrar_capacidad(paso ->> 'formato') ||
            vec_ejecucion_documental_v4.encuadrar_capacidad(paso ->> 'zona') ||
            vec_ejecucion_documental_v4.encuadrar_capacidad(paso ->> 'mime') ||
            vec_ejecucion_documental_v4.encuadrar_capacidad((paso ->> 'tamano')::bigint::text) ||
            vec_ejecucion_documental_v4.encuadrar_capacidad(paso ->> 'huella_sha256');
    END LOOP;
    huella_manifiesto := encode(sha256(canonico), 'hex');
    manifiesto := jsonb_build_object(
        'esquema', 'vec.documentos.manifiesto-generacion.v1',
        'plantilla_id', 'plantilla:prueba', 'plantilla_version', 1,
        'modulo_id', 'vec', 'tipo_documental', 'informe_prueba',
        'huella_plantilla_sha256', huella_plantilla,
        'permiso_generar', 'vec.documentos.generar',
        'huella_manifiesto_sha256', huella_manifiesto, 'pasos', pasos
    );
    efecto := 'efecto:generacion-documental:prueba';
    canonico := ''::bytea;
    FOREACH huella_paso IN ARRAY ARRAY[
        'vec.almacen.contexto-operacion.v1', 'decision:documental:prueba',
        huella_decision, 'vec.documentos.generar', 'expediente:prueba',
        huella_recurso, 'tramitar_expediente', 'correlacion:documental:prueba',
        'operacion:documental:prueba', 'carga:documental:prueba', 'restringida',
        sujeto, huella_solicitud, efecto, '', '', huella_manifiesto
    ] LOOP
        canonico := canonico ||
            vec_ejecucion_documental_v4.encuadrar_capacidad(huella_paso);
    END LOOP;
    FOR paso IN SELECT e.valor FROM jsonb_array_elements(pasos) AS e(valor) LOOP
        canonico := canonico ||
            vec_ejecucion_documental_v4.encuadrar_capacidad(paso ->> 'paso_ref') ||
            vec_ejecucion_documental_v4.encuadrar_capacidad('escribir') ||
            vec_ejecucion_documental_v4.encuadrar_capacidad(paso ->> 'huella_paso_sha256');
    END LOOP;
    huella_plan := encode(sha256(canonico), 'hex');
    contexto := jsonb_build_object(
        'esquema', 'vec.almacen.contexto-operacion.v1',
        'operacion_ref', 'operacion:documental:prueba',
        'correlacion_ref', 'correlacion:documental:prueba',
        'autorizacion_ref', 'decision:documental:prueba',
        'finalidad', 'tramitar_expediente', 'clasificacion', 'restringida',
        'accion_negocio', 'vec.documentos.generar', 'accion_tecnica', 'escribir',
        'carga_ref', 'carga:documental:prueba', 'sujeto_seudonimo_hmac', sujeto,
        'recurso_ref', 'expediente:prueba', 'modulo_id', 'vec',
        'tipo_recurso', 'expediente', 'huella_recurso_sha256', huella_recurso,
        'huella_solicitud_hmac', huella_solicitud, 'efecto_ref', efecto,
        'huella_plan_efecto_sha256', huella_plan,
        'huella_manifiesto_sha256', huella_manifiesto,
        'huella_paso_sha256', pasos -> 0 ->> 'huella_paso_sha256',
        'paso_ref', pasos -> 0 ->> 'paso_ref',
        'objeto_vinculado_ref', '', 'objeto_vinculado_version', '',
        'huella_decision_sha256', huella_decision,
        'verificada_en', '2026-01-01T00:00:00.000000Z',
        'valida_hasta', '2099-01-01T00:00:00.000000Z'
    );
    conflicto := jsonb_set(contexto, '{efecto_ref}', '"efecto:generacion-documental:otro"');
    canonico := ''::bytea;
    FOREACH huella_paso IN ARRAY ARRAY[
        'vec.almacen.contexto-operacion.v1', 'decision:documental:prueba',
        huella_decision, 'vec.documentos.generar', 'expediente:prueba',
        huella_recurso, 'tramitar_expediente', 'correlacion:documental:prueba',
        'operacion:documental:prueba', 'carga:documental:prueba', 'restringida',
        sujeto, huella_solicitud, 'efecto:generacion-documental:otro', '', '',
        huella_manifiesto
    ] LOOP
        canonico := canonico ||
            vec_ejecucion_documental_v4.encuadrar_capacidad(huella_paso);
    END LOOP;
    FOR paso IN SELECT valor FROM jsonb_array_elements(pasos) AS valor LOOP
        canonico := canonico ||
            vec_ejecucion_documental_v4.encuadrar_capacidad(paso ->> 'paso_ref') ||
            vec_ejecucion_documental_v4.encuadrar_capacidad('escribir') ||
            vec_ejecucion_documental_v4.encuadrar_capacidad(paso ->> 'huella_paso_sha256');
    END LOOP;
    conflicto := jsonb_set(conflicto, '{huella_plan_efecto_sha256}',
        to_jsonb(encode(sha256(canonico), 'hex')));
    INSERT INTO entrada_efectos_documentales_v1 VALUES
        (contexto, conflicto, manifiesto);
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
            e.contexto, e.manifiesto
        ) AS r;
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
    entrada record;
    reserva record;
    paso_uno jsonb;
    paso_dos jsonb;
    confirmacion jsonb;
    marca jsonb;
    respuesta record;
    rechazado boolean;
    ahora text;
BEGIN
    SELECT * INTO STRICT entrada FROM entrada_efectos_documentales_v1;
    SELECT r.* INTO STRICT reserva
      FROM vec_ejecucion_documental_v4.reservar_efecto_generacion_documental_v1(
          jsonb_set(entrada.contexto, '{verificada_en}',
              '"2026-01-02T00:00:00.000000Z"'),
          entrada.manifiesto
      ) AS r;
    IF reserva.resultado <> 'repetida' OR NOT reserva.repetida
       OR jsonb_array_length(reserva.pasos) <> 2
       OR EXISTS (
           SELECT 1 FROM jsonb_array_elements(reserva.pasos) AS p
            WHERE p ->> 'estado' <> 'reservado'
       ) THEN
        RAISE EXCEPTION 'el replay de reserva no fue exacto';
    END IF;
    rechazado := false;
    BEGIN
        PERFORM *
          FROM vec_ejecucion_documental_v4.reservar_efecto_generacion_documental_v1(
              entrada.contexto_conflictivo, entrada.manifiesto
          );
    EXCEPTION WHEN check_violation THEN rechazado := true;
    END;
    IF NOT rechazado THEN
        RAISE EXCEPTION 'una decision se reutilizo para otro efecto';
    END IF;

    paso_uno := entrada.manifiesto -> 'pasos' -> 0;
    paso_dos := entrada.manifiesto -> 'pasos' -> 1;
    ahora := to_char(clock_timestamp() AT TIME ZONE 'UTC',
        'YYYY-MM-DD"T"HH24:MI:SS.US"Z"');
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
        'objeto_ref', 'objeto:documental:prueba', 'objeto_version', 'version:1',
        'conector_id', 'almacen:prueba',
        'evidencia_operacion_ref', 'evidencia:almacen:prueba',
        'realizada_en', ahora
    );
    SELECT * INTO STRICT respuesta
      FROM vec_ejecucion_documental_v4.confirmar_paso_generacion_documental_v1(
          confirmacion
      );
    IF respuesta.resultado <> 'confirmada' THEN
        RAISE EXCEPTION 'no se confirmo el paso reservado';
    END IF;
    SELECT * INTO STRICT respuesta
      FROM vec_ejecucion_documental_v4.confirmar_paso_generacion_documental_v1(
          confirmacion
      );
    IF respuesta.resultado <> 'repetida' THEN
        RAISE EXCEPTION 'la confirmacion exacta no fue idempotente';
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
        'incidente_ref', 'incidente:generacion-documental:prueba'
    );
    SELECT * INTO STRICT respuesta FROM
        vec_ejecucion_documental_v4.marcar_paso_generacion_documental_indeterminado_v1(marca);
    IF respuesta.resultado <> 'marcada' THEN
        RAISE EXCEPTION 'no se marco el paso ambiguo';
    END IF;
    SELECT * INTO STRICT respuesta FROM
        vec_ejecucion_documental_v4.marcar_paso_generacion_documental_indeterminado_v1(marca);
    IF respuesta.resultado <> 'repetida' THEN
        RAISE EXCEPTION 'la marca indeterminada exacta no fue idempotente';
    END IF;
    rechazado := false;
    BEGIN
        PERFORM * FROM
            vec_ejecucion_documental_v4.confirmar_paso_generacion_documental_v1(
                confirmacion || jsonb_build_object(
                    'paso_ref', paso_dos ->> 'paso_ref',
                    'huella_paso_sha256', paso_dos ->> 'huella_paso_sha256'
                )
            );
    EXCEPTION WHEN check_violation THEN rechazado := true;
    END;
    IF NOT rechazado THEN
        RAISE EXCEPTION 'un paso indeterminado se convirtio en confirmado';
    END IF;
    rechazado := false;
    BEGIN
        PERFORM * FROM
            vec_ejecucion_documental_v4.marcar_paso_generacion_documental_indeterminado_v1(
                marca || jsonb_build_object(
                    'paso_ref', paso_uno ->> 'paso_ref',
                    'huella_paso_sha256', paso_uno ->> 'huella_paso_sha256'
                )
            );
    EXCEPTION WHEN check_violation THEN rechazado := true;
    END;
    IF NOT rechazado THEN
        RAISE EXCEPTION 'un paso confirmado se degrado a indeterminado';
    END IF;
END
$contrato$;

DO $persistencia_y_acl$
DECLARE
    tabla text;
    rechazado boolean;
BEGIN
    IF (SELECT count(*) FROM
        vec_ejecucion_documental_v4.reserva_efecto_generacion_documental_v1) <> 1
       OR (SELECT count(*) FROM
        vec_ejecucion_documental_v4.paso_efecto_generacion_documental_v1) <> 4
       OR (SELECT count(*) FROM
        vec_ejecucion_documental_v4.paso_efecto_generacion_documental_v1
        WHERE estado = 'reservado') <> 2
       OR (SELECT count(*) FROM
        vec_ejecucion_documental_v4.paso_efecto_generacion_documental_v1
        WHERE estado IN ('confirmado', 'indeterminado')) <> 2
       OR (SELECT count(*) FROM
        vec_ejecucion_documental_v4.auditoria_efecto_generacion_documental_v1) <> 3
       OR (SELECT count(*) FROM
        vec_ejecucion_documental_v4.evento_outbox_efecto_generacion_documental_v1) <> 3
       OR NOT EXISTS (
           SELECT 1
             FROM vec_ejecucion_documental_v4.control_cadena_auditoria AS c
             JOIN vec_ejecucion_documental_v4.auditoria_efecto_generacion_documental_v1 AS a
               ON a.secuencia = c.ultima_secuencia
              AND a.huella_registro_sha256 = c.ultima_huella_sha256
            WHERE c.control_id = true
       ) THEN
        RAISE EXCEPTION 'reserva, pasos o cadena atomica incompletos';
    END IF;
    IF NOT EXISTS (
        SELECT 1
          FROM vec_ejecucion_documental_v4.reserva_efecto_generacion_documental_v1 AS r
          CROSS JOIN entrada_efectos_documentales_v1 AS e
         WHERE r.tupla_canonica = jsonb_build_object(
             'esquema', 'vec.documentos.registro-efectos-generacion.tupla.v1',
             'decision_ref', e.contexto ->> 'autorizacion_ref',
             'efecto_ref', e.contexto ->> 'efecto_ref',
             'huella_decision_sha256', e.contexto ->> 'huella_decision_sha256',
             'huella_plan_efecto_sha256', e.contexto ->> 'huella_plan_efecto_sha256',
             'huella_manifiesto_sha256', e.contexto ->> 'huella_manifiesto_sha256')
           AND r.huella_tupla_sha256 = encode(sha256(
               vec_ejecucion_documental_v4.encuadrar_capacidad(
                   'vec.documentos.registro-efectos-generacion.tupla.v1') ||
               vec_ejecucion_documental_v4.encuadrar_capacidad(e.contexto ->> 'autorizacion_ref') ||
               vec_ejecucion_documental_v4.encuadrar_capacidad(e.contexto ->> 'efecto_ref') ||
               vec_ejecucion_documental_v4.encuadrar_capacidad(e.contexto ->> 'huella_decision_sha256') ||
               vec_ejecucion_documental_v4.encuadrar_capacidad(e.contexto ->> 'huella_plan_efecto_sha256') ||
               vec_ejecucion_documental_v4.encuadrar_capacidad(e.contexto ->> 'huella_manifiesto_sha256')
           ), 'hex')
           AND r.reserva_ref = 'reserva:efecto-generacion-documental:v1:' ||
               encode(sha256(
                   vec_ejecucion_documental_v4.encuadrar_capacidad(
                       'vec.documentos.registro-efectos-generacion.reserva.v1') ||
                   vec_ejecucion_documental_v4.encuadrar_capacidad(r.huella_tupla_sha256) ||
                   vec_ejecucion_documental_v4.encuadrar_capacidad(r.huella_manifiesto_sha256)
               ), 'hex')
    ) THEN
        RAISE EXCEPTION 'tupla o referencia de reserva no derivada en SQL';
    END IF;
    FOREACH tabla IN ARRAY ARRAY[
        'reserva_efecto_generacion_documental_v1',
        'paso_efecto_generacion_documental_v1',
        'auditoria_efecto_generacion_documental_v1',
        'evento_outbox_efecto_generacion_documental_v1'
    ] LOOP
        IF has_table_privilege('vec_ejecucion_documental_v4_ejecutor_atestado',
               'vec_ejecucion_documental_v4.' || tabla,
               'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER')
           OR has_table_privilege('vec_ejecucion_documental_v4_emisor_capacidad',
               'vec_ejecucion_documental_v4.' || tabla,
               'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER')
           OR NOT EXISTS (
               SELECT 1 FROM pg_catalog.pg_class AS c
                WHERE c.oid = ('vec_ejecucion_documental_v4.' || tabla)::regclass
                  AND c.relrowsecurity AND c.relforcerowsecurity
           ) THEN
            RAISE EXCEPTION 'ACL/RLS invalida en %', tabla;
        END IF;
    END LOOP;
    IF NOT has_function_privilege(
           'vec_ejecucion_documental_v4_ejecutor_atestado',
           'vec_ejecucion_documental_v4.reservar_efecto_generacion_documental_v1(jsonb,jsonb)',
           'EXECUTE')
       OR NOT has_function_privilege(
           'vec_ejecucion_documental_v4_ejecutor_atestado',
           'vec_ejecucion_documental_v4.confirmar_paso_generacion_documental_v1(jsonb)',
           'EXECUTE')
       OR NOT has_function_privilege(
           'vec_ejecucion_documental_v4_ejecutor_atestado',
           'vec_ejecucion_documental_v4.marcar_paso_generacion_documental_indeterminado_v1(jsonb)',
           'EXECUTE')
       OR has_function_privilege(
           'vec_ejecucion_documental_v4_ejecutor_atestado',
           'vec_ejecucion_documental_v4.registrar_cambio_efecto_generacion_documental_v1(text,text,text,text,timestamptz)',
           'EXECUTE')
       OR (SELECT count(*)
             FROM pg_catalog.pg_proc AS p
             JOIN pg_catalog.pg_namespace AS n ON n.oid = p.pronamespace
            WHERE n.nspname = 'vec_ejecucion_documental_v4'
              AND p.proname = ANY (ARRAY[
                  'reservar_efecto_generacion_documental_v1',
                  'confirmar_paso_generacion_documental_v1',
                  'marcar_paso_generacion_documental_indeterminado_v1'
              ]) AND p.prosecdef AND p.provolatile = 'v'
              AND p.proconfig = ARRAY['search_path=pg_catalog, pg_temp']::text[]
       ) <> 3
       OR (SELECT p.prosecdef
             FROM pg_catalog.pg_proc AS p
            WHERE p.oid = to_regprocedure(
                'vec_ejecucion_documental_v4.registrar_cambio_efecto_generacion_documental_v1(text,text,text,text,timestamptz)'
            ))
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_proc AS p
           JOIN pg_catalog.pg_namespace AS n ON n.oid = p.pronamespace
            WHERE n.nspname = 'vec_ejecucion_documental_v4'
              AND p.proname ~ '(liberar|borrar|eliminar|reintentar).*efecto.*generacion'
       ) THEN
        RAISE EXCEPTION 'superficie funcional del registro de efectos invalida';
    END IF;
    rechazado := false;
    BEGIN
        UPDATE vec_ejecucion_documental_v4.paso_efecto_generacion_documental_v1
           SET estado = 'confirmado' WHERE estado = 'reservado';
    EXCEPTION WHEN SQLSTATE '55000' THEN rechazado := true;
    END;
    IF NOT rechazado THEN RAISE EXCEPTION 'los pasos no son append-only'; END IF;
    rechazado := false;
    BEGIN
        DELETE FROM vec_ejecucion_documental_v4.reserva_efecto_generacion_documental_v1;
    EXCEPTION WHEN SQLSTATE '55000' THEN rechazado := true;
    END;
    IF NOT rechazado THEN RAISE EXCEPTION 'la reserva se pudo liberar'; END IF;
    IF strpos(pg_get_functiondef(to_regprocedure(
           'vec_ejecucion_documental_v4.reservar_efecto_generacion_documental_v1(jsonb,jsonb)'
       )), 'FOR UPDATE') = 0
       OR strpos(pg_get_functiondef(to_regprocedure(
           'vec_ejecucion_documental_v4.confirmar_paso_generacion_documental_v1(jsonb)'
       )), 'FOR UPDATE') = 0
       OR strpos(pg_get_functiondef(to_regprocedure(
           'vec_ejecucion_documental_v4.marcar_paso_generacion_documental_indeterminado_v1(jsonb)'
       )), 'FOR UPDATE') = 0 THEN
        RAISE EXCEPTION 'falta serializacion FOR UPDATE';
    END IF;
END
$persistencia_y_acl$;
\endif
COMMIT;
