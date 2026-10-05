\set ON_ERROR_STOP on
-- B79: constitución de una bolsa desde un acta de CONVOCA ya importada cuando
-- RRHH confirma la carga desde la pantalla. Consume la decisión V3 propia de la
-- carga (AD203) en la misma transacción SERIALIZABLE que constituir_bolsa_v1;
-- el consumo deja el asiento en la auditoría común con el acta como recurso.
-- Si la decisión no vale, no es del actor o no es de esa acta, la transacción
-- se revierte y no queda ni consumo ni bolsa. La categoría queda ligada al acta:
-- la referencia del acta es sha256(huella del fichero || 0x1F || categoría) y
-- la bolsa canónica debe llevar esa huella, esa categoría y esa bolsa. constituir_bolsa_v1 no cambia y
-- sigue disponible para la línea de órdenes (constituir-bolsa).
-- Una sola vez; sin DOWN. Requiere AD203.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:migracion:000079',0));

DO $precondicion$
DECLARE actual jsonb; esperado jsonb:=pg_catalog.jsonb_build_object(
 'rol',true,'ad203',true,'constituir',true,'libre',true,'ejecutor',true);
BEGIN
 actual:=pg_catalog.jsonb_build_object(
  'rol',current_user='vec_bolsa_llamamientos_propietario',
  'ad203',pg_catalog.has_function_privilege(
    'vec_autorizacion_atestada_v3.consumir_carga_convoca_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE'),
  'constituir',EXISTS(SELECT 1 FROM pg_catalog.pg_proc
    WHERE oid=pg_catalog.to_regprocedure('vec_bolsa_llamamientos.constituir_bolsa_v1(text,text,text,text,bigint,bytea,timestamptz,text,bigint,bytea,timestamptz,timestamptz,jsonb,timestamptz)')
      AND proowner='vec_bolsa_llamamientos_propietario'::pg_catalog.regrole AND prosecdef),
  'libre',pg_catalog.to_regprocedure('vec_bolsa_llamamientos.constituir_bolsa_carga_convoca_v1(text,text,text,text,bigint,bytea,timestamptz,text,bigint,bytea,timestamptz,timestamptz,jsonb,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL,
  'ejecutor',EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname='vec_bolsa_llamamientos_ejecutor' AND NOT rolcanlogin AND NOT rolbypassrls));
 IF actual IS DISTINCT FROM esperado THEN
  RAISE EXCEPTION 'PARO clave=B79.preimagen, actual=%, esperado=%',actual,esperado USING ERRCODE='55000';
 END IF;
END $precondicion$;

