\set ON_ERROR_STOP on
-- Preimagen sintética de estructura para comprobar la inserción dinámica AD3-101.
-- No suplanta las pruebas criptográficas del núcleo completo.
CREATE ROLE vec_autorizacion_atestada_v3_propietario NOLOGIN;
CREATE ROLE vec_bolsa_llamamientos_propietario NOLOGIN;
CREATE ROLE vec_bolsa_llamamientos_ejecutor NOLOGIN;
CREATE ROLE vec_ad3_101_rrhh LOGIN INHERIT;
GRANT vec_bolsa_llamamientos_ejecutor TO vec_ad3_101_rrhh;
CREATE SCHEMA vec_autorizacion_atestada_v3 AUTHORIZATION vec_autorizacion_atestada_v3_propietario;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_bolsa_llamamientos_propietario;
SET ROLE vec_autorizacion_atestada_v3_propietario;
CREATE TABLE vec_autorizacion_atestada_v3.clave_capacidad_version(
 audiencia_consumo text NOT NULL,
 CONSTRAINT clave_capacidad_version_audiencia_consumo_check
 CHECK (audiencia_consumo = ANY (ARRAY['vec_bolsa_llamamientos.politica_ofertas.publicar.v1'::text,
  'vec_bolsa_llamamientos.politica_ofertas.consultar.v1'::text])));
CREATE TABLE vec_autorizacion_atestada_v3.atestacion_decision_v3(capacidad_canonica bytea NOT NULL);
CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
 p_perfil_mutacion text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,
 auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb;
BEGIN
 c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 IF p_perfil_mutacion IS DISTINCT FROM 'bolsa_llamamiento'
               AND p_perfil_mutacion IS DISTINCT FROM 'consulta_politica_ofertas_bolsa'
 THEN RAISE EXCEPTION 'perfil denegado' USING ERRCODE='42501'; END IF;
 IF (p_perfil_mutacion IS NOT DISTINCT FROM 'bolsa_llamamiento'
               OR p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_politica_ofertas_bolsa'
    ) AND NOT pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER')
 THEN RAISE EXCEPTION 'sesión denegada' USING ERRCODE='42501'; END IF;
 IF NOT (
        (p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_politica_ofertas_bolsa'
         AND c->>'operacion' IS NOT DISTINCT FROM 'bolsa.politica_ofertas.consultar'
       )
       OR c ->> 'suite' <> 'VEC-AD-3-COSE-EDDSA-1'
    ) THEN RAISE EXCEPTION 'decisión denegada' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT 'decision:'||gen_random_uuid()::text,c->>'efecto_ref',c->>'huella_efecto_sha256',
 repeat('a',64),'auditoria:'||gen_random_uuid()::text,clock_timestamp(),true;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_consulta_politica_ofertas_bolsa_v3_atestada(
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,
 auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
 SELECT 'decision:pre',NULL::text,NULL::text,NULL::text,'auditoria:pre',clock_timestamp(),false
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_consulta_politica_ofertas_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
RESET ROLE;
