-- Negativos BIC7/B79/B80 en clon desechable. No crea altas de negocio.
\set ON_ERROR_STOP on
BEGIN;
DO $acl$
DECLARE web text:='vec_bolsa_llamamientos_desarrollo';
BEGIN
 IF NOT pg_catalog.has_function_privilege(web,
   'vec_bolsa_llamamientos.confirmar_carga_convoca_v1(jsonb,jsonb,jsonb,text,text,text,text,bigint,bytea,timestamptz,text,bigint,bytea,timestamptz,timestamptz,jsonb,timestamptz,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 OR pg_catalog.has_function_privilege(web,
   'vec_bolsa_llamamientos.constituir_bolsa_v1(text,text,text,text,bigint,bytea,timestamptz,text,bigint,bytea,timestamptz,timestamptz,jsonb,timestamptz)','EXECUTE')
 OR pg_catalog.has_function_privilege(web,
   'vec_bolsa_llamamientos.registrar_vinculos_candidato_v1(text,jsonb,timestamptz)','EXECUTE')
 OR pg_catalog.has_function_privilege(web,
   'vec_bolsa_importacion_convoca.guardar_lote_v1(jsonb,jsonb)','EXECUTE')
 OR pg_catalog.has_function_privilege(web,
   'vec_bolsa_importacion_convoca.guardar_original_v1(jsonb,jsonb)','EXECUTE')
 OR pg_catalog.has_table_privilege(web,
   'vec_bolsa_importacion_convoca.original_exportacion_cifrada','SELECT')
 OR EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members
    WHERE roleid='vec_bolsa_llamamientos_constituidor'::pg_catalog.regrole)
 THEN RAISE EXCEPTION 'B79: ACL web/CLI divergente'; END IF;
END $acl$;
ROLLBACK;

SET SESSION AUTHORIZATION vec_bolsa_llamamientos_desarrollo;
BEGIN ISOLATION LEVEL SERIALIZABLE;
DO $negativos$
DECLARE h text:=pg_catalog.repeat('c',64); cat text:='categoria:rpt:x'; actor text:='per_x';
 acta text:='acta:importacion-convoca:'||pg_catalog.encode(pg_catalog.sha256(
  pg_catalog.convert_to(pg_catalog.repeat('c',64)||pg_catalog.chr(31)||'categoria:rpt:x','UTF8')),'hex');
 a jsonb; b bytea; e jsonb:='[{"participacion_ref":"p","fila_numero":1}]';
 v jsonb:='[{"participacion_ref":"p","candidato_ref":"c"}]';
 d bytea; codigo text;
BEGIN
 a:=pg_catalog.jsonb_build_object('acta_ref',acta,'actor_ref','actor:rrhh:'||pg_catalog.substr(
  pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to('vec.bolsa.carga_convoca.actor'||pg_catalog.chr(31)||actor,'UTF8')),'hex'),1,32),
  'categoria_ref',cat,'bolsa_ref','bolsa:x','huella_fichero_sha256',h);
 b:=pg_catalog.convert_to(pg_catalog.jsonb_build_object('bolsa_ref','bolsa:x',
  'categoria_ref',cat,'huella_listado_sha256',h)::text,'UTF8');
 d:=pg_catalog.convert_to(pg_catalog.jsonb_build_object('principal_id',actor,'recurso_ref',acta)::text,'UTF8');
 -- La misma acta y lista no conceden acceso sin material V3 auténtico.
 BEGIN
  PERFORM vec_bolsa_llamamientos.confirmar_carga_convoca_v1(
   a,'[{"numero":1}]'::jsonb,'{}'::jsonb,acta,actor,cat,'bolsa:x',1,b,now(),
   'ins',1,'\x7b7d',now(),now(),e,now(),v,
   '\x00',d,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
  RAISE EXCEPTION 'B79: material V3 sin firma aceptado';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 -- La decisión de otra persona no alcanza siquiera la fachada AD218.
 BEGIN
  PERFORM vec_bolsa_llamamientos.confirmar_carga_convoca_v1(
   a,'[{"numero":1}]'::jsonb,'{}'::jsonb,acta,actor,cat,'bolsa:x',1,b,now(),
   'ins',1,'\x7b7d',now(),now(),e,now(),v,
   '\x00',pg_catalog.convert_to(pg_catalog.jsonb_build_object('principal_id','per_y','recurso_ref',acta)::text,'UTF8'),
   '\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
  RAISE EXCEPTION 'B79: decisión de actor ajeno aceptada';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 -- No se pueden introducir participaciones adicionales fuera de las filas.
 BEGIN
  PERFORM vec_bolsa_llamamientos.confirmar_carga_convoca_v1(
   a,'[{"numero":1}]'::jsonb,'{}'::jsonb,acta,actor,cat,'bolsa:x',1,b,now(),
   'ins',1,'\x7b7d',now(),now(),e,now(),
   '[{"participacion_ref":"otra","candidato_ref":"c"}]'::jsonb,
   '\x00',d,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
  RAISE EXCEPTION 'B79: vínculo ajeno aceptado';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
END $negativos$;
ROLLBACK;
RESET SESSION AUTHORIZATION;
SELECT 'B79 negativos OK' AS resultado;
