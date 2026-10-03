\set ON_ERROR_STOP on
-- AD164 conserva AD160 y su historia. La provisión seguirá cerrada hasta
-- integrar auditoría común nominal y categoría Aplicación del catálogo real.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion:cargos_ct:provision:v1',0));

DO $preimagen$
DECLARE f record; tabla text; funcion regprocedure; propietario oid;
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='vec_autorizacion_propietario'
   AND NOT(rolcanlogin OR rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls))
 OR NOT EXISTS(SELECT 1 FROM pg_namespace WHERE nspname='vec_autorizacion'
   AND nspowner=to_regrole('vec_autorizacion_propietario'))
 OR to_regprocedure('vec_autorizacion.comprobar_independencia_cargo_ct_v1(text,bytea,text,text,boolean)') IS NOT NULL
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='vec_autorizacion_cargos_ct_ejecutor'
   AND NOT(rolcanlogin OR rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls))
 THEN RAISE EXCEPTION 'AD164: preimagen incompatible' USING ERRCODE='55000'; END IF;
 propietario:='vec_autorizacion_propietario'::regrole;
 FOREACH tabla IN ARRAY ARRAY['cargo_ct_plan','cargo_ct_aprobacion','cargo_ct_consumo','cargo_ct_auditoria','cargo_ct_recibo'] LOOP
  IF NOT EXISTS(SELECT 1 FROM pg_class c WHERE c.oid=to_regclass('vec_autorizacion.'||tabla)
    AND c.relowner=propietario AND c.relrowsecurity AND c.relforcerowsecurity)
  OR EXISTS(SELECT 1 FROM pg_class c CROSS JOIN LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) a
    WHERE c.oid=to_regclass('vec_autorizacion.'||tabla) AND a.grantee<>propietario)
  THEN RAISE EXCEPTION 'AD164: tabla o ACL divergente %',tabla USING ERRCODE='55000'; END IF;
 END LOOP;
 FOR f IN SELECT * FROM (VALUES
  ('operar_cargo_ct_interna_v1(text,bytea,bytea,bytea,numeric,numeric)','a2fbeee1e9e5a5112a4ef623938b21e83cee2f15694e96602b31ce16787e2ec8',NULL::text),
  ('acreditar_adjunto_plan_cargo_ct_v1(text,text,text,text)','f1c551630b9fb6303f57308eb3999ff5b0746f6587a5211c18ab7ee6f1b619bb','vec_contexto_actor_v1_propietario'),
  ('preparar_plan_cargo_ct_v1(bytea,bytea,bytea,numeric,numeric)',NULL,'vec_autorizacion_cargos_ct_ejecutor'),
  ('aprobar_plan_cargo_ct_v1(bytea,bytea,bytea,numeric,numeric)',NULL,'vec_autorizacion_cargos_ct_ejecutor'),
  ('aplicar_plan_cargo_ct_v1(bytea,bytea,bytea,numeric,numeric)',NULL,'vec_autorizacion_cargos_ct_ejecutor'),
  ('recuperar_plan_cargo_ct_v1(bytea,bytea,bytea,numeric,numeric)',NULL,'vec_autorizacion_cargos_ct_ejecutor')
 ) AS esperada(firma,huella,ejecutor) LOOP
  funcion:=to_regprocedure('vec_autorizacion.'||f.firma);
  IF funcion IS NULL OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=funcion
    AND p.proowner=propietario AND p.prosecdef AND p.provolatile='v'
    AND p.proconfig @> ARRAY['search_path=pg_catalog']
    AND (f.huella IS NULL OR encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')=f.huella))
  OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
    WHERE p.oid=funcion AND (a.privilege_type<>'EXECUTE' OR
     (a.grantee<>propietario AND (f.ejecutor IS NULL OR a.grantee<>to_regrole(f.ejecutor) OR a.is_grantable))))
  THEN RAISE EXCEPTION 'AD164: función o ACL divergente %',f.firma USING ERRCODE='55000'; END IF;
 END LOOP;
END $preimagen$;

