\set ON_ERROR_STOP on
-- AD189: corriente técnica de la frontera ADMIN sin atribuir Persona/perfil.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000189',0));
DO $pre$
DECLARE actual text;
BEGIN
 SELECT encode(sha256(convert_to(pg_get_constraintdef(c.oid,false),'UTF8')),'hex') INTO actual
 FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass AND c.conname='auditoria_tipo_disjunto_v4' AND c.convalidated;
 IF actual IS DISTINCT FROM '92f884b9e70c65f720a1448c00269949f035d14d6dd466ebe0678b2f27cfc2b4' THEN RAISE EXCEPTION 'AD189: PARO clave=CHECK_sha actual=% esperado=92f884b9e70c65f720a1448c00269949f035d14d6dd466ebe0678b2f27cfc2b4',coalesce(actual,'ausente') USING ERRCODE='55000';END IF;
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 OR to_regrole('vec_admin_frontera_tecnica_ejecutor') IS NOT NULL THEN RAISE EXCEPTION 'AD189: PARO clave=preimagen actual=incompatible esperado=POST188_PG18_super_sin189' USING ERRCODE='55000';END IF;
END $pre$;
CREATE ROLE vec_admin_frontera_tecnica_ejecutor NOLOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
DO $connect$ BEGIN EXECUTE format('GRANT CONNECT ON DATABASE %I TO vec_admin_frontera_tecnica_ejecutor',current_database());END $connect$;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
LOCK TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 IN ACCESS EXCLUSIVE MODE;
DO $familia$
DECLARE anterior text;nueva text;
BEGIN
 SELECT pg_get_constraintdef(c.oid,false) INTO STRICT anterior FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass AND c.conname='auditoria_tipo_disjunto_v4';
 nueva:='CHECK (('||substr(anterior,8,length(anterior)-8)||') OR ('||$tipo$
 tipo_registro IS NOT DISTINCT FROM 'frontera_admin_tecnica'
 AND decision_ref IS NULL AND efecto_ref IS NULL AND huella_efecto_sha256 IS NULL AND version_consumo IS NULL
 AND intento_ref IS NULL AND intento_material_sha256 IS NULL AND actor_ref IS NULL AND perfil_activo_ref IS NULL
 AND registro_contexto_ref IS NULL AND contexto_sha256 IS NULL AND procedencia_sha256 IS NULL
 AND autenticacion_ref IS NULL AND sesion_ref IS NULL AND autenticacion_sha256 IS NULL AND vinculo_sha256 IS NULL
 AND fuente_ref IS NULL AND fuente_sha256 IS NULL AND aprobacion_ref IS NULL AND plan_sha256 IS NULL
 AND fuentes_plan_ref IS NULL AND fuentes_preimagen_sha256 IS NULL AND fuentes_configuracion_sha256 IS NULL AND fuentes_alcance IS NULL AND fuentes_solicitud_sha256 IS NULL
 AND unidad_plan_ref IS NULL AND unidad_preimagen_sha256 IS NULL AND unidad_configuracion_sha256 IS NULL AND unidad_alcance IS NULL
 AND unidad_recibo_ref IS NULL AND unidad_recibo_sha256 IS NULL AND unidad_solicitud_sha256 IS NULL AND bootstrap_solicitud_sha256 IS NULL
 AND mantenimiento_detalle IS NULL AND mantenimiento_solicitud_sha256 IS NULL AND periodica_detalle IS NULL AND preservacion_detalle IS NULL
 AND gobierno_usuarios_detalle IS NULL AND gobierno_usuarios_solicitud_sha256 IS NULL
 AND evento_ref IS NOT NULL AND evento_material_sha256 IS NOT NULL AND operador_login IS NOT NULL
 AND accion IS NOT DISTINCT FROM 'controlar_frontera_admin_v1' AND modulo_id IS NOT DISTINCT FROM 'administracion'
 AND recurso_ref IS NOT NULL AND correlacion_ref IS NOT NULL AND resultado IS NOT NULL AND resultado IN ('denegado','error')
 AND motivo_ref IS NOT NULL AND proceso IS NOT NULL AND canal IS NOT DISTINCT FROM 'administracion_privilegiada'
 AND finalidad_ref IS NOT DISTINCT FROM 'control_frontera_admin'
 $tipo$||'))';
 ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 DROP CONSTRAINT auditoria_tipo_disjunto_v4;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 ADD CONSTRAINT auditoria_tipo_disjunto_v4 '||nueva;
