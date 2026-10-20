\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL statement_timeout='10s';
DO $contrato$
DECLARE f regprocedure;
BEGIN
 f:='vec_autorizacion_atestada_v3.revalidar_evidencia_consumo_consulta_rrhh_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f AND p.prosecdef
  AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
  AND p.proacl::text='{vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario}'
  AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=1s','enable_seqscan=off'])
 OR encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex')<>
  '266950eb649d62b73efaa124bb72a717ddbb810e2aae761b664dfc1358f074a1'
 THEN RAISE EXCEPTION 'AD303: contrato de función divergente'; END IF;
END $contrato$;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
DO $rechazo$
DECLARE anterior text:=current_setting('enable_seqscan'); vacio bytea:='\x7b7d';
BEGIN
 BEGIN
  PERFORM vec_autorizacion_atestada_v3.revalidar_evidencia_consumo_consulta_rrhh_v3_interna(
   'perfil_ajeno',vacio,vacio,vacio,vacio,1,1,vacio,vacio,vacio,vacio);
  RAISE EXCEPTION 'AD303: perfil ajeno aceptado';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 IF current_setting('enable_seqscan') IS DISTINCT FROM anterior THEN
  RAISE EXCEPTION 'AD303: configuración filtrada fuera de la función'; END IF;
END $rechazo$;
-- El plan puntual debe usar la PK incluso con una cola vacía o pequeña.
SET LOCAL enable_seqscan=off;
DO $plan$
DECLARE p jsonb;
BEGIN
 EXECUTE 'EXPLAIN (FORMAT JSON) SELECT 1 FROM vec_autorizacion_atestada_v3.pendiente_sellado_auditoria_v5 WHERE secuencia=0::numeric' INTO p;
 IF p#>>'{0,Plan,Index Name}' IS DISTINCT FROM 'pendiente_sellado_auditoria_v5_pkey'
 OR p#>>'{0,Plan,Node Type}' NOT IN('Index Scan','Index Only Scan') THEN
  RAISE EXCEPTION 'AD303: lectura puntual sin índice: %',p; END IF;
END $plan$;
ROLLBACK;
