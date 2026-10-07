\set ON_ERROR_STOP on
-- AD219: auditoría común de la admisión gobernada del catálogo de acciones
-- administrativas (AUT58). No publica perfiles ni concede permisos. El
-- registro material y el intento conservan referencias y huellas, sin copiar
-- el catálogo completo. AD207 sella los asientos de esta tabla por su cola.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion_atestada_v3:migracion:000219',0));
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
DO $pre$
DECLARE v_reserva text;v_check text;
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
 OR pg_catalog.current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 THEN RAISE EXCEPTION 'AD219: migrador PG18 no acreditado' USING ERRCODE='42501'; END IF;
 IF pg_catalog.to_regrole('vec_autorizacion_propietario') IS NULL
 OR pg_catalog.to_regrole('vec_autorizacion_atestada_v3_propietario') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.encuadrar_mac(text)') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.reservar_asiento_auditoria_v5()') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_perfiles_asignables_admin_v1(jsonb)') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_intento_perfiles_asignables_admin_v1(jsonb)') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_provision_identidad_interna_sintetica_v1(jsonb)') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_intento_identidad_interna_sintetica_v1(jsonb)') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.consumir_carga_convoca_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 OR pg_catalog.to_regclass('vec_autorizacion_atestada_v3.pendiente_sellado_auditoria_v5') IS NULL
 OR pg_catalog.to_regclass('vec_autorizacion_atestada_v3.auditoria_consumo_v3') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_catalogo_acciones_admin_v1(jsonb)') IS NOT NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_intento_catalogo_acciones_admin_v1(jsonb)') IS NOT NULL
 THEN RAISE EXCEPTION 'AD219: preimagen causal POST215 incompatible' USING ERRCODE='55000'; END IF;
 IF (SELECT pg_catalog.count(*) FROM pg_catalog.pg_attribute a
  WHERE a.attrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::pg_catalog.regclass
  AND NOT a.attisdropped AND a.attname IN ('identidad_operacion_ref','identidad_plan_ref',
   'identidad_preimagen_sha256','identidad_configuracion_sha256','identidad_alcance',
   'identidad_solicitud_sha256'))<>6
 THEN RAISE EXCEPTION 'AD219: columnas AD215 ausentes' USING ERRCODE='55000'; END IF;
 SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.pg_get_functiondef(p.oid),'UTF8')),'hex')
 INTO STRICT v_reserva FROM pg_catalog.pg_proc p
 WHERE p.oid='vec_autorizacion_atestada_v3.reservar_asiento_auditoria_v5()'::pg_catalog.regprocedure
 AND p.proowner='vec_autorizacion_atestada_v3_propietario'::pg_catalog.regrole
 AND p.provolatile='v' AND NOT p.prosecdef AND p.proparallel='u';
 IF v_reserva IS DISTINCT FROM '637fdca8b560cd6d043006f9aac6d35c263a1e7edf0d2b9c881adfe665b74aa3'
 THEN RAISE EXCEPTION 'AD219: reserva AD207 incompatible' USING ERRCODE='55000'; END IF;
 SELECT pg_catalog.pg_get_constraintdef(c.oid,false) INTO STRICT v_check FROM pg_catalog.pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::pg_catalog.regclass
 AND c.conname='auditoria_tipo_disjunto_v4' AND c.contype='c' AND c.convalidated;
 IF pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(v_check,'UTF8')),'hex')
    IS DISTINCT FROM '1e9277a8496b0166c44eadb59865e561ac2707ad8329fd20f58618771b62eaed'
 OR pg_catalog.left(v_check,7)<>'CHECK (' OR pg_catalog.right(v_check,1)<>')'
 THEN RAISE EXCEPTION 'AD219: CHECK preimagen POST215 incompatible' USING ERRCODE='55000'; END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_trigger t
 WHERE t.tgrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::pg_catalog.regclass
 AND t.tgname='encolar_sellado_ad207' AND t.tgenabled='O')
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_attribute a
 WHERE a.attrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::pg_catalog.regclass
 AND NOT a.attisdropped AND a.attname IN ('catalogo_acciones_detalle','catalogo_acciones_solicitud_sha256'))
 THEN RAISE EXCEPTION 'AD219: tabla o sellado preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
