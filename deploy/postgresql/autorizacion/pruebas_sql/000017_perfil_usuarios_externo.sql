\set ON_ERROR_STOP on
-- Ejecutar tras AUT-17 sobre un clon desechable. No conserva datos.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog;
CREATE TEMP TABLE aut17_rol_prueba(documento bytea) ON COMMIT DROP;
DO $ensayo$
DECLARE d jsonb; concesiones jsonb := '[]'::jsonb; x record; b bytea;
BEGIN
 FOR x IN SELECT * FROM (VALUES
  ('vec.preferencias.consultar','preferencias_persona','finalidad:usuarios:preferencias-propias:v1','["catalogo","valores","version"]'::jsonb),
  ('vec.preferencias.actualizar','preferencias_persona','finalidad:usuarios:preferencias-propias:v1','["valores","version"]'::jsonb),
  ('vec.imagen.consultar','imagen_persona','finalidad:usuarios:imagen-propia:v1','["foto","icono","modo","paleta","version"]'::jsonb),
  ('vec.imagen.actualizar','imagen_persona','finalidad:usuarios:imagen-propia:v1','["foto","icono","modo","paleta","version"]'::jsonb),
  ('vec.correos.consultar','correos_persona','finalidad:usuarios:correos-propios:v1','["activo","correo_ref","direccion","estado","version"]'::jsonb),
  ('vec.correos.anadir','correos_persona','finalidad:usuarios:correos-propios:v1','["correo_ref","direccion","estado","version"]'::jsonb),
  ('vec.correos.reenviar','correos_persona','finalidad:usuarios:correos-propios:v1','["correo_ref","estado","version"]'::jsonb),
  ('vec.correos.verificar','correos_persona','finalidad:usuarios:correos-propios:v1','["correo_ref","estado","version"]'::jsonb),
  ('vec.correos.activar','correos_persona','finalidad:usuarios:correos-propios:v1','["activo","correo_ref","version"]'::jsonb),
  ('vec.correos.retirar','correos_persona','finalidad:usuarios:correos-propios:v1','["activo","correo_ref","estado","version"]'::jsonb)
 ) AS c(accion,tipo,finalidad,campos) LOOP
  concesiones:=concesiones||pg_catalog.jsonb_build_array(pg_catalog.jsonb_build_object(
   'accion',x.accion,'modulo_id','usuarios','tipo_recurso',x.tipo,
   'finalidades',pg_catalog.jsonb_build_array(x.finalidad),'campos_permitidos',x.campos,'garantia_minima','alto'));
 END LOOP;
 d:=pg_catalog.jsonb_build_object('rol_id','candidato_usuarios_propios_desarrollo','version',1,
  'nombre','areaPersonal.usuarios.rolPropio','estado','publicada',
  'concesiones',concesiones,'publicada_por','seguridad:desarrollo:no-autoritativa',
  'publicada_en','2026-09-30T00:00:00Z');
 IF vec_autorizacion.rol_usuarios_externo_acotado_v1(d) IS NOT TRUE
 THEN RAISE EXCEPTION 'AUT-17: rol exacto rechazado'; END IF;
 IF vec_autorizacion.rol_usuarios_externo_acotado_v1(pg_catalog.jsonb_set(d,'{concesiones,0,modulo_id}','"bolsa"'::jsonb)) IS TRUE
    OR vec_autorizacion.rol_usuarios_externo_acotado_v1(pg_catalog.jsonb_set(d,'{concesiones,0,campos_permitidos}','["dato_personal"]'::jsonb)) IS TRUE
    OR vec_autorizacion.rol_usuarios_externo_acotado_v1(pg_catalog.jsonb_set(d,'{concesiones,0,finalidades}','["finalidad:bolsa"]'::jsonb)) IS TRUE
 THEN RAISE EXCEPTION 'AUT-17: rol ajeno o ampliado aceptado'; END IF;
 b:=pg_catalog.convert_to(d::text,'UTF8');
 INSERT INTO pg_temp.aut17_rol_prueba(documento) VALUES(b);
 BEGIN
  PERFORM vec_autorizacion.publicar_rol_usuarios_externo_v1(b,pg_catalog.encode(pg_catalog.sha256(b),'hex'),
   b,pg_catalog.encode(pg_catalog.sha256(b),'hex'),0,NULL,'seguridad:desarrollo:no-autoritativa','acto:sin-publicador');
  RAISE EXCEPTION 'AUT-17: publicación por sesión no autorizada';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 IF EXISTS (SELECT 1 FROM pg_catalog.pg_proc p,
       LATERAL pg_catalog.aclexplode(COALESCE(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
     WHERE p.oid IN ('vec_autorizacion.registrar_decision_usuarios_externo_v3(bytea,bytea,numeric,numeric)'::regprocedure,
       'vec_autorizacion.obtener_instantanea_usuarios_externo_v1(text,text)'::regprocedure)
       AND a.grantee=0 AND a.privilege_type='EXECUTE')
 THEN RAISE EXCEPTION 'AUT-17: EXECUTE público'; END IF;
END $ensayo$;
CREATE ROLE aut17_ensayo_publicador LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
GRANT vec_autorizacion_publicador_usuarios_externo TO aut17_ensayo_publicador
 WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
GRANT SELECT ON pg_temp.aut17_rol_prueba TO aut17_ensayo_publicador;
SET SESSION AUTHORIZATION aut17_ensayo_publicador;
DO $publicacion$
DECLARE b bytea; c jsonb; bc bytea; h text; hc text; publicada record;
BEGIN
 SELECT documento INTO STRICT b FROM pg_temp.aut17_rol_prueba;
 h:=pg_catalog.encode(pg_catalog.sha256(b),'hex');
 c:=pg_catalog.jsonb_build_object('version_rol_ref','rol:candidato_usuarios_propios_desarrollo:v1',
  'revision',1,'estado','habilitada','actualizado_por','seguridad:desarrollo:no-autoritativa',
  'actualizado_en','2026-09-30T00:00:00Z');
 bc:=pg_catalog.convert_to(c::text,'UTF8'); hc:=pg_catalog.encode(pg_catalog.sha256(bc),'hex');
 SELECT * INTO publicada FROM vec_autorizacion.publicar_rol_usuarios_externo_v1(
  b,h,bc,hc,0,NULL,'seguridad:desarrollo:no-autoritativa','acto:aut17:rol:ensayo');
 IF publicada.version_rol_ref IS DISTINCT FROM 'rol:candidato_usuarios_propios_desarrollo:v1'
    OR publicada.huella_rol IS DISTINCT FROM h OR publicada.huella_control IS DISTINCT FROM hc
 THEN RAISE EXCEPTION 'AUT-17: publicación positiva incorrecta'; END IF;
 BEGIN
  PERFORM vec_autorizacion.publicar_rol_usuarios_externo_v1(
   b,h,bc,hc,0,NULL,'seguridad:desarrollo:no-autoritativa','acto:aut17:rol:cas-repetido');
  RAISE EXCEPTION 'AUT-17: CAS repetido aceptado';
 EXCEPTION WHEN serialization_failure THEN NULL; END;
END $publicacion$;
RESET SESSION AUTHORIZATION;
ROLLBACK;
