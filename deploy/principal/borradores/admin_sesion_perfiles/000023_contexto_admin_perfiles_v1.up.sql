-- BORRADOR: CA22 real y fachada nominal AUT24 son dependencias previas.
\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_contexto_actor_v1:migracion:000023',0));
DO $preimagen$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
 OR pg_catalog.to_regprocedure('vec_contexto_actor_v1.bloquear_contexto_admin_v1(text,text,text,text,numeric,numeric,numeric,numeric)') IS NULL
 OR pg_catalog.to_regprocedure('vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2_propietaria_admin_v1(text,text,text,text,text,text,timestamptz,text[])') IS NULL
 OR pg_catalog.to_regprocedure('vec_contexto_actor_v1.reconciliar_contexto_actor_v2_propietaria_admin_v1(text,text,text,text,text,text,timestamptz,text[])') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion.consultar_asignacion_admin_perfiles_v1(text)') IS NULL
 OR pg_catalog.to_regrole('vec_identidad_sesiones_v1_admin_perfiles') IS NOT NULL
 THEN RAISE EXCEPTION 'CA23: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $preimagen$;
CREATE ROLE vec_identidad_sesiones_v1_admin_perfiles NOLOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
DO $conexion$ BEGIN
 EXECUTE pg_catalog.format('GRANT CONNECT ON DATABASE %I TO vec_identidad_sesiones_v1_admin_perfiles',current_database());
END $conexion$;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;

