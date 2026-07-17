BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';

DO $prevalidacion$
BEGIN
    IF to_regnamespace('vec_autorizacion_atestada_v2') IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'retire primero el registro atestado V2';
    END IF;
END
$prevalidacion$;

REVOKE ALL ON FUNCTION
    vec_autorizacion.obtener_instante_decision_atestada_v2(text, text)
    FROM PUBLIC, vec_autorizacion_atestada_v2_propietario;
DROP FUNCTION
    vec_autorizacion.obtener_instante_decision_atestada_v2(text, text);
REVOKE REFERENCES (decision_ref, huella_decision_sha256) ON
    vec_autorizacion.decision_autorizacion_solicitud_ligada_v2
    FROM vec_autorizacion_atestada_v2_propietario;
ALTER TABLE vec_autorizacion.decision_autorizacion_solicitud_ligada_v2
    DROP CONSTRAINT
        decision_solicitud_ligada_v2_ref_huella_atestada_unica;
COMMIT;
