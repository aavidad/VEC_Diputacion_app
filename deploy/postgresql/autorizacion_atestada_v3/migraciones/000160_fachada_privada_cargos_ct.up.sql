\set ON_ERROR_STOP on
-- AD160: provisión privada nominal sobre la autoridad central. No publica
-- plantilla Aplicación ni crea personas/perfiles CA. No se monta en HTTP.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion:cargos_ct:migracion:000160',0));
DO $preimagen$
DECLARE f record;
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
  OR pg_catalog.current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
  OR pg_catalog.to_regrole('vec_autorizacion_cargos_ct_ejecutor') IS NOT NULL
  OR pg_catalog.to_regclass('vec_autorizacion.cargo_ct_plan') IS NOT NULL
  OR pg_catalog.to_regclass('vec_autorizacion.cargo_ct_aprobacion') IS NOT NULL
  OR pg_catalog.to_regclass('vec_autorizacion.cargo_ct_recibo') IS NOT NULL
  OR pg_catalog.to_regclass('vec_autorizacion.cargo_ct_consumo') IS NOT NULL
  OR pg_catalog.to_regclass('vec_autorizacion.cargo_ct_auditoria') IS NOT NULL
  OR pg_catalog.to_regprocedure('vec_contexto_actor_v1.acreditar_destino_cargo_ct_v1(text,text,text)') IS NOT NULL
  OR pg_catalog.to_regprocedure('vec_identidad_sesiones_v1.acreditar_operador_cargos_ct_v1(jsonb)') IS NOT NULL
 THEN RAISE EXCEPTION 'AD160: preimagen incompatible' USING ERRCODE='55000'; END IF;
 FOR f IN SELECT * FROM (VALUES
  ('vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(bytea,bytea,numeric,numeric)','vec_autorizacion_propietario'),
  ('vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(text,text,text,text,text,text,numeric,text,numeric,text,numeric,text,numeric,text,text,timestamptz,timestamptz)','vec_contexto_actor_v1_propietario'),
  ('vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(text,text)','vec_identidad_sesiones_v1_propietario')) AS v(firma,propietario)
 LOOP
  IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_proc p WHERE p.oid=pg_catalog.to_regprocedure(f.firma)
   AND p.proowner=pg_catalog.to_regrole(f.propietario) AND p.prosecdef
   AND p.provolatile='v' AND EXISTS(SELECT 1 FROM pg_catalog.unnest(p.proconfig) c WHERE c IN ('search_path=pg_catalog','search_path=pg_catalog, pg_temp')))
  THEN RAISE EXCEPTION 'AD160: fuente central ausente %',f.firma USING ERRCODE='55000'; END IF;
 END LOOP;
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_class c WHERE c.oid='vec_autorizacion.asignacion_perfil_actual'::regclass
  AND c.relowner='vec_autorizacion_propietario'::regrole AND c.relrowsecurity AND c.relforcerowsecurity)
 THEN RAISE EXCEPTION 'AD160: asignación central no acreditada' USING ERRCODE='55000'; END IF;
END $preimagen$;
CREATE ROLE vec_autorizacion_cargos_ct_ejecutor NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
DO $conexion$ BEGIN
 EXECUTE pg_catalog.format('GRANT CONNECT ON DATABASE %I TO vec_autorizacion_cargos_ct_ejecutor',pg_catalog.current_database());
END $conexion$;

-- El propietario CA deriva la persona/perfil de un registro real y llama a su
-- propia acreditación viva. AUT no lee tablas de identidad ni fabrica Vínculo.
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
CREATE FUNCTION vec_contexto_actor_v1.acreditar_destino_cargo_ct_v1(p_registro text,p_persona text,p_perfil text)
RETURNS text LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE r record; x jsonb; ahora timestamptz; acreditada timestamptz; valida_hasta timestamptz;
BEGIN
 SELECT * INTO STRICT r FROM vec_contexto_actor_v1.registros_contexto WHERE registro_contexto_ref=p_registro;
 x:=pg_catalog.convert_from(r.representacion_canonica,'UTF8')::jsonb;
 IF x->>'persona_ref' IS DISTINCT FROM p_persona OR x->>'principal_ref' IS DISTINCT FROM p_persona
  OR x->>'perfil_activo_ref' IS DISTINCT FROM p_perfil OR r.perfil_ref IS DISTINCT FROM p_perfil
  OR x->>'estado' IS DISTINCT FROM 'activo' OR x->>'garantia' IS DISTINCT FROM 'alto'
  OR x->>'metodo' NOT IN ('certificado','dnie','kerberos_ad')
 THEN RETURN NULL; END IF;
 ahora:=pg_catalog.clock_timestamp();
 -- La acreditación es actual: no exige que persona/cuenta/perfil tengan
 -- necesariamente la misma fecha final del vínculo de contexto registrado.
 SELECT LEAST(c.vigente_hasta,pe.vigente_hasta,pf.vigente_hasta,(x->>'vigente_hasta')::timestamptz)
 INTO STRICT valida_hasta
 FROM vec_contexto_actor_v1.proyeccion_cuenta_actual ca
 JOIN vec_contexto_actor_v1.proyeccion_cuenta_versiones c USING(cuenta_ref,version)
 JOIN vec_contexto_actor_v1.persona_actual pa ON pa.persona_ref=p_persona
 JOIN vec_contexto_actor_v1.persona_versiones pe ON pe.persona_ref=pa.persona_ref AND pe.version=pa.version
 JOIN vec_contexto_actor_v1.perfil_actual fa ON fa.perfil_ref=p_perfil
 JOIN vec_contexto_actor_v1.perfil_versiones pf ON pf.perfil_ref=fa.perfil_ref AND pf.version=fa.version
 WHERE ca.cuenta_ref=x->>'cuenta_ref';
 acreditada:=vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(
  p_registro,x->>'esquema',r.huella_sha256,r.manifiesto_procedencia_huella_sha256,r.autoridad_efectiva,
  x->>'cuenta_ref',(x->>'cuenta_version')::numeric,p_persona,(x->>'persona_version')::numeric,
  p_perfil,(x->>'perfil_version')::numeric,x->>'contexto_actor_ref',(x->>'contexto_version')::numeric,
  x->>'metodo',x->>'garantia',ahora,valida_hasta);
 IF acreditada IS NULL THEN RETURN NULL; END IF;
 RETURN pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(r.huella_sha256||':'||r.manifiesto_procedencia_huella_sha256,'UTF8')),'hex');
