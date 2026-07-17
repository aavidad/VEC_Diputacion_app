-- Reversion conservadora: nunca destruye historia por accidente.
BEGIN;
SET LOCAL ROLE vec_bolsa_reglas_baremo_propietario;
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_bolsa_reglas_baremo:migracion:000001', 0
    )
);

DO $prevalidacion$
BEGIN
    IF to_regnamespace('vec_bolsa_reglas_baremo') IS NULL
       OR to_regclass(
           'vec_bolsa_reglas_baremo.version_reglas_baremo'
       ) IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'almacen de reglas de baremo no instalado';
    END IF;
    IF to_regprocedure(
           'vec_bolsa_reglas_baremo.confirmar_cambio_v1(jsonb,jsonb,bytea,bytea,bytea)'
       ) IS NOT NULL
       OR to_regprocedure(
           'vec_bolsa_reglas_baremo.obtener_version_exacta_v1(jsonb,jsonb,bytea,bytea)'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'retire antes las operaciones cerradas';
    END IF;
END
$prevalidacion$;

LOCK TABLE vec_bolsa_reglas_baremo.intencion_confirmada
    IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_bolsa_reglas_baremo.outbox IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_bolsa_reglas_baremo.auditoria IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_bolsa_reglas_baremo.auditoria_actual
    IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_bolsa_reglas_baremo.uso_prueba_transicion
    IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_bolsa_reglas_baremo.uso_decision IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_bolsa_reglas_baremo.estado_actual IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_bolsa_reglas_baremo.version_reglas_baremo
    IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_bolsa_reglas_baremo.contenido_reglas_baremo
    IN ACCESS EXCLUSIVE MODE;

DO $barrera_historia$
DECLARE
    existe_historia boolean;
BEGIN
    SELECT EXISTS (
        SELECT 1 FROM vec_bolsa_reglas_baremo.contenido_reglas_baremo
        UNION ALL
        SELECT 1 FROM vec_bolsa_reglas_baremo.version_reglas_baremo
        UNION ALL
        SELECT 1 FROM vec_bolsa_reglas_baremo.uso_decision
        UNION ALL
        SELECT 1 FROM vec_bolsa_reglas_baremo.uso_prueba_transicion
        UNION ALL
        SELECT 1 FROM vec_bolsa_reglas_baremo.auditoria
        UNION ALL
        SELECT 1 FROM vec_bolsa_reglas_baremo.outbox
        UNION ALL
        SELECT 1 FROM vec_bolsa_reglas_baremo.intencion_confirmada
    ) INTO existe_historia;
    IF existe_historia AND current_setting(
           'vec.confirmar_destruccion_bolsa_reglas_baremo', true
       ) IS DISTINCT FROM
          'DESTRUIR_HISTORIA_BOLSA_REGLAS_BAREMO_IRREVERSIBLE' THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: existe historia de reglas de baremo',
            HINT = 'establezca la confirmacion irreversible solo en una operacion formal de destruccion';
    END IF;
END
$barrera_historia$;

DROP TABLE vec_bolsa_reglas_baremo.intencion_confirmada;
DROP TABLE vec_bolsa_reglas_baremo.outbox;
DROP TABLE vec_bolsa_reglas_baremo.auditoria;
DROP TABLE vec_bolsa_reglas_baremo.auditoria_actual;
DROP TABLE vec_bolsa_reglas_baremo.uso_prueba_transicion;
DROP TABLE vec_bolsa_reglas_baremo.uso_decision;
DROP TABLE vec_bolsa_reglas_baremo.estado_actual;
DROP INDEX vec_bolsa_reglas_baremo.version_reglas_baremo_estado;
DROP TABLE vec_bolsa_reglas_baremo.version_reglas_baremo;
DROP TABLE vec_bolsa_reglas_baremo.contenido_reglas_baremo;
DROP TABLE vec_bolsa_reglas_baremo.configuracion_tenant;
DROP FUNCTION vec_bolsa_reglas_baremo.rechazar_borrado_o_truncado();
DROP FUNCTION vec_bolsa_reglas_baremo.rechazar_mutacion_inmutable();
DROP FUNCTION vec_bolsa_reglas_baremo.version_valida(numeric);
DROP FUNCTION vec_bolsa_reglas_baremo.huella_sha256_valida(text);
DROP FUNCTION vec_bolsa_reglas_baremo.referencia_valida(text);
DROP SCHEMA vec_bolsa_reglas_baremo;
-- Restablece los valores nativos para eliminar las ACL por defecto cuyo
-- propietario impediria retirar despues el rol NOLOGIN.
ALTER DEFAULT PRIVILEGES
    FOR ROLE vec_bolsa_reglas_baremo_propietario
    GRANT EXECUTE ON FUNCTIONS TO PUBLIC;
ALTER DEFAULT PRIVILEGES
    FOR ROLE vec_bolsa_reglas_baremo_propietario
    GRANT USAGE ON TYPES TO PUBLIC;
COMMIT;
