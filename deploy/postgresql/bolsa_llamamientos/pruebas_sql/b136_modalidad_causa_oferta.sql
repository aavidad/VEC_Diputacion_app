\set ON_ERROR_STOP on
-- Ejecutar después de CT235 y B136 en copia desechable. No fabrica decisiones ni recibos V3.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $test$
DECLARE datos jsonb;origen jsonb;v5 text;v6 text;canon text;v_otro text;
BEGIN
 datos:=jsonb_build_object('categoria','Administracion','centro','Granada',
  'fecha_inicio','2026-10-20','descripcion','Cobertura temporal');
 origen:=jsonb_build_object('organizacion_ref','org:granada','unidad_ref','unidad:rrhh',
  'expediente_ref','expediente:ct235','version_expediente',3,
  'clave_ct','11111111-1111-4111-8111-111111111111',
  'necesidad_ref','necesidad:administracion','acta_ref','acta:ct235',
  'categoria_ref','categoria:administracion','bolsa_ref','bolsa:administracion',
  'version_bolsa',1,'huella_bolsa_sha256',repeat('a',64),
  'instantanea_ref','instantanea:1','version_instantanea',1,
  'huella_instantanea_sha256',repeat('b',64),'politica_ref','politica:1',
  'version_politica',1,'huella_politica_sha256',repeat('c',64),
  'lectura',jsonb_build_object('corte','2026-10-18T08:00:00.000000Z',
    'entradas',jsonb_build_array(jsonb_build_object('orden_vigente',1))));
 v5:=vec_bolsa_llamamientos.huella_operacion_oferta_v5(
  'vec.bolsa.oferta.recuperar.v1','bolsa:administracion','ct-oferta:clave',
  'per_1234567890123456789012',2,datos,origen,NULL,NULL,NULL,NULL);
 IF v5 IS DISTINCT FROM 'b8913711f67cf664cdd459381088eba1021b201ceefa6211b7ac8103d522684c' THEN
  RAISE EXCEPTION 'B136: vector v5 histórico cambió'; END IF;
 datos:=datos||jsonb_build_object('modalidad_clave','sustitucion','causa_clave','baja_temporal');
 v6:=vec_bolsa_llamamientos.huella_operacion_oferta_v6(
  'vec.bolsa.oferta.recuperar.v1','bolsa:administracion','ct-oferta:clave',
  'per_1234567890123456789012',2,datos,origen,NULL,NULL,NULL,NULL);
 SELECT string_agg(octet_length(convert_to(x.valor,'UTF8'))::text||':'||x.valor,'' ORDER BY x.ord)
 INTO canon FROM unnest(ARRAY['vec.bolsa.oferta.canon.v6',v5,'sustitucion','baja_temporal'])
 WITH ORDINALITY x(valor,ord);
 IF v6 IS DISTINCT FROM '2335ca607f0f2650235b9af0680942f9ace8a7f52732b14b7802942c2ab8a840'
    OR v6 IS DISTINCT FROM encode(sha256(convert_to(canon,'UTF8')),'hex') THEN
  RAISE EXCEPTION 'B136: framing v6 distinto del contrato'; END IF;
 v_otro:=vec_bolsa_llamamientos.huella_operacion_oferta_v6(
  'vec.bolsa.oferta.recuperar.v1','bolsa:administracion','ct-oferta:clave',
  'per_1234567890123456789012',2,
  jsonb_set(datos,'{modalidad_clave}','"interinidad"'::jsonb),origen,NULL,NULL,NULL,NULL);
 IF v_otro=v6 THEN RAISE EXCEPTION 'B136: modalidad no liga la huella'; END IF;
 v_otro:=vec_bolsa_llamamientos.huella_operacion_oferta_v6(
  'vec.bolsa.oferta.recuperar.v1','bolsa:administracion','ct-oferta:clave',
  'per_1234567890123456789012',2,
  jsonb_set(datos,'{causa_clave}','"vacante"'::jsonb),origen,NULL,NULL,NULL,NULL);
 IF v_otro=v6 THEN RAISE EXCEPTION 'B136: causa no liga la huella'; END IF;
 BEGIN
  PERFORM vec_bolsa_llamamientos.huella_operacion_oferta_v6(
   'vec.bolsa.oferta.recuperar.v1','bolsa:administracion','ct-oferta:clave',
   'per_1234567890123456789012',2,datos-'causa_clave',origen,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'B136: par parcial admitido';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
 BEGIN
  PERFORM vec_bolsa_llamamientos.huella_operacion_oferta_v6(
   'vec.bolsa.oferta.recuperar.v1','bolsa:administracion','ct-oferta:clave',
   'per_1234567890123456789012',2,
   jsonb_set(datos,'{modalidad_clave}','true'::jsonb),origen,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'B136: modalidad booleana admitida';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
 BEGIN
  PERFORM vec_bolsa_llamamientos.huella_operacion_oferta_v6(
   'vec.bolsa.oferta.recuperar.v1','bolsa:administracion','ct-oferta:clave',
   'per_1234567890123456789012',2,
   jsonb_set(datos,'{causa_clave}','true'::jsonb),origen,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'B136: causa booleana admitida';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
 BEGIN
  PERFORM vec_bolsa_llamamientos.huella_operacion_oferta_v6(
   'vec.bolsa.oferta.recuperar.v1','bolsa:administracion','ct-oferta:clave',
   'per_1234567890123456789012',2,
   jsonb_set(datos,'{modalidad_clave}','null'::jsonb),origen,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'B136: modalidad JSON null admitida';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
 BEGIN
  PERFORM vec_bolsa_llamamientos.huella_operacion_oferta_v6(
   'vec.bolsa.oferta.recuperar.v1','bolsa:administracion','ct-oferta:clave',
   'per_1234567890123456789012',2,
   jsonb_set(datos,'{causa_clave}','null'::jsonb),origen,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'B136: causa JSON null admitida';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
 BEGIN
  PERFORM vec_bolsa_llamamientos.huella_operacion_oferta_v6(
   'vec.bolsa.oferta.recuperar.v1','bolsa:administracion','ct-oferta:clave',
   'per_1234567890123456789012',2,datos,NULL,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'B136: origen ausente admitido';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
 IF EXISTS(SELECT 1 FROM pg_proc p,
   aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) acl
   WHERE p.oid='vec_bolsa_llamamientos.huella_operacion_oferta_v6(text,text,text,text,integer,jsonb,jsonb,jsonb,timestamptz,timestamptz,jsonb)'::regprocedure
     AND acl.grantee=0 AND acl.privilege_type='EXECUTE') THEN
  RAISE EXCEPTION 'B136: helper público'; END IF;
END $test$;
DO $denegaciones$
DECLARE datos jsonb;origen jsonb;
BEGIN
 -- La publicación rechaza el par incompleto, y también el par sin origen CT,
 -- antes de leer ninguna decisión de autorización.
 datos:=jsonb_build_object('categoria','Administracion','centro','Granada',
  'fecha_inicio','2026-10-20','descripcion','Cobertura temporal','modalidad_clave','sustitucion');
 origen:=jsonb_build_object('expediente_ref','expediente:ct235');
 BEGIN
  PERFORM vec_bolsa_llamamientos.publicar_oferta_v5('oferta:b136','recibo:b136','bolsa:administracion',
   'per_1234567890123456789012','ct-oferta:clave',datos,NULL,NULL,NULL,NULL,NULL,NULL,NULL,
   NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,2,origen);
  RAISE EXCEPTION 'B136: publicación con par incompleto admitida';
 EXCEPTION WHEN SQLSTATE '22023' THEN
  IF SQLERRM<>'B136: modalidad y causa incompletas' THEN RAISE; END IF;
 END;
 datos:=datos||jsonb_build_object('causa_clave','baja_temporal');
 BEGIN
  PERFORM vec_bolsa_llamamientos.publicar_oferta_v5('oferta:b136','recibo:b136','bolsa:administracion',
   'per_1234567890123456789012','ct-oferta:clave',datos,NULL,NULL,NULL,NULL,NULL,NULL,NULL,
   NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,2,NULL);
  RAISE EXCEPTION 'B136: oferta nativa con modalidad admitida';
 EXCEPTION WHEN SQLSTATE '22023' THEN
  IF SQLERRM<>'B136: modalidad y causa incompletas' THEN RAISE; END IF;
 END;
 -- La recuperación solo la ejecuta el LOGIN de Bolsa en serializable.
 BEGIN
  PERFORM vec_bolsa_llamamientos.recuperar_oferta_operacion_v1('bolsa:administracion',
   'per_1234567890123456789012','ct-oferta:clave',datos,2,origen,NULL,NULL,NULL,NULL,NULL,
   NULL,NULL,NULL,NULL,NULL,NULL,'unidad:rrhh','ambito:rrhh');
  RAISE EXCEPTION 'B136: recuperación fuera de su LOGIN admitida';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL;
 END;
 IF has_function_privilege('vec_contratacion_temporal_ejecutor',
   'vec_bolsa_llamamientos.publicar_oferta_v5(text,text,text,text,text,jsonb,jsonb,timestamptz,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text,integer,jsonb)','EXECUTE')
  OR has_function_privilege('vec_bolsa_llamamientos_ejecutor',
   'vec_bolsa_llamamientos.huella_operacion_oferta_v6(text,text,text,text,integer,jsonb,jsonb,jsonb,timestamptz,timestamptz,jsonb)','EXECUTE')
  OR NOT has_function_privilege('vec_bolsa_llamamientos_ejecutor',
   'vec_bolsa_llamamientos.recuperar_oferta_operacion_v1(text,text,text,jsonb,integer,jsonb,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,text)','EXECUTE') THEN
  RAISE EXCEPTION 'B136: permisos de publicación distintos de lo esperado'; END IF;
END $denegaciones$;
ROLLBACK;

-- Publicación y recuperación de una oferta de CT con el par, sobre un
-- expediente real de la copia. Como LOGIN de Bolsa y en serializable, igual
-- que el servidor. Todo se revierte. Dentro de la transacción, y solo en ella:
-- el consumo V3 y el validador de origen son dobles (el material firmado y la
-- lectura de Bolsa se comprueban en el recorrido), la política de ofertas
-- recibe una versión mínima sin sus CHECK de forma y las llamadas corren con
-- session_replication_role=replica: así no saltan los disparadores de Bolsa
-- que cotejan plazo y aceptación con la política (B47, B75), que no son de
-- F1-11. Las ramas que se prueban están en el cuerpo de las funciones.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT r.rolname AS login_bolsa FROM pg_roles r
WHERE r.rolcanlogin AND NOT r.rolsuper
  AND pg_has_role(r.oid,'vec_bolsa_llamamientos_ejecutor','MEMBER')
  AND NOT pg_has_role(r.oid,'vec_bolsa_llamamientos_propietario','MEMBER')
ORDER BY r.rolname LIMIT 1 \gset

CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_emision_llamamiento_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,
 p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,
 auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql AS $doble$
 SELECT 'decision:doble-f111',d->>'recurso_ref',d->>'contexto_recurso_huella_sha256',repeat('e',64),
  'auditoria:doble-f111',clock_timestamp(),true
 FROM (SELECT convert_from(p_decision,'UTF8')::jsonb d) s $doble$;
CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.validar_origen_oferta_ct_v1(
 p_origen jsonb,p_bolsa text,p_numero integer,p_publicada timestamptz)
RETURNS integer LANGUAGE sql AS $doble$ SELECT 5 $doble$;

CREATE TEMP TABLE caso(k text PRIMARY KEY,v jsonb) ON COMMIT DROP;
GRANT SELECT ON caso TO PUBLIC;
DO $sin_check$ DECLARE r record; BEGIN
 FOR r IN SELECT conname FROM pg_constraint
  WHERE conrelid='vec_bolsa_llamamientos.politica_ofertas_version'::regclass AND contype='c' LOOP
  EXECUTE format('ALTER TABLE vec_bolsa_llamamientos.politica_ofertas_version DROP CONSTRAINT %I',r.conname);
 END LOOP;
END $sin_check$;
DO $prep$
DECLARE x record;bolsa text:='bolsa:administrativo:2026-09-17';actor text:='per_f111pruebamodalidadcau';
 publicada timestamptz:=date_trunc('microseconds',clock_timestamp());plazo jsonb;origen jsonb;datos jsonb;
BEGIN
 SET LOCAL session_replication_role=replica;
 INSERT INTO vec_bolsa_llamamientos.politica_ofertas_version VALUES(bolsa,99,'{"plazas":{}}','0'||repeat('a',63),
  true,actor,'clave-f111-politica',98,'recibo:f111',publicada,'decision:f111','auditoria:f111');
 SET LOCAL session_replication_role=origin;
 SELECT a.expediente_ref,a.version::bigint AS version,e.organizacion_ref,v.agregado_json AS g INTO x
 FROM vec_contratacion_temporal.expediente_integral_actual a
 JOIN vec_contratacion_temporal.expediente_alta e USING(expediente_ref)
 JOIN vec_contratacion_temporal.expediente_version_integral v
   ON v.expediente_ref=a.expediente_ref AND v.version=a.version
 WHERE v.agregado_json#>>'{analisis,modalidad_clave}' ~ '^[a-z][a-z0-9._-]{1,79}$'
   AND v.agregado_json#>>'{analisis,causa_clave}' ~ '^[a-z][a-z0-9._-]{1,79}$'
 ORDER BY a.expediente_ref LIMIT 1;
 IF NOT FOUND THEN RAISE EXCEPTION 'F1-11: la copia no tiene un expediente con modalidad y causa'; END IF;
 plazo:=jsonb_build_object('inicio','publicacion','regla_ref','regla:f111','huella_catalogo',repeat('b',64),
  'unidad','dias_habiles','cantidad',3,'computo','x','ultimo_dia','x','ejemplo','x',
  'calendarios','["cal:granada"]'::jsonb,'politica_version','99','municipio_sede','granada');
 origen:=jsonb_build_object('organizacion_ref',x.organizacion_ref,'unidad_ref','unidad:f111',
  'expediente_ref',x.expediente_ref,'version_expediente',x.version,
  'clave_ct','44444444-4444-4444-8444-444444444444','necesidad_ref','necesidad:f111',
  'acta_ref','acta:f111','categoria_ref',x.g#>>'{analisis,categoria_ref}','bolsa_ref',bolsa,
  'version_bolsa',1,'huella_bolsa_sha256',repeat('a',64),'instantanea_ref','instantanea:f111',
  'version_instantanea',1,'huella_instantanea_sha256',repeat('b',64),'politica_ref','politica:f111',
  'version_politica',1,'huella_politica_sha256',repeat('c',64),
  'lectura',jsonb_build_object('corte','2026-10-19T08:00:00.000000Z',
   'entradas',jsonb_build_array(jsonb_build_object('orden_vigente',1))));
 datos:=jsonb_build_object('categoria','Administrativo','centro','Granada','fecha_inicio','2026-11-02',
  'descripcion','Cobertura temporal','modalidad_clave',x.g#>>'{analisis,modalidad_clave}',
  'causa_clave',x.g#>>'{analisis,causa_clave}');
 INSERT INTO caso VALUES('b',to_jsonb(bolsa)),('a',to_jsonb(actor)),
  ('c',to_jsonb('ct-oferta:'||repeat('7',64))),('p',to_jsonb(publicada)),
  ('v',to_jsonb(publicada+interval '3 days')),('pl',plazo),('o',origen),('d',datos),
  ('u','"unidad:f111"'),('m','"ambito:f111"');
END $prep$;

-- Decisión ligada al material exacto: huella v6 si los datos llevan el par, v5 si no.
CREATE FUNCTION pg_temp.decision(p_prefijo text,p_datos jsonb,p_publicada timestamptz,p_vence timestamptz,p_plazo jsonb)
RETURNS bytea LANGUAGE plpgsql AS $f$
DECLARE b text;a text;c text;o jsonb;u text;m text;h text;
BEGIN
 SELECT v#>>'{}' INTO b FROM caso WHERE k='b'; SELECT v#>>'{}' INTO a FROM caso WHERE k='a';
 SELECT v#>>'{}' INTO c FROM caso WHERE k='c'; SELECT v INTO o FROM caso WHERE k='o';
 SELECT v#>>'{}' INTO u FROM caso WHERE k='u'; SELECT v#>>'{}' INTO m FROM caso WHERE k='m';
 IF p_datos ? 'modalidad_clave' THEN
  h:=vec_bolsa_llamamientos.huella_operacion_oferta_v6(p_prefijo,b,c,a,2,p_datos,o,NULL,p_publicada,p_vence,p_plazo);
 ELSE
  h:=vec_bolsa_llamamientos.huella_operacion_oferta_v5(p_prefijo,b,c,a,2,p_datos,o,NULL,p_publicada,p_vence,p_plazo);
 END IF;
 RETURN convert_to(jsonb_build_object('contexto_recurso_huella_sha256',encode(sha256(convert_to(
  '{"ambitos":{"ambito_ref":"'||m||'","unidad_ref":"'||u||'"},"atributos":{"material_sha256":"'||h||'"}}',
  'UTF8')),'hex'),'principal_id',a,'accion','llamamiento.emitir.v1','modulo_id','bolsa',
  'finalidad','gestion_llamamientos_bolsa','recurso_ref',b,'tipo_recurso','bolsa_constituida')::text,'UTF8');
END $f$;
-- Las huellas solo las ejecuta el propietario de Bolsa: se precalculan aquí.
CREATE TEMP TABLE dec(k text PRIMARY KEY,d bytea) ON COMMIT DROP;
GRANT SELECT ON dec TO PUBLIC;
WITH t AS (SELECT (SELECT v FROM caso WHERE k='d') d,
  (SELECT (v#>>'{}')::timestamptz FROM caso WHERE k='p') p,
  (SELECT (v#>>'{}')::timestamptz FROM caso WHERE k='v') ve,(SELECT v FROM caso WHERE k='pl') pl)
INSERT INTO dec
SELECT 'ok',pg_temp.decision('vec.bolsa.oferta.publicar.v1',d,p,ve,pl) FROM t UNION ALL
SELECT 'otra',pg_temp.decision('vec.bolsa.oferta.publicar.v1',jsonb_set(d,'{causa_clave}','"otra_causa_f111"'),p,ve,pl) FROM t UNION ALL
SELECT 'sinpar',pg_temp.decision('vec.bolsa.oferta.publicar.v1',d-'modalidad_clave'-'causa_clave',p,ve,pl) FROM t UNION ALL
SELECT 'rec',pg_temp.decision('vec.bolsa.oferta.recuperar.v1',d,NULL,NULL,NULL) FROM t UNION ALL
SELECT 'recv5',pg_temp.decision('vec.bolsa.oferta.recuperar.v1',d-'modalidad_clave'-'causa_clave',NULL,NULL,NULL) FROM t;

SET LOCAL session_replication_role=replica;
SET SESSION AUTHORIZATION :"login_bolsa";
DO $publicacion$
DECLARE b text;a text;c text;p timestamptz;ve timestamptz;pl jsonb;o jsonb;d jsonb;u text;m text;
 r record;rec text;ofe text;
BEGIN
 SELECT v#>>'{}' INTO b FROM caso WHERE k='b'; SELECT v#>>'{}' INTO a FROM caso WHERE k='a';
 SELECT v#>>'{}' INTO c FROM caso WHERE k='c'; SELECT (v#>>'{}')::timestamptz INTO p FROM caso WHERE k='p';
 SELECT (v#>>'{}')::timestamptz INTO ve FROM caso WHERE k='v'; SELECT v INTO pl FROM caso WHERE k='pl';
 SELECT v INTO o FROM caso WHERE k='o'; SELECT v INTO d FROM caso WHERE k='d';
 SELECT v#>>'{}' INTO u FROM caso WHERE k='u'; SELECT v#>>'{}' INTO m FROM caso WHERE k='m';
 rec:='recibo:oferta:'||encode(sha256(convert_to(b||chr(31)||c,'UTF8')),'hex');
 ofe:=replace(rec,'recibo:oferta:','oferta:');
 -- 1. Una oferta de CT nueva sin el par se rechaza, aunque la decisión cuadre.
 BEGIN
  PERFORM vec_bolsa_llamamientos.publicar_oferta_v5(ofe,rec,b,a,c,d-'modalidad_clave'-'causa_clave',pl,p,ve,
   NULL,(SELECT dec.d FROM dec WHERE k='sinpar'),NULL,NULL,1,1,NULL,NULL,NULL,NULL,u,m,2,o);
  RAISE EXCEPTION 'F1-11: oferta de CT nueva sin el par admitida';
 EXCEPTION WHEN SQLSTATE '22023' THEN
  IF SQLERRM<>'B136: nueva oferta CT sin modalidad y causa' THEN RAISE; END IF;
 END;
 -- 2. Un par distinto del análisis vigente se deniega en el cotejo con CT221.
 BEGIN
  PERFORM vec_bolsa_llamamientos.publicar_oferta_v5(ofe,rec,b,a,c,jsonb_set(d,'{causa_clave}','"otra_causa_f111"'),
   pl,p,ve,NULL,(SELECT dec.d FROM dec WHERE k='otra'),NULL,NULL,1,1,NULL,NULL,NULL,NULL,u,m,2,o);
  RAISE EXCEPTION 'F1-11: par distinto del análisis admitido';
 EXCEPTION WHEN SQLSTATE '42501' THEN
  IF SQLERRM<>'B136: modalidad o causa distinta del análisis vigente' THEN RAISE; END IF;
 END;
 -- 3. Una decisión calculada sin el par (huella v5) no autoriza datos con el par.
 BEGIN
  PERFORM vec_bolsa_llamamientos.publicar_oferta_v5(ofe,rec,b,a,c,d,pl,p,ve,
   NULL,(SELECT dec.d FROM dec WHERE k='sinpar'),NULL,NULL,1,1,NULL,NULL,NULL,NULL,u,m,2,o);
  RAISE EXCEPTION 'F1-11: decisión v5 admitida para datos con el par';
 EXCEPTION WHEN SQLSTATE '42501' THEN
  IF SQLERRM<>'B117: decisión no ligada a la publicación' THEN RAISE; END IF;
 END;
 -- 4. Con el par del análisis publica, y el reintento reutiliza la misma oferta.
 SELECT * INTO r FROM vec_bolsa_llamamientos.publicar_oferta_v5(ofe,rec,b,a,c,d,pl,p,ve,
  NULL,(SELECT dec.d FROM dec WHERE k='ok'),NULL,NULL,1,1,NULL,NULL,NULL,NULL,u,m,2,o);
 IF r.reutilizada IS NOT FALSE THEN RAISE EXCEPTION 'F1-11: publicación no nueva: %',r; END IF;
 SELECT * INTO r FROM vec_bolsa_llamamientos.publicar_oferta_v5(ofe,rec,b,a,c,d,pl,p,ve,
  NULL,(SELECT dec.d FROM dec WHERE k='ok'),NULL,NULL,1,1,NULL,NULL,NULL,NULL,u,m,2,o);
 IF r.reutilizada IS NOT TRUE THEN RAISE EXCEPTION 'F1-11: el reintento no reutilizó la oferta'; END IF;
 -- 5. La recuperación exige la huella v6: una decisión sin el par se deniega.
 BEGIN
  PERFORM vec_bolsa_llamamientos.recuperar_oferta_operacion_v1(b,a,c,d,2,o,NULL,
   NULL,(SELECT dec.d FROM dec WHERE k='recv5'),NULL,NULL,1,1,NULL,NULL,NULL,NULL,u,m);
  RAISE EXCEPTION 'F1-11: recuperación con huella v5 admitida';
 EXCEPTION WHEN SQLSTATE '42501' THEN
  IF SQLERRM<>'B117: recuperación no ligada a la intención' THEN RAISE; END IF;
 END;
 -- 6. Con la huella v6 la encuentra; con otra causa no.
 SELECT * INTO r FROM vec_bolsa_llamamientos.recuperar_oferta_operacion_v1(b,a,c,d,2,o,NULL,
  NULL,(SELECT dec.d FROM dec WHERE k='rec'),NULL,NULL,1,1,NULL,NULL,NULL,NULL,u,m);
 IF r.encontrada IS NOT TRUE THEN RAISE EXCEPTION 'F1-11: recuperación v6 no encontrada'; END IF;
 BEGIN
  SELECT * INTO r FROM vec_bolsa_llamamientos.recuperar_oferta_operacion_v1(b,a,c,
   jsonb_set(d,'{causa_clave}','"otra_causa_f111"'),2,o,NULL,
   NULL,(SELECT dec.d FROM dec WHERE k='rec'),NULL,NULL,1,1,NULL,NULL,NULL,NULL,u,m);
  RAISE EXCEPTION 'F1-11: recuperación con otra causa admitida: %',r;
 EXCEPTION WHEN SQLSTATE '42501' THEN
  IF SQLERRM<>'B117: recuperación no ligada a la intención' THEN RAISE; END IF;
 END;
END $publicacion$;
RESET SESSION AUTHORIZATION;
SET LOCAL session_replication_role=origin;
DO $fila$
DECLARE n integer;
BEGIN
 SELECT count(*) INTO n FROM vec_bolsa_llamamientos.oferta_publicada x
 JOIN vec_bolsa_llamamientos.origen_oferta_ct_v1 o ON o.oferta_ref=x.oferta_ref
 WHERE x.clave_idempotencia=(SELECT v#>>'{}' FROM caso WHERE k='c')
   AND x.datos->>'modalidad_clave'=(SELECT v->>'modalidad_clave' FROM caso WHERE k='d')
   AND x.datos->>'causa_clave'=(SELECT v->>'causa_clave' FROM caso WHERE k='d');
 IF n<>1 THEN RAISE EXCEPTION 'F1-11: oferta publicada con el par esperada 1, hay %',n; END IF;
 RAISE NOTICE 'F1-11: publicación y recuperación de oferta de CT con el par superadas';
END $fila$;
ROLLBACK;
