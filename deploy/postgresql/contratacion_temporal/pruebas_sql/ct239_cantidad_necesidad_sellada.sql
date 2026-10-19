\set ON_ERROR_STOP on
-- Prueba de funciones puras. ROLLBACK: no reproduce firma ni acto V3.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL statement_timeout='30s';
DO $ct239$
DECLARE
  raw bytea := pg_catalog.convert_to(
   '{"esquema":"vec.ct.necesidades_alta.v1","referencia":"catalogo:ct:ejemplo","version":2,"jornada_referencia_minutos":2250,"causas":[{"clave":"sustitucion","regla_ref":"regla:ct:sustitucion:v1","fecha_fin":"opcional","causa_fin":"reincorporacion_titular","maximo_meses":36,"campos_permitidos":["numero_personas","puesto_codigo"],"campos_obligatorios":["numero_personas","puesto_codigo"]}]}',
   'UTF8');
  n jsonb;
  s jsonb;
  v2 jsonb;
  v_legacy text;
  v_nuevo text;
  v_agregado jsonb;
  v_agregado_plural jsonb;
  v_periodo_utc jsonb;
  v_total bigint;
  v_distintos bigint;
BEGIN
 n:=pg_catalog.jsonb_build_object(
  'esquema','vec.ct.necesidad_alta.v1',
  'catalogo_ref','catalogo:ct:ejemplo','catalogo_version',2,
  'catalogo_huella_sha256',pg_catalog.encode(pg_catalog.sha256(raw),'hex'),
  'causa_clave','sustitucion',
  'periodo',pg_catalog.jsonb_build_object('inicio','2026-10-01','fin','2026-10-31'),
  'jornada_minutos',2250,
  'campos',pg_catalog.jsonb_build_object('numero_personas','2','puesto_codigo','auxiliar'),
  'catalogo_instantanea',pg_catalog.replace(pg_catalog.encode(raw,'base64'),E'\n',''));
 s:=pg_catalog.jsonb_build_object(
  'centro_ref','centro:ejemplo','contacto_ref','contacto:ejemplo',
  'categoria_ref','categoria:ejemplo','grupo_subgrupo','C2',
  'motivo_clave','sustitucion','detalle','Prueba',
  'periodo',n->'periodo',
  'rc',pg_catalog.jsonb_build_object('existe',false,'numero','','fecha','',
       'importe',pg_catalog.jsonb_build_object('centimos',0,'moneda','EUR'),
       'documento_ref',''),
  'documentos_adjuntos','[]'::jsonb,'observaciones','','necesidad',n);
 v2:=s||'{"numero_personas":2}'::jsonb;
 IF vec_contratacion_temporal.necesidad_alta_valida_v3(s) IS NOT TRUE
    OR vec_contratacion_temporal.necesidad_alta_valida_v3(v2) IS NOT TRUE
    OR vec_contratacion_temporal.datos_puesto_peticion_validos_ct200(s) IS NOT TRUE
    OR vec_contratacion_temporal.datos_puesto_peticion_validos_ct200(v2) IS NOT TRUE
    OR vec_contratacion_temporal.datos_puesto_peticion_validos_ct200(
       pg_catalog.jsonb_set(v2,'{numero_personas}','3'::jsonb)) IS NOT FALSE
    OR vec_contratacion_temporal.datos_puesto_peticion_validos_ct200(
       pg_catalog.jsonb_set(v2,'{numero_personas}','0'::jsonb)) IS NOT FALSE
    OR vec_contratacion_temporal.datos_puesto_peticion_validos_ct200(
       pg_catalog.jsonb_set(v2,'{numero_personas}','2.0'::jsonb)) IS NOT FALSE
    OR vec_contratacion_temporal.datos_puesto_peticion_validos_ct200(
       pg_catalog.jsonb_set(v2,'{numero_personas}','true'::jsonb)) IS NOT FALSE
    OR vec_contratacion_temporal.datos_puesto_peticion_validos_ct200(
       pg_catalog.jsonb_set(v2,'{numero_personas}','null'::jsonb)) IS NOT FALSE
    OR vec_contratacion_temporal.datos_puesto_peticion_validos_ct200(
       v2-'necesidad') IS NOT FALSE
    OR vec_contratacion_temporal.datos_puesto_peticion_validos_ct200(
       v2||'{"jornada_minutos":2250}'::jsonb) IS NOT FALSE
    OR vec_contratacion_temporal.datos_puesto_peticion_validos_ct200(
       pg_catalog.jsonb_set(v2,'{necesidad,jornada_minutos}','0'::jsonb)) IS NOT FALSE
    OR vec_contratacion_temporal.datos_puesto_peticion_validos_ct200(
       pg_catalog.jsonb_set(v2,'{necesidad,campos,numero_personas}','"3"'::jsonb)) IS NOT FALSE THEN
   RAISE EXCEPTION 'CT239: admision de cantidad sellada divergente';
 END IF;
 IF vec_contratacion_temporal.datos_puesto_peticion_validos_ct200(
       '{"jornada_minutos":2250,"numero_personas":2,"puesto_solicitado":"Administrativo C2"}'::jsonb) IS NOT TRUE
    OR vec_contratacion_temporal.datos_puesto_peticion_validos_ct200(
       '{"jornada_minutos":2250,"numero_personas":2}'::jsonb) IS NOT FALSE THEN
   RAISE EXCEPTION 'CT239: terna anterior divergente';
 END IF;
 v_legacy:=vec_contratacion_temporal.reconstruir_solicitud_efecto_v3(s);
 v_nuevo:=vec_contratacion_temporal.reconstruir_solicitud_efecto_v3(v2);
 IF v_legacy LIKE '%"numero_personas":2,"necesidad":%'
    OR v_nuevo IS DISTINCT FROM pg_catalog.replace(v_legacy,
       ',"necesidad":',',"numero_personas":2,"necesidad":')
    OR (v_nuevo::jsonb #>> '{numero_personas}') IS DISTINCT FROM '2'
    OR (v_nuevo::jsonb #>> '{necesidad,campos,numero_personas}') IS DISTINCT FROM '2'
    OR (v_nuevo::jsonb #>> '{numero_personas}') IS DISTINCT FROM
       (v_nuevo::jsonb #>> '{necesidad,campos,numero_personas}') THEN
   RAISE EXCEPTION 'CT239: canon V1/V2 o cantidad CT235 divergente';
 END IF;
 -- Una versión integral válida anterior sirve de estructura real para CT202;
 -- la variante se comprueba solo en memoria y se revierte con la transacción.
 SELECT agregado_json::jsonb INTO v_agregado
   FROM vec_contratacion_temporal.expediente_version_integral
  WHERE vec_contratacion_temporal.expediente_analisis_valido_v2(
          agregado_json::jsonb,false) IS TRUE
    AND NOT ((agregado_json::jsonb->'solicitud') ? 'necesidad')
  LIMIT 1;
 IF v_agregado IS NULL THEN
   RAISE EXCEPTION 'CT239: falta agregado V1 válido para analizar';
 END IF;
 v_periodo_utc:=v_agregado #> '{solicitud,periodo}';
 n:=pg_catalog.jsonb_set(n,'{periodo}',
   pg_catalog.jsonb_build_object(
     'inicio',pg_catalog.left(v_periodo_utc->>'inicio',10),
     'fin',pg_catalog.left(v_periodo_utc->>'fin',10)));
 v_agregado_plural:=pg_catalog.jsonb_set(v_agregado,'{solicitud}',
   (v_agregado->'solicitud') || pg_catalog.jsonb_build_object(
     'motivo_clave','sustitucion','numero_personas',2,
     'necesidad',pg_catalog.jsonb_set(n,'{periodo}',v_periodo_utc)));
 IF vec_contratacion_temporal.expediente_analisis_valido_v2(
      v_agregado_plural,false) IS NOT TRUE
    OR vec_contratacion_temporal.expediente_analisis_valido_v2(
      pg_catalog.jsonb_set(v_agregado_plural,
        '{solicitud,numero_personas}','3'::jsonb),false) IS NOT FALSE
    OR vec_contratacion_temporal.expediente_analisis_valido_v2(
      v_agregado_plural #- '{solicitud,numero_personas}',false) IS NOT TRUE THEN
   RAISE EXCEPTION 'CT239: analisis CT202 divergente';
 END IF;
 SELECT pg_catalog.count(*), pg_catalog.count(*) FILTER (WHERE
   vec_contratacion_temporal.reconstruir_efecto_segun_esquema_v3(
    pg_catalog.convert_from(alta_canonica,'UTF8')::jsonb) IS DISTINCT FROM alta_canonica)
 INTO v_total,v_distintos
 FROM vec_contratacion_temporal.expediente_alta_version;
 IF v_total=0 OR v_distintos<>0 THEN
  RAISE EXCEPTION 'CT239: historia previa divergente %/%',v_distintos,v_total;
 END IF;
END $ct239$;
ROLLBACK;
