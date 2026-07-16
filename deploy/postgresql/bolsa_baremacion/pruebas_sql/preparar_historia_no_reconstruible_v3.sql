-- Construye una version V2 heredada mínima y coherente con las restricciones
-- V1. 000005 debe rechazarla: no existe archivo probatorio byte-exacto que
-- pueda reconstruirse de manera honesta a partir de esas columnas.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

-- El fixture solo necesita demostrar una fila numero>1 preexistente. Se
-- transforma la V1 ya confirmada conservando sus FK reales de auditoria y
-- outbox. DISABLE TRIGGER ALL es deliberadamente exclusivo del arnes DBA;
-- nunca forma parte de una ruta operativa ni de la migracion.
ALTER TABLE vec_bolsa_baremacion.version_baremacion DISABLE TRIGGER ALL;

WITH origen AS (
    SELECT version.baremacion_merito_ref,
           jsonb_set(
               version.agregado, '{decisiones}', '[{}]'::jsonb, false
           ) AS agregado_v2
      FROM vec_bolsa_baremacion.version_baremacion AS version
     WHERE version.baremacion_merito_ref = 'baremacion:001'
       AND version.numero = 1
), canon AS (
    SELECT origen.*,
           convert_to(origen.agregado_v2::text, 'UTF8') AS agregado_v2_bytes
      FROM origen
)
UPDATE vec_bolsa_baremacion.version_baremacion AS version
   SET numero = 2,
       huella_estado_sha256 = encode(sha256(canon.agregado_v2_bytes), 'hex'),
       agregado_canonico = canon.agregado_v2_bytes,
       agregado = canon.agregado_v2,
       confirmada_en = version.confirmada_en + interval '1 second'
  FROM canon
 WHERE version.baremacion_merito_ref = canon.baremacion_merito_ref
   AND version.numero = 1;

UPDATE vec_bolsa_baremacion.baremacion_actual AS actual
   SET numero = 2,
       huella_estado_sha256 = version.huella_estado_sha256,
       actualizada_en = actual.actualizada_en + interval '1 second'
  FROM vec_bolsa_baremacion.version_baremacion AS version
 WHERE version.baremacion_merito_ref = actual.baremacion_merito_ref
   AND version.numero = 2;

ALTER TABLE vec_bolsa_baremacion.version_baremacion ENABLE TRIGGER ALL;

DO $verificar_historia_v2$
BEGIN
    IF (SELECT count(*)
          FROM vec_bolsa_baremacion.version_baremacion
         WHERE numero = 2) <> 1
       OR (SELECT count(*)
             FROM vec_bolsa_baremacion.baremacion_actual
            WHERE numero = 2) <> 1 THEN
        RAISE EXCEPTION 'no se preparo historia heredada V2';
    END IF;
END
$verificar_historia_v2$;

COMMIT;
