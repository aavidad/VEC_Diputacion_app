BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_autorizacion_atestada_v3:migracion:000002', 0
    )
);

DO $proteccion$
BEGIN
    IF EXISTS (
        SELECT 1
          FROM vec_autorizacion_atestada_v3.consumo_decision_v3
    )
       AND pg_catalog.current_setting(
           'vec.confirmar_destruccion_autorizacion_atestada_v3',
           true
       ) IS DISTINCT FROM
       'DESTRUIR_AUTORIZACION_ATESTADA_V3_IRREVERSIBLE' THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'retirada del consumidor VEC-AD-3 protegida';
    END IF;
END
$proteccion$;

REVOKE EXECUTE ON FUNCTION
    vec_autorizacion_atestada_v3.registrar_y_consumir_decision_v3_atestada(
        bytea, bytea, bytea, bytea, numeric, numeric,
        bytea, bytea, bytea, bytea
    ) FROM vec_contratacion_temporal_propietario;
REVOKE REFERENCES (
    decision_ref, efecto_ref, huella_efecto_sha256
) ON vec_autorizacion_atestada_v3.consumo_decision_v3
  FROM vec_contratacion_temporal_propietario;
REVOKE USAGE ON SCHEMA vec_autorizacion_atestada_v3
    FROM vec_contratacion_temporal_propietario;
DROP FUNCTION
    vec_autorizacion_atestada_v3.registrar_y_consumir_decision_v3_atestada(
        bytea, bytea, bytea, bytea, numeric, numeric,
        bytea, bytea, bytea, bytea
    );

DO $triggers_gobierno$
DECLARE
    v_tabla text;
BEGIN
    FOREACH v_tabla IN ARRAY ARRAY[
        'clave_capacidad_version', 'puntero_clave_emision',
        'revocacion_clave_capacidad', 'configuracion_confianza_version',
        'raiz_confianza_version', 'configuracion_raiz',
        'puntero_configuracion_actual', 'revocacion_configuracion',
        'revocacion_raiz'
    ] LOOP
        EXECUTE pg_catalog.format(
            'DROP TRIGGER checkpoint_despues ON vec_autorizacion_atestada_v3.%I',
            v_tabla
        );
    END LOOP;
END
$triggers_gobierno$;

DROP POLICY propietario_exacto
    ON vec_autorizacion_atestada_v3.checkpoint_gobierno;
ALTER TABLE vec_autorizacion_atestada_v3.checkpoint_gobierno
    NO FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_autorizacion_atestada_v3.checkpoint_gobierno
    DISABLE ROW LEVEL SECURITY;
DROP FUNCTION vec_autorizacion_atestada_v3.avanzar_checkpoint();
DROP FUNCTION vec_autorizacion_atestada_v3.bytea_igual_constante(
    bytea, bytea
);
DROP FUNCTION vec_autorizacion_atestada_v3.preimagen_mac(jsonb);
DROP FUNCTION vec_autorizacion_atestada_v3.encuadrar_mac(text);
DROP FUNCTION vec_autorizacion_atestada_v3.capacidad_canonica(jsonb);
DROP FUNCTION vec_autorizacion_atestada_v3.capacidad_cruda_prevalida(bytea);
DROP FUNCTION vec_autorizacion_atestada_v3.texto_json_go(text);

COMMIT;
