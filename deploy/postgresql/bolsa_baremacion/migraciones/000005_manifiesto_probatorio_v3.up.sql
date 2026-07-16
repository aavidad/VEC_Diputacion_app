-- Archivo probatorio V3 para las decisiones tecnicas de baremacion.
--
-- Este corte NO implementa DEC-045 ni sustituye el verificador HMAC/KMS. La
-- aplicacion debe verificar fuera de la transaccion los sellos de todo el
-- snapshot devuelto por la reserva y el sello del manifiesto nuevo. Al volver
-- para confirmar, OCC y este archivo append-only revalidan que el historial no
-- ha cambiado. Nunca se mantiene un bloqueo PostgreSQL durante E/S con KMS.
BEGIN;
SET LOCAL ROLE vec_bolsa_baremacion_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_bolsa_baremacion:migracion_up:manifiesto_probatorio_v3', 0
    )
);

-- No es una migracion en caliente: las funciones V1 no conocen esta barrera.
-- La red o el pool deben estar detenidos antes de emitir el opt-in. El chequeo
-- de sesiones y los locks son defensa adicional, no sustituyen el aislamiento.
DO $mantenimiento_explicito$
DECLARE
    ejecutor_oid oid;
BEGIN
    IF current_setting(
           'vec.confirmar_mantenimiento_bolsa_baremacion_v3', true
       ) IS DISTINCT FROM
          'INSTALAR_MIGRACION_BOLSA_BAREMACION_V3_SIN_TRAFICO' THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'migracion V3 rechazada: falta ventana de mantenimiento';
    END IF;
    SELECT oid INTO STRICT ejecutor_oid
      FROM pg_catalog.pg_roles
     WHERE rolname = 'vec_bolsa_baremacion_ejecutor';
    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_stat_activity AS actividad
         WHERE actividad.datid = (
                   SELECT oid FROM pg_catalog.pg_database
                    WHERE datname = current_database()
               )
           AND actividad.pid <> pg_backend_pid()
           AND actividad.usesysid IS NOT NULL
           AND pg_catalog.pg_has_role(
                   actividad.usesysid, ejecutor_oid, 'MEMBER'
               )
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'migracion V3 rechazada: existe trafico ejecutor activo';
    END IF;
END
$mantenimiento_explicito$;

LOCK TABLE vec_bolsa_baremacion.version_baremacion,
    vec_bolsa_baremacion.baremacion_actual,
    vec_bolsa_baremacion.reserva_version,
    vec_bolsa_baremacion.reserva_actual,
    vec_bolsa_baremacion.token_reserva,
    vec_bolsa_baremacion.uso_decision,
    vec_bolsa_baremacion.auditoria,
    vec_bolsa_baremacion.evento_outbox
    IN ACCESS EXCLUSIVE MODE;

