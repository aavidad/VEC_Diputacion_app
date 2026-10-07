\set ON_ERROR_STOP on
-- AD215: auditoría común del alta sintética de persona y cuenta ordinaria AUT57.
-- Preimagen del CHECK medida por dirección en PG18 aislado: misma huella
-- antes y después de AD214. AD215 depende de ese CHECK y AD207, no de AD214.
-- Los intentos (incluidos replay, denegación y recuperación) son asientos propios;
-- AUT57 los confirma después de deshacer el subbloque fallido, sin lanzar después
-- una excepción que borre la auditoría. No contiene DNI, nombre, HMAC ni secretos.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion_atestada_v3:migracion:000215',0));
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
DO $pre$
DECLARE actual text;
BEGIN
 IF pg_catalog.current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
 THEN RAISE EXCEPTION 'AD215: migrador_PG18_no_acreditado' USING ERRCODE='42501'; END IF;
 IF pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.reservar_asiento_auditoria_v5()') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.encuadrar_mac(text)') IS NULL
 OR pg_catalog.to_regrole('vec_autorizacion_propietario') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_provision_identidad_interna_sintetica_v1(jsonb)') IS NOT NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_intento_identidad_interna_sintetica_v1(jsonb)') IS NOT NULL
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_attribute WHERE attrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
 AND NOT attisdropped AND attname IN ('identidad_operacion_ref','identidad_plan_ref','identidad_preimagen_sha256',
 'identidad_configuracion_sha256','identidad_alcance','identidad_solicitud_sha256'))
 OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_trigger WHERE tgrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
 AND tgname='encolar_sellado_ad207' AND tgenabled='O' AND NOT tgisinternal
 AND tgfoid=pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.encolar_asiento_auditoria_v5()'))
 THEN RAISE EXCEPTION 'AD215: preimagen_incompatible' USING ERRCODE='55000'; END IF;
 SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.pg_get_functiondef(p.oid),'UTF8')),'hex')
 INTO STRICT actual FROM pg_catalog.pg_proc p
 WHERE p.oid=pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.reservar_asiento_auditoria_v5()')
 AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND NOT p.prosecdef AND p.provolatile='v' AND p.proparallel='u';
 IF actual IS DISTINCT FROM '637fdca8b560cd6d043006f9aac6d35c263a1e7edf0d2b9c881adfe665b74aa3'
 THEN RAISE EXCEPTION 'AD215: reserva_AD207_incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
LOCK TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 IN ACCESS EXCLUSIVE MODE;
-- Medición y sustitución en el mismo bloqueo, sin aceptar otra variante.
DO $check_pre$
DECLARE actual text;esperado_check_sha256 constant text:='506b974efe29bb9dbac5ecb775338d7031c5dcc7977d7d31ea68c51615a6d088';
BEGIN
 SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.pg_get_constraintdef(c.oid,false),'UTF8')),'hex')
 INTO STRICT actual FROM pg_catalog.pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
 AND c.conname='auditoria_tipo_disjunto_v4' AND c.contype='c' AND c.convalidated;
 IF esperado_check_sha256 IS NULL OR actual IS DISTINCT FROM esperado_check_sha256
 THEN RAISE EXCEPTION 'AD215: CHECK_preimagen_no_acreditada' USING ERRCODE='55000'; END IF;
END $check_pre$;
ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3
 ADD COLUMN identidad_operacion_ref text,
 ADD COLUMN identidad_plan_ref text,
 ADD COLUMN identidad_preimagen_sha256 text,
 ADD COLUMN identidad_configuracion_sha256 text,
 ADD COLUMN identidad_alcance text,
 ADD COLUMN identidad_solicitud_sha256 text;
DO $familia$
DECLARE anterior text;nuevo text;nulas text;
 propias constant text[]:=ARRAY['auditoria_ref','secuencia','anterior_sha256','huella_sha256','registrada_en','tipo_registro',
 'evento_ref','evento_material_sha256','operador_login','plan_sha256','aprobacion_ref','fuente_ref','fuente_sha256',
 'accion','modulo_id','recurso_ref','finalidad_ref','resultado','motivo_ref','proceso','canal','correlacion_ref',
 'identidad_operacion_ref','identidad_plan_ref','identidad_preimagen_sha256','identidad_configuracion_sha256','identidad_alcance','identidad_solicitud_sha256'];