LOCK TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 IN ACCESS EXCLUSIVE MODE;
ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3
 ADD COLUMN catalogo_acciones_detalle jsonb,
 ADD COLUMN catalogo_acciones_solicitud_sha256 text;
DO $familia$
DECLARE anterior text;nuevo text;nulas text;
 propias constant text[]:=ARRAY['auditoria_ref','secuencia','anterior_sha256','huella_sha256','registrada_en','tipo_registro',
  'evento_ref','evento_material_sha256','operador_login','plan_sha256','aprobacion_ref','fuente_ref','fuente_sha256',
  'accion','modulo_id','recurso_ref','finalidad_ref','resultado','motivo_ref','proceso','canal','correlacion_ref',
  'catalogo_acciones_detalle','catalogo_acciones_solicitud_sha256'];
BEGIN
 SELECT pg_catalog.pg_get_constraintdef(c.oid,false) INTO STRICT anterior FROM pg_catalog.pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::pg_catalog.regclass
 AND c.conname='auditoria_tipo_disjunto_v4' AND c.contype='c' AND c.convalidated;
 SELECT pg_catalog.string_agg(pg_catalog.format('%I IS NULL',a.attname),' AND ' ORDER BY a.attnum) INTO STRICT nulas
 FROM pg_catalog.pg_attribute a WHERE a.attrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::pg_catalog.regclass
 AND a.attnum>0 AND NOT a.attisdropped AND NOT a.attname=ANY(propias);
 IF nulas IS NULL OR nulas NOT LIKE '%identidad_operacion_ref IS NULL%'
 OR nulas NOT LIKE '%identidad_solicitud_sha256 IS NULL%'
 OR nulas NOT LIKE '%gobierno_usuarios_detalle IS NULL%'
 OR nulas NOT LIKE '%perfiles_asignables_detalle IS NULL%'
 OR nulas NOT LIKE '%decision_ref IS NULL%'
 OR nulas NOT LIKE '%transaccion_origen IS NULL%'
 THEN RAISE EXCEPTION 'AD219: columnas ajenas no acreditadas' USING ERRCODE='55000'; END IF;
 nuevo:='CHECK ((catalogo_acciones_detalle IS NULL AND catalogo_acciones_solicitud_sha256 IS NULL AND ('
 ||pg_catalog.substr(anterior,8,pg_catalog.length(anterior)-8)||')) OR ('
 ||'tipo_registro IN (''catalogo_acciones_admin'',''intento_catalogo_acciones_admin'') AND '||nulas||$tipo$
 AND evento_ref IS NOT NULL AND evento_material_sha256 IS NOT NULL AND operador_login IS NOT NULL
 AND accion IS NOT DISTINCT FROM 'registrar_catalogo_acciones_admin_v1'
 AND modulo_id IS NOT DISTINCT FROM 'administracion'
 AND proceso IS NOT DISTINCT FROM 'postgresql' AND canal IS NOT DISTINCT FROM 'operacion_tecnica_privada'
 AND finalidad_ref IS NOT DISTINCT FROM 'catalogo_acciones_admin'
 AND recurso_ref IS NOT NULL AND correlacion_ref IS NOT NULL AND resultado IS NOT NULL AND motivo_ref IS NOT NULL
 AND ((tipo_registro='catalogo_acciones_admin'
  AND plan_sha256 IS NOT NULL AND aprobacion_ref IS NOT NULL AND fuente_ref IS NOT NULL AND fuente_sha256 IS NOT NULL
  AND catalogo_acciones_detalle IS NOT NULL AND catalogo_acciones_solicitud_sha256 IS NULL
  AND resultado IS NOT DISTINCT FROM 'permitido' AND motivo_ref IS NOT DISTINCT FROM 'catalogo_acciones_registrado'
  AND recurso_ref IS NOT DISTINCT FROM 'catalogo_acciones_admin:'||(catalogo_acciones_detalle->>'operacion_ref'))
 OR (tipo_registro='intento_catalogo_acciones_admin'
  AND plan_sha256 IS NULL AND aprobacion_ref IS NULL AND fuente_ref IS NULL AND fuente_sha256 IS NULL
  AND catalogo_acciones_detalle IS NULL AND catalogo_acciones_solicitud_sha256 IS NOT NULL))
$tipo$||'))';
 ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 DROP CONSTRAINT auditoria_tipo_disjunto_v4;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 ADD CONSTRAINT auditoria_tipo_disjunto_v4 '||nuevo;
