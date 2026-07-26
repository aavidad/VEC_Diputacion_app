-- T17 B1: aislamiento RLS separado del modelo durable.
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

CREATE FUNCTION vec_bolsa_importacion_convoca.lote_integro(
    p_importacion_ref text
)
RETURNS boolean LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path = pg_catalog
SET row_security = on AS $funcion$
DECLARE
    actual vec_bolsa_importacion_convoca.lote%ROWTYPE;
    filas integer;
    huella text;
    huella_semantica text;
BEGIN
    SELECT * INTO actual FROM vec_bolsa_importacion_convoca.lote
     WHERE importacion_ref = p_importacion_ref;
    IF NOT FOUND
       OR vec_bolsa_importacion_convoca.acta_valida(actual.acta_canonica)
          IS NOT TRUE
       OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
           actual.acta_canonica::text, 'UTF8'
       )), 'hex') <> actual.huella_acta_sha256
       OR vec_bolsa_importacion_convoca.historia_integra(
           p_importacion_ref
       ) IS NOT TRUE THEN
        RETURN false;
    END IF;
    SELECT pg_catalog.count(*),
           pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
               COALESCE(pg_catalog.string_agg(
                   pg_catalog.concat_ws(pg_catalog.chr(30), numero::text,
                       esquema_proteccion, clave_ref, clave_derivacion_ref,
                       clave_atestacion_ref,
                       pg_catalog.encode(nonce, 'hex'),
                       pg_catalog.encode(contenido_cifrado, 'hex'),
                       huella_contenido_cifrado_sha256,
                       pg_catalog.encode(
                           derivacion_documento_hmac_sha256, 'hex'
                       ),
                       pg_catalog.encode(
                           atestacion_fila_hmac_sha256, 'hex'
                       )
                   ),
                   ',' ORDER BY numero
               ), ''), 'UTF8'
           )), 'hex'),
           pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
               COALESCE(pg_catalog.string_agg(
                   numero::text || ':' || clave_atestacion_ref || ':' ||
                   pg_catalog.encode(atestacion_fila_hmac_sha256, 'hex'),
                   ',' ORDER BY numero
               ), ''), 'UTF8'
           )), 'hex')
      INTO filas, huella, huella_semantica
      FROM vec_bolsa_importacion_convoca.fila_staging
     WHERE importacion_ref = p_importacion_ref;
    IF actual.estado_staging = 'disponible' THEN
        RETURN (
               filas = (actual.acta_canonica->>'filas_aceptadas')::integer
           AND huella = actual.huella_staging_sha256
           AND huella_semantica =
               actual.huella_staging_semantica_sha256
        ) IS TRUE;
    END IF;
    RETURN (actual.estado_staging = 'expurgado' AND filas = 0) IS TRUE;
END
$funcion$;

ALTER TABLE vec_bolsa_importacion_convoca.lote ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_importacion_convoca.lote FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_importacion_convoca.fila_staging ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_importacion_convoca.fila_staging FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_importacion_convoca.conciliacion ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_importacion_convoca.conciliacion FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_importacion_convoca.decision_retencion ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_importacion_convoca.decision_retencion FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_importacion_convoca.ejecucion_retencion ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_importacion_convoca.ejecucion_retencion FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_importacion_convoca.historia_estado ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_importacion_convoca.historia_estado FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_importacion_convoca.outbox ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_importacion_convoca.outbox FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_importacion_convoca.politica_retencion
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_importacion_convoca.politica_retencion
    FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_importacion_convoca.politica_retencion_actual
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_importacion_convoca.politica_retencion_actual
    FORCE ROW LEVEL SECURITY;

CREATE POLICY propietario_lote ON vec_bolsa_importacion_convoca.lote
FOR ALL TO vec_bolsa_importacion_convoca_propietario USING (true) WITH CHECK (true);
CREATE POLICY propietario_staging ON vec_bolsa_importacion_convoca.fila_staging
FOR ALL TO vec_bolsa_importacion_convoca_propietario USING (true) WITH CHECK (true);
CREATE POLICY propietario_conciliacion ON vec_bolsa_importacion_convoca.conciliacion
FOR ALL TO vec_bolsa_importacion_convoca_propietario USING (true) WITH CHECK (true);
CREATE POLICY propietario_decision ON vec_bolsa_importacion_convoca.decision_retencion
FOR ALL TO vec_bolsa_importacion_convoca_propietario USING (true) WITH CHECK (true);
CREATE POLICY propietario_ejecucion ON vec_bolsa_importacion_convoca.ejecucion_retencion
FOR ALL TO vec_bolsa_importacion_convoca_propietario USING (true) WITH CHECK (true);
CREATE POLICY propietario_historia ON vec_bolsa_importacion_convoca.historia_estado
FOR ALL TO vec_bolsa_importacion_convoca_propietario USING (true) WITH CHECK (true);
CREATE POLICY propietario_outbox ON vec_bolsa_importacion_convoca.outbox
FOR ALL TO vec_bolsa_importacion_convoca_propietario USING (true) WITH CHECK (true);
CREATE POLICY propietario_politica
ON vec_bolsa_importacion_convoca.politica_retencion
FOR ALL TO vec_bolsa_importacion_convoca_propietario USING (true) WITH CHECK (true);
CREATE POLICY propietario_puntero_politica
ON vec_bolsa_importacion_convoca.politica_retencion_actual
FOR ALL TO vec_bolsa_importacion_convoca_propietario USING (true) WITH CHECK (true);

REVOKE ALL ON ALL TABLES IN SCHEMA vec_bolsa_importacion_convoca FROM PUBLIC;
REVOKE ALL ON ALL FUNCTIONS IN SCHEMA vec_bolsa_importacion_convoca FROM PUBLIC;
COMMIT;
