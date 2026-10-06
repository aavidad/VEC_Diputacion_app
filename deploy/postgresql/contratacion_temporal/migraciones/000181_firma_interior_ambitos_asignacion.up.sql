\set ON_ERROR_STOP on
-- CT181: registro de firma V2 con la huella interior de AD206 (ámbitos de la
-- asignación de quien firma: organización y, si la tiene, unidad).
--
-- Sucesores con sustituciones exactas sobre las definiciones medidas:
--   * registrar_firma_verificada_v3 = CT172 v2 con la huella de AD206 y la
--     llamada a AD170 v3;
--   * registrar_firma_con_plan_v3 = CT176 v2 con la huella interior de AD206
--     y la llamada a registrar_firma_verificada_v3. La huella exterior (plan)
--     sigue sólo con organización: es el corte siguiente.
-- El ejecutor CT pasa de las v2 a las v3. Requiere AD206. Una sola vez; sin DOWN.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_contratacion_temporal:migracion:000181',0));
DO $pre$
BEGIN
 IF pg_catalog.current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
 THEN RAISE EXCEPTION 'CT181: PARO clave=migrador_PG actual=no_acreditado esperado=superusuario_PG18' USING ERRCODE='42501'; END IF;
 IF pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.huella_recurso_firma_interior_ct_v1(text,bytea,bytea)') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_firma_descriptor_ct_v3_atestada(text,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 OR NOT pg_catalog.has_function_privilege('vec_contratacion_temporal_propietario',
     'vec_autorizacion_atestada_v3.huella_recurso_firma_interior_ct_v1(text,bytea,bytea)','EXECUTE')
 THEN RAISE EXCEPTION 'CT181: PARO clave=AD206 actual=ausente esperado=instalada' USING ERRCODE='55000'; END IF;
 IF pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.pg_get_functiondef(
     'vec_contratacion_temporal.registrar_firma_verificada_v2(text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea)'::regprocedure),'UTF8')),'hex')
     IS DISTINCT FROM 'bad0e55668f9b955ea0df127b27cfc4ce3d6ef6644a08fdbd08c06e9d3f26269'
 OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.pg_get_functiondef(
     'vec_contratacion_temporal.registrar_firma_con_plan_v2(text,timestamptz,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure),'UTF8')),'hex')
     IS DISTINCT FROM '73756b891974b300409e69c7e0e344407770cd4156b0ee4e24d8ada8dda139cf'
 THEN RAISE EXCEPTION 'CT181: PARO clave=CT172_CT176 actual=distinto esperado=definiciones_medidas' USING ERRCODE='55000'; END IF;
 IF pg_catalog.to_regprocedure('vec_contratacion_temporal.registrar_firma_verificada_v3(text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea)') IS NOT NULL
 OR pg_catalog.to_regprocedure('vec_contratacion_temporal.registrar_firma_con_plan_v3(text,timestamptz,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 THEN RAISE EXCEPTION 'CT181: PARO clave=preimagen actual=v3_presente esperado=ausente' USING ERRCODE='55000'; END IF;
END $pre$;

-- Cada sustitución se comprueba: la marca aparece exactamente una vez.
CREATE FUNCTION pg_temp.sustituir_ct181(t text,viejo text,nuevo text) RETURNS text LANGUAGE plpgsql AS $f$
BEGIN
 IF (pg_catalog.length(t)-pg_catalog.length(pg_catalog.replace(t,viejo,'')))/pg_catalog.length(viejo)<>1 THEN
  RAISE EXCEPTION 'CT181: PARO clave=marca actual=no_unica esperado=una_vez detalle=%',pg_catalog.left(viejo,60) USING ERRCODE='55000'; END IF;
 RETURN pg_catalog.replace(t,viejo,nuevo);
END $f$;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
DO $v3$
DECLARE t text;
BEGIN
 t:=pg_catalog.pg_get_functiondef('vec_contratacion_temporal.registrar_firma_verificada_v2(text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea)'::regprocedure);
 t:=pg_temp.sustituir_ct181(t,'vec_contratacion_temporal.registrar_firma_verificada_v2(','vec_contratacion_temporal.registrar_firma_verificada_v3(');
 t:=pg_temp.sustituir_ct181(t,'registrar_y_consumir_firma_descriptor_ct_v2_atestada(','registrar_y_consumir_firma_descriptor_ct_v3_atestada(');
 t:=pg_temp.sustituir_ct181(t,
  E' contexto_h := encode(sha256(convert_to(''{"ambitos":{"organizacion_ref":"''||(s->>''OrganizacionRef'')||\n   ''"},"atributos":{"descriptor_firma_sha256":"''||descriptor_h||''","material_sha256":"''||h||''"}}'',''UTF8'')),''hex'');',
  ' contexto_h := vec_autorizacion_atestada_v3.huella_recurso_firma_interior_ct_v1(p_solicitud,p_descriptor_nominal,p_decision);');
 EXECUTE t;
 t:=pg_catalog.pg_get_functiondef('vec_contratacion_temporal.registrar_firma_con_plan_v2(text,timestamptz,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure);
 t:=pg_temp.sustituir_ct181(t,'vec_contratacion_temporal.registrar_firma_con_plan_v2(','vec_contratacion_temporal.registrar_firma_con_plan_v3(');
 t:=pg_temp.sustituir_ct181(t,'vec_contratacion_temporal.registrar_firma_verificada_v2(','vec_contratacion_temporal.registrar_firma_verificada_v3(');
 t:=pg_temp.sustituir_ct181(t,
  E' contexto_i:=encode(sha256(convert_to(''{"ambitos":{"organizacion_ref":"''||(s->>''OrganizacionRef'')||\n  ''"},"atributos":{"descriptor_firma_sha256":"''||dh||''","material_sha256":"''||h||''"}}'',''UTF8'')),''hex'');',
  ' contexto_i:=vec_autorizacion_atestada_v3.huella_recurso_firma_interior_ct_v1(p_solicitud,descriptor_exacto,p_dec_i);');
 EXECUTE t;
END $v3$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.registrar_firma_verificada_v3(text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.registrar_firma_con_plan_v3(text,timestamptz,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
-- La v3 de CT172 no se concede al ejecutor: sólo se llega a ella por CT176
-- v3, que liga tipo, acción, finalidad, cargo y enlace al plan publicado.
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.registrar_firma_con_plan_v3(text,timestamptz,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_contratacion_temporal_ejecutor;
-- Las v2 calculan la huella sólo con la organización: dejan de ser ejecutables.
REVOKE EXECUTE ON FUNCTION vec_contratacion_temporal.registrar_firma_verificada_v2(text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea) FROM vec_contratacion_temporal_ejecutor;
REVOKE EXECUTE ON FUNCTION vec_contratacion_temporal.registrar_firma_con_plan_v2(text,timestamptz,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM vec_contratacion_temporal_ejecutor;
RESET ROLE;

-- Con las v2 de CT cerradas, la v2 de AD170 se queda sin llamador: también se
-- cierra para el propietario CT (la concedió el propietario AD en AD170).
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
REVOKE EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_firma_descriptor_ct_v2_atestada(text,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM vec_contratacion_temporal_propietario;
RESET ROLE;

DO $post$
DECLARE par record;
 acl_plan aclitem[]:=ARRAY['vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario',
  'vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario']::aclitem[];
 acl_directa aclitem[]:=ARRAY['vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario']::aclitem[];
BEGIN
 FOR par IN SELECT * FROM (VALUES
  ('vec_contratacion_temporal.registrar_firma_verificada_v2(text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea)'::regprocedure,
   'vec_contratacion_temporal.registrar_firma_verificada_v3(text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea)'::regprocedure,
   'huella_recurso_firma_interior_ct_v1(p_solicitud,p_descriptor_nominal,p_decision)'),
  ('vec_contratacion_temporal.registrar_firma_con_plan_v2(text,timestamptz,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
   'vec_contratacion_temporal.registrar_firma_con_plan_v3(text,timestamptz,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
   'huella_recurso_firma_interior_ct_v1(p_solicitud,descriptor_exacto,p_dec_i)')
 ) v(v2,v3,marca) LOOP
  IF (SELECT row(proowner,prosecdef,proconfig,provolatile,proparallel) FROM pg_catalog.pg_proc WHERE oid=par.v3)
     IS DISTINCT FROM (SELECT row(proowner,prosecdef,proconfig,provolatile,proparallel) FROM pg_catalog.pg_proc WHERE oid=par.v2)
  OR pg_catalog.strpos(pg_catalog.pg_get_functiondef(par.v3),par.marca)=0
  OR (SELECT proacl FROM pg_catalog.pg_proc WHERE oid=par.v3) IS DISTINCT FROM
     (CASE WHEN par.v3::text LIKE '%registrar_firma_con_plan_v3%' THEN acl_plan ELSE acl_directa END)
  OR pg_catalog.has_function_privilege('vec_contratacion_temporal_ejecutor',par.v2,'EXECUTE')
  OR pg_catalog.has_function_privilege('vec_contratacion_temporal_propietario',
     'vec_autorizacion_atestada_v3.registrar_y_consumir_firma_descriptor_ct_v2_atestada(text,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,'EXECUTE')
  THEN RAISE EXCEPTION 'CT181: PARO clave=postimagen actual=divergente esperado=v3_plan_para_ejecutor_v3_directa_cerrada_y_v2_cerrada' USING ERRCODE='55000'; END IF;
 END LOOP;
END $post$;
COMMIT;
