\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal.dependencias.incorporacion_ejercicio.v2',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:000073:seguimiento:v1',0));
SET LOCAL ROLE vec_contratacion_temporal_propietario;
DO $guardas$
DECLARE firmas text[]:=ARRAY[
 'estado_seguimiento_canonico_v1(jsonb,jsonb)',
 'seguimiento73_rehidratar(jsonb,jsonb)', 'seguimiento73_periodos(jsonb,jsonb,text)',
 'seguimiento73_definicion(jsonb)', 'seguimiento73_ordenar(jsonb,text)',
 'seguimiento73_nodo(jsonb,text)', 'seguimiento73_elementos(jsonb,integer)',
 'seguimiento73_array(jsonb,integer,integer,boolean)', 'seguimiento73_escalar(jsonb,text)',
 'seguimiento73_micro(jsonb,boolean)', 'seguimiento73_forma(jsonb,text[],text[])', 'seguimiento73_exigir(boolean)'];
 ids oid[]:='{}'; firma text; id oid; f record;
BEGIN
 FOREACH firma IN ARRAY firmas LOOP
  id:=to_regprocedure('vec_contratacion_temporal.'||firma);
  IF id IS NULL THEN RAISE EXCEPTION USING ERRCODE='55000',MESSAGE='codec seguimiento incompleto'; END IF;
  ids:=array_append(ids,id);
 END LOOP;
 IF (SELECT count(*) FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
  WHERE n.nspname='vec_contratacion_temporal' AND (p.proname LIKE 'seguimiento73\_%' ESCAPE '\' OR p.proname='estado_seguimiento_canonico_v1'))<>12
  OR EXISTS (SELECT 1 FROM pg_proc WHERE oid=ANY(ids) AND (
   proowner<>'vec_contratacion_temporal_propietario'::regrole OR prokind<>'f' OR prosecdef OR provolatile<>'i'
   OR proparallel<>'s' OR proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog']
   OR prorettype<>CASE
    WHEN proname IN ('seguimiento73_exigir','seguimiento73_forma') THEN 'void'::regtype
    WHEN proname='seguimiento73_micro' THEN 'bigint'::regtype
    WHEN proname='seguimiento73_elementos' THEN 'integer'::regtype
    WHEN proname IN ('estado_seguimiento_canonico_v1','seguimiento73_escalar','seguimiento73_nodo') THEN 'bytea'::regtype
    ELSE 'jsonb'::regtype END))
  OR EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=ANY(ids) AND a.grantee<>p.proowner)
 THEN RAISE EXCEPTION USING ERRCODE='55000',MESSAGE='codec seguimiento alterado'; END IF;
 -- PL/pgSQL puede no declarar dependencias en pg_depend: cotejo por nombre.
 IF EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid<>ALL(ids) AND
  (strpos(p.prosrc,'estado_seguimiento_canonico_v1')>0 OR strpos(p.prosrc,'seguimiento73_')>0))
  OR EXISTS (SELECT 1 FROM pg_depend d WHERE d.refclassid='pg_proc'::regclass AND d.refobjid=ANY(ids)
   AND NOT (d.classid='pg_proc'::regclass AND d.objid=ANY(ids)))
 THEN RAISE EXCEPTION USING ERRCODE='55000',MESSAGE='codec seguimiento tiene consumidores'; END IF;
 -- No CASCADE ni retirada de historia. CT72 es independiente; sólo dependencias
 -- reales sobre este codec bloquean aquí. Su propia retirada conserva sus filas.
 FOREACH firma IN ARRAY firmas LOOP
  EXECUTE 'DROP FUNCTION vec_contratacion_temporal.'||firma||' RESTRICT';
 END LOOP;
END $guardas$;
COMMIT;
