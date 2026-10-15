\set ON_ERROR_STOP on
-- CT227. El ámbito autorizado de vínculo, consulta y descarte de oferta
-- es organización + unidad gestora del expediente. Centro y categoría
-- permanecen en los cánones y se cotejan con la versión del expediente.
-- CT220, CT223 y CT225 están ya instaladas; esta migración es prospectiva.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000227',0));
DO $pre$
DECLARE f oid; p pg_proc%ROWTYPE; r record;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario'
    OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
    OR to_regprocedure('vec_contratacion_temporal.preparar_contexto_estado_oferta_ct227(text,text)') IS NOT NULL
 THEN RAISE EXCEPTION 'CT227: preimagen o número incompatible' USING ERRCODE='55000'; END IF;
 FOR r IN SELECT * FROM (VALUES
  ('vec_contratacion_temporal.consumir_etapa_seleccion_oferta_ct220(text,jsonb,jsonb,jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
   '059b45d8c5118e9fb874a3eb7908deca78c4cec50d256dd58191f2acc610d404',
   '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario}'),
  ('vec_contratacion_temporal.descartar_seleccion_oferta_ct220(jsonb,text,jsonb,jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
   '0d1db657d7359a7b09cb0f6781e2af59f2fa060292de65fc536fcfa5b14e3d8e',
   '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario}'),
  ('vec_contratacion_temporal.consultar_estado_oferta_expediente_ct225(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
   '3f4c78128de156e3e8537f537114aeffd05bc276e1e099b5ecb3ae026ec7ec23',
   '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario}'),
  ('vec_contratacion_temporal.preparar_contexto_estado_oferta_ct225(text,text)',
   '89934f3ad1e74b6487ce3adb536aa9d8754950be697091f50cfd0a4d10222031',
   '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario}')
 ) AS x(firma,huella,acl) LOOP
 f:=to_regprocedure(r.firma);
 IF f IS NULL THEN RAISE EXCEPTION 'CT227: función ausente %',r.firma USING ERRCODE='55000'; END IF;
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=f;
 IF encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex') IS DISTINCT FROM r.huella
    OR p.proowner IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
    OR NOT p.prosecdef OR p.proacl::text IS DISTINCT FROM r.acl
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s','statement_timeout=15s']::text[]
 THEN RAISE EXCEPTION 'CT227: preimagen, dueño, configuración o ACL divergente %',r.firma USING ERRCODE='55000'; END IF;
 END LOOP;
END $pre$;

CREATE FUNCTION vec_contratacion_temporal.preparar_contexto_estado_oferta_ct227(
 p_organizacion text,p_expediente text)
RETURNS TABLE(version_actual bigint,categoria_ref text,centro_ref text,unidad_gestora_ref text)
LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
SET lock_timeout='2s' SET statement_timeout='15s' AS $f$
BEGIN
 -- CT225 valida sesión, organización y expediente antes de exponer contexto.
 IF current_user<>'vec_contratacion_temporal_propietario'
    OR session_user=current_user
    OR pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER') IS NOT TRUE
    OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER') IS TRUE
 THEN RAISE EXCEPTION 'CT227: contexto denegado' USING ERRCODE='42501'; END IF;
 RETURN QUERY
 SELECT c.version_actual,c.categoria_ref,c.centro_ref,
        v.agregado_json#>>'{asignacion,unidad_ref}'
 FROM vec_contratacion_temporal.preparar_contexto_estado_oferta_ct225(p_organizacion,p_expediente) c
 JOIN vec_contratacion_temporal.expediente_integral_actual a
   ON a.expediente_ref=p_expediente AND a.version=c.version_actual
 JOIN vec_contratacion_temporal.expediente_version_integral v
   ON v.expediente_ref=a.expediente_ref AND v.version=a.version
 WHERE v.agregado_json->>'organizacion_ref'=p_organizacion
   AND v.agregado_json#>>'{asignacion,unidad_ref}' ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$';
