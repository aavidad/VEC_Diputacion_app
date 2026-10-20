\set ON_ERROR_STOP on
-- Ejecutar tras CT250 en la copia local de la principal.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $ct250$
DECLARE p pg_proc%ROWTYPE; original text; v_ok boolean;
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
       <>'82a116f0069fb7220fa81e6411b01ea6fde4a474f66e3f5863f8bb2d944485b1' THEN
  RAISE EXCEPTION 'CT250 prueba: definición, dueño o ACL divergentes'; END IF;
 original:=p.prosrc;
 IF position('consultar_detalle_rrhh_atestado_v1' in original)=0
    OR position('leer_resultados_emisiones_ct_v1' in original)=0
    OR position('origen_ct_confirmado' in original)=0
    OR position('e.situacion=''confirmada''' in original)=0
    OR position('sin_retencion_declarada' in original)<>0
    OR position('o6 AS MATERIALIZED' in original)=0
    OR position('h.version=CASE' in original)=0
    OR position('e.version_recibo=to_jsonb(x.version_expediente_esperada)' in original)<>0
    OR position('jsonb_object_agg(llamamiento_ref,cantidad=1)' in original)=0
    OR position('jsonb_array_length(b.valor->''participaciones'')=1' in original)=0
    OR position('part.valor->>''respuesta''=''acepta''' in original)=0
    OR position('part.valor->>''modo''=''firme''' in original)=0
    OR position('vec_bolsa_llamamientos.respuesta_portal_llamamiento' in original)<>0 THEN
  RAISE EXCEPTION 'CT250 prueba: proyección sin origen O6 o lectura Bolsa no nominal'; END IF;
 -- Datos sintéticos: O6 ordinaria o sin retención en v6, vínculo CT201 en v7;
 -- emisión manual sin O6. Ningún caso depende de la versión actual.
 WITH vinculos(tipo,llamamiento,version_vinculo) AS (VALUES
   ('ordinaria','llamamiento:uno',7),('sin_retencion','llamamiento:dos',7),
   ('manual','llamamiento:tres',7)),
 o6 AS (
  SELECT v.tipo,v.llamamiento,6 AS version_o6,
   jsonb_build_object('organizacion_ref','org:uno','expediente_ref','exp:uno',
    'version_expediente',6) AS solicitud,
   jsonb_build_object('organizacion_ref','org:uno','expediente_ref','exp:uno',
    'version_expediente',6,'bolsa',jsonb_build_object('referencia','bolsa:uno'),
    'llamamiento_ref',v.llamamiento,'recibo_ref','recibo:'||v.llamamiento,
    'propuesta_generada',true,'procedencia',v.procedencia) AS recibo
  FROM (VALUES ('ordinaria','llamamiento:uno','{}'::jsonb),
   ('sin_retencion','llamamiento:dos','{"sin_retencion_declarada":true}'::jsonb))
   v(tipo,llamamiento,procedencia)),
 resultados AS (
  SELECT v.tipo,count(o.tipo)=1 AS confirmado FROM vinculos v LEFT JOIN o6 o
   ON o.llamamiento=v.llamamiento AND o.recibo->>'recibo_ref'='recibo:'||v.llamamiento
   AND o.recibo->'version_expediente'=o.solicitud->'version_expediente'
   AND o.version_o6=6 AND o.version_o6<>v.version_vinculo
  GROUP BY v.tipo)
 SELECT bool_and(CASE WHEN tipo='manual' THEN NOT confirmado ELSE confirmado END)
 INTO v_ok FROM resultados;
 IF v_ok IS NOT TRUE THEN RAISE EXCEPTION 'CT250 prueba: clasificación O6 ordinaria/manual divergente'; END IF;
 SELECT * INTO p FROM pg_proc WHERE oid=to_regprocedure(
  'vec_contratacion_temporal.registrar_vinculo_emision_bolsa_ct_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF NOT FOUND OR p.proowner<>'vec_contratacion_temporal_propietario'::regrole
    OR p.prosecdef IS NOT TRUE OR p.provolatile<>'v'
    OR p.proacl::text IS DISTINCT FROM '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario}'
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s']
    OR encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex')
      <>'9f8ccd5cf39f77be1f33607ab63041115eaacb76fa62fad1f242b9815e356ae3'
    OR position('numero:=s->>''expediente_ref''' in p.prosrc)=0
    OR position('numero_visible_vigente_v1' in p.prosrc)=0
    OR position('verificar_emision_ct_v1' in p.prosrc)=0 THEN
  RAISE EXCEPTION 'CT250 prueba: vínculo opaco/manual o autoridad divergente'; END IF;
END $ct250$;
ROLLBACK;
