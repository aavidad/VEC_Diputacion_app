\set ON_ERROR_STOP on
-- Ejecutar tras CT197, B81 y B90 en PostgreSQL 18 desechable.
-- Toda la evidencia sintética se deshace. Comprueba que B13 pendiente de B45
-- excluye un turno y conserva señal sin fecha en B82 y en la consulta por lote.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL statement_timeout='20s';
CREATE ROLE vec_b91_ejecutor_test LOGIN;
GRANT vec_bolsa_llamamientos_ejecutor TO vec_b91_ejecutor_test;
SET ROLE vec_bolsa_llamamientos_propietario;
WITH origen AS (
 SELECT c.bolsa_ref,vc.participacion_ref,vc.candidato_ref
   FROM vec_bolsa_llamamientos.constitucion c
   JOIN vec_bolsa_llamamientos.vinculo_candidato vc ON vc.acta_ref=c.acta_ref
  WHERE EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(c.bolsa_ref,clock_timestamp()) o
                 WHERE o.participacion_ref=vc.participacion_ref AND o.orden_vigente IS NOT NULL)
  ORDER BY c.confirmada_en DESC,vc.participacion_ref LIMIT 1
)
SELECT quote_literal(bolsa_ref) AS bolsa_sql,
 quote_literal(participacion_ref) AS participacion_sql,
 quote_literal(candidato_ref) AS candidato_sql
FROM origen \gset

WITH dato AS (
 SELECT 'origen:b91:prueba:20261008'::text AS origen_ref,
        'llamamiento:b91:prueba:20261008'::text AS llamamiento_ref,
        'evento:ct:contrato-bolsa:'||encode(sha256(convert_to(
         'cese'||chr(31)||'origen:b91:prueba:20261008','UTF8')),'hex') AS evento_ref
), evento AS (
 SELECT d.*,jsonb_build_object('evento_ref',d.evento_ref,'tipo','cese',
  'origen_ref',d.origen_ref,'organizacion_ref','organizacion:desarrollo:dipgra',
  'expediente_ref','expediente:b91:prueba','llamamiento_ref',d.llamamiento_ref) AS cuerpo
 FROM dato d
)
INSERT INTO vec_bolsa_llamamientos.contrato_participacion(
 evento_ref,huella_sha256,evento,origen_ref,origen_creada_en,origen_posicion,
 tipo,organizacion_ref,expediente_ref,llamamiento_ref,participacion_ref,bolsa_ref,
 ocurrido_en,recibido_en)
SELECT evento_ref,encode(sha256(convert_to(cuerpo::text,'UTF8')),'hex'),cuerpo,
 origen_ref,clock_timestamp(),99999991,'cese','organizacion:desarrollo:dipgra',
 'expediente:b91:prueba',llamamiento_ref,:participacion_sql,:bolsa_sql,
 clock_timestamp(),clock_timestamp()
FROM evento;
RESET ROLE;
SELECT set_config('vec.b91_participacion',:participacion_sql,true);
SELECT set_config('vec.b91_bolsa',:bolsa_sql,true);
SET SESSION AUTHORIZATION vec_b91_ejecutor_test;
DO $prueba$
DECLARE v_lote record; v_orden record; v_resumen record; v_filas bigint;
 v_antes boolean; v_corte_anterior timestamptz:=clock_timestamp()-interval '1 day';
