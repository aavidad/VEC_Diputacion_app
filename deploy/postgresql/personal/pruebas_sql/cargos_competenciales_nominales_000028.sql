\set ON_ERROR_STOP on
-- Comprobación estructural sin altas. Ejecutar sólo sobre un clon desechable
-- después de AD166 y Personal28; la transacción no conserva efectos.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
DO $test$
DECLARE f oid; p record; n text;
BEGIN
 f:=to_regprocedure('vec_personal.leer_revalidar_cargo_ocupante_ct_v1(bytea)');
 IF f IS NULL THEN RAISE EXCEPTION 'Personal28: lector ausente'; END IF;
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=f;
 IF p.proowner<>'vec_personal_propietario'::regrole OR NOT p.prosecdef
  OR p.provolatile<>'v' OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
  OR EXISTS(SELECT 1 FROM aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
    WHERE a.grantee NOT IN (p.proowner,'vec_autorizacion_propietario'::regrole))
  OR NOT has_function_privilege('vec_autorizacion_propietario',f,'EXECUTE')
  OR has_function_privilege('vec_personal_ejecutor',f,'EXECUTE') THEN
  RAISE EXCEPTION 'Personal28: ACL del lector divergente'; END IF;
 f:=to_regprocedure('vec_personal.publicar_cargo_competencial_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF f IS NULL OR NOT has_function_privilege('vec_personal_ejecutor',f,'EXECUTE')
  OR EXISTS(SELECT 1 FROM pg_proc proc_catalogo,
    LATERAL aclexplode(coalesce(proc_catalogo.proacl,acldefault('f',proc_catalogo.proowner))) a
    WHERE proc_catalogo.oid=f AND a.grantee=0) THEN
  RAISE EXCEPTION 'Personal28: fachada de publicación divergente'; END IF;
 FOREACH n IN ARRAY ARRAY['cargo_competencial_historia','cargo_competencial_actual',
  'enlace_cargo_competencial_historia','enlace_cargo_competencial_actual','recibo_publicacion_cargo_competencial'] LOOP
  IF NOT EXISTS(SELECT 1 FROM pg_class c WHERE c.oid=to_regclass('vec_personal.'||n)
    AND c.relowner='vec_personal_propietario'::regrole AND c.relrowsecurity AND c.relforcerowsecurity)
   OR has_table_privilege('vec_personal_ejecutor','vec_personal.'||n,'SELECT') THEN
   RAISE EXCEPTION 'Personal28: tabla expuesta %',n; END IF;
 END LOOP;
END $test$;
-- Negativo real de la fachada, con sesión ejecutora sintética sin autoridad
-- de propietario. El material inválido se rechaza antes de consumir V3.
-- La identidad y las concesiones PostgreSQL de ensayo desaparecen en ROLLBACK.
CREATE ROLE vec_personal28_cas_prueba LOGIN INHERIT NOSUPERUSER NOCREATEDB
 NOCREATEROLE NOREPLICATION NOBYPASSRLS;
GRANT vec_personal_ejecutor TO vec_personal28_cas_prueba
 WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
SET LOCAL SESSION AUTHORIZATION vec_personal28_cas_prueba;
DO $cas$
DECLARE operacion text;objeto text;m jsonb;d jsonb;c jsonb;caso jsonb;base jsonb;
BEGIN
 FOREACH operacion IN ARRAY ARRAY['cargo','enlace'] LOOP
  objeto:=CASE operacion WHEN 'cargo' THEN 'car_' ELSE 'enc_' END||repeat('a',22);
  base:=jsonb_build_object('esquema','vec.personal.cargo-competencial.publicacion.v1',
   'operacion',operacion,'clave_idempotencia',repeat('a',32),'objeto_ref',objeto,
   'organizacion_ref','org_sintetica','unidad_ref','unidad_sintetica',
   'version_esperada',0,'huella_esperada',NULL,
   'datos',jsonb_build_object(CASE operacion WHEN 'cargo' THEN 'cargo_ref' ELSE 'enlace_ref' END,
    objeto,'version',1,'estado','vigente'));
  d:=jsonb_build_object('concedida',true,'accion','personal.cargo_competencial.publicar',
   'modulo_id','personal','tipo_recurso','cargo_competencial',
   'finalidad','administrar_cargos_competenciales','recurso_ref',objeto,
   'campos_permitidos','["cargo","enlace","huella_sha256","recibo","version"]'::jsonb,
   'decision_ref','decision:personal28:cas:sintetica');
  c:=jsonb_build_object('operacion','personal.cargo_competencial.publicar',
   'audiencia_consumo','vec_personal.cargo_competencial.publicar.v1','efecto_ref',objeto,
   'decision_ref',d->>'decision_ref');
  FOR caso IN SELECT x FROM jsonb_array_elements('[
    {"version_esperada":null}, {"version_esperada":"0"},
    {"version_esperada":-1}, {"version_esperada":0.5},
    {"version_esperada":9223372036854775808},
    {"version_esperada":1,"huella_esperada":null}
   ]'::jsonb) AS v(x) LOOP
   m:=base||caso;
   IF caso->>'version_esperada'='1' THEN
    m:=jsonb_set(m,'{datos,version}','2'::jsonb);
   END IF;
   BEGIN
    PERFORM vec_personal.publicar_cargo_competencial_v1(
     convert_to(m::text,'UTF8'),convert_to(c::text,'UTF8'),convert_to(d::text,'UTF8'),
     convert_to('{}','UTF8'),convert_to('{}','UTF8'),1,1,
     convert_to('{}','UTF8'),convert_to('{}','UTF8'),convert_to('{}','UTF8'),convert_to('{}','UTF8'));
    RAISE EXCEPTION 'Personal28: aceptada entrada CAS inválida para %: %',operacion,caso;
   EXCEPTION WHEN SQLSTATE '22023' THEN NULL;
   END;
  END LOOP;
 END LOOP;
END $cas$;
RESET SESSION AUTHORIZATION;
ROLLBACK;
