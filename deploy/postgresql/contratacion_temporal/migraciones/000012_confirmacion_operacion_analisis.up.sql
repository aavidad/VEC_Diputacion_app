BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:000012_confirmacion_operacion_analisis', 0
    )
);

DO $prevalidacion$
BEGIN
    IF pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.entrada_confirmacion_analisis_valida_v1(jsonb)'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.transicion_confirmacion_analisis_valida_v1(jsonb,jsonb)'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_autorizacion.revalidar_decision_analisis_contratacion_temporal_v1(bytea,bytea,numeric,numeric,jsonb)'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.confirmar_operacion_analisis_v1(jsonb)'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'dependencias de confirmación de análisis ausentes';
    END IF;
END
$prevalidacion$;

CREATE FUNCTION
vec_contratacion_temporal.confirmar_operacion_analisis_v1(
    o jsonb
)
RETURNS TABLE (recibo_json jsonb)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
SET lock_timeout = '2s'
AS $funcion$
DECLARE
    r vec_contratacion_temporal.reserva_operacion_analisis%ROWTYPE;
    v_estado text;
    v_revision bigint;
    v_agregado_actual jsonb;
    v_version_actual numeric;
    v_ahora timestamptz(6);
    v_decision_ref text;
    v_decision_huella text;
    v_prueba_fuentes bytea;
    v_operacion_ref text;
    v_consumo_ref text;
    v_auditoria_ref text;
    v_evento_ref text;
    v_agregado_huella text;
    v_prueba_version bytea;
    v_prueba_actuacion bytea;
    v_prueba_auditoria bytea;
    v_payload_evento bytea;
    v_huella_payload text;
    v_anterior_auditoria text;
    v_anterior_outbox text;
    v_secuencia_auditoria numeric;
    v_secuencia_outbox numeric;
    v_huella_auditoria text;
    v_huella_outbox text;
    v_recibo jsonb;
    v_statement_ms numeric;
    v_idle_ms numeric;
    v_fuente jsonb;
    v_alias jsonb;
