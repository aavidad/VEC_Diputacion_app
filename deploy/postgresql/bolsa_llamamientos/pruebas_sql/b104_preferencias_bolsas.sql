\set ON_ERROR_STOP on
-- Ensayo focal en PostgreSQL 18 desechable, después de AD241 y B104.
-- Los materiales nominales de V3 se prueban desde el fixture de atestación;
-- aquí se cotejan estructura, ACL, historia, proyección y denegación directa.
BEGIN;
CREATE ROLE vec_bolsa_preferencias_ajeno_prueba LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL statement_timeout='15s';

DO $prueba$
DECLARE v_candidato text:='can_AAAAAAAAAAAAAAAAAAAAAA'; v_estado jsonb; v_rechazada boolean:=false;
 v_tabla oid:='vec_bolsa_llamamientos.preferencia_bolsas_version'::regclass;
 v_outbox oid:='vec_bolsa_llamamientos.preferencia_bolsas_outbox'::regclass;
 v_funcion oid:=to_regprocedure('vec_bolsa_llamamientos.ordenar_preferencias_bolsas_v1(text,text[],bigint,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 v_lectura oid:=to_regprocedure('vec_bolsa_llamamientos.consultar_preferencias_bolsas_v1(text,bytea,bytea,jsonb)');
BEGIN
 IF v_funcion IS NULL OR v_lectura IS NULL
    OR NOT EXISTS (SELECT 1 FROM pg_class WHERE oid=v_tabla AND relrowsecurity AND relforcerowsecurity)
    OR NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgrelid=v_tabla AND tgname='preferencia_bolsas_version_inmutable' AND tgenabled='O')
    OR NOT EXISTS (SELECT 1 FROM pg_class WHERE oid=v_outbox AND relrowsecurity AND relforcerowsecurity)
    OR NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgrelid=v_outbox AND tgname='preferencia_bolsas_outbox_inmutable' AND tgenabled='O')
    OR NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgrelid=v_outbox AND tgname='preferencia_bolsas_outbox_no_truncar' AND tgenabled='O')
    OR has_table_privilege('public',v_outbox,'SELECT,INSERT,UPDATE,DELETE')
    OR has_table_privilege('vec_bolsa_llamamientos_ejecutor',v_outbox,'SELECT,INSERT,UPDATE,DELETE')
    OR has_type_privilege('public',(SELECT reltype FROM pg_class WHERE oid=v_outbox),'USAGE')
    OR has_table_privilege('vec_bolsa_llamamientos_ejecutor',v_tabla,'SELECT')
    OR has_function_privilege('vec_bolsa_llamamientos_ejecutor',v_funcion,'EXECUTE')
    OR NOT has_function_privilege('vec_bolsa_llamamientos_portal_externo',v_funcion,'EXECUTE')
    OR has_function_privilege('public',v_funcion,'EXECUTE')
    OR has_function_privilege('vec_bolsa_preferencias_ajeno_prueba',v_funcion,'EXECUTE')
    OR NOT has_function_privilege('vec_bolsa_llamamientos_portal_externo',v_lectura,'EXECUTE')
    OR has_function_privilege('public',v_lectura,'EXECUTE')
    OR has_function_privilege('vec_bolsa_preferencias_ajeno_prueba',v_lectura,'EXECUTE')
    OR has_function_privilege('vec_bolsa_llamamientos_portal_externo',
     'vec_bolsa_llamamientos.preferencias_bolsas_estado_interno_v1(text,timestamptz)','EXECUTE')
 THEN RAISE EXCEPTION 'B104: ACL/historia incompatibles' USING ERRCODE='55000'; END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_constraint c
   WHERE c.conrelid='vec_bolsa_llamamientos.denegacion_frontera_portal_externo'::regclass
    AND c.contype='c' AND c.convalidated
    AND pg_get_constraintdef(c.oid) LIKE '%/api/vec/bolsa/mi-bolsa/preferencias%')
    OR strpos(pg_get_functiondef('vec_bolsa_llamamientos.registrar_denegacion_portal_externo_v1(text,text,text,text,text)'::regprocedure),
     '/api/vec/bolsa/mi-bolsa/preferencias')=0
 THEN RAISE EXCEPTION 'B104: frontera de denegación ausente' USING ERRCODE='55000'; END IF;
 BEGIN
  PERFORM vec_bolsa_llamamientos.consultar_preferencias_bolsas_v1(v_candidato,
   convert_to('{}','UTF8'),convert_to('{}','UTF8'),'{}'::jsonb);
  RAISE EXCEPTION 'B104: lectura sin sesión aceptada' USING ERRCODE='55000';
 EXCEPTION WHEN insufficient_privilege THEN v_rechazada:=true; END;
 IF NOT v_rechazada THEN RAISE EXCEPTION 'B104: lectura sin sesión no denegada' USING ERRCODE='55000'; END IF;
 v_rechazada:=false;
 v_estado:=vec_bolsa_llamamientos.preferencias_bolsas_estado_interno_v1(v_candidato,clock_timestamp());
 IF v_estado->>'version' IS DISTINCT FROM '0' OR v_estado->>'revision_requerida' IS DISTINCT FROM 'true'
    OR v_estado->'orden_guardado' IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'B104: estado inicial incorrecto' USING ERRCODE='55000'; END IF;
 INSERT INTO vec_bolsa_llamamientos.preferencia_bolsas_version(
  candidato_ref,version,orden,conjunto_huella_sha256,clave_idempotencia,
  solicitud_huella_sha256,recibo_ref,decision_ref,auditoria_ref,registrada_en)
 VALUES(v_candidato,1,ARRAY['bolsa:a','bolsa:b'],repeat('a',64),'clave-prueba-1',repeat('b',64),
  'recibo:preferencias-bolsas:'||repeat('c',64),'decision:prueba:1','auditoria:prueba:1',clock_timestamp());
 v_estado:=vec_bolsa_llamamientos.preferencias_bolsas_estado_interno_v1(v_candidato,clock_timestamp());
 IF v_estado->>'version' IS DISTINCT FROM '1' OR v_estado->>'revision_requerida' IS DISTINCT FROM 'true'
    OR v_estado->'orden_guardado' IS DISTINCT FROM '["bolsa:a","bolsa:b"]'::jsonb
    OR v_estado->'bolsas_retiradas' IS DISTINCT FROM '["bolsa:a","bolsa:b"]'::jsonb
 THEN RAISE EXCEPTION 'B104: proyeccion de retirada incorrecta' USING ERRCODE='55000'; END IF;
 BEGIN
  UPDATE vec_bolsa_llamamientos.preferencia_bolsas_version SET orden=ARRAY['bolsa:b','bolsa:a']
   WHERE candidato_ref=v_candidato AND version=1;
 EXCEPTION WHEN sqlstate '55000' THEN v_rechazada:=true; END;
 IF NOT v_rechazada THEN RAISE EXCEPTION 'B104: historia mutable' USING ERRCODE='55000'; END IF;
