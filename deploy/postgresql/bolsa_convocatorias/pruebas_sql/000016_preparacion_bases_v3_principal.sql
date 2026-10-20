\set ON_ERROR_STOP on
-- Sin escenario positivo ficticio: contrato/ACL y validación mecánica únicamente.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL statement_timeout='15s';
DO $contrato$
DECLARE x record;f oid;obj oid;t text;
BEGIN
 FOR x IN SELECT * FROM (VALUES
  ('guardar_preparacion_bases_v3','text,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea','vec_bolsa_convocatorias_ejecutor_preparacion_bases','vec_bolsa_convocatorias_lector_preparacion_bases'),
  ('obtener_preparacion_bases_v3','text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea','vec_bolsa_convocatorias_lector_preparacion_bases','vec_bolsa_convocatorias_ejecutor_preparacion_bases')) q(nombre,firma,propio,ajeno) LOOP
  f:=to_regprocedure('vec_bolsa_convocatorias.'||x.nombre||'('||x.firma||')');
  IF f IS NULL OR NOT has_function_privilege(x.propio,f,'EXECUTE') OR has_function_privilege(x.ajeno,f,'EXECUTE')
  OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f AND p.proowner='vec_bolsa_convocatorias_propietario'::regrole AND p.prosecdef
    AND p.proconfig @> ARRAY['search_path=pg_catalog, pg_temp','row_security=on','lock_timeout=2s','statement_timeout=15s']
    AND array_length(p.proallargtypes,1)-p.pronargs=17)
  OR EXISTS(SELECT 1 FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
   WHERE p.oid=f AND (a.grantee NOT IN(p.proowner,x.propio::regrole) OR a.grantor<>p.proowner OR a.is_grantable))
  THEN RAISE EXCEPTION 'BC8: ABI o permisos de función divergentes'; END IF;
 END LOOP;
 FOREACH t IN ARRAY ARRAY['preparacion_bases_version_v3','preparacion_bases_actual_v3','preparacion_bases_historia_v3','preparacion_bases_outbox_v3','preparacion_bases_recibo_v3','preparacion_bases_acceso_v3'] LOOP
  obj:=to_regclass('vec_bolsa_convocatorias.'||t);
  IF obj IS NULL OR NOT EXISTS(SELECT 1 FROM pg_class c WHERE c.oid=obj AND c.relrowsecurity AND c.relforcerowsecurity AND c.relowner='vec_bolsa_convocatorias_propietario'::regrole)
  OR (SELECT count(*) FROM pg_policy p WHERE p.polrelid=obj)<>1
  OR NOT EXISTS(SELECT 1 FROM pg_policy p WHERE p.polrelid=obj AND p.polroles=ARRAY['vec_bolsa_convocatorias_propietario'::regrole::oid])
  OR EXISTS(SELECT 1 FROM pg_class c,LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) a WHERE c.oid=obj AND a.grantee<>c.relowner)
  OR has_table_privilege('vec_bolsa_convocatorias_ejecutor_preparacion_bases',obj,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE')
  OR has_table_privilege('vec_bolsa_convocatorias_lector_preparacion_bases',obj,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE')
  THEN RAISE EXCEPTION 'BC8: tabla fuera de límites nominales'; END IF;
 END LOOP;
END $contrato$;
SET LOCAL ROLE vec_bolsa_convocatorias_propietario;
DO $material$
DECLARE m text;base text;bad text;contador integer:=0;
BEGIN
 base:=',"actor_ref":"per_'||repeat('a',22)||'","contexto_actor_ref":"vca_'||repeat('b',22)||'","contexto_version":18446744073709551615,"persona_version":9007199254740991,"perfil_ref":"prf_'||repeat('c',22)||'","perfil_version":9007199254740991,"correlacion_ref":"correlacion_'||repeat('d',32)||'"}';
 m:='{"esquema":"vec.bolsa.preparacion-bases.consultar.v3","preparacion_ref":"prep:sintetica","organizacion_ref":"org_diputaciongranada","unidad_gestion_ref":"","modo":"actual","revision":0,"huella_material_sha256":""'||base;
 PERFORM vec_bolsa_convocatorias.validar_material_preparacion_bases_v3(m,NULL,false);
 m:=replace(replace(replace(m,'"modo":"actual"','"modo":"exacta"'),'"revision":0','"revision":1000000'),'"huella_material_sha256":""','"huella_material_sha256":"'||repeat('a',64)||'"');
 PERFORM vec_bolsa_convocatorias.validar_material_preparacion_bases_v3(m,NULL,false);
 FOREACH bad IN ARRAY ARRAY[
  replace(m,'"revision":1000000','"revision":1000001'),
  replace(m,'"persona_version":9007199254740991','"persona_version":9007199254740992'),
  replace(m,'"contexto_version":18446744073709551615','"contexto_version":18446744073709551616'),
  replace(m,'"modo":"exacta"','"modo":"actual"'),
  replace(m,'"organizacion_ref":"org_diputaciongranada"','"organizacion_ref":"org:*"'),
  replace(m,'{"esquema":','{ "esquema":')
 ] LOOP
  BEGIN
   PERFORM vec_bolsa_convocatorias.validar_material_preparacion_bases_v3(bad,NULL,false);
   RAISE EXCEPTION 'BC8: material inválido aceptado';
  EXCEPTION WHEN invalid_parameter_value THEN contador:=contador+1; END;
 END LOOP;
 IF contador<>6 THEN RAISE EXCEPTION 'BC8: negativos de material incompletos'; END IF;
END $material$;
ROLLBACK;
