\set ON_ERROR_STOP on
-- BORRADOR AUT24. No instalar. Dependencias descritas en CONTRATO.md.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec:admin:continuidad:v1',0));
DO $preimagen$
BEGIN
 IF to_regclass('vec_autorizacion.control_continuidad_admin') IS NULL
 OR to_regprocedure('vec_contexto_actor_v1.crear_perfil_vinculo_admin_v1(text,text,numeric,numeric,text,text,text,numeric,text,timestamptz)') IS NULL
 THEN
  RAISE EXCEPTION 'AUT24: dependencia esperado=presente observado=ausente' USING ERRCODE='55000';
 END IF;
 IF NOT EXISTS(SELECT 1 FROM vec_autorizacion.control_continuidad_admin WHERE control_id AND bootstrap_minimo_personas=2 AND minimo_personas=1) THEN RAISE EXCEPTION 'AUT24: continuidad esperado=bootstrap2_minimo1 observado=divergente' USING ERRCODE='55000'; END IF;
END $preimagen$;
-- No se crean LOGIN ni personas. El rol técnico no permite SET ROLE owner.
DO $rol$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='vec_admin_perfiles_ejecutor') THEN
  CREATE ROLE vec_admin_perfiles_ejecutor NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOBYPASSRLS;
 END IF;
 IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='vec_admin_perfiles_ejecutor'
  AND (rolcanlogin OR rolsuper OR rolcreatedb OR rolcreaterole OR rolbypassrls))
 OR EXISTS(SELECT 1 FROM pg_auth_members WHERE member='vec_admin_perfiles_ejecutor'::regrole) THEN
  RAISE EXCEPTION 'AUT24: rol_runtime esperado=aislado observado=privilegiado' USING ERRCODE='55000';
 END IF;
