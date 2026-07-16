-- Debe fallar en COMMIT con SQLSTATE 23514. El trigger diferido de la unica
-- cabecera cuenta ambos conjuntos de hijos y no permite una historia parcial.
\set VERBOSITY verbose

BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

CREATE TEMP TABLE pg_temp.cabecera_incompleta_v3 ON COMMIT DROP AS
SELECT *
  FROM vec_bolsa_baremacion.manifiesto_probatorio_v3
 WHERE baremacion_merito_ref = 'baremacion:001'
   AND numero_version = 2;

ALTER TABLE vec_bolsa_baremacion.manifiesto_autorizacion_v3
    DISABLE TRIGGER manifiesto_autorizacion_v3_inmutable;
ALTER TABLE vec_bolsa_baremacion.manifiesto_evidencia_v3
    DISABLE TRIGGER manifiesto_evidencia_v3_inmutable;
ALTER TABLE vec_bolsa_baremacion.manifiesto_probatorio_v3
    DISABLE TRIGGER manifiesto_probatorio_v3_inmutable;

DELETE FROM vec_bolsa_baremacion.manifiesto_autorizacion_v3
 WHERE manifiesto_ref IN (SELECT referencia FROM pg_temp.cabecera_incompleta_v3);
DELETE FROM vec_bolsa_baremacion.manifiesto_evidencia_v3
 WHERE manifiesto_ref IN (SELECT referencia FROM pg_temp.cabecera_incompleta_v3);
DELETE FROM vec_bolsa_baremacion.manifiesto_probatorio_v3
 WHERE referencia IN (SELECT referencia FROM pg_temp.cabecera_incompleta_v3);

ALTER TABLE vec_bolsa_baremacion.manifiesto_autorizacion_v3
    ENABLE TRIGGER manifiesto_autorizacion_v3_inmutable;
ALTER TABLE vec_bolsa_baremacion.manifiesto_evidencia_v3
    ENABLE TRIGGER manifiesto_evidencia_v3_inmutable;
ALTER TABLE vec_bolsa_baremacion.manifiesto_probatorio_v3
    ENABLE TRIGGER manifiesto_probatorio_v3_inmutable;

INSERT INTO vec_bolsa_baremacion.manifiesto_probatorio_v3
SELECT * FROM pg_temp.cabecera_incompleta_v3;

COMMIT;
