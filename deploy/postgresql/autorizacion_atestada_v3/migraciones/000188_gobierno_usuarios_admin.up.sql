\set ON_ERROR_STOP on
-- AD188: dos capacidades ADMIN, sin autoridad nominal ni raíz nueva.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000188',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
DO $pre$
DECLARE actual text;
BEGIN
 SELECT encode(sha256(convert_to(pg_get_constraintdef(c.oid,false),'UTF8')),'hex') INTO actual
 FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
 AND c.conname='auditoria_tipo_disjunto_v4' AND c.convalidated;
 IF actual IS DISTINCT FROM '97d754fef3d60b4799f82fac0d09417133f4caa5b25e8f2bcba005fb1f72a09f'
 THEN RAISE EXCEPTION 'AD188: PARO clave=CHECK_sha actual=% esperado=97d754fef3d60b4799f82fac0d09417133f4caa5b25e8f2bcba005fb1f72a09f',coalesce(actual,'ausente') USING ERRCODE='55000'; END IF;
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999 OR to_regrole('vec_gobierno_usuarios_admin_operador') IS NOT NULL
 THEN RAISE EXCEPTION 'AD188: PARO clave=preimagen actual=incompatible esperado=POST187_sin188' USING ERRCODE='55000';END IF;
END $pre$;
CREATE ROLE vec_gobierno_usuarios_admin_operador NOLOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
DO $connect$ BEGIN EXECUTE format('GRANT CONNECT ON DATABASE %I TO vec_gobierno_usuarios_admin_operador',current_database()); END $connect$;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
LOCK TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 IN ACCESS EXCLUSIVE MODE;
ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 ADD COLUMN gobierno_usuarios_detalle jsonb, ADD COLUMN gobierno_usuarios_solicitud_sha256 text;
DO $familia$
DECLARE anterior text;nuevo text;
BEGIN
 SELECT pg_get_constraintdef(c.oid,false) INTO STRICT anterior FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass AND c.conname='auditoria_tipo_disjunto_v4';
 nuevo:='CHECK ((gobierno_usuarios_detalle IS NULL AND gobierno_usuarios_solicitud_sha256 IS NULL AND ('||substr(anterior,8,length(anterior)-8)||')) OR ('||$tipo$
 tipo_registro IN ('gobierno_usuarios_admin','intento_gobierno_usuarios_admin')
 AND decision_ref IS NULL AND efecto_ref IS NULL AND huella_efecto_sha256 IS NULL AND version_consumo IS NULL
 AND intento_ref IS NULL AND intento_material_sha256 IS NULL AND actor_ref IS NULL AND perfil_activo_ref IS NULL
 AND registro_contexto_ref IS NULL AND contexto_sha256 IS NULL AND procedencia_sha256 IS NULL
 AND autenticacion_ref IS NULL AND sesion_ref IS NULL AND autenticacion_sha256 IS NULL AND vinculo_sha256 IS NULL
 AND fuente_ref IS NULL AND fuente_sha256 IS NULL AND aprobacion_ref IS NULL
 AND fuentes_plan_ref IS NULL AND fuentes_preimagen_sha256 IS NULL AND fuentes_configuracion_sha256 IS NULL AND fuentes_alcance IS NULL AND fuentes_solicitud_sha256 IS NULL
 AND unidad_plan_ref IS NULL AND unidad_preimagen_sha256 IS NULL AND unidad_configuracion_sha256 IS NULL AND unidad_alcance IS NULL
 AND unidad_recibo_ref IS NULL AND unidad_recibo_sha256 IS NULL AND unidad_solicitud_sha256 IS NULL AND bootstrap_solicitud_sha256 IS NULL
 AND mantenimiento_detalle IS NULL AND mantenimiento_solicitud_sha256 IS NULL AND periodica_detalle IS NULL AND preservacion_detalle IS NULL
 AND evento_ref IS NOT NULL AND evento_material_sha256 IS NOT NULL AND operador_login IS NOT NULL
 AND accion IS NOT DISTINCT FROM 'aprovisionar_gobierno_usuarios_admin_v1' AND modulo_id IS NOT DISTINCT FROM 'administracion'
 AND proceso IS NOT DISTINCT FROM 'postgresql' AND canal IS NOT DISTINCT FROM 'operacion_tecnica_privada'
 AND finalidad_ref IS NOT DISTINCT FROM 'gobierno_usuarios_admin' AND recurso_ref IS NOT NULL AND correlacion_ref IS NOT NULL
 AND resultado IS NOT NULL AND motivo_ref IS NOT NULL
 AND ((tipo_registro='gobierno_usuarios_admin' AND plan_sha256 IS NOT NULL AND gobierno_usuarios_detalle IS NOT NULL AND gobierno_usuarios_solicitud_sha256 IS NULL
 AND resultado IS NOT DISTINCT FROM 'permitido' AND motivo_ref IS NOT DISTINCT FROM 'gobierno_usuarios_registrado' AND recurso_ref IS NOT DISTINCT FROM 'gobierno_usuarios:'||substr(plan_sha256,1,32))
 OR (tipo_registro='intento_gobierno_usuarios_admin' AND plan_sha256 IS NULL AND gobierno_usuarios_detalle IS NULL AND gobierno_usuarios_solicitud_sha256 IS NOT NULL))
 $tipo$||'))';
 ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 DROP CONSTRAINT auditoria_tipo_disjunto_v4;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 ADD CONSTRAINT auditoria_tipo_disjunto_v4 '||nuevo;
END $familia$;
CREATE FUNCTION vec_autorizacion_atestada_v3.detalle_gobierno_usuarios_valido_v1(p jsonb)
RETURNS boolean LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog AS $f$
DECLARE claves constant text[]:=ARRAY['preimagen_sha256','configuracion_origen_ref','configuracion_destino_ref','claves_sha256'];k text;
BEGIN
 IF jsonb_typeof(p) IS DISTINCT FROM 'object' OR NOT p ?& claves OR (SELECT count(*) FROM jsonb_object_keys(p))<>cardinality(claves) THEN RETURN false; END IF;
 FOREACH k IN ARRAY claves LOOP
  IF jsonb_typeof(p->k) IS DISTINCT FROM 'string' THEN RETURN false; END IF;
  IF right(k,7)='_sha256' THEN
   IF p->>k !~ '^[0-9a-f]{64}$' THEN RETURN false; END IF;
  ELSIF p->>k !~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,199}$' THEN RETURN false; END IF;
 END LOOP;
 RETURN true;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.detalle_gobierno_usuarios_valido_v1(jsonb) FROM PUBLIC;