BEGIN
 SELECT pg_catalog.pg_get_constraintdef(c.oid,false) INTO STRICT anterior FROM pg_catalog.pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass AND c.conname='auditoria_tipo_disjunto_v4';
 IF pg_catalog.left(anterior,7)<>'CHECK (' OR pg_catalog.right(anterior,1)<>')'
 THEN RAISE EXCEPTION 'AD215: CHECK_formato_incompatible' USING ERRCODE='55000'; END IF;
 SELECT pg_catalog.string_agg(pg_catalog.format('%I IS NULL',a.attname),' AND ' ORDER BY a.attnum) INTO STRICT nulas
 FROM pg_catalog.pg_attribute a WHERE a.attrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
 AND a.attnum>0 AND NOT a.attisdropped AND NOT a.attname=ANY(propias);
 IF nulas IS NULL OR nulas NOT LIKE '%transaccion_origen IS NULL%' OR nulas NOT LIKE '%actor_ref IS NULL%'
 OR nulas NOT LIKE '%perfiles_asignables_detalle IS NULL%' OR nulas NOT LIKE '%decision_ref IS NULL%'
 THEN RAISE EXCEPTION 'AD215: columnas_previas_incompatibles' USING ERRCODE='55000'; END IF;
 nuevo:='CHECK ((identidad_operacion_ref IS NULL AND identidad_plan_ref IS NULL AND identidad_preimagen_sha256 IS NULL'
 ||' AND identidad_configuracion_sha256 IS NULL AND identidad_alcance IS NULL AND identidad_solicitud_sha256 IS NULL AND ('
 ||pg_catalog.substr(anterior,8,pg_catalog.length(anterior)-8)||')) OR ('
 ||'tipo_registro IN (''provision_identidad_interna_sintetica'',''intento_identidad_interna_sintetica'') AND '||nulas||$tipo$
 AND evento_ref IS NOT NULL AND evento_material_sha256 IS NOT NULL AND operador_login IS NOT NULL
 AND accion IS NOT NULL AND modulo_id IS NOT DISTINCT FROM 'administracion'
 AND proceso IS NOT DISTINCT FROM 'postgresql' AND canal IS NOT DISTINCT FROM 'operacion_tecnica_privada'
 AND finalidad_ref IS NOT DISTINCT FROM 'identidad_interna_sintetica'
 AND recurso_ref IS NOT NULL AND correlacion_ref IS NOT NULL AND resultado IS NOT NULL AND motivo_ref IS NOT NULL
 AND ((tipo_registro='provision_identidad_interna_sintetica'
 AND accion IS NOT DISTINCT FROM 'provisionar_identidad_interna_sintetica_v1'
 AND resultado IS NOT DISTINCT FROM 'permitido' AND motivo_ref IS NOT DISTINCT FROM 'identidad_interna_registrada'
 AND identidad_operacion_ref IS NOT NULL AND identidad_plan_ref IS NOT NULL AND identidad_preimagen_sha256 IS NOT NULL
 AND identidad_configuracion_sha256 IS NOT NULL AND identidad_alcance IS NOT DISTINCT FROM 'sintetico_declarado'
 AND plan_sha256 IS NOT NULL AND aprobacion_ref IS NOT NULL AND fuente_ref IS NOT NULL AND fuente_sha256 IS NOT NULL
 AND identidad_solicitud_sha256 IS NULL
 AND recurso_ref IS NOT DISTINCT FROM 'identidad_interna_sintetica:'||identidad_operacion_ref)
 OR (tipo_registro='intento_identidad_interna_sintetica'
 AND accion IN ('provisionar_identidad_interna_sintetica_v1','recuperar_identidad_interna_sintetica_v1')
 AND identidad_solicitud_sha256 IS NOT NULL AND identidad_operacion_ref IS NULL AND identidad_plan_ref IS NULL
 AND identidad_preimagen_sha256 IS NULL AND identidad_configuracion_sha256 IS NULL AND identidad_alcance IS NULL
 AND plan_sha256 IS NULL AND aprobacion_ref IS NULL AND fuente_ref IS NULL AND fuente_sha256 IS NULL))
