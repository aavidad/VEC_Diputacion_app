-- B145 (F1-60, segundo corte): los actos de Bolsa cotejan el ámbito de la
-- bolsa. Todo termina en ROLLBACK.
-- Comprueba la ACL de la lectura del ámbito y de la constitución (el rol de
-- línea de órdenes ya no constituye), que la lectura devuelve el ámbito
-- declarado y nada para una bolsa sin él, que la confirmación de aceptación
-- (B128) deniega una bolsa de otra unidad o sin ámbito antes de consumir, y que
-- la recuperación de una carga cuya bolsa no tiene ámbito se deniega.
-- Material V3 ficticio: los casos correctos pasan el cotejo y caen después.
\set ON_ERROR_STOP on
BEGIN READ ONLY;
DO $estructura$
DECLARE l oid:=pg_catalog.to_regprocedure('vec_bolsa_llamamientos.leer_ambito_bolsa_v1(text)');
        c oid:=pg_catalog.to_regprocedure('vec_bolsa_llamamientos.constituir_bolsa_v1(text,text,text,text,bigint,bytea,timestamptz,text,bigint,bytea,timestamptz,timestamptz,jsonb,timestamptz)');
BEGIN
 IF l IS NULL OR c IS NULL
 OR NOT pg_catalog.has_function_privilege('vec_bolsa_llamamientos_ejecutor',l,'EXECUTE')
 OR pg_catalog.has_function_privilege('vec_bolsa_llamamientos_constituidor',c,'EXECUTE')
 OR pg_catalog.has_function_privilege('vec_bolsa_llamamientos_constituidor',
   'vec_bolsa_llamamientos.registrar_vinculos_candidato_v1(text,jsonb,timestamptz)','EXECUTE')
 OR pg_catalog.has_schema_privilege('vec_bolsa_llamamientos_constituidor','vec_bolsa_llamamientos','USAGE')
 OR pg_catalog.has_function_privilege('vec_bolsa_llamamientos_ejecutor',c,'EXECUTE')
 OR EXISTS (SELECT 1 FROM pg_catalog.pg_proc p CROSS JOIN LATERAL
     pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
     WHERE p.oid=l AND a.grantee NOT IN (p.proowner,'vec_bolsa_llamamientos_ejecutor'::pg_catalog.regrole))
 THEN RAISE EXCEPTION 'B145: ACL divergente' USING ERRCODE='55000'; END IF;
 -- El rol de línea de órdenes no lee ámbitos (el de desarrollo sí, porque
 -- pertenece al ejecutor web).
 IF pg_catalog.has_function_privilege('vec_bolsa_llamamientos_constituidor',l,'EXECUTE') THEN
  RAISE EXCEPTION 'B145: el constituidor no debe leer el ámbito' USING ERRCODE='55000'; END IF;
END $estructura$;
ROLLBACK;

BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL TIME ZONE 'UTC';
SET LOCAL statement_timeout='10s';
SET LOCAL idle_in_transaction_session_timeout='15s';
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
DO $preparacion$
DECLARE h text:=pg_catalog.repeat('d',64); acta text;
BEGIN
 -- f145a: bolsa de la unidad del actor; f145b: de otra unidad; f145c: sin ámbito.
 PERFORM vec_bolsa_llamamientos.constituir_bolsa_v1('acta:f145:a','actor:f145','categoria:f145a',
  'bolsa:f145:a',1,'\x7b7d',now(),'instantanea:f145:a',1,'\x7b7d',now(),now(),
  '[{"orden":1,"participacion_ref":"participacion:f145:a","fila_numero":1}]',now());
 PERFORM vec_bolsa_llamamientos.declarar_ambito_bolsa_v1('bolsa:f145:a',
  'ambito:desarrollo:bolsa','unidad:desarrollo:rrhh','semilla:prueba:b145');
 PERFORM vec_bolsa_llamamientos.constituir_bolsa_v1('acta:f145:b','actor:f145','categoria:f145b',
  'bolsa:f145:b',1,'\x7b7d',now(),'instantanea:f145:b',1,'\x7b7d',now(),now(),
  '[{"orden":1,"participacion_ref":"participacion:f145:b","fila_numero":1}]',now());
 PERFORM vec_bolsa_llamamientos.declarar_ambito_bolsa_v1('bolsa:f145:b',
  'ambito:desarrollo:bolsa','unidad:desarrollo:otra','semilla:prueba:b145');
 PERFORM vec_bolsa_llamamientos.constituir_bolsa_v1('acta:f145:c','actor:f145','categoria:f145c',
  'bolsa:f145:c',1,'\x7b7d',now(),'instantanea:f145:c',1,'\x7b7d',now(),now(),
  '[{"orden":1,"participacion_ref":"participacion:f145:c","fila_numero":1}]',now());
 -- f145r: carga de CONVOCA anterior a B144 con recibo y sin ámbito declarado.
 acta:='acta:importacion-convoca:'||pg_catalog.encode(pg_catalog.sha256(
  pg_catalog.convert_to(h||pg_catalog.chr(31)||'categoria:f145r','UTF8')),'hex');
 PERFORM vec_bolsa_llamamientos.constituir_bolsa_v1(acta,'actor:f145','categoria:f145r',
  'bolsa:f145:r',1,'\x7b7d',now(),'instantanea:f145:r',1,'\x7b7d',now(),now(),
  '[{"orden":1,"participacion_ref":"participacion:f145:r","fila_numero":1}]',now());
 INSERT INTO vec_bolsa_llamamientos.recibo_carga_convoca(acta_ref,actor_ref,categoria_ref,bolsa_ref,
  huella_fichero_sha256,huella_entradas_sha256,huella_vinculos_sha256,recibo_constitucion,recibo_vinculos,
  decision_ref,auditoria_ref,consumida_en)
 VALUES(acta,'per_f145','categoria:f145r','bolsa:f145:r',h,pg_catalog.repeat('e',64),pg_catalog.repeat('f',64),
  '{}','{}','decision:f145:r','aud_v3_'||pg_catalog.repeat('a',32),now());
END $preparacion$;
RESET ROLE;

-- La entrada web lee el ámbito con la función y nunca la tabla.
CREATE ROLE f145_sesion NOLOGIN IN ROLE vec_bolsa_llamamientos_ejecutor;
SET LOCAL SESSION AUTHORIZATION f145_sesion;
DO $lectura$
DECLARE a record; n bigint;
BEGIN
 SELECT * INTO STRICT a FROM vec_bolsa_llamamientos.leer_ambito_bolsa_v1('bolsa:f145:b');
 IF a.ambito_ref IS DISTINCT FROM 'ambito:desarrollo:bolsa' OR a.unidad_ref IS DISTINCT FROM 'unidad:desarrollo:otra' THEN
  RAISE EXCEPTION 'B145: ámbito leído divergente %', a USING ERRCODE='55000'; END IF;
 SELECT pg_catalog.count(*) INTO n FROM vec_bolsa_llamamientos.leer_ambito_bolsa_v1('bolsa:f145:c');
 IF n<>0 THEN RAISE EXCEPTION 'B145: bolsa sin ámbito con ámbito leído' USING ERRCODE='55000'; END IF;
 BEGIN
  PERFORM 1 FROM vec_bolsa_llamamientos.ambito_bolsa;
  RAISE EXCEPTION 'B145: la entrada web lee la tabla' USING ERRCODE='55000';
 EXCEPTION WHEN insufficient_privilege THEN NULL;
 END;
END $lectura$;

