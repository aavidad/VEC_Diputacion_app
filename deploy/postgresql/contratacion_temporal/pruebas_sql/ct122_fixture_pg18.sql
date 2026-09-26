\set ON_ERROR_STOP on
-- Fixture de CT122 sobre la estructura real restaurada: rol de ejecución de
-- pruebas y doble explícito de la fachada AD3-87 (prueba la transacción CT,
-- no la criptografía V3). Base desechable: se confirma.
CREATE ROLE vec_ct122_runtime LOGIN INHERIT IN ROLE vec_contratacion_temporal_ejecutor;
GRANT CONNECT ON DATABASE postgres TO vec_ct122_runtime;
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_cancelacion_expediente_ct_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE d jsonb:=convert_from(p_decision,'UTF8')::jsonb;
BEGIN
 IF d->>'accion'<>'contratacion_temporal.expediente.cancelar' THEN RAISE EXCEPTION 'doble: cancelación denegada' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT d->>'decision_ref', d->>'recurso_ref', d->>'contexto_recurso_huella_sha256', encode(sha256(p_capacidad||p_decision),'hex'),
   'aud_v3_'||md5(p_capacidad||p_decision), clock_timestamp(), true;
END $f$;