CREATE FUNCTION vec_autorizacion_atestada_v3.motivo_intento_gobierno_usuarios_valido_v1(p_resultado text,p_motivo text)
RETURNS boolean LANGUAGE sql IMMUTABLE STRICT PARALLEL SAFE SET search_path=pg_catalog AS $f$
 SELECT EXISTS(SELECT 1 FROM (VALUES ('permitido','gobierno_usuarios_registrado'),('permitido','gobierno_usuarios_replay'),
 ('denegado','gobierno_usuarios_denegado'),('error','gobierno_usuarios_error')) AS m(resultado,motivo_ref)
 WHERE m.resultado=p_resultado AND m.motivo_ref=p_motivo)
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.motivo_intento_gobierno_usuarios_valido_v1(text,text) FROM PUBLIC;
ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 ADD CONSTRAINT auditoria_gobierno_usuarios_formato_v1 CHECK (
 tipo_registro NOT IN('gobierno_usuarios_admin','intento_gobierno_usuarios_admin') OR (
 evento_ref ~ '^evento_[0-9a-f]{32}$' AND evento_material_sha256 ~ '^[0-9a-f]{64}$' AND correlacion_ref ~ '^correlacion_[0-9a-f]{32}$'
 AND ((tipo_registro='gobierno_usuarios_admin' AND plan_sha256 ~ '^[0-9a-f]{64}$' AND vec_autorizacion_atestada_v3.detalle_gobierno_usuarios_valido_v1(gobierno_usuarios_detalle))
 OR (tipo_registro='intento_gobierno_usuarios_admin' AND gobierno_usuarios_solicitud_sha256 ~ '^[0-9a-f]{64}$'
  AND recurso_ref ~ '^solicitud_gobierno_usuarios:[0-9a-f]{32}$' AND vec_autorizacion_atestada_v3.motivo_intento_gobierno_usuarios_valido_v1(resultado,motivo_ref)))));
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_gobierno_usuarios_admin_v1(p_evento jsonb)
RETURNS TABLE(auditoria_ref text,secuencia numeric,huella_sha256 text,correlacion_ref text,registrada_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' SET statement_timeout='10s'
AS $funcion$
DECLARE
 v_orden constant text[]:=ARRAY['tipo_registro','evento_ref','operador_login','plan_sha256','preimagen_sha256','configuracion_origen_ref','configuracion_destino_ref','claves_sha256','proceso','canal','finalidad_ref','correlacion_ref'];
 v_claves text[];v_clave text;v_material bytea;v_material_sha text;v_anterior text;
 v_secuencia numeric;v_instante timestamptz(6);v_ref text;v_huella text;v_existente record;
BEGIN
 IF pg_catalog.current_setting('transaction_isolation')<>'serializable'
 OR pg_catalog.current_setting('transaction_read_only')<>'off' OR pg_catalog.current_setting('TimeZone')<>'UTC'
 THEN RAISE EXCEPTION 'AD188: PARO clave=transaccion actual=incompatible esperado=serializable_rw_UTC' USING ERRCODE='25000'; END IF;
 IF pg_catalog.jsonb_typeof(p_evento) IS DISTINCT FROM 'object' OR pg_catalog.octet_length(p_evento::text)>8192
 THEN RAISE EXCEPTION 'AD188: PARO clave=evento actual=incompatible esperado=objeto_acotado' USING ERRCODE='22023'; END IF;
 SELECT pg_catalog.array_agg(k ORDER BY k COLLATE "C") INTO v_claves FROM pg_catalog.jsonb_object_keys(p_evento) k;
 IF v_claves IS DISTINCT FROM (SELECT pg_catalog.array_agg(k ORDER BY k COLLATE "C") FROM pg_catalog.unnest(v_orden) k)
 THEN RAISE EXCEPTION 'AD188: PARO clave=campos actual=incompatible esperado=ABI_exacto' USING ERRCODE='22023'; END IF;
 FOREACH v_clave IN ARRAY v_orden LOOP
  IF pg_catalog.jsonb_typeof(p_evento->v_clave) IS DISTINCT FROM 'string'
  OR pg_catalog.octet_length(p_evento->>v_clave) NOT BETWEEN 1 AND 200
  THEN RAISE EXCEPTION 'AD188: PARO clave=valor actual=incompatible esperado=string_acotado' USING ERRCODE='22023'; END IF;
 END LOOP;
 IF p_evento->>'tipo_registro' IS DISTINCT FROM 'gobierno_usuarios_admin'
 OR p_evento->>'operador_login' IS DISTINCT FROM session_user::text
 OR pg_catalog.octet_length(p_evento->>'operador_login')>63
 OR p_evento->>'evento_ref' !~ '^evento_[0-9a-f]{32}$'
 OR p_evento->>'plan_sha256' !~ '^[0-9a-f]{64}$'
 OR vec_autorizacion_atestada_v3.detalle_gobierno_usuarios_valido_v1(p_evento-ARRAY['tipo_registro','evento_ref','operador_login','plan_sha256','proceso','canal','finalidad_ref','correlacion_ref']) IS DISTINCT FROM true
 OR p_evento->>'proceso' IS DISTINCT FROM 'postgresql'
 OR p_evento->>'canal' IS DISTINCT FROM 'operacion_tecnica_privada'
 OR p_evento->>'finalidad_ref' IS DISTINCT FROM 'gobierno_usuarios_admin'
 OR p_evento->>'correlacion_ref' !~ '^correlacion_[0-9a-f]{32}$'
 THEN RAISE EXCEPTION 'AD188: PARO clave=semantica actual=incompatible esperado=gobierno_usuarios_privado' USING ERRCODE='22023'; END IF;
 v_material:=vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.gobierno-usuarios-admin.v1');
 FOREACH v_clave IN ARRAY v_orden LOOP
  v_material:=v_material||vec_autorizacion_atestada_v3.encuadrar_mac(p_evento->>v_clave);
 END LOOP;
 v_material_sha:=pg_catalog.encode(pg_catalog.sha256(v_material),'hex');
 -- Mismo espacio de cerrojo y misma unicidad que los eventos AD171.
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
  'vec_autorizacion_atestada_v3:evento-admin:'||(p_evento->>'evento_ref'),0));
 SELECT a.tipo_registro,a.auditoria_ref,a.secuencia,a.huella_sha256,a.correlacion_ref,a.registrada_en,
  a.evento_material_sha256 INTO v_existente
 FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 a WHERE a.evento_ref=p_evento->>'evento_ref';
 IF FOUND THEN
  IF v_existente.tipo_registro IS DISTINCT FROM 'gobierno_usuarios_admin'
  OR v_existente.evento_material_sha256 IS DISTINCT FROM v_material_sha
  THEN RAISE EXCEPTION 'AD188: PARO clave=replay actual=material_distinto esperado=material_original' USING ERRCODE='23505'; END IF;
  RETURN QUERY SELECT v_existente.auditoria_ref,v_existente.secuencia,v_existente.huella_sha256,
   v_existente.correlacion_ref,v_existente.registrada_en;
  RETURN;
 END IF;
 SELECT c.secuencia,c.cabeza_sha256 INTO STRICT v_secuencia,v_anterior
 FROM vec_autorizacion_atestada_v3.control_cadena_auditoria c WHERE c.control_id FOR UPDATE;
 IF v_secuencia>=9007199254740991::numeric
 THEN RAISE EXCEPTION 'AD188: PARO clave=secuencia actual=limite esperado=entero_seguro' USING ERRCODE='22003'; END IF;
 v_secuencia:=v_secuencia+1;v_instante:=pg_catalog.clock_timestamp();
 v_ref:='aud_v3_gu_'||pg_catalog.substr(p_evento->>'evento_ref',8,32);
 v_huella:=pg_catalog.encode(pg_catalog.sha256(
  vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.eslabon.gobierno-usuarios-admin.v1')||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_secuencia::text)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_anterior)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_ref)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_material_sha)||
  vec_autorizacion_atestada_v3.encuadrar_mac(pg_catalog.to_char(v_instante AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))),'hex');
 INSERT INTO vec_autorizacion_atestada_v3.auditoria_consumo_v3(
  auditoria_ref,secuencia,anterior_sha256,huella_sha256,registrada_en,tipo_registro,
  evento_ref,evento_material_sha256,operador_login,plan_sha256,gobierno_usuarios_detalle,
  accion,modulo_id,recurso_ref,finalidad_ref,resultado,motivo_ref,proceso,canal,correlacion_ref)
 VALUES(v_ref,v_secuencia,v_anterior,v_huella,v_instante,'gobierno_usuarios_admin',
  p_evento->>'evento_ref',v_material_sha,(p_evento->>'operador_login')::name,p_evento->>'plan_sha256',
  p_evento-ARRAY['tipo_registro','evento_ref','operador_login','plan_sha256','proceso','canal','finalidad_ref','correlacion_ref'],
  'aprovisionar_gobierno_usuarios_admin_v1','administracion','gobierno_usuarios:'||pg_catalog.substr(p_evento->>'plan_sha256',1,32),
  'gobierno_usuarios_admin','permitido','gobierno_usuarios_registrado',p_evento->>'proceso',p_evento->>'canal',p_evento->>'correlacion_ref');
 UPDATE vec_autorizacion_atestada_v3.control_cadena_auditoria
 SET secuencia=v_secuencia,cabeza_sha256=v_huella,actualizada_en=v_instante WHERE control_id;
 RETURN QUERY SELECT v_ref,v_secuencia,v_huella,p_evento->>'correlacion_ref',v_instante;
