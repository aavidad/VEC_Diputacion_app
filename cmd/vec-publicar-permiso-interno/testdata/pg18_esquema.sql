\set ON_ERROR_STOP on
-- Solo para vec_permiso_v3, DB desechable. Replica nombres y tipos leídos
-- por el publicador; las garantías RLS/ACL requieren migraciones canónicas.
CREATE SCHEMA vec_identidad_sesiones_v1;
CREATE TABLE vec_identidad_sesiones_v1.politica_certificado_personal_desarrollo_v1 (
 singleton boolean PRIMARY KEY, politica_ref text NOT NULL, huella_sha256 text NOT NULL,
 retirar_en timestamptz(6) NOT NULL, activa boolean NOT NULL, registrada_en timestamptz(6) NOT NULL);
INSERT INTO vec_identidad_sesiones_v1.politica_certificado_personal_desarrollo_v1
VALUES (true,'pga_0123456789abcdefghijkl',repeat('a',64),'2026-10-31 23:00:00+00',true,'2026-09-01 00:00:00+00');

CREATE SCHEMA vec_contexto_actor_v1;
CREATE TABLE vec_contexto_actor_v1.proyeccion_cuenta_versiones (
 cuenta_ref text, version numeric(20,0), estado text,
 vigente_desde timestamptz, vigente_hasta timestamptz, PRIMARY KEY(cuenta_ref,version));
CREATE TABLE vec_contexto_actor_v1.proyeccion_cuenta_actual (
 cuenta_ref text PRIMARY KEY, version numeric(20,0));
CREATE TABLE vec_contexto_actor_v1.persona_versiones (
 persona_ref text, version numeric(20,0), estado text,
 vigente_desde timestamptz, vigente_hasta timestamptz, PRIMARY KEY(persona_ref,version));
CREATE TABLE vec_contexto_actor_v1.persona_actual (
 persona_ref text PRIMARY KEY, version numeric(20,0));
CREATE TABLE vec_contexto_actor_v1.perfil_versiones (
 perfil_ref text, version numeric(20,0), persona_ref text, estado text,
 vigente_desde timestamptz, vigente_hasta timestamptz, PRIMARY KEY(perfil_ref,version));
CREATE TABLE vec_contexto_actor_v1.perfil_actual (
 perfil_ref text PRIMARY KEY, version numeric(20,0));
CREATE TABLE vec_contexto_actor_v1.vinculo_contexto_versiones (
 vinculo_ref text, version numeric(20,0), cuenta_ref text, perfil_ref text,
 persona_ref text, estado text, vigente_desde timestamptz, vigente_hasta timestamptz,
 PRIMARY KEY(vinculo_ref,version));
CREATE TABLE vec_contexto_actor_v1.vinculo_contexto_actual (
 vinculo_ref text PRIMARY KEY, version numeric(20,0));
INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_versiones VALUES
 ('cta_0123456789abcdefghijkl',1,'activo','2026-09-01','2027-01-01');
INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_actual VALUES ('cta_0123456789abcdefghijkl',1);
INSERT INTO vec_contexto_actor_v1.persona_versiones VALUES
 ('per_0123456789abcdefghijkl',1,'activo','2026-09-01','2027-01-01');
INSERT INTO vec_contexto_actor_v1.persona_actual VALUES ('per_0123456789abcdefghijkl',1);
INSERT INTO vec_contexto_actor_v1.perfil_versiones VALUES
 ('prf_0123456789abcdefghijkl',1,'per_0123456789abcdefghijkl','activo','2026-09-01','2027-01-01'),
 ('prf_alto123456789abcdefghijk',1,'per_0123456789abcdefghijkl','activo','2026-09-01','2027-01-01');
INSERT INTO vec_contexto_actor_v1.perfil_actual VALUES
 ('prf_0123456789abcdefghijkl',1),('prf_alto123456789abcdefghijk',1);
INSERT INTO vec_contexto_actor_v1.vinculo_contexto_versiones VALUES
 ('vca_0123456789abcdefghijkl',1,'cta_0123456789abcdefghijkl',
  'prf_0123456789abcdefghijkl','per_0123456789abcdefghijkl','activo','2026-09-01','2027-01-01');
INSERT INTO vec_contexto_actor_v1.vinculo_contexto_actual VALUES ('vca_0123456789abcdefghijkl',1);
CREATE TABLE vec_contexto_actor_v1.organizacion_versiones (
 organizacion_ref text, version numeric(20,0), procedencia_ref text,
 procedencia_version numeric(20,0), procedencia_huella_sha256 text,
 procedencia_autoridad text, estado text, vigente_desde timestamptz,
 vigente_hasta timestamptz, PRIMARY KEY(organizacion_ref,version));
CREATE TABLE vec_contexto_actor_v1.organizacion_actual (
 organizacion_ref text PRIMARY KEY, version numeric(20,0));
