\set ON_ERROR_STOP on
-- CA48 (tarea F1-105): las lecturas de una misma persona ya no se interbloquean.
--
-- La ficha del candidato de Bolsa lanza seis peticiones a la vez. La de datos
-- de contacto coteja el contexto actual con
-- cotejar_contexto_lectura_bolsa_actual_v1 (CA40, rehecha en CA42 y CA43), que
-- bloquea con FOR SHARE OF va,ca,pa,xa,vc,oa: primero el vínculo de contexto y
-- después la cuenta. Las otras cinco registran su decisión V3
-- (registrar_decision_contexto_actor_v3), que acredita el uso del contexto
-- (acreditar_uso_registro_contexto_interno_v2, CA2) con FOR UPDATE en el orden
-- común del resolutor (CA36): cuenta, perfil, vínculo de contexto, persona y
-- referencias. Con órdenes distintos, la lectura retenía el vínculo y esperaba
-- la cuenta mientras la decisión retenía la cuenta y esperaba el vínculo:
-- PostgreSQL abortaba una de las dos con 40P01 y la sección salía en 503.
--
-- CA48 solo reconstruye el cotejo: bloquea los cuatro punteros uno a uno en ese
-- orden común, y después la misma consulta con FOR SHARE OF ca,pa,va,xa,vc,oa.
-- El nivel sigue siendo FOR SHARE: los punteros no pueden cambiar hasta el
-- COMMIT que asienta la lectura, como antes. No cambia firma, resultado,
-- comprobaciones, propietario, configuración ni permisos; no toca la
-- acreditación, el resolutor ni el núcleo V3. Solo UP; sin DOWN.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contexto_actor_v1:migracion:000048',0));
SET LOCAL ROLE vec_contexto_actor_v1_propietario;

DO $pre$
DECLARE
 f regprocedure := to_regprocedure('vec_contexto_actor_v1.cotejar_contexto_lectura_bolsa_actual_v1(bytea)');
 p pg_proc%ROWTYPE;
BEGIN
 SELECT * INTO p FROM pg_proc WHERE oid=f;
 IF current_user <> 'vec_contexto_actor_v1_propietario'
    OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
    OR f IS NULL
    OR to_regclass('vec_contexto_actor_v1.proyeccion_cuenta_actual') IS NULL
    OR to_regclass('vec_contexto_actor_v1.perfil_actual') IS NULL
    OR to_regclass('vec_contexto_actor_v1.vinculo_contexto_actual') IS NULL
    OR to_regclass('vec_contexto_actor_v1.persona_actual') IS NULL
    OR to_regclass('vec_contexto_actor_v1.vinculo_corporativo_actual') IS NULL
    OR to_regclass('vec_contexto_actor_v1.organizacion_actual') IS NULL
    -- Preimagen exacta de CA43 instalada.
    OR encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex')
       IS DISTINCT FROM '52e78b540d474ff49fa7f942794b5fb44ccb993d1b060d3a19d6922d35feeb14'
    OR p.proowner IS DISTINCT FROM 'vec_contexto_actor_v1_propietario'::regrole
    OR NOT p.prosecdef
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on','lock_timeout=2s']::text[]
    OR p.proacl IS DISTINCT FROM ARRAY['vec_contexto_actor_v1_propietario=X/vec_contexto_actor_v1_propietario',
       'vec_autorizacion_atestada_v3_propietario=X/vec_contexto_actor_v1_propietario']::aclitem[]
 THEN RAISE EXCEPTION 'CA48: PARO clave=cotejo_lectura_bolsa actual=deriva esperado=CA43; no reaplicar' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE OR REPLACE FUNCTION vec_contexto_actor_v1.cotejar_contexto_lectura_bolsa_actual_v1(p_contexto_canonico bytea)
 RETURNS boolean
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET lock_timeout TO '2s'
AS $function$
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
 -- CA48: los punteros se bloquean en el orden común del resolutor (CA36) y de
 -- la acreditación de uso (CA2): cuenta, perfil, vínculo de contexto y persona.
 -- Con el orden de FOR SHARE OF de CA40-CA43 (vínculo antes que cuenta), una
 -- lectura de Bolsa y la decisión V3 de otra petición de la misma persona se
 -- esperaban en cruz (40P01). El nivel sigue siendo FOR SHARE.
 PERFORM 1 FROM vec_contexto_actor_v1.proyeccion_cuenta_actual ca
  WHERE ca.cuenta_ref=j->>'cuenta_ref' FOR SHARE OF ca;
 PERFORM 1 FROM vec_contexto_actor_v1.perfil_actual pa
  WHERE pa.perfil_ref=j->>'perfil_activo_ref' FOR SHARE OF pa;
 PERFORM 1 FROM vec_contexto_actor_v1.vinculo_contexto_actual va
  WHERE va.vinculo_ref=j->>'contexto_actor_ref' FOR SHARE OF va;
 PERFORM 1 FROM vec_contexto_actor_v1.persona_actual xa
  WHERE xa.persona_ref=j->>'persona_ref' FOR SHARE OF xa;
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
  ON vc.cuenta_ref=v.cuenta_ref AND vc.perfil_ref=v.perfil_ref
   AND vc.superficie='interna_corporativa'
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
 FOR SHARE OF ca,pa,va,xa,vc,oa;
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
END $function$
;

DO $post$
DECLARE
 f regprocedure := 'vec_contexto_actor_v1.cotejar_contexto_lectura_bolsa_actual_v1(bytea)'::regprocedure;
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f
       AND p.proowner='vec_contexto_actor_v1_propietario'::regrole AND p.prosecdef
       AND p.prorettype='boolean'::regtype
       AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on','lock_timeout=2s']::text[]
       AND p.proacl=ARRAY['vec_contexto_actor_v1_propietario=X/vec_contexto_actor_v1_propietario',
        'vec_autorizacion_atestada_v3_propietario=X/vec_contexto_actor_v1_propietario']::aclitem[])
    OR encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex')
       IS DISTINCT FROM '16b225cf7f61482fb14748c8bdbea409413e41a9528be59f5039451cd4c83893'
 THEN RAISE EXCEPTION 'CA48: PARO clave=posimagen actual=deriva esperado=orden_comun' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
