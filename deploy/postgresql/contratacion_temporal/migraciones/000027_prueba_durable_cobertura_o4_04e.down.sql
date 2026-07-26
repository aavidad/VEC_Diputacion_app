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
 WHERE control AND version_esquema = 7
 FOR UPDATE;

DO $prevalidacion$
BEGIN
    IF NOT EXISTS (
           SELECT 1
             FROM vec_contratacion_temporal.control_migracion_cobertura_o4
            WHERE control AND version_esquema = 7
       )
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.confirmar_operacion_decision_cobertura_o404e_v1(jsonb)'
       ) IS NOT NULL
       OR pg_catalog.to_regclass(
           'vec_contratacion_temporal.acreditacion_gobierno_decision_cobertura'
       ) IS NULL THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'retirada de prueba durable O4-04E fuera de orden';
    END IF;
END
$prevalidacion$;

DO $proteccion$
BEGIN
    IF (
        EXISTS (
            SELECT 1 FROM
            vec_contratacion_temporal.acreditacion_gobierno_decision_cobertura
        )
        OR EXISTS (
            SELECT 1 FROM
            vec_contratacion_temporal.decision_cobertura_gobernada_durable
        )
        OR EXISTS (
            SELECT 1 FROM
            vec_contratacion_temporal.auditoria_decision_cobertura
        )
    ) AND pg_catalog.current_setting(
        'vec.confirmar_destruccion_prueba_cobertura_o404e', true
    ) IS DISTINCT FROM
        'DESTRUIR_HISTORIA_PRUEBA_COBERTURA_O404E_IRREVERSIBLE' THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'retirada O4-04E protegida por historia durable';
    END IF;
END
$proteccion$;

ALTER TABLE vec_contratacion_temporal.terminal_operacion_decision_cobertura
    DROP CONSTRAINT terminal_cobertura_huellas_rama_o404e_ck,
    DROP CONSTRAINT terminal_cobertura_confirmacion_base_o404e_fk,
    DROP CONSTRAINT terminal_cobertura_confirmacion_exacta_o404e_fk,
    DROP CONSTRAINT terminal_cobertura_outbox_compuesto_o404e_fk,
    DROP CONSTRAINT terminal_cobertura_outbox_o404e_fk,
    DROP CONSTRAINT terminal_cobertura_actuacion_o404e_fk,
    DROP CONSTRAINT terminal_cobertura_decision_o404e_fk,
    DROP CONSTRAINT terminal_cobertura_lote_compuesto_o404e_fk,
    DROP CONSTRAINT terminal_cobertura_lote_o404e_fk,
    DROP CONSTRAINT terminal_cobertura_gobierno_compuesto_o404e_fk,
    DROP CONSTRAINT terminal_cobertura_gobierno_o404e_fk,
    DROP CONSTRAINT terminal_cobertura_auditoria_o404e_fk,
    DROP CONSTRAINT terminal_cobertura_enlace_vec_o404e_fk,
    DROP COLUMN decision_cobertura_huella_sha256,
    DROP COLUMN codigo_probatorio_vec,
    DROP COLUMN decision_vec_huella_sha256;

ALTER TABLE
vec_contratacion_temporal.confirmacion_operacion_decision_cobertura
    DROP CONSTRAINT confirmacion_cobertura_decision_o404e_fk,
    DROP CONSTRAINT confirmacion_cobertura_auditoria_o404e_fk,
    DROP CONSTRAINT confirmacion_cobertura_enlace_vec_o404e_fk,
    DROP CONSTRAINT confirmacion_cobertura_terminal_compuesta_o404e,
    DROP CONSTRAINT confirmacion_cobertura_terminal_base_o404e;

DROP TABLE vec_contratacion_temporal.auditoria_decision_cobertura RESTRICT;
DROP TABLE
    vec_contratacion_temporal.decision_cobertura_gobernada_durable
    RESTRICT;
DROP TABLE
    vec_contratacion_temporal.acreditacion_gobierno_decision_cobertura
    RESTRICT;

ALTER TABLE vec_contratacion_temporal.actuacion_expediente_integral
    DROP CONSTRAINT actuacion_integral_identidad_compuesta_o404e;
ALTER TABLE
vec_contratacion_temporal.reserva_operacion_decision_cobertura
    DROP CONSTRAINT reserva_cobertura_identidad_compuesta_o404e;
ALTER TABLE vec_contratacion_temporal.consumo_cobertura_lote
    DROP CONSTRAINT consumo_cobertura_lote_identidad_compuesta_o404e,
    DROP CONSTRAINT consumo_cobertura_lote_terminal_o404e,
    DROP CONSTRAINT consumo_cobertura_lote_ref_huella_o404e;
DROP TRIGGER ligar_outbox_terminal_o404e
ON vec_contratacion_temporal.outbox_expediente_integral;
ALTER TABLE vec_contratacion_temporal.outbox_expediente_integral
    DROP CONSTRAINT outbox_expediente_terminal_o404e,
    DROP COLUMN o404e_rama,
    DROP COLUMN o404e_decision_vec_ref,
    DROP COLUMN o404e_auditoria_ref,
    DROP COLUMN o404e_recibo_ref;
ALTER TABLE vec_contratacion_temporal.gobi_o404b_evento
    DROP CONSTRAINT gobi_o404b_evento_identidad_compuesta_o404e;

UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4
   SET version_esquema = 6,
       actualizada_en = pg_catalog.date_trunc(
           'microseconds', pg_catalog.clock_timestamp()
       )
 WHERE control AND version_esquema = 7;

COMMIT;
