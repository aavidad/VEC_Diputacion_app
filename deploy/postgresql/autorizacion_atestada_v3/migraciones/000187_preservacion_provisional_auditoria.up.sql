\set ON_ERROR_STOP on
-- AD187: preservación técnica provisional; no valoración documental ni expurgo.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000187',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
DO $pre$
DECLARE v_sha text;
BEGIN
 SELECT encode(sha256(convert_to(pg_get_constraintdef(c.oid,false),'UTF8')),'hex') INTO v_sha FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass AND c.conname='auditoria_tipo_disjunto_v4' AND c.convalidated;
 IF v_sha IS DISTINCT FROM 'f31523524c58dece701182585498562c40a1e1de89fefae7953916cd409e3b5e'
 THEN RAISE EXCEPTION 'AD187: PARO clave=CHECK_sha actual=% esperado=f31523524c58dece701182585498562c40a1e1de89fefae7953916cd409e3b5e',coalesce(v_sha,'ausente') USING ERRCODE='55000'; END IF;
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR to_regclass('vec_autorizacion_atestada_v3.preservacion_auditoria_version_v1') IS NOT NULL
 OR to_regrole('vec_auditoria_preservacion_configurador') IS NOT NULL OR to_regrole('vec_auditoria_preservacion_consultor') IS NOT NULL
 THEN RAISE EXCEPTION 'AD187: PARO clave=preimagen actual=incompatible esperado=POST186_sin_AD187' USING ERRCODE='55000'; END IF;
END $pre$;
CREATE ROLE vec_auditoria_preservacion_configurador NOLOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_auditoria_preservacion_consultor NOLOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
LOCK TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 IN ACCESS EXCLUSIVE MODE;
ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 ADD COLUMN preservacion_detalle jsonb;
DO $familia$
DECLARE v_def text;v_nuevo text;
BEGIN
 SELECT pg_get_constraintdef(c.oid,false) INTO STRICT v_def FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass AND c.conname='auditoria_tipo_disjunto_v4';
 v_nuevo:='CHECK ((preservacion_detalle IS NULL AND ('||substr(v_def,8,length(v_def)-8)||')) OR ('||$tipo$
 tipo_registro IS NOT DISTINCT FROM 'operacion_tecnica_preservacion_auditoria'
 AND decision_ref IS NULL AND efecto_ref IS NULL AND huella_efecto_sha256 IS NULL AND version_consumo IS NULL
 AND intento_ref IS NULL AND intento_material_sha256 IS NULL AND actor_ref IS NULL AND perfil_activo_ref IS NULL
 AND registro_contexto_ref IS NULL AND contexto_sha256 IS NULL AND procedencia_sha256 IS NULL
 AND autenticacion_ref IS NULL AND sesion_ref IS NULL AND autenticacion_sha256 IS NULL AND vinculo_sha256 IS NULL
 AND fuente_ref IS NULL AND fuente_sha256 IS NULL AND plan_sha256 IS NULL AND aprobacion_ref IS NULL
 AND fuentes_plan_ref IS NULL AND fuentes_preimagen_sha256 IS NULL AND fuentes_configuracion_sha256 IS NULL AND fuentes_alcance IS NULL AND fuentes_solicitud_sha256 IS NULL
 AND unidad_plan_ref IS NULL AND unidad_preimagen_sha256 IS NULL AND unidad_configuracion_sha256 IS NULL AND unidad_alcance IS NULL
 AND unidad_recibo_ref IS NULL AND unidad_recibo_sha256 IS NULL AND unidad_solicitud_sha256 IS NULL AND bootstrap_solicitud_sha256 IS NULL
 AND mantenimiento_detalle IS NULL AND mantenimiento_solicitud_sha256 IS NULL
 AND periodica_detalle IS NULL
 AND evento_ref IS NOT NULL AND evento_material_sha256 IS NOT NULL AND operador_login IS NOT NULL
 AND preservacion_detalle IS NOT NULL AND jsonb_typeof(preservacion_detalle)='object' AND octet_length(preservacion_detalle::text)<=8192
 AND modulo_id IS NOT DISTINCT FROM 'auditoria' AND recurso_ref IS NOT DISTINCT FROM 'auditoria_periodica:comun_interna'
 AND finalidad_ref IS NOT DISTINCT FROM 'preservacion_provisional_auditoria'
 AND proceso IS NOT DISTINCT FROM 'postgresql' AND canal IS NOT DISTINCT FROM 'operacion_tecnica_privada'
 AND correlacion_ref IS NOT NULL AND resultado IS NOT NULL AND motivo_ref IS NOT NULL AND accion IS NOT NULL
 AND accion IN('configurar_preservacion_auditoria_v1','consultar_preservacion_auditoria_v1') AND resultado IN('permitido','denegado','error')
 $tipo$||'))';
 ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 DROP CONSTRAINT auditoria_tipo_disjunto_v4;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 ADD CONSTRAINT auditoria_tipo_disjunto_v4 '||v_nuevo;