BEGIN
    IF session_user = current_user
       OR NOT pg_catalog.pg_has_role(
           session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER'
       )
       OR pg_catalog.pg_has_role(
           session_user, 'vec_contratacion_temporal_propietario', 'MEMBER'
       )
       OR pg_catalog.pg_has_role(
           session_user, 'vec_contratacion_temporal_migrador', 'MEMBER'
       )
       OR pg_catalog.current_setting('transaction_isolation') <>
          'serializable'
       OR pg_catalog.current_setting('transaction_read_only') <> 'off' THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'identidad o transacción de confirmación no autorizada';
    END IF;
    SELECT CASE WHEN unit = 'ms' AND setting ~ '^[0-9]{1,18}$'
                THEN setting::numeric END
      INTO v_statement_ms
      FROM pg_catalog.pg_settings
     WHERE name = 'statement_timeout';
    SELECT CASE WHEN unit = 'ms' AND setting ~ '^[0-9]{1,18}$'
                THEN setting::numeric END
      INTO v_idle_ms
      FROM pg_catalog.pg_settings
     WHERE name = 'idle_in_transaction_session_timeout';
    IF v_statement_ms IS NULL OR v_statement_ms NOT BETWEEN 1 AND 15000
       OR v_idle_ms IS NULL OR v_idle_ms NOT BETWEEN 1 AND 20000
       OR o IS NULL
       OR vec_contratacion_temporal.entrada_confirmacion_analisis_valida_v1(o)
          IS NOT TRUE THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'confirmación de análisis inválida';
    END IF;
    SELECT base.* INTO STRICT r
      FROM vec_contratacion_temporal.reserva_operacion_analisis base
     WHERE base.ambito_raiz_hmac = o ->> 'ambito_raiz_hmac'
     FOR UPDATE;
    SELECT actual.revision, version.estado
      INTO STRICT v_revision, v_estado
      FROM vec_contratacion_temporal.reserva_operacion_analisis_actual actual
      JOIN vec_contratacion_temporal.reserva_operacion_analisis_version version
        USING (ambito_raiz_hmac, revision)
     WHERE actual.ambito_raiz_hmac = r.ambito_raiz_hmac
     FOR UPDATE OF actual, version;
    IF r.reserva_ref <> o ->> 'reserva_ref'
       OR r.recibo_ref <> o ->> 'recibo_ref'
       OR r.operacion <> o ->> 'operacion'
       OR r.organizacion_ref <> o ->> 'organizacion_ref'
       OR r.expediente_ref <> o ->> 'expediente_ref'
       OR r.version_expediente <> (o ->> 'version_anterior')::numeric
       OR r.actor_ref <> o ->> 'actor_ref'
       OR r.perfil_ref <> o ->> 'perfil_ref'
       OR r.artefacto_ref <> o ->> 'artefacto_ref'
       OR r.artefacto_huella_sha256 <>
            o ->> 'artefacto_huella_sha256'
       OR r.huella_semantica_raiz_hmac <>
            o ->> 'huella_semantica_hmac' THEN
        RAISE EXCEPTION USING
            ERRCODE = '23505',
            MESSAGE = 'reserva de análisis reutilizada con otros datos';
    END IF;
    IF v_estado = 'confirmada' THEN
        SELECT c.recibo_json INTO STRICT v_recibo
          FROM vec_contratacion_temporal.confirmacion_operacion_analisis c
         WHERE c.ambito_raiz_hmac = r.ambito_raiz_hmac;
        IF v_recibo ->> 'recibo_ref' <> r.recibo_ref
           OR v_recibo ->> 'huella_consumo_fuentes_sha256' <>
              o #>> '{fuentes,conjunto_huella_sha256}'
           OR v_recibo ->> 'concesion_v3_decision_ref' <>
              o #>> '{autorizacion,decision_ref}'
           OR NOT EXISTS (
               SELECT 1
                 FROM vec_contratacion_temporal.consumo_fuentes_analisis f
                WHERE f.ambito_raiz_hmac = r.ambito_raiz_hmac
                  AND f.conjunto_huella_sha256 =
                      o #>> '{fuentes,conjunto_huella_sha256}'
           )
           OR NOT EXISTS (
               SELECT 1
                 FROM vec_contratacion_temporal.consumo_decision_analisis d
                WHERE d.ambito_raiz_hmac = r.ambito_raiz_hmac
                  AND d.decision_ref = o #>> '{autorizacion,decision_ref}'
                  AND d.decision_huella_sha256 =
                      o #>> '{autorizacion,decision_huella_sha256}'
           ) THEN
            RAISE EXCEPTION USING
                ERRCODE = '23505',
                MESSAGE = 'replay de análisis divergente';
        END IF;
        RETURN QUERY SELECT v_recibo;
        RETURN;
    END IF;
    IF v_estado <> 'reservada' OR v_revision <> 1 THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'estado de reserva de análisis incompatible';
    END IF;
    SELECT actual.version, version.agregado_json
      INTO STRICT v_version_actual, v_agregado_actual
      FROM vec_contratacion_temporal.expediente_integral_actual actual
      JOIN vec_contratacion_temporal.expediente_version_integral version
        USING (expediente_ref, version)
     WHERE actual.expediente_ref = r.expediente_ref
     FOR UPDATE OF actual, version;
    IF v_version_actual <> r.version_expediente
       OR vec_contratacion_temporal.transicion_confirmacion_analisis_valida_v1(
           o, v_agregado_actual
       ) IS NOT TRUE THEN
        RAISE EXCEPTION USING
            ERRCODE = '40001',
            MESSAGE = 'versión o transición de análisis en conflicto';
    END IF;
    IF o #>> '{autorizacion,accion}' <>
           o #>> '{politica,accion}'
       OR o #>> '{autorizacion,finalidad}' <>
           o #>> '{politica,finalidad}'
       OR o #>> '{autorizacion,recurso_ref}' <> r.expediente_ref
       OR o #>> '{autorizacion,principal_id}' <> r.actor_ref
       OR o #>> '{autorizacion,perfil_activo_ref}' <> r.perfil_ref
       OR o #>> '{autorizacion,contexto_recurso_huella_sha256}' <>
          vec_contratacion_temporal.huella_contexto_recurso_analisis_v1(o)
       OR pg_catalog.encode(
           pg_catalog.sha256(pg_catalog.decode(
               o #>> '{autorizacion,decision_canonica_hex}', 'hex'
           )), 'hex'
       ) <> o #>> '{autorizacion,decision_huella_sha256}'
       OR substring(
           o ->> 'ambito_consulta_hmac'
           FROM '/v([1-9][0-9]{0,8}):'
       ) <> substring(
           o ->> 'huella_consulta_hmac'
           FROM '/v([1-9][0-9]{0,8}):'
       ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'vínculo de autorización de análisis inválido';
    END IF;
    SELECT revalidada_en, decision_ref, decision_huella_sha256
      INTO v_ahora, v_decision_ref, v_decision_huella
      FROM vec_autorizacion
           .revalidar_decision_analisis_contratacion_temporal_v1(
          pg_catalog.decode(
              o #>> '{autorizacion,decision_canonica_hex}', 'hex'
          ),
          pg_catalog.decode(
              o #>> '{autorizacion,motivo_canonico_hex}', 'hex'
          ),
          (o #>> '{autorizacion,persona_version}')::numeric,
          (o #>> '{autorizacion,perfil_version}')::numeric,
          pg_catalog.jsonb_build_object(
              'accion', o #>> '{autorizacion,accion}',
              'contexto_recurso_huella_sha256',
                  o #>> '{autorizacion,contexto_recurso_huella_sha256}',
              'decision_ref', o #>> '{autorizacion,decision_ref}',
              'finalidad', o #>> '{autorizacion,finalidad}',
              'perfil_activo_ref',
                  o #>> '{autorizacion,perfil_activo_ref}',
              'principal_id', o #>> '{autorizacion,principal_id}',
              'recurso_ref', o #>> '{autorizacion,recurso_ref}'
          )
      );
    IF v_ahora IS NULL
       OR v_decision_ref <> o #>> '{autorizacion,decision_ref}'
       OR v_decision_huella <>
            o #>> '{autorizacion,decision_huella_sha256}' THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'decisión VEC de análisis no vigente';
    END IF;
    FOR v_fuente IN
        SELECT e.v
          FROM pg_catalog.jsonb_array_elements(
              CASE WHEN o #> '{fuentes,coste}' = 'null'::jsonb
                   THEN pg_catalog.jsonb_build_array(
                       o #> '{fuentes,rc}'
                   )
                   ELSE pg_catalog.jsonb_build_array(
                       o #> '{fuentes,rc}', o #> '{fuentes,coste}'
                   ) END
          ) AS e(v)
    LOOP
        IF (v_fuente ->> 'verificada_en')::timestamptz > v_ahora
           OR v_ahora >=
              (v_fuente ->> 'valida_hasta')::timestamptz
           OR (
               v_fuente -> 'publicacion' <> 'null'::jsonb
               AND (v_fuente #>> '{publicacion,verificada_en}')::timestamptz
                   > v_ahora
           ) THEN
            RAISE EXCEPTION USING
                ERRCODE = '22023',
                MESSAGE = 'fuente de análisis caducada o futura';
        END IF;
    END LOOP;
    v_prueba_fuentes :=
        vec_contratacion_temporal.reconstruir_prueba_fuentes_analisis_v1(o);
    v_operacion_ref := 'operacion:analisis:' || pg_catalog.substr(
        pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
            r.ambito_raiz_hmac || ':' || r.recibo_ref, 'UTF8'
        )), 'hex'), 1, 32
    );
    v_consumo_ref := 'consumo:fuentes:analisis:' || pg_catalog.substr(
        pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
            'fuentes:' || r.ambito_raiz_hmac, 'UTF8'
        )), 'hex'), 1, 32
    );
    v_auditoria_ref := 'auditoria:analisis:' || pg_catalog.substr(
        pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
            'auditoria:' || r.ambito_raiz_hmac, 'UTF8'
        )), 'hex'), 1, 32
    );
    v_evento_ref := 'evento:analisis:' || pg_catalog.substr(
        pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
            'evento:' || r.ambito_raiz_hmac, 'UTF8'
        )), 'hex'), 1, 32
    );
    v_agregado_huella := pg_catalog.encode(pg_catalog.sha256(
        pg_catalog.convert_to(
            (o -> 'expediente_siguiente')::text, 'UTF8'
        )
    ), 'hex');
    v_prueba_version := pg_catalog.convert_to(
        'VEC-CT-EXPEDIENTE-INTEGRAL-V1' || chr(10), 'UTF8'
    ) || vec_contratacion_temporal.encuadrar_texto_v1(r.expediente_ref)
      || vec_contratacion_temporal.encuadrar_texto_v1(
          (r.version_expediente + 1)::text
      )
      || vec_contratacion_temporal.encuadrar_texto_v1(v_agregado_huella)
      || vec_contratacion_temporal.encuadrar_texto_v1(
          o #>> '{expediente_siguiente,flujo,definicion_ref}'
      )
      || vec_contratacion_temporal.encuadrar_texto_v1(
          o #>> '{expediente_siguiente,flujo,version}'
      )
      || vec_contratacion_temporal.encuadrar_texto_v1(
          o #>> '{expediente_siguiente,flujo,huella_sha256}'
      )
      || vec_contratacion_temporal.encuadrar_texto_v1(
          o #>> '{expediente_siguiente,fase_actual}'
      )
      || vec_contratacion_temporal.encuadrar_texto_v1(
          o #>> '{expediente_siguiente,estado_actual}'
      )
      || vec_contratacion_temporal.encuadrar_texto_v1('analisis_o3')
      || vec_contratacion_temporal.encuadrar_texto_v1(v_operacion_ref)
      || vec_contratacion_temporal.encuadrar_texto_v1(
          vec_contratacion_temporal.instante_utc_v1(v_ahora)
      );
    INSERT INTO vec_contratacion_temporal.expediente_version_integral
    VALUES (
        r.expediente_ref, r.version_expediente + 1,
        o -> 'expediente_siguiente', v_agregado_huella,
        v_prueba_version,
        pg_catalog.encode(pg_catalog.sha256(v_prueba_version), 'hex'),
        o #>> '{expediente_siguiente,flujo,definicion_ref}',
        (o #>> '{expediente_siguiente,flujo,version}')::numeric,
        o #>> '{expediente_siguiente,flujo,huella_sha256}',
        o #>> '{expediente_siguiente,fase_actual}',
        o #>> '{expediente_siguiente,estado_actual}',
        'analisis_o3', v_operacion_ref, v_ahora
    );
    UPDATE vec_contratacion_temporal.expediente_integral_actual
       SET version = r.version_expediente + 1,
           actualizada_en = v_ahora,
           operacion_ref = v_operacion_ref
     WHERE expediente_ref = r.expediente_ref
       AND version = r.version_expediente;
    v_prueba_actuacion := pg_catalog.convert_to(
        'VEC-CT-ACTUACION-INTEGRAL-V1' || chr(10), 'UTF8'
    ) || vec_contratacion_temporal.encuadrar_texto_v1(r.expediente_ref)
      || vec_contratacion_temporal.encuadrar_texto_v1(
          (r.version_expediente + 1)::text
      )
      || vec_contratacion_temporal.encuadrar_texto_v1(v_operacion_ref)
      || vec_contratacion_temporal.encuadrar_texto_v1(r.recibo_ref)
      || vec_contratacion_temporal.encuadrar_texto_v1(
          pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
              (o -> 'actuacion')::text, 'UTF8'
          )), 'hex')
      );
    INSERT INTO vec_contratacion_temporal.actuacion_expediente_integral
    VALUES (
        r.expediente_ref, r.version_expediente + 1,
        r.version_expediente + 1, v_operacion_ref, r.recibo_ref,
        o -> 'actuacion',
        pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
            (o -> 'actuacion')::text, 'UTF8'
        )), 'hex'),
        v_prueba_actuacion,
        pg_catalog.encode(pg_catalog.sha256(v_prueba_actuacion), 'hex'),
        v_ahora
    );
    INSERT INTO vec_contratacion_temporal.consumo_fuentes_analisis
    VALUES (
        v_consumo_ref, o #>> '{fuentes,conjunto_huella_sha256}',
        r.ambito_raiz_hmac, r.organizacion_ref, r.expediente_ref,
        r.version_expediente + 1, r.artefacto_ref,
        r.artefacto_huella_sha256, o -> 'fuentes',
        pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
            (o -> 'fuentes')::text, 'UTF8'
        )), 'hex'), v_ahora, v_prueba_fuentes
    );
    INSERT INTO vec_contratacion_temporal.consumo_decision_analisis
    VALUES (
        v_decision_ref, v_decision_huella, r.ambito_raiz_hmac, v_ahora
    );
    SELECT secuencia_auditoria, cabeza_auditoria_sha256,
           secuencia_outbox, cabeza_outbox_sha256
      INTO STRICT v_secuencia_auditoria, v_anterior_auditoria,
                  v_secuencia_outbox, v_anterior_outbox
      FROM vec_contratacion_temporal.control_cadenas_expediente_integral
     WHERE control_id FOR UPDATE;
    v_secuencia_auditoria := v_secuencia_auditoria + 1;
    v_secuencia_outbox := v_secuencia_outbox + 1;
    v_prueba_auditoria := pg_catalog.convert_to(
        'VEC-CT-AUDITORIA-ANALISIS-V1' || chr(10), 'UTF8'
    ) || vec_contratacion_temporal.encuadrar_texto_v1(
          v_secuencia_auditoria::text
      )
      || vec_contratacion_temporal.encuadrar_texto_v1(v_auditoria_ref)
      || vec_contratacion_temporal.encuadrar_texto_v1(v_operacion_ref)
      || vec_contratacion_temporal.encuadrar_texto_v1(r.expediente_ref)
      || vec_contratacion_temporal.encuadrar_texto_v1(v_decision_ref)
      || vec_contratacion_temporal.encuadrar_texto_v1(v_consumo_ref)
      || vec_contratacion_temporal.encuadrar_texto_v1(v_anterior_auditoria);
    v_huella_auditoria := pg_catalog.encode(pg_catalog.sha256(
        v_anterior_auditoria::bytea || v_prueba_auditoria
    ), 'hex');
    INSERT INTO vec_contratacion_temporal.auditoria_expediente_integral
    VALUES (
        v_auditoria_ref, v_secuencia_auditoria, v_operacion_ref,
        r.expediente_ref, r.version_expediente + 1, v_decision_ref,
        v_consumo_ref, v_prueba_auditoria, v_anterior_auditoria,
        v_huella_auditoria, v_ahora
    );
    v_payload_evento := pg_catalog.convert_to(
        pg_catalog.jsonb_build_object(
            'evento_ref', v_evento_ref,
            'tipo', 'contratacion_temporal.analisis_confirmado',
            'expediente_ref', r.expediente_ref,
            'version', r.version_expediente + 1,
            'operacion_ref', v_operacion_ref,
            'confirmada_en',
                vec_contratacion_temporal.instante_utc_v1(v_ahora)
        )::text, 'UTF8'
    );
    v_huella_payload := pg_catalog.encode(
        pg_catalog.sha256(v_payload_evento), 'hex'
    );
    v_huella_outbox := pg_catalog.encode(pg_catalog.sha256(
        v_anterior_outbox::bytea || v_payload_evento
    ), 'hex');
    INSERT INTO vec_contratacion_temporal.outbox_expediente_integral
    VALUES (
        v_evento_ref, v_secuencia_outbox, v_operacion_ref,
        r.expediente_ref, r.version_expediente + 1,
        'contratacion_temporal.analisis_confirmado',
        v_payload_evento, v_huella_payload, v_anterior_outbox,
        v_huella_outbox, v_ahora
    );
    UPDATE vec_contratacion_temporal.control_cadenas_expediente_integral
       SET secuencia_auditoria = v_secuencia_auditoria,
           cabeza_auditoria_sha256 = v_huella_auditoria,
           secuencia_outbox = v_secuencia_outbox,
           cabeza_outbox_sha256 = v_huella_outbox,
           actualizada_en = v_ahora
     WHERE control_id;
    v_recibo := pg_catalog.jsonb_build_object(
        'operacion', r.operacion,
        'organizacion_ref', r.organizacion_ref,
        'expediente_ref', r.expediente_ref,
        'version_anterior', r.version_expediente,
        'version_resultante', r.version_expediente + 1,
        'secuencia_actuacion', r.version_expediente + 1,
        'artefacto_ref', r.artefacto_ref,
        'artefacto_huella_sha256', r.artefacto_huella_sha256,
        'recibo_ref', r.recibo_ref,
        'auditoria_ref', v_auditoria_ref,
        'evento_ref', v_evento_ref,
        'consumo_fuentes_ref', v_consumo_ref,
        'huella_consumo_fuentes_sha256',
            o #>> '{fuentes,conjunto_huella_sha256}',
        'concesion_v3_decision_ref', v_decision_ref,
        'huella_semantica_hmac', r.huella_semantica_raiz_hmac,
        'ambito_consulta_hmac', o ->> 'ambito_consulta_hmac',
        'huella_consulta_hmac', o ->> 'huella_consulta_hmac',
        'confirmada_en',
            vec_contratacion_temporal.instante_utc_v1(v_ahora)
    );
    INSERT INTO vec_contratacion_temporal.confirmacion_operacion_analisis
    VALUES (
        r.ambito_raiz_hmac, v_recibo,
        pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
            v_recibo::text, 'UTF8'
        )), 'hex'), v_ahora
    );
    FOR v_alias IN
        SELECT e.v
          FROM pg_catalog.jsonb_array_elements(
              o -> 'aliases_consulta'
          ) AS e(v)
    LOOP
        INSERT INTO
        vec_contratacion_temporal.alias_consulta_operacion_analisis
        VALUES (
            v_alias ->> 'ambito_hmac', r.ambito_raiz_hmac,
            (v_alias ->> 'generacion')::integer, v_ahora
        );
    END LOOP;
    INSERT INTO vec_contratacion_temporal.reserva_operacion_analisis_version
    VALUES (
        r.ambito_raiz_hmac, 2, 'confirmada', v_ahora, v_ahora
    );
    UPDATE vec_contratacion_temporal.reserva_operacion_analisis_actual
       SET revision = 2
     WHERE ambito_raiz_hmac = r.ambito_raiz_hmac
       AND revision = 1;
    RETURN QUERY SELECT v_recibo;
END
$funcion$;

REVOKE ALL ON FUNCTION
vec_contratacion_temporal.confirmar_operacion_analisis_v1(jsonb)
FROM PUBLIC;
GRANT EXECUTE ON FUNCTION
vec_contratacion_temporal.confirmar_operacion_analisis_v1(jsonb)
TO vec_contratacion_temporal_ejecutor;

COMMIT;
