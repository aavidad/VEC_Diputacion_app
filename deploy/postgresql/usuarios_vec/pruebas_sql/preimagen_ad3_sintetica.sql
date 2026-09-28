\set ON_ERROR_STOP on
-- Doble estructural V3: valida la composición SQL, no la criptografía COSE.
CREATE ROLE vec_autorizacion_atestada_v3_propietario NOLOGIN NOBYPASSRLS;
CREATE ROLE vec_contratacion_temporal_ejecutor NOLOGIN NOBYPASSRLS;
CREATE ROLE vec_usuarios_prueba LOGIN INHERIT NOBYPASSRLS;
CREATE ROLE vec_ct_prueba LOGIN INHERIT NOBYPASSRLS;
GRANT vec_usuarios_ejecutor TO vec_usuarios_prueba WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
GRANT vec_contratacion_temporal_ejecutor TO vec_ct_prueba WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
CREATE SCHEMA vec_autorizacion_atestada_v3 AUTHORIZATION vec_autorizacion_atestada_v3_propietario;
SET ROLE vec_autorizacion_atestada_v3_propietario;
CREATE TABLE vec_autorizacion_atestada_v3.clave_capacidad_version(
 audiencia_consumo text CONSTRAINT clave_capacidad_version_audiencia_consumo_check
 CHECK (audiencia_consumo = ANY (ARRAY['vec_contratacion_temporal.comunicaciones_expediente.consultar.v1'::text])));
CREATE TABLE vec_autorizacion_atestada_v3.prueba_consumos(nonce text PRIMARY KEY);
CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
 p_perfil_mutacion text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,
 consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb;
BEGIN
 c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 IF p_payload=convert_to('falsa','UTF8') THEN
  RAISE EXCEPTION 'V3 sintética rechazada' USING ERRCODE='42501'; END IF;
 IF p_perfil_mutacion IS DISTINCT FROM 'consulta_politica_ofertas_bolsa'
               AND p_perfil_mutacion IS DISTINCT FROM 'consulta_reincorporacion_titular_bolsa'
 THEN RAISE EXCEPTION 'perfil rechazado' USING ERRCODE='42501'; END IF;
 IF NOT (
           (p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_politica_ofertas_bolsa'
               OR p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_reincorporacion_titular_bolsa'
            )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_comunicaciones_expediente_ct'
           )
       ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501', MESSAGE='sesión denegada';
 END IF;
 IF NOT (
       (p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_comunicaciones_expediente_ct')
       )
       OR c ->> 'suite' <> 'VEC-AD-3-COSE-EDDSA-1'
 THEN RAISE EXCEPTION 'perfil rechazado' USING ERRCODE='42501'; END IF;
 INSERT INTO vec_autorizacion_atestada_v3.prueba_consumos(nonce)
 VALUES(convert_from(p_payload,'UTF8'));
 RETURN QUERY SELECT d->>'decision_ref',c->>'efecto_ref',
  c->>'huella_efecto_sha256',encode(sha256(convert_to(gen_random_uuid()::text,'UTF8')),'hex'),
  'aud_'||replace(gen_random_uuid()::text,'-',''),clock_timestamp(),true;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_lista_comunicaciones_ct_v3_atestada(
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,
 consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
 SELECT 'stub'::text,NULL::text,NULL::text,NULL::text,NULL::text,clock_timestamp(),false
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_lista_comunicaciones_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
RESET ROLE;
