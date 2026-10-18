\set ON_ERROR_STOP on
-- B138 en copia desechable, después de AD291/AUT104/B138. Todo revierte.
-- Este doble V3 prueba atomicidad, CAS y replay; no acredita firma real.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $golden$
DECLARE c text:='can_'||repeat('a',43);opciones jsonb;h text;ctx text;
BEGIN
 opciones:=jsonb_build_array(
  jsonb_build_object('clave','vacante','catalogo_id','vec.contratacion_temporal.reglas',
   'catalogo_version',3,'catalogo_huella_sha256',repeat('b',64),'entrada_clave','c12.modalidad.vacante'),
  jsonb_build_object('clave','programa','catalogo_id','vec.contratacion_temporal.reglas',
   'catalogo_version',3,'catalogo_huella_sha256',repeat('b',64),'entrada_clave','c12.modalidad.programa'),
  jsonb_build_object('clave','sustitucion','catalogo_id','vec.contratacion_temporal.reglas',
   'catalogo_version',3,'catalogo_huella_sha256',repeat('b',64),'entrada_clave','c12.modalidad.sustitucion'));
 h:=vec_bolsa_llamamientos.huella_solicitud_modalidades_v1(c,7,
  ARRAY['sustitucion','vacante'],opciones);
 IF h IS DISTINCT FROM '2b894c9c699723d958f3c0ca679e632ef478651966d99e2d64656bf29e26c76a'
 OR vec_bolsa_llamamientos.huella_contexto_modalidades_v1(c,h) IS DISTINCT FROM
  '1717c482a88d6cafbbd27f81b7a2dd238429f5b6c709e2fec048a2498fc52648'
 OR h IS DISTINCT FROM vec_bolsa_llamamientos.huella_solicitud_modalidades_v1(c,7,
  ARRAY['vacante','sustitucion'],opciones)
 OR vec_bolsa_llamamientos.huella_solicitud_modalidades_v1(c,0,NULL,opciones) IS NOT DISTINCT FROM
  vec_bolsa_llamamientos.huella_solicitud_modalidades_v1(c,0,ARRAY[]::text[],opciones)
 THEN RAISE EXCEPTION 'B138: huella Go/SQL divergente' USING ERRCODE='55000'; END IF;
 IF has_function_privilege('public','vec_bolsa_llamamientos.guardar_preferencias_modalidades_v1(text,text[],jsonb,bigint,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 OR has_function_privilege('vec_bolsa_llamamientos_portal_externo','vec_bolsa_llamamientos.guardar_preferencias_modalidades_v1(text,text[],jsonb,bigint,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE') IS NOT TRUE
 OR has_function_privilege('vec_bolsa_llamamientos_preferencias_interna','vec_bolsa_llamamientos.guardar_preferencias_modalidades_v1(text,text[],jsonb,bigint,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE') IS NOT TRUE
 OR NOT EXISTS(SELECT 1 FROM pg_class t WHERE t.oid='vec_bolsa_llamamientos.preferencia_modalidades_version'::regclass
  AND t.relrowsecurity AND t.relforcerowsecurity)
 THEN RAISE EXCEPTION 'B138: ACL o RLS abiertos' USING ERRCODE='55000'; END IF;
END $golden$;
ROLLBACK;

BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';
DO $roles$
BEGIN
 IF to_regrole('vec_externo_bolsa_desarrollo') IS NULL THEN
  CREATE ROLE vec_externo_bolsa_desarrollo LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
 END IF;
 IF NOT pg_has_role('vec_externo_bolsa_desarrollo','vec_bolsa_llamamientos_portal_externo','MEMBER') THEN
  GRANT vec_bolsa_llamamientos_portal_externo TO vec_externo_bolsa_desarrollo
   WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
 END IF;
END $roles$;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_preferencias_bolsas_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,
 p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,
 consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $doble$
 SELECT c->>'decision_ref',c->>'efecto_ref',c->>'huella_efecto_sha256',repeat('f',64),
  'auditoria:prueba-b138:'||(c->>'decision_ref'),clock_timestamp(),true
 FROM (SELECT convert_from(p_capacidad,'UTF8')::jsonb c) x
$doble$;
RESET ROLE;
CREATE TEMP TABLE prueba_b138(tipo text PRIMARY KEY,candidato text,opciones jsonb,capacidad bytea,decision bytea,contexto bytea)
ON COMMIT DROP;
GRANT SELECT ON prueba_b138 TO vec_externo_bolsa_desarrollo;
DO $material$
DECLARE c text:='can_'||repeat('z',43);opciones jsonb;h text;ctx text;version integer;
BEGIN
 opciones:=jsonb_build_array(
  jsonb_build_object('clave','vacante','catalogo_id','vec.contratacion_temporal.reglas',
   'catalogo_version',3,'catalogo_huella_sha256',repeat('b',64),'entrada_clave','c12.modalidad.vacante'),
  jsonb_build_object('clave','sustitucion','catalogo_id','vec.contratacion_temporal.reglas',
   'catalogo_version',3,'catalogo_huella_sha256',repeat('b',64),'entrada_clave','c12.modalidad.sustitucion'));
 FOR version IN 0..2 LOOP
  h:=vec_bolsa_llamamientos.huella_solicitud_modalidades_v1(c,version,
   CASE WHEN version=0 THEN ARRAY['sustitucion','vacante']
    WHEN version=1 THEN ARRAY[]::text[] ELSE NULL::text[] END,opciones);
  ctx:=vec_bolsa_llamamientos.huella_contexto_modalidades_v1(c,h);
  INSERT INTO prueba_b138 VALUES(version::text,c,opciones,
   convert_to(jsonb_build_object('operacion','bolsa.preferencias_bolsas.ordenar_propias',
    'audiencia_consumo','vec.bolsa.preferencias-bolsas.ordenar.v1',
    'efecto_ref','mi-bolsa:'||c,'huella_efecto_sha256',ctx,
    'decision_ref','decision:prueba-b138:'||version)::text,'UTF8'),
   convert_to(jsonb_build_object('accion','bolsa.preferencias_bolsas.ordenar_propias',
    'modulo_id','bolsa','tipo_recurso','participaciones_candidato',
    'finalidad','gestion_preferencias_bolsas_propias','recurso_ref','mi-bolsa:'||c,
    'contexto_recurso_huella_sha256',ctx,'principal_id','per_prueba_b138',
    'perfil_activo_ref','prf_prueba_b138','campos_permitidos','[]'::jsonb,
    'obligaciones','[]'::jsonb,
    'vinculo_autenticacion_actor',jsonb_build_object('superficie','externa_personal'))::text,'UTF8'),
   convert_to(jsonb_build_object('principal_ref','per_prueba_b138','perfil_activo_ref','prf_prueba_b138',
    'vinculos',jsonb_build_array(jsonb_build_object('tipo','candidato','estado','activo','referencia',c)))::text,'UTF8'));
 END LOOP;
END $material$;
SET SESSION AUTHORIZATION vec_externo_bolsa_desarrollo;
DO $efecto$
DECLARE p prueba_b138%ROWTYPE;q prueba_b138%ROWTYPE;r1 record;r2 record;r3 record;r4 record;
BEGIN
 SELECT * INTO STRICT p FROM prueba_b138 WHERE tipo='0';
 SELECT * INTO STRICT r1 FROM vec_bolsa_llamamientos.guardar_preferencias_modalidades_v1(
  p.candidato,ARRAY['sustitucion','vacante'],p.opciones,0,'clave-prueba-b138-a',clock_timestamp(),
  p.capacidad,p.decision,'\x01',p.contexto,1,1,'\x01','\x01','\x01','\x01');
 IF r1.reutilizada OR r1.version<>1 OR r1.modalidades IS DISTINCT FROM ARRAY['sustitucion','vacante']::text[]
 THEN RAISE EXCEPTION 'B138: primera selección incorrecta' USING ERRCODE='55000'; END IF;
 SELECT * INTO STRICT r2 FROM vec_bolsa_llamamientos.guardar_preferencias_modalidades_v1(
  p.candidato,ARRAY['vacante','sustitucion'],p.opciones,0,'clave-prueba-b138-a',clock_timestamp(),
  p.capacidad,p.decision,'\x02',p.contexto,1,1,'\x01','\x01','\x01','\x01');
 IF NOT r2.reutilizada OR r2.recibo_ref IS DISTINCT FROM r1.recibo_ref
 OR r2.registrada_en IS DISTINCT FROM r1.registrada_en OR r2.modalidades IS DISTINCT FROM r1.modalidades
 THEN RAISE EXCEPTION 'B138: replay no conservó recibo' USING ERRCODE='55000'; END IF;
 SELECT * INTO STRICT q FROM prueba_b138 WHERE tipo='1';
 SELECT * INTO STRICT r3 FROM vec_bolsa_llamamientos.guardar_preferencias_modalidades_v1(
  q.candidato,ARRAY[]::text[],q.opciones,1,'clave-prueba-b138-b',clock_timestamp(),
  q.capacidad,q.decision,'\x03',q.contexto,1,1,'\x01','\x01','\x01','\x01');
 IF r3.version<>2 OR r3.modalidades IS NULL OR cardinality(r3.modalidades)<>0
 THEN RAISE EXCEPTION 'B138: ninguna modalidad se confundió con todas' USING ERRCODE='55000'; END IF;
 SELECT * INTO STRICT q FROM prueba_b138 WHERE tipo='2';
 SELECT * INTO STRICT r4 FROM vec_bolsa_llamamientos.guardar_preferencias_modalidades_v1(
  q.candidato,NULL::text[],q.opciones,2,'clave-prueba-b138-d',clock_timestamp(),
  q.capacidad,q.decision,'\x05',q.contexto,1,1,'\x01','\x01','\x01','\x01');
 IF r4.version<>3 OR r4.modalidades IS NOT NULL OR r4.recibo_ref IS NOT DISTINCT FROM r3.recibo_ref
 THEN RAISE EXCEPTION 'B138: todas las modalidades se confundieron con ninguna' USING ERRCODE='55000'; END IF;
 BEGIN
  PERFORM * FROM vec_bolsa_llamamientos.guardar_preferencias_modalidades_v1(
   p.candidato,ARRAY['vacante','vacante'],p.opciones,0,'clave-prueba-b138-dup',clock_timestamp(),
   p.capacidad,p.decision,'\x06',p.contexto,1,1,'\x01','\x01','\x01','\x01');
  RAISE EXCEPTION 'B138: selección duplicada aceptada' USING ERRCODE='55000';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 BEGIN
  PERFORM * FROM vec_bolsa_llamamientos.guardar_preferencias_modalidades_v1(
   p.candidato,ARRAY['sustitucion','vacante'],p.opciones,0,'clave-prueba-b138-c',clock_timestamp(),
   p.capacidad,p.decision,'\x04',p.contexto,1,1,'\x01','\x01','\x01','\x01');
  RAISE EXCEPTION 'B138: CAS antiguo admitido' USING ERRCODE='55000';
 EXCEPTION WHEN sqlstate 'VBP01' THEN NULL; END;
END $efecto$;
RESET SESSION AUTHORIZATION;
DO $historia$
BEGIN
 SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
 IF (SELECT count(*) FROM vec_bolsa_llamamientos.preferencia_modalidades_version
  WHERE candidato_ref='can_'||repeat('z',43))<>3
 THEN RAISE EXCEPTION 'B138: historia duplicada' USING ERRCODE='55000'; END IF;
END $historia$;
ROLLBACK;
