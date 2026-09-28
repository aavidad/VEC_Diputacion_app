\set ON_ERROR_STOP on
-- Base efímera sintética. El núcleo AD3 es un doble de prueba y solo sirve
-- para comprobar composición SQL, ACL y consumo; no acredita criptografía V3.
CREATE ROLE vec_contratacion_temporal_propietario NOLOGIN;
CREATE ROLE vec_contratacion_temporal_ejecutor NOLOGIN;
CREATE ROLE vec_contratacion_temporal_migrador NOLOGIN;
CREATE ROLE vec_autorizacion_atestada_v3_propietario NOLOGIN;
CREATE ROLE vec_ct140_login LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS;
GRANT vec_contratacion_temporal_ejecutor TO vec_ct140_login WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
CREATE SCHEMA vec_contratacion_temporal AUTHORIZATION vec_contratacion_temporal_propietario;
CREATE SCHEMA vec_autorizacion_atestada_v3 AUTHORIZATION vec_autorizacion_atestada_v3_propietario;
GRANT USAGE ON SCHEMA vec_contratacion_temporal TO vec_contratacion_temporal_ejecutor;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_contratacion_temporal_propietario;

SET ROLE vec_autorizacion_atestada_v3_propietario;
CREATE TABLE vec_autorizacion_atestada_v3.clave_capacidad_version(
 audiencia_consumo text CONSTRAINT clave_capacidad_version_audiencia_consumo_check
 CHECK (audiencia_consumo = ANY (ARRAY['vec_contratacion_temporal.lectura_reincorporacion_titular.v1'::text])));
