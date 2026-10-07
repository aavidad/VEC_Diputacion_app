\set ON_ERROR_STOP on
-- SOLO para el clon desechable de probar_custodia_firmado_pg18.sh. Sustituye
-- las fachadas AD3 que consumen la V3 (firma de CT, AD3-85; operación de
-- Documentos con recuperación, AD3-113, y lectura del original, AD3-60) por
-- dobles que devuelven un consumo sintético nuevo,
-- sin COSE, y crea los LOGIN de la prueba Go. Nunca se instala en otra base:
-- el consumo real lo cubren AD3 y sus propias pruebas.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_firma_documento_ct_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE d jsonb := convert_from(p_decision,'UTF8')::jsonb;
BEGIN
 RETURN QUERY SELECT d->>'decision_ref', d->>'recurso_ref', d->>'contexto_recurso_huella_sha256',
  encode(sha256(convert_to(gen_random_uuid()::text,'UTF8')),'hex'), 'auditoria:'||gen_random_uuid()::text, clock_timestamp(), true;
END $f$;
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.consumir_operacion_documentos_replay_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE c jsonb := convert_from(p_capacidad,'UTF8')::jsonb;
BEGIN
 RETURN QUERY SELECT c->>'decision_ref', c->>'efecto_ref', c->>'huella_efecto_sha256',
  encode(sha256(convert_to(gen_random_uuid()::text,'UTF8')),'hex'), 'auditoria:'||gen_random_uuid()::text, clock_timestamp(), true;
END $f$;
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.consumir_operacion_documentos_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE c jsonb := convert_from(p_capacidad,'UTF8')::jsonb;
BEGIN
 RETURN QUERY SELECT c->>'decision_ref', c->>'efecto_ref', c->>'huella_efecto_sha256',
  encode(sha256(convert_to(gen_random_uuid()::text,'UTF8')),'hex'), 'auditoria:'||gen_random_uuid()::text, clock_timestamp(), true;
END $f$;
RESET ROLE;
CREATE ROLE vec_e2e_custodia_ct LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS;
GRANT vec_contratacion_temporal_ejecutor TO vec_e2e_custodia_ct;
CREATE ROLE vec_e2e_custodia_documentos LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS;
GRANT vec_documentos_ejecutor TO vec_e2e_custodia_documentos;
COMMIT;
