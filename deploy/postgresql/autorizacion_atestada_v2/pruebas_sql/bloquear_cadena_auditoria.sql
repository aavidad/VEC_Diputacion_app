BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v2_propietario;
SET LOCAL search_path = pg_catalog;
SELECT ultima_secuencia
  FROM vec_autorizacion_atestada_v2.control_cadena_auditoria
 WHERE control_id = true
 FOR UPDATE;
SELECT pg_sleep(4);
ROLLBACK;
