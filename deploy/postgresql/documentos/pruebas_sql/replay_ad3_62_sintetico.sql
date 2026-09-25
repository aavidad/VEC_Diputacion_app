\set ON_ERROR_STOP on
-- Solo en PostgreSQL desechable: preimagen AD3 sintética, sin COSE real.
SET ROLE vec_autorizacion_atestada_v3_propietario;
ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version
 ADD COLUMN clave_id text, ADD COLUMN version numeric, ADD COLUMN revision_gobierno numeric,
 ADD COLUMN huella_gobierno_sha256 text, ADD COLUMN emisor_id text,
 ADD COLUMN valida_desde timestamptz, ADD COLUMN valida_hasta timestamptz;
INSERT INTO vec_autorizacion_atestada_v3.clave_capacidad_version
 (audiencia_consumo,clave_id,version,revision_gobierno,huella_gobierno_sha256,emisor_id,valida_desde,valida_hasta)
 VALUES('vec_documentos.operacion.v1','clave:ensayo',1,1,repeat('a',64),'emisor:ensayo',now()-interval '1 day',now()+interval '1 day');
CREATE TABLE vec_autorizacion_atestada_v3.puntero_clave_emision
 (orden bigint,clave_id text,version numeric,establecida_en timestamptz);
INSERT INTO vec_autorizacion_atestada_v3.puntero_clave_emision VALUES(1,'clave:ensayo',1,now()-interval '1 day');
CREATE TABLE vec_autorizacion_atestada_v3.checkpoint_gobierno
 (control_id boolean,configuracion_secuencia_minima numeric,raiz_version_minima numeric);
INSERT INTO vec_autorizacion_atestada_v3.checkpoint_gobierno VALUES(true,1,1);
CREATE TABLE vec_autorizacion_atestada_v3.configuracion_confianza_version
 (revision text,secuencia numeric,huella_configuracion_sha256 text,expira_en timestamptz);
INSERT INTO vec_autorizacion_atestada_v3.configuracion_confianza_version
 VALUES('config:ensayo',1,repeat('b',64),now()+interval '1 day');
CREATE TABLE vec_autorizacion_atestada_v3.puntero_configuracion_actual
 (orden bigint,configuracion_revision text,establecida_en timestamptz);
INSERT INTO vec_autorizacion_atestada_v3.puntero_configuracion_actual VALUES(1,'config:ensayo',now()-interval '1 day');
CREATE TABLE vec_autorizacion_atestada_v3.raiz_confianza_version
 (clave_id text,version numeric,huella_spki_sha256 text,clave_publica_spki bytea,
  suite text,audiencia_despliegue text,valida_desde timestamptz,valida_hasta timestamptz);
INSERT INTO vec_autorizacion_atestada_v3.raiz_confianza_version
 VALUES('raiz:ensayo',1,encode(sha256(decode(repeat('ab',44),'hex')),'hex'),
  decode(repeat('ab',44),'hex'),'VEC-AD-3-COSE-EDDSA-1','despliegue:ensayo',
  now()-interval '1 day',now()+interval '1 day');
CREATE TABLE vec_autorizacion_atestada_v3.configuracion_raiz
 (configuracion_revision text,raiz_clave_id text,raiz_version numeric);
INSERT INTO vec_autorizacion_atestada_v3.configuracion_raiz VALUES('config:ensayo','raiz:ensayo',1);
CREATE TABLE vec_autorizacion_atestada_v3.revocacion_clave_capacidad
 (clave_id text,version numeric,revocada_en timestamptz);
CREATE TABLE vec_autorizacion_atestada_v3.revocacion_configuracion
 (configuracion_revision text,revocada_en timestamptz);
CREATE TABLE vec_autorizacion_atestada_v3.revocacion_raiz
 (raiz_clave_id text,raiz_version numeric,revocada_en timestamptz);
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
 p_perfil_mutacion text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE c jsonb:=convert_from(p_capacidad,'UTF8')::jsonb;
