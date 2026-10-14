\set ON_ERROR_STOP on
-- Ensayo focal en la copia PG18.4 de H. Exige -v canonical_ref=<ref leída como postgres>.
-- Sólo sustituye la definición del resolutor y concede USAGE dentro de BEGIN/ROLLBACK.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
DO $pre$
DECLARE huella text;
BEGIN
 IF pg_catalog.has_schema_privilege('vec_contratacion_temporal_propietario','vec_contexto_actor_v1','USAGE') THEN
  RAISE EXCEPTION 'CA45 foco: USAGE ya presente antes del ensayo' USING ERRCODE='55000'; END IF;
 huella:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.pg_get_functiondef(
  'vec_contexto_actor_v1.resolver_correspondencia_organizacion_v1(text)'::regprocedure),'UTF8')),'hex');
 IF huella IS DISTINCT FROM 'bb8b457e69cfdcfa073cd8355bb8fe2f94ff619e170b373fb14096153a3a11e2' THEN
  RAISE EXCEPTION 'CA45 foco: preimagen del resolutor divergente: %',huella USING ERRCODE='55000'; END IF;
END $pre$;
SELECT pg_catalog.set_config('vec.ca45_canonical_ref', :'canonical_ref', true);
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
GRANT USAGE ON SCHEMA vec_contexto_actor_v1 TO vec_contratacion_temporal_propietario;
CREATE OR REPLACE FUNCTION vec_contexto_actor_v1.resolver_correspondencia_organizacion_v1(p_organizacion_operativa text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC' AS $f$
DECLARE v record; org record; prueba jsonb; canonica text; ahora timestamptz(6);
BEGIN
 IF p_organizacion_operativa IS NULL OR length(p_organizacion_operativa) NOT BETWEEN 3 AND 512 THEN
  RAISE EXCEPTION 'correspondencia_organizacion_referencia_invalida' USING ERRCODE='42501';
 END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock_shared(pg_catalog.hashtextextended('organizacion_correspondencia:'||p_organizacion_operativa,0));
 IF vec_contexto_actor_v1.organizacion_ref_valida(p_organizacion_operativa) THEN
  canonica:=p_organizacion_operativa; prueba:=NULL;
 ELSE
  SELECT h.* INTO STRICT v FROM vec_contexto_actor_v1.correspondencia_organizacion_actual_v1 a
   JOIN vec_contexto_actor_v1.correspondencia_organizacion_versiones_v1 h USING(organizacion_operativa_ref,version)
   WHERE a.organizacion_operativa_ref=p_organizacion_operativa FOR SHARE OF a,h;
  prueba:=pg_catalog.convert_from(v.documento_canonico,'UTF8')::jsonb||pg_catalog.jsonb_build_object('huella_sha256',v.huella_sha256);
  ahora:=pg_catalog.clock_timestamp();
  IF prueba->>'estado'<>'vigente' OR ahora<(prueba->>'vigente_desde')::timestamptz OR ahora>=(prueba->>'vigente_hasta')::timestamptz THEN
   RAISE EXCEPTION 'correspondencia_organizacion_no_vigente' USING ERRCODE='42501';
  END IF;
  canonica:=v.organizacion_canonica_ref;
 END IF;
 SELECT h.* INTO STRICT org FROM vec_contexto_actor_v1.organizacion_actual a
  JOIN vec_contexto_actor_v1.organizacion_versiones h USING(organizacion_ref,version)
  WHERE a.organizacion_ref=canonica FOR SHARE OF a,h;
 ahora:=pg_catalog.clock_timestamp();
 IF org.estado<>'activo' OR ahora<org.vigente_desde OR ahora>=org.vigente_hasta
  OR vec_contexto_actor_v1.procedencia_valida(org.procedencia_ref,org.procedencia_version,org.procedencia_huella_sha256,org.procedencia_autoridad) IS NOT TRUE THEN
  RAISE EXCEPTION 'correspondencia_organizacion_fuente_no_vigente' USING ERRCODE='42501';
 END IF;
 IF prueba IS NOT NULL THEN
  IF org.version IS DISTINCT FROM v.organizacion_version
   OR org.procedencia_ref IS DISTINCT FROM prueba->>'procedencia_ref'
   OR org.procedencia_version IS DISTINCT FROM (prueba->>'procedencia_version')::numeric
   OR org.procedencia_huella_sha256 IS DISTINCT FROM prueba->>'procedencia_huella_sha256'
   OR org.procedencia_autoridad IS DISTINCT FROM prueba->>'procedencia_autoridad' THEN
   RAISE EXCEPTION 'correspondencia_organizacion_fuente_no_vigente' USING ERRCODE='42501';
  END IF;
 END IF;
 RETURN pg_catalog.jsonb_build_object('organizacion_operativa_ref',p_organizacion_operativa,'organizacion_canonica_ref',canonica,'prueba',prueba);
EXCEPTION WHEN NO_DATA_FOUND OR TOO_MANY_ROWS THEN
 RAISE EXCEPTION 'correspondencia_organizacion_no_unica' USING ERRCODE='42501';
END $f$;
SET LOCAL ROLE postgres;
DO $dentro$
BEGIN
 IF pg_catalog.has_schema_privilege('vec_contratacion_temporal_propietario','vec_contexto_actor_v1','USAGE') IS NOT TRUE THEN
  RAISE EXCEPTION 'CA45 foco: USAGE ausente dentro de la transacción' USING ERRCODE='55000'; END IF;
END $dentro$;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
DO $canonical$
DECLARE canonical_ref text:=pg_catalog.current_setting('vec.ca45_canonical_ref'); r jsonb;
BEGIN
 IF pg_catalog.has_schema_privilege(current_user,'vec_contexto_actor_v1','USAGE') IS NOT TRUE
  OR pg_catalog.has_function_privilege(current_user,'vec_contexto_actor_v1.resolver_correspondencia_organizacion_v1(text)','EXECUTE') IS NOT TRUE THEN
  RAISE EXCEPTION 'CA45 foco: CT no puede invocar resolutor' USING ERRCODE='42501'; END IF;
 r:=vec_contexto_actor_v1.resolver_correspondencia_organizacion_v1(canonical_ref);
 IF pg_catalog.jsonb_typeof(r) IS DISTINCT FROM 'object'
  OR r->>'organizacion_operativa_ref' IS DISTINCT FROM canonical_ref
  OR r->>'organizacion_canonica_ref' IS DISTINCT FROM canonical_ref
  OR r->'prueba' IS DISTINCT FROM 'null'::jsonb THEN
  RAISE EXCEPTION 'CA45 foco: resultado canónico divergente: %',r USING ERRCODE='55000'; END IF;
 RAISE NOTICE 'CA45-CANONICA-OK ref=% prueba=null',canonical_ref;
END $canonical$;
SET LOCAL ROLE postgres;
ROLLBACK;
DO $post$
DECLARE huella text;
BEGIN
 IF pg_catalog.has_schema_privilege('vec_contratacion_temporal_propietario','vec_contexto_actor_v1','USAGE') THEN
  RAISE EXCEPTION 'CA45 foco: USAGE persistió tras ROLLBACK' USING ERRCODE='55000'; END IF;
 huella:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.pg_get_functiondef(
  'vec_contexto_actor_v1.resolver_correspondencia_organizacion_v1(text)'::regprocedure),'UTF8')),'hex');
 IF huella IS DISTINCT FROM 'bb8b457e69cfdcfa073cd8355bb8fe2f94ff619e170b373fb14096153a3a11e2' THEN
  RAISE EXCEPTION 'CA45 foco: definición cambió tras ROLLBACK: %',huella USING ERRCODE='55000'; END IF;
 RAISE NOTICE 'CA45-ROLLBACK-OK huella original y USAGE ausente';
END $post$;