END $funcion$;
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_intento_gobierno_usuarios_admin_v1(p_evento jsonb)
RETURNS TABLE(auditoria_ref text,secuencia numeric,huella_sha256 text,correlacion_ref text,registrada_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' SET statement_timeout='10s'
AS $funcion$
DECLARE
 v_orden constant text[]:=ARRAY['tipo_registro','evento_ref','operador_login','solicitud_sha256',
  'accion','recurso_ref','resultado','motivo_ref','proceso','canal','finalidad_ref','correlacion_ref'];
 v_claves text[];v_clave text;v_material bytea;v_material_sha text;v_anterior text;
 v_secuencia numeric;v_instante timestamptz(6);v_ref text;v_huella text;v_existente record;
BEGIN
 IF pg_catalog.current_setting('transaction_isolation')<>'serializable'
 OR pg_catalog.current_setting('transaction_read_only')<>'off' OR pg_catalog.current_setting('TimeZone')<>'UTC'
 THEN RAISE EXCEPTION 'AD188: PARO clave=transaccion actual=incompatible esperado=serializable_rw_UTC' USING ERRCODE='25000'; END IF;
 IF pg_catalog.jsonb_typeof(p_evento) IS DISTINCT FROM 'object' OR pg_catalog.octet_length(p_evento::text)>8192
 THEN RAISE EXCEPTION 'AD188: PARO clave=evento actual=incompatible esperado=objeto_acotado' USING ERRCODE='22023'; END IF;
 SELECT pg_catalog.array_agg(k ORDER BY k COLLATE "C") INTO v_claves FROM pg_catalog.jsonb_object_keys(p_evento) k;
 IF v_claves IS DISTINCT FROM (SELECT pg_catalog.array_agg(k ORDER BY k COLLATE "C") FROM pg_catalog.unnest(v_orden) k)
 THEN RAISE EXCEPTION 'AD188: PARO clave=campos actual=incompatible esperado=ABI_exacto' USING ERRCODE='22023'; END IF;
 FOREACH v_clave IN ARRAY v_orden LOOP
  IF pg_catalog.jsonb_typeof(p_evento->v_clave) IS DISTINCT FROM 'string'
  OR pg_catalog.octet_length(p_evento->>v_clave) NOT BETWEEN 1 AND 200
  THEN RAISE EXCEPTION 'AD188: PARO clave=valor actual=incompatible esperado=string_acotado' USING ERRCODE='22023'; END IF;
 END LOOP;
 IF p_evento->>'tipo_registro' IS DISTINCT FROM 'intento_gobierno_usuarios_admin'
 OR p_evento->>'operador_login' IS DISTINCT FROM session_user::text
 OR pg_catalog.octet_length(p_evento->>'operador_login')>63
 OR p_evento->>'evento_ref' !~ '^evento_[0-9a-f]{32}$'
 OR p_evento->>'solicitud_sha256' !~ '^[0-9a-f]{64}$'
 OR p_evento->>'accion' IS DISTINCT FROM 'aprovisionar_gobierno_usuarios_admin_v1'
 OR p_evento->>'recurso_ref' !~ '^solicitud_gobierno_usuarios:[0-9a-f]{32}$'
 OR vec_autorizacion_atestada_v3.motivo_intento_gobierno_usuarios_valido_v1(p_evento->>'resultado',p_evento->>'motivo_ref') IS DISTINCT FROM true
 OR p_evento->>'proceso' IS DISTINCT FROM 'postgresql'
 OR p_evento->>'canal' IS DISTINCT FROM 'operacion_tecnica_privada'
 OR p_evento->>'finalidad_ref' IS DISTINCT FROM 'gobierno_usuarios_admin'
 OR p_evento->>'correlacion_ref' !~ '^correlacion_[0-9a-f]{32}$'
 THEN RAISE EXCEPTION 'AD188: PARO clave=semantica actual=incompatible esperado=gobierno_usuarios_privado' USING ERRCODE='22023'; END IF;
 v_material:=vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.intento-gobierno-usuarios-admin.v1');
 FOREACH v_clave IN ARRAY v_orden LOOP
  v_material:=v_material||vec_autorizacion_atestada_v3.encuadrar_mac(p_evento->>v_clave);
 END LOOP;
 v_material_sha:=pg_catalog.encode(pg_catalog.sha256(v_material),'hex');
 -- Mismo espacio de cerrojo y misma unicidad que los eventos AD171.
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
  'vec_autorizacion_atestada_v3:evento-admin:'||(p_evento->>'evento_ref'),0));
 SELECT a.tipo_registro,a.auditoria_ref,a.secuencia,a.huella_sha256,a.correlacion_ref,a.registrada_en,
  a.evento_material_sha256 INTO v_existente
 FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 a WHERE a.evento_ref=p_evento->>'evento_ref';
 IF FOUND THEN
  IF v_existente.tipo_registro IS DISTINCT FROM 'intento_gobierno_usuarios_admin'
  OR v_existente.evento_material_sha256 IS DISTINCT FROM v_material_sha
  THEN RAISE EXCEPTION 'AD188: PARO clave=replay actual=material_distinto esperado=material_original' USING ERRCODE='23505'; END IF;
  RETURN QUERY SELECT v_existente.auditoria_ref,v_existente.secuencia,v_existente.huella_sha256,
   v_existente.correlacion_ref,v_existente.registrada_en;
  RETURN;
 END IF;
 SELECT c.secuencia,c.cabeza_sha256 INTO STRICT v_secuencia,v_anterior
 FROM vec_autorizacion_atestada_v3.control_cadena_auditoria c WHERE c.control_id FOR UPDATE;
 IF v_secuencia>=9007199254740991::numeric
 THEN RAISE EXCEPTION 'AD188: PARO clave=secuencia actual=limite esperado=entero_seguro' USING ERRCODE='22003'; END IF;
 v_secuencia:=v_secuencia+1;v_instante:=pg_catalog.clock_timestamp();
 v_ref:='aud_v3_gui_'||pg_catalog.substr(p_evento->>'evento_ref',8,32);
 v_huella:=pg_catalog.encode(pg_catalog.sha256(
  vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.eslabon.intento-gobierno-usuarios-admin.v1')||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_secuencia::text)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_anterior)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_ref)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_material_sha)||
  vec_autorizacion_atestada_v3.encuadrar_mac(pg_catalog.to_char(v_instante AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))),'hex');
 INSERT INTO vec_autorizacion_atestada_v3.auditoria_consumo_v3(
  auditoria_ref,secuencia,anterior_sha256,huella_sha256,registrada_en,tipo_registro,
  evento_ref,evento_material_sha256,operador_login,gobierno_usuarios_solicitud_sha256,
  accion,modulo_id,recurso_ref,finalidad_ref,resultado,motivo_ref,proceso,canal,correlacion_ref)
 VALUES(v_ref,v_secuencia,v_anterior,v_huella,v_instante,'intento_gobierno_usuarios_admin',
  p_evento->>'evento_ref',v_material_sha,(p_evento->>'operador_login')::name,p_evento->>'solicitud_sha256',
  p_evento->>'accion','administracion',p_evento->>'recurso_ref',p_evento->>'finalidad_ref',
  p_evento->>'resultado',p_evento->>'motivo_ref',p_evento->>'proceso',p_evento->>'canal',p_evento->>'correlacion_ref');
 UPDATE vec_autorizacion_atestada_v3.control_cadena_auditoria
 SET secuencia=v_secuencia,cabeza_sha256=v_huella,actualizada_en=v_instante WHERE control_id;
 RETURN QUERY SELECT v_ref,v_secuencia,v_huella,p_evento->>'correlacion_ref',v_instante;
