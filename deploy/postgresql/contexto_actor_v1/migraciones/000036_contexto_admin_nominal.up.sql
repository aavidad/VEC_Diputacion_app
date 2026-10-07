\set ON_ERROR_STOP on
-- CA36: contexto ADMIN con runtime segregado.
-- Orden: AD194 -> IS16 -> CA36 (deploy/principal/lista_sql_codexk_admin_runtime_20261005.txt).
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec:admin:continuidad:v1',0));
DO $pre$
DECLARE r record;n integer:=0;
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 OR to_regrole('vec_contexto_actor_v1_admin_contexto') IS NOT NULL
 OR to_regprocedure('vec_identidad_sesiones_v1.cotejar_vinculo_sesion_admin_v1(text,numeric,text,text,text,text,text,timestamptz,text)') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_contexto_admin_pre_v2_ca_v1(jsonb)') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.cotejar_contexto_admin_pre_v2_ca_v1(jsonb)') IS NULL
 OR to_regprocedure('vec_contexto_actor_v1.listar_admin_preperfil_propietaria_v1(text,text,text)') IS NULL
 OR to_regprocedure('vec_contexto_actor_v1.resolver_contexto_ca36_interno_v1(text,text,text,text,text,text,timestamptz,text[])') IS NOT NULL
 OR to_regprocedure('vec_contexto_actor_v1.reconciliar_contexto_ca36_interno_v1(text,text,text,text,text,text,timestamptz,text[])') IS NOT NULL
 THEN RAISE EXCEPTION 'CA36: preimagen incompatible' USING ERRCODE='55000'; END IF;
 FOR r IN SELECT p.oid,p.proowner,p.prosecdef,p.proconfig,
    ARRAY(SELECT a.acl::text FROM unnest(p.proacl) AS a(acl) ORDER BY a.acl::text) AS acl_texto,
    encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') AS fuente_sha,
    encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex') AS definicion_sha
   FROM pg_proc p WHERE p.oid IN(
    to_regprocedure('vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2(text,text,text,text,text,text,timestamptz,text[])'),
    to_regprocedure('vec_contexto_actor_v1.reconciliar_contexto_actor_v2(text,text,text,text,text,text,timestamptz,text[])')) LOOP
  n:=n+1;
  IF r.proowner IS DISTINCT FROM to_regrole('vec_contexto_actor_v1_propietario')
   OR r.prosecdef IS NOT TRUE OR r.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog']::text[]
   OR r.acl_texto IS DISTINCT FROM ARRAY[
     'vec_contexto_actor_v1_propietario=X/vec_contexto_actor_v1_propietario',
     'vec_contexto_actor_v1_runtime=X/vec_contexto_actor_v1_propietario']::text[]
   OR (r.oid=to_regprocedure('vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2(text,text,text,text,text,text,timestamptz,text[])')
      AND (r.fuente_sha IS DISTINCT FROM '903fa70a63e86ccb3cf71090b87e948288e8dde1958652ebf032f7050634496c'
       OR r.definicion_sha IS DISTINCT FROM '7ef8569f502fbd5b8e5562c83b3720f7385e972fa9178b84483759c807d8f88c'))
   OR (r.oid=to_regprocedure('vec_contexto_actor_v1.reconciliar_contexto_actor_v2(text,text,text,text,text,text,timestamptz,text[])')
      AND (r.fuente_sha IS DISTINCT FROM 'cff0c40236bb2ba32dac5cbf422e41a0b35e5184526652adf76ee86b5e136fb1'
       OR r.definicion_sha IS DISTINCT FROM 'c0f78c8896672dd167cbe509d80bbe3c38941f38a4b0d958b90c201f059db14f'))
  THEN RAISE EXCEPTION 'CA36: núcleo V2 divergente' USING ERRCODE='55000'; END IF;
 END LOOP;
 IF n<>2 THEN RAISE EXCEPTION 'CA36: núcleo V2 incompleto' USING ERRCODE='55000'; END IF;
 SELECT p.proowner,p.prosecdef,p.proconfig,
  ARRAY(SELECT a.acl::text FROM unnest(p.proacl) AS a(acl) ORDER BY a.acl::text) AS acl_texto,
  encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') AS fuente_sha,
  encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex') AS definicion_sha
 INTO r FROM pg_proc p WHERE p.oid=to_regprocedure('vec_contexto_actor_v1.exigir_runtime_contexto_actor_v1()');
 IF NOT FOUND OR r.proowner IS DISTINCT FROM to_regrole('vec_contexto_actor_v1_propietario')
  OR r.prosecdef IS NOT TRUE OR r.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog']::text[]
  OR r.acl_texto IS DISTINCT FROM ARRAY['vec_contexto_actor_v1_propietario=X/vec_contexto_actor_v1_propietario']::text[]
  OR r.fuente_sha IS DISTINCT FROM '005fff9328377a75bcde2a23e997959f2d1603f658ae39a2e6f2eb8d8986d3c8'
  OR r.definicion_sha IS DISTINCT FROM 'dbaa84ba1a9878e1410c38cf5e8eb25d25ceabed16ecff49a406dce0cf92c79c'
 THEN RAISE EXCEPTION 'CA36: runtime general divergente' USING ERRCODE='55000'; END IF;
END $pre$;
-- AD192 tal como se instaló falla al repetir un evento (secuencia numeric en
-- una salida bigint). AD194 lo corrige. Se exige que la cadena defectuosa ya
-- no esté en vigor sin depender del texto exacto de la corrección: basta con
-- que la fachada ya no llame a la interna de AD192 o que esta haya cambiado.
DO $ad194$ BEGIN
 IF EXISTS(SELECT 1 FROM(VALUES
   ('registrar_contexto_admin_pre_v2_ca_v1(jsonb)','registrar_contexto_admin_pre_v2_interna_v1','90219f669dccc426c8ba66958937bb0a85d028ae515868b40cdea788aec2fdf1'),
   ('cotejar_contexto_admin_pre_v2_ca_v1(jsonb)','cotejar_contexto_admin_pre_v2_interna_v1','ff9c375abc732eb4ace4d2bc942b130cabe7623c9874e63802798aa450892b4c')) d(fachada,interna,sha_ad192)
  JOIN pg_proc f ON f.oid=to_regprocedure('vec_autorizacion_atestada_v3.'||d.fachada)
  JOIN pg_proc i ON i.oid=to_regprocedure('vec_autorizacion_atestada_v3.'||d.interna||'(jsonb,text)')
  WHERE position(d.interna||'(' IN f.prosrc)>0 AND encode(sha256(convert_to(i.prosrc,'UTF8')),'hex')=d.sha_ad192)
 THEN RAISE EXCEPTION 'CA36: falta AD194 (repetición AD192 sin corregir)' USING ERRCODE='55000'; END IF;
END $ad194$;
CREATE ROLE vec_contexto_actor_v1_admin_contexto NOLOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
DO $base$ BEGIN
 EXECUTE format('GRANT CONNECT ON DATABASE %I TO vec_contexto_actor_v1_admin_contexto',current_database());
