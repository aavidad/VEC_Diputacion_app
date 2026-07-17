-- Destruccion deliberada. Sin datos, revierte normalmente; con historia exige
-- una confirmacion explicita de operacion irreversible.
BEGIN;
SET LOCAL ROLE vec_bolsa_convocatorias_propietario;
SET LOCAL search_path = pg_catalog;

DO $prevalidacion$
BEGIN
    IF to_regnamespace('vec_bolsa_convocatorias') IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: falta el esquema esperado';
    END IF;
    IF to_regprocedure(
           'vec_bolsa_convocatorias.obtener_version_exacta_v1(jsonb,jsonb,bytea,bytea)'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: la consulta exacta sigue instalada';
    END IF;
END
$prevalidacion$;

LOCK TABLE vec_bolsa_convocatorias.auditoria_actual IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_bolsa_convocatorias.atestacion_autorizacion_actual IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_bolsa_convocatorias.auditoria IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_bolsa_convocatorias.uso_decision_consulta IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_bolsa_convocatorias.atestacion_autorizacion_version IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_bolsa_convocatorias.instancia_flujo_version IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_bolsa_convocatorias.version_convocatoria IN ACCESS EXCLUSIVE MODE;

DO $confirmar_historia$
BEGIN
    IF (
        EXISTS (SELECT 1 FROM vec_bolsa_convocatorias.version_convocatoria)
        OR EXISTS (SELECT 1 FROM vec_bolsa_convocatorias.instancia_flujo_version)
        OR EXISTS (SELECT 1 FROM vec_bolsa_convocatorias.atestacion_autorizacion_version)
        OR EXISTS (SELECT 1 FROM vec_bolsa_convocatorias.atestacion_autorizacion_actual)
        OR EXISTS (SELECT 1 FROM vec_bolsa_convocatorias.uso_decision_consulta)
        OR EXISTS (SELECT 1 FROM vec_bolsa_convocatorias.auditoria)
    ) AND current_setting(
        'vec.confirmar_destruccion_bolsa_convocatorias', true
    ) IS DISTINCT FROM
        'DESTRUIR_HISTORIA_BOLSA_CONVOCATORIAS_IRREVERSIBLE' THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: existe historia durable',
            HINT = 'use el procedimiento formal de destruccion y su confirmacion explicita';
    END IF;
END
$confirmar_historia$;

DROP TABLE vec_bolsa_convocatorias.auditoria_actual;
DROP TABLE vec_bolsa_convocatorias.auditoria;
DROP TABLE vec_bolsa_convocatorias.uso_decision_consulta;
DROP TABLE vec_bolsa_convocatorias.atestacion_autorizacion_actual;
DROP TABLE vec_bolsa_convocatorias.atestacion_autorizacion_version;
DROP TABLE vec_bolsa_convocatorias.instancia_flujo_version;
DROP TABLE vec_bolsa_convocatorias.version_convocatoria;

DROP FUNCTION vec_bolsa_convocatorias.validar_avance_auditoria_actual();
DROP FUNCTION vec_bolsa_convocatorias.validar_avance_atestacion_actual();
DROP FUNCTION vec_bolsa_convocatorias.rechazar_mutacion_inmutable();
DROP FUNCTION vec_bolsa_convocatorias.instante_utc_microsegundo_valido(text);
DROP FUNCTION vec_bolsa_convocatorias.huella_sha256_valida(text);
DROP FUNCTION vec_bolsa_convocatorias.texto_opaco_valido(text, integer);

ALTER DEFAULT PRIVILEGES FOR ROLE vec_bolsa_convocatorias_propietario
    GRANT EXECUTE ON FUNCTIONS TO PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_bolsa_convocatorias_propietario
    GRANT USAGE ON TYPES TO PUBLIC;
DROP SCHEMA vec_bolsa_convocatorias;
COMMIT;
