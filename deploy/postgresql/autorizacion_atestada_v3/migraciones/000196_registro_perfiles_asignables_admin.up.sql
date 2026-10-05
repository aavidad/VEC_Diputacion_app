\set ON_ERROR_STOP on
-- AD196: auditoría común del registro técnico de perfiles asignables (AUT49).
-- Dos tipos nuevos en auditoria_consumo_v3, con el patrón de AD183/AD188:
-- columnas propias, CHECK disjunto ampliado sobre la postimagen medida y dos
-- registradores que sólo puede ejecutar el propietario de Autorización.
-- No concede permisos, no crea LOGIN y no toca registros anteriores.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000196',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
DO $pre$
DECLARE actual text;
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 THEN RAISE EXCEPTION 'AD196: PARO clave=migrador_PG actual=no_acreditado esperado=superusuario_PG18' USING ERRCODE='42501'; END IF;
 IF to_regprocedure('vec_autorizacion_atestada_v3.encuadrar_mac(text)') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_gobierno_usuarios_admin_v1(jsonb)') IS NULL
 OR to_regrole('vec_autorizacion_propietario') IS NULL
 OR NOT EXISTS(SELECT 1 FROM pg_attribute WHERE attrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass AND attname='transaccion_origen' AND NOT attisdropped)
 OR EXISTS(SELECT 1 FROM pg_attribute WHERE attrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass AND NOT attisdropped
  AND attname IN('perfiles_asignables_detalle','perfiles_asignables_solicitud_sha256'))
 OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_perfiles_asignables_admin_v1(jsonb)') IS NOT NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_intento_perfiles_asignables_admin_v1(jsonb)') IS NOT NULL
 THEN RAISE EXCEPTION 'AD196: PARO clave=preimagen actual=incompatible esperado=POST193_sin_AD196' USING ERRCODE='55000'; END IF;
 -- Postimagen medida del CHECK disjunto tras AD193 (principal H12).
 SELECT encode(sha256(convert_to(pg_get_constraintdef(c.oid,false),'UTF8')),'hex') INTO actual
 FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
 AND c.conname='auditoria_tipo_disjunto_v4' AND c.contype='c' AND c.convalidated;
 IF actual IS DISTINCT FROM 'ca7e3bdeae8f7218dea440a999519c91a0be7423ee6d214b5420a47da0d6fcc0'
 THEN RAISE EXCEPTION 'AD196: PARO clave=CHECK_sha actual=% esperado=ca7e3bdeae8f7218dea440a999519c91a0be7423ee6d214b5420a47da0d6fcc0',coalesce(actual,'ausente') USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
LOCK TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 IN ACCESS EXCLUSIVE MODE;
ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3
 ADD COLUMN perfiles_asignables_detalle jsonb, ADD COLUMN perfiles_asignables_solicitud_sha256 text;
-- La familia nueva exige NULL en todas las columnas propias de otras familias.
-- La lista se obtiene del catálogo en el mismo instante en que se mide el CHECK;
-- sólo quedan fuera las columnas comunes del eslabón y las de esta familia.
DO $familia$
DECLARE anterior text;nuevo text;nulas text;
 propias constant text[]:=ARRAY['auditoria_ref','secuencia','anterior_sha256','huella_sha256','registrada_en','tipo_registro',
  'evento_ref','evento_material_sha256','operador_login','plan_sha256','accion','modulo_id','recurso_ref','finalidad_ref',
  'resultado','motivo_ref','proceso','canal','correlacion_ref','perfiles_asignables_detalle','perfiles_asignables_solicitud_sha256'];
