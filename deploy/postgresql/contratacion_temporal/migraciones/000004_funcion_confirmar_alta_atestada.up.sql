BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:000004_confirmar_alta_atestada', 0
    )
);

DO $prevalidacion$
BEGIN
    IF pg_catalog.to_regclass(
           'vec_contratacion_temporal.expediente_alta'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.confirmar_alta_atestada_v1(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea)'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'estado incompatible para confirmación atestada';
    END IF;
END
$prevalidacion$;

CREATE FUNCTION vec_contratacion_temporal.confirmar_alta_atestada_v1(
    p_capacidad_canonica bytea,
    p_decision_canonica bytea,
    p_motivo_canonico bytea,
    p_contexto_actor_canonico bytea,
    p_persona_version numeric,
    p_perfil_version numeric,
    p_payload_vec_ad_3 bytea,
    p_sobre_cose_sign1 bytea,
    p_evidencia_verificacion bytea,
    p_raiz_publica_spki bytea,
    p_alta_canonica bytea,
    p_sellos_hmac_canonicos bytea
)
RETURNS TABLE (
    expediente_ref text,
    numero_visible text,
    version numeric,
    recibo_ref text,
    auditoria_ref text,
    evento_ref text,
    confirmada_en timestamptz,
    recibo_huella_sha256 text
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
SET lock_timeout = '2s'
AS $funcion$
DECLARE
    a jsonb;
    s jsonb;
    d jsonb;
    v_consumo record;
    v_identidad record;
    v_existente record;
    v_par jsonb;
    v_pares jsonb;
    v_generaciones integer[];
    v_generaciones_politica integer[];
    v_aliases text[];
    v_raices text[];
    v_raiz text;
    v_activo_ambito text;
    v_activo_huella text;
    v_huella_alta text;
    v_huella_contexto_recurso text;
    v_contexto_recurso bytea;
    v_ahora timestamptz(6);
    v_revision bigint;
    v_auditoria_ref text;
    v_evento_ref text;
    v_anterior_auditoria text;
    v_anterior_outbox text;
    v_secuencia_auditoria numeric(20, 0);
    v_secuencia_outbox numeric(20, 0);
    v_huella_auditoria text;
    v_huella_outbox text;
    v_payload_outbox bytea;
    v_huella_payload text;
    v_recibo_huella text;
    v_statement numeric;
    v_idle numeric;
BEGIN
    IF pg_catalog.current_setting('transaction_isolation') <> 'serializable'
       OR pg_catalog.current_setting('transaction_read_only') <> 'off'
       OR pg_catalog.current_setting('TimeZone') <> 'UTC'
       OR session_user = current_user
       OR NOT pg_catalog.pg_has_role(
           session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER'
       )
       OR pg_catalog.pg_has_role(
           session_user, 'vec_contratacion_temporal_migrador', 'MEMBER'
       )
       OR pg_catalog.pg_has_role(
           session_user, 'vec_contratacion_temporal_propietario', 'MEMBER'
       ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'confirmación de alta rechazada';
    END IF;
    SELECT setting::numeric INTO v_statement
      FROM pg_catalog.pg_settings
     WHERE name = 'statement_timeout' AND unit = 'ms';
    SELECT setting::numeric INTO v_idle
      FROM pg_catalog.pg_settings
     WHERE name = 'idle_in_transaction_session_timeout' AND unit = 'ms';
    IF v_statement IS NULL OR v_statement NOT BETWEEN 1 AND 15000
       OR v_idle IS NULL OR v_idle NOT BETWEEN 1 AND 20000
       OR pg_catalog.octet_length(p_alta_canonica) NOT BETWEEN 256 AND 32768
       OR pg_catalog.octet_length(p_sellos_hmac_canonicos)
          NOT BETWEEN 256 AND 8192 THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'entrada de alta inválida';
    END IF;
    BEGIN
        a := pg_catalog.convert_from(p_alta_canonica, 'UTF8')::jsonb;
        s := pg_catalog.convert_from(p_sellos_hmac_canonicos, 'UTF8')::jsonb;
        d := pg_catalog.convert_from(p_decision_canonica, 'UTF8')::jsonb;
    EXCEPTION
        WHEN data_exception OR invalid_text_representation
          OR character_not_in_repertoire OR untranslatable_character THEN
            RAISE EXCEPTION USING
                ERRCODE = '22023',
                MESSAGE = 'entrada de alta inválida';
    END;
    IF pg_catalog.jsonb_typeof(a) <> 'object'
       OR (SELECT pg_catalog.count(*)
             FROM pg_catalog.jsonb_object_keys(a)) <> 20
       OR NOT (a ?& ARRAY[
           'esquema', 'reserva_ref', 'expediente_ref', 'numero_visible',
           'recibo_ref', 'organizacion_ref', 'centro_ref',
           'categoria_ref', 'actor_ref', 'perfil_ref', 'version',
           'flujo_ref', 'flujo_version', 'flujo_huella_sha256',
           'fase_clave', 'estado', 'solicitud_huella_sha256',
           'accion_clave', 'unidad_ref', 'realizada_en'
       ])
       OR vec_contratacion_temporal.reconstruir_alta_v1(a)
          IS DISTINCT FROM p_alta_canonica
       OR a ->> 'esquema' <>
          'vec.contratacion-temporal.alta-persistencia.v1'
       OR a ->> 'version' <> '1'
       OR a ->> 'flujo_version' !~ '^[1-9][0-9]{0,15}$'
       OR (a ->> 'flujo_version')::numeric >
          9007199254740991::numeric
       OR a ->> 'estado' <> 'en_curso'
       OR a ->> 'numero_visible' !~
          '^[0-9]{4}/[A-Za-z0-9._-]{1,40}$'
       OR EXISTS (
           SELECT 1 FROM pg_catalog.unnest(ARRAY[
               a ->> 'reserva_ref', a ->> 'expediente_ref',
               a ->> 'recibo_ref', a ->> 'organizacion_ref',
               a ->> 'centro_ref', a ->> 'categoria_ref',
               a ->> 'actor_ref', a ->> 'perfil_ref',
               a ->> 'flujo_ref', a ->> 'fase_clave',
               a ->> 'accion_clave', a ->> 'unidad_ref'
           ]) AS r(valor)
           WHERE r.valor !~
             '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       )
       OR a ->> 'flujo_huella_sha256' !~ '^[0-9a-f]{64}$'
       OR a ->> 'solicitud_huella_sha256' !~ '^[0-9a-f]{64}$'
       OR a ->> 'realizada_en' !~
          '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]{1,6})?Z$'
       OR a ->> 'actor_ref' <> d ->> 'principal_id'
       OR a ->> 'perfil_ref' <> d ->> 'perfil_activo_ref' THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'proyección de alta inválida';
    END IF;

    IF pg_catalog.jsonb_typeof(s) <> 'object'
       OR (SELECT pg_catalog.count(*)
             FROM pg_catalog.jsonb_object_keys(s)) <> 3
       OR NOT (s ?& ARRAY['esquema', 'activo', 'retenidos'])
       OR s ->> 'esquema' <>
          'vec.contratacion-temporal.sellos-hmac.v1'
       OR pg_catalog.jsonb_typeof(s -> 'activo') <> 'object'
       OR pg_catalog.jsonb_typeof(s -> 'retenidos') <> 'array'
       OR pg_catalog.jsonb_array_length(s -> 'retenidos') > 3
       OR vec_contratacion_temporal.reconstruir_sellos_hmac_v1(s)
          IS DISTINCT FROM p_sellos_hmac_canonicos THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'sellos HMAC inválidos';
    END IF;
    v_pares := pg_catalog.jsonb_build_array(s -> 'activo') ||
               (s -> 'retenidos');
    IF EXISTS (
        SELECT 1
          FROM pg_catalog.jsonb_array_elements(v_pares) AS e(valor)
         WHERE pg_catalog.jsonb_typeof(e.valor) <> 'object'
            OR (SELECT pg_catalog.count(*)
                  FROM pg_catalog.jsonb_object_keys(e.valor)) <> 3
            OR NOT (e.valor ?& ARRAY[
                'generacion', 'ambito_hmac', 'huella_hmac'
            ])
            OR e.valor ->> 'generacion' !~ '^[1-9][0-9]{0,8}$'
            OR e.valor ->> 'ambito_hmac' !~
               (
                 '^hmac-sha256:vec[.]contratacion-temporal[.]'
                 || 'ambito-idempotencia/v[1-9][0-9]{0,8}:[a-f0-9]{64}$'
               )
            OR e.valor ->> 'huella_hmac' !~
               (
                 '^hmac-sha256:vec[.]contratacion-temporal[.]'
                 || 'huella-peticion/v[1-9][0-9]{0,8}:[a-f0-9]{64}$'
               )
            OR pg_catalog.right(e.valor ->> 'ambito_hmac', 64) =
               pg_catalog.repeat('0', 64)
            OR pg_catalog.right(e.valor ->> 'huella_hmac', 64) =
               pg_catalog.repeat('0', 64)
            OR substring(
                 e.valor ->> 'ambito_hmac'
                 FROM '/v([1-9][0-9]{0,8}):'
               )::integer <> (e.valor ->> 'generacion')::integer
            OR substring(
                 e.valor ->> 'huella_hmac'
                 FROM '/v([1-9][0-9]{0,8}):'
               )::integer <> (e.valor ->> 'generacion')::integer
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'sellos HMAC inválidos';
    END IF;
    SELECT pg_catalog.array_agg(
               (e.valor ->> 'generacion')::integer ORDER BY e.orden
           ),
           pg_catalog.array_agg(
               e.valor ->> 'ambito_hmac' ORDER BY e.orden
           )
      INTO v_generaciones, v_aliases
      FROM pg_catalog.jsonb_array_elements(v_pares)
           WITH ORDINALITY AS e(valor, orden);
    SELECT pg_catalog.array_agg(
               generacion ORDER BY posicion
           )
      INTO v_generaciones_politica
      FROM vec_contratacion_temporal.politica_generaciones_hmac_alta;
    IF v_generaciones IS DISTINCT FROM v_generaciones_politica
       OR pg_catalog.cardinality(v_generaciones) <>
          pg_catalog.cardinality(
              ARRAY(SELECT DISTINCT x FROM pg_catalog.unnest(
                  v_generaciones
              ) AS u(x))
          ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'política HMAC no satisfecha';
    END IF;
    v_activo_ambito := s #>> '{activo,ambito_hmac}';
    v_activo_huella := s #>> '{activo,huella_hmac}';

    -- Cierra la ligadura completa del recurso que originó la decisión V3.
    v_contexto_recurso := pg_catalog.convert_to(
        '{"ambitos":{"categoria_ref":' ||
          vec_contratacion_temporal.texto_json_go_v1(
              a ->> 'categoria_ref'
          ) ||
        ',"centro_ref":' ||
          vec_contratacion_temporal.texto_json_go_v1(a ->> 'centro_ref') ||
        ',"organizacion_ref":' ||
          vec_contratacion_temporal.texto_json_go_v1(
              a ->> 'organizacion_ref'
          ) ||
        '},"atributos":{"flujo_huella_sha256":' ||
          vec_contratacion_temporal.texto_json_go_v1(
              a ->> 'flujo_huella_sha256'
          ) ||
        ',"flujo_ref":' ||
          vec_contratacion_temporal.texto_json_go_v1(a ->> 'flujo_ref') ||
        ',"flujo_version":' ||
          vec_contratacion_temporal.texto_json_go_v1(
              a ->> 'flujo_version'
          ) ||
        ',"huella_peticion_hmac_activa":' ||
          vec_contratacion_temporal.texto_json_go_v1(v_activo_huella) ||
        '}}',
        'UTF8'
    );
    v_huella_contexto_recurso := pg_catalog.encode(
        pg_catalog.sha256(v_contexto_recurso), 'hex'
    );
    IF d ->> 'recurso_ref' <> v_activo_ambito
       OR d ->> 'modulo_id' <> 'contratacion_temporal'
       OR d ->> 'tipo_recurso' <> 'expediente_contratacion_temporal'
       OR d ->> 'accion' <> 'contratacion_temporal.solicitud.crear'
       OR d ->> 'finalidad' <> 'tramitar_necesidad_personal_temporal'
       OR d ->> 'contexto_recurso_huella_sha256' <>
          v_huella_contexto_recurso THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'efecto de alta no autorizado';
    END IF;

    SELECT * INTO STRICT v_consumo
      FROM vec_autorizacion_atestada_v3.
           registrar_y_consumir_decision_v3_atestada(
               p_capacidad_canonica, p_decision_canonica,
               p_motivo_canonico, p_contexto_actor_canonico,
               p_persona_version, p_perfil_version,
               p_payload_vec_ad_3, p_sobre_cose_sign1,
               p_evidencia_verificacion, p_raiz_publica_spki
           );
    IF v_consumo.efecto_ref <> v_activo_ambito
       OR v_consumo.huella_efecto_sha256 <>
          v_huella_contexto_recurso THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'consumo de alta incoherente';
    END IF;

    -- Orden total de locks de alias para que todas las sesiones converjan.
    PERFORM pg_catalog.pg_advisory_xact_lock(
        pg_catalog.hashtextextended('vec_ct:alias:' || alias, 0)
    )
      FROM pg_catalog.unnest(v_aliases) AS u(alias)
     ORDER BY alias COLLATE "C";
    SELECT pg_catalog.array_agg(
               DISTINCT ambito_raiz_hmac ORDER BY ambito_raiz_hmac
           )
      INTO v_raices
      FROM vec_contratacion_temporal.alias_ambito_alta
     WHERE alias_hmac = ANY (v_aliases);
    IF pg_catalog.cardinality(v_raices) > 1 THEN
        RAISE EXCEPTION USING
            ERRCODE = '23505',
            MESSAGE = 'alias HMAC divergentes';
    END IF;
    IF pg_catalog.cardinality(v_raices) = 1 THEN
        v_raiz := v_raices[1];
    ELSE
        v_raiz := v_activo_ambito;
        INSERT INTO vec_contratacion_temporal.identidad_reserva_alta (
            ambito_hmac, reserva_ref, expediente_ref, numero_visible,
            recibo_ref, huella_peticion_hmac, organizacion_ref,
            actor_ref, perfil_ref, creada_en
        ) VALUES (
            v_raiz, a ->> 'reserva_ref', a ->> 'expediente_ref',
            a ->> 'numero_visible', a ->> 'recibo_ref',
            v_activo_huella, a ->> 'organizacion_ref',
            a ->> 'actor_ref', a ->> 'perfil_ref',
            (a ->> 'realizada_en')::timestamptz
        );
        INSERT INTO vec_contratacion_temporal.reserva_alta_version (
            ambito_hmac, revision, estado, registrada_en
        ) VALUES (v_raiz, 1, 'reservada', clock_timestamp());
        INSERT INTO vec_contratacion_temporal.reserva_alta_actual
            VALUES (v_raiz, 1);
    END IF;

    SELECT * INTO STRICT v_identidad
      FROM vec_contratacion_temporal.identidad_reserva_alta
     WHERE ambito_hmac = v_raiz;
    IF v_identidad.reserva_ref <> a ->> 'reserva_ref'
       OR v_identidad.expediente_ref <> a ->> 'expediente_ref'
       OR v_identidad.numero_visible <> a ->> 'numero_visible'
       OR v_identidad.recibo_ref <> a ->> 'recibo_ref'
       OR v_identidad.organizacion_ref <> a ->> 'organizacion_ref'
       OR v_identidad.actor_ref <> a ->> 'actor_ref'
       OR v_identidad.perfil_ref <> a ->> 'perfil_ref' THEN
        RAISE EXCEPTION USING
            ERRCODE = '23505',
            MESSAGE = 'reserva de alta en conflicto';
    END IF;

    FOR v_par IN
        SELECT e.valor
          FROM pg_catalog.jsonb_array_elements(v_pares)
               WITH ORDINALITY AS e(valor, orden)
         ORDER BY e.orden
    LOOP
        IF EXISTS (
            SELECT 1
              FROM vec_contratacion_temporal.alias_ambito_alta x
             WHERE x.alias_hmac = v_par ->> 'ambito_hmac'
               AND x.ambito_raiz_hmac <> v_raiz
        ) OR EXISTS (
            SELECT 1
              FROM vec_contratacion_temporal.alias_huella_alta x
             WHERE x.ambito_raiz_hmac = v_raiz
               AND x.generacion =
                   (v_par ->> 'generacion')::integer
               AND x.alias_hmac <> v_par ->> 'huella_hmac'
        ) THEN
            RAISE EXCEPTION USING
                ERRCODE = '23505',
                MESSAGE = 'par HMAC en conflicto';
        END IF;
        INSERT INTO vec_contratacion_temporal.alias_ambito_alta (
            alias_hmac, ambito_raiz_hmac, generacion, registrada_en
        ) VALUES (
            v_par ->> 'ambito_hmac', v_raiz,
            (v_par ->> 'generacion')::integer, clock_timestamp()
        ) ON CONFLICT DO NOTHING;
        INSERT INTO vec_contratacion_temporal.alias_huella_alta (
            ambito_raiz_hmac, generacion, alias_hmac, registrada_en
        ) VALUES (
            v_raiz, (v_par ->> 'generacion')::integer,
            v_par ->> 'huella_hmac', clock_timestamp()
        ) ON CONFLICT DO NOTHING;
    END LOOP;

    v_huella_alta := pg_catalog.encode(
        pg_catalog.sha256(p_alta_canonica), 'hex'
    );
    SELECT e.expediente_ref, e.numero_visible, v.version,
           i.recibo_ref, au.auditoria_ref, o.evento_ref,
           rv.confirmada_en, v.huella_alta_sha256,
           e.decision_ref, e.efecto_ref, e.huella_efecto_sha256
      INTO v_existente
      FROM vec_contratacion_temporal.expediente_alta e
      JOIN vec_contratacion_temporal.expediente_alta_version v
        ON v.expediente_ref = e.expediente_ref AND v.version = 1
      JOIN vec_contratacion_temporal.identidad_reserva_alta i
        ON i.reserva_ref = e.reserva_ref
      JOIN vec_contratacion_temporal.reserva_alta_actual ra
        ON ra.ambito_hmac = i.ambito_hmac
      JOIN vec_contratacion_temporal.reserva_alta_version rv
        ON rv.ambito_hmac = ra.ambito_hmac AND rv.revision = ra.revision
      JOIN vec_contratacion_temporal.auditoria_alta au
        ON au.expediente_ref = e.expediente_ref
      JOIN vec_contratacion_temporal.outbox_alta o
        ON o.expediente_ref = e.expediente_ref
     WHERE e.expediente_ref = a ->> 'expediente_ref';
    IF FOUND THEN
        IF v_existente.huella_alta_sha256 <> v_huella_alta
           OR v_existente.decision_ref <> v_consumo.decision_ref
           OR v_existente.efecto_ref <> v_consumo.efecto_ref
           OR v_existente.huella_efecto_sha256 <>
              v_consumo.huella_efecto_sha256 THEN
            RAISE EXCEPTION USING
                ERRCODE = '23505',
                MESSAGE = 'expediente de alta en conflicto';
        END IF;
        v_recibo_huella := pg_catalog.encode(pg_catalog.sha256(
            pg_catalog.convert_to(
                v_existente.expediente_ref || ':' ||
                v_existente.numero_visible || ':' ||
                v_existente.version::text || ':' ||
                v_existente.recibo_ref || ':' ||
                v_existente.auditoria_ref || ':' ||
                v_existente.evento_ref || ':' ||
                v_existente.confirmada_en::text,
                'UTF8'
            )
        ), 'hex');
        RETURN QUERY SELECT
            v_existente.expediente_ref, v_existente.numero_visible,
            v_existente.version, v_existente.recibo_ref,
            v_existente.auditoria_ref, v_existente.evento_ref,
            v_existente.confirmada_en, v_recibo_huella;
        RETURN;
    END IF;

    v_ahora := pg_catalog.date_trunc(
        'microseconds', clock_timestamp()
    );
    IF v_ahora < (a ->> 'realizada_en')::timestamptz THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'instante de alta inválido';
    END IF;
    v_auditoria_ref := 'aud_ct_' || pg_catalog.substr(
        pg_catalog.encode(pg_catalog.sha256(
            pg_catalog.convert_to(
                v_consumo.decision_ref || ':' ||
                (a ->> 'expediente_ref'), 'UTF8'
            )
        ), 'hex'), 1, 32
    );
    v_evento_ref := 'evt_ct_' || pg_catalog.substr(
        pg_catalog.encode(pg_catalog.sha256(
            pg_catalog.convert_to(
                (a ->> 'expediente_ref') || ':' ||
                v_consumo.consumo_huella_sha256, 'UTF8'
            )
        ), 'hex'), 1, 32
    );

    INSERT INTO vec_contratacion_temporal.expediente_alta (
        expediente_ref, reserva_ref, numero_visible, organizacion_ref,
        actor_ref, perfil_ref, decision_ref, efecto_ref,
        huella_efecto_sha256, creada_en
    ) VALUES (
        a ->> 'expediente_ref', a ->> 'reserva_ref',
        a ->> 'numero_visible', a ->> 'organizacion_ref',
        a ->> 'actor_ref', a ->> 'perfil_ref',
        v_consumo.decision_ref, v_consumo.efecto_ref,
        v_consumo.huella_efecto_sha256,
        (a ->> 'realizada_en')::timestamptz
    );
    INSERT INTO vec_contratacion_temporal.expediente_alta_version (
        expediente_ref, version, alta_canonica, huella_alta_sha256,
        flujo_ref, flujo_version, flujo_huella_sha256, fase_clave,
        estado, solicitud_huella_sha256, registrada_en
    ) VALUES (
        a ->> 'expediente_ref', 1, p_alta_canonica, v_huella_alta,
        a ->> 'flujo_ref', (a ->> 'flujo_version')::numeric,
        a ->> 'flujo_huella_sha256', a ->> 'fase_clave',
        a ->> 'estado', a ->> 'solicitud_huella_sha256', v_ahora
    );
    INSERT INTO vec_contratacion_temporal.actuacion_alta (
        expediente_ref, secuencia, version_expediente, accion_clave,
        actor_ref, unidad_ref, recibo_ref, fase_destino,
        estado_destino, realizada_en, huella_sha256
    ) VALUES (
        a ->> 'expediente_ref', 1, 1, a ->> 'accion_clave',
        a ->> 'actor_ref', a ->> 'unidad_ref', a ->> 'recibo_ref',
        a ->> 'fase_clave', a ->> 'estado',
        (a ->> 'realizada_en')::timestamptz,
        pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
            (a ->> 'expediente_ref') || ':1:' ||
            (a ->> 'accion_clave') || ':' || (a ->> 'actor_ref') || ':' ||
            (a ->> 'unidad_ref') || ':' || (a ->> 'recibo_ref') || ':' ||
            (a ->> 'fase_clave') || ':' || (a ->> 'estado') || ':' ||
            (a ->> 'realizada_en'), 'UTF8'
        )), 'hex')
    );

    SELECT secuencia_auditoria, cabeza_auditoria_sha256,
           secuencia_outbox, cabeza_outbox_sha256
      INTO STRICT v_secuencia_auditoria, v_anterior_auditoria,
                  v_secuencia_outbox, v_anterior_outbox
      FROM vec_contratacion_temporal.control_cadenas_alta
     WHERE control_id
     FOR UPDATE;
    v_secuencia_auditoria := v_secuencia_auditoria + 1;
    v_secuencia_outbox := v_secuencia_outbox + 1;
    v_huella_auditoria := pg_catalog.encode(pg_catalog.sha256(
        pg_catalog.convert_to(
            v_secuencia_auditoria::text || ':' ||
            v_anterior_auditoria || ':' || v_auditoria_ref || ':' ||
            (a ->> 'expediente_ref') || ':' ||
            v_consumo.decision_ref || ':' ||
            v_consumo.consumo_huella_sha256 || ':' || v_huella_alta,
            'UTF8'
        )
    ), 'hex');
    INSERT INTO vec_contratacion_temporal.auditoria_alta (
        auditoria_ref, secuencia, expediente_ref, decision_ref,
        anterior_sha256, huella_sha256, registrada_en
    ) VALUES (
        v_auditoria_ref, v_secuencia_auditoria,
        a ->> 'expediente_ref', v_consumo.decision_ref,
        v_anterior_auditoria, v_huella_auditoria, v_ahora
    );

    v_payload_outbox := pg_catalog.convert_to(
        '{"esquema":"vec.contratacion-temporal.evento-expediente-registrado.v1"' ||
        ',"evento_ref":' ||
          vec_contratacion_temporal.texto_json_go_v1(v_evento_ref) ||
        ',"expediente_ref":' ||
          vec_contratacion_temporal.texto_json_go_v1(
              a ->> 'expediente_ref'
          ) ||
        ',"version":1,"ocurrido_en":' ||
          vec_contratacion_temporal.texto_json_go_v1(
              pg_catalog.to_char(
                  v_ahora AT TIME ZONE 'UTC',
                  'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
              )
          ) || '}',
        'UTF8'
    );
    v_huella_payload := pg_catalog.encode(
        pg_catalog.sha256(v_payload_outbox), 'hex'
    );
    v_huella_outbox := pg_catalog.encode(pg_catalog.sha256(
        pg_catalog.convert_to(
            v_secuencia_outbox::text || ':' || v_anterior_outbox || ':' ||
            v_evento_ref || ':' || v_huella_payload,
            'UTF8'
        )
    ), 'hex');
    INSERT INTO vec_contratacion_temporal.outbox_alta (
        evento_ref, secuencia, expediente_ref, tipo_evento,
        payload_canonico, payload_huella_sha256, anterior_sha256,
        huella_sha256, registrada_en
    ) VALUES (
        v_evento_ref, v_secuencia_outbox, a ->> 'expediente_ref',
        'contratacion_temporal.expediente.registrado.v1',
        v_payload_outbox, v_huella_payload, v_anterior_outbox,
        v_huella_outbox, v_ahora
    );
    UPDATE vec_contratacion_temporal.control_cadenas_alta
       SET secuencia_auditoria = v_secuencia_auditoria,
           cabeza_auditoria_sha256 = v_huella_auditoria,
           secuencia_outbox = v_secuencia_outbox,
           cabeza_outbox_sha256 = v_huella_outbox,
           actualizada_en = v_ahora
     WHERE control_id;

    SELECT revision INTO STRICT v_revision
      FROM vec_contratacion_temporal.reserva_alta_actual
     WHERE ambito_hmac = v_raiz
     FOR UPDATE;
    v_revision := v_revision + 1;
    INSERT INTO vec_contratacion_temporal.reserva_alta_version (
        ambito_hmac, revision, estado, version_expediente,
        auditoria_ref, evento_ref, confirmada_en, registrada_en
    ) VALUES (
        v_raiz, v_revision, 'confirmada', 1,
        v_auditoria_ref, v_evento_ref, v_ahora, v_ahora
    );
    UPDATE vec_contratacion_temporal.reserva_alta_actual
       SET revision = v_revision
     WHERE ambito_hmac = v_raiz;

    v_recibo_huella := pg_catalog.encode(pg_catalog.sha256(
        pg_catalog.convert_to(
            (a ->> 'expediente_ref') || ':' ||
            (a ->> 'numero_visible') || ':1:' ||
            (a ->> 'recibo_ref') || ':' || v_auditoria_ref || ':' ||
            v_evento_ref || ':' || v_ahora::text,
            'UTF8'
        )
    ), 'hex');
    RETURN QUERY SELECT
        a ->> 'expediente_ref', a ->> 'numero_visible', 1::numeric,
        a ->> 'recibo_ref', v_auditoria_ref, v_evento_ref,
        v_ahora, v_recibo_huella;
EXCEPTION
    WHEN invalid_text_representation OR datetime_field_overflow
      OR numeric_value_out_of_range THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'entrada de alta inválida';
END
$funcion$;

REVOKE EXECUTE ON FUNCTION
    vec_contratacion_temporal.preparar_alta_v2(jsonb)
    FROM vec_contratacion_temporal_ejecutor;
REVOKE ALL ON FUNCTION
    vec_contratacion_temporal.confirmar_alta_atestada_v1(
        bytea, bytea, bytea, bytea, numeric, numeric,
        bytea, bytea, bytea, bytea, bytea, bytea
    ) FROM PUBLIC, vec_contratacion_temporal_ejecutor;
GRANT EXECUTE ON FUNCTION
    vec_contratacion_temporal.confirmar_alta_atestada_v1(
        bytea, bytea, bytea, bytea, numeric, numeric,
        bytea, bytea, bytea, bytea, bytea, bytea
    ) TO vec_contratacion_temporal_ejecutor;

COMMIT;
