-- Reversion conservadora del archivo probatorio V3. No borra historia ni
-- consumos de autorizacion: solo desmonta una instalacion V3 realmente vacia.
BEGIN;
SET LOCAL ROLE vec_bolsa_baremacion_propietario;
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_bolsa_baremacion:migracion_down:manifiesto_probatorio_v3', 0
    )
);

DO $confirmacion_reversion$
DECLARE
    confirmacion_constante constant text :=
        'REVERTIR_MIGRACION_BOLSA_BAREMACION_V3';
BEGIN
    IF current_setting(
        'vec.confirmar_reversion_bolsa_baremacion_v3', true
    ) IS DISTINCT FROM confirmacion_constante THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down manifiesto V3 rechazado: falta confirmacion explicita',
            DETAIL = 'la reversion esta denegada por defecto, incluso sin filas',
            HINT = 'configure el opt-in literal solo en la sesion aprobada de migracion';
    END IF;
END
$confirmacion_reversion$;

-- Los bloqueos se adquieren antes de la primera comprobacion y se conservan
-- hasta COMMIT. RESTRICT impide retirar objetos adoptados por otra migracion.
LOCK TABLE vec_bolsa_baremacion.uso_decision IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_bolsa_baremacion.manifiesto_probatorio_v3
    IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_bolsa_baremacion.manifiesto_autorizacion_v3
    IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_bolsa_baremacion.manifiesto_evidencia_v3
    IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_bolsa_baremacion.prevalidacion_archivo_probatorio_v3
    IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_bolsa_baremacion.resultado_prevalidacion_archivo_v3
    IN ACCESS EXCLUSIVE MODE;

