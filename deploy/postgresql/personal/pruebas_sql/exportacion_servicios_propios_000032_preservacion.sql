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
RESET ROLE;
\ir ../migraciones/000032_exportacion_servicios_propios.up.sql
SET ROLE vec_personal_propietario;
-- Una diferencia provoca división por cero y ON_ERROR_STOP detiene la prueba.
SELECT 1 / ((count(*)=:'export32_antes_recibos'::bigint
 AND count(*) FILTER(WHERE vigente_en IS NULL AND conocido_en IS NULL)=count(*)
 AND encode(sha256(convert_to(coalesce(jsonb_agg(to_jsonb(r)-'vigente_en'-'conocido_en' ORDER BY recibo_ref),'[]'::jsonb)::text,'UTF8')),'hex')=:'export32_antes_recibos_sha')::integer)
FROM vec_personal.recibo_ficha_propia_empleado r;
SELECT 1 / ((encode(sha256(convert_to((to_jsonb(p)-'prosrc')::text,'UTF8')),'hex')=:'export32_antes_meta_sha')::integer)
FROM pg_proc p WHERE p.oid='vec_personal.consultar_ficha_propia_empleado_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
-- La reversión textual del único INSERT vuelve al prosrc original completo.
SELECT 1 / ((encode(sha256(convert_to(replace(p.prosrc,
 $nuevo$  (recibo_ref,empleado_ref,material_sha256,decision_ref,auditoria_ref,consumo_huella_sha256,cardinalidad,consultada_en,vigente_en,conocido_en)
 VALUES(recibo,empleado,material_sha,consumo.decision_ref,consumo.auditoria_ref,consumo.consumo_huella_sha256,
   jsonb_array_length(relaciones)+jsonb_array_length(servicios),ahora,fecha,conocido);$nuevo$,
 $anterior$  (recibo_ref,empleado_ref,material_sha256,decision_ref,auditoria_ref,consumo_huella_sha256,cardinalidad,consultada_en)
 VALUES(recibo,empleado,material_sha,consumo.decision_ref,consumo.auditoria_ref,consumo.consumo_huella_sha256,
   jsonb_array_length(relaciones)+jsonb_array_length(servicios),ahora);$anterior$),'UTF8')),'hex')=:'export32_antes_src_sha')::integer)
FROM pg_proc p WHERE p.oid='vec_personal.consultar_ficha_propia_empleado_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
BEGIN;
DO $inmutabilidad$
BEGIN
 IF EXISTS(SELECT 1 FROM vec_personal.recibo_ficha_propia_empleado) THEN
  BEGIN
   UPDATE vec_personal.recibo_ficha_propia_empleado SET vigente_en=CURRENT_DATE,conocido_en=transaction_timestamp()
    WHERE recibo_ref=(SELECT recibo_ref FROM vec_personal.recibo_ficha_propia_empleado ORDER BY recibo_ref LIMIT 1);
   RAISE EXCEPTION 'prueba: recibo histórico mutable';
  EXCEPTION WHEN SQLSTATE '55000' THEN NULL; END;
 END IF;
END $inmutabilidad$;
ROLLBACK;
RESET ROLE;
