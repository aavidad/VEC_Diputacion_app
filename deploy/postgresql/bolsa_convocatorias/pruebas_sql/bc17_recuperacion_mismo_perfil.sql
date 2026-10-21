\set ON_ERROR_STOP on
-- BC17: comprueba la definición instalada y que la reconstrucción conserva
-- la única entrada pública de la declaración. El recorrido con decisiones V3
-- válidas se comprueba desde la API con material firmado real.
BEGIN READ ONLY;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL statement_timeout='10s';
DO $test$
DECLARE
 f regprocedure;
 p record;
 ejecutor oid:='vec_bolsa_llamamientos_ejecutor'::regrole;
BEGIN
 f:=to_regprocedure('vec_bolsa_convocatorias.declarar_publicacion_oficial_bases_rrhh_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF f IS NULL THEN RAISE EXCEPTION 'BC17: falta la función'; END IF;
 SELECT proowner,prosecdef,provolatile,proparallel,proacl,proconfig INTO STRICT p
 FROM pg_proc WHERE oid=f;
 IF pg_get_userbyid(p.proowner)<>'vec_bolsa_convocatorias_propietario'
 OR p.prosecdef IS DISTINCT FROM true
 OR p.provolatile IS DISTINCT FROM 'v'
 OR p.proparallel IS DISTINCT FROM 'u'
 OR p.proacl::text IS DISTINCT FROM
  '{vec_bolsa_convocatorias_propietario=X/vec_bolsa_convocatorias_propietario,vec_bolsa_llamamientos_ejecutor=X/vec_bolsa_convocatorias_propietario}'
 OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on',
  'TimeZone=UTC','lock_timeout=2s','statement_timeout=15s']::text[]
 OR encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex')
  <>'6f0fd898510d55b7f0662aa12d0b8990263d508b90216b953e02904ca185fbb0'
 THEN RAISE EXCEPTION 'BC17: definición, dueño o permisos divergentes'; END IF;
 IF NOT has_function_privilege('vec_bolsa_llamamientos_ejecutor',f,'EXECUTE')
 OR NOT has_schema_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_convocatorias','USAGE')
 OR has_table_privilege('vec_bolsa_llamamientos_ejecutor',
  'vec_bolsa_convocatorias.publicacion_oficial_bases_autorizacion','SELECT')
 OR EXISTS(SELECT 1 FROM pg_proc x
  CROSS JOIN LATERAL aclexplode(coalesce(x.proacl,acldefault('f',x.proowner))) a
  WHERE x.oid=f AND a.grantee=0)
 OR EXISTS(SELECT 1 FROM pg_proc x JOIN pg_namespace n ON n.oid=x.pronamespace
  CROSS JOIN LATERAL aclexplode(coalesce(x.proacl,acldefault('f',x.proowner))) a
  WHERE n.nspname='vec_bolsa_convocatorias' AND a.grantee=ejecutor AND x.oid<>f
   AND x.oid<>to_regprocedure('vec_bolsa_convocatorias.consultar_publicaciones_oficiales_bases_rrhh_v1(text,integer)'))
 THEN RAISE EXCEPTION 'BC17: concesiones ampliadas'; END IF;
END $test$;
ROLLBACK;
