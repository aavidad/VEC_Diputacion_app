\set ON_ERROR_STOP on
WITH filas AS (
  SELECT 'vec_autorizacion.decision' AS relacion,to_jsonb(t)::text AS fila
    FROM vec_autorizacion.decision_concedida_contexto_actor_v3 t
  UNION ALL SELECT 'vec_autorizacion.decision_denegada',to_jsonb(t)::text
    FROM vec_autorizacion.decision_denegada_contexto_actor_v3 t
  UNION ALL SELECT 'vec_autorizacion.enlace',to_jsonb(t)::text
    FROM vec_autorizacion.enlace_decision_cobertura_ct_o404e t
  UNION ALL SELECT 'ct.reserva',to_jsonb(t)::text
    FROM vec_contratacion_temporal.reserva_operacion_decision_cobertura t
  UNION ALL SELECT 'ct.alias',to_jsonb(t)::text
    FROM vec_contratacion_temporal.alias_operacion_decision_cobertura t
  UNION ALL SELECT 'ct.reserva_version',to_jsonb(t)::text
    FROM vec_contratacion_temporal.reserva_operacion_decision_cobertura_version t
  UNION ALL SELECT 'ct.reserva_actual',to_jsonb(t)::text
    FROM vec_contratacion_temporal.reserva_operacion_decision_cobertura_actual t
  UNION ALL SELECT 'ct.lote',to_jsonb(t)::text
    FROM vec_contratacion_temporal.consumo_cobertura_lote t
  UNION ALL SELECT 'ct.evidencia',to_jsonb(t)::text
    FROM vec_contratacion_temporal.consumo_cobertura_evidencia t
  UNION ALL SELECT 'ct.gobierno',to_jsonb(t)::text
    FROM vec_contratacion_temporal.acreditacion_gobierno_decision_cobertura t
  UNION ALL SELECT 'ct.gobi_actual',to_jsonb(t)::text
    FROM vec_contratacion_temporal.gobi_o404b_actual t
  UNION ALL SELECT 'ct.gobi_actuacion',to_jsonb(t)::text
    FROM vec_contratacion_temporal.gobi_o404b_actuacion t
  UNION ALL SELECT 'ct.gobi_catalogo',to_jsonb(t)::text
    FROM vec_contratacion_temporal.gobi_o404b_catalogo t
  UNION ALL SELECT 'ct.gobi_checkpoint',to_jsonb(t)::text
    FROM vec_contratacion_temporal.gobi_o404b_checkpoint t
  UNION ALL SELECT 'ct.gobi_evento',to_jsonb(t)::text
    FROM vec_contratacion_temporal.gobi_o404b_evento t
  UNION ALL SELECT 'ct.gobi_politica',to_jsonb(t)::text
    FROM vec_contratacion_temporal.gobi_o404b_politica t
  UNION ALL SELECT 'ct.gobi_retirada',to_jsonb(t)::text
    FROM vec_contratacion_temporal.gobi_o404b_retirada t
  UNION ALL SELECT 'ct.control_migracion',to_jsonb(t)::text
    FROM vec_contratacion_temporal.control_migracion_cobertura_o4 t
  UNION ALL SELECT 'ct.expediente_alta',to_jsonb(t)::text
    FROM vec_contratacion_temporal.expediente_alta t
  UNION ALL SELECT 'ct.expediente_version',to_jsonb(t)::text
    FROM vec_contratacion_temporal.expediente_version_integral t
  UNION ALL SELECT 'ct.expediente_actual',to_jsonb(t)::text
    FROM vec_contratacion_temporal.expediente_integral_actual t
  UNION ALL SELECT 'ct.actuacion',to_jsonb(t)::text
    FROM vec_contratacion_temporal.actuacion_expediente_integral t
  UNION ALL SELECT 'ct.decision_durable',to_jsonb(t)::text
    FROM vec_contratacion_temporal.decision_cobertura_gobernada_durable t
  UNION ALL SELECT 'ct.prueba_denegacion',to_jsonb(t)::text
    FROM vec_contratacion_temporal.prueba_denegacion_decision_cobertura t
  UNION ALL SELECT 'ct.auditoria',to_jsonb(t)::text
    FROM vec_contratacion_temporal.auditoria_decision_cobertura t
  UNION ALL SELECT 'ct.outbox',to_jsonb(t)::text
    FROM vec_contratacion_temporal.outbox_expediente_integral t
  UNION ALL SELECT 'ct.cadenas',to_jsonb(t)::text
    FROM vec_contratacion_temporal.control_cadenas_expediente_integral t
  UNION ALL SELECT 'ct.confirmacion',to_jsonb(t)::text
    FROM vec_contratacion_temporal.confirmacion_operacion_decision_cobertura t
  UNION ALL SELECT 'ct.terminal',to_jsonb(t)::text
    FROM vec_contratacion_temporal.terminal_operacion_decision_cobertura t
  UNION ALL SELECT 'fixture.denegada',to_jsonb(t)::text
    FROM vec_o404e_prueba.carga_denegada t
)
SELECT encode(sha256(convert_to(coalesce(string_agg(
  relacion||chr(31)||fila,chr(30) ORDER BY relacion,fila),''),'UTF8')),'hex')
  AS produccion
FROM filas
\gset
SELECT to_regclass('vec_o404e_concedida.cargas') IS NOT NULL
  AS existe_concedida
\gset
\if :existe_concedida
SELECT :'produccion'||'|'||encode(sha256(convert_to(coalesce(string_agg(
  to_jsonb(t)::text,chr(30) ORDER BY to_jsonb(t)::text),''),'UTF8')),'hex')
FROM vec_o404e_concedida.cargas t;
\else
SELECT :'produccion'||'|';
\endif
