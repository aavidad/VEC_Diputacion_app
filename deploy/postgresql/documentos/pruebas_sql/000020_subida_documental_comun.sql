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
   NOT EXISTS(SELECT 1 FROM pg_policy WHERE polrelid=t AND
    polroles=ARRAY['vec_documentos_propietario'::regrole::oid]) OR
   EXISTS(SELECT 1 FROM pg_policy WHERE polrelid=t AND 0=ANY(polroles)) OR
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
 FOREACH nombre IN ARRAY ARRAY['subida_local_historia','subida_local_outbox'] LOOP
  t:=to_regclass('vec_documentos.'||nombre);
  IF NOT EXISTS(SELECT 1 FROM pg_attribute a WHERE a.attrelid=t AND
   a.attname='huella_postestado_sha256' AND a.attnotnull AND NOT a.attisdropped)
  THEN RAISE EXCEPTION 'D20: % sin huella durable de postestado',nombre; END IF;
 END LOOP;
 f:='vec_documentos.verificar_integridad_subida_local_v1(text)'::regprocedure;
 def:=pg_get_functiondef(f);
 IF NOT EXISTS(SELECT 1 FROM pg_proc WHERE oid=f AND provolatile='v') OR
  has_function_privilege('vec_documentos_subida_datos',f,'EXECUTE') OR
  has_function_privilege('vec_documentos_frontera_externa',f,'EXECUTE') OR
  strpos(def,'FULL JOIN vec_documentos.subida_local_outbox')=0 OR
  strpos(def,'e.huella_postestado_sha256 IS DISTINCT FROM postestado')=0 OR
  strpos(def,'siguiente IS DISTINCT FROM r.version+1')=0
 THEN RAISE EXCEPTION 'D20: verificación de historia y postestado incompleta'; END IF;
 def:=pg_get_functiondef(to_regprocedure('vec_documentos.operar_subida_local_interna_v1('||firma||')'));
 IF strpos(def,'registrar_cambio_subida_local_v1')=0 OR
  strpos(def,'r.perfil_activo_ref IS DISTINCT FROM j->>''perfil_activo_ref''')=0 OR
  strpos(def,'verificar_integridad_subida_local_v1(r.carga_id)')=0 OR
  strpos(def,'fase=''registrar_analisis''')=0 OR
  strpos(def,'v_analisis IS DISTINCT FROM r.analisis')=0 OR
  strpos(def,'j->>''objeto_version'' IS DISTINCT FROM r.admitido')=0 OR
  strpos(def,'r.propietario_version>p_persona_version')=0 OR
  strpos(def,'documento_version=1')=0
 THEN RAISE EXCEPTION 'D20: transición, recuperación u objeto exacto ausentes'; END IF;
 def:=pg_get_functiondef('vec_documentos.registrar_cambio_subida_local_v1(text,bigint,text,text,text,text)'::regprocedure);
 IF strpos(def,'to_jsonb(r)::text')=0 OR
  strpos(def,'''postestado_sha256'',postestado')=0 OR
  strpos(def,'r.version IS DISTINCT FROM p_version')=0 OR
  NOT EXISTS(SELECT 1 FROM pg_proc WHERE
   oid='vec_documentos.instantanea_subida_local_v1(text)'::regprocedure AND provolatile='v') OR
  strpos(pg_get_functiondef('vec_documentos.instantanea_subida_local_v1(text)'::regprocedure),
   'verificar_integridad_subida_local_v1(r.carga_id)')=0
 THEN RAISE EXCEPTION 'D20: cambio o proyección sin postestado'; END IF;
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

-- Solo en esta transacción desechable. Estas filas prueban la cadena local;
-- no representan decisiones V3 ni recibos funcionales.
SET LOCAL ROLE vec_documentos_propietario;
DO $integridad$
DECLARE n integer; id text; clave text; r vec_documentos.subida_local%ROWTYPE;
 postestado text; huella text;
BEGIN
 FOR n IN 1..3 LOOP
  id:='ref:'||encode(sha256(convert_to('d20:integridad:ensayo:'||n::text,'UTF8')),'hex');
  clave:='hmac-sha256:local_v1:'||encode(sha256(convert_to('d20:clave:ensayo:'||n::text,'UTF8')),'hex');
  INSERT INTO vec_documentos.subida_local(carga_id,recurso_ref,persona_ref,perfil_activo_ref,
   modulo_id,tipo_recurso,finalidad,clasificacion,propietario_version,clave_idempotencia_hmac,
   huella_solicitud_hmac,huella_pdf_sha256,tamano,mime,correlacion_ref,registro_contexto_ref,
   huella_contexto_sha256,version,estado)
  VALUES(id,'ref:'||repeat('1',64),'per_0123456789abcdefghijkl','prf_0123456789abcdefghijkl',
   'meritos','hecho','custodiar_evidencia_meritos','confidencial',1,clave,
   'hmac-sha256:local_v1:'||repeat('2',64),repeat('3',64),5,'application/pdf',
   'correlacion_0123456789abcdef','rca_0123456789abcdefghijkl',repeat('4',64),1,'reservada');
  SELECT * INTO STRICT r FROM vec_documentos.subida_local WHERE carga_id=id;
  postestado:=encode(sha256(convert_to(to_jsonb(r)::text,'UTF8')),'hex');
  IF n=1 THEN
   BEGIN
    PERFORM vec_documentos.verificar_integridad_subida_local_v1(id);
    RAISE EXCEPTION 'D20: historia ausente aceptada';
   EXCEPTION WHEN SQLSTATE '42501' THEN NULL;
   END;
  END IF;
  IF n=3 THEN postestado:=repeat('f',64); END IF;
  huella:=encode(sha256(convert_to(jsonb_build_object('carga_id',id,'version',1,
   'estado','reservada','decision_ref','prueba_d20_decision_'||n::text,
   'auditoria_ref','prueba_d20_auditoria_'||n::text,
   'preimagen_sha256',repeat('5',64),'postestado_sha256',postestado,
   'anterior_sha256',repeat('0',64))::text,'UTF8')),'hex');
  INSERT INTO vec_documentos.subida_local_historia(carga_id,version,estado,decision_ref,
   auditoria_v3_ref,huella_preimagen_sha256,huella_postestado_sha256,anterior_sha256,huella_sha256)
  VALUES(id,1,'reservada','prueba_d20_decision_'||n::text,
   'prueba_d20_auditoria_'||n::text,repeat('5',64),postestado,repeat('0',64),huella);
  IF n=1 THEN
   BEGIN
    PERFORM vec_documentos.verificar_integridad_subida_local_v1(id);
    RAISE EXCEPTION 'D20: outbox ausente aceptado';
   EXCEPTION WHEN SQLSTATE '42501' THEN NULL;
   END;
  END IF;
  INSERT INTO vec_documentos.subida_local_outbox(carga_id,version,estado,huella_postestado_sha256)
  VALUES(id,1,'reservada',CASE WHEN n=1 THEN repeat('e',64) ELSE postestado END);
  IF n=2 THEN
   PERFORM vec_documentos.verificar_integridad_subida_local_v1(id);
  ELSE
   BEGIN
    PERFORM vec_documentos.verificar_integridad_subida_local_v1(id);
    RAISE EXCEPTION 'D20: outbox o postestado divergente aceptado';
   EXCEPTION WHEN SQLSTATE '42501' THEN NULL;
   END;
  END IF;
 END LOOP;
END $integridad$;
ROLLBACK;
