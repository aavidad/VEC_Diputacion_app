\set ON_ERROR_STOP on
-- AUT100: revalidar la sesión documental externa por su LOGIN y grupo exactos.
-- Las dos puertas externas previas conservan sus contratos y permisos.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='45s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion:migracion:000100',0));
DO $pre$
DECLARE p pg_proc%ROWTYPE;
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 OR to_regrole('vec_documentos_bolsa_subida_datos') IS NULL
 OR to_regprocedure('vec_autorizacion.registrar_y_revalidar_documentos_bolsa_externa_v3(bytea,bytea,numeric,numeric)') IS NULL
 OR to_regprocedure('vec_autorizacion.revalidar_documentos_bolsa_externa_v3_viva(bytea,bytea,numeric,numeric)') IS NULL
 THEN RAISE EXCEPTION 'AUT100: dependencia nominal ausente' USING ERRCODE='55000'; END IF;
 SELECT * INTO STRICT p FROM pg_proc WHERE oid='vec_autorizacion.revalidar_sesion_vinculo_externo_v2(jsonb,timestamptz,timestamptz,timestamptz)'::regprocedure;
 IF p.proowner IS DISTINCT FROM 'vec_autorizacion_propietario'::regrole
 OR p.prosecdef IS NOT TRUE OR p.provolatile IS DISTINCT FROM 'v'
 OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp']::text[]
 OR p.proacl::text IS DISTINCT FROM '{vec_autorizacion_propietario=X/vec_autorizacion_propietario,vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_propietario}'
 OR encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex') IS DISTINCT FROM 'd2e93fe30afe14e5ffb46a20f10cda0415f026ef7584e750d0025d62652c14e4'
 THEN RAISE EXCEPTION 'AUT100: preimagen divergente' USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_autorizacion_propietario;
CREATE OR REPLACE FUNCTION vec_autorizacion.revalidar_sesion_vinculo_externo_v2(p_vinculo jsonb, p_emitida_en timestamp with time zone, p_valida_hasta timestamp with time zone, p_instante timestamp with time zone)
 RETURNS boolean
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
AS $function$
BEGIN
 IF vec_autorizacion.vinculo_contexto_actor_v2_valido(p_vinculo) IS NOT TRUE
    OR NOT (vec_autorizacion.login_candidato_externo_v1('vec_externo_v3_registro_autorizacion_desarrollo','vec_autorizacion_registro_externo')
       OR vec_autorizacion.login_candidato_externo_v1('vec_externo_bolsa_desarrollo','vec_bolsa_llamamientos_portal_externo')
       OR vec_autorizacion.login_candidato_externo_v1('vec_externo_documentos_bolsa_datos','vec_documentos_bolsa_subida_datos'))
    OR p_vinculo->>'superficie' IS DISTINCT FROM 'externa_personal'
    OR p_vinculo->>'cuenta_privilegiada' IS DISTINCT FROM 'false'
    OR p_emitida_en IS NULL OR p_valida_hasta IS NULL OR p_instante IS NULL
    OR p_valida_hasta<=p_emitida_en
 THEN RETURN false; END IF;
 IF vec_identidad_externa_v1.acreditar_sesion_externa_v1(
   p_vinculo->>'autenticacion_ref',p_vinculo->>'autenticacion_huella_sha256',
   p_vinculo->>'asercion_ref',p_vinculo->>'sesion_ref',p_vinculo->>'cuenta_ref',
   p_vinculo->>'cuenta_ordinaria_ref',(p_vinculo->>'cuenta_privilegiada')::boolean,
   p_vinculo->>'superficie',p_vinculo->>'metodo_observado',p_vinculo->>'garantia_observada',
   p_vinculo->>'politica_garantia_ref',p_vinculo->>'politica_garantia_huella_sha256',
   (p_vinculo->>'autenticacion_verificada_en')::timestamptz,(p_vinculo->>'sesion_emitida_en')::timestamptz,
   p_vinculo->>'control_sesion_ref',p_vinculo->>'control_sesion_revision','activa',
   p_vinculo->>'control_sesion_huella_sha256',
   (p_vinculo->>'sesion_revalidada_en')::timestamptz,(p_vinculo->>'sesion_valida_hasta')::timestamptz
 ) IS NOT TRUE THEN RETURN false; END IF;
 RETURN p_emitida_en >= (p_vinculo->>'sesion_revalidada_en')::timestamptz
    AND p_valida_hasta <= (p_vinculo->>'sesion_valida_hasta')::timestamptz
    AND p_instante >= (p_vinculo->>'sesion_revalidada_en')::timestamptz
    AND p_instante < (p_vinculo->>'sesion_valida_hasta')::timestamptz;
EXCEPTION WHEN data_exception OR invalid_text_representation OR datetime_field_overflow THEN RETURN false;
END $function$
;
RESET ROLE;
DO $post$
DECLARE p pg_proc%ROWTYPE;
BEGIN
 SELECT * INTO STRICT p FROM pg_proc WHERE oid='vec_autorizacion.revalidar_sesion_vinculo_externo_v2(jsonb,timestamptz,timestamptz,timestamptz)'::regprocedure;
 IF p.proowner IS DISTINCT FROM 'vec_autorizacion_propietario'::regrole
 OR p.prosecdef IS NOT TRUE OR p.provolatile IS DISTINCT FROM 'v'
 OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp']::text[]
 OR p.proacl::text IS DISTINCT FROM '{vec_autorizacion_propietario=X/vec_autorizacion_propietario,vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_propietario}'
 OR encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex') IS DISTINCT FROM '5fe5f3036e839d296329a9b99bc97d4040794841ec174e5af717e38d20c1f59d'
 THEN RAISE EXCEPTION 'AUT100: postimagen divergente' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
