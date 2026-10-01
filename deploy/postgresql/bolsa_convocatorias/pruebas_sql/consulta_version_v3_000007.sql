\set ON_ERROR_STOP on
-- Comprobaciones de frontera. No acreditan una concesión positiva firmada.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL statement_timeout='15s';
SET LOCAL lock_timeout='2s';
DO $acl$
DECLARE f regprocedure:='vec_bolsa_convocatorias.obtener_version_exacta_v3(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_proc WHERE oid=f AND proowner='vec_bolsa_convocatorias_propietario'::regrole AND prosecdef AND proconfig=ARRAY['search_path=pg_catalog','row_security=on','TimeZone=UTC','lock_timeout=2s'])
 OR NOT has_function_privilege('vec_bolsa_convocatorias_ejecutor_consulta',f,'EXECUTE')
 OR has_function_privilege('vec_bolsa_convocatorias_ejecutor_consulta','vec_bolsa_convocatorias.obtener_version_exacta_v1(jsonb,jsonb,bytea,bytea)','EXECUTE')
 OR has_table_privilege('vec_bolsa_convocatorias_ejecutor_consulta','vec_bolsa_convocatorias.version_convocatoria','SELECT')
 OR has_table_privilege('vec_bolsa_convocatorias_ejecutor_consulta','vec_bolsa_convocatorias.lectura_version_v3','SELECT')
 OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f AND a.grantee=0)
 OR NOT EXISTS(SELECT 1 FROM pg_class WHERE oid='vec_bolsa_convocatorias.lectura_version_v3'::regclass AND relrowsecurity AND relforcerowsecurity)
 OR NOT EXISTS(SELECT 1 FROM pg_policy WHERE polrelid='vec_bolsa_convocatorias.lectura_version_v3'::regclass AND polroles=ARRAY['vec_bolsa_convocatorias_propietario'::regrole::oid])
 THEN RAISE EXCEPTION 'S1: frontera o ACL incorrecta'; END IF;
END $acl$;

CREATE ROLE s1_prueba_lectura_v3 LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
GRANT vec_bolsa_convocatorias_ejecutor_consulta TO s1_prueba_lectura_v3 WITH ADMIN FALSE,INHERIT TRUE,SET FALSE;
SET SESSION AUTHORIZATION s1_prueba_lectura_v3;
DO $sin_material$
BEGIN
 BEGIN
  PERFORM * FROM vec_bolsa_convocatorias.obtener_version_exacta_v3(NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'S1: lectura sin material autorizada';
 EXCEPTION WHEN insufficient_privilege THEN NULL;
 END;
END $sin_material$;
RESET SESSION AUTHORIZATION;
DO $sin_efectos$
BEGIN
 IF EXISTS(SELECT 1 FROM vec_bolsa_convocatorias.lectura_version_v3) THEN
  RAISE EXCEPTION 'S1: intento sin material dejó recibo';
 END IF;
END $sin_efectos$;
ROLLBACK;
