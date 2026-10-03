\set ON_ERROR_STOP on
-- Prueba focal posterior a AD165, AD166, AD167 y AUT32. No publica datos.
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL timezone='UTC';
DO $test$
DECLARE f regprocedure; x record; n bigint;
BEGIN
 IF to_regprocedure('vec_autorizacion.acreditar_competencia_nominal_firmante_ct_v1(bytea,jsonb,jsonb)') IS NULL
  OR to_regprocedure('vec_autorizacion.recuperar_evidencia_competencia_firmante_ct_v1(text,text,bytea,jsonb,jsonb)') IS NULL
  OR to_regclass('vec_autorizacion.evidencia_competencia_firmante_ct_v1') IS NULL THEN
  RAISE EXCEPTION 'AUT32 ausente' USING ERRCODE='55000'; END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_class c WHERE c.oid='vec_autorizacion.evidencia_competencia_firmante_ct_v1'::regclass
  AND c.relrowsecurity AND c.relforcerowsecurity)
  OR NOT EXISTS(SELECT 1 FROM pg_attribute a WHERE a.attrelid=
   'vec_autorizacion.evidencia_competencia_firmante_ct_v1'::regclass
   AND a.attname='organizacion_destino' AND a.attnotnull)
  OR EXISTS(SELECT 1 FROM pg_class c CROSS JOIN LATERAL
   aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) a
   WHERE c.oid='vec_autorizacion.evidencia_competencia_firmante_ct_v1'::regclass
    AND a.grantee<>'vec_autorizacion_propietario'::regrole) THEN
  RAISE EXCEPTION 'AUT32 RLS inactivo' USING ERRCODE='55000'; END IF;
 FOR f IN SELECT unnest(ARRAY[
  'vec_autorizacion.acreditar_competencia_nominal_firmante_ct_v1(bytea,jsonb,jsonb)'::regprocedure,
  'vec_autorizacion.recuperar_evidencia_competencia_firmante_ct_v1(text,text,bytea,jsonb,jsonb)'::regprocedure]) LOOP
  IF EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL
   aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f
   AND a.grantee NOT IN ('vec_autorizacion_propietario'::regrole,
     'vec_contratacion_temporal_propietario'::regrole))
   OR NOT has_function_privilege('vec_contratacion_temporal_propietario',f,'EXECUTE') THEN
   RAISE EXCEPTION 'AUT32 ACL de función divergente: %',f USING ERRCODE='55000'; END IF;
 END LOOP;
 SELECT count(*) INTO n FROM vec_autorizacion.evidencia_competencia_firmante_ct_v1;
 -- La validación no debe convertir ausencia de fuentes o consumo en evidencia.
 BEGIN
  PERFORM vec_autorizacion.acreditar_competencia_nominal_firmante_ct_v1(
   convert_to('{}','UTF8'),'{}'::jsonb,'{}'::jsonb);
  RAISE EXCEPTION 'AUT32 aceptó entrada vacía' USING ERRCODE='55000';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL;
 END;
 IF (SELECT count(*) FROM vec_autorizacion.evidencia_competencia_firmante_ct_v1) <> n THEN
  RAISE EXCEPTION 'AUT32 escribió tras denegar' USING ERRCODE='55000'; END IF;
END $test$;
-- Para ejecutar también el favorable, pasar -v aut32_fixture_sql=<fichero>.
-- El fichero sólo usa datos sintéticos y prepara, dentro de ESTA transacción,
-- las fuentes CA25/Personal28, rol/asignación vigentes y consumo real V3/AD167.
-- No debe abrir/cerrar transacciones, sustituir funciones ni desactivar guardas.
-- Deja variables psql aut32_contexto_hex (canon original hexadecimal),
-- aut32_relacion_ct y aut32_consumo_v3 (JSON). El efecto aún no tiene evidencia;
-- las filas de consumo/auditoría deben tener xmin de esta transacción.
\if :{?aut32_fixture_sql}
\i :aut32_fixture_sql
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL timezone='UTC';
SELECT set_config('vec_test.aut32_contexto_hex', :'aut32_contexto_hex', true) IS NOT NULL,
 set_config('vec_test.aut32_relacion_ct', :'aut32_relacion_ct', true) IS NOT NULL,
 set_config('vec_test.aut32_consumo_v3', :'aut32_consumo_v3', true) IS NOT NULL;
DO $favorable$
DECLARE
 canon bytea := decode(current_setting('vec_test.aut32_contexto_hex'),'hex');
 relacion jsonb := current_setting('vec_test.aut32_relacion_ct')::jsonb;
 consumo jsonb := current_setting('vec_test.aut32_consumo_v3')::jsonb;
 primera jsonb; replay jsonb; original bytea; n bigint; fila jsonb;
BEGIN
 SELECT count(*) INTO n FROM vec_autorizacion.evidencia_competencia_firmante_ct_v1;
 primera := vec_autorizacion.acreditar_competencia_nominal_firmante_ct_v1(canon,relacion,consumo);
 IF primera->'recuperada' IS DISTINCT FROM 'false'::jsonb
  OR primera->>'huella_sha256' IS DISTINCT FROM encode(sha256(canon),'hex')
  OR (SELECT count(*) FROM vec_autorizacion.evidencia_competencia_firmante_ct_v1) <> n+1 THEN
  RAISE EXCEPTION 'AUT32 alta favorable divergente' USING ERRCODE='55000'; END IF;
 SELECT to_jsonb(e) INTO STRICT fila
 FROM vec_autorizacion.evidencia_competencia_firmante_ct_v1 e
 WHERE e.evidencia_ref=primera->>'evidencia_ref';
 replay := vec_autorizacion.acreditar_competencia_nominal_firmante_ct_v1(canon,relacion,consumo);
 IF replay->'recuperada' IS DISTINCT FROM 'true'::jsonb
  OR (replay - 'recuperada') IS DISTINCT FROM (primera - 'recuperada')
  OR (SELECT count(*) FROM vec_autorizacion.evidencia_competencia_firmante_ct_v1) <> n+1
  OR (SELECT to_jsonb(e) FROM vec_autorizacion.evidencia_competencia_firmante_ct_v1 e
      WHERE e.evidencia_ref=primera->>'evidencia_ref') IS DISTINCT FROM fila THEN
  RAISE EXCEPTION 'AUT32 replay cambió evidencia' USING ERRCODE='55000'; END IF;
 original := vec_autorizacion.recuperar_evidencia_competencia_firmante_ct_v1(
  primera->>'evidencia_ref',primera->>'huella_sha256',canon,relacion,consumo);
 IF original IS DISTINCT FROM canon THEN
  RAISE EXCEPTION 'AUT32 recuperación cambió canon' USING ERRCODE='55000'; END IF;
END $favorable$;
\else
\echo 'AUT32 favorable/replay NO EJECUTADOS: falta aut32_fixture_sql con fuentes y consumo V3 en esta transacción.'
\endif
ROLLBACK;
