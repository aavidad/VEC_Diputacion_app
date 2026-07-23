BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:000004_integridad_agregado_alta', 0
    )
);

DO $prevalidacion$
BEGIN
    IF pg_catalog.to_regclass(
           'vec_contratacion_temporal.expediente_alta') IS NULL
       OR pg_catalog.to_regclass(
           'vec_contratacion_temporal.confirmacion_agregado_alta') IS NOT NULL
       OR EXISTS (
           SELECT 1 FROM vec_contratacion_temporal.expediente_alta
       ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'estado incompatible para integridad de alta';
    END IF;
END
$prevalidacion$;

ALTER TABLE vec_contratacion_temporal.reserva_alta_version
    ADD COLUMN confirmacion_ref text,
    ADD CONSTRAINT reserva_confirmacion_ref_formato CHECK (
        confirmacion_ref IS NULL
        OR confirmacion_ref ~ '^cnf_ct_[0-9a-f]{32}$'
    ),
    ADD CONSTRAINT reserva_confirmacion_ref_estado CHECK (
        (estado = 'reservada' AND confirmacion_ref IS NULL)
        OR (estado = 'confirmada' AND confirmacion_ref IS NOT NULL)
    ),
    ADD CONSTRAINT reserva_confirmacion_unica
        UNIQUE (ambito_hmac, revision, confirmacion_ref);

ALTER TABLE vec_contratacion_temporal.expediente_alta
    ADD COLUMN confirmacion_ref text NOT NULL,
    ADD CONSTRAINT expediente_confirmacion_ref_formato CHECK (
        confirmacion_ref ~ '^cnf_ct_[0-9a-f]{32}$'
    ),
    ADD CONSTRAINT expediente_confirmacion_unica
        UNIQUE (expediente_ref, confirmacion_ref);

ALTER TABLE vec_contratacion_temporal.expediente_alta_version
    ADD COLUMN confirmacion_ref text NOT NULL,
    ADD CONSTRAINT version_confirmacion_ref_formato CHECK (
        confirmacion_ref ~ '^cnf_ct_[0-9a-f]{32}$'
    ),
    ADD CONSTRAINT version_confirmacion_unica
        UNIQUE (expediente_ref, version, confirmacion_ref);

ALTER TABLE vec_contratacion_temporal.actuacion_alta
    ADD COLUMN confirmacion_ref text NOT NULL,
    ADD CONSTRAINT actuacion_confirmacion_ref_formato CHECK (
        confirmacion_ref ~ '^cnf_ct_[0-9a-f]{32}$'
    ),
    ADD CONSTRAINT actuacion_confirmacion_unica
        UNIQUE (expediente_ref, secuencia, confirmacion_ref);

ALTER TABLE vec_contratacion_temporal.auditoria_alta
    ADD COLUMN confirmacion_ref text NOT NULL,
    ADD COLUMN consumo_huella_sha256 text NOT NULL,
    ADD CONSTRAINT auditoria_confirmacion_ref_formato CHECK (
        confirmacion_ref ~ '^cnf_ct_[0-9a-f]{32}$'
    ),
    ADD CONSTRAINT auditoria_consumo_huella_formato CHECK (
        consumo_huella_sha256 ~ '^[0-9a-f]{64}$'
    ),
    ADD CONSTRAINT auditoria_confirmacion_unica
        UNIQUE (auditoria_ref, confirmacion_ref);

ALTER TABLE vec_contratacion_temporal.outbox_alta
    ADD COLUMN confirmacion_ref text NOT NULL,
    ADD CONSTRAINT outbox_confirmacion_ref_formato CHECK (
        confirmacion_ref ~ '^cnf_ct_[0-9a-f]{32}$'
    ),
    ADD CONSTRAINT outbox_confirmacion_unica
        UNIQUE (evento_ref, confirmacion_ref);

ALTER TABLE vec_contratacion_temporal.identidad_reserva_alta
    ADD CONSTRAINT identidad_agregado_unica UNIQUE (
        ambito_hmac, reserva_ref, expediente_ref, numero_visible, recibo_ref
    );

CREATE TABLE vec_contratacion_temporal.confirmacion_agregado_alta (
    confirmacion_ref text PRIMARY KEY,
    agregado_huella_sha256 text NOT NULL UNIQUE,
    ambito_hmac text NOT NULL,
    reserva_revision bigint NOT NULL,
    reserva_ref text NOT NULL UNIQUE,
    expediente_ref text NOT NULL UNIQUE,
    numero_visible text NOT NULL UNIQUE,
    recibo_ref text NOT NULL UNIQUE,
    decision_ref text NOT NULL UNIQUE,
    efecto_ref text NOT NULL UNIQUE,
    huella_efecto_sha256 text NOT NULL,
    consumo_huella_sha256 text NOT NULL UNIQUE,
    version_expediente numeric(20, 0) NOT NULL,
    huella_alta_sha256 text NOT NULL UNIQUE,
    actuacion_secuencia numeric(20, 0) NOT NULL,
    actuacion_huella_sha256 text NOT NULL UNIQUE,
    auditoria_ref text NOT NULL UNIQUE,
    auditoria_secuencia numeric(20, 0) NOT NULL UNIQUE,
    auditoria_anterior_sha256 text NOT NULL,
    auditoria_huella_sha256 text NOT NULL UNIQUE,
    evento_ref text NOT NULL UNIQUE,
    outbox_secuencia numeric(20, 0) NOT NULL UNIQUE,
    payload_huella_sha256 text NOT NULL UNIQUE,
    outbox_anterior_sha256 text NOT NULL,
    outbox_huella_sha256 text NOT NULL UNIQUE,
    confirmada_en timestamptz(6) NOT NULL,
    recibo_huella_sha256 text NOT NULL UNIQUE,
    creada_en timestamptz(6) NOT NULL,
    CHECK (confirmacion_ref ~ '^cnf_ct_[0-9a-f]{32}$'),
    CHECK (version_expediente = 1 AND actuacion_secuencia = 1),
    CHECK (reserva_revision > 1),
    CHECK (confirmada_en = creada_en),
    CHECK (confirmada_en = pg_catalog.date_trunc(
        'microseconds', confirmada_en
    )),
    CHECK (agregado_huella_sha256 ~ '^[0-9a-f]{64}$'),
    CHECK (huella_efecto_sha256 ~ '^[0-9a-f]{64}$'),
    CHECK (consumo_huella_sha256 ~ '^[0-9a-f]{64}$'),
    CHECK (huella_alta_sha256 ~ '^[0-9a-f]{64}$'),
    CHECK (actuacion_huella_sha256 ~ '^[0-9a-f]{64}$'),
    CHECK (auditoria_anterior_sha256 ~ '^[0-9a-f]{64}$'),
    CHECK (auditoria_huella_sha256 ~ '^[0-9a-f]{64}$'),
    CHECK (payload_huella_sha256 ~ '^[0-9a-f]{64}$'),
    CHECK (outbox_anterior_sha256 ~ '^[0-9a-f]{64}$'),
    CHECK (outbox_huella_sha256 ~ '^[0-9a-f]{64}$'),
    CHECK (recibo_huella_sha256 ~ '^[0-9a-f]{64}$'),
    FOREIGN KEY (
        ambito_hmac, reserva_ref, expediente_ref, numero_visible, recibo_ref
    ) REFERENCES vec_contratacion_temporal.identidad_reserva_alta (
        ambito_hmac, reserva_ref, expediente_ref, numero_visible, recibo_ref
    ) DEFERRABLE INITIALLY DEFERRED,
    FOREIGN KEY (ambito_hmac, reserva_revision, confirmacion_ref)
        REFERENCES vec_contratacion_temporal.reserva_alta_version (
            ambito_hmac, revision, confirmacion_ref
        ) DEFERRABLE INITIALLY DEFERRED,
    FOREIGN KEY (expediente_ref, confirmacion_ref)
        REFERENCES vec_contratacion_temporal.expediente_alta (
            expediente_ref, confirmacion_ref
        ) DEFERRABLE INITIALLY DEFERRED,
    FOREIGN KEY (expediente_ref, version_expediente, confirmacion_ref)
        REFERENCES vec_contratacion_temporal.expediente_alta_version (
            expediente_ref, version, confirmacion_ref
        ) DEFERRABLE INITIALLY DEFERRED,
    FOREIGN KEY (expediente_ref, actuacion_secuencia, confirmacion_ref)
        REFERENCES vec_contratacion_temporal.actuacion_alta (
            expediente_ref, secuencia, confirmacion_ref
        ) DEFERRABLE INITIALLY DEFERRED,
    FOREIGN KEY (auditoria_ref, confirmacion_ref)
        REFERENCES vec_contratacion_temporal.auditoria_alta (
            auditoria_ref, confirmacion_ref
        ) DEFERRABLE INITIALLY DEFERRED,
    FOREIGN KEY (evento_ref, confirmacion_ref)
        REFERENCES vec_contratacion_temporal.outbox_alta (
            evento_ref, confirmacion_ref
        ) DEFERRABLE INITIALLY DEFERRED
);

ALTER TABLE vec_contratacion_temporal.reserva_alta_version
    ADD CONSTRAINT reserva_marcador_fk FOREIGN KEY (confirmacion_ref)
        REFERENCES vec_contratacion_temporal.confirmacion_agregado_alta
        DEFERRABLE INITIALLY DEFERRED,
    ADD CONSTRAINT reserva_auditoria_marcador_fk
        FOREIGN KEY (auditoria_ref, confirmacion_ref)
        REFERENCES vec_contratacion_temporal.auditoria_alta (
            auditoria_ref, confirmacion_ref
        ) DEFERRABLE INITIALLY DEFERRED,
    ADD CONSTRAINT reserva_outbox_marcador_fk
        FOREIGN KEY (evento_ref, confirmacion_ref)
        REFERENCES vec_contratacion_temporal.outbox_alta (
            evento_ref, confirmacion_ref
        ) DEFERRABLE INITIALLY DEFERRED;

ALTER TABLE vec_contratacion_temporal.expediente_alta
    ADD CONSTRAINT expediente_marcador_fk FOREIGN KEY (confirmacion_ref)
        REFERENCES vec_contratacion_temporal.confirmacion_agregado_alta
        DEFERRABLE INITIALLY DEFERRED;
ALTER TABLE vec_contratacion_temporal.expediente_alta_version
    ADD CONSTRAINT version_marcador_fk FOREIGN KEY (confirmacion_ref)
        REFERENCES vec_contratacion_temporal.confirmacion_agregado_alta
        DEFERRABLE INITIALLY DEFERRED;
ALTER TABLE vec_contratacion_temporal.actuacion_alta
    ADD CONSTRAINT actuacion_marcador_fk FOREIGN KEY (confirmacion_ref)
        REFERENCES vec_contratacion_temporal.confirmacion_agregado_alta
        DEFERRABLE INITIALLY DEFERRED;
ALTER TABLE vec_contratacion_temporal.auditoria_alta
    ADD CONSTRAINT auditoria_marcador_fk FOREIGN KEY (confirmacion_ref)
        REFERENCES vec_contratacion_temporal.confirmacion_agregado_alta
        DEFERRABLE INITIALLY DEFERRED;
ALTER TABLE vec_contratacion_temporal.outbox_alta
    ADD CONSTRAINT outbox_marcador_fk FOREIGN KEY (confirmacion_ref)
        REFERENCES vec_contratacion_temporal.confirmacion_agregado_alta
        DEFERRABLE INITIALLY DEFERRED;

CREATE TRIGGER confirmacion_agregado_inmutable
BEFORE UPDATE OR DELETE
ON vec_contratacion_temporal.confirmacion_agregado_alta
FOR EACH ROW
EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();
CREATE TRIGGER outbox_alta_inmutable
BEFORE UPDATE OR DELETE ON vec_contratacion_temporal.outbox_alta
FOR EACH ROW
EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();

CREATE FUNCTION vec_contratacion_temporal.huella_prueba_agregado_alta_v1(
    VARIADIC valores text[]
)
RETURNS text
LANGUAGE plpgsql
IMMUTABLE
STRICT
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_preimagen bytea;
BEGIN
    IF pg_catalog.cardinality(valores) <> 27
       OR pg_catalog.array_position(valores, NULL) IS NOT NULL THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'prueba de agregado inválida';
    END IF;
    SELECT pg_catalog.string_agg(
               vec_contratacion_temporal.encuadrar_texto_v1(valor),
               ''::bytea ORDER BY orden
           )
      INTO STRICT v_preimagen
      FROM pg_catalog.unnest(valores) WITH ORDINALITY AS v(valor, orden);
    RETURN pg_catalog.encode(pg_catalog.sha256(v_preimagen), 'hex');
END
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.reconciliar_agregado_alta_v1(
    p_alta_canonica bytea,
    p_ambito_hmac text,
    p_huella_peticion_hmac text,
    p_decision_ref text,
    p_efecto_ref text,
    p_huella_efecto_sha256 text,
    p_consumo_huella_sha256 text
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
STABLE
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
DECLARE
    a jsonb;
    i record;
    ra record;
    rv record;
    e record;
    ev record;
    ac record;
    au record;
    o record;
    m record;
    c record;
    v_raiz text;
    v_huella_alias text;
    v_confirmacion_ref text;
    v_auditoria_ref text;
    v_evento_ref text;
    v_huella_alta text;
    v_huella_solicitud text;
    v_huella_actuacion text;
    v_huella_auditoria text;
    v_payload bytea;
    v_huella_payload text;
    v_huella_outbox text;
    v_huella_recibo text;
    v_huella_agregado text;
    v_incompleto boolean := false;
BEGIN
    BEGIN
        a := pg_catalog.convert_from(p_alta_canonica, 'UTF8')::jsonb;
    EXCEPTION
        WHEN data_exception OR invalid_text_representation
          OR character_not_in_repertoire OR untranslatable_character THEN
            RAISE EXCEPTION USING
                ERRCODE = '22023',
                MESSAGE = 'reconciliación de alta inválida';
    END;
    SELECT aa.ambito_raiz_hmac, ah.alias_hmac
      INTO v_raiz, v_huella_alias
      FROM vec_contratacion_temporal.alias_ambito_alta aa
      JOIN vec_contratacion_temporal.alias_huella_alta ah
        ON ah.ambito_raiz_hmac = aa.ambito_raiz_hmac
       AND ah.generacion = aa.generacion
     WHERE aa.alias_hmac = p_ambito_hmac;
    IF NOT FOUND
       OR v_huella_alias IS DISTINCT FROM p_huella_peticion_hmac THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'integridad del agregado de alta no acreditada';
    END IF;
    v_huella_alta := pg_catalog.encode(
        pg_catalog.sha256(p_alta_canonica), 'hex'
    );
    v_huella_solicitud := pg_catalog.encode(pg_catalog.sha256(
        pg_catalog.convert_to(
            vec_contratacion_temporal.reconstruir_solicitud_efecto_v2(
                a -> 'solicitud'
            ), 'UTF8'
        )
    ), 'hex');
    v_huella_actuacion := pg_catalog.encode(pg_catalog.sha256(
        vec_contratacion_temporal.encuadrar_texto_v1(
            a ->> 'expediente_ref'
        ) ||
        vec_contratacion_temporal.encuadrar_texto_v1(
            (a #> '{actuacion}')::text
        )
    ), 'hex');
    v_confirmacion_ref := 'cnf_ct_' || pg_catalog.substr(
        pg_catalog.encode(pg_catalog.sha256(
            vec_contratacion_temporal.encuadrar_texto_v1(p_decision_ref) ||
            vec_contratacion_temporal.encuadrar_texto_v1(
                a ->> 'expediente_ref'
            ) ||
            vec_contratacion_temporal.encuadrar_texto_v1(
                p_consumo_huella_sha256
            )
        ), 'hex'), 1, 32
    );
    v_auditoria_ref := 'aud_ct_' || pg_catalog.substr(
        pg_catalog.encode(pg_catalog.sha256(
            vec_contratacion_temporal.encuadrar_texto_v1(p_decision_ref) ||
            vec_contratacion_temporal.encuadrar_texto_v1(
                a ->> 'expediente_ref'
            )
        ), 'hex'), 1, 32
    );
    v_evento_ref := 'evt_ct_' || pg_catalog.substr(
        pg_catalog.encode(pg_catalog.sha256(
            vec_contratacion_temporal.encuadrar_texto_v1(
                a ->> 'expediente_ref'
            ) ||
            vec_contratacion_temporal.encuadrar_texto_v1(
                p_consumo_huella_sha256
            )
        ), 'hex'), 1, 32
    );
    SELECT * INTO i
      FROM vec_contratacion_temporal.identidad_reserva_alta ii
     WHERE ii.ambito_hmac = v_raiz;
    IF NOT FOUND THEN v_incompleto := true; END IF;
    SELECT * INTO ra
      FROM vec_contratacion_temporal.reserva_alta_actual raa
     WHERE raa.ambito_hmac = v_raiz;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'integridad del agregado de alta no acreditada';
    END IF;
    SELECT * INTO rv
      FROM vec_contratacion_temporal.reserva_alta_version rav
     WHERE rav.ambito_hmac = v_raiz
       AND rav.revision = ra.revision;
    IF NOT FOUND THEN v_incompleto := true; END IF;
    SELECT * INTO e
      FROM vec_contratacion_temporal.expediente_alta ea
     WHERE ea.expediente_ref = a ->> 'expediente_ref';
    IF NOT FOUND THEN v_incompleto := true; END IF;
    SELECT * INTO ev
      FROM vec_contratacion_temporal.expediente_alta_version eav
     WHERE eav.expediente_ref = a ->> 'expediente_ref'
       AND eav.version = 1;
    IF NOT FOUND THEN v_incompleto := true; END IF;
    SELECT * INTO ac
      FROM vec_contratacion_temporal.actuacion_alta aa
     WHERE aa.expediente_ref = a ->> 'expediente_ref'
       AND aa.secuencia = 1;
    IF NOT FOUND THEN v_incompleto := true; END IF;
    SELECT * INTO au
      FROM vec_contratacion_temporal.auditoria_alta aua
     WHERE aua.expediente_ref = a ->> 'expediente_ref';
    IF NOT FOUND THEN v_incompleto := true; END IF;
    SELECT * INTO o
      FROM vec_contratacion_temporal.outbox_alta oa
     WHERE oa.expediente_ref = a ->> 'expediente_ref';
    IF NOT FOUND THEN v_incompleto := true; END IF;
    SELECT * INTO m
      FROM vec_contratacion_temporal.confirmacion_agregado_alta caa
     WHERE caa.confirmacion_ref = v_confirmacion_ref;
    IF NOT FOUND THEN v_incompleto := true; END IF;
    SELECT * INTO c
      FROM vec_contratacion_temporal.control_cadenas_alta cca
     WHERE cca.control_id;
    IF NOT FOUND THEN v_incompleto := true; END IF;
    IF v_incompleto THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'integridad del agregado de alta no acreditada';
    END IF;
    v_huella_auditoria := pg_catalog.encode(pg_catalog.sha256(
        vec_contratacion_temporal.encuadrar_texto_v1(au.secuencia::text) ||
        vec_contratacion_temporal.encuadrar_texto_v1(au.anterior_sha256) ||
        vec_contratacion_temporal.encuadrar_texto_v1(v_auditoria_ref) ||
        vec_contratacion_temporal.encuadrar_texto_v1(
            a ->> 'expediente_ref'
        ) ||
        vec_contratacion_temporal.encuadrar_texto_v1(p_decision_ref) ||
        vec_contratacion_temporal.encuadrar_texto_v1(
            p_consumo_huella_sha256
        ) ||
        vec_contratacion_temporal.encuadrar_texto_v1(v_huella_alta)
    ), 'hex');
    v_payload := pg_catalog.convert_to(
        '{"esquema":"vec.contratacion-temporal.evento-expediente-registrado.v1"' ||
        ',"evento_ref":' ||
          vec_contratacion_temporal.texto_json_go_v1(v_evento_ref) ||
        ',"expediente_ref":' ||
          vec_contratacion_temporal.texto_json_go_v1(
              a ->> 'expediente_ref'
          ) ||
        ',"version":1,"ocurrido_en":' ||
          vec_contratacion_temporal.texto_json_go_v1(
              vec_contratacion_temporal.instante_utc_v1(m.confirmada_en)
          ) || '}',
        'UTF8'
    );
    v_huella_payload := pg_catalog.encode(
        pg_catalog.sha256(v_payload), 'hex'
    );
    v_huella_outbox := pg_catalog.encode(pg_catalog.sha256(
        vec_contratacion_temporal.encuadrar_texto_v1(o.secuencia::text) ||
        vec_contratacion_temporal.encuadrar_texto_v1(o.anterior_sha256) ||
        vec_contratacion_temporal.encuadrar_texto_v1(v_evento_ref) ||
        vec_contratacion_temporal.encuadrar_texto_v1(v_huella_payload)
    ), 'hex');
    v_huella_recibo := pg_catalog.encode(pg_catalog.sha256(
        vec_contratacion_temporal.encuadrar_texto_v1(
            a ->> 'expediente_ref'
        ) ||
        vec_contratacion_temporal.encuadrar_texto_v1(
            a ->> 'numero_visible'
        ) ||
        vec_contratacion_temporal.encuadrar_texto_v1('1') ||
        vec_contratacion_temporal.encuadrar_texto_v1(a ->> 'recibo_ref') ||
        vec_contratacion_temporal.encuadrar_texto_v1(v_auditoria_ref) ||
        vec_contratacion_temporal.encuadrar_texto_v1(v_evento_ref) ||
        vec_contratacion_temporal.encuadrar_texto_v1(
            vec_contratacion_temporal.instante_utc_v1(m.confirmada_en)
        )
    ), 'hex');
    v_huella_agregado :=
      vec_contratacion_temporal.huella_prueba_agregado_alta_v1(
        VARIADIC ARRAY[
          'vec.contratacion-temporal.confirmacion-agregado-alta.v1',
          v_confirmacion_ref, v_raiz, rv.revision::text,
          a ->> 'reserva_ref', a ->> 'expediente_ref',
          a ->> 'numero_visible', a ->> 'recibo_ref',
          p_decision_ref, p_efecto_ref, p_huella_efecto_sha256,
          p_consumo_huella_sha256, '1', v_huella_alta, '1',
          v_huella_actuacion, v_auditoria_ref, au.secuencia::text,
          au.anterior_sha256, v_huella_auditoria, v_evento_ref,
          o.secuencia::text, v_huella_payload, o.anterior_sha256,
          v_huella_outbox,
          vec_contratacion_temporal.instante_utc_v1(m.confirmada_en),
          v_huella_recibo
        ]
      );
    IF i.reserva_ref IS DISTINCT FROM a ->> 'reserva_ref'
       OR i.expediente_ref IS DISTINCT FROM a ->> 'expediente_ref'
       OR i.numero_visible IS DISTINCT FROM a ->> 'numero_visible'
       OR i.recibo_ref IS DISTINCT FROM a ->> 'recibo_ref'
       OR i.organizacion_ref IS DISTINCT FROM a ->> 'organizacion_ref'
       OR i.actor_ref IS DISTINCT FROM a ->> 'actor_ref'
       OR i.perfil_ref IS DISTINCT FROM a ->> 'perfil_ref'
       OR i.creada_en IS DISTINCT FROM
          (a #>> '{actuacion,realizada_en}')::timestamptz
       OR NOT EXISTS (
           SELECT 1
             FROM vec_contratacion_temporal.alias_ambito_alta ia
             JOIN vec_contratacion_temporal.alias_huella_alta ih
               ON ih.ambito_raiz_hmac = ia.ambito_raiz_hmac
              AND ih.generacion = ia.generacion
            WHERE ia.alias_hmac = v_raiz
              AND ih.alias_hmac = i.huella_peticion_hmac
       )
       OR ra.revision IS DISTINCT FROM rv.revision
       OR rv.estado IS DISTINCT FROM 'confirmada'
       OR rv.version_expediente IS DISTINCT FROM 1
       OR rv.auditoria_ref IS DISTINCT FROM v_auditoria_ref
       OR rv.evento_ref IS DISTINCT FROM v_evento_ref
       OR rv.confirmacion_ref IS DISTINCT FROM v_confirmacion_ref
       OR rv.confirmada_en IS DISTINCT FROM m.confirmada_en
       OR rv.registrada_en IS DISTINCT FROM m.confirmada_en
       OR e.reserva_ref IS DISTINCT FROM a ->> 'reserva_ref'
       OR e.numero_visible IS DISTINCT FROM a ->> 'numero_visible'
       OR e.organizacion_ref IS DISTINCT FROM a ->> 'organizacion_ref'
       OR e.actor_ref IS DISTINCT FROM a ->> 'actor_ref'
       OR e.perfil_ref IS DISTINCT FROM a ->> 'perfil_ref'
       OR e.decision_ref IS DISTINCT FROM p_decision_ref
       OR e.efecto_ref IS DISTINCT FROM p_efecto_ref
       OR e.huella_efecto_sha256 IS DISTINCT FROM p_huella_efecto_sha256
       OR e.creada_en IS DISTINCT FROM (a ->> 'creado_en')::timestamptz
       OR e.confirmacion_ref IS DISTINCT FROM v_confirmacion_ref
       OR ev.alta_canonica IS DISTINCT FROM p_alta_canonica
       OR ev.huella_alta_sha256 IS DISTINCT FROM v_huella_alta
       OR ev.flujo_ref IS DISTINCT FROM a #>> '{flujo,definicion_ref}'
       OR ev.flujo_version IS DISTINCT FROM
          (a #>> '{flujo,version}')::numeric
       OR ev.flujo_huella_sha256 IS DISTINCT FROM
          a #>> '{flujo,huella_sha256}'
       OR ev.fase_clave IS DISTINCT FROM a ->> 'fase_actual'
       OR ev.estado IS DISTINCT FROM a ->> 'estado_actual'
       OR ev.solicitud_huella_sha256 IS DISTINCT FROM v_huella_solicitud
       OR ev.registrada_en IS DISTINCT FROM m.confirmada_en
       OR ev.confirmacion_ref IS DISTINCT FROM v_confirmacion_ref
       OR ac.version_expediente IS DISTINCT FROM 1
       OR ac.accion_clave IS DISTINCT FROM a #>> '{actuacion,accion_clave}'
       OR ac.actor_ref IS DISTINCT FROM a #>> '{actuacion,actor_ref}'
       OR ac.unidad_ref IS DISTINCT FROM a #>> '{actuacion,unidad_ref}'
       OR ac.recibo_ref IS DISTINCT FROM a #>> '{actuacion,recibo_ref}'
       OR ac.fase_destino IS DISTINCT FROM a #>> '{actuacion,fase_destino}'
       OR ac.estado_destino IS DISTINCT FROM
          a #>> '{actuacion,estado_destino}'
       OR ac.realizada_en IS DISTINCT FROM
          (a #>> '{actuacion,realizada_en}')::timestamptz
       OR ac.huella_sha256 IS DISTINCT FROM v_huella_actuacion
       OR ac.confirmacion_ref IS DISTINCT FROM v_confirmacion_ref
       OR au.auditoria_ref IS DISTINCT FROM v_auditoria_ref
       OR au.decision_ref IS DISTINCT FROM p_decision_ref
       OR au.consumo_huella_sha256 IS DISTINCT FROM
          p_consumo_huella_sha256
       OR au.anterior_sha256 IS NULL
       OR au.huella_sha256 IS DISTINCT FROM v_huella_auditoria
       OR au.registrada_en IS DISTINCT FROM m.confirmada_en
       OR au.confirmacion_ref IS DISTINCT FROM v_confirmacion_ref
       OR o.evento_ref IS DISTINCT FROM v_evento_ref
       OR o.tipo_evento IS DISTINCT FROM
          'contratacion_temporal.expediente.registrado.v1'
       OR o.payload_canonico IS DISTINCT FROM v_payload
       OR o.payload_huella_sha256 IS DISTINCT FROM v_huella_payload
       OR o.anterior_sha256 IS NULL
       OR o.huella_sha256 IS DISTINCT FROM v_huella_outbox
       OR o.registrada_en IS DISTINCT FROM m.confirmada_en
       OR o.confirmacion_ref IS DISTINCT FROM v_confirmacion_ref
       OR m.confirmacion_ref IS DISTINCT FROM v_confirmacion_ref
       OR m.agregado_huella_sha256 IS DISTINCT FROM v_huella_agregado
       OR m.ambito_hmac IS DISTINCT FROM v_raiz
       OR m.reserva_revision IS DISTINCT FROM rv.revision
       OR m.reserva_ref IS DISTINCT FROM a ->> 'reserva_ref'
       OR m.expediente_ref IS DISTINCT FROM a ->> 'expediente_ref'
       OR m.numero_visible IS DISTINCT FROM a ->> 'numero_visible'
       OR m.recibo_ref IS DISTINCT FROM a ->> 'recibo_ref'
       OR m.decision_ref IS DISTINCT FROM p_decision_ref
       OR m.efecto_ref IS DISTINCT FROM p_efecto_ref
       OR m.huella_efecto_sha256 IS DISTINCT FROM p_huella_efecto_sha256
       OR m.consumo_huella_sha256 IS DISTINCT FROM
          p_consumo_huella_sha256
       OR m.version_expediente IS DISTINCT FROM 1
       OR m.huella_alta_sha256 IS DISTINCT FROM v_huella_alta
       OR m.actuacion_secuencia IS DISTINCT FROM 1
       OR m.actuacion_huella_sha256 IS DISTINCT FROM v_huella_actuacion
       OR m.auditoria_ref IS DISTINCT FROM v_auditoria_ref
       OR m.auditoria_secuencia IS DISTINCT FROM au.secuencia
       OR m.auditoria_anterior_sha256 IS DISTINCT FROM au.anterior_sha256
       OR m.auditoria_huella_sha256 IS DISTINCT FROM v_huella_auditoria
       OR m.evento_ref IS DISTINCT FROM v_evento_ref
       OR m.outbox_secuencia IS DISTINCT FROM o.secuencia
       OR m.payload_huella_sha256 IS DISTINCT FROM v_huella_payload
       OR m.outbox_anterior_sha256 IS DISTINCT FROM o.anterior_sha256
       OR m.outbox_huella_sha256 IS DISTINCT FROM v_huella_outbox
       OR m.recibo_huella_sha256 IS DISTINCT FROM v_huella_recibo
       OR m.creada_en IS DISTINCT FROM m.confirmada_en THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'integridad del agregado de alta no acreditada';
    END IF;
    IF au.secuencia > c.secuencia_auditoria
       OR o.secuencia > c.secuencia_outbox
       OR (
           au.secuencia = 1
           AND au.anterior_sha256 <> pg_catalog.repeat('0', 64)
       )
       OR (
           au.secuencia > 1
           AND NOT EXISTS (
               SELECT 1 FROM vec_contratacion_temporal.auditoria_alta p
                WHERE p.secuencia = au.secuencia - 1
                  AND p.huella_sha256 = au.anterior_sha256
           )
       )
       OR (
           au.secuencia = c.secuencia_auditoria
           AND au.huella_sha256 <> c.cabeza_auditoria_sha256
       )
       OR (
           au.secuencia < c.secuencia_auditoria
           AND NOT EXISTS (
               SELECT 1 FROM vec_contratacion_temporal.auditoria_alta n
                WHERE n.secuencia = au.secuencia + 1
                  AND n.anterior_sha256 = au.huella_sha256
           )
       )
       OR NOT EXISTS (
           SELECT 1
             FROM vec_contratacion_temporal.auditoria_alta h
             JOIN vec_contratacion_temporal.expediente_alta he
               USING (expediente_ref)
             JOIN vec_contratacion_temporal.expediente_alta_version hv
               ON hv.expediente_ref = he.expediente_ref
              AND hv.version = 1
            WHERE h.secuencia = c.secuencia_auditoria
              AND h.huella_sha256 = c.cabeza_auditoria_sha256
              AND h.huella_sha256 = pg_catalog.encode(pg_catalog.sha256(
                  vec_contratacion_temporal.encuadrar_texto_v1(
                      h.secuencia::text
                  ) ||
                  vec_contratacion_temporal.encuadrar_texto_v1(
                      h.anterior_sha256
                  ) ||
                  vec_contratacion_temporal.encuadrar_texto_v1(
                      h.auditoria_ref
                  ) ||
                  vec_contratacion_temporal.encuadrar_texto_v1(
                      h.expediente_ref
                  ) ||
                  vec_contratacion_temporal.encuadrar_texto_v1(
                      h.decision_ref
                  ) ||
                  vec_contratacion_temporal.encuadrar_texto_v1(
                      h.consumo_huella_sha256
                  ) ||
                  vec_contratacion_temporal.encuadrar_texto_v1(
                      hv.huella_alta_sha256
                  )
              ), 'hex')
       )
       OR (
           o.secuencia = 1
           AND o.anterior_sha256 <> pg_catalog.repeat('0', 64)
       )
       OR (
           o.secuencia > 1
           AND NOT EXISTS (
               SELECT 1 FROM vec_contratacion_temporal.outbox_alta p
                WHERE p.secuencia = o.secuencia - 1
                  AND p.huella_sha256 = o.anterior_sha256
           )
       )
       OR (
           o.secuencia = c.secuencia_outbox
           AND o.huella_sha256 <> c.cabeza_outbox_sha256
       )
       OR (
           o.secuencia < c.secuencia_outbox
           AND NOT EXISTS (
               SELECT 1 FROM vec_contratacion_temporal.outbox_alta n
                WHERE n.secuencia = o.secuencia + 1
                  AND n.anterior_sha256 = o.huella_sha256
           )
       )
       OR NOT EXISTS (
           SELECT 1 FROM vec_contratacion_temporal.outbox_alta h
            WHERE h.secuencia = c.secuencia_outbox
              AND h.huella_sha256 = c.cabeza_outbox_sha256
              AND h.payload_huella_sha256 = pg_catalog.encode(
                  pg_catalog.sha256(h.payload_canonico), 'hex'
              )
              AND h.huella_sha256 = pg_catalog.encode(pg_catalog.sha256(
                  vec_contratacion_temporal.encuadrar_texto_v1(
                      h.secuencia::text
                  ) ||
                  vec_contratacion_temporal.encuadrar_texto_v1(
                      h.anterior_sha256
                  ) ||
                  vec_contratacion_temporal.encuadrar_texto_v1(
                      h.evento_ref
                  ) ||
                  vec_contratacion_temporal.encuadrar_texto_v1(
                      h.payload_huella_sha256
                  )
              ), 'hex')
       ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'integridad del agregado de alta no acreditada';
    END IF;
    RETURN QUERY SELECT
        m.expediente_ref, m.numero_visible, m.version_expediente,
        m.recibo_ref, m.auditoria_ref, m.evento_ref,
        m.confirmada_en, m.recibo_huella_sha256;
END
$funcion$;

REVOKE ALL ON TABLE vec_contratacion_temporal.confirmacion_agregado_alta
    FROM PUBLIC, vec_contratacion_temporal_ejecutor;
REVOKE ALL ON FUNCTION
    vec_contratacion_temporal.huella_prueba_agregado_alta_v1(text[]),
    vec_contratacion_temporal.reconciliar_agregado_alta_v1(
        bytea, text, text, text, text, text, text
    ) FROM PUBLIC, vec_contratacion_temporal_ejecutor;

COMMIT;
