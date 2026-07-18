-- Prueba integral del corte durable. Todo queda dentro de una transaccion y
-- se revierte; las carreras entre sesiones se prueban desde el runbook shell.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL ROLE vec_bolsa_convocatorias_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

CREATE FUNCTION vec_bolsa_convocatorias.inyectar_fallo_outbox_borrador_prueba()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, pg_temp
AS $funcion$
BEGIN
    RAISE EXCEPTION USING ERRCODE = 'P0001',
        MESSAGE = 'fallo_outbox_borrador_inyectado';
END
$funcion$;

DO $pruebas$
DECLARE
    ahora timestamptz := date_trunc('microseconds', clock_timestamp());
    ahora_texto text;
    emitida_texto text;
    vence_texto text;
    lease_fin_texto text;
    version_json jsonb;
    version_canonica bytea;
    huella_estado text;
    huella_estado_anterior text;
    material_json jsonb;
    material_canonico bytea;
    huella_material text;
    contexto_json jsonb;
    contexto_canonico bytea;
    huella_contexto text;
    decision_json jsonb;
    decision_canonica bytea;
    huella_decision text;
    huella_atestacion text := encode(
        sha256(convert_to('{}', 'UTF8')), 'hex'
    );
    atestacion_json jsonb;
    decision_proyeccion jsonb;
    identidad jsonb;
    identidad_historica jsonb;
    identidad_rotada jsonb;
    identidad_alias_g1 jsonb;
    identidad_alias_g4 jsonb;
    identidad_actualizacion jsonb;
    reserva jsonb;
    sellado jsonb;
    envoltura jsonb;
    proyeccion jsonb;
    confirmacion jsonb;
    sobre_cifrado bytea := decode(repeat('cd', 32), 'hex');
    resultado record;
    consulta record;
    detalle record;
    lectura jsonb;
    lista jsonb;
    total bigint;
    mensaje_error text;
