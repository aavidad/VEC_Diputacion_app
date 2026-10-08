\set ON_ERROR_STOP on
-- CT186: consulta y recuperación R5 V2 con la huella de AD210 (ámbitos de la
-- asignación de quien consulta) y, si el material trae UnidadRef, la unidad
-- ligada al paso del plan publicado (CC10). Opción A del 06/10.
--
-- Sucesoras con sustituciones exactas sobre las definiciones medidas:
--   * consultar_firmas_r5_atestadas_v3 = CT172 v2;
--   * recuperar_firmas_r5_atestadas_v3 = CT175 v2.
-- En las dos: el nombre; UnidadRef deja de ser obligatoriamente null (texto
-- de referencia, sólo vía VEC); la huella sale de AD210 y, con UnidadRef, CC10
-- debe confirmar que el plan publicado tiene ese paso en esa unidad. Sin
-- UnidadRef se comportan como las v2. El ejecutor CT pasa de las v2 a las v3.
-- Requiere AD210 y CC10. Una sola vez; sin DOWN.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_contratacion_temporal:migracion:000186',0));
DO $pre$
BEGIN
 IF pg_catalog.current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
 THEN RAISE EXCEPTION 'CT186: PARO clave=migrador_PG actual=no_acreditado esperado=superusuario_PG18' USING ERRCODE='42501'; END IF;
 IF pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.huella_recurso_consulta_firmas_r5_ct_v1(text,bytea)') IS NULL
 OR pg_catalog.to_regprocedure('vec_catalogos_configurables.paso_plan_firma_con_unidad_v1(text,text,integer,text,text)') IS NULL
 OR NOT pg_catalog.has_function_privilege('vec_contratacion_temporal_propietario',
     'vec_autorizacion_atestada_v3.huella_recurso_consulta_firmas_r5_ct_v1(text,bytea)','EXECUTE')
 OR NOT pg_catalog.has_function_privilege('vec_contratacion_temporal_propietario',
     'vec_catalogos_configurables.paso_plan_firma_con_unidad_v1(text,text,integer,text,text)','EXECUTE')
 THEN RAISE EXCEPTION 'CT186: PARO clave=AD210_CC10 actual=ausente esperado=instaladas' USING ERRCODE='55000'; END IF;
 IF pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.pg_get_functiondef(pg_catalog.to_regprocedure(
     'vec_contratacion_temporal.consultar_firmas_r5_atestadas_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')),'UTF8')),'hex')
     IS DISTINCT FROM '845d0f2c939a2127fb2980fb1ec22486a43cc66cee4f39a31ee891bb2ae93b71'
 OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.pg_get_functiondef(pg_catalog.to_regprocedure(
     'vec_contratacion_temporal.recuperar_firmas_r5_atestadas_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')),'UTF8')),'hex')
     IS DISTINCT FROM '990118080d6909dccdc4a5850cdf42ad627ad4c45b8b12e093fa1ade7b3e87e3'
 THEN RAISE EXCEPTION 'CT186: PARO clave=CT172_CT175 actual=distinto esperado=definiciones_medidas' USING ERRCODE='55000'; END IF;
 IF pg_catalog.to_regprocedure('vec_contratacion_temporal.consultar_firmas_r5_atestadas_v3(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 OR pg_catalog.to_regprocedure('vec_contratacion_temporal.recuperar_firmas_r5_atestadas_v3(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 THEN RAISE EXCEPTION 'CT186: PARO clave=preimagen actual=v3_presente esperado=ausente' USING ERRCODE='55000'; END IF;
END $pre$;

-- Cada sustitución se comprueba: la marca aparece exactamente una vez.
CREATE FUNCTION pg_temp.sustituir_ct186(t text,viejo text,nuevo text) RETURNS text LANGUAGE plpgsql AS $f$
BEGIN
 IF (pg_catalog.length(t)-pg_catalog.length(pg_catalog.replace(t,viejo,'')))/pg_catalog.length(viejo)<>1 THEN
  RAISE EXCEPTION 'CT186: PARO clave=marca actual=no_unica esperado=una_vez detalle=%',pg_catalog.left(viejo,60) USING ERRCODE='55000'; END IF;
 RETURN pg_catalog.replace(t,viejo,nuevo);
END $f$;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
DO $v3$
DECLARE t text; nombre text;
 viejo_unidad text:=E'  OR s->''UnidadRef'' IS DISTINCT FROM ''null''::jsonb\n';
 nuevo_unidad text:=E'  OR (s->''UnidadRef'' IS DISTINCT FROM ''null''::jsonb AND (jsonb_typeof(s->''UnidadRef'') IS DISTINCT FROM ''string''\n'
  ||E'   OR s->>''UnidadRef'' !~ ''^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'' OR s->>''Via'' IS DISTINCT FROM ''certificado_vec''))\n';
 viejo_ctx text:=E' contexto_h:=encode(sha256(convert_to(''{"ambitos":{"organizacion_ref":"''||(s->>''OrganizacionRef'')||\n  ''"},"atributos":{"material_sha256":"''||h||''"}}'',''UTF8'')),''hex'');';
 nuevo_ctx text:=E' contexto_h:=vec_autorizacion_atestada_v3.huella_recurso_consulta_firmas_r5_ct_v1(p_solicitud,p_decision);\n'
  ||E' IF s->''UnidadRef'' IS DISTINCT FROM ''null''::jsonb AND vec_catalogos_configurables.paso_plan_firma_con_unidad_v1(\n'
  ||E'   s->>''CatalogoHuella'',s->>''Documento'',(s->>''PasoOrden'')::integer,s->>''OrganizacionRef'',s->>''UnidadRef'') IS NOT TRUE THEN\n'
  ||E'  RAISE EXCEPTION ''lectura V2 unidad sin paso en el plan'' USING ERRCODE=''42501''; END IF;';
BEGIN
 FOREACH nombre IN ARRAY ARRAY['consultar_firmas_r5_atestadas','recuperar_firmas_r5_atestadas'] LOOP
  t:=pg_catalog.pg_get_functiondef(('vec_contratacion_temporal.'||nombre||'_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')::regprocedure);
  t:=pg_temp.sustituir_ct186(t,'vec_contratacion_temporal.'||nombre||'_v2(','vec_contratacion_temporal.'||nombre||'_v3(');
  t:=pg_temp.sustituir_ct186(t,viejo_unidad,nuevo_unidad);
  t:=pg_temp.sustituir_ct186(t,viejo_ctx,nuevo_ctx);
  EXECUTE t;
 END LOOP;
END $v3$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.consultar_firmas_r5_atestadas_v3(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_contratacion_temporal.recuperar_firmas_r5_atestadas_v3(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.consultar_firmas_r5_atestadas_v3(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_contratacion_temporal.recuperar_firmas_r5_atestadas_v3(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_contratacion_temporal_ejecutor;
-- Las v2 calculan la huella sólo con la organización: dejan de ser ejecutables.
REVOKE EXECUTE ON FUNCTION vec_contratacion_temporal.consultar_firmas_r5_atestadas_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_contratacion_temporal.recuperar_firmas_r5_atestadas_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM vec_contratacion_temporal_ejecutor;
RESET ROLE;

DO $post$
DECLARE nombre text; v2 regprocedure; v3 regprocedure; def text;
BEGIN
 FOREACH nombre IN ARRAY ARRAY['consultar_firmas_r5_atestadas','recuperar_firmas_r5_atestadas'] LOOP
  v2:=('vec_contratacion_temporal.'||nombre||'_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')::regprocedure;
  v3:=('vec_contratacion_temporal.'||nombre||'_v3(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')::regprocedure;
  def:=pg_catalog.pg_get_functiondef(v3);
  IF (SELECT row(proowner,prosecdef,proconfig,provolatile,proparallel) FROM pg_catalog.pg_proc WHERE oid=v3)
     IS DISTINCT FROM (SELECT row(proowner,prosecdef,proconfig,provolatile,proparallel) FROM pg_catalog.pg_proc WHERE oid=v2)
  OR pg_catalog.strpos(def,'huella_recurso_consulta_firmas_r5_ct_v1(p_solicitud,p_decision)')=0
  OR pg_catalog.strpos(def,'paso_plan_firma_con_unidad_v1(')=0
  OR (SELECT proacl FROM pg_catalog.pg_proc WHERE oid=v3) IS DISTINCT FROM
     ARRAY['vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario',
           'vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario']::aclitem[]
  OR pg_catalog.has_function_privilege('vec_contratacion_temporal_ejecutor',v2,'EXECUTE')
  THEN RAISE EXCEPTION 'CT186: PARO clave=postimagen actual=divergente esperado=v3_para_ejecutor_y_v2_cerradas' USING ERRCODE='55000'; END IF;
 END LOOP;
END $post$;
COMMIT;
