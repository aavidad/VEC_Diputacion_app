\set ON_ERROR_STOP on
-- Tipos de notificación a RRHH SINTÉTICOS para desarrollo y demostración.
-- La aplicación actual pide un «tipo» sin que conste su catálogo: estos
-- valores son provisionales y están SIN CONFIRMAR hasta que RRHH responda la
-- duda 48. Cada fila lleva sintetico=true y la fuente
-- «fuente:sintetica:a-confirmar-duda-48». Cuando RRHH publique el catálogo
-- real se añadirá una versión nueva que cierre ésta; nunca se editan filas.
-- Requiere cronos_v1 000010. Se puede reejecutar sin duplicar.
BEGIN;
SET LOCAL ROLE vec_cronos_v1_propietario;
SET LOCAL search_path=pg_catalog;
INSERT INTO vec_cronos_v1.notificacion_tipo
 (version_ref,tipo_ref,nombre,orden,vigente_desde,vigente_hasta,sintetico,fuente_ref,publicada_en)
SELECT 'notificacion:cronos:tipo:'||v.clave||':sintetico-1','notificacion:cronos:tipo:'||v.clave,v.nombre,v.orden,
  '2026-01-01 00:00:00'::timestamp AT TIME ZONE 'Europe/Madrid',NULL,true,'fuente:sintetica:a-confirmar-duda-48',clock_timestamp()
FROM (VALUES
 ('incidencia-marcaje','Incidencia en el marcaje',10),
 ('ausencia-imprevista','Ausencia o retraso imprevisto',20),
 ('otra-comunicacion','Otra comunicación a RRHH',30)
) v(clave,nombre,orden)
WHERE NOT EXISTS (SELECT 1 FROM vec_cronos_v1.notificacion_tipo t WHERE t.version_ref='notificacion:cronos:tipo:'||v.clave||':sintetico-1');
COMMIT;