END $familia$;

CREATE FUNCTION vec_autorizacion_atestada_v3.detalle_catalogo_acciones_valido_v1(p jsonb)
RETURNS boolean LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE SET search_path=pg_catalog,pg_temp AS $f$
DECLARE k text;
 claves constant text[]:=ARRAY['operacion_ref','catalogo_ref','catalogo_version','catalogo_sha256',
  'paquete_version','censo_sha256','entradas_numero','perfiles_numero','aprobacion_sha256'];
BEGIN
 IF pg_catalog.jsonb_typeof(p) IS DISTINCT FROM 'object' THEN RETURN false; END IF;
 IF (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_object_keys(p))<>9
 OR NOT (p ?& claves) THEN RETURN false; END IF;
 FOREACH k IN ARRAY claves LOOP
  IF pg_catalog.jsonb_typeof(p->k) IS DISTINCT FROM 'string'
  OR pg_catalog.octet_length(p->>k) NOT BETWEEN 1 AND 128 THEN RETURN false; END IF;
 END LOOP;
 IF p->>'operacion_ref' !~ '^caa_[A-Za-z0-9_-]{22,123}$'
 OR p->>'catalogo_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'
 OR p->>'catalogo_sha256' !~ '^[0-9a-f]{64}$'
 OR p->>'censo_sha256' !~ '^[0-9a-f]{64}$'
 OR p->>'aprobacion_sha256' !~ '^[0-9a-f]{64}$'
 OR p->>'catalogo_version' !~ '^[1-9][0-9]{0,9}$'
 OR p->>'paquete_version' !~ '^[1-9][0-9]{0,9}$'
 OR p->>'entradas_numero' !~ '^[1-9][0-9]{0,2}$'
 OR p->>'perfiles_numero' !~ '^[1-9][0-9]{0,2}$'
 THEN RETURN false; END IF;
 RETURN (p->>'catalogo_version')::numeric<=2147483647
 AND (p->>'paquete_version')::numeric<=2147483647
 AND (p->>'entradas_numero')::integer<=512
 AND (p->>'perfiles_numero')::integer<=512;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.detalle_catalogo_acciones_valido_v1(jsonb) FROM PUBLIC;