END $base$;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
GRANT USAGE ON SCHEMA vec_contexto_actor_v1 TO vec_contexto_actor_v1_admin_contexto;
-- Extrae exactamente el cuerpo medido; las entradas generales retienen el
-- chequeo de su runtime. Los helpers nuevos sólo pertenecen al owner CA.
DO $extraer$
DECLARE original oid;definicion text;cabecera text;nombre text;marca text;
BEGIN
 FOREACH original IN ARRAY ARRAY[
  to_regprocedure('vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2(text,text,text,text,text,text,timestamptz,text[])'),
  to_regprocedure('vec_contexto_actor_v1.reconciliar_contexto_actor_v2(text,text,text,text,text,text,timestamptz,text[])')
 ] LOOP
  definicion:=pg_get_functiondef(original);
  IF original=to_regprocedure('vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2(text,text,text,text,text,text,timestamptz,text[])') THEN
   cabecera:='CREATE OR REPLACE FUNCTION vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2(';
   nombre:='CREATE FUNCTION vec_contexto_actor_v1.resolver_contexto_ca36_interno_v1(';
  ELSE
   cabecera:='CREATE OR REPLACE FUNCTION vec_contexto_actor_v1.reconciliar_contexto_actor_v2(';
   nombre:='CREATE FUNCTION vec_contexto_actor_v1.reconciliar_contexto_ca36_interno_v1(';
  END IF;
  marca:='    PERFORM vec_contexto_actor_v1.exigir_runtime_contexto_actor_v1();';
  IF left(definicion,length(cabecera)) IS DISTINCT FROM cabecera
   OR (length(definicion)-length(replace(definicion,cabecera,''))) IS DISTINCT FROM length(cabecera)
   OR (length(definicion)-length(replace(definicion,marca,''))) IS DISTINCT FROM length(marca)
  THEN RAISE EXCEPTION 'CA36: extracción privada no exacta' USING ERRCODE='55000'; END IF;
  EXECUTE replace(replace(definicion,cabecera,nombre),marca,'');
 END LOOP;
END $extraer$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.resolver_contexto_ca36_interno_v1(text,text,text,text,text,text,timestamptz,text[]),
 vec_contexto_actor_v1.reconciliar_contexto_ca36_interno_v1(text,text,text,text,text,text,timestamptz,text[]) FROM PUBLIC;
-- Enlace de dominio: conserva el frame propietario que autorizó esta fila
-- del núcleo. La cadena y el acuse de auditoría permanecen sólo en AD192.
CREATE TABLE vec_contexto_actor_v1.enlace_contexto_admin_v1 (
 operacion_ref text PRIMARY KEY REFERENCES vec_contexto_actor_v1.registros_contexto(operacion_ref),
 registro_contexto_ref text NOT NULL UNIQUE REFERENCES vec_contexto_actor_v1.registros_contexto(registro_contexto_ref),
 vinculo_sesion_ref text NOT NULL CHECK(vinculo_sesion_ref ~ '^vis_[0-9a-f]{32}$'),
 vinculo_sesion_version numeric(20,0) NOT NULL CHECK(vinculo_sesion_version BETWEEN 1 AND 18446744073709551615),
 vinculo_sesion_sha256 text NOT NULL CHECK(vinculo_sesion_sha256 ~ '^[0-9a-f]{64}$'),
 autenticacion_ref text NOT NULL CHECK(vec_contexto_actor_v1.referencia_valida(autenticacion_ref,'aut_')),
 sesion_ref text NOT NULL CHECK(vec_contexto_actor_v1.referencia_valida(sesion_ref,'ses_')),
 cuenta_ref text NOT NULL CHECK(vec_contexto_actor_v1.referencia_valida(cuenta_ref,'cta_')),
 actor_ref text NOT NULL CHECK(vec_contexto_actor_v1.referencia_valida(actor_ref,'per_')),
 perfil_ref text NOT NULL CHECK(vec_contexto_actor_v1.referencia_valida(perfil_ref,'prf_')),
 fuente_ref text NOT NULL CHECK(fuente_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,159}$'),
 fuente_sha256 text NOT NULL CHECK(fuente_sha256 ~ '^[0-9a-f]{64}$'),
 evento_ref text NOT NULL UNIQUE CHECK(evento_ref ~ '^evento_[0-9a-f]{32}$'),
 proceso text NOT NULL CHECK(proceso ~ '^[a-z][a-z0-9._-]{1,79}$'),
 correlacion_ref text NOT NULL CHECK(correlacion_ref ~ '^correlacion_[0-9a-f]{32}$'),
 motivo_ref text NOT NULL CHECK(motivo_ref='contexto_admin_pre_v2_permitido'),
 motivo_catalogo_version smallint NOT NULL CHECK(motivo_catalogo_version=1),
 enlazado_en timestamptz NOT NULL CHECK(isfinite(enlazado_en))
);
ALTER TABLE vec_contexto_actor_v1.enlace_contexto_admin_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contexto_actor_v1.enlace_contexto_admin_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_exacto ON vec_contexto_actor_v1.enlace_contexto_admin_v1
 TO vec_contexto_actor_v1_propietario USING(current_user='vec_contexto_actor_v1_propietario')
 WITH CHECK(current_user='vec_contexto_actor_v1_propietario');
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_contexto_actor_v1.enlace_contexto_admin_v1
 FOR EACH ROW EXECUTE FUNCTION vec_contexto_actor_v1.rechazar_mutacion_historia();
CREATE TRIGGER historia_no_truncable BEFORE TRUNCATE ON vec_contexto_actor_v1.enlace_contexto_admin_v1
 FOR EACH STATEMENT EXECUTE FUNCTION vec_contexto_actor_v1.rechazar_truncado();
CREATE FUNCTION vec_contexto_actor_v1.validar_enlace_contexto_admin_v1()
RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $f$
DECLARE r record;canon jsonb;
BEGIN
 SELECT x.* INTO STRICT r FROM vec_contexto_actor_v1.registros_contexto x
 WHERE x.operacion_ref=NEW.operacion_ref AND x.registro_contexto_ref=NEW.registro_contexto_ref FOR SHARE;
 canon:=convert_from(r.representacion_canonica,'UTF8')::jsonb;
 IF r.cuenta_ref IS DISTINCT FROM NEW.cuenta_ref OR r.perfil_ref IS DISTINCT FROM NEW.perfil_ref
  OR r.metodo NOT IN('certificado','dnie') OR r.garantia IS DISTINCT FROM 'alto'
  OR NEW.enlazado_en<r.resuelto_en
  OR canon->>'persona_ref' IS DISTINCT FROM NEW.actor_ref
  OR canon->>'perfil_activo_ref' IS DISTINCT FROM NEW.perfil_ref
  OR canon->>'cuenta_ref' IS DISTINCT FROM NEW.cuenta_ref
  OR vec_contexto_actor_v1.alcance_registro_contexto_v2(r.representacion_canonica) IS DISTINCT FROM '{}'::text[]
 THEN RAISE EXCEPTION 'CA36: enlace incompatible con contexto' USING ERRCODE='42501'; END IF;
 RETURN NEW;
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.validar_enlace_contexto_admin_v1() FROM PUBLIC;
CREATE TRIGGER enlace_exacto BEFORE INSERT ON vec_contexto_actor_v1.enlace_contexto_admin_v1
 FOR EACH ROW EXECUTE FUNCTION vec_contexto_actor_v1.validar_enlace_contexto_admin_v1();
REVOKE ALL ON TABLE vec_contexto_actor_v1.enlace_contexto_admin_v1 FROM PUBLIC;
REVOKE ALL ON TYPE vec_contexto_actor_v1.enlace_contexto_admin_v1 FROM PUBLIC;
CREATE FUNCTION vec_contexto_actor_v1.cotejar_contexto_admin_actual_v1(
 p_cuenta text,p_persona text,p_perfil text,p_revision numeric,p_hasta timestamptz,p_canon bytea)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET timezone='UTC' AS $f$
