\set ON_ERROR_STOP on
BEGIN;
SET TRANSACTION READ ONLY;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';

-- Material inerte: comprueba bytes y huella; no crea recibos ni firma.
DO $canon$
DECLARE
 ref jsonb:=jsonb_build_object('referencia','ref:prueba','version',1,
  'huella_sha256',repeat('a',64));
 evidencia jsonb:=jsonb_build_object(
  'evidencia_ref','evidencia:prueba','clave_verificacion_ref','clave:prueba',
  'sello_hmac','sello:prueba','emitida_en','2026-10-20T08:00:00Z',
  'valida_hasta','2026-10-20T08:05:00Z','retener_hasta','0001-01-01T00:00:00Z');
 paginas jsonb:='[{"recibo_ref":"recibo:pagina:001","auditoria_ref":"aud:pagina:001"},
                 {"recibo_ref":"recibo:pagina:002","auditoria_ref":"aud:pagina:002"}]'::jsonb;
 recibo jsonb; anterior text; actual text; compacto text;
 mal jsonb; fuera jsonb;
BEGIN
 recibo:=jsonb_build_object(
  'operacion_ref','operacion:prueba','organizacion_ref','organizacion:prueba',
  'expediente_ref','expediente:prueba','version_expediente',1,
  'correlacion_ref','correlacion:prueba','necesidad',ref,'bolsa',ref,
  'orden',ref,'politica',ref,'resultado',ref,
  'propuesta_generada',true,'propuesta',ref,'accion_evento',ref,
  'llamamiento_ref','llamamiento:prueba','seleccion_ref','seleccion:prueba',
  'retencion_seleccion',ref,'orden_seleccionado',1,
  'recibo_ref','recibo:prueba','auditoria_ref','auditoria:prueba',
  'evento_ref','evento:prueba','confirmada_en','2026-10-20T08:01:00Z',
  'procedencia',jsonb_build_object(
   'autoridad_ref','autoridad:prueba','respuesta_ref','respuesta:prueba',
   'contrato_version',1,'fuente',ref,'evidencia',evidencia,
   'sin_retencion_declarada',true));
 anterior:=vec_contratacion_temporal.recibo_json_seleccion_llamamiento_o6_v1(recibo);
 compacto:='[{"recibo_ref":"recibo:pagina:001","auditoria_ref":"aud:pagina:001"},'||
  '{"recibo_ref":"recibo:pagina:002","auditoria_ref":"aud:pagina:002"}]';
 IF anterior IS NULL OR
  vec_contratacion_temporal.recibo_desde_texto_seleccion_llamamiento_o6_v1(anterior)
   IS DISTINCT FROM recibo OR
  vec_contratacion_temporal.consultas_personal_json_seleccion_o6_v1(recibo->'procedencia')
   IS DISTINCT FROM ''
 THEN RAISE EXCEPTION 'CT245: canon anterior alterado'; END IF;
 recibo:=jsonb_set(recibo,'{procedencia,consultas_personal}',paginas);
 actual:=vec_contratacion_temporal.recibo_json_seleccion_llamamiento_o6_v1(recibo);
 IF actual IS DISTINCT FROM replace(anterior,
  ',"sin_retencion_declarada":true}}',
  ',"sin_retencion_declarada":true,"consultas_personal":'||compacto||'}}')
  OR vec_contratacion_temporal.consultas_personal_json_seleccion_o6_v1(recibo->'procedencia')
    IS DISTINCT FROM compacto
  OR vec_contratacion_temporal.recibo_desde_texto_seleccion_llamamiento_o6_v1(actual)
    IS DISTINCT FROM recibo
 THEN RAISE EXCEPTION 'CT245: JSON compacto o orden distinto'; END IF;
 IF vec_contratacion_temporal.consultas_personal_json_seleccion_o6_v1(
  jsonb_set(recibo->'procedencia','{consultas_personal}',
   '[{"recibo_ref":"recibo:pagina:002","auditoria_ref":"aud:pagina:002"},
     {"recibo_ref":"recibo:pagina:001","auditoria_ref":"aud:pagina:001"}]'::jsonb))
  IS NOT DISTINCT FROM compacto
 THEN RAISE EXCEPTION 'CT245: se perdió el orden de páginas'; END IF;
 FOREACH mal IN ARRAY ARRAY[
  '[]'::jsonb,'null'::jsonb,'{}'::jsonb,
  '[{"recibo_ref":"recibo:uno"}]'::jsonb,
  '[{"recibo_ref":"recibo:uno","auditoria_ref":"aud:uno","extra":1}]'::jsonb,
  '[{"recibo_ref":"x","auditoria_ref":"aud:uno"}]'::jsonb,
  '[{"recibo_ref":null,"auditoria_ref":"aud:uno"}]'::jsonb,
  (SELECT jsonb_agg(jsonb_build_object('recibo_ref','recibo:uno',
    'auditoria_ref','aud:uno')) FROM generate_series(1,65))
 ] LOOP
  fuera:=jsonb_set(recibo,'{procedencia,consultas_personal}',mal);
  IF vec_contratacion_temporal.consultas_personal_json_seleccion_o6_v1(
   fuera->'procedencia') IS NOT NULL OR
   vec_contratacion_temporal.recibo_json_seleccion_llamamiento_o6_v1(fuera) IS NOT NULL
  THEN RAISE EXCEPTION 'CT245: página malformada admitida'; END IF;
 END LOOP;
 fuera:=jsonb_set(recibo,'{procedencia,sin_retencion_declarada}','false'::jsonb);
 IF vec_contratacion_temporal.consultas_personal_json_seleccion_o6_v1(
  fuera->'procedencia') IS NOT NULL OR
  vec_contratacion_temporal.recibo_json_seleccion_llamamiento_o6_v1(fuera) IS NOT NULL
 THEN RAISE EXCEPTION 'CT245: lectura admitida fuera de circuito nativo'; END IF;
END $canon$;
ROLLBACK;