END $funcion$;

REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_gobierno_usuarios_admin_v1(jsonb) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_intento_gobierno_usuarios_admin_v1(jsonb) FROM PUBLIC;

-- Fachadas de publicación y configuración privada a continuación.
CREATE TABLE vec_autorizacion_atestada_v3.config_gobierno_usuarios_admin_v1(
 identidad_login name PRIMARY KEY,
 plan_sha256 text NOT NULL CHECK(plan_sha256 ~ '^[0-9a-f]{64}$'),
 preimagen_sha256 text NOT NULL CHECK(preimagen_sha256 ~ '^[0-9a-f]{64}$'),
 material_sha256 text NOT NULL CHECK(material_sha256 ~ '^[0-9a-f]{64}$'),
 vigente_desde timestamptz NOT NULL,vigente_hasta timestamptz NOT NULL CHECK(vigente_hasta>vigente_desde),
 entorno text NOT NULL CHECK(entorno='desarrollo'));
REVOKE ALL ON TABLE vec_autorizacion_atestada_v3.config_gobierno_usuarios_admin_v1 FROM PUBLIC,vec_gobierno_usuarios_admin_operador;
CREATE FUNCTION vec_autorizacion_atestada_v3.exigir_operador_gobierno_usuarios_admin_v1()
RETURNS vec_autorizacion_atestada_v3.config_gobierno_usuarios_admin_v1 LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE c vec_autorizacion_atestada_v3.config_gobierno_usuarios_admin_v1;l record;g oid;
BEGIN
 g:='vec_gobierno_usuarios_admin_operador'::regrole;
 SELECT * INTO STRICT l FROM pg_roles WHERE rolname=session_user;
 IF NOT l.rolcanlogin OR NOT l.rolinherit OR l.rolsuper OR l.rolcreatedb OR l.rolcreaterole OR l.rolreplication OR l.rolbypassrls
 OR NOT pg_has_role(session_user,g,'MEMBER')
 OR EXISTS(WITH RECURSIVE m(oid) AS(SELECT roleid FROM pg_auth_members WHERE member=l.oid UNION SELECT a.roleid FROM pg_auth_members a JOIN m ON a.member=m.oid) SELECT 1 FROM m WHERE oid<>g)
 OR EXISTS(SELECT 1 FROM pg_auth_members WHERE member=l.oid AND (admin_option OR set_option OR NOT inherit_option))
 OR EXISTS(SELECT 1 FROM pg_shdepend d WHERE d.refclassid='pg_authid'::regclass AND d.refobjid=l.oid AND d.deptype IN('a','o'))
 OR EXISTS(SELECT 1 FROM pg_shdepend d WHERE d.refclassid='pg_authid'::regclass AND d.refobjid=g AND d.deptype='a' AND NOT
   ((d.classid='pg_database'::regclass AND d.objid=(SELECT oid FROM pg_database WHERE datname=current_database()))
    OR (d.classid='pg_namespace'::regclass AND d.objid='vec_autorizacion_atestada_v3'::regnamespace)
    OR (d.classid='pg_proc'::regclass AND d.objid=to_regprocedure('vec_autorizacion_atestada_v3.aprovisionar_gobierno_usuarios_admin_v1(text,text,text)'))))
 OR EXISTS(SELECT 1 FROM pg_database d CROSS JOIN LATERAL aclexplode(coalesce(d.datacl,acldefault('d',d.datdba))) a WHERE d.datname=current_database() AND a.grantee=g AND (a.privilege_type<>'CONNECT' OR a.is_grantable))
 OR EXISTS(SELECT 1 FROM pg_namespace n CROSS JOIN LATERAL aclexplode(coalesce(n.nspacl,acldefault('n',n.nspowner))) a WHERE n.oid='vec_autorizacion_atestada_v3'::regnamespace AND a.grantee=g AND (a.privilege_type<>'USAGE' OR a.is_grantable))
 OR EXISTS(SELECT 1 FROM pg_proc f CROSS JOIN LATERAL aclexplode(coalesce(f.proacl,acldefault('f',f.proowner))) a WHERE f.oid=to_regprocedure('vec_autorizacion_atestada_v3.aprovisionar_gobierno_usuarios_admin_v1(text,text,text)') AND a.grantee=g AND (a.privilege_type<>'EXECUTE' OR a.is_grantable))
 OR has_database_privilege(session_user,current_database(),'CREATE') OR has_database_privilege(session_user,current_database(),'TEMP')
 THEN RAISE EXCEPTION 'AD188: PARO clave=LOGIN actual=incompatible esperado=tecnico_exclusivo' USING ERRCODE='42501'; END IF;
 SELECT * INTO c FROM vec_autorizacion_atestada_v3.config_gobierno_usuarios_admin_v1 WHERE identidad_login=session_user FOR SHARE;
 IF NOT FOUND OR clock_timestamp()<c.vigente_desde OR clock_timestamp()>=c.vigente_hasta
 THEN RAISE EXCEPTION 'AD188: PARO clave=configuracion actual=ausente_o_caducada esperado=aprobada_vigente' USING ERRCODE='42501'; END IF;
 RETURN c;