CREATE FUNCTION vec_autorizacion_atestada_v3.motivo_intento_catalogo_acciones_valido_v1(p_resultado text,p_motivo text)
RETURNS boolean LANGUAGE sql IMMUTABLE STRICT PARALLEL SAFE SET search_path=pg_catalog,pg_temp AS $f$
 SELECT EXISTS(SELECT 1 FROM (VALUES
 ('permitido','catalogo_acciones_registrado'),('permitido','catalogo_acciones_replay'),
 ('denegado','catalogo_acciones_denegado'),('error','catalogo_acciones_error')) m(resultado,motivo_ref)
 WHERE m.resultado=p_resultado AND m.motivo_ref=p_motivo)
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.motivo_intento_catalogo_acciones_valido_v1(text,text) FROM PUBLIC;
ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3
 ADD CONSTRAINT auditoria_catalogo_acciones_formato_v1 CHECK (
 tipo_registro NOT IN('catalogo_acciones_admin','intento_catalogo_acciones_admin') OR (
 evento_ref ~ '^evento_[0-9a-f]{32}$' AND evento_material_sha256 ~ '^[0-9a-f]{64}$'
 AND correlacion_ref ~ '^correlacion_[0-9a-f]{32}$'
 AND ((tipo_registro='catalogo_acciones_admin'
  AND plan_sha256 ~ '^[0-9a-f]{64}$' AND aprobacion_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'
  AND fuente_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$' AND fuente_sha256 ~ '^[0-9a-f]{64}$'
  AND vec_autorizacion_atestada_v3.detalle_catalogo_acciones_valido_v1(catalogo_acciones_detalle))
 OR (tipo_registro='intento_catalogo_acciones_admin'
  AND catalogo_acciones_solicitud_sha256 ~ '^[0-9a-f]{64}$'
  AND recurso_ref ~ '^solicitud_catalogo_acciones_admin:[0-9a-f]{32}$'
  AND vec_autorizacion_atestada_v3.motivo_intento_catalogo_acciones_valido_v1(resultado,motivo_ref)))));

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_catalogo_acciones_admin_v1(p_evento jsonb)
RETURNS TABLE(auditoria_ref text,secuencia numeric,huella_sha256 text,correlacion_ref text,registrada_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s' SET statement_timeout='10s'
AS $funcion$
DECLARE
 v_orden constant text[]:=ARRAY['tipo_registro','evento_ref','operador_login','operacion_ref','plan_sha256',
  'catalogo_ref','catalogo_version','catalogo_sha256','paquete_ref','paquete_version','paquete_sha256',
  'censo_sha256','entradas_numero','perfiles_numero','aprobacion_ref','aprobacion_sha256',
  'proceso','canal','finalidad_ref','correlacion_ref'];
 v_claves text[];v_clave text;v_material bytea;v_material_sha text;v_anterior text;
 v_secuencia numeric;v_instante timestamptz(6);v_ref text;v_huella text;v_existente record;
 v_detalle jsonb;
BEGIN
 IF pg_catalog.current_setting('transaction_isolation')<>'serializable'
 OR pg_catalog.current_setting('transaction_read_only')<>'off'
 OR pg_catalog.current_setting('TimeZone')<>'UTC'
 THEN RAISE EXCEPTION 'AD219: transaccion incompatible' USING ERRCODE='25000'; END IF;
 IF pg_catalog.jsonb_typeof(p_evento) IS DISTINCT FROM 'object'
 OR pg_catalog.octet_length(p_evento::text)>8192
 THEN RAISE EXCEPTION 'AD219: evento invalido' USING ERRCODE='22023'; END IF;
 SELECT pg_catalog.array_agg(k ORDER BY k COLLATE "C") INTO v_claves FROM pg_catalog.jsonb_object_keys(p_evento) k;
 IF v_claves IS DISTINCT FROM (SELECT pg_catalog.array_agg(k ORDER BY k COLLATE "C") FROM pg_catalog.unnest(v_orden) k)
 THEN RAISE EXCEPTION 'AD219: ABI de evento incompatible' USING ERRCODE='22023'; END IF;
 FOREACH v_clave IN ARRAY v_orden LOOP
  IF pg_catalog.jsonb_typeof(p_evento->v_clave) IS DISTINCT FROM 'string'
  OR pg_catalog.octet_length(p_evento->>v_clave) NOT BETWEEN 1 AND 200
  THEN RAISE EXCEPTION 'AD219: valor de evento invalido' USING ERRCODE='22023'; END IF;
 END LOOP;
 v_detalle:=p_evento-ARRAY['tipo_registro','evento_ref','operador_login','plan_sha256','paquete_ref',
  'paquete_sha256','aprobacion_ref','proceso','canal','finalidad_ref','correlacion_ref'];
 IF p_evento->>'tipo_registro' IS DISTINCT FROM 'catalogo_acciones_admin'
 OR p_evento->>'operador_login' IS DISTINCT FROM session_user::text
 OR pg_catalog.octet_length(p_evento->>'operador_login')>63
 OR p_evento->>'evento_ref' !~ '^evento_[0-9a-f]{32}$'
 OR p_evento->>'plan_sha256' !~ '^[0-9a-f]{64}$'
 OR p_evento->>'paquete_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'
 OR p_evento->>'paquete_sha256' !~ '^[0-9a-f]{64}$'
 OR p_evento->>'aprobacion_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'
 OR vec_autorizacion_atestada_v3.detalle_catalogo_acciones_valido_v1(v_detalle) IS DISTINCT FROM true
 OR p_evento->>'proceso' IS DISTINCT FROM 'postgresql'
 OR p_evento->>'canal' IS DISTINCT FROM 'operacion_tecnica_privada'
 OR p_evento->>'finalidad_ref' IS DISTINCT FROM 'catalogo_acciones_admin'
 OR p_evento->>'correlacion_ref' !~ '^correlacion_[0-9a-f]{32}$'
 THEN RAISE EXCEPTION 'AD219: semantica de catalogo invalida' USING ERRCODE='22023'; END IF;
 v_material:=vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.catalogo-acciones-admin.v1');
 FOREACH v_clave IN ARRAY v_orden LOOP
  v_material:=v_material||vec_autorizacion_atestada_v3.encuadrar_mac(p_evento->>v_clave);
 END LOOP;
 v_material_sha:=pg_catalog.encode(pg_catalog.sha256(v_material),'hex');
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
  'vec_autorizacion_atestada_v3:evento-admin:'||(p_evento->>'evento_ref'),0));
 SELECT a.tipo_registro,a.auditoria_ref,a.secuencia,a.huella_sha256,a.correlacion_ref,a.registrada_en,
  a.evento_material_sha256 INTO v_existente FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 a
 WHERE a.evento_ref=p_evento->>'evento_ref';
 IF FOUND THEN
  IF v_existente.tipo_registro IS DISTINCT FROM 'catalogo_acciones_admin'
  OR v_existente.evento_material_sha256 IS DISTINCT FROM v_material_sha
  THEN RAISE EXCEPTION 'AD219: replay con material distinto' USING ERRCODE='23505'; END IF;
  RETURN QUERY SELECT v_existente.auditoria_ref,v_existente.secuencia,v_existente.huella_sha256,
   v_existente.correlacion_ref,v_existente.registrada_en;
  RETURN;
 END IF;
 SELECT r.secuencia_previa,r.anterior_sha256 INTO STRICT v_secuencia,v_anterior
 FROM vec_autorizacion_atestada_v3.reservar_asiento_auditoria_v5() r;
 IF v_secuencia>=9007199254740991::numeric
 THEN RAISE EXCEPTION 'AD219: secuencia agotada' USING ERRCODE='22003'; END IF;
 v_secuencia:=v_secuencia+1;v_instante:=pg_catalog.clock_timestamp();
 v_ref:='aud_v3_caa_'||pg_catalog.substr(p_evento->>'evento_ref',8,32);
 v_huella:=pg_catalog.encode(pg_catalog.sha256(
  vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.eslabon.catalogo-acciones-admin.v1')||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_secuencia::text)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_anterior)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_ref)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_material_sha)||
  vec_autorizacion_atestada_v3.encuadrar_mac(pg_catalog.to_char(v_instante AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))),'hex');
 INSERT INTO vec_autorizacion_atestada_v3.auditoria_consumo_v3(
  auditoria_ref,secuencia,anterior_sha256,huella_sha256,registrada_en,tipo_registro,
  evento_ref,evento_material_sha256,operador_login,plan_sha256,aprobacion_ref,fuente_ref,fuente_sha256,
  catalogo_acciones_detalle,accion,modulo_id,recurso_ref,finalidad_ref,resultado,motivo_ref,proceso,canal,correlacion_ref)
 VALUES(v_ref,v_secuencia,v_anterior,v_huella,v_instante,'catalogo_acciones_admin',
  p_evento->>'evento_ref',v_material_sha,(p_evento->>'operador_login')::name,p_evento->>'plan_sha256',
  p_evento->>'aprobacion_ref',p_evento->>'paquete_ref',p_evento->>'paquete_sha256',v_detalle,
  'registrar_catalogo_acciones_admin_v1','administracion','catalogo_acciones_admin:'||(p_evento->>'operacion_ref'),
  'catalogo_acciones_admin','permitido','catalogo_acciones_registrado',
  p_evento->>'proceso',p_evento->>'canal',p_evento->>'correlacion_ref');
 -- AD207 encola el asiento mediante su disparador; el control antiguo está congelado.
 RETURN QUERY SELECT v_ref,v_secuencia,v_huella,p_evento->>'correlacion_ref',v_instante;
