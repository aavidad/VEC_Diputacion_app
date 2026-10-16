\set ON_ERROR_STOP on
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
DO $t$
DECLARE p pg_proc%ROWTYPE;f text;
BEGIN
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=
  'vec_autorizacion.revalidar_sesion_vinculo_externo_v2(jsonb,timestamptz,timestamptz,timestamptz)'::regprocedure;
 f:=pg_get_functiondef(p.oid);
 IF p.proowner IS DISTINCT FROM 'vec_autorizacion_propietario'::regrole
 OR p.prosecdef IS NOT TRUE OR p.provolatile IS DISTINCT FROM 'v'
 OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp']::text[]
 OR p.proacl::text IS DISTINCT FROM
  '{vec_autorizacion_propietario=X/vec_autorizacion_propietario,vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_propietario}'
 OR encode(sha256(convert_to(f,'UTF8')),'hex') IS DISTINCT FROM
  '5fe5f3036e839d296329a9b99bc97d4040794841ec174e5af717e38d20c1f59d'
 OR position('vec_externo_v3_registro_autorizacion_desarrollo' IN f)=0
 OR position('vec_externo_bolsa_desarrollo' IN f)=0
 OR position('vec_externo_documentos_bolsa_datos' IN f)=0
 OR position('vec_documentos_bolsa_subida_datos' IN f)=0
 THEN RAISE EXCEPTION 'AUT100: contrato de sesión externa divergente'; END IF;
 IF has_function_privilege('vec_documentos_bolsa_rrhh',p.oid,'EXECUTE')
 OR has_function_privilege('vec_documentos_bolsa_subida_datos',p.oid,'EXECUTE')
 THEN RAISE EXCEPTION 'AUT100: acceso directo de aplicación no permitido'; END IF;
 -- Un LOGIN ajeno falla antes de acreditar sesión, también si presenta un
 -- vínculo sintácticamente vacío. SET ROLE no cambia session_user.
 IF vec_autorizacion.revalidar_sesion_vinculo_externo_v2(
  '{}'::jsonb,clock_timestamp(),clock_timestamp()+interval '1 second',clock_timestamp()) IS TRUE
 THEN RAISE EXCEPTION 'AUT100: LOGIN ajeno admitido'; END IF;
END $t$;
ROLLBACK;
