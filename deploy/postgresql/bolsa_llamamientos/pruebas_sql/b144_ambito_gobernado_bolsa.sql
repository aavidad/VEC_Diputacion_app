-- B144 (F1-60): ámbito gobernado de la bolsa. Todo termina en ROLLBACK.
-- Comprueba estructura y ACL, la declaración del propietario (idempotencia,
-- conflicto, bolsa inexistente, inmutabilidad) y que confirmar_carga_convoca_v2
-- deniega una bolsa de otro ámbito antes de consumir la decisión V3.
-- Material V3 ficticio: el ámbito correcto debe llegar hasta AD218 y caer allí.
\set ON_ERROR_STOP on
BEGIN READ ONLY;
DO $estructura$
DECLARE f oid:=pg_catalog.to_regprocedure('vec_bolsa_llamamientos.declarar_ambito_bolsa_v1(text,text,text,text)');
        rol text;
BEGIN
 IF f IS NULL OR pg_catalog.to_regclass('vec_bolsa_llamamientos.ambito_bolsa') IS NULL
 OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_class c
     WHERE c.oid='vec_bolsa_llamamientos.ambito_bolsa'::pg_catalog.regclass
       AND c.relrowsecurity AND c.relforcerowsecurity)
 THEN RAISE EXCEPTION 'B144: estructura ausente' USING ERRCODE='55000'; END IF;
 FOREACH rol IN ARRAY ARRAY['vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos_desarrollo',
   'vec_bolsa_llamamientos_constituidor'] LOOP
  IF pg_catalog.has_function_privilege(rol,f,'EXECUTE')
  OR pg_catalog.has_table_privilege(rol,'vec_bolsa_llamamientos.ambito_bolsa','SELECT,INSERT,UPDATE,DELETE,TRUNCATE')
  THEN RAISE EXCEPTION 'B144: % no debe declarar ni leer el ámbito directamente', rol USING ERRCODE='55000'; END IF;
 END LOOP;
END $estructura$;
ROLLBACK;

BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL TIME ZONE 'UTC';
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
DO $declaracion$
DECLARE r jsonb;
BEGIN
 -- Una bolsa que no existe no recibe ámbito.
 BEGIN
  PERFORM vec_bolsa_llamamientos.declarar_ambito_bolsa_v1('bolsa:f160:no-existe',
   'ambito:desarrollo:bolsa','unidad:desarrollo:rrhh','semilla:prueba:b144');
  RAISE EXCEPTION 'B144: bolsa inexistente declarada' USING ERRCODE='55000';
 EXCEPTION WHEN invalid_parameter_value THEN
  IF SQLERRM NOT LIKE 'B144: bolsa no constituida%' THEN RAISE; END IF;
 END;
 PERFORM vec_bolsa_llamamientos.constituir_bolsa_v1('acta:f160:previa','actor:f160','categoria:f160',
  'bolsa:f160',1,'\x7b7d',now(),'instantanea:f160',1,'\x7b7d',now(),now(),
  '[{"orden":1,"participacion_ref":"participacion:f160:1","fila_numero":1}]',now());
 -- Referencia de declaración que no es una semilla: inválida.
 BEGIN
  PERFORM vec_bolsa_llamamientos.declarar_ambito_bolsa_v1('bolsa:f160',
   'ambito:desarrollo:bolsa','unidad:desarrollo:rrhh','acta:importacion-convoca:x');
  RAISE EXCEPTION 'B144: declaración sin semilla aceptada' USING ERRCODE='55000';
 EXCEPTION WHEN invalid_parameter_value THEN
  IF SQLERRM NOT LIKE 'B144: declaración de ámbito inválida%' THEN RAISE; END IF;
 END;
 r:=vec_bolsa_llamamientos.declarar_ambito_bolsa_v1('bolsa:f160',
  'ambito:desarrollo:bolsa','unidad:desarrollo:rrhh','semilla:prueba:b144');
 IF r->>'reutilizada' IS DISTINCT FROM 'false' OR r->>'origen' IS DISTINCT FROM 'semilla_declarada' THEN
  RAISE EXCEPTION 'B144: primera declaración divergente %', r USING ERRCODE='55000'; END IF;
 r:=vec_bolsa_llamamientos.declarar_ambito_bolsa_v1('bolsa:f160',
  'ambito:desarrollo:bolsa','unidad:desarrollo:rrhh','semilla:prueba:otra');
 IF r->>'reutilizada' IS DISTINCT FROM 'true' THEN
  RAISE EXCEPTION 'B144: repetición no idempotente %', r USING ERRCODE='55000'; END IF;
 -- Otra unidad gestora para la misma bolsa: conflicto y sin escritura.
 BEGIN
  PERFORM vec_bolsa_llamamientos.declarar_ambito_bolsa_v1('bolsa:f160',
   'ambito:desarrollo:bolsa','unidad:desarrollo:otra','semilla:prueba:b144');
  RAISE EXCEPTION 'B144: segunda unidad aceptada' USING ERRCODE='55000';
 EXCEPTION WHEN unique_violation THEN
  IF SQLERRM NOT LIKE 'B144: la bolsa ya tiene otro ámbito%' THEN RAISE; END IF;
 END;
 BEGIN
  UPDATE vec_bolsa_llamamientos.ambito_bolsa SET unidad_ref='unidad:desarrollo:otra' WHERE bolsa_ref='bolsa:f160';
  RAISE EXCEPTION 'B144: ámbito modificable' USING ERRCODE='55000';
 EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL;
 END;
 BEGIN
  DELETE FROM vec_bolsa_llamamientos.ambito_bolsa WHERE bolsa_ref='bolsa:f160';
  RAISE EXCEPTION 'B144: ámbito borrable' USING ERRCODE='55000';
 EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL;
 END;
 IF (SELECT pg_catalog.count(*) FROM vec_bolsa_llamamientos.ambito_bolsa
     WHERE bolsa_ref='bolsa:f160' AND unidad_ref='unidad:desarrollo:rrhh'
       AND fuente_ref='semilla:prueba:b144')<>1 THEN
  RAISE EXCEPTION 'B144: fila declarada divergente' USING ERRCODE='55000'; END IF;