END $f$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.preparar_contexto_estado_oferta_ct227(text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.preparar_contexto_estado_oferta_ct227(text,text)
 TO vec_contratacion_temporal_ejecutor;

-- Reconstrucción acotada de consumir_etapa_seleccion_oferta_ct220 desde su cuerpo instalado.
DO $r_consumir_etapa_seleccion_oferta_ct220$
DECLARE f oid; actual text; anterior text; siguiente text;
BEGIN
 f:=to_regprocedure('vec_contratacion_temporal.consumir_etapa_seleccion_oferta_ct220(text,jsonb,jsonb,jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 actual:=pg_get_functiondef(f);
 IF encode(sha256(convert_to(actual,'UTF8')),'hex') IS DISTINCT FROM '059b45d8c5118e9fb874a3eb7908deca78c4cec50d256dd58191f2acc610d404'
 THEN RAISE EXCEPTION 'CT227: preimagen cambió consumir_etapa_seleccion_oferta_ct220' USING ERRCODE='55000'; END IF;
 anterior:=$ct227_0_0$ctx text;$ct227_0_0$; siguiente:=$ct227_0_0$ctx text;v_unidad_gestora text;$ct227_0_0$;
 IF length(actual)-length(replace(actual,anterior,'')) IS DISTINCT FROM length(anterior)
 THEN RAISE EXCEPTION 'CT227: marca divergente consumir_etapa_seleccion_oferta_ct220/0' USING ERRCODE='55000'; END IF;
 actual:=replace(actual,anterior,siguiente);
 anterior:=$ct227_0_1$h:=encode(sha256(convert_to(p_material,'UTF8')),'hex');$ct227_0_1$; siguiente:=$ct227_0_1$SELECT v.agregado_json#>>'{asignacion,unidad_ref}' INTO v_unidad_gestora
 FROM vec_contratacion_temporal.expediente_version_integral v
 WHERE v.expediente_ref=p_intencion->>'expediente_ref'
   AND v.version=(p_intencion->>'version_expediente')::bigint
   AND v.agregado_json->>'organizacion_ref'=p_intencion->>'organizacion_ref';
 IF v_unidad_gestora IS NULL OR v_unidad_gestora !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
 THEN RAISE EXCEPTION 'CT227: unidad gestora no acreditada' USING ERRCODE='42501'; END IF;
 h:=encode(sha256(convert_to(p_material,'UTF8')),'hex');$ct227_0_1$;
 IF length(actual)-length(replace(actual,anterior,'')) IS DISTINCT FROM length(anterior)
 THEN RAISE EXCEPTION 'CT227: marca divergente consumir_etapa_seleccion_oferta_ct220/1' USING ERRCODE='55000'; END IF;
 actual:=replace(actual,anterior,siguiente);
 anterior:=$ct227_0_2$ctx:=encode(sha256(convert_to('{"ambitos":{"categoria_ref":'||
   to_json(p_intencion->>'categoria_ref')::text||',"centro_ref":'||
   to_json(p_intencion->>'centro_ref')::text||',"organizacion_ref":'||
   to_json(p_intencion->>'organizacion_ref')::text||
   '},"atributos":{"material_sha256":"'||h||'"}}','UTF8')),'hex');$ct227_0_2$; siguiente:=$ct227_0_2$ctx:=encode(sha256(convert_to('{"ambitos":{"organizacion_ref":'||
   to_json(p_intencion->>'organizacion_ref')::text||',"unidad_ref":'||
   to_json(v_unidad_gestora)::text||
   '},"atributos":{"material_sha256":"'||h||'"}}','UTF8')),'hex');$ct227_0_2$;
 IF length(actual)-length(replace(actual,anterior,'')) IS DISTINCT FROM length(anterior)
 THEN RAISE EXCEPTION 'CT227: marca divergente consumir_etapa_seleccion_oferta_ct220/2' USING ERRCODE='55000'; END IF;
 actual:=replace(actual,anterior,siguiente);
 EXECUTE actual;
END $r_consumir_etapa_seleccion_oferta_ct220$;

-- Reconstrucción acotada de descartar_seleccion_oferta_ct220 desde su cuerpo instalado.
DO $r_descartar_seleccion_oferta_ct220$
DECLARE f oid; actual text; anterior text; siguiente text;
BEGIN
 f:=to_regprocedure('vec_contratacion_temporal.descartar_seleccion_oferta_ct220(jsonb,text,jsonb,jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 actual:=pg_get_functiondef(f);
 IF encode(sha256(convert_to(actual,'UTF8')),'hex') IS DISTINCT FROM '0d1db657d7359a7b09cb0f6781e2af59f2fa060292de65fc536fcfa5b14e3d8e'
 THEN RAISE EXCEPTION 'CT227: preimagen cambió descartar_seleccion_oferta_ct220' USING ERRCODE='55000'; END IF;
 anterior:=$ct227_1_0$ctx text;$ct227_1_0$; siguiente:=$ct227_1_0$ctx text;v_unidad_gestora text;$ct227_1_0$;
 IF length(actual)-length(replace(actual,anterior,'')) IS DISTINCT FROM length(anterior)
 THEN RAISE EXCEPTION 'CT227: marca divergente descartar_seleccion_oferta_ct220/0' USING ERRCODE='55000'; END IF;
 actual:=replace(actual,anterior,siguiente);
 anterior:=$ct227_1_1$h:=encode(sha256(convert_to(p_material,'UTF8')),'hex');$ct227_1_1$; siguiente:=$ct227_1_1$SELECT v.agregado_json#>>'{asignacion,unidad_ref}' INTO v_unidad_gestora
 FROM vec_contratacion_temporal.expediente_version_integral v
 WHERE v.expediente_ref=p_intencion->>'expediente_ref'
   AND v.version=(p_intencion->>'version_expediente')::bigint
   AND v.agregado_json->>'organizacion_ref'=p_intencion->>'organizacion_ref';
 IF v_unidad_gestora IS NULL OR v_unidad_gestora !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
 THEN RAISE EXCEPTION 'CT227: unidad gestora no acreditada' USING ERRCODE='42501'; END IF;
 h:=encode(sha256(convert_to(p_material,'UTF8')),'hex');$ct227_1_1$;
 IF length(actual)-length(replace(actual,anterior,'')) IS DISTINCT FROM length(anterior)
 THEN RAISE EXCEPTION 'CT227: marca divergente descartar_seleccion_oferta_ct220/1' USING ERRCODE='55000'; END IF;
 actual:=replace(actual,anterior,siguiente);
 anterior:=$ct227_1_2$ctx:=encode(sha256(convert_to('{"ambitos":{"categoria_ref":'||
   to_json(p_intencion->>'categoria_ref')::text||',"centro_ref":'||
   to_json(p_intencion->>'centro_ref')::text||',"organizacion_ref":'||
   to_json(p_intencion->>'organizacion_ref')::text||
   '},"atributos":{"material_sha256":"'||h||'"}}','UTF8')),'hex');$ct227_1_2$; siguiente:=$ct227_1_2$ctx:=encode(sha256(convert_to('{"ambitos":{"organizacion_ref":'||
   to_json(p_intencion->>'organizacion_ref')::text||',"unidad_ref":'||
   to_json(v_unidad_gestora)::text||
   '},"atributos":{"material_sha256":"'||h||'"}}','UTF8')),'hex');$ct227_1_2$;
 IF length(actual)-length(replace(actual,anterior,'')) IS DISTINCT FROM length(anterior)
 THEN RAISE EXCEPTION 'CT227: marca divergente descartar_seleccion_oferta_ct220/2' USING ERRCODE='55000'; END IF;
 actual:=replace(actual,anterior,siguiente);
 EXECUTE actual;
END $r_descartar_seleccion_oferta_ct220$;

-- Reconstrucción acotada de consultar_estado_oferta_expediente_ct225 desde su cuerpo instalado.
DO $r_consultar_estado_oferta_expediente_ct225$
DECLARE f oid; actual text; anterior text; siguiente text;
BEGIN
 f:=to_regprocedure('vec_contratacion_temporal.consultar_estado_oferta_expediente_ct225(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 actual:=pg_get_functiondef(f);
 IF encode(sha256(convert_to(actual,'UTF8')),'hex') IS DISTINCT FROM '3f4c78128de156e3e8537f537114aeffd05bc276e1e099b5ecb3ae026ec7ec23'
 THEN RAISE EXCEPTION 'CT227: preimagen cambió consultar_estado_oferta_expediente_ct225' USING ERRCODE='55000'; END IF;
 anterior:=$ct227_2_0$h:=encode(sha256(convert_to(p_material,'UTF8')),'hex');$ct227_2_0$; siguiente:=$ct227_2_0$IF c.agregado_json#>>'{asignacion,unidad_ref}' IS NULL
    OR c.agregado_json#>>'{asignacion,unidad_ref}' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
 THEN RAISE EXCEPTION 'CT227: unidad gestora no acreditada' USING ERRCODE='42501'; END IF;
 h:=encode(sha256(convert_to(p_material,'UTF8')),'hex');$ct227_2_0$;
 IF length(actual)-length(replace(actual,anterior,'')) IS DISTINCT FROM length(anterior)
 THEN RAISE EXCEPTION 'CT227: marca divergente consultar_estado_oferta_expediente_ct225/0' USING ERRCODE='55000'; END IF;
 actual:=replace(actual,anterior,siguiente);
 anterior:=$ct227_2_1$ctx:=encode(sha256(convert_to('{"ambitos":{"categoria_ref":'||
   to_json(m->>'categoria_ref')::text||',"centro_ref":'||
   to_json(m->>'centro_ref')::text||',"organizacion_ref":'||
   to_json(m->>'organizacion_ref')::text||
   '},"atributos":{"material_sha256":"'||h||'"}}','UTF8')),'hex');$ct227_2_1$; siguiente:=$ct227_2_1$ctx:=encode(sha256(convert_to('{"ambitos":{"organizacion_ref":'||
   to_json(m->>'organizacion_ref')::text||',"unidad_ref":'||
   to_json(c.agregado_json#>>'{asignacion,unidad_ref}')::text||
   '},"atributos":{"material_sha256":"'||h||'"}}','UTF8')),'hex');$ct227_2_1$;
 IF length(actual)-length(replace(actual,anterior,'')) IS DISTINCT FROM length(anterior)
 THEN RAISE EXCEPTION 'CT227: marca divergente consultar_estado_oferta_expediente_ct225/1' USING ERRCODE='55000'; END IF;
 actual:=replace(actual,anterior,siguiente);
 EXECUTE actual;
END $r_consultar_estado_oferta_expediente_ct225$;

DO $post$
DECLARE f oid; p pg_proc%ROWTYPE; r record;
BEGIN
 FOR r IN SELECT * FROM (VALUES
  ('vec_contratacion_temporal.consumir_etapa_seleccion_oferta_ct220(text,jsonb,jsonb,jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','6a322faf1d1b8519ac8f5326f491589142b4b51334b1d4c9765199e3c582ef3a',
   '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario}'),
  ('vec_contratacion_temporal.descartar_seleccion_oferta_ct220(jsonb,text,jsonb,jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','49da40eac4a0f44a8a9b7c096446b09a74099d3bc3f908be5195a8af0b48b001',
   '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario}'),
  ('vec_contratacion_temporal.consultar_estado_oferta_expediente_ct225(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','1ee860093ab0ce7e71d4b304a0f0ed6f2eae31f84a4e8952260a088aea50a35f',
   '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario}')
 ) AS x(firma,huella,acl) LOOP
 f:=to_regprocedure(r.firma);
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=f;
 IF encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex') IS DISTINCT FROM r.huella
    OR p.proowner IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
    OR NOT p.prosecdef OR p.proacl::text IS DISTINCT FROM r.acl
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s','statement_timeout=15s']::text[]
 THEN RAISE EXCEPTION 'CT227: postimagen divergente %',r.firma USING ERRCODE='55000'; END IF;
 END LOOP;
 f:=to_regprocedure('vec_contratacion_temporal.preparar_contexto_estado_oferta_ct227(text,text)');
 IF f IS NULL THEN RAISE EXCEPTION 'CT227: preparación ausente' USING ERRCODE='55000'; END IF;
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=f;
 IF p.proowner IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
    OR NOT p.prosecdef OR p.provolatile<>'s'
    OR p.proacl::text IS DISTINCT FROM
       '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario}'
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s','statement_timeout=15s']::text[]
 THEN RAISE EXCEPTION 'CT227: preparación o ACL divergente' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
