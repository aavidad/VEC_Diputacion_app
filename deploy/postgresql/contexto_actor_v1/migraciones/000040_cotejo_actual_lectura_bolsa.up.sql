\set ON_ERROR_STOP on
-- CA40: cotejo actual del contexto V2 corporativo para la lectura auditada B103.
-- Reutiliza la relación CA9; solo el propietario AD3 puede recibir un booleano.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='45s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contexto_actor_v1:migracion:000040',0));
DO $pre$
DECLARE h text;
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 OR to_regprocedure('vec_contexto_actor_v1.cotejar_contexto_lectura_bolsa_actual_v1(bytea)') IS NOT NULL
 OR to_regrole('vec_autorizacion_atestada_v3_propietario') IS NULL
 THEN RAISE EXCEPTION 'CA40: preimagen estructural incompatible' USING ERRCODE='55000'; END IF;
 SELECT encode(sha256(convert_to(pg_get_functiondef(
  'vec_contexto_actor_v1.revalidar_vinculo_corporativo_rrhh_v1(text,text,text,text,numeric)'::regprocedure),'UTF8')),'hex') INTO h;
 IF h IS DISTINCT FROM '18a5f91afeb229c337e5d456df6d27462c3310f9d96c1b7a7a572530281557d0'
 THEN RAISE EXCEPTION 'CA40: CA9 actual=% esperado=18a5f91afeb229c337e5d456df6d27462c3310f9d96c1b7a7a572530281557d0',h USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
CREATE FUNCTION vec_contexto_actor_v1.cotejar_contexto_lectura_bolsa_actual_v1(
 p_contexto_canonico bytea)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE j jsonb;r record;ahora timestamptz(6);
