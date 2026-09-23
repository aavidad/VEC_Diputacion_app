-- CA6: seleccion nominal y reacreditacion del ambito corporativo RRHH.
-- Instalar despues de CA4/CA5; nunca modificar la historia instalada.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
DO $preimagen$
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
    OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
    OR current_setting('server_encoding') <> 'UTF8'
    OR to_regclass('vec_contexto_actor_v1.vinculo_corporativo_actual') IS NULL
    OR to_regclass('vec_contexto_actor_v1.organizacion_actual') IS NULL
    OR to_regclass('vec_contexto_actor_v1.control_generacion_punteros_actuales_v2') IS NULL
    OR to_regprocedure('vec_contexto_actor_v1.leer_contexto_original_v2(text,text,text)') IS NULL
    OR to_regrole('vec_contexto_actor_corporativo_rrhh_selector') IS NULL
    OR to_regrole('vec_contratacion_temporal_propietario') IS NULL
    OR has_schema_privilege('vec_contexto_actor_corporativo_rrhh_selector',
         'vec_contexto_actor_v1','USAGE')
    OR has_schema_privilege('vec_contratacion_temporal_propietario',
         'vec_contexto_actor_v1','USAGE')
    OR EXISTS (SELECT 1 FROM pg_namespace n,
        LATERAL aclexplode(coalesce(n.nspacl,acldefault('n',n.nspowner))) a
       WHERE n.nspname='vec_contexto_actor_v1'
         AND a.grantee IN ('vec_contexto_actor_corporativo_rrhh_selector'::regrole,
                           'vec_contratacion_temporal_propietario'::regrole))
 THEN RAISE EXCEPTION 'CA6: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $preimagen$;

SET LOCAL ROLE vec_contexto_actor_v1_propietario;
SET LOCAL search_path = pg_catalog;

