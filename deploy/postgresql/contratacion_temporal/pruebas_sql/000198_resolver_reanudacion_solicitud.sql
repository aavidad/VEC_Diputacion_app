\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

SELECT 1 / ((
    has_function_privilege('vec_contratacion_temporal_ejecutor',
      'vec_contratacion_temporal.reanudar_solicitud_llamamiento_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)', 'EXECUTE')
    AND NOT has_function_privilege('vec_bolsa_llamamientos_ejecutor',
      'vec_contratacion_temporal.reanudar_solicitud_llamamiento_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)', 'EXECUTE')
    AND has_function_privilege('vec_contratacion_temporal_propietario',
      'vec_autorizacion_atestada_v3.registrar_y_consumir_reanudacion_solicitud_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)', 'EXECUTE')
    AND NOT has_function_privilege('vec_contratacion_temporal_ejecutor',
      'vec_autorizacion_atestada_v3.registrar_y_consumir_reanudacion_solicitud_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)', 'EXECUTE')
  )::integer) AS permisos_nominales_exactos;

SELECT 1 / ((count(*) = 4 AND bool_and('search_path=pg_catalog, pg_temp' = ANY(p.proconfig)))::integer)
  AS rutas_sql_sin_esquema_temporal_adelantado
FROM pg_proc p
WHERE p.oid IN (
 'vec_contratacion_temporal.reanudar_solicitud_llamamiento_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
 'vec_contratacion_temporal.confirmar_seleccion_llamamiento_o6_v1(uuid,text,text,text,text,text)'::regprocedure,
 'vec_contratacion_temporal.resolver_terminal_autorizado_seleccion_llamamiento_o6_v2(uuid,text)'::regprocedure,
 'vec_autorizacion_atestada_v3.registrar_y_consumir_reanudacion_solicitud_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
);

SELECT 1 / ((count(*) > 0 AND bool_and(
    vec_contratacion_temporal.confirmacion_canonica_seleccion_llamamiento_o6_v1(
      e.artefacto_canonico::jsonb, e.artefacto_canonico, e.recibo_json,
      vec_contratacion_temporal.recibo_json_seleccion_llamamiento_o6_v1(e.recibo_json)
    )))::integer) AS terminales_anteriores_intactos
FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 e
WHERE e.situacion = 'confirmada';

SELECT 1 / ((count(*) > 0 AND bool_and(
    vec_contratacion_temporal.recibo_desde_texto_seleccion_llamamiento_o6_v1(
      vec_contratacion_temporal.recibo_json_seleccion_llamamiento_o6_v1(
        e.recibo_json || '{"llamamiento_recuperado":true}'::jsonb
      )
    ) IS NOT NULL
    AND vec_contratacion_temporal.recibo_desde_texto_seleccion_llamamiento_o6_v1(
      vec_contratacion_temporal.recibo_json_seleccion_llamamiento_o6_v1(
        e.recibo_json || '{"llamamiento_recuperado":false}'::jsonb
      )
    ) IS NULL
  ))::integer) AS canon_marca_estricta
FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 e
WHERE e.situacion='confirmada';

-- La fila procede exclusivamente del clon sintético y vuelve intacta por ROLLBACK.
CREATE TEMP TABLE orq1_original ON COMMIT DROP AS
 SELECT e.clave_idempotencia, e.huella_semantica, e.reserva_ref,
        vec_contratacion_temporal.solicitud_json_seleccion_llamamiento_o6_v1(e.solicitud_json) AS solicitud_texto,
        vec_contratacion_temporal.recibo_json_seleccion_llamamiento_o6_v1(e.recibo_json) AS recibo_texto,
        e.artefacto_canonico AS artefacto_texto
 FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 e
 WHERE e.situacion='confirmada' ORDER BY e.creada_en LIMIT 1;
GRANT SELECT ON pg_temp.orq1_original TO vec_contratacion_temporal_ejecutor;