END $f$;
CREATE FUNCTION vec_autorizacion_atestada_v3.preimagen_gobierno_usuarios_admin_v1()
RETURNS jsonb LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
 SELECT jsonb_build_object('configuracion',(SELECT to_jsonb(c) FROM vec_autorizacion_atestada_v3.puntero_configuracion_actual p JOIN vec_autorizacion_atestada_v3.configuracion_confianza_version c ON c.revision=p.configuracion_revision ORDER BY p.orden DESC LIMIT 1),
 'raices',(SELECT coalesce(jsonb_agg(to_jsonb(r) ORDER BY r.clave_id,r.version),'[]'::jsonb) FROM vec_autorizacion_atestada_v3.raiz_confianza_version r JOIN vec_autorizacion_atestada_v3.configuracion_raiz cr ON cr.raiz_clave_id=r.clave_id AND cr.raiz_version=r.version WHERE cr.configuracion_revision=(SELECT configuracion_revision FROM vec_autorizacion_atestada_v3.puntero_configuracion_actual ORDER BY orden DESC LIMIT 1)),
 'checkpoint',(SELECT to_jsonb(c) FROM vec_autorizacion_atestada_v3.checkpoint_gobierno c WHERE control_id),
 'orden_configuracion',(SELECT coalesce(max(orden),0) FROM vec_autorizacion_atestada_v3.puntero_configuracion_actual),
 'orden_claves',(SELECT coalesce(max(orden),0) FROM vec_autorizacion_atestada_v3.puntero_clave_emision),
 'revocaciones_raiz',(SELECT coalesce(jsonb_agg(to_jsonb(r) ORDER BY raiz_clave_id,raiz_version),'[]'::jsonb) FROM vec_autorizacion_atestada_v3.revocacion_raiz r),
 'revocaciones_configuracion',(SELECT coalesce(jsonb_agg(to_jsonb(r) ORDER BY configuracion_revision),'[]'::jsonb) FROM vec_autorizacion_atestada_v3.revocacion_configuracion r),
 'claves_usuarios',(SELECT coalesce(jsonb_agg(to_jsonb(k)-'secreto_hmac' ORDER BY clave_id,version),'[]'::jsonb) FROM vec_autorizacion_atestada_v3.clave_capacidad_version k WHERE audiencia_consumo IN('vec.admin.usuarios.listar.v1','vec.admin.usuarios.consultar.v1')))
