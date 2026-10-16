\set ON_ERROR_STOP on
-- Tras B125/D27 en copia propia. Pasar con -v referencias ficticias privadas:
-- d27_bolsa_ref,d27_participacion_ref,d27_actor_ref,d27_perfil_ref,d27_contexto_ref.
-- No firma ni consume una capacidad válida; toda prueba acaba en ROLLBACK.
BEGIN;
SET TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
CREATE TEMP TABLE d27_fixture(bolsa_ref text,participacion_ref text,actor_ref text,
 perfil_ref text,contexto_ref text) ON COMMIT DROP;
INSERT INTO d27_fixture VALUES(:'d27_bolsa_ref',:'d27_participacion_ref',
 :'d27_actor_ref',:'d27_perfil_ref',:'d27_contexto_ref');
GRANT SELECT ON d27_fixture TO vec_interno_documentos_bolsa_rrhh;
DO $orden$
DECLARE f record; cuerpo text;
BEGIN
 FOR f IN SELECT * FROM (VALUES
  ('vec_documentos.consultar_justificante_renuncia_rrhh_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
   'eb4f7f73213052daa9664b479b98e325c7c6386a9c0eebe81b2f422ba5b633f7'),
  ('vec_documentos.resolver_consulta_justificante_renuncia_bolsa_rrhh_v1(text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
   '845b022f144e1372e0bd5fea9d9fe57af116b276d2db68ebcc1d516e41550812')
 ) v(firma,huella) LOOP
  SELECT p.prosrc INTO STRICT cuerpo FROM pg_proc p WHERE p.oid=to_regprocedure(f.firma);
  IF encode(sha256(convert_to(cuerpo,'UTF8')),'hex') IS DISTINCT FROM f.huella OR
   position('consumir_subida_local_bolsa_v1(' IN cuerpo)=0 OR
   position('resolver_ambito_justificante_rrhh_documental_v1(' IN cuerpo)=0 OR
   position('consumir_subida_local_bolsa_v1(' IN cuerpo)>
    position('resolver_ambito_justificante_rrhh_documental_v1(' IN cuerpo)
  THEN RAISE EXCEPTION 'D27 prueba: lectura relacional precede permiso %',f.firma; END IF;
 END LOOP;
END $orden$;
SET SESSION AUTHORIZATION vec_interno_documentos_bolsa_rrhh;
DO $negativa$
DECLARE f record; contexto bytea; c jsonb; d jsonb; pre bytea; auth jsonb;
 bolsa text; participacion text; fase text; codigo text; mensaje text;
 codigo_real text; mensaje_real text; documento constant text:='ref:'||repeat('a',64);
 recurso constant text:='justificante-renuncia:'||repeat('b',64);
BEGIN
 SELECT * INTO STRICT f FROM pg_temp.d27_fixture;
 contexto:=convert_to(jsonb_build_object('principal_ref',f.actor_ref,
  'perfil_activo_ref',f.perfil_ref)::text,'UTF8');
 FOREACH fase IN ARRAY ARRAY['preparar','abrir'] LOOP
  codigo_real:=NULL; mensaje_real:=NULL;
  FOR bolsa,participacion IN SELECT * FROM (VALUES
   (f.bolsa_ref,f.participacion_ref),('bolsa:inexistente','participacion:inexistente')
  ) q(bolsa,participacion) LOOP
   c:=jsonb_build_object('contexto_ref',f.contexto_ref,
    'huella_contexto_sha256',encode(sha256(contexto),'hex'),
    'efecto_ref',CASE WHEN fase='preparar' THEN participacion ELSE documento END,
    'operacion','bolsa.justificante_renuncia.descargar_rrhh',
    'audiencia_consumo',CASE WHEN fase='preparar' THEN
     'vec_documentos.bolsa.justificante_renuncia.preparar_descarga_rrhh.v1' ELSE
     'vec_documentos.bolsa.justificante_renuncia.descarga_rrhh.v1' END,
    'huella_efecto_sha256',repeat('c',64));
   d:=jsonb_build_object('accion','bolsa.justificante_renuncia.descargar_rrhh',
    'recurso_ref',CASE WHEN fase='preparar' THEN participacion ELSE documento END,
    'modulo_id','bolsa','tipo_recurso','justificante_renuncia',
    'finalidad','custodiar_justificante_renuncia_bolsa',
    'principal_id',f.actor_ref,'perfil_activo_ref',f.perfil_ref,
    'contexto_recurso_huella_sha256',repeat('c',64),
    'campos_permitidos','["contenido"]'::jsonb,'obligaciones','[]'::jsonb,
    'vinculo_autenticacion_actor',jsonb_build_object(
     'registro_contexto_ref',f.contexto_ref,
     'contexto_actor_huella_sha256',encode(sha256(contexto),'hex'),
     'superficie','interna_corporativa'));
   codigo:='sin_error'; mensaje:='sin_error';
   BEGIN
    IF fase='preparar' THEN
     PERFORM vec_documentos.resolver_consulta_justificante_renuncia_bolsa_rrhh_v1(
      bolsa,participacion,f.actor_ref,f.perfil_ref,f.contexto_ref,
      convert_to(c::text,'UTF8'),convert_to(d::text,'UTF8'),NULL,contexto,
      1,1,NULL,NULL,NULL,NULL);
    ELSE
     pre:=convert_to(jsonb_build_object('fase','abrir_rrhh',
      'accion','bolsa.justificante_renuncia.descargar_rrhh','carga_id','',
      'documento_id',documento,'recurso_ref',recurso,'persona_ref',
      'per_0123456789abcdefghijkl','actor_ref',f.actor_ref,
      'perfil_activo_ref',f.perfil_ref,'bolsa_ref',bolsa,
      'participacion_ref',participacion,'registro_contexto_ref',f.contexto_ref,
      'huella_contexto_sha256',encode(sha256(contexto),'hex'),
      'correlacion_ref','corr:prueba')::text,'UTF8');
     auth:=jsonb_build_object('accion','bolsa.justificante_renuncia.descargar_rrhh',
      'recurso_ref',documento,'ambito_ref',recurso,'principal_id',f.actor_ref,
      'perfil_activo_ref',f.perfil_ref,
      'finalidad','custodiar_justificante_renuncia_bolsa','correlacion_ref','corr:prueba');
     PERFORM vec_documentos.consultar_justificante_renuncia_rrhh_v1(
      pre,auth,convert_to(c::text,'UTF8'),convert_to(d::text,'UTF8'),NULL,
      contexto,1,1,NULL,NULL,NULL,NULL);
    END IF;
   EXCEPTION WHEN OTHERS THEN GET STACKED DIAGNOSTICS
    codigo=RETURNED_SQLSTATE,mensaje=MESSAGE_TEXT;
   END;
   IF codigo='sin_error' OR mensaje LIKE 'B124:%' OR mensaje LIKE 'D22:%'
   THEN RAISE EXCEPTION 'D27 prueba: capacidad falsa alcanzó relación % %',fase,codigo; END IF;
   IF codigo_real IS NULL THEN codigo_real:=codigo; mensaje_real:=mensaje;
   ELSIF codigo IS DISTINCT FROM codigo_real OR mensaje IS DISTINCT FROM mensaje_real
   THEN RAISE EXCEPTION 'D27 prueba: existencia distinguible %',fase; END IF;
  END LOOP;
 END LOOP;
END $negativa$;
ROLLBACK;
\echo D27_PRUEBA_CAPACIDAD_INVALIDA_OK
