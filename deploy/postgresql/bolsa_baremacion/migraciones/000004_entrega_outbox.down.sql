-- No destruye historia de entrega. Solo revierte una instalacion vacia.
BEGIN;
SET LOCAL ROLE vec_bolsa_baremacion_propietario;
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_bolsa_baremacion:migracion_down:v2',
        0
    )
);

DO $confirmacion_reversion$
DECLARE
    confirmacion_constante constant text :=
        'REVERTIR_MIGRACION_BOLSA_BAREMACION_V1';
BEGIN
    IF current_setting(
        'vec.confirmar_reversion_bolsa_baremacion',
        true
    ) IS DISTINCT FROM confirmacion_constante THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down outbox rechazado: falta confirmacion explicita',
            DETAIL = 'la reversion esta denegada por defecto, incluso sin filas',
            HINT = 'configure el opt-in literal solo en la sesion aprobada de migracion';
    END IF;
END
$confirmacion_reversion$;

-- No basta con comprobar las tablas incorporadas por 000004: retirar la
-- entrega mientras queda historia de baremacion tambien dejaria una
-- instalacion parcialmente desmontada. Se inspecciona todo el esquema bajo
-- bloqueos que se conservan hasta el final de la transaccion.
DO $prevalidar_historia$
DECLARE
    relacion record;
    contiene_filas boolean;
BEGIN
    IF pg_catalog.to_regnamespace('vec_bolsa_baremacion') IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down outbox rechazado: falta el esquema Bolsa';
    END IF;
    FOR relacion IN
        SELECT espacio.nspname, clase.relname
          FROM pg_catalog.pg_class AS clase
          JOIN pg_catalog.pg_namespace AS espacio
            ON espacio.oid = clase.relnamespace
         WHERE espacio.nspname = 'vec_bolsa_baremacion'
           AND clase.relkind IN ('r', 'p')
         ORDER BY clase.oid
    LOOP
        EXECUTE format(
            'LOCK TABLE %I.%I IN ACCESS EXCLUSIVE MODE',
            relacion.nspname,
            relacion.relname
        );
        EXECUTE format(
            'SELECT EXISTS (SELECT 1 FROM %I.%I LIMIT 1)',
            relacion.nspname,
            relacion.relname
        ) INTO contiene_filas;
        IF contiene_filas THEN
            RAISE EXCEPTION USING ERRCODE = '55000',
                MESSAGE = 'down outbox rechazado: existe historia durable',
                DETAIL = relacion.nspname || '.' || relacion.relname;
        END IF;
    END LOOP;
END
$prevalidar_historia$;

REVOKE EXECUTE ON FUNCTION vec_bolsa_baremacion.reclamar_evento_outbox(
    text, bytea, integer
) FROM vec_bolsa_baremacion_lector_outbox;
REVOKE EXECUTE ON FUNCTION vec_bolsa_baremacion.finalizar_entrega_outbox(
    text, text, bytea, text, text
) FROM vec_bolsa_baremacion_lector_outbox;
REVOKE USAGE ON SCHEMA vec_bolsa_baremacion
    FROM vec_bolsa_baremacion_lector_outbox;

DROP FUNCTION vec_bolsa_baremacion.finalizar_entrega_outbox(
    text, text, bytea, text, text
) RESTRICT;
DROP FUNCTION vec_bolsa_baremacion.reclamar_evento_outbox(
    text, bytea, integer
) RESTRICT;
DROP FUNCTION vec_bolsa_baremacion.autenticar_consumidor_outbox(text) RESTRICT;

DROP TABLE vec_bolsa_baremacion.cursor_outbox_actual RESTRICT;
DROP TABLE vec_bolsa_baremacion.cursor_outbox_version RESTRICT;
DROP TABLE vec_bolsa_baremacion.entrega_outbox_actual RESTRICT;
DROP TABLE vec_bolsa_baremacion.entrega_outbox_version RESTRICT;
DROP TABLE vec_bolsa_baremacion.consumidor_outbox_actual RESTRICT;
DROP TABLE vec_bolsa_baremacion.consumidor_outbox_version RESTRICT;

DROP FUNCTION vec_bolsa_baremacion.validar_avance_cursor_outbox() RESTRICT;
DROP FUNCTION vec_bolsa_baremacion.validar_avance_entrega_outbox() RESTRICT;
DROP FUNCTION vec_bolsa_baremacion.validar_avance_consumidor_outbox() RESTRICT;
ALTER TABLE vec_bolsa_baremacion.evento_outbox
    DROP CONSTRAINT evento_outbox_referencia_secuencia_unica RESTRICT;
COMMIT;
