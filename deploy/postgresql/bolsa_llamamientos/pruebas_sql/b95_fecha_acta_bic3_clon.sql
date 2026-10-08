-- B95: el helper BIC3 reemplaza sólo la fecha técnica registrada_en.
-- Ejecutar como DBA en clon PG18 desechable; todos los efectos se revierten.
\set ON_ERROR_STOP on
SELECT count(*) AS lotes_antes FROM vec_bolsa_importacion_convoca.lote \gset
SELECT count(*) AS filas_antes FROM vec_bolsa_importacion_convoca.fila_staging \gset
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL TIME ZONE 'UTC';
DO $prueba$
DECLARE
 huella text:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to('vec-b95-fecha-rollback-20261008','UTF8')),'hex');
 categoria text:='categoria:rpt:administrativo';
 ref text; entrada jsonb; primera record; repetida record;
BEGIN
 ref:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(huella||pg_catalog.chr(31)||categoria,'UTF8')),'hex');
 entrada:=pg_catalog.jsonb_build_object(
  'acta_ref','acta:importacion-convoca:'||ref,
  'importacion_ref','importacion:convoca:'||ref,
  'huella_fichero_sha256',huella,
  'fichero_custodiado_ref','original:convoca:'||ref,
  'nombre_fichero','ensayo-fecha.xls',
  'actor_ref','actor:rrhh:ensayo-fecha',
  'categoria_ref',categoria,
  'bolsa_ref','bolsa:ensayo-fecha',
  'esquema','convoca_resumen_persona_v1',
  'filas_leidas',0,'filas_aceptadas',0,'filas_rechazadas',0,
  'incidencias','[]'::jsonb,
  'procedencia',pg_catalog.jsonb_build_object(
   'esquema','vec.bolsa.importacion-convoca.procedencia.v1',
   'fuente','Convoca (exportacion enmascarada)',
   'autoridad','no_autoritativa',
   'habilita_actos_con_efectos',false,
   'requiere_confirmacion_registro',true,
   'uso_puntos_autobaremacion','historico_contraste'),
  'registrada_en','2026-10-08T12:00:00.000000Z');
 IF vec_bolsa_importacion_convoca.acta_valida(entrada) IS NOT TRUE
 OR vec_bolsa_importacion_convoca.filas_protegidas_validas('[]'::jsonb,0) IS NOT TRUE
 THEN RAISE EXCEPTION 'B95: fixture BIC3 inválida' USING ERRCODE='55000'; END IF;
 SELECT * INTO STRICT primera FROM vec_bolsa_importacion_convoca.guardar_lote_v1(entrada,'[]'::jsonb);
 IF primera.reutilizada IS NOT FALSE
 OR primera.acta_canonica - 'registrada_en' IS DISTINCT FROM entrada - 'registrada_en'
 OR primera.acta_canonica->>'registrada_en' IS NOT DISTINCT FROM entrada->>'registrada_en'
 OR vec_bolsa_importacion_convoca.instante_microsegundo_valido(primera.acta_canonica->>'registrada_en') IS NOT TRUE
 THEN RAISE EXCEPTION 'B95: alta BIC3 cambió algo más que fecha técnica' USING ERRCODE='55000'; END IF;
 SELECT * INTO STRICT repetida FROM vec_bolsa_importacion_convoca.guardar_lote_v1(entrada,'[]'::jsonb);
 IF repetida.reutilizada IS NOT TRUE
 OR repetida.acta_canonica IS DISTINCT FROM primera.acta_canonica
 THEN RAISE EXCEPTION 'B95: replay BIC3 cambió acta durable' USING ERRCODE='55000'; END IF;
END $prueba$;
ROLLBACK;
SELECT CASE WHEN
 (SELECT count(*) FROM vec_bolsa_importacion_convoca.lote)=:'lotes_antes'::bigint
 AND (SELECT count(*) FROM vec_bolsa_importacion_convoca.fila_staging)=:'filas_antes'::bigint
 THEN 'true' ELSE 'false' END AS historia_intacta \gset
\if :historia_intacta
\else
SELECT 1/0 AS b95_fecha_rollback_incompleto;
\endif
SELECT 'B95 fecha BIC3 OK' AS resultado;
