\set ON_ERROR_STOP on
-- AD303 / F1-105: la revalidación de CT consulta una secuencia de la cola.
-- El recorrido secuencial toma lecturas de predicado sobre entradas ajenas y
-- provoca 40001 entre consultas simultáneas. Se usa la PK existente, sin
-- duplicar el índice ni modificar el cuerpo, la autoridad o la auditoría.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000303',0));
DO $pre$
DECLARE f regprocedure; t regclass; i regclass; p record;
BEGIN
 f:=to_regprocedure('vec_autorizacion_atestada_v3.revalidar_evidencia_consumo_consulta_rrhh_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 t:=to_regclass('vec_autorizacion_atestada_v3.pendiente_sellado_auditoria_v5');
 i:=to_regclass('vec_autorizacion_atestada_v3.pendiente_sellado_auditoria_v5_pkey');
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR current_user<>'vec_autorizacion_atestada_v3_propietario'
 OR f IS NULL OR t IS NULL OR i IS NULL THEN
  RAISE EXCEPTION 'AD303: preimagen ausente' USING ERRCODE='55000'; END IF;
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=f;
 IF p.proowner<>'vec_autorizacion_atestada_v3_propietario'::regrole
 OR NOT p.prosecdef OR p.provolatile<>'v'
 OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=1s']
 OR p.proacl::text IS DISTINCT FROM '{vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario}'
 OR encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex') IS DISTINCT FROM
  'ad81322f33bbd1640230f33567700376154eb595072a72d74da2fc8239fe7580'
 OR NOT EXISTS(SELECT 1 FROM pg_index x JOIN pg_class c ON c.oid=x.indexrelid
   JOIN pg_am m ON m.oid=c.relam JOIN pg_attribute a ON a.attrelid=t AND a.attname='secuencia'
   WHERE x.indexrelid=i AND x.indrelid=t AND x.indisprimary AND x.indisunique
    AND x.indisvalid AND x.indisready AND x.indislive AND x.indimmediate
    AND x.indnkeyatts=1 AND x.indnatts=1 AND x.indkey[0]=a.attnum
    AND x.indpred IS NULL AND x.indexprs IS NULL AND m.amname='btree'
    AND NOT a.attisdropped AND a.attnotnull AND format_type(a.atttypid,a.atttypmod)='numeric(20,0)')
 THEN RAISE EXCEPTION 'AD303: función o índice divergentes' USING ERRCODE='55000'; END IF;
END $pre$;

-- La preferencia de plan termina al salir de esta función, también con error.
ALTER FUNCTION vec_autorizacion_atestada_v3.revalidar_evidencia_consumo_consulta_rrhh_v3_interna(
 text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) SET enable_seqscan=off;

DO $post$
DECLARE f regprocedure;
BEGIN
 f:='vec_autorizacion_atestada_v3.revalidar_evidencia_consumo_consulta_rrhh_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f
  AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef AND p.provolatile='v'
  AND p.proacl::text='{vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario}'
  AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=1s','enable_seqscan=off'])
 OR encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex') IS DISTINCT FROM
  '266950eb649d62b73efaa124bb72a717ddbb810e2aae761b664dfc1358f074a1'
 THEN RAISE EXCEPTION 'AD303: postimagen divergente' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
