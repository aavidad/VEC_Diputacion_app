\set ON_ERROR_STOP on
-- AD3-100: consumo documental acotado al triple ámbito RRHH exacto.
-- Mantiene el núcleo V3 y deja sin EXECUTE la fachada AD3-99 documental.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000100',0));
DO $pre$
DECLARE nueva oid; vieja oid;
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario' THEN
  RAISE EXCEPTION 'AD3-100: propietario requerido' USING ERRCODE='55000'; END IF;
 nueva:=to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_plantillas_doc_ct_ambitos_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 vieja:=to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_plantillas_doc_ct_org_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF nueva IS NOT NULL OR vieja IS NULL
 THEN RAISE EXCEPTION 'AD3-100: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_plantillas_doc_ct_ambitos_v3_atestada(
 p_material jsonb,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; x record; material_h text; contexto_h text;
BEGIN
 BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'AD3-100: material inválido' USING ERRCODE='22023'; END;
 IF p_material IS NULL OR jsonb_typeof(p_material)<>'object'
    OR p_material->>'organizacion_ref' IS DISTINCT FROM 'organizacion:desarrollo:dipgra'
    OR p_material->>'clase_ambito' IS DISTINCT FROM 'organizacion'
    OR p_material->>'ambito_ref' IS DISTINCT FROM p_material->>'organizacion_ref'
    OR p_material->>'operacion' IS NULL
    OR p_material->>'operacion' <> ALL (ARRAY['listar','descargar'])
    OR p_material->>'expediente_ref' IS NULL
    OR (p_material->>'operacion'='listar' AND
      (p_material-ARRAY['operacion','organizacion_ref','clase_ambito','ambito_ref','expediente_ref','version_observada','consulta_huella_sha256'])<>'{}'::jsonb)
    OR (p_material->>'operacion'='descargar' AND
      (p_material-ARRAY['operacion','organizacion_ref','clase_ambito','ambito_ref','expediente_ref','version_observada','consulta_huella_sha256','tipo','formato'])<>'{}'::jsonb)
 THEN RAISE EXCEPTION 'AD3-100: ámbito organizativo requerido' USING ERRCODE='42501'; END IF;
 material_h:=encode(sha256(convert_to(p_material::text,'UTF8')),'hex');
 contexto_h:=encode(sha256(convert_to('{"ambitos":{"ambito_ref":"'||(p_material->>'ambito_ref')||'","clase_ambito":"organizacion","organizacion_ref":"'||(p_material->>'organizacion_ref')||'"},"atributos":{"material_sha256":"'||material_h||'"}}','UTF8')),'hex');
 IF d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM contexto_h
    OR c->>'huella_efecto_sha256' IS DISTINCT FROM contexto_h
    OR p_material->>'expediente_ref' IS NULL
    OR c->>'efecto_ref' IS DISTINCT FROM p_material->>'expediente_ref'
 THEN RAISE EXCEPTION 'AD3-100: contexto organizativo divergente' USING ERRCODE='42501'; END IF;
 IF c->>'audiencia_consumo' IS DISTINCT FROM 'vec_contratacion_temporal.catalogo_plantillas_documental.v1'
    OR c->>'operacion' <> ALL (ARRAY['contratacion_temporal.plantillas_documentos.documental_listar','contratacion_temporal.plantillas_documentos.documental_descargar'])
    OR c->>'operacion' IS NULL
    OR d->>'accion' IS DISTINCT FROM c->>'operacion'
    OR d->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'catalogo_plantillas_documental_ct'
    OR d->>'finalidad' IS DISTINCT FROM 'consultar_borradores_expediente'
    OR d->>'recurso_ref' IS NULL OR d->>'recurso_ref' NOT LIKE 'expediente:%'
    OR d->>'accion' IS DISTINCT FROM 'contratacion_temporal.plantillas_documentos.documental_'||(p_material->>'operacion')
    OR c->>'efecto_ref' IS DISTINCT FROM d->>'recurso_ref'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
    OR d->'campos_permitidos' IS DISTINCT FROM '["catalogo","catalogo_huella_sha256","contenido_json_sha256","procedencia_ref","revision","version"]'::jsonb
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'AD3-100: decisión de plantillas denegada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  'catalogo_plantillas_documental_ct',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE THEN RAISE EXCEPTION 'AD3-100: requiere consumo nuevo' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;

REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_plantillas_doc_ct_ambitos_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_plantillas_doc_ct_ambitos_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_contratacion_temporal_propietario;
REVOKE EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_plantillas_doc_ct_org_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM vec_contratacion_temporal_propietario;

DO $acl$
DECLARE f regprocedure:='vec_autorizacion_atestada_v3.registrar_y_consumir_plantillas_doc_ct_ambitos_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
BEGIN
 IF (SELECT proowner FROM pg_proc WHERE oid=f)<>'vec_autorizacion_atestada_v3_propietario'::regrole
    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS NOT TRUE
    OR NOT has_function_privilege('vec_contratacion_temporal_propietario',f,'EXECUTE')
    OR has_function_privilege('vec_contratacion_temporal_propietario',
      'vec_autorizacion_atestada_v3.registrar_y_consumir_plantillas_doc_ct_org_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,'EXECUTE')
    OR EXISTS (SELECT 1 FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x
      WHERE p.oid=f AND (x.grantee NOT IN (p.proowner,'vec_contratacion_temporal_propietario'::regrole)
       OR x.privilege_type<>'EXECUTE'))
 THEN RAISE EXCEPTION 'AD3-100: ACL incompatible' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
