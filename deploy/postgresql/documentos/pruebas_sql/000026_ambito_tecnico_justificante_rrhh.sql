\set ON_ERROR_STOP on
-- Copia propia tras B124/D26. Pasar referencias ficticias privadas con -v:
-- d26_bolsa_ref, d26_participacion_ref, d26_actor_ref, d26_perfil_ref,
-- d26_contexto_ref, d26_unidad_ref, d26_ambito_ref. No usa firmas falsas.
BEGIN;
SET TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
CREATE TEMP TABLE d26_fixture(
 bolsa_ref text,participacion_ref text,actor_ref text,perfil_ref text,
 contexto_ref text,unidad_ref text,ambito_ref text) ON COMMIT DROP;
INSERT INTO d26_fixture VALUES(:'d26_bolsa_ref',:'d26_participacion_ref',
 :'d26_actor_ref',:'d26_perfil_ref',:'d26_contexto_ref',
 :'d26_unidad_ref',:'d26_ambito_ref');
GRANT SELECT ON d26_fixture TO vec_interno_documentos_bolsa_rrhh;
DO $estructura$
DECLARE p pg_proc%ROWTYPE; f record;
BEGIN
 FOR f IN SELECT * FROM (VALUES
  ('vec_documentos.consultar_justificante_renuncia_rrhh_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
   '6dce99272be07fc4c02ae306d9c37233d485900ee0edcae84a2981f7e88f242c'),
  ('vec_documentos.resolver_consulta_justificante_renuncia_bolsa_rrhh_v1(text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
   '0ea95cdc1bf1b69e15c2c75f47f09f1d20d635afdc3527ca7be06d6b6adcd4c5')
 ) v(firma,huella) LOOP
  SELECT * INTO STRICT p FROM pg_proc WHERE oid=to_regprocedure(f.firma);
  IF p.proowner IS DISTINCT FROM 'vec_documentos_propietario'::regrole OR
   p.prosecdef IS NOT TRUE OR
   encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM f.huella
  THEN RAISE EXCEPTION 'D26 prueba: función divergente %',f.firma; END IF;
 END LOOP;
END $estructura$;
SET SESSION AUTHORIZATION vec_interno_documentos_bolsa_rrhh;
DO $denegacion$
DECLARE f record; contexto bytea; c jsonb; d jsonb; h text;
 modo text; fallo text;
BEGIN
 SELECT * INTO STRICT f FROM pg_temp.d26_fixture;
 IF f.ambito_ref='ambito:ajeno' OR
  f.actor_ref !~ '^per_[A-Za-z0-9_-]{22,128}$' OR
  f.perfil_ref !~ '^prf_[A-Za-z0-9_-]{22,128}$' OR
  f.contexto_ref !~ '^rca_[A-Za-z0-9_-]{22,128}$'
 THEN RAISE EXCEPTION 'D26 prueba: fixture RRHH inválida'; END IF;
 contexto:=convert_to(jsonb_build_object('principal_ref',f.actor_ref,
  'perfil_activo_ref',f.perfil_ref)::text,'UTF8');
 FOREACH modo IN ARRAY ARRAY['correcto','ambito_ajeno'] LOOP
  h:=encode(sha256(convert_to(format(
   '{"ambitos":{"ambito_ref":"%s","unidad_ref":"%s"},"atributos":{"bolsa_ref":"%s","participacion_ref":"%s"}}',
   CASE WHEN modo='ambito_ajeno' THEN 'ambito:ajeno' ELSE f.ambito_ref END,
   f.unidad_ref,f.bolsa_ref,f.participacion_ref),'UTF8')),'hex');
  c:=jsonb_build_object('contexto_ref',f.contexto_ref,
   'huella_contexto_sha256',encode(sha256(contexto),'hex'),
   'efecto_ref',f.participacion_ref,
   'operacion','bolsa.justificante_renuncia.descargar_rrhh',
   'audiencia_consumo','vec_documentos.bolsa.justificante_renuncia.preparar_descarga_rrhh.v1',
   'huella_efecto_sha256',h);
  d:=jsonb_build_object('accion','bolsa.justificante_renuncia.descargar_rrhh',
   'recurso_ref',f.participacion_ref,'modulo_id','bolsa',
   'tipo_recurso','justificante_renuncia',
   'finalidad','custodiar_justificante_renuncia_bolsa',
   'principal_id',f.actor_ref,'perfil_activo_ref',f.perfil_ref,
   'contexto_recurso_huella_sha256',h,'campos_permitidos','["contenido"]'::jsonb,
   'obligaciones','[]'::jsonb,
   'vinculo_autenticacion_actor',jsonb_build_object(
    'registro_contexto_ref',f.contexto_ref,
    'contexto_actor_huella_sha256',encode(sha256(contexto),'hex'),
    'superficie','interna_corporativa'));
  fallo:='sin_error';
  BEGIN
   PERFORM vec_documentos.resolver_consulta_justificante_renuncia_bolsa_rrhh_v1(
    f.bolsa_ref,f.participacion_ref,f.actor_ref,f.perfil_ref,f.contexto_ref,
    convert_to(c::text,'UTF8'),convert_to(d::text,'UTF8'),NULL,contexto,
    1,1,NULL,NULL,NULL,NULL);
  EXCEPTION WHEN OTHERS THEN GET STACKED DIAGNOSTICS fallo=MESSAGE_TEXT;
  END;
  IF modo='ambito_ajeno' THEN
   IF fallo IS DISTINCT FROM 'D22: preparación RRHH divergente'
   THEN RAISE EXCEPTION 'D26 prueba: otro ámbito no denegado (%)',fallo; END IF;
  ELSIF fallo='sin_error' OR fallo LIKE 'D22: preparación RRHH %' THEN
   RAISE EXCEPTION 'D26 prueba: ámbito propio no alcanzó consumidor V3 (%)',fallo;
  END IF;
 END LOOP;
END $denegacion$;
ROLLBACK;
\echo D26_PRUEBA_AMBITO_AJENO_OK