END $prueba$;
ROLLBACK;

-- La fachada AD241 real deniega una audiencia de lectura aun cuando la
-- acción de escritura sea nominal; se corta antes del núcleo y sin efecto.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
DO $audiencia_ajena$
DECLARE v_code text;
BEGIN
 BEGIN
  PERFORM 1 FROM vec_autorizacion_atestada_v3.registrar_y_consumir_preferencias_bolsas_v3_atestada(
   convert_to('{"operacion":"bolsa.preferencias_bolsas.ordenar_propias","audiencia_consumo":"vec.bolsa.mi-bolsa.v1"}','UTF8'),
   convert_to('{}','UTF8'),decode('01','hex'),decode('01','hex'),1,1,
   decode('01','hex'),decode('01','hex'),decode('01','hex'),decode('01','hex'));
 EXCEPTION WHEN others THEN GET STACKED DIAGNOSTICS v_code=RETURNED_SQLSTATE; END;
 IF v_code IS DISTINCT FROM '42501' THEN RAISE EXCEPTION 'AD241: audiencia ajena admitida: %',v_code USING ERRCODE='55000'; END IF;
END $audiencia_ajena$;
ROLLBACK;

-- Doble privado SOLO en este ROLLBACK: prueba el contrato B104 de CAS/replay
-- sin afirmar que el material COSE sintético haya pasado el núcleo real AD3.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_preferencias_bolsas_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,
 p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,
 consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET lock_timeout='2s' AS $doble$
