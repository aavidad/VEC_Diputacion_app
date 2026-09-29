\set ON_ERROR_STOP on
-- Ejecutar solo en clon desechable PG18 después de Bolsa 60 y 61.
SET search_path=pg_catalog;
DO $pre$ BEGIN
 IF to_regrole('vec_b61_registro_prueba') IS NOT NULL THEN
  RAISE EXCEPTION 'B61: rol de prueba ya existe';
 END IF;
 IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper) THEN
  RAISE EXCEPTION 'B61: ensayo requiere DBA en clon';
 END IF;
END $pre$;
CREATE ROLE vec_b61_registro_prueba LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
GRANT vec_bolsa_llamamientos_registrador_portal_externo TO vec_b61_registro_prueba
 WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
SET SESSION AUTHORIZATION vec_b61_registro_prueba;
DO $valido$ BEGIN
 IF NOT vec_bolsa_llamamientos.registrar_denegacion_portal_externo_v1(
  'corr_11111111111111111111111111111111','autenticacion_requerida',
  'api.bolsa.mi_bolsa.ruta_exacta','/api/vec/bolsa/mi-bolsa',NULL)
 THEN RAISE EXCEPTION 'B61: 401 no registrado'; END IF;
 IF NOT vec_bolsa_llamamientos.registrar_denegacion_portal_externo_v1(
  'corr_22222222222222222222222222222222','acceso_denegado',
  'api.bolsa.mi_bolsa.ruta_exacta','/api/vec/bolsa/mi-bolsa/contacto',
  'per_AAAAAAAAAAAAAAAAAAAAAA')
 THEN RAISE EXCEPTION 'B61: 403 no registrado'; END IF;
END $valido$;
DO $seis_rutas$ DECLARE ruta text; numero integer:=2; BEGIN
 FOREACH ruta IN ARRAY ARRAY[
  '/api/vec/bolsa/mi-bolsa/historial',
  '/api/vec/bolsa/mi-bolsa/solicitudes',
  '/api/vec/bolsa/mi-bolsa/respuestas',
  '/api/vec/bolsa/mi-bolsa/disposiciones'] LOOP
  numero:=numero+1;
  IF NOT vec_bolsa_llamamientos.registrar_denegacion_portal_externo_v1(
   'corr_'||pg_catalog.lpad(pg_catalog.to_hex(numero),32,'0'),
   'acceso_denegado','api.bolsa.mi_bolsa.ruta_exacta',ruta,
   'per_AAAAAAAAAAAAAAAAAAAAAA')
  THEN RAISE EXCEPTION 'B61: ruta no registrada: %',ruta; END IF;
 END LOOP;
END $seis_rutas$;
DO $rechazos$ DECLARE n integer; BEGIN
 n:=0;
 BEGIN
  PERFORM vec_bolsa_llamamientos.registrar_denegacion_portal_externo_v1(
   'corr_11111111111111111111111111111111','autenticacion_requerida',
   'api.bolsa.mi_bolsa.ruta_exacta','/api/vec/bolsa/mi-bolsa',NULL);
 EXCEPTION WHEN unique_violation THEN n:=n+1; END;
 BEGIN
  PERFORM vec_bolsa_llamamientos.registrar_denegacion_portal_externo_v1(
   'corr_11111111111111111111111111111111','acceso_denegado',
   'api.bolsa.mi_bolsa.ruta_exacta','/api/vec/bolsa/mi-bolsa/contacto',
   'per_AAAAAAAAAAAAAAAAAAAAAA');
 EXCEPTION WHEN unique_violation THEN n:=n+1; END;
 BEGIN
  PERFORM vec_bolsa_llamamientos.registrar_denegacion_portal_externo_v1(
   'corr_33333333333333333333333333333333','autenticacion_requerida',
   'api.bolsa.mi_bolsa.ruta_exacta','/api/vec/bolsa/mi-bolsa?persona=1',NULL);
 EXCEPTION WHEN invalid_parameter_value THEN n:=n+1; END;
 BEGIN
  PERFORM vec_bolsa_llamamientos.registrar_denegacion_portal_externo_v1(
   'corr_44444444444444444444444444444444','acceso_denegado',
   'api.bolsa.mi_bolsa.ruta_exacta','/api/vec/bolsa/mi-bolsa','per_A');
 EXCEPTION WHEN invalid_parameter_value THEN n:=n+1; END;
 IF n<>4 THEN RAISE EXCEPTION 'B61: replay o entrada libre aceptados: %',n; END IF;
 IF has_table_privilege(current_user,'vec_bolsa_llamamientos.denegacion_frontera_portal_externo','SELECT,INSERT,UPDATE,DELETE,TRUNCATE')
    OR has_function_privilege(current_user,
     'vec_bolsa_llamamientos.consultar_mi_bolsa_v1(text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 THEN RAISE EXCEPTION 'B61: registrador con ACL de negocio'; END IF;
END $rechazos$;
RESET SESSION AUTHORIZATION;

-- Una concesión ajena convierte el mismo LOGIN en inválido en tiempo de uso.
GRANT SELECT ON vec_bolsa_llamamientos.denegacion_frontera_portal_externo TO vec_b61_registro_prueba;
SET SESSION AUTHORIZATION vec_b61_registro_prueba;
DO $extra$ BEGIN
 BEGIN
  PERFORM vec_bolsa_llamamientos.registrar_denegacion_portal_externo_v1(
   'corr_55555555555555555555555555555555','autenticacion_requerida',
   'api.bolsa.mi_bolsa.ruta_exacta','/api/vec/bolsa/mi-bolsa',NULL);
  RAISE EXCEPTION 'B61: ACL extra aceptada';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $extra$;
RESET SESSION AUTHORIZATION;
REVOKE SELECT ON vec_bolsa_llamamientos.denegacion_frontera_portal_externo FROM vec_b61_registro_prueba;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.registrar_denegacion_portal_externo_v1(text,text,text,text,text) TO PUBLIC;
SET SESSION AUTHORIZATION vec_b61_registro_prueba;
DO $publico$ BEGIN
 BEGIN
  PERFORM vec_bolsa_llamamientos.registrar_denegacion_portal_externo_v1(
   'corr_66666666666666666666666666666666','autenticacion_requerida',
   'api.bolsa.mi_bolsa.ruta_exacta','/api/vec/bolsa/mi-bolsa',NULL);
  RAISE EXCEPTION 'B61: EXECUTE de PUBLIC aceptado';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $publico$;
RESET SESSION AUTHORIZATION;
REVOKE EXECUTE ON FUNCTION vec_bolsa_llamamientos.registrar_denegacion_portal_externo_v1(text,text,text,text,text) FROM PUBLIC;

-- El ejecutor B60 no puede registrar ni leer los rechazos.
DO $separacion$ BEGIN
 IF has_function_privilege('vec_bolsa_llamamientos_portal_externo',
   'vec_bolsa_llamamientos.registrar_denegacion_portal_externo_v1(text,text,text,text,text)','EXECUTE')
   OR has_table_privilege('vec_bolsa_llamamientos_portal_externo',
     'vec_bolsa_llamamientos.denegacion_frontera_portal_externo','SELECT,INSERT,UPDATE,DELETE,TRUNCATE')
 THEN RAISE EXCEPTION 'B61: ejecutor externo puede acceder a auditoría'; END IF;
 IF (SELECT count(*) FROM vec_bolsa_llamamientos.denegacion_frontera_portal_externo)<>6
 THEN RAISE EXCEPTION 'B61: historia no exacta'; END IF;
END $separacion$;
