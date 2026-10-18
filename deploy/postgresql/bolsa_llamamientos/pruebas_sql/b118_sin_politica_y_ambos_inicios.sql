\set ON_ERROR_STOP on
-- Copia desechable con CT221, B117 y B118 instaladas; no confirma actos V3.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
DO $prueba$ DECLARE
 v_bolsa constant text:='bolsa:administrativo:algicjlleggibhgagmjmoipmakdijibk';
 v_categoria constant text:='categoria:rpt:administrativo';
 v_original record;v_contextual record;v_entradas jsonb;
 v_orden_general jsonb;v_orden_contextual jsonb;
 v_politica jsonb;v_inicio text;v_rechazo boolean;
BEGIN
 IF EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.politica_ofertas_version
   WHERE bolsa_ref=v_bolsa) THEN
  RAISE EXCEPTION 'B118: fixture sin política alterado'; END IF;
 SELECT * INTO STRICT v_original
 FROM vec_bolsa_llamamientos.leer_origen_llamamiento_ct_v1(v_categoria);
 SELECT * INTO STRICT v_contextual
 FROM vec_bolsa_llamamientos.leer_origen_llamamiento_ct_contextual_v1(
  v_categoria,'vacante','vacante');
 SELECT jsonb_agg(j.valor-'eventos_relacion' ORDER BY (j.valor->>'orden_acta')::bigint)
 INTO v_entradas FROM jsonb_array_elements(v_contextual.entradas) j(valor);
 IF v_contextual.bolsa_ref IS DISTINCT FROM v_bolsa
 OR v_contextual.criterios_ofrecimiento IS NOT NULL
 OR v_entradas IS DISTINCT FROM v_original.entradas THEN
  RAISE EXCEPTION 'B118: sin política cambió la lectura B110'; END IF;
 SELECT jsonb_agg(to_jsonb(o) ORDER BY o.orden_acta) INTO v_orden_general
 FROM vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(
  v_bolsa,v_contextual.corte) o;
 SELECT jsonb_agg(to_jsonb(o)-'eventos_relacion' ORDER BY o.orden_acta)
 INTO v_orden_contextual
 FROM vec_bolsa_llamamientos.leer_orden_contextual_oferta_v1(
  v_bolsa,'vacante','vacante',v_contextual.corte) o;
 IF v_orden_contextual IS DISTINCT FROM v_orden_general THEN
  RAISE EXCEPTION 'B118: sin política cambió el orden B6'; END IF;

 v_politica:='{"plazo":{"unidad":"dias_naturales","cantidad":2,
  "computo":"administrativo","municipio_sede":"18087","inicio":"notificacion"},
  "adjudicacion":{"criterio":"orden_vigente",
   "elegibilidad":"disposicion_en_plazo","confirmacion":"aceptacion_previa"},
  "no_cubierta":{"accion":"llamamiento_directo",
   "condicion":"sin_disposiciones_elegibles"},
  "criterios_ofrecimiento":{}}'::jsonb;
 FOREACH v_inicio IN ARRAY ARRAY['notificacion','publicacion'] LOOP
  v_rechazo:=false;
  BEGIN
   PERFORM vec_bolsa_llamamientos.publicar_politica_ofertas_v1(
    v_bolsa,0,jsonb_set(v_politica,'{plazo,inicio}',to_jsonb(v_inicio)),
    'per_'||repeat('a',22),'b118test',
    'recibo:politica-ofertas:'||repeat('a',64),
    decode(repeat('00',32),'hex'),decode(repeat('00',32),'hex'),
    decode(repeat('00',32),'hex'),decode(repeat('00',32),'hex'),
    1,1,decode(repeat('00',32),'hex'),decode(repeat('00',32),'hex'),
    decode(repeat('00',32),'hex'),decode(repeat('00',32),'hex'));
  EXCEPTION WHEN others THEN
   IF SQLERRM LIKE 'B47: política de ejemplo inválida%'
      OR SQLERRM LIKE 'B71: inicio de plazo no configurado%' THEN
    RAISE EXCEPTION 'B118: inicio % rechazado antes de V3: %',v_inicio,SQLERRM;
   END IF;
   v_rechazo:=true;
  END;
  IF NOT v_rechazo THEN
   RAISE EXCEPTION 'B118: material V3 ficticio aceptado'; END IF;
 END LOOP;
END $prueba$;
ROLLBACK;
