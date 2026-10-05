\set ON_ERROR_STOP on
-- IS16: cuenta nominal y vínculo exacto de sesión antes de contexto V2.
-- Orden: AD194 -> IS16 -> CA36 (deploy/principal/lista_sql_codexk_admin_runtime_20261005.txt).
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_identidad_sesiones_v1:migracion:000016',0));
DO $pre$ BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 OR to_regprocedure('vec_identidad_sesiones_v1.resolver_identidad_admin_perfiles_propietaria_v1(text,text,text,text,text,timestamptz,timestamptz)') IS NULL
 OR to_regprocedure('vec_contexto_actor_v1.listar_admin_preperfil_propietaria_v1(text,text,text)') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_contexto_admin_pre_v2_is_v1(jsonb)') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.cotejar_contexto_admin_pre_v2_is_v1(jsonb)') IS NULL
 OR to_regclass('vec_identidad_sesiones_v1.vinculo_sesion_admin_v1') IS NOT NULL
 OR to_regrole('vec_identidad_sesiones_v1_admin_perfiles_runtime') IS NOT NULL
 THEN RAISE EXCEPTION 'IS16: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
-- AD192 tal como se instaló falla al repetir un evento (secuencia numeric en
-- una salida bigint). AD194 lo corrige. Se exige que la cadena defectuosa ya
-- no esté en vigor sin depender del texto exacto de la corrección: basta con
-- que la fachada ya no llame a la interna de AD192 o que esta haya cambiado.
DO $ad194$ BEGIN
 IF EXISTS(SELECT 1 FROM(VALUES
   ('registrar_contexto_admin_pre_v2_is_v1(jsonb)','registrar_contexto_admin_pre_v2_interna_v1','90219f669dccc426c8ba66958937bb0a85d028ae515868b40cdea788aec2fdf1'),
   ('cotejar_contexto_admin_pre_v2_is_v1(jsonb)','cotejar_contexto_admin_pre_v2_interna_v1','ff9c375abc732eb4ace4d2bc942b130cabe7623c9874e63802798aa450892b4c')) d(fachada,interna,sha_ad192)
  JOIN pg_proc f ON f.oid=to_regprocedure('vec_autorizacion_atestada_v3.'||d.fachada)
  JOIN pg_proc i ON i.oid=to_regprocedure('vec_autorizacion_atestada_v3.'||d.interna||'(jsonb,text)')
  WHERE position(d.interna||'(' IN f.prosrc)>0 AND encode(sha256(convert_to(i.prosrc,'UTF8')),'hex')=d.sha_ad192)
 THEN RAISE EXCEPTION 'IS16: falta AD194 (repetición AD192 sin corregir)' USING ERRCODE='55000'; END IF;
