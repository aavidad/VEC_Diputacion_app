-- Solo copia sintética PG18 HZ24 posterior a CT210. No ejecutar en principal.
\set ON_ERROR_STOP on
\timing on

CREATE TEMP TABLE ct210_antes ON COMMIT PRESERVE ROWS AS
SELECT count(*) AS filas, max(evento_id) AS ultimo
FROM vec_contratacion_temporal.auditoria_frontera_ruta_exacta;

DO $contrato$
DECLARE f oid:=pg_catalog.to_regprocedure(
  'vec_contratacion_temporal.registrar_auditoria_frontera_ruta_exacta_v1(text,text,text,text,text)');
BEGIN
 IF pg_catalog.pg_get_userbyid((SELECT proowner FROM pg_catalog.pg_proc WHERE oid=f))
      <> 'vec_contratacion_temporal_propietario'
    OR NOT pg_catalog.has_function_privilege(
      'vec_contratacion_temporal_registrador_frontera',f,'EXECUTE')
    OR pg_catalog.has_table_privilege('vec_contratacion_temporal_registrador_frontera',
       'vec_contratacion_temporal.auditoria_frontera_ruta_exacta','SELECT')
    OR pg_catalog.has_table_privilege('vec_contratacion_temporal_registrador_frontera',
       'vec_contratacion_temporal.auditoria_frontera_ruta_exacta','INSERT') THEN
   RAISE EXCEPTION 'CT210 ACL no mínimas';
 END IF;
END $contrato$;

BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_registrador_frontera;
DO $ensayo$
DECLARE
  superficie constant text := 'api.bolsa.rrhh.documentos.ruta_exacta';
  prefijo constant text := '/api/vec/bolsa/bolsas/bolsa:01/documentos';
  correlacion constant text := 'corr_21021021021021021021021021021021';
  x record;
  rechazados integer := 0;
BEGIN
  FOR x IN SELECT * FROM (VALUES
    (prefijo, 'autenticacion_requerida', NULL::text),
    (prefijo || '/acta_01/recibo', 'acceso_denegado', 'actor:sintetico'),
    (prefijo, 'servicio_no_disponible', NULL::text),
    (prefijo || '/' || 'a' || pg_catalog.repeat('b',511) || '/recibo',
       'acceso_denegado','actor:sintetico')
  ) v(ruta,motivo,actor) LOOP
    IF vec_contratacion_temporal.registrar_auditoria_frontera_ruta_exacta_v1(
       correlacion,x.motivo,superficie,x.ruta,x.actor) IS NOT TRUE THEN
      RAISE EXCEPTION 'CT210 acuse positivo ausente';
    END IF;
  END LOOP;

  FOR x IN SELECT * FROM (VALUES
    (prefijo || '?limit=2','acceso_denegado',NULL::text,superficie),
    (prefijo || '/%61cta/recibo','acceso_denegado',NULL::text,superficie),
    (prefijo || '/a..b/recibo','acceso_denegado',NULL::text,superficie),
    (prefijo || '//recibo','acceso_denegado',NULL::text,superficie),
    (prefijo || '/_acta/recibo','acceso_denegado',NULL::text,superficie),
    (prefijo || '/' || 'a' || pg_catalog.repeat('b',512) || '/recibo',
       'acceso_denegado',NULL::text,superficie),
    ('/api/vec/bolsa/mi-bolsa','acceso_denegado',NULL::text,superficie),
    (prefijo,'autenticacion_requerida','actor:sintetico',superficie),
    ('/api/vec/contratacion-temporal/expedientes','servicio_no_disponible',NULL::text,
       'api.contratacion_temporal.ruta_exacta')
  ) v(ruta,motivo,actor,sup) LOOP
    BEGIN
      PERFORM vec_contratacion_temporal.registrar_auditoria_frontera_ruta_exacta_v1(
        correlacion,x.motivo,x.sup,x.ruta,x.actor);
      RAISE EXCEPTION 'CT210 variante ajena admitida';
    EXCEPTION WHEN invalid_parameter_value THEN rechazados := rechazados+1;
    END;
  END LOOP;
  IF rechazados <> 9 THEN RAISE EXCEPTION 'CT210 negativos incompletos'; END IF;
END $ensayo$;
COMMIT;

DO $post$
DECLARE antes record; actual record;
BEGIN
  SELECT * INTO STRICT antes FROM pg_temp.ct210_antes;
  SELECT count(*) AS filas,max(evento_id) AS ultimo INTO STRICT actual
  FROM vec_contratacion_temporal.auditoria_frontera_ruta_exacta;
  IF actual.filas<>antes.filas+4 OR actual.ultimo<antes.ultimo+4
     OR (SELECT count(*) FROM vec_contratacion_temporal.auditoria_frontera_ruta_exacta
         WHERE evento_id<=antes.ultimo)<>antes.filas
     OR (SELECT count(*) FROM vec_contratacion_temporal.auditoria_frontera_ruta_exacta
         WHERE evento_id>antes.ultimo
           AND superficie='api.bolsa.rrhh.documentos.ruta_exacta'
           AND correlacion_ref='corr_21021021021021021021021021021021')<>4 THEN
    RAISE EXCEPTION 'CT210 historia o cardinalidad divergente';
  END IF;
  RAISE NOTICE 'CT210-ENSAYO-OK: cuatro asientos, nueve negativos, historia previa intacta';
END $post$;
