-- O4-04E/4: prueba durable estructural. No publica todavía la función exterior.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:o4_04:migraciones', 0
    )
);
SELECT control
  FROM vec_contratacion_temporal.control_migracion_cobertura_o4
 WHERE control AND version_esquema = 6
 FOR UPDATE;

DO $prevalidacion$
BEGIN
    IF NOT EXISTS (
           SELECT 1
             FROM vec_contratacion_temporal.control_migracion_cobertura_o4
            WHERE control AND version_esquema = 6
       )
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.o404e_decision_cobertura_exacta_v1(jsonb)'
       ) IS NULL
       OR pg_catalog.to_regclass(
           'vec_autorizacion.enlace_decision_cobertura_ct_o404e'
       ) IS NULL
       OR pg_catalog.to_regclass(
           'vec_contratacion_temporal.acreditacion_gobierno_decision_cobertura'
       ) IS NOT NULL
       OR EXISTS (
           SELECT 1
             FROM vec_contratacion_temporal
                  .confirmacion_operacion_decision_cobertura
       )
       OR EXISTS (
           SELECT 1
             FROM vec_contratacion_temporal
                  .terminal_operacion_decision_cobertura
       ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'estado incompatible para prueba durable O4-04E';
    END IF;
END
$prevalidacion$;
ALTER TABLE vec_contratacion_temporal.gobi_o404b_evento
    ADD CONSTRAINT gobi_o404b_evento_identidad_compuesta_o404e
    UNIQUE (secuencia, evento_ref, huella_evento_sha256);
ALTER TABLE vec_contratacion_temporal.consumo_cobertura_lote
    ADD CONSTRAINT consumo_cobertura_lote_ref_huella_o404e
    UNIQUE (lote_ref, lote_huella_sha256),
    ADD CONSTRAINT consumo_cobertura_lote_terminal_o404e
    UNIQUE (
        lote_ref, lote_huella_sha256, huella_orden_sha256,
        decision_vec_ref
    ),
    ADD CONSTRAINT consumo_cobertura_lote_identidad_compuesta_o404e
    UNIQUE (
        lote_ref, lote_huella_sha256, organizacion_ref, expediente_ref,
        version_expediente, reserva_ref, decision_vec_ref,
        decision_vec_huella_sha256, codigo_probatorio_vec
    );
ALTER TABLE vec_contratacion_temporal.outbox_expediente_integral
    ADD COLUMN o404e_recibo_ref text,
    ADD COLUMN o404e_auditoria_ref text,
    ADD COLUMN o404e_decision_vec_ref text,
    ADD COLUMN o404e_rama text,
    ADD CONSTRAINT outbox_expediente_terminal_o404e
    UNIQUE (
        evento_ref, o404e_recibo_ref, o404e_auditoria_ref,
        o404e_decision_vec_ref, o404e_rama
    );
CREATE TRIGGER ligar_outbox_terminal_o404e
BEFORE INSERT ON vec_contratacion_temporal.outbox_expediente_integral
FOR EACH ROW EXECUTE FUNCTION
    vec_contratacion_temporal.o404e_ligar_outbox_terminal_v1();
ALTER TABLE
vec_contratacion_temporal.reserva_operacion_decision_cobertura
    ADD CONSTRAINT reserva_cobertura_identidad_compuesta_o404e
    UNIQUE (
        reserva_ref, organizacion_ref, expediente_ref, version_expediente,
        recibo_ref, actuacion_ref, auditoria_ref, decision_vec_ref
    );
ALTER TABLE vec_contratacion_temporal.actuacion_expediente_integral
    ADD CONSTRAINT actuacion_integral_identidad_compuesta_o404e
    UNIQUE (operacion_ref, expediente_ref, version_expediente);
ALTER TABLE
vec_contratacion_temporal.confirmacion_operacion_decision_cobertura
    ADD CONSTRAINT confirmacion_cobertura_terminal_base_o404e
    UNIQUE (
        ambito_raiz_hmac, recibo_ref, huella_orden_sha256, rama,
        decision_vec_ref, auditoria_ref, codigo_probatorio_vec
    ),
    ADD CONSTRAINT confirmacion_cobertura_terminal_compuesta_o404e
    UNIQUE (
        ambito_raiz_hmac, recibo_ref, huella_orden_sha256, rama,
        decision_vec_ref, auditoria_ref, codigo_probatorio_vec,
        decision_cobertura_ref, decision_cobertura_huella_sha256,
        version_resultante, actuacion_ref
    );
CREATE TABLE
vec_contratacion_temporal.acreditacion_gobierno_decision_cobertura (
    acreditacion_ref text PRIMARY KEY,
    gobierno_huella_sha256 text NOT NULL,
    prueba_canonica bytea NOT NULL,
    reserva_ref text NOT NULL UNIQUE,
    recibo_ref text NOT NULL,
    actuacion_ref text NOT NULL,
    auditoria_ref text NOT NULL,
    organizacion_ref text NOT NULL,
    expediente_ref text NOT NULL,
    version_expediente numeric(20, 0) NOT NULL,
    accion text NOT NULL,
    codigo_probatorio_vec text NOT NULL DEFAULT 'concedida',
    catalogo_ref text NOT NULL,
    catalogo_version numeric(20, 0) NOT NULL,
    catalogo_huella_sha256 text NOT NULL,
    politica_ref text NOT NULL,
    politica_version numeric(20, 0) NOT NULL,
    politica_huella_sha256 text NOT NULL,
    actuacion_gobierno_ref text NOT NULL,
    actuacion_gobierno_version numeric(20, 0) NOT NULL,
    actuacion_gobierno_huella_sha256 text NOT NULL,
    checkpoint_secuencia bigint NOT NULL,
    checkpoint_evento_ref text NOT NULL,
    checkpoint_evento_huella_sha256 text NOT NULL,
    decision_vec_ref text NOT NULL UNIQUE,
    rama text NOT NULL DEFAULT 'concedida',
    decision_vec_huella_sha256 text NOT NULL,
    evaluada_en timestamptz(6) NOT NULL,
    acreditada_en timestamptz(6) NOT NULL,
    UNIQUE (acreditacion_ref, gobierno_huella_sha256),
    UNIQUE (
        acreditacion_ref, gobierno_huella_sha256, recibo_ref,
        auditoria_ref, decision_vec_ref
    ),
    UNIQUE (
        acreditacion_ref, gobierno_huella_sha256, reserva_ref,
        organizacion_ref, expediente_ref, version_expediente, accion,
        decision_vec_ref, rama, codigo_probatorio_vec,
        decision_vec_huella_sha256
    ),
    FOREIGN KEY (
        reserva_ref, organizacion_ref, expediente_ref, version_expediente,
        recibo_ref, actuacion_ref, auditoria_ref, decision_vec_ref
    ) REFERENCES
        vec_contratacion_temporal.reserva_operacion_decision_cobertura (
            reserva_ref, organizacion_ref, expediente_ref,
            version_expediente, recibo_ref, actuacion_ref,
            auditoria_ref, decision_vec_ref
        ) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (
        catalogo_ref, catalogo_version, catalogo_huella_sha256
    ) REFERENCES vec_contratacion_temporal.gobi_o404b_catalogo (
        referencia, version, huella_sha256
    ) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (
        politica_ref, politica_version, politica_huella_sha256
    ) REFERENCES vec_contratacion_temporal.gobi_o404b_politica (
        referencia, version, huella_sha256
    ) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (
        actuacion_gobierno_ref, actuacion_gobierno_version,
        actuacion_gobierno_huella_sha256
    ) REFERENCES vec_contratacion_temporal.gobi_o404b_actuacion (
        referencia, version, huella_sha256
    ) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (
        checkpoint_secuencia, checkpoint_evento_ref,
        checkpoint_evento_huella_sha256
    ) REFERENCES vec_contratacion_temporal.gobi_o404b_evento (
        secuencia, evento_ref, huella_evento_sha256
    ) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (
        decision_vec_ref, rama, codigo_probatorio_vec, accion,
        decision_vec_huella_sha256, organizacion_ref, expediente_ref,
        version_expediente, reserva_ref
    ) REFERENCES vec_autorizacion.enlace_decision_cobertura_ct_o404e (
        decision_ref, rama, codigo, accion, decision_huella_sha256,
        organizacion_ref, expediente_ref, version_expediente, reserva_ref
    ) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CHECK (
        acreditacion_ref =
            'gobierno-cobertura:sha256:' || gobierno_huella_sha256
        AND gobierno_huella_sha256 ~ '^[a-f0-9]{64}$'
        AND gobierno_huella_sha256 = pg_catalog.encode(
            pg_catalog.sha256(prueba_canonica), 'hex'
        )
        AND pg_catalog.octet_length(prueba_canonica)
            BETWEEN 128 AND 32768
    ),
    CHECK (
        organizacion_ref ~
            '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND expediente_ref ~
            '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND reserva_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND recibo_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND actuacion_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND auditoria_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND version_expediente BETWEEN 2 AND 9007199254740990::numeric
        AND accion IN (
            'contratacion_temporal.cobertura.decidir',
            'contratacion_temporal.cobertura.rectificar'
        )
        AND rama = 'concedida'
        AND codigo_probatorio_vec = 'concedida'
    ),
    CHECK (
        decision_vec_huella_sha256 ~ '^[a-f0-9]{64}$'
        AND catalogo_huella_sha256 ~ '^[a-f0-9]{64}$'
        AND politica_huella_sha256 ~ '^[a-f0-9]{64}$'
        AND actuacion_gobierno_huella_sha256 ~ '^[a-f0-9]{64}$'
        AND checkpoint_evento_huella_sha256 ~ '^[a-f0-9]{64}$'
        AND decision_vec_huella_sha256 <> pg_catalog.repeat('0', 64)
        AND catalogo_huella_sha256 <> pg_catalog.repeat('0', 64)
        AND politica_huella_sha256 <> pg_catalog.repeat('0', 64)
        AND actuacion_gobierno_huella_sha256 <>
            pg_catalog.repeat('0', 64)
        AND checkpoint_evento_huella_sha256 <>
            pg_catalog.repeat('0', 64)
    ),
    CHECK (
        evaluada_en = pg_catalog.date_trunc('microseconds', evaluada_en)
        AND acreditada_en =
            pg_catalog.date_trunc('microseconds', acreditada_en)
        AND acreditada_en >= evaluada_en
    )
);
CREATE TABLE
vec_contratacion_temporal.decision_cobertura_gobernada_durable (
    decision_ref text PRIMARY KEY,
    decision_huella_sha256 text NOT NULL UNIQUE,
    decision_json jsonb NOT NULL,
    decision_canon bytea NOT NULL,
    propuesta_ref text NOT NULL,
    propuesta_huella_sha256 text NOT NULL,
    propuesta_json jsonb NOT NULL,
    propuesta_canon bytea NOT NULL,
    tipo text NOT NULL,
    organizacion_ref text NOT NULL,
    expediente_ref text NOT NULL,
    version_expediente_origen numeric(20, 0) NOT NULL,
    version_expediente numeric(20, 0) NOT NULL,
    reserva_ref text NOT NULL UNIQUE,
    recibo_ref text NOT NULL UNIQUE,
    actuacion_ref text NOT NULL UNIQUE,
    auditoria_ref text NOT NULL UNIQUE,
    accion text NOT NULL,
    actor_ref text NOT NULL,
    perfil_ref text NOT NULL,
    via_elegida text NOT NULL,
    via_recomendada text NOT NULL,
    predecesora_ref text,
    predecesora_huella_sha256 text,
    acreditacion_gobierno_ref text NOT NULL UNIQUE,
    gobierno_huella_sha256 text NOT NULL,
    consumo_c1_lote_ref text NOT NULL UNIQUE,
    consumo_c1_lote_huella_sha256 text NOT NULL,
    decision_vec_ref text NOT NULL UNIQUE,
    rama text NOT NULL DEFAULT 'concedida',
    codigo_probatorio_vec text NOT NULL DEFAULT 'concedida',
    decision_vec_huella_sha256 text NOT NULL,
    decidida_en timestamptz(6) NOT NULL,
    persistida_en timestamptz(6) NOT NULL,
    UNIQUE (decision_ref, decision_huella_sha256),
    UNIQUE (
        decision_ref, decision_huella_sha256, codigo_probatorio_vec
    ),
    UNIQUE (
        decision_ref, decision_huella_sha256, reserva_ref,
        organizacion_ref, expediente_ref, version_expediente_origen,
        recibo_ref, actuacion_ref, auditoria_ref, accion, decision_vec_ref,
        codigo_probatorio_vec
    ),
    FOREIGN KEY (expediente_ref, version_expediente)
        REFERENCES vec_contratacion_temporal.expediente_version_integral (
            expediente_ref, version
        ) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (
        reserva_ref, organizacion_ref, expediente_ref,
        version_expediente_origen, recibo_ref, actuacion_ref,
        auditoria_ref, decision_vec_ref
    ) REFERENCES
        vec_contratacion_temporal.reserva_operacion_decision_cobertura (
            reserva_ref, organizacion_ref, expediente_ref,
            version_expediente, recibo_ref, actuacion_ref,
            auditoria_ref, decision_vec_ref
        ) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (
        actuacion_ref, expediente_ref, version_expediente
    ) REFERENCES vec_contratacion_temporal.actuacion_expediente_integral (
        operacion_ref, expediente_ref, version_expediente
    ) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (
        acreditacion_gobierno_ref, gobierno_huella_sha256, reserva_ref,
        organizacion_ref, expediente_ref, version_expediente_origen,
        accion, decision_vec_ref, rama, codigo_probatorio_vec,
        decision_vec_huella_sha256
    ) REFERENCES
        vec_contratacion_temporal.acreditacion_gobierno_decision_cobertura (
            acreditacion_ref, gobierno_huella_sha256, reserva_ref,
            organizacion_ref, expediente_ref, version_expediente, accion,
            decision_vec_ref, rama, codigo_probatorio_vec,
            decision_vec_huella_sha256
        ) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (
        consumo_c1_lote_ref, consumo_c1_lote_huella_sha256,
        organizacion_ref, expediente_ref, version_expediente_origen,
        reserva_ref, decision_vec_ref, decision_vec_huella_sha256,
        codigo_probatorio_vec
    ) REFERENCES vec_contratacion_temporal.consumo_cobertura_lote (
        lote_ref, lote_huella_sha256, organizacion_ref, expediente_ref,
        version_expediente, reserva_ref, decision_vec_ref,
        decision_vec_huella_sha256, codigo_probatorio_vec
        ) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (
        decision_vec_ref, rama, codigo_probatorio_vec, accion,
        decision_vec_huella_sha256, organizacion_ref, expediente_ref,
        version_expediente_origen, reserva_ref
    ) REFERENCES vec_autorizacion.enlace_decision_cobertura_ct_o404e (
        decision_ref, rama, codigo, accion, decision_huella_sha256,
        organizacion_ref, expediente_ref, version_expediente, reserva_ref
    ) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (predecesora_ref, predecesora_huella_sha256)
        REFERENCES
        vec_contratacion_temporal.decision_cobertura_gobernada_durable (
            decision_ref, decision_huella_sha256
        ) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CHECK (
        vec_contratacion_temporal
            .o404e_decision_cobertura_exacta_v1(decision_json)
        AND decision_ref = decision_json ->> 'referencia'
        AND decision_huella_sha256 = decision_json ->> 'huella_sha256'
        AND decision_canon =
            vec_contratacion_temporal
                .o404e_material_decision_cobertura_v1(decision_json)
    ),
    CHECK (
        vec_contratacion_temporal
            .o404e_propuesta_cobertura_exacta_v1(propuesta_json)
        AND propuesta_ref = propuesta_json ->> 'referencia'
        AND propuesta_huella_sha256 = propuesta_json ->> 'huella_sha256'
        AND propuesta_canon =
            vec_contratacion_temporal
                .o404e_material_propuesta_cobertura_v1(propuesta_json)
        AND propuesta_huella_sha256 <> pg_catalog.repeat('0', 64)
    ),
    CHECK (
        decision_json ->> 'propuesta_ref' = propuesta_ref
        AND decision_json ->> 'propuesta_huella_sha256' =
            propuesta_huella_sha256
        AND decision_json ->> 'organizacion_ref' = organizacion_ref
        AND decision_json ->> 'expediente_ref' = expediente_ref
        AND (decision_json ->> 'version_expediente_origen')::numeric =
            version_expediente_origen
        AND (decision_json ->> 'version_expediente')::numeric =
            version_expediente
        AND decision_json ->> 'tipo' = tipo
        AND (decision_json ->> 'decidida_en')::timestamptz =
            decidida_en
        AND decision_json #>> '{actuacion,recibo_ref}' = recibo_ref
        AND decision_json #>> '{actuacion,accion_clave}' = accion
        AND decision_json #>> '{actuacion,accion_clave}' =
            CASE tipo
                WHEN 'inicial'
                    THEN 'contratacion_temporal.cobertura.decidir'
                ELSE 'contratacion_temporal.cobertura.rectificar'
            END
        AND decision_json ->> 'actor_ref' = actor_ref
        AND decision_json ->> 'perfil_ref' = perfil_ref
        AND decision_json ->> 'via_elegida' = via_elegida
        AND decision_json ->> 'via_recomendada' = via_recomendada
    ),
    CHECK (
        tipo IN ('inicial', 'rectificacion')
        AND rama = 'concedida'
        AND codigo_probatorio_vec = 'concedida'
        AND decision_huella_sha256 <> pg_catalog.repeat('0', 64)
        AND gobierno_huella_sha256 <> pg_catalog.repeat('0', 64)
        AND consumo_c1_lote_huella_sha256 <>
            pg_catalog.repeat('0', 64)
        AND decision_vec_huella_sha256 <> pg_catalog.repeat('0', 64)
        AND version_expediente_origen BETWEEN
            2 AND 9007199254740990::numeric
        AND version_expediente = version_expediente_origen + 1
        AND version_expediente BETWEEN 3 AND 9007199254740991::numeric
        AND decidida_en = pg_catalog.date_trunc(
            'microseconds', decidida_en
        )
        AND persistida_en = pg_catalog.date_trunc(
            'microseconds', persistida_en
        )
        AND persistida_en >= decidida_en
    ),
    CHECK (
        (
            tipo = 'inicial'
            AND predecesora_ref IS NULL
            AND predecesora_huella_sha256 IS NULL
        )
        OR (
            tipo = 'rectificacion'
            AND predecesora_ref =
                decision_json ->> 'predecesora_ref'
            AND predecesora_huella_sha256 =
                decision_json ->> 'predecesora_huella_sha256'
        )
    )
);

CREATE TABLE vec_contratacion_temporal.auditoria_decision_cobertura (
    auditoria_ref text PRIMARY KEY,
    secuencia numeric(20, 0) NOT NULL UNIQUE,
    rama text NOT NULL,
    reserva_ref text NOT NULL UNIQUE,
    recibo_ref text NOT NULL UNIQUE,
    huella_orden_sha256 text NOT NULL UNIQUE,
    organizacion_ref text NOT NULL,
    expediente_ref text NOT NULL,
    version_expediente_origen numeric(20, 0) NOT NULL,
    version_resultante numeric(20, 0),
    operacion_ref text NOT NULL,
    accion text NOT NULL,
    decision_vec_ref text NOT NULL UNIQUE,
    decision_vec_huella_sha256 text NOT NULL,
    codigo_probatorio_vec text NOT NULL,
    acreditacion_gobierno_ref text,
    gobierno_huella_sha256 text,
    consumo_c1_lote_ref text,
    consumo_c1_lote_huella_sha256 text,
    decision_cobertura_ref text,
    decision_cobertura_huella_sha256 text,
    actuacion_ref text,
    prueba_canonica bytea NOT NULL,
    anterior_sha256 text NOT NULL,
    huella_sha256 text NOT NULL UNIQUE,
    registrada_en timestamptz(6) NOT NULL,
    UNIQUE (
        auditoria_ref, rama, codigo_probatorio_vec, decision_vec_ref,
        decision_vec_huella_sha256
    ),
    FOREIGN KEY (
        reserva_ref, organizacion_ref, expediente_ref,
        version_expediente_origen, recibo_ref, operacion_ref,
        auditoria_ref, decision_vec_ref
    ) REFERENCES
        vec_contratacion_temporal.reserva_operacion_decision_cobertura (
            reserva_ref, organizacion_ref, expediente_ref,
            version_expediente, recibo_ref, actuacion_ref,
            auditoria_ref, decision_vec_ref
        ) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (expediente_ref, version_expediente_origen)
        REFERENCES vec_contratacion_temporal.expediente_version_integral (
            expediente_ref, version
        ) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (
        decision_vec_ref, rama, codigo_probatorio_vec, accion,
        decision_vec_huella_sha256, organizacion_ref, expediente_ref,
        version_expediente_origen, reserva_ref
    ) REFERENCES vec_autorizacion.enlace_decision_cobertura_ct_o404e (
        decision_ref, rama, codigo, accion, decision_huella_sha256,
        organizacion_ref, expediente_ref, version_expediente, reserva_ref
    ) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (
        acreditacion_gobierno_ref, gobierno_huella_sha256, reserva_ref,
        organizacion_ref, expediente_ref, version_expediente_origen,
        accion, decision_vec_ref, rama, codigo_probatorio_vec,
        decision_vec_huella_sha256
    ) REFERENCES
        vec_contratacion_temporal.acreditacion_gobierno_decision_cobertura (
            acreditacion_ref, gobierno_huella_sha256, reserva_ref,
            organizacion_ref, expediente_ref, version_expediente, accion,
            decision_vec_ref, rama, codigo_probatorio_vec,
            decision_vec_huella_sha256
        ) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (
        consumo_c1_lote_ref, consumo_c1_lote_huella_sha256,
        organizacion_ref, expediente_ref, version_expediente_origen,
        reserva_ref, decision_vec_ref, decision_vec_huella_sha256,
        codigo_probatorio_vec
    ) REFERENCES vec_contratacion_temporal.consumo_cobertura_lote (
        lote_ref, lote_huella_sha256, organizacion_ref, expediente_ref,
        version_expediente, reserva_ref, decision_vec_ref,
        decision_vec_huella_sha256, codigo_probatorio_vec
    ) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (
        decision_cobertura_ref, decision_cobertura_huella_sha256,
        reserva_ref, organizacion_ref, expediente_ref,
        version_expediente_origen, recibo_ref, operacion_ref,
        auditoria_ref, accion, decision_vec_ref, codigo_probatorio_vec
    ) REFERENCES
        vec_contratacion_temporal.decision_cobertura_gobernada_durable (
            decision_ref, decision_huella_sha256, reserva_ref,
            organizacion_ref, expediente_ref, version_expediente_origen,
            recibo_ref, actuacion_ref, auditoria_ref, accion,
            decision_vec_ref, codigo_probatorio_vec
        ) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (
        actuacion_ref, expediente_ref, version_resultante
    ) REFERENCES vec_contratacion_temporal.actuacion_expediente_integral (
        operacion_ref, expediente_ref, version_expediente
    ) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CHECK (
        secuencia BETWEEN 1 AND 9007199254740991::numeric
        AND huella_orden_sha256 ~ '^[a-f0-9]{64}$'
        AND decision_vec_huella_sha256 ~ '^[a-f0-9]{64}$'
        AND operacion_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND accion IN (
            'contratacion_temporal.cobertura.decidir',
            'contratacion_temporal.cobertura.rectificar'
        )
        AND anterior_sha256 ~ '^[a-f0-9]{64}$'
        AND huella_sha256 = pg_catalog.encode(
            pg_catalog.sha256(
                anterior_sha256::bytea || prueba_canonica
            ), 'hex'
        )
        AND pg_catalog.octet_length(prueba_canonica)
            BETWEEN 128 AND 32768
        AND huella_orden_sha256 <> pg_catalog.repeat('0', 64)
        AND decision_vec_huella_sha256 <> pg_catalog.repeat('0', 64)
        AND (
            (
                secuencia = 1
                AND anterior_sha256 = pg_catalog.repeat('0', 64)
            )
            OR (
                secuencia > 1
                AND anterior_sha256 <> pg_catalog.repeat('0', 64)
            )
        )
        AND huella_sha256 <> pg_catalog.repeat('0', 64)
        AND registrada_en =
            pg_catalog.date_trunc('microseconds', registrada_en)
    ),
    CHECK (
        (
            rama = 'concedida'
            AND codigo_probatorio_vec = 'concedida'
            AND acreditacion_gobierno_ref IS NOT NULL
            AND gobierno_huella_sha256 IS NOT NULL
            AND consumo_c1_lote_ref IS NOT NULL
            AND consumo_c1_lote_huella_sha256 IS NOT NULL
            AND decision_cobertura_ref IS NOT NULL
            AND decision_cobertura_huella_sha256 IS NOT NULL
            AND actuacion_ref IS NOT NULL
            AND operacion_ref = actuacion_ref
            AND version_resultante = version_expediente_origen + 1
        )
        OR (
            rama = 'denegada'
            AND codigo_probatorio_vec IN (
                'perfil_no_vigente',
                'ambito_no_autorizado',
                'rol_no_publicado',
                'rol_retirado',
                'accion_no_concedida',
                'finalidad_no_autorizada',
                'denegada_por_politica',
                'restriccion_abac_incumplida',
                'garantia_insuficiente'
            )
            AND acreditacion_gobierno_ref IS NULL
            AND gobierno_huella_sha256 IS NULL
            AND consumo_c1_lote_ref IS NULL
            AND consumo_c1_lote_huella_sha256 IS NULL
            AND decision_cobertura_ref IS NULL
            AND decision_cobertura_huella_sha256 IS NULL
            AND actuacion_ref IS NULL
            AND version_resultante IS NULL
        )
    )
);
ALTER TABLE
vec_contratacion_temporal.confirmacion_operacion_decision_cobertura
    ADD CONSTRAINT confirmacion_cobertura_enlace_vec_o404e_fk
    FOREIGN KEY (
        decision_vec_ref, rama, codigo_probatorio_vec,
        decision_vec_huella_sha256
    ) REFERENCES vec_autorizacion.enlace_decision_cobertura_ct_o404e (
        decision_ref, rama, codigo, decision_huella_sha256
    ) ON UPDATE RESTRICT ON DELETE RESTRICT,
    ADD CONSTRAINT confirmacion_cobertura_auditoria_o404e_fk
    FOREIGN KEY (
        auditoria_ref, rama, codigo_probatorio_vec, decision_vec_ref,
        decision_vec_huella_sha256
    ) REFERENCES vec_contratacion_temporal.auditoria_decision_cobertura (
        auditoria_ref, rama, codigo_probatorio_vec, decision_vec_ref,
        decision_vec_huella_sha256
    ) ON UPDATE RESTRICT ON DELETE RESTRICT,
    ADD CONSTRAINT confirmacion_cobertura_decision_o404e_fk
    FOREIGN KEY (
        decision_cobertura_ref, decision_cobertura_huella_sha256,
        codigo_probatorio_vec
    ) REFERENCES
        vec_contratacion_temporal.decision_cobertura_gobernada_durable (
            decision_ref, decision_huella_sha256, codigo_probatorio_vec
        ) ON UPDATE RESTRICT ON DELETE RESTRICT;
ALTER TABLE vec_contratacion_temporal.terminal_operacion_decision_cobertura
    ADD COLUMN decision_vec_huella_sha256 text NOT NULL,
    ADD COLUMN codigo_probatorio_vec text NOT NULL,
    ADD COLUMN decision_cobertura_huella_sha256 text,
    ADD CONSTRAINT terminal_cobertura_enlace_vec_o404e_fk
    FOREIGN KEY (
        decision_vec_ref, rama, codigo_probatorio_vec,
        decision_vec_huella_sha256
    ) REFERENCES vec_autorizacion.enlace_decision_cobertura_ct_o404e (
        decision_ref, rama, codigo, decision_huella_sha256
    ) ON UPDATE RESTRICT ON DELETE RESTRICT,
    ADD CONSTRAINT terminal_cobertura_auditoria_o404e_fk
    FOREIGN KEY (
        auditoria_ref, rama, codigo_probatorio_vec, decision_vec_ref,
        decision_vec_huella_sha256
    ) REFERENCES vec_contratacion_temporal.auditoria_decision_cobertura (
        auditoria_ref, rama, codigo_probatorio_vec, decision_vec_ref,
        decision_vec_huella_sha256
    ) ON UPDATE RESTRICT ON DELETE RESTRICT,
    ADD CONSTRAINT terminal_cobertura_gobierno_o404e_fk
    FOREIGN KEY (gobierno_ref, gobierno_huella_sha256)
    REFERENCES
        vec_contratacion_temporal.acreditacion_gobierno_decision_cobertura (
            acreditacion_ref, gobierno_huella_sha256
        ) ON UPDATE RESTRICT ON DELETE RESTRICT,
    ADD CONSTRAINT terminal_cobertura_gobierno_compuesto_o404e_fk
    FOREIGN KEY (
        gobierno_ref, gobierno_huella_sha256, recibo_ref,
        auditoria_ref, decision_vec_ref
    ) REFERENCES
        vec_contratacion_temporal.acreditacion_gobierno_decision_cobertura (
            acreditacion_ref, gobierno_huella_sha256, recibo_ref,
            auditoria_ref, decision_vec_ref
        ) ON UPDATE RESTRICT ON DELETE RESTRICT,
    ADD CONSTRAINT terminal_cobertura_lote_o404e_fk
    FOREIGN KEY (
        consumo_c1_lote_ref, consumo_c1_lote_huella_sha256
    ) REFERENCES vec_contratacion_temporal.consumo_cobertura_lote (
        lote_ref, lote_huella_sha256
    ) ON UPDATE RESTRICT ON DELETE RESTRICT,
    ADD CONSTRAINT terminal_cobertura_lote_compuesto_o404e_fk
    FOREIGN KEY (
        consumo_c1_lote_ref, consumo_c1_lote_huella_sha256,
        huella_orden_sha256, decision_vec_ref
    ) REFERENCES vec_contratacion_temporal.consumo_cobertura_lote (
        lote_ref, lote_huella_sha256, huella_orden_sha256,
        decision_vec_ref
    ) ON UPDATE RESTRICT ON DELETE RESTRICT,
    ADD CONSTRAINT terminal_cobertura_decision_o404e_fk
    FOREIGN KEY (
        decision_cobertura_ref, decision_cobertura_huella_sha256,
        codigo_probatorio_vec
    ) REFERENCES
        vec_contratacion_temporal.decision_cobertura_gobernada_durable (
            decision_ref, decision_huella_sha256, codigo_probatorio_vec
        ) ON UPDATE RESTRICT ON DELETE RESTRICT,
    ADD CONSTRAINT terminal_cobertura_actuacion_o404e_fk
    FOREIGN KEY (actuacion_ref)
    REFERENCES vec_contratacion_temporal.actuacion_expediente_integral (
        operacion_ref
    ) ON UPDATE RESTRICT ON DELETE RESTRICT,
    ADD CONSTRAINT terminal_cobertura_outbox_o404e_fk
    FOREIGN KEY (outbox_ref)
    REFERENCES vec_contratacion_temporal.outbox_expediente_integral (
        evento_ref
    ) ON UPDATE RESTRICT ON DELETE RESTRICT,
    ADD CONSTRAINT terminal_cobertura_outbox_compuesto_o404e_fk
    FOREIGN KEY (
        outbox_ref, recibo_ref, auditoria_ref, decision_vec_ref, rama
    ) REFERENCES vec_contratacion_temporal.outbox_expediente_integral (
        evento_ref, o404e_recibo_ref, o404e_auditoria_ref,
        o404e_decision_vec_ref, o404e_rama
    ) ON UPDATE RESTRICT ON DELETE RESTRICT,
    ADD CONSTRAINT terminal_cobertura_confirmacion_exacta_o404e_fk
    FOREIGN KEY (
        ambito_raiz_hmac, recibo_ref, huella_orden_sha256, rama,
        decision_vec_ref, auditoria_ref, codigo_probatorio_vec,
        decision_cobertura_ref, decision_cobertura_huella_sha256,
        version_resultante, actuacion_ref
    ) REFERENCES
        vec_contratacion_temporal.confirmacion_operacion_decision_cobertura (
            ambito_raiz_hmac, recibo_ref, huella_orden_sha256, rama,
            decision_vec_ref, auditoria_ref, codigo_probatorio_vec,
            decision_cobertura_ref, decision_cobertura_huella_sha256,
            version_resultante, actuacion_ref
        ) ON UPDATE RESTRICT ON DELETE RESTRICT,
    ADD CONSTRAINT terminal_cobertura_confirmacion_base_o404e_fk
    FOREIGN KEY (
        ambito_raiz_hmac, recibo_ref, huella_orden_sha256, rama,
        decision_vec_ref, auditoria_ref, codigo_probatorio_vec
    ) REFERENCES
        vec_contratacion_temporal.confirmacion_operacion_decision_cobertura (
            ambito_raiz_hmac, recibo_ref, huella_orden_sha256, rama,
            decision_vec_ref, auditoria_ref, codigo_probatorio_vec
        ) ON UPDATE RESTRICT ON DELETE RESTRICT,
    ADD CONSTRAINT terminal_cobertura_huellas_rama_o404e_ck CHECK (
        decision_vec_huella_sha256 ~ '^[a-f0-9]{64}$'
        AND decision_vec_huella_sha256 <> pg_catalog.repeat('0', 64)
        AND (
            (
                rama = 'concedida'
                AND codigo_probatorio_vec = 'concedida'
                AND decision_cobertura_huella_sha256 ~ '^[a-f0-9]{64}$'
                AND decision_cobertura_huella_sha256 <>
                    pg_catalog.repeat('0', 64)
            )
            OR (
                rama = 'denegada'
                AND codigo_probatorio_vec IN (
                    'perfil_no_vigente',
                    'ambito_no_autorizado',
                    'rol_no_publicado',
                    'rol_retirado',
                    'accion_no_concedida',
                    'finalidad_no_autorizada',
                    'denegada_por_politica',
                    'restriccion_abac_incumplida',
                    'garantia_insuficiente'
                )
                AND decision_cobertura_huella_sha256 IS NULL
            )
        )
    );

DO $protecciones$
DECLARE
    v_tabla text;
BEGIN
    FOREACH v_tabla IN ARRAY ARRAY[
        'acreditacion_gobierno_decision_cobertura',
        'decision_cobertura_gobernada_durable',
        'auditoria_decision_cobertura'
    ]::text[]
    LOOP
        EXECUTE pg_catalog.format(
            'CREATE TRIGGER bloquear_mutacion ' ||
            'BEFORE UPDATE OR DELETE ON vec_contratacion_temporal.%I ' ||
            'FOR EACH ROW EXECUTE FUNCTION ' ||
            'vec_contratacion_temporal.rechazar_mutacion_historia_v1()',
            v_tabla
        );
        EXECUTE pg_catalog.format(
            'CREATE TRIGGER bloquear_truncado ' ||
            'BEFORE TRUNCATE ON vec_contratacion_temporal.%I ' ||
            'FOR EACH STATEMENT EXECUTE FUNCTION ' ||
            'vec_contratacion_temporal.rechazar_mutacion_historia_v1()',
            v_tabla
        );
        EXECUTE pg_catalog.format(
            'ALTER TABLE vec_contratacion_temporal.%I ' ||
            'ENABLE ROW LEVEL SECURITY',
            v_tabla
        );
        EXECUTE pg_catalog.format(
            'ALTER TABLE vec_contratacion_temporal.%I ' ||
            'FORCE ROW LEVEL SECURITY',
            v_tabla
        );
        EXECUTE pg_catalog.format(
            'CREATE POLICY propietario_total ON ' ||
            'vec_contratacion_temporal.%I ' ||
            'TO vec_contratacion_temporal_propietario ' ||
            'USING (true) WITH CHECK (true)',
            v_tabla
        );
    END LOOP;
END
$protecciones$;

REVOKE ALL ON TABLE
    vec_contratacion_temporal.acreditacion_gobierno_decision_cobertura,
    vec_contratacion_temporal.decision_cobertura_gobernada_durable,
    vec_contratacion_temporal.auditoria_decision_cobertura
FROM PUBLIC, vec_contratacion_temporal_ejecutor,
     vec_contratacion_temporal_migrador,
     vec_contratacion_temporal_gobernador;

UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4
   SET version_esquema = 7,
       actualizada_en = pg_catalog.date_trunc(
           'microseconds', pg_catalog.clock_timestamp()
       )
 WHERE control AND version_esquema = 6;

COMMIT;