BEGIN
 IF current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off'
 OR p_contexto_canonico IS NULL OR octet_length(p_contexto_canonico) NOT BETWEEN 1 AND 65536
 THEN RETURN false; END IF;
 j:=convert_from(p_contexto_canonico,'UTF8')::jsonb;
 IF jsonb_typeof(j) IS DISTINCT FROM 'object'
 OR j->>'esquema' IS DISTINCT FROM 'vec.contexto-actor.vinculado.v2'
 OR j->>'estado' IS DISTINCT FROM 'activo'
 OR vec_contexto_actor_v1.referencia_valida(j->>'cuenta_ref','cta_') IS NOT TRUE
 OR vec_contexto_actor_v1.referencia_valida(j->>'persona_ref','per_') IS NOT TRUE
 OR vec_contexto_actor_v1.referencia_valida(j->>'perfil_activo_ref','prf_') IS NOT TRUE
 OR vec_contexto_actor_v1.referencia_valida(j->>'contexto_actor_ref','vca_') IS NOT TRUE
 OR (j->>'cuenta_version') !~ '^[1-9][0-9]{0,19}$'
 OR (j->>'persona_version') !~ '^[1-9][0-9]{0,19}$'
 OR (j->>'perfil_version') !~ '^[1-9][0-9]{0,19}$'
 OR (j->>'contexto_version') !~ '^[1-9][0-9]{0,19}$'
 THEN RETURN false; END IF;
 -- CA9 ya publica la relación corporativa. Aquí se fijan sus punteros actuales
 -- hasta el COMMIT que también registra la lectura en AD237 y proyecta B103.
 SELECT v.*,c.version AS cuenta_version,c.estado AS cuenta_estado,
  c.vigente_desde AS cuenta_desde,c.vigente_hasta AS cuenta_hasta,
  c.procedencia_autoridad AS cuenta_autoridad,
  x.version AS persona_version,x.estado AS persona_estado,
  x.vigente_desde AS persona_desde,x.vigente_hasta AS persona_hasta,
  x.procedencia_autoridad AS persona_autoridad,
  p.version AS perfil_version,p.estado AS perfil_estado,p.persona_ref AS perfil_persona_ref,
  p.vigente_desde AS perfil_desde,p.vigente_hasta AS perfil_hasta,
  p.procedencia_autoridad AS perfil_autoridad,
  cv.estado AS corporativo_estado,cv.vigente_desde AS corporativo_desde,
  cv.vigente_hasta AS corporativo_hasta,cv.procedencia_autoridad AS corporativo_autoridad,
  cv.cuenta_ref AS corporativo_cuenta_ref,cv.perfil_ref AS corporativo_perfil_ref,
  cv.persona_ref AS corporativo_persona_ref,
  cv.vinculo_contexto_ref AS corporativo_vinculo_ref,
  cv.vinculo_contexto_version AS corporativo_vinculo_version,
  cv.cuenta_version AS corporativo_cuenta_version,
  cv.perfil_version AS corporativo_perfil_version,
  cv.persona_version AS corporativo_persona_version,
  cv.organizacion_version AS corporativo_organizacion_version,
  cv.organizacion_procedencia_ref AS corporativo_organizacion_procedencia_ref,
  cv.organizacion_procedencia_version AS corporativo_organizacion_procedencia_version,
  cv.organizacion_procedencia_huella_sha256 AS corporativo_organizacion_procedencia_huella_sha256,
  cv.organizacion_procedencia_autoridad AS corporativo_organizacion_procedencia_autoridad,
  ov.version AS organizacion_version,ov.procedencia_ref AS organizacion_procedencia_ref,
  ov.procedencia_version AS organizacion_procedencia_version,
  ov.procedencia_huella_sha256 AS organizacion_procedencia_huella_sha256,
  ov.estado AS organizacion_estado,ov.vigente_desde AS organizacion_desde,
  ov.vigente_hasta AS organizacion_hasta,ov.procedencia_autoridad AS organizacion_autoridad
 INTO STRICT r
 FROM vec_contexto_actor_v1.vinculo_contexto_actual va
 JOIN vec_contexto_actor_v1.vinculo_contexto_versiones v
  ON v.vinculo_ref=va.vinculo_ref AND v.version=va.version
 JOIN vec_contexto_actor_v1.proyeccion_cuenta_actual ca ON ca.cuenta_ref=v.cuenta_ref
 JOIN vec_contexto_actor_v1.proyeccion_cuenta_versiones c
  ON c.cuenta_ref=ca.cuenta_ref AND c.version=ca.version
 JOIN vec_contexto_actor_v1.perfil_actual pa ON pa.perfil_ref=v.perfil_ref
 JOIN vec_contexto_actor_v1.perfil_versiones p
  ON p.perfil_ref=pa.perfil_ref AND p.version=pa.version
 JOIN vec_contexto_actor_v1.persona_actual xa ON xa.persona_ref=v.persona_ref
 JOIN vec_contexto_actor_v1.persona_versiones x
  ON x.persona_ref=xa.persona_ref AND x.version=xa.version
 JOIN vec_contexto_actor_v1.vinculo_corporativo_actual vc
  ON vc.cuenta_ref=v.cuenta_ref AND vc.superficie='interna_corporativa'
   AND vc.uso='consulta_rrhh'
 JOIN vec_contexto_actor_v1.vinculo_corporativo_versiones cv
  ON cv.vinculo_corporativo_ref=vc.vinculo_corporativo_ref AND cv.version=vc.version
 JOIN vec_contexto_actor_v1.organizacion_actual oa
  ON oa.organizacion_ref=cv.organizacion_ref
 JOIN vec_contexto_actor_v1.organizacion_versiones ov
  ON ov.organizacion_ref=oa.organizacion_ref AND ov.version=oa.version
 WHERE va.vinculo_ref=j->>'contexto_actor_ref'
  AND v.cuenta_ref=j->>'cuenta_ref' AND v.persona_ref=j->>'persona_ref'
  AND v.perfil_ref=j->>'perfil_activo_ref'
  AND cv.superficie='interna_corporativa' AND cv.uso='consulta_rrhh'
 FOR SHARE OF va,ca,pa,xa,vc,oa;
 ahora:=clock_timestamp();
 RETURN r.version IS NOT DISTINCT FROM (j->>'contexto_version')::numeric
  AND r.cuenta_version IS NOT DISTINCT FROM (j->>'cuenta_version')::numeric
  AND r.persona_version IS NOT DISTINCT FROM (j->>'persona_version')::numeric
  AND r.perfil_version IS NOT DISTINCT FROM (j->>'perfil_version')::numeric
  AND r.estado='activo' AND r.cuenta_estado='activo'
  AND r.persona_estado='activo' AND r.perfil_estado='activo'
  AND r.corporativo_estado='activo' AND r.organizacion_estado='activo'
  AND r.procedencia_autoridad='autoridad_maestra_acreditada'
  AND r.cuenta_autoridad='autoridad_maestra_acreditada'
  AND r.persona_autoridad='autoridad_maestra_acreditada'
  AND r.perfil_autoridad='autoridad_maestra_acreditada'
  AND r.corporativo_autoridad='autoridad_maestra_acreditada'
  AND r.organizacion_autoridad='autoridad_maestra_acreditada'
  AND r.persona_ref IS NOT DISTINCT FROM r.perfil_persona_ref
  AND r.cuenta_ref IS NOT DISTINCT FROM r.corporativo_cuenta_ref
  AND r.perfil_ref IS NOT DISTINCT FROM r.corporativo_perfil_ref
  AND r.persona_ref IS NOT DISTINCT FROM r.corporativo_persona_ref
  AND r.vinculo_ref IS NOT DISTINCT FROM r.corporativo_vinculo_ref
  AND r.version IS NOT DISTINCT FROM r.corporativo_vinculo_version
  AND r.cuenta_version IS NOT DISTINCT FROM r.corporativo_cuenta_version
  AND r.perfil_version IS NOT DISTINCT FROM r.corporativo_perfil_version
  AND r.persona_version IS NOT DISTINCT FROM r.corporativo_persona_version
  AND r.organizacion_version IS NOT DISTINCT FROM r.corporativo_organizacion_version
  AND r.organizacion_procedencia_ref IS NOT DISTINCT FROM r.corporativo_organizacion_procedencia_ref
  AND r.organizacion_procedencia_version IS NOT DISTINCT FROM r.corporativo_organizacion_procedencia_version
  AND r.organizacion_procedencia_huella_sha256 IS NOT DISTINCT FROM r.corporativo_organizacion_procedencia_huella_sha256
  AND r.organizacion_autoridad IS NOT DISTINCT FROM r.corporativo_organizacion_procedencia_autoridad
  AND ahora>=GREATEST(r.vigente_desde,r.cuenta_desde,r.persona_desde,r.perfil_desde,
   r.corporativo_desde,r.organizacion_desde)
  AND ahora<LEAST(r.vigente_hasta,r.cuenta_hasta,r.persona_hasta,r.perfil_hasta,
   r.corporativo_hasta,r.organizacion_hasta,(j->>'vigente_hasta')::timestamptz);