DECLARE c jsonb:=convert_from(p_capacidad,'UTF8')::jsonb;
BEGIN
 RETURN QUERY SELECT c->>'decision_ref',c->>'efecto_ref',repeat('a',64),repeat('b',64),
  'auditoria:prueba:'||(c->>'decision_ref'),clock_timestamp(),true;
END $doble$;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.preferencias_bolsas_actuales_interna_v1(
 p_candidato_ref text,p_corte timestamptz)
RETURNS text[] LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $doble$
BEGIN
 IF p_candidato_ref !~ '^can_[A-Za-z0-9_-]{22,128}$' THEN
  RAISE EXCEPTION 'candidato inválido' USING ERRCODE='42501'; END IF;
 RETURN ARRAY['bolsa:a','bolsa:b']::text[];
END $doble$;
SET LOCAL timezone='UTC';
SET LOCAL statement_timeout='15s';

DO $contrato$
DECLARE v_candidato text:='can_AAAAAAAAAAAAAAAAAAAAAA';
 v_ajeno text:='can_BBBBBBBBBBBBBBBBBBBBBB';
 c jsonb; d jsonb; x jsonb; r record; v_recibo text; v_fecha timestamptz;
 v_sqlstate text;
BEGIN
 d:=jsonb_build_object('accion','bolsa.preferencias_bolsas.ordenar_propias',
  'modulo_id','bolsa','tipo_recurso','participaciones_candidato',
  'finalidad','gestion_preferencias_bolsas_propias','recurso_ref','mi-bolsa:'||v_candidato,
  'contexto_recurso_huella_sha256',repeat('a',64),'campos_permitidos','[]'::jsonb,
  'obligaciones','[]'::jsonb);
 x:=jsonb_build_object('vinculos',jsonb_build_array(jsonb_build_object(
  'tipo','candidato','estado','activo','referencia',v_candidato)));
 c:=jsonb_build_object('operacion','bolsa.preferencias_bolsas.ordenar_propias',
  'audiencia_consumo','vec.bolsa.preferencias-bolsas.ordenar.v1',
  'efecto_ref','mi-bolsa:'||v_candidato,'huella_efecto_sha256',repeat('a',64),
  'decision_ref','decision:preferencias:1');
 SELECT * INTO STRICT r FROM vec_bolsa_llamamientos.ordenar_preferencias_bolsas_v1(
  v_candidato,ARRAY['bolsa:b','bolsa:a'],0,'clave-prueba-cas-1',clock_timestamp(),
  convert_to(c::text,'UTF8'),convert_to(d::text,'UTF8'),decode('01','hex'),
  convert_to(x::text,'UTF8'),1,1,decode('01','hex'),decode('01','hex'),
  decode('01','hex'),decode('01','hex'));
 IF r.reutilizada IS DISTINCT FROM false OR r.version IS DISTINCT FROM 1 OR r.orden IS DISTINCT FROM ARRAY['bolsa:b','bolsa:a'] THEN
  RAISE EXCEPTION 'B104: alta incorrecta' USING ERRCODE='55000'; END IF;
 v_recibo:=r.recibo_ref; v_fecha:=r.registrada_en;
 IF (SELECT count(*) FROM vec_bolsa_llamamientos.preferencia_bolsas_outbox o
     WHERE o.candidato_ref=v_candidato AND o.version=1 AND o.recibo_ref=v_recibo
      AND o.decision_ref='decision:preferencias:1'
      AND o.auditoria_ref='auditoria:prueba:decision:preferencias:1'
      AND o.evento->'orden'='["bolsa:b","bolsa:a"]'::jsonb)<>1 THEN
  RAISE EXCEPTION 'B104: estado sin evento/auditoría' USING ERRCODE='55000'; END IF;
 c:=jsonb_set(c,'{decision_ref}','"decision:preferencias:2"'::jsonb);
 SELECT * INTO STRICT r FROM vec_bolsa_llamamientos.ordenar_preferencias_bolsas_v1(
  v_candidato,ARRAY['bolsa:b','bolsa:a'],0,'clave-prueba-cas-1',clock_timestamp(),
  convert_to(c::text,'UTF8'),convert_to(d::text,'UTF8'),decode('01','hex'),
  convert_to(x::text,'UTF8'),1,1,decode('01','hex'),decode('01','hex'),
  decode('01','hex'),decode('01','hex'));
 IF r.reutilizada IS DISTINCT FROM true OR r.recibo_ref IS DISTINCT FROM v_recibo OR r.registrada_en IS DISTINCT FROM v_fecha OR r.version IS DISTINCT FROM 1
    OR (SELECT count(*) FROM vec_bolsa_llamamientos.preferencia_bolsas_version p
        WHERE p.candidato_ref=v_candidato)<>1
    OR (SELECT count(*) FROM vec_bolsa_llamamientos.preferencia_bolsas_outbox o
        WHERE o.candidato_ref=v_candidato)<>1 THEN
  RAISE EXCEPTION 'B104: replay duplicó o sustituyó recibo' USING ERRCODE='55000'; END IF;
 BEGIN
  PERFORM 1 FROM vec_bolsa_llamamientos.ordenar_preferencias_bolsas_v1(
   v_candidato,ARRAY['bolsa:a','bolsa:b'],0,'clave-prueba-cas-2',clock_timestamp(),
   convert_to(c::text,'UTF8'),convert_to(d::text,'UTF8'),decode('01','hex'),
   convert_to(x::text,'UTF8'),1,1,decode('01','hex'),decode('01','hex'),
   decode('01','hex'),decode('01','hex'));
 EXCEPTION WHEN others THEN GET STACKED DIAGNOSTICS v_sqlstate=RETURNED_SQLSTATE; END;
 IF v_sqlstate IS DISTINCT FROM 'VBP01' THEN RAISE EXCEPTION 'B104: CAS aceptó versión anterior: %',v_sqlstate USING ERRCODE='55000'; END IF;
 v_sqlstate:=NULL;
 BEGIN
  PERFORM 1 FROM vec_bolsa_llamamientos.ordenar_preferencias_bolsas_v1(
   v_candidato,ARRAY['bolsa:a','bolsa:b'],0,'clave-prueba-cas-1',clock_timestamp(),
   convert_to(c::text,'UTF8'),convert_to(d::text,'UTF8'),decode('01','hex'),
   convert_to(x::text,'UTF8'),1,1,decode('01','hex'),decode('01','hex'),
   decode('01','hex'),decode('01','hex'));
 EXCEPTION WHEN others THEN GET STACKED DIAGNOSTICS v_sqlstate=RETURNED_SQLSTATE; END;
 IF v_sqlstate IS DISTINCT FROM 'VBP02' THEN RAISE EXCEPTION 'B104: clave cambió de significado: %',v_sqlstate USING ERRCODE='55000'; END IF;
 v_sqlstate:=NULL;
 x:=jsonb_build_object('vinculos',jsonb_build_array(jsonb_build_object(
  'tipo','candidato','estado','activo','referencia',v_ajeno)));
 BEGIN
  PERFORM 1 FROM vec_bolsa_llamamientos.ordenar_preferencias_bolsas_v1(
   v_candidato,ARRAY['bolsa:a','bolsa:b'],1,'clave-prueba-ajeno',clock_timestamp(),
   convert_to(c::text,'UTF8'),convert_to(d::text,'UTF8'),decode('01','hex'),
   convert_to(x::text,'UTF8'),1,1,decode('01','hex'),decode('01','hex'),
   decode('01','hex'),decode('01','hex'));
 EXCEPTION WHEN others THEN GET STACKED DIAGNOSTICS v_sqlstate=RETURNED_SQLSTATE; END;
 IF v_sqlstate IS DISTINCT FROM '42501' THEN RAISE EXCEPTION 'B104: actor ajeno aceptado: %',v_sqlstate USING ERRCODE='55000'; END IF;
END $contrato$;
ROLLBACK;