BEGIN
    ahora_texto := to_char(ahora AT TIME ZONE 'UTC',
        'YYYY-MM-DD"T"HH24:MI:SS.US"Z"');
    emitida_texto := to_char((ahora - interval '1 second') AT TIME ZONE 'UTC',
        'YYYY-MM-DD"T"HH24:MI:SS.US"Z"');
    vence_texto := to_char((ahora + interval '4 minutes') AT TIME ZONE 'UTC',
        'YYYY-MM-DD"T"HH24:MI:SS.US"Z"');
    lease_fin_texto := to_char((ahora + interval '2 minutes') AT TIME ZONE 'UTC',
        'YYYY-MM-DD"T"HH24:MI:SS.US"Z"');

    version_json := jsonb_build_object(
        'id', 'convocatoria-prueba', 'secuencia', 1,
        'codigo_version_publica', 'v1', 'revision', 1,
        'instancia_flujo_ref', 'flujo-prueba',
        'ambito_organizativo', jsonb_build_object(
            'organizacion_ref', 'org_0123456789abcdef',
            'unidad_gestion_ref', 'uni_0123456789abcdef'
        ),
        'contenido', jsonb_build_object(
            'identificador_publico', 'aux-2026', 'tipo', 'bolsa',
            'catalogo_categorias', jsonb_build_object(
                'id', 'categorias', 'version', 1,
                'huella_contenido_sha256', repeat('1', 64)
            ),
            'categorias', jsonb_build_array('auxiliar'),
            'titulo', 'Bolsa auxiliar sintetica',
            'resumen', 'Resumen sintetico',
            'descripcion', 'Descripcion sintetica',
            'plazos', jsonb_build_array(jsonb_build_object(
                'referencia', 'plazo-1'
            )),
            'requisitos', '[]'::jsonb,
            'documentos', jsonb_build_array(jsonb_build_object(
                'referencia', 'documento-1'
            )),
            'ayuda', '[]'::jsonb
        ),
        'configuracion', jsonb_build_object('sintetica', true),
        'expediente_ref', 'expediente-prueba',
        'motivo_creacion', 'NO_DEBE_PERSISTIR_EN_CLARO',
        'estado_gobierno', 'borrador',
        'creada_por', 'PRINCIPAL_NO_DEBE_PERSISTIR',
        'creada_en', ahora_texto
    );
    version_canonica := convert_to(version_json::text, 'UTF8');
    huella_estado := encode(sha256(version_canonica), 'hex');
    material_json := jsonb_build_object(
        'esquema', 'bolsa.convocatoria.intencion.v2',
        'accion', 'bolsa.convocatoria.borrador.crear',
        'estado_principal_nuevo', jsonb_build_object(
            'referencia', 'convocatoria-prueba#1',
            'revision', 1, 'huella_estado_sha256', huella_estado
        ),
        'dominio_criptografico_motivo',
            'bolsa.convocatoria.motivo.v1',
        'generacion_clave_motivo', 3,
        'huella_motivo_hmac_sha256',
            'hmac-sha256:motivo-gobierno-v3:' || repeat('a', 64)
    );
    material_canonico := convert_to(material_json::text, 'UTF8');
    huella_material := encode(sha256(material_canonico), 'hex');
    contexto_json := jsonb_build_object(
        'ambitos', version_json -> 'ambito_organizativo',
        'atributos', jsonb_build_object(
            'huella_intencion_sha256', huella_material
        )
    );
    contexto_canonico := convert_to(contexto_json::text, 'UTF8');
    huella_contexto := encode(sha256(contexto_canonico), 'hex');
    decision_json := jsonb_build_object(
        'esquema',
            'vec.autorizacion.decision.reforzada.v1.autenticacion-actor',
        'decision_ref', 'decision-borrador-prueba-1',
        'concedida', true, 'codigo', 'concedida',
        'principal_id', 'PRINCIPAL_EFIMERO',
        'perfil_activo_ref', 'perfil-rrhh',
        'accion', 'bolsa.convocatoria.borrador.crear',
        'recurso_ref', 'convocatoria-prueba#1',
        'modulo_id', 'bolsa',
        'tipo_recurso', 'version_convocatoria_gobernada',
        'contexto_recurso_huella_sha256', huella_contexto,
        'finalidad', 'gobierno_convocatorias',
        'correlacion_ref', 'correlacion-prueba',
        'asignacion_ref', 'asignacion-rrhh-prueba',
        'asignacion_huella_sha256', repeat('6', 64),
        'version_rol_ref', 'rol-rrhh-v1',
        'version_rol_huella_sha256', repeat('2', 64),
        'control_vigencia_version_rol_ref', 'rol-rrhh-v1',
        'control_vigencia_version_rol_revision', 1,
        'control_vigencia_version_rol_huella_sha256', repeat('7', 64),
        'revision_catalogo_politicas', 1,
        'catalogo_politicas_huella_sha256', repeat('3', 64),
        'campos_permitidos', jsonb_build_array(
            'auditoria', 'evento_outbox', 'version_convocatoria'
        ),
        'obligaciones', '[]'::jsonb,
        'garantia_minima', 'alto',
        'emitida_en', emitida_texto, 'valida_hasta', vence_texto
    );
    decision_canonica := convert_to(decision_json::text, 'UTF8');
    huella_decision := encode(sha256(decision_canonica), 'hex');

    INSERT INTO vec_bolsa_convocatorias.atestacion_autorizacion_version(
        decision_ref, atestacion_ref, version, estado,
        huella_decision_sha256, evidencia_canonica,
        huella_evidencia_sha256, sobre_cose_sign1,
        huella_sobre_sha256, clave_id, revision_confianza,
        verificada_en, valida_desde, valida_hasta, registrada_en
    ) VALUES (
        'decision-borrador-prueba-1', 'atestacion-pdp-prueba-1',
        1, 'activa', huella_decision, convert_to('{}', 'UTF8'),
        huella_atestacion, decode(repeat('ab', 16), 'hex'),
        encode(sha256(decode(repeat('ab', 16), 'hex')), 'hex'),
        'clave-pdp-prueba', 'confianza-prueba', ahora,
        ahora - interval '1 minute', ahora + interval '5 minutes', ahora
    );
    INSERT INTO vec_bolsa_convocatorias.atestacion_autorizacion_actual
    VALUES (
        'decision-borrador-prueba-1', 'atestacion-pdp-prueba-1',
        1, 'activa', ahora
    );
    INSERT INTO vec_bolsa_convocatorias.atestacion_pdp_borrador
    VALUES (
        'decision-borrador-prueba-1', 'atestacion-pdp-prueba-1', 1,
        'activa', huella_decision, huella_atestacion,
        'verificador-pdp-prueba', ahora, ahora
    );
    atestacion_json := jsonb_build_object(
        'decision_ref', 'decision-borrador-prueba-1',
        'atestacion_ref', 'atestacion-pdp-prueba-1',
        'version', 1, 'estado', 'activa',
        'huella_atestacion_sha256', huella_atestacion,
        'verificador_ref', 'verificador-pdp-prueba',
        'verificada_en', ahora_texto
    );
    decision_proyeccion := jsonb_build_object(
        'esquema_huella',
            'vec.autorizacion.decision.reforzada.v1.autenticacion-actor',
        'decision_ref', 'decision-borrador-prueba-1',
        'huella_decision_sha256', huella_decision,
        'accion', 'bolsa.convocatoria.borrador.crear',
        'recurso_ref', 'convocatoria-prueba#1',
        'modulo_id', 'bolsa',
        'tipo_recurso', 'version_convocatoria_gobernada',
        'contexto_recurso_huella_sha256', huella_contexto,
        'finalidad', 'gobierno_convocatorias',
        'asignacion_ref', 'asignacion-rrhh-prueba',
        'asignacion_huella_sha256', repeat('6', 64),
        'version_rol_ref', 'rol-rrhh-v1',
        'version_rol_huella_sha256', repeat('2', 64),
        'control_vigencia_version_rol_ref', 'rol-rrhh-v1',
        'control_vigencia_version_rol_revision', 1,
        'control_vigencia_version_rol_huella_sha256', repeat('7', 64),
        'revision_catalogo_politicas', 1,
        'catalogo_politicas_huella_sha256', repeat('3', 64),
        'emitida_en', emitida_texto,
        'verificada_en', ahora_texto, 'valida_hasta', vence_texto,
        'atestacion_pdp', atestacion_json
    );
    identidad := jsonb_build_object(
        'localizador', jsonb_build_object(
            'version_esquema', 1, 'dominio', 'localizador',
            'clave_ref', 'clave:hmac:convocatorias:localizador:prueba-g3',
            'generacion_clave', 3, 'hmac_sha256', repeat('4', 64)
        ),
        'huella_solicitud', jsonb_build_object(
            'version_esquema', 1, 'dominio', 'huella_solicitud',
            'clave_ref', 'clave:hmac:convocatorias:huella:prueba-g3',
            'generacion_clave', 3, 'hmac_sha256', repeat('5', 64)
        )
    );
    identidad_historica := jsonb_build_object(
        'localizador', jsonb_build_object(
            'version_esquema', 1, 'dominio', 'localizador',
            'clave_ref', 'clave:hmac:convocatorias:localizador:prueba-g2',
            'generacion_clave', 2, 'hmac_sha256', repeat('6', 64)
        ),
        'huella_solicitud', jsonb_build_object(
            'version_esquema', 1, 'dominio', 'huella_solicitud',
            'clave_ref', 'clave:hmac:convocatorias:huella:prueba-g2',
            'generacion_clave', 2, 'hmac_sha256', repeat('7', 64)
        )
    );
    reserva := jsonb_build_object(
        'esquema', 'vec.bolsa.convocatoria.reserva-decision.v2',
        'identidad', identidad,
        'identidades_consulta', jsonb_build_array(
            identidad, identidad_historica
        ),
        'accion', 'bolsa.convocatoria.borrador.crear',
        'huella_material_sha256', huella_material,
        'recurso_ref', 'convocatoria-prueba#1',
        'contexto_recurso_huella_sha256', huella_contexto,
        'solicitada_en', ahora_texto,
        'arrendamiento_inicia_en', ahora_texto,
        'arrendamiento_vence_en', lease_fin_texto,
        'decision', decision_proyeccion
    );
    SELECT * INTO STRICT resultado
      FROM vec_bolsa_convocatorias.reservar_decision_borrador_interna_v1(
          reserva, material_canonico, version_canonica,
          decision_canonica, contexto_canonico
      );
    IF resultado.estado <> 'reservado' OR resultado.revision <> 1
       OR resultado.cercado <> 1
       OR resultado.identidades_consultadas IS DISTINCT FROM
          jsonb_build_array(identidad, identidad_historica)
       OR resultado.identidad_primaria IS DISTINCT FROM identidad
       OR (SELECT count(*)
             FROM vec_bolsa_convocatorias.identidad_alias_borrador) <> 2 THEN
        RAISE EXCEPTION 'reserva separada incompleta: %', resultado;
    END IF;
    IF (resultado.identidades_consultadas::text ||
        resultado.identidad_primaria::text) ~ '(principal|motivo)' THEN
        RAISE EXCEPTION
            'la resolucion de identidad expuso principal o motivo';
    END IF;
    SELECT * INTO STRICT consulta
      FROM vec_bolsa_convocatorias.consultar_diario_borrador_interna_v1(
          identidad
      );
    IF consulta.estado <> 'reservado' OR consulta.revision <> 1 THEN
        RAISE EXCEPTION 'consulta del diario no releyo la reserva';
    END IF;

    sellado := jsonb_build_object(
        'accion', 'bolsa.convocatoria.borrador.crear',
        'convocatoria_ref', 'convocatoria-prueba#1',
        'hmac', jsonb_build_object(
            'dominio_criptografico', 'bolsa.convocatoria.motivo.v1',
            'generacion_clave', 3,
            'clave_hmac_ref', 'motivo-gobierno-v3',
            'valor_hmac_sha256', repeat('a', 64)
        ),
        'atestacion_ref', 'atestacion-motivo-prueba-1',
        'version_atestacion', 1, 'estado_atestacion', 'verificada',
        'huella_atestacion_sha256', repeat('6', 64),
        'token_consumo_ref', 'consumo-motivo-prueba-1',
        'materializador_ref', 'hsm-motivo-prueba',
        'atestacion_emitida_en', ahora_texto,
        'atestacion_valida_hasta', to_char(
            (ahora + interval '3 minutes') AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        )
    );
    envoltura := jsonb_build_object(
        'algoritmo', 'A256GCM', 'clave_cifrado_ref', 'kms-borradores-v1',
        'generacion_clave', 1, 'nonce_hex', repeat('1a', 12),
        'etiqueta_autenticacion_hex', repeat('2b', 16),
        'atestacion_cifrado_ref', 'atestacion-cifrado-prueba-1',
        'huella_atestacion_cifrado_sha256', repeat('7', 64),
        'huella_sobre_cifrado_sha256',
            encode(sha256(sobre_cifrado), 'hex')
    );
    proyeccion := jsonb_build_object(
        'convocatoria_id', 'convocatoria-prueba', 'secuencia', 1,
        'referencia', 'convocatoria-prueba#1', 'revision', 1,
        'huella_estado_sha256', huella_estado,
        'codigo_version_publica', 'v1',
        'identificador_publico', 'aux-2026',
        'titulo', 'Bolsa auxiliar sintetica', 'tipo', 'bolsa',
        'categorias', jsonb_build_array('auxiliar'),
        'expediente_ref', 'expediente-prueba',
        'organizacion_ref', 'org_0123456789abcdef',
        'unidad_gestion_ref', 'uni_0123456789abcdef',
        'numero_plazos', 1, 'numero_requisitos', 0,
        'numero_documentos', 1, 'numero_ayudas', 0,
        'creada_en', ahora_texto, 'actualizada_en', ahora_texto
    );
    confirmacion := jsonb_build_object(
        'esquema', 'vec.bolsa.convocatoria.confirmacion-borrador.v2',
        'identidad', identidad, 'revision', 1, 'cercado', 1,
        'solicitada_en', ahora_texto, 'sellado_motivo', sellado,
        'envoltura_cifrado', envoltura, 'proyeccion_ligera', proyeccion
    );

    -- Fault injection real en el ultimo componente previo al recibo. La
    -- excepcion debe deshacer agregado, consumo HSM, auditoria, outbox,
    -- en_curso y punteros, dejando solamente la reserva previa.
    EXECUTE 'CREATE TRIGGER fallo_outbox_borrador_prueba '
         || 'BEFORE INSERT ON vec_bolsa_convocatorias.outbox_borrador '
         || 'FOR EACH ROW EXECUTE FUNCTION '
         || 'vec_bolsa_convocatorias.'
         || 'inyectar_fallo_outbox_borrador_prueba()';
    BEGIN
        PERFORM *
          FROM vec_bolsa_convocatorias.confirmar_borrador_interna_v1(
              confirmacion, material_canonico, version_canonica,
              sobre_cifrado
          );
        RAISE EXCEPTION 'la inyeccion outbox no interrumpio el COMMIT';
    EXCEPTION WHEN raise_exception THEN
        IF SQLERRM <> 'fallo_outbox_borrador_inyectado' THEN
            RAISE;
        END IF;
    END;
    EXECUTE 'DROP TRIGGER fallo_outbox_borrador_prueba '
         || 'ON vec_bolsa_convocatorias.outbox_borrador';
    SELECT * INTO STRICT consulta
      FROM vec_bolsa_convocatorias.consultar_diario_borrador_interna_v1(
          identidad
      );
    IF consulta.estado <> 'reservado' OR consulta.revision <> 1
       OR (SELECT count(*) FROM
               vec_bolsa_convocatorias.borrador_convocatoria_version) <> 0
       OR (SELECT count(*) FROM
               vec_bolsa_convocatorias.sellado_motivo_borrador) <> 0
       OR (SELECT count(*) FROM
               vec_bolsa_convocatorias.auditoria_borrador) <> 0
       OR (SELECT count(*) FROM
               vec_bolsa_convocatorias.outbox_borrador) <> 0
       OR (SELECT count(*) FROM
               vec_bolsa_convocatorias.diario_borrador_version) <> 1 THEN
        RAISE EXCEPTION 'rollback parcial tras fallo outbox: %', consulta;
    END IF;

    SELECT * INTO STRICT resultado
      FROM vec_bolsa_convocatorias.confirmar_borrador_interna_v1(
          confirmacion, material_canonico, version_canonica, sobre_cifrado
      );
    IF resultado.resultado <> 'confirmada'
       OR resultado.estado_diario <> 'confirmado'
       OR resultado.revision_diario <> 3
       OR resultado.estado_principal_ref <> 'convocatoria-prueba#1'
       OR resultado.estado_principal_revision <> 1
       OR resultado.estado_principal_huella_sha256 <> huella_estado
       OR resultado.transaccion_ref IS NULL
       OR resultado.auditoria_ref IS NULL
       OR resultado.huella_auditoria_sha256 IS NULL
       OR resultado.evento_outbox_ref IS NULL
       OR resultado.huella_evento_outbox_sha256 IS NULL
       OR resultado.recibo ->> 'esquema' <>
          'bolsa.convocatoria.borrador.recibo.v2'
       OR vec_bolsa_convocatorias.objeto_json_exacto(
           resultado.recibo, ARRAY[
               'accion','arrendamiento_inicia_en',
               'arrendamiento_vence_en','auditoria_ref',
               'cercado_confirmado','confirmada_en','decision','esquema',
               'estado_principal','evento_outbox_ref',
               'huella_auditoria_sha256','huella_evento_outbox_sha256',
               'identidad','recibo_ref','revision_confirmada',
               'sellado_motivo','transaccion_ref'
           ]
       ) IS NOT TRUE
       OR vec_bolsa_convocatorias.objeto_json_exacto(
           resultado.recibo -> 'decision', ARRAY[
               'accion','asignacion_huella_sha256','asignacion_ref',
               'atestacion_pdp','catalogo_politicas_huella_sha256',
               'contexto_recurso_huella_sha256',
               'control_vigencia_version_rol_huella_sha256',
               'control_vigencia_version_rol_ref',
               'control_vigencia_version_rol_revision','decision_ref',
               'emitida_en','esquema_huella','finalidad',
               'huella_decision_sha256','modulo_id','recurso_ref',
               'revision_catalogo_politicas','tipo_recurso','valida_hasta',
               'verificada_en','version_rol_huella_sha256',
               'version_rol_ref'
           ]
       ) IS NOT TRUE
       OR vec_bolsa_convocatorias.objeto_json_exacto(
           resultado.recibo -> 'sellado_motivo', ARRAY[
               'accion','atestacion_emitida_en','atestacion_ref',
               'atestacion_valida_hasta','convocatoria_ref',
               'estado_atestacion','hmac','huella_atestacion_sha256',
               'materializador_ref','token_consumo_ref',
               'version_atestacion'
           ]
       ) IS NOT TRUE
       OR vec_bolsa_convocatorias.objeto_json_exacto(
           resultado.recibo #> '{sellado_motivo,hmac}', ARRAY[
               'clave_hmac_ref','dominio_criptografico',
               'generacion_clave','valor_hmac_sha256'
           ]
       ) IS NOT TRUE
       OR resultado.recibo ->> 'recibo_ref' IS NULL
       OR resultado.recibo #>> '{identidad,localizador,hmac_sha256}' <>
          repeat('4', 64)
       OR resultado.recibo #>> '{decision,decision_ref}' <>
          'decision-borrador-prueba-1'
       OR resultado.recibo #>> '{decision,asignacion_ref}' <>
          'asignacion-rrhh-prueba'
       OR resultado.recibo #>>
          '{decision,control_vigencia_version_rol_ref}' <> 'rol-rrhh-v1'
       OR resultado.recibo #>> '{decision,emitida_en}' <> emitida_texto
       OR resultado.recibo #>>
          '{sellado_motivo,hmac,valor_hmac_sha256}' <> repeat('a', 64)
       OR resultado.recibo #>> '{sellado_motivo,token_consumo_ref}' <>
          'consumo-motivo-prueba-1' THEN
        RAISE EXCEPTION 'recibo atomico incompleto: %', resultado;
    END IF;
    IF (SELECT count(*)
          FROM vec_bolsa_convocatorias.diario_borrador_version) <> 3
       OR (SELECT count(*)
             FROM vec_bolsa_convocatorias.uso_decision_borrador) <> 1
       OR (SELECT count(*)
             FROM vec_bolsa_convocatorias.sellado_motivo_borrador) <> 1
       OR (SELECT count(*)
             FROM vec_bolsa_convocatorias.auditoria_borrador) <> 1
       OR (SELECT count(*)
             FROM vec_bolsa_convocatorias.outbox_borrador) <> 1 THEN
        RAISE EXCEPTION 'confirmacion no fue atomica';
    END IF;
    SELECT * INTO STRICT consulta
      FROM vec_bolsa_convocatorias.reconciliar_operacion_borrador_interna_v1(
          identidad, 'reservado', 1, 1, clock_timestamp()
      );
    IF consulta.estado <> 'confirmado' OR consulta.revision <> 3
       OR consulta.cercado <> 1 OR consulta.recibo IS NULL THEN
        RAISE EXCEPTION 'reconciliacion confirmada rompio revision/fence: %',
            consulta;
    END IF;

    -- Replay terminal: no vuelve a consumir decision, HSM, auditoria u outbox.
    SELECT * INTO STRICT resultado
      FROM vec_bolsa_convocatorias.confirmar_borrador_interna_v1(
          confirmacion, material_canonico, version_canonica, sobre_cifrado
      );
    IF resultado.resultado <> 'idempotencia_reutilizada'
       OR (SELECT count(*)
             FROM vec_bolsa_convocatorias.auditoria_borrador) <> 1 THEN
        RAISE EXCEPTION 'replay terminal no fue estable';
    END IF;

    -- Rotar/revocar la atestacion despues del COMMIT no puede cambiar el
    -- veredicto temporal del mismo L/F. Si se intenta una operacion nueva, en
    -- cambio, la relectura del puntero actual debe fallar cerrada.
    INSERT INTO vec_bolsa_convocatorias.atestacion_autorizacion_version(
        decision_ref, atestacion_ref, version, estado,
        huella_decision_sha256, evidencia_canonica,
        huella_evidencia_sha256, sobre_cose_sign1,
        huella_sobre_sha256, clave_id, revision_confianza,
        verificada_en, valida_desde, valida_hasta, registrada_en
    ) VALUES (
        'decision-borrador-prueba-1', 'atestacion-pdp-prueba-revocada',
        2, 'revocada', huella_decision, convert_to('{}', 'UTF8'),
        huella_atestacion, decode(repeat('ac', 16), 'hex'),
        encode(sha256(decode(repeat('ac', 16), 'hex')), 'hex'),
        'clave-pdp-prueba-rotada', 'confianza-prueba-2', ahora,
        ahora - interval '1 minute', ahora + interval '5 minutes', ahora
    );
    UPDATE vec_bolsa_convocatorias.atestacion_autorizacion_actual
       SET atestacion_ref = 'atestacion-pdp-prueba-revocada',
           version = 2, estado = 'revocada',
           actualizada_en = ahora + interval '1 microsecond'
     WHERE decision_ref = 'decision-borrador-prueba-1';
    SELECT * INTO STRICT resultado
      FROM vec_bolsa_convocatorias.confirmar_borrador_interna_v1(
          confirmacion, material_canonico, version_canonica, sobre_cifrado
      );
    IF resultado.resultado <> 'idempotencia_reutilizada'
       OR resultado.estado_diario <> 'confirmado' THEN
        RAISE EXCEPTION 'replay cambio tras revocacion PDP: %', resultado;
    END IF;
    identidad_rotada := jsonb_build_object(
        'localizador', jsonb_build_object(
            'version_esquema', 1, 'dominio', 'localizador',
            'clave_ref', 'clave:hmac:convocatorias:localizador:prueba-g1',
            'generacion_clave', 1, 'hmac_sha256', repeat('8', 64)
        ),
        'huella_solicitud', jsonb_build_object(
            'version_esquema', 1, 'dominio', 'huella_solicitud',
            'clave_ref', 'clave:hmac:convocatorias:huella:prueba-g1',
            'generacion_clave', 1, 'hmac_sha256', repeat('9', 64)
        )
    );
    identidad_alias_g1 := identidad_rotada;
    SELECT * INTO STRICT consulta
      FROM vec_bolsa_convocatorias.consultar_identidades_borrador_interna_v1(
          jsonb_build_array(identidad_historica, identidad_rotada)
      );
    IF consulta.estado <> 'confirmado' OR consulta.revision <> 3
       OR consulta.identidades_consultadas IS DISTINCT FROM
          jsonb_build_array(identidad_historica)
       OR consulta.identidad_primaria IS DISTINCT FROM identidad
       OR consulta.recibo IS NULL THEN
        RAISE EXCEPTION 'consulta rotada no encontro la identidad historica: %',
            consulta;
    END IF;
    SELECT * INTO STRICT resultado
      FROM vec_bolsa_convocatorias.reservar_decision_borrador_interna_v1(
          reserva || jsonb_build_object(
              'identidad', identidad_historica,
              'identidades_consulta',
                  jsonb_build_array(identidad_historica, identidad_rotada)
          ), material_canonico, version_canonica,
          decision_canonica, contexto_canonico
      );
    IF resultado.estado <> 'confirmado' OR resultado.revision <> 3
       OR resultado.identidades_consultadas IS DISTINCT FROM
          jsonb_build_array(identidad_historica, identidad_rotada)
       OR resultado.identidad_primaria IS DISTINCT FROM identidad
       OR resultado.recibo IS NULL THEN
        RAISE EXCEPTION 'reserva replay perdio veredicto temporal: %',
            resultado;
    END IF;
    IF (SELECT count(*)
          FROM vec_bolsa_convocatorias.identidad_alias_borrador AS a
         WHERE a.primario_localizador_hmac = decode(repeat('4', 64), 'hex'))
       <> 3 THEN
        RAISE EXCEPTION 'la ventana deslizante no retuvo g3/g2/g1';
    END IF;
    SELECT * INTO STRICT consulta
      FROM vec_bolsa_convocatorias.consultar_identidades_borrador_interna_v1(
          jsonb_build_array(identidad_rotada)
      );
    IF consulta.estado <> 'confirmado'
       OR consulta.identidades_consultadas IS DISTINCT FROM
          jsonb_build_array(identidad_rotada)
       OR consulta.identidad_primaria IS DISTINCT FROM identidad THEN
        RAISE EXCEPTION 'el alias g1 incorporado no quedo durable: %',
            consulta;
    END IF;
    identidad_alias_g4 := jsonb_build_object(
        'localizador', jsonb_build_object(
            'version_esquema', 1, 'dominio', 'localizador',
            'clave_ref', 'clave:hmac:convocatorias:localizador:prueba-g4',
            'generacion_clave', 4, 'hmac_sha256', repeat('a', 64)
        ),
        'huella_solicitud', jsonb_build_object(
            'version_esquema', 1, 'dominio', 'huella_solicitud',
            'clave_ref', 'clave:hmac:convocatorias:huella:prueba-g4',
            'generacion_clave', 4, 'hmac_sha256', repeat('b', 64)
        )
    );
    IF vec_bolsa_convocatorias.identidades_operacion_borrador_validas(
           jsonb_build_array(identidad)
       ) IS NOT TRUE
       OR vec_bolsa_convocatorias.identidades_operacion_borrador_validas(
           jsonb_build_array(identidad, identidad_historica)
       ) IS NOT TRUE
       OR vec_bolsa_convocatorias.identidades_operacion_borrador_validas(
           jsonb_build_array(
               identidad, identidad_historica, identidad_alias_g1
           )
       ) IS NOT TRUE
       OR vec_bolsa_convocatorias.identidades_operacion_borrador_validas(
           jsonb_build_array(
               identidad_alias_g4, identidad, identidad_historica,
               identidad_alias_g1
           )
       ) IS NOT TRUE THEN
        RAISE EXCEPTION 'las cardinalidades validas 1..4 fueron rechazadas';
    END IF;
    SELECT * INTO STRICT consulta
      FROM vec_bolsa_convocatorias.consultar_identidades_borrador_interna_v1(
          jsonb_build_array(
              identidad, identidad_historica, identidad_alias_g1
          )
      );
    IF consulta.estado <> 'confirmado' OR consulta.recibo IS NULL
       OR consulta.identidades_consultadas IS DISTINCT FROM
          jsonb_build_array(
              identidad, identidad_historica, identidad_alias_g1
          )
       OR consulta.identidad_primaria IS DISTINCT FROM identidad THEN
        RAISE EXCEPTION 'consulta de tres aliases no alcanzo el veredicto: %',
            consulta;
    END IF;
    SELECT * INTO STRICT resultado
      FROM vec_bolsa_convocatorias.reservar_decision_borrador_interna_v1(
          reserva || jsonb_build_object(
              'identidad', identidad_alias_g4,
              'identidades_consulta', jsonb_build_array(
                  identidad_alias_g4, identidad, identidad_historica,
                  identidad_alias_g1
              )
          ), material_canonico, version_canonica,
          decision_canonica, contexto_canonico
      );
    IF resultado.estado <> 'confirmado' OR resultado.recibo IS NULL
       OR resultado.identidades_consultadas IS DISTINCT FROM
          jsonb_build_array(
              identidad_alias_g4, identidad, identidad_historica,
              identidad_alias_g1
          )
       OR resultado.identidad_primaria IS DISTINCT FROM identidad
       OR (SELECT count(*)
             FROM vec_bolsa_convocatorias.identidad_alias_borrador AS a
            WHERE a.primario_localizador_hmac =
                  decode(repeat('4', 64), 'hex')) <> 4 THEN
        RAISE EXCEPTION 'ventana de cuatro aliases no quedo durable: %',
            resultado;
    END IF;
    identidad_rotada := jsonb_build_object(
        'localizador', jsonb_build_object(
            'version_esquema', 1, 'dominio', 'localizador',
            'clave_ref', 'clave:hmac:convocatorias:localizador:nueva-g4',
            'generacion_clave', 4, 'hmac_sha256', repeat('d', 64)
        ),
        'huella_solicitud', jsonb_build_object(
            'version_esquema', 1, 'dominio', 'huella_solicitud',
            'clave_ref', 'clave:hmac:convocatorias:huella:nueva-g4',
            'generacion_clave', 4, 'hmac_sha256', repeat('e', 64)
        )
    );
    BEGIN
        PERFORM *
          FROM vec_bolsa_convocatorias.reservar_decision_borrador_interna_v1(
              reserva || jsonb_build_object(
                  'identidad', identidad_rotada,
                  'identidades_consulta', jsonb_build_array(
                      identidad_rotada, identidad
                  )
              ), material_canonico, version_canonica,
              decision_canonica, contexto_canonico
          );
        RAISE EXCEPTION
            'se acepto una segunda pareja L/F de la generacion g4';
    EXCEPTION WHEN unique_violation THEN
        GET STACKED DIAGNOSTICS mensaje_error = MESSAGE_TEXT;
        IF mensaje_error <>
           'pareja L/F alternativa para generacion primaria' THEN
            RAISE EXCEPTION 'rechazo generacional no determinista: %',
                mensaje_error;
        END IF;
    END;
    IF (SELECT count(*)
          FROM vec_bolsa_convocatorias.identidad_alias_borrador AS a
         WHERE a.primario_localizador_hmac =
               decode(repeat('4', 64), 'hex')) <> 4 THEN
        RAISE EXCEPTION
            'el rechazo generacional altero los aliases durables';
    END IF;
    BEGIN
        PERFORM *
          FROM vec_bolsa_convocatorias.reservar_decision_borrador_interna_v1(
              reserva || jsonb_build_object(
                  'identidad', identidad_rotada,
                  'identidades_consulta', jsonb_build_array(identidad_rotada)
              ), material_canonico, version_canonica,
              decision_canonica, contexto_canonico
          );
        RAISE EXCEPTION 'alta nueva acepto una atestacion revocada';
    EXCEPTION WHEN insufficient_privilege THEN
        NULL;
    END;

    -- Mismo L con F diferente nunca se degrada a un replay.
    SELECT * INTO STRICT consulta
      FROM vec_bolsa_convocatorias.consultar_diario_borrador_interna_v1(
          jsonb_set(
              identidad, '{huella_solicitud,hmac_sha256}',
              to_jsonb(repeat('8', 64))
          )
      );
    IF consulta.estado <> 'conflicto' THEN
        RAISE EXCEPTION 'L/F cruzadas no produjeron conflicto';
    END IF;

    -- El sobre es realmente cifrado: ni principal ni motivo claro aparecen
    -- en las tablas nuevas, aunque estaban presentes en la version efimera.
    SELECT count(*) INTO total
      FROM pg_catalog.pg_attribute AS a
      JOIN pg_catalog.pg_class AS c ON c.oid = a.attrelid
      JOIN pg_catalog.pg_namespace AS n ON n.oid = c.relnamespace
     WHERE n.nspname = 'vec_bolsa_convocatorias'
       AND c.relname IN (
           'diario_borrador_version','material_borrador',
           'borrador_convocatoria_version'
       )
       AND NOT a.attisdropped
       AND a.attname IN (
           'principal','principal_ref','principal_id','motivo','clave_cliente'
       );
    IF total <> 0 OR position(
           convert_to('PRINCIPAL_NO_DEBE_PERSISTIR', 'UTF8')
           IN (SELECT b.sobre_cifrado
                 FROM vec_bolsa_convocatorias.borrador_convocatoria_version AS b
                WHERE b.convocatoria_id = 'convocatoria-prueba')
    ) > 0 THEN
        RAISE EXCEPTION 'se persistio principal, motivo o clave en claro';
    END IF;

    -- Actualizacion real con CAS sobre la revision 1. La referencia del
    -- recurso sigue siendo ID#secuencia; revision y huella viven en el estado.
    huella_estado_anterior := huella_estado;
    version_json := version_json
        || jsonb_build_object(
            'revision', 2, 'ultima_modificacion_en', ahora_texto
        );
    version_json := jsonb_set(
        version_json, '{contenido,titulo}',
        to_jsonb('Bolsa auxiliar actualizada'::text)
    );
    version_canonica := convert_to(version_json::text, 'UTF8');
    huella_estado := encode(sha256(version_canonica), 'hex');
    material_json := jsonb_build_object(
        'esquema', 'bolsa.convocatoria.intencion.v2',
        'accion', 'bolsa.convocatoria.borrador.actualizar',
        'estado_principal_esperado', jsonb_build_object(
            'referencia', 'convocatoria-prueba#1',
            'revision', 1,
            'huella_estado_sha256', huella_estado_anterior
        ),
        'estado_principal_nuevo', jsonb_build_object(
            'referencia', 'convocatoria-prueba#1',
            'revision', 2, 'huella_estado_sha256', huella_estado
        ),
        'dominio_criptografico_motivo',
            'bolsa.convocatoria.motivo.v1',
        'generacion_clave_motivo', 3,
        'huella_motivo_hmac_sha256',
            'hmac-sha256:motivo-gobierno-v3:' || repeat('a', 64)
    );
    material_canonico := convert_to(material_json::text, 'UTF8');
    huella_material := encode(sha256(material_canonico), 'hex');
    contexto_json := jsonb_build_object(
        'ambitos', version_json -> 'ambito_organizativo',
        'atributos', jsonb_build_object(
            'huella_intencion_sha256', huella_material
        )
    );
    contexto_canonico := convert_to(contexto_json::text, 'UTF8');
    huella_contexto := encode(sha256(contexto_canonico), 'hex');
    decision_json := decision_json || jsonb_build_object(
        'decision_ref', 'decision-borrador-prueba-actualizar',
        'accion', 'bolsa.convocatoria.borrador.actualizar',
        'contexto_recurso_huella_sha256', huella_contexto
    );
    decision_canonica := convert_to(decision_json::text, 'UTF8');
    huella_decision := encode(sha256(decision_canonica), 'hex');
    INSERT INTO vec_bolsa_convocatorias.atestacion_autorizacion_version(
        decision_ref, atestacion_ref, version, estado,
        huella_decision_sha256, evidencia_canonica,
        huella_evidencia_sha256, sobre_cose_sign1,
        huella_sobre_sha256, clave_id, revision_confianza,
        verificada_en, valida_desde, valida_hasta, registrada_en
    ) VALUES (
        'decision-borrador-prueba-actualizar',
        'atestacion-pdp-prueba-actualizar', 1, 'activa',
        huella_decision, convert_to('{}', 'UTF8'), huella_atestacion,
        decode(repeat('ad', 16), 'hex'),
        encode(sha256(decode(repeat('ad', 16), 'hex')), 'hex'),
        'clave-pdp-prueba', 'confianza-prueba', ahora,
        ahora - interval '1 minute', ahora + interval '5 minutes', ahora
    );
    INSERT INTO vec_bolsa_convocatorias.atestacion_autorizacion_actual
    VALUES (
        'decision-borrador-prueba-actualizar',
        'atestacion-pdp-prueba-actualizar', 1, 'activa', ahora
    );
    INSERT INTO vec_bolsa_convocatorias.atestacion_pdp_borrador
    VALUES (
        'decision-borrador-prueba-actualizar',
        'atestacion-pdp-prueba-actualizar', 1, 'activa',
        huella_decision, huella_atestacion,
        'verificador-pdp-prueba-actualizar', ahora, ahora
    );
    atestacion_json := jsonb_build_object(
        'decision_ref', 'decision-borrador-prueba-actualizar',
        'atestacion_ref', 'atestacion-pdp-prueba-actualizar',
        'version', 1, 'estado', 'activa',
        'huella_atestacion_sha256', huella_atestacion,
        'verificador_ref', 'verificador-pdp-prueba-actualizar',
        'verificada_en', ahora_texto
    );
    decision_proyeccion := decision_proyeccion || jsonb_build_object(
        'decision_ref', 'decision-borrador-prueba-actualizar',
        'huella_decision_sha256', huella_decision,
        'accion', 'bolsa.convocatoria.borrador.actualizar',
        'contexto_recurso_huella_sha256', huella_contexto,
        'atestacion_pdp', atestacion_json
    );
    identidad := jsonb_build_object(
        'localizador', jsonb_build_object(
            'version_esquema', 1, 'dominio', 'localizador',
            'clave_ref', 'clave:hmac:convocatorias:localizador:actualizar',
            'generacion_clave', 2, 'hmac_sha256', repeat('e', 64)
        ),
        'huella_solicitud', jsonb_build_object(
            'version_esquema', 1, 'dominio', 'huella_solicitud',
            'clave_ref', 'clave:hmac:convocatorias:huella:actualizar',
            'generacion_clave', 2, 'hmac_sha256', repeat('f', 64)
        )
    );
    identidad_actualizacion := identidad;
    reserva := jsonb_build_object(
        'esquema', 'vec.bolsa.convocatoria.reserva-decision.v2',
        'identidad', identidad,
        'identidades_consulta', jsonb_build_array(identidad),
        'accion', 'bolsa.convocatoria.borrador.actualizar',
        'huella_material_sha256', huella_material,
        'recurso_ref', 'convocatoria-prueba#1',
        'contexto_recurso_huella_sha256', huella_contexto,
        'solicitada_en', ahora_texto,
        'arrendamiento_inicia_en', ahora_texto,
        'arrendamiento_vence_en', lease_fin_texto,
        'decision', decision_proyeccion
    );
    PERFORM *
      FROM vec_bolsa_convocatorias.reservar_decision_borrador_interna_v1(
          reserva, material_canonico, version_canonica,
          decision_canonica, contexto_canonico
      );
    sellado := sellado || jsonb_build_object(
        'accion', 'bolsa.convocatoria.borrador.actualizar',
        'atestacion_ref', 'atestacion-motivo-prueba-actualizar',
        'token_consumo_ref', 'consumo-motivo-prueba-actualizar'
    );
    sobre_cifrado := decode(repeat('de', 32), 'hex');
    envoltura := envoltura || jsonb_build_object(
        'nonce_hex', repeat('3a', 12),
        'etiqueta_autenticacion_hex', repeat('4b', 16),
        'atestacion_cifrado_ref', 'atestacion-cifrado-prueba-actualizar',
        'huella_sobre_cifrado_sha256',
            encode(sha256(sobre_cifrado), 'hex')
    );
    proyeccion := proyeccion || jsonb_build_object(
        'revision', 2, 'huella_estado_sha256', huella_estado,
        'titulo', 'Bolsa auxiliar actualizada',
        'actualizada_en', ahora_texto
    );
    confirmacion := jsonb_build_object(
        'esquema', 'vec.bolsa.convocatoria.confirmacion-borrador.v2',
        'identidad', identidad, 'revision', 1, 'cercado', 1,
        'solicitada_en', ahora_texto, 'sellado_motivo', sellado,
        'envoltura_cifrado', envoltura, 'proyeccion_ligera', proyeccion
    );
    SELECT * INTO STRICT resultado
      FROM vec_bolsa_convocatorias.confirmar_borrador_interna_v1(
          confirmacion, material_canonico, version_canonica, sobre_cifrado
      );
    IF resultado.resultado <> 'confirmada'
       OR resultado.estado_principal_revision <> 2
       OR (SELECT revision
             FROM vec_bolsa_convocatorias.borrador_convocatoria_actual
            WHERE convocatoria_id = 'convocatoria-prueba'
              AND secuencia = 1) <> 2
       OR (SELECT count(*) FROM
               vec_bolsa_convocatorias.auditoria_borrador) <> 2
       OR (SELECT count(*) FROM
               vec_bolsa_convocatorias.outbox_borrador) <> 2
       OR (SELECT count(*) FROM
               vec_bolsa_convocatorias.sellado_motivo_borrador) <> 2
       OR (SELECT count(*) FROM
               vec_bolsa_convocatorias.prueba_desenlace_borrador) <> 0 THEN
        RAISE EXCEPTION 'actualizacion CAS no fue atomica: %', resultado;
    END IF;

    -- Dos generaciones que ya apuntan a operaciones diferentes no pueden
    -- resolverse escogiendo silenciosamente la primera.
    BEGIN
        PERFORM *
          FROM vec_bolsa_convocatorias.consultar_identidades_borrador_interna_v1(
              jsonb_build_array(
                  identidad_actualizacion, identidad_alias_g1
              )
          );
        RAISE EXCEPTION 'consulta multigeneracion ambigua fue aceptada';
    EXCEPTION WHEN cardinality_violation THEN
        NULL;
    END;
    BEGIN
        PERFORM *
          FROM vec_bolsa_convocatorias.reservar_decision_borrador_interna_v1(
              reserva || jsonb_build_object(
                  'identidades_consulta', jsonb_build_array(
                      identidad_actualizacion, identidad_alias_g1
                  )
              ), material_canonico, version_canonica,
              decision_canonica, contexto_canonico
          );
        RAISE EXCEPTION 'reserva multigeneracion ambigua fue aceptada';
    EXCEPTION WHEN cardinality_violation THEN
        NULL;
    END;

    -- Otra operacion autorizada contra el estado 1 conserva un veredicto
    -- durable no_aplicado y no duplica agregado, auditoria ni outbox.
    decision_json := decision_json || jsonb_build_object(
        'decision_ref', 'decision-borrador-prueba-cas-obsoleto'
    );
    decision_canonica := convert_to(decision_json::text, 'UTF8');
    huella_decision := encode(sha256(decision_canonica), 'hex');
    INSERT INTO vec_bolsa_convocatorias.atestacion_autorizacion_version(
        decision_ref, atestacion_ref, version, estado,
        huella_decision_sha256, evidencia_canonica,
        huella_evidencia_sha256, sobre_cose_sign1,
        huella_sobre_sha256, clave_id, revision_confianza,
        verificada_en, valida_desde, valida_hasta, registrada_en
    ) VALUES (
        'decision-borrador-prueba-cas-obsoleto',
        'atestacion-pdp-prueba-cas-obsoleto', 1, 'activa',
        huella_decision, convert_to('{}', 'UTF8'), huella_atestacion,
        decode(repeat('ae', 16), 'hex'),
        encode(sha256(decode(repeat('ae', 16), 'hex')), 'hex'),
        'clave-pdp-prueba', 'confianza-prueba', ahora,
        ahora - interval '1 minute', ahora + interval '5 minutes', ahora
    );
    INSERT INTO vec_bolsa_convocatorias.atestacion_autorizacion_actual
    VALUES (
        'decision-borrador-prueba-cas-obsoleto',
        'atestacion-pdp-prueba-cas-obsoleto', 1, 'activa', ahora
    );
    INSERT INTO vec_bolsa_convocatorias.atestacion_pdp_borrador
    VALUES (
        'decision-borrador-prueba-cas-obsoleto',
        'atestacion-pdp-prueba-cas-obsoleto', 1, 'activa',
        huella_decision, huella_atestacion,
        'verificador-pdp-prueba-cas-obsoleto', ahora, ahora
    );
    atestacion_json := atestacion_json || jsonb_build_object(
        'decision_ref', 'decision-borrador-prueba-cas-obsoleto',
        'atestacion_ref', 'atestacion-pdp-prueba-cas-obsoleto',
        'verificador_ref', 'verificador-pdp-prueba-cas-obsoleto'
    );
    decision_proyeccion := decision_proyeccion || jsonb_build_object(
        'decision_ref', 'decision-borrador-prueba-cas-obsoleto',
        'huella_decision_sha256', huella_decision,
        'atestacion_pdp', atestacion_json
    );
    identidad := jsonb_build_object(
        'localizador', jsonb_build_object(
            'version_esquema', 1, 'dominio', 'localizador',
            'clave_ref', 'clave:hmac:convocatorias:localizador:cas',
            'generacion_clave', 1, 'hmac_sha256', repeat('0', 64)
        ),
        'huella_solicitud', jsonb_build_object(
            'version_esquema', 1, 'dominio', 'huella_solicitud',
            'clave_ref', 'clave:hmac:convocatorias:huella:cas',
            'generacion_clave', 1, 'hmac_sha256', repeat('1', 64)
        )
    );
    reserva := reserva || jsonb_build_object(
        'identidad', identidad,
        'identidades_consulta', jsonb_build_array(identidad),
        'decision', decision_proyeccion
    );
    PERFORM *
      FROM vec_bolsa_convocatorias.reservar_decision_borrador_interna_v1(
          reserva, material_canonico, version_canonica,
          decision_canonica, contexto_canonico
      );
    sellado := sellado || jsonb_build_object(
        'atestacion_ref', 'atestacion-motivo-prueba-cas-obsoleto',
        'token_consumo_ref', 'consumo-motivo-prueba-cas-obsoleto'
    );
    confirmacion := confirmacion || jsonb_build_object(
        'identidad', identidad, 'sellado_motivo', sellado
    );
    SELECT * INTO STRICT resultado
      FROM vec_bolsa_convocatorias.confirmar_borrador_interna_v1(
          confirmacion, material_canonico, version_canonica, sobre_cifrado
      );
    IF resultado.resultado <> 'conflicto_cas'
       OR resultado.estado_diario <> 'no_aplicado'
       OR (SELECT count(*) FROM
               vec_bolsa_convocatorias.borrador_convocatoria_version) <> 2
       OR (SELECT count(*) FROM
               vec_bolsa_convocatorias.auditoria_borrador) <> 2
       OR (SELECT count(*) FROM
               vec_bolsa_convocatorias.outbox_borrador) <> 2
       OR (SELECT count(*) FROM
               vec_bolsa_convocatorias.sellado_motivo_borrador) <> 2
       OR (SELECT count(*) FROM
               vec_bolsa_convocatorias.prueba_desenlace_borrador) <> 1 THEN
        RAISE EXCEPTION 'CAS obsoleto tuvo efectos parciales: %', resultado;
    END IF;

    -- Listado ligero: filtro de texto/categoria, maximo 50, sin leer ni
    -- devolver el sobre. La decision local se consume y audita una sola vez.
    INSERT INTO vec_bolsa_convocatorias.atestacion_autorizacion_version(
        decision_ref, atestacion_ref, version, estado,
        huella_decision_sha256, evidencia_canonica,
        huella_evidencia_sha256, sobre_cose_sign1,
        huella_sobre_sha256, clave_id, revision_confianza,
        verificada_en, valida_desde, valida_hasta, registrada_en
    ) VALUES (
        'decision-listado-prueba', 'atestacion-listado-prueba', 1,
        'activa', repeat('9', 64), convert_to('{}', 'UTF8'),
        huella_atestacion, decode(repeat('bc', 16), 'hex'),
        encode(sha256(decode(repeat('bc', 16), 'hex')), 'hex'),
        'clave-pdp-listado', 'confianza-prueba', ahora,
        ahora - interval '1 minute', ahora + interval '5 minutes', ahora
    );
    INSERT INTO vec_bolsa_convocatorias.atestacion_autorizacion_actual
    VALUES (
        'decision-listado-prueba', 'atestacion-listado-prueba',
        1, 'activa', ahora
    );
    INSERT INTO vec_bolsa_convocatorias.atestacion_pdp_borrador
    VALUES (
        'decision-listado-prueba', 'atestacion-listado-prueba', 1,
        'activa', repeat('9', 64), huella_atestacion,
        'verificador-listado-prueba', ahora, ahora
    );
    lectura := jsonb_build_object(
        'decision_ref', 'decision-listado-prueba',
        'huella_decision_sha256', repeat('9', 64),
        'atestacion_ref', 'atestacion-listado-prueba',
        'atestacion_version', 1, 'estado_atestacion', 'activa',
        'huella_atestacion_sha256', huella_atestacion,
        'accion', 'bolsa.convocatoria.borrador.listar',
        'recurso_ref', 'borradores:org_0123456789abcdef',
        'organizacion_ref', 'org_0123456789abcdef',
        'unidad_gestion_ref', 'uni_0123456789abcdef'
    );
    lista := vec_bolsa_convocatorias.listar_borradores_interna_v1(
        jsonb_build_object(
            'limite', 50, 'cursor', '', 'texto', 'auxiliar',
            'categoria', 'auxiliar'
        ), lectura
    );
    IF lista ->> 'esquema' <> 'vec.bolsa.borradores.lista.v1'
       OR jsonb_array_length(lista -> 'elementos') <> 1
       OR lista #>> '{elementos,0,referencia_estado,referencia}' <>
          'convocatoria-prueba#1'
       OR lista::text LIKE '%sobre_cifrado%'
       OR (SELECT count(*) FROM
               vec_bolsa_convocatorias.uso_decision_lectura_borrador) <> 1
       OR (SELECT count(*) FROM
               vec_bolsa_convocatorias.auditoria_lectura_borrador) <> 1 THEN
        RAISE EXCEPTION 'listado ligero/auditado incorrecto: %', lista;
    END IF;

    INSERT INTO vec_bolsa_convocatorias.atestacion_autorizacion_version(
        decision_ref, atestacion_ref, version, estado,
        huella_decision_sha256, evidencia_canonica,
        huella_evidencia_sha256, sobre_cose_sign1,
        huella_sobre_sha256, clave_id, revision_confianza,
        verificada_en, valida_desde, valida_hasta, registrada_en
    ) VALUES (
        'decision-detalle-prueba', 'atestacion-detalle-prueba', 1,
        'activa', repeat('b', 64), convert_to('{}', 'UTF8'),
        huella_atestacion, decode(repeat('bd', 16), 'hex'),
        encode(sha256(decode(repeat('bd', 16), 'hex')), 'hex'),
        'clave-pdp-detalle', 'confianza-prueba', ahora,
        ahora - interval '1 minute', ahora + interval '5 minutes', ahora
    );
    INSERT INTO vec_bolsa_convocatorias.atestacion_autorizacion_actual
    VALUES (
        'decision-detalle-prueba', 'atestacion-detalle-prueba',
        1, 'activa', ahora
    );
    INSERT INTO vec_bolsa_convocatorias.atestacion_pdp_borrador
    VALUES (
        'decision-detalle-prueba', 'atestacion-detalle-prueba', 1,
        'activa', repeat('b', 64), huella_atestacion,
        'verificador-detalle-prueba', ahora, ahora
    );
    lectura := jsonb_build_object(
        'decision_ref', 'decision-detalle-prueba',
        'huella_decision_sha256', repeat('b', 64),
        'atestacion_ref', 'atestacion-detalle-prueba',
        'atestacion_version', 1, 'estado_atestacion', 'activa',
        'huella_atestacion_sha256', huella_atestacion,
        'accion', 'bolsa.convocatoria.borrador.consultar',
        'recurso_ref', 'convocatoria-prueba#1',
        'organizacion_ref', 'org_0123456789abcdef',
        'unidad_gestion_ref', 'uni_0123456789abcdef'
    );
    SELECT * INTO STRICT detalle
      FROM vec_bolsa_convocatorias.obtener_borrador_interna_v1(
          'convocatoria-prueba#1', lectura
      );
    IF detalle.metadatos #>> '{referencia_estado,referencia}' <>
          'convocatoria-prueba#1'
       OR detalle.sobre_cifrado <> sobre_cifrado
       OR detalle.huella_sobre_cifrado_sha256 <>
          encode(sha256(sobre_cifrado), 'hex')
       OR (SELECT count(*) FROM
               vec_bolsa_convocatorias.uso_decision_lectura_borrador) <> 2
       OR (SELECT count(*) FROM
               vec_bolsa_convocatorias.auditoria_lectura_borrador) <> 2 THEN
        RAISE EXCEPTION 'detalle cifrado/auditado incorrecto: %', detalle;
    END IF;
    BEGIN
        PERFORM *
          FROM vec_bolsa_convocatorias.obtener_borrador_interna_v1(
              'convocatoria-prueba#1',
              jsonb_set(
                  lectura, '{organizacion_ref}',
                  '"org_ffffffffffffffff"'::jsonb
              )
          );
        RAISE EXCEPTION 'detalle cruzo el ambito organizativo';
    EXCEPTION WHEN no_data_found THEN
        NULL;
    END;

    -- La historia no es mutable ni siquiera para el propietario NOLOGIN.
    BEGIN
        UPDATE vec_bolsa_convocatorias.diario_borrador_version
           SET estado = 'no_aplicado';
        RAISE EXCEPTION 'historia de diario mutable';
    EXCEPTION WHEN object_not_in_prerequisite_state THEN
        NULL;
    END;
