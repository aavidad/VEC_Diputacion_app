\set ON_ERROR_STOP on
-- Dobles mínimos de las dos fachadas de Autorización para probar únicamente
-- el preflight de LOGIN separados. No sustituyen la fuente V3 ni el PDP real.
CREATE ROLE vec_autorizacion_fuente NOLOGIN INHERIT NOBYPASSRLS;
CREATE ROLE vec_autorizacion_motivos_evaluador NOLOGIN INHERIT NOBYPASSRLS;
CREATE ROLE vec_plantillas_fuente_ensayo LOGIN INHERIT NOBYPASSRLS;
CREATE ROLE vec_plantillas_motivos_ensayo LOGIN INHERIT NOBYPASSRLS;
GRANT vec_autorizacion_fuente TO vec_plantillas_fuente_ensayo WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
GRANT vec_autorizacion_motivos_evaluador TO vec_plantillas_motivos_ensayo WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
CREATE SCHEMA vec_autorizacion;
REVOKE ALL ON SCHEMA vec_autorizacion FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion TO vec_autorizacion_fuente, vec_autorizacion_motivos_evaluador;
CREATE FUNCTION vec_autorizacion.obtener_instantanea(text,text)
RETURNS jsonb LANGUAGE sql AS $f$ SELECT '{}'::jsonb $f$;
CREATE FUNCTION vec_autorizacion.resolver_motivo_autorizacion_v2_historico(text,integer,text,text,timestamptz)
RETURNS jsonb LANGUAGE sql AS $f$ SELECT '{}'::jsonb $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.obtener_instantanea(text,text),
 vec_autorizacion.resolver_motivo_autorizacion_v2_historico(text,integer,text,text,timestamptz)
 FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion.obtener_instantanea(text,text) TO vec_autorizacion_fuente;
GRANT EXECUTE ON FUNCTION vec_autorizacion.resolver_motivo_autorizacion_v2_historico(text,integer,text,text,timestamptz)
 TO vec_autorizacion_motivos_evaluador;