CREATE TABLE vec_contexto_actor_v1.vinculo_corporativo_versiones (
 vinculo_corporativo_ref text, version numeric(20,0), cuenta_ref text,
 cuenta_version numeric(20,0), persona_ref text, persona_version numeric(20,0),
 perfil_ref text, perfil_version numeric(20,0), vinculo_contexto_ref text,
 vinculo_contexto_version numeric(20,0), organizacion_ref text,
 organizacion_version numeric(20,0), organizacion_procedencia_ref text,
 organizacion_procedencia_version numeric(20,0), organizacion_procedencia_huella_sha256 text,
 organizacion_procedencia_autoridad text, superficie text, uso text,
 procedencia_autoridad text, estado text, vigente_desde timestamptz,
 vigente_hasta timestamptz, PRIMARY KEY(vinculo_corporativo_ref,version));
CREATE TABLE vec_contexto_actor_v1.vinculo_corporativo_actual (
 cuenta_ref text, superficie text, uso text, vinculo_corporativo_ref text,
 version numeric(20,0), PRIMARY KEY(cuenta_ref,superficie,uso));
INSERT INTO vec_contexto_actor_v1.organizacion_versiones VALUES
 ('org_0123456789abcdefghijkl',1,'fuente:org:prueba',1,repeat('a',64),
  'autoridad_maestra_acreditada','activo','2026-09-01','2027-01-01');
INSERT INTO vec_contexto_actor_v1.organizacion_actual VALUES
 ('org_0123456789abcdefghijkl',1);
INSERT INTO vec_contexto_actor_v1.vinculo_corporativo_versiones VALUES
 ('vcr_0123456789abcdefghijkl',1,'cta_0123456789abcdefghijkl',1,
  'per_0123456789abcdefghijkl',1,'prf_0123456789abcdefghijkl',1,
  'vca_0123456789abcdefghijkl',1,'org_0123456789abcdefghijkl',1,
  'fuente:org:prueba',1,repeat('a',64),'autoridad_maestra_acreditada',
  'interna_corporativa','consulta_rrhh','autoridad_maestra_acreditada',
  'activo','2026-09-01','2027-01-01');
INSERT INTO vec_contexto_actor_v1.vinculo_corporativo_actual VALUES
 ('cta_0123456789abcdefghijkl','interna_corporativa','consulta_rrhh',
  'vcr_0123456789abcdefghijkl',1);

CREATE SCHEMA vec_autorizacion;
CREATE TABLE vec_autorizacion.version_rol (
 version_rol_ref text PRIMARY KEY, rol_id text, version bigint, huella_sha256 text,
 publicada_en timestamptz(6), documento jsonb);
CREATE TABLE vec_autorizacion.control_vigencia_version_rol (
 version_rol_ref text, revision numeric(20,0), estado text, huella_sha256 text,
 actualizado_en timestamptz(6), documento jsonb, PRIMARY KEY(version_rol_ref,revision));
CREATE TABLE vec_autorizacion.control_vigencia_version_rol_actual (
 version_rol_ref text PRIMARY KEY, revision numeric(20,0), actualizada_en timestamptz(6),
 actualizada_por text, acto_ref text);
CREATE TABLE vec_autorizacion.asignacion_perfil (
 asignacion_ref text PRIMARY KEY, asignacion_id text, version bigint, perfil_activo_ref text,
 principal_id text, version_rol_ref text, huella_sha256 text, emitida_en timestamptz(6), documento jsonb);
CREATE TABLE vec_autorizacion.asignacion_perfil_actual (
 perfil_activo_ref text PRIMARY KEY, asignacion_ref text, actualizada_en timestamptz(6),
 actualizada_por text, acto_ref text);
INSERT INTO vec_autorizacion.version_rol VALUES
 ('rol:rrhh:alto:v1','rrhh:alto',1,repeat('a',64),'2026-09-01',
  jsonb_build_object('estado','publicada','concesiones',jsonb_build_array(jsonb_build_object(
   'accion','contratacion_temporal.expediente.consultar','modulo_id','contratacion_temporal',
   'tipo_recurso','expediente_contratacion_temporal','garantia_minima','alto'))));
INSERT INTO vec_autorizacion.control_vigencia_version_rol VALUES
 ('rol:rrhh:alto:v1',1,'habilitada',repeat('b',64),'2026-09-01','{}');
INSERT INTO vec_autorizacion.control_vigencia_version_rol_actual VALUES
 ('rol:rrhh:alto:v1',1,'2026-09-01','prueba','acto:prueba:control');
INSERT INTO vec_autorizacion.asignacion_perfil VALUES
 ('asignacion:rrhh:alto:v1','rrhh:alto',1,'prf_alto123456789abcdefghijk',
  'per_0123456789abcdefghijkl','rol:rrhh:alto:v1',repeat('c',64),'2026-09-01',
  jsonb_build_object('estado','activa','vigente_desde','2026-09-01T00:00:00Z',
   'vigente_hasta','2027-01-01T00:00:00Z'));
INSERT INTO vec_autorizacion.asignacion_perfil_actual VALUES
 ('prf_alto123456789abcdefghijk','asignacion:rrhh:alto:v1','2026-09-01','prueba','acto:prueba:asignacion');
