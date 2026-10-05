\set ON_ERROR_STOP on
-- Prueba causal de AD200 en un clon desechable con AD200 instalada y ANTES de AD201 (después la v1
-- ya no es ejecutable por el grupo; ver ad201_gobierno_plan_firma_ambitos.sql). Todo dentro
-- de una transacción que termina en ROLLBACK: crea LOGIN sintéticos y llama a la
-- fachada de gobierno con un material coherente y una decisión sin firma.
-- Esperado:
--  * LOGIN exclusivo del grupo dedicado: pasa la fachada y la rama de sesión del
--    núcleo y se detiene después (nunca «consumo VEC-AD-3 rechazado»).
--  * LOGIN del grupo con otra pertenencia: el núcleo lo rechaza en la sesión.
--  * LOGIN del runtime CT: ya no tiene EXECUTE sobre la fachada.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog;
SET LOCAL idle_in_transaction_session_timeout='20s';
SET LOCAL statement_timeout='15s';
CREATE ROLE ad200_prueba_gobierno LOGIN INHERIT;
GRANT vec_plan_firma_gobierno_ejecutor TO ad200_prueba_gobierno WITH INHERIT TRUE, SET FALSE;
CREATE ROLE ad200_prueba_doble LOGIN INHERIT;
GRANT vec_plan_firma_gobierno_ejecutor TO ad200_prueba_doble WITH INHERIT TRUE, SET FALSE;
GRANT pg_read_all_stats TO ad200_prueba_doble;
CREATE ROLE ad200_prueba_runtime LOGIN INHERIT;
GRANT vec_contratacion_temporal_ejecutor TO ad200_prueba_runtime WITH INHERIT TRUE, SET FALSE;
CREATE TEMP TABLE ad200_entrada ON COMMIT DROP AS
WITH cat AS (SELECT convert_to('{"estado":"publicado","id":"plan.prueba.ad200","modulo_id":"contratacion_temporal","revision":1,"version":1}','UTF8') AS b),
mat AS (SELECT convert_to(json_build_object('esquema','vec.catalogos.plan-firma.gobierno.v1','operacion','publicar',
  'catalogo_id','plan.prueba.ad200','version',1,'revision_esperada',1,'huella_esperada',repeat('0',64),'clave_operacion','ad200-prueba',
  'catalogo_canonico_base64',encode(cat.b,'base64'),'catalogo_sha256',encode(sha256(cat.b),'hex'),
  'traza_canonica_base64','e30=','traza_sha256',repeat('1',64),'evento_canonico_base64','e30=','evento_sha256',repeat('2',64))::text,'UTF8') AS m FROM cat)
SELECT m, encode(sha256(convert_to('{"ambitos":{},"atributos":{"estado":"publicado","material_sha256":"'||encode(sha256(m),'hex')||'","revision":"1"}}','UTF8')),'hex') AS h FROM mat;
GRANT SELECT ON ad200_entrada TO ad200_prueba_gobierno, ad200_prueba_doble, ad200_prueba_runtime;
CREATE FUNCTION pg_temp.ad200_llamar(esperado text) RETURNS void LANGUAGE plpgsql AS $f$
DECLARE e record; estado text; mensaje text;
BEGIN
 SELECT * INTO STRICT e FROM pg_temp.ad200_entrada;
 BEGIN
  PERFORM vec_autorizacion_atestada_v3.registrar_y_confirmar_gobierno_plan_firma_v1(e.m,
   convert_to(json_build_object('operacion','vec.catalogos.publicar','efecto_ref','plan.prueba.ad200:1','huella_efecto_sha256',e.h,
     'audiencia_consumo','vec_catalogos_configurables.plan_nominal_firma.gobierno.v1')::text,'UTF8'),
   convert_to(json_build_object('accion','vec.catalogos.publicar','recurso_ref','plan.prueba.ad200:1','contexto_recurso_huella_sha256',e.h,
     'modulo_id','contratacion_temporal','tipo_recurso','catalogo_configurable','finalidad','gestionar_contratacion_temporal',
     'campos_permitidos','[]'::json,'obligaciones','[]'::json)::text,'UTF8'),
   '\x7b7d','\x7b7d',1,1,'\x7b7d','\x7b7d','\x7b7d','\x00');
  RAISE EXCEPTION 'AD200 prueba: la llamada no debía terminar bien (%)',session_user;
 EXCEPTION WHEN OTHERS THEN
  GET STACKED DIAGNOSTICS estado=RETURNED_SQLSTATE, mensaje=MESSAGE_TEXT;
  IF mensaje LIKE 'AD200 prueba:%' THEN RAISE; END IF;
  IF esperado='pasa_sesion' AND (estado<>'22023' OR mensaje<>'entrada VEC-AD-3 inválida') THEN
   RAISE EXCEPTION 'AD200 prueba: % no pasó la sesión: % %',session_user,estado,mensaje; END IF;
  IF esperado='rechazo_sesion' AND mensaje<>'consumo VEC-AD-3 rechazado' THEN
   RAISE EXCEPTION 'AD200 prueba: % no se rechazó en la sesión: % %',session_user,estado,mensaje; END IF;
  IF esperado='sin_execute' AND (estado<>'42501' OR mensaje NOT LIKE 'permission denied for function%') THEN
   RAISE EXCEPTION 'AD200 prueba: % conserva EXECUTE: % %',session_user,estado,mensaje; END IF;
  RAISE NOTICE 'AD200 prueba OK: % → % %',session_user,estado,mensaje;
 END;
END $f$;
SET SESSION AUTHORIZATION ad200_prueba_gobierno;
SELECT pg_temp.ad200_llamar('pasa_sesion');
RESET SESSION AUTHORIZATION;
SET SESSION AUTHORIZATION ad200_prueba_doble;
SELECT pg_temp.ad200_llamar('rechazo_sesion');
RESET SESSION AUTHORIZATION;
SET SESSION AUTHORIZATION ad200_prueba_runtime;
SELECT pg_temp.ad200_llamar('sin_execute');
RESET SESSION AUTHORIZATION;
ROLLBACK;
\echo AD200_PRUEBA_OK
