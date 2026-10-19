\set ON_ERROR_STOP on
-- F1-71: fachada nominal y denegación antes de cualquier consumo.
-- El positivo firmado, replay y revocación se ensayan con B128 desde la web.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';SET LOCAL lock_timeout='5s';SET LOCAL statement_timeout='30s';
CREATE FUNCTION pg_temp.exigir(caso text,ok boolean) RETURNS void LANGUAGE plpgsql AS $f$
BEGIN IF ok IS NOT TRUE THEN RAISE EXCEPTION 'AD288 prueba: %',caso;END IF;END $f$;
CREATE FUNCTION pg_temp.estado(sentencia text) RETURNS text LANGUAGE plpgsql AS $f$
BEGIN EXECUTE sentencia;RETURN 'admitida';EXCEPTION WHEN OTHERS THEN RETURN SQLSTATE;END $f$;
SELECT pg_temp.exigir('fachada privada', (SELECT p.prosecdef
 AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
 AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
 AND p.proacl::text='{vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario,vec_bolsa_llamamientos_propietario=X/vec_autorizacion_atestada_v3_propietario}'
 FROM pg_proc p WHERE oid='vec_autorizacion_atestada_v3.registrar_y_consumir_resolucion_respuesta_revision_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure));
SELECT pg_temp.exigir('Bolsa no alcanza directamente el núcleo',
 NOT has_function_privilege('vec_bolsa_llamamientos_propietario',
 'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE'));
SELECT pg_temp.exigir('origen nominal único',(SELECT count(*)=1
 FROM vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1
 WHERE audiencia_consumo='vec_bolsa_llamamientos.respuesta_revision.resolver.v1'
 AND operacion='bolsa.respuesta_revision.resolver' AND login_nombre='vec_bolsa_llamamientos_desarrollo'
 AND proceso='vec-server' AND canal_permitido='interna_corporativa'));
SELECT count(*) AS consumos_antes FROM vec_autorizacion_atestada_v3.consumo_decision_v3 \gset
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SELECT pg_temp.exigir('JSON inválido',pg_temp.estado($q$
 SELECT vec_autorizacion_atestada_v3.registrar_y_consumir_resolucion_respuesta_revision_bolsa_v3_atestada(
 decode('ff','hex'),convert_to('{}','UTF8'),NULL,NULL,1,1,NULL,NULL,NULL,NULL)$q$)='22023');
SELECT pg_temp.exigir('sin contrato nominal',pg_temp.estado($q$
 SELECT vec_autorizacion_atestada_v3.registrar_y_consumir_resolucion_respuesta_revision_bolsa_v3_atestada(
 convert_to('{}','UTF8'),convert_to('{}','UTF8'),NULL,NULL,1,1,NULL,NULL,NULL,NULL)$q$)='42501');
SET LOCAL ROLE vec_bolsa_llamamientos_ejecutor;
SELECT pg_temp.exigir('ejecutor solo entra por el módulo',pg_temp.estado($q$
 SELECT vec_autorizacion_atestada_v3.registrar_y_consumir_resolucion_respuesta_revision_bolsa_v3_atestada(
 convert_to('{}','UTF8'),convert_to('{}','UTF8'),NULL,NULL,1,1,NULL,NULL,NULL,NULL)$q$)='42501');
RESET ROLE;
SELECT pg_temp.exigir('sin consumo',(SELECT count(*)=:consumos_antes FROM vec_autorizacion_atestada_v3.consumo_decision_v3));
ROLLBACK;
