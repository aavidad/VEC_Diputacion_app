BEGIN;
SET LOCAL ROLE vec_bolsa_reglas_baremo_propietario;
DO $verificar$
BEGIN
    IF (SELECT count(*)
          FROM vec_bolsa_reglas_baremo.version_reglas_baremo
         WHERE contenido_ref = 'reglas:concurrencia') <> 2
       OR (SELECT revision
             FROM vec_bolsa_reglas_baremo.estado_actual
            WHERE contenido_ref = 'reglas:concurrencia') <> 2
       OR (SELECT count(*)
             FROM vec_bolsa_reglas_baremo.uso_decision) <> 2
       OR (SELECT count(*)
             FROM vec_bolsa_reglas_baremo.intencion_confirmada) <> 2
       OR (SELECT count(*)
             FROM vec_bolsa_reglas_baremo.outbox) <> 2
       OR (SELECT count(*)
             FROM vec_bolsa_reglas_baremo.auditoria) <> 2 THEN
        RAISE EXCEPTION 'la carrera CAS confirmo efectos parciales o dos ganadores';
    END IF;
END
$verificar$;
ROLLBACK;