END $funcion$;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_intento_catalogo_acciones_admin_v1(p_evento jsonb)
RETURNS TABLE(auditoria_ref text,secuencia numeric,huella_sha256 text,correlacion_ref text,registrada_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s' SET statement_timeout='10s'
AS $funcion$
DECLARE
 v_orden constant text[]:=ARRAY['tipo_registro','evento_ref','operador_login','solicitud_sha256',
  'accion','recurso_ref','resultado','motivo_ref','proceso','canal','finalidad_ref','correlacion_ref'];
 v_claves text[];v_clave text;v_material bytea;v_material_sha text;v_anterior text;
 v_secuencia numeric;v_instante timestamptz(6);v_ref text;v_huella text;v_existente record;
BEGIN
 IF pg_catalog.current_setting('transaction_isolation')<>'serializable'
 OR pg_catalog.current_setting('transaction_read_only')<>'off'
 OR pg_catalog.current_setting('TimeZone')<>'UTC'
 THEN RAISE EXCEPTION 'AD219: transaccion incompatible' USING ERRCODE='25000'; END IF;
 IF pg_catalog.jsonb_typeof(p_evento) IS DISTINCT FROM 'object'
 OR pg_catalog.octet_length(p_evento::text)>8192
 THEN RAISE EXCEPTION 'AD219: intento invalido' USING ERRCODE='22023'; END IF;
 SELECT pg_catalog.array_agg(k ORDER BY k COLLATE "C") INTO v_claves FROM pg_catalog.jsonb_object_keys(p_evento) k;
 IF v_claves IS DISTINCT FROM (SELECT pg_catalog.array_agg(k ORDER BY k COLLATE "C") FROM pg_catalog.unnest(v_orden) k)
 THEN RAISE EXCEPTION 'AD219: ABI de intento incompatible' USING ERRCODE='22023'; END IF;
 FOREACH v_clave IN ARRAY v_orden LOOP
  IF pg_catalog.jsonb_typeof(p_evento->v_clave) IS DISTINCT FROM 'string'
  OR pg_catalog.octet_length(p_evento->>v_clave) NOT BETWEEN 1 AND 200
  THEN RAISE EXCEPTION 'AD219: valor de intento invalido' USING ERRCODE='22023'; END IF;
 END LOOP;
 IF p_evento->>'tipo_registro' IS DISTINCT FROM 'intento_catalogo_acciones_admin'
 OR p_evento->>'operador_login' IS DISTINCT FROM session_user::text
 OR pg_catalog.octet_length(p_evento->>'operador_login')>63
 OR p_evento->>'evento_ref' !~ '^evento_[0-9a-f]{32}$'
 OR p_evento->>'solicitud_sha256' !~ '^[0-9a-f]{64}$'
 OR p_evento->>'accion' IS DISTINCT FROM 'registrar_catalogo_acciones_admin_v1'
 OR p_evento->>'recurso_ref' !~ '^solicitud_catalogo_acciones_admin:[0-9a-f]{32}$'
 OR vec_autorizacion_atestada_v3.motivo_intento_catalogo_acciones_valido_v1(p_evento->>'resultado',p_evento->>'motivo_ref') IS DISTINCT FROM true
 OR p_evento->>'proceso' IS DISTINCT FROM 'postgresql'
 OR p_evento->>'canal' IS DISTINCT FROM 'operacion_tecnica_privada'
 OR p_evento->>'finalidad_ref' IS DISTINCT FROM 'catalogo_acciones_admin'
 OR p_evento->>'correlacion_ref' !~ '^correlacion_[0-9a-f]{32}$'
 THEN RAISE EXCEPTION 'AD219: semantica de intento invalida' USING ERRCODE='22023'; END IF;
 v_material:=vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.intento-catalogo-acciones-admin.v1');
 FOREACH v_clave IN ARRAY v_orden LOOP
  v_material:=v_material||vec_autorizacion_atestada_v3.encuadrar_mac(p_evento->>v_clave);
 END LOOP;
 v_material_sha:=pg_catalog.encode(pg_catalog.sha256(v_material),'hex');
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
  'vec_autorizacion_atestada_v3:evento-admin:'||(p_evento->>'evento_ref'),0));
 SELECT a.tipo_registro,a.auditoria_ref,a.secuencia,a.huella_sha256,a.correlacion_ref,a.registrada_en,
  a.evento_material_sha256 INTO v_existente FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 a
 WHERE a.evento_ref=p_evento->>'evento_ref';
 IF FOUND THEN
  IF v_existente.tipo_registro IS DISTINCT FROM 'intento_catalogo_acciones_admin'
  OR v_existente.evento_material_sha256 IS DISTINCT FROM v_material_sha
  THEN RAISE EXCEPTION 'AD219: replay de intento con material distinto' USING ERRCODE='23505'; END IF;
  RETURN QUERY SELECT v_existente.auditoria_ref,v_existente.secuencia,v_existente.huella_sha256,
   v_existente.correlacion_ref,v_existente.registrada_en;
  RETURN;
 END IF;
 SELECT r.secuencia_previa,r.anterior_sha256 INTO STRICT v_secuencia,v_anterior
 FROM vec_autorizacion_atestada_v3.reservar_asiento_auditoria_v5() r;
 IF v_secuencia>=9007199254740991::numeric
 THEN RAISE EXCEPTION 'AD219: secuencia agotada' USING ERRCODE='22003'; END IF;
 v_secuencia:=v_secuencia+1;v_instante:=pg_catalog.clock_timestamp();
 v_ref:='aud_v3_caai_'||pg_catalog.substr(p_evento->>'evento_ref',8,32);
 v_huella:=pg_catalog.encode(pg_catalog.sha256(
  vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.eslabon.intento-catalogo-acciones-admin.v1')||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_secuencia::text)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_anterior)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_ref)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_material_sha)||
  vec_autorizacion_atestada_v3.encuadrar_mac(pg_catalog.to_char(v_instante AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))),'hex');
 INSERT INTO vec_autorizacion_atestada_v3.auditoria_consumo_v3(
  auditoria_ref,secuencia,anterior_sha256,huella_sha256,registrada_en,tipo_registro,
  evento_ref,evento_material_sha256,operador_login,catalogo_acciones_solicitud_sha256,
  accion,modulo_id,recurso_ref,finalidad_ref,resultado,motivo_ref,proceso,canal,correlacion_ref)
 VALUES(v_ref,v_secuencia,v_anterior,v_huella,v_instante,'intento_catalogo_acciones_admin',
  p_evento->>'evento_ref',v_material_sha,(p_evento->>'operador_login')::name,p_evento->>'solicitud_sha256',
  p_evento->>'accion','administracion',p_evento->>'recurso_ref',p_evento->>'finalidad_ref',
  p_evento->>'resultado',p_evento->>'motivo_ref',p_evento->>'proceso',p_evento->>'canal',p_evento->>'correlacion_ref');
 RETURN QUERY SELECT v_ref,v_secuencia,v_huella,p_evento->>'correlacion_ref',v_instante;
