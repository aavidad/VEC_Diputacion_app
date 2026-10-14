\set ON_ERROR_STOP on
-- Provisión de fuente corporativa con el migrador existente; no concede perfiles.
-- Parámetros obligatorios: organizacion_operativa_ref, organizacion_canonica_ref.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
SET LOCAL timezone='UTC';
SET LOCAL search_path=pg_catalog,pg_temp;
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('organizacion_correspondencia:'||:'organizacion_operativa_ref',0));
WITH actual AS (
 SELECT h.* FROM vec_contexto_actor_v1.correspondencia_organizacion_actual_v1 a
 JOIN vec_contexto_actor_v1.correspondencia_organizacion_versiones_v1 h USING(organizacion_operativa_ref,version)
 WHERE a.organizacion_operativa_ref=:'organizacion_operativa_ref'
), fuente AS (
 SELECT h.* FROM vec_contexto_actor_v1.organizacion_actual a
 JOIN vec_contexto_actor_v1.organizacion_versiones h USING(organizacion_ref,version)
 WHERE a.organizacion_ref=:'organizacion_canonica_ref'
)
SELECT vec_contexto_actor_v1.publicar_correspondencia_organizacion_v1(
 pg_catalog.jsonb_build_object('esquema','vec.organizacion.correspondencia.v1',
  'correspondencia_ref',coalesce(x.correspondencia_ref,'correspondencia-org:'||pg_catalog.gen_random_uuid()::text),
  'version',coalesce(x.version,0)+1,'organizacion_operativa_ref',:'organizacion_operativa_ref',
  'organizacion_canonica_ref',f.organizacion_ref,'organizacion_version',f.version,
  'procedencia_ref',f.procedencia_ref,'procedencia_version',f.procedencia_version,
  'procedencia_huella_sha256',f.procedencia_huella_sha256,'procedencia_autoridad',f.procedencia_autoridad,
  'estado','vigente','vigente_desde',f.vigente_desde,'vigente_hasta',f.vigente_hasta)::text,
 coalesce(x.version,0),coalesce(x.huella_sha256,''))
FROM fuente f LEFT JOIN actual x ON true;
COMMIT;
