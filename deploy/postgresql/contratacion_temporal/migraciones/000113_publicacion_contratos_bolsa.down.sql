\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_contratacion_temporal:migracion:000113', 0)
);

-- CT113 añade la posición de publicación (transaccion_publicacion) a cada
-- fila nueva del outbox. Retirarla con filas ya posicionadas sí pierde datos:
-- al reinstalar, esas filas pasarían a posición 0, por detrás del cursor que
-- Bolsa guarda en su inbox, y los eventos aún no leídos no se publicarían
-- nunca. Por eso el DOWN solo se admite mientras ninguna fila tenga posición
-- (ninguna incorporación registrada desde que se instaló CT113).
DO $prevalidacion$
BEGIN
    IF current_user <> 'vec_contratacion_temporal_propietario'
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.leer_contratos_bolsa_v1(bigint,text,integer)'
       ) IS NULL
       -- CT115 reescribe esta lectura: se retira antes.
       OR pg_catalog.to_regclass('vec_contratacion_temporal.cese_nombramiento_v1') IS NOT NULL
       OR pg_catalog.to_regprocedure(
           'vec_contratacion_temporal.instante_contrato_bolsa_v1(timestamptz)'
       ) IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'CT113 no está instalada';
    END IF;
    -- Sin escritores concurrentes entre la comprobación y el DROP COLUMN.
    LOCK TABLE vec_contratacion_temporal.incorporacion_outbox_v2 IN SHARE ROW EXCLUSIVE MODE;
    IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.incorporacion_outbox_v2
                WHERE transaccion_publicacion IS NOT NULL) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'CT113 tiene publicaciones posicionadas: retirarla perdería eventos hacia Bolsa';
    END IF;
END
$prevalidacion$;

DROP FUNCTION vec_contratacion_temporal.leer_contratos_bolsa_v1(bigint, text, integer);
DROP FUNCTION vec_contratacion_temporal.posicion_contrato_bolsa_v1(xid8);
DROP INDEX vec_contratacion_temporal.incorporacion_outbox_v2_publicacion_bolsa;
ALTER TABLE vec_contratacion_temporal.incorporacion_outbox_v2 DROP COLUMN transaccion_publicacion;
DROP FUNCTION vec_contratacion_temporal.instante_contrato_bolsa_v1(timestamptz);
COMMIT;
