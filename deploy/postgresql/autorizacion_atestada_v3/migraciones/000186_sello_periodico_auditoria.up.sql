\set ON_ERROR_STOP on
-- A3: autoridad técnica específica de captura y conservación de checkpoints.
-- No concede permisos humanos ni usa los owners como ejecutores del sellado.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000186',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
DO $pre$
DECLARE v_sha text;
BEGIN
 SELECT encode(sha256(convert_to(pg_get_constraintdef(c.oid,false),'UTF8')),'hex') INTO v_sha
 FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
 AND c.conname='auditoria_tipo_disjunto_v4' AND c.convalidated;
 IF v_sha IS DISTINCT FROM 'dacbd820f1679fc2f6a02bb3d001a0a69df8528c5693f8e9ea33c17cb7fe4fa5'
 THEN RAISE EXCEPTION 'AD186: PARO clave=CHECK_sha actual=% esperado=dacbd820f1679fc2f6a02bb3d001a0a69df8528c5693f8e9ea33c17cb7fe4fa5',coalesce(v_sha,'ausente') USING ERRCODE='55000'; END IF;
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR to_regclass('vec_autorizacion_atestada_v3.politicas_sello_periodico_v1') IS NOT NULL
 OR to_regrole('vec_auditoria_periodica_configurador') IS NOT NULL
 OR to_regrole('vec_auditoria_periodica_sellador') IS NOT NULL
 THEN RAISE EXCEPTION 'AD186: PARO clave=preimagen actual=incompatible esperado=POST183_sin_AD186' USING ERRCODE='55000'; END IF;
END $pre$;
CREATE ROLE vec_auditoria_periodica_configurador NOLOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_auditoria_periodica_sellador NOLOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
LOCK TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 IN ACCESS EXCLUSIVE MODE;
ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 ADD COLUMN periodica_detalle jsonb;
DO $familia$
DECLARE v_def text;v_nuevo text;
BEGIN
 SELECT pg_get_constraintdef(c.oid,false) INTO STRICT v_def FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass AND c.conname='auditoria_tipo_disjunto_v4';
 v_nuevo:='CHECK ((periodica_detalle IS NULL AND ('||substr(v_def,8,length(v_def)-8)||')) OR ('||$tipo$
 tipo_registro IS NOT DISTINCT FROM 'operacion_tecnica_auditoria_periodica'
 AND decision_ref IS NULL AND efecto_ref IS NULL AND huella_efecto_sha256 IS NULL AND version_consumo IS NULL
 AND intento_ref IS NULL AND intento_material_sha256 IS NULL AND actor_ref IS NULL AND perfil_activo_ref IS NULL
 AND registro_contexto_ref IS NULL AND contexto_sha256 IS NULL AND procedencia_sha256 IS NULL
 AND autenticacion_ref IS NULL AND sesion_ref IS NULL AND autenticacion_sha256 IS NULL AND vinculo_sha256 IS NULL
 AND fuente_ref IS NULL AND fuente_sha256 IS NULL AND plan_sha256 IS NULL AND aprobacion_ref IS NULL
 AND fuentes_plan_ref IS NULL AND fuentes_preimagen_sha256 IS NULL AND fuentes_configuracion_sha256 IS NULL AND fuentes_alcance IS NULL AND fuentes_solicitud_sha256 IS NULL
 AND unidad_plan_ref IS NULL AND unidad_preimagen_sha256 IS NULL AND unidad_configuracion_sha256 IS NULL AND unidad_alcance IS NULL
 AND unidad_recibo_ref IS NULL AND unidad_recibo_sha256 IS NULL AND unidad_solicitud_sha256 IS NULL AND bootstrap_solicitud_sha256 IS NULL
 AND mantenimiento_detalle IS NULL AND mantenimiento_solicitud_sha256 IS NULL
 AND evento_ref IS NOT NULL AND evento_material_sha256 IS NOT NULL AND operador_login IS NOT NULL
 AND periodica_detalle IS NOT NULL AND jsonb_typeof(periodica_detalle)='object' AND octet_length(periodica_detalle::text)<=8192
 AND modulo_id IS NOT DISTINCT FROM 'auditoria' AND recurso_ref IS NOT DISTINCT FROM 'auditoria_periodica:comun_interna'
 AND proceso IS NOT DISTINCT FROM 'postgresql' AND canal IS NOT DISTINCT FROM 'operacion_tecnica_privada'
 AND finalidad_ref IS NOT DISTINCT FROM 'integridad_auditoria_periodica' AND correlacion_ref IS NOT NULL
 AND resultado IS NOT NULL AND motivo_ref IS NOT NULL
 AND accion IN('configurar_sello_periodico_v1','capturar_sello_periodico_v1','confirmar_sello_periodico_v1')
 AND resultado IN('permitido','denegado','error')
 $tipo$||'))';
 ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 DROP CONSTRAINT auditoria_tipo_disjunto_v4;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 ADD CONSTRAINT auditoria_tipo_disjunto_v4 '||v_nuevo;