DO $prevalidar_historia$
BEGIN
    IF EXISTS (
        SELECT 1 FROM vec_bolsa_baremacion.manifiesto_probatorio_v3
    ) OR EXISTS (
        SELECT 1 FROM vec_bolsa_baremacion.manifiesto_autorizacion_v3
    ) OR EXISTS (
        SELECT 1 FROM vec_bolsa_baremacion.manifiesto_evidencia_v3
    ) OR EXISTS (
        SELECT 1
          FROM vec_bolsa_baremacion.prevalidacion_archivo_probatorio_v3
    ) OR EXISTS (
        SELECT 1
          FROM vec_bolsa_baremacion.resultado_prevalidacion_archivo_v3
    ) OR EXISTS (
        SELECT 1
          FROM vec_bolsa_baremacion.uso_decision
         WHERE tipo_efecto = 'prevalidacion_archivo'
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down manifiesto V3 rechazado: existe historia durable';
    END IF;
END
$prevalidar_historia$;

REVOKE EXECUTE ON FUNCTION
    vec_bolsa_baremacion.reservar_cambio_con_archivo_probatorio_v3(
        jsonb, jsonb, bytea, bytea
    ) FROM vec_bolsa_baremacion_ejecutor;
REVOKE EXECUTE ON FUNCTION
    vec_bolsa_baremacion.obtener_archivo_probatorio_previo_cambio_v3(
        jsonb, jsonb, bytea, bytea
    ) FROM vec_bolsa_baremacion_ejecutor;
REVOKE EXECUTE ON FUNCTION
    vec_bolsa_baremacion.confirmar_cambio_con_archivo_probatorio_v3(
        jsonb, jsonb, bytea, bytea, bytea, jsonb, bytea, bytea,
        bytea, text
    ) FROM vec_bolsa_baremacion_ejecutor;
REVOKE EXECUTE ON FUNCTION
    vec_bolsa_baremacion.obtener_version_vigente_con_archivo_probatorio_v3(
        jsonb, jsonb, bytea, bytea
    ) FROM vec_bolsa_baremacion_ejecutor;
REVOKE EXECUTE ON FUNCTION
    vec_bolsa_baremacion.obtener_version_con_archivo_probatorio_v3(
        jsonb, jsonb, bytea, bytea
    ) FROM vec_bolsa_baremacion_ejecutor;
REVOKE EXECUTE ON FUNCTION
    vec_bolsa_baremacion.obtener_evidencia_transaccion_con_archivo_probatorio_v3(
        jsonb, jsonb, bytea, bytea
    ) FROM vec_bolsa_baremacion_ejecutor;

DROP FUNCTION
    vec_bolsa_baremacion.confirmar_cambio_con_archivo_probatorio_v3(
        jsonb, jsonb, bytea, bytea, bytea, jsonb, bytea, bytea,
        bytea, text
    ) RESTRICT;
DROP FUNCTION
    vec_bolsa_baremacion.obtener_evidencia_transaccion_con_archivo_probatorio_v3(
        jsonb, jsonb, bytea, bytea
    ) RESTRICT;
DROP FUNCTION
    vec_bolsa_baremacion.obtener_version_con_archivo_probatorio_v3(
        jsonb, jsonb, bytea, bytea
    ) RESTRICT;
DROP FUNCTION
    vec_bolsa_baremacion.obtener_version_vigente_con_archivo_probatorio_v3(
        jsonb, jsonb, bytea, bytea
    ) RESTRICT;
DROP FUNCTION
    vec_bolsa_baremacion.reservar_cambio_con_archivo_probatorio_v3(
        jsonb, jsonb, bytea, bytea
    ) RESTRICT;
DROP FUNCTION
    vec_bolsa_baremacion.obtener_archivo_probatorio_previo_cambio_v3(
        jsonb, jsonb, bytea, bytea
    ) RESTRICT;

DROP FUNCTION vec_bolsa_baremacion.registrar_manifiesto_probatorio_v3(
    jsonb, bytea, bytea, bytea, numeric, text, text, text,
    timestamp with time zone
) RESTRICT;

DROP TABLE vec_bolsa_baremacion.resultado_prevalidacion_archivo_v3 RESTRICT;
DROP TABLE vec_bolsa_baremacion.prevalidacion_archivo_probatorio_v3 RESTRICT;
DROP TABLE vec_bolsa_baremacion.manifiesto_evidencia_v3 RESTRICT;
DROP TABLE vec_bolsa_baremacion.manifiesto_autorizacion_v3 RESTRICT;
DROP TABLE vec_bolsa_baremacion.manifiesto_probatorio_v3 RESTRICT;

DROP FUNCTION
    vec_bolsa_baremacion.validar_cardinalidad_manifiesto_v3() RESTRICT;
DROP FUNCTION
    vec_bolsa_baremacion.validar_autorizaciones_manifiesto_v3() RESTRICT;
DROP FUNCTION
    vec_bolsa_baremacion.validar_evidencias_manifiesto_v3() RESTRICT;
DROP FUNCTION vec_bolsa_baremacion.huella_prevalidacion_archivo_v3(
    text, numeric, text, text, text, text, text, text, text
) RESTRICT;
DROP FUNCTION vec_bolsa_baremacion.construir_archivo_probatorio_v3(
    text, numeric
) RESTRICT;
DROP FUNCTION
    vec_bolsa_baremacion.archivo_unitario_manifiesto_v3_valido(
        jsonb, bytea, bytea, bytea
    ) RESTRICT;
DROP FUNCTION vec_bolsa_baremacion.contenido_manifiesto_probatorio_v3(
    jsonb
) RESTRICT;
DROP FUNCTION vec_bolsa_baremacion.instante_canonico_manifiesto_v3(
    text
) RESTRICT;
DROP FUNCTION vec_bolsa_baremacion.sello_hmac_manifiesto_v3_valido(
    text
) RESTRICT;
DROP FUNCTION vec_bolsa_baremacion.referencia_manifiesto_v3_valida(
    text, integer
) RESTRICT;
DROP FUNCTION vec_bolsa_baremacion.tipo_evidencia_manifiesto_v3_valido(
    text
) RESTRICT;
DROP FUNCTION vec_bolsa_baremacion.accion_recurso_manifiesto_v3_valida(
    text, text
) RESTRICT;
DROP FUNCTION vec_bolsa_baremacion.parte_canonica_manifiesto_v3(
    text
) RESTRICT;

ALTER TABLE vec_bolsa_baremacion.uso_decision
    DROP CONSTRAINT uso_perfil_cerrado;
ALTER TABLE vec_bolsa_baremacion.uso_decision
    ADD CONSTRAINT uso_perfil_cerrado CHECK (
        esquema_huella_decision =
            'vec.autorizacion.decision.reforzada.v1.autenticacion-actor'
        AND tipo_efecto IN (
            'reserva', 'confirmacion', 'abandono',
            'lectura_vigente', 'lectura_version', 'lectura_evidencia'
        )
        AND vec_bolsa_baremacion.huella_sha256_valida(
            huella_decision_sha256
        )
        AND vec_bolsa_baremacion.huella_sha256_valida(huella_efecto_sha256)
        AND vec_bolsa_baremacion.texto_opaco_valido(resultado_ref, 512)
    );

GRANT EXECUTE ON FUNCTION vec_bolsa_baremacion.reservar_cambio(
    jsonb, jsonb, bytea, bytea
) TO vec_bolsa_baremacion_ejecutor;
GRANT EXECUTE ON FUNCTION vec_bolsa_baremacion.confirmar_cambio(
    jsonb, jsonb, bytea, bytea, bytea
) TO vec_bolsa_baremacion_ejecutor;
GRANT EXECUTE ON FUNCTION vec_bolsa_baremacion.obtener_version_vigente(
    jsonb, jsonb, bytea, bytea
) TO vec_bolsa_baremacion_ejecutor;
GRANT EXECUTE ON FUNCTION vec_bolsa_baremacion.obtener_version(
    jsonb, jsonb, bytea, bytea
) TO vec_bolsa_baremacion_ejecutor;
GRANT EXECUTE ON FUNCTION
    vec_bolsa_baremacion.obtener_evidencia_transaccion(
        jsonb, jsonb, bytea, bytea
    ) TO vec_bolsa_baremacion_ejecutor;
COMMIT;
