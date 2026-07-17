-- Retirada destructiva, exacta y sin CASCADE. No es una operacion runtime.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';

DO $prevalidacion$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname = current_user AND rolsuper
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'retirada atestada V2 requiere superusuario';
    END IF;
    IF current_setting(
           'vec.confirmar_destruccion_autorizacion_atestada_v2', true
       ) IS DISTINCT FROM
       'DESTRUIR_AUTORIZACION_ATESTADA_V2_IRREVERSIBLE' THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'retirada atestada V2 no confirmada';
    END IF;
END
$prevalidacion$;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_autorizacion_atestada_v2:migracion:000001', 0
    )
);
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_autorizacion_atestada_v2:gobierno_clave:v1', 0
    )
);

LOCK TABLE
    vec_autorizacion_atestada_v2.auditoria_consumo_v2,
    vec_autorizacion_atestada_v2.control_cadena_auditoria,
    vec_autorizacion_atestada_v2.consumo_decision_v2,
    vec_autorizacion_atestada_v2.consumo_capacidad_v2,
    vec_autorizacion_atestada_v2.atestacion_decision_v2,
    vec_autorizacion_atestada_v2.checkpoint_gobierno_clave,
    vec_autorizacion_atestada_v2.puntero_clave_capacidad,
    vec_autorizacion_atestada_v2.revocacion_clave_capacidad,
    vec_autorizacion_atestada_v2.clave_capacidad_version
IN ACCESS EXCLUSIVE MODE;

DROP FUNCTION
    vec_autorizacion_atestada_v2.reconciliar_consumo_decision_v2(
        text, text, text, text, text
    );
DROP FUNCTION
    vec_autorizacion_atestada_v2.registrar_y_consumir_decision_v2_atestada(
        bytea, bytea, bytea, bytea, bytea, bytea, jsonb
    );
DROP FUNCTION
    vec_autorizacion_atestada_v2.obtener_material_emisor_capacidad();
DROP FUNCTION
    vec_autorizacion_atestada_v2.identidad_runtime_valida(text, boolean);

DROP TABLE vec_autorizacion_atestada_v2.auditoria_consumo_v2;
DROP TABLE vec_autorizacion_atestada_v2.control_cadena_auditoria;
DROP TABLE vec_autorizacion_atestada_v2.consumo_decision_v2;
DROP TABLE vec_autorizacion_atestada_v2.consumo_capacidad_v2;
DROP TABLE vec_autorizacion_atestada_v2.atestacion_decision_v2;
DROP TABLE vec_autorizacion_atestada_v2.checkpoint_gobierno_clave;
DROP TABLE vec_autorizacion_atestada_v2.puntero_clave_capacidad;
DROP TABLE vec_autorizacion_atestada_v2.revocacion_clave_capacidad;
DROP TABLE vec_autorizacion_atestada_v2.clave_capacidad_version;

DROP FUNCTION
    vec_autorizacion_atestada_v2.avanzar_checkpoint_gobierno_clave();
DROP FUNCTION
    vec_autorizacion_atestada_v2.sellar_conocimiento_gobierno_clave();
DROP FUNCTION
    vec_autorizacion_atestada_v2.validar_gobierno_clave();
DROP FUNCTION
    vec_autorizacion_atestada_v2.rechazar_mutacion_inmutable();
DROP FUNCTION
    vec_autorizacion_atestada_v2.bytea_igual_constante(bytea, bytea);
DROP FUNCTION
    vec_autorizacion_atestada_v2.preimagen_capacidad(jsonb);
DROP FUNCTION
    vec_autorizacion_atestada_v2.encuadrar_capacidad(text);
DROP FUNCTION
    vec_autorizacion_atestada_v2.instante_texto_valido(text);
DROP FUNCTION
    vec_autorizacion_atestada_v2.sujeto_hmac_valido(text);
DROP FUNCTION
    vec_autorizacion_atestada_v2.huella_sha256_valida(text);
DROP FUNCTION
    vec_autorizacion_atestada_v2.texto_tecnico_valido(text, integer);

ALTER DEFAULT PRIVILEGES
    FOR ROLE vec_autorizacion_atestada_v2_propietario
    GRANT EXECUTE ON FUNCTIONS TO PUBLIC;
ALTER DEFAULT PRIVILEGES
    FOR ROLE vec_autorizacion_atestada_v2_propietario
    GRANT USAGE ON TYPES TO PUBLIC;

DROP SCHEMA vec_autorizacion_atestada_v2;
COMMIT;