SET LOCAL ROLE vec_autorizacion_propietario;
-- Esta comprobación acredita sólo la independencia de personas del registro
-- durable. No autoriza una operación ni sustituye PDP, sesión o auditoría.
CREATE FUNCTION vec_autorizacion.comprobar_independencia_cargo_ct_v1(
 p_clave text,p_plan bytea,p_huella text,p_aprobador_actual text,p_exigir_aprobacion boolean
) RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE plan record; aprobacion record; destino text; documento jsonb;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off' OR p_exigir_aprobacion IS NULL THEN
  RAISE EXCEPTION 'cargo_ct_rechazado' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT plan FROM vec_autorizacion.cargo_ct_plan WHERE clave=p_clave FOR SHARE;
 IF plan.plan_canonico IS DISTINCT FROM p_plan OR plan.plan_sha256 IS DISTINCT FROM p_huella
 OR encode(sha256(p_plan),'hex') IS DISTINCT FROM p_huella
 OR vec_autorizacion.texto_positivo_valido(plan.proponente_ref,512) IS NOT TRUE THEN
  RAISE EXCEPTION 'cargo_ct_rechazado' USING ERRCODE='23514'; END IF;
 documento:=convert_from(plan.plan_canonico,'UTF8')::jsonb;
 destino:=convert_from(decode(documento->>'asignacion_canonica','base64'),'UTF8')::jsonb->>'principal_id';
 IF vec_autorizacion.texto_positivo_valido(destino,512) IS NOT TRUE
 OR plan.proponente_ref IS NOT DISTINCT FROM destino
 OR (p_aprobador_actual IS NOT NULL AND
   (vec_autorizacion.texto_positivo_valido(p_aprobador_actual,512) IS NOT TRUE
    OR p_aprobador_actual IS NOT DISTINCT FROM plan.proponente_ref
    OR p_aprobador_actual IS NOT DISTINCT FROM destino)) THEN
  RAISE EXCEPTION 'cargo_ct_rechazado' USING ERRCODE='42501'; END IF;
 SELECT * INTO aprobacion FROM vec_autorizacion.cargo_ct_aprobacion WHERE clave=p_clave FOR SHARE;
 IF NOT FOUND THEN
  IF p_exigir_aprobacion THEN RAISE EXCEPTION 'cargo_ct_rechazado' USING ERRCODE='42501'; END IF;
  RETURN true;
 END IF;
 IF aprobacion.plan_sha256 IS DISTINCT FROM plan.plan_sha256
 OR vec_autorizacion.texto_positivo_valido(aprobacion.aprobador_ref,512) IS NOT TRUE
 OR aprobacion.aprobador_ref IS NOT DISTINCT FROM plan.proponente_ref
 OR aprobacion.aprobador_ref IS NOT DISTINCT FROM destino
 OR convert_from(aprobacion.decision_canonica,'UTF8')::jsonb->>'principal_id' IS DISTINCT FROM aprobacion.aprobador_ref
 OR (p_aprobador_actual IS NOT NULL AND aprobacion.aprobador_ref IS DISTINCT FROM p_aprobador_actual) THEN
  RAISE EXCEPTION 'cargo_ct_rechazado' USING ERRCODE='42501'; END IF;
 RETURN true;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.comprobar_independencia_cargo_ct_v1(text,bytea,text,text,boolean)
 FROM PUBLIC,vec_autorizacion_cargos_ct_ejecutor;