END
$pruebas$;

ROLLBACK;

-- Fixture exclusivamente para la carrera entre dos conexiones reales. La
-- elimina el script antes del down; no forma parte del esquema productivo.
CREATE TABLE public.fixture_reserva_borrador_concurrente (
    reserva jsonb NOT NULL,
    reserva_solapada jsonb NOT NULL,
    material bytea NOT NULL,
    version_canonica bytea NOT NULL,
    decision_canonica bytea NOT NULL,
    decision_canonica_solapada bytea NOT NULL,
    contexto bytea NOT NULL
);
GRANT SELECT, INSERT, UPDATE ON public.fixture_reserva_borrador_concurrente
    TO vec_bolsa_convocatorias_propietario;
GRANT USAGE ON SCHEMA public TO vec_bolsa_convocatorias_propietario;

BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL ROLE vec_bolsa_convocatorias_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
DO $fixture_concurrente$
DECLARE
    ahora timestamptz := date_trunc('microseconds', clock_timestamp());
    ahora_texto text := to_char(ahora AT TIME ZONE 'UTC',
        'YYYY-MM-DD"T"HH24:MI:SS.US"Z"');
    emitida_texto text := to_char(
        (ahora - interval '1 second') AT TIME ZONE 'UTC',
        'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
    );
    vence_texto text := to_char(
        (ahora + interval '4 minutes') AT TIME ZONE 'UTC',
        'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
    );
    version_json jsonb;
    version_bytes bytea;
    estado_huella text;
    material_json jsonb;
    material_bytes bytea;
    material_huella text;
    contexto_json jsonb;
    contexto_bytes bytea;
    contexto_huella text;
    decision_json jsonb;
    decision_bytes bytea;
    decision_huella text;
	decision_json_solapada jsonb;
	decision_bytes_solapada bytea;
	decision_huella_solapada text;
    evidencia_huella text := encode(
        sha256(convert_to('{}', 'UTF8')), 'hex'
    );
    atestacion jsonb;
	atestacion_solapada jsonb;
    proyeccion_decision jsonb;
	proyeccion_decision_solapada jsonb;
    identidad jsonb;
    identidad_intermedia jsonb;
    identidad_antigua jsonb;
