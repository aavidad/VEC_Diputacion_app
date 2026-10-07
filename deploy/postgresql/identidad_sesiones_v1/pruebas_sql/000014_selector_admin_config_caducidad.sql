\set ON_ERROR_STOP on
-- Sólo clon sintético; parámetros del mismo fixture gobernado que 40001.sql.
-- No crea personas/asignaciones: un LOGIN temporal usa la configuración técnica
-- del fixture, con vigencia breve, y las fuentes IS9/AUT24/CA31 originales.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='2s';
SET LOCAL statement_timeout='10s';
CREATE ROLE vec_prueba_is14_caducidad LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
GRANT vec_identidad_sesiones_v1_admin_preperfil TO vec_prueba_is14_caducidad WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
-- Únicamente añade demora al cuerpo real; su resultado y guardas se conservan.
DO $demora$
DECLARE cuerpo text;marca text:=' RETURN jsonb_build_object(''revision''';
BEGIN
 SELECT pg_get_functiondef(to_regprocedure('vec_contexto_actor_v1.listar_admin_preperfil_propietaria_v1(text,text,text)')) INTO STRICT cuerpo;
 IF (length(cuerpo)-length(replace(cuerpo,marca,'')))/length(marca)<>1 THEN RAISE EXCEPTION 'marca de demora no única'; END IF;
 EXECUTE replace(cuerpo,marca,' PERFORM pg_catalog.pg_sleep(2);'||chr(10)||marca);
END $demora$;
CREATE TEMP TABLE is14_selecciones_antes AS SELECT count(*) AS total FROM vec_contexto_actor_v1.seleccion_admin_auditada_v1;
CREATE FUNCTION pg_temp.assert_config_caducada(
 entorno text,host text,audiencia text,cert text,ca text,
 autenticada timestamptz,revocada timestamptz,crl_hasta timestamptz,certificado_hasta timestamptz,perfil text)
RETURNS void LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
DECLARE r record;
BEGIN
 SELECT * INTO STRICT r FROM vec_identidad_sesiones_v1.seleccionar_perfil_admin_auditado_v1(entorno,host,audiencia,cert,ca,autenticada,revocada,crl_hasta,certificado_hasta,perfil,0,'evento_'||repeat('e',32),'correlacion_'||repeat('f',32));
 IF r.resultado->>'estado' IS DISTINCT FROM 'denegado'
 OR r.auditoria_comun_ref IS NULL OR r.auditoria_comun_ref !~ '^aud_v3_p_[0-9a-f]{32}$'
 THEN RAISE EXCEPTION 'configuración caducada permitió elección'; END IF;
END $f$;
GRANT EXECUTE ON FUNCTION pg_temp.assert_config_caducada(text,text,text,text,text,timestamptz,timestamptz,timestamptz,timestamptz,text) TO PUBLIC;
SET LOCAL ROLE vec_identidad_sesiones_v1_propietario;
INSERT INTO vec_identidad_sesiones_v1.config_runtime_admin_preperfil_v1(identidad_login,proceso,entorno,host_admin,audiencia,vigente_hasta)
SELECT 'vec_prueba_is14_caducidad',proceso,entorno,host_admin,audiencia,clock_timestamp()+interval '1 second'
FROM vec_identidad_sesiones_v1.config_runtime_admin_preperfil_v1 WHERE identidad_login=:'admin_test_login';
RESET ROLE;
SET SESSION AUTHORIZATION vec_prueba_is14_caducidad;
SELECT pg_temp.assert_config_caducada(:'admin_entorno',:'admin_host',:'admin_audiencia',:'admin_cert_sha256',:'admin_ca_sha256',:'admin_autenticada'::timestamptz,:'admin_revocada'::timestamptz,:'admin_crl_hasta'::timestamptz,:'admin_certificado_hasta'::timestamptz,:'admin_perfil_ref');
RESET SESSION AUTHORIZATION;
DO $sin_efecto$
BEGIN
 IF (SELECT count(*) FROM vec_contexto_actor_v1.seleccion_admin_auditada_v1) IS DISTINCT FROM (SELECT total FROM pg_temp.is14_selecciones_antes)
 OR NOT EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 WHERE evento_ref='evento_'||repeat('e',32) AND resultado='denegado')
 OR EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 WHERE evento_ref='evento_'||repeat('e',32) AND resultado='permitido')
 THEN RAISE EXCEPTION 'caducidad config dejó elección o auditoría favorable'; END IF;
END $sin_efecto$;
ROLLBACK;
