BEGIN;
SET LOCAL ROLE vec_bolsa_importacion_convoca_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '2min';
SET LOCAL idle_in_transaction_session_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_bolsa_importacion_convoca:migraciones', 0
    )
);

DROP POLICY propietario_puntero_politica
    ON vec_bolsa_importacion_convoca.politica_retencion_actual;
DROP POLICY propietario_politica
    ON vec_bolsa_importacion_convoca.politica_retencion;
DROP POLICY propietario_outbox ON vec_bolsa_importacion_convoca.outbox;
DROP POLICY propietario_historia
    ON vec_bolsa_importacion_convoca.historia_estado;
DROP POLICY propietario_ejecucion
    ON vec_bolsa_importacion_convoca.ejecucion_retencion;
DROP POLICY propietario_decision
    ON vec_bolsa_importacion_convoca.decision_retencion;
DROP POLICY propietario_conciliacion
    ON vec_bolsa_importacion_convoca.conciliacion;
DROP POLICY propietario_staging
    ON vec_bolsa_importacion_convoca.fila_staging;
DROP POLICY propietario_lote ON vec_bolsa_importacion_convoca.lote;

ALTER TABLE vec_bolsa_importacion_convoca.politica_retencion_actual
    NO FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_importacion_convoca.politica_retencion_actual
    DISABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_importacion_convoca.politica_retencion
    NO FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_importacion_convoca.politica_retencion
    DISABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_importacion_convoca.outbox NO FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_importacion_convoca.outbox DISABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_importacion_convoca.historia_estado
    NO FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_importacion_convoca.historia_estado
    DISABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_importacion_convoca.ejecucion_retencion
    NO FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_importacion_convoca.ejecucion_retencion
    DISABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_importacion_convoca.decision_retencion
    NO FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_importacion_convoca.decision_retencion
    DISABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_importacion_convoca.conciliacion
    NO FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_importacion_convoca.conciliacion
    DISABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_importacion_convoca.fila_staging
    NO FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_importacion_convoca.fila_staging
    DISABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_importacion_convoca.lote NO FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_importacion_convoca.lote DISABLE ROW LEVEL SECURITY;
DROP FUNCTION vec_bolsa_importacion_convoca.lote_integro(text);
COMMIT;