END $familia$;
ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 ADD CONSTRAINT auditoria_preservacion_formato_v1 CHECK(
 tipo_registro<>'operacion_tecnica_preservacion_auditoria' OR (
 evento_ref ~ '^evento_[0-9a-f]{32}$' AND evento_material_sha256 ~ '^[0-9a-f]{64}$' AND correlacion_ref ~ '^correlacion_[0-9a-f]{32}$'
 AND motivo_ref IN('preservacion_publicada','preservacion_replay','preservacion_consultada','preservacion_ausente','operacion_denegada','operacion_error')));
CREATE TABLE vec_autorizacion_atestada_v3.preservacion_auditoria_version_v1(
 version bigint PRIMARY KEY CHECK(version BETWEEN 1 AND 999999999),
 publicacion_ref text NOT NULL UNIQUE CHECK(publicacion_ref ~ '^preservacion_[0-9a-f]{32}$'),
 configuracion jsonb NOT NULL, huella_sha256 text NOT NULL UNIQUE CHECK(huella_sha256 ~ '^[0-9a-f]{64}$'),
 registrada_en timestamptz(6) NOT NULL, auditoria_ref text NOT NULL UNIQUE REFERENCES vec_autorizacion_atestada_v3.auditoria_consumo_v3(auditoria_ref));
