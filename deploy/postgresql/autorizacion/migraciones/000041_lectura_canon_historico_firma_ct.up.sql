\set ON_ERROR_STOP on
-- AUT41: lectura privada de los bytes históricos AUT32 para una consulta CT
-- autorizada ahora. El helper AD178 pendiente valida el consumo de lectura.
BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL TimeZone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion:migracion:000041',0));
DO $pre$
DECLARE helper regprocedure;
BEGIN
 helper:=pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.comprobar_consumo_recuperacion_firmas_ct_v2(bytea,jsonb)');
 IF current_user<>'vec_autorizacion_propietario'
  OR pg_catalog.to_regprocedure('vec_autorizacion.recuperar_canon_historico_firmante_ct_v1(jsonb,bytea,jsonb)') IS NOT NULL
  OR helper IS NULL
  OR NOT pg_catalog.has_schema_privilege('vec_autorizacion_propietario','vec_autorizacion_atestada_v3','USAGE')
  OR pg_catalog.to_regrole('vec_contratacion_temporal_propietario') IS NULL
  OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_class c
    WHERE c.oid='vec_autorizacion.evidencia_competencia_firmante_ct_v1'::pg_catalog.regclass
      AND c.relowner=current_user::pg_catalog.regrole AND c.relrowsecurity AND c.relforcerowsecurity)
  OR EXISTS(SELECT 1 FROM pg_catalog.pg_roles r
    WHERE r.rolname IN ('vec_autorizacion_propietario','vec_contratacion_temporal_propietario')
      AND (r.rolcanlogin OR r.rolsuper OR r.rolbypassrls)) THEN
  RAISE EXCEPTION 'AUT41: preimagen incompatible' USING ERRCODE='55000'; END IF;
 IF NOT pg_catalog.has_function_privilege('vec_autorizacion_propietario',helper,'EXECUTE') THEN
  RAISE EXCEPTION 'AUT41: comprobador sin permiso propietario' USING ERRCODE='55000'; END IF;
END $pre$;
CREATE FUNCTION vec_autorizacion.recuperar_canon_historico_firmante_ct_v1(
 p_selector_historico jsonb,p_material_consulta bytea,p_consumo_actual jsonb
) RETURNS bytea LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog SET row_security='on' SET lock_timeout='2s' SET statement_timeout='15s' SET TimeZone='UTC'
AS $f$
DECLARE prueba jsonb; fila vec_autorizacion.evidencia_competencia_firmante_ct_v1%ROWTYPE;
 historico jsonb; ahora timestamptz(6);
