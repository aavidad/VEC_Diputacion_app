\set ON_ERROR_STOP on
-- Sólo datos sintéticos en clon desechable. La transacción no deja historia.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
CREATE TEMP TABLE ct167_casos (
 tipo text, ambitos jsonb, organizacion_ref text, expediente_ref text,
 actor_ref text, perfil_ref text, operacion text, version_anterior numeric,
 esperado jsonb
) ON COMMIT DROP;
-- El agregado mínimo del caso sintético no se publica en la bandeja RRHH.
-- El trigger se desactiva únicamente en esta transacción desechable.
ALTER TABLE vec_contratacion_temporal.expediente_version_integral
 DISABLE TRIGGER expediente_version_integral_publicar_rrhh;

DO $preparar$
DECLARE
 v_exp text; v_version numeric; v_root text;
 v_organizacion text:='org:ct167:sintetica';
 v_actor text:='actor:ct167:sintetico';
 v_perfil text:='perfil:ct167:sintetico';
 v_recibo text:='recibo:ct167:sintetico';
 v_agregado jsonb;
 v_recibo_json jsonb;
 v_prueba bytea;
 v_tiempo timestamptz(6):=pg_catalog.date_trunc('microseconds',pg_catalog.clock_timestamp());
 v_huella text;
BEGIN
 SELECT x.expediente_ref,pg_catalog.max(x.version)
   INTO v_exp,v_version
   FROM vec_contratacion_temporal.expediente_version_integral x
  GROUP BY x.expediente_ref ORDER BY pg_catalog.max(x.version) DESC LIMIT 1;
 IF v_exp IS NULL THEN RAISE EXCEPTION 'CT167 clon sin expediente sintético'; END IF;
 v_root:='hmac-sha256:vec.contratacion-temporal.analisis.ambito-idempotencia/v1:'||pg_catalog.repeat('a',64);
 v_agregado:='{"analisis":{"periodo":{"inicio":"2026-10-02T00:00:00Z","causa_fin":"reincorporacion_titular","politica_fin":{"regla_ref":"regla:modalidad:sustitucion","catalogo_version":2,"catalogo_huella_sha256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","fecha_fin":"no_aplica","causa_fin":"reincorporacion_titular"}}}}'::jsonb;
 v_recibo_json:=pg_catalog.jsonb_build_object(
   'recibo_ref',v_recibo,'organizacion_ref',v_organizacion,
   'expediente_ref',v_exp,'version_resultante',v_version+1);
 v_prueba:=pg_catalog.convert_to(pg_catalog.repeat('x',128),'UTF8');
 v_huella:=pg_catalog.encode(pg_catalog.sha256(
   pg_catalog.convert_to(v_agregado::text,'UTF8')),'hex');
 SET CONSTRAINTS ALL DEFERRED;
 INSERT INTO vec_contratacion_temporal.reserva_operacion_analisis (
   ambito_raiz_hmac,reserva_ref,recibo_ref,operacion,organizacion_ref,
   expediente_ref,version_expediente,actor_ref,perfil_ref,artefacto_ref,
   artefacto_huella_sha256,huella_semantica_raiz_hmac,creada_en)
 VALUES (v_root,'reserva:ct167:sintetica',v_recibo,'registrar',v_organizacion,
   v_exp,v_version,v_actor,v_perfil,'artefacto:ct167:sintetico',
   pg_catalog.repeat('c',64),
   'hmac-sha256:vec.contratacion-temporal.analisis.huella-semantica/v1:'||
     pg_catalog.repeat('d',64),v_tiempo);
 INSERT INTO vec_contratacion_temporal.expediente_version_integral (
   expediente_ref,version,agregado_json,agregado_json_huella_sha256,
   prueba_canonica,prueba_huella_sha256,flujo_ref,flujo_version,
   flujo_huella_sha256,fase_clave,estado,origen_version,operacion_ref,
   registrada_en)
 VALUES (v_exp,v_version+1,v_agregado,v_huella,v_prueba,
   pg_catalog.encode(pg_catalog.sha256(v_prueba),'hex'),
   'flujo:ct167:sintetico',1,pg_catalog.repeat('e',64),'solicitud',
   'en_curso','analisis_o3',
   'operacion:analisis:'||pg_catalog.substr(pg_catalog.encode(pg_catalog.sha256(
      pg_catalog.convert_to(v_root||':'||v_recibo,'UTF8')),'hex'),1,32),
   v_tiempo);
 INSERT INTO vec_contratacion_temporal.confirmacion_operacion_analisis (
   ambito_raiz_hmac,recibo_json,recibo_huella_sha256,confirmada_en)
 VALUES (v_root,v_recibo_json,pg_catalog.encode(pg_catalog.sha256(
   pg_catalog.convert_to(v_recibo_json::text,'UTF8')),'hex'),v_tiempo);
 INSERT INTO vec_contratacion_temporal.alias_consulta_operacion_analisis (
   alias_ambito_consulta_hmac,ambito_raiz_hmac,generacion,registrada_en)
 VALUES (v_root,v_root,1,v_tiempo);
 INSERT INTO ct167_casos VALUES (
   'analisis_nuevo',pg_catalog.jsonb_build_array(v_root),v_organizacion,
   v_exp,v_actor,v_perfil,'registrar',v_version,
   v_agregado#>'{analisis,periodo,politica_fin}');

 INSERT INTO ct167_casos
 SELECT 'analisis_legacy',pg_catalog.jsonb_build_array(a.alias_ambito_consulta_hmac),
   r.organizacion_ref,r.expediente_ref,r.actor_ref,r.perfil_ref,
   r.operacion,r.version_expediente,NULL::jsonb
 FROM vec_contratacion_temporal.alias_consulta_operacion_analisis a
 JOIN vec_contratacion_temporal.reserva_operacion_analisis r
   ON r.ambito_raiz_hmac=a.ambito_raiz_hmac
 JOIN vec_contratacion_temporal.confirmacion_operacion_analisis c
   ON c.ambito_raiz_hmac=r.ambito_raiz_hmac
 JOIN vec_contratacion_temporal.expediente_version_integral v
   ON v.expediente_ref=r.expediente_ref AND v.version=r.version_expediente+1
 WHERE v.agregado_json#>'{analisis,periodo,politica_fin}' IS NULL
   AND r.ambito_raiz_hmac<>v_root LIMIT 1;
 INSERT INTO ct167_casos
 SELECT 'alta_legacy',pg_catalog.jsonb_build_array(a.alias_hmac),
   i.organizacion_ref,'',i.actor_ref,i.perfil_ref,NULL,NULL,NULL::jsonb
 FROM vec_contratacion_temporal.alias_ambito_alta a
 JOIN vec_contratacion_temporal.identidad_reserva_alta i
   ON i.ambito_hmac=a.ambito_raiz_hmac
 JOIN vec_contratacion_temporal.confirmacion_agregado_alta c
   ON c.ambito_hmac=i.ambito_hmac
 JOIN vec_contratacion_temporal.expediente_version_integral v
   ON v.expediente_ref=c.expediente_ref AND v.version=1
 WHERE v.agregado_json#>'{solicitud,periodo,politica_fin}' IS NULL LIMIT 1;
 IF NOT EXISTS (SELECT 1 FROM ct167_casos WHERE tipo='analisis_legacy')
    OR NOT EXISTS (SELECT 1 FROM ct167_casos WHERE tipo='alta_legacy') THEN
    RAISE EXCEPTION 'CT167 clon sin casos legacy confirmados';
 END IF;
END
$preparar$;

GRANT SELECT ON ct167_casos TO vec_contratacion_temporal_ejecutor;
SET SESSION AUTHORIZATION vec_contratacion_temporal_ejecutor;
DO $comprobar$
DECLARE r record; v_politica jsonb; v_count integer; v_conflicto boolean;
BEGIN
 FOR r IN SELECT * FROM ct167_casos LOOP
   IF r.tipo='alta_legacy' THEN
     SELECT pg_catalog.count(*),pg_catalog.max(politica_fin::text)::jsonb
       INTO v_count,v_politica
       FROM vec_contratacion_temporal.consultar_politica_fin_alta_confirmada_v1(
         r.ambitos,r.organizacion_ref,r.expediente_ref,r.actor_ref,r.perfil_ref);
   ELSE
     SELECT pg_catalog.count(*),pg_catalog.max(politica_fin::text)::jsonb
       INTO v_count,v_politica
       FROM vec_contratacion_temporal.consultar_politica_fin_analisis_confirmado_v1(
         r.ambitos,r.organizacion_ref,r.expediente_ref,r.actor_ref,
         r.perfil_ref,r.operacion,r.version_anterior);
   END IF;
   IF v_count<>1 OR v_politica IS DISTINCT FROM r.esperado THEN
      RAISE EXCEPTION 'CT167 caso % incorrecto',r.tipo;
   END IF;
 END LOOP;
 SELECT * INTO r FROM ct167_casos WHERE tipo='analisis_nuevo';
 v_conflicto:=false;
 BEGIN
   PERFORM * FROM vec_contratacion_temporal.consultar_politica_fin_analisis_confirmado_v1(
     r.ambitos,r.organizacion_ref,r.expediente_ref,'actor:otro',
     r.perfil_ref,r.operacion,r.version_anterior);
 EXCEPTION WHEN unique_violation THEN v_conflicto:=true;
 END;
 IF NOT v_conflicto THEN RAISE EXCEPTION 'CT167 no detectó conflicto de actor'; END IF;
 SELECT pg_catalog.count(*) INTO v_count
 FROM vec_contratacion_temporal.consultar_politica_fin_analisis_confirmado_v1(
   pg_catalog.jsonb_build_array('hmac-sha256:vec.contratacion-temporal.analisis.ambito-idempotencia/v1:'||pg_catalog.repeat('f',64)),
   r.organizacion_ref,r.expediente_ref,r.actor_ref,r.perfil_ref,
   r.operacion,r.version_anterior);
 IF v_count<>0 THEN RAISE EXCEPTION 'CT167 reveló snapshot sin confirmación'; END IF;
 SELECT * INTO r FROM ct167_casos WHERE tipo='alta_legacy';
 v_conflicto:=false;
 BEGIN
   PERFORM * FROM vec_contratacion_temporal.consultar_politica_fin_alta_confirmada_v1(
     r.ambitos,r.organizacion_ref,'','actor:otro',r.perfil_ref);
 EXCEPTION WHEN unique_violation THEN v_conflicto:=true;
 END;
 IF NOT v_conflicto THEN RAISE EXCEPTION 'CT167 alta no detectó conflicto de actor'; END IF;
 SELECT pg_catalog.count(*) INTO v_count
 FROM vec_contratacion_temporal.consultar_politica_fin_alta_confirmada_v1(
   pg_catalog.jsonb_build_array('hmac-sha256:vec.contratacion-temporal.ambito-idempotencia/v1:'||pg_catalog.repeat('f',64)),
   r.organizacion_ref,'',r.actor_ref,r.perfil_ref);
 IF v_count<>0 THEN RAISE EXCEPTION 'CT167 alta reveló snapshot sin confirmación'; END IF;
 SELECT * INTO r FROM ct167_casos WHERE tipo='analisis_nuevo';
 v_conflicto:=false;
 BEGIN
   PERFORM * FROM vec_contratacion_temporal.consultar_politica_fin_analisis_confirmado_v1(
     r.ambitos || (SELECT ambitos FROM ct167_casos WHERE tipo='analisis_legacy'),
     r.organizacion_ref,r.expediente_ref,r.actor_ref,r.perfil_ref,
     r.operacion,r.version_anterior);
 EXCEPTION WHEN unique_violation THEN v_conflicto:=true;
 END;
 IF NOT v_conflicto THEN RAISE EXCEPTION 'CT167 no detectó alias ambiguos'; END IF;
END
$comprobar$;
RESET SESSION AUTHORIZATION;
ROLLBACK;
