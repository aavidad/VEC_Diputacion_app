\set ON_ERROR_STOP on
-- Sólo clon desechable PG18 con AD175 cerrada/instalada y Personal32 ausente.
-- Instala Personal32 UNA vez; no usa DOWN ni modifica recibos históricos.
-- Requiere el canal migrador autorizado; no crea LOGIN ni concede TEMP.
SET ROLE vec_personal_propietario;
SELECT count(*) AS recibos,
 encode(sha256(convert_to(coalesce(jsonb_agg(to_jsonb(r) ORDER BY recibo_ref),'[]'::jsonb)::text,'UTF8')),'hex') AS recibos_sha
FROM vec_personal.recibo_ficha_propia_empleado r
\gset export32_antes_
SELECT encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') AS src_sha,
 encode(sha256(convert_to((to_jsonb(p)-'prosrc')::text,'UTF8')),'hex') AS meta_sha
FROM pg_proc p WHERE p.oid='vec_personal.consultar_ficha_propia_empleado_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
\gset export32_antes_
-- Conservar sólo medidas en GUC de esta sesión para el DO posterior al UP.
-- \gset no imprime las huellas ni las deja en una tabla.
SELECT set_config('vec_export32_test.recibos', :'export32_antes_recibos', false) AS recibos,
 set_config('vec_export32_test.recibos_sha', :'export32_antes_recibos_sha', false) AS recibos_sha,
 set_config('vec_export32_test.src_sha', :'export32_antes_src_sha', false) AS src_sha,
 set_config('vec_export32_test.meta_sha', :'export32_antes_meta_sha', false) AS meta_sha
\gset export32_guc_
RESET ROLE;
\ir ../migraciones/000032_exportacion_servicios_propios.up.sql
SET ROLE vec_personal_propietario;
DO $preservacion$
DECLARE
 esperado_recibos bigint:=current_setting('vec_export32_test.recibos')::bigint;
 esperado_recibos_sha text:=current_setting('vec_export32_test.recibos_sha');
 esperado_src_sha text:=current_setting('vec_export32_test.src_sha');
 esperado_meta_sha text:=current_setting('vec_export32_test.meta_sha');
 actual_recibos bigint; actual_sin_corte bigint; actual_recibos_sha text;
 actual_src_sha text; actual_meta_sha text;
BEGIN
 SELECT count(*),count(*) FILTER(WHERE vigente_en IS NULL AND conocido_en IS NULL),
  encode(sha256(convert_to(coalesce(jsonb_agg(to_jsonb(r)-'vigente_en'-'conocido_en' ORDER BY recibo_ref),'[]'::jsonb)::text,'UTF8')),'hex')
 INTO actual_recibos,actual_sin_corte,actual_recibos_sha
 FROM vec_personal.recibo_ficha_propia_empleado r;
 IF actual_recibos IS DISTINCT FROM esperado_recibos THEN
  RAISE EXCEPTION 'Personal32 prueba: clave=recibos_cantidad esperado=% actual=%',esperado_recibos,actual_recibos;
 END IF;
 IF actual_sin_corte IS DISTINCT FROM esperado_recibos THEN
  RAISE EXCEPTION 'Personal32 prueba: clave=recibos_historicos_sin_corte esperado=% actual=%',esperado_recibos,actual_sin_corte;
 END IF;
 IF actual_recibos_sha IS DISTINCT FROM esperado_recibos_sha THEN
  RAISE EXCEPTION 'Personal32 prueba: clave=recibos_SHA esperado=% actual=%',esperado_recibos_sha,actual_recibos_sha;
 END IF;
 SELECT encode(sha256(convert_to((to_jsonb(p)-'prosrc')::text,'UTF8')),'hex'),
  encode(sha256(convert_to(replace(p.prosrc,
   $nuevo$  (recibo_ref,empleado_ref,material_sha256,decision_ref,auditoria_ref,consumo_huella_sha256,cardinalidad,consultada_en,vigente_en,conocido_en)
 VALUES(recibo,empleado,material_sha,consumo.decision_ref,consumo.auditoria_ref,consumo.consumo_huella_sha256,
   jsonb_array_length(relaciones)+jsonb_array_length(servicios),ahora,fecha,conocido);$nuevo$,
   $anterior$  (recibo_ref,empleado_ref,material_sha256,decision_ref,auditoria_ref,consumo_huella_sha256,cardinalidad,consultada_en)
 VALUES(recibo,empleado,material_sha,consumo.decision_ref,consumo.auditoria_ref,consumo.consumo_huella_sha256,
   jsonb_array_length(relaciones)+jsonb_array_length(servicios),ahora);$anterior$),'UTF8')),'hex')
 INTO actual_meta_sha,actual_src_sha
 FROM pg_proc p WHERE p.oid='vec_personal.consultar_ficha_propia_empleado_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 IF actual_meta_sha IS DISTINCT FROM esperado_meta_sha THEN
  RAISE EXCEPTION 'Personal32 prueba: clave=consulta_metadatos_SHA esperado=% actual=%',esperado_meta_sha,actual_meta_sha;
 END IF;
 IF actual_src_sha IS DISTINCT FROM esperado_src_sha THEN
  RAISE EXCEPTION 'Personal32 prueba: clave=consulta_fuente_invertida_SHA esperado=% actual=%',esperado_src_sha,actual_src_sha;
 END IF;
END $preservacion$;
BEGIN;
DO $inmutabilidad$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM vec_personal.recibo_ficha_propia_empleado) THEN
  RAISE EXCEPTION 'Personal32 prueba: clave=recibo_fixture esperado=al_menos_1 actual=0';
 END IF;
 BEGIN
  UPDATE vec_personal.recibo_ficha_propia_empleado SET vigente_en=CURRENT_DATE,conocido_en=transaction_timestamp()
   WHERE recibo_ref=(SELECT recibo_ref FROM vec_personal.recibo_ficha_propia_empleado ORDER BY recibo_ref LIMIT 1);
  RAISE EXCEPTION 'Personal32 prueba: clave=recibo_inmutable esperado=55000 actual=actualizado';
 EXCEPTION WHEN SQLSTATE '55000' THEN NULL; END;
END $inmutabilidad$;
ROLLBACK;
RESET ROLE;