END $rol$;
SET LOCAL ROLE vec_autorizacion_propietario;
CREATE TABLE vec_autorizacion.rol_administrable_exacto_v1(
 version_rol_ref text PRIMARY KEY REFERENCES vec_autorizacion.version_rol(version_rol_ref),
 clase text NOT NULL CHECK(clase IN ('ordinario','administrador','intervencion')),
 huella_sha256 text NOT NULL CHECK(huella_sha256 ~ '^[0-9a-f]{64}$'),
 vigente_desde timestamptz NOT NULL,
 vigente_hasta timestamptz NOT NULL,
 unidad_requerida boolean NOT NULL,
 audiencia_administrativa text NOT NULL CHECK(vec_autorizacion.texto_positivo_valido(audiencia_administrativa,256) IS TRUE),
 ambitos_fijos jsonb NOT NULL CHECK(jsonb_typeof(ambitos_fijos)='array'),
 duracion_propuesta interval NOT NULL CHECK(duracion_propuesta>interval '0 seconds'),
 CHECK(isfinite(vigente_desde) AND isfinite(vigente_hasta) AND vigente_hasta>vigente_desde)
);
-- Este catálogo registra referencias publicadas; nunca publica concesiones.
-- Dietas entra por su rol fijo AUT27 y una huella verificada, no por prefijos.
-- No se insertan valores de ejemplo en la autoridad de permisos.
CREATE TABLE vec_autorizacion.registro_acto_admin_v1(
 operacion_ref text PRIMARY KEY,
 material bytea NOT NULL CHECK(octet_length(material) BETWEEN 1 AND 65536),
 huella_sha256 text NOT NULL CHECK(huella_sha256=encode(sha256(material),'hex')),
 actor_persona_ref text NOT NULL,
 actor_perfil_ref text NOT NULL,
 decision_ref text NOT NULL,
 auditoria_ref text NOT NULL,
 resultado jsonb NOT NULL CHECK(jsonb_typeof(resultado)='object'),
 registrada_en timestamptz NOT NULL CHECK(isfinite(registrada_en))
);
CREATE TABLE vec_autorizacion.outbox_acto_admin_v1(
 operacion_ref text PRIMARY KEY REFERENCES vec_autorizacion.registro_acto_admin_v1(operacion_ref),
 evento text NOT NULL CHECK(evento IN ('perfil_otorgado','perfil_revocado','propuesta_registrada','propuesta_aprobada','propuesta_rechazada')),
 auditoria_ref text NOT NULL,
 creada_en timestamptz NOT NULL CHECK(isfinite(creada_en))
);
DO $historia$
DECLARE t text;
BEGIN
 FOREACH t IN ARRAY ARRAY['rol_administrable_exacto_v1','registro_acto_admin_v1','outbox_acto_admin_v1'] LOOP
  EXECUTE format('ALTER TABLE vec_autorizacion.%I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE format('ALTER TABLE vec_autorizacion.%I FORCE ROW LEVEL SECURITY',t);
  EXECUTE format('CREATE POLICY propietario_exacto ON vec_autorizacion.%I FOR ALL TO vec_autorizacion_propietario USING(current_user=''vec_autorizacion_propietario'') WITH CHECK(current_user=''vec_autorizacion_propietario'')',t);
  EXECUTE format('CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_autorizacion.%I FOR EACH ROW EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable()',t);
  EXECUTE format('CREATE TRIGGER historia_no_truncable BEFORE TRUNCATE ON vec_autorizacion.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable()',t);
  EXECUTE format('REVOKE ALL ON TABLE vec_autorizacion.%I FROM PUBLIC,vec_autorizacion_fuente,vec_admin_perfiles_ejecutor',t);
  EXECUTE format('REVOKE ALL ON TYPE vec_autorizacion.%I FROM PUBLIC,vec_autorizacion_fuente,vec_admin_perfiles_ejecutor',t);
 END LOOP;
END $historia$;
CREATE FUNCTION vec_autorizacion.resolver_rol_administrable_v1(p_version_ref text)
RETURNS jsonb LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE r record; instante timestamptz:=statement_timestamp();
BEGIN
 SELECT a.* INTO r FROM vec_autorizacion.rol_administrable_exacto_v1 a
 JOIN vec_autorizacion.version_rol v USING(version_rol_ref)
 JOIN vec_autorizacion.control_vigencia_version_rol_actual ca USING(version_rol_ref)
 JOIN vec_autorizacion.control_vigencia_version_rol c ON c.version_rol_ref=ca.version_rol_ref AND c.revision=ca.revision
 WHERE a.version_rol_ref=p_version_ref AND a.huella_sha256=v.huella_sha256
 AND v.documento->>'estado'='publicada' AND c.estado='habilitada'
 AND instante>=a.vigente_desde AND instante<a.vigente_hasta;
 IF NOT FOUND THEN RAISE EXCEPTION 'AUT24: rol no administrable' USING ERRCODE='42501'; END IF;
 RETURN jsonb_build_object('version_ref',r.version_rol_ref,'clase',r.clase,'huella_sha256',r.huella_sha256,
  'vigente_desde',r.vigente_desde,'vigente_hasta',r.vigente_hasta,'unidad_requerida',r.unidad_requerida);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.resolver_rol_administrable_v1(text) FROM PUBLIC;

RESET ROLE;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
CREATE FUNCTION vec_contexto_actor_v1.preimagen_admin_interna_v1(p_cuenta text,p_persona text,p_perfil text,p_vinculo text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE c record;pe record;pf record;v record; dc jsonb;dp jsonb;df jsonb:=null;dv jsonb:=null;ahora timestamptz:=clock_timestamp();
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off' THEN RAISE EXCEPTION 'AUT24: preimagen requiere SERIALIZABLE' USING ERRCODE='25000'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec:admin:continuidad:v1',0));
 SELECT x.* INTO STRICT c FROM vec_contexto_actor_v1.proyeccion_cuenta_actual a JOIN vec_contexto_actor_v1.proyeccion_cuenta_versiones x USING(cuenta_ref,version) WHERE a.cuenta_ref=p_cuenta FOR UPDATE OF a;
 SELECT x.* INTO STRICT pe FROM vec_contexto_actor_v1.persona_actual a JOIN vec_contexto_actor_v1.persona_versiones x USING(persona_ref,version) WHERE a.persona_ref=p_persona FOR UPDATE OF a;
 SELECT x.* INTO pf FROM vec_contexto_actor_v1.perfil_actual a JOIN vec_contexto_actor_v1.perfil_versiones x USING(perfil_ref,version) WHERE a.perfil_ref=p_perfil FOR UPDATE OF a;
 IF FOUND THEN df:=jsonb_build_object('perfil_ref',pf.perfil_ref,'version',pf.version,'persona_ref',pf.persona_ref,'estado',pf.estado,'procedencia_ref',pf.procedencia_ref,'procedencia_version',pf.procedencia_version,'procedencia_huella_sha256',pf.procedencia_huella_sha256,'procedencia_autoridad',pf.procedencia_autoridad,'vigente_desde',pf.vigente_desde,'vigente_hasta',pf.vigente_hasta); END IF;
 SELECT x.* INTO v FROM vec_contexto_actor_v1.vinculo_contexto_actual a JOIN vec_contexto_actor_v1.vinculo_contexto_versiones x USING(vinculo_ref,version) WHERE a.vinculo_ref=p_vinculo FOR UPDATE OF a;
 IF FOUND THEN dv:=jsonb_build_object('vinculo_ref',v.vinculo_ref,'version',v.version,'cuenta_ref',v.cuenta_ref,'persona_ref',v.persona_ref,'perfil_ref',v.perfil_ref,'estado',v.estado,'procedencia_ref',v.procedencia_ref,'procedencia_version',v.procedencia_version,'procedencia_huella_sha256',v.procedencia_huella_sha256,'procedencia_autoridad',v.procedencia_autoridad,'vigente_desde',v.vigente_desde,'vigente_hasta',v.vigente_hasta); END IF;
 IF c.estado<>'activo' OR pe.estado<>'activo' OR c.procedencia_autoridad<>'autoridad_maestra_acreditada' OR pe.procedencia_autoridad<>'autoridad_maestra_acreditada' OR ahora<GREATEST(c.vigente_desde,pe.vigente_desde) OR ahora>=LEAST(c.vigente_hasta,pe.vigente_hasta) THEN RAISE EXCEPTION 'AUT24: objetivo no acreditado' USING ERRCODE='42501'; END IF;
 dc:=jsonb_build_object('cuenta_ref',c.cuenta_ref,'version',c.version,'estado',c.estado,'procedencia_ref',c.procedencia_ref,'procedencia_version',c.procedencia_version,'procedencia_huella_sha256',c.procedencia_huella_sha256,'procedencia_autoridad',c.procedencia_autoridad,'vigente_desde',c.vigente_desde,'vigente_hasta',c.vigente_hasta);
 dp:=jsonb_build_object('persona_ref',pe.persona_ref,'version',pe.version,'estado',pe.estado,'procedencia_ref',pe.procedencia_ref,'procedencia_version',pe.procedencia_version,'procedencia_huella_sha256',pe.procedencia_huella_sha256,'procedencia_autoridad',pe.procedencia_autoridad,'vigente_desde',pe.vigente_desde,'vigente_hasta',pe.vigente_hasta);
 RETURN jsonb_build_object('cuenta',dc,'persona',dp,'perfil',df,'vinculo',dv);
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.preimagen_admin_interna_v1(text,text,text,text) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_contexto_actor_v1 TO vec_autorizacion_propietario;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.preimagen_admin_interna_v1(text,text,text,text),vec_contexto_actor_v1.crear_perfil_vinculo_admin_v1(text,text,numeric,numeric,text,text,text,numeric,text,timestamptz),vec_contexto_actor_v1.revocar_perfil_vinculo_admin_v1(text,text,text,text,numeric,numeric,numeric,numeric,text,numeric,text),vec_contexto_actor_v1.bloquear_contexto_admin_v1(text,text,text,text,numeric,numeric,numeric,numeric) TO vec_autorizacion_propietario;
RESET ROLE;
SET LOCAL ROLE vec_identidad_sesiones_v1_propietario;
-- Población efectiva para continuidad. No sustituye la revalidación de sesión
-- de quien propone/aprueba: AD150 la exige por la fachada nominal IS12.
CREATE FUNCTION vec_identidad_sesiones_v1.estado_admin_interno_v1(p_persona text,p_cuenta text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE b record;pol record;c record;
BEGIN
 SELECT * INTO STRICT pol FROM vec_identidad_sesiones_v1.politica_certificado_admin_v1 WHERE singleton FOR SHARE;
 SELECT v.vinculo_ref,v.version,v.estado,v.vigente_desde,v.vigente_hasta INTO STRICT b
 FROM vec_identidad_sesiones_v1.vinculo_certificado_admin_actual_v1 a JOIN vec_identidad_sesiones_v1.vinculo_certificado_admin_v1 v USING(vinculo_ref,version)
 WHERE v.persona_ref=p_persona AND v.cuenta_privilegiada_ref=p_cuenta AND v.politica_ref=pol.politica_ref AND v.ca_sha256=pol.ca_sha256 FOR SHARE OF a;
 SELECT pc.cuenta_privilegiada,pc.cuenta_ordinaria_ref,s.estado,s.revision,so.estado AS estado_ordinaria,so.revision AS revision_ordinaria INTO STRICT c
 FROM vec_identidad_sesiones_v1.cuenta pc
 JOIN vec_identidad_sesiones_v1.estado_cuenta_actual a USING(cuenta_ref)
 JOIN vec_identidad_sesiones_v1.estado_cuenta s USING(cuenta_ref,revision)
 JOIN vec_identidad_sesiones_v1.estado_cuenta_actual ao ON ao.cuenta_ref=pc.cuenta_ordinaria_ref
 JOIN vec_identidad_sesiones_v1.estado_cuenta so ON so.cuenta_ref=ao.cuenta_ref AND so.revision=ao.revision
 WHERE pc.cuenta_ref=p_cuenta FOR SHARE OF a,ao;
 IF NOT pol.activa OR clock_timestamp()>=pol.vigente_hasta OR b.estado<>'activo' OR clock_timestamp()<b.vigente_desde OR clock_timestamp()>=b.vigente_hasta OR NOT c.cuenta_privilegiada OR c.estado<>'activa' OR c.estado_ordinaria<>'activa' THEN RETURN null; END IF;
 RETURN jsonb_build_object('persona_ref',p_persona,'cuenta_ref',p_cuenta,'cuenta_revision',c.revision,'cuenta_ordinaria_ref',c.cuenta_ordinaria_ref,'cuenta_ordinaria_revision',c.revision_ordinaria,'certificado_vinculo_ref',b.vinculo_ref,'certificado_vinculo_version',b.version,'politica_ref',pol.politica_ref,'politica_huella',pol.huella_aprobacion_sha256,'vigente_desde',b.vigente_desde,'vigente_hasta',LEAST(b.vigente_hasta,pol.vigente_hasta));
EXCEPTION WHEN no_data_found OR too_many_rows THEN RETURN null;
END $f$;
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.estado_admin_interno_v1(text,text) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_identidad_sesiones_v1 TO vec_autorizacion_propietario;
GRANT EXECUTE ON FUNCTION vec_identidad_sesiones_v1.estado_admin_interno_v1(text,text) TO vec_autorizacion_propietario;
RESET ROLE;
SET LOCAL ROLE vec_autorizacion_propietario;
CREATE FUNCTION vec_autorizacion.administradores_efectivos_internos_v1()
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE a record;ca jsonb; identidad jsonb;resultado jsonb:='[]'::jsonb;
BEGIN
 FOR a IN SELECT x.* FROM vec_autorizacion.asignacion_perfil_actual ac JOIN vec_autorizacion.asignacion_perfil x USING(asignacion_ref)
 JOIN vec_autorizacion.rol_sensible_exacto s USING(version_rol_ref) JOIN vec_autorizacion.version_rol r USING(version_rol_ref)
 JOIN vec_autorizacion.control_vigencia_version_rol_actual rc USING(version_rol_ref)
 JOIN vec_autorizacion.control_vigencia_version_rol c ON c.version_rol_ref=rc.version_rol_ref AND c.revision=rc.revision
 WHERE s.clase='administrador' AND x.documento->>'estado'='activa' AND s.huella_sha256=r.huella_sha256 AND c.estado='habilitada'
 AND clock_timestamp()>=(x.documento->>'vigente_desde')::timestamptz AND clock_timestamp()<(x.documento->>'vigente_hasta')::timestamptz ORDER BY x.principal_id,x.perfil_activo_ref FOR UPDATE OF ac LOOP
  ca:=vec_contexto_actor_v1.preimagen_admin_interna_v1(a.documento->>'cuenta_ref',a.principal_id,a.perfil_activo_ref,a.documento->>'vinculo_ref');
  identidad:=vec_identidad_sesiones_v1.estado_admin_interno_v1(a.principal_id,a.documento->>'cuenta_ref');
  IF identidad IS NOT NULL AND ca#>>'{perfil,estado}'='activo' AND ca#>>'{vinculo,estado}'='activo'
  AND ca#>>'{perfil,persona_ref}'=a.principal_id AND ca#>>'{vinculo,persona_ref}'=a.principal_id
  AND clock_timestamp()>=(ca#>>'{perfil,vigente_desde}')::timestamptz AND clock_timestamp()<(ca#>>'{perfil,vigente_hasta}')::timestamptz
  AND clock_timestamp()>=(ca#>>'{vinculo,vigente_desde}')::timestamptz AND clock_timestamp()<(ca#>>'{vinculo,vigente_hasta}')::timestamptz THEN
   resultado:=resultado||jsonb_build_array(jsonb_build_object('persona_ref',a.principal_id,'perfil_ref',a.perfil_activo_ref,'asignacion_ref',a.asignacion_ref,'asignacion_version',a.version,'asignacion_huella',a.huella_sha256,'contexto',ca,'identidad',identidad));
  END IF;
 END LOOP;
 RETURN resultado;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.administradores_efectivos_internos_v1() FROM PUBLIC;
CREATE FUNCTION vec_autorizacion.preimagen_cambio_admin_interna_v1(p_material jsonb)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET timezone='UTC' AS $f$
DECLARE o jsonb:=p_material->'objetivo';c jsonb;r record;cv record;a record;continuidad record;actual_asignacion jsonb:=null;efectivos jsonb;
BEGIN
 PERFORM pg_advisory_xact_lock(hashtextextended('vec:admin:continuidad:v1',0));
 SELECT * INTO STRICT continuidad FROM vec_autorizacion.control_continuidad_admin WHERE control_id FOR UPDATE;
 c:=vec_contexto_actor_v1.preimagen_admin_interna_v1(o->>'cuenta_ref',o->>'persona_ref',o->>'perfil_ref',o->>'vinculo_ref');
 SELECT v.* INTO STRICT r FROM vec_autorizacion.version_rol v WHERE version_rol_ref=p_material->>'rol_version_ref';
 SELECT v.* INTO STRICT cv FROM vec_autorizacion.control_vigencia_version_rol_actual a JOIN vec_autorizacion.control_vigencia_version_rol v USING(version_rol_ref,revision) WHERE a.version_rol_ref=r.version_rol_ref FOR UPDATE OF a;
 SELECT v.* INTO a FROM vec_autorizacion.asignacion_perfil_actual ac JOIN vec_autorizacion.asignacion_perfil v USING(asignacion_ref) WHERE ac.perfil_activo_ref=o->>'perfil_ref' FOR UPDATE OF ac;
 IF FOUND THEN actual_asignacion:=jsonb_build_object('asignacion_ref',a.asignacion_ref,'version',a.version,'huella_sha256',a.huella_sha256,'estado',a.documento->>'estado','ambitos',a.documento->'ambitos','version_rol_ref',a.version_rol_ref,'principal_id',a.principal_id); END IF;
 efectivos:=vec_autorizacion.administradores_efectivos_internos_v1();
 RETURN jsonb_build_object('esquema','vec.admin.perfiles.preimagen.v1','operacion',p_material->>'operacion','rol_version_ref',r.version_rol_ref,'rol_huella_sha256',r.huella_sha256,'control_revision',cv.revision,'control_huella_sha256',cv.huella_sha256,'contexto',c,'asignacion',actual_asignacion,'unidad_ref',p_material->>'unidad_ref','objetivo',o-'huella_sha256','continuidad',jsonb_build_object('revision',continuidad.revision,'bootstrap_estado',continuidad.bootstrap_estado,'minimo_personas',continuidad.minimo_personas,'bootstrap_minimo_personas',continuidad.bootstrap_minimo_personas),'administradores_efectivos',efectivos);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.preimagen_cambio_admin_interna_v1(jsonb) FROM PUBLIC;

-- Valida el DTO de negocio, nunca utiliza su actor como prueba de identidad.
CREATE FUNCTION vec_autorizacion.validar_material_acto_admin_v1(p_material text,p_propuesta boolean)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE m jsonb; o jsonb; r jsonb; claves text[];
BEGIN
 IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 1 AND 65536 THEN
  RAISE EXCEPTION 'AUT24: material invalido' USING ERRCODE='22023'; END IF;
 m:=p_material::jsonb; o:=m->'objetivo';
 SELECT array_agg(key ORDER BY key) INTO claves FROM jsonb_each(m);
 IF m->>'esquema' IS DISTINCT FROM 'administracion_perfiles_acto_v1'
 OR NOT m ?& ARRAY['esquema','operacion_ref','operacion','rol_version_ref','rol_huella_sha256','actor_persona_ref','actor_perfil_ref','objetivo','motivo','correlacion_ref']
 OR jsonb_path_exists(m,'$.** ? (@ == null)')
 OR claves <@ ARRAY['esquema','operacion_ref','operacion','rol_version_ref','rol_huella_sha256','actor_persona_ref','actor_perfil_ref','objetivo','motivo','correlacion_ref','unidad_ref','referencia_acto'] IS NOT TRUE
 OR jsonb_typeof(o) IS DISTINCT FROM 'object'
 OR jsonb_typeof(m->'motivo') IS DISTINCT FROM 'object'
 OR NOT (m->'motivo') ?& ARRAY['catalogo_id','catalogo_version','catalogo_huella_sha256','entrada_clave']
 OR (SELECT count(*) FROM jsonb_object_keys(m->'motivo'))<>4
 OR vec_autorizacion.texto_positivo_valido(m#>>'{motivo,catalogo_id}',256) IS NOT TRUE
 OR vec_autorizacion.texto_positivo_valido(m#>>'{motivo,entrada_clave}',128) IS NOT TRUE
 OR (m#>>'{motivo,catalogo_version}')::numeric<1
 OR m#>>'{motivo,catalogo_huella_sha256}' !~ '^[0-9a-f]{64}$'
 OR m->>'operacion' NOT IN ('otorgar','revocar')
 OR (CASE WHEN p_propuesta THEN m->>'operacion_ref' !~ '^propuesta_admin:[0-9a-f]{32}$' ELSE m->>'operacion_ref' !~ '^acto_admin:[0-9a-f]{32}$' END)
 OR vec_autorizacion.texto_positivo_valido(m->>'actor_persona_ref',512) IS NOT TRUE
 OR vec_autorizacion.texto_positivo_valido(m->>'actor_perfil_ref',512) IS NOT TRUE
 OR vec_autorizacion.texto_positivo_valido(m->>'correlacion_ref',512) IS NOT TRUE THEN
  RAISE EXCEPTION 'AUT24: DTO de acto invalido' USING ERRCODE='22023'; END IF;
 r:=vec_autorizacion.resolver_rol_administrable_v1(m->>'rol_version_ref');
 IF r->>'huella_sha256' IS DISTINCT FROM m->>'rol_huella_sha256'
 OR (p_propuesta AND r->>'clase' NOT IN ('administrador','intervencion'))
 OR (NOT p_propuesta AND r->>'clase' IS DISTINCT FROM 'ordinario')
 OR ((r->>'unidad_requerida')::boolean AND vec_autorizacion.texto_positivo_valido(m->>'unidad_ref',512) IS NOT TRUE)
 OR (NOT (r->>'unidad_requerida')::boolean AND m ? 'unidad_ref')
 OR (m ? 'referencia_acto' AND vec_autorizacion.texto_positivo_valido(m->>'referencia_acto',512) IS NOT TRUE)
 OR (m->>'actor_persona_ref' IS NOT DISTINCT FROM o->>'persona_ref'
  AND NOT(p_propuesta AND r->>'clase'='administrador' AND m->>'operacion'='revocar')) THEN
  RAISE EXCEPTION 'AUT24: rol, unidad o independencia invalidos' USING ERRCODE='42501'; END IF;
 IF NOT o ?& ARRAY['cuenta_ref','cuenta_version','persona_ref','persona_version','perfil_ref','perfil_version','vinculo_ref','vinculo_version','huella_sha256','revision_continuidad','procedencia_ref','procedencia_version','procedencia_huella_sha256','vigente_hasta']
 OR (SELECT count(*) FROM jsonb_object_keys(o))<>14
 OR o->>'cuenta_ref' !~ '^cta_[A-Za-z0-9_-]{22,128}$'
 OR o->>'persona_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$'
 OR o->>'perfil_ref' !~ '^prf_[A-Za-z0-9_-]{22,128}$'
 OR o->>'vinculo_ref' !~ '^vca_[A-Za-z0-9_-]{22,128}$'
 OR o->>'huella_sha256' !~ '^[0-9a-f]{64}$'
 OR o->>'procedencia_huella_sha256' !~ '^[0-9a-f]{64}$'
 OR (o->>'cuenta_version')::numeric NOT BETWEEN 1 AND 18446744073709551615
 OR (o->>'persona_version')::numeric NOT BETWEEN 1 AND 18446744073709551615
 OR (o->>'procedencia_version')::numeric NOT BETWEEN 1 AND 18446744073709551615
 OR (m->>'operacion'='otorgar' AND ((o->>'perfil_version')::numeric<>0 OR (o->>'vinculo_version')::numeric<>0 OR NOT isfinite((o->>'vigente_hasta')::timestamptz)))
 OR (m->>'operacion'='revocar' AND ((o->>'perfil_version')::numeric<1 OR (o->>'vinculo_version')::numeric<1 OR o->>'vigente_hasta'<>'0001-01-01T00:00:00Z')) THEN
  RAISE EXCEPTION 'AUT24: preimagen estructural invalida' USING ERRCODE='22023'; END IF;
 RETURN m;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.validar_material_acto_admin_v1(text,boolean) FROM PUBLIC;

-- Primitiva compartida: consumo V3 antes de toda recuperación o cambio.
-- p_contexto sigue siendo material central; no se reconstruye desde JSON HTTP.
CREATE FUNCTION vec_autorizacion.consumir_material_admin_interno_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE m jsonb:=p_material::jsonb; d jsonb:=convert_from(p_decision,'UTF8')::jsonb; c jsonb:=convert_from(p_capacidad,'UTF8')::jsonb; x record;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off' THEN
  RAISE EXCEPTION 'AUT24: requiere SERIALIZABLE escritura' USING ERRCODE='25000'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec:admin:continuidad:v1',0));
 IF c->>'efecto_ref' IS DISTINCT FROM m->>'operacion_ref'
 OR c->>'huella_efecto_sha256' IS DISTINCT FROM encode(sha256(convert_to(p_material,'UTF8')),'hex')
 OR d->>'principal_id' IS DISTINCT FROM m->>'actor_persona_ref'
 OR d->>'perfil_activo_ref' IS DISTINCT FROM m->>'actor_perfil_ref'
 OR d->>'modulo_id' IS DISTINCT FROM 'administracion'
 OR d->>'finalidad' IS DISTINCT FROM 'gestion_perfiles'
 OR d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'administracion_privilegiada'
 OR d#>>'{vinculo_autenticacion_actor,cuenta_privilegiada}' IS DISTINCT FROM 'true' THEN
  RAISE EXCEPTION 'AUT24: sello de negocio divergente' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.registrar_consumir_admin_perfiles_v3(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 -- AD150 exige IS12 nominal dentro de la misma transacción, también replay.
 RETURN jsonb_build_object('decision_ref',x.decision_ref,'auditoria_ref',x.auditoria_ref,'consumida_en',x.consumida_en);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.consumir_material_admin_interno_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;

CREATE TABLE vec_autorizacion.sello_efecto_admin_tx_v1(
 asignacion_ref text PRIMARY KEY REFERENCES vec_autorizacion.asignacion_perfil(asignacion_ref),
 transaccion bigint NOT NULL,
 operacion_ref text NOT NULL,
 auditoria_ref text NOT NULL
);
ALTER TABLE vec_autorizacion.sello_efecto_admin_tx_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_autorizacion.sello_efecto_admin_tx_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_exacto ON vec_autorizacion.sello_efecto_admin_tx_v1 FOR ALL TO vec_autorizacion_propietario USING(current_user='vec_autorizacion_propietario') WITH CHECK(current_user='vec_autorizacion_propietario');
CREATE TRIGGER sello_inmutable BEFORE UPDATE OR DELETE ON vec_autorizacion.sello_efecto_admin_tx_v1 FOR EACH ROW EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable();
CREATE TRIGGER sello_no_truncar BEFORE TRUNCATE ON vec_autorizacion.sello_efecto_admin_tx_v1 FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable();
REVOKE ALL ON TABLE vec_autorizacion.sello_efecto_admin_tx_v1 FROM PUBLIC,vec_autorizacion_fuente,vec_admin_perfiles_ejecutor;
REVOKE ALL ON TYPE vec_autorizacion.sello_efecto_admin_tx_v1 FROM PUBLIC,vec_autorizacion_fuente,vec_admin_perfiles_ejecutor;
-- Reemplaza sólo la guarda prospectiva AUT23 por un sello nominal indivisible.
-- Una variable de sesión editable o SET ROLE no puede fabricar este sello.
CREATE OR REPLACE FUNCTION vec_autorizacion.bloquear_asignacion_sensible_hasta_aut24_v1()
RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE rol text;referencia text;
BEGIN
 SELECT r.rol_id,r.version_rol_ref INTO STRICT rol,referencia FROM vec_autorizacion.asignacion_perfil a JOIN vec_autorizacion.version_rol r USING(version_rol_ref) WHERE a.asignacion_ref=NEW.asignacion_ref;
 IF (rol='administracion_perfiles' OR EXISTS(SELECT 1 FROM vec_autorizacion.rol_sensible_exacto WHERE version_rol_ref=referencia))
 AND NOT EXISTS(SELECT 1 FROM vec_autorizacion.sello_efecto_admin_tx_v1 WHERE asignacion_ref=NEW.asignacion_ref AND transaccion=txid_current()) THEN
  RAISE EXCEPTION 'AUT24: asignacion sensible sin acto atestado' USING ERRCODE='42501'; END IF;
 RETURN NEW;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.bloquear_asignacion_sensible_hasta_aut24_v1() FROM PUBLIC;

CREATE FUNCTION vec_autorizacion.ejecutar_cambio_admin_interno_v1(p_material jsonb,p_operacion_ref text,p_propuesta_ref text,p_auditoria text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET timezone='UTC' AS $f$
DECLARE o jsonb:=p_material->'objetivo';configuracion record;antes jsonb;despues jsonb;r jsonb;a record;doc jsonb;
 huella text;asig_id text;asig_ref text;ver bigint;ahora timestamptz;acto text;recibo text;estado text;version_ca numeric;ambitos jsonb;hdespues text;
BEGIN
 antes:=vec_autorizacion.preimagen_cambio_admin_interna_v1(p_material);
 huella:=encode(sha256(convert_to(antes::text,'UTF8')),'hex');
 IF huella IS DISTINCT FROM o->>'huella_sha256' THEN RAISE EXCEPTION 'AUT24: preimagen_huella esperado=% observado=%',o->>'huella_sha256',huella USING ERRCODE='40001'; END IF;
 IF (antes#>>'{contexto,cuenta,version}')::numeric IS DISTINCT FROM (o->>'cuenta_version')::numeric
 OR (antes#>>'{contexto,persona,version}')::numeric IS DISTINCT FROM (o->>'persona_version')::numeric
 OR antes#>>'{continuidad,bootstrap_estado}' IS DISTINCT FROM 'consumido' THEN
  RAISE EXCEPTION 'AUT24: CAS de objetivo o bootstrap divergente' USING ERRCODE='40001'; END IF;
 r:=vec_autorizacion.resolver_rol_administrable_v1(p_material->>'rol_version_ref');
 IF r->>'huella_sha256' IS DISTINCT FROM p_material->>'rol_huella_sha256' THEN RAISE EXCEPTION 'AUT24: rol no vigente' USING ERRCODE='42501'; END IF;
 IF r->>'clase'<>'ordinario' AND ((antes#>>'{continuidad,revision}')::bigint IS DISTINCT FROM (o->>'revision_continuidad')::bigint
 OR (SELECT count(DISTINCT x->>'persona_ref') FROM jsonb_array_elements(antes->'administradores_efectivos') x)<2) THEN
  RAISE EXCEPTION 'AUT24: doble control o continuidad divergente' USING ERRCODE='40001'; END IF;
 ahora:=clock_timestamp();acto:='acto_admin:'||substr(encode(sha256(convert_to(p_operacion_ref,'UTF8')),'hex'),1,32);
 recibo:='recibo_admin:'||substr(encode(sha256(convert_to(p_operacion_ref||':recibo','UTF8')),'hex'),1,32);
 IF p_material->>'operacion'='otorgar' THEN
  IF r->>'clase'='administrador' AND (vec_identidad_sesiones_v1.estado_admin_interno_v1(o->>'persona_ref',o->>'cuenta_ref') IS NULL OR EXISTS(SELECT 1 FROM jsonb_array_elements(antes->'administradores_efectivos') x WHERE x->>'persona_ref'=o->>'persona_ref')) THEN RAISE EXCEPTION 'AUT24: alta de administrador sin identidad privilegiada unica' USING ERRCODE='42501'; END IF;
  IF antes#>'{contexto,perfil}' IS DISTINCT FROM 'null'::jsonb OR antes#>'{contexto,vinculo}' IS DISTINCT FROM 'null'::jsonb OR antes->'asignacion' IS DISTINCT FROM 'null'::jsonb THEN RAISE EXCEPTION 'AUT24: alta requiere referencias nuevas' USING ERRCODE='40001'; END IF;
  PERFORM vec_contexto_actor_v1.crear_perfil_vinculo_admin_v1(o->>'cuenta_ref',o->>'persona_ref',(o->>'cuenta_version')::numeric,(o->>'persona_version')::numeric,o->>'perfil_ref',o->>'vinculo_ref',o->>'procedencia_ref',(o->>'procedencia_version')::numeric,o->>'procedencia_huella_sha256',(o->>'vigente_hasta')::timestamptz);
  version_ca:=1;estado:='activo';ver:=1;asig_id:='admin_'||substr(encode(sha256(convert_to(p_operacion_ref,'UTF8')),'hex'),1,32);
  IF (r->>'unidad_requerida')::boolean THEN ambitos:=jsonb_build_array(jsonb_build_object('clave','unidad','valores',jsonb_build_array(p_material->>'unidad_ref')));
  ELSE SELECT * INTO STRICT configuracion FROM vec_autorizacion.rol_administrable_exacto_v1 WHERE version_rol_ref=p_material->>'rol_version_ref'; ambitos:=configuracion.ambitos_fijos; IF vec_autorizacion.ambitos_positivos_validos(jsonb_build_object('ambitos',ambitos)) IS NOT TRUE THEN RAISE EXCEPTION 'AUT24: ambito fijo no publicado' USING ERRCODE='42501'; END IF; END IF;
  doc:=jsonb_build_object('asignacion_id',asig_id,'version',ver,'perfil_activo_ref',o->>'perfil_ref','principal_id',o->>'persona_ref','version_rol_ref',p_material->>'rol_version_ref','rol_huella_sha256',p_material->>'rol_huella_sha256','estado','activa','cuenta_ref',o->>'cuenta_ref','vinculo_ref',o->>'vinculo_ref','ambitos',ambitos,'emitida_por',p_material->>'actor_persona_ref','emitida_en',to_char(ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'vigente_desde',to_char(ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'vigente_hasta',to_char((o->>'vigente_hasta')::timestamptz AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'referencia_acto',p_material->>'referencia_acto');
 ELSE
  IF antes#>>'{contexto,perfil,persona_ref}' IS DISTINCT FROM o->>'persona_ref' OR antes#>>'{contexto,vinculo,persona_ref}' IS DISTINCT FROM o->>'persona_ref' OR antes#>>'{contexto,vinculo,cuenta_ref}' IS DISTINCT FROM o->>'cuenta_ref' OR antes#>>'{contexto,vinculo,perfil_ref}' IS DISTINCT FROM o->>'perfil_ref'
  OR (antes#>>'{contexto,perfil,version}')::numeric IS DISTINCT FROM (o->>'perfil_version')::numeric OR (antes#>>'{contexto,vinculo,version}')::numeric IS DISTINCT FROM (o->>'vinculo_version')::numeric
  OR antes#>>'{asignacion,version_rol_ref}' IS DISTINCT FROM p_material->>'rol_version_ref' OR antes#>>'{asignacion,estado}' IS DISTINCT FROM 'activa' THEN RAISE EXCEPTION 'AUT24: baja sin vinculo y asignacion exactos' USING ERRCODE='40001'; END IF;
  IF (r->>'unidad_requerida')::boolean AND antes#>'{asignacion,ambitos}' IS DISTINCT FROM jsonb_build_array(jsonb_build_object('clave','unidad','valores',jsonb_build_array(p_material->>'unidad_ref'))) THEN RAISE EXCEPTION 'AUT24: unidad de baja divergente' USING ERRCODE='42501'; END IF;
  SELECT * INTO STRICT a FROM vec_autorizacion.asignacion_perfil WHERE asignacion_ref=antes#>>'{asignacion,asignacion_ref}';
  PERFORM vec_contexto_actor_v1.revocar_perfil_vinculo_admin_v1(o->>'cuenta_ref',o->>'persona_ref',o->>'perfil_ref',o->>'vinculo_ref',(o->>'cuenta_version')::numeric,(o->>'persona_version')::numeric,(o->>'perfil_version')::numeric,(o->>'vinculo_version')::numeric,o->>'procedencia_ref',(o->>'procedencia_version')::numeric,o->>'procedencia_huella_sha256');
  version_ca:=(o->>'perfil_version')::numeric+1;estado:='revocado';ver:=a.version+1;asig_id:=a.asignacion_id;
  doc:=jsonb_set(jsonb_set(a.documento,'{version}',to_jsonb(ver)),'{estado}','"revocada"'::jsonb);
 END IF;
 asig_ref:='asignacion:'||asig_id||':v'||ver;
 INSERT INTO vec_autorizacion.asignacion_perfil(asignacion_ref,asignacion_id,version,perfil_activo_ref,principal_id,version_rol_ref,huella_sha256,emitida_en,documento)
 VALUES(asig_ref,asig_id,ver,o->>'perfil_ref',o->>'persona_ref',p_material->>'rol_version_ref',encode(sha256(convert_to(doc::text,'UTF8')),'hex'),(doc->>'emitida_en')::timestamptz,doc);
 INSERT INTO vec_autorizacion.sello_efecto_admin_tx_v1 VALUES(asig_ref,txid_current(),p_operacion_ref,p_auditoria);
 IF p_material->>'operacion'='otorgar' THEN INSERT INTO vec_autorizacion.asignacion_perfil_actual VALUES(o->>'perfil_ref',asig_ref,ahora,p_material->>'actor_persona_ref',acto);
 ELSE UPDATE vec_autorizacion.asignacion_perfil_actual SET asignacion_ref=asig_ref,actualizada_en=ahora,actualizada_por=p_material->>'actor_persona_ref',acto_ref=acto WHERE perfil_activo_ref=o->>'perfil_ref' AND asignacion_ref=antes#>>'{asignacion,asignacion_ref}'; IF NOT FOUND THEN RAISE EXCEPTION 'AUT24: CAS de asignacion perdido' USING ERRCODE='40001'; END IF; END IF;
 IF r->>'clase'<>'ordinario' THEN
  PERFORM vec_autorizacion.avanzar_continuidad_admin_interna_v1((o->>'revision_continuidad')::bigint);
  IF (SELECT count(DISTINCT x->>'persona_ref') FROM jsonb_array_elements(vec_autorizacion.administradores_efectivos_internos_v1()) x)<1 THEN RAISE EXCEPTION 'AUT24: ultimo administrador no revocable' USING ERRCODE='42501'; END IF;
 END IF;
 despues:=vec_autorizacion.preimagen_cambio_admin_interna_v1(p_material);
 hdespues:=encode(sha256(convert_to(despues::text,'UTF8')),'hex');
 RETURN jsonb_build_object('operacion_ref',p_operacion_ref,'acto_ref',acto,'recibo_ref',recibo,'propuesta_ref',coalesce(p_propuesta_ref,''),'auditoria_ref',p_auditoria,'objetivo_persona_ref',o->>'persona_ref','perfil_ref',o->>'perfil_ref','vinculo_ref',o->>'vinculo_ref','estado_posterior',estado,'version_posterior',version_ca,'huella_antes_sha256',huella,'huella_despues_sha256',hdespues,'confirmado_en',ahora,'unidad_ref',coalesce(p_material->>'unidad_ref',''),'referencia_acto',coalesce(p_material->>'referencia_acto',''));
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.ejecutar_cambio_admin_interno_v1(jsonb,text,text,text) FROM PUBLIC;

CREATE FUNCTION vec_autorizacion.registrar_resultado_admin_interno_v1(p_material text,p_consumo jsonb,p_resultado jsonb,p_evento text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE m jsonb:=p_material::jsonb;
BEGIN
 INSERT INTO vec_autorizacion.registro_acto_admin_v1 VALUES(m->>'operacion_ref',convert_to(p_material,'UTF8'),encode(sha256(convert_to(p_material,'UTF8')),'hex'),m->>'actor_persona_ref',m->>'actor_perfil_ref',p_consumo->>'decision_ref',p_consumo->>'auditoria_ref',p_resultado,clock_timestamp());
 INSERT INTO vec_autorizacion.outbox_acto_admin_v1 VALUES(m->>'operacion_ref',p_evento,p_consumo->>'auditoria_ref',clock_timestamp());
 RETURN p_resultado;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.registrar_resultado_admin_interno_v1(text,jsonb,jsonb,text) FROM PUBLIC;
CREATE FUNCTION vec_autorizacion.recuperar_resultado_admin_interno_v1(p_material text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE r record;m jsonb:=p_material::jsonb;
BEGIN
 SELECT * INTO r FROM vec_autorizacion.registro_acto_admin_v1 WHERE operacion_ref=m->>'operacion_ref';
 IF NOT FOUND THEN RETURN null; END IF;
 IF r.material IS DISTINCT FROM convert_to(p_material,'UTF8') THEN RAISE EXCEPTION 'AUT24: idempotencia divergente' USING ERRCODE='23505'; END IF;
 RETURN r.resultado;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.recuperar_resultado_admin_interno_v1(text) FROM PUBLIC;

CREATE FUNCTION vec_autorizacion.aplicar_acto_ordinario_admin_v1(p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET timezone='UTC' AS $f$
DECLARE m jsonb;consumo jsonb;resultado jsonb;d jsonb:=convert_from(p_decision,'UTF8')::jsonb;efectivos jsonb;
BEGIN
 m:=vec_autorizacion.validar_material_acto_admin_v1(p_material,false);
 IF d->>'accion' IS DISTINCT FROM 'administracion.perfiles.'||(m->>'operacion') THEN RAISE EXCEPTION 'AUT24: accion ordinaria divergente' USING ERRCODE='42501'; END IF;
 consumo:=vec_autorizacion.consumir_material_admin_interno_v1(p_material,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 efectivos:=vec_autorizacion.administradores_efectivos_internos_v1();
 IF NOT EXISTS(SELECT 1 FROM jsonb_array_elements(efectivos) x WHERE x->>'persona_ref'=m->>'actor_persona_ref' AND x->>'perfil_ref'=m->>'actor_perfil_ref') THEN RAISE EXCEPTION 'AUT24: administrador no efectivo' USING ERRCODE='42501'; END IF;
 resultado:=vec_autorizacion.recuperar_resultado_admin_interno_v1(p_material);
 IF resultado IS NOT NULL THEN RETURN resultado; END IF;
 resultado:=vec_autorizacion.ejecutar_cambio_admin_interno_v1(m,m->>'operacion_ref',null,consumo->>'auditoria_ref');
 RETURN vec_autorizacion.registrar_resultado_admin_interno_v1(p_material,consumo,resultado,CASE WHEN m->>'operacion'='otorgar' THEN 'perfil_otorgado' ELSE 'perfil_revocado' END);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.aplicar_acto_ordinario_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion.aplicar_acto_ordinario_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_admin_perfiles_ejecutor;

CREATE FUNCTION vec_autorizacion.proponer_acto_admin_v1(p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET timezone='UTC' AS $f$
DECLARE m jsonb;consumo jsonb;resultado jsonb;preimagen jsonb;rol jsonb;huella text;caduca timestamptz;ahora timestamptz;configuracion record;efectivos jsonb;d jsonb:=convert_from(p_decision,'UTF8')::jsonb;
BEGIN
 m:=vec_autorizacion.validar_material_acto_admin_v1(p_material,true);
 IF d->>'accion' IS DISTINCT FROM 'administracion.perfiles.proponer' THEN RAISE EXCEPTION 'AUT24: accion propuesta divergente' USING ERRCODE='42501'; END IF;
 consumo:=vec_autorizacion.consumir_material_admin_interno_v1(p_material,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 efectivos:=vec_autorizacion.administradores_efectivos_internos_v1();
 IF (SELECT count(DISTINCT x->>'persona_ref') FROM jsonb_array_elements(efectivos) x)<2 OR NOT EXISTS(SELECT 1 FROM jsonb_array_elements(efectivos) x WHERE x->>'persona_ref'=m->>'actor_persona_ref' AND x->>'perfil_ref'=m->>'actor_perfil_ref') THEN RAISE EXCEPTION 'AUT24: doble control no disponible' USING ERRCODE='42501'; END IF;
 resultado:=vec_autorizacion.recuperar_resultado_admin_interno_v1(p_material);
 IF resultado IS NOT NULL THEN RETURN resultado; END IF;
 preimagen:=vec_autorizacion.preimagen_cambio_admin_interna_v1(m);
 huella:=encode(sha256(convert_to(preimagen::text,'UTF8')),'hex');
 IF huella IS DISTINCT FROM m#>>'{objetivo,huella_sha256}' OR (preimagen#>>'{continuidad,revision}')::bigint IS DISTINCT FROM (m#>>'{objetivo,revision_continuidad}')::bigint OR preimagen#>>'{continuidad,bootstrap_estado}' IS DISTINCT FROM 'consumido' THEN RAISE EXCEPTION 'AUT24: preimagen de propuesta divergente' USING ERRCODE='40001'; END IF;
 rol:=vec_autorizacion.resolver_rol_administrable_v1(m->>'rol_version_ref');
 SELECT * INTO STRICT configuracion FROM vec_autorizacion.rol_administrable_exacto_v1 WHERE version_rol_ref=m->>'rol_version_ref';
 ahora:=clock_timestamp();caduca:=LEAST(ahora+configuracion.duracion_propuesta,configuracion.vigente_hasta);
 INSERT INTO vec_autorizacion.propuesta_perfil_sensible(propuesta_ref,clase,operacion,version_rol_ref,objetivo_persona_ref,objetivo_cuenta_ref,objetivo_perfil_ref,proponente_persona_ref,proponente_cuenta_ref,proponente_perfil_ref,motivo_codigo,revision_continuidad_esperada,preimagen_huella_sha256,documento_canonico,huella_sha256,creada_en,caduca_en)
 VALUES(m->>'operacion_ref',rol->>'clase',m->>'operacion',m->>'rol_version_ref',m#>>'{objetivo,persona_ref}',m#>>'{objetivo,cuenta_ref}',m#>>'{objetivo,perfil_ref}',m->>'actor_persona_ref',d#>>'{vinculo_autenticacion_actor,cuenta_ref}',m->>'actor_perfil_ref',m#>>'{motivo,entrada_clave}',(m#>>'{objetivo,revision_continuidad}')::bigint,huella,convert_to(p_material,'UTF8'),encode(sha256(convert_to(p_material,'UTF8')),'hex'),ahora,caduca);
 resultado:=jsonb_build_object('operacion_ref',m->>'operacion_ref','propuesta_ref',m->>'operacion_ref','huella_sha256',encode(sha256(convert_to(p_material,'UTF8')),'hex'),'proponente_persona_ref',m->>'actor_persona_ref','objetivo_persona_ref',m#>>'{objetivo,persona_ref}','caduca_en',caduca);
 RETURN vec_autorizacion.registrar_resultado_admin_interno_v1(p_material,consumo,resultado,'propuesta_registrada');
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.proponer_acto_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion.proponer_acto_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_admin_perfiles_ejecutor;

CREATE FUNCTION vec_autorizacion.cerrar_propuesta_admin_v1(p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET timezone='UTC' AS $f$
DECLARE m jsonb:=p_material::jsonb;consumo jsonb;resultado jsonb;propuesta record;material_propuesta jsonb;rol jsonb;revision bigint;efectivos jsonb;huella text;ahora timestamptz;recibo jsonb:=null;acto jsonb;d jsonb:=convert_from(p_decision,'UTF8')::jsonb;claves text[];
BEGIN
 IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 1 AND 65536 THEN RAISE EXCEPTION 'AUT24: cierre invalido' USING ERRCODE='22023'; END IF;
 SELECT array_agg(key ORDER BY key) INTO claves FROM jsonb_each(m);
 IF claves IS DISTINCT FROM ARRAY['actor_perfil_ref','actor_persona_ref','correlacion_ref','decision','esquema','motivo','operacion_ref','propuesta_huella_sha256','propuesta_ref']::text[] OR jsonb_path_exists(m,'$.** ? (@ == null)') OR m->>'esquema' IS DISTINCT FROM 'administracion_perfiles_cierre_v1' OR m->>'operacion_ref' !~ '^cierre_admin:[0-9a-f]{32}$' OR m->>'propuesta_ref' !~ '^propuesta_admin:[0-9a-f]{32}$' OR m->>'propuesta_huella_sha256' !~ '^[0-9a-f]{64}$' OR m->>'decision' NOT IN ('aprobada','rechazada') THEN RAISE EXCEPTION 'AUT24: DTO cierre invalido' USING ERRCODE='22023'; END IF;
 IF jsonb_typeof(m->'motivo') IS DISTINCT FROM 'object' OR NOT (m->'motivo') ?& ARRAY['catalogo_id','catalogo_version','catalogo_huella_sha256','entrada_clave'] OR (SELECT count(*) FROM jsonb_object_keys(m->'motivo'))<>4 OR vec_autorizacion.texto_positivo_valido(m#>>'{motivo,catalogo_id}',256) IS NOT TRUE OR vec_autorizacion.texto_positivo_valido(m#>>'{motivo,entrada_clave}',128) IS NOT TRUE OR (m#>>'{motivo,catalogo_version}')::numeric<1 OR m#>>'{motivo,catalogo_huella_sha256}' !~ '^[0-9a-f]{64}$' THEN RAISE EXCEPTION 'AUT24: motivo de cierre invalido' USING ERRCODE='22023'; END IF;
 IF d->>'accion' IS DISTINCT FROM (CASE WHEN m->>'decision'='aprobada' THEN 'administracion.perfiles.aprobar' ELSE 'administracion.perfiles.rechazar' END) THEN RAISE EXCEPTION 'AUT24: accion cierre divergente' USING ERRCODE='42501'; END IF;
 consumo:=vec_autorizacion.consumir_material_admin_interno_v1(p_material,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 efectivos:=vec_autorizacion.administradores_efectivos_internos_v1();
 IF NOT EXISTS(SELECT 1 FROM jsonb_array_elements(efectivos) x WHERE x->>'persona_ref'=m->>'actor_persona_ref' AND x->>'perfil_ref'=m->>'actor_perfil_ref') THEN RAISE EXCEPTION 'AUT24: aprobador no efectivo' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT propuesta FROM vec_autorizacion.propuesta_perfil_sensible WHERE propuesta_ref=m->>'propuesta_ref' FOR SHARE;
 IF propuesta.huella_sha256 IS DISTINCT FROM m->>'propuesta_huella_sha256' OR m->>'actor_persona_ref' IN (propuesta.proponente_persona_ref,propuesta.objetivo_persona_ref) THEN RAISE EXCEPTION 'AUT24: cierre no independiente' USING ERRCODE='42501'; END IF;
 resultado:=vec_autorizacion.recuperar_resultado_admin_interno_v1(p_material);
 IF resultado IS NOT NULL THEN RETURN resultado; END IF;
 SELECT cc.revision INTO STRICT revision FROM vec_autorizacion.control_continuidad_admin cc WHERE cc.control_id FOR UPDATE;
 IF EXISTS(SELECT 1 FROM vec_autorizacion.cierre_propuesta_perfil_sensible WHERE propuesta_ref=propuesta.propuesta_ref) THEN RAISE EXCEPTION 'AUT24: propuesta ya cerrada' USING ERRCODE='23505'; END IF;
 material_propuesta:=convert_from(propuesta.documento_canonico,'UTF8')::jsonb;
 IF m->>'decision'='aprobada' THEN
  IF clock_timestamp()>=propuesta.caduca_en OR revision IS DISTINCT FROM propuesta.revision_continuidad_esperada OR (SELECT count(DISTINCT x->>'persona_ref') FROM jsonb_array_elements(efectivos) x)<2 OR NOT EXISTS(SELECT 1 FROM jsonb_array_elements(efectivos) x WHERE x->>'persona_ref'=propuesta.proponente_persona_ref AND x->>'perfil_ref'=propuesta.proponente_perfil_ref) THEN RAISE EXCEPTION 'AUT24: propuesta obsoleta o proponente revocado' USING ERRCODE='40001'; END IF;
 END IF;
 ahora:=clock_timestamp();huella:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 INSERT INTO vec_autorizacion.cierre_propuesta_perfil_sensible(propuesta_ref,cierre_ref,resultado,aprobador_persona_ref,aprobador_cuenta_ref,aprobador_perfil_ref,motivo_codigo,revision_continuidad_observada,documento_canonico,huella_sha256,cerrada_en)
 VALUES(propuesta.propuesta_ref,m->>'operacion_ref',m->>'decision',m->>'actor_persona_ref',d#>>'{vinculo_autenticacion_actor,cuenta_ref}',m->>'actor_perfil_ref',m#>>'{motivo,entrada_clave}',revision,convert_to(p_material,'UTF8'),huella,ahora);
 IF m->>'decision'='aprobada' THEN
  recibo:=vec_autorizacion.ejecutar_cambio_admin_interno_v1(material_propuesta,m->>'operacion_ref',propuesta.propuesta_ref,consumo->>'auditoria_ref');
  ahora:=(recibo->>'confirmado_en')::timestamptz;
  acto:=jsonb_build_object('operacion_ref',m->>'operacion_ref','propuesta_ref',propuesta.propuesta_ref,'preimagen_huella_sha256',recibo->>'huella_antes_sha256','postimagen_huella_sha256',recibo->>'huella_despues_sha256','auditoria_ref',consumo->>'auditoria_ref');
  INSERT INTO vec_autorizacion.acto_perfil_sensible VALUES(recibo->>'acto_ref',propuesta.propuesta_ref,m->>'operacion_ref',revision,revision+1,recibo->>'huella_antes_sha256',recibo->>'huella_despues_sha256',convert_to(acto::text,'UTF8'),encode(sha256(convert_to(acto::text,'UTF8')),'hex'),ahora);
  INSERT INTO vec_autorizacion.recibo_perfil_sensible VALUES(recibo->>'recibo_ref',recibo->>'acto_ref',convert_to(recibo::text,'UTF8'),encode(sha256(convert_to(recibo::text,'UTF8')),'hex'),ahora);
 END IF;
 resultado:=jsonb_build_object('operacion_ref',m->>'operacion_ref','propuesta_ref',propuesta.propuesta_ref,'decision',m->>'decision','huella_cierre_sha256',huella,'confirmado_en',ahora,'recibo',recibo);
 RETURN vec_autorizacion.registrar_resultado_admin_interno_v1(p_material,consumo,resultado,CASE WHEN m->>'decision'='aprobada' THEN 'propuesta_aprobada' ELSE 'propuesta_rechazada' END);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.cerrar_propuesta_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion.cerrar_propuesta_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_admin_perfiles_ejecutor;

GRANT USAGE ON SCHEMA vec_autorizacion TO vec_admin_perfiles_ejecutor;
GRANT EXECUTE ON FUNCTION vec_autorizacion.resolver_rol_administrable_v1(text) TO vec_admin_perfiles_ejecutor;
CREATE FUNCTION vec_autorizacion.consultar_asignacion_admin_perfiles_v1(p_cuenta text)
RETURNS TABLE(persona_ref text,perfil_ref text,vinculo_ref text,cuenta_version numeric,persona_version numeric,perfil_version numeric,vinculo_version numeric,audiencia text,vigente_hasta timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET timezone='UTC' AS $f$
DECLARE a record;c jsonb;i jsonb;cfg record;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off' THEN RAISE EXCEPTION 'AUT24: contexto requiere SERIALIZABLE' USING ERRCODE='25000'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec:admin:continuidad:v1',0));
 IF NOT EXISTS(SELECT 1 FROM vec_autorizacion.control_continuidad_admin WHERE control_id AND bootstrap_estado='consumido') THEN RETURN; END IF;
 FOR a IN SELECT x.* FROM vec_autorizacion.asignacion_perfil_actual ac JOIN vec_autorizacion.asignacion_perfil x USING(asignacion_ref)
 JOIN vec_autorizacion.rol_sensible_exacto sx USING(version_rol_ref) JOIN vec_autorizacion.version_rol r USING(version_rol_ref)
 JOIN vec_autorizacion.control_vigencia_version_rol_actual rc USING(version_rol_ref)
 JOIN vec_autorizacion.control_vigencia_version_rol cv ON cv.version_rol_ref=rc.version_rol_ref AND cv.revision=rc.revision
 WHERE x.documento->>'cuenta_ref'=p_cuenta AND x.documento->>'estado'='activa'
 AND x.version_rol_ref='rol:administracion_perfiles:v2' AND sx.clase='administrador' AND sx.huella_sha256=r.huella_sha256 AND r.documento->>'estado'='publicada' AND cv.estado='habilitada'
 AND clock_timestamp()>=(x.documento->>'vigente_desde')::timestamptz AND clock_timestamp()<(x.documento->>'vigente_hasta')::timestamptz
 ORDER BY x.principal_id,x.perfil_activo_ref FOR UPDATE OF ac LOOP
  SELECT * INTO STRICT cfg FROM vec_autorizacion.rol_administrable_exacto_v1 WHERE version_rol_ref=a.version_rol_ref AND huella_sha256=a.documento->>'rol_huella_sha256';
  IF cfg.clase<>'administrador' OR clock_timestamp()<cfg.vigente_desde OR clock_timestamp()>=cfg.vigente_hasta THEN CONTINUE; END IF;
  c:=vec_contexto_actor_v1.preimagen_admin_interna_v1(p_cuenta,a.principal_id,a.perfil_activo_ref,a.documento->>'vinculo_ref');
  i:=vec_identidad_sesiones_v1.estado_admin_interno_v1(a.principal_id,p_cuenta);
  IF i IS NULL OR c#>>'{perfil,estado}' IS DISTINCT FROM 'activo' OR c#>>'{vinculo,estado}' IS DISTINCT FROM 'activo' OR c#>>'{perfil,persona_ref}' IS DISTINCT FROM a.principal_id OR c#>>'{vinculo,persona_ref}' IS DISTINCT FROM a.principal_id OR c#>>'{vinculo,cuenta_ref}' IS DISTINCT FROM p_cuenta OR c#>>'{vinculo,perfil_ref}' IS DISTINCT FROM a.perfil_activo_ref THEN CONTINUE; END IF;
  RETURN QUERY SELECT a.principal_id,a.perfil_activo_ref,a.documento->>'vinculo_ref',(c#>>'{cuenta,version}')::numeric,(c#>>'{persona,version}')::numeric,(c#>>'{perfil,version}')::numeric,(c#>>'{vinculo,version}')::numeric,cfg.audiencia_administrativa,LEAST((c#>>'{cuenta,vigente_hasta}')::timestamptz,(c#>>'{persona,vigente_hasta}')::timestamptz,(c#>>'{perfil,vigente_hasta}')::timestamptz,(c#>>'{vinculo,vigente_hasta}')::timestamptz,(i->>'vigente_hasta')::timestamptz,(a.documento->>'vigente_hasta')::timestamptz,cfg.vigente_hasta);
 END LOOP;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.consultar_asignacion_admin_perfiles_v1(text) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion TO vec_contexto_actor_v1_propietario;
GRANT EXECUTE ON FUNCTION vec_autorizacion.consultar_asignacion_admin_perfiles_v1(text) TO vec_contexto_actor_v1_propietario;
COMMIT;
