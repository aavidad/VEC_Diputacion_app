BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path = pg_catalog;

REVOKE EXECUTE ON FUNCTION
    vec_autorizacion.revalidar_decision_ejecucion_documental_v4(jsonb, bytea)
    FROM vec_ejecucion_documental_v4_propietario;
REVOKE USAGE ON SCHEMA vec_autorizacion
    FROM vec_ejecucion_documental_v4_propietario;
DROP FUNCTION vec_autorizacion.revalidar_decision_ejecucion_documental_v4(jsonb, bytea);
DROP FUNCTION vec_autorizacion.decision_canonica_documental_v4_estructura_valida(jsonb);
DROP FUNCTION vec_autorizacion.huella_lista_documental_v4(text, jsonb);
COMMIT;
