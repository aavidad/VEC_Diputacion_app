\set ON_ERROR_STOP on
-- CA36: contexto ADMIN con runtime segregado. Borrador cerrado hasta IS16/AD192.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec:admin:continuidad:v1',0));
DO $pre$
DECLARE r record;n integer:=0;
BEGIN
 IF true THEN RAISE EXCEPTION 'CA36: borrador dependiente de IS16/AD192 y dos revisiones' USING ERRCODE='55000'; END IF;
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 OR to_regrole('vec_contexto_actor_v1_admin_contexto') IS NOT NULL
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
  to_regprocedure('vec_contexto_actor_v1.recuperar_contexto_admin_v1(text,text,text,text,text,text,timestamptz,text,numeric,text,text,text,text,text,text)'),
  to_regprocedure('vec_contexto_actor_v1.reconciliar_contexto_admin_v1(text,text,text,text,text,text,timestamptz,text,numeric,text,text,text,text,text,text)')];
 IF current_setting('role')<>'none' OR l.oid IS NULL OR g.oid IS NULL OR dbo IS NULL OR nso IS NULL
  OR array_position(fs,NULL) IS NOT NULL OR cardinality(fs)<>4
  OR l.rolcanlogin IS NOT TRUE OR l.rolinherit IS NOT TRUE
  OR l.rolsuper OR l.rolcreaterole OR l.rolcreatedb OR l.rolreplication OR l.rolbypassrls OR l.rolconfig IS NOT NULL
  OR g.rolcanlogin OR g.rolinherit OR g.rolsuper OR g.rolcreaterole OR g.rolcreatedb OR g.rolreplication OR g.rolbypassrls OR g.rolconfig IS NOT NULL
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
-- Las fachadas ADMIN, enlace durable y ACL exclusivas se añaden tras fijar
-- IS16 y la consulta AD192; el bloqueo $pre$ impide instalar este borrador.
COMMIT;
