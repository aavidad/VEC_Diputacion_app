-- O4-04E/1: enlace durable entre la decisión VEC y su efecto CT.
-- La función exterior CT todavía no se publica en este corte.
BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contratacion_temporal:o4_04:migraciones', 0
    )
);

DO $prevalidacion$
BEGIN
    IF pg_catalog.to_regprocedure(
           'vec_autorizacion.registrar_decision_cobertura_contratacion_temporal_v1(bytea,bytea,numeric,numeric,jsonb)'
       ) IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_autorizacion.o404d_registrar_decision_cobertura_sin_enlace_v1(bytea,bytea,numeric,numeric,jsonb)'
       ) IS NOT NULL
       OR pg_catalog.to_regclass(
           'vec_autorizacion.enlace_decision_cobertura_ct_o404e'
       ) IS NOT NULL
       OR NOT EXISTS (
           SELECT 1
             FROM pg_catalog.pg_roles
            WHERE rolname = 'vec_contratacion_temporal_propietario'
              AND NOT rolcanlogin AND NOT rolsuper AND NOT rolbypassrls
       ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'estado incompatible para enlace VEC-CT O4-04E';
    END IF;
END
$prevalidacion$;

ALTER TABLE vec_autorizacion.decision_concedida_contexto_actor_v3
    ADD COLUMN codigo_probatorio_o404e text GENERATED ALWAYS AS (
        documento ->> 'codigo'
    ) STORED NOT NULL,
    ADD COLUMN accion_o404e text GENERATED ALWAYS AS (
        documento ->> 'accion'
    ) STORED NOT NULL,
    ADD COLUMN correlacion_ref_o404e text GENERATED ALWAYS AS (
        documento ->> 'correlacion_ref'
    ) STORED NOT NULL,
    ADD COLUMN recurso_ref_o404e text GENERATED ALWAYS AS (
        documento ->> 'recurso_ref'
    ) STORED NOT NULL,
    ADD COLUMN contexto_recurso_huella_o404e text GENERATED ALWAYS AS (
        documento ->> 'contexto_recurso_huella_sha256'
    ) STORED NOT NULL,
    ADD CONSTRAINT decision_concedida_v3_identidad_compuesta_o404e
    UNIQUE (
        decision_ref, huella_decision_sha256,
        codigo_probatorio_o404e, accion_o404e,
        correlacion_ref_o404e, recurso_ref_o404e,
        contexto_recurso_huella_o404e
    );
ALTER TABLE vec_autorizacion.decision_denegada_contexto_actor_v3
    ADD COLUMN codigo_probatorio_o404e text GENERATED ALWAYS AS (
        documento ->> 'codigo'
    ) STORED NOT NULL,
    ADD COLUMN accion_o404e text GENERATED ALWAYS AS (
        documento ->> 'accion'
    ) STORED NOT NULL,
    ADD COLUMN correlacion_ref_o404e text GENERATED ALWAYS AS (
        documento ->> 'correlacion_ref'
    ) STORED NOT NULL,
    ADD COLUMN recurso_ref_o404e text GENERATED ALWAYS AS (
        documento ->> 'recurso_ref'
    ) STORED NOT NULL,
    ADD COLUMN contexto_recurso_huella_o404e text GENERATED ALWAYS AS (
        documento ->> 'contexto_recurso_huella_sha256'
    ) STORED NOT NULL,
    ADD CONSTRAINT decision_denegada_v3_identidad_compuesta_o404e
    UNIQUE (
        decision_ref, huella_decision_sha256,
        codigo_probatorio_o404e, accion_o404e,
        correlacion_ref_o404e, recurso_ref_o404e,
        contexto_recurso_huella_o404e
    );

CREATE TABLE vec_autorizacion.enlace_decision_cobertura_ct_o404e (
    decision_ref text PRIMARY KEY,
    rama text NOT NULL,
    concedida boolean NOT NULL,
    codigo text NOT NULL,
    accion text NOT NULL,
    decision_huella_sha256 text NOT NULL,
    decision_concedida_ref text,
    decision_denegada_ref text,
    correlacion_ref text NOT NULL,
    organizacion_ref text NOT NULL,
    expediente_ref text NOT NULL,
    version_expediente numeric(20, 0) NOT NULL,
    reserva_ref text NOT NULL,
    contexto_recurso_huella_sha256 text NOT NULL,
    huella_orden_sha256 text NOT NULL,
    lote_huella_sha256 text,
    prueba_vinculo_sha256 text NOT NULL,
    registrada_en timestamptz(6) NOT NULL,
    revalidada_en timestamptz(6),
    vinculada_en timestamptz(6) NOT NULL,
    UNIQUE (decision_ref, rama, codigo, decision_huella_sha256),
    UNIQUE (
        decision_ref, rama, codigo, accion, decision_huella_sha256,
        organizacion_ref, expediente_ref, version_expediente, reserva_ref
    ),
    UNIQUE (
        decision_ref, rama, codigo, accion, decision_huella_sha256,
        correlacion_ref, organizacion_ref, expediente_ref,
        version_expediente, reserva_ref,
        contexto_recurso_huella_sha256, huella_orden_sha256
    ),
    UNIQUE (prueba_vinculo_sha256),
    FOREIGN KEY (
        decision_concedida_ref, decision_huella_sha256, codigo, accion,
        correlacion_ref, reserva_ref, contexto_recurso_huella_sha256
    )
        REFERENCES vec_autorizacion.decision_concedida_contexto_actor_v3 (
            decision_ref, huella_decision_sha256,
            codigo_probatorio_o404e, accion_o404e,
            correlacion_ref_o404e, recurso_ref_o404e,
            contexto_recurso_huella_o404e
        ) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (
        decision_denegada_ref, decision_huella_sha256, codigo, accion,
        correlacion_ref, reserva_ref, contexto_recurso_huella_sha256
    )
        REFERENCES vec_autorizacion.decision_denegada_contexto_actor_v3 (
            decision_ref, huella_decision_sha256,
            codigo_probatorio_o404e, accion_o404e,
            correlacion_ref_o404e, recurso_ref_o404e,
            contexto_recurso_huella_o404e
        ) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CHECK (
        decision_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND correlacion_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND organizacion_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND expediente_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND reserva_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND version_expediente BETWEEN 2 AND 9007199254740990::numeric
        AND accion IN (
            'contratacion_temporal.cobertura.decidir',
            'contratacion_temporal.cobertura.rectificar'
        )
    ),
    CHECK (
        decision_huella_sha256 ~ '^[a-f0-9]{64}$'
        AND contexto_recurso_huella_sha256 ~ '^[a-f0-9]{64}$'
        AND huella_orden_sha256 ~ '^[a-f0-9]{64}$'
        AND prueba_vinculo_sha256 ~ '^[a-f0-9]{64}$'
        AND decision_huella_sha256 <> pg_catalog.repeat('0', 64)
        AND contexto_recurso_huella_sha256 <> pg_catalog.repeat('0', 64)
        AND huella_orden_sha256 <> pg_catalog.repeat('0', 64)
        AND prueba_vinculo_sha256 <> pg_catalog.repeat('0', 64)
    ),
    CHECK (
        registrada_en = pg_catalog.date_trunc(
            'microseconds', registrada_en
        )
        AND vinculada_en = pg_catalog.date_trunc(
            'microseconds', vinculada_en
        )
        AND vinculada_en >= registrada_en
        AND (
            revalidada_en IS NULL
            OR (
                revalidada_en = pg_catalog.date_trunc(
                    'microseconds', revalidada_en
                )
                AND revalidada_en >= registrada_en
                AND vinculada_en >= revalidada_en
            )
        )
    ),
    CHECK (
        (
            rama = 'concedida'
            AND concedida
            AND codigo = 'concedida'
            AND decision_concedida_ref = decision_ref
            AND decision_denegada_ref IS NULL
            AND lote_huella_sha256 ~ '^[a-f0-9]{64}$'
            AND lote_huella_sha256 <> pg_catalog.repeat('0', 64)
            AND revalidada_en IS NOT NULL
        )
        OR (
            rama = 'denegada'
            AND NOT concedida
            AND codigo IN (
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
            AND decision_concedida_ref IS NULL
            AND decision_denegada_ref = decision_ref
            AND lote_huella_sha256 IS NULL
            AND revalidada_en IS NULL
        )
    )
);

ALTER TABLE vec_autorizacion.enlace_decision_cobertura_ct_o404e
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_autorizacion.enlace_decision_cobertura_ct_o404e
    FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_total
    ON vec_autorizacion.enlace_decision_cobertura_ct_o404e
    TO vec_autorizacion_propietario
    USING (true) WITH CHECK (true);
CREATE TRIGGER enlace_decision_cobertura_inmutable
    BEFORE UPDATE OR DELETE
    ON vec_autorizacion.enlace_decision_cobertura_ct_o404e
    FOR EACH ROW
    EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable();
CREATE TRIGGER enlace_decision_cobertura_no_truncar
    BEFORE TRUNCATE
    ON vec_autorizacion.enlace_decision_cobertura_ct_o404e
    FOR EACH STATEMENT
    EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable();

ALTER FUNCTION
vec_autorizacion.registrar_decision_cobertura_contratacion_temporal_v1(
    bytea, bytea, numeric, numeric, jsonb
) RENAME TO o404d_registrar_decision_cobertura_sin_enlace_v1;

REVOKE ALL ON FUNCTION
vec_autorizacion.o404d_registrar_decision_cobertura_sin_enlace_v1(
    bytea, bytea, numeric, numeric, jsonb
) FROM PUBLIC, vec_autorizacion_registro, vec_autorizacion_fuente,
       vec_autorizacion_motivos_proyector,
       vec_autorizacion_motivos_evaluador,
       vec_contratacion_temporal_ejecutor,
       vec_contratacion_temporal_migrador,
       vec_contratacion_temporal_gobernador,
       vec_contratacion_temporal_propietario;

CREATE FUNCTION
vec_autorizacion.registrar_decision_cobertura_contratacion_temporal_v1(
    p_decision_canonica bytea,
    p_motivo_canonico bytea,
    p_persona_version numeric,
    p_perfil_version numeric,
    p_vinculo_efecto jsonb
)
RETURNS TABLE (
    rama text,
    concedida boolean,
    codigo text,
    decision_ref text,
    correlacion_ref text,
    organizacion_ref text,
    expediente_ref text,
    version_expediente numeric,
    reserva_ref text,
    contexto_recurso_huella_sha256 text,
    decision_huella_sha256 text,
    huella_orden_sha256 text,
    lote_huella_sha256 text,
    prueba_vinculo_sha256 text,
    registrada_en timestamptz,
    revalidada_en timestamptz
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
SET row_security = 'on'
SET timezone = 'UTC'
SET lock_timeout = '2s'
AS $funcion$
DECLARE
    v record;
    v_vinculada_en timestamptz(6);
BEGIN
    SELECT * INTO v
      FROM vec_autorizacion
           .o404d_registrar_decision_cobertura_sin_enlace_v1(
               p_decision_canonica,
               p_motivo_canonico,
               p_persona_version,
               p_perfil_version,
               p_vinculo_efecto
           );
    IF NOT FOUND THEN
        RETURN;
    END IF;

    v_vinculada_en := pg_catalog.date_trunc(
        'microseconds', pg_catalog.clock_timestamp()
    );
    INSERT INTO vec_autorizacion.enlace_decision_cobertura_ct_o404e (
        decision_ref, rama, concedida, codigo, accion,
        decision_huella_sha256,
        decision_concedida_ref, decision_denegada_ref, correlacion_ref,
        organizacion_ref, expediente_ref, version_expediente, reserva_ref,
        contexto_recurso_huella_sha256, huella_orden_sha256,
        lote_huella_sha256, prueba_vinculo_sha256, registrada_en,
        revalidada_en, vinculada_en
    ) VALUES (
        v.decision_ref, v.rama, v.concedida, v.codigo,
        p_vinculo_efecto ->> 'accion',
        v.decision_huella_sha256,
        CASE WHEN v.rama = 'concedida' THEN v.decision_ref END,
        CASE WHEN v.rama = 'denegada' THEN v.decision_ref END,
        v.correlacion_ref, v.organizacion_ref, v.expediente_ref,
        v.version_expediente, v.reserva_ref,
        v.contexto_recurso_huella_sha256, v.huella_orden_sha256,
        v.lote_huella_sha256, v.prueba_vinculo_sha256,
        v.registrada_en, v.revalidada_en, v_vinculada_en
    ) ON CONFLICT (decision_ref) DO NOTHING;

    PERFORM 1
      FROM vec_autorizacion.enlace_decision_cobertura_ct_o404e AS e
     WHERE e.decision_ref = v.decision_ref
       AND e.rama = v.rama
       AND e.concedida = v.concedida
       AND e.codigo = v.codigo
       AND e.accion = p_vinculo_efecto ->> 'accion'
       AND e.decision_huella_sha256 = v.decision_huella_sha256
       AND e.correlacion_ref = v.correlacion_ref
       AND e.organizacion_ref = v.organizacion_ref
       AND e.expediente_ref = v.expediente_ref
       AND e.version_expediente = v.version_expediente
       AND e.reserva_ref = v.reserva_ref
       AND e.contexto_recurso_huella_sha256 =
           v.contexto_recurso_huella_sha256
       AND e.huella_orden_sha256 = v.huella_orden_sha256
       AND e.lote_huella_sha256 IS NOT DISTINCT FROM v.lote_huella_sha256
       AND e.prueba_vinculo_sha256 = v.prueba_vinculo_sha256
       AND e.registrada_en = v.registrada_en
       AND e.revalidada_en IS NOT DISTINCT FROM v.revalidada_en;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING
            ERRCODE = '23505',
            MESSAGE = 'colisión de enlace probatorio VEC-CT O4-04E';
    END IF;

    RETURN QUERY SELECT
        v.rama::text, v.concedida::boolean, v.codigo::text,
        v.decision_ref::text, v.correlacion_ref::text,
        v.organizacion_ref::text, v.expediente_ref::text,
        v.version_expediente::numeric, v.reserva_ref::text,
        v.contexto_recurso_huella_sha256::text,
        v.decision_huella_sha256::text, v.huella_orden_sha256::text,
        v.lote_huella_sha256::text, v.prueba_vinculo_sha256::text,
        v.registrada_en::timestamptz, v.revalidada_en::timestamptz;
END
$funcion$;

REVOKE ALL ON TABLE vec_autorizacion.enlace_decision_cobertura_ct_o404e
    FROM PUBLIC, vec_autorizacion_registro, vec_autorizacion_fuente,
         vec_autorizacion_motivos_proyector,
         vec_autorizacion_motivos_evaluador,
         vec_contratacion_temporal_ejecutor,
         vec_contratacion_temporal_migrador,
         vec_contratacion_temporal_gobernador,
         vec_contratacion_temporal_propietario;
REVOKE ALL ON FUNCTION
vec_autorizacion.registrar_decision_cobertura_contratacion_temporal_v1(
    bytea, bytea, numeric, numeric, jsonb
) FROM PUBLIC, vec_autorizacion_registro, vec_autorizacion_fuente,
       vec_autorizacion_motivos_proyector,
       vec_autorizacion_motivos_evaluador,
       vec_contratacion_temporal_ejecutor,
       vec_contratacion_temporal_migrador,
       vec_contratacion_temporal_gobernador;

GRANT REFERENCES (
    decision_ref, rama, codigo, accion, decision_huella_sha256,
    correlacion_ref, organizacion_ref, expediente_ref,
    version_expediente, reserva_ref,
    contexto_recurso_huella_sha256, huella_orden_sha256
) ON TABLE vec_autorizacion.enlace_decision_cobertura_ct_o404e
    TO vec_contratacion_temporal_propietario;
GRANT EXECUTE ON FUNCTION
vec_autorizacion.registrar_decision_cobertura_contratacion_temporal_v1(
    bytea, bytea, numeric, numeric, jsonb
) TO vec_contratacion_temporal_propietario;

COMMENT ON TABLE vec_autorizacion.enlace_decision_cobertura_ct_o404e IS
    'Enlace append-only VEC-CT O4-04E; no contiene PII y solo se materializa dentro del wrapper específico.';
COMMENT ON FUNCTION
vec_autorizacion.registrar_decision_cobertura_contratacion_temporal_v1(
    bytea, bytea, numeric, numeric, jsonb
) IS
    'Wrapper interno O4-04E: registra VEC y su enlace probatorio CT en la misma transacción; no es una función de canal.';

COMMIT;
