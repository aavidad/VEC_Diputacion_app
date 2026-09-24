\set ON_ERROR_STOP on
CREATE ROLE vec_autorizacion_atestada_v3_propietario NOLOGIN;
CREATE ROLE vec_personal_ejecutor NOLOGIN NOBYPASSRLS;
CREATE ROLE vec_personal_propietario NOLOGIN;
CREATE SCHEMA vec_autorizacion_atestada_v3 AUTHORIZATION vec_autorizacion_atestada_v3_propietario;
SET ROLE vec_autorizacion_atestada_v3_propietario;
CREATE TABLE vec_autorizacion_atestada_v3.clave_capacidad_version (
 audiencia_consumo text NOT NULL,
 CONSTRAINT clave_capacidad_version_audiencia_consumo_check
 CHECK (audiencia_consumo = ANY (ARRAY['vec_dietas_rutas_v1.acceso.v1'::text]))
);
CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
 p_perfil_mutacion text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
 RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
 LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb;
BEGIN
 c:=convert_from(p_capacidad,'UTF8')::jsonb;
 d:=convert_from(p_decision,'UTF8')::jsonb;
 IF p_perfil_mutacion IS DISTINCT FROM 'otro'
               AND p_perfil_mutacion IS DISTINCT FROM 'acceso_rutas_dietas'
 THEN RAISE EXCEPTION 'perfil'; END IF;
 IF ((true
           )
       ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'rechazado';
 END IF;
 IF (false
       )
       OR c ->> 'suite' <> 'VEC-AD-3-COSE-EDDSA-1'
 THEN RAISE EXCEPTION 'suite'; END IF;
 RETURN QUERY SELECT 'decision:stub','org:stub',repeat('a',64),repeat('b',64),'auditoria:stub',clock_timestamp(),true;
END $f$;
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_acceso_rutas_dietas_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,
 p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
 RETURNS TABLE(decision_ref text) LANGUAGE sql SECURITY DEFINER SET search_path=pg_catalog AS $f$
 SELECT 'decision:stub'::text
$f$;
RESET ROLE;
