\set ON_ERROR_STOP on
-- Dos modos separados: inspección propietaria o recorrido nominal CT.
-- El favorable aún necesita un fixture externo sintético (no existe aquí).
-- Ejecutarlo con LOGIN runtime CT y -v aut32_fixture_sql=<fichero>.
-- El fixture entra por la fachada CT REAL; nunca llama directamente a AUT32,
-- nunca hace SET ROLE/SESSION AUTHORIZATION ni sustituye funciones o guardas.
-- Prepara fuentes CA25/Personal28 y permisos mediante los contratos reales.
-- Cada efecto y sus consumo/auditoría V3 deben compartir xid con la evidencia.
-- Aserciones del fixture: alta favorable con huella del canon, repetición local
-- con el mismo consumo sin otra evidencia y recuperación exacta del canon.
-- Esa repetición local NO acredita replay con nueva decisión V3: este exige
-- otra transacción y un nuevo consumo real; queda pendiente con el fixture.
-- El fixture no contiene BEGIN/COMMIT/ROLLBACK; este fichero revierte el ensayo.
\if :{?aut32_fixture_sql}
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL timezone='UTC';
DO $runtime$
BEGIN
 IF current_setting('role') <> 'none' OR current_user <> session_user
  OR NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
  OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
  OR pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
  OR EXISTS(SELECT 1 FROM pg_roles WHERE rolname=session_user AND rolsuper) THEN
  RAISE EXCEPTION 'AUT32 favorable requiere LOGIN runtime CT sin SET ROLE'
   USING ERRCODE='42501'; END IF;
END $runtime$;
\i :aut32_fixture_sql
ROLLBACK;
\else
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
ROLLBACK;
\echo 'AUT32 favorable y repetición local NO EJECUTADOS: falta fixture CT runtime autorizado.'
\echo 'AUT32 replay con nueva decisión V3 NO EJECUTADO: requiere otro consumo y transacción.'
\endif