$f$;
CREATE FUNCTION vec_autorizacion_atestada_v3.instante_gobierno_usuarios_v1(t timestamptz)
RETURNS text LANGUAGE sql IMMUTABLE STRICT SET search_path=pg_catalog AS $f$
 SELECT to_char(t AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS')||CASE WHEN to_char(t AT TIME ZONE 'UTC','US')='000000' THEN '' ELSE '.'||rtrim(to_char(t AT TIME ZONE 'UTC','US'),'0') END||'Z'
$f$;
CREATE FUNCTION vec_autorizacion_atestada_v3.huella_configuracion_gobierno_usuarios_v1(g jsonb,r vec_autorizacion_atestada_v3.raiz_confianza_version)
RETURNS text LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog AS $f$
DECLARE campos text[];campo text;material bytea:=''::bytea;
BEGIN
 campos:=ARRAY['vec.configuracion-confianza-atestacion-autorizacion.v3',g->>'revision',((g->>'secuencia')::bigint)::text,
 vec_autorizacion_atestada_v3.instante_gobierno_usuarios_v1((g->>'publicada_en')::timestamptz),vec_autorizacion_atestada_v3.instante_gobierno_usuarios_v1((g->>'expira_en')::timestamptz),
 r.clave_id,r.version::text,'EdDSA',r.huella_spki_sha256,r.suite,r.audiencia_despliegue,'activa',
 vec_autorizacion_atestada_v3.instante_gobierno_usuarios_v1(r.valida_desde),vec_autorizacion_atestada_v3.instante_gobierno_usuarios_v1(r.valida_hasta),''];
 FOREACH campo IN ARRAY campos LOOP
  IF campo IS NULL THEN RETURN NULL;END IF;
  material:=material||int8send(octet_length(convert_to(campo,'UTF8'))::bigint)||convert_to(campo,'UTF8');
 END LOOP;
 RETURN encode(sha256(material),'hex');
END $f$;
CREATE FUNCTION vec_autorizacion_atestada_v3.revalidar_gobierno_usuarios_admin_v1(p jsonb,m jsonb)
RETURNS void LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE k jsonb;g record;r vec_autorizacion_atestada_v3.raiz_confianza_version;checkpoint record;ahora timestamptz;
BEGIN
 PERFORM vec_autorizacion_atestada_v3.exigir_operador_gobierno_usuarios_admin_v1();
 SELECT x.* INTO STRICT g FROM vec_autorizacion_atestada_v3.puntero_configuracion_actual a JOIN vec_autorizacion_atestada_v3.configuracion_confianza_version x ON x.revision=a.configuracion_revision WHERE a.orden=(SELECT max(orden) FROM vec_autorizacion_atestada_v3.puntero_configuracion_actual) FOR SHARE OF a,x;
 SELECT x.* INTO STRICT r FROM vec_autorizacion_atestada_v3.configuracion_raiz cr JOIN vec_autorizacion_atestada_v3.raiz_confianza_version x ON x.clave_id=cr.raiz_clave_id AND x.version=cr.raiz_version WHERE cr.configuracion_revision=g.revision FOR SHARE OF cr,x;
 SELECT * INTO STRICT checkpoint FROM vec_autorizacion_atestada_v3.checkpoint_gobierno WHERE control_id FOR SHARE;
 PERFORM vec_autorizacion_atestada_v3.exigir_operador_gobierno_usuarios_admin_v1();
 ahora:=clock_timestamp();
 IF (p->>'caduca_en')::timestamptz<=ahora THEN RAISE EXCEPTION 'AD188: PARO clave=plan_vigencia actual=caducado esperado=vigente' USING ERRCODE='42501';END IF;
 IF g.revision IS DISTINCT FROM p->'configuracion'->>'revision' OR g.secuencia IS DISTINCT FROM (p->'configuracion'->>'secuencia')::bigint
 OR g.huella_configuracion_sha256 IS DISTINCT FROM p->'configuracion'->>'huella_sha256'
 OR g.publicada_en>ahora OR g.expira_en<=ahora OR r.valida_desde>ahora OR r.valida_hasta<=ahora
 OR g.secuencia<checkpoint.configuracion_secuencia_minima OR r.version<checkpoint.raiz_version_minima
 OR EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.revocacion_raiz x WHERE x.raiz_clave_id=r.clave_id AND x.raiz_version=r.version)
 OR EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.revocacion_configuracion x WHERE x.configuracion_revision=g.revision)
 OR vec_autorizacion_atestada_v3.huella_configuracion_gobierno_usuarios_v1(p->'configuracion',r) IS DISTINCT FROM g.huella_configuracion_sha256
 THEN RAISE EXCEPTION 'AD188: PARO clave=gobierno_actual actual=incompatible esperado=original_vigente_no_retirado' USING ERRCODE='42501';END IF;
 FOR k IN SELECT value FROM jsonb_array_elements(m->'claves') LOOP
  IF NOT EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.clave_capacidad_version x JOIN vec_autorizacion_atestada_v3.puntero_clave_emision a ON a.clave_id=x.clave_id AND a.version=x.version
   WHERE x.clave_id=k->>'clave_id' AND x.version=(k->>'version')::bigint AND x.audiencia_consumo=k->>'audiencia'
   AND x.revision_gobierno=(k->>'revision_gobierno')::bigint AND x.huella_gobierno_sha256=k->>'huella_gobierno_sha256' AND x.huella_secreto_sha256=k->>'huella_secreto_sha256'
   AND x.emisor_id=k->>'emisor_id' AND x.valida_desde=(k->>'valida_desde')::timestamptz AND x.valida_hasta=(k->>'valida_hasta')::timestamptz
   AND x.valida_desde<=ahora AND x.valida_hasta>ahora
   AND a.orden=(SELECT max(pc.orden) FROM vec_autorizacion_atestada_v3.puntero_clave_emision pc JOIN vec_autorizacion_atestada_v3.clave_capacidad_version kc ON kc.clave_id=pc.clave_id AND kc.version=pc.version WHERE kc.audiencia_consumo=x.audiencia_consumo AND pc.establecida_en<=ahora)
   AND NOT EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.revocacion_clave_capacidad rc WHERE rc.clave_id=x.clave_id AND rc.version=x.version))
  THEN RAISE EXCEPTION 'AD188: PARO clave=clave_actual actual=retirada_o_distinta esperado=original_vigente' USING ERRCODE='42501';END IF;
 END LOOP;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.instante_gobierno_usuarios_v1(timestamptz),vec_autorizacion_atestada_v3.huella_configuracion_gobierno_usuarios_v1(jsonb,vec_autorizacion_atestada_v3.raiz_confianza_version),vec_autorizacion_atestada_v3.revalidar_gobierno_usuarios_admin_v1(jsonb,jsonb) FROM PUBLIC;