END $ad194$;
CREATE ROLE vec_identidad_sesiones_v1_admin_perfiles_runtime NOLOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
SET LOCAL ROLE vec_identidad_sesiones_v1_propietario;
CREATE TABLE vec_identidad_sesiones_v1.config_runtime_admin_perfiles_v1(
 identidad_login text PRIMARY KEY,
 proceso text NOT NULL CHECK(proceso ~ '^[a-z][a-z0-9._-]{1,79}$'),
 entorno text NOT NULL CHECK(entorno IN('desarrollo','cidonia')),
 -- Mismo formato de nombre de host que la política de certificado de IS15.
 host_admin text NOT NULL CHECK(octet_length(host_admin) BETWEEN 4 AND 253
  AND host_admin ~ '^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?(\.[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?)+$'),
 audiencia text NOT NULL CHECK(audiencia ~ '^[a-z0-9][a-z0-9._:-]{3,255}$'),
 -- PostgreSQL no admite repeticiones {m,n} mayores que 255: la longitud se
 -- limita aparte (8 de https:// más 500 como máximo).
 espacio_identidad text NOT NULL CHECK(espacio_identidad ~ '^https://[^[:space:]]+$' AND length(espacio_identidad)<=508),
 configurada_en timestamptz NOT NULL DEFAULT clock_timestamp(),
 vigente_hasta timestamptz NOT NULL CHECK(isfinite(vigente_hasta)),
 CHECK(vigente_hasta>configurada_en)
);
CREATE TABLE vec_identidad_sesiones_v1.vinculo_sesion_admin_v1(
 referencia text PRIMARY KEY CHECK(referencia ~ '^vis_[0-9a-f]{32}$'),
 version numeric(20,0) NOT NULL CHECK(version=1),
 huella_sha256 text NOT NULL CHECK(huella_sha256 ~ '^[0-9a-f]{64}$'),
 autenticacion_ref text NOT NULL,
 sesion_ref text NOT NULL,
 cuenta_ref text NOT NULL REFERENCES vec_identidad_sesiones_v1.cuenta(cuenta_ref),
 documento jsonb NOT NULL,
 observacion jsonb NOT NULL,
 plan_sha256 text NOT NULL CHECK(plan_sha256 ~ '^[0-9a-f]{64}$'),
 operador_login text NOT NULL,
 evento jsonb NOT NULL,
 acuse jsonb NOT NULL,
 UNIQUE(autenticacion_ref,sesion_ref),
 CHECK(huella_sha256=encode(sha256(convert_to((documento-'huella_sha256'-'fuente_sha256')::text,'UTF8')),'hex')),
 CHECK(documento->>'referencia'=referencia AND(documento->>'version')::numeric=version
  AND documento->>'autenticacion_ref'=autenticacion_ref AND documento->>'sesion_ref'=sesion_ref
  AND documento->>'cuenta_ref'=cuenta_ref AND documento->>'huella_sha256'=huella_sha256
  AND documento->>'fuente_sha256'=huella_sha256)
);
DO $tablas$ DECLARE t text; BEGIN
 FOREACH t IN ARRAY ARRAY['config_runtime_admin_perfiles_v1','vinculo_sesion_admin_v1'] LOOP
  EXECUTE format('ALTER TABLE vec_identidad_sesiones_v1.%I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE format('ALTER TABLE vec_identidad_sesiones_v1.%I FORCE ROW LEVEL SECURITY',t);
  EXECUTE format('CREATE POLICY propietario_exacto ON vec_identidad_sesiones_v1.%I TO vec_identidad_sesiones_v1_propietario USING(current_user=%L) WITH CHECK(current_user=%L)',t,'vec_identidad_sesiones_v1_propietario','vec_identidad_sesiones_v1_propietario');
  EXECUTE format('CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE OR TRUNCATE ON vec_identidad_sesiones_v1.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_identidad_sesiones_v1.rechazar_mutacion()',t);
  EXECUTE format('REVOKE ALL ON TABLE vec_identidad_sesiones_v1.%I FROM PUBLIC',t);
  EXECUTE format('REVOKE ALL ON TYPE vec_identidad_sesiones_v1.%I FROM PUBLIC',t);
 END LOOP;
END $tablas$;

-- Acreditación exclusiva: cuatro fachadas, USAGE y CONNECT. No tablas, SET ROLE,
-- membresías adicionales, ajustes de rol ni concesiones con grant option.
CREATE FUNCTION vec_identidad_sesiones_v1.exigir_runtime_admin_perfiles_v1()
RETURNS vec_identidad_sesiones_v1.config_runtime_admin_perfiles_v1
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $f$
DECLARE c vec_identidad_sesiones_v1.config_runtime_admin_perfiles_v1;l record;g record;fs oid[];ns oid;db oid;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
 OR current_setting('TimeZone')<>'UTC' OR current_setting('role')<>'none' OR pg_is_in_recovery()
 THEN RAISE EXCEPTION 'IS16: requiere SERIALIZABLE READ WRITE UTC' USING ERRCODE='25000'; END IF;
 SELECT * INTO l FROM pg_roles WHERE rolname=session_user;
 SELECT * INTO g FROM pg_roles WHERE rolname='vec_identidad_sesiones_v1_admin_perfiles_runtime';
 fs:=ARRAY[to_regprocedure('vec_identidad_sesiones_v1.acreditar_runtime_admin_perfiles_v1()'),to_regprocedure('vec_identidad_sesiones_v1.rechazar_fuente_cuenta_admin_v1(text,text,text)'),
  to_regprocedure('vec_identidad_sesiones_v1.resolver_cuenta_admin_perfiles_v1(text,text,text,text,text,timestamptz,timestamptz,timestamptz,timestamptz,text,text)'),
  to_regprocedure('vec_identidad_sesiones_v1.vincular_sesion_admin_perfiles_v1(text,text,text,text,text,timestamptz,timestamptz,timestamptz,timestamptz,text,text,jsonb,text,text,text)')];
 ns:=to_regnamespace('vec_identidad_sesiones_v1');
 SELECT oid INTO db FROM pg_database WHERE datname=current_database();
 IF l.oid IS NULL OR g.oid IS NULL OR NOT l.rolcanlogin OR NOT l.rolinherit
 OR l.rolsuper OR l.rolcreaterole OR l.rolcreatedb OR l.rolreplication OR l.rolbypassrls OR l.rolconfig IS NOT NULL
 OR(l.rolvaliduntil IS NOT NULL AND clock_timestamp()>=l.rolvaliduntil)
 OR g.rolcanlogin OR g.rolinherit OR g.rolsuper OR g.rolcreaterole OR g.rolcreatedb OR g.rolreplication OR g.rolbypassrls OR g.rolconfig IS NOT NULL
 OR g.rolconnlimit<>-1 OR g.rolvaliduntil IS NOT NULL OR array_position(fs,NULL) IS NOT NULL
 OR(SELECT count(*) FROM pg_auth_members WHERE member IN(l.oid,g.oid) OR roleid IN(l.oid,g.oid) OR grantor IN(l.oid,g.oid))<>1
 OR NOT EXISTS(SELECT 1 FROM pg_auth_members m JOIN pg_roles otorgante ON otorgante.oid=m.grantor
  WHERE m.member=l.oid AND m.roleid=g.oid AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option AND otorgante.rolsuper)
 OR EXISTS(SELECT 1 FROM pg_db_role_setting WHERE setrole IN(l.oid,g.oid))
 OR EXISTS(SELECT 1 FROM pg_shdepend WHERE refclassid='pg_catalog.pg_authid'::regclass AND refobjid=l.oid)
 OR EXISTS(SELECT 1 FROM pg_default_acl d LEFT JOIN LATERAL aclexplode(coalesce(d.defaclacl,'{}'::aclitem[])) a ON true WHERE d.defaclrole IN(l.oid,g.oid) OR a.grantee IN(l.oid,g.oid) OR a.grantor IN(l.oid,g.oid))
 OR EXISTS(SELECT 1 FROM pg_policy p WHERE l.oid=ANY(p.polroles) OR g.oid=ANY(p.polroles))
 OR NOT coalesce((SELECT count(*)=6 AND bool_and(deptype='a' AND objsubid=0 AND((classid='pg_catalog.pg_database'::regclass AND objid=db) OR(classid='pg_catalog.pg_namespace'::regclass AND objid=ns) OR(classid='pg_catalog.pg_proc'::regclass AND objid=ANY(fs)))) FROM pg_shdepend WHERE refclassid='pg_catalog.pg_authid'::regclass AND refobjid=g.oid),false)
 OR NOT coalesce((SELECT count(*)=4 AND bool_and(a.privilege_type='EXECUTE' AND NOT a.is_grantable) FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=ANY(fs) AND a.grantee=g.oid),false)
 OR NOT coalesce((SELECT count(*)=1 AND bool_and(a.privilege_type='USAGE' AND NOT a.is_grantable) FROM pg_namespace n CROSS JOIN LATERAL aclexplode(coalesce(n.nspacl,acldefault('n',n.nspowner))) a WHERE n.oid=ns AND a.grantee=g.oid),false)
 OR NOT coalesce((SELECT count(*)=1 AND bool_and(a.privilege_type='CONNECT' AND NOT a.is_grantable) FROM pg_database d CROSS JOIN LATERAL aclexplode(coalesce(d.datacl,acldefault('d',d.datdba))) a WHERE d.oid=db AND a.grantee=g.oid),false)
 OR EXISTS(SELECT 1 FROM pg_class t JOIN pg_namespace n ON n.oid=t.relnamespace WHERE n.nspname LIKE 'vec\_%' ESCAPE '\' AND t.relkind IN('r','p','v','m','S') AND(has_table_privilege(l.oid,t.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER') OR has_any_column_privilege(l.oid,t.oid,'SELECT,INSERT,UPDATE,REFERENCES')))
 THEN RAISE EXCEPTION 'IS16: consumidor no acreditado' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT c FROM vec_identidad_sesiones_v1.config_runtime_admin_perfiles_v1 WHERE identidad_login=session_user FOR SHARE;
 IF clock_timestamp()>=c.vigente_hasta THEN RAISE EXCEPTION 'IS16: configuración caducada' USING ERRCODE='42501'; END IF;
 RETURN c;