-- Solo consume la asignación central nominal de AUT24. Nunca crea perfiles,
-- asignaciones ni vínculos, ni toma un perfil solicitado como autoridad.
CREATE FUNCTION vec_contexto_actor_v1.consultar_perfil_admin_perfiles_v1(p_cuenta text)
RETURNS TABLE(persona_ref text,perfil_ref text,vinculo_ref text,cuenta_version numeric,persona_version numeric,perfil_version numeric,vinculo_version numeric,audiencia text,vigente_hasta timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE b record;ahora timestamptz;
BEGIN
 IF current_setting('transaction_read_only')<>'off' OR current_setting('transaction_isolation')<>'serializable' THEN RETURN; END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec:admin:continuidad:v1',0));
 SELECT * INTO STRICT b FROM vec_autorizacion.consultar_asignacion_admin_perfiles_v1(p_cuenta);
 IF b.persona_ref IS NULL OR b.perfil_ref IS NULL OR b.vinculo_ref IS NULL
 OR b.cuenta_version IS NULL OR b.persona_version IS NULL OR b.perfil_version IS NULL OR b.vinculo_version IS NULL
 OR b.audiencia IS NULL OR b.audiencia !~ '^[a-z0-9][a-z0-9._:-]{3,255}$' OR b.vigente_hasta IS NULL OR NOT pg_catalog.isfinite(b.vigente_hasta) THEN RETURN; END IF;
 PERFORM vec_contexto_actor_v1.bloquear_contexto_admin_v1(p_cuenta,b.persona_ref,b.perfil_ref,b.vinculo_ref,b.cuenta_version,b.persona_version,b.perfil_version,b.vinculo_version);
 ahora:=pg_catalog.clock_timestamp();
 IF ahora>=b.vigente_hasta THEN RETURN; END IF;
 RETURN QUERY SELECT b.persona_ref,b.perfil_ref,b.vinculo_ref,b.cuenta_version,b.persona_version,b.perfil_version,b.vinculo_version,b.audiencia,b.vigente_hasta;
EXCEPTION WHEN no_data_found OR too_many_rows THEN RETURN;
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.consultar_perfil_admin_perfiles_v1(text) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.consultar_perfil_admin_perfiles_v1(text) FROM PUBLIC;

CREATE FUNCTION vec_contexto_actor_v1.exigir_runtime_admin_perfiles_v1()
RETURNS text LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE l record;g record;fs oid[];ns oid[];base oid;
BEGIN
 SELECT * INTO l FROM pg_catalog.pg_roles WHERE rolname=session_user;
 SELECT * INTO g FROM pg_catalog.pg_roles WHERE rolname='vec_identidad_sesiones_v1_admin_perfiles';
 SELECT oid INTO base FROM pg_catalog.pg_database WHERE datname=current_database();
 fs:=ARRAY[pg_catalog.to_regprocedure('vec_contexto_actor_v1.acreditar_runtime_admin_perfiles_v1()'),pg_catalog.to_regprocedure('vec_contexto_actor_v1.resolver_y_registrar_contexto_admin_perfiles_v1(text,text,text,timestamptz)'),pg_catalog.to_regprocedure('vec_contexto_actor_v1.reconciliar_contexto_admin_perfiles_v1(text,text,text,timestamptz)'),pg_catalog.to_regprocedure('vec_identidad_sesiones_v1.resolver_cuenta_admin_perfiles_v1(text,text,text,text,text,timestamptz,timestamptz)'),pg_catalog.to_regprocedure('vec_identidad_sesiones_v1.vincular_sesion_admin_perfiles_v1(text,text,text,text,text,timestamptz,timestamptz,timestamptz,timestamptz,text,text)')];
 ns:=ARRAY[pg_catalog.to_regnamespace('vec_contexto_actor_v1'),pg_catalog.to_regnamespace('vec_identidad_sesiones_v1')];
 IF l.oid IS NULL OR g.oid IS NULL OR array_position(fs,NULL) IS NOT NULL OR array_position(ns,NULL) IS NOT NULL
 OR NOT l.rolcanlogin OR NOT l.rolinherit OR l.rolsuper OR l.rolcreaterole OR l.rolcreatedb OR l.rolreplication OR l.rolbypassrls OR l.rolconfig IS NOT NULL
 OR g.rolcanlogin OR g.rolinherit OR g.rolsuper OR g.rolcreaterole OR g.rolcreatedb OR g.rolreplication OR g.rolbypassrls OR g.rolconfig IS NOT NULL
 OR current_setting('role')<>'none'
 OR (SELECT count(*) FROM pg_catalog.pg_auth_members WHERE member=l.oid)<>1
 OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members WHERE member=l.oid AND roleid=g.oid AND NOT admin_option AND inherit_option AND NOT set_option)
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_roles r WHERE r.oid<>g.oid AND r.oid<>l.oid AND pg_catalog.pg_has_role(l.oid,r.oid,'MEMBER'))
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members WHERE member=g.oid)
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_db_role_setting WHERE setrole IN(l.oid,g.oid))
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_shdepend WHERE refclassid='pg_catalog.pg_authid'::regclass AND refobjid=l.oid)
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_default_acl d LEFT JOIN LATERAL pg_catalog.aclexplode(COALESCE(d.defaclacl,'{}'::aclitem[])) a ON true WHERE d.defaclrole IN(l.oid,g.oid) OR a.grantee IN(l.oid,g.oid) OR a.grantor IN(l.oid,g.oid))
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_policy p WHERE l.oid=ANY(p.polroles) OR g.oid=ANY(p.polroles))
 OR NOT COALESCE((SELECT count(*)=1 AND bool_and(a.privilege_type='CONNECT' AND NOT a.is_grantable) FROM pg_catalog.pg_database d CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(d.datacl,pg_catalog.acldefault('d',d.datdba))) a WHERE d.oid=base AND a.grantee=g.oid),false)
 OR NOT COALESCE((SELECT count(*)=2 AND bool_and(a.privilege_type='USAGE' AND NOT a.is_grantable) FROM pg_catalog.pg_namespace n CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(n.nspacl,pg_catalog.acldefault('n',n.nspowner))) a WHERE n.oid=ANY(ns) AND a.grantee=g.oid),false)
 OR NOT COALESCE((SELECT count(*)=5 AND bool_and(a.privilege_type='EXECUTE' AND NOT a.is_grantable) FROM pg_catalog.pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(p.proacl,pg_catalog.acldefault('f',p.proowner))) a WHERE p.oid=ANY(fs) AND a.grantee=g.oid),false)
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_namespace n CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(n.nspacl,pg_catalog.acldefault('n',n.nspowner))) a WHERE n.oid=ANY(ns) AND a.grantee=0)
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(p.proacl,pg_catalog.acldefault('f',p.proowner))) a WHERE p.pronamespace=ANY(ns) AND a.grantee=0 AND p.prosecdef)
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_class c WHERE c.relnamespace=ANY(ns) AND c.relkind IN('r','p','v','m','S') AND (pg_catalog.has_table_privilege(l.oid,c.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER') OR pg_catalog.has_any_column_privilege(l.oid,c.oid,'SELECT,INSERT,UPDATE,REFERENCES')))

 OR NOT COALESCE((SELECT count(*)=8 AND bool_and(deptype='a' AND objsubid=0 AND ((classid='pg_catalog.pg_database'::regclass AND objid=base) OR (classid='pg_catalog.pg_namespace'::regclass AND objid=ANY(ns)) OR (classid='pg_catalog.pg_proc'::regclass AND objid=ANY(fs)))) FROM pg_catalog.pg_shdepend WHERE refclassid='pg_catalog.pg_authid'::regclass AND refobjid=g.oid),false)
 THEN RAISE EXCEPTION 'CA23: LOGIN ADMIN no acreditado' USING ERRCODE='42501'; END IF;
 RETURN session_user;
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.exigir_runtime_admin_perfiles_v1() FROM PUBLIC;

CREATE FUNCTION vec_contexto_actor_v1.acreditar_runtime_admin_perfiles_v1()
RETURNS TABLE(identidad_login text,acreditada boolean) LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
BEGIN
 RETURN QUERY SELECT vec_contexto_actor_v1.exigir_runtime_admin_perfiles_v1(),true;
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.acreditar_runtime_admin_perfiles_v1() FROM PUBLIC;

CREATE FUNCTION vec_contexto_actor_v1.resolver_y_registrar_contexto_admin_perfiles_v1(p_operacion text,p_registro text,p_cuenta text,p_solicitado timestamptz)
RETURNS TABLE(operacion_ref text,registro_contexto_ref text,representacion_canonica bytea,huella_sha256 text,manifiesto_procedencia_canonico bytea,manifiesto_procedencia_huella_sha256 text,autoridad_efectiva text,resuelto_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE b record;
BEGIN
 PERFORM vec_contexto_actor_v1.exigir_runtime_admin_perfiles_v1();
 SELECT * INTO STRICT b FROM vec_contexto_actor_v1.consultar_perfil_admin_perfiles_v1(p_cuenta);
 RETURN QUERY SELECT * FROM vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2_propietaria_admin_v1(p_operacion,p_registro,p_cuenta,b.perfil_ref,'certificado','alto',p_solicitado,'{}'::text[]);
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.resolver_y_registrar_contexto_admin_perfiles_v1(text,text,text,timestamptz) FROM PUBLIC;
CREATE FUNCTION vec_contexto_actor_v1.reconciliar_contexto_admin_perfiles_v1(p_operacion text,p_registro text,p_cuenta text,p_solicitado timestamptz)
RETURNS TABLE(operacion_ref text,registro_contexto_ref text,representacion_canonica bytea,huella_sha256 text,manifiesto_procedencia_canonico bytea,manifiesto_procedencia_huella_sha256 text,autoridad_efectiva text,resuelto_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE b record;
BEGIN
 PERFORM vec_contexto_actor_v1.exigir_runtime_admin_perfiles_v1();
 SELECT * INTO STRICT b FROM vec_contexto_actor_v1.consultar_perfil_admin_perfiles_v1(p_cuenta);
 RETURN QUERY SELECT * FROM vec_contexto_actor_v1.reconciliar_contexto_actor_v2_propietaria_admin_v1(p_operacion,p_registro,p_cuenta,b.perfil_ref,'certificado','alto',p_solicitado,'{}'::text[]);
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.reconciliar_contexto_admin_perfiles_v1(text,text,text,timestamptz) FROM PUBLIC;


REVOKE ALL ON FUNCTION vec_contexto_actor_v1.exigir_runtime_admin_perfiles_v1(),vec_contexto_actor_v1.acreditar_runtime_admin_perfiles_v1(),vec_contexto_actor_v1.resolver_y_registrar_contexto_admin_perfiles_v1(text,text,text,timestamptz),vec_contexto_actor_v1.reconciliar_contexto_admin_perfiles_v1(text,text,text,timestamptz) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_contexto_actor_v1 TO vec_identidad_sesiones_v1_admin_perfiles,vec_identidad_sesiones_v1_propietario;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.exigir_runtime_admin_perfiles_v1(),vec_contexto_actor_v1.consultar_perfil_admin_perfiles_v1(text) TO vec_identidad_sesiones_v1_propietario;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.acreditar_runtime_admin_perfiles_v1(),vec_contexto_actor_v1.resolver_y_registrar_contexto_admin_perfiles_v1(text,text,text,timestamptz),vec_contexto_actor_v1.reconciliar_contexto_admin_perfiles_v1(text,text,text,timestamptz) TO vec_identidad_sesiones_v1_admin_perfiles;
COMMIT;
