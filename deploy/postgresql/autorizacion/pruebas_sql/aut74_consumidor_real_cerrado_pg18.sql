\set ON_ERROR_STOP on
-- Solo clon desechable: conexión temporal al consumidor real sin activar ACL de producto.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
GRANT EXECUTE ON FUNCTION vec_autorizacion.consumir_material_admin_interno_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_admin_perfiles_ejecutor;
CREATE ROLE prueba_aut74_login LOGIN;
GRANT vec_admin_perfiles_ejecutor TO prueba_aut74_login WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
SET SESSION AUTHORIZATION prueba_aut74_login;
DO $prueba$
DECLARE accion text; audiencia text; referencia text; m text; c jsonb; d jsonb; huella text; mensaje text; codigo text;
BEGIN
 FOR accion IN SELECT unnest(ARRAY[
  'administracion.perfiles.otorgar','administracion.perfiles.revocar',
  'administracion.perfiles.proponer','administracion.perfiles.aprobar',
  'administracion.perfiles.rechazar']) LOOP
  audiencia:=CASE
   WHEN accion IN ('administracion.perfiles.otorgar','administracion.perfiles.revocar') THEN 'vec_autorizacion.administracion_perfiles.ordinario.v1'
   WHEN accion='administracion.perfiles.proponer' THEN 'vec_autorizacion.administracion_perfiles.propuesta.v1'
   ELSE 'vec_autorizacion.administracion_perfiles.cierre.v1' END;
  referencia:=CASE
   WHEN accion='administracion.perfiles.proponer' THEN 'propuesta_admin:'
   WHEN accion IN ('administracion.perfiles.aprobar','administracion.perfiles.rechazar') THEN 'cierre_admin:'
   ELSE 'acto_admin:' END||repeat('a',32);
  m:=jsonb_build_object('operacion_ref',referencia,'asignacion_ref','asignacion:prueba:v1',
   'actor_persona_ref','per_'||repeat('b',22),'actor_perfil_ref','prf_'||repeat('b',22))::text;
  huella:=encode(sha256(convert_to(m,'UTF8')),'hex');
  c:=jsonb_build_object('efecto_ref',referencia,'huella_efecto_sha256',huella,
   'operacion',accion,'audiencia_consumo',audiencia);
  d:=jsonb_build_object('accion',accion,'recurso_ref',referencia,
   'principal_id','per_'||repeat('b',22),'perfil_activo_ref','prf_'||repeat('b',22),
   'asignacion_ref','asignacion:prueba:v1','modulo_id','administracion',
   'finalidad','gestion_perfiles','vinculo_autenticacion_actor',
    jsonb_build_object('superficie','administracion_privilegiada','cuenta_privilegiada',true));
  BEGIN
   PERFORM vec_autorizacion.consumir_material_admin_interno_v1(m,convert_to(c::text,'UTF8'),
    convert_to(d::text,'UTF8'),'\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
   RAISE EXCEPTION 'AUT74: acto sin firma admitido %',accion;
  EXCEPTION WHEN insufficient_privilege THEN
   GET STACKED DIAGNOSTICS codigo=RETURNED_SQLSTATE,mensaje=MESSAGE_TEXT;
   IF codigo<>'42501' OR mensaje NOT LIKE 'AD236:%' THEN
    RAISE EXCEPTION 'AUT74: acto % no llegó al consumidor nominal: % %',accion,codigo,mensaje;
   END IF;
  END;
 END LOOP;
END $prueba$;
RESET SESSION AUTHORIZATION;
ROLLBACK;
