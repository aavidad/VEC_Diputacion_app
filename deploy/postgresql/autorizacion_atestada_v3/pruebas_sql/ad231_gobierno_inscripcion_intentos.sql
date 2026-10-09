\set ON_ERROR_STOP on
-- Sólo PG18 aislado: el ROL, la membresía y el intento se revierten juntos.
-- Ejecutar tras AUT66, AD227→AD230→AD228→AD229 y AD231.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
CREATE ROLE vec_ad231_prueba LOGIN INHERIT NOSUPERUSER NOCREATEROLE NOCREATEDB NOREPLICATION NOBYPASSRLS;
GRANT vec_admin_version_inscripcion_ejecutor TO vec_ad231_prueba WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_ad231_prueba;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_intento_version_inscripcion_v1(jsonb) TO vec_ad231_prueba;
SET SESSION AUTHORIZATION vec_ad231_prueba;
DO $test$
DECLARE evento jsonb;
DECLARE primero record;
DECLARE replay record;
BEGIN
 evento:=jsonb_build_object(
  'tipo_registro','intento_version_inscripcion',
  'evento_ref','evento_'||repeat('a',32),
  'operador_login',session_user::text,
  'solicitud_sha256',repeat('b',64),
  'accion','administracion.perfiles.version_inscripcion.proponer',
  'recurso_ref','solicitud_version_inscripcion:'||repeat('c',32),
  'resultado','denegado',
  'motivo_ref','version_inscripcion_denegado',
  'proceso','postgresql',
  'canal','operacion_tecnica_privada',
  'finalidad_ref','gobierno_definiciones_perfiles',
  'correlacion_ref','correlacion_'||repeat('d',32));
 SELECT * INTO STRICT primero
 FROM vec_autorizacion_atestada_v3.registrar_intento_version_inscripcion_v1(evento);
 SELECT * INTO STRICT replay
 FROM vec_autorizacion_atestada_v3.registrar_intento_version_inscripcion_v1(evento);
 IF primero.auditoria_ref IS DISTINCT FROM replay.auditoria_ref
 OR primero.secuencia IS DISTINCT FROM replay.secuencia
 OR primero.huella_sha256 IS DISTINCT FROM replay.huella_sha256
 OR primero.correlacion_ref IS DISTINCT FROM replay.correlacion_ref
 OR primero.registrada_en IS DISTINCT FROM replay.registrada_en
 THEN RAISE EXCEPTION 'AD231: replay alteró recibo' USING ERRCODE='P0001'; END IF;
 BEGIN
  PERFORM * FROM vec_autorizacion_atestada_v3.registrar_intento_version_inscripcion_v1(
   evento||jsonb_build_object('resultado','error','motivo_ref','version_inscripcion_error'));
  RAISE EXCEPTION 'AD231: material distinto reutilizó evento' USING ERRCODE='P0001';
 EXCEPTION WHEN unique_violation THEN NULL;
 END;
END $test$;
ROLLBACK;