END $familia$;
ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 ADD CONSTRAINT auditoria_periodica_formato_v1 CHECK(
 tipo_registro<>'operacion_tecnica_auditoria_periodica' OR (
 evento_ref ~ '^evento_[0-9a-f]{32}$' AND evento_material_sha256 ~ '^[0-9a-f]{64}$'
 AND correlacion_ref ~ '^correlacion_[0-9a-f]{32}$' AND motivo_ref IN('politica_registrada','captura_registrada','captura_recuperada','no_vencido','recibo_registrado','recibo_recuperado','politica_inactiva','operacion_denegada','operacion_error')));

CREATE TABLE vec_autorizacion_atestada_v3.politicas_sello_periodico_v1(
 version bigint PRIMARY KEY CHECK(version>0), anterior_sha256 text NOT NULL CHECK(anterior_sha256 ~ '^[0-9a-f]{64}$'),
 huella_sha256 text NOT NULL UNIQUE CHECK(huella_sha256 ~ '^[0-9a-f]{64}$'), configuracion jsonb NOT NULL,
 registrada_en timestamptz(6) NOT NULL, auditoria_ref text NOT NULL UNIQUE
 REFERENCES vec_autorizacion_atestada_v3.auditoria_consumo_v3(auditoria_ref));
CREATE TABLE vec_autorizacion_atestada_v3.capturas_sello_periodico_v1(
 captura_ref text PRIMARY KEY CHECK(captura_ref ~ '^captura_[0-9a-f]{32}$'),
 configuracion_version bigint NOT NULL REFERENCES vec_autorizacion_atestada_v3.politicas_sello_periodico_v1(version),
 checkpoint jsonb NOT NULL, creada_en timestamptz(6) NOT NULL,
 auditoria_ref text NOT NULL UNIQUE REFERENCES vec_autorizacion_atestada_v3.auditoria_consumo_v3(auditoria_ref));
CREATE TABLE vec_autorizacion_atestada_v3.recibos_sello_periodico_v1(
 captura_ref text PRIMARY KEY REFERENCES vec_autorizacion_atestada_v3.capturas_sello_periodico_v1(captura_ref),
 recibo jsonb NOT NULL, recibo_texto text NOT NULL, recibo_sha256 text NOT NULL CHECK(recibo_sha256 ~ '^[0-9a-f]{64}$'),
 confirmado_en timestamptz(6) NOT NULL, auditoria_ref text NOT NULL UNIQUE
 REFERENCES vec_autorizacion_atestada_v3.auditoria_consumo_v3(auditoria_ref));
REVOKE ALL ON TABLE vec_autorizacion_atestada_v3.politicas_sello_periodico_v1,vec_autorizacion_atestada_v3.capturas_sello_periodico_v1,vec_autorizacion_atestada_v3.recibos_sello_periodico_v1 FROM PUBLIC;
CREATE FUNCTION vec_autorizacion_atestada_v3.rechazar_cambio_periodico_v1() RETURNS trigger
LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
BEGIN RAISE EXCEPTION 'sello_periodico_solo_adicion' USING ERRCODE='42501'; END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.rechazar_cambio_periodico_v1() FROM PUBLIC;
CREATE TRIGGER solo_adicion BEFORE UPDATE OR DELETE ON vec_autorizacion_atestada_v3.politicas_sello_periodico_v1
FOR EACH ROW EXECUTE FUNCTION vec_autorizacion_atestada_v3.rechazar_cambio_periodico_v1();
CREATE TRIGGER solo_adicion BEFORE UPDATE OR DELETE ON vec_autorizacion_atestada_v3.capturas_sello_periodico_v1
FOR EACH ROW EXECUTE FUNCTION vec_autorizacion_atestada_v3.rechazar_cambio_periodico_v1();
CREATE TRIGGER solo_adicion BEFORE UPDATE OR DELETE ON vec_autorizacion_atestada_v3.recibos_sello_periodico_v1
FOR EACH ROW EXECUTE FUNCTION vec_autorizacion_atestada_v3.rechazar_cambio_periodico_v1();

