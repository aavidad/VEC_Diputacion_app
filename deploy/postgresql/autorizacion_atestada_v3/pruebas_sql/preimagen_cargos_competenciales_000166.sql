\set ON_ERROR_STOP on
-- Ejecutar antes de AD166, sobre el clon post-AD165 acreditado.
-- Sólo lee estructura: no aplica SQL ni publica perfiles o asignaciones.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $test$
DECLARE f oid;p record;actual text;esperado text;
BEGIN
 f:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF f IS NULL THEN RAISE EXCEPTION 'AD166: preimagen núcleo ausente'; END IF;
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=f;
 esperado:='684229f6e5d3c3a3f4e7b849cc03b55ee275d58cb0b40a5cdeeff39edd9c7beb';
 actual:=encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex');
 IF actual IS DISTINCT FROM esperado THEN
  RAISE EXCEPTION 'AD166: preimagen def actual=% esperado=%',actual,esperado; END IF;
 esperado:='7b04ba29e1ed6943647b445985e951f70130ce323c1acc2d2ea4eafe00f80549';
 actual:=encode(sha256(convert_to(p.prosrc,'UTF8')),'hex');
 IF actual IS DISTINCT FROM esperado THEN
  RAISE EXCEPTION 'AD166: preimagen fuente actual=% esperado=%',actual,esperado; END IF;
 IF p.proowner<>'vec_autorizacion_atestada_v3_propietario'::regrole OR NOT p.prosecdef
  OR p.provolatile<>'v' OR p.proparallel<>'u'
  OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
  OR (SELECT count(*) FROM aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))))<>1
  OR EXISTS(SELECT 1 FROM aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
    WHERE a.grantee<>p.proowner OR a.grantor<>p.proowner
     OR a.privilege_type<>'EXECUTE' OR a.is_grantable) THEN
  RAISE EXCEPTION 'AD166: preimagen metadatos o ACL divergentes'; END IF;
 SELECT regexp_replace(pg_get_constraintdef(c.oid,false),'\s+',' ','g') INTO STRICT actual
 FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
  AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 esperado:='89fb71c30c516e0da102b81154716e36d17a4bcad7fe261076f70f6ed758ff37';
 actual:=encode(sha256(convert_to(actual,'UTF8')),'hex');
 IF actual IS DISTINCT FROM esperado THEN
  RAISE EXCEPTION 'AD166: preimagen CHECK actual=% esperado=%',actual,esperado; END IF;
END $test$;
ROLLBACK;
