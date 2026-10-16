\set ON_ERROR_STOP on
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
DO $t$
DECLARE p pg_proc%ROWTYPE; f text;
BEGIN
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=
  'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_externa_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 f:=pg_get_functiondef(p.oid);
 IF p.proowner IS DISTINCT FROM 'vec_autorizacion_atestada_v3_propietario'::regrole
 OR p.prosecdef IS NOT TRUE OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']::text[]
 OR p.proacl::text IS DISTINCT FROM '{vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario}'
 OR encode(sha256(convert_to(f,'UTF8')),'hex') IS DISTINCT FROM
  'e81b04a640290700aa81f47a9d28a06bf69093a13eda098d7b5c6dfd6642a382'
 OR position('puntero_clave_emision_externa p' IN f)=0
 OR position('consumo_decision_v3_externa' IN f)=0
 OR position('auditoria_consumo_v3_externa' IN f)=0
 OR position('registrar_y_revalidar_documentos_bolsa_externa_v3(' IN f)=0
 OR position('revalidar_documentos_bolsa_externa_v3_viva(' IN f)=0
 THEN RAISE EXCEPTION 'AD284: núcleo externo divergente'; END IF;
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=
  'vec_autorizacion_atestada_v3.consumir_subida_local_bolsa_v1(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 f:=pg_get_functiondef(p.oid);
 IF p.proowner IS DISTINCT FROM 'vec_autorizacion_atestada_v3_propietario'::regrole
 OR p.prosecdef IS NOT TRUE OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']::text[]
 OR p.proacl::text IS DISTINCT FROM '{vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario,vec_documentos_propietario=X/vec_autorizacion_atestada_v3_propietario}'
 OR encode(sha256(convert_to(f,'UTF8')),'hex') IS DISTINCT FROM
  '5fdd76197357383684a1daf2cbaa60745eabbc9570eb14148c7223e51f312336'
 OR position('IF perfil=''documentos_bolsa_externa'' THEN' IN f)=0
 OR position('consumir_decision_mutacion_v3_externa_interna(' IN f)=0
 OR position('ELSE' IN f)=0
 OR position('consumir_decision_mutacion_v3_interna(' IN f)=0
 THEN RAISE EXCEPTION 'AD284: despacho documental divergente'; END IF;
END $t$;
-- Un LOGIN distinto del documental no puede usar el núcleo externo, ni con
-- material incompleto. El usuario de esta prueba conserva session_user original.
DO $denegado$
BEGIN
 BEGIN
  PERFORM * FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_externa_interna(
   'documentos_bolsa_externa',NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'AD284: núcleo externo aceptó otro LOGIN';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $denegado$;
ROLLBACK;
