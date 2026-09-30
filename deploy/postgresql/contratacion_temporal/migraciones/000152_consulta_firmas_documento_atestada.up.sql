\set ON_ERROR_STOP on
-- CT152: consulta nominal sobre la proyección histórica CT145. No crea
-- firmas ni custodia. AD3-125 debe estar instalada antes.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:000152',0));
DO $pre$
DECLARE f regprocedure; x record;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario'
    OR to_regclass('vec_contratacion_temporal.expediente_integral_actual') IS NULL
    OR to_regclass('vec_contratacion_temporal.expediente_version_integral') IS NULL
    OR to_regclass('vec_contratacion_temporal.firma_documento_v1') IS NULL
    OR to_regclass('vec_contratacion_temporal.firma_documento_custodia_v1') IS NULL
    OR to_regprocedure('vec_contratacion_temporal.consultar_firmas_documento_v1(text,text)') IS NULL
    OR to_regprocedure('vec_contratacion_temporal.consultar_firmas_documento_v2(text,text)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_consulta_firmas_documento_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_contratacion_temporal.consultar_firmas_documento_atestadas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 THEN RAISE EXCEPTION 'CT152: preimagen incompatible' USING ERRCODE='55000'; END IF;
 -- La reversión solo puede restaurar la ACL nominal de CT145. Una ACL
 -- distinta se revisa expresamente; no se normaliza perdiendo su preimagen.
 FOREACH f IN ARRAY ARRAY[
  'vec_contratacion_temporal.consultar_firmas_documento_v1(text,text)'::regprocedure,
  'vec_contratacion_temporal.consultar_firmas_documento_v2(text,text)'::regprocedure
 ] LOOP
  IF (SELECT proowner FROM pg_proc WHERE oid=f)<>'vec_contratacion_temporal_propietario'::regrole
     OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS NOT TRUE
  THEN RAISE EXCEPTION 'CT152: lector histórico incompatible' USING ERRCODE='55000'; END IF;
  FOR x IN SELECT a.* FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f LOOP
   IF x.privilege_type<>'EXECUTE' OR x.is_grantable
      OR (x.grantee<>'vec_contratacion_temporal_propietario'::regrole::oid
          AND (f<>'vec_contratacion_temporal.consultar_firmas_documento_v2(text,text)'::regprocedure
               OR x.grantee<>'vec_contratacion_temporal_ejecutor'::regrole::oid))
   THEN RAISE EXCEPTION 'CT152: ACL previa incompatible' USING ERRCODE='55000'; END IF;
  END LOOP;
 END LOOP;
 IF NOT has_function_privilege('vec_contratacion_temporal_ejecutor','vec_contratacion_temporal.consultar_firmas_documento_v2(text,text)','EXECUTE')
 THEN RAISE EXCEPTION 'CT152: ACL previa CT145 ausente' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE FUNCTION vec_contratacion_temporal.consultar_firmas_documento_atestadas_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog SET row_security='on' SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE s jsonb; c jsonb; d jsonb; v_hash text; v_material_hash text; v_consumo record; v_firmas jsonb; v_total bigint;
 campos constant jsonb:='["CatalogoHuella","CatalogoRef","ClaveIdempotencia","ConMotivoDevolucion","Documento","DocumentoCustodiaRef","DocumentoCustodiaVersion","ExpedienteVersion","FirmaRef","FirmadoHuella","OriginalHuella","PasoOrden","PasoRef","ReciboRef","RegistradaEn","Resultado","Secuencia","SelloTiempoEstado"]'::jsonb;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario'
    OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
 THEN RAISE EXCEPTION 'CT152: sesión denegada' USING ERRCODE='42501'; END IF;
 IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 1 AND 512
 THEN RAISE EXCEPTION 'CT152: material inválido' USING ERRCODE='22023'; END IF;
 BEGIN s:=p_material::jsonb;
 EXCEPTION WHEN data_exception THEN RAISE EXCEPTION 'CT152: material inválido' USING ERRCODE='22023'; END;
 IF jsonb_typeof(s) IS DISTINCT FROM 'object'
 THEN RAISE EXCEPTION 'CT152: objeto requerido' USING ERRCODE='22023'; END IF;
 IF (SELECT count(*) FROM json_each(p_material::json))<>2
    OR (SELECT count(*) FROM jsonb_object_keys(s))<>2
    OR jsonb_typeof(s->'OrganizacionRef') IS DISTINCT FROM 'string'
    OR jsonb_typeof(s->'ExpedienteRef') IS DISTINCT FROM 'string'
    OR (s->>'OrganizacionRef')!~'^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
    OR (s->>'ExpedienteRef')!~'^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
    OR p_material IS DISTINCT FROM '{"OrganizacionRef":'||to_json(s->>'OrganizacionRef')::text||',"ExpedienteRef":'||to_json(s->>'ExpedienteRef')::text||'}'
 THEN RAISE EXCEPTION 'CT152: material no canónico' USING ERRCODE='22023'; END IF;
 v_material_hash:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 v_hash:=encode(sha256(convert_to(
  '{"ambitos":{"organizacion_ref":"'||(s->>'OrganizacionRef')||
  '"},"atributos":{"material_sha256":"'||v_material_hash||'"}}','UTF8')),'hex');
 IF p_capacidad IS NULL OR p_decision IS NULL OR p_motivo IS NULL OR p_contexto IS NULL
    OR p_payload IS NULL OR p_sobre IS NULL OR p_evidencia IS NULL OR p_raiz IS NULL
    OR octet_length(p_capacidad) NOT BETWEEN 512 AND 32768
    OR octet_length(p_decision) NOT BETWEEN 1 AND 524288
    OR octet_length(p_motivo) NOT BETWEEN 1 AND 65536
    OR octet_length(p_contexto) NOT BETWEEN 1 AND 262144
    OR octet_length(p_payload) NOT BETWEEN 1 AND 1048576
    OR octet_length(p_sobre) NOT BETWEEN 1 AND 1048576
    OR octet_length(p_evidencia) NOT BETWEEN 1 AND 262144
    OR octet_length(p_raiz)<>44
    OR p_persona_version IS NULL OR p_perfil_version IS NULL
    OR p_persona_version NOT BETWEEN 1 AND 9007199254740991
    OR p_perfil_version NOT BETWEEN 1 AND 9007199254740991
    OR p_persona_version<>trunc(p_persona_version)
    OR p_perfil_version<>trunc(p_perfil_version)
 THEN RAISE EXCEPTION 'CT152: autorización denegada' USING ERRCODE='42501'; END IF;
 BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'CT152: autorización denegada' USING ERRCODE='42501'; END;
 IF c->>'audiencia_consumo' IS DISTINCT FROM 'vec_contratacion_temporal.firmas_documento.consultar.v1'
    OR c->>'operacion' IS DISTINCT FROM 'contratacion_temporal.documento.firmas.consultar'
    OR c->>'efecto_ref' IS DISTINCT FROM s->>'ExpedienteRef'
    OR c->>'huella_efecto_sha256' IS DISTINCT FROM v_hash
    OR c->>'huella_decision_sha256' IS DISTINCT FROM encode(sha256(p_decision),'hex')
    OR d->>'accion' IS DISTINCT FROM c->>'operacion'
    OR d->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'expediente_contratacion_temporal'
    OR d->>'finalidad' IS DISTINCT FROM 'gestionar_contratacion_temporal'
    OR d->>'recurso_ref' IS DISTINCT FROM s->>'ExpedienteRef'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_hash
    OR d->'campos_permitidos' IS DISTINCT FROM campos
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'CT152: autorización divergente' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT v_consumo FROM vec_autorizacion_atestada_v3.consumir_consulta_firmas_documento_ct_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF v_consumo.consumo_nuevo IS NOT TRUE
    OR v_consumo.efecto_ref IS DISTINCT FROM s->>'ExpedienteRef'
    OR v_consumo.huella_efecto_sha256 IS DISTINCT FROM v_hash
 THEN RAISE EXCEPTION 'CT152: consumo divergente' USING ERRCODE='42501'; END IF;
 -- Ausencia autorizada: se devuelve un valor, sin excepción, para que el
 -- adaptador confirme consumo/auditoría antes de responder 404.
 IF NOT EXISTS (
  SELECT 1 FROM vec_contratacion_temporal.expediente_integral_actual a
  JOIN vec_contratacion_temporal.expediente_version_integral v ON v.expediente_ref=a.expediente_ref AND v.version=a.version
  WHERE a.expediente_ref=s->>'ExpedienteRef' AND v.agregado_json->>'organizacion_ref'=s->>'OrganizacionRef'
 ) THEN
  RETURN jsonb_build_object('Encontrado',false,'ExpedienteRef',s->>'ExpedienteRef','Firmas','[]'::jsonb);
 END IF;
 SELECT count(*) INTO v_total FROM vec_contratacion_temporal.firma_documento_v1
  WHERE organizacion_ref=s->>'OrganizacionRef' AND expediente_ref=s->>'ExpedienteRef';
 -- CT145 tiene un límite de 1000. No devolver una historia truncada como completa.
 IF v_total>1000 THEN RAISE EXCEPTION 'CT152: historia no representable' USING ERRCODE='P1525'; END IF;
 v_firmas:=vec_contratacion_temporal.consultar_firmas_documento_v2(s->>'OrganizacionRef',s->>'ExpedienteRef');
 IF jsonb_typeof(v_firmas) IS DISTINCT FROM 'array'
 THEN RAISE EXCEPTION 'CT152: proyección incompatible' USING ERRCODE='P1525'; END IF;
 IF jsonb_array_length(v_firmas)<>v_total OR EXISTS (
  SELECT 1 FROM jsonb_array_elements(v_firmas) f
  WHERE jsonb_typeof(f) IS DISTINCT FROM 'object'
     OR vec_contratacion_temporal.fiscalizacion_claves_exactas_v1(f,ARRAY(
         SELECT jsonb_array_elements_text(campos))) IS NOT TRUE
 ) THEN RAISE EXCEPTION 'CT152: proyección incompatible' USING ERRCODE='P1525'; END IF;
 RETURN jsonb_build_object('Encontrado',true,'ExpedienteRef',s->>'ExpedienteRef','Firmas',v_firmas);
EXCEPTION WHEN serialization_failure OR deadlock_detected OR lock_not_available THEN
 RAISE EXCEPTION 'CT152: consulta transitoria' USING ERRCODE='P1525';
END $f$;

DO $acl$
DECLARE f regprocedure; x record;
BEGIN
 FOREACH f IN ARRAY ARRAY[
  'vec_contratacion_temporal.consultar_firmas_documento_atestadas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
  'vec_contratacion_temporal.consultar_firmas_documento_v1(text,text)'::regprocedure,
  'vec_contratacion_temporal.consultar_firmas_documento_v2(text,text)'::regprocedure
 ] LOOP
  FOR x IN SELECT DISTINCT a.grantee FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f AND a.grantee<>p.proowner LOOP
   EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %s',f::text,CASE WHEN x.grantee=0 THEN 'PUBLIC' ELSE quote_ident(pg_get_userbyid(x.grantee)) END);
  END LOOP;
 END LOOP;
 GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.consultar_firmas_documento_atestadas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_contratacion_temporal_ejecutor;
 IF has_function_privilege('vec_contratacion_temporal_ejecutor','vec_contratacion_temporal.consultar_firmas_documento_v1(text,text)','EXECUTE')
    OR has_function_privilege('vec_contratacion_temporal_ejecutor','vec_contratacion_temporal.consultar_firmas_documento_v2(text,text)','EXECUTE')
 THEN RAISE EXCEPTION 'CT152: lectura directa abierta' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
