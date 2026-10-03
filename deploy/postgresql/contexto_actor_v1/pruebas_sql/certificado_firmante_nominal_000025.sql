\set ON_ERROR_STOP on
-- Ejecutar sólo en PostgreSQL 18 desechable, tras CA25. Todo se revierte.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';
SET LOCAL search_path=pg_catalog;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
INSERT INTO vec_contexto_actor_v1.procedencias VALUES
 ('prc_ca25_sintetica_autoridad_aaaaaaaa',1,repeat('a',64),'autoridad_maestra_acreditada');
INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_versiones VALUES
 ('cta_ca25_sintetica_aaaaaaaaaaaaaaa',1,'prc_ca25_sintetica_autoridad_aaaaaaaa',1,repeat('a',64),
  'autoridad_maestra_acreditada','activo',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '1 day');
INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_actual VALUES('cta_ca25_sintetica_aaaaaaaaaaaaaaa',1);
INSERT INTO vec_contexto_actor_v1.persona_versiones VALUES
 ('per_ca25_sintetica_bbbbbbbbbbbbbbb',1,'prc_ca25_sintetica_autoridad_aaaaaaaa',1,repeat('a',64),
  'autoridad_maestra_acreditada','activo',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '1 day');
INSERT INTO vec_contexto_actor_v1.persona_actual VALUES('per_ca25_sintetica_bbbbbbbbbbbbbbb',1);
INSERT INTO vec_contexto_actor_v1.perfil_versiones VALUES
 ('prf_ca25_sintetico_ccccccccccccccc',1,'per_ca25_sintetica_bbbbbbbbbbbbbbb','prc_ca25_sintetica_autoridad_aaaaaaaa',1,repeat('a',64),
  'autoridad_maestra_acreditada','activo',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '1 day');
INSERT INTO vec_contexto_actor_v1.perfil_actual VALUES('prf_ca25_sintetico_ccccccccccccccc',1);
INSERT INTO vec_contexto_actor_v1.vinculo_contexto_versiones VALUES
 ('vca_ca25_sintetico_ddddddddddddddd',1,'cta_ca25_sintetica_aaaaaaaaaaaaaaa','prf_ca25_sintetico_ccccccccccccccc',
  'per_ca25_sintetica_bbbbbbbbbbbbbbb','prc_ca25_sintetica_autoridad_aaaaaaaa',1,repeat('a',64),
  'autoridad_maestra_acreditada','activo',clock_timestamp()-interval '1 hour',clock_timestamp()+interval '1 day');
INSERT INTO vec_contexto_actor_v1.vinculo_contexto_actual VALUES('vca_ca25_sintetico_ddddddddddddddd',1);
RESET ROLE;
DO $prueba$
DECLARE d jsonb;b bytea;r jsonb;clave text:='aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa';der text:=repeat('a',64);
BEGIN
 IF has_function_privilege('vec_contexto_actor_v1_runtime',
  'vec_contexto_actor_v1.leer_revalidar_certificado_firmante_ct_v2(text)','EXECUTE')
  OR has_function_privilege('vec_contexto_actor_v1_runtime',
  'vec_contexto_actor_v1.publicar_certificado_firmante_ct_v2(bytea,text,text)','EXECUTE')
  OR NOT has_function_privilege('vec_autorizacion_propietario',
  'vec_contexto_actor_v1.leer_revalidar_certificado_firmante_ct_v2(text)','EXECUTE')
 THEN RAISE EXCEPTION 'CA25: ACL de fachada divergente'; END IF;
 d:=jsonb_build_object('esquema','vec.contexto-actor.certificado-firmante.publicacion.v2',
  'clave',clave,'vinculo_ref','vcc_ca25_sintetico_eeeeeeeeeeeeeee','version',1,
  'certificado_der_sha256',der,'cuenta_ref','cta_ca25_sintetica_aaaaaaaaaaaaaaa',
  'persona_ref','per_ca25_sintetica_bbbbbbbbbbbbbbb','vinculo_cuenta_persona_ref','vca_ca25_sintetico_ddddddddddddddd',
  'estado','vigente','vigente_desde',to_char((clock_timestamp()-interval '1 hour') AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  'vigente_hasta',to_char((clock_timestamp()+interval '1 day') AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  'evidencia_ref','evi_ca25_sintetica_fffffffffffffff','evidencia_sha256',repeat('f',64),
  'preimagen_ref','','preimagen_version',0,'preimagen_sha256','none');
 b:=convert_to(d::text,'UTF8');
 r:=vec_contexto_actor_v1.publicar_certificado_firmante_ct_v2(b,'decision-ca25-sintetica','auditoria-ca25-sintetica');
 IF r->>'recibo_ref' IS DISTINCT FROM 'recibo_certificado_nominal:'||clave
  OR (SELECT count(*) FROM vec_contexto_actor_v1.certificado_firmante_nominal_versiones WHERE certificado_der_sha256=der)<>1
 THEN RAISE EXCEPTION 'CA25: alta no persistida'; END IF;
 IF vec_contexto_actor_v1.publicar_certificado_firmante_ct_v2(b,'decision-ca25-replay','auditoria-ca25-replay') IS DISTINCT FROM r
 THEN RAISE EXCEPTION 'CA25: replay cambió recibo'; END IF;
 IF (SELECT count(*) FROM vec_contexto_actor_v1.certificado_firmante_nominal_versiones WHERE certificado_der_sha256=der)<>1
 THEN RAISE EXCEPTION 'CA25: replay duplicado'; END IF;
END $prueba$;
SET LOCAL ROLE vec_autorizacion_propietario;
DO $lectura$
DECLARE j jsonb;
BEGIN
 j:=vec_contexto_actor_v1.leer_revalidar_certificado_firmante_ct_v2(repeat('a',64));
 IF j->>'esquema' IS DISTINCT FROM 'vec.contexto-actor.certificado-firmante-ct.v2'
  OR j#>>'{vinculo_certificado,certificado_der_sha256}' IS DISTINCT FROM repeat('a',64)
  OR j#>>'{cuenta,referencia}' IS DISTINCT FROM 'cta_ca25_sintetica_aaaaaaaaaaaaaaa'
  OR j#>>'{vinculo_cuenta_persona,persona_ref}' IS DISTINCT FROM 'per_ca25_sintetica_bbbbbbbbbbbbbbb'
  OR j#>>'{vinculo_certificado,auditoria_ref}' IS DISTINCT FROM 'auditoria-ca25-sintetica'
 THEN RAISE EXCEPTION 'CA25: lectura nominal incorrecta'; END IF;
END $lectura$;
RESET ROLE;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
INSERT INTO vec_contexto_actor_v1.persona_versiones
 SELECT persona_ref,2,procedencia_ref,procedencia_version,procedencia_huella_sha256,procedencia_autoridad,
        estado,vigente_desde,vigente_hasta
 FROM vec_contexto_actor_v1.persona_versiones
 WHERE persona_ref='per_ca25_sintetica_bbbbbbbbbbbbbbb' AND version=1;
UPDATE vec_contexto_actor_v1.persona_actual SET version=2 WHERE persona_ref='per_ca25_sintetica_bbbbbbbbbbbbbbb';
RESET ROLE;
SET LOCAL ROLE vec_autorizacion_propietario;
DO $revocacion_fuente$
BEGIN
 IF vec_contexto_actor_v1.leer_revalidar_certificado_firmante_ct_v2(repeat('a',64)) IS NOT NULL
 THEN RAISE EXCEPTION 'CA25: fuente versionada cambiada todavía autorizada'; END IF;
END $revocacion_fuente$;
ROLLBACK;