END $familia$;
CREATE TABLE vec_autorizacion_atestada_v3.config_frontera_admin_tecnica_v1(
 login_nombre name PRIMARY KEY,
 proceso text NOT NULL CHECK(proceso ~ '^[a-z][a-z0-9._-]{1,79}$'),
 canal text NOT NULL CHECK(canal='administracion_privilegiada'),
 vigente_desde timestamptz NOT NULL,vigente_hasta timestamptz NOT NULL,
 CHECK(isfinite(vigente_desde) AND isfinite(vigente_hasta) AND vigente_hasta>vigente_desde)
);
ALTER TABLE vec_autorizacion_atestada_v3.config_frontera_admin_tecnica_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_autorizacion_atestada_v3.config_frontera_admin_tecnica_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario ON vec_autorizacion_atestada_v3.config_frontera_admin_tecnica_v1 TO vec_autorizacion_atestada_v3_propietario USING(current_user='vec_autorizacion_atestada_v3_propietario') WITH CHECK(current_user='vec_autorizacion_atestada_v3_propietario');
-- No se siembra LOGIN ni configuración favorable en la migración.
CREATE FUNCTION vec_autorizacion_atestada_v3.rechazar_mutacion_config_frontera_admin_v1()
RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog AS $f$BEGIN RAISE EXCEPTION 'AD189: configuracion_inmutable' USING ERRCODE='55000';END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.rechazar_mutacion_config_frontera_admin_v1() FROM PUBLIC;
CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_autorizacion_atestada_v3.config_frontera_admin_tecnica_v1 FOR EACH ROW EXECUTE FUNCTION vec_autorizacion_atestada_v3.rechazar_mutacion_config_frontera_admin_v1();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_autorizacion_atestada_v3.config_frontera_admin_tecnica_v1 FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion_atestada_v3.rechazar_mutacion_config_frontera_admin_v1();
REVOKE ALL ON TABLE vec_autorizacion_atestada_v3.config_frontera_admin_tecnica_v1 FROM PUBLIC;
REVOKE ALL ON TYPE vec_autorizacion_atestada_v3.config_frontera_admin_tecnica_v1 FROM PUBLIC;
CREATE FUNCTION vec_autorizacion_atestada_v3.codigos_frontera_admin_tecnica_v1()
RETURNS jsonb LANGUAGE sql IMMUTABLE SET search_path=pg_catalog AS $f$
 SELECT $datos$[{"codigo_ref":"autenticacion_requerida","resultado":"denegado"},{"codigo_ref":"acceso_denegado","resultado":"denegado"},{"codigo_ref":"metodo_no_permitido","resultado":"denegado"},{"codigo_ref":"solicitud_invalida","resultado":"denegado"},{"codigo_ref":"conflicto_estado","resultado":"denegado"},{"codigo_ref":"recurso_no_encontrado","resultado":"denegado"},{"codigo_ref":"servicio_no_disponible","resultado":"error"},{"codigo_ref":"respuesta_incompatible","resultado":"error"}]$datos$::jsonb
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.codigos_frontera_admin_tecnica_v1() FROM PUBLIC;
CREATE FUNCTION vec_autorizacion_atestada_v3.exigir_frontera_admin_tecnica_v1()
RETURNS vec_autorizacion_atestada_v3.config_frontera_admin_tecnica_v1 LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $f$
DECLARE l record;g record;c vec_autorizacion_atestada_v3.config_frontera_admin_tecnica_v1;ns oid;db oid;f1 oid;f2 oid;
BEGIN
 SELECT * INTO l FROM pg_roles WHERE rolname=session_user;SELECT * INTO g FROM pg_roles WHERE rolname='vec_admin_frontera_tecnica_ejecutor';
 ns:=to_regnamespace('vec_autorizacion_atestada_v3');SELECT oid INTO db FROM pg_database WHERE datname=current_database();f1:=to_regprocedure('vec_autorizacion_atestada_v3.acreditar_frontera_admin_tecnica_v1()');f2:=to_regprocedure('vec_autorizacion_atestada_v3.registrar_frontera_admin_tecnica_v1(jsonb)');
 IF current_setting('role')<>'none' OR l.oid IS NULL OR g.oid IS NULL OR NOT l.rolcanlogin OR NOT l.rolinherit OR l.rolsuper OR l.rolcreaterole OR l.rolcreatedb OR l.rolreplication OR l.rolbypassrls OR l.rolconfig IS NOT NULL
 OR g.rolcanlogin OR g.rolinherit OR g.rolsuper OR g.rolcreaterole OR g.rolcreatedb OR g.rolreplication OR g.rolbypassrls OR g.rolconfig IS NOT NULL
 OR(SELECT count(*) FROM pg_auth_members WHERE member=l.oid)<>1 OR NOT EXISTS(SELECT 1 FROM pg_auth_members WHERE member=l.oid AND roleid=g.oid AND inherit_option AND NOT set_option AND NOT admin_option)
 OR EXISTS(SELECT 1 FROM pg_auth_members WHERE member=g.oid) OR EXISTS(SELECT 1 FROM pg_db_role_setting WHERE setrole IN(l.oid,g.oid))
 OR EXISTS(SELECT 1 FROM pg_shdepend WHERE refclassid='pg_catalog.pg_authid'::regclass AND refobjid=l.oid)
 OR EXISTS(SELECT 1 FROM pg_shdepend WHERE refclassid='pg_catalog.pg_authid'::regclass AND refobjid=g.oid AND NOT((dbid=db AND classid='pg_catalog.pg_namespace'::regclass AND objid=ns AND deptype='a') OR(dbid=db AND classid='pg_catalog.pg_proc'::regclass AND objid IN(f1,f2) AND deptype='a') OR(dbid=0 AND classid='pg_catalog.pg_database'::regclass AND objid=db AND deptype='a')))
 OR NOT COALESCE((SELECT count(*)=1 AND bool_and(a.privilege_type='CONNECT' AND NOT a.is_grantable) FROM pg_database d CROSS JOIN LATERAL aclexplode(coalesce(d.datacl,acldefault('d',d.datdba))) a WHERE d.oid=db AND a.grantee=g.oid),false)
 OR NOT COALESCE((SELECT count(*)=1 AND bool_and(a.privilege_type='USAGE' AND NOT a.is_grantable) FROM pg_namespace n CROSS JOIN LATERAL aclexplode(coalesce(n.nspacl,acldefault('n',n.nspowner))) a WHERE n.oid=ns AND a.grantee=g.oid),false)
 OR(SELECT count(*) FROM pg_proc p WHERE p.oid IN(f1,f2) AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef AND EXISTS(SELECT 1 FROM aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE a.grantee=g.oid AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)
 AND NOT EXISTS(SELECT 1 FROM aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE a.grantee NOT IN(p.proowner,g.oid) OR a.privilege_type<>'EXECUTE' OR a.is_grantable))<>2
 OR has_schema_privilege(l.oid,'vec_autorizacion_atestada_v3','CREATE') OR has_database_privilege(l.oid,current_database(),'CREATE,TEMP') THEN RAISE EXCEPTION 'AD189: operador_no_acreditado' USING ERRCODE='42501';END IF;
 SELECT * INTO c FROM vec_autorizacion_atestada_v3.config_frontera_admin_tecnica_v1 WHERE login_nombre=session_user FOR SHARE;
 IF NOT FOUND OR clock_timestamp()<c.vigente_desde OR clock_timestamp()>=c.vigente_hasta THEN RAISE EXCEPTION 'AD189: configuracion_no_vigente' USING ERRCODE='42501';END IF;
 RETURN c;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.exigir_frontera_admin_tecnica_v1() FROM PUBLIC;