EXCEPTION WHEN no_data_found OR too_many_rows OR data_exception OR invalid_text_representation THEN
 RETURN false;
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.cotejar_contexto_lectura_bolsa_actual_v1(bytea) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_contexto_actor_v1 TO vec_autorizacion_atestada_v3_propietario;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.cotejar_contexto_lectura_bolsa_actual_v1(bytea)
 TO vec_autorizacion_atestada_v3_propietario;
RESET ROLE;
DO $post$
DECLARE h text;f oid:=to_regprocedure('vec_contexto_actor_v1.cotejar_contexto_lectura_bolsa_actual_v1(bytea)');
BEGIN
 IF f IS NULL OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f
  AND p.proowner='vec_contexto_actor_v1_propietario'::regrole AND p.prosecdef
  AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on','lock_timeout=2s'])
 OR has_function_privilege('public',f,'EXECUTE') IS TRUE
 OR has_function_privilege('vec_bolsa_llamamientos_ejecutor',f,'EXECUTE') IS TRUE
 OR has_function_privilege('vec_autorizacion_atestada_v3_propietario',f,'EXECUTE') IS NOT TRUE
 THEN RAISE EXCEPTION 'CA40: postimagen ACL o dueño divergente' USING ERRCODE='55000'; END IF;
 SELECT encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex') INTO h;
 IF h IS DISTINCT FROM 'cdaf5ff8bd35c92084d14e6747103c6859c2e438778c9dc4c7167a78f8eb77e0'
 THEN RAISE EXCEPTION 'CA40: postimagen actual=%',h USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