BEGIN
 IF p_perfil_mutacion<>'operacion_documentos_comunes' THEN RAISE EXCEPTION 'perfil sintético inválido'; END IF;
 RETURN QUERY SELECT c->>'decision_ref',c->>'efecto_ref',c->>'huella_efecto_sha256',
  repeat('c',64),'auditoria:ensayo',clock_timestamp(),false;
END $f$;
RESET ROLE;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_documentos_ensayo;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_operacion_documentos_replay_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_documentos_ensayo;
CREATE TABLE public.ensayo_ad3_62(capacidad bytea,decision bytea,raiz bytea);
INSERT INTO public.ensayo_ad3_62
SELECT convert_to(jsonb_build_object(
 'audiencia_consumo','vec_documentos.operacion.v1','operacion','documentos.generado.alta',
 'efecto_ref','doc:00000000-0000-4000-8000-000000000001','huella_efecto_sha256',repeat('d',64),
 'decision_ref','decision:ensayo','clave_id','clave:ensayo','clave_version',1,
 'revision_gobierno',1,'huella_gobierno_sha256',repeat('a',64),'emisor_id','emisor:ensayo',
 'revision_confianza','config:ensayo','configuracion_secuencia',1,
 'huella_configuracion_sha256',repeat('b',64),'raiz_clave_id','raiz:ensayo','raiz_version',1,
 'huella_raiz_spki_sha256',encode(sha256(decode(repeat('ab',44),'hex')),'hex'),
 'suite','VEC-AD-3-COSE-EDDSA-1','audiencia_despliegue','despliegue:ensayo',
 'expira_en',to_char(now()+interval '1 hour','YYYY-MM-DD"T"HH24:MI:SS"Z"'),
 'decision_valida_hasta',to_char(now()+interval '1 hour','YYYY-MM-DD"T"HH24:MI:SS"Z"'))::text,'UTF8'),
 convert_to(jsonb_build_object('accion','documentos.generado.alta','modulo_id','documentos',
 'tipo_recurso','documento_generado','finalidad','alta_documento_generado',
 'campos_permitidos',jsonb_build_array('documento','recibo'),'obligaciones','[]'::jsonb,
 'recurso_ref','doc:00000000-0000-4000-8000-000000000001',
 'contexto_recurso_huella_sha256',repeat('d',64))::text,'UTF8'),
 decode(repeat('ab',44),'hex');
GRANT SELECT ON public.ensayo_ad3_62 TO vec_documentos_ensayo;

\connect postgres vec_documentos_ensayo
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';
DO $test$
DECLARE f record; r record;
BEGIN
 SELECT * INTO STRICT f FROM public.ensayo_ad3_62;
 SELECT * INTO STRICT r FROM vec_autorizacion_atestada_v3.consumir_operacion_documentos_replay_v3_atestada(
  f.capacidad,f.decision,'\x01'::bytea,'\x01'::bytea,1,1,
  '\x01'::bytea,'\x01'::bytea,'\x01'::bytea,f.raiz);
 IF r.consumo_nuevo IS DISTINCT FROM false OR r.decision_ref<>'decision:ensayo'
 THEN RAISE EXCEPTION 'AD3-62: replay sintético no revalidado'; END IF;
END $test$;
COMMIT;

\connect postgres postgres
INSERT INTO vec_autorizacion_atestada_v3.revocacion_clave_capacidad VALUES('clave:ensayo',1,now());
\connect postgres vec_documentos_ensayo
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';
DO $test$
DECLARE f record;
BEGIN
 SELECT * INTO STRICT f FROM public.ensayo_ad3_62;
 BEGIN
  PERFORM vec_autorizacion_atestada_v3.consumir_operacion_documentos_replay_v3_atestada(
   f.capacidad,f.decision,'\x01'::bytea,'\x01'::bytea,1,1,
   '\x01'::bytea,'\x01'::bytea,'\x01'::bytea,f.raiz);
  RAISE EXCEPTION 'AD3-62 aceptó clave revocada';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
END $test$;
COMMIT;