CREATE FUNCTION vec_autorizacion_atestada_v3.efecto_gobierno_usuarios_admin_v1(p_plan text,p_aprobado text,p_material text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE c vec_autorizacion_atestada_v3.config_gobierno_usuarios_admin_v1;p jsonb;m jsonb;g jsonb;pre jsonb;sha text;k jsonb;claves jsonb;root vec_autorizacion_atestada_v3.raiz_confianza_version;anterior record;a record;previo record;secret bytea;ord bigint;i integer:=0;ref text;claves_sha text;ahora timestamptz;
BEGIN
 c:=vec_autorizacion_atestada_v3.exigir_operador_gobierno_usuarios_admin_v1();
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off' OR current_setting('TimeZone')<>'UTC'
 THEN RAISE EXCEPTION 'AD188: PARO clave=transaccion actual=incompatible esperado=serializable_rw_UTC' USING ERRCODE='25000';END IF;
 IF octet_length(p_plan)>16384 OR octet_length(p_material)>16384 THEN RAISE EXCEPTION 'AD188: PARO clave=entrada actual=grande esperado=acotada' USING ERRCODE='22023';END IF;
 sha:=encode(sha256(convert_to(p_plan,'UTF8')),'hex');
 IF sha IS DISTINCT FROM p_aprobado OR sha IS DISTINCT FROM c.plan_sha256
 OR encode(sha256(convert_to(p_material,'UTF8')),'hex') IS DISTINCT FROM c.material_sha256
 THEN RAISE EXCEPTION 'AD188: PARO clave=huella actual=distinta esperado=aprobada' USING ERRCODE='42501';END IF;
 p:=p_plan::jsonb;m:=p_material::jsonb;g:=p->'configuracion';claves:=m->'claves';
 IF jsonb_typeof(p)<>'object' OR (SELECT count(*) FROM jsonb_object_keys(p))<>7
 OR NOT p ?& ARRAY['version','operacion_ref','preparado_en','caduca_en','preimagen_sha256','configuracion','clave_ordenes']
 OR p->>'version'<>'1' OR p->>'operacion_ref' !~ '^gcu_[A-Za-z0-9_-]{22,124}$'
 OR p->>'preimagen_sha256' IS DISTINCT FROM c.preimagen_sha256
 OR (p->>'preparado_en')::timestamptz>clock_timestamp() OR (p->>'caduca_en')::timestamptz<=clock_timestamp()
 OR jsonb_typeof(g)<>'object' OR (SELECT count(*) FROM jsonb_object_keys(g))<>5 OR NOT g ?& ARRAY['revision','secuencia','huella_sha256','publicada_en','expira_en']
 OR jsonb_typeof(m)<>'object' OR (SELECT count(*) FROM jsonb_object_keys(m))<>1
 OR jsonb_typeof(claves)<>'array' OR jsonb_array_length(claves)<>2 OR jsonb_array_length(p->'clave_ordenes')<>2
 THEN RAISE EXCEPTION 'AD188: PARO clave=plan actual=incompatible esperado=ABI188' USING ERRCODE='22023';END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec:ct:desarrollo:gobierno-atestacion',0));
 -- Publicadores del gobierno existente usan este cerrojo; además CAS de tablas.
 LOCK TABLE vec_autorizacion_atestada_v3.puntero_configuracion_actual,vec_autorizacion_atestada_v3.puntero_clave_emision IN SHARE ROW EXCLUSIVE MODE;
 PERFORM vec_autorizacion_atestada_v3.exigir_operador_gobierno_usuarios_admin_v1();
 IF (p->>'caduca_en')::timestamptz<=clock_timestamp() THEN RAISE EXCEPTION 'AD188: PARO clave=plan_vigencia actual=caducado esperado=vigente' USING ERRCODE='42501';END IF;
 SELECT * INTO previo FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 WHERE tipo_registro='gobierno_usuarios_admin' AND plan_sha256=sha;
 IF FOUND THEN
  PERFORM vec_autorizacion_atestada_v3.revalidar_gobierno_usuarios_admin_v1(p,m);
  RETURN jsonb_build_object('auditoria_ref',previo.auditoria_ref,'secuencia',previo.secuencia,'huella_sha256',previo.huella_sha256,'correlacion_ref',previo.correlacion_ref,'registrada_en',previo.registrada_en,'replay',true,'plan_sha256',sha,'preimagen_sha256',c.preimagen_sha256,'material_sha256',c.material_sha256,'claves_sha256',previo.gobierno_usuarios_detalle->>'claves_sha256','configuracion_ref',g->>'revision');
 END IF;
 pre:=vec_autorizacion_atestada_v3.preimagen_gobierno_usuarios_admin_v1();
 IF encode(sha256(convert_to(pre::text,'UTF8')),'hex') IS DISTINCT FROM c.preimagen_sha256 THEN RAISE EXCEPTION 'AD188: PARO clave=preimagen actual=distinta esperado=aprobada' USING ERRCODE='40001';END IF;
 SELECT x.* INTO STRICT anterior FROM vec_autorizacion_atestada_v3.puntero_configuracion_actual a JOIN vec_autorizacion_atestada_v3.configuracion_confianza_version x ON x.revision=a.configuracion_revision ORDER BY a.orden DESC LIMIT 1;
 SELECT x.* INTO STRICT root FROM vec_autorizacion_atestada_v3.configuracion_raiz cr JOIN vec_autorizacion_atestada_v3.raiz_confianza_version x ON x.clave_id=cr.raiz_clave_id AND x.version=cr.raiz_version WHERE cr.configuracion_revision=anterior.revision;
 ahora:=clock_timestamp();
 IF root.valida_desde>ahora OR root.valida_hasta<=ahora OR EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.revocacion_raiz r WHERE r.raiz_clave_id=root.clave_id AND r.raiz_version=root.version)
 OR EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.revocacion_configuracion r WHERE r.configuracion_revision=anterior.revision)
 OR (g->>'publicada_en')::timestamptz<>date_trunc('day',ahora) OR (g->>'expira_en')::timestamptz<>(g->>'publicada_en')::timestamptz+interval '1 day'
 OR g->>'revision' !~ '^confianza:atestacion:ct:desarrollo:[0-9]{4}-[0-9]{2}-[0-9]{2}(:r[0-9]+)?$'
 OR vec_autorizacion_atestada_v3.huella_configuracion_gobierno_usuarios_v1(g,root) IS DISTINCT FROM g->>'huella_sha256'
 OR g->>'huella_sha256' !~ '^[0-9a-f]{64}$' OR (g->>'secuencia')::bigint<=anterior.secuencia
 OR (g->>'secuencia')::bigint<=(pre->>'orden_configuracion')::bigint
 THEN RAISE EXCEPTION 'AD188: PARO clave=renovacion actual=incompatible esperado=misma_raiz_diaria_aprobada' USING ERRCODE='42501';END IF;
 FOR k IN SELECT value FROM jsonb_array_elements(claves) LOOP
  i:=i+1;ord:=(p->'clave_ordenes'->>(i-1))::bigint;
  IF jsonb_typeof(k)<>'object' OR (SELECT count(*) FROM jsonb_object_keys(k))<>10
  OR NOT k ?& ARRAY['audiencia','clave_id','version','revision_gobierno','huella_gobierno_sha256','secreto_hmac','huella_secreto_sha256','emisor_id','valida_desde','valida_hasta']
  OR k->>'audiencia' IS DISTINCT FROM (CASE i WHEN 1 THEN 'vec.admin.usuarios.listar.v1' ELSE 'vec.admin.usuarios.consultar.v1' END)
  OR k->>'clave_id' !~ '^clave:capacidad:admin:usuarios:[a-z0-9:._-]{1,120}$' OR k->>'emisor_id' !~ '^emisor:admin:[a-z0-9:._-]{1,120}$'
  OR (k->>'version')::bigint NOT BETWEEN 1 AND 9007199254740991 OR (k->>'revision_gobierno')::bigint NOT BETWEEN 1 AND 9007199254740991
  OR (k->>'valida_desde')::timestamptz>ahora OR (k->>'valida_hasta')::timestamptz<=ahora OR (k->>'valida_hasta')::timestamptz>root.valida_hasta
  OR ord<=(pre->>'orden_claves')::bigint OR ord>9007199254740991
  OR k->>'huella_gobierno_sha256' !~ '^[0-9a-f]{64}$' OR k->>'huella_secreto_sha256' !~ '^[0-9a-f]{64}$'
  THEN RAISE EXCEPTION 'AD188: PARO clave=clave actual=incompatible esperado=audiencia_dedicada_aprobada' USING ERRCODE='22023';END IF;
  secret:=decode(k->>'secreto_hmac','base64');
  IF octet_length(secret)<>32 OR encode(sha256(secret),'hex') IS DISTINCT FROM k->>'huella_secreto_sha256'
  OR EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.clave_capacidad_version x WHERE x.clave_id=k->>'clave_id' OR x.secreto_hmac=secret)
  THEN RAISE EXCEPTION 'AD188: PARO clave=material actual=incompatible esperado=nuevo_dedicado_32bytes' USING ERRCODE='42501';END IF;
  INSERT INTO vec_autorizacion_atestada_v3.clave_capacidad_version(clave_id,version,revision_gobierno,huella_gobierno_sha256,secreto_hmac,huella_secreto_sha256,emisor_id,audiencia_consumo,valida_desde,valida_hasta,acto_ref)
  VALUES(k->>'clave_id',(k->>'version')::bigint,(k->>'revision_gobierno')::bigint,k->>'huella_gobierno_sha256',secret,k->>'huella_secreto_sha256',k->>'emisor_id',k->>'audiencia',(k->>'valida_desde')::timestamptz,(k->>'valida_hasta')::timestamptz,'acto_tecnico:admin:usuarios:clave:'||sha||':'||i::text);
  INSERT INTO vec_autorizacion_atestada_v3.puntero_clave_emision(orden,clave_id,version,establecida_en,acto_ref) VALUES(ord,k->>'clave_id',(k->>'version')::bigint,ahora,'acto_tecnico:admin:usuarios:puntero:'||sha||':'||i::text);
 END LOOP;
 -- Sólo configuración y enlace/puntero nuevos; raíz original intacta.
 INSERT INTO vec_autorizacion_atestada_v3.configuracion_confianza_version(revision,secuencia,huella_configuracion_sha256,publicada_en,expira_en,acto_ref)
 VALUES(g->>'revision',(g->>'secuencia')::bigint,g->>'huella_sha256',(g->>'publicada_en')::timestamptz,(g->>'expira_en')::timestamptz,'acto:ct:desarrollo:configuracion:r'||(g->>'secuencia'));
 INSERT INTO vec_autorizacion_atestada_v3.configuracion_raiz(configuracion_revision,raiz_clave_id,raiz_version) VALUES(g->>'revision',root.clave_id,root.version);
 INSERT INTO vec_autorizacion_atestada_v3.puntero_configuracion_actual(orden,configuracion_revision,establecida_en,acto_ref)
 VALUES((g->>'secuencia')::bigint,g->>'revision',(g->>'publicada_en')::timestamptz,'acto:ct:desarrollo:puntero-configuracion:r'||(g->>'secuencia'));
 claves_sha:=encode(sha256(convert_to((SELECT jsonb_agg(value-'secreto_hmac') FROM jsonb_array_elements(claves))::text,'UTF8')),'hex');
 SELECT * INTO a FROM vec_autorizacion_atestada_v3.registrar_gobierno_usuarios_admin_v1(jsonb_build_object('tipo_registro','gobierno_usuarios_admin','evento_ref','evento_'||substr(sha,1,32),'operador_login',session_user::text,'plan_sha256',sha,'preimagen_sha256',c.preimagen_sha256,'configuracion_origen_ref',anterior.revision,'configuracion_destino_ref',g->>'revision','claves_sha256',claves_sha,'proceso','postgresql','canal','operacion_tecnica_privada','finalidad_ref','gobierno_usuarios_admin','correlacion_ref','correlacion_'||substr(sha,1,32)));
 PERFORM vec_autorizacion_atestada_v3.revalidar_gobierno_usuarios_admin_v1(p,m);
 RETURN to_jsonb(a)||jsonb_build_object('replay',false,'plan_sha256',sha,'preimagen_sha256',c.preimagen_sha256,'material_sha256',c.material_sha256,'claves_sha256',claves_sha,'configuracion_ref',g->>'revision');
