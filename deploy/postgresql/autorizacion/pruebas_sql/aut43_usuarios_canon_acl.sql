\set ON_ERROR_STOP on
-- No emite V3 favorable: transportes sintéticos y fachadas puras privadas.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';
CREATE TEMP TABLE aut43_vector(material text,contexto text,huella text);
INSERT INTO aut43_vector VALUES ('{"esquema":"vec.admin.usuarios.listar.v1","organizacion_ref":"org_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","unidad_ref":"unidad_admin_sintetica","conjunto_ref":"conjunto_admin:dfa8fa3eef2981ce04ad7dcccbee704d","filtros":{"perfil_ref":"","unidad_ref":"","estado":""},"cursor":"","limite":50}','{"ambitos":{"organizacion_ref":"org_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","unidad_ref":"unidad_admin_sintetica"},"atributos":{"material_sha256":"eb9b4ee7842f339489059734f0613261f5a4464555226c2bc5dc6111cbdf6776"}}','a44970b1679b0fb6b534263e3f3923540fa881ba4817b30084a052d24b6c787a'),('{"esquema":"vec.admin.usuarios.consultar.v1","organizacion_ref":"org_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","unidad_ref":"unidad_admin_sintetica","conjunto_ref":"conjunto_admin:dfa8fa3eef2981ce04ad7dcccbee704d","persona_ref":"per_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}','{"ambitos":{"organizacion_ref":"org_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","unidad_ref":"unidad_admin_sintetica"},"atributos":{"material_sha256":"8a6e67f3c815c001060699aeab2101310c675dbbd093f5875c9739e8033de874"}}','23488ca0bca6ef20ede975f7aaabe00976ddadb349c3e757acd4d110c00d82b1');
GRANT SELECT ON TABLE pg_temp.aut43_vector TO vec_autorizacion_atestada_v3_propietario,vec_autorizacion_propietario;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
DO $vector$
DECLARE v record;r jsonb;mutado jsonb;
BEGIN
 FOR v IN SELECT * FROM pg_temp.aut43_vector LOOP
  r:=vec_autorizacion.recurso_lectura_usuarios_admin_v1(v.material);
  IF r->>'contexto_canonico' IS DISTINCT FROM v.contexto OR r->>'contexto_sha256' IS DISTINCT FROM v.huella THEN RAISE EXCEPTION 'AUT43: contexto_divergente';END IF;
  IF vec_autorizacion.canon_lectura_usuarios_admin_v1(v.material::jsonb) IS DISTINCT FROM v.material THEN RAISE EXCEPTION 'AUT43: canon_divergente';END IF;
  BEGIN PERFORM vec_autorizacion.canon_lectura_usuarios_admin_v1(v.material::jsonb||'{"actor_ref":"inventado"}'::jsonb);RAISE EXCEPTION 'AUT43: actor_en_material';EXCEPTION WHEN invalid_parameter_value THEN NULL;END;
  BEGIN PERFORM vec_autorizacion.canon_lectura_usuarios_admin_v1(jsonb_set(v.material::jsonb,'{conjunto_ref}','"conjunto_admin:00000000000000000000000000000000"'));RAISE EXCEPTION 'AUT43: conjunto_inventado';EXCEPTION WHEN invalid_parameter_value THEN NULL;END;
  IF r->'material'->>'esquema'='vec.admin.usuarios.listar.v1' THEN
   mutado:=jsonb_set(v.material::jsonb,'{filtros,estado}','"vigente"');
   IF vec_autorizacion.recurso_lectura_usuarios_admin_v1(vec_autorizacion.canon_lectura_usuarios_admin_v1(mutado))->>'contexto_sha256'=v.huella THEN RAISE EXCEPTION 'AUT43: filtro_no_ligado';END IF;
   BEGIN PERFORM vec_autorizacion.canon_lectura_usuarios_admin_v1(jsonb_set(v.material::jsonb,'{limite}','51'));RAISE EXCEPTION 'AUT43: limite_abierto';EXCEPTION WHEN invalid_parameter_value THEN NULL;END;
  END IF;
 END LOOP;
END $vector$;
RESET ROLE;
SET LOCAL ROLE vec_autorizacion_propietario;
DO $cursor$
DECLARE m jsonb;r jsonb;
BEGIN
 SELECT material::jsonb INTO m FROM pg_temp.aut43_vector WHERE material LIKE '%"filtros"%';
 m:=jsonb_set(m,'{cursor}','"usuarios:0000000000000000000000000000000000000000000000000000000000000000:per_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"');
 BEGIN PERFORM vec_autorizacion.proyectar_usuarios_admin_v1(m);RAISE EXCEPTION 'AUT43: cursor_ajeno_admitido';EXCEPTION WHEN invalid_parameter_value THEN NULL;END;
END $cursor$;
RESET ROLE;
DO $acl$
DECLARE f oid;
BEGIN
 FOR f IN SELECT p.oid FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname='vec_autorizacion' AND p.proname IN('proyectar_usuarios_admin_v1','proyectar_persona_usuarios_admin_v1','validar_administrador_usuarios_v1') LOOP
  IF has_function_privilege('vec_admin_usuarios_lector',f,'EXECUTE') THEN RAISE EXCEPTION 'AUT43: lector_accede_helper';END IF;
 END LOOP;
END $acl$;
ROLLBACK;
