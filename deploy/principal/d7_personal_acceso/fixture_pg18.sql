\set ON_ERROR_STOP on
REVOKE ALL ON DATABASE postgres FROM PUBLIC;
CREATE ROLE vec_personal_propietario NOLOGIN;
CREATE ROLE vec_personal_d7_ejecutor NOLOGIN INHERIT;
CREATE ROLE vec_personal_registrador_frontera NOLOGIN INHERIT;
GRANT CONNECT ON DATABASE postgres TO vec_personal_registrador_frontera;
CREATE SCHEMA vec_personal AUTHORIZATION vec_personal_propietario;
GRANT USAGE ON SCHEMA vec_personal TO vec_personal_d7_ejecutor,vec_personal_registrador_frontera;
CREATE SCHEMA vec_autorizacion;
CREATE SCHEMA vec_contratacion_temporal;
CREATE FUNCTION vec_contratacion_temporal.efecto_sensible() RETURNS boolean LANGUAGE sql AS 'SELECT true';
REVOKE ALL ON FUNCTION vec_contratacion_temporal.efecto_sensible() FROM PUBLIC;
CREATE TABLE vec_autorizacion.version_rol(version_rol_ref text PRIMARY KEY,rol_id text NOT NULL);
CREATE TABLE vec_autorizacion.asignacion_perfil(asignacion_ref text PRIMARY KEY,version_rol_ref text NOT NULL,version integer NOT NULL,huella_sha256 text NOT NULL,documento jsonb NOT NULL);
CREATE TABLE vec_autorizacion.asignacion_perfil_actual(asignacion_ref text PRIMARY KEY,actualizada_por text NOT NULL,acto_ref text NOT NULL);
INSERT INTO vec_autorizacion.version_rol VALUES('rol:dietas_r1d_provisional:v1','dietas_r1d_provisional');
INSERT INTO vec_autorizacion.asignacion_perfil VALUES
 ('asignacion:dietas_r1d_sintetica:v3','rol:dietas_r1d_provisional:v1',3,repeat('a',64),
  '{"asignacion_id":"dietas_r1d_sintetica","version_rol_ref":"rol:dietas_r1d_provisional:v1","version":"3","estado":"activa","emitida_por":"administracion:f4:reactivacion-dietas-r1d","vigente_desde":"2026-09-24T00:00:00Z","vigente_hasta":"2099-01-01T00:00:00Z"}');
INSERT INTO vec_autorizacion.asignacion_perfil_actual VALUES
 ('asignacion:dietas_r1d_sintetica:v3','administracion:f4:reactivacion-dietas-r1d','acto:f4:reactivacion-dietas-r1d:20260924');
CREATE ROLE vec_dietas_r1d_registro_identidad_desarrollo LOGIN;
CREATE ROLE vec_dietas_r1d_revalidacion_identidad_desarrollo LOGIN;
CREATE ROLE vec_dietas_r1d_contexto_desarrollo LOGIN;
CREATE ROLE vec_dietas_r1d_fuente_autorizacion_desarrollo LOGIN;
CREATE ROLE vec_dietas_r1d_registro_autorizacion_desarrollo LOGIN;
CREATE ROLE vec_dietas_r1d_motivos_desarrollo LOGIN;
CREATE ROLE vec_dietas_r1d_dietas_desarrollo LOGIN;
CREATE ROLE vec_dietas_r1d_personal_desarrollo LOGIN;
SET ROLE vec_personal_propietario;
CREATE FUNCTION vec_personal.registrar_asignacion_dietas_inicial_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
RETURNS boolean LANGUAGE sql SECURITY DEFINER AS 'SELECT true';
CREATE FUNCTION vec_personal.consultar_asignacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
RETURNS boolean LANGUAGE sql SECURITY DEFINER AS 'SELECT true';
CREATE FUNCTION vec_personal.corregir_asignacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
RETURNS boolean LANGUAGE sql SECURITY DEFINER AS 'SELECT true';
CREATE FUNCTION vec_personal.corregir_grupo_dieta_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
RETURNS boolean LANGUAGE sql SECURITY DEFINER AS 'SELECT true';
CREATE FUNCTION vec_personal.registrar_auditoria_frontera_asignacion_dietas_v1(text,text,text,text,text,text,text,smallint)
RETURNS boolean LANGUAGE sql SECURITY DEFINER AS 'SELECT true';
REVOKE ALL ON ALL FUNCTIONS IN SCHEMA vec_personal FROM PUBLIC;
GRANT EXECUTE ON FUNCTION
 vec_personal.registrar_asignacion_dietas_inicial_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_personal.consultar_asignacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_personal.corregir_asignacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_personal.corregir_grupo_dieta_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_personal_d7_ejecutor;
GRANT EXECUTE ON FUNCTION vec_personal.registrar_auditoria_frontera_asignacion_dietas_v1(text,text,text,text,text,text,text,smallint)
 TO vec_personal_registrador_frontera;
RESET ROLE;