END $f$;
CREATE FUNCTION vec_autorizacion_atestada_v3.aprovisionar_gobierno_usuarios_admin_v1(p_plan text,p_aprobado text,p_material text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE recibo jsonb;intento record;estado text:='permitido';motivo text:='gobierno_usuarios_registrado';codigo text:='gobierno_usuarios_registrado';solicitud text;
BEGIN
 solicitud:=encode(sha256(convert_to(coalesce(p_plan,''),'UTF8')),'hex');
 BEGIN
  recibo:=vec_autorizacion_atestada_v3.efecto_gobierno_usuarios_admin_v1(p_plan,p_aprobado,p_material);
  IF recibo->>'replay'='true' THEN motivo:='gobierno_usuarios_replay';codigo:=motivo;END IF;
 SELECT * INTO intento FROM vec_autorizacion_atestada_v3.registrar_intento_gobierno_usuarios_admin_v1(jsonb_build_object('tipo_registro','intento_gobierno_usuarios_admin','evento_ref','evento_'||replace(gen_random_uuid()::text,'-',''),'operador_login',session_user::text,'solicitud_sha256',solicitud,'accion','aprovisionar_gobierno_usuarios_admin_v1','recurso_ref','solicitud_gobierno_usuarios:'||substr(solicitud,1,32),'resultado',estado,'motivo_ref',motivo,'proceso','postgresql','canal','operacion_tecnica_privada','finalidad_ref','gobierno_usuarios_admin','correlacion_ref','correlacion_'||replace(gen_random_uuid()::text,'-','')));
  -- Incluye la última espera de auditoría en el subbloque del efecto.
  PERFORM vec_autorizacion_atestada_v3.revalidar_gobierno_usuarios_admin_v1(p_plan::jsonb,p_material::jsonb);
 EXCEPTION WHEN insufficient_privilege OR serialization_failure OR invalid_parameter_value OR invalid_text_representation OR datetime_field_overflow OR unique_violation OR no_data_found THEN
  estado:='denegado';motivo:='gobierno_usuarios_denegado';codigo:=motivo;recibo:=NULL;
 WHEN OTHERS THEN estado:='error';motivo:='gobierno_usuarios_error';codigo:=motivo;recibo:=NULL;
 END;
 IF estado<>'permitido' THEN
  -- El efecto/confirmación/intento permitido se han revertido juntos.
 SELECT * INTO intento FROM vec_autorizacion_atestada_v3.registrar_intento_gobierno_usuarios_admin_v1(jsonb_build_object('tipo_registro','intento_gobierno_usuarios_admin','evento_ref','evento_'||replace(gen_random_uuid()::text,'-',''),'operador_login',session_user::text,'solicitud_sha256',solicitud,'accion','aprovisionar_gobierno_usuarios_admin_v1','recurso_ref','solicitud_gobierno_usuarios:'||substr(solicitud,1,32),'resultado',estado,'motivo_ref',motivo,'proceso','postgresql','canal','operacion_tecnica_privada','finalidad_ref','gobierno_usuarios_admin','correlacion_ref','correlacion_'||replace(gen_random_uuid()::text,'-','')));
 END IF;
 RETURN jsonb_build_object('estado',estado,'codigo',codigo,'recibo',recibo,'auditoria_intento',to_jsonb(intento)||jsonb_build_object('solicitud_sha256',solicitud));
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.exigir_operador_gobierno_usuarios_admin_v1(),vec_autorizacion_atestada_v3.preimagen_gobierno_usuarios_admin_v1(),vec_autorizacion_atestada_v3.efecto_gobierno_usuarios_admin_v1(text,text,text),vec_autorizacion_atestada_v3.aprovisionar_gobierno_usuarios_admin_v1(text,text,text) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_gobierno_usuarios_admin_operador;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.aprovisionar_gobierno_usuarios_admin_v1(text,text,text) TO vec_gobierno_usuarios_admin_operador;
COMMIT;
