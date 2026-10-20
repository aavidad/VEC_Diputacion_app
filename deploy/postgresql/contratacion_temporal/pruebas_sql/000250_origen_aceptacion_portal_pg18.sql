\set ON_ERROR_STOP on
-- Ejecutar tras CT250 en la copia local de la principal.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $ct250$
DECLARE p pg_proc%ROWTYPE; original text; n bigint;
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999 THEN
  RAISE EXCEPTION 'CT250 prueba: PostgreSQL 18 requerido'; END IF;
 SELECT * INTO p FROM pg_proc WHERE oid=to_regprocedure(
  'vec_contratacion_temporal.consultar_detalle_rrhh_con_bolsa_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text)');
 IF NOT FOUND OR p.proowner<>'vec_contratacion_temporal_propietario'::regrole
    OR p.prosecdef IS NOT TRUE OR p.provolatile<>'v'
    OR p.proacl::text IS DISTINCT FROM '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_consultor_rrhh=X/vec_contratacion_temporal_propietario}'
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=1s','statement_timeout=4s','idle_in_transaction_session_timeout=6s']
    OR encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex')
       <>'00d03db4030cadeafafe7e95faebfcc98d68efa94329d0507e2732e63030f623' THEN
  RAISE EXCEPTION 'CT250 prueba: definición, dueño o ACL divergentes'; END IF;
 original:=p.prosrc;
 IF position('consultar_detalle_rrhh_atestado_v1' in original)=0
    OR position('leer_resultados_emisiones_ct_v1' in original)=0
    OR position('origen_ct_confirmado' in original)=0
    OR position('e.situacion=''confirmada''' in original)=0
    OR position('sin_retencion_declarada' in original)=0
    OR position('o6 AS MATERIALIZED' in original)=0
    OR position('jsonb_object_agg(llamamiento_ref,cantidad=1)' in original)=0
    OR position('jsonb_array_length(b.valor->''participaciones'')=1' in original)=0
    OR position('part.valor->>''respuesta''=''acepta''' in original)=0
    OR position('part.valor->>''modo''=''firme''' in original)=0
    OR position('vec_bolsa_llamamientos.respuesta_portal_llamamiento' in original)<>0 THEN
  RAISE EXCEPTION 'CT250 prueba: proyección sin origen O6 o lectura Bolsa no nominal'; END IF;
 -- Una emisión manual sin O6 exacta debe evaluar el origen como falso.
 SELECT count(*) INTO n FROM vec_contratacion_temporal.vinculo_emision_bolsa_ct_v1 v
 WHERE NOT EXISTS (SELECT 1 FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 e
   WHERE e.situacion='confirmada' AND e.solicitud_json->>'organizacion_ref'=v.organizacion_ref
     AND e.solicitud_json->>'expediente_ref'=v.expediente_ref
     AND e.recibo_json->>'llamamiento_ref'=v.llamamiento_ref
     AND e.recibo_json->>'recibo_ref'=v.recibo_emision_ref);
 RAISE NOTICE 'CT250 prueba: % vínculos sin O6 no acreditan origen nativo',n;
END $ct250$;
ROLLBACK;
