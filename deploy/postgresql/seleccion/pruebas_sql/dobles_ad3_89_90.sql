-- Dobles de ensayo de las fachadas AD3-89 y AD3-90 para recorrer Selección
-- 000001 sin criptografía: conservan firma, propietario y ACL (CREATE OR
-- REPLACE) y devuelven una decisión única por llamada. Una capacidad con
-- "repetida" simula un consumo ya hecho. La autorización real se prueba
-- aparte con las fachadas instaladas (rechazo del material).
SET ROLE vec_autorizacion_atestada_v3_propietario;
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_solicitud_propia_seleccion_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
 SELECT 'decision:doble:'||gen_random_uuid()::text, convert_from(p_decision,'UTF8')::jsonb->>'recurso_ref', repeat('a',64), repeat('b',64), 'auditoria:doble', now(),
        NOT (convert_from(p_capacidad,'UTF8')::jsonb ? 'repetida')
$f$;
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_solicitudes_seleccion_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
 SELECT 'decision:doble:'||gen_random_uuid()::text, convert_from(p_decision,'UTF8')::jsonb->>'recurso_ref', repeat('a',64), repeat('b',64), 'auditoria:doble', now(),
        NOT (convert_from(p_capacidad,'UTF8')::jsonb ? 'repetida')
$f$;
RESET ROLE;