BEGIN
 SELECT pg_get_constraintdef(c.oid,false) INTO STRICT anterior FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass AND c.conname='auditoria_tipo_disjunto_v4';
 SELECT string_agg(format('%I IS NULL',a.attname),' AND ' ORDER BY a.attnum) INTO STRICT nulas
 FROM pg_attribute a WHERE a.attrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
 AND a.attnum>0 AND NOT a.attisdropped AND NOT a.attname=ANY(propias);
 IF nulas IS NULL OR nulas NOT LIKE '%transaccion_origen IS NULL%' OR nulas NOT LIKE '%gobierno_usuarios_detalle IS NULL%'
 OR nulas NOT LIKE '%decision_ref IS NULL%' OR nulas NOT LIKE '%actor_ref IS NULL%'
 THEN RAISE EXCEPTION 'AD196: PARO clave=columnas actual=incompatible esperado=familias_previas' USING ERRCODE='55000'; END IF;
 nuevo:='CHECK ((perfiles_asignables_detalle IS NULL AND perfiles_asignables_solicitud_sha256 IS NULL AND ('||substr(anterior,8,length(anterior)-8)||')) OR ('||
 'tipo_registro IN (''perfiles_asignables_admin'',''intento_perfiles_asignables_admin'') AND '||nulas||$tipo$
 AND evento_ref IS NOT NULL AND evento_material_sha256 IS NOT NULL AND operador_login IS NOT NULL
 AND accion IS NOT DISTINCT FROM 'registrar_perfiles_asignables_admin_v1' AND modulo_id IS NOT DISTINCT FROM 'administracion'
 AND proceso IS NOT DISTINCT FROM 'postgresql' AND canal IS NOT DISTINCT FROM 'operacion_tecnica_privada'
 AND finalidad_ref IS NOT DISTINCT FROM 'perfiles_asignables_admin' AND recurso_ref IS NOT NULL AND correlacion_ref IS NOT NULL
 AND resultado IS NOT NULL AND motivo_ref IS NOT NULL
 AND ((tipo_registro='perfiles_asignables_admin' AND plan_sha256 IS NOT NULL AND perfiles_asignables_detalle IS NOT NULL AND perfiles_asignables_solicitud_sha256 IS NULL
 AND resultado IS NOT DISTINCT FROM 'permitido' AND motivo_ref IS NOT DISTINCT FROM 'perfiles_asignables_registrado' AND recurso_ref IS NOT DISTINCT FROM 'perfiles_asignables:'||substr(plan_sha256,1,32))
 OR (tipo_registro='intento_perfiles_asignables_admin' AND plan_sha256 IS NULL AND perfiles_asignables_detalle IS NULL AND perfiles_asignables_solicitud_sha256 IS NOT NULL))
 $tipo$||'))';
 ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 DROP CONSTRAINT auditoria_tipo_disjunto_v4;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 ADD CONSTRAINT auditoria_tipo_disjunto_v4 '||nuevo;
END $familia$;

-- Detalle cerrado: referencia de la operación, huella de la lista registrada,
-- número de perfiles (1..32) y huella de la aprobación externa. La lista
-- completa y el plan exacto quedan en el registro de AUT49.
CREATE FUNCTION vec_autorizacion_atestada_v3.detalle_perfiles_asignables_valido_v1(p jsonb)
RETURNS boolean LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog AS $f$
BEGIN
 RETURN jsonb_typeof(p)='object' AND (SELECT count(*) FROM jsonb_object_keys(p))=4
  AND p ?& ARRAY['operacion_ref','perfiles_sha256','perfiles_numero','aprobacion_sha256']
  AND jsonb_typeof(p->'aprobacion_sha256')='string' AND p->>'aprobacion_sha256' ~ '^[0-9a-f]{64}$'
  AND jsonb_typeof(p->'operacion_ref')='string' AND p->>'operacion_ref' ~ '^rpa_[A-Za-z0-9_-]{22,124}$'
  AND jsonb_typeof(p->'perfiles_sha256')='string' AND p->>'perfiles_sha256' ~ '^[0-9a-f]{64}$'
  AND jsonb_typeof(p->'perfiles_numero')='string' AND p->>'perfiles_numero' ~ '^[1-9][0-9]?$'
  AND (p->>'perfiles_numero')::integer<=32;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.detalle_perfiles_asignables_valido_v1(jsonb) FROM PUBLIC;
