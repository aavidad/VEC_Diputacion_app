\set ON_ERROR_STOP on
CREATE SCHEMA vec_autorizacion_atestada_v3;
CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_consulta_organizacion_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
 RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
 LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $fn$
 SELECT d->>'decision_ref',d->>'recurso_ref',d->>'contexto_recurso_huella_sha256',
  encode(sha256(p_capacidad||p_decision),'hex'),'auditoria:stub',clock_timestamp(),true
 FROM (SELECT convert_from(p_decision,'UTF8')::jsonb d) q
$fn$;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_personal_propietario;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_consulta_organizacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_personal_propietario;
SET ROLE vec_personal_propietario;
CREATE TABLE vec_personal.asignacion_dietas (dummy integer);
RESET ROLE;
CREATE ROLE vec_prueba_personal LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
GRANT vec_personal_ejecutor TO vec_prueba_personal WITH ADMIN FALSE,INHERIT TRUE,SET FALSE;
CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_importacion_organizacion_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
 RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
 LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $fn$
 SELECT d->>'decision_ref',d->>'recurso_ref',d->>'contexto_recurso_huella_sha256',
  encode(sha256(p_capacidad||p_decision),'hex'),'auditoria:stub',clock_timestamp(),true
 FROM (SELECT convert_from(p_decision,'UTF8')::jsonb d) q
$fn$;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_importacion_organizacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_personal_propietario;