EXCEPTION WHEN no_data_found OR too_many_rows THEN RAISE EXCEPTION 'IS16: configuración ausente' USING ERRCODE='42501';
END $f$;
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.exigir_runtime_admin_perfiles_v1() FROM PUBLIC;
CREATE FUNCTION vec_identidad_sesiones_v1.acreditar_runtime_admin_perfiles_v1()
RETURNS TABLE(identidad_login text,acreditada boolean,proceso text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE c vec_identidad_sesiones_v1.config_runtime_admin_perfiles_v1; BEGIN
 c:=vec_identidad_sesiones_v1.exigir_runtime_admin_perfiles_v1();
 RETURN QUERY SELECT session_user::text,true,c.proceso;
END $f$;
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.acreditar_runtime_admin_perfiles_v1() FROM PUBLIC;

-- Devuelve coordenadas propietarias, nunca preimágenes ni identificadores civiles.
CREATE FUNCTION vec_identidad_sesiones_v1.cuenta_admin_propietaria_v1(
 p_entorno text,p_host text,p_audiencia text,p_certificado text,p_ca text,p_autenticada timestamptz,p_revocada timestamptz,
 p_crl_hasta timestamptz,p_certificado_hasta timestamptz,p_espacio text,p_config_hasta timestamptz)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $f$
DECLARE i record;t record;ao record;ap record;f record;lista jsonb;perfil text;hasta timestamptz;
BEGIN
 IF p_crl_hasta IS NULL OR p_certificado_hasta IS NULL OR NOT isfinite(p_crl_hasta) OR NOT isfinite(p_certificado_hasta)
 THEN RAISE EXCEPTION 'IS16: observación inválida' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT i FROM vec_identidad_sesiones_v1.resolver_identidad_admin_perfiles_propietaria_v1(p_entorno,p_host,p_audiencia,p_certificado,p_ca,p_autenticada,p_revocada);
 lista:=vec_contexto_actor_v1.listar_admin_preperfil_propietaria_v1(i.cuenta_ref,i.persona_ref,p_audiencia);
 perfil:=lista->>'perfil_activo_ref';
 IF perfil IS NULL OR perfil='' OR(lista->>'revision')::numeric<1
 OR NOT EXISTS(SELECT 1 FROM jsonb_array_elements(lista->'perfiles') x WHERE x->>'perfil_ref'=perfil AND x->>'categoria_admin'='aplicacion')
 THEN RAISE EXCEPTION 'IS16: selección propia pendiente' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT t FROM vec_identidad_sesiones_v1.titularidad_cuenta_persona_v1 WHERE cuenta_ref=i.cuenta_ref AND persona_ref=i.persona_ref;
 SELECT * INTO STRICT f FROM vec_identidad_sesiones_v1.fuentes_iniciales_admin_v1 WHERE operacion_ref=t.operacion_ref;
 -- La pareja procede del acto inicial exacto; una rotación no elige un alias
 -- por fecha ni cambia silenciosamente las coordenadas del proveedor original.
 SELECT * INTO STRICT ap FROM vec_identidad_sesiones_v1.alias_hmac_cuenta a JOIN vec_identidad_sesiones_v1.cuenta c USING(cuenta_ref)
 WHERE a.cuenta_ref=i.cuenta_ref AND a.acto_ref=c.acto_ref;
 SELECT * INTO STRICT ao FROM vec_identidad_sesiones_v1.alias_hmac_cuenta a JOIN vec_identidad_sesiones_v1.cuenta c USING(cuenta_ref)
 WHERE a.cuenta_ref=i.cuenta_ordinaria_ref AND a.acto_ref=c.acto_ref;
 IF ap.esquema_hmac IS DISTINCT FROM ao.esquema_hmac OR ap.dominio_hmac_ref IS DISTINCT FROM ao.dominio_hmac_ref
 OR ap.clave_hmac_id IS DISTINCT FROM ao.clave_hmac_id OR ap.clave_hmac_version IS DISTINCT FROM ao.clave_hmac_version
 OR ap.sujeto_id_hmac IS DISTINCT FROM ao.sujeto_id_hmac
 OR NOT EXISTS(SELECT 1 FROM vec_identidad_sesiones_v1.titularidad_cuenta_persona_v1 o WHERE o.cuenta_ref=i.cuenta_ordinaria_ref AND o.persona_ref=i.persona_ref AND o.operacion_ref=t.operacion_ref AND clock_timestamp()<o.vigente_hasta)
 THEN RAISE EXCEPTION 'IS16: fuente propietaria divergente' USING ERRCODE='42501'; END IF;
 hasta:=LEAST(i.vigente_hasta,(lista->>'perfiles_vigentes_hasta')::timestamptz,t.vigente_hasta,p_crl_hasta,p_certificado_hasta,p_config_hasta);
 IF clock_timestamp()>=hasta THEN RAISE EXCEPTION 'IS16: autoridad caducada' USING ERRCODE='42501'; END IF;
 RETURN jsonb_build_object('persona_ref',i.persona_ref,'cuenta_ref',i.cuenta_ref,'cuenta_ordinaria_ref',i.cuenta_ordinaria_ref,
  'perfil_activo_ref',perfil,'rol_id','administracion_perfiles','vinculo_ref',i.vinculo_ref,'vinculo_version',i.vinculo_version,
  'politica_garantia_ref',i.politica_ref,'politica_garantia_huella_sha256',i.politica_huella_sha256,'garantia_observada','alto',
  'vigente_hasta',hasta,'seleccion_revision',(lista->>'revision')::numeric,'espacio_identidad',p_espacio,
  'esquema_hmac',ap.esquema_hmac,'dominio_hmac_ref',ap.dominio_hmac_ref,'clave_hmac_id',ap.clave_hmac_id,'clave_hmac_version',ap.clave_hmac_version,
  'sujeto_hmac_hex',encode(ap.sujeto_id_hmac,'hex'),'cuenta_hmac_hex',encode(ap.cuenta_id_hmac,'hex'),'cuenta_ordinaria_hmac_hex',encode(ao.cuenta_id_hmac,'hex'),
  'fuente_ref',f.plan#>>'{fuente_hmac,referencia}','fuente_sha256',f.plan#>>'{fuente_hmac,huella_sha256}');
END $f$;
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.cuenta_admin_propietaria_v1(text,text,text,text,text,timestamptz,timestamptz,timestamptz,timestamptz,text,timestamptz) FROM PUBLIC;

CREATE FUNCTION vec_identidad_sesiones_v1.auditar_contexto_admin_is16_v1(p_evento text,p_correlacion text,p_proceso text,p_accion text,p_resultado text,p_cuenta jsonb,p_fuente text,p_sha text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE e jsonb;a record; BEGIN
 e:=jsonb_build_object('tipo_registro','contexto_admin_pre_v2','evento_ref',p_evento,'operador_login',session_user::text,
  'actor_ref',p_cuenta->>'persona_ref','perfil_activo_ref',p_cuenta->>'perfil_activo_ref','accion',p_accion,
  'recurso_ref',COALESCE(p_cuenta->>'cuenta_ref','administracion:contexto'),'resultado',p_resultado,
  'motivo_ref','contexto_admin_pre_v2_'||p_resultado,'proceso',p_proceso,'canal','administracion_privilegiada',
  'finalidad_ref','establecer_contexto_admin','correlacion_ref',p_correlacion,'fuente_ref',p_fuente,'fuente_sha256',p_sha);
 SELECT * INTO STRICT a FROM vec_autorizacion_atestada_v3.registrar_contexto_admin_pre_v2_is_v1(e);
 IF a.auditoria_ref IS DISTINCT FROM 'aud_v3_ap2_'||substr(p_evento,8) OR a.secuencia IS NULL OR a.secuencia NOT BETWEEN 1 AND 9007199254740991
 OR a.huella_sha256 IS NULL OR a.huella_sha256 !~ '^[0-9a-f]{64}$' OR a.huella_sha256=repeat('0',64)
 OR a.correlacion_ref IS DISTINCT FROM p_correlacion OR a.registrada_en IS NULL OR NOT isfinite(a.registrada_en)
 THEN RAISE EXCEPTION 'IS16: acuse común inválido' USING ERRCODE='55000'; END IF;
 RETURN jsonb_build_object('evento',e,'acuse',jsonb_build_object('auditoria_ref',a.auditoria_ref,'secuencia',a.secuencia,'huella_sha256',a.huella_sha256,'correlacion_ref',a.correlacion_ref,'registrada_en',a.registrada_en));
END $f$;
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.auditar_contexto_admin_is16_v1(text,text,text,text,text,jsonb,text,text) FROM PUBLIC;

-- Go revierte el SAVEPOINT de la resolución o del vínculo si falla su cotejo
-- (identificadores originales o vínculo devuelto) y registra aquí el error,
-- fuera de ese SAVEPOINT, con la acción que se estaba cotejando.
CREATE FUNCTION vec_identidad_sesiones_v1.rechazar_fuente_cuenta_admin_v1(p_evento text,p_correlacion text,p_accion text)
RETURNS TABLE(resultado jsonb,acuse jsonb)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $f$
DECLARE cfg vec_identidad_sesiones_v1.config_runtime_admin_perfiles_v1;a jsonb; BEGIN
 cfg:=vec_identidad_sesiones_v1.exigir_runtime_admin_perfiles_v1();
 IF p_accion IS NULL OR p_accion NOT IN('resolver_cuenta_admin','vincular_sesion_admin')
 THEN RAISE EXCEPTION 'IS16: acción de rechazo inválida' USING ERRCODE='22023'; END IF;
 a:=vec_identidad_sesiones_v1.auditar_contexto_admin_is16_v1(p_evento,p_correlacion,cfg.proceso,p_accion,'error',NULL,NULL,NULL);
 RETURN QUERY SELECT jsonb_build_object('estado','error','datos',NULL),a->'acuse';
END $f$;
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.rechazar_fuente_cuenta_admin_v1(text,text,text) FROM PUBLIC;

CREATE FUNCTION vec_identidad_sesiones_v1.resolver_cuenta_admin_perfiles_v1(
 p_entorno text,p_host text,p_audiencia text,p_certificado text,p_ca text,p_autenticada timestamptz,p_revocada timestamptz,
 p_crl_hasta timestamptz,p_certificado_hasta timestamptz,p_evento text,p_correlacion text)
RETURNS TABLE(resultado jsonb,acuse jsonb)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $f$
DECLARE cfg vec_identidad_sesiones_v1.config_runtime_admin_perfiles_v1;c jsonb;a jsonb;estado text:='permitido'; BEGIN
 cfg:=vec_identidad_sesiones_v1.exigir_runtime_admin_perfiles_v1();
 BEGIN
  IF cfg.entorno IS DISTINCT FROM p_entorno OR cfg.host_admin IS DISTINCT FROM p_host OR cfg.audiencia IS DISTINCT FROM p_audiencia
  THEN RAISE EXCEPTION 'IS16: configuración divergente' USING ERRCODE='42501'; END IF;
  c:=vec_identidad_sesiones_v1.cuenta_admin_propietaria_v1(p_entorno,p_host,p_audiencia,p_certificado,p_ca,p_autenticada,p_revocada,p_crl_hasta,p_certificado_hasta,cfg.espacio_identidad,cfg.vigente_hasta);
  a:=vec_identidad_sesiones_v1.auditar_contexto_admin_is16_v1(p_evento,p_correlacion,cfg.proceso,'resolver_cuenta_admin',estado,c,c->>'fuente_ref',c->>'fuente_sha256');
  IF clock_timestamp()>=(c->>'vigente_hasta')::timestamptz THEN RAISE EXCEPTION 'IS16: caducidad antes de retorno' USING ERRCODE='42501'; END IF;
 EXCEPTION WHEN insufficient_privilege OR no_data_found OR too_many_rows OR invalid_parameter_value THEN
  c:=NULL;a:=NULL;estado:='denegado';
 WHEN serialization_failure OR deadlock_detected OR query_canceled THEN RAISE;
 WHEN OTHERS THEN c:=NULL;a:=NULL;estado:='error';
 END;
 IF estado<>'permitido' THEN a:=vec_identidad_sesiones_v1.auditar_contexto_admin_is16_v1(p_evento,p_correlacion,cfg.proceso,'resolver_cuenta_admin',estado,NULL,NULL,NULL); END IF;
 RETURN QUERY SELECT jsonb_build_object('estado',estado,'datos',c),a->'acuse';
END $f$;
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.resolver_cuenta_admin_perfiles_v1(text,text,text,text,text,timestamptz,timestamptz,timestamptz,timestamptz,text,text) FROM PUBLIC;

CREATE FUNCTION vec_identidad_sesiones_v1.vincular_sesion_admin_perfiles_v1(
 p_entorno text,p_host text,p_audiencia text,p_certificado text,p_ca text,p_autenticada timestamptz,p_revocada timestamptz,
 p_crl_hasta timestamptz,p_certificado_hasta timestamptz,p_aut text,p_ses text,p_esperada jsonb,p_vis text,p_evento text,p_correlacion text)
RETURNS TABLE(resultado jsonb,acuse jsonb)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $f$
DECLARE cfg vec_identidad_sesiones_v1.config_runtime_admin_perfiles_v1;cfg_final vec_identidad_sesiones_v1.config_runtime_admin_perfiles_v1;c jsonb;s record;v vec_identidad_sesiones_v1.vinculo_sesion_admin_v1;doc jsonb;obs jsonb;plan text;a jsonb;estado text:='permitido';hasta timestamptz;instante timestamptz;h text;metodo text; BEGIN
 cfg:=vec_identidad_sesiones_v1.exigir_runtime_admin_perfiles_v1();
 BEGIN
  IF p_vis IS NULL OR p_vis !~ '^vis_[0-9a-f]{32}$' OR cfg.entorno IS DISTINCT FROM p_entorno OR cfg.host_admin IS DISTINCT FROM p_host OR cfg.audiencia IS DISTINCT FROM p_audiencia
  THEN RAISE EXCEPTION 'IS16: material de vínculo inválido' USING ERRCODE='42501'; END IF;
  c:=vec_identidad_sesiones_v1.cuenta_admin_propietaria_v1(p_entorno,p_host,p_audiencia,p_certificado,p_ca,p_autenticada,p_revocada,p_crl_hasta,p_certificado_hasta,cfg.espacio_identidad,cfg.vigente_hasta);
  IF p_esperada IS DISTINCT FROM(c-'vigente_hasta') THEN RAISE EXCEPTION 'IS16: cuenta o selección cambiadas' USING ERRCODE='42501'; END IF;
  SELECT * INTO STRICT s FROM vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(p_aut,p_ses);
  IF s.cuenta_ref IS DISTINCT FROM c->>'cuenta_ref' OR s.cuenta_ordinaria_ref IS DISTINCT FROM c->>'cuenta_ordinaria_ref'
  OR s.cuenta_privilegiada IS DISTINCT FROM true OR s.superficie IS DISTINCT FROM 'administracion_privilegiada'
  OR (s.metodo_observado IS NULL OR s.metodo_observado NOT IN('certificado','dnie')) OR s.garantia_observada IS DISTINCT FROM 'alto'
  OR s.politica_garantia_ref IS DISTINCT FROM c->>'politica_garantia_ref' OR s.politica_garantia_huella_sha256 IS DISTINCT FROM c->>'politica_garantia_huella_sha256'
  OR s.autenticacion_verificada_en IS DISTINCT FROM p_autenticada
  THEN RAISE EXCEPTION 'IS16: sesión divergente' USING ERRCODE='42501'; END IF;
  metodo:=s.metodo_observado;
  obs:=jsonb_build_object('entorno',p_entorno,'host',p_host,'audiencia',p_audiencia,'certificado',p_certificado,'ca',p_ca,'autenticada',p_autenticada,'revocada',p_revocada,'crl_hasta',p_crl_hasta,'certificado_hasta',p_certificado_hasta);
  plan:=encode(sha256(convert_to(jsonb_build_object('observacion',obs,'esperada',p_esperada,'aut',p_aut,'ses',p_ses,'vis',p_vis,'evento',p_evento,'correlacion',p_correlacion)::text,'UTF8')),'hex');
  PERFORM pg_advisory_xact_lock(hashtextextended('vec:is16:sesion:'||p_ses,0));
  SELECT * INTO v FROM vec_identidad_sesiones_v1.vinculo_sesion_admin_v1 WHERE autenticacion_ref=p_aut AND sesion_ref=p_ses;
  IF FOUND THEN
   IF v.plan_sha256 IS DISTINCT FROM plan OR v.referencia IS DISTINCT FROM p_vis OR v.operador_login IS DISTINCT FROM session_user::text THEN RAISE EXCEPTION 'IS16: replay divergente' USING ERRCODE='23505'; END IF;
   SELECT * INTO STRICT s FROM vec_autorizacion_atestada_v3.cotejar_contexto_admin_pre_v2_is_v1(v.evento);
   a:=jsonb_build_object('auditoria_ref',s.auditoria_ref,'secuencia',s.secuencia,'huella_sha256',s.huella_sha256,'correlacion_ref',s.correlacion_ref,'registrada_en',s.registrada_en);
   IF a IS DISTINCT FROM v.acuse THEN RAISE EXCEPTION 'IS16: acuse original ausente' USING ERRCODE='55000'; END IF;
   -- El ACK puede esperar un cerrojo. Revalidar después de esa espera, dentro
   -- de la misma subtransacción, sin reescribir el evento ni el vínculo.
   cfg_final:=vec_identidad_sesiones_v1.exigir_runtime_admin_perfiles_v1();
   IF cfg_final IS DISTINCT FROM cfg
   OR vec_identidad_sesiones_v1.cuenta_admin_propietaria_v1(p_entorno,p_host,p_audiencia,p_certificado,p_ca,p_autenticada,p_revocada,p_crl_hasta,p_certificado_hasta,cfg_final.espacio_identidad,cfg_final.vigente_hasta) IS DISTINCT FROM c
   OR vec_identidad_sesiones_v1.cotejar_vinculo_sesion_admin_v1(v.referencia,v.version,v.huella_sha256,p_aut,p_ses,v.cuenta_ref,v.documento->>'perfil_activo_ref',clock_timestamp(),metodo) IS NULL
   THEN RAISE EXCEPTION 'IS16: replay sin autoridad vigente tras acuse' USING ERRCODE='42501'; END IF;
   RETURN QUERY SELECT jsonb_build_object('estado','permitido','datos',v.documento),v.acuse;RETURN;
  END IF;
  instante:=clock_timestamp();hasta:=LEAST((c->>'vigente_hasta')::timestamptz,s.sesion_valida_hasta);
  IF instante>=hasta THEN RAISE EXCEPTION 'IS16: sesión caducada' USING ERRCODE='42501'; END IF;
  doc:=jsonb_build_object('referencia',p_vis,'version',1,'autenticacion_ref',p_aut,'sesion_ref',p_ses,'persona_ref',c->>'persona_ref',
   'cuenta_ref',c->>'cuenta_ref','cuenta_ordinaria_ref',c->>'cuenta_ordinaria_ref','perfil_activo_ref',c->>'perfil_activo_ref',
   'certificado_sha256',p_certificado,'ca_sha256',p_ca,'vinculo_certificado_ref',c->>'vinculo_ref','vinculo_certificado_version',c->'vinculo_version',
   'politica_ref',c->>'politica_garantia_ref','politica_sha256',c->>'politica_garantia_huella_sha256','seleccion_revision',c->'seleccion_revision',
   'control_sesion_ref',s.control_sesion_ref,'control_sesion_revision',s.control_sesion_revision::numeric,'control_sesion_sha256',s.control_sesion_huella_sha256,
   'vinculada_en',instante,'vigente_hasta',hasta,'fuente_ref','vinculo_sesion_admin:'||substr(p_vis,5));
  h:=encode(sha256(convert_to(doc::text,'UTF8')),'hex');doc:=doc||jsonb_build_object('huella_sha256',h,'fuente_sha256',h);
  a:=vec_identidad_sesiones_v1.auditar_contexto_admin_is16_v1(p_evento,p_correlacion,cfg.proceso,'vincular_sesion_admin','permitido',c,doc->>'fuente_ref',h);
  INSERT INTO vec_identidad_sesiones_v1.vinculo_sesion_admin_v1 VALUES(p_vis,1,h,p_aut,p_ses,c->>'cuenta_ref',doc,obs,plan,session_user,a->'evento',a->'acuse');
  IF clock_timestamp()>=hasta THEN RAISE EXCEPTION 'IS16: caducidad antes de vínculo' USING ERRCODE='42501'; END IF;
 EXCEPTION WHEN insufficient_privilege OR no_data_found OR too_many_rows OR invalid_parameter_value THEN
  c:=NULL;doc:=NULL;a:=NULL;estado:='denegado';
 WHEN serialization_failure OR deadlock_detected OR query_canceled THEN RAISE;
 WHEN OTHERS THEN c:=NULL;doc:=NULL;a:=NULL;estado:='error';
 END;
 IF estado<>'permitido' THEN a:=vec_identidad_sesiones_v1.auditar_contexto_admin_is16_v1(p_evento,p_correlacion,cfg.proceso,'vincular_sesion_admin',estado,NULL,NULL,NULL); END IF;
 RETURN QUERY SELECT jsonb_build_object('estado',estado,'datos',doc),a->'acuse';
END $f$;
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.vincular_sesion_admin_perfiles_v1(text,text,text,text,text,timestamptz,timestamptz,timestamptz,timestamptz,text,text,jsonb,text,text,text) FROM PUBLIC;

-- Sólo el owner CA puede consultar la ligadura. En READ COMMITTED CA36
-- coteja además su selección CA31 actual, bajo sus propios cerrojos.
CREATE FUNCTION vec_identidad_sesiones_v1.cotejar_vinculo_sesion_admin_v1(p_ref text,p_version numeric,p_sha text,p_aut text,p_ses text,p_cuenta text,p_perfil text,p_solicitado timestamptz,p_metodo text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $f$
DECLARE v vec_identidad_sesiones_v1.vinculo_sesion_admin_v1;i record;s record;lista jsonb;ahora timestamptz; BEGIN
 IF current_setting('transaction_isolation') NOT IN('serializable','read committed') OR current_setting('transaction_read_only')<>'off'
 OR p_metodo IS NULL OR p_metodo NOT IN('certificado','dnie') OR p_solicitado IS NULL OR NOT isfinite(p_solicitado) OR p_solicitado>clock_timestamp() OR p_version IS NULL OR p_version<>1 THEN RETURN NULL; END IF;
 SELECT * INTO v FROM vec_identidad_sesiones_v1.vinculo_sesion_admin_v1 WHERE referencia=p_ref AND version=p_version AND huella_sha256=p_sha
 AND autenticacion_ref=p_aut AND sesion_ref=p_ses AND cuenta_ref=p_cuenta AND documento->>'perfil_activo_ref'=p_perfil;
 IF NOT FOUND THEN RETURN NULL; END IF;
 SELECT * INTO STRICT i FROM vec_identidad_sesiones_v1.resolver_identidad_admin_perfiles_propietaria_v1(v.observacion->>'entorno',v.observacion->>'host',v.observacion->>'audiencia',v.observacion->>'certificado',v.observacion->>'ca',(v.observacion->>'autenticada')::timestamptz,(v.observacion->>'revocada')::timestamptz);
 SELECT * INTO STRICT s FROM vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(p_aut,p_ses);
 IF i.persona_ref IS DISTINCT FROM v.documento->>'persona_ref' OR i.cuenta_ref IS DISTINCT FROM p_cuenta OR i.cuenta_ordinaria_ref IS DISTINCT FROM v.documento->>'cuenta_ordinaria_ref'
 OR i.vinculo_ref IS DISTINCT FROM v.documento->>'vinculo_certificado_ref' OR i.vinculo_version IS DISTINCT FROM(v.documento->>'vinculo_certificado_version')::numeric
 OR i.politica_ref IS DISTINCT FROM v.documento->>'politica_ref' OR i.politica_huella_sha256 IS DISTINCT FROM v.documento->>'politica_sha256'
 OR s.metodo_observado IS DISTINCT FROM p_metodo OR s.cuenta_ref IS DISTINCT FROM p_cuenta OR s.cuenta_ordinaria_ref IS DISTINCT FROM i.cuenta_ordinaria_ref OR s.cuenta_privilegiada IS DISTINCT FROM true
 OR s.superficie IS DISTINCT FROM 'administracion_privilegiada' OR (s.metodo_observado IS NULL OR s.metodo_observado NOT IN('certificado','dnie')) OR s.garantia_observada IS DISTINCT FROM 'alto'
 OR s.control_sesion_ref IS DISTINCT FROM v.documento->>'control_sesion_ref' OR s.control_sesion_revision::numeric IS DISTINCT FROM(v.documento->>'control_sesion_revision')::numeric
 OR s.control_sesion_huella_sha256 IS DISTINCT FROM v.documento->>'control_sesion_sha256'
 OR s.politica_garantia_ref IS DISTINCT FROM i.politica_ref OR s.politica_garantia_huella_sha256 IS DISTINCT FROM i.politica_huella_sha256
 OR s.autenticacion_verificada_en IS DISTINCT FROM(v.observacion->>'autenticada')::timestamptz THEN RETURN NULL; END IF;
 IF current_setting('transaction_isolation')='serializable' THEN
  lista:=vec_contexto_actor_v1.listar_admin_preperfil_propietaria_v1(p_cuenta,i.persona_ref,v.observacion->>'audiencia');
  IF lista->>'perfil_activo_ref' IS DISTINCT FROM p_perfil OR(lista->>'revision')::numeric IS DISTINCT FROM(v.documento->>'seleccion_revision')::numeric THEN RETURN NULL; END IF;
 END IF;
 ahora:=clock_timestamp();
 IF p_solicitado<(v.documento->>'vinculada_en')::timestamptz
 OR ahora>=LEAST(i.vigente_hasta,s.sesion_valida_hasta,(v.documento->>'vigente_hasta')::timestamptz,(v.observacion->>'crl_hasta')::timestamptz,(v.observacion->>'certificado_hasta')::timestamptz) THEN RETURN NULL; END IF;
 RETURN jsonb_build_object('referencia',v.referencia,'version',v.version,'huella_sha256',v.huella_sha256,
  'autenticacion_ref',v.autenticacion_ref,'sesion_ref',v.sesion_ref,'persona_ref',i.persona_ref,'cuenta_ref',i.cuenta_ref,
  'perfil_ref',p_perfil,'seleccion_revision',(v.documento->>'seleccion_revision')::numeric,'fuente_ref',v.documento->>'fuente_ref',
  'fuente_sha256',v.documento->>'fuente_sha256','vigente_hasta',(v.documento->>'vigente_hasta')::timestamptz);
EXCEPTION WHEN no_data_found OR too_many_rows OR data_exception THEN RETURN NULL;
END $f$;
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.cotejar_vinculo_sesion_admin_v1(text,numeric,text,text,text,text,text,timestamptz,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_identidad_sesiones_v1.cotejar_vinculo_sesion_admin_v1(text,numeric,text,text,text,text,text,timestamptz,text) TO vec_contexto_actor_v1_propietario;
GRANT USAGE ON SCHEMA vec_identidad_sesiones_v1 TO vec_identidad_sesiones_v1_admin_perfiles_runtime;
GRANT EXECUTE ON FUNCTION vec_identidad_sesiones_v1.acreditar_runtime_admin_perfiles_v1(),
 vec_identidad_sesiones_v1.rechazar_fuente_cuenta_admin_v1(text,text,text),
 vec_identidad_sesiones_v1.resolver_cuenta_admin_perfiles_v1(text,text,text,text,text,timestamptz,timestamptz,timestamptz,timestamptz,text,text),
 vec_identidad_sesiones_v1.vincular_sesion_admin_perfiles_v1(text,text,text,text,text,timestamptz,timestamptz,timestamptz,timestamptz,text,text,jsonb,text,text,text)
 TO vec_identidad_sesiones_v1_admin_perfiles_runtime;
RESET ROLE;
DO $connect$ BEGIN EXECUTE format('GRANT CONNECT ON DATABASE %I TO vec_identidad_sesiones_v1_admin_perfiles_runtime',current_database()); END $connect$;
COMMIT;
