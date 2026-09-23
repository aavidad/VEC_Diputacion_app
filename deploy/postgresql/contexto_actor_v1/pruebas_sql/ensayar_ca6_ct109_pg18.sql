-- Datos exclusivamente sinteticos en contenedor sin red.
BEGIN;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
INSERT INTO vec_contexto_actor_v1.procedencias VALUES
 ('prc_autoridad_corporativa_rrhh_0001',1,repeat('a',64),'autoridad_maestra_acreditada');
INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_versiones VALUES
 ('cta_corporativa_rrhh_000000000001',1,'prc_autoridad_corporativa_rrhh_0001',1,
  repeat('a',64),'autoridad_maestra_acreditada','activo',
  clock_timestamp()-interval '1 hour',clock_timestamp()+interval '1 hour');
INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_actual VALUES
 ('cta_corporativa_rrhh_000000000001',1);
INSERT INTO vec_contexto_actor_v1.persona_versiones VALUES
 ('per_corporativa_rrhh_000000000001',1,'prc_autoridad_corporativa_rrhh_0001',1,
  repeat('a',64),'autoridad_maestra_acreditada','activo',
  clock_timestamp()-interval '1 hour',clock_timestamp()+interval '1 hour');
INSERT INTO vec_contexto_actor_v1.persona_actual VALUES
 ('per_corporativa_rrhh_000000000001',1);
INSERT INTO vec_contexto_actor_v1.perfil_versiones VALUES
 ('prf_corporativo_rrhh_000000000001',1,'per_corporativa_rrhh_000000000001',
  'prc_autoridad_corporativa_rrhh_0001',1,repeat('a',64),
  'autoridad_maestra_acreditada','activo',
  clock_timestamp()-interval '1 hour',clock_timestamp()+interval '1 hour');
INSERT INTO vec_contexto_actor_v1.perfil_actual VALUES
 ('prf_corporativo_rrhh_000000000001',1);
INSERT INTO vec_contexto_actor_v1.vinculo_contexto_versiones VALUES
 ('vca_corporativo_rrhh_000000000001',1,
  'cta_corporativa_rrhh_000000000001','prf_corporativo_rrhh_000000000001',
  'per_corporativa_rrhh_000000000001',
  'prc_autoridad_corporativa_rrhh_0001',1,repeat('a',64),
  'autoridad_maestra_acreditada','activo',
  clock_timestamp()-interval '1 hour',clock_timestamp()+interval '1 hour');
INSERT INTO vec_contexto_actor_v1.vinculo_contexto_actual VALUES
 ('vca_corporativo_rrhh_000000000001',1);
INSERT INTO vec_contexto_actor_v1.organizacion_versiones VALUES
 ('org_diputaciondemo0001',1,'prc_autoridad_corporativa_rrhh_0001',1,
  repeat('a',64),'autoridad_maestra_acreditada','activo',
  clock_timestamp()-interval '1 hour',clock_timestamp()+interval '1 hour');
INSERT INTO vec_contexto_actor_v1.organizacion_actual VALUES
 ('org_diputaciondemo0001',1);
INSERT INTO vec_contexto_actor_v1.vinculo_corporativo_versiones VALUES
 ('vcr_corporativo_rrhh_000000000001',1,
  'cta_corporativa_rrhh_000000000001',1,
  'per_corporativa_rrhh_000000000001',1,
  'prf_corporativo_rrhh_000000000001',1,
  'vca_corporativo_rrhh_000000000001',1,
  'org_diputaciondemo0001',1,
  'prc_autoridad_corporativa_rrhh_0001',1,repeat('a',64),
  'autoridad_maestra_acreditada','interna_corporativa','consulta_rrhh',
  'prc_autoridad_corporativa_rrhh_0001',1,repeat('a',64),
  'autoridad_maestra_acreditada','activo',
  clock_timestamp()-interval '1 hour',clock_timestamp()+interval '1 hour');
INSERT INTO vec_contexto_actor_v1.vinculo_corporativo_actual VALUES
 ('cta_corporativa_rrhh_000000000001','interna_corporativa','consulta_rrhh',
  'vcr_corporativo_rrhh_000000000001',1);
COMMIT;

CREATE TABLE public.ca6_comprobante_prueba AS
 SELECT vec_contexto_actor_v1.resolver_comprobante_ambito_rrhh_v1(
  'cta_corporativa_rrhh_000000000001',1,
  'per_corporativa_rrhh_000000000001',1,
  'prf_corporativo_rrhh_000000000001',1,
  'vca_corporativo_rrhh_000000000001',1) AS comprobante;

DO $positivo$
DECLARE p jsonb;
BEGIN
 SELECT comprobante INTO STRICT p FROM public.ca6_comprobante_prueba;
 IF p IS NULL OR (SELECT count(*) FROM jsonb_object_keys(p))<>24
   OR p->>'organizacion_ref'<>'org_diputaciondemo0001'
   OR p->>'vinculo_corporativo_version'<>'1'
   OR p->>'contexto_ref'<>'vca_corporativo_rrhh_000000000001'
 THEN RAISE EXCEPTION 'CA6: comprobante positivo ausente'; END IF;
END $positivo$;

BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
DO $acreditacion$
DECLARE p jsonb; instante timestamptz;
BEGIN
 SELECT comprobante INTO STRICT p FROM public.ca6_comprobante_prueba;
 instante:=vec_contexto_actor_v1.acreditar_ambito_rrhh_v1(
  p,p->>'cuenta_ref',p->>'persona_ref',p->>'perfil_ref',
  p->>'contexto_ref',p->>'organizacion_ref');
 IF instante IS NULL OR instante<=(p->>'vigente_desde')::timestamptz
   OR instante>=(p->>'vigente_hasta')::timestamptz
 THEN RAISE EXCEPTION 'CA6: acreditacion positiva ausente'; END IF;
 IF vec_contexto_actor_v1.acreditar_ambito_rrhh_v1(
   p,p->>'cuenta_ref',p->>'persona_ref',p->>'perfil_ref',
   p->>'contexto_ref','org_ajena00000000001') IS NOT NULL
 THEN RAISE EXCEPTION 'CA6: organizacion ajena admitida'; END IF;
END $acreditacion$;
COMMIT;

DO $acl$
DECLARE grupo oid:='vec_contratacion_temporal_consultor_rrhh_ambito'::regrole;
 legado oid:='vec_contratacion_temporal_consultor_rrhh'::regrole;
BEGIN
 IF has_function_privilege(grupo,
  'vec_contratacion_temporal.consultar_estadisticas_rrhh_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,text,date,date)'::regprocedure,'EXECUTE')
 OR has_function_privilege(grupo,
  'vec_contratacion_temporal.consultar_detalle_rrhh_atestado_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,'EXECUTE')
 OR NOT has_function_privilege(legado,
  'vec_contratacion_temporal.consultar_detalle_rrhh_atestado_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,'EXECUTE')
 THEN RAISE EXCEPTION 'CT109: ACL de productos/historia incompatible'; END IF;
END $acl$;