BEGIN
 SELECT coalesce(bool_or(cese_pendiente),false) INTO v_antes
  FROM vec_bolsa_llamamientos.consultar_estado_cese_bolsa_lote_v2(
   ARRAY[current_setting('vec.b91_participacion')]::text[],v_corte_anterior);
 IF v_antes THEN
  RAISE EXCEPTION 'B90: clave=pendiente_antes_del_cese esperado=false actual=true';
 END IF;
 SELECT count(*) INTO v_filas FROM vec_bolsa_llamamientos.consultar_estado_cese_bolsa_lote_v2(
  ARRAY[current_setting('vec.b91_participacion'),current_setting('vec.b91_participacion')]::text[],clock_timestamp());
 IF v_filas<>1 THEN
  RAISE EXCEPTION 'B90: clave=lote_sin_duplicado esperado=1 actual=%',v_filas;
 END IF;
 SELECT * INTO STRICT v_lote FROM vec_bolsa_llamamientos.consultar_estado_cese_bolsa_lote_v2(
  ARRAY[current_setting('vec.b91_participacion')]::text[],clock_timestamp());
 IF v_lote.cese_pendiente IS NOT TRUE OR v_lote.pendiente_desde IS NULL
    OR v_lote.pendiente_desde<=v_corte_anterior THEN
  RAISE EXCEPTION 'B90: clave=cese_pendiente_y_desde esperado=true_y_fecha_real actual=%',row_to_json(v_lote);
 END IF;
 SELECT orden_vigente,situacion,razon INTO STRICT v_orden
   FROM vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(current_setting('vec.b91_bolsa'),clock_timestamp())
  WHERE participacion_ref=current_setting('vec.b91_participacion');
 IF v_orden.orden_vigente IS NOT NULL OR v_orden.situacion<>'no_disponible'
    OR v_orden.razon<>'cese_pendiente' THEN
  RAISE EXCEPTION 'B90: clave=orden_pendiente esperado=NULL/no_disponible/cese_pendiente actual=%',row_to_json(v_orden);
 END IF;
 SELECT cese_fecha_efecto,cese_disponible_desde,cese_en_restriccion,cese_trabajo_cesado,desde
   INTO STRICT v_resumen FROM vec_bolsa_llamamientos.leer_resumen_situaciones_bolsas_v1(clock_timestamp())
  WHERE participacion_ref=current_setting('vec.b91_participacion');
 IF v_resumen.cese_fecha_efecto IS NOT NULL OR v_resumen.cese_disponible_desde IS NOT NULL
    OR v_resumen.cese_en_restriccion IS NOT TRUE OR v_resumen.cese_trabajo_cesado IS NOT FALSE
    OR v_resumen.desde IS DISTINCT FROM v_lote.pendiente_desde THEN
  RAISE EXCEPTION 'B90: clave=resumen_pendiente esperado=NULL_NULL_true_false actual=%',row_to_json(v_resumen);
 END IF;
END $prueba$;
RESET SESSION AUTHORIZATION;

-- Una exclusión ya decidida conserva su propio estado y su fecha: el cese
-- pendiente no convierte «excluido desde X» en «excluido desde B13».
SAVEPOINT estado_excluido;
SET ROLE vec_bolsa_llamamientos_propietario;
DO $alta_excluida$
DECLARE instante timestamptz:=date_trunc('microseconds',clock_timestamp());
BEGIN
 INSERT INTO vec_bolsa_llamamientos.situacion_participacion(
  participacion_ref,situacion,desde,motivo,actor,registrada_en,clave_idempotencia,recibo_ref)
 VALUES(current_setting('vec.b91_participacion'),'excluido',instante,
  'Exclusión sintética B90','actor:prueba-b90',instante,
  'b90:exclusion:prueba','recibo:b90:exclusion:prueba');
 PERFORM set_config('vec.b90.excluido_desde',instante::text,true);
END $alta_excluida$;
RESET ROLE;
SET SESSION AUTHORIZATION vec_b91_ejecutor_test;
DO $fecha_excluida$
DECLARE v record; orden record;
BEGIN
 SELECT situacion,desde INTO STRICT v
  FROM vec_bolsa_llamamientos.leer_resumen_situaciones_bolsas_v1(clock_timestamp())
 WHERE participacion_ref=current_setting('vec.b91_participacion');
 IF v.situacion<>'excluido' OR v.desde IS DISTINCT FROM current_setting('vec.b90.excluido_desde')::timestamptz THEN
  RAISE EXCEPTION 'B90: clave=exclusion_fecha esperado=excluido/% actual=%',
   current_setting('vec.b90.excluido_desde'),row_to_json(v);
 END IF;
 SELECT situacion,razon,orden_vigente INTO STRICT orden
  FROM vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(current_setting('vec.b91_bolsa'),clock_timestamp())
 WHERE participacion_ref=current_setting('vec.b91_participacion');
 IF orden.situacion<>'excluido' OR orden.razon='cese_pendiente' OR orden.orden_vigente IS NOT NULL THEN
  RAISE EXCEPTION 'B90: clave=exclusion_orden esperado=excluido/sin_turno actual=%',row_to_json(orden);
 END IF;
END $fecha_excluida$;
RESET SESSION AUTHORIZATION;
ROLLBACK TO SAVEPOINT estado_excluido;
ROLLBACK;
