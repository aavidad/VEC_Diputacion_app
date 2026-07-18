BEGIN;
SET LOCAL ROLE vec_identidad_sesiones_v1_propietario;
SET LOCAL search_path = pg_catalog;

DO $prevalidacion$
BEGIN
    IF EXISTS (
        SELECT 1 FROM vec_identidad_sesiones_v1.consumo_asercion
    ) OR EXISTS (
        SELECT 1 FROM vec_identidad_sesiones_v1.cuenta
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down de operaciones rechazado: existe historia de identidad';
    END IF;
END
$prevalidacion$;

DROP FUNCTION vec_identidad_sesiones_v1.revocar_sesion_v1(
    text, text, text, text
);
DROP FUNCTION vec_identidad_sesiones_v1.cambiar_estado_cuenta_v1(
    text, text, text, text
);
DROP FUNCTION vec_identidad_sesiones_v1.revalidar_sesion_y_cuentas_v1(
    text, text, text, text, text, text, boolean, text, text, text,
    text, text, timestamptz, timestamptz, text, text, text, text,
    timestamptz, timestamptz
);
DROP FUNCTION vec_identidad_sesiones_v1.reconciliar_registro_sesion_v1(
    text, text, text, text, bigint, bytea, bytea, bytea, bytea,
    bytea, boolean, text, text, text, text, timestamptz,
    timestamptz, timestamptz, text, text
);
DROP FUNCTION vec_identidad_sesiones_v1.registrar_sesion_v1(
    text, text, text, text, bigint, bytea, bytea, bytea, bytea,
    bytea, boolean, text, text, text, text, timestamptz,
    timestamptz, timestamptz, text, text
);
DROP FUNCTION vec_identidad_sesiones_v1.provisionar_cuenta_v1(
    text, text, text, text, bigint, bytea, bytea, boolean, bytea
);
DROP FUNCTION vec_identidad_sesiones_v1.registrar_alias_hmac_cuenta_v1(
    text, text, text, text, text, bigint, bytea, bytea
);
DROP FUNCTION vec_identidad_sesiones_v1.huella_control_sesion_v1(
    text, numeric, text, text, timestamptz, timestamptz, text
);
DROP FUNCTION vec_identidad_sesiones_v1.encuadrar(text);
DROP FUNCTION vec_identidad_sesiones_v1.nueva_referencia(text);
COMMIT;
