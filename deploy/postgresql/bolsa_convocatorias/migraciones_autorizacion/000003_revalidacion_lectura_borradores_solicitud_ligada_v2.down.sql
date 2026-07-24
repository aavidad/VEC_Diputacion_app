BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path = pg_catalog;

REVOKE ALL ON FUNCTION
    vec_autorizacion.revalidar_decision_lectura_borrador_solicitud_ligada_v2(
        jsonb,bytea,bytea,text,text,text,text,jsonb
    ) FROM PUBLIC, vec_bolsa_convocatorias_propietario;
DROP FUNCTION
    vec_autorizacion.revalidar_decision_lectura_borrador_solicitud_ligada_v2(
        jsonb,bytea,bytea,text,text,text,text,jsonb
    );
DROP FUNCTION
    vec_autorizacion.frescura_prueba_lectura_borrador_valida_v1(
        timestamptz,timestamptz
    );
COMMIT;
