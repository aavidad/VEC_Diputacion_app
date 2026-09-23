-- Retirada exclusivamente antes de aprovisionar cualquier login productivo.
BEGIN;
SET LOCAL search_path=pg_catalog;
DO $preimagen$
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
    OR EXISTS (SELECT 1 FROM pg_auth_members
       WHERE roleid='vec_contratacion_temporal_consultor_rrhh_ambito'::regrole)
 THEN RAISE EXCEPTION 'CT109: retirada con identidad activa' USING ERRCODE='55000'; END IF;
END $preimagen$;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog;
DROP FUNCTION vec_contratacion_temporal.consultar_cuadro_rrhh_ambito_v1(
 vec_contratacion_temporal.alcance_consulta_rrhh_v1,
 vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
 jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_contratacion_temporal.consultar_detalle_rrhh_ambito_v1(
 vec_contratacion_temporal.alcance_consulta_rrhh_v1,
 vec_contratacion_temporal.consulta_detalle_rrhh_v1,
 jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_contratacion_temporal.login_consultor_rrhh_ambito_v1();

DO $restaurar_guardas_privadas$
DECLARE f regprocedure; anterior pg_proc%ROWTYPE; posterior pg_proc%ROWTYPE;
 viejo constant text := '''vec_contratacion_temporal_consultor_rrhh''';
 nuevo constant text := '(CASE WHEN pg_catalog.pg_has_role(SESSION_USER, ''vec_contratacion_temporal_consultor_rrhh_ambito'', ''MEMBER'') THEN ''vec_contratacion_temporal_consultor_rrhh_ambito'' ELSE ''vec_contratacion_temporal_consultor_rrhh'' END)';
 definicion text; cuerpo text; esperado integer; huella_pre text; huella_post text;
BEGIN
 FOR f,esperado,huella_pre,huella_post IN SELECT * FROM (VALUES
  ('vec_contratacion_temporal.acreditar_contexto_motor_consultas_rrhh_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.material_autorizacion_consulta_rrhh_v3)'::regprocedure,1,'1ff9a0b40207685e151507380347c3cd3f5656ca30327c932ea827aeb07d05a1','c3e64993bd7bbb7018a03e487b363486d3d501ff1952d994ef7cea732d5afaa8'),
  ('vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,3,'19c325c76ac29290b71abb7fb3eb13b840ece8336d155b174103774f209af36d','d10eb5676125f8e02ca0cf0857b439ce0b240f9f4b48904170396046e970acbd'),
  ('vec_contratacion_temporal.consultar_detalle_rrhh_atestado_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,3,'982179dcc2783fad81f93c1c96b8598e267507b434e8e3bece6690312382df15','7b7d6c4a419262d54ddb7f2a096e5a4d6e1c962bc58e3027546e221717ed1814')
 ) v(firma,apariciones,pre,post) LOOP
  SELECT * INTO STRICT anterior FROM pg_proc WHERE oid=f;
  IF anterior.proowner<>'vec_contratacion_temporal_propietario'::regrole
     OR NOT anterior.prosecdef OR anterior.provolatile<>'v'
     OR anterior.proparallel<>'u' OR anterior.proleakproof
     OR anterior.prolang<>(SELECT oid FROM pg_language WHERE lanname='plpgsql')
     OR anterior.proconfig IS DISTINCT FROM ARRAY[
       'search_path=pg_catalog','row_security=on','TimeZone=UTC',
       'lock_timeout=1s','statement_timeout=4s',
       'idle_in_transaction_session_timeout=6s']::text[]
     OR (length(anterior.prosrc)-length(replace(anterior.prosrc,nuevo,'')))/length(nuevo)<>esperado
     OR encode(sha256(convert_to(anterior.prosrc,'UTF8')),'hex') IS DISTINCT FROM huella_pre
  THEN RAISE EXCEPTION 'CT109: guarda modificada' USING ERRCODE='55000'; END IF;
  definicion:=pg_get_functiondef(f);
  IF strpos(definicion,anterior.prosrc)=0 THEN
   RAISE EXCEPTION 'CT109: definicion no univoca' USING ERRCODE='55000'; END IF;
  cuerpo:=replace(anterior.prosrc,nuevo,viejo);
  EXECUTE replace(definicion,anterior.prosrc,cuerpo);
  SELECT * INTO STRICT posterior FROM pg_proc WHERE oid=f;
  IF posterior.prosrc IS DISTINCT FROM cuerpo
     OR encode(sha256(convert_to(posterior.prosrc,'UTF8')),'hex') IS DISTINCT FROM huella_post
     OR posterior.proacl IS DISTINCT FROM anterior.proacl
     OR posterior.proowner IS DISTINCT FROM anterior.proowner
  THEN RAISE EXCEPTION 'CT109: restauracion altero ACL' USING ERRCODE='55000'; END IF;
 END LOOP;
END $restaurar_guardas_privadas$;
REVOKE USAGE ON SCHEMA vec_contratacion_temporal FROM vec_contratacion_temporal_consultor_rrhh_ambito;
COMMIT;
