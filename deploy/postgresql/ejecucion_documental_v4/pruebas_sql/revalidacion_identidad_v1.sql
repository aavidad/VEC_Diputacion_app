-- Contrato estructural previo a la integracion Go. Las pruebas de ejecucion,
-- revocacion y concurrencia usan COSE real y se ejecutan desde Go; este bloque
-- comprueba que no sobreviva ninguna via SQL parcial o con privilegios cruzados.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

DO $prueba$
DECLARE
    definicion_ejecucion text;
    decision_canonica_30 jsonb;
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname IN (
             'vec_ejecucion_documental_v4_emisor_capacidad',
             'vec_ejecucion_documental_v4_ejecutor_atestado'
         ) AND rolcanlogin
    ) THEN
        RAISE EXCEPTION 'un rol estructural V4 permite LOGIN';
    END IF;
    IF NOT EXISTS (
        SELECT 1
          FROM vec_autorizacion.decision_autorizacion
         WHERE decision_ref = 'decision:prueba:legacy:v1'
           AND (SELECT count(*) FROM jsonb_object_keys(documento)) = 30
           AND NOT (documento ? 'vinculo_autenticacion_actor')
    ) THEN
        RAISE EXCEPTION 'la evolucion no preservo la decision historica de 30 claves';
    END IF;
    IF to_regprocedure(
           'vec_ejecucion_documental_v4.registrar_atestacion_pdp(jsonb,bytea,bytea,bytea,bytea)'
       ) IS NOT NULL
       OR to_regprocedure(
           'vec_ejecucion_documental_v4.preparar_reserva(jsonb)'
       ) IS NOT NULL
       OR to_regprocedure(
           'vec_ejecucion_documental_v4.consumir_decision(jsonb)'
       ) IS NOT NULL THEN
        RAISE EXCEPTION 'sobrevive una via parcial de registro o consumo V4';
    END IF;
    IF to_regprocedure(
           'vec_ejecucion_documental_v4.ejecutar_plan_atestado(bytea,bytea,bytea,bytea,bytea,bytea,bytea,jsonb)'
       ) IS NULL
       OR to_regprocedure(
           'vec_ejecucion_documental_v4.obtener_confianza_actual()'
       ) IS NULL
       OR to_regprocedure(
           'vec_ejecucion_documental_v4.obtener_material_emisor_capacidad()'
       ) IS NULL
       OR to_regprocedure(
           'vec_autorizacion.revalidar_decision_ejecucion_documental_v4(jsonb,bytea)'
       ) IS NULL
       OR to_regprocedure(
           'vec_autorizacion.decision_canonica_documental_v4_estructura_valida(jsonb)'
       ) IS NULL THEN
        RAISE EXCEPTION 'falta una funcion cerrada V4';
    END IF;
    SELECT jsonb_object_agg(clave, to_jsonb(clave))
      INTO decision_canonica_30
      FROM unnest(ARRAY[
          'esquema', 'decision_ref', 'concedida', 'codigo', 'principal_id',
          'perfil_activo_ref', 'accion', 'recurso_ref', 'modulo_id',
          'tipo_recurso', 'contexto_recurso_huella_sha256', 'finalidad',
          'correlacion_ref', 'vinculo_autenticacion_actor', 'asignacion_ref',
          'asignacion_huella_sha256', 'version_rol_ref',
          'version_rol_huella_sha256', 'control_vigencia_version_rol_ref',
          'control_vigencia_version_rol_revision',
          'control_vigencia_version_rol_huella_sha256',
          'revision_catalogo_politicas', 'catalogo_politicas_huella_sha256',
          'politicas_evaluadas', 'politicas_aplicables', 'garantia_minima',
          'campos_permitidos', 'obligaciones', 'emitida_en', 'valida_hasta'
      ]) AS clave;
    IF vec_autorizacion.decision_canonica_documental_v4_estructura_valida(
           decision_canonica_30
       ) IS NOT TRUE
       OR vec_autorizacion.decision_canonica_documental_v4_estructura_valida(
           decision_canonica_30 || jsonb_build_object('extra', true)
       ) IS TRUE
       OR vec_autorizacion.decision_canonica_documental_v4_estructura_valida(
           decision_canonica_30 - 'vinculo_autenticacion_actor'
       ) IS TRUE THEN
        RAISE EXCEPTION 'el contrato funcional de decision canonica no es 30 exactas';
    END IF;
    SELECT pg_get_functiondef(to_regprocedure(
               'vec_ejecucion_documental_v4.ejecutar_plan_atestado(bytea,bytea,bytea,bytea,bytea,bytea,bytea,jsonb)'
           ))
      INTO definicion_ejecucion;
    IF strpos(definicion_ejecucion, 'public.hmac(') = 0
       OR strpos(definicion_ejecucion, 'metadatos := convert_from(') = 0
       OR strpos(definicion_ejecucion, 'public.hmac(') >=
          strpos(definicion_ejecucion, 'metadatos := convert_from(') THEN
        RAISE EXCEPTION 'los artefactos JSON se interpretan antes de autenticar la capacidad';
    END IF;
    IF has_table_privilege(
           'vec_ejecucion_documental_v4_ejecutor_atestado',
           'vec_ejecucion_documental_v4.atestacion_pdp', 'SELECT'
       ) OR has_table_privilege(
           'vec_ejecucion_documental_v4_ejecutor_atestado',
           'vec_ejecucion_documental_v4.configuracion_confianza_version',
           'INSERT,UPDATE,DELETE,SELECT'
       ) OR has_table_privilege(
           'vec_ejecucion_documental_v4_ejecutor_atestado',
           'vec_ejecucion_documental_v4.raiz_confianza_version',
           'INSERT,UPDATE,DELETE,SELECT'
       ) OR has_table_privilege(
           'vec_ejecucion_documental_v4_ejecutor_atestado',
           'vec_ejecucion_documental_v4.clave_capacidad_version',
           'INSERT,UPDATE,DELETE,SELECT'
       ) OR has_table_privilege(
           'vec_ejecucion_documental_v4_emisor_capacidad',
           'vec_ejecucion_documental_v4.clave_capacidad_version',
           'INSERT,UPDATE,DELETE,SELECT'
       ) THEN
        RAISE EXCEPTION 'el runtime accede directamente a tablas protegidas';
    END IF;
    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_class AS tabla
          JOIN pg_catalog.pg_namespace AS espacio
            ON espacio.oid = tabla.relnamespace
         WHERE espacio.nspname = 'vec_ejecucion_documental_v4'
           AND tabla.relkind IN ('r', 'p', 'v', 'm', 'S', 'f')
           AND (
               has_table_privilege(
                   'vec_ejecucion_documental_v4_emisor_capacidad',
                   tabla.oid, 'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER'
               ) OR has_table_privilege(
                   'vec_ejecucion_documental_v4_ejecutor_atestado',
                   tabla.oid, 'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER'
               )
           )
    ) THEN
        RAISE EXCEPTION 'un rol runtime V4 tiene privilegio directo sobre una relacion';
    END IF;
    IF NOT has_function_privilege(
           'vec_ejecucion_documental_v4_ejecutor_atestado',
           'vec_ejecucion_documental_v4.ejecutar_plan_atestado(bytea,bytea,bytea,bytea,bytea,bytea,bytea,jsonb)',
           'EXECUTE'
       ) OR has_function_privilege(
           'vec_ejecucion_documental_v4_ejecutor_atestado',
           'vec_ejecucion_documental_v4.obtener_confianza_actual()',
           'EXECUTE'
       ) OR has_function_privilege(
           'vec_ejecucion_documental_v4_ejecutor_atestado',
           'vec_ejecucion_documental_v4.obtener_material_emisor_capacidad()',
           'EXECUTE'
       ) OR has_function_privilege(
           'vec_ejecucion_documental_v4_emisor_capacidad',
           'vec_ejecucion_documental_v4.ejecutar_plan_atestado(bytea,bytea,bytea,bytea,bytea,bytea,bytea,jsonb)',
           'EXECUTE'
       ) OR NOT has_function_privilege(
           'vec_ejecucion_documental_v4_emisor_capacidad',
           'vec_ejecucion_documental_v4.obtener_confianza_actual()',
           'EXECUTE'
       ) OR NOT has_function_privilege(
           'vec_ejecucion_documental_v4_emisor_capacidad',
           'vec_ejecucion_documental_v4.obtener_material_emisor_capacidad()',
           'EXECUTE'
       ) OR has_function_privilege(
           'vec_ejecucion_documental_v4_ejecutor_atestado',
           'vec_autorizacion.revalidar_decision_ejecucion_documental_v4(jsonb,bytea)',
           'EXECUTE'
       ) OR has_function_privilege(
           'vec_ejecucion_documental_v4_ejecutor_atestado',
           'vec_autorizacion.decision_canonica_documental_v4_estructura_valida(jsonb)',
           'EXECUTE'
       ) OR has_function_privilege(
           'vec_ejecucion_documental_v4_emisor_capacidad',
           'vec_autorizacion.decision_canonica_documental_v4_estructura_valida(jsonb)',
           'EXECUTE'
       ) THEN
        RAISE EXCEPTION 'separacion de funciones runtime/propietario invalida';
    END IF;
    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc AS funcion
          JOIN pg_catalog.pg_namespace AS espacio
            ON espacio.oid = funcion.pronamespace
         WHERE espacio.nspname = 'vec_ejecucion_documental_v4'
           AND has_function_privilege(
               'vec_ejecucion_documental_v4_ejecutor_atestado',
               funcion.oid, 'EXECUTE'
           )
           AND funcion.oid <> to_regprocedure(
               'vec_ejecucion_documental_v4.ejecutar_plan_atestado(bytea,bytea,bytea,bytea,bytea,bytea,bytea,jsonb)'
           )
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc AS funcion
          JOIN pg_catalog.pg_namespace AS espacio
            ON espacio.oid = funcion.pronamespace
         WHERE espacio.nspname = 'vec_ejecucion_documental_v4'
           AND has_function_privilege(
               'vec_ejecucion_documental_v4_emisor_capacidad',
               funcion.oid, 'EXECUTE'
           )
           AND funcion.oid NOT IN (
               to_regprocedure(
                   'vec_ejecucion_documental_v4.obtener_confianza_actual()'
               ),
               to_regprocedure(
                   'vec_ejecucion_documental_v4.obtener_material_emisor_capacidad()'
               )
           )
    ) THEN
        RAISE EXCEPTION 'un rol runtime V4 ejecuta funciones fuera de su frontera';
    END IF;
    IF EXISTS (
        SELECT 1 FROM vec_ejecucion_documental_v4.configuracion_confianza_actual
    ) OR EXISTS (
        SELECT 1 FROM vec_ejecucion_documental_v4.raiz_confianza_actual
    ) OR EXISTS (
        SELECT 1 FROM vec_ejecucion_documental_v4.clave_capacidad_actual
    ) THEN
        RAISE EXCEPTION 'una migracion no debe sembrar confianza productiva';
    END IF;
