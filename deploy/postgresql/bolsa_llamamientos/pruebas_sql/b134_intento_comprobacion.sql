\set ON_ERROR_STOP on
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL timezone='UTC';
DO $login$ BEGIN
 EXECUTE format('CREATE ROLE vec_b134_fixture LOGIN INHERIT NOSUPERUSER NOCREATEDB '
  'NOCREATEROLE NOREPLICATION NOBYPASSRLS VALID UNTIL %L',clock_timestamp()+interval '90 days');
END $login$;
GRANT vec_bolsa_llamamientos_registrador_frontera TO vec_b134_fixture
 WITH ADMIN FALSE,INHERIT TRUE,SET FALSE;
SET LOCAL SESSION AUTHORIZATION vec_b134_fixture;
SELECT vec_bolsa_llamamientos.registrar_intento_borrador_llamamiento_v1(
 'correlacion:b134:nominal','registrar_comprobacion_ofrecimiento',
 'comprobacion_ofrecimiento',NULL,'acceso_denegado');
DO $negativos$ BEGIN
 BEGIN
  PERFORM vec_bolsa_llamamientos.registrar_intento_borrador_llamamiento_v1(
   'correlacion:b134:otra','cambiar_situacion','comprobacion_ofrecimiento',NULL,'correcto');
  RAISE EXCEPTION 'se atribuyó otro acto a la comprobación';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 BEGIN
  PERFORM vec_bolsa_llamamientos.registrar_intento_borrador_llamamiento_v1(
   'correlacion:b134:otra','registrar_comprobacion_ofrecimiento','situacion',NULL,'correcto');
  RAISE EXCEPTION 'se perdió la clase nominal';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
END $negativos$;
RESET SESSION AUTHORIZATION;
DO $registro$ BEGIN
 IF (SELECT count(*) FROM vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento
  WHERE correlacion_ref='correlacion:b134:nominal'
   AND accion='registrar_comprobacion_ofrecimiento'
   AND ruta_clase='comprobacion_ofrecimiento' AND resultado='acceso_denegado')<>1
 THEN RAISE EXCEPTION 'falta la denegación nominal'; END IF;
END $registro$;
ROLLBACK;
