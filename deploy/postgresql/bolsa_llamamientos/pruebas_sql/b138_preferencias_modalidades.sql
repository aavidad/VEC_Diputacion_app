\set ON_ERROR_STOP on
-- B138 en copia desechable, después de AD296/AUT104/B138. Todo revierte.
-- Este doble V3 prueba atomicidad, CAS y replay; no acredita firma real.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $metadatos$
DECLARE v_presente oid;v_ausente oid;v_firma_ajena oid;
BEGIN
 IF has_schema_privilege(current_user,'vec_autorizacion','USAGE') IS NOT FALSE
 THEN RAISE EXCEPTION 'B138: la prueba requiere Bolsa sin USAGE en AUT' USING ERRCODE='55000'; END IF;
 SELECT p.oid INTO v_presente FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
 WHERE n.nspname='vec_autorizacion' AND p.proname='cotejar_campos_preferencias_modalidades_aut104'
  AND p.proargtypes='3802 3802'::oidvector AND p.pronargs=2
  AND p.prorettype=16::oid AND p.prokind='f';
 SELECT p.oid INTO v_ausente FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
 WHERE n.nspname='vec_autorizacion' AND p.proname='cotejar_campos_preferencias_modalidades_aut104_ausente'
  AND p.proargtypes='3802 3802'::oidvector AND p.pronargs=2
  AND p.prorettype=16::oid AND p.prokind='f';
 SELECT p.oid INTO v_firma_ajena FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
 WHERE n.nspname='vec_autorizacion' AND p.proname='cotejar_campos_preferencias_modalidades_aut104'
  AND p.proargtypes='3802 23'::oidvector AND p.pronargs=2
  AND p.prorettype=16::oid AND p.prokind='f';
 IF v_presente IS NULL OR v_ausente IS NOT NULL OR v_firma_ajena IS NOT NULL
 THEN RAISE EXCEPTION 'B138: búsqueda exacta de AUT104 divergente' USING ERRCODE='55000'; END IF;
