\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal.dependencias.incorporacion_ejercicio.v2',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:000074:transicion:v2',0));
SET LOCAL ROLE vec_contratacion_temporal_propietario;
DO $dependencias$
DECLARE ids oid[]; nombre text; firma text; f oid;
 nombres text[]:=ARRAY['aplicar_incorporacion_seguimiento_v2','normalizar_salida_transicion_incorporacion_v2'];
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_namespace WHERE nspname='vec_contratacion_temporal' AND nspowner=current_user::regrole)
 OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=current_user AND NOT rolcanlogin AND NOT rolsuper
     AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication AND NOT rolbypassrls) THEN
  RAISE EXCEPTION 'CT74: propietario incompatible' USING ERRCODE='55000';
 END IF;
 SELECT array_agg(p.oid) INTO ids FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
  WHERE n.nspname='vec_contratacion_temporal' AND p.proname=ANY(nombres);
 IF cardinality(ids) IS DISTINCT FROM 2 THEN
  RAISE EXCEPTION 'CT74: inventario o sobrecargas incompatible' USING ERRCODE='55000';
 END IF;
 FOREACH firma IN ARRAY ARRAY[
   'vec_contratacion_temporal.aplicar_incorporacion_seguimiento_v2(jsonb,jsonb,jsonb,text,text,text)',
   'vec_contratacion_temporal.normalizar_salida_transicion_incorporacion_v2(jsonb)'] LOOP
  f:=to_regprocedure(firma);
  IF f IS NULL OR NOT EXISTS (SELECT 1 FROM pg_proc WHERE oid=f AND proowner=current_user::regrole
     AND prorettype='jsonb'::regtype AND prokind='f' AND NOT prosecdef AND NOT proleakproof
     AND provolatile='i' AND proparallel='s' AND proconfig=ARRAY['search_path=pg_catalog']
     AND obj_description(oid,'pg_proc')='CT74:transicion-pura-incorporacion-v2;sin-autoridad-ni-commit')
  OR EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(COALESCE(p.proacl,acldefault('f',p.proowner))) a
     WHERE p.oid=f AND (a.grantee<>p.proowner OR a.grantor<>p.proowner)) THEN
   RAISE EXCEPTION 'CT74: firma/autoridad alterada' USING ERRCODE='55000';
  END IF;
 END LOOP;
 -- Rechaza consumidores de cualquier esquema/sobrecarga, incluso PL/pgSQL
 -- por nombre sin dependencia registrada. No consulta ni elimina historia.
 IF EXISTS (SELECT 1 FROM pg_depend d WHERE d.refclassid='pg_proc'::regclass AND d.refobjid=ANY(ids)
     AND NOT (d.classid='pg_proc'::regclass AND d.objid=ANY(ids)))
 OR EXISTS (SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE p.oid<>ALL(ids)
     AND ((n.nspname='vec_contratacion_temporal' AND p.proname='registrar_incorporacion_ejercicio_v2')
       OR EXISTS (SELECT 1 FROM unnest(nombres) x WHERE strpos(p.prosrc,x)>0))) THEN
  RAISE EXCEPTION 'CT74: dependencia conservada' USING ERRCODE='55000';
 END IF;
END $dependencias$;
DROP FUNCTION vec_contratacion_temporal.aplicar_incorporacion_seguimiento_v2(jsonb,jsonb,jsonb,text,text,text) RESTRICT;
DROP FUNCTION vec_contratacion_temporal.normalizar_salida_transicion_incorporacion_v2(jsonb) RESTRICT;
COMMIT;
