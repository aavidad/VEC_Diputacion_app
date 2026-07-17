-- Operaciones atomicas cerradas. Deliberadamente no se concede EXECUTE a
-- ningun rol runtime mientras este almacen no se componga atomicamente con la
-- puerta VEC-AD-2 existente y application no publique el contrato estable de
-- accion/recurso/finalidad.
BEGIN;
SET LOCAL ROLE vec_bolsa_reglas_baremo_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_bolsa_reglas_baremo:migracion:000002', 0
    )
);

DO $prevalidacion$
BEGIN
    IF to_regclass(
           'vec_bolsa_reglas_baremo.version_reglas_baremo'
       ) IS NULL
       OR to_regprocedure(
           'vec_autorizacion.revalidar_decision_reglas_baremo_v1(jsonb,bytea,bytea,text,text,text,text,timestamp with time zone)'
       ) IS NULL
       OR to_regprocedure(
           'vec_bolsa_reglas_baremo.confirmar_cambio_v1(jsonb,jsonb,bytea,bytea,bytea)'
       ) IS NOT NULL
       OR to_regprocedure(
           'vec_bolsa_reglas_baremo.obtener_version_exacta_v1(jsonb,jsonb,bytea,bytea)'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para instalar operaciones de reglas de baremo';
    END IF;
END
$prevalidacion$;

CREATE FUNCTION vec_bolsa_reglas_baremo.confirmar_cambio_v1(
    p_operacion jsonb,
    p_prueba_autorizacion jsonb,
    p_decision_canonica bytea,
    p_recurso_canonico bytea,
    p_version_canonica bytea
)
RETURNS TABLE (
    resultado text,
    contenido_ref text,
    contenido_version numeric,
    huella_contenido_sha256 text,
    revision numeric,
    estado text,
    huella_estado_sha256 text,
    transaccion_ref text,
    transaccion_version numeric,
    huella_transaccion_sha256 text,
    auditoria_ref text,
    auditoria_version numeric,
    huella_auditoria_sha256 text,
    outbox_ref text,
    outbox_version numeric,
    huella_evento_sha256 text,
    consumo_autorizacion_ref text,
    consumo_autorizacion_version numeric,
    huella_consumo_autorizacion_sha256 text,
    consumo_prueba_ref text,
    consumo_prueba_version numeric,
    huella_consumo_prueba_sha256 text,
    confirmada_en timestamptz
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
SET timezone = 'UTC'
AS $funcion$
DECLARE
    v_tenant constant text := 'diputacion_granada';
    v_operacion text;
    v_intencion_ref text;
    v_intencion_version numeric(20, 0);
    v_intencion_huella text;
    v_esperado_revision numeric(20, 0);
    v_esperado_huella text;
    v_contenido_ref text;
    v_contenido_version numeric(20, 0);
    v_huella_contenido text;
    v_resultado_revision numeric(20, 0);
    v_resultado_estado text;
    v_resultado_huella text;
    v_prueba_ref text;
    v_prueba_version numeric(20, 0);
    v_prueba_huella text;
    v_accion text;
    v_accion_esperada text;
    v_recurso_ref text;
    v_correlacion_ref text;
    v_huella_contexto text;
    v_efectuar_en timestamptz;
    v_ahora timestamptz(6);
    v_principal_ref text;
    v_actual record;
    v_previa record;
    v_existente record;
    v_efecto bytea;
    v_huella_efecto text;
    v_consumo_autorizacion_ref text;
    v_huella_consumo_autorizacion text;
    v_consumo_autorizacion_canonico bytea;
    v_consumo_prueba_ref text;
    v_huella_consumo_prueba text;
    v_consumo_prueba_canonico bytea;
    v_transaccion_ref text;
    v_huella_transaccion text;
    v_transaccion_canonica bytea;
    v_ultima_secuencia bigint;
    v_huella_anterior text;
    v_auditoria_ref text;
    v_registro_auditoria bytea;
    v_huella_auditoria text;
    v_outbox_ref text;
    v_evento bytea;
    v_huella_evento text;
BEGIN
    IF current_setting('transaction_isolation') <> 'serializable' THEN
        RAISE EXCEPTION USING ERRCODE = '25001',
            MESSAGE = 'confirmacion rechazada: requiere SERIALIZABLE';
    END IF;
    IF p_operacion IS NULL OR jsonb_typeof(p_operacion) <> 'object'
       OR (SELECT count(*) FROM jsonb_object_keys(p_operacion)) <> 21
       OR NOT (p_operacion ?& ARRAY[
           'esquema', 'operacion', 'intencion_ref', 'intencion_version',
           'intencion_huella_sha256', 'esperado_revision',
           'esperado_huella_estado_sha256', 'contenido_ref',
           'contenido_version', 'huella_contenido_sha256',
           'resultado_revision', 'resultado_estado',
           'resultado_huella_estado_sha256', 'prueba_ref',
           'prueba_version', 'prueba_huella_sha256', 'accion',
           'recurso_ref', 'correlacion_ref',
           'huella_contexto_recurso_sha256', 'efectuar_en'
       ])
       OR jsonb_typeof(p_operacion -> 'esquema') <> 'string'
       OR jsonb_typeof(p_operacion -> 'operacion') <> 'string'
       OR jsonb_typeof(p_operacion -> 'intencion_ref') <> 'string'
       OR jsonb_typeof(p_operacion -> 'intencion_version') <> 'number'
       OR jsonb_typeof(
              p_operacion -> 'intencion_huella_sha256'
          ) <> 'string'
       OR jsonb_typeof(p_operacion -> 'esperado_revision')
          NOT IN ('number', 'null')
       OR jsonb_typeof(
              p_operacion -> 'esperado_huella_estado_sha256'
          ) NOT IN ('string', 'null')
       OR jsonb_typeof(p_operacion -> 'contenido_ref') <> 'string'
       OR jsonb_typeof(p_operacion -> 'contenido_version') <> 'number'
       OR jsonb_typeof(
              p_operacion -> 'huella_contenido_sha256'
          ) <> 'string'
       OR jsonb_typeof(p_operacion -> 'resultado_revision') <> 'number'
       OR jsonb_typeof(p_operacion -> 'resultado_estado') <> 'string'
       OR jsonb_typeof(
              p_operacion -> 'resultado_huella_estado_sha256'
          ) <> 'string'
       OR jsonb_typeof(p_operacion -> 'prueba_ref')
          NOT IN ('string', 'null')
       OR jsonb_typeof(p_operacion -> 'prueba_version')
          NOT IN ('number', 'null')
       OR jsonb_typeof(p_operacion -> 'prueba_huella_sha256')
          NOT IN ('string', 'null')
       OR jsonb_typeof(p_operacion -> 'accion') <> 'string'
       OR jsonb_typeof(p_operacion -> 'recurso_ref') <> 'string'
       OR jsonb_typeof(p_operacion -> 'correlacion_ref') <> 'string'
       OR jsonb_typeof(
              p_operacion -> 'huella_contexto_recurso_sha256'
          ) <> 'string'
       OR jsonb_typeof(p_operacion -> 'efectuar_en') <> 'string'
       OR p_operacion ->> 'esquema' IS DISTINCT FROM
          'vec.bolsa.reglas-baremo.confirmacion-postgresql.v1'
       OR (p_operacion ->> 'intencion_version') !~ '^[1-9][0-9]{0,9}$'
       OR (p_operacion ->> 'contenido_version') !~ '^[1-9][0-9]{0,9}$'
       OR (p_operacion ->> 'resultado_revision') !~ '^[1-9][0-9]{0,9}$'
       OR (p_operacion -> 'esperado_revision' <> 'null'::jsonb
           AND (p_operacion ->> 'esperado_revision') !~
               '^[1-9][0-9]{0,9}$')
       OR (p_operacion -> 'prueba_version' <> 'null'::jsonb
           AND (p_operacion ->> 'prueba_version') !~
               '^[1-9][0-9]{0,9}$')
       OR p_version_canonica IS NULL
       OR octet_length(p_version_canonica) NOT BETWEEN 2 AND 5242880
       OR p_decision_canonica IS NULL
       OR p_recurso_canonico IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'orden de reglas de baremo invalida';
    END IF;

    BEGIN
        v_operacion := p_operacion ->> 'operacion';
        v_intencion_ref := p_operacion ->> 'intencion_ref';
        v_intencion_version :=
            (p_operacion ->> 'intencion_version')::numeric;
        v_intencion_huella := p_operacion ->> 'intencion_huella_sha256';
        v_esperado_revision := CASE
            WHEN p_operacion -> 'esperado_revision' = 'null'::jsonb THEN NULL
            ELSE (p_operacion ->> 'esperado_revision')::numeric
        END;
        v_esperado_huella := p_operacion ->>
            'esperado_huella_estado_sha256';
        v_contenido_ref := p_operacion ->> 'contenido_ref';
        v_contenido_version :=
            (p_operacion ->> 'contenido_version')::numeric;
        v_huella_contenido := p_operacion ->> 'huella_contenido_sha256';
        v_resultado_revision :=
            (p_operacion ->> 'resultado_revision')::numeric;
        v_resultado_estado := p_operacion ->> 'resultado_estado';
        v_resultado_huella := p_operacion ->>
            'resultado_huella_estado_sha256';
        v_prueba_ref := p_operacion ->> 'prueba_ref';
        v_prueba_version := CASE
            WHEN p_operacion -> 'prueba_version' = 'null'::jsonb THEN NULL
            ELSE (p_operacion ->> 'prueba_version')::numeric
        END;
        v_prueba_huella := p_operacion ->> 'prueba_huella_sha256';
        v_accion := p_operacion ->> 'accion';
        v_recurso_ref := p_operacion ->> 'recurso_ref';
        v_correlacion_ref := p_operacion ->> 'correlacion_ref';
        v_huella_contexto := p_operacion ->>
            'huella_contexto_recurso_sha256';
        v_efectuar_en := (p_operacion ->> 'efectuar_en')::timestamptz;
        IF convert_from(p_recurso_canonico, 'UTF8')::jsonb IS DISTINCT FROM
           '{"ambitos":{},"atributos":{}}'::jsonb THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'contexto de recurso no canonico';
        END IF;
    EXCEPTION WHEN data_exception OR invalid_text_representation
        OR numeric_value_out_of_range OR datetime_field_overflow
        OR character_not_in_repertoire
        OR untranslatable_character THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'orden de reglas de baremo invalida';
    END;

    CASE v_operacion
    WHEN 'alta_borrador' THEN
        v_accion_esperada := 'bolsa.reglas_baremo.borrador.crear';
        IF v_resultado_estado <> 'borrador' OR v_resultado_revision <> 1
           OR v_esperado_revision IS NOT NULL
           OR v_esperado_huella IS NOT NULL
           OR v_prueba_ref IS NOT NULL OR v_prueba_version IS NOT NULL
           OR v_prueba_huella IS NOT NULL THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'transicion de reglas de baremo invalida';
        END IF;
    WHEN 'publicar' THEN
        v_accion_esperada := 'bolsa.reglas_baremo.publicar';
        IF v_resultado_estado <> 'publicada'
           OR v_esperado_revision <> 1 OR v_resultado_revision <> 2 THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'transicion de reglas de baremo invalida';
        END IF;
    WHEN 'activar' THEN
        v_accion_esperada := 'bolsa.reglas_baremo.activar';
        IF v_resultado_estado <> 'activa'
           OR v_esperado_revision <> 2 OR v_resultado_revision <> 3 THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'transicion de reglas de baremo invalida';
        END IF;
    WHEN 'sustituir' THEN
        v_accion_esperada := 'bolsa.reglas_baremo.sustituir';
        IF v_resultado_estado <> 'sustituida'
           OR v_esperado_revision <> 3 OR v_resultado_revision <> 4 THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'transicion de reglas de baremo invalida';
        END IF;
    WHEN 'retirar' THEN
        v_accion_esperada := 'bolsa.reglas_baremo.retirar';
        IF v_resultado_estado <> 'retirada'
           OR v_esperado_revision <> 3 OR v_resultado_revision <> 4 THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'transicion de reglas de baremo invalida';
        END IF;
    WHEN 'descartar' THEN
        v_accion_esperada := 'bolsa.reglas_baremo.descartar';
        IF v_resultado_estado <> 'descartada'
           OR v_esperado_revision <> 1 OR v_resultado_revision <> 2 THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'transicion de reglas de baremo invalida';
        END IF;
    ELSE
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'operacion de reglas de baremo desconocida';
    END CASE;

    IF vec_bolsa_reglas_baremo.referencia_valida(v_intencion_ref) IS NOT TRUE
       OR vec_bolsa_reglas_baremo.version_valida(
           v_intencion_version
       ) IS NOT TRUE
       OR vec_bolsa_reglas_baremo.huella_sha256_valida(
           v_intencion_huella
       ) IS NOT TRUE
       OR vec_bolsa_reglas_baremo.referencia_valida(
           v_contenido_ref
       ) IS NOT TRUE
       OR vec_bolsa_reglas_baremo.version_valida(
           v_contenido_version
       ) IS NOT TRUE
       OR vec_bolsa_reglas_baremo.huella_sha256_valida(
           v_huella_contenido
       ) IS NOT TRUE
       OR vec_bolsa_reglas_baremo.huella_sha256_valida(
           v_resultado_huella
       ) IS NOT TRUE
       OR encode(sha256(p_version_canonica), 'hex') <> v_resultado_huella
       OR v_accion IS DISTINCT FROM v_accion_esperada
       OR v_recurso_ref IS DISTINCT FROM
          'reglas-baremo:' || v_resultado_huella
       OR v_correlacion_ref !~ '^correlacion_[0-9a-f]{32}$'
       OR vec_bolsa_reglas_baremo.huella_sha256_valida(
           v_huella_contexto
       ) IS NOT TRUE
       OR encode(sha256(p_recurso_canonico), 'hex') <>
          v_huella_contexto
       OR to_char(v_efectuar_en AT TIME ZONE 'UTC',
              'YYYY-MM-DD"T"HH24:MI:SS.US"Z"') <>
          p_operacion ->> 'efectuar_en'
       OR (v_operacion <> 'alta_borrador' AND (
           vec_bolsa_reglas_baremo.huella_sha256_valida(
               v_esperado_huella
           ) IS NOT TRUE
           OR vec_bolsa_reglas_baremo.referencia_valida(
               v_prueba_ref
           ) IS NOT TRUE
           OR vec_bolsa_reglas_baremo.version_valida(
               v_prueba_version
           ) IS NOT TRUE
           OR vec_bolsa_reglas_baremo.huella_sha256_valida(
               v_prueba_huella
           ) IS NOT TRUE
       )) THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'proyecciones de reglas de baremo invalidas';
    END IF;

    v_ahora := clock_timestamp();
    IF NOT isfinite(v_efectuar_en)
       OR v_ahora < v_efectuar_en
       OR v_ahora - v_efectuar_en > interval '30 seconds' THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'ventana de confirmacion invalida';
    END IF;
    IF vec_autorizacion.revalidar_decision_reglas_baremo_v1(
           p_prueba_autorizacion, p_decision_canonica,
           p_recurso_canonico, v_operacion, v_correlacion_ref,
           v_recurso_ref, v_huella_contexto, v_ahora
       ) IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'autorizacion V2 no revalidada';
    END IF;
    BEGIN
        v_principal_ref := convert_from(
            p_decision_canonica, 'UTF8'
        )::jsonb ->> 'principal_id';
    EXCEPTION WHEN OTHERS THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'decision V2 no interpretable';
    END;
    IF vec_bolsa_reglas_baremo.referencia_valida(
           v_principal_ref
       ) IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'principal V2 invalido';
    END IF;

    PERFORM pg_advisory_xact_lock(hashtextextended(
        v_tenant || ':' || v_contenido_ref || ':' ||
        v_contenido_version::text, 0
    ));

    SELECT confirmacion.*,
           version.version_canonica,
           uso.huella_consumo_autorizacion_sha256,
           auditoria.auditoria_version,
           auditoria.huella_auditoria_sha256,
           salida.outbox_version,
           salida.huella_evento_sha256,
           prueba.prueba_ref,
           prueba.prueba_version,
           prueba.prueba_huella_sha256
      INTO v_existente
      FROM vec_bolsa_reglas_baremo.intencion_confirmada AS confirmacion
      JOIN vec_bolsa_reglas_baremo.version_reglas_baremo AS version
        ON version.tenant_id = confirmacion.tenant_id
       AND version.contenido_ref = confirmacion.contenido_ref
       AND version.contenido_version = confirmacion.contenido_version
       AND version.revision = confirmacion.resultado_revision
       AND version.huella_estado_sha256 =
           confirmacion.resultado_huella_estado_sha256
      JOIN vec_bolsa_reglas_baremo.uso_decision AS uso
        ON uso.tenant_id = confirmacion.tenant_id
       AND uso.decision_ref = confirmacion.decision_ref
      JOIN vec_bolsa_reglas_baremo.auditoria AS auditoria
        ON auditoria.tenant_id = confirmacion.tenant_id
       AND auditoria.auditoria_ref = confirmacion.auditoria_ref
      JOIN vec_bolsa_reglas_baremo.outbox AS salida
        ON salida.tenant_id = confirmacion.tenant_id
       AND salida.outbox_ref = confirmacion.outbox_ref
      LEFT JOIN vec_bolsa_reglas_baremo.uso_prueba_transicion AS prueba
        ON prueba.tenant_id = confirmacion.tenant_id
       AND prueba.consumo_prueba_ref = confirmacion.prueba_consumo_ref
     WHERE confirmacion.tenant_id = v_tenant
       AND confirmacion.intencion_ref = v_intencion_ref
       AND confirmacion.intencion_version = v_intencion_version
     FOR SHARE OF confirmacion, version, uso, auditoria, salida;
    IF FOUND THEN
        IF v_existente.intencion_huella_sha256 <> v_intencion_huella
           OR v_existente.operacion <> v_operacion
           OR v_existente.esperado_revision IS DISTINCT FROM
              v_esperado_revision
           OR v_existente.esperado_huella_estado_sha256 IS DISTINCT FROM
              v_esperado_huella
           OR v_existente.contenido_ref <> v_contenido_ref
           OR v_existente.contenido_version <> v_contenido_version
           OR v_existente.huella_contenido_sha256 <> v_huella_contenido
           OR v_existente.resultado_revision <> v_resultado_revision
           OR v_existente.resultado_estado <> v_resultado_estado
           OR v_existente.resultado_huella_estado_sha256 <>
              v_resultado_huella
           OR v_existente.version_canonica <> p_version_canonica
           OR v_existente.decision_ref <>
              p_prueba_autorizacion ->> 'decision_ref'
           OR v_existente.prueba_ref IS DISTINCT FROM v_prueba_ref
           OR v_existente.prueba_version IS DISTINCT FROM v_prueba_version
           OR v_existente.prueba_huella_sha256 IS DISTINCT FROM
              v_prueba_huella THEN
            RAISE EXCEPTION USING ERRCODE = '23505',
                MESSAGE = 'indice idempotente reutilizado con otra intencion';
        END IF;
        RETURN QUERY SELECT
            'repetida'::text, v_existente.contenido_ref,
            v_existente.contenido_version,
            v_existente.huella_contenido_sha256,
            v_existente.resultado_revision, v_existente.resultado_estado,
            v_existente.resultado_huella_estado_sha256,
            v_existente.transaccion_ref,
            v_existente.transaccion_version,
            v_existente.huella_transaccion_sha256,
            v_existente.auditoria_ref,
            v_existente.auditoria_version,
            v_existente.huella_auditoria_sha256,
            v_existente.outbox_ref, v_existente.outbox_version,
            v_existente.huella_evento_sha256,
            v_existente.consumo_autorizacion_ref,
            1::numeric, v_existente.huella_consumo_autorizacion_sha256,
            v_existente.prueba_consumo_ref,
            v_existente.prueba_consumo_version,
            v_existente.prueba_consumo_huella_sha256,
            v_existente.confirmada_en;
        RETURN;
    END IF;

    IF v_operacion = 'alta_borrador' THEN
        IF EXISTS (
            SELECT 1
              FROM vec_bolsa_reglas_baremo.estado_actual AS existente
             WHERE existente.tenant_id = v_tenant
               AND existente.contenido_ref = v_contenido_ref
               AND existente.contenido_version = v_contenido_version
        ) THEN
            RAISE EXCEPTION USING ERRCODE = '40001',
                MESSAGE = 'conflicto OCC: contenido ya existente';
        END IF;
        INSERT INTO vec_bolsa_reglas_baremo.contenido_reglas_baremo(
            tenant_id, contenido_ref, contenido_version,
            huella_contenido_sha256, creada_en
        ) VALUES (
            v_tenant, v_contenido_ref, v_contenido_version,
            v_huella_contenido, v_ahora
        );
    ELSE
        SELECT actual.*, version.estado
          INTO v_actual
          FROM vec_bolsa_reglas_baremo.estado_actual AS actual
          JOIN vec_bolsa_reglas_baremo.version_reglas_baremo AS version
            ON version.tenant_id = actual.tenant_id
           AND version.contenido_ref = actual.contenido_ref
           AND version.contenido_version = actual.contenido_version
           AND version.revision = actual.revision
           AND version.huella_estado_sha256 = actual.huella_estado_sha256
         WHERE actual.tenant_id = v_tenant
           AND actual.contenido_ref = v_contenido_ref
           AND actual.contenido_version = v_contenido_version
         FOR UPDATE OF actual;
        IF NOT FOUND THEN
            RAISE EXCEPTION USING ERRCODE = 'P0002',
                MESSAGE = 'estado de reglas de baremo no encontrado';
        END IF;
        IF v_actual.huella_contenido_sha256 <> v_huella_contenido
           OR v_actual.revision <> v_esperado_revision
           OR v_actual.huella_estado_sha256 <> v_esperado_huella
           OR (v_operacion = 'publicar' AND v_actual.estado <> 'borrador')
           OR (v_operacion = 'activar' AND v_actual.estado <> 'publicada')
           OR (v_operacion IN ('sustituir', 'retirar')
               AND v_actual.estado <> 'activa')
           OR (v_operacion = 'descartar'
               AND v_actual.estado <> 'borrador') THEN
            RAISE EXCEPTION USING ERRCODE = '40001',
                MESSAGE = 'conflicto OCC en reglas de baremo';
        END IF;
    END IF;

    INSERT INTO vec_bolsa_reglas_baremo.version_reglas_baremo(
        tenant_id, contenido_ref, contenido_version,
        huella_contenido_sha256, revision, estado, version_canonica,
        huella_estado_sha256, operacion_origen, intencion_ref,
        intencion_version, intencion_huella_sha256, registrada_en
    ) VALUES (
        v_tenant, v_contenido_ref, v_contenido_version,
        v_huella_contenido, v_resultado_revision, v_resultado_estado,
        p_version_canonica, v_resultado_huella, v_operacion,
        v_intencion_ref, v_intencion_version, v_intencion_huella, v_ahora
    );
    IF v_operacion = 'alta_borrador' THEN
        INSERT INTO vec_bolsa_reglas_baremo.estado_actual(
            tenant_id, contenido_ref, contenido_version,
            huella_contenido_sha256, revision, huella_estado_sha256,
            actualizada_en
        ) VALUES (
            v_tenant, v_contenido_ref, v_contenido_version,
            v_huella_contenido, v_resultado_revision,
            v_resultado_huella, v_ahora
        );
    ELSE
        UPDATE vec_bolsa_reglas_baremo.estado_actual AS actual
           SET revision = v_resultado_revision,
               huella_estado_sha256 = v_resultado_huella,
               actualizada_en = v_ahora
         WHERE actual.tenant_id = v_tenant
           AND actual.contenido_ref = v_contenido_ref
           AND actual.contenido_version = v_contenido_version
           AND actual.revision = v_esperado_revision
           AND actual.huella_estado_sha256 = v_esperado_huella;
        IF NOT FOUND THEN
            RAISE EXCEPTION USING ERRCODE = '40001',
                MESSAGE = 'conflicto OCC al avanzar reglas de baremo';
        END IF;
    END IF;

    v_efecto := convert_to(jsonb_build_object(
        'esquema', 'vec.bolsa.reglas-baremo.efecto-confirmacion.v1',
        'operacion', v_operacion,
        'intencion_ref', v_intencion_ref,
        'intencion_version', v_intencion_version,
        'intencion_huella_sha256', v_intencion_huella,
        'contenido_ref', v_contenido_ref,
        'contenido_version', v_contenido_version,
        'huella_contenido_sha256', v_huella_contenido,
        'revision', v_resultado_revision,
        'estado', v_resultado_estado,
        'huella_estado_sha256', v_resultado_huella,
        'decision_ref', p_prueba_autorizacion ->> 'decision_ref',
        'confirmada_en', to_char(v_ahora AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
    )::text, 'UTF8');
    v_huella_efecto := encode(sha256(v_efecto), 'hex');

    v_consumo_autorizacion_canonico := convert_to(jsonb_build_object(
        'esquema', 'vec.bolsa.reglas-baremo.consumo-autorizacion.v1',
        'decision_ref', p_prueba_autorizacion ->> 'decision_ref',
        'huella_decision_sha256',
            p_prueba_autorizacion ->> 'huella_decision_sha256',
        'principal_ref', v_principal_ref,
        'operacion', v_operacion,
        'accion', v_accion,
        'recurso_ref', v_recurso_ref,
        'correlacion_ref', v_correlacion_ref,
        'huella_contexto_recurso_sha256', v_huella_contexto,
        'huella_efecto_sha256', v_huella_efecto,
        'consumida_en', to_char(v_ahora AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
    )::text, 'UTF8');
    v_huella_consumo_autorizacion := encode(
        sha256(v_consumo_autorizacion_canonico), 'hex'
    );
    v_consumo_autorizacion_ref := 'consumo-reglas-' ||
        v_huella_consumo_autorizacion;
    INSERT INTO vec_bolsa_reglas_baremo.uso_decision(
        tenant_id, decision_ref, consumo_autorizacion_ref,
        consumo_autorizacion_version,
        huella_consumo_autorizacion_sha256, principal_ref, operacion,
        accion, recurso_ref, correlacion_ref, huella_decision_sha256,
        huella_contexto_recurso_sha256, contenido_ref,
        contenido_version, revision, huella_estado_sha256,
        huella_efecto_sha256, consumida_en
    ) VALUES (
        v_tenant, p_prueba_autorizacion ->> 'decision_ref',
        v_consumo_autorizacion_ref, 1, v_huella_consumo_autorizacion,
        v_principal_ref, v_operacion, v_accion, v_recurso_ref,
        v_correlacion_ref,
        p_prueba_autorizacion ->> 'huella_decision_sha256',
        v_huella_contexto, v_contenido_ref, v_contenido_version,
        v_resultado_revision, v_resultado_huella, v_huella_efecto,
        v_ahora
    );

    IF v_prueba_ref IS NOT NULL THEN
        v_consumo_prueba_canonico := convert_to(jsonb_build_object(
            'esquema', 'vec.bolsa.reglas-baremo.consumo-prueba.v1',
            'prueba_ref', v_prueba_ref,
            'prueba_version', v_prueba_version,
            'prueba_huella_sha256', v_prueba_huella,
            'intencion_ref', v_intencion_ref,
            'intencion_version', v_intencion_version,
            'huella_estado_sha256', v_resultado_huella,
            'consumida_en', to_char(v_ahora AT TIME ZONE 'UTC',
                'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
        )::text, 'UTF8');
        v_huella_consumo_prueba := encode(
            sha256(v_consumo_prueba_canonico), 'hex'
        );
        v_consumo_prueba_ref := 'consumo-prueba-reglas-' ||
            v_huella_consumo_prueba;
        INSERT INTO vec_bolsa_reglas_baremo.uso_prueba_transicion(
            tenant_id, prueba_ref, prueba_version, prueba_huella_sha256,
            consumo_prueba_ref, consumo_prueba_version,
            huella_consumo_prueba_sha256, intencion_ref,
            intencion_version, contenido_ref, contenido_version,
            revision, huella_estado_sha256, consumida_en
        ) VALUES (
            v_tenant, v_prueba_ref, v_prueba_version, v_prueba_huella,
            v_consumo_prueba_ref, 1, v_huella_consumo_prueba,
            v_intencion_ref, v_intencion_version, v_contenido_ref,
            v_contenido_version, v_resultado_revision,
            v_resultado_huella, v_ahora
        );
    END IF;

    v_transaccion_canonica := convert_to(jsonb_build_object(
        'esquema', 'vec.bolsa.reglas-baremo.transaccion.v1',
        'intencion_ref', v_intencion_ref,
        'intencion_version', v_intencion_version,
        'huella_efecto_sha256', v_huella_efecto,
        'consumo_autorizacion_ref', v_consumo_autorizacion_ref,
        'consumo_prueba_ref', v_consumo_prueba_ref,
        'confirmada_en', to_char(v_ahora AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
    )::text, 'UTF8');
    v_huella_transaccion := encode(sha256(v_transaccion_canonica), 'hex');
    v_transaccion_ref := 'transaccion-reglas-' || v_huella_transaccion;

    SELECT ultima_secuencia, ultima_huella_sha256
      INTO STRICT v_ultima_secuencia, v_huella_anterior
      FROM vec_bolsa_reglas_baremo.auditoria_actual
     WHERE tenant_id = v_tenant
     FOR UPDATE;
    v_auditoria_ref := 'auditoria-reglas-' || encode(sha256(convert_to(
        v_consumo_autorizacion_ref || ':' || v_huella_anterior,
        'UTF8'
    )), 'hex');
    v_registro_auditoria := convert_to(jsonb_build_object(
        'esquema', 'vec.bolsa.reglas-baremo.auditoria.v1',
        'auditoria_ref', v_auditoria_ref,
        'secuencia', v_ultima_secuencia + 1,
        'huella_anterior_sha256', v_huella_anterior,
        'operacion', v_operacion,
        'intencion_ref', v_intencion_ref,
        'decision_ref', p_prueba_autorizacion ->> 'decision_ref',
        'consumo_autorizacion_ref', v_consumo_autorizacion_ref,
        'huella_efecto_sha256', v_huella_efecto,
        'registrada_en', to_char(v_ahora AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
    )::text, 'UTF8');
    v_huella_auditoria := encode(sha256(
        decode(v_huella_anterior, 'hex') || v_registro_auditoria
    ), 'hex');
    INSERT INTO vec_bolsa_reglas_baremo.auditoria(
        tenant_id, secuencia, auditoria_ref, auditoria_version,
        decision_ref, consumo_autorizacion_ref, operacion,
        registro_canonico, huella_anterior_sha256,
        huella_auditoria_sha256, registrada_en
    ) VALUES (
        v_tenant, v_ultima_secuencia + 1, v_auditoria_ref, 1,
        p_prueba_autorizacion ->> 'decision_ref',
        v_consumo_autorizacion_ref, v_operacion, v_registro_auditoria,
        v_huella_anterior, v_huella_auditoria, v_ahora
    );
    UPDATE vec_bolsa_reglas_baremo.auditoria_actual
       SET ultima_secuencia = v_ultima_secuencia + 1,
           ultima_huella_sha256 = v_huella_auditoria,
           actualizada_en = v_ahora
     WHERE tenant_id = v_tenant
       AND ultima_secuencia = v_ultima_secuencia
       AND ultima_huella_sha256 = v_huella_anterior;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING ERRCODE = '40001',
            MESSAGE = 'conflicto al avanzar auditoria de reglas de baremo';
    END IF;

    v_evento := convert_to(jsonb_build_object(
        'esquema', 'vec.bolsa.reglas-baremo.estado-confirmado.v1',
        'operacion', v_operacion,
        'contenido_ref', v_contenido_ref,
        'contenido_version', v_contenido_version,
        'huella_contenido_sha256', v_huella_contenido,
        'revision', v_resultado_revision,
        'estado', v_resultado_estado,
        'huella_estado_sha256', v_resultado_huella,
        'transaccion_ref', v_transaccion_ref,
        'auditoria_ref', v_auditoria_ref,
        'confirmada_en', to_char(v_ahora AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
    )::text, 'UTF8');
    v_huella_evento := encode(sha256(v_evento), 'hex');
    v_outbox_ref := 'outbox-reglas-' || v_huella_evento;
    INSERT INTO vec_bolsa_reglas_baremo.outbox(
        tenant_id, outbox_ref, outbox_version, ruta, esquema_evento,
        evento_canonico, huella_evento_sha256, contenido_ref,
        contenido_version, revision, huella_estado_sha256, creada_en
    ) VALUES (
        v_tenant, v_outbox_ref, 1,
        'bolsa.reglas_baremo.estado_confirmado.v1',
        'vec.bolsa.reglas-baremo.estado-confirmado.v1',
        v_evento, v_huella_evento, v_contenido_ref,
        v_contenido_version, v_resultado_revision,
        v_resultado_huella, v_ahora
    );

    INSERT INTO vec_bolsa_reglas_baremo.intencion_confirmada(
        tenant_id, intencion_ref, intencion_version,
        intencion_huella_sha256, operacion, esperado_revision,
        esperado_huella_estado_sha256, contenido_ref,
        contenido_version, huella_contenido_sha256,
        resultado_revision, resultado_estado,
        resultado_huella_estado_sha256, transaccion_ref,
        transaccion_version, huella_transaccion_sha256, decision_ref,
        consumo_autorizacion_ref, prueba_consumo_ref,
        prueba_consumo_version, prueba_consumo_huella_sha256,
        auditoria_ref, outbox_ref, confirmada_en
    ) VALUES (
        v_tenant, v_intencion_ref, v_intencion_version,
        v_intencion_huella, v_operacion, v_esperado_revision,
        v_esperado_huella, v_contenido_ref, v_contenido_version,
        v_huella_contenido, v_resultado_revision, v_resultado_estado,
        v_resultado_huella, v_transaccion_ref, 1,
        v_huella_transaccion,
        p_prueba_autorizacion ->> 'decision_ref',
        v_consumo_autorizacion_ref, v_consumo_prueba_ref,
        CASE WHEN v_consumo_prueba_ref IS NULL THEN NULL ELSE 1 END,
        v_huella_consumo_prueba, v_auditoria_ref, v_outbox_ref, v_ahora
    );

    RETURN QUERY SELECT
        'confirmada'::text, v_contenido_ref, v_contenido_version,
        v_huella_contenido, v_resultado_revision, v_resultado_estado,
        v_resultado_huella, v_transaccion_ref, 1::numeric,
        v_huella_transaccion, v_auditoria_ref, 1::numeric,
        v_huella_auditoria, v_outbox_ref, 1::numeric,
        v_huella_evento, v_consumo_autorizacion_ref, 1::numeric,
        v_huella_consumo_autorizacion, v_consumo_prueba_ref,
        CASE WHEN v_consumo_prueba_ref IS NULL THEN NULL ELSE 1::numeric END,
        v_huella_consumo_prueba, v_ahora;
END
$funcion$;

CREATE FUNCTION vec_bolsa_reglas_baremo.obtener_version_exacta_v1(
    p_consulta jsonb,
    p_prueba_autorizacion jsonb,
    p_decision_canonica bytea,
    p_recurso_canonico bytea
)
RETURNS TABLE (
    version_canonica bytea,
    huella_estado_sha256 text,
    estado text,
    auditoria_ref text,
    auditoria_version numeric,
    huella_auditoria_sha256 text,
    consumo_autorizacion_ref text,
    consumo_autorizacion_version numeric,
    huella_consumo_autorizacion_sha256 text,
    consultada_en timestamptz
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog, pg_temp
SET timezone = 'UTC'
AS $funcion$
DECLARE
    v_tenant constant text := 'diputacion_granada';
    v_contenido_ref text;
    v_contenido_version numeric(20, 0);
    v_huella_contenido text;
    v_revision numeric(20, 0);
    v_huella_estado text;
    v_accion text;
    v_recurso_ref text;
    v_correlacion_ref text;
    v_huella_contexto text;
    v_solicitada_en timestamptz;
    v_ahora timestamptz(6);
    v_principal_ref text;
    v_version record;
    v_efecto bytea;
    v_huella_efecto text;
    v_consumo_canonico bytea;
    v_consumo_ref text;
    v_huella_consumo text;
    v_ultima_secuencia bigint;
    v_huella_anterior text;
    v_auditoria_ref text;
    v_registro_auditoria bytea;
    v_huella_auditoria text;
BEGIN
    IF current_setting('transaction_isolation') <> 'serializable' THEN
        RAISE EXCEPTION USING ERRCODE = '25001',
            MESSAGE = 'consulta rechazada: requiere SERIALIZABLE';
    END IF;
    IF p_consulta IS NULL OR jsonb_typeof(p_consulta) <> 'object'
       OR (SELECT count(*) FROM jsonb_object_keys(p_consulta)) <> 11
       OR NOT (p_consulta ?& ARRAY[
           'esquema', 'contenido_ref', 'contenido_version',
           'huella_contenido_sha256', 'revision',
           'huella_estado_sha256', 'accion', 'recurso_ref',
           'correlacion_ref', 'huella_contexto_recurso_sha256',
           'solicitada_en'
       ])
       OR jsonb_typeof(p_consulta -> 'esquema') <> 'string'
       OR jsonb_typeof(p_consulta -> 'contenido_ref') <> 'string'
       OR jsonb_typeof(p_consulta -> 'contenido_version') <> 'number'
       OR jsonb_typeof(
              p_consulta -> 'huella_contenido_sha256'
          ) <> 'string'
       OR jsonb_typeof(p_consulta -> 'revision') <> 'number'
       OR jsonb_typeof(
              p_consulta -> 'huella_estado_sha256'
          ) <> 'string'
       OR jsonb_typeof(p_consulta -> 'accion') <> 'string'
       OR jsonb_typeof(p_consulta -> 'recurso_ref') <> 'string'
       OR jsonb_typeof(p_consulta -> 'correlacion_ref') <> 'string'
       OR jsonb_typeof(
              p_consulta -> 'huella_contexto_recurso_sha256'
          ) <> 'string'
       OR jsonb_typeof(p_consulta -> 'solicitada_en') <> 'string'
       OR p_consulta ->> 'esquema' IS DISTINCT FROM
          'vec.bolsa.reglas-baremo.consulta-postgresql.v1'
       OR (p_consulta ->> 'contenido_version') !~ '^[1-9][0-9]{0,9}$'
       OR (p_consulta ->> 'revision') !~ '^[1-9][0-9]{0,9}$' THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'consulta exacta de reglas de baremo invalida';
    END IF;
    BEGIN
        v_contenido_ref := p_consulta ->> 'contenido_ref';
        v_contenido_version :=
            (p_consulta ->> 'contenido_version')::numeric;
        v_huella_contenido := p_consulta ->> 'huella_contenido_sha256';
        v_revision := (p_consulta ->> 'revision')::numeric;
        v_huella_estado := p_consulta ->> 'huella_estado_sha256';
        v_accion := p_consulta ->> 'accion';
        v_recurso_ref := p_consulta ->> 'recurso_ref';
        v_correlacion_ref := p_consulta ->> 'correlacion_ref';
        v_huella_contexto := p_consulta ->>
            'huella_contexto_recurso_sha256';
        v_solicitada_en := (p_consulta ->> 'solicitada_en')::timestamptz;
        IF convert_from(p_recurso_canonico, 'UTF8')::jsonb IS DISTINCT FROM
           '{"ambitos":{},"atributos":{}}'::jsonb THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'contexto de recurso no canonico';
        END IF;
    EXCEPTION WHEN data_exception OR invalid_text_representation
        OR numeric_value_out_of_range OR datetime_field_overflow
        OR character_not_in_repertoire
        OR untranslatable_character THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'consulta exacta de reglas de baremo invalida';
    END;
    IF vec_bolsa_reglas_baremo.referencia_valida(
           v_contenido_ref
       ) IS NOT TRUE
       OR vec_bolsa_reglas_baremo.version_valida(
           v_contenido_version
       ) IS NOT TRUE
       OR vec_bolsa_reglas_baremo.version_valida(v_revision) IS NOT TRUE
       OR vec_bolsa_reglas_baremo.huella_sha256_valida(
           v_huella_contenido
       ) IS NOT TRUE
       OR vec_bolsa_reglas_baremo.huella_sha256_valida(
           v_huella_estado
       ) IS NOT TRUE
       OR v_accion <> 'bolsa.reglas_baremo.version.consultar'
       OR v_recurso_ref <> 'reglas-baremo:' || v_huella_estado
       OR v_correlacion_ref !~ '^correlacion_[0-9a-f]{32}$'
       OR vec_bolsa_reglas_baremo.huella_sha256_valida(
           v_huella_contexto
       ) IS NOT TRUE
       OR encode(sha256(p_recurso_canonico), 'hex') <>
          v_huella_contexto
       OR to_char(v_solicitada_en AT TIME ZONE 'UTC',
              'YYYY-MM-DD"T"HH24:MI:SS.US"Z"') <>
          p_consulta ->> 'solicitada_en' THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'selector exacto de reglas de baremo invalido';
    END IF;
    v_ahora := clock_timestamp();
    IF NOT isfinite(v_solicitada_en)
       OR v_ahora < v_solicitada_en
       OR v_ahora - v_solicitada_en > interval '30 seconds' THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'ventana de consulta invalida';
    END IF;
    IF vec_autorizacion.revalidar_decision_reglas_baremo_v1(
           p_prueba_autorizacion, p_decision_canonica,
           p_recurso_canonico, 'consultar_version_exacta',
           v_correlacion_ref, v_recurso_ref, v_huella_contexto, v_ahora
       ) IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'autorizacion V2 no revalidada';
    END IF;
    BEGIN
        v_principal_ref := convert_from(
            p_decision_canonica, 'UTF8'
        )::jsonb ->> 'principal_id';
    EXCEPTION WHEN OTHERS THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'decision V2 no interpretable';
    END;

    SELECT almacen.version_canonica, almacen.huella_estado_sha256,
           almacen.estado
      INTO v_version
      FROM vec_bolsa_reglas_baremo.version_reglas_baremo AS almacen
     WHERE almacen.tenant_id = v_tenant
       AND almacen.contenido_ref = v_contenido_ref
       AND almacen.contenido_version = v_contenido_version
       AND almacen.huella_contenido_sha256 = v_huella_contenido
       AND almacen.revision = v_revision
       AND almacen.huella_estado_sha256 = v_huella_estado
     FOR SHARE OF almacen;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING ERRCODE = 'P0002',
            MESSAGE = 'estado exacto de reglas de baremo no encontrado';
    END IF;

    v_efecto := convert_to(jsonb_build_object(
        'esquema', 'vec.bolsa.reglas-baremo.efecto-consulta.v1',
        'contenido_ref', v_contenido_ref,
        'contenido_version', v_contenido_version,
        'revision', v_revision,
        'huella_estado_sha256', v_huella_estado,
        'decision_ref', p_prueba_autorizacion ->> 'decision_ref',
        'consultada_en', to_char(v_ahora AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
    )::text, 'UTF8');
    v_huella_efecto := encode(sha256(v_efecto), 'hex');
    v_consumo_canonico := convert_to(jsonb_build_object(
        'esquema', 'vec.bolsa.reglas-baremo.consumo-autorizacion.v1',
        'decision_ref', p_prueba_autorizacion ->> 'decision_ref',
        'huella_decision_sha256',
            p_prueba_autorizacion ->> 'huella_decision_sha256',
        'principal_ref', v_principal_ref,
        'operacion', 'consultar_version_exacta',
        'accion', v_accion,
        'recurso_ref', v_recurso_ref,
        'correlacion_ref', v_correlacion_ref,
        'huella_contexto_recurso_sha256', v_huella_contexto,
        'huella_efecto_sha256', v_huella_efecto,
        'consumida_en', to_char(v_ahora AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
    )::text, 'UTF8');
    v_huella_consumo := encode(sha256(v_consumo_canonico), 'hex');
    v_consumo_ref := 'consumo-reglas-' || v_huella_consumo;
    INSERT INTO vec_bolsa_reglas_baremo.uso_decision(
        tenant_id, decision_ref, consumo_autorizacion_ref,
        consumo_autorizacion_version,
        huella_consumo_autorizacion_sha256, principal_ref, operacion,
        accion, recurso_ref, correlacion_ref, huella_decision_sha256,
        huella_contexto_recurso_sha256, contenido_ref,
        contenido_version, revision, huella_estado_sha256,
        huella_efecto_sha256, consumida_en
    ) VALUES (
        v_tenant, p_prueba_autorizacion ->> 'decision_ref',
        v_consumo_ref, 1, v_huella_consumo, v_principal_ref,
        'consultar_version_exacta', v_accion, v_recurso_ref,
        v_correlacion_ref,
        p_prueba_autorizacion ->> 'huella_decision_sha256',
        v_huella_contexto, v_contenido_ref, v_contenido_version,
        v_revision, v_huella_estado, v_huella_efecto, v_ahora
    );

    SELECT ultima_secuencia, ultima_huella_sha256
      INTO STRICT v_ultima_secuencia, v_huella_anterior
      FROM vec_bolsa_reglas_baremo.auditoria_actual
     WHERE tenant_id = v_tenant
     FOR UPDATE;
    v_auditoria_ref := 'auditoria-reglas-' || encode(sha256(convert_to(
        v_consumo_ref || ':' || v_huella_anterior, 'UTF8'
    )), 'hex');
    v_registro_auditoria := convert_to(jsonb_build_object(
        'esquema', 'vec.bolsa.reglas-baremo.auditoria.v1',
        'auditoria_ref', v_auditoria_ref,
        'secuencia', v_ultima_secuencia + 1,
        'huella_anterior_sha256', v_huella_anterior,
        'operacion', 'consultar_version_exacta',
        'decision_ref', p_prueba_autorizacion ->> 'decision_ref',
        'consumo_autorizacion_ref', v_consumo_ref,
        'huella_efecto_sha256', v_huella_efecto,
        'registrada_en', to_char(v_ahora AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
    )::text, 'UTF8');
    v_huella_auditoria := encode(sha256(
        decode(v_huella_anterior, 'hex') || v_registro_auditoria
    ), 'hex');
    INSERT INTO vec_bolsa_reglas_baremo.auditoria(
        tenant_id, secuencia, auditoria_ref, auditoria_version,
        decision_ref, consumo_autorizacion_ref, operacion,
        registro_canonico, huella_anterior_sha256,
        huella_auditoria_sha256, registrada_en
    ) VALUES (
        v_tenant, v_ultima_secuencia + 1, v_auditoria_ref, 1,
        p_prueba_autorizacion ->> 'decision_ref', v_consumo_ref,
        'consultar_version_exacta', v_registro_auditoria,
        v_huella_anterior, v_huella_auditoria, v_ahora
    );
    UPDATE vec_bolsa_reglas_baremo.auditoria_actual
       SET ultima_secuencia = v_ultima_secuencia + 1,
           ultima_huella_sha256 = v_huella_auditoria,
           actualizada_en = v_ahora
     WHERE tenant_id = v_tenant
       AND ultima_secuencia = v_ultima_secuencia
       AND ultima_huella_sha256 = v_huella_anterior;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING ERRCODE = '40001',
            MESSAGE = 'conflicto al avanzar auditoria de consulta';
    END IF;

    RETURN QUERY SELECT
        v_version.version_canonica::bytea,
        v_version.huella_estado_sha256::text,
        v_version.estado::text, v_auditoria_ref, 1::numeric,
        v_huella_auditoria, v_consumo_ref, 1::numeric,
        v_huella_consumo, v_ahora;
END
$funcion$;

REVOKE ALL ON FUNCTION
    vec_bolsa_reglas_baremo.confirmar_cambio_v1(
        jsonb, jsonb, bytea, bytea, bytea
    ) FROM PUBLIC;
REVOKE ALL ON FUNCTION
    vec_bolsa_reglas_baremo.obtener_version_exacta_v1(
        jsonb, jsonb, bytea, bytea
    ) FROM PUBLIC;

COMMENT ON FUNCTION vec_bolsa_reglas_baremo.confirmar_cambio_v1(
    jsonb, jsonb, bytea, bytea, bytea
) IS
    'CAS e idempotencia atomicos para reglas gobernadas; cerrada hasta componer el consumo VEC-AD-2 y estabilizar el contrato application.';
COMMENT ON FUNCTION vec_bolsa_reglas_baremo.obtener_version_exacta_v1(
    jsonb, jsonb, bytea, bytea
) IS
    'Consulta historica exacta con consumo y auditoria; cerrada hasta componer el consumo VEC-AD-2 y estabilizar el contrato application.';
COMMIT;