END $funcion$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_catalogo_acciones_admin_v1(jsonb) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_intento_catalogo_acciones_admin_v1(jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_catalogo_acciones_admin_v1(jsonb),
 vec_autorizacion_atestada_v3.registrar_intento_catalogo_acciones_admin_v1(jsonb) TO vec_autorizacion_propietario;
DO $acl$
DECLARE f pg_catalog.oid;n integer:=0;
BEGIN
 FOR f IN SELECT p.oid FROM pg_catalog.pg_proc p WHERE p.oid IN (
  'vec_autorizacion_atestada_v3.registrar_catalogo_acciones_admin_v1(jsonb)'::pg_catalog.regprocedure,
  'vec_autorizacion_atestada_v3.registrar_intento_catalogo_acciones_admin_v1(jsonb)'::pg_catalog.regprocedure) LOOP
  n:=n+1;
  IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_proc p WHERE p.oid=f
   AND p.proowner='vec_autorizacion_atestada_v3_propietario'::pg_catalog.regrole
   AND p.prosecdef AND p.provolatile='v' AND p.proparallel='u')
  OR EXISTS(SELECT 1 FROM pg_catalog.pg_proc p CROSS JOIN LATERAL
   pg_catalog.aclexplode(COALESCE(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
   WHERE p.oid=f AND (a.grantee NOT IN ('vec_autorizacion_atestada_v3_propietario'::pg_catalog.regrole,
    'vec_autorizacion_propietario'::pg_catalog.regrole) OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
  OR NOT pg_catalog.has_function_privilege('vec_autorizacion_propietario',f,'EXECUTE')
  THEN RAISE EXCEPTION 'AD219: ACL de escritor incompatible' USING ERRCODE='55000'; END IF;
 END LOOP;
 IF n<>2 OR EXISTS(SELECT 1 FROM pg_catalog.pg_proc p CROSS JOIN LATERAL
  pg_catalog.aclexplode(COALESCE(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
  WHERE p.oid IN (
   'vec_autorizacion_atestada_v3.detalle_catalogo_acciones_valido_v1(jsonb)'::pg_catalog.regprocedure,
   'vec_autorizacion_atestada_v3.motivo_intento_catalogo_acciones_valido_v1(text,text)'::pg_catalog.regprocedure)
  AND a.grantee<>p.proowner)
 THEN RAISE EXCEPTION 'AD219: ACL de validadores incompatible' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
