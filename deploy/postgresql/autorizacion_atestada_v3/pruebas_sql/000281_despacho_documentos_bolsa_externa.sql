\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $t$
DECLARE f text;p pg_proc%ROWTYPE;
BEGIN
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=
  'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 f:=pg_get_functiondef(p.oid);
 IF p.proowner IS DISTINCT FROM 'vec_autorizacion_atestada_v3_propietario'::regrole
 OR p.prosecdef IS NOT TRUE
 OR p.proacl::text IS DISTINCT FROM
  '{vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario}'
 OR position('p_perfil_mutacion IS NOT DISTINCT FROM ''documentos_bolsa_externa'' THEN' IN f)=0
 OR position('vec_autorizacion.registrar_y_revalidar_documentos_bolsa_externa_v3(' IN f)=0
 OR position('vec_autorizacion.revalidar_documentos_bolsa_externa_v3_viva(' IN f)=0
 OR encode(sha256(convert_to(f,'UTF8')),'hex') IS DISTINCT FROM
  'cfcacf14cae0132aefb589b341fe920fbeab1e4d6df1ce3f7685ccd0507c5960'
 THEN RAISE EXCEPTION 'AD281: despacho externo o ACL divergente'; END IF;
END $t$;
ROLLBACK;
