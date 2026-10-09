\set ON_ERROR_STOP on
-- Sobre clon v8(I primero) o v9(B primero), después de instalar AUT67.
BEGIN;
SET LOCAL search_path = pg_catalog;
DO $prueba$
DECLARE
  origen record;
  concesiones jsonb;
  grupo oid;
  fachada oid;
  catalogo_esperado integer;
  asignacion_esperada integer;
  destino_ref text;
BEGIN
  SELECT * INTO STRICT origen FROM vec_autorizacion.version_rol
   WHERE rol_id = 'administracion_perfiles' ORDER BY version DESC LIMIT 1;
  IF origen.version = 8 THEN
    catalogo_esperado := 22; asignacion_esperada := 5;
    destino_ref := 'rol:administracion_perfiles:v9';
  ELSIF origen.version = 9 THEN
    catalogo_esperado := 24; asignacion_esperada := 6;
    destino_ref := 'rol:administracion_perfiles:v10';
  ELSE RAISE EXCEPTION 'AUT67: versión origen inesperada'; END IF;
  IF origen.rol_id <> 'administracion_perfiles'
   OR origen.huella_sha256 IS DISTINCT FROM
      encode(sha256(convert_to(vec_autorizacion.canon_version_rol_admin_v1(origen.documento), 'UTF8')), 'hex')
   OR EXISTS (SELECT 1 FROM vec_autorizacion.version_rol
               WHERE version_rol_ref = destino_ref)
   OR (SELECT count(*) FROM vec_autorizacion.catalogo_accion_nominal_v1
        WHERE version_rol_ref = origen.version_rol_ref) <> catalogo_esperado
   OR (SELECT count(*) FROM vec_autorizacion.asignacion_perfil_actual q
        JOIN vec_autorizacion.asignacion_perfil a USING (perfil_activo_ref, asignacion_ref)
        WHERE a.version_rol_ref = origen.version_rol_ref AND a.version = asignacion_esperada) <> 2
  THEN RAISE EXCEPTION 'AUT67: la instalación alteró la preimagen ADMIN'; END IF;

  concesiones := vec_autorizacion.concesiones_version_inscripcion_admin_v1();
  IF jsonb_array_length(concesiones) <> 2
   OR (SELECT count(DISTINCT v->>'accion') FROM jsonb_array_elements(concesiones) AS j(v)) <> 2
   OR NOT EXISTS (SELECT 1 FROM jsonb_array_elements(concesiones) AS j(v)
      WHERE v->>'accion' = 'administracion.perfiles.version_inscripcion.proponer'
       AND v->>'tipo_recurso' = 'definicion_rol')
   OR NOT EXISTS (SELECT 1 FROM jsonb_array_elements(concesiones) AS j(v)
      WHERE v->>'accion' = 'administracion.perfiles.version_inscripcion.aprobar'
       AND v->>'tipo_recurso' = 'propuesta_definicion_rol')
   OR EXISTS (SELECT 1 FROM jsonb_array_elements(concesiones) AS j(v)
      WHERE v->>'modulo_id' <> 'administracion'
       OR v->'finalidades' <> '["gobierno_definiciones_perfiles"]'::jsonb
       OR v->>'garantia_minima' <> 'alto'
       OR v->'campos_permitidos' <> '[]'::jsonb
       OR v->'obligaciones' <> '[]'::jsonb)
  THEN RAISE EXCEPTION 'AUT67: concesiones nuevas distintas del contrato'; END IF;

  grupo := 'vec_admin_mantenimiento_version_inscripcion_ejecutor'::regrole;
  fachada := to_regprocedure('vec_autorizacion.mantener_version_perfil_fijo_version_inscripcion_admin_v1(text,text)');
  IF fachada IS NULL
   OR EXISTS (SELECT 1 FROM pg_roles WHERE oid = grupo AND (rolcanlogin OR rolsuper OR rolbypassrls))
   OR EXISTS (SELECT 1 FROM pg_database d
       CROSS JOIN LATERAL aclexplode(COALESCE(d.datacl, acldefault('d', d.datdba))) acl
       WHERE d.datname = current_database() AND acl.grantee = grupo)
   OR NOT has_function_privilege(grupo, fachada, 'EXECUTE')
   OR EXISTS (SELECT 1 FROM pg_proc p
       CROSS JOIN LATERAL aclexplode(COALESCE(p.proacl, acldefault('f', p.proowner))) acl
       WHERE p.oid = fachada AND acl.grantee = 0)
   OR EXISTS (SELECT 1 FROM pg_proc p
       CROSS JOIN LATERAL aclexplode(COALESCE(p.proacl, acldefault('f', p.proowner))) acl
       WHERE p.pronamespace = 'vec_autorizacion'::regnamespace
        AND p.proname IN ('aplicar_mantenimiento_version_inscripcion_admin_v1',
                          'preimagen_mantenimiento_version_inscripcion_admin_v1',
                          'exigir_operador_mantenimiento_version_inscripcion_admin_v1')
        AND acl.grantee = grupo)
   OR EXISTS (SELECT 1 FROM pg_class c
       WHERE c.oid IN ('vec_autorizacion.config_mantenimiento_version_inscripcion_admin_v1'::regclass,
                        'vec_autorizacion.registro_mantenimiento_version_inscripcion_admin_v1'::regclass)
        AND (NOT c.relrowsecurity OR NOT c.relforcerowsecurity))
  THEN RAISE EXCEPTION 'AUT67: ACL o RLS divergente'; END IF;
END $prueba$;
ROLLBACK;