END $declaracion$;
RESET ROLE;

-- La entrada web (rol ejecutor) no puede declarar.
SET LOCAL ROLE vec_bolsa_llamamientos_ejecutor;
DO $ajeno$
BEGIN
 PERFORM vec_bolsa_llamamientos.declarar_ambito_bolsa_v1('bolsa:f160',
  'ambito:desarrollo:bolsa','unidad:desarrollo:rrhh','semilla:prueba:b144');
 RAISE EXCEPTION 'B144: el ejecutor declaró un ámbito' USING ERRCODE='55000';
EXCEPTION WHEN insufficient_privilege THEN NULL;
END $ajeno$;

-- Confirmación de una carga sobre esa bolsa desde otra unidad: 42501 de B144
-- antes de AD218. Desde la misma unidad, el material ficticio cae en AD218.
DO $confirmacion$
DECLARE h text:=pg_catalog.repeat('c',64); cat text:='categoria:f160'; actor text:='per_f160';
 acta text:='acta:importacion-convoca:'||pg_catalog.encode(pg_catalog.sha256(
  pg_catalog.convert_to(pg_catalog.repeat('c',64)||pg_catalog.chr(31)||'categoria:f160','UTF8')),'hex');
 a jsonb; b bytea; filas jsonb:='[{"numero":1}]'; entradas jsonb:='[{"participacion_ref":"p","fila_numero":1}]';
 contexto bytea; c bytea; d bytea; huella text; unidad text;
BEGIN
 a:=pg_catalog.jsonb_build_object('acta_ref',acta,'categoria_ref',cat,'bolsa_ref','bolsa:f160',
  'huella_fichero_sha256',h,'actor_ref','actor:rrhh:'||pg_catalog.substr(pg_catalog.encode(
   pg_catalog.sha256(pg_catalog.convert_to('vec.bolsa.carga_convoca.actor'||pg_catalog.chr(31)||actor,'UTF8')),'hex'),1,32));
 b:=pg_catalog.convert_to(pg_catalog.jsonb_build_object('bolsa_ref','bolsa:f160',
  'categoria_ref',cat,'huella_listado_sha256',h)::text,'UTF8');
 FOREACH unidad IN ARRAY ARRAY['unidad:desarrollo:otra','unidad:desarrollo:rrhh'] LOOP
  contexto:=pg_catalog.convert_to('{"ambitos":{"ambito_ref":"ambito:desarrollo:bolsa","unidad_ref":"'
   ||unidad||'"},"atributos":{}}','UTF8');
  huella:=pg_catalog.encode(pg_catalog.sha256(contexto),'hex');
  c:=pg_catalog.convert_to(pg_catalog.jsonb_build_object('efecto_ref',acta,
   'huella_efecto_sha256',huella,'audiencia_consumo','vec_bolsa_llamamientos.carga_convoca.confirmar.v1')::text,'UTF8');
  d:=pg_catalog.convert_to(pg_catalog.jsonb_build_object('decision_ref','decision:ficticia',
   'principal_id',actor,'recurso_ref',acta,'contexto_recurso_huella_sha256',huella)::text,'UTF8');
  BEGIN
   PERFORM vec_bolsa_llamamientos.confirmar_carga_convoca_v2(a,filas,'{}',acta,actor,cat,'bolsa:f160',
    2,b,now(),'ins',1,'\x7b7d',now(),now(),entradas,now(),'[]',contexto,
    c,d,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
   RAISE EXCEPTION 'B144: confirmación ficticia aceptada' USING ERRCODE='55000';
  EXCEPTION WHEN insufficient_privilege THEN
   IF unidad='unidad:desarrollo:otra' AND SQLERRM NOT LIKE 'B144: bolsa de otro ámbito%' THEN RAISE; END IF;
   IF unidad='unidad:desarrollo:rrhh' AND SQLERRM NOT LIKE 'AD218:%' THEN RAISE; END IF;
  END;
 END LOOP;
END $confirmacion$;
ROLLBACK;
SELECT 'B144 ámbito gobernado OK' AS resultado;