-- Helper privado. El ejecutor no puede fabricar eventos favorables.
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_operacion_periodica_v1(p_accion text,p_resultado text,p_motivo text,p_correlacion text,p_detalle jsonb)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE SET search_path=pg_catalog SET row_security=on AS $f$
DECLARE v_seq numeric;v_prev text;v_ref text;v_evento text;v_material text;v_hash text;v_fecha timestamptz(6);v_bytes bytea;v_valor text;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off' OR current_setting('TimeZone')<>'UTC'
 THEN RAISE EXCEPTION 'periodica_transaccion_invalida' USING ERRCODE='25000'; END IF;
 IF p_correlacion IS NULL OR p_correlacion !~ '^correlacion_[0-9a-f]{32}$' OR p_detalle IS NULL
 THEN RAISE EXCEPTION 'periodica_entrada_invalida' USING ERRCODE='22023'; END IF;
 SELECT secuencia,cabeza_sha256 INTO STRICT v_seq,v_prev FROM vec_autorizacion_atestada_v3.control_cadena_auditoria WHERE control_id FOR UPDATE;
 v_seq:=v_seq+1;v_fecha:=clock_timestamp();v_evento:='evento_'||replace(gen_random_uuid()::text,'-','');v_ref:='aud_v3_per_'||substr(v_evento,8);
 v_bytes:=vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.periodica.material.v1');
 FOREACH v_valor IN ARRAY ARRAY['operacion_tecnica_auditoria_periodica',v_evento,session_user::text,p_accion,'auditoria','auditoria_periodica:comun_interna','integridad_auditoria_periodica',p_resultado,p_motivo,'postgresql','operacion_tecnica_privada',p_correlacion,p_detalle::text] LOOP
  v_bytes:=v_bytes||vec_autorizacion_atestada_v3.encuadrar_mac(v_valor);
 END LOOP;
 v_material:=encode(sha256(v_bytes),'hex');
 v_hash:=encode(sha256(vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.periodica.eslabon.v1')||
 vec_autorizacion_atestada_v3.encuadrar_mac(v_seq::text)||vec_autorizacion_atestada_v3.encuadrar_mac(v_prev)||
 vec_autorizacion_atestada_v3.encuadrar_mac(v_ref)||vec_autorizacion_atestada_v3.encuadrar_mac(v_material)||
 vec_autorizacion_atestada_v3.encuadrar_mac(to_char(v_fecha AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))),'hex');
 INSERT INTO vec_autorizacion_atestada_v3.auditoria_consumo_v3(auditoria_ref,secuencia,anterior_sha256,huella_sha256,registrada_en,tipo_registro,evento_ref,evento_material_sha256,operador_login,accion,modulo_id,recurso_ref,finalidad_ref,resultado,motivo_ref,proceso,canal,correlacion_ref,periodica_detalle)
 VALUES(v_ref,v_seq,v_prev,v_hash,v_fecha,'operacion_tecnica_auditoria_periodica',v_evento,v_material,session_user::name,p_accion,'auditoria','auditoria_periodica:comun_interna','integridad_auditoria_periodica',p_resultado,p_motivo,'postgresql','operacion_tecnica_privada',p_correlacion,p_detalle);
 UPDATE vec_autorizacion_atestada_v3.control_cadena_auditoria SET secuencia=v_seq,cabeza_sha256=v_hash,actualizada_en=v_fecha WHERE control_id;
 RETURN jsonb_build_object('auditoria_ref',v_ref,'secuencia',v_seq,'huella_sha256',v_hash,'registrada_en',to_char(v_fecha AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'correlacion_ref',p_correlacion);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_operacion_periodica_v1(text,text,text,text,jsonb) FROM PUBLIC;

CREATE FUNCTION vec_autorizacion_atestada_v3.configurar_sello_periodico_v1(p_configuracion jsonb,p_preimagen text,p_correlacion text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' SET statement_timeout='10s' AS $f$
DECLARE v_anterior record;v_sha text;v_version bigint;v_acuse jsonb;v_p jsonb;v_k text;
BEGIN
 IF pg_has_role(session_user,'vec_autorizacion_atestada_v3_propietario','MEMBER')
 OR pg_has_role(session_user,'vec_autorizacion_propietario','MEMBER')
 OR NOT pg_has_role(session_user,'vec_auditoria_periodica_configurador','MEMBER')
 THEN RAISE EXCEPTION 'periodica_autoridad_denegada' USING ERRCODE='42501'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:periodica',0));
 IF jsonb_typeof(p_configuracion) IS DISTINCT FROM 'object' OR octet_length(p_configuracion::text)>4096
 OR NOT p_configuracion ?& ARRAY['version','cadena_id','intervalo_segundos','activa','politica','pin_spki_sha256']
 OR (SELECT count(*) FROM jsonb_object_keys(p_configuracion))<>6
 OR jsonb_typeof(p_configuracion->'version') IS DISTINCT FROM 'number'
 OR (p_configuracion->>'version') !~ '^[1-9][0-9]{0,8}$'
 OR jsonb_typeof(p_configuracion->'cadena_id') IS DISTINCT FROM 'string'
 OR (p_configuracion->>'cadena_id') !~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,159}$'
 OR jsonb_typeof(p_configuracion->'intervalo_segundos') IS DISTINCT FROM 'number'
 OR (p_configuracion->>'intervalo_segundos') !~ '^[1-9][0-9]{0,7}$'
 OR (p_configuracion->>'intervalo_segundos')::bigint>31536000
 OR jsonb_typeof(p_configuracion->'activa') IS DISTINCT FROM 'boolean'
 OR jsonb_typeof(p_configuracion->'pin_spki_sha256') IS DISTINCT FROM 'string'
 OR (p_configuracion->>'pin_spki_sha256') !~ '^[0-9a-f]{64}$'
 OR (p_configuracion->>'pin_spki_sha256')=repeat('0',64)
 THEN RAISE EXCEPTION 'periodica_configuracion_invalida' USING ERRCODE='22023'; END IF;
 v_p:=p_configuracion->'politica';
 IF jsonb_typeof(v_p) IS DISTINCT FROM 'object'
 OR NOT v_p ?& ARRAY['version','politica_ref','politica_version','clave_ref','clave_version','proveedor_kms','proveedor_kms_version','proveedor_tsa','proveedor_tsa_version','operacion_tsa','modo']
 OR (SELECT count(*) FROM jsonb_object_keys(v_p))<>11 OR jsonb_typeof(v_p->'version') IS DISTINCT FROM 'number' OR v_p->>'modo' IS DISTINCT FROM 'DESARROLLO' OR v_p->>'version' IS DISTINCT FROM '1'
 THEN RAISE EXCEPTION 'periodica_politica_invalida' USING ERRCODE='22023'; END IF;
 FOREACH v_k IN ARRAY ARRAY['politica_ref','clave_ref','proveedor_kms','proveedor_tsa','operacion_tsa'] LOOP
  IF jsonb_typeof(v_p->v_k) IS DISTINCT FROM 'string' OR (v_p->>v_k) !~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,159}$'
  THEN RAISE EXCEPTION 'periodica_politica_invalida' USING ERRCODE='22023'; END IF;
 END LOOP;
 FOREACH v_k IN ARRAY ARRAY['politica_version','clave_version','proveedor_kms_version','proveedor_tsa_version'] LOOP
  IF jsonb_typeof(v_p->v_k) IS DISTINCT FROM 'number' OR (v_p->>v_k) !~ '^[1-9][0-9]{0,8}$'
  THEN RAISE EXCEPTION 'periodica_politica_invalida' USING ERRCODE='22023'; END IF;
 END LOOP;
 SELECT * INTO v_anterior FROM vec_autorizacion_atestada_v3.politicas_sello_periodico_v1 ORDER BY version DESC LIMIT 1;
 v_version:=p_configuracion->>'version';
 IF p_preimagen IS DISTINCT FROM coalesce(v_anterior.huella_sha256,repeat('0',64)) OR v_version<>coalesce(v_anterior.version,0)+1
 THEN RAISE EXCEPTION 'periodica_preimagen_no_coincide' USING ERRCODE='40001'; END IF;
 IF v_anterior.version IS NOT NULL AND (p_configuracion->>'cadena_id' IS DISTINCT FROM v_anterior.configuracion->>'cadena_id')
 THEN RAISE EXCEPTION 'periodica_cadena_inmutable' USING ERRCODE='22023'; END IF;
 IF EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.capturas_sello_periodico_v1) AND
 (p_configuracion->'politica' IS DISTINCT FROM v_anterior.configuracion->'politica' OR p_configuracion->>'pin_spki_sha256' IS DISTINCT FROM v_anterior.configuracion->>'pin_spki_sha256')
 THEN RAISE EXCEPTION 'periodica_raiz_inmutable' USING ERRCODE='22023'; END IF;
 -- Un cambio nunca deja sin confirmación una captura emitida con otra política.
 IF EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.capturas_sello_periodico_v1 c LEFT JOIN vec_autorizacion_atestada_v3.recibos_sello_periodico_v1 r USING(captura_ref) WHERE r.captura_ref IS NULL)
 THEN RAISE EXCEPTION 'periodica_captura_pendiente' USING ERRCODE='55000'; END IF;
 v_sha:=encode(sha256(convert_to(p_configuracion::text,'UTF8')),'hex');
 v_acuse:=vec_autorizacion_atestada_v3.registrar_operacion_periodica_v1('configurar_sello_periodico_v1','permitido','politica_registrada',p_correlacion,jsonb_build_object('version',v_version,'anterior_sha256',p_preimagen,'configuracion_sha256',v_sha,'perfil_tecnico_ref','vec_auditoria_periodica_configurador'));
 INSERT INTO vec_autorizacion_atestada_v3.politicas_sello_periodico_v1 VALUES(v_version,p_preimagen,v_sha,p_configuracion,clock_timestamp(),v_acuse->>'auditoria_ref');
 RETURN jsonb_build_object('estado','configurada','configuracion_version',v_version,'configuracion_sha256',v_sha,'acuse',v_acuse);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.configurar_sello_periodico_v1(jsonb,text,text) FROM PUBLIC;