END $metadatos$;
DO $golden$
DECLARE c text:='can_'||repeat('a',43);opciones jsonb;h text;ctx text;
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE
  p.oid='vec_bolsa_llamamientos.guardar_preferencias_modalidades_v1(text,text[],jsonb,bigint,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
  AND p.pronargs=16 AND p.proargtypes=
   '25 1009 3802 20 25 1184 17 17 17 17 1700 1700 17 17 17 17'::oidvector AND p.prokind='f')
 THEN RAISE EXCEPTION 'B138: firma de escritora divergente' USING ERRCODE='55000'; END IF;
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
 IF EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
  WHERE p.oid IN ('vec_bolsa_llamamientos.huella_solicitud_modalidades_v1(text,bigint,text[],jsonb)'::regprocedure,
   'vec_bolsa_llamamientos.huella_contexto_modalidades_v1(text,text)'::regprocedure,
   'vec_bolsa_llamamientos.guardar_preferencias_modalidades_v1(text,text[],jsonb,bigint,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure)
   AND a.grantee=0 AND a.privilege_type='EXECUTE')
 OR has_function_privilege('vec_bolsa_llamamientos_portal_externo','vec_bolsa_llamamientos.guardar_preferencias_modalidades_v1(text,text[],jsonb,bigint,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE') IS NOT TRUE
 OR has_function_privilege('vec_bolsa_llamamientos_preferencias_interna','vec_bolsa_llamamientos.guardar_preferencias_modalidades_v1(text,text[],jsonb,bigint,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE') IS NOT TRUE
 OR NOT EXISTS(SELECT 1 FROM pg_class t WHERE t.oid='vec_bolsa_llamamientos.preferencia_modalidades_version'::regclass
  AND t.relrowsecurity AND t.relforcerowsecurity
  AND (SELECT count(*)=8 AND count(DISTINCT a.privilege_type)=8 AND
    bool_and(a.grantee=t.relowner AND a.grantor=t.relowner AND NOT a.is_grantable)
   FROM aclexplode(coalesce(t.relacl,acldefault('r',t.relowner))) a) IS TRUE)
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
DECLARE p prueba_b138%ROWTYPE;q prueba_b138%ROWTYPE;r1 record;r2 record;r3 record;r4 record;campo text;
 v_metadatos jsonb;v_opciones jsonb;v_huella text;v_contexto text;v_capacidad bytea;v_decision bytea;
 v_contexto_actor bytea;v_replay record;
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
 SELECT jsonb_agg(jsonb_set(jsonb_set(e,'{catalogo_version}','4'::jsonb),
  '{catalogo_huella_sha256}',to_jsonb(repeat('c',64))) ORDER BY e->>'clave')
 INTO v_metadatos FROM jsonb_array_elements(p.opciones) e;
 FOREACH v_opciones IN ARRAY ARRAY[v_metadatos,jsonb_build_array(p.opciones->1),
  p.opciones||jsonb_build_array(jsonb_build_object('clave','programa',
   'catalogo_id','vec.contratacion_temporal.reglas','catalogo_version',4,
   'catalogo_huella_sha256',repeat('c',64),'entrada_clave','c12.modalidad.programa'))] LOOP
  v_huella:=vec_bolsa_llamamientos.huella_solicitud_modalidades_v1(
   p.candidato,0,ARRAY['sustitucion','vacante'],v_opciones);
  v_contexto:=vec_bolsa_llamamientos.huella_contexto_modalidades_v1(p.candidato,v_huella);
  v_capacidad:=convert_to(jsonb_set(convert_from(p.capacidad,'UTF8')::jsonb,
   '{huella_efecto_sha256}',to_jsonb(v_contexto))::text,'UTF8');
  v_decision:=convert_to(jsonb_set(convert_from(p.decision,'UTF8')::jsonb,
   '{contexto_recurso_huella_sha256}',to_jsonb(v_contexto))::text,'UTF8');
  SELECT * INTO STRICT v_replay FROM vec_bolsa_llamamientos.guardar_preferencias_modalidades_v1(
   p.candidato,ARRAY['vacante','sustitucion'],v_opciones,0,'clave-prueba-b138-a',clock_timestamp(),
   v_capacidad,v_decision,'\x09',p.contexto,1,1,'\x01','\x01','\x01','\x01');
  IF v_huella IS NOT DISTINCT FROM vec_bolsa_llamamientos.huella_solicitud_modalidades_v1(
   p.candidato,0,ARRAY['sustitucion','vacante'],p.opciones)
   OR NOT v_replay.reutilizada OR v_replay.recibo_ref IS DISTINCT FROM r1.recibo_ref
   OR v_replay.registrada_en IS DISTINCT FROM r1.registrada_en
   OR v_replay.modalidades IS DISTINCT FROM r1.modalidades
  THEN RAISE EXCEPTION 'B138: replay con catálogo actualizado divergente' USING ERRCODE='55000'; END IF;
 END LOOP;
 v_huella:=vec_bolsa_llamamientos.huella_solicitud_modalidades_v1(
  p.candidato,0,ARRAY['sustitucion'],p.opciones);
 v_contexto:=vec_bolsa_llamamientos.huella_contexto_modalidades_v1(p.candidato,v_huella);
 v_capacidad:=convert_to(jsonb_set(convert_from(p.capacidad,'UTF8')::jsonb,
  '{huella_efecto_sha256}',to_jsonb(v_contexto))::text,'UTF8');
 v_decision:=convert_to(jsonb_set(convert_from(p.decision,'UTF8')::jsonb,
  '{contexto_recurso_huella_sha256}',to_jsonb(v_contexto))::text,'UTF8');
 BEGIN
  PERFORM * FROM vec_bolsa_llamamientos.guardar_preferencias_modalidades_v1(
   p.candidato,ARRAY['sustitucion'],p.opciones,0,'clave-prueba-b138-a',clock_timestamp(),
   v_capacidad,v_decision,'\x0a',p.contexto,1,1,'\x01','\x01','\x01','\x01');
  RAISE EXCEPTION 'B138: otra selección reutilizó clave' USING ERRCODE='55000';
 EXCEPTION WHEN sqlstate 'VBP02' THEN NULL; END;
 v_huella:=vec_bolsa_llamamientos.huella_solicitud_modalidades_v1(
  p.candidato,3,ARRAY['ajena'],p.opciones);
 v_contexto:=vec_bolsa_llamamientos.huella_contexto_modalidades_v1(p.candidato,v_huella);
 v_capacidad:=convert_to(jsonb_set(convert_from(p.capacidad,'UTF8')::jsonb,
  '{huella_efecto_sha256}',to_jsonb(v_contexto))::text,'UTF8');
 v_decision:=convert_to(jsonb_set(convert_from(p.decision,'UTF8')::jsonb,
  '{contexto_recurso_huella_sha256}',to_jsonb(v_contexto))::text,'UTF8');
 BEGIN
  PERFORM * FROM vec_bolsa_llamamientos.guardar_preferencias_modalidades_v1(
   p.candidato,ARRAY['ajena'],p.opciones,3,'clave-prueba-b138-ajena',clock_timestamp(),
   v_capacidad,v_decision,'\x0b',p.contexto,1,1,'\x01','\x01','\x01','\x01');
  RAISE EXCEPTION 'B138: escritura nueva con modalidad ajena' USING ERRCODE='55000';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 v_huella:=vec_bolsa_llamamientos.huella_solicitud_modalidades_v1(
  p.candidato,1,ARRAY['sustitucion','vacante'],p.opciones);
 v_contexto:=vec_bolsa_llamamientos.huella_contexto_modalidades_v1(p.candidato,v_huella);
 v_capacidad:=convert_to(jsonb_set(convert_from(p.capacidad,'UTF8')::jsonb,
  '{huella_efecto_sha256}',to_jsonb(v_contexto))::text,'UTF8');
 v_decision:=convert_to(jsonb_set(convert_from(p.decision,'UTF8')::jsonb,
  '{contexto_recurso_huella_sha256}',to_jsonb(v_contexto))::text,'UTF8');
 BEGIN
  PERFORM * FROM vec_bolsa_llamamientos.guardar_preferencias_modalidades_v1(
   p.candidato,ARRAY['sustitucion','vacante'],p.opciones,1,'clave-prueba-b138-a',clock_timestamp(),
   v_capacidad,v_decision,'\x0c',p.contexto,1,1,'\x01','\x01','\x01','\x01');
  RAISE EXCEPTION 'B138: otra versión reutilizó clave' USING ERRCODE='55000';
 EXCEPTION WHEN sqlstate 'VBP02' THEN NULL; END;
 FOREACH campo IN ARRAY ARRAY['principal_ref','perfil_activo_ref'] LOOP
  v_contexto_actor:=convert_to(jsonb_set(convert_from(p.contexto,'UTF8')::jsonb,
   ARRAY[campo],to_jsonb('otro_'||campo))::text,'UTF8');
  v_decision:=convert_to(jsonb_set(convert_from(p.decision,'UTF8')::jsonb,
   ARRAY[CASE WHEN campo='principal_ref' THEN 'principal_id' ELSE campo END],
   to_jsonb('otro_'||campo))::text,'UTF8');
  BEGIN
   PERFORM * FROM vec_bolsa_llamamientos.guardar_preferencias_modalidades_v1(
    p.candidato,ARRAY['sustitucion','vacante'],p.opciones,0,'clave-prueba-b138-a',clock_timestamp(),
    p.capacidad,v_decision,'\x0d',v_contexto_actor,1,1,'\x01','\x01','\x01','\x01');
   RAISE EXCEPTION 'B138: otra identidad reutilizó clave' USING ERRCODE='55000';
  EXCEPTION WHEN sqlstate 'VBP02' THEN NULL; END;
 END LOOP;
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
 FOREACH campo IN ARRAY ARRAY['clave','catalogo_id','catalogo_version',
  'catalogo_huella_sha256','entrada_clave'] LOOP
  BEGIN
   PERFORM * FROM vec_bolsa_llamamientos.guardar_preferencias_modalidades_v1(
    p.candidato,ARRAY['sustitucion','vacante'],
    jsonb_set(p.opciones,ARRAY['0',campo],'null'::jsonb),0,
    'clave-prueba-b138-null',clock_timestamp(),p.capacidad,p.decision,
    '\x07',p.contexto,1,1,'\x01','\x01','\x01','\x01');
   RAISE EXCEPTION 'B138: campo nulo aceptado' USING ERRCODE='55000';
  EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 END LOOP;
 BEGIN
  PERFORM * FROM vec_bolsa_llamamientos.guardar_preferencias_modalidades_v1(
   p.candidato,ARRAY['sustitucion','vacante'],
   jsonb_set(p.opciones,'{0,catalogo_huella_sha256}',to_jsonb(repeat('1',64)::numeric)),0,
   'clave-prueba-b138-tipo',clock_timestamp(),p.capacidad,p.decision,
   '\x08',p.contexto,1,1,'\x01','\x01','\x01','\x01');
  RAISE EXCEPTION 'B138: huella numérica aceptada' USING ERRCODE='55000';
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
 IF NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.preferencia_modalidades_version h
  JOIN prueba_b138 p ON p.tipo='0' AND p.candidato=h.candidato_ref
  WHERE h.version=1 AND h.catalogo_snapshot=p.opciones
   AND h.modalidades=ARRAY['sustitucion','vacante']::text[]
   AND h.solicitud_huella_sha256=vec_bolsa_llamamientos.huella_solicitud_modalidades_v1(
    h.candidato_ref,0,h.modalidades,h.catalogo_snapshot)
   AND h.contexto_huella_sha256=vec_bolsa_llamamientos.huella_contexto_modalidades_v1(
    h.candidato_ref,h.solicitud_huella_sha256)
   AND h.recibo_ref='recibo:preferencias-modalidades:'||encode(sha256(convert_to(
    h.candidato_ref||chr(31)||h.clave_idempotencia||chr(31)||h.solicitud_huella_sha256,'UTF8')),'hex'))
 THEN RAISE EXCEPTION 'B138: replay alteró integridad original' USING ERRCODE='55000'; END IF;
END $historia$;
ROLLBACK;
