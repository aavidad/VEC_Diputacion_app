\set ON_ERROR_STOP on
-- IS14: identidad sin perfil para el selector ADMIN y auditoría común AD171.
-- Migración nueva; no reaplicar ni ejecutar DOWN sobre historia.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_identidad_sesiones_v1:migracion:000014',0));
DO $pre$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
 OR pg_catalog.to_regprocedure('vec_identidad_sesiones_v1.revalidar_certificado_admin_v1(text,text,text,text,text,text,boolean,timestamptz,boolean,boolean)') IS NULL
 OR pg_catalog.to_regprocedure('vec_identidad_sesiones_v1.resolver_identidad_admin_perfiles_propietaria_v1(text,text,text,text,text,timestamptz,timestamptz)') IS NOT NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_evento_admin_preperfil_v1(jsonb)') IS NULL
 OR pg_catalog.to_regprocedure('vec_contexto_actor_v1.listar_admin_preperfil_propietaria_v1(text,text,text)') IS NULL
 OR pg_catalog.to_regrole('vec_identidad_sesiones_v1_admin_preperfil') IS NOT NULL
 THEN RAISE EXCEPTION 'IS14: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_identidad_sesiones_v1_propietario;
CREATE FUNCTION vec_identidad_sesiones_v1.resolver_identidad_admin_perfiles_propietaria_v1(p_entorno text,p_host text,p_audiencia text,p_certificado text,p_ca text,p_autenticada timestamptz,p_revocada timestamptz)
RETURNS TABLE(persona_ref text,cuenta_ref text,cuenta_ordinaria_ref text,vinculo_ref text,vinculo_version numeric,politica_ref text,politica_huella_sha256 text,vigente_hasta timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET timezone='UTC' AS $f$
DECLARE pol record;v record;c record;ahora timestamptz;
BEGIN
 IF current_setting('transaction_read_only')<>'off' OR current_setting('transaction_isolation') NOT IN('serializable','read committed')
 OR p_audiencia IS NULL OR p_audiencia !~ '^[a-z0-9][a-z0-9._:-]{3,255}$'
 OR p_autenticada IS NULL OR NOT pg_catalog.isfinite(p_autenticada)
 OR p_certificado IS NULL OR p_certificado !~ '^[0-9a-f]{64}$' OR p_certificado=pg_catalog.repeat('0',64)
 OR p_ca IS NULL OR p_ca !~ '^[0-9a-f]{64}$' OR p_ca=pg_catalog.repeat('0',64)
 THEN RETURN; END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec:admin:continuidad:v1',0));
 SELECT * INTO pol FROM vec_identidad_sesiones_v1.politica_certificado_admin_v1 WHERE singleton FOR SHARE;
 ahora:=pg_catalog.clock_timestamp();
 IF NOT FOUND OR p_autenticada>ahora OR p_autenticada<pol.registrada_en OR NOT pol.activa
 OR pol.entorno IS DISTINCT FROM p_entorno OR pol.host_admin IS DISTINCT FROM p_host OR pol.ca_sha256 IS DISTINCT FROM p_ca OR ahora>=pol.vigente_hasta THEN RETURN; END IF;
 SELECT x.* INTO STRICT v FROM vec_identidad_sesiones_v1.vinculo_certificado_admin_actual_v1 a JOIN vec_identidad_sesiones_v1.vinculo_certificado_admin_v1 x USING(vinculo_ref,version)
 WHERE x.certificado_sha256=p_certificado AND x.ca_sha256=p_ca AND x.politica_ref=pol.politica_ref FOR SHARE OF a;
 -- Este corte usa únicamente certificado directo: producción sigue cerrada
 -- hasta disponer de las evidencias corporativas exigidas por IS9.
 IF vec_identidad_sesiones_v1.revalidar_certificado_admin_v1(p_entorno,p_host,v.persona_ref,v.cuenta_privilegiada_ref,p_certificado,p_ca,true,p_revocada,false,false) IS NOT TRUE
 OR p_autenticada<v.vigente_desde THEN RETURN; END IF;
 SELECT * INTO STRICT c FROM vec_identidad_sesiones_v1.cuenta WHERE cuenta.cuenta_ref=v.cuenta_privilegiada_ref;
 IF c.cuenta_ordinaria_ref=c.cuenta_ref OR NOT c.cuenta_privilegiada
 OR NOT EXISTS(SELECT 1 FROM vec_identidad_sesiones_v1.cuenta o WHERE o.cuenta_ref=c.cuenta_ordinaria_ref AND NOT o.cuenta_privilegiada AND o.cuenta_ordinaria_ref IS NULL) THEN RETURN; END IF;
 ahora:=pg_catalog.clock_timestamp();
 IF ahora>=LEAST(pol.vigente_hasta,v.vigente_hasta,p_revocada+pol.maxima_edad_revocacion) THEN RETURN; END IF;
 RETURN QUERY SELECT v.persona_ref,c.cuenta_ref,c.cuenta_ordinaria_ref,v.vinculo_ref,v.version,pol.politica_ref,pol.huella_aprobacion_sha256,LEAST(pol.vigente_hasta,v.vigente_hasta,p_revocada+pol.maxima_edad_revocacion);
EXCEPTION WHEN no_data_found OR too_many_rows OR data_exception THEN RETURN;
END $f$;
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.resolver_identidad_admin_perfiles_propietaria_v1(text,text,text,text,text,timestamptz,timestamptz) FROM PUBLIC;

-- Acreditación del consumidor mTLS. Se aprovisiona desde el canal privado;
-- no crea LOGIN, personas, certificados ni perfiles.
CREATE TABLE vec_identidad_sesiones_v1.config_runtime_admin_preperfil_v1 (
 identidad_login text PRIMARY KEY,
 proceso text NOT NULL CHECK(proceso ~ '^[a-z][a-z0-9._-]{1,79}$'),
 entorno text NOT NULL CHECK(entorno IN('desarrollo','cidonia','produccion')),
 host_admin text NOT NULL,
 audiencia text NOT NULL CHECK(audiencia ~ '^[a-z0-9][a-z0-9._:-]{3,255}$'),
 configurada_en timestamptz NOT NULL DEFAULT pg_catalog.clock_timestamp(),
 vigente_hasta timestamptz NOT NULL CHECK(pg_catalog.isfinite(vigente_hasta)),
 CHECK(vigente_hasta>configurada_en)
);
CREATE TABLE vec_identidad_sesiones_v1.observacion_admin_preperfil_v1 (
 evento_ref text PRIMARY KEY CHECK(evento_ref ~ '^evento_[0-9a-f]{32}$'),
 fuente_ref text NOT NULL UNIQUE CHECK(fuente_ref ~ '^observacion_admin_preperfil:[0-9a-f]{32}$'),
 fuente_sha256 text NOT NULL CHECK(fuente_sha256 ~ '^[0-9a-f]{64}$'),
 documento jsonb NOT NULL,
 registrada_en timestamptz NOT NULL DEFAULT pg_catalog.clock_timestamp(),
 CHECK(fuente_sha256=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(documento::text,'UTF8')),'hex'))
);
DO $historia$
DECLARE nombre text;
BEGIN
 FOREACH nombre IN ARRAY ARRAY['config_runtime_admin_preperfil_v1','observacion_admin_preperfil_v1'] LOOP
  EXECUTE pg_catalog.format('ALTER TABLE vec_identidad_sesiones_v1.%I ENABLE ROW LEVEL SECURITY',nombre);
  EXECUTE pg_catalog.format('ALTER TABLE vec_identidad_sesiones_v1.%I FORCE ROW LEVEL SECURITY',nombre);
  EXECUTE pg_catalog.format('CREATE POLICY propietario_exacto ON vec_identidad_sesiones_v1.%I FOR ALL TO vec_identidad_sesiones_v1_propietario USING(current_user=%L) WITH CHECK(current_user=%L)',nombre,'vec_identidad_sesiones_v1_propietario','vec_identidad_sesiones_v1_propietario');
  EXECUTE pg_catalog.format('CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE OR TRUNCATE ON vec_identidad_sesiones_v1.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_identidad_sesiones_v1.rechazar_mutacion()',nombre);
  EXECUTE pg_catalog.format('REVOKE ALL ON TABLE vec_identidad_sesiones_v1.%I FROM PUBLIC',nombre);
  EXECUTE pg_catalog.format('REVOKE ALL ON TYPE vec_identidad_sesiones_v1.%I FROM PUBLIC',nombre);
 END LOOP;