CREATE FUNCTION vec_bolsa_llamamientos.constituir_bolsa_carga_convoca_v1(
 p_acta_ref text,p_actor_ref text,p_categoria_ref text,p_bolsa_ref text,p_version_bolsa bigint,
 p_bolsa_canonica bytea,p_vigente_desde timestamptz,p_instantanea_ref text,p_version_instantanea bigint,
 p_instantanea_canonica bytea,p_referida_en timestamptz,p_generada_en timestamptz,p_entradas jsonb,
 p_confirmada_en timestamptz,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,
 p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET timezone='UTC'
SET lock_timeout='2s' SET statement_timeout='30s' AS $f$
DECLARE consumo record; decision jsonb; recibo jsonb; bolsa jsonb;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR pg_catalog.current_setting('transaction_isolation')<>'serializable'
    OR p_acta_ref IS NULL OR p_acta_ref !~ '^acta:importacion-convoca:[0-9a-f]{64}$'
    OR p_actor_ref IS NULL OR p_actor_ref<>pg_catalog.btrim(p_actor_ref)
    OR pg_catalog.octet_length(p_actor_ref) NOT BETWEEN 1 AND 256 THEN
  RAISE EXCEPTION 'B79: carga de bolsa no autorizada' USING ERRCODE='42501'; END IF;
 BEGIN decision:=pg_catalog.convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'B79: carga de bolsa no autorizada' USING ERRCODE='42501'; END;
 IF pg_catalog.jsonb_typeof(decision) IS DISTINCT FROM 'object'
    OR decision->>'principal_id' IS DISTINCT FROM p_actor_ref
    OR decision->>'recurso_ref' IS DISTINCT FROM p_acta_ref THEN
  RAISE EXCEPTION 'B79: carga de bolsa no autorizada' USING ERRCODE='42501'; END IF;
 BEGIN bolsa:=pg_catalog.convert_from(p_bolsa_canonica,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'B79: carga de bolsa no autorizada' USING ERRCODE='42501'; END;
 IF pg_catalog.jsonb_typeof(bolsa) IS DISTINCT FROM 'object'
    OR bolsa->>'categoria_ref' IS DISTINCT FROM p_categoria_ref
    OR bolsa->>'bolsa_ref' IS DISTINCT FROM p_bolsa_ref
    OR bolsa->>'huella_listado_sha256' IS NULL OR bolsa->>'huella_listado_sha256' !~ '^[0-9a-f]{64}$'
    OR p_acta_ref IS DISTINCT FROM 'acta:importacion-convoca:'||pg_catalog.encode(pg_catalog.sha256(
         pg_catalog.convert_to((bolsa->>'huella_listado_sha256')||pg_catalog.chr(31)||p_categoria_ref,'UTF8')),'hex') THEN
  RAISE EXCEPTION 'B79: carga de bolsa no autorizada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.consumir_carga_convoca_bolsa_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo.consumo_nuevo IS NOT TRUE OR consumo.efecto_ref IS DISTINCT FROM p_acta_ref
    OR consumo.decision_ref IS NULL OR consumo.auditoria_ref IS NULL OR consumo.consumida_en IS NULL THEN
  RAISE EXCEPTION 'B79: carga de bolsa no autorizada' USING ERRCODE='42501'; END IF;
 recibo:=vec_bolsa_llamamientos.constituir_bolsa_v1(
  p_acta_ref,p_actor_ref,p_categoria_ref,p_bolsa_ref,p_version_bolsa,p_bolsa_canonica,p_vigente_desde,
  p_instantanea_ref,p_version_instantanea,p_instantanea_canonica,p_referida_en,p_generada_en,p_entradas,p_confirmada_en);
 IF pg_catalog.jsonb_typeof(recibo) IS DISTINCT FROM 'object' OR recibo->>'acta_ref' IS DISTINCT FROM p_acta_ref THEN
  RAISE EXCEPTION 'B79: constitución divergente' USING ERRCODE='42501'; END IF;
 RETURN recibo||pg_catalog.jsonb_build_object(
  'decision_ref',consumo.decision_ref,'auditoria_ref',consumo.auditoria_ref,
  'consumida_en',pg_catalog.to_char(consumo.consumida_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'));
END $f$;

DO $acl$
DECLARE f pg_catalog.regprocedure:='vec_bolsa_llamamientos.constituir_bolsa_carga_convoca_v1(text,text,text,text,bigint,bytea,timestamptz,text,bigint,bytea,timestamptz,timestamptz,jsonb,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::pg_catalog.regprocedure;
 x record;
BEGIN
 FOR x IN SELECT DISTINCT a.grantee FROM pg_catalog.pg_proc q
  CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(q.proacl,pg_catalog.acldefault('f',q.proowner))) a
  WHERE q.oid=f AND a.grantee<>q.proowner LOOP
  EXECUTE pg_catalog.format('REVOKE ALL ON FUNCTION %s FROM %s',f::text,
   CASE WHEN x.grantee=0 THEN 'PUBLIC' ELSE pg_catalog.quote_ident(pg_catalog.pg_get_userbyid(x.grantee)) END);
 END LOOP;
 EXECUTE pg_catalog.format('GRANT EXECUTE ON FUNCTION %s TO vec_bolsa_llamamientos_ejecutor',f::text);
 IF (SELECT proowner FROM pg_catalog.pg_proc WHERE oid=f) IS DISTINCT FROM 'vec_bolsa_llamamientos_propietario'::pg_catalog.regrole
    OR NOT (SELECT prosecdef FROM pg_catalog.pg_proc WHERE oid=f)
    OR NOT pg_catalog.has_function_privilege('vec_bolsa_llamamientos_ejecutor',f,'EXECUTE')
    OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_proc q
         CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(q.proacl,pg_catalog.acldefault('f',q.proowner))) a
         WHERE q.oid=f AND a.grantee NOT IN (q.proowner,'vec_bolsa_llamamientos_ejecutor'::pg_catalog.regrole))<>0 THEN
  RAISE EXCEPTION 'PARO clave=B79.ACL, esperado=propietario_y_ejecutor' USING ERRCODE='55000';
 END IF;
END $acl$;
COMMIT;
