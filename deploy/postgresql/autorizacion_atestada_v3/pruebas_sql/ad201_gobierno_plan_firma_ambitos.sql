\set ON_ERROR_STOP on
-- Prueba de AD201 en un clon desechable con AD200 y AD201 instaladas. Todo en
-- una transacción que termina en ROLLBACK. Con un LOGIN exclusivo del grupo:
--  * v2 con la huella que incluye organización y unidad: la fachada la acepta y
--    llega al núcleo, que se detiene en la entrada sintética (22023);
--  * v2 con la huella antigua sin ámbitos: la fachada la deniega (42501);
--  * v2 con una organización mal formada: 22023 de AD201 antes de nada;
--  * v1: sin EXECUTE.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog;
SET LOCAL idle_in_transaction_session_timeout='20s';
SET LOCAL statement_timeout='15s';
CREATE ROLE ad201_prueba_gobierno LOGIN INHERIT;
GRANT vec_plan_firma_gobierno_ejecutor TO ad201_prueba_gobierno WITH INHERIT TRUE, SET FALSE;
CREATE TEMP TABLE ad201_entrada ON COMMIT DROP AS
WITH cat AS (SELECT convert_to('{"estado":"publicado","id":"plan.prueba.ad201","modulo_id":"contratacion_temporal","revision":1,"version":1}','UTF8') AS b),
mat AS (SELECT convert_to(json_build_object('esquema','vec.catalogos.plan-firma.gobierno.v1','operacion','publicar',
  'catalogo_id','plan.prueba.ad201','version',1,'revision_esperada',1,'huella_esperada',repeat('0',64),'clave_operacion','ad201-prueba',
  'catalogo_canonico_base64',encode(cat.b,'base64'),'catalogo_sha256',encode(sha256(cat.b),'hex'),
  'traza_canonica_base64','e30=','traza_sha256',repeat('1',64),'evento_canonico_base64','e30=','evento_sha256',repeat('2',64))::text,'UTF8') AS m FROM cat)
SELECT m, 'org_0123456789abcdef0123'::text AS org, 'unidad:sintetica:ad201'::text AS unidad,
 encode(sha256(convert_to('{"ambitos":{"organizacion_ref":"org_0123456789abcdef0123","unidad_ref":"unidad:sintetica:ad201"},"atributos":{"estado":"publicado","material_sha256":"'||encode(sha256(m),'hex')||'","revision":"1"}}','UTF8')),'hex') AS h,
 encode(sha256(convert_to('{"ambitos":{},"atributos":{"estado":"publicado","material_sha256":"'||encode(sha256(m),'hex')||'","revision":"1"}}','UTF8')),'hex') AS h_sin
FROM mat;
GRANT SELECT ON ad201_entrada TO ad201_prueba_gobierno;
CREATE FUNCTION pg_temp.ad201_llamar(caso text) RETURNS void LANGUAGE plpgsql AS $f$
DECLARE e record; h text; org text; estado text; mensaje text;
BEGIN
 SELECT * INTO STRICT e FROM pg_temp.ad201_entrada;
 h:=CASE WHEN caso='sin_ambitos' THEN e.h_sin ELSE e.h END;
 org:=CASE WHEN caso='organizacion_invalida' THEN 'org_X' ELSE e.org END;
 BEGIN
  IF caso='v1' THEN
   PERFORM vec_autorizacion_atestada_v3.registrar_y_confirmar_gobierno_plan_firma_v1(e.m,'\x7b7d','\x7b7d','\x7b7d','\x7b7d',1,1,'\x7b7d','\x7b7d','\x7b7d','\x00');
  ELSE
   PERFORM vec_autorizacion_atestada_v3.registrar_y_confirmar_gobierno_plan_firma_v2(e.m,org,e.unidad,
    convert_to(json_build_object('operacion','vec.catalogos.publicar','efecto_ref','plan.prueba.ad201:1','huella_efecto_sha256',h,
      'audiencia_consumo','vec_catalogos_configurables.plan_nominal_firma.gobierno.v1')::text,'UTF8'),
    convert_to(json_build_object('accion','vec.catalogos.publicar','recurso_ref','plan.prueba.ad201:1','contexto_recurso_huella_sha256',h,
      'modulo_id','contratacion_temporal','tipo_recurso','catalogo_configurable','finalidad','gestionar_contratacion_temporal',
      'campos_permitidos','[]'::json,'obligaciones','[]'::json)::text,'UTF8'),
    '\x7b7d','\x7b7d',1,1,'\x7b7d','\x7b7d','\x7b7d','\x00');
  END IF;
  RAISE EXCEPTION 'AD201 prueba: % no debía terminar bien',caso;
 EXCEPTION WHEN OTHERS THEN
  GET STACKED DIAGNOSTICS estado=RETURNED_SQLSTATE, mensaje=MESSAGE_TEXT;
  IF mensaje LIKE 'AD201 prueba:%' THEN RAISE; END IF;
  IF (caso='con_ambitos' AND (estado,mensaje)<>('22023','entrada VEC-AD-3 inválida'))
  OR (caso='sin_ambitos' AND (estado,mensaje)<>('42501','AD201 gobierno de plan denegado'))
  OR (caso='organizacion_invalida' AND (estado,mensaje)<>('22023','AD201 material de gobierno inválido'))
  OR (caso='v1' AND (estado<>'42501' OR mensaje NOT LIKE 'permission denied for function%')) THEN
   RAISE EXCEPTION 'AD201 prueba: % inesperado: % %',caso,estado,mensaje; END IF;
  RAISE NOTICE 'AD201 prueba OK: % → % %',caso,estado,mensaje;
 END;
END $f$;
SET SESSION AUTHORIZATION ad201_prueba_gobierno;
SELECT pg_temp.ad201_llamar('con_ambitos');
SELECT pg_temp.ad201_llamar('sin_ambitos');
SELECT pg_temp.ad201_llamar('organizacion_invalida');
SELECT pg_temp.ad201_llamar('v1');
RESET SESSION AUTHORIZATION;
ROLLBACK;
\echo AD201_PRUEBA_OK