END $historia$;

CREATE FUNCTION vec_identidad_sesiones_v1.exigir_runtime_admin_preperfil_v1(p_entorno text,p_host text,p_audiencia text)
RETURNS text LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $f$
DECLARE cfg record;l record;g record;fs oid[];ns oid;db oid;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
 OR current_setting('TimeZone')<>'UTC' THEN RAISE EXCEPTION 'IS14: requiere SERIALIZABLE READ WRITE UTC' USING ERRCODE='25000'; END IF;
 SELECT * INTO l FROM pg_roles WHERE rolname=session_user;
 SELECT * INTO g FROM pg_roles WHERE rolname='vec_identidad_sesiones_v1_admin_preperfil';
 fs:=ARRAY[to_regprocedure('vec_identidad_sesiones_v1.acreditar_runtime_preperfil_admin_v1()'),to_regprocedure('vec_identidad_sesiones_v1.listar_perfiles_admin_auditado_v1(text,text,text,text,text,timestamptz,timestamptz,timestamptz,timestamptz,text,text)'),to_regprocedure('vec_identidad_sesiones_v1.seleccionar_perfil_admin_auditado_v1(text,text,text,text,text,timestamptz,timestamptz,timestamptz,timestamptz,text,numeric,text,text)')];
 ns:=to_regnamespace('vec_identidad_sesiones_v1');
 SELECT oid INTO db FROM pg_database WHERE datname=current_database();
 IF l.oid IS NULL OR g.oid IS NULL OR NOT l.rolcanlogin OR NOT l.rolinherit
 OR l.rolsuper OR l.rolcreaterole OR l.rolcreatedb OR l.rolreplication OR l.rolbypassrls OR l.rolconfig IS NOT NULL
 OR g.rolcanlogin OR g.rolinherit OR g.rolsuper OR g.rolcreaterole OR g.rolcreatedb OR g.rolreplication OR g.rolbypassrls OR g.rolconfig IS NOT NULL
 OR current_setting('role')<>'none'
 OR (SELECT count(*) FROM pg_auth_members WHERE member=l.oid)<>1
 OR NOT EXISTS(SELECT 1 FROM pg_auth_members WHERE member=l.oid AND roleid=g.oid AND NOT admin_option AND inherit_option AND NOT set_option)
 OR EXISTS(SELECT 1 FROM pg_auth_members WHERE member=g.oid)
 OR EXISTS(SELECT 1 FROM pg_db_role_setting WHERE setrole IN(l.oid,g.oid))
 OR EXISTS(SELECT 1 FROM pg_shdepend WHERE refclassid='pg_catalog.pg_authid'::regclass AND refobjid=l.oid)
 OR array_position(fs,NULL) IS NOT NULL
 OR EXISTS(SELECT 1 FROM pg_default_acl d LEFT JOIN LATERAL aclexplode(coalesce(d.defaclacl,'{}'::aclitem[])) a ON true WHERE d.defaclrole IN(l.oid,g.oid) OR a.grantee IN(l.oid,g.oid) OR a.grantor IN(l.oid,g.oid))
 OR EXISTS(SELECT 1 FROM pg_policy p WHERE l.oid=ANY(p.polroles) OR g.oid=ANY(p.polroles))
 OR NOT coalesce((SELECT count(*)=5 AND bool_and(deptype='a' AND objsubid=0 AND ((classid='pg_catalog.pg_database'::regclass AND objid=db) OR (classid='pg_catalog.pg_namespace'::regclass AND objid=ns) OR (classid='pg_catalog.pg_proc'::regclass AND objid=ANY(fs)))) FROM pg_shdepend WHERE refclassid='pg_catalog.pg_authid'::regclass AND refobjid=g.oid),false)
 OR NOT coalesce((SELECT count(*)=3 AND bool_and(a.privilege_type='EXECUTE' AND NOT a.is_grantable) FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=ANY(fs) AND a.grantee=g.oid),false)
 OR NOT coalesce((SELECT count(*)=1 AND bool_and(a.privilege_type='USAGE' AND NOT a.is_grantable) FROM pg_namespace n CROSS JOIN LATERAL aclexplode(coalesce(n.nspacl,acldefault('n',n.nspowner))) a WHERE n.oid=ns AND a.grantee=g.oid),false)
 OR NOT coalesce((SELECT count(*)=1 AND bool_and(a.privilege_type='CONNECT' AND NOT a.is_grantable) FROM pg_database d CROSS JOIN LATERAL aclexplode(coalesce(d.datacl,acldefault('d',d.datdba))) a WHERE d.oid=db AND a.grantee=g.oid),false)
 OR EXISTS(SELECT 1 FROM pg_class c WHERE c.relnamespace IN(to_regnamespace('vec_identidad_sesiones_v1'),to_regnamespace('vec_contexto_actor_v1'),to_regnamespace('vec_autorizacion'),to_regnamespace('vec_autorizacion_atestada_v3')) AND c.relkind IN('r','p','v','m','S') AND (has_table_privilege(l.oid,c.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER') OR has_any_column_privilege(l.oid,c.oid,'SELECT,INSERT,UPDATE,REFERENCES')))
 THEN RAISE EXCEPTION 'IS14: consumidor no acreditado' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT cfg FROM vec_identidad_sesiones_v1.config_runtime_admin_preperfil_v1 WHERE identidad_login=session_user FOR SHARE;
 IF cfg.entorno IS DISTINCT FROM p_entorno OR cfg.host_admin IS DISTINCT FROM p_host OR cfg.audiencia IS DISTINCT FROM p_audiencia OR clock_timestamp()>=cfg.vigente_hasta
 THEN RAISE EXCEPTION 'IS14: configuración no vigente' USING ERRCODE='42501'; END IF;
 RETURN cfg.proceso;
EXCEPTION WHEN no_data_found OR too_many_rows THEN RAISE EXCEPTION 'IS14: consumidor sin configuración' USING ERRCODE='42501';
END $f$;
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.exigir_runtime_admin_preperfil_v1(text,text,text) FROM PUBLIC;

-- Sólo recibe documentos construidos por las fachadas propietarias. El
-- LOGIN no tiene EXECUTE y no puede escribir la observación ni el evento.
CREATE FUNCTION vec_identidad_sesiones_v1.auditar_observacion_admin_preperfil_v1(p_evento text,p_correlacion text,p_actor text,p_accion text,p_recurso text,p_resultado text,p_motivo text,p_proceso text,p_documento jsonb)
RETURNS text LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $f$
DECLARE fuente text;sha text;prev record;a record;evento jsonb;
BEGIN
 IF p_evento IS NULL OR p_evento !~ '^evento_[0-9a-f]{32}$' OR p_correlacion IS NULL OR p_correlacion !~ '^correlacion_[0-9a-f]{32}$' THEN RAISE EXCEPTION 'IS14: correlación inválida' USING ERRCODE='22023'; END IF;
 fuente:='observacion_admin_preperfil:'||substring(p_evento FROM 8);
 sha:=encode(sha256(convert_to(p_documento::text,'UTF8')),'hex');
 SELECT * INTO prev FROM vec_identidad_sesiones_v1.observacion_admin_preperfil_v1 WHERE evento_ref=p_evento;
 IF FOUND THEN
  IF prev.documento IS DISTINCT FROM p_documento THEN RAISE EXCEPTION 'IS14: evento con material divergente' USING ERRCODE='23505'; END IF;
 ELSE INSERT INTO vec_identidad_sesiones_v1.observacion_admin_preperfil_v1(evento_ref,fuente_ref,fuente_sha256,documento) VALUES(p_evento,fuente,sha,p_documento);
 END IF;
 evento:=jsonb_build_object('tipo_registro','preperfil_autenticado','evento_ref',p_evento,'actor_ref',p_actor,'accion',p_accion,'recurso_ref',p_recurso,'resultado',p_resultado,'motivo_ref',p_motivo,'proceso',p_proceso,'canal','administracion_privilegiada','finalidad_ref','seleccion_perfil','correlacion_ref',p_correlacion,'fuente_ref',fuente,'fuente_sha256',sha);
 SELECT * INTO STRICT a FROM vec_autorizacion_atestada_v3.registrar_evento_admin_preperfil_v1(evento);
 IF a.auditoria_ref IS NULL OR a.correlacion_ref IS DISTINCT FROM p_correlacion THEN RAISE EXCEPTION 'IS14: auditoría no confirmada' USING ERRCODE='55000'; END IF;
 RETURN a.auditoria_ref;
END $f$;
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.auditar_observacion_admin_preperfil_v1(text,text,text,text,text,text,text,text,jsonb) FROM PUBLIC;

CREATE FUNCTION vec_identidad_sesiones_v1.ejecutar_selector_admin_auditado_propietaria_v1(
 p_entorno text,p_host text,p_audiencia text,p_certificado text,p_ca text,
 p_autenticada timestamptz,p_revocada timestamptz,p_crl_hasta timestamptz,p_certificado_hasta timestamptz,
 p_perfil text,p_revision numeric,p_evento text,p_correlacion text,p_seleccion boolean)
RETURNS TABLE(resultado jsonb,auditoria_comun_ref text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $f$
DECLARE r record;proceso text;hasta timestamptz;config_hasta timestamptz;doc jsonb;salida jsonb;audit text;
 accion text;estado text:='permitido';motivo text:='identidad_y_perfiles_propios_vigentes';
BEGIN
 proceso:=vec_identidad_sesiones_v1.exigir_runtime_admin_preperfil_v1(p_entorno,p_host,p_audiencia);
 SELECT vigente_hasta INTO STRICT config_hasta FROM vec_identidad_sesiones_v1.config_runtime_admin_preperfil_v1 WHERE identidad_login=session_user FOR SHARE;
 IF p_evento IS NULL OR p_evento !~ '^evento_[0-9a-f]{32}$' OR p_correlacion IS NULL OR p_correlacion !~ '^correlacion_[0-9a-f]{32}$'
 THEN RAISE EXCEPTION 'IS14: evento o correlación inválidos' USING ERRCODE='22023'; END IF;
 -- Sin actor autenticado no se fabrica un registro preperfil. La frontera F
 -- conserva responsabilidad de auditar los fallos de autenticación mTLS.
 SELECT * INTO STRICT r FROM vec_identidad_sesiones_v1.resolver_identidad_admin_perfiles_propietaria_v1(p_entorno,p_host,p_audiencia,p_certificado,p_ca,p_autenticada,p_revocada);
 accion:=CASE WHEN p_seleccion THEN 'seleccionar_perfil_admin' ELSE 'listar_perfiles_propios_admin' END;
 doc:=jsonb_build_object('esquema','vec.admin.observacion-preperfil.v1','evento_ref',p_evento,'correlacion_ref',p_correlacion,
  'actor_ref',r.persona_ref,'cuenta_ref',r.cuenta_ref,'cuenta_ordinaria_ref',r.cuenta_ordinaria_ref,
  'vinculo_ref',r.vinculo_ref,'vinculo_version',r.vinculo_version,'politica_ref',r.politica_ref,'politica_sha256',r.politica_huella_sha256,
  'entorno',p_entorno,'host',p_host,'audiencia',p_audiencia,'autenticada_en',p_autenticada,'revocacion_verificada_en',p_revocada,
  'crl_hasta',p_crl_hasta,'certificado_hasta',p_certificado_hasta,'identidad_hasta',r.vigente_hasta,
  'accion',accion,'perfil_solicitado_ref',p_perfil,'revision_esperada',p_revision,
  'proceso',proceso,'config_runtime_vigente_hasta',config_hasta,'canal','administracion_privilegiada','finalidad_ref','seleccion_perfil');
 BEGIN
  IF p_crl_hasta IS NULL OR p_certificado_hasta IS NULL OR NOT isfinite(p_crl_hasta) OR NOT isfinite(p_certificado_hasta)
  THEN RAISE EXCEPTION 'IS14: observación inválida' USING ERRCODE='42501'; END IF;
  hasta:=LEAST(r.vigente_hasta,p_crl_hasta,p_certificado_hasta,config_hasta);
  IF clock_timestamp()>=hasta THEN RAISE EXCEPTION 'IS14: observación caducada' USING ERRCODE='42501'; END IF;
  salida:=vec_contexto_actor_v1.listar_admin_preperfil_propietaria_v1(r.cuenta_ref,r.persona_ref,p_audiencia);
  IF jsonb_array_length(salida->'perfiles')=0 THEN RAISE EXCEPTION 'IS14: no hay perfil propio acreditado' USING ERRCODE='42501'; END IF;
  hasta:=LEAST(hasta,(salida->>'perfiles_vigentes_hasta')::timestamptz);
  salida:=salida-'perfiles_vigentes_hasta';
  IF p_seleccion THEN
   -- Auditoría y CAS comparten subtransacción: un CAS denegado elimina el
   -- supuesto éxito antes de registrar fuera su denegación real.
   audit:=vec_identidad_sesiones_v1.auditar_observacion_admin_preperfil_v1(p_evento,p_correlacion,r.persona_ref,accion,r.cuenta_ref,'permitido',motivo,proceso,doc||jsonb_build_object('resultado','permitido','motivo_ref',motivo));
   IF clock_timestamp()>=hasta THEN RAISE EXCEPTION 'IS14: observación caducada antes del CAS' USING ERRCODE='42501'; END IF;
   salida:=vec_contexto_actor_v1.seleccionar_admin_preperfil_propietaria_v1(r.cuenta_ref,r.persona_ref,p_audiencia,p_perfil,p_revision,audit);
  ELSE
   audit:=vec_identidad_sesiones_v1.auditar_observacion_admin_preperfil_v1(p_evento,p_correlacion,r.persona_ref,accion,r.cuenta_ref,'permitido',motivo,proceso,doc||jsonb_build_object('resultado','permitido','motivo_ref',motivo,'listado',salida));
  END IF;
  IF clock_timestamp()>=hasta THEN RAISE EXCEPTION 'IS14: observación caducada antes del retorno' USING ERRCODE='42501'; END IF;
 -- Sólo el CAS deliberado CA31 es una denegación funcional. Un 40001
 -- de PostgreSQL, CA20 o AD171 se propaga para reintentar la TX completa.
 EXCEPTION WHEN insufficient_privilege OR invalid_parameter_value OR SQLSTATE 'VCA31' OR no_data_found OR too_many_rows THEN
  estado:='denegado';
  motivo:=CASE SQLSTATE WHEN 'VCA31' THEN 'seleccion_revision_obsoleta' WHEN '22023' THEN 'seleccion_material_invalido' ELSE 'perfil_propio_no_acreditado' END;
  salida:=jsonb_build_object('estado',estado,'motivo_ref',motivo);
  audit:=NULL;
 END;
 IF estado='denegado' THEN
  audit:=vec_identidad_sesiones_v1.auditar_observacion_admin_preperfil_v1(p_evento,p_correlacion,r.persona_ref,accion,r.cuenta_ref,estado,motivo,proceso,doc||jsonb_build_object('resultado',estado,'motivo_ref',motivo));
 END IF;
 RETURN QUERY SELECT salida,audit;
EXCEPTION WHEN no_data_found OR too_many_rows THEN RAISE EXCEPTION 'IS14: identidad preperfil no acreditada' USING ERRCODE='42501';
END $f$;
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.ejecutar_selector_admin_auditado_propietaria_v1(text,text,text,text,text,timestamptz,timestamptz,timestamptz,timestamptz,text,numeric,text,text,boolean) FROM PUBLIC;

CREATE FUNCTION vec_identidad_sesiones_v1.listar_perfiles_admin_auditado_v1(
 p_entorno text,p_host text,p_audiencia text,p_certificado text,p_ca text,p_autenticada timestamptz,p_revocada timestamptz,p_crl_vigente_hasta timestamptz,p_certificado_vigente_hasta timestamptz,p_evento_ref text,p_correlacion_ref text)
RETURNS TABLE(resultado jsonb,auditoria_comun_ref text)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $f$
 SELECT * FROM vec_identidad_sesiones_v1.ejecutar_selector_admin_auditado_propietaria_v1(p_entorno,p_host,p_audiencia,p_certificado,p_ca,p_autenticada,p_revocada,p_crl_vigente_hasta,p_certificado_vigente_hasta,NULL,NULL,p_evento_ref,p_correlacion_ref,false)
$f$;
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.listar_perfiles_admin_auditado_v1(text,text,text,text,text,timestamptz,timestamptz,timestamptz,timestamptz,text,text) FROM PUBLIC;
CREATE FUNCTION vec_identidad_sesiones_v1.seleccionar_perfil_admin_auditado_v1(
 p_entorno text,p_host text,p_audiencia text,p_certificado text,p_ca text,p_autenticada timestamptz,p_revocada timestamptz,p_crl_vigente_hasta timestamptz,p_certificado_vigente_hasta timestamptz,p_perfil_ref text,p_revision_esperada numeric,p_evento_ref text,p_correlacion_ref text)
RETURNS TABLE(resultado jsonb,auditoria_comun_ref text)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $f$
 SELECT * FROM vec_identidad_sesiones_v1.ejecutar_selector_admin_auditado_propietaria_v1(p_entorno,p_host,p_audiencia,p_certificado,p_ca,p_autenticada,p_revocada,p_crl_vigente_hasta,p_certificado_vigente_hasta,p_perfil_ref,p_revision_esperada,p_evento_ref,p_correlacion_ref,true)
$f$;
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.seleccionar_perfil_admin_auditado_v1(text,text,text,text,text,timestamptz,timestamptz,timestamptz,timestamptz,text,numeric,text,text) FROM PUBLIC;

CREATE FUNCTION vec_identidad_sesiones_v1.acreditar_runtime_preperfil_admin_v1()
RETURNS TABLE(acreditada boolean,proceso text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $f$
DECLARE cfg record;p text;
BEGIN
 SELECT * INTO STRICT cfg FROM vec_identidad_sesiones_v1.config_runtime_admin_preperfil_v1 WHERE identidad_login=session_user;
 p:=vec_identidad_sesiones_v1.exigir_runtime_admin_preperfil_v1(cfg.entorno,cfg.host_admin,cfg.audiencia);
 RETURN QUERY SELECT true,p;
EXCEPTION WHEN no_data_found OR too_many_rows THEN RAISE EXCEPTION 'IS14: consumidor no configurado' USING ERRCODE='42501';
END $f$;
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.acreditar_runtime_preperfil_admin_v1() FROM PUBLIC;
RESET ROLE;
CREATE ROLE vec_identidad_sesiones_v1_admin_preperfil NOLOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
DO $conexion$ BEGIN
 EXECUTE format('GRANT CONNECT ON DATABASE %I TO vec_identidad_sesiones_v1_admin_preperfil',current_database());
END $conexion$;
GRANT USAGE ON SCHEMA vec_identidad_sesiones_v1 TO vec_identidad_sesiones_v1_admin_preperfil;
GRANT EXECUTE ON FUNCTION vec_identidad_sesiones_v1.acreditar_runtime_preperfil_admin_v1(),vec_identidad_sesiones_v1.listar_perfiles_admin_auditado_v1(text,text,text,text,text,timestamptz,timestamptz,timestamptz,timestamptz,text,text),vec_identidad_sesiones_v1.seleccionar_perfil_admin_auditado_v1(text,text,text,text,text,timestamptz,timestamptz,timestamptz,timestamptz,text,numeric,text,text) TO vec_identidad_sesiones_v1_admin_preperfil;
COMMIT;
