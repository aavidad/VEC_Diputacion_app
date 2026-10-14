\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $test$
DECLARE nombre text; t oid; f oid; def text; firma text:=
 'bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea';
BEGIN
 FOREACH nombre IN ARRAY ARRAY['subida_local','subida_local_historia','subida_local_lectura',
  'subida_local_outbox','frontera_subida_local'] LOOP
  t:=to_regclass('vec_documentos.'||nombre);
  IF t IS NULL OR NOT EXISTS(SELECT 1 FROM pg_class WHERE oid=t AND relrowsecurity AND relforcerowsecurity) OR
   has_table_privilege('vec_documentos_subida_datos',t,'SELECT,INSERT,UPDATE,DELETE') OR
   has_table_privilege('vec_documentos_frontera_externa',t,'SELECT,INSERT,UPDATE,DELETE')
  THEN RAISE EXCEPTION 'D20: tabla sin frontera RLS: %',nombre; END IF;
 END LOOP;
 FOREACH nombre IN ARRAY ARRAY['reservar_subida_local_v1','confirmar_recepcion_subida_local_v1',
  'iniciar_analisis_subida_local_v1','registrar_analisis_subida_local_v1',
  'cerrar_analisis_subida_local_v1','consultar_subida_local_propia_v1'] LOOP
  f:=to_regprocedure('vec_documentos.'||nombre||'('||firma||')');
  IF f IS NULL OR NOT has_function_privilege('vec_documentos_subida_datos',f,'EXECUTE') OR
   has_function_privilege('vec_documentos_frontera_externa',f,'EXECUTE')
  THEN RAISE EXCEPTION 'D20: ACL de fachada: %',nombre; END IF;
  def:=pg_get_functiondef(f);
  IF strpos(def,'SECURITY DEFINER')=0 OR strpos(def,'SET search_path TO ''pg_catalog'', ''pg_temp''')=0 OR
   strpos(def,'subida_local_login_valido_v1(false)')=0
  THEN RAISE EXCEPTION 'D20: fachada sin guardia: %',nombre; END IF;
 END LOOP;
 def:=pg_get_functiondef(to_regprocedure('vec_documentos.operar_subida_local_interna_v1('||firma||')'));
 IF strpos(def,'registrar_cambio_subida_local_v1')=0 OR
  strpos(def,'fase=''registrar_analisis''')=0 OR
  strpos(def,'v_analisis IS DISTINCT FROM r.analisis')=0 OR
  strpos(def,'j->>''objeto_version'' IS DISTINCT FROM r.admitido')=0 OR
  strpos(def,'r.propietario_version>p_persona_version')=0 OR
  strpos(def,'documento_version=1')=0
 THEN RAISE EXCEPTION 'D20: transición, recuperación u objeto exacto ausentes'; END IF;
 IF strpos(pg_get_functiondef(to_regprocedure('vec_documentos.consumir_preimagen_subida_local_v1('||firma||')')),
  '{"ambitos":{"persona_ref":"%s"},"atributos":{"preimagen_sha256":"%s"}}')=0 OR
  encode(sha256(convert_to(format('{"ambitos":{"persona_ref":"%s"},"atributos":{"preimagen_sha256":"%s"}}',
   'per_test_123',repeat('a',64)),'UTF8')),'hex') IS DISTINCT FROM
   '978dfbe59d81be12bc57dbcfbd6b1e24c645205cba09cb868c981c0f8244c43f'
 THEN RAISE EXCEPTION 'D20: huella canónica PDP divergente'; END IF;
 f:='vec_documentos.registrar_denegacion_frontera_subida_local_v1(text,text,text,text,timestamptz)'::regprocedure;
 IF NOT has_function_privilege('vec_documentos_frontera_externa',f,'EXECUTE') OR
  has_function_privilege('vec_documentos_subida_datos',f,'EXECUTE') OR
  strpos(pg_get_functiondef(f),'/api/vec/meritos/cartera/comprobantes/estado')=0
 THEN RAISE EXCEPTION 'D20: auditoría de frontera no segregada'; END IF;
 IF encode(sha256(convert_to(pg_get_functiondef(
  'vec_documentos.consumir_carga_directa_v3_v1(bytea,text,text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure),'UTF8')),'hex')
  IS DISTINCT FROM '91b2867207f01b3ae70d4d339c4b6155b2e35699ce5db2bec6aa1939f61733af' OR
  NOT EXISTS(SELECT 1 FROM pg_trigger WHERE tgrelid='vec_documentos.referencia_externa'::regclass
   AND tgname='guardia_meritos_mayan' AND tgenabled='O')
 THEN RAISE EXCEPTION 'D20: historia Mayan alterada'; END IF;
END $test$;
ROLLBACK;
