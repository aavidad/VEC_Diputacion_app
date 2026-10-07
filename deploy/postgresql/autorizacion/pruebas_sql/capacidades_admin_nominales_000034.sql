\set ON_ERROR_STOP on
-- Puerta estructural y negativos sin identidades provisionadas.
-- No acredita el favorable V3 ni caducidad/huella de una asignación real.
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL search_path=pg_catalog;
SET LOCAL ROLE vec_autorizacion_propietario;
DO $catalogo$
DECLARE v record;c record;meta record;
BEGIN
 SELECT * INTO STRICT v FROM vec_autorizacion.version_rol WHERE version_rol_ref='rol:administracion_perfiles:v4';
 SELECT * INTO STRICT c FROM vec_autorizacion.catalogo_accion_nominal_v1
 WHERE accion_ref='accion:administracion.perfiles.consultar' AND version=1;
 SELECT * INTO STRICT meta FROM vec_autorizacion.perfil_fijo_categoria_nominal_v1 WHERE version_rol_ref=v.version_rol_ref;
 IF c.version_rol_ref<>v.version_rol_ref OR c.fuente_ref<>v.version_rol_ref OR c.fuente_version<>4
 OR c.fuente_huella_sha256<>v.huella_sha256 OR c.clase_control<>'administrador_aplicacion'
 OR c.dimensiones_ambito<>'["organizacion_ref"]'::jsonb OR c.vigente_desde<>v.publicada_en OR c.vigente_hasta IS NOT NULL
 OR meta.categoria_administrativa<>'aplicacion' OR meta.version_rol_huella_sha256<>v.huella_sha256
 OR v.huella_sha256<>pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(vec_autorizacion.canon_version_rol_admin_v1(v.documento),'UTF8')),'hex')
 OR NOT EXISTS(SELECT 1 FROM pg_catalog.jsonb_array_elements(v.documento->'concesiones') e WHERE e=c.concesion)
 THEN RAISE EXCEPTION 'AUT34: catálogo o perfil divergente'; END IF;
 BEGIN
  UPDATE vec_autorizacion.catalogo_accion_nominal_v1 SET fuente_huella_sha256=repeat('0',64)
  WHERE accion_ref=c.accion_ref AND version=c.version;
  RAISE EXCEPTION 'AUT34: catálogo reescrito';
 EXCEPTION WHEN insufficient_privilege OR check_violation OR object_not_in_prerequisite_state THEN NULL; END;
 IF vec_autorizacion.acreditar_perfil_aplicacion_nominal_v1(
  'rol:administracion_perfiles:v3','ausente','ausente','ausente','administracion.perfiles.consultar',
  'administracion','perfil','gestion_perfiles',c.concesion->'campos_permitidos','{}'::jsonb) IS NOT FALSE
 OR vec_autorizacion.acreditar_perfil_aplicacion_nominal_v1(
  'rol:sistemas:v1','ausente','ausente','ausente','administracion.perfiles.consultar',
  'administracion','perfil','gestion_perfiles',c.concesion->'campos_permitidos','{}'::jsonb) IS NOT FALSE
 THEN RAISE EXCEPTION 'AUT34: versión anterior o Sistemas obtiene categoría Aplicación'; END IF;
END $catalogo$;
RESET ROLE;
DO $rol$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname='vec_admin_perfiles_lector'
 AND NOT(rolcanlogin OR rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls))
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members WHERE member='vec_admin_perfiles_lector'::regrole)
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_namespace n WHERE n.nspowner='vec_admin_perfiles_lector'::regrole)
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_class t WHERE t.relowner='vec_admin_perfiles_lector'::regrole)
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_class t JOIN pg_catalog.pg_namespace n ON n.oid=t.relnamespace
 WHERE n.nspname LIKE 'vec_%' AND t.relkind IN ('r','p','v','m','S','f')
 AND pg_catalog.has_table_privilege('vec_admin_perfiles_lector',t.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER'))
 THEN RAISE EXCEPTION 'AUT34: lector con autoridad de tablas o rol'; END IF;
END $rol$;
ROLLBACK;
