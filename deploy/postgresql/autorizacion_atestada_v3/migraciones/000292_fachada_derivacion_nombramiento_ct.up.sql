\set ON_ERROR_STOP on
-- AD292 / F1-19: fachada nominal CT y origen técnico desde el pool existente.
-- Instalar tras AD291 y antes de CT234, una sola vez en la candidata revisada.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
LOCK TABLE vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1 IN SHARE ROW EXCLUSIVE MODE;
DO $pre$ DECLARE f regprocedure;
BEGIN
 f:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR current_user<>'vec_autorizacion_atestada_v3_propietario' OR f IS NULL
 OR encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex') IS DISTINCT FROM
   '6e07b8c0690439b3071a0d82fbb230d936e5848faddd84b59c6c8acc08292bd0'
 OR NOT EXISTS(SELECT 1 FROM pg_proc WHERE oid=f AND prosecdef
   AND proowner='vec_autorizacion_atestada_v3_propietario'::regrole
   AND proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
   AND proacl::text='{vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario}')
 OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_derivacion_nombramiento_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 OR (SELECT count(*) FROM vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1
   WHERE audiencia_consumo='vec_contratacion_temporal.vinculo_emision_bolsa.v1'
    AND operacion='contratacion_temporal.bolsa.vincular'
    AND proceso='vec-server' AND canal_permitido='interna_corporativa')<>1
 OR EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1
   WHERE audiencia_consumo='vec.ct.nombramiento-derivado.v1'
      OR operacion='contratacion_temporal.nombramiento.derivar')
 THEN RAISE EXCEPTION 'AD292: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_derivacion_nombramiento_ct_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,
 consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; x record;
BEGIN
 IF p_capacidad IS NULL OR octet_length(p_capacidad) NOT BETWEEN 1 AND 524288
 OR p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 1 AND 524288
 THEN RAISE EXCEPTION 'AD292: material inválido' USING ERRCODE='22023'; END IF;
 BEGIN
  c:=convert_from(p_capacidad,'UTF8')::jsonb;
  d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception THEN
  RAISE EXCEPTION 'AD292: material inválido' USING ERRCODE='22023';
 END;
 IF jsonb_typeof(c) IS DISTINCT FROM 'object' OR jsonb_typeof(d) IS DISTINCT FROM 'object'
 OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec.ct.nombramiento-derivado.v1'
 OR c->>'operacion' IS DISTINCT FROM 'contratacion_temporal.nombramiento.derivar'
 OR d->>'accion' IS DISTINCT FROM c->>'operacion'
 OR d->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
 OR d->>'tipo_recurso' IS DISTINCT FROM 'nombramiento_derivado_ct'
 OR d->>'finalidad' IS DISTINCT FROM 'gestionar_contratacion_temporal'
 OR d->>'garantia_minima' IS DISTINCT FROM 'alto'
 OR d->>'recurso_ref' IS NULL
 OR d->>'recurso_ref' !~ '^expediente:nombramiento:[0-9a-f]{64}$'
 OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
 OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
 OR d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'interna_corporativa'
 OR d#>>'{vinculo_autenticacion_actor,metodo_observado}' IS DISTINCT FROM 'certificado'
 OR d#>>'{vinculo_autenticacion_actor,garantia_observada}' IS DISTINCT FROM 'alto'
 OR jsonb_typeof(d->'ambitos') IS DISTINCT FROM 'object'
 OR (SELECT array_agg(k ORDER BY k) FROM jsonb_object_keys(d->'ambitos') k)
    IS DISTINCT FROM ARRAY['organizacion_ref','unidad_ref']
 OR d->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb
 OR d->'obligaciones' IS DISTINCT FROM '["auditar"]'::jsonb
 THEN RAISE EXCEPTION 'AD292: contrato nominal rechazado' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  'ct_nombramiento_derivado',p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NULL OR x.decision_ref IS DISTINCT FROM c->>'decision_ref'
 OR x.efecto_ref IS DISTINCT FROM c->>'efecto_ref'
 OR x.huella_efecto_sha256 IS DISTINCT FROM c->>'huella_efecto_sha256'
 THEN RAISE EXCEPTION 'AD292: acuse de consumo divergente' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,
  x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,x.consumo_nuevo;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_derivacion_nombramiento_ct_v3_atestada(
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_derivacion_nombramiento_ct_v3_atestada(
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_contratacion_temporal_propietario;

INSERT INTO vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1
 (login_nombre,audiencia_consumo,operacion,proceso,canal_permitido)
SELECT anterior.login_nombre,'vec.ct.nombramiento-derivado.v1',
 'contratacion_temporal.nombramiento.derivar',anterior.proceso,anterior.canal_permitido
FROM vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1 anterior
WHERE anterior.audiencia_consumo='vec_contratacion_temporal.vinculo_emision_bolsa.v1'
 AND anterior.operacion='contratacion_temporal.bolsa.vincular'
 AND anterior.proceso='vec-server' AND anterior.canal_permitido='interna_corporativa'
 AND EXISTS(SELECT 1 FROM pg_roles l JOIN pg_auth_members m ON m.member=l.oid
  WHERE l.rolname=anterior.login_nombre AND l.rolcanlogin AND l.rolinherit
   AND NOT(l.rolsuper OR l.rolcreatedb OR l.rolcreaterole OR l.rolreplication OR l.rolbypassrls)
   AND m.roleid='vec_contratacion_temporal_ejecutor'::regrole
   AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option
   AND (SELECT count(*) FROM pg_auth_members x WHERE x.member=l.oid)=1);

DO $post$ DECLARE f regprocedure;
BEGIN
 f:='vec_autorizacion_atestada_v3.registrar_y_consumir_derivacion_nombramiento_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 IF NOT EXISTS(SELECT 1 FROM pg_proc WHERE oid=f AND prosecdef AND provolatile='v'
   AND proowner='vec_autorizacion_atestada_v3_propietario'::regrole
   AND proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
   AND proacl::text='{vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario,vec_contratacion_temporal_propietario=X/vec_autorizacion_atestada_v3_propietario}')
 OR (SELECT count(*) FROM vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1
   WHERE audiencia_consumo='vec.ct.nombramiento-derivado.v1'
    AND operacion='contratacion_temporal.nombramiento.derivar'
    AND proceso='vec-server' AND canal_permitido='interna_corporativa')<>1
 THEN RAISE EXCEPTION 'AD292: postimagen incompatible' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
