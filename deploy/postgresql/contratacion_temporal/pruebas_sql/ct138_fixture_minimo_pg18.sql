\set ON_ERROR_STOP on
-- Base sintética desechable. Solo comprueba CT138; el ensayo sobre la
-- preimagen restaurada de la principal corresponde al canal de Dirección.
CREATE ROLE vec_contratacion_temporal_propietario NOLOGIN;
CREATE ROLE vec_contratacion_temporal_ejecutor NOLOGIN;
CREATE ROLE vec_contratacion_temporal_migrador NOLOGIN;
CREATE ROLE vec_autorizacion_atestada_v3_propietario NOLOGIN;
CREATE ROLE vec_ct138_login LOGIN NOSUPERUSER NOBYPASSRLS;
GRANT vec_contratacion_temporal_ejecutor TO vec_ct138_login WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
CREATE SCHEMA vec_contratacion_temporal AUTHORIZATION vec_contratacion_temporal_propietario;
CREATE SCHEMA vec_autorizacion_atestada_v3 AUTHORIZATION vec_autorizacion_atestada_v3_propietario;
GRANT USAGE ON SCHEMA vec_contratacion_temporal TO vec_contratacion_temporal_ejecutor;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_contratacion_temporal_propietario;

SET ROLE vec_contratacion_temporal_propietario;
CREATE TABLE vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6(
    clave_idempotencia uuid PRIMARY KEY, situacion text, solicitud_json jsonb, recibo_json jsonb);
CREATE TABLE vec_contratacion_temporal.comunicacion_llamamiento_local(
    comunicacion_ref text PRIMARY KEY, organizacion_ref text, expediente_ref text,
    llamamiento_ref text, estado text, version_resultante numeric,
    material_json jsonb, seleccion_clave uuid);
CREATE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1()
RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'historia inmutable'; END $$;
CREATE FUNCTION vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(s jsonb, claves text[])
RETURNS boolean LANGUAGE sql IMMUTABLE AS $$
    SELECT jsonb_typeof(s)='object' AND
      (SELECT array_agg(k ORDER BY k) FROM jsonb_object_keys(s) k)=
      (SELECT array_agg(k ORDER BY k) FROM unnest(claves) k)
$$;
RESET ROLE;

SET ROLE vec_autorizacion_atestada_v3_propietario;
CREATE TABLE vec_autorizacion_atestada_v3.ct138_consumos_sinteticos(
    auditoria_ref text PRIMARY KEY, efecto_ref text NOT NULL);
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_respuesta_recibida_rrhh_v3_atestada(
    p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,
    p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(consumo_nuevo boolean,efecto_ref text,huella_efecto_sha256 text,
    auditoria_ref text,decision_ref text,consumo_huella_sha256 text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $funcion$
DECLARE d jsonb; v_auditoria text;
BEGIN
    d:=convert_from(p_decision,'UTF8')::jsonb;
    v_auditoria:='auditoria:'||gen_random_uuid()::text;
    INSERT INTO vec_autorizacion_atestada_v3.ct138_consumos_sinteticos VALUES(v_auditoria,d->>'recurso_ref');
    RETURN QUERY SELECT true,d->>'recurso_ref',d->>'contexto_recurso_huella_sha256',
        v_auditoria,'decision:'||gen_random_uuid()::text,
        encode(sha256(convert_to(v_auditoria,'UTF8')),'hex');
END $funcion$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_respuesta_recibida_rrhh_v3_atestada(
    bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_respuesta_recibida_rrhh_v3_atestada(
    bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_contratacion_temporal_propietario;
RESET ROLE;

SET ROLE vec_contratacion_temporal_propietario;
INSERT INTO vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 VALUES
 ('11111111-1111-4111-8111-111111111111','confirmada',
  '{"organizacion_ref":"org:ct138","expediente_ref":"exp:ct138"}',
  jsonb_build_object('organizacion_ref','org:ct138','expediente_ref','exp:ct138',
    'llamamiento_ref','llamamiento:legado','recibo_ref','recibo:seleccion-legado',
    'seleccion_ref','hmac-sha256:vec.contratacion-temporal.seleccion/v1:'||repeat('a',64),
    'propuesta_generada',true)),
 ('22222222-2222-4222-8222-222222222222','confirmada',
  '{"organizacion_ref":"org:ct138","expediente_ref":"exp:ct138"}',
  jsonb_build_object('organizacion_ref','org:ct138','expediente_ref','exp:ct138',
    'llamamiento_ref','llamamiento:nuevo','recibo_ref','recibo:seleccion-nuevo',
    'seleccion_ref','hmac-sha256:vec.contratacion-temporal.seleccion/v1:'||repeat('b',64),
    'propuesta_generada',true)),
 ('33333333-3333-4333-8333-333333333333','confirmada',
  '{"organizacion_ref":"org:ct138","expediente_ref":"exp:ct138"}',
  jsonb_build_object('organizacion_ref','org:ct138','expediente_ref','exp:ct138',
    'llamamiento_ref','llamamiento:concurrente','recibo_ref','recibo:seleccion-concurrente',
    'seleccion_ref','hmac-sha256:vec.contratacion-temporal.seleccion/v1:'||repeat('c',64),
    'propuesta_generada',true));
INSERT INTO vec_contratacion_temporal.comunicacion_llamamiento_local VALUES
 ('comunicacion:legada','org:ct138','exp:ct138','llamamiento:legado','registrada_localmente',2,
  '{"solicitud":{"PruebaEntregaRef":"recibo:seleccion-legado"}}','11111111-1111-4111-8111-111111111111'),
 ('comunicacion:nueva','org:ct138','exp:ct138','llamamiento:nuevo','registrada_localmente',2,
  '{"solicitud":{"PruebaEntregaRef":"recibo:seleccion-nuevo"}}','22222222-2222-4222-8222-222222222222'),
 ('comunicacion:nueva-otra','org:ct138','exp:ct138','llamamiento:nuevo','registrada_localmente',2,
  '{"solicitud":{"PruebaEntregaRef":"recibo:seleccion-nuevo"}}','22222222-2222-4222-8222-222222222222'),
 ('comunicacion:concurrente','org:ct138','exp:ct138','llamamiento:concurrente','registrada_localmente',2,
  '{"solicitud":{"PruebaEntregaRef":"recibo:seleccion-concurrente"}}','33333333-3333-4333-8333-333333333333');
RESET ROLE;
