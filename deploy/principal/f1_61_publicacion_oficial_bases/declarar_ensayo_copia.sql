\set ON_ERROR_STOP on
-- F1-61, dato de la copia local: declara la publicación oficial de las bases
-- de la convocatoria de ensayo de Administrativo/a (versión 2 del respaldo
-- HZ37). Es un dato ficticio con referencia de ensayo, no una publicación real
-- ni un acto de RRHH. Solo para copias locales; nunca en cidonia ni producción.
-- Repetirlo devuelve el mismo recibo. Si la versión o su huella no coinciden,
-- la función se niega y no escribe.
BEGIN;
SET LOCAL ROLE vec_bolsa_convocatorias_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SELECT vec_bolsa_convocatorias.declarar_publicacion_oficial_bases_v1(jsonb_build_object(
 'convocatoria_id','proceso:bolsa:ensayo-auxiliar-2026','secuencia',2,
 'huella_version_sha256','f25ddcbf67c25f97beb2efd7a9d012a0a42940717749946af724e9234ae66ce3',
 'medio','bop_granada','boletin_numero',195,'anuncio_ref','ENSAYO-BOP-GRA-2026-000195',
 'fecha_publicacion','2026-10-09','direccion',NULL,'sustituye_ref',NULL,
 'declarada_por','actor:copia-local:datos-ficticios',
 'motivo_ref','motivos_bolsa:1:publicacion_bases_ensayo',
 'clave_idempotencia','copia-f1-61-auxiliar-v2'));
COMMIT;