BEGIN
    version_json := jsonb_build_object(
        'id', 'convocatoria-concurrente', 'secuencia', 1,
        'revision', 1, 'estado_gobierno', 'borrador',
        'ambito_organizativo', jsonb_build_object(
            'organizacion_ref', 'org_0123456789abcdef'
        ),
        'contenido', jsonb_build_object('sintetico', true)
    );
    version_bytes := convert_to(version_json::text, 'UTF8');
    estado_huella := encode(sha256(version_bytes), 'hex');
    material_json := jsonb_build_object(
        'esquema', 'bolsa.convocatoria.intencion.v2',
        'accion', 'bolsa.convocatoria.borrador.crear',
        'estado_principal_nuevo', jsonb_build_object(
            'referencia', 'convocatoria-concurrente#1',
            'revision', 1, 'huella_estado_sha256', estado_huella
        ),
        'dominio_criptografico_motivo',
            'bolsa.convocatoria.motivo.v1',
        'generacion_clave_motivo', 1,
        'huella_motivo_hmac_sha256',
            'hmac-sha256:motivo-concurrente:' || repeat('c', 64)
    );
    material_bytes := convert_to(material_json::text, 'UTF8');
    material_huella := encode(sha256(material_bytes), 'hex');
    contexto_json := jsonb_build_object(
        'ambitos', version_json -> 'ambito_organizativo',
        'atributos', jsonb_build_object(
            'huella_intencion_sha256', material_huella
        )
    );
    contexto_bytes := convert_to(contexto_json::text, 'UTF8');
    contexto_huella := encode(sha256(contexto_bytes), 'hex');
    decision_json := jsonb_build_object(
        'decision_ref', 'decision-concurrente',
        'accion', 'bolsa.convocatoria.borrador.crear',
        'recurso_ref', 'convocatoria-concurrente#1',
        'modulo_id', 'bolsa',
        'tipo_recurso', 'version_convocatoria_gobernada',
        'contexto_recurso_huella_sha256', contexto_huella,
        'finalidad', 'gobierno_convocatorias',
        'asignacion_ref', 'asignacion-rrhh-concurrente',
        'asignacion_huella_sha256', repeat('a', 64),
        'version_rol_ref', 'rol-rrhh-v1',
        'version_rol_huella_sha256', repeat('d', 64),
        'control_vigencia_version_rol_ref', 'rol-rrhh-v1',
        'control_vigencia_version_rol_revision', 1,
        'control_vigencia_version_rol_huella_sha256', repeat('b', 64),
        'revision_catalogo_politicas', 1,
        'catalogo_politicas_huella_sha256', repeat('e', 64),
        'campos_permitidos', jsonb_build_array(
            'auditoria','evento_outbox','version_convocatoria'
        ),
        'obligaciones', '[]'::jsonb, 'garantia_minima', 'alto',
        'emitida_en', emitida_texto, 'valida_hasta', vence_texto
    );
    decision_bytes := convert_to(decision_json::text, 'UTF8');
    decision_huella := encode(sha256(decision_bytes), 'hex');
	decision_json_solapada := decision_json || jsonb_build_object(
		'decision_ref', 'decision-concurrente-solapada'
	);
	decision_bytes_solapada := convert_to(
		decision_json_solapada::text, 'UTF8'
	);
	decision_huella_solapada := encode(
		sha256(decision_bytes_solapada), 'hex'
	);
    INSERT INTO vec_bolsa_convocatorias.atestacion_autorizacion_version(
        decision_ref, atestacion_ref, version, estado,
        huella_decision_sha256, evidencia_canonica,
        huella_evidencia_sha256, sobre_cose_sign1,
        huella_sobre_sha256, clave_id, revision_confianza,
        verificada_en, valida_desde, valida_hasta, registrada_en
    ) VALUES (
        'decision-concurrente', 'atestacion-concurrente', 1, 'activa',
        decision_huella, convert_to('{}', 'UTF8'), evidencia_huella,
        decode(repeat('ce', 16), 'hex'),
        encode(sha256(decode(repeat('ce', 16), 'hex')), 'hex'),
        'clave-concurrente', 'confianza-concurrente', ahora,
        ahora - interval '1 minute', ahora + interval '5 minutes', ahora
    );
    INSERT INTO vec_bolsa_convocatorias.atestacion_autorizacion_actual
    VALUES (
        'decision-concurrente', 'atestacion-concurrente', 1,
        'activa', ahora
    );
    INSERT INTO vec_bolsa_convocatorias.atestacion_pdp_borrador
    VALUES (
        'decision-concurrente', 'atestacion-concurrente', 1, 'activa',
        decision_huella, evidencia_huella, 'verificador-concurrente',
        ahora, ahora
    );
	INSERT INTO vec_bolsa_convocatorias.atestacion_autorizacion_version(
		decision_ref, atestacion_ref, version, estado,
		huella_decision_sha256, evidencia_canonica,
		huella_evidencia_sha256, sobre_cose_sign1,
		huella_sobre_sha256, clave_id, revision_confianza,
		verificada_en, valida_desde, valida_hasta, registrada_en
	) VALUES (
		'decision-concurrente-solapada', 'atestacion-concurrente-solapada',
		1, 'activa', decision_huella_solapada,
		convert_to('{}', 'UTF8'), evidencia_huella,
		decode(repeat('cf', 16), 'hex'),
		encode(sha256(decode(repeat('cf', 16), 'hex')), 'hex'),
		'clave-concurrente-solapada', 'confianza-concurrente-solapada',
		ahora, ahora - interval '1 minute', ahora + interval '5 minutes', ahora
	);
	INSERT INTO vec_bolsa_convocatorias.atestacion_autorizacion_actual
	VALUES (
		'decision-concurrente-solapada', 'atestacion-concurrente-solapada',
		1, 'activa', ahora
	);
	INSERT INTO vec_bolsa_convocatorias.atestacion_pdp_borrador
	VALUES (
		'decision-concurrente-solapada', 'atestacion-concurrente-solapada',
		1, 'activa', decision_huella_solapada, evidencia_huella,
		'verificador-concurrente-solapado', ahora, ahora
	);
    atestacion := jsonb_build_object(
        'decision_ref', 'decision-concurrente',
        'atestacion_ref', 'atestacion-concurrente', 'version', 1,
        'estado', 'activa', 'huella_atestacion_sha256', evidencia_huella,
        'verificador_ref', 'verificador-concurrente',
        'verificada_en', ahora_texto
    );
	atestacion_solapada := jsonb_build_object(
		'decision_ref', 'decision-concurrente-solapada',
		'atestacion_ref', 'atestacion-concurrente-solapada', 'version', 1,
		'estado', 'activa', 'huella_atestacion_sha256', evidencia_huella,
		'verificador_ref', 'verificador-concurrente-solapado',
		'verificada_en', ahora_texto
	);
    proyeccion_decision := jsonb_build_object(
        'esquema_huella',
            'vec.autorizacion.decision.reforzada.v1.autenticacion-actor',
        'decision_ref', 'decision-concurrente',
        'huella_decision_sha256', decision_huella,
        'accion', 'bolsa.convocatoria.borrador.crear',
        'recurso_ref', 'convocatoria-concurrente#1',
        'modulo_id', 'bolsa',
        'tipo_recurso', 'version_convocatoria_gobernada',
        'contexto_recurso_huella_sha256', contexto_huella,
        'finalidad', 'gobierno_convocatorias',
        'asignacion_ref', 'asignacion-rrhh-concurrente',
        'asignacion_huella_sha256', repeat('a', 64),
        'version_rol_ref', 'rol-rrhh-v1',
        'version_rol_huella_sha256', repeat('d', 64),
        'control_vigencia_version_rol_ref', 'rol-rrhh-v1',
        'control_vigencia_version_rol_revision', 1,
        'control_vigencia_version_rol_huella_sha256', repeat('b', 64),
        'revision_catalogo_politicas', 1,
        'catalogo_politicas_huella_sha256', repeat('e', 64),
        'emitida_en', emitida_texto,
        'verificada_en', ahora_texto, 'valida_hasta', vence_texto,
        'atestacion_pdp', atestacion
    );
	proyeccion_decision_solapada := proyeccion_decision || jsonb_build_object(
		'decision_ref', 'decision-concurrente-solapada',
		'huella_decision_sha256', decision_huella_solapada,
		'atestacion_pdp', atestacion_solapada
	);
    identidad := jsonb_build_object(
        'localizador', jsonb_build_object(
            'version_esquema', 1, 'dominio', 'localizador',
            'clave_ref', 'clave:hmac:convocatorias:localizador:concurrente-g3',
            'generacion_clave', 3, 'hmac_sha256', repeat('1', 64)
        ),
        'huella_solicitud', jsonb_build_object(
            'version_esquema', 1, 'dominio', 'huella_solicitud',
            'clave_ref', 'clave:hmac:convocatorias:huella:concurrente-g3',
            'generacion_clave', 3, 'hmac_sha256', repeat('2', 64)
        )
    );
    identidad_intermedia := jsonb_build_object(
        'localizador', jsonb_build_object(
            'version_esquema', 1, 'dominio', 'localizador',
            'clave_ref', 'clave:hmac:convocatorias:localizador:concurrente-g2',
            'generacion_clave', 2, 'hmac_sha256', repeat('3', 64)
        ),
        'huella_solicitud', jsonb_build_object(
            'version_esquema', 1, 'dominio', 'huella_solicitud',
            'clave_ref', 'clave:hmac:convocatorias:huella:concurrente-g2',
            'generacion_clave', 2, 'hmac_sha256', repeat('4', 64)
        )
    );
    identidad_antigua := jsonb_build_object(
        'localizador', jsonb_build_object(
            'version_esquema', 1, 'dominio', 'localizador',
            'clave_ref', 'clave:hmac:convocatorias:localizador:concurrente-g1',
            'generacion_clave', 1, 'hmac_sha256', repeat('5', 64)
        ),
        'huella_solicitud', jsonb_build_object(
            'version_esquema', 1, 'dominio', 'huella_solicitud',
            'clave_ref', 'clave:hmac:convocatorias:huella:concurrente-g1',
            'generacion_clave', 1, 'hmac_sha256', repeat('6', 64)
        )
    );
    INSERT INTO public.fixture_reserva_borrador_concurrente VALUES (
        jsonb_build_object(
            'esquema', 'vec.bolsa.convocatoria.reserva-decision.v2',
            'identidad', identidad,
            'identidades_consulta', jsonb_build_array(
                identidad, identidad_intermedia
            ),
            'accion', 'bolsa.convocatoria.borrador.crear',
            'huella_material_sha256', material_huella,
            'recurso_ref', 'convocatoria-concurrente#1',
            'contexto_recurso_huella_sha256', contexto_huella,
            'solicitada_en', ahora_texto,
            'arrendamiento_inicia_en', ahora_texto,
            'arrendamiento_vence_en', to_char(
                (ahora + interval '8 seconds') AT TIME ZONE 'UTC',
                'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
            ),
            'decision', proyeccion_decision
        ),
        jsonb_build_object(
            'esquema', 'vec.bolsa.convocatoria.reserva-decision.v2',
            'identidad', identidad_intermedia,
            'identidades_consulta', jsonb_build_array(
                identidad_intermedia, identidad_antigua
            ),
            'accion', 'bolsa.convocatoria.borrador.crear',
            'huella_material_sha256', material_huella,
            'recurso_ref', 'convocatoria-concurrente#1',
            'contexto_recurso_huella_sha256', contexto_huella,
            'solicitada_en', to_char(
				(ahora + interval '1 microsecond') AT TIME ZONE 'UTC',
				'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
			),
			'arrendamiento_inicia_en', to_char(
				(ahora + interval '1 microsecond') AT TIME ZONE 'UTC',
				'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
			),
            'arrendamiento_vence_en', to_char(
				(ahora + interval '7 seconds') AT TIME ZONE 'UTC',
                'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
            ),
			'decision', proyeccion_decision_solapada
		), material_bytes, version_bytes, decision_bytes,
		decision_bytes_solapada, contexto_bytes
    );
END
$fixture_concurrente$;
COMMIT;
