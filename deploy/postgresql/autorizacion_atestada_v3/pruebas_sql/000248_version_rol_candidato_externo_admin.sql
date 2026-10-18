\set ON_ERROR_STOP on
-- PostgreSQL 18 desechable tras AD248; sólo comprobaciones, ROLLBACK.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $prueba$
DECLARE v record;funcion record;nucleo record;cinco record;seis record;def text;grupo record;
BEGIN
 SELECT p.prosrc,p.proowner,p.prosecdef,p.provolatile,p.proparallel,p.proconfig,p.proacl
 INTO nucleo FROM pg_proc p WHERE p.oid=to_regprocedure(
  'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF NOT FOUND OR encode(sha256(convert_to(nucleo.prosrc,'UTF8')),'hex') IS DISTINCT FROM
  '21293bfb02cda7760d3395ebbe2066528577e5eb20ff859e5ea2d652d1c64d7b'
 OR nucleo.proowner<>to_regrole('vec_autorizacion_atestada_v3_propietario')
 OR nucleo.prosecdef IS NOT TRUE OR nucleo.provolatile<>'v' OR nucleo.proparallel<>'u'
 OR nucleo.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']::text[]
 OR nucleo.proacl IS DISTINCT FROM ARRAY[
  'vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario'::aclitem]
 THEN RAISE EXCEPTION 'AD248 prueba: núcleo u OID/ACL divergente' USING ERRCODE='55000';END IF;
 FOR v IN SELECT * FROM (VALUES
  ('vec_autorizacion_atestada_v3.login_version_rol_bolsa_externa_valido_v1()',
   '03f59b80c0157fb8034758735740da4d2cd6b463a005fc70dc2b683ed42b25c3'),
  ('vec_autorizacion_atestada_v3.consumir_version_rol_bolsa_externa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
   '83fd76c91d414a29661b410549c204286e75cc41c90f3fde7685ce3a3582da15'),
  ('vec_autorizacion_atestada_v3.registrar_intento_version_rol_bolsa_externa_v1(jsonb)',
   'c0b6b99e4b3cdce6f6acf9e1904f18cc3d39ee2c5d8590ced2f881663d9290f1')
 ) AS x(firma,fuente_sha) LOOP
  SELECT catalogo.prosrc,catalogo.proowner,catalogo.prosecdef,catalogo.proconfig,catalogo.proacl
   INTO funcion FROM pg_proc catalogo WHERE catalogo.oid=to_regprocedure(v.firma);
  IF NOT FOUND OR encode(sha256(convert_to(funcion.prosrc,'UTF8')),'hex') IS DISTINCT FROM v.fuente_sha
  OR funcion.proowner<>to_regrole('vec_autorizacion_atestada_v3_propietario') OR funcion.prosecdef IS NOT TRUE
  OR funcion.proconfig IS NULL OR funcion.proconfig[1] IS DISTINCT FROM 'search_path=pg_catalog, pg_temp'
  OR has_function_privilege('public',v.firma,'EXECUTE') IS TRUE
  OR has_function_privilege('vec_autorizacion_propietario',v.firma,'EXECUTE') IS NOT TRUE
  OR has_function_privilege('vec_admin_version_rol_bolsa_externa_ejecutor',v.firma,'EXECUTE') IS TRUE
  THEN RAISE EXCEPTION 'AD248 prueba: consumidor externo expuesto %',v.firma USING ERRCODE='42501';END IF;
 END LOOP;
 SELECT * INTO grupo FROM pg_roles WHERE rolname='vec_admin_version_rol_bolsa_externa_ejecutor';
 IF NOT FOUND OR grupo.rolcanlogin OR grupo.rolinherit OR grupo.rolsuper
 OR grupo.rolcreatedb OR grupo.rolcreaterole OR grupo.rolreplication OR grupo.rolbypassrls
 THEN RAISE EXCEPTION 'AD248 prueba: grupo externo no aislado' USING ERRCODE='42501';END IF;
 IF (SELECT encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') FROM pg_proc p
  WHERE p.oid=to_regprocedure('vec_autorizacion_atestada_v3.login_version_rol_bolsa_valido_v1()'))
  IS DISTINCT FROM '6ea566931f00890d5aab8aef1c4465813d1a91af97023d963c950c62ca8b7a25'
 OR (SELECT encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') FROM pg_proc p
  WHERE p.oid=to_regprocedure('vec_autorizacion_atestada_v3.login_version_rol_bolsa_b103_valido_v1()'))
  IS DISTINCT FROM '19db91da58ef7ecd57cef024c2d0d4e2923e5deab6c680377014b7915b18c6d2'
 OR (SELECT encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') FROM pg_proc p
  WHERE p.oid=to_regprocedure('vec_autorizacion_atestada_v3.consumir_version_rol_bolsa_b103_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'))
  IS DISTINCT FROM '61c084b180080dd021826871613bbe3bb055e78315e33ed85c2fb43e069cb7dd'
 THEN RAISE EXCEPTION 'AD248 prueba: B103 modificado' USING ERRCODE='55000';END IF;
 SELECT * INTO cinco FROM vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1 WHERE version=5;
 SELECT * INTO seis FROM vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1 WHERE version=6;
 IF NOT FOUND OR cardinality(cinco.audiencias)<>8 OR cardinality(seis.audiencias)<>10
 OR seis.audiencias IS DISTINCT FROM cinco.audiencias||ARRAY[
  'vec_autorizacion.versionar_rol_bolsa_externa.propuesta.v1',
  'vec_autorizacion.versionar_rol_bolsa_externa.cierre.v1']::text[]
 OR seis.segmentos IS DISTINCT FROM cinco.segmentos||ARRAY[
  'perfiles:version-bolsa-externa:propuesta',
  'perfiles:version-bolsa-externa:cierre']::text[]
 OR (SELECT count(*) FROM vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1)<>6
 THEN RAISE EXCEPTION 'AD248 prueba: conjunto 6 divergente' USING ERRCODE='55000';END IF;
 SELECT pg_get_constraintdef(c.oid,false) INTO def FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
 AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 IF NOT FOUND OR strpos(def,'vec_autorizacion.versionar_rol_bolsa_externa.propuesta.v1')=0
 OR strpos(def,'vec_autorizacion.versionar_rol_bolsa_externa.cierre.v1')=0
 OR strpos(def,'vec_autorizacion.versionar_rol_bolsa.propuesta.v1')=0
 OR strpos(def,'vec_autorizacion.versionar_rol_bolsa.cierre.v1')=0
 THEN RAISE EXCEPTION 'AD248 prueba: CHECK de audiencias incompleta' USING ERRCODE='55000';END IF;
END $prueba$;
ROLLBACK;