$tipo$||'))';
 ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 DROP CONSTRAINT auditoria_tipo_disjunto_v4;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 ADD CONSTRAINT auditoria_tipo_disjunto_v4 '||nuevo;
END $familia$;

CREATE FUNCTION vec_autorizacion_atestada_v3.motivo_intento_identidad_interna_valido_v1(p_accion text,p_resultado text,p_motivo text)
RETURNS boolean LANGUAGE sql IMMUTABLE STRICT PARALLEL SAFE SET search_path=pg_catalog,pg_temp
AS $catalogo$
 SELECT EXISTS(SELECT 1 FROM (VALUES
 ('provisionar_identidad_interna_sintetica_v1','permitido','identidad_interna_registrada'),
 ('provisionar_identidad_interna_sintetica_v1','permitido','identidad_interna_replay'),
 ('recuperar_identidad_interna_sintetica_v1','permitido','identidad_interna_recuperada'),
 ('provisionar_identidad_interna_sintetica_v1','denegado','identidad_interna_denegada'),
 ('recuperar_identidad_interna_sintetica_v1','denegado','identidad_interna_denegada'),
 ('provisionar_identidad_interna_sintetica_v1','error','identidad_interna_error'),
 ('recuperar_identidad_interna_sintetica_v1','error','identidad_interna_error')) AS m(accion,resultado,motivo)
 WHERE m.accion=p_accion AND m.resultado=p_resultado AND m.motivo=p_motivo)
$catalogo$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.motivo_intento_identidad_interna_valido_v1(text,text,text) FROM PUBLIC;
ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3
 ADD CONSTRAINT auditoria_identidad_interna_formato_v1 CHECK (
 tipo_registro NOT IN ('provision_identidad_interna_sintetica','intento_identidad_interna_sintetica') OR (
 evento_ref ~ '^evento_[0-9a-f]{32}$' AND evento_material_sha256 ~ '^[0-9a-f]{64}$'
 AND correlacion_ref ~ '^correlacion_[0-9a-f]{32}$'
 AND ((tipo_registro='provision_identidad_interna_sintetica'
 AND identidad_operacion_ref ~ '^piis_[A-Za-z0-9_-]{22,123}$'
 AND identidad_plan_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'
 AND plan_sha256 ~ '^[0-9a-f]{64}$' AND identidad_preimagen_sha256 ~ '^[0-9a-f]{64}$'
 AND identidad_configuracion_sha256 ~ '^[0-9a-f]{64}$'
 AND aprobacion_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'
 AND fuente_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$' AND fuente_sha256 ~ '^[0-9a-f]{64}$')
 OR (tipo_registro='intento_identidad_interna_sintetica'
 AND identidad_solicitud_sha256 ~ '^[0-9a-f]{64}$' AND recurso_ref ~ '^solicitud_identidad_interna:[0-9a-f]{32}$'
 AND vec_autorizacion_atestada_v3.motivo_intento_identidad_interna_valido_v1(accion,resultado,motivo_ref) IS TRUE))));
CREATE UNIQUE INDEX auditoria_identidad_interna_operacion_uq
 ON vec_autorizacion_atestada_v3.auditoria_consumo_v3(identidad_operacion_ref)
 WHERE tipo_registro='provision_identidad_interna_sintetica';

-- No se concede esta guarda ni los registradores al LOGIN. AUT57 es el único
-- llamador funcional; coteja su configuración y aprobación antes del efecto.
CREATE FUNCTION vec_autorizacion_atestada_v3.acreditar_operador_identidad_interna_v1()
RETURNS void LANGUAGE plpgsql VOLATILE SET search_path=pg_catalog,pg_temp AS $guarda$
DECLARE grupo oid:=pg_catalog.to_regrole('vec_identidad_interna_sintetica_ejecutor');
BEGIN
 IF grupo IS NULL
 OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=session_user AND rolcanlogin AND rolinherit
 AND NOT (rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls) AND rolconfig IS NULL)
 OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE oid=grupo AND NOT rolcanlogin AND NOT rolinherit
 AND NOT (rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls) AND rolconfig IS NULL)
 OR (SELECT count(*) FROM pg_catalog.pg_auth_members WHERE member=session_user::regrole)<>1
 OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members WHERE member=session_user::regrole AND roleid=grupo
 AND inherit_option AND NOT set_option AND NOT admin_option)
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members WHERE member=grupo)
 THEN RAISE EXCEPTION 'AD215: operador_tecnico_no_acreditado' USING ERRCODE='42501'; END IF;