DECLARE s record;c record;pe record;pf record;v record;j jsonb;ahora timestamptz;
BEGIN
 IF current_setting('transaction_read_only')<>'off' OR current_setting('transaction_isolation') NOT IN('serializable','read committed')
  OR vec_contexto_actor_v1.referencia_valida(p_cuenta,'cta_') IS NOT TRUE
  OR vec_contexto_actor_v1.referencia_valida(p_persona,'per_') IS NOT TRUE
  OR vec_contexto_actor_v1.referencia_valida(p_perfil,'prf_') IS NOT TRUE
  OR p_revision IS NULL OR p_revision<>trunc(p_revision) OR p_revision NOT BETWEEN 1 AND 18446744073709551615::numeric
  OR vec_contexto_actor_v1.instante_valido(p_hasta) IS NOT TRUE OR p_canon IS NULL OR octet_length(p_canon) NOT BETWEEN 1 AND 65536
 THEN RETURN false; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec:admin:continuidad:v1',0));
 SELECT x.* INTO s FROM vec_contexto_actor_v1.seleccion_admin_actual_auditada_v1 a
 JOIN vec_contexto_actor_v1.seleccion_admin_auditada_v1 x USING(cuenta_ref,revision)
 WHERE a.cuenta_ref=p_cuenta FOR SHARE OF a;
 IF NOT FOUND OR s.revision IS DISTINCT FROM p_revision OR s.persona_ref IS DISTINCT FROM p_persona
  OR s.perfil_ref IS DISTINCT FROM p_perfil THEN RETURN false; END IF;
 SELECT x.* INTO c FROM vec_contexto_actor_v1.proyeccion_cuenta_actual a
 JOIN vec_contexto_actor_v1.proyeccion_cuenta_versiones x USING(cuenta_ref,version)
 WHERE a.cuenta_ref=p_cuenta FOR SHARE OF a;
 IF NOT FOUND THEN RETURN false; END IF;
 SELECT x.* INTO pe FROM vec_contexto_actor_v1.persona_actual a
 JOIN vec_contexto_actor_v1.persona_versiones x USING(persona_ref,version)
 WHERE a.persona_ref=p_persona FOR SHARE OF a;
 IF NOT FOUND THEN RETURN false; END IF;
 SELECT x.* INTO pf FROM vec_contexto_actor_v1.perfil_actual a
 JOIN vec_contexto_actor_v1.perfil_versiones x USING(perfil_ref,version)
 WHERE a.perfil_ref=p_perfil FOR SHARE OF a;
 IF NOT FOUND THEN RETURN false; END IF;
 SELECT x.* INTO v FROM vec_contexto_actor_v1.vinculo_contexto_actual a
 JOIN vec_contexto_actor_v1.vinculo_contexto_versiones x USING(vinculo_ref,version)
 WHERE a.vinculo_ref=s.vinculo_ref FOR SHARE OF a;
 IF NOT FOUND THEN RETURN false; END IF;
 j:=convert_from(p_canon,'UTF8')::jsonb;
 ahora:=clock_timestamp();
 RETURN jsonb_typeof(j)='object' AND j->>'esquema'='vec.contexto-actor.vinculado.v2'
  AND j->>'cuenta_ref'=p_cuenta AND j->>'persona_ref'=p_persona AND j->>'perfil_activo_ref'=p_perfil
  AND (j->>'cuenta_version')::numeric IS NOT DISTINCT FROM c.version
  AND (j->>'persona_version')::numeric IS NOT DISTINCT FROM pe.version
  AND (j->>'perfil_version')::numeric IS NOT DISTINCT FROM pf.version
  AND (j->>'contexto_version')::numeric IS NOT DISTINCT FROM v.version
  AND j->>'contexto_actor_ref'=v.vinculo_ref
  AND v.cuenta_ref=p_cuenta AND v.persona_ref=p_persona AND v.perfil_ref=p_perfil
  AND pf.persona_ref=p_persona
  AND c.estado='activo' AND pe.estado='activo' AND pf.estado='activo' AND v.estado='activo'
  AND c.procedencia_autoridad='autoridad_maestra_acreditada'
  AND pe.procedencia_autoridad='autoridad_maestra_acreditada'
  AND pf.procedencia_autoridad='autoridad_maestra_acreditada'
  AND v.procedencia_autoridad='autoridad_maestra_acreditada'
  AND ahora>=GREATEST(c.vigente_desde,pe.vigente_desde,pf.vigente_desde,v.vigente_desde)
  AND ahora<LEAST(c.vigente_hasta,pe.vigente_hasta,pf.vigente_hasta,v.vigente_hasta,p_hasta);
EXCEPTION WHEN data_exception THEN RETURN false;
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.cotejar_contexto_admin_actual_v1(text,text,text,numeric,timestamptz,bytea) FROM PUBLIC;
CREATE FUNCTION vec_contexto_actor_v1.evento_contexto_admin_pre_v2_v1(
 p_evento text,p_correlacion text,p_proceso text,p_accion text,p_recurso text,p_resultado text,
 p_actor text,p_perfil text,p_fuente text,p_fuente_sha text)
RETURNS jsonb LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE motivo text;
BEGIN
 IF p_evento !~ '^evento_[0-9a-f]{32}$' OR p_correlacion !~ '^correlacion_[0-9a-f]{32}$'
  OR p_proceso !~ '^[a-z][a-z0-9._-]{1,79}$' OR p_recurso !~ '^oca_[A-Za-z0-9_-]{22,128}$'
  OR p_accion NOT IN('registrar_contexto_admin','reconciliar_contexto_admin')
  OR p_resultado NOT IN('permitido','denegado','error')
  OR ((p_fuente IS NULL) IS DISTINCT FROM (p_fuente_sha IS NULL))
  OR (p_actor IS NOT NULL AND p_fuente IS NULL)
  OR (p_perfil IS NOT NULL AND p_actor IS NULL)
  OR (p_resultado='permitido' AND (p_actor IS NULL OR p_perfil IS NULL OR p_fuente IS NULL))
 THEN RAISE EXCEPTION 'CA36: evento PRE-V2 inválido' USING ERRCODE='22023'; END IF;
 motivo:=CASE p_resultado WHEN 'permitido' THEN 'contexto_admin_pre_v2_permitido'
  WHEN 'denegado' THEN 'contexto_admin_pre_v2_denegado'
  ELSE 'contexto_admin_pre_v2_error' END;
 RETURN jsonb_build_object('tipo_registro','contexto_admin_pre_v2','evento_ref',p_evento,
  'operador_login',session_user::text,'actor_ref',p_actor,'perfil_activo_ref',p_perfil,
  'accion',p_accion,'recurso_ref',p_recurso,'resultado',p_resultado,'motivo_ref',motivo,
  'proceso',p_proceso,'canal','administracion_privilegiada','finalidad_ref','establecer_contexto_admin',
  'correlacion_ref',p_correlacion,'fuente_ref',p_fuente,'fuente_sha256',p_fuente_sha);
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.evento_contexto_admin_pre_v2_v1(text,text,text,text,text,text,text,text,text,text) FROM PUBLIC;
CREATE OR REPLACE FUNCTION vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2(
 p_operacion_ref text,p_registro_contexto_ref text,p_cuenta_ref text,p_perfil_ref text,
 p_metodo text,p_garantia text,p_solicitado_en timestamptz,p_proyecciones text[])
RETURNS TABLE(operacion_ref text,registro_contexto_ref text,representacion_canonica bytea,
 huella_sha256 text,manifiesto_procedencia_canonico bytea,
 manifiesto_procedencia_huella_sha256 text,autoridad_efectiva text,resuelto_en timestamptz)
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
BEGIN
 PERFORM vec_contexto_actor_v1.exigir_runtime_contexto_actor_v1();
 RETURN QUERY SELECT * FROM vec_contexto_actor_v1.resolver_contexto_ca36_interno_v1(
  p_operacion_ref,p_registro_contexto_ref,p_cuenta_ref,p_perfil_ref,p_metodo,p_garantia,p_solicitado_en,p_proyecciones);
END $f$;
CREATE OR REPLACE FUNCTION vec_contexto_actor_v1.reconciliar_contexto_actor_v2(
 p_operacion_ref text,p_registro_contexto_ref text,p_cuenta_ref text,p_perfil_ref text,
 p_metodo text,p_garantia text,p_solicitado_en timestamptz,p_proyecciones text[])