SELECT e.clave_idempotencia::text AS clave,
       e.huella_semantica AS huella,
       e.solicitud_json::text AS solicitud,
       '{"organizacion_ref":' || (e.solicitud_json->'organizacion_ref')::text ||
       ',"expediente_ref":' || (e.solicitud_json->'expediente_ref')::text ||
       ',"version_expediente":' || (e.solicitud_json->'version_expediente')::text ||
       ',"correlacion_ref":' || (e.solicitud_json->'correlacion_ref')::text ||
       ',"autoridad_solicitante":' || (e.solicitud_json->'autoridad_solicitante')::text ||
       ',"autorizacion":' || vec_contratacion_temporal.referencia_json_seleccion_llamamiento_o6_v1(e.solicitud_json->'autorizacion_consulta') ||
       ',"accion":' || vec_contratacion_temporal.referencia_json_seleccion_llamamiento_o6_v1(e.solicitud_json->'accion_consulta') ||
       ',"recurso":' || vec_contratacion_temporal.referencia_json_seleccion_llamamiento_o6_v1(e.solicitud_json->'recurso_consulta') ||
       ',"finalidad":' || vec_contratacion_temporal.referencia_json_seleccion_llamamiento_o6_v1(e.solicitud_json->'finalidad') || '}' AS consulta
  FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 e
 WHERE e.situacion = 'confirmada'
 ORDER BY e.creada_en LIMIT 1
\gset orq1_

SET LOCAL ROLE vec_contratacion_temporal_propietario;
UPDATE vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6
   SET situacion='indeterminada', efecto='solicitar_llamamiento',
       recibo_json=NULL, artefacto_canonico=NULL,
       actualizada_en=pg_catalog.date_trunc('microseconds',pg_catalog.clock_timestamp()-interval '31 seconds'),
       lease_hasta=pg_catalog.date_trunc('microseconds',pg_catalog.clock_timestamp()-interval '1 second')
 WHERE clave_idempotencia=:'orq1_clave'::uuid;
RESET ROLE;

SET LOCAL ROLE vec_contratacion_temporal_ejecutor;
SELECT 1 / ((v.situacion = '' AND v.solicitud_json = '' AND
              v.reserva_ref = '' AND v.efecto = '' AND
              v.recibo_json = '')::integer) AS segunda_ventana_recuperable
  FROM vec_contratacion_temporal.resolver_terminal_autorizado_seleccion_llamamiento_o6_v2(
       :'orq1_clave'::uuid, :'orq1_consulta') v;

RESET ROLE;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
UPDATE vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6
 SET situacion='propietaria',
     reserva_ref=vec_contratacion_temporal.nuevo_token_fencing_seleccion_llamamiento_o6_v2(),
     fencing_version=fencing_version+1,
     lease_hasta=pg_catalog.date_trunc('microseconds',pg_catalog.clock_timestamp()+interval '30 seconds'),
     actualizada_en=pg_catalog.date_trunc('microseconds',pg_catalog.clock_timestamp())
 WHERE clave_idempotencia=:'orq1_clave'::uuid;
RESET ROLE;
SET LOCAL ROLE vec_contratacion_temporal_ejecutor;
DO $fencing$
DECLARE v_original record;
BEGIN
 SELECT * INTO STRICT v_original FROM pg_temp.orq1_original;
 BEGIN
  PERFORM vec_contratacion_temporal.confirmar_seleccion_llamamiento_o6_v1(
   v_original.clave_idempotencia,v_original.huella_semantica,v_original.reserva_ref,
   v_original.solicitud_texto,v_original.recibo_texto,v_original.artefacto_texto);
  RAISE EXCEPTION 'CT198: reserva antigua confirmó efecto' USING ERRCODE='P0001';
 EXCEPTION WHEN SQLSTATE '55000' THEN NULL;
 END;
END
$fencing$;
SELECT 1 AS reserva_antigua_rechazada;
ROLLBACK;
