\set ON_ERROR_STOP on
-- Ejecutar ANTES de considerar la UP histórica AD218. Sólo lee catálogos;
-- PRE218 termina con aviso y exit0; POST218 y estado mixto detienen el paso.
BEGIN READ ONLY;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL statement_timeout='10s';
DO $estado$
DECLARE
 nucleo oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 fachada oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_carga_convoca_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 definicion text; fuente text; audiencia text;
 def_sha text; src_sha text; check_sha text; validado boolean;
 origenes bigint; origen_exacto boolean; fachada_exacta boolean;
 pre_def constant text:='96d92f060ddfd4970bf47167c5b23ee140bc67bd521166363c942f6f83203bc3';
 pre_src constant text:='03621298761d25e5e5decb2a8bee13cb2193f9fc07035a6dd7145ea5e5eec1cd';
 pre_check constant text:='8dae0267b85ad237770d0ccdb7fbf9862afe6d7d022c0c93f4385c182bcbc68e';
 post_def constant text:='c74551eab17bea78bb564d14d96b77f33bd24a5874b7bdb6e100a59d8f2fe714';
 post_src constant text:='79d2f29752235a01716d49095777fe8e890a6deed5269a8647d1f866d4b671b5';
 post_check constant text:='4cba2a36938c4dca934f30963ec09302dd2f0734718c45002404bedf7f3185f5';
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR session_user<>current_user
 OR NOT EXISTS (SELECT 1 FROM pg_roles r WHERE r.rolname=session_user AND r.rolsuper)
 THEN RAISE EXCEPTION 'AD218 PARO clave=lector esperado=DBA_PG18_sin_SET_ROLE observado=otro'
  USING ERRCODE='42501'; END IF;
 IF nucleo IS NULL OR to_regclass('vec_autorizacion_atestada_v3.clave_capacidad_version') IS NULL
 OR to_regclass('vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1') IS NULL
 THEN RAISE EXCEPTION 'AD218 PARO clave=esquema esperado=nucleo_CHECK_origen observados=ausentes'
  USING ERRCODE='55000'; END IF;
 SELECT pg_get_functiondef(p.oid),p.prosrc INTO STRICT definicion,fuente FROM pg_proc p WHERE p.oid=nucleo;
 def_sha:=encode(sha256(convert_to(definicion,'UTF8')),'hex');
 src_sha:=encode(sha256(convert_to(fuente,'UTF8')),'hex');
 SELECT pg_get_constraintdef(c.oid,false),c.convalidated INTO audiencia,validado FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
  AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c';
 check_sha:=CASE WHEN audiencia IS NULL THEN NULL ELSE encode(sha256(convert_to(audiencia,'UTF8')),'hex') END;
 SELECT count(*),coalesce(bool_and(proceso='vec-server' AND canal_permitido='interna_corporativa'),false)
 INTO origenes,origen_exacto FROM vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1
 WHERE login_nombre='vec_bolsa_llamamientos_desarrollo'
  AND audiencia_consumo='vec_bolsa_llamamientos.carga_convoca.confirmar.v1'
  AND operacion='bolsa.carga_convoca.confirmar';
 SELECT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=fachada
  AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef
  AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
  AND has_function_privilege('vec_bolsa_llamamientos_propietario',fachada,'EXECUTE')
  AND NOT has_function_privilege('vec_bolsa_llamamientos_ejecutor',fachada,'EXECUTE'))
 INTO fachada_exacta;
 IF fachada_exacta THEN
  SELECT count(*)=2 INTO fachada_exacta FROM pg_proc p CROSS JOIN LATERAL
   aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=fachada;
 END IF;
 IF def_sha=post_def AND src_sha=post_src AND check_sha=post_check AND validado
  AND fachada_exacta AND origenes=1 AND origen_exacto THEN
  RAISE EXCEPTION 'AD218 ya instalada clave=estado esperado=POST216_antes_UP observado=POST218_def=%_src=%_CHECK=%',
   def_sha,src_sha,check_sha USING ERRCODE='55000';
 END IF;
 IF def_sha=pre_def AND src_sha=pre_src AND check_sha=pre_check AND validado
  AND fachada IS NULL AND origenes=0 THEN
  RAISE NOTICE 'AD218 pendiente clave=estado esperado=PRE218_observado=PRE218 def=% src=% CHECK=%',
   def_sha,src_sha,check_sha;
  RETURN;
 END IF;
 RAISE EXCEPTION 'AD218 PARO clave=estado esperado=PRE218_o_POST218_exactos observado=def:% src:% CHECK:% validado:% fachada:% origenes:% origen_exacto:%',
  coalesce(def_sha,'ausente'),coalesce(src_sha,'ausente'),coalesce(check_sha,'ausente'),
  coalesce(validado,false),fachada IS NOT NULL,origenes,origen_exacto USING ERRCODE='55000';
END $estado$;
ROLLBACK;