REVOKE ALL ON TABLE vec_autorizacion_atestada_v3.preservacion_auditoria_version_v1 FROM PUBLIC;
REVOKE ALL ON TYPE vec_autorizacion_atestada_v3.preservacion_auditoria_version_v1 FROM PUBLIC;
CREATE TRIGGER solo_adicion BEFORE UPDATE OR DELETE ON vec_autorizacion_atestada_v3.preservacion_auditoria_version_v1
FOR EACH ROW EXECUTE FUNCTION vec_autorizacion_atestada_v3.rechazar_mutacion();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_autorizacion_atestada_v3.preservacion_auditoria_version_v1
FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion_atestada_v3.rechazar_truncado();
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_operacion_preservacion_v1(p_accion text,p_resultado text,p_motivo text,p_correlacion text,p_detalle jsonb)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE SET search_path=pg_catalog SET row_security=on AS $f$
DECLARE v_seq numeric;v_prev text;v_ref text;v_evento text;v_material text;v_hash text;v_fecha timestamptz(6);v_bytes bytea;v_valor text;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off' OR current_setting('TimeZone')<>'UTC'
 THEN RAISE EXCEPTION 'preservacion_transaccion_invalida' USING ERRCODE='25000'; END IF;
 IF p_correlacion IS NULL OR p_correlacion !~ '^correlacion_[0-9a-f]{32}$' OR p_detalle IS NULL
 THEN RAISE EXCEPTION 'preservacion_entrada_invalida' USING ERRCODE='22023'; END IF;
 SELECT secuencia,cabeza_sha256 INTO STRICT v_seq,v_prev FROM vec_autorizacion_atestada_v3.control_cadena_auditoria WHERE control_id FOR UPDATE;
 v_seq:=v_seq+1;v_fecha:=clock_timestamp();v_evento:='evento_'||replace(gen_random_uuid()::text,'-','');v_ref:='aud_v3_pre_'||substr(v_evento,8);
 v_bytes:=vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.preservacion.material.v1');
 FOREACH v_valor IN ARRAY ARRAY['operacion_tecnica_preservacion_auditoria',v_evento,session_user::text,p_accion,'auditoria','auditoria_periodica:comun_interna','preservacion_provisional_auditoria',p_resultado,p_motivo,'postgresql','operacion_tecnica_privada',p_correlacion,p_detalle::text] LOOP
  v_bytes:=v_bytes||vec_autorizacion_atestada_v3.encuadrar_mac(v_valor);
 END LOOP;
 v_material:=encode(sha256(v_bytes),'hex');
 v_hash:=encode(sha256(vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.preservacion.eslabon.v1')||
 vec_autorizacion_atestada_v3.encuadrar_mac(v_seq::text)||vec_autorizacion_atestada_v3.encuadrar_mac(v_prev)||
 vec_autorizacion_atestada_v3.encuadrar_mac(v_ref)||vec_autorizacion_atestada_v3.encuadrar_mac(v_material)||
 vec_autorizacion_atestada_v3.encuadrar_mac(to_char(v_fecha AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))),'hex');
 INSERT INTO vec_autorizacion_atestada_v3.auditoria_consumo_v3(auditoria_ref,secuencia,anterior_sha256,huella_sha256,registrada_en,tipo_registro,evento_ref,evento_material_sha256,operador_login,accion,modulo_id,recurso_ref,finalidad_ref,resultado,motivo_ref,proceso,canal,correlacion_ref,preservacion_detalle)
 VALUES(v_ref,v_seq,v_prev,v_hash,v_fecha,'operacion_tecnica_preservacion_auditoria',v_evento,v_material,session_user::name,p_accion,'auditoria','auditoria_periodica:comun_interna','preservacion_provisional_auditoria',p_resultado,p_motivo,'postgresql','operacion_tecnica_privada',p_correlacion,p_detalle);
 UPDATE vec_autorizacion_atestada_v3.control_cadena_auditoria SET secuencia=v_seq,cabeza_sha256=v_hash,actualizada_en=v_fecha WHERE control_id;
 RETURN jsonb_build_object('auditoria_ref',v_ref,'secuencia',v_seq,'huella_sha256',v_hash,'registrada_en',to_char(v_fecha AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'correlacion_ref',p_correlacion);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_operacion_preservacion_v1(text,text,text,text,jsonb) FROM PUBLIC;


CREATE FUNCTION vec_autorizacion_atestada_v3.resultado_preservacion_v1(p vec_autorizacion_atestada_v3.preservacion_auditoria_version_v1,p_estado text,p_acuse jsonb)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE SET search_path=pg_catalog SET row_security=on AS $f$
DECLARE a record;
BEGIN
 SELECT * INTO STRICT a FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 WHERE auditoria_ref=p.auditoria_ref;
 RETURN jsonb_build_object('estado',p_estado,'configuracion',p.configuracion,'configuracion_sha256',p.huella_sha256,
 'registrada_en',to_char(p.registrada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
 'acuse_original',jsonb_build_object('auditoria_ref',a.auditoria_ref,'secuencia',a.secuencia,'huella_sha256',a.huella_sha256,
 'registrada_en',to_char(a.registrada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'correlacion_ref',a.correlacion_ref),'acuse_acceso',p_acuse);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.resultado_preservacion_v1(vec_autorizacion_atestada_v3.preservacion_auditoria_version_v1,text,jsonb) FROM PUBLIC;
CREATE FUNCTION vec_autorizacion_atestada_v3.configurar_preservacion_auditoria_v1(p_texto text,p_correlacion text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' SET statement_timeout='10s' AS $f$
DECLARE j json;c jsonb;v_sha text;v_ultima record;v_existente vec_autorizacion_atestada_v3.preservacion_auditoria_version_v1;a jsonb;v_detalle jsonb;
BEGIN
 IF pg_has_role(session_user,'vec_autorizacion_atestada_v3_propietario','MEMBER') OR pg_has_role(session_user,'vec_autorizacion_propietario','MEMBER')
 OR NOT pg_has_role(session_user,'vec_auditoria_preservacion_configurador','MEMBER') THEN RAISE EXCEPTION 'preservacion_autoridad_denegada' USING ERRCODE='42501'; END IF;
 IF p_texto IS NULL OR octet_length(p_texto)>4096 THEN RAISE EXCEPTION 'preservacion_configuracion_invalida' USING ERRCODE='22023'; END IF;
 BEGIN j:=p_texto::json;c:=p_texto::jsonb; EXCEPTION WHEN invalid_text_representation THEN RAISE EXCEPTION 'preservacion_configuracion_invalida' USING ERRCODE='22023'; END;
 IF json_typeof(j) IS DISTINCT FROM 'object' OR (SELECT count(*) FROM json_each(j))<>6 OR (SELECT count(DISTINCT key) FROM json_each(j))<>6
 OR NOT c ?& ARRAY['publicacion_ref','version','preimagen_sha256','decision_tecnica_ref','estado','medida']
 OR jsonb_typeof(c->'version') IS DISTINCT FROM 'number' OR c->>'version' !~ '^[1-9][0-9]{0,8}$'
 OR jsonb_typeof(c->'publicacion_ref') IS DISTINCT FROM 'string' OR c->>'publicacion_ref' !~ '^preservacion_[0-9a-f]{32}$'
 OR jsonb_typeof(c->'preimagen_sha256') IS DISTINCT FROM 'string' OR c->>'preimagen_sha256' !~ '^[0-9a-f]{64}$'
 OR jsonb_typeof(c->'decision_tecnica_ref') IS DISTINCT FROM 'string' OR c->>'decision_tecnica_ref' !~ '^decision_tecnica_[0-9a-f]{32}$'
 OR c->>'estado' IS DISTINCT FROM 'provisional' OR c->>'medida' IS DISTINCT FROM 'conservar_todo_sin_expurgo'
 THEN RAISE EXCEPTION 'preservacion_configuracion_invalida' USING ERRCODE='22023'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:preservacion',0));
 v_sha:=encode(sha256(convert_to(c::text,'UTF8')),'hex');
 SELECT * INTO v_existente FROM vec_autorizacion_atestada_v3.preservacion_auditoria_version_v1 WHERE publicacion_ref=c->>'publicacion_ref';
 v_detalle:=c||jsonb_build_object('configuracion_sha256',v_sha,'perfil_tecnico_ref','vec_auditoria_preservacion_configurador');
 IF v_existente.version IS NOT NULL THEN
  IF v_existente.configuracion IS DISTINCT FROM c OR v_existente.huella_sha256 IS DISTINCT FROM v_sha THEN RAISE EXCEPTION 'preservacion_publicacion_conflicto' USING ERRCODE='23505'; END IF;
  a:=vec_autorizacion_atestada_v3.registrar_operacion_preservacion_v1('configurar_preservacion_auditoria_v1','permitido','preservacion_replay',p_correlacion,v_detalle);
  RETURN vec_autorizacion_atestada_v3.resultado_preservacion_v1(v_existente,'replay',a);
 END IF;
 SELECT * INTO v_ultima FROM vec_autorizacion_atestada_v3.preservacion_auditoria_version_v1 ORDER BY version DESC LIMIT 1;
 IF (c->>'version')::bigint<>coalesce(v_ultima.version,0)+1 OR c->>'preimagen_sha256' IS DISTINCT FROM coalesce(v_ultima.huella_sha256,repeat('0',64))
 THEN RAISE EXCEPTION 'preservacion_preimagen_no_coincide' USING ERRCODE='40001'; END IF;
 a:=vec_autorizacion_atestada_v3.registrar_operacion_preservacion_v1('configurar_preservacion_auditoria_v1','permitido','preservacion_publicada',p_correlacion,v_detalle);
 INSERT INTO vec_autorizacion_atestada_v3.preservacion_auditoria_version_v1 VALUES((c->>'version')::bigint,c->>'publicacion_ref',c,v_sha,(a->>'registrada_en')::timestamptz,a->>'auditoria_ref') RETURNING * INTO v_existente;
 RETURN vec_autorizacion_atestada_v3.resultado_preservacion_v1(v_existente,'publicada',a);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.configurar_preservacion_auditoria_v1(text,text) FROM PUBLIC;
CREATE FUNCTION vec_autorizacion_atestada_v3.consultar_preservacion_auditoria_v1(p_version bigint,p_correlacion text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' SET statement_timeout='10s' AS $f$
DECLARE v vec_autorizacion_atestada_v3.preservacion_auditoria_version_v1;a jsonb;
BEGIN
 IF pg_has_role(session_user,'vec_autorizacion_atestada_v3_propietario','MEMBER') OR pg_has_role(session_user,'vec_autorizacion_propietario','MEMBER')
 OR NOT pg_has_role(session_user,'vec_auditoria_preservacion_consultor','MEMBER') THEN RAISE EXCEPTION 'preservacion_autoridad_denegada' USING ERRCODE='42501'; END IF;
 IF p_version IS NULL OR p_version<0 OR p_version>999999999 THEN RAISE EXCEPTION 'preservacion_version_invalida' USING ERRCODE='22023'; END IF;
 SELECT * INTO v FROM vec_autorizacion_atestada_v3.preservacion_auditoria_version_v1 WHERE p_version=0 OR version=p_version ORDER BY version DESC LIMIT 1;
 IF v.version IS NULL THEN
  a:=vec_autorizacion_atestada_v3.registrar_operacion_preservacion_v1('consultar_preservacion_auditoria_v1','permitido','preservacion_ausente',p_correlacion,jsonb_build_object('version_solicitada',p_version,'perfil_tecnico_ref','vec_auditoria_preservacion_consultor'));
  RETURN jsonb_build_object('estado','no_publicada','acuse_acceso',a);
 END IF;
 a:=vec_autorizacion_atestada_v3.registrar_operacion_preservacion_v1('consultar_preservacion_auditoria_v1','permitido','preservacion_consultada',p_correlacion,jsonb_build_object('publicacion_ref',v.publicacion_ref,'version',v.version,'configuracion_sha256',v.huella_sha256,'perfil_tecnico_ref','vec_auditoria_preservacion_consultor'));
 RETURN vec_autorizacion_atestada_v3.resultado_preservacion_v1(v,'consultada',a);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consultar_preservacion_auditoria_v1(bigint,text) FROM PUBLIC;
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_intento_preservacion_v1(p_accion text,p_resultado text,p_correlacion text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' SET statement_timeout='10s' AS $f$
DECLARE v_perfil text;
BEGIN
 v_perfil:=CASE WHEN p_accion='configurar_preservacion_auditoria_v1' THEN 'vec_auditoria_preservacion_configurador' ELSE 'vec_auditoria_preservacion_consultor' END;
 IF p_accion IS NULL OR p_resultado IS NULL OR p_accion NOT IN('configurar_preservacion_auditoria_v1','consultar_preservacion_auditoria_v1') OR p_resultado NOT IN('denegado','error')
 OR pg_has_role(session_user,'vec_autorizacion_atestada_v3_propietario','MEMBER') OR pg_has_role(session_user,'vec_autorizacion_propietario','MEMBER') OR NOT pg_has_role(session_user,v_perfil,'MEMBER')
 THEN RAISE EXCEPTION 'preservacion_intento_denegado' USING ERRCODE='42501'; END IF;
 RETURN vec_autorizacion_atestada_v3.registrar_operacion_preservacion_v1(p_accion,p_resultado,CASE WHEN p_resultado='denegado' THEN 'operacion_denegada' ELSE 'operacion_error' END,p_correlacion,jsonb_build_object('perfil_tecnico_ref',v_perfil));
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_intento_preservacion_v1(text,text,text) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_auditoria_preservacion_configurador,vec_auditoria_preservacion_consultor;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.configurar_preservacion_auditoria_v1(text,text),vec_autorizacion_atestada_v3.registrar_intento_preservacion_v1(text,text,text) TO vec_auditoria_preservacion_configurador;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consultar_preservacion_auditoria_v1(bigint,text),vec_autorizacion_atestada_v3.registrar_intento_preservacion_v1(text,text,text) TO vec_auditoria_preservacion_consultor;
COMMIT;