-- La preimagen anterior fija los cuerpos exactos. Cada marca ha de aparecer
-- una sola vez; no se modifica el fichero AD160 ni se reescribe su historia.
DO $guardas$
DECLARE firma text; definicion text; marca text; reemplazo text; item record;
BEGIN
 FOR item IN SELECT * FROM (VALUES
  ('operar_cargo_ct_interna_v1(text,bytea,bytea,bytea,numeric,numeric)',
   $marca$BEGIN
 -- Frontera técnica no configurable por el plan.$marca$,
   $reemplazo$BEGIN
 -- Cierre incondicional: ni heredar el propietario ni SET ROLE abre el efecto.
 RAISE EXCEPTION 'cargo_ct_rechazado' USING ERRCODE='42501';
 -- Frontera técnica no configurable por el plan.$reemplazo$),
  ('acreditar_adjunto_plan_cargo_ct_v1(text,text,text,text)',
   $marca$BEGIN
 SELECT * INTO STRICT p FROM vec_autorizacion.cargo_ct_plan$marca$,
   $reemplazo$BEGIN
 -- El adjunto tampoco habilita CA24 mientras falta el contrato común.
 RAISE EXCEPTION 'cargo_ct_rechazado' USING ERRCODE='42501';
 SELECT * INTO STRICT p FROM vec_autorizacion.cargo_ct_plan$reemplazo$),
  ('operar_cargo_ct_interna_v1(text,bytea,bytea,bytea,numeric,numeric)',
   '   -- Replay: autoriza ANTES de conocer el recibo y mantiene sus bytes/fecha.',
   E'   PERFORM vec_autorizacion.comprobar_independencia_cargo_ct_v1(clave,b,h,NULL,true);\n   -- Replay: autoriza ANTES de conocer el recibo y mantiene sus bytes/fecha.'),
  ('operar_cargo_ct_interna_v1(text,bytea,bytea,bytea,numeric,numeric)',
   $marca$    IF op='aprobar' THEN
$marca$,
   $reemplazo$    IF op='aprobar' THEN
     IF vec_autorizacion.comprobar_independencia_cargo_ct_v1(clave,b,h,actor,false) IS NOT TRUE THEN
      RAISE EXCEPTION 'cargo_ct_rechazado' USING ERRCODE='42501'; END IF;
$reemplazo$),
  ('operar_cargo_ct_interna_v1(text,bytea,bytea,bytea,numeric,numeric)',
   E'     IF NOT FOUND OR ap.plan_sha256<>h\n',
   $reemplazo$     IF vec_autorizacion.comprobar_independencia_cargo_ct_v1(clave,b,h,NULL,true) IS NOT TRUE THEN
      RAISE EXCEPTION 'cargo_ct_rechazado' USING ERRCODE='42501'; END IF;
     IF NOT FOUND OR ap.plan_sha256<>h
$reemplazo$),
  ('acreditar_adjunto_plan_cargo_ct_v1(text,text,text,text)',
   $marca$ x:=pg_catalog.convert_from(p.plan_canonico,'UTF8')::jsonb;$marca$,
   $reemplazo$ PERFORM vec_autorizacion.comprobar_independencia_cargo_ct_v1(k,p.plan_canonico,h,NULL,true);
 x:=pg_catalog.convert_from(p.plan_canonico,'UTF8')::jsonb;$reemplazo$)
 ) AS cambios(firma,marca,reemplazo) LOOP
  firma:='vec_autorizacion.'||item.firma;
  definicion:=pg_get_functiondef(firma::regprocedure);
  marca:=item.marca; reemplazo:=item.reemplazo;
  IF (length(definicion)-length(replace(definicion,marca,'')))/length(marca)<>1 THEN
   RAISE EXCEPTION 'AD164: marca de guarda divergente %',firma USING ERRCODE='55000'; END IF;
  EXECUTE replace(definicion,marca,reemplazo);
 END LOOP;
END $guardas$;

-- Cierre incondicional del efecto y del acceso SQL heredado. No se abre por
-- detectar una función o por recibir un rol, categoría o aprobador del cliente.
DO $cierre$
DECLARE nombre text; funcion regprocedure;
BEGIN
 FOREACH nombre IN ARRAY ARRAY['preparar','aprobar','aplicar','recuperar'] LOOP
  EXECUTE format('CREATE OR REPLACE FUNCTION vec_autorizacion.%I(b bytea,d bytea,m bytea,pv numeric,fv numeric) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $cerrada$ BEGIN RAISE EXCEPTION ''cargo_ct_rechazado'' USING ERRCODE=''42501''; END $cerrada$',nombre||'_plan_cargo_ct_v1');
  funcion:=to_regprocedure('vec_autorizacion.'||nombre||'_plan_cargo_ct_v1(bytea,bytea,bytea,numeric,numeric)');
  EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC,vec_autorizacion_cargos_ct_ejecutor',funcion);
  IF EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
    WHERE p.oid=funcion AND a.grantee<>p.proowner)
  OR has_function_privilege('vec_autorizacion_cargos_ct_ejecutor',funcion,'EXECUTE') THEN
   RAISE EXCEPTION 'AD164: fachada conserva concesión directa ajena %',nombre USING ERRCODE='55000'; END IF;
 END LOOP;
END $cierre$;
DO $comprobacion_privada$
BEGIN
 IF EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
   WHERE p.oid='vec_autorizacion.comprobar_independencia_cargo_ct_v1(text,bytea,text,text,boolean)'::regprocedure
   AND a.grantee<>p.proowner)
 OR has_function_privilege('vec_autorizacion_cargos_ct_ejecutor',
   'vec_autorizacion.comprobar_independencia_cargo_ct_v1(text,bytea,text,text,boolean)','EXECUTE') THEN
  RAISE EXCEPTION 'AD164: comprobación privada con concesión directa ajena' USING ERRCODE='55000'; END IF;
END $comprobacion_privada$;
-- Se conservan las autoridades históricas y sus ACL. Un dueño con capacidad
-- DDL puede cambiar una función; este corte cierra los cuerpos publicados,
-- no promete aislar al administrador del esquema de su propia autoridad.
COMMIT;
