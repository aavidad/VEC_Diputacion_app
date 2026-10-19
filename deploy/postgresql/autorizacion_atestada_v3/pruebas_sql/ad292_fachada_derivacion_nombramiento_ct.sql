\set ON_ERROR_STOP on
-- Fachada nominal: denegación anterior al consumo, sin decisiones V3 falsas.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';
SET LOCAL search_path=pg_catalog,pg_temp;
CREATE FUNCTION pg_temp.exigir(caso text,ok boolean) RETURNS void LANGUAGE plpgsql AS $f$
BEGIN IF ok IS NOT TRUE THEN RAISE EXCEPTION 'AD292 prueba: %',caso; END IF; END $f$;
CREATE FUNCTION pg_temp.estado(sentencia text) RETURNS text LANGUAGE plpgsql AS $f$
BEGIN EXECUTE sentencia; RETURN 'admitida'; EXCEPTION WHEN OTHERS THEN RETURN SQLSTATE; END $f$;
SELECT pg_temp.exigir('fachada privada', EXISTS(SELECT 1 FROM pg_proc p
 WHERE p.oid='vec_autorizacion_atestada_v3.registrar_y_consumir_derivacion_nombramiento_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
  AND p.prosecdef AND p.provolatile='v'
  AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
  AND p.proacl::text='{vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario,vec_contratacion_temporal_propietario=X/vec_autorizacion_atestada_v3_propietario}'));
SELECT pg_temp.exigir('booleano de consumo propagado', (SELECT
 position('x.consumo_nuevo IS NULL' IN pg_get_functiondef(p.oid))>0
 AND position('x.decision_ref IS DISTINCT FROM c->>''decision_ref''' IN pg_get_functiondef(p.oid))>0
 AND position('x.consumida_en,x.consumo_nuevo' IN pg_get_functiondef(p.oid))>0
 FROM pg_proc p WHERE p.oid=
  'vec_autorizacion_atestada_v3.registrar_y_consumir_derivacion_nombramiento_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure));
SELECT pg_temp.exigir('origen CT derivado único', (SELECT count(*)=1
 FROM vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1
 WHERE audiencia_consumo='vec.ct.nombramiento-derivado.v1'
  AND operacion='contratacion_temporal.nombramiento.derivar'
  AND proceso='vec-server' AND canal_permitido='interna_corporativa'));
SELECT count(*) AS consumos_antes FROM vec_autorizacion_atestada_v3.consumo_decision_v3 \gset
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SELECT pg_temp.exigir('material roto',pg_temp.estado($q$
 SELECT vec_autorizacion_atestada_v3.registrar_y_consumir_derivacion_nombramiento_ct_v3_atestada(
 decode('ff','hex'),convert_to('{}','UTF8'),NULL,NULL,1,1,NULL,NULL,NULL,NULL)$q$)='22023');
SELECT pg_temp.exigir('contrato ajeno',pg_temp.estado($q$
 SELECT vec_autorizacion_atestada_v3.registrar_y_consumir_derivacion_nombramiento_ct_v3_atestada(
 convert_to('{}','UTF8'),convert_to('{}','UTF8'),NULL,NULL,1,1,NULL,NULL,NULL,NULL)$q$)='42501');
RESET ROLE;
SELECT pg_temp.exigir('sin consumo', (SELECT count(*)=:consumos_antes FROM vec_autorizacion_atestada_v3.consumo_decision_v3));
ROLLBACK;