BEGIN
 IF pg_catalog.current_setting('transaction_isolation')<>'serializable'
  OR pg_catalog.current_setting('transaction_read_only')<>'off'
  OR pg_catalog.pg_is_in_recovery()
  OR pg_catalog.jsonb_typeof(p_selector_historico) IS DISTINCT FROM 'object'
  OR (SELECT count(*) FROM pg_catalog.jsonb_object_keys(p_selector_historico))<>9
  OR NOT (p_selector_historico ?& ARRAY['evidencia_ref','canon_sha256','efecto_original_ref',
    'organizacion_ref','unidad_ref','expediente_ref','documento_ref',
    'consumo_original_decision_ref','consumo_original_huella_sha256'])
  OR p_material_consulta IS NULL OR pg_catalog.octet_length(p_material_consulta) NOT BETWEEN 2 AND 65536
  OR pg_catalog.jsonb_typeof(p_consumo_actual) IS DISTINCT FROM 'object' THEN
  RAISE EXCEPTION 'AUT41: selector o consumo no disponible' USING ERRCODE='42501'; END IF;
 IF EXISTS(SELECT 1 FROM pg_catalog.jsonb_object_keys(p_selector_historico) AS keys(nombre)
   WHERE pg_catalog.jsonb_typeof(p_selector_historico->keys.nombre) IS DISTINCT FROM 'string')
  OR pg_catalog.jsonb_typeof(p_selector_historico->'evidencia_ref') IS DISTINCT FROM 'string'
  OR (p_selector_historico->>'evidencia_ref' ~ '^evidencia:competencia-firmante-ct:[0-9a-f]{64}$') IS NOT TRUE
  OR pg_catalog.jsonb_typeof(p_selector_historico->'canon_sha256') IS DISTINCT FROM 'string'
  OR (p_selector_historico->>'canon_sha256' ~ '^[0-9a-f]{64}$') IS NOT TRUE
  OR (p_selector_historico->>'consumo_original_huella_sha256' ~ '^[0-9a-f]{64}$') IS NOT TRUE THEN
  RAISE EXCEPTION 'AUT41: selector histórico inválido' USING ERRCODE='42501'; END IF;
 IF vec_autorizacion.texto_positivo_valido(p_selector_historico->>'efecto_original_ref',512) IS NOT TRUE
  OR vec_autorizacion.texto_positivo_valido(p_selector_historico->>'organizacion_ref',512) IS NOT TRUE
  OR vec_autorizacion.texto_positivo_valido(p_selector_historico->>'unidad_ref',512) IS NOT TRUE
  OR vec_autorizacion.texto_positivo_valido(p_selector_historico->>'expediente_ref',512) IS NOT TRUE
  OR vec_autorizacion.texto_positivo_valido(p_selector_historico->>'documento_ref',512) IS NOT TRUE
  OR vec_autorizacion.texto_positivo_valido(p_selector_historico->>'consumo_original_decision_ref',512) IS NOT TRUE THEN
  RAISE EXCEPTION 'AUT41: referencias históricas inválidas' USING ERRCODE='42501'; END IF;
 -- AD178 relee consumo, decisión y auditoría del consultante en la misma TX.
 -- No aporta ni reinterpreta la competencia del firmante original.
 SELECT vec_autorizacion_atestada_v3.comprobar_consumo_recuperacion_firmas_ct_v2(
  p_material_consulta,p_consumo_actual) INTO STRICT prueba;
 ahora:=pg_catalog.clock_timestamp();
 IF pg_catalog.jsonb_typeof(prueba) IS DISTINCT FROM 'object'
  OR (SELECT count(*) FROM pg_catalog.jsonb_object_keys(prueba))<>8
  OR NOT (prueba ?& ARRAY['organizacion_ref','expediente_ref','documento','version_expediente',
    'decision_ref','consumo_huella_sha256','auditoria_ref','decision_valida_hasta'])
  OR pg_catalog.jsonb_typeof(prueba->'organizacion_ref') IS DISTINCT FROM 'string'
  OR pg_catalog.jsonb_typeof(prueba->'expediente_ref') IS DISTINCT FROM 'string'
  OR pg_catalog.jsonb_typeof(prueba->'documento') IS DISTINCT FROM 'string'
  OR pg_catalog.jsonb_typeof(prueba->'version_expediente') IS DISTINCT FROM 'number'
  OR pg_catalog.jsonb_typeof(prueba->'decision_ref') IS DISTINCT FROM 'string'
  OR pg_catalog.jsonb_typeof(prueba->'consumo_huella_sha256') IS DISTINCT FROM 'string'
  OR pg_catalog.jsonb_typeof(prueba->'auditoria_ref') IS DISTINCT FROM 'string'
  OR pg_catalog.jsonb_typeof(prueba->'decision_valida_hasta') IS DISTINCT FROM 'string'
  OR prueba->>'organizacion_ref' IS DISTINCT FROM p_selector_historico->>'organizacion_ref'
  OR prueba->>'expediente_ref' IS DISTINCT FROM p_selector_historico->>'expediente_ref'
  OR (prueba->>'documento' ~ '^[a-z][a-z0-9_]{1,63}$') IS NOT TRUE
  OR (prueba->>'version_expediente' ~ '^[1-9][0-9]{0,15}$') IS NOT TRUE
  OR vec_autorizacion.texto_positivo_valido(prueba->>'decision_ref',512) IS NOT TRUE
  OR (prueba->>'consumo_huella_sha256' ~ '^[0-9a-f]{64}$') IS NOT TRUE
  OR vec_autorizacion.texto_positivo_valido(prueba->>'auditoria_ref',512) IS NOT TRUE
  OR prueba->>'decision_ref' IS DISTINCT FROM p_consumo_actual->>'decision_ref'
  OR prueba->>'consumo_huella_sha256' IS DISTINCT FROM p_consumo_actual->>'consumo_huella_sha256'
  OR prueba->>'auditoria_ref' IS DISTINCT FROM p_consumo_actual->>'auditoria_ref'
  OR prueba->>'decision_valida_hasta' IS NULL
  OR (prueba->>'decision_valida_hasta')::timestamptz<=ahora THEN
  RAISE EXCEPTION 'AUT41: consulta actual no acreditada' USING ERRCODE='42501'; END IF;
 IF (prueba->>'version_expediente')::numeric>9007199254740991 THEN
  RAISE EXCEPTION 'AUT41: versión de consulta no admisible' USING ERRCODE='42501'; END IF;
 SELECT * INTO fila FROM vec_autorizacion.evidencia_competencia_firmante_ct_v1 x
  WHERE x.evidencia_ref=p_selector_historico->>'evidencia_ref'
    AND x.huella_sha256=p_selector_historico->>'canon_sha256' FOR SHARE;
 IF NOT FOUND THEN
  RAISE EXCEPTION 'AUT41: evidencia histórica no disponible' USING ERRCODE='42501'; END IF;
 IF fila.efecto_ref IS DISTINCT FROM p_selector_historico->>'efecto_original_ref'
  OR fila.organizacion_ref IS DISTINCT FROM p_selector_historico->>'organizacion_ref'
  OR fila.unidad_ref IS DISTINCT FROM p_selector_historico->>'unidad_ref'
  OR fila.expediente_ref IS DISTINCT FROM p_selector_historico->>'expediente_ref'
  OR fila.documento_ref IS DISTINCT FROM p_selector_historico->>'documento_ref'
  OR fila.consumo_decision_ref IS DISTINCT FROM p_selector_historico->>'consumo_original_decision_ref'
  OR fila.consumo_huella_sha256 IS DISTINCT FROM p_selector_historico->>'consumo_original_huella_sha256'
  OR fila.esquema IS DISTINCT FROM 'vec.competencia-firmante.historica.v1'
  OR pg_catalog.octet_length(fila.canonico) NOT BETWEEN 512 AND 32768
  OR fila.huella_sha256 IS DISTINCT FROM pg_catalog.encode(pg_catalog.sha256(fila.canonico),'hex')
  OR fila.evidencia_ref IS DISTINCT FROM 'evidencia:competencia-firmante-ct:'||
      pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(fila.efecto_ref,'UTF8')||fila.canonico),'hex') THEN
  RAISE EXCEPTION 'AUT41: evidencia histórica incoherente' USING ERRCODE='55000'; END IF;
 BEGIN historico:=pg_catalog.convert_from(fila.canonico,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN
  RAISE EXCEPTION 'AUT41: canon original no disponible' USING ERRCODE='55000'; END;
 IF historico->>'esquema' IS DISTINCT FROM fila.esquema
  OR historico#>>'{recurso,organizacion_ref}' IS DISTINCT FROM fila.organizacion_ref
  OR historico#>>'{recurso,unidad_ref}' IS DISTINCT FROM fila.unidad_ref
  OR historico#>>'{recurso,expediente_ref}' IS DISTINCT FROM fila.expediente_ref
  OR historico#>>'{recurso,documento_ref}' IS DISTINCT FROM fila.documento_ref THEN
  RAISE EXCEPTION 'AUT41: canon original incoherente' USING ERRCODE='55000'; END IF;
 IF (prueba->>'decision_valida_hasta')::timestamptz<=pg_catalog.clock_timestamp() THEN
  RAISE EXCEPTION 'AUT41: consulta caducada durante lectura' USING ERRCODE='42501'; END IF;
 RETURN fila.canonico;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.recuperar_canon_historico_firmante_ct_v1(jsonb,bytea,jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion.recuperar_canon_historico_firmante_ct_v1(jsonb,bytea,jsonb)
 TO vec_contratacion_temporal_propietario;
DO $acl$
DECLARE f regprocedure:='vec_autorizacion.recuperar_canon_historico_firmante_ct_v1(jsonb,bytea,jsonb)'::regprocedure;
 x record; owner_id oid:='vec_autorizacion_propietario'::regrole;
 ct_id oid:='vec_contratacion_temporal_propietario'::regrole;
BEGIN
 FOR x IN SELECT DISTINCT a.grantee FROM pg_catalog.pg_proc p CROSS JOIN LATERAL
  pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
  WHERE p.oid=f AND a.grantee NOT IN(owner_id,ct_id) LOOP
  EXECUTE pg_catalog.format('REVOKE ALL ON FUNCTION %s FROM %s',f::text,
   CASE WHEN x.grantee=0 THEN 'PUBLIC' ELSE pg_catalog.quote_ident(pg_catalog.pg_get_userbyid(x.grantee)) END);
 END LOOP;
 IF (SELECT p.proowner FROM pg_catalog.pg_proc p WHERE p.oid=f) IS DISTINCT FROM owner_id
  OR (SELECT count(*) FROM pg_catalog.pg_proc p CROSS JOIN LATERAL
   pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
   WHERE p.oid=f)<>2
  OR EXISTS(SELECT 1 FROM pg_catalog.pg_proc p CROSS JOIN LATERAL
   pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
   WHERE p.oid=f AND (a.grantee NOT IN(owner_id,ct_id) OR a.privilege_type<>'EXECUTE'
     OR (a.grantee=ct_id AND (a.grantor<>owner_id OR a.is_grantable))))
  OR NOT pg_catalog.has_function_privilege(ct_id,f,'EXECUTE') THEN
  RAISE EXCEPTION 'AUT41: ACL de lectura incompatible' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