CREATE TABLE vec_autorizacion_atestada_v3.prueba_consumos(
 efecto_ref text,operacion text,registrado_en timestamptz DEFAULT clock_timestamp());
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_recibo_respuesta_ct_v3_atestada(
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql AS $$ SELECT ''::text,''::text,''::text,''::text,''::text,now(),false $$;
CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
 p_perfil_mutacion text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb;
BEGIN
 c:=convert_from(p_capacidad,'UTF8')::jsonb;
 d:=convert_from(p_decision,'UTF8')::jsonb;
 IF NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER') THEN
  RAISE EXCEPTION 'ejecutor requerido' USING ERRCODE='42501';
 END IF;
 IF (p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_recibo_respuesta_ct'
       )
       OR c ->> 'suite' <> 'VEC-AD-3-COSE-EDDSA-1'
 THEN NULL; END IF;
 INSERT INTO vec_autorizacion_atestada_v3.prueba_consumos(efecto_ref,operacion)
 VALUES(c->>'efecto_ref',c->>'operacion');
 RETURN QUERY SELECT 'decision:ct140'::text,c->>'efecto_ref',c->>'huella_efecto_sha256',
  repeat('a',64),'aud_v3_'||repeat('b',32),clock_timestamp(),true;
END $f$;
RESET ROLE;

SET ROLE vec_contratacion_temporal_propietario;
CREATE FUNCTION vec_contratacion_temporal.consultar_recibo_respuesta_rrhh_v1(
 text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
RETURNS jsonb LANGUAGE sql AS $$ SELECT '{}'::jsonb $$;
CREATE TABLE vec_contratacion_temporal.expediente_alta(
 expediente_ref text PRIMARY KEY,organizacion_ref text NOT NULL);
CREATE TABLE vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6(
 clave_idempotencia uuid PRIMARY KEY,situacion text,solicitud_json jsonb,recibo_json jsonb);
CREATE TABLE vec_contratacion_temporal.comunicacion_llamamiento_local(
 comunicacion_ref text PRIMARY KEY,organizacion_ref text,expediente_ref text,
 llamamiento_ref text,seleccion_clave uuid,version_resultante numeric,
 material_json jsonb,recibo_json jsonb,estado text,registrada_en timestamptz(6));
CREATE TABLE vec_contratacion_temporal.resolucion_manual_respuesta_rrhh(
 comunicacion_ref text,seleccion_clave uuid,organizacion_ref text,expediente_ref text,
 llamamiento_ref text,resolucion_ref text,estado text,solicitud_json jsonb,
 continuacion_clave uuid,continuacion_recibo jsonb);
DO $rls$
DECLARE t text;
BEGIN
 FOREACH t IN ARRAY ARRAY['expediente_alta','ejecucion_seleccion_llamamiento_o6',
 'comunicacion_llamamiento_local','resolucion_manual_respuesta_rrhh'] LOOP
  EXECUTE format('ALTER TABLE vec_contratacion_temporal.%I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE format('ALTER TABLE vec_contratacion_temporal.%I FORCE ROW LEVEL SECURITY',t);
  EXECUTE format('CREATE POLICY propietario ON vec_contratacion_temporal.%I TO vec_contratacion_temporal_propietario USING (true) WITH CHECK (true)',t);
  EXECUTE format('REVOKE ALL ON vec_contratacion_temporal.%I FROM PUBLIC, vec_contratacion_temporal_ejecutor',t);
 END LOOP;
END $rls$;
INSERT INTO vec_contratacion_temporal.expediente_alta VALUES
 ('expediente:ct140-a','organizacion:ct140-a'),('expediente:ct140-b','organizacion:ct140-a'),
 ('expediente:ct140-c','organizacion:ct140-b'),('expediente:ct140-vacio','organizacion:ct140-a');
INSERT INTO vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 VALUES
 ('10000000-0000-4000-8000-000000000001','confirmada',
  '{"organizacion_ref":"organizacion:ct140-a","expediente_ref":"expediente:ct140-a"}',
  '{"organizacion_ref":"organizacion:ct140-a","expediente_ref":"expediente:ct140-a","llamamiento_ref":"llamamiento:ct140-a","recibo_ref":"recibo:seleccion-a","propuesta_generada":true}'),
 ('10000000-0000-4000-8000-000000000002','confirmada',
  '{"organizacion_ref":"organizacion:ct140-a","expediente_ref":"expediente:ct140-b"}',
  '{"organizacion_ref":"organizacion:ct140-a","expediente_ref":"expediente:ct140-b","llamamiento_ref":"llamamiento:ct140-b","recibo_ref":"recibo:seleccion-b","propuesta_generada":true}'),
 ('10000000-0000-4000-8000-000000000003','confirmada',
  '{"organizacion_ref":"organizacion:ct140-b","expediente_ref":"expediente:ct140-c"}',
  '{"organizacion_ref":"organizacion:ct140-b","expediente_ref":"expediente:ct140-c","llamamiento_ref":"llamamiento:ct140-c","recibo_ref":"recibo:seleccion-c","propuesta_generada":true}');
INSERT INTO vec_contratacion_temporal.comunicacion_llamamiento_local
SELECT 'comunicacion:ct140-a','organizacion:ct140-a','expediente:ct140-a','llamamiento:ct140-a',
 '10000000-0000-4000-8000-000000000001',2,
 jsonb_build_object('solicitud',jsonb_build_object('OrganizacionRef','organizacion:ct140-a',
  'ExpedienteRef','expediente:ct140-a','LlamamientoRef','llamamiento:ct140-a',
  'PruebaEntregaRef','recibo:seleccion-a')),
 '{}'::jsonb,'registrada_localmente','2026-09-28 12:00:00.000001+00';
INSERT INTO vec_contratacion_temporal.resolucion_manual_respuesta_rrhh VALUES
 ('comunicacion:ct140-a','10000000-0000-4000-8000-000000000001',
  'organizacion:ct140-a','expediente:ct140-a','llamamiento:ct140-a',
  'resolucion:ct140-a','confirmado','{"Respuesta":"renuncia"}',
  '20000000-0000-4000-8000-000000000001',
  '{"Estado":"confirmado","ReciboRef":"recibo:continuacion-a","LlamamientoAnteriorRef":"llamamiento:ct140-a","Solicitud":{"OrganizacionRef":"organizacion:ct140-a","ExpedienteRef":"expediente:ct140-a","ResolucionRef":"resolucion:ct140-a","ClaveIdempotencia":"20000000-0000-4000-8000-000000000001"},"ReciboBolsa":{"LlamamientoRef":"llamamiento:ct140-a2"}}');
INSERT INTO vec_contratacion_temporal.comunicacion_llamamiento_local
SELECT 'comunicacion:ct140-a2','organizacion:ct140-a','expediente:ct140-a','llamamiento:ct140-a2',
 '10000000-0000-4000-8000-000000000001',2,
 jsonb_build_object('solicitud',jsonb_build_object('OrganizacionRef','organizacion:ct140-a',
  'ExpedienteRef','expediente:ct140-a','LlamamientoRef','llamamiento:ct140-a2',
  'PruebaEntregaRef','recibo:continuacion-a','TipoAntecedente','continuacion_confirmada')),
 '{}'::jsonb,'registrada_localmente','2026-09-28 12:00:00.000002+00';
INSERT INTO vec_contratacion_temporal.comunicacion_llamamiento_local
SELECT 'comunicacion:ct140-b','organizacion:ct140-a','expediente:ct140-b','llamamiento:ct140-b',
 '10000000-0000-4000-8000-000000000002',2,
 jsonb_build_object('solicitud',jsonb_build_object('OrganizacionRef','organizacion:ct140-a',
  'ExpedienteRef','expediente:ct140-b','LlamamientoRef','llamamiento:ct140-b',
  'PruebaEntregaRef','recibo:seleccion-b')),
 '{}'::jsonb,'registrada_localmente','2026-09-28 12:00:00.000003+00';
INSERT INTO vec_contratacion_temporal.comunicacion_llamamiento_local
SELECT 'comunicacion:ct140-c','organizacion:ct140-b','expediente:ct140-c','llamamiento:ct140-c',
 '10000000-0000-4000-8000-000000000003',2,
 jsonb_build_object('solicitud',jsonb_build_object('OrganizacionRef','organizacion:ct140-b',
  'ExpedienteRef','expediente:ct140-c','LlamamientoRef','llamamiento:ct140-c',
  'PruebaEntregaRef','recibo:seleccion-c')),
 '{}'::jsonb,'registrada_localmente','2026-09-28 12:00:00.000004+00';
UPDATE vec_contratacion_temporal.comunicacion_llamamiento_local x SET recibo_json=
 jsonb_build_object('ComunicacionRef',x.comunicacion_ref,'ReciboRef',
 'recibo:'||split_part(x.comunicacion_ref,':',2),'Solicitud',x.material_json->'solicitud',
 'Estado',x.estado,'RegistradaEn',x.registrada_en);
RESET ROLE;