RETURNS TABLE(operacion_ref text,registro_contexto_ref text,representacion_canonica bytea,
 huella_sha256 text,manifiesto_procedencia_canonico bytea,
 manifiesto_procedencia_huella_sha256 text,autoridad_efectiva text,resuelto_en timestamptz)
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
BEGIN
 PERFORM vec_contexto_actor_v1.exigir_runtime_contexto_actor_v1();
 RETURN QUERY SELECT * FROM vec_contexto_actor_v1.reconciliar_contexto_ca36_interno_v1(
  p_operacion_ref,p_registro_contexto_ref,p_cuenta_ref,p_perfil_ref,p_metodo,p_garantia,p_solicitado_en,p_proyecciones);
END $f$;
CREATE FUNCTION vec_contexto_actor_v1.exigir_runtime_contexto_admin_v1()
RETURNS text LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE l record;g record;dbo oid;nso oid;fs oid[];
BEGIN
 SELECT * INTO l FROM pg_roles WHERE rolname=session_user;
 SELECT * INTO g FROM pg_roles WHERE rolname='vec_contexto_actor_v1_admin_contexto';
 dbo:=(SELECT oid FROM pg_database WHERE datname=current_database());
 nso:=to_regnamespace('vec_contexto_actor_v1');
 fs:=ARRAY[
  to_regprocedure('vec_contexto_actor_v1.acreditar_runtime_contexto_admin_v1()'),
  to_regprocedure('vec_contexto_actor_v1.registrar_contexto_admin_v1(text,text,text,text,text,text,timestamptz,text,numeric,text,text,text,text,text,text)'),
  to_regprocedure('vec_contexto_actor_v1.recuperar_contexto_admin_v1(text,text,text,text,text,text,timestamptz,text,numeric,text,text,text,text,text,text,jsonb)'),
  to_regprocedure('vec_contexto_actor_v1.reconciliar_contexto_admin_v1(text,text,text,text,text,text,timestamptz,text,numeric,text,text,text,text,text,text)')];
 IF current_setting('role')<>'none' OR l.oid IS NULL OR g.oid IS NULL OR dbo IS NULL OR nso IS NULL
  OR array_position(fs,NULL) IS NOT NULL OR cardinality(fs)<>4
  OR l.rolcanlogin IS NOT TRUE OR l.rolinherit IS NOT TRUE
  OR l.rolsuper OR l.rolcreaterole OR l.rolcreatedb OR l.rolreplication OR l.rolbypassrls OR l.rolconfig IS NOT NULL
  OR (l.rolvaliduntil IS NOT NULL AND clock_timestamp()>=l.rolvaliduntil)
  OR g.rolcanlogin OR g.rolinherit OR g.rolsuper OR g.rolcreaterole OR g.rolcreatedb OR g.rolreplication OR g.rolbypassrls OR g.rolconfig IS NOT NULL
  OR g.rolvaliduntil IS NOT NULL OR g.rolconnlimit<>-1
  OR (SELECT count(*) FROM pg_auth_members m WHERE m.member=l.oid)<>1
  OR NOT EXISTS(SELECT 1 FROM pg_auth_members m WHERE m.member=l.oid AND m.roleid=g.oid
      AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
  OR EXISTS(SELECT 1 FROM pg_auth_members m WHERE m.member=g.oid)
  OR EXISTS(SELECT 1 FROM pg_db_role_setting s WHERE s.setrole IN(l.oid,g.oid))
  OR EXISTS(SELECT 1 FROM pg_default_acl d LEFT JOIN LATERAL aclexplode(coalesce(d.defaclacl,'{}'::aclitem[])) a ON true
      WHERE d.defaclrole IN(l.oid,g.oid) OR a.grantee IN(l.oid,g.oid) OR a.grantor IN(l.oid,g.oid))
  OR EXISTS(SELECT 1 FROM pg_policy p WHERE l.oid=ANY(p.polroles) OR g.oid=ANY(p.polroles))
  OR EXISTS(SELECT 1 FROM pg_shdepend d WHERE d.refclassid='pg_catalog.pg_authid'::regclass AND d.refobjid=l.oid)
  OR NOT COALESCE((SELECT count(*)=6 AND bool_and(d.deptype='a' AND d.objsubid=0 AND
       ((d.classid='pg_catalog.pg_database'::regclass AND d.objid=dbo) OR
        (d.classid='pg_catalog.pg_namespace'::regclass AND d.objid=nso) OR
        (d.classid='pg_catalog.pg_proc'::regclass AND d.objid=ANY(fs))))
      FROM pg_shdepend d WHERE d.refclassid='pg_catalog.pg_authid'::regclass AND d.refobjid=g.oid),false)
  OR NOT COALESCE((SELECT count(*)=1 AND bool_and(a.privilege_type='CONNECT' AND NOT a.is_grantable)
      FROM pg_database b CROSS JOIN LATERAL aclexplode(coalesce(b.datacl,acldefault('d',b.datdba))) a
      WHERE b.oid=dbo AND a.grantee=g.oid),false)
  OR NOT COALESCE((SELECT count(*)=1 AND bool_and(a.privilege_type='USAGE' AND NOT a.is_grantable)
      FROM pg_namespace n CROSS JOIN LATERAL aclexplode(coalesce(n.nspacl,acldefault('n',n.nspowner))) a
      WHERE n.oid=nso AND a.grantee=g.oid),false)
  OR NOT COALESCE((SELECT count(*)=4 AND count(DISTINCT p.oid)=4 AND bool_and(a.privilege_type='EXECUTE' AND NOT a.is_grantable)
      FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
      WHERE p.oid=ANY(fs) AND a.grantee=g.oid),false)
  OR vec_contexto_actor_v1.privilegios_efectivos_runtime_minimos(l.oid,dbo,nso,fs) IS NOT TRUE
 THEN RAISE EXCEPTION 'CA36: LOGIN contexto ADMIN no acreditado' USING ERRCODE='42501'; END IF;
 RETURN session_user;
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.exigir_runtime_contexto_admin_v1() FROM PUBLIC;
CREATE FUNCTION vec_contexto_actor_v1.acreditar_runtime_contexto_admin_v1()
RETURNS TABLE(identidad_login text,acreditada boolean)
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
BEGIN
 identidad_login:=vec_contexto_actor_v1.exigir_runtime_contexto_admin_v1();
 acreditada:=true;RETURN NEXT;
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.acreditar_runtime_contexto_admin_v1() FROM PUBLIC;
CREATE FUNCTION vec_contexto_actor_v1.registrar_contexto_admin_v1(
 p_operacion text,p_recibo text,p_cuenta text,p_perfil text,p_metodo text,p_garantia text,
 p_solicitado timestamptz,p_vis text,p_vis_version numeric,p_vis_sha text,p_aut text,p_ses text,
 p_evento text,p_correlacion text,p_proceso text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='4s' AS $f$
DECLARE j jsonb;j_final jsonb;r record;l record;a record;a_final record;e jsonb;codigo text;
 clase text;ahora timestamptz;fuente_acreditada boolean:=false;evento_fallo text;evento_previo text;
BEGIN
 PERFORM vec_contexto_actor_v1.exigir_runtime_contexto_admin_v1();
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
  OR p_metodo NOT IN('certificado','dnie') OR p_garantia IS DISTINCT FROM 'alto'
  OR vec_contexto_actor_v1.referencia_operacion_valida(p_operacion,'oca_') IS NOT TRUE
  OR vec_contexto_actor_v1.referencia_operacion_valida(p_recibo,'rca_') IS NOT TRUE
 OR p_evento !~ '^evento_[0-9a-f]{32}$' OR p_correlacion !~ '^correlacion_[0-9a-f]{32}$'
 THEN RAISE EXCEPTION 'CA36: entrada ADMIN inválida' USING ERRCODE='22023'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_contexto_actor_v1:operacion:v2:'||p_operacion,0));
 SELECT x.evento_ref INTO evento_previo FROM vec_contexto_actor_v1.enlace_contexto_admin_v1 x
 WHERE x.operacion_ref=p_operacion;
 BEGIN
  j:=vec_identidad_sesiones_v1.cotejar_vinculo_sesion_admin_v1(
   p_vis,p_vis_version,p_vis_sha,p_aut,p_ses,p_cuenta,p_perfil,p_solicitado,p_metodo);
  IF jsonb_typeof(j) IS DISTINCT FROM 'object' OR (SELECT count(*) FROM jsonb_object_keys(j))<>12
   OR j->>'referencia' IS DISTINCT FROM p_vis OR (j->>'version')::numeric IS DISTINCT FROM p_vis_version
   OR j->>'huella_sha256' IS DISTINCT FROM p_vis_sha OR j->>'autenticacion_ref' IS DISTINCT FROM p_aut
   OR j->>'sesion_ref' IS DISTINCT FROM p_ses OR j->>'cuenta_ref' IS DISTINCT FROM p_cuenta
   OR j->>'perfil_ref' IS DISTINCT FROM p_perfil
   OR vec_contexto_actor_v1.referencia_valida(j->>'persona_ref','per_') IS NOT TRUE
   OR j->>'fuente_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,159}$'
   OR j->>'fuente_sha256' !~ '^[0-9a-f]{64}$'
   OR vec_contexto_actor_v1.instante_valido((j->>'vigente_hasta')::timestamptz) IS NOT TRUE
  THEN RAISE EXCEPTION 'CA36: vínculo IS incompatible' USING ERRCODE='42501'; END IF;
  fuente_acreditada:=true;
  SELECT * INTO l FROM vec_contexto_actor_v1.enlace_contexto_admin_v1
   WHERE operacion_ref=p_operacion FOR SHARE;
  IF FOUND THEN
   IF l.registro_contexto_ref IS DISTINCT FROM p_recibo OR l.vinculo_sesion_ref IS DISTINCT FROM p_vis
    OR l.vinculo_sesion_version IS DISTINCT FROM p_vis_version OR l.vinculo_sesion_sha256 IS DISTINCT FROM p_vis_sha
    OR l.autenticacion_ref IS DISTINCT FROM p_aut OR l.sesion_ref IS DISTINCT FROM p_ses
    OR l.cuenta_ref IS DISTINCT FROM p_cuenta OR l.perfil_ref IS DISTINCT FROM p_perfil
    OR l.actor_ref IS DISTINCT FROM j->>'persona_ref' OR l.fuente_ref IS DISTINCT FROM j->>'fuente_ref'
    OR l.fuente_sha256 IS DISTINCT FROM j->>'fuente_sha256'
    OR l.evento_ref IS DISTINCT FROM p_evento OR l.proceso IS DISTINCT FROM p_proceso
    OR l.correlacion_ref IS DISTINCT FROM p_correlacion
   THEN RAISE EXCEPTION 'CA36: replay de otro vínculo' USING ERRCODE='23505'; END IF;
   SELECT * INTO STRICT r FROM vec_contexto_actor_v1.registros_contexto
    WHERE operacion_ref=p_operacion AND registro_contexto_ref=p_recibo FOR SHARE;
   IF r.solicitado_en IS DISTINCT FROM p_solicitado OR r.metodo IS DISTINCT FROM p_metodo
    OR r.garantia IS DISTINCT FROM p_garantia
    OR vec_contexto_actor_v1.cotejar_contexto_admin_actual_v1(p_cuenta,j->>'persona_ref',p_perfil,
     (j->>'seleccion_revision')::numeric,(j->>'vigente_hasta')::timestamptz,r.representacion_canonica) IS NOT TRUE
   THEN RAISE EXCEPTION 'CA36: replay sin contexto actual' USING ERRCODE='42501'; END IF;
   e:=vec_contexto_actor_v1.evento_contexto_admin_pre_v2_v1(p_evento,p_correlacion,p_proceso,
    'registrar_contexto_admin',p_operacion,'permitido',l.actor_ref,l.perfil_ref,l.fuente_ref,l.fuente_sha256);
   SELECT * INTO STRICT a FROM vec_autorizacion_atestada_v3.cotejar_contexto_admin_pre_v2_ca_v1(e);
   j_final:=vec_identidad_sesiones_v1.cotejar_vinculo_sesion_admin_v1(
    p_vis,p_vis_version,p_vis_sha,p_aut,p_ses,p_cuenta,p_perfil,p_solicitado,p_metodo);
   IF j_final IS DISTINCT FROM j OR vec_contexto_actor_v1.cotejar_contexto_admin_actual_v1(
    p_cuenta,j->>'persona_ref',p_perfil,(j->>'seleccion_revision')::numeric,
    (j->>'vigente_hasta')::timestamptz,r.representacion_canonica) IS NOT TRUE
    OR clock_timestamp()>=(j->>'vigente_hasta')::timestamptz
   THEN RAISE EXCEPTION 'CA36: replay vencido' USING ERRCODE='42501'; END IF;
   RETURN jsonb_build_object('estado','permitido','motivo_ref',e->>'motivo_ref','evento',e,'acuse',to_jsonb(a),
    'contexto',jsonb_build_object('operacion_ref',r.operacion_ref,'registro_contexto_ref',r.registro_contexto_ref,
     'representacion_canonica_base64',encode(r.representacion_canonica,'base64'),'huella_sha256',r.huella_sha256,
     'manifiesto_procedencia_canonico_base64',encode(r.manifiesto_procedencia_canonico,'base64'),
     'manifiesto_procedencia_huella_sha256',r.manifiesto_procedencia_huella_sha256,
     'autoridad_efectiva',r.autoridad_efectiva,'resuelto_en',r.resuelto_en));
  END IF;
  IF EXISTS(SELECT 1 FROM vec_contexto_actor_v1.registros_contexto WHERE operacion_ref=p_operacion)
  THEN RAISE EXCEPTION 'CA36: operación preexistente sin enlace' USING ERRCODE='23505'; END IF;
  -- En SERIALIZABLE el cotejo IS16 usa la fachada CA31/AUT24 y verifica la
  -- selección positiva de Aplicación. Este helper verifica además CA actual.
  SELECT * INTO STRICT r FROM vec_contexto_actor_v1.resolver_contexto_ca36_interno_v1(
   p_operacion,p_recibo,p_cuenta,p_perfil,p_metodo,p_garantia,p_solicitado,'{}'::text[]);
  IF r.operacion_ref IS DISTINCT FROM p_operacion OR r.registro_contexto_ref IS DISTINCT FROM p_recibo
   OR vec_contexto_actor_v1.cotejar_contexto_admin_actual_v1(p_cuenta,j->>'persona_ref',p_perfil,
    (j->>'seleccion_revision')::numeric,(j->>'vigente_hasta')::timestamptz,r.representacion_canonica) IS NOT TRUE
  THEN RAISE EXCEPTION 'CA36: contexto actual incompatible' USING ERRCODE='42501'; END IF;
  e:=vec_contexto_actor_v1.evento_contexto_admin_pre_v2_v1(p_evento,p_correlacion,p_proceso,
   'registrar_contexto_admin',p_operacion,'permitido',j->>'persona_ref',p_perfil,j->>'fuente_ref',j->>'fuente_sha256');
  SELECT * INTO STRICT a FROM vec_autorizacion_atestada_v3.registrar_contexto_admin_pre_v2_ca_v1(e);
  IF a.auditoria_ref IS DISTINCT FROM 'aud_v3_ap2_'||substr(p_evento,8,32)
   OR a.secuencia IS NULL OR a.secuencia<1 OR a.huella_sha256 IS NULL OR a.huella_sha256 !~ '^[0-9a-f]{64}$'
   OR a.correlacion_ref IS DISTINCT FROM p_correlacion OR a.registrada_en IS NULL
  THEN RAISE EXCEPTION 'CA36: acuse AD192 incompatible' USING ERRCODE='42501'; END IF;
  ahora:=clock_timestamp();
  INSERT INTO vec_contexto_actor_v1.enlace_contexto_admin_v1 VALUES(
   p_operacion,p_recibo,p_vis,p_vis_version,p_vis_sha,p_aut,p_ses,p_cuenta,j->>'persona_ref',p_perfil,
   j->>'fuente_ref',j->>'fuente_sha256',p_evento,p_proceso,p_correlacion,e->>'motivo_ref',1,ahora);
  j_final:=vec_identidad_sesiones_v1.cotejar_vinculo_sesion_admin_v1(
   p_vis,p_vis_version,p_vis_sha,p_aut,p_ses,p_cuenta,p_perfil,p_solicitado,p_metodo);
  IF j_final IS DISTINCT FROM j OR vec_contexto_actor_v1.cotejar_contexto_admin_actual_v1(
   p_cuenta,j->>'persona_ref',p_perfil,(j->>'seleccion_revision')::numeric,
   (j->>'vigente_hasta')::timestamptz,r.representacion_canonica) IS NOT TRUE
  THEN RAISE EXCEPTION 'CA36: vigencia perdida tras enlace' USING ERRCODE='42501'; END IF;
  SELECT * INTO STRICT a_final FROM vec_autorizacion_atestada_v3.registrar_contexto_admin_pre_v2_ca_v1(e);
  IF a_final.auditoria_ref IS DISTINCT FROM a.auditoria_ref OR a_final.secuencia IS DISTINCT FROM a.secuencia
   OR a_final.huella_sha256 IS DISTINCT FROM a.huella_sha256 OR a_final.correlacion_ref IS DISTINCT FROM a.correlacion_ref
   OR a_final.registrada_en IS DISTINCT FROM a.registrada_en
   OR clock_timestamp()>=(j->>'vigente_hasta')::timestamptz
  THEN RAISE EXCEPTION 'CA36: acuse o vigencia final divergente' USING ERRCODE='42501'; END IF;
  RETURN jsonb_build_object('estado','permitido','motivo_ref',e->>'motivo_ref','evento',e,'acuse',to_jsonb(a),
   'contexto',jsonb_build_object('operacion_ref',r.operacion_ref,'registro_contexto_ref',r.registro_contexto_ref,
    'representacion_canonica_base64',encode(r.representacion_canonica,'base64'),'huella_sha256',r.huella_sha256,
    'manifiesto_procedencia_canonico_base64',encode(r.manifiesto_procedencia_canonico,'base64'),
    'manifiesto_procedencia_huella_sha256',r.manifiesto_procedencia_huella_sha256,
    'autoridad_efectiva',r.autoridad_efectiva,'resuelto_en',r.resuelto_en));
 EXCEPTION WHEN serialization_failure OR deadlock_detected OR query_canceled THEN RAISE;
 WHEN OTHERS THEN
  GET STACKED DIAGNOSTICS codigo=RETURNED_SQLSTATE;
  clase:=CASE WHEN codigo IN('42501','22023','23505','P0002','VCA31') THEN 'denegado' ELSE 'error' END;
 END;
 -- La subtransacción anterior deshizo el núcleo y el acuse favorable. El
 -- evento negativo se confirma por AD192 antes de responder, sin V2 ficticio.
 evento_fallo:=p_evento;
 IF evento_previo IS NOT NULL AND evento_previo=p_evento THEN
  evento_fallo:='evento_'||replace(gen_random_uuid()::text,'-','');
 END IF;
 e:=vec_contexto_actor_v1.evento_contexto_admin_pre_v2_v1(evento_fallo,p_correlacion,p_proceso,
  'registrar_contexto_admin',p_operacion,clase,
  CASE WHEN fuente_acreditada THEN j->>'persona_ref' ELSE NULL END,
  CASE WHEN fuente_acreditada THEN p_perfil ELSE NULL END,
  CASE WHEN fuente_acreditada THEN j->>'fuente_ref' ELSE NULL END,
  CASE WHEN fuente_acreditada THEN j->>'fuente_sha256' ELSE NULL END);
 SELECT * INTO STRICT a FROM vec_autorizacion_atestada_v3.registrar_contexto_admin_pre_v2_ca_v1(e);
 IF a.auditoria_ref IS DISTINCT FROM 'aud_v3_ap2_'||substr(evento_fallo,8,32)
  OR a.secuencia IS NULL OR a.secuencia<1 OR a.huella_sha256 IS NULL OR a.huella_sha256 !~ '^[0-9a-f]{64}$'
  OR a.correlacion_ref IS DISTINCT FROM p_correlacion OR a.registrada_en IS NULL
 THEN RAISE EXCEPTION 'CA36: acuse negativo incompatible' USING ERRCODE='42501'; END IF;
 RETURN jsonb_build_object('estado',clase,'motivo_ref',e->>'motivo_ref','evento',e,'acuse',to_jsonb(a),'contexto',NULL);
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.registrar_contexto_admin_v1(text,text,text,text,text,text,timestamptz,text,numeric,text,text,text,text,text,text) FROM PUBLIC;
CREATE FUNCTION vec_contexto_actor_v1.recuperar_contexto_admin_v1(
 p_operacion text,p_recibo text,p_cuenta text,p_perfil text,p_metodo text,p_garantia text,
 p_solicitado timestamptz,p_vis text,p_vis_version numeric,p_vis_sha text,p_aut text,p_ses text,
 p_evento text,p_correlacion text,p_proceso text,p_evento_material jsonb)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='4s' AS $f$
DECLARE l record;r record;a record;j jsonb;j_final jsonb;e jsonb;enlace_evento text;enlace_encontrado boolean;
BEGIN
 PERFORM vec_contexto_actor_v1.exigir_runtime_contexto_admin_v1();
 IF current_setting('transaction_isolation')<>'read committed' OR current_setting('transaction_read_only')<>'off'
  OR p_metodo NOT IN('certificado','dnie') OR p_garantia IS DISTINCT FROM 'alto'
  OR vec_contexto_actor_v1.referencia_operacion_valida(p_operacion,'oca_') IS NOT TRUE
  OR vec_contexto_actor_v1.referencia_operacion_valida(p_recibo,'rca_') IS NOT TRUE
  OR p_evento !~ '^evento_[0-9a-f]{32}$' OR p_correlacion !~ '^correlacion_[0-9a-f]{32}$'
  OR jsonb_typeof(p_evento_material) IS DISTINCT FROM 'object'
  OR p_evento_material->>'evento_ref' IS DISTINCT FROM p_evento
  OR p_evento_material->>'accion' IS DISTINCT FROM 'registrar_contexto_admin'
  OR p_evento_material->>'recurso_ref' IS DISTINCT FROM p_operacion
  OR p_evento_material->>'correlacion_ref' IS DISTINCT FROM p_correlacion
  OR p_evento_material->>'proceso' IS DISTINCT FROM p_proceso
 THEN RAISE EXCEPTION 'CA36: recuperación incompatible' USING ERRCODE='42501'; END IF;
 -- No se crea auditoría ni se reaplica un efecto. La ausencia del acuse
 -- original tampoco acredita una denegación histórica.
 SELECT * INTO STRICT a FROM vec_autorizacion_atestada_v3.cotejar_contexto_admin_pre_v2_ca_v1(p_evento_material);
 IF a.auditoria_ref IS DISTINCT FROM 'aud_v3_ap2_'||substr(p_evento,8,32)
  OR a.secuencia IS NULL OR a.secuencia<1 OR a.huella_sha256 IS NULL OR a.huella_sha256 !~ '^[0-9a-f]{64}$'
  OR a.correlacion_ref IS DISTINCT FROM p_correlacion OR a.registrada_en IS NULL
 THEN RAISE EXCEPTION 'CA36: acuse histórico incompatible' USING ERRCODE='42501'; END IF;
 SELECT * INTO l FROM vec_contexto_actor_v1.enlace_contexto_admin_v1
  WHERE operacion_ref=p_operacion FOR SHARE;
 enlace_encontrado:=FOUND;
 IF enlace_encontrado THEN enlace_evento:=l.evento_ref; END IF;
 IF p_evento_material->>'resultado' IN('denegado','error') THEN
  -- La denegación recupera su hecho AD192. Una fila histórica de otro
  -- intento no se convierte en efecto de este evento ni se declara ausente.
  IF enlace_evento IS NOT DISTINCT FROM p_evento
   OR p_evento_material->'actor_ref' IS DISTINCT FROM 'null'::jsonb AND
      (p_evento_material->>'perfil_activo_ref' IS NULL OR p_evento_material->>'fuente_ref' IS NULL)
  THEN RAISE EXCEPTION 'CA36: ausencia no conciliable' USING ERRCODE='42501'; END IF;
  RETURN jsonb_build_object('estado',p_evento_material->>'resultado',
   'motivo_ref',p_evento_material->>'motivo_ref','evento',p_evento_material,
   'acuse',to_jsonb(a),'contexto',NULL);
 END IF;
 IF p_evento_material->>'resultado' IS DISTINCT FROM 'permitido' OR enlace_encontrado IS NOT TRUE
 THEN RAISE EXCEPTION 'CA36: efecto favorable ausente' USING ERRCODE='42501'; END IF;
 IF l.registro_contexto_ref IS DISTINCT FROM p_recibo OR l.vinculo_sesion_ref IS DISTINCT FROM p_vis
  OR l.vinculo_sesion_version IS DISTINCT FROM p_vis_version OR l.vinculo_sesion_sha256 IS DISTINCT FROM p_vis_sha
  OR l.autenticacion_ref IS DISTINCT FROM p_aut OR l.sesion_ref IS DISTINCT FROM p_ses
  OR l.cuenta_ref IS DISTINCT FROM p_cuenta OR l.perfil_ref IS DISTINCT FROM p_perfil
  OR l.evento_ref IS DISTINCT FROM p_evento OR l.correlacion_ref IS DISTINCT FROM p_correlacion
  OR l.proceso IS DISTINCT FROM p_proceso
 THEN RAISE EXCEPTION 'CA36: enlace de recuperación divergente' USING ERRCODE='42501'; END IF;
 e:=vec_contexto_actor_v1.evento_contexto_admin_pre_v2_v1(p_evento,p_correlacion,p_proceso,
  'registrar_contexto_admin',p_operacion,'permitido',l.actor_ref,l.perfil_ref,l.fuente_ref,l.fuente_sha256);
 IF p_evento_material IS DISTINCT FROM e THEN RAISE EXCEPTION 'CA36: evento histórico sustituido' USING ERRCODE='42501'; END IF;
 j:=vec_identidad_sesiones_v1.cotejar_vinculo_sesion_admin_v1(
  p_vis,p_vis_version,p_vis_sha,p_aut,p_ses,p_cuenta,p_perfil,p_solicitado,p_metodo);
 IF jsonb_typeof(j) IS DISTINCT FROM 'object' OR j->>'persona_ref' IS DISTINCT FROM l.actor_ref
  OR j->>'fuente_ref' IS DISTINCT FROM l.fuente_ref OR j->>'fuente_sha256' IS DISTINCT FROM l.fuente_sha256
 THEN RAISE EXCEPTION 'CA36: vínculo recuperado no actual' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT r FROM vec_contexto_actor_v1.reconciliar_contexto_ca36_interno_v1(
  p_operacion,p_recibo,p_cuenta,p_perfil,p_metodo,p_garantia,p_solicitado,'{}'::text[]);
 IF vec_contexto_actor_v1.cotejar_contexto_admin_actual_v1(p_cuenta,l.actor_ref,p_perfil,
  (j->>'seleccion_revision')::numeric,(j->>'vigente_hasta')::timestamptz,r.representacion_canonica) IS NOT TRUE
 THEN RAISE EXCEPTION 'CA36: contexto recuperado no actual' USING ERRCODE='42501'; END IF;
 j_final:=vec_identidad_sesiones_v1.cotejar_vinculo_sesion_admin_v1(
  p_vis,p_vis_version,p_vis_sha,p_aut,p_ses,p_cuenta,p_perfil,p_solicitado,p_metodo);
 IF j_final IS DISTINCT FROM j OR vec_contexto_actor_v1.cotejar_contexto_admin_actual_v1(p_cuenta,l.actor_ref,p_perfil,
  (j->>'seleccion_revision')::numeric,(j->>'vigente_hasta')::timestamptz,r.representacion_canonica) IS NOT TRUE
  OR clock_timestamp()>=(j->>'vigente_hasta')::timestamptz
 THEN RAISE EXCEPTION 'CA36: recuperación vencida' USING ERRCODE='42501'; END IF;
 RETURN jsonb_build_object('estado','permitido','motivo_ref',e->>'motivo_ref','evento',e,'acuse',to_jsonb(a),
  'contexto',jsonb_build_object('operacion_ref',r.operacion_ref,'registro_contexto_ref',r.registro_contexto_ref,
   'representacion_canonica_base64',encode(r.representacion_canonica,'base64'),'huella_sha256',r.huella_sha256,
   'manifiesto_procedencia_canonico_base64',encode(r.manifiesto_procedencia_canonico,'base64'),
   'manifiesto_procedencia_huella_sha256',r.manifiesto_procedencia_huella_sha256,
   'autoridad_efectiva',r.autoridad_efectiva,'resuelto_en',r.resuelto_en));
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.recuperar_contexto_admin_v1(text,text,text,text,text,text,timestamptz,text,numeric,text,text,text,text,text,text,jsonb) FROM PUBLIC;
CREATE FUNCTION vec_contexto_actor_v1.reconciliar_contexto_admin_v1(
 p_operacion text,p_recibo text,p_cuenta text,p_perfil text,p_metodo text,p_garantia text,
 p_solicitado timestamptz,p_vis text,p_vis_version numeric,p_vis_sha text,p_aut text,p_ses text,
 p_evento text,p_correlacion text,p_proceso text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='4s' AS $f$
DECLARE l record;r record;a record;j jsonb;j_final jsonb;e jsonb;clase text;codigo text;
 fuente_acreditada boolean:=false;evento_previo text;evento_fallo text;
BEGIN
 PERFORM vec_contexto_actor_v1.exigir_runtime_contexto_admin_v1();
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
  OR p_metodo NOT IN('certificado','dnie') OR p_garantia IS DISTINCT FROM 'alto'
  OR vec_contexto_actor_v1.referencia_operacion_valida(p_operacion,'oca_') IS NOT TRUE
  OR vec_contexto_actor_v1.referencia_operacion_valida(p_recibo,'rca_') IS NOT TRUE
 OR p_evento !~ '^evento_[0-9a-f]{32}$' OR p_correlacion !~ '^correlacion_[0-9a-f]{32}$'
 THEN RAISE EXCEPTION 'CA36: consulta ADMIN inválida' USING ERRCODE='22023'; END IF;
 SELECT x.evento_ref INTO evento_previo FROM vec_contexto_actor_v1.enlace_contexto_admin_v1 x
 WHERE x.operacion_ref=p_operacion;
 BEGIN
  j:=vec_identidad_sesiones_v1.cotejar_vinculo_sesion_admin_v1(
   p_vis,p_vis_version,p_vis_sha,p_aut,p_ses,p_cuenta,p_perfil,p_solicitado,p_metodo);
  IF jsonb_typeof(j) IS DISTINCT FROM 'object' OR j->>'cuenta_ref' IS DISTINCT FROM p_cuenta
   OR j->>'perfil_ref' IS DISTINCT FROM p_perfil OR j->>'referencia' IS DISTINCT FROM p_vis
   OR (j->>'version')::numeric IS DISTINCT FROM p_vis_version OR j->>'huella_sha256' IS DISTINCT FROM p_vis_sha
   OR j->>'autenticacion_ref' IS DISTINCT FROM p_aut OR j->>'sesion_ref' IS DISTINCT FROM p_ses
  THEN RAISE EXCEPTION 'CA36: vínculo consulta incompatible' USING ERRCODE='42501'; END IF;
  fuente_acreditada:=true;
  SELECT * INTO STRICT l FROM vec_contexto_actor_v1.enlace_contexto_admin_v1 WHERE operacion_ref=p_operacion FOR SHARE;
  IF l.registro_contexto_ref IS DISTINCT FROM p_recibo OR l.vinculo_sesion_ref IS DISTINCT FROM p_vis
   OR l.vinculo_sesion_version IS DISTINCT FROM p_vis_version OR l.vinculo_sesion_sha256 IS DISTINCT FROM p_vis_sha
   OR l.autenticacion_ref IS DISTINCT FROM p_aut OR l.sesion_ref IS DISTINCT FROM p_ses
   OR l.cuenta_ref IS DISTINCT FROM p_cuenta OR l.perfil_ref IS DISTINCT FROM p_perfil
   OR l.actor_ref IS DISTINCT FROM j->>'persona_ref' OR l.fuente_ref IS DISTINCT FROM j->>'fuente_ref'
   OR l.fuente_sha256 IS DISTINCT FROM j->>'fuente_sha256' OR l.evento_ref IS NOT DISTINCT FROM p_evento
  THEN RAISE EXCEPTION 'CA36: consulta de otro contexto' USING ERRCODE='42501'; END IF;
  SELECT * INTO STRICT r FROM vec_contexto_actor_v1.registros_contexto
   WHERE operacion_ref=p_operacion AND registro_contexto_ref=p_recibo FOR SHARE;
  IF r.solicitado_en IS DISTINCT FROM p_solicitado OR r.metodo IS DISTINCT FROM p_metodo
   OR r.garantia IS DISTINCT FROM p_garantia
   OR vec_contexto_actor_v1.cotejar_contexto_admin_actual_v1(p_cuenta,l.actor_ref,p_perfil,
   (j->>'seleccion_revision')::numeric,(j->>'vigente_hasta')::timestamptz,r.representacion_canonica) IS NOT TRUE
  THEN RAISE EXCEPTION 'CA36: consulta sin contexto actual' USING ERRCODE='42501'; END IF;
  e:=vec_contexto_actor_v1.evento_contexto_admin_pre_v2_v1(p_evento,p_correlacion,p_proceso,
   'reconciliar_contexto_admin',p_operacion,'permitido',l.actor_ref,l.perfil_ref,l.fuente_ref,l.fuente_sha256);
  SELECT * INTO STRICT a FROM vec_autorizacion_atestada_v3.registrar_contexto_admin_pre_v2_ca_v1(e);
  IF a.auditoria_ref IS DISTINCT FROM 'aud_v3_ap2_'||substr(p_evento,8,32)
   OR a.secuencia IS NULL OR a.secuencia<1 OR a.huella_sha256 IS NULL OR a.huella_sha256 !~ '^[0-9a-f]{64}$'
   OR a.correlacion_ref IS DISTINCT FROM p_correlacion OR a.registrada_en IS NULL
  THEN RAISE EXCEPTION 'CA36: acuse de consulta incompatible' USING ERRCODE='42501'; END IF;
  j_final:=vec_identidad_sesiones_v1.cotejar_vinculo_sesion_admin_v1(
   p_vis,p_vis_version,p_vis_sha,p_aut,p_ses,p_cuenta,p_perfil,p_solicitado,p_metodo);
  IF j_final IS DISTINCT FROM j OR vec_contexto_actor_v1.cotejar_contexto_admin_actual_v1(p_cuenta,l.actor_ref,p_perfil,
   (j->>'seleccion_revision')::numeric,(j->>'vigente_hasta')::timestamptz,r.representacion_canonica) IS NOT TRUE
   OR clock_timestamp()>=(j->>'vigente_hasta')::timestamptz
  THEN RAISE EXCEPTION 'CA36: consulta vencida' USING ERRCODE='42501'; END IF;
  RETURN jsonb_build_object('estado','permitido','motivo_ref',e->>'motivo_ref','evento',e,'acuse',to_jsonb(a),
   'contexto',jsonb_build_object('operacion_ref',r.operacion_ref,'registro_contexto_ref',r.registro_contexto_ref,
    'representacion_canonica_base64',encode(r.representacion_canonica,'base64'),'huella_sha256',r.huella_sha256,
    'manifiesto_procedencia_canonico_base64',encode(r.manifiesto_procedencia_canonico,'base64'),
    'manifiesto_procedencia_huella_sha256',r.manifiesto_procedencia_huella_sha256,
    'autoridad_efectiva',r.autoridad_efectiva,'resuelto_en',r.resuelto_en));
 EXCEPTION WHEN serialization_failure OR deadlock_detected OR query_canceled THEN RAISE;
 WHEN OTHERS THEN
  GET STACKED DIAGNOSTICS codigo=RETURNED_SQLSTATE;
  clase:=CASE WHEN codigo IN('42501','22023','23505','P0002','VCA31') THEN 'denegado' ELSE 'error' END;
 END;
 evento_fallo:=p_evento;
 IF evento_previo IS NOT NULL AND evento_previo=p_evento THEN
  evento_fallo:='evento_'||replace(gen_random_uuid()::text,'-','');
 END IF;
 e:=vec_contexto_actor_v1.evento_contexto_admin_pre_v2_v1(evento_fallo,p_correlacion,p_proceso,
  'reconciliar_contexto_admin',p_operacion,clase,
  CASE WHEN fuente_acreditada THEN j->>'persona_ref' ELSE NULL END,
  CASE WHEN fuente_acreditada THEN p_perfil ELSE NULL END,
  CASE WHEN fuente_acreditada THEN j->>'fuente_ref' ELSE NULL END,
  CASE WHEN fuente_acreditada THEN j->>'fuente_sha256' ELSE NULL END);
 SELECT * INTO STRICT a FROM vec_autorizacion_atestada_v3.registrar_contexto_admin_pre_v2_ca_v1(e);
 IF a.auditoria_ref IS DISTINCT FROM 'aud_v3_ap2_'||substr(evento_fallo,8,32)
  OR a.secuencia IS NULL OR a.secuencia<1 OR a.huella_sha256 IS NULL OR a.huella_sha256 !~ '^[0-9a-f]{64}$'
  OR a.correlacion_ref IS DISTINCT FROM p_correlacion OR a.registrada_en IS NULL
 THEN RAISE EXCEPTION 'CA36: acuse negativo de consulta incompatible' USING ERRCODE='42501'; END IF;
 RETURN jsonb_build_object('estado',clase,'motivo_ref',e->>'motivo_ref','evento',e,'acuse',to_jsonb(a),'contexto',NULL);
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.reconciliar_contexto_admin_v1(text,text,text,text,text,text,timestamptz,text,numeric,text,text,text,text,text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.acreditar_runtime_contexto_admin_v1(),
 vec_contexto_actor_v1.registrar_contexto_admin_v1(text,text,text,text,text,text,timestamptz,text,numeric,text,text,text,text,text,text),
 vec_contexto_actor_v1.recuperar_contexto_admin_v1(text,text,text,text,text,text,timestamptz,text,numeric,text,text,text,text,text,text,jsonb),
 vec_contexto_actor_v1.reconciliar_contexto_admin_v1(text,text,text,text,text,text,timestamptz,text,numeric,text,text,text,text,text,text)
 TO vec_contexto_actor_v1_admin_contexto;
-- Instalación única: no reaplicar ni ejecutar DOWN una vez haya enlaces.
-- Conflictos de serialización, interbloqueos y cancelaciones se relanzan:
-- la transacción entera falla sin auditar un «error» y se puede repetir.
COMMIT;
