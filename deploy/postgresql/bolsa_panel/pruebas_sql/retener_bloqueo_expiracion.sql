BEGIN;
SET LOCAL ROLE vec_bolsa_panel_propietario;

SELECT actual.decision_ref
  FROM vec_bolsa_panel.atestacion_autorizacion_actual AS actual
  JOIN vec_bolsa_panel.atestacion_autorizacion_version AS version_atestacion
    ON version_atestacion.decision_ref = actual.decision_ref
   AND version_atestacion.atestacion_ref = actual.atestacion_ref
   AND version_atestacion.version = actual.version
 WHERE actual.decision_ref = 'decision:panel:concurrencia:1'
 FOR UPDATE OF actual, version_atestacion;

-- El advisory lock permite al lanzador saber que los dos row locks ya existen.
SELECT pg_advisory_xact_lock(726163849201);
SELECT pg_sleep(12);
COMMIT;
