\set ON_ERROR_STOP on
-- No ejecutar sobre historia conservada. Solo retirada controlada sin consumo B11.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_autorizacion_atestada_v3:migracion:000041', 0));

DO $preservar_historia$
DECLARE v_def text; v_nueva text;
BEGIN
 IF pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_consumo_participaciones_propias_b11_v3(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.revalidar_consumo_participaciones_propias_b11_v3(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR pg_catalog.to_regprocedure('vec_bolsa_llamamientos.consultar_participaciones_propias_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.clave_capacidad_version WHERE audiencia_consumo='vec.bolsa.mi-bolsa.v1')
    OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.atestacion_decision_v3
                WHERE pg_catalog.convert_from(capacidad_canonica,'UTF8')::jsonb->>'audiencia_consumo'='vec.bolsa.mi-bolsa.v1') THEN
   RAISE EXCEPTION 'AD3-000041: retirada bloqueada por fachada, clave o historia B11' USING ERRCODE='55000';
 END IF;
 LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
 SELECT pg_catalog.regexp_replace(pg_catalog.pg_get_constraintdef(c.oid,true),'\s+',' ','g') INTO STRICT v_def
 FROM pg_catalog.pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
   AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated AND c.conkey=ARRAY[8]::smallint[];
 IF v_def ~ '^CHECK \(audiencia_consumo IN \(.+\)\)$'
    AND pg_catalog.strpos(v_def,', ''vec.bolsa.mi-bolsa.v1''))')<>0 THEN
   v_nueva := pg_catalog.replace(v_def,', ''vec.bolsa.mi-bolsa.v1''))','))');
 ELSIF v_def ~ '^CHECK \(audiencia_consumo = ANY \(ARRAY\[.+\]\)\)$'
    AND pg_catalog.strpos(v_def,', ''vec.bolsa.mi-bolsa.v1''::text]))')<>0 THEN
   v_nueva := pg_catalog.replace(v_def,', ''vec.bolsa.mi-bolsa.v1''::text]))',']))');
 ELSE
   RAISE EXCEPTION 'AD3-000041: CHECK de audiencia no reversible' USING ERRCODE='55000';
 END IF;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check';
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '
      || v_nueva;
END $preservar_historia$;

DROP FUNCTION vec_autorizacion_atestada_v3.registrar_consumo_participaciones_propias_b11_v3(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_autorizacion_atestada_v3.revalidar_consumo_participaciones_propias_b11_v3(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_autorizacion_atestada_v3.consumir_b11_participaciones_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_autorizacion_atestada_v3.revalidar_b11_participaciones_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
COMMIT;
