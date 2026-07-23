BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_autorizacion_atestada_v3:migracion:000001', 0
    )
);

DO $proteccion$
DECLARE
    v_historia bigint;
BEGIN
    SELECT
        (SELECT count(*) FROM
            vec_autorizacion_atestada_v3.clave_capacidad_version)
      + (SELECT count(*) FROM
            vec_autorizacion_atestada_v3.puntero_clave_emision)
      + (SELECT count(*) FROM
            vec_autorizacion_atestada_v3.revocacion_clave_capacidad)
      + (SELECT count(*) FROM
            vec_autorizacion_atestada_v3.configuracion_confianza_version)
      + (SELECT count(*) FROM
            vec_autorizacion_atestada_v3.raiz_confianza_version)
      + (SELECT count(*) FROM
            vec_autorizacion_atestada_v3.configuracion_raiz)
      + (SELECT count(*) FROM
            vec_autorizacion_atestada_v3.puntero_configuracion_actual)
      + (SELECT count(*) FROM
            vec_autorizacion_atestada_v3.revocacion_configuracion)
      + (SELECT count(*) FROM
            vec_autorizacion_atestada_v3.revocacion_raiz)
      + (SELECT count(*) FROM
            vec_autorizacion_atestada_v3.checkpoint_gobierno)
      + (SELECT count(*) FROM
            vec_autorizacion_atestada_v3.atestacion_decision_v3)
      + (SELECT count(*) FROM
            vec_autorizacion_atestada_v3.consumo_decision_v3)
      + (SELECT count(*) FROM
            vec_autorizacion_atestada_v3.auditoria_consumo_v3)
      INTO v_historia;
    IF v_historia > 0
       AND pg_catalog.current_setting(
           'vec.confirmar_destruccion_autorizacion_atestada_v3',
           true
       ) IS DISTINCT FROM
       'DESTRUIR_AUTORIZACION_ATESTADA_V3_IRREVERSIBLE' THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'retirada VEC-AD-3 protegida';
    END IF;
END
$proteccion$;

DROP TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3;
DROP TABLE vec_autorizacion_atestada_v3.control_cadena_auditoria;
DROP TABLE vec_autorizacion_atestada_v3.consumo_decision_v3;
DROP TABLE vec_autorizacion_atestada_v3.atestacion_decision_v3;
DROP TABLE vec_autorizacion_atestada_v3.checkpoint_gobierno;
DROP TABLE vec_autorizacion_atestada_v3.revocacion_raiz;
DROP TABLE vec_autorizacion_atestada_v3.revocacion_configuracion;
DROP TABLE vec_autorizacion_atestada_v3.puntero_configuracion_actual;
DROP TABLE vec_autorizacion_atestada_v3.configuracion_raiz;
DROP TABLE vec_autorizacion_atestada_v3.raiz_confianza_version;
DROP TABLE vec_autorizacion_atestada_v3.configuracion_confianza_version;
DROP TABLE vec_autorizacion_atestada_v3.revocacion_clave_capacidad;
DROP TABLE vec_autorizacion_atestada_v3.puntero_clave_emision;
DROP TABLE vec_autorizacion_atestada_v3.clave_capacidad_version;
DROP FUNCTION vec_autorizacion_atestada_v3.rechazar_truncado();
DROP FUNCTION vec_autorizacion_atestada_v3.rechazar_mutacion();
DROP FUNCTION vec_autorizacion_atestada_v3.capacidad_tipos_validos(jsonb);
DROP FUNCTION vec_autorizacion_atestada_v3.huella_sha256_valida(text);
DROP FUNCTION vec_autorizacion_atestada_v3.texto_tecnico_valido(
    text, integer
);
DROP SCHEMA vec_autorizacion_atestada_v3;

-- Restablece exactamente los valores predeterminados de PostgreSQL. La
-- migración up retiró EXECUTE/USAGE globales para que ningún objeto futuro
-- naciera abierto; si no se revierte este catálogo, PostgreSQL conserva una
-- dependencia pg_default_acl e impide retirar el rol propietario.
ALTER DEFAULT PRIVILEGES
    FOR ROLE vec_autorizacion_atestada_v3_propietario
    GRANT EXECUTE ON FUNCTIONS TO PUBLIC;
ALTER DEFAULT PRIVILEGES
    FOR ROLE vec_autorizacion_atestada_v3_propietario
    GRANT USAGE ON TYPES TO PUBLIC;

COMMIT;
