\set ON_ERROR_STOP on
-- Ejecutar en PostgreSQL 18 desechable después de 000007, nunca en la base
-- principal. No crea datos personales ni ejecuta DOWN.
BEGIN;
SET LOCAL search_path=pg_catalog;
DO $prueba$
DECLARE nombre text; p record;
BEGIN
 IF to_regclass('vec_dietas.cola_circuito_comision') IS NULL
    OR (SELECT count(*) FROM vec_dietas.cola_circuito_comision)<>0
 THEN RAISE EXCEPTION '000007: cola inicial incompatible'; END IF;
 IF NOT EXISTS (SELECT 1 FROM pg_class c WHERE c.oid='vec_dietas.cola_circuito_comision'::regclass
                 AND c.relrowsecurity AND c.relforcerowsecurity) THEN
  RAISE EXCEPTION '000007: RLS de cola ausente';
 END IF;
 IF (SELECT count(*) FROM pg_attribute a
      WHERE a.attrelid='vec_dietas.recibo_operacion_comision'::regclass
        AND a.attname IN ('regla_ref','regla_huella_sha256')
        AND a.attnotnull AND NOT a.attisdropped AND a.atttypid='text'::regtype)<>2 THEN
  RAISE EXCEPTION '000007: recibo sin regla y huella obligatorias';
 END IF;
 IF has_table_privilege('vec_dietas_ejecutor','vec_dietas.cola_circuito_comision','SELECT')
    OR has_table_privilege('vec_dietas_ejecutor','vec_dietas.cola_circuito_comision','INSERT')
    OR has_table_privilege('vec_dietas_ejecutor','vec_dietas.cola_circuito_comision','UPDATE')
    OR has_table_privilege('vec_dietas_ejecutor','vec_dietas.cola_circuito_comision','DELETE') THEN
  RAISE EXCEPTION '000007: ejecutor con acceso directo a cola';
 END IF;
 IF NOT EXISTS (SELECT 1 FROM pg_trigger t
  WHERE t.tgrelid='vec_dietas.comision_revision'::regclass
    AND t.tgname='abrir_cola_revision' AND t.tgenabled='O' AND NOT t.tgisinternal) THEN
  RAISE EXCEPTION '000007: trigger de envío ausente';
 END IF;
 FOREACH nombre IN ARRAY ARRAY[
  'vec_dietas.preleer_circuito_comision_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'vec_dietas.decidir_comision_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'vec_dietas.listar_bandeja_comisiones_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
 ] LOOP
  SELECT prosecdef,proowner,proconfig INTO p FROM pg_proc WHERE oid=nombre::regprocedure;
  IF NOT FOUND OR NOT p.prosecdef OR p.proowner<>'vec_dietas_propietario'::regrole
     OR NOT has_function_privilege('vec_dietas_ejecutor',nombre,'EXECUTE')
     OR has_function_privilege('vec_dietas_registrador_frontera',nombre,'EXECUTE') THEN
   RAISE EXCEPTION '000007: ACL o propiedad incorrecta de %',nombre;
  END IF;
 END LOOP;
END $prueba$;
ROLLBACK;