EXCEPTION WHEN no_data_found OR too_many_rows OR data_exception THEN RETURN NULL;
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.acreditar_destino_cargo_ct_v1(text,text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.acreditar_destino_cargo_ct_v1(text,text,text) TO vec_autorizacion_propietario;

-- IS conserva sus barreras de revocación hasta COMMIT. La proyección booleana
-- exige la sesión administrativa que aparece en la decisión registrada.
RESET ROLE;
SET LOCAL ROLE vec_identidad_sesiones_v1_propietario;
CREATE FUNCTION vec_identidad_sesiones_v1.acreditar_operador_cargos_ct_v1(v jsonb)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE s record;
BEGIN
 SELECT * INTO STRICT s FROM vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(v->>'autenticacion_ref',v->>'sesion_ref');
 RETURN s.cuenta_privilegiada IS TRUE AND s.superficie='administracion_privilegiada'
  AND s.garantia_observada='alto' AND s.metodo_observado IN ('certificado','dnie')
  AND s.autenticacion_huella_sha256=v->>'autenticacion_huella_sha256'
  AND s.asercion_ref=v->>'asercion_ref' AND s.cuenta_ref=v->>'cuenta_ref'
  AND s.cuenta_ordinaria_ref=v->>'cuenta_ordinaria_ref'
  AND s.control_sesion_ref=v->>'control_sesion_ref'
  AND s.control_sesion_revision=v->>'control_sesion_revision'
  AND s.control_sesion_huella_sha256=v->>'control_sesion_huella_sha256';
EXCEPTION WHEN no_data_found OR too_many_rows OR data_exception THEN RETURN false;
END $f$;
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.acreditar_operador_cargos_ct_v1(jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_identidad_sesiones_v1.acreditar_operador_cargos_ct_v1(jsonb) TO vec_autorizacion_propietario;
RESET ROLE;
GRANT USAGE ON SCHEMA vec_identidad_sesiones_v1 TO vec_autorizacion_propietario;
SET LOCAL ROLE vec_autorizacion_propietario;
CREATE TABLE vec_autorizacion.cargo_ct_plan (
 clave text PRIMARY KEY CHECK(clave ~ '^[0-9a-f]{32}$'),
 plan_canonico bytea NOT NULL CHECK(pg_catalog.octet_length(plan_canonico) BETWEEN 1 AND 32768),
 plan_sha256 text NOT NULL CHECK(plan_sha256=pg_catalog.encode(pg_catalog.sha256(plan_canonico),'hex')),
 proponente_ref text NOT NULL,decision_ref text NOT NULL UNIQUE,
 creada_en timestamptz(6) NOT NULL,caduca_en timestamptz(6) NOT NULL CHECK(caduca_en>creada_en)
);
CREATE TABLE vec_autorizacion.cargo_ct_aprobacion (
 clave text PRIMARY KEY REFERENCES vec_autorizacion.cargo_ct_plan(clave),
 plan_sha256 text NOT NULL CHECK(plan_sha256 ~ '^[0-9a-f]{64}$'),
 aprobador_ref text NOT NULL,decision_ref text NOT NULL UNIQUE,
 decision_canonica bytea NOT NULL,motivo_canonico bytea NOT NULL,
 persona_version numeric NOT NULL,perfil_version numeric NOT NULL,
 aprobada_en timestamptz(6) NOT NULL
);
CREATE TABLE vec_autorizacion.cargo_ct_consumo (
 decision_ref text PRIMARY KEY,clave text NOT NULL REFERENCES vec_autorizacion.cargo_ct_plan(clave),
 operacion text NOT NULL CHECK(operacion IN ('preparar','aprobar','aplicar','recuperar')),
 consumida_en timestamptz(6) NOT NULL
);
CREATE TABLE vec_autorizacion.cargo_ct_auditoria (
 auditoria_ref text PRIMARY KEY,clave text,plan_sha256 text NOT NULL CHECK(plan_sha256 ~ '^[0-9a-f]{64}$'),
 operacion text NOT NULL CHECK(operacion IN ('preparar','aprobar','aplicar','recuperar')),
 actor_ref text NOT NULL,resultado text NOT NULL CHECK(resultado IN ('confirmada','denegada')),
 registrada_en timestamptz(6) NOT NULL
);
CREATE TABLE vec_autorizacion.cargo_ct_recibo (
 clave text PRIMARY KEY REFERENCES vec_autorizacion.cargo_ct_plan(clave),
 recibo_ref text NOT NULL UNIQUE,plan_sha256 text NOT NULL CHECK(plan_sha256 ~ '^[0-9a-f]{64}$'),
 asignacion_ref text NOT NULL REFERENCES vec_autorizacion.asignacion_perfil(asignacion_ref),
 auditoria_ref text NOT NULL REFERENCES vec_autorizacion.cargo_ct_auditoria(auditoria_ref),
 documento jsonb NOT NULL CHECK(pg_catalog.jsonb_typeof(documento)='object'),
 emitido_en timestamptz(6) NOT NULL
);
DO $historia$
DECLARE n text;
BEGIN
 FOREACH n IN ARRAY ARRAY['cargo_ct_plan','cargo_ct_aprobacion','cargo_ct_consumo','cargo_ct_auditoria','cargo_ct_recibo'] LOOP
  EXECUTE pg_catalog.format('ALTER TABLE vec_autorizacion.%I ENABLE ROW LEVEL SECURITY',n);
  EXECUTE pg_catalog.format('ALTER TABLE vec_autorizacion.%I FORCE ROW LEVEL SECURITY',n);
  EXECUTE pg_catalog.format('CREATE POLICY propietario_exacto ON vec_autorizacion.%I FOR ALL TO vec_autorizacion_propietario USING(current_user=''vec_autorizacion_propietario'') WITH CHECK(current_user=''vec_autorizacion_propietario'')',n);
  EXECUTE pg_catalog.format('CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_autorizacion.%I FOR EACH ROW EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable()',n);
  EXECUTE pg_catalog.format('CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_autorizacion.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable()',n);
  EXECUTE pg_catalog.format('REVOKE ALL ON TABLE vec_autorizacion.%I FROM PUBLIC',n);
  EXECUTE pg_catalog.format('REVOKE ALL ON TYPE vec_autorizacion.%I FROM PUBLIC',n);
 END LOOP;
END $historia$;

-- La huella incluye TODOS los perfiles de ese cargo y ámbito, incluso futuros
-- o revocados. Una alta de otro titular cambia la preimagen. El bloqueo de tabla
-- central coopera también con publicadores históricos que no usan este kit.
CREATE FUNCTION vec_autorizacion.preimagen_cargo_ct_interna_v1(rol_id text,organizacion text,unidad text)
RETURNS text LANGUAGE sql VOLATILE SET search_path=pg_catalog AS $f$
 SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
  'vec.cargos.preimagen.v1:'||COALESCE(pg_catalog.string_agg(pg_catalog.octet_length(a.perfil_activo_ref)::text||':'||a.perfil_activo_ref||pg_catalog.octet_length(a.asignacion_ref)::text||':'||a.asignacion_ref||a.huella_sha256,'' ORDER BY a.perfil_activo_ref COLLATE "C"),'ausente')||pg_catalog.octet_length($1)::text||':'||$1||pg_catalog.octet_length($2)::text||':'||$2||pg_catalog.octet_length($3)::text||':'||$3,'UTF8')),'hex')
 FROM vec_autorizacion.asignacion_perfil_actual x
 JOIN vec_autorizacion.asignacion_perfil a USING(perfil_activo_ref,asignacion_ref)
 JOIN vec_autorizacion.version_rol r USING(version_rol_ref)
 WHERE r.rol_id=$1 AND a.documento->'ambitos' @> pg_catalog.jsonb_build_array(
  pg_catalog.jsonb_build_object('clave','organizacion_ref','valores',pg_catalog.jsonb_build_array(organizacion)),
  pg_catalog.jsonb_build_object('clave','unidad_ref','valores',pg_catalog.jsonb_build_array(unidad)))
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion.preimagen_cargo_ct_interna_v1(text,text,text) FROM PUBLIC;