END
$prueba$;

-- Las transiciones se ejercitan contra PostgreSQL real. Cada rechazo se
-- captura en un subbloque para que el resto de la prueba continue y todo se
-- revierte al final.
DO $transiciones$
DECLARE
    ahora timestamptz(6) := clock_timestamp();
    spki bytea := decode(repeat('ab', 32), 'hex');
    secreto bytea := decode(repeat('cd', 32), 'hex');
    metadatos bytea := convert_to('{}', 'UTF8');
    payload bytea := decode('00', 'hex');
    sobre bytea := decode(repeat('00', 16), 'hex');
    evidencia bytea := decode('01', 'hex');
    preimagen bytea := decode('02', 'hex');
    decision bytea := convert_to(repeat('d', 128), 'UTF8');
    efecto bytea := convert_to('{}', 'UTF8');
    capacidad jsonb;
    resultado_capacidad text;
    rechazo boolean;
BEGIN
    INSERT INTO vec_ejecucion_documental_v4.configuracion_confianza_version (
        revision, huella_configuracion_sha256, publicada_en, expira_en,
        estado, revocada_en, acto_ref
    ) VALUES
        ('config-prueba-1', repeat('1', 64), ahora - interval '10 minutes',
         ahora + interval '1 hour', 'activa', NULL, 'acto:config:1'),
        ('config-prueba-2', repeat('2', 64), ahora - interval '5 minutes',
         ahora + interval '1 hour', 'revocada', ahora - interval '4 minutes',
         'acto:config:2');
    INSERT INTO vec_ejecucion_documental_v4.configuracion_confianza_actual (
        control_id, revision, huella_configuracion_sha256, actualizada_en,
        acto_ref
    ) VALUES (true, 'config-prueba-1', repeat('1', 64), ahora, 'acto:actual:1');

    INSERT INTO vec_ejecucion_documental_v4.raiz_confianza_version (
        clave_id, version, revision_configuracion,
        huella_configuracion_sha256, algoritmo_cose, suite, audiencia_cose,
        audiencia_despliegue, clave_publica_spki, huella_clave_sha256,
        valida_desde, valida_hasta, estado, revocada_en, acto_ref
    ) VALUES
        ('raiz-prueba', 1, 'config-prueba-1', repeat('1', 64), 'EdDSA',
         'VEC-AD-COSE-EDDSA-1', 'atestacion_autorizacion_pdp',
         'despliegue-prueba', spki, encode(sha256(spki), 'hex'),
         ahora - interval '10 minutes', ahora + interval '1 hour',
         'activa', NULL, 'acto:raiz:1');
    INSERT INTO vec_ejecucion_documental_v4.raiz_confianza_actual (
        clave_id, version, revision_configuracion,
        huella_configuracion_sha256, actualizada_en, acto_ref
    ) VALUES (
        'raiz-prueba', 1, 'config-prueba-1', repeat('1', 64), ahora,
        'acto:raiz-actual:1'
    );

    INSERT INTO vec_ejecucion_documental_v4.clave_capacidad_version (
        clave_id, version, secreto_hmac, huella_secreto_sha256, emisor_id,
        valida_desde, valida_hasta, estado, revocada_en, acto_ref
    ) VALUES
        ('capacidad-prueba', 1, secreto, encode(sha256(secreto), 'hex'),
         'emisor-prueba', ahora - interval '10 minutes',
         ahora + interval '1 hour', 'activa', NULL, 'acto:capacidad:1');
    INSERT INTO vec_ejecucion_documental_v4.clave_capacidad_actual (
        control_id, clave_id, version, actualizada_en, acto_ref
    ) VALUES (
        true, 'capacidad-prueba', 1, ahora, 'acto:capacidad-actual:1'
    );

    capacidad := jsonb_build_object(
        'esquema', 'vec.documentos.capacidad-ejecucion.v4',
        'clave_id', 'capacidad-prueba', 'clave_version', 1,
        'emisor_id', 'emisor-prueba',
        'audiencia', 'vec_ejecucion_documental_v4.ejecutar_plan_atestado',
        'nonce', repeat('a', 64),
        'emitida_en', to_char(
            ahora AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ),
        'expira_en', to_char(
            (ahora + interval '10 seconds') AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ),
        'huella_metadatos_sha256', encode(sha256(metadatos), 'hex'),
        'huella_payload_sha256', encode(sha256(payload), 'hex'),
        'huella_sobre_sha256', encode(sha256(sobre), 'hex'),
        'huella_evidencia_sha256', encode(sha256(evidencia), 'hex'),
        'huella_preimagen_sha256', encode(sha256(preimagen), 'hex'),
        'huella_decision_sha256', encode(sha256(decision), 'hex'),
        'huella_efecto_sha256', encode(sha256(efecto), 'hex'),
        'revision_confianza', 'config-prueba-1',
        'huella_configuracion_sha256', repeat('1', 64),
        'raiz_clave_id', 'raiz-prueba',
        'huella_raiz_sha256', encode(sha256(spki), 'hex'),
        'mac_sha256', repeat('0', 64)
    );
    capacidad := jsonb_set(
        capacidad, '{mac_sha256}', to_jsonb(encode(public.hmac(
            vec_ejecucion_documental_v4.preimagen_capacidad(capacidad),
            secreto, 'sha256'
        ), 'hex'))
    );
    IF vec_ejecucion_documental_v4.bytea_igual_constante(
           public.hmac(
               vec_ejecucion_documental_v4.preimagen_capacidad(capacidad),
               secreto, 'sha256'
           ),
           decode(capacidad ->> 'mac_sha256', 'hex')
       ) IS NOT TRUE THEN
        RAISE EXCEPTION 'la preimagen HMAC de capacidad no es estable';
    END IF;
    SELECT ejecucion.resultado
      INTO resultado_capacidad
      FROM vec_ejecucion_documental_v4.ejecutar_plan_atestado(
          metadatos, payload, sobre, evidencia, preimagen, decision, efecto,
          capacidad
      ) AS ejecucion;
    IF resultado_capacidad <> 'rechazada' OR EXISTS (
        SELECT 1 FROM vec_ejecucion_documental_v4.consumo_capacidad
    ) OR EXISTS (
        SELECT 1 FROM vec_ejecucion_documental_v4.orden_generacion_documental
    ) THEN
        RAISE EXCEPTION 'una capacidad no autoriza artefactos semanticamente invalidos';
    END IF;
    SELECT ejecucion.resultado
      INTO resultado_capacidad
      FROM vec_ejecucion_documental_v4.ejecutar_plan_atestado(
          metadatos || decode('00', 'hex'), payload, sobre, evidencia,
          preimagen, decision, efecto, capacidad
      ) AS ejecucion;
    IF resultado_capacidad <> 'rechazada' OR EXISTS (
        SELECT 1 FROM vec_ejecucion_documental_v4.consumo_capacidad
    ) THEN
        RAISE EXCEPTION 'se aceptaron bytes distintos de los ligados por HMAC';
    END IF;

    INSERT INTO vec_ejecucion_documental_v4.raiz_confianza_version (
        clave_id, version, revision_configuracion,
        huella_configuracion_sha256, algoritmo_cose, suite, audiencia_cose,
        audiencia_despliegue, clave_publica_spki, huella_clave_sha256,
        valida_desde, valida_hasta, estado, revocada_en, acto_ref
    ) VALUES (
        'raiz-prueba', 2, 'config-prueba-2', repeat('2', 64), 'EdDSA',
        'VEC-AD-COSE-EDDSA-1', 'atestacion_autorizacion_pdp',
        'despliegue-prueba', spki, encode(sha256(spki), 'hex'),
        ahora - interval '5 minutes', ahora + interval '1 hour',
        'revocada', ahora - interval '4 minutes', 'acto:raiz:2'
    );
    INSERT INTO vec_ejecucion_documental_v4.clave_capacidad_version (
        clave_id, version, secreto_hmac, huella_secreto_sha256, emisor_id,
        valida_desde, valida_hasta, estado, revocada_en, acto_ref
    ) VALUES (
        'capacidad-prueba', 2, secreto, encode(sha256(secreto), 'hex'),
        'emisor-prueba', ahora - interval '5 minutes',
        ahora + interval '1 hour', 'revocada', ahora - interval '4 minutes',
        'acto:capacidad:2'
    );

    UPDATE vec_ejecucion_documental_v4.configuracion_confianza_actual
       SET revision = 'config-prueba-2',
           huella_configuracion_sha256 = repeat('2', 64),
           actualizada_en = ahora + interval '1 microsecond',
           acto_ref = 'acto:actual:2'
     WHERE control_id = true;
    UPDATE vec_ejecucion_documental_v4.raiz_confianza_actual
       SET version = 2, revision_configuracion = 'config-prueba-2',
           huella_configuracion_sha256 = repeat('2', 64),
           actualizada_en = ahora + interval '1 microsecond',
           acto_ref = 'acto:raiz-actual:2'
     WHERE clave_id = 'raiz-prueba';
    UPDATE vec_ejecucion_documental_v4.clave_capacidad_actual
       SET version = 2, actualizada_en = ahora + interval '1 microsecond',
           acto_ref = 'acto:capacidad-actual:2'
     WHERE control_id = true;

    rechazo := false;
    BEGIN
        UPDATE vec_ejecucion_documental_v4.configuracion_confianza_actual
           SET revision = 'config-prueba-1',
               huella_configuracion_sha256 = repeat('1', 64),
               actualizada_en = ahora + interval '2 microseconds'
         WHERE control_id = true;
    EXCEPTION WHEN SQLSTATE '55000' THEN rechazo := true;
    END;
    IF NOT rechazo THEN RAISE EXCEPTION 'se acepto rollback de configuracion'; END IF;

    rechazo := false;
    BEGIN
        UPDATE vec_ejecucion_documental_v4.raiz_confianza_actual
           SET version = 1, revision_configuracion = 'config-prueba-1',
               huella_configuracion_sha256 = repeat('1', 64),
               actualizada_en = ahora + interval '2 microseconds'
         WHERE clave_id = 'raiz-prueba';
    EXCEPTION WHEN SQLSTATE '55000' THEN rechazo := true;
    END;
    IF NOT rechazo THEN RAISE EXCEPTION 'se acepto rollback de raiz'; END IF;

    rechazo := false;
    BEGIN
        UPDATE vec_ejecucion_documental_v4.clave_capacidad_actual
           SET version = 1,
               actualizada_en = ahora + interval '2 microseconds'
         WHERE control_id = true;
    EXCEPTION WHEN SQLSTATE '55000' THEN rechazo := true;
    END;
    IF NOT rechazo THEN RAISE EXCEPTION 'se acepto rollback de capacidad'; END IF;

    INSERT INTO vec_ejecucion_documental_v4.configuracion_confianza_version (
        revision, huella_configuracion_sha256, publicada_en, expira_en,
        estado, revocada_en, acto_ref
    ) VALUES
        ('config-terminal', repeat('3', 64), ahora - interval '3 minutes',
         ahora + interval '1 hour', 'revocada', ahora - interval '2 minutes',
         'acto:config-terminal:1'),
        ('config-terminal', repeat('4', 64), ahora - interval '1 minute',
         ahora + interval '1 hour', 'activa', NULL,
         'acto:config-terminal:2');
    UPDATE vec_ejecucion_documental_v4.configuracion_confianza_actual
       SET revision = 'config-terminal',
           huella_configuracion_sha256 = repeat('3', 64),
           actualizada_en = ahora + interval '3 microseconds',
           acto_ref = 'acto:config-terminal-actual:1'
     WHERE control_id = true;
    rechazo := false;
    BEGIN
        UPDATE vec_ejecucion_documental_v4.configuracion_confianza_actual
           SET huella_configuracion_sha256 = repeat('4', 64),
               actualizada_en = ahora + interval '4 microseconds',
               acto_ref = 'acto:config-terminal-actual:2'
         WHERE control_id = true;
    EXCEPTION WHEN SQLSTATE '55000' THEN rechazo := true;
    END;
    IF NOT rechazo THEN
        RAISE EXCEPTION 'se resucito una revision de configuracion revocada';
    END IF;

    INSERT INTO vec_ejecucion_documental_v4.raiz_confianza_version (
        clave_id, version, revision_configuracion,
        huella_configuracion_sha256, algoritmo_cose, suite, audiencia_cose,
        audiencia_despliegue, clave_publica_spki, huella_clave_sha256,
        valida_desde, valida_hasta, estado, revocada_en, acto_ref
    ) VALUES (
        'raiz-prueba', 3, 'config-prueba-2', repeat('2', 64), 'EdDSA',
        'VEC-AD-COSE-EDDSA-1', 'atestacion_autorizacion_pdp',
        'despliegue-prueba', spki, encode(sha256(spki), 'hex'),
        ahora - interval '1 minute', ahora + interval '1 hour',
        'activa', NULL, 'acto:raiz:3'
    );
    rechazo := false;
    BEGIN
        UPDATE vec_ejecucion_documental_v4.raiz_confianza_actual
           SET version = 3, actualizada_en = ahora + interval '3 microseconds',
               acto_ref = 'acto:raiz-actual:3'
         WHERE clave_id = 'raiz-prueba';
    EXCEPTION WHEN SQLSTATE '55000' THEN rechazo := true;
    END;
    IF NOT rechazo THEN RAISE EXCEPTION 'se resucito una raiz revocada'; END IF;

    INSERT INTO vec_ejecucion_documental_v4.raiz_confianza_version (
        clave_id, version, revision_configuracion,
        huella_configuracion_sha256, algoritmo_cose, suite, audiencia_cose,
        audiencia_despliegue, clave_publica_spki, huella_clave_sha256,
        valida_desde, valida_hasta, estado, revocada_en, acto_ref
    ) VALUES (
        'raiz-alias', 1, 'config-prueba-2', repeat('2', 64), 'EdDSA',
        'VEC-AD-COSE-EDDSA-1', 'atestacion_autorizacion_pdp',
        'despliegue-prueba', spki, encode(sha256(spki), 'hex'),
        ahora - interval '1 minute', ahora + interval '1 hour',
        'activa', NULL, 'acto:raiz-alias:1'
    );
    rechazo := false;
    BEGIN
        INSERT INTO vec_ejecucion_documental_v4.raiz_confianza_actual (
            clave_id, version, revision_configuracion,
            huella_configuracion_sha256, actualizada_en, acto_ref
        ) VALUES (
            'raiz-alias', 1, 'config-prueba-2', repeat('2', 64),
            ahora + interval '4 microseconds', 'acto:raiz-alias-actual:1'
        );
    EXCEPTION WHEN SQLSTATE '55000' THEN rechazo := true;
    END;
    IF NOT rechazo THEN RAISE EXCEPTION 'se resucito una SPKI bajo alias'; END IF;

    INSERT INTO vec_ejecucion_documental_v4.clave_capacidad_version (
        clave_id, version, secreto_hmac, huella_secreto_sha256, emisor_id,
        valida_desde, valida_hasta, estado, revocada_en, acto_ref
    ) VALUES (
        'capacidad-nueva', 3, secreto, encode(sha256(secreto), 'hex'),
        'emisor-prueba', ahora - interval '1 minute',
        ahora + interval '1 hour', 'activa', NULL, 'acto:capacidad:3'
    );
    rechazo := false;
    BEGIN
        UPDATE vec_ejecucion_documental_v4.clave_capacidad_actual
           SET clave_id = 'capacidad-nueva', version = 3,
               actualizada_en = ahora + interval '3 microseconds',
               acto_ref = 'acto:capacidad-actual:3'
         WHERE control_id = true;
    EXCEPTION WHEN SQLSTATE '55000' THEN rechazo := true;
    END;
    IF NOT rechazo THEN RAISE EXCEPTION 'se resucito una clave de capacidad'; END IF;

    rechazo := false;
    BEGIN
        DELETE FROM vec_ejecucion_documental_v4.configuracion_confianza_actual;
    EXCEPTION WHEN SQLSTATE '55000' THEN rechazo := true;
    END;
    IF NOT rechazo THEN RAISE EXCEPTION 'se elimino un puntero actual'; END IF;

    rechazo := false;
    BEGIN
        EXECUTE 'TRUNCATE vec_ejecucion_documental_v4.clave_capacidad_actual';
    EXCEPTION WHEN SQLSTATE '55000' THEN rechazo := true;
    END;
    IF NOT rechazo THEN RAISE EXCEPTION 'se trunco un puntero actual'; END IF;

    rechazo := false;
    BEGIN
        UPDATE vec_ejecucion_documental_v4.clave_capacidad_version
           SET secreto_hmac = decode(repeat('ef', 32), 'hex')
         WHERE version = 1;
    EXCEPTION WHEN SQLSTATE '55000' THEN rechazo := true;
    END;
    IF NOT rechazo THEN RAISE EXCEPTION 'se muto una clave versionada'; END IF;
END
$transiciones$;

ROLLBACK;
