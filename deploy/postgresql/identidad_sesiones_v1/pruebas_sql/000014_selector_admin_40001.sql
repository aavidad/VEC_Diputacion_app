\set ON_ERROR_STOP on
-- Regresión por inyección de fallos; sólo clon sintético, todo en ROLLBACK.
-- Requiere fixture privado auténtico de IS9 + configuración v4/asignación
-- gobernadas, NO una fuente fingida que devuelva un perfil favorable.
-- Parámetros psql del fixture: admin_test_login, admin_entorno, admin_host,
-- admin_audiencia, admin_cert_sha256, admin_ca_sha256, admin_autenticada,
-- admin_revocada, admin_crl_hasta, admin_certificado_hasta, admin_perfil_ref.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='2s';
SET LOCAL statement_timeout='10s';
CREATE FUNCTION pg_temp.assert_listado_40001(
 entorno text,host text,audiencia text,cert text,ca text,
 autenticada timestamptz,revocada timestamptz,crl_hasta timestamptz,certificado_hasta timestamptz)
RETURNS void LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
BEGIN
 BEGIN
  PERFORM vec_identidad_sesiones_v1.listar_perfiles_admin_auditado_v1(entorno,host,audiencia,cert,ca,autenticada,revocada,crl_hasta,certificado_hasta,'evento_'||repeat('a',32),'correlacion_'||repeat('b',32));
  RAISE EXCEPTION '40001 convertido en resultado';
 EXCEPTION WHEN serialization_failure THEN NULL;
 END;
END $f$;
GRANT EXECUTE ON FUNCTION pg_temp.assert_listado_40001(text,text,text,text,text,timestamptz,timestamptz,timestamptz,timestamptz) TO PUBLIC;
SAVEPOINT fallo_listado;
-- Sólo falla: no crea una asignación ni acredita identidad o éxito.
CREATE OR REPLACE FUNCTION vec_contexto_actor_v1.listar_admin_preperfil_propietaria_v1(p_cuenta text,p_persona text,p_audiencia text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
BEGIN RAISE EXCEPTION 'inyeccion: SSI en listado' USING ERRCODE='40001'; END $f$;
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.registrar_evento_admin_preperfil_v1(p_evento jsonb)
RETURNS TABLE(auditoria_ref text,secuencia numeric,huella_sha256 text,correlacion_ref text,registrada_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
BEGIN RAISE EXCEPTION '40001 de listado intentó registrar denegación'; END $f$;
SET SESSION AUTHORIZATION :"admin_test_login";
SELECT pg_temp.assert_listado_40001(:'admin_entorno',:'admin_host',:'admin_audiencia',:'admin_cert_sha256',:'admin_ca_sha256',:'admin_autenticada'::timestamptz,:'admin_revocada'::timestamptz,:'admin_crl_hasta'::timestamptz,:'admin_certificado_hasta'::timestamptz);
RESET SESSION AUTHORIZATION;
ROLLBACK TO SAVEPOINT fallo_listado;
SAVEPOINT fallo_cadena;
-- Se alcanza AD171 sólo mediante las fuentes originales y el fixture gobernado.
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.registrar_evento_admin_preperfil_v1(p_evento jsonb)
RETURNS TABLE(auditoria_ref text,secuencia numeric,huella_sha256 text,correlacion_ref text,registrada_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
BEGIN
 IF p_evento->>'resultado'='denegado' THEN RAISE EXCEPTION 'SSI de cadena convertido en denegación'; END IF;
 RAISE EXCEPTION 'inyeccion: SSI en cabeza AD171' USING ERRCODE='40001';
END $f$;
SET SESSION AUTHORIZATION :"admin_test_login";
SELECT pg_temp.assert_listado_40001(:'admin_entorno',:'admin_host',:'admin_audiencia',:'admin_cert_sha256',:'admin_ca_sha256',:'admin_autenticada'::timestamptz,:'admin_revocada'::timestamptz,:'admin_crl_hasta'::timestamptz,:'admin_certificado_hasta'::timestamptz);
RESET SESSION AUTHORIZATION;
ROLLBACK TO SAVEPOINT fallo_cadena;
-- El CAS semántico sí devuelve denegación auditada, sin efecto de elección.
CREATE TEMP TABLE is14_selecciones_antes AS SELECT count(*) AS total FROM vec_contexto_actor_v1.seleccion_admin_auditada_v1;
CREATE FUNCTION pg_temp.assert_cas_semantico(
 entorno text,host text,audiencia text,cert text,ca text,
 autenticada timestamptz,revocada timestamptz,crl_hasta timestamptz,certificado_hasta timestamptz,perfil text)
RETURNS void LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
DECLARE r record;
BEGIN
 SELECT * INTO STRICT r FROM vec_identidad_sesiones_v1.seleccionar_perfil_admin_auditado_v1(entorno,host,audiencia,cert,ca,autenticada,revocada,crl_hasta,certificado_hasta,perfil,18446744073709551614::numeric,'evento_'||repeat('c',32),'correlacion_'||repeat('d',32));
 IF r.resultado->>'estado' IS DISTINCT FROM 'denegado'
 OR r.resultado->>'motivo_ref' IS DISTINCT FROM 'seleccion_revision_obsoleta'
 OR r.auditoria_comun_ref IS NULL OR r.auditoria_comun_ref !~ '^aud_v3_p_[0-9a-f]{32}$'
 THEN RAISE EXCEPTION 'CAS semántico sin denegación auditada'; END IF;
END $f$;
GRANT EXECUTE ON FUNCTION pg_temp.assert_cas_semantico(text,text,text,text,text,timestamptz,timestamptz,timestamptz,timestamptz,text) TO PUBLIC;
SET SESSION AUTHORIZATION :"admin_test_login";
SELECT pg_temp.assert_cas_semantico(:'admin_entorno',:'admin_host',:'admin_audiencia',:'admin_cert_sha256',:'admin_ca_sha256',:'admin_autenticada'::timestamptz,:'admin_revocada'::timestamptz,:'admin_crl_hasta'::timestamptz,:'admin_certificado_hasta'::timestamptz,:'admin_perfil_ref');
RESET SESSION AUTHORIZATION;
DO $sin_efecto$
BEGIN
 IF (SELECT count(*) FROM vec_contexto_actor_v1.seleccion_admin_auditada_v1) IS DISTINCT FROM (SELECT total FROM pg_temp.is14_selecciones_antes)
 OR NOT EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 WHERE evento_ref='evento_'||repeat('c',32) AND resultado='denegado')
 THEN RAISE EXCEPTION 'CAS semántico cambió elección o perdió la auditoría'; END IF;
END $sin_efecto$;
ROLLBACK;
