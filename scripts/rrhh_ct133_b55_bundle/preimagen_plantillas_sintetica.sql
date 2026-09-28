\set ON_ERROR_STOP on
-- SOLO para un contenedor PG18 vacío. Núcleo AD3 anterior sintético;
-- CT131/CT133/CT135/CT137 y AD3-99/100 se instalan desde migraciones reales.
DO $pre$
BEGIN
 IF current_setting('server_version_num')::integer / 100 <> 1800
    OR to_regnamespace('vec_contratacion_temporal') IS NOT NULL
 THEN RAISE EXCEPTION 'preimagen de plantillas: requiere PG18 vacío'; END IF;
END $pre$;
CREATE ROLE vec_contratacion_temporal_propietario NOLOGIN NOINHERIT NOBYPASSRLS;
CREATE ROLE vec_contratacion_temporal_migrador NOLOGIN NOINHERIT NOBYPASSRLS;
CREATE ROLE vec_contratacion_temporal_ejecutor NOLOGIN INHERIT NOBYPASSRLS;
CREATE ROLE vec_contratacion_temporal_gobernador NOLOGIN INHERIT NOBYPASSRLS;
CREATE ROLE vec_autorizacion_atestada_v3_propietario NOLOGIN NOINHERIT NOBYPASSRLS;
CREATE ROLE vec_plantillas_migrador_ensayo LOGIN INHERIT NOBYPASSRLS;
CREATE ROLE vec_plantillas_ejecutor_ensayo LOGIN INHERIT NOBYPASSRLS;
GRANT vec_contratacion_temporal_propietario TO vec_contratacion_temporal_migrador WITH INHERIT FALSE, SET FALSE, ADMIN FALSE;
GRANT vec_contratacion_temporal_migrador TO vec_plantillas_migrador_ensayo WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
GRANT vec_contratacion_temporal_ejecutor TO vec_plantillas_ejecutor_ensayo WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
CREATE SCHEMA vec_contratacion_temporal AUTHORIZATION vec_contratacion_temporal_propietario;
CREATE SCHEMA vec_autorizacion_atestada_v3 AUTHORIZATION vec_autorizacion_atestada_v3_propietario;
REVOKE ALL ON SCHEMA vec_contratacion_temporal, vec_autorizacion_atestada_v3 FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_contratacion_temporal TO vec_contratacion_temporal_ejecutor;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_contratacion_temporal_propietario;

SET ROLE vec_contratacion_temporal_propietario;
CREATE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1()
RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
BEGIN RAISE EXCEPTION 'historia inmutable' USING ERRCODE='55000'; END $f$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1() FROM PUBLIC;
RESET ROLE;

-- Doble del núcleo AD3: solo permite comprobar la envoltura organizativa de
-- AD3-99. No emite ni verifica una decisión V3, firma o consumo real.
SET ROLE vec_autorizacion_atestada_v3_propietario;
CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
 text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,
 consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql AS $f$
DECLARE c jsonb;
BEGIN
 c:=convert_from($2,'UTF8')::jsonb;
 RETURN QUERY SELECT 'decision:sintetica'::text,c->>'efecto_ref',c->>'huella_efecto_sha256',
  repeat('b',64)::text,'auditoria:sintetica'::text,clock_timestamp(),true;
END $f$;
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_catalogo_plantillas_ct_v3_atestada(
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,
 consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql SECURITY DEFINER AS $f$ SELECT 'decision:sintetica'::text,
 'vec.contratacion_temporal.plantillas_documentos'::text,repeat('a',64)::text,
 repeat('b',64)::text,'auditoria:sintetica'::text,clock_timestamp(),true $f$;
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_plantillas_doc_ct_v3_atestada(
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,
 consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql SECURITY DEFINER AS $f$ SELECT 'decision:sintetica'::text,
 'expediente:sintetico'::text,repeat('a',64)::text,
 repeat('b',64)::text,'auditoria:sintetica'::text,clock_timestamp(),true $f$;
REVOKE ALL ON FUNCTION
 vec_autorizacion_atestada_v3.registrar_y_consumir_catalogo_plantillas_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_autorizacion_atestada_v3.registrar_y_consumir_plantillas_doc_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 FROM PUBLIC;
GRANT EXECUTE ON FUNCTION
 vec_autorizacion_atestada_v3.registrar_y_consumir_catalogo_plantillas_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_autorizacion_atestada_v3.registrar_y_consumir_plantillas_doc_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_contratacion_temporal_propietario;
RESET ROLE;