CREATE FUNCTION vec_autorizacion_atestada_v3.capturar_sello_periodico_v1(p_correlacion text,p_max_registros numeric)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' SET statement_timeout='10s' AS $f$
DECLARE v_cfg record;v_c record;v_ultimo record;v_a record;v_acuse jsonb;v_n numeric;v_h text;v_desde numeric;v_prev text;v_ref text;v_cp jsonb;
BEGIN
 IF pg_has_role(session_user,'vec_autorizacion_atestada_v3_propietario','MEMBER')
 OR pg_has_role(session_user,'vec_autorizacion_propietario','MEMBER')
 OR NOT pg_has_role(session_user,'vec_auditoria_periodica_sellador','MEMBER')
 THEN RAISE EXCEPTION 'periodica_autoridad_denegada' USING ERRCODE='42501'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:periodica',0));
 SELECT * INTO v_cfg FROM vec_autorizacion_atestada_v3.politicas_sello_periodico_v1 ORDER BY version DESC LIMIT 1;
 IF v_cfg.version IS NULL OR NOT (v_cfg.configuracion->>'activa')::boolean THEN
  v_acuse:=vec_autorizacion_atestada_v3.registrar_operacion_periodica_v1('capturar_sello_periodico_v1','denegado','politica_inactiva',p_correlacion,jsonb_build_object('perfil_tecnico_ref','vec_auditoria_periodica_sellador'));
  RETURN jsonb_build_object('estado','denegado','acuse',v_acuse);
 END IF;
 SELECT c.* INTO v_c FROM vec_autorizacion_atestada_v3.capturas_sello_periodico_v1 c LEFT JOIN vec_autorizacion_atestada_v3.recibos_sello_periodico_v1 r USING(captura_ref) WHERE r.captura_ref IS NULL ORDER BY c.creada_en DESC LIMIT 1;
 IF v_c.captura_ref IS NOT NULL THEN
  PERFORM vec_autorizacion_atestada_v3.registrar_operacion_periodica_v1('capturar_sello_periodico_v1','permitido','captura_recuperada',p_correlacion,jsonb_build_object('captura_ref',v_c.captura_ref,'configuracion_sha256',v_cfg.huella_sha256,'perfil_tecnico_ref','vec_auditoria_periodica_sellador'));
  SELECT * INTO STRICT v_a FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 WHERE auditoria_ref=v_c.auditoria_ref;
  v_acuse:=jsonb_build_object('auditoria_ref',v_a.auditoria_ref,'secuencia',v_a.secuencia,'huella_sha256',v_a.huella_sha256,'registrada_en',to_char(v_a.registrada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'correlacion_ref',v_a.correlacion_ref);
  RETURN jsonb_build_object('estado','pendiente','captura_ref',v_c.captura_ref,'configuracion_version',v_c.configuracion_version,'configuracion_sha256',v_cfg.huella_sha256,'pin_spki_sha256',v_cfg.configuracion->>'pin_spki_sha256','checkpoint',v_c.checkpoint,'acuse',v_acuse);
 END IF;
 SELECT * INTO v_ultimo FROM vec_autorizacion_atestada_v3.capturas_sello_periodico_v1 ORDER BY (checkpoint->'cobertura'->>'ultima_secuencia')::numeric DESC LIMIT 1;
 IF v_ultimo.captura_ref IS NOT NULL AND clock_timestamp()<v_ultimo.creada_en+make_interval(secs=>(v_cfg.configuracion->>'intervalo_segundos')::integer) THEN
  v_acuse:=vec_autorizacion_atestada_v3.registrar_operacion_periodica_v1('capturar_sello_periodico_v1','permitido','no_vencido',p_correlacion,jsonb_build_object('configuracion_sha256',v_cfg.huella_sha256,'perfil_tecnico_ref','vec_auditoria_periodica_sellador'));
  RETURN jsonb_build_object('estado','no_vencido','acuse',v_acuse);
 END IF;
 SELECT secuencia,cabeza_sha256 INTO STRICT v_n,v_h FROM vec_autorizacion_atestada_v3.control_cadena_auditoria WHERE control_id FOR UPDATE;
 v_desde:=coalesce((v_ultimo.checkpoint->'cobertura'->>'ultima_secuencia')::numeric,0)+1;
 v_prev:=coalesce(v_ultimo.checkpoint->'cobertura'->>'cabeza_sha256',repeat('0',64));
 IF p_max_registros IS NULL OR p_max_registros<>trunc(p_max_registros) OR p_max_registros<1 OR p_max_registros>9007199254740991 OR v_n-v_desde+2>p_max_registros
 THEN RAISE EXCEPTION 'periodica_limite_cobertura' USING ERRCODE='22023'; END IF;
 v_ref:='captura_'||replace(gen_random_uuid()::text,'-','');
 v_acuse:=vec_autorizacion_atestada_v3.registrar_operacion_periodica_v1('capturar_sello_periodico_v1','permitido','captura_registrada',p_correlacion,
 jsonb_build_object('captura_ref',v_ref,'configuracion_sha256',v_cfg.huella_sha256,'previa_secuencia',v_n,'previa_cabeza_sha256',v_h,'perfil_tecnico_ref','vec_auditoria_periodica_sellador'));
 v_cp:=jsonb_build_object('esquema','vec.auditoria.checkpoint.desarrollo.v1','politica',v_cfg.configuracion->'politica','cobertura',
 jsonb_build_object('cadena_id',v_cfg.configuracion->>'cadena_id','primera_secuencia',v_desde,'ultima_secuencia',v_acuse->'secuencia','registros',(v_acuse->>'secuencia')::numeric-v_desde+1,'anterior_sha256',v_prev,'cabeza_sha256',v_acuse->>'huella_sha256'));
 INSERT INTO vec_autorizacion_atestada_v3.capturas_sello_periodico_v1 VALUES(v_ref,v_cfg.version,v_cp,clock_timestamp(),v_acuse->>'auditoria_ref');
 RETURN jsonb_build_object('estado','pendiente','captura_ref',v_ref,'configuracion_version',v_cfg.version,'configuracion_sha256',v_cfg.huella_sha256,'pin_spki_sha256',v_cfg.configuracion->>'pin_spki_sha256','checkpoint',v_cp,'acuse',v_acuse);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.capturar_sello_periodico_v1(text,numeric) FROM PUBLIC;

CREATE FUNCTION vec_autorizacion_atestada_v3.confirmar_sello_periodico_v1(p_captura text,p_recibo_texto text,p_recibo_sha text,p_correlacion text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' SET statement_timeout='10s' AS $f$
DECLARE v_c record;v_cfg record;v_r record;v_acuse jsonb;v_a record;v_recibo jsonb;
BEGIN
 IF pg_has_role(session_user,'vec_autorizacion_atestada_v3_propietario','MEMBER')
 OR pg_has_role(session_user,'vec_autorizacion_propietario','MEMBER')
 OR NOT pg_has_role(session_user,'vec_auditoria_periodica_sellador','MEMBER')
 THEN RAISE EXCEPTION 'periodica_autoridad_denegada' USING ERRCODE='42501'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:periodica',0));
 SELECT * INTO v_cfg FROM vec_autorizacion_atestada_v3.politicas_sello_periodico_v1 ORDER BY version DESC LIMIT 1;
 SELECT * INTO v_c FROM vec_autorizacion_atestada_v3.capturas_sello_periodico_v1 WHERE captura_ref=p_captura;
 IF v_c.captura_ref IS NULL OR v_c.configuracion_version IS DISTINCT FROM v_cfg.version OR NOT (v_cfg.configuracion->>'activa')::boolean
 THEN RAISE EXCEPTION 'periodica_captura_denegada' USING ERRCODE='42501'; END IF;
 IF p_recibo_texto IS NULL OR octet_length(p_recibo_texto)>16384 OR p_recibo_sha IS DISTINCT FROM encode(sha256(convert_to(p_recibo_texto,'UTF8')),'hex')
 THEN RAISE EXCEPTION 'periodica_recibo_huella_invalida' USING ERRCODE='22023'; END IF;
 BEGIN v_recibo:=p_recibo_texto::jsonb; EXCEPTION WHEN invalid_text_representation THEN RAISE EXCEPTION 'periodica_recibo_invalido' USING ERRCODE='22023'; END;
 IF jsonb_typeof(v_recibo) IS DISTINCT FROM 'object'
 OR NOT v_recibo ?& ARRAY['checkpoint','tsa','pin_spki_sha256','firma_base64'] OR (SELECT count(*) FROM jsonb_object_keys(v_recibo))<>4
 OR v_recibo->'checkpoint' IS DISTINCT FROM v_c.checkpoint OR jsonb_typeof(v_recibo->'firma_base64') IS DISTINCT FROM 'string'
 OR length(v_recibo->>'firma_base64')<>88 OR v_recibo->>'firma_base64' !~ '^[A-Za-z0-9+/]{86}==$'
 OR jsonb_typeof(v_recibo->'pin_spki_sha256') IS DISTINCT FROM 'string'
 OR v_recibo->>'pin_spki_sha256' IS DISTINCT FROM v_cfg.configuracion->>'pin_spki_sha256'
 OR jsonb_typeof(v_recibo->'tsa') IS DISTINCT FROM 'object'
 OR NOT (v_recibo->'tsa') ?& ARRAY['referencia','huella_preimagen_sha256','huella_checkpoint_sha256','autoridad','esquema']
 OR (SELECT count(*) FROM jsonb_object_keys(v_recibo->'tsa'))<>5
 OR v_recibo->'tsa'->>'autoridad' IS DISTINCT FROM 'no_autoritativo'
 OR v_recibo->'tsa'->>'esquema' IS DISTINCT FROM 'vec.tsa.desarrollo.v1'
 OR jsonb_typeof(v_recibo->'tsa'->'referencia') IS DISTINCT FROM 'string'
 OR v_recibo->'tsa'->>'referencia' !~ '^tsa-desarrollo:hmac-sha256:[0-9a-f]{64}$'
 OR jsonb_typeof(v_recibo->'tsa'->'huella_preimagen_sha256') IS DISTINCT FROM 'string'
 OR v_recibo->'tsa'->>'huella_preimagen_sha256' !~ '^[0-9a-f]{64}$'
 OR jsonb_typeof(v_recibo->'tsa'->'huella_checkpoint_sha256') IS DISTINCT FROM 'string'
 OR v_recibo->'tsa'->>'huella_checkpoint_sha256' !~ '^[0-9a-f]{64}$' OR p_recibo_sha IS NULL OR p_recibo_sha !~ '^[0-9a-f]{64}$'
 THEN RAISE EXCEPTION 'periodica_recibo_invalido' USING ERRCODE='22023'; END IF;
 -- PostgreSQL coteja contenido, no acredita firma/TSA: lo hace el verificador externo.
 SELECT * INTO v_r FROM vec_autorizacion_atestada_v3.recibos_sello_periodico_v1 WHERE captura_ref=p_captura;
 IF v_r.captura_ref IS NOT NULL THEN
  IF v_r.recibo IS DISTINCT FROM v_recibo OR v_r.recibo_texto IS DISTINCT FROM p_recibo_texto OR v_r.recibo_sha256 IS DISTINCT FROM p_recibo_sha
  THEN RAISE EXCEPTION 'periodica_recibo_conflicto' USING ERRCODE='23505'; END IF;
  PERFORM vec_autorizacion_atestada_v3.registrar_operacion_periodica_v1('confirmar_sello_periodico_v1','permitido','recibo_recuperado',p_correlacion,jsonb_build_object('captura_ref',p_captura,'recibo_sha256',p_recibo_sha,'perfil_tecnico_ref','vec_auditoria_periodica_sellador'));
  SELECT * INTO STRICT v_a FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 WHERE auditoria_ref=v_r.auditoria_ref;
  v_acuse:=jsonb_build_object('auditoria_ref',v_a.auditoria_ref,'secuencia',v_a.secuencia,'huella_sha256',v_a.huella_sha256,'registrada_en',to_char(v_a.registrada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'correlacion_ref',v_a.correlacion_ref);
 ELSE
  v_acuse:=vec_autorizacion_atestada_v3.registrar_operacion_periodica_v1('confirmar_sello_periodico_v1','permitido','recibo_registrado',p_correlacion,jsonb_build_object('captura_ref',p_captura,'recibo_sha256',p_recibo_sha,'perfil_tecnico_ref','vec_auditoria_periodica_sellador'));
  INSERT INTO vec_autorizacion_atestada_v3.recibos_sello_periodico_v1 VALUES(p_captura,v_recibo,p_recibo_texto,p_recibo_sha,clock_timestamp(),v_acuse->>'auditoria_ref');
 END IF;
 RETURN v_acuse||jsonb_build_object('captura_ref',p_captura,'recibo_huella_sha256',p_recibo_sha);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.confirmar_sello_periodico_v1(text,text,text,text) FROM PUBLIC;

CREATE FUNCTION vec_autorizacion_atestada_v3.recuperar_sello_periodico_v1(p_captura text,p_correlacion text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' SET statement_timeout='10s' AS $f$
DECLARE v_cfg record;v_original record;v_c record;v_r record;v_a record;v_acuse jsonb;v_confirmacion jsonb;v_salida jsonb;
BEGIN
 IF pg_has_role(session_user,'vec_autorizacion_atestada_v3_propietario','MEMBER')
 OR pg_has_role(session_user,'vec_autorizacion_propietario','MEMBER')
 OR NOT pg_has_role(session_user,'vec_auditoria_periodica_sellador','MEMBER')
 THEN RAISE EXCEPTION 'periodica_autoridad_denegada' USING ERRCODE='42501'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:periodica',0));
 SELECT * INTO v_cfg FROM vec_autorizacion_atestada_v3.politicas_sello_periodico_v1 ORDER BY version DESC LIMIT 1;
 SELECT * INTO v_c FROM vec_autorizacion_atestada_v3.capturas_sello_periodico_v1 WHERE captura_ref=p_captura;
 IF v_c.captura_ref IS NULL OR v_cfg.version IS NULL OR NOT (v_cfg.configuracion->>'activa')::boolean
 THEN RAISE EXCEPTION 'periodica_recuperacion_denegada' USING ERRCODE='42501'; END IF;
 SELECT * INTO v_r FROM vec_autorizacion_atestada_v3.recibos_sello_periodico_v1 WHERE captura_ref=p_captura;
 PERFORM vec_autorizacion_atestada_v3.registrar_operacion_periodica_v1('capturar_sello_periodico_v1','permitido','captura_recuperada',p_correlacion,jsonb_build_object('captura_ref',p_captura,'configuracion_sha256',v_cfg.huella_sha256,'perfil_tecnico_ref','vec_auditoria_periodica_sellador'));
 SELECT * INTO STRICT v_a FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 WHERE auditoria_ref=v_c.auditoria_ref;
 v_acuse:=jsonb_build_object('auditoria_ref',v_a.auditoria_ref,'secuencia',v_a.secuencia,'huella_sha256',v_a.huella_sha256,'registrada_en',to_char(v_a.registrada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'correlacion_ref',v_a.correlacion_ref);
 SELECT * INTO STRICT v_original FROM vec_autorizacion_atestada_v3.politicas_sello_periodico_v1 WHERE version=v_c.configuracion_version;
 v_salida:=jsonb_build_object('estado','pendiente','captura_ref',p_captura,'configuracion_version',v_c.configuracion_version,'configuracion_sha256',v_original.huella_sha256,'pin_spki_sha256',v_original.configuracion->>'pin_spki_sha256','checkpoint',v_c.checkpoint,'acuse',v_acuse);
 IF v_r.captura_ref IS NOT NULL THEN
  SELECT * INTO STRICT v_a FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 WHERE auditoria_ref=v_r.auditoria_ref;
  v_confirmacion:=jsonb_build_object('auditoria_ref',v_a.auditoria_ref,'secuencia',v_a.secuencia,'huella_sha256',v_a.huella_sha256,'registrada_en',to_char(v_a.registrada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'correlacion_ref',v_a.correlacion_ref,'captura_ref',p_captura,'recibo_huella_sha256',v_r.recibo_sha256);
  v_salida:=v_salida||jsonb_build_object('estado','confirmado','recibo',v_r.recibo,'recibo_texto',v_r.recibo_texto,'acuse_confirmacion',v_confirmacion);
 END IF;
 RETURN v_salida;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.recuperar_sello_periodico_v1(text,text) FROM PUBLIC;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_intento_periodico_v1(p_accion text,p_resultado text,p_correlacion text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' SET statement_timeout='10s' AS $f$
DECLARE v_perfil text;
BEGIN
 v_perfil:=CASE WHEN p_accion='configurar_sello_periodico_v1' THEN 'vec_auditoria_periodica_configurador' ELSE 'vec_auditoria_periodica_sellador' END;
 IF p_accion IS NULL OR p_resultado IS NULL OR p_accion NOT IN('configurar_sello_periodico_v1','capturar_sello_periodico_v1','confirmar_sello_periodico_v1') OR p_resultado NOT IN('denegado','error')
 OR pg_has_role(session_user,'vec_autorizacion_atestada_v3_propietario','MEMBER')
 OR pg_has_role(session_user,'vec_autorizacion_propietario','MEMBER')
 OR NOT pg_has_role(session_user,v_perfil,'MEMBER')
 THEN RAISE EXCEPTION 'periodica_intento_denegado' USING ERRCODE='42501'; END IF;
 RETURN vec_autorizacion_atestada_v3.registrar_operacion_periodica_v1(p_accion,p_resultado,CASE WHEN p_resultado='denegado' THEN 'operacion_denegada' ELSE 'operacion_error' END,p_correlacion,jsonb_build_object('perfil_tecnico_ref',v_perfil));
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_intento_periodico_v1(text,text,text) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_auditoria_periodica_configurador,vec_auditoria_periodica_sellador;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.configurar_sello_periodico_v1(jsonb,text,text),vec_autorizacion_atestada_v3.registrar_intento_periodico_v1(text,text,text) TO vec_auditoria_periodica_configurador;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.capturar_sello_periodico_v1(text,numeric),vec_autorizacion_atestada_v3.confirmar_sello_periodico_v1(text,text,text,text),vec_autorizacion_atestada_v3.recuperar_sello_periodico_v1(text,text),vec_autorizacion_atestada_v3.registrar_intento_periodico_v1(text,text,text) TO vec_auditoria_periodica_sellador;
COMMIT;