CREATE FUNCTION vec_autorizacion_atestada_v3.motivo_intento_perfiles_asignables_valido_v1(p_resultado text,p_motivo text)
RETURNS boolean LANGUAGE sql IMMUTABLE STRICT PARALLEL SAFE SET search_path=pg_catalog AS $f$
 SELECT EXISTS(SELECT 1 FROM (VALUES ('permitido','perfiles_asignables_registrado'),('permitido','perfiles_asignables_replay'),
 ('denegado','perfiles_asignables_denegado'),('error','perfiles_asignables_error')) AS m(resultado,motivo_ref)
 WHERE m.resultado=p_resultado AND m.motivo_ref=p_motivo)
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.motivo_intento_perfiles_asignables_valido_v1(text,text) FROM PUBLIC;
ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 ADD CONSTRAINT auditoria_perfiles_asignables_formato_v1 CHECK (
 tipo_registro NOT IN('perfiles_asignables_admin','intento_perfiles_asignables_admin') OR (
 evento_ref ~ '^evento_[0-9a-f]{32}$' AND evento_material_sha256 ~ '^[0-9a-f]{64}$' AND correlacion_ref ~ '^correlacion_[0-9a-f]{32}$'
 AND ((tipo_registro='perfiles_asignables_admin' AND plan_sha256 ~ '^[0-9a-f]{64}$' AND vec_autorizacion_atestada_v3.detalle_perfiles_asignables_valido_v1(perfiles_asignables_detalle))
 OR (tipo_registro='intento_perfiles_asignables_admin' AND perfiles_asignables_solicitud_sha256 ~ '^[0-9a-f]{64}$'
  AND recurso_ref ~ '^solicitud_perfiles_asignables:[0-9a-f]{32}$' AND vec_autorizacion_atestada_v3.motivo_intento_perfiles_asignables_valido_v1(resultado,motivo_ref)))));

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_perfiles_asignables_admin_v1(p_evento jsonb)
RETURNS TABLE(auditoria_ref text,secuencia numeric,huella_sha256 text,correlacion_ref text,registrada_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' SET statement_timeout='10s'
AS $funcion$
DECLARE
 v_orden constant text[]:=ARRAY['tipo_registro','evento_ref','operador_login','plan_sha256','operacion_ref','perfiles_sha256','perfiles_numero','aprobacion_sha256','proceso','canal','finalidad_ref','correlacion_ref'];
 v_claves text[];v_clave text;v_material bytea;v_material_sha text;v_anterior text;
 v_secuencia numeric;v_instante timestamptz(6);v_ref text;v_huella text;v_existente record;
BEGIN
 IF pg_catalog.current_setting('transaction_isolation')<>'serializable'
 OR pg_catalog.current_setting('transaction_read_only')<>'off' OR pg_catalog.current_setting('TimeZone')<>'UTC'
 THEN RAISE EXCEPTION 'AD196: PARO clave=transaccion actual=incompatible esperado=serializable_rw_UTC' USING ERRCODE='25000'; END IF;
 IF pg_catalog.jsonb_typeof(p_evento) IS DISTINCT FROM 'object' OR pg_catalog.octet_length(p_evento::text)>8192
 THEN RAISE EXCEPTION 'AD196: PARO clave=evento actual=incompatible esperado=objeto_acotado' USING ERRCODE='22023'; END IF;
 SELECT pg_catalog.array_agg(k ORDER BY k COLLATE "C") INTO v_claves FROM pg_catalog.jsonb_object_keys(p_evento) k;
 IF v_claves IS DISTINCT FROM (SELECT pg_catalog.array_agg(k ORDER BY k COLLATE "C") FROM pg_catalog.unnest(v_orden) k)
 THEN RAISE EXCEPTION 'AD196: PARO clave=campos actual=incompatible esperado=ABI_exacto' USING ERRCODE='22023'; END IF;
 FOREACH v_clave IN ARRAY v_orden LOOP
  IF pg_catalog.jsonb_typeof(p_evento->v_clave) IS DISTINCT FROM 'string'
  OR pg_catalog.octet_length(p_evento->>v_clave) NOT BETWEEN 1 AND 200
  THEN RAISE EXCEPTION 'AD196: PARO clave=valor actual=incompatible esperado=string_acotado' USING ERRCODE='22023'; END IF;
 END LOOP;
 IF p_evento->>'tipo_registro' IS DISTINCT FROM 'perfiles_asignables_admin'
 OR p_evento->>'operador_login' IS DISTINCT FROM session_user::text
 OR pg_catalog.octet_length(p_evento->>'operador_login')>63
 OR p_evento->>'evento_ref' !~ '^evento_[0-9a-f]{32}$'
 OR p_evento->>'plan_sha256' !~ '^[0-9a-f]{64}$'
 OR vec_autorizacion_atestada_v3.detalle_perfiles_asignables_valido_v1(p_evento-ARRAY['tipo_registro','evento_ref','operador_login','plan_sha256','proceso','canal','finalidad_ref','correlacion_ref']) IS DISTINCT FROM true
 OR p_evento->>'proceso' IS DISTINCT FROM 'postgresql'
 OR p_evento->>'canal' IS DISTINCT FROM 'operacion_tecnica_privada'
 OR p_evento->>'finalidad_ref' IS DISTINCT FROM 'perfiles_asignables_admin'
 OR p_evento->>'correlacion_ref' !~ '^correlacion_[0-9a-f]{32}$'
 THEN RAISE EXCEPTION 'AD196: PARO clave=semantica actual=incompatible esperado=perfiles_asignables_privado' USING ERRCODE='22023'; END IF;
 v_material:=vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.perfiles-asignables-admin.v1');
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
  IF v_existente.tipo_registro IS DISTINCT FROM 'perfiles_asignables_admin'
  OR v_existente.evento_material_sha256 IS DISTINCT FROM v_material_sha
  THEN RAISE EXCEPTION 'AD196: PARO clave=replay actual=material_distinto esperado=material_original' USING ERRCODE='23505'; END IF;
  RETURN QUERY SELECT v_existente.auditoria_ref,v_existente.secuencia,v_existente.huella_sha256,
   v_existente.correlacion_ref,v_existente.registrada_en;
  RETURN;
 END IF;
 SELECT c.secuencia,c.cabeza_sha256 INTO STRICT v_secuencia,v_anterior
 FROM vec_autorizacion_atestada_v3.control_cadena_auditoria c WHERE c.control_id FOR UPDATE;
 IF v_secuencia>=9007199254740991::numeric
 THEN RAISE EXCEPTION 'AD196: PARO clave=secuencia actual=limite esperado=entero_seguro' USING ERRCODE='22003'; END IF;
 v_secuencia:=v_secuencia+1;v_instante:=pg_catalog.clock_timestamp();
 v_ref:='aud_v3_pa_'||pg_catalog.substr(p_evento->>'evento_ref',8,32);
 v_huella:=pg_catalog.encode(pg_catalog.sha256(
  vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.eslabon.perfiles-asignables-admin.v1')||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_secuencia::text)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_anterior)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_ref)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_material_sha)||
  vec_autorizacion_atestada_v3.encuadrar_mac(pg_catalog.to_char(v_instante AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))),'hex');
 INSERT INTO vec_autorizacion_atestada_v3.auditoria_consumo_v3(
  auditoria_ref,secuencia,anterior_sha256,huella_sha256,registrada_en,tipo_registro,
  evento_ref,evento_material_sha256,operador_login,plan_sha256,perfiles_asignables_detalle,
  accion,modulo_id,recurso_ref,finalidad_ref,resultado,motivo_ref,proceso,canal,correlacion_ref)
 VALUES(v_ref,v_secuencia,v_anterior,v_huella,v_instante,'perfiles_asignables_admin',
  p_evento->>'evento_ref',v_material_sha,(p_evento->>'operador_login')::name,p_evento->>'plan_sha256',
  p_evento-ARRAY['tipo_registro','evento_ref','operador_login','plan_sha256','proceso','canal','finalidad_ref','correlacion_ref'],
  'registrar_perfiles_asignables_admin_v1','administracion','perfiles_asignables:'||pg_catalog.substr(p_evento->>'plan_sha256',1,32),
  'perfiles_asignables_admin','permitido','perfiles_asignables_registrado',p_evento->>'proceso',p_evento->>'canal',p_evento->>'correlacion_ref');
 UPDATE vec_autorizacion_atestada_v3.control_cadena_auditoria
 SET secuencia=v_secuencia,cabeza_sha256=v_huella,actualizada_en=v_instante WHERE control_id;
 RETURN QUERY SELECT v_ref,v_secuencia,v_huella,p_evento->>'correlacion_ref',v_instante;
END $funcion$;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_intento_perfiles_asignables_admin_v1(p_evento jsonb)
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
 THEN RAISE EXCEPTION 'AD196: PARO clave=transaccion actual=incompatible esperado=serializable_rw_UTC' USING ERRCODE='25000'; END IF;
 IF pg_catalog.jsonb_typeof(p_evento) IS DISTINCT FROM 'object' OR pg_catalog.octet_length(p_evento::text)>8192
 THEN RAISE EXCEPTION 'AD196: PARO clave=evento actual=incompatible esperado=objeto_acotado' USING ERRCODE='22023'; END IF;
 SELECT pg_catalog.array_agg(k ORDER BY k COLLATE "C") INTO v_claves FROM pg_catalog.jsonb_object_keys(p_evento) k;
 IF v_claves IS DISTINCT FROM (SELECT pg_catalog.array_agg(k ORDER BY k COLLATE "C") FROM pg_catalog.unnest(v_orden) k)
 THEN RAISE EXCEPTION 'AD196: PARO clave=campos actual=incompatible esperado=ABI_exacto' USING ERRCODE='22023'; END IF;
 FOREACH v_clave IN ARRAY v_orden LOOP
  IF pg_catalog.jsonb_typeof(p_evento->v_clave) IS DISTINCT FROM 'string'
  OR pg_catalog.octet_length(p_evento->>v_clave) NOT BETWEEN 1 AND 200
  THEN RAISE EXCEPTION 'AD196: PARO clave=valor actual=incompatible esperado=string_acotado' USING ERRCODE='22023'; END IF;
 END LOOP;
 IF p_evento->>'tipo_registro' IS DISTINCT FROM 'intento_perfiles_asignables_admin'
 OR p_evento->>'operador_login' IS DISTINCT FROM session_user::text
 OR pg_catalog.octet_length(p_evento->>'operador_login')>63
 OR p_evento->>'evento_ref' !~ '^evento_[0-9a-f]{32}$'
 OR p_evento->>'solicitud_sha256' !~ '^[0-9a-f]{64}$'
 OR p_evento->>'accion' IS DISTINCT FROM 'registrar_perfiles_asignables_admin_v1'
 OR p_evento->>'recurso_ref' !~ '^solicitud_perfiles_asignables:[0-9a-f]{32}$'
 OR vec_autorizacion_atestada_v3.motivo_intento_perfiles_asignables_valido_v1(p_evento->>'resultado',p_evento->>'motivo_ref') IS DISTINCT FROM true
 OR p_evento->>'proceso' IS DISTINCT FROM 'postgresql'
 OR p_evento->>'canal' IS DISTINCT FROM 'operacion_tecnica_privada'
 OR p_evento->>'finalidad_ref' IS DISTINCT FROM 'perfiles_asignables_admin'
 OR p_evento->>'correlacion_ref' !~ '^correlacion_[0-9a-f]{32}$'
 THEN RAISE EXCEPTION 'AD196: PARO clave=semantica actual=incompatible esperado=perfiles_asignables_privado' USING ERRCODE='22023'; END IF;
 v_material:=vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.intento-perfiles-asignables-admin.v1');
 FOREACH v_clave IN ARRAY v_orden LOOP
  v_material:=v_material||vec_autorizacion_atestada_v3.encuadrar_mac(p_evento->>v_clave);
 END LOOP;
 v_material_sha:=pg_catalog.encode(pg_catalog.sha256(v_material),'hex');
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
  'vec_autorizacion_atestada_v3:evento-admin:'||(p_evento->>'evento_ref'),0));
 SELECT a.tipo_registro,a.auditoria_ref,a.secuencia,a.huella_sha256,a.correlacion_ref,a.registrada_en,
  a.evento_material_sha256 INTO v_existente
 FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 a WHERE a.evento_ref=p_evento->>'evento_ref';
 IF FOUND THEN
  IF v_existente.tipo_registro IS DISTINCT FROM 'intento_perfiles_asignables_admin'
  OR v_existente.evento_material_sha256 IS DISTINCT FROM v_material_sha
  THEN RAISE EXCEPTION 'AD196: PARO clave=replay actual=material_distinto esperado=material_original' USING ERRCODE='23505'; END IF;
  RETURN QUERY SELECT v_existente.auditoria_ref,v_existente.secuencia,v_existente.huella_sha256,
   v_existente.correlacion_ref,v_existente.registrada_en;
  RETURN;
 END IF;
 SELECT c.secuencia,c.cabeza_sha256 INTO STRICT v_secuencia,v_anterior
 FROM vec_autorizacion_atestada_v3.control_cadena_auditoria c WHERE c.control_id FOR UPDATE;
 IF v_secuencia>=9007199254740991::numeric
 THEN RAISE EXCEPTION 'AD196: PARO clave=secuencia actual=limite esperado=entero_seguro' USING ERRCODE='22003'; END IF;
 v_secuencia:=v_secuencia+1;v_instante:=pg_catalog.clock_timestamp();
 v_ref:='aud_v3_pai_'||pg_catalog.substr(p_evento->>'evento_ref',8,32);
 v_huella:=pg_catalog.encode(pg_catalog.sha256(
  vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.eslabon.intento-perfiles-asignables-admin.v1')||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_secuencia::text)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_anterior)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_ref)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_material_sha)||
  vec_autorizacion_atestada_v3.encuadrar_mac(pg_catalog.to_char(v_instante AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))),'hex');
 INSERT INTO vec_autorizacion_atestada_v3.auditoria_consumo_v3(
  auditoria_ref,secuencia,anterior_sha256,huella_sha256,registrada_en,tipo_registro,
  evento_ref,evento_material_sha256,operador_login,perfiles_asignables_solicitud_sha256,
  accion,modulo_id,recurso_ref,finalidad_ref,resultado,motivo_ref,proceso,canal,correlacion_ref)
 VALUES(v_ref,v_secuencia,v_anterior,v_huella,v_instante,'intento_perfiles_asignables_admin',
  p_evento->>'evento_ref',v_material_sha,(p_evento->>'operador_login')::name,p_evento->>'solicitud_sha256',
  p_evento->>'accion','administracion',p_evento->>'recurso_ref',p_evento->>'finalidad_ref',
  p_evento->>'resultado',p_evento->>'motivo_ref',p_evento->>'proceso',p_evento->>'canal',p_evento->>'correlacion_ref');
 UPDATE vec_autorizacion_atestada_v3.control_cadena_auditoria
 SET secuencia=v_secuencia,cabeza_sha256=v_huella,actualizada_en=v_instante WHERE control_id;
 RETURN QUERY SELECT v_ref,v_secuencia,v_huella,p_evento->>'correlacion_ref',v_instante;