END $guarda$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.acreditar_operador_identidad_interna_v1() FROM PUBLIC;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_provision_identidad_interna_sintetica_v1(p_evento jsonb)
RETURNS TABLE(auditoria_ref text,secuencia numeric,huella_sha256 text,correlacion_ref text,registrada_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s' SET statement_timeout='10s'
AS $funcion$
DECLARE
 v_orden constant text[]:=ARRAY['tipo_registro','evento_ref','operador_login','operacion_ref','plan_ref','plan_sha256',
  'preimagen_sha256','configuracion_sha256','aprobacion_ref','alcance_fuente','accion','recurso_ref',
  'resultado','motivo_ref','proceso','canal','finalidad_ref','correlacion_ref','fuente_ref','fuente_sha256'];
 v_claves text[];v_clave text;v_material bytea;v_material_sha text;v_anterior text;
 v_secuencia numeric;v_instante timestamptz(6);v_ref text;v_huella text;v_existente record;
BEGIN
 PERFORM vec_autorizacion_atestada_v3.acreditar_operador_identidad_interna_v1();
 IF pg_catalog.current_setting('transaction_isolation')<>'serializable'
 OR pg_catalog.current_setting('transaction_read_only')<>'off' OR pg_catalog.current_setting('TimeZone')<>'UTC'
 THEN RAISE EXCEPTION 'AD215: PARO clave=transaccion actual=incompatible esperado=serializable_rw_UTC' USING ERRCODE='25000'; END IF;
 IF pg_catalog.jsonb_typeof(p_evento) IS DISTINCT FROM 'object' OR pg_catalog.octet_length(p_evento::text)>8192
 THEN RAISE EXCEPTION 'AD215: PARO clave=evento actual=incompatible esperado=objeto_acotado' USING ERRCODE='22023'; END IF;
 SELECT pg_catalog.array_agg(k ORDER BY k COLLATE "C") INTO v_claves FROM pg_catalog.jsonb_object_keys(p_evento) k;
 IF v_claves IS DISTINCT FROM (SELECT pg_catalog.array_agg(k ORDER BY k COLLATE "C") FROM pg_catalog.unnest(v_orden) k)
 THEN RAISE EXCEPTION 'AD215: PARO clave=campos actual=incompatible esperado=ABI_exacto' USING ERRCODE='22023'; END IF;
 FOREACH v_clave IN ARRAY v_orden LOOP
  IF pg_catalog.jsonb_typeof(p_evento->v_clave) IS DISTINCT FROM 'string'
  OR pg_catalog.octet_length(p_evento->>v_clave) NOT BETWEEN 1 AND 200
  THEN RAISE EXCEPTION 'AD215: PARO clave=valor actual=incompatible esperado=string_acotado' USING ERRCODE='22023'; END IF;
 END LOOP;
 IF p_evento->>'tipo_registro' IS DISTINCT FROM 'provision_identidad_interna_sintetica'
 OR p_evento->>'operador_login' IS DISTINCT FROM session_user::text
 OR pg_catalog.octet_length(p_evento->>'operador_login')>63
 OR p_evento->>'evento_ref' !~ '^evento_[0-9a-f]{32}$'
 OR p_evento->>'operacion_ref' !~ '^piis_[A-Za-z0-9_-]{22,123}$'
 OR p_evento->>'plan_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'
 OR p_evento->>'plan_sha256' !~ '^[0-9a-f]{64}$'
 OR p_evento->>'preimagen_sha256' !~ '^[0-9a-f]{64}$'
 OR p_evento->>'configuracion_sha256' !~ '^[0-9a-f]{64}$'
 OR p_evento->>'aprobacion_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'
 OR p_evento->>'alcance_fuente' IS DISTINCT FROM 'sintetico_declarado'
 OR p_evento->>'accion' IS DISTINCT FROM 'provisionar_identidad_interna_sintetica_v1'
 OR p_evento->>'recurso_ref' IS DISTINCT FROM 'identidad_interna_sintetica:'||(p_evento->>'operacion_ref')
 OR p_evento->>'resultado' IS DISTINCT FROM 'permitido'
 OR p_evento->>'motivo_ref' IS DISTINCT FROM 'identidad_interna_registrada'
 OR p_evento->>'proceso' IS DISTINCT FROM 'postgresql'
 OR p_evento->>'canal' IS DISTINCT FROM 'operacion_tecnica_privada'
 OR p_evento->>'finalidad_ref' IS DISTINCT FROM 'identidad_interna_sintetica'
 OR p_evento->>'correlacion_ref' !~ '^correlacion_[0-9a-f]{32}$'
 OR p_evento->>'fuente_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'
 OR p_evento->>'fuente_sha256' !~ '^[0-9a-f]{64}$'
 THEN RAISE EXCEPTION 'AD215: PARO clave=semantica actual=incompatible esperado=provision_identidad_privada' USING ERRCODE='22023'; END IF;
 v_material:=vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.identidad-interna-sintetica.v1');
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
  IF v_existente.tipo_registro IS DISTINCT FROM 'provision_identidad_interna_sintetica'
  OR v_existente.evento_material_sha256 IS DISTINCT FROM v_material_sha
  THEN RAISE EXCEPTION 'AD215: PARO clave=replay actual=material_distinto esperado=material_original' USING ERRCODE='23505'; END IF;
  RETURN QUERY SELECT v_existente.auditoria_ref,v_existente.secuencia,v_existente.huella_sha256,
   v_existente.correlacion_ref,v_existente.registrada_en;
  RETURN;
 END IF;
 SELECT r.secuencia_previa,r.anterior_sha256 INTO STRICT v_secuencia,v_anterior
 FROM vec_autorizacion_atestada_v3.reservar_asiento_auditoria_v5() r;
 IF v_secuencia>=9007199254740991::numeric
 THEN RAISE EXCEPTION 'AD215: PARO clave=secuencia actual=limite esperado=entero_seguro' USING ERRCODE='22003'; END IF;
 v_secuencia:=v_secuencia+1;v_instante:=pg_catalog.clock_timestamp();
 v_ref:='aud_v3_ii_'||pg_catalog.substr(p_evento->>'evento_ref',8,32);
 v_huella:=pg_catalog.encode(pg_catalog.sha256(
  vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.eslabon.identidad-interna-sintetica.v1')||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_secuencia::text)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_anterior)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_ref)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_material_sha)||
  vec_autorizacion_atestada_v3.encuadrar_mac(pg_catalog.to_char(v_instante AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))),'hex');
 INSERT INTO vec_autorizacion_atestada_v3.auditoria_consumo_v3(
  auditoria_ref,secuencia,anterior_sha256,huella_sha256,registrada_en,tipo_registro,
  evento_ref,evento_material_sha256,fuente_ref,fuente_sha256,operador_login,plan_sha256,aprobacion_ref,
  identidad_operacion_ref,identidad_plan_ref,identidad_preimagen_sha256,identidad_configuracion_sha256,identidad_alcance,
  accion,modulo_id,recurso_ref,finalidad_ref,resultado,motivo_ref,proceso,canal,correlacion_ref)
 VALUES(v_ref,v_secuencia,v_anterior,v_huella,v_instante,'provision_identidad_interna_sintetica',
  p_evento->>'evento_ref',v_material_sha,p_evento->>'fuente_ref',p_evento->>'fuente_sha256',
  (p_evento->>'operador_login')::name,p_evento->>'plan_sha256',p_evento->>'aprobacion_ref',
  p_evento->>'operacion_ref',p_evento->>'plan_ref',p_evento->>'preimagen_sha256',p_evento->>'configuracion_sha256',p_evento->>'alcance_fuente',
  p_evento->>'accion','administracion',p_evento->>'recurso_ref',p_evento->>'finalidad_ref','permitido',
  p_evento->>'motivo_ref',p_evento->>'proceso',p_evento->>'canal',p_evento->>'correlacion_ref');
 -- AD207 encola el asiento en el disparador común; la cabecera queda congelada.
 RETURN QUERY SELECT v_ref,v_secuencia,v_huella,p_evento->>'correlacion_ref',v_instante;
END $funcion$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_provision_identidad_interna_sintetica_v1(jsonb) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_autorizacion_propietario;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_provision_identidad_interna_sintetica_v1(jsonb) TO vec_autorizacion_propietario;
DO $acl$
DECLARE f oid:=pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_provision_identidad_interna_sintetica_v1(jsonb)');
BEGIN
 IF EXISTS(SELECT 1 FROM pg_catalog.pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
  WHERE p.oid=f AND (a.grantee NOT IN ('vec_autorizacion_atestada_v3_propietario'::regrole,'vec_autorizacion_propietario'::regrole)
   OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
 THEN RAISE EXCEPTION 'AD215: PARO clave=ACL actual=ampliada esperado=owners_orquestador' USING ERRCODE='55000'; END IF;
END $acl$;
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_intento_identidad_interna_sintetica_v1(p_evento jsonb)
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
 PERFORM vec_autorizacion_atestada_v3.acreditar_operador_identidad_interna_v1();
 IF pg_catalog.current_setting('transaction_isolation')<>'serializable'
 OR pg_catalog.current_setting('transaction_read_only')<>'off' OR pg_catalog.current_setting('TimeZone')<>'UTC'
 THEN RAISE EXCEPTION 'AD215: PARO clave=transaccion actual=incompatible esperado=serializable_rw_UTC' USING ERRCODE='25000'; END IF;
 IF pg_catalog.jsonb_typeof(p_evento) IS DISTINCT FROM 'object' OR pg_catalog.octet_length(p_evento::text)>8192
 THEN RAISE EXCEPTION 'AD215: PARO clave=evento actual=incompatible esperado=objeto_acotado' USING ERRCODE='22023'; END IF;
 SELECT pg_catalog.array_agg(k ORDER BY k COLLATE "C") INTO v_claves FROM pg_catalog.jsonb_object_keys(p_evento) k;
 IF v_claves IS DISTINCT FROM (SELECT pg_catalog.array_agg(k ORDER BY k COLLATE "C") FROM pg_catalog.unnest(v_orden) k)
 THEN RAISE EXCEPTION 'AD215: PARO clave=campos actual=incompatible esperado=ABI_exacto' USING ERRCODE='22023'; END IF;
 FOREACH v_clave IN ARRAY v_orden LOOP
  IF pg_catalog.jsonb_typeof(p_evento->v_clave) IS DISTINCT FROM 'string'
  OR pg_catalog.octet_length(p_evento->>v_clave) NOT BETWEEN 1 AND 200
  THEN RAISE EXCEPTION 'AD215: PARO clave=valor actual=incompatible esperado=string_acotado' USING ERRCODE='22023'; END IF;
 END LOOP;
 IF p_evento->>'tipo_registro' IS DISTINCT FROM 'intento_identidad_interna_sintetica'
 OR p_evento->>'operador_login' IS DISTINCT FROM session_user::text
 OR pg_catalog.octet_length(p_evento->>'operador_login')>63
 OR p_evento->>'evento_ref' !~ '^evento_[0-9a-f]{32}$'
 OR p_evento->>'solicitud_sha256' !~ '^[0-9a-f]{64}$'
 OR p_evento->>'accion' NOT IN ('provisionar_identidad_interna_sintetica_v1','recuperar_identidad_interna_sintetica_v1')
 OR p_evento->>'recurso_ref' !~ '^solicitud_identidad_interna:[0-9a-f]{32}$'
 OR vec_autorizacion_atestada_v3.motivo_intento_identidad_interna_valido_v1(p_evento->>'accion',p_evento->>'resultado',p_evento->>'motivo_ref') IS DISTINCT FROM true
 OR p_evento->>'proceso' IS DISTINCT FROM 'postgresql'
 OR p_evento->>'canal' IS DISTINCT FROM 'operacion_tecnica_privada'
 OR p_evento->>'finalidad_ref' IS DISTINCT FROM 'identidad_interna_sintetica'
 OR p_evento->>'correlacion_ref' !~ '^correlacion_[0-9a-f]{32}$'
 THEN RAISE EXCEPTION 'AD215: PARO clave=semantica actual=incompatible esperado=provision_identidad_privada' USING ERRCODE='22023'; END IF;
 v_material:=vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.intento-identidad-interna-sintetica.v1');
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
  IF v_existente.tipo_registro IS DISTINCT FROM 'intento_identidad_interna_sintetica'
  OR v_existente.evento_material_sha256 IS DISTINCT FROM v_material_sha
  THEN RAISE EXCEPTION 'AD215: PARO clave=replay actual=material_distinto esperado=material_original' USING ERRCODE='23505'; END IF;
  RETURN QUERY SELECT v_existente.auditoria_ref,v_existente.secuencia,v_existente.huella_sha256,
   v_existente.correlacion_ref,v_existente.registrada_en;
  RETURN;
 END IF;
 SELECT r.secuencia_previa,r.anterior_sha256 INTO STRICT v_secuencia,v_anterior
 FROM vec_autorizacion_atestada_v3.reservar_asiento_auditoria_v5() r;
 IF v_secuencia>=9007199254740991::numeric
 THEN RAISE EXCEPTION 'AD215: PARO clave=secuencia actual=limite esperado=entero_seguro' USING ERRCODE='22003'; END IF;
 v_secuencia:=v_secuencia+1;v_instante:=pg_catalog.clock_timestamp();
 v_ref:='aud_v3_iii_'||pg_catalog.substr(p_evento->>'evento_ref',8,32);
 v_huella:=pg_catalog.encode(pg_catalog.sha256(
  vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.eslabon.intento-identidad-interna-sintetica.v1')||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_secuencia::text)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_anterior)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_ref)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_material_sha)||
  vec_autorizacion_atestada_v3.encuadrar_mac(pg_catalog.to_char(v_instante AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))),'hex');
 INSERT INTO vec_autorizacion_atestada_v3.auditoria_consumo_v3(
  auditoria_ref,secuencia,anterior_sha256,huella_sha256,registrada_en,tipo_registro,
  evento_ref,evento_material_sha256,operador_login,identidad_solicitud_sha256,
  accion,modulo_id,recurso_ref,finalidad_ref,resultado,motivo_ref,proceso,canal,correlacion_ref)
 VALUES(v_ref,v_secuencia,v_anterior,v_huella,v_instante,'intento_identidad_interna_sintetica',
  p_evento->>'evento_ref',v_material_sha,(p_evento->>'operador_login')::name,p_evento->>'solicitud_sha256',
  p_evento->>'accion','administracion',p_evento->>'recurso_ref',p_evento->>'finalidad_ref',
  p_evento->>'resultado',p_evento->>'motivo_ref',p_evento->>'proceso',p_evento->>'canal',p_evento->>'correlacion_ref');
 -- AD207 encola el asiento en el disparador común; la cabecera queda congelada.
 RETURN QUERY SELECT v_ref,v_secuencia,v_huella,p_evento->>'correlacion_ref',v_instante;
END $funcion$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_intento_identidad_interna_sintetica_v1(jsonb) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_autorizacion_propietario;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_intento_identidad_interna_sintetica_v1(jsonb) TO vec_autorizacion_propietario;
DO $acl$
DECLARE f oid:=pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_intento_identidad_interna_sintetica_v1(jsonb)');
BEGIN
 IF EXISTS(SELECT 1 FROM pg_catalog.pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
  WHERE p.oid=f AND (a.grantee NOT IN ('vec_autorizacion_atestada_v3_propietario'::regrole,'vec_autorizacion_propietario'::regrole)
   OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
 THEN RAISE EXCEPTION 'AD215: PARO clave=ACL actual=ampliada esperado=owners_orquestador' USING ERRCODE='55000'; END IF;
END $acl$;
DO $acl_auxiliares$
BEGIN
 IF EXISTS(SELECT 1 FROM pg_catalog.pg_proc p CROSS JOIN LATERAL
 pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
 WHERE p.oid IN (
 pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.acreditar_operador_identidad_interna_v1()'),
 pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.motivo_intento_identidad_interna_valido_v1(text,text,text)'))
 AND (a.grantee<>p.proowner OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
 THEN RAISE EXCEPTION 'AD215: ACL_auxiliares_ampliada' USING ERRCODE='55000'; END IF;
END $acl_auxiliares$;
COMMIT;