-- La lectura de punteros, incluida la generacion, ocurre tras el advisory
-- compartido. Un escritor previo invisible provoca 40001 sobre la fila MVCC;
-- uno posterior espera hasta que termine la transaccion lectora.
CREATE FUNCTION vec_contexto_actor_v1.resolver_comprobante_ambito_rrhh_v1(
 p_cuenta_ref text, p_cuenta_version numeric,
 p_persona_ref text, p_persona_version numeric,
 p_perfil_ref text, p_perfil_version numeric,
 p_contexto_ref text, p_contexto_version numeric
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path = pg_catalog SET row_security = 'on' SET timezone = 'UTC'
AS $funcion$
DECLARE
 c record; p record; f record; x record; v record; o record;
 generacion numeric; ahora timestamptz;
BEGIN
 IF current_setting('transaction_read_only') <> 'off'
    OR vec_contexto_actor_v1.referencia_valida(p_cuenta_ref,'cta_') IS NOT TRUE
    OR vec_contexto_actor_v1.referencia_valida(p_persona_ref,'per_') IS NOT TRUE
    OR vec_contexto_actor_v1.referencia_valida(p_perfil_ref,'prf_') IS NOT TRUE
    OR vec_contexto_actor_v1.referencia_valida(p_contexto_ref,'vca_') IS NOT TRUE
    OR p_cuenta_version IS NULL OR p_persona_version IS NULL
    OR p_perfil_version IS NULL OR p_contexto_version IS NULL
    OR p_cuenta_version NOT BETWEEN 1 AND 18446744073709551615::numeric
    OR p_persona_version NOT BETWEEN 1 AND 18446744073709551615::numeric
    OR p_perfil_version NOT BETWEEN 1 AND 18446744073709551615::numeric
    OR p_contexto_version NOT BETWEEN 1 AND 18446744073709551615::numeric
    OR scale(p_cuenta_version)<>0 OR scale(p_persona_version)<>0
    OR scale(p_perfil_version)<>0 OR scale(p_contexto_version)<>0
 THEN RETURN NULL; END IF;

 PERFORM pg_advisory_xact_lock_shared(hashtextextended(
  'vec_contexto_actor_v1:mutacion_punteros_actuales:v2',0));
 SELECT z.* INTO c FROM vec_contexto_actor_v1.proyeccion_cuenta_actual a
 JOIN vec_contexto_actor_v1.proyeccion_cuenta_versiones z USING(cuenta_ref,version)
 WHERE a.cuenta_ref=p_cuenta_ref FOR SHARE OF a;
 IF NOT FOUND THEN RETURN NULL; END IF;
 SELECT z.* INTO f FROM vec_contexto_actor_v1.perfil_actual a
 JOIN vec_contexto_actor_v1.perfil_versiones z USING(perfil_ref,version)
 WHERE a.perfil_ref=p_perfil_ref FOR SHARE OF a;
 IF NOT FOUND THEN RETURN NULL; END IF;
 SELECT z.* INTO x FROM vec_contexto_actor_v1.vinculo_contexto_actual a
 JOIN vec_contexto_actor_v1.vinculo_contexto_versiones z USING(vinculo_ref,version)
 WHERE a.vinculo_ref=p_contexto_ref FOR SHARE OF a;
 IF NOT FOUND THEN RETURN NULL; END IF;
 SELECT z.* INTO p FROM vec_contexto_actor_v1.persona_actual a
 JOIN vec_contexto_actor_v1.persona_versiones z USING(persona_ref,version)
 WHERE a.persona_ref=p_persona_ref FOR SHARE OF a;
 IF NOT FOUND THEN RETURN NULL; END IF;
 SELECT z.* INTO v FROM vec_contexto_actor_v1.vinculo_corporativo_actual a
 JOIN vec_contexto_actor_v1.vinculo_corporativo_versiones z
   USING(cuenta_ref,superficie,uso,vinculo_corporativo_ref,version)
 WHERE a.cuenta_ref=p_cuenta_ref AND a.superficie='interna_corporativa'
   AND a.uso='consulta_rrhh' FOR SHARE OF a;
 IF NOT FOUND THEN RETURN NULL; END IF;
 SELECT z.* INTO o FROM vec_contexto_actor_v1.organizacion_actual a
 JOIN vec_contexto_actor_v1.organizacion_versiones z USING(organizacion_ref,version)
 WHERE a.organizacion_ref=v.organizacion_ref FOR SHARE OF a;
 IF NOT FOUND THEN RETURN NULL; END IF;

 SELECT g.generacion INTO STRICT generacion
 FROM vec_contexto_actor_v1.control_generacion_punteros_actuales_v2 g
 WHERE g.control_id FOR SHARE;
 ahora:=clock_timestamp();
 IF c.version IS DISTINCT FROM p_cuenta_version OR c.estado<>'activo'
    OR c.procedencia_autoridad<>'autoridad_maestra_acreditada'
    OR p.version IS DISTINCT FROM p_persona_version OR p.estado<>'activo'
    OR p.procedencia_autoridad<>'autoridad_maestra_acreditada'
    OR f.version IS DISTINCT FROM p_perfil_version OR f.estado<>'activo'
    OR f.procedencia_autoridad<>'autoridad_maestra_acreditada'
    OR x.version IS DISTINCT FROM p_contexto_version OR x.estado<>'activo'
    OR x.procedencia_autoridad<>'autoridad_maestra_acreditada'
    OR v.estado<>'activo' OR o.estado<>'activo'
    OR f.persona_ref IS DISTINCT FROM p_persona_ref
    OR x.cuenta_ref IS DISTINCT FROM p_cuenta_ref
    OR x.persona_ref IS DISTINCT FROM p_persona_ref
    OR x.perfil_ref IS DISTINCT FROM p_perfil_ref
    OR v.cuenta_version IS DISTINCT FROM c.version
    OR v.persona_ref IS DISTINCT FROM p_persona_ref
    OR v.persona_version IS DISTINCT FROM p.version
    OR v.perfil_ref IS DISTINCT FROM p_perfil_ref
    OR v.perfil_version IS DISTINCT FROM f.version
    OR v.vinculo_contexto_ref IS DISTINCT FROM p_contexto_ref
    OR v.vinculo_contexto_version IS DISTINCT FROM x.version
    OR v.organizacion_version IS DISTINCT FROM o.version
    OR v.organizacion_procedencia_ref IS DISTINCT FROM o.procedencia_ref
    OR v.organizacion_procedencia_version IS DISTINCT FROM o.procedencia_version
    OR v.organizacion_procedencia_huella_sha256 IS DISTINCT FROM o.procedencia_huella_sha256
    OR v.organizacion_procedencia_autoridad IS DISTINCT FROM o.procedencia_autoridad
    OR o.procedencia_autoridad<>'autoridad_maestra_acreditada'
    OR v.procedencia_autoridad<>'autoridad_maestra_acreditada'
    OR NOT (ahora>=c.vigente_desde AND ahora<c.vigente_hasta
      AND ahora>=p.vigente_desde AND ahora<p.vigente_hasta
      AND ahora>=f.vigente_desde AND ahora<f.vigente_hasta
      AND ahora>=x.vigente_desde AND ahora<x.vigente_hasta
      AND ahora>=v.vigente_desde AND ahora<v.vigente_hasta
      AND ahora>=o.vigente_desde AND ahora<o.vigente_hasta)
 THEN RETURN NULL; END IF;
 RETURN jsonb_build_object(
  'cuenta_ref',p_cuenta_ref,'cuenta_version',c.version,
  'persona_ref',p_persona_ref,'persona_version',p.version,
  'perfil_ref',p_perfil_ref,'perfil_version',f.version,
  'contexto_ref',p_contexto_ref,'contexto_version',x.version,
  'vinculo_corporativo_ref',v.vinculo_corporativo_ref,
  'vinculo_corporativo_version',v.version,
  'organizacion_ref',o.organizacion_ref,'organizacion_version',o.version,
  'organizacion_procedencia_ref',o.procedencia_ref,
  'organizacion_procedencia_version',o.procedencia_version,
  'organizacion_procedencia_huella_sha256',o.procedencia_huella_sha256,
  'organizacion_procedencia_autoridad',o.procedencia_autoridad,
  'vinculo_procedencia_ref',v.procedencia_ref,
  'vinculo_procedencia_version',v.procedencia_version,
  'vinculo_procedencia_huella_sha256',v.procedencia_huella_sha256,
  'vinculo_procedencia_autoridad',v.procedencia_autoridad,
  'superficie',v.superficie,'uso',v.uso,
  'vigente_desde',to_char(greatest(c.vigente_desde,p.vigente_desde,
    f.vigente_desde,x.vigente_desde,v.vigente_desde,o.vigente_desde)
    AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  'vigente_hasta',to_char(least(c.vigente_hasta,p.vigente_hasta,
    f.vigente_hasta,x.vigente_hasta,v.vigente_hasta,o.vigente_hasta)
    AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'));
END $funcion$;

CREATE FUNCTION vec_contexto_actor_v1.acreditar_ambito_rrhh_v1(
 p_comprobante jsonb, p_cuenta_ref text, p_persona_ref text,
 p_perfil_ref text, p_contexto_ref text, p_organizacion_ref text
) RETURNS timestamptz LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path = pg_catalog SET row_security = 'on' SET timezone = 'UTC'
AS $funcion$
DECLARE observado jsonb; ahora timestamptz;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
    OR p_comprobante IS NULL OR jsonb_typeof(p_comprobante)<>'object'
    OR (SELECT count(*) FROM jsonb_object_keys(p_comprobante))<>24
    OR p_comprobante->>'cuenta_ref' IS DISTINCT FROM p_cuenta_ref
    OR p_comprobante->>'persona_ref' IS DISTINCT FROM p_persona_ref
    OR p_comprobante->>'perfil_ref' IS DISTINCT FROM p_perfil_ref
    OR p_comprobante->>'contexto_ref' IS DISTINCT FROM p_contexto_ref
    OR p_comprobante->>'organizacion_ref' IS DISTINCT FROM p_organizacion_ref
 THEN RETURN NULL; END IF;
 observado:=vec_contexto_actor_v1.resolver_comprobante_ambito_rrhh_v1(
  p_cuenta_ref,(p_comprobante->>'cuenta_version')::numeric,
  p_persona_ref,(p_comprobante->>'persona_version')::numeric,
  p_perfil_ref,(p_comprobante->>'perfil_version')::numeric,
  p_contexto_ref,(p_comprobante->>'contexto_version')::numeric);
 IF observado IS NULL OR observado<>p_comprobante THEN RETURN NULL; END IF;
 ahora:=clock_timestamp();
 IF ahora<(p_comprobante->>'vigente_desde')::timestamptz
    OR ahora>=(p_comprobante->>'vigente_hasta')::timestamptz
 THEN RETURN NULL; END IF;
 RETURN ahora;
EXCEPTION WHEN invalid_text_representation OR numeric_value_out_of_range
  OR datetime_field_overflow THEN RETURN NULL;
END $funcion$;

REVOKE ALL ON FUNCTION vec_contexto_actor_v1.resolver_comprobante_ambito_rrhh_v1(
 text,numeric,text,numeric,text,numeric,text,numeric) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.acreditar_ambito_rrhh_v1(
 jsonb,text,text,text,text,text) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_contexto_actor_v1
 TO vec_contexto_actor_corporativo_rrhh_selector,vec_contratacion_temporal_propietario;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.resolver_comprobante_ambito_rrhh_v1(
 text,numeric,text,numeric,text,numeric,text,numeric)
 TO vec_contexto_actor_corporativo_rrhh_selector;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.acreditar_ambito_rrhh_v1(
 jsonb,text,text,text,text,text) TO vec_contratacion_temporal_propietario;
DO $postimagen_acl$
DECLARE cantidad integer;
BEGIN
 SELECT count(*) INTO cantidad FROM pg_namespace n,
  LATERAL aclexplode(n.nspacl) a
 WHERE n.nspname='vec_contexto_actor_v1'
   AND a.grantee IN ('vec_contexto_actor_corporativo_rrhh_selector'::regrole,
                     'vec_contratacion_temporal_propietario'::regrole)
   AND a.grantor='vec_contexto_actor_v1_propietario'::regrole
   AND a.privilege_type='USAGE' AND NOT a.is_grantable;
 IF cantidad<>2 OR EXISTS (SELECT 1 FROM pg_namespace n,
   LATERAL aclexplode(n.nspacl) a
   WHERE n.nspname='vec_contexto_actor_v1'
     AND a.grantee IN ('vec_contexto_actor_corporativo_rrhh_selector'::regrole,
                       'vec_contratacion_temporal_propietario'::regrole)
     AND (a.grantor<>'vec_contexto_actor_v1_propietario'::regrole
       OR a.privilege_type<>'USAGE' OR a.is_grantable))
 THEN RAISE EXCEPTION 'CA6: ACL de esquema incompatible' USING ERRCODE='55000'; END IF;
END $postimagen_acl$;
COMMIT;