-- Una instalacion antigua con decisiones ya confirmadas solo conserva la
-- referencia y la huella del manifiesto. No se inventan autorizaciones,
-- evidencias ni preimagenes ausentes: la migracion se cierra antes de crear
-- ningun objeto V3.
DO $prevalidar_historia_heredada$
BEGIN
    IF EXISTS (
        SELECT 1
          FROM vec_bolsa_baremacion.version_baremacion
         WHERE numero > 1
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'migracion V3 rechazada: existe historia no reconstruible',
            DETAIL = 'hay versiones tecnicas anteriores sin archivo byte-exacto',
            HINT = 'importe y verifique los manifiestos por un procedimiento extraordinario antes de reintentar';
    END IF;
END
$prevalidar_historia_heredada$;

ALTER TABLE vec_bolsa_baremacion.uso_decision
    DROP CONSTRAINT uso_perfil_cerrado;
ALTER TABLE vec_bolsa_baremacion.uso_decision
    ADD CONSTRAINT uso_perfil_cerrado CHECK (
        esquema_huella_decision =
            'vec.autorizacion.decision.reforzada.v1.autenticacion-actor'
        AND tipo_efecto IN (
            'reserva', 'confirmacion', 'abandono',
            'lectura_vigente', 'lectura_version', 'lectura_evidencia',
            'prevalidacion_archivo'
        )
        AND vec_bolsa_baremacion.huella_sha256_valida(
            huella_decision_sha256
        )
        AND vec_bolsa_baremacion.huella_sha256_valida(huella_efecto_sha256)
        AND vec_bolsa_baremacion.texto_opaco_valido(resultado_ref, 512)
    );

CREATE FUNCTION vec_bolsa_baremacion.parte_canonica_manifiesto_v3(
    p_valor text
)
RETURNS bytea
LANGUAGE sql
IMMUTABLE
STRICT
SET search_path = pg_catalog, pg_temp
AS $funcion$
    SELECT int8send(octet_length(convert_to(p_valor, 'UTF8'))::bigint)
        || convert_to(p_valor, 'UTF8')
$funcion$;

CREATE FUNCTION vec_bolsa_baremacion.accion_recurso_manifiesto_v3_valida(
    p_accion text,
    p_clase text
)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
STRICT
SET search_path = pg_catalog, pg_temp
AS $funcion$
    SELECT EXISTS (
        SELECT 1
          FROM (VALUES
            ('bolsa.baremacion.alta.reservar', 'baremacion'),
            ('bolsa.baremacion.alta.confirmar', 'baremacion'),
            ('bolsa.baremacion.alta.abandonar', 'baremacion'),
            ('bolsa.baremacion.decision.reservar', 'baremacion'),
            ('bolsa.baremacion.archivo.prevalidar', 'baremacion'),
            ('bolsa.baremacion.decision.confirmar', 'baremacion'),
            ('bolsa.baremacion.decision.inicial.adoptar', 'baremacion'),
            ('bolsa.baremacion.decision.rectificar', 'baremacion'),
            ('bolsa.baremacion.decision.revocar', 'baremacion'),
            ('bolsa.baremacion.decision.rehabilitar', 'baremacion'),
            ('bolsa.baremacion.decision.abandonar', 'baremacion'),
            ('bolsa.baremacion.vigente.consultar', 'baremacion'),
            ('bolsa.baremacion.version.consultar', 'baremacion'),
            ('bolsa.criterio.consultar', 'proceso'),
            ('bolsa.evidencia.consultar', 'evidencia'),
            ('bolsa.representacion.consultar', 'representacion'),
            ('bolsa.puntuacion.calcular', 'baremacion'),
            ('bolsa.puntuacion.calculo.recuperar', 'calculo'),
            ('bolsa.firma.politica.consultar', 'politica_firma'),
            ('bolsa.decision.codificar', 'decision'),
            ('bolsa.decision.custodiar', 'decision'),
            ('bolsa.decision.firma.preparar', 'decision'),
            ('bolsa.decision.firma.consultar', 'sesion_firma'),
            ('bolsa.decision.firma.validar', 'artefacto_firma'),
            ('bolsa.decision.firma.sellar_tiempo', 'artefacto_firma'),
            ('bolsa.decision.firma.aumentar', 'artefacto_firma'),
            ('bolsa.decision.firma.binario.recuperar', 'documento_firmado'),
            ('bolsa.decision.firma.documento.custodiar', 'documento_firmado'),
            ('bolsa.decision.firma.documento.retener', 'documento_firmado'),
            ('bolsa.decision.firma.artefacto.recuperar', 'artefacto_firma'),
            ('bolsa.decision.firma.validacion.recuperar', 'validacion_firma'),
            ('bolsa.decision.firma.sello_tiempo.recuperar', 'sello_tiempo'),
            ('bolsa.decision.firma.aumento.recuperar', 'aumento_firma'),
            ('bolsa.baremacion.transaccion.consultar', 'transaccion')
          ) AS permitida(accion, clase)
         WHERE permitida.accion = p_accion
           AND permitida.clase = p_clase
    )
$funcion$;

CREATE FUNCTION vec_bolsa_baremacion.tipo_evidencia_manifiesto_v3_valido(
    p_tipo text
)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
STRICT
SET search_path = pg_catalog, pg_temp
AS $funcion$
    SELECT p_tipo = ANY (ARRAY[
        'estado_base', 'calculo_oficial', 'criterio_publicado',
        'documento_merito', 'representacion_documento',
        'contenido_decision', 'politica_firma', 'documento_canonico',
        'custodia_firmable', 'preparacion_firma', 'consulta_firma',
        'validacion_firma_inicial', 'sello_tiempo',
        'vinculo_revision_sellada', 'validacion_documento_sellado',
        'aumento_longevidad', 'vinculo_revision_longeva',
        'validacion_firma_final', 'recuperacion_documento_firmado',
        'custodia_documento_firmado', 'retencion_documento_firmado'
    ]::text[])
$funcion$;

-- Perfil nominal compartido con Go: ASCII visible, limite en bytes y sin '*',
-- que se reserva como comodin y nunca identifica una evidencia concreta.
CREATE FUNCTION vec_bolsa_baremacion.referencia_manifiesto_v3_valida(
    p_valor text,
    p_maximo integer
)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
STRICT
SET search_path = pg_catalog, pg_temp
AS $funcion$
    WITH codificado AS MATERIALIZED (
        -- Una sola conversion hace que el perfil sea independiente del
        -- server_encoding y evita recalcular UTF-8 para cada posicion.
        SELECT convert_to(p_valor, 'UTF8') AS bytes
    )
    SELECT p_maximo > 0
       AND octet_length(codificado.bytes) BETWEEN 1 AND p_maximo
       AND NOT EXISTS (
           SELECT 1
             FROM generate_series(
                 0, octet_length(codificado.bytes) - 1
             ) AS posicion(indice)
            WHERE get_byte(codificado.bytes, posicion.indice)
                  NOT BETWEEN 33 AND 126
               OR get_byte(codificado.bytes, posicion.indice) = 42
       )
      FROM codificado
$funcion$;

-- El dominio del sello HMAC tiene el mismo limite de 128 bytes que claveValida
-- en Go. La expresion cerrada garantiza exactamente tres componentes.
CREATE FUNCTION vec_bolsa_baremacion.sello_hmac_manifiesto_v3_valido(
    p_valor text
)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
STRICT
SET search_path = pg_catalog, pg_temp
AS $funcion$
    SELECT p_valor ~ '^hmac-sha256:[a-z0-9][a-z0-9._-]*:[0-9a-f]{64}$'
       AND octet_length(split_part(p_valor, ':', 2)) BETWEEN 1 AND 128
$funcion$;

-- Mismo perfil textual que time.RFC3339Nano canonico de Go: UTC literal,
-- hora civil cerrada y fraccion sin ceros finales. El cast valida calendario.
CREATE FUNCTION vec_bolsa_baremacion.instante_canonico_manifiesto_v3(
    p_valor text
)
RETURNS boolean
LANGUAGE plpgsql
IMMUTABLE
STRICT
SET search_path = pg_catalog, pg_temp
AS $funcion$
BEGIN
    IF p_valor = '0001-01-01T00:00:00Z'
       OR p_valor !~
       '^[0-9]{4}-[0-9]{2}-[0-9]{2}T([01][0-9]|2[0-3]):[0-5][0-9]:[0-5][0-9]([.][0-9]{0,8}[1-9])?Z$' THEN
        RETURN false;
    END IF;
    PERFORM p_valor::timestamptz;
    RETURN true;
EXCEPTION WHEN datetime_field_overflow OR invalid_datetime_format THEN
    RETURN false;
END
$funcion$;

-- Reconstruye exactamente materialCanonico(false) del puerto Go. Todos los
-- escalares son texto UTF-8 precedido por uint64 big-endian. Los conteos y las
-- secuencias tambien son texto decimal, como strconv.Itoa/FormatUint.
CREATE FUNCTION vec_bolsa_baremacion.contenido_manifiesto_probatorio_v3(
    p_manifiesto jsonb
)
RETURNS bytea
LANGUAGE plpgsql
IMMUTABLE
STRICT
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    contenido bytea;
    autorizacion jsonb;
    evidencia jsonb;
    orden bigint;
    total_autorizaciones integer;
    total_evidencias integer;
    total_partes bigint;
BEGIN
    IF jsonb_typeof(p_manifiesto) <> 'object'
       OR (SELECT count(*) FROM jsonb_object_keys(p_manifiesto)) <> 16
       OR NOT (p_manifiesto ?& ARRAY[
           'esquema', 'finalidad', 'version_esquema', 'referencia',
           'proceso_ref', 'solicitud_ref', 'sujeto_ref',
           'baremacion_merito_ref', 'decision_ref', 'version_base',
           'huella_version_base_sha256', 'autorizaciones', 'evidencias',
           'creado_en', 'huella_manifiesto_sha256',
           'sello_manifiesto_hmac_sha256'
       ])
       OR jsonb_typeof(p_manifiesto -> 'esquema') IS DISTINCT FROM 'string'
       OR jsonb_typeof(p_manifiesto -> 'finalidad') IS DISTINCT FROM 'string'
       OR jsonb_typeof(p_manifiesto -> 'version_esquema') IS DISTINCT FROM
          'number'
       OR jsonb_typeof(p_manifiesto -> 'referencia') IS DISTINCT FROM 'string'
       OR jsonb_typeof(p_manifiesto -> 'proceso_ref') IS DISTINCT FROM 'string'
       OR jsonb_typeof(p_manifiesto -> 'solicitud_ref') IS DISTINCT FROM
          'string'
       OR jsonb_typeof(p_manifiesto -> 'sujeto_ref') IS DISTINCT FROM 'string'
       OR jsonb_typeof(p_manifiesto -> 'baremacion_merito_ref')
          IS DISTINCT FROM 'string'
       OR jsonb_typeof(p_manifiesto -> 'decision_ref') IS DISTINCT FROM
          'string'
       OR jsonb_typeof(p_manifiesto -> 'version_base') IS DISTINCT FROM
          'number'
       OR jsonb_typeof(p_manifiesto -> 'huella_version_base_sha256')
          IS DISTINCT FROM 'string'
       OR jsonb_typeof(p_manifiesto -> 'autorizaciones') IS DISTINCT FROM
          'array'
       OR jsonb_typeof(p_manifiesto -> 'evidencias') IS DISTINCT FROM 'array'
       OR jsonb_typeof(p_manifiesto -> 'creado_en') IS DISTINCT FROM 'string'
       OR jsonb_typeof(p_manifiesto -> 'huella_manifiesto_sha256')
          IS DISTINCT FROM 'string'
       OR jsonb_typeof(p_manifiesto -> 'sello_manifiesto_hmac_sha256')
          IS DISTINCT FROM 'string'
       OR p_manifiesto ->> 'esquema' IS DISTINCT FROM
          'vec.bolsa.manifiesto_probatorio'
       OR p_manifiesto ->> 'finalidad' IS DISTINCT FROM
          'decision_tecnica_baremacion'
       OR p_manifiesto ->> 'version_esquema' IS DISTINCT FROM '3'
       OR vec_bolsa_baremacion.referencia_manifiesto_v3_valida(
           p_manifiesto ->> 'referencia', 512
       ) IS NOT TRUE
       OR vec_bolsa_baremacion.referencia_manifiesto_v3_valida(
           p_manifiesto ->> 'proceso_ref', 512
       ) IS NOT TRUE
       OR vec_bolsa_baremacion.referencia_manifiesto_v3_valida(
           p_manifiesto ->> 'solicitud_ref', 512
       ) IS NOT TRUE
       OR vec_bolsa_baremacion.referencia_manifiesto_v3_valida(
           p_manifiesto ->> 'sujeto_ref', 512
       ) IS NOT TRUE
       OR vec_bolsa_baremacion.referencia_manifiesto_v3_valida(
           p_manifiesto ->> 'baremacion_merito_ref', 512
       ) IS NOT TRUE
       OR vec_bolsa_baremacion.referencia_manifiesto_v3_valida(
           p_manifiesto ->> 'decision_ref', 512
       ) IS NOT TRUE
       OR p_manifiesto ->> 'version_base' IS NULL
       OR (p_manifiesto ->> 'version_base') !~ '^[1-9][0-9]{0,19}$'
       OR (p_manifiesto ->> 'version_base')::numeric >
          18446744073709551615
       OR vec_bolsa_baremacion.huella_sha256_valida(
           p_manifiesto ->> 'huella_version_base_sha256'
       ) IS NOT TRUE
       OR vec_bolsa_baremacion.instante_canonico_manifiesto_v3(
           p_manifiesto ->> 'creado_en'
       ) IS NOT TRUE
       OR vec_bolsa_baremacion.huella_sha256_valida(
           p_manifiesto ->> 'huella_manifiesto_sha256'
       ) IS NOT TRUE
       OR vec_bolsa_baremacion.sello_hmac_manifiesto_v3_valido(
           p_manifiesto ->> 'sello_manifiesto_hmac_sha256'
       ) IS NOT TRUE
       THEN
        RETURN NULL;
    END IF;

    total_autorizaciones := jsonb_array_length(
        p_manifiesto -> 'autorizaciones'
    );
    total_evidencias := jsonb_array_length(p_manifiesto -> 'evidencias');
    IF total_autorizaciones NOT BETWEEN 1 AND 4096
       OR total_evidencias NOT BETWEEN 1 AND 4096 THEN
        RETURN NULL;
    END IF;

    FOR autorizacion, orden IN
        SELECT elemento, ordinalidad
          FROM jsonb_array_elements(p_manifiesto -> 'autorizaciones')
               WITH ORDINALITY AS lista(elemento, ordinalidad)
         ORDER BY ordinalidad
    LOOP
        IF jsonb_typeof(autorizacion) <> 'object'
           OR (SELECT count(*) FROM jsonb_object_keys(autorizacion)) <> 5
           OR NOT (autorizacion ?& ARRAY[
               'secuencia', 'accion', 'clase_recurso', 'recurso_ref',
               'autorizacion_ref'
           ])
           OR jsonb_typeof(autorizacion -> 'secuencia') IS DISTINCT FROM
              'number'
           OR jsonb_typeof(autorizacion -> 'accion') IS DISTINCT FROM
              'string'
           OR jsonb_typeof(autorizacion -> 'clase_recurso') IS DISTINCT FROM
              'string'
           OR jsonb_typeof(autorizacion -> 'recurso_ref') IS DISTINCT FROM
              'string'
           OR jsonb_typeof(autorizacion -> 'autorizacion_ref')
              IS DISTINCT FROM 'string'
           OR autorizacion ->> 'secuencia' IS DISTINCT FROM orden::text
           OR vec_bolsa_baremacion.accion_recurso_manifiesto_v3_valida(
               autorizacion ->> 'accion',
               autorizacion ->> 'clase_recurso'
           ) IS NOT TRUE
           OR vec_bolsa_baremacion.referencia_manifiesto_v3_valida(
               autorizacion ->> 'recurso_ref', 512
           ) IS NOT TRUE
           OR vec_bolsa_baremacion.referencia_manifiesto_v3_valida(
               autorizacion ->> 'autorizacion_ref', 512
           ) IS NOT TRUE THEN
            RETURN NULL;
        END IF;
    END LOOP;

    FOR evidencia, orden IN
        SELECT elemento, ordinalidad
          FROM jsonb_array_elements(p_manifiesto -> 'evidencias')
               WITH ORDINALITY AS lista(elemento, ordinalidad)
         ORDER BY ordinalidad
    LOOP
        IF jsonb_typeof(evidencia) <> 'object'
           OR (SELECT count(*) FROM jsonb_object_keys(evidencia)) <> 4
           OR NOT (evidencia ?& ARRAY[
               'secuencia', 'tipo', 'referencia',
               'huella_evidencia_sha256'
           ])
           OR jsonb_typeof(evidencia -> 'secuencia') IS DISTINCT FROM
              'number'
           OR jsonb_typeof(evidencia -> 'tipo') IS DISTINCT FROM 'string'
           OR jsonb_typeof(evidencia -> 'referencia') IS DISTINCT FROM
              'string'
           OR jsonb_typeof(evidencia -> 'huella_evidencia_sha256')
              IS DISTINCT FROM 'string'
           OR evidencia ->> 'secuencia' IS DISTINCT FROM orden::text
           OR vec_bolsa_baremacion.tipo_evidencia_manifiesto_v3_valido(
               evidencia ->> 'tipo'
           ) IS NOT TRUE
           OR vec_bolsa_baremacion.referencia_manifiesto_v3_valida(
               evidencia ->> 'referencia', 512
           ) IS NOT TRUE
           OR vec_bolsa_baremacion.huella_sha256_valida(
               evidencia ->> 'huella_evidencia_sha256'
           ) IS NOT TRUE THEN
            RETURN NULL;
        END IF;
    END LOOP;
    WITH partes(grupo, elemento, campo, valor) AS (
        SELECT 0, 0::bigint, fijo.campo, fijo.valor
          FROM (VALUES
            (1, p_manifiesto ->> 'esquema'),
            (2, p_manifiesto ->> 'finalidad'),
            (3, p_manifiesto ->> 'version_esquema'),
            (4, p_manifiesto ->> 'referencia'),
            (5, p_manifiesto ->> 'proceso_ref'),
            (6, p_manifiesto ->> 'solicitud_ref'),
            (7, p_manifiesto ->> 'sujeto_ref'),
            (8, p_manifiesto ->> 'baremacion_merito_ref'),
            (9, p_manifiesto ->> 'decision_ref'),
            (10, p_manifiesto ->> 'version_base'),
            (11, p_manifiesto ->> 'huella_version_base_sha256'),
            (12, p_manifiesto ->> 'creado_en'),
            (13, total_autorizaciones::text)
          ) AS fijo(campo, valor)
        UNION ALL
        SELECT 1, lista.ordinalidad, parte.campo, parte.valor
          FROM jsonb_array_elements(p_manifiesto -> 'autorizaciones')
               WITH ORDINALITY AS lista(elemento, ordinalidad)
          CROSS JOIN LATERAL (VALUES
            (1, lista.elemento ->> 'secuencia'),
            (2, lista.elemento ->> 'accion'),
            (3, lista.elemento ->> 'clase_recurso'),
            (4, lista.elemento ->> 'recurso_ref'),
            (5, lista.elemento ->> 'autorizacion_ref')
          ) AS parte(campo, valor)
        UNION ALL
        SELECT 2, 0::bigint, 1, total_evidencias::text
        UNION ALL
        SELECT 3, lista.ordinalidad, parte.campo, parte.valor
          FROM jsonb_array_elements(p_manifiesto -> 'evidencias')
               WITH ORDINALITY AS lista(elemento, ordinalidad)
          CROSS JOIN LATERAL (VALUES
            (1, lista.elemento ->> 'secuencia'),
            (2, lista.elemento ->> 'tipo'),
            (3, lista.elemento ->> 'referencia'),
            (4, lista.elemento ->> 'huella_evidencia_sha256')
          ) AS parte(campo, valor)
    )
    SELECT count(valor), string_agg(
               vec_bolsa_baremacion.parte_canonica_manifiesto_v3(valor),
               ''::bytea ORDER BY grupo, elemento, campo
           )
      INTO total_partes, contenido
      FROM partes;
    IF total_partes <> 14 + 5 * total_autorizaciones
                         + 4 * total_evidencias THEN
        RETURN NULL;
    END IF;
    RETURN contenido;
EXCEPTION
    WHEN data_exception OR invalid_text_representation
        OR numeric_value_out_of_range OR character_not_in_repertoire
        OR untranslatable_character THEN
        RETURN NULL;
END
$funcion$;


CREATE FUNCTION vec_bolsa_baremacion.obtener_version_vigente_con_archivo_probatorio_v3(
    p_operacion jsonb,
    p_prueba jsonb,
    p_decision_canonica bytea,
    p_recurso_canonico bytea
)
RETURNS TABLE (
    resultado text,
    numero_version text,
    huella_estado_sha256 text,
    agregado_canonico bytea,
    confirmada_en timestamptz,
    auditoria_ref text,
    archivo_probatorio_documento jsonb
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    operacion_v1 jsonb;
    respuesta record;
BEGIN
    resultado := 'rechazada';
    numero_version := '';
    huella_estado_sha256 := '';
    auditoria_ref := '';
    IF p_operacion IS NULL OR jsonb_typeof(p_operacion) <> 'object'
       OR p_operacion ->> 'esquema' <>
          'vec.bolsa.baremacion.lectura-vigente-postgresql.v3'
       OR vec_bolsa_baremacion.referencia_manifiesto_v3_valida(
           p_operacion ->> 'baremacion_merito_ref', 512
       ) IS NOT TRUE THEN
        RETURN NEXT;
        RETURN;
    END IF;
    operacion_v1 := jsonb_set(
        p_operacion, '{esquema}',
        to_jsonb(
            'vec.bolsa.baremacion.lectura-vigente-postgresql.v1'::text
        ), false
    );
    SELECT * INTO respuesta
      FROM vec_bolsa_baremacion.obtener_version_vigente(
          operacion_v1, p_prueba, p_decision_canonica,
          p_recurso_canonico
      );
    resultado := respuesta.resultado;
    numero_version := respuesta.numero_version;
    huella_estado_sha256 := respuesta.huella_estado_sha256;
    agregado_canonico := respuesta.agregado_canonico;
    confirmada_en := respuesta.confirmada_en;
    auditoria_ref := respuesta.auditoria_ref;
    IF resultado = 'obtenida' THEN
        archivo_probatorio_documento :=
            vec_bolsa_baremacion.construir_archivo_probatorio_v3(
                p_operacion ->> 'baremacion_merito_ref',
                numero_version::numeric
            );
        IF archivo_probatorio_documento IS NULL THEN
            resultado := 'evidencia_no_confiable';
            numero_version := '';
            huella_estado_sha256 := '';
            agregado_canonico := NULL;
            confirmada_en := NULL;
            auditoria_ref := '';
        END IF;
    END IF;
    RETURN NEXT;
EXCEPTION
    WHEN data_exception OR invalid_text_representation
        OR numeric_value_out_of_range OR no_data_found OR too_many_rows THEN
        resultado := 'rechazada';
        numero_version := '';
        huella_estado_sha256 := '';
        agregado_canonico := NULL;
        confirmada_en := NULL;
        auditoria_ref := '';
        archivo_probatorio_documento := NULL;
        RETURN NEXT;
END
$funcion$;

CREATE FUNCTION vec_bolsa_baremacion.obtener_version_con_archivo_probatorio_v3(
    p_operacion jsonb,
    p_prueba jsonb,
    p_decision_canonica bytea,
    p_recurso_canonico bytea
)
RETURNS TABLE (
    resultado text,
    numero_version text,
    huella_estado_sha256 text,
    agregado_canonico bytea,
    confirmada_en timestamptz,
    auditoria_ref text,
    archivo_probatorio_documento jsonb
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    instante timestamptz(6) := clock_timestamp();
    numero_solicitado numeric(20, 0);
    autorizacion record;
    uso record;
BEGIN
    resultado := 'rechazada';
    numero_version := '';
    huella_estado_sha256 := '';
    auditoria_ref := '';
    archivo_probatorio_documento := NULL;
    IF p_operacion IS NULL OR jsonb_typeof(p_operacion) <> 'object'
       OR (SELECT count(*) FROM jsonb_object_keys(p_operacion)) <> 4
       OR NOT (p_operacion ?& ARRAY[
           'esquema', 'baremacion_merito_ref', 'numero_version',
           'huella_efecto_sha256'
       ])
       OR jsonb_typeof(p_operacion -> 'esquema') IS DISTINCT FROM 'string'
       OR jsonb_typeof(p_operacion -> 'baremacion_merito_ref')
          IS DISTINCT FROM 'string'
       OR jsonb_typeof(p_operacion -> 'numero_version')
          IS DISTINCT FROM 'string'
       OR jsonb_typeof(p_operacion -> 'huella_efecto_sha256')
          IS DISTINCT FROM 'string'
       OR p_operacion ->> 'esquema' IS DISTINCT FROM
          'vec.bolsa.baremacion.lectura-version-postgresql.v3'
       OR vec_bolsa_baremacion.referencia_manifiesto_v3_valida(
           p_operacion ->> 'baremacion_merito_ref', 512
       ) IS NOT TRUE
       OR (p_operacion ->> 'numero_version') !~ '^[1-9][0-9]{0,19}$'
       OR vec_bolsa_baremacion.huella_sha256_valida(
           p_operacion ->> 'huella_efecto_sha256'
       ) IS NOT TRUE THEN
        RETURN NEXT;
        RETURN;
    END IF;
    numero_solicitado := (p_operacion ->> 'numero_version')::numeric;
    IF numero_solicitado > 18446744073709551615 THEN
        RETURN NEXT;
        RETURN;
    END IF;
    SELECT * INTO autorizacion
      FROM vec_bolsa_baremacion.revalidar_operacion(
          p_prueba, p_decision_canonica, p_recurso_canonico,
          'bolsa.baremacion.version.consultar', 'baremacion',
          p_operacion ->> 'baremacion_merito_ref',
          '["baremacion"]'::jsonb, instante
      );
    IF NOT FOUND THEN
        resultado := 'autorizacion_obsoleta';
        RETURN NEXT;
        RETURN;
    END IF;
    PERFORM pg_advisory_xact_lock(hashtextextended(
        'vec_bolsa_baremacion:uso-lectura:' ||
            (p_prueba ->> 'decision_ref'), 0
    ));
    SELECT * INTO uso
      FROM vec_bolsa_baremacion.uso_decision
     WHERE decision_ref = p_prueba ->> 'decision_ref'
     FOR SHARE;
    IF FOUND THEN
        IF uso.esquema_huella_decision IS DISTINCT FROM
               p_prueba ->> 'esquema_huella'
           OR uso.huella_decision_sha256 IS DISTINCT FROM
               p_prueba ->> 'huella_decision_sha256'
           OR uso.huella_efecto_sha256 IS DISTINCT FROM
               p_operacion ->> 'huella_efecto_sha256'
           OR uso.tipo_efecto <> 'lectura_version' THEN
            resultado := 'autorizacion_reutilizada';
            RETURN NEXT;
            RETURN;
        END IF;
        SELECT almacenada.numero::text,
               almacenada.huella_estado_sha256,
               almacenada.agregado_canonico, almacenada.confirmada_en,
               almacenada.auditoria_ref
          INTO numero_version, huella_estado_sha256, agregado_canonico,
               confirmada_en, auditoria_ref
          FROM vec_bolsa_baremacion.version_baremacion AS almacenada
         WHERE almacenada.auditoria_ref = uso.resultado_ref
           AND almacenada.baremacion_merito_ref =
               p_operacion ->> 'baremacion_merito_ref'
           AND almacenada.numero = numero_solicitado
           AND almacenada.sujeto_ref = p_prueba ->> 'sujeto_ref';
        IF NOT FOUND THEN
            resultado := 'evidencia_no_confiable';
            numero_version := '';
            huella_estado_sha256 := '';
            agregado_canonico := NULL;
            confirmada_en := NULL;
            auditoria_ref := '';
            archivo_probatorio_documento := NULL;
            RETURN NEXT;
            RETURN;
        END IF;
    ELSE
        SELECT almacenada.numero::text,
               almacenada.huella_estado_sha256,
               almacenada.agregado_canonico, almacenada.confirmada_en,
               almacenada.auditoria_ref
          INTO numero_version, huella_estado_sha256, agregado_canonico,
               confirmada_en, auditoria_ref
          FROM vec_bolsa_baremacion.version_baremacion AS almacenada
         WHERE almacenada.baremacion_merito_ref =
               p_operacion ->> 'baremacion_merito_ref'
           AND almacenada.numero = numero_solicitado
           AND almacenada.sujeto_ref = p_prueba ->> 'sujeto_ref'
         FOR SHARE;
        IF NOT FOUND THEN
            resultado := 'no_encontrada';
            numero_version := '';
            huella_estado_sha256 := '';
            agregado_canonico := NULL;
            confirmada_en := NULL;
            auditoria_ref := '';
            archivo_probatorio_documento := NULL;
            RETURN NEXT;
            RETURN;
        END IF;
        INSERT INTO vec_bolsa_baremacion.uso_decision (
            decision_ref, esquema_huella_decision,
            huella_decision_sha256, huella_efecto_sha256, tipo_efecto,
            resultado_ref, atestacion_ref, atestacion_version, consumida_en
        ) VALUES (
            p_prueba ->> 'decision_ref', p_prueba ->> 'esquema_huella',
            p_prueba ->> 'huella_decision_sha256',
            p_operacion ->> 'huella_efecto_sha256', 'lectura_version',
            auditoria_ref, autorizacion.atestacion_ref,
            autorizacion.atestacion_version, instante
        );
    END IF;
    archivo_probatorio_documento :=
        vec_bolsa_baremacion.construir_archivo_probatorio_v3(
            p_operacion ->> 'baremacion_merito_ref', numero_solicitado
        );
    IF archivo_probatorio_documento IS NULL THEN
        resultado := 'evidencia_no_confiable';
        numero_version := '';
        huella_estado_sha256 := '';
        agregado_canonico := NULL;
        confirmada_en := NULL;
        auditoria_ref := '';
        RETURN NEXT;
        RETURN;
    END IF;
    resultado := 'obtenida';
    RETURN NEXT;
EXCEPTION
    WHEN unique_violation THEN
        resultado := 'colision';
        numero_version := '';
        huella_estado_sha256 := '';
        agregado_canonico := NULL;
        confirmada_en := NULL;
        auditoria_ref := '';
        archivo_probatorio_documento := NULL;
        RETURN NEXT;
    WHEN data_exception OR invalid_text_representation
        OR numeric_value_out_of_range OR datetime_field_overflow
        OR check_violation OR foreign_key_violation
        OR no_data_found OR too_many_rows THEN
        resultado := 'rechazada';
        numero_version := '';
        huella_estado_sha256 := '';
        agregado_canonico := NULL;
        confirmada_en := NULL;
        auditoria_ref := '';
        archivo_probatorio_documento := NULL;
        RETURN NEXT;
END
$funcion$;

CREATE FUNCTION vec_bolsa_baremacion.obtener_evidencia_transaccion_con_archivo_probatorio_v3(
    p_operacion jsonb,
    p_prueba jsonb,
    p_decision_canonica bytea,
    p_recurso_canonico bytea
)
RETURNS TABLE (
    resultado text,
    numero_version text,
    huella_estado_sha256 text,
    agregado_canonico bytea,
    confirmada_en timestamptz,
    auditoria_documento jsonb,
    evento_documento jsonb,
    archivo_probatorio_documento jsonb
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    instante timestamptz(6) := clock_timestamp();
    numero_solicitado numeric(20, 0);
    autorizacion record;
    uso record;
    uso_encontrado boolean := false;
BEGIN
    resultado := 'rechazada';
    numero_version := '';
    huella_estado_sha256 := '';
    IF p_operacion IS NULL OR jsonb_typeof(p_operacion) <> 'object'
       OR (SELECT count(*) FROM jsonb_object_keys(p_operacion)) <> 6
       OR NOT (p_operacion ?& ARRAY[
           'esquema', 'baremacion_merito_ref', 'numero_version',
           'auditoria_ref', 'evento_outbox_ref', 'huella_efecto_sha256'
       ])
       OR jsonb_typeof(p_operacion -> 'esquema') IS DISTINCT FROM 'string'
       OR jsonb_typeof(p_operacion -> 'baremacion_merito_ref')
          IS DISTINCT FROM 'string'
       OR jsonb_typeof(p_operacion -> 'numero_version')
          IS DISTINCT FROM 'string'
       OR jsonb_typeof(p_operacion -> 'auditoria_ref')
          IS DISTINCT FROM 'string'
       OR jsonb_typeof(p_operacion -> 'evento_outbox_ref')
          IS DISTINCT FROM 'string'
       OR jsonb_typeof(p_operacion -> 'huella_efecto_sha256')
          IS DISTINCT FROM 'string'
       OR p_operacion ->> 'esquema' IS DISTINCT FROM
          'vec.bolsa.baremacion.lectura-evidencia-postgresql.v3'
       OR vec_bolsa_baremacion.referencia_manifiesto_v3_valida(
           p_operacion ->> 'baremacion_merito_ref', 512
       ) IS NOT TRUE
       OR (p_operacion ->> 'numero_version') !~ '^[1-9][0-9]{0,19}$'
       OR vec_bolsa_baremacion.referencia_manifiesto_v3_valida(
           p_operacion ->> 'auditoria_ref', 512
       ) IS NOT TRUE
       OR vec_bolsa_baremacion.referencia_manifiesto_v3_valida(
           p_operacion ->> 'evento_outbox_ref', 512
       ) IS NOT TRUE
       OR p_operacion ->> 'auditoria_ref' =
          p_operacion ->> 'evento_outbox_ref'
       OR vec_bolsa_baremacion.huella_sha256_valida(
           p_operacion ->> 'huella_efecto_sha256'
       ) IS NOT TRUE THEN
        RETURN NEXT;
        RETURN;
    END IF;
    numero_solicitado := (p_operacion ->> 'numero_version')::numeric;
    IF numero_solicitado > 18446744073709551615 THEN
        RETURN NEXT;
        RETURN;
    END IF;
    SELECT * INTO autorizacion
      FROM vec_bolsa_baremacion.revalidar_operacion(
          p_prueba, p_decision_canonica, p_recurso_canonico,
          'bolsa.baremacion.transaccion.consultar', 'transaccion',
          p_operacion ->> 'auditoria_ref',
          '["auditoria","evento_outbox","evidencia_transaccion"]'::jsonb,
          instante
      );
    IF NOT FOUND THEN
        resultado := 'autorizacion_obsoleta';
        RETURN NEXT;
        RETURN;
    END IF;
    PERFORM pg_advisory_xact_lock(hashtextextended(
        'vec_bolsa_baremacion:uso-lectura:' ||
            (p_prueba ->> 'decision_ref'), 0
    ));
    SELECT * INTO uso
      FROM vec_bolsa_baremacion.uso_decision
     WHERE decision_ref = p_prueba ->> 'decision_ref'
     FOR SHARE;
    uso_encontrado := FOUND;
    IF uso_encontrado AND (
       uso.esquema_huella_decision IS DISTINCT FROM
           p_prueba ->> 'esquema_huella'
       OR uso.huella_decision_sha256 IS DISTINCT FROM
           p_prueba ->> 'huella_decision_sha256'
       OR uso.huella_efecto_sha256 IS DISTINCT FROM
           p_operacion ->> 'huella_efecto_sha256'
       OR uso.tipo_efecto <> 'lectura_evidencia'
       OR uso.resultado_ref IS DISTINCT FROM
          p_operacion ->> 'auditoria_ref') THEN
        resultado := 'autorizacion_reutilizada';
        RETURN NEXT;
        RETURN;
    END IF;
    SELECT version_almacenada.numero::text,
           version_almacenada.huella_estado_sha256,
           version_almacenada.agregado_canonico,
           version_almacenada.confirmada_en,
           to_jsonb(auditoria_almacenada), to_jsonb(evento_almacenado)
      INTO numero_version, huella_estado_sha256, agregado_canonico,
           confirmada_en, auditoria_documento, evento_documento
      FROM vec_bolsa_baremacion.version_baremacion AS version_almacenada
      JOIN vec_bolsa_baremacion.auditoria AS auditoria_almacenada
        ON auditoria_almacenada.referencia =
           version_almacenada.auditoria_ref
      JOIN vec_bolsa_baremacion.evento_outbox AS evento_almacenado
        ON evento_almacenado.referencia =
           version_almacenada.evento_outbox_ref
       AND evento_almacenado.auditoria_ref =
           auditoria_almacenada.referencia
       AND evento_almacenado.huella_auditoria_sha256 =
           auditoria_almacenada.huella_registro_sha256
     WHERE version_almacenada.baremacion_merito_ref =
           p_operacion ->> 'baremacion_merito_ref'
       AND version_almacenada.numero = numero_solicitado
       AND version_almacenada.auditoria_ref =
           p_operacion ->> 'auditoria_ref'
       AND version_almacenada.evento_outbox_ref =
           p_operacion ->> 'evento_outbox_ref'
       AND version_almacenada.sujeto_ref = p_prueba ->> 'sujeto_ref'
       AND auditoria_almacenada.sujeto_ref =
           version_almacenada.sujeto_ref
       AND auditoria_almacenada.baremacion_merito_ref =
           version_almacenada.baremacion_merito_ref
       AND auditoria_almacenada.version_nueva =
           version_almacenada.numero
       AND auditoria_almacenada.huella_nueva_sha256 =
           version_almacenada.huella_estado_sha256
       AND auditoria_almacenada.registrada_en =
           version_almacenada.confirmada_en
       AND evento_almacenado.sujeto_ref = version_almacenada.sujeto_ref
       AND evento_almacenado.baremacion_merito_ref =
           version_almacenada.baremacion_merito_ref
       AND evento_almacenado.version_nueva = version_almacenada.numero
       AND evento_almacenado.huella_nueva_sha256 =
           version_almacenada.huella_estado_sha256
       AND evento_almacenado.registrada_en =
           version_almacenada.confirmada_en
     FOR SHARE OF version_almacenada, auditoria_almacenada,
                  evento_almacenado;
    IF NOT FOUND THEN
        IF uso_encontrado THEN
            resultado := 'evidencia_no_confiable';
        ELSE
            resultado := 'no_encontrada';
        END IF;
        numero_version := '';
        huella_estado_sha256 := '';
        agregado_canonico := NULL;
        confirmada_en := NULL;
        auditoria_documento := NULL;
        evento_documento := NULL;
        archivo_probatorio_documento := NULL;
        RETURN NEXT;
        RETURN;
    END IF;
    IF NOT uso_encontrado THEN
        INSERT INTO vec_bolsa_baremacion.uso_decision (
            decision_ref, esquema_huella_decision,
            huella_decision_sha256, huella_efecto_sha256, tipo_efecto,
            resultado_ref, atestacion_ref, atestacion_version, consumida_en
        ) VALUES (
            p_prueba ->> 'decision_ref', p_prueba ->> 'esquema_huella',
            p_prueba ->> 'huella_decision_sha256',
            p_operacion ->> 'huella_efecto_sha256', 'lectura_evidencia',
            p_operacion ->> 'auditoria_ref', autorizacion.atestacion_ref,
            autorizacion.atestacion_version, instante
        );
    END IF;
    archivo_probatorio_documento :=
        vec_bolsa_baremacion.construir_archivo_probatorio_v3(
            p_operacion ->> 'baremacion_merito_ref', numero_solicitado
        );
    IF archivo_probatorio_documento IS NULL THEN
        resultado := 'evidencia_no_confiable';
        numero_version := '';
        huella_estado_sha256 := '';
        agregado_canonico := NULL;
        confirmada_en := NULL;
        auditoria_documento := NULL;
        evento_documento := NULL;
        archivo_probatorio_documento := NULL;
        RETURN NEXT;
        RETURN;
    END IF;
    resultado := 'obtenida';
    RETURN NEXT;
EXCEPTION
    WHEN unique_violation THEN
        resultado := 'colision';
        numero_version := '';
        huella_estado_sha256 := '';
        agregado_canonico := NULL;
        confirmada_en := NULL;
        auditoria_documento := NULL;
        evento_documento := NULL;
        archivo_probatorio_documento := NULL;
        RETURN NEXT;
    WHEN data_exception OR invalid_text_representation
        OR numeric_value_out_of_range OR datetime_field_overflow
        OR check_violation OR foreign_key_violation
        OR no_data_found OR too_many_rows THEN
        resultado := 'rechazada';
        numero_version := '';
        huella_estado_sha256 := '';
        agregado_canonico := NULL;
        confirmada_en := NULL;
        auditoria_documento := NULL;
        evento_documento := NULL;
        archivo_probatorio_documento := NULL;
        RETURN NEXT;
END
$funcion$;

-- Prevalidacion explicita: revalida y consume una autorizacion dedicada en una
-- transaccion corta. El cliente debe cerrar la llamada antes de verificar el
-- snapshot con KMS; una caida exige otra autorizacion salvo replay exacto.
CREATE FUNCTION vec_bolsa_baremacion.obtener_archivo_probatorio_previo_cambio_v3(
    p_operacion jsonb,
    p_prueba jsonb,
    p_decision_canonica bytea,
    p_recurso_canonico bytea
)
RETURNS TABLE (
    resultado text,
    numero_version text,
    huella_estado_sha256 text,
    agregado_canonico bytea,
    confirmada_en timestamptz,
    archivo_probatorio_documento jsonb,
    huella_prevalidacion_sha256 text
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    instante timestamptz(6) := clock_timestamp();
    numero_solicitado numeric(20, 0);
    numero_resultado numeric(20, 0);
    huella_resultado text := '';
    autorizacion record;
    reserva record;
    version_resultado record;
    uso record;
    consumo record;
    resultado_final record;
    huella_archivo text;
    huella_calculada text;
    huella_efecto_calculada text;
BEGIN
    resultado := 'rechazada';
    numero_version := '';
    huella_estado_sha256 := '';
    huella_prevalidacion_sha256 := '';
    IF p_operacion IS NULL OR jsonb_typeof(p_operacion) <> 'object'
       OR (SELECT count(*) FROM jsonb_object_keys(p_operacion)) <> 8
       OR NOT (p_operacion ?& ARRAY[
           'esquema', 'clase', 'baremacion_merito_ref',
           'version_esperada', 'huella_version_esperada_sha256',
           'huella_token_sha256', 'huella_confirmacion_sha256',
           'huella_efecto_prevalidacion_sha256'
       ])
       OR p_operacion ->> 'esquema' <>
          'vec.bolsa.baremacion.prevalidacion-archivo-postgresql.v3'
       OR p_operacion ->> 'clase' <> 'incorporar_decision'
       OR vec_bolsa_baremacion.referencia_manifiesto_v3_valida(
           p_operacion ->> 'baremacion_merito_ref', 512
       ) IS NOT TRUE
       OR vec_bolsa_baremacion.huella_sha256_valida(
           p_operacion ->> 'huella_token_sha256'
       ) IS NOT TRUE
       OR vec_bolsa_baremacion.huella_sha256_valida(
           p_operacion ->> 'huella_confirmacion_sha256'
       ) IS NOT TRUE
       OR vec_bolsa_baremacion.huella_sha256_valida(
           p_operacion ->> 'huella_efecto_prevalidacion_sha256'
       ) IS NOT TRUE THEN
        RETURN NEXT;
        RETURN;
    END IF;
    IF (p_operacion ->> 'version_esperada') !~
          '^[1-9][0-9]{0,19}$'
       OR vec_bolsa_baremacion.huella_sha256_valida(
           p_operacion ->> 'huella_version_esperada_sha256'
       ) IS NOT TRUE THEN
        RETURN NEXT;
        RETURN;
    END IF;
    numero_solicitado :=
        (p_operacion ->> 'version_esperada')::numeric;
    IF numero_solicitado > 4096 THEN
        RETURN NEXT;
        RETURN;
    END IF;
    huella_efecto_calculada := vec_bolsa_baremacion.huella_canonica(ARRAY[
        'efecto-prevalidacion-archivo-probatorio-baremacion-v3',
        p_operacion ->> 'huella_confirmacion_sha256'
    ]);
    IF huella_efecto_calculada IS DISTINCT FROM
       p_operacion ->> 'huella_efecto_prevalidacion_sha256' THEN
        RETURN NEXT;
        RETURN;
    END IF;

    SELECT * INTO autorizacion
      FROM vec_bolsa_baremacion.revalidar_operacion(
          p_prueba, p_decision_canonica, p_recurso_canonico,
          'bolsa.baremacion.archivo.prevalidar', 'baremacion',
          p_operacion ->> 'baremacion_merito_ref',
          '["archivo_probatorio"]'::jsonb, instante
      );
    IF NOT FOUND THEN
        resultado := 'autorizacion_obsoleta';
        RETURN NEXT;
        RETURN;
    END IF;

    PERFORM pg_advisory_xact_lock(hashtextextended(
        'vec_bolsa_baremacion:prevalidacion-archivo:' ||
            (p_operacion ->> 'huella_token_sha256'), 0
    ));
    SELECT actual.reserva_ref, actual.version AS revision_reserva,
           actual.estado, version.principal_ref, version.sujeto_ref,
           version.vinculo_autenticacion_actor,
           version.baremacion_merito_ref, version.clase,
           version.version_esperada, version.huella_version_esperada_sha256,
           version.expira_en, version.numero_version_confirmada,
           version.huella_confirmacion_sha256
      INTO reserva
      FROM vec_bolsa_baremacion.token_reserva AS token
      JOIN vec_bolsa_baremacion.reserva_actual AS actual
        ON actual.ambito_idempotencia_sha256 =
           token.ambito_idempotencia_sha256
       AND actual.reserva_ref = token.reserva_ref
      JOIN vec_bolsa_baremacion.reserva_version AS version
        ON version.ambito_idempotencia_sha256 =
           actual.ambito_idempotencia_sha256
       AND version.reserva_ref = actual.reserva_ref
       AND version.version = actual.version
     WHERE token.huella_token_sha256 =
           p_operacion ->> 'huella_token_sha256'
     FOR UPDATE OF actual;
    IF NOT FOUND
       OR reserva.estado NOT IN ('activa', 'confirmada')
       OR reserva.principal_ref IS DISTINCT FROM
          autorizacion.decision_canonica ->> 'principal_id'
       OR reserva.sujeto_ref IS DISTINCT FROM p_prueba ->> 'sujeto_ref'
       OR reserva.vinculo_autenticacion_actor IS DISTINCT FROM
          autorizacion.decision_canonica -> 'vinculo_autenticacion_actor'
       OR reserva.baremacion_merito_ref IS DISTINCT FROM
          p_operacion ->> 'baremacion_merito_ref'
       OR reserva.clase IS DISTINCT FROM p_operacion ->> 'clase'
       OR reserva.version_esperada IS DISTINCT FROM numero_solicitado
       OR reserva.huella_version_esperada_sha256 IS DISTINCT FROM
          p_operacion ->> 'huella_version_esperada_sha256'
       OR (reserva.estado = 'activa' AND instante >= reserva.expira_en) THEN
        resultado := 'reserva_invalida';
        RETURN NEXT;
        RETURN;
    END IF;

    SELECT * INTO uso
      FROM vec_bolsa_baremacion.uso_decision
     WHERE decision_ref = p_prueba ->> 'decision_ref'
     FOR SHARE;
    IF FOUND THEN
        SELECT * INTO consumo
          FROM vec_bolsa_baremacion.prevalidacion_archivo_probatorio_v3
         WHERE autorizacion_prevalidacion_ref =
               p_prueba ->> 'decision_ref';
        IF NOT FOUND OR uso.tipo_efecto <> 'prevalidacion_archivo'
           OR uso.esquema_huella_decision IS DISTINCT FROM
              p_prueba ->> 'esquema_huella'
           OR uso.huella_decision_sha256 IS DISTINCT FROM
              p_prueba ->> 'huella_decision_sha256'
           OR uso.huella_efecto_sha256 IS DISTINCT FROM
              consumo.huella_efecto_prevalidacion_sha256
           OR consumo.huella_efecto_prevalidacion_sha256 IS DISTINCT FROM
              huella_efecto_calculada
           OR consumo.huella_confirmacion_sha256 IS DISTINCT FROM
              p_operacion ->> 'huella_confirmacion_sha256'
           OR uso.resultado_ref IS DISTINCT FROM
              p_prueba ->> 'decision_ref'
           OR uso.atestacion_ref IS DISTINCT FROM
              autorizacion.atestacion_ref
           OR uso.atestacion_version IS DISTINCT FROM
              autorizacion.atestacion_version
           OR consumo.reserva_ref IS DISTINCT FROM reserva.reserva_ref
           OR consumo.clase IS DISTINCT FROM reserva.clase
           OR consumo.baremacion_merito_ref IS DISTINCT FROM
              reserva.baremacion_merito_ref
           OR consumo.huella_token_sha256 IS DISTINCT FROM
              p_operacion ->> 'huella_token_sha256'
           OR consumo.principal_ref IS DISTINCT FROM reserva.principal_ref
           OR consumo.sujeto_ref IS DISTINCT FROM reserva.sujeto_ref
           OR consumo.finalidad_clave IS DISTINCT FROM
              autorizacion.decision_canonica ->> 'finalidad' THEN
            resultado := 'autorizacion_reutilizada';
            RETURN NEXT;
            RETURN;
        END IF;
        IF reserva.estado = 'activa' THEN
            numero_resultado := consumo.numero_version;
            huella_resultado := consumo.huella_estado_sha256;
            huella_prevalidacion_sha256 :=
                consumo.huella_prevalidacion_sha256;
        ELSIF consumo.estado_resultado = 'confirmada' THEN
            numero_resultado := consumo.numero_version;
            huella_resultado := consumo.huella_estado_sha256;
            huella_prevalidacion_sha256 :=
                consumo.huella_prevalidacion_sha256;
        ELSE
            SELECT * INTO resultado_final
              FROM vec_bolsa_baremacion.resultado_prevalidacion_archivo_v3
             WHERE autorizacion_prevalidacion_ref =
                   consumo.autorizacion_prevalidacion_ref;
            IF NOT FOUND OR resultado_final.numero_version IS DISTINCT FROM
                  reserva.numero_version_confirmada
               OR resultado_final.huella_confirmacion_sha256 IS DISTINCT FROM
                  reserva.huella_confirmacion_sha256 THEN
                resultado := 'evidencia_no_confiable';
                RETURN NEXT;
                RETURN;
            END IF;
            numero_resultado := resultado_final.numero_version;
            huella_resultado := resultado_final.huella_estado_sha256;
            huella_prevalidacion_sha256 :=
                resultado_final.huella_prevalidacion_sha256;
        END IF;
    ELSE
        IF reserva.estado = 'activa' THEN
            numero_resultado := numero_solicitado;
            huella_resultado :=
                p_operacion ->> 'huella_version_esperada_sha256';
        ELSE
            numero_resultado := reserva.numero_version_confirmada;
            SELECT version.huella_estado_sha256
              INTO huella_resultado
              FROM vec_bolsa_baremacion.version_baremacion AS version
             WHERE version.baremacion_merito_ref =
                   reserva.baremacion_merito_ref
               AND version.numero = numero_resultado;
            IF NOT FOUND THEN
                resultado := 'evidencia_no_confiable';
                RETURN NEXT;
                RETURN;
            END IF;
        END IF;
    END IF;

    SELECT version.numero, version.huella_estado_sha256,
           version.agregado_canonico, version.confirmada_en,
           version.sujeto_ref
      INTO version_resultado
      FROM vec_bolsa_baremacion.version_baremacion AS version
     WHERE version.baremacion_merito_ref = reserva.baremacion_merito_ref
       AND version.numero = numero_resultado;
    IF NOT FOUND OR version_resultado.sujeto_ref IS DISTINCT FROM
          reserva.sujeto_ref
       OR version_resultado.huella_estado_sha256 IS DISTINCT FROM
          huella_resultado THEN
        resultado := 'evidencia_no_confiable';
        RETURN NEXT;
        RETURN;
    END IF;
    agregado_canonico := version_resultado.agregado_canonico;
    confirmada_en := version_resultado.confirmada_en;
    huella_estado_sha256 := version_resultado.huella_estado_sha256;
    archivo_probatorio_documento :=
        vec_bolsa_baremacion.construir_archivo_probatorio_v3(
            reserva.baremacion_merito_ref, numero_resultado
        );
    IF archivo_probatorio_documento IS NULL THEN
        resultado := 'evidencia_no_confiable';
        RETURN NEXT;
        RETURN;
    END IF;
    huella_archivo :=
        archivo_probatorio_documento ->> 'huella_archivo_sha256';
    huella_calculada :=
        vec_bolsa_baremacion.huella_prevalidacion_archivo_v3(
            reserva.baremacion_merito_ref, numero_resultado,
            huella_resultado, p_operacion ->> 'huella_token_sha256',
            reserva.principal_ref, reserva.sujeto_ref,
            autorizacion.decision_canonica ->> 'finalidad',
            p_prueba ->> 'decision_ref',
            p_operacion ->> 'huella_confirmacion_sha256'
        );
    IF huella_calculada IS NULL THEN
        resultado := 'evidencia_no_confiable';
        RETURN NEXT;
        RETURN;
    END IF;

    IF uso.decision_ref IS NULL THEN
        INSERT INTO vec_bolsa_baremacion.uso_decision (
            decision_ref, esquema_huella_decision,
            huella_decision_sha256, huella_efecto_sha256, tipo_efecto,
            resultado_ref, atestacion_ref, atestacion_version, consumida_en
        ) VALUES (
            p_prueba ->> 'decision_ref', p_prueba ->> 'esquema_huella',
            p_prueba ->> 'huella_decision_sha256',
            p_operacion ->> 'huella_efecto_prevalidacion_sha256',
            'prevalidacion_archivo', p_prueba ->> 'decision_ref',
            autorizacion.atestacion_ref, autorizacion.atestacion_version,
            instante
        );
        INSERT INTO
            vec_bolsa_baremacion.prevalidacion_archivo_probatorio_v3 (
                autorizacion_prevalidacion_ref,
                huella_prevalidacion_sha256, reserva_ref, clase,
                estado_resultado, baremacion_merito_ref, numero_version,
                huella_estado_sha256, huella_token_sha256, principal_ref,
                huella_confirmacion_sha256,
                huella_efecto_prevalidacion_sha256,
                sujeto_ref, finalidad_clave, total_manifiestos,
                huella_archivo_sha256, consumida_en
            ) VALUES (
                p_prueba ->> 'decision_ref', huella_calculada,
                reserva.reserva_ref, reserva.clase, reserva.estado,
                reserva.baremacion_merito_ref, numero_resultado,
                huella_resultado, p_operacion ->> 'huella_token_sha256',
                reserva.principal_ref,
                p_operacion ->> 'huella_confirmacion_sha256',
                p_operacion ->> 'huella_efecto_prevalidacion_sha256',
                reserva.sujeto_ref,
                autorizacion.decision_canonica ->> 'finalidad',
                greatest(numero_resultado::integer - 1, 0),
                huella_archivo, instante
            );
        huella_prevalidacion_sha256 := huella_calculada;
    ELSIF huella_prevalidacion_sha256 IS DISTINCT FROM huella_calculada THEN
        resultado := 'evidencia_no_confiable';
        RETURN NEXT;
        RETURN;
    END IF;
    resultado := reserva.estado;
    numero_version := numero_resultado::text;
    RETURN NEXT;
EXCEPTION
    WHEN unique_violation THEN
        resultado := 'autorizacion_reutilizada';
        numero_version := '';
        huella_estado_sha256 := '';
        agregado_canonico := NULL;
        confirmada_en := NULL;
        archivo_probatorio_documento := NULL;
        huella_prevalidacion_sha256 := '';
        RETURN NEXT;
    WHEN data_exception OR invalid_text_representation
        OR numeric_value_out_of_range OR datetime_field_overflow
        OR check_violation OR foreign_key_violation
        OR no_data_found OR too_many_rows THEN
        resultado := 'rechazada';
        numero_version := '';
        huella_estado_sha256 := '';
        agregado_canonico := NULL;
        confirmada_en := NULL;
        archivo_probatorio_documento := NULL;
        huella_prevalidacion_sha256 := '';
        RETURN NEXT;
END
$funcion$;

CREATE FUNCTION vec_bolsa_baremacion.reservar_cambio_con_archivo_probatorio_v3(
    p_operacion jsonb,
    p_prueba jsonb,
    p_decision_canonica bytea,
    p_recurso_canonico bytea
)
RETURNS TABLE (
    resultado text,
    reserva_ref text,
    expira_en timestamptz,
    numero_version text,
    huella_estado_sha256 text,
    agregado_canonico bytea,
    confirmada_en timestamptz,
    auditoria_ref text,
    huella_auditoria_sha256 text,
    evento_outbox_ref text,
    huella_evento_outbox_sha256 text,
    archivo_probatorio_documento jsonb
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    operacion_v1 jsonb;
    respuesta record;
    numero_archivo numeric(20, 0);
BEGIN
    resultado := 'rechazada';
    reserva_ref := '';
    numero_version := '';
    huella_estado_sha256 := '';
    auditoria_ref := '';
    huella_auditoria_sha256 := '';
    evento_outbox_ref := '';
    huella_evento_outbox_sha256 := '';
    IF p_operacion IS NULL OR jsonb_typeof(p_operacion) <> 'object'
       OR p_operacion ->> 'esquema' <>
          'vec.bolsa.baremacion.reserva-postgresql.v3'
       OR vec_bolsa_baremacion.referencia_manifiesto_v3_valida(
           p_operacion ->> 'reserva_ref', 512
       ) IS NOT TRUE
       OR vec_bolsa_baremacion.referencia_manifiesto_v3_valida(
           p_operacion ->> 'baremacion_merito_ref', 512
       ) IS NOT TRUE
       OR vec_bolsa_baremacion.sello_hmac_manifiesto_v3_valido(
           p_operacion ->> 'huella_solicitud_hmac'
       ) IS NOT TRUE THEN
        RETURN NEXT;
        RETURN;
    END IF;
    operacion_v1 := jsonb_set(
        p_operacion, '{esquema}',
        to_jsonb('vec.bolsa.baremacion.reserva-postgresql.v1'::text), false
    );
    SELECT * INTO respuesta
      FROM vec_bolsa_baremacion.reservar_cambio(
          operacion_v1, p_prueba, p_decision_canonica,
          p_recurso_canonico
      );
    resultado := respuesta.resultado;
    reserva_ref := respuesta.reserva_ref;
    expira_en := respuesta.expira_en;
    numero_version := respuesta.numero_version;
    huella_estado_sha256 := respuesta.huella_estado_sha256;
    agregado_canonico := respuesta.agregado_canonico;
    confirmada_en := respuesta.confirmada_en;
    auditoria_ref := respuesta.auditoria_ref;
    huella_auditoria_sha256 := respuesta.huella_auditoria_sha256;
    evento_outbox_ref := respuesta.evento_outbox_ref;
    huella_evento_outbox_sha256 :=
        respuesta.huella_evento_outbox_sha256;
    IF resultado <> 'confirmada' THEN
        archivo_probatorio_documento := NULL;
        RETURN NEXT;
        RETURN;
    END IF;
    numero_archivo := numero_version::numeric;
    archivo_probatorio_documento :=
        vec_bolsa_baremacion.construir_archivo_probatorio_v3(
            p_operacion ->> 'baremacion_merito_ref', numero_archivo
        );
    IF archivo_probatorio_documento IS NULL THEN
        resultado := 'evidencia_no_confiable';
        reserva_ref := '';
        expira_en := NULL;
        numero_version := '';
        huella_estado_sha256 := '';
        agregado_canonico := NULL;
        confirmada_en := NULL;
        auditoria_ref := '';
        huella_auditoria_sha256 := '';
        evento_outbox_ref := '';
        huella_evento_outbox_sha256 := '';
    END IF;
    RETURN NEXT;
EXCEPTION
    WHEN data_exception OR invalid_text_representation
        OR numeric_value_out_of_range OR no_data_found OR too_many_rows THEN
        resultado := 'rechazada';
        reserva_ref := '';
        expira_en := NULL;
        numero_version := '';
        huella_estado_sha256 := '';
        agregado_canonico := NULL;
        confirmada_en := NULL;
        auditoria_ref := '';
        huella_auditoria_sha256 := '';
        evento_outbox_ref := '';
        huella_evento_outbox_sha256 := '';
        archivo_probatorio_documento := NULL;
        RETURN NEXT;
END
$funcion$;


CREATE FUNCTION vec_bolsa_baremacion.archivo_unitario_manifiesto_v3_valido(
    p_manifiesto jsonb,
    p_contenido bytea,
    p_representacion bytea,
    p_preimagen_hmac bytea
)
RETURNS boolean
LANGUAGE plpgsql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    contenido_calculado bytea;
    material_con_huella bytea;
    representacion_calculada bytea;
    preimagen_calculada bytea;
    finalidad_constante constant text :=
        'manifiesto_probatorio_baremacion_v3';
BEGIN
    IF p_manifiesto IS NULL OR p_contenido IS NULL
       OR p_representacion IS NULL OR p_preimagen_hmac IS NULL THEN
        RETURN false;
    END IF;
    contenido_calculado :=
        vec_bolsa_baremacion.contenido_manifiesto_probatorio_v3(
            p_manifiesto
        );
    IF contenido_calculado IS NULL
       OR contenido_calculado IS DISTINCT FROM p_contenido
       OR encode(sha256(contenido_calculado), 'hex') IS DISTINCT FROM
          p_manifiesto ->> 'huella_manifiesto_sha256' THEN
        RETURN false;
    END IF;
    material_con_huella := contenido_calculado ||
        vec_bolsa_baremacion.parte_canonica_manifiesto_v3(
            p_manifiesto ->> 'huella_manifiesto_sha256'
        );
    representacion_calculada :=
        vec_bolsa_baremacion.parte_canonica_manifiesto_v3(
            finalidad_constante
        ) || int8send(octet_length(material_con_huella)::bigint)
          || material_con_huella;
    preimagen_calculada := convert_to(finalidad_constante, 'UTF8')
        || decode('00', 'hex') || representacion_calculada;
    RETURN representacion_calculada = p_representacion
       AND preimagen_calculada = p_preimagen_hmac
       AND octet_length(p_contenido) BETWEEN 1 AND 16777216
       AND octet_length(p_representacion) BETWEEN 1 AND 16777344
       AND octet_length(p_preimagen_hmac) BETWEEN 1 AND 16777408;
EXCEPTION
    WHEN data_exception OR invalid_text_representation
        OR numeric_value_out_of_range OR character_not_in_repertoire
        OR untranslatable_character THEN
        RETURN false;
END
$funcion$;

CREATE TABLE vec_bolsa_baremacion.manifiesto_probatorio_v3 (
    referencia text PRIMARY KEY,
    baremacion_merito_ref text NOT NULL,
    numero_version numeric(20, 0) NOT NULL,
    decision_ref text NOT NULL UNIQUE,
    version_base numeric(20, 0) NOT NULL,
    huella_version_base_sha256 text NOT NULL,
    auditoria_ref text NOT NULL UNIQUE,
    evento_outbox_ref text NOT NULL UNIQUE,
    reserva_ref text NOT NULL UNIQUE,
    total_autorizaciones integer NOT NULL,
    total_evidencias integer NOT NULL,
    manifiesto jsonb NOT NULL,
    contenido_manifiesto_canonico bytea NOT NULL,
    representacion_manifiesto_canonica bytea NOT NULL,
    preimagen_hmac_manifiesto bytea NOT NULL,
    huella_manifiesto_sha256 text NOT NULL,
    sello_manifiesto_hmac_sha256 text NOT NULL,
    registrado_en timestamptz(6) NOT NULL,
    UNIQUE (baremacion_merito_ref, numero_version),
    UNIQUE (baremacion_merito_ref, numero_version, referencia),
    FOREIGN KEY (baremacion_merito_ref, numero_version)
        REFERENCES vec_bolsa_baremacion.version_baremacion(
            baremacion_merito_ref, numero
        ),
    FOREIGN KEY (auditoria_ref)
        REFERENCES vec_bolsa_baremacion.auditoria(referencia),
    FOREIGN KEY (evento_outbox_ref)
        REFERENCES vec_bolsa_baremacion.evento_outbox(referencia),
    FOREIGN KEY (reserva_ref)
        REFERENCES vec_bolsa_baremacion.token_reserva(reserva_ref),
    CONSTRAINT manifiesto_probatorio_v3_rango CHECK (
        numero_version BETWEEN 2 AND 4097
        AND version_base = numero_version - 1
        AND total_autorizaciones BETWEEN 1 AND 4096
        AND total_evidencias BETWEEN 1 AND 4096
    ),
    CONSTRAINT manifiesto_probatorio_v3_identidad CHECK (
        vec_bolsa_baremacion.referencia_manifiesto_v3_valida(
            referencia, 512
        )
        AND vec_bolsa_baremacion.referencia_manifiesto_v3_valida(
            baremacion_merito_ref, 512
        )
        AND vec_bolsa_baremacion.referencia_manifiesto_v3_valida(
            decision_ref, 512
        )
        AND vec_bolsa_baremacion.referencia_manifiesto_v3_valida(
            auditoria_ref, 512
        )
        AND vec_bolsa_baremacion.referencia_manifiesto_v3_valida(
            evento_outbox_ref, 512
        )
        AND vec_bolsa_baremacion.referencia_manifiesto_v3_valida(
            reserva_ref, 512
        )
        AND referencia = manifiesto ->> 'referencia'
        AND baremacion_merito_ref = manifiesto ->> 'baremacion_merito_ref'
        AND decision_ref = manifiesto ->> 'decision_ref'
        AND version_base::text = manifiesto ->> 'version_base'
        AND huella_version_base_sha256 =
            manifiesto ->> 'huella_version_base_sha256'
        AND total_autorizaciones =
            jsonb_array_length(manifiesto -> 'autorizaciones')
        AND total_evidencias = jsonb_array_length(manifiesto -> 'evidencias')
        AND huella_manifiesto_sha256 =
            manifiesto ->> 'huella_manifiesto_sha256'
        AND sello_manifiesto_hmac_sha256 =
            manifiesto ->> 'sello_manifiesto_hmac_sha256'
        AND vec_bolsa_baremacion.archivo_unitario_manifiesto_v3_valido(
            manifiesto, contenido_manifiesto_canonico,
            representacion_manifiesto_canonica,
            preimagen_hmac_manifiesto
        )
    )
);

CREATE TABLE vec_bolsa_baremacion.manifiesto_autorizacion_v3 (
    manifiesto_ref text NOT NULL,
    secuencia integer NOT NULL,
    accion text NOT NULL,
    clase_recurso text NOT NULL,
    recurso_ref text NOT NULL,
    autorizacion_ref text NOT NULL,
    PRIMARY KEY (manifiesto_ref, secuencia),
    UNIQUE (manifiesto_ref, autorizacion_ref),
    FOREIGN KEY (manifiesto_ref)
        REFERENCES vec_bolsa_baremacion.manifiesto_probatorio_v3(referencia),
    CONSTRAINT manifiesto_autorizacion_v3_perfil CHECK (
        secuencia BETWEEN 1 AND 4096
        AND vec_bolsa_baremacion.accion_recurso_manifiesto_v3_valida(
            accion, clase_recurso
        )
        AND vec_bolsa_baremacion.referencia_manifiesto_v3_valida(
            recurso_ref, 512
        )
        AND vec_bolsa_baremacion.referencia_manifiesto_v3_valida(
            autorizacion_ref, 512
        )
    )
);

CREATE TABLE vec_bolsa_baremacion.manifiesto_evidencia_v3 (
    manifiesto_ref text NOT NULL,
    secuencia integer NOT NULL,
    tipo text NOT NULL,
    referencia text NOT NULL,
    huella_evidencia_sha256 text NOT NULL,
    PRIMARY KEY (manifiesto_ref, secuencia),
    UNIQUE (manifiesto_ref, tipo, referencia),
    FOREIGN KEY (manifiesto_ref)
        REFERENCES vec_bolsa_baremacion.manifiesto_probatorio_v3(referencia),
    CONSTRAINT manifiesto_evidencia_v3_perfil CHECK (
        secuencia BETWEEN 1 AND 4096
        AND vec_bolsa_baremacion.tipo_evidencia_manifiesto_v3_valido(tipo)
        AND vec_bolsa_baremacion.referencia_manifiesto_v3_valida(
            referencia, 512
        )
        AND vec_bolsa_baremacion.huella_sha256_valida(
            huella_evidencia_sha256
        )
    )
);

-- Consumo durable que demuestra qué autorización dedicada prevalidó qué
-- snapshot. No es un recibo KMS: solo liga hechos durables y OCC.
CREATE TABLE vec_bolsa_baremacion.prevalidacion_archivo_probatorio_v3 (
    autorizacion_prevalidacion_ref text PRIMARY KEY,
    huella_prevalidacion_sha256 text NOT NULL UNIQUE,
    reserva_ref text NOT NULL,
    clase text NOT NULL,
    estado_resultado text NOT NULL,
    baremacion_merito_ref text NOT NULL,
    numero_version numeric(20, 0) NOT NULL,
    huella_estado_sha256 text NOT NULL,
    huella_token_sha256 text NOT NULL,
    huella_confirmacion_sha256 text NOT NULL,
    huella_efecto_prevalidacion_sha256 text NOT NULL,
    principal_ref text NOT NULL,
    sujeto_ref text NOT NULL,
    finalidad_clave text NOT NULL,
    total_manifiestos integer NOT NULL,
    huella_archivo_sha256 text NOT NULL,
    consumida_en timestamptz(6) NOT NULL,
    FOREIGN KEY (autorizacion_prevalidacion_ref)
        REFERENCES vec_bolsa_baremacion.uso_decision(decision_ref),
    FOREIGN KEY (huella_token_sha256)
        REFERENCES vec_bolsa_baremacion.token_reserva(huella_token_sha256),
    CONSTRAINT prevalidacion_archivo_probatorio_v3_perfil CHECK (
        numero_version BETWEEN 1 AND 4097
        AND clase = 'incorporar_decision'
        AND estado_resultado IN ('activa', 'confirmada')
        AND total_manifiestos = numero_version::integer - 1
        AND vec_bolsa_baremacion.huella_sha256_valida(
            huella_estado_sha256
        )
        AND vec_bolsa_baremacion.huella_sha256_valida(
            huella_prevalidacion_sha256
        )
        AND vec_bolsa_baremacion.huella_sha256_valida(
            huella_token_sha256
        )
        AND vec_bolsa_baremacion.huella_sha256_valida(
            huella_confirmacion_sha256
        )
        AND vec_bolsa_baremacion.huella_sha256_valida(
            huella_efecto_prevalidacion_sha256
        )
        AND vec_bolsa_baremacion.huella_sha256_valida(
            huella_archivo_sha256
        )
        AND vec_bolsa_baremacion.referencia_manifiesto_v3_valida(
            baremacion_merito_ref, 512
        )
        AND vec_bolsa_baremacion.referencia_manifiesto_v3_valida(
            reserva_ref, 512
        )
        AND vec_bolsa_baremacion.referencia_manifiesto_v3_valida(
            principal_ref, 512
        )
        AND vec_bolsa_baremacion.referencia_manifiesto_v3_valida(
            sujeto_ref, 512
        )
        AND vec_bolsa_baremacion.referencia_manifiesto_v3_valida(
            finalidad_clave, 512
        )
    )
);

-- Resultado final append-only asociado al consumo anterior. Permite recuperar
-- una confirmacion cuya respuesta se perdio sin reusar la autorizacion para
-- producir un efecto distinto.
CREATE TABLE vec_bolsa_baremacion.resultado_prevalidacion_archivo_v3 (
    autorizacion_prevalidacion_ref text PRIMARY KEY,
    huella_prevalidacion_sha256 text NOT NULL UNIQUE,
    numero_version numeric(20, 0) NOT NULL,
    huella_estado_sha256 text NOT NULL,
    total_manifiestos integer NOT NULL,
    huella_archivo_sha256 text NOT NULL,
    huella_confirmacion_sha256 text NOT NULL,
    registrada_en timestamptz(6) NOT NULL,
    FOREIGN KEY (autorizacion_prevalidacion_ref)
        REFERENCES vec_bolsa_baremacion.prevalidacion_archivo_probatorio_v3(
            autorizacion_prevalidacion_ref
        ),
    CONSTRAINT resultado_prevalidacion_archivo_v3_perfil CHECK (
        numero_version BETWEEN 1 AND 4097
        AND total_manifiestos = numero_version::integer - 1
        AND vec_bolsa_baremacion.huella_sha256_valida(
            huella_prevalidacion_sha256
        )
        AND vec_bolsa_baremacion.huella_sha256_valida(
            huella_estado_sha256
        )
        AND vec_bolsa_baremacion.huella_sha256_valida(
            huella_archivo_sha256
        )
        AND vec_bolsa_baremacion.huella_sha256_valida(
            huella_confirmacion_sha256
        )
    )
);

CREATE FUNCTION vec_bolsa_baremacion.validar_cardinalidad_manifiesto_v3()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    autorizaciones bigint;
    evidencias bigint;
BEGIN
    SELECT count(*)
      INTO autorizaciones
      FROM vec_bolsa_baremacion.manifiesto_autorizacion_v3
     WHERE manifiesto_ref = NEW.referencia;
    SELECT count(*)
      INTO evidencias
      FROM vec_bolsa_baremacion.manifiesto_evidencia_v3
     WHERE manifiesto_ref = NEW.referencia;
    IF autorizaciones IS DISTINCT FROM NEW.total_autorizaciones::bigint
       OR evidencias IS DISTINCT FROM NEW.total_evidencias::bigint THEN
        RAISE EXCEPTION USING ERRCODE = '23514',
            MESSAGE = 'manifiesto V3 incompleto al confirmar la transaccion';
    END IF;
    RETURN NULL;
END
$funcion$;

-- Cada INSERT SELECT de hijos se valida como conjunto. La transition table y
-- la expansion MATERIALIZED detoastan cada manifiesto una sola vez.
CREATE FUNCTION vec_bolsa_baremacion.validar_autorizaciones_manifiesto_v3()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, pg_temp
AS $funcion$
BEGIN
    IF EXISTS (
        SELECT 1
          FROM filas_nuevas AS nueva
          LEFT JOIN vec_bolsa_baremacion.manifiesto_probatorio_v3 AS cabecera
            ON cabecera.referencia = nueva.manifiesto_ref
         WHERE cabecera.referencia IS NULL
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '23503',
            MESSAGE = 'falta la cabecera del manifiesto V3';
    END IF;
    IF EXISTS (
        SELECT 1
          FROM filas_nuevas AS nueva
          JOIN vec_bolsa_baremacion.manifiesto_probatorio_v3 AS cabecera
            ON cabecera.referencia = nueva.manifiesto_ref
         WHERE nueva.secuencia < 1
            OR nueva.secuencia > cabecera.total_autorizaciones
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '23514',
            MESSAGE = 'secuencia de autorizacion fuera del manifiesto V3';
    END IF;
    IF EXISTS (
        WITH referencias AS MATERIALIZED (
            SELECT DISTINCT nueva.manifiesto_ref
              FROM filas_nuevas AS nueva
        ), cabeceras AS MATERIALIZED (
            SELECT cabecera.referencia, cabecera.manifiesto
              FROM referencias
              JOIN vec_bolsa_baremacion.manifiesto_probatorio_v3 AS cabecera
                ON cabecera.referencia = referencias.manifiesto_ref
        ), esperadas AS MATERIALIZED (
            SELECT cabecera.referencia AS manifiesto_ref,
                   lista.ordinalidad::integer AS secuencia,
                   lista.elemento
              FROM cabeceras AS cabecera
              CROSS JOIN LATERAL jsonb_array_elements(
                  cabecera.manifiesto -> 'autorizaciones'
              ) WITH ORDINALITY AS lista(elemento, ordinalidad)
        )
        SELECT 1
          FROM filas_nuevas AS nueva
          LEFT JOIN esperadas
            ON esperadas.manifiesto_ref = nueva.manifiesto_ref
           AND esperadas.secuencia = nueva.secuencia
         WHERE esperadas.elemento IS NULL
            OR esperadas.elemento IS DISTINCT FROM jsonb_build_object(
                'secuencia', nueva.secuencia,
                'accion', nueva.accion,
                'clase_recurso', nueva.clase_recurso,
                'recurso_ref', nueva.recurso_ref,
                'autorizacion_ref', nueva.autorizacion_ref
            )
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '23514',
            MESSAGE = 'autorizacion divergente del manifiesto V3';
    END IF;
    RETURN NULL;
END
$funcion$;

CREATE FUNCTION vec_bolsa_baremacion.validar_evidencias_manifiesto_v3()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, pg_temp
AS $funcion$
BEGIN
    IF EXISTS (
        SELECT 1
          FROM filas_nuevas AS nueva
          LEFT JOIN vec_bolsa_baremacion.manifiesto_probatorio_v3 AS cabecera
            ON cabecera.referencia = nueva.manifiesto_ref
         WHERE cabecera.referencia IS NULL
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '23503',
            MESSAGE = 'falta la cabecera del manifiesto V3';
    END IF;
    IF EXISTS (
        SELECT 1
          FROM filas_nuevas AS nueva
          JOIN vec_bolsa_baremacion.manifiesto_probatorio_v3 AS cabecera
            ON cabecera.referencia = nueva.manifiesto_ref
         WHERE nueva.secuencia < 1
            OR nueva.secuencia > cabecera.total_evidencias
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '23514',
            MESSAGE = 'secuencia de evidencia fuera del manifiesto V3';
    END IF;
    IF EXISTS (
        WITH referencias AS MATERIALIZED (
            SELECT DISTINCT nueva.manifiesto_ref
              FROM filas_nuevas AS nueva
        ), cabeceras AS MATERIALIZED (
            SELECT cabecera.referencia, cabecera.manifiesto
              FROM referencias
              JOIN vec_bolsa_baremacion.manifiesto_probatorio_v3 AS cabecera
                ON cabecera.referencia = referencias.manifiesto_ref
        ), esperadas AS MATERIALIZED (
            SELECT cabecera.referencia AS manifiesto_ref,
                   lista.ordinalidad::integer AS secuencia,
                   lista.elemento
              FROM cabeceras AS cabecera
              CROSS JOIN LATERAL jsonb_array_elements(
                  cabecera.manifiesto -> 'evidencias'
              ) WITH ORDINALITY AS lista(elemento, ordinalidad)
        )
        SELECT 1
          FROM filas_nuevas AS nueva
          LEFT JOIN esperadas
            ON esperadas.manifiesto_ref = nueva.manifiesto_ref
           AND esperadas.secuencia = nueva.secuencia
         WHERE esperadas.elemento IS NULL
            OR esperadas.elemento IS DISTINCT FROM jsonb_build_object(
                'secuencia', nueva.secuencia,
                'tipo', nueva.tipo,
                'referencia', nueva.referencia,
                'huella_evidencia_sha256', nueva.huella_evidencia_sha256
            )
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '23514',
            MESSAGE = 'evidencia divergente del manifiesto V3';
    END IF;
    RETURN NULL;
END
$funcion$;

CREATE TRIGGER manifiesto_autorizacion_v3_correspondencia
    AFTER INSERT ON vec_bolsa_baremacion.manifiesto_autorizacion_v3
    REFERENCING NEW TABLE AS filas_nuevas
    FOR EACH STATEMENT EXECUTE FUNCTION
        vec_bolsa_baremacion.validar_autorizaciones_manifiesto_v3();
CREATE TRIGGER manifiesto_evidencia_v3_correspondencia
    AFTER INSERT ON vec_bolsa_baremacion.manifiesto_evidencia_v3
    REFERENCING NEW TABLE AS filas_nuevas
    FOR EACH STATEMENT EXECUTE FUNCTION
        vec_bolsa_baremacion.validar_evidencias_manifiesto_v3();

-- Un solo evento diferido por cabecera cuenta ambos hijos una vez. Los PK,
-- el rango inmediato y count=total demuestran que no hay huecos ni extras.
CREATE CONSTRAINT TRIGGER manifiesto_probatorio_v3_completitud
    AFTER INSERT ON vec_bolsa_baremacion.manifiesto_probatorio_v3
    DEFERRABLE INITIALLY DEFERRED
    FOR EACH ROW EXECUTE FUNCTION
        vec_bolsa_baremacion.validar_cardinalidad_manifiesto_v3();

DO $protecciones$
DECLARE
    tabla text;
BEGIN
    FOREACH tabla IN ARRAY ARRAY[
        'manifiesto_probatorio_v3', 'manifiesto_autorizacion_v3',
        'manifiesto_evidencia_v3',
        'prevalidacion_archivo_probatorio_v3',
        'resultado_prevalidacion_archivo_v3'
    ] LOOP
        EXECUTE format(
            'CREATE TRIGGER %I_inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_baremacion.%I FOR EACH ROW EXECUTE FUNCTION vec_bolsa_baremacion.rechazar_mutacion_inmutable()',
            tabla, tabla
        );
        EXECUTE format(
            'CREATE TRIGGER %I_no_truncar BEFORE TRUNCATE ON vec_bolsa_baremacion.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_bolsa_baremacion.rechazar_mutacion_inmutable()',
            tabla, tabla
        );
        EXECUTE format(
            'ALTER TABLE vec_bolsa_baremacion.%I ENABLE ROW LEVEL SECURITY',
            tabla
        );
        EXECUTE format(
            'ALTER TABLE vec_bolsa_baremacion.%I FORCE ROW LEVEL SECURITY',
            tabla
        );
        EXECUTE format(
            'CREATE POLICY acceso_propietario_exacto ON vec_bolsa_baremacion.%I FOR ALL TO vec_bolsa_baremacion_propietario USING (current_user = %L) WITH CHECK (current_user = %L)',
            tabla, 'vec_bolsa_baremacion_propietario',
            'vec_bolsa_baremacion_propietario'
        );
    END LOOP;
END
$protecciones$;

-- Ensambla el unico documento que cruza el puerto PostgreSQL. Cada elemento
-- lleva JSON estructurado y los tres artefactos exactos en hexadecimal. La
-- cardinalidad es N-1 para una version confirmada N; V1 produce [].
CREATE FUNCTION vec_bolsa_baremacion.construir_archivo_probatorio_v3(
    p_baremacion_merito_ref text,
    p_numero_version numeric
)
RETURNS jsonb
LANGUAGE plpgsql
STABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    esquema_constante constant text :=
        'vec.bolsa.archivo_probatorio-postgresql.v3';
    limite_archivo constant bigint := 67108864;
    manifiestos jsonb := '[]'::jsonb;
    total bigint;
    total_esperado bigint;
    tamano_conservador numeric;
    elemento record;
    material_huella bytea;
    total_fragmentos bigint;
    documento jsonb;
    huella_archivo text;
BEGIN
    IF vec_bolsa_baremacion.referencia_manifiesto_v3_valida(
           p_baremacion_merito_ref, 512
       ) IS NOT TRUE
       OR p_numero_version NOT BETWEEN 0 AND 4097 THEN
        RETURN NULL;
    END IF;
    IF p_numero_version > 0 AND NOT EXISTS (
        SELECT 1
          FROM vec_bolsa_baremacion.version_baremacion
         WHERE baremacion_merito_ref = p_baremacion_merito_ref
           AND numero = p_numero_version
    ) THEN
        RETURN NULL;
    END IF;
    total_esperado := greatest(p_numero_version::bigint - 1, 0);
    -- El hexadecimal duplica cada byte. Se suma ademas el JSON estructurado y
    -- 4 KiB conservadores por elemento antes de construir jsonb_agg/arrays.
    SELECT count(*), COALESCE(sum(
               octet_length(almacenado.manifiesto::text)::numeric
               + 2 * octet_length(almacenado.contenido_manifiesto_canonico)
               + 2 * octet_length(
                   almacenado.representacion_manifiesto_canonica
                 )
               + 2 * octet_length(almacenado.preimagen_hmac_manifiesto)
               + 4096
           ), 0)
      INTO total, tamano_conservador
      FROM vec_bolsa_baremacion.manifiesto_probatorio_v3 AS almacenado
     WHERE almacenado.baremacion_merito_ref = p_baremacion_merito_ref
       AND almacenado.numero_version <= p_numero_version;
    IF total <> total_esperado OR tamano_conservador > limite_archivo
       OR EXISTS (
        SELECT 1
          FROM generate_series(2, p_numero_version::integer) AS esperada(numero)
          LEFT JOIN vec_bolsa_baremacion.manifiesto_probatorio_v3 AS almacenado
            ON almacenado.baremacion_merito_ref = p_baremacion_merito_ref
           AND almacenado.numero_version = esperada.numero
         WHERE almacenado.referencia IS NULL
            OR almacenado.version_base <> esperada.numero - 1
    ) THEN
        RETURN NULL;
    END IF;
    SELECT COALESCE(jsonb_agg(jsonb_build_object(
               'manifiesto', almacenado.manifiesto,
               'contenido_manifiesto_canonico_hex',
                   encode(almacenado.contenido_manifiesto_canonico, 'hex'),
               'representacion_manifiesto_canonica_hex',
                   encode(almacenado.representacion_manifiesto_canonica, 'hex'),
               'preimagen_hmac_manifiesto_hex',
                   encode(almacenado.preimagen_hmac_manifiesto, 'hex')
           ) ORDER BY almacenado.numero_version), '[]'::jsonb)
      INTO manifiestos
      FROM vec_bolsa_baremacion.manifiesto_probatorio_v3 AS almacenado
     WHERE almacenado.baremacion_merito_ref = p_baremacion_merito_ref
       AND almacenado.numero_version <= p_numero_version;
    FOR elemento IN
        SELECT *
          FROM vec_bolsa_baremacion.manifiesto_probatorio_v3
         WHERE baremacion_merito_ref = p_baremacion_merito_ref
           AND numero_version <= p_numero_version
         ORDER BY numero_version
    LOOP
        IF vec_bolsa_baremacion.archivo_unitario_manifiesto_v3_valido(
            elemento.manifiesto, elemento.contenido_manifiesto_canonico,
            elemento.representacion_manifiesto_canonica,
            elemento.preimagen_hmac_manifiesto
        ) IS NOT TRUE THEN
            RETURN NULL;
        END IF;
    END LOOP;
    WITH fragmentos(grupo, numero, campo, fragmento) AS (
        SELECT 0, 0::numeric, fijo.campo,
               vec_bolsa_baremacion.parte_canonica_manifiesto_v3(
                   fijo.valor
               )
          FROM (VALUES
            (1, esquema_constante),
            (2, p_baremacion_merito_ref),
            (3, p_numero_version::text),
            (4, total_esperado::text)
          ) AS fijo(campo, valor)
        UNION ALL
        SELECT 1, almacenado.numero_version, parte.campo, parte.fragmento
          FROM vec_bolsa_baremacion.manifiesto_probatorio_v3 AS almacenado
          CROSS JOIN LATERAL (VALUES
            (1, int8send(octet_length(
                    almacenado.contenido_manifiesto_canonico
                )::bigint) || almacenado.contenido_manifiesto_canonico),
            (2, int8send(octet_length(
                    almacenado.representacion_manifiesto_canonica
                )::bigint) || almacenado.representacion_manifiesto_canonica),
            (3, int8send(octet_length(
                    almacenado.preimagen_hmac_manifiesto
                )::bigint) || almacenado.preimagen_hmac_manifiesto),
            (4, vec_bolsa_baremacion.parte_canonica_manifiesto_v3(
                    almacenado.sello_manifiesto_hmac_sha256
                ))
          ) AS parte(campo, fragmento)
         WHERE almacenado.baremacion_merito_ref = p_baremacion_merito_ref
           AND almacenado.numero_version <= p_numero_version
    )
    SELECT count(fragmento), string_agg(
               fragmento, ''::bytea ORDER BY grupo, numero, campo
           )
      INTO total_fragmentos, material_huella
      FROM fragmentos;
    IF total_fragmentos <> 4 + 4 * total_esperado THEN
        RETURN NULL;
    END IF;
    huella_archivo := encode(sha256(material_huella), 'hex');
    documento := jsonb_build_object(
        'esquema', esquema_constante,
        'baremacion_merito_ref', p_baremacion_merito_ref,
        'numero_version', p_numero_version::text,
        'huella_archivo_sha256', huella_archivo,
        'manifiestos', manifiestos
    );
    IF octet_length(documento::text) > limite_archivo THEN
        RETURN NULL;
    END IF;
    RETURN documento;
END
$funcion$;

-- Huella portable acordada para el consumo de prevalidacion. Usa la misma
-- codificacion uint64be-longitud-UTF8 que huella_canonica y no incorpora JSON
-- textual dependiente de PostgreSQL.
CREATE FUNCTION vec_bolsa_baremacion.huella_prevalidacion_archivo_v3(
    p_baremacion_merito_ref text,
    p_numero_version numeric,
    p_huella_estado_sha256 text,
    p_huella_token_sha256 text,
    p_principal_ref text,
    p_sujeto_ref text,
    p_finalidad text,
    p_autorizacion_prevalidacion_ref text,
    p_huella_confirmacion_sha256 text
)
RETURNS text
LANGUAGE plpgsql
STABLE
SET search_path = pg_catalog, pg_temp
AS $funcion$
DECLARE
    total integer;
    total_real bigint;
    version_existe boolean;
    archivos_validos boolean;
    huella text;
BEGIN
    IF p_numero_version IS NULL
       OR p_numero_version NOT BETWEEN 0 AND 4097
       OR vec_bolsa_baremacion.huella_sha256_valida(
           p_huella_token_sha256
       ) IS NOT TRUE
       OR (p_numero_version = 0 AND p_huella_estado_sha256 <> '')
       OR (p_numero_version > 0 AND
           vec_bolsa_baremacion.huella_sha256_valida(
               p_huella_estado_sha256
           ) IS NOT TRUE)
       OR vec_bolsa_baremacion.referencia_manifiesto_v3_valida(
           p_baremacion_merito_ref, 512
       ) IS NOT TRUE
       OR vec_bolsa_baremacion.referencia_manifiesto_v3_valida(
           p_principal_ref, 512
       ) IS NOT TRUE
       OR vec_bolsa_baremacion.referencia_manifiesto_v3_valida(
           p_sujeto_ref, 512
       ) IS NOT TRUE
       OR vec_bolsa_baremacion.referencia_manifiesto_v3_valida(
           p_finalidad, 512
       ) IS NOT TRUE
       OR vec_bolsa_baremacion.referencia_manifiesto_v3_valida(
           p_autorizacion_prevalidacion_ref, 512
       ) IS NOT TRUE
       OR vec_bolsa_baremacion.huella_sha256_valida(
           p_huella_confirmacion_sha256
       ) IS NOT TRUE THEN
        RETURN NULL;
    END IF;
    total := greatest(p_numero_version::integer - 1, 0);
    SELECT p_numero_version = 0 OR EXISTS (
               SELECT 1
                 FROM vec_bolsa_baremacion.version_baremacion AS version
                WHERE version.baremacion_merito_ref =
                      p_baremacion_merito_ref
                  AND version.numero = p_numero_version
           ),
           count(*),
           COALESCE(bool_and(
               almacenado.version_base = almacenado.numero_version - 1
               AND vec_bolsa_baremacion.archivo_unitario_manifiesto_v3_valido(
                   almacenado.manifiesto,
                   almacenado.contenido_manifiesto_canonico,
                   almacenado.representacion_manifiesto_canonica,
                   almacenado.preimagen_hmac_manifiesto
               )
           ), true)
      INTO version_existe, total_real, archivos_validos
      FROM vec_bolsa_baremacion.manifiesto_probatorio_v3 AS almacenado
     WHERE almacenado.baremacion_merito_ref = p_baremacion_merito_ref
       AND almacenado.numero_version <= p_numero_version;
    IF version_existe IS NOT TRUE OR total_real <> total
       OR archivos_validos IS NOT TRUE OR EXISTS (
        SELECT 1
          FROM generate_series(2, p_numero_version::integer) AS esperada(numero)
          LEFT JOIN vec_bolsa_baremacion.manifiesto_probatorio_v3 AS almacenado
            ON almacenado.baremacion_merito_ref = p_baremacion_merito_ref
           AND almacenado.numero_version = esperada.numero
         WHERE almacenado.referencia IS NULL
    ) THEN
        RETURN NULL;
    END IF;
    WITH partes(grupo, numero, campo, valor) AS (
        SELECT 0, 0::numeric, fijo.campo, fijo.valor
          FROM (VALUES
            (1, 'prevalidacion-archivo-probatorio-baremacion-v3'),
            (2, p_baremacion_merito_ref),
            (3, p_numero_version::text),
            (4, p_huella_estado_sha256),
            (5, p_huella_token_sha256),
            (6, p_principal_ref),
            (7, p_sujeto_ref),
            (8, p_finalidad),
            (9, p_autorizacion_prevalidacion_ref),
            (10, p_huella_confirmacion_sha256),
            (11, total::text)
          ) AS fijo(campo, valor)
        UNION ALL
        SELECT 1, almacenado.numero_version, parte.campo, parte.valor
          FROM vec_bolsa_baremacion.manifiesto_probatorio_v3 AS almacenado
          CROSS JOIN LATERAL (VALUES
            (1, almacenado.referencia),
            (2, almacenado.huella_manifiesto_sha256),
            (3, almacenado.sello_manifiesto_hmac_sha256),
            (4, encode(sha256(
                    almacenado.contenido_manifiesto_canonico
                ), 'hex')),
            (5, encode(sha256(
                    almacenado.representacion_manifiesto_canonica
                ), 'hex')),
            (6, encode(sha256(
                    almacenado.preimagen_hmac_manifiesto
                ), 'hex'))
          ) AS parte(campo, valor)
         WHERE almacenado.baremacion_merito_ref = p_baremacion_merito_ref
           AND almacenado.numero_version <= p_numero_version
    ), material AS (
        SELECT count(valor) AS total_partes,
               string_agg(
                   int8send(octet_length(valor)::bigint)
                       || convert_to(valor, 'UTF8'),
                   ''::bytea ORDER BY grupo, numero, campo
               ) AS contenido
          FROM partes
    )
    SELECT CASE WHEN total_partes = 11 + 6 * total
                THEN encode(sha256(contenido), 'hex') END
      INTO huella
      FROM material;
    RETURN huella;
END
$funcion$;

CREATE FUNCTION vec_bolsa_baremacion.registrar_manifiesto_probatorio_v3(
    p_manifiesto jsonb,
    p_contenido bytea,
    p_representacion bytea,
    p_preimagen_hmac bytea,
    p_numero_version numeric,
    p_auditoria_ref text,
    p_evento_outbox_ref text,
    p_reserva_ref text,
    p_registrado_en timestamptz
)
RETURNS void
LANGUAGE plpgsql
VOLATILE
SET search_path = pg_catalog, pg_temp
AS $funcion$
BEGIN
    IF vec_bolsa_baremacion.archivo_unitario_manifiesto_v3_valido(
        p_manifiesto, p_contenido, p_representacion, p_preimagen_hmac
    ) IS NOT TRUE OR p_numero_version::text IS DISTINCT FROM
       ((p_manifiesto ->> 'version_base')::numeric + 1)::text THEN
        RAISE EXCEPTION USING ERRCODE = '23514',
            MESSAGE = 'archivo unitario de manifiesto V3 invalido';
    END IF;
    INSERT INTO vec_bolsa_baremacion.manifiesto_probatorio_v3 (
        referencia, baremacion_merito_ref, numero_version, decision_ref,
        version_base, huella_version_base_sha256, auditoria_ref,
        evento_outbox_ref, reserva_ref, total_autorizaciones,
        total_evidencias, manifiesto, contenido_manifiesto_canonico,
        representacion_manifiesto_canonica, preimagen_hmac_manifiesto,
        huella_manifiesto_sha256, sello_manifiesto_hmac_sha256,
        registrado_en
    ) VALUES (
        p_manifiesto ->> 'referencia',
        p_manifiesto ->> 'baremacion_merito_ref', p_numero_version,
        p_manifiesto ->> 'decision_ref',
        (p_manifiesto ->> 'version_base')::numeric,
        p_manifiesto ->> 'huella_version_base_sha256', p_auditoria_ref,
        p_evento_outbox_ref, p_reserva_ref,
        jsonb_array_length(p_manifiesto -> 'autorizaciones'),
        jsonb_array_length(p_manifiesto -> 'evidencias'), p_manifiesto,
        p_contenido, p_representacion, p_preimagen_hmac,
        p_manifiesto ->> 'huella_manifiesto_sha256',
        p_manifiesto ->> 'sello_manifiesto_hmac_sha256', p_registrado_en
    );
    INSERT INTO vec_bolsa_baremacion.manifiesto_autorizacion_v3 (
        manifiesto_ref, secuencia, accion, clase_recurso, recurso_ref,
        autorizacion_ref
    ) SELECT p_manifiesto ->> 'referencia', ordinalidad::integer,
             elemento ->> 'accion', elemento ->> 'clase_recurso',
             elemento ->> 'recurso_ref', elemento ->> 'autorizacion_ref'
        FROM jsonb_array_elements(p_manifiesto -> 'autorizaciones')
             WITH ORDINALITY AS lista(elemento, ordinalidad);
    INSERT INTO vec_bolsa_baremacion.manifiesto_evidencia_v3 (
        manifiesto_ref, secuencia, tipo, referencia,
        huella_evidencia_sha256
    ) SELECT p_manifiesto ->> 'referencia', ordinalidad::integer,
             elemento ->> 'tipo', elemento ->> 'referencia',
             elemento ->> 'huella_evidencia_sha256'
        FROM jsonb_array_elements(p_manifiesto -> 'evidencias')
             WITH ORDINALITY AS lista(elemento, ordinalidad);
END
$funcion$;

CREATE FUNCTION vec_bolsa_baremacion.confirmar_cambio_con_archivo_probatorio_v3(
    p_operacion jsonb,
    p_prueba jsonb,
    p_decision_canonica bytea,
    p_recurso_canonico bytea,
    p_agregado_canonico bytea,
    p_manifiesto_probatorio jsonb,
    p_contenido_manifiesto_canonico bytea,
    p_representacion_manifiesto_canonica bytea,
    p_preimagen_hmac_manifiesto bytea,
    p_huella_prevalidacion_sha256 text
)
RETURNS TABLE (
    resultado text,
    numero_version text,
    huella_estado_sha256 text,
    agregado_canonico bytea,
    confirmada_en timestamptz,
    auditoria_ref text,
    huella_auditoria_sha256 text,
    evento_outbox_ref text,
    huella_evento_outbox_sha256 text,
    archivo_probatorio_documento jsonb,
    huella_prevalidacion_sha256 text
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
AS $funcion$
#variable_conflict use_variable
DECLARE
    instante timestamptz(6);
    solicitada_confirmacion timestamptz(6);
    accion_autorizada text;
    campos jsonb;
    version_esperada numeric(20, 0);
    huella_esperada text;
    autorizacion record;
    agregado jsonb;
    reserva record;
    uso record;
    actual_baremacion record;
    uso_encontrado boolean := false;
    confirmada_anterior timestamptz(6);
    agregado_anterior jsonb;
    prefijo_decisiones jsonb;
    ultima_decision jsonb;
    version_anterior numeric(20, 0);
    version_nueva numeric(20, 0);
    huella_anterior text;
    secuencia_auditoria numeric(20, 0);
    secuencia_evento numeric(20, 0);
    huella_auditoria_anterior text;
    huella_evento_anterior text;
    accion_auditoria text;
    tipo_evento text;
    decision_tecnica_ref text := '';
    manifiesto_ref text := '';
    huella_manifiesto text := '';
    documento_firmado_ref text := '';
    evidencia_custodia_ref text := '';
    evidencia_retencion_ref text := '';
    huella_auditoria text;
    huella_evento text;
    campos_unidos bytea;
    consumo_prevalidacion record;
    uso_prevalidacion record;
    resultado_prevalidacion record;
    archivo_base jsonb;
    huella_prevalidacion_calculada text;
    huella_prevalidacion_final text;
BEGIN
    resultado := 'rechazada';
    numero_version := '';
    huella_estado_sha256 := '';
    auditoria_ref := '';
    huella_auditoria_sha256 := '';
    evento_outbox_ref := '';
    huella_evento_outbox_sha256 := '';
    huella_prevalidacion_sha256 := '';

    IF p_operacion IS NULL OR jsonb_typeof(p_operacion) <> 'object'
       OR (SELECT count(*) FROM jsonb_object_keys(p_operacion)) <> 16
       OR NOT (p_operacion ?& ARRAY[
           'esquema', 'huella_token_sha256', 'clase',
           'version_esperada', 'huella_version_esperada_sha256',
           'huella_solicitud_hmac', 'huella_efecto_sha256',
           'huella_agregado_sha256', 'motivo_clave', 'motivo',
           'confirmada_en', 'auditoria_ref', 'evento_outbox_ref',
           'autorizacion_prevalidacion_ref',
           'huella_confirmacion_sha256',
           'huella_efecto_prevalidacion_sha256'
       ])
       OR p_operacion ->> 'esquema' <>
          'vec.bolsa.baremacion.confirmacion-postgresql.v3'
       OR vec_bolsa_baremacion.huella_sha256_valida(
           p_operacion ->> 'huella_token_sha256'
       ) IS NOT TRUE
       OR p_operacion ->> 'clase' NOT IN ('alta', 'incorporar_decision')
       OR vec_bolsa_baremacion.sello_hmac_manifiesto_v3_valido(
           p_operacion ->> 'huella_solicitud_hmac'
       ) IS NOT TRUE
       OR vec_bolsa_baremacion.huella_sha256_valida(
           p_operacion ->> 'huella_efecto_sha256'
       ) IS NOT TRUE
       OR vec_bolsa_baremacion.huella_sha256_valida(
           p_operacion ->> 'huella_confirmacion_sha256'
       ) IS NOT TRUE
       OR p_operacion ->> 'huella_confirmacion_sha256' IS DISTINCT FROM
          p_operacion ->> 'huella_efecto_sha256'
       OR vec_bolsa_baremacion.huella_sha256_valida(
           p_operacion ->> 'huella_agregado_sha256'
       ) IS NOT TRUE
       OR vec_bolsa_baremacion.texto_opaco_valido(
           p_operacion ->> 'motivo_clave', 128
       ) IS NOT TRUE
       OR p_operacion ->> 'motivo' IS NULL
       OR octet_length(p_operacion ->> 'motivo') NOT BETWEEN 1 AND 8000
       OR p_operacion ->> 'motivo' <> btrim(p_operacion ->> 'motivo')
       OR vec_bolsa_baremacion.instante_canonico_manifiesto_v3(
           p_operacion ->> 'confirmada_en'
       ) IS NOT TRUE
       OR vec_bolsa_baremacion.referencia_manifiesto_v3_valida(
           p_operacion ->> 'auditoria_ref', 512
       ) IS NOT TRUE
       OR vec_bolsa_baremacion.referencia_manifiesto_v3_valida(
           p_operacion ->> 'evento_outbox_ref', 512
       ) IS NOT TRUE
       OR p_operacion ->> 'auditoria_ref' =
          p_operacion ->> 'evento_outbox_ref'
       OR p_agregado_canonico IS NULL
       OR octet_length(p_agregado_canonico) NOT BETWEEN 1 AND 33554432
       OR encode(sha256(p_agregado_canonico), 'hex') IS DISTINCT FROM
          p_operacion ->> 'huella_agregado_sha256'
       OR (p_operacion ->> 'clase' = 'alta' AND (
           p_operacion ->> 'autorizacion_prevalidacion_ref' <> ''
           OR p_huella_prevalidacion_sha256 <> ''
           OR p_operacion ->> 'huella_efecto_prevalidacion_sha256' <> ''
           OR p_manifiesto_probatorio IS NOT NULL
           OR p_contenido_manifiesto_canonico IS NOT NULL
           OR p_representacion_manifiesto_canonica IS NOT NULL
           OR p_preimagen_hmac_manifiesto IS NOT NULL
       ))
       OR (p_operacion ->> 'clase' = 'incorporar_decision' AND
           (vec_bolsa_baremacion.referencia_manifiesto_v3_valida(
               p_operacion ->> 'autorizacion_prevalidacion_ref', 512
            ) IS NOT TRUE
            OR vec_bolsa_baremacion.huella_sha256_valida(
                p_huella_prevalidacion_sha256
            ) IS NOT TRUE
            OR vec_bolsa_baremacion.huella_sha256_valida(
                p_operacion ->> 'huella_efecto_prevalidacion_sha256'
            ) IS NOT TRUE
            OR p_operacion ->> 'huella_efecto_prevalidacion_sha256'
               IS DISTINCT FROM vec_bolsa_baremacion.huella_canonica(ARRAY[
                   'efecto-prevalidacion-archivo-probatorio-baremacion-v3',
                   p_operacion ->> 'huella_confirmacion_sha256'
               ])
            OR vec_bolsa_baremacion.archivo_unitario_manifiesto_v3_valido(
               p_manifiesto_probatorio,
               p_contenido_manifiesto_canonico,
               p_representacion_manifiesto_canonica,
               p_preimagen_hmac_manifiesto
            ) IS NOT TRUE)) THEN
        RETURN NEXT;
        RETURN;
    END IF;

    BEGIN
        agregado := convert_from(p_agregado_canonico, 'UTF8')::jsonb;
        solicitada_confirmacion :=
            (p_operacion ->> 'confirmada_en')::timestamptz;
    EXCEPTION
        WHEN character_not_in_repertoire OR untranslatable_character
            OR invalid_text_representation OR data_exception THEN
            RETURN NEXT;
            RETURN;
    END;
    instante := clock_timestamp();
    IF solicitada_confirmacion > instante
       OR jsonb_typeof(agregado) <> 'object'
       OR (SELECT count(*) FROM jsonb_object_keys(agregado)) <> 10
       OR NOT (agregado ?& ARRAY[
           'id', 'proceso_ref', 'solicitud_ref', 'sujeto_ref', 'criterio',
           'evidencias_iniciales', 'puntos_declarados', 'calculo_inicial',
           'creada_en', 'decisiones'
       ])
       OR vec_bolsa_baremacion.referencia_manifiesto_v3_valida(
           agregado ->> 'id', 512
       ) IS NOT TRUE
       OR agregado ->> 'sujeto_ref' IS DISTINCT FROM
          p_prueba ->> 'sujeto_ref'
       OR jsonb_typeof(agregado -> 'decisiones') <> 'array'
       OR jsonb_array_length(agregado -> 'decisiones') > 4096 THEN
        RETURN NEXT;
        RETURN;
    END IF;

    IF p_operacion ->> 'clase' = 'alta' THEN
        accion_autorizada := 'bolsa.baremacion.alta.confirmar';
        campos := '["baremacion","evidencia_transaccion"]'::jsonb;
        IF p_operacion ->> 'version_esperada' <> '0'
           OR p_operacion ->> 'huella_version_esperada_sha256' <> ''
           OR jsonb_array_length(agregado -> 'decisiones') <> 0 THEN
            RETURN NEXT;
            RETURN;
        END IF;
        version_esperada := NULL;
        huella_esperada := NULL;
    ELSE
        accion_autorizada := 'bolsa.baremacion.decision.confirmar';
        campos := '["baremacion","decision","evidencia_transaccion"]'::jsonb;
        IF (p_operacion ->> 'version_esperada') !~ '^[1-9][0-9]{0,19}$'
           OR vec_bolsa_baremacion.huella_sha256_valida(
               p_operacion ->> 'huella_version_esperada_sha256'
           ) IS NOT TRUE THEN
            RETURN NEXT;
            RETURN;
        END IF;
        version_esperada :=
            (p_operacion ->> 'version_esperada')::numeric;
        huella_esperada :=
            p_operacion ->> 'huella_version_esperada_sha256';
        IF version_esperada > 4096
           OR jsonb_array_length(agregado -> 'decisiones') IS DISTINCT FROM
              version_esperada::integer THEN
            RETURN NEXT;
            RETURN;
        END IF;
    END IF;

    SELECT * INTO autorizacion
      FROM vec_bolsa_baremacion.revalidar_operacion(
          p_prueba, p_decision_canonica, p_recurso_canonico,
          accion_autorizada, 'baremacion', agregado ->> 'id',
          campos, instante
      );
    IF NOT FOUND THEN
        resultado := 'autorizacion_obsoleta';
        RETURN NEXT;
        RETURN;
    END IF;

    PERFORM pg_advisory_xact_lock(hashtextextended(
        'vec_bolsa_baremacion:confirmacion:' ||
            (p_operacion ->> 'huella_token_sha256'), 0
    ));
    PERFORM pg_advisory_xact_lock(hashtextextended(
        'vec_bolsa_baremacion:agregado:' || (agregado ->> 'id'), 0
    ));

    SELECT actual.ambito_idempotencia_sha256, actual.reserva_ref,
           actual.version, actual.estado, version.principal_ref,
           version.sujeto_ref, version.vinculo_autenticacion_actor,
           version.baremacion_merito_ref, version.clase,
           version.version_esperada, version.huella_version_esperada_sha256,
           version.huella_solicitud_hmac, version.solicitada_en,
           version.expira_en, version.huella_confirmacion_sha256,
           version.numero_version_confirmada
      INTO reserva
      FROM vec_bolsa_baremacion.token_reserva AS token
      JOIN vec_bolsa_baremacion.reserva_actual AS actual
        ON actual.ambito_idempotencia_sha256 = token.ambito_idempotencia_sha256
       AND actual.reserva_ref = token.reserva_ref
      JOIN vec_bolsa_baremacion.reserva_version AS version
        ON version.ambito_idempotencia_sha256 =
           actual.ambito_idempotencia_sha256
       AND version.reserva_ref = actual.reserva_ref
       AND version.version = actual.version
     WHERE token.huella_token_sha256 =
           p_operacion ->> 'huella_token_sha256'
     FOR UPDATE OF actual;
    IF NOT FOUND
       OR reserva.principal_ref IS DISTINCT FROM
          autorizacion.decision_canonica ->> 'principal_id'
       OR reserva.sujeto_ref IS DISTINCT FROM p_prueba ->> 'sujeto_ref'
       OR reserva.vinculo_autenticacion_actor IS DISTINCT FROM
          autorizacion.decision_canonica -> 'vinculo_autenticacion_actor'
       OR reserva.baremacion_merito_ref IS DISTINCT FROM agregado ->> 'id'
       OR reserva.clase IS DISTINCT FROM p_operacion ->> 'clase'
       OR reserva.version_esperada IS DISTINCT FROM version_esperada
       OR reserva.huella_version_esperada_sha256 IS DISTINCT FROM
          huella_esperada
       OR reserva.huella_solicitud_hmac IS NOT DISTINCT FROM
          p_operacion ->> 'huella_solicitud_hmac'
       OR solicitada_confirmacion < reserva.solicitada_en
       OR solicitada_confirmacion >= reserva.expira_en THEN
        resultado := 'reserva_invalida';
        RETURN NEXT;
        RETURN;
    END IF;

    IF reserva.clase = 'incorporar_decision' THEN
        IF p_operacion ->> 'autorizacion_prevalidacion_ref' =
           p_prueba ->> 'decision_ref' THEN
            resultado := 'autorizacion_reutilizada';
            RETURN NEXT;
            RETURN;
        END IF;
        SELECT consumo.*, uso_pre.huella_efecto_sha256,
               uso_pre.tipo_efecto
          INTO consumo_prevalidacion
          FROM vec_bolsa_baremacion.prevalidacion_archivo_probatorio_v3
               AS consumo
          JOIN vec_bolsa_baremacion.uso_decision AS uso_pre
            ON uso_pre.decision_ref =
               consumo.autorizacion_prevalidacion_ref
         WHERE consumo.autorizacion_prevalidacion_ref =
               p_operacion ->> 'autorizacion_prevalidacion_ref'
         FOR SHARE OF consumo, uso_pre;
        IF NOT FOUND
           OR consumo_prevalidacion.tipo_efecto <>
              'prevalidacion_archivo'
           OR consumo_prevalidacion.huella_efecto_sha256 IS DISTINCT FROM
              p_operacion ->> 'huella_efecto_prevalidacion_sha256'
           OR consumo_prevalidacion.huella_efecto_prevalidacion_sha256
              IS DISTINCT FROM
              p_operacion ->> 'huella_efecto_prevalidacion_sha256'
           OR consumo_prevalidacion.huella_confirmacion_sha256
              IS DISTINCT FROM
              p_operacion ->> 'huella_confirmacion_sha256'
           OR consumo_prevalidacion.reserva_ref IS DISTINCT FROM
              reserva.reserva_ref
           OR consumo_prevalidacion.clase <> 'incorporar_decision'
           OR consumo_prevalidacion.baremacion_merito_ref IS DISTINCT FROM
              reserva.baremacion_merito_ref
           OR consumo_prevalidacion.huella_token_sha256 IS DISTINCT FROM
              p_operacion ->> 'huella_token_sha256'
           OR consumo_prevalidacion.principal_ref IS DISTINCT FROM
              reserva.principal_ref
           OR consumo_prevalidacion.sujeto_ref IS DISTINCT FROM
              reserva.sujeto_ref
           OR consumo_prevalidacion.finalidad_clave IS DISTINCT FROM
              autorizacion.decision_canonica ->> 'finalidad' THEN
            resultado := 'evidencia_no_confiable';
            RETURN NEXT;
            RETURN;
        END IF;

        IF consumo_prevalidacion.estado_resultado = 'activa' THEN
            archivo_base :=
                vec_bolsa_baremacion.construir_archivo_probatorio_v3(
                    reserva.baremacion_merito_ref, version_esperada
                );
            huella_prevalidacion_calculada :=
                vec_bolsa_baremacion.huella_prevalidacion_archivo_v3(
                    reserva.baremacion_merito_ref, version_esperada,
                    huella_esperada,
                    p_operacion ->> 'huella_token_sha256',
                    reserva.principal_ref, reserva.sujeto_ref,
                    autorizacion.decision_canonica ->> 'finalidad',
                    p_operacion ->> 'autorizacion_prevalidacion_ref',
                    p_operacion ->> 'huella_confirmacion_sha256'
                );
            IF archivo_base IS NULL
               OR consumo_prevalidacion.numero_version IS DISTINCT FROM
                  version_esperada
               OR consumo_prevalidacion.huella_estado_sha256 IS DISTINCT FROM
                  huella_esperada
               OR consumo_prevalidacion.huella_archivo_sha256 IS DISTINCT FROM
                  archivo_base ->> 'huella_archivo_sha256'
               OR consumo_prevalidacion.huella_prevalidacion_sha256
                  IS DISTINCT FROM huella_prevalidacion_calculada THEN
                resultado := 'evidencia_no_confiable';
                RETURN NEXT;
                RETURN;
            END IF;
            huella_prevalidacion_sha256 :=
                huella_prevalidacion_calculada;
        ELSE
            IF reserva.estado <> 'confirmada' THEN
                resultado := 'evidencia_no_confiable';
                RETURN NEXT;
                RETURN;
            END IF;
            huella_prevalidacion_calculada :=
                vec_bolsa_baremacion.huella_prevalidacion_archivo_v3(
                    reserva.baremacion_merito_ref,
                    reserva.numero_version_confirmada,
                    consumo_prevalidacion.huella_estado_sha256,
                    p_operacion ->> 'huella_token_sha256',
                    reserva.principal_ref, reserva.sujeto_ref,
                    autorizacion.decision_canonica ->> 'finalidad',
                    p_operacion ->> 'autorizacion_prevalidacion_ref',
                    p_operacion ->> 'huella_confirmacion_sha256'
                );
            IF consumo_prevalidacion.numero_version IS DISTINCT FROM
                  reserva.numero_version_confirmada
               OR consumo_prevalidacion.huella_prevalidacion_sha256
                  IS DISTINCT FROM huella_prevalidacion_calculada THEN
                resultado := 'evidencia_no_confiable';
                RETURN NEXT;
                RETURN;
            END IF;
            huella_prevalidacion_sha256 :=
                huella_prevalidacion_calculada;
        END IF;

        IF reserva.estado = 'activa' THEN
            IF p_huella_prevalidacion_sha256 IS DISTINCT FROM
               huella_prevalidacion_sha256 THEN
                resultado := 'evidencia_no_confiable';
                RETURN NEXT;
                RETURN;
            END IF;
        ELSIF consumo_prevalidacion.estado_resultado = 'activa' THEN
            SELECT * INTO resultado_prevalidacion
              FROM vec_bolsa_baremacion.resultado_prevalidacion_archivo_v3
             WHERE autorizacion_prevalidacion_ref =
                   consumo_prevalidacion.autorizacion_prevalidacion_ref;
            IF NOT FOUND OR resultado_prevalidacion.numero_version
                  IS DISTINCT FROM reserva.numero_version_confirmada
               OR resultado_prevalidacion.huella_confirmacion_sha256
                  IS DISTINCT FROM reserva.huella_confirmacion_sha256 THEN
                resultado := 'evidencia_no_confiable';
                RETURN NEXT;
                RETURN;
            END IF;
            huella_prevalidacion_final :=
                resultado_prevalidacion.huella_prevalidacion_sha256;
            IF p_huella_prevalidacion_sha256 IS DISTINCT FROM
                   huella_prevalidacion_sha256
               AND p_huella_prevalidacion_sha256 IS DISTINCT FROM
                   huella_prevalidacion_final THEN
                resultado := 'evidencia_no_confiable';
                RETURN NEXT;
                RETURN;
            END IF;
            huella_prevalidacion_sha256 := huella_prevalidacion_final;
        ELSIF p_huella_prevalidacion_sha256 IS DISTINCT FROM
              huella_prevalidacion_sha256 THEN
            resultado := 'evidencia_no_confiable';
            RETURN NEXT;
            RETURN;
        END IF;
    END IF;

    SELECT * INTO uso FROM vec_bolsa_baremacion.uso_decision
     WHERE decision_ref = p_prueba ->> 'decision_ref' FOR SHARE;
    uso_encontrado := FOUND;
    IF reserva.estado = 'confirmada' THEN
        IF reserva.huella_confirmacion_sha256 IS DISTINCT FROM
               p_operacion ->> 'huella_efecto_sha256'
           OR NOT uso_encontrado
           OR uso.huella_decision_sha256 IS DISTINCT FROM
               p_prueba ->> 'huella_decision_sha256'
           OR uso.huella_efecto_sha256 IS DISTINCT FROM
               p_operacion ->> 'huella_efecto_sha256'
           OR uso.tipo_efecto <> 'confirmacion' THEN
            resultado := 'idempotencia_reutilizada';
            RETURN NEXT;
            RETURN;
        END IF;
        SELECT version.numero::text, version.huella_estado_sha256,
               version.agregado_canonico, version.confirmada_en,
               version.auditoria_ref, auditoria.huella_registro_sha256,
               version.evento_outbox_ref, evento.huella_registro_sha256
          INTO numero_version, huella_estado_sha256, agregado_canonico,
               confirmada_en, auditoria_ref, huella_auditoria_sha256,
               evento_outbox_ref, huella_evento_outbox_sha256
          FROM vec_bolsa_baremacion.version_baremacion AS version
          JOIN vec_bolsa_baremacion.auditoria
            ON auditoria.referencia = version.auditoria_ref
          JOIN vec_bolsa_baremacion.evento_outbox AS evento
            ON evento.referencia = version.evento_outbox_ref
         WHERE version.baremacion_merito_ref = reserva.baremacion_merito_ref
           AND version.numero = reserva.numero_version_confirmada;
        resultado := CASE WHEN FOUND THEN 'confirmada' ELSE
            'evidencia_no_confiable' END;
        IF resultado = 'confirmada'
           AND reserva.clase = 'incorporar_decision'
           AND NOT EXISTS (
               SELECT 1
                 FROM vec_bolsa_baremacion.manifiesto_probatorio_v3 AS m
                WHERE m.baremacion_merito_ref = reserva.baremacion_merito_ref
                  AND m.numero_version = reserva.numero_version_confirmada
                  AND m.manifiesto = p_manifiesto_probatorio
                  AND m.contenido_manifiesto_canonico =
                      p_contenido_manifiesto_canonico
                  AND m.representacion_manifiesto_canonica =
                      p_representacion_manifiesto_canonica
                  AND m.preimagen_hmac_manifiesto =
                      p_preimagen_hmac_manifiesto
           ) THEN
            resultado := 'idempotencia_reutilizada';
        END IF;
        IF resultado = 'confirmada' THEN
            archivo_probatorio_documento :=
                vec_bolsa_baremacion.construir_archivo_probatorio_v3(
                    reserva.baremacion_merito_ref,
                    reserva.numero_version_confirmada
                );
            IF archivo_probatorio_documento IS NULL THEN
                resultado := 'evidencia_no_confiable';
            END IF;
        END IF;
        RETURN NEXT;
        RETURN;
    END IF;
    IF reserva.estado <> 'activa' THEN
        resultado := 'reserva_invalida';
        RETURN NEXT;
        RETURN;
    END IF;
    IF instante >= reserva.expira_en THEN
        INSERT INTO vec_bolsa_baremacion.reserva_version (
            reserva_ref, version, estado, ambito_idempotencia_sha256,
            principal_ref, sujeto_ref, vinculo_autenticacion_actor,
            baremacion_merito_ref, clase, version_esperada,
            huella_version_esperada_sha256, huella_solicitud_hmac,
            huella_efecto_reserva_sha256, decision_reserva_ref,
            huella_decision_reserva_sha256, solicitada_en, expira_en,
            registrada_en
        ) SELECT reserva_ref, version + 1, 'expirada',
            ambito_idempotencia_sha256, principal_ref, sujeto_ref,
            vinculo_autenticacion_actor, baremacion_merito_ref, clase,
            version_esperada, huella_version_esperada_sha256,
            huella_solicitud_hmac, huella_efecto_reserva_sha256,
            decision_reserva_ref, huella_decision_reserva_sha256,
            solicitada_en, expira_en, instante
          FROM vec_bolsa_baremacion.reserva_version
         WHERE reserva_ref = reserva.reserva_ref
           AND version = reserva.version;
        UPDATE vec_bolsa_baremacion.reserva_actual
           SET version = reserva.version + 1, estado = 'expirada'
         WHERE ambito_idempotencia_sha256 =
               reserva.ambito_idempotencia_sha256
           AND version = reserva.version AND estado = 'activa';
        IF NOT FOUND THEN
            RAISE EXCEPTION USING ERRCODE = '40001',
                MESSAGE = 'CAS de expiracion perdido';
        END IF;
        resultado := 'reserva_invalida';
        RETURN NEXT;
        RETURN;
    END IF;
    IF uso_encontrado THEN
        resultado := 'autorizacion_reutilizada';
        RETURN NEXT;
        RETURN;
    END IF;

    SELECT actual.numero, actual.huella_estado_sha256,
           version.agregado, version.sujeto_ref, version.confirmada_en
      INTO actual_baremacion
      FROM vec_bolsa_baremacion.baremacion_actual AS actual
      JOIN vec_bolsa_baremacion.version_baremacion AS version
        ON version.baremacion_merito_ref = actual.baremacion_merito_ref
       AND version.numero = actual.numero
     WHERE actual.baremacion_merito_ref = agregado ->> 'id'
     FOR UPDATE OF actual;
    IF p_operacion ->> 'clase' = 'alta' THEN
        IF FOUND THEN
            resultado := CASE
                WHEN actual_baremacion.sujeto_ref IS DISTINCT FROM
                     p_prueba ->> 'sujeto_ref' THEN 'no_encontrada'
                ELSE 'ya_existe' END;
            RETURN NEXT;
            RETURN;
        END IF;
        version_anterior := 0;
        version_nueva := 1;
        huella_anterior := '';
        confirmada_anterior := reserva.solicitada_en;
        accion_auditoria := 'crear_baremacion';
        tipo_evento := 'bolsa.baremacion_creada.v1';
    ELSE
        IF NOT FOUND OR actual_baremacion.sujeto_ref IS DISTINCT FROM
           p_prueba ->> 'sujeto_ref' THEN
            resultado := 'no_encontrada';
            RETURN NEXT;
            RETURN;
        END IF;
        IF actual_baremacion.numero IS DISTINCT FROM version_esperada
           OR actual_baremacion.huella_estado_sha256 IS DISTINCT FROM
              huella_esperada THEN
            resultado := 'conflicto_version';
            RETURN NEXT;
            RETURN;
        END IF;
        agregado_anterior := actual_baremacion.agregado;
        confirmada_anterior := actual_baremacion.confirmada_en;
        IF (agregado - 'decisiones') IS DISTINCT FROM
               (agregado_anterior - 'decisiones')
           OR jsonb_array_length(agregado -> 'decisiones') < 1 THEN
            resultado := 'historial_no_anexable';
            RETURN NEXT;
            RETURN;
        END IF;
        SELECT COALESCE(jsonb_agg(elemento ORDER BY orden), '[]'::jsonb)
          INTO prefijo_decisiones
          FROM jsonb_array_elements(agregado -> 'decisiones')
               WITH ORDINALITY AS d(elemento, orden)
         WHERE orden <= jsonb_array_length(
             agregado_anterior -> 'decisiones'
         );
        IF prefijo_decisiones IS DISTINCT FROM
           agregado_anterior -> 'decisiones' THEN
            resultado := 'historial_no_anexable';
            RETURN NEXT;
            RETURN;
        END IF;
        ultima_decision := agregado -> 'decisiones'
            -> (jsonb_array_length(agregado -> 'decisiones') - 1);
        IF jsonb_typeof(ultima_decision) <> 'object'
           OR ultima_decision -> 'contenido' ->> 'baremacion_merito_ref'
              IS DISTINCT FROM agregado ->> 'id'
           OR ultima_decision -> 'contenido' ->> 'proceso_ref'
              IS DISTINCT FROM agregado ->> 'proceso_ref'
           OR ultima_decision -> 'contenido' ->> 'solicitud_ref'
              IS DISTINCT FROM agregado ->> 'solicitud_ref'
           OR ultima_decision -> 'contenido' ->> 'sujeto_ref'
              IS DISTINCT FROM agregado ->> 'sujeto_ref'
           OR (ultima_decision -> 'contenido'
               ->> 'version_anterior_baremacion')::numeric
              IS DISTINCT FROM version_esperada
           OR (ultima_decision -> 'contenido'
               ->> 'version_baremacion')::numeric
              IS DISTINCT FROM version_esperada + 1
           OR ultima_decision -> 'contenido' ->> 'decisor_ref'
              IS DISTINCT FROM autorizacion.decision_canonica
                  ->> 'principal_id'
           OR ultima_decision -> 'contenido' ->> 'perfil_decisor_clave'
              IS DISTINCT FROM autorizacion.decision_canonica
                  ->> 'perfil_activo_ref'
           OR ultima_decision -> 'contenido' ->> 'autorizacion_ref' =
              p_prueba ->> 'decision_ref'
           OR ultima_decision -> 'contenido' ->> 'finalidad_clave'
              IS DISTINCT FROM autorizacion.decision_canonica
                  ->> 'finalidad'
           OR ultima_decision -> 'contenido' ->> 'correlacion_ref'
              IS DISTINCT FROM autorizacion.decision_canonica
                  ->> 'correlacion_ref'
           OR vec_bolsa_baremacion.referencia_manifiesto_v3_valida(
               ultima_decision -> 'contenido' ->> 'id', 512
           ) IS NOT TRUE
           OR vec_bolsa_baremacion.huella_sha256_valida(
               ultima_decision ->> 'huella_sha256'
           ) IS NOT TRUE
           OR vec_bolsa_baremacion.referencia_manifiesto_v3_valida(
               ultima_decision -> 'firma'
                   ->> 'documento_firmado_custodiado_ref', 512
           ) IS NOT TRUE
           OR vec_bolsa_baremacion.referencia_manifiesto_v3_valida(
               ultima_decision -> 'firma'
                   ->> 'evidencia_custodia_documento_firmado_ref', 512
           ) IS NOT TRUE
           OR vec_bolsa_baremacion.referencia_manifiesto_v3_valida(
               ultima_decision -> 'firma'
                   ->> 'evidencia_retencion_documento_firmado_ref', 512
           ) IS NOT TRUE
           OR vec_bolsa_baremacion.huella_sha256_valida(
               ultima_decision -> 'firma'
                   ->> 'huella_manifiesto_probatorio_sha256'
           ) IS NOT TRUE
           OR p_manifiesto_probatorio ->> 'baremacion_merito_ref'
              IS DISTINCT FROM agregado ->> 'id'
           OR p_manifiesto_probatorio ->> 'proceso_ref'
              IS DISTINCT FROM agregado ->> 'proceso_ref'
           OR p_manifiesto_probatorio ->> 'solicitud_ref'
              IS DISTINCT FROM agregado ->> 'solicitud_ref'
           OR p_manifiesto_probatorio ->> 'sujeto_ref'
              IS DISTINCT FROM agregado ->> 'sujeto_ref'
           OR p_manifiesto_probatorio ->> 'decision_ref'
              IS DISTINCT FROM ultima_decision -> 'contenido' ->> 'id'
           OR (p_manifiesto_probatorio ->> 'version_base')::numeric
              IS DISTINCT FROM version_esperada
           OR p_manifiesto_probatorio ->> 'huella_version_base_sha256'
              IS DISTINCT FROM huella_esperada
           OR p_manifiesto_probatorio ->> 'referencia'
              IS DISTINCT FROM ultima_decision -> 'firma'
                  ->> 'manifiesto_probatorio_ref'
           OR p_manifiesto_probatorio ->> 'huella_manifiesto_sha256'
              IS DISTINCT FROM ultima_decision -> 'firma'
                  ->> 'huella_manifiesto_probatorio_sha256'
           OR p_manifiesto_probatorio ->> 'sello_manifiesto_hmac_sha256'
              IS DISTINCT FROM ultima_decision -> 'firma'
                  ->> 'sello_manifiesto_probatorio_hmac_sha256' THEN
            resultado := 'historial_no_anexable';
            RETURN NEXT;
            RETURN;
        END IF;
        version_anterior := version_esperada;
        version_nueva := version_esperada + 1;
        huella_anterior := huella_esperada;
        accion_auditoria := 'incorporar_decision_baremacion';
        tipo_evento := 'bolsa.decision_baremacion_incorporada.v1';
        decision_tecnica_ref := ultima_decision -> 'contenido' ->> 'id';
        manifiesto_ref := ultima_decision -> 'firma'
            ->> 'manifiesto_probatorio_ref';
        huella_manifiesto := ultima_decision -> 'firma'
            ->> 'huella_manifiesto_probatorio_sha256';
        documento_firmado_ref := ultima_decision -> 'firma'
            ->> 'documento_firmado_custodiado_ref';
        evidencia_custodia_ref := ultima_decision -> 'firma'
            ->> 'evidencia_custodia_documento_firmado_ref';
        evidencia_retencion_ref := ultima_decision -> 'firma'
            ->> 'evidencia_retencion_documento_firmado_ref';
    END IF;

    IF solicitada_confirmacion < confirmada_anterior THEN
        resultado := 'historial_no_anexable';
        RETURN NEXT;
        RETURN;
    END IF;

    PERFORM pg_advisory_xact_lock(hashtextextended(
        'vec_bolsa_baremacion:cadenas-transaccion:v1', 0
    ));
    SELECT secuencia, huella_registro_sha256
      INTO secuencia_auditoria, huella_auditoria_anterior
      FROM vec_bolsa_baremacion.auditoria
     ORDER BY secuencia DESC LIMIT 1 FOR SHARE;
    IF NOT FOUND THEN
        secuencia_auditoria := 1;
        huella_auditoria_anterior := '';
    ELSE
        secuencia_auditoria := secuencia_auditoria + 1;
    END IF;
    SELECT secuencia, huella_registro_sha256
      INTO secuencia_evento, huella_evento_anterior
      FROM vec_bolsa_baremacion.evento_outbox
     ORDER BY secuencia DESC LIMIT 1 FOR SHARE;
    IF NOT FOUND THEN
        secuencia_evento := 1;
        huella_evento_anterior := '';
    ELSE
        secuencia_evento := secuencia_evento + 1;
    END IF;
    IF secuencia_auditoria IS DISTINCT FROM secuencia_evento
       OR secuencia_auditoria > 18446744073709551615 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'cadenas de auditoria y outbox divergentes';
    END IF;

    confirmada_en := instante;
    auditoria_ref := p_operacion ->> 'auditoria_ref';
    evento_outbox_ref := p_operacion ->> 'evento_outbox_ref';
    campos_unidos := vec_bolsa_baremacion.unir_textos_nul(campos);
    huella_auditoria := vec_bolsa_baremacion.huella_canonica_bytes(ARRAY[
        convert_to(auditoria_ref, 'UTF8'),
        convert_to(secuencia_auditoria::text, 'UTF8'),
        convert_to(autorizacion.decision_canonica ->> 'principal_id', 'UTF8'),
        convert_to(p_prueba ->> 'sujeto_ref', 'UTF8'),
        convert_to(autorizacion.decision_canonica ->> 'perfil_activo_ref', 'UTF8'),
        convert_to(autorizacion.decision_canonica -> 'vinculo_autenticacion_actor' ->> 'metodo_observado', 'UTF8'),
        convert_to(autorizacion.decision_canonica -> 'vinculo_autenticacion_actor' ->> 'garantia_observada', 'UTF8'),
        convert_to(autorizacion.decision_canonica ->> 'garantia_minima', 'UTF8'),
        convert_to(autorizacion.decision_canonica -> 'vinculo_autenticacion_actor' ->> 'autenticacion_ref', 'UTF8'),
        convert_to(p_prueba ->> 'decision_ref', 'UTF8'),
        convert_to(accion_autorizada, 'UTF8'), convert_to('baremacion', 'UTF8'),
        convert_to(agregado ->> 'id', 'UTF8'), campos_unidos,
        convert_to(autorizacion.decision_canonica ->> 'finalidad', 'UTF8'),
        convert_to(autorizacion.decision_canonica ->> 'correlacion_ref', 'UTF8'),
        convert_to('bolsa', 'UTF8'), convert_to(accion_auditoria, 'UTF8'),
        convert_to(p_operacion ->> 'clase', 'UTF8'),
        convert_to(agregado ->> 'proceso_ref', 'UTF8'),
        convert_to(agregado ->> 'solicitud_ref', 'UTF8'),
        convert_to(agregado ->> 'id', 'UTF8'),
        convert_to(decision_tecnica_ref, 'UTF8'),
        convert_to(manifiesto_ref, 'UTF8'), convert_to(huella_manifiesto, 'UTF8'),
        convert_to(documento_firmado_ref, 'UTF8'),
        convert_to(evidencia_custodia_ref, 'UTF8'),
        convert_to(evidencia_retencion_ref, 'UTF8'),
        convert_to(version_anterior::text, 'UTF8'),
        convert_to(version_nueva::text, 'UTF8'),
        convert_to(huella_anterior, 'UTF8'),
        convert_to(p_operacion ->> 'huella_agregado_sha256', 'UTF8'),
        convert_to(p_operacion ->> 'motivo_clave', 'UTF8'),
        convert_to(p_operacion ->> 'motivo', 'UTF8'),
        convert_to(p_operacion ->> 'huella_solicitud_hmac', 'UTF8'),
        convert_to('correcto', 'UTF8'),
        convert_to(p_operacion ->> 'confirmada_en', 'UTF8'),
        convert_to(vec_bolsa_baremacion.instante_rfc3339nano(instante), 'UTF8'),
        convert_to(huella_auditoria_anterior, 'UTF8')
    ]);
    huella_evento := vec_bolsa_baremacion.huella_canonica(ARRAY[
        evento_outbox_ref, secuencia_evento::text, tipo_evento, 'pendiente',
        'bolsa', agregado ->> 'proceso_ref', agregado ->> 'solicitud_ref',
        agregado ->> 'id', decision_tecnica_ref, manifiesto_ref,
        huella_manifiesto, documento_firmado_ref, evidencia_custodia_ref,
        evidencia_retencion_ref, p_prueba ->> 'sujeto_ref',
        autorizacion.decision_canonica ->> 'principal_id',
        version_nueva::text, p_operacion ->> 'huella_agregado_sha256',
        auditoria_ref, huella_auditoria,
        autorizacion.decision_canonica ->> 'correlacion_ref',
        vec_bolsa_baremacion.instante_rfc3339nano(instante),
        huella_evento_anterior
    ]);

    INSERT INTO vec_bolsa_baremacion.auditoria (
        referencia, secuencia, principal_ref, sujeto_ref,
        perfil_actor_clave, metodo_autenticacion, nivel_autenticacion,
        garantia_minima, autenticacion_ref, autorizacion_ref,
        accion_autorizada, clase_recurso_autorizada,
        recurso_autorizado_ref, campos_permitidos, finalidad_clave,
        correlacion_ref, modulo, accion, clase_cambio, proceso_ref,
        solicitud_ref, baremacion_merito_ref, decision_ref,
        manifiesto_probatorio_ref, huella_manifiesto_sha256,
        documento_firmado_custodiado_ref,
        evidencia_custodia_firmado_ref, evidencia_retencion_firmado_ref,
        version_anterior, version_nueva, huella_anterior_sha256,
        huella_nueva_sha256, motivo_clave, motivo, huella_solicitud_hmac,
        resultado, solicitada_confirmacion_en,
        solicitada_confirmacion_canonica, registrada_en,
        huella_anterior_auditoria_sha256, huella_registro_sha256
    ) VALUES (
        auditoria_ref, secuencia_auditoria,
        autorizacion.decision_canonica ->> 'principal_id',
        p_prueba ->> 'sujeto_ref',
        autorizacion.decision_canonica ->> 'perfil_activo_ref',
        autorizacion.decision_canonica -> 'vinculo_autenticacion_actor'
            ->> 'metodo_observado',
        autorizacion.decision_canonica -> 'vinculo_autenticacion_actor'
            ->> 'garantia_observada',
        autorizacion.decision_canonica ->> 'garantia_minima',
        autorizacion.decision_canonica -> 'vinculo_autenticacion_actor'
            ->> 'autenticacion_ref',
        p_prueba ->> 'decision_ref', accion_autorizada, 'baremacion',
        agregado ->> 'id', campos,
        autorizacion.decision_canonica ->> 'finalidad',
        autorizacion.decision_canonica ->> 'correlacion_ref',
        'bolsa', accion_auditoria, p_operacion ->> 'clase',
        agregado ->> 'proceso_ref', agregado ->> 'solicitud_ref',
        agregado ->> 'id', decision_tecnica_ref, manifiesto_ref,
        huella_manifiesto, documento_firmado_ref, evidencia_custodia_ref,
        evidencia_retencion_ref, version_anterior, version_nueva,
        huella_anterior, p_operacion ->> 'huella_agregado_sha256',
        p_operacion ->> 'motivo_clave', p_operacion ->> 'motivo',
        p_operacion ->> 'huella_solicitud_hmac', 'correcto',
        solicitada_confirmacion, p_operacion ->> 'confirmada_en',
        instante, huella_auditoria_anterior,
        huella_auditoria
    );
    INSERT INTO vec_bolsa_baremacion.evento_outbox (
        referencia, secuencia, tipo, estado, modulo, proceso_ref,
        solicitud_ref, baremacion_merito_ref, decision_ref,
        manifiesto_probatorio_ref, huella_manifiesto_sha256,
        documento_firmado_ref, evidencia_custodia_firmado_ref,
        evidencia_retencion_firmado_ref, sujeto_ref, principal_ref,
        version_nueva, huella_nueva_sha256, auditoria_ref,
        huella_auditoria_sha256, correlacion_ref, registrada_en,
        huella_evento_anterior_sha256, huella_registro_sha256
    ) VALUES (
        evento_outbox_ref, secuencia_evento, tipo_evento, 'pendiente',
        'bolsa', agregado ->> 'proceso_ref', agregado ->> 'solicitud_ref',
        agregado ->> 'id', decision_tecnica_ref, manifiesto_ref,
        huella_manifiesto, documento_firmado_ref, evidencia_custodia_ref,
        evidencia_retencion_ref, p_prueba ->> 'sujeto_ref',
        autorizacion.decision_canonica ->> 'principal_id', version_nueva,
        p_operacion ->> 'huella_agregado_sha256', auditoria_ref,
        huella_auditoria,
        autorizacion.decision_canonica ->> 'correlacion_ref',
        instante, huella_evento_anterior, huella_evento
    );
    INSERT INTO vec_bolsa_baremacion.version_baremacion (
        baremacion_merito_ref, numero, huella_estado_sha256,
        agregado_canonico, agregado, sujeto_ref, proceso_ref,
        solicitud_ref, confirmada_en, reserva_ref, auditoria_ref,
        evento_outbox_ref
    ) VALUES (
        agregado ->> 'id', version_nueva,
        p_operacion ->> 'huella_agregado_sha256', p_agregado_canonico,
        agregado, p_prueba ->> 'sujeto_ref', agregado ->> 'proceso_ref',
        agregado ->> 'solicitud_ref', instante, reserva.reserva_ref,
        auditoria_ref, evento_outbox_ref
    );
    IF version_nueva > 1 THEN
        PERFORM vec_bolsa_baremacion.registrar_manifiesto_probatorio_v3(
            p_manifiesto_probatorio, p_contenido_manifiesto_canonico,
            p_representacion_manifiesto_canonica,
            p_preimagen_hmac_manifiesto, version_nueva, auditoria_ref,
            evento_outbox_ref, reserva.reserva_ref, instante
        );
    END IF;
    IF version_nueva = 1 THEN
        INSERT INTO vec_bolsa_baremacion.baremacion_actual (
            baremacion_merito_ref, numero, huella_estado_sha256,
            actualizada_en
        ) VALUES (
            agregado ->> 'id', 1,
            p_operacion ->> 'huella_agregado_sha256', instante
        );
    ELSE
        UPDATE vec_bolsa_baremacion.baremacion_actual AS actual
           SET numero = version_nueva,
               huella_estado_sha256 =
                   p_operacion ->> 'huella_agregado_sha256',
               actualizada_en = instante
         WHERE actual.baremacion_merito_ref = agregado ->> 'id'
           AND actual.numero = version_anterior
           AND actual.huella_estado_sha256 = huella_anterior;
        IF NOT FOUND THEN
            RAISE EXCEPTION USING ERRCODE = '40001',
                MESSAGE = 'OCC de baremacion perdido';
        END IF;
    END IF;
    INSERT INTO vec_bolsa_baremacion.reserva_version (
        reserva_ref, version, estado, ambito_idempotencia_sha256,
        principal_ref, sujeto_ref, vinculo_autenticacion_actor,
        baremacion_merito_ref, clase, version_esperada,
        huella_version_esperada_sha256, huella_solicitud_hmac,
        huella_efecto_reserva_sha256, decision_reserva_ref,
        huella_decision_reserva_sha256, solicitada_en, expira_en,
        huella_confirmacion_sha256, numero_version_confirmada,
        registrada_en
    ) SELECT reserva_ref, version + 1, 'confirmada',
        ambito_idempotencia_sha256, principal_ref, sujeto_ref,
        vinculo_autenticacion_actor, baremacion_merito_ref, clase,
        version_esperada, huella_version_esperada_sha256,
        huella_solicitud_hmac, huella_efecto_reserva_sha256,
        decision_reserva_ref, huella_decision_reserva_sha256,
        solicitada_en, expira_en,
        p_operacion ->> 'huella_efecto_sha256', version_nueva, instante
      FROM vec_bolsa_baremacion.reserva_version
     WHERE reserva_ref = reserva.reserva_ref AND version = reserva.version;
    UPDATE vec_bolsa_baremacion.reserva_actual
       SET version = reserva.version + 1, estado = 'confirmada'
     WHERE ambito_idempotencia_sha256 =
           reserva.ambito_idempotencia_sha256
       AND version = reserva.version AND estado = 'activa';
    IF NOT FOUND THEN
        RAISE EXCEPTION USING ERRCODE = '40001',
            MESSAGE = 'CAS de reserva perdido';
    END IF;
    INSERT INTO vec_bolsa_baremacion.uso_decision (
        decision_ref, esquema_huella_decision, huella_decision_sha256,
        huella_efecto_sha256, tipo_efecto, resultado_ref,
        atestacion_ref, atestacion_version, consumida_en
    ) VALUES (
        p_prueba ->> 'decision_ref', p_prueba ->> 'esquema_huella',
        p_prueba ->> 'huella_decision_sha256',
        p_operacion ->> 'huella_efecto_sha256', 'confirmacion',
        auditoria_ref, autorizacion.atestacion_ref,
        autorizacion.atestacion_version, instante
    );

    resultado := 'confirmada';
    numero_version := version_nueva::text;
    huella_estado_sha256 := p_operacion ->> 'huella_agregado_sha256';
    agregado_canonico := p_agregado_canonico;
    huella_auditoria_sha256 := huella_auditoria;
    huella_evento_outbox_sha256 := huella_evento;
    archivo_probatorio_documento :=
        vec_bolsa_baremacion.construir_archivo_probatorio_v3(
            agregado ->> 'id', version_nueva
        );
    IF archivo_probatorio_documento IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '23514',
            MESSAGE = 'archivo probatorio V3 incompleto tras confirmar';
    END IF;
    IF p_operacion ->> 'clase' = 'incorporar_decision' THEN
        huella_prevalidacion_final :=
            vec_bolsa_baremacion.huella_prevalidacion_archivo_v3(
                agregado ->> 'id', version_nueva,
                p_operacion ->> 'huella_agregado_sha256',
                p_operacion ->> 'huella_token_sha256',
                reserva.principal_ref, reserva.sujeto_ref,
                autorizacion.decision_canonica ->> 'finalidad',
                p_operacion ->> 'autorizacion_prevalidacion_ref',
                p_operacion ->> 'huella_confirmacion_sha256'
            );
        IF huella_prevalidacion_final IS NULL THEN
            RAISE EXCEPTION USING ERRCODE = '23514',
                MESSAGE = 'resultado de prevalidacion V3 no reconstruible';
        END IF;
        INSERT INTO
            vec_bolsa_baremacion.resultado_prevalidacion_archivo_v3 (
                autorizacion_prevalidacion_ref,
                huella_prevalidacion_sha256, numero_version,
                huella_estado_sha256, total_manifiestos,
                huella_archivo_sha256, huella_confirmacion_sha256,
                registrada_en
            ) VALUES (
                p_operacion ->> 'autorizacion_prevalidacion_ref',
                huella_prevalidacion_final, version_nueva,
                p_operacion ->> 'huella_agregado_sha256',
                version_nueva::integer - 1,
                archivo_probatorio_documento ->> 'huella_archivo_sha256',
                p_operacion ->> 'huella_efecto_sha256', instante
            );
        huella_prevalidacion_sha256 := huella_prevalidacion_final;
    END IF;
    RETURN NEXT;
EXCEPTION
    WHEN unique_violation THEN
        resultado := 'colision';
        RETURN NEXT;
    WHEN data_exception OR invalid_text_representation
        OR datetime_field_overflow OR check_violation
        OR foreign_key_violation OR no_data_found OR too_many_rows THEN
        resultado := 'rechazada';
        RETURN NEXT;
END
$funcion$;

-- Cierre explicito de relaciones y helpers internos. El runtime solo atraviesa
-- las seis funciones SECURITY DEFINER V3; las operaciones V1 quedan revocadas.
REVOKE ALL ON TABLE
    vec_bolsa_baremacion.manifiesto_probatorio_v3,
    vec_bolsa_baremacion.manifiesto_autorizacion_v3,
    vec_bolsa_baremacion.manifiesto_evidencia_v3,
    vec_bolsa_baremacion.prevalidacion_archivo_probatorio_v3,
    vec_bolsa_baremacion.resultado_prevalidacion_archivo_v3
    FROM PUBLIC, vec_bolsa_baremacion_ejecutor,
         vec_bolsa_baremacion_lector_outbox,
         vec_bolsa_baremacion_registrador_atestacion;

REVOKE EXECUTE ON FUNCTION vec_bolsa_baremacion.reservar_cambio(
    jsonb, jsonb, bytea, bytea
) FROM vec_bolsa_baremacion_ejecutor;
REVOKE EXECUTE ON FUNCTION vec_bolsa_baremacion.confirmar_cambio(
    jsonb, jsonb, bytea, bytea, bytea
) FROM vec_bolsa_baremacion_ejecutor;
REVOKE EXECUTE ON FUNCTION vec_bolsa_baremacion.obtener_version_vigente(
    jsonb, jsonb, bytea, bytea
) FROM vec_bolsa_baremacion_ejecutor;
REVOKE EXECUTE ON FUNCTION vec_bolsa_baremacion.obtener_version(
    jsonb, jsonb, bytea, bytea
) FROM vec_bolsa_baremacion_ejecutor;
REVOKE EXECUTE ON FUNCTION
    vec_bolsa_baremacion.obtener_evidencia_transaccion(
        jsonb, jsonb, bytea, bytea
    ) FROM vec_bolsa_baremacion_ejecutor;

GRANT EXECUTE ON FUNCTION
    vec_bolsa_baremacion.reservar_cambio_con_archivo_probatorio_v3(
        jsonb, jsonb, bytea, bytea
    ) TO vec_bolsa_baremacion_ejecutor;
GRANT EXECUTE ON FUNCTION
    vec_bolsa_baremacion.obtener_archivo_probatorio_previo_cambio_v3(
        jsonb, jsonb, bytea, bytea
    ) TO vec_bolsa_baremacion_ejecutor;
GRANT EXECUTE ON FUNCTION
    vec_bolsa_baremacion.confirmar_cambio_con_archivo_probatorio_v3(
        jsonb, jsonb, bytea, bytea, bytea, jsonb, bytea, bytea,
        bytea, text
    ) TO vec_bolsa_baremacion_ejecutor;
GRANT EXECUTE ON FUNCTION
    vec_bolsa_baremacion.obtener_version_vigente_con_archivo_probatorio_v3(
        jsonb, jsonb, bytea, bytea
    ) TO vec_bolsa_baremacion_ejecutor;
GRANT EXECUTE ON FUNCTION
    vec_bolsa_baremacion.obtener_version_con_archivo_probatorio_v3(
        jsonb, jsonb, bytea, bytea
    ) TO vec_bolsa_baremacion_ejecutor;
GRANT EXECUTE ON FUNCTION
    vec_bolsa_baremacion.obtener_evidencia_transaccion_con_archivo_probatorio_v3(
        jsonb, jsonb, bytea, bytea
    ) TO vec_bolsa_baremacion_ejecutor;
COMMIT;
