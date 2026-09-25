\set ON_ERROR_STOP on
-- Solo PostgreSQL 18 desechable: fachada AD45 TEST-ONLY para instalar Personal
-- 000008 en el ensayo de postimagen de Dietas. No sustituye la migración AD3 real.
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_consumir_consulta_relacion_propia_dietas_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
 RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
 LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $fn$ SELECT NULL::text,NULL::text,NULL::text,NULL::text,NULL::text,NULL::timestamptz,false WHERE false $fn$;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_consumir_consulta_relacion_propia_dietas_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_personal_propietario;
