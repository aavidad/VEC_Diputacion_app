\set ON_ERROR_STOP on
-- B134 / F1-09: intento nominal, sobre el cuerpo instalado exportado de main.
-- Última definición: B17 de origin/main, SHA256 del fichero
-- 120bd57990fd8a5385a6322a09d421c1bba84c6033d6e1ad4ae23d1f6136a2db.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000134',0));
DO $pre$ BEGIN
 IF current_setting('server_version_num')::int NOT BETWEEN 180000 AND 189999
 OR to_regclass('vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento') IS NULL
 OR encode(sha256(convert_to(pg_get_functiondef(to_regprocedure(
  'vec_bolsa_llamamientos.registrar_intento_borrador_llamamiento_v1(text,text,text,text,text)')),'UTF8')),'hex')
  IS DISTINCT FROM '5d49b03401c71f4551ae644d138cdf0a0e8503741eb7f6c50a498e08a4429dec'
 OR (SELECT encode(sha256(convert_to(pg_get_constraintdef(oid),'UTF8')),'hex')
  FROM pg_constraint WHERE conrelid='vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento'::regclass
  AND conname='bitacora_intento_borrador_llamamiento_accion_check' AND convalidated) IS DISTINCT FROM 'ba3fa61fb63f10d6d046beb341475a495ffaba0f13efaded8dbb276b2fac83a6'
 OR (SELECT encode(sha256(convert_to(pg_get_constraintdef(oid),'UTF8')),'hex')
  FROM pg_constraint WHERE conrelid='vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento'::regclass
  AND conname='bitacora_intento_borrador_llamamiento_ruta_clase_check' AND convalidated) IS DISTINCT FROM 'e673d62c5771fa1d5d576291820d93fa83cd237ae3e9fc5a116e44709e925532'
 THEN RAISE EXCEPTION 'B134: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
LOCK TABLE vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento IN SHARE ROW EXCLUSIVE MODE;
ALTER TABLE vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento
 DROP CONSTRAINT bitacora_intento_borrador_llamamiento_accion_check,
 ADD CONSTRAINT bitacora_intento_borrador_llamamiento_accion_check CHECK ((accion = ANY (ARRAY['crear'::text, 'consultar'::text, 'cambiar_situacion'::text, 'registrar_contacto'::text, 'registrar_datos_contacto'::text, 'consultar_datos_contacto'::text, 'emitir_llamamiento'::text, 'recuperar_llamamiento'::text, 'registrar_comprobacion_ofrecimiento'::text]))) NOT VALID;
ALTER TABLE vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento
 VALIDATE CONSTRAINT bitacora_intento_borrador_llamamiento_accion_check;
ALTER TABLE vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento
 DROP CONSTRAINT bitacora_intento_borrador_llamamiento_ruta_clase_check,
 ADD CONSTRAINT bitacora_intento_borrador_llamamiento_ruta_clase_check CHECK ((ruta_clase = ANY (ARRAY['coleccion'::text, 'detalle'::text, 'situacion'::text, 'contactos'::text, 'datos_contacto'::text, 'emisiones'::text, 'comprobacion_ofrecimiento'::text]))) NOT VALID;
ALTER TABLE vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento
 VALIDATE CONSTRAINT bitacora_intento_borrador_llamamiento_ruta_clase_check;
CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.registrar_intento_borrador_llamamiento_v1(p_correlacion_ref text, p_accion text, p_ruta_clase text, p_actor_ref text, p_resultado text)
 RETURNS void
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET lock_timeout TO '2s'
 SET statement_timeout TO '5s'
AS $function$
DECLARE v_ref text; BEGIN
 PERFORM vec_bolsa_llamamientos.exigir_runtime_registrador_frontera_borrador_llamamiento();
	 IF p_correlacion_ref IS NULL OR p_correlacion_ref !~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{7,191}$' OR p_accion NOT IN('crear','consultar','cambiar_situacion','registrar_contacto','consultar_datos_contacto','registrar_datos_contacto','emitir_llamamiento','recuperar_llamamiento','registrar_comprobacion_ofrecimiento') OR p_ruta_clase NOT IN('coleccion','detalle','situacion','contactos','datos_contacto','emisiones','comprobacion_ofrecimiento') OR ((p_accion='registrar_comprobacion_ofrecimiento') IS DISTINCT FROM (p_ruta_clase='comprobacion_ofrecimiento')) OR (p_actor_ref IS NOT NULL AND p_actor_ref !~ '^per_[A-Za-z0-9_-]{22,128}$') OR p_resultado NOT IN('autenticacion_requerida','acceso_denegado','recurso_no_disponible','infraestructura_no_disponible','resultado_indeterminado','correcto') THEN RAISE EXCEPTION 'intento de frontera Bolsa inválido' USING ERRCODE='22023'; END IF;
 v_ref:='intento:'||translate(encode(sha256(convert_to(p_correlacion_ref||':'||p_accion||':'||p_ruta_clase||':'||coalesce(p_actor_ref,'')||':'||p_resultado||':'||clock_timestamp()::text,'UTF8')),'hex'),'0123456789','ghijklmnop');
 INSERT INTO vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento(intento_ref,correlacion_ref,accion,ruta_clase,actor_ref,resultado,registrada_en) VALUES(v_ref,p_correlacion_ref,p_accion,p_ruta_clase,p_actor_ref,p_resultado,clock_timestamp());
END $function$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.registrar_intento_borrador_llamamiento_v1(text,text,text,text,text) FROM PUBLIC;
DO $post$ DECLARE p pg_proc%ROWTYPE;
BEGIN
 SELECT * INTO STRICT p FROM pg_proc WHERE oid='vec_bolsa_llamamientos.registrar_intento_borrador_llamamiento_v1(text,text,text,text,text)'::regprocedure;
 IF p.proowner IS DISTINCT FROM 'vec_bolsa_llamamientos_propietario'::regrole OR p.prosecdef IS NOT TRUE
 OR p.proconfig @> ARRAY['search_path=pg_catalog, pg_temp','statement_timeout=5s'] IS NOT TRUE
 OR encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex') IS DISTINCT FROM '624338d1b7bc30a3a9e29ef6138e0128230d53d45d1f1da67f2c603aac64f99e'
 OR EXISTS(SELECT 1 FROM aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
  WHERE a.grantee<>p.proowner AND (a.grantee<>'vec_bolsa_llamamientos_registrador_frontera'::regrole
   OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
 OR has_function_privilege('vec_bolsa_llamamientos_registrador_frontera',p.oid,'EXECUTE') IS NOT TRUE
 THEN RAISE EXCEPTION 'B134: posimagen incompatible' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