END $funcion$;

REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_perfiles_asignables_admin_v1(jsonb) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_intento_perfiles_asignables_admin_v1(jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_perfiles_asignables_admin_v1(jsonb),vec_autorizacion_atestada_v3.registrar_intento_perfiles_asignables_admin_v1(jsonb) TO vec_autorizacion_propietario;
DO $acl$
DECLARE f record;roles oid[]:=ARRAY['vec_autorizacion_atestada_v3_propietario'::regrole,'vec_autorizacion_propietario'::regrole];
BEGIN
 FOR f IN SELECT oid FROM pg_proc WHERE oid IN(to_regprocedure('vec_autorizacion_atestada_v3.registrar_perfiles_asignables_admin_v1(jsonb)'),to_regprocedure('vec_autorizacion_atestada_v3.registrar_intento_perfiles_asignables_admin_v1(jsonb)')) LOOP
  IF EXISTS(SELECT 1 FROM pg_proc p,LATERAL aclexplode(COALESCE(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f.oid
   AND (a.grantee<>ALL(roles) OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
  THEN RAISE EXCEPTION 'AD196: PARO clave=ACL actual=ampliada esperado=AD_AUT_owners' USING ERRCODE='55000'; END IF;
 END LOOP;
 IF EXISTS(SELECT 1 FROM pg_proc p,LATERAL aclexplode(COALESCE(p.proacl,acldefault('f',p.proowner))) a
  WHERE p.oid IN(to_regprocedure('vec_autorizacion_atestada_v3.detalle_perfiles_asignables_valido_v1(jsonb)'),to_regprocedure('vec_autorizacion_atestada_v3.motivo_intento_perfiles_asignables_valido_v1(text,text)'))
  AND a.grantee<>p.proowner)
 THEN RAISE EXCEPTION 'AD196: PARO clave=ACL_validadores actual=ampliada esperado=solo_propietario' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