CREATE FUNCTION vec_autorizacion_atestada_v3.acreditar_frontera_admin_tecnica_v1()
RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $f$
DECLARE c vec_autorizacion_atestada_v3.config_frontera_admin_tecnica_v1;
BEGIN
 c:=vec_autorizacion_atestada_v3.exigir_frontera_admin_tecnica_v1();
 RETURN jsonb_build_object('operador_login',session_user::text,'proceso',c.proceso,'canal',c.canal,'codigos',vec_autorizacion_atestada_v3.codigos_frontera_admin_tecnica_v1());
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.acreditar_frontera_admin_tecnica_v1() FROM PUBLIC;
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_frontera_admin_tecnica_v1(e jsonb)
RETURNS TABLE(auditoria_ref text,secuencia numeric,huella_sha256 text,correlacion_ref text,registrada_en timestamptz)
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog SET row_security=on SET timezone='UTC' AS $f$
DECLARE c vec_autorizacion_atestada_v3.config_frontera_admin_tecnica_v1;campo text;orden text[]:=ARRAY['tipo_registro','evento_ref','operador_login','accion','recurso_ref','resultado','codigo_ref','proceso','canal','finalidad_ref','correlacion_ref'];material bytea;material_sha text;existente record;s numeric;anterior text;instante timestamptz;ref text;h text;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off' THEN RAISE EXCEPTION 'AD189: transaccion_incompatible' USING ERRCODE='25000';END IF;
 c:=vec_autorizacion_atestada_v3.exigir_frontera_admin_tecnica_v1();
 IF jsonb_typeof(e) IS DISTINCT FROM 'object' OR(SELECT count(*) FROM jsonb_object_keys(e))<>11 OR NOT e ?& orden THEN RAISE EXCEPTION 'AD189: evento_invalido' USING ERRCODE='22023';END IF;
 FOREACH campo IN ARRAY orden LOOP IF jsonb_typeof(e->campo) IS DISTINCT FROM 'string' THEN RAISE EXCEPTION 'AD189: campos_invalidos' USING ERRCODE='22023';END IF;END LOOP;
 IF e->>'tipo_registro' IS DISTINCT FROM 'frontera_admin_tecnica' OR e->>'evento_ref' !~ '^evento_[0-9a-f]{32}$'
 OR e->>'operador_login' IS DISTINCT FROM session_user::text OR e->>'accion' IS DISTINCT FROM 'controlar_frontera_admin_v1'
 OR e->>'correlacion_ref' !~ '^correlacion_[0-9a-f]{32}$' OR e->>'recurso_ref' IS DISTINCT FROM 'solicitud_admin:'||substr(encode(sha256(convert_to(E'vec.admin.frontera.solicitud.v1\n'||(e->>'correlacion_ref'),'UTF8')),'hex'),1,32)
 OR e->>'proceso' IS DISTINCT FROM c.proceso OR e->>'canal' IS DISTINCT FROM c.canal OR e->>'finalidad_ref' IS DISTINCT FROM 'control_frontera_admin'
 OR NOT EXISTS(SELECT 1 FROM jsonb_array_elements(vec_autorizacion_atestada_v3.codigos_frontera_admin_tecnica_v1()) x WHERE x->>'codigo_ref'=e->>'codigo_ref' AND x->>'resultado'=e->>'resultado') THEN RAISE EXCEPTION 'AD189: semantica_incompatible' USING ERRCODE='22023';END IF;
 material:=vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.frontera-admin-tecnica.v1');
 FOREACH campo IN ARRAY orden LOOP material:=material||vec_autorizacion_atestada_v3.encuadrar_mac(e->>campo);END LOOP;
 material_sha:=encode(sha256(material),'hex');
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:evento-admin:'||(e->>'evento_ref'),0));
 SELECT a.* INTO existente FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 a WHERE a.evento_ref=e->>'evento_ref';
 IF FOUND THEN
  IF existente.tipo_registro IS DISTINCT FROM 'frontera_admin_tecnica' OR existente.evento_material_sha256 IS DISTINCT FROM material_sha THEN RAISE EXCEPTION 'AD189: replay_distinto' USING ERRCODE='23505';END IF;
  PERFORM vec_autorizacion_atestada_v3.exigir_frontera_admin_tecnica_v1();
  RETURN QUERY SELECT existente.auditoria_ref,existente.secuencia,existente.huella_sha256,existente.correlacion_ref,existente.registrada_en;RETURN;
 END IF;
 SELECT v.secuencia,v.cabeza_sha256 INTO STRICT s,anterior FROM vec_autorizacion_atestada_v3.control_cadena_auditoria v WHERE v.control_id FOR UPDATE;
 IF s>=9007199254740991 THEN RAISE EXCEPTION 'AD189: secuencia_fuera_limite' USING ERRCODE='22003';END IF;
 PERFORM vec_autorizacion_atestada_v3.exigir_frontera_admin_tecnica_v1();
 s:=s+1;instante:=clock_timestamp();ref:='aud_v3_fat_'||substr(e->>'evento_ref',8,32);
 h:=encode(sha256(vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.eslabon.frontera-admin-tecnica.v1')||vec_autorizacion_atestada_v3.encuadrar_mac(s::text)||vec_autorizacion_atestada_v3.encuadrar_mac(anterior)||vec_autorizacion_atestada_v3.encuadrar_mac(ref)||vec_autorizacion_atestada_v3.encuadrar_mac(material_sha)||vec_autorizacion_atestada_v3.encuadrar_mac(to_char(instante AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))),'hex');
 INSERT INTO vec_autorizacion_atestada_v3.auditoria_consumo_v3(auditoria_ref,secuencia,anterior_sha256,huella_sha256,registrada_en,tipo_registro,evento_ref,evento_material_sha256,operador_login,accion,modulo_id,recurso_ref,resultado,motivo_ref,proceso,canal,finalidad_ref,correlacion_ref)
 VALUES(ref,s,anterior,h,instante,'frontera_admin_tecnica',e->>'evento_ref',material_sha,session_user,'controlar_frontera_admin_v1','administracion',e->>'recurso_ref',e->>'resultado',e->>'codigo_ref',c.proceso,c.canal,'control_frontera_admin',e->>'correlacion_ref');
 UPDATE vec_autorizacion_atestada_v3.control_cadena_auditoria SET secuencia=s,cabeza_sha256=h,actualizada_en=instante WHERE control_id;
 RETURN QUERY SELECT ref,s,h,e->>'correlacion_ref',instante;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_frontera_admin_tecnica_v1(jsonb) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_admin_frontera_tecnica_ejecutor;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.acreditar_frontera_admin_tecnica_v1(),vec_autorizacion_atestada_v3.registrar_frontera_admin_tecnica_v1(jsonb) TO vec_admin_frontera_tecnica_ejecutor;
COMMIT;
