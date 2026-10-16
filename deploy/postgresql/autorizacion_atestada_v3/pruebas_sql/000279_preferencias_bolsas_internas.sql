\set ON_ERROR_STOP on
-- Prueba focal de AD279 en PostgreSQL 18 desechable, tras la lista
-- AD279 → AUT96 → B116 → AD275 (deploy/principal/lista_sql_claude_preferencias_internas_20261016.txt).
-- Se ejecuta como superusuario de la copia; todo termina en ROLLBACK.
-- Cubre: huellas y ACL posteriores, denegación de toda sesión que no sea el
-- LOGIN nominal, paso de la compuerta para ese LOGIN y, con el consumidor V3
-- sustituido dentro de la transacción, recibo, idempotencia y conflictos de
-- B116. La firma V3 real se ensaya desde la aplicación, no aquí.

-- 1. Postimagen exacta: núcleo, ayudante, restricciones, ACL y RLS.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $estructura$
DECLARE nucleo oid := 'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 interno oid := to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_preferencias_bolsas_interna_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 definicion text; audiencia text; lectura text;
BEGIN
 IF interno IS NULL THEN RAISE EXCEPTION 'AD279: consumidor nominal ausente' USING ERRCODE='55000'; END IF;
 definicion:=pg_get_functiondef(nucleo);
 -- La huella fija todas las ramas previas (O, R y S); las búsquedas solo
 -- explican qué se conserva si la huella cambia en una sucesora.
 IF encode(sha256(convert_to(definicion,'UTF8')),'hex') IS DISTINCT FROM
    'a46491328a00b33d5d9276c9fa6b4eec6287ba015d865df14aed807fe2d58fb2'
    OR strpos(definicion,'documentos_bolsa_externa')=0
    OR strpos(definicion,'documentos_bolsa_rrhh')=0
    OR strpos(definicion,'meritos_cartera_retirada')=0
    OR strpos(definicion,'cierre_intencion_oferta_ct_bolsa')=0
    OR strpos(definicion,'politica_transiciones_bolsa')=0
    OR strpos(definicion,'p_perfil_mutacion IS NOT DISTINCT FROM ''preferencias_bolsas_interna''')=0
    OR strpos(definicion,'AND p_perfil_mutacion IS DISTINCT FROM ''preferencias_bolsas_interna''')=0
 THEN RAISE EXCEPTION 'AD279: núcleo divergente' USING ERRCODE='55000'; END IF;
 IF (SELECT p.proacl::text FROM pg_proc p WHERE p.oid=nucleo) IS DISTINCT FROM
    '{vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario}'
    OR (SELECT p.proacl::text FROM pg_proc p WHERE p.oid=interno) IS DISTINCT FROM
    '{vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario,vec_bolsa_llamamientos_propietario=X/vec_autorizacion_atestada_v3_propietario}'
    OR EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid IN (nucleo,interno)
      AND (p.proowner<>'vec_autorizacion_atestada_v3_propietario'::regrole OR NOT p.prosecdef
       OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']))
    OR EXISTS (SELECT 1 FROM pg_class r WHERE r.oid IN (
      'vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass,
      'vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass)
      AND (r.relowner<>'vec_autorizacion_atestada_v3_propietario'::regrole
       OR NOT r.relrowsecurity OR NOT r.relforcerowsecurity OR r.relacl IS NOT NULL))
 THEN RAISE EXCEPTION 'AD279: propietario, ACL o RLS divergentes' USING ERRCODE='55000'; END IF;
 SELECT pg_get_constraintdef(c.oid,true) INTO STRICT audiencia FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
  AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 SELECT pg_get_constraintdef(c.oid,false) INTO STRICT lectura FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
  AND c.conname='auditoria_lectura_preferencias_bolsas_formato_v2' AND c.contype='c' AND c.convalidated;
 IF encode(sha256(convert_to(audiencia,'UTF8')),'hex') IS DISTINCT FROM
    '504fd62c39258745ea34f9a7033e8993a2b44d78feed7eab2e3ffa01f0a94b15'
    OR encode(sha256(convert_to(lectura,'UTF8')),'hex') IS DISTINCT FROM
    '87b134b21a26da7c26b89d2125305dd24c96e742a3aced1dbb9e68e94cf7aa9e'
    OR EXISTS(SELECT 1 FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
      AND c.conname='auditoria_lectura_preferencias_bolsas_formato_v1')
 THEN RAISE EXCEPTION 'AD279: restricción de audiencias o lectura divergente' USING ERRCODE='55000'; END IF;
END $estructura$;
ROLLBACK;

-- 2. Denegación: una sesión que no es el LOGIN nominal no pasa la compuerta
-- del núcleo ni la del ayudante, antes de leer ningún material.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
DO $denegacion$
DECLARE fallo boolean;
BEGIN
 fallo:=false;
 BEGIN
  PERFORM * FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
   'preferencias_bolsas_interna','\x00','\x00','\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
 EXCEPTION WHEN insufficient_privilege THEN fallo:=true; END;
 IF NOT fallo THEN RAISE EXCEPTION 'AD279: núcleo admitió una sesión no nominal' USING ERRCODE='55000'; END IF;
 fallo:=false;
 BEGIN
  PERFORM * FROM vec_autorizacion_atestada_v3.registrar_y_consumir_preferencias_bolsas_interna_v3_atestada(
   NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
 EXCEPTION WHEN insufficient_privilege THEN fallo:=true; END;
 IF NOT fallo THEN RAISE EXCEPTION 'AD279: ayudante admitió una sesión no nominal' USING ERRCODE='55000'; END IF;
END $denegacion$;
ROLLBACK;

-- 3. LOGIN nominal de prueba (o el de la copia si ya está activado) y un
-- LOGIN sin permiso. Material con la forma que exigen B116 y el ayudante.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';
SET LOCAL idle_in_transaction_session_timeout='10s';
DO $logins$
BEGIN
 IF to_regrole('vec_interno_bolsa_preferencias_desarrollo') IS NULL THEN
  CREATE ROLE vec_interno_bolsa_preferencias_desarrollo LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
  GRANT vec_bolsa_llamamientos_preferencias_interna TO vec_interno_bolsa_preferencias_desarrollo
   WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
 END IF;
 CREATE ROLE vec_preferencias_sin_permiso_prueba LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
END $logins$;
CREATE TEMP TABLE prueba_ad279(candidato text, bolsas text[], c bytea, d bytea, x bytea) ON COMMIT DROP;
GRANT SELECT ON prueba_ad279 TO PUBLIC;
DO $material$
DECLARE v_candidato text; v_bolsas text[];
BEGIN
 SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
 SELECT vc.candidato_ref INTO v_candidato
  FROM (SELECT DISTINCT candidato_ref FROM vec_bolsa_llamamientos.vinculo_candidato
         WHERE candidato_ref ~ '^can_[A-Za-z0-9_-]{22,128}$') vc
  WHERE cardinality(vec_bolsa_llamamientos.preferencias_bolsas_actuales_interna_v1(vc.candidato_ref,clock_timestamp()))>0
  ORDER BY 1 LIMIT 1;
 IF v_candidato IS NULL THEN RAISE EXCEPTION 'AD279: la copia no tiene candidaturas vigentes' USING ERRCODE='55000'; END IF;
 v_bolsas:=vec_bolsa_llamamientos.preferencias_bolsas_actuales_interna_v1(v_candidato,clock_timestamp());
 RESET ROLE;
 INSERT INTO prueba_ad279 VALUES(v_candidato,v_bolsas,
  convert_to(jsonb_build_object('operacion','bolsa.preferencias_bolsas.ordenar_propias',
   'audiencia_consumo','vec.bolsa.preferencias-bolsas.ordenar.interna.v1',
   'efecto_ref','mi-bolsa:'||v_candidato,'huella_efecto_sha256',repeat('a',64))::text,'UTF8'),
  convert_to(jsonb_build_object('accion','bolsa.preferencias_bolsas.ordenar_propias','modulo_id','bolsa',
   'tipo_recurso','participaciones_candidato','finalidad','gestion_preferencias_bolsas_propias',
   'recurso_ref','mi-bolsa:'||v_candidato,'contexto_recurso_huella_sha256',repeat('a',64),
   'principal_id','per_prueba_ad279','perfil_activo_ref','prf_prueba_ad279',
   'vinculo_autenticacion_actor',jsonb_build_object('superficie','interna_corporativa'),
   'campos_permitidos','[]'::jsonb,'obligaciones','[]'::jsonb)::text,'UTF8'),
  convert_to(jsonb_build_object('principal_ref','per_prueba_ad279','perfil_activo_ref','prf_prueba_ad279',
   'vinculos',jsonb_build_array(jsonb_build_object('tipo','candidato','estado','activo','referencia',v_candidato)))::text,'UTF8'));
END $material$;

-- 3a. Sin permiso: un LOGIN fuera del grupo no puede ejecutar la fachada.
SET SESSION AUTHORIZATION vec_preferencias_sin_permiso_prueba;
DO $sin_permiso$
DECLARE p prueba_ad279%ROWTYPE; fallo boolean:=false;
BEGIN
 SELECT * INTO STRICT p FROM prueba_ad279;
 BEGIN
  PERFORM * FROM vec_bolsa_llamamientos.ordenar_preferencias_bolsas_interna_v1(p.candidato,p.bolsas,0,
   'clave-prueba-ad279',clock_timestamp(),p.c,p.d,'\x01',p.x,1,1,'\x01','\x01','\x01','\x01');
 EXCEPTION WHEN insufficient_privilege THEN fallo:=true; END;
 IF NOT fallo THEN RAISE EXCEPTION 'AD279: LOGIN sin permiso ejecutó la fachada' USING ERRCODE='55000'; END IF;
END $sin_permiso$;
RESET SESSION AUTHORIZATION;

-- 3b. Positiva de compuerta: el LOGIN nominal atraviesa B116, el ayudante y
-- la compuerta de sesión del núcleo; el material falso se rechaza después
-- como entrada inválida (22023), nunca como sesión rechazada (42501).
SET SESSION AUTHORIZATION vec_interno_bolsa_preferencias_desarrollo;
DO $compuerta$
DECLARE p prueba_ad279%ROWTYPE; estado text; mensaje text;
BEGIN
 SELECT * INTO STRICT p FROM prueba_ad279;
 BEGIN
  PERFORM * FROM vec_bolsa_llamamientos.ordenar_preferencias_bolsas_interna_v1(p.candidato,p.bolsas,0,
   'clave-prueba-ad279',clock_timestamp(),p.c,p.d,'\x01',p.x,1,1,'\x01','\x01','\x01','\x01');
  RAISE EXCEPTION 'AD279: material falso aceptado' USING ERRCODE='55000';
 EXCEPTION WHEN invalid_parameter_value OR insufficient_privilege THEN
  GET STACKED DIAGNOSTICS estado=RETURNED_SQLSTATE, mensaje=MESSAGE_TEXT;
 END;
 IF estado IS DISTINCT FROM '22023' OR strpos(mensaje,'VEC-AD-3')=0 THEN
  RAISE EXCEPTION 'AD279: compuerta nominal no superada (%: %)',estado,mensaje USING ERRCODE='55000';
 END IF;
END $compuerta$;
RESET SESSION AUTHORIZATION;

-- 3c. El mismo nombre con una segunda membresía ya no es el LOGIN nominal:
-- la compuerta del núcleo lo rechaza como sesión (42501).
GRANT vec_preferencias_sin_permiso_prueba TO vec_interno_bolsa_preferencias_desarrollo
 WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
SET SESSION AUTHORIZATION vec_interno_bolsa_preferencias_desarrollo;
DO $exclusiva$
DECLARE p prueba_ad279%ROWTYPE; estado text; mensaje text;
BEGIN
 SELECT * INTO STRICT p FROM prueba_ad279;
 BEGIN
  PERFORM * FROM vec_bolsa_llamamientos.ordenar_preferencias_bolsas_interna_v1(p.candidato,p.bolsas,0,
   'clave-prueba-ad279',clock_timestamp(),p.c,p.d,'\x01',p.x,1,1,'\x01','\x01','\x01','\x01');
  RAISE EXCEPTION 'AD279: material falso aceptado' USING ERRCODE='55000';
 EXCEPTION WHEN invalid_parameter_value OR insufficient_privilege THEN
  GET STACKED DIAGNOSTICS estado=RETURNED_SQLSTATE, mensaje=MESSAGE_TEXT;
 END;
 IF estado IS DISTINCT FROM '42501' OR mensaje IS DISTINCT FROM 'consumo VEC-AD-3 rechazado' THEN
  RAISE EXCEPTION 'AD279: LOGIN con dos membresías admitido (%: %)',estado,mensaje USING ERRCODE='55000';
 END IF;
END $exclusiva$;
RESET SESSION AUTHORIZATION;
ROLLBACK;

-- 4. Recibo e idempotencia de B116 sobre la historia real. Solo dentro de
-- esta transacción, el ayudante V3 se sustituye por un doble que devuelve un
-- consumo nuevo; la firma no forma parte de esta comprobación.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';
DO $logins$
BEGIN
 IF to_regrole('vec_interno_bolsa_preferencias_desarrollo') IS NULL THEN
  CREATE ROLE vec_interno_bolsa_preferencias_desarrollo LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
  GRANT vec_bolsa_llamamientos_preferencias_interna TO vec_interno_bolsa_preferencias_desarrollo
   WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
 END IF;
END $logins$;
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_preferencias_bolsas_interna_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,
 auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $doble$
 SELECT 'decision:prueba-ad279:'||md5(p_motivo::text), convert_from(p_capacidad,'UTF8')::jsonb->>'efecto_ref',
  repeat('a',64), repeat('b',64), 'auditoria:prueba-ad279:'||md5(p_motivo::text), clock_timestamp(), true
$doble$;
CREATE TEMP TABLE prueba_ad279(candidato text, bolsas text[], c bytea, d bytea, x bytea) ON COMMIT DROP;
GRANT SELECT ON prueba_ad279 TO PUBLIC;
DO $material$
DECLARE v_candidato text; v_bolsas text[];
BEGIN
 SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
 SELECT vc.candidato_ref INTO v_candidato
  FROM (SELECT DISTINCT candidato_ref FROM vec_bolsa_llamamientos.vinculo_candidato
         WHERE candidato_ref ~ '^can_[A-Za-z0-9_-]{22,128}$') vc
  WHERE cardinality(vec_bolsa_llamamientos.preferencias_bolsas_actuales_interna_v1(vc.candidato_ref,clock_timestamp()))>0
    AND NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.preferencia_bolsas_version p WHERE p.candidato_ref=vc.candidato_ref)
  ORDER BY 1 LIMIT 1;
 IF v_candidato IS NULL THEN RAISE EXCEPTION 'AD279: la copia no tiene candidaturas libres' USING ERRCODE='55000'; END IF;
 v_bolsas:=vec_bolsa_llamamientos.preferencias_bolsas_actuales_interna_v1(v_candidato,clock_timestamp());
 RESET ROLE;
 INSERT INTO prueba_ad279 VALUES(v_candidato,v_bolsas,
  convert_to(jsonb_build_object('operacion','bolsa.preferencias_bolsas.ordenar_propias',
   'audiencia_consumo','vec.bolsa.preferencias-bolsas.ordenar.interna.v1',
   'efecto_ref','mi-bolsa:'||v_candidato,'huella_efecto_sha256',repeat('a',64))::text,'UTF8'),
  convert_to(jsonb_build_object('accion','bolsa.preferencias_bolsas.ordenar_propias','modulo_id','bolsa',
   'tipo_recurso','participaciones_candidato','finalidad','gestion_preferencias_bolsas_propias',
   'recurso_ref','mi-bolsa:'||v_candidato,'contexto_recurso_huella_sha256',repeat('a',64),
   'principal_id','per_prueba_ad279','perfil_activo_ref','prf_prueba_ad279',
   'campos_permitidos','[]'::jsonb,'obligaciones','[]'::jsonb)::text,'UTF8'),
  convert_to(jsonb_build_object('principal_ref','per_prueba_ad279','perfil_activo_ref','prf_prueba_ad279',
   'vinculos',jsonb_build_array(jsonb_build_object('tipo','candidato','estado','activo','referencia',v_candidato)))::text,'UTF8'));
END $material$;
SET SESSION AUTHORIZATION vec_interno_bolsa_preferencias_desarrollo;
DO $idempotencia$
DECLARE p prueba_ad279%ROWTYPE; r1 record; r2 record; estado text;
BEGIN
 SELECT * INTO STRICT p FROM prueba_ad279;
 SELECT * INTO STRICT r1 FROM vec_bolsa_llamamientos.ordenar_preferencias_bolsas_interna_v1(p.candidato,p.bolsas,0,
  'clave-prueba-ad279',clock_timestamp(),p.c,p.d,'\x01',p.x,1,1,'\x01','\x01','\x01','\x01');
 IF r1.reutilizada OR r1.version<>1 OR r1.orden IS DISTINCT FROM p.bolsas
    OR r1.recibo_ref !~ '^recibo:preferencias-bolsas:[0-9a-f]{64}$'
 THEN RAISE EXCEPTION 'AD279: primera orden sin recibo nuevo' USING ERRCODE='55000'; END IF;
 -- La misma clave y el mismo orden devuelven el recibo y la fecha originales.
 SELECT * INTO STRICT r2 FROM vec_bolsa_llamamientos.ordenar_preferencias_bolsas_interna_v1(p.candidato,p.bolsas,0,
  'clave-prueba-ad279',clock_timestamp(),p.c,p.d,'\x02',p.x,1,1,'\x01','\x01','\x01','\x01');
 IF NOT r2.reutilizada OR r2.recibo_ref IS DISTINCT FROM r1.recibo_ref OR r2.version<>1
    OR r2.registrada_en IS DISTINCT FROM r1.registrada_en
 THEN RAISE EXCEPTION 'AD279: la repetición no devolvió el recibo original' USING ERRCODE='55000'; END IF;
 -- La misma clave con otra versión esperada es otro orden: conflicto, sin fila nueva.
 BEGIN
  PERFORM * FROM vec_bolsa_llamamientos.ordenar_preferencias_bolsas_interna_v1(p.candidato,p.bolsas,1,
   'clave-prueba-ad279',clock_timestamp(),p.c,p.d,'\x03',p.x,1,1,'\x01','\x01','\x01','\x01');
  RAISE EXCEPTION 'AD279: clave reutilizada aceptada' USING ERRCODE='55000';
 EXCEPTION WHEN sqlstate 'VBP02' THEN NULL; END;
 -- Otra clave con una versión ya superada se rechaza sin escribir.
 BEGIN
  PERFORM * FROM vec_bolsa_llamamientos.ordenar_preferencias_bolsas_interna_v1(p.candidato,p.bolsas,0,
   'clave-prueba-ad279-b',clock_timestamp(),p.c,p.d,'\x04',p.x,1,1,'\x01','\x01','\x01','\x01');
  RAISE EXCEPTION 'AD279: versión superada aceptada' USING ERRCODE='55000';
 EXCEPTION WHEN sqlstate 'VBP01' THEN NULL; END;
END $idempotencia$;
RESET SESSION AUTHORIZATION;
DO $historia$
DECLARE p prueba_ad279%ROWTYPE; estado jsonb;
BEGIN
 SELECT * INTO STRICT p FROM prueba_ad279;
 SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
 IF (SELECT count(*) FROM vec_bolsa_llamamientos.preferencia_bolsas_version v WHERE v.candidato_ref=p.candidato)<>1
    OR (SELECT count(*) FROM vec_bolsa_llamamientos.preferencia_bolsas_outbox o WHERE o.candidato_ref=p.candidato)<>1
    OR EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.preferencia_bolsas_version v WHERE v.candidato_ref=p.candidato
      AND (v.clave_idempotencia !~ '^@[0-9a-f]{64}$' OR v.actor_ref_interno IS DISTINCT FROM 'per_prueba_ad279'
       OR v.perfil_ref_interno IS DISTINCT FROM 'prf_prueba_ad279'))
 THEN RAISE EXCEPTION 'AD279: historia o outbox divergentes' USING ERRCODE='55000'; END IF;
 estado:=vec_bolsa_llamamientos.preferencias_bolsas_estado_interno_v1(p.candidato,clock_timestamp());
 IF estado->>'version' IS DISTINCT FROM '1' OR estado->'orden_guardado' IS DISTINCT FROM to_jsonb(p.bolsas)
 THEN RAISE EXCEPTION 'AD279: la lectura no refleja el orden guardado' USING ERRCODE='55000'; END IF;
END $historia$;
ROLLBACK;
