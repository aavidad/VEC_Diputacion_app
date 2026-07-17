-- Vínculo exacto entre la decisión nominal y su huella atestada.
BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path = pg_catalog;

DO $prevalidacion$
BEGIN
    IF to_regnamespace('vec_autorizacion_atestada_v2') IS NOT NULL
       OR NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_roles
            WHERE rolname = 'vec_autorizacion_atestada_v2_propietario'
              AND NOT rolcanlogin AND NOT rolsuper
       )
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_constraint
            WHERE conrelid =
                  'vec_autorizacion.decision_autorizacion_solicitud_ligada_v2'::regclass
              AND conname =
                  'decision_solicitud_ligada_v2_ref_huella_atestada_unica'
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para vinculo nominal atestado V2';
    END IF;
END
$prevalidacion$;

ALTER TABLE vec_autorizacion.decision_autorizacion_solicitud_ligada_v2
    ADD CONSTRAINT
        decision_solicitud_ligada_v2_ref_huella_atestada_unica
    UNIQUE (decision_ref, huella_decision_sha256);

GRANT REFERENCES (decision_ref, huella_decision_sha256) ON
    vec_autorizacion.decision_autorizacion_solicitud_ligada_v2
    TO vec_autorizacion_atestada_v2_propietario;

CREATE FUNCTION vec_autorizacion.obtener_instante_decision_atestada_v2(
    p_decision_ref text,
    p_huella_decision_sha256 text
)
RETURNS timestamptz
LANGUAGE sql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
    SELECT decision.registrada_en
      FROM vec_autorizacion.decision_autorizacion_solicitud_ligada_v2
           AS decision
     WHERE decision.decision_ref = p_decision_ref
       AND decision.huella_decision_sha256 = p_huella_decision_sha256
$funcion$;
REVOKE ALL ON FUNCTION
    vec_autorizacion.obtener_instante_decision_atestada_v2(text, text)
    FROM PUBLIC;
GRANT EXECUTE ON FUNCTION
    vec_autorizacion.obtener_instante_decision_atestada_v2(text, text)
    TO vec_autorizacion_atestada_v2_propietario;
COMMIT;
