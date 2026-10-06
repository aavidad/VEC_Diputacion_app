\set ON_ERROR_STOP on
-- CT185: registro de firma V2 con la huella exterior (plan) de AD209: ámbitos
-- de la asignación de quien firma, como la interior desde CT181.
--
-- registrar_firma_con_plan_v4 = CT181 v3 con tres sustituciones exactas sobre
-- la definición medida: el nombre, la huella exterior (AD209) y la llamada al
-- consumo exterior v3 (AD209). La interior sigue siendo la de AD206 y el
-- registro sigue entrando por registrar_firma_verificada_v3.
-- El ejecutor CT pasa de la v3 a la v4 y AD177 v2 se cierra para el
-- propietario CT, que ya no la llama. Requiere CT181 y AD209. Una sola vez;
-- sin DOWN.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_contratacion_temporal:migracion:000185',0));
DO $pre$
BEGIN
 IF pg_catalog.current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
 THEN RAISE EXCEPTION 'CT185: PARO clave=migrador_PG actual=no_acreditado esperado=superusuario_PG18' USING ERRCODE='42501'; END IF;
 IF pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.huella_recurso_plan_firma_ct_v1(text,bytea,bytea)') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.consumir_plan_firma_ct_v3_atestada(text,bytea,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 OR NOT pg_catalog.has_function_privilege('vec_contratacion_temporal_propietario',
     'vec_autorizacion_atestada_v3.huella_recurso_plan_firma_ct_v1(text,bytea,bytea)','EXECUTE')
 OR NOT pg_catalog.has_function_privilege('vec_contratacion_temporal_propietario',
     'vec_autorizacion_atestada_v3.consumir_plan_firma_ct_v3_atestada(text,bytea,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 THEN RAISE EXCEPTION 'CT185: PARO clave=AD209 actual=ausente esperado=instalada' USING ERRCODE='55000'; END IF;
 IF pg_catalog.to_regprocedure('vec_contratacion_temporal.registrar_firma_con_plan_v3(text,timestamptz,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.pg_get_functiondef(
     'vec_contratacion_temporal.registrar_firma_con_plan_v3(text,timestamptz,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure),'UTF8')),'hex')
     IS DISTINCT FROM 'f6417b515afa4eeb5eb0de0d4598a5189a5b1eafbe0ac879764f83eef3d204d8'
 THEN RAISE EXCEPTION 'CT185: PARO clave=CT181 actual=distinto esperado=definicion_medida' USING ERRCODE='55000'; END IF;
 IF pg_catalog.to_regprocedure('vec_contratacion_temporal.registrar_firma_con_plan_v4(text,timestamptz,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 THEN RAISE EXCEPTION 'CT185: PARO clave=preimagen actual=v4_presente esperado=ausente' USING ERRCODE='55000'; END IF;
END $pre$;

-- Cada sustitución se comprueba: la marca aparece exactamente una vez.
CREATE FUNCTION pg_temp.sustituir_ct185(t text,viejo text,nuevo text) RETURNS text LANGUAGE plpgsql AS $f$
BEGIN
 IF (pg_catalog.length(t)-pg_catalog.length(pg_catalog.replace(t,viejo,'')))/pg_catalog.length(viejo)<>1 THEN
  RAISE EXCEPTION 'CT185: PARO clave=marca actual=no_unica esperado=una_vez detalle=%',pg_catalog.left(viejo,60) USING ERRCODE='55000'; END IF;
 RETURN pg_catalog.replace(t,viejo,nuevo);
END $f$;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
DO $v4$
DECLARE t text;
BEGIN
 t:=pg_catalog.pg_get_functiondef('vec_contratacion_temporal.registrar_firma_con_plan_v3(text,timestamptz,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure);
 t:=pg_temp.sustituir_ct185(t,'vec_contratacion_temporal.registrar_firma_con_plan_v3(','vec_contratacion_temporal.registrar_firma_con_plan_v4(');
 t:=pg_temp.sustituir_ct185(t,'consumir_plan_firma_ct_v2_atestada(','consumir_plan_firma_ct_v3_atestada(');
 t:=pg_temp.sustituir_ct185(t,
  E' contexto_e:=encode(sha256(convert_to(''{"ambitos":{"organizacion_ref":"''||(s->>''OrganizacionRef'')||\n  ''"},"atributos":{"material_sha256":"''||h||''","plan_firma_sha256":"''||eh||''"}}'',''UTF8'')),''hex'');',
  ' contexto_e:=vec_autorizacion_atestada_v3.huella_recurso_plan_firma_ct_v1(p_solicitud,p_envoltorio,p_dec_e);');
 EXECUTE t;
END $v4$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.registrar_firma_con_plan_v4(text,timestamptz,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.registrar_firma_con_plan_v4(text,timestamptz,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_contratacion_temporal_ejecutor;
-- La v3 calcula la huella exterior sólo con la organización: deja de ser ejecutable.
REVOKE EXECUTE ON FUNCTION vec_contratacion_temporal.registrar_firma_con_plan_v3(text,timestamptz,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM vec_contratacion_temporal_ejecutor;
RESET ROLE;

-- Con las v2 y v3 de CT176 cerradas, AD177 v2 se queda sin llamador: también
-- se cierra para el propietario CT (la concedió el propietario AD en AD177).
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
REVOKE EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_plan_firma_ct_v2_atestada(text,bytea,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM vec_contratacion_temporal_propietario;
RESET ROLE;

DO $post$
DECLARE v3 regprocedure:='vec_contratacion_temporal.registrar_firma_con_plan_v3(text,timestamptz,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 v4 regprocedure:='vec_contratacion_temporal.registrar_firma_con_plan_v4(text,timestamptz,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 def text;
BEGIN
 def:=pg_catalog.pg_get_functiondef(v4);
 IF (SELECT row(proowner,prosecdef,proconfig,provolatile,proparallel) FROM pg_catalog.pg_proc WHERE oid=v4)
    IS DISTINCT FROM (SELECT row(proowner,prosecdef,proconfig,provolatile,proparallel) FROM pg_catalog.pg_proc WHERE oid=v3)
 OR pg_catalog.strpos(def,'huella_recurso_plan_firma_ct_v1(p_solicitud,p_envoltorio,p_dec_e)')=0
 OR pg_catalog.strpos(def,'consumir_plan_firma_ct_v3_atestada(')=0
 OR pg_catalog.strpos(def,'huella_recurso_firma_interior_ct_v1(p_solicitud,descriptor_exacto,p_dec_i)')=0
 OR pg_catalog.strpos(def,'registrar_firma_verificada_v3(')=0
 OR (SELECT proacl FROM pg_catalog.pg_proc WHERE oid=v4) IS DISTINCT FROM
    ARRAY['vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario',
          'vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario']::aclitem[]
 OR pg_catalog.has_function_privilege('vec_contratacion_temporal_ejecutor',v3,'EXECUTE')
 OR pg_catalog.has_function_privilege('vec_contratacion_temporal_propietario',
     'vec_autorizacion_atestada_v3.consumir_plan_firma_ct_v2_atestada(text,bytea,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,'EXECUTE')
 THEN RAISE EXCEPTION 'CT185: PARO clave=postimagen actual=divergente esperado=v4_para_ejecutor_v3_y_AD177v2_cerradas' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