-- B128: confirmación de una aceptación directa con material V3 ficticio. La
-- bolsa de otra unidad y la bolsa sin ámbito se deniegan con B145 antes de
-- consumir; la de la unidad del actor pasa el cotejo y cae en la decisión.
DO $aceptacion$
DECLARE caso record; canon text; h text; contexto bytea;
BEGIN
 FOR caso IN SELECT * FROM (VALUES
   ('bolsa:f145:b','unidad:desarrollo:rrhh','B145: bolsa sin ámbito o de otro ámbito%'),
   ('bolsa:f145:c','unidad:desarrollo:rrhh','B145: bolsa sin ámbito o de otro ámbito%'),
   ('bolsa:f145:a','unidad:desarrollo:otra','B145: bolsa sin ámbito o de otro ámbito%'),
   ('bolsa:f145:a','unidad:desarrollo:rrhh','B128: decisión no ligada%')) AS v(bolsa,unidad,esperado)
 LOOP
  canon:=pg_catalog.jsonb_build_object('esquema','vec.bolsa.respuesta-revision.confirmar-aceptacion.v1',
   'bolsa_ref',caso.bolsa,'participacion_ref','participacion:f145','llamamiento_ref','llamamiento:f145',
   'actor_ref','per_f145abcdefghijklmnopqrstuv','destino','pendiente_incorporacion','motivo','Acepta',
   'clave_idempotencia','clave:f145:aceptacion',
   'situacion_desde_esperada','2026-10-01T10:00:00.000000Z')::text;
  h:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(canon,'UTF8')),'hex');
  contexto:=pg_catalog.convert_to('{"ambitos":{"ambito_ref":"ambito:desarrollo:bolsa","unidad_ref":"'
   ||caso.unidad||'"},"atributos":{"confirmacion_aceptacion_sha256":"'||h||'"}}','UTF8');
  BEGIN
   PERFORM vec_bolsa_llamamientos.confirmar_aceptacion_directa_v1(canon,contexto,
    '{}'::bytea,'{}'::bytea,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
   RAISE EXCEPTION 'B145: aceptación ficticia aceptada (%)', caso.bolsa USING ERRCODE='55000';
  EXCEPTION WHEN insufficient_privilege THEN
   IF SQLERRM NOT LIKE caso.esperado THEN
    RAISE EXCEPTION 'B145: % (%) dio «%»', caso.bolsa, caso.unidad, SQLERRM USING ERRCODE='55000'; END IF;
  END;
 END LOOP;
END $aceptacion$;
RESET SESSION AUTHORIZATION;

-- B95/B144: la recuperación de la carga anterior cuya bolsa no tiene ámbito
-- se deniega antes de AD218 (antes de B145 devolvía el recibo original).
SET LOCAL ROLE vec_bolsa_llamamientos_ejecutor;
DO $recuperacion$
DECLARE actor text:='per_f145'; h text:=pg_catalog.repeat('d',64); cat text:='categoria:f145r';
 bolsa text:='bolsa:f145:r'; acta text; a jsonb; b bytea; contexto bytea; c bytea; d bytea; huella text;
BEGIN
 acta:='acta:importacion-convoca:'||pg_catalog.encode(pg_catalog.sha256(
  pg_catalog.convert_to(h||pg_catalog.chr(31)||cat,'UTF8')),'hex');
 a:=pg_catalog.jsonb_build_object('acta_ref',acta,'categoria_ref',cat,'bolsa_ref',bolsa,
  'huella_fichero_sha256',h,'actor_ref','actor:rrhh:'||pg_catalog.substr(pg_catalog.encode(
   pg_catalog.sha256(pg_catalog.convert_to('vec.bolsa.carga_convoca.actor'||pg_catalog.chr(31)||actor,'UTF8')),'hex'),1,32));
 b:=pg_catalog.convert_to(pg_catalog.jsonb_build_object('bolsa_ref',bolsa,
  'categoria_ref',cat,'huella_listado_sha256',h)::text,'UTF8');
 contexto:=pg_catalog.convert_to('{"ambitos":{"ambito_ref":"ambito:desarrollo:bolsa","unidad_ref":"unidad:desarrollo:rrhh"},"atributos":{}}','UTF8');
 huella:=pg_catalog.encode(pg_catalog.sha256(contexto),'hex');
 c:=pg_catalog.convert_to(pg_catalog.jsonb_build_object('efecto_ref',acta,
  'huella_efecto_sha256',huella,'audiencia_consumo','vec_bolsa_llamamientos.carga_convoca.confirmar.v1')::text,'UTF8');
 d:=pg_catalog.convert_to(pg_catalog.jsonb_build_object('decision_ref','decision:ficticia',
  'principal_id',actor,'recurso_ref',acta,'contexto_recurso_huella_sha256',huella)::text,'UTF8');
 BEGIN
  PERFORM vec_bolsa_llamamientos.confirmar_carga_convoca_v2(a,'[{"numero":1}]','{}',acta,actor,cat,bolsa,
   1,b,now(),'instantanea:f145:r',1,'\x7b7d',now(),now(),'[{"participacion_ref":"p","fila_numero":1}]',now(),'[]',contexto,
   c,d,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
  RAISE EXCEPTION 'B145: recuperación sin ámbito aceptada' USING ERRCODE='55000';
 EXCEPTION WHEN insufficient_privilege THEN
  IF SQLERRM NOT LIKE 'B145: bolsa sin ámbito declarado%' THEN
   RAISE EXCEPTION 'B145: recuperación dio «%»', SQLERRM USING ERRCODE='55000'; END IF;
 END;
END $recuperacion$;
ROLLBACK;
SELECT 'B145 ámbito en los actos de Bolsa OK' AS resultado;
