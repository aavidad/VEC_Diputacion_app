\set ON_ERROR_STOP on
-- Solo para base sintética desechable: sustituye el núcleo V3 por un testigo
-- que permite comprobar los contratos de las dos fachadas.
CREATE ROLE vec_contratacion_temporal_propietario NOLOGIN;
CREATE ROLE vec_contratacion_temporal_ejecutor NOLOGIN;
CREATE ROLE vec_contratacion_temporal_migrador NOLOGIN;
CREATE ROLE vec_contratacion_temporal_gobernador NOLOGIN;
CREATE ROLE vec_autorizacion_atestada_v3_propietario NOLOGIN;
CREATE ROLE vec_ct137_login LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
GRANT vec_contratacion_temporal_ejecutor TO vec_ct137_login WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
CREATE SCHEMA vec_contratacion_temporal AUTHORIZATION vec_contratacion_temporal_propietario;
CREATE SCHEMA vec_autorizacion_atestada_v3 AUTHORIZATION vec_autorizacion_atestada_v3_propietario;
GRANT USAGE ON SCHEMA vec_contratacion_temporal TO vec_contratacion_temporal_ejecutor;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_contratacion_temporal_propietario;
SET ROLE vec_contratacion_temporal_propietario;
CREATE TABLE vec_contratacion_temporal.expediente_version_integral(expediente_ref text,version bigint,agregado_json jsonb);
CREATE TABLE vec_contratacion_temporal.catalogo_plantillas_historia_v1(
 secuencia bigint,origen text,estado text,catalogo jsonb,version bigint,revision bigint,
 contenido_json_sha256 text,catalogo_huella_sha256 text,consumo_huella_sha256 text,
 auditoria_ref text,actor_ref text,provision_fuente_ref text,provision_aprobacion_ref text,
 registrada_en timestamptz,recibo_ref text);
CREATE TABLE vec_contratacion_temporal.catalogo_plantillas_provision_auditoria_v1(
 historia_secuencia bigint,auditoria_ref text,solicitud_huella_sha256 text,
 instalador_ref text,fuente_ref text,aprobacion_ref text,registrada_en timestamptz,recibo_ref text);
CREATE TABLE vec_contratacion_temporal.catalogo_plantillas_outbox_v1(
 historia_secuencia bigint,recibo_ref text,tipo text);
CREATE FUNCTION vec_contratacion_temporal.obtener_catalogo_plantillas_publicado_documental_v1(
 jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
RETURNS jsonb LANGUAGE sql AS $$ SELECT '{}'::jsonb $$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.obtener_catalogo_plantillas_publicado_documental_v1(
 jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.obtener_catalogo_plantillas_publicado_documental_v1(
 jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_contratacion_temporal_ejecutor;
INSERT INTO vec_contratacion_temporal.expediente_version_integral VALUES
 ('expediente:ct137-sintetico',1,'{"organizacion_ref":"organizacion:desarrollo:dipgra"}');
WITH c AS (SELECT '{"id":"vec.contratacion_temporal.plantillas_documentos","modulo_id":"contratacion_temporal","estado":"publicado","version":1,"revision":0,"entradas":[],"fuente_ref":"fuente:sintetica","aprobacion_ref":"aprobacion:sintetica"}'::jsonb AS j)
INSERT INTO vec_contratacion_temporal.catalogo_plantillas_historia_v1
SELECT 1,'bootstrap','publicado',c.j,1,0,encode(sha256(convert_to(c.j::text,'UTF8')),'hex'),repeat('a',64),repeat('b',64),
 'auditoria:sintetica','actor:sintetico','fuente:sintetica','aprobacion:sintetica',
 '2026-09-28 10:00:00+00'::timestamptz,NULL FROM c;
INSERT INTO vec_contratacion_temporal.catalogo_plantillas_provision_auditoria_v1
VALUES(1,'auditoria:sintetica',repeat('b',64),'actor:sintetico','fuente:sintetica','aprobacion:sintetica',
 '2026-09-28 10:00:00+00'::timestamptz,'recibo:sintetico');
RESET ROLE;
SET ROLE vec_autorizacion_atestada_v3_propietario;
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_plantillas_doc_ct_org_v3_atestada(
 jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql AS $$ SELECT 'decision:vieja'::text,'expediente:ct137-sintetico'::text,repeat('a',64),repeat('b',64),'auditoria:vieja'::text,now(),true $$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_plantillas_doc_ct_org_v3_atestada(
 jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_plantillas_doc_ct_org_v3_atestada(
 jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_contratacion_temporal_propietario;
CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
 text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql AS $$ SELECT 'decision:sintetica'::text,(convert_from($2,'UTF8')::jsonb)->>'efecto_ref',
 (convert_from($2,'UTF8')::jsonb)->>'huella_efecto_sha256',repeat('c',64),'auditoria:sintetica'::text,now(),true $$;
RESET ROLE;
