\set ON_ERROR_STOP on
-- Copia propia tras B124: pasar -v b124_bolsa_ref=... -v b124_participacion_ref=...
-- con referencias ficticias del justificante firme. Sin documentos ni PII en Git.
BEGIN;
SET TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
CREATE TEMP TABLE b124_fixture(bolsa_ref text,participacion_ref text) ON COMMIT DROP;
INSERT INTO b124_fixture VALUES(:'b124_bolsa_ref',:'b124_participacion_ref');
GRANT SELECT ON b124_fixture TO vec_bolsa_llamamientos_desarrollo;
DO $acl$
BEGIN
 IF EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(p.proacl) a
  WHERE p.oid='vec_bolsa_llamamientos.resolver_ambito_justificante_rrhh_documental_v1(text,text,text)'::regprocedure
   AND a.grantee=0 AND a.privilege_type='EXECUTE') OR
  has_function_privilege('vec_bolsa_llamamientos_ejecutor',
  'vec_bolsa_llamamientos.resolver_ambito_justificante_rrhh_documental_v1(text,text,text)',
  'EXECUTE') OR
  has_function_privilege('vec_documentos_bolsa_rrhh',
  'vec_bolsa_llamamientos.resolver_ambito_justificante_rrhh_documental_v1(text,text,text)',
  'EXECUTE') OR
  NOT has_function_privilege('vec_documentos_propietario',
  'vec_bolsa_llamamientos.resolver_ambito_justificante_rrhh_documental_v1(text,text,text)',
  'EXECUTE')
 THEN RAISE EXCEPTION 'B124 prueba: fachada documental abierta'; END IF;
END $acl$;
SET SESSION AUTHORIZATION vec_bolsa_llamamientos_desarrollo;
DO $ambito$
DECLARE f record; m jsonb; errado jsonb; fallo text;
BEGIN
 SELECT * INTO STRICT f FROM pg_temp.b124_fixture;
 m:=vec_bolsa_llamamientos.resolver_metadatos_justificante_rrhh_v1(
  f.bolsa_ref,f.participacion_ref);
 IF m->>'bolsa_ref' IS DISTINCT FROM f.bolsa_ref OR
  m->>'participacion_ref' IS DISTINCT FROM f.participacion_ref OR
  m->>'llamamiento_ref' !~ '^llamamiento:[0-9a-f]{64}$' OR
  nullif(m->>'unidad_ref','') IS NULL OR nullif(m->>'ambito_ref','') IS NULL OR
  (SELECT count(*) FROM jsonb_object_keys(m))<>5
 THEN RAISE EXCEPTION 'B124 prueba: metadatos propios inválidos'; END IF;
 errado:=jsonb_build_object('accion','bolsa.rrhh.respuestas_llamamiento.consultar',
  'recurso_ref',m->>'llamamiento_ref','bolsa_ref',f.bolsa_ref,
  'unidad_ref',m->>'unidad_ref','ambito_ref','ambito:ajeno');
 fallo:='sin_error';
 BEGIN
  PERFORM vec_bolsa_llamamientos.resolver_ambito_justificante_rrhh_auditado_v1(
   f.bolsa_ref,f.participacion_ref,NULL,NULL,errado);
 EXCEPTION WHEN OTHERS THEN GET STACKED DIAGNOSTICS fallo=MESSAGE_TEXT;
 END;
 IF fallo IS DISTINCT FROM 'B124: captura RRHH divergente'
 THEN RAISE EXCEPTION 'B124 prueba: ámbito ajeno no denegado (%)',fallo; END IF;
END $ambito$;
ROLLBACK;
\echo B124_PRUEBA_AMBITO_AJENO_OK
