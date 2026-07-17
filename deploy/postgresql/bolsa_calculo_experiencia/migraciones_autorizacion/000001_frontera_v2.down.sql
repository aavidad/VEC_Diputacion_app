BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path = pg_catalog;

DO $prevalidacion$
BEGIN
    IF to_regnamespace('vec_bolsa_calculo_experiencia') IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down de frontera V2 rechazado: el almacen sigue instalado';
    END IF;
    IF to_regprocedure(
        'vec_autorizacion.revalidar_decision_calculo_experiencia_v1(text,text,text,text,text,text,text,text)'
    ) IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down de frontera V2 rechazado: falta la funcion';
    END IF;
END
$prevalidacion$;

REVOKE EXECUTE ON FUNCTION
    vec_autorizacion.revalidar_decision_calculo_experiencia_v1(
        text, text, text, text, text, text, text, text
    ) FROM vec_bolsa_calculo_experiencia_aplicacion;
REVOKE REFERENCES (decision_ref) ON
    vec_autorizacion.decision_autorizacion_solicitud_ligada_v2
    FROM vec_bolsa_calculo_experiencia_propietario;
REVOKE USAGE ON SCHEMA vec_autorizacion FROM
    vec_bolsa_calculo_experiencia_propietario,
    vec_bolsa_calculo_experiencia_aplicacion;
DROP FUNCTION vec_autorizacion.revalidar_decision_calculo_experiencia_v1(
    text, text, text, text, text, text, text, text
);
COMMIT;