-- Proyección privada del plan aprobado al propietario CA para su adjunto.
-- El recibo anticipado tiene una referencia fija; no viene del descriptor.
CREATE FUNCTION vec_autorizacion.acreditar_adjunto_plan_cargo_ct_v1(k text,h text,aprobacion_ref text,recibo_ref text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE p record;ap record;x jsonb;a jsonb;
BEGIN
 SELECT * INTO STRICT p FROM vec_autorizacion.cargo_ct_plan WHERE clave=k FOR SHARE;
 SELECT * INTO STRICT ap FROM vec_autorizacion.cargo_ct_aprobacion WHERE clave=k FOR SHARE;
 IF p.plan_sha256 IS DISTINCT FROM h OR ap.plan_sha256 IS DISTINCT FROM h
  OR ap.decision_ref IS DISTINCT FROM aprobacion_ref OR recibo_ref IS DISTINCT FROM 'recibo_cargo_ct:'||k
  OR pg_catalog.clock_timestamp()>=p.caduca_en
  OR vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(ap.decision_canonica,ap.motivo_canonico,ap.persona_version,ap.perfil_version) IS NULL
  OR vec_identidad_sesiones_v1.acreditar_operador_cargos_ct_v1(pg_catalog.convert_from(ap.decision_canonica,'UTF8')::jsonb->'vinculo_autenticacion_actor') IS NOT TRUE
 THEN RETURN NULL; END IF;
 x:=pg_catalog.convert_from(p.plan_canonico,'UTF8')::jsonb;
 a:=pg_catalog.convert_from(pg_catalog.decode(x->>'asignacion_canonica','base64'),'UTF8')::jsonb;
 IF NOT x ? 'vinculo_certificado_canonico' OR NOT EXISTS(
  SELECT 1 FROM vec_autorizacion.asignacion_perfil_actual actual JOIN vec_autorizacion.asignacion_perfil asign USING(perfil_activo_ref,asignacion_ref)
  WHERE actual.perfil_activo_ref=a->>'perfil_activo_ref' AND actual.acto_ref='cargo_ct:'||k
   AND asign.documento=a AND asign.huella_sha256=pg_catalog.encode(pg_catalog.sha256(pg_catalog.decode(x->>'asignacion_canonica','base64')),'hex'))
 THEN RETURN NULL; END IF;
 RETURN pg_catalog.jsonb_build_object('persona_ref',a->>'principal_id','registro_destino_ref',x->>'registro_destino_ref','vinculo_certificado_canonico',x->>'vinculo_certificado_canonico');
EXCEPTION WHEN no_data_found OR too_many_rows OR data_exception THEN RETURN NULL;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.acreditar_adjunto_plan_cargo_ct_v1(text,text,text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion.acreditar_adjunto_plan_cargo_ct_v1(text,text,text,text) TO vec_contexto_actor_v1_propietario;
GRANT USAGE ON SCHEMA vec_autorizacion TO vec_contexto_actor_v1_propietario;

CREATE FUNCTION vec_autorizacion.operar_cargo_ct_interna_v1(op text,b bytea,dc bytea,mc bytea,pv numeric,fv numeric)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
#variable_conflict use_variable
DECLARE p jsonb;a jsonb;ab bytea;d jsonb;v jsonb;h text;clave text;accion text;tipo text;contexto text;
 actor text:='no_acreditado';aud text;fecha timestamptz(6);prev record;rol record;ap record;rec record;
 asignacion_ref text;asignacion_sha text;registrada timestamptz;estado text;resultado jsonb;
BEGIN
 -- Frontera técnica no configurable por el plan. No adopta nombres de LOGIN.
 IF op NOT IN ('preparar','aprobar','aplicar','recuperar') OR op IS NULL
  OR pg_catalog.current_setting('transaction_isolation')<>'serializable'
  OR pg_catalog.current_setting('transaction_read_only')<>'off' OR pg_catalog.current_setting('timezone')<>'UTC'
  OR pg_catalog.current_setting('role')<>'none'
  OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=session_user AND rolcanlogin AND rolinherit
    AND NOT rolsuper AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication AND NOT rolbypassrls AND rolconfig IS NULL)
  OR (SELECT count(*) FROM pg_catalog.pg_auth_members WHERE member=session_user::regrole)<>1
  OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members WHERE member=session_user::regrole
    AND roleid='vec_autorizacion_cargos_ct_ejecutor'::regrole AND inherit_option AND NOT set_option AND NOT admin_option)
  OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_settings WHERE name='statement_timeout' AND unit='ms' AND setting::numeric BETWEEN 1 AND 15000)
  OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_settings WHERE name='idle_in_transaction_session_timeout' AND unit='ms' AND setting::numeric BETWEEN 1 AND 20000)
  OR b IS NULL OR pg_catalog.octet_length(b) NOT BETWEEN 1 AND 32768
  OR dc IS NULL OR pg_catalog.octet_length(dc) NOT BETWEEN 1 AND 524288
  OR mc IS NULL OR pg_catalog.octet_length(mc) NOT BETWEEN 1 AND 65536
  OR pv IS NULL OR fv IS NULL OR pg_catalog.scale(pv)<>0 OR pg_catalog.scale(fv)<>0
  OR pv NOT BETWEEN 1 AND 9007199254740991 OR fv NOT BETWEEN 1 AND 9007199254740991
 THEN RAISE EXCEPTION 'cargo_ct_rechazado' USING ERRCODE='42501'; END IF;
 h:=pg_catalog.encode(pg_catalog.sha256(b),'hex');
 aud:='auditoria_cargo_ct:'||pg_catalog.replace(pg_catalog.gen_random_uuid()::text,'-','');
 BEGIN
  -- Serializa las operaciones de este kit antes de adquirir locks CA/IS/AUT.
  -- Evita convertir dos locks de fila en un upgrade de tabla concurrente.
  PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion:cargos_ct:provision:v1',0));
  p:=pg_catalog.convert_from(b,'UTF8')::jsonb;clave:=p->>'clave';
  IF pg_catalog.jsonb_typeof(p)<>'object' OR (SELECT count(*) FROM pg_catalog.jsonb_object_keys(p))<>(CASE WHEN p ? 'vinculo_certificado_canonico' THEN 14 ELSE 13 END)
   OR (p ? 'vinculo_certificado_canonico' AND (pg_catalog.octet_length(p->>'vinculo_certificado_canonico') NOT BETWEEN 2 AND 16384
      OR pg_catalog.jsonb_typeof((p->>'vinculo_certificado_canonico')::jsonb) IS DISTINCT FROM 'object'))
   OR NOT p ?& ARRAY['version','clave','rol_id','version_rol_ref','version_rol_sha256','control_rol_sha256','preimagen_sha256','registro_destino_ref','destino_sha256','organizacion_ref','unidad_ref','asignacion_canonica','caduca_en']
   OR p->>'version' IS DISTINCT FROM '1' OR clave !~ '^[0-9a-f]{32}$'
   OR p->>'rol_id' NOT IN ('ct_cargo_tecnico_solicitante','ct_cargo_delegacion_solicitante','ct_cargo_direccion_rrhh','ct_cargo_jefatura_servicio_rrhh','ct_cargo_diputacion_delegada_rrhh')
   OR EXISTS(SELECT 1 FROM pg_catalog.jsonb_each(p) e WHERE e.key<>'version' AND pg_catalog.jsonb_typeof(e.value) IS DISTINCT FROM 'string')
   OR EXISTS(SELECT 1 FROM pg_catalog.unnest(ARRAY[p->>'version_rol_sha256',p->>'control_rol_sha256',p->>'preimagen_sha256',p->>'destino_sha256']) x WHERE x !~ '^[0-9a-f]{64}$' OR x=pg_catalog.repeat('0',64))
   OR NOT pg_catalog.isfinite((p->>'caduca_en')::timestamptz)
  THEN RAISE EXCEPTION 'cargo_ct_rechazado' USING ERRCODE='22023'; END IF;
  ab:=pg_catalog.decode(p->>'asignacion_canonica','base64');a:=pg_catalog.convert_from(ab,'UTF8')::jsonb;
  IF pg_catalog.octet_length(ab) NOT BETWEEN 1 AND 16384 OR pg_catalog.jsonb_typeof(a) IS DISTINCT FROM 'object'
   OR (SELECT count(*) FROM pg_catalog.jsonb_object_keys(a))<>12
   OR NOT a ?& ARRAY['asignacion_id','version','perfil_activo_ref','principal_id','version_rol_ref','estado','ambitos','vigente_desde','vigente_hasta','emitida_por','emitida_en','revocada_en']
   OR a->>'revocada_en' IS DISTINCT FROM '0001-01-01T00:00:00Z'
   OR pg_catalog.jsonb_typeof(a->'version') IS DISTINCT FROM 'number'
   OR EXISTS(SELECT 1 FROM pg_catalog.jsonb_each(a) e WHERE e.key NOT IN ('version','ambitos') AND pg_catalog.jsonb_typeof(e.value) IS DISTINCT FROM 'string')
   OR vec_autorizacion.texto_positivo_valido(a->>'asignacion_id',512) IS NOT TRUE
   OR vec_autorizacion.texto_positivo_valido(a->>'principal_id',512) IS NOT TRUE
   OR vec_autorizacion.texto_positivo_valido(a->>'perfil_activo_ref',512) IS NOT TRUE
   OR vec_autorizacion.texto_positivo_valido(a->>'emitida_por',512) IS NOT TRUE
   OR vec_autorizacion.instante_utc_microsegundo_valido(a->>'emitida_en') IS NOT TRUE
   OR vec_autorizacion.instante_utc_microsegundo_valido(a->>'vigente_desde') IS NOT TRUE
   OR vec_autorizacion.instante_utc_microsegundo_valido(a->>'vigente_hasta') IS NOT TRUE
   OR a->>'version_rol_ref' IS DISTINCT FROM p->>'version_rol_ref' OR a->>'estado' IS DISTINCT FROM 'activa'
   OR pg_catalog.jsonb_array_length(a->'ambitos')<>2
   OR a->'ambitos' IS DISTINCT FROM pg_catalog.jsonb_build_array(
    pg_catalog.jsonb_build_object('clave','organizacion_ref','valores',pg_catalog.jsonb_build_array(p->>'organizacion_ref')),
    pg_catalog.jsonb_build_object('clave','unidad_ref','valores',pg_catalog.jsonb_build_array(p->>'unidad_ref')))
   OR (a->>'version')::bigint<1 OR (a->>'vigente_hasta')::timestamptz<=(a->>'vigente_desde')::timestamptz
   OR (a->>'vigente_desde')::timestamptz<(a->>'emitida_en')::timestamptz
  THEN RAISE EXCEPTION 'cargo_ct_rechazado' USING ERRCODE='22023'; END IF;
  IF p ? 'vinculo_certificado_canonico' AND
   (p->>'vinculo_certificado_canonico')::jsonb->>'persona_ref' IS DISTINCT FROM a->>'principal_id'
  THEN RAISE EXCEPTION 'cargo_ct_rechazado' USING ERRCODE='23514'; END IF;
  -- Orden común CA -> IS -> autorización. Las fuentes mantienen sus bloqueos.
  IF vec_contexto_actor_v1.acreditar_destino_cargo_ct_v1(p->>'registro_destino_ref',a->>'principal_id',a->>'perfil_activo_ref') IS DISTINCT FROM p->>'destino_sha256'
  THEN RAISE EXCEPTION 'cargo_ct_rechazado' USING ERRCODE='42501'; END IF;
  d:=pg_catalog.convert_from(dc,'UTF8')::jsonb;v:=d->'vinculo_autenticacion_actor';
  IF vec_identidad_sesiones_v1.acreditar_operador_cargos_ct_v1(v) IS NOT TRUE
  THEN RAISE EXCEPTION 'cargo_ct_rechazado' USING ERRCODE='42501'; END IF;
  accion:=CASE op WHEN 'preparar' THEN 'administracion.perfiles.proponer' WHEN 'aprobar' THEN 'administracion.perfiles.aprobar' WHEN 'aplicar' THEN 'administracion.perfiles.otorgar' ELSE 'administracion.perfiles.recibo.consultar' END;
  tipo:=CASE op WHEN 'aprobar' THEN 'propuesta_perfil' WHEN 'recuperar' THEN 'recibo_perfil' ELSE 'perfil' END;
  contexto:='{"ambitos":{"organizacion_ref":'||vec_autorizacion.texto_json_go_v3(p->>'organizacion_ref')||',"unidad_ref":'||vec_autorizacion.texto_json_go_v3(p->>'unidad_ref')||'},"atributos":{"plan_sha256":'||vec_autorizacion.texto_json_go_v3(h)||'}}';
  IF d->>'accion' IS DISTINCT FROM accion OR d->>'modulo_id' IS DISTINCT FROM 'administracion'
   OR d->>'tipo_recurso' IS DISTINCT FROM tipo OR d->>'recurso_ref' IS DISTINCT FROM 'cargo_ct:'||clave
   OR d->>'finalidad' IS DISTINCT FROM 'gestion_perfiles' OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
   OR d->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb
   OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(contexto,'UTF8')),'hex')
   OR d->>'principal_id' IS NOT DISTINCT FROM a->>'principal_id'
   OR NOT EXISTS(SELECT 1 FROM vec_autorizacion.version_rol r
    JOIN vec_autorizacion.asignacion_perfil asign ON asign.asignacion_ref=d->>'asignacion_ref' AND asign.version_rol_ref=r.version_rol_ref
    WHERE r.version_rol_ref=d->>'version_rol_ref' AND r.rol_id='administracion_perfiles'
     AND pg_catalog.jsonb_array_length(asign.documento->'ambitos')=2
     AND asign.documento->'ambitos' @> pg_catalog.jsonb_build_array(
      pg_catalog.jsonb_build_object('clave','organizacion_ref','valores',pg_catalog.jsonb_build_array(p->>'organizacion_ref')),
      pg_catalog.jsonb_build_object('clave','unidad_ref','valores',pg_catalog.jsonb_build_array(p->>'unidad_ref')))
     AND EXISTS(SELECT 1 FROM pg_catalog.jsonb_array_elements(r.documento->'concesiones') c
      WHERE c->>'accion'=accion AND c->>'modulo_id'='administracion' AND c->>'tipo_recurso'=tipo
       AND c->'finalidades'='["gestion_perfiles"]'::jsonb AND c->>'garantia_minima'='alto'
       AND COALESCE(c->'campos_permitidos','[]'::jsonb)='[]'::jsonb
       AND COALESCE(c->'obligaciones','[]'::jsonb)='[]'::jsonb))
   OR vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(dc,mc,pv,fv) IS NULL
  THEN RAISE EXCEPTION 'cargo_ct_rechazado' USING ERRCODE='42501'; END IF;
  actor:=d->>'principal_id';
  -- Bloquea altas ajenas, incluye ausencia y evita otro titular en mismo ámbito.
  LOCK TABLE vec_autorizacion.asignacion_perfil_actual IN SHARE ROW EXCLUSIVE MODE;
  PERFORM 1 FROM vec_autorizacion.cargo_ct_plan WHERE cargo_ct_plan.clave=clave FOR UPDATE;
  SELECT * INTO rec FROM vec_autorizacion.cargo_ct_recibo WHERE cargo_ct_recibo.clave=clave;
  IF FOUND THEN
   IF rec.plan_sha256<>h OR rec.clave<>clave THEN RAISE EXCEPTION 'cargo_ct_rechazado' USING ERRCODE='23514'; END IF;
   IF op NOT IN ('aplicar','recuperar') THEN RAISE EXCEPTION 'cargo_ct_rechazado' USING ERRCODE='23514'; END IF;
   -- Replay: autoriza ANTES de conocer el recibo y mantiene sus bytes/fecha.
   resultado:=rec.documento;
  ELSE
   IF op='recuperar' THEN RAISE EXCEPTION 'cargo_ct_rechazado' USING ERRCODE='23514'; END IF;
   fecha:=pg_catalog.clock_timestamp();
   IF fecha>=(p->>'caduca_en')::timestamptz OR (p->>'caduca_en')::timestamptz>fecha+interval '24 hours'
   THEN RAISE EXCEPTION 'cargo_ct_rechazado' USING ERRCODE='23514'; END IF;
   SELECT r.*,c.huella_sha256 AS control_sha,c.estado AS control_estado INTO STRICT rol
   FROM vec_autorizacion.version_rol r JOIN vec_autorizacion.control_vigencia_version_rol_actual ca USING(version_rol_ref)
   JOIN vec_autorizacion.control_vigencia_version_rol c USING(version_rol_ref,revision)
   WHERE r.version_rol_ref=p->>'version_rol_ref' FOR SHARE OF ca;
   fecha:=pg_catalog.clock_timestamp();
   IF fecha>=(p->>'caduca_en')::timestamptz THEN RAISE EXCEPTION 'cargo_ct_rechazado' USING ERRCODE='23514'; END IF;
   IF rol.rol_id IS DISTINCT FROM p->>'rol_id' OR rol.huella_sha256 IS DISTINCT FROM p->>'version_rol_sha256'
    OR rol.control_sha IS DISTINCT FROM p->>'control_rol_sha256' OR rol.control_estado IS DISTINCT FROM 'habilitada'
    OR rol.documento->>'estado' IS DISTINCT FROM 'publicada' OR rol.publicada_en>fecha
    OR vec_autorizacion.preimagen_cargo_ct_interna_v1(p->>'rol_id',p->>'organizacion_ref',p->>'unidad_ref') IS DISTINCT FROM p->>'preimagen_sha256'
   THEN RAISE EXCEPTION 'cargo_ct_rechazado' USING ERRCODE='23514'; END IF;
   SELECT * INTO prev FROM vec_autorizacion.cargo_ct_plan WHERE cargo_ct_plan.clave=clave;
   IF op='preparar' THEN
    IF FOUND AND (prev.plan_sha256<>h OR prev.proponente_ref<>actor) THEN RAISE EXCEPTION 'cargo_ct_rechazado' USING ERRCODE='23514'; END IF;
    IF NOT FOUND THEN INSERT INTO vec_autorizacion.cargo_ct_plan VALUES(clave,b,h,actor,d->>'decision_ref',fecha,(p->>'caduca_en')::timestamptz); END IF;
    estado:='preparado';
   ELSE
    IF NOT FOUND OR prev.plan_canonico<>b OR prev.plan_sha256<>h THEN RAISE EXCEPTION 'cargo_ct_rechazado' USING ERRCODE='23514'; END IF;
    SELECT * INTO ap FROM vec_autorizacion.cargo_ct_aprobacion WHERE cargo_ct_aprobacion.clave=clave;
    IF op='aprobar' THEN
     IF FOUND AND (ap.plan_sha256<>h OR ap.aprobador_ref<>actor) THEN RAISE EXCEPTION 'cargo_ct_rechazado' USING ERRCODE='23514'; END IF;
     IF NOT FOUND THEN INSERT INTO vec_autorizacion.cargo_ct_aprobacion VALUES(clave,h,actor,d->>'decision_ref',dc,mc,pv,fv,fecha); END IF;
     estado:='aprobado';
    ELSE
     IF NOT FOUND OR ap.plan_sha256<>h
      OR vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(ap.decision_canonica,ap.motivo_canonico,ap.persona_version,ap.perfil_version) IS NULL
      OR vec_identidad_sesiones_v1.acreditar_operador_cargos_ct_v1(pg_catalog.convert_from(ap.decision_canonica,'UTF8')::jsonb->'vinculo_autenticacion_actor') IS NOT TRUE
     THEN RAISE EXCEPTION 'cargo_ct_rechazado' USING ERRCODE='42501'; END IF;
     IF EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil_actual x JOIN vec_autorizacion.asignacion_perfil apf USING(perfil_activo_ref,asignacion_ref)
      JOIN vec_autorizacion.version_rol r USING(version_rol_ref) WHERE r.rol_id=p->>'rol_id' AND apf.documento->>'estado'='activa'
       AND apf.documento->'ambitos'=a->'ambitos' AND apf.perfil_activo_ref<>a->>'perfil_activo_ref')
     THEN RAISE EXCEPTION 'cargo_ct_rechazado' USING ERRCODE='23514'; END IF;
     SELECT apf.* INTO prev FROM vec_autorizacion.asignacion_perfil_actual x JOIN vec_autorizacion.asignacion_perfil apf USING(perfil_activo_ref,asignacion_ref)
     WHERE x.perfil_activo_ref=a->>'perfil_activo_ref';
     IF FOUND THEN
      IF prev.documento->>'estado' IS DISTINCT FROM 'activa'
       OR prev.principal_id<>a->>'principal_id' OR prev.asignacion_id<>a->>'asignacion_id'
       OR prev.version+1<>(a->>'version')::bigint OR prev.version_rol_ref<>a->>'version_rol_ref'
      THEN RAISE EXCEPTION 'cargo_ct_rechazado' USING ERRCODE='23514'; END IF;
     ELSIF (a->>'version')::bigint<>1 THEN RAISE EXCEPTION 'cargo_ct_rechazado' USING ERRCODE='23514'; END IF;
     IF a->>'emitida_por' IS DISTINCT FROM actor OR (a->>'emitida_en')::timestamptz>fecha OR fecha>=(a->>'vigente_hasta')::timestamptz THEN RAISE EXCEPTION 'cargo_ct_rechazado' USING ERRCODE='23514'; END IF;
     asignacion_ref:='asignacion:'||(a->>'asignacion_id')||':v'||(a->>'version');
     asignacion_sha:=pg_catalog.encode(pg_catalog.sha256(ab),'hex');
     INSERT INTO vec_autorizacion.asignacion_perfil(asignacion_ref,asignacion_id,version,perfil_activo_ref,principal_id,version_rol_ref,huella_sha256,emitida_en,documento)
     VALUES(asignacion_ref,a->>'asignacion_id',(a->>'version')::bigint,a->>'perfil_activo_ref',a->>'principal_id',a->>'version_rol_ref',asignacion_sha,(a->>'emitida_en')::timestamptz,a);
     INSERT INTO vec_autorizacion.asignacion_perfil_actual AS actual(perfil_activo_ref,asignacion_ref,actualizada_en,actualizada_por,acto_ref)
     VALUES(a->>'perfil_activo_ref',asignacion_ref,fecha,actor,'cargo_ct:'||clave)
     ON CONFLICT(perfil_activo_ref) DO UPDATE SET asignacion_ref=EXCLUDED.asignacion_ref,actualizada_en=EXCLUDED.actualizada_en,actualizada_por=EXCLUDED.actualizada_por,acto_ref=EXCLUDED.acto_ref
     WHERE actual.asignacion_ref=prev.asignacion_ref;
     IF NOT FOUND THEN RAISE EXCEPTION 'cargo_ct_rechazado' USING ERRCODE='40001'; END IF;
     IF p ? 'vinculo_certificado_canonico' THEN
      IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_proc f WHERE f.oid=pg_catalog.to_regprocedure('vec_contexto_actor_v1.registrar_vinculo_certificado_aprobado_ct_v1(bytea,text,text,text,text)')
       AND f.proowner='vec_contexto_actor_v1_propietario'::regrole AND f.prosecdef AND f.provolatile='v' AND f.prorettype='boolean'::regtype
       AND f.proconfig @> ARRAY['search_path=pg_catalog'])
      THEN RAISE EXCEPTION 'cargo_ct_rechazado' USING ERRCODE='42501'; END IF;
      IF vec_contexto_actor_v1.registrar_vinculo_certificado_aprobado_ct_v1(
       pg_catalog.convert_to(p->>'vinculo_certificado_canonico','UTF8'),clave,h,ap.decision_ref,'recibo_cargo_ct:'||clave) IS NOT TRUE
      THEN RAISE EXCEPTION 'cargo_ct_rechazado' USING ERRCODE='42501'; END IF;
     END IF;
     estado:='publicado';
    END IF;
   END IF;
   resultado:=pg_catalog.jsonb_build_object('estado',estado,'clave',clave,'plan_sha256',h,'auditoria_ref',aud,'fecha',fecha);
   IF op='aplicar' THEN
    resultado:=resultado||pg_catalog.jsonb_build_object('recibo_ref','recibo_cargo_ct:'||clave,'version',(a->>'version')::bigint);
   END IF;
  END IF;
  -- Releer el reloj y autoridad después de todos los bloqueos y antes de
  -- confirmar incluso un replay. Un vencimiento revierte la subtransacción.
  IF vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(dc,mc,pv,fv) IS NULL
   OR vec_identidad_sesiones_v1.acreditar_operador_cargos_ct_v1(v) IS NOT TRUE
   OR vec_contexto_actor_v1.acreditar_destino_cargo_ct_v1(p->>'registro_destino_ref',a->>'principal_id',a->>'perfil_activo_ref') IS DISTINCT FROM p->>'destino_sha256'
  THEN RAISE EXCEPTION 'cargo_ct_rechazado' USING ERRCODE='42501'; END IF;
  fecha:=pg_catalog.clock_timestamp();
  IF EXISTS(SELECT 1 FROM vec_autorizacion.cargo_ct_consumo c WHERE c.decision_ref=d->>'decision_ref' AND (c.clave<>clave OR c.operacion<>op))
  THEN RAISE EXCEPTION 'cargo_ct_rechazado' USING ERRCODE='23514'; END IF;
  INSERT INTO vec_autorizacion.cargo_ct_consumo VALUES(d->>'decision_ref',clave,op,fecha) ON CONFLICT(decision_ref) DO NOTHING;
  INSERT INTO vec_autorizacion.cargo_ct_auditoria VALUES(aud,clave,h,op,actor,'confirmada',fecha);
  IF op='aplicar' AND estado='publicado' THEN
   INSERT INTO vec_autorizacion.cargo_ct_recibo VALUES(clave,resultado->>'recibo_ref',h,asignacion_ref,aud,resultado,(resultado->>'fecha')::timestamptz);
  END IF;
  RETURN resultado;
 EXCEPTION WHEN insufficient_privilege OR check_violation OR invalid_parameter_value OR data_exception OR no_data_found OR too_many_rows OR unique_violation THEN
  -- La subtransacción revierte negocio/consumo. Este intento queda en la
  -- autoridad segregada y se confirma antes de devolver rechazo al operador.
  fecha:=pg_catalog.clock_timestamp();
  INSERT INTO vec_autorizacion.cargo_ct_auditoria VALUES(aud,NULL,h,op,actor,'denegada',fecha);
  RETURN pg_catalog.jsonb_build_object('estado','denegado','plan_sha256',h,'auditoria_ref',aud,'fecha',fecha);
 END;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.operar_cargo_ct_interna_v1(text,bytea,bytea,bytea,numeric,numeric) FROM PUBLIC;
DO $fachadas$
DECLARE n text;
BEGIN
 FOREACH n IN ARRAY ARRAY['preparar','aprobar','aplicar','recuperar'] LOOP
  EXECUTE pg_catalog.format('CREATE FUNCTION vec_autorizacion.%I(b bytea,d bytea,m bytea,pv numeric,fv numeric) RETURNS jsonb LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $fn$ SELECT vec_autorizacion.operar_cargo_ct_interna_v1(%L,b,d,m,pv,fv) $fn$',n||'_plan_cargo_ct_v1',n);
  EXECUTE pg_catalog.format('REVOKE ALL ON FUNCTION vec_autorizacion.%I(bytea,bytea,bytea,numeric,numeric) FROM PUBLIC',n||'_plan_cargo_ct_v1');
  EXECUTE pg_catalog.format('GRANT EXECUTE ON FUNCTION vec_autorizacion.%I(bytea,bytea,bytea,numeric,numeric) TO vec_autorizacion_cargos_ct_ejecutor',n||'_plan_cargo_ct_v1');
 END LOOP;
END $fachadas$;
GRANT USAGE ON SCHEMA vec_autorizacion TO vec_autorizacion_cargos_ct_ejecutor;
COMMIT;
